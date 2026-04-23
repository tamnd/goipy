package vm

import (
	"math"
	"math/big"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildOperator() *object.Module {
	m := &object.Module{Name: "operator", Dict: object.NewDict()}

	// Helpers: wrap a binary or unary function as a BuiltinFunc.
	bin := func(name string, fn func(a, b object.Object) (object.Object, error)) {
		m.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 2 {
				return nil, object.Errorf(i.typeErr, "%s() takes 2 arguments", name)
			}
			return fn(a[0], a[1])
		}})
	}
	un := func(name string, fn func(a object.Object) (object.Object, error)) {
		m.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "%s() takes 1 argument", name)
			}
			return fn(a[0])
		}})
	}

	// ===== Arithmetic binary operators =====
	bin("add", i.add)
	bin("sub", i.sub)
	bin("mul", i.mul)
	bin("truediv", i.truediv)
	bin("floordiv", i.floordiv)
	bin("mod", i.mod)
	bin("pow", i.pow)
	bin("matmul", i.matmul)

	// concat(a, b): sequence concatenation; a must support sequence addition.
	bin("concat", func(a, b object.Object) (object.Object, error) {
		switch a.(type) {
		case *object.Str, *object.List, *object.Tuple, *object.Bytes, *object.Bytearray:
			return i.add(a, b)
		}
		if inst, ok := a.(*object.Instance); ok {
			if r, ok2, err := i.callInstanceDunder(inst, "__add__", b); ok2 {
				return r, err
			}
		}
		return nil, object.Errorf(i.typeErr, "'%s' object can't be concatenated", object.TypeName(a))
	})

	// ===== Bitwise binary operators =====
	bin("and_", func(a, b object.Object) (object.Object, error) { return i.bitop(a, b, "&") })
	bin("or_", func(a, b object.Object) (object.Object, error) { return i.bitop(a, b, "|") })
	bin("xor", func(a, b object.Object) (object.Object, error) { return i.bitop(a, b, "^") })
	bin("lshift", func(a, b object.Object) (object.Object, error) { return i.shift(a, b, true) })
	bin("rshift", func(a, b object.Object) (object.Object, error) { return i.shift(a, b, false) })

	// ===== Unary operators =====
	un("neg", i.unaryNeg)
	un("pos", func(a object.Object) (object.Object, error) {
		if inst, ok := a.(*object.Instance); ok {
			if r, ok2, err := i.callInstanceDunder(inst, "__pos__"); ok2 {
				return r, err
			}
		}
		return a, nil
	})
	un("not_", func(a object.Object) (object.Object, error) { return object.BoolOf(!object.Truthy(a)), nil })
	un("truth", func(a object.Object) (object.Object, error) { return object.BoolOf(object.Truthy(a)), nil })
	un("index", func(a object.Object) (object.Object, error) {
		if n, ok := toInt64(a); ok {
			return object.NewInt(n), nil
		}
		if inst, ok := a.(*object.Instance); ok {
			if r, ok2, err := i.callInstanceDunder(inst, "__index__"); ok2 {
				return r, err
			}
		}
		return nil, object.Errorf(i.typeErr, "'%s' object cannot be interpreted as an integer", object.TypeName(a))
	})

	// abs(a)
	un("abs", func(a object.Object) (object.Object, error) {
		if inst, ok := a.(*object.Instance); ok {
			if r, ok2, err := i.callInstanceDunder(inst, "__abs__"); ok2 {
				return r, err
			}
		}
		switch v := a.(type) {
		case *object.Bool:
			if v.V {
				return object.NewInt(1), nil
			}
			return object.NewInt(0), nil
		case *object.Int:
			return object.IntFromBig(new(big.Int).Abs(&v.V)), nil
		case *object.Float:
			if v.V < 0 {
				return &object.Float{V: -v.V}, nil
			}
			return a, nil
		case *object.Complex:
			return &object.Float{V: math.Hypot(v.Real, v.Imag)}, nil
		}
		return nil, object.Errorf(i.typeErr, "bad operand type for abs(): '%s'", object.TypeName(a))
	})

	// inv / invert: bitwise NOT (~a)
	invertFn := func(a object.Object) (object.Object, error) {
		if inst, ok := a.(*object.Instance); ok {
			if r, ok2, err := i.callInstanceDunder(inst, "__invert__"); ok2 {
				return r, err
			}
		}
		bi, ok := toBigInt(a)
		if !ok {
			return nil, object.Errorf(i.typeErr, "bad operand type for unary ~: '%s'", object.TypeName(a))
		}
		return object.IntFromBig(new(big.Int).Not(bi)), nil
	}
	un("inv", invertFn)
	un("invert", invertFn)

	// ===== Comparison operators =====
	cmpKinds := map[string]int{"lt": 0, "le": 1, "eq": 2, "ne": 3, "gt": 4, "ge": 5}
	for name, kind := range cmpKinds {
		kind, name := kind, name
		m.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 2 {
				return nil, object.Errorf(i.typeErr, "%s() takes 2 arguments", name)
			}
			return i.compare(a[0], a[1], kind)
		}})
	}

	// is_(a, b): identity check (a is b)
	bin("is_", func(a, b object.Object) (object.Object, error) {
		if _, ok := a.(*object.NoneType); ok {
			_, ok2 := b.(*object.NoneType)
			return object.BoolOf(ok2), nil
		}
		return object.BoolOf(a == b), nil
	})
	// is_not(a, b): identity check (a is not b)
	bin("is_not", func(a, b object.Object) (object.Object, error) {
		if _, ok := a.(*object.NoneType); ok {
			_, ok2 := b.(*object.NoneType)
			return object.BoolOf(!ok2), nil
		}
		return object.BoolOf(a != b), nil
	})

	// ===== Subscript / sequence protocol =====
	m.Dict.SetStr("getitem", &object.BuiltinFunc{Name: "getitem", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "getitem() takes 2 arguments")
		}
		return i.getitem(a[0], a[1])
	}})
	m.Dict.SetStr("setitem", &object.BuiltinFunc{Name: "setitem", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 3 {
			return nil, object.Errorf(i.typeErr, "setitem() takes 3 arguments")
		}
		return object.None, i.setitem(a[0], a[1], a[2])
	}})
	m.Dict.SetStr("delitem", &object.BuiltinFunc{Name: "delitem", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "delitem() takes 2 arguments")
		}
		return object.None, i.delitem(a[0], a[1])
	}})
	m.Dict.SetStr("contains", &object.BuiltinFunc{Name: "contains", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "contains() takes 2 arguments")
		}
		r, err := i.contains(a[0], a[1])
		if err != nil {
			return nil, err
		}
		return object.BoolOf(r), nil
	}})

	// length_hint(obj, default=0): return len(obj) or obj.__length_hint__(), else default
	m.Dict.SetStr("length_hint", &object.BuiltinFunc{Name: "length_hint", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "length_hint() takes at least 1 argument")
		}
		obj := a[0]
		defVal := object.Object(object.NewInt(0))
		if len(a) >= 2 {
			defVal = a[1]
		}
		if kw != nil {
			if v, ok := kw.GetStr("default"); ok {
				defVal = v
			}
		}
		n, err := i.length(obj)
		if err == nil {
			return object.NewInt(n), nil
		}
		if inst, ok := obj.(*object.Instance); ok {
			if r, ok2, err2 := i.callInstanceDunder(inst, "__length_hint__"); ok2 {
				return r, err2
			}
		}
		return defVal, nil
	}})

	// countOf(a, b): count occurrences of b in iterable a
	bin("countOf", func(a, b object.Object) (object.Object, error) {
		items, err := iterate(i, a)
		if err != nil {
			return nil, err
		}
		count := int64(0)
		for _, x := range items {
			eq, eqErr := object.Eq(x, b)
			if eqErr != nil {
				return nil, eqErr
			}
			if eq {
				count++
			}
		}
		return object.NewInt(count), nil
	})

	// indexOf(a, b): index of first occurrence of b in iterable a; ValueError if not found
	bin("indexOf", func(a, b object.Object) (object.Object, error) {
		items, err := iterate(i, a)
		if err != nil {
			return nil, err
		}
		for idx, x := range items {
			eq, eqErr := object.Eq(x, b)
			if eqErr != nil {
				return nil, eqErr
			}
			if eq {
				return object.NewInt(int64(idx)), nil
			}
		}
		return nil, object.Errorf(i.valueErr, "indexOf(): value not found")
	})

	// ===== In-place (augmented assignment) operators =====
	//
	// Each iXxx(a, b) tries a.__iXxx__(b) first (mutating, returns self for
	// mutable containers), then falls back to the regular operator.

	listInplaceAdd := func(a, b object.Object) (object.Object, error) {
		if l, ok := a.(*object.List); ok {
			items, err := iterate(i, b)
			if err != nil {
				return nil, err
			}
			l.V = append(l.V, items...)
			return l, nil
		}
		return i.add(a, b)
	}

	makeInplace := func(name, dunder string, fallback func(a, b object.Object) (object.Object, error)) {
		bin(name, func(a, b object.Object) (object.Object, error) {
			if inst, ok := a.(*object.Instance); ok {
				if r, ok2, err := i.callInstanceDunder(inst, dunder, b); ok2 {
					return r, err
				}
			}
			return fallback(a, b)
		})
	}

	makeInplace("iadd", "__iadd__", listInplaceAdd)
	makeInplace("iand", "__iand__", func(a, b object.Object) (object.Object, error) { return i.bitop(a, b, "&") })
	makeInplace("iconcat", "__iconcat__", listInplaceAdd)
	makeInplace("ifloordiv", "__ifloordiv__", i.floordiv)
	makeInplace("ilshift", "__ilshift__", func(a, b object.Object) (object.Object, error) { return i.shift(a, b, true) })
	makeInplace("imod", "__imod__", i.mod)
	makeInplace("imul", "__imul__", i.mul)
	makeInplace("imatmul", "__imatmul__", i.matmul)
	makeInplace("ior", "__ior__", func(a, b object.Object) (object.Object, error) { return i.bitop(a, b, "|") })
	makeInplace("ipow", "__ipow__", i.pow)
	makeInplace("irshift", "__irshift__", func(a, b object.Object) (object.Object, error) { return i.shift(a, b, false) })
	makeInplace("isub", "__isub__", i.sub)
	makeInplace("itruediv", "__itruediv__", i.truediv)
	makeInplace("ixor", "__ixor__", func(a, b object.Object) (object.Object, error) { return i.bitop(a, b, "^") })

	// ===== Higher-order callables =====

	// attrgetter(name, ...) → callable that reads named attribute(s).
	m.Dict.SetStr("attrgetter", &object.BuiltinFunc{Name: "attrgetter", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "attrgetter expected at least 1 argument")
		}
		names := make([]string, len(a))
		for idx, x := range a {
			s, ok := x.(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "attrgetter argument must be str")
			}
			names[idx] = s.V
		}
		getOne := func(obj object.Object, name string) (object.Object, error) {
			for _, part := range splitDot(name) {
				v, err := i.getAttr(obj, part)
				if err != nil {
					return nil, err
				}
				obj = v
			}
			return obj, nil
		}
		return &object.BuiltinFunc{Name: "attrgetter", Call: func(_ any, b []object.Object, _ *object.Dict) (object.Object, error) {
			if len(b) != 1 {
				return nil, object.Errorf(i.typeErr, "attrgetter() takes 1 argument")
			}
			if len(names) == 1 {
				return getOne(b[0], names[0])
			}
			out := make([]object.Object, len(names))
			for idx, n := range names {
				v, err := getOne(b[0], n)
				if err != nil {
					return nil, err
				}
				out[idx] = v
			}
			return &object.Tuple{V: out}, nil
		}}, nil
	}})

	// itemgetter(key, ...) → callable that reads item(s) by key.
	m.Dict.SetStr("itemgetter", &object.BuiltinFunc{Name: "itemgetter", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "itemgetter expected at least 1 argument")
		}
		keys := append([]object.Object{}, a...)
		return &object.BuiltinFunc{Name: "itemgetter", Call: func(_ any, b []object.Object, _ *object.Dict) (object.Object, error) {
			if len(b) != 1 {
				return nil, object.Errorf(i.typeErr, "itemgetter() takes 1 argument")
			}
			if len(keys) == 1 {
				return i.getitem(b[0], keys[0])
			}
			out := make([]object.Object, len(keys))
			for idx, k := range keys {
				v, err := i.getitem(b[0], k)
				if err != nil {
					return nil, err
				}
				out[idx] = v
			}
			return &object.Tuple{V: out}, nil
		}}, nil
	}})

	// methodcaller(name, *args, **kw) → callable that invokes named method.
	m.Dict.SetStr("methodcaller", &object.BuiltinFunc{Name: "methodcaller", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "methodcaller expected method name")
		}
		name, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "methodcaller name must be str")
		}
		args := append([]object.Object{}, a[1:]...)
		boundKw := kw
		return &object.BuiltinFunc{Name: "methodcaller", Call: func(_ any, b []object.Object, _ *object.Dict) (object.Object, error) {
			if len(b) != 1 {
				return nil, object.Errorf(i.typeErr, "methodcaller() takes 1 argument")
			}
			method, err := i.getAttr(b[0], name.V)
			if err != nil {
				return nil, err
			}
			return i.callObject(method, args, boundKw)
		}}, nil
	}})

	// ===== Dunder-name aliases =====
	// operator.lt == operator.__lt__, operator.add == operator.__add__, etc.
	aliases := [][2]string{
		{"__lt__", "lt"}, {"__le__", "le"}, {"__eq__", "eq"}, {"__ne__", "ne"},
		{"__ge__", "ge"}, {"__gt__", "gt"},
		{"__not__", "not_"}, {"__abs__", "abs"},
		{"__add__", "add"}, {"__and__", "and_"}, {"__floordiv__", "floordiv"},
		{"__index__", "index"}, {"__inv__", "inv"}, {"__invert__", "invert"},
		{"__lshift__", "lshift"}, {"__mod__", "mod"}, {"__mul__", "mul"},
		{"__matmul__", "matmul"}, {"__neg__", "neg"}, {"__or__", "or_"},
		{"__pos__", "pos"}, {"__pow__", "pow"}, {"__rshift__", "rshift"},
		{"__sub__", "sub"}, {"__truediv__", "truediv"}, {"__xor__", "xor"},
		{"__concat__", "concat"}, {"__contains__", "contains"},
		{"__delitem__", "delitem"}, {"__getitem__", "getitem"}, {"__setitem__", "setitem"},
		{"__iadd__", "iadd"}, {"__iand__", "iand"}, {"__iconcat__", "iconcat"},
		{"__ifloordiv__", "ifloordiv"}, {"__ilshift__", "ilshift"}, {"__imod__", "imod"},
		{"__imul__", "imul"}, {"__imatmul__", "imatmul"}, {"__ior__", "ior"},
		{"__ipow__", "ipow"}, {"__irshift__", "irshift"}, {"__isub__", "isub"},
		{"__itruediv__", "itruediv"}, {"__ixor__", "ixor"},
	}
	for _, pair := range aliases {
		if v, ok := m.Dict.GetStr(pair[1]); ok {
			m.Dict.SetStr(pair[0], v)
		}
	}

	return m
}

// splitDot splits a dotted attribute name (e.g. "a.b.c") into parts.
func splitDot(s string) []string {
	var parts []string
	start := 0
	for k := 0; k <= len(s); k++ {
		if k == len(s) || s[k] == '.' {
			parts = append(parts, s[start:k])
			start = k + 1
		}
	}
	return parts
}

package vm

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/tamnd/goipy/object"
)

// initBuiltins creates the interpreter's builtin namespace and exception
// classes.
func (i *Interp) initBuiltins() {
	b := object.NewDict()
	i.Builtins = b

	// Exception classes.
	mk := func(name string, bases ...*object.Class) *object.Class {
		c := &object.Class{Name: name, Dict: object.NewDict()}
		c.Bases = bases
		b.SetStr(name, c)
		return c
	}
	i.baseExc = mk("BaseException")
	i.exception = mk("Exception", i.baseExc)
	i.typeErr = mk("TypeError", i.exception)
	i.valueErr = mk("ValueError", i.exception)
	i.nameErr = mk("NameError", i.exception)
	i.lookupErr = mk("LookupError", i.exception)
	i.keyErr = mk("KeyError", i.lookupErr)
	i.indexErr = mk("IndexError", i.lookupErr)
	i.attrErr = mk("AttributeError", i.exception)
	i.arithErr = mk("ArithmeticError", i.exception)
	i.zeroDivErr = mk("ZeroDivisionError", i.arithErr)
	i.overflowErr = mk("OverflowError", i.arithErr)
	i.runtimeErr = mk("RuntimeError", i.exception)
	i.stopIter = mk("StopIteration", i.exception)
	i.notImpl = mk("NotImplementedError", i.runtimeErr)
	i.assertErr = mk("AssertionError", i.exception)
	i.importErr = mk("ImportError", i.exception)
	i.recursionErr = mk("RecursionError", i.runtimeErr)

	// Singletons & types.
	b.SetStr("None", object.None)
	b.SetStr("True", object.True)
	b.SetStr("False", object.False)

	// Constructors / types.
	b.SetStr("int", &object.BuiltinFunc{Name: "int", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if len(a) == 0 {
			return object.NewInt(0), nil
		}
		switch v := a[0].(type) {
		case *object.Int:
			return v, nil
		case *object.Bool:
			if v.V {
				return object.NewInt(1), nil
			}
			return object.NewInt(0), nil
		case *object.Float:
			n := new(big.Int)
			new(big.Float).SetFloat64(v.V).Int(n)
			return &object.Int{V: n}, nil
		case *object.Str:
			base := 10
			if len(a) > 1 {
				if n, ok := toInt64(a[1]); ok {
					base = int(n)
				}
			}
			n, ok := new(big.Int).SetString(strings.TrimSpace(v.V), base)
			if !ok {
				return nil, object.Errorf(in.valueErr, "invalid literal for int()")
			}
			return &object.Int{V: n}, nil
		}
		return nil, object.Errorf(in.typeErr, "int() argument must be str or int")
	}})
	b.SetStr("float", &object.BuiltinFunc{Name: "float", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if len(a) == 0 {
			return &object.Float{V: 0}, nil
		}
		switch v := a[0].(type) {
		case *object.Float:
			return v, nil
		case *object.Int:
			f, _ := new(big.Float).SetInt(v.V).Float64()
			return &object.Float{V: f}, nil
		case *object.Bool:
			if v.V {
				return &object.Float{V: 1}, nil
			}
			return &object.Float{V: 0}, nil
		case *object.Str:
			f, err := strconv.ParseFloat(strings.TrimSpace(v.V), 64)
			if err != nil {
				return nil, object.Errorf(in.valueErr, "could not convert to float")
			}
			return &object.Float{V: f}, nil
		}
		return nil, object.Errorf(in.typeErr, "float() bad arg")
	}})
	b.SetStr("str", &object.BuiltinFunc{Name: "str", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return &object.Str{V: ""}, nil
		}
		return &object.Str{V: object.Str_(a[0])}, nil
	}})
	b.SetStr("bool", &object.BuiltinFunc{Name: "bool", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return object.False, nil
		}
		return object.BoolOf(object.Truthy(a[0])), nil
	}})
	b.SetStr("bytes", &object.BuiltinFunc{Name: "bytes", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return &object.Bytes{V: nil}, nil
		}
		if n, ok := toInt64(a[0]); ok {
			return &object.Bytes{V: make([]byte, n)}, nil
		}
		items, err := iterate(ii.(*Interp), a[0])
		if err != nil {
			return nil, err
		}
		out := make([]byte, len(items))
		for k, x := range items {
			n, ok := toInt64(x)
			if !ok || n < 0 || n > 255 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "bytes must be in range(0, 256)")
			}
			out[k] = byte(n)
		}
		return &object.Bytes{V: out}, nil
	}})
	b.SetStr("list", &object.BuiltinFunc{Name: "list", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return &object.List{}, nil
		}
		items, err := iterate(ii.(*Interp), a[0])
		if err != nil {
			return nil, err
		}
		return &object.List{V: items}, nil
	}})
	b.SetStr("tuple", &object.BuiltinFunc{Name: "tuple", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return &object.Tuple{}, nil
		}
		items, err := iterate(ii.(*Interp), a[0])
		if err != nil {
			return nil, err
		}
		return &object.Tuple{V: items}, nil
	}})
	b.SetStr("dict", &object.BuiltinFunc{Name: "dict", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		d := object.NewDict()
		if len(a) > 0 {
			if src, ok := a[0].(*object.Dict); ok {
				keys, vals := src.Items()
				for k, key := range keys {
					_ = d.Set(key, vals[k])
				}
			} else {
				items, err := iterate(in, a[0])
				if err != nil {
					return nil, err
				}
				for _, pair := range items {
					var k, v object.Object
					switch p := pair.(type) {
					case *object.Tuple:
						if len(p.V) != 2 {
							return nil, object.Errorf(in.valueErr, "dict update sequence element has length %d; 2 required", len(p.V))
						}
						k, v = p.V[0], p.V[1]
					case *object.List:
						if len(p.V) != 2 {
							return nil, object.Errorf(in.valueErr, "dict update sequence element has length %d; 2 required", len(p.V))
						}
						k, v = p.V[0], p.V[1]
					default:
						return nil, object.Errorf(in.typeErr, "cannot convert dictionary update sequence element to 2-tuple")
					}
					if err := d.Set(k, v); err != nil {
						return nil, err
					}
				}
			}
		}
		if kw != nil {
			keys, vals := kw.Items()
			for k, key := range keys {
				_ = d.Set(key, vals[k])
			}
		}
		return d, nil
	}})
	b.SetStr("set", &object.BuiltinFunc{Name: "set", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s := object.NewSet()
		if len(a) > 0 {
			items, err := iterate(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			for _, x := range items {
				_ = s.Add(x)
			}
		}
		return s, nil
	}})
	b.SetStr("frozenset", &object.BuiltinFunc{Name: "frozenset", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s := object.NewFrozenset()
		if len(a) > 0 {
			items, err := iterate(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			for _, x := range items {
				if err := s.Add(x); err != nil {
					return nil, err
				}
			}
		}
		return s, nil
	}})

	// Core functions.
	b.SetStr("print", &object.BuiltinFunc{Name: "print", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		sep := " "
		end := "\n"
		if kw != nil {
			if v, ok := kw.GetStr("sep"); ok {
				if s, ok := v.(*object.Str); ok {
					sep = s.V
				}
			}
			if v, ok := kw.GetStr("end"); ok {
				if s, ok := v.(*object.Str); ok {
					end = s.V
				}
			}
		}
		parts := make([]string, len(a))
		for k, x := range a {
			parts[k] = object.Str_(x)
		}
		fmt.Fprint(in.Stdout, strings.Join(parts, sep)+end)
		return object.None, nil
	}})
	b.SetStr("repr", &object.BuiltinFunc{Name: "repr", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return &object.Str{V: ""}, nil
		}
		return &object.Str{V: object.Repr(a[0])}, nil
	}})
	b.SetStr("len", &object.BuiltinFunc{Name: "len", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, err := ii.(*Interp).length(a[0])
		if err != nil {
			return nil, err
		}
		return object.NewInt(n), nil
	}})
	b.SetStr("range", &object.BuiltinFunc{Name: "range", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		var start, stop, step int64 = 0, 0, 1
		switch len(a) {
		case 1:
			n, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(in.typeErr, "range arg must be int")
			}
			stop = n
		case 2:
			n, _ := toInt64(a[0])
			m, _ := toInt64(a[1])
			start, stop = n, m
		case 3:
			n, _ := toInt64(a[0])
			m, _ := toInt64(a[1])
			s, _ := toInt64(a[2])
			start, stop, step = n, m, s
		default:
			return nil, object.Errorf(in.typeErr, "range requires 1..3 args")
		}
		if step == 0 {
			return nil, object.Errorf(in.valueErr, "range step cannot be zero")
		}
		return &object.Range{Start: start, Stop: stop, Step: step}, nil
	}})
	b.SetStr("iter", &object.BuiltinFunc{Name: "iter", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return ii.(*Interp).getIter(a[0])
	}})
	b.SetStr("next", &object.BuiltinFunc{Name: "next", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if gen, ok := a[0].(*object.Generator); ok {
			v, err := in.resumeGenerator(gen, object.None)
			if err != nil {
				if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, in.stopIter) {
					if len(a) > 1 {
						return a[1], nil
					}
				}
				return nil, err
			}
			return v, nil
		}
		it, ok := a[0].(*object.Iter)
		if !ok {
			return nil, object.Errorf(in.typeErr, "next() arg is not an iterator")
		}
		v, ok2, err := it.Next()
		if err != nil {
			return nil, err
		}
		if !ok2 {
			if len(a) > 1 {
				return a[1], nil
			}
			return nil, object.Errorf(in.stopIter, "")
		}
		return v, nil
	}})
	b.SetStr("abs", &object.BuiltinFunc{Name: "abs", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		switch v := a[0].(type) {
		case *object.Int:
			return &object.Int{V: new(big.Int).Abs(v.V)}, nil
		case *object.Float:
			if v.V < 0 {
				return &object.Float{V: -v.V}, nil
			}
			return v, nil
		case *object.Bool:
			if v.V {
				return object.NewInt(1), nil
			}
			return object.NewInt(0), nil
		case *object.Complex:
			return &object.Float{V: math.Hypot(v.Real, v.Imag)}, nil
		}
		return nil, object.Errorf(ii.(*Interp).typeErr, "bad abs() arg")
	}})
	b.SetStr("complex", &object.BuiltinFunc{Name: "complex", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		re, im := 0.0, 0.0
		if len(a) >= 1 {
			r, i2, ok := asComplex(a[0])
			if !ok {
				return nil, object.Errorf(in.typeErr, "complex() first argument must be a number, not '%s'", object.TypeName(a[0]))
			}
			re, im = r, i2
		}
		if len(a) >= 2 {
			br, bi, ok := asComplex(a[1])
			if !ok {
				return nil, object.Errorf(in.typeErr, "complex() second argument must be a number, not '%s'", object.TypeName(a[1]))
			}
			// complex(r, i): result is (r.real - i.imag) + (r.imag + i.real)j
			re, im = re-bi, im+br
		}
		return &object.Complex{Real: re, Imag: im}, nil
	}})
	b.SetStr("min", &object.BuiltinFunc{Name: "min", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return reduceMinMax(ii.(*Interp), a, true)
	}})
	b.SetStr("max", &object.BuiltinFunc{Name: "max", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return reduceMinMax(ii.(*Interp), a, false)
	}})
	b.SetStr("sum", &object.BuiltinFunc{Name: "sum", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		items, err := iterate(in, a[0])
		if err != nil {
			return nil, err
		}
		var acc object.Object = object.NewInt(0)
		if len(a) > 1 {
			acc = a[1]
		}
		for _, x := range items {
			acc, err = in.add(acc, x)
			if err != nil {
				return nil, err
			}
		}
		return acc, nil
	}})
	b.SetStr("sorted", &object.BuiltinFunc{Name: "sorted", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		items, err := iterate(in, a[0])
		if err != nil {
			return nil, err
		}
		cp := make([]object.Object, len(items))
		copy(cp, items)
		reverse := false
		if kw != nil {
			if v, ok := kw.GetStr("reverse"); ok {
				reverse = object.Truthy(v)
			}
		}
		if err := sortList(in, cp, reverse); err != nil {
			return nil, err
		}
		return &object.List{V: cp}, nil
	}})
	b.SetStr("reversed", &object.BuiltinFunc{Name: "reversed", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		items, err := iterate(ii.(*Interp), a[0])
		if err != nil {
			return nil, err
		}
		out := make([]object.Object, len(items))
		for k := range items {
			out[k] = items[len(items)-1-k]
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(out) {
				return nil, false, nil
			}
			r := out[idx]
			idx++
			return r, true, nil
		}}, nil
	}})
	b.SetStr("enumerate", &object.BuiltinFunc{Name: "enumerate", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		it, err := in.getIter(a[0])
		if err != nil {
			return nil, err
		}
		start := int64(0)
		if len(a) > 1 {
			if n, ok := toInt64(a[1]); ok {
				start = n
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("start"); ok {
				if n, ok := toInt64(v); ok {
					start = n
				}
			}
		}
		return &object.Iter{Next: func() (object.Object, bool, error) {
			v, ok, err := it.Next()
			if err != nil || !ok {
				return nil, ok, err
			}
			t := &object.Tuple{V: []object.Object{object.NewInt(start), v}}
			start++
			return t, true, nil
		}}, nil
	}})
	b.SetStr("zip", &object.BuiltinFunc{Name: "zip", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		iters := make([]*object.Iter, len(a))
		for k, x := range a {
			it, err := in.getIter(x)
			if err != nil {
				return nil, err
			}
			iters[k] = it
		}
		return &object.Iter{Next: func() (object.Object, bool, error) {
			items := make([]object.Object, len(iters))
			for k, it := range iters {
				v, ok, err := it.Next()
				if err != nil {
					return nil, false, err
				}
				if !ok {
					return nil, false, nil
				}
				items[k] = v
			}
			return &object.Tuple{V: items}, true, nil
		}}, nil
	}})
	b.SetStr("map", &object.BuiltinFunc{Name: "map", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		fn := a[0]
		iters := make([]*object.Iter, len(a)-1)
		for k, x := range a[1:] {
			it, err := in.getIter(x)
			if err != nil {
				return nil, err
			}
			iters[k] = it
		}
		return &object.Iter{Next: func() (object.Object, bool, error) {
			args := make([]object.Object, len(iters))
			for k, it := range iters {
				v, ok, err := it.Next()
				if err != nil || !ok {
					return nil, ok, err
				}
				args[k] = v
			}
			r, err := in.callObject(fn, args, nil)
			if err != nil {
				return nil, false, err
			}
			return r, true, nil
		}}, nil
	}})
	b.SetStr("filter", &object.BuiltinFunc{Name: "filter", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		fn := a[0]
		it, err := in.getIter(a[1])
		if err != nil {
			return nil, err
		}
		return &object.Iter{Next: func() (object.Object, bool, error) {
			for {
				v, ok, err := it.Next()
				if err != nil || !ok {
					return nil, ok, err
				}
				keep := false
				if _, isNone := fn.(*object.NoneType); isNone {
					keep = object.Truthy(v)
				} else {
					r, err := in.callObject(fn, []object.Object{v}, nil)
					if err != nil {
						return nil, false, err
					}
					keep = object.Truthy(r)
				}
				if keep {
					return v, true, nil
				}
			}
		}}, nil
	}})
	b.SetStr("any", &object.BuiltinFunc{Name: "any", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		items, err := iterate(ii.(*Interp), a[0])
		if err != nil {
			return nil, err
		}
		for _, x := range items {
			if object.Truthy(x) {
				return object.True, nil
			}
		}
		return object.False, nil
	}})
	b.SetStr("all", &object.BuiltinFunc{Name: "all", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		items, err := iterate(ii.(*Interp), a[0])
		if err != nil {
			return nil, err
		}
		for _, x := range items {
			if !object.Truthy(x) {
				return object.False, nil
			}
		}
		return object.True, nil
	}})
	b.SetStr("isinstance", &object.BuiltinFunc{Name: "isinstance", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(isinstance(a[0], a[1])), nil
	}})
	b.SetStr("issubclass", &object.BuiltinFunc{Name: "issubclass", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		ca, okA := a[0].(*object.Class)
		cb, okB := a[1].(*object.Class)
		if !okA || !okB {
			return object.False, nil
		}
		return object.BoolOf(object.IsSubclass(ca, cb)), nil
	}})
	b.SetStr("type", &object.BuiltinFunc{Name: "type", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if inst, ok := a[0].(*object.Instance); ok {
			return inst.Class, nil
		}
		if exc, ok := a[0].(*object.Exception); ok {
			return exc.Class, nil
		}
		// Fallback: synthesize a bare class with just the name; good enough
		// for `type(x).__name__` on builtin types.
		return &object.Class{Name: object.TypeName(a[0]), Dict: object.NewDict()}, nil
	}})
	b.SetStr("id", &object.BuiltinFunc{Name: "id", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(fmt.Sprintf("%p", a[0])[2] ^ 0x42)), nil
	}})
	b.SetStr("hash", &object.BuiltinFunc{Name: "hash", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		h, err := object.Hash(a[0])
		if err != nil {
			return nil, object.Errorf(ii.(*Interp).typeErr, "unhashable type: '%s'", object.TypeName(a[0]))
		}
		return object.NewInt(int64(h)), nil
	}})
	b.SetStr("ord", &object.BuiltinFunc{Name: "ord", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(ii.(*Interp).typeErr, "ord expects str")
		}
		rs := s.Runes()
		if len(rs) != 1 {
			return nil, object.Errorf(ii.(*Interp).typeErr, "ord expects single char")
		}
		return object.NewInt(int64(rs[0])), nil
	}})
	b.SetStr("chr", &object.BuiltinFunc{Name: "chr", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, _ := toInt64(a[0])
		return &object.Str{V: string(rune(n))}, nil
	}})
	b.SetStr("hex", &object.BuiltinFunc{Name: "hex", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, _ := toBigInt(a[0])
		if n.Sign() < 0 {
			return &object.Str{V: "-0x" + new(big.Int).Abs(n).Text(16)}, nil
		}
		return &object.Str{V: "0x" + n.Text(16)}, nil
	}})
	b.SetStr("oct", &object.BuiltinFunc{Name: "oct", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, _ := toBigInt(a[0])
		if n.Sign() < 0 {
			return &object.Str{V: "-0o" + new(big.Int).Abs(n).Text(8)}, nil
		}
		return &object.Str{V: "0o" + n.Text(8)}, nil
	}})
	b.SetStr("bin", &object.BuiltinFunc{Name: "bin", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, _ := toBigInt(a[0])
		if n.Sign() < 0 {
			return &object.Str{V: "-0b" + new(big.Int).Abs(n).Text(2)}, nil
		}
		return &object.Str{V: "0b" + n.Text(2)}, nil
	}})
	b.SetStr("round", &object.BuiltinFunc{Name: "round", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		ndigits := 0
		if len(a) > 1 {
			if n, ok := toInt64(a[1]); ok {
				ndigits = int(n)
			}
		}
		switch v := a[0].(type) {
		case *object.Float:
			mult := 1.0
			for k := 0; k < ndigits; k++ {
				mult *= 10
			}
			for k := 0; k < -ndigits; k++ {
				mult /= 10
			}
			r := v.V * mult
			// Python's banker's rounding; we approximate with math.Round.
			if r >= 0 {
				r = float64(int64(r + 0.5))
			} else {
				r = float64(int64(r - 0.5))
			}
			return &object.Float{V: r / mult}, nil
		case *object.Int:
			return v, nil
		case *object.Bool:
			if v.V {
				return object.NewInt(1), nil
			}
			return object.NewInt(0), nil
		}
		return nil, nil
	}})
	b.SetStr("getattr", &object.BuiltinFunc{Name: "getattr", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		name := a[1].(*object.Str).V
		v, err := ii.(*Interp).getAttr(a[0], name)
		if err != nil {
			if len(a) > 2 {
				return a[2], nil
			}
			return nil, err
		}
		return v, nil
	}})
	b.SetStr("setattr", &object.BuiltinFunc{Name: "setattr", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, ii.(*Interp).setAttr(a[0], a[1].(*object.Str).V, a[2])
	}})
	b.SetStr("hasattr", &object.BuiltinFunc{Name: "hasattr", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		_, err := ii.(*Interp).getAttr(a[0], a[1].(*object.Str).V)
		return object.BoolOf(err == nil), nil
	}})
	b.SetStr("property", &object.BuiltinFunc{Name: "property", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p := &object.Property{}
		if len(a) > 0 {
			p.Fget = a[0]
		}
		if len(a) > 1 {
			p.Fset = a[1]
		}
		if len(a) > 2 {
			p.Fdel = a[2]
		}
		return p, nil
	}})
	b.SetStr("classmethod", &object.BuiltinFunc{Name: "classmethod", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.ClassMethod{Fn: a[0]}, nil
	}})
	b.SetStr("staticmethod", &object.BuiltinFunc{Name: "staticmethod", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.StaticMethod{Fn: a[0]}, nil
	}})
	// super is a stub — LOAD_SUPER_ATTR handles the real work by receiving
	// (super, __class__, self) on the stack.
	b.SetStr("super", &object.BuiltinFunc{Name: "super", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	b.SetStr("callable", &object.BuiltinFunc{Name: "callable", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		switch a[0].(type) {
		case *object.BuiltinFunc, *object.Function, *object.BoundMethod, *object.Class:
			return object.True, nil
		}
		return object.False, nil
	}})
	b.SetStr("divmod", &object.BuiltinFunc{Name: "divmod", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		q, err := in.floordiv(a[0], a[1])
		if err != nil {
			return nil, err
		}
		r, err := in.mod(a[0], a[1])
		if err != nil {
			return nil, err
		}
		return &object.Tuple{V: []object.Object{q, r}}, nil
	}})
	b.SetStr("__build_class__", &object.BuiltinFunc{Name: "__build_class__", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// args: func, name, *bases, **kwds
		in := ii.(*Interp)
		fn := a[0].(*object.Function)
		name := a[1].(*object.Str).V
		var bases []*object.Class
		for _, b := range a[2:] {
			if cls, ok := b.(*object.Class); ok {
				bases = append(bases, cls)
			}
		}
		// Execute class body in a fresh dict.
		ns := object.NewDict()
		frame := NewFrame(fn.Code, fn.Globals, in.Builtins, ns)
		if _, err := in.runFrame(frame); err != nil {
			return nil, err
		}
		cls := &object.Class{Name: name, Bases: bases, Dict: ns}
		// Populate __class__ cell so zero-arg super() in methods resolves.
		if v, ok := ns.GetStr("__classcell__"); ok {
			if c, ok := v.(*object.Cell); ok {
				c.V = cls
				c.Set = true
			}
		}
		return cls, nil
	}})
}

func reduceMinMax(in *Interp, a []object.Object, isMin bool) (object.Object, error) {
	var items []object.Object
	if len(a) == 1 {
		its, err := iterate(in, a[0])
		if err != nil {
			return nil, err
		}
		items = its
	} else {
		items = a
	}
	if len(items) == 0 {
		return nil, object.Errorf(in.valueErr, "min/max of empty sequence")
	}
	best := items[0]
	for _, x := range items[1:] {
		less, err := in.lt(x, best)
		if err != nil {
			return nil, err
		}
		if isMin && less {
			best = x
		}
		if !isMin && !less {
			eq, _ := object.Eq(x, best)
			if !eq {
				best = x
			}
		}
	}
	return best, nil
}

func isinstance(o, t object.Object) bool {
	if cls, ok := t.(*object.Class); ok {
		if inst, ok := o.(*object.Instance); ok {
			return object.IsSubclass(inst.Class, cls)
		}
		if e, ok := o.(*object.Exception); ok {
			return object.IsSubclass(e.Class, cls)
		}
		return false
	}
	if tup, ok := t.(*object.Tuple); ok {
		for _, tt := range tup.V {
			if isinstance(o, tt) {
				return true
			}
		}
	}
	// check by type name for builtin types
	if s, ok := t.(*object.Str); ok {
		return object.TypeName(o) == s.V
	}
	if bf, ok := t.(*object.BuiltinFunc); ok {
		return matchBuiltinType(o, bf.Name)
	}
	return false
}

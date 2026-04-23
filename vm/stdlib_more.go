package vm

import (
	"strings"

	"github.com/tamnd/goipy/object"
)

// --- collections ---

func (i *Interp) buildCollections() *object.Module {
	m := &object.Module{Name: "collections", Dict: object.NewDict()}

	m.Dict.SetStr("deque", &object.BuiltinFunc{Name: "deque", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		dq := &object.Deque{MaxLen: -1}
		if len(a) >= 1 {
			if _, isNone := a[0].(*object.NoneType); !isNone {
				items, err := iterate(i, a[0])
				if err != nil {
					return nil, err
				}
				dq.V = items
			}
		}
		parseMaxlen := func(o object.Object) {
			if _, isNone := o.(*object.NoneType); isNone {
				return
			}
			if n, ok := toInt64(o); ok {
				dq.MaxLen = int(n)
			}
		}
		if len(a) >= 2 {
			parseMaxlen(a[1])
		}
		if kw != nil {
			if v, ok := kw.GetStr("maxlen"); ok {
				parseMaxlen(v)
			}
		}
		// Enforce maxlen at construction: drop leading items.
		if dq.MaxLen >= 0 {
			for len(dq.V) > dq.MaxLen {
				dq.V = dq.V[1:]
			}
		}
		return dq, nil
	}})

	m.Dict.SetStr("Counter", &object.BuiltinFunc{Name: "Counter", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		c := &object.Counter{D: object.NewDict()}
		if len(a) >= 1 {
			if err := addCounts(c.D, a[0], +1, i); err != nil {
				return nil, err
			}
		}
		if kw != nil {
			keys, vals := kw.Items()
			for idx, k := range keys {
				if err := c.D.Set(k, vals[idx]); err != nil {
					return nil, err
				}
			}
		}
		return c, nil
	}})

	m.Dict.SetStr("defaultdict", &object.BuiltinFunc{Name: "defaultdict", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		dd := &object.DefaultDict{D: object.NewDict()}
		if len(a) >= 1 {
			if _, isNone := a[0].(*object.NoneType); !isNone {
				dd.Factory = a[0]
			}
		}
		if len(a) >= 2 {
			switch src := a[1].(type) {
			case *object.Dict:
				keys, vals := src.Items()
				for idx, k := range keys {
					dd.D.Set(k, vals[idx])
				}
			}
		}
		return dd, nil
	}})

	m.Dict.SetStr("OrderedDict", &object.BuiltinFunc{Name: "OrderedDict", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		od := &object.OrderedDict{D: object.NewDict()}
		if len(a) >= 1 {
			switch src := a[0].(type) {
			case *object.Dict:
				keys, vals := src.Items()
				for idx, k := range keys {
					od.D.Set(k, vals[idx])
				}
			case *object.OrderedDict:
				keys, vals := src.D.Items()
				for idx, k := range keys {
					od.D.Set(k, vals[idx])
				}
			default:
				items, err := iterate(i, src)
				if err != nil {
					return nil, err
				}
				for _, x := range items {
					pair, err := iterate(i, x)
					if err != nil {
						return nil, err
					}
					if len(pair) != 2 {
						return nil, object.Errorf(i.valueErr, "OrderedDict update needs key-value pair")
					}
					od.D.Set(pair[0], pair[1])
				}
			}
		}
		if kw != nil {
			keys, vals := kw.Items()
			for idx, k := range keys {
				od.D.Set(k, vals[idx])
			}
		}
		return od, nil
	}})

	// namedtuple(typename, fields[, rename=..., defaults=...]) — returns a
	// class whose instances expose fields by name and by position.
	m.Dict.SetStr("namedtuple", &object.BuiltinFunc{Name: "namedtuple", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "namedtuple() missing typename/field_names")
		}
		typeName, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "typename must be str")
		}
		var fieldNames []string
		switch f := a[1].(type) {
		case *object.Str:
			for _, name := range strings.FieldsFunc(f.V, func(r rune) bool { return r == ',' || r == ' ' }) {
				name = strings.TrimSpace(name)
				if name != "" {
					fieldNames = append(fieldNames, name)
				}
			}
		default:
			items, err := iterate(i, a[1])
			if err != nil {
				return nil, err
			}
			for _, it := range items {
				s, ok := it.(*object.Str)
				if !ok {
					return nil, object.Errorf(i.typeErr, "field name must be str")
				}
				fieldNames = append(fieldNames, s.V)
			}
		}
		defaults := []object.Object{}
		if kw != nil {
			if v, ok := kw.GetStr("defaults"); ok {
				if _, isNone := v.(*object.NoneType); !isNone {
					items, err := iterate(i, v)
					if err != nil {
						return nil, err
					}
					defaults = items
				}
			}
		}
		return i.makeNamedTuple(typeName.V, fieldNames, defaults), nil
	}})

	i.extendCollections(m)
	return m
}

// makeNamedTuple builds a Class whose __init__ accepts positional or
// keyword arguments for each field name, stores them on the instance, and
// whose __repr__/_asdict/_replace mirror CPython namedtuple semantics.
func (i *Interp) makeNamedTuple(typeName string, fields []string, defaults []object.Object) *object.Class {
	cls := &object.Class{Name: typeName, Dict: object.NewDict()}
	cls.Dict.SetStr("_fields", tupleOfStrs(fields))

	// __init__ binds each field from positional or kw args (fallback to
	// defaults if supplied).
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "missing self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "self must be instance")
		}
		pos := a[1:]
		// Defaults align with the *last* N fields.
		baseDefaultIdx := len(fields) - len(defaults)
		for idx, name := range fields {
			var v object.Object
			if idx < len(pos) {
				v = pos[idx]
			} else if kw != nil {
				if x, ok := kw.GetStr(name); ok {
					v = x
				}
			}
			if v == nil {
				if idx >= baseDefaultIdx {
					v = defaults[idx-baseDefaultIdx]
				} else {
					return nil, object.Errorf(i.typeErr, "missing field: %s", name)
				}
			}
			inst.Dict.SetStr(name, v)
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		parts := make([]string, len(fields))
		for k, name := range fields {
			v, _ := inst.Dict.GetStr(name)
			parts[k] = name + "=" + object.Repr(v)
		}
		return &object.Str{V: typeName + "(" + strings.Join(parts, ", ") + ")"}, nil
	}})

	cls.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(fields) {
				return nil, false, nil
			}
			v, _ := inst.Dict.GetStr(fields[idx])
			idx++
			return v, true, nil
		}}, nil
	}})

	cls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(len(fields))), nil
	}})

	cls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		n, ok := toInt64(a[1])
		if !ok {
			return nil, object.Errorf(i.typeErr, "tuple indices must be integers")
		}
		L := int64(len(fields))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return nil, object.Errorf(i.indexErr, "tuple index out of range")
		}
		v, _ := inst.Dict.GetStr(fields[n])
		return v, nil
	}})

	// namedtuples inherit from tuple in CPython, so equality is tuple
	// equality: compare element-by-element regardless of which namedtuple
	// class produced the other value.
	cls.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		left := make([]object.Object, len(fields))
		for k, name := range fields {
			v, _ := inst.Dict.GetStr(name)
			left[k] = v
		}
		var right []object.Object
		switch o := a[1].(type) {
		case *object.Tuple:
			right = o.V
		case *object.Instance:
			if _, ok := o.Class.Dict.GetStr("_fields"); !ok {
				return object.NotImplemented, nil
			}
			fieldsAttr, _ := o.Class.Dict.GetStr("_fields")
			ft, _ := fieldsAttr.(*object.Tuple)
			if ft == nil {
				return object.NotImplemented, nil
			}
			right = make([]object.Object, len(ft.V))
			for k, fv := range ft.V {
				s, _ := fv.(*object.Str)
				if s == nil {
					return object.NotImplemented, nil
				}
				v, _ := o.Dict.GetStr(s.V)
				right[k] = v
			}
		default:
			return object.NotImplemented, nil
		}
		if len(left) != len(right) {
			return object.BoolOf(false), nil
		}
		for k := range left {
			eq, err := object.Eq(left[k], right[k])
			if err != nil {
				return nil, err
			}
			if !eq {
				return object.BoolOf(false), nil
			}
		}
		return object.BoolOf(true), nil
	}})

	cls.Dict.SetStr("_asdict", &object.BuiltinFunc{Name: "_asdict", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		d := object.NewDict()
		for _, name := range fields {
			v, _ := inst.Dict.GetStr(name)
			d.Set(&object.Str{V: name}, v)
		}
		return d, nil
	}})

	cls.Dict.SetStr("_replace", &object.BuiltinFunc{Name: "_replace", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		newInst := &object.Instance{Class: cls, Dict: object.NewDict()}
		for _, name := range fields {
			v, _ := inst.Dict.GetStr(name)
			newInst.Dict.SetStr(name, v)
		}
		if kw != nil {
			keys, vals := kw.Items()
			for idx, k := range keys {
				s, _ := k.(*object.Str)
				if s == nil {
					continue
				}
				newInst.Dict.SetStr(s.V, vals[idx])
			}
		}
		return newInst, nil
	}})

	// _make(iterable) — class method that creates a new instance from iterable.
	cls.Dict.SetStr("_make", &object.BuiltinFunc{Name: "_make", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var src object.Object
		if len(a) >= 1 {
			src = a[0]
		} else {
			return nil, object.Errorf(i.typeErr, "_make() takes 1 argument")
		}
		items, err := iterate(i, src)
		if err != nil {
			return nil, err
		}
		if len(items) != len(fields) {
			return nil, object.Errorf(i.typeErr, "_make() takes %d-field iterable", len(fields))
		}
		inst := &object.Instance{Class: cls, Dict: object.NewDict()}
		for idx, name := range fields {
			inst.Dict.SetStr(name, items[idx])
		}
		return inst, nil
	}})

	// _field_defaults — dict of field→default for fields that have defaults.
	{
		baseDefaultIdx := len(fields) - len(defaults)
		fd := object.NewDict()
		for idx := baseDefaultIdx; idx < len(fields); idx++ {
			fd.Set(&object.Str{V: fields[idx]}, defaults[idx-baseDefaultIdx])
		}
		cls.Dict.SetStr("_field_defaults", fd)
	}

	cls.Dict.SetStr("count", &object.BuiltinFunc{Name: "count", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "count() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		target := a[1]
		c := int64(0)
		for _, name := range fields {
			v, _ := inst.Dict.GetStr(name)
			eq, err := object.Eq(v, target)
			if err != nil {
				return nil, err
			}
			if eq {
				c++
			}
		}
		return object.NewInt(c), nil
	}})

	cls.Dict.SetStr("index", &object.BuiltinFunc{Name: "index", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "index() takes at least 1 argument")
		}
		inst := a[0].(*object.Instance)
		target := a[1]
		start, stop := 0, len(fields)
		if len(a) >= 3 {
			if n, ok := toInt64(a[2]); ok {
				start = int(n)
			}
		}
		if len(a) >= 4 {
			if n, ok := toInt64(a[3]); ok {
				stop = int(n)
			}
		}
		if start < 0 {
			start = 0
		}
		if stop > len(fields) {
			stop = len(fields)
		}
		for idx := start; idx < stop; idx++ {
			v, _ := inst.Dict.GetStr(fields[idx])
			eq, err := object.Eq(v, target)
			if err != nil {
				return nil, err
			}
			if eq {
				return object.NewInt(int64(idx)), nil
			}
		}
		return nil, object.Errorf(i.valueErr, "tuple.index(x): x not in tuple")
	}})

	cls.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		vals := make([]object.Object, len(fields))
		for idx, name := range fields {
			v, _ := inst.Dict.GetStr(name)
			vals[idx] = v
		}
		h, err := object.Hash(&object.Tuple{V: vals})
		if err != nil {
			return nil, err
		}
		return object.NewInt(int64(h)), nil
	}})

	return cls
}

func tupleOfStrs(xs []string) *object.Tuple {
	out := make([]object.Object, len(xs))
	for i, s := range xs {
		out[i] = &object.Str{V: s}
	}
	return &object.Tuple{V: out}
}


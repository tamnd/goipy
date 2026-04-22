package vm

import (
	"strings"

	"github.com/tamnd/goipy/object"
)

// extendCollections adds ChainMap, UserDict, UserList, UserString, and
// fromkeys class methods on Counter and OrderedDict to the collections module.
func (i *Interp) extendCollections(m *object.Module) {
	// Counter.fromkeys raises NotImplementedError.
	if v, ok := m.Dict.GetStr("Counter"); ok {
		if bf, ok := v.(*object.BuiltinFunc); ok {
			if bf.Attrs == nil {
				bf.Attrs = object.NewDict()
			}
			bf.Attrs.SetStr("fromkeys", &object.BuiltinFunc{Name: "fromkeys", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return nil, object.Errorf(i.notImpl, "Counter.fromkeys() is undefined. Use Counter(elem_list) instead.")
			}})
		}
	}

	// OrderedDict.fromkeys(iterable, value=None).
	if v, ok := m.Dict.GetStr("OrderedDict"); ok {
		if bf, ok := v.(*object.BuiltinFunc); ok {
			if bf.Attrs == nil {
				bf.Attrs = object.NewDict()
			}
			bf.Attrs.SetStr("fromkeys", &object.BuiltinFunc{Name: "fromkeys", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) < 1 {
					return nil, object.Errorf(i.typeErr, "fromkeys() takes at least 1 argument")
				}
				keys, err := iterate(i, a[0])
				if err != nil {
					return nil, err
				}
				var val object.Object = object.None
				if len(a) >= 2 {
					val = a[1]
				}
				if kw != nil {
					if v, ok := kw.GetStr("value"); ok {
						val = v
					}
				}
				od := &object.OrderedDict{D: object.NewDict()}
				for _, k := range keys {
					od.D.Set(k, val)
				}
				return od, nil
			}})
		}
	}

	m.Dict.SetStr("ChainMap", &object.BuiltinFunc{Name: "ChainMap", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		maps := make([]object.Object, len(a))
		copy(maps, a)
		if len(maps) == 0 {
			maps = []object.Object{object.NewDict()}
		}
		return i.buildChainMapInst(maps), nil
	}})

	m.Dict.SetStr("UserDict", &object.BuiltinFunc{Name: "UserDict", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		data := object.NewDict()
		if len(a) >= 1 {
			switch src := a[0].(type) {
			case *object.Dict:
				ks, vs := src.Items()
				for idx, k := range ks {
					data.Set(k, vs[idx])
				}
			}
		}
		if kw != nil {
			ks, vs := kw.Items()
			for idx, k := range ks {
				data.Set(k, vs[idx])
			}
		}
		return i.buildUserDictInst(data), nil
	}})

	m.Dict.SetStr("UserList", &object.BuiltinFunc{Name: "UserList", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var items []object.Object
		if len(a) >= 1 {
			var err error
			items, err = iterate(i, a[0])
			if err != nil {
				return nil, err
			}
		}
		data := &object.List{V: items}
		return i.buildUserListInst(data), nil
	}})

	m.Dict.SetStr("UserString", &object.BuiltinFunc{Name: "UserString", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s := ""
		if len(a) >= 1 {
			if sv, ok := a[0].(*object.Str); ok {
				s = sv.V
			} else {
				s = object.Str_(a[0])
			}
		}
		data := &object.Str{V: s}
		return i.buildUserStringInst(data), nil
	}})
}

// --- ChainMap ---

func (i *Interp) buildChainMapInst(maps []object.Object) *object.Instance {
	mapsObj := &object.List{V: append([]object.Object{}, maps...)}

	inst := &object.Instance{
		Class: &object.Class{Name: "ChainMap", Dict: object.NewDict()},
		Dict:  object.NewDict(),
	}
	inst.Dict.SetStr("maps", mapsObj)

	uniqueKeys := func() []object.Object {
		seen := object.NewDict()
		var keys []object.Object
		for _, m := range mapsObj.V {
			var ks []object.Object
			switch d := m.(type) {
			case *object.Dict:
				ks, _ = d.Items()
			case *object.OrderedDict:
				ks, _ = d.D.Items()
			}
			for _, k := range ks {
				if _, ok, _ := seen.Get(k); !ok {
					seen.Set(k, object.None)
					keys = append(keys, k)
				}
			}
		}
		return keys
	}

	searchKey := func(key object.Object) (object.Object, bool, error) {
		for _, m := range mapsObj.V {
			switch d := m.(type) {
			case *object.Dict:
				v, ok, err := d.Get(key)
				if err != nil {
					return nil, false, err
				}
				if ok {
					return v, true, nil
				}
			case *object.OrderedDict:
				v, ok, err := d.D.Get(key)
				if err != nil {
					return nil, false, err
				}
				if ok {
					return v, true, nil
				}
			}
		}
		return nil, false, nil
	}

	firstDict := func() (*object.Dict, bool) {
		if len(mapsObj.V) == 0 {
			return nil, false
		}
		d, ok := mapsObj.V[0].(*object.Dict)
		return d, ok
	}

	inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		parts := make([]string, len(mapsObj.V))
		for k, m := range mapsObj.V {
			parts[k] = object.Repr(m)
		}
		return &object.Str{V: "ChainMap(" + strings.Join(parts, ", ") + ")"}, nil
	}})

	inst.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		v, ok, err := searchKey(a[0])
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
		}
		return v, nil
	}})

	inst.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		d, ok := firstDict()
		if !ok {
			return nil, object.Errorf(i.typeErr, "ChainMap: first map is not a dict")
		}
		return object.None, d.Set(a[0], a[1])
	}})

	inst.Dict.SetStr("__delitem__", &object.BuiltinFunc{Name: "__delitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		d, ok := firstDict()
		if !ok {
			return nil, object.Errorf(i.typeErr, "ChainMap: first map is not a dict")
		}
		existed, err := d.Delete(a[0])
		if err != nil {
			return nil, err
		}
		if !existed {
			return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
		}
		return object.None, nil
	}})

	inst.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		_, ok, err := searchKey(a[0])
		if err != nil {
			return nil, err
		}
		return object.BoolOf(ok), nil
	}})

	inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(len(uniqueKeys()))), nil
	}})

	inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		keys := uniqueKeys()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(keys) {
				return nil, false, nil
			}
			v := keys[idx]
			idx++
			return v, true, nil
		}}, nil
	}})

	inst.Dict.SetStr("get", &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "get() takes at least 1 argument")
		}
		v, ok, err := searchKey(a[0])
		if err != nil {
			return nil, err
		}
		if ok {
			return v, nil
		}
		if len(a) >= 2 {
			return a[1], nil
		}
		return object.None, nil
	}})

	inst.Dict.SetStr("keys", &object.BuiltinFunc{Name: "keys", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.List{V: uniqueKeys()}, nil
	}})

	inst.Dict.SetStr("values", &object.BuiltinFunc{Name: "values", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		keys := uniqueKeys()
		vals := make([]object.Object, len(keys))
		for k, key := range keys {
			v, _, _ := searchKey(key)
			vals[k] = v
		}
		return &object.List{V: vals}, nil
	}})

	inst.Dict.SetStr("items", &object.BuiltinFunc{Name: "items", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		keys := uniqueKeys()
		items := make([]object.Object, len(keys))
		for k, key := range keys {
			v, _, _ := searchKey(key)
			items[k] = &object.Tuple{V: []object.Object{key, v}}
		}
		return &object.List{V: items}, nil
	}})

	inst.Dict.SetStr("new_child", &object.BuiltinFunc{Name: "new_child", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var newMap object.Object = object.NewDict()
		if len(a) >= 1 {
			if _, isNone := a[0].(*object.NoneType); !isNone {
				newMap = a[0]
			}
		}
		newMaps := append([]object.Object{newMap}, mapsObj.V...)
		return i.buildChainMapInst(newMaps), nil
	}})

	// parents is a computed property (not a callable), so store it in the class.
	inst.Class.Dict.SetStr("parents", &object.Property{Fget: &object.BuiltinFunc{Name: "parents", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if len(mapsObj.V) <= 1 {
			return i.buildChainMapInst([]object.Object{object.NewDict()}), nil
		}
		return i.buildChainMapInst(mapsObj.V[1:]), nil
	}}})

	return inst
}

// --- UserDict ---

func (i *Interp) buildUserDictInst(data *object.Dict) *object.Instance {
	inst := &object.Instance{
		Class: &object.Class{Name: "UserDict", Dict: object.NewDict()},
		Dict:  object.NewDict(),
	}
	inst.Dict.SetStr("data", data)

	inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: object.Repr(data)}, nil
	}})

	inst.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		v, ok, err := data.Get(a[0])
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
		}
		return v, nil
	}})

	inst.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, data.Set(a[0], a[1])
	}})

	inst.Dict.SetStr("__delitem__", &object.BuiltinFunc{Name: "__delitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		existed, err := data.Delete(a[0])
		if err != nil {
			return nil, err
		}
		if !existed {
			return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
		}
		return object.None, nil
	}})

	inst.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		_, ok, err := data.Get(a[0])
		if err != nil {
			return nil, err
		}
		return object.BoolOf(ok), nil
	}})

	inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(data.Len())), nil
	}})

	inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		keys, _ := data.Items()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(keys) {
				return nil, false, nil
			}
			v := keys[idx]
			idx++
			return v, true, nil
		}}, nil
	}})

	inst.Dict.SetStr("keys", &object.BuiltinFunc{Name: "keys", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		keys, _ := data.Items()
		return &object.List{V: keys}, nil
	}})

	inst.Dict.SetStr("values", &object.BuiltinFunc{Name: "values", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		_, vals := data.Items()
		return &object.List{V: vals}, nil
	}})

	inst.Dict.SetStr("items", &object.BuiltinFunc{Name: "items", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		keys, vals := data.Items()
		out := make([]object.Object, len(keys))
		for k, key := range keys {
			out[k] = &object.Tuple{V: []object.Object{key, vals[k]}}
		}
		return &object.List{V: out}, nil
	}})

	inst.Dict.SetStr("get", &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "get() takes at least 1 argument")
		}
		v, ok, err := data.Get(a[0])
		if err != nil {
			return nil, err
		}
		if ok {
			return v, nil
		}
		if len(a) >= 2 {
			return a[1], nil
		}
		return object.None, nil
	}})

	inst.Dict.SetStr("pop", &object.BuiltinFunc{Name: "pop", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "pop() takes at least 1 argument")
		}
		v, ok, err := data.Get(a[0])
		if err != nil {
			return nil, err
		}
		if !ok {
			if len(a) >= 2 {
				return a[1], nil
			}
			return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
		}
		data.Delete(a[0])
		return v, nil
	}})

	inst.Dict.SetStr("popitem", &object.BuiltinFunc{Name: "popitem", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		keys, vals := data.Items()
		if len(keys) == 0 {
			return nil, object.Errorf(i.keyErr, "dictionary is empty")
		}
		idx := len(keys) - 1
		k, v := keys[idx], vals[idx]
		data.Delete(k)
		return &object.Tuple{V: []object.Object{k, v}}, nil
	}})

	inst.Dict.SetStr("setdefault", &object.BuiltinFunc{Name: "setdefault", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "setdefault() takes at least 1 argument")
		}
		v, ok, err := data.Get(a[0])
		if err != nil {
			return nil, err
		}
		if ok {
			return v, nil
		}
		var def object.Object = object.None
		if len(a) >= 2 {
			def = a[1]
		}
		if err := data.Set(a[0], def); err != nil {
			return nil, err
		}
		return def, nil
	}})

	inst.Dict.SetStr("update", &object.BuiltinFunc{Name: "update", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) >= 1 {
			switch src := a[0].(type) {
			case *object.Dict:
				ks, vs := src.Items()
				for idx, k := range ks {
					data.Set(k, vs[idx])
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
						return nil, object.Errorf(i.valueErr, "update: expected key-value pair")
					}
					data.Set(pair[0], pair[1])
				}
			}
		}
		if kw != nil {
			ks, vs := kw.Items()
			for idx, k := range ks {
				data.Set(k, vs[idx])
			}
		}
		return object.None, nil
	}})

	inst.Dict.SetStr("clear", &object.BuiltinFunc{Name: "clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		keys, _ := data.Items()
		for _, k := range keys {
			data.Delete(k)
		}
		return object.None, nil
	}})

	inst.Dict.SetStr("copy", &object.BuiltinFunc{Name: "copy", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		newData := object.NewDict()
		ks, vs := data.Items()
		for idx, k := range ks {
			newData.Set(k, vs[idx])
		}
		return i.buildUserDictInst(newData), nil
	}})

	return inst
}

// --- UserList ---

func (i *Interp) buildUserListInst(data *object.List) *object.Instance {
	inst := &object.Instance{
		Class: &object.Class{Name: "UserList", Dict: object.NewDict()},
		Dict:  object.NewDict(),
	}
	inst.Dict.SetStr("data", data)

	inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: object.Repr(data)}, nil
	}})

	inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(len(data.V))), nil
	}})

	inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(data.V) {
				return nil, false, nil
			}
			v := data.V[idx]
			idx++
			return v, true, nil
		}}, nil
	}})

	inst.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if sl, ok := a[0].(*object.Slice); ok {
			start, stop, step, err := i.resolveSlice(sl, len(data.V))
			if err != nil {
				return nil, err
			}
			var out []object.Object
			if step == 1 {
				out = append([]object.Object{}, data.V[start:stop]...)
			} else {
				for idx := start; (step > 0 && idx < stop) || (step < 0 && idx > stop); idx += step {
					out = append(out, data.V[idx])
				}
			}
			return i.buildUserListInst(&object.List{V: out}), nil
		}
		n, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "list indices must be integers")
		}
		L := int64(len(data.V))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return nil, object.Errorf(i.indexErr, "list index out of range")
		}
		return data.V[n], nil
	}})

	inst.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "list indices must be integers")
		}
		L := int64(len(data.V))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return nil, object.Errorf(i.indexErr, "list assignment index out of range")
		}
		data.V[n] = a[1]
		return object.None, nil
	}})

	inst.Dict.SetStr("__delitem__", &object.BuiltinFunc{Name: "__delitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "list indices must be integers")
		}
		L := int64(len(data.V))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return nil, object.Errorf(i.indexErr, "list assignment index out of range")
		}
		data.V = append(data.V[:n], data.V[n+1:]...)
		return object.None, nil
	}})

	inst.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		for _, x := range data.V {
			eq, err := object.Eq(x, a[0])
			if err != nil {
				return nil, err
			}
			if eq {
				return object.True, nil
			}
		}
		return object.False, nil
	}})

	inst.Dict.SetStr("__add__", &object.BuiltinFunc{Name: "__add__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var other []object.Object
		switch o := a[0].(type) {
		case *object.List:
			other = o.V
		case *object.Instance:
			if v, ok := o.Dict.GetStr("data"); ok {
				if l, ok := v.(*object.List); ok {
					other = l.V
				}
			}
		default:
			return object.NotImplemented, nil
		}
		out := append(append([]object.Object{}, data.V...), other...)
		return i.buildUserListInst(&object.List{V: out}), nil
	}})

	inst.Dict.SetStr("append", &object.BuiltinFunc{Name: "append", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		data.V = append(data.V, a[0])
		return object.None, nil
	}})

	inst.Dict.SetStr("extend", &object.BuiltinFunc{Name: "extend", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		data.V = append(data.V, items...)
		return object.None, nil
	}})

	inst.Dict.SetStr("insert", &object.BuiltinFunc{Name: "insert", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, _ := toInt64(a[0])
		L := len(data.V)
		idx := int(n)
		if idx < 0 {
			idx = L + idx
		}
		if idx < 0 {
			idx = 0
		}
		if idx > L {
			idx = L
		}
		data.V = append(data.V, nil)
		copy(data.V[idx+1:], data.V[idx:])
		data.V[idx] = a[1]
		return object.None, nil
	}})

	inst.Dict.SetStr("remove", &object.BuiltinFunc{Name: "remove", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		for k, x := range data.V {
			eq, err := object.Eq(x, a[0])
			if err != nil {
				return nil, err
			}
			if eq {
				data.V = append(data.V[:k], data.V[k+1:]...)
				return object.None, nil
			}
		}
		return nil, object.Errorf(i.valueErr, "list.remove(x): x not in list")
	}})

	inst.Dict.SetStr("pop", &object.BuiltinFunc{Name: "pop", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		L := len(data.V)
		if L == 0 {
			return nil, object.Errorf(i.indexErr, "pop from empty list")
		}
		idx := L - 1
		if len(a) >= 1 {
			n, _ := toInt64(a[0])
			idx = int(n)
			if idx < 0 {
				idx = L + idx
			}
		}
		if idx < 0 || idx >= L {
			return nil, object.Errorf(i.indexErr, "list index out of range")
		}
		v := data.V[idx]
		data.V = append(data.V[:idx], data.V[idx+1:]...)
		return v, nil
	}})

	inst.Dict.SetStr("clear", &object.BuiltinFunc{Name: "clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		data.V = nil
		return object.None, nil
	}})

	inst.Dict.SetStr("count", &object.BuiltinFunc{Name: "count", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		c := int64(0)
		for _, x := range data.V {
			eq, err := object.Eq(x, a[0])
			if err != nil {
				return nil, err
			}
			if eq {
				c++
			}
		}
		return object.NewInt(c), nil
	}})

	inst.Dict.SetStr("index", &object.BuiltinFunc{Name: "index", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "index() takes at least 1 argument")
		}
		start, stop := 0, len(data.V)
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok {
				start = int(n)
			}
		}
		if len(a) >= 3 {
			if n, ok := toInt64(a[2]); ok {
				stop = int(n)
			}
		}
		for k := start; k < stop && k < len(data.V); k++ {
			eq, err := object.Eq(data.V[k], a[0])
			if err != nil {
				return nil, err
			}
			if eq {
				return object.NewInt(int64(k)), nil
			}
		}
		return nil, object.Errorf(i.valueErr, "list.index(x): x not in list")
	}})

	inst.Dict.SetStr("sort", &object.BuiltinFunc{Name: "sort", Call: func(_ any, _ []object.Object, kw *object.Dict) (object.Object, error) {
		var key object.Object
		reverse := false
		if kw != nil {
			if v, ok := kw.GetStr("key"); ok {
				if _, isNone := v.(*object.NoneType); !isNone {
					key = v
				}
			}
			if v, ok := kw.GetStr("reverse"); ok {
				reverse = object.Truthy(v)
			}
		}
		return object.None, sortListKey(i, data.V, key, reverse)
	}})

	inst.Dict.SetStr("reverse", &object.BuiltinFunc{Name: "reverse", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		for lo, hi := 0, len(data.V)-1; lo < hi; lo, hi = lo+1, hi-1 {
			data.V[lo], data.V[hi] = data.V[hi], data.V[lo]
		}
		return object.None, nil
	}})

	inst.Dict.SetStr("copy", &object.BuiltinFunc{Name: "copy", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		out := &object.List{V: append([]object.Object{}, data.V...)}
		return i.buildUserListInst(out), nil
	}})

	return inst
}

// --- UserString ---

func (i *Interp) buildUserStringInst(data *object.Str) *object.Instance {
	inst := &object.Instance{
		Class: &object.Class{Name: "UserString", Dict: object.NewDict()},
		Dict:  object.NewDict(),
	}
	inst.Dict.SetStr("data", data)

	// Delegate all str methods through strMethod on data.
	strMethods := []string{
		"upper", "lower", "strip", "lstrip", "rstrip",
		"capitalize", "title", "swapcase",
		"split", "rsplit", "splitlines",
		"join", "replace", "find", "rfind",
		"index", "rindex",
		"startswith", "endswith",
		"count", "encode",
		"isdigit", "isalpha", "isalnum", "isspace",
		"isupper", "islower", "istitle",
		"center", "ljust", "rjust", "zfill",
		"expandtabs", "format", "format_map",
		"partition", "rpartition",
		"removeprefix", "removesuffix",
	}
	for _, name := range strMethods {
		name := name
		if m, ok := strMethod(data, name); ok {
			inst.Dict.SetStr(name, m)
		}
	}

	inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: object.Repr(data)}, nil
	}})

	inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return data, nil
	}})

	inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(len(data.Runes()))), nil
	}})

	inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		rs := data.Runes()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(rs) {
				return nil, false, nil
			}
			r := &object.Str{V: string(rs[idx])}
			idx++
			return r, true, nil
		}}, nil
	}})

	inst.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		rs := data.Runes()
		if sl, ok := a[0].(*object.Slice); ok {
			start, stop, step, err := i.resolveSlice(sl, len(rs))
			if err != nil {
				return nil, err
			}
			var out []rune
			if step == 1 {
				out = rs[start:stop]
			} else {
				for idx := start; (step > 0 && idx < stop) || (step < 0 && idx > stop); idx += step {
					out = append(out, rs[idx])
				}
			}
			return i.buildUserStringInst(&object.Str{V: string(out)}), nil
		}
		n, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "string indices must be integers")
		}
		L := int64(len(rs))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return nil, object.Errorf(i.indexErr, "string index out of range")
		}
		return i.buildUserStringInst(&object.Str{V: string(rs[n])}), nil
	}})

	inst.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var sub string
		switch s := a[0].(type) {
		case *object.Str:
			sub = s.V
		case *object.Instance:
			if v, ok := s.Dict.GetStr("data"); ok {
				if sv, ok := v.(*object.Str); ok {
					sub = sv.V
				}
			}
		}
		return object.BoolOf(strings.Contains(data.V, sub)), nil
	}})

	inst.Dict.SetStr("__add__", &object.BuiltinFunc{Name: "__add__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var other string
		switch s := a[0].(type) {
		case *object.Str:
			other = s.V
		case *object.Instance:
			if v, ok := s.Dict.GetStr("data"); ok {
				if sv, ok := v.(*object.Str); ok {
					other = sv.V
				}
			}
		default:
			return object.NotImplemented, nil
		}
		return i.buildUserStringInst(&object.Str{V: data.V + other}), nil
	}})

	inst.Dict.SetStr("__radd__", &object.BuiltinFunc{Name: "__radd__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var other string
		switch s := a[0].(type) {
		case *object.Str:
			other = s.V
		default:
			return object.NotImplemented, nil
		}
		return i.buildUserStringInst(&object.Str{V: other + data.V}), nil
	}})

	inst.Dict.SetStr("__mul__", &object.BuiltinFunc{Name: "__mul__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, ok := toInt64(a[0])
		if !ok {
			return object.NotImplemented, nil
		}
		if n <= 0 {
			return i.buildUserStringInst(&object.Str{V: ""}), nil
		}
		return i.buildUserStringInst(&object.Str{V: strings.Repeat(data.V, int(n))}), nil
	}})

	inst.Dict.SetStr("__rmul__", &object.BuiltinFunc{Name: "__rmul__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, ok := toInt64(a[0])
		if !ok {
			return object.NotImplemented, nil
		}
		if n <= 0 {
			return i.buildUserStringInst(&object.Str{V: ""}), nil
		}
		return i.buildUserStringInst(&object.Str{V: strings.Repeat(data.V, int(n))}), nil
	}})

	// Comparison operators compare against the underlying string value.
	cmpStr := func(a object.Object) string {
		switch s := a.(type) {
		case *object.Str:
			return s.V
		case *object.Instance:
			if v, ok := s.Dict.GetStr("data"); ok {
				if sv, ok := v.(*object.Str); ok {
					return sv.V
				}
			}
		}
		return ""
	}
	inst.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(data.V == cmpStr(a[0])), nil
	}})
	inst.Dict.SetStr("__ne__", &object.BuiltinFunc{Name: "__ne__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(data.V != cmpStr(a[0])), nil
	}})
	inst.Dict.SetStr("__lt__", &object.BuiltinFunc{Name: "__lt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(data.V < cmpStr(a[0])), nil
	}})
	inst.Dict.SetStr("__le__", &object.BuiltinFunc{Name: "__le__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(data.V <= cmpStr(a[0])), nil
	}})
	inst.Dict.SetStr("__gt__", &object.BuiltinFunc{Name: "__gt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(data.V > cmpStr(a[0])), nil
	}})
	inst.Dict.SetStr("__ge__", &object.BuiltinFunc{Name: "__ge__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(data.V >= cmpStr(a[0])), nil
	}})

	inst.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		h, err := object.Hash(data)
		if err != nil {
			return nil, err
		}
		return object.NewInt(int64(h)), nil
	}})

	return inst
}

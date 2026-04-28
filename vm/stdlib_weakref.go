package vm

import (
	"sync"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildWeakref() *object.Module {
	m := &object.Module{Name: "weakref", Dict: object.NewDict()}

	// --- ref table: track weak refs per object (for getweakrefcount/getweakrefs) ---
	// tableMu guards table for concurrent goroutine access (no-GIL threading).
	type refSlice struct{ refs []*object.PyWeakRef }
	var tableMu sync.Mutex
	table := map[object.Object]*refSlice{}

	isWeakRefable := func(o object.Object) bool {
		switch o.(type) {
		case *object.Instance, *object.Function, *object.Class:
			return true
		}
		return false
	}

	// addRef appends r under target's slice; tableMu must be held.
	addRef := func(target object.Object, r *object.PyWeakRef) {
		if e, ok := table[target]; ok {
			e.refs = append(e.refs, r)
		} else {
			table[target] = &refSlice{refs: []*object.PyWeakRef{r}}
		}
	}

	// canonical(target) returns the existing no-callback ref if present.
	// tableMu must be held.
	canonical := func(target object.Object) *object.PyWeakRef {
		if e, ok := table[target]; ok {
			for _, r := range e.refs {
				if r.Callback == nil {
					return r
				}
			}
		}
		return nil
	}

	// --- ref() ---
	refFn := &object.BuiltinFunc{Name: "weakref.ref", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "ref() takes at least 1 argument")
		}
		target := a[0]
		if !isWeakRefable(target) {
			return nil, object.Errorf(i.typeErr, "cannot create weak reference to '%s' object", object.TypeName(target))
		}
		var cb object.Object
		if len(a) >= 2 {
			cb = a[1]
		}
		tableMu.Lock()
		defer tableMu.Unlock()
		// Canonical ref: reuse existing no-callback ref.
		if cb == nil || isNoneObj(cb) {
			if existing := canonical(target); existing != nil {
				return existing, nil
			}
		}
		r := &object.PyWeakRef{Target: target, Callback: cb}
		if isNoneObj(cb) {
			r.Callback = nil
		}
		addRef(target, r)
		return r, nil
	}}
	m.Dict.SetStr("ref", refFn)

	// --- ReferenceType ---
	refType := &object.Class{Name: "weakref.ReferenceType", Dict: object.NewDict()}
	refType.ABCCheck = func(o object.Object) bool {
		_, ok := o.(*object.PyWeakRef)
		return ok
	}
	m.Dict.SetStr("ReferenceType", refType)

	// --- proxy() ---
	proxyFn := &object.BuiltinFunc{Name: "proxy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "proxy() takes at least 1 argument")
		}
		target := a[0]
		if !isWeakRefable(target) {
			return nil, object.Errorf(i.typeErr, "cannot create weak reference to '%s' object", object.TypeName(target))
		}
		callable := false
		switch target.(type) {
		case *object.Function, *object.Class:
			callable = true
		case *object.Instance:
			if inst, ok := target.(*object.Instance); ok {
				if _, has := classLookup(inst.Class, "__call__"); has {
					callable = true
				}
			}
		}
		return &object.PyProxy{Target: target, Callable: callable}, nil
	}}
	m.Dict.SetStr("proxy", proxyFn)

	// --- ProxyType / CallableProxyType / ProxyTypes ---
	proxyType := &object.Class{Name: "ProxyType", Dict: object.NewDict()}
	proxyType.ABCCheck = func(o object.Object) bool {
		if px, ok := o.(*object.PyProxy); ok {
			return !px.Callable
		}
		return false
	}
	callableProxyType := &object.Class{Name: "CallableProxyType", Dict: object.NewDict()}
	callableProxyType.ABCCheck = func(o object.Object) bool {
		if px, ok := o.(*object.PyProxy); ok {
			return px.Callable
		}
		return false
	}
	m.Dict.SetStr("ProxyType", proxyType)
	m.Dict.SetStr("CallableProxyType", callableProxyType)
	m.Dict.SetStr("ProxyTypes", &object.Tuple{V: []object.Object{proxyType, callableProxyType}})

	// --- getweakrefcount() ---
	m.Dict.SetStr("getweakrefcount", &object.BuiltinFunc{Name: "getweakrefcount", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "getweakrefcount() takes 1 argument")
		}
		tableMu.Lock()
		defer tableMu.Unlock()
		if e, ok := table[a[0]]; ok {
			return object.NewInt(int64(len(e.refs))), nil
		}
		return object.NewInt(0), nil
	}})

	// --- getweakrefs() ---
	m.Dict.SetStr("getweakrefs", &object.BuiltinFunc{Name: "getweakrefs", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "getweakrefs() takes 1 argument")
		}
		tableMu.Lock()
		defer tableMu.Unlock()
		if e, ok := table[a[0]]; ok {
			out := make([]object.Object, len(e.refs))
			for k, r := range e.refs {
				out[k] = r
			}
			return &object.List{V: out}, nil
		}
		return &object.List{}, nil
	}})

	// --- finalize ---
	finalizeFn := &object.BuiltinFunc{Name: "finalize", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "finalize() takes at least 2 arguments")
		}
		target := a[0]
		fn := a[1]
		args := make([]object.Object, 0)
		if len(a) > 2 {
			args = append(args, a[2:]...)
		}
		return &object.PyFinalizer{
			Target: target,
			Fn:     fn,
			Args:   args,
			Kwargs: kw,
			Alive:  true,
			Atexit: true,
		}, nil
	}}
	m.Dict.SetStr("finalize", finalizeFn)

	// --- WeakMethod ---
	// Weak reference to a bound method. Calling wm() returns the bound method.
	m.Dict.SetStr("WeakMethod", &object.BuiltinFunc{Name: "WeakMethod", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "WeakMethod() takes 1 argument")
		}
		bm, ok := a[0].(*object.BoundMethod)
		if !ok {
			return nil, object.Errorf(i.typeErr, "argument must be a bound method, not '%s'", object.TypeName(a[0]))
		}
		return &object.PyWeakRef{Target: bm, TypeName: "WeakMethod"}, nil
	}})

	// --- WeakValueDictionary ---
	wvdClass := weakContainerClass(i, "WeakValueDictionary")
	m.Dict.SetStr("WeakValueDictionary", &object.BuiltinFunc{Name: "WeakValueDictionary", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := &object.Instance{Class: wvdClass, Dict: object.NewDict()}
		d := object.NewDict()
		inst.Dict.SetStr("_d", d)
		attachDictMethods(i, inst, d)
		// initializer: optional dict or iterable of pairs
		if len(a) >= 1 {
			if err := dictUpdateFrom(i, d, a[0]); err != nil {
				return nil, err
			}
		}
		return inst, nil
	}})

	// --- WeakKeyDictionary ---
	// Uses identity-based (pointer equality) slice since Instance objects
	// are not hashable as Dict keys.
	wkdClass := weakContainerClass(i, "WeakKeyDictionary")
	m.Dict.SetStr("WeakKeyDictionary", &object.BuiltinFunc{Name: "WeakKeyDictionary", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := &object.Instance{Class: wkdClass, Dict: object.NewDict()}
		attachKeyDictMethods(i, inst)
		if len(a) >= 1 {
			if err := keyDictUpdateFrom(i, inst, a[0]); err != nil {
				return nil, err
			}
		}
		return inst, nil
	}})

	// --- WeakSet ---
	wsClass := weakContainerClass(i, "WeakSet")
	m.Dict.SetStr("WeakSet", &object.BuiltinFunc{Name: "WeakSet", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := &object.Instance{Class: wsClass, Dict: object.NewDict()}
		items := []object.Object{}
		inst.Dict.SetStr("_items", &object.List{V: items})
		attachSetMethods(i, inst)
		if len(a) >= 1 {
			elems, err := iterate(i, a[0])
			if err != nil {
				return nil, err
			}
			v, _ := inst.Dict.GetStr("_items")
			lst := v.(*object.List)
			for _, x := range elems {
				if !weakSetContains(lst.V, x) {
					lst.V = append(lst.V, x)
				}
			}
		}
		return inst, nil
	}})

	return m
}

// weakContainerClass creates a bare Class used as the type for the weak container.
func weakContainerClass(_ *Interp, name string) *object.Class {
	return &object.Class{Name: name, Dict: object.NewDict()}
}

// attachDictMethods installs dict-like methods on a WeakValueDictionary /
// WeakKeyDictionary instance, all backed by the Dict d.
func attachDictMethods(i *Interp, inst *object.Instance, d *object.Dict) {
	getd := func() *object.Dict {
		v, _ := inst.Dict.GetStr("_d")
		if dd, ok := v.(*object.Dict); ok {
			return dd
		}
		return d
	}

	inst.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		v, ok, err := getd().Get(a[0])
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
		}
		return v, nil
	}})

	inst.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "__setitem__ requires 2 args")
		}
		return object.None, getd().Set(a[0], a[1])
	}})

	inst.Dict.SetStr("__delitem__", &object.BuiltinFunc{Name: "__delitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		ok, err := getd().Delete(a[0])
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
		}
		return object.None, nil
	}})

	inst.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		_, ok, err := getd().Get(a[0])
		if err != nil {
			return nil, err
		}
		return object.BoolOf(ok), nil
	}})

	inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(getd().Len())), nil
	}})

	inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		keys, _ := getd().Items()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(keys) {
				return nil, false, nil
			}
			k := keys[idx]
			idx++
			return k, true, nil
		}}, nil
	}})

	inst.Dict.SetStr("keys", &object.BuiltinFunc{Name: "keys", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		keys, _ := getd().Items()
		out := make([]object.Object, len(keys))
		copy(out, keys)
		return &object.List{V: out}, nil
	}})

	inst.Dict.SetStr("values", &object.BuiltinFunc{Name: "values", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		_, vals := getd().Items()
		out := make([]object.Object, len(vals))
		copy(out, vals)
		return &object.List{V: out}, nil
	}})

	inst.Dict.SetStr("items", &object.BuiltinFunc{Name: "items", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		keys, vals := getd().Items()
		out := make([]object.Object, len(keys))
		for k := range keys {
			out[k] = &object.Tuple{V: []object.Object{keys[k], vals[k]}}
		}
		return &object.List{V: out}, nil
	}})

	inst.Dict.SetStr("get", &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "get() requires at least 1 argument")
		}
		v, ok, err := getd().Get(a[0])
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
			return nil, object.Errorf(i.typeErr, "pop() requires at least 1 argument")
		}
		v, ok, err := getd().Get(a[0])
		if err != nil {
			return nil, err
		}
		if ok {
			getd().Delete(a[0]) //nolint
			return v, nil
		}
		if len(a) >= 2 {
			return a[1], nil
		}
		return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
	}})

	inst.Dict.SetStr("setdefault", &object.BuiltinFunc{Name: "setdefault", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "setdefault() requires at least 1 argument")
		}
		v, ok, err := getd().Get(a[0])
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
		_ = getd().Set(a[0], def)
		return def, nil
	}})

	inst.Dict.SetStr("update", &object.BuiltinFunc{Name: "update", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		return object.None, dictUpdateFrom(i, getd(), a[0])
	}})

	inst.Dict.SetStr("clear", &object.BuiltinFunc{Name: "clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		dd := getd()
		keys, _ := dd.Items()
		for _, k := range keys {
			dd.Delete(k) //nolint
		}
		return object.None, nil
	}})
}

// attachSetMethods installs WeakSet methods on inst, backed by inst._items (*List).
func attachSetMethods(i *Interp, inst *object.Instance) {
	getItems := func() *object.List {
		v, _ := inst.Dict.GetStr("_items")
		if l, ok := v.(*object.List); ok {
			return l
		}
		return &object.List{}
	}

	inst.Dict.SetStr("add", &object.BuiltinFunc{Name: "add", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "add() requires 1 argument")
		}
		lst := getItems()
		if !weakSetContains(lst.V, a[0]) {
			lst.V = append(lst.V, a[0])
		}
		return object.None, nil
	}})

	inst.Dict.SetStr("discard", &object.BuiltinFunc{Name: "discard", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "discard() requires 1 argument")
		}
		lst := getItems()
		for k, x := range lst.V {
			if eq, _ := object.Eq(x, a[0]); eq {
				lst.V = append(lst.V[:k], lst.V[k+1:]...)
				return object.None, nil
			}
		}
		return object.None, nil
	}})

	inst.Dict.SetStr("remove", &object.BuiltinFunc{Name: "remove", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "remove() requires 1 argument")
		}
		lst := getItems()
		for k, x := range lst.V {
			if eq, _ := object.Eq(x, a[0]); eq {
				lst.V = append(lst.V[:k], lst.V[k+1:]...)
				return object.None, nil
			}
		}
		return nil, object.Errorf(i.keyErr, "WeakSet.remove(x): x not in WeakSet")
	}})

	inst.Dict.SetStr("pop", &object.BuiltinFunc{Name: "pop", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		lst := getItems()
		if len(lst.V) == 0 {
			return nil, object.Errorf(i.keyErr, "pop from an empty set")
		}
		v := lst.V[0]
		lst.V = lst.V[1:]
		return v, nil
	}})

	inst.Dict.SetStr("clear", &object.BuiltinFunc{Name: "clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		lst := getItems()
		lst.V = lst.V[:0]
		return object.None, nil
	}})

	inst.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.False, nil
		}
		return object.BoolOf(weakSetContains(getItems().V, a[0])), nil
	}})

	inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(len(getItems().V))), nil
	}})

	inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		snapshot := append([]object.Object{}, getItems().V...)
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(snapshot) {
				return nil, false, nil
			}
			v := snapshot[idx]
			idx++
			return v, true, nil
		}}, nil
	}})
}

func weakSetContains(items []object.Object, needle object.Object) bool {
	for _, x := range items {
		if eq, _ := object.Eq(x, needle); eq {
			return true
		}
	}
	return false
}

// dictUpdateFrom merges src (dict or iterable of pairs) into dst.
func dictUpdateFrom(i *Interp, dst *object.Dict, src object.Object) error {
	switch s := src.(type) {
	case *object.Dict:
		keys, vals := s.Items()
		for k, key := range keys {
			if err := dst.Set(key, vals[k]); err != nil {
				return err
			}
		}
	case *object.Instance:
		// Try keys() / __getitem__ duck-typing.
		if keysFn, ok := s.Dict.GetStr("keys"); ok {
			keysList, err := i.callObject(keysFn, nil, nil)
			if err != nil {
				return err
			}
			keys, err := iterate(i, keysList)
			if err != nil {
				return err
			}
			for _, k := range keys {
				getitem, ok2 := s.Dict.GetStr("__getitem__")
				if !ok2 {
					break
				}
				v, err := i.callObject(getitem, []object.Object{k}, nil)
				if err != nil {
					return err
				}
				if err := dst.Set(k, v); err != nil {
					return err
				}
			}
		}
	default:
		pairs, err := iterate(i, src)
		if err != nil {
			return err
		}
		for _, p := range pairs {
			pair, err := iterate(i, p)
			if err != nil {
				return err
			}
			if len(pair) != 2 {
				return object.Errorf(i.valueErr, "dictionary update sequence element has wrong length")
			}
			if err := dst.Set(pair[0], pair[1]); err != nil {
				return err
			}
		}
	}
	return nil
}

func isNoneObj(o object.Object) bool {
	_, ok := o.(*object.NoneType)
	return ok
}

// identEntry is one slot in the identity-keyed WeakKeyDictionary.
type identEntry struct {
	key object.Object
	val object.Object
}

// identPairs wraps []identEntry so we can store it in a Dict as an Object.
type identPairs struct{ entries []identEntry }

// attachKeyDictMethods installs dict-like methods on a WeakKeyDictionary
// instance backed by an identity-keyed (pointer equality) slice.
func attachKeyDictMethods(i *Interp, inst *object.Instance) {
	pairs := &identPairs{}
	inst.Dict.SetStr("_pairs", pairs)

	get := func() []identEntry { return pairs.entries }
	idxOf := func(key object.Object) int {
		for k, e := range pairs.entries {
			if e.key == key {
				return k
			}
		}
		return -1
	}

	inst.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if k := idxOf(a[0]); k >= 0 {
			return get()[k].val, nil
		}
		return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
	}})

	inst.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "__setitem__ requires 2 args")
		}
		if k := idxOf(a[0]); k >= 0 {
			pairs.entries[k].val = a[1]
		} else {
			pairs.entries = append(pairs.entries, identEntry{a[0], a[1]})
		}
		return object.None, nil
	}})

	inst.Dict.SetStr("__delitem__", &object.BuiltinFunc{Name: "__delitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		k := idxOf(a[0])
		if k < 0 {
			return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
		}
		pairs.entries = append(pairs.entries[:k], pairs.entries[k+1:]...)
		return object.None, nil
	}})

	inst.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(idxOf(a[0]) >= 0), nil
	}})

	inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(len(get()))), nil
	}})

	inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		snap := append([]identEntry{}, get()...)
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(snap) {
				return nil, false, nil
			}
			k := snap[idx].key
			idx++
			return k, true, nil
		}}, nil
	}})

	inst.Dict.SetStr("keys", &object.BuiltinFunc{Name: "keys", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		es := get()
		out := make([]object.Object, len(es))
		for k, e := range es {
			out[k] = e.key
		}
		return &object.List{V: out}, nil
	}})

	inst.Dict.SetStr("values", &object.BuiltinFunc{Name: "values", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		es := get()
		out := make([]object.Object, len(es))
		for k, e := range es {
			out[k] = e.val
		}
		return &object.List{V: out}, nil
	}})

	inst.Dict.SetStr("items", &object.BuiltinFunc{Name: "items", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		es := get()
		out := make([]object.Object, len(es))
		for k, e := range es {
			out[k] = &object.Tuple{V: []object.Object{e.key, e.val}}
		}
		return &object.List{V: out}, nil
	}})

	inst.Dict.SetStr("get", &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "get() requires at least 1 argument")
		}
		if k := idxOf(a[0]); k >= 0 {
			return get()[k].val, nil
		}
		if len(a) >= 2 {
			return a[1], nil
		}
		return object.None, nil
	}})

	inst.Dict.SetStr("pop", &object.BuiltinFunc{Name: "pop", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "pop() requires at least 1 argument")
		}
		k := idxOf(a[0])
		if k >= 0 {
			v := pairs.entries[k].val
			pairs.entries = append(pairs.entries[:k], pairs.entries[k+1:]...)
			return v, nil
		}
		if len(a) >= 2 {
			return a[1], nil
		}
		return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
	}})

	inst.Dict.SetStr("update", &object.BuiltinFunc{Name: "update", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		return object.None, keyDictUpdateFrom(i, inst, a[0])
	}})

	inst.Dict.SetStr("clear", &object.BuiltinFunc{Name: "clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		pairs.entries = pairs.entries[:0]
		return object.None, nil
	}})
}

// keyDictUpdateFrom merges src into a WeakKeyDictionary inst.
func keyDictUpdateFrom(i *Interp, inst *object.Instance, src object.Object) error {
	setitem, _ := inst.Dict.GetStr("__setitem__")
	switch s := src.(type) {
	case *object.Dict:
		keys, vals := s.Items()
		for k, key := range keys {
			if _, err := i.callObject(setitem, []object.Object{key, vals[k]}, nil); err != nil {
				return err
			}
		}
	default:
		pairs, err := iterate(i, src)
		if err != nil {
			return err
		}
		for _, p := range pairs {
			pair, err := iterate(i, p)
			if err != nil {
				return err
			}
			if len(pair) != 2 {
				return object.Errorf(i.valueErr, "dictionary update sequence element has wrong length")
			}
			if _, err := i.callObject(setitem, pair, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

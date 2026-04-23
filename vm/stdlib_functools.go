package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildFunctools() *object.Module {
	m := &object.Module{Name: "functools", Dict: object.NewDict()}

	// WRAPPER_ASSIGNMENTS matches Python 3.14 (uses __annotate__ and __type_params__)
	wrapperAssignments := &object.Tuple{V: []object.Object{
		&object.Str{V: "__module__"},
		&object.Str{V: "__name__"},
		&object.Str{V: "__qualname__"},
		&object.Str{V: "__doc__"},
		&object.Str{V: "__annotate__"},
		&object.Str{V: "__type_params__"},
	}}
	m.Dict.SetStr("WRAPPER_ASSIGNMENTS", wrapperAssignments)

	// WRAPPER_UPDATES
	wrapperUpdates := &object.Tuple{V: []object.Object{
		&object.Str{V: "__dict__"},
	}}
	m.Dict.SetStr("WRAPPER_UPDATES", wrapperUpdates)

	// setAttrObj writes an attribute into a Python object's attribute storage.
	setAttrObj := func(obj object.Object, name string, val object.Object) {
		switch v := obj.(type) {
		case *object.BuiltinFunc:
			if v.Attrs == nil {
				v.Attrs = object.NewDict()
			}
			v.Attrs.SetStr(name, val)
		case *object.Function:
			if v.Dict == nil {
				v.Dict = object.NewDict()
			}
			v.Dict.SetStr(name, val)
		case *object.Instance:
			v.Dict.SetStr(name, val)
		}
	}

	defaultAssigned := []string{"__module__", "__name__", "__qualname__", "__doc__", "__annotate__", "__type_params__"}
	defaultUpdated := []string{"__dict__"}

	// updateWrapperFn copies attributes from wrapped onto wrapper, then sets __wrapped__.
	updateWrapperFn := func(wrapper, wrapped object.Object, assigned, updated []string) {
		for _, attr := range assigned {
			val, err := i.getAttr(wrapped, attr)
			if err != nil {
				continue
			}
			setAttrObj(wrapper, attr, val)
			if attr == "__name__" {
				if s, ok := val.(*object.Str); ok {
					switch v := wrapper.(type) {
					case *object.BuiltinFunc:
						v.Name = s.V
					case *object.Function:
						v.Name = s.V
					}
				}
			}
		}
		for _, attr := range updated {
			if attr != "__dict__" {
				continue
			}
			wrappedDict, err := i.getAttr(wrapped, "__dict__")
			if err != nil {
				continue
			}
			if d, ok := wrappedDict.(*object.Dict); ok {
				keys, vals := d.Items()
				for idx, k := range keys {
					if ks, ok2 := k.(*object.Str); ok2 {
						setAttrObj(wrapper, ks.V, vals[idx])
					}
				}
			}
		}
		setAttrObj(wrapper, "__wrapped__", wrapped)
	}

	// update_wrapper(wrapper, wrapped, assigned=..., updated=...)
	m.Dict.SetStr("update_wrapper", &object.BuiltinFunc{Name: "update_wrapper", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "update_wrapper() takes at least 2 arguments")
		}
		wrapper, wrapped := a[0], a[1]
		assigned := defaultAssigned
		updated := defaultUpdated
		if kw != nil {
			if v, ok := kw.GetStr("assigned"); ok {
				items, err := iterate(i, v)
				if err != nil {
					return nil, err
				}
				assigned = nil
				for _, item := range items {
					if s, ok2 := item.(*object.Str); ok2 {
						assigned = append(assigned, s.V)
					}
				}
			}
			if v, ok := kw.GetStr("updated"); ok {
				items, err := iterate(i, v)
				if err != nil {
					return nil, err
				}
				updated = nil
				for _, item := range items {
					if s, ok2 := item.(*object.Str); ok2 {
						updated = append(updated, s.V)
					}
				}
			}
		}
		updateWrapperFn(wrapper, wrapped, assigned, updated)
		return wrapper, nil
	}})

	// reduce(function, iterable[, initial])
	m.Dict.SetStr("reduce", &object.BuiltinFunc{Name: "reduce", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 || len(a) > 3 {
			return nil, object.Errorf(i.typeErr, "reduce() takes 2 or 3 arguments")
		}
		fn := a[0]
		items, err := iterate(i, a[1])
		if err != nil {
			return nil, err
		}
		var acc object.Object
		start := 0
		if len(a) == 3 {
			acc = a[2]
		} else {
			if len(items) == 0 {
				return nil, object.Errorf(i.typeErr, "reduce() of empty iterable with no initial value")
			}
			acc = items[0]
			start = 1
		}
		for _, x := range items[start:] {
			acc, err = i.callObject(fn, []object.Object{acc, x}, nil)
			if err != nil {
				return nil, err
			}
		}
		return acc, nil
	}})

	// Placeholder sentinel (Python 3.14)
	placeholderCls := &object.Class{Name: "_PlaceholderType", Bases: []*object.Class{}, Dict: object.NewDict()}
	placeholder := &object.Instance{Class: placeholderCls, Dict: object.NewDict()}
	m.Dict.SetStr("Placeholder", placeholder)

	isPlaceholder := func(v object.Object) bool {
		if inst, ok := v.(*object.Instance); ok {
			return inst.Class == placeholderCls
		}
		return false
	}

	// partial(func, *args, **keywords)
	// Exposes .func, .args, .keywords attributes.
	// Supports Placeholder in bound args (Python 3.14).
	m.Dict.SetStr("partial", &object.BuiltinFunc{Name: "partial", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "partial() takes at least one argument")
		}
		inner := a[0]
		bound := append([]object.Object{}, a[1:]...)
		var boundKw *object.Dict
		if kw != nil {
			boundKw = object.NewDict()
			keys, vals := kw.Items()
			for idx, k := range keys {
				boundKw.Set(k, vals[idx])
			}
		}

		hasPlaceholder := false
		for _, b := range bound {
			if isPlaceholder(b) {
				hasPlaceholder = true
				break
			}
		}

		result := &object.BuiltinFunc{Name: "partial", Call: func(_ any, ca []object.Object, ckw *object.Dict) (object.Object, error) {
			var all []object.Object
			if hasPlaceholder {
				caIdx := 0
				for _, b := range bound {
					if isPlaceholder(b) {
						if caIdx < len(ca) {
							all = append(all, ca[caIdx])
							caIdx++
						} else {
							all = append(all, b)
						}
					} else {
						all = append(all, b)
					}
				}
				all = append(all, ca[caIdx:]...)
			} else {
				all = append([]object.Object{}, bound...)
				all = append(all, ca...)
			}
			mergedKw := boundKw
			if ckw != nil {
				mergedKw = object.NewDict()
				if boundKw != nil {
					bk, bv := boundKw.Items()
					for idx, k := range bk {
						mergedKw.Set(k, bv[idx])
					}
				}
				ck, cv := ckw.Items()
				for idx, k := range ck {
					mergedKw.Set(k, cv[idx])
				}
			}
			return i.callObject(inner, all, mergedKw)
		}}
		result.Attrs = object.NewDict()
		result.Attrs.SetStr("func", inner)
		result.Attrs.SetStr("args", &object.Tuple{V: append([]object.Object{}, bound...)})
		if boundKw != nil {
			result.Attrs.SetStr("keywords", boundKw)
		} else {
			result.Attrs.SetStr("keywords", object.NewDict())
		}
		return result, nil
	}})

	// cmp_to_key(mycmp): convert old-style cmp function to a key class.
	m.Dict.SetStr("cmp_to_key", &object.BuiltinFunc{Name: "cmp_to_key", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "cmp_to_key() takes one argument")
		}
		cmpFn := a[0]
		keyCls := &object.Class{Name: "K", Bases: []*object.Class{}, Dict: object.NewDict()}
		makeCmpMethod := func(op string) *object.BuiltinFunc {
			return &object.BuiltinFunc{Name: op, Call: func(_ any, ia []object.Object, _ *object.Dict) (object.Object, error) {
				if len(ia) < 2 {
					return object.NotImplemented, nil
				}
				si, ok1 := ia[0].(*object.Instance)
				oi, ok2 := ia[1].(*object.Instance)
				if !ok1 || !ok2 {
					return object.NotImplemented, nil
				}
				so, _ := si.Dict.GetStr("obj")
				oo, _ := oi.Dict.GetStr("obj")
				r, err := i.callObject(cmpFn, []object.Object{so, oo}, nil)
				if err != nil {
					return nil, err
				}
				n, _ := toInt64(r)
				switch op {
				case "__lt__":
					return object.BoolOf(n < 0), nil
				case "__le__":
					return object.BoolOf(n <= 0), nil
				case "__eq__":
					return object.BoolOf(n == 0), nil
				case "__ne__":
					return object.BoolOf(n != 0), nil
				case "__gt__":
					return object.BoolOf(n > 0), nil
				default: // "__ge__"
					return object.BoolOf(n >= 0), nil
				}
			}}
		}
		for _, op := range []string{"__lt__", "__le__", "__eq__", "__ne__", "__gt__", "__ge__"} {
			keyCls.Dict.SetStr(op, makeCmpMethod(op))
		}
		return &object.BuiltinFunc{Name: "K", Call: func(_ any, ia []object.Object, _ *object.Dict) (object.Object, error) {
			if len(ia) != 1 {
				return nil, object.Errorf(i.typeErr, "K() takes one argument")
			}
			inst := &object.Instance{Class: keyCls, Dict: object.NewDict()}
			inst.Dict.SetStr("obj", ia[0])
			return inst, nil
		}}, nil
	}})

	// total_ordering: fill in missing comparison methods given __eq__ + one ordering op.
	m.Dict.SetStr("total_ordering", &object.BuiltinFunc{Name: "total_ordering", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "total_ordering() takes one argument")
		}
		cls, ok := a[0].(*object.Class)
		if !ok {
			return nil, object.Errorf(i.typeErr, "total_ordering() requires a class")
		}

		has := func(name string) bool {
			v, ok2 := cls.Dict.GetStr(name)
			return ok2 && v != nil
		}

		// callClsOp calls a method on the class with (self, other).
		callClsOp := func(meth string, self, other object.Object) (bool, error) {
			fn, ok2 := cls.Dict.GetStr(meth)
			if !ok2 || fn == nil {
				return false, nil
			}
			r, err := i.callObject(fn, []object.Object{self, other}, nil)
			if err != nil {
				return false, err
			}
			return object.Truthy(r), nil
		}

		makeMethod := func(name string, body func(self, other object.Object) (bool, error)) *object.BuiltinFunc {
			return &object.BuiltinFunc{Name: name, Call: func(_ any, ia []object.Object, _ *object.Dict) (object.Object, error) {
				if len(ia) < 2 {
					return object.NotImplemented, nil
				}
				r, err := body(ia[0], ia[1])
				if err != nil {
					return nil, err
				}
				return object.BoolOf(r), nil
			}}
		}

		addIfMissing := func(name string, fn *object.BuiltinFunc) {
			if !has(name) {
				cls.Dict.SetStr(name, fn)
			}
		}

		root := ""
		for _, r := range []string{"__lt__", "__le__", "__gt__", "__ge__"} {
			if has(r) {
				root = r
				break
			}
		}
		if root == "" {
			return nil, object.Errorf(i.valueErr, "total_ordering: class must define at least one ordering operation")
		}

		switch root {
		case "__lt__":
			addIfMissing("__le__", makeMethod("__le__", func(s, o object.Object) (bool, error) {
				lt, e := callClsOp("__lt__", s, o)
				if e != nil {
					return false, e
				}
				eq, e := callClsOp("__eq__", s, o)
				return lt || eq, e
			}))
			addIfMissing("__gt__", makeMethod("__gt__", func(s, o object.Object) (bool, error) {
				lt, e := callClsOp("__lt__", s, o)
				if e != nil {
					return false, e
				}
				eq, e := callClsOp("__eq__", s, o)
				return !lt && !eq, e
			}))
			addIfMissing("__ge__", makeMethod("__ge__", func(s, o object.Object) (bool, error) {
				lt, e := callClsOp("__lt__", s, o)
				return !lt, e
			}))
		case "__le__":
			addIfMissing("__lt__", makeMethod("__lt__", func(s, o object.Object) (bool, error) {
				le, e := callClsOp("__le__", s, o)
				if e != nil {
					return false, e
				}
				eq, e := callClsOp("__eq__", s, o)
				return le && !eq, e
			}))
			addIfMissing("__gt__", makeMethod("__gt__", func(s, o object.Object) (bool, error) {
				le, e := callClsOp("__le__", s, o)
				return !le, e
			}))
			addIfMissing("__ge__", makeMethod("__ge__", func(s, o object.Object) (bool, error) {
				le, e := callClsOp("__le__", s, o)
				if e != nil {
					return false, e
				}
				eq, e := callClsOp("__eq__", s, o)
				return !le || eq, e
			}))
		case "__gt__":
			addIfMissing("__lt__", makeMethod("__lt__", func(s, o object.Object) (bool, error) {
				gt, e := callClsOp("__gt__", s, o)
				if e != nil {
					return false, e
				}
				eq, e := callClsOp("__eq__", s, o)
				return !gt && !eq, e
			}))
			addIfMissing("__le__", makeMethod("__le__", func(s, o object.Object) (bool, error) {
				gt, e := callClsOp("__gt__", s, o)
				return !gt, e
			}))
			addIfMissing("__ge__", makeMethod("__ge__", func(s, o object.Object) (bool, error) {
				gt, e := callClsOp("__gt__", s, o)
				if e != nil {
					return false, e
				}
				eq, e := callClsOp("__eq__", s, o)
				return gt || eq, e
			}))
		case "__ge__":
			addIfMissing("__lt__", makeMethod("__lt__", func(s, o object.Object) (bool, error) {
				ge, e := callClsOp("__ge__", s, o)
				return !ge, e
			}))
			addIfMissing("__le__", makeMethod("__le__", func(s, o object.Object) (bool, error) {
				ge, e := callClsOp("__ge__", s, o)
				if e != nil {
					return false, e
				}
				eq, e := callClsOp("__eq__", s, o)
				return !ge || eq, e
			}))
			addIfMissing("__gt__", makeMethod("__gt__", func(s, o object.Object) (bool, error) {
				ge, e := callClsOp("__ge__", s, o)
				if e != nil {
					return false, e
				}
				eq, e := callClsOp("__eq__", s, o)
				return ge && !eq, e
			}))
		}
		return cls, nil
	}})

	// partialmethod(func, *args, **keywords)
	// Returns a descriptor that, when accessed on an instance, returns
	// a partial bound with that instance as the first argument.
	m.Dict.SetStr("partialmethod", &object.BuiltinFunc{Name: "partialmethod", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "partialmethod() takes at least one argument")
		}
		fn := a[0]
		boundArgs := append([]object.Object{}, a[1:]...)
		boundKw := kw

		pmCls := &object.Class{Name: "partialmethod", Bases: []*object.Class{}, Dict: object.NewDict()}
		pmCls.Dict.SetStr("__get__", &object.BuiltinFunc{Name: "__get__", Call: func(_ any, ga []object.Object, _ *object.Dict) (object.Object, error) {
			// ga[0]=self(descriptor), ga[1]=obj(instance or None), ga[2]=owner(cls)
			if len(ga) < 2 {
				return ga[0], nil
			}
			obj := ga[1]
			if _, isNone := obj.(*object.NoneType); isNone {
				return ga[0], nil
			}
			// Build the bound call list: func(obj, *boundArgs, ...)
			boundToInst := append([]object.Object{obj}, boundArgs...)
			return &object.BuiltinFunc{Name: "partialmethod_bound", Call: func(_ any, ca []object.Object, ckw *object.Dict) (object.Object, error) {
				callArgs := append([]object.Object{}, boundToInst...)
				callArgs = append(callArgs, ca...)
				mergedKw := boundKw
				if ckw != nil {
					mergedKw = object.NewDict()
					if boundKw != nil {
						bk, bv := boundKw.Items()
						for idx, k := range bk {
							mergedKw.Set(k, bv[idx])
						}
					}
					ck, cv := ckw.Items()
					for idx, k := range ck {
						mergedKw.Set(k, cv[idx])
					}
				}
				return i.callObject(fn, callArgs, mergedKw)
			}}, nil
		}})

		pmInst := &object.Instance{Class: pmCls, Dict: object.NewDict()}
		pmInst.Dict.SetStr("func", fn)
		pmInst.Dict.SetStr("args", &object.Tuple{V: append([]object.Object{}, boundArgs...)})
		if boundKw != nil {
			pmInst.Dict.SetStr("keywords", boundKw)
		} else {
			pmInst.Dict.SetStr("keywords", object.NewDict())
		}
		return pmInst, nil
	}})

	// singledispatch: single-dispatch generic function decorator.
	//
	// Usage:
	//   @singledispatch
	//   def fun(arg): ...
	//
	//   @fun.register(int)
	//   def _(arg): ...
	//
	// Dispatch order: exact type → subclass (LIFO registration order) → default.
	m.Dict.SetStr("singledispatch", &object.BuiltinFunc{Name: "singledispatch", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "singledispatch() takes one argument")
		}
		defaultFn := a[0]

		type reg struct {
			typ object.Object
			fn  object.Object
		}
		var registrations []reg
		registry := object.NewDict()
		// Register object base (object type) → defaultFn
		// We represent it as a sentinel nil key; but we'll just use defaultFn as fallback.

		dispatchForArg := func(arg object.Object) object.Object {
			for j := len(registrations) - 1; j >= 0; j-- {
				if isinstance(arg, registrations[j].typ) {
					return registrations[j].fn
				}
			}
			return defaultFn
		}

		dispatchForType := func(typ object.Object) object.Object {
			for j := len(registrations) - 1; j >= 0; j-- {
				rt := registrations[j].typ
				// Check if typ is equal to or a subtype of rt
				if rt == typ {
					return registrations[j].fn
				}
				// Both are BuiltinFunc (builtin types like int, str)
				if rtBf, ok1 := rt.(*object.BuiltinFunc); ok1 {
					if tBf, ok2 := typ.(*object.BuiltinFunc); ok2 && rtBf.Name == tBf.Name {
						return registrations[j].fn
					}
				}
				// Both are Class objects
				if rtCls, ok1 := rt.(*object.Class); ok1 {
					if tCls, ok2 := typ.(*object.Class); ok2 && object.IsSubclass(tCls, rtCls) {
						return registrations[j].fn
					}
				}
			}
			return defaultFn
		}

		wrapper := &object.BuiltinFunc{Name: "singledispatch_wrapper"}
		wrapper.Attrs = object.NewDict()

		wrapper.Call = func(_ any, wa []object.Object, wkw *object.Dict) (object.Object, error) {
			if len(wa) == 0 {
				return i.callObject(defaultFn, wa, wkw)
			}
			return i.callObject(dispatchForArg(wa[0]), wa, wkw)
		}

		registerFn := &object.BuiltinFunc{Name: "register", Call: func(_ any, ra []object.Object, _ *object.Dict) (object.Object, error) {
			if len(ra) == 0 {
				return nil, object.Errorf(i.typeErr, "register() takes at least one argument")
			}
			typ := ra[0]
			if len(ra) >= 2 {
				// register(type, func) direct form
				registrations = append(registrations, reg{typ, ra[1]})
				registry.Set(typ, ra[1])
				return ra[1], nil
			}
			// register(type) decorator form
			return &object.BuiltinFunc{Name: "register_decorator", Call: func(_ any, da []object.Object, _ *object.Dict) (object.Object, error) {
				if len(da) != 1 {
					return nil, object.Errorf(i.typeErr, "register decorator takes one argument")
				}
				registrations = append(registrations, reg{typ, da[0]})
				registry.Set(typ, da[0])
				return da[0], nil
			}}, nil
		}}

		wrapper.Attrs.SetStr("register", registerFn)
		wrapper.Attrs.SetStr("dispatch", &object.BuiltinFunc{Name: "dispatch", Call: func(_ any, da []object.Object, _ *object.Dict) (object.Object, error) {
			if len(da) != 1 {
				return nil, object.Errorf(i.typeErr, "dispatch() takes one argument")
			}
			return dispatchForType(da[0]), nil
		}})
		wrapper.Attrs.SetStr("registry", registry)
		updateWrapperFn(wrapper, defaultFn, defaultAssigned, nil)
		return wrapper, nil
	}})

	// singledispatchmethod: singledispatch for methods (dispatches on first non-self arg).
	m.Dict.SetStr("singledispatchmethod", &object.BuiltinFunc{Name: "singledispatchmethod", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "singledispatchmethod() takes one argument")
		}
		fn := a[0]

		type reg struct {
			typ object.Object
			fn  object.Object
		}
		var registrations []reg
		registry := object.NewDict()

		dispatchForArg := func(arg object.Object) object.Object {
			for j := len(registrations) - 1; j >= 0; j-- {
				if isinstance(arg, registrations[j].typ) {
					return registrations[j].fn
				}
			}
			return fn
		}

		dispatchForType := func(typ object.Object) object.Object {
			for j := len(registrations) - 1; j >= 0; j-- {
				rt := registrations[j].typ
				if rt == typ {
					return registrations[j].fn
				}
				if rtBf, ok1 := rt.(*object.BuiltinFunc); ok1 {
					if tBf, ok2 := typ.(*object.BuiltinFunc); ok2 && rtBf.Name == tBf.Name {
						return registrations[j].fn
					}
				}
				if rtCls, ok1 := rt.(*object.Class); ok1 {
					if tCls, ok2 := typ.(*object.Class); ok2 && object.IsSubclass(tCls, rtCls) {
						return registrations[j].fn
					}
				}
			}
			return fn
		}

		registerFn := &object.BuiltinFunc{Name: "register", Call: func(_ any, ra []object.Object, _ *object.Dict) (object.Object, error) {
			if len(ra) == 0 {
				return nil, object.Errorf(i.typeErr, "register() takes at least one argument")
			}
			typ := ra[0]
			if len(ra) >= 2 {
				registrations = append(registrations, reg{typ, ra[1]})
				registry.Set(typ, ra[1])
				return ra[1], nil
			}
			return &object.BuiltinFunc{Name: "register_decorator", Call: func(_ any, da []object.Object, _ *object.Dict) (object.Object, error) {
				if len(da) != 1 {
					return nil, object.Errorf(i.typeErr, "register decorator takes one argument")
				}
				registrations = append(registrations, reg{typ, da[0]})
				registry.Set(typ, da[0])
				return da[0], nil
			}}, nil
		}}

		// Return a descriptor Instance with __get__ that builds a bound dispatch wrapper.
		// register/dispatch/registry go in inst.Dict so they are not bound as methods.
		sdmCls := &object.Class{Name: "singledispatchmethod", Bases: []*object.Class{}, Dict: object.NewDict()}
		sdmCls.Dict.SetStr("__get__", &object.BuiltinFunc{Name: "__get__", Call: func(_ any, ga []object.Object, _ *object.Dict) (object.Object, error) {
			// ga[0]=descriptor_inst, ga[1]=obj(instance or None), ga[2]=owner
			if len(ga) < 2 {
				return ga[0], nil
			}
			obj := ga[1]
			if _, isNone := obj.(*object.NoneType); isNone {
				return ga[0], nil
			}
			// Return a bound dispatcher that dispatches on first arg after self.
			return &object.BuiltinFunc{Name: "sdm_bound", Call: func(_ any, wa []object.Object, wkw *object.Dict) (object.Object, error) {
				if len(wa) == 0 {
					return i.callObject(fn, []object.Object{obj}, wkw)
				}
				impl := dispatchForArg(wa[0])
				allArgs := append([]object.Object{obj}, wa...)
				return i.callObject(impl, allArgs, wkw)
			}}, nil
		}})

		sdmInst := &object.Instance{Class: sdmCls, Dict: object.NewDict()}
		sdmInst.Dict.SetStr("register", registerFn)
		sdmInst.Dict.SetStr("registry", registry)
		sdmInst.Dict.SetStr("dispatch", &object.BuiltinFunc{Name: "dispatch", Call: func(_ any, da []object.Object, _ *object.Dict) (object.Object, error) {
			if len(da) != 1 {
				return nil, object.Errorf(i.typeErr, "dispatch() takes one argument")
			}
			return dispatchForType(da[0]), nil
		}})
		return sdmInst, nil
	}})

	// lru_cache(maxsize=128, typed=False) and cache (unbounded).
	lruCacheFn := func(maxsize int, typed bool) object.Object {
		return &object.BuiltinFunc{Name: "lru_cache_decorator", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "lru_cache decorator takes exactly one argument")
			}
			fn := a[0]
			cache := object.NewDict()
			var order []object.Object
			hits, misses := 0, 0

			wrapper := &object.BuiltinFunc{Name: "lru_wrapper", Call: func(_ any, ca []object.Object, ckw *object.Dict) (object.Object, error) {
				key := buildCacheKey(ca, ckw, typed)
				if v, ok, err := cache.Get(key); err == nil && ok {
					hits++
					for idx, k := range order {
						eq, _ := object.Eq(k, key)
						if eq {
							order = append(order[:idx], order[idx+1:]...)
							break
						}
					}
					order = append(order, key)
					return v, nil
				}
				misses++
				r, err := i.callObject(fn, ca, ckw)
				if err != nil {
					return nil, err
				}
				if err2 := cache.Set(key, r); err2 != nil {
					return nil, err2
				}
				order = append(order, key)
				if maxsize > 0 && len(order) > maxsize {
					oldest := order[0]
					order = order[1:]
					cache.Delete(oldest)
				}
				return r, nil
			}}
			wrapper.Attrs = object.NewDict()
			wrapper.Attrs.SetStr("cache_info", &object.BuiltinFunc{Name: "cache_info", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				ms := object.Object(object.NewInt(int64(maxsize)))
				if maxsize == 0 {
					ms = object.None
				}
				return &object.Tuple{V: []object.Object{
					object.NewInt(int64(hits)),
					object.NewInt(int64(misses)),
					ms,
					object.NewInt(int64(len(order))),
				}}, nil
			}})
			wrapper.Attrs.SetStr("cache_clear", &object.BuiltinFunc{Name: "cache_clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				cache = object.NewDict()
				order = nil
				hits, misses = 0, 0
				return object.None, nil
			}})
			wrapper.Attrs.SetStr("cache_parameters", &object.BuiltinFunc{Name: "cache_parameters", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				d := object.NewDict()
				ms := object.Object(object.NewInt(int64(maxsize)))
				if maxsize == 0 {
					ms = object.None
				}
				d.SetStr("maxsize", ms)
				d.SetStr("typed", object.BoolOf(typed))
				return d, nil
			}})
			wrapper.Attrs.SetStr("__wrapped__", fn)
			updateWrapperFn(wrapper, fn, defaultAssigned, nil)
			return wrapper, nil
		}}
	}

	m.Dict.SetStr("lru_cache", &object.BuiltinFunc{Name: "lru_cache", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		// lru_cache(fn) → wrap with default maxsize=128
		if len(a) == 1 {
			if _, isFunc := a[0].(*object.Function); isFunc {
				return lruCacheFn(128, false).(*object.BuiltinFunc).Call(nil, a, nil)
			}
			if _, isBf := a[0].(*object.BuiltinFunc); isBf {
				return lruCacheFn(128, false).(*object.BuiltinFunc).Call(nil, a, nil)
			}
		}
		maxsize := 128
		typed := false
		if len(a) >= 1 {
			if n, ok := toInt64(a[0]); ok {
				maxsize = int(n)
			} else if _, isNone := a[0].(*object.NoneType); isNone {
				maxsize = 0
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("maxsize"); ok {
				if n, ok2 := toInt64(v); ok2 {
					maxsize = int(n)
				} else if _, isNone := v.(*object.NoneType); isNone {
					maxsize = 0
				}
			}
			if v, ok := kw.GetStr("typed"); ok {
				typed = object.Truthy(v)
			}
		}
		return lruCacheFn(maxsize, typed), nil
	}})

	m.Dict.SetStr("cache", lruCacheFn(0, false))

	// wraps(wrapped, assigned=..., updated=...) — full update_wrapper logic.
	m.Dict.SetStr("wraps", &object.BuiltinFunc{Name: "wraps", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "wraps() takes one argument")
		}
		wrapped := a[0]
		assigned := defaultAssigned
		updated := defaultUpdated
		if kw != nil {
			if v, ok := kw.GetStr("assigned"); ok {
				items, err := iterate(i, v)
				if err != nil {
					return nil, err
				}
				assigned = nil
				for _, item := range items {
					if s, ok2 := item.(*object.Str); ok2 {
						assigned = append(assigned, s.V)
					}
				}
			}
			if v, ok := kw.GetStr("updated"); ok {
				items, err := iterate(i, v)
				if err != nil {
					return nil, err
				}
				updated = nil
				for _, item := range items {
					if s, ok2 := item.(*object.Str); ok2 {
						updated = append(updated, s.V)
					}
				}
			}
		}
		return &object.BuiltinFunc{Name: "wraps_decorator", Call: func(_ any, b []object.Object, _ *object.Dict) (object.Object, error) {
			if len(b) != 1 {
				return nil, object.Errorf(i.typeErr, "wraps decorator takes one argument")
			}
			wrapper := b[0]
			updateWrapperFn(wrapper, wrapped, assigned, updated)
			return wrapper, nil
		}}, nil
	}})

	// cached_property(fn): non-data descriptor that caches fn(inst) in inst.__dict__.
	m.Dict.SetStr("cached_property", &object.BuiltinFunc{Name: "cached_property", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "cached_property() takes one argument")
		}
		return &cachedProperty{fn: a[0], interp: i}, nil
	}})

	return m
}

// buildCacheKey packs positional and keyword arguments into a Tuple usable as a
// dict key. When typed=true, type names are appended after the values so that
// 1 (int) and 1.0 (float) produce distinct keys.
func buildCacheKey(args []object.Object, kw *object.Dict, typed bool) object.Object {
	parts := make([]object.Object, 0, len(args)+1)
	parts = append(parts, args...)
	if kw != nil {
		keys, vals := kw.Items()
		for idx, k := range keys {
			parts = append(parts, k, vals[idx])
		}
	}
	if typed {
		for _, arg := range args {
			parts = append(parts, &object.Str{V: object.TypeName(arg)})
		}
	}
	return &object.Tuple{V: parts}
}

// cachedProperty acts like a non-data descriptor: first access runs
// fn(inst) and stashes the result in inst.__dict__[name], so subsequent
// accesses find it directly in the instance dict (bypassing the descriptor).
type cachedProperty struct {
	fn     object.Object
	name   string
	interp *Interp
}

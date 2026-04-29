package vm

import (
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildTypes() *object.Module {
	m := &object.Module{Name: "types", Dict: object.NewDict()}

	// --- type constants (Class objects with ABCCheck for isinstance) ---

	noneType := &object.Class{Name: "NoneType", Dict: object.NewDict()}
	noneType.ABCCheck = func(o object.Object) bool { _, ok := o.(*object.NoneType); return ok }
	m.Dict.SetStr("NoneType", noneType)

	ellipsisType := &object.Class{Name: "EllipsisType", Dict: object.NewDict()}
	ellipsisType.ABCCheck = func(o object.Object) bool { _, ok := o.(*object.EllipsisType); return ok }
	m.Dict.SetStr("EllipsisType", ellipsisType)

	notImplType := &object.Class{Name: "NotImplementedType", Dict: object.NewDict()}
	notImplType.ABCCheck = func(o object.Object) bool { _, ok := o.(*object.NotImplementedType); return ok }
	m.Dict.SetStr("NotImplementedType", notImplType)

	funcType := &object.Class{Name: "function", Dict: object.NewDict()}
	funcType.ABCCheck = func(o object.Object) bool { _, ok := o.(*object.Function); return ok }
	m.Dict.SetStr("FunctionType", funcType)
	m.Dict.SetStr("LambdaType", funcType)

	builtinFuncType := &object.Class{Name: "builtin_function_or_method", Dict: object.NewDict()}
	builtinFuncType.ABCCheck = func(o object.Object) bool { _, ok := o.(*object.BuiltinFunc); return ok }
	m.Dict.SetStr("BuiltinFunctionType", builtinFuncType)
	m.Dict.SetStr("BuiltinMethodType", builtinFuncType)

	methodType := &object.Class{Name: "method", Dict: object.NewDict()}
	methodType.ABCCheck = func(o object.Object) bool { _, ok := o.(*object.BoundMethod); return ok }
	m.Dict.SetStr("MethodType", methodType)

	generatorType := &object.Class{Name: "generator", Dict: object.NewDict()}
	generatorType.ABCCheck = func(o object.Object) bool { _, ok := o.(*object.Generator); return ok }
	m.Dict.SetStr("GeneratorType", generatorType)

	codeType := &object.Class{Name: "code", Dict: object.NewDict()}
	codeType.ABCCheck = func(o object.Object) bool { _, ok := o.(*object.Code); return ok }
	m.Dict.SetStr("CodeType", codeType)

	frameType := &object.Class{Name: "frame", Dict: object.NewDict()}
	frameType.ABCCheck = func(o object.Object) bool { _, ok := o.(*Frame); return ok }
	m.Dict.SetStr("FrameType", frameType)

	tracebackType := &object.Class{Name: "traceback", Dict: object.NewDict()}
	tracebackType.ABCCheck = func(o object.Object) bool { _, ok := o.(*object.Traceback); return ok }
	m.Dict.SetStr("TracebackType", tracebackType)

	// ModuleType: BuiltinFunc so calling it constructs a *Module.
	// isinstance dispatch is via matchBuiltinType("module") → added in ops.go.
	m.Dict.SetStr("ModuleType", &object.BuiltinFunc{Name: "module", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "ModuleType() requires a name")
		}
		nameStr, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "module name must be str")
		}
		mod := &object.Module{Name: nameStr.V, Dict: object.NewDict()}
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				mod.Dict.SetStr("__doc__", &object.Str{V: s.V})
			}
		}
		return mod, nil
	}})

	// --- SimpleNamespace ---
	m.Dict.SetStr("SimpleNamespace", i.buildSimpleNamespaceClass())

	// --- MappingProxyType ---
	m.Dict.SetStr("MappingProxyType", &object.BuiltinFunc{Name: "MappingProxyType", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "MappingProxyType() takes exactly 1 argument")
		}
		src, ok := a[0].(*object.Dict)
		if !ok {
			return nil, object.Errorf(i.typeErr, "MappingProxyType() argument must be a dict, not '%s'", object.TypeName(a[0]))
		}
		return i.buildMappingProxy(src), nil
	}})

	// --- new_class ---
	m.Dict.SetStr("new_class", &object.BuiltinFunc{Name: "new_class", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "new_class() requires at least a name")
		}
		nameStr, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "new_class() name must be str")
		}
		cls := &object.Class{Name: nameStr.V, Dict: object.NewDict()}

		// bases: 2nd positional arg (tuple)
		if len(a) >= 2 {
			if tup, ok := a[1].(*object.Tuple); ok {
				for _, b := range tup.V {
					if bc, ok := b.(*object.Class); ok {
						cls.Bases = append(cls.Bases, bc)
					}
				}
			}
		}

		// exec_body: 4th positional arg (callable receiving a dict namespace)
		if len(a) >= 4 && a[3] != nil && !isNoneObj(a[3]) {
			ns := object.NewDict()
			if _, err := i.callObject(a[3], []object.Object{ns}, nil); err != nil {
				return nil, err
			}
			keys, vals := ns.Items()
			for k, key := range keys {
				if s, ok := key.(*object.Str); ok {
					cls.Dict.SetStr(s.V, vals[k])
				}
			}
		}
		object.BumpClassEpoch()
		return cls, nil
	}})

	// --- prepare_class ---
	m.Dict.SetStr("prepare_class", &object.BuiltinFunc{Name: "prepare_class", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		meta := &object.Class{Name: "type", Dict: object.NewDict()}
		ns := object.NewDict()
		kwds := object.NewDict()
		return &object.Tuple{V: []object.Object{meta, ns, kwds}}, nil
	}})

	return m
}

// buildSimpleNamespaceClass returns the SimpleNamespace class.
// Instances store their attributes in inst.Dict; repr shows namespace(k=v, ...).
func (i *Interp) buildSimpleNamespaceClass() *object.Class {
	cls := &object.Class{Name: "SimpleNamespace", Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		if kw != nil {
			keys, vals := kw.Items()
			for k, key := range keys {
				if s, ok := key.(*object.Str); ok {
					inst.Dict.SetStr(s.V, vals[k])
				}
			}
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "namespace()"}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "namespace()"}, nil
		}
		keys, vals := inst.Dict.Items()
		parts := make([]string, 0, len(keys))
		for k, key := range keys {
			if s, ok := key.(*object.Str); ok {
				parts = append(parts, s.V+"="+object.Repr(vals[k]))
			}
		}
		return &object.Str{V: "namespace(" + strings.Join(parts, ", ") + ")"}, nil
	}})

	cls.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		ia, aok := a[0].(*object.Instance)
		ib, bok := a[1].(*object.Instance)
		if !aok || !bok || ia.Class != ib.Class {
			return object.False, nil
		}
		ak, av := ia.Dict.Items()
		bk, _ := ib.Dict.Items()
		if len(ak) != len(bk) {
			return object.False, nil
		}
		for k, key := range ak {
			s, ok := key.(*object.Str)
			if !ok {
				continue
			}
			bval, found, err := ib.Dict.Get(&object.Str{V: s.V})
			if err != nil || !found {
				return object.False, nil
			}
			eq, err := object.Eq(av[k], bval)
			if err != nil || !eq {
				return object.False, nil
			}
		}
		return object.True, nil
	}})

	return cls
}

// buildMappingProxy creates a read-only proxy Instance backed by src.
func (i *Interp) buildMappingProxy(src *object.Dict) *object.Instance {
	cls := &object.Class{Name: "mappingproxy", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	inst.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		v, ok, err := src.Get(a[0])
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
		}
		return v, nil
	}})

	inst.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return nil, object.Errorf(i.typeErr, "'mappingproxy' object does not support item assignment")
	}})

	inst.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		_, ok, err := src.Get(a[0])
		if err != nil {
			return nil, err
		}
		return object.BoolOf(ok), nil
	}})

	inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(src.Len())), nil
	}})

	inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		keys, _ := src.Items()
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
		keys, _ := src.Items()
		out := make([]object.Object, len(keys))
		copy(out, keys)
		return &object.List{V: out}, nil
	}})

	inst.Dict.SetStr("values", &object.BuiltinFunc{Name: "values", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		_, vals := src.Items()
		out := make([]object.Object, len(vals))
		copy(out, vals)
		return &object.List{V: out}, nil
	}})

	inst.Dict.SetStr("items", &object.BuiltinFunc{Name: "items", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		keys, vals := src.Items()
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
		v, ok, err := src.Get(a[0])
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

	inst.Dict.SetStr("copy", &object.BuiltinFunc{Name: "copy", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		d := object.NewDict()
		keys, vals := src.Items()
		for k, key := range keys {
			if err := d.Set(key, vals[k]); err != nil {
				return nil, err
			}
		}
		return d, nil
	}})

	return inst
}

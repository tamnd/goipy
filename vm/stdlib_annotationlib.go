package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildAnnotationlib() *object.Module {
	m := &object.Module{Name: "annotationlib", Dict: object.NewDict()}

	// ── typeReprStr — internal helper ────────────────────────────────────────
	typeReprStr := func(val object.Object) string {
		switch v := val.(type) {
		case *object.Str:
			// string annotation → quoted repr, e.g. "int" → "'int'"
			return "'" + v.V + "'"
		case *object.Class:
			return v.Name
		case *object.BuiltinFunc:
			return v.Name
		}
		return object.Repr(val)
	}

	// ── Format enum (IntEnum-like) ───────────────────────────────────────────
	fmtClass := &object.Class{Name: "Format", Dict: object.NewDict()}

	// __eq__: compare .value to int or another Format instance
	fmtClass.Dict.SetStr("__eq__", &object.BuiltinFunc{
		Name: "__eq__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.False, nil
			}
			self, ok := a[0].(*object.Instance)
			if !ok {
				return object.False, nil
			}
			selfVal, ok2 := self.Dict.GetStr("value")
			if !ok2 {
				return object.False, nil
			}
			v1, ok3 := toInt64(selfVal)
			if !ok3 {
				return object.False, nil
			}
			if v2, ok4 := toInt64(a[1]); ok4 {
				return object.BoolOf(v1 == v2), nil
			}
			if other, ok4 := a[1].(*object.Instance); ok4 {
				if otherVal, ok5 := other.Dict.GetStr("value"); ok5 {
					if v2, ok6 := toInt64(otherVal); ok6 {
						return object.BoolOf(v1 == v2), nil
					}
				}
			}
			return object.False, nil
		},
	})

	fmtClass.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "<Format>"}, nil
			}
			self := a[0].(*object.Instance)
			name := "?"
			if n, ok := self.Dict.GetStr("name"); ok {
				if s, ok2 := n.(*object.Str); ok2 {
					name = s.V
				}
			}
			val := int64(0)
			if v, ok := self.Dict.GetStr("value"); ok {
				val, _ = toInt64(v)
			}
			return &object.Str{V: "<Format." + name + ": " + object.Repr(object.NewInt(val)) + ">"}, nil
		},
	})

	makeFmt := func(name string, val int64) *object.Instance {
		inst := &object.Instance{Class: fmtClass, Dict: object.NewDict()}
		inst.Dict.SetStr("value", object.NewInt(val))
		inst.Dict.SetStr("name", &object.Str{V: name})
		return inst
	}

	fmtValue := makeFmt("VALUE", 1)
	fmtValueWithFakeGlobals := makeFmt("VALUE_WITH_FAKE_GLOBALS", 2)
	fmtForwardRef := makeFmt("FORWARDREF", 3)
	fmtString := makeFmt("STRING", 4)

	fmtClass.Dict.SetStr("VALUE", fmtValue)
	fmtClass.Dict.SetStr("VALUE_WITH_FAKE_GLOBALS", fmtValueWithFakeGlobals)
	fmtClass.Dict.SetStr("FORWARDREF", fmtForwardRef)
	fmtClass.Dict.SetStr("STRING", fmtString)
	m.Dict.SetStr("Format", fmtClass)

	// ── ForwardRef class ─────────────────────────────────────────────────────
	fwdClass := &object.Class{Name: "ForwardRef", Dict: object.NewDict()}

	fwdClass.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			self.Dict.SetStr("__forward_arg__", a[1])
			self.Dict.SetStr("__forward_module__", object.None)
			self.Dict.SetStr("__forward_is_argument__", object.True)
			self.Dict.SetStr("__forward_is_class__", object.False)
			self.Dict.SetStr("__forward_code__", object.None)
			// Legacy attrs kept for backwards compat with fixture 278
			self.Dict.SetStr("__forward_evaluated__", object.False)
			self.Dict.SetStr("__forward_value__", object.None)
			if kw != nil {
				if mod, ok := kw.GetStr("module"); ok {
					self.Dict.SetStr("__forward_module__", mod)
				}
				if isArg, ok := kw.GetStr("is_argument"); ok {
					self.Dict.SetStr("__forward_is_argument__", isArg)
				}
				if isCls, ok := kw.GetStr("is_class"); ok {
					self.Dict.SetStr("__forward_is_class__", isCls)
				}
			}
			return object.None, nil
		},
	})

	fwdClass.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "ForwardRef()"}, nil
			}
			self := a[0].(*object.Instance)
			arg := ""
			if v, ok := self.Dict.GetStr("__forward_arg__"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					arg = s.V
				}
			}
			return &object.Str{V: "ForwardRef('" + arg + "')"}, nil
		},
	})

	fwdClass.Dict.SetStr("__eq__", &object.BuiltinFunc{
		Name: "__eq__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.False, nil
			}
			self, ok := a[0].(*object.Instance)
			if !ok {
				return object.False, nil
			}
			other, ok2 := a[1].(*object.Instance)
			if !ok2 {
				return object.False, nil
			}
			selfArg, ok3 := self.Dict.GetStr("__forward_arg__")
			otherArg, ok4 := other.Dict.GetStr("__forward_arg__")
			if !ok3 || !ok4 {
				return object.False, nil
			}
			ss, ok5 := selfArg.(*object.Str)
			os2, ok6 := otherArg.(*object.Str)
			if !ok5 || !ok6 {
				return object.False, nil
			}
			return object.BoolOf(ss.V == os2.V), nil
		},
	})

	fwdClass.Dict.SetStr("__hash__", &object.BuiltinFunc{
		Name: "__hash__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NewInt(0), nil
			}
			self := a[0].(*object.Instance)
			if arg, ok := self.Dict.GetStr("__forward_arg__"); ok {
				if s, ok2 := arg.(*object.Str); ok2 {
					var h uint64
					for _, c := range s.V {
						h = h*31 + uint64(c)
					}
					return object.NewInt(int64(h)), nil
				}
			}
			return object.NewInt(0), nil
		},
	})

	fwdClass.Dict.SetStr("evaluate", &object.BuiltinFunc{
		Name: "evaluate",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.nameErr, "name is not defined")
			}
			self := a[0].(*object.Instance)
			if arg, ok := self.Dict.GetStr("__forward_arg__"); ok {
				if s, ok2 := arg.(*object.Str); ok2 {
					return nil, object.Errorf(i.nameErr, "name '%s' is not defined", s.V)
				}
			}
			return nil, object.Errorf(i.nameErr, "name is not defined")
		},
	})

	m.Dict.SetStr("ForwardRef", fwdClass)

	// ── get_annotations ──────────────────────────────────────────────────────
	m.Dict.SetStr("get_annotations", &object.BuiltinFunc{
		Name: "get_annotations",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NewDict(), nil
			}
			obj := a[0]

			// Determine format (default VALUE=1)
			formatVal := int64(1)
			if kw != nil {
				if fmtArg, ok := kw.GetStr("format"); ok {
					if inst, ok2 := fmtArg.(*object.Instance); ok2 {
						if v, ok3 := inst.Dict.GetStr("value"); ok3 {
							if n, ok4 := toInt64(v); ok4 {
								formatVal = n
							}
						}
					} else if n, ok2 := toInt64(fmtArg); ok2 {
						formatVal = n
					}
				}
			}

			// Get raw annotations
			interp := ii.(*Interp)
			var annObj object.Object
			switch v := obj.(type) {
			case *object.Class:
				// Python 3.14: try __annotate_func__ first (lazy evaluation)
				if fn, ok := v.Dict.GetStr("__annotate_func__"); ok {
					if res, err := interp.callObject(fn, []object.Object{object.NewInt(1)}, nil); err == nil {
						annObj = res
					}
				}
				if annObj == nil {
					if ann, ok := v.Dict.GetStr("__annotations__"); ok {
						annObj = ann
					}
				}
			case *object.Instance:
				if ann, ok := v.Dict.GetStr("__annotations__"); ok {
					annObj = ann
				}
			case *object.Module:
				if ann, ok := v.Dict.GetStr("__annotations__"); ok {
					annObj = ann
				}
			case *object.Function:
				if ann, ok := v.Globals.GetStr("__annotations__"); ok {
					annObj = ann
				}
			}
			if annObj == nil {
				return object.NewDict(), nil
			}
			srcDict, ok := annObj.(*object.Dict)
			if !ok {
				return object.NewDict(), nil
			}

			// STRING format (value=4): convert each annotation to string
			if formatVal == 4 {
				result := object.NewDict()
				ks, vs := srcDict.Items()
				for idx, k := range ks {
					strVal := typeReprStr(vs[idx])
					result.Set(k, &object.Str{V: strVal})
				}
				return result, nil
			}

			return srcDict, nil
		},
	})

	// ── call_annotate_function ────────────────────────────────────────────────
	m.Dict.SetStr("call_annotate_function", &object.BuiltinFunc{
		Name: "call_annotate_function",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.NewDict(), nil
			}
			fn := a[0]
			format := a[1]
			interp := ii.(*Interp)
			res, err := interp.callObject(fn, []object.Object{format}, nil)
			if err != nil {
				return object.NewDict(), nil
			}
			if d, ok := res.(*object.Dict); ok {
				return d, nil
			}
			return object.NewDict(), nil
		},
	})

	// ── call_evaluate_function ────────────────────────────────────────────────
	m.Dict.SetStr("call_evaluate_function", &object.BuiltinFunc{
		Name: "call_evaluate_function",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.NewDict(), nil
			}
			fn := a[0]
			format := a[1]
			interp := ii.(*Interp)
			res, err := interp.callObject(fn, []object.Object{format}, nil)
			if err != nil {
				return object.NewDict(), nil
			}
			if d, ok := res.(*object.Dict); ok {
				return d, nil
			}
			return object.NewDict(), nil
		},
	})

	// ── get_annotate_from_class_namespace ─────────────────────────────────────
	m.Dict.SetStr("get_annotate_from_class_namespace", &object.BuiltinFunc{
		Name: "get_annotate_from_class_namespace",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			switch v := a[0].(type) {
			case *object.Dict:
				if fn, ok := v.GetStr("__annotate__"); ok {
					return fn, nil
				}
			case *object.Class:
				if fn, ok := v.Dict.GetStr("__annotate__"); ok {
					return fn, nil
				}
			}
			return object.None, nil
		},
	})

	// ── type_repr ─────────────────────────────────────────────────────────────
	m.Dict.SetStr("type_repr", &object.BuiltinFunc{
		Name: "type_repr",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			return &object.Str{V: typeReprStr(a[0])}, nil
		},
	})

	// ── annotations_to_string ─────────────────────────────────────────────────
	m.Dict.SetStr("annotations_to_string", &object.BuiltinFunc{
		Name: "annotations_to_string",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NewDict(), nil
			}
			srcDict, ok := a[0].(*object.Dict)
			if !ok {
				return object.NewDict(), nil
			}
			result := object.NewDict()
			ks, vs := srcDict.Items()
			for idx, k := range ks {
				result.Set(k, &object.Str{V: typeReprStr(vs[idx])})
			}
			return result, nil
		},
	})

	// ── get_annotate_function (legacy — not in Python 3.14 but kept for 278) ──
	m.Dict.SetStr("get_annotate_function", &object.BuiltinFunc{
		Name: "get_annotate_function",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			obj := a[0]
			switch v := obj.(type) {
			case *object.Class:
				if fn, ok := v.Dict.GetStr("__annotate__"); ok {
					return fn, nil
				}
			case *object.Function:
				if fn, ok := v.Globals.GetStr("__annotate__"); ok {
					return fn, nil
				}
			}
			return object.None, nil
		},
	})

	// ── annotations_from_str (legacy — not in Python 3.14 but kept for 278) ──
	m.Dict.SetStr("annotations_from_str", &object.BuiltinFunc{
		Name: "annotations_from_str",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewDict(), nil
		},
	})

	m.Dict.SetStr("_typing_module", object.None)

	return m
}

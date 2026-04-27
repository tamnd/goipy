package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildAnnotationlib() *object.Module {
	m := &object.Module{Name: "annotationlib", Dict: object.NewDict()}

	// Format enum values
	m.Dict.SetStr("Format", &object.Class{Name: "Format", Dict: func() *object.Dict {
		d := object.NewDict()
		d.SetStr("VALUE", object.NewInt(1))
		d.SetStr("FORWARDREF", object.NewInt(2))
		d.SetStr("STRING", object.NewInt(3))
		return d
	}()})

	// ForwardRef class — stands in for unresolvable annotations
	fwdClass := &object.Class{Name: "ForwardRef", Dict: object.NewDict()}
	fwdClass.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			self.Dict.SetStr("__forward_arg__", a[1])
			self.Dict.SetStr("__forward_evaluated__", object.False)
			self.Dict.SetStr("__forward_value__", object.None)
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
	m.Dict.SetStr("ForwardRef", fwdClass)

	m.Dict.SetStr("get_annotations", &object.BuiltinFunc{
		Name: "get_annotations",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NewDict(), nil
			}
			obj := a[0]
			// Look for __annotations__ or __annotate__
			var annObj object.Object
			switch v := obj.(type) {
			case *object.Class:
				if ann, ok := v.Dict.GetStr("__annotations__"); ok {
					annObj = ann
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
			if d, ok := annObj.(*object.Dict); ok {
				return d, nil
			}
			return object.NewDict(), nil
		},
	})

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

	m.Dict.SetStr("annotations_from_str", &object.BuiltinFunc{
		Name: "annotations_from_str",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewDict(), nil
		},
	})

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

	m.Dict.SetStr("type_repr", &object.BuiltinFunc{
		Name: "type_repr",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			switch v := a[0].(type) {
			case *object.Str:
				return v, nil
			case *object.Class:
				return &object.Str{V: v.Name}, nil
			case *object.BuiltinFunc:
				return &object.Str{V: v.Name}, nil
			}
			return &object.Str{V: object.Repr(a[0])}, nil
		},
	})

	m.Dict.SetStr("_typing_module", object.None)

	return m
}

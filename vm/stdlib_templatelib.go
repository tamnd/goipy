package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildTemplatelib returns the string.templatelib module (PEP 750).
func (i *Interp) buildTemplatelib() *object.Module {
	m := &object.Module{Name: "string.templatelib", Dict: object.NewDict()}

	templateCls := i.makeTemplatelibTemplateClass()
	interpCls := i.makeTemplatelibInterpolationClass()

	m.Dict.SetStr("Template", templateCls)
	m.Dict.SetStr("Interpolation", interpCls)
	m.Dict.SetStr("convert", &object.BuiltinFunc{Name: "convert", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "convert() requires 2 arguments")
		}
		obj := a[0]
		conv := a[1]
		if conv == object.None {
			return obj, nil
		}
		cs, ok := conv.(*object.Str)
		if !ok {
			return obj, nil
		}
		switch cs.V {
		case "s":
			return &object.Str{V: object.Str_(obj)}, nil
		case "r":
			return &object.Str{V: object.Repr(obj)}, nil
		case "a":
			return &object.Str{V: asciiRepr(obj)}, nil
		}
		return obj, nil
	}})

	return m
}

// makeTemplatelibTemplateClass returns a callable that constructs Template objects
// from *args of str | Interpolation, following PEP 750 constructor semantics:
// consecutive strings are merged; consecutive interpolations get empty string gaps.
func (i *Interp) makeTemplatelibTemplateClass() *object.BuiltinFunc {
	return &object.BuiltinFunc{Name: "Template", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		var strs []*object.Str
		var interps []*object.Interpolation
		pendingStr := ""
		hasPending := false

		flushStr := func() {
			strs = append(strs, &object.Str{V: pendingStr})
			pendingStr = ""
			hasPending = false
		}

		for _, arg := range args {
			switch v := arg.(type) {
			case *object.Str:
				pendingStr += v.V
				hasPending = true
			case *object.Interpolation:
				if !hasPending {
					// no preceding string since last interp → insert empty
					strs = append(strs, &object.Str{V: ""})
				} else {
					flushStr()
				}
				interps = append(interps, v)
				hasPending = false
			}
		}
		// final trailing string: strings always has one more element than interpolations
		if hasPending {
			strs = append(strs, &object.Str{V: pendingStr})
		} else if len(strs) <= len(interps) {
			strs = append(strs, &object.Str{V: ""})
		}
		if len(strs) == 0 {
			strs = append(strs, &object.Str{V: ""})
		}

		return &object.Template{Strings: strs, Interpolations: interps}, nil
	}}
}

// makeTemplatelibInterpolationClass returns a callable that constructs Interpolation.
func (i *Interp) makeTemplatelibInterpolationClass() *object.BuiltinFunc {
	return &object.BuiltinFunc{Name: "Interpolation", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "Interpolation() requires at least 2 arguments (value, expression)")
		}
		value := args[0]
		expr := ""
		if s, ok := args[1].(*object.Str); ok {
			expr = s.V
		}
		conv := ""
		if len(args) >= 3 && args[2] != object.None {
			if s, ok := args[2].(*object.Str); ok {
				conv = s.V
			}
		}
		spec := ""
		if len(args) >= 4 {
			if s, ok := args[3].(*object.Str); ok {
				spec = s.V
			}
		}
		return &object.Interpolation{
			Value:      value,
			Expression: expr,
			Conversion: conv,
			FormatSpec: spec,
		}, nil
	}}
}

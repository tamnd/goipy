package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildNumbers constructs the `numbers` module: Number, Complex, Real, Rational,
// Integral — the PEP 3141 numeric tower ABCs.
func (i *Interp) buildNumbers() *object.Module {
	m := &object.Module{Name: "numbers", Dict: object.NewDict()}

	// makeABC creates one ABC class with:
	//   • Bases set for proper subclass walk (issubclass / inheritance)
	//   • ABCCheck for built-in Go types (enables isinstance for int/float/complex)
	//   • register(cls) for virtual-subclass registration
	//   • __instancecheck__ / __subclasscheck__ for explicit dispatch
	makeABC := func(name string, base *object.Class, check func(object.Object) bool) *object.Class {
		registered := []*object.Class{}

		var bases []*object.Class
		if base != nil {
			bases = []*object.Class{base}
		}
		cls := &object.Class{Name: name, Bases: bases, Dict: object.NewDict()}

		cls.ABCCheck = func(o object.Object) bool {
			// Virtual subclass check: instances of registered classes pass.
			if inst, ok := o.(*object.Instance); ok {
				for _, r := range registered {
					if object.IsSubclass(inst.Class, r) {
						return true
					}
				}
			}
			return check(o)
		}

		cls.Dict.SetStr("register", &object.BuiltinFunc{Name: "register", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "register() takes 1 argument")
			}
			if c, ok := a[0].(*object.Class); ok {
				registered = append(registered, c)
			}
			return a[0], nil
		}})

		cls.Dict.SetStr("__instancecheck__", &object.BuiltinFunc{Name: "__instancecheck__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			return object.BoolOf(cls.ABCCheck(a[0])), nil
		}})

		cls.Dict.SetStr("__subclasscheck__", &object.BuiltinFunc{Name: "__subclasscheck__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			c, ok := a[0].(*object.Class)
			if !ok {
				return object.False, nil
			}
			// Direct or inherited subclass.
			if object.IsSubclass(c, cls) {
				return object.True, nil
			}
			// Virtual subclass via register().
			for _, r := range registered {
				if object.IsSubclass(c, r) {
					return object.True, nil
				}
			}
			return object.False, nil
		}})

		m.Dict.SetStr(name, cls)
		return cls
	}

	// Number — root; matches any built-in numeric type.
	numberCls := makeABC("Number", nil, func(o object.Object) bool {
		switch o.(type) {
		case *object.Int, *object.Bool, *object.Float, *object.Complex:
			return true
		}
		return false
	})

	// Complex — complex, float, int, bool.
	complexCls := makeABC("Complex", numberCls, func(o object.Object) bool {
		switch o.(type) {
		case *object.Int, *object.Bool, *object.Float, *object.Complex:
			return true
		}
		return false
	})

	// Real — float, int, bool (pure complex excluded).
	realCls := makeABC("Real", complexCls, func(o object.Object) bool {
		switch o.(type) {
		case *object.Int, *object.Bool, *object.Float:
			return true
		}
		return false
	})

	// Rational — int, bool (fractions.Fraction is registered separately).
	rationalCls := makeABC("Rational", realCls, func(o object.Object) bool {
		switch o.(type) {
		case *object.Int, *object.Bool:
			return true
		}
		return false
	})

	// Integral — int, bool.
	makeABC("Integral", rationalCls, func(o object.Object) bool {
		switch o.(type) {
		case *object.Int, *object.Bool:
			return true
		}
		return false
	})

	return m
}

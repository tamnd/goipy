package vm

import (
	"math"
	"math/cmplx"

	"github.com/tamnd/goipy/object"
)

// toComplex coerces any Python numeric value to a Go complex128.
// Accepts int, bool, float, complex.
func toComplex(o object.Object) (complex128, bool) {
	switch v := o.(type) {
	case *object.Complex:
		return complex(v.Real, v.Imag), true
	case *object.Float:
		return complex(v.V, 0), true
	case *object.Int:
		f, _ := v.V.Float64()
		return complex(f, 0), true
	case *object.Bool:
		if v.V {
			return complex(1, 0), true
		}
		return complex(0, 0), true
	}
	return 0, false
}

// pyComplex wraps a Go complex128 as *object.Complex.
func pyComplex(z complex128) *object.Complex {
	return &object.Complex{Real: real(z), Imag: imag(z)}
}

func (i *Interp) buildCmath() *object.Module {
	m := &object.Module{Name: "cmath", Dict: object.NewDict()}

	// ---- Constants ----
	m.Dict.SetStr("pi", &object.Float{V: math.Pi})
	m.Dict.SetStr("e", &object.Float{V: math.E})
	m.Dict.SetStr("tau", &object.Float{V: 2 * math.Pi})
	m.Dict.SetStr("inf", &object.Float{V: math.Inf(1)})
	m.Dict.SetStr("nan", &object.Float{V: math.NaN()})
	m.Dict.SetStr("infj", &object.Complex{Real: 0, Imag: math.Inf(1)})
	m.Dict.SetStr("nanj", &object.Complex{Real: 0, Imag: math.NaN()})

	// Helper: one-arg complex → complex function.
	c1 := func(name string, fn func(complex128) complex128) {
		m.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "%s() takes exactly one argument", name)
			}
			z, ok := toComplex(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "%s() requires a number", name)
			}
			return pyComplex(fn(z)), nil
		}})
	}

	// ---- Power and logarithmic ----
	c1("exp", cmplx.Exp)
	c1("sqrt", cmplx.Sqrt)
	c1("log10", cmplx.Log10)

	// log(z[, base])
	m.Dict.SetStr("log", &object.BuiltinFunc{Name: "log", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 || len(a) > 2 {
			return nil, object.Errorf(i.typeErr, "log() takes 1 or 2 arguments")
		}
		z, ok := toComplex(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "log() requires a number")
		}
		if len(a) == 2 {
			base, ok := toComplex(a[1])
			if !ok {
				return nil, object.Errorf(i.typeErr, "log() base must be a number")
			}
			return pyComplex(cmplx.Log(z) / cmplx.Log(base)), nil
		}
		return pyComplex(cmplx.Log(z)), nil
	}})

	// ---- Trigonometric ----
	c1("sin", cmplx.Sin)
	c1("cos", cmplx.Cos)
	c1("tan", cmplx.Tan)
	c1("asin", cmplx.Asin)
	c1("acos", cmplx.Acos)
	c1("atan", cmplx.Atan)

	// ---- Hyperbolic ----
	c1("sinh", cmplx.Sinh)
	c1("cosh", cmplx.Cosh)
	c1("tanh", cmplx.Tanh)
	c1("asinh", cmplx.Asinh)
	c1("acosh", cmplx.Acosh)
	c1("atanh", cmplx.Atanh)

	// ---- Polar conversion ----

	// phase(z) -> float  — atan2(z.imag, z.real)
	m.Dict.SetStr("phase", &object.BuiltinFunc{Name: "phase", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "phase() takes exactly one argument")
		}
		z, ok := toComplex(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "phase() requires a number")
		}
		return &object.Float{V: cmplx.Phase(z)}, nil
	}})

	// polar(z) -> (r, phi)
	m.Dict.SetStr("polar", &object.BuiltinFunc{Name: "polar", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "polar() takes exactly one argument")
		}
		z, ok := toComplex(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "polar() requires a number")
		}
		r, phi := cmplx.Polar(z)
		return &object.Tuple{V: []object.Object{
			&object.Float{V: r},
			&object.Float{V: phi},
		}}, nil
	}})

	// rect(r, phi) -> complex
	m.Dict.SetStr("rect", &object.BuiltinFunc{Name: "rect", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "rect() takes exactly two arguments")
		}
		r, okR := toFloat64Any(a[0])
		phi, okP := toFloat64Any(a[1])
		if !okR || !okP {
			return nil, object.Errorf(i.typeErr, "rect() requires real numbers")
		}
		return pyComplex(cmplx.Rect(r, phi)), nil
	}})

	// ---- Classification ----

	// isfinite(z) — both real and imaginary parts are finite.
	m.Dict.SetStr("isfinite", &object.BuiltinFunc{Name: "isfinite", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "isfinite() takes exactly one argument")
		}
		z, ok := toComplex(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "isfinite() requires a number")
		}
		return object.BoolOf(!math.IsInf(real(z), 0) && !math.IsNaN(real(z)) &&
			!math.IsInf(imag(z), 0) && !math.IsNaN(imag(z))), nil
	}})

	// isinf(z) — either part is infinite.
	m.Dict.SetStr("isinf", &object.BuiltinFunc{Name: "isinf", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "isinf() takes exactly one argument")
		}
		z, ok := toComplex(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "isinf() requires a number")
		}
		return object.BoolOf(math.IsInf(real(z), 0) || math.IsInf(imag(z), 0)), nil
	}})

	// isnan(z) — either part is NaN.
	m.Dict.SetStr("isnan", &object.BuiltinFunc{Name: "isnan", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "isnan() takes exactly one argument")
		}
		z, ok := toComplex(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "isnan() requires a number")
		}
		return object.BoolOf(math.IsNaN(real(z)) || math.IsNaN(imag(z))), nil
	}})

	// isclose(a, b, *, rel_tol=1e-09, abs_tol=0.0)
	m.Dict.SetStr("isclose", &object.BuiltinFunc{Name: "isclose", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "isclose() takes at least 2 arguments")
		}
		za, okA := toComplex(a[0])
		zb, okB := toComplex(a[1])
		if !okA || !okB {
			return nil, object.Errorf(i.typeErr, "isclose() requires numbers")
		}
		relTol := 1e-9
		absTol := 0.0
		if kw != nil {
			if v, ok := kw.GetStr("rel_tol"); ok {
				relTol, _ = toFloat64Any(v)
			}
			if v, ok := kw.GetStr("abs_tol"); ok {
				absTol, _ = toFloat64Any(v)
			}
		}
		// NaN is never close to anything.
		if cmplx.IsNaN(za) || cmplx.IsNaN(zb) {
			return object.False, nil
		}
		// inf is only close to itself.
		if cmplx.IsInf(za) || cmplx.IsInf(zb) {
			return object.BoolOf(za == zb), nil
		}
		diff := cmplx.Abs(za - zb)
		threshold := math.Max(relTol*math.Max(cmplx.Abs(za), cmplx.Abs(zb)), absTol)
		return object.BoolOf(diff <= threshold), nil
	}})

	return m
}

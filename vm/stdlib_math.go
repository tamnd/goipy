package vm

import (
	"math"
	"math/big"
	gorand "math/rand"

	"github.com/tamnd/goipy/object"
)

// toFloat64Any coerces any numeric Python value to a Go float64. ok=false
// when the value is not numeric.
func toFloat64Any(o object.Object) (float64, bool) {
	switch v := o.(type) {
	case *object.Bool:
		if v.V {
			return 1, true
		}
		return 0, true
	case *object.Int:
		f, _ := new(big.Float).SetInt(&v.V).Float64()
		return f, true
	case *object.Float:
		return v.V, true
	}
	return 0, false
}

// --- math ---

// buildMath exposes the math module. Coverage is deliberately broad
// because many scripts reach for math.sqrt / math.pi / math.floor before
// anything else — a minimal surface forces users to hand-code replacements.
func (i *Interp) buildMath() *object.Module {
	m := &object.Module{Name: "math", Dict: object.NewDict()}

	// Constants.
	m.Dict.SetStr("pi", &object.Float{V: math.Pi})
	m.Dict.SetStr("e", &object.Float{V: math.E})
	m.Dict.SetStr("tau", &object.Float{V: 2 * math.Pi})
	m.Dict.SetStr("inf", &object.Float{V: math.Inf(1)})
	m.Dict.SetStr("nan", &object.Float{V: math.NaN()})

	// Helper to build one-argument float functions.
	f1 := func(name string, fn func(float64) float64) {
		m.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "%s() takes 1 argument", name)
			}
			x, ok := toFloat64Any(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "%s() requires a number", name)
			}
			return &object.Float{V: fn(x)}, nil
		}})
	}

	f1("sqrt", math.Sqrt)
	f1("exp", math.Exp)
	f1("expm1", math.Expm1)
	f1("log2", math.Log2)
	f1("log10", math.Log10)
	f1("log1p", math.Log1p)
	f1("sin", math.Sin)
	f1("cos", math.Cos)
	f1("tan", math.Tan)
	f1("asin", math.Asin)
	f1("acos", math.Acos)
	f1("atan", math.Atan)
	f1("sinh", math.Sinh)
	f1("cosh", math.Cosh)
	f1("tanh", math.Tanh)
	f1("asinh", math.Asinh)
	f1("acosh", math.Acosh)
	f1("atanh", math.Atanh)
	f1("degrees", func(x float64) float64 { return x * 180 / math.Pi })
	f1("radians", func(x float64) float64 { return x * math.Pi / 180 })
	f1("fabs", math.Abs)
	f1("erf", math.Erf)
	f1("erfc", math.Erfc)
	f1("gamma", math.Gamma)
	f1("lgamma", func(x float64) float64 { v, _ := math.Lgamma(x); return v })

	// log(x[, base]).
	m.Dict.SetStr("log", &object.BuiltinFunc{Name: "log", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 || len(a) > 2 {
			return nil, object.Errorf(i.typeErr, "log() takes 1 or 2 arguments")
		}
		x, ok := toFloat64Any(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "log() requires a number")
		}
		if len(a) == 2 {
			base, ok := toFloat64Any(a[1])
			if !ok {
				return nil, object.Errorf(i.typeErr, "log() base must be a number")
			}
			return &object.Float{V: math.Log(x) / math.Log(base)}, nil
		}
		return &object.Float{V: math.Log(x)}, nil
	}})

	// atan2(y, x).
	m.Dict.SetStr("atan2", &object.BuiltinFunc{Name: "atan2", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "atan2() takes 2 arguments")
		}
		y, ok := toFloat64Any(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "atan2() requires numbers")
		}
		x, ok := toFloat64Any(a[1])
		if !ok {
			return nil, object.Errorf(i.typeErr, "atan2() requires numbers")
		}
		return &object.Float{V: math.Atan2(y, x)}, nil
	}})

	// copysign(x, y).
	m.Dict.SetStr("copysign", &object.BuiltinFunc{Name: "copysign", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		x, okX := toFloat64Any(a[0])
		y, okY := toFloat64Any(a[1])
		if !okX || !okY {
			return nil, object.Errorf(i.typeErr, "copysign() requires numbers")
		}
		return &object.Float{V: math.Copysign(x, y)}, nil
	}})

	// fmod(x, y).
	m.Dict.SetStr("fmod", &object.BuiltinFunc{Name: "fmod", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		x, okX := toFloat64Any(a[0])
		y, okY := toFloat64Any(a[1])
		if !okX || !okY {
			return nil, object.Errorf(i.typeErr, "fmod() requires numbers")
		}
		return &object.Float{V: math.Mod(x, y)}, nil
	}})

	// hypot(*coords).
	m.Dict.SetStr("hypot", &object.BuiltinFunc{Name: "hypot", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var s float64
		for _, x := range a {
			f, ok := toFloat64Any(x)
			if !ok {
				return nil, object.Errorf(i.typeErr, "hypot() requires numbers")
			}
			s += f * f
		}
		return &object.Float{V: math.Sqrt(s)}, nil
	}})

	// dist(p, q).
	m.Dict.SetStr("dist", &object.BuiltinFunc{Name: "dist", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "dist() takes 2 arguments")
		}
		p, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		q, err := iterate(i, a[1])
		if err != nil {
			return nil, err
		}
		if len(p) != len(q) {
			return nil, object.Errorf(i.valueErr, "both points must have the same number of dimensions")
		}
		var s float64
		for k := range p {
			pf, _ := toFloat64Any(p[k])
			qf, _ := toFloat64Any(q[k])
			d := pf - qf
			s += d * d
		}
		return &object.Float{V: math.Sqrt(s)}, nil
	}})

	// pow(x, y) — math.pow always returns float, unlike builtin pow.
	m.Dict.SetStr("pow", &object.BuiltinFunc{Name: "pow", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		x, okX := toFloat64Any(a[0])
		y, okY := toFloat64Any(a[1])
		if !okX || !okY {
			return nil, object.Errorf(i.typeErr, "pow() requires numbers")
		}
		return &object.Float{V: math.Pow(x, y)}, nil
	}})

	// ceil / floor / trunc — return int.
	// If the argument has __ceil__/__floor__/__trunc__, delegate to it (e.g. Fraction).
	m.Dict.SetStr("ceil", &object.BuiltinFunc{Name: "ceil", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "ceil() requires a number")
		}
		if n, ok := toInt64(a[0]); ok {
			return object.NewInt(n), nil
		}
		if inst, ok := a[0].(*object.Instance); ok {
			if fn, ok2 := classLookup(inst.Class, "__ceil__"); ok2 {
				return i.callObject(fn, []object.Object{inst}, nil)
			}
		}
		x, ok := toFloat64Any(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "ceil() requires a number")
		}
		return object.NewInt(int64(math.Ceil(x))), nil
	}})
	m.Dict.SetStr("floor", &object.BuiltinFunc{Name: "floor", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "floor() requires a number")
		}
		if n, ok := toInt64(a[0]); ok {
			return object.NewInt(n), nil
		}
		if inst, ok := a[0].(*object.Instance); ok {
			if fn, ok2 := classLookup(inst.Class, "__floor__"); ok2 {
				return i.callObject(fn, []object.Object{inst}, nil)
			}
		}
		x, ok := toFloat64Any(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "floor() requires a number")
		}
		return object.NewInt(int64(math.Floor(x))), nil
	}})
	m.Dict.SetStr("trunc", &object.BuiltinFunc{Name: "trunc", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "trunc() requires a number")
		}
		if n, ok := toInt64(a[0]); ok {
			return object.NewInt(n), nil
		}
		if inst, ok := a[0].(*object.Instance); ok {
			if fn, ok2 := classLookup(inst.Class, "__trunc__"); ok2 {
				return i.callObject(fn, []object.Object{inst}, nil)
			}
		}
		x, ok := toFloat64Any(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "trunc() requires a number")
		}
		return object.NewInt(int64(math.Trunc(x))), nil
	}})

	// isnan / isinf / isfinite.
	m.Dict.SetStr("isnan", &object.BuiltinFunc{Name: "isnan", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		x, ok := toFloat64Any(a[0])
		if !ok {
			return object.BoolOf(false), nil
		}
		return object.BoolOf(math.IsNaN(x)), nil
	}})
	m.Dict.SetStr("isinf", &object.BuiltinFunc{Name: "isinf", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		x, ok := toFloat64Any(a[0])
		if !ok {
			return object.BoolOf(false), nil
		}
		return object.BoolOf(math.IsInf(x, 0)), nil
	}})
	m.Dict.SetStr("isfinite", &object.BuiltinFunc{Name: "isfinite", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		x, ok := toFloat64Any(a[0])
		if !ok {
			return object.BoolOf(false), nil
		}
		return object.BoolOf(!math.IsNaN(x) && !math.IsInf(x, 0)), nil
	}})

	// isclose(a, b, *, rel_tol=1e-09, abs_tol=0.0).
	m.Dict.SetStr("isclose", &object.BuiltinFunc{Name: "isclose", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "isclose() takes 2 arguments")
		}
		x, _ := toFloat64Any(a[0])
		y, _ := toFloat64Any(a[1])
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
		diff := math.Abs(x - y)
		if diff == 0 {
			return object.BoolOf(true), nil
		}
		return object.BoolOf(diff <= math.Max(relTol*math.Max(math.Abs(x), math.Abs(y)), absTol)), nil
	}})

	// gcd(*ints).
	m.Dict.SetStr("gcd", &object.BuiltinFunc{Name: "gcd", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return object.NewInt(0), nil
		}
		acc, ok := toBigInt(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "gcd() requires ints")
		}
		acc = new(big.Int).Abs(acc)
		for _, x := range a[1:] {
			n, ok := toBigInt(x)
			if !ok {
				return nil, object.Errorf(i.typeErr, "gcd() requires ints")
			}
			acc = new(big.Int).GCD(nil, nil, acc, new(big.Int).Abs(n))
		}
		return object.IntFromBig(acc), nil
	}})

	// lcm(*ints).
	m.Dict.SetStr("lcm", &object.BuiltinFunc{Name: "lcm", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return object.NewInt(1), nil
		}
		acc, ok := toBigInt(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "lcm() requires ints")
		}
		acc = new(big.Int).Abs(acc)
		for _, x := range a[1:] {
			n, ok := toBigInt(x)
			if !ok {
				return nil, object.Errorf(i.typeErr, "lcm() requires ints")
			}
			n = new(big.Int).Abs(n)
			if acc.Sign() == 0 || n.Sign() == 0 {
				acc = big.NewInt(0)
				continue
			}
			g := new(big.Int).GCD(nil, nil, acc, n)
			acc = new(big.Int).Mul(new(big.Int).Quo(acc, g), n)
		}
		return object.IntFromBig(acc), nil
	}})

	// factorial(n).
	m.Dict.SetStr("factorial", &object.BuiltinFunc{Name: "factorial", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, ok := toInt64(a[0])
		if !ok || n < 0 {
			return nil, object.Errorf(i.valueErr, "factorial() not defined for negative or non-integer")
		}
		r := big.NewInt(1)
		for k := int64(2); k <= n; k++ {
			r.Mul(r, big.NewInt(k))
		}
		return object.IntFromBig(r), nil
	}})

	// comb(n, k) — "n choose k".
	m.Dict.SetStr("comb", &object.BuiltinFunc{Name: "comb", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, ok1 := toInt64(a[0])
		k, ok2 := toInt64(a[1])
		if !ok1 || !ok2 || n < 0 || k < 0 {
			return nil, object.Errorf(i.valueErr, "comb() requires non-negative ints")
		}
		if k > n {
			return object.NewInt(0), nil
		}
		if k > n-k {
			k = n - k
		}
		r := big.NewInt(1)
		for j := int64(0); j < k; j++ {
			r.Mul(r, big.NewInt(n-j))
			r.Quo(r, big.NewInt(j+1))
		}
		return object.IntFromBig(r), nil
	}})

	// perm(n[, k]).
	m.Dict.SetStr("perm", &object.BuiltinFunc{Name: "perm", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, ok1 := toInt64(a[0])
		if !ok1 || n < 0 {
			return nil, object.Errorf(i.valueErr, "perm() requires non-negative ints")
		}
		k := n
		if len(a) >= 2 {
			kv, ok2 := toInt64(a[1])
			if !ok2 || kv < 0 {
				return nil, object.Errorf(i.valueErr, "perm() requires non-negative ints")
			}
			k = kv
		}
		if k > n {
			return object.NewInt(0), nil
		}
		r := big.NewInt(1)
		for j := int64(0); j < k; j++ {
			r.Mul(r, big.NewInt(n-j))
		}
		return object.IntFromBig(r), nil
	}})

	// prod(iterable[, start=1]).
	m.Dict.SetStr("prod", &object.BuiltinFunc{Name: "prod", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "prod() missing iterable")
		}
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		var start object.Object = object.NewInt(1)
		if kw != nil {
			if v, ok := kw.GetStr("start"); ok {
				start = v
			}
		}
		acc := start
		for _, x := range items {
			acc, err = i.mul(acc, x)
			if err != nil {
				return nil, err
			}
		}
		return acc, nil
	}})

	// fsum(iterable) — Kahan-Neumaier compensated summation, so cases like
	// sum([0.1]*10) round exactly to 1.0 instead of drifting through float
	// rounding error.
	m.Dict.SetStr("fsum", &object.BuiltinFunc{Name: "fsum", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		var s, c float64
		for _, x := range items {
			f, ok := toFloat64Any(x)
			if !ok {
				return nil, object.Errorf(i.typeErr, "fsum() requires numbers")
			}
			t := s + f
			if math.Abs(s) >= math.Abs(f) {
				c += (s - t) + f
			} else {
				c += (f - t) + s
			}
			s = t
		}
		return &object.Float{V: s + c}, nil
	}})

	// modf(x) -> (frac, int).
	m.Dict.SetStr("modf", &object.BuiltinFunc{Name: "modf", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		x, ok := toFloat64Any(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "modf() requires a number")
		}
		intPart, frac := math.Modf(x)
		return &object.Tuple{V: []object.Object{&object.Float{V: frac}, &object.Float{V: intPart}}}, nil
	}})

	// frexp(x) -> (m, e).
	m.Dict.SetStr("frexp", &object.BuiltinFunc{Name: "frexp", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		x, ok := toFloat64Any(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "frexp() requires a number")
		}
		mant, exp := math.Frexp(x)
		return &object.Tuple{V: []object.Object{&object.Float{V: mant}, object.NewInt(int64(exp))}}, nil
	}})

	// ldexp(x, i) -> x * 2**i.
	m.Dict.SetStr("ldexp", &object.BuiltinFunc{Name: "ldexp", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		x, ok := toFloat64Any(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "ldexp() requires a number")
		}
		n, ok := toInt64(a[1])
		if !ok {
			return nil, object.Errorf(i.typeErr, "ldexp() exponent must be int")
		}
		return &object.Float{V: math.Ldexp(x, int(n))}, nil
	}})

	// isqrt(n) — integer square root (floor of exact square root).
	m.Dict.SetStr("isqrt", &object.BuiltinFunc{Name: "isqrt", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "isqrt() takes 1 argument")
		}
		n, ok := toBigInt(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "isqrt() requires an integer")
		}
		if n.Sign() < 0 {
			return nil, object.Errorf(i.valueErr, "isqrt() argument must be nonnegative")
		}
		r := new(big.Int).Sqrt(n)
		return object.IntFromBig(r), nil
	}})

	// cbrt(x) — cube root (Python 3.11+).
	f1("cbrt", math.Cbrt)

	// exp2(x) — 2**x (Python 3.11+).
	f1("exp2", math.Exp2)

	// remainder(x, y) — IEEE 754 remainder.
	m.Dict.SetStr("remainder", &object.BuiltinFunc{Name: "remainder", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "remainder() takes 2 arguments")
		}
		x, okX := toFloat64Any(a[0])
		y, okY := toFloat64Any(a[1])
		if !okX || !okY {
			return nil, object.Errorf(i.typeErr, "remainder() requires numbers")
		}
		if y == 0 {
			return nil, object.Errorf(i.valueErr, "math domain error")
		}
		return &object.Float{V: math.Remainder(x, y)}, nil
	}})

	// nextafter(x, y, *, steps=1) — next float value after x toward y.
	m.Dict.SetStr("nextafter", &object.BuiltinFunc{Name: "nextafter", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "nextafter() takes at least 2 arguments")
		}
		x, okX := toFloat64Any(a[0])
		y, okY := toFloat64Any(a[1])
		if !okX || !okY {
			return nil, object.Errorf(i.typeErr, "nextafter() requires numbers")
		}
		steps := int64(1)
		if kw != nil {
			if v, ok := kw.GetStr("steps"); ok {
				if n, ok := toInt64(v); ok {
					steps = n
				}
			}
		}
		r := x
		for k := int64(0); k < steps; k++ {
			r = math.Nextafter(r, y)
		}
		return &object.Float{V: r}, nil
	}})

	// ulp(x) — unit in the last place (Python 3.9+).
	m.Dict.SetStr("ulp", &object.BuiltinFunc{Name: "ulp", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "ulp() takes 1 argument")
		}
		x, ok := toFloat64Any(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "ulp() requires a number")
		}
		if math.IsNaN(x) {
			return &object.Float{V: math.NaN()}, nil
		}
		if math.IsInf(x, 0) {
			return &object.Float{V: math.Inf(1)}, nil
		}
		x = math.Abs(x)
		return &object.Float{V: math.Nextafter(x, math.Inf(1)) - x}, nil
	}})

	// sumprod(p, q) — sum of element-wise products (Python 3.12+).
	m.Dict.SetStr("sumprod", &object.BuiltinFunc{Name: "sumprod", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "sumprod() takes 2 arguments")
		}
		ps, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		qs, err := iterate(i, a[1])
		if err != nil {
			return nil, err
		}
		if len(ps) != len(qs) {
			return nil, object.Errorf(i.valueErr, "sumprod() arguments must have the same length")
		}
		// Use integer accumulation if all values are int, else float.
		allInt := true
		for _, v := range append(ps, qs...) {
			switch v.(type) {
			case *object.Int, *object.Bool:
			default:
				allInt = false
			}
		}
		if allInt {
			acc := new(big.Int)
			for k := range ps {
				pn, _ := toBigInt(ps[k])
				qn, _ := toBigInt(qs[k])
				prod := new(big.Int).Mul(pn, qn)
				acc.Add(acc, prod)
			}
			return object.IntFromBig(acc), nil
		}
		var sum float64
		for k := range ps {
			pf, _ := toFloat64Any(ps[k])
			qf, _ := toFloat64Any(qs[k])
			sum += pf * qf
		}
		return &object.Float{V: sum}, nil
	}})

	// fma(x, y, z) — fused multiply-add: x*y + z without intermediate rounding (Python 3.13+).
	m.Dict.SetStr("fma", &object.BuiltinFunc{Name: "fma", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 3 {
			return nil, object.Errorf(i.typeErr, "fma() takes 3 arguments")
		}
		x, okX := toFloat64Any(a[0])
		y, okY := toFloat64Any(a[1])
		z, okZ := toFloat64Any(a[2])
		if !okX || !okY || !okZ {
			return nil, object.Errorf(i.typeErr, "fma() requires numbers")
		}
		return &object.Float{V: math.FMA(x, y, z)}, nil
	}})

	return m
}

// --- heapq (min-heap) ---

func (i *Interp) buildHeapq() *object.Module {
	m := &object.Module{Name: "heapq", Dict: object.NewDict()}

	m.Dict.SetStr("heappush", &object.BuiltinFunc{Name: "heappush", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "heappush() takes 2 arguments")
		}
		l, ok := a[0].(*object.List)
		if !ok {
			return nil, object.Errorf(i.typeErr, "heap argument must be a list")
		}
		l.V = append(l.V, a[1])
		if err := heapSiftDown(i, l.V, 0, len(l.V)-1); err != nil {
			return nil, err
		}
		return object.None, nil
	}})

	m.Dict.SetStr("heappop", &object.BuiltinFunc{Name: "heappop", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		l, ok := a[0].(*object.List)
		if !ok {
			return nil, object.Errorf(i.typeErr, "heap argument must be a list")
		}
		if len(l.V) == 0 {
			return nil, object.Errorf(i.indexErr, "index out of range")
		}
		last := l.V[len(l.V)-1]
		l.V = l.V[:len(l.V)-1]
		if len(l.V) == 0 {
			return last, nil
		}
		top := l.V[0]
		l.V[0] = last
		if err := heapSiftUp(i, l.V, 0); err != nil {
			return nil, err
		}
		return top, nil
	}})

	m.Dict.SetStr("heapify", &object.BuiltinFunc{Name: "heapify", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		l, ok := a[0].(*object.List)
		if !ok {
			return nil, object.Errorf(i.typeErr, "heap argument must be a list")
		}
		n := len(l.V)
		for k := n/2 - 1; k >= 0; k-- {
			if err := heapSiftUp(i, l.V, k); err != nil {
				return nil, err
			}
		}
		return object.None, nil
	}})

	m.Dict.SetStr("heappushpop", &object.BuiltinFunc{Name: "heappushpop", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		l, ok := a[0].(*object.List)
		if !ok {
			return nil, object.Errorf(i.typeErr, "heap argument must be a list")
		}
		item := a[1]
		if len(l.V) == 0 {
			return item, nil
		}
		less, err := i.lt(l.V[0], item)
		if err != nil {
			return nil, err
		}
		if less {
			item, l.V[0] = l.V[0], item
			if err := heapSiftUp(i, l.V, 0); err != nil {
				return nil, err
			}
		}
		return item, nil
	}})

	m.Dict.SetStr("heapreplace", &object.BuiltinFunc{Name: "heapreplace", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		l, ok := a[0].(*object.List)
		if !ok {
			return nil, object.Errorf(i.typeErr, "heap argument must be a list")
		}
		if len(l.V) == 0 {
			return nil, object.Errorf(i.indexErr, "index out of range")
		}
		top := l.V[0]
		l.V[0] = a[1]
		if err := heapSiftUp(i, l.V, 0); err != nil {
			return nil, err
		}
		return top, nil
	}})

	m.Dict.SetStr("nlargest", &object.BuiltinFunc{Name: "nlargest", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return i.heapNBest(a, kw, true)
	}})
	m.Dict.SetStr("nsmallest", &object.BuiltinFunc{Name: "nsmallest", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return i.heapNBest(a, kw, false)
	}})

	m.Dict.SetStr("merge", &object.BuiltinFunc{Name: "merge", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		reverse := false
		var key object.Object
		if kw != nil {
			if v, ok := kw.GetStr("reverse"); ok {
				reverse = object.Truthy(v)
			}
			if v, ok := kw.GetStr("key"); ok {
				if _, isNone := v.(*object.NoneType); !isNone {
					key = v
				}
			}
		}
		// Materialize inputs and sort the flat collection — adequate for
		// test-scale usage; real CPython uses a tournament tree.
		var all []object.Object
		for _, it := range a {
			items, err := iterate(i, it)
			if err != nil {
				return nil, err
			}
			all = append(all, items...)
		}
		if err := sortListKey(i, all, key, reverse); err != nil {
			return nil, err
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(all) {
				return nil, false, nil
			}
			v := all[idx]
			idx++
			return v, true, nil
		}}, nil
	}})

	return m
}

// heapSiftDown assumes heap[:pos] is a valid heap and sifts the newly-added
// element at pos up toward the root.
func heapSiftDown(i *Interp, heap []object.Object, startPos, pos int) error {
	newItem := heap[pos]
	for pos > startPos {
		parentPos := (pos - 1) >> 1
		parent := heap[parentPos]
		less, err := i.lt(newItem, parent)
		if err != nil {
			return err
		}
		if less {
			heap[pos] = parent
			pos = parentPos
			continue
		}
		break
	}
	heap[pos] = newItem
	return nil
}

// heapSiftUp sifts the element at pos down toward the leaves.
func heapSiftUp(i *Interp, heap []object.Object, pos int) error {
	endPos := len(heap)
	startPos := pos
	newItem := heap[pos]
	childPos := 2*pos + 1
	for childPos < endPos {
		rightPos := childPos + 1
		if rightPos < endPos {
			less, err := i.lt(heap[childPos], heap[rightPos])
			if err != nil {
				return err
			}
			if !less {
				childPos = rightPos
			}
		}
		heap[pos] = heap[childPos]
		pos = childPos
		childPos = 2*pos + 1
	}
	heap[pos] = newItem
	return heapSiftDown(i, heap, startPos, pos)
}

func (i *Interp) heapNBest(a []object.Object, kw *object.Dict, largest bool) (object.Object, error) {
	if len(a) < 2 {
		return nil, object.Errorf(i.typeErr, "requires n and iterable")
	}
	n, ok := toInt64(a[0])
	if !ok {
		return nil, object.Errorf(i.typeErr, "n must be int")
	}
	items, err := iterate(i, a[1])
	if err != nil {
		return nil, err
	}
	var key object.Object
	if kw != nil {
		if v, ok := kw.GetStr("key"); ok {
			if _, isNone := v.(*object.NoneType); !isNone {
				key = v
			}
		}
	}
	cp := append([]object.Object{}, items...)
	if err := sortListKey(i, cp, key, largest); err != nil {
		return nil, err
	}
	if int64(len(cp)) > n {
		cp = cp[:n]
	}
	return &object.List{V: cp}, nil
}

// --- bisect ---

func (i *Interp) buildBisect() *object.Module {
	m := &object.Module{Name: "bisect", Dict: object.NewDict()}

	// bisect_left(a, x, lo=0, hi=len(a), *, key=None)
	bisect := func(left bool) func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "bisect() takes at least 2 arguments")
			}
			lst, ok := a[0].(*object.List)
			if !ok {
				return nil, object.Errorf(i.typeErr, "bisect() requires a list")
			}
			x := a[1]
			lo, hi := 0, len(lst.V)
			if len(a) >= 3 {
				if n, ok := toInt64(a[2]); ok {
					lo = int(n)
				}
			}
			if len(a) >= 4 {
				if n, ok := toInt64(a[3]); ok {
					hi = int(n)
				}
			}
			var key object.Object
			if kw != nil {
				if v, ok := kw.GetStr("lo"); ok {
					if n, ok := toInt64(v); ok {
						lo = int(n)
					}
				}
				if v, ok := kw.GetStr("hi"); ok {
					if n, ok := toInt64(v); ok {
						hi = int(n)
					}
				}
				if v, ok := kw.GetStr("key"); ok {
					if _, isNone := v.(*object.NoneType); !isNone {
						key = v
					}
				}
			}
			if lo < 0 {
				return nil, object.Errorf(i.valueErr, "lo must be non-negative")
			}
			keyOf := func(v object.Object) (object.Object, error) {
				if key == nil {
					return v, nil
				}
				return i.callObject(key, []object.Object{v}, nil)
			}
			// Per CPython bisect semantics the needle is NOT transformed by
			// key; only list elements are. This lets callers search for a
			// key value directly without wrapping it.
			needle := x
			for lo < hi {
				mid := (lo + hi) / 2
				cur, err := keyOf(lst.V[mid])
				if err != nil {
					return nil, err
				}
				var less bool
				if left {
					less, err = i.lt(cur, needle)
				} else {
					less, err = i.lt(needle, cur)
					less = !less
				}
				if err != nil {
					return nil, err
				}
				if less {
					lo = mid + 1
				} else {
					hi = mid
				}
			}
			return object.NewInt(int64(lo)), nil
		}
	}

	m.Dict.SetStr("bisect_left", &object.BuiltinFunc{Name: "bisect_left", Call: bisect(true)})
	m.Dict.SetStr("bisect_right", &object.BuiltinFunc{Name: "bisect_right", Call: bisect(false)})
	m.Dict.SetStr("bisect", &object.BuiltinFunc{Name: "bisect", Call: bisect(false)})

	insort := func(left bool) func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			lst, ok := a[0].(*object.List)
			if !ok {
				return nil, object.Errorf(i.typeErr, "insort() requires a list")
			}
			// insort differs from bisect when a key is supplied: the new item
			// must also be keyified so it compares with the key view of
			// existing elements. Do it here before forwarding to bisect.
			bisectArgs := a
			if kw != nil {
				if keyFn, ok := kw.GetStr("key"); ok {
					if _, isNone := keyFn.(*object.NoneType); !isNone {
						needleKey, err := i.callObject(keyFn, []object.Object{a[1]}, nil)
						if err != nil {
							return nil, err
						}
						bisectArgs = append([]object.Object{a[0], needleKey}, a[2:]...)
					}
				}
			}
			idxObj, err := bisect(left)(nil, bisectArgs, kw)
			if err != nil {
				return nil, err
			}
			idx, _ := toInt64(idxObj)
			p := int(idx)
			lst.V = append(lst.V, nil)
			copy(lst.V[p+1:], lst.V[p:])
			lst.V[p] = a[1]
			return object.None, nil
		}
	}

	m.Dict.SetStr("insort_left", &object.BuiltinFunc{Name: "insort_left", Call: insort(true)})
	m.Dict.SetStr("insort_right", &object.BuiltinFunc{Name: "insort_right", Call: insort(false)})
	m.Dict.SetStr("insort", &object.BuiltinFunc{Name: "insort", Call: insort(false)})

	return m
}

// --- random (seedable PRNG, independent from Go's global rand) ---

// buildRandom exposes a minimal random module. Each module instance keeps
// its own *rand.Rand so seed() is deterministic and doesn't leak into Go's
// global source. Crypto-grade randomness is not a goal here.
func (i *Interp) buildRandom() *object.Module {
	m := &object.Module{Name: "random", Dict: object.NewDict()}
	rng := gorand.New(gorand.NewSource(1))

	m.Dict.SetStr("seed", &object.BuiltinFunc{Name: "seed", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var s int64
		if len(a) == 0 {
			s = 0
		} else if n, ok := toInt64(a[0]); ok {
			s = n
		} else if f, ok := toFloat64Any(a[0]); ok {
			s = int64(f)
		} else {
			// Hash other hashable values to a seed.
			h, err := object.Hash(a[0])
			if err != nil {
				return nil, err
			}
			s = int64(h)
		}
		rng.Seed(s)
		return object.None, nil
	}})

	m.Dict.SetStr("random", &object.BuiltinFunc{Name: "random", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Float{V: rng.Float64()}, nil
	}})

	m.Dict.SetStr("uniform", &object.BuiltinFunc{Name: "uniform", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		lo, _ := toFloat64Any(a[0])
		hi, _ := toFloat64Any(a[1])
		return &object.Float{V: lo + rng.Float64()*(hi-lo)}, nil
	}})

	m.Dict.SetStr("randint", &object.BuiltinFunc{Name: "randint", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		lo, _ := toInt64(a[0])
		hi, _ := toInt64(a[1])
		if hi < lo {
			return nil, object.Errorf(i.valueErr, "empty range for randint")
		}
		return object.NewInt(lo + rng.Int63n(hi-lo+1)), nil
	}})

	m.Dict.SetStr("randrange", &object.BuiltinFunc{Name: "randrange", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var start, stop, step int64 = 0, 0, 1
		switch len(a) {
		case 1:
			stop, _ = toInt64(a[0])
		case 2:
			start, _ = toInt64(a[0])
			stop, _ = toInt64(a[1])
		case 3:
			start, _ = toInt64(a[0])
			stop, _ = toInt64(a[1])
			step, _ = toInt64(a[2])
		default:
			return nil, object.Errorf(i.typeErr, "randrange() takes 1-3 arguments")
		}
		if step == 0 {
			return nil, object.Errorf(i.valueErr, "step must be non-zero")
		}
		width := stop - start
		if (step > 0 && width <= 0) || (step < 0 && width >= 0) {
			return nil, object.Errorf(i.valueErr, "empty range for randrange")
		}
		n := (width + step - sign(step)) / step
		if n <= 0 {
			return nil, object.Errorf(i.valueErr, "empty range for randrange")
		}
		return object.NewInt(start + step*rng.Int63n(n)), nil
	}})

	m.Dict.SetStr("choice", &object.BuiltinFunc{Name: "choice", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, object.Errorf(i.indexErr, "Cannot choose from an empty sequence")
		}
		return items[rng.Intn(len(items))], nil
	}})

	m.Dict.SetStr("choices", &object.BuiltinFunc{Name: "choices", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		k := 1
		if kw != nil {
			if v, ok := kw.GetStr("k"); ok {
				if n, ok := toInt64(v); ok {
					k = int(n)
				}
			}
		}
		if len(items) == 0 {
			return nil, object.Errorf(i.indexErr, "Cannot choose from an empty sequence")
		}
		out := make([]object.Object, k)
		for j := 0; j < k; j++ {
			out[j] = items[rng.Intn(len(items))]
		}
		return &object.List{V: out}, nil
	}})

	m.Dict.SetStr("shuffle", &object.BuiltinFunc{Name: "shuffle", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		lst, ok := a[0].(*object.List)
		if !ok {
			return nil, object.Errorf(i.typeErr, "shuffle requires a list")
		}
		for k := len(lst.V) - 1; k > 0; k-- {
			j := rng.Intn(k + 1)
			lst.V[k], lst.V[j] = lst.V[j], lst.V[k]
		}
		return object.None, nil
	}})

	m.Dict.SetStr("sample", &object.BuiltinFunc{Name: "sample", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		k, _ := toInt64(a[1])
		if int(k) > len(items) {
			return nil, object.Errorf(i.valueErr, "sample larger than population")
		}
		pool := append([]object.Object{}, items...)
		out := make([]object.Object, k)
		for j := int64(0); j < k; j++ {
			idx := rng.Intn(len(pool) - int(j))
			out[j] = pool[idx]
			pool[idx] = pool[len(pool)-1-int(j)]
		}
		return &object.List{V: out}, nil
	}})

	return m
}

func sign(n int64) int64 {
	if n > 0 {
		return 1
	}
	if n < 0 {
		return -1
	}
	return 0
}

package vm

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"unicode"

	"github.com/tamnd/goipy/object"
)

// fracState holds a reduced rational number: value = num/den, den > 0.
type fracState struct {
	num *big.Int // signed numerator
	den *big.Int // always positive denominator
}

func fracReduce(num, den *big.Int) fracState {
	// Ensure den > 0.
	if den.Sign() < 0 {
		num = new(big.Int).Neg(num)
		den = new(big.Int).Neg(den)
	} else {
		num = new(big.Int).Set(num)
		den = new(big.Int).Set(den)
	}
	if den.Sign() == 0 {
		// denominator 0 — caller should have checked
		return fracState{num: num, den: den}
	}
	g := new(big.Int).GCD(nil, nil, new(big.Int).Abs(num), den)
	if g.Cmp(big.NewInt(1)) != 0 {
		num.Quo(num, g)
		den.Quo(den, g)
	}
	return fracState{num: num, den: den}
}

func fracZero() fracState {
	return fracState{num: big.NewInt(0), den: big.NewInt(1)}
}

// newFracInst creates a Python Fraction instance from fracState.
func newFracInst(cls *object.Class, f fracState) *object.Instance {
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("_n", object.IntFromBig(f.num))
	inst.Dict.SetStr("_d", object.IntFromBig(f.den))
	return inst
}

// getFracState retrieves fracState from a Fraction instance.
func getFracState(inst *object.Instance) (fracState, bool) {
	nv, ok1 := inst.Dict.GetStr("_n")
	dv, ok2 := inst.Dict.GetStr("_d")
	if !ok1 || !ok2 {
		return fracState{}, false
	}
	ni, ok3 := nv.(*object.Int)
	di, ok4 := dv.(*object.Int)
	if !ok3 || !ok4 {
		return fracState{}, false
	}
	return fracState{
		num: new(big.Int).Set(&ni.V),
		den: new(big.Int).Set(&di.V),
	}, true
}

// fracParseStr parses a Fraction from strings like "3/4", "-1/2", "1.5", "3".
func fracParseStr(s string) (fracState, error) {
	s = strings.TrimSpace(s)
	// Remove underscores (Python 3.6+ allows underscores in numeric literals).
	s = strings.ReplaceAll(s, "_", "")
	if s == "" {
		return fracState{}, fmt.Errorf("invalid literal for Fraction: empty string")
	}

	// Try "numerator/denominator".
	if idx := strings.Index(s, "/"); idx >= 0 {
		numStr := strings.TrimSpace(s[:idx])
		denStr := strings.TrimSpace(s[idx+1:])
		num, ok1 := parseBigInt(numStr)
		den, ok2 := parseBigInt(denStr)
		if !ok1 || !ok2 {
			return fracState{}, fmt.Errorf("invalid literal for Fraction: '%s'", s)
		}
		if den.Sign() == 0 {
			return fracState{}, fmt.Errorf("Fraction('%s'): denominator is zero", s)
		}
		return fracReduce(num, den), nil
	}

	// Try decimal / integer string (may have E notation).
	return fracFromDecStr(s)
}

func parseBigInt(s string) (*big.Int, bool) {
	s = strings.TrimSpace(s)
	neg := false
	if s != "" && s[0] == '-' {
		neg = true
		s = s[1:]
	} else if s != "" && s[0] == '+' {
		s = s[1:]
	}
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return nil, false
		}
	}
	if s == "" {
		return nil, false
	}
	n := new(big.Int)
	n.SetString(s, 10)
	if neg {
		n.Neg(n)
	}
	return n, true
}

// fracFromDecStr converts a decimal/integer string (no '/') to a Fraction.
func fracFromDecStr(s string) (fracState, error) {
	neg := false
	if s != "" && s[0] == '-' {
		neg = true
		s = s[1:]
	} else if s != "" && s[0] == '+' {
		s = s[1:]
	}

	exp := 0
	if eIdx := strings.IndexAny(s, "eE"); eIdx >= 0 {
		expStr := s[eIdx+1:]
		s = s[:eIdx]
		e, err := fracParseIntStr(expStr)
		if err != nil {
			return fracState{}, fmt.Errorf("invalid Fraction literal")
		}
		exp = e
	}

	dotIdx := strings.Index(s, ".")
	fracDigits := 0
	if dotIdx >= 0 {
		fracDigits = len(s) - dotIdx - 1
		s = s[:dotIdx] + s[dotIdx+1:]
	}
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return fracState{}, fmt.Errorf("invalid Fraction literal: '%s'", s)
		}
	}
	if s == "" {
		return fracState{}, fmt.Errorf("invalid Fraction literal")
	}

	num := new(big.Int)
	num.SetString(s, 10)
	netExp := exp - fracDigits // net power of 10

	var den *big.Int
	if netExp >= 0 {
		// Multiply numerator by 10^netExp.
		scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(netExp)), nil)
		num.Mul(num, scale)
		den = big.NewInt(1)
	} else {
		den = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-netExp)), nil)
	}
	if neg {
		num.Neg(num)
	}
	return fracReduce(num, den), nil
}

func fracParseIntStr(s string) (int, error) {
	neg := false
	if s != "" && s[0] == '-' {
		neg = true
		s = s[1:]
	} else if s != "" && s[0] == '+' {
		s = s[1:]
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("non-digit")
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return n, nil
}

// fracFromFloat converts a float64 to an exact Fraction using its IEEE 754 ratio.
func fracFromFloat(f float64) (fracState, error) {
	if math.IsInf(f, 0) {
		return fracState{}, fmt.Errorf("cannot convert Inf to Fraction")
	}
	if math.IsNaN(f) {
		return fracState{}, fmt.Errorf("cannot convert NaN to Fraction")
	}
	rat := new(big.Rat).SetFloat64(f)
	n := new(big.Int).Set(rat.Num())
	d := new(big.Int).Set(rat.Denom())
	return fracReduce(n, d), nil
}

// fracToFloat converts a Fraction to float64.
func fracToFloat(f fracState) float64 {
	r := new(big.Rat).SetFrac(f.num, f.den)
	v, _ := r.Float64()
	return v
}

// fracAdd returns a + b.
func fracAdd(a, b fracState) fracState {
	// a.num/a.den + b.num/b.den = (a.num*b.den + b.num*a.den) / (a.den*b.den)
	num := new(big.Int).Add(
		new(big.Int).Mul(a.num, b.den),
		new(big.Int).Mul(b.num, a.den),
	)
	den := new(big.Int).Mul(a.den, b.den)
	return fracReduce(num, den)
}

// fracSub returns a - b.
func fracSub(a, b fracState) fracState {
	num := new(big.Int).Sub(
		new(big.Int).Mul(a.num, b.den),
		new(big.Int).Mul(b.num, a.den),
	)
	den := new(big.Int).Mul(a.den, b.den)
	return fracReduce(num, den)
}

// fracMul returns a * b.
func fracMul(a, b fracState) fracState {
	num := new(big.Int).Mul(a.num, b.num)
	den := new(big.Int).Mul(a.den, b.den)
	return fracReduce(num, den)
}

// fracDiv returns a / b.
func fracDiv(a, b fracState) (fracState, error) {
	if b.num.Sign() == 0 {
		return fracState{}, fmt.Errorf("Fraction(%s, %s): denominator is zero", a.num, b.num)
	}
	num := new(big.Int).Mul(a.num, b.den)
	den := new(big.Int).Mul(a.den, b.num)
	return fracReduce(num, den), nil
}

// fracFloorDiv returns floor(a / b) as an integer.
func fracFloorDiv(a, b fracState) (*big.Int, error) {
	if b.num.Sign() == 0 {
		return nil, fmt.Errorf("division by zero")
	}
	// floor(a.num * b.den / (a.den * b.num))
	num := new(big.Int).Mul(a.num, b.den)
	den := new(big.Int).Mul(a.den, b.num)
	// Euclidean division: adjust for floor toward -inf.
	q := new(big.Int).Quo(num, den)
	r := new(big.Int).Sub(num, new(big.Int).Mul(q, den))
	if r.Sign() != 0 && (num.Sign() < 0) != (den.Sign() < 0) {
		q.Sub(q, big.NewInt(1))
	}
	return q, nil
}

// fracMod returns a % b = a - floor(a/b)*b.
func fracMod(a, b fracState) (fracState, error) {
	if b.num.Sign() == 0 {
		return fracState{}, fmt.Errorf("division by zero")
	}
	q, err := fracFloorDiv(a, b)
	if err != nil {
		return fracState{}, err
	}
	// a - q*b
	qFrac := fracState{num: q, den: big.NewInt(1)}
	return fracSub(a, fracMul(qFrac, b)), nil
}

// fracPow raises a to an integer power. For negative powers, result is Fraction.
func fracPow(a fracState, exp int) (fracState, error) {
	if exp == 0 {
		return fracState{num: big.NewInt(1), den: big.NewInt(1)}, nil
	}
	if exp < 0 {
		if a.num.Sign() == 0 {
			return fracState{}, fmt.Errorf("Fraction(0) cannot be raised to a negative power")
		}
		// a^(-n) = (den/num)^n
		a = fracState{num: new(big.Int).Set(a.den), den: new(big.Int).Abs(a.num)}
		if a.den.Sign() == 0 {
			return fracState{}, fmt.Errorf("zero denominator")
		}
		// Handle sign: if original num was negative, (-1/d)^n
		exp = -exp
	}
	n := new(big.Int).Exp(a.num, big.NewInt(int64(exp)), nil)
	d := new(big.Int).Exp(a.den, big.NewInt(int64(exp)), nil)
	return fracReduce(n, d), nil
}

// fracCmp returns -1, 0, 1 for a < b, a == b, a > b.
func fracCmp(a, b fracState) int {
	// Compare a.num * b.den vs b.num * a.den
	lhs := new(big.Int).Mul(a.num, b.den)
	rhs := new(big.Int).Mul(b.num, a.den)
	return lhs.Cmp(rhs)
}

// fracLimitDenominator finds the closest Fraction with denominator <= maxDen.
// Direct port of CPython Lib/fractions.py limit_denominator().
func fracLimitDenominator(f fracState, maxDen int64) fracState {
	if maxDen < 1 {
		maxDen = 1
	}
	maxDenBI := big.NewInt(maxDen)
	if f.den.Cmp(maxDenBI) <= 0 {
		return f
	}

	p0, q0 := big.NewInt(0), big.NewInt(1)
	p1, q1 := big.NewInt(1), big.NewInt(0)
	// Work with |numerator| / denominator; restore sign at the end.
	n := new(big.Int).Abs(f.num)
	d := new(big.Int).Set(f.den)

	for {
		a := new(big.Int).Quo(n, d)
		q2 := new(big.Int).Add(q0, new(big.Int).Mul(a, q1))
		if q2.Cmp(maxDenBI) > 0 {
			break
		}
		// p0, q0, p1, q1 = p1, q1, p0+a*p1, q2
		newP1 := new(big.Int).Add(p0, new(big.Int).Mul(a, p1))
		p0.Set(p1)
		q0.Set(q1)
		p1.Set(newP1)
		q1.Set(q2)
		// n, d = d, n - a*d
		newD := new(big.Int).Sub(n, new(big.Int).Mul(a, d))
		n.Set(d)
		d.Set(newD)
	}

	k := new(big.Int).Quo(new(big.Int).Sub(maxDenBI, q0), q1)

	// bound1 = (p0 + k*p1) / (q0 + k*q1)
	b1num := new(big.Int).Add(p0, new(big.Int).Mul(k, p1))
	b1den := new(big.Int).Add(q0, new(big.Int).Mul(k, q1))
	// bound2 = p1 / q1
	b2num := new(big.Int).Set(p1)
	b2den := new(big.Int).Set(q1)

	// Compare |bound2 - f| <= |bound1 - f| using cross-multiplication.
	// |bXnum/bXden - n/f.den| → |bXnum*f.den - n*bXden| / (bXden * f.den)
	absN := new(big.Int).Abs(f.num)
	diff2 := new(big.Int).Abs(new(big.Int).Sub(
		new(big.Int).Mul(b2num, f.den),
		new(big.Int).Mul(absN, b2den),
	))
	diff1 := new(big.Int).Abs(new(big.Int).Sub(
		new(big.Int).Mul(b1num, f.den),
		new(big.Int).Mul(absN, b1den),
	))
	// diff2/b2den <= diff1/b1den  ↔  diff2*b1den <= diff1*b2den
	lhs := new(big.Int).Mul(diff2, b1den)
	rhs := new(big.Int).Mul(diff1, b2den)

	var resNum, resDen *big.Int
	if lhs.Cmp(rhs) <= 0 {
		resNum, resDen = b2num, b2den
	} else {
		resNum, resDen = b1num, b1den
	}
	if f.num.Sign() < 0 {
		resNum = new(big.Int).Neg(resNum)
	}
	return fracReduce(resNum, resDen)
}

func (i *Interp) buildFractions() *object.Module {
	m := &object.Module{Name: "fractions", Dict: object.NewDict()}

	fracCls := &object.Class{Name: "Fraction", Dict: object.NewDict()}

	// Helper: coerce any Python object to fracState.
	toFrac := func(o object.Object) (fracState, bool) {
		switch v := o.(type) {
		case *object.Instance:
			f, ok := getFracState(v)
			return f, ok
		case *object.Int:
			return fracState{num: new(big.Int).Set(&v.V), den: big.NewInt(1)}, true
		case *object.Bool:
			if v.V {
				return fracState{num: big.NewInt(1), den: big.NewInt(1)}, true
			}
			return fracZero(), true
		case *object.Float:
			f, err := fracFromFloat(v.V)
			if err != nil {
				return fracState{}, false
			}
			return f, true
		}
		return fracState{}, false
	}

	// __new__(cls, numerator=0, denominator=None)
	fracCls.Dict.SetStr("__new__", &object.BuiltinFunc{Name: "__new__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		cls := fracCls
		if len(a) >= 1 {
			if c, ok := a[0].(*object.Class); ok {
				cls = c
				a = a[1:]
			}
		}
		if len(a) == 0 {
			return newFracInst(cls, fracZero()), nil
		}

		// Single-arg: Fraction(x) where x is int, float, Fraction, Decimal, or string.
		if len(a) == 1 {
			switch v := a[0].(type) {
			case *object.Str:
				f, err := fracParseStr(v.V)
				if err != nil {
					return nil, object.Errorf(i.valueErr, "%s", err.Error())
				}
				return newFracInst(cls, f), nil
			case *object.Int:
				return newFracInst(cls, fracState{num: new(big.Int).Set(&v.V), den: big.NewInt(1)}), nil
			case *object.Bool:
				if v.V {
					return newFracInst(cls, fracState{num: big.NewInt(1), den: big.NewInt(1)}), nil
				}
				return newFracInst(cls, fracZero()), nil
			case *object.Float:
				f, err := fracFromFloat(v.V)
				if err != nil {
					return nil, object.Errorf(i.valueErr, "%s", err.Error())
				}
				return newFracInst(cls, f), nil
			case *object.Instance:
				// Fraction or Decimal instance.
				if fs, ok := getFracState(v); ok {
					return newFracInst(cls, fs), nil
				}
				// Try Decimal: read _n/_d or _s/_c/_e/_f.
				if ds, ok := fracFromDecimalInst(v); ok {
					return newFracInst(cls, ds), nil
				}
				return nil, object.Errorf(i.typeErr, "Fraction() argument must be a numeric type")
			}
			return nil, object.Errorf(i.typeErr, "Fraction() argument must be a numeric type or string")
		}

		// Two-arg: Fraction(numerator, denominator).
		if len(a) >= 2 {
			numF, ok1 := toFrac(a[0])
			denF, ok2 := toFrac(a[1])
			if !ok1 || !ok2 {
				return nil, object.Errorf(i.typeErr, "Fraction() requires numeric arguments")
			}
			if denF.num.Sign() == 0 {
				return nil, object.Errorf(i.valueErr, "Fraction(%s, 0): denominator is zero", a[0])
			}
			// Fraction(n1/d1, n2/d2) = (n1*d2) / (d1*n2)
			num := new(big.Int).Mul(numF.num, denF.den)
			den := new(big.Int).Mul(numF.den, denF.num)
			return newFracInst(cls, fracReduce(num, den)), nil
		}
		return newFracInst(cls, fracZero()), nil
	}})

	// numerator property.
	fracCls.Dict.SetStr("numerator", &object.Property{
		Fget: &object.BuiltinFunc{Name: "numerator", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NewInt(0), nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.NewInt(0), nil
			}
			f, ok := getFracState(inst)
			if !ok {
				return object.NewInt(0), nil
			}
			return object.IntFromBig(f.num), nil
		}},
	})

	// denominator property.
	fracCls.Dict.SetStr("denominator", &object.Property{
		Fget: &object.BuiltinFunc{Name: "denominator", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NewInt(1), nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.NewInt(1), nil
			}
			f, ok := getFracState(inst)
			if !ok {
				return object.NewInt(1), nil
			}
			return object.IntFromBig(f.den), nil
		}},
	})

	// __repr__ → "Fraction(n, d)"
	fracCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "Fraction(0, 1)"}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "Fraction(0, 1)"}, nil
		}
		f, ok := getFracState(inst)
		if !ok {
			return &object.Str{V: "Fraction(0, 1)"}, nil
		}
		if f.den.Cmp(big.NewInt(1)) == 0 {
			return &object.Str{V: fmt.Sprintf("Fraction(%s, 1)", f.num.String())}, nil
		}
		return &object.Str{V: fmt.Sprintf("Fraction(%s, %s)", f.num.String(), f.den.String())}, nil
	}})

	// __str__ → "n/d" (or "n" when denominator is 1)
	fracCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "0"}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "0"}, nil
		}
		f, ok := getFracState(inst)
		if !ok {
			return &object.Str{V: "0"}, nil
		}
		if f.den.Cmp(big.NewInt(1)) == 0 {
			return &object.Str{V: f.num.String()}, nil
		}
		return &object.Str{V: fmt.Sprintf("%s/%s", f.num.String(), f.den.String())}, nil
	}})

	// __float__
	fracCls.Dict.SetStr("__float__", &object.BuiltinFunc{Name: "__float__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Float{V: 0}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Float{V: 0}, nil
		}
		f, ok := getFracState(inst)
		if !ok {
			return &object.Float{V: 0}, nil
		}
		return &object.Float{V: fracToFloat(f)}, nil
	}})

	// __int__ → truncates toward zero
	fracCls.Dict.SetStr("__int__", &object.BuiltinFunc{Name: "__int__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		f, ok := getFracState(inst)
		if !ok {
			return object.NewInt(0), nil
		}
		q := new(big.Int).Quo(f.num, f.den)
		return object.IntFromBig(q), nil
	}})

	// __trunc__ (same as __int__ for Fraction)
	fracCls.Dict.SetStr("__trunc__", &object.BuiltinFunc{Name: "__trunc__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		f, ok := getFracState(inst)
		if !ok {
			return object.NewInt(0), nil
		}
		q := new(big.Int).Quo(f.num, f.den)
		return object.IntFromBig(q), nil
	}})

	// __floor__ → greatest int <= self
	fracCls.Dict.SetStr("__floor__", &object.BuiltinFunc{Name: "__floor__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		f, ok := getFracState(inst)
		if !ok {
			return object.NewInt(0), nil
		}
		// floor(n/d): q = n div d, but floor toward -inf.
		q := new(big.Int).Quo(f.num, f.den)
		r := new(big.Int).Sub(f.num, new(big.Int).Mul(q, f.den))
		if r.Sign() < 0 {
			q.Sub(q, big.NewInt(1))
		}
		return object.IntFromBig(q), nil
	}})

	// __ceil__ → smallest int >= self
	fracCls.Dict.SetStr("__ceil__", &object.BuiltinFunc{Name: "__ceil__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		f, ok := getFracState(inst)
		if !ok {
			return object.NewInt(0), nil
		}
		// ceil(n/d) = -floor(-n/d)
		neg := fracState{num: new(big.Int).Neg(f.num), den: f.den}
		q := new(big.Int).Quo(neg.num, neg.den)
		r := new(big.Int).Sub(neg.num, new(big.Int).Mul(q, neg.den))
		if r.Sign() < 0 {
			q.Sub(q, big.NewInt(1))
		}
		q.Neg(q)
		return object.IntFromBig(q), nil
	}})

	// __round__(ndigits=None) — round to nearest int (half-to-even) or to ndigits.
	fracCls.Dict.SetStr("__round__", &object.BuiltinFunc{Name: "__round__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		f, ok := getFracState(inst)
		if !ok {
			return object.NewInt(0), nil
		}

		if len(a) == 1 {
			// round to nearest int, ties to even.
			result := fracRoundHalfEven(f)
			return object.IntFromBig(result), nil
		}

		// round(frac, ndigits) → Fraction with denominator 10^ndigits.
		ndigits, ok2 := toInt64(a[1])
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "__round__ requires an integer ndigits")
		}
		if ndigits >= 0 {
			scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(ndigits), nil)
			// Round f * scale to nearest integer (half-even), then divide by scale.
			scaled := fracMul(f, fracState{num: scale, den: big.NewInt(1)})
			rounded := fracRoundHalfEven(scaled)
			result := fracReduce(rounded, scale)
			return newFracInst(fracCls, result), nil
		}
		// Negative ndigits: round to nearest 10^(-ndigits).
		scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(-ndigits), nil)
		scaledF := fracState{num: scale, den: big.NewInt(1)}
		divided := fracState{num: f.num, den: new(big.Int).Mul(f.den, scale)}
		rounded := fracRoundHalfEven(divided)
		result := fracMul(fracState{num: rounded, den: big.NewInt(1)}, scaledF)
		return object.IntFromBig(result.num), nil
	}})

	// __hash__
	fracCls.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		f, ok := getFracState(inst)
		if !ok {
			return object.NewInt(0), nil
		}
		// CPython: hash(Fraction(n,d)) = hash(n/d) using sys.hash_info.modulus=2305843009213693951.
		// Simple approximation: use float hash or int hash when denominator=1.
		if f.den.Cmp(big.NewInt(1)) == 0 {
			h := f.num.Int64()
			return object.NewInt(h), nil
		}
		fv := fracToFloat(f)
		h := int64(math.Round(fv * 1e9))
		return object.NewInt(h), nil
	}})

	// Comparison operators.
	cmpOp := func(name string, check func(int) bool) {
		fracCls.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.NotImplemented, nil
			}
			af, ok1 := toFrac(a[0])
			bf, ok2 := toFrac(a[1])
			if !ok1 || !ok2 {
				return object.NotImplemented, nil
			}
			return object.BoolOf(check(fracCmp(af, bf))), nil
		}})
	}
	cmpOp("__eq__", func(c int) bool { return c == 0 })
	cmpOp("__lt__", func(c int) bool { return c < 0 })
	cmpOp("__le__", func(c int) bool { return c <= 0 })
	cmpOp("__gt__", func(c int) bool { return c > 0 })
	cmpOp("__ge__", func(c int) bool { return c >= 0 })

	// Arithmetic operators.
	binArith := func(name string, fn func(a, b fracState) (fracState, error)) {
		fracCls.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.NotImplemented, nil
			}
			af, ok1 := toFrac(a[0])
			bf, ok2 := toFrac(a[1])
			if !ok1 || !ok2 {
				return object.NotImplemented, nil
			}
			res, err := fn(af, bf)
			if err != nil {
				return nil, object.Errorf(i.valueErr, "%s", err.Error())
			}
			return newFracInst(fracCls, res), nil
		}})
	}
	binArith("__add__", func(a, b fracState) (fracState, error) { return fracAdd(a, b), nil })
	binArith("__radd__", func(a, b fracState) (fracState, error) { return fracAdd(b, a), nil })
	binArith("__sub__", func(a, b fracState) (fracState, error) { return fracSub(a, b), nil })
	binArith("__rsub__", func(a, b fracState) (fracState, error) { return fracSub(b, a), nil })
	binArith("__mul__", func(a, b fracState) (fracState, error) { return fracMul(a, b), nil })
	binArith("__rmul__", func(a, b fracState) (fracState, error) { return fracMul(b, a), nil })
	binArith("__truediv__", fracDiv)
	binArith("__rtruediv__", func(a, b fracState) (fracState, error) { return fracDiv(b, a) })
	binArith("__mod__", func(a, b fracState) (fracState, error) { return fracMod(a, b) })
	binArith("__rmod__", func(a, b fracState) (fracState, error) { return fracMod(b, a) })

	// __floordiv__ and __rfloordiv__ return an integer.
	floorDivFn := func(name string, flip bool) {
		fracCls.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.NotImplemented, nil
			}
			af, ok1 := toFrac(a[0])
			bf, ok2 := toFrac(a[1])
			if !ok1 || !ok2 {
				return object.NotImplemented, nil
			}
			if flip {
				af, bf = bf, af
			}
			q, err := fracFloorDiv(af, bf)
			if err != nil {
				return nil, object.Errorf(i.valueErr, "%s", err.Error())
			}
			return object.IntFromBig(q), nil
		}})
	}
	floorDivFn("__floordiv__", false)
	floorDivFn("__rfloordiv__", true)

	// __pow__
	fracCls.Dict.SetStr("__pow__", &object.BuiltinFunc{Name: "__pow__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.NotImplemented, nil
		}
		af, ok := toFrac(a[0])
		if !ok {
			return object.NotImplemented, nil
		}
		// Exponent must be integer for exact result.
		switch ev := a[1].(type) {
		case *object.Int:
			exp64, ok2 := toInt64(a[1])
			if !ok2 {
				// Very large exponent — use float fallback.
				fv := fracToFloat(af)
				ef, _ := ev.V.Float64()
				return &object.Float{V: math.Pow(fv, ef)}, nil
			}
			res, err := fracPow(af, int(exp64))
			if err != nil {
				return nil, object.Errorf(i.valueErr, "%s", err.Error())
			}
			return newFracInst(fracCls, res), nil
		case *object.Bool:
			if ev.V {
				return newFracInst(fracCls, af), nil
			}
			return newFracInst(fracCls, fracState{num: big.NewInt(1), den: big.NewInt(1)}), nil
		case *object.Instance:
			// Fraction exponent — only if it simplifies to an integer.
			bf, ok2 := getFracState(ev)
			if !ok2 {
				return object.NotImplemented, nil
			}
			if bf.den.Cmp(big.NewInt(1)) != 0 {
				// Non-integer exponent → float.
				fv := fracToFloat(af)
				bv := fracToFloat(bf)
				return &object.Float{V: math.Pow(fv, bv)}, nil
			}
			exp64 := bf.num.Int64()
			res, err := fracPow(af, int(exp64))
			if err != nil {
				return nil, object.Errorf(i.valueErr, "%s", err.Error())
			}
			return newFracInst(fracCls, res), nil
		}
		return object.NotImplemented, nil
	}})

	fracCls.Dict.SetStr("__rpow__", &object.BuiltinFunc{Name: "__rpow__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// base ** Fraction(self)
		if len(a) < 2 {
			return object.NotImplemented, nil
		}
		bf, ok := toFrac(a[0])
		if !ok {
			return object.NotImplemented, nil
		}
		// a[1] is the base (LHS).
		baseFrac, ok2 := toFrac(a[1])
		if !ok2 {
			return object.NotImplemented, nil
		}
		// If bf is integer, use fracPow.
		if bf.den.Cmp(big.NewInt(1)) == 0 {
			exp64 := bf.num.Int64()
			res, err := fracPow(baseFrac, int(exp64))
			if err != nil {
				return nil, object.Errorf(i.valueErr, "%s", err.Error())
			}
			return newFracInst(fracCls, res), nil
		}
		fv := fracToFloat(baseFrac)
		bv := fracToFloat(bf)
		return &object.Float{V: math.Pow(fv, bv)}, nil
	}})

	// __neg__, __pos__, __abs__
	fracCls.Dict.SetStr("__neg__", &object.BuiltinFunc{Name: "__neg__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NotImplemented, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NotImplemented, nil
		}
		f, ok := getFracState(inst)
		if !ok {
			return object.NotImplemented, nil
		}
		return newFracInst(fracCls, fracState{num: new(big.Int).Neg(f.num), den: f.den}), nil
	}})
	fracCls.Dict.SetStr("__pos__", &object.BuiltinFunc{Name: "__pos__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NotImplemented, nil
		}
		if _, ok := a[0].(*object.Instance); !ok {
			return object.NotImplemented, nil
		}
		return a[0], nil
	}})
	fracCls.Dict.SetStr("__abs__", &object.BuiltinFunc{Name: "__abs__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NotImplemented, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NotImplemented, nil
		}
		f, ok := getFracState(inst)
		if !ok {
			return object.NotImplemented, nil
		}
		return newFracInst(fracCls, fracState{num: new(big.Int).Abs(f.num), den: f.den}), nil
	}})

	// __bool__
	fracCls.Dict.SetStr("__bool__", &object.BuiltinFunc{Name: "__bool__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.False, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.True, nil
		}
		f, ok := getFracState(inst)
		if !ok {
			return object.False, nil
		}
		return object.BoolOf(f.num.Sign() != 0), nil
	}})

	// as_integer_ratio() → (numerator, denominator)
	fracCls.Dict.SetStr("as_integer_ratio", &object.BuiltinFunc{Name: "as_integer_ratio", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "as_integer_ratio() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "as_integer_ratio() requires a Fraction")
		}
		f, ok := getFracState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "as_integer_ratio() requires a Fraction")
		}
		return &object.Tuple{V: []object.Object{
			object.IntFromBig(f.num),
			object.IntFromBig(f.den),
		}}, nil
	}})

	// is_integer() → True if denominator == 1
	fracCls.Dict.SetStr("is_integer", &object.BuiltinFunc{Name: "is_integer", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.False, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.False, nil
		}
		f, ok := getFracState(inst)
		if !ok {
			return object.False, nil
		}
		return object.BoolOf(f.den.Cmp(big.NewInt(1)) == 0), nil
	}})

	// limit_denominator(max_denominator=1000000)
	fracCls.Dict.SetStr("limit_denominator", &object.BuiltinFunc{Name: "limit_denominator", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "limit_denominator() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "limit_denominator() requires a Fraction")
		}
		f, ok := getFracState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "limit_denominator() requires a Fraction")
		}
		maxDen := int64(1000000)
		if len(a) >= 2 {
			if n, ok2 := toInt64(a[1]); ok2 {
				maxDen = n
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("max_denominator"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					maxDen = n
				}
			}
		}
		return newFracInst(fracCls, fracLimitDenominator(f, maxDen)), nil
	}})

	// from_float(f) — class method
	fracCls.Dict.SetStr("from_float", &object.BuiltinFunc{Name: "from_float", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// May be called as Fraction.from_float(f) or as classmethod.
		args := a
		if len(args) >= 1 {
			if _, ok := args[0].(*object.Class); ok {
				args = args[1:]
			}
		}
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "from_float() requires a float argument")
		}
		switch v := args[0].(type) {
		case *object.Float:
			f, err := fracFromFloat(v.V)
			if err != nil {
				return nil, object.Errorf(i.valueErr, "%s", err.Error())
			}
			return newFracInst(fracCls, f), nil
		case *object.Int:
			return newFracInst(fracCls, fracState{num: new(big.Int).Set(&v.V), den: big.NewInt(1)}), nil
		case *object.Bool:
			if v.V {
				return newFracInst(fracCls, fracState{num: big.NewInt(1), den: big.NewInt(1)}), nil
			}
			return newFracInst(fracCls, fracZero()), nil
		}
		return nil, object.Errorf(i.typeErr, "from_float() requires a float or integral argument")
	}})

	// from_decimal(dec) — class method
	fracCls.Dict.SetStr("from_decimal", &object.BuiltinFunc{Name: "from_decimal", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		args := a
		if len(args) >= 1 {
			if _, ok := args[0].(*object.Class); ok {
				args = args[1:]
			}
		}
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "from_decimal() requires a Decimal argument")
		}
		switch v := args[0].(type) {
		case *object.Instance:
			if ds, ok := fracFromDecimalInst(v); ok {
				return newFracInst(fracCls, ds), nil
			}
			// Fallback: try str conversion.
			return nil, object.Errorf(i.typeErr, "from_decimal() requires a Decimal instance")
		case *object.Int:
			return newFracInst(fracCls, fracState{num: new(big.Int).Set(&v.V), den: big.NewInt(1)}), nil
		case *object.Bool:
			if v.V {
				return newFracInst(fracCls, fracState{num: big.NewInt(1), den: big.NewInt(1)}), nil
			}
			return newFracInst(fracCls, fracZero()), nil
		}
		return nil, object.Errorf(i.typeErr, "from_decimal() requires a Decimal or integral argument")
	}})

	// from_number(number) — 3.14+ unified constructor
	fracCls.Dict.SetStr("from_number", &object.BuiltinFunc{Name: "from_number", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		args := a
		if len(args) >= 1 {
			if _, ok := args[0].(*object.Class); ok {
				args = args[1:]
			}
		}
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "from_number() requires an argument")
		}
		f, ok := toFrac(args[0])
		if !ok {
			// Try decimal.
			if inst, ok2 := args[0].(*object.Instance); ok2 {
				if ds, ok3 := fracFromDecimalInst(inst); ok3 {
					return newFracInst(fracCls, ds), nil
				}
			}
			return nil, object.Errorf(i.typeErr, "from_number() requires a numeric argument")
		}
		return newFracInst(fracCls, f), nil
	}})

	m.Dict.SetStr("Fraction", fracCls)
	return m
}

// fracRoundHalfEven rounds a Fraction to the nearest integer (half-to-even).
func fracRoundHalfEven(f fracState) *big.Int {
	// q = floor(n/d), rem = n - q*d
	q := new(big.Int).Quo(f.num, f.den)
	r := new(big.Int).Sub(f.num, new(big.Int).Mul(q, f.den))
	// Adjust for negative quotient.
	if r.Sign() < 0 {
		q.Sub(q, big.NewInt(1))
		r.Add(r, f.den)
	}
	// 2*rem vs den.
	two_r := new(big.Int).Mul(r, big.NewInt(2))
	cmp := two_r.Cmp(f.den)
	if cmp > 0 || (cmp == 0 && new(big.Int).And(q, big.NewInt(1)).Sign() != 0) {
		q.Add(q, big.NewInt(1))
	}
	return q
}

// fracFromDecimalInst tries to build a fracState from a decimal.Decimal instance.
// Decimal instances store _s (sign), _c (coeff), _e (exp), _f (form).
func fracFromDecimalInst(inst *object.Instance) (fracState, bool) {
	sv, ok1 := inst.Dict.GetStr("_s")
	cv, ok2 := inst.Dict.GetStr("_c")
	ev, ok3 := inst.Dict.GetStr("_e")
	fv, ok4 := inst.Dict.GetStr("_f")
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return fracState{}, false
	}
	s, _ := toInt64(sv)
	e, _ := toInt64(ev)
	form, _ := toInt64(fv)
	if form != 0 {
		return fracState{}, false // Inf or NaN — not convertible
	}
	ci, ok := cv.(*object.Int)
	if !ok {
		return fracState{}, false
	}
	coeff := new(big.Int).Set(&ci.V)
	if s == 1 {
		coeff.Neg(coeff)
	}
	var num, den *big.Int
	if e >= 0 {
		scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(e), nil)
		num = new(big.Int).Mul(coeff, scale)
		den = big.NewInt(1)
	} else {
		num = coeff
		den = new(big.Int).Exp(big.NewInt(10), big.NewInt(-e), nil)
	}
	return fracReduce(num, den), true
}

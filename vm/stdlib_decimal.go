package vm

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"unicode"

	"github.com/tamnd/goipy/object"
)

// ── internal decimal representation ─────────────────────────────────────────
//
// A finite Decimal is: (-1)^sign × coeff × 10^exp
// where coeff is a non-negative *big.Int.
//
// Special forms:
//   form 0 = finite
//   form 1 = +Infinity / -Infinity (sign distinguishes)
//   form 2 = quiet NaN (payload in coeff)
//   form 3 = signaling NaN (payload in coeff)

type decState struct {
	sign  int      // 0=positive, 1=negative
	coeff *big.Int // non-negative integer coefficient
	exp   int      // exponent: value = (-1)^sign * coeff * 10^exp
	form  int      // 0=finite, 1=inf, 2=qnan, 3=snan
}

var (
	bigZero = big.NewInt(0)
	bigOne  = big.NewInt(1)
	bigTen  = big.NewInt(10)
)

func decFiniteZero() decState       { return decState{coeff: new(big.Int)} }
func decInfinity(sign int) decState { return decState{sign: sign, coeff: new(big.Int), form: 1} }
func decNaN(sign int) decState      { return decState{sign: sign, coeff: new(big.Int), form: 2} }
func decSNaN(sign int) decState     { return decState{sign: sign, coeff: new(big.Int), form: 3} }

func (d decState) isFinite() bool { return d.form == 0 }
func (d decState) isInf() bool    { return d.form == 1 }
func (d decState) isNaN() bool    { return d.form == 2 || d.form == 3 }
func (d decState) isQNaN() bool   { return d.form == 2 }
func (d decState) isSNaN() bool   { return d.form == 3 }
func (d decState) isZero() bool   { return d.form == 0 && d.coeff.Sign() == 0 }
func (d decState) isSigned() bool { return d.sign == 1 }
func (d decState) numDigits() int {
	if d.coeff.Sign() == 0 {
		return 1
	}
	return len(d.coeff.String())
}
func (d decState) adjusted() int { return d.exp + d.numDigits() - 1 }

// ── string parsing ────────────────────────────────────────────────────────────

func decParseStr(s string) (decState, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return decState{}, fmt.Errorf("invalid decimal literal: empty string")
	}
	sign := 0
	i := 0
	if i < len(s) && s[i] == '-' {
		sign = 1
		i++
	} else if i < len(s) && s[i] == '+' {
		i++
	}
	// Special values.
	upper := strings.ToUpper(s[i:])
	if upper == "INF" || upper == "INFINITY" {
		return decInfinity(sign), nil
	}
	if upper == "NAN" || upper == "QNAN" {
		return decNaN(sign), nil
	}
	if upper == "SNAN" {
		return decSNaN(sign), nil
	}
	// NaN with payload: NaN123
	if strings.HasPrefix(upper, "NAN") {
		return decNaN(sign), nil
	}
	if strings.HasPrefix(upper, "SNAN") {
		return decSNaN(sign), nil
	}

	// Parse coefficient (may contain decimal point) and optional exponent.
	expShift := 0
	eIdx := strings.IndexAny(s[i:], "eE")
	coeffStr := s[i:]
	if eIdx >= 0 {
		expStr := s[i+eIdx+1:]
		coeffStr = s[i : i+eIdx]
		ev, err := decParseInt(expStr)
		if err != nil {
			return decState{}, fmt.Errorf("invalid decimal exponent: %s", expStr)
		}
		expShift = ev
	}
	// Remove decimal point, track its position.
	dotIdx := strings.Index(coeffStr, ".")
	fracDigits := 0
	if dotIdx >= 0 {
		fracDigits = len(coeffStr) - dotIdx - 1
		coeffStr = coeffStr[:dotIdx] + coeffStr[dotIdx+1:]
	}
	// Validate digits.
	for _, c := range coeffStr {
		if !unicode.IsDigit(c) {
			return decState{}, fmt.Errorf("invalid decimal literal: %s", s)
		}
	}
	if coeffStr == "" {
		return decState{}, fmt.Errorf("invalid decimal literal: %s", s)
	}
	coeff := new(big.Int)
	coeff.SetString(coeffStr, 10)
	exp := expShift - fracDigits
	return decState{sign: sign, coeff: coeff, exp: exp, form: 0}, nil
}

func decParseInt(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	} else if s[0] == '+' {
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

// ── string formatting ─────────────────────────────────────────────────────────

func decToStr(d decState) string {
	switch d.form {
	case 1:
		if d.sign == 1 {
			return "-Infinity"
		}
		return "Infinity"
	case 2:
		prefix := ""
		if d.sign == 1 {
			prefix = "-"
		}
		if d.coeff.Sign() == 0 {
			return prefix + "NaN"
		}
		return prefix + "NaN" + d.coeff.String()
	case 3:
		prefix := ""
		if d.sign == 1 {
			prefix = "-"
		}
		if d.coeff.Sign() == 0 {
			return prefix + "sNaN"
		}
		return prefix + "sNaN" + d.coeff.String()
	}
	// Finite.
	signStr := ""
	if d.sign == 1 {
		signStr = "-"
	}
	coeffStr := d.coeff.String()
	if d.coeff.Sign() == 0 {
		coeffStr = "0"
	}
	n := len(coeffStr)
	e := d.exp

	if e > 0 {
		// GDA: exponent > 0 → always scientific notation: d.ddd...E+adj
		adj := e + n - 1
		var sb strings.Builder
		sb.WriteString(signStr)
		sb.WriteByte(coeffStr[0])
		if n > 1 {
			sb.WriteByte('.')
			sb.WriteString(coeffStr[1:])
		}
		sb.WriteString(fmt.Sprintf("E+%d", adj))
		return sb.String()
	}
	if e == 0 {
		return signStr + coeffStr
	}
	// e < 0
	dotplace := n + e // position from left where decimal point goes
	if dotplace > 0 {
		// decimal point is inside the coefficient string
		return signStr + coeffStr[:dotplace] + "." + coeffStr[dotplace:]
	} else if dotplace == 0 {
		return signStr + "0." + coeffStr
	} else {
		// dotplace < 0: need leading zeros
		adj := e + n - 1
		if adj >= -6 {
			// decimal notation with leading zeros
			return signStr + "0." + strings.Repeat("0", -dotplace) + coeffStr
		}
		// scientific notation
		var sb strings.Builder
		sb.WriteString(signStr)
		sb.WriteByte(coeffStr[0])
		if n > 1 {
			sb.WriteByte('.')
			sb.WriteString(coeffStr[1:])
		}
		sb.WriteString("E")
		if adj >= 0 {
			sb.WriteString("+")
		}
		sb.WriteString(fmt.Sprintf("%d", adj))
		return sb.String()
	}
}

func decRepr(d decState) string {
	return "Decimal('" + decToStr(d) + "')"
}

// ── rounding ─────────────────────────────────────────────────────────────────

// roundCoeff rounds coeff (with numDigits digits) to prec significant digits
// using the given rounding mode. Returns the rounded coefficient and any
// exponent adjustment (+1 if the rounding caused a carry).
func roundCoeff(coeff *big.Int, numDigits, prec int, rounding string) (*big.Int, int) {
	if numDigits <= prec || coeff.Sign() == 0 {
		return coeff, 0
	}
	drop := numDigits - prec
	if drop <= 0 {
		return coeff, 0
	}
	// pow10(drop)
	divisor := new(big.Int).Exp(bigTen, big.NewInt(int64(drop)), nil)
	quo := new(big.Int).Quo(coeff, divisor)
	rem := new(big.Int).Sub(coeff, new(big.Int).Mul(quo, divisor))

	// half = divisor / 2
	half := new(big.Int).Quo(divisor, bigTwo)

	// Determine whether to round up.
	roundUp := false
	switch rounding {
	case "ROUND_UP":
		roundUp = rem.Sign() > 0
	case "ROUND_DOWN":
		roundUp = false
	case "ROUND_CEILING":
		// positive: round up if rem > 0 (handled by sign at call site)
		roundUp = rem.Sign() > 0
	case "ROUND_FLOOR":
		roundUp = false
	case "ROUND_HALF_UP":
		roundUp = rem.Cmp(half) >= 0
	case "ROUND_HALF_DOWN":
		roundUp = rem.Cmp(half) > 0
	case "ROUND_HALF_EVEN":
		cmp := rem.Cmp(half)
		if cmp > 0 {
			roundUp = true
		} else if cmp == 0 {
			// round to even: check if quo is odd
			roundUp = new(big.Int).And(quo, bigOne).Sign() != 0
		}
	case "ROUND_05UP":
		// round up only if last digit of quo is 0 or 5
		lastDigit := new(big.Int).Mod(quo, bigTen).Int64()
		roundUp = (lastDigit == 0 || lastDigit == 5) && rem.Sign() > 0
	default: // ROUND_HALF_EVEN
		cmp := rem.Cmp(half)
		if cmp > 0 {
			roundUp = true
		} else if cmp == 0 {
			roundUp = new(big.Int).And(quo, bigOne).Sign() != 0
		}
	}

	expAdj := 0
	if roundUp {
		quo = new(big.Int).Add(quo, bigOne)
		// Check if rounding caused a carry (e.g., 9.99 → 10.0)
		if len(quo.String()) > prec {
			// One extra digit: divide by 10 and note the exponent shift.
			quo = new(big.Int).Quo(quo, bigTen)
			expAdj = 1
		}
	}
	return quo, expAdj
}

var bigTwo = big.NewInt(2)

// decTrimZeros removes trailing zeros from the coefficient (increasing exp).
func decTrimZeros(d decState) decState {
	if !d.isFinite() || d.coeff.Sign() == 0 {
		return d
	}
	c := new(big.Int).Set(d.coeff)
	exp := d.exp
	q, r := new(big.Int), new(big.Int)
	for {
		q.DivMod(c, bigTen, r)
		if r.Sign() != 0 {
			break
		}
		c.Set(q)
		exp++
	}
	return decState{sign: d.sign, coeff: c, exp: exp, form: d.form}
}

// decNormalize reduces to at most prec significant digits, adjusting exp.
func decNormalize(d decState, prec int, rounding string) decState {
	if !d.isFinite() || d.coeff.Sign() == 0 {
		return d
	}
	n := len(d.coeff.String())
	if n <= prec {
		return d
	}
	drop := n - prec
	rounded, adj := roundCoeff(d.coeff, n, prec, rounding)
	return decState{sign: d.sign, coeff: rounded, exp: d.exp + drop + adj, form: 0}
}

// ── arithmetic ────────────────────────────────────────────────────────────────

func decAdd(a, b decState, prec int, rounding string) (decState, error) {
	if a.isNaN() {
		return a, nil
	}
	if b.isNaN() {
		return b, nil
	}
	if a.isInf() && b.isInf() {
		if a.sign != b.sign {
			return decNaN(0), fmt.Errorf("InvalidOperation: inf + (-inf)")
		}
		return a, nil
	}
	if a.isInf() {
		return a, nil
	}
	if b.isInf() {
		return b, nil
	}
	// Both finite.
	// Align exponents.
	aCoeff := new(big.Int).Set(a.coeff)
	bCoeff := new(big.Int).Set(b.coeff)
	resExp := a.exp
	if a.exp > b.exp {
		// Scale a up.
		scale := new(big.Int).Exp(bigTen, big.NewInt(int64(a.exp-b.exp)), nil)
		aCoeff.Mul(aCoeff, scale)
		resExp = b.exp
	} else if b.exp > a.exp {
		scale := new(big.Int).Exp(bigTen, big.NewInt(int64(b.exp-a.exp)), nil)
		bCoeff.Mul(bCoeff, scale)
	}
	// Convert to signed.
	if a.sign == 1 {
		aCoeff.Neg(aCoeff)
	}
	if b.sign == 1 {
		bCoeff.Neg(bCoeff)
	}
	sum := new(big.Int).Add(aCoeff, bCoeff)
	resSgn := 0
	if sum.Sign() < 0 {
		resSgn = 1
		sum.Neg(sum)
	}
	res := decState{sign: resSgn, coeff: sum, exp: resExp, form: 0}
	return decNormalize(res, prec, rounding), nil
}

func decSub(a, b decState, prec int, rounding string) (decState, error) {
	neg := decState{sign: 1 - b.sign, coeff: b.coeff, exp: b.exp, form: b.form}
	if b.isNaN() {
		neg = b
	}
	return decAdd(a, neg, prec, rounding)
}

func decMul(a, b decState, prec int, rounding string) (decState, error) {
	if a.isNaN() {
		return a, nil
	}
	if b.isNaN() {
		return b, nil
	}
	resSgn := a.sign ^ b.sign
	if a.isInf() || b.isInf() {
		if a.isZero() || b.isZero() {
			return decNaN(resSgn), fmt.Errorf("InvalidOperation: 0 * inf")
		}
		return decInfinity(resSgn), nil
	}
	prod := new(big.Int).Mul(a.coeff, b.coeff)
	res := decState{sign: resSgn, coeff: prod, exp: a.exp + b.exp, form: 0}
	return decNormalize(res, prec, rounding), nil
}

func decDiv(a, b decState, prec int, rounding string) (decState, error) {
	if a.isNaN() {
		return a, nil
	}
	if b.isNaN() {
		return b, nil
	}
	resSgn := a.sign ^ b.sign
	if b.isZero() {
		if a.isZero() {
			return decNaN(resSgn), fmt.Errorf("InvalidOperation: 0/0")
		}
		return decInfinity(resSgn), fmt.Errorf("DivisionByZero")
	}
	if a.isInf() {
		if b.isInf() {
			return decNaN(resSgn), fmt.Errorf("InvalidOperation: inf/inf")
		}
		return decInfinity(resSgn), nil
	}
	if b.isInf() {
		return decFiniteZero(), nil
	}
	// Scale dividend to get prec+1 extra digits.
	extra := prec + 1
	scale := new(big.Int).Exp(bigTen, big.NewInt(int64(extra)), nil)
	dividend := new(big.Int).Mul(a.coeff, scale)
	quo, rem := new(big.Int).DivMod(dividend, b.coeff, new(big.Int))
	// Simple round-half-even for the last digit.
	if rem.Sign() != 0 {
		half := new(big.Int).Quo(b.coeff, bigTwo)
		cmp := rem.Cmp(half)
		if cmp > 0 || (cmp == 0 && new(big.Int).And(quo, bigOne).Sign() != 0) {
			quo.Add(quo, bigOne)
		}
	}
	resExp := a.exp - b.exp - extra
	res := decState{sign: resSgn, coeff: quo, exp: resExp, form: 0}
	return decNormalize(res, prec, rounding), nil
}

func decFloorDiv(a, b decState, prec int, rounding string) (decState, error) {
	if a.isNaN() {
		return a, nil
	}
	if b.isNaN() {
		return b, nil
	}
	resSgn := a.sign ^ b.sign
	if b.isZero() {
		if a.isZero() {
			return decNaN(resSgn), fmt.Errorf("InvalidOperation")
		}
		return decInfinity(resSgn), fmt.Errorf("DivisionByZero")
	}
	// Align and do integer division.
	aCoeff := new(big.Int).Set(a.coeff)
	bCoeff := new(big.Int).Set(b.coeff)
	minExp := a.exp
	if b.exp < minExp {
		minExp = b.exp
	}
	if a.exp > minExp {
		aCoeff.Mul(aCoeff, new(big.Int).Exp(bigTen, big.NewInt(int64(a.exp-minExp)), nil))
	}
	if b.exp > minExp {
		bCoeff.Mul(bCoeff, new(big.Int).Exp(bigTen, big.NewInt(int64(b.exp-minExp)), nil))
	}
	if a.sign == 1 {
		aCoeff.Neg(aCoeff)
	}
	if b.sign == 1 {
		bCoeff.Neg(bCoeff)
	}
	q := new(big.Int).Quo(aCoeff, bCoeff)
	// Floor division: round toward -infinity
	rem := new(big.Int).Sub(aCoeff, new(big.Int).Mul(q, bCoeff))
	if rem.Sign() != 0 && (aCoeff.Sign() < 0) != (bCoeff.Sign() < 0) {
		q.Sub(q, bigOne)
	}
	s := 0
	if q.Sign() < 0 {
		s = 1
		q.Neg(q)
	}
	return decState{sign: s, coeff: q, exp: 0, form: 0}, nil
}

func decMod(a, b decState, prec int, rounding string) (decState, error) {
	// Decimal uses truncated division: a % b = a - truncate(a/b)*b
	// (sign of result follows sign of dividend)
	if a.isNaN() {
		return a, nil
	}
	if b.isNaN() {
		return b, nil
	}
	if b.isZero() {
		if a.isZero() {
			return decNaN(0), fmt.Errorf("InvalidOperation")
		}
		return decNaN(0), fmt.Errorf("DivisionByZero")
	}
	// Compute truncated quotient (round toward zero), then remainder.
	aCoeff := new(big.Int).Set(a.coeff)
	bCoeff := new(big.Int).Set(b.coeff)
	minExp := a.exp
	if b.exp < minExp {
		minExp = b.exp
	}
	if a.exp > minExp {
		aCoeff.Mul(aCoeff, new(big.Int).Exp(bigTen, big.NewInt(int64(a.exp-minExp)), nil))
	}
	if b.exp > minExp {
		bCoeff.Mul(bCoeff, new(big.Int).Exp(bigTen, big.NewInt(int64(b.exp-minExp)), nil))
	}
	if a.sign == 1 {
		aCoeff.Neg(aCoeff)
	}
	if b.sign == 1 {
		bCoeff.Neg(bCoeff)
	}
	// Quo = truncated division toward zero.
	rem := new(big.Int).Mod(aCoeff, bCoeff) // adjusted below
	rem.Sub(aCoeff, new(big.Int).Mul(new(big.Int).Quo(aCoeff, bCoeff), bCoeff))
	s := 0
	if rem.Sign() < 0 {
		s = 1
		rem.Neg(rem)
	}
	return decNormalize(decState{sign: s, coeff: rem, exp: minExp, form: 0}, prec, rounding), nil
}

func decPow(a, b decState, prec int, rounding string) (decState, error) {
	if a.isNaN() {
		return a, nil
	}
	if b.isNaN() {
		return b, nil
	}
	// Only handle integer exponents for exact results.
	if b.isFinite() && b.exp >= 0 {
		n := new(big.Int).Mul(b.coeff, new(big.Int).Exp(bigTen, big.NewInt(int64(b.exp)), nil))
		if b.sign == 1 {
			// Negative integer exponent: use float.
			af, _ := decToFloat64(a)
			bf, _ := decToFloat64(b)
			r := math.Pow(af, bf)
			return decFromFloat64(r, prec)
		}
		// Positive integer power.
		base := new(big.Int).Set(a.coeff)
		result := new(big.Int).Exp(base, n, nil)
		resExp := a.exp * int(n.Int64())
		return decNormalize(decState{sign: a.sign * (int(n.Int64()) % 2), coeff: result, exp: resExp, form: 0}, prec, rounding), nil
	}
	// Fallback to float64.
	af, _ := decToFloat64(a)
	bf, _ := decToFloat64(b)
	r := math.Pow(af, bf)
	return decFromFloat64(r, prec)
}

func decToFloat64(d decState) (float64, bool) {
	if d.isInf() {
		if d.sign == 1 {
			return math.Inf(-1), true
		}
		return math.Inf(1), true
	}
	if d.isNaN() {
		return math.NaN(), true
	}
	f, _ := new(big.Float).SetPrec(256).SetInt(d.coeff).Float64()
	if d.exp > 0 {
		e, _ := new(big.Float).SetPrec(256).SetInt(new(big.Int).Exp(bigTen, big.NewInt(int64(d.exp)), nil)).Float64()
		f *= e
	} else if d.exp < 0 {
		e, _ := new(big.Float).SetPrec(256).SetInt(new(big.Int).Exp(bigTen, big.NewInt(int64(-d.exp)), nil)).Float64()
		f /= e
	}
	if d.sign == 1 {
		f = -f
	}
	return f, true
}

func decFromFloat64(f float64, prec int) (decState, error) {
	if math.IsInf(f, 1) {
		return decInfinity(0), nil
	}
	if math.IsInf(f, -1) {
		return decInfinity(1), nil
	}
	if math.IsNaN(f) {
		return decNaN(0), nil
	}
	s := fmt.Sprintf("%.*g", prec, f)
	return decParseStr(s)
}

// ── quantize & sqrt ───────────────────────────────────────────────────────────

func decQuantize(d decState, targetExp int, rounding string) (decState, error) {
	if d.isNaN() {
		return d, nil
	}
	if d.isInf() {
		return decState{}, fmt.Errorf("InvalidOperation: cannot quantize infinity")
	}
	if d.isZero() {
		return decState{sign: d.sign, coeff: new(big.Int), exp: targetExp, form: 0}, nil
	}
	diff := d.exp - targetExp
	coeff := new(big.Int).Set(d.coeff)
	if diff > 0 {
		// Need to reduce exponent: multiply coeff by 10^diff.
		coeff.Mul(coeff, new(big.Int).Exp(bigTen, big.NewInt(int64(diff)), nil))
	} else if diff < 0 {
		// Need to increase exponent: divide coeff with rounding.
		n := len(coeff.String())
		prec := n + diff // target significant digits
		if prec <= 0 {
			// All digits would be rounded away.
			coeff = new(big.Int)
		} else {
			var adj int
			coeff, adj = roundCoeff(coeff, n, prec, rounding)
			if adj > 0 {
				// carry: increase targetExp
				return decState{sign: d.sign, coeff: coeff, exp: targetExp + adj, form: 0}, nil
			}
		}
	}
	return decState{sign: d.sign, coeff: coeff, exp: targetExp, form: 0}, nil
}

func decSqrtState(d decState, prec int, rounding string) (decState, error) {
	if d.isNaN() {
		return d, nil
	}
	if d.sign == 1 && d.coeff.Sign() > 0 {
		return decNaN(0), fmt.Errorf("InvalidOperation: sqrt of negative")
	}
	if d.isInf() && d.sign == 0 {
		return decInfinity(0), nil
	}
	if d.isZero() {
		return decState{sign: 0, coeff: new(big.Int), exp: d.exp / 2, form: 0}, nil
	}
	// Use big.Float for sqrt computation.
	bitPrec := uint(prec*10 + 64)
	// value = coeff * 10^exp
	f := new(big.Float).SetPrec(bitPrec).SetInt(d.coeff)
	if d.exp != 0 {
		ep := new(big.Int).Abs(big.NewInt(int64(d.exp)))
		e10 := new(big.Float).SetPrec(bitPrec).SetInt(new(big.Int).Exp(bigTen, ep, nil))
		if d.exp > 0 {
			f.Mul(f, e10)
		} else {
			f.Quo(f, e10)
		}
	}
	r := new(big.Float).SetPrec(bitPrec).Sqrt(f)
	// Scale to get prec integer digits.
	scale := new(big.Float).SetPrec(bitPrec).SetInt(new(big.Int).Exp(bigTen, big.NewInt(int64(prec-1)), nil))
	r.Mul(r, scale)
	ri, _ := r.Int(nil)
	res := decState{sign: 0, coeff: ri, exp: -(prec - 1), form: 0}
	res = decNormalize(res, prec, rounding)
	return decTrimZeros(res), nil
}

func decExpState(d decState, prec int, rounding string) (decState, error) {
	if d.isNaN() {
		return d, nil
	}
	if d.isInf() {
		if d.sign == 1 {
			return decFiniteZero(), nil
		}
		return decInfinity(0), nil
	}
	f, _ := decToFloat64(d)
	result := math.Exp(f)
	return decFromFloat64(result, prec)
}

func decLnState(d decState, prec int, rounding string) (decState, error) {
	if d.isNaN() {
		return d, nil
	}
	if d.isInf() && d.sign == 0 {
		return decInfinity(0), nil
	}
	if d.isZero() || d.sign == 1 {
		return decNaN(0), fmt.Errorf("InvalidOperation: ln of non-positive")
	}
	f, _ := decToFloat64(d)
	r := math.Log(f)
	return decFromFloat64(r, prec)
}

func decLog10State(d decState, prec int, rounding string) (decState, error) {
	if d.isNaN() {
		return d, nil
	}
	if d.isInf() && d.sign == 0 {
		return decInfinity(0), nil
	}
	if d.isZero() || d.sign == 1 {
		return decNaN(0), fmt.Errorf("InvalidOperation: log10 of non-positive")
	}
	f, _ := decToFloat64(d)
	r := math.Log10(f)
	return decFromFloat64(r, prec)
}

// ── comparison ────────────────────────────────────────────────────────────────

// decCmp returns -1, 0, 1 for a < b, a == b, a > b.
// NaN comparisons return 2 (unordered).
func decCmp(a, b decState) int {
	if a.isNaN() || b.isNaN() {
		return 2
	}
	if a.isInf() && b.isInf() {
		if a.sign == b.sign {
			return 0
		}
		if a.sign == 0 {
			return 1
		}
		return -1
	}
	if a.isInf() {
		if a.sign == 0 {
			return 1
		}
		return -1
	}
	if b.isInf() {
		if b.sign == 0 {
			return -1
		}
		return 1
	}
	// Both finite.
	if a.isZero() && b.isZero() {
		return 0
	}
	if a.sign != b.sign {
		if a.isZero() {
			return 0
		}
		if b.isZero() {
			return 0
		}
		if a.sign == 0 {
			return 1
		}
		return -1
	}
	// Same sign: align and compare coefficients.
	aCoeff := new(big.Int).Set(a.coeff)
	bCoeff := new(big.Int).Set(b.coeff)
	if a.exp > b.exp {
		aCoeff.Mul(aCoeff, new(big.Int).Exp(bigTen, big.NewInt(int64(a.exp-b.exp)), nil))
	} else if b.exp > a.exp {
		bCoeff.Mul(bCoeff, new(big.Int).Exp(bigTen, big.NewInt(int64(b.exp-a.exp)), nil))
	}
	c := aCoeff.Cmp(bCoeff)
	if a.sign == 1 {
		return -c
	}
	return c
}

// ── Python instance helpers ──────────────────────────────────────────────────

// Decimal values are stored as *object.Instance with fields in Dict:
//   _s → Int (sign 0/1)
//   _c → Int (coefficient, big.Int)
//   _e → Int (exponent)
//   _f → Int (form: 0=finite,1=inf,2=nan,3=snan)

func newDecInst(cls *object.Class, d decState) *object.Instance {
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("_s", object.NewInt(int64(d.sign)))
	inst.Dict.SetStr("_c", object.IntFromBig(d.coeff))
	inst.Dict.SetStr("_e", object.NewInt(int64(d.exp)))
	inst.Dict.SetStr("_f", object.NewInt(int64(d.form)))
	return inst
}

func getDecState(inst *object.Instance) (decState, bool) {
	sv, ok1 := inst.Dict.GetStr("_s")
	cv, ok2 := inst.Dict.GetStr("_c")
	ev, ok3 := inst.Dict.GetStr("_e")
	fv, ok4 := inst.Dict.GetStr("_f")
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return decState{}, false
	}
	s, _ := toInt64(sv)
	e, _ := toInt64(ev)
	f, _ := toInt64(fv)
	ci, ok := cv.(*object.Int)
	if !ok {
		return decState{}, false
	}
	c := new(big.Int).Set(&ci.V)
	return decState{sign: int(s), coeff: c, exp: int(e), form: int(f)}, true
}

// toDecState coerces any Python numeric object to a decState.
func toDecState(o object.Object, cls *object.Class) (decState, bool) {
	switch v := o.(type) {
	case *object.Instance:
		if _, hasS := v.Dict.GetStr("_s"); v.Class == cls || hasS {
			ds, ok := getDecState(v)
			return ds, ok
		}
	case *object.Int:
		bi := new(big.Int).Set(&v.V)
		s := 0
		if bi.Sign() < 0 {
			s = 1
			bi.Neg(bi)
		}
		return decState{sign: s, coeff: bi, exp: 0, form: 0}, true
	case *object.Bool:
		if v.V {
			return decState{sign: 0, coeff: big.NewInt(1), exp: 0, form: 0}, true
		}
		return decFiniteZero(), true
	case *object.Float:
		if math.IsInf(v.V, 1) {
			return decInfinity(0), true
		}
		if math.IsInf(v.V, -1) {
			return decInfinity(1), true
		}
		if math.IsNaN(v.V) {
			return decNaN(0), true
		}
		// Convert float to exact decimal string.
		s := fmt.Sprintf("%.17g", v.V)
		ds, err := decParseStr(s)
		if err != nil {
			return decState{}, false
		}
		return ds, true
	}
	return decState{}, false
}

// ── context ──────────────────────────────────────────────────────────────────

type ctxState struct {
	prec     int
	rounding string
	Emin     int
	Emax     int
	capitals int
	clamp    int
}

func defaultCtx() *ctxState {
	return &ctxState{
		prec:     28,
		rounding: "ROUND_HALF_EVEN",
		Emin:     -999999999,
		Emax:     999999999,
		capitals: 1,
		clamp:    0,
	}
}

func basicCtx() *ctxState {
	return &ctxState{
		prec:     9,
		rounding: "ROUND_HALF_UP",
		Emin:     -999999999,
		Emax:     999999999,
		capitals: 1,
		clamp:    0,
	}
}

func extendedCtx() *ctxState {
	return &ctxState{
		prec:     9,
		rounding: "ROUND_HALF_EVEN",
		Emin:     -999999999,
		Emax:     999999999,
		capitals: 1,
		clamp:    0,
	}
}

func ctxToInst(ctx *ctxState, ctxCls *object.Class) *object.Instance {
	inst := &object.Instance{Class: ctxCls, Dict: object.NewDict()}
	inst.Dict.SetStr("prec", object.NewInt(int64(ctx.prec)))
	inst.Dict.SetStr("rounding", &object.Str{V: ctx.rounding})
	inst.Dict.SetStr("Emin", object.NewInt(int64(ctx.Emin)))
	inst.Dict.SetStr("Emax", object.NewInt(int64(ctx.Emax)))
	inst.Dict.SetStr("capitals", object.NewInt(int64(ctx.capitals)))
	inst.Dict.SetStr("clamp", object.NewInt(int64(ctx.clamp)))
	inst.Dict.SetStr("flags", &object.Dict{})
	inst.Dict.SetStr("traps", &object.Dict{})
	inst.Dict.SetStr("_ctx_", object.NewInt(1)) // marker
	return inst
}

func instToCtx(inst *object.Instance) *ctxState {
	ctx := defaultCtx()
	if v, ok := inst.Dict.GetStr("prec"); ok {
		if n, ok := toInt64(v); ok {
			ctx.prec = int(n)
		}
	}
	if v, ok := inst.Dict.GetStr("rounding"); ok {
		if s, ok := v.(*object.Str); ok {
			ctx.rounding = s.V
		}
	}
	if v, ok := inst.Dict.GetStr("Emin"); ok {
		if n, ok := toInt64(v); ok {
			ctx.Emin = int(n)
		}
	}
	if v, ok := inst.Dict.GetStr("Emax"); ok {
		if n, ok := toInt64(v); ok {
			ctx.Emax = int(n)
		}
	}
	return ctx
}

// ── buildDecimal ──────────────────────────────────────────────────────────────

func (i *Interp) buildDecimal() *object.Module {
	m := &object.Module{Name: "decimal", Dict: object.NewDict()}

	// ---- Rounding constants ----
	roundModes := []string{
		"ROUND_UP", "ROUND_DOWN", "ROUND_CEILING", "ROUND_FLOOR",
		"ROUND_HALF_UP", "ROUND_HALF_DOWN", "ROUND_HALF_EVEN", "ROUND_05UP",
	}
	for _, name := range roundModes {
		m.Dict.SetStr(name, &object.Str{V: name})
	}

	// ---- Exception classes ----
	makeExcClass := func(name string, parent *object.Class) *object.Class {
		cls := &object.Class{Name: name, Bases: []*object.Class{parent}, Dict: object.NewDict()}
		m.Dict.SetStr(name, cls)
		return cls
	}
	decExcClass := makeExcClass("DecimalException", i.arithErr)
	clampedCls := makeExcClass("Clamped", decExcClass)
	divByZeroCls := makeExcClass("DivisionByZero", decExcClass)
	inexactCls := makeExcClass("Inexact", decExcClass)
	invalidOpCls := makeExcClass("InvalidOperation", decExcClass)
	overflowCls := makeExcClass("Overflow", decExcClass)
	roundedCls := makeExcClass("Rounded", decExcClass)
	subnormalCls := makeExcClass("Subnormal", decExcClass)
	underflowCls := makeExcClass("Underflow", decExcClass)
	floatOpCls := makeExcClass("FloatOperation", decExcClass)
	_ = clampedCls
	_ = inexactCls
	_ = overflowCls
	_ = roundedCls
	_ = subnormalCls
	_ = underflowCls
	_ = floatOpCls

	// ---- Global context ----
	var globalCtx = defaultCtx()

	// ---- Context class ----
	ctxCls := &object.Class{Name: "Context", Dict: object.NewDict()}

	ctxCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		// Defaults.
		self.Dict.SetStr("prec", object.NewInt(28))
		self.Dict.SetStr("rounding", &object.Str{V: "ROUND_HALF_EVEN"})
		self.Dict.SetStr("Emin", object.NewInt(-999999999))
		self.Dict.SetStr("Emax", object.NewInt(999999999))
		self.Dict.SetStr("capitals", object.NewInt(1))
		self.Dict.SetStr("clamp", object.NewInt(0))
		self.Dict.SetStr("flags", object.NewDict())
		self.Dict.SetStr("traps", object.NewDict())
		self.Dict.SetStr("_ctx_", object.NewInt(1))
		// Apply kwargs.
		if kw != nil {
			for _, field := range []string{"prec", "rounding", "Emin", "Emax", "capitals", "clamp", "flags", "traps"} {
				if v, ok := kw.GetStr(field); ok {
					self.Dict.SetStr(field, v)
				}
			}
		}
		return object.None, nil
	}})

	ctxCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "Context()"}, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "Context()"}, nil
		}
		ctx := instToCtx(self)
		return &object.Str{V: fmt.Sprintf("Context(prec=%d, rounding=%s, Emin=%d, Emax=%d, capitals=%d, clamp=%d, flags=[], traps=[])",
			ctx.prec, ctx.rounding, ctx.Emin, ctx.Emax, ctx.capitals, ctx.clamp)}, nil
	}})

	ctxCls.Dict.SetStr("clear_flags", &object.BuiltinFunc{Name: "clear_flags", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	ctxCls.Dict.SetStr("clear_traps", &object.BuiltinFunc{Name: "clear_traps", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	ctxCls.Dict.SetStr("copy", &object.BuiltinFunc{Name: "copy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return ctxToInst(defaultCtx(), ctxCls), nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return ctxToInst(defaultCtx(), ctxCls), nil
		}
		return ctxToInst(instToCtx(self), ctxCls), nil
	}})

	// Property descriptors so that ctx.prec = N / ctx.rounding = R etc. update globalCtx.
	makeCtxProp := func(field string, setter func(v object.Object)) {
		getter := &object.BuiltinFunc{Name: field, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			if inst, ok := a[0].(*object.Instance); ok {
				if v, ok2 := inst.Dict.GetStr(field); ok2 {
					return v, nil
				}
			}
			return object.None, nil
		}}
		fset := &object.BuiltinFunc{Name: field + ".setter", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			if inst, ok := a[0].(*object.Instance); ok {
				inst.Dict.SetStr(field, a[1])
			}
			setter(a[1])
			return object.None, nil
		}}
		ctxCls.Dict.SetStr(field, &object.Property{Fget: getter, Fset: fset})
	}
	makeCtxProp("prec", func(v object.Object) {
		if n, ok := toInt64(v); ok {
			globalCtx.prec = int(n)
		}
	})
	makeCtxProp("rounding", func(v object.Object) {
		if s, ok := v.(*object.Str); ok {
			globalCtx.rounding = s.V
		}
	})
	makeCtxProp("Emin", func(v object.Object) {
		if n, ok := toInt64(v); ok {
			globalCtx.Emin = int(n)
		}
	})
	makeCtxProp("Emax", func(v object.Object) {
		if n, ok := toInt64(v); ok {
			globalCtx.Emax = int(n)
		}
	})

	m.Dict.SetStr("Context", ctxCls)

	// ---- Decimal class ----
	decCls := &object.Class{Name: "Decimal", Dict: object.NewDict()}

	// Helper to get decState from any object.
	getAny := func(o object.Object) (decState, bool) {
		if inst, ok := o.(*object.Instance); ok {
			return getDecState(inst)
		}
		return toDecState(o, decCls)
	}

	// Helper to get current prec/rounding from optional context arg.
	getCtx := func(args []object.Object, kwCtxIdx int) (int, string) {
		// kwCtxIdx is the index of the context positional arg if any.
		if kwCtxIdx < len(args) {
			if inst, ok := args[kwCtxIdx].(*object.Instance); ok {
				ctx := instToCtx(inst)
				return ctx.prec, ctx.rounding
			}
		}
		return globalCtx.prec, globalCtx.rounding
	}

	// __new__(cls, value='0', context=None)
	decCls.Dict.SetStr("__new__", &object.BuiltinFunc{Name: "__new__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		cls := decCls
		if len(a) >= 1 {
			if c, ok := a[0].(*object.Class); ok {
				cls = c
				a = a[1:]
			}
		}
		if len(a) == 0 {
			return newDecInst(cls, decFiniteZero()), nil
		}
		val := a[0]
		switch v := val.(type) {
		case *object.Str:
			d, err := decParseStr(v.V)
			if err != nil {
				return nil, object.Errorf(i.valueErr, "invalid literal for Decimal: '%s'", v.V)
			}
			return newDecInst(cls, d), nil
		case *object.Int:
			bi := new(big.Int).Set(&v.V)
			s := 0
			if bi.Sign() < 0 {
				s = 1
				bi.Neg(bi)
			}
			return newDecInst(cls, decState{sign: s, coeff: bi, exp: 0, form: 0}), nil
		case *object.Bool:
			if v.V {
				return newDecInst(cls, decState{sign: 0, coeff: big.NewInt(1), exp: 0, form: 0}), nil
			}
			return newDecInst(cls, decFiniteZero()), nil
		case *object.Float:
			s := fmt.Sprintf("%.17g", v.V)
			d, _ := decParseStr(s)
			return newDecInst(cls, d), nil
		case *object.Instance:
			// Copy from another Decimal.
			if ds, ok := getDecState(v); ok {
				return newDecInst(cls, ds), nil
			}
		case *object.Tuple:
			// (sign, (d0, d1, ...), exp)
			if len(v.V) != 3 {
				return nil, object.Errorf(i.valueErr, "Decimal tuple must have 3 elements")
			}
			sgn, ok1 := toInt64(v.V[0])
			digits, ok2 := v.V[1].(*object.Tuple)
			exp, ok3 := toInt64(v.V[2])
			if !ok1 || !ok2 || !ok3 {
				return nil, object.Errorf(i.valueErr, "invalid Decimal tuple")
			}
			var sb strings.Builder
			for _, dv := range digits.V {
				n, _ := toInt64(dv)
				sb.WriteByte(byte('0' + n))
			}
			coeff := new(big.Int)
			if sb.Len() > 0 {
				coeff.SetString(sb.String(), 10)
			}
			return newDecInst(cls, decState{sign: int(sgn), coeff: coeff, exp: int(exp), form: 0}), nil
		}
		return nil, object.Errorf(i.typeErr, "cannot convert %s to Decimal", object.TypeName(val))
	}})

	decCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// __str__ / __repr__
	decCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "0"}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "?"}, nil
		}
		d, ok := getDecState(inst)
		if !ok {
			return &object.Str{V: "?"}, nil
		}
		return &object.Str{V: decToStr(d)}, nil
	}})

	decCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "Decimal('0')"}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "Decimal('?')"}, nil
		}
		d, ok := getDecState(inst)
		if !ok {
			return &object.Str{V: "Decimal('?')"}, nil
		}
		return &object.Str{V: decRepr(d)}, nil
	}})

	// __format__
	decCls.Dict.SetStr("__format__", &object.BuiltinFunc{Name: "__format__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: ""}, nil
		}
		d, ok := getDecState(inst)
		if !ok {
			return &object.Str{V: ""}, nil
		}
		spec := ""
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				spec = s.V
			}
		}
		if spec == "" {
			return &object.Str{V: decToStr(d)}, nil
		}
		// Handle simple format specs like "f", ".2f", "e", ".4e"
		f, _ := decToFloat64(d)
		formatted := fmt.Sprintf("%"+spec, f)
		return &object.Str{V: formatted}, nil
	}})

	// Arithmetic binary ops.
	binOp := func(name string, fn func(a, b decState, prec int, rnd string) (decState, error)) {
		decCls.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "%s() requires 2 arguments", name)
			}
			da, ok1 := getAny(a[0])
			db, ok2 := getAny(a[1])
			if !ok1 || !ok2 {
				return object.NotImplemented, nil
			}
			prec, rnd := globalCtx.prec, globalCtx.rounding
			res, err := fn(da, db, prec, rnd)
			if err != nil {
				msg := err.Error()
				if strings.Contains(msg, "DivisionByZero") {
					return nil, object.Errorf(divByZeroCls, "division by zero")
				}
				if strings.Contains(msg, "InvalidOperation") {
					return nil, object.Errorf(invalidOpCls, "%s", msg)
				}
			}
			return newDecInst(decCls, res), nil
		}})
	}

	binOp("__add__", decAdd)
	binOp("__radd__", func(a, b decState, p int, r string) (decState, error) { return decAdd(b, a, p, r) })
	binOp("__sub__", decSub)
	binOp("__rsub__", func(a, b decState, p int, r string) (decState, error) { return decSub(b, a, p, r) })
	binOp("__mul__", decMul)
	binOp("__rmul__", func(a, b decState, p int, r string) (decState, error) { return decMul(b, a, p, r) })
	binOp("__truediv__", decDiv)
	binOp("__rtruediv__", func(a, b decState, p int, r string) (decState, error) { return decDiv(b, a, p, r) })
	binOp("__floordiv__", decFloorDiv)
	binOp("__mod__", decMod)
	binOp("__pow__", decPow)

	// __neg__, __pos__, __abs__
	decCls.Dict.SetStr("__neg__", &object.BuiltinFunc{Name: "__neg__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "__neg__ requires argument")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NotImplemented, nil
		}
		d, ok := getDecState(inst)
		if !ok {
			return object.NotImplemented, nil
		}
		return newDecInst(decCls, decState{sign: 1 - d.sign, coeff: d.coeff, exp: d.exp, form: d.form}), nil
	}})
	decCls.Dict.SetStr("__pos__", &object.BuiltinFunc{Name: "__pos__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "__pos__ requires argument")
		}
		return a[0], nil
	}})
	decCls.Dict.SetStr("__abs__", &object.BuiltinFunc{Name: "__abs__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "__abs__ requires argument")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NotImplemented, nil
		}
		d, ok := getDecState(inst)
		if !ok {
			return object.NotImplemented, nil
		}
		return newDecInst(decCls, decState{sign: 0, coeff: d.coeff, exp: d.exp, form: d.form}), nil
	}})

	// __bool__
	decCls.Dict.SetStr("__bool__", &object.BuiltinFunc{Name: "__bool__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.False, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.False, nil
		}
		d, ok := getDecState(inst)
		if !ok {
			return object.False, nil
		}
		if d.isZero() {
			return object.False, nil
		}
		return object.True, nil
	}})

	// __int__ / __float__
	decCls.Dict.SetStr("__int__", &object.BuiltinFunc{Name: "__int__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		d, ok := getDecState(inst)
		if !ok {
			return object.NewInt(0), nil
		}
		f, _ := decToFloat64(d)
		return object.NewInt(int64(f)), nil
	}})
	decCls.Dict.SetStr("__float__", &object.BuiltinFunc{Name: "__float__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Float{V: 0}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Float{V: 0}, nil
		}
		d, ok := getDecState(inst)
		if !ok {
			return &object.Float{V: 0}, nil
		}
		f, _ := decToFloat64(d)
		return &object.Float{V: f}, nil
	}})

	// __hash__
	decCls.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		d, ok := getDecState(inst)
		if !ok {
			return object.NewInt(0), nil
		}
		// Special forms: ±Inf hash to ±314159 (matching float); NaN gets
		// a stable hash by id (we use 0 for now — CPython 3.10+ would
		// raise TypeError on signaling NaN, but we permit hashing).
		if d.isNaN() {
			if d.isSNaN() {
				return nil, object.Errorf(i.typeErr, "Cannot hash a signaling NaN value.")
			}
			return object.NewInt(0), nil
		}
		if d.isInf() {
			if d.sign == 0 {
				return object.NewInt(314159), nil
			}
			return object.NewInt(-314159), nil
		}
		// Finite: value = (-1)^sign * coeff * 10^exp.
		// Compose num/den from sign, coeff, exp; let HashRational do the
		// modular reduction. Exact match for any int / float value.
		num := new(big.Int).Set(d.coeff)
		if d.sign == 1 {
			num.Neg(num)
		}
		den := big.NewInt(1)
		if d.exp >= 0 {
			ten := big.NewInt(10)
			for i := 0; i < d.exp; i++ {
				num.Mul(num, ten)
			}
		} else {
			ten := big.NewInt(10)
			for i := 0; i < -d.exp; i++ {
				den.Mul(den, ten)
			}
		}
		return object.NewInt(object.HashRational(num, den)), nil
	}})

	// Comparison ops.
	cmpOp := func(name string, want func(c int) bool) {
		decCls.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.NotImplemented, nil
			}
			da, ok1 := getAny(a[0])
			db, ok2 := getAny(a[1])
			if !ok1 || !ok2 {
				return object.NotImplemented, nil
			}
			c := decCmp(da, db)
			if c == 2 {
				if name == "__eq__" {
					return object.False, nil
				}
				if name == "__ne__" {
					return object.True, nil
				}
				return object.NotImplemented, nil
			}
			return object.BoolOf(want(c)), nil
		}})
	}
	cmpOp("__eq__", func(c int) bool { return c == 0 })
	cmpOp("__ne__", func(c int) bool { return c != 0 })
	cmpOp("__lt__", func(c int) bool { return c < 0 })
	cmpOp("__le__", func(c int) bool { return c <= 0 })
	cmpOp("__gt__", func(c int) bool { return c > 0 })
	cmpOp("__ge__", func(c int) bool { return c >= 0 })

	// compare(other) → Decimal(-1/0/1)
	decCls.Dict.SetStr("compare", &object.BuiltinFunc{Name: "compare", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "compare() takes at least 2 arguments")
		}
		da, ok1 := getAny(a[0])
		db, ok2 := getAny(a[1])
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "compare() requires Decimal arguments")
		}
		c := decCmp(da, db)
		if c == 2 {
			return newDecInst(decCls, decNaN(0)), nil
		}
		return newDecInst(decCls, decState{sign: 0, coeff: big.NewInt(int64(c)), exp: 0, form: 0}), nil
	}})

	// Predicate methods.
	pred := func(name string, fn func(d decState) bool) {
		decCls.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.False, nil
			}
			d, ok := getDecState(inst)
			if !ok {
				return object.False, nil
			}
			return object.BoolOf(fn(d)), nil
		}})
	}
	pred("is_finite", func(d decState) bool { return d.isFinite() })
	pred("is_infinite", func(d decState) bool { return d.isInf() })
	pred("is_nan", func(d decState) bool { return d.isNaN() })
	pred("is_qnan", func(d decState) bool { return d.isQNaN() })
	pred("is_snan", func(d decState) bool { return d.isSNaN() })
	pred("is_zero", func(d decState) bool { return d.isZero() })
	pred("is_signed", func(d decState) bool { return d.isSigned() })
	pred("is_canonical", func(d decState) bool { return true })
	pred("is_normal", func(d decState) bool { return d.isFinite() && !d.isZero() })
	pred("is_subnormal", func(d decState) bool { return false })

	// as_tuple()
	decCls.Dict.SetStr("as_tuple", &object.BuiltinFunc{Name: "as_tuple", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		d, ok := getDecState(inst)
		if !ok {
			return object.None, nil
		}
		signObj := object.NewInt(int64(d.sign))
		var expObj object.Object = object.NewInt(int64(d.exp))
		var digits []object.Object
		if d.isInf() {
			expObj = &object.Str{V: "F"}
			digits = []object.Object{}
		} else if d.isNaN() {
			if d.isQNaN() {
				expObj = &object.Str{V: "n"}
			} else {
				expObj = &object.Str{V: "N"}
			}
			digits = []object.Object{}
		} else {
			coeffStr := d.coeff.String()
			if d.coeff.Sign() == 0 {
				coeffStr = "0"
			}
			for _, c := range coeffStr {
				digits = append(digits, object.NewInt(int64(c-'0')))
			}
		}
		digitsTuple := &object.Tuple{V: digits}
		// Return as a namedtuple-like tuple. Use a plain tuple with named fields exposed.
		// CPython returns DecimalTuple(sign=…, digits=(…), exponent=…).
		// We return a plain tuple and also set a __class__ name via a dict approach.
		// Simplest: just return a named tuple class instance.
		ntCls := &object.Class{Name: "DecimalTuple", Dict: object.NewDict()}
		nt := &object.Instance{Class: ntCls, Dict: object.NewDict()}
		nt.Dict.SetStr("sign", signObj)
		nt.Dict.SetStr("digits", digitsTuple)
		nt.Dict.SetStr("exponent", expObj)
		// Also store as positional tuple for iteration.
		nt.Dict.SetStr("_tup_", &object.Tuple{V: []object.Object{signObj, digitsTuple, expObj}})
		ntCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "DecimalTuple()"}, nil
			}
			self := a[0].(*object.Instance)
			sv, _ := self.Dict.GetStr("sign")
			dv, _ := self.Dict.GetStr("digits")
			ev, _ := self.Dict.GetStr("exponent")
			return &object.Str{V: fmt.Sprintf("DecimalTuple(sign=%s, digits=%s, exponent=%s)",
				object.Repr(sv), object.Repr(dv), object.Repr(ev))}, nil
		}})
		ntCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.indexErr, "index out of range")
			}
			self := a[0].(*object.Instance)
			tup, _ := self.Dict.GetStr("_tup_")
			idx, _ := toInt64(a[1])
			t := tup.(*object.Tuple)
			if int(idx) < 0 || int(idx) >= len(t.V) {
				return nil, object.Errorf(i.indexErr, "index out of range")
			}
			return t.V[idx], nil
		}})
		return nt, nil
	}})

	// as_integer_ratio()
	decCls.Dict.SetStr("as_integer_ratio", &object.BuiltinFunc{Name: "as_integer_ratio", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "as_integer_ratio() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "as_integer_ratio() requires Decimal")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "as_integer_ratio() requires Decimal")
		}
		if !d.isFinite() {
			return nil, object.Errorf(i.valueErr, "cannot convert %s to integer ratio", decToStr(d))
		}
		num := new(big.Int).Set(d.coeff)
		den := big.NewInt(1)
		if d.exp >= 0 {
			num.Mul(num, new(big.Int).Exp(bigTen, big.NewInt(int64(d.exp)), nil))
		} else {
			den.Mul(den, new(big.Int).Exp(bigTen, big.NewInt(int64(-d.exp)), nil))
		}
		if d.sign == 1 {
			num.Neg(num)
		}
		// Simplify.
		g := new(big.Int).GCD(nil, nil, new(big.Int).Abs(num), den)
		num.Quo(num, g)
		den.Quo(den, g)
		return &object.Tuple{V: []object.Object{
			object.IntFromBig(num),
			object.IntFromBig(den),
		}}, nil
	}})

	// adjusted()
	decCls.Dict.SetStr("adjusted", &object.BuiltinFunc{Name: "adjusted", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		d, ok := getDecState(inst)
		if !ok {
			return object.NewInt(0), nil
		}
		return object.NewInt(int64(d.adjusted())), nil
	}})

	// quantize(exp[, rounding[, context]])
	decCls.Dict.SetStr("quantize", &object.BuiltinFunc{Name: "quantize", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "quantize() requires exp argument")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "quantize() requires Decimal self")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "quantize() requires Decimal self")
		}
		expDec, ok2 := getAny(a[1])
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "quantize() exp must be Decimal")
		}
		targetExp := expDec.exp
		prec, rnd := getCtx(a, 3)
		if kw != nil {
			if v, ok := kw.GetStr("rounding"); ok {
				if s, ok := v.(*object.Str); ok {
					rnd = s.V
				}
			}
		}
		res, err := decQuantize(d, targetExp, rnd)
		if err != nil {
			return nil, object.Errorf(invalidOpCls, "%s", err.Error())
		}
		_ = prec
		return newDecInst(decCls, res), nil
	}})

	// to_integral_value([rounding[, context]])
	decCls.Dict.SetStr("to_integral_value", &object.BuiltinFunc{Name: "to_integral_value", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "to_integral_value() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "to_integral_value() requires Decimal")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "to_integral_value() requires Decimal")
		}
		_, rnd := getCtx(a, 2)
		if kw != nil {
			if v, ok := kw.GetStr("rounding"); ok {
				if s, ok := v.(*object.Str); ok {
					rnd = s.V
				}
			}
		}
		res, err := decQuantize(d, 0, rnd)
		if err != nil {
			return nil, object.Errorf(invalidOpCls, "%s", err.Error())
		}
		return newDecInst(decCls, res), nil
	}})
	// to_integral is an alias.
	if v, ok := decCls.Dict.GetStr("to_integral_value"); ok {
		decCls.Dict.SetStr("to_integral", v)
		decCls.Dict.SetStr("to_integral_exact", v)
	}

	// normalize([context])
	decCls.Dict.SetStr("normalize", &object.BuiltinFunc{Name: "normalize", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "normalize() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "normalize() requires Decimal")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "normalize() requires Decimal")
		}
		if !d.isFinite() {
			return inst, nil
		}
		if d.isZero() {
			return newDecInst(decCls, decState{sign: d.sign, coeff: new(big.Int), exp: 0, form: 0}), nil
		}
		// Strip trailing zeros from coefficient (increasing exponent).
		c := new(big.Int).Set(d.coeff)
		e := d.exp
		rem := new(big.Int)
		for {
			q, r := new(big.Int).QuoRem(c, bigTen, rem)
			if r.Sign() != 0 {
				break
			}
			c = q
			e++
		}
		return newDecInst(decCls, decState{sign: d.sign, coeff: c, exp: e, form: 0}), nil
	}})

	// copy methods
	decCls.Dict.SetStr("copy_abs", &object.BuiltinFunc{Name: "copy_abs", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "copy_abs() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "copy_abs() requires Decimal")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "copy_abs() requires Decimal")
		}
		return newDecInst(decCls, decState{sign: 0, coeff: d.coeff, exp: d.exp, form: d.form}), nil
	}})

	decCls.Dict.SetStr("copy_negate", &object.BuiltinFunc{Name: "copy_negate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "copy_negate() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "copy_negate() requires Decimal")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "copy_negate() requires Decimal")
		}
		return newDecInst(decCls, decState{sign: 1 - d.sign, coeff: d.coeff, exp: d.exp, form: d.form}), nil
	}})

	decCls.Dict.SetStr("copy_sign", &object.BuiltinFunc{Name: "copy_sign", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "copy_sign() requires 2 arguments")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "copy_sign() requires Decimal")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "copy_sign() requires Decimal")
		}
		other, ok2 := getAny(a[1])
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "copy_sign() other must be Decimal")
		}
		return newDecInst(decCls, decState{sign: other.sign, coeff: d.coeff, exp: d.exp, form: d.form}), nil
	}})

	// sqrt([context])
	decCls.Dict.SetStr("sqrt", &object.BuiltinFunc{Name: "sqrt", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "sqrt() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "sqrt() requires Decimal")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "sqrt() requires Decimal")
		}
		prec, rnd := getCtx(a, 1)
		res, err := decSqrtState(d, prec, rnd)
		if err != nil {
			return nil, object.Errorf(invalidOpCls, "%s", err.Error())
		}
		return newDecInst(decCls, res), nil
	}})

	// exp([context])
	decCls.Dict.SetStr("exp", &object.BuiltinFunc{Name: "exp", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "exp() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "exp() requires Decimal")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "exp() requires Decimal")
		}
		prec, rnd := getCtx(a, 1)
		res, err := decExpState(d, prec, rnd)
		if err != nil {
			return nil, object.Errorf(invalidOpCls, "%s", err.Error())
		}
		return newDecInst(decCls, res), nil
	}})

	// ln([context])
	decCls.Dict.SetStr("ln", &object.BuiltinFunc{Name: "ln", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "ln() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "ln() requires Decimal")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "ln() requires Decimal")
		}
		prec, rnd := getCtx(a, 1)
		res, err := decLnState(d, prec, rnd)
		if err != nil {
			return nil, object.Errorf(invalidOpCls, "%s", err.Error())
		}
		return newDecInst(decCls, res), nil
	}})

	// log10([context])
	decCls.Dict.SetStr("log10", &object.BuiltinFunc{Name: "log10", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "log10() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "log10() requires Decimal")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "log10() requires Decimal")
		}
		prec, rnd := getCtx(a, 1)
		res, err := decLog10State(d, prec, rnd)
		if err != nil {
			return nil, object.Errorf(invalidOpCls, "%s", err.Error())
		}
		return newDecInst(decCls, res), nil
	}})

	// fma(other, third[, context]) — self * other + third
	decCls.Dict.SetStr("fma", &object.BuiltinFunc{Name: "fma", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "fma() requires 3 arguments")
		}
		da, ok1 := getAny(a[0])
		db, ok2 := getAny(a[1])
		dc, ok3 := getAny(a[2])
		if !ok1 || !ok2 || !ok3 {
			return nil, object.Errorf(i.typeErr, "fma() requires Decimal arguments")
		}
		prec, rnd := globalCtx.prec, globalCtx.rounding
		prod, err := decMul(da, db, prec*2, rnd)
		if err != nil {
			return nil, object.Errorf(invalidOpCls, "%s", err.Error())
		}
		res, err := decAdd(prod, dc, prec, rnd)
		if err != nil {
			return nil, object.Errorf(invalidOpCls, "%s", err.Error())
		}
		return newDecInst(decCls, res), nil
	}})

	// min / max
	decCls.Dict.SetStr("max", &object.BuiltinFunc{Name: "max", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "max() requires 2 arguments")
		}
		da, ok1 := getAny(a[0])
		db, ok2 := getAny(a[1])
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "max() requires Decimal arguments")
		}
		if decCmp(da, db) >= 0 {
			return a[0], nil
		}
		return a[1], nil
	}})

	decCls.Dict.SetStr("min", &object.BuiltinFunc{Name: "min", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "min() requires 2 arguments")
		}
		da, ok1 := getAny(a[0])
		db, ok2 := getAny(a[1])
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "min() requires Decimal arguments")
		}
		if decCmp(da, db) <= 0 {
			return a[0], nil
		}
		return a[1], nil
	}})

	// remainder_near
	decCls.Dict.SetStr("remainder_near", &object.BuiltinFunc{Name: "remainder_near", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "remainder_near() requires 2 arguments")
		}
		da, ok1 := getAny(a[0])
		db, ok2 := getAny(a[1])
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "remainder_near() requires Decimal arguments")
		}
		prec, rnd := globalCtx.prec, globalCtx.rounding
		af, _ := decToFloat64(da)
		bf, _ := decToFloat64(db)
		r := math.Remainder(af, bf)
		res, _ := decFromFloat64(r, prec)
		_ = rnd
		return newDecInst(decCls, res), nil
	}})

	// logb
	decCls.Dict.SetStr("logb", &object.BuiltinFunc{Name: "logb", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "logb() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "logb() requires Decimal")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "logb() requires Decimal")
		}
		return newDecInst(decCls, decState{sign: 0, coeff: big.NewInt(int64(d.adjusted())), exp: 0, form: 0}), nil
	}})

	// next_minus / next_plus
	decCls.Dict.SetStr("next_minus", &object.BuiltinFunc{Name: "next_minus", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "next_minus() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "next_minus() requires Decimal")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "next_minus() requires Decimal")
		}
		f, _ := decToFloat64(d)
		f = math.Nextafter(f, math.Inf(-1))
		res, _ := decFromFloat64(f, globalCtx.prec)
		return newDecInst(decCls, res), nil
	}})

	decCls.Dict.SetStr("next_plus", &object.BuiltinFunc{Name: "next_plus", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "next_plus() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "next_plus() requires Decimal")
		}
		d, ok := getDecState(inst)
		if !ok {
			return nil, object.Errorf(i.typeErr, "next_plus() requires Decimal")
		}
		f, _ := decToFloat64(d)
		f = math.Nextafter(f, math.Inf(1))
		res, _ := decFromFloat64(f, globalCtx.prec)
		return newDecInst(decCls, res), nil
	}})

	// same_quantum
	decCls.Dict.SetStr("same_quantum", &object.BuiltinFunc{Name: "same_quantum", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		da, ok1 := getAny(a[0])
		db, ok2 := getAny(a[1])
		if !ok1 || !ok2 {
			return object.False, nil
		}
		if da.isNaN() && db.isNaN() {
			return object.True, nil
		}
		if da.isInf() && db.isInf() {
			return object.True, nil
		}
		if da.isFinite() && db.isFinite() {
			return object.BoolOf(da.exp == db.exp), nil
		}
		return object.False, nil
	}})

	// number_class
	decCls.Dict.SetStr("number_class", &object.BuiltinFunc{Name: "number_class", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "+Zero"}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "+Zero"}, nil
		}
		d, ok := getDecState(inst)
		if !ok {
			return &object.Str{V: "+Zero"}, nil
		}
		switch {
		case d.isNaN():
			return &object.Str{V: "NaN"}, nil
		case d.isInf() && d.sign == 0:
			return &object.Str{V: "+Infinity"}, nil
		case d.isInf() && d.sign == 1:
			return &object.Str{V: "-Infinity"}, nil
		case d.isZero() && d.sign == 0:
			return &object.Str{V: "+Zero"}, nil
		case d.isZero() && d.sign == 1:
			return &object.Str{V: "-Zero"}, nil
		case d.sign == 0:
			return &object.Str{V: "+Normal"}, nil
		default:
			return &object.Str{V: "-Normal"}, nil
		}
	}})

	// canonical() — returns self (already canonical)
	decCls.Dict.SetStr("canonical", &object.BuiltinFunc{Name: "canonical", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "canonical() requires self")
		}
		return a[0], nil
	}})

	// radix() — returns Decimal(10)
	decCls.Dict.SetStr("radix", &object.BuiltinFunc{Name: "radix", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return newDecInst(decCls, decState{sign: 0, coeff: big.NewInt(10), exp: 0, form: 0}), nil
	}})

	// from_float classmethod
	decCls.Dict.SetStr("from_float", &object.BuiltinFunc{Name: "from_float", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "from_float() requires argument")
		}
		fv := a[0]
		if len(a) >= 2 {
			fv = a[1] // called as classmethod: a[0] is the class
		}
		switch v := fv.(type) {
		case *object.Float:
			// Exact decimal representation of the float.
			s := fmt.Sprintf("%.17g", v.V)
			d, err := decParseStr(s)
			if err != nil {
				return nil, object.Errorf(i.valueErr, "cannot convert %g to Decimal", v.V)
			}
			return newDecInst(decCls, d), nil
		case *object.Int:
			bi := new(big.Int).Set(&v.V)
			s := 0
			if bi.Sign() < 0 {
				s = 1
				bi.Neg(bi)
			}
			return newDecInst(decCls, decState{sign: s, coeff: bi, exp: 0, form: 0}), nil
		}
		return nil, object.Errorf(i.typeErr, "from_float() argument must be a float or int")
	}})

	// to_eng_string
	decCls.Dict.SetStr("to_eng_string", &object.BuiltinFunc{Name: "to_eng_string", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "0"}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "0"}, nil
		}
		d, ok := getDecState(inst)
		if !ok {
			return &object.Str{V: "0"}, nil
		}
		return &object.Str{V: decToEngStr(d)}, nil
	}})

	// compare_total and compare_total_mag (total ordering)
	decCls.Dict.SetStr("compare_total", &object.BuiltinFunc{Name: "compare_total", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "compare_total() requires 2 args")
		}
		da, ok1 := getAny(a[0])
		db, ok2 := getAny(a[1])
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "compare_total() requires Decimal args")
		}
		c := decCmpTotal(da, db)
		return newDecInst(decCls, decState{sign: 0, coeff: big.NewInt(int64(c)), exp: 0, form: 0}), nil
	}})

	decCls.Dict.SetStr("compare_signal", &object.BuiltinFunc{Name: "compare_signal", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "compare_signal() requires 2 args")
		}
		da, ok1 := getAny(a[0])
		db, ok2 := getAny(a[1])
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "compare_signal() requires Decimal args")
		}
		if da.isNaN() || db.isNaN() {
			return nil, object.Errorf(invalidOpCls, "compare_signal with NaN")
		}
		c := decCmp(da, db)
		return newDecInst(decCls, decState{sign: 0, coeff: big.NewInt(int64(c)), exp: 0, form: 0}), nil
	}})

	decCls.Dict.SetStr("compare_total_mag", &object.BuiltinFunc{Name: "compare_total_mag", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "compare_total_mag() requires 2 args")
		}
		da, ok1 := getAny(a[0])
		db, ok2 := getAny(a[1])
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "compare_total_mag() requires Decimal args")
		}
		da.sign = 0
		db.sign = 0
		c := decCmpTotal(da, db)
		return newDecInst(decCls, decState{sign: 0, coeff: big.NewInt(int64(c)), exp: 0, form: 0}), nil
	}})

	m.Dict.SetStr("Decimal", decCls)

	// ---- getcontext / setcontext / localcontext ----
	m.Dict.SetStr("getcontext", &object.BuiltinFunc{Name: "getcontext", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return ctxToInst(globalCtx, ctxCls), nil
	}})

	m.Dict.SetStr("setcontext", &object.BuiltinFunc{Name: "setcontext", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "setcontext() requires a Context argument")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "setcontext() requires a Context argument")
		}
		globalCtx = instToCtx(inst)
		return object.None, nil
	}})

	// localcontext: returns a context manager that saves/restores the global ctx.
	m.Dict.SetStr("localcontext", &object.BuiltinFunc{Name: "localcontext", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		saved := globalCtx
		var base *ctxState
		if len(a) >= 1 {
			if inst, ok := a[0].(*object.Instance); ok {
				base = instToCtx(inst)
			}
		}
		if base == nil {
			base = &ctxState{
				prec:     saved.prec,
				rounding: saved.rounding,
				Emin:     saved.Emin,
				Emax:     saved.Emax,
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("prec"); ok {
				if n, ok2 := toInt64(v); ok2 {
					base.prec = int(n)
				}
			}
			if v, ok := kw.GetStr("rounding"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					base.rounding = s.V
				}
			}
		}
		ctxInst := ctxToInst(base, ctxCls)
		// Return a context manager instance.
		cmCls := &object.Class{Name: "_LocalContextManager", Dict: object.NewDict()}
		cmCls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			globalCtx = base
			return ctxInst, nil
		}})
		cmCls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			globalCtx = saved
			return object.False, nil
		}})
		cm := &object.Instance{Class: cmCls, Dict: object.NewDict()}
		cm.Dict.SetStr("_ctx_", ctxInst)
		return cm, nil
	}})

	// ---- Predefined contexts ----
	m.Dict.SetStr("DefaultContext", ctxToInst(defaultCtx(), ctxCls))
	m.Dict.SetStr("BasicContext", ctxToInst(basicCtx(), ctxCls))
	m.Dict.SetStr("ExtendedContext", ctxToInst(extendedCtx(), ctxCls))

	// ---- Platform constants ----
	m.Dict.SetStr("MAX_PREC", object.NewInt(425000000))
	m.Dict.SetStr("MAX_EMAX", object.NewInt(999999999999999999))
	m.Dict.SetStr("MIN_EMIN", object.NewInt(-999999999999999999))
	m.Dict.SetStr("MIN_ETINY", object.NewInt(-1999999999999999997))
	m.Dict.SetStr("HAVE_THREADS", object.True)
	m.Dict.SetStr("HAVE_CONTEXTVAR", object.True)

	return m
}

// ── engineering string ────────────────────────────────────────────────────────

func decToEngStr(d decState) string {
	switch d.form {
	case 1:
		if d.sign == 1 {
			return "-Infinity"
		}
		return "Infinity"
	case 2, 3:
		return decToStr(d)
	}
	signStr := ""
	if d.sign == 1 {
		signStr = "-"
	}
	coeffStr := d.coeff.String()
	if d.coeff.Sign() == 0 {
		coeffStr = "0"
	}
	n := len(coeffStr)
	e := d.exp

	if e >= 0 {
		if e == 0 {
			return signStr + coeffStr
		}
		return signStr + coeffStr + "E+" + fmt.Sprintf("%d", e)
	}
	// Engineering notation: exponent is multiple of 3.
	adj := e + n - 1
	// Adjust adj to be a multiple of 3 (round down).
	engExp := (adj / 3) * 3
	if adj < 0 && adj%3 != 0 {
		engExp -= 3
	}
	leftDigits := adj - engExp + 1
	var sb strings.Builder
	sb.WriteString(signStr)
	if leftDigits >= n {
		sb.WriteString(coeffStr)
		sb.WriteString(strings.Repeat("0", leftDigits-n))
	} else {
		sb.WriteString(coeffStr[:leftDigits])
		sb.WriteByte('.')
		sb.WriteString(coeffStr[leftDigits:])
	}
	if engExp != 0 {
		if engExp > 0 {
			fmt.Fprintf(&sb, "E+%d", engExp)
		} else {
			fmt.Fprintf(&sb, "E%d", engExp)
		}
	}
	return sb.String()
}

// ── total ordering ────────────────────────────────────────────────────────────

func decCmpTotal(a, b decState) int {
	// NaN handling: sNaN > qNaN > finite > -Inf; signs matter.
	aRank := formRank(a)
	bRank := formRank(b)
	if a.sign != b.sign {
		if a.sign == 0 {
			return 1
		}
		return -1
	}
	if aRank != bRank {
		r := 0
		if aRank > bRank {
			r = 1
		} else {
			r = -1
		}
		if a.sign == 1 {
			return -r
		}
		return r
	}
	// Same form.
	if a.form != 0 {
		return 0
	}
	// Both finite: first numeric comparison, then exponent tie-break.
	c := decCmp(a, b)
	if c != 0 {
		return c
	}
	// Numerically equal: higher exponent wins in total ordering.
	if a.exp > b.exp {
		if a.sign == 0 {
			return 1
		}
		return -1
	}
	if a.exp < b.exp {
		if a.sign == 0 {
			return -1
		}
		return 1
	}
	return 0
}

func formRank(d decState) int {
	switch d.form {
	case 0: // finite
		if d.isInf() {
			if d.sign == 0 {
				return 3
			}
			return -3
		}
		return 0
	case 1:
		if d.sign == 0 {
			return 4
		}
		return -4
	case 2:
		return 5
	case 3:
		return 6
	}
	return 0
}

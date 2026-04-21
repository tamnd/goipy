package vm

import (
	"math"
	"math/big"
	"strings"

	"github.com/tamnd/goipy/object"
	"github.com/tamnd/goipy/op"
)

// binaryOp executes a BINARY_OP with oparg = NB_*.
func (i *Interp) binaryOp(a, b object.Object, nb uint32) (object.Object, error) {
	switch nb {
	case op.NB_ADD:
		return i.add(a, b)
	case op.NB_INPLACE_ADD:
		return i.inplaceOp(a, b, "__iadd__", (*Interp).add)
	case op.NB_SUBTRACT:
		return i.sub(a, b)
	case op.NB_INPLACE_SUBTRACT:
		return i.inplaceOp(a, b, "__isub__", (*Interp).sub)
	case op.NB_MULTIPLY:
		return i.mul(a, b)
	case op.NB_INPLACE_MULTIPLY:
		return i.inplaceOp(a, b, "__imul__", (*Interp).mul)
	case op.NB_TRUE_DIVIDE:
		return i.truediv(a, b)
	case op.NB_INPLACE_TRUE_DIVIDE:
		return i.inplaceOp(a, b, "__itruediv__", (*Interp).truediv)
	case op.NB_FLOOR_DIVIDE:
		return i.floordiv(a, b)
	case op.NB_INPLACE_FLOOR_DIVIDE:
		return i.inplaceOp(a, b, "__ifloordiv__", (*Interp).floordiv)
	case op.NB_REMAINDER:
		return i.mod(a, b)
	case op.NB_INPLACE_REMAINDER:
		return i.inplaceOp(a, b, "__imod__", (*Interp).mod)
	case op.NB_POWER:
		return i.pow(a, b)
	case op.NB_INPLACE_POWER:
		return i.inplaceOp(a, b, "__ipow__", (*Interp).pow)
	case op.NB_MATRIX_MULTIPLY:
		return i.matmul(a, b)
	case op.NB_INPLACE_MATRIX_MULTIPLY:
		return i.inplaceOp(a, b, "__imatmul__", (*Interp).matmul)
	case op.NB_LSHIFT:
		return i.shift(a, b, true)
	case op.NB_INPLACE_LSHIFT:
		return i.inplaceOp(a, b, "__ilshift__", func(ii *Interp, x, y object.Object) (object.Object, error) { return ii.shift(x, y, true) })
	case op.NB_RSHIFT:
		return i.shift(a, b, false)
	case op.NB_INPLACE_RSHIFT:
		return i.inplaceOp(a, b, "__irshift__", func(ii *Interp, x, y object.Object) (object.Object, error) { return ii.shift(x, y, false) })
	case op.NB_AND:
		return i.bitop(a, b, "&")
	case op.NB_INPLACE_AND:
		return i.inplaceOp(a, b, "__iand__", func(ii *Interp, x, y object.Object) (object.Object, error) { return ii.bitop(x, y, "&") })
	case op.NB_OR:
		return i.bitop(a, b, "|")
	case op.NB_INPLACE_OR:
		return i.inplaceOp(a, b, "__ior__", func(ii *Interp, x, y object.Object) (object.Object, error) { return ii.bitop(x, y, "|") })
	case op.NB_XOR:
		return i.bitop(a, b, "^")
	case op.NB_INPLACE_XOR:
		return i.inplaceOp(a, b, "__ixor__", func(ii *Interp, x, y object.Object) (object.Object, error) { return ii.bitop(x, y, "^") })
	case op.NB_SUBSCR:
		return i.getitem(a, b)
	}
	return nil, object.Errorf(i.typeErr, "unsupported BINARY_OP %d", nb)
}

// inplaceOp tries __iop__ on an instance; if not defined or returns
// NotImplemented, falls back to the regular op.
func (i *Interp) inplaceOp(a, b object.Object, name string, fallback func(*Interp, object.Object, object.Object) (object.Object, error)) (object.Object, error) {
	if inst, ok := a.(*object.Instance); ok {
		if r, ok, err := i.callInstanceDunder(inst, name, b); ok {
			if err != nil {
				return nil, err
			}
			if !isNotImplemented(r) {
				return r, nil
			}
		}
	}
	return fallback(i, a, b)
}

// matmul dispatches @ to __matmul__/__rmatmul__. There is no builtin type
// backing @, so without a dunder it's a TypeError.
func (i *Interp) matmul(a, b object.Object) (object.Object, error) {
	if hasInstance(a, b) {
		if r, ok, err := i.tryBinaryDunder(a, b, "__matmul__", "__rmatmul__"); ok {
			return r, err
		}
	}
	return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for @: '%s' and '%s'", object.TypeName(a), object.TypeName(b))
}

// --- arithmetic ---

// asComplex coerces a numeric scalar to a Complex value. Returns ok=false
// for non-numeric objects.
func asComplex(o object.Object) (re, im float64, ok bool) {
	switch v := o.(type) {
	case *object.Bool:
		if v.V {
			return 1, 0, true
		}
		return 0, 0, true
	case *object.Int:
		f, _ := new(big.Float).SetInt(v.V).Float64()
		return f, 0, true
	case *object.Float:
		return v.V, 0, true
	case *object.Complex:
		return v.Real, v.Imag, true
	}
	return 0, 0, false
}

// complexResult returns a Complex unless the imaginary part is exactly
// zero and neither input was already a Complex — in which case callers
// prefer the plain int/float result.
func isComplex(o object.Object) bool {
	_, ok := o.(*object.Complex)
	return ok
}

func asIntOrFloat(o object.Object) (ibig *big.Int, f float64, isFloat bool, ok bool) {
	switch v := o.(type) {
	case *object.Bool:
		if v.V {
			return big.NewInt(1), 0, false, true
		}
		return big.NewInt(0), 0, false, true
	case *object.Int:
		return v.V, 0, false, true
	case *object.Float:
		return nil, v.V, true, true
	}
	return nil, 0, false, false
}

func toFloat(ibig *big.Int, f float64, isFloat bool) float64 {
	if isFloat {
		return f
	}
	x, _ := new(big.Float).SetInt(ibig).Float64()
	return x
}

// bytesBytesOrArray returns the raw byte slice of o and true if it's a bytes
// or bytearray.
func bytesBytesOrArray(o object.Object) ([]byte, bool) {
	switch v := o.(type) {
	case *object.Bytes:
		return v.V, true
	case *object.Bytearray:
		return v.V, true
	}
	return nil, false
}

func (i *Interp) add(a, b object.Object) (object.Object, error) {
	if hasInstance(a, b) {
		if r, ok, err := i.tryBinaryDunder(a, b, "__add__", "__radd__"); ok {
			return r, err
		}
	}
	// str + str
	if sa, ok := a.(*object.Str); ok {
		if sb, ok := b.(*object.Str); ok {
			return &object.Str{V: sa.V + sb.V}, nil
		}
	}
	// list + list
	if la, ok := a.(*object.List); ok {
		if lb, ok := b.(*object.List); ok {
			out := make([]object.Object, 0, len(la.V)+len(lb.V))
			out = append(out, la.V...)
			out = append(out, lb.V...)
			return &object.List{V: out}, nil
		}
	}
	// tuple + tuple
	if ta, ok := a.(*object.Tuple); ok {
		if tb, ok := b.(*object.Tuple); ok {
			out := make([]object.Object, 0, len(ta.V)+len(tb.V))
			out = append(out, ta.V...)
			out = append(out, tb.V...)
			return &object.Tuple{V: out}, nil
		}
	}
	// bytes/bytearray concatenation. Result type follows the left operand:
	// bytes + bytes → bytes; bytearray + bytes → bytearray; bytes + bytearray → bytes.
	if ab, ok := a.(*object.Bytes); ok {
		if bb, ok := bytesBytesOrArray(b); ok {
			out := make([]byte, 0, len(ab.V)+len(bb))
			out = append(out, ab.V...)
			out = append(out, bb...)
			return &object.Bytes{V: out}, nil
		}
	}
	if aba, ok := a.(*object.Bytearray); ok {
		if bb, ok := bytesBytesOrArray(b); ok {
			out := make([]byte, 0, len(aba.V)+len(bb))
			out = append(out, aba.V...)
			out = append(out, bb...)
			return &object.Bytearray{V: out}, nil
		}
	}
	if isComplex(a) || isComplex(b) {
		ar, ai2, ok1 := asComplex(a)
		br, bi2, ok2 := asComplex(b)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for +: '%s' and '%s'", object.TypeName(a), object.TypeName(b))
		}
		return &object.Complex{Real: ar + br, Imag: ai2 + bi2}, nil
	}
	ai, af, aF, aok := asIntOrFloat(a)
	bi, bf, bF, bok := asIntOrFloat(b)
	if !aok || !bok {
		return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for +: '%s' and '%s'", object.TypeName(a), object.TypeName(b))
	}
	if aF || bF {
		return &object.Float{V: toFloat(ai, af, aF) + toFloat(bi, bf, bF)}, nil
	}
	r := new(big.Int).Add(ai, bi)
	return &object.Int{V: r}, nil
}

func (i *Interp) sub(a, b object.Object) (object.Object, error) {
	if hasInstance(a, b) {
		if r, ok, err := i.tryBinaryDunder(a, b, "__sub__", "__rsub__"); ok {
			return r, err
		}
	}
	if isSetLike(a) && isSetLike(b) {
		return setBitop(a, b, "-"), nil
	}
	if isComplex(a) || isComplex(b) {
		ar, ai2, ok1 := asComplex(a)
		br, bi2, ok2 := asComplex(b)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for -: '%s' and '%s'", object.TypeName(a), object.TypeName(b))
		}
		return &object.Complex{Real: ar - br, Imag: ai2 - bi2}, nil
	}
	ai, af, aF, aok := asIntOrFloat(a)
	bi, bf, bF, bok := asIntOrFloat(b)
	if !aok || !bok {
		return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for -: '%s' and '%s'", object.TypeName(a), object.TypeName(b))
	}
	if aF || bF {
		return &object.Float{V: toFloat(ai, af, aF) - toFloat(bi, bf, bF)}, nil
	}
	return &object.Int{V: new(big.Int).Sub(ai, bi)}, nil
}

func (i *Interp) mul(a, b object.Object) (object.Object, error) {
	if hasInstance(a, b) {
		if r, ok, err := i.tryBinaryDunder(a, b, "__mul__", "__rmul__"); ok {
			return r, err
		}
	}
	// str * int
	if sa, ok := a.(*object.Str); ok {
		if n, ok := toInt64(b); ok {
			if n < 0 {
				n = 0
			}
			return &object.Str{V: strings.Repeat(sa.V, int(n))}, nil
		}
	}
	if sb, ok := b.(*object.Str); ok {
		if n, ok := toInt64(a); ok {
			if n < 0 {
				n = 0
			}
			return &object.Str{V: strings.Repeat(sb.V, int(n))}, nil
		}
	}
	// list * int
	if la, ok := a.(*object.List); ok {
		if n, ok := toInt64(b); ok {
			return &object.List{V: repeatSlice(la.V, int(n))}, nil
		}
	}
	if lb, ok := b.(*object.List); ok {
		if n, ok := toInt64(a); ok {
			return &object.List{V: repeatSlice(lb.V, int(n))}, nil
		}
	}
	// tuple * int
	if ta, ok := a.(*object.Tuple); ok {
		if n, ok := toInt64(b); ok {
			return &object.Tuple{V: repeatSlice(ta.V, int(n))}, nil
		}
	}
	if isComplex(a) || isComplex(b) {
		ar, ai2, ok1 := asComplex(a)
		br, bi2, ok2 := asComplex(b)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for *: '%s' and '%s'", object.TypeName(a), object.TypeName(b))
		}
		return &object.Complex{Real: ar*br - ai2*bi2, Imag: ar*bi2 + ai2*br}, nil
	}
	ai, af, aF, aok := asIntOrFloat(a)
	bi, bf, bF, bok := asIntOrFloat(b)
	if !aok || !bok {
		return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for *: '%s' and '%s'", object.TypeName(a), object.TypeName(b))
	}
	if aF || bF {
		return &object.Float{V: toFloat(ai, af, aF) * toFloat(bi, bf, bF)}, nil
	}
	return &object.Int{V: new(big.Int).Mul(ai, bi)}, nil
}

func (i *Interp) truediv(a, b object.Object) (object.Object, error) {
	if hasInstance(a, b) {
		if r, ok, err := i.tryBinaryDunder(a, b, "__truediv__", "__rtruediv__"); ok {
			return r, err
		}
	}
	if isComplex(a) || isComplex(b) {
		ar, ai2, ok1 := asComplex(a)
		br, bi2, ok2 := asComplex(b)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for /")
		}
		denom := br*br + bi2*bi2
		if denom == 0 {
			return nil, object.Errorf(i.zeroDivErr, "complex division by zero")
		}
		return &object.Complex{
			Real: (ar*br + ai2*bi2) / denom,
			Imag: (ai2*br - ar*bi2) / denom,
		}, nil
	}
	ai, af, aF, aok := asIntOrFloat(a)
	bi, bf, bF, bok := asIntOrFloat(b)
	if !aok || !bok {
		return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for /")
	}
	fa, fb := toFloat(ai, af, aF), toFloat(bi, bf, bF)
	if fb == 0 {
		return nil, object.Errorf(i.zeroDivErr, "division by zero")
	}
	return &object.Float{V: fa / fb}, nil
}

func (i *Interp) floordiv(a, b object.Object) (object.Object, error) {
	if hasInstance(a, b) {
		if r, ok, err := i.tryBinaryDunder(a, b, "__floordiv__", "__rfloordiv__"); ok {
			return r, err
		}
	}
	ai, af, aF, aok := asIntOrFloat(a)
	bi, bf, bF, bok := asIntOrFloat(b)
	if !aok || !bok {
		return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for //")
	}
	if aF || bF {
		fa, fb := toFloat(ai, af, aF), toFloat(bi, bf, bF)
		if fb == 0 {
			return nil, object.Errorf(i.zeroDivErr, "float floor division by zero")
		}
		return &object.Float{V: math.Floor(fa / fb)}, nil
	}
	if bi.Sign() == 0 {
		return nil, object.Errorf(i.zeroDivErr, "integer division or modulo by zero")
	}
	q, _ := new(big.Int).QuoRem(ai, bi, new(big.Int))
	// Python floor division: adjust toward negative infinity
	r := new(big.Int).Sub(ai, new(big.Int).Mul(q, bi))
	if r.Sign() != 0 && (r.Sign() != bi.Sign()) {
		q.Sub(q, big.NewInt(1))
	}
	return &object.Int{V: q}, nil
}

func (i *Interp) mod(a, b object.Object) (object.Object, error) {
	if hasInstance(a, b) {
		if r, ok, err := i.tryBinaryDunder(a, b, "__mod__", "__rmod__"); ok {
			return r, err
		}
	}
	// str % tuple-ish — skip; not implementing printf-style formatting.
	ai, af, aF, aok := asIntOrFloat(a)
	bi, bf, bF, bok := asIntOrFloat(b)
	if !aok || !bok {
		return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for %%")
	}
	if aF || bF {
		fa, fb := toFloat(ai, af, aF), toFloat(bi, bf, bF)
		if fb == 0 {
			return nil, object.Errorf(i.zeroDivErr, "float modulo")
		}
		return &object.Float{V: fa - math.Floor(fa/fb)*fb}, nil
	}
	if bi.Sign() == 0 {
		return nil, object.Errorf(i.zeroDivErr, "integer division or modulo by zero")
	}
	r := new(big.Int).Mod(ai, bi)
	if r.Sign() != 0 && (r.Sign() != bi.Sign()) {
		r.Add(r, bi)
	}
	return &object.Int{V: r}, nil
}

func (i *Interp) pow(a, b object.Object) (object.Object, error) {
	if hasInstance(a, b) {
		if r, ok, err := i.tryBinaryDunder(a, b, "__pow__", "__rpow__"); ok {
			return r, err
		}
	}
	if isComplex(a) || isComplex(b) {
		ar, ai2, ok1 := asComplex(a)
		br, bi2, ok2 := asComplex(b)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for **")
		}
		r, im := complexPow(ar, ai2, br, bi2)
		return &object.Complex{Real: r, Imag: im}, nil
	}
	ai, af, aF, aok := asIntOrFloat(a)
	bi, bf, bF, bok := asIntOrFloat(b)
	if !aok || !bok {
		return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for **")
	}
	if aF || bF || (bi != nil && bi.Sign() < 0) {
		return &object.Float{V: math.Pow(toFloat(ai, af, aF), toFloat(bi, bf, bF))}, nil
	}
	return &object.Int{V: new(big.Int).Exp(ai, bi, nil)}, nil
}

// complexPow is a|b for complex numbers using the standard polar
// formulation. 0**0 == 1+0j to match CPython.
func complexPow(ar, ai, br, bi float64) (float64, float64) {
	if br == 0 && bi == 0 {
		return 1, 0
	}
	if ar == 0 && ai == 0 {
		return 0, 0
	}
	r := math.Hypot(ar, ai)
	theta := math.Atan2(ai, ar)
	logR := math.Log(r)
	// exponent applied in log-polar space
	lnRe := br*logR - bi*theta
	lnIm := br*theta + bi*logR
	mag := math.Exp(lnRe)
	return mag * math.Cos(lnIm), mag * math.Sin(lnIm)
}

func (i *Interp) shift(a, b object.Object, left bool) (object.Object, error) {
	if hasInstance(a, b) {
		fwd, rev := "__lshift__", "__rlshift__"
		if !left {
			fwd, rev = "__rshift__", "__rrshift__"
		}
		if r, ok, err := i.tryBinaryDunder(a, b, fwd, rev); ok {
			return r, err
		}
	}
	n, okA := toBigInt(a)
	k, okB := toBigInt(b)
	if !okA || !okB {
		return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for shift")
	}
	if !k.IsInt64() || k.Int64() < 0 {
		return nil, object.Errorf(i.valueErr, "negative shift count")
	}
	nk := uint(k.Int64())
	r := new(big.Int)
	if left {
		r.Lsh(n, nk)
	} else {
		r.Rsh(n, nk)
	}
	return &object.Int{V: r}, nil
}

// hasInstance reports whether either operand is a user class instance.
func hasInstance(a, b object.Object) bool {
	_, aok := a.(*object.Instance)
	_, bok := b.(*object.Instance)
	return aok || bok
}

func bitopDunderNames(kind string) (fwd, rev string) {
	switch kind {
	case "&":
		return "__and__", "__rand__"
	case "|":
		return "__or__", "__ror__"
	case "^":
		return "__xor__", "__rxor__"
	}
	return "", ""
}

// isSetLike reports whether o is a set or frozenset.
func isSetLike(o object.Object) bool {
	switch o.(type) {
	case *object.Set, *object.Frozenset:
		return true
	}
	return false
}

func setItems(o object.Object) []object.Object {
	switch s := o.(type) {
	case *object.Set:
		return s.Items()
	case *object.Frozenset:
		return s.Items()
	}
	return nil
}

func setContains(o, x object.Object) bool {
	switch s := o.(type) {
	case *object.Set:
		c, _ := s.Contains(x)
		return c
	case *object.Frozenset:
		c, _ := s.Contains(x)
		return c
	}
	return false
}

// setBitop computes |, &, ^, - between two set-like operands. Result type
// follows the left operand (CPython semantics).
func setBitop(a, b object.Object, kind string) object.Object {
	_, frozen := a.(*object.Frozenset)
	var addS func(object.Object)
	var result object.Object
	if frozen {
		s := object.NewFrozenset()
		addS = func(x object.Object) { _ = s.Add(x) }
		result = s
	} else {
		s := object.NewSet()
		addS = func(x object.Object) { _ = s.Add(x) }
		result = s
	}
	aItems := setItems(a)
	bItems := setItems(b)
	switch kind {
	case "|":
		for _, x := range aItems {
			addS(x)
		}
		for _, x := range bItems {
			addS(x)
		}
	case "&":
		for _, x := range aItems {
			if setContains(b, x) {
				addS(x)
			}
		}
	case "^":
		for _, x := range aItems {
			if !setContains(b, x) {
				addS(x)
			}
		}
		for _, x := range bItems {
			if !setContains(a, x) {
				addS(x)
			}
		}
	case "-":
		for _, x := range aItems {
			if !setContains(b, x) {
				addS(x)
			}
		}
	}
	return result
}

func (i *Interp) bitop(a, b object.Object, kind string) (object.Object, error) {
	if hasInstance(a, b) {
		fwd, rev := bitopDunderNames(kind)
		if r, ok, err := i.tryBinaryDunder(a, b, fwd, rev); ok {
			return r, err
		}
	}
	if isSetLike(a) && isSetLike(b) {
		return setBitop(a, b, kind), nil
	}
	ni, okA := toBigInt(a)
	nj, okB := toBigInt(b)
	if !okA || !okB {
		return nil, object.Errorf(i.typeErr, "unsupported operand type(s) for %s", kind)
	}
	r := new(big.Int)
	switch kind {
	case "&":
		r.And(ni, nj)
	case "|":
		r.Or(ni, nj)
	case "^":
		r.Xor(ni, nj)
	}
	return &object.Int{V: r}, nil
}

// --- subscript / slicing ---

func (i *Interp) getitem(container, key object.Object) (object.Object, error) {
	if inst, ok := container.(*object.Instance); ok {
		if r, ok, err := i.callInstanceDunder(inst, "__getitem__", key); ok {
			return r, err
		}
	}
	switch c := container.(type) {
	case *object.List:
		return i.seqGetitem(c.V, key, "list")
	case *object.Tuple:
		return i.seqGetitem(c.V, key, "tuple")
	case *object.Str:
		return i.strGetitem(c, key)
	case *object.Bytes:
		return i.bytesGetitem(c.V, key, false)
	case *object.Bytearray:
		return i.bytesGetitem(c.V, key, true)
	case *object.Memoryview:
		return i.memoryviewGetitem(c, key)
	case *object.Dict:
		v, ok, err := c.Get(key)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, object.Errorf(i.keyErr, "%s", object.Repr(key))
		}
		return v, nil
	case *object.Range:
		n, ok := toInt64(key)
		if !ok {
			return nil, object.Errorf(i.typeErr, "range indices must be integers")
		}
		length := rangeLen(c)
		if n < 0 {
			n += length
		}
		if n < 0 || n >= length {
			return nil, object.Errorf(i.indexErr, "range object index out of range")
		}
		return object.NewInt(c.Start + n*c.Step), nil
	}
	return nil, object.Errorf(i.typeErr, "'%s' object is not subscriptable", object.TypeName(container))
}

func (i *Interp) seqGetitem(seq []object.Object, key object.Object, name string) (object.Object, error) {
	if sl, ok := key.(*object.Slice); ok {
		start, stop, step, err := i.resolveSlice(sl, len(seq))
		if err != nil {
			return nil, err
		}
		out := sliceSeq(seq, start, stop, step)
		if name == "tuple" {
			return &object.Tuple{V: out}, nil
		}
		return &object.List{V: out}, nil
	}
	n, ok := toInt64(key)
	if !ok {
		return nil, object.Errorf(i.typeErr, "%s indices must be integers", name)
	}
	L := int64(len(seq))
	if n < 0 {
		n += L
	}
	if n < 0 || n >= L {
		return nil, object.Errorf(i.indexErr, "%s index out of range", name)
	}
	return seq[n], nil
}

func (i *Interp) strGetitem(s *object.Str, key object.Object) (object.Object, error) {
	rs := s.Runes()
	L := int64(len(rs))
	if sl, ok := key.(*object.Slice); ok {
		start, stop, step, err := i.resolveSlice(sl, int(L))
		if err != nil {
			return nil, err
		}
		return &object.Str{V: string(sliceRunes(rs, start, stop, step))}, nil
	}
	n, ok := toInt64(key)
	if !ok {
		return nil, object.Errorf(i.typeErr, "string indices must be integers")
	}
	if n < 0 {
		n += L
	}
	if n < 0 || n >= L {
		return nil, object.Errorf(i.indexErr, "string index out of range")
	}
	return &object.Str{V: string(rs[n])}, nil
}

func (i *Interp) bytesGetitem(data []byte, key object.Object, mutable bool) (object.Object, error) {
	if sl, ok := key.(*object.Slice); ok {
		start, stop, step, err := i.resolveSlice(sl, len(data))
		if err != nil {
			return nil, err
		}
		out := []byte{}
		if step > 0 {
			for j := start; j < stop; j += step {
				out = append(out, data[j])
			}
		} else if step < 0 {
			for j := start; j > stop; j += step {
				out = append(out, data[j])
			}
		}
		if mutable {
			return &object.Bytearray{V: out}, nil
		}
		return &object.Bytes{V: out}, nil
	}
	n, ok := toInt64(key)
	if !ok {
		return nil, object.Errorf(i.typeErr, "byte indices must be integers")
	}
	L := int64(len(data))
	if n < 0 {
		n += L
	}
	if n < 0 || n >= L {
		return nil, object.Errorf(i.indexErr, "index out of range")
	}
	return object.NewInt(int64(data[n])), nil
}

// memoryviewGetitem supports integer indexing (returns int) and contiguous
// slicing (returns a memoryview sharing the backing buffer). Non-unit step
// slices are rejected to keep the view semantics trivial.
func (i *Interp) memoryviewGetitem(m *object.Memoryview, key object.Object) (object.Object, error) {
	buf := m.Buf()
	L := len(buf)
	if sl, ok := key.(*object.Slice); ok {
		start, stop, step, err := i.resolveSlice(sl, L)
		if err != nil {
			return nil, err
		}
		if step != 1 {
			return nil, object.Errorf(i.valueErr, "memoryview extended slicing not supported")
		}
		return &object.Memoryview{
			Backing:  m.Backing,
			Start:    m.Start + start,
			Stop:     m.Start + stop,
			Readonly: m.Readonly,
		}, nil
	}
	n, ok := toInt64(key)
	if !ok {
		return nil, object.Errorf(i.typeErr, "memoryview indices must be integers")
	}
	if n < 0 {
		n += int64(L)
	}
	if n < 0 || n >= int64(L) {
		return nil, object.Errorf(i.indexErr, "memoryview index out of range")
	}
	return object.NewInt(int64(buf[n])), nil
}

func (i *Interp) resolveSlice(s *object.Slice, length int) (start, stop, step int, err error) {
	step = 1
	if s.Step != nil {
		if n, ok := toInt64(s.Step); ok {
			step = int(n)
		} else if _, isNone := s.Step.(*object.NoneType); !isNone {
			return 0, 0, 0, object.Errorf(i.typeErr, "slice indices must be integers")
		}
	}
	if step == 0 {
		return 0, 0, 0, object.Errorf(i.valueErr, "slice step cannot be zero")
	}
	// start
	if s.Start == nil {
		if step > 0 {
			start = 0
		} else {
			start = length - 1
		}
	} else if _, isNone := s.Start.(*object.NoneType); isNone {
		if step > 0 {
			start = 0
		} else {
			start = length - 1
		}
	} else if n, ok := toInt64(s.Start); ok {
		start = int(n)
		if start < 0 {
			start += length
		}
		if step > 0 {
			if start < 0 {
				start = 0
			}
			if start > length {
				start = length
			}
		} else {
			if start < -1 {
				start = -1
			}
			if start > length-1 {
				start = length - 1
			}
		}
	}
	// stop
	if s.Stop == nil {
		if step > 0 {
			stop = length
		} else {
			stop = -1
		}
	} else if _, isNone := s.Stop.(*object.NoneType); isNone {
		if step > 0 {
			stop = length
		} else {
			stop = -1
		}
	} else if n, ok := toInt64(s.Stop); ok {
		stop = int(n)
		if stop < 0 {
			stop += length
		}
		if step > 0 {
			if stop < 0 {
				stop = 0
			}
			if stop > length {
				stop = length
			}
		} else {
			if stop < -1 {
				stop = -1
			}
			if stop > length-1 {
				stop = length - 1
			}
		}
	}
	return
}

func sliceSeq(s []object.Object, start, stop, step int) []object.Object {
	var out []object.Object
	if step > 0 {
		for j := start; j < stop; j += step {
			out = append(out, s[j])
		}
	} else {
		for j := start; j > stop; j += step {
			out = append(out, s[j])
		}
	}
	return out
}

func sliceRunes(s []rune, start, stop, step int) []rune {
	var out []rune
	if step > 0 {
		for j := start; j < stop; j += step {
			out = append(out, s[j])
		}
	} else {
		for j := start; j > stop; j += step {
			out = append(out, s[j])
		}
	}
	return out
}

// toInt64 converts small Int/Bool to int64; returns ok=false otherwise.
func toInt64(o object.Object) (int64, bool) {
	switch v := o.(type) {
	case *object.Bool:
		if v.V {
			return 1, true
		}
		return 0, true
	case *object.Int:
		if v.V.IsInt64() {
			return v.V.Int64(), true
		}
	}
	return 0, false
}

// toBigInt converts Int/Bool to *big.Int.
func toBigInt(o object.Object) (*big.Int, bool) {
	switch v := o.(type) {
	case *object.Bool:
		if v.V {
			return big.NewInt(1), true
		}
		return big.NewInt(0), true
	case *object.Int:
		return v.V, true
	}
	return nil, false
}

// rangeLen returns the number of elements in a range.
func rangeLen(r *object.Range) int64 {
	if r.Step > 0 && r.Stop > r.Start {
		return (r.Stop - r.Start + r.Step - 1) / r.Step
	}
	if r.Step < 0 && r.Stop < r.Start {
		return (r.Start - r.Stop - r.Step - 1) / (-r.Step)
	}
	return 0
}

func repeatSlice(s []object.Object, n int) []object.Object {
	if n <= 0 {
		return nil
	}
	out := make([]object.Object, 0, len(s)*n)
	for k := 0; k < n; k++ {
		out = append(out, s...)
	}
	return out
}

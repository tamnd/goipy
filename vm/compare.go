package vm

import (
	"math/big"

	"github.com/tamnd/goipy/object"
)

// Comparison indices match the 3.14 COMPARE_OP oparg>>5 encoding.
const (
	cmpLT = 0
	cmpLE = 1
	cmpEQ = 2
	cmpNE = 3
	cmpGT = 4
	cmpGE = 5
)

func (i *Interp) compare(a, b object.Object, kind int) (object.Object, error) {
	if kind == cmpEQ {
		eq, err := object.Eq(a, b)
		if err != nil {
			return nil, err
		}
		return object.BoolOf(eq), nil
	}
	if kind == cmpNE {
		eq, err := object.Eq(a, b)
		if err != nil {
			return nil, err
		}
		return object.BoolOf(!eq), nil
	}
	// Sets use a partial order (subset) rather than the total-order path.
	if isSetLike(a) && isSetLike(b) {
		return setOrder(a, b, kind, i)
	}
	lt, err := i.lt(a, b)
	if err != nil {
		return nil, err
	}
	switch kind {
	case cmpLT:
		return object.BoolOf(lt), nil
	case cmpGE:
		return object.BoolOf(!lt), nil
	case cmpGT:
		// a > b == b < a
		gt, err := i.lt(b, a)
		if err != nil {
			return nil, err
		}
		return object.BoolOf(gt), nil
	case cmpLE:
		gt, err := i.lt(b, a)
		if err != nil {
			return nil, err
		}
		return object.BoolOf(!gt), nil
	}
	return nil, object.Errorf(i.typeErr, "bad compare op %d", kind)
}

// setOrder implements <, <=, >, >= for set/frozenset as subset relations.
func setOrder(a, b object.Object, kind int, i *Interp) (object.Object, error) {
	aLen := len(setItems(a))
	bLen := len(setItems(b))
	subset := func(x, y object.Object) bool {
		for _, e := range setItems(x) {
			if !setContains(y, e) {
				return false
			}
		}
		return true
	}
	switch kind {
	case cmpLT:
		return object.BoolOf(aLen < bLen && subset(a, b)), nil
	case cmpLE:
		return object.BoolOf(aLen <= bLen && subset(a, b)), nil
	case cmpGT:
		return object.BoolOf(aLen > bLen && subset(b, a)), nil
	case cmpGE:
		return object.BoolOf(aLen >= bLen && subset(b, a)), nil
	}
	return nil, object.Errorf(i.typeErr, "bad set compare op %d", kind)
}

func (i *Interp) lt(a, b object.Object) (bool, error) {
	// Both numeric?
	if ai, af, aF, aok := asIntOrFloat(a); aok {
		if bi, bf, bF, bok := asIntOrFloat(b); bok {
			if aF || bF {
				return toFloat(ai, af, aF) < toFloat(bi, bf, bF), nil
			}
			return ai.Cmp(bi) < 0, nil
		}
	}
	// Strings
	if as, ok := a.(*object.Str); ok {
		if bs, ok := b.(*object.Str); ok {
			return as.V < bs.V, nil
		}
	}
	// Bytes / bytearray (any mix).
	if ab, ok := bytesBytesOrArray(a); ok {
		if bb, ok := bytesBytesOrArray(b); ok {
			return string(ab) < string(bb), nil
		}
	}
	// Sequences lex compare
	if al, ok := a.(*object.List); ok {
		if bl, ok := b.(*object.List); ok {
			return i.seqLess(al.V, bl.V)
		}
	}
	if at, ok := a.(*object.Tuple); ok {
		if bt, ok := b.(*object.Tuple); ok {
			return i.seqLess(at.V, bt.V)
		}
	}
	return false, object.Errorf(i.typeErr, "'<' not supported between '%s' and '%s'", object.TypeName(a), object.TypeName(b))
}

func (i *Interp) seqLess(a, b []object.Object) (bool, error) {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for k := 0; k < n; k++ {
		eq, err := object.Eq(a[k], b[k])
		if err != nil {
			return false, err
		}
		if eq {
			continue
		}
		return i.lt(a[k], b[k])
	}
	return len(a) < len(b), nil
}

// containsOp implements `in` / `not in`.
func (i *Interp) containsOp(container, needle object.Object, invert bool) (object.Object, error) {
	found, err := i.contains(container, needle)
	if err != nil {
		return nil, err
	}
	if invert {
		found = !found
	}
	return object.BoolOf(found), nil
}

func bytesContains(data []byte, needle object.Object) (bool, error) {
	switch n := needle.(type) {
	case *object.Bytes:
		return bytesHasSub(data, n.V), nil
	case *object.Bytearray:
		return bytesHasSub(data, n.V), nil
	}
	if n, ok := toInt64(needle); ok {
		if n < 0 || n > 255 {
			return false, nil
		}
		b := byte(n)
		for _, x := range data {
			if x == b {
				return true, nil
			}
		}
		return false, nil
	}
	return false, nil
}

func bytesHasSub(hay, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		match := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func (i *Interp) contains(container, needle object.Object) (bool, error) {
	switch c := container.(type) {
	case *object.List:
		for _, x := range c.V {
			eq, err := object.Eq(x, needle)
			if err != nil {
				return false, err
			}
			if eq {
				return true, nil
			}
		}
		return false, nil
	case *object.Tuple:
		for _, x := range c.V {
			eq, err := object.Eq(x, needle)
			if err != nil {
				return false, err
			}
			if eq {
				return true, nil
			}
		}
		return false, nil
	case *object.Str:
		if ns, ok := needle.(*object.Str); ok {
			return containsStr(c.V, ns.V), nil
		}
		return false, object.Errorf(i.typeErr, "'in <string>' requires string as left operand")
	case *object.Bytes:
		return bytesContains(c.V, needle)
	case *object.Bytearray:
		return bytesContains(c.V, needle)
	case *object.Memoryview:
		return bytesContains(c.Buf(), needle)
	case *object.Dict:
		_, ok, err := c.Get(needle)
		return ok, err
	case *object.Set:
		return c.Contains(needle)
	case *object.Frozenset:
		return c.Contains(needle)
	case *object.Range:
		n, ok := toBigInt(needle)
		if !ok {
			return false, nil
		}
		// Check n in range
		start := big.NewInt(c.Start)
		stop := big.NewInt(c.Stop)
		step := big.NewInt(c.Step)
		if c.Step > 0 {
			if n.Cmp(start) < 0 || n.Cmp(stop) >= 0 {
				return false, nil
			}
		} else {
			if n.Cmp(start) > 0 || n.Cmp(stop) <= 0 {
				return false, nil
			}
		}
		diff := new(big.Int).Sub(n, start)
		rem := new(big.Int).Mod(diff, step)
		return rem.Sign() == 0, nil
	}
	return false, object.Errorf(i.typeErr, "argument of type '%s' is not iterable", object.TypeName(container))
}

func containsStr(hay, needle string) bool {
	if needle == "" {
		return true
	}
	return stringIndex(hay, needle) >= 0
}

func stringIndex(hay, needle string) int {
	// simple substring search
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		if hay[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

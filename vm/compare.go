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

// compareEqNe handles == and != which don't need ordering.
func (i *Interp) compareEqNe(a, b object.Object, kind int) (object.Object, bool, error) {
	if kind != cmpEQ && kind != cmpNE {
		return nil, false, nil
	}
	eq, err := object.Eq(a, b)
	if err != nil {
		return nil, true, err
	}
	return object.BoolOf(eq == (kind == cmpEQ)), true, nil
}

// compareOrdered handles <, <=, >, >= after EQ/NE are ruled out.
func (i *Interp) compareOrdered(a, b object.Object, kind int) (object.Object, error) {
	if isSetLike(a) && isSetLike(b) {
		return setOrder(a, b, kind, i)
	}
	lt, err := i.lt(a, b)
	if err != nil {
		return nil, err
	}
	if kind == cmpLT {
		return object.BoolOf(lt), nil
	}
	if kind == cmpGE {
		return object.BoolOf(!lt), nil
	}
	gt, err := i.lt(b, a)
	if err != nil {
		return nil, err
	}
	if kind == cmpGT {
		return object.BoolOf(gt), nil
	}
	if kind == cmpLE {
		return object.BoolOf(!gt), nil
	}
	return nil, object.Errorf(i.typeErr, "bad compare op %d", kind)
}

func (i *Interp) compare(a, b object.Object, kind int) (object.Object, error) {
	if _, ok := a.(*object.Instance); ok {
		if r, ok, err := i.tryCompareDunder(a, b, kind); ok {
			return object.BoolOf(object.Truthy(r)), err
		}
	} else if _, ok := b.(*object.Instance); ok {
		if r, ok, err := i.tryCompareDunder(a, b, kind); ok {
			return object.BoolOf(object.Truthy(r)), err
		}
	}
	a, b = unboxBuiltin(a), unboxBuiltin(b)
	if r, handled, err := i.compareEqNe(a, b, kind); handled {
		return r, err
	}
	return i.compareOrdered(a, b, kind)
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

// ltDunder tries the __lt__ dunder on instances. Returns (result, handled, error).
func (i *Interp) ltDunder(a, b object.Object) (bool, bool, error) {
	if _, ok := a.(*object.Instance); ok {
		if r, ok2, err := i.tryCompareDunder(a, b, cmpLT); ok2 {
			return object.Truthy(r), true, err
		}
	} else if _, ok := b.(*object.Instance); ok {
		if r, ok2, err := i.tryCompareDunder(a, b, cmpLT); ok2 {
			return object.Truthy(r), true, err
		}
	}
	return false, false, nil
}

func (i *Interp) lt(a, b object.Object) (bool, error) {
	if r, ok, err := i.ltDunder(a, b); ok {
		return r, err
	}
	a, b = unboxBuiltin(a), unboxBuiltin(b)
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
	if inst, ok := container.(*object.Instance); ok {
		if r, ok, err := i.callInstanceDunder(inst, "__contains__", needle); ok {
			if err != nil {
				return false, err
			}
			return object.Truthy(r), nil
		}
		// Fall back to iterating via __iter__ / __getitem__.
		if it, ok, err := i.instanceIter(inst); ok {
			if err != nil {
				return false, err
			}
			for {
				x, ok, err := it.Next()
				if err != nil {
					return false, err
				}
				if !ok {
					return false, nil
				}
				eq, err := object.Eq(x, needle)
				if err != nil {
					return false, err
				}
				if eq {
					return true, nil
				}
			}
		}
		// Builtin-subclass instance with no __contains__/__iter__ override:
		// route through the underlying builtin payload.
		if inst.BuiltinValue != nil {
			return i.contains(inst.BuiltinValue, needle)
		}
	}
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
	case *object.Counter:
		_, ok, err := c.D.Get(needle)
		return ok, err
	case *object.DefaultDict:
		_, ok, err := c.D.Get(needle)
		return ok, err
	case *object.OrderedDict:
		_, ok, err := c.D.Get(needle)
		return ok, err
	case *object.PyArray:
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
	case *object.Deque:
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
	case *object.Class:
		// Enum class containment: `Color.RED in Color` checks if needle is a member.
		if c.EnumData != nil {
			if inst, ok := needle.(*object.Instance); ok {
				return object.IsSubclass(inst.Class, c), nil
			}
			return false, nil
		}
	case *object.SectionProxyObj:
		if ks, ok := needle.(*object.Str); ok {
			return cfgHasOption(c.Parser, c.Section, ks.V), nil
		}
		return false, nil
	case *object.ConfigParserObj:
		if ks, ok := needle.(*object.Str); ok {
			if ks.V == c.DefaultSection {
				return true, nil
			}
			return cfgHasSection(c, ks.V), nil
		}
		return false, nil
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

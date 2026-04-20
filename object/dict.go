package object

import (
	"fmt"
	"math/big"
)

// Set inserts or replaces a value for key in the dict.
func (d *Dict) Set(key, val Object) error {
	if s, ok := key.(*Str); ok {
		if i, ok := d.index[s.V]; ok {
			d.vals[i] = val
			return nil
		}
		d.index[s.V] = len(d.keys)
		d.keys = append(d.keys, key)
		d.vals = append(d.vals, val)
		return nil
	}
	h, err := Hash(key)
	if err != nil {
		return err
	}
	for _, idx := range d.oHash[h] {
		eq, err := Eq(d.keys[idx], key)
		if err != nil {
			return err
		}
		if eq {
			d.vals[idx] = val
			return nil
		}
	}
	d.oHash[h] = append(d.oHash[h], len(d.keys))
	d.keys = append(d.keys, key)
	d.vals = append(d.vals, val)
	return nil
}

// Get returns the value for key (or nil, false).
func (d *Dict) Get(key Object) (Object, bool, error) {
	if s, ok := key.(*Str); ok {
		if i, ok := d.index[s.V]; ok {
			return d.vals[i], true, nil
		}
		return nil, false, nil
	}
	h, err := Hash(key)
	if err != nil {
		return nil, false, err
	}
	for _, idx := range d.oHash[h] {
		eq, err := Eq(d.keys[idx], key)
		if err != nil {
			return nil, false, err
		}
		if eq {
			return d.vals[idx], true, nil
		}
	}
	return nil, false, nil
}

// GetStr is a fast-path for string keys.
func (d *Dict) GetStr(key string) (Object, bool) {
	if i, ok := d.index[key]; ok {
		return d.vals[i], true
	}
	return nil, false
}

// SetStr stores a value under a string key.
func (d *Dict) SetStr(key string, val Object) {
	if i, ok := d.index[key]; ok {
		d.vals[i] = val
		return
	}
	d.index[key] = len(d.keys)
	d.keys = append(d.keys, &Str{V: key})
	d.vals = append(d.vals, val)
}

// Delete removes key. Returns true if a key was removed.
func (d *Dict) Delete(key Object) (bool, error) {
	idx := -1
	if s, ok := key.(*Str); ok {
		i, ok := d.index[s.V]
		if !ok {
			return false, nil
		}
		idx = i
		delete(d.index, s.V)
	} else {
		h, err := Hash(key)
		if err != nil {
			return false, err
		}
		bucket := d.oHash[h]
		found := -1
		for bi, j := range bucket {
			eq, err := Eq(d.keys[j], key)
			if err != nil {
				return false, err
			}
			if eq {
				idx = j
				found = bi
				break
			}
		}
		if idx == -1 {
			return false, nil
		}
		d.oHash[h] = append(bucket[:found], bucket[found+1:]...)
	}
	// Remove key/val and reindex everything after idx
	d.keys = append(d.keys[:idx], d.keys[idx+1:]...)
	d.vals = append(d.vals[:idx], d.vals[idx+1:]...)
	// Shift indices
	for k, i := range d.index {
		if i > idx {
			d.index[k] = i - 1
		}
	}
	for h, bucket := range d.oHash {
		for bi, j := range bucket {
			if j > idx {
				d.oHash[h][bi] = j - 1
			}
		}
	}
	return true, nil
}

// --- equality and hashing ---

// Eq reports whether a == b in Python semantics (used for dict/set lookups).
func Eq(a, b Object) (bool, error) {
	if a == b {
		return true, nil
	}
	// None
	if _, ok := a.(*NoneType); ok {
		_, ok2 := b.(*NoneType)
		return ok2, nil
	}
	if _, ok := b.(*NoneType); ok {
		return false, nil
	}
	// Bool compares with int/float via numeric path.
	switch av := a.(type) {
	case *Bool:
		ai := boolToInt(av)
		return Eq(ai, b)
	case *Int:
		switch bv := b.(type) {
		case *Bool:
			return Eq(a, boolToInt(bv))
		case *Int:
			return av.V.Cmp(bv.V) == 0, nil
		case *Float:
			return bigIntEqFloat(av.V, bv.V), nil
		}
	case *Float:
		switch bv := b.(type) {
		case *Bool:
			return av.V == float64(btoi(bv.V)), nil
		case *Int:
			return bigIntEqFloat(bv.V, av.V), nil
		case *Float:
			return av.V == bv.V, nil
		case *Complex:
			return bv.Imag == 0 && av.V == bv.Real, nil
		}
	case *Complex:
		switch bv := b.(type) {
		case *Bool:
			return av.Imag == 0 && av.Real == float64(btoi(bv.V)), nil
		case *Int:
			return av.Imag == 0 && bigIntEqFloat(bv.V, av.Real), nil
		case *Float:
			return av.Imag == 0 && av.Real == bv.V, nil
		case *Complex:
			return av.Real == bv.Real && av.Imag == bv.Imag, nil
		}
	case *Str:
		if bv, ok := b.(*Str); ok {
			return av.V == bv.V, nil
		}
	case *Bytes:
		switch bv := b.(type) {
		case *Bytes:
			return string(av.V) == string(bv.V), nil
		case *Bytearray:
			return string(av.V) == string(bv.V), nil
		case *Memoryview:
			return string(av.V) == string(bv.Bytes()), nil
		}
	case *Bytearray:
		switch bv := b.(type) {
		case *Bytes:
			return string(av.V) == string(bv.V), nil
		case *Bytearray:
			return string(av.V) == string(bv.V), nil
		case *Memoryview:
			return string(av.V) == string(bv.Bytes()), nil
		}
	case *Memoryview:
		aBytes := av.Bytes()
		switch bv := b.(type) {
		case *Bytes:
			return string(aBytes) == string(bv.V), nil
		case *Bytearray:
			return string(aBytes) == string(bv.V), nil
		case *Memoryview:
			return string(aBytes) == string(bv.Bytes()), nil
		}
	case *Tuple:
		bv, ok := b.(*Tuple)
		if !ok {
			return false, nil
		}
		return seqEq(av.V, bv.V)
	case *List:
		bv, ok := b.(*List)
		if !ok {
			return false, nil
		}
		return seqEq(av.V, bv.V)
	case *Set:
		return setEq(av.items, b)
	case *Frozenset:
		return setEq(av.items, b)
	}
	return false, nil
}

// setEq is true when every element of a's items is in b (a Set or Frozenset)
// and sizes match. CPython treats set == frozenset by element equality.
func setEq(aItems []Object, b Object) (bool, error) {
	var bLen int
	var bContains func(Object) (bool, error)
	switch bv := b.(type) {
	case *Set:
		bLen = bv.Len()
		bContains = bv.Contains
	case *Frozenset:
		bLen = bv.Len()
		bContains = bv.Contains
	default:
		return false, nil
	}
	if len(aItems) != bLen {
		return false, nil
	}
	for _, x := range aItems {
		ok, err := bContains(x)
		if err != nil || !ok {
			return ok, err
		}
	}
	return true, nil
}

func seqEq(a, b []Object) (bool, error) {
	if len(a) != len(b) {
		return false, nil
	}
	for i := range a {
		eq, err := Eq(a[i], b[i])
		if err != nil || !eq {
			return eq, err
		}
	}
	return true, nil
}

// Hash returns a 64-bit hash of a hashable object.
func Hash(o Object) (uint64, error) {
	switch v := o.(type) {
	case *NoneType:
		return 0xdeadbeef, nil
	case *Bool:
		if v.V {
			return 1, nil
		}
		return 0, nil
	case *Int:
		return v.V.Uint64() ^ uint64(v.V.Sign())*0x9e3779b97f4a7c15, nil
	case *Float:
		return uint64(v.V * 1e6), nil
	case *Complex:
		return uint64(v.Real*1e6) ^ (uint64(v.Imag*1e6) * 0x9e3779b97f4a7c15), nil
	case *Str:
		return stringHash(v.V), nil
	case *Bytes:
		return stringHash(string(v.V)), nil
	case *Tuple:
		var h uint64 = 0x811c9dc5
		for _, x := range v.V {
			xh, err := Hash(x)
			if err != nil {
				return 0, err
			}
			h = (h * 16777619) ^ xh
		}
		return h, nil
	case *Frozenset:
		// Commutative combiner so hash is order-independent.
		var h uint64 = 0x9e3779b97f4a7c15
		for _, x := range v.items {
			xh, err := Hash(x)
			if err != nil {
				return 0, err
			}
			h ^= xh*0x100000001b3 + 0x9e3779b97f4a7c15
		}
		return h, nil
	}
	return 0, fmt.Errorf("TypeError: unhashable type: '%s'", TypeName(o))
}

func stringHash(s string) uint64 {
	var h uint64 = 0xcbf29ce484222325
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 0x100000001b3
	}
	return h
}

func boolToInt(b *Bool) *Int {
	if b.V {
		return NewInt(1)
	}
	return NewInt(0)
}

func btoi(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func bigIntEqFloat(b *big.Int, f float64) bool {
	if f != f { // NaN
		return false
	}
	if f == float64(int64(f)) {
		return b.Cmp(big.NewInt(int64(f))) == 0
	}
	return false
}

package vm

import (
	"sort"

	"github.com/tamnd/goipy/object"
)

// --- array methods ---

func arrayMethod(i *Interp, arr *object.PyArray, name string) (object.Object, bool) {
	switch name {
	case "typecode":
		return &object.Str{V: arr.Typecode}, true
	case "itemsize":
		return object.NewInt(int64(object.ArrayItemSize(arr.Typecode))), true
	case "append":
		return &object.BuiltinFunc{Name: "append", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "append() takes exactly one argument")
			}
			v, err := arrayValidate(arr.Typecode, a[0])
			if err != nil {
				return nil, object.Errorf(i.typeErr, "%s", err.Error())
			}
			arr.V = append(arr.V, v)
			return object.None, nil
		}}, true
	case "extend":
		return &object.BuiltinFunc{Name: "extend", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "extend() takes exactly one argument")
			}
			items, err := iterate(i, a[0])
			if err != nil {
				return nil, err
			}
			for _, x := range items {
				v, err := arrayValidate(arr.Typecode, x)
				if err != nil {
					return nil, object.Errorf(i.typeErr, "%s", err.Error())
				}
				arr.V = append(arr.V, v)
			}
			return object.None, nil
		}}, true
	case "fromlist":
		return &object.BuiltinFunc{Name: "fromlist", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "fromlist() takes exactly one argument")
			}
			lst, ok := a[0].(*object.List)
			if !ok {
				return nil, object.Errorf(i.typeErr, "fromlist() requires a list")
			}
			for _, x := range lst.V {
				v, err := arrayValidate(arr.Typecode, x)
				if err != nil {
					return nil, object.Errorf(i.typeErr, "%s", err.Error())
				}
				arr.V = append(arr.V, v)
			}
			return object.None, nil
		}}, true
	case "frombytes":
		return &object.BuiltinFunc{Name: "frombytes", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "frombytes() takes exactly one argument")
			}
			var raw []byte
			switch v := a[0].(type) {
			case *object.Bytes:
				raw = v.V
			case *object.Bytearray:
				raw = v.V
			default:
				return nil, object.Errorf(i.typeErr, "a bytes-like object is required")
			}
			return object.None, arrayFromBytes(i, arr, raw)
		}}, true
	case "tobytes":
		return &object.BuiltinFunc{Name: "tobytes", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			sz := object.ArrayItemSize(arr.Typecode)
			out := make([]byte, 0, len(arr.V)*sz)
			for _, v := range arr.V {
				out = append(out, arrayItemBytes(arr.Typecode, v)...)
			}
			return &object.Bytes{V: out}, nil
		}}, true
	case "tolist":
		return &object.BuiltinFunc{Name: "tolist", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			out := make([]object.Object, len(arr.V))
			copy(out, arr.V)
			return &object.List{V: out}, nil
		}}, true
	case "insert":
		return &object.BuiltinFunc{Name: "insert", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 2 {
				return nil, object.Errorf(i.typeErr, "insert() takes exactly 2 arguments")
			}
			idx, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "integer argument required")
			}
			v, err := arrayValidate(arr.Typecode, a[1])
			if err != nil {
				return nil, object.Errorf(i.typeErr, "%s", err.Error())
			}
			n := int(idx)
			if n < 0 {
				n += len(arr.V)
				if n < 0 {
					n = 0
				}
			}
			if n > len(arr.V) {
				n = len(arr.V)
			}
			arr.V = append(arr.V, nil)
			copy(arr.V[n+1:], arr.V[n:])
			arr.V[n] = v
			return object.None, nil
		}}, true
	case "pop":
		return &object.BuiltinFunc{Name: "pop", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(arr.V) == 0 {
				return nil, object.Errorf(i.indexErr, "pop from empty array")
			}
			idx := int64(len(arr.V) - 1)
			if len(a) >= 1 {
				n, ok := toInt64(a[0])
				if !ok {
					return nil, object.Errorf(i.typeErr, "integer argument required")
				}
				idx = n
			}
			L := int64(len(arr.V))
			if idx < 0 {
				idx += L
			}
			if idx < 0 || idx >= L {
				return nil, object.Errorf(i.indexErr, "pop index out of range")
			}
			v := arr.V[idx]
			arr.V = append(arr.V[:idx], arr.V[idx+1:]...)
			return v, nil
		}}, true
	case "remove":
		return &object.BuiltinFunc{Name: "remove", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "remove() takes exactly one argument")
			}
			for k, x := range arr.V {
				eq, err := object.Eq(x, a[0])
				if err != nil {
					return nil, err
				}
				if eq {
					arr.V = append(arr.V[:k], arr.V[k+1:]...)
					return object.None, nil
				}
			}
			return nil, object.Errorf(i.valueErr, "array.remove(x): x not in array")
		}}, true
	case "count":
		return &object.BuiltinFunc{Name: "count", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "count() takes exactly one argument")
			}
			cnt := int64(0)
			for _, x := range arr.V {
				eq, err := object.Eq(x, a[0])
				if err != nil {
					return nil, err
				}
				if eq {
					cnt++
				}
			}
			return object.NewInt(cnt), nil
		}}, true
	case "index":
		return &object.BuiltinFunc{Name: "index", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "index() takes at least 1 argument")
			}
			needle := a[0]
			start, stop := 0, len(arr.V)
			if len(a) >= 2 {
				n, ok := toInt64(a[1])
				if !ok {
					return nil, object.Errorf(i.typeErr, "integer argument required")
				}
				start = int(n)
				if start < 0 {
					start += len(arr.V)
					if start < 0 {
						start = 0
					}
				}
			}
			if len(a) >= 3 {
				n, ok := toInt64(a[2])
				if !ok {
					return nil, object.Errorf(i.typeErr, "integer argument required")
				}
				stop = int(n)
				if stop < 0 {
					stop += len(arr.V)
				}
			}
			for k := start; k < stop && k < len(arr.V); k++ {
				eq, err := object.Eq(arr.V[k], needle)
				if err != nil {
					return nil, err
				}
				if eq {
					return object.NewInt(int64(k)), nil
				}
			}
			return nil, object.Errorf(i.valueErr, "%s is not in array", object.Repr(needle))
		}}, true
	case "reverse":
		return &object.BuiltinFunc{Name: "reverse", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			for lo, hi := 0, len(arr.V)-1; lo < hi; lo, hi = lo+1, hi-1 {
				arr.V[lo], arr.V[hi] = arr.V[hi], arr.V[lo]
			}
			return object.None, nil
		}}, true
	case "byteswap":
		return &object.BuiltinFunc{Name: "byteswap", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			sz := object.ArrayItemSize(arr.Typecode)
			for k, v := range arr.V {
				raw := arrayItemBytes(arr.Typecode, v)
				// reverse the bytes
				for lo, hi := 0, sz-1; lo < hi; lo, hi = lo+1, hi-1 {
					raw[lo], raw[hi] = raw[hi], raw[lo]
				}
				// re-parse
				tmp := &object.PyArray{Typecode: arr.Typecode}
				if err := arrayFromBytes(i, tmp, raw); err != nil {
					return nil, err
				}
				if len(tmp.V) > 0 {
					arr.V[k] = tmp.V[0]
				}
			}
			return object.None, nil
		}}, true
	case "buffer_info":
		return &object.BuiltinFunc{Name: "buffer_info", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			// Return (0, length) — no real memory address in a VM.
			return &object.Tuple{V: []object.Object{
				object.NewInt(0),
				object.NewInt(int64(len(arr.V))),
			}}, nil
		}}, true
	}
	return nil, false
}

// --- deque methods ---

func dequeMethod(i *Interp, dq *object.Deque, name string) (object.Object, bool) {
	truncate := func() {
		if dq.MaxLen < 0 {
			return
		}
		for len(dq.V) > dq.MaxLen {
			dq.V = dq.V[1:]
		}
	}
	truncateLeft := func() {
		if dq.MaxLen < 0 {
			return
		}
		for len(dq.V) > dq.MaxLen {
			dq.V = dq.V[:len(dq.V)-1]
		}
	}
	switch name {
	case "append":
		return &object.BuiltinFunc{Name: "append", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "append() takes 1 argument")
			}
			dq.V = append(dq.V, a[0])
			truncate()
			return object.None, nil
		}}, true
	case "appendleft":
		return &object.BuiltinFunc{Name: "appendleft", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "appendleft() takes 1 argument")
			}
			dq.V = append([]object.Object{a[0]}, dq.V...)
			truncateLeft()
			return object.None, nil
		}}, true
	case "pop":
		return &object.BuiltinFunc{Name: "pop", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if len(dq.V) == 0 {
				return nil, object.Errorf(i.indexErr, "pop from an empty deque")
			}
			n := len(dq.V)
			v := dq.V[n-1]
			dq.V = dq.V[:n-1]
			return v, nil
		}}, true
	case "popleft":
		return &object.BuiltinFunc{Name: "popleft", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if len(dq.V) == 0 {
				return nil, object.Errorf(i.indexErr, "pop from an empty deque")
			}
			v := dq.V[0]
			dq.V = dq.V[1:]
			return v, nil
		}}, true
	case "extend":
		return &object.BuiltinFunc{Name: "extend", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "extend() takes 1 argument")
			}
			items, err := iterate(i, a[0])
			if err != nil {
				return nil, err
			}
			dq.V = append(dq.V, items...)
			truncate()
			return object.None, nil
		}}, true
	case "extendleft":
		return &object.BuiltinFunc{Name: "extendleft", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "extendleft() takes 1 argument")
			}
			items, err := iterate(i, a[0])
			if err != nil {
				return nil, err
			}
			// extendleft reverses the iterable's order.
			for _, x := range items {
				dq.V = append([]object.Object{x}, dq.V...)
			}
			truncateLeft()
			return object.None, nil
		}}, true
	case "rotate":
		return &object.BuiltinFunc{Name: "rotate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			n := 1
			if len(a) >= 1 {
				if x, ok := toInt64(a[0]); ok {
					n = int(x)
				}
			}
			L := len(dq.V)
			if L == 0 {
				return object.None, nil
			}
			n %= L
			if n < 0 {
				n += L
			}
			// rotate(n>0) moves the last n items to the front.
			dq.V = append(append([]object.Object{}, dq.V[L-n:]...), dq.V[:L-n]...)
			return object.None, nil
		}}, true
	case "clear":
		return &object.BuiltinFunc{Name: "clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			dq.V = nil
			return object.None, nil
		}}, true
	case "count":
		return &object.BuiltinFunc{Name: "count", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "count() takes 1 argument")
			}
			c := 0
			for _, x := range dq.V {
				eq, err := object.Eq(x, a[0])
				if err != nil {
					return nil, err
				}
				if eq {
					c++
				}
			}
			return object.NewInt(int64(c)), nil
		}}, true
	case "index":
		return &object.BuiltinFunc{Name: "index", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "index() takes at least 1 argument")
			}
			start, stop := 0, len(dq.V)
			if len(a) >= 2 {
				if n, ok := toInt64(a[1]); ok {
					start = int(n)
					if start < 0 {
						start = 0
					}
				}
			}
			if len(a) >= 3 {
				if n, ok := toInt64(a[2]); ok {
					stop = int(n)
					if stop > len(dq.V) {
						stop = len(dq.V)
					}
				}
			}
			for k := start; k < stop; k++ {
				eq, err := object.Eq(dq.V[k], a[0])
				if err != nil {
					return nil, err
				}
				if eq {
					return object.NewInt(int64(k)), nil
				}
			}
			return nil, object.Errorf(i.valueErr, "deque.index(x): x not in deque")
		}}, true
	case "reverse":
		return &object.BuiltinFunc{Name: "reverse", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			for lo, hi := 0, len(dq.V)-1; lo < hi; lo, hi = lo+1, hi-1 {
				dq.V[lo], dq.V[hi] = dq.V[hi], dq.V[lo]
			}
			return object.None, nil
		}}, true
	case "copy":
		return &object.BuiltinFunc{Name: "copy", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			out := &object.Deque{MaxLen: dq.MaxLen, V: append([]object.Object{}, dq.V...)}
			return out, nil
		}}, true
	case "insert":
		return &object.BuiltinFunc{Name: "insert", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "insert() requires index and value")
			}
			idx, _ := toInt64(a[0])
			L := len(dq.V)
			n := int(idx)
			if n < 0 {
				n = L + n
			}
			if n < 0 {
				n = 0
			}
			if n > L {
				n = L
			}
			dq.V = append(dq.V, nil)
			copy(dq.V[n+1:], dq.V[n:])
			dq.V[n] = a[1]
			if dq.MaxLen >= 0 && len(dq.V) > dq.MaxLen {
				dq.V = dq.V[:dq.MaxLen]
			}
			return object.None, nil
		}}, true
	case "remove":
		return &object.BuiltinFunc{Name: "remove", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "remove() requires value")
			}
			for k, x := range dq.V {
				eq, err := object.Eq(x, a[0])
				if err != nil {
					return nil, err
				}
				if eq {
					dq.V = append(dq.V[:k], dq.V[k+1:]...)
					return object.None, nil
				}
			}
			return nil, object.Errorf(i.valueErr, "deque.remove(x): x not in deque")
		}}, true
	case "maxlen":
		if dq.MaxLen < 0 {
			return object.None, true
		}
		return object.NewInt(int64(dq.MaxLen)), true
	}
	return nil, false
}

// --- Counter methods ---

func counterMethod(i *Interp, c *object.Counter, name string) (object.Object, bool) {
	// Counter-specific methods take priority over dict protocol. update and
	// subtract deliberately differ from dict.update — they count elements
	// from an iterable rather than merging key/value pairs.
	switch name {
	case "most_common":
		return &object.BuiltinFunc{Name: "most_common", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			keys, vals := c.D.Items()
			type pair struct {
				k object.Object
				v object.Object
			}
			pairs := make([]pair, len(keys))
			for k, key := range keys {
				pairs[k] = pair{key, vals[k]}
			}
			sort.SliceStable(pairs, func(a, b int) bool {
				ai, _ := toInt64(pairs[a].v)
				bi, _ := toInt64(pairs[b].v)
				return ai > bi
			})
			n := len(pairs)
			if len(a) >= 1 {
				if x, ok := toInt64(a[0]); ok {
					if int(x) < n {
						n = int(x)
					}
				}
			}
			out := make([]object.Object, n)
			for k := 0; k < n; k++ {
				out[k] = &object.Tuple{V: []object.Object{pairs[k].k, pairs[k].v}}
			}
			return &object.List{V: out}, nil
		}}, true
	case "total":
		return &object.BuiltinFunc{Name: "total", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			var sum int64 = 0
			_, vals := c.D.Items()
			for _, v := range vals {
				if n, ok := toInt64(v); ok {
					sum += n
				}
			}
			return object.NewInt(sum), nil
		}}, true
	case "update":
		return &object.BuiltinFunc{Name: "update", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.None, nil
			}
			return nil, addCounts(c.D, a[0], +1, i)
		}}, true
	case "subtract":
		return &object.BuiltinFunc{Name: "subtract", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.None, nil
			}
			return nil, addCounts(c.D, a[0], -1, i)
		}}, true
	case "elements":
		return &object.BuiltinFunc{Name: "elements", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			keys, vals := c.D.Items()
			var out []object.Object
			for k, key := range keys {
				count, _ := toInt64(vals[k])
				for j := int64(0); j < count; j++ {
					out = append(out, key)
				}
			}
			idx := 0
			return &object.Iter{Next: func() (object.Object, bool, error) {
				if idx >= len(out) {
					return nil, false, nil
				}
				v := out[idx]
				idx++
				return v, true, nil
			}}, nil
		}}, true
	case "copy":
		return &object.BuiltinFunc{Name: "copy", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			out := &object.Counter{D: object.NewDict()}
			keys, vals := c.D.Items()
			for k, key := range keys {
				out.D.Set(key, vals[k])
			}
			return out, nil
		}}, true
	}
	// Fall back to the shared dict protocol (keys/values/items/...).
	if m, ok := dictMethod(c.D, name); ok {
		return m, ok
	}
	return nil, false
}

// addCounts merges counts from src (either a Counter/dict with int values or
// an iterable of hashables) into dst, multiplied by sign.
func addCounts(dst *object.Dict, src object.Object, sign int64, i *Interp) error {
	add := func(k object.Object, n int64) error {
		cur, ok, err := dst.Get(k)
		if err != nil {
			return err
		}
		base := int64(0)
		if ok {
			base, _ = toInt64(cur)
		}
		return dst.Set(k, object.NewInt(base+n*sign))
	}
	switch s := src.(type) {
	case *object.Dict:
		keys, vals := s.Items()
		for k, key := range keys {
			n, _ := toInt64(vals[k])
			if err := add(key, n); err != nil {
				return err
			}
		}
		return nil
	case *object.Counter:
		return addCounts(dst, s.D, sign, i)
	}
	items, err := iterate(i, src)
	if err != nil {
		return err
	}
	for _, x := range items {
		if err := add(x, 1); err != nil {
			return err
		}
	}
	return nil
}

// --- defaultdict methods ---

func defaultDictMethod(i *Interp, dd *object.DefaultDict, name string) (object.Object, bool) {
	if name == "copy" {
		return &object.BuiltinFunc{Name: "copy", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			out := &object.DefaultDict{D: object.NewDict(), Factory: dd.Factory}
			keys, vals := dd.D.Items()
			for k, key := range keys {
				out.D.Set(key, vals[k])
			}
			return out, nil
		}}, true
	}
	if m, ok := dictMethod(dd.D, name); ok {
		return m, ok
	}
	if name == "default_factory" {
		if dd.Factory == nil {
			return object.None, true
		}
		return dd.Factory, true
	}
	return nil, false
}

// --- OrderedDict methods ---

func orderedDictMethod(i *Interp, od *object.OrderedDict, name string) (object.Object, bool) {
	switch name {
	case "move_to_end":
		return &object.BuiltinFunc{Name: "move_to_end", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "move_to_end() takes 1 argument")
			}
			last := true
			if len(a) >= 2 {
				last = object.Truthy(a[1])
			}
			if kw != nil {
				if v, ok := kw.GetStr("last"); ok {
					last = object.Truthy(v)
				}
			}
			v, ok, err := od.D.Get(a[0])
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
			}
			od.D.Delete(a[0])
			if last {
				od.D.Set(a[0], v)
			} else {
				// Rebuild with key at front by clearing and re-inserting.
				keys, vals := od.D.Items()
				od.D.Clear()
				od.D.Set(a[0], v) //nolint
				for k, key := range keys {
					od.D.Set(key, vals[k]) //nolint
				}
			}
			return object.None, nil
		}}, true
	case "popitem":
		return &object.BuiltinFunc{Name: "popitem", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			last := true
			if len(a) >= 1 {
				last = object.Truthy(a[0])
			}
			if kw != nil {
				if v, ok := kw.GetStr("last"); ok {
					last = object.Truthy(v)
				}
			}
			keys, vals := od.D.Items()
			if len(keys) == 0 {
				return nil, object.Errorf(i.keyErr, "dictionary is empty")
			}
			idx := 0
			if last {
				idx = len(keys) - 1
			}
			k, v := keys[idx], vals[idx]
			od.D.Delete(k)
			return &object.Tuple{V: []object.Object{k, v}}, nil
		}}, true
	case "copy":
		return &object.BuiltinFunc{Name: "copy", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			out := &object.OrderedDict{D: object.NewDict()}
			keys, vals := od.D.Items()
			for k, key := range keys {
				out.D.Set(key, vals[k])
			}
			return out, nil
		}}, true
	}
	// Fall back to regular dict methods for everything else.
	if m, ok := dictMethod(od.D, name); ok {
		return m, ok
	}
	return nil, false
}

package vm

import (
	"github.com/tamnd/goipy/object"
)

// --- functools ---

func (i *Interp) buildFunctools() *object.Module {
	m := &object.Module{Name: "functools", Dict: object.NewDict()}

	// reduce(fn, iterable[, init])
	m.Dict.SetStr("reduce", &object.BuiltinFunc{Name: "reduce", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 || len(a) > 3 {
			return nil, object.Errorf(i.typeErr, "reduce() takes 2 or 3 arguments")
		}
		fn := a[0]
		items, err := iterate(i, a[1])
		if err != nil {
			return nil, err
		}
		var acc object.Object
		start := 0
		if len(a) == 3 {
			acc = a[2]
		} else {
			if len(items) == 0 {
				return nil, object.Errorf(i.typeErr, "reduce() of empty iterable with no initial value")
			}
			acc = items[0]
			start = 1
		}
		for _, x := range items[start:] {
			acc, err = i.callObject(fn, []object.Object{acc, x}, nil)
			if err != nil {
				return nil, err
			}
		}
		return acc, nil
	}})

	// partial(fn, *args, **kwargs) — returns a callable that prepends args and
	// merges kwargs with call-time ones (call-time wins).
	m.Dict.SetStr("partial", &object.BuiltinFunc{Name: "partial", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "partial() takes at least one argument")
		}
		inner := a[0]
		bound := append([]object.Object{}, a[1:]...)
		var boundKw *object.Dict
		if kw != nil {
			boundKw = object.NewDict()
			keys, vals := kw.Items()
			for idx, k := range keys {
				boundKw.Set(k, vals[idx])
			}
		}
		return &object.BuiltinFunc{Name: "partial", Call: func(_ any, ca []object.Object, ckw *object.Dict) (object.Object, error) {
			all := append([]object.Object{}, bound...)
			all = append(all, ca...)
			mergedKw := boundKw
			if ckw != nil {
				mergedKw = object.NewDict()
				if boundKw != nil {
					bk, bv := boundKw.Items()
					for idx, k := range bk {
						mergedKw.Set(k, bv[idx])
					}
				}
				ck, cv := ckw.Items()
				for idx, k := range ck {
					mergedKw.Set(k, cv[idx])
				}
			}
			return i.callObject(inner, all, mergedKw)
		}}, nil
	}})

	// lru_cache(maxsize=128) and cache — both decorate a callable with a
	// memoizing wrapper. Keys are (args, sorted-kwargs) tuples; we rely on
	// object.Hash/Eq which already route Instance dunders.
	lruCache := func(maxsize int) object.Object {
		return &object.BuiltinFunc{Name: "lru_cache_decorator", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "lru_cache decorator takes exactly one argument")
			}
			fn := a[0]
			cache := object.NewDict()
			var order []object.Object
			hits, misses := 0, 0
			wrapper := &object.BuiltinFunc{Name: "lru_wrapper", Call: func(_ any, ca []object.Object, ckw *object.Dict) (object.Object, error) {
				key := buildCacheKey(ca, ckw)
				if v, ok, err := cache.Get(key); err == nil && ok {
					hits++
					// LRU: move to end.
					for idx, k := range order {
						eq, _ := object.Eq(k, key)
						if eq {
							order = append(order[:idx], order[idx+1:]...)
							break
						}
					}
					order = append(order, key)
					return v, nil
				}
				misses++
				r, err := i.callObject(fn, ca, ckw)
				if err != nil {
					return nil, err
				}
				if err := cache.Set(key, r); err != nil {
					return nil, err
				}
				order = append(order, key)
				if maxsize > 0 && len(order) > maxsize {
					oldest := order[0]
					order = order[1:]
					cache.Delete(oldest)
				}
				return r, nil
			}}
			wrapper.Attrs = object.NewDict()
			wrapper.Attrs.SetStr("cache_info", &object.BuiltinFunc{Name: "cache_info", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.Tuple{V: []object.Object{
					object.NewInt(int64(hits)),
					object.NewInt(int64(misses)),
					object.NewInt(int64(maxsize)),
					object.NewInt(int64(len(order))),
				}}, nil
			}})
			wrapper.Attrs.SetStr("cache_clear", &object.BuiltinFunc{Name: "cache_clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				cache = object.NewDict()
				order = nil
				hits, misses = 0, 0
				return object.None, nil
			}})
			wrapper.Attrs.SetStr("__wrapped__", fn)
			return wrapper, nil
		}}
	}
	m.Dict.SetStr("lru_cache", &object.BuiltinFunc{Name: "lru_cache", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		// Callable as lru_cache(fn) OR lru_cache(maxsize=128). Distinguish by
		// whether the single positional arg is callable (like CPython).
		if len(a) == 1 {
			if _, callable := a[0].(*object.Function); callable {
				return lruCache(128).(*object.BuiltinFunc).Call(nil, a, nil)
			}
			if _, callable := a[0].(*object.BuiltinFunc); callable {
				return lruCache(128).(*object.BuiltinFunc).Call(nil, a, nil)
			}
		}
		maxsize := 128
		if len(a) >= 1 {
			if n, ok := toInt64(a[0]); ok {
				maxsize = int(n)
			} else if _, isNone := a[0].(*object.NoneType); isNone {
				maxsize = 0
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("maxsize"); ok {
				if n, ok := toInt64(v); ok {
					maxsize = int(n)
				}
			}
		}
		return lruCache(maxsize), nil
	}})
	m.Dict.SetStr("cache", lruCache(0))

	// wraps(wrapped) — returns a decorator that copies name/doc onto the
	// wrapper. Minimal: just sets __name__, __doc__, __wrapped__.
	m.Dict.SetStr("wraps", &object.BuiltinFunc{Name: "wraps", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "wraps() takes one argument")
		}
		wrapped := a[0]
		return &object.BuiltinFunc{Name: "wraps_decorator", Call: func(_ any, b []object.Object, _ *object.Dict) (object.Object, error) {
			if len(b) != 1 {
				return nil, object.Errorf(i.typeErr, "wraps decorator takes one argument")
			}
			wrapper := b[0]
			if bf, ok := wrapper.(*object.BuiltinFunc); ok {
				if bf.Attrs == nil {
					bf.Attrs = object.NewDict()
				}
				if nm, err := i.getAttr(wrapped, "__name__"); err == nil {
					bf.Attrs.SetStr("__name__", nm)
					if s, ok := nm.(*object.Str); ok {
						bf.Name = s.V
					}
				}
				bf.Attrs.SetStr("__wrapped__", wrapped)
			}
			if fn, ok := wrapper.(*object.Function); ok {
				if nm, err := i.getAttr(wrapped, "__name__"); err == nil {
					if s, ok := nm.(*object.Str); ok {
						fn.Name = s.V
					}
				}
				if fn.Dict == nil {
					fn.Dict = object.NewDict()
				}
				fn.Dict.SetStr("__wrapped__", wrapped)
			}
			return wrapper, nil
		}}, nil
	}})

	// cached_property(fn): a non-data descriptor that stores its result in
	// inst.__dict__ under the attribute name. Implemented as a Property whose
	// Fget runs once and caches.
	m.Dict.SetStr("cached_property", &object.BuiltinFunc{Name: "cached_property", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "cached_property() takes one argument")
		}
		return &cachedProperty{fn: a[0], interp: i}, nil
	}})

	return m
}

// buildCacheKey packs positional and keyword arguments into a Tuple usable
// as a dict key. Kwargs are flattened in insertion order as (k, v) pairs
// appended after the positional args; this matches CPython's _make_key
// closely enough for correctness.
func buildCacheKey(args []object.Object, kw *object.Dict) object.Object {
	parts := make([]object.Object, 0, len(args)+1)
	parts = append(parts, args...)
	if kw != nil {
		keys, vals := kw.Items()
		for idx, k := range keys {
			parts = append(parts, k, vals[idx])
		}
	}
	return &object.Tuple{V: parts}
}

// cachedProperty acts like a non-data descriptor: first access runs
// fn(inst) and stashes the result in inst.__dict__[name], so subsequent
// accesses find it directly in the instance dict (bypassing the
// descriptor, which is non-data).
type cachedProperty struct {
	fn     object.Object
	name   string
	interp *Interp
}

// --- itertools ---

func (i *Interp) buildItertools() *object.Module {
	m := &object.Module{Name: "itertools", Dict: object.NewDict()}

	m.Dict.SetStr("count", &object.BuiltinFunc{Name: "count", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		start := object.Object(object.NewInt(0))
		step := object.Object(object.NewInt(1))
		if len(a) >= 1 {
			start = a[0]
		}
		if len(a) >= 2 {
			step = a[1]
		}
		cur := start
		return &object.Iter{Next: func() (object.Object, bool, error) {
			v := cur
			next, err := i.add(cur, step)
			if err != nil {
				return nil, false, err
			}
			cur = next
			return v, true, nil
		}}, nil
	}})

	m.Dict.SetStr("cycle", &object.BuiltinFunc{Name: "cycle", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "cycle() takes one argument")
		}
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if len(items) == 0 {
				return nil, false, nil
			}
			v := items[idx%len(items)]
			idx++
			return v, true, nil
		}}, nil
	}})

	m.Dict.SetStr("repeat", &object.BuiltinFunc{Name: "repeat", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 || len(a) > 2 {
			return nil, object.Errorf(i.typeErr, "repeat() takes 1 or 2 arguments")
		}
		val := a[0]
		times := -1
		if len(a) == 2 {
			n, ok := toInt64(a[1])
			if !ok {
				return nil, object.Errorf(i.typeErr, "repeat() times must be int")
			}
			times = int(n)
		}
		n := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if times >= 0 && n >= times {
				return nil, false, nil
			}
			n++
			return val, true, nil
		}}, nil
	}})

	m.Dict.SetStr("chain", &object.BuiltinFunc{Name: "chain", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		iters := make([]*object.Iter, 0, len(a))
		for _, v := range a {
			it, err := i.getIter(v)
			if err != nil {
				return nil, err
			}
			iters = append(iters, it)
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			for idx < len(iters) {
				v, ok, err := iters[idx].Next()
				if err != nil {
					return nil, false, err
				}
				if ok {
					return v, true, nil
				}
				idx++
			}
			return nil, false, nil
		}}, nil
	}})

	chainFromIterable := &object.BuiltinFunc{Name: "from_iterable", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "from_iterable() takes one argument")
		}
		outer, err := i.getIter(a[0])
		if err != nil {
			return nil, err
		}
		var inner *object.Iter
		return &object.Iter{Next: func() (object.Object, bool, error) {
			for {
				if inner != nil {
					v, ok, err := inner.Next()
					if err != nil {
						return nil, false, err
					}
					if ok {
						return v, true, nil
					}
					inner = nil
				}
				v, ok, err := outer.Next()
				if err != nil || !ok {
					return nil, false, err
				}
				inner, err = i.getIter(v)
				if err != nil {
					return nil, false, err
				}
			}
		}}, nil
	}}
	chainFn, _ := m.Dict.GetStr("chain")
	chainFn.(*object.BuiltinFunc).Attrs = object.NewDict()
	chainFn.(*object.BuiltinFunc).Attrs.SetStr("from_iterable", chainFromIterable)

	m.Dict.SetStr("compress", &object.BuiltinFunc{Name: "compress", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "compress() takes 2 arguments")
		}
		data, err := i.getIter(a[0])
		if err != nil {
			return nil, err
		}
		sel, err := i.getIter(a[1])
		if err != nil {
			return nil, err
		}
		return &object.Iter{Next: func() (object.Object, bool, error) {
			for {
				d, ok1, err := data.Next()
				if err != nil || !ok1 {
					return nil, false, err
				}
				s, ok2, err := sel.Next()
				if err != nil || !ok2 {
					return nil, false, err
				}
				if object.Truthy(s) {
					return d, true, nil
				}
			}
		}}, nil
	}})

	m.Dict.SetStr("dropwhile", &object.BuiltinFunc{Name: "dropwhile", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "dropwhile() takes 2 arguments")
		}
		pred, src := a[0], a[1]
		it, err := i.getIter(src)
		if err != nil {
			return nil, err
		}
		dropping := true
		return &object.Iter{Next: func() (object.Object, bool, error) {
			for {
				v, ok, err := it.Next()
				if err != nil || !ok {
					return nil, false, err
				}
				if dropping {
					r, err := i.callObject(pred, []object.Object{v}, nil)
					if err != nil {
						return nil, false, err
					}
					if object.Truthy(r) {
						continue
					}
					dropping = false
				}
				return v, true, nil
			}
		}}, nil
	}})

	m.Dict.SetStr("takewhile", &object.BuiltinFunc{Name: "takewhile", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "takewhile() takes 2 arguments")
		}
		pred, src := a[0], a[1]
		it, err := i.getIter(src)
		if err != nil {
			return nil, err
		}
		done := false
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if done {
				return nil, false, nil
			}
			v, ok, err := it.Next()
			if err != nil || !ok {
				return nil, false, err
			}
			r, err := i.callObject(pred, []object.Object{v}, nil)
			if err != nil {
				return nil, false, err
			}
			if !object.Truthy(r) {
				done = true
				return nil, false, nil
			}
			return v, true, nil
		}}, nil
	}})

	m.Dict.SetStr("islice", &object.BuiltinFunc{Name: "islice", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 || len(a) > 4 {
			return nil, object.Errorf(i.typeErr, "islice() takes 2..4 arguments")
		}
		src := a[0]
		start, stop, step := 0, -1, 1
		parseOptInt := func(o object.Object, def int) int {
			if _, isNone := o.(*object.NoneType); isNone {
				return def
			}
			n, _ := toInt64(o)
			return int(n)
		}
		switch len(a) {
		case 2:
			stop = parseOptInt(a[1], -1)
		case 3:
			start = parseOptInt(a[1], 0)
			stop = parseOptInt(a[2], -1)
		case 4:
			start = parseOptInt(a[1], 0)
			stop = parseOptInt(a[2], -1)
			step = parseOptInt(a[3], 1)
			if step <= 0 {
				return nil, object.Errorf(i.valueErr, "islice step must be positive")
			}
		}
		it, err := i.getIter(src)
		if err != nil {
			return nil, err
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			for {
				v, ok, err := it.Next()
				if err != nil || !ok {
					return nil, false, err
				}
				cur := idx
				idx++
				if cur < start {
					continue
				}
				if stop >= 0 && cur >= stop {
					return nil, false, nil
				}
				if (cur-start)%step != 0 {
					continue
				}
				return v, true, nil
			}
		}}, nil
	}})

	m.Dict.SetStr("starmap", &object.BuiltinFunc{Name: "starmap", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "starmap() takes 2 arguments")
		}
		fn := a[0]
		it, err := i.getIter(a[1])
		if err != nil {
			return nil, err
		}
		return &object.Iter{Next: func() (object.Object, bool, error) {
			v, ok, err := it.Next()
			if err != nil || !ok {
				return nil, false, err
			}
			args, err := iterate(i, v)
			if err != nil {
				return nil, false, err
			}
			r, err := i.callObject(fn, args, nil)
			if err != nil {
				return nil, false, err
			}
			return r, true, nil
		}}, nil
	}})

	m.Dict.SetStr("tee", &object.BuiltinFunc{Name: "tee", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n := 2
		if len(a) >= 2 {
			if x, ok := toInt64(a[1]); ok {
				n = int(x)
			}
		}
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "tee() takes at least 1 argument")
		}
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		out := make([]object.Object, n)
		for k := 0; k < n; k++ {
			idx := 0
			out[k] = &object.Iter{Next: func() (object.Object, bool, error) {
				if idx >= len(items) {
					return nil, false, nil
				}
				v := items[idx]
				idx++
				return v, true, nil
			}}
		}
		return &object.Tuple{V: out}, nil
	}})

	m.Dict.SetStr("zip_longest", &object.BuiltinFunc{Name: "zip_longest", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		var fill object.Object = object.None
		if kw != nil {
			if v, ok := kw.GetStr("fillvalue"); ok {
				fill = v
			}
		}
		iters := make([]*object.Iter, 0, len(a))
		for _, v := range a {
			it, err := i.getIter(v)
			if err != nil {
				return nil, err
			}
			iters = append(iters, it)
		}
		done := make([]bool, len(iters))
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if len(iters) == 0 {
				return nil, false, nil
			}
			row := make([]object.Object, len(iters))
			allDone := true
			for k, it := range iters {
				if done[k] {
					row[k] = fill
					continue
				}
				v, ok, err := it.Next()
				if err != nil {
					return nil, false, err
				}
				if !ok {
					done[k] = true
					row[k] = fill
					continue
				}
				row[k] = v
				allDone = false
			}
			if allDone {
				return nil, false, nil
			}
			return &object.Tuple{V: row}, true, nil
		}}, nil
	}})

	m.Dict.SetStr("product", &object.BuiltinFunc{Name: "product", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		repeat := 1
		if kw != nil {
			if v, ok := kw.GetStr("repeat"); ok {
				if n, ok := toInt64(v); ok {
					repeat = int(n)
				}
			}
		}
		pools := make([][]object.Object, 0, len(a)*repeat)
		for r := 0; r < repeat; r++ {
			for _, v := range a {
				items, err := iterate(i, v)
				if err != nil {
					return nil, err
				}
				pools = append(pools, items)
			}
		}
		var results []object.Object
		if len(pools) == 0 {
			results = []object.Object{&object.Tuple{V: nil}}
		} else {
			results = []object.Object{&object.Tuple{V: []object.Object{}}}
			for _, pool := range pools {
				var next []object.Object
				for _, prefix := range results {
					ptuple := prefix.(*object.Tuple)
					for _, x := range pool {
						combo := make([]object.Object, 0, len(ptuple.V)+1)
						combo = append(combo, ptuple.V...)
						combo = append(combo, x)
						next = append(next, &object.Tuple{V: combo})
					}
				}
				results = next
			}
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(results) {
				return nil, false, nil
			}
			v := results[idx]
			idx++
			return v, true, nil
		}}, nil
	}})

	m.Dict.SetStr("permutations", &object.BuiltinFunc{Name: "permutations", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "permutations() requires iterable")
		}
		pool, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		r := len(pool)
		if len(a) >= 2 {
			if _, isNone := a[1].(*object.NoneType); !isNone {
				n, ok := toInt64(a[1])
				if !ok {
					return nil, object.Errorf(i.typeErr, "r must be int")
				}
				r = int(n)
			}
		}
		results := permute(pool, r)
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(results) {
				return nil, false, nil
			}
			v := results[idx]
			idx++
			return v, true, nil
		}}, nil
	}})

	m.Dict.SetStr("combinations", &object.BuiltinFunc{Name: "combinations", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "combinations() takes 2 arguments")
		}
		pool, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		rI, _ := toInt64(a[1])
		results := combinations(pool, int(rI), false)
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(results) {
				return nil, false, nil
			}
			v := results[idx]
			idx++
			return v, true, nil
		}}, nil
	}})

	m.Dict.SetStr("combinations_with_replacement", &object.BuiltinFunc{Name: "combinations_with_replacement", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "combinations_with_replacement() takes 2 arguments")
		}
		pool, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		rI, _ := toInt64(a[1])
		results := combinations(pool, int(rI), true)
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(results) {
				return nil, false, nil
			}
			v := results[idx]
			idx++
			return v, true, nil
		}}, nil
	}})

	m.Dict.SetStr("accumulate", &object.BuiltinFunc{Name: "accumulate", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "accumulate() requires iterable")
		}
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		var fn object.Object
		if len(a) >= 2 {
			if _, isNone := a[1].(*object.NoneType); !isNone {
				fn = a[1]
			}
		}
		var initial object.Object
		hasInitial := false
		if kw != nil {
			if v, ok := kw.GetStr("initial"); ok {
				if _, isNone := v.(*object.NoneType); !isNone {
					initial = v
					hasInitial = true
				}
			}
		}
		var out []object.Object
		var acc object.Object
		started := false
		if hasInitial {
			acc = initial
			out = append(out, acc)
			started = true
		}
		for _, x := range items {
			if !started {
				acc = x
				out = append(out, acc)
				started = true
				continue
			}
			if fn != nil {
				acc, err = i.callObject(fn, []object.Object{acc, x}, nil)
				if err != nil {
					return nil, err
				}
			} else {
				acc, err = i.add(acc, x)
				if err != nil {
					return nil, err
				}
			}
			out = append(out, acc)
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
	}})

	m.Dict.SetStr("pairwise", &object.BuiltinFunc{Name: "pairwise", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "pairwise() takes 1 argument")
		}
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx+1 >= len(items) {
				return nil, false, nil
			}
			v := &object.Tuple{V: []object.Object{items[idx], items[idx+1]}}
			idx++
			return v, true, nil
		}}, nil
	}})

	m.Dict.SetStr("groupby", &object.BuiltinFunc{Name: "groupby", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "groupby() requires iterable")
		}
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		var keyFn object.Object
		if len(a) >= 2 {
			if _, isNone := a[1].(*object.NoneType); !isNone {
				keyFn = a[1]
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("key"); ok {
				if _, isNone := v.(*object.NoneType); !isNone {
					keyFn = v
				}
			}
		}
		keyOf := func(x object.Object) (object.Object, error) {
			if keyFn == nil {
				return x, nil
			}
			return i.callObject(keyFn, []object.Object{x}, nil)
		}
		// Precompute groups (unlike CPython's streaming groupby — acceptable
		// since this VM eagerly materializes most iterables anyway).
		type group struct {
			key object.Object
			v   []object.Object
		}
		var groups []group
		for _, x := range items {
			k, err := keyOf(x)
			if err != nil {
				return nil, err
			}
			if len(groups) > 0 {
				eq, err := object.Eq(groups[len(groups)-1].key, k)
				if err != nil {
					return nil, err
				}
				if eq {
					groups[len(groups)-1].v = append(groups[len(groups)-1].v, x)
					continue
				}
			}
			groups = append(groups, group{key: k, v: []object.Object{x}})
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(groups) {
				return nil, false, nil
			}
			g := groups[idx]
			idx++
			inner := 0
			sub := &object.Iter{Next: func() (object.Object, bool, error) {
				if inner >= len(g.v) {
					return nil, false, nil
				}
				v := g.v[inner]
				inner++
				return v, true, nil
			}}
			return &object.Tuple{V: []object.Object{g.key, sub}}, true, nil
		}}, nil
	}})

	m.Dict.SetStr("filterfalse", &object.BuiltinFunc{Name: "filterfalse", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 2 {
			return nil, object.Errorf(i.typeErr, "filterfalse() takes 2 arguments")
		}
		pred, src := a[0], a[1]
		it, err := i.getIter(src)
		if err != nil {
			return nil, err
		}
		return &object.Iter{Next: func() (object.Object, bool, error) {
			for {
				v, ok, err := it.Next()
				if err != nil || !ok {
					return nil, false, err
				}
				var keep bool
				if _, isNone := pred.(*object.NoneType); isNone {
					keep = !object.Truthy(v)
				} else {
					r, err := i.callObject(pred, []object.Object{v}, nil)
					if err != nil {
						return nil, false, err
					}
					keep = !object.Truthy(r)
				}
				if keep {
					return v, true, nil
				}
			}
		}}, nil
	}})

	return m
}

func permute(pool []object.Object, r int) []object.Object {
	if r < 0 || r > len(pool) {
		return nil
	}
	var out []object.Object
	indices := make([]int, len(pool))
	for i := range indices {
		indices[i] = i
	}
	used := make([]bool, len(pool))
	var rec func(cur []int)
	rec = func(cur []int) {
		if len(cur) == r {
			row := make([]object.Object, r)
			for k, idx := range cur {
				row[k] = pool[idx]
			}
			out = append(out, &object.Tuple{V: row})
			return
		}
		for k := 0; k < len(pool); k++ {
			if used[k] {
				continue
			}
			used[k] = true
			rec(append(cur, k))
			used[k] = false
		}
	}
	rec(nil)
	return out
}

func combinations(pool []object.Object, r int, withRepl bool) []object.Object {
	if r < 0 {
		return nil
	}
	if !withRepl && r > len(pool) {
		return nil
	}
	var out []object.Object
	var rec func(start int, cur []int)
	rec = func(start int, cur []int) {
		if len(cur) == r {
			row := make([]object.Object, r)
			for k, idx := range cur {
				row[k] = pool[idx]
			}
			out = append(out, &object.Tuple{V: row})
			return
		}
		for k := start; k < len(pool); k++ {
			nextStart := k + 1
			if withRepl {
				nextStart = k
			}
			rec(nextStart, append(cur, k))
		}
	}
	rec(0, nil)
	return out
}

package vm

import (
	"crypto/rand"
	"encoding/binary"
	"math"

	"github.com/tamnd/goipy/object"
)

// splitmix64 is a simple, fast PRNG whose full state is a single uint64.
// This lets us implement getstate/setstate without reflection into math/rand.
type splitmix64 struct {
	state    uint64
	gaussNext float64 // cached second Gaussian variate (NaN = empty)
}

func newSplitmix64(seed int64) *splitmix64 {
	s := &splitmix64{state: uint64(seed), gaussNext: math.NaN()}
	// Warm up.
	for k := 0; k < 20; k++ {
		s.next()
	}
	return s
}

func (s *splitmix64) next() uint64 {
	s.state += 0x9e3779b97f4a7c15
	z := s.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// float64 returns a uniform float in [0, 1).
func (s *splitmix64) float64() float64 {
	return float64(s.next()>>11) * (1.0 / (1 << 53))
}

// int63n returns a uniform int in [0, n).
func (s *splitmix64) int63n(n int64) int64 {
	if n <= 0 {
		return 0
	}
	// Rejection sampling for unbiased result.
	threshold := uint64(-uint64(n)) % uint64(n)
	for {
		r := s.next()
		if r >= threshold {
			return int64(r % uint64(n))
		}
	}
}

// intn returns uniform int in [0, n).
func (s *splitmix64) intn(n int) int {
	return int(s.int63n(int64(n)))
}

func (i *Interp) buildRandom() *object.Module {
	m := &object.Module{Name: "random", Dict: object.NewDict()}

	// seed with OS random so default state isn't always the same.
	var seedBytes [8]byte
	_, _ = rand.Read(seedBytes[:])
	rng := newSplitmix64(int64(binary.LittleEndian.Uint64(seedBytes[:])))

	// ── helpers ──────────────────────────────────────────────────────────────

	// normalvariate0 is the inner Gaussian using Box-Muller / Kinderman-Monahan.
	// Used internally by gammavariate.
	normalvariate0 := func() float64 {
		// Kinderman-Monahan method (same as CPython normalvariate).
		const NV_MAGICCONST = 1.7155277699214135 // 4 * exp(-0.5) / sqrt(2)
		for {
			u1 := rng.float64()
			u2 := 1.0 - rng.float64()
			z := NV_MAGICCONST * (u1 - 0.5) / u2
			zz := z * z / 4.0
			if zz <= -math.Log(u2) {
				return z
			}
		}
	}

	// gammavariate0 is the inner gamma sampler (alpha, 1.0).
	var gammavariate0 func(alpha float64) float64
	gammavariate0 = func(alpha float64) float64 {
		if alpha > 1.0 {
			// Marsaglia-Tsang (2000).
			d := alpha - 1.0/3.0
			c := 1.0 / math.Sqrt(9.0*d)
			for {
				x := normalvariate0()
				if x > -1.0/c {
					v := (1.0 + c*x)
					v = v * v * v
					u := rng.float64()
					if u < 1.0-0.0331*(x*x)*(x*x) {
						return d * v
					}
					if math.Log(u) < 0.5*x*x+d*(1.0-v+math.Log(v)) {
						return d * v
					}
				}
			}
		} else if alpha == 1.0 {
			return -math.Log(1.0 - rng.float64())
		}
		// alpha < 1.0: Ahrens-Dieter (1974).
		e := math.E + alpha
		for {
			u := rng.float64()
			p := e * u
			var x float64
			if p <= 1.0 {
				x = math.Pow(p, 1.0/alpha)
			} else {
				x = -math.Log((e - p) / alpha)
			}
			u1 := rng.float64()
			if p > 1.0 {
				if u1 <= math.Pow(x, alpha-1.0) {
					return x
				}
			} else if u1 <= math.Exp(-x) {
				return x
			}
		}
	}

	// ── Bookkeeping ───────────────────────────────────────────────────────────

	m.Dict.SetStr("seed", &object.BuiltinFunc{Name: "seed", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var s int64
		if len(a) == 0 {
			var b [8]byte
			_, _ = rand.Read(b[:])
			s = int64(binary.LittleEndian.Uint64(b[:]))
		} else if a[0] == object.None {
			var b [8]byte
			_, _ = rand.Read(b[:])
			s = int64(binary.LittleEndian.Uint64(b[:]))
		} else if n, ok := toInt64(a[0]); ok {
			s = n
		} else if f, ok := toFloat64Any(a[0]); ok {
			s = int64(math.Float64bits(f))
		} else {
			h, err := object.Hash(a[0])
			if err != nil {
				return nil, err
			}
			s = int64(h)
		}
		rng.state = uint64(s)
		rng.gaussNext = math.NaN()
		for k := 0; k < 20; k++ {
			rng.next()
		}
		return object.None, nil
	}})

	// getstate() → (version=3, (state_uint64,), gauss_next)
	m.Dict.SetStr("getstate", &object.BuiltinFunc{Name: "getstate", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var gn object.Object = object.None
		if !math.IsNaN(rng.gaussNext) {
			gn = &object.Float{V: rng.gaussNext}
		}
		internalState := &object.Tuple{V: []object.Object{object.NewInt(int64(rng.state))}}
		return &object.Tuple{V: []object.Object{
			object.NewInt(3),
			internalState,
			gn,
		}}, nil
	}})

	// setstate(state) — restore state from getstate() output.
	m.Dict.SetStr("setstate", &object.BuiltinFunc{Name: "setstate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "setstate() requires a state argument")
		}
		tup, ok := a[0].(*object.Tuple)
		if !ok || len(tup.V) < 3 {
			return nil, object.Errorf(i.valueErr, "setstate() requires state from getstate()")
		}
		// Expect (3, (state_int,), gauss_next_or_None)
		inner, ok2 := tup.V[1].(*object.Tuple)
		if !ok2 || len(inner.V) < 1 {
			return nil, object.Errorf(i.valueErr, "setstate() invalid state")
		}
		if n, ok3 := toInt64(inner.V[0]); ok3 {
			rng.state = uint64(n)
		}
		if tup.V[2] == object.None {
			rng.gaussNext = math.NaN()
		} else if f, ok3 := toFloat64Any(tup.V[2]); ok3 {
			rng.gaussNext = f
		}
		return object.None, nil
	}})

	// ── Bytes ─────────────────────────────────────────────────────────────────

	m.Dict.SetStr("randbytes", &object.BuiltinFunc{Name: "randbytes", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "randbytes() requires 1 argument")
		}
		n, ok := toInt64(a[0])
		if !ok || n < 0 {
			return nil, object.Errorf(i.valueErr, "randbytes() requires a non-negative integer")
		}
		b := make([]byte, n)
		for k := int64(0); k < n; k++ {
			b[k] = byte(rng.next())
		}
		return &object.Bytes{V: b}, nil
	}})

	// ── Integers ──────────────────────────────────────────────────────────────

	m.Dict.SetStr("getrandbits", &object.BuiltinFunc{Name: "getrandbits", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "getrandbits() requires 1 argument")
		}
		k, ok := toInt64(a[0])
		if !ok || k < 0 {
			return nil, object.Errorf(i.valueErr, "getrandbits() requires a non-negative integer")
		}
		if k == 0 {
			return object.NewInt(0), nil
		}
		// Build k random bits from 64-bit words.
		words := (k + 63) / 64
		bits := make([]uint64, words)
		for j := int64(0); j < words; j++ {
			bits[j] = rng.next()
		}
		// Mask the top word to exactly k bits.
		topBits := k % 64
		if topBits != 0 {
			bits[words-1] &= (1 << topBits) - 1
		}
		// Convert to big.Int.
		result := new(object.Int)
		for j := words - 1; j >= 0; j-- {
			result.V.Lsh(&result.V, 64)
			var w object.Int
			w.V.SetUint64(bits[j])
			result.V.Or(&result.V, &w.V)
		}
		return result, nil
	}})

	m.Dict.SetStr("randint", &object.BuiltinFunc{Name: "randint", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "randint() requires 2 arguments")
		}
		lo, _ := toInt64(a[0])
		hi, _ := toInt64(a[1])
		if hi < lo {
			return nil, object.Errorf(i.valueErr, "empty range for randint(%d, %d)", lo, hi)
		}
		return object.NewInt(lo + rng.int63n(hi-lo+1)), nil
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
			return nil, object.Errorf(i.valueErr, "zero step for randrange()")
		}
		width := stop - start
		if (step > 0 && width <= 0) || (step < 0 && width >= 0) {
			return nil, object.Errorf(i.valueErr, "empty range for randrange()")
		}
		n := (width + step - randSign(step)) / step
		if n <= 0 {
			return nil, object.Errorf(i.valueErr, "empty range for randrange()")
		}
		return object.NewInt(start + step*rng.int63n(n)), nil
	}})

	// ── Sequences ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("choice", &object.BuiltinFunc{Name: "choice", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "choice() requires a sequence")
		}
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, object.Errorf(i.indexErr, "Cannot choose from an empty sequence")
		}
		return items[rng.intn(len(items))], nil
	}})

	m.Dict.SetStr("choices", &object.BuiltinFunc{Name: "choices", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "choices() requires a population")
		}
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, object.Errorf(i.indexErr, "Cannot choose from an empty sequence")
		}
		k := 1
		var weights []float64
		var cumWeights []float64
		if kw != nil {
			if v, ok := kw.GetStr("k"); ok {
				if n, ok2 := toInt64(v); ok2 {
					k = int(n)
				}
			}
			if v, ok := kw.GetStr("weights"); ok && v != object.None {
				ws, err2 := iterate(i, v)
				if err2 == nil {
					for _, w := range ws {
						f, _ := toFloat64Any(w)
						weights = append(weights, f)
					}
				}
			}
			if v, ok := kw.GetStr("cum_weights"); ok && v != object.None {
				cws, err2 := iterate(i, v)
				if err2 == nil {
					for _, w := range cws {
						f, _ := toFloat64Any(w)
						cumWeights = append(cumWeights, f)
					}
				}
			}
		}
		if len(weights) > 0 && len(cumWeights) > 0 {
			return nil, object.Errorf(i.typeErr, "Cannot specify both weights and cumulative weights")
		}
		// Build cumulative weights.
		if len(weights) > 0 {
			cumWeights = make([]float64, len(weights))
			cumWeights[0] = weights[0]
			for j := 1; j < len(weights); j++ {
				cumWeights[j] = cumWeights[j-1] + weights[j]
			}
		}
		out := make([]object.Object, k)
		if len(cumWeights) == 0 {
			for j := 0; j < k; j++ {
				out[j] = items[rng.intn(len(items))]
			}
		} else {
			total := cumWeights[len(cumWeights)-1]
			for j := 0; j < k; j++ {
				r := rng.float64() * total
				// Binary search for the bucket.
				lo2, hi2 := 0, len(cumWeights)-1
				for lo2 < hi2 {
					mid := (lo2 + hi2) / 2
					if cumWeights[mid] < r {
						lo2 = mid + 1
					} else {
						hi2 = mid
					}
				}
				out[j] = items[lo2]
			}
		}
		return &object.List{V: out}, nil
	}})

	m.Dict.SetStr("shuffle", &object.BuiltinFunc{Name: "shuffle", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "shuffle() requires a list")
		}
		lst, ok := a[0].(*object.List)
		if !ok {
			return nil, object.Errorf(i.typeErr, "shuffle requires a list")
		}
		for k := len(lst.V) - 1; k > 0; k-- {
			j := rng.intn(k + 1)
			lst.V[k], lst.V[j] = lst.V[j], lst.V[k]
		}
		return object.None, nil
	}})

	m.Dict.SetStr("sample", &object.BuiltinFunc{Name: "sample", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "sample() requires population and k")
		}
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		// Handle counts= keyword: expand population.
		if kw != nil {
			if cv, ok := kw.GetStr("counts"); ok && cv != object.None {
				counts, err2 := iterate(i, cv)
				if err2 == nil && len(counts) == len(items) {
					var expanded []object.Object
					for j, item := range items {
						cnt := int64(1)
						if n, ok2 := toInt64(counts[j]); ok2 {
							cnt = n
						}
						for c := int64(0); c < cnt; c++ {
							expanded = append(expanded, item)
						}
					}
					items = expanded
				}
			}
		}
		k, _ := toInt64(a[1])
		if k < 0 || int(k) > len(items) {
			return nil, object.Errorf(i.valueErr, "sample larger than population")
		}
		pool := append([]object.Object{}, items...)
		out := make([]object.Object, k)
		for j := int64(0); j < k; j++ {
			idx := rng.intn(len(pool) - int(j))
			out[j] = pool[idx]
			pool[idx] = pool[len(pool)-1-int(j)]
		}
		return &object.List{V: out}, nil
	}})

	// ── Discrete distributions ────────────────────────────────────────────────

	m.Dict.SetStr("binomialvariate", &object.BuiltinFunc{Name: "binomialvariate", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		n := int64(1)
		p := 0.5
		if len(a) >= 1 {
			if v, ok := toInt64(a[0]); ok {
				n = v
			}
		}
		if len(a) >= 2 {
			if v, ok := toFloat64Any(a[1]); ok {
				p = v
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("n"); ok {
				if nv, ok2 := toInt64(v); ok2 {
					n = nv
				}
			}
			if v, ok := kw.GetStr("p"); ok {
				if pv, ok2 := toFloat64Any(v); ok2 {
					p = pv
				}
			}
		}
		if n < 0 {
			return nil, object.Errorf(i.valueErr, "binomialvariate: n must be >= 0")
		}
		if p < 0 || p > 1 {
			return nil, object.Errorf(i.valueErr, "binomialvariate: p must be in [0,1]")
		}
		// Direct method (acceptable for small n).
		count := int64(0)
		for k := int64(0); k < n; k++ {
			if rng.float64() < p {
				count++
			}
		}
		return object.NewInt(count), nil
	}})

	// ── Real-valued distributions ─────────────────────────────────────────────

	m.Dict.SetStr("random", &object.BuiltinFunc{Name: "random", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Float{V: rng.float64()}, nil
	}})

	m.Dict.SetStr("uniform", &object.BuiltinFunc{Name: "uniform", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "uniform() requires 2 arguments")
		}
		lo, _ := toFloat64Any(a[0])
		hi, _ := toFloat64Any(a[1])
		return &object.Float{V: lo + rng.float64()*(hi-lo)}, nil
	}})

	m.Dict.SetStr("triangular", &object.BuiltinFunc{Name: "triangular", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		low, high, mode := 0.0, 1.0, math.NaN()
		if len(a) >= 1 {
			low, _ = toFloat64Any(a[0])
		}
		if len(a) >= 2 {
			high, _ = toFloat64Any(a[1])
		}
		if len(a) >= 3 {
			mode, _ = toFloat64Any(a[2])
		}
		if kw != nil {
			if v, ok := kw.GetStr("low"); ok {
				low, _ = toFloat64Any(v)
			}
			if v, ok := kw.GetStr("high"); ok {
				high, _ = toFloat64Any(v)
			}
			if v, ok := kw.GetStr("mode"); ok {
				mode, _ = toFloat64Any(v)
			}
		}
		u := rng.float64()
		var c float64
		if high == low {
			return &object.Float{V: low}, nil
		}
		if math.IsNaN(mode) {
			c = 0.5
		} else {
			c = (mode - low) / (high - low)
		}
		var result float64
		if u > c {
			u = 1.0 - u
			c = 1.0 - c
			low, high = high, low
		}
		result = low + (high-low)*math.Sqrt(u/c)
		return &object.Float{V: result}, nil
	}})

	m.Dict.SetStr("expovariate", &object.BuiltinFunc{Name: "expovariate", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		lambd := 1.0
		if len(a) >= 1 {
			lambd, _ = toFloat64Any(a[0])
		}
		if kw != nil {
			if v, ok := kw.GetStr("lambd"); ok {
				lambd, _ = toFloat64Any(v)
			}
		}
		return &object.Float{V: -math.Log(1.0-rng.float64()) / lambd}, nil
	}})

	m.Dict.SetStr("gammavariate", &object.BuiltinFunc{Name: "gammavariate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "gammavariate() requires alpha and beta")
		}
		alpha, _ := toFloat64Any(a[0])
		beta, _ := toFloat64Any(a[1])
		if alpha <= 0 || beta <= 0 {
			return nil, object.Errorf(i.valueErr, "gammavariate: alpha and beta must be > 0")
		}
		return &object.Float{V: gammavariate0(alpha) * beta}, nil
	}})

	m.Dict.SetStr("gauss", &object.BuiltinFunc{Name: "gauss", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		mu, sigma := 0.0, 1.0
		if len(a) >= 1 {
			mu, _ = toFloat64Any(a[0])
		}
		if len(a) >= 2 {
			sigma, _ = toFloat64Any(a[1])
		}
		if kw != nil {
			if v, ok := kw.GetStr("mu"); ok {
				mu, _ = toFloat64Any(v)
			}
			if v, ok := kw.GetStr("sigma"); ok {
				sigma, _ = toFloat64Any(v)
			}
		}
		// Use Box-Muller with cached second variate (CPython gauss behavior).
		var z float64
		if !math.IsNaN(rng.gaussNext) {
			z = rng.gaussNext
			rng.gaussNext = math.NaN()
		} else {
			x2pi := rng.float64() * 2 * math.Pi
			g2rad := math.Sqrt(-2.0 * math.Log(1.0-rng.float64()))
			z = math.Cos(x2pi) * g2rad
			rng.gaussNext = math.Sin(x2pi) * g2rad
		}
		return &object.Float{V: mu + z*sigma}, nil
	}})

	m.Dict.SetStr("normalvariate", &object.BuiltinFunc{Name: "normalvariate", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		mu, sigma := 0.0, 1.0
		if len(a) >= 1 {
			mu, _ = toFloat64Any(a[0])
		}
		if len(a) >= 2 {
			sigma, _ = toFloat64Any(a[1])
		}
		if kw != nil {
			if v, ok := kw.GetStr("mu"); ok {
				mu, _ = toFloat64Any(v)
			}
			if v, ok := kw.GetStr("sigma"); ok {
				sigma, _ = toFloat64Any(v)
			}
		}
		return &object.Float{V: mu + normalvariate0()*sigma}, nil
	}})

	m.Dict.SetStr("lognormvariate", &object.BuiltinFunc{Name: "lognormvariate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "lognormvariate() requires mu and sigma")
		}
		mu, _ := toFloat64Any(a[0])
		sigma, _ := toFloat64Any(a[1])
		return &object.Float{V: math.Exp(mu + normalvariate0()*sigma)}, nil
	}})

	m.Dict.SetStr("betavariate", &object.BuiltinFunc{Name: "betavariate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "betavariate() requires alpha and beta")
		}
		alpha, _ := toFloat64Any(a[0])
		beta, _ := toFloat64Any(a[1])
		if alpha <= 0 || beta <= 0 {
			return nil, object.Errorf(i.valueErr, "betavariate: alpha and beta must be > 0")
		}
		y := gammavariate0(alpha)
		if y == 0 {
			return &object.Float{V: 0.0}, nil
		}
		return &object.Float{V: y / (y + gammavariate0(beta))}, nil
	}})

	m.Dict.SetStr("vonmisesvariate", &object.BuiltinFunc{Name: "vonmisesvariate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "vonmisesvariate() requires mu and kappa")
		}
		mu, _ := toFloat64Any(a[0])
		kappa, _ := toFloat64Any(a[1])
		const twoPi = 2 * math.Pi
		if kappa <= 1e-6 {
			return &object.Float{V: twoPi * rng.float64()}, nil
		}
		s := 0.5 / kappa
		r := s + math.Sqrt(1.0+s*s)
		for {
			u1 := rng.float64()
			z := math.Cos(math.Pi * u1)
			d := z / (r + z)
			u2 := rng.float64()
			if u2 < 1.0-d*d || u2 <= (1.0-d)*math.Exp(d) {
				q := 1.0 / r
				f := (q + z) / (1.0 + q*z)
				u3 := rng.float64()
				var theta float64
				if u3 > 0.5 {
					theta = math.Mod(mu+math.Acos(f), twoPi)
				} else {
					theta = math.Mod(mu-math.Acos(f), twoPi)
				}
				if theta < 0 {
					theta += twoPi
				}
				return &object.Float{V: theta}, nil
			}
		}
	}})

	m.Dict.SetStr("paretovariate", &object.BuiltinFunc{Name: "paretovariate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "paretovariate() requires alpha")
		}
		alpha, _ := toFloat64Any(a[0])
		u := 1.0 - rng.float64()
		return &object.Float{V: math.Pow(u, -1.0/alpha)}, nil
	}})

	m.Dict.SetStr("weibullvariate", &object.BuiltinFunc{Name: "weibullvariate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "weibullvariate() requires alpha and beta")
		}
		alpha, _ := toFloat64Any(a[0])
		beta, _ := toFloat64Any(a[1])
		u := 1.0 - rng.float64()
		return &object.Float{V: alpha * math.Pow(-math.Log(u), 1.0/beta)}, nil
	}})

	// ── Random class ──────────────────────────────────────────────────────────
	// Exposes a class whose instances each have their own PRNG state.

	randomCls := &object.Class{Name: "Random", Dict: object.NewDict()}

	randomCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		var seed int64
		if len(a) >= 2 && a[1] != object.None {
			if n, ok2 := toInt64(a[1]); ok2 {
				seed = n
			}
		} else {
			var b [8]byte
			_, _ = rand.Read(b[:])
			seed = int64(binary.LittleEndian.Uint64(b[:]))
		}
		r := newSplitmix64(seed)
		self.Dict.SetStr("_rng_state", object.NewInt(int64(r.state)))
		self.Dict.SetStr("_gauss_next", object.None)
		return object.None, nil
	}})

	// Helper to get/update the instance's PRNG.
	getRng := func(self *object.Instance) *splitmix64 {
		sv, _ := self.Dict.GetStr("_rng_state")
		n, _ := toInt64(sv)
		r := &splitmix64{state: uint64(n), gaussNext: math.NaN()}
		if gv, ok := self.Dict.GetStr("_gauss_next"); ok && gv != object.None {
			if f, ok2 := toFloat64Any(gv); ok2 {
				r.gaussNext = f
			}
		}
		return r
	}
	saveRng := func(self *object.Instance, r *splitmix64) {
		self.Dict.SetStr("_rng_state", object.NewInt(int64(r.state)))
		if math.IsNaN(r.gaussNext) {
			self.Dict.SetStr("_gauss_next", object.None)
		} else {
			self.Dict.SetStr("_gauss_next", &object.Float{V: r.gaussNext})
		}
	}

	addInstanceMethod := func(name string, fn func(r *splitmix64, a []object.Object, kw *object.Dict) (object.Object, error)) {
		randomCls.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "%s() requires self", name)
			}
			self, ok := a[0].(*object.Instance)
			if !ok {
				return nil, object.Errorf(i.typeErr, "%s() requires a Random instance", name)
			}
			r := getRng(self)
			result, err := fn(r, a[1:], kw)
			saveRng(self, r)
			return result, err
		}})
	}

	addInstanceMethod("seed", func(r *splitmix64, a []object.Object, _ *object.Dict) (object.Object, error) {
		var s int64
		if len(a) == 0 || a[0] == object.None {
			var b [8]byte
			_, _ = rand.Read(b[:])
			s = int64(binary.LittleEndian.Uint64(b[:]))
		} else if n, ok := toInt64(a[0]); ok {
			s = n
		} else if f, ok := toFloat64Any(a[0]); ok {
			s = int64(math.Float64bits(f))
		}
		r.state = uint64(s)
		r.gaussNext = math.NaN()
		for k := 0; k < 20; k++ {
			r.next()
		}
		return object.None, nil
	})

	addInstanceMethod("random", func(r *splitmix64, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Float{V: r.float64()}, nil
	})
	addInstanceMethod("uniform", func(r *splitmix64, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "uniform() requires 2 args")
		}
		lo, _ := toFloat64Any(a[0])
		hi, _ := toFloat64Any(a[1])
		return &object.Float{V: lo + r.float64()*(hi-lo)}, nil
	})
	addInstanceMethod("randint", func(r *splitmix64, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "randint() requires 2 args")
		}
		lo, _ := toInt64(a[0])
		hi, _ := toInt64(a[1])
		if hi < lo {
			return nil, object.Errorf(i.valueErr, "empty range for randint")
		}
		return object.NewInt(lo + r.int63n(hi-lo+1)), nil
	})
	addInstanceMethod("choice", func(r *splitmix64, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "choice() requires a sequence")
		}
		items, err := iterate(i, a[0])
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, object.Errorf(i.indexErr, "Cannot choose from an empty sequence")
		}
		return items[r.intn(len(items))], nil
	})
	addInstanceMethod("shuffle", func(r *splitmix64, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "shuffle() requires a list")
		}
		lst, ok := a[0].(*object.List)
		if !ok {
			return nil, object.Errorf(i.typeErr, "shuffle() requires a list")
		}
		for k := len(lst.V) - 1; k > 0; k-- {
			j := r.intn(k + 1)
			lst.V[k], lst.V[j] = lst.V[j], lst.V[k]
		}
		return object.None, nil
	})
	addInstanceMethod("gauss", func(r *splitmix64, a []object.Object, kw *object.Dict) (object.Object, error) {
		mu, sigma := 0.0, 1.0
		if len(a) >= 1 {
			mu, _ = toFloat64Any(a[0])
		}
		if len(a) >= 2 {
			sigma, _ = toFloat64Any(a[1])
		}
		var z float64
		if !math.IsNaN(r.gaussNext) {
			z = r.gaussNext
			r.gaussNext = math.NaN()
		} else {
			x2pi := r.float64() * 2 * math.Pi
			g2rad := math.Sqrt(-2.0 * math.Log(1.0-r.float64()))
			z = math.Cos(x2pi) * g2rad
			r.gaussNext = math.Sin(x2pi) * g2rad
		}
		return &object.Float{V: mu + z*sigma}, nil
	})
	addInstanceMethod("normalvariate", func(r *splitmix64, a []object.Object, _ *object.Dict) (object.Object, error) {
		mu, sigma := 0.0, 1.0
		if len(a) >= 1 {
			mu, _ = toFloat64Any(a[0])
		}
		if len(a) >= 2 {
			sigma, _ = toFloat64Any(a[1])
		}
		// Kinderman-Monahan using r's own PRNG.
		const NV_MAGICCONST = 1.7155277699214135
		for {
			u1 := r.float64()
			u2 := 1.0 - r.float64()
			z := NV_MAGICCONST * (u1 - 0.5) / u2
			zz := z * z / 4.0
			if zz <= -math.Log(u2) {
				return &object.Float{V: mu + z*sigma}, nil
			}
		}
	})

	addInstanceMethod("getstate", func(r *splitmix64, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var gn object.Object = object.None
		if !math.IsNaN(r.gaussNext) {
			gn = &object.Float{V: r.gaussNext}
		}
		return &object.Tuple{V: []object.Object{
			object.NewInt(3),
			&object.Tuple{V: []object.Object{object.NewInt(int64(r.state))}},
			gn,
		}}, nil
	})
	addInstanceMethod("setstate", func(r *splitmix64, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "setstate() requires a state")
		}
		tup, ok := a[0].(*object.Tuple)
		if !ok || len(tup.V) < 3 {
			return object.None, nil
		}
		inner, ok2 := tup.V[1].(*object.Tuple)
		if ok2 && len(inner.V) >= 1 {
			if n, ok3 := toInt64(inner.V[0]); ok3 {
				r.state = uint64(n)
			}
		}
		if tup.V[2] == object.None {
			r.gaussNext = math.NaN()
		} else if f, ok3 := toFloat64Any(tup.V[2]); ok3 {
			r.gaussNext = f
		}
		return object.None, nil
	})

	m.Dict.SetStr("Random", randomCls)

	// ── SystemRandom class ────────────────────────────────────────────────────
	// Uses crypto/rand for each draw — not seedable.

	sysrndCls := &object.Class{Name: "SystemRandom", Dict: object.NewDict()}
	sysrndCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	sysrndCls.Dict.SetStr("random", &object.BuiltinFunc{Name: "random", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var b [8]byte
		_, _ = rand.Read(b[:])
		return &object.Float{V: float64(binary.LittleEndian.Uint64(b[:])>>11) * (1.0 / (1 << 53))}, nil
	}})
	sysrndCls.Dict.SetStr("getrandbits", &object.BuiltinFunc{Name: "getrandbits", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.NewInt(0), nil
		}
		k, _ := toInt64(a[1])
		if k <= 0 {
			return object.NewInt(0), nil
		}
		words := (k + 63) / 64
		result := new(object.Int)
		for j := int64(0); j < words; j++ {
			var b [8]byte
			_, _ = rand.Read(b[:])
			w := binary.LittleEndian.Uint64(b[:])
			result.V.Lsh(&result.V, 64)
			var wInt object.Int
			wInt.V.SetUint64(w)
			result.V.Or(&result.V, &wInt.V)
		}
		// Mask top to k bits.
		topBits := k % 64
		if topBits != 0 {
			mask := new(object.Int)
			mask.V.SetUint64((1 << topBits) - 1)
			result.V.And(&result.V, &mask.V)
		}
		return result, nil
	}})
	m.Dict.SetStr("SystemRandom", sysrndCls)

	return m
}

func randSign(n int64) int64 {
	if n > 0 {
		return 1
	}
	if n < 0 {
		return -1
	}
	return 0
}

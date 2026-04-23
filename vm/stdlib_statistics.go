package vm

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sort"

	"github.com/tamnd/goipy/object"
)

// buildStatistics constructs the full Python 3.14 statistics module.
func (i *Interp) buildStatistics() *object.Module {
	m := &object.Module{Name: "statistics", Dict: object.NewDict()}

	// StatisticsError is a subclass of ValueError.
	statsErr := &object.Class{Name: "StatisticsError", Bases: []*object.Class{i.valueErr}, Dict: object.NewDict()}
	m.Dict.SetStr("StatisticsError", statsErr)

	errorf := func(format string, args ...any) error {
		return object.Errorf(statsErr, format, args...)
	}

	// collectFloats extracts []float64 from a single iterable argument.
	collectFloats := func(a []object.Object, fn string) ([]float64, bool, error) {
		if len(a) == 0 {
			return nil, false, object.Errorf(i.typeErr, "%s() missing argument", fn)
		}
		seq, err := i.iterObjects(a[:1], fn)
		if err != nil {
			return nil, false, err
		}
		xs := make([]float64, 0, len(seq))
		allInt := true
		for _, o := range seq {
			switch o.(type) {
			case *object.Int, *object.Bool:
			default:
				allInt = false
			}
			f, ok := toFloat64Any(o)
			if !ok {
				return nil, false, object.Errorf(i.typeErr, "%s() requires numeric data", fn)
			}
			xs = append(xs, f)
		}
		return xs, allInt, nil
	}

	sliceMean := func(xs []float64) float64 {
		s := 0.0
		for _, x := range xs {
			s += x
		}
		return s / float64(len(xs))
	}

	// --- mean ---
	m.Dict.SetStr("mean", &object.BuiltinFunc{Name: "mean", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		xs, allInt, err := collectFloats(a, "mean")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, errorf("mean requires at least one data point")
		}
		avg := sliceMean(xs)
		if allInt && avg == math.Trunc(avg) && !math.IsInf(avg, 0) {
			return object.IntFromBig(big.NewInt(int64(avg))), nil
		}
		return &object.Float{V: avg}, nil
	}})

	// --- fmean (with optional weights keyword) ---
	m.Dict.SetStr("fmean", &object.BuiltinFunc{Name: "fmean", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		xs, _, err := collectFloats(a, "fmean")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, errorf("fmean requires at least one data point")
		}
		if kw != nil {
			if wv, ok := kw.GetStr("weights"); ok {
				ws, err := i.iterObjects([]object.Object{wv}, "fmean")
				if err != nil {
					return nil, err
				}
				if len(ws) != len(xs) {
					return nil, errorf("fmean: weights must have the same length as data")
				}
				wsum, wdata := 0.0, 0.0
				for k, w := range ws {
					wf, ok := toFloat64Any(w)
					if !ok {
						return nil, object.Errorf(i.typeErr, "fmean() weights must be numeric")
					}
					if wf < 0 {
						return nil, errorf("fmean: weights must be non-negative")
					}
					wsum += wf
					wdata += xs[k] * wf
				}
				if wsum == 0 {
					return nil, errorf("fmean: total weight must be positive")
				}
				return &object.Float{V: wdata / wsum}, nil
			}
		}
		s := 0.0
		for _, x := range xs {
			s += x
		}
		return &object.Float{V: s / float64(len(xs))}, nil
	}})

	// --- geometric_mean ---
	m.Dict.SetStr("geometric_mean", &object.BuiltinFunc{Name: "geometric_mean", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		xs, _, err := collectFloats(a, "geometric_mean")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, errorf("geometric_mean requires at least one data point")
		}
		logSum := 0.0
		for _, x := range xs {
			if x <= 0 {
				return nil, errorf("geometric_mean requires positive values")
			}
			logSum += math.Log(x)
		}
		return &object.Float{V: math.Exp(logSum / float64(len(xs)))}, nil
	}})

	// --- harmonic_mean (with optional weights keyword) ---
	m.Dict.SetStr("harmonic_mean", &object.BuiltinFunc{Name: "harmonic_mean", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		xs, _, err := collectFloats(a, "harmonic_mean")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, errorf("harmonic_mean requires at least one data point")
		}
		for _, x := range xs {
			if x < 0 {
				return nil, errorf("harmonic_mean requires non-negative values")
			}
		}
		if kw != nil {
			if wv, ok := kw.GetStr("weights"); ok {
				ws, err := i.iterObjects([]object.Object{wv}, "harmonic_mean")
				if err != nil {
					return nil, err
				}
				if len(ws) != len(xs) {
					return nil, errorf("harmonic_mean: weights must have the same length as data")
				}
				wsum, recSum := 0.0, 0.0
				for k, w := range ws {
					wf, ok := toFloat64Any(w)
					if !ok {
						return nil, object.Errorf(i.typeErr, "harmonic_mean() weights must be numeric")
					}
					if wf < 0 {
						return nil, errorf("harmonic_mean: weights must be non-negative")
					}
					wsum += wf
					if xs[k] == 0 {
						recSum = math.Inf(1)
					} else {
						recSum += wf / xs[k]
					}
				}
				if wsum <= 0 {
					return nil, errorf("harmonic_mean: total weight must be positive")
				}
				return &object.Float{V: wsum / recSum}, nil
			}
		}
		recSum := 0.0
		for _, x := range xs {
			if x == 0 {
				return &object.Float{V: 0}, nil
			}
			recSum += 1 / x
		}
		return &object.Float{V: float64(len(xs)) / recSum}, nil
	}})

	// --- median ---
	m.Dict.SetStr("median", &object.BuiltinFunc{Name: "median", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		xs, allInt, err := collectFloats(a, "median")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, errorf("median requires at least one data point")
		}
		sort.Float64s(xs)
		n := len(xs)
		if n%2 == 1 {
			v := xs[n/2]
			if allInt {
				return object.IntFromBig(big.NewInt(int64(v))), nil
			}
			return &object.Float{V: v}, nil
		}
		return &object.Float{V: (xs[n/2-1] + xs[n/2]) / 2}, nil
	}})

	// --- median_low ---
	m.Dict.SetStr("median_low", &object.BuiltinFunc{Name: "median_low", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		xs, allInt, err := collectFloats(a, "median_low")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, errorf("median_low requires at least one data point")
		}
		sort.Float64s(xs)
		n := len(xs)
		var v float64
		if n%2 == 1 {
			v = xs[n/2]
		} else {
			v = xs[n/2-1]
		}
		if allInt {
			return object.IntFromBig(big.NewInt(int64(v))), nil
		}
		return &object.Float{V: v}, nil
	}})

	// --- median_high ---
	m.Dict.SetStr("median_high", &object.BuiltinFunc{Name: "median_high", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		xs, allInt, err := collectFloats(a, "median_high")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, errorf("median_high requires at least one data point")
		}
		sort.Float64s(xs)
		v := xs[len(xs)/2]
		if allInt {
			return object.IntFromBig(big.NewInt(int64(v))), nil
		}
		return &object.Float{V: v}, nil
	}})

	// --- median_grouped ---
	m.Dict.SetStr("median_grouped", &object.BuiltinFunc{Name: "median_grouped", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		xs, _, err := collectFloats(a, "median_grouped")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, errorf("median_grouped requires at least one data point")
		}
		interval := 1.0
		if len(a) >= 2 {
			if f, ok := toFloat64Any(a[1]); ok {
				interval = f
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("interval"); ok {
				if f, ok2 := toFloat64Any(v); ok2 {
					interval = f
				}
			}
		}
		sort.Float64s(xs)
		n := len(xs)
		x := xs[n/2]
		count, l := 0, 0
		for _, v := range xs {
			if v == x {
				count++
			} else if v < x {
				l++
			}
		}
		result := x - interval/2 + interval*float64(n/2-l)/float64(count)
		return &object.Float{V: result}, nil
	}})

	// --- mode ---
	m.Dict.SetStr("mode", &object.BuiltinFunc{Name: "mode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		seq, err := i.iterObjects(a, "mode")
		if err != nil {
			return nil, err
		}
		if len(seq) == 0 {
			return nil, errorf("mode requires at least one data point")
		}
		order := []object.Object{}
		counts := map[string]int{}
		reps := map[string]object.Object{}
		for _, o := range seq {
			k := object.Repr(o)
			if _, seen := counts[k]; !seen {
				order = append(order, o)
				reps[k] = o
			}
			counts[k]++
		}
		var best object.Object = order[0]
		bestK := object.Repr(best)
		for _, o := range order {
			k := object.Repr(o)
			if counts[k] > counts[bestK] {
				best = reps[k]
				bestK = k
			}
		}
		return best, nil
	}})

	// --- multimode ---
	m.Dict.SetStr("multimode", &object.BuiltinFunc{Name: "multimode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		seq, err := i.iterObjects(a, "multimode")
		if err != nil {
			return nil, err
		}
		if len(seq) == 0 {
			return &object.List{V: nil}, nil
		}
		order := []object.Object{}
		counts := map[string]int{}
		reps := map[string]object.Object{}
		for _, o := range seq {
			k := object.Repr(o)
			if _, seen := counts[k]; !seen {
				order = append(order, o)
				reps[k] = o
			}
			counts[k]++
		}
		best := 0
		for _, o := range order {
			if c := counts[object.Repr(o)]; c > best {
				best = c
			}
		}
		var out []object.Object
		for _, o := range order {
			if counts[object.Repr(o)] == best {
				out = append(out, reps[object.Repr(o)])
			}
		}
		return &object.List{V: out}, nil
	}})

	// --- quantiles ---
	m.Dict.SetStr("quantiles", &object.BuiltinFunc{Name: "quantiles", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		xs, _, err := collectFloats(a, "quantiles")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, errorf("quantiles requires at least one data point")
		}
		n := 4
		method := "exclusive"
		if kw != nil {
			if v, ok := kw.GetStr("n"); ok {
				if iv, ok2 := toInt64(v); ok2 {
					n = int(iv)
				}
			}
			if v, ok := kw.GetStr("method"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					method = s.V
				}
			}
		}
		if n < 1 {
			return nil, errorf("n must be at least 1")
		}
		sort.Float64s(xs)
		L := len(xs)
		out := make([]object.Object, n-1)
		for k := 1; k < n; k++ {
			var pos float64
			if method == "inclusive" {
				pos = float64(k) * float64(L-1) / float64(n)
			} else {
				pos = float64(k)*float64(L+1)/float64(n) - 1
			}
			lo := int(math.Floor(pos))
			if lo < 0 {
				lo = 0
			}
			if lo >= L-1 {
				lo = L - 2
			}
			frac := pos - float64(lo)
			out[k-1] = &object.Float{V: xs[lo] + frac*(xs[lo+1]-xs[lo])}
		}
		return &object.List{V: out}, nil
	}})

	// variance helper; muKey is the keyword name for the precomputed mean ("mu" or "xbar").
	// Returns (value, allInt, muWasFloat, error) where muWasFloat=true when the caller supplied
	// a float mu/xbar (which forces a float return type even for integer data).
	statsVariance := func(a []object.Object, kw *object.Dict, sample bool, fn, muKey string) (float64, bool, bool, error) {
		xs, allInt, err := collectFloats(a, fn)
		if err != nil {
			return 0, false, false, err
		}
		minN := 1
		if sample {
			minN = 2
		}
		if len(xs) < minN {
			return 0, false, false, errorf("%s requires at least %d data points", fn, minN)
		}
		mu := sliceMean(xs)
		muWasFloat := false
		// Accept mu/xbar as second positional or keyword argument.
		if len(a) >= 2 && a[1] != object.None {
			if f, ok := toFloat64Any(a[1]); ok {
				mu = f
				if _, isInt := a[1].(*object.Int); !isInt {
					if _, isBool := a[1].(*object.Bool); !isBool {
						muWasFloat = true
					}
				}
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr(muKey); ok && v != object.None {
				if f, ok2 := toFloat64Any(v); ok2 {
					mu = f
					if _, isInt := v.(*object.Int); !isInt {
						if _, isBool := v.(*object.Bool); !isBool {
							muWasFloat = true
						}
					}
				}
			}
		}
		sq := 0.0
		for _, x := range xs {
			d := x - mu
			sq += d * d
		}
		div := float64(len(xs))
		if sample {
			div--
		}
		return sq / div, allInt, muWasFloat, nil
	}

	// --- pvariance ---
	m.Dict.SetStr("pvariance", &object.BuiltinFunc{Name: "pvariance", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		v, allInt, muWasFloat, err := statsVariance(a, kw, false, "pvariance", "mu")
		if err != nil {
			return nil, err
		}
		if !muWasFloat && allInt && v == math.Trunc(v) && !math.IsInf(v, 0) {
			return object.IntFromBig(big.NewInt(int64(v))), nil
		}
		return &object.Float{V: v}, nil
	}})

	// --- variance ---
	m.Dict.SetStr("variance", &object.BuiltinFunc{Name: "variance", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		v, allInt, muWasFloat, err := statsVariance(a, kw, true, "variance", "xbar")
		if err != nil {
			return nil, err
		}
		if !muWasFloat && allInt && v == math.Trunc(v) && !math.IsInf(v, 0) {
			return object.IntFromBig(big.NewInt(int64(v))), nil
		}
		return &object.Float{V: v}, nil
	}})

	// --- pstdev ---
	m.Dict.SetStr("pstdev", &object.BuiltinFunc{Name: "pstdev", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		v, _, _, err := statsVariance(a, kw, false, "pstdev", "mu")
		if err != nil {
			return nil, err
		}
		return &object.Float{V: math.Sqrt(v)}, nil
	}})

	// --- stdev ---
	m.Dict.SetStr("stdev", &object.BuiltinFunc{Name: "stdev", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		v, _, _, err := statsVariance(a, kw, true, "stdev", "xbar")
		if err != nil {
			return nil, err
		}
		return &object.Float{V: math.Sqrt(v)}, nil
	}})

	// --- covariance ---
	m.Dict.SetStr("covariance", &object.BuiltinFunc{Name: "covariance", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "covariance() requires two arguments")
		}
		xs, _, err := collectFloats(a[:1], "covariance")
		if err != nil {
			return nil, err
		}
		ys, _, err := collectFloats(a[1:2], "covariance")
		if err != nil {
			return nil, err
		}
		if len(xs) != len(ys) {
			return nil, errorf("covariance: x and y must have the same length")
		}
		if len(xs) < 2 {
			return nil, errorf("covariance requires at least 2 data points")
		}
		mx, my := sliceMean(xs), sliceMean(ys)
		cov := 0.0
		for k := range xs {
			cov += (xs[k] - mx) * (ys[k] - my)
		}
		return &object.Float{V: cov / float64(len(xs)-1)}, nil
	}})

	// --- correlation ---
	m.Dict.SetStr("correlation", &object.BuiltinFunc{Name: "correlation", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "correlation() requires two arguments")
		}
		xs, _, err := collectFloats(a[:1], "correlation")
		if err != nil {
			return nil, err
		}
		ys, _, err := collectFloats(a[1:2], "correlation")
		if err != nil {
			return nil, err
		}
		if len(xs) != len(ys) {
			return nil, errorf("correlation: x and y must have the same length")
		}
		if len(xs) < 2 {
			return nil, errorf("correlation requires at least 2 data points")
		}
		method := "linear"
		if kw != nil {
			if v, ok := kw.GetStr("method"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					method = s.V
				}
			}
		}
		cxs, cys := xs, ys
		if method == "ranked" {
			cxs = statsRankData(xs)
			cys = statsRankData(ys)
		}
		mx, my := sliceMean(cxs), sliceMean(cys)
		cov, vx, vy := 0.0, 0.0, 0.0
		for k := range cxs {
			dx, dy := cxs[k]-mx, cys[k]-my
			cov += dx * dy
			vx += dx * dx
			vy += dy * dy
		}
		if vx == 0 || vy == 0 {
			return nil, errorf("correlation: variance of input is zero")
		}
		return &object.Float{V: cov / math.Sqrt(vx*vy)}, nil
	}})

	// --- linear_regression ---
	m.Dict.SetStr("linear_regression", &object.BuiltinFunc{Name: "linear_regression", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "linear_regression() requires two arguments")
		}
		xs, _, err := collectFloats(a[:1], "linear_regression")
		if err != nil {
			return nil, err
		}
		ys, _, err := collectFloats(a[1:2], "linear_regression")
		if err != nil {
			return nil, err
		}
		if len(xs) != len(ys) {
			return nil, errorf("linear_regression: x and y must have the same length")
		}
		if len(xs) < 2 {
			return nil, errorf("linear_regression requires at least 2 data points")
		}
		proportional := false
		if kw != nil {
			if v, ok := kw.GetStr("proportional"); ok {
				proportional = object.Truthy(v)
			}
		}
		var slope, intercept float64
		if proportional {
			sxy, sxx := 0.0, 0.0
			for k := range xs {
				sxy += xs[k] * ys[k]
				sxx += xs[k] * xs[k]
			}
			if sxx == 0 {
				return nil, errorf("linear_regression: x is constant (all zeros)")
			}
			slope = sxy / sxx
		} else {
			mx, my := sliceMean(xs), sliceMean(ys)
			sxy, sxx := 0.0, 0.0
			for k := range xs {
				dx := xs[k] - mx
				sxy += dx * (ys[k] - my)
				sxx += dx * dx
			}
			if sxx == 0 {
				return nil, errorf("linear_regression: x is constant")
			}
			slope = sxy / sxx
			intercept = my - slope*mx
		}
		// Return a SimpleNamespace-like instance with slope and intercept.
		cls := &object.Class{Name: "LinearRegression", Dict: object.NewDict()}
		inst := &object.Instance{Class: cls, Dict: object.NewDict()}
		inst.Dict.SetStr("slope", &object.Float{V: slope})
		inst.Dict.SetStr("intercept", &object.Float{V: intercept})
		return inst, nil
	}})

	// --- kde ---
	m.Dict.SetStr("kde", &object.BuiltinFunc{Name: "kde", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "kde() requires data argument")
		}
		data, _, err := collectFloats(a[:1], "kde")
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			return nil, errorf("kde requires at least one data point")
		}
		// h can be 2nd positional arg or keyword arg.
		var h float64
		if len(a) >= 2 {
			var ok bool
			h, ok = toFloat64Any(a[1])
			if !ok {
				return nil, errorf("kde: bandwidth h must be a positive number")
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("h"); ok2 {
				if f, ok3 := toFloat64Any(v); ok3 {
					h = f
				}
			}
		}
		if h <= 0 {
			return nil, errorf("kde: bandwidth h must be a positive number")
		}
		kernel := "normal"
		cumulative := false
		if len(a) >= 3 {
			if s, ok2 := a[2].(*object.Str); ok2 {
				kernel = s.V
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("kernel"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					kernel = s.V
				}
			}
			if v, ok2 := kw.GetStr("cumulative"); ok2 {
				cumulative = object.Truthy(v)
			}
		}
		n := float64(len(data))
		if cumulative {
			kCDF, err2 := kdeKernelCDF(kernel)
			if err2 != nil {
				return nil, errorf("kde: unknown kernel %q", kernel)
			}
			return &object.BuiltinFunc{Name: "kde_cdf", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
				if len(args) == 0 {
					return nil, object.Errorf(i.typeErr, "kde cdf requires x argument")
				}
				x, ok := toFloat64Any(args[0])
				if !ok {
					return nil, object.Errorf(i.typeErr, "kde cdf requires numeric argument")
				}
				sum := 0.0
				for _, xi := range data {
					sum += kCDF((x - xi) / h)
				}
				return &object.Float{V: sum / n}, nil
			}}, nil
		}
		kfn, err2 := kdeKernelPDF(kernel)
		if err2 != nil {
			return nil, errorf("kde: unknown kernel %q", kernel)
		}
		return &object.BuiltinFunc{Name: "kde_pdf", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if len(args) == 0 {
				return nil, object.Errorf(i.typeErr, "kde pdf requires x argument")
			}
			x, ok := toFloat64Any(args[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "kde pdf requires numeric argument")
			}
			sum := 0.0
			for _, xi := range data {
				sum += kfn((x - xi) / h)
			}
			return &object.Float{V: sum / (n * h)}, nil
		}}, nil
	}})

	// --- kde_random ---
	m.Dict.SetStr("kde_random", &object.BuiltinFunc{Name: "kde_random", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "kde_random() requires data argument")
		}
		data, _, err := collectFloats(a[:1], "kde_random")
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			return nil, errorf("kde_random requires at least one data point")
		}
		var h float64
		if len(a) >= 2 {
			var ok bool
			h, ok = toFloat64Any(a[1])
			if !ok {
				return nil, errorf("kde_random: bandwidth h must be a positive number")
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("h"); ok2 {
				if f, ok3 := toFloat64Any(v); ok3 {
					h = f
				}
			}
		}
		if h <= 0 {
			return nil, errorf("kde_random: bandwidth h must be a positive number")
		}
		kernel := "normal"
		if len(a) >= 3 {
			if s, ok2 := a[2].(*object.Str); ok2 {
				kernel = s.V
			}
		}
		var seedVal int64
		hasSeed := false
		if kw != nil {
			if v, ok2 := kw.GetStr("kernel"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					kernel = s.V
				}
			}
			if v, ok2 := kw.GetStr("seed"); ok2 && v != object.None {
				if iv, ok3 := toInt64(v); ok3 {
					seedVal = iv
					hasSeed = true
				}
			}
		}
		if _, err2 := kdeKernelPDF(kernel); err2 != nil {
			return nil, errorf("kde_random: unknown kernel %q", kernel)
		}
		var rng *rand.Rand
		if hasSeed {
			rng = rand.New(rand.NewSource(seedVal)) //nolint:gosec
		} else {
			rng = rand.New(rand.NewSource(rand.Int63())) //nolint:gosec
		}
		return &object.BuiltinFunc{Name: "kde_random_sampler", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			xi := data[rng.Intn(len(data))]
			noise := kdeKernelSample(kernel, rng)
			return &object.Float{V: xi + h*noise}, nil
		}}, nil
	}})

	// --- NormalDist class ---
	ndCls := &object.Class{Name: "NormalDist", Dict: object.NewDict()}

	ndCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "NormalDist.__init__() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "NormalDist.__init__() requires self")
		}
		mu, sigma := 0.0, 1.0
		if len(a) >= 2 {
			if f, ok2 := toFloat64Any(a[1]); ok2 {
				mu = f
			}
		}
		if len(a) >= 3 {
			if f, ok2 := toFloat64Any(a[2]); ok2 {
				sigma = f
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("mu"); ok2 {
				if f, ok3 := toFloat64Any(v); ok3 {
					mu = f
				}
			}
			if v, ok2 := kw.GetStr("sigma"); ok2 {
				if f, ok3 := toFloat64Any(v); ok3 {
					sigma = f
				}
			}
		}
		if sigma < 0 {
			return nil, errorf("NormalDist: sigma must be non-negative")
		}
		inst.Dict.SetStr("_mu", &object.Float{V: mu})
		inst.Dict.SetStr("_sigma", &object.Float{V: sigma})
		return object.None, nil
	}})

	getNDMuSigma := func(inst *object.Instance) (float64, float64, bool) {
		mv, ok1 := inst.Dict.GetStr("_mu")
		sv, ok2 := inst.Dict.GetStr("_sigma")
		if !ok1 || !ok2 {
			return 0, 0, false
		}
		mu, ok3 := toFloat64Any(mv)
		sigma, ok4 := toFloat64Any(sv)
		return mu, sigma, ok3 && ok4
	}

	makeProp := func(getter func(*object.Instance) object.Object) *object.Property {
		return &object.Property{
			Fget: &object.BuiltinFunc{Name: "getter", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "property requires instance")
				}
				inst, ok := a[0].(*object.Instance)
				if !ok {
					return nil, object.Errorf(i.typeErr, "property requires instance")
				}
				return getter(inst), nil
			}},
		}
	}

	ndCls.Dict.SetStr("mean", makeProp(func(inst *object.Instance) object.Object {
		mu, _, _ := getNDMuSigma(inst)
		return &object.Float{V: mu}
	}))
	ndCls.Dict.SetStr("median", makeProp(func(inst *object.Instance) object.Object {
		mu, _, _ := getNDMuSigma(inst)
		return &object.Float{V: mu}
	}))
	ndCls.Dict.SetStr("mode", makeProp(func(inst *object.Instance) object.Object {
		mu, _, _ := getNDMuSigma(inst)
		return &object.Float{V: mu}
	}))
	ndCls.Dict.SetStr("stdev", makeProp(func(inst *object.Instance) object.Object {
		_, sigma, _ := getNDMuSigma(inst)
		return &object.Float{V: sigma}
	}))
	ndCls.Dict.SetStr("variance", makeProp(func(inst *object.Instance) object.Object {
		_, sigma, _ := getNDMuSigma(inst)
		return &object.Float{V: sigma * sigma}
	}))

	// NormalDist.from_samples classmethod
	ndCls.Dict.SetStr("from_samples", &object.BuiltinFunc{Name: "from_samples", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var dataArg object.Object
		if len(a) >= 2 {
			dataArg = a[1]
		} else if len(a) == 1 {
			dataArg = a[0]
		} else {
			return nil, object.Errorf(i.typeErr, "from_samples() requires data argument")
		}
		seq, err := i.iterObjects([]object.Object{dataArg}, "from_samples")
		if err != nil {
			return nil, err
		}
		if len(seq) < 2 {
			return nil, errorf("NormalDist.from_samples requires at least 2 data points")
		}
		xs := make([]float64, len(seq))
		for k, o := range seq {
			f, ok := toFloat64Any(o)
			if !ok {
				return nil, object.Errorf(i.typeErr, "from_samples() requires numeric data")
			}
			xs[k] = f
		}
		mu := sliceMean(xs)
		sq := 0.0
		for _, x := range xs {
			d := x - mu
			sq += d * d
		}
		sigma := math.Sqrt(sq / float64(len(xs)-1))
		inst := &object.Instance{Class: ndCls, Dict: object.NewDict()}
		inst.Dict.SetStr("_mu", &object.Float{V: mu})
		inst.Dict.SetStr("_sigma", &object.Float{V: sigma})
		return inst, nil
	}})

	// NormalDist.samples(n, *, seed=None)
	ndCls.Dict.SetStr("samples", &object.BuiltinFunc{Name: "samples", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "samples() requires self and n")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "samples() requires NormalDist instance")
		}
		mu, sigma, _ := getNDMuSigma(inst)
		n, ok2 := toInt64(a[1])
		if !ok2 || n < 0 {
			return nil, object.Errorf(i.typeErr, "samples() requires non-negative integer n")
		}
		var rng *rand.Rand
		if kw != nil {
			if sv, ok3 := kw.GetStr("seed"); ok3 && sv != object.None {
				if iv, ok4 := toInt64(sv); ok4 {
					rng = rand.New(rand.NewSource(iv)) //nolint:gosec
				}
			}
		}
		if rng == nil {
			rng = rand.New(rand.NewSource(rand.Int63())) //nolint:gosec
		}
		out := make([]object.Object, int(n))
		for k := range out {
			out[k] = &object.Float{V: mu + sigma*rng.NormFloat64()}
		}
		return &object.List{V: out}, nil
	}})

	// NormalDist.pdf(x)
	ndCls.Dict.SetStr("pdf", &object.BuiltinFunc{Name: "pdf", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "pdf() requires self and x")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "pdf() requires NormalDist instance")
		}
		mu, sigma, _ := getNDMuSigma(inst)
		x, ok2 := toFloat64Any(a[1])
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "pdf() requires numeric x")
		}
		if sigma == 0 {
			if x == mu {
				return &object.Float{V: math.Inf(1)}, nil
			}
			return &object.Float{V: 0}, nil
		}
		z := (x - mu) / sigma
		return &object.Float{V: math.Exp(-0.5*z*z) / (sigma * math.Sqrt(2*math.Pi))}, nil
	}})

	// NormalDist.cdf(x)
	ndCls.Dict.SetStr("cdf", &object.BuiltinFunc{Name: "cdf", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "cdf() requires self and x")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "cdf() requires NormalDist instance")
		}
		mu, sigma, _ := getNDMuSigma(inst)
		x, ok2 := toFloat64Any(a[1])
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "cdf() requires numeric x")
		}
		if sigma == 0 {
			if x < mu {
				return &object.Float{V: 0}, nil
			}
			return &object.Float{V: 1}, nil
		}
		return &object.Float{V: 0.5 * math.Erfc(-(x-mu)/(sigma*math.Sqrt2))}, nil
	}})

	// NormalDist.inv_cdf(p)
	ndCls.Dict.SetStr("inv_cdf", &object.BuiltinFunc{Name: "inv_cdf", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "inv_cdf() requires self and p")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "inv_cdf() requires NormalDist instance")
		}
		mu, sigma, _ := getNDMuSigma(inst)
		p, ok2 := toFloat64Any(a[1])
		if !ok2 || p < 0 || p > 1 {
			return nil, errorf("inv_cdf: p must be in [0, 1]")
		}
		return &object.Float{V: mu + sigma*statsNormalInvCDF(p)}, nil
	}})

	// NormalDist.overlap(other)
	ndCls.Dict.SetStr("overlap", &object.BuiltinFunc{Name: "overlap", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "overlap() requires self and other")
		}
		inst1, ok1 := a[0].(*object.Instance)
		inst2, ok2 := a[1].(*object.Instance)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "overlap() requires two NormalDist instances")
		}
		mu1, sigma1, _ := getNDMuSigma(inst1)
		mu2, sigma2, _ := getNDMuSigma(inst2)
		return &object.Float{V: statsNormalOverlap(mu1, sigma1, mu2, sigma2)}, nil
	}})

	// NormalDist.quantiles(n=4)
	ndCls.Dict.SetStr("quantiles", &object.BuiltinFunc{Name: "quantiles", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "quantiles() requires self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "quantiles() requires NormalDist instance")
		}
		mu, sigma, _ := getNDMuSigma(inst)
		n := 4
		if len(a) >= 2 {
			if iv, ok2 := toInt64(a[1]); ok2 {
				n = int(iv)
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("n"); ok2 {
				if iv, ok3 := toInt64(v); ok3 {
					n = int(iv)
				}
			}
		}
		if n < 1 {
			return nil, errorf("n must be at least 1")
		}
		out := make([]object.Object, n-1)
		for k := 1; k < n; k++ {
			p := float64(k) / float64(n)
			out[k-1] = &object.Float{V: mu + sigma*statsNormalInvCDF(p)}
		}
		return &object.List{V: out}, nil
	}})

	// NormalDist.zscore(x)
	ndCls.Dict.SetStr("zscore", &object.BuiltinFunc{Name: "zscore", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "zscore() requires self and x")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "zscore() requires NormalDist instance")
		}
		mu, sigma, _ := getNDMuSigma(inst)
		x, ok2 := toFloat64Any(a[1])
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "zscore() requires numeric x")
		}
		if sigma == 0 {
			return nil, errorf("zscore: sigma is zero")
		}
		return &object.Float{V: (x - mu) / sigma}, nil
	}})

	// NormalDist arithmetic operators
	ndArith := func(name string, fn func(mu1, sigma1, mu2, sigma2 float64) (float64, float64)) {
		ndCls.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.NotImplemented, nil
			}
			inst1, ok1 := a[0].(*object.Instance)
			if !ok1 {
				return object.NotImplemented, nil
			}
			mu1, sigma1, _ := getNDMuSigma(inst1)
			if inst2, ok2 := a[1].(*object.Instance); ok2 {
				mu2, sigma2, _ := getNDMuSigma(inst2)
				rmu, rsigma := fn(mu1, sigma1, mu2, sigma2)
				out := &object.Instance{Class: ndCls, Dict: object.NewDict()}
				out.Dict.SetStr("_mu", &object.Float{V: rmu})
				out.Dict.SetStr("_sigma", &object.Float{V: rsigma})
				return out, nil
			}
			if f, ok2 := toFloat64Any(a[1]); ok2 {
				rmu, rsigma := fn(mu1, sigma1, f, 0)
				out := &object.Instance{Class: ndCls, Dict: object.NewDict()}
				out.Dict.SetStr("_mu", &object.Float{V: rmu})
				out.Dict.SetStr("_sigma", &object.Float{V: rsigma})
				return out, nil
			}
			return object.NotImplemented, nil
		}})
	}

	ndArith("__add__", func(mu1, sigma1, mu2, sigma2 float64) (float64, float64) {
		return mu1 + mu2, math.Sqrt(sigma1*sigma1 + sigma2*sigma2)
	})
	ndArith("__radd__", func(mu1, sigma1, mu2, sigma2 float64) (float64, float64) {
		return mu2 + mu1, sigma1
	})
	ndArith("__sub__", func(mu1, sigma1, mu2, sigma2 float64) (float64, float64) {
		return mu1 - mu2, math.Sqrt(sigma1*sigma1 + sigma2*sigma2)
	})
	ndArith("__mul__", func(mu1, sigma1, mu2, _ float64) (float64, float64) {
		return mu1 * mu2, math.Abs(sigma1 * mu2)
	})
	ndArith("__rmul__", func(mu1, sigma1, mu2, _ float64) (float64, float64) {
		return mu2 * mu1, math.Abs(mu2 * sigma1)
	})

	ndCls.Dict.SetStr("__truediv__", &object.BuiltinFunc{Name: "__truediv__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.NotImplemented, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NotImplemented, nil
		}
		mu, sigma, _ := getNDMuSigma(inst)
		f, ok2 := toFloat64Any(a[1])
		if !ok2 {
			return object.NotImplemented, nil
		}
		if f == 0 {
			return nil, object.Errorf(i.zeroDivErr, "division by zero")
		}
		out := &object.Instance{Class: ndCls, Dict: object.NewDict()}
		out.Dict.SetStr("_mu", &object.Float{V: mu / f})
		out.Dict.SetStr("_sigma", &object.Float{V: math.Abs(sigma / f)})
		return out, nil
	}})

	ndCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return &object.Str{V: "NormalDist()"}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "NormalDist()"}, nil
		}
		mu, sigma, _ := getNDMuSigma(inst)
		return &object.Str{V: fmt.Sprintf("NormalDist(mu=%g, sigma=%g)", mu, sigma)}, nil
	}})

	m.Dict.SetStr("NormalDist", ndCls)

	return m
}

// iterObjects expands a single iterable argument into a []object.Object slice.
func (i *Interp) iterObjects(a []object.Object, fn string) ([]object.Object, error) {
	if len(a) == 0 {
		return nil, object.Errorf(i.typeErr, "%s() missing argument", fn)
	}
	switch v := a[0].(type) {
	case *object.List:
		return append([]object.Object(nil), v.V...), nil
	case *object.Tuple:
		return append([]object.Object(nil), v.V...), nil
	case *object.Range:
		return statsRangeToList(v), nil
	case *object.Set:
		return append([]object.Object(nil), v.Items()...), nil
	case *object.Frozenset:
		return append([]object.Object(nil), v.Items()...), nil
	case *object.Iter:
		var out []object.Object
		for {
			x, ok, err := v.Next()
			if err != nil {
				return nil, err
			}
			if !ok {
				break
			}
			out = append(out, x)
		}
		return out, nil
	}
	return nil, object.Errorf(i.typeErr, "%s() requires an iterable", fn)
}

// statsRangeToList converts a Range object to a flat slice of Int objects.
func statsRangeToList(r *object.Range) []object.Object {
	var out []object.Object
	if r.Step == 0 {
		return out
	}
	for v := r.Start; (r.Step > 0 && v < r.Stop) || (r.Step < 0 && v > r.Stop); v += r.Step {
		out = append(out, object.IntFromBig(big.NewInt(v)))
	}
	return out
}

// statsRankData returns fractional ranks (1-based, averaged for ties).
func statsRankData(xs []float64) []float64 {
	n := len(xs)
	idx := make([]int, n)
	for k := range idx {
		idx[k] = k
	}
	sort.Slice(idx, func(a, b int) bool { return xs[idx[a]] < xs[idx[b]] })
	ranks := make([]float64, n)
	for i := 0; i < n; {
		j := i + 1
		for j < n && xs[idx[j]] == xs[idx[i]] {
			j++
		}
		avg := float64(i+1+j) / 2.0
		for k := i; k < j; k++ {
			ranks[idx[k]] = avg
		}
		i = j
	}
	return ranks
}

// statsNormalInvCDF approximates the inverse normal CDF (Abramowitz & Stegun 26.2.17).
func statsNormalInvCDF(p float64) float64 {
	if p <= 0 {
		return math.Inf(-1)
	}
	if p >= 1 {
		return math.Inf(1)
	}
	if p == 0.5 {
		return 0
	}
	sign := 1.0
	pp := p
	if p < 0.5 {
		sign = -1
		pp = 1 - p
	}
	t := math.Sqrt(-2 * math.Log(1-pp))
	c := [3]float64{2.515517, 0.802853, 0.010328}
	d := [3]float64{1.432788, 0.189269, 0.001308}
	num := c[0] + t*(c[1]+t*c[2])
	den := 1 + t*(d[0]+t*(d[1]+t*d[2]))
	return sign * (t - num/den)
}

// statsNormalOverlap computes the area-overlap coefficient of two normal distributions.
func statsNormalOverlap(mu1, sigma1, mu2, sigma2 float64) float64 {
	if sigma1 == 0 && sigma2 == 0 {
		if mu1 == mu2 {
			return 1
		}
		return 0
	}
	if sigma1 == 0 {
		cdf := 0.5 * math.Erfc(-(mu1-mu2)/(sigma2*math.Sqrt2))
		return 2 * math.Min(cdf, 1-cdf)
	}
	if sigma2 == 0 {
		cdf := 0.5 * math.Erfc(-(mu2-mu1)/(sigma1*math.Sqrt2))
		return 2 * math.Min(cdf, 1-cdf)
	}
	if sigma1 == sigma2 {
		d := math.Abs(mu2-mu1) / sigma1
		return math.Erfc(d / (2 * math.Sqrt2))
	}
	// General: find quadratic intersections.
	a := 1/(2*sigma1*sigma1) - 1/(2*sigma2*sigma2)
	b := mu2/(sigma2*sigma2) - mu1/(sigma1*sigma1)
	c2 := mu1*mu1/(2*sigma1*sigma1) - mu2*mu2/(2*sigma2*sigma2) + math.Log(sigma2/sigma1)
	if a == 0 {
		x0 := -c2 / b
		p1 := 0.5 * math.Erfc(-(x0-mu1)/(sigma1*math.Sqrt2))
		p2 := 0.5 * math.Erfc(-(x0-mu2)/(sigma2*math.Sqrt2))
		return p1 + (1 - p2)
	}
	disc := b*b - 4*a*c2
	if disc < 0 {
		return 0
	}
	x1 := (-b - math.Sqrt(disc)) / (2 * a)
	x2 := (-b + math.Sqrt(disc)) / (2 * a)
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	cdf := func(x, mu, sigma float64) float64 { return 0.5 * math.Erfc(-(x-mu)/(sigma*math.Sqrt2)) }
	p1a, p1b := cdf(x1, mu1, sigma1), cdf(x1, mu2, sigma2)
	p2a, p2b := cdf(x2, mu1, sigma1), cdf(x2, mu2, sigma2)
	area := math.Abs(p1a-p1b) + math.Abs(p2a-p2b) + math.Abs((1-p2a)-(1-p2b))
	if area > 1 {
		area = 1
	}
	return area
}

// kdeKernelPDF returns the unit kernel PDF function for the given kernel name.
func kdeKernelPDF(name string) (func(float64) float64, error) {
	switch name {
	case "normal", "gauss":
		return func(u float64) float64 { return math.Exp(-0.5*u*u) / math.Sqrt(2*math.Pi) }, nil
	case "logistic":
		return func(u float64) float64 {
			e := math.Exp(-u)
			d := 1 + e
			return e / (d * d)
		}, nil
	case "sigmoid":
		return func(u float64) float64 { return 2 / (math.Pi * (math.Exp(u) + math.Exp(-u))) }, nil
	case "rectangular", "uniform":
		return func(u float64) float64 {
			if math.Abs(u) <= 1 {
				return 0.5
			}
			return 0
		}, nil
	case "triangular":
		return func(u float64) float64 {
			au := math.Abs(u)
			if au <= 1 {
				return 1 - au
			}
			return 0
		}, nil
	case "parabolic", "epanechnikov":
		return func(u float64) float64 {
			if math.Abs(u) <= 1 {
				return 0.75 * (1 - u*u)
			}
			return 0
		}, nil
	case "quartic", "biweight":
		return func(u float64) float64 {
			if math.Abs(u) <= 1 {
				t := 1 - u*u
				return 15.0 / 16.0 * t * t
			}
			return 0
		}, nil
	case "triweight":
		return func(u float64) float64 {
			if math.Abs(u) <= 1 {
				t := 1 - u*u
				return 35.0 / 32.0 * t * t * t
			}
			return 0
		}, nil
	case "cosine":
		return func(u float64) float64 {
			if math.Abs(u) <= 1 {
				return math.Pi / 4 * math.Cos(math.Pi/2*u)
			}
			return 0
		}, nil
	}
	return nil, fmt.Errorf("unknown kernel: %s", name)
}

// kdeKernelCDF returns the kernel CDF function.
func kdeKernelCDF(name string) (func(float64) float64, error) {
	switch name {
	case "normal", "gauss":
		return func(u float64) float64 { return 0.5 * math.Erfc(-u/math.Sqrt2) }, nil
	case "rectangular", "uniform":
		return func(u float64) float64 {
			if u <= -1 {
				return 0
			}
			if u >= 1 {
				return 1
			}
			return 0.5 + u/2
		}, nil
	case "triangular":
		return func(u float64) float64 {
			if u <= -1 {
				return 0
			}
			if u >= 1 {
				return 1
			}
			if u < 0 {
				return 0.5 + u + u*u/2
			}
			return 0.5 + u - u*u/2
		}, nil
	case "parabolic", "epanechnikov":
		return func(u float64) float64 {
			if u <= -1 {
				return 0
			}
			if u >= 1 {
				return 1
			}
			return 0.5 + 0.75*(u-u*u*u/3)
		}, nil
	default:
		pdf, err := kdeKernelPDF(name)
		if err != nil {
			return nil, err
		}
		// Numeric integration via midpoint rule.
		return func(u float64) float64 {
			lo := -10.0
			if u >= 10 {
				return 1
			}
			if u <= -10 {
				return 0
			}
			steps := 200
			dx := (u - lo) / float64(steps)
			s := 0.0
			for k := 0; k < steps; k++ {
				x := lo + (float64(k)+0.5)*dx
				s += pdf(x) * dx
			}
			return s
		}, nil
	}
}

// kdeKernelSample draws a noise sample from the given kernel distribution.
func kdeKernelSample(name string, rng *rand.Rand) float64 {
	switch name {
	case "normal", "gauss":
		return rng.NormFloat64()
	case "rectangular", "uniform":
		return rng.Float64()*2 - 1
	case "triangular":
		u := rng.Float64()
		if u < 0.5 {
			return math.Sqrt(2*u) - 1
		}
		return 1 - math.Sqrt(2*(1-u))
	default:
		pdf, err := kdeKernelPDF(name)
		if err != nil {
			return rng.NormFloat64()
		}
		// Rejection sampling.
		for {
			u := rng.Float64()*2 - 1
			v := rng.Float64()
			if v <= pdf(u)*2 {
				return u
			}
		}
	}
}

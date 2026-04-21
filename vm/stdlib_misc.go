package vm

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/tamnd/goipy/object"
)

// --- statistics module -----------------------------------------------------

// buildStatistics exposes the statistics module. Covers mean/median/mode/
// variance/stdev and a few derived helpers. Inputs are coerced to float64 for
// computation; mean preserves int when the result is integral over int inputs.
func (i *Interp) buildStatistics() *object.Module {
	m := &object.Module{Name: "statistics", Dict: object.NewDict()}

	m.Dict.SetStr("mean", &object.BuiltinFunc{Name: "mean", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		xs, allInt, err := i.collectNumbers(a, "mean")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, object.Errorf(i.valueErr, "mean requires at least one data point")
		}
		sum := 0.0
		for _, x := range xs {
			sum += x
		}
		avg := sum / float64(len(xs))
		if allInt && avg == math.Trunc(avg) && !math.IsInf(avg, 0) {
			return &object.Int{V: big.NewInt(int64(avg))}, nil
		}
		return &object.Float{V: avg}, nil
	}})

	m.Dict.SetStr("fmean", &object.BuiltinFunc{Name: "fmean", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		xs, _, err := i.collectNumbers(a, "fmean")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, object.Errorf(i.valueErr, "fmean requires at least one data point")
		}
		sum := 0.0
		for _, x := range xs {
			sum += x
		}
		return &object.Float{V: sum / float64(len(xs))}, nil
	}})

	m.Dict.SetStr("geometric_mean", &object.BuiltinFunc{Name: "geometric_mean", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		xs, _, err := i.collectNumbers(a, "geometric_mean")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, object.Errorf(i.valueErr, "geometric_mean requires at least one data point")
		}
		logSum := 0.0
		for _, x := range xs {
			if x <= 0 {
				return nil, object.Errorf(i.valueErr, "geometric_mean requires positive values")
			}
			logSum += math.Log(x)
		}
		return &object.Float{V: math.Exp(logSum / float64(len(xs)))}, nil
	}})

	m.Dict.SetStr("harmonic_mean", &object.BuiltinFunc{Name: "harmonic_mean", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		xs, _, err := i.collectNumbers(a, "harmonic_mean")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, object.Errorf(i.valueErr, "harmonic_mean requires at least one data point")
		}
		recSum := 0.0
		for _, x := range xs {
			if x <= 0 {
				return nil, object.Errorf(i.valueErr, "harmonic_mean requires positive values")
			}
			recSum += 1 / x
		}
		return &object.Float{V: float64(len(xs)) / recSum}, nil
	}})

	m.Dict.SetStr("median", &object.BuiltinFunc{Name: "median", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		xs, allInt, err := i.collectNumbers(a, "median")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, object.Errorf(i.valueErr, "median requires at least one data point")
		}
		sort.Float64s(xs)
		n := len(xs)
		if n%2 == 1 {
			v := xs[n/2]
			if allInt {
				return &object.Int{V: big.NewInt(int64(v))}, nil
			}
			return &object.Float{V: v}, nil
		}
		_ = allInt
		v := (xs[n/2-1] + xs[n/2]) / 2
		return &object.Float{V: v}, nil
	}})

	m.Dict.SetStr("median_low", &object.BuiltinFunc{Name: "median_low", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		xs, allInt, err := i.collectNumbers(a, "median_low")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, object.Errorf(i.valueErr, "median_low requires at least one data point")
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
			return &object.Int{V: big.NewInt(int64(v))}, nil
		}
		return &object.Float{V: v}, nil
	}})

	m.Dict.SetStr("median_high", &object.BuiltinFunc{Name: "median_high", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		xs, allInt, err := i.collectNumbers(a, "median_high")
		if err != nil {
			return nil, err
		}
		if len(xs) == 0 {
			return nil, object.Errorf(i.valueErr, "median_high requires at least one data point")
		}
		sort.Float64s(xs)
		n := len(xs)
		v := xs[n/2]
		if allInt {
			return &object.Int{V: big.NewInt(int64(v))}, nil
		}
		return &object.Float{V: v}, nil
	}})

	m.Dict.SetStr("mode", &object.BuiltinFunc{Name: "mode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		seq, err := i.iterObjects(a, "mode")
		if err != nil {
			return nil, err
		}
		if len(seq) == 0 {
			return nil, object.Errorf(i.valueErr, "mode requires at least one data point")
		}
		// Track first-seen order + counts.
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
		// First max wins (CPython behaviour).
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
		for _, o := range seq {
			k := object.Repr(o)
			if _, seen := counts[k]; !seen {
				order = append(order, o)
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
				out = append(out, o)
			}
		}
		return &object.List{V: out}, nil
	}})

	m.Dict.SetStr("pvariance", &object.BuiltinFunc{Name: "pvariance", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		v, allInt, err := i.varianceWithKind(a, false, "pvariance")
		if err != nil {
			return nil, err
		}
		if allInt && v == math.Trunc(v) && !math.IsInf(v, 0) {
			return &object.Int{V: big.NewInt(int64(v))}, nil
		}
		return &object.Float{V: v}, nil
	}})

	m.Dict.SetStr("variance", &object.BuiltinFunc{Name: "variance", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		v, allInt, err := i.varianceWithKind(a, true, "variance")
		if err != nil {
			return nil, err
		}
		if allInt && v == math.Trunc(v) && !math.IsInf(v, 0) {
			return &object.Int{V: big.NewInt(int64(v))}, nil
		}
		return &object.Float{V: v}, nil
	}})

	m.Dict.SetStr("pstdev", &object.BuiltinFunc{Name: "pstdev", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		v, err := i.varianceOf(a, false, "pstdev")
		if err != nil {
			return nil, err
		}
		return &object.Float{V: math.Sqrt(v)}, nil
	}})

	m.Dict.SetStr("stdev", &object.BuiltinFunc{Name: "stdev", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		v, err := i.varianceOf(a, true, "stdev")
		if err != nil {
			return nil, err
		}
		return &object.Float{V: math.Sqrt(v)}, nil
	}})

	m.Dict.SetStr("quantiles", &object.BuiltinFunc{Name: "quantiles", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		xs, _, err := i.collectNumbers(a, "quantiles")
		if err != nil {
			return nil, err
		}
		if len(xs) < 2 {
			return nil, object.Errorf(i.valueErr, "quantiles requires at least two data points")
		}
		n := 4
		method := "exclusive"
		if kw != nil {
			if v, ok := kw.GetStr("n"); ok {
				if iv, ok := toInt64(v); ok {
					n = int(iv)
				}
			}
			if v, ok := kw.GetStr("method"); ok {
				if s, ok := v.(*object.Str); ok {
					method = s.V
				}
			}
		}
		if n < 1 {
			return nil, object.Errorf(i.valueErr, "n must be at least 1")
		}
		sort.Float64s(xs)
		out := make([]object.Object, n-1)
		L := len(xs)
		for k := 1; k < n; k++ {
			var pos float64
			if method == "inclusive" {
				pos = float64(k) * float64(L-1) / float64(n)
			} else {
				pos = float64(k) * float64(L+1) / float64(n)
				pos -= 1 // zero-based
			}
			lo := int(math.Floor(pos))
			if lo < 0 {
				lo = 0
			}
			if lo >= L-1 {
				lo = L - 2
			}
			frac := pos - float64(lo)
			v := xs[lo] + frac*(xs[lo+1]-xs[lo])
			out[k-1] = &object.Float{V: v}
		}
		return &object.List{V: out}, nil
	}})

	return m
}

func (i *Interp) collectNumbers(a []object.Object, fn string) ([]float64, bool, error) {
	if len(a) == 0 {
		return nil, false, object.Errorf(i.typeErr, "%s() missing argument", fn)
	}
	seq, err := i.iterObjects(a, fn)
	if err != nil {
		return nil, false, err
	}
	xs := make([]float64, 0, len(seq))
	allInt := true
	for _, o := range seq {
		if _, ok := o.(*object.Int); !ok {
			if _, ok := o.(*object.Bool); !ok {
				allInt = false
			}
		}
		f, ok := toFloat64Any(o)
		if !ok {
			return nil, false, object.Errorf(i.typeErr, "%s() requires numeric data", fn)
		}
		xs = append(xs, f)
	}
	return xs, allInt, nil
}

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
		return rangeToList(v), nil
	case *object.Set:
		return append([]object.Object(nil), v.Items()...), nil
	case *object.Frozenset:
		return append([]object.Object(nil), v.Items()...), nil
	case *object.Iter:
		out := []object.Object{}
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

func (i *Interp) varianceOf(a []object.Object, sample bool, fn string) (float64, error) {
	v, _, err := i.varianceWithKind(a, sample, fn)
	return v, err
}

func (i *Interp) varianceWithKind(a []object.Object, sample bool, fn string) (float64, bool, error) {
	xs, allInt, err := i.collectNumbers(a, fn)
	if err != nil {
		return 0, false, err
	}
	minN := 1
	if sample {
		minN = 2
	}
	if len(xs) < minN {
		return 0, false, object.Errorf(i.valueErr, "%s requires at least %d data points", fn, minN)
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	mean := sum / float64(len(xs))
	sq := 0.0
	for _, x := range xs {
		d := x - mean
		sq += d * d
	}
	div := float64(len(xs))
	if sample {
		div -= 1
	}
	return sq / div, allInt, nil
}

func rangeToList(r *object.Range) []object.Object {
	out := []object.Object{}
	if r.Step == 0 {
		return out
	}
	for v := r.Start; (r.Step > 0 && v < r.Stop) || (r.Step < 0 && v > r.Stop); v += r.Step {
		out = append(out, &object.Int{V: big.NewInt(v)})
	}
	return out
}

// --- calendar module -------------------------------------------------------

var calMonthName = []string{"", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
var calMonthAbbr = []string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
var calDayName = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
var calDayAbbr = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

func (i *Interp) buildCalendar() *object.Module {
	m := &object.Module{Name: "calendar", Dict: object.NewDict()}

	// Constants for weekday names.
	m.Dict.SetStr("MONDAY", &object.Int{V: big.NewInt(0)})
	m.Dict.SetStr("TUESDAY", &object.Int{V: big.NewInt(1)})
	m.Dict.SetStr("WEDNESDAY", &object.Int{V: big.NewInt(2)})
	m.Dict.SetStr("THURSDAY", &object.Int{V: big.NewInt(3)})
	m.Dict.SetStr("FRIDAY", &object.Int{V: big.NewInt(4)})
	m.Dict.SetStr("SATURDAY", &object.Int{V: big.NewInt(5)})
	m.Dict.SetStr("SUNDAY", &object.Int{V: big.NewInt(6)})

	m.Dict.SetStr("month_name", strList(calMonthName))
	m.Dict.SetStr("month_abbr", strList(calMonthAbbr))
	m.Dict.SetStr("day_name", strList(calDayName))
	m.Dict.SetStr("day_abbr", strList(calDayAbbr))

	m.Dict.SetStr("isleap", &object.BuiltinFunc{Name: "isleap", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		y, err := intArg(i, a, "isleap")
		if err != nil {
			return nil, err
		}
		return object.BoolOf(isLeap(y)), nil
	}})

	m.Dict.SetStr("leapdays", &object.BuiltinFunc{Name: "leapdays", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "leapdays() requires y1 and y2")
		}
		y1v, ok := toInt64(a[0])
		y2v, ok2 := toInt64(a[1])
		if !ok || !ok2 {
			return nil, object.Errorf(i.typeErr, "leapdays() requires int args")
		}
		y1, y2 := int(y1v), int(y2v)
		y1 -= 1
		y2 -= 1
		d := func(y int) int {
			return y/4 - y/100 + y/400
		}
		return &object.Int{V: big.NewInt(int64(d(y2) - d(y1)))}, nil
	}})

	m.Dict.SetStr("weekday", &object.BuiltinFunc{Name: "weekday", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "weekday() requires year, month, day")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		d, _ := toInt64(a[2])
		t := time.Date(int(y), time.Month(mo), int(d), 0, 0, 0, 0, time.UTC)
		// Go: Sunday=0..Saturday=6. Python: Monday=0..Sunday=6.
		w := (int(t.Weekday()) + 6) % 7
		return &object.Int{V: big.NewInt(int64(w))}, nil
	}})

	m.Dict.SetStr("monthrange", &object.BuiltinFunc{Name: "monthrange", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "monthrange() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		if mo < 1 || mo > 12 {
			return nil, object.Errorf(i.valueErr, "bad month number %d", mo)
		}
		first := time.Date(int(y), time.Month(mo), 1, 0, 0, 0, 0, time.UTC)
		w := (int(first.Weekday()) + 6) % 7
		days := daysInMonth(int(y), int(mo))
		return &object.Tuple{V: []object.Object{
			&object.Int{V: big.NewInt(int64(w))},
			&object.Int{V: big.NewInt(int64(days))},
		}}, nil
	}})

	m.Dict.SetStr("monthcalendar", &object.BuiltinFunc{Name: "monthcalendar", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "monthcalendar() requires year, month")
		}
		y, _ := toInt64(a[0])
		mo, _ := toInt64(a[1])
		if mo < 1 || mo > 12 {
			return nil, object.Errorf(i.valueErr, "bad month number %d", mo)
		}
		first := time.Date(int(y), time.Month(mo), 1, 0, 0, 0, 0, time.UTC)
		firstWd := (int(first.Weekday()) + 6) % 7
		days := daysInMonth(int(y), int(mo))
		weeks := [][]int{}
		row := make([]int, 7)
		for i := 0; i < firstWd; i++ {
			row[i] = 0
		}
		day := 1
		col := firstWd
		for day <= days {
			row[col] = day
			day++
			col++
			if col == 7 {
				weeks = append(weeks, row)
				row = make([]int, 7)
				col = 0
			}
		}
		if col != 0 {
			weeks = append(weeks, row)
		}
		out := make([]object.Object, len(weeks))
		for i, w := range weeks {
			lst := make([]object.Object, 7)
			for j, d := range w {
				lst[j] = &object.Int{V: big.NewInt(int64(d))}
			}
			out[i] = &object.List{V: lst}
		}
		return &object.List{V: out}, nil
	}})

	m.Dict.SetStr("timegm", &object.BuiltinFunc{Name: "timegm", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "timegm() requires a tuple")
		}
		var parts []object.Object
		switch v := a[0].(type) {
		case *object.Tuple:
			parts = v.V
		case *object.List:
			parts = v.V
		default:
			return nil, object.Errorf(i.typeErr, "timegm() requires a tuple")
		}
		if len(parts) < 6 {
			return nil, object.Errorf(i.typeErr, "timegm() needs 6+ fields")
		}
		y, _ := toInt64(parts[0])
		mo, _ := toInt64(parts[1])
		d, _ := toInt64(parts[2])
		h, _ := toInt64(parts[3])
		mi, _ := toInt64(parts[4])
		se, _ := toInt64(parts[5])
		t := time.Date(int(y), time.Month(mo), int(d), int(h), int(mi), int(se), 0, time.UTC)
		return &object.Int{V: big.NewInt(t.Unix())}, nil
	}})

	return m
}

func isLeap(y int64) bool {
	return (y%4 == 0 && y%100 != 0) || y%400 == 0
}

func daysInMonth(y, m int) int {
	days := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if m == 2 && isLeap(int64(y)) {
		return 29
	}
	return days[m-1]
}

func strList(ss []string) *object.List {
	v := make([]object.Object, len(ss))
	for i, s := range ss {
		v[i] = &object.Str{V: s}
	}
	return &object.List{V: v}
}

func intArg(i *Interp, a []object.Object, fn string) (int64, error) {
	if len(a) == 0 {
		return 0, object.Errorf(i.typeErr, "%s() missing argument", fn)
	}
	v, ok := toInt64(a[0])
	if !ok {
		return 0, object.Errorf(i.typeErr, "%s() argument must be int", fn)
	}
	return v, nil
}

// --- pprint module ---------------------------------------------------------

func (i *Interp) buildPprint() *object.Module {
	m := &object.Module{Name: "pprint", Dict: object.NewDict()}

	doFormat := func(o object.Object, kw *object.Dict) (string, error) {
		opts := pformatOpts{
			indent:    1,
			width:     80,
			depth:     -1,
			compact:   false,
			sortDicts: true,
		}
		if kw != nil {
			if v, ok := kw.GetStr("indent"); ok {
				if iv, ok := toInt64(v); ok {
					opts.indent = int(iv)
				}
			}
			if v, ok := kw.GetStr("width"); ok {
				if iv, ok := toInt64(v); ok {
					opts.width = int(iv)
				}
			}
			if v, ok := kw.GetStr("depth"); ok {
				if iv, ok := toInt64(v); ok {
					opts.depth = int(iv)
				}
			}
			if v, ok := kw.GetStr("compact"); ok {
				if b, ok := v.(*object.Bool); ok {
					opts.compact = b.V
				}
			}
			if v, ok := kw.GetStr("sort_dicts"); ok {
				if b, ok := v.(*object.Bool); ok {
					opts.sortDicts = b.V
				}
			}
		}
		return pformat(o, opts), nil
	}

	m.Dict.SetStr("pformat", &object.BuiltinFunc{Name: "pformat", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "pformat() missing argument")
		}
		s, err := doFormat(a[0], kw)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: s}, nil
	}})

	m.Dict.SetStr("pprint", &object.BuiltinFunc{Name: "pprint", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "pprint() missing argument")
		}
		s, err := doFormat(a[0], kw)
		if err != nil {
			return nil, err
		}
		fmt.Fprintln(i.Stdout, s)
		return object.None, nil
	}})

	m.Dict.SetStr("isreadable", &object.BuiltinFunc{Name: "isreadable", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "isreadable() missing argument")
		}
		return object.BoolOf(isReadable(a[0])), nil
	}})

	m.Dict.SetStr("isrecursive", &object.BuiltinFunc{Name: "isrecursive", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})

	m.Dict.SetStr("saferepr", &object.BuiltinFunc{Name: "saferepr", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "saferepr() missing argument")
		}
		return &object.Str{V: object.Repr(a[0])}, nil
	}})

	return m
}

type pformatOpts struct {
	indent    int
	width     int
	depth     int
	compact   bool
	sortDicts bool
}

func pformat(o object.Object, opts pformatOpts) string {
	return pformatAt(o, opts, 0, 0)
}

// pformatAt formats at the given column position; returns a possibly multi-line
// string whose first line starts at col.
func pformatAt(o object.Object, opts pformatOpts, col, depth int) string {
	if opts.depth >= 0 && depth > opts.depth {
		return "..."
	}
	switch v := o.(type) {
	case *object.List:
		return formatSeq(v.V, "[", "]", opts, col, depth)
	case *object.Tuple:
		if len(v.V) == 1 {
			s := pformatAt(v.V[0], opts, col+1, depth+1)
			return "(" + s + ",)"
		}
		return formatSeq(v.V, "(", ")", opts, col, depth)
	case *object.Set:
		items := append([]object.Object(nil), v.Items()...)
		if len(items) == 0 {
			return "set()"
		}
		sortItemsByRepr(items)
		return formatSeq(items, "{", "}", opts, col, depth)
	case *object.Frozenset:
		items := append([]object.Object(nil), v.Items()...)
		if len(items) == 0 {
			return "frozenset()"
		}
		sortItemsByRepr(items)
		return "frozenset(" + formatSeq(items, "{", "}", opts, col+len("frozenset("), depth) + ")"
	case *object.Dict:
		return formatDict(v, opts, col, depth)
	}
	return object.Repr(o)
}

func formatSeq(items []object.Object, open, close string, opts pformatOpts, col, depth int) string {
	if len(items) == 0 {
		return open + close
	}
	// Attempt single-line.
	parts := make([]string, len(items))
	total := 0
	for idx, it := range items {
		parts[idx] = object.Repr(it)
		total += len(parts[idx])
	}
	total += 2 * (len(items) - 1)
	oneLine := open + strings.Join(parts, ", ") + close
	if col+len(oneLine) <= opts.width {
		return oneLine
	}
	// Multi-line: one item per line, aligned under opening delimiter.
	pad := strings.Repeat(" ", col+len(open))
	var b strings.Builder
	b.WriteString(open)
	for idx, it := range items {
		if idx > 0 {
			b.WriteString(",\n")
			b.WriteString(pad)
		}
		b.WriteString(pformatAt(it, opts, col+len(open), depth+1))
	}
	b.WriteString(close)
	return b.String()
}

func formatDict(d *object.Dict, opts pformatOpts, col, depth int) string {
	keys, vals := d.Items()
	if len(keys) == 0 {
		return "{}"
	}
	type kv struct {
		k, v object.Object
	}
	pairs := make([]kv, len(keys))
	for i := range keys {
		pairs[i] = kv{keys[i], vals[i]}
	}
	if opts.sortDicts {
		sort.SliceStable(pairs, func(a, b int) bool {
			return dictKeyLess(pairs[a].k, pairs[b].k)
		})
	}
	// Try single-line.
	parts := make([]string, len(pairs))
	for idx, p := range pairs {
		parts[idx] = object.Repr(p.k) + ": " + object.Repr(p.v)
	}
	oneLine := "{" + strings.Join(parts, ", ") + "}"
	if col+len(oneLine) <= opts.width {
		return oneLine
	}
	pad := strings.Repeat(" ", col+1)
	var b strings.Builder
	b.WriteString("{")
	for idx, p := range pairs {
		if idx > 0 {
			b.WriteString(",\n")
			b.WriteString(pad)
		}
		kRepr := object.Repr(p.k)
		b.WriteString(kRepr)
		b.WriteString(": ")
		b.WriteString(pformatAt(p.v, opts, col+1+len(kRepr)+2, depth+1))
	}
	b.WriteString("}")
	return b.String()
}

func sortItemsByRepr(items []object.Object) {
	sort.SliceStable(items, func(i, j int) bool {
		return object.Repr(items[i]) < object.Repr(items[j])
	})
}

func dictKeyLess(a, b object.Object) bool {
	return object.Repr(a) < object.Repr(b)
}

func isReadable(o object.Object) bool {
	switch v := o.(type) {
	case *object.List:
		for _, x := range v.V {
			if !isReadable(x) {
				return false
			}
		}
		return true
	case *object.Tuple:
		for _, x := range v.V {
			if !isReadable(x) {
				return false
			}
		}
		return true
	case *object.Dict:
		_, vals := v.Items()
		for _, x := range vals {
			if !isReadable(x) {
				return false
			}
		}
		return true
	case *object.Set, *object.Frozenset, *object.NoneType, *object.Bool, *object.Int, *object.Float, *object.Str, *object.Bytes, *object.Range:
		return true
	}
	return false
}

// --- html module -----------------------------------------------------------

func (i *Interp) buildHtml() *object.Module {
	m := &object.Module{Name: "html", Dict: object.NewDict()}

	m.Dict.SetStr("escape", &object.BuiltinFunc{Name: "escape", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "escape")
		if err != nil {
			return nil, err
		}
		quote := true
		if len(a) >= 2 {
			if b, ok := a[1].(*object.Bool); ok {
				quote = b.V
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("quote"); ok {
				if b, ok := v.(*object.Bool); ok {
					quote = b.V
				}
			}
		}
		return &object.Str{V: htmlEscape(s, quote)}, nil
	}})

	m.Dict.SetStr("unescape", &object.BuiltinFunc{Name: "unescape", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "unescape")
		if err != nil {
			return nil, err
		}
		return &object.Str{V: htmlUnescape(s)}, nil
	}})

	return m
}

func htmlEscape(s string, quote bool) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	if quote {
		s = strings.ReplaceAll(s, `"`, "&quot;")
		s = strings.ReplaceAll(s, "'", "&#x27;")
	}
	return s
}

// htmlEntities maps the common named entities we support. This is a subset of
// HTML5's entity list — enough for real-world text without pulling in a full
// 2000+ entry table.
var htmlEntities = map[string]string{
	"amp": "&", "lt": "<", "gt": ">", "quot": "\"", "apos": "'",
	"nbsp": "\u00a0", "copy": "\u00a9", "reg": "\u00ae", "trade": "\u2122",
	"hellip": "\u2026", "mdash": "\u2014", "ndash": "\u2013",
	"lsquo": "\u2018", "rsquo": "\u2019", "ldquo": "\u201c", "rdquo": "\u201d",
	"laquo": "\u00ab", "raquo": "\u00bb",
	"deg": "\u00b0", "plusmn": "\u00b1", "times": "\u00d7", "divide": "\u00f7",
	"euro": "\u20ac", "pound": "\u00a3", "yen": "\u00a5", "cent": "\u00a2",
	"sect": "\u00a7", "para": "\u00b6", "middot": "\u00b7",
	"iexcl": "\u00a1", "iquest": "\u00bf",
}

func htmlUnescape(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		// Find trailing semicolon within a reasonable window.
		end := -1
		for j := i + 1; j < len(s) && j-i < 40; j++ {
			if s[j] == ';' {
				end = j
				break
			}
			if s[j] == '&' || s[j] == ' ' || s[j] == '\t' || s[j] == '\n' {
				break
			}
		}
		if end < 0 {
			b.WriteByte(s[i])
			i++
			continue
		}
		body := s[i+1 : end]
		if len(body) >= 2 && body[0] == '#' {
			var n int64
			var err error
			if body[1] == 'x' || body[1] == 'X' {
				_, err = fmt.Sscanf(body[2:], "%x", &n)
			} else {
				_, err = fmt.Sscanf(body[1:], "%d", &n)
			}
			if err == nil && n >= 0 && n <= 0x10FFFF {
				b.WriteRune(rune(n))
				i = end + 1
				continue
			}
		} else if v, ok := htmlEntities[body]; ok {
			b.WriteString(v)
			i = end + 1
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

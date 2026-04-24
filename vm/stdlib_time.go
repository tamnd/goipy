package vm

import (
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/tamnd/goipy/object"
)

// buildTime exposes the full Python `time` module.
func (i *Interp) buildTime() *object.Module {
	m := &object.Module{Name: "time", Dict: object.NewDict()}

	// --- monotonic / perf clocks ---
	origin := time.Now()
	perf := &object.BuiltinFunc{Name: "perf_counter", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Float{V: time.Since(origin).Seconds()}, nil
	}}
	m.Dict.SetStr("perf_counter", perf)
	m.Dict.SetStr("perf_counter_ns", &object.BuiltinFunc{Name: "perf_counter_ns", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(time.Since(origin).Nanoseconds()), nil
	}})
	m.Dict.SetStr("monotonic", perf)
	m.Dict.SetStr("monotonic_ns", &object.BuiltinFunc{Name: "monotonic_ns", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(time.Since(origin).Nanoseconds()), nil
	}})

	// --- wall clock ---
	m.Dict.SetStr("time", &object.BuiltinFunc{Name: "time", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Float{V: float64(time.Now().UnixNano()) / 1e9}, nil
	}})
	m.Dict.SetStr("time_ns", &object.BuiltinFunc{Name: "time_ns", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(time.Now().UnixNano()), nil
	}})

	// --- sleep ---
	m.Dict.SetStr("sleep", &object.BuiltinFunc{Name: "sleep", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "sleep() takes exactly one argument")
		}
		secs, ok := toFloat64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "sleep() argument must be numeric")
		}
		if secs > 0 {
			time.Sleep(time.Duration(secs * float64(time.Second)))
		}
		return object.None, nil
	}})

	// --- struct_time class ---
	stCls := buildStructTimeClass()
	m.Dict.SetStr("struct_time", stCls)

	// --- localtime([secs]) ---
	m.Dict.SetStr("localtime", &object.BuiltinFunc{Name: "localtime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		t := time.Now()
		if len(a) >= 1 && a[0] != object.None {
			secs, ok := toFloat64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "localtime() argument must be numeric")
			}
			sec := int64(secs)
			nsec := int64((secs - float64(sec)) * 1e9)
			t = time.Unix(sec, nsec).In(time.Local)
		} else {
			t = t.In(time.Local)
		}
		return goTimeToStructTime(stCls, t), nil
	}})

	// --- gmtime([secs]) ---
	m.Dict.SetStr("gmtime", &object.BuiltinFunc{Name: "gmtime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		t := time.Now().UTC()
		if len(a) >= 1 && a[0] != object.None {
			secs, ok := toFloat64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "gmtime() argument must be numeric")
			}
			sec := int64(secs)
			nsec := int64((secs - float64(sec)) * 1e9)
			t = time.Unix(sec, nsec).UTC()
		}
		return goTimeToStructTime(stCls, t), nil
	}})

	// --- mktime(t) ---
	m.Dict.SetStr("mktime", &object.BuiltinFunc{Name: "mktime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "mktime() requires 1 argument")
		}
		gt, err := structTimeToGoTime(a[0], time.Local)
		if err != nil {
			return nil, object.Errorf(i.typeErr, "%v", err)
		}
		return &object.Float{V: float64(gt.UnixNano()) / 1e9}, nil
	}})

	// --- asctime([t]) ---
	m.Dict.SetStr("asctime", &object.BuiltinFunc{Name: "asctime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var gt time.Time
		if len(a) >= 1 && a[0] != object.None {
			var err error
			gt, err = structTimeToGoTime(a[0], time.Local)
			if err != nil {
				return nil, object.Errorf(i.typeErr, "%v", err)
			}
		} else {
			gt = time.Now().In(time.Local)
		}
		return &object.Str{V: asctime(gt)}, nil
	}})

	// --- ctime([secs]) ---
	m.Dict.SetStr("ctime", &object.BuiltinFunc{Name: "ctime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		t := time.Now().In(time.Local)
		if len(a) >= 1 && a[0] != object.None {
			secs, ok := toFloat64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "ctime() argument must be numeric")
			}
			sec := int64(secs)
			nsec := int64((secs - float64(sec)) * 1e9)
			t = time.Unix(sec, nsec).In(time.Local)
		}
		return &object.Str{V: asctime(t)}, nil
	}})

	// --- strftime(format[, t]) ---
	m.Dict.SetStr("strftime", &object.BuiltinFunc{Name: "strftime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "strftime() requires at least 1 argument")
		}
		fmtStr, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "strftime() format must be str")
		}
		gt := time.Now().In(time.Local)
		if len(a) >= 2 && a[1] != object.None {
			var err error
			gt, err = structTimeToGoTime(a[1], time.Local)
			if err != nil {
				return nil, object.Errorf(i.typeErr, "%v", err)
			}
		}
		return &object.Str{V: strftime(fmtStr.V, gt)}, nil
	}})

	// --- strptime(string[, format]) ---
	m.Dict.SetStr("strptime", &object.BuiltinFunc{Name: "strptime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "strptime() requires at least 1 argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "strptime() first argument must be str")
		}
		fmtPy := "%a %b %e %H:%M:%S %Y" // default like ctime
		if len(a) >= 2 {
			fs, ok2 := a[1].(*object.Str)
			if !ok2 {
				return nil, object.Errorf(i.typeErr, "strptime() format must be str")
			}
			fmtPy = fs.V
		}
		goLayout := pyFmtToGo(fmtPy)
		t, err := time.Parse(goLayout, s.V)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "time data %q does not match format %q", s.V, fmtPy)
		}
		return goTimeToStructTime(stCls, t), nil
	}})

	// --- timezone / altzone / daylight / tzname ---
	_, tzOffset := time.Now().In(time.Local).Zone()
	// Python convention: timezone is seconds WEST of UTC (positive west).
	m.Dict.SetStr("timezone", object.NewInt(int64(-tzOffset)))
	// Estimate altzone as timezone - 3600 (common DST offset).
	_, isDST := time.Date(time.Now().Year(), 7, 1, 0, 0, 0, 0, time.Local).Zone()
	m.Dict.SetStr("altzone", object.NewInt(int64(-isDST)))
	stdName, _ := time.Now().In(time.Local).Zone()
	dstName, _ := time.Date(time.Now().Year(), 7, 1, 0, 0, 0, 0, time.Local).Zone()
	daylight := 0
	if stdName != dstName {
		daylight = 1
	}
	m.Dict.SetStr("daylight", object.NewInt(int64(daylight)))
	m.Dict.SetStr("tzname", &object.Tuple{V: []object.Object{
		&object.Str{V: stdName},
		&object.Str{V: dstName},
	}})

	// --- process_time / process_time_ns ---
	m.Dict.SetStr("process_time", &object.BuiltinFunc{Name: "process_time", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var ru syscall.Rusage
		_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
		secs := float64(ru.Utime.Sec) + float64(ru.Stime.Sec) +
			float64(ru.Utime.Usec+ru.Stime.Usec)/1e6
		return &object.Float{V: secs}, nil
	}})
	m.Dict.SetStr("process_time_ns", &object.BuiltinFunc{Name: "process_time_ns", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var ru syscall.Rusage
		_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
		ns := (int64(ru.Utime.Sec)+int64(ru.Stime.Sec))*1e9 +
			int64(ru.Utime.Usec+ru.Stime.Usec)*1000
		return object.NewInt(ns), nil
	}})

	// --- thread_time / thread_time_ns (platform-specific, best-effort) ---
	m.Dict.SetStr("thread_time", &object.BuiltinFunc{Name: "thread_time", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		ns := threadTimeNs()
		return &object.Float{V: float64(ns) / 1e9}, nil
	}})
	m.Dict.SetStr("thread_time_ns", &object.BuiltinFunc{Name: "thread_time_ns", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(threadTimeNs()), nil
	}})

	// --- get_clock_info(name) ---
	m.Dict.SetStr("get_clock_info", &object.BuiltinFunc{Name: "get_clock_info", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "get_clock_info() requires 1 argument")
		}
		name, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "get_clock_info() argument must be str")
		}
		info := clockInfo(name.V)
		if info == nil {
			return nil, object.Errorf(i.valueErr, "unknown clock: %s", name.V)
		}
		return info, nil
	}})

	// --- CLOCK_* constants (values from platform-specific files) ---
	m.Dict.SetStr("CLOCK_REALTIME", object.NewInt(clockRealtime))
	m.Dict.SetStr("CLOCK_MONOTONIC", object.NewInt(clockMonotonic))
	m.Dict.SetStr("CLOCK_PROCESS_CPUTIME_ID", object.NewInt(clockProcessCPUTime))
	m.Dict.SetStr("CLOCK_THREAD_CPUTIME_ID", object.NewInt(clockThreadCPUTime))

	return m
}

// buildStructTimeClass constructs the struct_time class with __getitem__ and
// __repr__ support.
func buildStructTimeClass() *object.Class {
	cls := &object.Class{Name: "struct_time", Dict: object.NewDict()}
	indices := []string{
		"tm_year", "tm_mon", "tm_mday",
		"tm_hour", "tm_min", "tm_sec",
		"tm_wday", "tm_yday", "tm_isdst",
	}
	cls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, fmt.Errorf("__getitem__ requires 2 arguments")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, fmt.Errorf("__getitem__ requires struct_time instance")
		}
		idx, ok2 := toInt64(a[1])
		if !ok2 {
			return nil, fmt.Errorf("struct_time indices must be integers")
		}
		if idx < 0 {
			idx += int64(len(indices))
		}
		if idx < 0 || int(idx) >= len(indices) {
			return nil, fmt.Errorf("struct_time index out of range")
		}
		if v, ok3 := self.Dict.GetStr(indices[idx]); ok3 {
			return v, nil
		}
		return object.NewInt(0), nil
	}})
	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "time.struct_time()"}, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "time.struct_time()"}, nil
		}
		parts := make([]string, len(indices))
		for k, attr := range indices {
			v, _ := self.Dict.GetStr(attr)
			parts[k] = fmt.Sprintf("%s=%s", attr, object.Repr(v))
		}
		return &object.Str{V: "time.struct_time(" + strings.Join(parts, ", ") + ")"}, nil
	}})
	return cls
}

// goTimeToStructTime converts a Go time.Time to a Python struct_time Instance.
func goTimeToStructTime(cls *object.Class, t time.Time) *object.Instance {
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	wday := int(t.Weekday()+6) % 7 // Python: 0=Monday
	yday := t.YearDay()
	_, off := t.Zone()
	isDST := 0
	// crude DST detection: check if offset matches non-DST offset
	stdName, stdOff := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location()).Zone()
	if off != stdOff {
		isDST = 1
	}
	zoneName, _ := t.Zone()

	inst.Dict.SetStr("tm_year", object.NewInt(int64(year)))
	inst.Dict.SetStr("tm_mon", object.NewInt(int64(month)))
	inst.Dict.SetStr("tm_mday", object.NewInt(int64(day)))
	inst.Dict.SetStr("tm_hour", object.NewInt(int64(hour)))
	inst.Dict.SetStr("tm_min", object.NewInt(int64(min)))
	inst.Dict.SetStr("tm_sec", object.NewInt(int64(sec)))
	inst.Dict.SetStr("tm_wday", object.NewInt(int64(wday)))
	inst.Dict.SetStr("tm_yday", object.NewInt(int64(yday)))
	inst.Dict.SetStr("tm_isdst", object.NewInt(int64(isDST)))
	inst.Dict.SetStr("tm_gmtoff", object.NewInt(int64(off)))
	inst.Dict.SetStr("tm_zone", &object.Str{V: zoneName})
	_ = stdName
	return inst
}

// structTimeToGoTime converts a Python struct_time (or 9-element tuple/list)
// to a Go time.Time in the given location.
func structTimeToGoTime(v object.Object, loc *time.Location) (time.Time, error) {
	get := func(name string, idx int) int {
		switch x := v.(type) {
		case *object.Instance:
			if val, ok := x.Dict.GetStr(name); ok {
				if n, ok2 := toInt64(val); ok2 {
					return int(n)
				}
			}
		case *object.Tuple:
			if idx < len(x.V) {
				if n, ok := toInt64(x.V[idx]); ok {
					return int(n)
				}
			}
		case *object.List:
			if idx < len(x.V) {
				if n, ok := toInt64(x.V[idx]); ok {
					return int(n)
				}
			}
		}
		return 0
	}
	year := get("tm_year", 0)
	mon := get("tm_mon", 1)
	mday := get("tm_mday", 2)
	hour := get("tm_hour", 3)
	min := get("tm_min", 4)
	sec := get("tm_sec", 5)
	if year == 0 {
		return time.Time{}, fmt.Errorf("invalid struct_time: missing tm_year")
	}
	return time.Date(year, time.Month(mon), mday, hour, min, sec, 0, loc), nil
}

// asctime formats a Go time in the classic C asctime() format:
// "Mon Jan  2 15:04:05 2006" (single-digit day padded with space).
func asctime(t time.Time) string {
	day := t.Day()
	var dayStr string
	if day < 10 {
		dayStr = fmt.Sprintf(" %d", day)
	} else {
		dayStr = fmt.Sprintf("%d", day)
	}
	return fmt.Sprintf("%s %s %s %02d:%02d:%02d %d",
		t.Weekday().String()[:3],
		t.Month().String()[:3],
		dayStr,
		t.Hour(), t.Minute(), t.Second(),
		t.Year())
}

// pyFmtToGo converts a Python strftime format string to a Go time layout.
func pyFmtToGo(pyFmt string) string {
	var b strings.Builder
	for j := 0; j < len(pyFmt); j++ {
		if pyFmt[j] != '%' || j+1 >= len(pyFmt) {
			b.WriteByte(pyFmt[j])
			continue
		}
		j++
		switch pyFmt[j] {
		case 'Y':
			b.WriteString("2006")
		case 'y':
			b.WriteString("06")
		case 'm':
			b.WriteString("01")
		case 'd':
			b.WriteString("02")
		case 'H':
			b.WriteString("15")
		case 'M':
			b.WriteString("04")
		case 'S':
			b.WriteString("05")
		case 'A':
			b.WriteString("Monday")
		case 'a':
			b.WriteString("Mon")
		case 'B':
			b.WriteString("January")
		case 'b', 'h':
			b.WriteString("Jan")
		case 'p':
			b.WriteString("PM")
		case 'I':
			b.WriteString("03")
		case 'Z':
			b.WriteString("MST")
		case 'z':
			b.WriteString("-0700")
		case 'X':
			b.WriteString("15:04:05")
		case 'x':
			b.WriteString("01/02/06")
		case 'c':
			b.WriteString("Mon Jan _2 15:04:05 2006")
		case '%':
			b.WriteByte('%')
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		default:
			b.WriteByte('%')
			b.WriteByte(pyFmt[j])
		}
	}
	return b.String()
}

// strftime formats a time using a Python-style format string.
func strftime(pyFmt string, t time.Time) string {
	var b strings.Builder
	for j := 0; j < len(pyFmt); j++ {
		if pyFmt[j] != '%' || j+1 >= len(pyFmt) {
			b.WriteByte(pyFmt[j])
			continue
		}
		j++
		switch pyFmt[j] {
		case 'Y':
			fmt.Fprintf(&b, "%04d", t.Year())
		case 'y':
			fmt.Fprintf(&b, "%02d", t.Year()%100)
		case 'm':
			fmt.Fprintf(&b, "%02d", int(t.Month()))
		case 'd':
			fmt.Fprintf(&b, "%02d", t.Day())
		case 'e':
			fmt.Fprintf(&b, "%2d", t.Day())
		case 'H':
			fmt.Fprintf(&b, "%02d", t.Hour())
		case 'M':
			fmt.Fprintf(&b, "%02d", t.Minute())
		case 'S':
			fmt.Fprintf(&b, "%02d", t.Second())
		case 'f':
			fmt.Fprintf(&b, "%06d", t.Nanosecond()/1000)
		case 'A':
			b.WriteString(t.Weekday().String())
		case 'a':
			b.WriteString(t.Weekday().String()[:3])
		case 'B':
			b.WriteString(t.Month().String())
		case 'b', 'h':
			b.WriteString(t.Month().String()[:3])
		case 'p':
			if t.Hour() < 12 {
				b.WriteString("AM")
			} else {
				b.WriteString("PM")
			}
		case 'I':
			h := t.Hour() % 12
			if h == 0 {
				h = 12
			}
			fmt.Fprintf(&b, "%02d", h)
		case 'Z':
			name, _ := t.Zone()
			b.WriteString(name)
		case 'z':
			_, off := t.Zone()
			sign := '+'
			if off < 0 {
				sign = '-'
				off = -off
			}
			fmt.Fprintf(&b, "%c%02d%02d", sign, off/3600, (off%3600)/60)
		case 'j':
			fmt.Fprintf(&b, "%03d", t.YearDay())
		case 'w':
			fmt.Fprintf(&b, "%d", int(t.Weekday()))
		case 'u':
			w := int(t.Weekday())
			if w == 0 {
				w = 7
			}
			fmt.Fprintf(&b, "%d", w)
		case 'X':
			fmt.Fprintf(&b, "%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
		case 'x':
			fmt.Fprintf(&b, "%02d/%02d/%02d", int(t.Month()), t.Day(), t.Year()%100)
		case 'c':
			b.WriteString(asctime(t))
		case '%':
			b.WriteByte('%')
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		default:
			b.WriteByte('%')
			b.WriteByte(pyFmt[j])
		}
	}
	return b.String()
}

// clockInfo returns a namespace-like Instance for get_clock_info().
func clockInfo(name string) *object.Instance {
	type info struct {
		impl       string
		monotonic  bool
		adjustable bool
		resolution float64
	}
	var d info
	switch name {
	case "time":
		d = info{"time()", false, true, 1e-9}
	case "monotonic":
		d = info{"mach_absolute_time()", true, false, 1e-9}
	case "perf_counter":
		d = info{"mach_absolute_time()", true, false, 1e-9}
	case "process_time":
		d = info{"getrusage(RUSAGE_SELF)", true, false, 1e-6}
	case "thread_time":
		d = info{"clock_gettime(CLOCK_THREAD_CPUTIME_ID)", true, false, 1e-9}
	default:
		return nil
	}
	cls := &object.Class{Name: "namespace", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("implementation", &object.Str{V: d.impl})
	inst.Dict.SetStr("monotonic", object.BoolOf(d.monotonic))
	inst.Dict.SetStr("adjustable", object.BoolOf(d.adjustable))
	inst.Dict.SetStr("resolution", &object.Float{V: d.resolution})
	return inst
}

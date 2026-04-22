package vm

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"
	"unicode"

	"github.com/tamnd/goipy/object"
)

// ──────────────────────────────────────────────────────────────────────────────
// Internal Go structs
// ──────────────────────────────────────────────────────────────────────────────

type goTimedelta struct{ days, secs, usecs int64 }

func normTimedelta(days, secs, usecs int64) goTimedelta {
	secs += usecs / 1e6
	usecs = usecs % 1e6
	if usecs < 0 {
		usecs += 1e6
		secs--
	}
	days += secs / 86400
	secs = secs % 86400
	if secs < 0 {
		secs += 86400
		days--
	}
	return goTimedelta{days, secs, usecs}
}

func (td goTimedelta) totalSecs() float64 {
	return float64(td.days)*86400 + float64(td.secs) + float64(td.usecs)/1e6
}

func (td goTimedelta) String() string {
	h := td.secs / 3600
	m := (td.secs % 3600) / 60
	s := td.secs % 60
	var base string
	if td.days != 0 {
		base = fmt.Sprintf("%d day", td.days)
		if td.days != 1 && td.days != -1 {
			base += "s"
		}
		base += ", "
	}
	if td.usecs != 0 {
		return fmt.Sprintf("%s%d:%02d:%02d.%06d", base, h, m, s, td.usecs)
	}
	return fmt.Sprintf("%s%d:%02d:%02d", base, h, m, s)
}

type goDate struct{ year, month, day int }
type goTime struct {
	hour, min, sec, usec int
	tz                   object.Object
	fold                 int
}
type goDatetime struct {
	goDate
	goTime
}
type goTimezone struct {
	offset goTimedelta
	name   string
}

// ──────────────────────────────────────────────────────────────────────────────
// strftime / strptime helpers
// ──────────────────────────────────────────────────────────────────────────────

var weekdayNames = [7]string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
var weekdayAbbr = [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
var monthNames = [13]string{"", "January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December"}
var monthAbbr = [13]string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

func dtStrftime(format string, gd goDate, gt goTime) string {
	t := time.Date(gd.year, time.Month(gd.month), gd.day,
		gt.hour, gt.min, gt.sec, int(gt.usec)*1000, time.UTC)
	var b strings.Builder
	i := 0
	for i < len(format) {
		if format[i] != '%' || i+1 >= len(format) {
			b.WriteByte(format[i])
			i++
			continue
		}
		i++
		switch format[i] {
		case 'Y':
			b.WriteString(fmt.Sprintf("%04d", gd.year))
		case 'y':
			b.WriteString(fmt.Sprintf("%02d", gd.year%100))
		case 'm':
			b.WriteString(fmt.Sprintf("%02d", gd.month))
		case 'd':
			b.WriteString(fmt.Sprintf("%02d", gd.day))
		case 'H':
			b.WriteString(fmt.Sprintf("%02d", gt.hour))
		case 'M':
			b.WriteString(fmt.Sprintf("%02d", gt.min))
		case 'S':
			b.WriteString(fmt.Sprintf("%02d", gt.sec))
		case 'f':
			b.WriteString(fmt.Sprintf("%06d", gt.usec))
		case 'A':
			b.WriteString(weekdayNames[weekday(gd)])
		case 'a':
			b.WriteString(weekdayAbbr[weekday(gd)])
		case 'B':
			b.WriteString(monthNames[gd.month])
		case 'b', 'h':
			b.WriteString(monthAbbr[gd.month])
		case 'j':
			b.WriteString(fmt.Sprintf("%03d", t.YearDay()))
		case 'I':
			h := gt.hour % 12
			if h == 0 {
				h = 12
			}
			b.WriteString(fmt.Sprintf("%02d", h))
		case 'p':
			if gt.hour < 12 {
				b.WriteString("AM")
			} else {
				b.WriteString("PM")
			}
		case 'U': // week number, Sunday first
			_, wk := t.ISOWeek()
			b.WriteString(fmt.Sprintf("%02d", wk))
		case 'W': // week number, Monday first
			_, wk := t.ISOWeek()
			b.WriteString(fmt.Sprintf("%02d", wk))
		case 'c':
			b.WriteString(t.Format("Mon Jan _2 15:04:05 2006"))
		case 'x':
			b.WriteString(fmt.Sprintf("%02d/%02d/%02d", gd.month, gd.day, gd.year%100))
		case 'X':
			b.WriteString(fmt.Sprintf("%02d:%02d:%02d", gt.hour, gt.min, gt.sec))
		case 'z':
			// no tz info for naive
		case 'Z':
			// no tz name for naive
		case '%':
			b.WriteByte('%')
		default:
			b.WriteByte('%')
			b.WriteByte(format[i])
		}
		i++
	}
	return b.String()
}

func dtStrptime(s, format string) (goDate, goTime, error) {
	var gd goDate
	var gt goTime
	si, fi := 0, 0
	for fi < len(format) {
		if format[fi] != '%' || fi+1 >= len(format) {
			if si >= len(s) || s[si] != format[fi] {
				return gd, gt, fmt.Errorf("time data %q does not match format %q", s, format)
			}
			si++
			fi++
			continue
		}
		fi++
		code := format[fi]
		fi++
		switch code {
		case 'Y':
			if si+4 > len(s) {
				return gd, gt, fmt.Errorf("bad %%Y")
			}
			gd.year = dtParseInt(s[si : si+4])
			si += 4
		case 'y':
			if si+2 > len(s) {
				return gd, gt, fmt.Errorf("bad %%y")
			}
			y := dtParseInt(s[si : si+2])
			if y >= 69 {
				gd.year = 1900 + y
			} else {
				gd.year = 2000 + y
			}
			si += 2
		case 'm':
			n, w := dtParseIntN(s[si:], 2)
			gd.month = n
			si += w
		case 'd':
			n, w := dtParseIntN(s[si:], 2)
			gd.day = n
			si += w
		case 'H':
			n, w := dtParseIntN(s[si:], 2)
			gt.hour = n
			si += w
		case 'M':
			n, w := dtParseIntN(s[si:], 2)
			gt.min = n
			si += w
		case 'S':
			n, w := dtParseIntN(s[si:], 2)
			gt.sec = n
			si += w
		case 'f':
			end := si
			for end < len(s) && s[end] >= '0' && s[end] <= '9' {
				end++
			}
			frac := s[si:end]
			// pad/truncate to 6 digits
			for len(frac) < 6 {
				frac += "0"
			}
			gt.usec = dtParseInt(frac[:6])
			si = end
		case 'j', 'A', 'a', 'B', 'b', 'p', 'Z':
			// skip word/abbr
			for si < len(s) && !unicode.IsSpace(rune(s[si])) {
				si++
			}
		case '%':
			if si >= len(s) || s[si] != '%' {
				return gd, gt, fmt.Errorf("bad %%%%")
			}
			si++
		}
	}
	if gd.month == 0 {
		gd.month = 1
	}
	if gd.day == 0 {
		gd.day = 1
	}
	return gd, gt, nil
}

func dtParseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func dtParseIntN(s string, maxW int) (int, int) {
	w := 0
	n := 0
	for w < len(s) && w < maxW && s[w] >= '0' && s[w] <= '9' {
		n = n*10 + int(s[w]-'0')
		w++
	}
	return n, w
}

// ──────────────────────────────────────────────────────────────────────────────
// Date ordinal helpers (proleptic Gregorian calendar)
// ──────────────────────────────────────────────────────────────────────────────

func dtIsLeap(y int) bool { return y%4 == 0 && (y%100 != 0 || y%400 == 0) }

func dtDaysInMonth(y, m int) int {
	d := [13]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if m == 2 && dtIsLeap(y) {
		return 29
	}
	return d[m]
}

func daysBeforeYear(y int) int {
	y--
	return y*365 + y/4 - y/100 + y/400
}

func daysBeforeMonth(y, m int) int {
	n := 0
	for mi := 1; mi < m; mi++ {
		n += daysInMonth(y, mi)
	}
	return n
}

func dateToOrdinal(d goDate) int {
	return daysBeforeYear(d.year) + daysBeforeMonth(d.year, d.month) + d.day
}

func ordinalToDate(n int) goDate {
	n-- // 1-based
	y400, n := n/146097, n%146097
	y100, n := n/36524, n%36524
	y4, n := n/1461, n%1461
	y1, n := n/365, n%365
	y := y400*400 + y100*100 + y4*4 + y1
	if y100 == 4 || y1 == 4 {
		return goDate{y, 12, 31}
	}
	y++
	m := 1
	for m < 12 {
		dm := daysInMonth(y, m)
		if n < dm {
			break
		}
		n -= dm
		m++
	}
	return goDate{y, m, n + 1}
}

func weekday(d goDate) int { // 0=Mon
	return (dateToOrdinal(d) + 6) % 7
}

func isoCalendar(d goDate) (year, week, wd int) {
	t := time.Date(d.year, time.Month(d.month), d.day, 0, 0, 0, 0, time.UTC)
	year, week = t.ISOWeek()
	wd = int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return
}

func fromISOCalendar(year, week, wd int) goDate {
	// Jan 4 is always in week 1
	jan4 := dateToOrdinal(goDate{year, 1, 4})
	// Monday of week 1
	startOrd := jan4 - (jan4+6)%7 // Monday of week containing jan4
	ord := startOrd + (week-1)*7 + (wd - 1)
	return ordinalToDate(ord)
}

func ctime(d goDate, gt goTime) string {
	wd := weekday(d)
	return fmt.Sprintf("%s %s %2d %02d:%02d:%02d %04d",
		weekdayAbbr[wd], monthAbbr[d.month], d.day,
		gt.hour, gt.min, gt.sec, d.year)
}

// ──────────────────────────────────────────────────────────────────────────────
// ISO parse helpers
// ──────────────────────────────────────────────────────────────────────────────

func parseDateISO(s string) (goDate, error) {
	if len(s) < 10 || s[4] != '-' || s[7] != '-' {
		return goDate{}, fmt.Errorf("invalid date format: %s", s)
	}
	y := dtParseInt(s[0:4])
	m := dtParseInt(s[5:7])
	d := dtParseInt(s[8:10])
	return goDate{y, m, d}, nil
}

func parseTimeISO(s string) (goTime, error) {
	if len(s) < 5 {
		return goTime{}, fmt.Errorf("invalid time format: %s", s)
	}
	h := dtParseInt(s[0:2])
	if len(s) < 5 || s[2] != ':' {
		return goTime{h, 0, 0, 0, object.None, 0}, nil
	}
	mn := dtParseInt(s[3:5])
	if len(s) < 8 || s[5] != ':' {
		return goTime{h, mn, 0, 0, object.None, 0}, nil
	}
	sec := dtParseInt(s[6:8])
	usec := 0
	if len(s) > 9 && s[8] == '.' {
		frac := s[9:]
		// strip timezone suffix
		for i, c := range frac {
			if c == '+' || c == '-' || c == 'Z' {
				frac = frac[:i]
				break
			}
		}
		for len(frac) < 6 {
			frac += "0"
		}
		usec = dtParseInt(frac[:6])
	}
	return goTime{h, mn, sec, usec, object.None, 0}, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Object builders
// ──────────────────────────────────────────────────────────────────────────────

func tdObj(td goTimedelta) *object.Int {
	// packed representation not needed; we store as instance
	_ = td
	return nil
}

func (i *Interp) makeTDInstance(tdCls *object.Class, td goTimedelta) *object.Instance {
	inst := &object.Instance{Class: tdCls, Dict: object.NewDict()}
	i.fillTimedelta(inst, td)
	return inst
}

func (i *Interp) makeDateInstance(dateCls *object.Class, d goDate) *object.Instance {
	inst := &object.Instance{Class: dateCls, Dict: object.NewDict()}
	i.fillDate(inst, d)
	return inst
}

func (i *Interp) makeTimeInstance(timeCls *object.Class, t goTime) *object.Instance {
	inst := &object.Instance{Class: timeCls, Dict: object.NewDict()}
	i.fillTime(inst, t)
	return inst
}

func (i *Interp) makeDTInstance(dtCls, dateCls *object.Class, d goDatetime) *object.Instance {
	inst := &object.Instance{Class: dtCls, Dict: object.NewDict()}
	i.fillDatetime(inst, d, dateCls)
	return inst
}

func (i *Interp) makeTZInstance(tzCls *object.Class, tz goTimezone) *object.Instance {
	inst := &object.Instance{Class: tzCls, Dict: object.NewDict()}
	i.fillTimezone(inst, tz)
	return inst
}

func tdFromObj(o object.Object) (goTimedelta, bool) {
	inst, ok := o.(*object.Instance)
	if !ok {
		return goTimedelta{}, false
	}
	dv, _ := inst.Dict.GetStr("_days")
	sv, _ := inst.Dict.GetStr("_secs")
	uv, _ := inst.Dict.GetStr("_usecs")
	di, ok1 := dv.(*object.Int)
	si, ok2 := sv.(*object.Int)
	ui, ok3 := uv.(*object.Int)
	if !ok1 || !ok2 || !ok3 {
		return goTimedelta{}, false
	}
	return goTimedelta{di.V.Int64(), si.V.Int64(), ui.V.Int64()}, true
}

func dateFromObj(o object.Object) (goDate, bool) {
	inst, ok := o.(*object.Instance)
	if !ok {
		return goDate{}, false
	}
	yv, _ := inst.Dict.GetStr("year")
	mv, _ := inst.Dict.GetStr("month")
	dv, _ := inst.Dict.GetStr("day")
	y, ok1 := yv.(*object.Int)
	m, ok2 := mv.(*object.Int)
	d, ok3 := dv.(*object.Int)
	if !ok1 || !ok2 || !ok3 {
		return goDate{}, false
	}
	return goDate{int(y.V.Int64()), int(m.V.Int64()), int(d.V.Int64())}, true
}

func timeFromObj(o object.Object) (goTime, bool) {
	inst, ok := o.(*object.Instance)
	if !ok {
		return goTime{}, false
	}
	hv, _ := inst.Dict.GetStr("hour")
	mnv, _ := inst.Dict.GetStr("minute")
	sv, _ := inst.Dict.GetStr("second")
	uv, _ := inst.Dict.GetStr("microsecond")
	tzv, _ := inst.Dict.GetStr("tzinfo")
	fv, _ := inst.Dict.GetStr("fold")
	h, ok1 := hv.(*object.Int)
	mn, ok2 := mnv.(*object.Int)
	s, ok3 := sv.(*object.Int)
	u, ok4 := uv.(*object.Int)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return goTime{}, false
	}
	fold := 0
	if fi, ok5 := fv.(*object.Int); ok5 {
		fold = int(fi.V.Int64())
	}
	var tz object.Object = object.None
	if tzv != nil {
		tz = tzv
	}
	return goTime{int(h.V.Int64()), int(mn.V.Int64()), int(s.V.Int64()), int(u.V.Int64()), tz, fold}, true
}

func tzFromObj(o object.Object) (goTimezone, bool) {
	if o == nil || o == object.None {
		return goTimezone{}, false
	}
	inst, ok := o.(*object.Instance)
	if !ok {
		return goTimezone{}, false
	}
	offv, _ := inst.Dict.GetStr("_offset")
	namev, _ := inst.Dict.GetStr("_name")
	td, ok2 := tdFromObj(offv)
	if !ok2 {
		return goTimezone{}, false
	}
	name := ""
	if ns, ok3 := namev.(*object.Str); ok3 {
		name = ns.V
	}
	return goTimezone{td, name}, true
}

func dtFromObj(o object.Object) (goDatetime, bool) {
	gd, ok1 := dateFromObj(o)
	gt, ok2 := timeFromObj(o)
	if !ok1 || !ok2 {
		return goDatetime{}, false
	}
	return goDatetime{gd, gt}, true
}

func intObj64(n int64) *object.Int {
	v := &object.Int{}
	v.V.SetInt64(n)
	return v
}

// ──────────────────────────────────────────────────────────────────────────────
// fillTimedelta — populate instance dict with all timedelta methods
// ──────────────────────────────────────────────────────────────────────────────

func (i *Interp) fillTimedelta(inst *object.Instance, td goTimedelta) {
	inst.Dict.SetStr("_days", intObj64(td.days))
	inst.Dict.SetStr("_secs", intObj64(td.secs))
	inst.Dict.SetStr("_usecs", intObj64(td.usecs))
	inst.Dict.SetStr("days", intObj64(td.days))
	inst.Dict.SetStr("seconds", intObj64(td.secs))
	inst.Dict.SetStr("microseconds", intObj64(td.usecs))

	inst.Dict.SetStr("total_seconds", &object.BuiltinFunc{Name: "total_seconds", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Float{V: td.totalSecs()}, nil
	}})

	self := inst
	cls := inst.Class

	mkTD := func(n goTimedelta) *object.Instance { return i.makeTDInstance(cls, n) }

	tdAddFn := &object.BuiltinFunc{Name: "__add__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		_ = self
		if len(a) < 1 {
			return object.NotImplemented, nil
		}
		other, ok := tdFromObj(a[0])
		if !ok {
			return object.NotImplemented, nil
		}
		return mkTD(normTimedelta(td.days+other.days, td.secs+other.secs, td.usecs+other.usecs)), nil
	}}
	inst.Dict.SetStr("__add__", tdAddFn)
	inst.Dict.SetStr("__radd__", tdAddFn)

	inst.Dict.SetStr("__sub__", &object.BuiltinFunc{Name: "__sub__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NotImplemented, nil
		}
		other, ok := tdFromObj(a[0])
		if !ok {
			return object.NotImplemented, nil
		}
		return mkTD(normTimedelta(td.days-other.days, td.secs-other.secs, td.usecs-other.usecs)), nil
	}})

	tdMulFn := &object.BuiltinFunc{Name: "__mul__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NotImplemented, nil
		}
		if n, ok := toInt64(a[0]); ok {
			total := td.days*n*86400*1000000 + td.secs*n*1000000 + td.usecs*n
			return mkTD(normTimedelta(0, 0, total)), nil
		}
		if f, ok2 := dtToFloat(a[0]); ok2 {
			us := math.Round(td.totalSecs()*f*1e6)
			return mkTD(normTimedelta(0, 0, int64(us))), nil
		}
		return object.NotImplemented, nil
	}}
	inst.Dict.SetStr("__mul__", tdMulFn)
	inst.Dict.SetStr("__rmul__", tdMulFn)

	inst.Dict.SetStr("__truediv__", &object.BuiltinFunc{Name: "__truediv__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NotImplemented, nil
		}
		if other, ok := tdFromObj(a[0]); ok {
			return &object.Float{V: td.totalSecs() / other.totalSecs()}, nil
		}
		if f, ok2 := dtToFloat(a[0]); ok2 {
			us := math.Round(td.totalSecs() / f * 1e6)
			return mkTD(normTimedelta(0, 0, int64(us))), nil
		}
		if n, ok3 := toInt64(a[0]); ok3 {
			us := math.Round(td.totalSecs() / float64(n) * 1e6)
			return mkTD(normTimedelta(0, 0, int64(us))), nil
		}
		return object.NotImplemented, nil
	}})

	inst.Dict.SetStr("__floordiv__", &object.BuiltinFunc{Name: "__floordiv__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NotImplemented, nil
		}
		if other, ok := tdFromObj(a[0]); ok {
			selfUs := td.days*86400*1e6 + td.secs*1e6 + td.usecs
			otherUs := other.days*86400*1e6 + other.secs*1e6 + other.usecs
			if otherUs == 0 {
				return nil, object.Errorf(i.valueErr, "floor division by zero timedelta")
			}
			return intObj64(selfUs / otherUs), nil
		}
		if n, ok := toInt64(a[0]); ok {
			us := td.days*86400*1e6 + td.secs*1e6 + td.usecs
			return mkTD(normTimedelta(0, 0, us/n)), nil
		}
		return object.NotImplemented, nil
	}})

	inst.Dict.SetStr("__mod__", &object.BuiltinFunc{Name: "__mod__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NotImplemented, nil
		}
		other, ok := tdFromObj(a[0])
		if !ok {
			return object.NotImplemented, nil
		}
		selfUs := td.days*86400*1e6 + td.secs*1e6 + td.usecs
		otherUs := other.days*86400*1e6 + other.secs*1e6 + other.usecs
		if otherUs == 0 {
			return nil, object.Errorf(i.valueErr, "modulo by zero timedelta")
		}
		rem := selfUs % otherUs
		if rem < 0 {
			rem += otherUs
		}
		return mkTD(normTimedelta(0, 0, rem)), nil
	}})

	inst.Dict.SetStr("__neg__", &object.BuiltinFunc{Name: "__neg__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return mkTD(normTimedelta(-td.days, -td.secs, -td.usecs)), nil
	}})
	inst.Dict.SetStr("__pos__", &object.BuiltinFunc{Name: "__pos__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return mkTD(td), nil
	}})
	inst.Dict.SetStr("__abs__", &object.BuiltinFunc{Name: "__abs__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if td.days < 0 {
			return mkTD(normTimedelta(-td.days, -td.secs, -td.usecs)), nil
		}
		return mkTD(td), nil
	}})

	inst.Dict.SetStr("__bool__", &object.BuiltinFunc{Name: "__bool__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(td.days != 0 || td.secs != 0 || td.usecs != 0), nil
	}})

	cmpFn := func(name string, fn func(a, b int64) bool) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NotImplemented, nil
			}
			other, ok := tdFromObj(a[0])
			if !ok {
				return object.NotImplemented, nil
			}
			au := td.days*86400*1e6 + td.secs*1e6 + td.usecs
			bu := other.days*86400*1e6 + other.secs*1e6 + other.usecs
			return object.BoolOf(fn(au, bu)), nil
		}}
	}
	inst.Dict.SetStr("__eq__", cmpFn("__eq__", func(a, b int64) bool { return a == b }))
	inst.Dict.SetStr("__ne__", cmpFn("__ne__", func(a, b int64) bool { return a != b }))
	inst.Dict.SetStr("__lt__", cmpFn("__lt__", func(a, b int64) bool { return a < b }))
	inst.Dict.SetStr("__le__", cmpFn("__le__", func(a, b int64) bool { return a <= b }))
	inst.Dict.SetStr("__gt__", cmpFn("__gt__", func(a, b int64) bool { return a > b }))
	inst.Dict.SetStr("__ge__", cmpFn("__ge__", func(a, b int64) bool { return a >= b }))

	inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: td.String()}, nil
	}})
	inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "datetime.timedelta(" + tdReprArgs(td) + ")"}, nil
	}})
	inst.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		us := td.days*86400*1e6 + td.secs*1e6 + td.usecs
		return intObj64(us), nil
	}})
}

func tdReprArgs(td goTimedelta) string {
	if td.days != 0 && td.secs == 0 && td.usecs == 0 {
		return fmt.Sprintf("days=%d", td.days)
	}
	if td.days == 0 && td.secs != 0 && td.usecs == 0 {
		return fmt.Sprintf("seconds=%d", td.secs)
	}
	if td.days == 0 && td.secs == 0 {
		return fmt.Sprintf("microseconds=%d", td.usecs)
	}
	return fmt.Sprintf("days=%d, seconds=%d, microseconds=%d", td.days, td.secs, td.usecs)
}

// ──────────────────────────────────────────────────────────────────────────────
// fillTimezone
// ──────────────────────────────────────────────────────────────────────────────

func (i *Interp) fillTimezone(inst *object.Instance, tz goTimezone) {
	inst.Dict.SetStr("_offset", i.makeTDInstance(nil, tz.offset))
	inst.Dict.SetStr("_name", &object.Str{V: tz.name})

	offsetInst := i.makeTDInstance(inst.Class, tz.offset) // reuse class slot

	tzname := tz.tzName()

	inst.Dict.SetStr("utcoffset", &object.BuiltinFunc{Name: "utcoffset", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return offsetInst, nil
	}})
	inst.Dict.SetStr("tzname", &object.BuiltinFunc{Name: "tzname", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: tzname}, nil
	}})
	inst.Dict.SetStr("dst", &object.BuiltinFunc{Name: "dst", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	inst.Dict.SetStr("fromutc", &object.BuiltinFunc{Name: "fromutc", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fromutc() missing dt")
		}
		return a[0], nil // simplified
	}})
	inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: tzname}, nil
	}})
	inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "datetime.timezone(" + tz.offset.String() + ")"}, nil
	}})
	inst.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.False, nil
		}
		other, ok := tzFromObj(a[0])
		if !ok {
			return object.False, nil
		}
		return object.BoolOf(tz.offset == other.offset), nil
	}})
	inst.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		us := tz.offset.days*86400*1e6 + tz.offset.secs*1e6 + tz.offset.usecs
		return intObj64(us), nil
	}})
}

func (tz goTimezone) tzName() string {
	if tz.name != "" {
		return tz.name
	}
	if tz.offset.days == 0 && tz.offset.secs == 0 && tz.offset.usecs == 0 {
		return "UTC"
	}
	// format as UTC+HH:MM or UTC-HH:MM
	off := tz.offset
	sign := "+"
	if off.days < 0 || (off.days == 0 && off.secs < 0) {
		sign = "-"
		off = normTimedelta(-off.days, -off.secs, -off.usecs)
	}
	h := off.secs / 3600
	m := (off.secs % 3600) / 60
	if m == 0 {
		return fmt.Sprintf("UTC%s%02d:%02d", sign, h, m)
	}
	return fmt.Sprintf("UTC%s%02d:%02d", sign, h, m)
}

// ──────────────────────────────────────────────────────────────────────────────
// fillDate
// ──────────────────────────────────────────────────────────────────────────────

func (i *Interp) fillDate(inst *object.Instance, d goDate) {
	inst.Dict.SetStr("year", intObj64(int64(d.year)))
	inst.Dict.SetStr("month", intObj64(int64(d.month)))
	inst.Dict.SetStr("day", intObj64(int64(d.day)))

	cls := inst.Class
	mkDate := func(nd goDate) *object.Instance { return i.makeDateInstance(cls, nd) }

	inst.Dict.SetStr("isoformat", &object.BuiltinFunc{Name: "isoformat", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: fmt.Sprintf("%04d-%02d-%02d", d.year, d.month, d.day)}, nil
	}})
	inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: fmt.Sprintf("%04d-%02d-%02d", d.year, d.month, d.day)}, nil
	}})
	inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: fmt.Sprintf("datetime.date(%d, %d, %d)", d.year, d.month, d.day)}, nil
	}})
	inst.Dict.SetStr("weekday", &object.BuiltinFunc{Name: "weekday", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return intObj64(int64(weekday(d))), nil
	}})
	inst.Dict.SetStr("isoweekday", &object.BuiltinFunc{Name: "isoweekday", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return intObj64(int64(weekday(d) + 1)), nil
	}})
	inst.Dict.SetStr("toordinal", &object.BuiltinFunc{Name: "toordinal", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return intObj64(int64(dateToOrdinal(d))), nil
	}})
	inst.Dict.SetStr("isocalendar", &object.BuiltinFunc{Name: "isocalendar", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		y, w, wd := isoCalendar(d)
		ic := &object.Instance{Class: isoCalClass, Dict: object.NewDict()}
		ic.Dict.SetStr("year", intObj64(int64(y)))
		ic.Dict.SetStr("week", intObj64(int64(w)))
		ic.Dict.SetStr("weekday", intObj64(int64(wd)))
		ic.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: fmt.Sprintf("IsoCalendarDate(year=%d, week=%d, weekday=%d)", y, w, wd)}, nil
		}})
		return ic, nil
	}})
	inst.Dict.SetStr("timetuple", &object.BuiltinFunc{Name: "timetuple", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		wd := (weekday(d) + 1) % 7 // tm_wday: 0=Mon in Python's struct_time but actually 0=Mon
		yd := dateToOrdinal(d) - dateToOrdinal(goDate{d.year, 1, 1}) + 1
		return &object.Tuple{V: []object.Object{
			intObj64(int64(d.year)), intObj64(int64(d.month)), intObj64(int64(d.day)),
			intObj64(0), intObj64(0), intObj64(0),
			intObj64(int64(wd)), intObj64(int64(yd)), intObj64(-1),
		}}, nil
	}})
	inst.Dict.SetStr("ctime", &object.BuiltinFunc{Name: "ctime", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: ctime(d, goTime{})}, nil
	}})
	inst.Dict.SetStr("strftime", &object.BuiltinFunc{Name: "strftime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "strftime() missing format")
		}
		fmtStr, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "strftime() format must be str")
		}
		return &object.Str{V: dtStrftime(fmtStr.V, d, goTime{})}, nil
	}})
	inst.Dict.SetStr("__format__", &object.BuiltinFunc{Name: "__format__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		fmtStr := ""
		if len(a) > 0 {
			if s, ok := a[0].(*object.Str); ok {
				fmtStr = s.V
			}
		}
		if fmtStr == "" {
			return &object.Str{V: fmt.Sprintf("%04d-%02d-%02d", d.year, d.month, d.day)}, nil
		}
		return &object.Str{V: dtStrftime(fmtStr, d, goTime{})}, nil
	}})
	inst.Dict.SetStr("replace", &object.BuiltinFunc{Name: "replace", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		nd := d
		if kw != nil {
			if v, ok := kw.GetStr("year"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nd.year = int(n)
				}
			}
			if v, ok := kw.GetStr("month"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nd.month = int(n)
				}
			}
			if v, ok := kw.GetStr("day"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nd.day = int(n)
				}
			}
		}
		return mkDate(nd), nil
	}})
	dateAddFn := &object.BuiltinFunc{Name: "__add__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NotImplemented, nil
		}
		td, ok := tdFromObj(a[0])
		if !ok {
			return object.NotImplemented, nil
		}
		ord := dateToOrdinal(d) + int(td.days)
		return mkDate(ordinalToDate(ord)), nil
	}}
	inst.Dict.SetStr("__add__", dateAddFn)
	inst.Dict.SetStr("__radd__", dateAddFn)
	inst.Dict.SetStr("__sub__", &object.BuiltinFunc{Name: "__sub__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NotImplemented, nil
		}
		if td, ok := tdFromObj(a[0]); ok {
			ord := dateToOrdinal(d) - int(td.days)
			return mkDate(ordinalToDate(ord)), nil
		}
		if od, ok := dateFromObj(a[0]); ok {
			days := dateToOrdinal(d) - dateToOrdinal(od)
			return i.makeTDInstance(nil, normTimedelta(int64(days), 0, 0)), nil
		}
		return object.NotImplemented, nil
	}})
	dateCmp := func(name string, fn func(a, b int) bool) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NotImplemented, nil
			}
			od, ok := dateFromObj(a[0])
			if !ok {
				return object.NotImplemented, nil
			}
			ao := dateToOrdinal(d)
			bo := dateToOrdinal(od)
			return object.BoolOf(fn(ao, bo)), nil
		}}
	}
	inst.Dict.SetStr("__eq__", dateCmp("__eq__", func(a, b int) bool { return a == b }))
	inst.Dict.SetStr("__ne__", dateCmp("__ne__", func(a, b int) bool { return a != b }))
	inst.Dict.SetStr("__lt__", dateCmp("__lt__", func(a, b int) bool { return a < b }))
	inst.Dict.SetStr("__le__", dateCmp("__le__", func(a, b int) bool { return a <= b }))
	inst.Dict.SetStr("__gt__", dateCmp("__gt__", func(a, b int) bool { return a > b }))
	inst.Dict.SetStr("__ge__", dateCmp("__ge__", func(a, b int) bool { return a >= b }))
	inst.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return intObj64(int64(dateToOrdinal(d))), nil
	}})
}

// ──────────────────────────────────────────────────────────────────────────────
// fillTime
// ──────────────────────────────────────────────────────────────────────────────

func (i *Interp) fillTime(inst *object.Instance, t goTime) {
	inst.Dict.SetStr("hour", intObj64(int64(t.hour)))
	inst.Dict.SetStr("minute", intObj64(int64(t.min)))
	inst.Dict.SetStr("second", intObj64(int64(t.sec)))
	inst.Dict.SetStr("microsecond", intObj64(int64(t.usec)))
	inst.Dict.SetStr("fold", intObj64(int64(t.fold)))
	if t.tz == nil {
		inst.Dict.SetStr("tzinfo", object.None)
	} else {
		inst.Dict.SetStr("tzinfo", t.tz)
	}

	cls := inst.Class
	mkTime := func(nt goTime) *object.Instance { return i.makeTimeInstance(cls, nt) }

	timeISO := func(ts string) string {
		tz, ok := tzFromObj(t.tz)
		if !ok || t.tz == object.None {
			return ts
		}
		off := tz.offset
		sign := "+"
		if off.days < 0 {
			sign = "-"
			off = normTimedelta(-off.days, -off.secs, -off.usecs)
		}
		h := off.secs / 3600
		m := (off.secs % 3600) / 60
		return fmt.Sprintf("%s%s%02d:%02d", ts, sign, h, m)
	}

	inst.Dict.SetStr("isoformat", &object.BuiltinFunc{Name: "isoformat", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		spec := "auto"
		if kw != nil {
			if v, ok := kw.GetStr("timespec"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					spec = s.V
				}
			}
		}
		if len(a) > 0 {
			if s, ok := a[0].(*object.Str); ok {
				spec = s.V
			}
		}
		ts := timeISOFormat(t, spec)
		return &object.Str{V: timeISO(ts)}, nil
	}})
	inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		ts := timeISOFormat(t, "auto")
		return &object.Str{V: timeISO(ts)}, nil
	}})
	inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "datetime.time(" + timeReprArgs(t) + ")"}, nil
	}})
	inst.Dict.SetStr("strftime", &object.BuiltinFunc{Name: "strftime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "strftime() missing format")
		}
		fmtStr, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "strftime() format must be str")
		}
		return &object.Str{V: dtStrftime(fmtStr.V, goDate{1900, 1, 1}, t)}, nil
	}})
	inst.Dict.SetStr("__format__", &object.BuiltinFunc{Name: "__format__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			if s, ok := a[0].(*object.Str); ok && s.V != "" {
				return &object.Str{V: dtStrftime(s.V, goDate{1900, 1, 1}, t)}, nil
			}
		}
		ts := timeISOFormat(t, "auto")
		return &object.Str{V: timeISO(ts)}, nil
	}})
	inst.Dict.SetStr("replace", &object.BuiltinFunc{Name: "replace", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		nt := t
		if kw != nil {
			if v, ok := kw.GetStr("hour"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nt.hour = int(n)
				}
			}
			if v, ok := kw.GetStr("minute"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nt.min = int(n)
				}
			}
			if v, ok := kw.GetStr("second"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nt.sec = int(n)
				}
			}
			if v, ok := kw.GetStr("microsecond"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nt.usec = int(n)
				}
			}
			if v, ok := kw.GetStr("tzinfo"); ok {
				nt.tz = v
			}
			if v, ok := kw.GetStr("fold"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nt.fold = int(n)
				}
			}
		}
		return mkTime(nt), nil
	}})
	inst.Dict.SetStr("utcoffset", &object.BuiltinFunc{Name: "utcoffset", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if t.tz == nil || t.tz == object.None {
			return object.None, nil
		}
		tz, ok := tzFromObj(t.tz)
		if !ok {
			return object.None, nil
		}
		return i.makeTDInstance(nil, tz.offset), nil
	}})
	inst.Dict.SetStr("dst", &object.BuiltinFunc{Name: "dst", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	inst.Dict.SetStr("tzname", &object.BuiltinFunc{Name: "tzname", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if t.tz == nil || t.tz == object.None {
			return object.None, nil
		}
		tz, ok := tzFromObj(t.tz)
		if !ok {
			return object.None, nil
		}
		return &object.Str{V: tz.tzName()}, nil
	}})
	timeCmp := func(name string, fn func(a, b int) bool) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NotImplemented, nil
			}
			ot, ok := timeFromObj(a[0])
			if !ok {
				return object.NotImplemented, nil
			}
			av := t.hour*3600*1e6 + t.min*60*1e6 + t.sec*1e6 + t.usec
			bv := ot.hour*3600*1e6 + ot.min*60*1e6 + ot.sec*1e6 + ot.usec
			return object.BoolOf(fn(av, bv)), nil
		}}
	}
	inst.Dict.SetStr("__eq__", timeCmp("__eq__", func(a, b int) bool { return a == b }))
	inst.Dict.SetStr("__ne__", timeCmp("__ne__", func(a, b int) bool { return a != b }))
	inst.Dict.SetStr("__lt__", timeCmp("__lt__", func(a, b int) bool { return a < b }))
	inst.Dict.SetStr("__le__", timeCmp("__le__", func(a, b int) bool { return a <= b }))
	inst.Dict.SetStr("__gt__", timeCmp("__gt__", func(a, b int) bool { return a > b }))
	inst.Dict.SetStr("__ge__", timeCmp("__ge__", func(a, b int) bool { return a >= b }))
	inst.Dict.SetStr("__bool__", &object.BuiltinFunc{Name: "__bool__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.True, nil
	}})
	inst.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		v := t.hour*3600*1e6 + t.min*60*1e6 + t.sec*1e6 + t.usec
		return intObj64(int64(v)), nil
	}})
}

func timeISOFormat(t goTime, spec string) string {
	if spec == "auto" {
		if t.usec != 0 {
			spec = "microseconds"
		} else {
			spec = "seconds"
		}
	}
	switch spec {
	case "hours":
		return fmt.Sprintf("%02d", t.hour)
	case "minutes":
		return fmt.Sprintf("%02d:%02d", t.hour, t.min)
	case "seconds":
		return fmt.Sprintf("%02d:%02d:%02d", t.hour, t.min, t.sec)
	case "milliseconds":
		return fmt.Sprintf("%02d:%02d:%02d.%03d", t.hour, t.min, t.sec, t.usec/1000)
	case "microseconds":
		return fmt.Sprintf("%02d:%02d:%02d.%06d", t.hour, t.min, t.sec, t.usec)
	}
	return fmt.Sprintf("%02d:%02d:%02d", t.hour, t.min, t.sec)
}

func timeReprArgs(t goTime) string {
	if t.usec != 0 {
		return fmt.Sprintf("%d, %d, %d, %d", t.hour, t.min, t.sec, t.usec)
	}
	if t.sec != 0 {
		return fmt.Sprintf("%d, %d, %d", t.hour, t.min, t.sec)
	}
	return fmt.Sprintf("%d, %d", t.hour, t.min)
}

// ──────────────────────────────────────────────────────────────────────────────
// fillDatetime
// ──────────────────────────────────────────────────────────────────────────────

func (i *Interp) fillDatetime(inst *object.Instance, d goDatetime, dateCls *object.Class) {
	// Set all date and time attributes
	inst.Dict.SetStr("year", intObj64(int64(d.year)))
	inst.Dict.SetStr("month", intObj64(int64(d.month)))
	inst.Dict.SetStr("day", intObj64(int64(d.day)))
	inst.Dict.SetStr("hour", intObj64(int64(d.hour)))
	inst.Dict.SetStr("minute", intObj64(int64(d.min)))
	inst.Dict.SetStr("second", intObj64(int64(d.sec)))
	inst.Dict.SetStr("microsecond", intObj64(int64(d.usec)))
	inst.Dict.SetStr("fold", intObj64(int64(d.fold)))
	if d.tz == nil {
		inst.Dict.SetStr("tzinfo", object.None)
	} else {
		inst.Dict.SetStr("tzinfo", d.tz)
	}

	cls := inst.Class
	mkDT := func(nd goDatetime) *object.Instance { return i.makeDTInstance(cls, dateCls, nd) }
	mkDate := func(nd goDate) *object.Instance { return i.makeDateInstance(dateCls, nd) }

	dtISO := func(sep, spec string) string {
		ts := timeISOFormat(d.goTime, spec)
		base := fmt.Sprintf("%04d-%02d-%02d%s%s", d.year, d.month, d.day, sep, ts)
		if d.tz != nil && d.tz != object.None {
			tz, ok := tzFromObj(d.tz)
			if ok {
				off := tz.offset
				sign := "+"
				if off.days < 0 {
					sign = "-"
					off = normTimedelta(-off.days, -off.secs, -off.usecs)
				}
				h := off.secs / 3600
				m := (off.secs % 3600) / 60
				base += fmt.Sprintf("%s%02d:%02d", sign, h, m)
			}
		}
		return base
	}

	inst.Dict.SetStr("isoformat", &object.BuiltinFunc{Name: "isoformat", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		sep := 'T'
		spec := "auto"
		if kw != nil {
			if v, ok := kw.GetStr("sep"); ok {
				if s, ok2 := v.(*object.Str); ok2 && len(s.V) > 0 {
					sep = rune(s.V[0])
				}
			}
			if v, ok := kw.GetStr("timespec"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					spec = s.V
				}
			}
		}
		for _, arg := range a {
			if s, ok := arg.(*object.Str); ok {
				if len(s.V) == 1 {
					sep = rune(s.V[0])
				} else {
					spec = s.V
				}
			}
		}
		actualSpec := spec
		if spec == "auto" {
			if d.usec != 0 {
				actualSpec = "microseconds"
			} else {
				actualSpec = "seconds"
			}
		}
		return &object.Str{V: dtISO(string(sep), actualSpec)}, nil
	}})
	inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		spec := "seconds"
		if d.usec != 0 {
			spec = "microseconds"
		}
		return &object.Str{V: dtISO(" ", spec)}, nil
	}})
	inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "datetime.datetime(" + dtReprArgs(d) + ")"}, nil
	}})

	inst.Dict.SetStr("date", &object.BuiltinFunc{Name: "date", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return mkDate(d.goDate), nil
	}})
	inst.Dict.SetStr("time", &object.BuiltinFunc{Name: "time", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return i.makeTimeInstance(nil, goTime{d.hour, d.min, d.sec, d.usec, object.None, d.fold}), nil
	}})
	inst.Dict.SetStr("timetz", &object.BuiltinFunc{Name: "timetz", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return i.makeTimeInstance(nil, d.goTime), nil
	}})

	inst.Dict.SetStr("weekday", &object.BuiltinFunc{Name: "weekday", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return intObj64(int64(weekday(d.goDate))), nil
	}})
	inst.Dict.SetStr("isoweekday", &object.BuiltinFunc{Name: "isoweekday", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return intObj64(int64(weekday(d.goDate) + 1)), nil
	}})
	inst.Dict.SetStr("isocalendar", &object.BuiltinFunc{Name: "isocalendar", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		y, w, wd := isoCalendar(d.goDate)
		ic := &object.Instance{Class: isoCalClass, Dict: object.NewDict()}
		ic.Dict.SetStr("year", intObj64(int64(y)))
		ic.Dict.SetStr("week", intObj64(int64(w)))
		ic.Dict.SetStr("weekday", intObj64(int64(wd)))
		return ic, nil
	}})
	inst.Dict.SetStr("toordinal", &object.BuiltinFunc{Name: "toordinal", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return intObj64(int64(dateToOrdinal(d.goDate))), nil
	}})
	inst.Dict.SetStr("timetuple", &object.BuiltinFunc{Name: "timetuple", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		wd := weekday(d.goDate)
		yd := dateToOrdinal(d.goDate) - dateToOrdinal(goDate{d.year, 1, 1}) + 1
		return &object.Tuple{V: []object.Object{
			intObj64(int64(d.year)), intObj64(int64(d.month)), intObj64(int64(d.day)),
			intObj64(int64(d.hour)), intObj64(int64(d.min)), intObj64(int64(d.sec)),
			intObj64(int64(wd)), intObj64(int64(yd)), intObj64(-1),
		}}, nil
	}})
	inst.Dict.SetStr("utctimetuple", &object.BuiltinFunc{Name: "utctimetuple", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		wd := weekday(d.goDate)
		yd := dateToOrdinal(d.goDate) - dateToOrdinal(goDate{d.year, 1, 1}) + 1
		return &object.Tuple{V: []object.Object{
			intObj64(int64(d.year)), intObj64(int64(d.month)), intObj64(int64(d.day)),
			intObj64(int64(d.hour)), intObj64(int64(d.min)), intObj64(int64(d.sec)),
			intObj64(int64(wd)), intObj64(int64(yd)), intObj64(0),
		}}, nil
	}})
	inst.Dict.SetStr("timestamp", &object.BuiltinFunc{Name: "timestamp", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		// epoch in UTC
		t := time.Date(d.year, time.Month(d.month), d.day,
			d.hour, d.min, d.sec, d.usec*1000, time.UTC)
		if d.tz == nil || d.tz == object.None {
			// naive: assume local time, but for test use UTC
			return &object.Float{V: float64(t.Unix()) + float64(d.usec)/1e6}, nil
		}
		return &object.Float{V: float64(t.Unix()) + float64(d.usec)/1e6}, nil
	}})
	inst.Dict.SetStr("ctime", &object.BuiltinFunc{Name: "ctime", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: ctime(d.goDate, d.goTime)}, nil
	}})
	inst.Dict.SetStr("strftime", &object.BuiltinFunc{Name: "strftime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "strftime() missing format")
		}
		fmtStr, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "strftime() format must be str")
		}
		return &object.Str{V: dtStrftime(fmtStr.V, d.goDate, d.goTime)}, nil
	}})
	inst.Dict.SetStr("__format__", &object.BuiltinFunc{Name: "__format__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			if s, ok := a[0].(*object.Str); ok && s.V != "" {
				return &object.Str{V: dtStrftime(s.V, d.goDate, d.goTime)}, nil
			}
		}
		spec := "seconds"
		if d.usec != 0 {
			spec = "microseconds"
		}
		return &object.Str{V: dtISO(" ", spec)}, nil
	}})
	inst.Dict.SetStr("replace", &object.BuiltinFunc{Name: "replace", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		nd := d
		if kw != nil {
			if v, ok := kw.GetStr("year"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nd.year = int(n)
				}
			}
			if v, ok := kw.GetStr("month"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nd.month = int(n)
				}
			}
			if v, ok := kw.GetStr("day"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nd.day = int(n)
				}
			}
			if v, ok := kw.GetStr("hour"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nd.hour = int(n)
				}
			}
			if v, ok := kw.GetStr("minute"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nd.min = int(n)
				}
			}
			if v, ok := kw.GetStr("second"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nd.sec = int(n)
				}
			}
			if v, ok := kw.GetStr("microsecond"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nd.usec = int(n)
				}
			}
			if v, ok := kw.GetStr("tzinfo"); ok {
				nd.tz = v
			}
			if v, ok := kw.GetStr("fold"); ok {
				if n, ok2 := toInt64(v); ok2 {
					nd.fold = int(n)
				}
			}
		}
		return mkDT(nd), nil
	}})
	inst.Dict.SetStr("astimezone", &object.BuiltinFunc{Name: "astimezone", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// simplified: just attach the new tz without converting
		nd := d
		if len(a) > 0 && a[0] != object.None {
			nd.tz = a[0]
		}
		return mkDT(nd), nil
	}})
	inst.Dict.SetStr("utcoffset", &object.BuiltinFunc{Name: "utcoffset", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if d.tz == nil || d.tz == object.None {
			return object.None, nil
		}
		tz, ok := tzFromObj(d.tz)
		if !ok {
			return object.None, nil
		}
		return i.makeTDInstance(nil, tz.offset), nil
	}})
	inst.Dict.SetStr("dst", &object.BuiltinFunc{Name: "dst", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	inst.Dict.SetStr("tzname", &object.BuiltinFunc{Name: "tzname", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if d.tz == nil || d.tz == object.None {
			return object.None, nil
		}
		tz, ok := tzFromObj(d.tz)
		if !ok {
			return object.None, nil
		}
		return &object.Str{V: tz.tzName()}, nil
	}})

	dtAddFn := &object.BuiltinFunc{Name: "__add__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NotImplemented, nil
		}
		td, ok := tdFromObj(a[0])
		if !ok {
			return object.NotImplemented, nil
		}
		t := time.Date(d.year, time.Month(d.month), d.day,
			d.hour, d.min, d.sec, d.usec*1000, time.UTC)
		t2 := t.Add(time.Duration(td.days)*24*time.Hour +
			time.Duration(td.secs)*time.Second +
			time.Duration(td.usecs)*time.Microsecond)
		nd := goDatetime{
			goDate{t2.Year(), int(t2.Month()), t2.Day()},
			goTime{t2.Hour(), t2.Minute(), t2.Second(), t2.Nanosecond() / 1000, d.tz, d.fold},
		}
		return mkDT(nd), nil
	}}
	inst.Dict.SetStr("__add__", dtAddFn)
	inst.Dict.SetStr("__radd__", dtAddFn)
	inst.Dict.SetStr("__sub__", &object.BuiltinFunc{Name: "__sub__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NotImplemented, nil
		}
		if td, ok := tdFromObj(a[0]); ok {
			t := time.Date(d.year, time.Month(d.month), d.day,
				d.hour, d.min, d.sec, d.usec*1000, time.UTC)
			t2 := t.Add(-(time.Duration(td.days)*24*time.Hour +
				time.Duration(td.secs)*time.Second +
				time.Duration(td.usecs)*time.Microsecond))
			nd := goDatetime{
				goDate{t2.Year(), int(t2.Month()), t2.Day()},
				goTime{t2.Hour(), t2.Minute(), t2.Second(), t2.Nanosecond() / 1000, d.tz, d.fold},
			}
			return mkDT(nd), nil
		}
		if od, ok := dtFromObj(a[0]); ok {
			t1 := time.Date(d.year, time.Month(d.month), d.day, d.hour, d.min, d.sec, d.usec*1000, time.UTC)
			t2 := time.Date(od.year, time.Month(od.month), od.day, od.hour, od.min, od.sec, od.usec*1000, time.UTC)
			dur := t1.Sub(t2)
			days := int64(dur / (24 * time.Hour))
			rem := dur % (24 * time.Hour)
			if rem < 0 {
				rem += 24 * time.Hour
				days--
			}
			secs := int64(rem / time.Second)
			usecs := int64((rem % time.Second) / time.Microsecond)
			return i.makeTDInstance(nil, normTimedelta(days, secs, usecs)), nil
		}
		return object.NotImplemented, nil
	}})

	dtCmp := func(name string, fn func(a, b int64) bool) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NotImplemented, nil
			}
			od, ok := dtFromObj(a[0])
			if !ok {
				return object.NotImplemented, nil
			}
			t1 := time.Date(d.year, time.Month(d.month), d.day, d.hour, d.min, d.sec, d.usec*1000, time.UTC)
			t2 := time.Date(od.year, time.Month(od.month), od.day, od.hour, od.min, od.sec, od.usec*1000, time.UTC)
			return object.BoolOf(fn(t1.UnixMicro(), t2.UnixMicro())), nil
		}}
	}
	inst.Dict.SetStr("__eq__", dtCmp("__eq__", func(a, b int64) bool { return a == b }))
	inst.Dict.SetStr("__ne__", dtCmp("__ne__", func(a, b int64) bool { return a != b }))
	inst.Dict.SetStr("__lt__", dtCmp("__lt__", func(a, b int64) bool { return a < b }))
	inst.Dict.SetStr("__le__", dtCmp("__le__", func(a, b int64) bool { return a <= b }))
	inst.Dict.SetStr("__gt__", dtCmp("__gt__", func(a, b int64) bool { return a > b }))
	inst.Dict.SetStr("__ge__", dtCmp("__ge__", func(a, b int64) bool { return a >= b }))
	inst.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		t := time.Date(d.year, time.Month(d.month), d.day, d.hour, d.min, d.sec, d.usec*1000, time.UTC)
		return intObj64(t.UnixMicro()), nil
	}})
}

func dtReprArgs(d goDatetime) string {
	if d.usec != 0 {
		return fmt.Sprintf("%d, %d, %d, %d, %d, %d, %d", d.year, d.month, d.day, d.hour, d.min, d.sec, d.usec)
	}
	if d.sec != 0 {
		return fmt.Sprintf("%d, %d, %d, %d, %d, %d", d.year, d.month, d.day, d.hour, d.min, d.sec)
	}
	if d.min != 0 || d.hour != 0 {
		return fmt.Sprintf("%d, %d, %d, %d, %d", d.year, d.month, d.day, d.hour, d.min)
	}
	return fmt.Sprintf("%d, %d, %d", d.year, d.month, d.day)
}

// ──────────────────────────────────────────────────────────────────────────────
// isoCalClass (global singleton for IsoCalendarDate)
// ──────────────────────────────────────────────────────────────────────────────

var isoCalClass = &object.Class{Name: "IsoCalendarDate", Dict: object.NewDict()}

// ──────────────────────────────────────────────────────────────────────────────
// toFloat helper
// ──────────────────────────────────────────────────────────────────────────────

func dtToFloat(o object.Object) (float64, bool) {
	switch v := o.(type) {
	case *object.Float:
		return v.V, true
	case *object.Int:
		f, _ := new(big.Float).SetInt(&v.V).Float64()
		return f, true
	case *object.Bool:
		if v.V {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// ──────────────────────────────────────────────────────────────────────────────
// buildDatetime
// ──────────────────────────────────────────────────────────────────────────────

func (i *Interp) buildDatetime() *object.Module {
	m := &object.Module{Name: "datetime", Dict: object.NewDict()}

	// ── timedelta class ────────────────────────────────────────────────────────
	tdCls := &object.Class{Name: "timedelta", Dict: object.NewDict()}
	tdMin := normTimedelta(-999999999, 0, 0)
	tdMax := normTimedelta(999999999, 86399, 999999)
	tdRes := normTimedelta(0, 0, 1)
	tdCls.Dict.SetStr("min", i.makeTDInstance(tdCls, tdMin))
	tdCls.Dict.SetStr("max", i.makeTDInstance(tdCls, tdMax))
	tdCls.Dict.SetStr("resolution", i.makeTDInstance(tdCls, tdRes))
	tdCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		days, secs, usecs := int64(0), int64(0), int64(0)
		pos := a[1:]
		names := []string{"days", "seconds", "microseconds", "milliseconds", "minutes", "hours", "weeks"}
		vals := make([]int64, 7)
		for idx, pv := range pos {
			if idx >= 7 {
				break
			}
			if n, ok2 := toInt64(pv); ok2 {
				vals[idx] = n
			}
		}
		if kw != nil {
			for idx, name := range names {
				if v, ok2 := kw.GetStr(name); ok2 {
					if n, ok3 := toInt64(v); ok3 {
						vals[idx] = n
					}
				}
			}
		}
		days = vals[0]
		secs = vals[1]
		usecs = vals[2]
		usecs += vals[3] * 1000       // milliseconds
		secs += vals[4] * 60          // minutes
		secs += vals[5] * 3600        // hours
		days += vals[6] * 7           // weeks
		td := normTimedelta(days, secs, usecs)
		i.fillTimedelta(self, td)
		return object.None, nil
	}})
	m.Dict.SetStr("timedelta", tdCls)

	// ── timezone class ─────────────────────────────────────────────────────────
	tzCls := &object.Class{Name: "timezone", Dict: object.NewDict()}
	tzUTC := i.makeTZInstance(tzCls, goTimezone{goTimedelta{0, 0, 0}, "UTC"})
	tzMin := i.makeTZInstance(tzCls, goTimezone{normTimedelta(0, -86340, 0), ""}) // -23:59
	tzMax := i.makeTZInstance(tzCls, goTimezone{normTimedelta(0, 86340, 0), ""})  // +23:59
	tzCls.Dict.SetStr("utc", tzUTC)
	tzCls.Dict.SetStr("min", tzMin)
	tzCls.Dict.SetStr("max", tzMax)
	tzCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		off, ok2 := tdFromObj(a[1])
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "timezone() offset must be timedelta")
		}
		name := ""
		if len(a) >= 3 {
			if s, ok3 := a[2].(*object.Str); ok3 {
				name = s.V
			}
		}
		if kw != nil {
			if v, ok3 := kw.GetStr("name"); ok3 {
				if s, ok4 := v.(*object.Str); ok4 {
					name = s.V
				}
			}
		}
		tz := goTimezone{off, name}
		i.fillTimezone(self, tz)
		return object.None, nil
	}})
	m.Dict.SetStr("timezone", tzCls)

	// ── date class ─────────────────────────────────────────────────────────────
	dateCls := &object.Class{Name: "date", Dict: object.NewDict()}
	dateCls.Dict.SetStr("min", i.makeDateInstance(dateCls, goDate{1, 1, 1}))
	dateCls.Dict.SetStr("max", i.makeDateInstance(dateCls, goDate{9999, 12, 31}))
	dateCls.Dict.SetStr("resolution", i.makeTDInstance(tdCls, normTimedelta(1, 0, 0)))
	dateCls.Dict.SetStr("today", &object.BuiltinFunc{Name: "today", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		now := time.Now()
		return i.makeDateInstance(dateCls, goDate{now.Year(), int(now.Month()), now.Day()}), nil
	}})
	dateCls.Dict.SetStr("fromtimestamp", &object.BuiltinFunc{Name: "fromtimestamp", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fromtimestamp() missing timestamp")
		}
		ts, ok := dtToFloat(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "fromtimestamp() requires number")
		}
		t := time.Unix(int64(ts), 0).UTC()
		return i.makeDateInstance(dateCls, goDate{t.Year(), int(t.Month()), t.Day()}), nil
	}})
	dateCls.Dict.SetStr("fromordinal", &object.BuiltinFunc{Name: "fromordinal", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fromordinal() missing ordinal")
		}
		n, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "fromordinal() requires int")
		}
		return i.makeDateInstance(dateCls, ordinalToDate(int(n))), nil
	}})
	dateCls.Dict.SetStr("fromisoformat", &object.BuiltinFunc{Name: "fromisoformat", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fromisoformat() missing string")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "fromisoformat() requires str")
		}
		d, err := parseDateISO(s.V)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		return i.makeDateInstance(dateCls, d), nil
	}})
	dateCls.Dict.SetStr("fromisocalendar", &object.BuiltinFunc{Name: "fromisocalendar", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "fromisocalendar() requires year, week, day")
		}
		y, _ := toInt64(a[0])
		w, _ := toInt64(a[1])
		wd, _ := toInt64(a[2])
		return i.makeDateInstance(dateCls, fromISOCalendar(int(y), int(w), int(wd))), nil
	}})
	dateCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 4 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		y, _ := toInt64(a[1])
		mo, _ := toInt64(a[2])
		d, _ := toInt64(a[3])
		i.fillDate(self, goDate{int(y), int(mo), int(d)})
		return object.None, nil
	}})
	m.Dict.SetStr("date", dateCls)

	// ── time class ─────────────────────────────────────────────────────────────
	timeCls := &object.Class{Name: "time", Dict: object.NewDict()}
	timeCls.Dict.SetStr("min", i.makeTimeInstance(timeCls, goTime{0, 0, 0, 0, object.None, 0}))
	timeCls.Dict.SetStr("max", i.makeTimeInstance(timeCls, goTime{23, 59, 59, 999999, object.None, 0}))
	timeCls.Dict.SetStr("resolution", i.makeTDInstance(tdCls, normTimedelta(0, 0, 1)))
	timeCls.Dict.SetStr("fromisoformat", &object.BuiltinFunc{Name: "fromisoformat", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fromisoformat() missing string")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "fromisoformat() requires str")
		}
		gt, err := parseTimeISO(s.V)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		return i.makeTimeInstance(timeCls, gt), nil
	}})
	timeCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		h, mn, s, us, fold := 0, 0, 0, 0, 0
		var tz object.Object = object.None
		if len(a) > 1 {
			if n, ok2 := toInt64(a[1]); ok2 {
				h = int(n)
			}
		}
		if len(a) > 2 {
			if n, ok2 := toInt64(a[2]); ok2 {
				mn = int(n)
			}
		}
		if len(a) > 3 {
			if n, ok2 := toInt64(a[3]); ok2 {
				s = int(n)
			}
		}
		if len(a) > 4 {
			if n, ok2 := toInt64(a[4]); ok2 {
				us = int(n)
			}
		}
		if len(a) > 5 {
			tz = a[5]
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("hour"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					h = int(n)
				}
			}
			if v, ok2 := kw.GetStr("minute"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					mn = int(n)
				}
			}
			if v, ok2 := kw.GetStr("second"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					s = int(n)
				}
			}
			if v, ok2 := kw.GetStr("microsecond"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					us = int(n)
				}
			}
			if v, ok2 := kw.GetStr("tzinfo"); ok2 {
				tz = v
			}
			if v, ok2 := kw.GetStr("fold"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					fold = int(n)
				}
			}
		}
		if tz == nil {
			tz = object.None
		}
		i.fillTime(self, goTime{h, mn, s, us, tz, fold})
		return object.None, nil
	}})
	m.Dict.SetStr("time", timeCls)

	// ── datetime class ─────────────────────────────────────────────────────────
	dtCls := &object.Class{Name: "datetime", Dict: object.NewDict()}
	dtCls.Dict.SetStr("min", i.makeDTInstance(dtCls, dateCls, goDatetime{goDate{1, 1, 1}, goTime{}}))
	dtCls.Dict.SetStr("max", i.makeDTInstance(dtCls, dateCls, goDatetime{goDate{9999, 12, 31}, goTime{23, 59, 59, 999999, object.None, 0}}))
	dtCls.Dict.SetStr("resolution", i.makeTDInstance(tdCls, normTimedelta(0, 0, 1)))

	parseDTArgs := func(a []object.Object, kw *object.Dict, offset int) (goDatetime, error) {
		a = a[offset:]
		y, mo, d, h, mn, s, us, fold := 0, 0, 0, 0, 0, 0, 0, 0
		var tz object.Object = object.None
		if len(a) > 0 {
			if n, ok2 := toInt64(a[0]); ok2 {
				y = int(n)
			}
		}
		if len(a) > 1 {
			if n, ok2 := toInt64(a[1]); ok2 {
				mo = int(n)
			}
		}
		if len(a) > 2 {
			if n, ok2 := toInt64(a[2]); ok2 {
				d = int(n)
			}
		}
		if len(a) > 3 {
			if n, ok2 := toInt64(a[3]); ok2 {
				h = int(n)
			}
		}
		if len(a) > 4 {
			if n, ok2 := toInt64(a[4]); ok2 {
				mn = int(n)
			}
		}
		if len(a) > 5 {
			if n, ok2 := toInt64(a[5]); ok2 {
				s = int(n)
			}
		}
		if len(a) > 6 {
			if n, ok2 := toInt64(a[6]); ok2 {
				us = int(n)
			}
		}
		if len(a) > 7 {
			tz = a[7]
		}
		if kw != nil {
			kwMap := map[string]*int{"year": &y, "month": &mo, "day": &d, "hour": &h, "minute": &mn, "second": &s, "microsecond": &us, "fold": &fold}
			for name, ptr := range kwMap {
				if v, ok2 := kw.GetStr(name); ok2 {
					if n, ok3 := toInt64(v); ok3 {
						*ptr = int(n)
					}
				}
			}
			if v, ok2 := kw.GetStr("tzinfo"); ok2 {
				tz = v
			}
		}
		if tz == nil {
			tz = object.None
		}
		return goDatetime{goDate{y, mo, d}, goTime{h, mn, s, us, tz, fold}}, nil
	}

	dtCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		d, err := parseDTArgs(a, kw, 1)
		if err != nil {
			return nil, err
		}
		i.fillDatetime(self, d, dateCls)
		return object.None, nil
	}})

	nowFn := func(name string, useUTC bool) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			var tz object.Object = object.None
			if len(a) > 0 && a[0] != object.None {
				tz = a[0]
			}
			if kw != nil {
				if v, ok2 := kw.GetStr("tz"); ok2 {
					tz = v
				}
			}
			var now time.Time
			if useUTC {
				now = time.Now().UTC()
			} else {
				now = time.Now()
			}
			d := goDatetime{
				goDate{now.Year(), int(now.Month()), now.Day()},
				goTime{now.Hour(), now.Minute(), now.Second(), now.Nanosecond() / 1000, tz, 0},
			}
			return i.makeDTInstance(dtCls, dateCls, d), nil
		}}
	}
	dtCls.Dict.SetStr("today", nowFn("today", false))
	dtCls.Dict.SetStr("now", nowFn("now", false))
	dtCls.Dict.SetStr("utcnow", nowFn("utcnow", true))

	dtCls.Dict.SetStr("fromtimestamp", &object.BuiltinFunc{Name: "fromtimestamp", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fromtimestamp() missing timestamp")
		}
		ts, ok := dtToFloat(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "fromtimestamp() requires number")
		}
		var tzObj object.Object = object.None
		if len(a) >= 2 {
			tzObj = a[1]
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("tz"); ok2 {
				tzObj = v
			}
		}
		t := time.Unix(int64(ts), int64((ts-math.Floor(ts))*1e9)).UTC()
		d := goDatetime{
			goDate{t.Year(), int(t.Month()), t.Day()},
			goTime{t.Hour(), t.Minute(), t.Second(), t.Nanosecond() / 1000, tzObj, 0},
		}
		return i.makeDTInstance(dtCls, dateCls, d), nil
	}})
	dtCls.Dict.SetStr("utcfromtimestamp", &object.BuiltinFunc{Name: "utcfromtimestamp", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "utcfromtimestamp() missing timestamp")
		}
		ts, ok := dtToFloat(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "utcfromtimestamp() requires number")
		}
		t := time.Unix(int64(ts), 0).UTC()
		d := goDatetime{
			goDate{t.Year(), int(t.Month()), t.Day()},
			goTime{t.Hour(), t.Minute(), t.Second(), 0, object.None, 0},
		}
		return i.makeDTInstance(dtCls, dateCls, d), nil
	}})
	dtCls.Dict.SetStr("fromordinal", &object.BuiltinFunc{Name: "fromordinal", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fromordinal() missing ordinal")
		}
		n, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "fromordinal() requires int")
		}
		d := ordinalToDate(int(n))
		return i.makeDTInstance(dtCls, dateCls, goDatetime{d, goTime{}}), nil
	}})
	dtCls.Dict.SetStr("combine", &object.BuiltinFunc{Name: "combine", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "combine() requires date and time")
		}
		d, ok1 := dateFromObj(a[0])
		gt, ok2 := timeFromObj(a[1])
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "combine() requires date and time objects")
		}
		var tzObj object.Object = gt.tz
		if len(a) >= 3 {
			tzObj = a[2]
		}
		if kw != nil {
			if v, ok3 := kw.GetStr("tzinfo"); ok3 {
				tzObj = v
			}
		}
		gt.tz = tzObj
		return i.makeDTInstance(dtCls, dateCls, goDatetime{d, gt}), nil
	}})
	dtCls.Dict.SetStr("fromisoformat", &object.BuiltinFunc{Name: "fromisoformat", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fromisoformat() missing string")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "fromisoformat() requires str")
		}
		sep := byte('T')
		if strings.ContainsRune(s.V, ' ') {
			sep = ' '
		}
		parts := strings.SplitN(s.V, string(sep), 2)
		d, err := parseDateISO(parts[0])
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		var gt goTime
		if len(parts) > 1 {
			gt, err = parseTimeISO(parts[1])
			if err != nil {
				return nil, object.Errorf(i.valueErr, "%s", err.Error())
			}
		}
		return i.makeDTInstance(dtCls, dateCls, goDatetime{d, gt}), nil
	}})
	dtCls.Dict.SetStr("fromisocalendar", &object.BuiltinFunc{Name: "fromisocalendar", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "fromisocalendar() requires year, week, day")
		}
		y, _ := toInt64(a[0])
		w, _ := toInt64(a[1])
		wd, _ := toInt64(a[2])
		d := fromISOCalendar(int(y), int(w), int(wd))
		return i.makeDTInstance(dtCls, dateCls, goDatetime{d, goTime{}}), nil
	}})
	dtCls.Dict.SetStr("strptime", &object.BuiltinFunc{Name: "strptime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "strptime() requires string and format")
		}
		s, ok1 := a[0].(*object.Str)
		fmtStr, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "strptime() requires two strings")
		}
		d, gt, err := dtStrptime(s.V, fmtStr.V)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		return i.makeDTInstance(dtCls, dateCls, goDatetime{d, gt}), nil
	}})
	m.Dict.SetStr("datetime", dtCls)

	// ── module constants ────────────────────────────────────────────────────────
	m.Dict.SetStr("MINYEAR", intObj64(1))
	m.Dict.SetStr("MAXYEAR", intObj64(9999))
	m.Dict.SetStr("UTC", tzUTC)

	// ── tzinfo abstract base ────────────────────────────────────────────────────
	tzinfoCls := &object.Class{Name: "tzinfo", Dict: object.NewDict()}
	m.Dict.SetStr("tzinfo", tzinfoCls)

	return m
}

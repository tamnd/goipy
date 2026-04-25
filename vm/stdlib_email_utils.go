package vm

import (
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/tamnd/goipy/object"
)

// RFC 2822 date format understood by Go's time package.
const rfc2822 = "Mon, 02 Jan 2006 15:04:05 -0700"

func (i *Interp) buildEmailUtils() *object.Module {
	m := &object.Module{Name: "email.utils", Dict: object.NewDict()}

	m.Dict.SetStr("parseaddr", &object.BuiltinFunc{Name: "parseaddr",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Tuple{V: []object.Object{&object.Str{}, &object.Str{}}}, nil
			}
			addr := object.Str_(a[0])
			name, address := parseMailAddress(addr)
			return &object.Tuple{V: []object.Object{
				&object.Str{V: name}, &object.Str{V: address},
			}}, nil
		}})

	m.Dict.SetStr("formataddr", &object.BuiltinFunc{Name: "formataddr",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{}, nil
			}
			pair, ok := a[0].(*object.Tuple)
			if !ok {
				if lst, ok2 := a[0].(*object.List); ok2 && len(lst.V) >= 2 {
					pair = &object.Tuple{V: lst.V}
				} else {
					return &object.Str{V: object.Str_(a[0])}, nil
				}
			}
			if len(pair.V) < 2 {
				return &object.Str{}, nil
			}
			name := object.Str_(pair.V[0])
			addr := object.Str_(pair.V[1])
			ma := mail.Address{Name: name, Address: addr}
			return &object.Str{V: ma.String()}, nil
		}})

	m.Dict.SetStr("getaddresses", &object.BuiltinFunc{Name: "getaddresses",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			var out []object.Object
			for _, arg := range a {
				var raw string
				switch v := arg.(type) {
				case *object.Str:
					raw = v.V
				case *object.List:
					for _, item := range v.V {
						raw += object.Str_(item) + ", "
					}
				}
				addrs, err := mail.ParseAddressList(raw)
				if err != nil {
					continue
				}
				for _, addr := range addrs {
					out = append(out, &object.Tuple{V: []object.Object{
						&object.Str{V: addr.Name}, &object.Str{V: addr.Address},
					}})
				}
			}
			return &object.List{V: out}, nil
		}})

	m.Dict.SetStr("formatdate", &object.BuiltinFunc{Name: "formatdate",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			var t time.Time
			if len(a) >= 1 && a[0] != object.None {
				if ts, ok := toFloat64(a[0]); ok {
					sec := int64(ts)
					nsec := int64((ts - float64(sec)) * 1e9)
					t = time.Unix(sec, nsec)
				}
			} else {
				t = time.Now()
			}
			localtime := false
			usegmt := false
			if len(a) >= 2 {
				if b, ok := a[1].(*object.Bool); ok {
					localtime = b.V
				}
			}
			if len(a) >= 3 {
				if b, ok := a[2].(*object.Bool); ok {
					usegmt = b.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("localtime"); ok {
					if b, ok2 := v.(*object.Bool); ok2 {
						localtime = b.V
					}
				}
				if v, ok := kw.GetStr("usegmt"); ok {
					if b, ok2 := v.(*object.Bool); ok2 {
						usegmt = b.V
					}
				}
			}
			if !localtime && !usegmt {
				t = t.UTC()
			}
			if usegmt {
				formatted := t.UTC().Format("Mon, 02 Jan 2006 15:04:05") + " GMT"
				return &object.Str{V: formatted}, nil
			}
			return &object.Str{V: t.Format(rfc2822)}, nil
		}})

	m.Dict.SetStr("format_datetime", &object.BuiltinFunc{Name: "format_datetime",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: time.Now().Format(rfc2822)}, nil
		}})

	m.Dict.SetStr("make_msgid", &object.BuiltinFunc{Name: "make_msgid",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			domain := "example.com"
			if kw != nil {
				if v, ok := kw.GetStr("domain"); ok && v != object.None {
					domain = object.Str_(v)
				}
			}
			if len(a) >= 2 && a[1] != object.None {
				domain = object.Str_(a[1])
			}
			ts := time.Now().UnixNano()
			msgid := fmt.Sprintf("<%d.%d@%s>", ts/1e9, ts%1e9, domain)
			return &object.Str{V: msgid}, nil
		}})

	parseDate := func(dateStr string) (time.Time, error) {
		layouts := []string{
			rfc2822,
			"Mon, 2 Jan 2006 15:04:05 -0700",
			"Mon, 02 Jan 2006 15:04:05 MST",
			"Mon, 2 Jan 2006 15:04:05 MST",
			"2 Jan 2006 15:04:05 -0700",
			"02 Jan 2006 15:04:05 -0700",
		}
		dateStr = strings.TrimSpace(dateStr)
		// Remove parenthetical comments like "Wed, 21 Oct 2015 07:28:00 GMT (comment)"
		if idx := strings.Index(dateStr, " ("); idx > 0 {
			dateStr = strings.TrimSpace(dateStr[:idx])
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, dateStr); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("cannot parse date: %q", dateStr)
	}

	m.Dict.SetStr("parsedate", &object.BuiltinFunc{Name: "parsedate",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			t, err := parseDate(object.Str_(a[0]))
			if err != nil {
				return object.None, nil
			}
			// Return 9-tuple: (year, month, day, hour, min, sec, weekday, julian, dst)
			wd := int(t.Weekday())
			if wd == 0 {
				wd = 6 // Python: Mon=0, Sun=6
			} else {
				wd--
			}
			return &object.Tuple{V: []object.Object{
				object.NewInt(int64(t.Year())),
				object.NewInt(int64(t.Month())),
				object.NewInt(int64(t.Day())),
				object.NewInt(int64(t.Hour())),
				object.NewInt(int64(t.Minute())),
				object.NewInt(int64(t.Second())),
				object.NewInt(int64(wd)),
				object.NewInt(int64(t.YearDay())),
				object.NewInt(-1), // DST flag unknown
			}}, nil
		}})

	m.Dict.SetStr("parsedate_tz", &object.BuiltinFunc{Name: "parsedate_tz",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			t, err := parseDate(object.Str_(a[0]))
			if err != nil {
				return object.None, nil
			}
			_, offset := t.Zone()
			wd := int(t.Weekday())
			if wd == 0 {
				wd = 6
			} else {
				wd--
			}
			return &object.Tuple{V: []object.Object{
				object.NewInt(int64(t.Year())),
				object.NewInt(int64(t.Month())),
				object.NewInt(int64(t.Day())),
				object.NewInt(int64(t.Hour())),
				object.NewInt(int64(t.Minute())),
				object.NewInt(int64(t.Second())),
				object.NewInt(int64(wd)),
				object.NewInt(int64(t.YearDay())),
				object.NewInt(-1),
				object.NewInt(int64(offset)),
			}}, nil
		}})

	m.Dict.SetStr("parsedate_to_datetime", &object.BuiltinFunc{Name: "parsedate_to_datetime",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// Returns None on parse failure (simplified)
			return object.None, nil
		}})

	m.Dict.SetStr("mktime_tz", &object.BuiltinFunc{Name: "mktime_tz",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NewInt(0), nil
			}
			tup, ok := a[0].(*object.Tuple)
			if !ok || len(tup.V) < 10 {
				return object.NewInt(0), nil
			}
			getI := func(idx int) int64 {
				n, _ := toInt64(tup.V[idx])
				return n
			}
			year, month, day := int(getI(0)), time.Month(getI(1)), int(getI(2))
			hour, min, sec := int(getI(3)), int(getI(4)), int(getI(5))
			tzOffset := int(getI(9)) // seconds east of UTC
			t := time.Date(year, month, day, hour, min, sec, 0, time.UTC)
			t = t.Add(-time.Duration(tzOffset) * time.Second)
			return object.NewInt(t.Unix()), nil
		}})

	m.Dict.SetStr("localtime", &object.BuiltinFunc{Name: "localtime",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil // stub
		}})

	m.Dict.SetStr("decode_rfc2231", &object.BuiltinFunc{Name: "decode_rfc2231",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Tuple{V: []object.Object{object.None, object.None, &object.Str{}}}, nil
			}
			s := object.Str_(a[0])
			parts := strings.SplitN(s, "'", 3)
			if len(parts) == 3 {
				return &object.Tuple{V: []object.Object{
					&object.Str{V: parts[0]}, &object.Str{V: parts[1]}, &object.Str{V: parts[2]},
				}}, nil
			}
			return &object.Tuple{V: []object.Object{object.None, object.None, &object.Str{V: s}}}, nil
		}})

	m.Dict.SetStr("encode_rfc2231", &object.BuiltinFunc{Name: "encode_rfc2231",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{}, nil
			}
			s := object.Str_(a[0])
			charset := ""
			lang := ""
			if len(a) >= 2 && a[1] != object.None {
				charset = object.Str_(a[1])
			}
			if len(a) >= 3 && a[2] != object.None {
				lang = object.Str_(a[2])
			}
			if charset != "" || lang != "" {
				return &object.Str{V: fmt.Sprintf("%s'%s'%s", charset, lang, s)}, nil
			}
			return &object.Str{V: s}, nil
		}})

	m.Dict.SetStr("collapse_rfc2231_value", &object.BuiltinFunc{Name: "collapse_rfc2231_value",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{}, nil
			}
			if t, ok := a[0].(*object.Tuple); ok && len(t.V) >= 3 {
				return t.V[2], nil
			}
			return a[0], nil
		}})

	return m
}


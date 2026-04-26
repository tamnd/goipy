package vm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tamnd/goipy/object"
)

// buildImaplib constructs the imaplib module matching CPython 3.14's API.
// No real IMAP connections are made; all network methods are stubs.
func (i *Interp) buildImaplib() *object.Module {
	m := &object.Module{Name: "imaplib", Dict: object.NewDict()}

	// ── Constants ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("IMAP4_PORT", object.NewInt(143))
	m.Dict.SetStr("IMAP4_SSL_PORT", object.NewInt(993))
	m.Dict.SetStr("Debug", object.NewInt(0))

	// ── Module-level utility functions ────────────────────────────────────────

	// Int2AP(num) — base-16 encoding using chars A-P; Int2AP(0) == b''
	ap := []byte("ABCDEFGHIJKLMNOP")
	m.Dict.SetStr("Int2AP", &object.BuiltinFunc{Name: "Int2AP", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Bytes{V: nil}, nil
		}
		n, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "Int2AP() argument must be int")
		}
		if n < 0 {
			n = -n
		}
		val := []byte{}
		for n > 0 {
			mod := n % 16
			n /= 16
			val = append([]byte{ap[mod]}, val...)
		}
		return &object.Bytes{V: val}, nil
	}})

	// ParseFlags(resp) — extract tuple of flag bytes from inside (...)
	reFlags := regexp.MustCompile(`(?s).*\((?P<flags>[^)]*)\)`)
	m.Dict.SetStr("ParseFlags", &object.BuiltinFunc{Name: "ParseFlags", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Tuple{V: nil}, nil
		}
		var s string
		switch v := a[0].(type) {
		case *object.Bytes:
			s = string(v.V)
		case *object.Str:
			s = v.V
		default:
			return &object.Tuple{V: nil}, nil
		}
		match := reFlags.FindStringSubmatch(s)
		if match == nil {
			return &object.Tuple{V: nil}, nil
		}
		content := match[1]
		fields := strings.Fields(content)
		if len(fields) == 0 {
			return &object.Tuple{V: nil}, nil
		}
		items := make([]object.Object, len(fields))
		_, isBytes := a[0].(*object.Bytes)
		for idx, f := range fields {
			if isBytes {
				items[idx] = &object.Bytes{V: []byte(f)}
			} else {
				items[idx] = &object.Str{V: f}
			}
		}
		return &object.Tuple{V: items}, nil
	}})

	// Time2Internaldate(date_time) — returns '"DD-Mmm-YYYY HH:MM:SS +HHMM"'
	m.Dict.SetStr("Time2Internaldate", &object.BuiltinFunc{Name: "Time2Internaldate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var t time.Time
		if len(a) < 1 {
			t = time.Now().UTC()
		} else {
			switch v := a[0].(type) {
			case *object.Int:
				if sec, ok := toInt64(v); ok {
					t = time.Unix(sec, 0).UTC()
				} else {
					t = time.Now().UTC()
				}
			case *object.Float:
				sec := int64(v.V)
				nsec := int64((v.V - float64(sec)) * 1e9)
				t = time.Unix(sec, nsec).UTC()
			default:
				t = time.Now().UTC()
			}
		}
		s := fmt.Sprintf(`"%02d-%s-%04d %02d:%02d:%02d +0000"`,
			t.Day(),
			[]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun",
				"Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}[t.Month()-1],
			t.Year(), t.Hour(), t.Minute(), t.Second())
		return &object.Str{V: s}, nil
	}})

	// Internaldate2tuple(resp) — parse IMAP INTERNALDATE string into tuple/None
	// Format: INTERNALDATE "DD-Mon-YYYY HH:MM:SS ±HHMM"
	reInternalDate := regexp.MustCompile(
		`INTERNALDATE "(?P<DD>[ 0-9][0-9])-(?P<Mon>[A-Za-z]{3})-(?P<YYYY>[0-9]{4})` +
			` (?P<hh>[0-9]{2}):(?P<mm>[0-9]{2}):(?P<ss>[0-9]{2})` +
			` (?P<z>[+-])(?P<zh>[0-9]{2})(?P<zm>[0-9]{2})"`)
	monthNames := map[string]int{
		"Jan": 1, "Feb": 2, "Mar": 3, "Apr": 4, "May": 5, "Jun": 6,
		"Jul": 7, "Aug": 8, "Sep": 9, "Oct": 10, "Nov": 11, "Dec": 12,
	}
	m.Dict.SetStr("Internaldate2tuple", &object.BuiltinFunc{Name: "Internaldate2tuple", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		var s string
		switch v := a[0].(type) {
		case *object.Bytes:
			s = string(v.V)
		case *object.Str:
			s = v.V
		default:
			return object.None, nil
		}
		match := reInternalDate.FindStringSubmatch(s)
		if match == nil {
			return object.None, nil
		}
		names := reInternalDate.SubexpNames()
		named := map[string]string{}
		for idx, name := range names {
			if name != "" {
				named[name] = match[idx]
			}
		}
		dd, _ := strconv.Atoi(strings.TrimSpace(named["DD"]))
		mon := monthNames[named["Mon"]]
		if mon == 0 {
			return object.None, nil
		}
		yyyy, _ := strconv.Atoi(named["YYYY"])
		hh, _ := strconv.Atoi(named["hh"])
		mm, _ := strconv.Atoi(named["mm"])
		ss, _ := strconv.Atoi(named["ss"])
		zh, _ := strconv.Atoi(named["zh"])
		zm, _ := strconv.Atoi(named["zm"])
		offSecs := zh*3600 + zm*60
		if named["z"] == "-" {
			offSecs = -offSecs
		}
		loc := time.FixedZone("", offSecs)
		t := time.Date(yyyy, time.Month(mon), dd, hh, mm, ss, 0, loc)
		lt := t.UTC()
		yday := lt.YearDay()
		wday := int(lt.Weekday())
		v := []object.Object{
			object.NewInt(int64(lt.Year())),
			object.NewInt(int64(lt.Month())),
			object.NewInt(int64(lt.Day())),
			object.NewInt(int64(lt.Hour())),
			object.NewInt(int64(lt.Minute())),
			object.NewInt(int64(lt.Second())),
			object.NewInt(int64(wday)),
			object.NewInt(int64(yday)),
			object.NewInt(-1),
		}
		return &object.Tuple{V: v}, nil
	}})

	// ── IMAP4 class ───────────────────────────────────────────────────────────

	imap4Cls := &object.Class{Name: "IMAP4", Dict: object.NewDict()}

	// Nested exception classes
	mkExc := func(name string, bases ...*object.Class) *object.Class {
		return &object.Class{Name: name, Dict: object.NewDict(), Bases: bases}
	}
	imap4ErrorCls := mkExc("error", i.exception)
	imap4AbortCls := mkExc("abort", imap4ErrorCls)
	imap4ReadonlyCls := mkExc("readonly", imap4AbortCls)

	imap4Cls.Dict.SetStr("error", imap4ErrorCls)
	imap4Cls.Dict.SetStr("abort", imap4AbortCls)
	imap4Cls.Dict.SetStr("readonly", imap4ReadonlyCls)
	imap4Cls.Dict.SetStr("PROTOCOL_VERSION", &object.Str{V: "IMAP4rev1"})

	noop := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}
	}

	okResp := func(msg string) *object.Tuple {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: []object.Object{&object.Bytes{V: []byte(msg)}}},
		}}
	}

	okRespFn := func(name, msg string) *object.BuiltinFunc {
		resp := okResp(msg)
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return resp, nil
		}}
	}

	imap4Cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		host := ""
		if len(a) > 1 {
			if s, ok2 := a[1].(*object.Str); ok2 {
				host = s.V
			}
		} else if kw != nil {
			if v, ok2 := kw.GetStr("host"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					host = s.V
				}
			}
		}
		port := object.Object(object.NewInt(143))
		if len(a) > 2 {
			port = a[2]
		}
		timeout := object.Object(object.None)
		if len(a) > 3 {
			timeout = a[3]
		}
		inst.Dict.SetStr("host", &object.Str{V: host})
		inst.Dict.SetStr("port", port)
		inst.Dict.SetStr("timeout", timeout)
		inst.Dict.SetStr("debug", object.NewInt(0))
		inst.Dict.SetStr("state", &object.Str{V: "LOGOUT"})
		inst.Dict.SetStr("literal", object.None)
		inst.Dict.SetStr("utf8_enabled", object.False)
		inst.Dict.SetStr("is_readonly", object.False)
		return object.None, nil
	}})

	imap4Cls.Dict.SetStr("set_debuglevel", &object.BuiltinFunc{Name: "set_debuglevel", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		if inst, ok := a[0].(*object.Instance); ok {
			inst.Dict.SetStr("debug", a[1])
		}
		return object.None, nil
	}})

	// stub network methods
	imap4Cls.Dict.SetStr("open", noop("open"))
	imap4Cls.Dict.SetStr("shutdown", noop("shutdown"))
	imap4Cls.Dict.SetStr("socket", &object.BuiltinFunc{Name: "socket", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	imap4Cls.Dict.SetStr("login", okRespFn("login", "LOGIN completed"))
	imap4Cls.Dict.SetStr("login_cram_md5", okRespFn("login_cram_md5", "LOGIN completed"))
	imap4Cls.Dict.SetStr("authenticate", &object.BuiltinFunc{Name: "authenticate", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: []object.Object{&object.Bytes{V: nil}}},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("logout", &object.BuiltinFunc{Name: "logout", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "BYE"},
			&object.List{V: []object.Object{&object.Bytes{V: []byte("LOGOUT")}}},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("capability", &object.BuiltinFunc{Name: "capability", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: []object.Object{&object.Bytes{V: []byte("CAPABILITY IMAP4rev1")}}},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("noop", okRespFn("noop", "NOOP completed"))
	imap4Cls.Dict.SetStr("select", &object.BuiltinFunc{Name: "select", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: []object.Object{&object.Bytes{V: []byte("0")}}},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("unselect", okRespFn("unselect", "UNSELECT completed"))
	imap4Cls.Dict.SetStr("examine", &object.BuiltinFunc{Name: "examine", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: []object.Object{&object.Bytes{V: []byte("0")}}},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("search", &object.BuiltinFunc{Name: "search", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: []object.Object{&object.Bytes{V: nil}}},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("fetch", &object.BuiltinFunc{Name: "fetch", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: []object.Object{object.None}},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("store", &object.BuiltinFunc{Name: "store", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: []object.Object{&object.Bytes{V: nil}}},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("copy", okRespFn("copy", "COPY completed"))
	imap4Cls.Dict.SetStr("move", okRespFn("move", "MOVE completed"))
	imap4Cls.Dict.SetStr("expunge", okRespFn("expunge", "EXPUNGE completed"))
	imap4Cls.Dict.SetStr("append", okRespFn("append", "APPEND completed"))
	imap4Cls.Dict.SetStr("check", okRespFn("check", "CHECK completed"))
	imap4Cls.Dict.SetStr("close", okRespFn("close", "CLOSE completed"))
	imap4Cls.Dict.SetStr("list", &object.BuiltinFunc{Name: "list", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: nil},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("lsub", &object.BuiltinFunc{Name: "lsub", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: nil},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("create", okRespFn("create", "CREATE completed"))
	imap4Cls.Dict.SetStr("delete", okRespFn("delete", "DELETE completed"))
	imap4Cls.Dict.SetStr("rename", okRespFn("rename", "RENAME completed"))
	imap4Cls.Dict.SetStr("subscribe", okRespFn("subscribe", "SUBSCRIBE completed"))
	imap4Cls.Dict.SetStr("unsubscribe", okRespFn("unsubscribe", "UNSUBSCRIBE completed"))
	imap4Cls.Dict.SetStr("status", &object.BuiltinFunc{Name: "status", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: []object.Object{&object.Bytes{V: []byte("STATUS")}}},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("sort", &object.BuiltinFunc{Name: "sort", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: []object.Object{&object.Bytes{V: nil}}},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("thread", &object.BuiltinFunc{Name: "thread", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: []object.Object{&object.Bytes{V: nil}}},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("uid", &object.BuiltinFunc{Name: "uid", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "OK"},
			&object.List{V: []object.Object{&object.Bytes{V: nil}}},
		}}, nil
	}})
	imap4Cls.Dict.SetStr("getacl", okRespFn("getacl", ""))
	imap4Cls.Dict.SetStr("setacl", okRespFn("setacl", ""))
	imap4Cls.Dict.SetStr("deleteacl", okRespFn("deleteacl", ""))
	imap4Cls.Dict.SetStr("myrights", okRespFn("myrights", ""))
	imap4Cls.Dict.SetStr("getquota", okRespFn("getquota", ""))
	imap4Cls.Dict.SetStr("setquota", okRespFn("setquota", ""))
	imap4Cls.Dict.SetStr("getquotaroot", okRespFn("getquotaroot", ""))
	imap4Cls.Dict.SetStr("namespace", okRespFn("namespace", ""))
	imap4Cls.Dict.SetStr("enable", okRespFn("enable", ""))
	imap4Cls.Dict.SetStr("xatom", okRespFn("xatom", ""))
	imap4Cls.Dict.SetStr("getannotation", okRespFn("getannotation", ""))
	imap4Cls.Dict.SetStr("setannotation", okRespFn("setannotation", ""))
	imap4Cls.Dict.SetStr("proxyauth", okRespFn("proxyauth", ""))
	imap4Cls.Dict.SetStr("starttls", okRespFn("starttls", ""))
	imap4Cls.Dict.SetStr("partial", okRespFn("partial", ""))
	imap4Cls.Dict.SetStr("recent", &object.BuiltinFunc{Name: "recent", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{&object.Str{V: "OK"}, object.None}}, nil
	}})
	imap4Cls.Dict.SetStr("response", &object.BuiltinFunc{Name: "response", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{object.None, &object.List{V: []object.Object{object.None}}}}, nil
	}})

	// __enter__ / __exit__
	imap4Cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			return a[0], nil
		}
		return object.None, nil
	}})
	imap4Cls.Dict.SetStr("__exit__", noop("__exit__"))

	m.Dict.SetStr("IMAP4", imap4Cls)

	// ── IMAP4_SSL ─────────────────────────────────────────────────────────────

	imap4SSLCls := &object.Class{Name: "IMAP4_SSL", Bases: []*object.Class{imap4Cls}, Dict: object.NewDict()}
	imap4SSLCls.Dict.SetStr("default_port", object.NewInt(993))
	m.Dict.SetStr("IMAP4_SSL", imap4SSLCls)

	// ── IMAP4_stream ──────────────────────────────────────────────────────────

	imap4StreamCls := &object.Class{Name: "IMAP4_stream", Bases: []*object.Class{imap4Cls}, Dict: object.NewDict()}
	m.Dict.SetStr("IMAP4_stream", imap4StreamCls)

	return m
}

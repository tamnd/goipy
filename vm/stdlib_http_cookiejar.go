package vm

import (
	"fmt"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildHttpCookiejar() *object.Module {
	m := &object.Module{Name: "http.cookiejar", Dict: object.NewDict()}

	// ── LoadError(OSError) ────────────────────────────────────────────────────

	loadErrCls := &object.Class{Name: "LoadError", Dict: object.NewDict(), Bases: []*object.Class{i.osErr}}
	m.Dict.SetStr("LoadError", loadErrCls)

	// ── CookiePolicy ──────────────────────────────────────────────────────────

	noop := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}
	}

	cookiePolicyCls := &object.Class{Name: "CookiePolicy", Dict: object.NewDict()}
	cookiePolicyCls.Dict.SetStr("netscape", object.False)
	cookiePolicyCls.Dict.SetStr("rfc2965", object.False)
	cookiePolicyCls.Dict.SetStr("return_ok", noop("return_ok"))
	cookiePolicyCls.Dict.SetStr("domain_return_ok", noop("domain_return_ok"))
	cookiePolicyCls.Dict.SetStr("path_return_ok", noop("path_return_ok"))
	cookiePolicyCls.Dict.SetStr("set_ok", noop("set_ok"))
	m.Dict.SetStr("CookiePolicy", cookiePolicyCls)

	// ── DefaultCookiePolicy(CookiePolicy) ─────────────────────────────────────

	defaultPolicyCls := &object.Class{Name: "DefaultCookiePolicy", Bases: []*object.Class{cookiePolicyCls}, Dict: object.NewDict()}
	defaultPolicyCls.Dict.SetStr("DomainStrictNoDots", object.NewInt(1))
	defaultPolicyCls.Dict.SetStr("DomainStrictNonDomain", object.NewInt(2))
	defaultPolicyCls.Dict.SetStr("DomainRFC2965Match", object.NewInt(4))
	defaultPolicyCls.Dict.SetStr("DomainLiberal", object.NewInt(0))
	defaultPolicyCls.Dict.SetStr("DomainStrict", object.NewInt(3))
	defaultPolicyCls.Dict.SetStr("return_ok", noop("return_ok"))
	defaultPolicyCls.Dict.SetStr("domain_return_ok", noop("domain_return_ok"))
	defaultPolicyCls.Dict.SetStr("path_return_ok", noop("path_return_ok"))
	defaultPolicyCls.Dict.SetStr("set_ok", noop("set_ok"))
	defaultPolicyCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		// defaults
		inst.Dict.SetStr("netscape", object.True)
		inst.Dict.SetStr("rfc2965", object.False)
		inst.Dict.SetStr("rfc2109_as_netscape", object.None)
		inst.Dict.SetStr("hide_cookie2", object.False)
		inst.Dict.SetStr("strict_domain", object.False)
		inst.Dict.SetStr("strict_rfc2965_unverifiable", object.True)
		inst.Dict.SetStr("strict_ns_unverifiable", object.False)
		inst.Dict.SetStr("strict_ns_domain", object.NewInt(0))
		inst.Dict.SetStr("strict_ns_set_initial_dollar", object.False)
		inst.Dict.SetStr("strict_ns_set_path", object.False)
		// apply kwargs
		if kw != nil {
			for _, key := range []string{
				"netscape", "rfc2965", "rfc2109_as_netscape", "hide_cookie2",
				"strict_domain", "strict_rfc2965_unverifiable", "strict_ns_unverifiable",
				"strict_ns_domain", "strict_ns_set_initial_dollar", "strict_ns_set_path",
			} {
				if v, ok2 := kw.GetStr(key); ok2 {
					inst.Dict.SetStr(key, v)
				}
			}
		}
		return object.None, nil
	}})
	m.Dict.SetStr("DefaultCookiePolicy", defaultPolicyCls)

	// ── Cookie ────────────────────────────────────────────────────────────────

	cookieCls := &object.Class{Name: "Cookie", Dict: object.NewDict()}
	cookieCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}

		getKw := func(key string) object.Object {
			if kw != nil {
				if v, ok2 := kw.GetStr(key); ok2 {
					return v
				}
			}
			return object.None
		}
		getKwDefault := func(key string, def object.Object) object.Object {
			if kw != nil {
				if v, ok2 := kw.GetStr(key); ok2 {
					return v
				}
			}
			return def
		}

		inst.Dict.SetStr("version", getKwDefault("version", object.NewInt(0)))
		inst.Dict.SetStr("name", getKw("name"))
		inst.Dict.SetStr("value", getKw("value"))
		inst.Dict.SetStr("port", getKw("port"))
		inst.Dict.SetStr("port_specified", getKwDefault("port_specified", object.False))
		inst.Dict.SetStr("domain", getKwDefault("domain", &object.Str{V: ""}))
		inst.Dict.SetStr("domain_specified", getKwDefault("domain_specified", object.False))
		inst.Dict.SetStr("domain_initial_dot", getKwDefault("domain_initial_dot", object.False))
		inst.Dict.SetStr("path", getKwDefault("path", &object.Str{V: "/"}))
		inst.Dict.SetStr("path_specified", getKwDefault("path_specified", object.False))
		inst.Dict.SetStr("secure", getKwDefault("secure", object.False))
		inst.Dict.SetStr("expires", getKw("expires"))
		inst.Dict.SetStr("discard", getKwDefault("discard", object.True))
		inst.Dict.SetStr("comment", getKw("comment"))
		inst.Dict.SetStr("comment_url", getKw("comment_url"))
		inst.Dict.SetStr("rfc2109", getKwDefault("rfc2109", object.False))

		// rest is a dict stored for has_nonstandard_attr
		restVal := getKwDefault("rest", object.NewDict())
		inst.Dict.SetStr("_rest", restVal)

		inst.Dict.SetStr("has_nonstandard_attr", &object.BuiltinFunc{Name: "has_nonstandard_attr", Call: func(_ any, a2 []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a2) < 1 {
				return object.False, nil
			}
			key := object.Str_(a2[0])
			switch rv := restVal.(type) {
			case *object.Dict:
				_, ok2 := rv.GetStr(key)
				return object.BoolOf(ok2), nil
			}
			return object.False, nil
		}})

		return object.None, nil
	}})

	cookieCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "Cookie()"}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "Cookie()"}, nil
		}
		get := func(key string) string {
			if v, ok2 := inst.Dict.GetStr(key); ok2 {
				return object.Repr(v)
			}
			return "None"
		}
		s := fmt.Sprintf("Cookie(version=%s, name=%s, value=%s, port=%s, port_specified=%s, "+
			"domain=%s, domain_specified=%s, domain_initial_dot=%s, path=%s, path_specified=%s, "+
			"secure=%s, expires=%s, discard=%s, comment=%s, comment_url=%s, rest=%s, rfc2109=%s)",
			get("version"), get("name"), get("value"), get("port"), get("port_specified"),
			get("domain"), get("domain_specified"), get("domain_initial_dot"),
			get("path"), get("path_specified"), get("secure"), get("expires"),
			get("discard"), get("comment"), get("comment_url"), get("_rest"), get("rfc2109"))
		return &object.Str{V: s}, nil
	}})

	cookieCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "<Cookie>"}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "<Cookie>"}, nil
		}
		name := ""
		value := ""
		domain := ""
		path := "/"
		if v, ok2 := inst.Dict.GetStr("name"); ok2 {
			name = object.Str_(v)
		}
		if v, ok2 := inst.Dict.GetStr("value"); ok2 {
			value = object.Str_(v)
		}
		if v, ok2 := inst.Dict.GetStr("domain"); ok2 {
			domain = object.Str_(v)
		}
		if v, ok2 := inst.Dict.GetStr("path"); ok2 {
			path = object.Str_(v)
		}
		s := fmt.Sprintf("<Cookie %s=%s for %s%s>", name, value, domain, path)
		return &object.Str{V: s}, nil
	}})
	m.Dict.SetStr("Cookie", cookieCls)

	// ── CookieJar ─────────────────────────────────────────────────────────────

	setupCookieJarOnInst := func(inst *object.Instance, cls *object.Class) {
		var cookies []*object.Instance

		inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(len(cookies))), nil
		}})

		inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			snap := make([]*object.Instance, len(cookies))
			copy(snap, cookies)
			idx := 0
			return &object.Iter{Next: func() (object.Object, bool, error) {
				if idx >= len(snap) {
					return nil, false, nil
				}
				v := snap[idx]
				idx++
				return v, true, nil
			}}, nil
		}})

		inst.Dict.SetStr("set_cookie", &object.BuiltinFunc{Name: "set_cookie", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			c, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			cookies = append(cookies, c)
			return object.None, nil
		}})

		inst.Dict.SetStr("clear", &object.BuiltinFunc{Name: "clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			cookies = cookies[:0]
			return object.None, nil
		}})

		inst.Dict.SetStr("clear_session_cookies", &object.BuiltinFunc{Name: "clear_session_cookies", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			kept := cookies[:0]
			for _, c := range cookies {
				discard := false
				if v, ok2 := c.Dict.GetStr("discard"); ok2 {
					if b, ok3 := v.(*object.Bool); ok3 {
						discard = b == object.True
					}
				}
				if !discard {
					kept = append(kept, c)
				}
			}
			cookies = kept
			return object.None, nil
		}})

		inst.Dict.SetStr("clear_expired_cookies", &object.BuiltinFunc{Name: "clear_expired_cookies", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

		inst.Dict.SetStr("set_policy", &object.BuiltinFunc{Name: "set_policy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				inst.Dict.SetStr("_policy", a[0])
			}
			return object.None, nil
		}})

		inst.Dict.SetStr("set_cookie_if_ok", noop("set_cookie_if_ok"))
		inst.Dict.SetStr("make_cookies", noop("make_cookies"))
		inst.Dict.SetStr("extract_cookies", noop("extract_cookies"))
		inst.Dict.SetStr("add_cookie_header", noop("add_cookie_header"))

		_ = cls
	}

	var cookieJarCls *object.Class
	cookieJarCls = &object.Class{Name: "CookieJar", Dict: object.NewDict()}
	cookieJarCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		setupCookieJarOnInst(inst, cookieJarCls)
		return object.None, nil
	}})
	m.Dict.SetStr("CookieJar", cookieJarCls)

	// ── FileCookieJar(CookieJar) ──────────────────────────────────────────────

	var fileCookieJarCls *object.Class
	fileCookieJarCls = &object.Class{Name: "FileCookieJar", Bases: []*object.Class{cookieJarCls}, Dict: object.NewDict()}
	fileCookieJarCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		setupCookieJarOnInst(inst, fileCookieJarCls)
		// filename: positional a[1] or kwarg
		var filename object.Object = object.None
		if len(a) >= 2 {
			filename = a[1]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("filename"); ok2 {
				filename = v
			}
		}
		inst.Dict.SetStr("filename", filename)
		inst.Dict.SetStr("load", noop("load"))
		inst.Dict.SetStr("save", noop("save"))
		inst.Dict.SetStr("revert", noop("revert"))
		return object.None, nil
	}})
	m.Dict.SetStr("FileCookieJar", fileCookieJarCls)

	// ── MozillaCookieJar(FileCookieJar) ───────────────────────────────────────

	mozillaCls := &object.Class{Name: "MozillaCookieJar", Bases: []*object.Class{fileCookieJarCls}, Dict: object.NewDict()}
	mozillaCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		setupCookieJarOnInst(inst, mozillaCls)
		var filename object.Object = object.None
		if len(a) >= 2 {
			filename = a[1]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("filename"); ok2 {
				filename = v
			}
		}
		inst.Dict.SetStr("filename", filename)
		inst.Dict.SetStr("load", noop("load"))
		inst.Dict.SetStr("save", noop("save"))
		inst.Dict.SetStr("revert", noop("revert"))
		return object.None, nil
	}})
	m.Dict.SetStr("MozillaCookieJar", mozillaCls)

	// ── LWPCookieJar(FileCookieJar) ───────────────────────────────────────────

	lwpCls := &object.Class{Name: "LWPCookieJar", Bases: []*object.Class{fileCookieJarCls}, Dict: object.NewDict()}
	lwpCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		setupCookieJarOnInst(inst, lwpCls)
		var filename object.Object = object.None
		if len(a) >= 2 {
			filename = a[1]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("filename"); ok2 {
				filename = v
			}
		}
		inst.Dict.SetStr("filename", filename)
		inst.Dict.SetStr("load", noop("load"))
		inst.Dict.SetStr("save", noop("save"))
		inst.Dict.SetStr("revert", noop("revert"))
		return object.None, nil
	}})
	m.Dict.SetStr("LWPCookieJar", lwpCls)

	_ = strings.TrimSpace // suppress import if unused

	return m
}

package vm

import (
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildHttpCookies() *object.Module {
	m := &object.Module{Name: "http.cookies", Dict: object.NewDict()}

	// ── Exception ─────────────────────────────────────────────────────────────

	cookieErrCls := &object.Class{Name: "CookieError", Dict: object.NewDict(), Bases: []*object.Class{i.exception}}
	m.Dict.SetStr("CookieError", cookieErrCls)

	// ── _reserved and _flags ──────────────────────────────────────────────────

	// ordered slice keeps OutputString attribute order deterministic
	reservedKeys := []struct{ k, v string }{
		{"comment", "Comment"},
		{"domain", "Domain"},
		{"expires", "expires"},
		{"httponly", "HttpOnly"},
		{"max-age", "Max-Age"},
		{"partitioned", "Partitioned"},
		{"path", "Path"},
		{"samesite", "SameSite"},
		{"secure", "Secure"},
		{"version", "Version"},
	}
	classReserved := object.NewDict()
	for _, kv := range reservedKeys {
		classReserved.SetStr(kv.k, &object.Str{V: kv.v})
	}

	classFlags := object.NewSet()
	_ = classFlags.Add(&object.Str{V: "httponly"})
	_ = classFlags.Add(&object.Str{V: "partitioned"})
	_ = classFlags.Add(&object.Str{V: "secure"})

	// ── helpers ───────────────────────────────────────────────────────────────

	isTruthy := func(o object.Object) bool {
		if o == nil || o == object.None {
			return false
		}
		if b, ok := o.(*object.Bool); ok {
			return b == object.True
		}
		if s, ok := o.(*object.Str); ok {
			return s.V != ""
		}
		return true
	}

	// setupMorselOnInst installs all Morsel state and methods on inst.
	// Returns a Go-level setter so BaseCookie can create Morsels cheaply.
	setupMorselOnInst := func(inst *object.Instance) func(key, val, coded string) {
		var mkey, mvalue, mcodedValue object.Object = object.None, object.None, object.None
		attrs := object.NewDict()
		for _, kv := range reservedKeys {
			attrs.SetStr(kv.k, &object.Str{V: ""})
		}

		inst.Dict.SetStr("_reserved", classReserved)
		inst.Dict.SetStr("_flags", classFlags)
		inst.Dict.SetStr("key", object.None)
		inst.Dict.SetStr("value", object.None)
		inst.Dict.SetStr("coded_value", object.None)

		callSet := func(key, val, coded string) {
			mkey = &object.Str{V: key}
			mvalue = &object.Str{V: val}
			mcodedValue = &object.Str{V: coded}
			inst.Dict.SetStr("key", mkey)
			inst.Dict.SetStr("value", mvalue)
			inst.Dict.SetStr("coded_value", mcodedValue)
		}

		inst.Dict.SetStr("set", &object.BuiltinFunc{Name: "set", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return object.None, nil
			}
			callSet(object.Str_(a[0]), object.Str_(a[1]), object.Str_(a[2]))
			return object.None, nil
		}})

		inst.Dict.SetStr("isReservedKey", &object.BuiltinFunc{Name: "isReservedKey", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			_, ok := classReserved.GetStr(strings.ToLower(object.Str_(a[0])))
			return object.BoolOf(ok), nil
		}})

		inst.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			key := strings.ToLower(object.Str_(a[0]))
			if _, ok := classReserved.GetStr(key); !ok {
				return nil, object.Errorf(cookieErrCls, "Invalid attribute %q", key)
			}
			attrs.SetStr(key, a[1])
			return object.None, nil
		}})

		inst.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.keyErr, "missing key")
			}
			key := strings.ToLower(object.Str_(a[0]))
			if v, ok := attrs.GetStr(key); ok {
				return v, nil
			}
			return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
		}})

		inst.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			_, ok := attrs.GetStr(strings.ToLower(object.Str_(a[0])))
			return object.BoolOf(ok), nil
		}})

		inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(attrs.Len())), nil
		}})

		inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "<Morsel: " + object.Str_(mkey) + "=" + object.Str_(mcodedValue) + ">"}, nil
		}})

		buildOutputString := func() string {
			if mkey == nil || mkey == object.None {
				return ""
			}
			parts := []string{object.Str_(mkey) + "=" + object.Str_(mcodedValue)}
			for _, kv := range reservedKeys {
				v, ok := attrs.GetStr(kv.k)
				if !ok {
					continue
				}
				isFlag, _ := classFlags.Contains(&object.Str{V: kv.k})
				if isFlag {
					if isTruthy(v) {
						parts = append(parts, kv.v)
					}
				} else if isTruthy(v) {
					parts = append(parts, kv.v+"="+object.Str_(v))
				}
			}
			return strings.Join(parts, "; ")
		}

		inst.Dict.SetStr("OutputString", &object.BuiltinFunc{Name: "OutputString", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: buildOutputString()}, nil
		}})

		inst.Dict.SetStr("output", &object.BuiltinFunc{Name: "output", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			header := "Set-Cookie:"
			if kw != nil {
				if v, ok := kw.GetStr("header"); ok {
					header = object.Str_(v)
				}
			}
			if len(a) >= 2 {
				header = object.Str_(a[1])
			}
			return &object.Str{V: header + " " + buildOutputString()}, nil
		}})

		inst.Dict.SetStr("js_output", &object.BuiltinFunc{Name: "js_output", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "\n\t<script type=\"text/javascript\">\n\t<!-- begin hiding\n\tdocument.cookie = \"" + buildOutputString() + "\";\n\t// end hiding -->\n\t</script>\n"}, nil
		}})

		return callSet
	}

	// ── Morsel class ──────────────────────────────────────────────────────────

	morsClass := &object.Class{Name: "Morsel", Dict: object.NewDict()}
	morsClass.Dict.SetStr("_reserved", classReserved)
	morsClass.Dict.SetStr("_flags", classFlags)
	morsClass.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		setupMorselOnInst(inst)
		return object.None, nil
	}})
	m.Dict.SetStr("Morsel", morsClass)

	// newMorsel creates a fresh Morsel instance and returns it with a direct Go setter.
	newMorsel := func() (*object.Instance, func(key, val, coded string)) {
		inst := &object.Instance{Class: morsClass, Dict: object.NewDict()}
		callSet := setupMorselOnInst(inst)
		return inst, callSet
	}

	// ── BaseCookie / SimpleCookie ─────────────────────────────────────────────

	var baseCookieCls, simpleCookieCls *object.Class

	setupBaseCookieOnInst := func(inst *object.Instance, cls *object.Class, isSimple bool) {
		cookies := object.NewDict() // name (*Str) -> *Morsel instance

		valueDecode := func(val string) (decoded, coded string) {
			if isSimple && len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
				return val[1 : len(val)-1], val
			}
			return val, val
		}

		makeMorsel := func(key, val string) *object.Instance {
			mi, callSet := newMorsel()
			decoded, coded := valueDecode(val)
			callSet(key, decoded, coded)
			return mi
		}

		inst.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			key := object.Str_(a[0])
			if existingMorsel, ok := a[1].(*object.Instance); ok {
				_ = cookies.Set(&object.Str{V: key}, existingMorsel)
			} else {
				_ = cookies.Set(&object.Str{V: key}, makeMorsel(key, object.Str_(a[1])))
			}
			return object.None, nil
		}})

		inst.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.keyErr, "missing key")
			}
			v, ok, err := cookies.Get(&object.Str{V: object.Str_(a[0])})
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, object.Errorf(i.keyErr, "%s", object.Repr(a[0]))
			}
			return v, nil
		}})

		inst.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			_, ok, _ := cookies.Get(&object.Str{V: object.Str_(a[0])})
			return object.BoolOf(ok), nil
		}})

		inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(cookies.Len())), nil
		}})

		inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ks, _ := cookies.Items()
			idx := 0
			return &object.Iter{Next: func() (object.Object, bool, error) {
				if idx >= len(ks) {
					return nil, false, nil
				}
				v := ks[idx]
				idx++
				return v, true, nil
			}}, nil
		}})

		inst.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ks, vs := cookies.Items()
			parts := make([]string, 0, len(ks))
			for j, k := range ks {
				parts = append(parts, object.Repr(k)+" = "+object.Repr(vs[j]))
			}
			return &object.Str{V: "<" + cls.Name + ": " + strings.Join(parts, ", ") + ">"}, nil
		}})

		inst.Dict.SetStr("load", &object.BuiltinFunc{Name: "load", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			switch rawdata := a[0].(type) {
			case *object.Str:
				httpCookieParsePairs(rawdata.V, func(key, val string) {
					_ = cookies.Set(&object.Str{V: key}, makeMorsel(key, val))
				})
			case *object.Dict:
				ks, vs := rawdata.Items()
				for j, k := range ks {
					key := object.Str_(k)
					val := object.Str_(vs[j])
					_ = cookies.Set(&object.Str{V: key}, makeMorsel(key, val))
				}
			}
			return object.None, nil
		}})

		inst.Dict.SetStr("output", &object.BuiltinFunc{Name: "output", Call: func(_ any, _ []object.Object, kw *object.Dict) (object.Object, error) {
			sep := "\r\n"
			if kw != nil {
				if v, ok := kw.GetStr("sep"); ok {
					sep = object.Str_(v)
				}
			}
			_, vs := cookies.Items()
			parts := make([]string, 0, len(vs))
			for _, v := range vs {
				if mi, ok := v.(*object.Instance); ok {
					if outFn, ok2 := mi.Dict.GetStr("output"); ok2 {
						if bf, ok3 := outFn.(*object.BuiltinFunc); ok3 {
							if res, err := bf.Call(nil, nil, nil); err == nil {
								parts = append(parts, object.Str_(res))
							}
						}
					}
				}
			}
			return &object.Str{V: strings.Join(parts, sep)}, nil
		}})

		inst.Dict.SetStr("js_output", &object.BuiltinFunc{Name: "js_output", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			_, vs := cookies.Items()
			parts := make([]string, 0, len(vs))
			for _, v := range vs {
				if mi, ok := v.(*object.Instance); ok {
					if jsFn, ok2 := mi.Dict.GetStr("js_output"); ok2 {
						if bf, ok3 := jsFn.(*object.BuiltinFunc); ok3 {
							if res, err := bf.Call(nil, nil, nil); err == nil {
								parts = append(parts, object.Str_(res))
							}
						}
					}
				}
			}
			return &object.Str{V: strings.Join(parts, "")}, nil
		}})

		inst.Dict.SetStr("value_decode", &object.BuiltinFunc{Name: "value_decode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Tuple{V: []object.Object{object.None, object.None}}, nil
			}
			decoded, coded := valueDecode(object.Str_(a[0]))
			return &object.Tuple{V: []object.Object{&object.Str{V: decoded}, &object.Str{V: coded}}}, nil
		}})

		inst.Dict.SetStr("value_encode", &object.BuiltinFunc{Name: "value_encode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			return &object.Str{V: object.Str_(a[0])}, nil
		}})

		inst.Dict.SetStr("get", &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			if v, ok, _ := cookies.Get(&object.Str{V: object.Str_(a[0])}); ok {
				return v, nil
			}
			if len(a) >= 2 {
				return a[1], nil
			}
			return object.None, nil
		}})

		inst.Dict.SetStr("keys", &object.BuiltinFunc{Name: "keys", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ks, _ := cookies.Items()
			return &object.List{V: ks}, nil
		}})

		inst.Dict.SetStr("values", &object.BuiltinFunc{Name: "values", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			_, vs := cookies.Items()
			return &object.List{V: vs}, nil
		}})
	}

	baseCookieCls = &object.Class{Name: "BaseCookie", Dict: object.NewDict()}
	baseCookieCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		setupBaseCookieOnInst(inst, baseCookieCls, false)
		return object.None, nil
	}})
	m.Dict.SetStr("BaseCookie", baseCookieCls)

	simpleCookieCls = &object.Class{Name: "SimpleCookie", Bases: []*object.Class{baseCookieCls}, Dict: object.NewDict()}
	simpleCookieCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		setupBaseCookieOnInst(inst, simpleCookieCls, true)
		return object.None, nil
	}})
	m.Dict.SetStr("SimpleCookie", simpleCookieCls)

	return m
}

// httpCookieParsePairs parses a "name=value; name2=value2" cookie string.
func httpCookieParsePairs(raw string, fn func(key, val string)) {
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.IndexByte(part, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:idx])
		val := strings.TrimSpace(part[idx+1:])
		if key == "" {
			continue
		}
		fn(key, val)
	}
}

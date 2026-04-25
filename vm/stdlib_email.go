package vm

import (
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tamnd/goipy/object"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// emailGetHeaders returns the _headers list from a Message instance.
func emailGetHeaders(inst *object.Instance) *object.List {
	v, ok := inst.Dict.GetStr("_headers")
	if !ok {
		return nil
	}
	lst, _ := v.(*object.List)
	return lst
}

// emailHeaderGet returns the first matching header value (case-insensitive).
func emailHeaderGet(inst *object.Instance, name string) (string, bool) {
	lst := emailGetHeaders(inst)
	if lst == nil {
		return "", false
	}
	lower := strings.ToLower(name)
	for _, item := range lst.V {
		t, ok := item.(*object.Tuple)
		if !ok || len(t.V) < 2 {
			continue
		}
		if strings.ToLower(t.V[0].(*object.Str).V) == lower {
			return t.V[1].(*object.Str).V, true
		}
	}
	return "", false
}

func emailHeaderGetAll(inst *object.Instance, name string) []string {
	lst := emailGetHeaders(inst)
	if lst == nil {
		return nil
	}
	lower := strings.ToLower(name)
	var out []string
	for _, item := range lst.V {
		t, ok := item.(*object.Tuple)
		if !ok || len(t.V) < 2 {
			continue
		}
		if strings.ToLower(t.V[0].(*object.Str).V) == lower {
			out = append(out, t.V[1].(*object.Str).V)
		}
	}
	return out
}

func emailHeaderSet(inst *object.Instance, name, value string) {
	lst := emailGetHeaders(inst)
	if lst == nil {
		lst = &object.List{}
		inst.Dict.SetStr("_headers", lst)
	}
	lst.V = append(lst.V, &object.Tuple{V: []object.Object{
		&object.Str{V: name}, &object.Str{V: value},
	}})
}

func emailHeaderDel(inst *object.Instance, name string) {
	lst := emailGetHeaders(inst)
	if lst == nil {
		return
	}
	lower := strings.ToLower(name)
	var kept []object.Object
	for _, item := range lst.V {
		t, ok := item.(*object.Tuple)
		if !ok || len(t.V) < 2 {
			kept = append(kept, item)
			continue
		}
		if strings.ToLower(t.V[0].(*object.Str).V) != lower {
			kept = append(kept, item)
		}
	}
	lst.V = kept
}

// emailContentType parses the Content-Type header.
func emailContentType(inst *object.Instance) (maintype, subtype string, params map[string]string) {
	ct, ok := emailHeaderGet(inst, "Content-Type")
	if !ok {
		deftype := "text/plain"
		if v, ok2 := inst.Dict.GetStr("_deftype"); ok2 {
			if s, ok3 := v.(*object.Str); ok3 {
				deftype = s.V
			}
		}
		parts := strings.SplitN(deftype, "/", 2)
		if len(parts) == 2 {
			return parts[0], parts[1], nil
		}
		return deftype, "", nil
	}
	mt, p, err := mime.ParseMediaType(ct)
	if err != nil {
		parts := strings.SplitN(strings.TrimSpace(ct), "/", 2)
		if len(parts) == 2 {
			return strings.ToLower(strings.TrimSpace(parts[0])),
				strings.ToLower(strings.TrimSpace(parts[1])), nil
		}
		return strings.ToLower(strings.TrimSpace(ct)), "", nil
	}
	parts := strings.SplitN(mt, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], p
	}
	return mt, "", p
}

// emailInitInstance initialises the per-instance state for a Message.
func emailInitInstance(inst *object.Instance) {
	inst.Dict.SetStr("_headers", &object.List{})
	inst.Dict.SetStr("_payload", object.None)
	inst.Dict.SetStr("_deftype", &object.Str{V: "text/plain"})
	inst.Dict.SetStr("preamble", object.None)
	inst.Dict.SetStr("epilogue", object.None)
	inst.Dict.SetStr("defects", &object.List{})
}

// emailAsString serialises a Message to a string.
func emailAsString(inst *object.Instance) string {
	var sb strings.Builder
	lst := emailGetHeaders(inst)
	if lst != nil {
		for _, item := range lst.V {
			t, ok := item.(*object.Tuple)
			if !ok || len(t.V) < 2 {
				continue
			}
			sb.WriteString(t.V[0].(*object.Str).V)
			sb.WriteString(": ")
			sb.WriteString(t.V[1].(*object.Str).V)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")

	pl, ok := inst.Dict.GetStr("_payload")
	if !ok || pl == nil || pl == object.None {
		return sb.String()
	}
	switch v := pl.(type) {
	case *object.Str:
		sb.WriteString(v.V)
	case *object.Bytes:
		sb.WriteString(string(v.V))
	case *object.List:
		_, _, params := emailContentType(inst)
		boundary := ""
		if params != nil {
			boundary = params["boundary"]
		}
		if boundary == "" {
			boundary = fmt.Sprintf("===============%d==", time.Now().UnixNano()%1e12)
		}
		if pre, ok2 := inst.Dict.GetStr("preamble"); ok2 && pre != object.None {
			if ps, ok3 := pre.(*object.Str); ok3 && ps.V != "" {
				sb.WriteString(ps.V)
				sb.WriteString("\n")
			}
		}
		for _, part := range v.V {
			sb.WriteString("--")
			sb.WriteString(boundary)
			sb.WriteString("\n")
			if partInst, ok2 := part.(*object.Instance); ok2 {
				sb.WriteString(emailAsString(partInst))
			}
		}
		sb.WriteString("--")
		sb.WriteString(boundary)
		sb.WriteString("--\n")
		if epi, ok2 := inst.Dict.GetStr("epilogue"); ok2 && epi != object.None {
			if es, ok3 := epi.(*object.Str); ok3 && es.V != "" {
				sb.WriteString(es.V)
			}
		}
	}
	return sb.String()
}

// dictIterStr iterates a *object.Dict calling fn(key, value) for string keys.
func dictIterStr(d *object.Dict, fn func(key string, val object.Object)) {
	if d == nil {
		return
	}
	ks, vs := d.Items()
	for j, k := range ks {
		if s, ok := k.(*object.Str); ok {
			fn(s.V, vs[j])
		}
	}
}

// ── Message class ─────────────────────────────────────────────────────────────

func (i *Interp) buildMessageClass() *object.Class {
	cls := &object.Class{Name: "Message", Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			emailInitInstance(a[0].(*object.Instance))
			return object.None, nil
		}})

	cls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			name, _ := a[1].(*object.Str)
			if name == nil {
				return object.None, nil
			}
			if v, ok := emailHeaderGet(self, name.V); ok {
				return &object.Str{V: v}, nil
			}
			return object.None, nil
		}})

	cls.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			name, _ := a[1].(*object.Str)
			if name == nil {
				return object.None, nil
			}
			emailHeaderSet(self, name.V, object.Str_(a[2]))
			return object.None, nil
		}})

	cls.Dict.SetStr("__delitem__", &object.BuiltinFunc{Name: "__delitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			if name, ok := a[1].(*object.Str); ok {
				emailHeaderDel(self, name.V)
			}
			return object.None, nil
		}})

	cls.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.False, nil
			}
			self := a[0].(*object.Instance)
			name, _ := a[1].(*object.Str)
			if name == nil {
				return object.False, nil
			}
			_, ok := emailHeaderGet(self, name.V)
			return object.BoolOf(ok), nil
		}})

	cls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NewInt(0), nil
			}
			self := a[0].(*object.Instance)
			lst := emailGetHeaders(self)
			if lst == nil {
				return object.NewInt(0), nil
			}
			return object.NewInt(int64(len(lst.V))), nil
		}})

	cls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{}, nil
			}
			return &object.Str{V: emailAsString(a[0].(*object.Instance))}, nil
		}})

	cls.Dict.SetStr("keys", &object.BuiltinFunc{Name: "keys",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{}, nil
			}
			lst := emailGetHeaders(a[0].(*object.Instance))
			var out []object.Object
			if lst != nil {
				for _, item := range lst.V {
					if t, ok := item.(*object.Tuple); ok && len(t.V) >= 1 {
						out = append(out, t.V[0])
					}
				}
			}
			return &object.List{V: out}, nil
		}})

	cls.Dict.SetStr("values", &object.BuiltinFunc{Name: "values",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{}, nil
			}
			lst := emailGetHeaders(a[0].(*object.Instance))
			var out []object.Object
			if lst != nil {
				for _, item := range lst.V {
					if t, ok := item.(*object.Tuple); ok && len(t.V) >= 2 {
						out = append(out, t.V[1])
					}
				}
			}
			return &object.List{V: out}, nil
		}})

	cls.Dict.SetStr("items", &object.BuiltinFunc{Name: "items",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{}, nil
			}
			lst := emailGetHeaders(a[0].(*object.Instance))
			var out []object.Object
			if lst != nil {
				for _, item := range lst.V {
					if t, ok := item.(*object.Tuple); ok && len(t.V) >= 2 {
						out = append(out, &object.Tuple{V: []object.Object{t.V[0], t.V[1]}})
					}
				}
			}
			return &object.List{V: out}, nil
		}})

	cls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			name, _ := a[1].(*object.Str)
			if name == nil {
				return object.None, nil
			}
			failobj := object.Object(object.None)
			if len(a) >= 3 {
				failobj = a[2]
			}
			if kw != nil {
				if v, ok := kw.GetStr("failobj"); ok {
					failobj = v
				}
			}
			if v, ok := emailHeaderGet(self, name.V); ok {
				return &object.Str{V: v}, nil
			}
			return failobj, nil
		}})

	cls.Dict.SetStr("get_all", &object.BuiltinFunc{Name: "get_all",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			name, _ := a[1].(*object.Str)
			if name == nil {
				return object.None, nil
			}
			failobj := object.Object(object.None)
			if len(a) >= 3 {
				failobj = a[2]
			}
			vals := emailHeaderGetAll(self, name.V)
			if vals == nil {
				return failobj, nil
			}
			out := make([]object.Object, len(vals))
			for j, v := range vals {
				out[j] = &object.Str{V: v}
			}
			return &object.List{V: out}, nil
		}})

	cls.Dict.SetStr("add_header", &object.BuiltinFunc{Name: "add_header",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			name := object.Str_(a[1])
			value := object.Str_(a[2])
			if kw != nil {
				var params []string
				dictIterStr(kw, func(k string, v object.Object) {
					k2 := strings.ReplaceAll(k, "_", "-")
					if v == object.None {
						return
					}
					params = append(params, fmt.Sprintf("%s=%q", k2, object.Str_(v)))
				})
				if len(params) > 0 {
					value += "; " + strings.Join(params, "; ")
				}
			}
			emailHeaderSet(self, name, value)
			return object.None, nil
		}})

	cls.Dict.SetStr("replace_header", &object.BuiltinFunc{Name: "replace_header",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, object.Errorf(i.keyErr, "_name not found")
			}
			self := a[0].(*object.Instance)
			name, _ := a[1].(*object.Str)
			if name == nil {
				return nil, object.Errorf(i.keyErr, "header name required")
			}
			newval := object.Str_(a[2])
			lst := emailGetHeaders(self)
			if lst == nil {
				return nil, object.Errorf(i.keyErr, "%s", name.V)
			}
			lower := strings.ToLower(name.V)
			for j, item := range lst.V {
				t, ok := item.(*object.Tuple)
				if !ok || len(t.V) < 2 {
					continue
				}
				if strings.ToLower(t.V[0].(*object.Str).V) == lower {
					lst.V[j] = &object.Tuple{V: []object.Object{t.V[0], &object.Str{V: newval}}}
					return object.None, nil
				}
			}
			return nil, object.Errorf(i.keyErr, "%s", name.V)
		}})

	cls.Dict.SetStr("get_content_type", &object.BuiltinFunc{Name: "get_content_type",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "text/plain"}, nil
			}
			main, sub, _ := emailContentType(a[0].(*object.Instance))
			return &object.Str{V: main + "/" + sub}, nil
		}})

	cls.Dict.SetStr("get_content_maintype", &object.BuiltinFunc{Name: "get_content_maintype",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "text"}, nil
			}
			main, _, _ := emailContentType(a[0].(*object.Instance))
			return &object.Str{V: main}, nil
		}})

	cls.Dict.SetStr("get_content_subtype", &object.BuiltinFunc{Name: "get_content_subtype",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "plain"}, nil
			}
			_, sub, _ := emailContentType(a[0].(*object.Instance))
			return &object.Str{V: sub}, nil
		}})

	cls.Dict.SetStr("get_default_type", &object.BuiltinFunc{Name: "get_default_type",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "text/plain"}, nil
			}
			self := a[0].(*object.Instance)
			if v, ok := self.Dict.GetStr("_deftype"); ok {
				return v, nil
			}
			return &object.Str{V: "text/plain"}, nil
		}})

	cls.Dict.SetStr("set_default_type", &object.BuiltinFunc{Name: "set_default_type",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			a[0].(*object.Instance).Dict.SetStr("_deftype", a[1])
			return object.None, nil
		}})

	cls.Dict.SetStr("get_params", &object.BuiltinFunc{Name: "get_params",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			main, sub, params := emailContentType(self)
			if params == nil && main == "" {
				return object.None, nil
			}
			var out []object.Object
			out = append(out, &object.Tuple{V: []object.Object{
				&object.Str{V: main + "/" + sub}, &object.Str{V: ""},
			}})
			for k, v := range params {
				out = append(out, &object.Tuple{V: []object.Object{
					&object.Str{V: k}, &object.Str{V: v},
				}})
			}
			return &object.List{V: out}, nil
		}})

	cls.Dict.SetStr("get_param", &object.BuiltinFunc{Name: "get_param",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			param, _ := a[1].(*object.Str)
			if param == nil {
				return object.None, nil
			}
			failobj := object.Object(object.None)
			if len(a) >= 3 {
				failobj = a[2]
			}
			headerName := "content-type"
			if kw != nil {
				if v, ok := kw.GetStr("header"); ok {
					if s, ok2 := v.(*object.Str); ok2 {
						headerName = s.V
					}
				}
			}
			ct, ok := emailHeaderGet(self, headerName)
			if !ok {
				return failobj, nil
			}
			_, params, err := mime.ParseMediaType(ct)
			if err != nil {
				return failobj, nil
			}
			if v, ok2 := params[param.V]; ok2 {
				return &object.Str{V: v}, nil
			}
			return failobj, nil
		}})

	cls.Dict.SetStr("get_filename", &object.BuiltinFunc{Name: "get_filename",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			failobj := object.Object(object.None)
			if len(a) >= 2 {
				failobj = a[1]
			}
			cd, ok := emailHeaderGet(self, "Content-Disposition")
			if !ok {
				return failobj, nil
			}
			_, params, err := mime.ParseMediaType(cd)
			if err != nil {
				return failobj, nil
			}
			if fn, ok2 := params["filename"]; ok2 {
				return &object.Str{V: fn}, nil
			}
			return failobj, nil
		}})

	cls.Dict.SetStr("get_boundary", &object.BuiltinFunc{Name: "get_boundary",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			failobj := object.Object(object.None)
			if len(a) >= 2 {
				failobj = a[1]
			}
			_, _, params := emailContentType(self)
			if params == nil {
				return failobj, nil
			}
			if b, ok := params["boundary"]; ok {
				return &object.Str{V: b}, nil
			}
			return failobj, nil
		}})

	cls.Dict.SetStr("set_boundary", &object.BuiltinFunc{Name: "set_boundary",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			boundary, _ := a[1].(*object.Str)
			if boundary == nil {
				return object.None, nil
			}
			ct, ok := emailHeaderGet(self, "Content-Type")
			if !ok {
				return object.None, nil
			}
			mt, params, err := mime.ParseMediaType(ct)
			if err != nil {
				return object.None, nil
			}
			if params == nil {
				params = map[string]string{}
			}
			params["boundary"] = boundary.V
			emailHeaderDel(self, "Content-Type")
			emailHeaderSet(self, "Content-Type", mime.FormatMediaType(mt, params))
			return object.None, nil
		}})

	cls.Dict.SetStr("get_content_charset", &object.BuiltinFunc{Name: "get_content_charset",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			failobj := object.Object(object.None)
			if len(a) >= 2 {
				failobj = a[1]
			}
			_, _, params := emailContentType(self)
			if params == nil {
				return failobj, nil
			}
			if cs, ok := params["charset"]; ok {
				return &object.Str{V: strings.ToLower(cs)}, nil
			}
			return failobj, nil
		}})

	cls.Dict.SetStr("get_charset", &object.BuiltinFunc{Name: "get_charset",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	cls.Dict.SetStr("set_charset", &object.BuiltinFunc{Name: "set_charset",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	cls.Dict.SetStr("get_charsets", &object.BuiltinFunc{Name: "get_charsets",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{V: []object.Object{object.None}}, nil
			}
			self := a[0].(*object.Instance)
			_, _, params := emailContentType(self)
			if params != nil {
				if cs, ok := params["charset"]; ok {
					return &object.List{V: []object.Object{&object.Str{V: strings.ToLower(cs)}}}, nil
				}
			}
			return &object.List{V: []object.Object{object.None}}, nil
		}})

	cls.Dict.SetStr("get_payload", &object.BuiltinFunc{Name: "get_payload",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			decode := false
			idx := -1
			if len(a) >= 2 && a[1] != object.None {
				if n, ok := toInt64(a[1]); ok {
					idx = int(n)
				}
			}
			if len(a) >= 3 {
				if b, ok := a[2].(*object.Bool); ok {
					decode = b.V
				} else if n, ok2 := toInt64(a[2]); ok2 {
					decode = n != 0
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("i"); ok && v != object.None {
					if n, ok2 := toInt64(v); ok2 {
						idx = int(n)
					}
				}
				if v, ok := kw.GetStr("decode"); ok {
					if b, ok2 := v.(*object.Bool); ok2 {
						decode = b.V
					} else if n, ok2 := toInt64(v); ok2 {
						decode = n != 0
					}
				}
			}
			pl, ok := self.Dict.GetStr("_payload")
			if !ok {
				return object.None, nil
			}
			if lst, ok2 := pl.(*object.List); ok2 {
				if idx >= 0 {
					if idx < len(lst.V) {
						return lst.V[idx], nil
					}
					return nil, object.Errorf(i.indexErr, "index out of bounds")
				}
				return lst, nil
			}
			if !decode {
				return pl, nil
			}
			// decode=True: apply Content-Transfer-Encoding
			cte, _ := emailHeaderGet(self, "Content-Transfer-Encoding")
			cte = strings.ToLower(strings.TrimSpace(cte))
			var raw []byte
			switch v := pl.(type) {
			case *object.Str:
				raw = []byte(v.V)
			case *object.Bytes:
				raw = v.V
			default:
				return pl, nil
			}
			switch cte {
			case "base64":
				dec, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
				if err != nil {
					dec, err = base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(raw)))
					if err != nil {
						return &object.Bytes{V: raw}, nil
					}
				}
				return &object.Bytes{V: dec}, nil
			case "quoted-printable":
				r := quotedprintable.NewReader(strings.NewReader(string(raw)))
				var buf []byte
				tmp := make([]byte, 4096)
				for {
					n, err := r.Read(tmp)
					if n > 0 {
						buf = append(buf, tmp[:n]...)
					}
					if err != nil {
						break
					}
				}
				return &object.Bytes{V: buf}, nil
			default:
				return &object.Bytes{V: raw}, nil
			}
		}})

	cls.Dict.SetStr("set_payload", &object.BuiltinFunc{Name: "set_payload",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			a[0].(*object.Instance).Dict.SetStr("_payload", a[1])
			return object.None, nil
		}})

	cls.Dict.SetStr("is_multipart", &object.BuiltinFunc{Name: "is_multipart",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			self := a[0].(*object.Instance)
			pl, ok := self.Dict.GetStr("_payload")
			if !ok {
				return object.False, nil
			}
			_, ok2 := pl.(*object.List)
			return object.BoolOf(ok2), nil
		}})

	cls.Dict.SetStr("attach", &object.BuiltinFunc{Name: "attach",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			pl, ok := self.Dict.GetStr("_payload")
			var lst *object.List
			if ok {
				lst, _ = pl.(*object.List)
			}
			if lst == nil {
				lst = &object.List{}
				self.Dict.SetStr("_payload", lst)
			}
			lst.V = append(lst.V, a[1])
			return object.None, nil
		}})

	cls.Dict.SetStr("walk", &object.BuiltinFunc{Name: "walk",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{}, nil
			}
			self := a[0].(*object.Instance)
			var parts []object.Object
			var collect func(*object.Instance)
			collect = func(inst *object.Instance) {
				parts = append(parts, inst)
				pl, ok := inst.Dict.GetStr("_payload")
				if !ok {
					return
				}
				lst, ok2 := pl.(*object.List)
				if !ok2 {
					return
				}
				for _, p := range lst.V {
					if child, ok3 := p.(*object.Instance); ok3 {
						collect(child)
					}
				}
			}
			collect(self)
			return &object.List{V: parts}, nil
		}})

	cls.Dict.SetStr("as_string", &object.BuiltinFunc{Name: "as_string",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{}, nil
			}
			return &object.Str{V: emailAsString(a[0].(*object.Instance))}, nil
		}})

	cls.Dict.SetStr("as_bytes", &object.BuiltinFunc{Name: "as_bytes",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Bytes{}, nil
			}
			return &object.Bytes{V: []byte(emailAsString(a[0].(*object.Instance)))}, nil
		}})

	cls.Dict.SetStr("get_unixfrom", &object.BuiltinFunc{Name: "get_unixfrom",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			if v, ok := a[0].(*object.Instance).Dict.GetStr("_unixfrom"); ok {
				return v, nil
			}
			return object.None, nil
		}})

	cls.Dict.SetStr("set_unixfrom", &object.BuiltinFunc{Name: "set_unixfrom",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			a[0].(*object.Instance).Dict.SetStr("_unixfrom", a[1])
			return object.None, nil
		}})

	cls.Dict.SetStr("get_content_disposition", &object.BuiltinFunc{Name: "get_content_disposition",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			cd, ok := emailHeaderGet(a[0].(*object.Instance), "Content-Disposition")
			if !ok {
				return object.None, nil
			}
			mt, _, _ := mime.ParseMediaType(cd)
			if mt == "" {
				return object.None, nil
			}
			return &object.Str{V: mt}, nil
		}})

	cls.Dict.SetStr("is_attachment", &object.BuiltinFunc{Name: "is_attachment",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			cd, ok := emailHeaderGet(a[0].(*object.Instance), "Content-Disposition")
			if !ok {
				return object.False, nil
			}
			return object.BoolOf(strings.Contains(strings.ToLower(cd), "attachment")), nil
		}})

	return cls
}

// ── email.message_from_string / Parser ────────────────────────────────────────

func (i *Interp) emailParseString(raw string, msgCls *object.Class) (*object.Instance, error) {
	inst := &object.Instance{Class: msgCls, Dict: object.NewDict()}
	emailInitInstance(inst)

	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")

	// Parse headers
	idx := 0
	for idx < len(lines) {
		line := lines[idx]
		if line == "" {
			idx++
			break
		}
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			// Continuation line
			lst := emailGetHeaders(inst)
			if lst != nil && len(lst.V) > 0 {
				last := lst.V[len(lst.V)-1].(*object.Tuple)
				newval := last.V[1].(*object.Str).V + " " + strings.TrimSpace(line)
				lst.V[len(lst.V)-1] = &object.Tuple{V: []object.Object{
					last.V[0], &object.Str{V: newval},
				}}
			}
			idx++
			continue
		}
		colonIdx := strings.IndexByte(line, ':')
		if colonIdx < 0 {
			idx++
			continue
		}
		name := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		for idx+1 < len(lines) {
			next := lines[idx+1]
			if len(next) > 0 && (next[0] == ' ' || next[0] == '\t') {
				value += " " + strings.TrimSpace(next)
				idx++
			} else {
				break
			}
		}
		emailHeaderSet(inst, name, value)
		idx++
	}

	body := strings.Join(lines[idx:], "\n")

	// Check for multipart
	ct, hasCT := emailHeaderGet(inst, "Content-Type")
	if hasCT {
		mt, params, err := mime.ParseMediaType(ct)
		if err == nil && strings.HasPrefix(mt, "multipart/") {
			boundary := params["boundary"]
			if boundary != "" {
				r := multipart.NewReader(strings.NewReader(body), boundary)
				var partList []object.Object
				for {
					part, err2 := r.NextPart()
					if err2 != nil {
						break
					}
					var sb strings.Builder
					for hname, hvals := range part.Header {
						for _, hval := range hvals {
							sb.WriteString(hname)
							sb.WriteString(": ")
							sb.WriteString(hval)
							sb.WriteString("\n")
						}
					}
					sb.WriteString("\n")
					buf := make([]byte, 65536)
					for {
						n, rerr := part.Read(buf)
						if n > 0 {
							sb.Write(buf[:n])
						}
						if rerr != nil {
							break
						}
					}
					child, err3 := i.emailParseString(sb.String(), msgCls)
					if err3 == nil {
						partList = append(partList, child)
					}
				}
				inst.Dict.SetStr("_payload", &object.List{V: partList})
				return inst, nil
			}
		}
	}

	inst.Dict.SetStr("_payload", &object.Str{V: body})
	return inst, nil
}

// ── shared Message class (singleton) ─────────────────────────────────────────

var (
	emailMsgClsOnce sync.Once
	emailMsgCls     *object.Class
)

func (i *Interp) getEmailMessageClass() *object.Class {
	emailMsgClsOnce.Do(func() {
		emailMsgCls = i.buildMessageClass()
	})
	return emailMsgCls
}

// ── email module ──────────────────────────────────────────────────────────────

func (i *Interp) buildEmail() *object.Module {
	m := &object.Module{Name: "email", Dict: object.NewDict()}
	msgCls := i.getEmailMessageClass()

	parse := func(raw string, a []object.Object, kw *object.Dict) (object.Object, error) {
		cls := msgCls
		if kw != nil {
			if v, ok := kw.GetStr("_class"); ok {
				if c, ok2 := v.(*object.Class); ok2 {
					cls = c
				}
			}
		}
		return i.emailParseString(raw, cls)
	}

	m.Dict.SetStr("message_from_string", &object.BuiltinFunc{Name: "message_from_string",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "message_from_string() requires a string")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "message_from_string() requires str")
			}
			return parse(s.V, a, kw)
		}})

	m.Dict.SetStr("message_from_bytes", &object.BuiltinFunc{Name: "message_from_bytes",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "message_from_bytes() requires bytes")
			}
			var raw string
			switch v := a[0].(type) {
			case *object.Bytes:
				raw = string(v.V)
			case *object.Str:
				raw = v.V
			default:
				return nil, object.Errorf(i.typeErr, "message_from_bytes() requires bytes or str")
			}
			return parse(raw, a, kw)
		}})

	m.Dict.SetStr("message_from_file", &object.BuiltinFunc{Name: "message_from_file",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "message_from_file() requires a file")
			}
			var raw string
			switch v := a[0].(type) {
			case *object.StringIO:
				raw = string(v.V)
			case *object.File:
				osf := v.F.(*os.File)
				var buf []byte
				tmp := make([]byte, 4096)
				for {
					n, err := osf.Read(tmp)
					if n > 0 {
						buf = append(buf, tmp[:n]...)
					}
					if err != nil {
						break
					}
				}
				raw = string(buf)
			default:
				return nil, object.Errorf(i.typeErr, "message_from_file(): unsupported file type")
			}
			return parse(raw, a, kw)
		}})

	return m
}

// ── email.message submodule ───────────────────────────────────────────────────

func (i *Interp) buildEmailMessage() *object.Module {
	m := &object.Module{Name: "email.message", Dict: object.NewDict()}
	msgCls := i.getEmailMessageClass()
	m.Dict.SetStr("Message", msgCls)
	m.Dict.SetStr("EmailMessage", msgCls)
	m.Dict.SetStr("MIMEPart", msgCls)
	return m
}

// ── email.errors submodule ────────────────────────────────────────────────────

func (i *Interp) buildEmailErrors() *object.Module {
	m := &object.Module{Name: "email.errors", Dict: object.NewDict()}

	msgError := &object.Class{Name: "MessageError", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	msgParseError := &object.Class{Name: "MessageParseError", Bases: []*object.Class{msgError}, Dict: object.NewDict()}
	headerParseError := &object.Class{Name: "HeaderParseError", Bases: []*object.Class{msgParseError}, Dict: object.NewDict()}
	headerWriteError := &object.Class{Name: "HeaderWriteError", Bases: []*object.Class{msgError}, Dict: object.NewDict()}
	multipartConvErr := &object.Class{Name: "MultipartConversionError", Bases: []*object.Class{msgError, i.typeErr}, Dict: object.NewDict()}
	msgDefect := &object.Class{Name: "MessageDefect", Bases: []*object.Class{i.valueErr}, Dict: object.NewDict()}
	noBoundary := &object.Class{Name: "NoBoundaryInMultipartDefect", Bases: []*object.Class{msgDefect}, Dict: object.NewDict()}
	startBoundary := &object.Class{Name: "StartBoundaryNotFoundDefect", Bases: []*object.Class{msgDefect}, Dict: object.NewDict()}
	firstHeaderLine := &object.Class{Name: "FirstHeaderLineIsContinuationDefect", Bases: []*object.Class{msgDefect}, Dict: object.NewDict()}
	misplacedEnvelope := &object.Class{Name: "MisplacedEnvelopeHeaderDefect", Bases: []*object.Class{msgDefect}, Dict: object.NewDict()}
	multipartViolation := &object.Class{Name: "MultipartInvariantViolationDefect", Bases: []*object.Class{msgDefect}, Dict: object.NewDict()}
	invalidBase64 := &object.Class{Name: "InvalidBase64PaddingDefect", Bases: []*object.Class{msgDefect}, Dict: object.NewDict()}
	invalidDate := &object.Class{Name: "InvalidDateDefect", Bases: []*object.Class{msgDefect}, Dict: object.NewDict()}

	m.Dict.SetStr("MessageError", msgError)
	m.Dict.SetStr("MessageParseError", msgParseError)
	m.Dict.SetStr("HeaderParseError", headerParseError)
	m.Dict.SetStr("HeaderWriteError", headerWriteError)
	m.Dict.SetStr("MultipartConversionError", multipartConvErr)
	m.Dict.SetStr("MessageDefect", msgDefect)
	m.Dict.SetStr("NoBoundaryInMultipartDefect", noBoundary)
	m.Dict.SetStr("StartBoundaryNotFoundDefect", startBoundary)
	m.Dict.SetStr("FirstHeaderLineIsContinuationDefect", firstHeaderLine)
	m.Dict.SetStr("MisplacedEnvelopeHeaderDefect", misplacedEnvelope)
	m.Dict.SetStr("MultipartInvariantViolationDefect", multipartViolation)
	m.Dict.SetStr("InvalidBase64PaddingDefect", invalidBase64)
	m.Dict.SetStr("InvalidDateDefect", invalidDate)
	return m
}

// ── email.policy submodule ────────────────────────────────────────────────────

func (i *Interp) buildEmailPolicy() *object.Module {
	m := &object.Module{Name: "email.policy", Dict: object.NewDict()}

	emailPolicyCls := &object.Class{Name: "EmailPolicy", Dict: object.NewDict()}
	emailPolicyCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			self.Dict.SetStr("linesep", &object.Str{V: "\n"})
			self.Dict.SetStr("max_line_length", object.NewInt(78))
			self.Dict.SetStr("utf8", object.False)
			self.Dict.SetStr("raise_on_defect", object.False)
			if kw != nil {
				dictIterStr(kw, func(k string, v object.Object) {
					self.Dict.SetStr(k, v)
				})
			}
			return object.None, nil
		}})

	cloneFn := &object.BuiltinFunc{Name: "clone",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			newInst := &object.Instance{Class: emailPolicyCls, Dict: object.NewDict()}
			if len(a) >= 1 {
				if self, ok := a[0].(*object.Instance); ok {
					ks, vs := self.Dict.Items()
					for j, k := range ks {
						if s, ok2 := k.(*object.Str); ok2 {
							newInst.Dict.SetStr(s.V, vs[j])
						}
					}
				}
			}
			if kw != nil {
				dictIterStr(kw, func(k string, v object.Object) {
					newInst.Dict.SetStr(k, v)
				})
			}
			return newInst, nil
		}}
	emailPolicyCls.Dict.SetStr("clone", cloneFn)

	compat32Cls := &object.Class{Name: "Compat32", Dict: object.NewDict()}

	buildPolicy := func(linesep string, maxLen int64, utf8 bool) *object.Instance {
		inst := &object.Instance{Class: emailPolicyCls, Dict: object.NewDict()}
		inst.Dict.SetStr("linesep", &object.Str{V: linesep})
		if maxLen >= 0 {
			inst.Dict.SetStr("max_line_length", object.NewInt(maxLen))
		} else {
			inst.Dict.SetStr("max_line_length", object.None)
		}
		inst.Dict.SetStr("utf8", object.BoolOf(utf8))
		inst.Dict.SetStr("raise_on_defect", object.False)
		inst.Dict.SetStr("clone", cloneFn)
		return inst
	}

	compat32Inst := &object.Instance{Class: compat32Cls, Dict: object.NewDict()}
	compat32Inst.Dict.SetStr("linesep", &object.Str{V: "\n"})
	compat32Inst.Dict.SetStr("max_line_length", object.NewInt(78))
	compat32Inst.Dict.SetStr("mangle_from_", object.True)
	compat32Inst.Dict.SetStr("raise_on_defect", object.False)

	m.Dict.SetStr("EmailPolicy", emailPolicyCls)
	m.Dict.SetStr("Compat32", compat32Cls)
	m.Dict.SetStr("compat32", compat32Inst)
	m.Dict.SetStr("default", buildPolicy("\n", 78, false))
	m.Dict.SetStr("SMTP", buildPolicy("\r\n", 78, false))
	m.Dict.SetStr("SMTPUTF8", buildPolicy("\r\n", 78, true))
	m.Dict.SetStr("HTTP", buildPolicy("\r\n", -1, false))
	m.Dict.SetStr("strict", buildPolicy("\n", 78, false))
	return m
}

// ── email.charset submodule ───────────────────────────────────────────────────

func (i *Interp) buildEmailCharset() *object.Module {
	m := &object.Module{Name: "email.charset", Dict: object.NewDict()}

	charsetCls := &object.Class{Name: "Charset", Dict: object.NewDict()}
	charsetCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			charset := "us-ascii"
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					charset = s.V
				}
			}
			self.Dict.SetStr("input_charset", &object.Str{V: charset})
			self.Dict.SetStr("output_charset", &object.Str{V: charset})
			self.Dict.SetStr("header_encoding", object.None)
			self.Dict.SetStr("body_encoding", object.None)
			return object.None, nil
		}})
	charsetCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "us-ascii"}, nil
			}
			if v, ok := a[0].(*object.Instance).Dict.GetStr("input_charset"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					return &object.Str{V: strings.ToLower(s.V)}, nil
				}
			}
			return &object.Str{V: "us-ascii"}, nil
		}})
	charsetCls.Dict.SetStr("get_body_encoding", &object.BuiltinFunc{Name: "get_body_encoding",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "7bit"}, nil
		}})
	charsetCls.Dict.SetStr("get_output_charset", &object.BuiltinFunc{Name: "get_output_charset",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				if v, ok := a[0].(*object.Instance).Dict.GetStr("output_charset"); ok {
					return v, nil
				}
			}
			return &object.Str{V: "us-ascii"}, nil
		}})
	charsetCls.Dict.SetStr("header_encode", &object.BuiltinFunc{Name: "header_encode",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 2 {
				return a[1], nil
			}
			return &object.Str{}, nil
		}})
	charsetCls.Dict.SetStr("body_encode", &object.BuiltinFunc{Name: "body_encode",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 2 {
				return a[1], nil
			}
			return &object.Str{}, nil
		}})

	m.Dict.SetStr("Charset", charsetCls)
	m.Dict.SetStr("QP", object.NewInt(1))
	m.Dict.SetStr("BASE64", object.NewInt(2))
	m.Dict.SetStr("SHORTEST", object.NewInt(3))
	m.Dict.SetStr("DEFAULT_CHARSET", &object.Str{V: "us-ascii"})
	m.Dict.SetStr("add_charset", &object.BuiltinFunc{Name: "add_charset",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) { return object.None, nil }})
	m.Dict.SetStr("add_alias", &object.BuiltinFunc{Name: "add_alias",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) { return object.None, nil }})
	m.Dict.SetStr("add_codec", &object.BuiltinFunc{Name: "add_codec",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) { return object.None, nil }})
	return m
}

// ── net/mail helper ───────────────────────────────────────────────────────────

// parseMailAddress parses an RFC 5322 address into (name, addr).
func parseMailAddress(addr string) (name, address string) {
	a, err := mail.ParseAddress(addr)
	if err != nil {
		addr = strings.TrimSpace(addr)
		if strings.HasPrefix(addr, "<") && strings.HasSuffix(addr, ">") {
			return "", addr[1 : len(addr)-1]
		}
		return "", addr
	}
	return a.Name, a.Address
}

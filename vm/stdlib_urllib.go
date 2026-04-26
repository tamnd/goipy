package vm

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/tamnd/goipy/object"
)

// ---- urllib.error ----------------------------------------------------------

func (i *Interp) buildUrllibError() *object.Module {
	m := &object.Module{Name: "urllib.error", Dict: object.NewDict()}
	m.Dict.SetStr("__package__", &object.Str{V: "urllib"})

	urlErrCls := &object.Class{Name: "URLError", Dict: object.NewDict(), Bases: []*object.Class{i.osErr}}
	// URLError(reason) — reason is Args[0]
	urlErrCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		name := object.Str_(a[1])
		switch name {
		case "reason":
			if exc.Args != nil && len(exc.Args.V) > 0 {
				return exc.Args.V[0], nil
			}
			return &object.Str{V: ""}, nil
		}
		return nil, object.Errorf(i.attrErr, "'URLError' object has no attribute '%s'", name)
	}})
	urlErrCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		reason := ""
		if exc.Args != nil && len(exc.Args.V) > 0 {
			reason = object.Str_(exc.Args.V[0])
		}
		return &object.Str{V: fmt.Sprintf("<urlopen error %s>", reason)}, nil
	}})

	// HTTPError(url, code, msg, hdrs, fp)
	httpErrCls := &object.Class{Name: "HTTPError", Dict: object.NewDict(), Bases: []*object.Class{urlErrCls}}
	httpErrCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		name := object.Str_(a[1])
		args := exc.Args
		switch name {
		case "url", "filename":
			if args != nil && len(args.V) > 0 {
				return args.V[0], nil
			}
		case "code":
			if args != nil && len(args.V) > 1 {
				return args.V[1], nil
			}
		case "msg", "reason":
			if args != nil && len(args.V) > 2 {
				return args.V[2], nil
			}
		case "headers", "hdrs":
			if args != nil && len(args.V) > 3 {
				return args.V[3], nil
			}
		case "fp":
			if args != nil && len(args.V) > 4 {
				return args.V[4], nil
			}
		}
		return nil, object.Errorf(i.attrErr, "'HTTPError' object has no attribute '%s'", name)
	}})
	httpErrCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		code := ""
		msg := ""
		if exc.Args != nil {
			if len(exc.Args.V) > 1 {
				code = object.Str_(exc.Args.V[1])
			}
			if len(exc.Args.V) > 2 {
				msg = object.Str_(exc.Args.V[2])
			}
		}
		return &object.Str{V: fmt.Sprintf("HTTP Error %s: %s", code, msg)}, nil
	}})
	httpErrCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		code := ""
		var msgObj object.Object = &object.Str{}
		if exc.Args != nil {
			if len(exc.Args.V) > 1 {
				code = object.Str_(exc.Args.V[1])
			}
			if len(exc.Args.V) > 2 {
				msgObj = exc.Args.V[2]
			}
		}
		return &object.Str{V: fmt.Sprintf("<HTTPError %s: %s>", code, object.Repr(msgObj))}, nil
	}})
	httpErrFP := func(exc *object.Exception) object.Object {
		if exc.Args != nil && len(exc.Args.V) > 4 {
			return exc.Args.V[4]
		}
		return object.None
	}
	httpErrCls.Dict.SetStr("read", &object.BuiltinFunc{Name: "read", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		fp := httpErrFP(exc)
		if fp == object.None {
			return &object.Bytes{V: []byte{}}, nil
		}
		interp := ii.(*Interp)
		readFn, err := interp.getAttr(fp, "read")
		if err != nil {
			return nil, err
		}
		return interp.callObject(readFn, a[1:], nil)
	}})
	httpErrCls.Dict.SetStr("getcode", &object.BuiltinFunc{Name: "getcode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		if exc.Args != nil && len(exc.Args.V) > 1 {
			return exc.Args.V[1], nil
		}
		return object.None, nil
	}})
	httpErrCls.Dict.SetStr("geturl", &object.BuiltinFunc{Name: "geturl", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		if exc.Args != nil && len(exc.Args.V) > 0 {
			return exc.Args.V[0], nil
		}
		return object.None, nil
	}})
	httpErrCls.Dict.SetStr("info", &object.BuiltinFunc{Name: "info", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		if exc.Args != nil && len(exc.Args.V) > 3 {
			return exc.Args.V[3], nil
		}
		return object.None, nil
	}})
	httpErrCls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		fp := httpErrFP(exc)
		if fp == object.None {
			return object.None, nil
		}
		interp := ii.(*Interp)
		closeFn, err := interp.getAttr(fp, "close")
		if err != nil {
			return object.None, nil
		}
		interp.callObject(closeFn, nil, nil) //nolint
		return object.None, nil
	}})
	httpErrCls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return a[0], nil
	}})
	httpErrCls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		fp := httpErrFP(exc)
		if fp != object.None {
			interp := ii.(*Interp)
			if closeFn, err := interp.getAttr(fp, "close"); err == nil {
				interp.callObject(closeFn, nil, nil) //nolint
			}
		}
		return object.None, nil
	}})

	// ContentTooShortError(msg, content) — msg=Args[0], content=Args[1]
	contentTooShortCls := &object.Class{Name: "ContentTooShortError", Dict: object.NewDict(), Bases: []*object.Class{urlErrCls}}
	contentTooShortCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		name := object.Str_(a[1])
		switch name {
		case "content":
			if exc.Args != nil && len(exc.Args.V) > 1 {
				return exc.Args.V[1], nil
			}
			return object.None, nil
		case "reason":
			if exc.Args != nil && len(exc.Args.V) > 0 {
				return exc.Args.V[0], nil
			}
			return &object.Str{V: ""}, nil
		}
		return nil, object.Errorf(i.attrErr, "'ContentTooShortError' object has no attribute '%s'", name)
	}})

	m.Dict.SetStr("URLError", urlErrCls)
	m.Dict.SetStr("HTTPError", httpErrCls)
	m.Dict.SetStr("ContentTooShortError", contentTooShortCls)
	return m
}

// ---- urllib.request --------------------------------------------------------

// reqState holds per-Request Go state not easily stored in the instance Dict.
type reqState struct {
	headers            map[string]string // capitalized
	unredirectedHdrs   map[string]string // sent on first request, dropped on redirect
}

// openerState holds per-OpenerDirector handler chains.
type openerState struct {
	mu             sync.Mutex
	openHandlers   map[string][]object.Object // scheme -> []handler with scheme_open
	errorHandlers  map[string][]object.Object // "NNN" -> []handler
	reqHandlers    map[string][]object.Object // scheme -> []handler with scheme_request
	respHandlers   map[string][]object.Object // scheme -> []handler with scheme_response
}

// passwdEntry is one add_password record.
type passwdEntry struct {
	uri    string
	user   string
	passwd string
}

// passwdMgrState holds HTTPPasswordMgr data.
type passwdMgrState struct {
	mu      sync.Mutex
	entries map[string][]passwdEntry // realm -> []passwdEntry; "" = default realm
}

var (
	urllibReqMap      sync.Map // *object.Instance -> *reqState
	urllibOpenerState sync.Map // *object.Instance -> *openerState
	urllibPasswdState sync.Map // *object.Instance -> *passwdMgrState
	urllibGlobalOpener object.Object
	urllibTempFiles    sync.Map // string -> bool  (files to delete on urlcleanup)
)

func (i *Interp) buildUrllibRequest() *object.Module {
	m := &object.Module{Name: "urllib.request", Dict: object.NewDict()}
	m.Dict.SetStr("__package__", &object.Str{V: "urllib"})

	urllibErrMod, _ := i.loadModule("urllib.error")
	getErrCls := func(name string) *object.Class {
		if urllibErrMod != nil {
			if v, ok := urllibErrMod.Dict.GetStr(name); ok {
				if cls, ok := v.(*object.Class); ok {
					return cls
				}
			}
		}
		return i.exception
	}

	// ---- helpers shared within this builder ----
	reqHeaders := func(inst *object.Instance) map[string]string {
		if st, ok := urllibReqMap.Load(inst); ok {
			return st.(*reqState).headers
		}
		return nil
	}
	reqUnredirHdrs := func(inst *object.Instance) map[string]string {
		if st, ok := urllibReqMap.Load(inst); ok {
			return st.(*reqState).unredirectedHdrs
		}
		return nil
	}

	// ---- addinfourl response class ----
	respCls := &object.Class{Name: "addinfourl", Dict: object.NewDict()}
	buildAddinfourl := func(body []byte, statusCode int, headerMap map[string]string, urlStr string) *object.Instance {
		inst := &object.Instance{Class: respCls, Dict: object.NewDict()}
		inst.Dict.SetStr("_body", &object.Bytes{V: body})
		inst.Dict.SetStr("_pos", &object.Int{})
		inst.Dict.SetStr("url", &object.Str{V: urlStr})
		inst.Dict.SetStr("status", &object.Int{V: *new(big.Int).SetInt64(int64(statusCode))})
		inst.Dict.SetStr("code", &object.Int{V: *new(big.Int).SetInt64(int64(statusCode))})
		hDict := object.NewDict()
		for k, v := range headerMap {
			hDict.Set(&object.Str{V: k}, &object.Str{V: v})
		}
		inst.Dict.SetStr("headers", hDict)
		return inst
	}

	addRespMethods(respCls)

	// ---- Request class ----
	reqCls := &object.Class{Name: "Request", Dict: object.NewDict()}
	reqCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "Request() requires url")
		}
		inst := a[0].(*object.Instance)
		urlStr := ""
		if s, ok := a[1].(*object.Str); ok {
			urlStr = s.V
		}
		reqSetURL(inst, urlStr)

		// data (positional arg 2 or kwarg)
		var dataVal object.Object = object.None
		if len(a) > 2 {
			dataVal = a[2]
		} else if kw != nil {
			if v, ok := kw.GetStr("data"); ok {
				dataVal = v
			}
		}
		inst.Dict.SetStr("data", dataVal)

		// method (positional arg 5 or kwarg)
		method := ""
		if len(a) > 5 {
			if s, ok := a[5].(*object.Str); ok {
				method = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("method"); ok {
				if s, ok := v.(*object.Str); ok {
					method = s.V
				}
			}
		}
		inst.Dict.SetStr("_method", &object.Str{V: method})

		// unverifiable (arg 4 or kwarg)
		unverifiable := false
		if len(a) > 4 {
			if b, ok := a[4].(*object.Bool); ok {
				unverifiable = b.V
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("unverifiable"); ok {
				if b, ok := v.(*object.Bool); ok {
					unverifiable = b.V
				}
			}
		}
		inst.Dict.SetStr("unverifiable", object.BoolOf(unverifiable))

		// headers dict (arg 3 or kwarg)
		headers := make(map[string]string)
		hdrsObj := object.Object(nil)
		if len(a) > 3 {
			hdrsObj = a[3]
		} else if kw != nil {
			if v, ok := kw.GetStr("headers"); ok {
				hdrsObj = v
			}
		}
		if d, ok := hdrsObj.(*object.Dict); ok {
			ks, vs := d.Items()
			for idx, k := range ks {
				if ks2, ok := k.(*object.Str); ok {
					if vs2, ok := vs[idx].(*object.Str); ok {
						headers[capitalizeHeader(ks2.V)] = vs2.V
					}
				}
			}
		}
		urllibReqMap.Store(inst, &reqState{
			headers:          headers,
			unredirectedHdrs: make(map[string]string),
		})
		return object.None, nil
	}})

	reqCls.Dict.SetStr("get_method", &object.BuiltinFunc{Name: "get_method", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "GET"}, nil
		}
		inst := a[0].(*object.Instance)
		if m, ok := inst.Dict.GetStr("_method"); ok {
			if s, ok := m.(*object.Str); ok && s.V != "" {
				return s, nil
			}
		}
		if data, ok := inst.Dict.GetStr("data"); ok && data != object.None {
			if _, isNone := data.(*object.NoneType); !isNone {
				return &object.Str{V: "POST"}, nil
			}
		}
		return &object.Str{V: "GET"}, nil
	}})

	reqCls.Dict.SetStr("get_full_url", &object.BuiltinFunc{Name: "get_full_url", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		inst := a[0].(*object.Instance)
		if v, ok := inst.Dict.GetStr("full_url"); ok {
			return v, nil
		}
		return &object.Str{V: ""}, nil
	}})

	reqCls.Dict.SetStr("set_proxy", &object.BuiltinFunc{Name: "set_proxy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// set_proxy(host, type) — override routing host and scheme
		if len(a) < 3 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		if h, ok := a[1].(*object.Str); ok {
			inst.Dict.SetStr("host", h)
		}
		if t, ok := a[2].(*object.Str); ok {
			inst.Dict.SetStr("type", t)
		}
		return object.None, nil
	}})

	reqCls.Dict.SetStr("add_header", &object.BuiltinFunc{Name: "add_header", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "add_header() requires key and val")
		}
		inst := a[0].(*object.Instance)
		k, _ := a[1].(*object.Str)
		v, _ := a[2].(*object.Str)
		if k == nil || v == nil {
			return nil, object.Errorf(i.typeErr, "add_header() arguments must be str")
		}
		if h := reqHeaders(inst); h != nil {
			h[capitalizeHeader(k.V)] = v.V
		}
		return object.None, nil
	}})

	reqCls.Dict.SetStr("has_header", &object.BuiltinFunc{Name: "has_header", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		inst := a[0].(*object.Instance)
		k, _ := a[1].(*object.Str)
		if k == nil {
			return object.False, nil
		}
		key := capitalizeHeader(k.V)
		if h := reqHeaders(inst); h != nil {
			if _, ok := h[key]; ok {
				return object.True, nil
			}
		}
		if u := reqUnredirHdrs(inst); u != nil {
			if _, ok := u[key]; ok {
				return object.True, nil
			}
		}
		return object.False, nil
	}})

	reqCls.Dict.SetStr("get_header", &object.BuiltinFunc{Name: "get_header", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		k, _ := a[1].(*object.Str)
		if k == nil {
			return object.None, nil
		}
		var dflt object.Object = object.None
		if len(a) > 2 {
			dflt = a[2]
		}
		key := capitalizeHeader(k.V)
		if h := reqHeaders(inst); h != nil {
			if v, ok := h[key]; ok {
				return &object.Str{V: v}, nil
			}
		}
		if u := reqUnredirHdrs(inst); u != nil {
			if v, ok := u[key]; ok {
				return &object.Str{V: v}, nil
			}
		}
		return dflt, nil
	}})

	reqCls.Dict.SetStr("remove_header", &object.BuiltinFunc{Name: "remove_header", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		k, _ := a[1].(*object.Str)
		if k == nil {
			return object.None, nil
		}
		key := capitalizeHeader(k.V)
		if h := reqHeaders(inst); h != nil {
			delete(h, key)
		}
		return object.None, nil
	}})

	reqCls.Dict.SetStr("add_unredirected_header", &object.BuiltinFunc{Name: "add_unredirected_header", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "add_unredirected_header() requires key and val")
		}
		inst := a[0].(*object.Instance)
		k, _ := a[1].(*object.Str)
		v, _ := a[2].(*object.Str)
		if k == nil || v == nil {
			return nil, object.Errorf(i.typeErr, "add_unredirected_header() arguments must be str")
		}
		if u := reqUnredirHdrs(inst); u != nil {
			u[capitalizeHeader(k.V)] = v.V
		}
		return object.None, nil
	}})

	reqCls.Dict.SetStr("header_items", &object.BuiltinFunc{Name: "header_items", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{}, nil
		}
		inst := a[0].(*object.Instance)
		out := &object.List{}
		seen := make(map[string]bool)
		if h := reqHeaders(inst); h != nil {
			for k, v := range h {
				out.V = append(out.V, &object.Tuple{V: []object.Object{&object.Str{V: k}, &object.Str{V: v}}})
				seen[k] = true
			}
		}
		if u := reqUnredirHdrs(inst); u != nil {
			for k, v := range u {
				if !seen[k] {
					out.V = append(out.V, &object.Tuple{V: []object.Object{&object.Str{V: k}, &object.Str{V: v}}})
				}
			}
		}
		return out, nil
	}})

	m.Dict.SetStr("Request", reqCls)

	// ---- HTTPPasswordMgr ----
	passwdMgrCls := &object.Class{Name: "HTTPPasswordMgr", Dict: object.NewDict()}
	passwdMgrCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		urllibPasswdState.Store(inst, &passwdMgrState{entries: make(map[string][]passwdEntry)})
		return object.None, nil
	}})
	passwdMgrCls.Dict.SetStr("add_password", &object.BuiltinFunc{Name: "add_password", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// add_password(realm, uri, user, passwd)
		if len(a) < 5 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		realm := ""
		if a[1] != object.None {
			realm = object.Str_(a[1])
		}
		uri := object.Str_(a[2])
		user := object.Str_(a[3])
		passwd := object.Str_(a[4])
		stateVal, _ := urllibPasswdState.Load(inst)
		if stateVal == nil {
			return object.None, nil
		}
		st := stateVal.(*passwdMgrState)
		st.mu.Lock()
		defer st.mu.Unlock()
		st.entries[realm] = append(st.entries[realm], passwdEntry{uri: uri, user: user, passwd: passwd})
		return object.None, nil
	}})
	passwdMgrCls.Dict.SetStr("find_user_password", &object.BuiltinFunc{Name: "find_user_password", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// find_user_password(realm, authuri) -> (user, passwd) or (None, None)
		if len(a) < 3 {
			return &object.Tuple{V: []object.Object{object.None, object.None}}, nil
		}
		inst := a[0].(*object.Instance)
		realm := ""
		if a[1] != object.None {
			realm = object.Str_(a[1])
		}
		authURI := object.Str_(a[2])
		stateVal, _ := urllibPasswdState.Load(inst)
		if stateVal == nil {
			return &object.Tuple{V: []object.Object{object.None, object.None}}, nil
		}
		st := stateVal.(*passwdMgrState)
		st.mu.Lock()
		defer st.mu.Unlock()
		u, p := passwdFind(st, realm, authURI)
		if u == "" {
			return &object.Tuple{V: []object.Object{object.None, object.None}}, nil
		}
		return &object.Tuple{V: []object.Object{&object.Str{V: u}, &object.Str{V: p}}}, nil
	}})

	// HTTPPasswordMgrWithDefaultRealm — same but falls back to realm=""
	pwMgrDefaultCls := &object.Class{Name: "HTTPPasswordMgrWithDefaultRealm", Dict: object.NewDict(), Bases: []*object.Class{passwdMgrCls}}
	pwMgrDefaultCls.Dict.SetStr("find_user_password", &object.BuiltinFunc{Name: "find_user_password", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return &object.Tuple{V: []object.Object{object.None, object.None}}, nil
		}
		inst := a[0].(*object.Instance)
		realm := ""
		if a[1] != object.None {
			realm = object.Str_(a[1])
		}
		authURI := object.Str_(a[2])
		stateVal, _ := urllibPasswdState.Load(inst)
		if stateVal == nil {
			return &object.Tuple{V: []object.Object{object.None, object.None}}, nil
		}
		st := stateVal.(*passwdMgrState)
		st.mu.Lock()
		defer st.mu.Unlock()
		// try exact realm first
		u, p := passwdFind(st, realm, authURI)
		if u == "" {
			// fall back to default realm
			u, p = passwdFind(st, "", authURI)
		}
		if u == "" {
			return &object.Tuple{V: []object.Object{object.None, object.None}}, nil
		}
		return &object.Tuple{V: []object.Object{&object.Str{V: u}, &object.Str{V: p}}}, nil
	}})

	// HTTPPasswordMgrWithPriorAuth
	pwMgrPriorCls := &object.Class{Name: "HTTPPasswordMgrWithPriorAuth", Dict: object.NewDict(), Bases: []*object.Class{pwMgrDefaultCls}}
	pwMgrPriorCls.Dict.SetStr("update_authenticated", &object.BuiltinFunc{Name: "update_authenticated", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// update_authenticated(uri, is_authenticated)
		if len(a) < 3 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		uri := object.Str_(a[1])
		flag := false
		if b, ok := a[2].(*object.Bool); ok {
			flag = b.V
		}
		key := "_auth_" + uri
		inst.Dict.SetStr(key, object.BoolOf(flag))
		return object.None, nil
	}})
	pwMgrPriorCls.Dict.SetStr("is_authenticated", &object.BuiltinFunc{Name: "is_authenticated", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		inst := a[0].(*object.Instance)
		uri := object.Str_(a[1])
		key := "_auth_" + uri
		if v, ok := inst.Dict.GetStr(key); ok {
			if b, ok := v.(*object.Bool); ok && b.V {
				return object.True, nil
			}
		}
		return object.False, nil
	}})

	m.Dict.SetStr("HTTPPasswordMgr", passwdMgrCls)
	m.Dict.SetStr("HTTPPasswordMgrWithDefaultRealm", pwMgrDefaultCls)
	m.Dict.SetStr("HTTPPasswordMgrWithPriorAuth", pwMgrPriorCls)

	// ---- BaseHandler ----
	baseHandlerCls := &object.Class{Name: "BaseHandler", Dict: object.NewDict()}
	baseHandlerCls.Dict.SetStr("handler_order", &object.Int{V: *new(big.Int).SetInt64(500)})
	baseHandlerCls.Dict.SetStr("add_parent", &object.BuiltinFunc{Name: "add_parent", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		a[0].(*object.Instance).Dict.SetStr("parent", a[1])
		return object.None, nil
	}})
	baseHandlerCls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	m.Dict.SetStr("BaseHandler", baseHandlerCls)

	// ---- HTTPDefaultErrorHandler ----
	httpErrHandlerCls := &object.Class{Name: "HTTPDefaultErrorHandler", Dict: object.NewDict(), Bases: []*object.Class{baseHandlerCls}}
	httpErrHandlerCls.Dict.SetStr("http_error_default", &object.BuiltinFunc{Name: "http_error_default", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// http_error_default(self, req, fp, code, msg, hdrs) -> raise HTTPError
		if len(a) < 6 {
			return nil, object.Errorf(getErrCls("URLError"), "HTTP error")
		}
		code := int64(0)
		if n, ok := a[3].(*object.Int); ok {
			code = n.V.Int64()
		}
		msg := object.Str_(a[4])
		urlStr := ""
		if inst, ok := a[1].(*object.Instance); ok {
			if fu, ok2 := inst.Dict.GetStr("full_url"); ok2 {
				urlStr = object.Str_(fu)
			}
		}
		return nil, object.Errorf(getErrCls("HTTPError"), "HTTP Error %d: %s (url=%s)", code, msg, urlStr)
	}})
	m.Dict.SetStr("HTTPDefaultErrorHandler", httpErrHandlerCls)

	// ---- HTTPErrorProcessor ----
	httpErrProcCls := &object.Class{Name: "HTTPErrorProcessor", Dict: object.NewDict(), Bases: []*object.Class{baseHandlerCls}}
	httpErrProcCls.Dict.SetStr("http_response", &object.BuiltinFunc{Name: "http_response", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// http_response(self, req, response) — pass through 2xx; else call opener.error
		if len(a) < 3 {
			return object.None, nil
		}
		resp := a[2]
		code := int64(200)
		if inst, ok := resp.(*object.Instance); ok {
			if cv, ok2 := inst.Dict.GetStr("code"); ok2 {
				if n, ok3 := cv.(*object.Int); ok3 {
					code = n.V.Int64()
				}
			}
		}
		if code >= 200 && code < 300 {
			return resp, nil
		}
		// call opener.error — look up parent
		self := a[0].(*object.Instance)
		if parentObj, ok := self.Dict.GetStr("parent"); ok {
			if opener, ok2 := parentObj.(*object.Instance); ok2 {
				if errFn, ok3 := opener.Dict.GetStr("error"); ok3 {
					return ii.(*Interp).callObject(errFn, []object.Object{opener, &object.Str{V: "http"}, a[1], resp, &object.Int{V: *new(big.Int).SetInt64(code)}, &object.Str{V: "error"}, object.NewDict()}, nil)
				}
			}
		}
		return resp, nil
	}})
	httpErrProcCls.Dict.SetStr("https_response", &object.BuiltinFunc{Name: "https_response", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if fn, ok := httpErrProcCls.Dict.GetStr("http_response"); ok {
			return ii.(*Interp).callObject(fn, a, kw)
		}
		return object.None, nil
	}})
	m.Dict.SetStr("HTTPErrorProcessor", httpErrProcCls)

	// ---- HTTPRedirectHandler ----
	httpRedirCls := &object.Class{Name: "HTTPRedirectHandler", Dict: object.NewDict(), Bases: []*object.Class{baseHandlerCls}}
	httpRedirCls.Dict.SetStr("max_repeats", &object.Int{V: *new(big.Int).SetInt64(4)})
	httpRedirCls.Dict.SetStr("max_redirections", &object.Int{V: *new(big.Int).SetInt64(10)})
	httpRedirCls.Dict.SetStr("redirect_request", &object.BuiltinFunc{Name: "redirect_request", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// redirect_request(self, req, fp, code, msg, headers, newurl) -> Request
		if len(a) < 7 {
			return object.None, nil
		}
		newURL := object.Str_(a[6])
		newInst := &object.Instance{Class: reqCls, Dict: object.NewDict()}
		reqSetURL(newInst, newURL)
		newInst.Dict.SetStr("data", object.None)
		newInst.Dict.SetStr("_method", &object.Str{V: ""})
		newInst.Dict.SetStr("unverifiable", object.True)
		urllibReqMap.Store(newInst, &reqState{
			headers:          make(map[string]string),
			unredirectedHdrs: make(map[string]string),
		})
		return newInst, nil
	}})
	for _, code := range []string{"301", "302", "303", "307", "308"} {
		c := code
		httpRedirCls.Dict.SetStr("http_error_"+c, &object.BuiltinFunc{Name: "http_error_" + c, Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// http_error_NNN(self, req, fp, code, msg, hdrs) -> new response or None
			if len(a) < 6 {
				return object.None, nil
			}
			hdrsDict, _ := a[5].(*object.Dict)
			newURL := ""
			if hdrsDict != nil {
				if v, ok := hdrsDict.GetStr("Location"); ok {
					newURL = object.Str_(v)
				} else if v, ok := hdrsDict.GetStr("location"); ok {
					newURL = object.Str_(v)
				}
			}
			if newURL == "" {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			if redirectFn, ok := httpRedirCls.Dict.GetStr("redirect_request"); ok {
				newReq, err := ii.(*Interp).callObject(redirectFn, append([]object.Object{self}, append(a[1:], &object.Str{V: newURL})...), nil)
				if err != nil {
					return nil, err
				}
				if newReq != object.None {
					// ask parent opener to open new request
					if parentObj, ok2 := self.Dict.GetStr("parent"); ok2 {
						if opener, ok3 := parentObj.(*object.Instance); ok3 {
							if openFn, ok4 := opener.Dict.GetStr("open"); ok4 {
								return ii.(*Interp).callObject(openFn, []object.Object{opener, newReq}, nil)
							}
						}
					}
					return newReq, nil
				}
			}
			return object.None, nil
		}})
	}
	m.Dict.SetStr("HTTPRedirectHandler", httpRedirCls)

	// ---- HTTPCookieProcessor ----
	httpCookieCls := &object.Class{Name: "HTTPCookieProcessor", Dict: object.NewDict(), Bases: []*object.Class{baseHandlerCls}}
	httpCookieCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		var jar object.Object = object.None
		if len(a) > 1 && a[1] != object.None {
			jar = a[1]
		} else if kw != nil {
			if v, ok := kw.GetStr("cookiejar"); ok && v != object.None {
				jar = v
			}
		}
		inst.Dict.SetStr("cookiejar", jar)
		return object.None, nil
	}})
	httpCookieCls.Dict.SetStr("http_request", &object.BuiltinFunc{Name: "http_request", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		return a[1], nil // pass request through unchanged
	}})
	httpCookieCls.Dict.SetStr("http_response", &object.BuiltinFunc{Name: "http_response", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		return a[2], nil // pass response through unchanged
	}})
	m.Dict.SetStr("HTTPCookieProcessor", httpCookieCls)

	// ---- ProxyHandler ----
	proxyHandlerCls := &object.Class{Name: "ProxyHandler", Dict: object.NewDict(), Bases: []*object.Class{baseHandlerCls}}
	proxyHandlerCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		proxiesDict := object.NewDict()
		if len(a) > 1 && a[1] != object.None {
			if d, ok := a[1].(*object.Dict); ok {
				proxiesDict = d
			}
		}
		inst.Dict.SetStr("proxies", proxiesDict)
		return object.None, nil
	}})
	proxyHandlerCls.Dict.SetStr("set_proxy", &object.BuiltinFunc{Name: "set_proxy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	m.Dict.SetStr("ProxyHandler", proxyHandlerCls)

	// ---- AbstractBasicAuthHandler ----
	abBasicAuthCls := &object.Class{Name: "AbstractBasicAuthHandler", Dict: object.NewDict(), Bases: []*object.Class{baseHandlerCls}}
	abBasicAuthCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		var mgr object.Object = object.None
		if len(a) > 1 {
			mgr = a[1]
		} else if kw != nil {
			if v, ok := kw.GetStr("password_mgr"); ok {
				mgr = v
			}
		}
		inst.Dict.SetStr("passwd", mgr)
		return object.None, nil
	}})
	abBasicAuthCls.Dict.SetStr("http_error_auth_reqed", &object.BuiltinFunc{Name: "http_error_auth_reqed", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// http_error_auth_reqed(self, authreq, host, req, headers) -> response or None
		if len(a) < 5 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		authreqHdr := object.Str_(a[1]) // "WWW-Authenticate" or "Proxy-Authenticate"
		host := object.Str_(a[2])
		req := a[3]
		headers, _ := a[4].(*object.Dict)

		// extract realm from WWW-Authenticate: Basic realm="..."
		realm := ""
		if headers != nil {
			if v, ok := headers.GetStr(authreqHdr); ok {
				realm = extractBasicRealm(object.Str_(v))
			}
		}

		// find credentials
		passwdMgrObj, _ := self.Dict.GetStr("passwd")
		user, passwd := "", ""
		if mgrInst, ok := passwdMgrObj.(*object.Instance); ok {
			if fpFn, ok2 := mgrInst.Class.Dict.GetStr("find_user_password"); ok2 {
				result, err := ii.(*Interp).callObject(fpFn, []object.Object{mgrInst, &object.Str{V: realm}, &object.Str{V: host}}, nil)
				if err == nil {
					if tup, ok3 := result.(*object.Tuple); ok3 && len(tup.V) >= 2 {
						if s, ok4 := tup.V[0].(*object.Str); ok4 {
							user = s.V
						}
						if s, ok4 := tup.V[1].(*object.Str); ok4 {
							passwd = s.V
						}
					}
				}
			}
		}

		if user == "" {
			return object.None, nil
		}

		// build Authorization header
		creds := base64.StdEncoding.EncodeToString([]byte(user + ":" + passwd))
		if reqInst, ok := req.(*object.Instance); ok {
			if h := reqHeaders(reqInst); h != nil {
				h["Authorization"] = "Basic " + creds
			}
			// re-open via parent
			if parentObj, ok2 := self.Dict.GetStr("parent"); ok2 {
				if opener, ok3 := parentObj.(*object.Instance); ok3 {
					if openFn, ok4 := opener.Dict.GetStr("open"); ok4 {
						return ii.(*Interp).callObject(openFn, []object.Object{opener, req}, nil)
					}
				}
			}
		}
		return object.None, nil
	}})
	m.Dict.SetStr("AbstractBasicAuthHandler", abBasicAuthCls)

	// ---- HTTPBasicAuthHandler ----
	httpBasicAuthCls := &object.Class{Name: "HTTPBasicAuthHandler", Dict: object.NewDict(), Bases: []*object.Class{abBasicAuthCls}}
	httpBasicAuthCls.Dict.SetStr("http_error_401", &object.BuiltinFunc{Name: "http_error_401", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// http_error_401(self, req, fp, code, msg, headers)
		if len(a) < 6 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if fn, ok := abBasicAuthCls.Dict.GetStr("http_error_auth_reqed"); ok {
			return ii.(*Interp).callObject(fn, []object.Object{self, &object.Str{V: "www-authenticate"}, a[1], a[1], a[5]}, nil)
		}
		return object.None, nil
	}})
	m.Dict.SetStr("HTTPBasicAuthHandler", httpBasicAuthCls)

	// ---- ProxyBasicAuthHandler ----
	proxyBasicAuthCls := &object.Class{Name: "ProxyBasicAuthHandler", Dict: object.NewDict(), Bases: []*object.Class{abBasicAuthCls}}
	proxyBasicAuthCls.Dict.SetStr("http_error_407", &object.BuiltinFunc{Name: "http_error_407", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 6 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if fn, ok := abBasicAuthCls.Dict.GetStr("http_error_auth_reqed"); ok {
			return ii.(*Interp).callObject(fn, []object.Object{self, &object.Str{V: "proxy-authenticate"}, a[1], a[1], a[5]}, nil)
		}
		return object.None, nil
	}})
	m.Dict.SetStr("ProxyBasicAuthHandler", proxyBasicAuthCls)

	// ---- AbstractDigestAuthHandler ----
	abDigestCls := &object.Class{Name: "AbstractDigestAuthHandler", Dict: object.NewDict(), Bases: []*object.Class{baseHandlerCls}}
	abDigestCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		var mgr object.Object = object.None
		if len(a) > 1 {
			mgr = a[1]
		}
		inst.Dict.SetStr("passwd", mgr)
		inst.Dict.SetStr("_retried", &object.Int{})
		return object.None, nil
	}})
	abDigestCls.Dict.SetStr("http_error_auth_reqed", &object.BuiltinFunc{Name: "http_error_auth_reqed", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// stub — digest auth requires MD5 computation; we just return None
		return object.None, nil
	}})
	m.Dict.SetStr("AbstractDigestAuthHandler", abDigestCls)

	// ---- HTTPDigestAuthHandler / ProxyDigestAuthHandler ----
	httpDigestCls := &object.Class{Name: "HTTPDigestAuthHandler", Dict: object.NewDict(), Bases: []*object.Class{abDigestCls}}
	httpDigestCls.Dict.SetStr("http_error_401", &object.BuiltinFunc{Name: "http_error_401", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	m.Dict.SetStr("HTTPDigestAuthHandler", httpDigestCls)

	proxyDigestCls := &object.Class{Name: "ProxyDigestAuthHandler", Dict: object.NewDict(), Bases: []*object.Class{abDigestCls}}
	proxyDigestCls.Dict.SetStr("http_error_407", &object.BuiltinFunc{Name: "http_error_407", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	m.Dict.SetStr("ProxyDigestAuthHandler", proxyDigestCls)

	// ---- AbstractHTTPHandler ----
	abHTTPCls := &object.Class{Name: "AbstractHTTPHandler", Dict: object.NewDict(), Bases: []*object.Class{baseHandlerCls}}
	abHTTPCls.Dict.SetStr("do_request_", &object.BuiltinFunc{Name: "do_request_", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// do_request_(self, request) — adds Content-Type/Content-Length if data
		if len(a) < 2 {
			return object.None, nil
		}
		req, _ := a[1].(*object.Instance)
		if req == nil {
			return a[1], nil
		}
		data, _ := req.Dict.GetStr("data")
		if data == nil || data == object.None {
			return req, nil
		}
		if h := reqHeaders(req); h != nil {
			if _, hasContentType := h["Content-type"]; !hasContentType {
				h["Content-type"] = "application/x-www-form-urlencoded"
			}
			switch d := data.(type) {
			case *object.Bytes:
				h["Content-length"] = fmt.Sprintf("%d", len(d.V))
			case *object.Str:
				h["Content-length"] = fmt.Sprintf("%d", len(d.V))
			}
		}
		return req, nil
	}})
	abHTTPCls.Dict.SetStr("do_open", &object.BuiltinFunc{Name: "do_open", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// do_open(self, http_class, req) — performs the HTTP request
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "do_open() requires http_class and req")
		}
		req, _ := a[2].(*object.Instance)
		if req == nil {
			return nil, object.Errorf(i.typeErr, "do_open() req must be Request instance")
		}
		urlStr := ""
		if fu, ok := req.Dict.GetStr("full_url"); ok {
			urlStr = object.Str_(fu)
		}
		method := "GET"
		if mth, ok := req.Dict.GetStr("_method"); ok {
			if s, ok2 := mth.(*object.Str); ok2 && s.V != "" {
				method = s.V
			}
		}
		if data, ok := req.Dict.GetStr("data"); ok && data != object.None {
			if _, isNone := data.(*object.NoneType); !isNone {
				method = "POST"
			}
		}
		var bodyReader io.Reader
		if data, ok := req.Dict.GetStr("data"); ok && data != object.None {
			switch d := data.(type) {
			case *object.Bytes:
				bodyReader = strings.NewReader(string(d.V))
			case *object.Str:
				bodyReader = strings.NewReader(d.V)
			}
		}
		httpReq, err := http.NewRequest(method, urlStr, bodyReader)
		if err != nil {
			return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
		}
		// apply headers
		if h := reqHeaders(req); h != nil {
			for k, v := range h {
				httpReq.Header.Set(k, v)
			}
		}
		client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		}}
		resp, err := client.Do(httpReq)
		if err != nil {
			return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
		}
		hdrs := make(map[string]string)
		for k, vs := range resp.Header {
			if len(vs) > 0 {
				hdrs[k] = vs[0]
			}
		}
		return buildAddinfourl(body, resp.StatusCode, hdrs, urlStr), nil
	}})
	m.Dict.SetStr("AbstractHTTPHandler", abHTTPCls)

	// ---- HTTPHandler ----
	httpHandlerCls := &object.Class{Name: "HTTPHandler", Dict: object.NewDict(), Bases: []*object.Class{abHTTPCls}}
	httpHandlerCls.Dict.SetStr("http_open", &object.BuiltinFunc{Name: "http_open", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if doOpen, ok := abHTTPCls.Dict.GetStr("do_open"); ok {
			return ii.(*Interp).callObject(doOpen, []object.Object{self, object.None, a[1]}, nil)
		}
		return object.None, nil
	}})
	m.Dict.SetStr("HTTPHandler", httpHandlerCls)

	// ---- HTTPSHandler ----
	httpsHandlerCls := &object.Class{Name: "HTTPSHandler", Dict: object.NewDict(), Bases: []*object.Class{abHTTPCls}}
	httpsHandlerCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		inst.Dict.SetStr("debuglevel", &object.Int{})
		inst.Dict.SetStr("_context", object.None)
		return object.None, nil
	}})
	httpsHandlerCls.Dict.SetStr("https_open", &object.BuiltinFunc{Name: "https_open", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if doOpen, ok := abHTTPCls.Dict.GetStr("do_open"); ok {
			return ii.(*Interp).callObject(doOpen, []object.Object{self, object.None, a[1]}, nil)
		}
		return object.None, nil
	}})
	m.Dict.SetStr("HTTPSHandler", httpsHandlerCls)

	// ---- FileHandler ----
	fileHandlerCls := &object.Class{Name: "FileHandler", Dict: object.NewDict(), Bases: []*object.Class{baseHandlerCls}}
	fileHandlerCls.Dict.SetStr("open_local_file", &object.BuiltinFunc{Name: "open_local_file", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(getErrCls("URLError"), "open_local_file() requires request")
		}
		req, _ := a[1].(*object.Instance)
		if req == nil {
			return nil, object.Errorf(getErrCls("URLError"), "expected Request instance")
		}
		sel := ""
		if sv, ok := req.Dict.GetStr("selector"); ok {
			sel = object.Str_(sv)
		}
		data, err := readLocalFile(sel)
		if err != nil {
			return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
		}
		hdrs := map[string]string{"Content-Type": "application/octet-stream"}
		return buildAddinfourl(data, 200, hdrs, "file://"+sel), nil
	}})
	fileHandlerCls.Dict.SetStr("file_open", &object.BuiltinFunc{Name: "file_open", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if fn, ok := fileHandlerCls.Dict.GetStr("open_local_file"); ok {
			return ii.(*Interp).callObject(fn, []object.Object{self, a[1]}, nil)
		}
		return object.None, nil
	}})
	m.Dict.SetStr("FileHandler", fileHandlerCls)

	// ---- DataHandler ----
	dataHandlerCls := &object.Class{Name: "DataHandler", Dict: object.NewDict(), Bases: []*object.Class{baseHandlerCls}}
	dataHandlerCls.Dict.SetStr("data_open", &object.BuiltinFunc{Name: "data_open", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(getErrCls("URLError"), "data_open() requires request")
		}
		urlStr := ""
		switch v := a[1].(type) {
		case *object.Str:
			urlStr = v.V
		case *object.Instance:
			if fu, ok := v.Dict.GetStr("full_url"); ok {
				urlStr = object.Str_(fu)
			}
		}
		body, hdrs, err := parseDataURI(urlStr)
		if err != nil {
			return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
		}
		return buildAddinfourl(body, 200, hdrs, urlStr), nil
	}})
	m.Dict.SetStr("DataHandler", dataHandlerCls)

	// ---- UnknownHandler ----
	unknownHandlerCls := &object.Class{Name: "UnknownHandler", Dict: object.NewDict(), Bases: []*object.Class{baseHandlerCls}}
	unknownHandlerCls.Dict.SetStr("unknown_open", &object.BuiltinFunc{Name: "unknown_open", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		scheme := "unknown"
		if len(a) >= 2 {
			if req, ok := a[1].(*object.Instance); ok {
				if tv, ok2 := req.Dict.GetStr("type"); ok2 {
					scheme = object.Str_(tv)
				}
			}
		}
		return nil, object.Errorf(getErrCls("URLError"), "unknown url type: '%s'", scheme)
	}})
	m.Dict.SetStr("UnknownHandler", unknownHandlerCls)

	// ---- FTPHandler / CacheFTPHandler stubs ----
	ftpHandlerCls := &object.Class{Name: "FTPHandler", Dict: object.NewDict(), Bases: []*object.Class{baseHandlerCls}}
	m.Dict.SetStr("FTPHandler", ftpHandlerCls)
	cacheFTPCls := &object.Class{Name: "CacheFTPHandler", Dict: object.NewDict(), Bases: []*object.Class{ftpHandlerCls}}
	cacheFTPCls.Dict.SetStr("setTimeout", &object.BuiltinFunc{Name: "setTimeout", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	cacheFTPCls.Dict.SetStr("setMaxConns", &object.BuiltinFunc{Name: "setMaxConns", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	m.Dict.SetStr("CacheFTPHandler", cacheFTPCls)

	// ---- OpenerDirector ----
	openerCls := &object.Class{Name: "OpenerDirector", Dict: object.NewDict()}
	openerCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		st := &openerState{
			openHandlers:  make(map[string][]object.Object),
			errorHandlers: make(map[string][]object.Object),
			reqHandlers:   make(map[string][]object.Object),
			respHandlers:  make(map[string][]object.Object),
		}
		urllibOpenerState.Store(inst, st)
		return object.None, nil
	}})

	openerCls.Dict.SetStr("add_handler", &object.BuiltinFunc{Name: "add_handler", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		handler := a[1]
		// Set parent reference on handler
		if hInst, ok := handler.(*object.Instance); ok {
			if addP, ok2 := classLookup(hInst.Class, "add_parent"); ok2 {
				ii.(*Interp).callObject(addP, []object.Object{hInst, self}, nil) //nolint
			}
		}
		stateVal, _ := urllibOpenerState.Load(self)
		if stateVal == nil {
			return object.None, nil
		}
		st := stateVal.(*openerState)
		st.mu.Lock()
		defer st.mu.Unlock()

		// scan handler class for protocol_open / http_error_NNN etc.
		hCls := openerHandlerClass(handler)
		if hCls == nil {
			return object.None, nil
		}
		ks, _ := hCls.Dict.Items()
		for _, k := range ks {
			name := object.Str_(k)
			if strings.HasSuffix(name, "_open") {
				scheme := strings.TrimSuffix(name, "_open")
				st.openHandlers[scheme] = append(st.openHandlers[scheme], handler)
			} else if strings.HasPrefix(name, "http_error_") {
				code := strings.TrimPrefix(name, "http_error_")
				if code != "default" {
					st.errorHandlers[code] = append(st.errorHandlers[code], handler)
				} else {
					st.errorHandlers["default"] = append(st.errorHandlers["default"], handler)
				}
			} else if strings.HasSuffix(name, "_request") {
				scheme := strings.TrimSuffix(name, "_request")
				st.reqHandlers[scheme] = append(st.reqHandlers[scheme], handler)
			} else if strings.HasSuffix(name, "_response") {
				scheme := strings.TrimSuffix(name, "_response")
				st.respHandlers[scheme] = append(st.respHandlers[scheme], handler)
			}
		}
		return object.None, nil
	}})

	openerCls.Dict.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "OpenerDirector.open() requires url")
		}
		self := a[0].(*object.Instance)
		urlArg := a[1]

		// wrap string in Request if needed
		var req *object.Instance
		if s, ok := urlArg.(*object.Str); ok {
			req = &object.Instance{Class: reqCls, Dict: object.NewDict()}
			reqSetURL(req, s.V)
			req.Dict.SetStr("data", object.None)
			req.Dict.SetStr("_method", &object.Str{V: ""})
			req.Dict.SetStr("unverifiable", object.False)
			urllibReqMap.Store(req, &reqState{
				headers:          make(map[string]string),
				unredirectedHdrs: make(map[string]string),
			})
		} else if inst, ok := urlArg.(*object.Instance); ok {
			req = inst
		}

		stateVal, _ := urllibOpenerState.Load(self)
		if stateVal == nil {
			// No handlers — fall back to built-in urlopen
			if urlopen, ok := m.Dict.GetStr("urlopen"); ok {
				return ii.(*Interp).callObject(urlopen, []object.Object{urlArg}, kw)
			}
			return object.None, nil
		}
		st := stateVal.(*openerState)

		scheme := "http"
		if req != nil {
			if tv, ok := req.Dict.GetStr("type"); ok {
				scheme = object.Str_(tv)
			}
		}

		// protocol_request chain
		st.mu.Lock()
		reqChain := append([]object.Object(nil), st.reqHandlers[scheme]...)
		openChain := append([]object.Object(nil), st.openHandlers[scheme]...)
		respChain := append([]object.Object(nil), st.respHandlers[scheme]...)
		st.mu.Unlock()

		var cur object.Object = req
		if req != nil {
			cur = req
		} else {
			cur = urlArg
		}

		// run request processors
		for _, h := range reqChain {
			if hInst, ok := h.(*object.Instance); ok {
				if fn, ok2 := classLookup(hInst.Class, scheme+"_request"); ok2 {
					result, err := ii.(*Interp).callObject(fn, []object.Object{hInst, cur}, nil)
					if err != nil {
						return nil, err
					}
					if result != nil && result != object.None {
						cur = result
					}
				}
			}
		}

		// run open handlers
		var resp object.Object
		for _, h := range openChain {
			if hInst, ok := h.(*object.Instance); ok {
				if fn, ok2 := classLookup(hInst.Class, scheme+"_open"); ok2 {
					result, err := ii.(*Interp).callObject(fn, []object.Object{hInst, cur}, nil)
					if err != nil {
						return nil, err
					}
					if result != nil && result != object.None {
						resp = result
						break
					}
				}
			}
		}

		if resp == nil {
			// fall back to built-in urlopen
			if urlopen, ok := m.Dict.GetStr("urlopen"); ok {
				var urlObj object.Object = urlArg
				if req != nil {
					if fu, ok2 := req.Dict.GetStr("full_url"); ok2 {
						urlObj = fu
					}
				}
				result, err := ii.(*Interp).callObject(urlopen, []object.Object{urlObj}, nil)
				if err != nil {
					return nil, err
				}
				resp = result
			}
		}

		// run response processors
		for _, h := range respChain {
			if hInst, ok := h.(*object.Instance); ok {
				if fn, ok2 := classLookup(hInst.Class, scheme+"_response"); ok2 {
					result, err := ii.(*Interp).callObject(fn, []object.Object{hInst, cur, resp}, nil)
					if err != nil {
						return nil, err
					}
					if result != nil && result != object.None {
						resp = result
					}
				}
			}
		}

		return resp, nil
	}})

	openerCls.Dict.SetStr("error", &object.BuiltinFunc{Name: "error", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		// proto := object.Str_(a[1])  // e.g. "http"
		stateVal, _ := urllibOpenerState.Load(self)
		if stateVal == nil {
			return object.None, nil
		}
		st := stateVal.(*openerState)
		st.mu.Lock()
		defChain := append([]object.Object(nil), st.errorHandlers["default"]...)
		st.mu.Unlock()

		// try to raise HTTPError if we have response and code
		if len(a) >= 5 {
			code := int64(0)
			if n, ok := a[4].(*object.Int); ok {
				code = n.V.Int64()
			}
			if code >= 400 {
				for _, h := range defChain {
					if hInst, ok := h.(*object.Instance); ok {
						if fn, ok2 := classLookup(hInst.Class, "http_error_default"); ok2 {
							return ii.(*Interp).callObject(fn, append([]object.Object{hInst}, a[2:]...), nil)
						}
					}
				}
				return nil, object.Errorf(getErrCls("HTTPError"), "HTTP Error %d", code)
			}
		}
		return object.None, nil
	}})

	m.Dict.SetStr("OpenerDirector", openerCls)

	m.Dict.SetStr("build_opener", &object.BuiltinFunc{Name: "build_opener", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := &object.Instance{Class: openerCls, Dict: object.NewDict()}
		// init opener state
		st := &openerState{
			openHandlers:  make(map[string][]object.Object),
			errorHandlers: make(map[string][]object.Object),
			reqHandlers:   make(map[string][]object.Object),
			respHandlers:  make(map[string][]object.Object),
		}
		urllibOpenerState.Store(inst, st)

		// install default handlers
		defaults := []object.Object{
			&object.Instance{Class: unknownHandlerCls, Dict: object.NewDict()},
			&object.Instance{Class: httpHandlerCls, Dict: object.NewDict()},
			&object.Instance{Class: httpsHandlerCls, Dict: object.NewDict()},
			&object.Instance{Class: httpErrHandlerCls, Dict: object.NewDict()},
			&object.Instance{Class: httpErrProcCls, Dict: object.NewDict()},
			&object.Instance{Class: httpRedirCls, Dict: object.NewDict()},
			&object.Instance{Class: httpCookieCls, Dict: object.NewDict()},
			&object.Instance{Class: fileHandlerCls, Dict: object.NewDict()},
			&object.Instance{Class: dataHandlerCls, Dict: object.NewDict()},
		}
		// add user-provided handlers first, replacing defaults
		for _, h := range a {
			defaults = append(defaults, h)
		}
		if addH, ok := openerCls.Dict.GetStr("add_handler"); ok {
			for _, h := range defaults {
				ii.(*Interp).callObject(addH, []object.Object{inst, h}, nil) //nolint
			}
		}
		return inst, nil
	}})

	m.Dict.SetStr("install_opener", &object.BuiltinFunc{Name: "install_opener", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			urllibGlobalOpener = a[0]
		}
		return object.None, nil
	}})

	// ---- urlopen — delegates to global opener or built-in ----
	m.Dict.SetStr("urlopen", &object.BuiltinFunc{Name: "urlopen", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "urlopen() requires url")
		}
		urlStr := ""
		switch v := a[0].(type) {
		case *object.Str:
			urlStr = v.V
		case *object.Instance:
			if fu, ok := v.Dict.GetStr("full_url"); ok {
				urlStr = object.Str_(fu)
			}
		}

		// data: URI — RFC 2397
		if strings.HasPrefix(urlStr, "data:") {
			body, hdrs, err := parseDataURI(urlStr)
			if err != nil {
				return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
			}
			return buildAddinfourl(body, 200, hdrs, urlStr), nil
		}

		// file: URI
		if strings.HasPrefix(urlStr, "file://") {
			path := strings.TrimPrefix(urlStr, "file://")
			data, err := readLocalFile(path)
			if err != nil {
				return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
			}
			return buildAddinfourl(data, 200, map[string]string{"Content-Type": "application/octet-stream"}, urlStr), nil
		}

		// HTTP/HTTPS
		method := "GET"
		var bodyReader io.Reader
		var postData []byte
		if len(a) > 1 && a[1] != object.None {
			switch v := a[1].(type) {
			case *object.Bytes:
				postData = v.V
				method = "POST"
			case *object.Str:
				postData = []byte(v.V)
				method = "POST"
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("data"); ok && v != object.None {
				if b, ok2 := v.(*object.Bytes); ok2 {
					postData = b.V
					method = "POST"
				}
			}
		}
		if postData != nil {
			bodyReader = strings.NewReader(string(postData))
		}

		req, err := http.NewRequest(method, urlStr, bodyReader)
		if err != nil {
			return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
		}
		hdrs := make(map[string]string)
		for k, vs := range resp.Header {
			if len(vs) > 0 {
				hdrs[k] = vs[0]
			}
		}
		return buildAddinfourl(body, resp.StatusCode, hdrs, urlStr), nil
	}})

	// ---- utility functions ----
	m.Dict.SetStr("pathname2url", &object.BuiltinFunc{Name: "pathname2url", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "pathname2url")
		if err != nil {
			return nil, err
		}
		return &object.Str{V: strings.ReplaceAll(s, "\\", "/")}, nil
	}})
	m.Dict.SetStr("url2pathname", &object.BuiltinFunc{Name: "url2pathname", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "url2pathname")
		if err != nil {
			return nil, err
		}
		return &object.Str{V: s}, nil
	}})
	m.Dict.SetStr("getproxies", &object.BuiltinFunc{Name: "getproxies", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewDict(), nil
	}})

	m.Dict.SetStr("urlretrieve", &object.BuiltinFunc{Name: "urlretrieve", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "urlretrieve() requires url")
		}
		urlStr := object.Str_(a[0])
		filename := ""
		if len(a) > 1 {
			if s, ok := a[1].(*object.Str); ok {
				filename = s.V
			}
		}
		var reporthook object.Object
		if len(a) > 2 && a[2] != object.None {
			reporthook = a[2]
		}
		var postData []byte
		if len(a) > 3 && a[3] != object.None {
			if b, ok := a[3].(*object.Bytes); ok {
				postData = b.V
			}
		}

		// fetch URL
		var body []byte
		var hdrs map[string]string
		if strings.HasPrefix(urlStr, "data:") {
			var err error
			body, hdrs, err = parseDataURI(urlStr)
			if err != nil {
				return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
			}
		} else {
			method := "GET"
			var bodyReader io.Reader
			if postData != nil {
				method = "POST"
				bodyReader = strings.NewReader(string(postData))
			}
			req, err := http.NewRequest(method, urlStr, bodyReader)
			if err != nil {
				return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
			}
			resp, err := (&http.Client{}).Do(req)
			if err != nil {
				return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
			}
			defer resp.Body.Close()
			body, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
			}
			hdrs = make(map[string]string)
			for k, vs := range resp.Header {
				if len(vs) > 0 {
					hdrs[k] = vs[0]
				}
			}
		}

		// write to file
		if filename == "" {
			f, err := os.CreateTemp("", "urlretrieve-*")
			if err != nil {
				return nil, object.Errorf(i.osErr, "%s", err.Error())
			}
			filename = f.Name()
			f.Close()
			urllibTempFiles.Store(filename, true)
		}
		if err := os.WriteFile(filename, body, 0600); err != nil {
			return nil, object.Errorf(i.osErr, "%s", err.Error())
		}

		// call reporthook once (simplified: single block = whole body)
		if reporthook != nil {
			totalSize := int64(len(body))
			blockSize := int64(8192)
			blocks := (totalSize + blockSize - 1) / blockSize
			if blocks == 0 {
				blocks = 1
			}
			for blk := int64(0); blk < blocks; blk++ {
				ii.(*Interp).callObject(reporthook, []object.Object{
					&object.Int{V: *new(big.Int).SetInt64(blk)},
					&object.Int{V: *new(big.Int).SetInt64(blockSize)},
					&object.Int{V: *new(big.Int).SetInt64(totalSize)},
				}, nil) //nolint
			}
		}

		hDict := object.NewDict()
		for k, v := range hdrs {
			hDict.Set(&object.Str{V: k}, &object.Str{V: v})
		}
		return &object.Tuple{V: []object.Object{&object.Str{V: filename}, hDict}}, nil
	}})

	m.Dict.SetStr("urlcleanup", &object.BuiltinFunc{Name: "urlcleanup", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		urllibTempFiles.Range(func(k, _ any) bool {
			os.Remove(k.(string)) //nolint
			urllibTempFiles.Delete(k)
			return true
		})
		return object.None, nil
	}})

	return m
}

// reqSetURL parses urlStr and populates full_url/type/host/origin_req_host/selector on inst.
func reqSetURL(inst *object.Instance, urlStr string) {
	inst.Dict.SetStr("full_url", &object.Str{V: urlStr})
	scheme, host, sel := "", "", "/"
	if idx := strings.Index(urlStr, "://"); idx >= 0 {
		scheme = strings.ToLower(urlStr[:idx])
		rest := urlStr[idx+3:]
		if slash := strings.IndexByte(rest, '/'); slash >= 0 {
			host = rest[:slash]
			sel = rest[slash:]
		} else if q := strings.IndexByte(rest, '?'); q >= 0 {
			host = rest[:q]
			sel = rest[q:]
		} else {
			host = rest
		}
	}
	inst.Dict.SetStr("type", &object.Str{V: scheme})
	inst.Dict.SetStr("host", &object.Str{V: host})
	inst.Dict.SetStr("origin_req_host", &object.Str{V: host})
	inst.Dict.SetStr("selector", &object.Str{V: sel})
}

// addRespMethods populates the addinfourl class with read/readline/readlines/close/geturl/info/getcode.
func addRespMethods(cls *object.Class) {
	cls.Dict.SetStr("read", &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Bytes{}, nil
		}
		inst := a[0].(*object.Instance)
		bodyObj, _ := inst.Dict.GetStr("_body")
		body, _ := bodyObj.(*object.Bytes)
		if body == nil {
			return &object.Bytes{}, nil
		}
		posObj, _ := inst.Dict.GetStr("_pos")
		pos := int64(0)
		if n, ok := posObj.(*object.Int); ok {
			pos = n.V.Int64()
		}
		remaining := body.V[pos:]
		size := int64(-1)
		if len(a) > 1 {
			if n, ok := a[1].(*object.Int); ok {
				size = n.V.Int64()
			}
		}
		var result []byte
		if size < 0 || int(size) >= len(remaining) {
			result = remaining
			inst.Dict.SetStr("_pos", &object.Int{V: *new(big.Int).SetInt64(int64(len(body.V)))})
		} else {
			result = remaining[:size]
			inst.Dict.SetStr("_pos", &object.Int{V: *new(big.Int).SetInt64(pos + size)})
		}
		return &object.Bytes{V: result}, nil
	}})
	cls.Dict.SetStr("readline", &object.BuiltinFunc{Name: "readline", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Bytes{}, nil
		}
		inst := a[0].(*object.Instance)
		bodyObj, _ := inst.Dict.GetStr("_body")
		body, _ := bodyObj.(*object.Bytes)
		if body == nil {
			return &object.Bytes{}, nil
		}
		posObj, _ := inst.Dict.GetStr("_pos")
		pos := int64(0)
		if n, ok := posObj.(*object.Int); ok {
			pos = n.V.Int64()
		}
		remaining := body.V[pos:]
		nl := bytes.IndexByte(remaining, '\n')
		var line []byte
		if nl < 0 {
			line = remaining
			inst.Dict.SetStr("_pos", &object.Int{V: *new(big.Int).SetInt64(int64(len(body.V)))})
		} else {
			line = remaining[:nl+1]
			inst.Dict.SetStr("_pos", &object.Int{V: *new(big.Int).SetInt64(pos + int64(nl+1))})
		}
		return &object.Bytes{V: line}, nil
	}})
	cls.Dict.SetStr("readlines", &object.BuiltinFunc{Name: "readlines", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{}, nil
		}
		inst := a[0].(*object.Instance)
		bodyObj, _ := inst.Dict.GetStr("_body")
		body, _ := bodyObj.(*object.Bytes)
		if body == nil {
			return &object.List{}, nil
		}
		parts := bytes.Split(body.V, []byte("\n"))
		result := &object.List{}
		for idx2, part := range parts {
			if idx2 < len(parts)-1 {
				result.V = append(result.V, &object.Bytes{V: append(append([]byte{}, part...), '\n')})
			} else if len(part) > 0 {
				result.V = append(result.V, &object.Bytes{V: part})
			}
		}
		return result, nil
	}})
	cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	cls.Dict.SetStr("geturl", &object.BuiltinFunc{Name: "geturl", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{}, nil
		}
		inst := a[0].(*object.Instance)
		if v, ok := inst.Dict.GetStr("url"); ok {
			return v, nil
		}
		return &object.Str{}, nil
	}})
	cls.Dict.SetStr("info", &object.BuiltinFunc{Name: "info", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewDict(), nil
		}
		inst := a[0].(*object.Instance)
		if v, ok := inst.Dict.GetStr("headers"); ok {
			return v, nil
		}
		return object.NewDict(), nil
	}})
	cls.Dict.SetStr("getcode", &object.BuiltinFunc{Name: "getcode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Int{}, nil
		}
		inst := a[0].(*object.Instance)
		if v, ok := inst.Dict.GetStr("code"); ok {
			return v, nil
		}
		return &object.Int{}, nil
	}})
	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		return a[0], nil
	}})
	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})
}

// passwdFind finds (user, passwd) for the longest URI prefix match in realm.
func passwdFind(st *passwdMgrState, realm, authURI string) (string, string) {
	bestLen := -1
	bestUser, bestPasswd := "", ""
	for _, e := range st.entries[realm] {
		if strings.HasPrefix(authURI, e.uri) && len(e.uri) > bestLen {
			bestLen = len(e.uri)
			bestUser = e.user
			bestPasswd = e.passwd
		}
	}
	return bestUser, bestPasswd
}

// openerHandlerClass returns the *object.Class of a handler object.
func openerHandlerClass(handler object.Object) *object.Class {
	switch h := handler.(type) {
	case *object.Instance:
		return h.Class
	case *object.Class:
		return h
	}
	return nil
}

// extractBasicRealm parses realm= from a WWW-Authenticate: Basic realm="..." header value.
func extractBasicRealm(s string) string {
	lower := strings.ToLower(s)
	idx := strings.Index(lower, "realm=")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(s[idx+6:])
	if strings.HasPrefix(rest, `"`) {
		rest = rest[1:]
		if end := strings.IndexByte(rest, '"'); end >= 0 {
			return rest[:end]
		}
		return rest
	}
	if end := strings.IndexAny(rest, " ,"); end >= 0 {
		return rest[:end]
	}
	return rest
}

// readLocalFile reads a file from the local filesystem.
func readLocalFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// parseDataURI parses a data: URI and returns body bytes and content-type headers.
func parseDataURI(uri string) ([]byte, map[string]string, error) {
	// data:[<mediatype>][;base64],<data>
	rest := strings.TrimPrefix(uri, "data:")
	comma := strings.Index(rest, ",")
	if comma < 0 {
		return nil, nil, fmt.Errorf("invalid data URI: missing comma")
	}
	meta := rest[:comma]
	dataStr := rest[comma+1:]

	isBase64 := strings.HasSuffix(meta, ";base64")
	mediaType := strings.TrimSuffix(meta, ";base64")
	if mediaType == "" {
		mediaType = "text/plain;charset=US-ASCII"
	}

	var body []byte
	if isBase64 {
		var err error
		body, err = base64.StdEncoding.DecodeString(dataStr)
		if err != nil {
			body, err = base64.RawStdEncoding.DecodeString(dataStr)
			if err != nil {
				return nil, nil, fmt.Errorf("base64 decode: %w", err)
			}
		}
	} else {
		decoded := strings.ReplaceAll(dataStr, "%20", " ")
		// percent-decode
		var sb strings.Builder
		for j := 0; j < len(decoded); j++ {
			if decoded[j] == '%' && j+2 < len(decoded) {
				var b byte
				for _, c := range decoded[j+1 : j+3] {
					b <<= 4
					switch {
					case c >= '0' && c <= '9':
						b |= byte(c - '0')
					case c >= 'a' && c <= 'f':
						b |= byte(c-'a') + 10
					case c >= 'A' && c <= 'F':
						b |= byte(c-'A') + 10
					}
				}
				sb.WriteByte(b)
				j += 2
			} else {
				sb.WriteByte(decoded[j])
			}
		}
		body = []byte(sb.String())
	}

	hdrs := map[string]string{"Content-Type": mediaType}
	return body, hdrs, nil
}

func capitalizeHeader(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// ---- urllib.robotparser ---------------------------------------------------

type robotParserState struct {
	mu       sync.Mutex
	agents   map[string][]robotRule // lowercase agent -> rules
	sitemaps []string
	mtime    float64
}

type robotRule struct {
	allow bool
	path  string
}

var robotParserMap sync.Map // *object.Instance -> *robotParserState

func (i *Interp) buildUrllibRobotParser() *object.Module {
	m := &object.Module{Name: "urllib.robotparser", Dict: object.NewDict()}
	m.Dict.SetStr("__package__", &object.Str{V: "urllib"})

	rfpCls := &object.Class{Name: "RobotFileParser", Dict: object.NewDict()}

	rfpCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		urlStr := ""
		if len(a) > 1 {
			if s, ok := a[1].(*object.Str); ok {
				urlStr = s.V
			}
		}
		inst.Dict.SetStr("_url", &object.Str{V: urlStr})
		state := &robotParserState{agents: make(map[string][]robotRule)}
		robotParserMap.Store(inst, state)
		return object.None, nil
	}})

	rfpCls.Dict.SetStr("set_url", &object.BuiltinFunc{Name: "set_url", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		if s, ok := a[1].(*object.Str); ok {
			inst.Dict.SetStr("_url", s)
		}
		return object.None, nil
	}})

	rfpCls.Dict.SetStr("read", &object.BuiltinFunc{Name: "read", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		urlObj, _ := inst.Dict.GetStr("_url")
		urlStr := ""
		if s, ok := urlObj.(*object.Str); ok {
			urlStr = s.V
		}
		if urlStr == "" {
			return object.None, nil
		}
		resp, err := http.Get(urlStr) //nolint:noctx
		if err != nil {
			return object.None, nil
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		lines := strings.Split(string(data), "\n")
		lineObjs := &object.List{}
		for _, ln := range lines {
			lineObjs.V = append(lineObjs.V, &object.Str{V: ln})
		}
		// call parse
		if parseFn, ok := rfpCls.Dict.GetStr("parse"); ok {
			ii.(*Interp).callObject(parseFn, []object.Object{inst, lineObjs}, nil) //nolint
		}
		return object.None, nil
	}})

	rfpCls.Dict.SetStr("parse", &object.BuiltinFunc{Name: "parse", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		stateVal, ok := robotParserMap.Load(inst)
		if !ok {
			return object.None, nil
		}
		state := stateVal.(*robotParserState)
		state.mu.Lock()
		defer state.mu.Unlock()
		state.agents = make(map[string][]robotRule)
		state.sitemaps = nil

		// Iterate lines
		it, err := ii.(*Interp).getIter(a[1])
		if err != nil {
			return object.None, nil
		}
		var currentAgents []string
		for {
			v, ok2, err2 := it.Next()
			if err2 != nil || !ok2 {
				break
			}
			line := ""
			if s, ok3 := v.(*object.Str); ok3 {
				line = s.V
			} else {
				line = object.Str_(v)
			}
			// strip comment
			if idx := strings.Index(line, "#"); idx >= 0 {
				line = line[:idx]
			}
			line = strings.TrimSpace(line)
			if line == "" {
				currentAgents = nil
				continue
			}
			colonIdx := strings.Index(line, ":")
			if colonIdx < 0 {
				continue
			}
			field := strings.TrimSpace(line[:colonIdx])
			value := strings.TrimSpace(line[colonIdx+1:])
			switch strings.ToLower(field) {
			case "user-agent":
				currentAgents = append(currentAgents, strings.ToLower(value))
			case "disallow":
				for _, ag := range currentAgents {
					state.agents[ag] = append(state.agents[ag], robotRule{allow: false, path: value})
				}
			case "allow":
				for _, ag := range currentAgents {
					state.agents[ag] = append(state.agents[ag], robotRule{allow: true, path: value})
				}
			case "crawl-delay":
				for _, ag := range currentAgents {
					inst.Dict.SetStr("_crawl_delay_"+ag, &object.Str{V: value})
				}
			case "request-rate":
				for _, ag := range currentAgents {
					inst.Dict.SetStr("_request_rate_"+ag, &object.Str{V: value})
				}
			case "sitemap":
				state.sitemaps = append(state.sitemaps, value)
			}
		}
		return object.None, nil
	}})

	rfpCls.Dict.SetStr("can_fetch", &object.BuiltinFunc{Name: "can_fetch", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.True, nil
		}
		inst := a[0].(*object.Instance)
		agent := ""
		if s, ok := a[1].(*object.Str); ok {
			agent = strings.ToLower(s.V)
		}
		path := ""
		if s, ok := a[2].(*object.Str); ok {
			path = s.V
		}
		stateVal, ok := robotParserMap.Load(inst)
		if !ok {
			return object.True, nil
		}
		state := stateVal.(*robotParserState)
		state.mu.Lock()
		defer state.mu.Unlock()

		allowed := robotCanFetch(state, agent, path)
		if allowed {
			return object.True, nil
		}
		return object.False, nil
	}})

	rfpCls.Dict.SetStr("crawl_delay", &object.BuiltinFunc{Name: "crawl_delay", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		agent := ""
		if s, ok := a[1].(*object.Str); ok {
			agent = strings.ToLower(s.V)
		}
		if v, ok := inst.Dict.GetStr("_crawl_delay_" + agent); ok {
			return v, nil
		}
		return object.None, nil
	}})

	rfpCls.Dict.SetStr("request_rate", &object.BuiltinFunc{Name: "request_rate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		agent := ""
		if s, ok := a[1].(*object.Str); ok {
			agent = strings.ToLower(s.V)
		}
		if v, ok := inst.Dict.GetStr("_request_rate_" + agent); ok {
			return v, nil
		}
		return object.None, nil
	}})

	rfpCls.Dict.SetStr("site_maps", &object.BuiltinFunc{Name: "site_maps", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		stateVal, ok := robotParserMap.Load(inst)
		if !ok {
			return object.None, nil
		}
		state := stateVal.(*robotParserState)
		state.mu.Lock()
		defer state.mu.Unlock()
		if len(state.sitemaps) == 0 {
			return object.None, nil
		}
		out := &object.List{}
		for _, s := range state.sitemaps {
			out.V = append(out.V, &object.Str{V: s})
		}
		return out, nil
	}})

	rfpCls.Dict.SetStr("mtime", &object.BuiltinFunc{Name: "mtime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Float{V: 0}, nil
		}
		inst := a[0].(*object.Instance)
		stateVal, ok := robotParserMap.Load(inst)
		if !ok {
			return &object.Float{V: 0}, nil
		}
		state := stateVal.(*robotParserState)
		return &object.Float{V: state.mtime}, nil
	}})

	rfpCls.Dict.SetStr("modified", &object.BuiltinFunc{Name: "modified", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	m.Dict.SetStr("RobotFileParser", rfpCls)
	return m
}

// robotCanFetch applies longest-match-wins with Allow > Disallow precedence.
func robotCanFetch(state *robotParserState, agent, path string) bool {
	rules := robotRulesFor(state, agent)
	if rules == nil {
		return true
	}
	bestLen := -1
	bestAllow := true
	for _, rule := range rules {
		if rule.path == "" {
			if bestLen < 0 {
				bestLen = 0
				bestAllow = rule.allow
			}
			continue
		}
		if strings.HasPrefix(path, rule.path) {
			l := len(rule.path)
			if l > bestLen || (l == bestLen && rule.allow) {
				bestLen = l
				bestAllow = rule.allow
			}
		}
	}
	return bestAllow
}

func robotRulesFor(state *robotParserState, agent string) []robotRule {
	if r, ok := state.agents[agent]; ok {
		return r
	}
	// try agent prefix (e.g. "googlebot/2.1" matches "googlebot")
	for k, v := range state.agents {
		if strings.HasPrefix(agent, k) {
			return v
		}
	}
	if r, ok := state.agents["*"]; ok {
		return r
	}
	return nil
}


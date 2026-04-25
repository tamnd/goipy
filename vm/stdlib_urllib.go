package vm

import (
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"net/http"
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
		case "url":
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

var urllibRequestHeadersMap sync.Map // *object.Instance -> map[string]string
var urllibGlobalOpener object.Object // the currently installed opener

func (i *Interp) buildUrllibRequest() *object.Module {
	m := &object.Module{Name: "urllib.request", Dict: object.NewDict()}
	m.Dict.SetStr("__package__", &object.Str{V: "urllib"})

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
		inst.Dict.SetStr("full_url", &object.Str{V: urlStr})
		inst.Dict.SetStr("_full_url", &object.Str{V: urlStr})

		// parse parts
		scheme, rest, _ := strings.Cut(urlStr, "://")
		if rest == "" {
			scheme = ""
			rest = urlStr
		}
		host, path, _ := strings.Cut(rest, "/")
		if path == "" {
			path = "/"
		} else {
			path = "/" + path
		}
		// selector = path + query
		inst.Dict.SetStr("type", &object.Str{V: strings.ToLower(scheme)})
		inst.Dict.SetStr("host", &object.Str{V: host})
		inst.Dict.SetStr("selector", &object.Str{V: path})

		// data
		if len(a) > 2 {
			inst.Dict.SetStr("data", a[2])
		} else if kw != nil {
			if v, ok := kw.GetStr("data"); ok {
				inst.Dict.SetStr("data", v)
			} else {
				inst.Dict.SetStr("data", object.None)
			}
		} else {
			inst.Dict.SetStr("data", object.None)
		}

		// method
		method := ""
		if kw != nil {
			if v, ok := kw.GetStr("method"); ok {
				if s, ok := v.(*object.Str); ok {
					method = s.V
				}
			}
		}
		if len(a) > 5 {
			if s, ok := a[5].(*object.Str); ok {
				method = s.V
			}
		}
		inst.Dict.SetStr("_method", &object.Str{V: method})

		// headers as map stored in sync.Map
		headers := make(map[string]string)
		if len(a) > 3 {
			if d, ok := a[3].(*object.Dict); ok {
				ks, vs := d.Items()
				for idx, k := range ks {
					if ks2, ok := k.(*object.Str); ok {
						if vs2, ok := vs[idx].(*object.Str); ok {
							headers[capitalizeHeader(ks2.V)] = vs2.V
						}
					}
				}
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("headers"); ok {
				if d, ok := v.(*object.Dict); ok {
					ks, vs := d.Items()
					for idx, k := range ks {
						if ks2, ok := k.(*object.Str); ok {
							if vs2, ok := vs[idx].(*object.Str); ok {
								headers[capitalizeHeader(ks2.V)] = vs2.V
							}
						}
					}
				}
			}
		}
		urllibRequestHeadersMap.Store(inst, headers)
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
			return &object.Str{V: "POST"}, nil
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
		if h, ok := urllibRequestHeadersMap.Load(inst); ok {
			h.(map[string]string)[capitalizeHeader(k.V)] = v.V
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
		if h, ok := urllibRequestHeadersMap.Load(inst); ok {
			_, found := h.(map[string]string)[capitalizeHeader(k.V)]
			if found {
				return object.True, nil
			}
		}
		return object.False, nil
	}})

	reqCls.Dict.SetStr("get_header", &object.BuiltinFunc{Name: "get_header", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
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
		if h, ok := urllibRequestHeadersMap.Load(inst); ok {
			if v, found := h.(map[string]string)[capitalizeHeader(k.V)]; found {
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
		if h, ok := urllibRequestHeadersMap.Load(inst); ok {
			delete(h.(map[string]string), capitalizeHeader(k.V))
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
		if h, ok := urllibRequestHeadersMap.Load(inst); ok {
			h.(map[string]string)[capitalizeHeader(k.V)] = v.V
		}
		return object.None, nil
	}})

	reqCls.Dict.SetStr("header_items", &object.BuiltinFunc{Name: "header_items", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{}, nil
		}
		inst := a[0].(*object.Instance)
		out := &object.List{}
		if h, ok := urllibRequestHeadersMap.Load(inst); ok {
			for k, v := range h.(map[string]string) {
				out.V = append(out.V, &object.Tuple{V: []object.Object{&object.Str{V: k}, &object.Str{V: v}}})
			}
		}
		return out, nil
	}})

	// ---- addinfourl response class ----
	respCls := &object.Class{Name: "addinfourl", Dict: object.NewDict()}
	buildAddinfourl := func(body []byte, statusCode int, headerMap map[string]string, urlStr string) *object.Instance {
		inst := &object.Instance{Class: respCls, Dict: object.NewDict()}
		inst.Dict.SetStr("_body", &object.Bytes{V: body})
		inst.Dict.SetStr("_pos", &object.Int{})
		inst.Dict.SetStr("url", &object.Str{V: urlStr})
		inst.Dict.SetStr("status", &object.Int{V: *new(big.Int).SetInt64(int64(statusCode))})
		inst.Dict.SetStr("code", &object.Int{V: *new(big.Int).SetInt64(int64(statusCode))})
		// build headers dict
		hDict := object.NewDict()
		for k, v := range headerMap {
			hDict.Set(&object.Str{V: k}, &object.Str{V: v})
		}
		inst.Dict.SetStr("headers", hDict)
		return inst
	}

	respCls.Dict.SetStr("read", &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
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
		if p, ok := posObj.(*object.Int); ok {
			pos = p.V.Int64()
		}
		if len(a) > 1 {
			if n, ok := a[1].(*object.Int); ok {
				sz := n.V.Int64()
				end := pos + sz
				if end > int64(len(body.V)) {
					end = int64(len(body.V))
				}
				chunk := body.V[pos:end]
				inst.Dict.SetStr("_pos", &object.Int{V: *new(big.Int).SetInt64(end)})
				return &object.Bytes{V: chunk}, nil
			}
		}
		chunk := body.V[pos:]
		inst.Dict.SetStr("_pos", &object.Int{V: *new(big.Int).SetInt64(int64(len(body.V)))})
		return &object.Bytes{V: chunk}, nil
	}})

	respCls.Dict.SetStr("readline", &object.BuiltinFunc{Name: "readline", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
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
		if p, ok := posObj.(*object.Int); ok {
			pos = p.V.Int64()
		}
		data := body.V[pos:]
		nl := strings.IndexByte(string(data), '\n')
		if nl < 0 {
			inst.Dict.SetStr("_pos", &object.Int{V: *new(big.Int).SetInt64(int64(len(body.V)))})
			return &object.Bytes{V: data}, nil
		}
		line := data[:nl+1]
		inst.Dict.SetStr("_pos", &object.Int{V: *new(big.Int).SetInt64(pos + int64(nl+1))})
		return &object.Bytes{V: line}, nil
	}})

	respCls.Dict.SetStr("readlines", &object.BuiltinFunc{Name: "readlines", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{}, nil
		}
		inst := a[0].(*object.Instance)
		bodyObj, _ := inst.Dict.GetStr("_body")
		body, _ := bodyObj.(*object.Bytes)
		if body == nil {
			return &object.List{}, nil
		}
		posObj, _ := inst.Dict.GetStr("_pos")
		pos := int64(0)
		if p, ok := posObj.(*object.Int); ok {
			pos = p.V.Int64()
		}
		data := string(body.V[pos:])
		lines := strings.SplitAfter(data, "\n")
		out := &object.List{}
		for _, ln := range lines {
			if ln == "" {
				continue
			}
			out.V = append(out.V, &object.Bytes{V: []byte(ln)})
		}
		inst.Dict.SetStr("_pos", &object.Int{V: *new(big.Int).SetInt64(int64(len(body.V)))})
		return out, nil
	}})

	respCls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	respCls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			return a[0], nil
		}
		return object.None, nil
	}})
	respCls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})
	respCls.Dict.SetStr("geturl", &object.BuiltinFunc{Name: "geturl", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		inst := a[0].(*object.Instance)
		if v, ok := inst.Dict.GetStr("url"); ok {
			return v, nil
		}
		return &object.Str{V: ""}, nil
	}})
	respCls.Dict.SetStr("info", &object.BuiltinFunc{Name: "info", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewDict(), nil
		}
		inst := a[0].(*object.Instance)
		if v, ok := inst.Dict.GetStr("headers"); ok {
			return v, nil
		}
		return object.NewDict(), nil
	}})
	respCls.Dict.SetStr("getcode", &object.BuiltinFunc{Name: "getcode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		if v, ok := inst.Dict.GetStr("code"); ok {
			return v, nil
		}
		return object.None, nil
	}})

	// ---- urlopen ----
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
				if s, ok := fu.(*object.Str); ok {
					urlStr = s.V
				}
			}
		}

		// data: URI — RFC 2397
		if strings.HasPrefix(urlStr, "data:") {
			body, hdrs, err := parseDataURI(urlStr)
			if err != nil {
				return nil, object.Errorf(getErrCls("URLError"), "%s", err.Error())
			}
			resp := buildAddinfourl(body, 200, hdrs, urlStr)
			return resp, nil
		}

		// HTTP/HTTPS
		method := "GET"
		var bodyReader io.Reader
		var postData []byte
		if len(a) > 1 && a[1] != object.None {
			if b, ok := a[1].(*object.Bytes); ok {
				postData = b.V
				method = "POST"
			} else if s, ok := a[1].(*object.Str); ok {
				postData = []byte(s.V)
				method = "POST"
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("data"); ok && v != object.None {
				if b, ok := v.(*object.Bytes); ok {
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
		result := buildAddinfourl(body, resp.StatusCode, hdrs, urlStr)
		return result, nil
	}})

	// ---- OpenerDirector ----
	openerCls := &object.Class{Name: "OpenerDirector", Dict: object.NewDict()}
	openerCls.Dict.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		// delegate to urlopen
		if urlopen, ok := m.Dict.GetStr("urlopen"); ok {
			return ii.(*Interp).callObject(urlopen, a[1:], kw)
		}
		return nil, object.Errorf(i.runtimeErr, "OpenerDirector.open not available")
	}})
	openerCls.Dict.SetStr("add_handler", &object.BuiltinFunc{Name: "add_handler", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	openerCls.Dict.SetStr("error", &object.BuiltinFunc{Name: "error", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	m.Dict.SetStr("build_opener", &object.BuiltinFunc{Name: "build_opener", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := &object.Instance{Class: openerCls, Dict: object.NewDict()}
		return inst, nil
	}})

	m.Dict.SetStr("install_opener", &object.BuiltinFunc{Name: "install_opener", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			urllibGlobalOpener = a[0]
		}
		return object.None, nil
	}})

	// ---- handler stubs ----
	for _, name := range []string{
		"BaseHandler", "HTTPDefaultErrorHandler", "HTTPRedirectHandler",
		"HTTPCookieProcessor", "ProxyHandler", "HTTPPasswordMgr",
		"HTTPPasswordMgrWithDefaultRealm", "HTTPPasswordMgrWithPriorAuth",
		"HTTPBasicAuthHandler", "ProxyBasicAuthHandler", "HTTPDigestAuthHandler",
		"ProxyDigestAuthHandler", "HTTPHandler", "HTTPSHandler", "FileHandler",
		"DataHandler", "FTPHandler", "CacheFTPHandler", "UnknownHandler",
		"HTTPErrorProcessor",
	} {
		n := name
		cls := &object.Class{Name: n, Dict: object.NewDict()}
		m.Dict.SetStr(n, cls)
	}

	m.Dict.SetStr("OpenerDirector", openerCls)
	m.Dict.SetStr("Request", reqCls)

	// ---- simple utility functions ----
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
	m.Dict.SetStr("getproxies", &object.BuiltinFunc{Name: "getproxies", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewDict(), nil
	}})
	m.Dict.SetStr("urlretrieve", &object.BuiltinFunc{Name: "urlretrieve", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{&object.Str{V: ""}, object.NewDict()}}, nil
	}})
	m.Dict.SetStr("urlcleanup", &object.BuiltinFunc{Name: "urlcleanup", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	return m
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


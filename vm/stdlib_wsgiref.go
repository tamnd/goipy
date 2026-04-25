package vm

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/tamnd/goipy/object"
)

// wsgiHopByHopHeaders per RFC 2616
var wsgiHopByHopHeaders = map[string]bool{
	"connection":          true,
	"keep-alive":          true,
	"proxy-authenticate":  true,
	"proxy-authorization": true,
	"te":                  true,
	"trailers":            true,
	"transfer-encoding":   true,
	"upgrade":             true,
}

var wsgiFileWrapperMap sync.Map // *object.Instance -> *wsgiFileWrapperState
var wsgiServerMap sync.Map      // *object.Instance -> net.Listener

type wsgiFileWrapperState struct {
	filelike object.Object
	blksize  int64
	done     bool
}

// ── wsgiref (namespace) ───────────────────────────────────────────────────────

func (i *Interp) buildWsgiref() *object.Module {
	m := &object.Module{Name: "wsgiref", Dict: object.NewDict()}
	m.Dict.SetStr("__path__", &object.List{V: []object.Object{&object.Str{V: "wsgiref"}}})
	m.Dict.SetStr("__package__", &object.Str{V: "wsgiref"})
	return m
}

// ── wsgiref.util ──────────────────────────────────────────────────────────────

func (i *Interp) buildWsgirefUtil() *object.Module {
	m := &object.Module{Name: "wsgiref.util", Dict: object.NewDict()}

	getenv := func(d *object.Dict, key string) string {
		if v, ok := d.GetStr(key); ok {
			if s, ok2 := v.(*object.Str); ok2 {
				return s.V
			}
		}
		return ""
	}

	// guess_scheme(environ) -> str
	m.Dict.SetStr("guess_scheme", &object.BuiltinFunc{Name: "guess_scheme", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "http"}, nil
		}
		if d, ok := a[0].(*object.Dict); ok {
			v := strings.ToLower(getenv(d, "HTTPS"))
			if v == "on" || v == "yes" || v == "1" || v == "true" {
				return &object.Str{V: "https"}, nil
			}
		}
		return &object.Str{V: "http"}, nil
	}})

	// setup_testing_defaults(environ) -> None
	m.Dict.SetStr("setup_testing_defaults", &object.BuiltinFunc{Name: "setup_testing_defaults", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		d, ok := a[0].(*object.Dict)
		if !ok {
			return object.None, nil
		}
		d.SetStr("SERVER_NAME", &object.Str{V: "127.0.0.1"})
		d.SetStr("SERVER_PORT", &object.Str{V: "80"})
		d.SetStr("HTTP_HOST", &object.Str{V: "127.0.0.1"})
		d.SetStr("REQUEST_METHOD", &object.Str{V: "GET"})
		d.SetStr("SCRIPT_NAME", &object.Str{V: ""})
		d.SetStr("PATH_INFO", &object.Str{V: "/"})
		d.SetStr("SERVER_PROTOCOL", &object.Str{V: "HTTP/1.0"})
		d.SetStr("wsgi.version", &object.Tuple{V: []object.Object{object.IntFromInt64(1), object.IntFromInt64(0)}})
		d.SetStr("wsgi.url_scheme", &object.Str{V: "http"})
		d.SetStr("wsgi.multithread", object.False)
		d.SetStr("wsgi.multiprocess", object.False)
		d.SetStr("wsgi.run_once", object.False)
		interp := ii.(*Interp)
		var wsgiInput, wsgiErrors object.Object = object.None, object.None
		if ioMod, err2 := interp.loadModule("io"); err2 == nil {
			if bioClass, ok2 := ioMod.Dict.GetStr("BytesIO"); ok2 {
				if inst, err3 := interp.callObject(bioClass, nil, nil); err3 == nil {
					wsgiInput = inst
				}
				if inst, err3 := interp.callObject(bioClass, nil, nil); err3 == nil {
					wsgiErrors = inst
				}
			}
		}
		d.SetStr("wsgi.input", wsgiInput)
		d.SetStr("wsgi.errors", wsgiErrors)
		return object.None, nil
	}})

	// application_uri(environ) -> str
	m.Dict.SetStr("application_uri", &object.BuiltinFunc{Name: "application_uri", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "http://127.0.0.1/"}, nil
		}
		d, ok := a[0].(*object.Dict)
		if !ok {
			return &object.Str{V: "http://127.0.0.1/"}, nil
		}
		scheme := getenv(d, "wsgi.url_scheme")
		if scheme == "" {
			scheme = "http"
		}
		host := getenv(d, "HTTP_HOST")
		if host == "" {
			server := getenv(d, "SERVER_NAME")
			port := getenv(d, "SERVER_PORT")
			if port != "" && port != "80" && scheme == "http" {
				host = server + ":" + port
			} else if port != "" && port != "443" && scheme == "https" {
				host = server + ":" + port
			} else {
				host = server
			}
		}
		script := getenv(d, "SCRIPT_NAME")
		if script == "" {
			script = "/"
		}
		return &object.Str{V: scheme + "://" + host + script}, nil
	}})

	// request_uri(environ, include_query=True) -> str
	m.Dict.SetStr("request_uri", &object.BuiltinFunc{Name: "request_uri", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "http://127.0.0.1/"}, nil
		}
		d, ok := a[0].(*object.Dict)
		if !ok {
			return &object.Str{V: "http://127.0.0.1/"}, nil
		}
		includeQuery := true
		if len(a) > 1 {
			includeQuery = a[1] != object.False && a[1] != object.None
		}
		scheme := getenv(d, "wsgi.url_scheme")
		if scheme == "" {
			scheme = "http"
		}
		host := getenv(d, "HTTP_HOST")
		if host == "" {
			server := getenv(d, "SERVER_NAME")
			port := getenv(d, "SERVER_PORT")
			if port != "" && port != "80" && scheme == "http" {
				host = server + ":" + port
			} else if port != "" && port != "443" && scheme == "https" {
				host = server + ":" + port
			} else {
				host = server
			}
		}
		script := getenv(d, "SCRIPT_NAME")
		path := getenv(d, "PATH_INFO")
		query := getenv(d, "QUERY_STRING")
		url := scheme + "://" + host + script + path
		if includeQuery && query != "" {
			url += "?" + query
		}
		return &object.Str{V: url}, nil
	}})

	// shift_path_info(environ) -> str | None
	m.Dict.SetStr("shift_path_info", &object.BuiltinFunc{Name: "shift_path_info", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		d, ok := a[0].(*object.Dict)
		if !ok {
			return object.None, nil
		}
		pathInfo := getenv(d, "PATH_INFO")
		stripped := strings.TrimPrefix(pathInfo, "/")
		if stripped == "" {
			return object.None, nil
		}
		parts := strings.SplitN(stripped, "/", 2)
		name := parts[0]
		if name == "" {
			return object.None, nil
		}
		scriptName := getenv(d, "SCRIPT_NAME")
		d.SetStr("SCRIPT_NAME", &object.Str{V: scriptName + "/" + name})
		newPath := "/"
		if len(parts) > 1 {
			newPath = "/" + parts[1]
		}
		d.SetStr("PATH_INFO", &object.Str{V: newPath})
		return &object.Str{V: name}, nil
	}})

	// is_hop_by_hop(header_name) -> bool
	m.Dict.SetStr("is_hop_by_hop", &object.BuiltinFunc{Name: "is_hop_by_hop", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.False, nil
		}
		name := ""
		if s, ok := a[0].(*object.Str); ok {
			name = strings.ToLower(s.V)
		}
		if wsgiHopByHopHeaders[name] {
			return object.True, nil
		}
		return object.False, nil
	}})

	// FileWrapper class
	fwCls := &object.Class{Name: "FileWrapper", Dict: object.NewDict()}

	fwCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "FileWrapper.__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "FileWrapper requires filelike argument")
		}
		blksize := int64(8192)
		if len(a) > 2 {
			if n, ok := a[2].(*object.Int); ok {
				blksize = n.Int64()
			}
		}
		wsgiFileWrapperMap.Store(self, &wsgiFileWrapperState{filelike: a[1], blksize: blksize})
		return object.None, nil
	}})

	fwCls.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "FileWrapper.__iter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return a[0], nil
	}})

	fwCls.Dict.SetStr("__next__", &object.BuiltinFunc{Name: "FileWrapper.__next__", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		st, ok := wsgiFileWrapperMap.Load(self)
		if !ok {
			return nil, object.Errorf(i.stopIter, "")
		}
		state := st.(*wsgiFileWrapperState)
		if state.done {
			return nil, object.Errorf(i.stopIter, "")
		}
		interp := ii.(*Interp)
		readFn, err := interp.getAttr(state.filelike, "read")
		if err != nil {
			state.done = true
			return nil, object.Errorf(i.stopIter, "")
		}
		chunk, err := interp.callObject(readFn, []object.Object{object.IntFromInt64(state.blksize)}, nil)
		if err != nil {
			state.done = true
			return nil, object.Errorf(i.stopIter, "")
		}
		if b, ok2 := chunk.(*object.Bytes); ok2 && len(b.V) > 0 {
			return chunk, nil
		}
		state.done = true
		return nil, object.Errorf(i.stopIter, "")
	}})

	fwCls.Dict.SetStr("close", &object.BuiltinFunc{Name: "FileWrapper.close", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		if st, ok := wsgiFileWrapperMap.Load(self); ok {
			state := st.(*wsgiFileWrapperState)
			interp := ii.(*Interp)
			if closeFn, err := interp.getAttr(state.filelike, "close"); err == nil {
				interp.callObject(closeFn, nil, nil) //nolint
			}
			wsgiFileWrapperMap.Delete(self)
		}
		return object.None, nil
	}})

	m.Dict.SetStr("FileWrapper", fwCls)

	return m
}

// ── wsgiref.headers ───────────────────────────────────────────────────────────

func (i *Interp) buildWsgirefHeaders() *object.Module {
	m := &object.Module{Name: "wsgiref.headers", Dict: object.NewDict()}

	headersCls := &object.Class{Name: "Headers", Dict: object.NewDict()}

	getItems := func(self *object.Instance) *object.List {
		if v, ok := self.Dict.GetStr("_items"); ok {
			if lst, ok2 := v.(*object.List); ok2 {
				return lst
			}
		}
		lst := &object.List{V: []object.Object{}}
		self.Dict.SetStr("_items", lst)
		return lst
	}

	headersCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "Headers.__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		items := &object.List{V: []object.Object{}}
		if len(a) > 1 && a[1] != object.None {
			if lst, ok := a[1].(*object.List); ok {
				for _, item := range lst.V {
					if tup, ok2 := item.(*object.Tuple); ok2 && len(tup.V) == 2 {
						items.V = append(items.V, tup)
					}
				}
			}
		}
		self.Dict.SetStr("_items", items)
		return object.None, nil
	}})

	headersCls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "Headers.__len__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		return object.IntFromInt64(int64(len(getItems(self).V))), nil
	}})

	findHeader := func(items []object.Object, key string) (int, string) {
		keyLow := strings.ToLower(key)
		for idx, item := range items {
			if tup, ok := item.(*object.Tuple); ok && len(tup.V) == 2 {
				if s, ok2 := tup.V[0].(*object.Str); ok2 {
					if strings.ToLower(s.V) == keyLow {
						val := ""
						if v, ok3 := tup.V[1].(*object.Str); ok3 {
							val = v.V
						}
						return idx, val
					}
				}
			}
		}
		return -1, ""
	}

	headersCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "Headers.__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		_, val := findHeader(getItems(self).V, key)
		if val == "" {
			// Try to find if there really is no match
			if idx, _ := findHeader(getItems(self).V, key); idx < 0 {
				return object.None, nil // Python headers.get returns None for missing
			}
		}
		if idx, v := findHeader(getItems(self).V, key); idx >= 0 {
			return &object.Str{V: v}, nil
		}
		return object.None, nil
	}})

	headersCls.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "Headers.__setitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		val := ""
		if s, ok := a[2].(*object.Str); ok {
			val = s.V
		}
		lst := getItems(self)
		keyLow := strings.ToLower(key)
		newV := make([]object.Object, 0, len(lst.V))
		for _, item := range lst.V {
			if tup, ok := item.(*object.Tuple); ok && len(tup.V) == 2 {
				if s, ok2 := tup.V[0].(*object.Str); ok2 {
					if strings.ToLower(s.V) == keyLow {
						continue
					}
				}
			}
			newV = append(newV, item)
		}
		newV = append(newV, &object.Tuple{V: []object.Object{&object.Str{V: key}, &object.Str{V: val}}})
		lst.V = newV
		return object.None, nil
	}})

	headersCls.Dict.SetStr("__delitem__", &object.BuiltinFunc{Name: "Headers.__delitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		lst := getItems(self)
		keyLow := strings.ToLower(key)
		newV := make([]object.Object, 0, len(lst.V))
		for _, item := range lst.V {
			if tup, ok := item.(*object.Tuple); ok && len(tup.V) == 2 {
				if s, ok2 := tup.V[0].(*object.Str); ok2 {
					if strings.ToLower(s.V) == keyLow {
						continue
					}
				}
			}
			newV = append(newV, item)
		}
		lst.V = newV
		return object.None, nil
	}})

	headersCls.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "Headers.__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		if idx, _ := findHeader(getItems(self).V, key); idx >= 0 {
			return object.True, nil
		}
		return object.False, nil
	}})

	headersCls.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "Headers.__iter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		names := []object.Object{}
		for _, item := range getItems(self).V {
			if tup, ok := item.(*object.Tuple); ok && len(tup.V) == 2 {
				names = append(names, tup.V[0])
			}
		}
		return &object.List{V: names}, nil
	}})

	headersCls.Dict.SetStr("get", &object.BuiltinFunc{Name: "Headers.get", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		dflt := object.Object(object.None)
		if len(a) > 2 {
			dflt = a[2]
		}
		if idx, val := findHeader(getItems(self).V, key); idx >= 0 {
			return &object.Str{V: val}, nil
		}
		return dflt, nil
	}})

	headersCls.Dict.SetStr("setdefault", &object.BuiltinFunc{Name: "Headers.setdefault", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		val := ""
		if len(a) > 2 {
			if s, ok := a[2].(*object.Str); ok {
				val = s.V
			}
		}
		if idx, existing := findHeader(getItems(self).V, key); idx >= 0 {
			return &object.Str{V: existing}, nil
		}
		lst := getItems(self)
		lst.V = append(lst.V, &object.Tuple{V: []object.Object{&object.Str{V: key}, &object.Str{V: val}}})
		return &object.Str{V: val}, nil
	}})

	headersCls.Dict.SetStr("keys", &object.BuiltinFunc{Name: "Headers.keys", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		names := []object.Object{}
		for _, item := range getItems(self).V {
			if tup, ok := item.(*object.Tuple); ok && len(tup.V) == 2 {
				names = append(names, tup.V[0])
			}
		}
		return &object.List{V: names}, nil
	}})

	headersCls.Dict.SetStr("values", &object.BuiltinFunc{Name: "Headers.values", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		vals := []object.Object{}
		for _, item := range getItems(self).V {
			if tup, ok := item.(*object.Tuple); ok && len(tup.V) == 2 {
				vals = append(vals, tup.V[1])
			}
		}
		return &object.List{V: vals}, nil
	}})

	headersCls.Dict.SetStr("items", &object.BuiltinFunc{Name: "Headers.items", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		result := make([]object.Object, len(getItems(self).V))
		copy(result, getItems(self).V)
		return &object.List{V: result}, nil
	}})

	headersCls.Dict.SetStr("get_all", &object.BuiltinFunc{Name: "Headers.get_all", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		keyLow := strings.ToLower(key)
		vals := []object.Object{}
		for _, item := range getItems(self).V {
			if tup, ok := item.(*object.Tuple); ok && len(tup.V) == 2 {
				if s, ok2 := tup.V[0].(*object.Str); ok2 {
					if strings.ToLower(s.V) == keyLow {
						vals = append(vals, tup.V[1])
					}
				}
			}
		}
		return &object.List{V: vals}, nil
	}})

	headersCls.Dict.SetStr("add_header", &object.BuiltinFunc{Name: "Headers.add_header", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		val := ""
		if len(a) > 2 {
			if s, ok := a[2].(*object.Str); ok {
				val = s.V
			}
		}
		parts := []string{val}
		if kw != nil {
			ks, vs := kw.Items()
			for j, k := range ks {
				paramName := ""
				if s, ok := k.(*object.Str); ok {
					paramName = strings.ReplaceAll(s.V, "_", "-")
				}
				paramVal := ""
				if s, ok := vs[j].(*object.Str); ok {
					paramVal = s.V
				}
				if paramName != "" {
					parts = append(parts, paramName+"=\""+paramVal+"\"")
				}
			}
		}
		combined := strings.Join(parts, "; ")
		lst := getItems(self)
		lst.V = append(lst.V, &object.Tuple{V: []object.Object{&object.Str{V: name}, &object.Str{V: combined}}})
		return object.None, nil
	}})

	headersCls.Dict.SetStr("__bytes__", &object.BuiltinFunc{Name: "Headers.__bytes__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		var out []byte
		for _, item := range getItems(self).V {
			if tup, ok := item.(*object.Tuple); ok && len(tup.V) == 2 {
				n := ""
				v := ""
				if s, ok2 := tup.V[0].(*object.Str); ok2 {
					n = s.V
				}
				if s, ok2 := tup.V[1].(*object.Str); ok2 {
					v = s.V
				}
				out = append(out, []byte(n+": "+v+"\r\n")...)
			}
		}
		out = append(out, []byte("\r\n")...)
		return &object.Bytes{V: out}, nil
	}})

	m.Dict.SetStr("Headers", headersCls)

	return m
}

// ── wsgiref.simple_server ─────────────────────────────────────────────────────

func (i *Interp) buildWsgirefSimpleServer() *object.Module {
	m := &object.Module{Name: "wsgiref.simple_server", Dict: object.NewDict()}

	// WSGIServer class
	wsgiServerCls := &object.Class{Name: "WSGIServer", Dict: object.NewDict()}

	wsgiServerCls.Dict.SetStr("set_app", &object.BuiltinFunc{Name: "WSGIServer.set_app", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		if len(a) > 1 {
			self.Dict.SetStr("_app", a[1])
		}
		return object.None, nil
	}})

	wsgiServerCls.Dict.SetStr("get_app", &object.BuiltinFunc{Name: "WSGIServer.get_app", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		if v, ok := self.Dict.GetStr("_app"); ok {
			return v, nil
		}
		return object.None, nil
	}})

	wsgiServerCls.Dict.SetStr("server_close", &object.BuiltinFunc{Name: "WSGIServer.server_close", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		if ln, ok := wsgiServerMap.LoadAndDelete(self); ok {
			ln.(net.Listener).Close() //nolint
		}
		return object.None, nil
	}})

	wsgiServerCls.Dict.SetStr("serve_forever", &object.BuiltinFunc{Name: "WSGIServer.serve_forever", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	wsgiServerCls.Dict.SetStr("handle_request", &object.BuiltinFunc{Name: "WSGIServer.handle_request", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	m.Dict.SetStr("WSGIServer", wsgiServerCls)

	// WSGIRequestHandler class
	reqHandlerCls := &object.Class{Name: "WSGIRequestHandler", Dict: object.NewDict()}
	m.Dict.SetStr("WSGIRequestHandler", reqHandlerCls)

	// make_server(host, port, app, server_class=WSGIServer, handler_class=WSGIRequestHandler)
	m.Dict.SetStr("make_server", &object.BuiltinFunc{Name: "make_server", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "make_server requires host, port, app")
		}
		host := "127.0.0.1"
		if s, ok := a[0].(*object.Str); ok {
			host = s.V
		}
		port := int64(8080)
		if n, ok := a[1].(*object.Int); ok {
			port = n.Int64()
		}
		app := a[2]
		addr := fmt.Sprintf("%s:%d", host, port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return nil, object.Errorf(i.osErr, "make_server: %s", err.Error())
		}
		srv := &object.Instance{Class: wsgiServerCls, Dict: object.NewDict()}
		srv.Dict.SetStr("_app", app)
		srv.Dict.SetStr("server_name", &object.Str{V: host})
		srv.Dict.SetStr("server_port", object.IntFromInt64(port))
		wsgiServerMap.Store(srv, ln)
		return srv, nil
	}})

	// demo_app(environ, start_response) -> [bytes]
	m.Dict.SetStr("demo_app", &object.BuiltinFunc{Name: "demo_app", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "demo_app requires environ and start_response")
		}
		interp := ii.(*Interp)
		hdrs := &object.List{V: []object.Object{
			&object.Tuple{V: []object.Object{
				&object.Str{V: "Content-type"},
				&object.Str{V: "text/plain; charset=utf-8"},
			}},
		}}
		if _, err := interp.callObject(a[1], []object.Object{&object.Str{V: "200 OK"}, hdrs}, nil); err != nil {
			return nil, err
		}
		body := "Hello world!\n"
		return &object.List{V: []object.Object{&object.Bytes{V: []byte(body)}}}, nil
	}})

	return m
}

// ── wsgiref.handlers ──────────────────────────────────────────────────────────

func (i *Interp) buildWsgirefHandlers() *object.Module {
	m := &object.Module{Name: "wsgiref.handlers", Dict: object.NewDict()}

	// BaseHandler class (abstract stub)
	baseHandlerCls := &object.Class{Name: "BaseHandler", Dict: object.NewDict()}
	m.Dict.SetStr("BaseHandler", baseHandlerCls)

	// SimpleHandler class
	shCls := &object.Class{Name: "SimpleHandler", Dict: object.NewDict()}

	shCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "SimpleHandler.__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 5 {
			return nil, object.Errorf(i.typeErr, "SimpleHandler requires stdin, stdout, stderr, environ")
		}
		self := a[0].(*object.Instance)
		self.Dict.SetStr("stdin", a[1])
		self.Dict.SetStr("stdout", a[2])
		self.Dict.SetStr("stderr", a[3])
		self.Dict.SetStr("environ", a[4])
		return object.None, nil
	}})

	shCls.Dict.SetStr("run", &object.BuiltinFunc{Name: "SimpleHandler.run", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "run() requires application argument")
		}
		self := a[0].(*object.Instance)
		app := a[1]
		interp := ii.(*Interp)

		stdoutObj, _ := self.Dict.GetStr("stdout")
		environObj, _ := self.Dict.GetStr("environ")
		if environObj == nil {
			environObj = object.NewDict()
		}

		var responseStatus string
		var responseHeaders []object.Object

		startResponse := &object.BuiltinFunc{Name: "start_response", Call: func(_ any, sr []object.Object, _ *object.Dict) (object.Object, error) {
			if len(sr) > 0 {
				if s, ok := sr[0].(*object.Str); ok {
					responseStatus = s.V
				}
			}
			if len(sr) > 1 {
				if lst, ok := sr[1].(*object.List); ok {
					responseHeaders = lst.V
				}
			}
			return &object.BuiltinFunc{Name: "_write", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}}, nil
		}}

		result, err := interp.callObject(app, []object.Object{environObj, startResponse}, nil)
		if err != nil {
			return nil, err
		}

		if stdoutObj == nil || stdoutObj == object.None {
			return object.None, nil
		}
		writeFn, werr := interp.getAttr(stdoutObj, "write")
		if werr != nil {
			return object.None, nil
		}
		write := func(data []byte) {
			interp.callObject(writeFn, []object.Object{&object.Bytes{V: data}}, nil) //nolint
		}

		write([]byte("HTTP/1.0 " + responseStatus + "\r\n"))
		for _, hdrObj := range responseHeaders {
			if tup, ok := hdrObj.(*object.Tuple); ok && len(tup.V) == 2 {
				n, v := "", ""
				if s, ok2 := tup.V[0].(*object.Str); ok2 {
					n = s.V
				}
				if s, ok2 := tup.V[1].(*object.Str); ok2 {
					v = s.V
				}
				if n != "" {
					write([]byte(n + ": " + v + "\r\n"))
				}
			}
		}
		write([]byte("\r\n"))

		if lst, ok := result.(*object.List); ok {
			for _, chunk := range lst.V {
				if b, ok2 := chunk.(*object.Bytes); ok2 {
					write(b.V)
				}
			}
		}

		return object.None, nil
	}})

	m.Dict.SetStr("SimpleHandler", shCls)

	// BaseCGIHandler class
	baseCGIHandlerCls := &object.Class{Name: "BaseCGIHandler", Dict: object.NewDict()}
	m.Dict.SetStr("BaseCGIHandler", baseCGIHandlerCls)

	// CGIHandler class
	cgiHandlerCls := &object.Class{Name: "CGIHandler", Dict: object.NewDict()}
	m.Dict.SetStr("CGIHandler", cgiHandlerCls)

	// IISCGIHandler class
	iisCGIHandlerCls := &object.Class{Name: "IISCGIHandler", Dict: object.NewDict()}
	m.Dict.SetStr("IISCGIHandler", iisCGIHandlerCls)

	// read_environ() -> dict
	m.Dict.SetStr("read_environ", &object.BuiltinFunc{Name: "read_environ", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewDict(), nil
	}})

	return m
}

// ── wsgiref.validate ──────────────────────────────────────────────────────────

func (i *Interp) buildWsgirefValidate() *object.Module {
	m := &object.Module{Name: "wsgiref.validate", Dict: object.NewDict()}

	// validator(application) -> wrapped_app
	m.Dict.SetStr("validator", &object.BuiltinFunc{Name: "validator", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "validator requires an application argument")
		}
		app := a[0]
		return &object.BuiltinFunc{Name: "validator_app", Call: func(ii2 any, a2 []object.Object, kw2 *object.Dict) (object.Object, error) {
			interp := ii2.(*Interp)
			if len(a2) < 2 {
				return nil, object.Errorf(i.typeErr, "wsgi app requires environ and start_response")
			}
			return interp.callObject(app, a2, kw2)
		}}, nil
	}})

	return m
}

// ── wsgiref.types ─────────────────────────────────────────────────────────────

func (i *Interp) buildWsgirefTypes() *object.Module {
	m := &object.Module{Name: "wsgiref.types", Dict: object.NewDict()}
	// Type aliases (Python 3.11+); runtime values are None.
	for _, name := range []string{"WSGIEnvironment", "WSGIApplication", "StartResponse", "InputStream", "ErrorStream", "FileWrapper"} {
		m.Dict.SetStr(name, object.None)
	}
	return m
}

package vm

import (
	"fmt"
	"strings"

	"github.com/tamnd/goipy/object"
)

// buildHttpClient constructs the http.client module matching CPython 3.14's
// API surface. No real TCP connections are made; connection methods are stubs.
func (i *Interp) buildHttpClient() *object.Module {
	m := &object.Module{Name: "http.client", Dict: object.NewDict()}

	// ── Constants ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("HTTP_PORT", object.NewInt(80))
	m.Dict.SetStr("HTTPS_PORT", object.NewInt(443))
	m.Dict.SetStr("_MAXLINE", object.NewInt(65536))
	m.Dict.SetStr("_MAXHEADERS", object.NewInt(100))

	// ── Exception hierarchy ───────────────────────────────────────────────────

	mkExc := func(name string, bases ...*object.Class) *object.Class {
		return &object.Class{Name: name, Dict: object.NewDict(), Bases: bases}
	}

	httpExcCls := mkExc("HTTPException", i.exception)
	m.Dict.SetStr("HTTPException", httpExcCls)

	notConnectedCls := mkExc("NotConnected", httpExcCls)
	m.Dict.SetStr("NotConnected", notConnectedCls)

	invalidURLCls := mkExc("InvalidURL", httpExcCls)
	m.Dict.SetStr("InvalidURL", invalidURLCls)

	// UnknownProtocol — .version is Args[0]
	unknownProtoCls := mkExc("UnknownProtocol", httpExcCls)
	unknownProtoCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		name := object.Str_(a[1])
		if name == "version" {
			if exc.Args != nil && len(exc.Args.V) > 0 {
				return exc.Args.V[0], nil
			}
			return &object.Str{V: ""}, nil
		}
		return nil, object.Errorf(i.attrErr, "'UnknownProtocol' object has no attribute '%s'", name)
	}})
	m.Dict.SetStr("UnknownProtocol", unknownProtoCls)

	unknownTransferEncCls := mkExc("UnknownTransferEncoding", httpExcCls)
	m.Dict.SetStr("UnknownTransferEncoding", unknownTransferEncCls)

	unimplFileCls := mkExc("UnimplementedFileMode", httpExcCls)
	m.Dict.SetStr("UnimplementedFileMode", unimplFileCls)

	// IncompleteRead: .partial = Args[0], .expected = Args[1] or None
	incompleteReadCls := mkExc("IncompleteRead", httpExcCls)
	incompleteReadCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		name := object.Str_(a[1])
		switch name {
		case "partial":
			if exc.Args != nil && len(exc.Args.V) > 0 {
				return exc.Args.V[0], nil
			}
			return &object.Bytes{V: nil}, nil
		case "expected":
			if exc.Args != nil && len(exc.Args.V) > 1 {
				return exc.Args.V[1], nil
			}
			return object.None, nil
		}
		return nil, object.Errorf(i.attrErr, "'IncompleteRead' object has no attribute '%s'", name)
	}})
	incompleteReadRepr := func(exc *object.Exception) string {
		var partial object.Object
		if exc.Args != nil && len(exc.Args.V) > 0 {
			partial = exc.Args.V[0]
		}
		var expected object.Object = object.None
		if exc.Args != nil && len(exc.Args.V) > 1 {
			expected = exc.Args.V[1]
		}
		partialLen := int64(0)
		if b, ok2 := partial.(*object.Bytes); ok2 {
			partialLen = int64(len(b.V))
		} else if s, ok2 := partial.(*object.Str); ok2 {
			partialLen = int64(len(s.V))
		} else if partial != nil {
			partialLen, _ = toInt64(partial)
		}
		if expected == nil || expected == object.None {
			return fmt.Sprintf("IncompleteRead(%d bytes read)", partialLen)
		}
		expN, _ := toInt64(expected)
		return fmt.Sprintf("IncompleteRead(%d bytes read, %d more expected)", partialLen, expN)
	}
	incompleteReadCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "IncompleteRead()"}, nil
		}
		exc, ok := a[0].(*object.Exception)
		if !ok {
			return &object.Str{V: "IncompleteRead()"}, nil
		}
		return &object.Str{V: incompleteReadRepr(exc)}, nil
	}})
	// CPython: IncompleteRead.__str__ = object.__str__ which falls back to repr
	incompleteReadCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "IncompleteRead()"}, nil
		}
		exc, ok := a[0].(*object.Exception)
		if !ok {
			return &object.Str{V: "IncompleteRead()"}, nil
		}
		return &object.Str{V: incompleteReadRepr(exc)}, nil
	}})
	m.Dict.SetStr("IncompleteRead", incompleteReadCls)

	improperStateCls := mkExc("ImproperConnectionState", httpExcCls)
	m.Dict.SetStr("ImproperConnectionState", improperStateCls)

	cannotSendReqCls := mkExc("CannotSendRequest", improperStateCls)
	m.Dict.SetStr("CannotSendRequest", cannotSendReqCls)

	cannotSendHdrCls := mkExc("CannotSendHeader", improperStateCls)
	m.Dict.SetStr("CannotSendHeader", cannotSendHdrCls)

	responseNotReadyCls := mkExc("ResponseNotReady", improperStateCls)
	m.Dict.SetStr("ResponseNotReady", responseNotReadyCls)

	// BadStatusLine — .line is Args[0] (empty str normalized to "''")
	badStatusLineCls := mkExc("BadStatusLine", httpExcCls)
	badStatusLineCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		exc := a[0].(*object.Exception)
		name := object.Str_(a[1])
		if name == "line" {
			if exc.Args != nil && len(exc.Args.V) > 0 {
				line := exc.Args.V[0]
				if s, ok := line.(*object.Str); ok && s.V == "" {
					return &object.Str{V: "''"}, nil
				}
				return line, nil
			}
			return &object.Str{V: "''"}, nil
		}
		return nil, object.Errorf(i.attrErr, "'BadStatusLine' object has no attribute '%s'", name)
	}})
	m.Dict.SetStr("BadStatusLine", badStatusLineCls)

	// LineTooLong — __init__(line_type) is bypassed; str() computes message from Args[0]
	lineTooLongCls := mkExc("LineTooLong", httpExcCls)
	lineTooLongCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "got more than 65536 bytes when reading "}, nil
		}
		exc, ok := a[0].(*object.Exception)
		if !ok {
			return &object.Str{V: "got more than 65536 bytes when reading "}, nil
		}
		lineType := ""
		if exc.Args != nil && len(exc.Args.V) > 0 {
			if s, ok2 := exc.Args.V[0].(*object.Str); ok2 {
				lineType = s.V
			}
		}
		return &object.Str{V: fmt.Sprintf("got more than 65536 bytes when reading %s", lineType)}, nil
	}})
	m.Dict.SetStr("LineTooLong", lineTooLongCls)

	// RemoteDisconnected(ConnectionResetError, BadStatusLine)
	// Look up ConnectionResetError from builtins.
	var connResetErrCls *object.Class
	if v, ok := i.Builtins.GetStr("ConnectionResetError"); ok {
		if cls, ok2 := v.(*object.Class); ok2 {
			connResetErrCls = cls
		}
	}
	remoteDisconBases := []*object.Class{badStatusLineCls}
	if connResetErrCls != nil {
		remoteDisconBases = []*object.Class{connResetErrCls, badStatusLineCls}
	}
	remoteDisconCls := &object.Class{Name: "RemoteDisconnected", Dict: object.NewDict(), Bases: remoteDisconBases}
	// RemoteDisconnected: str() returns the message (first arg), line is "''"
	remoteDisconCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		name := object.Str_(a[1])
		if name == "line" {
			return &object.Str{V: "''"}, nil
		}
		return nil, object.Errorf(i.attrErr, "'RemoteDisconnected' object has no attribute '%s'", name)
	}})
	m.Dict.SetStr("RemoteDisconnected", remoteDisconCls)

	// ── HTTPMessage ───────────────────────────────────────────────────────────
	// A simple case-insensitive ordered multi-value headers container.

	httpMsgCls := &object.Class{Name: "HTTPMessage", Dict: object.NewDict()}

	type headerEntry struct {
		key string // original case
		val string
	}

	// makeHTTPMessage creates a new HTTPMessage instance with the given headers.
	makeHTTPMessage := func(entries []headerEntry) *object.Instance {
		inst := &object.Instance{Class: httpMsgCls, Dict: object.NewDict()}
		// Store headers as a list of [key, val] pairs in "_headers_" slot.
		pairs := make([]object.Object, 0, len(entries))
		for _, e := range entries {
			pair := &object.Tuple{V: []object.Object{&object.Str{V: e.key}, &object.Str{V: e.val}}}
			pairs = append(pairs, pair)
		}
		inst.Dict.SetStr("_headers_", &object.List{V: pairs})
		return inst
	}

	getMsgHeaders := func(inst *object.Instance) []headerEntry {
		raw, ok := inst.Dict.GetStr("_headers_")
		if !ok {
			return nil
		}
		lst, ok := raw.(*object.List)
		if !ok {
			return nil
		}
		entries := make([]headerEntry, 0, len(lst.V))
		for _, p := range lst.V {
			if t, ok2 := p.(*object.Tuple); ok2 && len(t.V) == 2 {
				k, _ := t.V[0].(*object.Str)
				v, _ := t.V[1].(*object.Str)
				if k != nil && v != nil {
					entries = append(entries, headerEntry{k.V, v.V})
				}
			}
		}
		return entries
	}

	httpMsgGet := func(inst *object.Instance, name string, failobj object.Object) object.Object {
		lower := strings.ToLower(name)
		for _, e := range getMsgHeaders(inst) {
			if strings.ToLower(e.key) == lower {
				return &object.Str{V: e.val}
			}
		}
		return failobj
	}

	httpMsgCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.keyErr, "missing key")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "not an HTTPMessage")
		}
		nameStr, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "key must be string")
		}
		result := httpMsgGet(inst, nameStr.V, nil)
		if result == nil {
			return nil, object.Errorf(i.keyErr, "%s", nameStr.V)
		}
		return result, nil
	}})

	httpMsgCls.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		key, ok1 := a[1].(*object.Str)
		val, ok2 := a[2].(*object.Str)
		if !ok1 || !ok2 {
			return object.None, nil
		}
		// Replace existing or append
		raw, exists := inst.Dict.GetStr("_headers_")
		if !exists {
			inst.Dict.SetStr("_headers_", &object.List{V: []object.Object{
				&object.Tuple{V: []object.Object{key, val}},
			}})
			return object.None, nil
		}
		lst := raw.(*object.List)
		lower := strings.ToLower(key.V)
		for idx, p := range lst.V {
			if t, ok3 := p.(*object.Tuple); ok3 && len(t.V) == 2 {
				if k, ok4 := t.V[0].(*object.Str); ok4 && strings.ToLower(k.V) == lower {
					lst.V[idx] = &object.Tuple{V: []object.Object{key, val}}
					return object.None, nil
				}
			}
		}
		lst.V = append(lst.V, &object.Tuple{V: []object.Object{key, val}})
		return object.None, nil
	}})

	httpMsgCls.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.False, nil
		}
		nameStr, ok := a[1].(*object.Str)
		if !ok {
			return object.False, nil
		}
		result := httpMsgGet(inst, nameStr.V, nil)
		return &object.Bool{V: result != nil}, nil
	}})

	httpMsgCls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		return object.NewInt(int64(len(getMsgHeaders(inst)))), nil
	}})

	httpMsgCls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		nameStr, ok := a[1].(*object.Str)
		if !ok {
			return object.None, nil
		}
		failobj := object.Object(object.None)
		if len(a) > 2 {
			failobj = a[2]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("failobj"); ok2 {
				failobj = v
			}
		}
		result := httpMsgGet(inst, nameStr.V, failobj)
		return result, nil
	}})

	httpMsgCls.Dict.SetStr("get_all", &object.BuiltinFunc{Name: "get_all", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		nameStr, ok := a[1].(*object.Str)
		if !ok {
			return object.None, nil
		}
		failobj := object.Object(object.None)
		if len(a) > 2 {
			failobj = a[2]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("failobj"); ok2 {
				failobj = v
			}
		}
		lower := strings.ToLower(nameStr.V)
		var results []object.Object
		for _, e := range getMsgHeaders(inst) {
			if strings.ToLower(e.key) == lower {
				results = append(results, &object.Str{V: e.val})
			}
		}
		if len(results) == 0 {
			return failobj, nil
		}
		return &object.List{V: results}, nil
	}})

	httpMsgCls.Dict.SetStr("get_content_type", &object.BuiltinFunc{Name: "get_content_type", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "text/plain"}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "text/plain"}, nil
		}
		ct := httpMsgGet(inst, "Content-Type", nil)
		if ct == nil {
			return &object.Str{V: "text/plain"}, nil
		}
		s := ct.(*object.Str).V
		if idx := strings.IndexByte(s, ';'); idx >= 0 {
			s = strings.TrimSpace(s[:idx])
		}
		return &object.Str{V: s}, nil
	}})

	httpMsgCls.Dict.SetStr("keys", &object.BuiltinFunc{Name: "keys", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.List{V: nil}, nil
		}
		entries := getMsgHeaders(inst)
		keys := make([]object.Object, len(entries))
		for j, e := range entries {
			keys[j] = &object.Str{V: e.key}
		}
		return &object.List{V: keys}, nil
	}})

	httpMsgCls.Dict.SetStr("values", &object.BuiltinFunc{Name: "values", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.List{V: nil}, nil
		}
		entries := getMsgHeaders(inst)
		vals := make([]object.Object, len(entries))
		for j, e := range entries {
			vals[j] = &object.Str{V: e.val}
		}
		return &object.List{V: vals}, nil
	}})

	httpMsgCls.Dict.SetStr("items", &object.BuiltinFunc{Name: "items", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.List{V: nil}, nil
		}
		entries := getMsgHeaders(inst)
		items := make([]object.Object, len(entries))
		for j, e := range entries {
			items[j] = &object.Tuple{V: []object.Object{&object.Str{V: e.key}, &object.Str{V: e.val}}}
		}
		return &object.List{V: items}, nil
	}})

	httpMsgCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: ""}, nil
		}
		var sb strings.Builder
		for _, e := range getMsgHeaders(inst) {
			sb.WriteString(e.key)
			sb.WriteString(": ")
			sb.WriteString(e.val)
			sb.WriteString("\n")
		}
		return &object.Str{V: sb.String()}, nil
	}})

	m.Dict.SetStr("HTTPMessage", httpMsgCls)

	// ── parse_headers(fp) ─────────────────────────────────────────────────────

	m.Dict.SetStr("parse_headers", &object.BuiltinFunc{Name: "parse_headers", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return makeHTTPMessage(nil), nil
		}
		fp := a[0]
		interp := ii.(*Interp)

		// Read lines from fp until blank line or EOF.
		var rawLines []string
		readMethod, err := interp.getAttr(fp, "read")
		if err != nil {
			// Try readline instead
			readMethod = nil
		}
		_ = readMethod

		readlineFn, err2 := interp.getAttr(fp, "readline")
		if err2 != nil {
			return makeHTTPMessage(nil), nil
		}
		for {
			lineObj, err3 := interp.callObject(readlineFn, nil, nil)
			if err3 != nil {
				break
			}
			var line string
			switch lv := lineObj.(type) {
			case *object.Bytes:
				line = strings.TrimRight(string(lv.V), "\r\n")
			case *object.Str:
				line = strings.TrimRight(lv.V, "\r\n")
			default:
				break
			}
			if line == "" {
				break
			}
			rawLines = append(rawLines, line)
		}

		var entries []headerEntry
		for _, line := range rawLines {
			if idx := strings.IndexByte(line, ':'); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				val := strings.TrimSpace(line[idx+1:])
				entries = append(entries, headerEntry{key, val})
			}
		}
		return makeHTTPMessage(entries), nil
	}})

	// ── HTTPConnection ────────────────────────────────────────────────────────

	httpConnCls := &object.Class{Name: "HTTPConnection", Dict: object.NewDict()}

	httpConnCls.Dict.SetStr("default_port", object.NewInt(80))
	httpConnCls.Dict.SetStr("debuglevel", object.NewInt(0))
	httpConnCls.Dict.SetStr("auto_open", object.NewInt(1))

	httpConnCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		host := ""
		if s, ok := a[1].(*object.Str); ok {
			host = s.V
		}
		port := int64(80)
		timeout := object.Object(object.None)

		if len(a) > 2 && a[2] != object.None {
			if n, ok := toInt64(a[2]); ok {
				port = n
			}
		} else if kw != nil {
			if pv, ok := kw.GetStr("port"); ok && pv != object.None {
				if n, ok2 := toInt64(pv); ok2 {
					port = n
				}
			}
		}
		if len(a) > 3 {
			timeout = a[3]
		} else if kw != nil {
			if tv, ok := kw.GetStr("timeout"); ok {
				timeout = tv
			}
		}

		inst.Dict.SetStr("host", &object.Str{V: host})
		inst.Dict.SetStr("port", object.NewInt(port))
		inst.Dict.SetStr("timeout", timeout)
		inst.Dict.SetStr("debuglevel", object.NewInt(0))
		return object.None, nil
	}})

	noopMethod := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}
	}

	httpConnCls.Dict.SetStr("connect", noopMethod("connect"))
	httpConnCls.Dict.SetStr("close", noopMethod("close"))
	httpConnCls.Dict.SetStr("send", noopMethod("send"))
	httpConnCls.Dict.SetStr("request", noopMethod("request"))
	httpConnCls.Dict.SetStr("putheader", noopMethod("putheader"))
	httpConnCls.Dict.SetStr("putrequest", noopMethod("putrequest"))
	httpConnCls.Dict.SetStr("endheaders", noopMethod("endheaders"))

	httpConnCls.Dict.SetStr("set_debuglevel", &object.BuiltinFunc{Name: "set_debuglevel", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		inst.Dict.SetStr("debuglevel", a[1])
		return object.None, nil
	}})

	httpConnCls.Dict.SetStr("set_tunnel", &object.BuiltinFunc{Name: "set_tunnel", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		host := ""
		if s, ok2 := a[1].(*object.Str); ok2 {
			host = s.V
		}
		inst.Dict.SetStr("_tunnel_host", &object.Str{V: host})
		tunnelPort := object.Object(object.None)
		if len(a) > 2 {
			tunnelPort = a[2]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("port"); ok2 {
				tunnelPort = v
			}
		}
		inst.Dict.SetStr("_tunnel_port", tunnelPort)
		tunnelHeaders := object.Object(object.None)
		if len(a) > 3 {
			tunnelHeaders = a[3]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("headers"); ok2 {
				tunnelHeaders = v
			}
		}
		inst.Dict.SetStr("_tunnel_headers", tunnelHeaders)
		return object.None, nil
	}})

	httpConnCls.Dict.SetStr("getresponse", &object.BuiltinFunc{Name: "getresponse", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return nil, object.NewException(responseNotReadyCls, "")
	}})

	m.Dict.SetStr("HTTPConnection", httpConnCls)

	// ── HTTPSConnection ───────────────────────────────────────────────────────

	httpsSConnCls := &object.Class{Name: "HTTPSConnection", Bases: []*object.Class{httpConnCls}, Dict: object.NewDict()}
	httpsSConnCls.Dict.SetStr("default_port", object.NewInt(443))
	// __init__ inherited from HTTPConnection but default port is 443
	httpsSConnCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		host := ""
		if s, ok := a[1].(*object.Str); ok {
			host = s.V
		}
		port := int64(443)
		timeout := object.Object(object.None)

		if len(a) > 2 && a[2] != object.None {
			if n, ok := toInt64(a[2]); ok {
				port = n
			}
		} else if kw != nil {
			if pv, ok := kw.GetStr("port"); ok && pv != object.None {
				if n, ok2 := toInt64(pv); ok2 {
					port = n
				}
			}
		}
		if len(a) > 3 {
			timeout = a[3]
		} else if kw != nil {
			if tv, ok := kw.GetStr("timeout"); ok {
				timeout = tv
			}
		}

		inst.Dict.SetStr("host", &object.Str{V: host})
		inst.Dict.SetStr("port", object.NewInt(port))
		inst.Dict.SetStr("timeout", timeout)
		inst.Dict.SetStr("debuglevel", object.NewInt(0))
		return object.None, nil
	}})
	httpsSConnCls.Dict.SetStr("connect", noopMethod("connect"))
	httpsSConnCls.Dict.SetStr("close", noopMethod("close"))
	httpsSConnCls.Dict.SetStr("send", noopMethod("send"))
	httpsSConnCls.Dict.SetStr("request", noopMethod("request"))
	httpsSConnCls.Dict.SetStr("putheader", noopMethod("putheader"))
	httpsSConnCls.Dict.SetStr("putrequest", noopMethod("putrequest"))
	httpsSConnCls.Dict.SetStr("endheaders", noopMethod("endheaders"))
	httpsSConnCls.Dict.SetStr("set_debuglevel", &object.BuiltinFunc{Name: "set_debuglevel", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		inst.Dict.SetStr("debuglevel", a[1])
		return object.None, nil
	}})
	if stFn, ok := httpConnCls.Dict.GetStr("set_tunnel"); ok {
		httpsSConnCls.Dict.SetStr("set_tunnel", stFn)
	}
	httpsSConnCls.Dict.SetStr("getresponse", &object.BuiltinFunc{Name: "getresponse", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return nil, object.NewException(responseNotReadyCls, "")
	}})
	m.Dict.SetStr("HTTPSConnection", httpsSConnCls)

	// ── responses dict ────────────────────────────────────────────────────────
	// Maps int status codes → phrase strings (same set as http.HTTPStatus).

	responsesDict := object.NewDict()
	type statusEntry struct {
		code   int
		phrase string
	}
	statusList := []statusEntry{
		{100, "Continue"}, {101, "Switching Protocols"}, {102, "Processing"}, {103, "Early Hints"},
		{200, "OK"}, {201, "Created"}, {202, "Accepted"}, {203, "Non-Authoritative Information"},
		{204, "No Content"}, {205, "Reset Content"}, {206, "Partial Content"},
		{207, "Multi-Status"}, {208, "Already Reported"}, {226, "IM Used"},
		{300, "Multiple Choices"}, {301, "Moved Permanently"}, {302, "Found"},
		{303, "See Other"}, {304, "Not Modified"}, {305, "Use Proxy"},
		{307, "Temporary Redirect"}, {308, "Permanent Redirect"},
		{400, "Bad Request"}, {401, "Unauthorized"}, {402, "Payment Required"},
		{403, "Forbidden"}, {404, "Not Found"}, {405, "Method Not Allowed"},
		{406, "Not Acceptable"}, {407, "Proxy Authentication Required"},
		{408, "Request Timeout"}, {409, "Conflict"}, {410, "Gone"},
		{411, "Length Required"}, {412, "Precondition Failed"},
		{413, "Request Entity Too Large"}, {414, "Request-URI Too Long"},
		{415, "Unsupported Media Type"}, {416, "Requested Range Not Satisfiable"},
		{417, "Expectation Failed"}, {418, "I'm a Teapot"},
		{421, "Misdirected Request"}, {422, "Unprocessable Entity"},
		{423, "Locked"}, {424, "Failed Dependency"}, {425, "Too Early"},
		{426, "Upgrade Required"}, {428, "Precondition Required"},
		{429, "Too Many Requests"}, {431, "Request Header Fields Too Large"},
		{451, "Unavailable For Legal Reasons"},
		{500, "Internal Server Error"}, {501, "Not Implemented"},
		{502, "Bad Gateway"}, {503, "Service Unavailable"},
		{504, "Gateway Timeout"}, {505, "HTTP Version Not Supported"},
		{506, "Variant Also Negotiates"}, {507, "Insufficient Storage"},
		{508, "Loop Detected"}, {510, "Not Extended"},
		{511, "Network Authentication Required"},
	}
	for _, s := range statusList {
		responsesDict.Set(object.NewInt(int64(s.code)), &object.Str{V: s.phrase})
	}
	m.Dict.SetStr("responses", responsesDict)

	// ── Status code integer re-exports ────────────────────────────────────────
	// All http.HTTPStatus names exported as plain ints (CPython compatibility).

	type intExport struct {
		name string
		code int
	}
	intExports := []intExport{
		{"CONTINUE", 100}, {"SWITCHING_PROTOCOLS", 101}, {"PROCESSING", 102}, {"EARLY_HINTS", 103},
		{"OK", 200}, {"CREATED", 201}, {"ACCEPTED", 202}, {"NON_AUTHORITATIVE_INFORMATION", 203},
		{"NO_CONTENT", 204}, {"RESET_CONTENT", 205}, {"PARTIAL_CONTENT", 206},
		{"MULTI_STATUS", 207}, {"ALREADY_REPORTED", 208}, {"IM_USED", 226},
		{"MULTIPLE_CHOICES", 300}, {"MOVED_PERMANENTLY", 301}, {"FOUND", 302},
		{"SEE_OTHER", 303}, {"NOT_MODIFIED", 304}, {"USE_PROXY", 305},
		{"TEMPORARY_REDIRECT", 307}, {"PERMANENT_REDIRECT", 308},
		{"BAD_REQUEST", 400}, {"UNAUTHORIZED", 401}, {"PAYMENT_REQUIRED", 402},
		{"FORBIDDEN", 403}, {"NOT_FOUND", 404}, {"METHOD_NOT_ALLOWED", 405},
		{"NOT_ACCEPTABLE", 406}, {"PROXY_AUTHENTICATION_REQUIRED", 407},
		{"REQUEST_TIMEOUT", 408}, {"CONFLICT", 409}, {"GONE", 410},
		{"LENGTH_REQUIRED", 411}, {"PRECONDITION_FAILED", 412},
		{"REQUEST_ENTITY_TOO_LARGE", 413}, {"REQUEST_URI_TOO_LONG", 414},
		{"UNSUPPORTED_MEDIA_TYPE", 415}, {"REQUESTED_RANGE_NOT_SATISFIABLE", 416},
		{"RANGE_NOT_SATISFIABLE", 416},
		{"EXPECTATION_FAILED", 417}, {"IM_A_TEAPOT", 418},
		{"MISDIRECTED_REQUEST", 421}, {"UNPROCESSABLE_ENTITY", 422},
		{"UNPROCESSABLE_CONTENT", 422},
		{"CONTENT_TOO_LARGE", 413},
		{"URI_TOO_LONG", 414},
		{"LOCKED", 423}, {"FAILED_DEPENDENCY", 424}, {"TOO_EARLY", 425},
		{"UPGRADE_REQUIRED", 426}, {"PRECONDITION_REQUIRED", 428},
		{"TOO_MANY_REQUESTS", 429}, {"REQUEST_HEADER_FIELDS_TOO_LARGE", 431},
		{"UNAVAILABLE_FOR_LEGAL_REASONS", 451},
		{"INTERNAL_SERVER_ERROR", 500}, {"NOT_IMPLEMENTED", 501},
		{"BAD_GATEWAY", 502}, {"SERVICE_UNAVAILABLE", 503},
		{"GATEWAY_TIMEOUT", 504}, {"HTTP_VERSION_NOT_SUPPORTED", 505},
		{"VARIANT_ALSO_NEGOTIATES", 506}, {"INSUFFICIENT_STORAGE", 507},
		{"LOOP_DETECTED", 508}, {"NOT_EXTENDED", 510},
		{"NETWORK_AUTHENTICATION_REQUIRED", 511},
	}
	for _, e := range intExports {
		m.Dict.SetStr(e.name, object.NewInt(int64(e.code)))
	}

	// error = HTTPException (CPython compat alias)
	m.Dict.SetStr("error", httpExcCls)

	return m
}

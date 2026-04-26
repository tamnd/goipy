package vm

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildXmlrpcClient() *object.Module {
	m := &object.Module{Name: "xmlrpc.client", Dict: object.NewDict()}

	// ── Exception hierarchy ───────────────────────────────────────────────────
	// Exception subclasses are *object.Exception in goipy. Attrs are exposed
	// via __getattr__ reading exc.Args.

	mkExc := func(name string, bases ...*object.Class) *object.Class {
		return &object.Class{Name: name, Dict: object.NewDict(), Bases: bases}
	}
	errCls := mkExc("Error", i.exception)
	faultCls := mkExc("Fault", errCls)
	protocolErrCls := mkExc("ProtocolError", errCls)
	responseErrCls := mkExc("ResponseError", errCls)

	// Fault(faultCode, faultString) — Args[0]=faultCode, Args[1]=faultString
	faultCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		name := object.Str_(a[1])
		exc, ok := a[0].(*object.Exception)
		if !ok {
			return nil, object.Errorf(i.attrErr, "'Fault' has no attribute '%s'", name)
		}
		switch name {
		case "faultCode":
			if exc.Args != nil && len(exc.Args.V) > 0 {
				return exc.Args.V[0], nil
			}
			return object.NewInt(0), nil
		case "faultString":
			if exc.Args != nil && len(exc.Args.V) > 1 {
				return exc.Args.V[1], nil
			}
			return &object.Str{V: ""}, nil
		}
		return nil, object.Errorf(i.attrErr, "'Fault' object has no attribute '%s'", name)
	}})

	// ProtocolError(url, errcode, errmsg, headers)
	protocolErrCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		name := object.Str_(a[1])
		exc, ok := a[0].(*object.Exception)
		if !ok {
			return nil, object.Errorf(i.attrErr, "'ProtocolError' has no attribute '%s'", name)
		}
		switch name {
		case "url":
			if exc.Args != nil && len(exc.Args.V) > 0 {
				return exc.Args.V[0], nil
			}
			return &object.Str{V: ""}, nil
		case "errcode":
			if exc.Args != nil && len(exc.Args.V) > 1 {
				return exc.Args.V[1], nil
			}
			return object.NewInt(0), nil
		case "errmsg":
			if exc.Args != nil && len(exc.Args.V) > 2 {
				return exc.Args.V[2], nil
			}
			return &object.Str{V: ""}, nil
		case "headers":
			if exc.Args != nil && len(exc.Args.V) > 3 {
				return exc.Args.V[3], nil
			}
			return object.None, nil
		}
		return nil, object.Errorf(i.attrErr, "'ProtocolError' object has no attribute '%s'", name)
	}})

	m.Dict.SetStr("Error", errCls)
	m.Dict.SetStr("Fault", faultCls)
	m.Dict.SetStr("ProtocolError", protocolErrCls)
	m.Dict.SetStr("ResponseError", responseErrCls)

	// ── Constants ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("MAXINT", object.NewInt(2147483647))
	m.Dict.SetStr("MININT", object.NewInt(-2147483648))
	m.Dict.SetStr("PARSE_ERROR", object.NewInt(-32700))
	m.Dict.SetStr("SERVER_ERROR", object.NewInt(-32600))
	m.Dict.SetStr("APPLICATION_ERROR", object.NewInt(-32500))
	m.Dict.SetStr("SYSTEM_ERROR", object.NewInt(-32400))
	m.Dict.SetStr("TRANSPORT_ERROR", object.NewInt(-32300))
	m.Dict.SetStr("NOT_WELLFORMED_ERROR", object.NewInt(-32600))
	m.Dict.SetStr("UNSUPPORTED_ENCODING", object.NewInt(-32701))
	m.Dict.SetStr("INVALID_ENCODING_CHAR", object.NewInt(-32702))
	m.Dict.SetStr("INVALID_XMLRPC", object.NewInt(-32600))
	m.Dict.SetStr("METHOD_NOT_FOUND", object.NewInt(-32601))
	m.Dict.SetStr("INVALID_METHOD_PARAMS", object.NewInt(-32602))
	m.Dict.SetStr("INTERNAL_ERROR", object.NewInt(-32603))

	// ── DateTime ──────────────────────────────────────────────────────────────

	dateTimeCls := &object.Class{Name: "DateTime", Dict: object.NewDict()}
	dateTimeCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		val := &object.Str{V: ""}
		if len(a) >= 2 {
			val = &object.Str{V: object.Str_(a[1])}
		}
		inst.Dict.SetStr("value", val)
		return object.None, nil
	}})
	dateTimeCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		val := ""
		if len(a) >= 1 {
			if inst, ok := a[0].(*object.Instance); ok {
				if v, ok2 := inst.Dict.GetStr("value"); ok2 {
					val = object.Str_(v)
				}
			}
		}
		return &object.Str{V: fmt.Sprintf("<DateTime '%s' at 0x0>", val)}, nil
	}})
	m.Dict.SetStr("DateTime", dateTimeCls)

	// ── Binary ────────────────────────────────────────────────────────────────

	binaryCls := &object.Class{Name: "Binary", Dict: object.NewDict()}
	binaryCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		var data []byte
		if len(a) >= 2 {
			switch v := a[1].(type) {
			case *object.Bytes:
				data = v.V
			case *object.Str:
				data = []byte(v.V)
			}
		}
		inst.Dict.SetStr("data", &object.Bytes{V: data})

		inst.Dict.SetStr("decode", &object.BuiltinFunc{Name: "decode", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if len(args) < 1 {
				return object.None, nil
			}
			var raw []byte
			switch v := args[0].(type) {
			case *object.Bytes:
				raw = v.V
			case *object.Str:
				raw = []byte(v.V)
			}
			decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
			if err != nil {
				return nil, object.Errorf(i.valueErr, "invalid base64: %v", err)
			}
			inst.Dict.SetStr("data", &object.Bytes{V: decoded})
			return object.None, nil
		}})

		inst.Dict.SetStr("encode", &object.BuiltinFunc{Name: "encode", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			var raw []byte
			if d, ok2 := inst.Dict.GetStr("data"); ok2 {
				if b, ok3 := d.(*object.Bytes); ok3 {
					raw = b.V
				}
			}
			encoded := base64.StdEncoding.EncodeToString(raw)
			xml := "<value><base64>\n" + encoded + "\n</base64></value>"
			if len(args) >= 1 {
				// write to file-like out (args[0])
				if outInst, ok2 := args[0].(*object.Instance); ok2 {
					if writeFn, ok3 := outInst.Dict.GetStr("write"); ok3 {
						if bf, ok4 := writeFn.(*object.BuiltinFunc); ok4 {
							_, _ = bf.Call(nil, []object.Object{&object.Str{V: xml}}, nil)
						}
					}
				}
			}
			return object.None, nil
		}})

		return object.None, nil
	}})
	m.Dict.SetStr("Binary", binaryCls)

	// ── Boolean / boolean() ───────────────────────────────────────────────────

	booleanCls := &object.Class{Name: "Boolean", Dict: object.NewDict()}
	m.Dict.SetStr("Boolean", booleanCls)
	m.Dict.SetStr("boolean", &object.BuiltinFunc{Name: "boolean", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.False, nil
		}
		switch v := a[0].(type) {
		case *object.Bool:
			return v, nil
		case *object.Int:
			return object.BoolOf(v.V.Sign() != 0), nil
		case *object.NoneType:
			return object.False, nil
		}
		return object.True, nil
	}})

	// ── dumps / loads ─────────────────────────────────────────────────────────

	var marshalXmlrpc func(obj object.Object) string
	marshalXmlrpc = func(obj object.Object) string {
		switch v := obj.(type) {
		case *object.Bool:
			if v == object.True {
				return "<value><boolean>1</boolean></value>"
			}
			return "<value><boolean>0</boolean></value>"
		case *object.Int:
			return "<value><int>" + v.V.String() + "</int></value>"
		case *object.Float:
			return fmt.Sprintf("<value><double>%g</double></value>", v.V)
		case *object.Str:
			s := v.V
			s = strings.ReplaceAll(s, "&", "&amp;")
			s = strings.ReplaceAll(s, "<", "&lt;")
			s = strings.ReplaceAll(s, ">", "&gt;")
			return "<value><string>" + s + "</string></value>"
		case *object.NoneType:
			return "<value><nil/></value>"
		case *object.Bytes:
			return "<value><base64>" + base64.StdEncoding.EncodeToString(v.V) + "</base64></value>"
		case *object.List:
			var sb strings.Builder
			sb.WriteString("<value><array><data>")
			for _, item := range v.V {
				sb.WriteString(marshalXmlrpc(item))
			}
			sb.WriteString("</data></array></value>")
			return sb.String()
		case *object.Tuple:
			var sb strings.Builder
			sb.WriteString("<value><array><data>")
			for _, item := range v.V {
				sb.WriteString(marshalXmlrpc(item))
			}
			sb.WriteString("</data></array></value>")
			return sb.String()
		case *object.Dict:
			var sb strings.Builder
			sb.WriteString("<value><struct>")
			ks, vs := v.Items()
			for j, k := range ks {
				sb.WriteString("<member><name>")
				sb.WriteString(object.Str_(k))
				sb.WriteString("</name>")
				sb.WriteString(marshalXmlrpc(vs[j]))
				sb.WriteString("</member>")
			}
			sb.WriteString("</struct></value>")
			return sb.String()
		}
		return "<value><string>" + object.Str_(obj) + "</string></value>"
	}

	m.Dict.SetStr("dumps", &object.BuiltinFunc{Name: "dumps", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "dumps requires params")
		}
		methodname := ""
		methodresponse := false
		if len(a) >= 2 && a[1] != object.None {
			methodname = object.Str_(a[1])
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("methodname"); ok2 && v != object.None {
				methodname = object.Str_(v)
			}
			if v, ok2 := kw.GetStr("methodresponse"); ok2 {
				if b, ok3 := v.(*object.Bool); ok3 {
					methodresponse = b == object.True
				}
			}
		}

		var params []object.Object
		switch pv := a[0].(type) {
		case *object.Tuple:
			params = pv.V
		case *object.List:
			params = pv.V
		default:
			params = []object.Object{a[0]}
		}

		var sb strings.Builder
		sb.WriteString("<?xml version='1.0'?>\n")
		if methodresponse {
			sb.WriteString("<methodResponse>\n<params>\n")
		} else {
			sb.WriteString("<methodCall>\n")
			if methodname != "" {
				sb.WriteString("<methodName>")
				sb.WriteString(methodname)
				sb.WriteString("</methodName>\n")
			}
			sb.WriteString("<params>\n")
		}
		for _, p := range params {
			sb.WriteString("<param>\n")
			sb.WriteString(marshalXmlrpc(p))
			sb.WriteString("\n</param>\n")
		}
		sb.WriteString("</params>\n")
		if methodresponse {
			sb.WriteString("</methodResponse>\n")
		} else {
			sb.WriteString("</methodCall>\n")
		}
		return &object.Str{V: sb.String()}, nil
	}})

	m.Dict.SetStr("loads", &object.BuiltinFunc{Name: "loads", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "loads requires data")
		}
		data := object.Str_(a[0])
		methodname, params, err := parseXmlrpc(data)
		if err != nil {
			return nil, object.Errorf(faultCls, "%v", err)
		}
		var mn object.Object = object.None
		if methodname != "" {
			mn = &object.Str{V: methodname}
		}
		return &object.Tuple{V: []object.Object{
			&object.Tuple{V: params},
			mn,
		}}, nil
	}})

	// ── ServerProxy / Server / MultiCall / Transport ──────────────────────────

	transportCls := &object.Class{Name: "Transport", Dict: object.NewDict()}
	m.Dict.SetStr("Transport", transportCls)

	safeTransportCls := &object.Class{Name: "SafeTransport", Bases: []*object.Class{transportCls}, Dict: object.NewDict()}
	m.Dict.SetStr("SafeTransport", safeTransportCls)

	serverProxyCls := &object.Class{Name: "ServerProxy", Dict: object.NewDict()}
	serverProxyCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		if len(a) >= 2 {
			inst.Dict.SetStr("_ServerProxy__uri", a[1])
		}
		return object.None, nil
	}})
	m.Dict.SetStr("ServerProxy", serverProxyCls)
	m.Dict.SetStr("Server", serverProxyCls)

	multiCallCls := &object.Class{Name: "MultiCall", Dict: object.NewDict()}
	multiCallCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		if len(a) >= 2 {
			inst.Dict.SetStr("_server", a[1])
		}
		return object.None, nil
	}})
	m.Dict.SetStr("MultiCall", multiCallCls)

	return m
}

// parseXmlrpc parses a minimal XML-RPC methodCall/methodResponse document.
func parseXmlrpc(data string) (method string, params []object.Object, err error) {
	if idx := strings.Index(data, "<methodName>"); idx >= 0 {
		rest := data[idx+len("<methodName>"):]
		if end := strings.Index(rest, "</methodName>"); end >= 0 {
			method = rest[:end]
		}
	}
	remaining := data
	for {
		start := strings.Index(remaining, "<param>")
		if start < 0 {
			break
		}
		end := strings.Index(remaining[start:], "</param>")
		if end < 0 {
			break
		}
		block := remaining[start+len("<param>") : start+end]
		val, parseErr := parseXmlrpcValue(strings.TrimSpace(block))
		if parseErr != nil {
			err = parseErr
			return
		}
		params = append(params, val)
		remaining = remaining[start+end+len("</param>"):]
	}
	return
}

// parseXmlrpcValue parses a single <value>...</value> element.
func parseXmlrpcValue(s string) (object.Object, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "<value>") && strings.HasSuffix(s, "</value>") {
		s = strings.TrimSpace(s[len("<value>") : len(s)-len("</value>")])
	}
	if inner, ok := xmlCut(s, "<int>", "</int>"); ok {
		n, err2 := strconv.ParseInt(strings.TrimSpace(inner), 10, 64)
		if err2 != nil {
			return nil, fmt.Errorf("bad int: %v", err2)
		}
		return object.NewInt(n), nil
	}
	if inner, ok := xmlCut(s, "<i4>", "</i4>"); ok {
		n, err2 := strconv.ParseInt(strings.TrimSpace(inner), 10, 64)
		if err2 != nil {
			return nil, fmt.Errorf("bad i4: %v", err2)
		}
		return object.NewInt(n), nil
	}
	if inner, ok := xmlCut(s, "<boolean>", "</boolean>"); ok {
		return object.BoolOf(strings.TrimSpace(inner) == "1"), nil
	}
	if inner, ok := xmlCut(s, "<double>", "</double>"); ok {
		f, err2 := strconv.ParseFloat(strings.TrimSpace(inner), 64)
		if err2 != nil {
			return nil, fmt.Errorf("bad double: %v", err2)
		}
		return &object.Float{V: f}, nil
	}
	if inner, ok := xmlCut(s, "<string>", "</string>"); ok {
		inner = strings.ReplaceAll(inner, "&amp;", "&")
		inner = strings.ReplaceAll(inner, "&lt;", "<")
		inner = strings.ReplaceAll(inner, "&gt;", ">")
		return &object.Str{V: inner}, nil
	}
	if inner, ok := xmlCut(s, "<base64>", "</base64>"); ok {
		decoded, err2 := base64.StdEncoding.DecodeString(strings.TrimSpace(inner))
		if err2 != nil {
			return nil, fmt.Errorf("bad base64: %v", err2)
		}
		return &object.Bytes{V: decoded}, nil
	}
	if s == "<nil/>" || strings.HasPrefix(s, "<nil/>") {
		return object.None, nil
	}
	if inner, ok := xmlCut(s, "<array>", "</array>"); ok {
		if data, ok2 := xmlCut(inner, "<data>", "</data>"); ok2 {
			inner = data
		}
		var items []object.Object
		for {
			vs := strings.Index(inner, "<value>")
			if vs < 0 {
				break
			}
			ve := strings.Index(inner[vs:], "</value>")
			if ve < 0 {
				break
			}
			block := inner[vs : vs+ve+len("</value>")]
			val, err2 := parseXmlrpcValue(strings.TrimSpace(block))
			if err2 != nil {
				return nil, err2
			}
			items = append(items, val)
			inner = inner[vs+ve+len("</value>"):]
		}
		return &object.List{V: items}, nil
	}
	if inner, ok := xmlCut(s, "<struct>", "</struct>"); ok {
		d := object.NewDict()
		for {
			memberInner, ok2 := xmlCut(inner, "<member>", "</member>")
			if !ok2 {
				break
			}
			if name, ok3 := xmlCut(memberInner, "<name>", "</name>"); ok3 {
				afterName := memberInner[strings.Index(memberInner, "</name>")+len("</name>"):]
				val, err2 := parseXmlrpcValue(strings.TrimSpace(afterName))
				if err2 != nil {
					return nil, err2
				}
				d.SetStr(name, val)
			}
			inner = inner[strings.Index(inner, "</member>")+len("</member>"):]
		}
		return d, nil
	}
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return &object.Str{V: s}, nil
}

// xmlCut extracts the content between open and close tags (first occurrence).
func xmlCut(s, open, close string) (string, bool) {
	si := strings.Index(s, open)
	if si < 0 {
		return "", false
	}
	rest := s[si+len(open):]
	ei := strings.Index(rest, close)
	if ei < 0 {
		return "", false
	}
	return rest[:ei], true
}

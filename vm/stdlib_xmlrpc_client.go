package vm

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
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

	dtGetValue := func(a []object.Object, idx int) string {
		if len(a) > idx {
			if inst, ok := a[idx].(*object.Instance); ok {
				if v, ok2 := inst.Dict.GetStr("value"); ok2 {
					return object.Str_(v)
				}
			}
		}
		return ""
	}
	dateTimeCls.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(dtGetValue(a, 0) == dtGetValue(a, 1)), nil
	}})
	dateTimeCls.Dict.SetStr("__lt__", &object.BuiltinFunc{Name: "__lt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(dtGetValue(a, 0) < dtGetValue(a, 1)), nil
	}})
	dateTimeCls.Dict.SetStr("__le__", &object.BuiltinFunc{Name: "__le__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(dtGetValue(a, 0) <= dtGetValue(a, 1)), nil
	}})
	dateTimeCls.Dict.SetStr("__gt__", &object.BuiltinFunc{Name: "__gt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(dtGetValue(a, 0) > dtGetValue(a, 1)), nil
	}})
	dateTimeCls.Dict.SetStr("__ge__", &object.BuiltinFunc{Name: "__ge__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(dtGetValue(a, 0) >= dtGetValue(a, 1)), nil
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

		inst.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			// callInstanceDunder passes only extra args (not self), so args[0] = other
			if len(args) < 1 {
				return object.False, nil
			}
			var selfData, otherData []byte
			if d, ok2 := inst.Dict.GetStr("data"); ok2 {
				if b, ok3 := d.(*object.Bytes); ok3 {
					selfData = b.V
				}
			}
			if otherInst, ok2 := args[0].(*object.Instance); ok2 {
				if d, ok3 := otherInst.Dict.GetStr("data"); ok3 {
					if b, ok4 := d.(*object.Bytes); ok4 {
						otherData = b.V
					}
				}
			}
			return object.BoolOf(bytes.Equal(selfData, otherData)), nil
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

	// ── escape ────────────────────────────────────────────────────────────────

	m.Dict.SetStr("escape", &object.BuiltinFunc{Name: "escape", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		s := object.Str_(a[0])
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		return &object.Str{V: s}, nil
	}})

	// ── WRAPPERS ──────────────────────────────────────────────────────────────

	m.Dict.SetStr("WRAPPERS", &object.Tuple{V: []object.Object{dateTimeCls, binaryCls}})

	// ── gzip_encode / gzip_decode ─────────────────────────────────────────────

	m.Dict.SetStr("gzip_encode", &object.BuiltinFunc{Name: "gzip_encode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "gzip_encode requires data")
		}
		var raw []byte
		switch v := a[0].(type) {
		case *object.Bytes:
			raw = v.V
		case *object.Str:
			raw = []byte(v.V)
		default:
			return nil, object.Errorf(i.typeErr, "gzip_encode requires bytes")
		}
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(raw); err != nil {
			return nil, object.Errorf(i.valueErr, "gzip_encode: %v", err)
		}
		if err := w.Close(); err != nil {
			return nil, object.Errorf(i.valueErr, "gzip_encode: %v", err)
		}
		return &object.Bytes{V: buf.Bytes()}, nil
	}})

	m.Dict.SetStr("gzip_decode", &object.BuiltinFunc{Name: "gzip_decode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "gzip_decode requires data")
		}
		var raw []byte
		switch v := a[0].(type) {
		case *object.Bytes:
			raw = v.V
		case *object.Str:
			raw = []byte(v.V)
		default:
			return nil, object.Errorf(i.typeErr, "gzip_decode requires bytes")
		}
		r, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, object.Errorf(i.valueErr, "gzip_decode: %v", err)
		}
		defer r.Close()
		out, err := io.ReadAll(r)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "gzip_decode: %v", err)
		}
		return &object.Bytes{V: out}, nil
	}})

	// ── Marshaller ────────────────────────────────────────────────────────────

	marshallerCls := &object.Class{Name: "Marshaller", Dict: object.NewDict()}
	marshallerCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		inst.Dict.SetStr("allow_none", object.False)
		inst.Dict.SetStr("encoding", object.None)
		if kw != nil {
			if v, ok2 := kw.GetStr("allow_none"); ok2 {
				inst.Dict.SetStr("allow_none", v)
			}
			if v, ok2 := kw.GetStr("encoding"); ok2 {
				inst.Dict.SetStr("encoding", v)
			}
		}

		inst.Dict.SetStr("dumps", &object.BuiltinFunc{Name: "dumps", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if len(args) < 1 {
				return nil, object.Errorf(i.typeErr, "dumps requires values")
			}
			var items []object.Object
			switch v := args[0].(type) {
			case *object.Tuple:
				items = v.V
			case *object.List:
				items = v.V
			default:
				items = []object.Object{args[0]}
			}
			var sb strings.Builder
			sb.WriteString("<params>\n")
			for _, item := range items {
				sb.WriteString("<param>\n")
				sb.WriteString(marshalXmlrpc(item))
				sb.WriteString("\n</param>\n")
			}
			sb.WriteString("</params>\n")
			return &object.Str{V: sb.String()}, nil
		}})

		return object.None, nil
	}})
	m.Dict.SetStr("Marshaller", marshallerCls)

	// ── Unmarshaller / ExpatParser / getparser ────────────────────────────────

	unmarshallerCls := &object.Class{Name: "Unmarshaller", Dict: object.NewDict()}
	unmarshallerCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		// state: accumulated xml buffer, parsed results
		inst.Dict.SetStr("_xml", &object.Str{V: ""})
		inst.Dict.SetStr("_method", object.None)
		inst.Dict.SetStr("_params", object.None) // None = not yet parsed

		inst.Dict.SetStr("getmethodname", &object.BuiltinFunc{Name: "getmethodname", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if v, ok2 := inst.Dict.GetStr("_method"); ok2 {
				return v, nil
			}
			return object.None, nil
		}})

		inst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			params, ok2 := inst.Dict.GetStr("_params")
			if !ok2 || params == object.None {
				return nil, object.Errorf(responseErrCls, "response error")
			}
			return params, nil
		}})

		return object.None, nil
	}})
	m.Dict.SetStr("Unmarshaller", unmarshallerCls)

	expatParserCls := &object.Class{Name: "ExpatParser", Dict: object.NewDict()}
	expatParserCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		inst.Dict.SetStr("_buf", &object.Str{V: ""})
		inst.Dict.SetStr("_target", object.None)
		return object.None, nil
	}})
	m.Dict.SetStr("ExpatParser", expatParserCls)

	m.Dict.SetStr("getparser", &object.BuiltinFunc{Name: "getparser", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// Create a linked Unmarshaller + ExpatParser pair
		uInst := &object.Instance{Class: unmarshallerCls, Dict: object.NewDict()}
		uInst.Dict.SetStr("_xml", &object.Str{V: ""})
		uInst.Dict.SetStr("_method", object.None)
		uInst.Dict.SetStr("_params", object.None)
		uInst.Dict.SetStr("getmethodname", &object.BuiltinFunc{Name: "getmethodname", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if v, ok2 := uInst.Dict.GetStr("_method"); ok2 {
				return v, nil
			}
			return object.None, nil
		}})
		uInst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			params, ok2 := uInst.Dict.GetStr("_params")
			if !ok2 || params == object.None {
				return nil, object.Errorf(responseErrCls, "response error")
			}
			return params, nil
		}})

		pInst := &object.Instance{Class: expatParserCls, Dict: object.NewDict()}
		pInst.Dict.SetStr("_buf", &object.Str{V: ""})
		pInst.Dict.SetStr("_target", uInst)

		pInst.Dict.SetStr("feed", &object.BuiltinFunc{Name: "feed", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if len(args) < 1 {
				return object.None, nil
			}
			chunk := object.Str_(args[0])
			if cur, ok2 := pInst.Dict.GetStr("_buf"); ok2 {
				pInst.Dict.SetStr("_buf", &object.Str{V: object.Str_(cur) + chunk})
			}
			return object.None, nil
		}})

		pInst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			var xmlData string
			if cur, ok2 := pInst.Dict.GetStr("_buf"); ok2 {
				xmlData = object.Str_(cur)
			}
			method, params, err := parseXmlrpc(xmlData)
			if err != nil {
				return nil, object.Errorf(responseErrCls, "%v", err)
			}
			if method != "" {
				uInst.Dict.SetStr("_method", &object.Str{V: method})
			}
			// Store (params_tuple,) — same shape as loads() but just the first element
			paramsTuple := &object.Tuple{V: params}
			// close() on Unmarshaller returns params as tuple (matching CPython)
			uInst.Dict.SetStr("_params", paramsTuple)
			return object.None, nil
		}})

		return &object.Tuple{V: []object.Object{pInst, uInst}}, nil
	}})

	// ── MultiCallIterator ─────────────────────────────────────────────────────

	multiCallIterCls := &object.Class{Name: "MultiCallIterator", Dict: object.NewDict()}
	multiCallIterCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		var results []object.Object
		if len(a) >= 2 {
			switch v := a[1].(type) {
			case *object.List:
				results = v.V
			case *object.Tuple:
				results = v.V
			}
		}
		idx := 0
		inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})
		inst.Dict.SetStr("__next__", &object.BuiltinFunc{Name: "__next__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if idx >= len(results) {
				return nil, object.Errorf(i.stopIter, "")
			}
			item := results[idx]
			idx++
			// If dict with faultCode → raise Fault
			if d, ok2 := item.(*object.Dict); ok2 {
				if fc, ok3 := d.GetStr("faultCode"); ok3 {
					var fs object.Object = &object.Str{V: ""}
					if v, ok4 := d.GetStr("faultString"); ok4 {
						fs = v
					}
					faultExc := &object.Exception{
						Class: faultCls,
						Args:  &object.Tuple{V: []object.Object{fc, fs}},
						Msg:   fmt.Sprintf("%v: %v", object.Str_(fc), object.Str_(fs)),
					}
					return nil, faultExc
				}
			}
			// If list → unwrap first element
			if lst, ok2 := item.(*object.List); ok2 {
				if len(lst.V) > 0 {
					return lst.V[0], nil
				}
				return object.None, nil
			}
			return item, nil
		}})
		return object.None, nil
	}})
	m.Dict.SetStr("MultiCallIterator", multiCallIterCls)

	// ── Fast* aliases (None in pure-Python mode) ──────────────────────────────

	m.Dict.SetStr("FastParser", object.None)
	m.Dict.SetStr("FastMarshaller", object.None)
	m.Dict.SetStr("FastUnmarshaller", object.None)

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

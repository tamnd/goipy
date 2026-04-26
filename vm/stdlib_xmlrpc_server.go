package vm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildXmlrpcServer() *object.Module {
	m := &object.Module{Name: "xmlrpc.server", Dict: object.NewDict()}

	noop := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}
	}

	// ── Base classes ──────────────────────────────────────────────────────────

	var tcpServerCls, baseHTTPHandlerCls *object.Class
	if ssMod, err := i.loadModule("socketserver"); err == nil && ssMod != nil {
		if v, ok := ssMod.Dict.GetStr("TCPServer"); ok {
			tcpServerCls, _ = v.(*object.Class)
		}
	}
	if srvMod, err := i.loadModule("http.server"); err == nil && srvMod != nil {
		if v, ok := srvMod.Dict.GetStr("BaseHTTPRequestHandler"); ok {
			baseHTTPHandlerCls, _ = v.(*object.Class)
		}
	}
	if tcpServerCls == nil {
		tcpServerCls = &object.Class{Name: "TCPServer", Dict: object.NewDict()}
	}
	if baseHTTPHandlerCls == nil {
		baseHTTPHandlerCls = &object.Class{Name: "BaseHTTPRequestHandler", Dict: object.NewDict()}
	}

	// ── Fault class from xmlrpc.client ────────────────────────────────────────

	var faultCls *object.Class
	if clientMod, err := i.loadModule("xmlrpc.client"); err == nil && clientMod != nil {
		// Re-export Fault/dumps/loads
		for _, name := range []string{"Fault", "dumps", "loads"} {
			if v, ok := clientMod.Dict.GetStr(name); ok {
				m.Dict.SetStr(name, v)
			}
		}
		if v, ok := clientMod.Dict.GetStr("Fault"); ok {
			faultCls, _ = v.(*object.Class)
		}
	}
	if faultCls == nil {
		faultCls = &object.Class{Name: "Fault", Dict: object.NewDict()}
	}

	// ── list_public_methods ───────────────────────────────────────────────────

	m.Dict.SetStr("list_public_methods", &object.BuiltinFunc{Name: "list_public_methods", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: []object.Object{}}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.List{V: []object.Object{}}, nil
		}
		seen := make(map[string]bool)
		var names []string
		if inst.Class != nil {
			var collectFromClass func(cls *object.Class)
			collectFromClass = func(cls *object.Class) {
				ks, vs := cls.Dict.Items()
				for j, k := range ks {
					name := object.Str_(k)
					if strings.HasPrefix(name, "_") || seen[name] {
						continue
					}
					switch vs[j].(type) {
					case *object.Function, *object.BuiltinFunc, *object.BoundMethod:
						seen[name] = true
						names = append(names, name)
					}
				}
				for _, base := range cls.Bases {
					collectFromClass(base)
				}
			}
			collectFromClass(inst.Class)
		}
		sort.Strings(names)
		result := make([]object.Object, len(names))
		for j, n := range names {
			result[j] = &object.Str{V: n}
		}
		return &object.List{V: result}, nil
	}})

	// ── resolve_dotted_attribute ──────────────────────────────────────────────

	m.Dict.SetStr("resolve_dotted_attribute", &object.BuiltinFunc{Name: "resolve_dotted_attribute", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.attrErr, "resolve_dotted_attribute requires obj and attr")
		}
		obj := a[0]
		attr := object.Str_(a[1])
		allowDotted := true
		if kw != nil {
			if v, ok2 := kw.GetStr("allow_dotted_names"); ok2 {
				if b, ok3 := v.(*object.Bool); ok3 {
					allowDotted = b == object.True
				}
			}
		}
		parts := strings.Split(attr, ".")
		if len(parts) > 1 && !allowDotted {
			return nil, object.Errorf(i.attrErr, "dotted names are not allowed: '%s'", attr)
		}
		cur := obj
		for _, part := range parts {
			if inst, ok2 := cur.(*object.Instance); ok2 {
				// Check instance dict first
				if v, ok3 := inst.Dict.GetStr(part); ok3 {
					cur = v
					continue
				}
				// Then class hierarchy — bind as method
				if inst.Class != nil {
					if fn, ok3 := classLookup(inst.Class, part); ok3 {
						switch f := fn.(type) {
						case *object.Function:
							cur = &object.BoundMethod{Self: inst, Fn: f}
						case *object.BuiltinFunc:
							cur = &object.BoundMethod{Self: inst, Fn: f}
						default:
							cur = fn
						}
						continue
					}
				}
			} else if mod, ok2 := cur.(*object.Module); ok2 {
				if v, ok3 := mod.Dict.GetStr(part); ok3 {
					cur = v
					continue
				}
			}
			return nil, object.Errorf(i.attrErr, "object has no attribute '%s'", part)
		}
		return cur, nil
	}})

	// ── SimpleXMLRPCDispatcher ────────────────────────────────────────────────

	var dispatcherCls *object.Class
	setupDispatcherOnInst := func(inst *object.Instance) {
		funcs := object.NewDict()
		inst.Dict.SetStr("funcs", funcs)
		inst.Dict.SetStr("instance", object.None)
		inst.Dict.SetStr("allow_dotted_names", object.False)

		// ── dispatch helpers ──────────────────────────────────────────────────

		// dispatchMethod resolves and calls methodName with params.
		dispatchMethod := func(methodName string, params []object.Object) (object.Object, error) {
			// 1. Look in funcs dict
			if fn, ok := funcs.GetStr(methodName); ok {
				return i.callObject(fn, params, nil)
			}

			// 2. Try registered instance
			if instanceObj, ok2 := inst.Dict.GetStr("instance"); ok2 && instanceObj != object.None {
				if pyInst, ok3 := instanceObj.(*object.Instance); ok3 {
					allowDotted := false
					if v, ok4 := inst.Dict.GetStr("allow_dotted_names"); ok4 {
						if b, ok5 := v.(*object.Bool); ok5 {
							allowDotted = b == object.True
						}
					}
					parts := strings.Split(methodName, ".")
					if len(parts) > 1 && !allowDotted {
						return nil, &object.Exception{
							Class: faultCls,
							Args:  &object.Tuple{V: []object.Object{object.NewInt(1), &object.Str{V: fmt.Sprintf("method \"%s\" is not supported", methodName)}}},
							Msg:   fmt.Sprintf("method \"%s\" is not supported", methodName),
						}
					}
					curObj := object.Object(pyInst)
					for idx, part := range parts {
						if curInst, ok4 := curObj.(*object.Instance); ok4 {
							if v, ok5 := curInst.Dict.GetStr(part); ok5 {
								if idx == len(parts)-1 {
									// leaf: call with params
									return i.callObject(v, params, nil)
								}
								curObj = v
								continue
							}
							if curInst.Class != nil {
								if fn, ok5 := classLookup(curInst.Class, part); ok5 {
									if idx == len(parts)-1 {
										// leaf: prepend self and call
										callArgs := append([]object.Object{curInst}, params...)
										return i.callObject(fn, callArgs, nil)
									}
									// Intermediate: bind and set as new curObj
									curObj = &object.BoundMethod{Self: curInst, Fn: fn}
									continue
								}
							}
						} else if bm, ok4 := curObj.(*object.BoundMethod); ok4 {
							if idx == len(parts)-1 {
								return i.callObject(bm, params, nil)
							}
							r, err := i.callObject(bm, nil, nil)
							if err != nil {
								break
							}
							curObj = r
							continue
						}
						break
					}
				}
			}

			return nil, &object.Exception{
				Class: faultCls,
				Args:  &object.Tuple{V: []object.Object{object.NewInt(1), &object.Str{V: fmt.Sprintf("method \"%s\" is not supported", methodName)}}},
				Msg:   fmt.Sprintf("method \"%s\" is not supported", methodName),
			}
		}

		// ── methods ───────────────────────────────────────────────────────────

		inst.Dict.SetStr("register_function", &object.BuiltinFunc{Name: "register_function", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			fn := a[0]
			var name string
			if len(a) >= 2 && a[1] != object.None {
				name = object.Str_(a[1])
			} else if kw != nil {
				if v, ok2 := kw.GetStr("name"); ok2 && v != object.None {
					name = object.Str_(v)
				}
			}
			if name == "" {
				switch f := fn.(type) {
				case *object.Function:
					name = f.Name
				case *object.BuiltinFunc:
					name = f.Name
				case *object.Instance:
					if v, ok2 := f.Dict.GetStr("__name__"); ok2 {
						name = object.Str_(v)
					}
				}
			}
			if name != "" {
				funcs.SetStr(name, fn)
			}
			return fn, nil
		}})

		inst.Dict.SetStr("register_instance", &object.BuiltinFunc{Name: "register_instance", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				inst.Dict.SetStr("instance", a[0])
			}
			var allowDotted object.Object = object.False
			if kw != nil {
				if v, ok2 := kw.GetStr("allow_dotted_names"); ok2 {
					allowDotted = v
				}
			}
			inst.Dict.SetStr("allow_dotted_names", allowDotted)
			return object.None, nil
		}})

		inst.Dict.SetStr("register_introspection_functions", &object.BuiltinFunc{Name: "register_introspection_functions", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			funcs.SetStr("system.listMethods", &object.BuiltinFunc{Name: "system.listMethods", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if fn, ok := inst.Dict.GetStr("system_listMethods"); ok {
					return i.callObject(fn, nil, nil)
				}
				return &object.List{V: []object.Object{}}, nil
			}})
			funcs.SetStr("system.methodHelp", &object.BuiltinFunc{Name: "system.methodHelp", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if fn, ok := inst.Dict.GetStr("system_methodHelp"); ok {
					return i.callObject(fn, a, nil)
				}
				return &object.Str{V: ""}, nil
			}})
			funcs.SetStr("system.methodSignature", &object.BuiltinFunc{Name: "system.methodSignature", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if fn, ok := inst.Dict.GetStr("system_methodSignature"); ok {
					return i.callObject(fn, a, nil)
				}
				return &object.Str{V: "signatures not supported"}, nil
			}})
			return object.None, nil
		}})

		inst.Dict.SetStr("register_multicall_functions", &object.BuiltinFunc{Name: "register_multicall_functions", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			funcs.SetStr("system.multicall", &object.BuiltinFunc{Name: "system.multicall", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if fn, ok := inst.Dict.GetStr("system_multicall"); ok {
					return i.callObject(fn, a, nil)
				}
				return &object.List{V: []object.Object{}}, nil
			}})
			return object.None, nil
		}})

		inst.Dict.SetStr("system_listMethods", &object.BuiltinFunc{Name: "system_listMethods", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			seen := make(map[string]bool)
			var names []string
			// Methods from funcs dict
			ks, _ := funcs.Items()
			for _, k := range ks {
				n := object.Str_(k)
				if !seen[n] {
					seen[n] = true
					names = append(names, n)
				}
			}
			// Methods from registered instance
			if instanceObj, ok2 := inst.Dict.GetStr("instance"); ok2 && instanceObj != object.None {
				if pyInst, ok3 := instanceObj.(*object.Instance); ok3 && pyInst.Class != nil {
					var collectFromClass func(cls *object.Class)
					collectFromClass = func(cls *object.Class) {
						ks2, vs2 := cls.Dict.Items()
						for j, k := range ks2 {
							n := object.Str_(k)
							if strings.HasPrefix(n, "_") || seen[n] {
								continue
							}
							switch vs2[j].(type) {
							case *object.Function, *object.BuiltinFunc:
								seen[n] = true
								names = append(names, n)
							}
						}
						for _, base := range cls.Bases {
							collectFromClass(base)
						}
					}
					collectFromClass(pyInst.Class)
				}
			}
			sort.Strings(names)
			result := make([]object.Object, len(names))
			for j, n := range names {
				result[j] = &object.Str{V: n}
			}
			return &object.List{V: result}, nil
		}})

		inst.Dict.SetStr("system_methodHelp", &object.BuiltinFunc{Name: "system_methodHelp", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: ""}, nil
		}})

		inst.Dict.SetStr("system_methodSignature", &object.BuiltinFunc{Name: "system_methodSignature", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "signatures not supported"}, nil
		}})

		inst.Dict.SetStr("system_multicall", &object.BuiltinFunc{Name: "system_multicall", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		}})

		inst.Dict.SetStr("_dispatch", &object.BuiltinFunc{Name: "_dispatch", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(faultCls, "dispatch requires method and params")
			}
			methodName := object.Str_(a[0])
			var params []object.Object
			switch p := a[1].(type) {
			case *object.Tuple:
				params = p.V
			case *object.List:
				params = p.V
			}
			return dispatchMethod(methodName, params)
		}})

		inst.Dict.SetStr("_marshaled_dispatch", &object.BuiltinFunc{Name: "_marshaled_dispatch", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "_marshaled_dispatch requires data")
			}
			var data string
			switch v := a[0].(type) {
			case *object.Bytes:
				data = string(v.V)
			case *object.Str:
				data = v.V
			default:
				data = object.Str_(a[0])
			}
			method, params, parseErr := parseXmlrpc(data)
			if parseErr != nil {
				// Return fault XML
				resp := buildFaultXML(1, parseErr.Error())
				return &object.Bytes{V: []byte(resp)}, nil
			}
			result, dispErr := dispatchMethod(method, params)
			if dispErr != nil {
				var faultCode int64 = 1
				var faultStr string
				if exc, ok2 := dispErr.(*object.Exception); ok2 {
					if exc.Args != nil && len(exc.Args.V) >= 2 {
						if fc, ok3 := exc.Args.V[0].(*object.Int); ok3 {
							faultCode = fc.Int64()
						}
						faultStr = object.Str_(exc.Args.V[1])
					} else {
						faultStr = exc.Error()
					}
				} else {
					faultStr = dispErr.Error()
				}
				resp := buildFaultXML(faultCode, faultStr)
				return &object.Bytes{V: []byte(resp)}, nil
			}
			// Build success response
			var sb strings.Builder
			sb.WriteString("<?xml version='1.0'?>\n<methodResponse>\n<params>\n<param>\n")
			sb.WriteString(marshalXmlrpcVal(result))
			sb.WriteString("\n</param>\n</params>\n</methodResponse>\n")
			return &object.Bytes{V: []byte(sb.String())}, nil
		}})
	}

	dispatcherCls = &object.Class{Name: "SimpleXMLRPCDispatcher", Dict: object.NewDict()}
	dispatcherCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		setupDispatcherOnInst(inst)
		var allowNone, encoding, useBuiltinTypes object.Object
		allowNone = object.False
		encoding = object.None
		useBuiltinTypes = object.False
		if kw != nil {
			if v, ok2 := kw.GetStr("allow_none"); ok2 {
				allowNone = v
			}
			if v, ok2 := kw.GetStr("encoding"); ok2 {
				encoding = v
			}
			if v, ok2 := kw.GetStr("use_builtin_types"); ok2 {
				useBuiltinTypes = v
			}
		}
		inst.Dict.SetStr("allow_none", allowNone)
		inst.Dict.SetStr("encoding", encoding)
		inst.Dict.SetStr("use_builtin_types", useBuiltinTypes)
		return object.None, nil
	}})
	m.Dict.SetStr("SimpleXMLRPCDispatcher", dispatcherCls)

	// ── SimpleXMLRPCRequestHandler ────────────────────────────────────────────

	requestHandlerCls := &object.Class{
		Name:  "SimpleXMLRPCRequestHandler",
		Bases: []*object.Class{baseHTTPHandlerCls},
		Dict:  object.NewDict(),
	}
	requestHandlerCls.Dict.SetStr("rpc_paths", &object.Tuple{V: []object.Object{
		&object.Str{V: "/"}, &object.Str{V: "/RPC2"},
	}})
	requestHandlerCls.Dict.SetStr("encode_threshold", object.NewInt(1400))
	requestHandlerCls.Dict.SetStr("do_POST", noop("do_POST"))
	requestHandlerCls.Dict.SetStr("do_GET", noop("do_GET"))
	m.Dict.SetStr("SimpleXMLRPCRequestHandler", requestHandlerCls)

	// ── SimpleXMLRPCServer ────────────────────────────────────────────────────

	xmlrpcServerCls := &object.Class{
		Name:  "SimpleXMLRPCServer",
		Bases: []*object.Class{tcpServerCls, dispatcherCls},
		Dict:  object.NewDict(),
	}
	xmlrpcServerCls.Dict.SetStr("allow_reuse_address", object.True)
	xmlrpcServerCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		setupDispatcherOnInst(inst)
		inst.Dict.SetStr("logRequests", object.True)
		inst.Dict.SetStr("allow_none", object.False)
		inst.Dict.SetStr("encoding", object.None)
		inst.Dict.SetStr("use_builtin_types", object.True)
		if len(a) >= 2 {
			inst.Dict.SetStr("server_address", a[1])
		}
		if kw != nil {
			for _, key := range []string{"logRequests", "allow_none", "encoding", "use_builtin_types"} {
				if v, ok2 := kw.GetStr(key); ok2 {
					inst.Dict.SetStr(key, v)
				}
			}
		}
		return object.None, nil
	}})
	xmlrpcServerCls.Dict.SetStr("server_close", noop("server_close"))
	m.Dict.SetStr("SimpleXMLRPCServer", xmlrpcServerCls)

	// ── CGIXMLRPCRequestHandler ───────────────────────────────────────────────

	cgiHandlerCls := &object.Class{Name: "CGIXMLRPCRequestHandler", Bases: []*object.Class{dispatcherCls}, Dict: object.NewDict()}
	cgiHandlerCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		setupDispatcherOnInst(inst)
		return object.None, nil
	}})
	cgiHandlerCls.Dict.SetStr("handle_xmlrpc", noop("handle_xmlrpc"))
	cgiHandlerCls.Dict.SetStr("handle_get", noop("handle_get"))
	cgiHandlerCls.Dict.SetStr("handle_request", noop("handle_request"))
	m.Dict.SetStr("CGIXMLRPCRequestHandler", cgiHandlerCls)

	// ── MultiPathXMLRPCServer ─────────────────────────────────────────────────

	multiPathCls := &object.Class{Name: "MultiPathXMLRPCServer", Bases: []*object.Class{xmlrpcServerCls}, Dict: object.NewDict()}
	m.Dict.SetStr("MultiPathXMLRPCServer", multiPathCls)

	// ── XMLRPCDocGenerator ────────────────────────────────────────────────────

	xmlrpcDocGenCls := &object.Class{Name: "XMLRPCDocGenerator", Dict: object.NewDict()}
	xmlrpcDocGenCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		inst.Dict.SetStr("server_name", &object.Str{V: "XML-RPC Server Documentation"})
		inst.Dict.SetStr("server_title", &object.Str{V: "XML-RPC Server Documentation"})
		inst.Dict.SetStr("server_documentation", &object.Str{V: "This server exports the following methods through the XML-RPC protocol."})

		inst.Dict.SetStr("set_server_name", &object.BuiltinFunc{Name: "set_server_name", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if len(args) >= 1 {
				inst.Dict.SetStr("server_name", args[0])
			}
			return object.None, nil
		}})
		inst.Dict.SetStr("set_server_title", &object.BuiltinFunc{Name: "set_server_title", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if len(args) >= 1 {
				inst.Dict.SetStr("server_title", args[0])
			}
			return object.None, nil
		}})
		inst.Dict.SetStr("set_server_documentation", &object.BuiltinFunc{Name: "set_server_documentation", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if len(args) >= 1 {
				inst.Dict.SetStr("server_documentation", args[0])
			}
			return object.None, nil
		}})
		inst.Dict.SetStr("generate_html_documentation", noop("generate_html_documentation"))
		return object.None, nil
	}})
	m.Dict.SetStr("XMLRPCDocGenerator", xmlrpcDocGenCls)

	// ── ServerHTMLDoc ─────────────────────────────────────────────────────────

	serverHTMLDocCls := &object.Class{Name: "ServerHTMLDoc", Dict: object.NewDict()}
	m.Dict.SetStr("ServerHTMLDoc", serverHTMLDocCls)

	// ── Doc variants ─────────────────────────────────────────────────────────

	// DocXMLRPCRequestHandler inherits only from SimpleXMLRPCRequestHandler (matches CPython)
	docRequestHandlerCls := &object.Class{Name: "DocXMLRPCRequestHandler", Bases: []*object.Class{requestHandlerCls}, Dict: object.NewDict()}
	m.Dict.SetStr("DocXMLRPCRequestHandler", docRequestHandlerCls)

	docServerCls := &object.Class{Name: "DocXMLRPCServer", Bases: []*object.Class{xmlrpcServerCls, xmlrpcDocGenCls}, Dict: object.NewDict()}
	m.Dict.SetStr("DocXMLRPCServer", docServerCls)

	docCgiCls := &object.Class{Name: "DocCGIXMLRPCRequestHandler", Bases: []*object.Class{cgiHandlerCls, xmlrpcDocGenCls}, Dict: object.NewDict()}
	m.Dict.SetStr("DocCGIXMLRPCRequestHandler", docCgiCls)

	// ── BaseHTTPRequestHandler re-export ─────────────────────────────────────

	m.Dict.SetStr("BaseHTTPRequestHandler", baseHTTPHandlerCls)

	_ = strings.TrimSpace // suppress unused import

	return m
}

// buildFaultXML constructs an XML-RPC fault methodResponse.
func buildFaultXML(code int64, msg string) string {
	msg = strings.ReplaceAll(msg, "&", "&amp;")
	msg = strings.ReplaceAll(msg, "<", "&lt;")
	msg = strings.ReplaceAll(msg, ">", "&gt;")
	return fmt.Sprintf(
		"<?xml version='1.0'?>\n<methodResponse>\n<fault>\n<value><struct>\n"+
			"<member><name>faultCode</name><value><int>%d</int></value></member>\n"+
			"<member><name>faultString</name><value><string>%s</string></value></member>\n"+
			"</struct></value>\n</fault>\n</methodResponse>\n",
		code, msg,
	)
}

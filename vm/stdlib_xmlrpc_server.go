package vm

import (
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

	// Re-export Fault/dumps/loads from xmlrpc.client
	if clientMod, err := i.loadModule("xmlrpc.client"); err == nil && clientMod != nil {
		for _, name := range []string{"Fault", "dumps", "loads"} {
			if v, ok := clientMod.Dict.GetStr(name); ok {
				m.Dict.SetStr(name, v)
			}
		}
	}

	// ── SimpleXMLRPCDispatcher ────────────────────────────────────────────────

	var dispatcherCls *object.Class
	setupDispatcherOnInst := func(inst *object.Instance) {
		funcs := object.NewDict()
		inst.Dict.SetStr("funcs", funcs)
		inst.Dict.SetStr("instance", object.None)

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

		inst.Dict.SetStr("register_instance", &object.BuiltinFunc{Name: "register_instance", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				inst.Dict.SetStr("instance", a[0])
			}
			return object.None, nil
		}})

		inst.Dict.SetStr("register_introspection_functions", &object.BuiltinFunc{Name: "register_introspection_functions", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

		inst.Dict.SetStr("register_multicall_functions", &object.BuiltinFunc{Name: "register_multicall_functions", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

		inst.Dict.SetStr("system_listMethods", &object.BuiltinFunc{Name: "system_listMethods", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ks, _ := funcs.Items()
			names := make([]string, 0, len(ks))
			for _, k := range ks {
				names = append(names, object.Str_(k))
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

	// ── Doc variants ─────────────────────────────────────────────────────────

	xmlrpcDocGenCls := &object.Class{Name: "XMLRPCDocGenerator", Dict: object.NewDict()}
	m.Dict.SetStr("XMLRPCDocGenerator", xmlrpcDocGenCls)

	docRequestHandlerCls := &object.Class{Name: "DocXMLRPCRequestHandler", Bases: []*object.Class{requestHandlerCls, xmlrpcDocGenCls}, Dict: object.NewDict()}
	m.Dict.SetStr("DocXMLRPCRequestHandler", docRequestHandlerCls)

	docServerCls := &object.Class{Name: "DocXMLRPCServer", Bases: []*object.Class{xmlrpcServerCls, xmlrpcDocGenCls}, Dict: object.NewDict()}
	m.Dict.SetStr("DocXMLRPCServer", docServerCls)

	docCgiCls := &object.Class{Name: "DocCGIXMLRPCRequestHandler", Bases: []*object.Class{cgiHandlerCls, xmlrpcDocGenCls}, Dict: object.NewDict()}
	m.Dict.SetStr("DocCGIXMLRPCRequestHandler", docCgiCls)

	_ = strings.TrimSpace // suppress unused import

	return m
}

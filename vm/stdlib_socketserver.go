package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildSocketserver() *object.Module {
	m := &object.Module{Name: "socketserver", Dict: object.NewDict()}

	noop := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}
	}

	stubTrue := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}}
	}

	// ── BaseServer ────────────────────────────────────────────────────────────

	baseServerCls := &object.Class{Name: "BaseServer", Dict: object.NewDict()}
	baseServerCls.Dict.SetStr("allow_reuse_address", object.False)
	baseServerCls.Dict.SetStr("request_queue_size", object.NewInt(5))
	baseServerCls.Dict.SetStr("socket_type", object.None)
	baseServerCls.Dict.SetStr("address_family", object.None)
	baseServerCls.Dict.SetStr("timeout", object.None)

	baseServerCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		if len(a) >= 2 {
			inst.Dict.SetStr("server_address", a[1])
		} else if kw != nil {
			if v, ok2 := kw.GetStr("server_address"); ok2 {
				inst.Dict.SetStr("server_address", v)
			}
		}
		if len(a) >= 3 {
			inst.Dict.SetStr("RequestHandlerClass", a[2])
		} else if kw != nil {
			if v, ok2 := kw.GetStr("RequestHandlerClass"); ok2 {
				inst.Dict.SetStr("RequestHandlerClass", v)
			}
		}
		return object.None, nil
	}})

	baseServerCls.Dict.SetStr("fileno", &object.BuiltinFunc{Name: "fileno", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(-1), nil
	}})
	baseServerCls.Dict.SetStr("handle_request", noop("handle_request"))
	baseServerCls.Dict.SetStr("serve_forever", noop("serve_forever"))
	baseServerCls.Dict.SetStr("service_actions", noop("service_actions"))
	baseServerCls.Dict.SetStr("shutdown", noop("shutdown"))
	baseServerCls.Dict.SetStr("server_close", noop("server_close"))
	baseServerCls.Dict.SetStr("finish_request", noop("finish_request"))
	baseServerCls.Dict.SetStr("get_request", noop("get_request"))
	baseServerCls.Dict.SetStr("handle_error", noop("handle_error"))
	baseServerCls.Dict.SetStr("handle_timeout", noop("handle_timeout"))
	baseServerCls.Dict.SetStr("process_request", noop("process_request"))
	baseServerCls.Dict.SetStr("server_activate", noop("server_activate"))
	baseServerCls.Dict.SetStr("close_request", noop("close_request"))
	baseServerCls.Dict.SetStr("shutdown_request", noop("shutdown_request"))
	baseServerCls.Dict.SetStr("verify_request", stubTrue("verify_request"))
	baseServerCls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			return a[0], nil
		}
		return object.None, nil
	}})
	baseServerCls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			if inst, ok := a[0].(*object.Instance); ok {
				if fn, ok2 := inst.Class.Dict.GetStr("server_close"); ok2 {
					if bf, ok3 := fn.(*object.BuiltinFunc); ok3 {
						bf.Call(nil, a[:1], nil)
					}
				}
			}
		}
		return object.False, nil
	}})

	m.Dict.SetStr("BaseServer", baseServerCls)

	// ── TCPServer(BaseServer) ─────────────────────────────────────────────────

	tcpServerCls := &object.Class{
		Name:  "TCPServer",
		Bases: []*object.Class{baseServerCls},
		Dict:  object.NewDict(),
	}
	tcpServerCls.Dict.SetStr("allow_reuse_address", object.False)
	tcpServerCls.Dict.SetStr("allow_reuse_port", object.False)
	tcpServerCls.Dict.SetStr("request_queue_size", object.NewInt(5))
	tcpServerCls.Dict.SetStr("address_family", object.NewInt(2))  // AF_INET
	tcpServerCls.Dict.SetStr("socket_type", object.NewInt(1))     // SOCK_STREAM

	tcpServerCls.Dict.SetStr("server_bind", noop("server_bind"))

	tcpInitFn := &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		if len(a) >= 2 {
			inst.Dict.SetStr("server_address", a[1])
		}
		if len(a) >= 3 {
			inst.Dict.SetStr("RequestHandlerClass", a[2])
		}
		// bind_and_activate defaults to True but we skip actual socket ops
		return object.None, nil
	}}
	tcpServerCls.Dict.SetStr("__init__", tcpInitFn)
	m.Dict.SetStr("TCPServer", tcpServerCls)

	// ── UDPServer(BaseServer) ─────────────────────────────────────────────────

	udpServerCls := &object.Class{
		Name:  "UDPServer",
		Bases: []*object.Class{baseServerCls},
		Dict:  object.NewDict(),
	}
	udpServerCls.Dict.SetStr("address_family", object.NewInt(2))  // AF_INET
	udpServerCls.Dict.SetStr("socket_type", object.NewInt(2))     // SOCK_DGRAM
	udpServerCls.Dict.SetStr("__init__", tcpInitFn)
	m.Dict.SetStr("UDPServer", udpServerCls)

	// ── UnixStreamServer(TCPServer) ───────────────────────────────────────────

	unixStreamCls := &object.Class{
		Name:  "UnixStreamServer",
		Bases: []*object.Class{tcpServerCls},
		Dict:  object.NewDict(),
	}
	unixStreamCls.Dict.SetStr("address_family", object.NewInt(1))  // AF_UNIX
	m.Dict.SetStr("UnixStreamServer", unixStreamCls)

	// ── UnixDatagramServer(UDPServer) ─────────────────────────────────────────

	unixDatagramCls := &object.Class{
		Name:  "UnixDatagramServer",
		Bases: []*object.Class{udpServerCls},
		Dict:  object.NewDict(),
	}
	unixDatagramCls.Dict.SetStr("address_family", object.NewInt(1))  // AF_UNIX
	m.Dict.SetStr("UnixDatagramServer", unixDatagramCls)

	// ── ThreadingMixIn ────────────────────────────────────────────────────────

	threadingMixInCls := &object.Class{Name: "ThreadingMixIn", Dict: object.NewDict()}
	threadingMixInCls.Dict.SetStr("daemon_threads", object.False)
	threadingMixInCls.Dict.SetStr("block_on_close", object.True)
	threadingMixInCls.Dict.SetStr("process_request", noop("process_request"))
	m.Dict.SetStr("ThreadingMixIn", threadingMixInCls)

	// ── ForkingMixIn ─────────────────────────────────────────────────────────

	forkingMixInCls := &object.Class{Name: "ForkingMixIn", Dict: object.NewDict()}
	forkingMixInCls.Dict.SetStr("max_children", object.NewInt(40))
	forkingMixInCls.Dict.SetStr("block_on_close", object.True)
	forkingMixInCls.Dict.SetStr("process_request", noop("process_request"))
	m.Dict.SetStr("ForkingMixIn", forkingMixInCls)

	// ── Pre-combined Threading classes ────────────────────────────────────────

	mkCombined := func(name string, mixin, server *object.Class) *object.Class {
		cls := &object.Class{
			Name:  name,
			Bases: []*object.Class{mixin, server},
			Dict:  object.NewDict(),
		}
		return cls
	}

	threadingTCPCls := mkCombined("ThreadingTCPServer", threadingMixInCls, tcpServerCls)
	threadingUDPCls := mkCombined("ThreadingUDPServer", threadingMixInCls, udpServerCls)
	threadingUnixStreamCls := mkCombined("ThreadingUnixStreamServer", threadingMixInCls, unixStreamCls)
	threadingUnixDatagramCls := mkCombined("ThreadingUnixDatagramServer", threadingMixInCls, unixDatagramCls)

	m.Dict.SetStr("ThreadingTCPServer", threadingTCPCls)
	m.Dict.SetStr("ThreadingUDPServer", threadingUDPCls)
	m.Dict.SetStr("ThreadingUnixStreamServer", threadingUnixStreamCls)
	m.Dict.SetStr("ThreadingUnixDatagramServer", threadingUnixDatagramCls)

	// ── Pre-combined Forking classes ──────────────────────────────────────────

	forkingTCPCls := mkCombined("ForkingTCPServer", forkingMixInCls, tcpServerCls)
	forkingUDPCls := mkCombined("ForkingUDPServer", forkingMixInCls, udpServerCls)
	forkingUnixStreamCls := mkCombined("ForkingUnixStreamServer", forkingMixInCls, unixStreamCls)
	forkingUnixDatagramCls := mkCombined("ForkingUnixDatagramServer", forkingMixInCls, unixDatagramCls)

	m.Dict.SetStr("ForkingTCPServer", forkingTCPCls)
	m.Dict.SetStr("ForkingUDPServer", forkingUDPCls)
	m.Dict.SetStr("ForkingUnixStreamServer", forkingUnixStreamCls)
	m.Dict.SetStr("ForkingUnixDatagramServer", forkingUnixDatagramCls)

	// ── BaseRequestHandler ────────────────────────────────────────────────────

	baseHandlerCls := &object.Class{Name: "BaseRequestHandler", Dict: object.NewDict()}

	baseHandlerCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		if len(a) >= 2 {
			inst.Dict.SetStr("request", a[1])
		} else {
			inst.Dict.SetStr("request", object.None)
		}
		if len(a) >= 3 {
			inst.Dict.SetStr("client_address", a[2])
		} else {
			inst.Dict.SetStr("client_address", object.None)
		}
		if len(a) >= 4 {
			inst.Dict.SetStr("server", a[3])
		} else {
			inst.Dict.SetStr("server", object.None)
		}
		return object.None, nil
	}})
	baseHandlerCls.Dict.SetStr("setup", noop("setup"))
	baseHandlerCls.Dict.SetStr("handle", noop("handle"))
	baseHandlerCls.Dict.SetStr("finish", noop("finish"))

	m.Dict.SetStr("BaseRequestHandler", baseHandlerCls)

	// ── StreamRequestHandler(BaseRequestHandler) ──────────────────────────────

	streamHandlerCls := &object.Class{
		Name:  "StreamRequestHandler",
		Bases: []*object.Class{baseHandlerCls},
		Dict:  object.NewDict(),
	}
	streamHandlerCls.Dict.SetStr("rbufsize", object.NewInt(-1))
	streamHandlerCls.Dict.SetStr("wbufsize", object.NewInt(0))
	streamHandlerCls.Dict.SetStr("timeout", object.None)
	streamHandlerCls.Dict.SetStr("disable_nagle_algorithm", object.False)
	streamHandlerCls.Dict.SetStr("setup", noop("setup"))
	streamHandlerCls.Dict.SetStr("finish", noop("finish"))

	m.Dict.SetStr("StreamRequestHandler", streamHandlerCls)

	// ── DatagramRequestHandler(BaseRequestHandler) ────────────────────────────

	datagramHandlerCls := &object.Class{
		Name:  "DatagramRequestHandler",
		Bases: []*object.Class{baseHandlerCls},
		Dict:  object.NewDict(),
	}
	datagramHandlerCls.Dict.SetStr("setup", noop("setup"))
	datagramHandlerCls.Dict.SetStr("finish", noop("finish"))

	m.Dict.SetStr("DatagramRequestHandler", datagramHandlerCls)

	return m
}

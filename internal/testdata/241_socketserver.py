import socketserver


def test_class_attrs_TCP():
    print(socketserver.TCPServer.allow_reuse_address)
    print(socketserver.TCPServer.allow_reuse_port)
    print(socketserver.TCPServer.request_queue_size)
    print(int(socketserver.TCPServer.socket_type))
    print(int(socketserver.TCPServer.address_family))
    print('tcp_attrs ok')


def test_class_attrs_UDP():
    print(int(socketserver.UDPServer.socket_type))
    print(int(socketserver.UDPServer.address_family))
    print('udp_attrs ok')


def test_class_attrs_Unix():
    print(int(socketserver.UnixStreamServer.address_family))
    print(int(socketserver.UnixDatagramServer.address_family))
    print('unix_attrs ok')


def test_class_attrs_Threading():
    print(socketserver.ThreadingMixIn.daemon_threads)
    print(socketserver.ThreadingMixIn.block_on_close)
    print('threading_attrs ok')


def test_class_attrs_Forking():
    print(socketserver.ForkingMixIn.max_children)
    print(socketserver.ForkingMixIn.block_on_close)
    print('forking_attrs ok')


def test_class_attrs_Stream():
    print(socketserver.StreamRequestHandler.rbufsize)
    print(socketserver.StreamRequestHandler.wbufsize)
    print(socketserver.StreamRequestHandler.timeout)
    print(socketserver.StreamRequestHandler.disable_nagle_algorithm)
    print('stream_attrs ok')


def test_hierarchy_server():
    print(issubclass(socketserver.TCPServer, socketserver.BaseServer))
    print(issubclass(socketserver.UDPServer, socketserver.BaseServer))
    print('server_hierarchy ok')


def test_hierarchy_unix():
    print(issubclass(socketserver.UnixStreamServer, socketserver.TCPServer))
    print(issubclass(socketserver.UnixDatagramServer, socketserver.UDPServer))
    print('unix_hierarchy ok')


def test_hierarchy_threading():
    print(issubclass(socketserver.ThreadingTCPServer, socketserver.TCPServer))
    print(issubclass(socketserver.ThreadingTCPServer, socketserver.ThreadingMixIn))
    print(issubclass(socketserver.ThreadingUDPServer, socketserver.UDPServer))
    print('threading_hierarchy ok')


def test_hierarchy_forking():
    print(issubclass(socketserver.ForkingTCPServer, socketserver.TCPServer))
    print(issubclass(socketserver.ForkingTCPServer, socketserver.ForkingMixIn))
    print(issubclass(socketserver.ForkingUnixStreamServer, socketserver.UnixStreamServer))
    print('forking_hierarchy ok')


def test_hierarchy_handlers():
    print(issubclass(socketserver.StreamRequestHandler, socketserver.BaseRequestHandler))
    print(issubclass(socketserver.DatagramRequestHandler, socketserver.BaseRequestHandler))
    print('handler_hierarchy ok')


def test_baseserver_methods():
    for name in ['verify_request', 'handle_error', 'server_close', 'handle_request',
                 'serve_forever', 'shutdown', 'finish_request', 'process_request',
                 'server_bind', 'server_activate', 'handle_timeout', 'service_actions']:
        print(name, hasattr(socketserver.BaseServer, name))
    print('baseserver_methods ok')


def test_instantiation():
    s = socketserver.TCPServer(('127.0.0.1', 0), socketserver.BaseRequestHandler,
                               bind_and_activate=False)
    print(type(s).__name__)
    print(s.server_address)
    print(s.RequestHandlerClass.__name__)
    print(s.allow_reuse_address)
    s.server_close()
    print('instantiation ok')


def test_verify_request():
    s = socketserver.TCPServer(('127.0.0.1', 0), socketserver.BaseRequestHandler,
                               bind_and_activate=False)
    print(s.verify_request(None, None))
    s.server_close()
    print('verify_request ok')


def test_handler_methods():
    for name in ['setup', 'handle', 'finish']:
        print(name, hasattr(socketserver.BaseRequestHandler, name))
    print('handler_methods ok')


def test_combined_classes():
    names = ['ThreadingTCPServer', 'ThreadingUDPServer', 'ThreadingUnixStreamServer',
             'ThreadingUnixDatagramServer', 'ForkingTCPServer', 'ForkingUDPServer',
             'ForkingUnixStreamServer', 'ForkingUnixDatagramServer']
    for name in names:
        print(name, hasattr(socketserver, name))
    print('combined_classes ok')


def test_module_exports():
    for name in ['BaseServer', 'TCPServer', 'UDPServer', 'UnixStreamServer',
                 'UnixDatagramServer', 'ThreadingMixIn', 'ForkingMixIn',
                 'BaseRequestHandler', 'StreamRequestHandler', 'DatagramRequestHandler']:
        print(name, name in dir(socketserver))
    print('module_exports ok')


test_class_attrs_TCP()
test_class_attrs_UDP()
test_class_attrs_Unix()
test_class_attrs_Threading()
test_class_attrs_Forking()
test_class_attrs_Stream()
test_hierarchy_server()
test_hierarchy_unix()
test_hierarchy_threading()
test_hierarchy_forking()
test_hierarchy_handlers()
test_baseserver_methods()
test_instantiation()
test_verify_request()
test_handler_methods()
test_combined_classes()
test_module_exports()

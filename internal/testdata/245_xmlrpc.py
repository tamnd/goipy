import xmlrpc.client
import xmlrpc.server
import socketserver


def test_error_hierarchy():
    xc = xmlrpc.client
    print(issubclass(xc.Error, Exception))
    print(issubclass(xc.Fault, xc.Error))
    print(issubclass(xc.ProtocolError, xc.Error))
    print(issubclass(xc.ResponseError, xc.Error))
    print('error_hierarchy ok')


def test_fault_attrs():
    f = xmlrpc.client.Fault(42, 'something went wrong')
    print(f.faultCode)
    print(f.faultString)
    print('fault_attrs ok')


def test_fault_repr():
    f = xmlrpc.client.Fault(7, 'bad input')
    r = repr(f)
    print('Fault' in r)
    print('7' in r)
    print('bad input' in r)
    print('fault_repr ok')


def test_protocol_error_attrs():
    pe = xmlrpc.client.ProtocolError('http://example.com/rpc', 404, 'Not Found', {})
    print(pe.url)
    print(pe.errcode)
    print(pe.errmsg)
    print(type(pe.headers).__name__)
    print('protocol_error_attrs ok')


def test_datetime():
    dt = xmlrpc.client.DateTime('20240101T12:00:00')
    print(type(dt).__name__)
    print(dt.value)
    print('datetime ok')


def test_binary():
    b = xmlrpc.client.Binary(b'hello')
    print(type(b).__name__)
    print(b.data)
    print('binary ok')


def test_binary_decode():
    b = xmlrpc.client.Binary()
    b.decode(b'aGVsbG8=')
    print(b.data)
    print('binary_decode ok')


def test_boolean():
    print(xmlrpc.client.boolean(True))
    print(xmlrpc.client.boolean(False))
    print(xmlrpc.client.boolean(1))
    print(xmlrpc.client.boolean(0))
    print('boolean ok')


def test_constants():
    print(xmlrpc.client.MAXINT)
    print(xmlrpc.client.MININT)
    print(xmlrpc.client.APPLICATION_ERROR)
    print('constants ok')


def test_dumps_str():
    data = xmlrpc.client.dumps((42, 'hello'), 'test_method')
    print(isinstance(data, str))
    print('methodCall' in data)
    print('test_method' in data)
    print('42' in data)
    print('hello' in data)
    print('dumps_str ok')


def test_loads():
    data = xmlrpc.client.dumps((42, 'hi'), 'my_func')
    params, method = xmlrpc.client.loads(data)
    print(method)
    print(params[0])
    print(params[1])
    print('loads ok')


def test_loads_bool():
    data = xmlrpc.client.dumps((True, False), 'bools')
    params, method = xmlrpc.client.loads(data)
    print(method)
    print(params[0])
    print(params[1])
    print('loads_bool ok')


def test_server_proxy():
    s = xmlrpc.client.ServerProxy('http://localhost:8000')
    print(type(s).__name__)
    print('server_proxy ok')


def test_multicall():
    s = xmlrpc.client.ServerProxy('http://localhost:8000')
    mc = xmlrpc.client.MultiCall(s)
    print(type(mc).__name__)
    print('multicall ok')


def test_dispatcher():
    d = xmlrpc.server.SimpleXMLRPCDispatcher(allow_none=True, encoding='utf-8')
    print(type(d).__name__)

    def add(x, y):
        return x + y

    d.register_function(add)
    methods = d.system_listMethods()
    print('add' in methods)
    print(d.system_methodSignature('add'))
    print('dispatcher ok')


def test_server_hierarchy():
    xs = xmlrpc.server
    print(issubclass(xs.SimpleXMLRPCServer, socketserver.TCPServer))
    print(issubclass(xs.SimpleXMLRPCServer, xs.SimpleXMLRPCDispatcher))
    print(issubclass(xs.MultiPathXMLRPCServer, xs.SimpleXMLRPCServer))
    print(issubclass(xs.CGIXMLRPCRequestHandler, xs.SimpleXMLRPCDispatcher))
    print('server_hierarchy ok')


def test_request_handler_attrs():
    rh = xmlrpc.server.SimpleXMLRPCRequestHandler
    print('/' in rh.rpc_paths)
    print('/RPC2' in rh.rpc_paths)
    print(rh.encode_threshold)
    print('request_handler_attrs ok')


def test_client_exports():
    xc = xmlrpc.client
    for name in ['Error', 'Fault', 'ProtocolError', 'ResponseError',
                 'DateTime', 'Binary', 'ServerProxy', 'MultiCall',
                 'Transport', 'dumps', 'loads', 'MAXINT', 'MININT']:
        print(name, name in dir(xc))
    print('client_exports ok')


def test_server_exports():
    xs = xmlrpc.server
    for name in ['SimpleXMLRPCServer', 'SimpleXMLRPCDispatcher',
                 'SimpleXMLRPCRequestHandler', 'CGIXMLRPCRequestHandler',
                 'MultiPathXMLRPCServer']:
        print(name, name in dir(xs))
    print('server_exports ok')


test_error_hierarchy()
test_fault_attrs()
test_fault_repr()
test_protocol_error_attrs()
test_datetime()
test_binary()
test_binary_decode()
test_boolean()
test_constants()
test_dumps_str()
test_loads()
test_loads_bool()
test_server_proxy()
test_multicall()
test_dispatcher()
test_server_hierarchy()
test_request_handler_attrs()
test_client_exports()
test_server_exports()

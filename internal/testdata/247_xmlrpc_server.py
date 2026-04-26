import xmlrpc.server as xs
import xmlrpc.client as xc
import socketserver


def test_list_public_methods():
    class MyService:
        def add(self, x, y): return x + y
        def compute(self, n): return n * 2
        def _private(self): pass

    methods = xs.list_public_methods(MyService())
    print(sorted(methods))
    print('add' in methods)
    print('_private' in methods)
    print('list_public_methods ok')


class _Inner:
    def hello(self): return 'inner hello'


class _Outer:
    pass


def test_resolve_dotted_attribute():
    outer = _Outer()
    outer.inner = _Inner()

    # Simple attribute (stored in instance dict)
    result = xs.resolve_dotted_attribute(outer, 'inner')
    print(type(result).__name__)

    # Dotted attribute
    fn = xs.resolve_dotted_attribute(outer, 'inner.hello', allow_dotted_names=True)
    print(fn())

    # Dotted blocked
    try:
        xs.resolve_dotted_attribute(outer, 'inner.hello', allow_dotted_names=False)
        print('no error')
    except AttributeError:
        print('AttributeError raised')

    print('resolve_dotted_attribute ok')


def test_dispatch_direct():
    d = xs.SimpleXMLRPCDispatcher(allow_none=True)

    def add(x, y): return x + y
    def echo(s): return s

    d.register_function(add)
    d.register_function(echo)

    print(d._dispatch('add', (3, 4)))
    print(d._dispatch('echo', ('hello',)))

    try:
        d._dispatch('missing', ())
        print('no error')
    except Exception as e:
        print('error raised')

    print('dispatch_direct ok')


def test_dispatch_instance():
    class MyObj:
        def hello(self): return 'world'
        def compute(self, n): return n * 3

    d = xs.SimpleXMLRPCDispatcher(allow_none=True)
    d.register_instance(MyObj())

    print(d._dispatch('hello', ()))
    print(d._dispatch('compute', (5,)))
    print('dispatch_instance ok')


def test_marshaled_dispatch_success():
    d = xs.SimpleXMLRPCDispatcher(allow_none=True)

    def add(x, y): return x + y
    d.register_function(add)

    req = xc.dumps((10, 20), 'add').encode()
    resp = d._marshaled_dispatch(req)
    print(isinstance(resp, bytes))
    print(b'methodResponse' in resp)
    print(b'30' in resp)
    print('marshaled_dispatch_success ok')


def test_marshaled_dispatch_fault():
    d = xs.SimpleXMLRPCDispatcher(allow_none=True)
    req = xc.dumps((), 'no_such_method').encode()
    resp = d._marshaled_dispatch(req)
    print(isinstance(resp, bytes))
    print(b'fault' in resp)
    print('marshaled_dispatch_fault ok')


def test_register_introspection():
    d = xs.SimpleXMLRPCDispatcher(allow_none=True)

    def add(x, y): return x + y
    d.register_function(add)
    d.register_introspection_functions()

    methods = d.system_listMethods()
    print('system.listMethods' in methods)
    print('system.methodHelp' in methods)
    print('system.methodSignature' in methods)
    print('add' in methods)
    print('register_introspection ok')


def test_register_multicall():
    d = xs.SimpleXMLRPCDispatcher(allow_none=True)
    d.register_multicall_functions()
    methods = d.system_listMethods()
    print('system.multicall' in methods)
    print('register_multicall ok')


def test_instance_methods_in_listMethods():
    class MyService:
        def greet(self): return 'hi'
        def calculate(self, n): return n + 1
        def _hidden(self): pass

    d = xs.SimpleXMLRPCDispatcher(allow_none=True)
    d.register_instance(MyService())
    methods = d.system_listMethods()
    print('greet' in methods)
    print('calculate' in methods)
    print('_hidden' in methods)
    print('instance_methods_in_listMethods ok')


def test_xmlrpc_doc_generator():
    gen = xs.XMLRPCDocGenerator()
    print(gen.server_name)
    print(gen.server_title)
    print('following methods' in gen.server_documentation)

    gen.set_server_name('My API')
    gen.set_server_title('My Title')
    gen.set_server_documentation('Custom docs.')
    print(gen.server_name)
    print(gen.server_title)
    print(gen.server_documentation)
    print('xmlrpc_doc_generator ok')


def test_server_html_doc():
    print(hasattr(xs, 'ServerHTMLDoc'))
    print(isinstance(xs.ServerHTMLDoc, type))
    print('server_html_doc ok')


def test_doc_request_handler_bases():
    bases = [b.__name__ for b in xs.DocXMLRPCRequestHandler.__bases__]
    print(bases)
    print('SimpleXMLRPCRequestHandler' in bases)
    print('XMLRPCDocGenerator' not in bases)
    print('doc_request_handler_bases ok')


def test_server_exports():
    names = ['SimpleXMLRPCServer', 'SimpleXMLRPCDispatcher',
             'SimpleXMLRPCRequestHandler', 'CGIXMLRPCRequestHandler',
             'MultiPathXMLRPCServer', 'DocXMLRPCServer',
             'DocXMLRPCRequestHandler', 'DocCGIXMLRPCRequestHandler',
             'XMLRPCDocGenerator', 'ServerHTMLDoc',
             'list_public_methods', 'resolve_dotted_attribute',
             'Fault', 'dumps', 'loads']
    for name in names:
        print(name, name in dir(xs))
    print('server_exports ok')


def test_server_hierarchy():
    print(issubclass(xs.SimpleXMLRPCServer, socketserver.TCPServer))
    print(issubclass(xs.SimpleXMLRPCServer, xs.SimpleXMLRPCDispatcher))
    print(issubclass(xs.MultiPathXMLRPCServer, xs.SimpleXMLRPCServer))
    print(issubclass(xs.CGIXMLRPCRequestHandler, xs.SimpleXMLRPCDispatcher))
    print(issubclass(xs.DocXMLRPCServer, xs.SimpleXMLRPCServer))
    print(issubclass(xs.DocXMLRPCServer, xs.XMLRPCDocGenerator))
    print(issubclass(xs.DocCGIXMLRPCRequestHandler, xs.XMLRPCDocGenerator))
    print('server_hierarchy ok')


test_list_public_methods()
test_resolve_dotted_attribute()
test_dispatch_direct()
test_dispatch_instance()
test_marshaled_dispatch_success()
test_marshaled_dispatch_fault()
test_register_introspection()
test_register_multicall()
test_instance_methods_in_listMethods()
test_xmlrpc_doc_generator()
test_server_html_doc()
test_doc_request_handler_bases()
test_server_exports()
test_server_hierarchy()

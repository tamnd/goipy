import http.server
import socketserver


def test_class_attrs_BaseHTTP():
    h = http.server.BaseHTTPRequestHandler
    print(h.server_version)
    print(h.protocol_version)
    print(h.default_request_version)
    print(h.error_content_type)
    print(h.disable_nagle_algorithm)
    print(h.rbufsize)
    print(h.wbufsize)
    print(h.timeout)
    print('base_attrs ok')


def test_sys_version_prefix():
    h = http.server.BaseHTTPRequestHandler
    print(h.sys_version.startswith('Python/'))
    print('sys_version ok')


def test_weekdayname():
    wn = http.server.BaseHTTPRequestHandler.weekdayname
    print(len(wn))
    print(wn[0])
    print(wn[4])
    print(wn[6])
    print('weekdayname ok')


def test_monthname():
    mn = http.server.BaseHTTPRequestHandler.monthname
    print(len(mn))
    print(mn[0])
    print(mn[1])
    print(mn[12])
    print('monthname ok')


def test_responses_200():
    r = http.server.BaseHTTPRequestHandler.responses
    print(r[200])
    print('responses_200 ok')


def test_responses_404():
    r = http.server.BaseHTTPRequestHandler.responses
    print(r[404])
    print('responses_404 ok')


def test_responses_500():
    r = http.server.BaseHTTPRequestHandler.responses
    print(r[500])
    print('responses_500 ok')


def test_responses_len():
    r = http.server.BaseHTTPRequestHandler.responses
    print(len(r) >= 50)
    print('responses_len ok')


def test_class_attrs_SimpleHTTP():
    s = http.server.SimpleHTTPRequestHandler
    print(s.server_version)
    print('simple_attrs ok')


def test_extensions_map():
    em = http.server.SimpleHTTPRequestHandler.extensions_map
    print(em['.gz'])
    print(em['.bz2'])
    print('.xz' in em)
    print('extensions_map ok')


def test_cgi_directories():
    cd = http.server.CGIHTTPRequestHandler.cgi_directories
    print('/cgi-bin' in cd)
    print('/htbin' in cd)
    print('cgi_directories ok')


def test_HTTPServer_allow_reuse():
    print(http.server.HTTPServer.allow_reuse_address)
    print('allow_reuse ok')


def test_hierarchy_HTTPServer():
    print(issubclass(http.server.HTTPServer, socketserver.TCPServer))
    print('http_server_hierarchy ok')


def test_hierarchy_ThreadingHTTPServer():
    print(issubclass(http.server.ThreadingHTTPServer, http.server.HTTPServer))
    print(issubclass(http.server.ThreadingHTTPServer, socketserver.ThreadingMixIn))
    print('threading_http_hierarchy ok')


def test_hierarchy_BaseHTTPHandler():
    print(issubclass(http.server.BaseHTTPRequestHandler,
                     socketserver.StreamRequestHandler))
    print('base_handler_hierarchy ok')


def test_hierarchy_SimpleHTTP():
    print(issubclass(http.server.SimpleHTTPRequestHandler,
                     http.server.BaseHTTPRequestHandler))
    print('simple_hierarchy ok')


def test_hierarchy_CGIHTTP():
    print(issubclass(http.server.CGIHTTPRequestHandler,
                     http.server.BaseHTTPRequestHandler))
    print('cgi_hierarchy ok')


def test_BaseHTTP_methods():
    h = http.server.BaseHTTPRequestHandler
    for name in ['send_response', 'send_header', 'end_headers', 'flush_headers',
                 'send_error', 'log_request', 'log_error', 'log_message',
                 'version_string', 'address_string', 'date_time_string',
                 'log_date_time_string', 'handle_one_request', 'parse_request']:
        print(name, hasattr(h, name))
    print('base_methods ok')


def test_instantiation():
    s = http.server.HTTPServer(('127.0.0.1', 0), http.server.BaseHTTPRequestHandler,
                               bind_and_activate=False)
    print(type(s).__name__)
    print(s.server_address)
    print(s.RequestHandlerClass.__name__)
    print(s.allow_reuse_address)
    s.server_close()
    print('instantiation ok')


def test_DEFAULT_constants():
    print(http.server.DEFAULT_ERROR_CONTENT_TYPE)
    print(type(http.server.DEFAULT_ERROR_MESSAGE).__name__)
    print('%(code)d' in http.server.DEFAULT_ERROR_MESSAGE)
    print('default_constants ok')


def test_module_exports():
    for name in ['HTTPServer', 'ThreadingHTTPServer', 'BaseHTTPRequestHandler',
                 'SimpleHTTPRequestHandler', 'CGIHTTPRequestHandler',
                 'DEFAULT_ERROR_CONTENT_TYPE', 'DEFAULT_ERROR_MESSAGE']:
        print(name, name in dir(http.server))
    print('module_exports ok')


test_class_attrs_BaseHTTP()
test_sys_version_prefix()
test_weekdayname()
test_monthname()
test_responses_200()
test_responses_404()
test_responses_500()
test_responses_len()
test_class_attrs_SimpleHTTP()
test_extensions_map()
test_cgi_directories()
test_HTTPServer_allow_reuse()
test_hierarchy_HTTPServer()
test_hierarchy_ThreadingHTTPServer()
test_hierarchy_BaseHTTPHandler()
test_hierarchy_SimpleHTTP()
test_hierarchy_CGIHTTP()
test_BaseHTTP_methods()
test_instantiation()
test_DEFAULT_constants()
test_module_exports()

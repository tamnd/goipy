import io
import wsgiref.util
import wsgiref.headers
import wsgiref.simple_server
import wsgiref.handlers
import wsgiref.validate


def test_guess_scheme():
    environ = {}
    assert wsgiref.util.guess_scheme(environ) == "http"
    environ["HTTPS"] = "on"
    assert wsgiref.util.guess_scheme(environ) == "https"
    environ["HTTPS"] = "1"
    assert wsgiref.util.guess_scheme(environ) == "https"
    print("guess_scheme ok")


def test_setup_testing_defaults():
    environ = {}
    wsgiref.util.setup_testing_defaults(environ)
    assert "REQUEST_METHOD" in environ
    assert environ["REQUEST_METHOD"] == "GET"
    assert "PATH_INFO" in environ
    assert "SERVER_NAME" in environ
    assert "wsgi.input" in environ
    assert "wsgi.errors" in environ
    assert "wsgi.version" in environ
    assert environ["wsgi.url_scheme"] == "http"
    print("setup_testing_defaults ok")


def test_request_uri():
    environ = {}
    wsgiref.util.setup_testing_defaults(environ)
    uri = wsgiref.util.request_uri(environ)
    assert uri.startswith("http://")
    assert "127.0.0.1" in uri
    print("request_uri ok")


def test_application_uri():
    environ = {}
    wsgiref.util.setup_testing_defaults(environ)
    uri = wsgiref.util.application_uri(environ)
    assert uri.startswith("http://")
    assert "127.0.0.1" in uri
    print("application_uri ok")


def test_shift_path_info():
    environ = {}
    wsgiref.util.setup_testing_defaults(environ)
    environ["PATH_INFO"] = "/foo/bar"
    environ["SCRIPT_NAME"] = ""
    name = wsgiref.util.shift_path_info(environ)
    assert name == "foo"
    assert environ["PATH_INFO"] == "/bar"
    assert environ["SCRIPT_NAME"].endswith("/foo")
    print("shift_path_info ok")


def test_shift_path_info_empty():
    environ = {}
    wsgiref.util.setup_testing_defaults(environ)
    environ["PATH_INFO"] = "/"
    result = wsgiref.util.shift_path_info(environ)
    assert result is None
    print("shift_path_info_empty ok")


def test_is_hop_by_hop():
    assert wsgiref.util.is_hop_by_hop("connection") == True
    assert wsgiref.util.is_hop_by_hop("Connection") == True
    assert wsgiref.util.is_hop_by_hop("transfer-encoding") == True
    assert wsgiref.util.is_hop_by_hop("content-type") == False
    assert wsgiref.util.is_hop_by_hop("content-length") == False
    assert wsgiref.util.is_hop_by_hop("upgrade") == True
    print("is_hop_by_hop ok")


def test_file_wrapper():
    data = b"hello world" * 100
    fileobj = io.BytesIO(data)
    wrapper = wsgiref.util.FileWrapper(fileobj, 16)
    chunks = b"".join(wrapper)
    assert chunks == data
    print("file_wrapper ok")


def test_headers_basic():
    h = wsgiref.headers.Headers()
    assert len(h) == 0
    h["Content-Type"] = "text/html"
    assert h["Content-Type"] == "text/html"
    assert len(h) == 1
    print("headers_basic ok")


def test_headers_case_insensitive():
    h = wsgiref.headers.Headers()
    h["Content-Type"] = "text/html"
    assert h["content-type"] == "text/html"
    assert h["CONTENT-TYPE"] == "text/html"
    assert "content-type" in h
    assert "CONTENT-TYPE" in h
    assert "X-Missing" not in h
    print("headers_case_insensitive ok")


def test_headers_del():
    h = wsgiref.headers.Headers()
    h["X-Foo"] = "bar"
    assert "X-Foo" in h
    del h["X-Foo"]
    assert "X-Foo" not in h
    assert len(h) == 0
    print("headers_del ok")


def test_headers_get():
    h = wsgiref.headers.Headers()
    assert h.get("X-Missing") is None
    assert h.get("X-Missing", "default") == "default"
    h["X-Foo"] = "bar"
    assert h.get("X-Foo") == "bar"
    assert h.get("x-foo") == "bar"
    print("headers_get ok")


def test_headers_get_all():
    h = wsgiref.headers.Headers([("X-Foo", "a"), ("X-Foo", "b"), ("X-Bar", "c")])
    assert h.get_all("X-Foo") == ["a", "b"]
    assert h.get_all("x-foo") == ["a", "b"]
    assert h.get_all("X-Bar") == ["c"]
    assert h.get_all("X-Missing") == []
    print("headers_get_all ok")


def test_headers_add_header():
    h = wsgiref.headers.Headers()
    h.add_header("content-disposition", "attachment", filename="file.txt")
    val = h["content-disposition"]
    assert val is not None
    assert "attachment" in val
    assert "filename" in val
    print("headers_add_header ok")


def test_headers_keys_values_items():
    h = wsgiref.headers.Headers([("A", "1"), ("B", "2"), ("C", "3")])
    assert sorted(h.keys()) == ["A", "B", "C"]
    assert sorted(h.values()) == ["1", "2", "3"]
    items = h.items()
    assert len(items) == 3
    print("headers_keys_values_items ok")


def test_headers_setdefault():
    h = wsgiref.headers.Headers()
    v = h.setdefault("X-Foo", "default")
    assert v == "default"
    assert h["X-Foo"] == "default"
    v2 = h.setdefault("X-Foo", "other")
    assert v2 == "default"
    assert h["X-Foo"] == "default"
    print("headers_setdefault ok")


def test_headers_bytes():
    h = wsgiref.headers.Headers([("Content-Type", "text/html"), ("X-Custom", "value")])
    b = h.__bytes__()
    assert isinstance(b, bytes)
    assert b"Content-Type: text/html" in b
    assert b"X-Custom: value" in b
    print("headers_bytes ok")


def test_demo_app():
    responses = []

    def start_response(status, headers, exc_info=None):
        responses.append((status, headers))
        return lambda data: None

    environ = {}
    wsgiref.util.setup_testing_defaults(environ)
    body = wsgiref.simple_server.demo_app(environ, start_response)
    assert len(responses) == 1
    status, headers = responses[0]
    assert status == "200 OK"
    assert isinstance(body, list)
    assert len(body) > 0
    assert isinstance(body[0], bytes)
    print("demo_app ok")


def test_make_server():
    import socket
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        port = s.getsockname()[1]

    def app(environ, start_response):
        start_response("200 OK", [("Content-Type", "text/plain")])
        return [b"hello"]

    server = wsgiref.simple_server.make_server("127.0.0.1", port, app)
    assert hasattr(server, "set_app")
    assert hasattr(server, "get_app")
    assert server.get_app() is app
    server.server_close()
    print("make_server ok")


def test_simple_handler_run():
    stdin = io.BytesIO(b"")
    stdout = io.BytesIO()
    stderr = io.BytesIO()

    environ = {}
    wsgiref.util.setup_testing_defaults(environ)

    def app(environ, start_response):
        start_response(
            "200 OK",
            [("Content-Type", "text/plain"), ("Content-Length", "5")],
        )
        return [b"hello"]

    handler = wsgiref.handlers.SimpleHandler(stdin, stdout, stderr, environ)
    handler.run(app)

    output = stdout.getvalue()
    assert isinstance(output, bytes)
    assert b"200 OK" in output
    assert b"hello" in output
    print("simple_handler_run ok")


def test_validator():
    def app(environ, start_response):
        start_response(
            "200 OK", [("Content-Type", "text/plain"), ("Content-Length", "5")]
        )
        return [b"hello"]

    wrapped = wsgiref.validate.validator(app)
    assert callable(wrapped)

    responses = []

    def start_response(status, headers, exc_info=None):
        responses.append((status, headers))
        return lambda data: None

    environ = {}
    wsgiref.util.setup_testing_defaults(environ)
    result = wrapped(environ, start_response)
    assert len(responses) == 1
    status, headers = responses[0]
    assert status == "200 OK"
    assert isinstance(result, list)
    print("validator ok")


test_guess_scheme()
test_setup_testing_defaults()
test_request_uri()
test_application_uri()
test_shift_path_info()
test_shift_path_info_empty()
test_is_hop_by_hop()
test_file_wrapper()
test_headers_basic()
test_headers_case_insensitive()
test_headers_del()
test_headers_get()
test_headers_get_all()
test_headers_add_header()
test_headers_keys_values_items()
test_headers_setdefault()
test_headers_bytes()
test_demo_app()
test_make_server()
test_simple_handler_run()
test_validator()

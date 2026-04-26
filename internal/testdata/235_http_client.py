import http.client
import io


def test_constants():
    assert http.client.HTTP_PORT == 80
    assert http.client.HTTPS_PORT == 443
    assert http.client._MAXLINE == 65536
    assert http.client._MAXHEADERS == 100
    print("constants ok")


def test_exception_hierarchy_base():
    assert issubclass(http.client.HTTPException, Exception)
    assert issubclass(http.client.NotConnected, http.client.HTTPException)
    assert issubclass(http.client.InvalidURL, http.client.HTTPException)
    assert issubclass(http.client.UnknownProtocol, http.client.HTTPException)
    assert issubclass(http.client.UnknownTransferEncoding, http.client.HTTPException)
    assert issubclass(http.client.UnimplementedFileMode, http.client.HTTPException)
    assert issubclass(http.client.IncompleteRead, http.client.HTTPException)
    assert issubclass(http.client.ImproperConnectionState, http.client.HTTPException)
    assert issubclass(http.client.BadStatusLine, http.client.HTTPException)
    assert issubclass(http.client.LineTooLong, http.client.HTTPException)
    print("exception_hierarchy_base ok")


def test_exception_hierarchy_improper():
    assert issubclass(http.client.CannotSendRequest, http.client.ImproperConnectionState)
    assert issubclass(http.client.CannotSendHeader, http.client.ImproperConnectionState)
    assert issubclass(http.client.ResponseNotReady, http.client.ImproperConnectionState)
    assert issubclass(http.client.CannotSendRequest, http.client.HTTPException)
    assert issubclass(http.client.CannotSendHeader, http.client.HTTPException)
    assert issubclass(http.client.ResponseNotReady, http.client.HTTPException)
    print("exception_hierarchy_improper ok")


def test_exception_hierarchy_remote():
    assert issubclass(http.client.RemoteDisconnected, http.client.BadStatusLine)
    assert issubclass(http.client.RemoteDisconnected, http.client.HTTPException)
    assert issubclass(http.client.RemoteDisconnected, ConnectionResetError)
    assert issubclass(http.client.RemoteDisconnected, OSError)
    print("exception_hierarchy_remote ok")


def test_unknown_protocol():
    try:
        raise http.client.UnknownProtocol("HTTP/0.1")
    except http.client.UnknownProtocol as e:
        assert e.version == "HTTP/0.1", repr(e.version)
        assert isinstance(e, http.client.HTTPException)
    print("unknown_protocol ok")


def test_incomplete_read_no_expected():
    try:
        raise http.client.IncompleteRead(b"partial data")
    except http.client.IncompleteRead as e:
        assert e.partial == b"partial data", repr(e.partial)
        assert e.expected is None
        r = repr(e)
        assert "IncompleteRead" in r
        assert "12" in r  # len(b"partial data")
        assert "more expected" not in r
    print("incomplete_read_no_expected ok")


def test_incomplete_read_with_expected():
    try:
        raise http.client.IncompleteRead(b"abc", 100)
    except http.client.IncompleteRead as e:
        assert e.partial == b"abc"
        assert e.expected == 100
        r = repr(e)
        assert "3" in r   # len(b"abc")
        assert "100" in r
        assert "more expected" in r
    print("incomplete_read_with_expected ok")


def test_bad_status_line():
    try:
        raise http.client.BadStatusLine("HTTP/0.9 garbage")
    except http.client.BadStatusLine as e:
        assert e.line == "HTTP/0.9 garbage", repr(e.line)
        assert isinstance(e, http.client.HTTPException)
    print("bad_status_line ok")


def test_line_too_long():
    try:
        raise http.client.LineTooLong("header line")
    except http.client.LineTooLong as e:
        s = str(e)
        assert "65536" in s, repr(s)
        assert "header line" in s, repr(s)
        assert isinstance(e, http.client.HTTPException)
    print("line_too_long ok")


def test_remote_disconnected():
    try:
        raise http.client.RemoteDisconnected("server closed connection")
    except http.client.RemoteDisconnected as e:
        assert str(e) == "server closed connection", repr(str(e))
        assert isinstance(e, http.client.BadStatusLine)
        assert isinstance(e, http.client.HTTPException)
        assert isinstance(e, ConnectionResetError)
        assert isinstance(e, OSError)
    print("remote_disconnected ok")


def test_remote_disconnected_catch_as_oserror():
    try:
        raise http.client.RemoteDisconnected("lost connection")
    except OSError:
        pass
    else:
        assert False, "should have been caught as OSError"
    print("remote_disconnected_catch_as_oserror ok")


def test_parse_headers_basic():
    raw = b"Content-Type: text/html\r\nContent-Length: 42\r\n\r\n"
    msg = http.client.parse_headers(io.BytesIO(raw))
    assert isinstance(msg, http.client.HTTPMessage)
    assert msg["Content-Type"] == "text/html"
    assert msg["Content-Length"] == "42"
    print("parse_headers_basic ok")


def test_parse_headers_case_insensitive():
    raw = b"Content-Type: text/html\r\nX-Custom: value\r\n\r\n"
    msg = http.client.parse_headers(io.BytesIO(raw))
    assert msg["content-type"] == "text/html"
    assert msg["CONTENT-TYPE"] == "text/html"
    assert msg["x-custom"] == "value"
    print("parse_headers_case_insensitive ok")


def test_parse_headers_get():
    raw = b"Content-Type: text/html\r\nX-Missing: no\r\n\r\n"
    msg = http.client.parse_headers(io.BytesIO(raw))
    assert msg.get("Content-Type") == "text/html"
    assert msg.get("X-Missing") == "no"
    assert msg.get("X-Nonexistent") is None
    assert msg.get("X-Nonexistent", "default") == "default"
    print("parse_headers_get ok")


def test_parse_headers_get_all():
    raw = b"X-Custom: foo\r\nX-Custom: bar\r\nContent-Type: text/html\r\n\r\n"
    msg = http.client.parse_headers(io.BytesIO(raw))
    all_vals = msg.get_all("X-Custom")
    assert all_vals is not None
    assert "foo" in all_vals
    assert "bar" in all_vals
    assert len(all_vals) == 2
    missing = msg.get_all("X-Missing")
    assert missing is None
    print("parse_headers_get_all ok")


def test_parse_headers_get_content_type():
    raw = b"Content-Type: text/html; charset=utf-8\r\n\r\n"
    msg = http.client.parse_headers(io.BytesIO(raw))
    ct = msg.get_content_type()
    assert ct == "text/html", repr(ct)
    print("parse_headers_get_content_type ok")


def test_parse_headers_items_keys_values():
    raw = b"Content-Type: text/html\r\nContent-Length: 100\r\n\r\n"
    msg = http.client.parse_headers(io.BytesIO(raw))
    keys = msg.keys()
    assert "Content-Type" in keys
    assert "Content-Length" in keys
    vals = msg.values()
    assert "text/html" in vals
    assert "100" in vals
    items = msg.items()
    assert ("Content-Type", "text/html") in items
    assert ("Content-Length", "100") in items
    print("parse_headers_items_keys_values ok")


def test_parse_headers_contains_and_len():
    raw = b"Content-Type: text/html\r\nContent-Length: 42\r\nX-Custom: val\r\n\r\n"
    msg = http.client.parse_headers(io.BytesIO(raw))
    assert "Content-Type" in msg
    assert "content-type" in msg
    assert "X-Missing" not in msg
    assert len(msg) == 3
    print("parse_headers_contains_and_len ok")


def test_http_connection_construction():
    conn = http.client.HTTPConnection("example.com")
    assert conn.host == "example.com"
    assert conn.port == 80
    print("http_connection_construction ok")


def test_http_connection_with_port():
    conn = http.client.HTTPConnection("example.com", 8080)
    assert conn.host == "example.com"
    assert conn.port == 8080
    print("http_connection_with_port ok")


def test_http_connection_with_timeout():
    conn = http.client.HTTPConnection("example.com", timeout=30)
    assert conn.timeout == 30
    print("http_connection_with_timeout ok")


def test_http_connection_default_port():
    assert http.client.HTTPConnection.default_port == 80
    print("http_connection_default_port ok")


def test_http_connection_set_debuglevel():
    conn = http.client.HTTPConnection("example.com")
    assert conn.debuglevel == 0
    conn.set_debuglevel(2)
    assert conn.debuglevel == 2
    print("http_connection_set_debuglevel ok")


def test_http_connection_set_tunnel():
    conn = http.client.HTTPConnection("proxy.example.com")
    conn.set_tunnel("target.example.com", 443)
    assert conn._tunnel_host == "target.example.com"
    print("http_connection_set_tunnel ok")


def test_http_connection_set_tunnel_with_headers():
    conn = http.client.HTTPConnection("proxy.example.com")
    conn.set_tunnel("target.example.com", 443, {"Authorization": "Basic abc"})
    assert conn._tunnel_host == "target.example.com"
    print("http_connection_set_tunnel_with_headers ok")


def test_http_connection_no_ops():
    conn = http.client.HTTPConnection("example.com")
    conn.connect()
    conn.close()
    conn.send(b"data")
    # putrequest → putheader → endheaders in correct order
    conn2 = http.client.HTTPConnection("example.com")
    conn2.putrequest("GET", "/path")
    conn2.putheader("Content-Type", "text/plain")
    conn2.endheaders()
    print("http_connection_no_ops ok")


def test_http_connection_getresponse_raises():
    conn = http.client.HTTPConnection("example.com")
    try:
        conn.getresponse()
        assert False, "should raise ResponseNotReady"
    except http.client.ResponseNotReady:
        pass
    print("http_connection_getresponse_raises ok")


def test_https_connection_default_port():
    assert http.client.HTTPSConnection.default_port == 443
    conn = http.client.HTTPSConnection("example.com")
    assert conn.host == "example.com"
    assert conn.port == 443
    print("https_connection_default_port ok")


def test_https_connection_is_subclass():
    assert issubclass(http.client.HTTPSConnection, http.client.HTTPConnection)
    conn = http.client.HTTPSConnection("example.com")
    assert isinstance(conn, http.client.HTTPConnection)
    print("https_connection_is_subclass ok")


def test_responses_dict():
    assert http.client.responses[200] == "OK"
    assert http.client.responses[404] == "Not Found"
    assert http.client.responses[500] == "Internal Server Error"
    assert http.client.responses[301] == "Moved Permanently"
    assert 200 in http.client.responses
    assert 999 not in http.client.responses
    assert http.client.responses.get(999) is None
    print("responses_dict ok")


def test_status_int_exports():
    assert http.client.OK == 200
    assert http.client.NOT_FOUND == 404
    assert http.client.INTERNAL_SERVER_ERROR == 500
    assert http.client.CONTINUE == 100
    assert http.client.MOVED_PERMANENTLY == 301
    assert http.client.UNAUTHORIZED == 401
    assert http.client.FORBIDDEN == 403
    assert http.client.BAD_REQUEST == 400
    assert http.client.CREATED == 201
    assert http.client.NO_CONTENT == 204
    print("status_int_exports ok")


def test_status_int_exports_aliases():
    assert http.client.RANGE_NOT_SATISFIABLE == 416
    assert http.client.REQUESTED_RANGE_NOT_SATISFIABLE == 416
    assert http.client.UNPROCESSABLE_ENTITY == 422
    assert http.client.UNPROCESSABLE_CONTENT == 422
    print("status_int_exports_aliases ok")


def test_module_exports():
    assert http.client.HTTPException is not None
    assert http.client.HTTPConnection is not None
    assert http.client.HTTPSConnection is not None
    assert http.client.HTTPMessage is not None
    assert callable(http.client.parse_headers)
    assert isinstance(http.client.responses, dict)
    assert http.client.error is http.client.HTTPException
    print("module_exports ok")


test_constants()
test_exception_hierarchy_base()
test_exception_hierarchy_improper()
test_exception_hierarchy_remote()
test_unknown_protocol()
test_incomplete_read_no_expected()
test_incomplete_read_with_expected()
test_bad_status_line()
test_line_too_long()
test_remote_disconnected()
test_remote_disconnected_catch_as_oserror()
test_parse_headers_basic()
test_parse_headers_case_insensitive()
test_parse_headers_get()
test_parse_headers_get_all()
test_parse_headers_get_content_type()
test_parse_headers_items_keys_values()
test_parse_headers_contains_and_len()
test_http_connection_construction()
test_http_connection_with_port()
test_http_connection_with_timeout()
test_http_connection_default_port()
test_http_connection_set_debuglevel()
test_http_connection_set_tunnel()
test_http_connection_set_tunnel_with_headers()
test_http_connection_no_ops()
test_http_connection_getresponse_raises()
test_https_connection_default_port()
test_https_connection_is_subclass()
test_responses_dict()
test_status_int_exports()
test_status_int_exports_aliases()
test_module_exports()

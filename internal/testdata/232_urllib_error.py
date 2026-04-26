import urllib.error
import io


def test_urlerror_reason_str():
    e = urllib.error.URLError("connection refused")
    assert e.reason == "connection refused", repr(e.reason)
    assert str(e) == "<urlopen error connection refused>", repr(str(e))
    print("urlerror_reason_str ok")


def test_urlerror_reason_exc():
    inner = ConnectionRefusedError(111, "Connection refused")
    e = urllib.error.URLError(inner)
    assert e.reason is inner, repr(e.reason)
    s = str(e)
    assert "Connection refused" in s, repr(s)
    print("urlerror_reason_exc ok")


def test_urlerror_args():
    e = urllib.error.URLError("timeout")
    assert isinstance(e.args, tuple), type(e.args)
    assert len(e.args) == 1, repr(e.args)
    assert e.args[0] == "timeout", repr(e.args[0])
    print("urlerror_args ok")


def test_urlerror_repr():
    e = urllib.error.URLError("bad url")
    r = repr(e)
    assert "URLError" in r, repr(r)
    assert "bad url" in r, repr(r)
    print("urlerror_repr ok")


def test_urlerror_hierarchy():
    assert issubclass(urllib.error.URLError, OSError)
    e = urllib.error.URLError("test")
    assert isinstance(e, OSError)
    print("urlerror_hierarchy ok")


def test_urlerror_catchable_as_oserror():
    try:
        raise urllib.error.URLError("network error")
    except OSError as e:
        assert "network error" in str(e), repr(str(e))
    print("urlerror_catchable_as_oserror ok")


def test_httperror_attrs():
    hdrs = {"Content-Type": "text/html"}
    fp = io.BytesIO(b"error body")
    e = urllib.error.HTTPError("http://example.com/page", 404, "Not Found", hdrs, fp)
    assert e.url == "http://example.com/page", repr(e.url)
    assert e.filename == "http://example.com/page", repr(e.filename)
    assert e.code == 404, repr(e.code)
    assert e.msg == "Not Found", repr(e.msg)
    assert e.reason == "Not Found", repr(e.reason)
    assert e.hdrs is hdrs, repr(e.hdrs)
    assert e.headers is hdrs, repr(e.headers)
    assert e.fp is fp, repr(e.fp)
    print("httperror_attrs ok")


def test_httperror_str():
    e = urllib.error.HTTPError("http://example.com", 404, "Not Found", {}, None)
    s = str(e)
    assert s == "HTTP Error 404: Not Found", repr(s)
    print("httperror_str ok")


def test_httperror_repr():
    e = urllib.error.HTTPError("http://example.com", 404, "Not Found", {}, None)
    r = repr(e)
    assert r == "<HTTPError 404: 'Not Found'>", repr(r)
    print("httperror_repr ok")


def test_httperror_read():
    body = b"Internal Server Error body"
    fp = io.BytesIO(body)
    e = urllib.error.HTTPError("http://example.com", 500, "Internal Server Error", {}, fp)
    data = e.read()
    assert data == body, repr(data)
    print("httperror_read ok")


def test_httperror_read_nbytes():
    fp = io.BytesIO(b"Hello World")
    e = urllib.error.HTTPError("http://example.com", 200, "OK", {}, fp)
    chunk = e.read(5)
    assert chunk == b"Hello", repr(chunk)
    rest = e.read()
    assert rest == b" World", repr(rest)
    print("httperror_read_nbytes ok")


def test_httperror_getcode():
    e = urllib.error.HTTPError("http://example.com", 403, "Forbidden", {}, None)
    assert e.getcode() == 403, repr(e.getcode())
    print("httperror_getcode ok")


def test_httperror_geturl():
    e = urllib.error.HTTPError("http://example.com/path", 404, "Not Found", {}, None)
    assert e.geturl() == "http://example.com/path", repr(e.geturl())
    print("httperror_geturl ok")


def test_httperror_info():
    hdrs = {"X-Custom": "value"}
    e = urllib.error.HTTPError("http://example.com", 200, "OK", hdrs, None)
    assert e.info() is hdrs, repr(e.info())
    print("httperror_info ok")


def test_httperror_close():
    fp = io.BytesIO(b"body")
    e = urllib.error.HTTPError("http://example.com", 200, "OK", {}, fp)
    e.close()
    print("httperror_close ok")


def test_httperror_context_manager():
    body = b"error response"
    fp = io.BytesIO(body)
    e = urllib.error.HTTPError("http://example.com", 503, "Service Unavailable", {}, fp)
    with e as resp:
        data = resp.read()
    assert data == body, repr(data)
    print("httperror_context_manager ok")


def test_httperror_hierarchy():
    e = urllib.error.HTTPError("http://example.com", 404, "Not Found", {}, None)
    assert isinstance(e, urllib.error.URLError)
    assert isinstance(e, OSError)
    assert issubclass(urllib.error.HTTPError, urllib.error.URLError)
    assert issubclass(urllib.error.HTTPError, OSError)
    print("httperror_hierarchy ok")


def test_httperror_as_urlerror():
    try:
        raise urllib.error.HTTPError("http://example.com", 500, "Server Error", {}, None)
    except urllib.error.URLError as e:
        assert e.code == 500, repr(e.code)
    print("httperror_as_urlerror ok")


def test_httperror_as_oserror():
    try:
        raise urllib.error.HTTPError("http://example.com", 401, "Unauthorized", {}, None)
    except OSError as e:
        assert "401" in str(e) or "Unauthorized" in str(e), repr(str(e))
    print("httperror_as_oserror ok")


def test_contenttoo_short_attrs():
    e = urllib.error.ContentTooShortError("retrieved too short", b"partial content")
    assert e.reason == "retrieved too short", repr(e.reason)
    assert e.content == b"partial content", repr(e.content)
    print("contenttoo_short_attrs ok")


def test_contenttoo_short_str():
    e = urllib.error.ContentTooShortError("only 100 bytes", b"data")
    s = str(e)
    assert "only 100 bytes" in s, repr(s)
    assert s.startswith("<urlopen error"), repr(s)
    print("contenttoo_short_str ok")


def test_contenttoo_short_hierarchy():
    e = urllib.error.ContentTooShortError("msg", b"")
    assert isinstance(e, urllib.error.URLError)
    assert isinstance(e, OSError)
    assert issubclass(urllib.error.ContentTooShortError, urllib.error.URLError)
    assert issubclass(urllib.error.ContentTooShortError, OSError)
    print("contenttoo_short_hierarchy ok")


def test_contenttoo_short_catchable():
    try:
        raise urllib.error.ContentTooShortError("too short", b"partial")
    except urllib.error.URLError as e:
        assert e.content == b"partial", repr(e.content)
    try:
        raise urllib.error.ContentTooShortError("too short", b"data")
    except OSError:
        pass
    print("contenttoo_short_catchable ok")


def test_exports():
    assert urllib.error.URLError is not None
    assert urllib.error.HTTPError is not None
    assert urllib.error.ContentTooShortError is not None
    assert issubclass(urllib.error.HTTPError, urllib.error.URLError)
    assert issubclass(urllib.error.ContentTooShortError, urllib.error.URLError)
    print("exports ok")


test_urlerror_reason_str()
test_urlerror_reason_exc()
test_urlerror_args()
test_urlerror_repr()
test_urlerror_hierarchy()
test_urlerror_catchable_as_oserror()
test_httperror_attrs()
test_httperror_str()
test_httperror_repr()
test_httperror_read()
test_httperror_read_nbytes()
test_httperror_getcode()
test_httperror_geturl()
test_httperror_info()
test_httperror_close()
test_httperror_context_manager()
test_httperror_hierarchy()
test_httperror_as_urlerror()
test_httperror_as_oserror()
test_contenttoo_short_attrs()
test_contenttoo_short_str()
test_contenttoo_short_hierarchy()
test_contenttoo_short_catchable()
test_exports()

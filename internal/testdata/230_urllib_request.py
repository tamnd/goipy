import urllib.request
import urllib.error
import os
import tempfile


def test_request_full_url_setter():
    req = urllib.request.Request("http://example.com/path?q=1")
    assert req.full_url == "http://example.com/path?q=1", repr(req.full_url)
    assert req.type == "http", repr(req.type)
    assert req.host == "example.com", repr(req.host)
    assert req.selector == "/path?q=1", repr(req.selector)
    req2 = urllib.request.Request("https://api.example.com/v1")
    assert req2.type == "https", repr(req2.type)
    assert req2.host == "api.example.com", repr(req2.host)
    assert req2.selector == "/v1", repr(req2.selector)
    print("request_full_url ok")


def test_request_data_setter():
    req = urllib.request.Request("http://example.com/")
    assert req.get_method() == "GET", req.get_method()
    req.data = b"hello"
    assert req.get_method() == "POST", req.get_method()
    req.data = None
    assert req.get_method() == "GET", req.get_method()
    req2 = urllib.request.Request("http://example.com/", data=b"world")
    assert req2.get_method() == "POST", req2.get_method()
    print("request_data_setter ok")


def test_request_origin_req_host():
    req = urllib.request.Request("http://example.com/some/path")
    assert req.origin_req_host == "example.com", repr(req.origin_req_host)
    print("request_origin_req_host ok")


def test_request_unverifiable():
    req = urllib.request.Request("http://example.com/")
    assert req.unverifiable == False, repr(req.unverifiable)
    req2 = urllib.request.Request("http://example.com/", unverifiable=True)
    assert req2.unverifiable == True, repr(req2.unverifiable)
    print("request_unverifiable ok")


def test_request_set_proxy():
    req = urllib.request.Request("http://example.com/path")
    req.set_proxy("proxy.corp:8080", "http")
    assert req.host == "proxy.corp:8080", repr(req.host)
    assert req.type == "http", repr(req.type)
    print("request_set_proxy ok")


def test_request_unredirected_headers():
    req = urllib.request.Request("http://example.com/")
    req.add_unredirected_header("Authorization", "Bearer token123")
    assert req.has_header("Authorization"), "should find unredirected header"
    val = req.get_header("Authorization")
    assert val == "Bearer token123", repr(val)
    req.add_header("Content-type", "application/json")
    assert req.has_header("Content-type"), "should find regular header"
    print("request_unredirected_headers ok")


def test_request_header_items():
    req = urllib.request.Request("http://example.com/")
    req.add_header("Accept", "application/json")
    req.add_unredirected_header("X-secret", "hidden")
    items = req.header_items()
    keys = [k for k, v in items]
    assert "Accept" in keys, repr(keys)
    assert "X-secret" in keys, repr(keys)
    assert len(items) == 2, repr(items)
    print("request_header_items ok")


def test_password_mgr_basic():
    mgr = urllib.request.HTTPPasswordMgr()
    mgr.add_password("Test Realm", "http://example.com/", "alice", "secret")
    u, p = mgr.find_user_password("Test Realm", "http://example.com/path")
    assert u == "alice", repr(u)
    assert p == "secret", repr(p)
    # non-matching prefix
    u2, p2 = mgr.find_user_password("Test Realm", "http://other.com/")
    assert u2 is None, repr(u2)
    assert p2 is None, repr(p2)
    print("password_mgr_basic ok")


def test_password_mgr_default_realm():
    mgr = urllib.request.HTTPPasswordMgrWithDefaultRealm()
    mgr.add_password(None, "http://example.com/", "bob", "pass2")
    # exact realm lookup fails, falls back to None-keyed entry
    u, p = mgr.find_user_password("Any Realm", "http://example.com/page")
    assert u == "bob", repr(u)
    assert p == "pass2", repr(p)
    # exact realm also works when entry matches
    mgr.add_password("Exact", "http://api.com/", "charlie", "pw3")
    u2, p2 = mgr.find_user_password("Exact", "http://api.com/resource")
    assert u2 == "charlie", repr(u2)
    print("password_mgr_default_realm ok")


def test_password_mgr_prior_auth():
    mgr = urllib.request.HTTPPasswordMgrWithPriorAuth()
    mgr.add_password("Realm", "http://example.com/", "dave", "pw4")
    assert not mgr.is_authenticated("http://example.com/")
    mgr.update_authenticated("http://example.com/", True)
    assert mgr.is_authenticated("http://example.com/")
    mgr.update_authenticated("http://example.com/", False)
    assert not mgr.is_authenticated("http://example.com/")
    print("password_mgr_prior_auth ok")


def test_http_basic_auth_handler():
    mgr = urllib.request.HTTPPasswordMgr()
    mgr.add_password("My Realm", "http://example.com/", "user1", "pass1")
    handler = urllib.request.HTTPBasicAuthHandler(mgr)
    assert handler.passwd is mgr
    u, p = mgr.find_user_password("My Realm", "http://example.com/protected")
    assert u == "user1", repr(u)
    assert p == "pass1", repr(p)
    print("http_basic_auth_handler ok")


def test_http_default_error_handler():
    handler = urllib.request.HTTPDefaultErrorHandler()
    req = urllib.request.Request("http://example.com/notfound")
    try:
        handler.http_error_default(req, None, 404, "Not Found", {})
        assert False, "expected HTTPError"
    except urllib.error.HTTPError as e:
        pass
    print("http_default_error_handler ok")


def test_http_redirect_handler():
    handler = urllib.request.HTTPRedirectHandler()
    req = urllib.request.Request("http://example.com/old")
    new_req = handler.redirect_request(req, None, 301, "Moved", {}, "http://example.com/new")
    assert new_req is not None
    assert new_req.full_url == "http://example.com/new", repr(new_req.full_url)
    assert new_req.unverifiable == True, repr(new_req.unverifiable)
    print("http_redirect_handler ok")


def test_http_cookie_processor():
    proc = urllib.request.HTTPCookieProcessor()
    assert proc.cookiejar is None
    proc2 = urllib.request.HTTPCookieProcessor(cookiejar="fake-jar")
    assert proc2.cookiejar == "fake-jar", repr(proc2.cookiejar)
    # http_request passes request through
    req = urllib.request.Request("http://example.com/")
    result = proc.http_request(req)
    assert result is req
    print("http_cookie_processor ok")


def test_proxy_handler():
    ph = urllib.request.ProxyHandler({"http": "http://proxy:3128"})
    assert ph.proxies is not None
    ph2 = urllib.request.ProxyHandler()
    assert ph2.proxies is not None
    print("proxy_handler ok")


def test_file_handler():
    f = tempfile.NamedTemporaryFile(delete=False, suffix=".txt")
    f.write(b"file content here")
    f.close()
    try:
        req = urllib.request.Request("file://" + f.name)
        handler = urllib.request.FileHandler()
        resp = handler.file_open(req)
        data = resp.read()
        assert data == b"file content here", repr(data)
        print("file_handler ok")
    finally:
        os.unlink(f.name)


def test_data_handler():
    handler = urllib.request.DataHandler()
    req = urllib.request.Request("data:text/plain,hello%20world")
    resp = handler.data_open(req)
    data = resp.read()
    assert data == b"hello world", repr(data)
    ct = resp.headers.get("Content-Type") or resp.headers.get("content-type")
    assert ct == "text/plain", repr(ct)
    print("data_handler ok")


def test_abstract_http_handler():
    handler = urllib.request.AbstractHTTPHandler()
    req = urllib.request.Request("http://example.com/post")
    req.data = b"body data"
    result = handler.do_request_(req)
    ct = result.get_header("Content-type")
    assert ct == "application/x-www-form-urlencoded", repr(ct)
    cl = result.get_header("Content-length")
    assert cl == "9", repr(cl)
    print("abstract_http_handler ok")


def test_http_handler_do_request():
    handler = urllib.request.HTTPHandler()
    req = urllib.request.Request("http://example.com/")
    req2 = handler.do_request_(req)
    assert req2 is req  # no data, no changes
    req3 = urllib.request.Request("http://example.com/api")
    req3.data = b"key=val"
    req4 = handler.do_request_(req3)
    assert req4.get_header("Content-type") == "application/x-www-form-urlencoded"
    print("http_handler_do_request ok")


def test_opener_director_add_handler():
    opener = urllib.request.OpenerDirector()
    h1 = urllib.request.DataHandler()
    opener.add_handler(h1)
    h2 = urllib.request.UnknownHandler()
    opener.add_handler(h2)
    print("opener_director_add_handler ok")


def test_opener_director_open_data():
    opener = urllib.request.build_opener()
    resp = opener.open("data:text/plain,hello%20world")
    data = resp.read()
    assert data == b"hello world", repr(data)
    print("opener_director_open_data ok")


def test_opener_director_error():
    opener = urllib.request.build_opener()
    try:
        opener.error("http", None, None, 404, "Not Found", {})
        assert False, "expected HTTPError"
    except urllib.error.HTTPError:
        pass
    print("opener_director_error ok")


def test_urlretrieve():
    fname, hdrs = urllib.request.urlretrieve("data:text/plain,hello%20retrieve")
    try:
        with open(fname, "rb") as f:
            content = f.read()
        assert content == b"hello retrieve", repr(content)
        assert isinstance(fname, str)
        assert isinstance(hdrs, dict)
    finally:
        os.unlink(fname)
    print("urlretrieve ok")


def test_urlcleanup():
    fname, _ = urllib.request.urlretrieve("data:text/plain,cleanup-test")
    assert os.path.exists(fname), "temp file should exist before cleanup"
    urllib.request.urlcleanup()
    assert not os.path.exists(fname), "temp file should be gone after urlcleanup"
    print("urlcleanup ok")


test_request_full_url_setter()
test_request_data_setter()
test_request_origin_req_host()
test_request_unverifiable()
test_request_set_proxy()
test_request_unredirected_headers()
test_request_header_items()
test_password_mgr_basic()
test_password_mgr_default_realm()
test_password_mgr_prior_auth()
test_http_basic_auth_handler()
test_http_default_error_handler()
test_http_redirect_handler()
test_http_cookie_processor()
test_proxy_handler()
test_file_handler()
test_data_handler()
test_abstract_http_handler()
test_http_handler_do_request()
test_opener_director_add_handler()
test_opener_director_open_data()
test_opener_director_error()
test_urlretrieve()
test_urlcleanup()

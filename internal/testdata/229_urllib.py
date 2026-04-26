import urllib.parse
import urllib.error
import urllib.request
import urllib.robotparser


def test_urldefrag():
    r = urllib.parse.urldefrag("http://example.com/path#section")
    assert r.url == "http://example.com/path", repr(r.url)
    assert r.fragment == "section", repr(r.fragment)
    r2 = urllib.parse.urldefrag("http://example.com/nofrag")
    assert r2.url == "http://example.com/nofrag"
    assert r2.fragment == ""
    print("urldefrag ok")


def test_quote_from_bytes():
    b = b"hello world"
    s = urllib.parse.quote_from_bytes(b, safe="")
    assert s == "hello%20world", repr(s)
    b2 = b"/path/to"
    s2 = urllib.parse.quote_from_bytes(b2)
    assert s2 == "/path/to", repr(s2)
    s3 = urllib.parse.quote_from_bytes(b2, safe="")
    assert s3 == "%2Fpath%2Fto", repr(s3)
    print("quote_from_bytes ok")


def test_unquote_to_bytes():
    b = urllib.parse.unquote_to_bytes("hello%20world")
    assert b == b"hello world", repr(b)
    b2 = urllib.parse.unquote_to_bytes("%2Fpath%2Fto")
    assert b2 == b"/path/to", repr(b2)
    print("unquote_to_bytes ok")


def test_unwrap():
    assert urllib.parse.unwrap("<http://example.com>") == "http://example.com"
    assert urllib.parse.unwrap("<URL:http://example.com>") == "http://example.com"
    assert urllib.parse.unwrap("http://example.com") == "http://example.com"
    print("unwrap ok")


def test_url_error():
    try:
        raise urllib.error.URLError("connection refused")
    except urllib.error.URLError as e:
        assert "connection refused" in str(e), repr(str(e))
    print("URLError ok")


def test_http_error():
    e = urllib.error.HTTPError("http://example.com", 404, "Not Found", {}, None)
    assert e.code == 404
    assert e.url == "http://example.com"
    assert e.reason == "Not Found"
    assert "404" in str(e)
    assert "Not Found" in str(e)
    print("HTTPError ok")


def test_content_too_short_error():
    e = urllib.error.ContentTooShortError("retrieved too short", b"partial")
    assert e.content == b"partial"
    print("ContentTooShortError ok")


def test_request_class():
    req = urllib.request.Request("http://example.com/path?q=1")
    assert req.full_url == "http://example.com/path?q=1"
    assert req.type == "http"
    assert req.host == "example.com"
    assert req.get_method() == "GET"
    assert req.get_full_url() == "http://example.com/path?q=1"
    print("Request class ok")


def test_request_post():
    req = urllib.request.Request("http://example.com/post", data=b"key=value")
    assert req.get_method() == "POST"
    print("Request POST ok")


def test_request_headers():
    req = urllib.request.Request("http://example.com", headers={"Content-Type": "application/json"})
    assert req.has_header("Content-type")
    req.add_header("Accept", "text/html")
    assert req.has_header("Accept")
    assert req.get_header("Accept") == "text/html"
    req.remove_header("Accept")
    assert not req.has_header("Accept")
    print("Request headers ok")


def test_request_method():
    req = urllib.request.Request("http://example.com", method="DELETE")
    assert req.get_method() == "DELETE"
    print("Request method ok")


def test_build_opener():
    opener = urllib.request.build_opener()
    assert opener is not None
    print("build_opener ok")


def test_install_opener():
    opener = urllib.request.build_opener()
    urllib.request.install_opener(opener)
    print("install_opener ok")


def test_urlopen_data():
    resp = urllib.request.urlopen("data:text/plain,hello%20world")
    body = resp.read()
    assert body == b"hello world", repr(body)
    assert resp.status == 200
    assert resp.code == 200
    print("urlopen data URI ok")


def test_urlopen_data_base64():
    import base64
    encoded = base64.b64encode(b"binary data").decode()
    resp = urllib.request.urlopen("data:application/octet-stream;base64," + encoded)
    body = resp.read()
    assert body == b"binary data", repr(body)
    print("urlopen data URI base64 ok")


def test_robotparser_basic():
    rfp = urllib.robotparser.RobotFileParser()
    lines = [
        "User-agent: *",
        "Disallow: /private/",
        "Allow: /public/",
        "",
    ]
    rfp.parse(lines)
    assert rfp.can_fetch("*", "/public/page") == True
    assert rfp.can_fetch("*", "/private/secret") == False
    assert rfp.can_fetch("*", "/other/page") == True
    print("robotparser basic ok")


def test_robotparser_crawl_delay():
    rfp = urllib.robotparser.RobotFileParser()
    lines = [
        "User-agent: mybot",
        "Disallow: /blocked/",
        "Crawl-delay: 10",
        "",
    ]
    rfp.parse(lines)
    assert rfp.crawl_delay("mybot") == 10
    assert rfp.crawl_delay("otherbot") is None
    print("robotparser crawl_delay ok")


def test_robotparser_site_maps():
    rfp = urllib.robotparser.RobotFileParser()
    lines = [
        "User-agent: *",
        "Disallow:",
        "Sitemap: http://example.com/sitemap.xml",
        "Sitemap: http://example.com/sitemap2.xml",
        "",
    ]
    rfp.parse(lines)
    sitemaps = rfp.site_maps()
    assert sitemaps is not None
    assert "http://example.com/sitemap.xml" in sitemaps
    assert "http://example.com/sitemap2.xml" in sitemaps
    print("robotparser site_maps ok")


test_urldefrag()
test_quote_from_bytes()
test_unquote_to_bytes()
test_unwrap()
test_url_error()
test_http_error()
test_content_too_short_error()
test_request_class()
test_request_post()
test_request_headers()
test_request_method()
test_build_opener()
test_install_opener()
test_urlopen_data()
test_urlopen_data_base64()
test_robotparser_basic()
test_robotparser_crawl_delay()
test_robotparser_site_maps()

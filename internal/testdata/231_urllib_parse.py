import urllib.parse


def test_urlparse_basic():
    r = urllib.parse.urlparse("https://user:pw@example.com:8080/path/to?k=v&x=1#sec")
    assert r.scheme == "https", repr(r.scheme)
    assert r.netloc == "user:pw@example.com:8080", repr(r.netloc)
    assert r.path == "/path/to", repr(r.path)
    assert r.query == "k=v&x=1", repr(r.query)
    assert r.fragment == "sec", repr(r.fragment)
    assert r.params == "", repr(r.params)
    print("urlparse_basic ok")


def test_urlparse_scheme_default():
    r = urllib.parse.urlparse("//example.com/path", scheme="https")
    assert r.scheme == "https", repr(r.scheme)
    assert r.netloc == "example.com", repr(r.netloc)
    assert r.path == "/path", repr(r.path)
    r2 = urllib.parse.urlparse("http://example.com/", scheme="ftp")
    assert r2.scheme == "http", repr(r2.scheme)  # existing scheme wins
    print("urlparse_scheme_default ok")


def test_urlparse_allow_fragments():
    r = urllib.parse.urlparse("http://example.com/path?q=1#frag", allow_fragments=False)
    assert r.fragment == "", repr(r.fragment)
    assert "#frag" in r.path or "#frag" in r.query, repr((r.path, r.query))
    r2 = urllib.parse.urlparse("http://example.com/path#frag", allow_fragments=True)
    assert r2.fragment == "frag", repr(r2.fragment)
    print("urlparse_allow_fragments ok")


def test_urlsplit_no_params():
    r = urllib.parse.urlsplit("http://example.com/path;params?query#frag")
    assert r.scheme == "http", repr(r.scheme)
    assert r.netloc == "example.com", repr(r.netloc)
    assert r.query == "query", repr(r.query)
    assert r.fragment == "frag", repr(r.fragment)
    # SplitResult has no params attribute in the tuple sense — 5 elements
    assert r[0] == "http", repr(r[0])
    assert r[1] == "example.com", repr(r[1])
    assert len(list(r)) == 5, len(list(r))
    print("urlsplit_no_params ok")


def test_parse_result_username_password():
    r = urllib.parse.urlparse("https://alice:s3cr3t@example.com/")
    assert r.username == "alice", repr(r.username)
    assert r.password == "s3cr3t", repr(r.password)
    r2 = urllib.parse.urlparse("https://bob@example.com/")
    assert r2.username == "bob", repr(r2.username)
    assert r2.password is None, repr(r2.password)
    r3 = urllib.parse.urlparse("https://example.com/")
    assert r3.username is None, repr(r3.username)
    assert r3.password is None, repr(r3.password)
    print("parse_result_username_password ok")


def test_parse_result_hostname_port():
    r = urllib.parse.urlparse("http://user:pw@Host.Example.COM:9000/path")
    assert r.hostname == "host.example.com", repr(r.hostname)
    assert r.port == 9000, repr(r.port)
    r2 = urllib.parse.urlparse("http://example.com/")
    assert r2.port is None, repr(r2.port)
    print("parse_result_hostname_port ok")


def test_parse_result_geturl():
    original = "http://example.com/path?q=hello#frag"
    r = urllib.parse.urlparse(original)
    assert r.geturl() == original, repr(r.geturl())
    r2 = urllib.parse.urlsplit("https://example.com/split?x=1")
    assert r2.geturl() == "https://example.com/split?x=1", repr(r2.geturl())
    print("parse_result_geturl ok")


def test_split_result_repr():
    r = urllib.parse.urlsplit("http://example.com/path?q=1#f")
    s = repr(r)
    assert s.startswith("SplitResult("), repr(s)
    assert "scheme=" in s
    assert "netloc=" in s
    print("split_result_repr ok")


def test_parse_result_repr():
    r = urllib.parse.urlparse("http://example.com/path?q=1#f")
    s = repr(r)
    assert s.startswith("ParseResult("), repr(s)
    assert "params=" in s
    print("parse_result_repr ok")


def test_urlunparse_roundtrip():
    original = "https://example.com/path?q=1#f"
    r = urllib.parse.urlparse(original)
    rebuilt = urllib.parse.urlunparse(r)
    assert rebuilt == original, repr(rebuilt)
    # From tuple
    t = ("http", "example.com", "/p", "par", "q=1", "frag")
    assert urllib.parse.urlunparse(t) == "http://example.com/p;par?q=1#frag"
    print("urlunparse_roundtrip ok")


def test_urlunsplit_roundtrip():
    original = "https://example.com/path?q=1#f"
    r = urllib.parse.urlsplit(original)
    rebuilt = urllib.parse.urlunsplit(r)
    assert rebuilt == original, repr(rebuilt)
    t = ("http", "example.com", "/p", "q=1", "frag")
    assert urllib.parse.urlunsplit(t) == "http://example.com/p?q=1#frag"
    print("urlunsplit_roundtrip ok")


def test_urljoin_allow_fragments():
    base = "http://example.com/path?q=1#frag"
    r1 = urllib.parse.urljoin(base, "other#anchor", allow_fragments=True)
    assert "#anchor" in r1, repr(r1)
    r2 = urllib.parse.urljoin(base, "other#anchor", allow_fragments=False)
    assert "#anchor" not in r2, repr(r2)
    print("urljoin_allow_fragments ok")


def test_urlencode_quote_via():
    # default quote_plus: spaces become +
    s1 = urllib.parse.urlencode({"msg": "hello world"})
    assert s1 == "msg=hello+world", repr(s1)
    # quote_via=quote: spaces become %20 instead of +
    s2 = urllib.parse.urlencode({"msg": "hello world"}, quote_via=urllib.parse.quote)
    assert s2 == "msg=hello%20world", repr(s2)
    print("urlencode_quote_via ok")


def test_urlencode_safe():
    # safe="/" preserves slashes when using quote_via=quote
    s = urllib.parse.urlencode({"path": "/api/data"}, safe="/", quote_via=urllib.parse.quote)
    assert s == "path=/api/data", repr(s)
    print("urlencode_safe ok")


def test_parse_qs_keep_blank():
    # by default, blank values are dropped
    r1 = urllib.parse.parse_qs("a=1&b=&c=3")
    assert "b" not in r1, repr(r1)
    # with keep_blank_values=True, blank values are kept
    r2 = urllib.parse.parse_qs("a=1&b=&c=3", keep_blank_values=True)
    assert "b" in r2, repr(r2)
    assert r2["b"] == [""], repr(r2["b"])
    print("parse_qs_keep_blank ok")


def test_parse_qs_separator():
    # semicolons as separator (legacy behavior)
    r1 = urllib.parse.parse_qs("a=1;b=2", separator=";")
    assert "a" in r1 and "b" in r1, repr(r1)
    # default '&' only — semicolons are NOT split
    r2 = urllib.parse.parse_qs("a=1;b=2")
    assert len(r2) == 1, repr(r2)  # treated as one param "a" with value "1;b=2"
    print("parse_qs_separator ok")


def test_parse_qsl_keep_blank():
    pairs = urllib.parse.parse_qsl("a=1&b=&c=3", keep_blank_values=True)
    keys = [k for k, v in pairs]
    assert "b" in keys, repr(pairs)
    pairs2 = urllib.parse.parse_qsl("a=1&b=&c=3")
    keys2 = [k for k, v in pairs2]
    assert "b" not in keys2, repr(pairs2)
    print("parse_qsl_keep_blank ok")


def test_parse_qs_max_num_fields():
    # max_num_fields limits the number of key-value pairs parsed
    try:
        urllib.parse.parse_qs("a=1&b=2&c=3", max_num_fields=2)
        assert False, "expected ValueError"
    except ValueError:
        pass
    r = urllib.parse.parse_qs("a=1&b=2", max_num_fields=5)
    assert "a" in r and "b" in r
    print("parse_qs_max_num_fields ok")


def test_defrag_result_tuple():
    r = urllib.parse.urldefrag("http://example.com/path#section")
    assert r[0] == "http://example.com/path", repr(r[0])
    assert r[1] == "section", repr(r[1])
    assert len(r) == 2, repr(len(r))
    # negative indexing
    assert r[-1] == "section", repr(r[-1])
    print("defrag_result_tuple ok")


def test_defrag_result_iter():
    r = urllib.parse.urldefrag("http://example.com/path#section")
    items = list(r)
    assert items == ["http://example.com/path", "section"], repr(items)
    url, frag = r
    assert url == "http://example.com/path"
    assert frag == "section"
    print("defrag_result_iter ok")


def test_defrag_encode():
    r = urllib.parse.urldefrag("http://example.com/path#section")
    rb = r.encode()
    assert rb.url == b"http://example.com/path", repr(rb.url)
    assert rb.fragment == b"section", repr(rb.fragment)
    # decode back to str result
    rs = rb.decode()
    assert rs.url == "http://example.com/path", repr(rs.url)
    assert rs.fragment == "section", repr(rs.fragment)
    print("defrag_encode ok")


def test_urlparse_bytes():
    rb = urllib.parse.urlparse(b"http://example.com/path?q=1")
    assert rb.scheme == b"http", repr(rb.scheme)
    assert rb.netloc == b"example.com", repr(rb.netloc)
    assert rb.path == b"/path", repr(rb.path)
    assert rb.query == b"q=1", repr(rb.query)
    # decode back
    rs = rb.decode()
    assert rs.scheme == "http", repr(rs.scheme)
    print("urlparse_bytes ok")


def test_urldefrag_bytes():
    rb = urllib.parse.urldefrag(b"http://example.com/path#frag")
    assert rb.url == b"http://example.com/path", repr(rb.url)
    assert rb.fragment == b"frag", repr(rb.fragment)
    print("urldefrag_bytes ok")


def test_classes_exported():
    assert urllib.parse.ParseResult is not None
    assert urllib.parse.SplitResult is not None
    assert urllib.parse.DefragResult is not None
    assert urllib.parse.ParseResultBytes is not None
    assert urllib.parse.SplitResultBytes is not None
    assert urllib.parse.DefragResultBytes is not None
    # urlparse result should be an instance of ParseResult? (not strictly but check accessible)
    r = urllib.parse.urlparse("http://example.com/")
    assert r.scheme == "http"
    print("classes_exported ok")


test_urlparse_basic()
test_urlparse_scheme_default()
test_urlparse_allow_fragments()
test_urlsplit_no_params()
test_parse_result_username_password()
test_parse_result_hostname_port()
test_parse_result_geturl()
test_split_result_repr()
test_parse_result_repr()
test_urlunparse_roundtrip()
test_urlunsplit_roundtrip()
test_urljoin_allow_fragments()
test_urlencode_quote_via()
test_urlencode_safe()
test_parse_qs_keep_blank()
test_parse_qs_separator()
test_parse_qsl_keep_blank()
test_parse_qs_max_num_fields()
test_defrag_result_tuple()
test_defrag_result_iter()
test_defrag_encode()
test_urlparse_bytes()
test_urldefrag_bytes()
test_classes_exported()

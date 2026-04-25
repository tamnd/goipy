import mimetypes

def test_guess_type_basic():
    assert mimetypes.guess_type("file.html") == ("text/html", None)
    assert mimetypes.guess_type("file.txt") == ("text/plain", None)
    assert mimetypes.guess_type("file.json") == ("application/json", None)
    assert mimetypes.guess_type("file.png") == ("image/png", None)
    assert mimetypes.guess_type("file.mp3") == ("audio/mpeg", None)
    print("guess_type_basic ok")

def test_guess_type_encoding():
    t, enc = mimetypes.guess_type("archive.tar.gz")
    assert t == "application/x-tar", f"got {t!r}"
    assert enc == "gzip", f"got enc {enc!r}"
    t2, enc2 = mimetypes.guess_type("file.txt.gz")
    assert t2 == "text/plain", f"got {t2!r}"
    assert enc2 == "gzip", f"got enc {enc2!r}"
    print("guess_type_encoding ok")

def test_guess_type_unknown():
    assert mimetypes.guess_type("file.unknownxyz") == (None, None)
    assert mimetypes.guess_type("noext") == (None, None)
    print("guess_type_unknown ok")

def test_guess_type_url():
    t, enc = mimetypes.guess_type("http://example.com/page.html?foo=bar")
    assert t == "text/html", f"got {t!r}"
    assert enc is None
    print("guess_type_url ok")

def test_guess_all_extensions():
    exts = mimetypes.guess_all_extensions("text/html")
    assert ".html" in exts or ".htm" in exts, f"got {exts}"
    exts2 = mimetypes.guess_all_extensions("image/png")
    assert ".png" in exts2
    print("guess_all_extensions ok")

def test_guess_extension():
    ext = mimetypes.guess_extension("image/png")
    assert ext == ".png", f"got {ext!r}"
    none_ext = mimetypes.guess_extension("application/x-totally-unknown")
    assert none_ext is None
    print("guess_extension ok")

def test_add_type():
    mimetypes.add_type("application/x-goipy", ".goipy")
    t, enc = mimetypes.guess_type("test.goipy")
    assert t == "application/x-goipy", f"got {t!r}"
    ext = mimetypes.guess_extension("application/x-goipy")
    assert ext == ".goipy", f"got {ext!r}"
    print("add_type ok")

def test_init():
    mimetypes.init()
    assert mimetypes.inited == True
    print("init ok")

def test_suffix_map():
    assert ".tgz" in mimetypes.suffix_map
    assert ".gz" in mimetypes.encodings_map
    assert mimetypes.encodings_map[".gz"] == "gzip"
    print("suffix_map ok")

def test_read_mime_types():
    import tempfile, os
    with tempfile.NamedTemporaryFile(mode='w', suffix='.types', delete=False) as f:
        f.write("application/x-test .xtst .xtst2\n")
        fname = f.name
    try:
        result = mimetypes.read_mime_types(fname)
        assert result is not None
        assert ".xtst" in result, f"got {result}"
        assert result[".xtst"] == "application/x-test"
    finally:
        os.unlink(fname)
    none_result = mimetypes.read_mime_types("/nonexistent/file.types")
    assert none_result is None
    print("read_mime_types ok")

def test_mimetypes_class():
    mt = mimetypes.MimeTypes()
    mt.add("text/x-custom", ".xcustom")
    t, enc = mt.guess_type("file.xcustom")
    assert t == "text/x-custom", f"got {t!r}"
    exts = mt.guess_all_extensions("text/x-custom")
    assert ".xcustom" in exts
    ext = mt.guess_extension("text/x-custom")
    assert ext == ".xcustom"
    print("mimetypes_class ok")

test_guess_type_basic()
test_guess_type_encoding()
test_guess_type_unknown()
test_guess_type_url()
test_guess_all_extensions()
test_guess_extension()
test_add_type()
test_init()
test_suffix_map()
test_read_mime_types()
test_mimetypes_class()
print("ALL OK")

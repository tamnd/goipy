import xml.sax.xmlreader as xr
import xml.sax.handler as h


def test_xmlreader_init_handlers():
    r = xr.XMLReader()
    ch = r.getContentHandler()
    assert ch is not None
    # verify it's a ContentHandler by calling a no-op method
    assert ch.startElement("x", {}) is None
    assert r.getDTDHandler() is not None
    assert r.getEntityResolver() is not None
    assert r.getErrorHandler() is not None
    print("xmlreader_init_handlers ok")


def test_xmlreader_handler_roundtrip():
    r = xr.XMLReader()
    ch = h.ContentHandler()
    r.setContentHandler(ch)
    assert r.getContentHandler() is ch

    dh = h.DTDHandler()
    r.setDTDHandler(dh)
    assert r.getDTDHandler() is dh

    er = h.EntityResolver()
    r.setEntityResolver(er)
    assert r.getEntityResolver() is er

    eh = h.ErrorHandler()
    r.setErrorHandler(eh)
    assert r.getErrorHandler() is eh
    print("xmlreader_handler_roundtrip ok")


def test_xmlreader_set_locale():
    r = xr.XMLReader()
    try:
        r.setLocale("en")
        assert False, "should raise"
    except Exception as e:
        assert "Locale" in str(e) or "not implemented" in str(e)
    print("xmlreader_set_locale ok")


def test_xmlreader_get_feature():
    r = xr.XMLReader()
    try:
        r.getFeature("http://unknown/feature")
        assert False, "should raise"
    except Exception as e:
        assert "not recognized" in str(e)
    print("xmlreader_get_feature ok")


def test_xmlreader_set_feature():
    r = xr.XMLReader()
    try:
        r.setFeature("http://unknown/feature", True)
        assert False, "should raise"
    except Exception as e:
        assert "not recognized" in str(e)
    print("xmlreader_set_feature ok")


def test_xmlreader_get_property():
    r = xr.XMLReader()
    try:
        r.getProperty("http://unknown/prop")
        assert False, "should raise"
    except Exception as e:
        assert "not recognized" in str(e)
    print("xmlreader_get_property ok")


def test_xmlreader_set_property():
    r = xr.XMLReader()
    try:
        r.setProperty("http://unknown/prop", "val")
        assert False, "should raise"
    except Exception as e:
        assert "not recognized" in str(e)
    print("xmlreader_set_property ok")


def test_xmlreader_parse_raises():
    r = xr.XMLReader()
    try:
        r.parse(xr.InputSource())
        assert False, "should raise"
    except NotImplementedError as e:
        assert "must be implemented" in str(e)
    print("xmlreader_parse_raises ok")


def test_incremental_parser_init():
    class MyParser(xr.IncrementalParser):
        def __init__(self):
            super().__init__(4096)
            self.chunks = []
        def feed(self, data): self.chunks.append(data)
        def close(self): pass
        def reset(self): self.chunks = []
        def prepareParser(self, source): pass

    p = MyParser()
    ch = h.ContentHandler()
    p.setContentHandler(ch)
    assert p.getContentHandler() is ch
    print("incremental_parser_init ok")


def test_incremental_parser_abstract():
    p = xr.IncrementalParser()
    try:
        p.feed(b"data")
        assert False
    except NotImplementedError as e:
        assert "must be implemented" in str(e)
    try:
        p.close()
        assert False
    except NotImplementedError:
        pass
    try:
        p.reset()
        assert False
    except NotImplementedError:
        pass
    try:
        p.prepareParser(xr.InputSource())
        assert False
    except NotImplementedError as e:
        assert "overridden" in str(e)
    print("incremental_parser_abstract ok")


def test_incremental_parser_parse():
    import io

    class ChunkParser(xr.IncrementalParser):
        def __init__(self, bufsize):
            super().__init__(bufsize)
            self.data = b""
            self.closed = False
        def feed(self, chunk): self.data += chunk
        def close(self): self.closed = True
        def reset(self): self.data = b""
        def prepareParser(self, source): pass

    src = xr.InputSource()
    src.setByteStream(io.BytesIO(b"hello world"))

    p = ChunkParser(4)
    p.parse(src)
    assert p.data == b"hello world"
    assert p.closed
    print("incremental_parser_parse ok")


def test_attributes_impl_values():
    a = xr.AttributesImpl({"href": "http://x.com", "class": "main"})
    vals = sorted(a.values())
    assert vals == ["http://x.com", "main"]
    print("attributes_impl_values ok")


def test_attributes_impl_get():
    a = xr.AttributesImpl({"k": "v"})
    assert a.get("k") == "v"
    assert a.get("missing") is None
    assert a.get("missing", "default") == "default"
    print("attributes_impl_get ok")


def test_attributes_impl_keyerror():
    a = xr.AttributesImpl({"k": "v"})
    try:
        _ = a["missing"]
        assert False
    except KeyError:
        pass
    try:
        a.getValue("missing")
        assert False
    except KeyError:
        pass
    try:
        a.getQNameByName("missing")
        assert False
    except KeyError:
        pass
    try:
        a.getNameByQName("missing")
        assert False
    except KeyError:
        pass
    print("attributes_impl_keyerror ok")


def test_attributes_ns_impl_basic():
    ns = "http://example.com"
    attrs = {(ns, "href"): "http://x.com", (ns, "class"): "main"}
    qnames = {(ns, "href"): "ex:href", (ns, "class"): "ex:class"}
    a = xr.AttributesNSImpl(attrs, qnames)
    assert len(a) == 2
    assert a.getLength() == 2
    assert a[(ns, "href")] == "http://x.com"
    assert (ns, "href") in a
    assert ("other", "href") not in a
    assert a.getValue((ns, "href")) == "http://x.com"
    print("attributes_ns_impl_basic ok")


def test_attributes_ns_impl_names():
    ns = "http://example.com"
    attrs = {(ns, "href"): "http://x.com"}
    qnames = {(ns, "href"): "ex:href"}
    a = xr.AttributesNSImpl(attrs, qnames)
    assert sorted(a.getNames()) == [(ns, "href")]
    assert sorted(a.keys()) == [(ns, "href")]
    assert a.getQNames() == ["ex:href"]
    print("attributes_ns_impl_names ok")


def test_attributes_ns_impl_qname_lookup():
    ns = "http://example.com"
    attrs = {(ns, "href"): "http://x.com", (ns, "class"): "main"}
    qnames = {(ns, "href"): "ex:href", (ns, "class"): "ex:class"}
    a = xr.AttributesNSImpl(attrs, qnames)
    assert a.getQNameByName((ns, "href")) == "ex:href"
    assert a.getNameByQName("ex:href") == (ns, "href")
    assert a.getValueByQName("ex:href") == "http://x.com"
    assert a.getType((ns, "href")) == "CDATA"
    print("attributes_ns_impl_qname_lookup ok")


def test_attributes_ns_impl_get_values_items():
    ns = "http://example.com"
    attrs = {(ns, "href"): "http://x.com"}
    qnames = {(ns, "href"): "ex:href"}
    a = xr.AttributesNSImpl(attrs, qnames)
    assert a.get((ns, "href")) == "http://x.com"
    assert a.get((ns, "missing"), "def") == "def"
    assert a.values() == ["http://x.com"]
    items = a.items()
    assert len(items) == 1
    assert items[0] == ((ns, "href"), "http://x.com")
    print("attributes_ns_impl_get_values_items ok")


def test_attributes_ns_impl_copy():
    ns = "http://example.com"
    attrs = {(ns, "x"): "1"}
    qnames = {(ns, "x"): "ns:x"}
    a = xr.AttributesNSImpl(attrs, qnames)
    c = a.copy()
    assert type(c).__name__ == "AttributesNSImpl"
    assert c[(ns, "x")] == "1"
    assert c.getQNameByName((ns, "x")) == "ns:x"
    print("attributes_ns_impl_copy ok")


def test_locator():
    loc = xr.Locator()
    assert loc.getColumnNumber() == -1
    assert loc.getLineNumber() == -1
    assert loc.getPublicId() is None
    assert loc.getSystemId() is None
    print("locator ok")


def test_input_source():
    src = xr.InputSource("http://example.com/file.xml")
    assert src.getSystemId() == "http://example.com/file.xml"
    assert src.getPublicId() is None
    assert src.getByteStream() is None
    assert src.getEncoding() is None
    src.setPublicId("-//FOO//EN")
    assert src.getPublicId() == "-//FOO//EN"
    src.setEncoding("utf-8")
    assert src.getEncoding() == "utf-8"
    import io
    bs = io.BytesIO(b"<x/>")
    src.setByteStream(bs)
    assert src.getByteStream() is bs
    print("input_source ok")


test_xmlreader_init_handlers()
test_xmlreader_handler_roundtrip()
test_xmlreader_set_locale()
test_xmlreader_get_feature()
test_xmlreader_set_feature()
test_xmlreader_get_property()
test_xmlreader_set_property()
test_xmlreader_parse_raises()
test_incremental_parser_init()
test_incremental_parser_abstract()
test_incremental_parser_parse()
test_attributes_impl_values()
test_attributes_impl_get()
test_attributes_impl_keyerror()
test_attributes_ns_impl_basic()
test_attributes_ns_impl_names()
test_attributes_ns_impl_qname_lookup()
test_attributes_ns_impl_get_values_items()
test_attributes_ns_impl_copy()
test_locator()
test_input_source()

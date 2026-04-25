import xml.sax
import xml.sax.handler
import xml.sax.saxutils
import xml.sax.xmlreader
from io import StringIO, BytesIO


def test_sax_exception():
    e = xml.sax.SAXException("bad xml")
    assert e.getMessage() == "bad xml"
    assert e.getException() is None
    assert str(e) == "bad xml"
    e2 = xml.sax.SAXException("wrapped", ValueError("inner"))
    assert e2.getMessage() == "wrapped"
    print("sax_exception ok")


def test_sax_parse_exception():
    loc = xml.sax.xmlreader.Locator()
    e = xml.sax.SAXParseException("parse error", None, loc)
    assert e.getMessage() == "parse error"
    assert e.getLineNumber() == -1
    assert e.getColumnNumber() == -1
    assert e.getSystemId() is None
    assert e.getPublicId() is None
    print("sax_parse_exception ok")


def test_sax_exception_classes():
    e1 = xml.sax.SAXNotRecognizedException("nr")
    assert isinstance(e1, xml.sax.SAXException)
    e2 = xml.sax.SAXNotSupportedException("ns")
    assert isinstance(e2, xml.sax.SAXException)
    e3 = xml.sax.SAXReaderNotAvailable("na")
    assert isinstance(e3, xml.sax.SAXNotSupportedException)
    assert isinstance(e3, xml.sax.SAXException)
    print("sax_exception_classes ok")


def test_parse_string_elements():
    class H(xml.sax.handler.ContentHandler):
        def __init__(self):
            self.events = []
        def startElement(self, name, attrs):
            self.events.append(("start", name))
        def endElement(self, name):
            self.events.append(("end", name))

    h = H()
    xml.sax.parseString(b"<root><child/></root>", h)
    assert ("start", "root") in h.events
    assert ("start", "child") in h.events
    assert ("end", "child") in h.events
    assert ("end", "root") in h.events
    print("parse_string_elements ok")


def test_parse_string_characters():
    class H(xml.sax.handler.ContentHandler):
        def __init__(self):
            self.texts = []
        def characters(self, content):
            self.texts.append(content)

    h = H()
    xml.sax.parseString(b"<root>hello world</root>", h)
    assert "hello world" in h.texts
    print("parse_string_characters ok")


def test_parse_string_pi():
    class H(xml.sax.handler.ContentHandler):
        def __init__(self):
            self.pis = []
        def processingInstruction(self, target, data):
            self.pis.append((target, data))

    h = H()
    xml.sax.parseString(b"<?xml version='1.0'?><root><?py run?></root>", h)
    assert ("py", "run") in h.pis
    print("parse_string_pi ok")


def test_make_parser_parse():
    class H(xml.sax.handler.ContentHandler):
        def __init__(self):
            self.events = []
        def startDocument(self):
            self.events.append("startDocument")
        def endDocument(self):
            self.events.append("endDocument")
        def startElement(self, name, attrs):
            self.events.append(("start", name))

    h = H()
    p = xml.sax.make_parser()
    p.setContentHandler(h)
    p.parse(BytesIO(b"<doc><item/></doc>"))
    assert "startDocument" in h.events
    assert ("start", "doc") in h.events
    assert "endDocument" in h.events
    print("make_parser_parse ok")


def test_make_parser_get_handler():
    class H(xml.sax.handler.ContentHandler):
        pass

    p = xml.sax.make_parser()
    h = H()
    p.setContentHandler(h)
    assert p.getContentHandler() is h
    print("make_parser_get_handler ok")


def test_content_handler_all_methods():
    class H(xml.sax.handler.ContentHandler):
        def __init__(self):
            self.log = []
        def startDocument(self):
            self.log.append("startDocument")
        def endDocument(self):
            self.log.append("endDocument")
        def startPrefixMapping(self, prefix, uri):
            self.log.append(("startPrefixMapping", prefix, uri))
        def endPrefixMapping(self, prefix):
            self.log.append(("endPrefixMapping", prefix))
        def startElement(self, name, attrs):
            self.log.append(("startElement", name))
        def endElement(self, name):
            self.log.append(("endElement", name))
        def characters(self, content):
            self.log.append(("characters", content))
        def ignorableWhitespace(self, whitespace):
            self.log.append("ignorableWhitespace")
        def processingInstruction(self, target, data):
            self.log.append(("pi", target, data))
        def skippedEntity(self, name):
            self.log.append(("skippedEntity", name))
        def setDocumentLocator(self, locator):
            self.log.append("setDocumentLocator")

    h = H()
    xml.sax.parseString(b"<root>text</root>", h)
    assert "startDocument" in h.log
    assert ("startElement", "root") in h.log
    assert ("characters", "text") in h.log
    assert ("endElement", "root") in h.log
    assert "endDocument" in h.log
    print("content_handler_all_methods ok")


def test_dtd_handler():
    h = xml.sax.handler.DTDHandler()
    h.notationDecl("name", "pub", "sys")
    h.unparsedEntityDecl("name", "pub", "sys", "ndata")
    print("dtd_handler ok")


def test_entity_resolver():
    r = xml.sax.handler.EntityResolver()
    src = r.resolveEntity("pub", "sys")
    assert src == "sys"
    print("entity_resolver ok")


def test_error_handler_fatal():
    eh = xml.sax.handler.ErrorHandler()
    try:
        eh.fatalError(xml.sax.SAXException("boom"))
        assert False, "should raise"
    except Exception as e:
        assert "boom" in str(e)
    print("error_handler_fatal ok")


def test_lexical_handler():
    lh = xml.sax.handler.LexicalHandler()
    lh.comment("a comment")
    lh.startDTD("root", None, None)
    lh.endDTD()
    lh.startCDATA()
    lh.endCDATA()
    print("lexical_handler ok")


def test_handler_classes_exist():
    assert hasattr(xml.sax.handler, "ContentHandler")
    assert hasattr(xml.sax.handler, "DTDHandler")
    assert hasattr(xml.sax.handler, "EntityResolver")
    assert hasattr(xml.sax.handler, "ErrorHandler")
    assert hasattr(xml.sax.handler, "LexicalHandler")
    print("handler_classes_exist ok")


def test_feature_constants():
    h = xml.sax.handler
    assert h.feature_namespaces == "http://xml.org/sax/features/namespaces"
    assert h.feature_namespace_prefixes == "http://xml.org/sax/features/namespace-prefixes"
    assert h.feature_string_interning == "http://xml.org/sax/features/string-interning"
    assert h.feature_validation == "http://xml.org/sax/features/validation"
    assert h.feature_external_ges == "http://xml.org/sax/features/external-general-entities"
    assert h.feature_external_pes == "http://xml.org/sax/features/external-parameter-entities"
    assert len(h.all_features) == 6
    print("feature_constants ok")


def test_property_constants():
    h = xml.sax.handler
    assert h.property_lexical_handler == "http://xml.org/sax/properties/lexical-handler"
    assert h.property_declaration_handler == "http://xml.org/sax/properties/declaration-handler"
    assert h.property_dom_node == "http://xml.org/sax/properties/dom-node"
    assert h.property_xml_string == "http://xml.org/sax/properties/xml-string"
    assert h.property_encoding == "http://www.python.org/sax/properties/encoding"
    assert h.property_interning_dict == "http://www.python.org/sax/properties/interning-dict"
    assert len(h.all_properties) == 6
    print("property_constants ok")


def test_saxutils_escape_unescape_quoteattr():
    assert xml.sax.saxutils.escape("a&b<c>d") == "a&amp;b&lt;c&gt;d"
    assert xml.sax.saxutils.unescape("a&amp;b&lt;c&gt;d") == "a&b<c>d"
    q = xml.sax.saxutils.quoteattr('say "hi"')
    assert q == '\'say "hi"\''
    q2 = xml.sax.saxutils.quoteattr("hello")
    assert q2 == '"hello"'
    print("saxutils_escape_unescape_quoteattr ok")


def test_xml_generator():
    out = StringIO()
    gen = xml.sax.saxutils.XMLGenerator(out, "utf-8")
    gen.startDocument()
    gen.startElement("root", {})
    gen.characters("hello")
    gen.endElement("root")
    result = out.getvalue()
    assert '<?xml version="1.0" encoding="utf-8"?>' in result
    assert "<root>" in result
    assert "hello" in result
    assert "</root>" in result
    print("xml_generator ok")


def test_xml_filter_base():
    f = xml.sax.saxutils.XMLFilterBase()
    assert f is not None
    f2 = xml.sax.saxutils.XMLFilterBase(None)
    assert f2 is not None
    print("xml_filter_base ok")


def test_input_source():
    src = xml.sax.xmlreader.InputSource("http://example.com")
    assert src.getSystemId() == "http://example.com"
    assert src.getPublicId() is None
    assert src.getByteStream() is None
    assert src.getCharacterStream() is None
    assert src.getEncoding() is None
    src.setPublicId("pub")
    assert src.getPublicId() == "pub"
    src.setEncoding("utf-8")
    assert src.getEncoding() == "utf-8"
    print("input_source ok")


def test_attributes_impl():
    attrs = xml.sax.xmlreader.AttributesImpl({"href": "http://x.com", "id": "1"})
    ks = attrs.keys()
    assert "href" in ks
    assert "id" in ks
    it = attrs.items()
    assert len(it) == 2
    c = attrs.copy()
    assert c.getValue("href") == "http://x.com"
    qnames = attrs.getQNames()
    assert "href" in qnames
    assert attrs.getLength() == 2
    assert len(attrs) == 2
    assert attrs["id"] == "1"
    assert "id" in attrs
    print("attributes_impl ok")


def test_attributes_ns_impl():
    ns_attrs = {"href": "url", "id": "1"}
    a = xml.sax.xmlreader.AttributesNSImpl(ns_attrs, {})
    assert a is not None
    print("attributes_ns_impl ok")


def test_locator():
    loc = xml.sax.xmlreader.Locator()
    assert loc.getColumnNumber() == -1
    assert loc.getLineNumber() == -1
    assert loc.getPublicId() is None
    assert loc.getSystemId() is None
    print("locator ok")


test_sax_exception()
test_sax_parse_exception()
test_sax_exception_classes()
test_parse_string_elements()
test_parse_string_characters()
test_parse_string_pi()
test_make_parser_parse()
test_make_parser_get_handler()
test_content_handler_all_methods()
test_dtd_handler()
test_entity_resolver()
test_error_handler_fatal()
test_lexical_handler()
test_handler_classes_exist()
test_feature_constants()
test_property_constants()
test_saxutils_escape_unescape_quoteattr()
test_xml_generator()
test_xml_filter_base()
test_input_source()
test_attributes_impl()
test_attributes_ns_impl()
test_locator()

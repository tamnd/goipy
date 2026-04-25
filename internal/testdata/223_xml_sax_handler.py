import xml.sax.handler
import xml.sax


def test_version():
    assert xml.sax.handler.version == "2.0beta"
    print("version ok")


def test_feature_constants():
    h = xml.sax.handler
    assert h.feature_namespaces == "http://xml.org/sax/features/namespaces"
    assert h.feature_namespace_prefixes == "http://xml.org/sax/features/namespace-prefixes"
    assert h.feature_string_interning == "http://xml.org/sax/features/string-interning"
    assert h.feature_validation == "http://xml.org/sax/features/validation"
    assert h.feature_external_ges == "http://xml.org/sax/features/external-general-entities"
    assert h.feature_external_pes == "http://xml.org/sax/features/external-parameter-entities"
    print("feature_constants ok")


def test_property_constants():
    h = xml.sax.handler
    assert h.property_lexical_handler == "http://xml.org/sax/properties/lexical-handler"
    assert h.property_dom_node == "http://xml.org/sax/properties/dom-node"
    assert h.property_declaration_handler == "http://xml.org/sax/properties/declaration-handler"
    assert h.property_xml_string == "http://xml.org/sax/properties/xml-string"
    assert h.property_encoding == "http://www.python.org/sax/properties/encoding"
    assert h.property_interning_dict == "http://www.python.org/sax/properties/interning-dict"
    print("property_constants ok")


def test_all_features_order():
    h = xml.sax.handler
    assert len(h.all_features) == 6
    assert h.all_features[0] == "http://xml.org/sax/features/namespaces"
    assert h.all_features[1] == "http://xml.org/sax/features/namespace-prefixes"
    assert h.all_features[2] == "http://xml.org/sax/features/string-interning"
    assert h.all_features[3] == "http://xml.org/sax/features/validation"
    assert h.all_features[4] == "http://xml.org/sax/features/external-general-entities"
    assert h.all_features[5] == "http://xml.org/sax/features/external-parameter-entities"
    print("all_features_order ok")


def test_all_properties_order():
    h = xml.sax.handler
    assert len(h.all_properties) == 6
    assert h.all_properties[0] == "http://xml.org/sax/properties/lexical-handler"
    assert h.all_properties[1] == "http://xml.org/sax/properties/dom-node"
    assert h.all_properties[2] == "http://xml.org/sax/properties/declaration-handler"
    assert h.all_properties[3] == "http://xml.org/sax/properties/xml-string"
    assert h.all_properties[4] == "http://www.python.org/sax/properties/encoding"
    assert h.all_properties[5] == "http://www.python.org/sax/properties/interning-dict"
    print("all_properties_order ok")


def test_content_handler_base_noop():
    ch = xml.sax.handler.ContentHandler()
    assert ch.startDocument() is None
    assert ch.endDocument() is None
    assert ch.startPrefixMapping("ns", "http://example.com") is None
    assert ch.endPrefixMapping("ns") is None
    assert ch.startElement("tag", {}) is None
    assert ch.endElement("tag") is None
    assert ch.startElementNS(("http://ns", "tag"), "ns:tag", {}) is None
    assert ch.endElementNS(("http://ns", "tag"), "ns:tag") is None
    assert ch.characters("text") is None
    assert ch.ignorableWhitespace("   ") is None
    assert ch.processingInstruction("target", "data") is None
    assert ch.skippedEntity("amp") is None
    assert ch.setDocumentLocator(None) is None
    print("content_handler_base_noop ok")


def test_content_handler_subclass_args():
    class H(xml.sax.handler.ContentHandler):
        def __init__(self):
            self.log = []
        def startElement(self, name, attrs):
            self.log.append(("start", name))
        def endElement(self, name):
            self.log.append(("end", name))
        def characters(self, content):
            self.log.append(("chars", content))
        def startPrefixMapping(self, prefix, uri):
            self.log.append(("prefix", prefix, uri))
        def processingInstruction(self, target, data):
            self.log.append(("pi", target, data))

    h = H()
    xml.sax.parseString(b"<root>hi</root>", h)
    assert ("start", "root") in h.log
    assert ("chars", "hi") in h.log
    assert ("end", "root") in h.log
    print("content_handler_subclass_args ok")


def test_dtd_handler():
    dh = xml.sax.handler.DTDHandler()
    assert dh.notationDecl("name", "pub", "sys") is None
    assert dh.unparsedEntityDecl("name", "pub", "sys", "ndata") is None
    print("dtd_handler ok")


def test_entity_resolver():
    er = xml.sax.handler.EntityResolver()
    result = er.resolveEntity(None, "sys")
    assert result == "sys"
    result2 = er.resolveEntity("pub", "other")
    assert result2 == "other"
    print("entity_resolver ok")


def test_error_handler_warning_noop():
    eh = xml.sax.handler.ErrorHandler()
    result = eh.warning(xml.sax.SAXException("warn"))
    assert result is None
    print("error_handler_warning_noop ok")


def test_error_handler_error_raises():
    eh = xml.sax.handler.ErrorHandler()
    exc = xml.sax.SAXException("recoverable error")
    try:
        eh.error(exc)
        assert False, "should have raised"
    except Exception as e:
        assert "recoverable error" in str(e)
    print("error_handler_error_raises ok")


def test_error_handler_fatal_raises():
    eh = xml.sax.handler.ErrorHandler()
    exc = xml.sax.SAXException("fatal error")
    try:
        eh.fatalError(exc)
        assert False, "should have raised"
    except Exception as e:
        assert "fatal error" in str(e)
    print("error_handler_fatal_raises ok")


def test_lexical_handler_noop():
    lh = xml.sax.handler.LexicalHandler()
    assert lh.comment("a comment") is None
    assert lh.startDTD("root", None, None) is None
    assert lh.endDTD() is None
    assert lh.startCDATA() is None
    assert lh.endCDATA() is None
    print("lexical_handler_noop ok")


def test_content_handler_parse_pi():
    class H(xml.sax.handler.ContentHandler):
        def __init__(self):
            self.pis = []
        def processingInstruction(self, target, data):
            self.pis.append((target, data))

    h = H()
    xml.sax.parseString(b"<root><?proc data?></root>", h)
    assert ("proc", "data") in h.pis
    print("content_handler_parse_pi ok")


test_version()
test_feature_constants()
test_property_constants()
test_all_features_order()
test_all_properties_order()
test_content_handler_base_noop()
test_content_handler_subclass_args()
test_dtd_handler()
test_entity_resolver()
test_error_handler_warning_noop()
test_error_handler_error_raises()
test_error_handler_fatal_raises()
test_lexical_handler_noop()
test_content_handler_parse_pi()

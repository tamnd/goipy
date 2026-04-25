import io
import xml.dom
import xml.dom.minidom


def test_writexml_basic():
    doc = xml.dom.minidom.parseString('<root x="1"><child>hello</child></root>')
    w = io.StringIO()
    doc.writexml(w)
    s = w.getvalue()
    assert '<?xml version="1.0" ?>' in s, s
    assert '<root x="1">' in s, s
    assert '<child>hello</child>' in s, s
    print("writexml_basic ok")


def test_writexml_indent():
    doc = xml.dom.minidom.parseString('<root><child>hello</child></root>')
    w = io.StringIO()
    doc.writexml(w, addindent='  ', newl='\n')
    s = w.getvalue()
    assert '<root>' in s, repr(s)
    assert '  <child>hello</child>' in s, repr(s)
    print("writexml_indent ok")


def test_writexml_single_text_inline():
    doc = xml.dom.minidom.parseString('<root><child>text</child></root>')
    root = doc.documentElement
    child = root.firstChild
    w = io.StringIO()
    child.writexml(w, indent='', addindent='  ', newl='\n')
    s = w.getvalue()
    assert s == '<child>text</child>\n', repr(s)
    print("writexml_single_text_inline ok")


def test_toxml_encoding():
    doc = xml.dom.minidom.parseString('<root/>')
    result = doc.toxml(encoding='utf-8')
    assert isinstance(result, bytes), type(result)
    assert result.startswith(b'<?xml version="1.0" encoding="utf-8"?>'), result[:50]
    print("toxml_encoding ok")


def test_toxml_standalone():
    doc = xml.dom.minidom.parseString('<root/>')
    s = doc.toxml(standalone=True)
    assert 'standalone="yes"' in s, s
    s2 = doc.toxml(standalone=False)
    assert 'standalone="no"' in s2, s2
    s3 = doc.toxml()
    assert 'standalone' not in s3, s3
    print("toxml_standalone ok")


def test_toprettyxml_newl_standalone():
    doc = xml.dom.minidom.parseString('<root><child/></root>')
    s = doc.toprettyxml(indent='  ', newl='\n', standalone=True)
    assert 'standalone="yes"' in s, s
    assert '\n' in s, repr(s)
    print("toprettyxml_newl_standalone ok")


def test_context_manager():
    with xml.dom.minidom.parseString('<root/>') as doc:
        assert doc is not None
        root = doc.documentElement
        assert root.tagName == 'root', root.tagName
    print("context_manager ok")


def test_attr_properties():
    doc = xml.dom.minidom.parseString('<root x="1"/>')
    root = doc.documentElement
    attr = root.getAttributeNode('x')
    assert attr is not None
    assert attr.specified is False, attr.specified
    assert attr.ownerElement is root, attr.ownerElement
    assert attr.localName == 'x', attr.localName
    assert attr.prefix is None, attr.prefix
    assert attr.namespaceURI is None, attr.namespaceURI
    print("attr_properties ok")


def test_document_doctype_none():
    doc = xml.dom.minidom.parseString('<root/>')
    assert doc.doctype is None, doc.doctype
    print("document_doctype_none ok")


def test_document_implementation():
    doc = xml.dom.minidom.parseString('<root/>')
    impl = doc.implementation
    assert impl is not None
    assert impl.hasFeature('XML', '1.0') is True
    print("document_implementation ok")


def test_node_localname():
    doc = xml.dom.minidom.parseString('<root><child/></root>')
    root = doc.documentElement
    assert root.localName == 'root', root.localName
    child = root.firstChild
    assert child.localName == 'child', child.localName
    t = doc.createTextNode('hi')
    assert t.localName is None, t.localName
    print("node_localname ok")


def test_prefixed_element_localname():
    doc = xml.dom.minidom.parseString('<ns:root xmlns:ns="http://example.com"/>')
    root = doc.documentElement
    assert root.localName == 'root', root.localName
    print("prefixed_element_localname ok")


test_writexml_basic()
test_writexml_indent()
test_writexml_single_text_inline()
test_toxml_encoding()
test_toxml_standalone()
test_toprettyxml_newl_standalone()
test_context_manager()
test_attr_properties()
test_document_doctype_none()
test_document_implementation()
test_node_localname()
test_prefixed_element_localname()
print("ALL OK")

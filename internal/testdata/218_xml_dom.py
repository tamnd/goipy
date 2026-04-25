import xml.dom
import xml.dom.minidom


def test_dom_constants():
    assert xml.dom.EMPTY_NAMESPACE is None
    assert xml.dom.XML_NAMESPACE == "http://www.w3.org/XML/1998/namespace"
    assert xml.dom.XMLNS_NAMESPACE == "http://www.w3.org/2000/xmlns/"
    assert xml.dom.XHTML_NAMESPACE == "http://www.w3.org/1999/xhtml"
    assert xml.dom.ELEMENT_NODE == 1
    assert xml.dom.ATTRIBUTE_NODE == 2
    assert xml.dom.TEXT_NODE == 3
    assert xml.dom.CDATA_SECTION_NODE == 4
    assert xml.dom.PROCESSING_INSTRUCTION_NODE == 7
    assert xml.dom.COMMENT_NODE == 8
    assert xml.dom.DOCUMENT_NODE == 9
    assert xml.dom.DOCUMENT_TYPE_NODE == 10
    assert xml.dom.DOCUMENT_FRAGMENT_NODE == 11
    print("dom_constants ok")


def test_dom_exception_hierarchy():
    e = xml.dom.IndexSizeErr("bad index")
    assert isinstance(e, xml.dom.DOMException)
    assert e.code == 1

    e2 = xml.dom.HierarchyRequestErr()
    assert e2.code == 3

    e3 = xml.dom.NotFoundErr()
    assert isinstance(e3, xml.dom.DOMException)
    assert e3.code == 8

    e4 = xml.dom.NotSupportedErr()
    assert e4.code == 9

    e5 = xml.dom.InvalidCharacterErr()
    assert e5.code == 5

    e6 = xml.dom.NamespaceErr()
    assert e6.code == 14
    print("dom_exception_hierarchy ok")


def test_dom_implementation():
    impl = xml.dom.getDOMImplementation()
    assert impl is not None
    assert impl.hasFeature("XML", "1.0") is True
    assert impl.hasFeature("Core", "2.0") is True
    impl2 = xml.dom.getDOMImplementation()
    assert impl is impl2
    print("dom_implementation ok")


def test_nodelist():
    doc = xml.dom.minidom.parseString("<root><a/><b/><c/></root>")
    root = doc.documentElement
    nl = root.childNodes
    assert nl.length == 3, nl.length
    assert len(nl) == 3, len(nl)
    assert nl.item(0).tagName == "a"
    assert nl.item(1).tagName == "b"
    assert nl.item(2).tagName == "c"
    assert nl.item(99) is None
    child_d = doc.createElement("d")
    root.appendChild(child_d)
    assert nl.length == 4, nl.length
    print("nodelist ok")


def test_named_node_map():
    doc = xml.dom.minidom.parseString('<root x="1" y="2"/>')
    root = doc.documentElement
    attrs = root.attributes
    assert attrs.length == 2, attrs.length
    assert len(attrs) == 2, len(attrs)
    item_x = attrs.getNamedItem("x")
    assert item_x is not None
    assert item_x.value == "1", item_x.value
    assert "x" in attrs
    assert "z" not in attrs
    assert attrs.getNamedItem("z") is None
    print("named_node_map ok")


def test_node_is_same():
    doc = xml.dom.minidom.parseString("<root><a/></root>")
    root = doc.documentElement
    a = root.childNodes.item(0)
    a2 = root.getElementsByTagName("a")[0]
    assert root.isSameNode(root) is True
    assert a.isSameNode(a2) is True
    assert a.isSameNode(root) is False
    print("node_is_same ok")


def test_get_attribute_node():
    doc = xml.dom.minidom.parseString('<root id="42" class="foo"/>')
    root = doc.documentElement
    attr = root.getAttributeNode("id")
    assert attr is not None
    assert attr.name == "id", attr.name
    assert attr.value == "42", attr.value
    assert attr.nodeName == "id"
    assert attr.nodeValue == "42"
    assert root.getAttributeNode("missing") is None
    print("get_attribute_node ok")


def test_set_attribute_node():
    doc = xml.dom.minidom.parseString("<root/>")
    root = doc.documentElement
    attr = doc.createAttribute("href")
    attr.value = "https://example.com"
    root.setAttributeNode(attr)
    assert root.getAttribute("href") == "https://example.com"
    print("set_attribute_node ok")


def test_document_ns():
    doc = xml.dom.minidom.parseString("<root/>")
    elem = doc.createElementNS("http://www.w3.org/1999/xhtml", "html:div")
    assert elem.tagName == "html:div", elem.tagName
    cdata = doc.createCDATASection("hello <world>")
    assert cdata.nodeType == xml.dom.CDATA_SECTION_NODE
    assert cdata.data == "hello <world>", cdata.data
    print("document_ns ok")


def test_named_node_map_remove():
    doc = xml.dom.minidom.parseString('<root x="1" y="2" z="3"/>')
    root = doc.documentElement
    attrs = root.attributes
    assert attrs.length == 3
    attrs.removeNamedItem("y")
    assert attrs.length == 2, attrs.length
    assert attrs.getNamedItem("y") is None
    assert attrs.getNamedItem("x") is not None
    print("named_node_map_remove ok")


test_dom_constants()
test_dom_exception_hierarchy()
test_dom_implementation()
test_nodelist()
test_named_node_map()
test_node_is_same()
test_get_attribute_node()
test_set_attribute_node()
test_document_ns()
test_named_node_map_remove()
print("ALL OK")

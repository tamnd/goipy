import xml.dom
import xml.dom.minidom


def test_firstchild_lastchild():
    doc = xml.dom.minidom.parseString("<root><a/><b/><c/></root>")
    root = doc.documentElement
    assert root.firstChild.tagName == "a", root.firstChild.tagName
    assert root.lastChild.tagName == "c", root.lastChild.tagName
    a = root.firstChild
    assert a.firstChild is None
    assert a.lastChild is None
    print("firstchild_lastchild ok")


def test_prev_next_sibling():
    doc = xml.dom.minidom.parseString("<root><a/><b/><c/></root>")
    root = doc.documentElement
    a = root.childNodes.item(0)
    b = root.childNodes.item(1)
    c = root.childNodes.item(2)
    assert a.previousSibling is None
    assert a.nextSibling.tagName == "b", a.nextSibling.tagName
    assert b.previousSibling.tagName == "a", b.previousSibling.tagName
    assert b.nextSibling.tagName == "c", b.nextSibling.tagName
    assert c.nextSibling is None
    print("prev_next_sibling ok")


def test_clone_shallow():
    doc = xml.dom.minidom.parseString('<root x="1"><child/></root>')
    root = doc.documentElement
    clone = root.cloneNode(False)
    assert clone is not root
    assert clone.tagName == "root", clone.tagName
    assert clone.getAttribute("x") == "1", clone.getAttribute("x")
    assert clone.firstChild is None
    print("clone_shallow ok")


def test_clone_deep():
    doc = xml.dom.minidom.parseString('<root x="1"><child y="2"/></root>')
    root = doc.documentElement
    clone = root.cloneNode(True)
    assert clone is not root
    assert clone.tagName == "root"
    assert clone.getAttribute("x") == "1"
    assert clone.firstChild is not None
    assert clone.firstChild.tagName == "child"
    assert clone.firstChild.getAttribute("y") == "2"
    root.setAttribute("x", "99")
    assert clone.getAttribute("x") == "1"
    print("clone_deep ok")


def test_normalize():
    doc = xml.dom.minidom.parseString("<root/>")
    root = doc.documentElement
    t1 = doc.createTextNode("hello ")
    t2 = doc.createTextNode("world")
    root.appendChild(t1)
    root.appendChild(t2)
    assert root.childNodes.length == 2
    root.normalize()
    assert root.childNodes.length == 1
    assert root.firstChild.data == "hello world", root.firstChild.data
    print("normalize ok")


def test_chardata_methods():
    doc = xml.dom.minidom.parseString("<root/>")
    t = doc.createTextNode("Hello, World!")
    assert t.length == 13
    assert t.substringData(0, 5) == "Hello"
    assert t.substringData(7, 5) == "World"
    t.appendData(" More")
    assert t.data == "Hello, World! More", t.data
    t.insertData(5, " there")
    assert t.data == "Hello there, World! More", t.data
    t.deleteData(5, 6)
    assert t.data == "Hello, World! More", t.data
    t.replaceData(7, 5, "Python")
    assert t.data == "Hello, Python! More", t.data
    print("chardata_methods ok")


def test_chardata_assignment():
    doc = xml.dom.minidom.parseString("<root/>")
    c = doc.createComment("original")
    assert c.data == "original"
    c.data = "modified"
    assert c.data == "modified", c.data
    c.appendData("!")
    assert c.data == "modified!", c.data
    print("chardata_assignment ok")


def test_split_text():
    doc = xml.dom.minidom.parseString("<root/>")
    root = doc.documentElement
    t = doc.createTextNode("HelloWorld")
    root.appendChild(t)
    tail = t.splitText(5)
    assert t.data == "Hello", t.data
    assert tail.data == "World", tail.data
    assert root.childNodes.length == 2
    assert root.firstChild.data == "Hello"
    assert root.lastChild.data == "World"
    print("split_text ok")


def test_document_fragment():
    doc = xml.dom.minidom.parseString("<root/>")
    frag = doc.createDocumentFragment()
    assert frag.nodeType == xml.dom.DOCUMENT_FRAGMENT_NODE
    a = doc.createElement("a")
    b = doc.createElement("b")
    frag.appendChild(a)
    frag.appendChild(b)
    assert frag.firstChild.tagName == "a"
    assert frag.lastChild.tagName == "b"
    assert frag.childNodes.length == 2
    print("document_fragment ok")


def test_import_node():
    doc1 = xml.dom.minidom.parseString('<root id="1"><child/></root>')
    doc2 = xml.dom.minidom.parseString("<other/>")
    root1 = doc1.documentElement
    imported = doc2.importNode(root1, True)
    assert imported is not root1
    assert imported.tagName == "root"
    assert imported.getAttribute("id") == "1"
    assert imported.firstChild.tagName == "child"
    print("import_node ok")


def test_named_node_map_ns():
    doc = xml.dom.minidom.parseString('<root x="1" y="2"/>')
    root = doc.documentElement
    attrs = root.attributes
    # unqualified attrs have namespaceURI=None
    item = attrs.getNamedItemNS(None, "x")
    assert item is not None
    assert item.value == "1", item.value
    # setNamedItemNS
    new_attr = doc.createAttributeNS(None, "z")
    new_attr.value = "3"
    attrs.setNamedItemNS(new_attr)
    assert attrs.getNamedItem("z").value == "3"
    # removeNamedItemNS
    removed = attrs.removeNamedItemNS(None, "z")
    assert removed.value == "3", removed.value
    assert attrs.getNamedItem("z") is None
    print("named_node_map_ns ok")


def test_has_attribute_ns():
    doc = xml.dom.minidom.parseString('<root x="1" y="2"/>')
    root = doc.documentElement
    assert root.hasAttributeNS(None, "x") is True
    assert root.hasAttributeNS(None, "z") is False
    print("has_attribute_ns ok")


def test_dom_implementation_create_document():
    impl = xml.dom.getDOMImplementation()
    doc = impl.createDocument(None, None, None)
    assert doc is not None
    assert doc.nodeType == xml.dom.DOCUMENT_NODE
    elem = doc.createElement("test")
    doc.appendChild(elem)
    assert doc.documentElement.tagName == "test"
    print("dom_implementation_create_document ok")


test_firstchild_lastchild()
test_prev_next_sibling()
test_clone_shallow()
test_clone_deep()
test_normalize()
test_chardata_methods()
test_chardata_assignment()
test_split_text()
test_document_fragment()
test_import_node()
test_named_node_map_ns()
test_has_attribute_ns()
test_dom_implementation_create_document()
print("ALL OK")

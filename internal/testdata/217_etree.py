import xml.etree.ElementTree as ET
import io


def test_comment_serialization():
    root = ET.Element("root")
    c = ET.Comment("hello")
    root.append(c)
    result = ET.tostring(root, encoding='unicode')
    assert "<!--hello-->" in result, repr(result)
    print("comment_serialization ok")


def test_pi_serialization():
    root = ET.Element("root")
    pi = ET.ProcessingInstruction("target", "data")
    root.append(pi)
    result = ET.tostring(root, encoding='unicode')
    assert "<?target data?>" in result, repr(result)
    print("pi_serialization ok")


def test_qname_basic():
    qn = ET.QName("{http://example.com}tag")
    assert qn.text == "{http://example.com}tag", qn.text
    assert qn.namespace == "http://example.com", qn.namespace
    assert qn.localname == "tag", qn.localname
    assert str(qn) == "{http://example.com}tag"
    print("qname_basic ok")


def test_qname_uri_tag():
    qn = ET.QName("http://example.com", "tag")
    assert qn.text == "{http://example.com}tag", qn.text
    assert qn.namespace == "http://example.com", qn.namespace
    assert qn.localname == "tag", qn.localname
    print("qname_uri_tag ok")


def test_qname_no_ns():
    qn = ET.QName("localonly")
    assert qn.text == "localonly", qn.text
    assert qn.namespace is None, qn.namespace
    assert qn.localname == "localonly", qn.localname
    print("qname_no_ns ok")


def test_qname_eq():
    qn1 = ET.QName("{http://ns}tag")
    qn2 = ET.QName("{http://ns}tag")
    qn3 = ET.QName("{http://ns}other")
    assert qn1 == qn2, "equal QNames should be equal"
    assert not (qn1 == qn3), "different QNames should not be equal"
    print("qname_eq ok")


def test_treebuilder_basic():
    b = ET.TreeBuilder()
    b.start("root", {"id": "1"})
    b.data("text")
    b.start("child", {})
    b.data("child text")
    b.end("child")
    b.end("root")
    root = b.close()
    assert root.tag == "root", root.tag
    assert root.get("id") == "1", root.get("id")
    assert root.text == "text", repr(root.text)
    assert len(root) == 1, len(root)
    assert root[0].tag == "child", root[0].tag
    assert root[0].text == "child text", repr(root[0].text)
    print("treebuilder_basic ok")


def test_treebuilder_nested():
    b = ET.TreeBuilder()
    b.start("a", {})
    b.start("b", {})
    b.data("val")
    b.end("b")
    b.end("a")
    root = b.close()
    assert root.tag == "a"
    assert root[0].tag == "b"
    assert root[0].text == "val", repr(root[0].text)
    print("treebuilder_nested ok")


def test_iterparse_start_end():
    src = io.StringIO("<root><child/><child/></root>")
    events = list(ET.iterparse(src, events=("start", "end")))
    tags = [e for e, _ in events]
    assert tags.count("start") == 3, tags.count("start")
    assert tags.count("end") == 3, tags.count("end")
    print("iterparse_start_end ok")


def test_iterparse_end_only():
    src = io.StringIO("<root><a/><b/></root>")
    events = list(ET.iterparse(src, events=("end",)))
    assert all(e == "end" for e, _ in events), events
    assert len(events) == 3, len(events)
    print("iterparse_end_only ok")


def test_iterparse_start_only():
    src = io.StringIO("<root><child/></root>")
    events = list(ET.iterparse(src, events=("start",)))
    assert all(e == "start" for e, _ in events), events
    assert len(events) == 2, len(events)
    print("iterparse_start_only ok")


def test_canonicalize():
    xml_str = '<root b="2" a="1"><child/></root>'
    result = ET.canonicalize(xml_str)
    assert result is not None
    assert 'a="1"' in result, repr(result)
    assert 'b="2"' in result, repr(result)
    assert result.index('a=') < result.index('b='), repr(result)
    print("canonicalize ok")


test_comment_serialization()
test_pi_serialization()
test_qname_basic()
test_qname_uri_tag()
test_qname_no_ns()
test_qname_eq()
test_treebuilder_basic()
test_treebuilder_nested()
test_iterparse_start_end()
test_iterparse_end_only()
test_iterparse_start_only()
test_canonicalize()
print("ALL OK")

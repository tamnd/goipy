import xml.etree.ElementTree as ET
import xml.sax
import xml.sax.handler
import xml.sax.saxutils
import xml.dom
import xml.dom.minidom
import xml.dom.pulldom
import xml.parsers.expat

SAMPLE = '''<?xml version="1.0"?>
<root attr="val">
  <child id="1">text1</child>
  <child id="2">text2</child>
  <empty/>
</root>'''

def test_et_fromstring():
    root = ET.fromstring(SAMPLE)
    assert root.tag == 'root', f"tag={root.tag!r}"
    assert root.get('attr') == 'val'
    print("et_fromstring ok")

def test_et_children():
    root = ET.fromstring(SAMPLE)
    children = list(root)
    assert len(children) == 3, f"len={len(children)}"
    assert children[0].tag == 'child'
    assert children[0].get('id') == '1'
    assert children[0].text == 'text1'
    print("et_children ok")

def test_et_find():
    root = ET.fromstring(SAMPLE)
    child = root.find('child')
    assert child is not None
    assert child.get('id') == '1'
    all_children = root.findall('child')
    assert len(all_children) == 2
    print("et_find ok")

def test_et_iter():
    root = ET.fromstring(SAMPLE)
    tags = [e.tag for e in root.iter()]
    assert 'root' in tags
    assert tags.count('child') == 2
    print("et_iter ok")

def test_et_tostring():
    root = ET.fromstring('<a><b>text</b></a>')
    s = ET.tostring(root, encoding='unicode')
    assert '<a>' in s and '<b>text</b>' in s, f"got {s!r}"
    print("et_tostring ok")

def test_et_modify():
    root = ET.Element('root')
    child = ET.SubElement(root, 'item', id='1')
    child.text = 'hello'
    assert len(root) == 1
    assert root.find('item').text == 'hello'
    root.remove(child)
    assert len(root) == 0
    print("et_modify ok")

def test_et_attribs():
    e = ET.Element('tag', {'a': '1', 'b': '2'})
    assert e.get('a') == '1'
    e.set('c', '3')
    assert 'c' in e.keys()
    assert e.get('missing', 'def') == 'def'
    print("et_attribs ok")

def test_et_indent():
    root = ET.fromstring('<root><child>x</child></root>')
    ET.indent(root)
    s = ET.tostring(root, encoding='unicode')
    assert '\n' in s or '  ' in s, f"got {s!r}"
    print("et_indent ok")

_sax_events = []

class _SaxParseH(xml.sax.handler.ContentHandler):
    def startElement(self, name, attrs):
        _sax_events.append(('start', name))
    def endElement(self, name):
        _sax_events.append(('end', name))
    def characters(self, content):
        if content.strip():
            _sax_events.append(('chars', content.strip()))

def test_sax_parse():
    del _sax_events[:]
    xml.sax.parseString(SAMPLE.encode(), _SaxParseH())
    starts = [e[1] for e in _sax_events if e[0] == 'start']
    assert 'root' in starts and 'child' in starts, f"starts={starts}"
    print("sax_parse ok")

_sax_attr_result = {}

class _SaxAttrH(xml.sax.handler.ContentHandler):
    def startElement(self, name, attrs):
        if name == 'root':
            _sax_attr_result['attr'] = attrs.getValue('attr')

def test_sax_attributes():
    _sax_attr_result.clear()
    xml.sax.parseString(SAMPLE.encode(), _SaxAttrH())
    assert _sax_attr_result.get('attr') == 'val', f"got {_sax_attr_result}"
    print("sax_attributes ok")

def test_saxutils_escape():
    assert xml.sax.saxutils.escape('<a>&b</a>') == '&lt;a&gt;&amp;b&lt;/a&gt;'
    assert xml.sax.saxutils.unescape('&lt;&gt;&amp;') == '<>&'
    q = xml.sax.saxutils.quoteattr('hello "world"')
    assert '"' in q or "'" in q
    print("saxutils_escape ok")

def test_dom_constants():
    assert xml.dom.ELEMENT_NODE == 1
    assert xml.dom.TEXT_NODE == 3
    assert xml.dom.DOCUMENT_NODE == 9
    assert xml.dom.COMMENT_NODE == 8
    print("dom_constants ok")

def test_minidom_parse():
    doc = xml.dom.minidom.parseString(SAMPLE.encode())
    root = doc.documentElement
    assert root.tagName == 'root'
    children = root.getElementsByTagName('child')
    assert len(children) == 2
    assert children[0].getAttribute('id') == '1'
    print("minidom_parse ok")

def test_minidom_create():
    doc = xml.dom.minidom.Document()
    root = doc.createElement('root')
    doc.appendChild(root)
    child = doc.createElement('item')
    child.setAttribute('x', '1')
    root.appendChild(child)
    txt = doc.createTextNode('hello')
    child.appendChild(txt)
    xml_str = doc.toxml()
    assert 'root' in xml_str and 'item' in xml_str
    print("minidom_create ok")

def test_pulldom():
    events = list(xml.dom.pulldom.parseString(SAMPLE.encode()))
    event_types = [e[0] for e in events]
    assert xml.dom.pulldom.START_ELEMENT in event_types
    assert xml.dom.pulldom.END_ELEMENT in event_types
    print("pulldom ok")

def test_expat():
    p = xml.parsers.expat.ParserCreate()
    starts = []
    p.StartElementHandler = lambda name, attrs: starts.append(name)
    p.Parse(SAMPLE, True)
    assert 'root' in starts, f"starts={starts}"
    print("expat ok")

test_et_fromstring()
test_et_children()
test_et_find()
test_et_iter()
test_et_tostring()
test_et_modify()
test_et_attribs()
test_et_indent()
test_sax_parse()
test_sax_attributes()
test_saxutils_escape()
test_dom_constants()
test_minidom_parse()
test_minidom_create()
test_pulldom()
test_expat()
print("ALL OK")

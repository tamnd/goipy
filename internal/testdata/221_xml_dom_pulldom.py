import xml.dom.pulldom


def test_constants():
    assert xml.dom.pulldom.START_ELEMENT == "START_ELEMENT"
    assert xml.dom.pulldom.END_ELEMENT == "END_ELEMENT"
    assert xml.dom.pulldom.COMMENT == "COMMENT"
    assert xml.dom.pulldom.START_DOCUMENT == "START_DOCUMENT"
    assert xml.dom.pulldom.END_DOCUMENT == "END_DOCUMENT"
    assert xml.dom.pulldom.CHARACTERS == "CHARACTERS"
    assert xml.dom.pulldom.PROCESSING_INSTRUCTION == "PROCESSING_INSTRUCTION"
    assert xml.dom.pulldom.IGNORABLE_WHITESPACE == "IGNORABLE_WHITESPACE"
    print("constants ok")


def test_default_bufsize():
    assert xml.dom.pulldom.default_bufsize == 8192
    print("default_bufsize ok")


def test_classes_exist():
    assert hasattr(xml.dom.pulldom, 'DOMEventStream')
    assert hasattr(xml.dom.pulldom, 'PullDOM')
    assert hasattr(xml.dom.pulldom, 'SAX2DOM')
    print("classes_exist ok")


def test_start_document_first():
    events = xml.dom.pulldom.parseString('<root/>')
    event, node = events.getEvent()
    assert event == xml.dom.pulldom.START_DOCUMENT, event
    print("start_document_first ok")


def test_end_document_last():
    events = xml.dom.pulldom.parseString('<root/>')
    all_events = list(events)
    last_event, _ = all_events[-1]
    assert last_event == xml.dom.pulldom.END_DOCUMENT, last_event
    print("end_document_last ok")


def test_event_types_simple():
    events = xml.dom.pulldom.parseString('<root/>')
    types = [ev for ev, _ in events]
    assert xml.dom.pulldom.START_DOCUMENT in types
    assert xml.dom.pulldom.START_ELEMENT in types
    assert xml.dom.pulldom.END_ELEMENT in types
    assert xml.dom.pulldom.END_DOCUMENT in types
    print("event_types_simple ok")


def test_start_element_node():
    events = xml.dom.pulldom.parseString('<root x="1" y="2"/>')
    for event, node in events:
        if event == xml.dom.pulldom.START_ELEMENT:
            assert node.tagName == 'root', node.tagName
            assert node.nodeName == 'root', node.nodeName
            assert node.getAttribute('x') == '1', node.getAttribute('x')
            assert node.getAttribute('y') == '2', node.getAttribute('y')
            print("start_element_node ok")
            return
    assert False, "no START_ELEMENT found"


def test_characters_node():
    events = xml.dom.pulldom.parseString('<root>hello</root>')
    for event, node in events:
        if event == xml.dom.pulldom.CHARACTERS:
            assert node.data == 'hello', node.data
            print("characters_node ok")
            return
    assert False, "no CHARACTERS found"


def test_comment_node():
    events = xml.dom.pulldom.parseString('<root><!-- my comment --></root>')
    for event, node in events:
        if event == xml.dom.pulldom.COMMENT:
            assert ' my comment ' in node.data, node.data
            print("comment_node ok")
            return
    assert False, "no COMMENT found"


def test_pi_node():
    events = xml.dom.pulldom.parseString('<root><?proc value?></root>')
    for event, node in events:
        if event == xml.dom.pulldom.PROCESSING_INSTRUCTION:
            assert node.target == 'proc', node.target
            assert node.data == 'value', node.data
            print("pi_node ok")
            return
    assert False, "no PROCESSING_INSTRUCTION found"


def test_end_element_same_node():
    events = xml.dom.pulldom.parseString('<root/>')
    start_node = None
    end_node = None
    for event, node in events:
        if event == xml.dom.pulldom.START_ELEMENT:
            start_node = node
        elif event == xml.dom.pulldom.END_ELEMENT:
            end_node = node
    assert start_node is not None
    assert end_node is not None
    assert start_node is end_node, "START and END should be same node instance"
    print("end_element_same_node ok")


def test_get_event():
    events = xml.dom.pulldom.parseString('<root/>')
    ev1, _ = events.getEvent()
    assert ev1 == xml.dom.pulldom.START_DOCUMENT, ev1
    ev2, node2 = events.getEvent()
    assert ev2 == xml.dom.pulldom.START_ELEMENT, ev2
    assert node2.tagName == 'root', node2.tagName
    print("get_event ok")


def test_reset():
    events = xml.dom.pulldom.parseString('<root/>')
    first_pass = [ev for ev, _ in events]
    events.reset()
    second_pass = [ev for ev, _ in events]
    assert first_pass == second_pass, (first_pass, second_pass)
    print("reset ok")


def test_expand_node_children():
    events = xml.dom.pulldom.parseString('<root><a>text</a><b/></root>')
    for event, node in events:
        if event == xml.dom.pulldom.START_ELEMENT and node.tagName == 'root':
            events.expandNode(node)
            assert node.childNodes.length == 2, node.childNodes.length
            a = node.firstChild
            assert a.tagName == 'a', a.tagName
            b = node.lastChild
            assert b.tagName == 'b', b.tagName
            print("expand_node_children ok")
            return
    assert False, "no root element found"


def test_expand_node_text_child():
    events = xml.dom.pulldom.parseString('<root><item>hello</item></root>')
    for event, node in events:
        if event == xml.dom.pulldom.START_ELEMENT and node.tagName == 'root':
            events.expandNode(node)
            item = node.firstChild
            assert item.tagName == 'item', item.tagName
            text = item.firstChild
            assert text.data == 'hello', text.data
            print("expand_node_text_child ok")
            return
    assert False, "no root found"


def test_get_attribute_after_expand():
    events = xml.dom.pulldom.parseString('<root><child id="42"/></root>')
    for event, node in events:
        if event == xml.dom.pulldom.START_ELEMENT and node.tagName == 'root':
            events.expandNode(node)
            child = node.firstChild
            assert child.getAttribute('id') == '42', child.getAttribute('id')
            print("get_attribute_after_expand ok")
            return
    assert False, "no root found"


def test_iterable():
    events = xml.dom.pulldom.parseString('<root><a/><b/></root>')
    count = sum(1 for ev, _ in events if ev == xml.dom.pulldom.START_ELEMENT)
    assert count == 3, count  # root, a, b
    print("iterable ok")


def test_get_event_exhausted():
    events = xml.dom.pulldom.parseString('<r/>')
    while True:
        result = events.getEvent()
        if result is None:
            break
    # further calls return None
    assert events.getEvent() is None
    print("get_event_exhausted ok")


test_constants()
test_default_bufsize()
test_classes_exist()
test_start_document_first()
test_end_document_last()
test_event_types_simple()
test_start_element_node()
test_characters_node()
test_comment_node()
test_pi_node()
test_end_element_same_node()
test_get_event()
test_reset()
test_expand_node_children()
test_expand_node_text_child()
test_get_attribute_after_expand()
test_iterable()
test_get_event_exhausted()
print("ALL OK")

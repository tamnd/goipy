import xml.sax.saxutils


def test_escape_basic():
    assert xml.sax.saxutils.escape("hello") == "hello"
    assert xml.sax.saxutils.escape("<tag>") == "&lt;tag&gt;"
    assert xml.sax.saxutils.escape("a & b") == "a &amp; b"
    assert xml.sax.saxutils.escape("a > b < c") == "a &gt; b &lt; c"
    print("escape_basic ok")


def test_escape_amp_first():
    # & must be escaped before < and >
    assert xml.sax.saxutils.escape("&lt;") == "&amp;lt;"
    assert xml.sax.saxutils.escape("&amp;") == "&amp;amp;"
    print("escape_amp_first ok")


def test_escape_custom_entities():
    assert xml.sax.saxutils.escape('"hello"', {'"': "&quot;"}) == "&quot;hello&quot;"
    assert xml.sax.saxutils.escape("hello", {"h": "H"}) == "Hello"
    print("escape_custom_entities ok")


def test_unescape_basic():
    assert xml.sax.saxutils.unescape("&lt;tag&gt;") == "<tag>"
    assert xml.sax.saxutils.unescape("a &amp; b") == "a & b"
    assert xml.sax.saxutils.unescape("hello") == "hello"
    print("unescape_basic ok")


def test_unescape_amp_last():
    # &amp;lt; -> &lt; (not <)
    assert xml.sax.saxutils.unescape("&amp;lt;") == "&lt;"
    assert xml.sax.saxutils.unescape("&amp;amp;") == "&amp;"
    print("unescape_amp_last ok")


def test_unescape_custom_entities():
    assert xml.sax.saxutils.unescape("&quot;hello&quot;", {"&quot;": '"'}) == '"hello"'
    print("unescape_custom_entities ok")


def test_quoteattr_basic():
    assert xml.sax.saxutils.quoteattr("hello") == '"hello"'
    assert xml.sax.saxutils.quoteattr("say &amp; go") == '"say &amp;amp; go"'
    print("quoteattr_basic ok")


def test_quoteattr_whitespace():
    assert xml.sax.saxutils.quoteattr("hello\nworld") == '"hello&#10;world"'
    assert xml.sax.saxutils.quoteattr("a\tb") == '"a&#9;b"'
    assert xml.sax.saxutils.quoteattr("a\rb") == '"a&#13;b"'
    print("quoteattr_whitespace ok")


def test_quoteattr_double_quote():
    # has double quote but no single -> wrap in single
    result = xml.sax.saxutils.quoteattr('say "hi"')
    assert result == '\'say "hi"\''
    print("quoteattr_double_quote ok")


def test_quoteattr_single_quote():
    # has single quote but no double -> wrap in double
    result = xml.sax.saxutils.quoteattr("it's fine")
    assert result == '"it\'s fine"'
    print("quoteattr_single_quote ok")


def test_quoteattr_both_quotes():
    # has both -> wrap in double, replace " with &quot;
    result = xml.sax.saxutils.quoteattr('both"and\'')
    assert result == '"both&quot;and\'"'
    print("quoteattr_both_quotes ok")


def test_xmlgenerator_start_document():
    import io
    out = io.StringIO()
    gen = xml.sax.saxutils.XMLGenerator(out, "utf-8")
    gen.startDocument()
    assert out.getvalue() == '<?xml version="1.0" encoding="utf-8"?>\n'
    print("xmlgenerator_start_document ok")


def test_xmlgenerator_basic():
    import io
    out = io.StringIO()
    gen = xml.sax.saxutils.XMLGenerator(out, "utf-8")
    gen.startElement("root", {})
    gen.characters("hello")
    gen.endElement("root")
    assert out.getvalue() == "<root>hello</root>"
    print("xmlgenerator_basic ok")


def test_xmlgenerator_attrs():
    import io
    out = io.StringIO()
    gen = xml.sax.saxutils.XMLGenerator(out)
    gen.startElement("a", {"href": 'say "hi"'})
    gen.endElement("a")
    result = out.getvalue()
    # double quote in attr value -> single-quote wrapping
    assert "href='say \"hi\"'" in result
    print("xmlgenerator_attrs ok")


def test_xmlgenerator_short_empty():
    import io
    out = io.StringIO()
    gen = xml.sax.saxutils.XMLGenerator(out, short_empty_elements=True)
    gen.startElement("br", {})
    gen.endElement("br")
    assert out.getvalue() == "<br/>"
    print("xmlgenerator_short_empty ok")


def test_xmlgenerator_short_empty_with_content():
    import io
    out = io.StringIO()
    gen = xml.sax.saxutils.XMLGenerator(out, short_empty_elements=True)
    gen.startElement("p", {})
    gen.characters("hi")
    gen.endElement("p")
    assert out.getvalue() == "<p>hi</p>"
    print("xmlgenerator_short_empty_with_content ok")


def test_xmlgenerator_characters_empty():
    import io
    out = io.StringIO()
    gen = xml.sax.saxutils.XMLGenerator(out)
    gen.startElement("x", {})
    gen.characters("")
    gen.endElement("x")
    assert out.getvalue() == "<x></x>"
    print("xmlgenerator_characters_empty ok")


def test_xmlgenerator_processing_instruction():
    import io
    out = io.StringIO()
    gen = xml.sax.saxutils.XMLGenerator(out)
    gen.processingInstruction("xml-stylesheet", "type='text/css'")
    assert out.getvalue() == "<?xml-stylesheet type='text/css'?>"
    print("xmlgenerator_processing_instruction ok")


def test_xmlfilterbase_parent():
    filt = xml.sax.saxutils.XMLFilterBase()
    assert filt.getParent() is None
    filt.setParent("mock")
    assert filt.getParent() == "mock"
    print("xmlfilterbase_parent ok")


def test_xmlfilterbase_content_handler():
    import xml.sax.handler
    filt = xml.sax.saxutils.XMLFilterBase()
    ch = xml.sax.handler.ContentHandler()
    filt.setContentHandler(ch)
    assert filt.getContentHandler() is ch
    print("xmlfilterbase_content_handler ok")


def test_prepare_input_source_passthrough():
    import xml.sax.xmlreader, io
    src = xml.sax.xmlreader.InputSource("myfile.xml")
    src.setByteStream(io.BytesIO(b"<x/>"))
    result = xml.sax.saxutils.prepare_input_source(src)
    assert result is src
    assert result.getSystemId() == "myfile.xml"
    print("prepare_input_source_passthrough ok")


test_escape_basic()
test_escape_amp_first()
test_escape_custom_entities()
test_unescape_basic()
test_unescape_amp_last()
test_unescape_custom_entities()
test_quoteattr_basic()
test_quoteattr_whitespace()
test_quoteattr_double_quote()
test_quoteattr_single_quote()
test_quoteattr_both_quotes()
test_xmlgenerator_start_document()
test_xmlgenerator_basic()
test_xmlgenerator_attrs()
test_xmlgenerator_short_empty()
test_xmlgenerator_short_empty_with_content()
test_xmlgenerator_characters_empty()
test_xmlgenerator_processing_instruction()
test_xmlfilterbase_parent()
test_xmlfilterbase_content_handler()
test_prepare_input_source_passthrough()

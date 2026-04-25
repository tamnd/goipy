import pyexpat
import pyexpat.errors
import pyexpat.model


def test_module_constants():
    assert pyexpat.native_encoding == "UTF-8"
    assert isinstance(pyexpat.EXPAT_VERSION, str)
    assert pyexpat.version_info[0] >= 2
    assert len(pyexpat.version_info) == 3
    assert pyexpat.XML_PARAM_ENTITY_PARSING_NEVER == 0
    assert pyexpat.XML_PARAM_ENTITY_PARSING_UNLESS_STANDALONE == 1
    assert pyexpat.XML_PARAM_ENTITY_PARSING_ALWAYS == 2
    print("module_constants ok")


def test_expat_error_alias():
    assert pyexpat.ExpatError is pyexpat.error
    print("expat_error_alias ok")


def test_error_string():
    assert pyexpat.ErrorString(0) is None
    assert pyexpat.ErrorString(2) == "syntax error"
    assert pyexpat.ErrorString(3) == "no element found"
    assert pyexpat.ErrorString(7) == "mismatched tag"
    print("error_string ok")


def test_errors_submodule():
    assert pyexpat.errors.XML_ERROR_SYNTAX == "syntax error"
    assert pyexpat.errors.XML_ERROR_NO_ELEMENTS == "no element found"
    assert pyexpat.errors.XML_ERROR_INVALID_TOKEN == "not well-formed (invalid token)"
    assert pyexpat.errors.XML_ERROR_UNDEFINED_ENTITY == "undefined entity"
    assert pyexpat.errors.XML_ERROR_UNCLOSED_TOKEN == "unclosed token"
    print("errors_submodule ok")


def test_errors_codes_messages():
    # codes: message -> int
    assert pyexpat.errors.codes["syntax error"] == 2
    assert pyexpat.errors.codes["no element found"] == 3
    # messages: int -> message
    assert pyexpat.errors.messages[2] == "syntax error"
    assert pyexpat.errors.messages[11] == "undefined entity"
    print("errors_codes_messages ok")


def test_model_submodule():
    assert pyexpat.model.XML_CTYPE_EMPTY == 1
    assert pyexpat.model.XML_CTYPE_ANY == 2
    assert pyexpat.model.XML_CTYPE_MIXED == 3
    assert pyexpat.model.XML_CTYPE_NAME == 4
    assert pyexpat.model.XML_CTYPE_CHOICE == 5
    assert pyexpat.model.XML_CTYPE_SEQ == 6
    assert pyexpat.model.XML_CQUANT_NONE == 0
    assert pyexpat.model.XML_CQUANT_OPT == 1
    assert pyexpat.model.XML_CQUANT_REP == 2
    assert pyexpat.model.XML_CQUANT_PLUS == 3
    print("model_submodule ok")


def test_parser_create():
    p = pyexpat.ParserCreate()
    assert type(p) is pyexpat.XMLParserType
    print("parser_create ok")


def test_parser_attrs():
    p = pyexpat.ParserCreate()
    assert p.buffer_size == 8192
    assert p.buffer_text == False
    assert p.ordered_attributes == False
    assert p.specified_attributes == False
    assert isinstance(p.intern, dict)
    assert p.ErrorCode == 0
    assert p.ErrorLineNumber == 1
    assert p.ErrorColumnNumber == 0
    assert p.ErrorByteIndex == -1
    assert p.CurrentLineNumber == 1
    print("parser_attrs ok")


def test_parse_elements():
    events = []
    p = pyexpat.ParserCreate()
    p.StartElementHandler = lambda name, attrs: events.append(("start", name, dict(attrs)))
    p.EndElementHandler = lambda name: events.append(("end", name))
    p.CharacterDataHandler = lambda data: events.append(("chars", data))

    result = p.Parse(b"<root a=\"1\">hello</root>", True)
    assert result == 1
    assert ("start", "root", {"a": "1"}) in events
    assert ("chars", "hello") in events
    assert ("end", "root") in events
    print("parse_elements ok")


def test_parse_processing_instruction():
    events = []
    p = pyexpat.ParserCreate()
    p.ProcessingInstructionHandler = lambda target, data: events.append(("pi", target, data))
    p.Parse(b"<root><?proc some-data?></root>", True)
    assert ("pi", "proc", "some-data") in events
    print("parse_processing_instruction ok")


def test_parse_comment():
    events = []
    p = pyexpat.ParserCreate()
    p.CommentHandler = lambda data: events.append(("comment", data))
    p.Parse(b"<root><!--a comment--></root>", True)
    assert ("comment", "a comment") in events
    print("parse_comment ok")


def test_parse_ordered_attributes():
    events = []
    p = pyexpat.ParserCreate()
    p.ordered_attributes = True
    p.StartElementHandler = lambda name, attrs: events.append(("start", name, attrs))
    p.Parse(b'<root z="3" a="1"/>', True)
    start = events[0]
    assert start[0] == "start"
    assert start[1] == "root"
    # flat list: [name, val, name, val, ...]
    attrs = start[2]
    assert isinstance(attrs, list)
    assert len(attrs) == 4
    d = dict(zip(attrs[0::2], attrs[1::2]))
    assert d["z"] == "3"
    assert d["a"] == "1"
    print("parse_ordered_attributes ok")


def test_xml_decl_handler():
    events = []
    p = pyexpat.ParserCreate()
    p.XmlDeclHandler = lambda version, encoding, standalone: events.append(
        ("xmldecl", version, encoding, standalone)
    )
    p.Parse(b'<?xml version="1.0" encoding="UTF-8"?><root/>', True)
    assert len(events) == 1
    ev = events[0]
    assert ev[0] == "xmldecl"
    assert ev[1] == "1.0"
    assert ev[2] == "UTF-8"
    print("xml_decl_handler ok")


def test_cdata_handlers():
    events = []
    p = pyexpat.ParserCreate()
    p.StartCdataSectionHandler = lambda: events.append("startcdata")
    p.EndCdataSectionHandler = lambda: events.append("endcdata")
    p.CharacterDataHandler = lambda d: events.append(("chars", d))
    p.Parse(b"<root><![CDATA[hello]]></root>", True)
    assert "startcdata" in events
    assert "endcdata" in events
    assert ("chars", "hello") in events
    idx_start = events.index("startcdata")
    idx_end = events.index("endcdata")
    assert idx_start < idx_end
    print("cdata_handlers ok")


def test_doctype_handlers():
    events = []
    p = pyexpat.ParserCreate()
    p.StartDoctypeDeclHandler = lambda name, sysid, pubid, has_internal: events.append(
        ("startdoctype", name, sysid, pubid, has_internal)
    )
    p.EndDoctypeDeclHandler = lambda: events.append("enddoctype")
    p.Parse(b"<!DOCTYPE root []><root/>", True)
    assert any(e[0] == "startdoctype" and e[1] == "root" for e in events if isinstance(e, tuple))
    assert "enddoctype" in events
    print("doctype_handlers ok")


def test_namespace_handlers():
    events = []
    p = pyexpat.ParserCreate("UTF-8", ":")
    p.StartNamespaceDeclHandler = lambda prefix, uri: events.append(("startns", prefix, uri))
    p.EndNamespaceDeclHandler = lambda prefix: events.append(("endns", prefix))
    p.StartElementHandler = lambda name, attrs: events.append(("start", name))
    p.Parse(b'<root xmlns:x="http://example.com"/>', True)
    assert ("startns", "x", "http://example.com") in events
    assert any(e[0] == "endns" and e[1] == "x" for e in events)
    print("namespace_handlers ok")


def test_parse_error():
    p = pyexpat.ParserCreate()
    try:
        p.Parse(b"<bad", True)
        assert False, "should have raised"
    except pyexpat.ExpatError as e:
        assert e.lineno >= 1
        assert e.code > 0
        assert isinstance(str(e), str)
    print("parse_error ok")


def test_error_attrs_after_error():
    p = pyexpat.ParserCreate()
    try:
        p.Parse(b"<root>\n<bad", True)
    except pyexpat.ExpatError:
        pass
    assert p.ErrorCode > 0
    assert p.ErrorLineNumber >= 1
    print("error_attrs_after_error ok")


def test_set_get_base():
    p = pyexpat.ParserCreate()
    assert p.GetBase() is None
    p.SetBase("http://example.com/")
    assert p.GetBase() == "http://example.com/"
    print("set_get_base ok")


def test_set_param_entity_parsing():
    p = pyexpat.ParserCreate()
    result = p.SetParamEntityParsing(pyexpat.XML_PARAM_ENTITY_PARSING_NEVER)
    assert result == 1
    print("set_param_entity_parsing ok")


def test_external_entity_parser_create():
    p = pyexpat.ParserCreate()
    sub = p.ExternalEntityParserCreate("context")
    assert type(sub) is pyexpat.XMLParserType
    print("external_entity_parser_create ok")


def test_use_foreign_dtd():
    p = pyexpat.ParserCreate()
    result = p.UseForeignDTD(True)
    assert result is None
    print("use_foreign_dtd ok")


def test_incremental_parse():
    events = []
    p = pyexpat.ParserCreate()
    p.StartElementHandler = lambda name, attrs: events.append(("start", name))
    p.EndElementHandler = lambda name: events.append(("end", name))
    p.CharacterDataHandler = lambda d: events.append(("chars", d))

    p.Parse(b"<roo", False)
    assert events == []  # nothing fired yet
    p.Parse(b"t>hello</root>", True)
    assert ("start", "root") in events
    assert ("chars", "hello") in events
    assert ("end", "root") in events
    print("incremental_parse ok")


def test_features():
    assert isinstance(pyexpat.features, list)
    assert len(pyexpat.features) > 0
    for item in pyexpat.features:
        assert len(item) == 2
    print("features ok")


def test_notation_decl_handler():
    events = []
    p = pyexpat.ParserCreate()
    p.NotationDeclHandler = lambda name, base, sysid, pubid: events.append(
        ("notation", name, sysid, pubid)
    )
    p.StartDoctypeDeclHandler = lambda *a: None
    p.EndDoctypeDeclHandler = lambda: None
    p.Parse(b'<!DOCTYPE root [<!NOTATION gif SYSTEM "image/gif">]><root/>', True)
    assert any(e[0] == "notation" and e[1] == "gif" for e in events)
    print("notation_decl_handler ok")


test_module_constants()
test_expat_error_alias()
test_error_string()
test_errors_submodule()
test_errors_codes_messages()
test_model_submodule()
test_parser_create()
test_parser_attrs()
test_parse_elements()
test_parse_processing_instruction()
test_parse_comment()
test_parse_ordered_attributes()
test_xml_decl_handler()
test_cdata_handlers()
test_doctype_handlers()
test_namespace_handlers()
test_parse_error()
test_error_attrs_after_error()
test_set_get_base()
test_set_param_entity_parsing()
test_external_entity_parser_create()
test_use_foreign_dtd()
test_incremental_parse()
test_features()
test_notation_decl_handler()

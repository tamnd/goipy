import html
import html.entities
from html.parser import HTMLParser

def test_escape_basic():
    assert html.escape('<script>alert("xss")</script>') == '&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;'
    assert html.escape("it's") == 'it&#x27;s'
    assert html.escape('<>&"\'') == '&lt;&gt;&amp;&quot;&#x27;'
    print("escape_basic ok")

def test_escape_quote_false():
    assert html.escape('<b>', quote=False) == '&lt;b&gt;'
    assert html.escape('"hello"', quote=False) == '"hello"'
    print("escape_quote_false ok")

def test_unescape_basic():
    assert html.unescape('&lt;&gt;&amp;&quot;') == '<>&"'
    assert html.unescape('&nbsp;') == '\xa0'
    assert html.unescape('&copy;') == '©'
    print("unescape_basic ok")

def test_unescape_numeric():
    assert html.unescape('&#60;') == '<'
    assert html.unescape('&#x3C;') == '<'
    assert html.unescape('&#X3c;') == '<'
    print("unescape_numeric ok")

def test_unescape_html4():
    # HTML4 entities
    assert html.unescape('&eacute;') == 'é'
    assert html.unescape('&alpha;') == 'α'
    assert html.unescape('&euro;') == '€'
    assert html.unescape('&trade;') == '™'
    print("unescape_html4 ok")

def test_entities_name2codepoint():
    assert html.entities.name2codepoint['amp'] == 38
    assert html.entities.name2codepoint['lt'] == 60
    assert html.entities.name2codepoint['gt'] == 62
    assert html.entities.name2codepoint['eacute'] == 233
    assert html.entities.name2codepoint['euro'] == 8364
    assert len(html.entities.name2codepoint) >= 200
    print("entities_name2codepoint ok")

def test_entities_codepoint2name():
    assert html.entities.codepoint2name[38] == 'amp'
    assert html.entities.codepoint2name[60] == 'lt'
    assert html.entities.codepoint2name[233] == 'eacute'
    print("entities_codepoint2name ok")

def test_entities_html5():
    assert html.entities.html5['amp;'] == '&'
    assert html.entities.html5['lt;'] == '<'
    assert html.entities.html5['gt;'] == '>'
    assert html.entities.html5['nbsp;'] == '\xa0'
    assert len(html.entities.html5) >= 200
    print("entities_html5 ok")

# Use module-level lists for parser tests (goipy closure limitation)
_parser_basic_results = []

class _ParserBasic(HTMLParser):
    def handle_starttag(self, tag, attrs):
        _parser_basic_results.append(('start', tag, attrs))
    def handle_endtag(self, tag):
        _parser_basic_results.append(('end', tag))
    def handle_data(self, data):
        _parser_basic_results.append(('data', data))

def test_parser_basic():
    del _parser_basic_results[:]
    p = _ParserBasic()
    p.feed('<p class="x">Hello</p>')
    assert _parser_basic_results[0] == ('start', 'p', [('class', 'x')]), f"got {_parser_basic_results[0]}"
    assert _parser_basic_results[1] == ('data', 'Hello'), f"got {_parser_basic_results[1]}"
    assert _parser_basic_results[2] == ('end', 'p'), f"got {_parser_basic_results[2]}"
    print("parser_basic ok")

_parser_comment_results = []

class _ParserComment(HTMLParser):
    def handle_comment(self, data):
        _parser_comment_results.append(data)

def test_parser_comment():
    del _parser_comment_results[:]
    p = _ParserComment()
    p.feed('<!-- hello -->')
    assert _parser_comment_results == [' hello '], f"got {_parser_comment_results}"
    print("parser_comment ok")

_parser_doctype_results = []

class _ParserDoctype(HTMLParser):
    def handle_decl(self, decl):
        _parser_doctype_results.append(decl)

def test_parser_doctype():
    del _parser_doctype_results[:]
    p = _ParserDoctype()
    p.feed('<!DOCTYPE html>')
    assert len(_parser_doctype_results) == 1 and 'DOCTYPE' in _parser_doctype_results[0], f"got {_parser_doctype_results}"
    print("parser_doctype ok")

_parser_selfclose_results = []

class _ParserSelfclose(HTMLParser):
    def handle_startendtag(self, tag, attrs):
        _parser_selfclose_results.append(('startend', tag))
    def handle_starttag(self, tag, attrs):
        _parser_selfclose_results.append(('start', tag))
    def handle_endtag(self, tag):
        _parser_selfclose_results.append(('end', tag))

def test_parser_selfclose():
    del _parser_selfclose_results[:]
    p = _ParserSelfclose()
    p.feed('<br/>')
    assert len(_parser_selfclose_results) >= 1
    found = any(r[0] in ('startend', 'start') and r[1] == 'br' for r in _parser_selfclose_results)
    assert found, f"got {_parser_selfclose_results}"
    print("parser_selfclose ok")

test_escape_basic()
test_escape_quote_false()
test_unescape_basic()
test_unescape_numeric()
test_unescape_html4()
test_entities_name2codepoint()
test_entities_codepoint2name()
test_entities_html5()
test_parser_basic()
test_parser_comment()
test_parser_doctype()
test_parser_selfclose()
print("ALL OK")

from html.parser import HTMLParser

# Module-level event buffers to work around goipy closure limitation.
_events = []

class _CollectParser(HTMLParser):
    def handle_starttag(self, tag, attrs):
        _events.append(('start', tag, list(attrs)))
    def handle_endtag(self, tag):
        _events.append(('end', tag))
    def handle_startendtag(self, tag, attrs):
        _events.append(('startend', tag, list(attrs)))
    def handle_data(self, data):
        _events.append(('data', data))
    def handle_comment(self, data):
        _events.append(('comment', data))
    def handle_decl(self, decl):
        _events.append(('decl', decl))
    def handle_pi(self, data):
        _events.append(('pi', data))
    def unknown_decl(self, data):
        _events.append(('unknown_decl', data))
    def handle_entityref(self, name):
        _events.append(('entityref', name))
    def handle_charref(self, name):
        _events.append(('charref', name))

def collect(html, **kw):
    del _events[:]
    p = _CollectParser(**kw)
    p.feed(html)
    p.close()
    return list(_events)

def test_basic_tags():
    ev = collect('<p>Hello</p>')
    assert ev[0] == ('start', 'p', []), f"got {ev}"
    assert ev[1] == ('data', 'Hello'), f"got {ev}"
    assert ev[2] == ('end', 'p'), f"got {ev}"
    print("basic_tags ok")

def test_attrs():
    ev = collect('<a href="http://x.com" class=\'y\'>text</a>')
    assert ev[0][0] == 'start'
    assert ev[0][1] == 'a'
    attrs = dict(ev[0][2])
    assert attrs['href'] == 'http://x.com', f"attrs={attrs}"
    assert attrs['class'] == 'y', f"attrs={attrs}"
    print("attrs ok")

def test_boolean_attr():
    ev = collect('<input disabled checked>')
    assert ev[0][0] == 'start' and ev[0][1] == 'input'
    attrs = dict(ev[0][2])
    assert 'disabled' in attrs
    assert 'checked' in attrs
    print("boolean_attr ok")

def test_self_closing():
    ev = collect('<br/>')
    tags = [e[1] for e in ev if e[0] in ('start', 'startend')]
    assert 'br' in tags, f"got {ev}"
    print("self_closing ok")

def test_heading_tags():
    ev = collect('<h1>Title</h1><h2>Sub</h2>')
    tags = [e[1] for e in ev if e[0] == 'start']
    assert tags == ['h1', 'h2'], f"got {tags}"
    print("heading_tags ok")

def test_comment():
    ev = collect('<!-- a comment -->')
    assert any(e == ('comment', ' a comment ') for e in ev), f"got {ev}"
    print("comment ok")

def test_doctype():
    ev = collect('<!DOCTYPE html>')
    assert any(e[0] == 'decl' and 'DOCTYPE' in e[1].upper() for e in ev), f"got {ev}"
    print("doctype ok")

def test_pi():
    ev = collect("<?xml version='1.0'?>")
    assert any(e[0] == 'pi' and 'xml' in e[1] for e in ev), f"got {ev}"
    print("pi ok")

def test_convert_charrefs_true():
    ev = collect('<p>&lt;b&gt; &#60; &#x3C;</p>')
    data = [e[1] for e in ev if e[0] == 'data']
    combined = ''.join(data)
    assert '<' in combined, f"got {combined!r}"
    print("convert_charrefs_true ok")

def test_convert_charrefs_false():
    ev = collect('&amp; &#60; &#x3C;', convert_charrefs=False)
    kinds = [e[0] for e in ev]
    assert 'entityref' in kinds or 'charref' in kinds, f"got {ev}"
    erefs = [e[1] for e in ev if e[0] == 'entityref']
    crefs = [e[1] for e in ev if e[0] == 'charref']
    assert 'amp' in erefs, f"entityrefs={erefs}"
    assert '60' in crefs or 'x3C' in crefs or 'x3c' in crefs, f"charrefs={crefs}"
    print("convert_charrefs_false ok")

# Incremental test uses module-level list and a top-level class.
_incr_events = []

class _IncrParser(HTMLParser):
    def handle_starttag(self, tag, attrs):
        _incr_events.append(('start', tag))
    def handle_data(self, data):
        _incr_events.append(('data', data))
    def handle_endtag(self, tag):
        _incr_events.append(('end', tag))

def test_incremental():
    del _incr_events[:]
    p = _IncrParser()
    p.feed('<p>hel')
    p.feed('lo</p>')
    starts = [e for e in _incr_events if e[0] == 'start']
    ends = [e for e in _incr_events if e[0] == 'end']
    assert len(starts) == 1 and starts[0][1] == 'p', f"got {_incr_events}"
    assert len(ends) == 1 and ends[0][1] == 'p', f"got {_incr_events}"
    print("incremental ok")

# getpos test uses module-level list and top-level class.
_getpos_results = []

class _GetposParser(HTMLParser):
    def handle_starttag(self, tag, attrs):
        _getpos_results.append(self.getpos())

def test_getpos():
    del _getpos_results[:]
    p = _GetposParser()
    p.feed('<p>\n<span>')
    assert len(_getpos_results) >= 1
    line, col = _getpos_results[0]
    assert isinstance(line, int) and line >= 1
    assert isinstance(col, int) and col >= 0
    print("getpos ok")

def test_cdata():
    ev = collect('<![CDATA[some data]]>')
    assert any(e[0] == 'unknown_decl' for e in ev), f"got {ev}"
    print("cdata ok")

def test_nested():
    ev = collect('<div id="main"><p class="c">text</p></div>')
    starts = [(e[1], dict(e[2])) for e in ev if e[0] == 'start']
    assert starts[0] == ('div', {'id': 'main'}), f"got {starts}"
    assert starts[1] == ('p', {'class': 'c'}), f"got {starts}"
    print("nested ok")

test_basic_tags()
test_attrs()
test_boolean_attr()
test_self_closing()
test_heading_tags()
test_comment()
test_doctype()
test_pi()
test_convert_charrefs_true()
test_convert_charrefs_false()
test_incremental()
test_getpos()
test_cdata()
test_nested()
print("ALL OK")

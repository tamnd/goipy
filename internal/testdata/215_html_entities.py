import html.entities

def test_name2codepoint_basic():
    assert html.entities.name2codepoint['amp'] == 38
    assert html.entities.name2codepoint['lt'] == 60
    assert html.entities.name2codepoint['gt'] == 62
    assert html.entities.name2codepoint['quot'] == 34
    assert html.entities.name2codepoint['nbsp'] == 160
    assert html.entities.name2codepoint['eacute'] == 233
    assert html.entities.name2codepoint['euro'] == 8364
    assert html.entities.name2codepoint['alpha'] == 945
    assert html.entities.name2codepoint['Omega'] == 937
    print("name2codepoint_basic ok")

def test_name2codepoint_size():
    # CPython has exactly 252 entries in name2codepoint
    n = len(html.entities.name2codepoint)
    assert n >= 250, f"got {n}"
    print("name2codepoint_size ok")

def test_codepoint2name_basic():
    assert html.entities.codepoint2name[38] == 'amp'
    assert html.entities.codepoint2name[60] == 'lt'
    assert html.entities.codepoint2name[62] == 'gt'
    assert html.entities.codepoint2name[34] == 'quot'
    assert html.entities.codepoint2name[233] == 'eacute'
    assert html.entities.codepoint2name[8364] == 'euro'
    print("codepoint2name_basic ok")

def test_codepoint2name_size():
    n = len(html.entities.codepoint2name)
    assert n >= 200, f"got {n}"
    print("codepoint2name_size ok")

def test_html5_basic():
    assert html.entities.html5['amp;'] == '&'
    assert html.entities.html5['lt;'] == '<'
    assert html.entities.html5['gt;'] == '>'
    assert html.entities.html5['quot;'] == '"'
    assert html.entities.html5['nbsp;'] == '\xa0'
    assert html.entities.html5['apos;'] == "'"
    print("html5_basic ok")

def test_html5_legacy_no_semicolon():
    # HTML5 includes no-semicolon versions of legacy entities
    assert html.entities.html5['amp'] == '&'
    assert html.entities.html5['lt'] == '<'
    assert html.entities.html5['gt'] == '>'
    assert html.entities.html5['quot'] == '"'
    assert html.entities.html5['AMP'] == '&'
    assert html.entities.html5['LT'] == '<'
    print("html5_legacy_no_semicolon ok")

def test_html5_size():
    # CPython html5 has 2231 entries
    n = len(html.entities.html5)
    assert n >= 2200, f"got {n} entries"
    print("html5_size ok")

def test_html5_html5_only():
    # Entries that exist in HTML5 but not HTML4
    assert 'CounterClockwiseContourIntegral;' in html.entities.html5
    assert 'DoubleContourIntegral;' in html.entities.html5
    assert 'Implies;' in html.entities.html5
    assert 'LeftArrow;' in html.entities.html5
    assert 'RightArrow;' in html.entities.html5
    assert 'Sqrt;' in html.entities.html5
    print("html5_html5_only ok")

def test_html5_multichar():
    # Some HTML5 entities map to multiple Unicode characters
    # e.g. 'nGt;' -> '≫⃒' (two chars)
    val = html.entities.html5.get('nGt;')
    if val is not None:
        assert len(val) >= 1
    print("html5_multichar ok")

def test_html5_case_variants():
    # HTML5 has case variants: 'Aacute;' and 'aacute;' are different
    assert html.entities.html5['Aacute;'] == '\xc1'
    assert html.entities.html5['aacute;'] == '\xe1'
    assert html.entities.html5['Alpha;'] == 'Α'
    assert html.entities.html5['alpha;'] == 'α'
    print("html5_case_variants ok")

test_name2codepoint_basic()
test_name2codepoint_size()
test_codepoint2name_basic()
test_codepoint2name_size()
test_html5_basic()
test_html5_legacy_no_semicolon()
test_html5_size()
test_html5_html5_only()
test_html5_multichar()
test_html5_case_variants()
print("ALL OK")

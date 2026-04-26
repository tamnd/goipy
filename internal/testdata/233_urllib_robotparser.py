import urllib.robotparser
import time


def test_constructor_no_url():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse([
        "User-agent: *",
        "Disallow: /private/",
        "Allow: /public/",
    ])
    assert rfp.can_fetch("*", "/public/page") == True
    assert rfp.can_fetch("*", "/private/secret") == False
    print("constructor_no_url ok")


def test_constructor_with_url():
    rfp = urllib.robotparser.RobotFileParser("http://example.com/robots.txt")
    # Just verify construction doesn't raise; URL is stored but no fetch occurs.
    rfp.parse(["User-agent: *", "Disallow: /"])
    assert rfp.can_fetch("*", "/anything") == False
    print("constructor_with_url ok")


def test_set_url():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.set_url("http://example.com/robots.txt")
    rfp.parse(["User-agent: *", "Allow: /"])
    assert rfp.can_fetch("*", "/page") == True
    print("set_url ok")


def test_parse_disallow():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse([
        "User-agent: *",
        "Disallow: /blocked/",
    ])
    assert rfp.can_fetch("*", "/blocked/page") == False
    assert rfp.can_fetch("*", "/allowed/page") == True
    assert rfp.can_fetch("*", "/blocked") == True   # not prefix of /blocked/
    print("parse_disallow ok")


def test_parse_allow():
    # Allow must come before Disallow to win (first-match wins, CPython 3.14).
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse([
        "User-agent: *",
        "Allow: /api/public/",
        "Disallow: /api/",
    ])
    assert rfp.can_fetch("*", "/api/private/") == False
    assert rfp.can_fetch("*", "/api/public/data") == True
    print("parse_allow ok")


def test_parse_disallow_all():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse(["User-agent: *", "Disallow: /"])
    assert rfp.can_fetch("*", "/") == False
    assert rfp.can_fetch("*", "/anything") == False
    assert rfp.can_fetch("*", "/a/b/c") == False
    print("parse_disallow_all ok")


def test_parse_empty_disallow():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse(["User-agent: *", "Disallow: "])
    assert rfp.can_fetch("*", "/private/") == True
    assert rfp.can_fetch("*", "/anything") == True
    print("parse_empty_disallow ok")


def test_parse_multiple_agents_block():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse([
        "User-agent: bot1",
        "User-agent: bot2",
        "Disallow: /secret/",
        "",
        "User-agent: *",
        "Allow: /",
    ])
    assert rfp.can_fetch("bot1", "/secret/page") == False
    assert rfp.can_fetch("bot2", "/secret/page") == False
    assert rfp.can_fetch("bot3", "/secret/page") == True
    print("parse_multiple_agents_block ok")


def test_agent_specific_overrides_wildcard():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse([
        "User-agent: *",
        "Disallow: /",
        "",
        "User-agent: googlebot",
        "Allow: /",
    ])
    assert rfp.can_fetch("googlebot", "/page") == True
    assert rfp.can_fetch("otherbot", "/page") == False
    print("agent_specific_overrides_wildcard ok")


def test_agent_case_insensitive():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse([
        "User-agent: Googlebot",
        "Disallow: /private/",
    ])
    assert rfp.can_fetch("Googlebot", "/private/") == False
    assert rfp.can_fetch("googlebot", "/private/") == False
    assert rfp.can_fetch("GOOGLEBOT", "/private/") == False
    print("agent_case_insensitive ok")


def test_crawl_delay_int():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse([
        "User-agent: mybot",
        "Crawl-delay: 10",
        "Disallow: /",
    ])
    cd = rfp.crawl_delay("mybot")
    assert cd == 10, repr(cd)
    assert isinstance(cd, int), type(cd)
    print("crawl_delay_int ok")


def test_crawl_delay_none_missing():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse(["User-agent: *", "Disallow: /"])
    assert rfp.crawl_delay("*") is None
    assert rfp.crawl_delay("unknownbot") is None
    print("crawl_delay_none_missing ok")


def test_crawl_delay_none_invalid_float():
    # CPython only accepts integer crawl-delay values; floats like 1.5 → None
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse(["User-agent: *", "Crawl-delay: 1.5"])
    assert rfp.crawl_delay("*") is None
    print("crawl_delay_none_invalid_float ok")


def test_request_rate():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse([
        "User-agent: testbot",
        "Request-rate: 3/10",
    ])
    rr = rfp.request_rate("testbot")
    assert rr is not None, repr(rr)
    assert rr.requests == 3, repr(rr.requests)
    assert rr.seconds == 10, repr(rr.seconds)
    print("request_rate ok")


def test_request_rate_attrs():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse(["User-agent: *", "Request-rate: 1/5"])
    rr = rfp.request_rate("*")
    assert isinstance(rr.requests, int), type(rr.requests)
    assert isinstance(rr.seconds, int), type(rr.seconds)
    assert rr.requests == 1
    assert rr.seconds == 5
    r = repr(rr)
    assert "RequestRate" in r, repr(r)
    assert "requests=1" in r, repr(r)
    assert "seconds=5" in r, repr(r)
    print("request_rate_attrs ok")


def test_request_rate_none():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse(["User-agent: *", "Disallow: /"])
    assert rfp.request_rate("*") is None
    assert rfp.request_rate("unknownbot") is None
    print("request_rate_none ok")


def test_site_maps_list():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse([
        "User-agent: *",
        "Disallow: /",
        "Sitemap: http://example.com/sitemap.xml",
        "Sitemap: http://example.com/sitemap2.xml",
    ])
    sm = rfp.site_maps()
    assert sm is not None
    assert isinstance(sm, list)
    assert "http://example.com/sitemap.xml" in sm
    assert "http://example.com/sitemap2.xml" in sm
    assert len(sm) == 2
    print("site_maps_list ok")


def test_site_maps_none():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse(["User-agent: *", "Disallow: /"])
    assert rfp.site_maps() is None
    print("site_maps_none ok")


def test_mtime_initial():
    rfp = urllib.robotparser.RobotFileParser()
    # mtime() is 0 before any parse() or modified() call
    assert rfp.mtime() == 0, repr(rfp.mtime())
    print("mtime_initial ok")


def test_mtime_after_parse():
    before = time.time()
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse(["User-agent: *", "Disallow: /"])
    mt = rfp.mtime()
    after = time.time()
    assert mt > 0, repr(mt)
    assert before <= mt <= after + 1, repr(mt)
    print("mtime_after_parse ok")


def test_modified():
    rfp = urllib.robotparser.RobotFileParser()
    before = time.time()
    rfp.modified()
    mt = rfp.mtime()
    after = time.time()
    assert before <= mt <= after + 1, repr(mt)
    print("modified ok")


def test_can_fetch_full_url():
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse([
        "User-agent: *",
        "Disallow: /private/",
        "Allow: /public/",
    ])
    # Full URL — robotparser extracts the path
    assert rfp.can_fetch("*", "http://example.com/private/page") == False
    assert rfp.can_fetch("*", "http://example.com/public/page") == True
    assert rfp.can_fetch("*", "https://example.com/private/data") == False
    print("can_fetch_full_url ok")


def test_first_match_wins():
    # CPython 3.14 uses first-match-wins: whichever rule matches first in the
    # robots.txt file wins, regardless of specificity.
    rfp = urllib.robotparser.RobotFileParser()
    rfp.parse([
        "User-agent: *",
        "Allow: /dir/public/",   # checked first
        "Disallow: /dir/",       # only reached if Allow didn't match
    ])
    assert rfp.can_fetch("*", "/dir/public/file") == True
    assert rfp.can_fetch("*", "/dir/secret/") == False
    print("first_match_wins ok")


def test_exports():
    assert urllib.robotparser.RobotFileParser is not None
    rfp = urllib.robotparser.RobotFileParser()
    assert isinstance(rfp, urllib.robotparser.RobotFileParser)
    print("exports ok")


test_constructor_no_url()
test_constructor_with_url()
test_set_url()
test_parse_disallow()
test_parse_allow()
test_parse_disallow_all()
test_parse_empty_disallow()
test_parse_multiple_agents_block()
test_agent_specific_overrides_wildcard()
test_agent_case_insensitive()
test_crawl_delay_int()
test_crawl_delay_none_missing()
test_crawl_delay_none_invalid_float()
test_request_rate()
test_request_rate_attrs()
test_request_rate_none()
test_site_maps_list()
test_site_maps_none()
test_mtime_initial()
test_mtime_after_parse()
test_modified()
test_can_fetch_full_url()
test_first_match_wins()
test_exports()

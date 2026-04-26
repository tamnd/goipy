import http.cookiejar


def test_load_error():
    print(issubclass(http.cookiejar.LoadError, OSError))
    try:
        raise http.cookiejar.LoadError("bad jar")
    except OSError as e:
        print(str(e))
    print('load_error ok')


def test_cookie_jar_empty():
    jar = http.cookiejar.CookieJar()
    print(type(jar).__name__)
    print(len(jar))
    print('cookie_jar_empty ok')


def _make_cookie(name, value, domain='example.com', path='/',
                 expires=None, discard=True, secure=False, rest=None):
    return http.cookiejar.Cookie(
        version=0, name=name, value=value,
        port=None, port_specified=False,
        domain=domain, domain_specified=True, domain_initial_dot=False,
        path=path, path_specified=True,
        secure=secure, expires=expires, discard=discard,
        comment=None, comment_url=None, rest=rest or {},
    )


def test_cookie_jar_set_cookie():
    jar = http.cookiejar.CookieJar()
    jar.set_cookie(_make_cookie('foo', 'bar'))
    print(len(jar))
    print('cookie_jar_set_cookie ok')


def test_cookie_jar_iter():
    jar = http.cookiejar.CookieJar()
    jar.set_cookie(_make_cookie('a', '1'))
    jar.set_cookie(_make_cookie('b', '2'))
    names = sorted(c.name for c in jar)
    print(names)
    print('cookie_jar_iter ok')


def test_cookie_jar_clear():
    jar = http.cookiejar.CookieJar()
    jar.set_cookie(_make_cookie('x', 'y'))
    jar.clear()
    print(len(jar))
    print('cookie_jar_clear ok')


def test_cookie_jar_clear_session():
    jar = http.cookiejar.CookieJar()
    jar.set_cookie(_make_cookie('sess', 'v1', discard=True, expires=None))
    jar.set_cookie(_make_cookie('perm', 'v2', discard=False, expires=9999999999))
    jar.clear_session_cookies()
    print(len(jar))
    names = [c.name for c in jar]
    print(names)
    print('cookie_jar_clear_session ok')


def test_cookie_attrs():
    c = _make_cookie('myname', 'myvalue', domain='example.com', path='/test',
                     expires=1000000, discard=False, secure=True,
                     rest={'HttpOnly': None})
    print(c.name)
    print(c.value)
    print(c.domain)
    print(c.path)
    print(c.secure)
    print(c.version)
    print(c.expires)
    print(c.discard)
    print(c.port is None)
    print('cookie_attrs ok')


def test_cookie_has_nonstandard_attr():
    c = _make_cookie('t', 'v', rest={'HttpOnly': None})
    print(c.has_nonstandard_attr('HttpOnly'))
    print(c.has_nonstandard_attr('Missing'))
    print('cookie_has_nonstandard_attr ok')


def test_cookie_str():
    c = _make_cookie('foo', 'bar', domain='example.com', path='/')
    s = str(c)
    print('<Cookie' in s)
    print('foo=bar' in s)
    print('example.com' in s)
    print('cookie_str ok')


def test_default_policy_class_attrs():
    dp = http.cookiejar.DefaultCookiePolicy
    print(dp.DomainStrictNoDots)
    print(dp.DomainStrictNonDomain)
    print(dp.DomainRFC2965Match)
    print(dp.DomainLiberal)
    print(dp.DomainStrict)
    print('default_policy_class_attrs ok')


def test_default_policy_init():
    dp = http.cookiejar.DefaultCookiePolicy()
    print(type(dp).__name__)
    print(dp.netscape)
    print(dp.rfc2965)
    print(dp.strict_domain)
    print(dp.strict_rfc2965_unverifiable)
    print(dp.strict_ns_unverifiable)
    print(dp.strict_ns_domain)
    print(dp.hide_cookie2)
    print('default_policy_init ok')


def test_cookie_policy_methods():
    cp = http.cookiejar.CookiePolicy
    for name in ['return_ok', 'domain_return_ok', 'path_return_ok', 'set_ok']:
        print(name, hasattr(cp, name))
    print('cookie_policy_methods ok')


def test_hierarchy_jars():
    cj = http.cookiejar
    print(issubclass(cj.FileCookieJar, cj.CookieJar))
    print(issubclass(cj.MozillaCookieJar, cj.FileCookieJar))
    print(issubclass(cj.LWPCookieJar, cj.FileCookieJar))
    print('hierarchy_jars ok')


def test_hierarchy_policy():
    cj = http.cookiejar
    print(issubclass(cj.DefaultCookiePolicy, cj.CookiePolicy))
    print('hierarchy_policy ok')


def test_jar_set_policy():
    jar = http.cookiejar.CookieJar()
    policy = http.cookiejar.DefaultCookiePolicy()
    jar.set_policy(policy)
    print(type(jar).__name__)
    print('jar_set_policy ok')


def test_module_exports():
    cj = http.cookiejar
    for name in ['CookieJar', 'FileCookieJar', 'MozillaCookieJar', 'LWPCookieJar',
                 'Cookie', 'CookiePolicy', 'DefaultCookiePolicy', 'LoadError']:
        print(name, name in dir(cj))
    print('module_exports ok')


test_load_error()
test_cookie_jar_empty()
test_cookie_jar_set_cookie()
test_cookie_jar_iter()
test_cookie_jar_clear()
test_cookie_jar_clear_session()
test_cookie_attrs()
test_cookie_has_nonstandard_attr()
test_cookie_str()
test_default_policy_class_attrs()
test_default_policy_init()
test_cookie_policy_methods()
test_hierarchy_jars()
test_hierarchy_policy()
test_jar_set_policy()
test_module_exports()

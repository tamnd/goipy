import http.cookies


def test_cookie_error():
    try:
        raise http.cookies.CookieError("bad cookie")
    except http.cookies.CookieError as e:
        print(str(e))
    print('cookie_error ok')


def test_morsel_reserved():
    m = http.cookies.Morsel()
    r = m._reserved
    print(len(r) == 10)
    print('path' in r)
    print('domain' in r)
    print('httponly' in r)
    print('morsel_reserved ok')


def test_morsel_flags():
    m = http.cookies.Morsel()
    f = m._flags
    print(len(f) == 3)
    print('httponly' in f)
    print('secure' in f)
    print('morsel_flags ok')


def test_morsel_initial():
    m = http.cookies.Morsel()
    print(m.key is None)
    print(m.value is None)
    print(m.coded_value is None)
    print('morsel_initial ok')


def test_morsel_set():
    m = http.cookies.Morsel()
    m.set('foo', 'bar', 'bar')
    print(m.key)
    print(m.value)
    print(m.coded_value)
    print('morsel_set ok')


def test_morsel_setitem_valid():
    m = http.cookies.Morsel()
    m.set('foo', 'bar', 'bar')
    m['path'] = '/test'
    print(m['path'])
    print('morsel_setitem_valid ok')


def test_morsel_setitem_invalid():
    m = http.cookies.Morsel()
    try:
        m['badkey'] = 'value'
        print(False)
    except http.cookies.CookieError:
        print(True)
    print('morsel_setitem_invalid ok')


def test_morsel_isreservedkey():
    m = http.cookies.Morsel()
    print(m.isReservedKey('path'))
    print(m.isReservedKey('badkey'))
    print('morsel_isreservedkey ok')


def test_morsel_output_string():
    m = http.cookies.Morsel()
    m.set('foo', 'bar', 'bar')
    m['path'] = '/test'
    m['httponly'] = True
    s = m.OutputString()
    print('foo=bar' in s)
    print('Path=/test' in s)
    print('HttpOnly' in s)
    print('morsel_output_string ok')


def test_morsel_output():
    m = http.cookies.Morsel()
    m.set('foo', 'bar', 'bar')
    s = m.output()
    print(s.startswith('Set-Cookie:'))
    print('foo=bar' in s)
    print('morsel_output ok')


def test_morsel_repr():
    m = http.cookies.Morsel()
    m.set('foo', 'bar', 'bar')
    r = repr(m)
    print('<Morsel: foo=bar>' == r)
    print('morsel_repr ok')


def test_basecookie_setitem():
    bc = http.cookies.BaseCookie()
    bc['name'] = 'Alice'
    print(type(bc['name']).__name__)
    print(bc['name'].value)
    print('basecookie_setitem ok')


def test_basecookie_output():
    bc = http.cookies.BaseCookie()
    bc['name'] = 'Alice'
    bc['name']['path'] = '/'
    s = bc.output()
    print('Set-Cookie:' in s)
    print('name=Alice' in s)
    print('Path=/' in s)
    print('basecookie_output ok')


def test_basecookie_load():
    bc = http.cookies.BaseCookie()
    bc.load('name=Alice; age=30')
    print(bc['name'].value)
    print(bc['age'].value)
    print('basecookie_load ok')


def test_simplecookie_load():
    sc = http.cookies.SimpleCookie()
    sc.load('greeting="hello world"')
    print(sc['greeting'].value)
    print('simplecookie_load ok')


def test_simplecookie_set():
    sc = http.cookies.SimpleCookie()
    sc['k'] = 'v'
    sc['k']['path'] = '/'
    s = sc.output()
    print('k=v' in s)
    print('Path=/' in s)
    print('simplecookie_set ok')


def test_hierarchy():
    print(issubclass(http.cookies.SimpleCookie, http.cookies.BaseCookie))
    print(issubclass(http.cookies.BaseCookie, http.cookies.BaseCookie))
    print('hierarchy ok')


def test_module_exports():
    for name in ['BaseCookie', 'CookieError', 'Morsel', 'SimpleCookie']:
        print(name, name in dir(http.cookies))
    print('module_exports ok')


test_cookie_error()
test_morsel_reserved()
test_morsel_flags()
test_morsel_initial()
test_morsel_set()
test_morsel_setitem_valid()
test_morsel_setitem_invalid()
test_morsel_isreservedkey()
test_morsel_output_string()
test_morsel_output()
test_morsel_repr()
test_basecookie_setitem()
test_basecookie_output()
test_basecookie_load()
test_simplecookie_load()
test_simplecookie_set()
test_hierarchy()
test_module_exports()

import xmlrpc.client as xc


def test_escape():
    print(xc.escape('hello & <world>'))
    print(xc.escape('no special chars'))
    print(xc.escape('a & b < c > d'))
    print('escape ok')


def test_wrappers():
    print(xc.WRAPPERS[0].__name__)
    print(xc.WRAPPERS[1].__name__)
    print(len(xc.WRAPPERS))
    print('wrappers ok')


def test_gzip_roundtrip():
    data = b'hello world from goipy'
    enc = xc.gzip_encode(data)
    print(type(enc).__name__)
    dec = xc.gzip_decode(enc)
    print(dec)
    print(dec == data)
    print('gzip ok')


def test_marshaller_dumps():
    m = xc.Marshaller()
    out = m.dumps((42, 'hello'))
    print(isinstance(out, str))
    print('<params>' in out)
    print('<int>42</int>' in out)
    print('<string>hello</string>' in out)
    print('marshaller ok')


def test_marshaller_bool():
    m = xc.Marshaller()
    out = m.dumps((True, False))
    print('<boolean>1</boolean>' in out)
    print('<boolean>0</boolean>' in out)
    print('marshaller_bool ok')


def test_getparser():
    p, u = xc.getparser()
    print(type(p).__name__)
    print(type(u).__name__)
    xml = ("<?xml version='1.0'?>"
           "<methodCall><methodName>add</methodName>"
           "<params><param><value><int>99</int></value></param></params>"
           "</methodCall>")
    p.feed(xml)
    p.close()
    result = u.close()
    print(result[0])
    print(u.getmethodname())
    print('getparser ok')


def test_unmarshaller_fresh_close():
    u = xc.Unmarshaller()
    try:
        u.close()
        print('no error')
    except xc.ResponseError:
        print('ResponseError raised')
    print('unmarshaller_fresh ok')


def test_multicall_iterator():
    mci = xc.MultiCallIterator([[1, 2], [42, 0]])
    results = list(mci)
    print(results[0])
    print(results[1])
    print('multicall_iter ok')


def test_multicall_iterator_fault():
    mci = xc.MultiCallIterator([{'faultCode': 5, 'faultString': 'err msg'}])
    try:
        for _ in mci:
            pass
        print('no fault')
    except xc.Fault as e:
        print(e.faultCode)
        print(e.faultString)
    print('multicall_fault ok')


def test_fast_variants():
    print(xc.FastParser is None)
    print(xc.FastMarshaller is None)
    print(xc.FastUnmarshaller is None)
    print('fast_variants ok')


def test_datetime_cmp():
    dt1 = xc.DateTime('20240101T12:00:00')
    dt2 = xc.DateTime('20240101T12:00:00')
    dt3 = xc.DateTime('20240102T12:00:00')
    print(dt1 == dt2)
    print(dt1 < dt3)
    print(dt1 > dt3)
    print(dt3 > dt1)
    print('datetime_cmp ok')


def test_binary_eq():
    b1 = xc.Binary(b'test data')
    b2 = xc.Binary(b'test data')
    b3 = xc.Binary(b'other')
    print(b1 == b2)
    print(b1 == b3)
    print('binary_eq ok')


test_escape()
test_wrappers()
test_gzip_roundtrip()
test_marshaller_dumps()
test_marshaller_bool()
test_getparser()
test_unmarshaller_fresh_close()
test_multicall_iterator()
test_multicall_iterator_fault()
test_fast_variants()
test_datetime_cmp()
test_binary_eq()

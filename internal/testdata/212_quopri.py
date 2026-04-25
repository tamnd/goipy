import quopri

def test_encodestring_basic():
    # plain ASCII stays the same
    enc = quopri.encodestring(b"Hello, World!\n")
    assert enc == b"Hello, World!\n", f"got {enc!r}"
    print("encodestring_basic ok")

def test_encodestring_special():
    # bytes > 0x7E encoded as =XX
    enc = quopri.encodestring(b"\xff\xfe")
    assert enc == b"=FF=FE", f"got {enc!r}"
    # = encoded as =3D
    enc2 = quopri.encodestring(b"x=y")
    assert enc2 == b"x=3Dy", f"got {enc2!r}"
    print("encodestring_special ok")

def test_encodestring_quotetabs():
    # without quotetabs: spaces stay (unless trailing)
    enc = quopri.encodestring(b"hello world")
    assert b" " in enc, f"got {enc!r}"
    # with quotetabs: spaces encoded
    enc2 = quopri.encodestring(b"hello world", quotetabs=True)
    assert b" " not in enc2, f"got {enc2!r}"
    assert b"=20" in enc2, f"got {enc2!r}"
    print("encodestring_quotetabs ok")

def test_encodestring_header():
    # header=True: spaces become underscores
    enc = quopri.encodestring(b"hello world", header=True)
    assert b"_" in enc, f"got {enc!r}"
    assert b" " not in enc, f"got {enc!r}"
    print("encodestring_header ok")

def test_decodestring_basic():
    assert quopri.decodestring(b"Hello, World!\n") == b"Hello, World!\n"
    assert quopri.decodestring(b"=FF=FE") == b"\xff\xfe"
    assert quopri.decodestring(b"x=3Dy") == b"x=y"
    print("decodestring_basic ok")

def test_decodestring_softbreak():
    # soft line break =\n should be removed
    enc = b"hello=\nworld"
    assert quopri.decodestring(enc) == b"helloworld", f"got {quopri.decodestring(enc)!r}"
    print("decodestring_softbreak ok")

def test_decodestring_header():
    # header=True: _ decoded as space
    assert quopri.decodestring(b"hello_world", header=True) == b"hello world"
    print("decodestring_header ok")

def test_roundtrip():
    data = b"Subject: caf\xe9 au lait\nFrom: user@example.com\n\nBody with = sign and\ttabs.\n"
    enc = quopri.encodestring(data)
    dec = quopri.decodestring(enc)
    assert dec == data, f"enc={enc!r}\ndec={dec!r}"
    print("roundtrip ok")

test_encodestring_basic()
test_encodestring_special()
test_encodestring_quotetabs()
test_encodestring_header()
test_decodestring_basic()
test_decodestring_softbreak()
test_decodestring_header()
test_roundtrip()
print("ALL OK")

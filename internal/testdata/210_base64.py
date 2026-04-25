import base64

def test_b64encode_basic():
    assert base64.b64encode(b"Hello") == b"SGVsbG8="
    assert base64.b64encode(b"") == b""
    print("b64encode_basic ok")

def test_b64encode_altchars():
    encoded = base64.b64encode(b"\xfb\xff", altchars=b"-_")
    assert encoded == b"-_8=", f"got {encoded!r}"
    print("b64encode_altchars ok")

def test_b64decode_basic():
    assert base64.b64decode(b"SGVsbG8=") == b"Hello"
    assert base64.b64decode("SGVsbG8=") == b"Hello"
    # whitespace stripped by default
    assert base64.b64decode(b"SGVs\nbG8=") == b"Hello"
    print("b64decode_basic ok")

def test_b64decode_validate():
    try:
        base64.b64decode(b"SGVs bG8=", validate=True)
        assert False, "should raise"
    except Exception:
        pass
    print("b64decode_validate ok")

def test_urlsafe():
    data = b"\xfb\xff\xfe"
    enc = base64.urlsafe_b64encode(data)
    assert b"+" not in enc and b"/" not in enc
    dec = base64.urlsafe_b64decode(enc)
    assert dec == data
    # missing padding
    dec2 = base64.urlsafe_b64decode(enc.rstrip(b"="))
    assert dec2 == data
    print("urlsafe ok")

def test_b32():
    assert base64.b32encode(b"Hello") == b"JBSWY3DP"
    assert base64.b32decode(b"JBSWY3DP") == b"Hello"
    assert base64.b32decode(b"jbswy3dp", casefold=True) == b"Hello"
    print("b32 ok")

def test_b32hex():
    enc = base64.b32hexencode(b"Hello")
    assert isinstance(enc, bytes)
    assert base64.b32hexdecode(enc) == b"Hello"
    assert base64.b32hexdecode(enc.lower(), casefold=True) == b"Hello"
    print("b32hex ok")

def test_b16():
    assert base64.b16encode(b"Hello") == b"48656C6C6F"
    assert base64.b16decode(b"48656C6C6F") == b"Hello"
    assert base64.b16decode(b"48656c6c6f", casefold=True) == b"Hello"
    print("b16 ok")

def test_standard_aliases():
    assert base64.standard_b64encode(b"test") == base64.b64encode(b"test")
    assert base64.standard_b64decode(b"dGVzdA==") == b"test"
    print("standard_aliases ok")

def test_encodebytes():
    data = b"A" * 60
    enc = base64.encodebytes(data)
    assert enc.endswith(b"\n")
    assert base64.decodebytes(enc) == data
    print("encodebytes ok")

def test_encodebytes_long():
    # 60 bytes -> 80 base64 chars -> split at 76 -> two lines
    data = b"x" * 57  # 57 bytes = 76 base64 chars exactly -> 1 line + newline
    enc = base64.encodebytes(data)
    lines = enc.split(b"\n")
    assert lines[-1] == b""  # trailing newline
    assert base64.decodebytes(enc) == data
    print("encodebytes_long ok")

def test_b85():
    assert base64.b85encode(b"Hello") == b"NM&qnZv"
    assert base64.b85decode(b"NM&qnZv") == b"Hello"
    rt = b"The quick brown fox"
    assert base64.b85decode(base64.b85encode(rt)) == rt
    print("b85 ok")

def test_a85():
    enc = base64.a85encode(b"Hello")
    assert isinstance(enc, bytes)
    assert base64.a85decode(enc) == b"Hello"
    # adobe wrapping
    enc2 = base64.a85encode(b"Hello", adobe=True)
    assert enc2.startswith(b"<~") and enc2.endswith(b"~>"), f"got {enc2!r}"
    assert base64.a85decode(enc2, adobe=True) == b"Hello"
    # round-trip
    data = b"The quick brown fox jumps"
    assert base64.a85decode(base64.a85encode(data)) == data
    print("a85 ok")

def test_b32_map01():
    # map01: map '0' to 'O', '1' to 'I' (or 'L')
    enc = base64.b32encode(b"\x00")
    # enc should be b"AA======"
    dec = base64.b32decode(b"AA======")
    assert dec == b"\x00"
    print("b32_map01 ok")

test_b64encode_basic()
test_b64encode_altchars()
test_b64decode_basic()
test_b64decode_validate()
test_urlsafe()
test_b32()
test_b32hex()
test_b16()
test_standard_aliases()
test_encodebytes()
test_encodebytes_long()
test_b85()
test_a85()
test_b32_map01()
print("ALL OK")

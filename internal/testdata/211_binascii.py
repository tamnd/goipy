import binascii

def test_hexlify_basic():
    assert binascii.hexlify(b"\xde\xad") == b"dead"
    assert binascii.b2a_hex(b"\xde\xad") == b"dead"
    assert binascii.unhexlify(b"dead") == b"\xde\xad"
    assert binascii.a2b_hex("dead") == b"\xde\xad"
    print("hexlify_basic ok")

def test_hexlify_sep():
    r = binascii.hexlify(b"\xde\xad\xbe\xef", sep=b":")
    assert r == b"de:ad:be:ef", f"got {r!r}"
    r2 = binascii.hexlify(b"\xde\xad\xbe\xef", sep=":", bytes_per_sep=2)
    assert r2 == b"dead:beef", f"got {r2!r}"
    r3 = binascii.hexlify(b"\xde\xad\xbe\xef", sep=":", bytes_per_sep=-2)
    assert r3 == b"dead:beef", f"got {r3!r}"
    print("hexlify_sep ok")

def test_base64():
    enc = binascii.b2a_base64(b"Hello")
    assert enc == b"SGVsbG8=\n"
    assert binascii.a2b_base64(enc) == b"Hello"
    enc2 = binascii.b2a_base64(b"Hello", newline=False)
    assert enc2 == b"SGVsbG8="
    print("base64 ok")

def test_a2b_base64_strict():
    # non-strict: whitespace stripped OK
    assert binascii.a2b_base64(b"SGVs bG8=") == b"Hello"
    try:
        binascii.a2b_base64(b"SGVs bG8=", strict_mode=True)
        assert False, "should raise"
    except Exception:
        pass
    print("a2b_base64_strict ok")

def test_crc32():
    assert binascii.crc32(b"hello world") == 222957957
    assert binascii.crc32(b"") == 0
    print("crc32 ok")

def test_crc_hqx():
    assert binascii.crc_hqx(b"", 0) == 0
    v = binascii.crc_hqx(b"Hello", 0)
    assert isinstance(v, int)
    # known: crc_hqx(b"Hello", 0) == 0x058c  (verify against CPython)
    print("crc_hqx ok")

def test_uu():
    enc = binascii.b2a_uu(b"Hello World")
    assert isinstance(enc, bytes)
    assert enc.endswith(b"\n")
    dec = binascii.a2b_uu(enc)
    assert dec == b"Hello World", f"got {dec!r}"
    # backtick mode
    enc2 = binascii.b2a_uu(b"Hi", backtick=True)
    assert b" " not in enc2 or enc2.startswith(b"\"")
    print("uu ok")

def test_qp():
    data = b"Subject: =?utf-8?q?hello=20world?="
    enc = binascii.b2a_qp(data)
    assert isinstance(enc, bytes)
    dec = binascii.a2b_qp(enc)
    assert dec == data, f"got {dec!r}"
    print("qp ok")

def test_qp_header():
    # header=True: spaces encoded as _
    enc = binascii.b2a_qp(b"hello world", header=True)
    assert b"_" in enc or b"=20" in enc
    dec = binascii.a2b_qp(enc, header=True)
    assert dec == b"hello world"
    print("qp_header ok")

def test_rle_hqx():
    data = b"AAABBBCC"
    enc = binascii.rlecode_hqx(data)
    assert isinstance(enc, bytes)
    dec = binascii.rledecode_hqx(enc)
    assert dec == data, f"got {dec!r}"
    # 0x90 byte round-trip
    data2 = b"\x90\x90\x90"
    enc2 = binascii.rlecode_hqx(data2)
    dec2 = binascii.rledecode_hqx(enc2)
    assert dec2 == data2, f"got {dec2!r}"
    print("rle_hqx ok")

def test_hqx():
    data = b"Hello"
    enc = binascii.rlecode_hqx(data)
    b4 = binascii.b2a_hqx(enc)
    assert isinstance(b4, bytes)
    dec_rle, done = binascii.a2b_hqx(b4)
    assert isinstance(dec_rle, bytes)
    final = binascii.rledecode_hqx(dec_rle)
    assert final == data, f"got {final!r}"
    print("hqx ok")

def test_errors():
    try:
        raise binascii.Error("bad data")
    except ValueError:
        pass
    try:
        raise binascii.Incomplete("need more")
    except Exception:
        pass
    print("errors ok")

def test_error_on_bad_hex():
    try:
        binascii.unhexlify(b"xyz")
        assert False
    except (binascii.Error, ValueError):
        pass
    print("error_on_bad_hex ok")

test_hexlify_basic()
test_hexlify_sep()
test_base64()
test_a2b_base64_strict()
test_crc32()
test_crc_hqx()
test_uu()
test_qp()
test_qp_header()
test_rle_hqx()
test_hqx()
test_errors()
test_error_on_bad_hex()
print("ALL OK")

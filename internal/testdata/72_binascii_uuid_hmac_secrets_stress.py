import binascii
import hmac
import hashlib
import secrets
import uuid

# --- binascii scenarios ----------------------------------------------------

# 1) hexlify of empty bytes.
print(binascii.hexlify(b""))

# 2) hexlify of a single byte.
print(binascii.hexlify(b"\x00"), binascii.hexlify(b"\xff"))

# 3) Uppercase hex still decodes.
print(binascii.unhexlify("DEADBEEF"))

# 4) Mixed-case hex decodes.
print(binascii.unhexlify("dEaDbEeF"))

# 5) Odd-length hex raises.
try:
    binascii.unhexlify("abc")
    print("no error")
except Exception:
    print("unhexlify odd raised")

# 6) Non-hex character raises.
try:
    binascii.unhexlify("zz")
    print("no error")
except Exception:
    print("unhexlify bad raised")

# 7) b2a_hex / a2b_hex aliases agree.
print(binascii.b2a_hex(b"abc") == binascii.hexlify(b"abc"))
print(binascii.a2b_hex(b"616263") == binascii.unhexlify(b"616263"))

# 8) unhexlify accepts bytearray input.
print(binascii.unhexlify(bytearray(b"68656c6c6f")))

# 9) b2a_base64 default appends newline.
print(binascii.b2a_base64(b"abc"))

# 10) b2a_base64 with newline=False suppresses it.
print(binascii.b2a_base64(b"abc", newline=False))

# 11) a2b_base64 strips surrounding whitespace.
print(binascii.a2b_base64(b"  YWJj\n  "))

# 12) a2b_base64 on empty returns b''.
print(binascii.a2b_base64(b""))

# 13) b2a_base64 + a2b_base64 round-trip.
for payload in [b"", b"x", b"hello world", b"\x00\x01\x02\xfe\xff"]:
    enc = binascii.b2a_base64(payload, newline=False)
    print(binascii.a2b_base64(enc) == payload)

# 14) crc32 seed chaining matches one-shot call.
chained = binascii.crc32(b"world", binascii.crc32(b"hello "))
oneshot = binascii.crc32(b"hello world")
print(chained == oneshot)

# 15) crc32 of bytes equals crc32 of bytearray.
print(binascii.crc32(b"abc") == binascii.crc32(bytearray(b"abc")))

# 16) crc32 of empty is 0.
print(binascii.crc32(b""))

# 17) hexlify of bytearray equals hexlify of bytes.
print(binascii.hexlify(bytearray(b"abc")) == binascii.hexlify(b"abc"))

# 18) unhexlify from str returns bytes.
print(type(binascii.unhexlify("616263")).__name__)

# --- hmac scenarios --------------------------------------------------------

# 19) hmac.new sha256 known vector (RFC 4231 style).
print(hmac.new(b"key", b"The quick brown fox jumps over the lazy dog", "sha256").hexdigest())

# 20) hmac.new md5 known vector.
print(hmac.new(b"key", b"hello", "md5").hexdigest())

# 21) hmac.new sha1 known vector.
print(hmac.new(b"key", b"hello", "sha1").hexdigest())

# 22) update() incrementally equals one-shot.
h1 = hmac.new(b"k", digestmod="sha256")
h1.update(b"abc")
h1.update(b"def")
h2 = hmac.new(b"k", b"abcdef", "sha256")
print(h1.hexdigest() == h2.hexdigest())

# 23) digest returns bytes, hexdigest returns str.
h = hmac.new(b"k", b"m", "sha256")
print(type(h.digest()).__name__, type(h.hexdigest()).__name__)

# 24) digest_size per algorithm.
print(hmac.new(b"k", digestmod="md5").digest_size)
print(hmac.new(b"k", digestmod="sha1").digest_size)
print(hmac.new(b"k", digestmod="sha256").digest_size)
print(hmac.new(b"k", digestmod="sha512").digest_size)

# 25) name carries the algorithm.
print(hmac.new(b"k", digestmod="sha256").name)

# 26) compare_digest equal bytes.
print(hmac.compare_digest(b"abc", b"abc"))

# 27) compare_digest different length.
print(hmac.compare_digest(b"abc", b"abcd"))

# 28) compare_digest for str.
print(hmac.compare_digest("hello", "hello"))

# 29) compare_digest differing content.
print(hmac.compare_digest(b"abc", b"abd"))

# 30) hmac.new with digestmod kwarg.
print(hmac.new(b"k", b"m", digestmod="sha256").hexdigest())

# 31) hmac.digest one-shot agrees with new().digest().
a = hmac.digest(b"k", b"m", "sha256")
b = hmac.new(b"k", b"m", "sha256").digest()
print(a == b)

# 32) hmac.new accepts hashlib constructor.
print(hmac.new(b"k", b"m", hashlib.sha1).hexdigest() == hmac.new(b"k", b"m", "sha1").hexdigest())

# 33) Empty message still digests.
print(len(hmac.new(b"k", b"", "sha256").digest()))

# 34) update after a digest extends state.
h = hmac.new(b"k", b"a", "sha256")
d1 = h.hexdigest()
h.update(b"b")
d2 = h.hexdigest()
print(d1 != d2)

# 35) sha512 digest is 64 bytes.
print(len(hmac.new(b"k", b"m", "sha512").digest()))

# 36) sha384 digest is 48 bytes.
print(len(hmac.new(b"k", b"m", "sha384").digest()))

# 37) Unknown algorithm raises.
try:
    hmac.new(b"k", b"m", "notareal")
    print("no error")
except Exception:
    print("unknown algo raised")

# --- secrets scenarios -----------------------------------------------------

# 38) token_bytes returns bytes of the requested length.
b = secrets.token_bytes(24)
print(isinstance(b, bytes), len(b))

# 39) token_bytes default is 32.
print(len(secrets.token_bytes()))

# 40) token_hex length is 2*n.
print(len(secrets.token_hex(12)))

# 41) token_urlsafe uses only URL-safe chars.
t = secrets.token_urlsafe(16)
print(isinstance(t, str))
safe = set("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_")
print(all(c in safe for c in t))

# 42) randbelow stays within [0, n).
for _ in range(5):
    v = secrets.randbelow(10)
    if not (0 <= v < 10):
        print("out of range:", v)
        break
else:
    print("randbelow in range")

# 43) randbelow(1) is always 0.
print(all(secrets.randbelow(1) == 0 for _ in range(10)))

# 44) randbits fits in k bits.
for _ in range(5):
    v = secrets.randbits(16)
    if not (0 <= v < 65536):
        print("out of range:", v)
        break
else:
    print("randbits in range")

# 45) randbits(0) is 0.
print(secrets.randbits(0))

# 46) choice on a list.
print(secrets.choice([10, 20, 30]) in (10, 20, 30))

# 47) choice on a tuple.
print(secrets.choice((1, 2, 3)) in (1, 2, 3))

# 48) choice on a str returns a 1-char str.
c = secrets.choice("abcdef")
print(isinstance(c, str) and len(c) == 1 and c in "abcdef")

# 49) compare_digest equal.
print(secrets.compare_digest(b"x", b"x"))

# 50) compare_digest empty inputs.
print(secrets.compare_digest(b"", b""))

# 51) token_bytes(0) returns empty bytes.
print(secrets.token_bytes(0))

# --- uuid scenarios --------------------------------------------------------

# 52) Parse hyphenated form.
print(uuid.UUID("12345678-1234-1234-8234-567812345678").hex)

# 53) Parse without hyphens.
print(uuid.UUID("12345678123412341234567812345678").hex)

# 54) Parse braced form.
print(uuid.UUID("{12345678-1234-5678-1234-567812345678}").hex)

# 55) Parse urn prefix.
print(uuid.UUID("urn:uuid:12345678-1234-5678-1234-567812345678").hex)

# 56) Invalid length raises.
try:
    uuid.UUID("not a uuid")
    print("no error")
except Exception:
    print("UUID invalid raised")

# 57) uuid4 version is 4.
print(uuid.uuid4().version)

# 58) uuid4 variant is RFC 4122.
print(uuid.uuid4().variant)

# 59) Two uuid4 calls differ.
print(uuid.uuid4() != uuid.uuid4())

# 60) str(UUID) is the canonical hyphenated form (36 chars).
print(len(str(uuid.uuid4())))

# 61) .hex has 32 chars and no hyphens.
h = uuid.uuid4().hex
print(len(h), "-" not in h)

# 62) .bytes is 16 bytes.
print(len(uuid.uuid4().bytes))

# 63) .bytes_le swaps the first three fields.
u = uuid.UUID("00112233-4455-6677-8899-aabbccddeeff")
print(u.bytes_le)

# 64) .int is non-negative and < 2**128.
u = uuid.uuid4()
print(0 <= u.int < (1 << 128))

# 65) .urn has the urn:uuid: prefix.
print(uuid.UUID("12345678-1234-5678-1234-567812345678").urn)

# 66) .fields returns a 6-tuple.
u = uuid.UUID("00112233-4455-6677-8899-aabbccddeeff")
print(u.fields)

# 67) UUID(bytes=...) round trips via .bytes.
u = uuid.uuid4()
u2 = uuid.UUID(bytes=u.bytes)
print(str(u) == str(u2))

# 68) UUID(int=...) round trips via .int.
u = uuid.uuid4()
u3 = uuid.UUID(int=u.int)
print(str(u) == str(u3))

# 69) uuid5 is deterministic.
a = uuid.uuid5(uuid.NAMESPACE_DNS, "example.com")
b = uuid.uuid5(uuid.NAMESPACE_DNS, "example.com")
print(a.hex == b.hex)

# 70) uuid3 is deterministic.
a = uuid.uuid3(uuid.NAMESPACE_DNS, "example.com")
b = uuid.uuid3(uuid.NAMESPACE_DNS, "example.com")
print(a.hex == b.hex)

# 71) uuid3 and uuid5 differ for the same inputs.
print(uuid.uuid3(uuid.NAMESPACE_DNS, "x").hex != uuid.uuid5(uuid.NAMESPACE_DNS, "x").hex)

# 72) NAMESPACE_DNS renders as the canonical RFC 4122 value.
print(str(uuid.NAMESPACE_DNS))

# --- cross-module scenarios ------------------------------------------------

# 73) unhexlify of hmac.hexdigest equals hmac.digest.
h = hmac.new(b"k", b"hi", "sha256")
print(binascii.unhexlify(h.hexdigest()) == h.digest())

# 74) token_hex length matches the hexlify of token_bytes.
n = 10
print(len(secrets.token_hex(n)) == len(binascii.hexlify(secrets.token_bytes(n))))

# 75) UUID.hex unhexlified equals UUID.bytes.
u = uuid.uuid4()
print(binascii.unhexlify(u.hex) == u.bytes)

# 76) An HMAC keyed by token_bytes still round-trips through compare_digest.
key = secrets.token_bytes(16)
msg = b"payload"
d1 = hmac.new(key, msg, "sha256").digest()
d2 = hmac.new(key, msg, "sha256").digest()
print(hmac.compare_digest(d1, d2))

# 77) token_hex(n) produces a string of 2n hex chars.
for n in [0, 1, 4, 16]:
    print(len(secrets.token_hex(n)) == 2 * n)

# 78) Two independent token_bytes calls differ (probabilistic; 16 bytes ≈ 10^-38 collision).
print(secrets.token_bytes(16) != secrets.token_bytes(16))

# 79) Different namespaces yield different uuid5s for the same name.
print(uuid.uuid5(uuid.NAMESPACE_DNS, "x") != uuid.uuid5(uuid.NAMESPACE_URL, "x"))

# 80) randbelow samples uniformly across a small sweep.
samples = [secrets.randbelow(3) for _ in range(50)]
print(all(0 <= s < 3 for s in samples))

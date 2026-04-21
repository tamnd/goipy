import binascii
import hmac
import hashlib
import secrets
import uuid

# --- binascii ---

print(binascii.hexlify(b"hello"))
print(binascii.unhexlify("68656c6c6f"))
print(binascii.b2a_hex(b"abc") == binascii.hexlify(b"abc"))
print(binascii.a2b_hex("616263") == binascii.unhexlify("616263"))

# base64 helpers include a trailing newline by default.
print(binascii.b2a_base64(b"hello"))
print(binascii.b2a_base64(b"hello", newline=False))
print(binascii.a2b_base64(b"aGVsbG8=\n"))

# crc32 matches the IEEE polynomial.
print(binascii.crc32(b"hello"))
print(binascii.crc32(b""))

# --- hmac ---

h = hmac.new(b"key", b"message", "sha256")
print(h.hexdigest())
print(h.name)
print(h.digest_size)

# hmac.digest is the one-shot API.
print(binascii.hexlify(hmac.digest(b"key", b"message", "sha256")))

# compare_digest is constant-time.
print(hmac.compare_digest(b"abc", b"abc"))
print(hmac.compare_digest(b"abc", b"abd"))

# Passing a hashlib constructor still works.
h = hmac.new(b"k", b"m", hashlib.sha1)
print(h.hexdigest())

# --- secrets ---

b = secrets.token_bytes(16)
print(isinstance(b, bytes), len(b))
print(len(secrets.token_hex(8)))
tok = secrets.token_urlsafe(8)
print(isinstance(tok, str))
print(0 <= secrets.randbelow(100) < 100)
print(0 <= secrets.randbits(8) < 256)
print(secrets.choice([1, 2, 3]) in (1, 2, 3))
print(secrets.compare_digest("a", "a"))

# --- uuid ---

u = uuid.UUID("12345678-1234-5678-1234-567812345678")
print(u)
print(u.hex)
print(u.bytes)
print(u.int)
print(u.version)

# uuid4 generates a v4 UUID.
r = uuid.uuid4()
print(r.version, len(str(r)))

# uuid5 is deterministic from namespace + name.
u5 = uuid.uuid5(uuid.NAMESPACE_DNS, "example.com")
print(u5)

# Round trip via bytes.
u2 = uuid.UUID(bytes=u.bytes)
print(u2 == u or str(u2) == str(u))

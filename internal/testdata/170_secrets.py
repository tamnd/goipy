import secrets
import string

# DEFAULT_ENTROPY constant
print("DEFAULT_ENTROPY:", secrets.DEFAULT_ENTROPY)

# token_bytes
print()
print("=== token_bytes ===")
t = secrets.token_bytes()
print("default type:", type(t).__name__)
print("default len:", len(t))
t16 = secrets.token_bytes(16)
print("16 type:", type(t16).__name__)
print("16 len:", len(t16))
t0 = secrets.token_bytes(0)
print("0 len:", len(t0))
print("0 type:", type(t0).__name__)
try:
    secrets.token_bytes(-1)
except ValueError as e:
    print("negative error:", e)

# token_hex
print()
print("=== token_hex ===")
h = secrets.token_hex()
print("default type:", type(h).__name__)
print("default len:", len(h))
h8 = secrets.token_hex(8)
print("8 len:", len(h8))
safe_hex = string.hexdigits[:16]  # 0-9a-f
print("all hex chars:", all(c in safe_hex for c in h))
try:
    secrets.token_hex(-1)
except ValueError as e:
    print("negative error:", e)

# token_urlsafe
print()
print("=== token_urlsafe ===")
u = secrets.token_urlsafe()
print("default type:", type(u).__name__)
print("default len:", len(u))
u16 = secrets.token_urlsafe(16)
print("16 len:", len(u16))
u64 = secrets.token_urlsafe(64)
print("64 len:", len(u64))
safe_chars = string.ascii_letters + string.digits + "-_"
print("all urlsafe:", all(c in safe_chars for c in u))
try:
    secrets.token_urlsafe(-1)
except ValueError as e:
    print("negative error:", e)

# randbelow
print()
print("=== randbelow ===")
for _ in range(20):
    r = secrets.randbelow(10)
    assert 0 <= r < 10, f"out of range: {r}"
print("randbelow(10) in range: True")
print("randbelow(1):", secrets.randbelow(1))
r_large = secrets.randbelow(1000000)
print("large in range:", 0 <= r_large < 1000000)
try:
    secrets.randbelow(0)
except ValueError as e:
    print("randbelow(0) error:", e)
try:
    secrets.randbelow(-5)
except ValueError as e:
    print("randbelow(-5) error:", e)

# randbits
print()
print("=== randbits ===")
print("randbits(0):", secrets.randbits(0))
for _ in range(20):
    r = secrets.randbits(8)
    assert 0 <= r < 256, f"out of range: {r}"
print("randbits(8) in range: True")
for _ in range(20):
    r = secrets.randbits(16)
    assert 0 <= r < 65536, f"out of range: {r}"
print("randbits(16) in range: True")
r1 = secrets.randbits(1)
print("randbits(1) in range:", r1 in (0, 1))
try:
    secrets.randbits(-1)
except ValueError as e:
    print("randbits(-1) error:", e)

# choice
print()
print("=== choice ===")
seq = [10, 20, 30, 40, 50]
for _ in range(20):
    c = secrets.choice(seq)
    assert c in seq, f"not in seq: {c}"
print("choice(list) always in seq: True")
word = "hello"
for _ in range(20):
    c = secrets.choice(word)
    assert c in word, f"not in word: {c}"
print("choice(str) always in str: True")
t = (100, 200, 300)
c = secrets.choice(t)
print("choice(tuple) in tuple:", c in t)
try:
    secrets.choice([])
except IndexError as e:
    print("choice([]) IndexError:", e)

# compare_digest
print()
print("=== compare_digest ===")
print("bytes equal:", secrets.compare_digest(b"abc", b"abc"))
print("bytes unequal:", secrets.compare_digest(b"abc", b"xyz"))
print("str equal:", secrets.compare_digest("abc", "abc"))
print("str unequal:", secrets.compare_digest("abc", "xyz"))
print("empty equal:", secrets.compare_digest(b"", b""))
print("empty unequal:", secrets.compare_digest(b"", b"x"))

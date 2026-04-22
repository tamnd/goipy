"""Tests for bytes and bytearray new methods."""

# --- bytes.fromhex ---
b = bytes.fromhex("deadbeef")
print(b)
print(bytes.fromhex("48 65 6c 6c 6f"))

# --- bytearray.fromhex ---
ba = bytearray.fromhex("deadbeef")
print(ba)

# --- hex ---
print(b"\xde\xad\xbe\xef".hex())
print(bytearray(b"\xca\xfe").hex())

# --- join ---
print(b", ".join([b"a", b"b", b"c"]))
print(bytearray(b"-").join([bytearray(b"x"), bytearray(b"y")]))

# --- strip / lstrip / rstrip ---
print(b"  hello  ".strip())
print(b"  hello  ".lstrip())
print(b"  hello  ".rstrip())
print(b"xxhello".lstrip(b"x"))
print(b"helloxx".rstrip(b"x"))
print(b"xxhelloxx".strip(b"x"))

# --- upper / lower ---
print(b"hello".upper())
print(b"WORLD".lower())
print(bytearray(b"hello").upper())
print(bytearray(b"WORLD").lower())

# --- center / ljust / rjust ---
print(b"hi".center(10))
print(b"hi".center(10, b"-"))
print(b"hi".ljust(10))
print(b"hi".ljust(10, b"."))
print(b"hi".rjust(10))
print(b"hi".rjust(10, b"."))

# --- zfill ---
print(b"42".zfill(5))
print(b"-42".zfill(5))
print(b"+42".zfill(5))

# --- count / find / index ---
print(b"hello".count(b"l"))
print(b"hello".find(b"l"))
print(b"hello".rfind(b"l"))
print(b"hello".index(b"l"))
try:
    b"hello".index(b"z")
except ValueError:
    print("ValueError")

# --- startswith / endswith ---
print(b"hello".startswith(b"he"))
print(b"hello".endswith(b"lo"))

# --- replace ---
print(b"hello".replace(b"l", b"r"))

# --- split ---
print(b"a b c".split())
print(b"a,b,c".split(b","))

# --- decode ---
print(b"hello".decode())
print(b"hello".decode("utf-8"))

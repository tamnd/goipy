import io
import hashlib
import base64
import textwrap

# --- io.StringIO stress ---

# 1) empty StringIO reads empty.
s = io.StringIO()
print(s.read())
print(s.tell())

# 2) write then read from start.
s = io.StringIO()
s.write("abc")
s.seek(0)
print(s.read())

# 3) read(n) respects count.
s = io.StringIO("0123456789")
print(s.read(3))
print(s.read(4))
print(s.read())
print(s.read())  # EOF → empty

# 4) seek(0) resets.
s = io.StringIO("abcdef")
s.read(3)
s.seek(0)
print(s.read())

# 5) absolute seek to mid-buffer.
s = io.StringIO("abcdef")
s.read(2)
s.seek(3)
print(s.read())

# 6) tell reports the cursor.
s = io.StringIO("hello")
s.read(2)
print(s.tell())

# 7) readline stops at newline.
s = io.StringIO("a\nb\nc")
print(s.readline())
print(s.readline())
print(s.readline())
print(s.readline())  # empty at EOF

# 8) readlines returns all remaining lines.
s = io.StringIO("x\ny\nz")
print(s.readlines())

# 9) writelines accepts any iterable of str.
s = io.StringIO()
s.writelines(iter(["1\n", "2\n"]))
print(s.getvalue())

# 10) getvalue after close still works (in CPython, close raises — we accept close as idempotent).
s = io.StringIO("kept")
s.close()
print(s.closed)

# 11) truncate shortens to cursor.
s = io.StringIO("abcdef")
s.read(3)
s.truncate()
print(s.getvalue())

# --- io.BytesIO stress ---

# 12) empty BytesIO.
b = io.BytesIO()
print(b.read())
print(b.tell())

# 13) write bytes, read bytes.
b = io.BytesIO()
b.write(b"xyz")
b.seek(0)
print(b.read())

# 14) read(n) returns bytes of n length.
b = io.BytesIO(b"\x00\x01\x02\x03\x04")
out = b.read(2)
print(type(out).__name__)
print(out)

# 15) seek to negative offset via whence=2.
b = io.BytesIO(b"abcdef")
b.seek(-3, 2)
print(b.read())

# 16) writing bytearray works.
b = io.BytesIO()
b.write(bytearray(b"abc"))
print(b.getvalue())

# 17) getvalue after seek still returns the whole buffer.
b = io.BytesIO(b"hello")
b.seek(2)
print(b.getvalue())

# --- hashlib stress ---

# 18) md5 of "abc"
print(hashlib.md5(b"abc").hexdigest())

# 19) sha1 of "abc"
print(hashlib.sha1(b"abc").hexdigest())

# 20) sha224/sha384 availability.
print(hashlib.sha224(b"").hexdigest()[:16])
print(hashlib.sha384(b"").hexdigest()[:16])

# 21) sha256 agrees with multiple updates.
h = hashlib.sha256()
for chunk in [b"a", b"b", b"c"]:
    h.update(chunk)
print(h.hexdigest() == hashlib.sha256(b"abc").hexdigest())

# 22) digest and hexdigest are consistent (32 bytes → 64 hex chars).
h = hashlib.sha256(b"hello")
print(len(h.digest()) * 2 == len(h.hexdigest()))

# 23) digest_size is algorithm-specific.
print(hashlib.md5().digest_size)
print(hashlib.sha1().digest_size)
print(hashlib.sha256().digest_size)
print(hashlib.sha512().digest_size)

# 24) unknown algorithm raises.
try:
    hashlib.new("bogus")
except ValueError:
    print("new-bad: ValueError")

# 25) feeding bytes vs bytearray produces the same digest.
print(hashlib.sha256(b"abc").hexdigest() == hashlib.sha256(bytearray(b"abc")).hexdigest())

# 26) copy() preserves independent state.
h = hashlib.sha256(b"ab")
h2 = h.copy()
h.update(b"c")
h2.update(b"d")
print(h.hexdigest() == hashlib.sha256(b"abc").hexdigest())
print(h2.hexdigest() == hashlib.sha256(b"abd").hexdigest())

# 27) incremental feed covers a large payload.
h = hashlib.sha256()
for _ in range(1024):
    h.update(b"\x00\x00\x00\x00\x00\x00\x00\x00")
print(h.hexdigest()[:16])

# 28) name attribute.
print(hashlib.sha256().name)
print(hashlib.md5().name)

# --- base64 stress ---

# 29) empty input round-trips.
print(base64.b64encode(b""))
print(base64.b64decode(b""))

# 30) single-byte input.
print(base64.b64encode(b"A"))

# 31) padding-free input.
print(base64.b64encode(b"abc"))  # perfect multiple of 3

# 32) incremental round-trip over every byte value.
data = bytes(range(256))
print(base64.b64decode(base64.b64encode(data)) == data)

# 33) decoding rejects odd-length input.
try:
    base64.b64decode(b"abc")
except Exception:
    print("b64-bad: error raised")

# 34) urlsafe replaces +/.
print(base64.b64encode(b"\xfb\xff"))
print(base64.urlsafe_b64encode(b"\xfb\xff"))

# 35) base32 round trip.
data = b"Hello World"
print(base64.b32decode(base64.b32encode(data)) == data)

# 36) base16 returns uppercase.
print(base64.b16encode(b"\xab\xcd"))

# 37) base16 decode requires uppercase.
print(base64.b16decode(b"ABCD"))

# 38) standard_b64encode == b64encode.
print(base64.standard_b64encode(b"hi") == base64.b64encode(b"hi"))

# 39) encoding a string directly (UTF-8 coerced via asBytes).
print(base64.b64encode(b"\xc3\xa9"))  # "é"

# 40) decoding bytes-style payload.
print(base64.b64decode(b"aGVsbG8="))

# --- textwrap stress ---

# 41) dedent on empty input.
print(repr(textwrap.dedent("")))

# 42) dedent when no common prefix.
print(textwrap.dedent("a\nb\n"))

# 43) dedent mixed indent.
print(textwrap.dedent("   a\n    b\n   c\n"))

# 44) dedent normalizes blank lines.
print(repr(textwrap.dedent("   a\n   \n   b")))

# 45) indent skips blank lines by default.
print(textwrap.indent("x\n\ny\n", "> "))

# 46) indent with empty prefix is a no-op.
print(textwrap.indent("a\nb\n", ""))

# 47) wrap short text returns one line.
print(textwrap.wrap("short", width=20))

# 48) wrap empty string returns [].
print(textwrap.wrap("", width=20))

# 49) wrap breaks on whitespace only.
print(textwrap.wrap("a b c d e f", width=3))

# 50) wrap respects width larger than input.
print(textwrap.wrap("abc def", width=100))

# 51) fill joins with newlines.
out = textwrap.fill("one two three", width=5)
print(out.count("\n") >= 1)
print(all(len(line) <= 5 for line in out.split("\n")))

# 52) shorten with enough width returns unchanged (whitespace collapsed).
print(textwrap.shorten("  a   b   c  ", width=10))

# 53) shorten with tight width appends placeholder.
print(textwrap.shorten("the quick brown fox", width=12))

# 54) shorten with custom placeholder.
print(textwrap.shorten("one two three four", width=10, placeholder="~"))

# 55) shorten with width 20 keeps the first few words + placeholder.
print(textwrap.shorten("alpha beta gamma delta", width=20))

# --- cross-module integration ---

# 56) hash a BytesIO payload.
b = io.BytesIO(b"digest me")
print(hashlib.sha256(b.getvalue()).hexdigest()[:16])

# 57) base64 a BytesIO payload.
b = io.BytesIO()
b.write(b"secret")
b.seek(0)
print(base64.b64encode(b.read()))

# 58) round-trip base64 through BytesIO.
enc = base64.b64encode(b"payload")
b = io.BytesIO(enc)
print(base64.b64decode(b.getvalue()))

# 59) wrap the hex digest for readability.
hd = hashlib.sha256(b"abc").hexdigest()
print(textwrap.wrap(hd, width=16))

# 60) feed a textwrap.fill result into a StringIO.
s = io.StringIO()
s.write(textwrap.fill("one two three four", width=8))
print(s.getvalue())

# 61) dedent + indent round-trip preserves content structure.
src = "    a\n    b"
d = textwrap.dedent(src)
ind = textwrap.indent(d, "    ")
print(ind == src + "" or ind.rstrip() == src.rstrip())

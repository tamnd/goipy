import io
import hashlib
import base64
import textwrap

# --- io.StringIO basics ---

s = io.StringIO()
s.write("hello ")
s.write("world")
print(s.getvalue())
print(s.tell())

s.seek(0)
print(s.read(5))
print(s.read())

# initial_value.
s = io.StringIO("line1\nline2\nline3\n")
print(s.readline())
print(s.readline())
print(s.readlines())

# writelines.
s = io.StringIO()
s.writelines(["a\n", "b\n", "c\n"])
print(s.getvalue())

# close.
s.close()
print(s.closed)

# --- io.BytesIO basics ---

b = io.BytesIO()
b.write(b"hello")
b.write(b" world")
print(b.getvalue())
print(b.tell())

b.seek(0)
print(b.read(5))

# initial_value.
b = io.BytesIO(b"\x01\x02\x03")
print(b.read())

# seek with whence=2 (end).
b = io.BytesIO(b"abcdef")
b.seek(-2, 2)
print(b.read())

# --- hashlib basics ---

# Empty input digests are well-known.
print(hashlib.md5(b"").hexdigest())
print(hashlib.sha1(b"").hexdigest())
print(hashlib.sha256(b"").hexdigest())

# "hello" digests.
print(hashlib.md5(b"hello").hexdigest())
print(hashlib.sha256(b"hello").hexdigest())

# update() matches one-shot.
h = hashlib.sha256()
h.update(b"hel")
h.update(b"lo")
print(h.hexdigest() == hashlib.sha256(b"hello").hexdigest())

# digest() returns bytes, hexdigest returns str.
h = hashlib.sha1(b"abc")
print(type(h.digest()).__name__)
print(type(h.hexdigest()).__name__)
print(len(h.digest()))
print(h.digest_size)
print(h.name)

# new() dispatches by name.
h = hashlib.new("sha256", b"hello")
print(h.hexdigest() == hashlib.sha256(b"hello").hexdigest())

# sha512.
print(hashlib.sha512(b"").hexdigest()[:16])

# --- base64 basics ---

print(base64.b64encode(b"hello"))
print(base64.b64decode(b"aGVsbG8="))
print(base64.b64decode("aGVsbG8="))

# Round trip.
data = b"some binary \x00\x01\x02 data"
print(base64.b64decode(base64.b64encode(data)) == data)

# URL-safe variant.
print(base64.urlsafe_b64encode(b"\xfb\xff?"))
print(base64.urlsafe_b64decode(b"-_8_"))

# base32.
print(base64.b32encode(b"foo"))
print(base64.b32decode(base64.b32encode(b"foo")))

# base16.
print(base64.b16encode(b"foo"))
print(base64.b16decode(b"666F6F"))

# --- textwrap basics ---

text = "   line one\n   line two\n   line three"
print(textwrap.dedent(text))

# indent skips blank lines by default.
print(textwrap.indent("a\n\nb\n", ">> "))

# wrap returns a list.
print(textwrap.wrap("The quick brown fox jumps over the lazy dog", width=15))

# fill returns a single string with newlines.
print(textwrap.fill("one two three four five six", width=10))

# shorten collapses whitespace.
print(textwrap.shorten("The quick brown fox jumps over the lazy dog", width=20))
print(textwrap.shorten("   a   b   c   ", width=10))

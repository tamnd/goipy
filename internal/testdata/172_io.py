"""Tests for the extended io module."""
import io
import os
import tempfile

# --- Module-level constants ---
print(isinstance(io.DEFAULT_BUFFER_SIZE, int) and io.DEFAULT_BUFFER_SIZE > 0)  # True
print(io.SEEK_SET == 0)                 # True
print(io.SEEK_CUR == 1)                 # True
print(io.SEEK_END == 2)                 # True

# --- UnsupportedOperation ---
print(issubclass(io.UnsupportedOperation, OSError))    # True
print(issubclass(io.UnsupportedOperation, ValueError)) # True

# --- io.open() ---
tmp = tempfile.mktemp(suffix=".txt")
with io.open(tmp, "w") as f:
    f.write("hello")
with io.open(tmp, "r") as f:
    print(f.read())  # hello
os.remove(tmp)

# --- StringIO: readable / writable / seekable ---
sio = io.StringIO("abc")
print(sio.readable())   # True
print(sio.writable())   # True
print(sio.seekable())   # True

# --- StringIO iteration ---
sio2 = io.StringIO("line1\nline2\nline3\n")
lines = list(sio2)
print(lines)  # ['line1\n', 'line2\n', 'line3\n']

# StringIO iteration partial (no trailing newline)
sio3 = io.StringIO("a\nb")
lines3 = [l for l in sio3]
print(lines3)  # ['a\n', 'b']

# --- BytesIO: readline ---
bio = io.BytesIO(b"line1\nline2\nline3\n")
print(bio.readline())  # b'line1\n'
print(bio.readline())  # b'line2\n'

# BytesIO: readlines
bio2 = io.BytesIO(b"a\nb\nc\n")
print(bio2.readlines())  # [b'a\n', b'b\n', b'c\n']

# BytesIO: writelines
bio3 = io.BytesIO()
bio3.writelines([b"foo", b"bar", b"baz"])
print(bio3.getvalue())  # b'foobarbaz'

# BytesIO: truncate
bio4 = io.BytesIO(b"hello world")
bio4.truncate(5)
print(bio4.getvalue())  # b'hello'

# BytesIO: truncate at current pos
bio5 = io.BytesIO(b"hello world")
bio5.seek(5)
bio5.truncate()
print(bio5.getvalue())  # b'hello'

# BytesIO: readable / writable / seekable
bio6 = io.BytesIO()
print(bio6.readable())   # True
print(bio6.writable())   # True
print(bio6.seekable())   # True

# BytesIO: iteration
bio7 = io.BytesIO(b"x\ny\nz\n")
blines = list(bio7)
print(blines)  # [b'x\n', b'y\n', b'z\n']

# BytesIO: writelines with generator
bio8 = io.BytesIO()
bio8.writelines(b + b";" for b in [b"a", b"b", b"c"])
print(bio8.getvalue())  # b'a;b;c;'

# --- File: seek / tell ---
tmp2 = tempfile.mktemp(suffix=".bin")
with open(tmp2, "wb") as f:
    f.write(b"0123456789")
with open(tmp2, "rb") as f:
    f.seek(5)
    print(f.tell())       # 5
    print(f.read(3))      # b'567'
    f.seek(-3, io.SEEK_END)
    print(f.read())       # b'789'
    f.seek(0, io.SEEK_SET)
    print(f.tell())       # 0

# File: truncate
tmp3 = tempfile.mktemp(suffix=".txt")
with open(tmp3, "w") as f:
    f.write("hello world")
with open(tmp3, "r+") as f:
    f.truncate(5)
    f.seek(0)
    print(f.read())  # hello

# File: fileno
tmp4 = tempfile.mktemp(suffix=".txt")
with open(tmp4, "w") as f:
    fd = f.fileno()
    print(isinstance(fd, int))  # True
    print(fd > 0)               # True

# File: isatty
with open(tmp4, "r") as f:
    print(f.isatty())  # False

# File: readable / writable / seekable
with open(tmp4, "r") as f:
    print(f.readable())   # True
    print(f.writable())   # False
    print(f.seekable())   # True

with open(tmp4, "w") as f:
    print(f.readable())   # False
    print(f.writable())   # True

# File: encoding / errors (text mode only)
with open(tmp4, "r") as f:
    print(f.encoding.lower() == "utf-8")  # True
    print(f.errors)                       # strict

# File: iteration via for loop
tmp5 = tempfile.mktemp(suffix=".txt")
with open(tmp5, "w") as f:
    f.writelines(["alpha\n", "beta\n", "gamma\n"])
with open(tmp5, "r") as f:
    flines = list(f)
print(flines)  # ['alpha\n', 'beta\n', 'gamma\n']

# File: readlines() preserves empty last line correctly
tmp6 = tempfile.mktemp(suffix=".txt")
with open(tmp6, "w") as f:
    f.write("a\nb\n")
with open(tmp6, "r") as f:
    print(f.readlines())  # ['a\n', 'b\n']

# File: readlines() no trailing newline
tmp7 = tempfile.mktemp(suffix=".txt")
with open(tmp7, "w") as f:
    f.write("x\ny")
with open(tmp7, "r") as f:
    print(f.readlines())  # ['x\n', 'y']

# File: writelines with generator
tmp8 = tempfile.mktemp(suffix=".txt")
with open(tmp8, "w") as f:
    f.writelines(s + "\n" for s in ["one", "two", "three"])
with open(tmp8, "r") as f:
    print(f.read())  # one\ntwo\nthree\n

# Clean up
for p in [tmp2, tmp3, tmp4, tmp5, tmp6, tmp7, tmp8]:
    try:
        os.remove(p)
    except OSError:
        pass

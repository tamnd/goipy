import gzip
import os
import tempfile

# ===== Constants =====
print(gzip.READ)                    # rb
print(gzip.WRITE)                   # wb
print(gzip.READ_BUFFER_SIZE)        # 131072
print(gzip.FTEXT)                   # 1
print(gzip.FHCRC)                   # 2
print(gzip.FEXTRA)                  # 4
print(gzip.FNAME)                   # 8
print(gzip.FCOMMENT)                # 16

# ===== BadGzipFile =====
print(issubclass(gzip.BadGzipFile, OSError))    # True
print(issubclass(gzip.BadGzipFile, Exception))  # True

# ===== compress / decompress =====
data = b"hello world " * 100
compressed = gzip.compress(data)
print(isinstance(compressed, bytes))             # True
print(compressed[:2] == b'\x1f\x8b')           # True  (gzip magic)
print(gzip.decompress(compressed) == data)      # True

# ===== compress levels =====
c1 = gzip.compress(data, 1)
c9 = gzip.compress(data, 9)
print(gzip.decompress(c1) == data)              # True
print(gzip.decompress(c9) == data)              # True
print(len(c1) >= len(c9))                       # True

# ===== compress with compresslevel kwarg =====
ck = gzip.compress(data, compresslevel=6)
print(gzip.decompress(ck) == data)              # True

# ===== compress mtime=0 is deterministic =====
a = gzip.compress(b"test data", mtime=0)
b_ = gzip.compress(b"test data", mtime=0)
print(a == b_)                                  # True
print(gzip.decompress(a) == b"test data")       # True

# ===== compress mtime=None still decompresses =====
cn = gzip.compress(b"mtime none test")
print(gzip.decompress(cn) == b"mtime none test") # True

# ===== decompress bad data raises BadGzipFile / OSError =====
try:
    gzip.decompress(b"not valid gzip data!!")
except (gzip.BadGzipFile, OSError):
    print("BadGzipFile raised")                 # BadGzipFile raised

# ===== decompress multi-member stream =====
m1 = gzip.compress(b"part1", mtime=0)
m2 = gzip.compress(b"part2", mtime=0)
print(gzip.decompress(m1 + m2) == b"part1part2") # True

with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, "test.gz")

    # ===== GzipFile write =====
    with gzip.GzipFile(path, 'wb') as f:
        n = f.write(b"hello gzipfile")
        print(n)                                 # 14
        print(f.writable())                      # True
        print(f.readable())                      # False
        print(f.seekable())                      # True
        print(f.closed)                          # False
    print(f.closed)                              # True

    # ===== GzipFile read =====
    with gzip.GzipFile(path, 'rb') as f:
        content = f.read()
        print(content)                           # b'hello gzipfile'
        print(f.readable())                      # True
        print(f.writable())                      # False
        print(f.seekable())                      # True

    # ===== GzipFile name and mode =====
    with gzip.GzipFile(path, 'rb') as f:
        print(isinstance(f.name, str))           # True
        print(f.mode == gzip.READ)               # True
    with gzip.GzipFile(path, 'wb') as f:
        print(f.mode == gzip.WRITE)              # True
        f.write(b"overwrite")

    # ===== GzipFile read(size) partial reads =====
    with gzip.GzipFile(path, 'wb') as f:
        f.write(b"abcdefghij")
    with gzip.GzipFile(path, 'rb') as f:
        print(f.read(3))                         # b'abc'
        print(f.tell())                          # 3
        print(f.read(3))                         # b'def'
        print(f.tell())                          # 6
        print(f.read())                          # b'ghij'

    # ===== GzipFile seek =====
    with gzip.GzipFile(path, 'rb') as f:
        f.seek(5)
        print(f.read(2))                         # b'fg'
        f.seek(0)
        print(f.read(1))                         # b'a'
        # seek from current (whence=1)
        f.seek(3, 1)
        print(f.read(1))                         # b'e'
        # seek from end (whence=2)
        f.seek(-2, 2)
        print(f.read())                          # b'ij'

    # ===== GzipFile tell on write =====
    with gzip.GzipFile(path, 'wb') as f:
        print(f.tell())                          # 0
        f.write(b"xyz")
        print(f.tell())                          # 3

    # ===== GzipFile readline =====
    lines_path = os.path.join(tmpdir, "lines.gz")
    with gzip.GzipFile(lines_path, 'wb') as f:
        f.write(b"line1\nline2\nline3")
    with gzip.GzipFile(lines_path, 'rb') as f:
        print(f.readline())                      # b'line1\n'
        print(f.readline())                      # b'line2\n'
        print(f.readline())                      # b'line3'
        print(f.readline())                      # b''  (EOF)

    # ===== GzipFile peek =====
    with gzip.GzipFile(path, 'rb') as f:
        p = f.peek(2)
        print(p[:2])                             # b'xy'
        print(f.tell())                          # 0  (peek doesn't advance)
        print(f.read(1))                         # b'x'

    # ===== gzip.open binary write + read =====
    path2 = os.path.join(tmpdir, "open.gz")
    with gzip.open(path2, 'wb') as f:
        f.write(b"gzip open write")
    with gzip.open(path2, 'rb') as f:
        print(f.read())                          # b'gzip open write'

    # ===== gzip.open compresslevel kwarg =====
    path3 = os.path.join(tmpdir, "lvl.gz")
    with gzip.open(path3, 'wb', compresslevel=1) as f:
        f.write(b"fast compress")
    with gzip.open(path3, 'rb') as f:
        print(f.read())                          # b'fast compress'

    # ===== gzip.open text write + read =====
    path4 = os.path.join(tmpdir, "text.gz")
    with gzip.open(path4, 'wt') as f:
        f.write("hello text mode\n")
    with gzip.open(path4, 'rt') as f:
        print(f.read())                          # hello text mode\n

    # ===== GzipFile multiple writes =====
    path5 = os.path.join(tmpdir, "multi.gz")
    with gzip.GzipFile(path5, 'wb') as f:
        f.write(b"part1")
        f.write(b"part2")
        f.write(b"part3")
    with gzip.GzipFile(path5, 'rb') as f:
        print(f.read())                          # b'part1part2part3'

    # ===== gzip.open with positional mode =====
    path6 = os.path.join(tmpdir, "pos.gz")
    with gzip.open(path6, 'wb') as f:
        f.write(b"positional mode")
    with gzip.open(path6, 'rb') as f:
        print(f.read())                          # b'positional mode'

print('done')

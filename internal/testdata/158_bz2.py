import bz2
import os
import tempfile

# ===== compress / decompress =====
data = b"hello bz2 world " * 50
compressed = bz2.compress(data)
print(isinstance(compressed, bytes))             # True
print(compressed[:3] == b'BZh')                  # True
print(bz2.decompress(compressed) == data)        # True

# ===== compresslevel =====
c1 = bz2.compress(data, 1)
c9 = bz2.compress(data, 9)
print(bz2.decompress(c1) == data)                # True
print(bz2.decompress(c9) == data)                # True

# ===== compresslevel kwarg =====
ck = bz2.compress(data, compresslevel=5)
print(bz2.decompress(ck) == data)                # True

# ===== multi-stream decompress =====
m1 = bz2.compress(b"stream1")
m2 = bz2.compress(b"stream2")
print(bz2.decompress(m1 + m2) == b"stream1stream2")  # True

# ===== decompress bad data =====
try:
    bz2.decompress(b"not a valid bz2 stream!!")
except OSError:
    print("OSError raised")                      # OSError raised

# ===== BZ2Compressor =====
c = bz2.BZ2Compressor()
out = c.compress(b"hello ") + c.compress(b"bz2compressor") + c.flush()
print(bz2.decompress(out) == b"hello bz2compressor")  # True

c2 = bz2.BZ2Compressor(1)
out2 = c2.compress(b"level 1") + c2.flush()
print(bz2.decompress(out2) == b"level 1")        # True

# ===== BZ2Decompressor basic =====
blob = bz2.compress(b"abcdefghij")
d = bz2.BZ2Decompressor()
print(d.decompress(blob) == b"abcdefghij")       # True
print(d.eof)                                     # True
print(d.unused_data)                             # b''

# ===== BZ2Decompressor chunked =====
d2 = bz2.BZ2Decompressor()
half = len(blob) // 2
p1 = d2.decompress(blob[:half])
p2 = d2.decompress(blob[half:])
print(p1 + p2 == b"abcdefghij")                  # True
print(d2.eof)                                    # True

# ===== BZ2Decompressor unused_data =====
trailing = blob + b"TRAILER"
d3 = bz2.BZ2Decompressor()
print(d3.decompress(trailing) == b"abcdefghij")  # True
print(d3.eof)                                    # True
print(d3.unused_data == b"TRAILER")              # True

# ===== BZ2Decompressor max_length =====
d4 = bz2.BZ2Decompressor()
first = d4.decompress(blob, max_length=3)
print(first == b"abc")                           # True
print(d4.needs_input)                            # False
rest = d4.decompress(b"")
print(first + rest == b"abcdefghij")             # True

with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, "test.bz2")

    # ===== BZ2File write =====
    with bz2.BZ2File(path, 'wb') as f:
        n = f.write(b"hello bz2file")
        print(n)                                 # 13
        print(f.writable())                      # True
        print(f.readable())                      # False
        print(f.closed)                          # False
    print(f.closed)                              # True

    # ===== BZ2File read =====
    with bz2.BZ2File(path, 'rb') as f:
        content = f.read()
        print(content)                           # b'hello bz2file'
        print(f.readable())                      # True
        print(f.writable())                      # False

    # ===== BZ2File name and mode =====
    with bz2.BZ2File(path, 'rb') as f:
        print(isinstance(f.name, str))           # True
        print(f.mode == 'rb')                    # True
    with bz2.BZ2File(path, 'wb') as f:
        print(f.mode == 'wb')                    # True
        f.write(b"overwrite")

    # ===== BZ2File partial reads =====
    with bz2.BZ2File(path, 'wb') as f:
        f.write(b"abcdefghij")
    with bz2.BZ2File(path, 'rb') as f:
        print(f.read(3))                         # b'abc'
        print(f.tell())                          # 3
        print(f.read(3))                         # b'def'
        print(f.tell())                          # 6
        print(f.read())                          # b'ghij'

    # ===== BZ2File seek =====
    with bz2.BZ2File(path, 'rb') as f:
        f.seek(5)
        print(f.read(2))                         # b'fg'
        f.seek(0)
        print(f.read(1))                         # b'a'
        f.seek(3, 1)
        print(f.read(1))                         # b'e'
        f.seek(-2, 2)
        print(f.read())                          # b'ij'

    # ===== BZ2File readline =====
    lines_path = os.path.join(tmpdir, "lines.bz2")
    with bz2.BZ2File(lines_path, 'wb') as f:
        f.write(b"line1\nline2\nline3")
    with bz2.BZ2File(lines_path, 'rb') as f:
        print(f.readline())                      # b'line1\n'
        print(f.readline())                      # b'line2\n'
        print(f.readline())                      # b'line3'
        print(f.readline())                      # b''

    # ===== BZ2File peek =====
    with bz2.BZ2File(path, 'rb') as f:
        p = f.peek(3)
        print(p[:3])                             # b'abc'
        print(f.tell())                          # 0
        print(f.read(1))                         # b'a'

    # ===== bz2.open binary =====
    path2 = os.path.join(tmpdir, "open.bz2")
    with bz2.open(path2, 'wb') as f:
        f.write(b"bz2 open write")
    with bz2.open(path2, 'rb') as f:
        print(f.read())                          # b'bz2 open write'

    # ===== bz2.open compresslevel =====
    path3 = os.path.join(tmpdir, "lvl.bz2")
    with bz2.open(path3, 'wb', compresslevel=1) as f:
        f.write(b"fast compress")
    with bz2.open(path3, 'rb') as f:
        print(f.read())                          # b'fast compress'

    # ===== bz2.open text =====
    path4 = os.path.join(tmpdir, "text.bz2")
    with bz2.open(path4, 'wt') as f:
        f.write("hello text mode\n")
    with bz2.open(path4, 'rt') as f:
        print(f.read())                          # hello text mode\n

    # ===== multiple writes =====
    path5 = os.path.join(tmpdir, "multi.bz2")
    with bz2.BZ2File(path5, 'wb') as f:
        f.write(b"part1")
        f.write(b"part2")
        f.write(b"part3")
    with bz2.BZ2File(path5, 'rb') as f:
        print(f.read())                          # b'part1part2part3'

print('done')

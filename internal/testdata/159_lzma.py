import lzma
import os
import tempfile

# ===== basic compress/decompress (FORMAT_XZ default) =====
data = b"hello lzma world " * 50
c = lzma.compress(data)
print(isinstance(c, bytes))                    # True
print(c[:6] == b'\xfd7zXZ\x00')                # True
print(lzma.decompress(c) == data)              # True

# ===== explicit check= argument =====
c_none = lzma.compress(data, check=lzma.CHECK_NONE)
c_c32 = lzma.compress(data, check=lzma.CHECK_CRC32)
c_c64 = lzma.compress(data, check=lzma.CHECK_CRC64)
c_sha = lzma.compress(data, check=lzma.CHECK_SHA256)
print(lzma.decompress(c_none) == data)         # True
print(lzma.decompress(c_c32) == data)          # True
print(lzma.decompress(c_c64) == data)          # True
print(lzma.decompress(c_sha) == data)          # True

# ===== FORMAT_ALONE =====
ca = lzma.compress(data, format=lzma.FORMAT_ALONE)
print(lzma.decompress(ca, format=lzma.FORMAT_ALONE) == data)  # True
# FORMAT_AUTO handles ALONE too
print(lzma.decompress(ca) == data)             # True

# ===== multi-stream decompress =====
m = lzma.compress(b"s1") + lzma.compress(b"s2")
print(lzma.decompress(m) == b"s1s2")           # True

# ===== invalid data =====
try:
    lzma.decompress(b"not a valid lzma stream!!")
except lzma.LZMAError:
    print("LZMAError raised")                   # LZMAError raised

# ===== LZMACompressor =====
co = lzma.LZMACompressor()
out = co.compress(b"hello ") + co.compress(b"compressor") + co.flush()
print(lzma.decompress(out) == b"hello compressor")  # True

co2 = lzma.LZMACompressor(format=lzma.FORMAT_ALONE)
out2 = co2.compress(b"alone mode") + co2.flush()
print(lzma.decompress(out2, format=lzma.FORMAT_ALONE) == b"alone mode")  # True

# ===== LZMADecompressor basic =====
d = lzma.LZMADecompressor()
print(d.decompress(c) == data)                 # True
print(d.eof)                                   # True
print(d.unused_data)                           # b''

# ===== LZMADecompressor chunked =====
d2 = lzma.LZMADecompressor()
half = len(c) // 2
p1 = d2.decompress(c[:half])
p2 = d2.decompress(c[half:])
print(p1 + p2 == data)                         # True
print(d2.eof)                                  # True

# ===== LZMADecompressor unused_data =====
trailing = c + b"TRAILER"
d3 = lzma.LZMADecompressor()
print(d3.decompress(trailing) == data)         # True
print(d3.eof)                                  # True
print(d3.unused_data == b"TRAILER")            # True

# ===== LZMADecompressor max_length =====
d4 = lzma.LZMADecompressor()
first = d4.decompress(c, max_length=3)
print(first == data[:3])                       # True
print(d4.needs_input)                          # False
rest = d4.decompress(b"")
print(first + rest == data)                    # True

# ===== is_check_supported =====
print(lzma.is_check_supported(lzma.CHECK_NONE))     # True
print(lzma.is_check_supported(lzma.CHECK_CRC32))    # True
print(lzma.is_check_supported(lzma.CHECK_CRC64))    # True
print(lzma.is_check_supported(lzma.CHECK_SHA256))   # True

with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, "test.xz")

    # ===== LZMAFile write =====
    with lzma.LZMAFile(path, 'wb') as f:
        n = f.write(b"hello lzmafile")
        print(n)                               # 14
        print(f.writable())                    # True
        print(f.readable())                    # False
        print(f.closed)                        # False
    print(f.closed)                            # True

    # ===== LZMAFile read =====
    with lzma.LZMAFile(path, 'rb') as f:
        content = f.read()
        print(content)                         # b'hello lzmafile'
        print(f.readable())                    # True
        print(f.writable())                    # False

    # ===== LZMAFile name and mode =====
    with lzma.LZMAFile(path, 'rb') as f:
        print(isinstance(f.name, str))         # True
        print(f.mode == 'rb')                  # True

    # ===== LZMAFile partial reads =====
    with lzma.LZMAFile(path, 'wb') as f:
        f.write(b"abcdefghij")
    with lzma.LZMAFile(path, 'rb') as f:
        print(f.read(3))                       # b'abc'
        print(f.tell())                        # 3
        print(f.read(3))                       # b'def'
        print(f.tell())                        # 6
        print(f.read())                        # b'ghij'

    # ===== LZMAFile seek =====
    with lzma.LZMAFile(path, 'rb') as f:
        f.seek(5)
        print(f.read(2))                       # b'fg'
        f.seek(0)
        print(f.read(1))                       # b'a'
        f.seek(3, 1)
        print(f.read(1))                       # b'e'
        f.seek(-2, 2)
        print(f.read())                        # b'ij'

    # ===== LZMAFile readline =====
    lines_path = os.path.join(tmpdir, "lines.xz")
    with lzma.LZMAFile(lines_path, 'wb') as f:
        f.write(b"line1\nline2\nline3")
    with lzma.LZMAFile(lines_path, 'rb') as f:
        print(f.readline())                    # b'line1\n'
        print(f.readline())                    # b'line2\n'
        print(f.readline())                    # b'line3'
        print(f.readline())                    # b''

    # ===== LZMAFile peek =====
    with lzma.LZMAFile(path, 'rb') as f:
        p = f.peek(3)
        print(p[:3])                           # b'abc'
        print(f.tell())                        # 0
        print(f.read(1))                       # b'a'

    # ===== lzma.open binary =====
    path2 = os.path.join(tmpdir, "open.xz")
    with lzma.open(path2, 'wb') as f:
        f.write(b"lzma open write")
    with lzma.open(path2, 'rb') as f:
        print(f.read())                        # b'lzma open write'

    # ===== lzma.open with check =====
    path3 = os.path.join(tmpdir, "check.xz")
    with lzma.open(path3, 'wb', check=lzma.CHECK_CRC32) as f:
        f.write(b"crc32 check")
    with lzma.open(path3, 'rb') as f:
        print(f.read())                        # b'crc32 check'

    # ===== lzma.open text =====
    path4 = os.path.join(tmpdir, "text.xz")
    with lzma.open(path4, 'wt') as f:
        f.write("hello text mode\n")
    with lzma.open(path4, 'rt') as f:
        print(f.read())                        # hello text mode\n

    # ===== multiple writes =====
    path5 = os.path.join(tmpdir, "multi.xz")
    with lzma.LZMAFile(path5, 'wb') as f:
        f.write(b"part1")
        f.write(b"part2")
        f.write(b"part3")
    with lzma.LZMAFile(path5, 'rb') as f:
        print(f.read())                        # b'part1part2part3'

print('done')

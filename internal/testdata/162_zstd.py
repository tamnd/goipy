import os
import tempfile
from compression import zstd

# ===== compress / decompress =====
empty = zstd.compress(b'')
print(zstd.decompress(empty) == b'')                       # True

short = zstd.compress(b'hello zstd')
print(zstd.decompress(short) == b'hello zstd')             # True

data = b'abcdefghij' * 20000   # 200 KB, forces multi-block
out = zstd.compress(data)
print(zstd.decompress(out) == data)                        # True
print(isinstance(out, bytes))                              # True

# ===== ZstdCompressor =====
co = zstd.ZstdCompressor()
part1 = co.compress(b'hello ')
part2 = co.compress(b'zstd compressor')
tail = co.flush()
frame = part1 + part2 + tail
print(zstd.decompress(frame) == b'hello zstd compressor')  # True
print(zstd.ZstdCompressor.CONTINUE == 0)                   # True
print(zstd.ZstdCompressor.FLUSH_BLOCK == 1)                # True
print(zstd.ZstdCompressor.FLUSH_FRAME == 2)                # True

# FLUSH_FRAME via compress(mode=...)
co2 = zstd.ZstdCompressor()
frame2 = co2.compress(b'all-at-once', zstd.ZstdCompressor.FLUSH_FRAME)
print(zstd.decompress(frame2) == b'all-at-once')           # True

# ===== ZstdDecompressor basic =====
comp = zstd.compress(b'streaming decompress test')
d = zstd.ZstdDecompressor()
print(d.decompress(comp) == b'streaming decompress test')  # True
print(d.eof)                                               # True
print(d.unused_data)                                       # b''

# chunked
d2 = zstd.ZstdDecompressor()
half = len(comp) // 2
p1 = d2.decompress(comp[:half])
p2 = d2.decompress(comp[half:])
print(p1 + p2 == b'streaming decompress test')             # True
print(d2.eof)                                              # True

# unused_data
d3 = zstd.ZstdDecompressor()
tr = comp + b'TRAILER'
print(d3.decompress(tr) == b'streaming decompress test')   # True
print(d3.eof)                                              # True
print(d3.unused_data == b'TRAILER')                        # True

# max_length
d4 = zstd.ZstdDecompressor()
first = d4.decompress(comp, max_length=5)
print(first == b'strea')                                   # True
rest = d4.decompress(b'')
print(first + rest == b'streaming decompress test')       # True

# ===== ZstdError on garbage =====
try:
    zstd.decompress(b'not a zstd frame at all!!!')
except zstd.ZstdError:
    print('ZstdError raised')                              # ZstdError raised

# ===== constants =====
print(zstd.COMPRESSION_LEVEL_DEFAULT == 3)                 # True
print(isinstance(zstd.zstd_version, str))                  # True
print(len(zstd.zstd_version_info) == 3)                    # True

# ===== ZstdDict accepted =====
zd = zstd.ZstdDict(b'samples-of-at-least-eight-bytes', is_raw=True)
print(isinstance(zd, zstd.ZstdDict))                       # True

# ===== ZstdFile =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, "out.zst")

    with zstd.ZstdFile(path, 'wb') as f:
        n = f.write(b'hello zstdfile')
        print(n)                                           # 14
        print(f.writable())                                # True
        print(f.readable())                                # False
        print(f.closed)                                    # False
    print(f.closed)                                        # True

    with zstd.ZstdFile(path, 'rb') as f:
        content = f.read()
        print(content)                                     # b'hello zstdfile'
        print(f.readable())                                # True
        print(f.writable())                                # False
        print(isinstance(f.name, str))                     # True
        print(f.mode == 'rb')                              # True

    # partial reads, seek, tell, peek
    with zstd.ZstdFile(path, 'wb') as f:
        f.write(b'abcdefghij')
    with zstd.ZstdFile(path, 'rb') as f:
        print(f.read(3))                                   # b'abc'
        print(f.tell())                                    # 3
        p = f.peek(2)
        print(p[:2])                                       # b'de'
        print(f.tell())                                    # 3
        print(f.read(3))                                   # b'def'
        f.seek(0)
        print(f.read(1))                                   # b'a'
        f.seek(-2, 2)
        print(f.read())                                    # b'ij'

    # readline
    lines_path = os.path.join(tmpdir, "lines.zst")
    with zstd.ZstdFile(lines_path, 'wb') as f:
        f.write(b'line1\nline2\nline3')
    with zstd.ZstdFile(lines_path, 'rb') as f:
        print(f.readline())                                # b'line1\n'
        print(f.readline())                                # b'line2\n'
        print(f.readline())                                # b'line3'
        print(f.readline())                                # b''

    # module-level open binary
    path2 = os.path.join(tmpdir, "mod.zst")
    with zstd.open(path2, 'wb') as f:
        f.write(b'via module open')
    with zstd.open(path2, 'rb') as f:
        print(f.read())                                    # b'via module open'

    # module-level open text
    path3 = os.path.join(tmpdir, "text.zst")
    with zstd.open(path3, 'wt') as f:
        f.write('hello text mode\n')
    with zstd.open(path3, 'rt') as f:
        print(f.read())                                    # hello text mode\n

print('done')

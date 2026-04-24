import zlib

# ===== Constants =====
print(zlib.Z_NO_COMPRESSION)        # 0
print(zlib.Z_BEST_SPEED)            # 1
print(zlib.Z_BEST_COMPRESSION)      # 9
print(zlib.Z_DEFAULT_COMPRESSION)   # -1
print(zlib.MAX_WBITS)               # 15
print(zlib.DEFLATED)                # 8
print(zlib.DEF_BUF_SIZE)            # 16384
print(zlib.DEF_MEM_LEVEL)           # 8
print(zlib.Z_DEFAULT_STRATEGY)      # 0
print(zlib.Z_FILTERED)              # 1
print(zlib.Z_HUFFMAN_ONLY)          # 2
print(zlib.Z_RLE)                   # 3
print(zlib.Z_FIXED)                 # 4
print(zlib.Z_NO_FLUSH)              # 0
print(zlib.Z_PARTIAL_FLUSH)         # 1
print(zlib.Z_SYNC_FLUSH)            # 2
print(zlib.Z_FULL_FLUSH)            # 3
print(zlib.Z_FINISH)                # 4
print(zlib.Z_BLOCK)                 # 5
print(zlib.Z_TREES)                 # 6

# ===== zlib.error =====
print(issubclass(zlib.error, Exception))   # True
print(issubclass(zlib.error, OSError))     # False

# ===== compress / decompress round-trip (zlib format) =====
data = b"hello world " * 100
compressed = zlib.compress(data)
print(isinstance(compressed, bytes))        # True
print(len(compressed) < len(data))         # True
print(zlib.decompress(compressed) == data) # True

# ===== compress with level =====
c1 = zlib.compress(data, 1)
c9 = zlib.compress(data, 9)
print(zlib.decompress(c1) == data)         # True
print(zlib.decompress(c9) == data)         # True
print(len(c1) >= len(c9))                  # True (lower level = less compression)

# ===== compress with level=0 (no compression) =====
c0 = zlib.compress(b"test", 0)
print(zlib.decompress(c0) == b"test")      # True

# ===== decompress bad data raises zlib.error =====
try:
    zlib.decompress(b"this is not compressed")
except zlib.error:
    print("zlib.error on bad data")        # zlib.error on bad data

# ===== crc32 =====
print(zlib.crc32(b""))                     # 0
print(zlib.crc32(b"hello"))                # 907060870
print(zlib.crc32(b"hello world"))          # 222957957
# chaining: crc32(a+b) == crc32(b, crc32(a))
combined = zlib.crc32(b" world", zlib.crc32(b"hello"))
print(combined == zlib.crc32(b"hello world"))   # True
# with explicit seed
print(zlib.crc32(b"abc", 0))               # 891568578

# ===== adler32 =====
print(zlib.adler32(b""))                   # 1
print(zlib.adler32(b"hello"))              # 103547413
print(zlib.adler32(b"hello world"))        # 436929629
# chaining: adler32(a+b) == adler32(b, adler32(a))
combined2 = zlib.adler32(b" world", zlib.adler32(b"hello"))
print(combined2 == zlib.adler32(b"hello world"))   # True
# with explicit seed=1 (same as default)
print(zlib.adler32(b"hello", 1) == zlib.adler32(b"hello"))  # True

# ===== compressobj basic =====
c = zlib.compressobj()
part1 = c.compress(b"hello ")
part2 = c.compress(b"world")
tail = c.flush()
combined3 = part1 + part2 + tail
print(zlib.decompress(combined3) == b"hello world")   # True

# ===== compressobj with level =====
c2 = zlib.compressobj(zlib.Z_BEST_COMPRESSION)
combined4 = c2.compress(b"repeat " * 50) + c2.flush()
print(zlib.decompress(combined4) == b"repeat " * 50)  # True

# ===== compressobj: empty compress calls =====
c3 = zlib.compressobj()
out = c3.compress(b"") + c3.compress(b"data") + c3.flush()
print(zlib.decompress(out) == b"data")     # True

# ===== compressobj Z_SYNC_FLUSH mid-stream =====
c4 = zlib.compressobj()
chunk1 = c4.compress(b"part1") + c4.flush(zlib.Z_SYNC_FLUSH)
chunk2 = c4.compress(b"part2") + c4.flush(zlib.Z_FINISH)
print(zlib.decompress(chunk1 + chunk2) == b"part1part2")  # True

# ===== decompressobj basic =====
raw = b"hello world " * 50
comp = zlib.compress(raw)
d = zlib.decompressobj()
out_d = d.decompress(comp) + d.flush()
print(out_d == raw)                        # True

# ===== decompressobj chunked input =====
d2 = zlib.decompressobj()
half = len(comp) // 2
out2 = d2.decompress(comp[:half])
out2 += d2.decompress(comp[half:])
out2 += d2.flush()
print(out2 == raw)                         # True

# ===== unused_data =====
extra = b"trailing bytes"
d3 = zlib.decompressobj()
d3.decompress(comp + extra)
d3.flush()
print(d3.unused_data == extra)             # True

# ===== decompressobj on empty data =====
d4 = zlib.decompressobj()
out4 = d4.decompress(zlib.compress(b"")) + d4.flush()
print(out4 == b"")                         # True

# ===== wbits: raw deflate (-15) =====
raw_comp = zlib.compress(b"raw deflate test", -1, -15)
print(zlib.decompress(raw_comp, -15) == b"raw deflate test")  # True

# ===== wbits: gzip format (MAX_WBITS + 16 = 31) =====
gzip_comp = zlib.compress(b"gzip test data", -1, 31)
print(zlib.decompress(gzip_comp, 31) == b"gzip test data")    # True

# ===== compressobj + decompressobj: raw deflate =====
c5 = zlib.compressobj(wbits=-15)
raw_stream = c5.compress(b"raw stream ") + c5.compress(b"content") + c5.flush()
d5 = zlib.decompressobj(wbits=-15)
out5 = d5.decompress(raw_stream) + d5.flush()
print(out5 == b"raw stream content")       # True

# ===== compressobj + decompressobj: gzip =====
c6 = zlib.compressobj(wbits=31)
gz_stream = c6.compress(b"gzip ") + c6.compress(b"stream") + c6.flush()
d6 = zlib.decompressobj(wbits=31)
out6 = d6.decompress(gz_stream) + d6.flush()
print(out6 == b"gzip stream")              # True

# ===== compressobj returns bytes =====
c7 = zlib.compressobj()
result = c7.compress(b"test")
print(isinstance(result, bytes))           # True
result2 = c7.flush()
print(isinstance(result2, bytes))          # True

# ===== decompressobj attributes =====
d7 = zlib.decompressobj()
print(isinstance(d7.unused_data, bytes))   # True
print(isinstance(d7.unconsumed_tail, bytes))  # True

# ===== crc32 result is int in range [0, 2**32) =====
v = zlib.crc32(b"test data")
print(isinstance(v, int))                  # True
print(0 <= v < 2**32)                      # True

# ===== adler32 result is int in range [0, 2**32) =====
v2 = zlib.adler32(b"test data")
print(isinstance(v2, int))                 # True
print(0 <= v2 < 2**32)                     # True

print('done')

import struct
import csv
import io
import urllib.parse as up
import zlib

# --- struct scenarios ------------------------------------------------------

# 1) calcsize over primitive chars.
print(struct.calcsize("<b"), struct.calcsize("<h"), struct.calcsize("<i"), struct.calcsize("<q"))

# 2) calcsize with leading repeat count.
print(struct.calcsize("<4i"), struct.calcsize("<10s"))

# 3) pack/unpack signed byte.
print(struct.unpack("<b", struct.pack("<b", -5)))

# 4) pack/unpack unsigned byte at the boundary.
print(struct.unpack("<B", struct.pack("<B", 255)))

# 5) pack/unpack short big-endian.
print(struct.unpack(">h", struct.pack(">h", -1)))

# 6) pack/unpack unsigned short little-endian.
print(struct.unpack("<H", struct.pack("<H", 65535)))

# 7) pack/unpack int32 extremes.
print(struct.unpack("<i", struct.pack("<i", -2147483648)))
print(struct.unpack("<i", struct.pack("<i", 2147483647)))

# 8) pack/unpack int64.
print(struct.unpack("<q", struct.pack("<q", -9000000000)))

# 9) pack/unpack uint64 above int64 max.
big = 0xFFFFFFFFFFFFFFFE
print(struct.unpack("<Q", struct.pack("<Q", big))[0] == big)

# 10) pack/unpack float with minor rounding.
val = struct.unpack("<f", struct.pack("<f", 1.5))[0]
print(val)

# 11) pack/unpack double preserves precision.
val = struct.unpack("<d", struct.pack("<d", 3.141592653589793))[0]
print(val)

# 12) '?' round-trips booleans (converts to 1/0 in unpack).
print(struct.unpack("<??", struct.pack("<??", True, False)))

# 13) 's' truncates longer input.
print(struct.pack("<3s", b"hello"))

# 14) 's' null-pads shorter input.
print(struct.pack("<5s", b"ab"))

# 15) 'c' single byte.
print(struct.pack("<c", b"A"))

# 16) Multiple fields together.
packed = struct.pack("<ihBf", 1000, -42, 200, 0.25)
print(struct.unpack("<ihBf", packed))

# 17) pack 'x' pad byte.
print(struct.pack("<bxb", 1, 2))

# 18) unpack_from honours offset.
buf = b"\x00\x00\x00" + struct.pack("<i", 99)
print(struct.unpack_from("<i", buf, 3))

# 19) calcsize with whitespace in format.
print(struct.calcsize("< i h b"))

# 20) Round-trip through bytearray.
data = bytearray(struct.pack("<I", 0xCAFEBABE))
print(struct.unpack("<I", bytes(data)))

# --- csv scenarios ---------------------------------------------------------

# 21) Empty reader input.
print(list(csv.reader([])))

# 22) Single-row reader.
print(list(csv.reader(["a,b"])))

# 23) Quoted field with comma.
print(list(csv.reader(['a,"b,c",d'])))

# 24) Empty fields are preserved.
print(list(csv.reader(["a,,c"])))

# 25) Writer with numeric cells converts to str.
buf = io.StringIO()
w = csv.writer(buf)
w.writerow([1, 2.5, "hi"])
print(buf.getvalue())

# 26) writerows writes multiple lines.
buf = io.StringIO()
w = csv.writer(buf)
w.writerows([["a", "b"], ["c", "d"]])
print(buf.getvalue())

# 27) excel-tab dialect delimits with tab.
buf = io.StringIO()
w = csv.writer(buf, "excel-tab")
w.writerow(["x", "y", "z"])
print(buf.getvalue())

# 28) Writer with custom delimiter.
buf = io.StringIO()
w = csv.writer(buf, delimiter=";")
w.writerow(["1", "2"])
print(buf.getvalue())

# 29) Reader iterates all rows in order.
for row in csv.reader(["a,b", "c,d"]):
    print(row)

# 30) Reader consumes multi-row input via list().
print(list(csv.reader(["1,2", "3,4", "5,6"])))

# 31) DictReader produces dicts keyed by header row.
rows = list(csv.DictReader(["k,v", "x,1", "y,2"]))
print([(r["k"], r["v"]) for r in rows])

# 32) DictReader with explicit fieldnames.
rows = list(csv.DictReader(["1,2", "3,4"], ["a", "b"]))
print([(r["a"], r["b"]) for r in rows])

# 33) DictWriter writeheader emits the header.
buf = io.StringIO()
dw = csv.DictWriter(buf, ["a", "b"])
dw.writeheader()
print(buf.getvalue())

# 34) DictWriter writerow writes a single dict row.
buf = io.StringIO()
dw = csv.DictWriter(buf, ["a", "b"])
dw.writerow({"a": 1, "b": 2})
print(buf.getvalue())

# 35) DictWriter writerows writes multiple dicts.
buf = io.StringIO()
dw = csv.DictWriter(buf, ["a", "b"])
dw.writerows([{"a": 1, "b": 2}, {"a": 3, "b": 4}])
print(buf.getvalue())

# 36) DictWriter fieldnames attribute round-trips.
dw = csv.DictWriter(io.StringIO(), ["x", "y", "z"])
print(dw.fieldnames)

# 37) DictWriter leaves missing keys empty.
buf = io.StringIO()
dw = csv.DictWriter(buf, ["a", "b", "c"])
dw.writerow({"a": 1, "c": 3})
print(buf.getvalue())

# 38) csv quoting constants are distinct.
print(csv.QUOTE_MINIMAL, csv.QUOTE_ALL, csv.QUOTE_NONNUMERIC, csv.QUOTE_NONE)

# --- urllib.parse scenarios ------------------------------------------------

# 39) urlparse extracts the common fields.
r = up.urlparse("https://user:pw@host.example:8080/path?q=1#f")
print(r.scheme, r.netloc, r.path, r.query, r.fragment)

# 40) hostname lowercases and strips userinfo/port.
print(r.hostname)

# 41) port is an int.
print(r.port, type(r.port).__name__)

# 42) Missing port yields None.
r2 = up.urlparse("https://example.com/")
print(r2.port)

# 43) urlparse on ftp scheme.
print(up.urlparse("ftp://mirror.example/x").scheme)

# 44) urlparse with fragment only.
print(up.urlparse("page#top").fragment)

# 45) urlparse with query only.
print(up.urlparse("?k=v").query)

# 46) urlsplit excludes params from the path split.
s = up.urlsplit("http://a/b;p?q#f")
print(s.path, s.query, s.fragment)

# 47) urlunparse round-trips through the 6-tuple.
round_trip = up.urlunparse(("https", "ex.com", "/a", "", "x=1", ""))
print(round_trip)

# 48) urlunsplit round-trips through the 5-tuple.
print(up.urlunsplit(("http", "a", "/b", "x=1", "f")))

# 49) urljoin resolves a relative path.
print(up.urljoin("https://a.com/b/c", "d"))

# 50) urljoin replaces with absolute path.
print(up.urljoin("https://a.com/b/c", "/d"))

# 51) urljoin on a full URL replaces the base.
print(up.urljoin("https://a.com/", "https://other.example/x"))

# 52) quote encodes spaces as %20 by default.
print(up.quote("hello world"))

# 53) quote leaves '/' alone by default.
print(up.quote("a/b/c"))

# 54) quote with empty safe encodes '/'.
print(up.quote("a/b", safe=""))

# 55) quote_plus uses '+' for space and '=' for equals.
print(up.quote_plus("x=1 y=2"))

# 56) unquote decodes %20.
print(up.unquote("one%20two"))

# 57) unquote_plus decodes '+' to space.
print(up.unquote_plus("one+two"))

# 58) urlencode on an insertion-ordered dict.
print(up.urlencode({"a": 1, "b": "hi"}))

# 59) urlencode on a list of 2-tuples preserves order.
print(up.urlencode([("z", 1), ("a", 2)]))

# 60) urlencode with doseq expands list values.
print(up.urlencode({"k": [1, 2, 3]}, doseq=True))

# 61) parse_qs groups duplicate keys.
print(up.parse_qs("a=1&a=2&b=3"))

# 62) parse_qsl preserves order and duplicates.
print(up.parse_qsl("a=1&b=2&a=3"))

# 63) URLParseResult supports integer indexing like a tuple.
r = up.urlparse("https://example.com/path?q=1#frag")
print(r[0], r[2], r[4], r[5])

# 64) Negative index into URLParseResult.
print(r[-1])

# 65) URLParseResult is iterable.
print(list(up.urlparse("http://a/b?q#f")))

# --- zlib scenarios --------------------------------------------------------

# 66) compress then decompress round-trips.
data = b"zlib stress test payload " * 40
c = zlib.compress(data)
print(zlib.decompress(c) == data)

# 67) Compressed output is strictly smaller on repetitive input.
print(len(c) < len(data))

# 68) compress level 0 still round-trips.
c0 = zlib.compress(data, 0)
print(zlib.decompress(c0) == data)

# 69) compress level 9 still round-trips.
c9 = zlib.compress(data, 9)
print(zlib.decompress(c9) == data)

# 70) compress empty input.
print(zlib.decompress(zlib.compress(b"")) == b"")

# 71) crc32 of empty is 0.
print(zlib.crc32(b""))

# 72) crc32 of "hello" matches the well-known value.
print(zlib.crc32(b"hello"))

# 73) crc32 with seed 0 equals the seedless call.
print(zlib.crc32(b"hello", 0) == zlib.crc32(b"hello"))

# 74) adler32 of empty is 1.
print(zlib.adler32(b""))

# 75) adler32 of "hello".
print(zlib.adler32(b"hello"))

# 76) Z_* and MAX_WBITS constants.
print(zlib.Z_NO_COMPRESSION, zlib.Z_BEST_SPEED, zlib.Z_BEST_COMPRESSION, zlib.Z_DEFAULT_COMPRESSION)
print(zlib.MAX_WBITS)

# 77) decompress on garbage raises.
try:
    zlib.decompress(b"not a real stream")
    print("no error")
except Exception:
    print("decompress raised")

# --- cross-module scenarios ------------------------------------------------

# 78) zlib round-trips the bytes from a struct.pack call.
packed = struct.pack("<10I", 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
print(zlib.decompress(zlib.compress(packed)) == packed)

# 79) urlencode + parse_qs survive a round-trip.
enc = up.urlencode({"a": "1", "b": "2"})
back = up.parse_qs(enc)
print(back["a"][0], back["b"][0])

# 80) A csv row carrying a urlencoded query string.
buf = io.StringIO()
w = csv.writer(buf)
w.writerow(["url", up.urlencode({"x": "a b"})])
print(buf.getvalue())

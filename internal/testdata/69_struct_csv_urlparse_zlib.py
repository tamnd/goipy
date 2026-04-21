import struct
import csv
import io
import urllib.parse
import zlib

# --- struct basics ---

print(struct.calcsize("<i"))
print(struct.calcsize(">ihb"))
print(struct.calcsize("5s"))

# Round trip an int.
data = struct.pack("<i", -123)
print(data)
print(struct.unpack("<i", data))

# Big-endian vs little-endian.
print(struct.pack(">H", 1))
print(struct.pack("<H", 1))

# Multiple fields.
packed = struct.pack("<ihB", 1000, -42, 255)
print(packed)
print(struct.unpack("<ihB", packed))

# Float + double.
print(struct.unpack("<f", struct.pack("<f", 1.5)))
print(struct.unpack("<d", struct.pack("<d", 3.14159)))

# Strings.
print(struct.pack("<5s", b"hello"))
print(struct.unpack("<5s", b"hello"))

# Unpack from offset.
buf = b"\x00\x00" + struct.pack("<I", 0xDEADBEEF)
print(struct.unpack_from("<I", buf, 2))

# --- csv basics ---

# Read rows from a list of lines.
rows = list(csv.reader(["a,b,c", "1,2,3", "x,y,z"]))
print(rows)

# Write via StringIO.
buf = io.StringIO()
w = csv.writer(buf)
w.writerow(["a", "b", "c"])
w.writerow([1, 2, 3])
print(buf.getvalue())

# writerows.
buf = io.StringIO()
w = csv.writer(buf)
w.writerows([["x", "y"], ["1", "2"], ["3", "4"]])
print(buf.getvalue())

# excel-tab dialect.
buf = io.StringIO()
w = csv.writer(buf, "excel-tab")
w.writerow(["a", "b"])
print(buf.getvalue())

# DictReader.
lines = ["name,age", "alice,30", "bob,25"]
for row in csv.DictReader(lines):
    print(row["name"], row["age"])

# DictWriter.
buf = io.StringIO()
w = csv.DictWriter(buf, ["name", "age"])
w.writeheader()
w.writerow({"name": "alice", "age": 30})
w.writerow({"name": "bob", "age": 25})
print(buf.getvalue())

# Constants.
print(csv.QUOTE_MINIMAL, csv.QUOTE_ALL, csv.QUOTE_NONNUMERIC, csv.QUOTE_NONE)

# --- urllib.parse basics ---

from urllib.parse import urlparse, urlunparse, urljoin, quote, unquote, quote_plus, unquote_plus, urlencode, parse_qs, parse_qsl

r = urlparse("https://user:pw@example.com:8080/a/b?x=1&y=2#frag")
print(r.scheme)
print(r.netloc)
print(r.path)
print(r.query)
print(r.fragment)
print(r.hostname)
print(r.port)

# Tuple access.
print(r[0], r[2])

# urlunparse round trip.
print(urlunparse(("https", "example.com", "/p", "", "q=1", "f")))

# urljoin.
print(urljoin("https://a.com/b/c", "d"))
print(urljoin("https://a.com/b/c", "/d"))

# quote / unquote.
print(quote("hello world/path"))
print(quote("hello world/path", safe=""))
print(unquote("hello%20world"))

# quote_plus / unquote_plus.
print(quote_plus("hello world&x=1"))
print(unquote_plus("hello+world"))

# urlencode.
print(urlencode({"a": 1, "b": "hi"}))
print(urlencode([("a", 1), ("b", 2)]))

# parse_qs / parse_qsl.
print(parse_qs("a=1&b=2&a=3"))
print(parse_qsl("a=1&b=2"))

# --- zlib basics ---

data = b"the quick brown fox jumps over the lazy dog" * 10
c = zlib.compress(data)
print(len(c) < len(data))
print(zlib.decompress(c) == data)

# Levels.
print(len(zlib.compress(data, 1)) >= len(zlib.compress(data, 9)) - 20)

# crc32 / adler32.
print(zlib.crc32(b"hello"))
print(zlib.adler32(b"hello"))
print(zlib.crc32(b"") == 0)

# Constants.
print(zlib.Z_BEST_SPEED, zlib.Z_BEST_COMPRESSION, zlib.Z_NO_COMPRESSION)

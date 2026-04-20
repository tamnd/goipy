# memoryview: zero-copy view over bytes / bytearray.

# Read-only view on bytes.
mv = memoryview(b"hello")
print(type(mv).__name__)
print(len(mv))
print(mv[0], mv[-1])
print(bytes(mv))
print(mv.tobytes())
print(mv.tolist())
print(mv.readonly)
print(mv.nbytes, mv.format, mv.itemsize)
print(list(mv))

# Slice produces another memoryview sharing the backing buffer.
sub = mv[1:4]
print(bytes(sub))
print(sub.readonly)

# Equality with bytes / bytearray / memoryview.
print(memoryview(b"ab") == b"ab")
print(b"ab" == memoryview(b"ab"))
print(memoryview(bytearray(b"ab")) == memoryview(b"ab"))

# Writable view over bytearray mutates the backing buffer.
ba = bytearray(b"abc")
mv2 = memoryview(ba)
print(mv2.readonly)
mv2[0] = 90
print(ba)
mv2[1:3] = b"YZ"
print(ba)

# Slice of a writable view shares memory.
mv3 = memoryview(bytearray(b"ABCDE"))
sub2 = mv3[1:4]
sub2[0] = ord("z")
print(mv3.tobytes())

# Writing to a read-only view raises TypeError.
try:
    memoryview(b"abc")[0] = 1
except TypeError:
    print("readonly rejects assignment")

# Membership and iteration.
print(104 in memoryview(b"hello"))
print(99 in memoryview(b"hello"))
print(sum(memoryview(b"\x01\x02\x03")))

# isinstance.
print(isinstance(mv, memoryview))
print(isinstance(b"", memoryview))

# release() is a no-op but returns None and is callable.
v = memoryview(b"xyz")
print(v.release())

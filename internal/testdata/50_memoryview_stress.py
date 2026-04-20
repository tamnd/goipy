# memoryview stress: nested slices, write-through, edge cases.

ba = bytearray(b"0123456789")
mv = memoryview(ba)

# Nested slicing shares the same backing buffer.
outer = mv[2:8]
inner = outer[1:4]
print(bytes(outer), bytes(inner))
inner[0] = ord("X")
print(bytes(mv), bytes(outer), bytes(inner))

# Negative indices in slices and integer access.
print(bytes(mv[-4:-1]))
print(mv[-1])

# Slice assignment with wrong size raises ValueError.
try:
    mv[0:2] = b"ABC"
except ValueError:
    print("size mismatch")

# enumerate over memoryview (iterates ints).
print(list(enumerate(memoryview(b"abc"))))

# Equality and inequality across types.
print(memoryview(b"ab") != b"ac")
print(memoryview(b"ab") != memoryview(b"ab"))

# Slice of a read-only view is also read-only.
v = memoryview(b"abc")[1:]
print(v.readonly)
try:
    v[0] = 99
except TypeError:
    print("slice of bytes view readonly")

# tolist and nbytes.
print(memoryview(b"AB").tolist())
print(memoryview(bytearray(b"12345")).nbytes)

# Empty view.
empty = memoryview(b"")
print(len(empty), bool(empty), empty.tobytes(), empty.tolist())

# Ordering comparisons are not defined on memoryview.
try:
    memoryview(b"a") < memoryview(b"b")
except TypeError:
    print("no ordering on memoryview")

# sum over a memoryview yields an int (iterator yields ints).
print(sum(memoryview(b"\x01\x02\x03\x04")))

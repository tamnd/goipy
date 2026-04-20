# bytearray stress: slice assignment, mutation methods, search, compare.

ba = bytearray(b"Hello, World!")

# Slice assignment (same length).
ba[0:5] = b"HOWDY"
print(ba)

# Slice assignment with shrink and grow.
ba[0:5] = b"Hi"
print(ba)
ba[2:2] = b"-there"
print(ba)

# Slice deletion.
del ba[0:3]
print(ba)

# Extended slicing via read-only access.
ba2 = bytearray(b"abcdef")
print(ba2[::2])
print(ba2[::-1])

# clear / reverse / insert / remove.
ba3 = bytearray(b"abcdef")
ba3.reverse()
print(ba3)
ba3.clear()
print(len(ba3))
ba4 = bytearray(b"abc")
ba4.insert(1, ord("Z"))
print(ba4)
ba4.remove(ord("Z"))
print(ba4)

# count / find / index / startswith / endswith.
ba5 = bytearray(b"ababcab")
print(ba5.count(b"ab"))
print(ba5.find(b"bc"))
print(ba5.find(b"zz"))
print(ba5.index(b"ab"))
print(ba5.startswith(b"ab"))
print(ba5.endswith(b"ab"))
print(ba5.startswith(b"zz"))

# replace / split.
print(ba5.replace(b"ab", b"X"))
print(bytearray(b"a,b,c").split(b","))

# Lex comparison across bytearray/bytes.
print(bytearray(b"abc") < bytearray(b"abd"))
print(bytearray(b"abc") < b"abd")
print(bytearray(b"abc") <= bytearray(b"abc"))

# Unhashable: bytearray can't be a dict key.
try:
    hash(bytearray(b"a"))
except TypeError:
    print("unhashable")

# In-place concatenation preserves content.
ba6 = bytearray(b"a")
ba6 += b"bc"
print(ba6)

# remove on missing byte raises ValueError.
try:
    bytearray(b"abc").remove(ord("z"))
except ValueError:
    print("remove missing raises")

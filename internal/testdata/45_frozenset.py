# frozenset: distinct hashable set type.

fs = frozenset([1, 2, 3])
print(fs)
print(type(fs).__name__)
print(len(fs))
print(frozenset())
print(frozenset([1, 1, 2, 2, 3]))
print(2 in fs, 9 in fs)

# Set algebra
a = frozenset([1, 2, 3])
b = frozenset([2, 3, 4])
print(a | b)
print(a & b)
print(a - b)
print(a ^ b)

# Equality across set / frozenset
print(frozenset([1, 2]) == {1, 2})
print({1, 2} == frozenset([1, 2]))
print(frozenset([1, 2]) == frozenset([2, 1]))
print(frozenset([1]) == frozenset([1, 2]))

# Hashable: usable as dict key
d = {frozenset([1, 2]): "a", frozenset([3]): "b"}
print(len(d))
print(d[frozenset([2, 1])])

# Nesting
meta = frozenset([frozenset([1]), frozenset([1, 2])])
print(len(meta))

# Truthiness
print(bool(frozenset()))
print(bool(frozenset([0])))

# Plain set is unhashable
try:
    hash({1, 2})
except TypeError:
    print("set unhashable")

# Iteration
total = 0
for x in frozenset([10, 20, 30]):
    total += x
print(total)

# isinstance discrimination
print(isinstance(frozenset(), frozenset))
print(isinstance(frozenset(), set))
print(isinstance(set(), frozenset))

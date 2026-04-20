# frozenset stress: subset/superset operators, methods, hashing, mixed ops.

a = frozenset([1, 2, 3])
b = frozenset([2, 3])
c = frozenset([4, 5])

# Subset / superset via comparison operators (partial order).
print(b < a, b <= a, a < a, a <= a, a > b, a >= b, a >= a)
print(b.issubset(a), a.issuperset(b), a.isdisjoint(c), a.isdisjoint(b))

# union / intersection / difference / symmetric_difference with iterables.
print(a.union([5, 6], (7,)))
print(a.intersection([2, 3, 99]))
print(a.difference({2}, [3]))
print(a.symmetric_difference([3, 4]))

# frozenset.copy() returns the same object (immutable).
fs = frozenset([1, 2])
print(fs.copy() is fs)
s = {1, 2}
print(s.copy() is s)

# Hash is stable across insertion order.
print(hash(frozenset([1, 2, 3])) == hash(frozenset([3, 2, 1])))

# Mixed operand: result type follows the left operand.
print(type({1} | frozenset([2])).__name__)
print(type(frozenset([1]) | {2}).__name__)

# set <= frozenset and frozenset <= set both work.
print({1} <= frozenset([1, 2]))
print(frozenset([1]) <= {1, 2})

# Nested frozensets in a set: equal frozensets collapse.
outer = {frozenset([1, 2]), frozenset([1, 2]), frozenset([3])}
print(len(outer))

# Empty frozenset is a subset of everything but not a proper subset of itself.
print(frozenset() <= frozenset([1]))
print(frozenset() < frozenset())

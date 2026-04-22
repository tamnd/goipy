# Set operations from the Python 3.13+ thread safety docs.

s = {1, 2, 3, 4, 5}

# --- Lock-free reads ---
print(3 in s)            # True
print(len(s))            # 5

# --- Locked single-element ---
s.add(6)
print(6 in s)            # True

s.discard(99)            # no error even if missing
s.discard(6)
print(6 in s)            # False

s2 = {1, 2, 3}
s2.remove(2)
print(2 in s2)           # False

p = s2.pop()
print(isinstance(p, int))  # True
print(len(s2))           # 1

# --- copy / clear ---
s3 = {10, 20, 30}
s4 = s3.copy()
print(sorted(s4))        # [10, 20, 30]
s3.clear()
print(len(s3))           # 0

# --- Operator forms ---
a = {1, 2, 3}
b = {3, 4, 5}
print(sorted(a | b))     # [1, 2, 3, 4, 5]
print(sorted(a & b))     # [3]
print(sorted(a - b))     # [1, 2]
print(sorted(a ^ b))     # [1, 2, 4, 5]

a2 = {1, 2, 3}
a2 |= {4}
print(4 in a2)           # True
a2 &= {1, 2, 4}
print(sorted(a2))        # [1, 2, 4]
a2 -= {2}
print(sorted(a2))        # [1, 4]
a2 ^= {1, 5}
print(sorted(a2))        # [4, 5]

# --- Multi-arg methods ---
x = {1, 2}
print(sorted(x.union({3}, {4})))           # [1, 2, 3, 4]
print(sorted(x.intersection({1, 3}, {1, 4})))  # [1]
print(sorted(x.difference({2}, {3})))     # [1]

y = {1, 2, 3}
y.update({4}, {5})
print(sorted(y))         # [1, 2, 3, 4, 5]

z = {1, 2, 3, 4}
z.intersection_update({1, 2, 5})
print(sorted(z))         # [1, 2]

w = {1, 2, 3}
w.difference_update({2, 3})
print(sorted(w))         # [1]

v = {1, 2, 3}
v.symmetric_difference_update({2, 4})
print(sorted(v))         # [1, 3, 4]

# symmetric_difference
print(sorted({1,2}.symmetric_difference({2,3})))  # [1, 3]

# --- Predicates ---
print({1, 2}.isdisjoint({3, 4}))   # True
print({1, 2}.issubset({1, 2, 3}))  # True
print({1, 2, 3}.issuperset({2}))   # True

# --- frozenset read operations ---
fs = frozenset([1, 2, 3])
print(2 in fs)           # True
print(len(fs))           # 3
print(sorted(fs))        # [1, 2, 3]
print(sorted(fs | {4}))  # [1, 2, 3, 4]
print(sorted(fs & {2, 3, 5}))  # [2, 3]
print(sorted(fs - {1}))  # [2, 3]

# --- Safe pattern ---
shared = {1, 2, 3}
shared.discard(99)       # no KeyError
shared.discard(2)
print(sorted(shared))    # [1, 3]

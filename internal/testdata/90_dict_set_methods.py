"""Tests for dict and set new methods."""

# --- dict.fromkeys ---
d = dict.fromkeys(["a", "b", "c"])
print(sorted(d.keys()))
print(all(v is None for v in d.values()))

d2 = dict.fromkeys(["x", "y"], 0)
print(d2["x"])
print(d2["y"])

# --- dict.popitem ---
d3 = {"a": 1}
k, v = d3.popitem()
print(k, v)
print(len(d3))

# --- dict __or__ / __ior__ ---
a = {"x": 1}
b = {"y": 2}
c = a | b
print(sorted(c.items()))

a |= {"z": 3}
print(sorted(a.items()))

# --- set.discard ---
s = {1, 2, 3}
s.discard(2)
print(sorted(s))
s.discard(99)
print(sorted(s))

# --- set.remove ---
s2 = {1, 2, 3}
s2.remove(2)
print(sorted(s2))
try:
    s2.remove(99)
except KeyError:
    print("KeyError")

# --- set.pop ---
s3 = {42}
v = s3.pop()
print(v)
print(len(s3))
try:
    s3.pop()
except KeyError:
    print("KeyError")

# --- set.clear ---
s4 = {1, 2, 3}
s4.clear()
print(len(s4))

# --- set.copy ---
s5 = {1, 2, 3}
s6 = s5.copy()
s6.add(4)
print(sorted(s5))
print(sorted(s6))

# --- set.update ---
s7 = {1, 2}
s7.update([3, 4], {5})
print(sorted(s7))

# --- set.intersection_update ---
s8 = {1, 2, 3, 4}
s8.intersection_update({2, 3, 5})
print(sorted(s8))

# --- set.difference_update ---
s9 = {1, 2, 3, 4}
s9.difference_update({2, 4})
print(sorted(s9))

# --- set.symmetric_difference_update ---
s10 = {1, 2, 3}
s10.symmetric_difference_update({2, 3, 4})
print(sorted(s10))

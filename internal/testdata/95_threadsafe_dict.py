# Dict operations from the Python 3.13+ thread safety docs.

d = {"a": 1, "b": 2, "c": 3}

# --- Lock-free reads ---
print(d["a"])            # 1
print(d.get("b"))        # 2
print(d.get("z", 0))     # 0
print("c" in d)          # True
print(len(d))            # 3

# --- Locked writes ---
d["d"] = 4
print(d["d"])            # 4
del d["d"]
print("d" in d)          # False

# pop
x = d.pop("c")
print(x)                 # 3
print("c" in d)          # False

# popitem (last inserted)
d2 = {"x": 10, "y": 20}
k, v = d2.popitem()
print(k, v)              # y 20

# setdefault
d3 = {"a": 1}
print(d3.setdefault("a", 99))   # 1
print(d3.setdefault("b", 42))   # 42
print(d3["b"])                   # 42

# --- New objects ---
d4 = {"a": 1, "b": 2}
c4 = d4.copy()
print(c4)                # {'a': 1, 'b': 2}

d5 = {"a": 1}
d6 = {"b": 2}
merged = d5 | d6
print(merged)            # {'a': 1, 'b': 2}

print(list(d4.keys()))   # ['a', 'b']
print(list(d4.values())) # [1, 2]
print(list(d4.items()))  # [('a', 1), ('b', 2)]

# --- clear / update / |= / == ---
d7 = {"a": 1, "b": 2}
d7.update({"c": 3})
print(d7)                # {'a': 1, 'b': 2, 'c': 3}
d7 |= {"d": 4}
print(d7["d"])           # 4
print({"a": 1} == {"a": 1})  # True
d7.clear()
print(d7)                # {}

# --- fromkeys ---
keys = ["x", "y", "z"]
fd = dict.fromkeys(keys, 0)
print(fd)                # {'x': 0, 'y': 0, 'z': 0}

fd2 = dict.fromkeys({"p", "q"}, None)
print(sorted(fd2.keys()))  # ['p', 'q']

fd3 = dict.fromkeys(frozenset(["m", "n"]), True)
print(sorted(fd3.keys()))  # ['m', 'n']

# --- Safe patterns ---
# pop-with-default instead of check-then-delete
shared = {"a": 1, "b": 2}
val = shared.pop("a", None)
print(val)               # 1
val2 = shared.pop("z", None)
print(val2)              # None

# copy-before-iterate
data = {"x": 10, "y": 20, "z": 30}
snapshot = data.copy()
total = sum(snapshot.values())
print(total)             # 60

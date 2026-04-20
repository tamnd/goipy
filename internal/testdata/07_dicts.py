d = {"a": 1, "b": 2, "c": 3}
print(len(d))
print(d["a"])
d["d"] = 4
print(sorted(d.keys()))
print(sorted(d.values()))
print(sorted(d.items()))
print("a" in d)
print("z" not in d)
print(d.get("a"))
print(d.get("missing", -1))
del d["a"]
print(sorted(d.keys()))
print({k: v * 2 for k, v in [("x", 1), ("y", 2)]})

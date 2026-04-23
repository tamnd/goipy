import copy

# ===== copy.copy — shallow copy =====

# scalars / immutables: same object returned (identity)
print(copy.copy(42) is 42)          # True (small int cache)
print(copy.copy('hello') == 'hello') # True

# list shallow copy
a = [1, [2, 3], 4]
b = copy.copy(a)
print(b == a)                       # True
print(b is a)                       # False
b[0] = 99
print(a[0])                         # 1  (independent top-level)
b[1].append(99)
print(a[1])                         # [2, 3, 99]  (shared inner list)

# dict shallow copy
d = {'x': [1, 2], 'y': 3}
d2 = copy.copy(d)
print(d2 == d)                      # True
print(d2 is d)                      # False
d['x'].append(9)
print(d2['x'])                      # [1, 2, 9]  (shared inner list)

# set shallow copy
s = {1, 2, 3}
s2 = copy.copy(s)
print(s2 == s)                      # True
print(s2 is s)                      # False

# tuple shallow copy (tuples are immutable so same object is ok)
t = (1, 2, 3)
t2 = copy.copy(t)
print(t2 == t)                      # True

# frozenset
fs = frozenset([1, 2, 3])
fs2 = copy.copy(fs)
print(fs2 == fs)                    # True

# bytes
by = b'hello'
by2 = copy.copy(by)
print(by2 == by)                    # True

# ===== __copy__ protocol =====
class WithCopy:
    def __init__(self, val):
        self.val = val
    def __copy__(self):
        return WithCopy(self.val * 2)

wc = WithCopy(5)
wc2 = copy.copy(wc)
print(wc2.val)                      # 10

# ===== copy.deepcopy — deep copy =====

# list deep copy
a = [1, [2, 3], 4]
c = copy.deepcopy(a)
print(c == a)                       # True
print(c is a)                       # False
c[1].append(99)
print(a[1])                         # [2, 3]  (not affected)

# dict deep copy
d = {'x': [1, 2], 'y': 3}
d3 = copy.deepcopy(d)
d['x'].append(9)
print(d3['x'])                      # [1, 2]

# nested dict
nested = {'a': {'b': [1, 2]}}
nested2 = copy.deepcopy(nested)
nested['a']['b'].append(3)
print(nested2['a']['b'])            # [1, 2]

# cyclic reference (deepcopy must handle it)
lst = [1, 2]
lst.append(lst)
lst2 = copy.deepcopy(lst)
print(lst2[0])                      # 1
print(lst2[1])                      # 2
print(lst2[2] is lst2)              # True  (cycle preserved)

# ===== __deepcopy__ protocol =====
class WithDeepCopy:
    def __init__(self, val):
        self.val = val
    def __deepcopy__(self, memo):
        return WithDeepCopy(self.val + 100)

wd = WithDeepCopy(7)
wd2 = copy.deepcopy(wd)
print(wd2.val)                      # 107

# ===== deepcopy of instance =====
class Point:
    def __init__(self, x, y):
        self.x = x
        self.y = y

p = Point(3, 4)
p2 = copy.deepcopy(p)
print(p2.x)                         # 3
print(p2.y)                         # 4
p.x = 99
print(p2.x)                         # 3 (independent)

# ===== copy.copy of instance (shallow) =====
p3 = Point(10, 20)
p3.data = [1, 2, 3]
p4 = copy.copy(p3)
print(p4.x)                         # 10
p3.data.append(99)
print(p4.data)                      # [1, 2, 99]  (shared)

# ===== copy.error =====
print(copy.error is copy.Error)     # True

# ===== copy.replace (Python 3.13+) =====
# Works with __replace__ protocol
class Config:
    def __init__(self, host, port, debug=False):
        self.host = host
        self.port = port
        self.debug = debug
    def __replace__(self, **changes):
        new = Config(self.host, self.port, self.debug)
        for k, v in changes.items():
            setattr(new, k, v)
        return new

cfg = Config('localhost', 8080)
cfg2 = copy.replace(cfg, port=9090)
print(cfg2.host)                    # localhost
print(cfg2.port)                    # 9090
print(cfg.port)                     # 8080  (original unchanged)

print('done')

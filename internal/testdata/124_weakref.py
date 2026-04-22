import weakref

# ===== setup: weakly-referenceable class =====
class Node:
    def __init__(self, val):
        self.val = val
    def double(self):
        return self.val * 2
    def __repr__(self):
        return f'Node({self.val})'

class Callable:
    def __call__(self, x):
        return x + 10

# ===== weakref.ref basics =====
n = Node(42)
r = weakref.ref(n)

# type name
print(type(r).__name__)        # ReferenceType

# calling returns the object
print(r() is n)                # True
print(r().val)                 # 42
print(r().double())            # 84

# ===== canonical ref (no callback → same ref object) =====
r2 = weakref.ref(n)
print(r is r2)                 # True  — canonical dedup

# with different callbacks → distinct refs
r3 = weakref.ref(n, lambda ref: None)
print(r is r3)                 # False

# ===== __callback__ attribute =====
print(r.__callback__ is None)  # True  (no callback)
print(r3.__callback__ is not None)  # True

# ===== isinstance checks =====
print(isinstance(r, weakref.ref))             # True
print(isinstance(r, weakref.ReferenceType))   # True
print(isinstance(n, weakref.ref))             # False

# ===== getweakrefcount =====
n2 = Node(99)
ra = weakref.ref(n2)              # one canonical ref
rb = weakref.ref(n2, lambda r: None)  # second with callback
print(weakref.getweakrefcount(n2))  # 2

# ===== getweakrefs =====
refs = weakref.getweakrefs(n2)
print(len(refs))                   # 2
print(all(isinstance(x, weakref.ref) for x in refs))  # True

# ===== TypeError for non-weakref-able types =====
for bad in [1, 1.5, "hello", b"bytes", [1, 2], (1,), {1: 2}]:
    try:
        weakref.ref(bad)
        print("no error for", type(bad).__name__)
    except TypeError:
        print("TypeError:", type(bad).__name__)

# ===== proxy basics =====
n3 = Node(7)
p = weakref.proxy(n3)
print(type(p).__name__)           # ProxyType
print(p.val)                      # 7
print(p.double())                 # 14

# proxy for callable object
c = Callable()
cp = weakref.proxy(c)
print(type(cp).__name__)          # CallableProxyType

# ===== ProxyType / CallableProxyType / ProxyTypes =====
print(isinstance(p, weakref.ProxyType))          # True
print(isinstance(p, weakref.CallableProxyType))  # False
print(isinstance(cp, weakref.CallableProxyType)) # True
print(len(weakref.ProxyTypes))                   # 2

# ===== ReferenceType =====
print(weakref.ReferenceType)                     # <class 'weakref.ReferenceType'>

# ===== WeakValueDictionary =====
wvd = weakref.WeakValueDictionary()
a = Node(1)
b = Node(2)
c2 = Node(3)

wvd['a'] = a
wvd['b'] = b
wvd['c'] = c2
print(len(wvd))                   # 3
print('a' in wvd)                 # True
print('z' in wvd)                 # False

print(wvd['a'].val)               # 1
print(wvd.get('b').val)           # 2
print(wvd.get('z', None))         # None

print(sorted(wvd.keys()))         # ['a', 'b', 'c']
print(sorted(v.val for v in wvd.values()))  # [1, 2, 3]
print(sorted((k, v.val) for k, v in wvd.items()))  # [('a',1),('b',2),('c',3)]

wvd.pop('a')
print(len(wvd))                   # 2

nd = Node(4)
wvd.setdefault('d', nd)
print(wvd['d'].val)               # 4
print(len(wvd))                   # 3

n4 = Node(10)
wvd.update({'x': n4})
print(wvd['x'].val)               # 10

del wvd['b']
print(len(wvd))                   # 3

wvd.clear()
print(len(wvd))                   # 0

# construction from dict
e = Node(5)
f2 = Node(6)
wvd2 = weakref.WeakValueDictionary({'e': e, 'f': f2})
print(len(wvd2))                  # 2
print(sorted(wvd2.keys()))        # ['e', 'f']

# ===== WeakKeyDictionary =====
wkd = weakref.WeakKeyDictionary()
k1 = Node(100)
k2 = Node(200)
wkd[k1] = 'alpha'
wkd[k2] = 'beta'
print(len(wkd))                   # 2
print(k1 in wkd)                  # True
print(wkd[k1])                    # alpha
print(sorted(wkd.values()))       # ['alpha', 'beta']

wkd.pop(k1)
print(len(wkd))                   # 1

wkd[k1] = 'gamma'
wkd.update({k2: 'delta'})
print(sorted(wkd.values()))       # ['delta', 'gamma']

wkd.clear()
print(len(wkd))                   # 0

# ===== WeakSet =====
ws = weakref.WeakSet()
s1 = Node(10)
s2 = Node(20)
s3 = Node(30)

ws.add(s1)
ws.add(s2)
ws.add(s3)
ws.add(s1)                        # duplicate → no effect
print(len(ws))                    # 3
print(s1 in ws)                   # True
print(s3 in ws)                   # True

ws.discard(s2)
print(s2 in ws)                   # False
print(len(ws))                    # 2

ws.remove(s3)
print(len(ws))                    # 1

ws.add(s2)
ws.add(s3)
item = ws.pop()
print(item in [s1, s2, s3])      # True
print(len(ws))                    # 2

ws.clear()
print(len(ws))                    # 0

# WeakSet from iterable
ws2 = weakref.WeakSet([s1, s2, s3])
print(len(ws2))                   # 3
print(s1 in ws2)                  # True

# iterate WeakSet
count = sum(1 for _ in ws2)
print(count)                      # 3

# ===== finalize =====
log = []
obj = Node(99)
fin = weakref.finalize(obj, log.append, 'cleaned')

print(fin.alive)                  # True
print(type(fin).__name__)         # finalize

# manual call
result = fin()
print(log)                        # ['cleaned']
print(fin.alive)                  # False

# second call is no-op
fin()
print(log)                        # ['cleaned']  (not called again)

# finalize with multiple args
log2 = []
obj2 = Node(88)
fin2 = weakref.finalize(obj2, lambda a, b, c: log2.append((a, b, c)), 1, 2, 3)
fin2()
print(log2)                       # [(1, 2, 3)]

# atexit attribute
print(fin2.alive)                 # False
obj3 = Node(77)
fin3 = weakref.finalize(obj3, log.append, 'x')
print(fin3.atexit)                # True
fin3.atexit = False
print(fin3.atexit)                # False
fin3()
print(log)                        # ['cleaned', 'x']

# ===== WeakMethod =====
class Calculator:
    def __init__(self, base):
        self.base = base
    def add(self, x):
        return self.base + x

calc = Calculator(10)
wm = weakref.WeakMethod(calc.add)
print(type(wm).__name__)          # WeakMethod
bound = wm()
print(bound(5))                   # 15
print(bound(20))                  # 30

print('done')

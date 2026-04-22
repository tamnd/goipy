from collections import (deque, Counter, defaultdict, OrderedDict,
                         namedtuple, ChainMap, UserDict, UserList, UserString)

# ===== deque extras =====
d = deque([1, 2, 3, 4, 5])
d2 = d.copy()
print(list(d2))                # [1, 2, 3, 4, 5]
d.insert(2, 99)
print(list(d))                 # [1, 2, 99, 3, 4, 5]
d.remove(99)
print(list(d))                 # [1, 2, 3, 4, 5]
# index with start/stop
print(d.index(3))              # 2
print(d.index(3, 1, 4))       # 2
# reversed
print(list(reversed(d)))       # [5, 4, 3, 2, 1]
# setitem / delitem
d[0] = 10
print(d[0])                    # 10
del d[0]
print(list(d))                 # [2, 3, 4, 5]
# in operator
print(3 in d)                  # True
print(99 in d)                 # False

# ===== Counter extras =====
c1 = Counter({'a': 3, 'b': 2})
c2 = Counter({'a': 1, 'b': 2, 'c': 1})
print(sorted((c1 + c2).items()))  # [('a', 4), ('b', 4), ('c', 1)]
print(sorted((c1 - c2).items()))  # [('a', 2)]
print(sorted((c1 & c2).items()))  # [('a', 1), ('b', 2)]
print(sorted((c1 | c2).items()))  # [('a', 3), ('b', 2), ('c', 1)]
c3 = Counter(a=3, b=-2)
print(sorted((+c3).items()))       # [('a', 3)]
print(sorted((-c3).items()))       # [('b', 2)]
c4 = c1.copy()
print(sorted(c4.items()))          # [('a', 3), ('b', 2)]
try:
    Counter.fromkeys([1, 2])
except NotImplementedError:
    print("Counter.fromkeys raises NotImplementedError")

# ===== defaultdict extras =====
dd = defaultdict(list, {'x': [1, 2]})
dd2 = dd.copy()
print(dd2['x'])                # [1, 2]
print(type(dd).__name__)       # defaultdict

# ===== OrderedDict extras =====
od = OrderedDict([('a', 1), ('b', 2), ('c', 3)])
print(list(reversed(od)))      # ['c', 'b', 'a']
od2 = od.copy()
print(list(od2.keys()))        # ['a', 'b', 'c']
od3 = OrderedDict.fromkeys(['x', 'y'], 0)
print(list(od3.items()))       # [('x', 0), ('y', 0)]
od4 = od | od3
print(type(od4).__name__)      # OrderedDict
print(list(od4.keys()))        # ['a', 'b', 'c', 'x', 'y']

# ===== namedtuple extras =====
Point = namedtuple('Point', ['x', 'y'], defaults=[0, 0])
print(Point._field_defaults)   # {'x': 0, 'y': 0}
p = Point._make([1, 2])
print(p.x, p.y)                # 1 2
print(p.count(1))              # 1
print(p.index(2))              # 1
# hash: should not raise and should be consistent
h1 = hash(p)
h2 = hash(Point(1, 2))
print(h1 == h2)                # True

# ===== ChainMap =====
cm = ChainMap({'a': 1, 'b': 2}, {'b': 3, 'c': 4})
print(cm['a'])                 # 1
print(cm['b'])                 # 2  (first map wins)
print(cm['c'])                 # 4
print(len(cm))                 # 3  (unique keys)
print(sorted(cm.keys()))       # ['a', 'b', 'c']
print(sorted(cm.values()))     # [1, 2, 4]
print('a' in cm)               # True
print('z' in cm)               # False
cm['d'] = 5                    # writes to first map
print(cm.maps[0]['d'])         # 5
del cm['b']                    # deletes from first map
print(cm['b'])                 # 3  (now falls through to second map)
print(len(cm.maps))            # 2
child = cm.new_child({'e': 6})
print(child['e'])              # 6
print(child['c'])              # 4
print(len(child.parents.maps)) # 2
print(type(cm).__name__)       # ChainMap
print(cm.get('a'))             # 1
print(cm.get('z', 99))         # 99

# ===== UserDict =====
ud = UserDict({'a': 1, 'b': 2})
print(ud['a'])                 # 1
ud['c'] = 3
print(ud['c'])                 # 3
print('a' in ud)               # True
del ud['a']
print('a' in ud)               # False
print(len(ud))                 # 2
print(sorted(ud.keys()))       # ['b', 'c']
print(sorted(ud.values()))     # [2, 3]
print(sorted(ud.items()))      # [('b', 2), ('c', 3)]
ud2 = ud.copy()
print(sorted(ud2.keys()))      # ['b', 'c']
print(type(ud).__name__)       # UserDict
print(sorted(ud.data.items())) # [('b', 2), ('c', 3)]
ud.update({'d': 4})
print(sorted(ud.keys()))       # ['b', 'c', 'd']
print(ud.get('b'))             # 2
print(ud.get('z', 0))         # 0
print(ud.pop('c'))             # 3
print(sorted(ud.keys()))       # ['b', 'd']
ud.setdefault('e', 5)
print(ud['e'])                 # 5
ud.clear()
print(len(ud))                 # 0

# ===== UserList =====
ul = UserList([1, 2, 3])
print(ul[0])                   # 1
ul.append(4)
print(list(ul))                # [1, 2, 3, 4]
ul[0] = 10
print(ul[0])                   # 10
del ul[0]
print(list(ul))                # [2, 3, 4]
print(len(ul))                 # 3
print(2 in ul)                 # True
ul2 = UserList([5, 6])
ul3 = ul + ul2
print(list(ul3))               # [2, 3, 4, 5, 6]
print(type(ul).__name__)       # UserList
ul.sort()
print(ul.data)                 # [2, 3, 4]
print(ul.count(2))             # 1
print(ul.index(3))             # 1
ul.reverse()
print(ul.data)                 # [4, 3, 2]
ul.insert(0, 0)
print(ul.data)                 # [0, 4, 3, 2]
ul.remove(0)
print(ul.data)                 # [4, 3, 2]
ul4 = ul.copy()
print(ul4.data)                # [4, 3, 2]
ul.extend([9, 8])
print(ul.data)                 # [4, 3, 2, 9, 8]
ul.pop()
print(ul.data)                 # [4, 3, 2, 9]
ul.clear()
print(ul.data)                 # []

# ===== UserString =====
us = UserString("hello")
print(us.upper())              # HELLO
print(us.lower())              # hello
print(str(us) + " world")      # hello world
print(us[0])                   # h
print(us[1:3])                 # el
print(len(us))                 # 5
print('h' in us)               # True
print('z' in us)               # False
us2 = UserString("world")
print(us < us2)                # True
print(us == UserString("hello")) # True
print(type(us).__name__)       # UserString
print(us.data)                 # hello
print(us.strip())              # hello
print(UserString("  hi  ").strip()) # hi
print(us.replace("l", "L"))    # heLLo
print(us.startswith("hel"))    # True
print(us.endswith("lo"))       # True
print(us.find("ll"))           # 2
print(us.split("l"))           # ['he', '', 'o']
print(us.join(["a", "b"]))     # ahellob
us3 = us + " world"
print(us3)                     # hello world
print(len(us3))                # 11

print('done')

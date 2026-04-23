import reprlib

# ===== Repr defaults =====
r = reprlib.Repr()
print(r.maxlevel)       # 6
print(r.maxdict)        # 4
print(r.maxlist)        # 6
print(r.maxtuple)       # 6
print(r.maxset)         # 6
print(r.maxfrozenset)   # 6
print(r.maxdeque)       # 6
print(r.maxarray)       # 5
print(r.maxstring)      # 30
print(r.maxlong)        # 40
print(r.maxother)       # 30
print(r.fillvalue)      # ...

# ===== module-level repr() =====
print(reprlib.repr(None))           # None
print(reprlib.repr(42))             # 42
print(reprlib.repr(3.14))           # 3.14
print(reprlib.repr(True))           # True
print(reprlib.repr('hello'))        # 'hello'

# string truncation (maxstring=30)
long_str = 'hello world this is a long string that exceeds 30 chars'
print(reprlib.repr(long_str))       # 'hello world ...eeds 30 chars'

# short string under limit
print(reprlib.repr('short'))        # 'short'

# list truncation (maxlist=6)
print(reprlib.repr(list(range(3))))   # [0, 1, 2]
print(reprlib.repr(list(range(10))))  # [0, 1, 2, 3, 4, 5, ...]

# tuple
print(reprlib.repr(tuple(range(3))))    # (0, 1, 2)
print(reprlib.repr(tuple(range(10))))   # (0, 1, 2, 3, 4, 5, ...)

# dict (maxdict=4)
print(reprlib.repr({'a': 1, 'b': 2, 'c': 3}))          # {'a': 1, 'b': 2, 'c': 3}
print(reprlib.repr({'a':1,'b':2,'c':3,'d':4,'e':5}))    # {'a': 1, 'b': 2, 'c': 3, 'd': 4, ...}

# set
print(reprlib.repr({1, 2, 3}))          # {1, 2, 3}
print(reprlib.repr(set(range(10))))     # {0, 1, 2, 3, 4, 5, ...}

# frozenset
print(reprlib.repr(frozenset([1,2,3])))     # frozenset({1, 2, 3})
print(reprlib.repr(frozenset(range(10))))   # frozenset({0, 1, 2, 3, 4, 5, ...})

# long int truncation (maxlong=40)
big_int = 123456789012345678901234567890123456789012345678901234567890
print(reprlib.repr(big_int))    # 123456789012345678...2345678901234567890

# nested depth (maxlevel=6)
nested = [[[[[['deep']]]]]]
print(reprlib.repr(nested))     # [[[[[['deep']]]]]]  (6 levels)
too_deep = [[[[[[['too deep']]]]]]]
print(reprlib.repr(too_deep))   # [[[[[[[...]]]]]]]

# ===== Repr class — custom limits =====
r2 = reprlib.Repr()
r2.maxlist = 3
print(r2.repr([1, 2, 3, 4, 5]))    # [1, 2, 3, ...]

r2.maxdict = 2
print(r2.repr({'a': 1, 'b': 2, 'c': 3}))  # {'a': 1, 'b': 2, ...}

r2.maxstring = 10
print(r2.repr('hello world'))      # 'he...rld'

r2.maxlevel = 2
print(r2.repr([[1, 2], [3, 4]]))   # [[1, 2], [3, 4]]
print(r2.repr([[[1, 2]]]))         # [[...]]

# fillvalue
r3 = reprlib.Repr()
r3.fillvalue = '<...>'
print(r3.repr(list(range(10))))    # [0, 1, 2, 3, 4, 5, <...>]

# ===== recursive_repr decorator =====
class RecList:
    def __init__(self, items):
        self.data = list(items)
    def append(self, x):
        self.data.append(x)
    @reprlib.recursive_repr()
    def __repr__(self):
        return '<' + '|'.join(map(repr, self.data)) + '>'

m = RecList(['a', 'b', 'c'])
m.append(m)
m.append('x')
print(repr(m))   # <'a'|'b'|'c'|...|'x'>

# ===== recursive_repr custom fillvalue =====
class RecList2:
    def __init__(self, items):
        self.data = list(items)
    def append(self, x):
        self.data.append(x)
    @reprlib.recursive_repr(fillvalue='CYCLE')
    def __repr__(self):
        return '<' + '|'.join(map(repr, self.data)) + '>'

m2 = RecList2([1, 2])
m2.append(m2)
print(repr(m2))   # <1|2|CYCLE>

# ===== repr1 method =====
r4 = reprlib.Repr()
print(r4.repr1([1, 2, 3], 6))      # [1, 2, 3]
print(r4.repr1([1, 2, 3], 0))      # [...]

# ===== aRepr module-level instance =====
print(reprlib.aRepr.maxlist)        # 6

print('done')

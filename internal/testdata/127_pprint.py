import pprint

# ===== pformat — basic types =====
print(pprint.pformat(None))              # None
print(pprint.pformat(42))               # 42
print(pprint.pformat(3.14))             # 3.14
print(pprint.pformat(True))             # True
print(pprint.pformat('hello'))          # 'hello'
print(pprint.pformat(b'bytes'))         # b'bytes'

# ===== pformat — containers (fits on one line) =====
print(pprint.pformat([1, 2, 3]))        # [1, 2, 3]
print(pprint.pformat((1, 2, 3)))        # (1, 2, 3)
print(pprint.pformat({'a': 1, 'b': 2})) # {'a': 1, 'b': 2}

# ===== pformat — sort_dicts (default True) =====
print(pprint.pformat({'z': 3, 'a': 1, 'm': 2}))  # {'a': 1, 'm': 2, 'z': 3}

# ===== pformat — sort_dicts=False =====
print(pprint.pformat({'z': 3, 'a': 1, 'm': 2}, sort_dicts=False))

# ===== pformat — width / multi-line list =====
print(pprint.pformat(list(range(10)), width=20))

# ===== pformat — depth =====
deep = {'a': {'b': {'c': {'d': 4}}}}
print(pprint.pformat(deep, depth=2))    # {'a': {'b': {...}}}

# ===== pformat — indent =====
print(pprint.pformat([1, 2, 3], indent=4, width=5))

# ===== pformat — compact=True vs compact=False =====
items = list(range(20))
print(pprint.pformat(items, width=30, compact=True))
print(pprint.pformat(items, width=30, compact=False))

# ===== pprint — writes to stdout + newline =====
pprint.pprint([1, 2, 3])               # [1, 2, 3]
pprint.pprint({'z': 9, 'a': 1})        # {'a': 1, 'z': 9}  (sorted)

# ===== pp — sort_dicts=False by default =====
pprint.pp({'z': 9, 'a': 1})            # insertion order

# ===== isreadable =====
print(pprint.isreadable([1, 2, 3]))     # True
print(pprint.isreadable({'a': 1}))      # True
print(pprint.isreadable(object()))      # False

# ===== isrecursive =====
lst = [1, 2]
print(pprint.isrecursive(lst))          # False
lst.append(lst)
print(pprint.isrecursive(lst))          # True

# ===== saferepr =====
print(pprint.saferepr('hello'))         # 'hello'
print(pprint.saferepr([1, 2]))          # [1, 2]
lst2 = []
lst2.append(lst2)
print('<Recursion on list' in pprint.saferepr(lst2))   # True

# ===== PrettyPrinter class =====
pp1 = pprint.PrettyPrinter(indent=4, width=40)
print(pp1.pformat([1, 2, 3]))
print(pp1.pformat({'b': 2, 'a': 1}))

# multi-line with indent=4
print(pp1.pformat([1, 2, 3, 4, 5, 6, 7, 8, 9, 10]))

# PrettyPrinter.pprint
pp2 = pprint.PrettyPrinter()
pp2.pprint({'b': 2, 'a': 1})

# PrettyPrinter.isreadable / isrecursive
print(pp2.isreadable([1, 2]))           # True
print(pp2.isreadable(object()))         # False
print(pp2.isrecursive([1, 2]))          # False

# PrettyPrinter with sort_dicts=False
pp3 = pprint.PrettyPrinter(sort_dicts=False)
print(pp3.pformat({'z': 3, 'a': 1}))   # {'z': 3, 'a': 1}

# PrettyPrinter with depth
pp4 = pprint.PrettyPrinter(depth=1)
print(pp4.pformat({'a': {'b': 1}}))    # {'a': {...}}

# PrettyPrinter.format — returns (repr_str, readable, recursive)
pp5 = pprint.PrettyPrinter()
s, readable, recursive = pp5.format([1, 2, 3], {}, 0, 0)
print(s)                               # [1, 2, 3]
print(readable)                        # True
print(recursive)                       # False

print('done')

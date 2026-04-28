import pyclbr

# ── 1. __all__ ────────────────────────────────────────────────────────────────
print(pyclbr.__all__ == ['readmodule', 'readmodule_ex', 'Class', 'Function'])

# ── 2. Class and Function are subclasses of _Object ──────────────────────────
print(issubclass(pyclbr.Class, pyclbr._Object))
print(issubclass(pyclbr.Function, pyclbr._Object))

# ── 3. _Object construction ───────────────────────────────────────────────────
obj = pyclbr._Object('mymod', 'MyObj', '/tmp/foo.py', 10, 20, None)
print(obj.module == 'mymod')
print(obj.name == 'MyObj')
print(obj.file == '/tmp/foo.py')
print(obj.lineno == 10)
print(obj.end_lineno == 20)
print(obj.parent is None)
print(obj.children == {})

# ── 4. Class construction ─────────────────────────────────────────────────────
cls = pyclbr.Class('mymod', 'MyClass', None, '/tmp/foo.py', 5, end_lineno=15)
print(cls.module == 'mymod')
print(cls.name == 'MyClass')
print(cls.file == '/tmp/foo.py')
print(cls.lineno == 5)
print(cls.end_lineno == 15)
print(cls.parent is None)
print(cls.children == {})
print(cls.super == [])
print(cls.methods == {})

# ── 5. Class with super list ──────────────────────────────────────────────────
base = pyclbr.Class('mymod', 'Base', None, '/tmp/foo.py', 1, end_lineno=10)
derived = pyclbr.Class('mymod', 'Derived', [base], '/tmp/foo.py', 12, end_lineno=20)
print(len(derived.super) == 1)
print(derived.super[0] is base)

# ── 6. Function construction ──────────────────────────────────────────────────
fn = pyclbr.Function('mymod', 'my_func', '/tmp/foo.py', 3, end_lineno=7)
print(fn.module == 'mymod')
print(fn.name == 'my_func')
print(fn.file == '/tmp/foo.py')
print(fn.lineno == 3)
print(fn.end_lineno == 7)
print(fn.parent is None)
print(fn.children == {})
print(fn.is_async == False)

# ── 7. Function with is_async=True ────────────────────────────────────────────
async_fn = pyclbr.Function('mymod', 'async_func', '/tmp/foo.py', 10, is_async=True, end_lineno=15)
print(async_fn.is_async == True)

# ── 8. Parent-child relationship ──────────────────────────────────────────────
parent_cls = pyclbr.Class('mymod', 'Parent', None, '/tmp/foo.py', 1, end_lineno=20)
child_fn = pyclbr.Function('mymod', 'method', '/tmp/foo.py', 2, parent=parent_cls, end_lineno=4)
print('method' in parent_cls.children)
print(parent_cls.children['method'] is child_fn)
print(child_fn.parent is parent_cls)

# ── 9. Function in Class adds to parent.methods ───────────────────────────────
print('method' in parent_cls.methods)
print(parent_cls.methods['method'] == 2)

# ── 10. Multiple methods in a class ──────────────────────────────────────────
cls2 = pyclbr.Class('mymod', 'MyClass2', None, '/tmp/foo.py', 5, end_lineno=30)
m1 = pyclbr.Function('mymod', 'method_one', '/tmp/foo.py', 6, parent=cls2, end_lineno=8)
m2 = pyclbr.Function('mymod', 'method_two', '/tmp/foo.py', 9, parent=cls2, end_lineno=11)
print(len(cls2.methods) == 2)
print('method_one' in cls2.methods)
print('method_two' in cls2.methods)
print(len(cls2.children) == 2)

# ── 11. Nested class (Class with Class parent) ────────────────────────────────
outer = pyclbr.Class('mymod', 'Outer', None, '/tmp/foo.py', 1, end_lineno=20)
inner = pyclbr.Class('mymod', 'Inner', None, '/tmp/foo.py', 2, parent=outer, end_lineno=5)
print('Inner' in outer.children)
print(outer.children['Inner'] is inner)
print(inner.parent is outer)

# ── 12. isinstance checks ─────────────────────────────────────────────────────
print(isinstance(cls, pyclbr._Object))
print(isinstance(fn, pyclbr._Object))
print(isinstance(cls, pyclbr.Class))
print(isinstance(fn, pyclbr.Function))

# ── 13. readmodule returns dict ───────────────────────────────────────────────
result = pyclbr.readmodule('os')
print(isinstance(result, dict))

# ── 14. readmodule_ex returns dict ────────────────────────────────────────────
result_ex = pyclbr.readmodule_ex('os')
print(isinstance(result_ex, dict))

# ── 15. callable checks ───────────────────────────────────────────────────────
print(callable(pyclbr.readmodule))
print(callable(pyclbr.readmodule_ex))

# ── 16. _Object with end_lineno=None ─────────────────────────────────────────
obj2 = pyclbr._Object('m', 'n', 'f', 1, None, None)
print(obj2.end_lineno is None)

# ── 17. Class with super=None gives empty list ───────────────────────────────
cls3 = pyclbr.Class('m', 'C', None, 'f', 1)
print(cls3.super == [])

# ── 18. Function without parent stays independent ────────────────────────────
standalone = pyclbr.Function('m', 'standalone', 'f', 5)
print(standalone.parent is None)
print(standalone.is_async == False)

print('done')

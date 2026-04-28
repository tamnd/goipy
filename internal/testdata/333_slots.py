# 333_slots.py — v0.0.333 __slots__ semantics
#
# goipy doesn't optimise memory layout for slot-bearing classes (slots
# still live in the instance's underlying dict). What it *does*
# enforce is the user-visible contract: setattr/delattr to a non-slot
# name raises AttributeError, and __dict__ is hidden (or filtered to
# exclude slot names) when slots are declared.

# ── 1. Basic tuple form ───────────────────────────────────────────────────────
class Point:
    __slots__ = ('x', 'y')

p = Point()
p.x = 3
p.y = 4
assert p.x == 3 and p.y == 4

try:
    p.z = 5
except AttributeError as e:
    assert 'z' in str(e)
else:
    raise AssertionError('expected AttributeError on non-slot setattr')

# __dict__ is hidden when slots are declared and no base allows dict.
try:
    _ = p.__dict__
except AttributeError:
    pass
else:
    raise AssertionError('expected AttributeError on __dict__')

# ── 2. Single-string form ─────────────────────────────────────────────────────
class Single:
    __slots__ = 'only'

s = Single()
s.only = 1
assert s.only == 1
try:
    s.other = 2
except AttributeError:
    pass
else:
    raise AssertionError('expected AttributeError')

# ── 3. List form ──────────────────────────────────────────────────────────────
class L:
    __slots__ = ['a', 'b']

x = L()
x.a, x.b = 10, 20
assert x.a == 10 and x.b == 20
try:
    x.c = 0
except AttributeError:
    pass
else:
    raise AssertionError

# ── 4. Dict form (keys are slot names) ────────────────────────────────────────
class D:
    __slots__ = {'k': 'doc-for-k', 'v': 'doc-for-v'}

d = D()
d.k = 'key'
d.v = 'val'
assert d.k == 'key' and d.v == 'val'
try:
    d.extra = 1
except AttributeError:
    pass
else:
    raise AssertionError

# ── 5. del on slot attr works ─────────────────────────────────────────────────
class Box:
    __slots__ = ('v',)

b = Box()
b.v = 42
del b.v
try:
    _ = b.v
except AttributeError:
    pass
else:
    raise AssertionError('reading deleted slot should raise')

# del on non-slot raises AttributeError too
try:
    del b.zz
except AttributeError:
    pass
else:
    raise AssertionError

# ── 6. Subclass without __slots__ regains __dict__ ────────────────────────────
class Strict:
    __slots__ = ('a',)

class Loose(Strict):
    pass

obj = Loose()
obj.a = 1            # parent slot still works
obj.dynamic = 'ok'   # subclass without slots → has __dict__
assert obj.a == 1
assert obj.dynamic == 'ok'
# __dict__ on a slot-bearing instance excludes slot names — slot 'a'
# is shadowed, only the dynamic attribute appears.
assert obj.__dict__ == {'dynamic': 'ok'}

# ── 7. Subclass with __slots__ extends the allowed names ──────────────────────
class Base:
    __slots__ = ('x',)

class Sub(Base):
    __slots__ = ('y',)

t = Sub()
t.x = 1
t.y = 2
assert t.x == 1 and t.y == 2
try:
    t.z = 3
except AttributeError:
    pass
else:
    raise AssertionError

# ── 8. '__dict__' in slots re-enables the dict path ───────────────────────────
class Mixed:
    __slots__ = ('fixed', '__dict__')

m = Mixed()
m.fixed = 'set'
m.anything = 'goes'   # allowed because __dict__ is in slots
assert m.fixed == 'set'
assert m.anything == 'goes'
# __dict__ visible
assert isinstance(m.__dict__, dict)

# ── 9. namedtuple-style empty slots ───────────────────────────────────────────
class Empty:
    __slots__ = ()

e = Empty()
try:
    e.foo = 1
except AttributeError:
    pass
else:
    raise AssertionError('empty __slots__=() must reject all setattr')

# ── 10. __weakref__ in slots is accepted (inert in goipy) ─────────────────────
class Wr:
    __slots__ = ('v', '__weakref__')

w = Wr()
w.v = 'hi'
assert w.v == 'hi'
try:
    w.other = 0
except AttributeError:
    pass
else:
    raise AssertionError

print('ok')

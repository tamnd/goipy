import collections
import operator
import functools

# --- deque stress ---

# 1) maxlen truncates on construction when iterable exceeds it.
d = collections.deque([1, 2, 3, 4, 5], maxlen=3)
print(list(d), d.maxlen)

# 2) append past maxlen drops from the left.
d = collections.deque([1, 2, 3], maxlen=3)
d.append(4)
d.append(5)
print(list(d))

# 3) appendleft past maxlen drops from the right.
d = collections.deque([1, 2, 3], maxlen=3)
d.appendleft(0)
d.appendleft(-1)
print(list(d))

# 4) extend past maxlen keeps the rightmost maxlen items.
d = collections.deque(maxlen=3)
d.extend(range(10))
print(list(d))

# 5) extendleft reverses the iterable and keeps the leftmost.
d = collections.deque(maxlen=3)
d.extendleft([1, 2, 3, 4])
print(list(d))

# 6) rotate by value larger than length (wraps).
d = collections.deque([1, 2, 3, 4])
d.rotate(7)
print(list(d))

# 7) rotate negative wraps.
d = collections.deque([1, 2, 3, 4])
d.rotate(-5)
print(list(d))

# 8) rotate on empty deque is a no-op.
d = collections.deque()
d.rotate(3)
print(list(d))

# 9) pop/popleft on empty raises IndexError.
try:
    collections.deque().pop()
except IndexError as e:
    print("pop-empty:", e)
try:
    collections.deque().popleft()
except IndexError as e:
    print("popleft-empty:", e)

# 10) index raises ValueError when not found.
d = collections.deque([1, 2, 3])
try:
    d.index(99)
except ValueError as e:
    print("idx-missing:", e)

# --- Counter stress ---

# 11) Counter from iterable and kwargs.
c = collections.Counter("aabbc")
print(c.most_common())

# 12) missing keys return 0 without inserting.
c = collections.Counter("abc")
print(c["z"])
print("z" in c)

# 13) arithmetic update with negative then total.
c = collections.Counter({"a": 3, "b": 5, "c": 1})
c.subtract({"a": 1, "b": 5})
print(sorted(c.items()))
print(c.total())

# 14) most_common with n=0 returns empty.
c = collections.Counter("aaabbc")
print(c.most_common(0))

# 15) most_common with n larger than keys returns all.
print(c.most_common(50))

# 16) update with an iterable.
c = collections.Counter()
c.update("aabbbc")
c.update("bcd")
print(sorted(c.items()))

# 17) elements respects counts.
c = collections.Counter(a=2, b=0, c=3)
print(sorted(c.elements()))

# --- defaultdict stress ---

# 18) factory=int gives 0 default.
dd = collections.defaultdict(int)
for word in "abc abc xyz".split():
    for ch in word:
        dd[ch] += 1
print(sorted(dd.items()))

# 19) factory=list groups values.
dd = collections.defaultdict(list)
for k, v in [("a", 1), ("b", 2), ("a", 3), ("c", 4), ("a", 5)]:
    dd[k].append(v)
print(sorted(dd.items()))

# 20) factory=set builds sets.
dd = collections.defaultdict(set)
for k, v in [("x", 1), ("x", 2), ("x", 1), ("y", 9)]:
    dd[k].add(v)
print(sorted((k, sorted(vs)) for k, vs in dd.items()))

# 21) default_factory attribute.
dd = collections.defaultdict(int)
print(dd.default_factory is int)

# 22) nested defaultdict.
tree = collections.defaultdict(lambda: collections.defaultdict(int))
tree["a"]["x"] += 1
tree["a"]["y"] += 2
tree["b"]["x"] += 10
print(sorted(tree["a"].items()))
print(sorted(tree["b"].items()))

# --- OrderedDict stress ---

# 23) preserves insertion order; equality with a plain dict ignores order.
od = collections.OrderedDict([("a", 1), ("b", 2), ("c", 3)])
print(list(od.items()))

# 24) move_to_end(last=True) vs last=False.
od = collections.OrderedDict([("a", 1), ("b", 2), ("c", 3)])
od.move_to_end("a")
print(list(od.keys()))
od.move_to_end("c", last=False)
print(list(od.keys()))

# 25) popitem default is last; last=False pops the first.
od = collections.OrderedDict([("a", 1), ("b", 2), ("c", 3)])
print(od.popitem())
print(od.popitem(last=False))
print(list(od.items()))

# 26) move_to_end on missing key raises KeyError.
od = collections.OrderedDict([("a", 1)])
try:
    od.move_to_end("missing")
except KeyError as e:
    print("mte-missing:", e)

# --- namedtuple stress ---

# 27) defaults align with trailing fields.
Account = collections.namedtuple("Account", ["owner", "balance", "currency"], defaults=["USD"])
a = Account("alice", 100)
print(a)
print(a.currency)

# 28) multiple defaults.
Rect = collections.namedtuple("Rect", "x y w h", defaults=[0, 0, 1, 1])
r = Rect()
print(r)
r2 = Rect(5)
print(r2)

# 29) missing required field raises TypeError.
try:
    Account()
    print("nt-missing: no error")
except TypeError:
    print("nt-missing: TypeError")

# 30) _replace preserves untouched fields.
Point = collections.namedtuple("Point", "x y z")
p = Point(1, 2, 3)
p2 = p._replace(y=99)
print(p2)
print(p == p2)

# 31) iteration yields fields in order.
print(list(Point(10, 20, 30)))

# 32) _asdict returns ordered-by-field dict.
print(Point(1, 2, 3)._asdict())

# 33) equality is within the same class only.
A = collections.namedtuple("A", "x")
B = collections.namedtuple("B", "x")
print(A(1) == A(1))
print(A(1) == B(1))

# --- operator stress ---

# 34) attrgetter with dotted path.
class Inner:
    val = 42

class Outer:
    inner = Inner()

print(operator.attrgetter("inner.val")(Outer()))

# 35) attrgetter with multiple dotted paths.
class X:
    def __init__(self, a, b):
        self.a = a
        self.b = b

x = X([1, 2, 3], X("nested", 7))
get = operator.attrgetter("a", "b.b", "b.a")
print(get(x))

# 36) itemgetter on dict.
d = {"a": 1, "b": 2, "c": 3}
print(operator.itemgetter("a", "c")(d))

# 37) itemgetter on tuple with negative index.
print(operator.itemgetter(-1)((10, 20, 30)))

# 38) methodcaller with positional+keyword args.
class Greeter:
    def greet(self, who, *, excited=False):
        return f"hi {who}" + ("!" if excited else "")

print(operator.methodcaller("greet", "world", excited=True)(Greeter()))

# 39) chained operator usage: sum via reduce.
print(functools.reduce(operator.add, range(11)))
print(functools.reduce(operator.mul, range(1, 6)))

# 40) sorted by itemgetter.
data = [("b", 2), ("a", 3), ("c", 1)]
print(sorted(data, key=operator.itemgetter(1)))
print(sorted(data, key=operator.itemgetter(0)))

# 41) sorted by attrgetter.
class Row:
    def __init__(self, name, n):
        self.name = name
        self.n = n
    def __repr__(self):
        return f"Row({self.name!r},{self.n})"

rows = [Row("c", 3), Row("a", 1), Row("b", 2)]
print(sorted(rows, key=operator.attrgetter("n")))

# 42) operator.index.
print(operator.index(5))
print(operator.index(True))

# 43) operator.pos on ints passes through.
print(operator.pos(7))

# 44) truthy / not_ chained.
print(operator.not_(operator.truth(0)))
print(operator.not_(operator.truth([])))
print(operator.not_(operator.truth([1])))

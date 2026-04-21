import collections
import operator

# --- deque basics ---
d = collections.deque([1, 2, 3])
d.append(4)
d.appendleft(0)
print(list(d))
print(d.pop(), d.popleft())
print(list(d))
d.extend([10, 20])
d.extendleft([-1, -2])
print(list(d))
d.rotate(2)
print(list(d))
d.rotate(-3)
print(list(d))
d.reverse()
print(list(d))
print(d.count(10), d.index(20))
d.clear()
print(list(d), len(d))

# deque with maxlen
dm = collections.deque([1, 2, 3], maxlen=3)
dm.append(4)
print(list(dm), dm.maxlen)
dm.appendleft(99)
print(list(dm))

# --- Counter basics ---
c = collections.Counter("abracadabra")
print(c["a"], c["b"], c["z"])
print(c.most_common(2))
print(sorted(c.elements()))
c.update("aaa")
print(c["a"])
c.subtract("a")
print(c["a"])
print(c.total())

c2 = collections.Counter(a=3, b=1)
print(c2["a"], c2["b"])

# --- defaultdict basics ---
dd = collections.defaultdict(list)
dd["x"].append(1)
dd["x"].append(2)
dd["y"].append(9)
print(dict(dd))
print(dd["z"])  # creates empty list

dd2 = collections.defaultdict(int)
for ch in "mississippi":
    dd2[ch] += 1
print(sorted(dd2.items()))

# --- OrderedDict basics ---
od = collections.OrderedDict()
od["a"] = 1
od["b"] = 2
od["c"] = 3
print(list(od.items()))
od.move_to_end("a")
print(list(od.keys()))
od.move_to_end("a", last=False)
print(list(od.keys()))
print(od.popitem())
print(od.popitem(last=False))

# --- namedtuple basics ---
Point = collections.namedtuple("Point", ["x", "y"])
p = Point(1, 2)
print(p.x, p.y)
print(p[0], p[1])
print(len(p))
print(list(p))
q = Point(x=3, y=4)
print(q == Point(3, 4))
print(p == q)
print(p._asdict())
print(p._replace(y=99))
print(Point._fields)

# namedtuple from string
Color = collections.namedtuple("Color", "r g b")
c = Color(1, 2, 3)
print(c.r, c.g, c.b)

# --- operator basics ---
print(operator.add(2, 3))
print(operator.sub(10, 4))
print(operator.mul(3, 4))
print(operator.truediv(7, 2))
print(operator.floordiv(7, 2))
print(operator.mod(7, 2))
print(operator.pow(2, 8))
print(operator.neg(5))
print(operator.not_(0), operator.not_(1))
print(operator.truth([]), operator.truth([1]))

print(operator.lt(1, 2), operator.le(2, 2), operator.eq(2, 2))
print(operator.ne(1, 2), operator.gt(3, 2), operator.ge(3, 3))

print(operator.and_(0b1100, 0b1010))
print(operator.or_(0b1100, 0b1010))
print(operator.xor(0b1100, 0b1010))
print(operator.lshift(1, 4))
print(operator.rshift(16, 2))

# getitem / setitem / delitem / contains
lst = [10, 20, 30]
print(operator.getitem(lst, 1))
operator.setitem(lst, 0, 99)
print(lst)
operator.delitem(lst, 0)
print(lst)
print(operator.contains([1, 2, 3], 2))
print(operator.contains("hello", "x"))

# attrgetter / itemgetter / methodcaller
class Obj:
    def __init__(self, x, y):
        self.x = x
        self.y = y

o = Obj(10, 20)
getx = operator.attrgetter("x")
getxy = operator.attrgetter("x", "y")
print(getx(o), getxy(o))

g0 = operator.itemgetter(0)
g02 = operator.itemgetter(0, 2)
print(g0([10, 20, 30]))
print(g02([10, 20, 30]))

upper = operator.methodcaller("upper")
print(upper("hello"))
replace = operator.methodcaller("replace", "l", "L")
print(replace("hello"))

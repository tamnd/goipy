# Stress: dunders through inheritance, NotImplemented fallback, rich compare
# with mixed types, iterator protocol, format/str/repr precedence, hash
# stability across containers.

# --- inheritance: base defines __add__, child inherits ---
class Base:
    def __init__(self, x):
        self.x = x
    def __repr__(self):
        return f"{type(self).__name__}({self.x})"
    def __add__(self, o):
        return type(self)(self.x + o.x)
    def __eq__(self, o):
        return isinstance(o, Base) and self.x == o.x
    def __hash__(self):
        return hash((type(self).__name__, self.x))

class Child(Base):
    pass

c1 = Child(3)
c2 = Child(4)
print(c1 + c2)
print(c1 == Child(3))
print(c1 == Base(3))  # same x, different type names -> distinct hashes but eq True by Base.__eq__

# --- NotImplemented fallback between two instance types ---
class A:
    def __init__(self, v):
        self.v = v
    def __add__(self, other):
        if isinstance(other, A):
            return A(self.v + other.v)
        return NotImplemented
    def __repr__(self):
        return f"A({self.v})"

class B:
    def __init__(self, v):
        self.v = v
    def __radd__(self, other):
        return ("B.radd", other.v, self.v)
    def __repr__(self):
        return f"B({self.v})"

print(A(1) + A(2))
print(A(1) + B(9))  # A.__add__ returns NotImplemented; falls through to B.__radd__

# --- __iter__ yielding from a generator function-ish manual iterator ---
class Count:
    def __init__(self, n):
        self.n = n
    def __iter__(self):
        self._i = 0
        return self
    def __next__(self):
        if self._i >= self.n:
            raise StopIteration
        v = self._i
        self._i += 1
        return v

print(list(Count(5)))
print(sum(Count(10)))

# --- __bool__ vs __len__ ---
class BoolFromLen:
    def __init__(self, n):
        self.n = n
    def __len__(self):
        return self.n

print(bool(BoolFromLen(0)), bool(BoolFromLen(3)))

class OnlyBool:
    def __bool__(self):
        return True

print(bool(OnlyBool()))

# --- __eq__ / __hash__ respected by set + dict ---
class Key:
    def __init__(self, x, tag):
        self.x = x
        self.tag = tag
    def __eq__(self, o):
        return isinstance(o, Key) and self.x == o.x
    def __hash__(self):
        return hash(self.x)

s = {Key(1, "a"), Key(1, "b"), Key(2, "c")}
print(len(s))  # 2 because Key(1,"a") == Key(1,"b")
d = {}
d[Key(5, "first")] = 1
d[Key(5, "second")] = 2  # overwrites
print(len(d), list(d.values()))

# --- repr/str precedence; str falls back to repr if __str__ missing ---
class OnlyRepr:
    def __repr__(self):
        return "R!"

print(str(OnlyRepr()))
print(repr(OnlyRepr()))
print(f"{OnlyRepr()}")  # uses __str__ -> __repr__ fallback

# --- reflected ops with same type: __add__ wins, no __radd__ call ---
class Log:
    calls = []
    def __init__(self, x):
        self.x = x
    def __add__(self, o):
        Log.calls.append("add")
        return Log(self.x + (o.x if isinstance(o, Log) else o))
    def __radd__(self, o):
        Log.calls.append("radd")
        return Log((o.x if isinstance(o, Log) else o) + self.x)
    def __repr__(self):
        return f"Log({self.x})"

Log.calls = []
r = Log(1) + Log(2)
print(r, Log.calls)
Log.calls = []
r = 10 + Log(5)
print(r, Log.calls)

# --- iteration via __getitem__, stop on IndexError ---
class IdxSeq:
    data = ["x", "y", "z"]
    def __getitem__(self, i):
        return self.data[i]

print(list(IdxSeq()))
print("y" in IdxSeq())

# --- __call__ with multiple args ---
class Adder2:
    def __init__(self, base):
        self.base = base
    def __call__(self, *args):
        return self.base + sum(args)

f = Adder2(100)
print(f(1, 2, 3))
print(f())

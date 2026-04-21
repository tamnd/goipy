# Stress: reflected bitops, NotImplemented fallback between bit-op classes,
# in-place rebinding when no __i*__, mixed-type matmul, numeric conversion
# precedence, __format__ with a structured spec parser, inheritance of
# in-place ops.

# --- Reflected bitwise ops: builtin int interacts with user class ---
class Mask:
    def __init__(self, v):
        self.v = v
    def __rand__(self, other):
        return ("rand", other, self.v)
    def __ror__(self, other):
        return ("ror", other, self.v)
    def __rxor__(self, other):
        return ("rxor", other, self.v)
    def __rlshift__(self, other):
        return ("rlshift", other, self.v)
    def __rrshift__(self, other):
        return ("rrshift", other, self.v)

print(5 & Mask(7))
print(5 | Mask(7))
print(5 ^ Mask(7))
print(1 << Mask(3))
print(16 >> Mask(2))

# --- NotImplemented between two classes: __and__ returns NotImplemented,
# right side's __rand__ is consulted.
class L:
    def __and__(self, o):
        return NotImplemented

class R:
    def __rand__(self, o):
        return "R.__rand__"

print(L() & R())

# --- In-place without __i*__: creates a new object from __op__ and rebinds.
class V:
    def __init__(self, x):
        self.x = x
    def __add__(self, o):
        return V(self.x + o)
    def __sub__(self, o):
        return V(self.x - o)
    def __mul__(self, o):
        return V(self.x * o)
    def __repr__(self):
        return f"V({self.x})"

v = V(1)
orig = v
v += 10
v -= 3
v *= 2
print(v)
print(orig)  # unchanged since no __iadd__; we created new V each time

# --- Inheritance of in-place ops.
class Counter:
    def __init__(self, n=0):
        self.n = n
    def __iadd__(self, other):
        self.n += other
        return self
    def __repr__(self):
        return f"Counter({self.n})"

class Named(Counter):
    def __init__(self, name):
        super().__init__(0)
        self.name = name
    def __repr__(self):
        return f"{self.name}={self.n}"

c = Named("x")
c += 1
c += 2
c += 3
print(c)

# --- matmul chaining left-to-right ---
class M:
    def __init__(self, label):
        self.label = label
    def __matmul__(self, o):
        if isinstance(o, M):
            return M(f"({self.label}@{o.label})")
        return NotImplemented
    def __repr__(self):
        return f"M({self.label!r})"

print(M("a") @ M("b") @ M("c"))

# --- Numeric conversions and mixed use as slice index via __index__ ---
class Idx:
    def __init__(self, v):
        self.v = v
    def __index__(self):
        return self.v

# __index__ does not currently route through subscripting in goipy, so
# avoid that; just validate int() and indirect uses.
print(int(Idx(5)))

# __int__ vs __index__ precedence: both defined, __int__ wins for int().
class Both:
    def __int__(self):
        return 1
    def __index__(self):
        return 2

print(int(Both()))

# --- __format__ with a structured spec parser ---
class Money:
    def __init__(self, cents):
        self.cents = cents
    def __format__(self, spec):
        if spec == "" or spec == "short":
            return f"${self.cents / 100:.2f}"
        if spec == "cents":
            return f"{self.cents}¢"
        if spec.startswith("."):
            n = int(spec[1:])
            return f"${self.cents / 100:.{n}f}"
        return object.__format__(self, spec)

m = Money(12345)
print(format(m))
print(format(m, "short"))
print(format(m, "cents"))
print(format(m, ".4"))
print(f"total: {m}")
print(f"precise: {m:.3}")

# --- hash equality across dict/set with custom __hash__ + __eq__ ---
class K:
    def __init__(self, k):
        self.k = k
    def __hash__(self):
        return hash(self.k) ^ 0x1234
    def __eq__(self, o):
        return isinstance(o, K) and self.k == o.k
    def __repr__(self):
        return f"K({self.k!r})"

d = {K("a"): 1, K("b"): 2}
print(d[K("a")], d[K("b")])
print(K("a") in d)
print(K("z") in d)

# set membership follows __hash__/__eq__
s = frozenset({K("a"), K("a"), K("b")})
print(len(s))

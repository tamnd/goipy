# Extra dunders: bit/shift ops, matmul, unary +/abs/~, in-place ops,
# numeric conversions, divmod/round, __format__.

class Bits:
    def __init__(self, x):
        self.x = x
    def __repr__(self):
        return f"Bits({self.x})"
    def __and__(self, o):
        return Bits(self.x & (o.x if isinstance(o, Bits) else o))
    def __or__(self, o):
        return Bits(self.x | (o.x if isinstance(o, Bits) else o))
    def __xor__(self, o):
        return Bits(self.x ^ (o.x if isinstance(o, Bits) else o))
    def __lshift__(self, o):
        return Bits(self.x << o)
    def __rshift__(self, o):
        return Bits(self.x >> o)
    def __invert__(self):
        return Bits(~self.x)
    def __pos__(self):
        return Bits(+self.x)
    def __abs__(self):
        return Bits(abs(self.x))
    def __neg__(self):
        return Bits(-self.x)

a = Bits(0b1100)
b = Bits(0b1010)
print(a & b)
print(a | b)
print(a ^ b)
print(a << 2)
print(a >> 1)
print(~a)
print(+a)
print(abs(Bits(-5)))
print(-a)

# Matmul
class M:
    def __init__(self, v):
        self.v = v
    def __matmul__(self, o):
        return ("matmul", self.v, o)
    def __repr__(self):
        return f"M({self.v})"

print(M("a") @ 3)

# In-place ops with __i*__ mutating self
class Box:
    def __init__(self, v):
        self.v = v
    def __iadd__(self, o):
        self.v += o
        return self
    def __isub__(self, o):
        self.v -= o
        return self
    def __imul__(self, o):
        self.v *= o
        return self
    def __ior__(self, o):
        self.v |= o
        return self
    def __repr__(self):
        return f"Box({self.v})"

p = Box(10)
p += 5
print(p)
p -= 3
print(p)
p *= 2
print(p)
p |= 1
print(p)

# In-place without __iadd__ falls back to __add__
class Add:
    def __init__(self, v):
        self.v = v
    def __add__(self, o):
        return Add(self.v + o)
    def __repr__(self):
        return f"Add({self.v})"

q = Add(7)
q += 3
print(q)

# __int__, __float__
class N:
    def __init__(self, x):
        self.x = x
    def __int__(self):
        return self.x
    def __float__(self):
        return float(self.x) + 0.25

print(int(N(42)))
print(float(N(8)))

# __index__ as int() fallback
class Idx:
    def __init__(self, x):
        self.x = x
    def __index__(self):
        return self.x

print(int(Idx(7)))

# divmod and round
class R:
    def __init__(self, x):
        self.x = x
    def __divmod__(self, o):
        return (self.x // o, self.x % o)
    def __round__(self, ndigits=None):
        if ndigits is None:
            return int(self.x)
        return round(self.x, ndigits)

print(divmod(R(17), 5))
print(round(R(3.14159)))
print(round(R(3.14159), 2))

# __format__ custom
class Tag:
    def __format__(self, spec):
        return f"<Tag spec={spec!r}>"

t = Tag()
print(format(t))
print(format(t, "hex"))
print(f"{t:>5}")

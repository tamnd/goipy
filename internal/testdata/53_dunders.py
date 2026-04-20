# User-defined dunder methods: arithmetic, comparison, container,
# conversion, call, iter.

class V:
    def __init__(self, x):
        self.x = x
    def __repr__(self):
        return f"V({self.x})"
    def __str__(self):
        return f"<v {self.x}>"
    def __eq__(self, other):
        if isinstance(other, V):
            return self.x == other.x
        return NotImplemented
    def __lt__(self, other):
        return self.x < other.x
    def __hash__(self):
        return hash(self.x)
    def __add__(self, other):
        if isinstance(other, V):
            return V(self.x + other.x)
        return V(self.x + other)
    def __radd__(self, other):
        return V(other + self.x)
    def __sub__(self, other):
        return V(self.x - (other.x if isinstance(other, V) else other))
    def __mul__(self, other):
        return V(self.x * other)
    def __neg__(self):
        return V(-self.x)
    def __bool__(self):
        return self.x != 0
    def __len__(self):
        return abs(self.x)

a = V(3)
b = V(5)

print(repr(a))
print(str(a))
print(a + b)
print(a + 10)
print(10 + a)
print(b - a)
print(a * 4)
print(-a)
print(a == V(3))
print(a == V(4))
print(a != V(4))
print(a < b)
print(b > a)
print(bool(a), bool(V(0)))
print(len(V(-7)))

# hash / dict / set use __hash__ + __eq__
d = {V(1): "one", V(2): "two"}
print(d[V(1)], d[V(2)])
s = {V(1), V(2), V(1)}
print(len(s))
print(V(1) in s)
print(V(9) in s)

# __getitem__, __setitem__, __contains__, __iter__
class Seq:
    def __init__(self):
        self.data = {}
    def __getitem__(self, k):
        return self.data[k]
    def __setitem__(self, k, v):
        self.data[k] = v
    def __delitem__(self, k):
        del self.data[k]
    def __len__(self):
        return len(self.data)
    def __contains__(self, k):
        return k in self.data
    def __iter__(self):
        return iter(self.data)

q = Seq()
q["a"] = 1
q["b"] = 2
print(q["a"])
print(len(q))
print("a" in q, "z" in q)
print(sorted(list(q)))
del q["a"]
print("a" in q)

# __call__
class Adder:
    def __init__(self, n):
        self.n = n
    def __call__(self, x):
        return x + self.n

inc = Adder(10)
print(inc(5))
print(inc(100))

# __getitem__ iteration protocol
class Count:
    def __getitem__(self, i):
        if i >= 3:
            raise IndexError
        return i * 10

print([x for x in Count()])

# Reflected op falls through to right operand
class Half:
    def __rsub__(self, other):
        return other - 0.5

print(10 - Half())

class Temp:
    def __init__(self, c):
        self._c = c
    @property
    def c(self):
        return self._c
    @property
    def f(self):
        return self._c * 9 / 5 + 32
    @classmethod
    def from_f(cls, f):
        return cls((f - 32) * 5 / 9)
    @staticmethod
    def unit():
        return "celsius"

t = Temp(100)
print(t.c, t.f, t.unit(), Temp.unit())
u = Temp.from_f(32)
print(u.c)

# classmethod chains through inheritance
class MC(Temp):
    @classmethod
    def boiling(cls):
        return cls(100)

b = MC.boiling()
print(type(b).__name__, b.f)

# super() calling through a property
class A:
    @property
    def tag(self):
        return "A"

class B(A):
    @property
    def tag(self):
        return "B+" + super().tag

print(B().tag)

# staticmethod accessed through instance still unbound
class S:
    @staticmethod
    def twice(x):
        return x * 2

s = S()
print(s.twice(7))
print(S.twice(7))

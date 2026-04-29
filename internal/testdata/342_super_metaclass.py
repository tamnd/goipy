# PEP-related: explicit super(C, instance) + metaclass __instancecheck__/__subclasscheck__.
# Section markers help diff against CPython.

print("# section 1: super(B, b).hi() walks past B to A.hi")


class A:
    def hi(self):
        return "A.hi"


class B(A):
    def hi(self):
        return "B.hi"


b = B()
print(b.hi())
print(super(B, b).hi())


print("# section 2: three-deep MRO")


class C(B):
    def hi(self):
        return "C.hi"


c = C()
print(c.hi())
print(super(C, c).hi())
print(super(B, c).hi())


print("# section 3: diamond — explicit super matches implicit")


class D:
    def hi(self):
        return "D.hi"


class E(D):
    def hi(self):
        return "E.hi" + "/" + super().hi()


class F(D):
    def hi(self):
        return "F.hi" + "/" + super().hi()


class G(E, F):
    def hi(self):
        return "G.hi" + "/" + super().hi()


g = G()
print(g.hi())
print(super(G, g).hi())
print(super(E, g).hi())


print("# section 4: super on classmethod still binds class")


class H:
    @classmethod
    def klass(cls):
        return cls.__name__


class I(H):
    @classmethod
    def klass(cls):
        return "I:" + super().klass()


print(I.klass())
print(super(I, I).klass())


print("# section 5: metaclass __instancecheck__ overrides isinstance")


class Meta(type):
    def __instancecheck__(cls, inst):
        return True


class Q(metaclass=Meta):
    pass


print(isinstance([], Q))
print(isinstance(42, Q))
print(isinstance("x", Q))


print("# section 6: metaclass __subclasscheck__ overrides issubclass")


class Meta2(type):
    def __subclasscheck__(cls, sub):
        return True


class R(metaclass=Meta2):
    pass


print(issubclass(int, R))
print(issubclass(list, R))


print("# section 7: ABCMeta-style virtual subclass via predicate")


class DuckMeta(type):
    def __instancecheck__(cls, inst):
        return hasattr(inst, "quack")


class Duck(metaclass=DuckMeta):
    pass


class Mallard:
    def quack(self):
        return "quack"


class NotADuck:
    pass


print(isinstance(Mallard(), Duck))
print(isinstance(NotADuck(), Duck))
print(isinstance(42, Duck))


print("# section 8: metaclass inherited via base class")


class Meta3(type):
    def __instancecheck__(cls, inst):
        return inst == "magic"


class S(metaclass=Meta3):
    pass


class T(S):
    pass


print(isinstance("magic", S))
print(isinstance("magic", T))
print(isinstance("nope", T))


print("# section 9: no metaclass → regular semantics still hold")


class U:
    pass


class V(U):
    pass


print(isinstance(V(), U))
print(isinstance(V(), V))
print(isinstance(42, U))
print(issubclass(V, U))
print(issubclass(U, V))

# Basic descriptor protocol: __get__, __set__, __delete__.

class Logged:
    def __init__(self, default=None):
        self.default = default
        self.name = None

    def __set_name__(self, owner, name):
        self.name = name

    def __get__(self, inst, owner):
        if inst is None:
            return self
        return inst.__dict__.get(self.name, self.default)

    def __set__(self, inst, value):
        inst.__dict__[self.name] = value

    def __delete__(self, inst):
        inst.__dict__.pop(self.name, None)


class Point:
    x = Logged(0)
    y = Logged(0)


p = Point()
print(p.x, p.y)
p.x = 3
p.y = 4
print(p.x, p.y)
del p.x
print(p.x, p.y)
print("desc names:", Point.x.name, Point.y.name)

# Accessed from the class, descriptor returns itself (inst is None).
print(isinstance(Point.x, Logged))


# Non-data descriptor: only __get__. Instance dict should win.
class NonData:
    def __get__(self, inst, owner):
        return "from-descriptor"


class Box:
    v = NonData()


b = Box()
print(b.v)
b.__dict__["v"] = "from-dict"
print(b.v)


# Data descriptor wins over instance dict.
class Always:
    def __get__(self, inst, owner):
        return "always-descriptor"

    def __set__(self, inst, value):
        inst.__dict__["_v"] = value


class Cell:
    v = Always()


c = Cell()
c.v = 99
print(c.v, c.__dict__)


# __class_getitem__: Foo[int] generic syntax.
class Generic:
    def __class_getitem__(cls, item):
        return f"{cls.__name__}[{item.__name__}]"


print(Generic[int])
print(Generic[str])


# __init_subclass__ sees the subclass.
class Base:
    subs = []

    def __init_subclass__(cls, **kwargs):
        Base.subs.append(cls.__name__)


class A(Base):
    pass


class B(Base):
    pass


print(Base.subs)

# Stress: descriptor protocol, class-creation hooks, and generic syntax.

# 1) Validating descriptor that enforces a type at __set__.
class Typed:
    def __init__(self, kind):
        self.kind = kind
        self.name = None

    def __set_name__(self, owner, name):
        self.name = "_" + name

    def __get__(self, inst, owner):
        if inst is None:
            return self
        try:
            return inst.__dict__[self.name]
        except KeyError:
            raise AttributeError(self.name[1:])

    def __set__(self, inst, value):
        if not isinstance(value, self.kind):
            raise TypeError(f"expected {self.kind.__name__}")
        inst.__dict__[self.name] = value

    def __delete__(self, inst):
        inst.__dict__.pop(self.name, None)


class Pet:
    name = Typed(str)
    age = Typed(int)


p = Pet()
p.name = "rex"
p.age = 4
print(p.name, p.age)
try:
    p.age = "old"
except TypeError as e:
    print("typeerr:", e)
try:
    Pet().age
except AttributeError as e:
    print("attrerr:", e)
del p.name
try:
    p.name
except AttributeError as e:
    print("after-del:", e)


# 2) Descriptor on base class, inherited by subclass.
class BaseDesc:
    x = Typed(int)


class SubDesc(BaseDesc):
    pass


s = SubDesc()
s.x = 7
print(s.x)
print(BaseDesc.x.name)  # set on the class where the descriptor was instantiated.


# 3) Descriptor combined with property on the same class.
class Combo:
    t = Typed(int)

    def __init__(self):
        self._temp = 0

    @property
    def temp(self):
        return self._temp

    @temp.setter
    def temp(self, v):
        self._temp = v * 2

    @temp.deleter
    def temp(self):
        self._temp = 0


c = Combo()
c.t = 10
c.temp = 5
print(c.t, c.temp)
del c.temp
print(c.temp)


# 4) __init_subclass__ chain through the MRO.
class Root:
    log = []

    def __init_subclass__(cls, tag=None, **kw):
        Root.log.append((cls.__name__, tag))


class Mid(Root, tag="mid"):
    pass


class Leaf(Mid, tag="leaf"):
    pass


print(Root.log)


# 5) __class_getitem__ with tuple keys for multi-parameter generics.
class Pair:
    def __class_getitem__(cls, item):
        if isinstance(item, tuple):
            return f"{cls.__name__}[{', '.join(t.__name__ for t in item)}]"
        return f"{cls.__name__}[{item.__name__}]"


print(Pair[int, str])
print(Pair[int])


# 6) Slot-like storage using a munged name: data descriptor wins over dict.
class Slot:
    def __init__(self, name):
        self.name = "__slot_" + name

    def __get__(self, inst, owner):
        if inst is None:
            return self
        return inst.__dict__.get(self.name, 0)

    def __set__(self, inst, value):
        inst.__dict__[self.name] = value


class Vec:
    x = Slot("x")


v = Vec()
v.x = 42
# Raw attempt to shadow via __dict__ — data descriptor still wins.
v.__dict__["x"] = "shadow"
print(v.x, v.__dict__["x"], v.__dict__["__slot_x"])


# 7) Delete-restoring-default: __delete__ resets to a default, subsequent get
# yields that default.
class WithDefault:
    def __init__(self, default):
        self.default = default
        self.name = None

    def __set_name__(self, owner, name):
        self.name = "_" + name

    def __get__(self, inst, owner):
        if inst is None:
            return self
        return inst.__dict__.get(self.name, self.default)

    def __set__(self, inst, value):
        inst.__dict__[self.name] = value

    def __delete__(self, inst):
        inst.__dict__[self.name] = self.default


class Config:
    timeout = WithDefault(30)


cfg = Config()
print(cfg.timeout)
cfg.timeout = 1
print(cfg.timeout)
del cfg.timeout
print(cfg.timeout)


# 8) Non-data descriptor yields to instance dict even when present on base.
class NDesc:
    def __get__(self, inst, owner):
        return "desc-value"


class Parent:
    m = NDesc()


class Child(Parent):
    pass


ch = Child()
print(ch.m)
ch.__dict__["m"] = "instance-value"
print(ch.m)


# 9) __set_name__ receives the defining class.
class Tracker:
    def __set_name__(self, owner, name):
        self.owner_name = owner.__name__
        self.attr_name = name


class Host:
    t = Tracker()


print(Host.t.owner_name, Host.t.attr_name)


# 10) __class_getitem__ returns a callable / preserves access on return value.
class Registry:
    items = {}

    def __class_getitem__(cls, key):
        Registry.items[key] = True
        return cls


Registry["a"]
Registry["b"]
print(sorted(Registry.items))


# 11) Descriptor __get__ called with owner=None equivalent when accessed from
# instance; verify the `owner` arg is the class.
class Probe:
    def __get__(self, inst, owner):
        return ("inst" if inst is not None else "noinst", owner.__name__)


class Use:
    p = Probe()


u = Use()
print(u.p)
print(Use.p)


# 12) __init_subclass__ with kwargs keeps unknown kwargs for cooperative super.
class KW:
    captured = []

    def __init_subclass__(cls, **kw):
        KW.captured.append(sorted(kw.items()))


class KWSub(KW, a=1, b=2):
    pass


print(KW.captured)


# 13) Descriptor chaining: property fset built via @name.setter.
class Temperature:
    def __init__(self):
        self._c = 0

    @property
    def c(self):
        return self._c

    @c.setter
    def c(self, v):
        self._c = int(v)


t = Temperature()
t.c = 25.9
print(t.c)

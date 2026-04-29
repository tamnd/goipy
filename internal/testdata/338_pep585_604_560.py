"""v0.0.338 PEP 585 / 604 / 560 — subscriptable builtins, union types,
__mro_entries__."""
import typing


# ── 1. PEP 585: list[int]
a = list[int]
print(repr(a))
assert a.__origin__ is list
assert a.__args__ == (int,)
print("list[int] ok")


# ── 2. PEP 585: dict[str, int]
d = dict[str, int]
print(repr(d))
assert d.__origin__ is dict
assert d.__args__ == (str, int)
print("dict[str,int] ok")


# ── 3. Re-subscription: dict[str, list[int]]
nested = dict[str, list[int]]
print(repr(nested))
assert nested.__origin__ is dict
print("re-subscription ok")


# ── 4. PEP 604: int | str
t = int | str
print(repr(t))
assert isinstance(1, t)
assert isinstance("hello", t)
assert not isinstance(1.0, t)
print("int|str ok")


# ── 5. Three-way union flattens
u = (int | str) | float
print(repr(u))
assert isinstance(1, u)
assert isinstance("x", u)
assert isinstance(1.0, u)
assert not isinstance([], u)
print("flatten ok")


# ── 6. list[int] | None — alias mixed with type. CPython forbids
# isinstance() against a parameterized generic, so check structure only.
mix = list[int] | None
print(repr(mix))
assert len(mix.__args__) == 2
print("alias|None ok")


# ── 7. PEP 560: __mro_entries__ rewrites bases
class G:
    def __mro_entries__(self, bases):
        return (object,)


g = G()


class C(g):
    pass


print([cls.__name__ for cls in C.__mro__])
print([cls.__name__ for cls in C.__bases__])
assert object in C.__bases__
print("mro_entries ok")


# ── 8. User class participates in unions via installed __or__
class MyCls:
    pass


u2 = MyCls | int
print(isinstance(MyCls(), u2), isinstance(7, u2), isinstance("x", u2))
print("user-class union ok")


# ── 9. typing.List[int] regression (still works alongside builtins)
tl = typing.List[int]
assert tl.__origin__ is list
assert tl.__args__ == (int,)
print("typing.List ok")

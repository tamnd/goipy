"""Tests for all built-in constants per docs.python.org/3/library/constants.html."""

# --- False ---
print(False)
print(type(False).__name__)
print(False == 0)
print(False + 1)
print(not False)
print(bool(0) is False)

# --- True ---
print(True)
print(type(True).__name__)
print(True == 1)
print(True + 1)
print(not True)
print(bool(1) is True)

# --- None ---
print(None)
print(type(None).__name__)
print(None is None)
print(None == None)
print(bool(None))
x = None
print(x is None)

# --- NotImplemented ---
print(NotImplemented)
print(type(NotImplemented).__name__)
print(NotImplemented is NotImplemented)


class MyNum:
    def __init__(self, v):
        self.v = v

    def __add__(self, other):
        if isinstance(other, MyNum):
            return MyNum(self.v + other.v)
        return NotImplemented

    def __radd__(self, other):
        return NotImplemented

    def __repr__(self):
        return f"MyNum({self.v})"


a = MyNum(3)
b = MyNum(4)
print(a + b)
try:
    a + "x"
except TypeError:
    print("TypeError: unsupported operand")


# --- Ellipsis / ... ---
print(Ellipsis)
print(...)
print(type(Ellipsis).__name__)
print(Ellipsis is ...)
print(... == Ellipsis)
print(repr(Ellipsis))
print(repr(...))


def accept_ellipsis(x):
    if x is ...:
        return "got ellipsis"
    return "other"


print(accept_ellipsis(...))
print(accept_ellipsis(None))


class Grid:
    def __getitem__(self, key):
        if key is Ellipsis:
            return "all"
        return key


g = Grid()
print(g[...])
print(g[1])


# --- __debug__ ---
print(__debug__)
print(type(__debug__).__name__)
print(bool(__debug__))

assert True
assert 1 == 1
assert "non-empty"

try:
    assert False, "this fails"
except AssertionError as e:
    print("AssertionError:", e)

try:
    assert 1 == 2
except AssertionError:
    print("AssertionError: no message")


# --- Interaction: constants in comparisons and bool context ---
print(True is not False)
print(None is not NotImplemented)
print(Ellipsis is not None)

# --- Constants as dict keys ---
d = {True: "true", False: "false", None: "none"}
print(d[True])
print(d[False])
print(d[None])

# --- Constants in collections ---
t = (None, True, False, Ellipsis, NotImplemented)
print(len(t))
print(t[0] is None)
print(t[1] is True)
print(t[2] is False)
print(t[3] is Ellipsis)
print(t[4] is NotImplemented)

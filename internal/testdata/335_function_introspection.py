"""v0.0.335 function introspection — surface __code__, __defaults__,
__kwdefaults__, __annotations__, __qualname__, __closure__, __globals__,
__module__, __dict__ on Function (and forward through BoundMethod)."""


def f(a, b=10, *, c=20) -> int:
    """docstring"""
    return a


# ── 1. core attributes
assert f.__name__ == 'f'
assert f.__qualname__ == 'f'
assert f.__doc__ == 'docstring'
assert f.__defaults__ == (10,), f.__defaults__
assert f.__kwdefaults__ == {'c': 20}, f.__kwdefaults__
assert f.__annotations__ == {'return': int}, f.__annotations__
assert f.__code__ is not None
assert f.__code__.co_name == 'f'
print("function attrs ok")


# ── 2. function with no defaults / kwdefaults / closure → None
def g(x):
    return x

assert g.__defaults__ is None, g.__defaults__
assert g.__kwdefaults__ is None, g.__kwdefaults__
assert g.__closure__ is None, g.__closure__
print("none defaults ok")


# ── 3. closure cells
def make():
    captured = 7
    def inner():
        return captured
    return inner

inner = make()
assert inner.__closure__ is not None
assert len(inner.__closure__) == 1
print("closure ok")


# ── 4. __globals__ is the module's globals
assert isinstance(f.__globals__, dict)
assert f.__globals__.get('__name__') == __name__
print("globals ok")


# ── 5. __module__ derived from globals['__name__']
assert f.__module__ == __name__, f.__module__
print("module ok")


# ── 6. __dict__ round-trip
assert f.__dict__ == {}
f.tag = 'hello'
assert f.tag == 'hello'
assert f.__dict__ == {'tag': 'hello'}
print("dict ok")


# ── 7. BoundMethod forwarding
class C:
    def m(self, x=42) -> int:
        """method doc"""
        return x

c = C()
assert c.m.__name__ == 'm'
assert c.m.__defaults__ == (42,)
assert c.m.__doc__ == 'method doc'
assert c.m.__self__ is c
# __func__ is the underlying function, callable on its own.
assert c.m.__func__ is not None
assert c.m.__func__(c, 99) == 99
print("boundmethod ok")


# ── 8. typing.get_type_hints reads __annotations__
import typing
hints = typing.get_type_hints(f)
assert hints == {'return': int}, hints
print("type hints ok")


print("ok")

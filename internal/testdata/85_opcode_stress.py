"""Systematic exercise of all major opcode groups in CPython 3.14."""
import asyncio
import sys


# --- 1. Stack ops: POP_TOP, COPY, SWAP ---
x = 1
_ = x  # POP_TOP via discard

a, b = 10, 20
a, b = b, a  # SWAP via tuple unpack
print(a, b)


# --- 2. Constants: LOAD_SMALL_INT, LOAD_CONST ---
print(0, 1, 255, 256, -1, -5, 1000)
print(3.14, 1+2j, "hello", b"bytes", None, True, False)


# --- 3. Arithmetic: BINARY_OP ---
print(3 + 4, 10 - 3, 6 * 7, 10 / 4)
print(10 // 3, 10 % 3, 2 ** 8)
print(0b1010 & 0b1100, 0b1010 | 0b0101, 0b1010 ^ 0b1100)
print(1 << 4, 256 >> 3)
print(10 @ 0 if False else "no matmul")


# --- 4. Comparisons ---
print(1 < 2, 2 <= 2, 3 > 2, 3 >= 3)
print(1 == 1, 1 != 2)
x1, x2 = 1, 1
print(x1 is x2, [] is not [])
x = [1, 2, 3]
print(1 in x, 4 not in x)


# --- 5. Logical ---
print(not True, not False, not 0, not 1)
print(-5, -(-3))
print(~0, ~1, ~(-1))


# --- 6. Augmented assignment ---
n = 10
n += 5
n -= 3
n *= 2
print(n)


# --- 7. Names and globals ---
_global_var = "global"
print(_global_var)


def show_global():
    global _global_var
    return _global_var


print(show_global())


# --- 8. Fast locals ---
def fast_locals(x, y):
    z = x + y
    return z


print(fast_locals(3, 4))


# --- 9. Closures ---
def make_counter(start=0):
    count = [start]
    def inc():
        count[0] += 1
        return count[0]
    return inc


c = make_counter(10)
print(c(), c(), c())


# --- 10. Attributes ---
class Vec2:
    def __init__(self, x, y):
        self.x = x
        self.y = y

    def length_sq(self):
        return self.x ** 2 + self.y ** 2

    def __repr__(self):
        return f"Vec2({self.x}, {self.y})"


v = Vec2(3, 4)
print(v.x, v.y)
print(v.length_sq())
del v.x
print(hasattr(v, "x"))


# --- 11. Subscripting ---
lst = [10, 20, 30, 40, 50]
print(lst[0], lst[-1])
print(lst[1:4])
print(lst[::2])
lst[0] = 99
print(lst[0])
del lst[0]
print(lst[0])


# --- 12. Builders ---
t = (1, 2, 3)
l = [4, 5, 6]
d = {"a": 1, "b": 2}
s = {7, 8, 9}
print(t, l, d, sorted(s))
print("x" + "y" + "z")
sl = slice(1, 5, 2)
print(sl)


# --- 13. Iterators and comprehensions ---
print(list(range(5)))
print([x * 2 for x in range(4)])
print({x: x ** 2 for x in range(4)})
print(sum(x for x in range(10)))


# --- 14. Unpacking ---
a, b, c = 1, 2, 3
print(a, b, c)
first, *rest = [10, 20, 30, 40]
print(first, rest)
*init, last = [1, 2, 3, 4]
print(init, last)


# --- 15. Jumps and conditionals ---
def sign(n):
    if n > 0:
        return "positive"
    elif n < 0:
        return "negative"
    else:
        return "zero"


print(sign(5), sign(-3), sign(0))

val = None
result = "got none" if val is None else "not none"
print(result)


# --- 16. Loops ---
total = 0
for i in range(5):
    if i == 3:
        continue
    total += i
print(total)

n = 5
fact = 1
while n > 0:
    fact *= n
    n -= 1
print(fact)


# --- 17. Generators ---
def gen_range(n):
    for i in range(n):
        yield i


print(list(gen_range(5)))


def gen_fib():
    a, b = 0, 1
    while True:
        yield a
        a, b = b, a + b


fibs = []
g = gen_fib()
for _ in range(8):
    fibs.append(next(g))
print(fibs)


# --- 18. Functions with *args and **kwargs ---
def variadic(*args, **kwargs):
    return args, sorted(kwargs.items())


print(variadic(1, 2, 3, x=4, y=5))


# --- 19. Classes and inheritance ---
class Shape:
    def area(self):
        return 0

    def __repr__(self):
        return f"{type(self).__name__}()"


class Rectangle(Shape):
    def __init__(self, w, h):
        self.w = w
        self.h = h

    def area(self):
        return self.w * self.h

    def __repr__(self):
        return f"Rectangle({self.w}, {self.h})"


class Square(Rectangle):
    def __init__(self, side):
        super().__init__(side, side)


r = Rectangle(3, 4)
sq = Square(5)
print(r.area(), sq.area())
print(isinstance(sq, Rectangle))
print(isinstance(sq, Shape))


# --- 20. Exceptions ---
def safe_div(a, b):
    try:
        return a / b
    except ZeroDivisionError:
        return "div by zero"
    finally:
        pass


print(safe_div(10, 2))
print(safe_div(10, 0))


def multi_except(x):
    try:
        if x == 0:
            raise ValueError("zero!")
        if x < 0:
            raise TypeError("negative!")
        return x
    except ValueError as e:
        return f"ValueError: {e}"
    except TypeError as e:
        return f"TypeError: {e}"


print(multi_except(5))
print(multi_except(0))
print(multi_except(-1))


# --- 21. Context managers ---
class Ctx:
    def __init__(self, name):
        self.name = name

    def __enter__(self):
        print(f"enter {self.name}")
        return self

    def __exit__(self, *_):
        print(f"exit {self.name}")
        return False


with Ctx("A") as ctx:
    print(f"inside {ctx.name}")


# --- 22. Async / coroutines ---
class AsyncRange:
    def __init__(self, n):
        self.n = n
        self.i = 0

    def __aiter__(self):
        return self

    async def __anext__(self):
        if self.i >= self.n:
            raise StopAsyncIteration
        v = self.i
        self.i += 1
        return v


async def async_add(a, b):
    return a + b


async def async_main():
    r = await async_add(3, 4)
    print(r)

    total = 0
    async for x in AsyncRange(5):
        total += x
    print(total)


asyncio.run(async_main())


# --- 23. Pattern matching ---
def match_shape(shape):
    match shape:
        case {"kind": "circle", "r": r}:
            return f"circle r={r}"
        case {"kind": "rect", "w": w, "h": h}:
            return f"rect {w}x{h}"
        case _:
            return "unknown"


print(match_shape({"kind": "circle", "r": 5}))
print(match_shape({"kind": "rect", "w": 3, "h": 4}))
print(match_shape({"kind": "triangle"}))


# --- 24. Formatting ---
name = "world"
pi = 3.14159
print(f"hello {name}")
print(f"pi = {pi:.3f}")
print(f"{name!r}")
print(f"{42:05d}")


# --- 25. Imports ---
import math
print(math.floor(3.7))
print(math.ceil(3.2))
from math import sqrt
print(sqrt(16))


# --- 26. String and bytes ---
s = "hello world"
print(s.upper())
print(s.split())
print(s.replace("world", "python"))

b = b"hello"
print(len(b), b[0])
print(b.decode("utf-8"))


# --- 27. List and dict operations ---
lst = [3, 1, 4, 1, 5, 9, 2]
lst.sort()
print(lst)
print(sorted([5, 2, 8], reverse=True))
d = {"a": 1}
d.update({"b": 2, "c": 3})
print(sorted(d.items()))


# --- 28. Lambda and higher-order ---
double = lambda x: x * 2
print(list(map(double, range(5))))
print(list(filter(lambda x: x % 2 == 0, range(8))))

nums = [3, 1, 4, 1, 5]
nums.sort(key=lambda x: -x)
print(nums)


# --- 29. Type conversions ---
print(int("42"), float("3.14"), str(123), bool(0), bool(1))
print(list((1, 2, 3)), tuple([4, 5, 6]))
print(set([1, 2, 2, 3]))


# --- 30. t-strings (PEP 750) ---
x = 42
msg = "ok"
t = t"value={x} {msg!r}"
print(type(t).__name__)
print(t.interpolations[0].value)
print(t.interpolations[1].conversion)
print(list(t))

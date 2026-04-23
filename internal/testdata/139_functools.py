import functools
import operator

# ===== WRAPPER_ASSIGNMENTS and WRAPPER_UPDATES =====
print(functools.WRAPPER_ASSIGNMENTS)
print(functools.WRAPPER_UPDATES)

# ===== update_wrapper =====
def orig():
    """original doc"""
    pass

def wrapper_fn(*a, **kw):
    return orig(*a, **kw)

functools.update_wrapper(wrapper_fn, orig)
print(wrapper_fn.__name__)              # orig
print(wrapper_fn.__wrapped__ is orig)   # True

# ===== reduce =====
print(functools.reduce(operator.add, [1, 2, 3, 4]))      # 10
print(functools.reduce(operator.mul, [1, 2, 3, 4], 1))   # 24
print(functools.reduce(max, [3, 1, 4, 1, 5]))             # 5
print(functools.reduce(operator.add, [], 0))              # 0
try:
    functools.reduce(operator.add, [])
except TypeError as e:
    print('reduce empty ok')

# ===== partial: .func .args .keywords =====
def power(base, exp):
    return base ** exp

sq = functools.partial(power, exp=2)
print(sq.func is power)       # True
print(sq.args)                # ()
print(list(sq.keywords.items()))  # [('exp', 2)]
print(sq(3))                  # 9
print(sq(5))                  # 25

add5 = functools.partial(operator.add, 5)
print(add5.args)              # (5,)
print(add5(3))                # 8

# partial with both positional and keyword
greet = functools.partial('{} {}!'.format, 'Hello')
print(greet('World'))         # Hello World!

# ===== Placeholder =====
from functools import Placeholder as _
f = functools.partial(power, _, 3)   # power(x, 3) = x**3
print(f(2))    # 8
print(f(3))    # 27

# multiple placeholders
def sub3(a, b, c):
    return a - b - c

g = functools.partial(sub3, _, _, 1)
print(g(10, 3))   # 10 - 3 - 1 = 6

# ===== cmp_to_key =====
data = ['banana', 'apple', 'cherry', 'date']
def cmp_len(a, b):
    return len(a) - len(b)

print(sorted(data, key=functools.cmp_to_key(cmp_len)))
# ['date', 'apple', 'banana', 'cherry']

def reverse_cmp(a, b):
    return b - a
print(sorted([3, 1, 4, 1, 5, 9], key=functools.cmp_to_key(reverse_cmp)))
# [9, 5, 4, 3, 1, 1]

# ===== total_ordering =====
@functools.total_ordering
class Version:
    def __init__(self, v):
        self.v = v
    def __eq__(self, other):
        return self.v == other.v
    def __lt__(self, other):
        return self.v < other.v

v1 = Version(1)
v2 = Version(2)
v3 = Version(1)
print(v1 < v2)    # True
print(v1 > v2)    # False
print(v1 <= v3)   # True
print(v2 >= v1)   # True
print(v1 == v3)   # True
print(v2 > v1)    # True

@functools.total_ordering
class Weight:
    def __init__(self, w):
        self.w = w
    def __eq__(self, other):
        return self.w == other.w
    def __le__(self, other):
        return self.w <= other.w

w1 = Weight(10)
w2 = Weight(20)
w3 = Weight(10)
print(w1 < w2)    # True
print(w2 > w1)    # True
print(w1 >= w3)   # True

# ===== partialmethod =====
class MyNum:
    def __init__(self, val):
        self.val = val

    def _op(self, factor, offset=0):
        return self.val * factor + offset

    double = functools.partialmethod(_op, 2)
    triple = functools.partialmethod(_op, 3)
    double_plus_ten = functools.partialmethod(_op, 2, offset=10)

x = MyNum(5)
print(x.double())            # 10
print(x.triple())            # 15
print(x.double_plus_ten())   # 20

# ===== singledispatch =====
@functools.singledispatch
def process(arg):
    return f'default: {type(arg).__name__}'

@process.register(int)
def _(arg):
    return f'int: {arg}'

@process.register(str)
def _(arg):
    return f'str: {arg}'

print(process(42))        # int: 42
print(process('hello'))   # str: hello
print(process([1, 2]))    # default: list
print(process(3.14))      # default: float

# register(type, fn) form
def handle_tuple(arg):
    return f'tuple: {len(arg)}'

process.register(tuple, handle_tuple)
print(process((1, 2, 3)))  # tuple: 3

# singledispatch function name preserved
print(process.__name__)   # process

# ===== singledispatchmethod =====
class Fmt:
    @functools.singledispatchmethod
    def fmt(self, arg):
        return f'obj: {arg}'

    @fmt.register(int)
    def _(self, arg):
        return f'int: {arg}'

    @fmt.register(str)
    def _(self, arg):
        return f'str: {arg}'

f = Fmt()
print(f.fmt(99))           # int: 99
print(f.fmt('hi'))         # str: hi
print(f.fmt([1, 2, 3]))    # obj: [1, 2, 3]

# ===== lru_cache: cache_parameters, typed =====
@functools.lru_cache(maxsize=4)
def fib(n):
    if n <= 1:
        return n
    return fib(n-1) + fib(n-2)

print(fib(10))                       # 55
params = fib.cache_parameters()
print(params['maxsize'])             # 4
print(params['typed'])               # False

@functools.lru_cache(maxsize=10, typed=True)
def typed_sq(n):
    return n * n

print(typed_sq(4))                   # 16
print(typed_sq(4.0))                 # 16.0
params2 = typed_sq.cache_parameters()
print(params2['maxsize'])            # 10
print(params2['typed'])              # True

# cache_info maxsize=None for unbounded cache
@functools.lru_cache(maxsize=None)
def unbounded(n):
    return n + 1

print(unbounded(5))                  # 6
info = unbounded.cache_info()
print(info[2] is None)               # True (maxsize=None)

# ===== wraps: copies __name__ =====
def decorator(fn):
    @functools.wraps(fn)
    def inner(*args, **kwargs):
        return fn(*args, **kwargs)
    return inner

@decorator
def compute(x):
    """Computes double."""
    return x * 2

print(compute.__name__)              # compute
print(compute.__wrapped__ is not None)  # True
print(compute(21))                   # 42

print('done')

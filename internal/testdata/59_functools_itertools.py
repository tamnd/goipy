import functools
import itertools

# --- functools basics ---

# reduce with and without initial.
print(functools.reduce(lambda a, b: a + b, [1, 2, 3, 4, 5]))
print(functools.reduce(lambda a, b: a + b, [1, 2, 3], 10))
print(functools.reduce(max, [3, 1, 4, 1, 5, 9, 2, 6]))

# partial with positional and keyword binding.
inc = functools.partial(lambda x, y: x + y, 1)
print(inc(10), inc(20))

def greet(greeting, name):
    return f"{greeting}, {name}!"

hi = functools.partial(greet, "Hi")
print(hi("Alice"))

to_base = functools.partial(int, base=16)
print(to_base("ff"), to_base("10"))

# lru_cache simple memoization.
calls = []

@functools.lru_cache(maxsize=128)
def square(n):
    calls.append(n)
    return n * n

print(square(3), square(3), square(4), square(3))
print(calls)

# cache (unbounded) variant.
@functools.cache
def fact(n):
    return 1 if n <= 1 else n * fact(n - 1)

print(fact(5), fact(10))

# wraps: preserves __name__.
def decorate(fn):
    @functools.wraps(fn)
    def wrapper(*a, **kw):
        return fn(*a, **kw)
    return wrapper

@decorate
def sample():
    """doc"""
    return 42

print(sample.__name__)
print(sample())

# cached_property.
class Sides:
    def __init__(self, n):
        self.n = n
        self.computes = 0

    @functools.cached_property
    def squares(self):
        self.computes += 1
        return [i * i for i in range(self.n)]

s = Sides(5)
print(s.squares)
print(s.squares)
print(s.computes)

# --- itertools basics ---

# count + islice
print(list(itertools.islice(itertools.count(10, 2), 5)))

# cycle + islice
print(list(itertools.islice(itertools.cycle("AB"), 6)))

# repeat
print(list(itertools.repeat("x", 3)))
print(list(itertools.islice(itertools.repeat(0), 4)))

# chain
print(list(itertools.chain([1, 2], (3, 4), "ab")))
print(list(itertools.chain.from_iterable([[1, 2], [3, 4], [5]])))

# compress
print(list(itertools.compress("ABCDE", [1, 0, 1, 0, 1])))

# dropwhile / takewhile
print(list(itertools.dropwhile(lambda x: x < 3, [1, 2, 3, 4, 1])))
print(list(itertools.takewhile(lambda x: x < 3, [1, 2, 3, 4, 1])))

# starmap
print(list(itertools.starmap(pow, [(2, 3), (3, 2), (10, 0)])))

# zip_longest
print(list(itertools.zip_longest([1, 2, 3], "ab", fillvalue="-")))

# product
print(list(itertools.product([1, 2], "ab")))

# permutations / combinations
print(list(itertools.permutations([1, 2, 3], 2)))
print(list(itertools.combinations([1, 2, 3, 4], 2)))
print(list(itertools.combinations_with_replacement([1, 2, 3], 2)))

# accumulate
print(list(itertools.accumulate([1, 2, 3, 4])))
print(list(itertools.accumulate([1, 2, 3, 4], lambda a, b: a * b)))

# pairwise
print(list(itertools.pairwise([1, 2, 3, 4])))

# filterfalse
print(list(itertools.filterfalse(lambda x: x % 2 == 0, range(6))))

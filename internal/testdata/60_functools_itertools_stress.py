import functools
import itertools

# --- functools stress ---

# 1) reduce: error on empty without initial, explicit initial works.
try:
    functools.reduce(lambda a, b: a + b, [])
except TypeError as e:
    print("reduce-empty:", e)
print(functools.reduce(lambda a, b: a + b, [], 0))
print(functools.reduce(lambda a, b: a * b, range(1, 6), 1))

# 2) partial merging: call-time kwargs override bound ones.
def f(a, b, c=0, d=0):
    return (a, b, c, d)

p = functools.partial(f, 1, c=10, d=20)
print(p(2))
print(p(2, c=99))  # call-time c overrides bound
print(p(2, c=3, d=4))

# 3) partial of a partial.
double_inc = functools.partial(functools.partial(f, 1), 2)
print(double_inc(3, 4))

# 4) lru_cache eviction: maxsize=2.
seen = []

@functools.lru_cache(maxsize=2)
def id2(x):
    seen.append(x)
    return x

id2("a")
id2("b")
id2("a")  # hit
id2("c")  # evicts "b"
id2("b")  # miss again
print(seen)

# 5) lru_cache with kwargs.
@functools.lru_cache(maxsize=32)
def kw(a, b=0):
    return a + b

kw(1, b=2)
kw(1, b=2)  # hit
kw(1, b=3)  # miss (different key)
info = kw.cache_info()
print("hits,misses,maxsize,size:", info[0], info[1], info[2], info[3])

# 6) cache_clear resets the stats.
kw.cache_clear()
info = kw.cache_info()
print("cleared:", info[0], info[1], info[3])

# 7) wraps sets __wrapped__ for tracing.
def tag(fn):
    @functools.wraps(fn)
    def inner(*a, **k):
        return fn(*a, **k)
    return inner

@tag
def greet(name):
    """hello doc"""
    return f"Hi {name}"

print(greet.__name__, greet.__wrapped__.__name__)

# 8) cached_property: independent state per instance.
class Counter:
    def __init__(self, base):
        self.base = base
        self.calls = 0

    @functools.cached_property
    def doubled(self):
        self.calls += 1
        return self.base * 2

a = Counter(10)
b = Counter(100)
print(a.doubled, b.doubled, a.doubled, b.doubled)
print(a.calls, b.calls)

# 9) cached_property can be overridden by deleting the instance entry.
computes = []

class C:
    @functools.cached_property
    def x(self):
        computes.append("hit")
        return [1, 2, 3]

c = C()
v1 = c.x
v2 = c.x
print(v1 is v2)
del c.__dict__["x"]  # force recompute
v3 = c.x
print(v3 is v1)
print(computes)

# --- itertools stress ---

# 10) islice with start/stop/step.
print(list(itertools.islice(range(20), 2, 15, 3)))

# 11) islice with None stop -> full rest.
print(list(itertools.islice(range(5), 2, None)))

# 12) product with repeat.
print(list(itertools.product([0, 1], repeat=3)))

# 13) product of empty iterable.
print(list(itertools.product()))
print(list(itertools.product([])))

# 14) permutations with r > len -> empty.
print(list(itertools.permutations([1, 2], 3)))
print(list(itertools.permutations([1, 2, 3])))
print(list(itertools.permutations([1, 2, 3], 0)))

# 15) combinations edges.
print(list(itertools.combinations([1, 2], 0)))
print(list(itertools.combinations([1, 2], 3)))

# 16) accumulate with initial.
print(list(itertools.accumulate([1, 2, 3, 4], initial=100)))

# 17) accumulate with max function.
print(list(itertools.accumulate([3, 1, 4, 1, 5, 9, 2, 6], max)))

# 18) chain.from_iterable with generator expression.
print(list(itertools.chain.from_iterable(range(i) for i in range(4))))

# 19) groupby with key fn.
words = ["apple", "ant", "banana", "berry", "cat"]
for k, grp in itertools.groupby(words, key=lambda w: w[0]):
    print(k, list(grp))

# 20) tee: two independent views of the same stream.
it1, it2 = itertools.tee([1, 2, 3, 4])
print(list(it1))
print(list(it2))

# 21) zip_longest with no fill.
print(list(itertools.zip_longest("abc", [1, 2])))

# 22) takewhile stops at first false, dropwhile drops only prefix.
print(list(itertools.takewhile(bool, [1, 1, 0, 1, 1])))
print(list(itertools.dropwhile(bool, [1, 1, 0, 1, 1])))

# 23) pairwise on short sequence.
print(list(itertools.pairwise([1])))
print(list(itertools.pairwise([])))

# 24) starmap of a builtin.
print(list(itertools.starmap(divmod, [(7, 2), (10, 3), (5, 5)])))

# 25) filterfalse with None predicate (falsy passes).
print(list(itertools.filterfalse(None, [0, 1, 2, 0, 3])))

# 26) combinations_with_replacement edges.
print(list(itertools.combinations_with_replacement("AB", 3)))

# 27) count with float step.
print(list(itertools.islice(itertools.count(0.5, 0.25), 4)))

# 28) cycle of empty -> empty iter.
print(list(itertools.islice(itertools.cycle([]), 3)))

# iter(callable, sentinel) + slice.indices + functools._CacheInfo

print("# section 1: iter(callable, sentinel)")

counter = [0]


def inc():
    counter[0] += 1
    return counter[0]


print(list(iter(inc, 3)))


# Sentinel never reached → consumer must stop on its own.
counter2 = [0]


def inc2():
    counter2[0] += 1
    return counter2[0]


it = iter(inc2, 999)
out = []
for _ in range(5):
    out.append(next(it))
print(out)


# Sentinel matches first call → empty iterator.
def always_zero():
    return 0


print(list(iter(always_zero, 0)))


print("# section 2: slice.indices(length)")

print(slice(1, 5, 2).indices(10))
print(slice(None, None, -1).indices(5))
print(slice(-100, 100).indices(5))
print(slice(None, None).indices(7))
print(slice(2, None).indices(10))
print(slice(None, 5).indices(10))
print(slice(10, 0, -2).indices(15))
print(slice(-3, None).indices(10))
print(slice(None, -3).indices(10))


# Step zero must raise.
try:
    slice(0, 5, 0).indices(10)
except ValueError as e:
    print("ValueError:", e)


print("# section 3: functools.lru_cache _CacheInfo")

import functools


@functools.lru_cache(maxsize=None)
def f(n):
    return n * 2


for i in range(5):
    f(i)
f(0)
f(1)
info = f.cache_info()
print(info.hits)
print(info.misses)
print(info.maxsize)
print(info.currsize)
print(type(info).__name__)
print(info[0], info[1], info[2], info[3])


@functools.lru_cache(maxsize=2)
def g(n):
    return n


g(1)
g(2)
g(3)
g(1)
info2 = g.cache_info()
print(info2.hits, info2.misses, info2.maxsize, info2.currsize)


print("# section 4: lru_cache _CacheInfo equals tuple")

info3 = f.cache_info()
print(info3 == (info3.hits, info3.misses, info3.maxsize, info3.currsize))

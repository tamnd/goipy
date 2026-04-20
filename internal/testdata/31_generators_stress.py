# Basic yield + return value
def collect(n):
    for i in range(n):
        yield i * i
    return "done"

it = collect(4)
print(list(it))

# Return value via StopIteration
def with_value():
    yield 1
    yield 2
    return "final"

g = with_value()
print(next(g))
print(next(g))
try:
    next(g)
except StopIteration as e:
    print("ret", e.args[0])

# .send() with two-way communication
def accumulator():
    total = 0
    while True:
        x = yield total
        if x is None:
            return total
        total += x

g = accumulator()
print(next(g))      # 0
print(g.send(5))    # 5
print(g.send(7))    # 12
print(g.send(3))    # 15
try:
    g.send(None)
except StopIteration as e:
    print("final", e.args[0])

# yield from various iterables
def flatten(xs):
    for x in xs:
        if isinstance(x, list):
            yield from flatten(x)
        else:
            yield x

print(list(flatten([1, [2, [3, 4], 5], 6, [[7, 8]], 9])))

# yield from range / tuple / str
def chain_everything():
    yield from range(3)
    yield from (10, 20)
    yield from "ab"

print(list(chain_everything()))

# Nested for over generator
def squares(n):
    for i in range(n):
        yield i * i

total = 0
for v in squares(5):
    total += v
print(total)

# Generator expression via for — but expressions use genexp syntax
# which is PR #5's secondary goal. Here we just verify iter()+next().
g = squares(3)
it = iter(g)
print(next(it))
print(next(it))
print(next(it))
try:
    next(it)
except StopIteration:
    print("exhausted")

# Early exit via close()
def counter():
    i = 0
    while True:
        yield i
        i += 1

g = counter()
print(next(g))
print(next(g))
g.close()
try:
    next(g)
except StopIteration:
    print("closed")

# Multiple generators interleaved
def letters():
    yield "a"
    yield "b"
    yield "c"

def numbers():
    yield 1
    yield 2
    yield 3

l, n = letters(), numbers()
pairs = list(zip(l, n))
print(pairs)

# yield from that captures the delegated return value
def sub():
    yield 1
    yield 2
    return "subret"

def outer():
    r = yield from sub()
    yield r
    yield "done"

print(list(outer()))


# Recursive yield from with return accumulation
def walk(node):
    if isinstance(node, list):
        total = 0
        for kid in node:
            total += yield from walk(kid)
        return total
    yield node
    return node

def driver(node):
    total = yield from walk(node)
    yield ("total", total)

print(list(driver([1, [2, [3, 4], 5], 6])))


# sum/min/max/any/all over a generator
def evens(n):
    for i in range(n):
        if i % 2 == 0:
            yield i

print(sum(evens(10)))
print(min(evens(10)))
print(max(evens(10)))
print(any(x > 5 for x in evens(10)))
print(all(x >= 0 for x in evens(10)))


# Generator expressions in various contexts
xs = [1, 2, 3, 4, 5]
print(sum(x * x for x in xs))
print(list(x for x in xs if x % 2))
print(dict((k, k * k) for k in range(4)))
print({x % 3 for x in range(10)})


# Nested generator expressions
print(sum(sum(x * y for y in range(3)) for x in range(3)))


# Generator consumed twice — second pass gives nothing
g = (x * 2 for x in range(3))
print(list(g))
print(list(g))


# Cross-generator: one generator feeds another via send loop
def doubler():
    while True:
        x = yield
        if x is None:
            return
        yield x * 2

def run_doubler(values):
    d = doubler()
    next(d)
    out = []
    for v in values:
        out.append(d.send(v))
        next(d)  # re-enter the top of the while
    return out

print(run_doubler([3, 5, 7]))


# StopIteration value flows through zip/tee-like composition
def first_n(src, n):
    for i, v in enumerate(src):
        if i >= n:
            return i
        yield v

def src():
    i = 0
    while True:
        yield i
        i += 1

print(list(first_n(src(), 5)))


# Generator inside a class method
class Counter:
    def __init__(self, n):
        self.n = n
    def values(self):
        for i in range(self.n):
            yield i

c = Counter(4)
print(list(c.values()))
print(sum(c.values()))


# enumerate + generator
def letters():
    yield "a"
    yield "b"
    yield "c"

for i, ch in enumerate(letters(), start=10):
    print(i, ch)

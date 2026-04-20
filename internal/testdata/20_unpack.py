a, b, c = [1, 2, 3]
print(a, b, c)

a, *rest = [1, 2, 3, 4]
print(a, rest)

*init, last = (10, 20, 30, 40)
print(init, last)

head, *mid, tail = range(6)
print(head, mid, tail)

def f(a, b, c):
    return a + b + c

args = [1, 2, 3]
print(f(*args))

def g(**kw):
    return sorted(kw.items())

print(g(**{"x": 1, "y": 2}))

first, *rest = "hello"
print(first, rest)

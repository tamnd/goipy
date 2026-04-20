def add(a, b):
    return a + b
print(add(2, 3))

def greet(name, greeting="Hello"):
    return greeting + ", " + name
print(greet("World"))
print(greet("Python", greeting="Hi"))

def varargs(*args, **kw):
    return (sum(args), sorted(kw.items()))
print(varargs(1, 2, 3, x=10, y=20))

def fact(n):
    if n <= 1:
        return 1
    return n * fact(n - 1)
print(fact(10))

def make_adder(x):
    def inner(y):
        return x + y
    return inner
add5 = make_adder(5)
print(add5(3))
print(add5(10))

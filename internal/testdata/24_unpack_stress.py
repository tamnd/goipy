# Starred unpacking in every position
a, b, c = 1, 2, 3
print(a, b, c)

a, *b = 1, 2, 3, 4
print(a, b)

*a, b = 1, 2, 3, 4
print(a, b)

a, *b, c = 1, 2, 3, 4, 5
print(a, b, c)

a, *b, c = [1, 2]
print(a, b, c)

# Nested
(a, b), (c, d) = (1, 2), (3, 4)
print(a, b, c, d)

(a, *b), c = [1, 2, 3], 9
print(a, b, c)

# In for-loops
for a, *b in [[1, 2, 3], [4, 5], [6]]:
    print(a, b)

# Splat function calls
def add3(x, y, z):
    return x + y + z
print(add3(*[1, 2, 3]))
print(add3(*(1, 2), 3))
print(add3(1, *[2, 3]))

# Double-star
def kw(**k):
    return sorted(k.items())
print(kw(**{"a": 1, "b": 2}))
print(kw(a=1, **{"b": 2}))

# List/tuple/set/dict literals with stars
print([*range(3), *range(5, 7)])
print((*"ab", *"cd"))
print({*[1, 2], *[2, 3]})
print({**{"a": 1}, **{"b": 2}})

# Mixed splat + kwargs + positional
def f(a, b, c, *, d=0):
    return (a, b, c, d)
print(f(*[1], **{"b": 2, "c": 3, "d": 4}))

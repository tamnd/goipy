# Complex number stress: pow, hash, bool, collections, equality edges.

# Power with int exponent
print((1 + 2j) ** 2)
print((1 + 2j) ** 3)
print((1 + 2j) ** 0)
print((2 + 0j) ** 10)

# Power with complex exponent (log-polar path)
z = (1 + 1j) ** (2 + 0j)
print(round(z.real, 6), round(z.imag, 6))

# Truthiness
print(bool(0j))
print(bool(0 + 0j))
print(bool(1j))
print(bool(1 + 0j))

# Equality across numeric types
print(1 + 0j == 1)
print(1 + 0j == 1.0)
print(complex(0) == 0)
print(complex(0) == False)
print((1 + 1j) == (1 + 1j))
print((1 + 1j) == (1 - 1j))

# Hashing / dict keys (equal values share slots)
d = {}
d[1 + 2j] = "a"
d[1 + 2j] = "b"
d[3 + 4j] = "c"
print(len(d))
print(d[1 + 2j], d[3 + 4j])

# Membership on list of complex
xs = [1 + 0j, 0 + 1j, 2 + 3j]
print((0 + 1j) in xs)
print((9 + 9j) in xs)

# conjugate chain and sanity
z = 3 + 4j
print(z.conjugate().conjugate() == z)
print(abs(z) * abs(z))

# Mixed arithmetic coercion
print((1 + 2j) + True)
print((1 + 2j) * 2)
print((1 + 2j) / 2)
print(2 / (1 + 1j))

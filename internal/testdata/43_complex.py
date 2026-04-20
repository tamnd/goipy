# Literals survive the marshal round-trip as proper complex values.
a = 1 + 2j
b = 3 - 4j
print(a, b)
print(repr(a), repr(b))
print(repr(2j), repr(0j), repr(-2j))

# complex() constructor
print(complex(5))
print(complex(5, 7))
print(complex(0, 0))

# .real / .imag / .conjugate()
print(a.real, a.imag)
print(b.conjugate())

# Arithmetic across complex/int/float
print(a + b)
print(a - b)
print(a * b)
print(a / b)
print(-a)
print(a + 10)
print(10 - b)
print(2.5 * a)

# Coercion from int/float
print(1 + 0j == 1)
print(1 + 0.5j == 1)
print(complex(2) == 2.0)

# abs() uses the modulus
print(abs(3 + 4j))
print(abs(0j))

# Mixed type arithmetic returns complex
x = 2
y = 0.5j
print(x + y)
print(type(x + y).__name__)

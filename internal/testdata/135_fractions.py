from fractions import Fraction
import math

# ===== Construction =====
print(Fraction())                    # 0
print(Fraction(3, 4))               # 3/4
print(Fraction(-3, 4))              # -3/4
print(Fraction(3, -4))              # -3/4
print(Fraction(6, 4))               # 3/2  (auto-reduced)
print(Fraction(-6, -4))             # 3/2
print(Fraction(5))                   # 5
print(Fraction(-7))                  # -7
print(Fraction(0))                   # 0
print(Fraction(True))                # 1
print(Fraction(False))               # 0

# ===== String construction =====
print(Fraction('3/4'))              # 3/4
print(Fraction('-1/2'))             # -1/2
print(Fraction('5'))                # 5
print(Fraction('1.5'))              # 3/2
print(Fraction('-0.25'))            # -1/4
print(Fraction('1.0e2'))            # 100
print(Fraction('2.5e-1'))           # 1/4

# ===== Float construction =====
print(Fraction(0.5))                # 1/2
print(Fraction(0.25))               # 1/4
print(Fraction(1.5))                # 3/2
print(Fraction(0.1))                # 3602879701896397/36028797018963968

# ===== Two-arg with Fractions =====
print(Fraction(Fraction(1, 2), Fraction(1, 3)))  # 3/2
print(Fraction(Fraction(3, 4), 2))               # 3/8

# ===== Properties =====
f = Fraction(3, 4)
print(f.numerator)                  # 3
print(f.denominator)                # 4
f2 = Fraction(-5, 6)
print(f2.numerator)                 # -5
print(f2.denominator)               # 6

# ===== repr and str =====
print(repr(Fraction(3, 4)))         # Fraction(3, 4)
print(repr(Fraction(5)))            # Fraction(5, 1)
print(str(Fraction(3, 4)))          # 3/4
print(str(Fraction(5)))             # 5
print(str(Fraction(-3, 4)))         # -3/4

# ===== Arithmetic =====
a = Fraction(1, 2)
b = Fraction(1, 3)
print(a + b)                        # 5/6
print(a - b)                        # 1/6
print(a * b)                        # 1/6
print(a / b)                        # 3/2
print(a + 1)                        # 3/2
print(1 + a)                        # 3/2
print(a - 1)                        # -1/2
print(1 - a)                        # 1/2
print(a * 2)                        # 1
print(2 * a)                        # 1
print(a / 2)                        # 1/4
print(2 / a)                        # 4

# ===== Floor division and modulo =====
print(Fraction(7, 2) // Fraction(1, 1))   # 3
print(Fraction(7, 2) % Fraction(1, 1))    # 1/2
print(Fraction(-7, 2) // Fraction(1, 1))  # -4
print(Fraction(-7, 2) % Fraction(1, 1))   # 1/2

# ===== Power =====
print(Fraction(2, 3) ** 2)          # 4/9
print(Fraction(2, 3) ** -1)         # 3/2
print(Fraction(2, 3) ** 0)          # 1
print(Fraction(2, 3) ** 3)          # 8/27
print(Fraction(4, 9) ** Fraction(1, 2))  # float: 0.666...

# ===== Unary =====
print(-Fraction(3, 4))              # -3/4
print(+Fraction(3, 4))              # 3/4
print(abs(Fraction(-3, 4)))         # 3/4
print(abs(Fraction(3, 4)))          # 3/4

# ===== Comparison =====
print(Fraction(1, 2) == Fraction(2, 4))   # True
print(Fraction(1, 2) == Fraction(1, 3))   # False
print(Fraction(1, 3) < Fraction(1, 2))    # True
print(Fraction(1, 2) > Fraction(1, 3))    # True
print(Fraction(1, 2) <= Fraction(1, 2))   # True
print(Fraction(1, 2) >= Fraction(2, 3))   # False
print(Fraction(1, 2) == 0.5)             # True
print(Fraction(1, 2) < 1)                # True

# ===== Conversion =====
print(float(Fraction(1, 4)))        # 0.25
print(float(Fraction(1, 3)))        # 0.3333333333333333
print(int(Fraction(7, 2)))          # 3
print(int(Fraction(-7, 2)))         # -3
print(bool(Fraction(0)))            # False
print(bool(Fraction(1, 3)))         # True

# ===== math integration =====
print(math.floor(Fraction(7, 2)))   # 3
print(math.floor(Fraction(-7, 2)))  # -4
print(math.ceil(Fraction(7, 2)))    # 4
print(math.ceil(Fraction(-7, 2)))   # -3

# ===== round =====
print(round(Fraction(7, 2)))        # 4  (half-to-even: 3.5 → 4)
print(round(Fraction(5, 2)))        # 2  (half-to-even: 2.5 → 2)
print(round(Fraction(1, 3)))        # 0
print(round(Fraction(2, 3)))        # 1
print(round(Fraction(-7, 2)))       # -4 (half-to-even)
print(round(Fraction(7, 4), 1))     # 7/4 rounded to 1 decimal

# ===== as_integer_ratio =====
n, d = Fraction(3, 4).as_integer_ratio()
print(n, d)                          # 3 4
n, d = Fraction(-5, 6).as_integer_ratio()
print(n, d)                          # -5 6

# ===== is_integer =====
print(Fraction(4, 2).is_integer())   # True
print(Fraction(3, 2).is_integer())   # False
print(Fraction(0).is_integer())      # True

# ===== limit_denominator =====
print(Fraction(1, 3).limit_denominator(10))      # 1/3
print(Fraction(math.pi).limit_denominator(10))   # 22/7
print(Fraction(math.pi).limit_denominator(100))  # 311/99
print(Fraction(1.1).limit_denominator(10))       # 11/10
print(Fraction(1, 7).limit_denominator(6))       # 1/6

# ===== from_float =====
print(Fraction.from_float(0.5))     # 1/2
print(Fraction.from_float(1.5))     # 3/2
print(Fraction.from_float(2))       # 2

# ===== from_decimal =====
from decimal import Decimal
print(Fraction.from_decimal(Decimal('0.5')))   # 1/2
print(Fraction.from_decimal(Decimal('1.5')))   # 3/2
print(Fraction.from_decimal(Decimal('0.1')))   # 1/10
print(Fraction.from_decimal(3))                # 3

# ===== from_number (3.14+) =====
print(Fraction.from_number(Fraction(3, 4)))    # 3/4
print(Fraction.from_number(2))                 # 2
print(Fraction.from_number(0.5))               # 1/2

# ===== Large numbers =====
print(Fraction(10**18, 10**18 - 1))            # 1000000000000000000/999999999999999999
print(Fraction(10**18, 10**18 - 1).numerator)  # 1000000000000000000

print('done')

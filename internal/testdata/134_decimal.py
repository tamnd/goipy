import decimal
from decimal import (
    Decimal, getcontext, setcontext, localcontext,
    ROUND_UP, ROUND_DOWN, ROUND_CEILING, ROUND_FLOOR,
    ROUND_HALF_UP, ROUND_HALF_DOWN, ROUND_HALF_EVEN, ROUND_05UP,
    InvalidOperation, DivisionByZero, Overflow,
    DefaultContext, BasicContext, ExtendedContext,
)

# ===== Construction =====
print(Decimal('3.14'))         # 3.14
print(Decimal('0'))            # 0
print(Decimal('-5'))           # -5
print(Decimal('1E+2'))         # 1E+2
print(Decimal('1.5e3'))        # 1.5E+3
print(Decimal('Inf'))          # Infinity
print(Decimal('-Infinity'))    # -Infinity
print(Decimal('NaN'))          # NaN
print(Decimal('sNaN'))         # sNaN
print(Decimal(5))              # 5
print(Decimal(-3))             # -3
print(Decimal(0))              # 0

# ===== Arithmetic =====
a = Decimal('1.1')
b = Decimal('2.2')
print(a + b)                   # 3.3
print(b - a)                   # 1.1
print(a * b)                   # 2.42
print(Decimal('10') / Decimal('3'))   # 3.333333333333333333333333333

# ===== Integer division and modulo =====
print(Decimal('10') // Decimal('3'))  # 3
print(Decimal('10') % Decimal('3'))   # 1
print(Decimal('-10') % Decimal('3'))  # 2

# ===== Power =====
print(Decimal('2') ** Decimal('10'))  # 1024
print(Decimal('2') ** 3)              # 8
print(Decimal('0.1') ** 2)            # 0.01

# ===== Unary =====
print(-Decimal('3.14'))        # -3.14
print(+Decimal('3.14'))        # 3.14
print(abs(Decimal('-3.14')))   # 3.14

# ===== Comparison =====
print(Decimal('1.0') == Decimal('1.0'))   # True
print(Decimal('1.0') == Decimal('1.00'))  # True
print(Decimal('1') < Decimal('2'))        # True
print(Decimal('2') > Decimal('1'))        # True
print(Decimal('1') <= Decimal('1'))       # True
print(Decimal('1') >= Decimal('2'))       # False

# ===== String representation =====
print(str(Decimal('1.0')))       # 1.0
print(str(Decimal('1e10')))      # 1E+10
print(str(Decimal('0.001')))     # 0.001
print(str(Decimal('1.2e-7')))    # 1.2E-7
print(str(Decimal('12345678901234567890')))  # 12345678901234567890

# ===== Conversion =====
print(int(Decimal('3.9')))      # 3
print(int(Decimal('-3.9')))     # -3
print(float(Decimal('3.14')))   # 3.14

# ===== Context precision =====
getcontext().prec = 5
print(Decimal('1') / Decimal('3'))   # 0.33333
getcontext().prec = 28

# ===== Rounding =====
d = Decimal('2.675')
print(d.quantize(Decimal('0.01'), rounding=ROUND_HALF_EVEN))  # 2.67
print(d.quantize(Decimal('0.01'), rounding=ROUND_HALF_UP))    # 2.68
print(d.quantize(Decimal('0.01'), rounding=ROUND_UP))         # 2.68
print(d.quantize(Decimal('0.01'), rounding=ROUND_DOWN))       # 2.67
print(d.quantize(Decimal('0.01'), rounding=ROUND_CEILING))    # 2.68
print(d.quantize(Decimal('0.01'), rounding=ROUND_FLOOR))      # 2.67

# ===== Special rounding =====
print(Decimal('2.5').quantize(Decimal('1'), rounding=ROUND_HALF_EVEN))  # 2
print(Decimal('3.5').quantize(Decimal('1'), rounding=ROUND_HALF_EVEN))  # 4
print(Decimal('2.5').quantize(Decimal('1'), rounding=ROUND_HALF_UP))    # 3
print(Decimal('-2.5').quantize(Decimal('1'), rounding=ROUND_HALF_UP))   # -3
print(Decimal('2.5').quantize(Decimal('1'), rounding=ROUND_HALF_DOWN))  # 2

# ===== adjusted() =====
print(Decimal('123.45').adjusted())   # 4
print(Decimal('0.001').adjusted())    # -3
print(Decimal('1e10').adjusted())     # 10

# ===== as_tuple() =====
t = Decimal('3.14').as_tuple()
print(t.sign)      # 0
print(t.digits)    # (3, 1, 4)
print(t.exponent)  # -2

t = Decimal('-1.5').as_tuple()
print(t.sign)      # 1
print(t.digits)    # (1, 5)
print(t.exponent)  # -1

t = Decimal('Infinity').as_tuple()
print(t.sign)      # 0
print(t.exponent)  # F

t = Decimal('NaN').as_tuple()
print(t.exponent)  # n

# ===== normalize() =====
print(Decimal('1.10').normalize())     # 1.1
print(Decimal('1.00').normalize())     # 1
print(Decimal('0.00').normalize())     # 0
print(Decimal('100').normalize())      # 1E+2

# ===== sqrt() =====
print(Decimal('4').sqrt())            # 2
print(Decimal('2').sqrt())            # 1.414213562373095048801688724
print(Decimal('0').sqrt())            # 0

# ===== Comparison: compare() =====
print(Decimal('1').compare(Decimal('2')))   # -1
print(Decimal('2').compare(Decimal('2')))   # 0
print(Decimal('3').compare(Decimal('2')))   # 1

# ===== copy_sign() =====
print(Decimal('3.14').copy_sign(Decimal('-1')))   # -3.14
print(Decimal('-3.14').copy_sign(Decimal('1')))   # 3.14

# ===== is_* predicates =====
print(Decimal('1').is_finite())     # True
print(Decimal('Inf').is_finite())   # False
print(Decimal('Inf').is_infinite()) # True
print(Decimal('1').is_infinite())   # False
print(Decimal('NaN').is_nan())      # True
print(Decimal('1').is_nan())        # False
print(Decimal('NaN').is_qnan())     # True
print(Decimal('sNaN').is_snan())    # True
print(Decimal('1').is_signed())     # False
print(Decimal('-1').is_signed())    # True
print(Decimal('0').is_zero())       # True
print(Decimal('1').is_zero())       # False
print(Decimal('1').is_normal())     # True
print(Decimal('0').is_normal())     # False

# ===== max/min =====
print(Decimal('3').max(Decimal('5')))   # 5
print(Decimal('3').min(Decimal('5')))   # 3

# ===== to_integral_value() =====
print(Decimal('2.7').to_integral_value())   # 3
print(Decimal('2.3').to_integral_value())   # 2
print(Decimal('2.5').to_integral_value(rounding=ROUND_HALF_EVEN))  # 2

# ===== Rounding constants =====
print(ROUND_UP)         # ROUND_UP
print(ROUND_DOWN)       # ROUND_DOWN
print(ROUND_CEILING)    # ROUND_CEILING
print(ROUND_FLOOR)      # ROUND_FLOOR
print(ROUND_HALF_UP)    # ROUND_HALF_UP
print(ROUND_HALF_DOWN)  # ROUND_HALF_DOWN
print(ROUND_HALF_EVEN)  # ROUND_HALF_EVEN
print(ROUND_05UP)       # ROUND_05UP

# ===== Context =====
ctx = getcontext()
print(ctx.prec)          # 28

# ===== localcontext =====
with localcontext() as ctx2:
    ctx2.prec = 3
    print(Decimal('1') / Decimal('3'))   # 0.333

print(Decimal('1') / Decimal('3'))       # 0.3333333333333333333333333333

# ===== DefaultContext / BasicContext / ExtendedContext =====
print(DefaultContext.prec)    # 28
print(BasicContext.prec)      # 9
print(ExtendedContext.prec)   # 9

# ===== Exception hierarchy =====
try:
    Decimal('1') / Decimal('0')
except DivisionByZero:
    print('DivisionByZero caught')

try:
    Decimal('1') / Decimal('0')
except DivisionByZero:
    print('DivisionByZero2 caught')

print('done')

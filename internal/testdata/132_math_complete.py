import math

# ===== Constants =====
print(f"{math.pi:.10f}")    # 3.1415926536
print(f"{math.e:.10f}")     # 2.7182818285
print(f"{math.tau:.10f}")   # 6.2831853072
print(math.inf)             # inf
print(math.isnan(math.nan)) # True

# ===== Number-theoretic =====
print(math.comb(10, 3))         # 120
print(math.comb(5, 0))          # 1
print(math.comb(5, 6))          # 0
print(math.factorial(0))        # 1
print(math.factorial(10))       # 3628800
print(math.gcd(0))              # 0
print(math.gcd(12, 18))         # 6
print(math.gcd(12, 18, 24))     # 6
print(math.isqrt(0))            # 0
print(math.isqrt(1))            # 1
print(math.isqrt(8))            # 2
print(math.isqrt(9))            # 3
print(math.isqrt(10**20))       # 10000000000
print(math.lcm(4, 6))           # 12
print(math.lcm(3, 4, 5))        # 60
print(math.lcm(0, 5))           # 0
print(math.perm(5, 2))          # 20
print(math.perm(5, 0))          # 1
print(math.perm(5))             # 120

# ===== Floating point arithmetic =====
print(math.ceil(1.1))           # 2
print(math.ceil(-1.1))          # -1
print(math.fabs(-3.5))          # 3.5
print(math.floor(1.9))          # 1
print(math.floor(-1.9))         # -2
print(math.fma(2.0, 3.0, 4.0)) # 10.0
print(math.fmod(10.0, 3.0))    # 1.0
frac, whole = math.modf(3.75)
print(f"{frac:.2f}")            # 0.75
print(f"{whole:.1f}")           # 3.0
print(math.remainder(10.0, 3.0))  # 1.0
print(math.trunc(1.9))          # 1
print(math.trunc(-1.9))         # -1

# ===== Floating point manipulation =====
print(math.copysign(5.0, -2.0)) # -5.0
m, e = math.frexp(8.0)
print(f"{m} {e}")               # 0.5 4
print(math.isclose(0.1 + 0.2, 0.3))         # True
print(math.isclose(1.0, 2.0))               # False
print(math.isclose(1e-10, 0, abs_tol=1e-9)) # True
print(math.isfinite(1.0))       # True
print(math.isfinite(math.inf))  # False
print(math.isfinite(math.nan))  # False
print(math.isinf(math.inf))     # True
print(math.isinf(-math.inf))    # True
print(math.isinf(1.0))          # False
print(math.isnan(math.nan))     # True
print(math.isnan(1.0))          # False
print(math.ldexp(0.5, 4))       # 8.0
# nextafter: next float after x toward y
print(math.nextafter(1.0, 2.0) > 1.0)   # True
print(math.nextafter(1.0, 0.0) < 1.0)   # True
print(math.nextafter(0.0, 0.0))         # 0.0
# ulp: unit in the last place
print(math.ulp(1.0) > 0)        # True
print(math.ulp(0.0) > 0)        # True
print(math.isnan(math.ulp(math.nan)))  # True
print(math.isinf(math.ulp(math.inf))) # True

# ===== Power and logarithmic =====
print(f"{math.cbrt(27.0):.1f}")    # 3.0
print(f"{math.cbrt(-8.0):.1f}")    # -2.0
print(f"{math.exp(0.0):.1f}")      # 1.0
print(f"{math.exp(1.0):.5f}")      # 2.71828
print(f"{math.exp2(10.0):.1f}")    # 1024.0
print(f"{math.expm1(0.0):.1f}")    # 0.0
print(f"{math.log(math.e):.5f}")   # 1.00000
print(f"{math.log(8.0, 2.0):.5f}") # 3.00000
print(f"{math.log1p(0.0):.5f}")    # 0.00000
print(f"{math.log2(1024.0):.1f}")  # 10.0
print(f"{math.log10(1000.0):.1f}") # 3.0
print(f"{math.pow(2.0, 10.0):.1f}") # 1024.0
print(f"{math.sqrt(16.0):.1f}")    # 4.0

# ===== Summation and product =====
print(math.dist([0, 0], [3, 4]))    # 5.0
print(math.dist([1, 2, 3], [4, 6, 3]))  # 5.0
print(math.fsum([0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1]))  # 1.0
print(f"{math.hypot(3, 4):.1f}")   # 5.0
print(f"{math.hypot(1, 2, 2):.1f}") # 3.0
print(math.prod([1, 2, 3, 4]))      # 24
print(math.prod([2, 3], start=10))  # 60
print(math.sumprod([1, 2, 3], [4, 5, 6]))   # 32  (1*4 + 2*5 + 3*6)
print(math.sumprod([1.0, 2.0], [3.0, 4.0])) # 11.0

# ===== Angular conversion =====
print(f"{math.degrees(math.pi):.1f}")   # 180.0
print(f"{math.radians(180.0):.5f}")     # 3.14159

# ===== Trigonometric =====
print(f"{math.acos(1.0):.5f}")   # 0.00000
print(f"{math.asin(0.0):.5f}")   # 0.00000
print(f"{math.atan(1.0):.5f}")   # 0.78540
print(f"{math.atan2(1.0, 1.0):.5f}")  # 0.78540
print(f"{math.cos(0.0):.5f}")    # 1.00000
print(f"{math.sin(0.0):.5f}")    # 0.00000
print(f"{math.tan(0.0):.5f}")    # 0.00000

# ===== Hyperbolic =====
print(f"{math.acosh(1.0):.5f}")  # 0.00000
print(f"{math.asinh(0.0):.5f}")  # 0.00000
print(f"{math.atanh(0.0):.5f}")  # 0.00000
print(f"{math.cosh(0.0):.5f}")   # 1.00000
print(f"{math.sinh(0.0):.5f}")   # 0.00000
print(f"{math.tanh(0.0):.5f}")   # 0.00000

# ===== Special =====
print(f"{math.erf(1.0):.5f}")    # 0.84270
print(f"{math.erfc(1.0):.5f}")   # 0.15730
print(f"{math.gamma(5.0):.1f}")  # 24.0
print(f"{math.lgamma(5.0):.5f}") # 3.17805

# ===== Error cases =====
try:
    math.isqrt(-1)
except ValueError:
    print('isqrt(-1) ValueError')

try:
    math.factorial(-1)
except ValueError:
    print('factorial(-1) ValueError')

try:
    math.remainder(1.0, 0.0)
except ValueError:
    print('remainder(1,0) ValueError')

print('done')

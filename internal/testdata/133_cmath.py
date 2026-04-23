import cmath
import math

# ===== Constants =====
print(f"{cmath.pi:.10f}")        # 3.1415926536
print(f"{cmath.e:.10f}")         # 2.7182818285
print(f"{cmath.tau:.10f}")       # 6.2831853072
print(cmath.inf)                  # inf
print(cmath.isinf(cmath.infj))   # True
print(cmath.isnan(cmath.nanj))   # True

# ===== phase(z) =====
print(f"{cmath.phase(1+0j):.5f}")   # 0.00000
print(f"{cmath.phase(-1+0j):.5f}")  # 3.14159
print(f"{cmath.phase(0+1j):.5f}")   # 1.57080
print(f"{cmath.phase(0-1j):.5f}")   # -1.57080
print(f"{cmath.phase(1+1j):.5f}")   # 0.78540

# ===== polar(z) =====
r, phi = cmath.polar(1+0j)
print(f"{r:.5f} {phi:.5f}")          # 1.00000 0.00000
r, phi = cmath.polar(0+1j)
print(f"{r:.5f} {phi:.5f}")          # 1.00000 1.57080
r, phi = cmath.polar(3+4j)
print(f"{r:.5f} {phi:.5f}")          # 5.00000 0.92730

# ===== rect(r, phi) =====
z = cmath.rect(1.0, 0.0)
print(f"{z.real:.5f} {z.imag:.5f}") # 1.00000 0.00000
z = cmath.rect(5.0, cmath.phase(3+4j))
print(f"{z.real:.4f} {z.imag:.4f}") # 3.0000 4.0000
z = cmath.rect(0.0, math.pi)
print(f"{z.real:.5f}")               # 0.00000

# ===== exp(z) =====
z = cmath.exp(0+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 1.00000 0.00000
z = cmath.exp(1+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 2.71828 0.00000
z = cmath.exp(0+1j)                  # e^(1j) = cos(1) + i*sin(1)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.54030 0.84147

# ===== log(z[, base]) =====
z = cmath.log(1+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 0.00000
z = cmath.log(-1+0j)                 # log(-1) = i*pi
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 3.14159
z = cmath.log(cmath.e+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 1.00000 0.00000
z = cmath.log(8+0j, 2)
print(f"{z.real:.5f} {z.imag:.5f}") # 3.00000 0.00000

# ===== log10(z) =====
z = cmath.log10(100+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 2.00000 0.00000

# ===== sqrt(z) =====
z = cmath.sqrt(4+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 2.00000 0.00000
z = cmath.sqrt(-1+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 1.00000
z = cmath.sqrt(0+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 0.00000

# ===== Trigonometric =====
z = cmath.sin(0+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 0.00000
z = cmath.cos(0+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 1.00000 0.00000
z = cmath.tan(0+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 0.00000
z = cmath.asin(0+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 0.00000
z = cmath.acos(1+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 0.00000
z = cmath.atan(0+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 0.00000

# ===== Hyperbolic =====
z = cmath.sinh(0+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 0.00000
z = cmath.cosh(0+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 1.00000 0.00000
z = cmath.tanh(0+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 0.00000
z = cmath.asinh(0+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 0.00000
z = cmath.acosh(1+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 0.00000
z = cmath.atanh(0+0j)
print(f"{z.real:.5f} {z.imag:.5f}") # 0.00000 0.00000

# ===== Accept int/float input =====
z = cmath.sqrt(4)
print(f"{z.real:.5f} {z.imag:.5f}") # 2.00000 0.00000
z = cmath.sqrt(4.0)
print(f"{z.real:.5f} {z.imag:.5f}") # 2.00000 0.00000
z = cmath.exp(0)
print(f"{z.real:.5f} {z.imag:.5f}") # 1.00000 0.00000

# ===== Classification: isfinite =====
print(cmath.isfinite(1+2j))         # True
print(cmath.isfinite(float('inf')+0j))  # False
print(cmath.isfinite(0+float('inf')*1j)) # False
print(cmath.isfinite(float('nan')+0j))  # False
# Accept real inputs
print(cmath.isfinite(1.0))          # True
print(cmath.isfinite(float('inf'))) # False

# ===== Classification: isinf =====
print(cmath.isinf(1+2j))            # False
print(cmath.isinf(float('inf')+0j)) # True
print(cmath.isinf(0+float('inf')*1j)) # True
print(cmath.isinf(1.0))             # False

# ===== Classification: isnan =====
print(cmath.isnan(1+2j))            # False
print(cmath.isnan(float('nan')+0j)) # True
print(cmath.isnan(0+float('nan')*1j)) # True
print(cmath.isnan(1.0))             # False

# ===== isclose =====
print(cmath.isclose(1+2j, 1+2j))           # True
print(cmath.isclose(1+2j, 1+2.001j))       # False
print(cmath.isclose(1+2j, 1+2.0000000001j)) # True
print(cmath.isclose(1+0j, 2+0j))           # False
# NaN is never close
print(cmath.isclose(float('nan'), float('nan'))) # False
# inf is only close to itself
print(cmath.isclose(float('inf'), float('inf'))) # True
print(cmath.isclose(float('inf'), -float('inf'))) # False
# abs_tol
print(cmath.isclose(0j, 1e-10j, abs_tol=1e-9)) # True

# ===== Round-trip: polar -> rect =====
z_orig = 3+4j
r, phi = cmath.polar(z_orig)
z_back = cmath.rect(r, phi)
print(abs(z_back - z_orig) < 1e-10)  # True

print('done')

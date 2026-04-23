from numbers import Number, Complex, Real, Rational, Integral

# ===== int is Integral (and all supers) =====
print(isinstance(1, Integral))     # True
print(isinstance(1, Rational))     # True
print(isinstance(1, Real))         # True
print(isinstance(1, Complex))      # True
print(isinstance(1, Number))       # True

# ===== bool is Integral (bool subclasses int in Python) =====
print(isinstance(True, Integral))  # True
print(isinstance(True, Rational))  # True
print(isinstance(True, Real))      # True
print(isinstance(True, Complex))   # True
print(isinstance(True, Number))    # True

# ===== float is Real but not Rational or Integral =====
print(isinstance(1.5, Number))     # True
print(isinstance(1.5, Complex))    # True
print(isinstance(1.5, Real))       # True
print(isinstance(1.5, Rational))   # False
print(isinstance(1.5, Integral))   # False

# ===== complex is Complex but not Real/Rational/Integral =====
print(isinstance(1+2j, Number))    # True
print(isinstance(1+2j, Complex))   # True
print(isinstance(1+2j, Real))      # False
print(isinstance(1+2j, Rational))  # False
print(isinstance(1+2j, Integral))  # False

# ===== non-numeric types =====
print(isinstance("hi", Number))    # False
print(isinstance([], Number))      # False
print(isinstance(None, Number))    # False
print(isinstance({}, Number))      # False

# ===== zero int / zero float / zero complex =====
print(isinstance(0, Integral))     # True
print(isinstance(0.0, Real))       # True
print(isinstance(0j, Complex))     # True
print(isinstance(0j, Real))        # False

# ===== large int =====
print(isinstance(10**100, Integral))  # True
print(isinstance(10**100, Number))    # True

# ===== negative values =====
print(isinstance(-1, Integral))    # True
print(isinstance(-1.5, Real))      # True

# ===== register() — virtual subclass =====
class MyNumber:
    pass

Number.register(MyNumber)
print(isinstance(MyNumber(), Number))   # True
print(isinstance(MyNumber(), Complex))  # False (only registered on Number)

# ===== register() on Integral — instance passes Integral and all superclasses =====
class MyInt:
    pass

Integral.register(MyInt)
print(isinstance(MyInt(), Integral))    # True

# ===== class hierarchy: Integral is subclass of Number etc. =====
print(issubclass(Integral, Rational))  # True
print(issubclass(Integral, Real))      # True
print(issubclass(Integral, Complex))   # True
print(issubclass(Integral, Number))    # True
print(issubclass(Rational, Real))      # True
print(issubclass(Rational, Complex))   # True
print(issubclass(Rational, Number))    # True
print(issubclass(Real, Complex))       # True
print(issubclass(Real, Number))        # True
print(issubclass(Complex, Number))     # True

# ===== not subclass in reverse =====
print(issubclass(Number, Integral))    # False
print(issubclass(Real, Integral))      # False

print('done')

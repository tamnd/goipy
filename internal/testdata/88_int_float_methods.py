"""Tests for int and float methods and classmethods."""

# --- int methods ---
print((255).bit_length())
print((0).bit_length())
print((1).bit_length())
print((256).bit_length())

print((255).bit_count())
print((0).bit_count())
print((7).bit_count())

print((1024).to_bytes(2, "big"))
print((1024).to_bytes(2, "little"))
print((255).to_bytes(1, "big"))
print((-1).to_bytes(1, "big", signed=True))
print((-128).to_bytes(1, "big", signed=True))

print((5).conjugate())
print((5).numerator)
print((5).denominator)
print((5).real)
print((5).imag)

n, d = (3).as_integer_ratio()
print(n, d)

# --- int.from_bytes ---
print(int.from_bytes(b"\x04\x00", "big"))
print(int.from_bytes(b"\x00\x04", "little"))
print(int.from_bytes(b"\xff", "big", signed=True))
print(int.from_bytes(b"\xff", "big", signed=False))

# --- float methods ---
print((1.0).is_integer())
print((1.5).is_integer())
print((0.0).is_integer())

n, d = (0.5).as_integer_ratio()
print(n, d)

n, d = (1.5).as_integer_ratio()
print(n, d)

print((1.0).conjugate())
print((1.5).real)
print((1.5).imag)

print(float.fromhex("0x1.8p+1"))
print(float.fromhex("1.8p+1"))
print(float.fromhex("-0x1p0"))

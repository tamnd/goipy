# Int formats
for n in [0, 1, -1, 255, -255, 1234567]:
    print(f"{n:d}|{n:5d}|{n:05d}|{n:+d}|{n: d}|{n:,}")

# Binary / octal / hex
for n in [0, 42, 255, 4096]:
    print(f"{n:b}|{n:o}|{n:x}|{n:X}")

# Float precision & width
for x in [0.0, 3.14159, -2.5, 1e6, 1e-6]:
    print(f"{x:.2f}|{x:10.3f}|{x:.4e}|{x:g}")

# String alignment and fill
for s in ["a", "hi", "hello"]:
    print(f"[{s:>8}]|[{s:<8}]|[{s:^8}]|[{s:*^8}]|[{s:.2}]")

# Mixed in expressions
a, b, c = 1, 2.5, "z"
print(f"a={a:03d} b={b:06.2f} c={c:>5}")

# Conversions
xs = [1, 2, 3]
print(f"{xs!r}")
print(f"{xs!s}")

# Debug form
name = "go"
print(f"{name=}")
print(f"{a+1=}")

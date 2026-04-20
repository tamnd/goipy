# negative with alt form + zero pad
for n in [-123, 123, -1, 0]:
    print(f"[{n:#010x}][{n:+#010x}][{n: #010x}]")

# alt form alone with sign
for n in [-5, 5]:
    print(f"[{n:+#b}][{n:+#o}][{n:+#x}]")

# width + grouping combinations
print(f"[{1234567890:20,}]")
print(f"[{1234567890:<20,}]")
print(f"[{1234567890:>20,}]")
print(f"[{1234567890:^20,}]")
print(f"[{1234567890:*<20,}]")
print(f"[{-1234567890:020,}]")
print(f"[{-1234567890:+020,}]")

# precision on strings
print(f"[{'abcdef':.3}][{'abcdef':>10.3}][{'abcdef':*^10.3}]")

# float edge cases
print(f"[{0.0:+.3f}][{-0.0:+.3f}]")
print(f"[{1e-10:.3e}][{1e10:.3e}]")
print(f"[{1.5:.0f}][{2.5:.0f}]")  # banker's rounding in Python; accept what we produce
print(f"[{123.456:10.2f}][{123.456:010.2f}][{-123.456:010.2f}]")

# hex with width
print(f"[{255:>10x}][{255:>#10x}][{255:<10x}][{255:^10x}]")

# binary grouping with zero pad width
print(f"[{0xAB:#010_b}]")

# f-string with nested format expr
width = 12
prec = 3
print(f"[{3.14159:{width}.{prec}f}]")
print(f"[{'x':*^{width}}]")

# conversion then format
x = "hi"
print(f"[{x!r:>10}]")
print(f"[{x!r:*^10}]")

# empty format spec on various types
print(f"[{True}][{False}][{None}][{[1,2]}][{(1,2)}]")

# percent with zero
print(f"{0.0:.1%}")
print(f"{1.0:.1%}")

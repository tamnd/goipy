# alt form prefixes
for n in [0, 10, 255, -255]:
    print(f"{n:#b}|{n:#o}|{n:#x}|{n:#X}")

# grouping with underscore
print(f"{1234567:_}")
print(f"{1234567:_d}")
print(f"{0xFFFFFFFF:_x}")
print(f"{0xFFFFFFFF:_b}")
print(f"{0xFFFFFFFF:#_x}")
print(f"{0b11111111111111111111111111111111:#_b}")

# float grouping
print(f"{1234567.89:,.2f}")
print(f"{1234567.89:_.2f}")

# float %
print(f"{0.125:.1%}")
print(f"{0.125:%}")
print(f"{-0.5:+.2%}")

# n type (locale-aware; we treat as d/g)
print(f"{1234:n}")
print(f"{3.14159:.3n}")

# sign modifiers
for n in [5, 0, -5]:
    print(f"[{n:+d}][{n: d}][{n:-d}]")

# alt form with g
print(f"{3.0:#g}")
print(f"{3.14:#g}")

# char type
print(f"{65:c}{66:c}{67:c}")

# zero-pad + alt prefix keeps the prefix outside the pad (Python: 0x007b)
print(f"{123:#08x}")
print(f"{123:#b}")

# combine group + width + zero pad
print(f"{1234567:015,d}")

# debug format (f-string =)
x = 42
print(f"{x=}")
print(f"{x+1=}")

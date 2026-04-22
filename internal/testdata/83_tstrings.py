"""Test BUILD_INTERPOLATION and BUILD_TEMPLATE opcodes (PEP 750 t-strings)."""

name = "world"
count = 42

# Simple template with one interpolation
t = t"hello {name}"
print(type(t).__name__)
print(t.strings)
print(t.interpolations[0].value)
print(t.interpolations[0].expression)
print(t.interpolations[0].conversion)

# Template with format spec
pi = 3.14159
t2 = t"pi is {pi:.2f} ok"
print(t2.interpolations[0].value)
print(t2.interpolations[0].format_spec)
print(t2.interpolations[0].conversion)

# Template with conversion
msg = "raw <text>"
t3 = t"safe: {msg!r}"
print(t3.interpolations[0].conversion)

# Pure literal template (no interpolations)
t4 = t"no interpolations"
print(len(t4.strings))
print(len(t4.interpolations))
print(t4.strings[0])

# Multiple interpolations
a, b = 1, 2
t5 = t"{a} + {b} = {a + b}"
print(len(t5.interpolations))
print(t5.interpolations[0].value)
print(t5.interpolations[1].value)
print(t5.interpolations[2].value)
print(t5.interpolations[2].expression)

# Iteration yields interleaved str+Interpolation
items = list(t)
print(len(items))
print(items[0])
print(type(items[1]).__name__)

# values shortcut
print(t5.values)

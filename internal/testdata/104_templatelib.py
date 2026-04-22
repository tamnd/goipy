from string.templatelib import Template, Interpolation, convert

# --- Basic t-string ---
x = 42
t = t'value is {x}'
print(t.strings)
print(t.values)
print(t.interpolations[0].value)
print(t.interpolations[0].expression)
print(t.interpolations[0].conversion)
print(t.interpolations[0].format_spec)

# --- repr ---
print(repr(t))
print(repr(t.interpolations[0]))

# --- Empty t-string ---
e = t''
print(e.strings)
print(e.values)

# --- t-string starting with interpolation ---
name = 'Alice'
t2 = t'{name} says hi'
print(t2.strings)
print(t2.values)

# --- Multiple interpolations ---
a, b = 3, 5
t3 = t'{a} + {b} = {a+b}'
print(t3.strings)
print(t3.values)

# --- Consecutive interpolations ---
p, q = 'foo', 'bar'
t4 = t'{p}{q}'
print(t4.strings)
print(t4.values)

# --- Iteration ---
items = list(t3)
print(len(items))  # 5: '' skipped, interp, ' + ', interp, ' = ', interp
for item in t3:
    if isinstance(item, Interpolation):
        print('interp:', item.value)
    else:
        print('str:', repr(item))

# --- Conversion ---
s = 'hello'
tc = t'{s!r}'
print(tc.interpolations[0].conversion)
ta = t'{s!a}'
print(ta.interpolations[0].conversion)
ts = t'{s!s}'
print(ts.interpolations[0].conversion)

# --- format_spec ---
n = 3.14159
tf = t'{n:.2f}'
print(tf.interpolations[0].format_spec)

# --- Both conv + spec ---
tcs = t'{n!s:.4f}'
print(tcs.interpolations[0].conversion)
print(tcs.interpolations[0].format_spec)

# --- Template + concatenation ---
t5 = t'hello '
t6 = t'{name}!'
t7 = t5 + t6
print(t7.strings)
print(t7.values)

# --- Manual Template constructor ---
t8 = Template('start ', Interpolation(99, 'num'), ' end')
print(t8.strings)
print(t8.values)

# --- Consecutive strings merged in constructor ---
t9 = Template('a', 'b', 'c')
print(t9.strings)

# --- Consecutive interps in constructor ---
t10 = Template(Interpolation(1, 'x'), Interpolation(2, 'y'))
print(t10.strings)
print(t10.values)

# --- Manual Interpolation constructor ---
i1 = Interpolation('world', 'name', 'r', '.10s')
print(i1.value)
print(i1.expression)
print(i1.conversion)
print(i1.format_spec)
print(repr(i1))

# --- convert() ---
print(convert('hello', 's'))
print(convert('hello', 'r'))
print(convert('hello', 'a'))
print(convert(42, None))

# --- isinstance ---
print(isinstance(t, Template))
print(isinstance(t.interpolations[0], Interpolation))
print(isinstance('hi', Template))

# --- type names ---
print(type(t).__name__)
print(type(t.interpolations[0]).__name__)

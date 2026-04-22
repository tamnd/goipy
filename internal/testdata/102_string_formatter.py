from string import Formatter

fmt = Formatter()

# --- format() ---
print(fmt.format('{}, {}', 'hello', 'world'))     # hello, world
print(fmt.format('{0} {1}', 'foo', 'bar'))        # foo bar
print(fmt.format('{name}', name='Alice'))          # Alice
print(fmt.format('{:d}', 42))                     # 42
print(fmt.format('{:.2f}', 3.14159))              # 3.14
print(fmt.format('{:>8}', 'x'))                   # '       x'
print(fmt.format('{:^10}', 'hi'))                 # '    hi    '

# --- vformat() ---
result = fmt.vformat('{0} + {1} = {2}', (1, 2, 3), {})
print(result)                                     # 1 + 2 = 3
result2 = fmt.vformat('{x}', (), {'x': 99})
print(result2)                                    # 99

# --- format_field() ---
print(fmt.format_field(42, 'd'))                  # 42
print(fmt.format_field(3.14159, '.2f'))           # 3.14
print(fmt.format_field('hello', '>10'))           # '     hello'
print(fmt.format_field(255, '#x'))                # 0xff

# --- convert_field() ---
print(fmt.convert_field('test', 's'))             # test
print(fmt.convert_field('test', 'r'))             # 'test'
print(fmt.convert_field(42, 's'))                 # 42

# --- parse() ---
parsed = fmt.parse('hello {name} world')
for item in parsed:
    print(item)
# ('hello ', 'name', '', None)
# (' world', None, None, None)

parsed2 = fmt.parse('{0:.2f} and {1!r}')
for item in parsed2:
    print(item)
# ('', '0', '.2f', None)
# (' and ', '1', '', 'r')

# --- get_value() ---
print(fmt.get_value(0, ('a', 'b', 'c'), {}))     # a
print(fmt.get_value(2, ('a', 'b', 'c'), {}))     # c
print(fmt.get_value('key', (), {'key': 'val'}))  # val

# --- check_unused_args is a no-op ---
fmt.check_unused_args(set(), [], {})
print('check_unused_args ok')                     # check_unused_args ok

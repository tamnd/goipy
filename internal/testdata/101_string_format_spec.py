# Format string syntax and format spec mini-language.

# --- Positional and auto-numbering ---
print('{}, {}, {}'.format('a', 'b', 'c'))        # a, b, c
print('{0}, {1}, {2}'.format('a', 'b', 'c'))      # a, b, c
print('{2}, {1}, {0}'.format('a', 'b', 'c'))      # c, b, a
print('{0}{1}{0}'.format('abra', 'cad'))           # abracadabra

# --- Named args ---
print('{name} is {age}'.format(name='Alice', age=30))  # Alice is 30

# --- Attribute access ---
c = 3-5j
print('{0.real} {0.imag}'.format(c))              # 3.0 -5.0

# --- Item access ---
coord = (3, 5)
print('X:{0[0]} Y:{0[1]}'.format(coord))          # X:3 Y:5

lst = [10, 20, 30]
print('{0[1]}'.format(lst))                       # 20

# --- Conversions ---
print('{!s}'.format('test'))                      # test
print('{!r}'.format('test'))                      # 'test'

# --- Escape braces ---
print('{{}}'.format())                            # {}
print('{{0}}'.format())                           # {0}

# --- String alignment ---
print('{:<10}'.format('left'))                    # 'left      '
print('{:>10}'.format('right'))                   # '     right'
print('{:^10}'.format('mid'))                     # '   mid    '
print('{:*^10}'.format('mid'))                    # '***mid****'
print('{:-<10}'.format('x'))                      # 'x---------'

# --- Sign ---
print('{:+d}'.format(42))                         # +42
print('{:+d}'.format(-42))                        # -42
print('{: d}'.format(42))                         #  42
print('{: d}'.format(-42))                        # -42
print('{:-d}'.format(42))                         # 42
print('{:-d}'.format(-42))                        # -42

# --- Integer types ---
print('{:d}'.format(42))                          # 42
print('{:b}'.format(42))                          # 101010
print('{:o}'.format(42))                          # 52
print('{:x}'.format(42))                          # 2a
print('{:X}'.format(42))                          # 2A

# --- Alternate form ---
print('{:#b}'.format(42))                         # 0b101010
print('{:#o}'.format(42))                         # 0o52
print('{:#x}'.format(42))                         # 0x2a
print('{:#X}'.format(42))                         # 0X2A

# --- Width and zero-pad ---
print('{:8d}'.format(42))                         # '      42'
print('{:08d}'.format(42))                        # '00000042'
print('{:08b}'.format(42))                        # '00101010'

# --- Float: f, e, g, % ---
print('{:f}'.format(3.14))                        # 3.140000
print('{:.2f}'.format(3.14159))                   # 3.14
print('{:8.3f}'.format(3.14))                     # '   3.140'
print('{:e}'.format(123456.789))                  # 1.234568e+05
print('{:.2e}'.format(123456.789))                # 1.23e+05
print('{:g}'.format(123456.789))                  # 123457
print('{:g}'.format(0.000123))                    # 0.000123
print('{:.2%}'.format(19/22))                     # 86.36%

# --- Grouping ---
print('{:,}'.format(1234567890))                  # 1,234,567,890
print('{:_}'.format(1234567890))                  # 1_234_567_890
print('{:_b}'.format(42))                         # 10_1010
print('{:_x}'.format(255))                        # ff

# --- chr type ---
print('{:c}'.format(65))                          # A
print('{:c}'.format(9731))                        # ☃

# --- Width with nested arg ---
print('{0:{1}}'.format('hello', 10))              # 'hello     '
print('{0:>{1}}'.format('x', 5))                  # '    x'

# --- str type ---
print('{:s}'.format('hello'))                     # hello
print('{:10s}'.format('hi'))                      # 'hi        '
print('{:.3s}'.format('hello'))                   # hel

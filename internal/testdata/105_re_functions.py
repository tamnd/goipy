import re

# --- search ---
m = re.search(r'\d+', 'abc 123 def')
print(m.group())                          # 123

# --- match ---
m = re.match(r'\w+', 'hello world')
print(m.group())                          # hello
print(re.match(r'\d', 'hello') is None)   # True

# --- fullmatch ---
print(re.fullmatch(r'\w+', 'hello') is not None)   # True
print(re.fullmatch(r'\w+', 'hello world') is None)  # True

# --- findall ---
print(re.findall(r'\d+', 'a1 b22 c333'))            # ['1', '22', '333']
print(re.findall(r'(\w+)=(\d+)', 'a=1 b=22'))       # [('a', '1'), ('b', '22')]

# --- finditer ---
matches = [m.group() for m in re.finditer(r'\d+', 'a1 b2 c3')]
print(matches)                                       # ['1', '2', '3']

# --- split ---
print(re.split(r'\s+', 'one two  three'))            # ['one', 'two', 'three']
print(re.split(r'(\s+)', 'one two'))                 # ['one', ' ', 'two']
print(re.split(r',', 'a,b,c,d', maxsplit=2))         # ['a', 'b', 'c,d']
print(re.split(r'X', 'hello'))                       # ['hello']

# --- sub ---
print(re.sub(r'\d', 'X', 'a1b2c3'))                 # aXbXcX
print(re.sub(r'(\w+) (\w+)', r'\2 \1', 'hi there')) # there hi
print(re.sub(r'\d', 'X', 'a1b2c3', count=2))        # aXbXc3
print(re.sub(r'\d+', lambda m: str(int(m.group())*2), 'a1 b22'))  # a2 b44

# --- subn ---
print(re.subn(r'\d', 'X', 'a1b2'))                  # ('aXbX', 2)
print(re.subn(r'\d', 'X', 'abc'))                   # ('abc', 0)

# --- escape ---
print(re.escape('a.b*c'))                           # a\.b\*c
print(re.escape('(x+y)'))                           # \(x\+y\)

# --- purge (no-op) ---
re.purge()
print('purge ok')                                   # purge ok

# --- Flag constants ---
print(int(re.IGNORECASE))    # 2
print(int(re.I))             # 2
print(int(re.MULTILINE))     # 8
print(int(re.M))             # 8
print(int(re.DOTALL))        # 16
print(int(re.S))             # 16
print(int(re.VERBOSE))       # 64
print(int(re.X))             # 64
print(int(re.ASCII))         # 256
print(int(re.A))             # 256
print(int(re.UNICODE))       # 32
print(int(re.U))             # 32
print(int(re.NOFLAG))        # 0

# --- IGNORECASE ---
m = re.search(r'hello', 'HELLO WORLD', re.IGNORECASE)
print(m.group())             # HELLO

# --- MULTILINE ---
m = re.search(r'^\d+', 'text\n42', re.MULTILINE)
print(m.group())             # 42

# --- DOTALL ---
m = re.search(r'a.b', 'a\nb', re.DOTALL)
print(m.group())             # a\nb

# --- VERBOSE ---
p = re.compile(r'''
    (\d+)   # digits
    \s+     # whitespace
    (\w+)   # word
''', re.VERBOSE)
m = p.search('42 hello')
print(m.group(1), m.group(2))  # 42 hello

# --- re.error ---
try:
    re.compile('[invalid')
except re.error:
    print('re.error ok')     # re.error ok

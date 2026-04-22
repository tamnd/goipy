import re

# --- compile / Pattern attributes ---
p = re.compile(r'(\w+)\s+(\w+)')
print(p.pattern)             # (\w+)\s+(\w+)
print(p.groups)              # 2

# Named group index
p2 = re.compile(r'(?P<first>\w+)\s+(?P<last>\w+)')
gi = p2.groupindex
print(gi['first'])           # 1
print(gi['last'])            # 2

# --- Pattern methods ---
m = p.search('hello world!')
print(m.group(0))            # hello world
print(m.group(1))            # hello
print(m.group(2))            # world

m2 = p2.match('John Doe extra')
print(m2.group('first'))     # John
print(m2.group('last'))      # Doe

# fullmatch
print(p.fullmatch('foo bar') is not None)   # True
print(p.fullmatch('foo bar baz') is None)   # True

# Pattern.findall
print(p.findall('hello world, foo bar'))  # ['hello world', 'foo bar']

# Pattern.split
print(p.split('aaa hello world bbb'))    # ['aaa ', ' bbb']

# Pattern.sub
print(p.sub(r'\2 \1', 'hello world'))   # world hello

# Pattern.subn
print(p.subn(r'\2 \1', 'hello world')) # ('world hello', 1)

# Pattern.finditer
results = [m.group() for m in p.finditer('foo bar, baz qux')]
print(results)                          # ['foo bar', 'baz qux']

# --- Match object ---
m3 = re.search(r'(?P<first>\w+)\s+(?P<last>\w+)', 'John Doe')

# group variants
print(m3.group())              # John Doe
print(m3.group(0))             # John Doe
print(m3.group(1))             # John
print(m3.group('first'))       # John
print(m3.group('last'))        # Doe
print(m3.group(1, 2))          # ('John', 'Doe')

# groups / groupdict
print(m3.groups())             # ('John', 'Doe')
d = m3.groupdict()
print(d['first'])              # John
print(d['last'])               # Doe

# start / end / span
m4 = re.search(r'\d+', 'abc 123 def')
print(m4.start())              # 4
print(m4.end())                # 7
print(m4.span())               # (4, 7)
print(m4.start(0))             # 4
print(m4.span(0))              # (4, 7)

# string / re / pos / endpos
print(m4.string)               # abc 123 def
print(m4.re is not None)       # True
print(m4.pos)                  # 0
print(m4.endpos)               # 11

# lastindex / lastgroup
m5 = re.search(r'(\d+)', 'abc 42')
print(m5.lastindex)            # 1

p3 = re.compile(r'(?P<digits>\d+)|(?P<letters>[a-z]+)')
m6 = p3.search('abc')
print(m6.lastindex)            # 2
print(m6.lastgroup)            # letters

m7 = re.search(r'(\d+)', '42')
print(m7.lastgroup)            # None

# __getitem__ subscript
m8 = re.search(r'(?P<first>\w+)\s+(?P<last>\w+)', 'John Doe')
print(m8[0])                   # John Doe
print(m8[1])                   # John
print(m8['first'])             # John
print(m8['last'])              # Doe

# expand
m9 = re.match(r'(\w+) (\w+)', 'hello world')
print(m9.expand(r'\2 \1'))     # world hello

# groups() with default
m10 = re.match(r'(\w+)(\s+)?(\w*)', 'hello')
print(m10.groups())            # ('hello', None, '')
print(m10.groups(default='X')) # ('hello', 'X', '')

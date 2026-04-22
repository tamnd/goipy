import re

# --- Character classes ---
print(re.findall(r'[aeiou]', 'hello world'))       # ['e', 'o', 'o']
print(re.findall(r'[^aeiou\s]+', 'hello world'))   # ['h', 'll', 'w', 'rld']
print(re.findall(r'[a-z]+', 'Hello World'))        # ['ello', 'orld']

# --- Quantifiers ---
print(re.search(r'ab+c', 'abbbc').group())         # abbbc
print(re.search(r'ab*c', 'ac').group())            # ac
print(re.search(r'ab?c', 'ac').group())            # ac
print(re.search(r'a{3}', 'aaaa').group())          # aaa
print(re.search(r'a{2,4}', 'aaaaa').group())       # aaaa

# --- Non-greedy ---
print(re.search(r'<.+?>', '<a><b>').group())       # <a>
print(re.search(r'<.+>', '<a><b>').group())        # <a><b>

# --- Alternation ---
print(re.search(r'cat|dog', 'I have a dog').group())  # dog
print(re.search(r'cat|dog', 'I have a cat').group())  # cat

# --- Groups ---
m = re.match(r'(a)(b)(c)', 'abc')
print(m.group(0))                                  # abc
print(m.group(1, 2, 3))                            # ('a', 'b', 'c')

# Non-capturing group
m2 = re.search(r'(?:hello) (\w+)', 'hello world')
print(m2.group(1))                                 # world
print(m2.groups())                                 # ('world',)

# Named groups
m3 = re.match(r'(?P<y>\d{4})-(?P<m>\d{2})-(?P<d>\d{2})', '2024-01-15')
print(m3.group('y'))                               # 2024
print(m3.group('m'))                               # 01
print(m3.group('d'))                               # 15

# --- Anchors ---
print(re.match(r'hello', 'hello world').group())   # hello
print(re.search(r'world$', 'hello world').group()) # world
print(re.search(r'\Ahello', 'hello world').group())# hello
print(re.search(r'world\Z', 'hello world').group())# world

# Multiline anchors
print(re.findall(r'^\w+', 'hello\nworld', re.MULTILINE))  # ['hello', 'world']
print(re.findall(r'\w+$', 'hello\nworld', re.MULTILINE))  # ['hello', 'world']

# --- Special sequences ---
print(re.findall(r'\w+', 'hello-world 42'))        # ['hello', 'world', '42']
print(re.findall(r'\d+', 'price: $42.00'))         # ['42', '00']
print(re.findall(r'\s+', 'a b\tc'))                # [' ', '\t']
print(re.findall(r'\S+', 'one  two'))              # ['one', 'two']

# Word boundary
print(re.findall(r'\bcat\b', 'cat concatenation'))  # ['cat']
print(re.findall(r'\Bcat\B', 'concatenation'))      # ['cat']

# Dot matches any (except newline by default)
print(re.search(r'a.b', 'aXb').group())            # aXb
print(re.search(r'a.b', 'a\nb') is None)           # True

# DOTALL: dot matches newline
print(re.search(r'a.b', 'a\nb', re.DOTALL).group()) # a\nb

# --- findall with multiple groups ---
print(re.findall(r'(\d+)-(\d+)', '12-34 56-78'))   # [('12', '34'), ('56', '78')]

# Single group findall returns strings
print(re.findall(r'(\d+)', 'a1 b2 c3'))            # ['1', '2', '3']

# --- sub with group reference ---
print(re.sub(r'(\w+) (\w+)', r'\2 \1', 'foo bar')) # bar foo
print(re.sub(r'(?P<a>\w+) (?P<b>\w+)', r'\g<b> \g<a>', 'foo bar'))  # bar foo

# --- split with groups ---
print(re.split(r'(\W+)', 'one, two! three'))       # ['one', ', ', 'two', '! ', 'three']

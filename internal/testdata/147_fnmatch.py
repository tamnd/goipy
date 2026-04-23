import fnmatch
import re

# ===== fnmatch() — Unix: case-sensitive =====
print(fnmatch.fnmatch('foo.py', '*.py'))         # True
print(fnmatch.fnmatch('foo.txt', '*.py'))        # False
print(fnmatch.fnmatch('Foo.py', '*.py'))         # True  (case-sensitive, same case)
print(fnmatch.fnmatch('foo.py', '*.PY'))         # False (case-sensitive, different case)
print(fnmatch.fnmatch('a', '?'))                 # True
print(fnmatch.fnmatch('ab', '?'))                # False
print(fnmatch.fnmatch('', '?'))                  # False
print(fnmatch.fnmatch('', '*'))                  # True
print(fnmatch.fnmatch('anything', '*'))          # True
print(fnmatch.fnmatch('a/b/c.py', '*.py'))       # True  (* matches /  in fnmatch)
print(fnmatch.fnmatch('.hidden', '*'))           # True  (fnmatch has no hidden-file concept)
print(fnmatch.fnmatch('.hidden', '.*'))          # True

# ===== ? wildcard =====
print(fnmatch.fnmatch('abc', 'a?c'))             # True
print(fnmatch.fnmatch('ac', 'a?c'))              # False
print(fnmatch.fnmatch('abbc', 'a?c'))            # False
print(fnmatch.fnmatch('axc', 'a?c'))             # True
print(fnmatch.fnmatch('a/c', 'a?c'))             # True  (? matches /)

# ===== character classes =====
print(fnmatch.fnmatch('a1c', 'a[0-9]c'))         # True
print(fnmatch.fnmatch('abc', 'a[0-9]c'))         # False
print(fnmatch.fnmatch('azc', 'a[!0-9]c'))        # True
print(fnmatch.fnmatch('a5c', 'a[!0-9]c'))        # False
print(fnmatch.fnmatch('aac', 'a[abc]c'))         # True
print(fnmatch.fnmatch('adc', 'a[abc]c'))         # False
print(fnmatch.fnmatch('aAc', 'a[A-Z]c'))         # True
print(fnmatch.fnmatch('a!c', 'a[!]c'))           # False  (! at start = negation, no chars to match)
print(fnmatch.fnmatch('axc', 'a[]x]c'))          # True   (] inside class if first char)
print(fnmatch.fnmatch('a]c', 'a[]x]c'))          # True

# ===== fnmatchcase() — always case-sensitive =====
print(fnmatch.fnmatchcase('Foo.PY', '*.py'))     # False
print(fnmatch.fnmatchcase('foo.py', '*.py'))     # True
print(fnmatch.fnmatchcase('FOO.PY', '*.PY'))     # True
print(fnmatch.fnmatchcase('ABC', 'abc'))         # False

# ===== filter() =====
names = ['a.py', 'b.txt', 'c.py', 'd.md', 'e.py']
print(fnmatch.filter(names, '*.py'))             # ['a.py', 'c.py', 'e.py']
print(fnmatch.filter(names, '*.txt'))            # ['b.txt']
print(fnmatch.filter(names, '*'))                # all 5
print(fnmatch.filter([], '*.py'))               # []
print(fnmatch.filter(names, '?.py'))             # ['a.py', 'c.py', 'e.py'] (single-char stem)
print(fnmatch.filter(['a.py', 'ab.py'], '?.py')) # ['a.py']

# filter preserves order
names2 = ['z.py', 'a.py', 'm.py']
print(fnmatch.filter(names2, '*.py'))            # ['z.py', 'a.py', 'm.py']

# ===== translate() =====
# Returns a regex string; we verify it works with re module
tr = fnmatch.translate('*.py')
print(isinstance(tr, str))                       # True
print(bool(re.match(tr, 'foo.py')))             # True
print(bool(re.match(tr, 'foo.txt')))            # False
print(bool(re.match(tr, '.hidden.py')))         # True  (fnmatch translate matches hidden)
print(bool(re.match(tr, 'a/b.py')))             # True  (* matches /)

tr2 = fnmatch.translate('a?c')
print(bool(re.match(tr2, 'abc')))               # True
print(bool(re.match(tr2, 'ac')))                # False

tr3 = fnmatch.translate('a[0-9]c')
print(bool(re.match(tr3, 'a5c')))              # True
print(bool(re.match(tr3, 'abc')))              # False

tr4 = fnmatch.translate('*.txt')
print(bool(re.match(tr4, 'readme.txt')))       # True
print(bool(re.match(tr4, 'readme.py')))        # False

# translate pattern includes (?s:...) wrapper or similar
tr5 = fnmatch.translate('*')
print(bool(re.match(tr5, '')))                 # True
print(bool(re.match(tr5, 'anything')))         # True

# ===== edge cases =====
# Literal special regex chars in pattern
print(fnmatch.fnmatch('a.b', 'a.b'))           # True
print(fnmatch.fnmatch('axb', 'a.b'))           # False  (. is literal in fnmatch)
print(fnmatch.fnmatch('a+b', 'a+b'))           # True   (+ is literal)
print(fnmatch.fnmatch('a(b', 'a(b'))           # True   (( is literal)
print(fnmatch.fnmatch('a$b', 'a$b'))           # True   ($ is literal)
print(fnmatch.fnmatch('a^b', 'a^b'))           # True   (^ is literal)

# backslash handling — pattern with backslash matches literal backslash
print(fnmatch.fnmatch('a', 'a'))               # True
print(fnmatch.fnmatch('', ''))                 # True
print(fnmatch.fnmatch('x', ''))               # False

# Multiple * wildcards
print(fnmatch.fnmatch('foo.bar.py', '*.*.py')) # True
print(fnmatch.fnmatch('foo.py', '*.*.py'))     # False

# Pattern with no wildcards
print(fnmatch.fnmatch('exact', 'exact'))       # True
print(fnmatch.fnmatch('exact', 'Exact'))       # False (case-sensitive on Unix)

print('done')

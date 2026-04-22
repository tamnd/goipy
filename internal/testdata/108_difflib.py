import difflib

# --- IS_LINE_JUNK ---
print(difflib.IS_LINE_JUNK(''))         # True
print(difflib.IS_LINE_JUNK('\n'))       # True
print(difflib.IS_LINE_JUNK('# \n'))    # True
print(difflib.IS_LINE_JUNK('hello\n')) # False

# --- IS_CHARACTER_JUNK ---
print(difflib.IS_CHARACTER_JUNK(' '))  # True
print(difflib.IS_CHARACTER_JUNK('\t')) # True
print(difflib.IS_CHARACTER_JUNK('a'))  # False

# --- get_close_matches ---
print(difflib.get_close_matches('appel', ['ape', 'apple', 'peach', 'puppy']))  # ['apple', 'ape']
print(difflib.get_close_matches('wheel', ['while', 'lame', 'weld']))           # ['weld', 'while']
print(difflib.get_close_matches('apple', ['ape', 'apple', 'peach'], n=1))      # ['apple']
print(difflib.get_close_matches('xyz', ['abc', 'def']))                        # []

# --- SequenceMatcher: strings ---
sm = difflib.SequenceMatcher(None, 'abcde', 'acde')
print(round(sm.ratio(), 4))            # 0.8889
print(round(sm.quick_ratio(), 4))      # 0.8889
print(round(sm.real_quick_ratio(), 4)) # 0.8889

sm.set_seqs('hello', 'hello')
print(sm.ratio())                      # 1.0

sm.set_seq1('abc')
sm.set_seq2('abc')
print(sm.ratio())                      # 1.0

# --- SequenceMatcher: lists ---
sm2 = difflib.SequenceMatcher(None, ['a', 'b', 'c', 'd'], ['a', 'c', 'd', 'e'])
print(round(sm2.ratio(), 4))           # 0.75

# get_matching_blocks
for b in sm2.get_matching_blocks():
    print(b.a, b.b, b.size)
# 0 0 1 / 2 1 2 / 4 4 0

# get_opcodes
for op in sm2.get_opcodes():
    print(op)
# ('equal',0,1,0,1) / ('delete',1,2,1,1) / ('equal',2,4,1,3) / ('insert',4,4,3,4)

# replace
sm3 = difflib.SequenceMatcher(None, ['a', 'b', 'c'], ['a', 'x', 'c'])
for op in sm3.get_opcodes():
    print(op)
# ('equal',0,1,0,1) / ('replace',1,2,1,2) / ('equal',2,3,2,3)

# find_longest_match
sm4 = difflib.SequenceMatcher(None, 'abcde', 'acde')
lm = sm4.find_longest_match()
print(lm.a, lm.b, lm.size)            # 2 1 3

sm5 = difflib.SequenceMatcher(None, ' abcd', 'abcd abcd')
lm2 = sm5.find_longest_match()
print(lm2.a, lm2.b, lm2.size)         # 0 4 5

# --- ndiff (cases without ? lines) ---
a1 = ['one\n', 'two\n', 'three\n']
b1 = ['one\n', 'TWO\n', 'three\n']
for line in difflib.ndiff(a1, b1):
    print(repr(line))
# '  one\n' / '- two\n' / '+ TWO\n' / '  three\n'

# --- unified_diff ---
for line in difflib.unified_diff(a1, b1, fromfile='a.txt', tofile='b.txt'):
    print(repr(line))

# --- context_diff ---
for line in difflib.context_diff(a1, b1, fromfile='a.txt', tofile='b.txt'):
    print(repr(line))

# --- restore ---
diff = list(difflib.ndiff(a1, b1))
print(list(difflib.restore(diff, 1)))
print(list(difflib.restore(diff, 2)))

# --- Differ ---
d = difflib.Differ()
for line in d.compare(['one\n', 'two\n'], ['one\n', 'TWO\n']):
    print(repr(line))

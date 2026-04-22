import rlcompleter

# --- Completer with explicit namespace ---
ns = {'alpha': 1, 'alphabet': 2, 'beta': 3, 'Beta': 4}
c = rlcompleter.Completer(ns)

# state=0,1,2... until None
matches = []
state = 0
while True:
    m = c.complete('al', state)
    if m is None:
        break
    matches.append(m)
    state += 1
print(sorted(matches))

# single match
print(c.complete('bet', 0))
print(c.complete('bet', 1))

# no match
print(c.complete('zzz', 0))

# --- Completer with no namespace (uses __main__ + builtins + keywords) ---
c2 = rlcompleter.Completer()

# builtins: print, property, pow
results = []
for s in range(20):
    m = c2.complete('pr', s)
    if m is None:
        break
    results.append(m)
print(sorted(results))

# keyword: for, from, finally, False
kw_results = []
for s in range(20):
    m = c2.complete('fo', s)
    if m is None:
        break
    kw_results.append(m)
print(sorted(kw_results))

# no match
print(c2.complete('zzzzzzz', 0))

# --- dotted completion ---
import os
c3 = rlcompleter.Completer({'os': os})
dot_matches = []
for s in range(100):
    m = c3.complete('os.path', s)
    if m is None:
        break
    dot_matches.append(m)
print(len(dot_matches) > 0)
print(all(m.startswith('os.path') for m in dot_matches))

# bad dotted expr (silenced)
c4 = rlcompleter.Completer({})
print(c4.complete('nonexistent.attr', 0))

print('done')

from graphlib import TopologicalSorter, CycleError

# ===== Basic topological sort (deterministic: list deps, linear chain) =====
ts = TopologicalSorter({'C': ['A'], 'B': ['A'], 'A': []})
result = sorted(ts.static_order())  # sort so output is deterministic
print(result)  # ['A', 'B', 'C'] (all three nodes present)

# ===== Constructor with no argument =====
ts2 = TopologicalSorter()
ts2.add('B', 'A')
ts2.add('A')
print(list(ts2.static_order()))  # ['A', 'B']

# ===== add() accumulates predecessors =====
ts3 = TopologicalSorter()
ts3.add('C', 'A')
ts3.add('C', 'B')  # second call unions deps
ts3.add('A')
ts3.add('B')
order3 = list(ts3.static_order())
print(order3.index('A') < order3.index('C'))  # True
print(order3.index('B') < order3.index('C'))  # True

# ===== prepare() + get_ready() + done() manual workflow =====
ts4 = TopologicalSorter({'B': ['A'], 'C': ['A'], 'A': []})
ts4.prepare()
print(ts4.is_active())   # True
r1 = ts4.get_ready()
print(sorted(r1))        # ['A']
ts4.done('A')
r2 = ts4.get_ready()
print(sorted(r2))        # ['B', 'C']
ts4.done('B', 'C')
r3 = ts4.get_ready()
print(list(r3))          # []
print(ts4.is_active())   # False

# ===== CycleError is a ValueError =====
print(issubclass(CycleError, ValueError))   # True

# ===== Cycle detection =====
try:
    ts5 = TopologicalSorter({'A': ['B'], 'B': ['A']})
    list(ts5.static_order())
    print('no error')
except CycleError as e:
    print('CycleError raised')
    print(isinstance(e, ValueError))   # True

# ===== add() after prepare() raises ValueError =====
ts6 = TopologicalSorter()
ts6.add('A')
ts6.prepare()
try:
    ts6.add('B')
    print('no error')
except ValueError:
    print('ValueError on add after prepare')

# ===== done() without prepare() raises ValueError =====
ts7 = TopologicalSorter()
ts7.add('A')
try:
    ts7.done('A')
    print('no error')
except ValueError:
    print('ValueError on done without prepare')

# ===== done() on node not returned by get_ready() =====
ts8 = TopologicalSorter({'B': ['A']})
ts8.prepare()
try:
    ts8.done('B')   # B is not ready yet
    print('no error')
except ValueError:
    print('ValueError on done for non-ready node')

# ===== is_active() before prepare() raises ValueError =====
ts9 = TopologicalSorter()
ts9.add('A')
try:
    ts9.is_active()
    print('no error')
except ValueError:
    print('ValueError on is_active before prepare')

# ===== Empty graph =====
ts10 = TopologicalSorter()
print(list(ts10.static_order()))  # []

# ===== Single node no deps =====
ts11 = TopologicalSorter({'X': []})
print(list(ts11.static_order()))  # ['X']

# ===== Linear chain =====
ts12 = TopologicalSorter({'C': ['B'], 'B': ['A'], 'A': []})
result12 = list(ts12.static_order())
print(result12.index('A') < result12.index('B'))  # True
print(result12.index('B') < result12.index('C'))  # True

# ===== Diamond dependency =====
ts13 = TopologicalSorter({'D': ['B', 'C'], 'B': ['A'], 'C': ['A'], 'A': []})
result13 = list(ts13.static_order())
print(result13.index('A') < result13.index('B'))  # True
print(result13.index('A') < result13.index('C'))  # True
print(result13.index('B') < result13.index('D'))  # True
print(result13.index('C') < result13.index('D'))  # True

# ===== CycleError args[1] contains cycle nodes =====
try:
    ts14 = TopologicalSorter({'A': ['B'], 'B': ['C'], 'C': ['A']})
    list(ts14.static_order())
except CycleError as e:
    print(len(e.args) >= 2)   # True
    print(isinstance(e.args[1], list))  # True

print('done')

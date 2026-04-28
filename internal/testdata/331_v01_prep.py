# 331_v01_prep.py — v0.0.331 v0.1.0 prep coverage:
#   - C3 MRO linearization on diamond
#   - PEP 695 generic class / type alias / generic function
#   - INTRINSIC_TYPEVAR / INTRINSIC_TYPEVAR_WITH_BOUND
#   - INTRINSIC_PREP_RERAISE_STAR (except*)
#   - INTRINSIC_IMPORT_STAR (from math import *)

# ── 1. C3 MRO on a classic diamond ─────────────────────────────────────────────
# A
# /\
# B  C
# \/
# D
# Expected MRO: D, B, C, A, object  (NOT D, B, A, C, A which depth-first gives)
class A:
    def who(self): return 'A'

class B(A):
    def who(self): return 'B'

class C(A):
    def who(self): return 'C'

class D(B, C):
    pass

mro_names = [c.__name__ for c in D.__mro__]
# Strip the trailing 'object' if present (depends on root class chain).
core = [n for n in mro_names if n != 'object']
assert core == ['D', 'B', 'C', 'A'], f'C3 MRO wrong: got {core}'

# super() chain follows MRO: D().who() walks D→B→C→A and returns 'B'.
assert D().who() == 'B'

# ── 2. PEP 695 generic class — exercises INTRINSIC_TYPEVAR + SUBSCRIPT_GENERIC ─
class Box[T]:
    def __init__(self, value):
        self.value = value

b = Box(42)
assert b.value == 42

# Box has __type_params__ (set by SET_FUNCTION_TYPE_PARAMS in 3.14, but the
# class-body machinery currently surfaces it via the generic-parameters frame;
# at minimum the class is callable and constructs).
# Don't assert __type_params__ shape — CPython and goipy both expose it,
# but the surface differs slightly.

# ── 3. PEP 695 type alias — exercises INTRINSIC_TYPEALIAS ─────────────────────
type Vector = list[int]
assert Vector.__name__ == 'Vector'

# ── 4. PEP 695 generic function with bound — exercises TYPEVAR_WITH_BOUND ─────
def identity[T: int](x: T) -> T:
    return x

assert identity(5) == 5
assert identity(0) == 0

# ── 5. PEP 695 generic function with constraints — TYPEVAR_WITH_CONSTRAINTS ───
def first[T: (int, str)](x: T) -> T:
    return x

assert first(7) == 7
assert first('hi') == 'hi'

# ── 6. except* / ExceptionGroup — exercises INTRINSIC_PREP_RERAISE_STAR ───────
got_value = False
try:
    raise ExceptionGroup('group', [ValueError('boom')])
except* ValueError as eg:
    got_value = True
assert got_value

# All-handled case: no re-raise.
handled = False
try:
    try:
        raise ExceptionGroup('g', [ValueError('v')])
    except* ValueError:
        handled = True
except BaseException:
    handled = False
assert handled

# ── 7. INTRINSIC_IMPORT_STAR — `from math import *` at module level ───────────
# (Inside a function body, `from x import *` is a SyntaxError in CPython,
# so we exercise it at module scope below.)
from math import *
# sqrt and pi must now be in scope.
assert sqrt(4) == 2.0
assert 3.14 < pi < 3.15

print('ok')

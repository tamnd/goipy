"""v0.0.336 hash invariant — hash(1) == hash(1.0) == hash(1+0j); the
data-model promise that x == y ⇒ hash(x) == hash(y) holds across
int/float/complex/Decimal/Fraction."""
from decimal import Decimal
from fractions import Fraction


# ── 1. Same-value, different types
assert hash(1) == 1
assert hash(1.0) == 1
assert hash(1 + 0j) == 1
assert hash(0) == 0
assert hash(0.0) == 0
assert hash(0 + 0j) == 0
assert hash(-1) == -2
assert hash(-1.0) == -2
print("trio ok")


# ── 2. Cross-type dict lookup
d = {1: 'a', 2.0: 'b', 3 + 0j: 'c'}
assert d[1.0] == 'a'
assert d[2] == 'b'
assert d[3] == 'c'
print("dict lookup ok")


# ── 3. Set deduplication
s = {1, 1.0, 1 + 0j, True}
assert len(s) == 1
print("set dedup ok")


# ── 4. Big-int boundaries
assert hash(2**61 - 1) == 0  # P - 1; (P-1) mod P = 0... wait, 2^61-1 mod (2^61-1) = 0 indeed
assert hash(2**61) == 1
assert hash(2**62) == 2
print("big int ok")


# ── 5. Big int matches its float counterpart when representable
n = 2**53
assert hash(n) == hash(float(n))
print("big-int/float ok")


# ── 6. Negatives
assert hash(-2**61) == -1 + (-1)  # actually: hash(-2**61) = -hash(2**61) = -1; but -1 → -2
# Above confused; just check parity:
assert hash(-(2**61 - 1)) == 0
print("negatives ok")


# ── 7. Specials
import math
assert hash(math.inf) == 314159
assert hash(-math.inf) == -314159
# CPython 3.10+ randomises NaN hashes; just require it to be a hashable int.
assert isinstance(hash(math.nan), int)
print("specials ok")


# ── 8. Decimal & Fraction
assert hash(Decimal(1)) == hash(1)
assert hash(Decimal('1.0')) == hash(1)
assert hash(Fraction(1, 1)) == hash(1)
assert hash(Fraction(2, 4)) == hash(Fraction(1, 2))
print("decimal/fraction ok")


# ── 9. Bool subclass invariant
assert hash(True) == hash(1) == 1
assert hash(False) == hash(0) == 0
print("bool ok")


print("ok")

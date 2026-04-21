import math
import heapq
import bisect
import random

# --- math stress ---

# 1) sqrt of a tiny positive is nearly zero.
print(math.sqrt(0))
print(math.sqrt(1e-300) < 1e-100)

# 2) gcd with all-zero returns 0.
print(math.gcd(0, 0))
print(math.gcd())
print(math.lcm())

# 3) lcm with a zero collapses to zero.
print(math.lcm(0, 4, 6))

# 4) gcd/lcm with negatives takes absolute values.
print(math.gcd(-12, 18))
print(math.lcm(-4, 6))

# 5) factorial on 0 and 1.
print(math.factorial(0), math.factorial(1))

# 6) factorial raises on negative.
try:
    math.factorial(-1)
except ValueError:
    print("fact-neg: ValueError")

# 7) comb edge cases.
print(math.comb(0, 0))
print(math.comb(5, 0))
print(math.comb(5, 5))
print(math.comb(5, 6))
print(math.comb(20, 10))

# 8) perm edge cases.
print(math.perm(0))
print(math.perm(5, 0))
print(math.perm(3, 5))

# 9) prod on empty returns start (1 by default).
print(math.prod([]))
print(math.prod([], start=42))

# 10) prod with floats.
print(math.prod([1.5, 2.0, 3.0]))

# 11) hypot with no args (==0 in CPython).
print(math.hypot())

# 12) dist in 3D.
print(math.dist([1, 2, 3], [4, 6, 3]))

# 13) isclose with huge numbers.
print(math.isclose(1e10, 1e10 + 1))  # True, within rel_tol
print(math.isclose(1e10, 1.1e10))   # False

# 14) log at small positives is a large negative.
print(math.log(1e-10) < 0)
print(math.log(1) == 0)

# 15) atan2 quadrant behavior.
print(f"{math.atan2(1, 1):.3f}")
print(f"{math.atan2(-1, -1):.3f}")
print(f"{math.atan2(0, -1):.3f}")

# 16) ceil/floor preserve int ints.
print(math.ceil(5))
print(math.floor(-5))

# 17) isnan on int returns False.
print(math.isnan(5))

# 18) modf preserves sign.
frac, whole = math.modf(-3.25)
print(f"{frac:.2f} {whole:.1f}")

# 19) frexp / ldexp are inverses.
m, e = math.frexp(1024.0)
print(math.ldexp(m, e))

# 20) copysign with zeros.
print(math.copysign(0.0, -1.0))
print(math.copysign(0.0, 1.0))

# 21) fsum on a tricky series.
print(math.fsum([0.1] * 10))

# --- heapq stress ---

# 22) heappush into empty yields a singleton.
h = []
heapq.heappush(h, 10)
print(h)

# 23) heapify + heappop returns ascending order.
h = [7, 3, 9, 1, 5, 2]
heapq.heapify(h)
out = sorted([heapq.heappop(h) for _ in range(len(h))])
print(out)

# 24) heappushpop on empty just returns the pushed value.
h = []
print(heapq.heappushpop(h, 5))
print(h)

# 25) heapreplace on empty raises.
try:
    heapq.heapreplace([], 1)
except IndexError:
    print("replace-empty: IndexError")

# 26) nlargest / nsmallest with n > len.
nums = [1, 2, 3]
print(heapq.nlargest(10, nums))
print(heapq.nsmallest(10, nums))

# 27) nlargest / nsmallest with n = 0.
print(heapq.nlargest(0, [1, 2, 3]))

# 28) nlargest with key.
words = ["apple", "fig", "banana", "kiwi"]
print(heapq.nlargest(2, words, key=len))
print(heapq.nsmallest(2, words, key=len))

# 29) heap of tuples sorts by first element.
h = []
for pair in [(3, "c"), (1, "a"), (2, "b")]:
    heapq.heappush(h, pair)
out = []
while h:
    out.append(heapq.heappop(h))
print(out)

# 30) merge with multiple iterables.
print(list(heapq.merge([1, 4, 7], [2, 5, 8], [3, 6, 9])))

# 31) merge with reverse=True.
print(list(heapq.merge([9, 6, 3], [8, 5, 2], reverse=True)))

# --- bisect stress ---

# 32) bisect_left vs bisect_right on duplicates.
a = [1, 2, 2, 2, 3]
print(bisect.bisect_left(a, 2))
print(bisect.bisect_right(a, 2))

# 33) bisect with lo/hi bounds.
a = [1, 2, 3, 4, 5]
print(bisect.bisect_left(a, 3, 0, 2))  # lo=0, hi=2 → 2
print(bisect.bisect_right(a, 3, 3, 5)) # 3 not in [4,5] — returns lo=3

# 34) bisect_left on value not in list.
print(bisect.bisect_left([1, 3, 5], 4))

# 35) bisect_left at far edges.
print(bisect.bisect_left([1, 2, 3], -1))
print(bisect.bisect_right([1, 2, 3], 99))

# 36) insort maintains order on random inserts.
data = []
for x in [5, 2, 8, 1, 9, 3, 7, 4, 6]:
    bisect.insort(data, x)
print(data)

# 37) bisect with key.
data = [("a", 1), ("b", 3), ("c", 7)]
idx = bisect.bisect_left(data, 5, key=lambda t: t[1])
print(idx)

# 38) insort with key.
data = [("a", 1), ("c", 5)]
bisect.insort(data, ("b", 3), key=lambda t: t[1])
print(data)

# --- random stress ---

# 39) seeded random() is deterministic.
random.seed(100)
v1 = [random.random() for _ in range(5)]
random.seed(100)
v2 = [random.random() for _ in range(5)]
print(v1 == v2)

# 40) randint is inclusive on both ends.
random.seed(1)
vals = [random.randint(1, 3) for _ in range(200)]
print(min(vals), max(vals))

# 41) randrange without step.
random.seed(2)
vals = [random.randrange(5, 10) for _ in range(200)]
print(min(vals), max(vals))

# 42) randrange with step.
random.seed(2)
vals = [random.randrange(0, 20, 4) for _ in range(200)]
print(sorted(set(vals)))  # {0, 4, 8, 12, 16}

# 43) randrange empty raises.
try:
    random.randrange(5, 5)
except ValueError:
    print("rr-empty: ValueError")

# 44) choice distribution over small set.
random.seed(3)
pop = ["x", "y", "z"]
out = {p: 0 for p in pop}
for _ in range(300):
    out[random.choice(pop)] += 1
print(all(v > 50 for v in out.values()))  # roughly balanced

# 45) shuffle preserves elements (same multiset).
random.seed(4)
data = list(range(20))
orig = sorted(data)
random.shuffle(data)
print(sorted(data) == orig)

# 46) sample yields unique items.
random.seed(5)
s = random.sample(list(range(10)), 5)
print(len(s), len(set(s)))

# 47) sample larger than population raises.
try:
    random.sample([1, 2], 5)
except ValueError:
    print("sample-big: ValueError")

# 48) uniform stays in range.
random.seed(6)
for _ in range(50):
    v = random.uniform(10, 20)
    if v < 10 or v >= 20:
        print("uniform out of range:", v)
        break
else:
    print("uniform-ok")

# 49) choices honors k.
random.seed(7)
c = random.choices(["a", "b"], k=10)
print(len(c), all(x in ("a", "b") for x in c))

# 50) seed(None)-like: reseeding with 0.
random.seed(0)
print(isinstance(random.random(), float))

# 51) random sequence from a list of tuples.
random.seed(8)
items = [(1, 'a'), (2, 'b'), (3, 'c')]
print(random.choice(items) in items)

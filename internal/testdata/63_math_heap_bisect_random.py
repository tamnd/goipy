import math
import heapq
import bisect
import random

# --- math basics ---

print(f"{math.pi:.5f}")
print(f"{math.e:.5f}")
print(f"{math.tau:.5f}")
print(math.inf > 1e308)
print(math.isnan(math.nan))

print(math.sqrt(16))
print(math.sqrt(2) > 1.41 and math.sqrt(2) < 1.42)

print(math.ceil(1.1), math.ceil(-1.1))
print(math.floor(1.9), math.floor(-1.9))
print(math.trunc(1.9), math.trunc(-1.9))
print(math.fabs(-3.5))

print(math.gcd(12, 18))
print(math.gcd(12, 18, 24))
print(math.lcm(4, 6))
print(math.lcm(3, 4, 5))

print(math.factorial(0), math.factorial(5), math.factorial(10))
print(math.comb(5, 2), math.comb(10, 3), math.comb(5, 7))
print(math.perm(5, 2), math.perm(5, 0), math.perm(5))

print(math.hypot(3, 4))
print(math.hypot(1, 2, 2))
print(math.dist([0, 0], [3, 4]))
print(math.prod([1, 2, 3, 4]))
print(math.prod([1, 2, 3], start=10))

print(math.isclose(0.1 + 0.2, 0.3))
print(math.isclose(1.0, 2.0))
print(math.isclose(1e-10, 0, abs_tol=1e-9))

print(f"{math.log(math.e):.5f}")
print(f"{math.log(8, 2):.5f}")
print(f"{math.log2(1024):.1f}")
print(f"{math.log10(1000):.1f}")
print(f"{math.exp(0):.1f}")

print(f"{math.sin(0):.5f}")
print(f"{math.cos(0):.5f}")
print(f"{math.tan(0):.5f}")
print(f"{math.atan2(1, 1):.5f}")

print(f"{math.degrees(math.pi):.1f}")
print(f"{math.radians(180):.5f}")

print(math.copysign(5, -2))
print(math.fmod(10, 3))

print(math.isfinite(1.0), math.isfinite(math.inf), math.isfinite(math.nan))
print(math.isinf(math.inf), math.isinf(1.0))

frac, whole = math.modf(3.75)
print(f"{frac:.2f} {whole:.1f}")

m, e = math.frexp(8.0)
print(f"{m} {e}")
print(math.ldexp(0.5, 4))

# --- heapq basics ---

h = []
for x in [3, 1, 4, 1, 5, 9, 2, 6]:
    heapq.heappush(h, x)
out = []
while h:
    out.append(heapq.heappop(h))
print(out)

# heapify an unsorted list
h = [5, 3, 8, 1, 9, 2]
heapq.heapify(h)
out = []
while h:
    out.append(heapq.heappop(h))
print(out)

# heappushpop / heapreplace
h = [1, 3, 5]
heapq.heapify(h)
print(heapq.heappushpop(h, 0))  # returns 0 without reshuffle
print(heapq.heappushpop(h, 4))  # returns 1, pushes 4
print(h)

h = [1, 3, 5]
heapq.heapify(h)
print(heapq.heapreplace(h, 2))
print(sorted(h))

# nlargest / nsmallest
nums = [3, 1, 4, 1, 5, 9, 2, 6, 5, 3, 5]
print(heapq.nlargest(3, nums))
print(heapq.nsmallest(3, nums))

# merge preserves sort order.
print(list(heapq.merge([1, 3, 5], [2, 4, 6])))

# --- bisect basics ---

a = [1, 3, 5, 7, 9]
print(bisect.bisect_left(a, 5))
print(bisect.bisect_right(a, 5))
print(bisect.bisect(a, 4))
print(bisect.bisect_left(a, 0))
print(bisect.bisect_right(a, 99))

b = [1, 3, 5]
bisect.insort(b, 4)
print(b)
bisect.insort_left(b, 3)
print(b)

# --- random basics ---

random.seed(42)
r1 = random.random()
random.seed(42)
r2 = random.random()
print(r1 == r2)

random.seed(7)
print(0 <= random.random() < 1)
print(1 <= random.randint(1, 10) <= 10)
print(random.choice([10, 20, 30]) in (10, 20, 30))

random.seed(11)
ch = random.choices(["a", "b", "c"], k=5)
print(len(ch), all(x in ("a", "b", "c") for x in ch))

random.seed(3)
data = [1, 2, 3, 4, 5]
random.shuffle(data)
print(sorted(data))  # same elements

random.seed(5)
s = random.sample([1, 2, 3, 4, 5], 3)
print(len(s), len(set(s)))  # unique

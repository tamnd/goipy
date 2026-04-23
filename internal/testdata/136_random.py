import random

# ===== Seed for determinism =====
random.seed(42)

# ===== random() =====
v = random.random()
print(0.0 <= v < 1.0)          # True

# ===== uniform() =====
random.seed(42)
v = random.uniform(1.0, 10.0)
print(1.0 <= v <= 10.0)         # True
v = random.uniform(5.0, 5.0)
print(v == 5.0)                  # True

# ===== randint() =====
random.seed(42)
for _ in range(10):
    n = random.randint(1, 6)
    assert 1 <= n <= 6, f"randint out of range: {n}"
print('randint ok')               # randint ok

# ===== randrange() =====
random.seed(42)
n = random.randrange(10)
print(0 <= n < 10)               # True
n = random.randrange(2, 10)
print(2 <= n < 10)               # True
n = random.randrange(0, 10, 2)
print(n % 2 == 0 and 0 <= n < 10)  # True
n = random.randrange(10, 0, -2)
print(n % 2 == 0 and 0 < n <= 10)  # True

# ===== getrandbits() =====
random.seed(42)
b = random.getrandbits(8)
print(0 <= b <= 255)             # True
b = random.getrandbits(64)
print(0 <= b < 2**64)            # True
b = random.getrandbits(0)
print(b == 0)                     # True

# ===== randbytes() =====
random.seed(42)
data = random.randbytes(4)
print(len(data) == 4)            # True
print(isinstance(data, bytes))   # True

# ===== choice() =====
random.seed(42)
items = [1, 2, 3, 4, 5]
chosen = random.choice(items)
print(chosen in items)           # True

# ===== choices() =====
random.seed(42)
result = random.choices([1, 2, 3], k=5)
print(len(result) == 5)          # True
print(all(x in [1, 2, 3] for x in result))  # True

# choices with weights
random.seed(42)
result = random.choices(['a', 'b', 'c'], weights=[10, 1, 1], k=100)
print(result.count('a') > 50)   # True (heavily weighted toward 'a')

# choices with cumulative weights
random.seed(42)
result = random.choices(['a', 'b'], cum_weights=[90, 100], k=100)
print(result.count('a') > 50)   # True

# ===== shuffle() =====
random.seed(42)
lst = list(range(10))
random.shuffle(lst)
print(sorted(lst) == list(range(10)))   # True (same elements)
print(lst != list(range(10)))           # True (order changed after seed)

# ===== sample() =====
random.seed(42)
result = random.sample(range(100), 10)
print(len(result) == 10)         # True
print(len(set(result)) == 10)    # True (no duplicates)

# sample with counts
random.seed(42)
result = random.sample(['a', 'b', 'c'], 2, counts=[3, 2, 1])
print(len(result) == 2)          # True

# ===== binomialvariate() =====
random.seed(42)
for _ in range(20):
    b = random.binomialvariate(10, 0.5)
    assert 0 <= b <= 10, f"binomialvariate out of range: {b}"
print('binomialvariate ok')      # binomialvariate ok

b = random.binomialvariate(0, 0.5)
print(b == 0)                     # True

# ===== triangular() =====
random.seed(42)
for _ in range(20):
    v = random.triangular(0.0, 1.0)
    assert 0.0 <= v <= 1.0, f"triangular out of range: {v}"
print('triangular ok')            # triangular ok

v = random.triangular(0.0, 10.0, 5.0)
print(isinstance(v, float))       # True

# ===== expovariate() =====
random.seed(42)
for _ in range(20):
    v = random.expovariate(1.0)
    assert v > 0, f"expovariate not positive: {v}"
print('expovariate ok')           # expovariate ok

v = random.expovariate(2.0)
print(isinstance(v, float) and v > 0)  # True

# ===== gauss() =====
random.seed(42)
vals = [random.gauss(0.0, 1.0) for _ in range(1000)]
mean = sum(vals) / len(vals)
print(abs(mean) < 0.2)           # True (close to 0)

# default args
random.seed(42)
v = random.gauss()
print(isinstance(v, float))       # True

# ===== normalvariate() =====
random.seed(42)
vals = [random.normalvariate(0.0, 1.0) for _ in range(1000)]
mean = sum(vals) / len(vals)
print(abs(mean) < 0.2)           # True

# default args
random.seed(42)
v = random.normalvariate()
print(isinstance(v, float))       # True

# ===== lognormvariate() =====
random.seed(42)
for _ in range(20):
    v = random.lognormvariate(0.0, 1.0)
    assert v > 0, f"lognormvariate not positive: {v}"
print('lognormvariate ok')        # lognormvariate ok

# ===== gammavariate() =====
random.seed(42)
for _ in range(20):
    v = random.gammavariate(2.0, 1.0)
    assert v > 0, f"gammavariate not positive: {v}"
print('gammavariate ok')          # gammavariate ok

# alpha < 1
random.seed(42)
v = random.gammavariate(0.5, 1.0)
print(v > 0)                      # True

# alpha == 1
random.seed(42)
v = random.gammavariate(1.0, 2.0)
print(v > 0)                      # True

# ===== betavariate() =====
random.seed(42)
for _ in range(20):
    v = random.betavariate(2.0, 3.0)
    assert 0.0 < v < 1.0, f"betavariate out of range: {v}"
print('betavariate ok')           # betavariate ok

# ===== vonmisesvariate() =====
import math
random.seed(42)
for _ in range(20):
    v = random.vonmisesvariate(0.0, 1.0)
    assert 0.0 <= v <= 2 * math.pi, f"vonmisesvariate out of range: {v}"
print('vonmisesvariate ok')       # vonmisesvariate ok

# small kappa (uniform)
v = random.vonmisesvariate(0.0, 0.0)
print(isinstance(v, float))       # True

# ===== paretovariate() =====
random.seed(42)
for _ in range(20):
    v = random.paretovariate(1.0)
    assert v >= 1.0, f"paretovariate out of range: {v}"
print('paretovariate ok')         # paretovariate ok

# ===== weibullvariate() =====
random.seed(42)
for _ in range(20):
    v = random.weibullvariate(1.0, 1.5)
    assert v > 0, f"weibullvariate not positive: {v}"
print('weibullvariate ok')        # weibullvariate ok

# ===== getstate / setstate =====
random.seed(42)
state = random.getstate()
v1 = random.random()
random.setstate(state)
v2 = random.random()
print(v1 == v2)                   # True (same state restored)

# ===== Random class =====
rng = random.Random(99)
v = rng.random()
print(0.0 <= v < 1.0)            # True

rng.seed(42)
v1 = rng.random()
rng.seed(42)
v2 = rng.random()
print(v1 == v2)                   # True (same seed → same output)

# Multiple instances are independent
rng1 = random.Random(1)
rng2 = random.Random(2)
v1 = rng1.random()
v2 = rng2.random()
print(isinstance(v1, float) and isinstance(v2, float))  # True

# Random.randint
rng = random.Random(42)
print(1 <= rng.randint(1, 10) <= 10)  # True

# Random.choice
rng = random.Random(42)
print(rng.choice([10, 20, 30]) in [10, 20, 30])  # True

# Random.shuffle
rng = random.Random(42)
lst = [1, 2, 3, 4, 5]
rng.shuffle(lst)
print(sorted(lst) == [1, 2, 3, 4, 5])  # True

# Random.gauss
rng = random.Random(42)
v = rng.gauss(0, 1)
print(isinstance(v, float))       # True

# Random.normalvariate
rng = random.Random(42)
v = rng.normalvariate(0, 1)
print(isinstance(v, float))       # True

# Random class getstate/setstate
rng = random.Random(42)
state = rng.getstate()
v1 = rng.random()
rng.setstate(state)
v2 = rng.random()
print(v1 == v2)                   # True

# ===== SystemRandom class =====
srng = random.SystemRandom()
v = srng.random()
print(0.0 <= v < 1.0)            # True

b = srng.getrandbits(8)
print(0 <= b <= 255)             # True

print('done')

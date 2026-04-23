import statistics
from statistics import StatisticsError

# ===== StatisticsError =====
try:
    statistics.mean([])
except StatisticsError:
    print('StatisticsError ok')      # StatisticsError ok

# ===== mean =====
print(statistics.mean([1, 2, 3, 4, 5]))       # 3
print(statistics.mean([1.0, 2.0, 3.0]))       # 2.0
print(statistics.mean(range(1, 6)))           # 3

# ===== fmean =====
print(statistics.fmean([1, 2, 3, 4, 5]))      # 3.0
print(statistics.fmean([1.0, 2.0, 3.0]))      # 2.0
# with weights
print(statistics.fmean([1, 2, 3], weights=[1, 1, 2]))  # 2.25

# ===== geometric_mean =====
v = statistics.geometric_mean([2, 8])
print(abs(v - 4.0) < 1e-9)                   # True
v = statistics.geometric_mean([1, 10, 100])
print(abs(v - 10.0) < 1e-9)                  # True
try:
    statistics.geometric_mean([1, -1])
except StatisticsError:
    print('geometric_mean negative ok')       # geometric_mean negative ok

# ===== harmonic_mean =====
v = statistics.harmonic_mean([1, 2, 4])
print(abs(v - 12/7) < 1e-9)                  # True
# with weights
v = statistics.harmonic_mean([1.0, 2.0], weights=[1, 1])
print(abs(v - 4/3) < 1e-9)                   # True
# zero value → 0
print(statistics.harmonic_mean([0, 1, 2]) == 0)  # True

# ===== median =====
print(statistics.median([1, 3, 5]))           # 3
print(statistics.median([1, 2, 3, 4]))        # 2.5
print(statistics.median([3, 1, 4, 1, 5]))     # 3

# ===== median_low =====
print(statistics.median_low([1, 2, 3, 4]))    # 2
print(statistics.median_low([1, 3, 5]))       # 3

# ===== median_high =====
print(statistics.median_high([1, 2, 3, 4]))   # 3
print(statistics.median_high([1, 3, 5]))      # 3

# ===== median_grouped =====
v = statistics.median_grouped([1, 2, 2, 3, 4], interval=1)
print(isinstance(v, float))                   # True
print(0 < v < 5)                              # True

# ===== mode =====
print(statistics.mode([1, 2, 2, 3]))          # 2
print(statistics.mode(['a', 'b', 'a']))       # a
# first mode wins for ties
print(statistics.mode([1, 2, 3]))             # 1

# ===== multimode =====
result = statistics.multimode([1, 2, 2, 3, 3])
print(sorted(result) == [2, 3])               # True
result = statistics.multimode([1, 2, 3])
print(sorted(result) == [1, 2, 3])            # True
print(statistics.multimode([]) == [])         # True

# ===== quantiles =====
q = statistics.quantiles([1, 2, 3, 4, 5], n=4)
print(len(q) == 3)                            # True
print(all(isinstance(v, float) for v in q))  # True
# inclusive method
q2 = statistics.quantiles([1, 2, 3, 4, 5], n=4, method='inclusive')
print(len(q2) == 3)                           # True

# ===== pvariance =====
v = statistics.pvariance([2, 4, 4, 4, 5, 5, 7, 9])
print(abs(v - 4.0) < 1e-9)                   # True
# with mu
v = statistics.pvariance([2, 4, 4, 4, 5, 5, 7, 9], mu=5.0)
print(isinstance(v, float))                   # True

# ===== variance =====
v = statistics.variance([2, 4, 4, 4, 5, 5, 7, 9])
print(abs(v - 4.571428571) < 1e-6)           # True
# at least 2 required
try:
    statistics.variance([1])
except StatisticsError:
    print('variance single ok')               # variance single ok

# ===== pstdev =====
v = statistics.pstdev([2, 4, 4, 4, 5, 5, 7, 9])
print(abs(v - 2.0) < 1e-9)                   # True

# ===== stdev =====
v = statistics.stdev([2, 4, 4, 4, 5, 5, 7, 9])
print(abs(v - 2.138) < 0.001)                # True

# ===== covariance =====
x = [1, 2, 3, 4, 5]
y = [2, 4, 6, 8, 10]
v = statistics.covariance(x, y)
print(abs(v - 5.0) < 1e-9)                   # True
# negatively correlated
v = statistics.covariance([1, 2, 3], [3, 2, 1])
print(v < 0)                                  # True

# ===== correlation =====
v = statistics.correlation([1, 2, 3, 4, 5], [2, 4, 6, 8, 10])
print(abs(v - 1.0) < 1e-9)                   # True
v = statistics.correlation([1, 2, 3], [3, 2, 1])
print(abs(v - (-1.0)) < 1e-9)                # True
# ranked (Spearman)
v = statistics.correlation([1, 2, 3, 4, 5], [5, 4, 3, 2, 1], method='ranked')
print(abs(v - (-1.0)) < 1e-9)               # True

# ===== linear_regression =====
lr = statistics.linear_regression([1, 2, 3, 4, 5], [2, 4, 6, 8, 10])
print(abs(lr.slope - 2.0) < 1e-9)            # True
print(abs(lr.intercept - 0.0) < 1e-9)        # True
lr2 = statistics.linear_regression([1, 2, 3], [1, 2, 3])
print(abs(lr2.slope - 1.0) < 1e-9)           # True
# proportional (through origin)
lr3 = statistics.linear_regression([1, 2, 3], [2, 4, 6], proportional=True)
print(abs(lr3.slope - 2.0) < 1e-9)           # True
print(abs(lr3.intercept - 0.0) < 1e-9)       # True

# ===== kde =====
data = [1.0, 2.0, 3.0, 4.0, 5.0]
pdf = statistics.kde(data, h=1.0)
v = pdf(3.0)
print(isinstance(v, float) and v > 0)        # True
# cumulative
cdf = statistics.kde(data, h=1.0, cumulative=True)
v0, v1 = cdf(0.0), cdf(100.0)
print(v0 < v1)                               # True
# different kernels
for kernel in ['normal', 'triangular', 'rectangular', 'epanechnikov']:
    f = statistics.kde(data, h=1.0, kernel=kernel)
    print(isinstance(f(3.0), float))         # True (x4)

# ===== kde_random =====
sampler = statistics.kde_random(data, h=1.0, seed=42)
samples = [sampler() for _ in range(50)]
print(len(samples) == 50)                    # True
print(all(isinstance(s, float) for s in samples))  # True

# ===== NormalDist =====
nd = statistics.NormalDist(5.0, 2.0)
print(nd.mean == 5.0)                         # True
print(nd.median == 5.0)                       # True
print(nd.mode == 5.0)                         # True
print(nd.stdev == 2.0)                        # True
print(nd.variance == 4.0)                     # True

# pdf
v = nd.pdf(5.0)
print(isinstance(v, float) and v > 0)        # True
# mode is the peak
print(nd.pdf(5.0) >= nd.pdf(4.0))            # True

# cdf
print(abs(nd.cdf(5.0) - 0.5) < 1e-9)        # True
print(nd.cdf(0.0) < 0.01)                    # True (very far below mean)
print(nd.cdf(10.0) > 0.99)                   # True (very far above mean)

# inv_cdf
print(abs(nd.inv_cdf(0.5) - 5.0) < 0.1)     # True
v_low = nd.inv_cdf(0.001)
v_high = nd.inv_cdf(0.999)
print(v_low < 5.0 < v_high)                  # True

# zscore
print(abs(nd.zscore(5.0) - 0.0) < 1e-9)     # True
print(abs(nd.zscore(7.0) - 1.0) < 1e-9)     # True

# quantiles
q = nd.quantiles(n=4)
print(len(q) == 3)                            # True
print(q[1] == 5.0)                            # True (median = mean)

# samples
s = nd.samples(10, seed=42)
print(len(s) == 10)                           # True
print(all(isinstance(v, float) for v in s))  # True

# from_samples
nd2 = statistics.NormalDist.from_samples([1, 2, 3, 4, 5, 6, 7, 8, 9, 10])
print(abs(nd2.mean - 5.5) < 1e-9)            # True
print(nd2.stdev > 0)                          # True

# overlap
nd_a = statistics.NormalDist(0, 1)
nd_b = statistics.NormalDist(0, 1)
print(abs(nd_a.overlap(nd_b) - 1.0) < 1e-9) # True
nd_c = statistics.NormalDist(100, 1)
print(nd_a.overlap(nd_c) < 0.001)            # True

# arithmetic
nd1 = statistics.NormalDist(3.0, 1.0)
nd2 = statistics.NormalDist(2.0, 1.0)
nd_sum = nd1 + nd2
print(abs(nd_sum.mean - 5.0) < 1e-9)         # True
import math
print(abs(nd_sum.stdev - math.sqrt(2)) < 1e-9)  # True

nd_diff = nd1 - nd2
print(abs(nd_diff.mean - 1.0) < 1e-9)        # True

nd_scaled = nd1 * 2
print(abs(nd_scaled.mean - 6.0) < 1e-9)      # True
print(abs(nd_scaled.stdev - 2.0) < 1e-9)     # True

nd_scaled2 = 3 * nd1
print(abs(nd_scaled2.mean - 9.0) < 1e-9)     # True

nd_div = nd1 / 2
print(abs(nd_div.mean - 1.5) < 1e-9)         # True

# repr
r = repr(statistics.NormalDist(1.0, 2.0))
print('NormalDist' in r)                      # True

# default args
nd0 = statistics.NormalDist()
print(nd0.mean == 0.0)                        # True
print(nd0.stdev == 1.0)                       # True

# negative sigma raises
try:
    statistics.NormalDist(0, -1)
except StatisticsError:
    print('negative sigma ok')                # negative sigma ok

print('done')

import time


def main():
    d = {}
    for i in range(100_000):
        d[i] = i * 7
    total = 0
    for k in range(0, 100_000, 3):
        total += d[k]
    # delete every fifth key
    for k in range(0, 100_000, 5):
        del d[k]
    # iterate remaining keys
    return (total + sum(d.values())) % 1_000_003


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

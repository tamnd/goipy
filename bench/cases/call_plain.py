import time


def inc(x):
    return x + 1


def main():
    v = 0
    for _ in range(2_000_000):
        v = inc(v)
    return v


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

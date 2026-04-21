import time


def main():
    total = 0
    for i in range(100_000):
        s = f"item-{i:05d}={i * 3}"
        total += len(s)
    return total


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

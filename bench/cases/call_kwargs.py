import time


def blend(a, b=10, *, c=3, d=7):
    return a + b + c + d


def main():
    total = 0
    for i in range(500_000):
        total += blend(i, c=i % 3, d=i % 5)
    return total % 1_000_003


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

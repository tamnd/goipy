import time


def counter(n):
    i = 0
    while i < n:
        yield i
        i += 1


def main():
    total = 0
    for v in counter(500_000):
        total += v
    return total % 1_000_003


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

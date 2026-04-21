import time


def main():
    n = 1_000_000
    a, b = 0, 1
    while n > 0:
        a, b = b, a + b
        n -= 1
    return a % 1_000_003


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

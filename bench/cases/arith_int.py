import time


def main():
    s = 0
    for i in range(2_000_000):
        s = (s + i * 3) % 1_000_003
    return s


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

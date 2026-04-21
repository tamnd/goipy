import time


def main():
    s = 0.0
    for i in range(500_000):
        s = s + i * 0.5 - (i % 7) * 0.25
    return round(s, 3)


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

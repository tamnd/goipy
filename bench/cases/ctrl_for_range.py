import time


def main():
    total = 0
    for i in range(5_000_000):
        if i % 2 == 0:
            total += i
        else:
            total -= i
    return total


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

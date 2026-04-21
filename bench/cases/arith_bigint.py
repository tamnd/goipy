import time


def main():
    x = 1
    for i in range(200):
        x = x * (i + 2)
    # 201! is ~ 4.7e378 digits; check by taking mod.
    return x % 1_000_000_007


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

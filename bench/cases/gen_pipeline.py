import time


def src(n):
    for i in range(n):
        yield i


def square(it):
    for v in it:
        yield v * v


def keep_even(it):
    for v in it:
        if v % 2 == 0:
            yield v


def main():
    total = 0
    for v in keep_even(square(src(200_000))):
        total += v
    return total % 1_000_003


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

import time


class Point:
    def __init__(self, x, y):
        self.x = x
        self.y = y


def main():
    total = 0
    for i in range(200_000):
        p = Point(i, i * 2)
        p.x = p.x + 1
        total += p.x + p.y
    return total % 1_000_003


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

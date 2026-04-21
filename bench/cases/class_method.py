import time


class Counter:
    def __init__(self):
        self.n = 0

    def bump(self, k):
        self.n += k
        return self.n


def main():
    c = Counter()
    for i in range(500_000):
        c.bump(1 if i % 2 else 2)
    return c.n


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

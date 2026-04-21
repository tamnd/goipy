import time


class A:
    def value(self):
        return 1


class B(A):
    pass


class C(B):
    pass


class D(C):
    pass


class E(D):
    pass


def main():
    e = E()
    total = 0
    for _ in range(500_000):
        total += e.value()
    return total


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

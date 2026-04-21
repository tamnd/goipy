import time


def make_adder(k):
    def adder(x):
        return x + k
    return adder


def main():
    add3 = make_adder(3)
    add5 = make_adder(5)
    v = 0
    for _ in range(1_000_000):
        v = add3(v)
        v = add5(v)
    return v


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

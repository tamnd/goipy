import time


def main():
    a = set()
    for i in range(50_000):
        a.add(i)
    b = set(range(25_000, 75_000))
    hits = 0
    for i in range(0, 75_000, 2):
        if i in a:
            hits += 1
    u = a | b
    x = a & b
    d = a - b
    return (hits + len(u) + len(x) + len(d)) % 1_000_003


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

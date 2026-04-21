import time


def main():
    parts = [str(i) for i in range(50_000)]
    joined = ",".join(parts)
    chunks = joined.split(",")
    total = 0
    for c in chunks:
        total += len(c)
    return total + len(joined)


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

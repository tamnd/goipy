import time


def main():
    out = []
    for i in range(200_000):
        out.append(i * 3)
    total = 0
    for x in out:
        total += x
    # slice + index + pop
    s2 = sum(out[100:200])
    tail = out[-1]
    out.pop()
    return (total + s2 + tail) % 1_000_003


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

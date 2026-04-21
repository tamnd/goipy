import time


def main():
    s = ""
    for i in range(5_000):
        s += "x"
    # reduce to a checksum — length + a character sample
    return len(s) + ord(s[-1])


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

import time


def solve(n):
    count = 0
    cols = [0] * n

    def place(row):
        nonlocal count
        if row == n:
            count += 1
            return
        for c in range(n):
            ok = True
            for r in range(row):
                if cols[r] == c or abs(cols[r] - c) == row - r:
                    ok = False
                    break
            if ok:
                cols[row] = c
                place(row + 1)

    place(0)
    return count


def main():
    return solve(9)


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

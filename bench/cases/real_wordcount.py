import time


CORPUS_WORDS = [
    "alpha", "beta", "gamma", "delta", "epsilon",
    "zeta", "eta", "theta", "iota", "kappa",
    "lambda", "mu", "nu", "xi", "omicron",
    "pi", "rho", "sigma", "tau", "upsilon",
]


def main():
    # Build a 200k-word corpus deterministically, then count frequencies.
    counts = {}
    n = 200_000
    width = len(CORPUS_WORDS)
    for i in range(n):
        w = CORPUS_WORDS[(i * 31 + 7) % width]
        counts[w] = counts.get(w, 0) + 1
    # Deterministic checksum: sum over sorted keys of count * hash-like value.
    total = 0
    for k in sorted(counts.keys()):
        total += counts[k] * (len(k) + ord(k[0]))
    return total


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

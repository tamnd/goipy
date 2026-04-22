"""Tests for range attributes and methods."""

r = range(10)
print(r.start)
print(r.stop)
print(r.step)

r2 = range(2, 20, 3)
print(r2.start)
print(r2.stop)
print(r2.step)

# --- count ---
print(range(10).count(5))
print(range(10).count(10))
print(range(0, 10, 2).count(4))
print(range(0, 10, 2).count(3))

# --- index ---
print(range(10).index(0))
print(range(10).index(9))
print(range(0, 10, 2).index(4))

try:
    range(10).index(10)
except ValueError:
    print("ValueError")

try:
    range(0, 10, 2).index(3)
except ValueError:
    print("ValueError")

# --- membership ---
print(5 in range(10))
print(10 in range(10))
print(4 in range(0, 10, 2))
print(3 in range(0, 10, 2))

# --- len ---
print(len(range(10)))
print(len(range(0, 10, 2)))
print(len(range(10, 0, -1)))

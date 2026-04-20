xs = list(range(12))

# All-default bounds combinations
print(xs[:])
print(xs[::])
print(xs[3:])
print(xs[:8])
print(xs[3:8])
print(xs[3:8:2])
print(xs[::-2])
print(xs[8:3:-1])
print(xs[-4:-1])
print(xs[-1:-4:-1])

# Out-of-range is silently clamped
print(xs[-100:5])
print(xs[5:1000])
print(xs[1000:])

# Assignment & deletion
ys = list(range(10))
ys[2:5] = [99]
print(ys)
ys[:0] = [-2, -1]
print(ys)
del ys[:2]
print(ys)
del ys[-3:]
print(ys)

# Tuples
t = (1, 2, 3, 4, 5)
print(t[1:4])
print(t[::-1])

# Strings
s = "abcdefgh"
print(s[:])
print(s[::-1])
print(s[1:-1:2])
print(s[100:])  # empty

# Bytes
b = bytes(range(10))
print(list(b[2:8]))
print(list(b[::2]))

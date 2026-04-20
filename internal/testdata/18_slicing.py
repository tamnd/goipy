xs = list(range(10))
print(xs[2:5])
print(xs[:4])
print(xs[6:])
print(xs[::2])
print(xs[::-1])
print(xs[-3:])

xs[1:3] = [20, 30, 40]
print(xs)

del xs[0:2]
print(xs)

s = "abcdefgh"
print(s[::2])
print(s[::-1])
print(s[1:-1])

# Basic if-walrus
if (n := 10) > 5:
    print("big", n)

# In while loop
stream = iter([1, 2, 3, 0, 4])
out = []
while (v := next(stream)) != 0:
    out.append(v)
print(out)

# In list comprehension
print([y for x in range(6) if (y := x * x) > 3])

# Repeatedly bound
data = [1, 2, 3, 4, 5]
if (first := data[0]) and (last := data[-1]):
    print(first, last)

# Nested walrus
xs = [1, 2, 3, 4]
if (s := sum(xs)) > (avg := s / len(xs)) * 2:
    print("skew", s, avg)
else:
    print("flat", s, avg)

# Walrus with negation
words = "one two three four".split()
w = None
for _ in range(5):
    if (w := words.pop(0) if words else None) is None:
        break
    print("w", w)

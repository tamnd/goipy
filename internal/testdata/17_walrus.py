xs = [1, 2, 3, 4, 5]
if (n := len(xs)) > 3:
    print("big", n)

data = iter([1, 2, 3, 4])
out = []
while (v := next(data, None)) is not None:
    out.append(v * 2)
print(out)

print([y for x in range(5) if (y := x * x) > 3])

print([x * x for x in range(6)])
print([x for x in range(20) if x % 3 == 0])
print([(x, y) for x in range(3) for y in range(3) if x != y])

print({x: x * x for x in range(5)})
print({x % 3 for x in range(10)})

m = [[1, 2, 3], [4, 5, 6], [7, 8, 9]]
print([row[i] for i, row in enumerate(m)])

print(sum(x for x in range(100) if x % 7 == 0))

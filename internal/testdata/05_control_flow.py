for i in range(5):
    print(i)

x = 10
if x > 5:
    print("big")
elif x == 5:
    print("mid")
else:
    print("small")

n = 0
while n < 3:
    print("n", n)
    n += 1

for i in range(10):
    if i == 5:
        break
    if i % 2 == 0:
        continue
    print("odd", i)

total = 0
for i in range(1, 11):
    total += i
print("total", total)

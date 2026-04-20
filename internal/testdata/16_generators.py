def count(n):
    i = 0
    while i < n:
        yield i
        i += 1

print(list(count(4)))

def echo():
    while True:
        x = yield
        if x is None:
            return
        print("got", x)

g = echo()
next(g)
g.send(1)
g.send(2)
try:
    g.send(None)
except StopIteration:
    print("done")

def chain(*its):
    for it in its:
        yield from it

print(list(chain([1, 2], (3, 4), range(5, 7))))

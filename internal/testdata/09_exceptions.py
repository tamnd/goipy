try:
    1 / 0
except ZeroDivisionError as e:
    print("caught zero div")

try:
    x = [1, 2, 3]
    print(x[99])
except IndexError:
    print("caught index")

try:
    raise ValueError("bad value")
except ValueError as e:
    print("got", e.args[0])

def f():
    raise RuntimeError("oops")

try:
    f()
except RuntimeError as e:
    print("runtime:", e.args[0])

try:
    try:
        raise KeyError("k")
    except ValueError:
        print("no")
except KeyError:
    print("outer caught key")

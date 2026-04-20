class CM:
    def __init__(self, name, suppress=False):
        self.name = name
        self.suppress = suppress
    def __enter__(self):
        print("enter", self.name)
        return self.name
    def __exit__(self, exc_type, exc, tb):
        print("exit", self.name, exc_type.__name__ if exc_type else None)
        return self.suppress

with CM("a") as x:
    print("body", x)

try:
    with CM("b"):
        raise ValueError("boom")
except ValueError as e:
    print("caught", e.args[0])

with CM("c", suppress=True):
    raise RuntimeError("swallowed")
print("after c")

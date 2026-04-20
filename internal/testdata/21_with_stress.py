class Log:
    def __init__(self, name, swallow=None):
        self.name = name
        self.swallow = swallow
        self.events = []
    def __enter__(self):
        self.events.append("enter")
        return self
    def __exit__(self, typ, val, tb):
        self.events.append(("exit", typ.__name__ if typ else None))
        if self.swallow is not None and typ is not None:
            return issubclass(typ, self.swallow)
        return False

def run():
    a = Log("a")
    b = Log("b")
    with a:
        with b:
            pass
    print("a", a.events)
    print("b", b.events)

    # Multi-item with.
    c = Log("c")
    d = Log("d")
    with c, d:
        pass
    print("c", c.events, "d", d.events)

    # Exception propagated through outer.
    outer = Log("outer", swallow=ValueError)
    inner = Log("inner")
    with outer:
        with inner:
            raise ValueError("x")
    print("outer", outer.events)
    print("inner", inner.events)

    # Exception not swallowed bubbles up.
    caught = None
    try:
        with Log("g"):
            raise RuntimeError("boom")
    except RuntimeError as e:
        caught = e.args[0]
    print("caught", caught)

    # Early return from with.
    e = Log("e")
    def fn():
        with e:
            return 42
    print("ret", fn())
    print("e", e.events)

run()

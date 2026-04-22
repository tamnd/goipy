"""Test EXIT_INIT_CHECK opcode: __init__ must return None."""


class Good:
    def __init__(self):
        self.x = 1


g = Good()
print(g.x)


class Bad:
    def __init__(self):
        return 42  # noqa: PLE0101


try:
    Bad()
    print("no error")
except TypeError as e:
    print("TypeError raised")


class AlsoGood:
    def __init__(self):
        return None


ag = AlsoGood()
print("AlsoGood ok")

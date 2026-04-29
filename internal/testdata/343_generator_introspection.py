# gi_*/cr_*/ag_* introspection attrs on Generator-backed objects.

print("# section 1: plain generator gi_*")


def g():
    yield 1
    yield 2


it = g()
print(it.gi_code.co_name)
print(it.gi_running)
print(it.gi_suspended)
print(it.gi_yieldfrom)
print(it.gi_frame is not None)
print(next(it))
print(it.gi_running)
print(it.gi_suspended)
print(next(it))
print(it.gi_suspended)
try:
    next(it)
except StopIteration:
    pass
print(it.gi_suspended)
print(it.gi_frame)


print("# section 2: coroutine cr_*")


async def coro():
    return 42


c = coro()
print(c.cr_code.co_name)
print(c.cr_running)
print(c.cr_await)
print(c.cr_origin)
print(c.cr_frame is not None)
try:
    c.send(None)
except StopIteration as e:
    print(e.args[0] if e.args else None)
print(c.cr_frame)


print("# section 3: async generator ag_*")


async def agen():
    yield 1


a = agen()
print(a.ag_code.co_name)
print(a.ag_running)
print(a.ag_await)
print(a.ag_frame is not None)


print("# section 4: gi_* not exposed on coroutine and vice versa")


def g2():
    yield


it2 = g2()
print(hasattr(it2, "cr_running"))
print(hasattr(it2, "gi_running"))


async def coro2():
    return None


c2 = coro2()
print(hasattr(c2, "gi_running"))
print(hasattr(c2, "cr_running"))
try:
    c2.send(None)
except StopIteration:
    pass


print("# section 5: gi_running True when introspecting from inside")


snapshot = []


def g3():
    snapshot.append(it3.gi_running)
    yield


it3 = g3()
next(it3)
print(snapshot)
print(it3.gi_running)

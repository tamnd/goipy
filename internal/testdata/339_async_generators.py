"""v0.0.339 async generators — async for, asend, athrow, aclose."""
import asyncio


# ── 1. async for collects yielded values
async def gen1():
    yield 1
    yield 2
    yield 3


async def collect():
    out = []
    async for v in gen1():
        out.append(v)
    return out


print(asyncio.run(collect()))


# ── 2. asend drives a state machine
async def echoer():
    while True:
        v = yield
        if v is None:
            return
        yield v * 10


async def drive_send():
    a = echoer()
    # Prime the gen so it reaches the first `yield` (anext sends None).
    await a.__anext__()
    out = []
    for x in (1, 2, 3):
        # Send x, gen resumes from `v = yield`, then yields x*10.
        r = await a.asend(x)
        out.append(r)
        # Loop back to the priming yield.
        await a.__anext__()
    return out


print(asyncio.run(drive_send()))


# ── 3. aclose raises GeneratorExit cleanly
async def gen3():
    try:
        yield 1
        yield 2
    except GeneratorExit:
        pass


async def drive_close():
    g = gen3()
    v = await g.__anext__()
    await g.aclose()
    return v


print(asyncio.run(drive_close()))


# ── 4. athrow injects an exception the agen catches
async def gen4():
    try:
        yield 1
    except ValueError:
        yield "caught"


async def drive_throw():
    g = gen4()
    first = await g.__anext__()
    second = await g.athrow(ValueError("boom"))
    return first, second


print(asyncio.run(drive_throw()))


# ── 5. async comprehension
async def comp():
    return [v async for v in gen1()]


print(asyncio.run(comp()))


# ── 6. Two interleaved async generators
async def lettering():
    yield "a"
    yield "b"
    yield "c"


async def interleave():
    out = []
    g1 = gen1()
    g2 = lettering()
    for _ in range(3):
        out.append(await g1.__anext__())
        out.append(await g2.__anext__())
    return out


print(asyncio.run(interleave()))

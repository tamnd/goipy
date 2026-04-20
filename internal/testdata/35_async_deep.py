import asyncio

# Exception raised inside a coroutine propagates out through await
async def bad():
    await asyncio.sleep(0)
    raise ValueError("boom")

async def catches():
    try:
        await bad()
    except ValueError as e:
        return f"caught:{e}"
    return "no-exc"

# Coroutine returning a non-scalar value through StopIteration payload
async def make_dict(n):
    await asyncio.sleep(0)
    return {i: i * i for i in range(n)}

# Nested gather: gather whose arguments are themselves coroutines that gather
async def leaf(x):
    await asyncio.sleep(0, x)
    return x + 1

async def branch(xs):
    inner = await asyncio.gather(*[leaf(x) for x in xs])
    return sum(inner)

async def tree():
    results = await asyncio.gather(
        branch([1, 2, 3]),
        branch([10, 20]),
        branch([100]),
    )
    return results

# sleep(0, value) delivers arbitrary object types
async def echo_list():
    v = await asyncio.sleep(0, [1, 2, 3])
    return v

async def echo_tuple():
    v = await asyncio.sleep(0, (7, 8, 9))
    return v

# Long linear await chain
async def chain(n):
    if n == 0:
        return 0
    await asyncio.sleep(0)
    return 1 + await chain(n - 1)

# Reusing the same coroutine factory many times inside gather
async def square(x):
    await asyncio.sleep(0)
    return x * x

async def many():
    return await asyncio.gather(*[square(i) for i in range(10)])

# Try/except inside a coroutine, with success and failure branches
async def maybe(flag):
    try:
        if flag:
            raise RuntimeError("x")
        await asyncio.sleep(0)
        return "ok"
    except RuntimeError:
        return "handled"

async def main():
    print(await catches())
    print(await make_dict(4))
    print(await tree())
    print(await echo_list())
    print(await echo_tuple())
    print(await chain(8))
    print(await many())
    print(await maybe(True), await maybe(False))

    # Exception from a coroutine inside gather propagates up
    try:
        await asyncio.gather(leaf(1), bad(), leaf(2))
    except ValueError as e:
        print("gather-caught:", e)

asyncio.run(main())

import asyncio

# Sequential awaits, return value through nested coroutines
async def compute(x):
    await asyncio.sleep(0)
    return x * x

async def sum_squares(n):
    total = 0
    for i in range(n):
        total += await compute(i)
    return total

# asyncio.sleep(0, value) delivers `value` as the await result
async def with_value():
    v = await asyncio.sleep(0, "hello")
    return v

# gather runs each awaitable and returns the list of results
async def parallel():
    results = await asyncio.gather(
        compute(1),
        compute(2),
        compute(3),
        compute(4),
    )
    return results

# Conditional await
async def pick(flag):
    if flag:
        return await compute(10)
    return await compute(3)

async def main():
    print(await sum_squares(5))
    print(await with_value())
    print(await parallel())
    print(await pick(True))
    print(await pick(False))

    # Awaiting the same helper twice
    a = await compute(7)
    b = await compute(8)
    print(a + b)

    # Deeply nested awaits
    async def one():
        return await compute(1)
    async def two():
        return await one() + await compute(2)
    async def three():
        return await two() + await compute(3)
    print(await three())

asyncio.run(main())

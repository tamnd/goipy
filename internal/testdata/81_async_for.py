"""Test GET_AITER, GET_ANEXT, END_ASYNC_FOR opcodes via async for."""
import asyncio


class AsyncCounter:
    def __init__(self, stop):
        self.current = 0
        self.stop = stop

    def __aiter__(self):
        return self

    async def __anext__(self):
        if self.current >= self.stop:
            raise StopAsyncIteration
        val = self.current
        self.current += 1
        return val


async def collect(n):
    results = []
    async for x in AsyncCounter(n):
        results.append(x)
    return results


async def nested():
    out = []
    async for x in AsyncCounter(3):
        async for y in AsyncCounter(2):
            out.append((x, y))
    return out


async def early_exit():
    results = []
    async for x in AsyncCounter(10):
        if x == 3:
            break
        results.append(x)
    return results


async def main():
    print(await collect(0))
    print(await collect(3))
    print(await collect(5))
    print(await nested())
    print(await early_exit())


asyncio.run(main())

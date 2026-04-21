import time
import asyncio


async def inner(i):
    return i * 2


async def driver(n):
    total = 0
    for i in range(n):
        total += await inner(i)
    return total


def main():
    return asyncio.run(driver(100_000)) % 1_000_003


t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")

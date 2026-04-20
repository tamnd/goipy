import asyncio

async def inner():
    await asyncio.sleep(0)
    return 42

async def main():
    x = await inner()
    print(x)

asyncio.run(main())

import sys
import asyncio

# --- sys constants ---------------------------------------------------------

# 1) version_info is a 5-tuple.
print(len(sys.version_info) == 5)

# 2) major version is 3.
print(sys.version_info[0] == 3)

# 3) minor version is a non-negative int.
print(sys.version_info[1] >= 0)

# 4) releaselevel is a str.
print(isinstance(sys.version_info[3], str))

# 5) version is a str.
print(isinstance(sys.version, str))

# 6) byteorder is a known value.
print(sys.byteorder in ("little", "big"))

# 7) maxsize is a positive int.
print(isinstance(sys.maxsize, int) and sys.maxsize > 0)

# 8) platform is a str.
print(isinstance(sys.platform, str))

# 9) argv is a list.
print(isinstance(sys.argv, list))

# 10) path is a list.
print(isinstance(sys.path, list))

# 11) modules is a dict.
print(isinstance(sys.modules, dict))

# 12) sys itself is in sys.modules.
print("sys" in sys.modules)

# 13) getrecursionlimit returns a positive int.
print(sys.getrecursionlimit() > 0)

# 14) setrecursionlimit round-trips.
old = sys.getrecursionlimit()
sys.setrecursionlimit(42)
print(sys.getrecursionlimit() == 42)
sys.setrecursionlimit(old)
print(sys.getrecursionlimit() == old)

# --- sys.stdout / sys.stderr ---------------------------------------------

# 15) stdout.write returns length.
n = sys.stdout.write("abc\n")
print(n == 4)

# 16) stdout exposes name.
print(sys.stdout.name)

# 17) stdout.mode is 'w'.
print(sys.stdout.mode == "w")

# 18) stdout.closed is False.
print(sys.stdout.closed is False)

# 19) stdout.flush is a no-op that returns None.
print(sys.stdout.flush() is None)

# 20) stderr write works and doesn't affect stdout assertions.
sys.stderr.write("")  # intentionally empty; output goes to test stderr which is discarded
print(True)

# --- sys.exc_info ---------------------------------------------------------

# 21) exc_info is (None, None, None) outside a handler.
print(sys.exc_info() == (None, None, None))

# 22) exc_info returns the handled class inside except.
try:
    raise ValueError("x")
except ValueError:
    t, v, _ = sys.exc_info()
    print(t is ValueError)

# 23) exc_info's value matches.
try:
    raise KeyError("k")
except KeyError as e:
    _, v, _ = sys.exc_info()
    print(v is e)

# 24) exc_info clears after except.
try:
    raise TypeError("t")
except TypeError:
    pass
print(sys.exc_info()[0] is None)

# 25) exc_info nests correctly.
try:
    raise ValueError("outer")
except ValueError:
    try:
        raise IndexError("inner")
    except IndexError:
        print(sys.exc_info()[0] is IndexError)
    print(sys.exc_info()[0] is ValueError)

# --- traceback basics -----------------------------------------------------

def _a():
    return 1 / 0

def _b():
    return _a()

def _c():
    return _b()

# 26) __traceback__ exists on caught exception.
try:
    _c()
except ZeroDivisionError as e:
    print(e.__traceback__ is not None)

# 27) traceback walk yields 4 frames (module, _c, _b, _a).
try:
    _c()
except ZeroDivisionError as e:
    tb = e.__traceback__
    count = 0
    while tb is not None:
        count += 1
        tb = tb.tb_next
    print(count == 4)

# 28) frame names in order.
try:
    _c()
except ZeroDivisionError as e:
    tb = e.__traceback__
    names = []
    while tb is not None:
        names.append(tb.tb_frame.f_code.co_name)
        tb = tb.tb_next
    print(names == ["<module>", "_c", "_b", "_a"])

# 29) tb_lineno is positive in each frame.
try:
    _c()
except ZeroDivisionError as e:
    tb = e.__traceback__
    ok = True
    while tb is not None:
        if tb.tb_lineno <= 0:
            ok = False
        tb = tb.tb_next
    print(ok)

# 30) tb_lasti is non-negative in each frame.
try:
    _c()
except ZeroDivisionError as e:
    tb = e.__traceback__
    ok = True
    while tb is not None:
        if tb.tb_lasti < 0:
            ok = False
        tb = tb.tb_next
    print(ok)

# 31) co_filename is a str.
try:
    _c()
except ZeroDivisionError as e:
    tb = e.__traceback__
    print(isinstance(tb.tb_frame.f_code.co_filename, str))

# --- exception chains -----------------------------------------------------

# 32) `raise X from Y` sets __cause__.
try:
    try:
        raise KeyError("inner")
    except KeyError as ke:
        raise ValueError("outer") from ke
except ValueError as e:
    print(e.__cause__ is not None)

# 33) __cause__ type matches.
try:
    try:
        raise KeyError("inner")
    except KeyError as ke:
        raise ValueError("outer") from ke
except ValueError as e:
    print(type(e.__cause__) is KeyError)

# 34) `raise from None` explicitly suppresses cause.
try:
    try:
        raise KeyError("k")
    except KeyError:
        raise ValueError("v") from None
except ValueError as e:
    print(e.__cause__ is None)

# 35) implicit context: raise during handling sets __context__.
try:
    try:
        raise TypeError("t")
    except TypeError:
        raise IndexError("i")
except IndexError as e:
    print(e.__context__ is not None)

# 36) implicit context type matches.
try:
    try:
        raise TypeError("t")
    except TypeError:
        raise IndexError("i")
except IndexError as e:
    print(type(e.__context__) is TypeError)

# 37) plain raise has no cause.
try:
    raise RuntimeError("r")
except RuntimeError as e:
    print(e.__cause__ is None)

# --- __await__ dispatch ---------------------------------------------------

class Ready:
    """Awaitable that immediately returns a value (no yields)."""
    def __init__(self, v):
        self.v = v

    def __await__(self):
        return self.v
        yield  # unreachable; makes this a generator function


class Counting:
    """Awaitable that yields n Nones before returning 100 + n."""
    def __init__(self, n):
        self.n = n

    def __await__(self):
        for _ in range(self.n):
            yield None
        return 100 + self.n


async def a_single():
    return await Counting(0)

# 38) awaiting a Counting(0) yields 100.
print(asyncio.run(a_single()) == 100)


async def a_multi():
    return await Counting(3)

# 39) awaiting Counting(3) yields 103.
print(asyncio.run(a_multi()) == 103)


async def a_ready():
    return await Ready(42)

# 40) Ready(42) returns 42.
print(asyncio.run(a_ready()) == 42)


async def a_sum():
    a = await Counting(2)
    b = await Counting(5)
    return a + b

# 41) two sequential awaits compose.
print(asyncio.run(a_sum()) == (102 + 105))


async def a_mixed():
    a = await Ready("x")
    b = await Counting(1)
    return (a, b)

# 42) mixed awaitables.
print(asyncio.run(a_mixed()) == ("x", 101))


class DoubleAwait:
    """An awaitable that itself awaits something."""
    def __init__(self, inner):
        self.inner = inner

    async def _run(self):
        return await self.inner

    def __await__(self):
        return self._run().__await__()


async def a_nested():
    return await DoubleAwait(Counting(2))

# 43) nested __await__.
print(asyncio.run(a_nested()) == 102)

# 44) type of Counting is a class.
print(isinstance(Counting, type))

# --- traceback & cause interplay -----------------------------------------

def reraise_wrap():
    try:
        raise ValueError("orig")
    except ValueError as e:
        raise RuntimeError("wrapped") from e

# 45) traceback entries are present on re-raised exceptions.
try:
    reraise_wrap()
except RuntimeError as e:
    print(e.__cause__ is not None and e.__traceback__ is not None)

# 46) cause has its own traceback.
try:
    reraise_wrap()
except RuntimeError as e:
    print(e.__cause__.__traceback__ is not None)

# --- sys.path usable ------------------------------------------------------

# 47) sys.path entries are strings.
print(all(isinstance(p, str) for p in sys.path))

# 48) argv entries are strings.
print(all(isinstance(a, str) for a in sys.argv))

# 49) sys.modules has at least one entry.
print(len(sys.modules) >= 1)

# 50) sys.modules["sys"] is sys.
print(sys.modules["sys"] is sys)

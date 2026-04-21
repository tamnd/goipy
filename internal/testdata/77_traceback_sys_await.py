import sys
import asyncio

# --- sys constants ---
print(sys.version_info[0], sys.version_info[1])
print(len(sys.version_info) == 5)
print(sys.byteorder)
print(isinstance(sys.maxsize, int))
print(sys.maxsize > 0)
print("sys" in sys.modules)
print(isinstance(sys.path, list))
print(isinstance(sys.argv, list))
print(sys.getrecursionlimit() > 0)

# --- sys.stdout write ---
n = sys.stdout.write("hello via sys.stdout\n")
print("wrote", n, "bytes")
sys.stdout.flush()

# --- sys.exc_info during except ---
try:
    raise ValueError("boom")
except ValueError:
    t, v, _ = sys.exc_info()
    print(t is ValueError)
    print(str(v))

# outside handler, exc_info is (None, None, None)
t, v, _ = sys.exc_info()
print(t is None, v is None)

# --- traceback via raise ---
def inner(x):
    return 10 / x

def outer(x):
    return inner(x)

try:
    outer(0)
except ZeroDivisionError as e:
    tb = e.__traceback__
    frames = []
    while tb is not None:
        frames.append(tb.tb_frame.f_code.co_name)
        tb = tb.tb_next
    print(frames)

# --- traceback line numbers after raise inside nested function ---
def raise_at_line():
    x = 1
    y = 2
    raise RuntimeError("at source line")  # keep this on a stable line

try:
    raise_at_line()
except RuntimeError as e:
    tb = e.__traceback__
    lines = []
    while tb is not None:
        lines.append(tb.tb_lineno)
        tb = tb.tb_next
    # We expect two nonzero line numbers (call site + raise site).
    print(all(ln > 0 for ln in lines))
    print(len(lines) == 2)

# --- exception chaining (raise X from Y) ---
try:
    try:
        raise KeyError("k")
    except KeyError as ke:
        raise ValueError("wrapped") from ke
except ValueError as e:
    print(e.__cause__ is not None)
    print(type(e.__cause__).__name__)

# --- implicit context (during handling of X, another occurred) ---
try:
    try:
        raise TypeError("t")
    except TypeError:
        raise IndexError("i")
except IndexError as e:
    print(e.__context__ is not None)
    print(type(e.__context__).__name__)

# --- __await__ on a user class ---
class Delay:
    def __init__(self, n):
        self.n = n

    def __await__(self):
        # produce n values then finish with self.n (via StopIteration.value)
        for _ in range(self.n):
            yield None
        return self.n * 10


async def run_await():
    result = await Delay(3)
    return result


print(asyncio.run(run_await()))


async def run_await_chain():
    a = await Delay(2)
    b = await Delay(4)
    return a + b


print(asyncio.run(run_await_chain()))

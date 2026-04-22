# Tests for previously-missing builtins: globals, locals, vars, object,
# aiter/anext, breakpoint, help, open.

# --- globals() ---
g = globals()
print(type(g).__name__)       # dict
print("__name__" in g)        # True

# --- locals() ---
def check_locals():
    x = 42
    y = "hello"
    locs = locals()
    print(sorted(k for k in locs if k != "locs"))  # ['x', 'y']

check_locals()

# --- vars(obj) ---
class Simple:
    pass

s = Simple()
s.a = 1
s.b = 2
print(sorted(vars(s).keys()))   # ['a', 'b']

class WithAttr:
    z = 99

print(vars(WithAttr)["z"])      # 99

# --- object as base class ---
class MyBase(object):
    pass

class Child(MyBase):
    pass

print(isinstance(MyBase(), object))  # True
print(isinstance(Child(), object))   # True

# --- aiter / anext ---
# Test aiter and anext via a class-based async iterator that does not
# rely on coroutines, so the test works even without CO_ASYNC_GENERATOR.
class AsyncCounter:
    def __init__(self, items):
        self._items = list(items)
        self._i = 0
    def __aiter__(self):
        return self
    def __anext__(self):
        if self._i >= len(self._items):
            raise StopAsyncIteration
        v = self._items[self._i]
        self._i += 1
        return v

ctr = AsyncCounter([10, 20])
it = aiter(ctr)
print(it is ctr)          # True — __aiter__ returns self
print(anext(it))          # 10
print(anext(it))          # 20
try:
    anext(it)             # raises StopAsyncIteration
except StopAsyncIteration:
    print("stop")         # stop

# --- breakpoint: just call it, verify no crash ---
import sys
# Override the hook so CPython doesn't start pdb; goipy ignores it anyway.
sys.breakpointhook = lambda *a, **kw: None
old_err = sys.stderr
class DevNull:
    def write(self, s): pass
    def flush(self): pass
sys.stderr = DevNull()
breakpoint()
sys.stderr = old_err
print("breakpoint ok")             # breakpoint ok

# help() is a no-op stub — not tested in this fixture because CPython's
# interactive help is not reproducible in a scripted test.

# --- open: text write and read ---
import os
path = "/tmp/goipy_80_test.txt"

with open(path, "w") as fh:
    fh.write("line one\n")
    fh.write("line two\n")

with open(path) as fh:
    content = fh.read()

print(content)                     # line one\nline two\n

with open(path) as fh:
    lines = fh.readlines()
print(lines)                       # ['line one\n', 'line two\n']

os.remove(path)

# --- open: binary write and read ---
with open(path, "wb") as fh:
    fh.write(b"bytes data\n")

with open(path, "rb") as fh:
    data = fh.read()
print(data)                        # b'bytes data\n'

os.remove(path)

# --- open: FileNotFoundError ---
try:
    open("/no/such/path/goipy.txt")
except FileNotFoundError:
    print("file not found")        # file not found

# --- vars() with no arg (same as locals()) ---
def check_vars_noarg():
    p = 7
    q = "q"
    d = vars()
    print(sorted(k for k in d if k != "d"))  # ['p', 'q']

check_vars_noarg()

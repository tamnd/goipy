"""Comprehensive atexit test — covers all public API from the Python docs."""
import atexit

# ── register returns the function (usable as decorator) ───────────────────────

@atexit.register
def _decorated():
    print("decorated handler ran")

print(_decorated.__name__)   # _decorated
print(atexit._ncallbacks())  # 1
atexit._clear()


# ── register with positional args ────────────────────────────────────────────

def _greet(name):
    print(f"hello {name}")

atexit.register(_greet, "world")
atexit._run_exitfuncs()   # hello world
atexit._clear()


# ── register with keyword args ───────────────────────────────────────────────

def _add(a, b, extra=0):
    print(a + b + extra)

atexit.register(_add, 1, 2, extra=10)
atexit._run_exitfuncs()   # 13
atexit._clear()


# ── register returns the callable ────────────────────────────────────────────

def _fn(): pass
result = atexit.register(_fn)
print(result is _fn)   # True
atexit._clear()


# ── LIFO order ───────────────────────────────────────────────────────────────

def _a(): print("A")
def _b(): print("B")
def _c(): print("C")

atexit.register(_a)
atexit.register(_b)
atexit.register(_c)
atexit._run_exitfuncs()   # C, B, A
atexit._clear()


# ── multiple registrations of same function all run ──────────────────────────

def _counter(n):
    print(f"counter {n}")

atexit.register(_counter, 1)
atexit.register(_counter, 2)
atexit.register(_counter, 3)
print(atexit._ncallbacks())   # 3
atexit._run_exitfuncs()       # counter 3, counter 2, counter 1
atexit._clear()


# ── unregister removes ALL instances of the function ─────────────────────────

def _multi(): print("multi")

atexit.register(_multi)
atexit.register(_multi)
atexit.register(_multi)
print(atexit._ncallbacks())   # 3
atexit.unregister(_multi)
print(atexit._ncallbacks())   # 0
atexit._clear()


# ── unregister on never-registered function is a no-op ───────────────────────

def _never(): pass
atexit.unregister(_never)     # no error
print(atexit._ncallbacks())   # 0


# ── unregister leaves other handlers intact ──────────────────────────────────

def _keep(): print("kept")
def _remove(): print("removed")

atexit.register(_keep)
atexit.register(_remove)
atexit.register(_keep)
print(atexit._ncallbacks())   # 3
atexit.unregister(_remove)
print(atexit._ncallbacks())   # 2
atexit._run_exitfuncs()       # kept, kept (LIFO: last _keep first)
atexit._clear()


# ── exception in handler is suppressed; remaining handlers still run ──────────

def _raise_err():
    raise ValueError("oops")

def _after_error():
    print("after error")

atexit.register(_after_error)
atexit.register(_raise_err)
atexit._run_exitfuncs()   # after_error still runs
atexit._clear()


# ── lambda registration ───────────────────────────────────────────────────────

atexit.register(lambda: print("lambda ran"))
atexit._run_exitfuncs()   # lambda ran
atexit._clear()


# ── nested register: handler registers another handler ───────────────────────

def _outer():
    print("outer")
    atexit.register(_inner)

def _inner():
    print("inner")

atexit.register(_outer)
atexit._run_exitfuncs()   # outer (inner registered during run)
print(atexit._ncallbacks())   # 1 (_inner was registered during _run_exitfuncs)
atexit._run_exitfuncs()   # inner
atexit._clear()


# ── _ncallbacks reflects current count ───────────────────────────────────────

def _f1(): pass
def _f2(): pass
def _f3(): pass

atexit.register(_f1)
atexit.register(_f2)
atexit.register(_f3)
print(atexit._ncallbacks())   # 3
atexit.unregister(_f2)
print(atexit._ncallbacks())   # 2
atexit._clear()
print(atexit._ncallbacks())   # 0


print("done")

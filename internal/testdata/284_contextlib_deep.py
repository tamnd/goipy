"""Comprehensive contextlib test — covers all public API from the Python docs."""
import contextlib
from contextlib import (
    contextmanager, asynccontextmanager,
    closing, aclosing,
    nullcontext,
    suppress,
    redirect_stdout, redirect_stderr,
    ExitStack, AsyncExitStack,
    AbstractContextManager, AbstractAsyncContextManager,
    ContextDecorator, AsyncContextDecorator,
    chdir,
)
import io
import os


# ── AbstractContextManager ────────────────────────────────────────────────────

def test_abstract_cm():
    class MyCM(AbstractContextManager):
        def __exit__(self, *exc):
            return False
    cm = MyCM()
    with cm as val:
        pass
    print(val is cm)   # True — default __enter__ returns self
    print('abstract_cm ok')

test_abstract_cm()


# ── AbstractAsyncContextManager ───────────────────────────────────────────────

def test_abstract_acm():
    class MyACM(AbstractAsyncContextManager):
        async def __aexit__(self, *exc):
            return False
    acm = MyACM()
    print(isinstance(acm, AbstractAsyncContextManager))   # True
    print('abstract_acm ok')

test_abstract_acm()


# ── ContextDecorator ──────────────────────────────────────────────────────────

def test_context_decorator():
    class MyCD(ContextDecorator):
        def __enter__(self):
            return self
        def __exit__(self, *exc):
            return False

    @MyCD()
    def greet():
        return 'hello'

    result = greet()
    print(result)          # hello
    print('context_decorator ok')

test_context_decorator()


# ── @contextmanager basic ────────────────────────────────────────────────────

def test_contextmanager_basic():
    @contextmanager
    def managed(x):
        yield x * 2

    with managed(5) as val:
        print(val)   # 10
    print('contextmanager_basic ok')

test_contextmanager_basic()


# ── @contextmanager as decorator ─────────────────────────────────────────────

def test_contextmanager_decorator():
    @contextmanager
    def add_one():
        yield

    @add_one()
    def compute():
        return 99

    print(compute())   # 99
    print('contextmanager_decorator ok')

test_contextmanager_decorator()


# ── @contextmanager exception suppression ────────────────────────────────────

def test_contextmanager_suppress():
    @contextmanager
    def swallow():
        try:
            yield
        except ValueError:
            pass

    with swallow():
        raise ValueError("ignored")
    print('contextmanager_suppress ok')

test_contextmanager_suppress()


# ── closing() ────────────────────────────────────────────────────────────────

def test_closing():
    class Closeable:
        def __init__(self):
            self.closed = False
        def close(self):
            self.closed = True

    obj = Closeable()
    with closing(obj):
        pass
    print(obj.closed)   # True
    print('closing ok')

test_closing()


# ── nullcontext() ────────────────────────────────────────────────────────────

def test_nullcontext():
    with nullcontext(42) as val:
        print(val)   # 42
    with nullcontext() as val:
        print(val is None)   # True
    print('nullcontext ok')

test_nullcontext()


# ── suppress() ───────────────────────────────────────────────────────────────

def test_suppress():
    with suppress(ValueError):
        raise ValueError("ignored")
    print('suppress ok')

    # Multiple exception types
    with suppress(KeyError, TypeError):
        raise KeyError("k")
    print('suppress_multi ok')

    # No exception — code runs normally
    x = 0
    with suppress(ValueError):
        x = 7
    print(x)   # 7
    print('suppress_noexc ok')

test_suppress()


# ── redirect_stdout() ────────────────────────────────────────────────────────

def test_redirect_stdout():
    buf = io.StringIO()
    with redirect_stdout(buf):
        print("hello stdout")
    print(buf.getvalue().strip())   # hello stdout
    print('redirect_stdout ok')

test_redirect_stdout()


# ── redirect_stderr() ────────────────────────────────────────────────────────

def test_redirect_stderr():
    buf = io.StringIO()
    import sys
    with redirect_stderr(buf):
        sys.stderr.write("hello stderr\n")
    print(buf.getvalue().strip())   # hello stderr
    print('redirect_stderr ok')

test_redirect_stderr()


# ── chdir() ──────────────────────────────────────────────────────────────────

def test_chdir():
    orig = os.getcwd()
    with chdir('/tmp'):
        inside = os.getcwd()
    after = os.getcwd()
    print(inside == '/tmp' or inside.startswith('/private/tmp'))   # True (macOS /private/tmp)
    print(after == orig)   # True — restored
    print('chdir ok')

test_chdir()


# ── ExitStack basic ──────────────────────────────────────────────────────────

_exitstack_log = []

class _CM:
    def __init__(self, name):
        self.name = name
    def __enter__(self):
        return self.name
    def __exit__(self, *exc):
        _exitstack_log.append(self.name)
        return False

def test_exitstack_basic():
    global _exitstack_log
    _exitstack_log = []

    with ExitStack() as stack:
        a = stack.enter_context(_CM('A'))
        b = stack.enter_context(_CM('B'))
        print(a, b)   # A B

    print(_exitstack_log)   # ['B', 'A']   — LIFO order
    print('exitstack_basic ok')

test_exitstack_basic()


# ── ExitStack.callback() ─────────────────────────────────────────────────────

def test_exitstack_callback():
    log = []

    with ExitStack() as stack:
        stack.callback(log.append, 'x')
        stack.callback(log.append, 'y')

    print(log)   # ['y', 'x']
    print('exitstack_callback ok')

test_exitstack_callback()


# ── ExitStack.push() ─────────────────────────────────────────────────────────

def test_exitstack_push():
    log = []

    def my_exit(exc_type, exc_val, exc_tb):
        log.append('exit_fn')
        return False

    with ExitStack() as stack:
        stack.push(my_exit)

    print(log)   # ['exit_fn']
    print('exitstack_push ok')

test_exitstack_push()


# ── ExitStack suppress exception ─────────────────────────────────────────────

def test_exitstack_suppress():
    with ExitStack() as stack:
        stack.callback(lambda: None)  # no-op first
        stack.enter_context(suppress(ValueError))
        raise ValueError("suppressed")
    print('exitstack_suppress ok')

test_exitstack_suppress()


# ── ExitStack.pop_all() ──────────────────────────────────────────────────────

def test_exitstack_pop_all():
    log = []

    outer = ExitStack()
    with outer as stack:
        stack.callback(log.append, 'a')
        new_stack = stack.pop_all()
    # outer ran with nothing; new_stack holds 'a'
    print(log)   # []
    with new_stack:
        pass
    print(log)   # ['a']
    print('exitstack_pop_all ok')

test_exitstack_pop_all()


# ── AsyncExitStack basic ──────────────────────────────────────────────────────

def test_async_exitstack():
    import asyncio

    async def run():
        log = []
        async with AsyncExitStack() as stack:
            stack.callback(log.append, 'cb')
        print(log)   # ['cb']
        print('async_exitstack ok')

    asyncio.run(run())

test_async_exitstack()


# ── @asynccontextmanager ──────────────────────────────────────────────────────

def test_asynccontextmanager():
    import asyncio

    @asynccontextmanager
    async def amgr(x):
        yield x + 1

    async def run():
        async with amgr(10) as val:
            print(val)   # 11
        print('asynccontextmanager ok')

    asyncio.run(run())

test_asynccontextmanager()


# ── aclosing() ────────────────────────────────────────────────────────────────

def test_aclosing():
    import asyncio

    class AsyncRes:
        def __init__(self):
            self.closed = False
        async def aclose(self):
            self.closed = True

    async def run():
        res = AsyncRes()
        async with aclosing(res):
            pass
        print(res.closed)   # True
        print('aclosing ok')

    asyncio.run(run())

test_aclosing()



# ── suppress() with no exception raised ──────────────────────────────────────

def test_suppress_passthrough():
    result = []
    with suppress(ValueError):
        result.append(1)
    print(result)   # [1]
    print('suppress_passthrough ok')

test_suppress_passthrough()


# ── nullcontext enter_result keyword arg ─────────────────────────────────────

def test_nullcontext_kw():
    with nullcontext(enter_result='hi') as val:
        print(val)   # hi
    print('nullcontext_kw ok')

test_nullcontext_kw()


print("done")

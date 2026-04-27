import contextlib
import io


# ── suppress ────────────────────────────────────────────────────────────────

def test_suppress():
    with contextlib.suppress(ValueError):
        raise ValueError('ignored')
    print('suppress ok')

    with contextlib.suppress(KeyError, TypeError):
        raise KeyError('also ignored')
    print('suppress multi ok')

    with contextlib.suppress(ValueError):
        pass
    print('suppress no-exc ok')


# ── closing ──────────────────────────────────────────────────────────────────

def test_closing():
    class Resource:
        def __init__(self):
            self.closed = False
        def close(self):
            self.closed = True

    r = Resource()
    with contextlib.closing(r) as res:
        print(res is r)
    print(r.closed)
    print('closing ok')


# ── nullcontext ───────────────────────────────────────────────────────────────

def test_nullcontext():
    with contextlib.nullcontext() as val:
        print(val is None)

    with contextlib.nullcontext(42) as val:
        print(val)

    print('nullcontext ok')


# ── contextmanager ────────────────────────────────────────────────────────────

def test_contextmanager():
    @contextlib.contextmanager
    def simple():
        yield 'hello'

    with simple() as v:
        print(v)

    @contextlib.contextmanager
    def with_value(x):
        yield x * 2

    with with_value(10) as v:
        print(v)

    print('contextmanager ok')


def test_contextmanager_cleanup():
    log = []

    @contextlib.contextmanager
    def tracked():
        log.append('enter')
        yield
        log.append('exit')

    with tracked():
        log.append('body')

    print(log)
    print('contextmanager_cleanup ok')


def test_contextmanager_exception_suppress():
    @contextlib.contextmanager
    def suppress_value():
        try:
            yield
        except ValueError:
            pass

    with suppress_value():
        raise ValueError('caught inside generator')
    print('contextmanager_exception_suppress ok')


# ── redirect_stdout ───────────────────────────────────────────────────────────

def test_redirect_stdout():
    buf = io.StringIO()
    with contextlib.redirect_stdout(buf):
        print('captured')
    print(buf.getvalue().strip())
    print('redirect_stdout ok')


def test_redirect_stderr():
    import sys
    buf = io.StringIO()
    with contextlib.redirect_stderr(buf):
        sys.stderr.write('err output\n')
    print(buf.getvalue().strip())
    print('redirect_stderr ok')


# ── ExitStack ─────────────────────────────────────────────────────────────────

def test_exitstack_callback():
    log = []
    with contextlib.ExitStack() as stack:
        stack.callback(log.append, 1)
        stack.callback(log.append, 2)
        stack.callback(log.append, 3)
    print(log)
    print('exitstack_callback ok')


_ec_log = []


class _LoggedCM:
    def __init__(self, name):
        self.name = name
    def __enter__(self):
        _ec_log.append('enter ' + self.name)
        return self
    def __exit__(self, *args):
        _ec_log.append('exit ' + self.name)
        return False


def test_exitstack_enter_context():
    _ec_log.clear()
    with contextlib.ExitStack() as stack:
        stack.enter_context(_LoggedCM('a'))
        stack.enter_context(_LoggedCM('b'))
        _ec_log.append('body')
    print(_ec_log)
    print('exitstack_enter_context ok')


def test_exitstack_suppress():
    log = []

    def suppress_handler(exc_type, exc_val, tb):
        if exc_type is ValueError:
            log.append('suppressed')
            return True
        return False

    with contextlib.ExitStack() as stack:
        stack.push(suppress_handler)
        raise ValueError('will be suppressed')
    print(log)
    print('exitstack_suppress ok')


# ── AbstractContextManager ────────────────────────────────────────────────────

def test_abstract_context_manager():
    print(hasattr(contextlib, 'AbstractContextManager'))
    print(hasattr(contextlib, 'AbstractAsyncContextManager'))
    print('abstract ok')


# ── chdir ─────────────────────────────────────────────────────────────────────

def test_chdir():
    import os
    import tempfile
    original = os.getcwd()
    with tempfile.TemporaryDirectory() as tmpdir:
        with contextlib.chdir(tmpdir):
            inside = os.getcwd()
        restored = os.getcwd()
    print(inside != original)
    print(restored == original)
    print('chdir ok')


# ── module attributes ─────────────────────────────────────────────────────────

def test_module_attrs():
    attrs = [
        'suppress', 'closing', 'nullcontext', 'contextmanager',
        'asynccontextmanager', 'redirect_stdout', 'redirect_stderr',
        'ExitStack', 'AsyncExitStack', 'AbstractContextManager',
        'AbstractAsyncContextManager', 'ContextDecorator',
    ]
    for attr in attrs:
        print(hasattr(contextlib, attr))
    print('module_attrs ok')


test_suppress()
test_closing()
test_nullcontext()
test_contextmanager()
test_contextmanager_cleanup()
test_contextmanager_exception_suppress()
test_redirect_stdout()
test_redirect_stderr()
test_exitstack_callback()
test_exitstack_enter_context()
test_exitstack_suppress()
test_abstract_context_manager()
test_chdir()
test_module_attrs()

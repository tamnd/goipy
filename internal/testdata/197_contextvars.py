"""Tests for contextvars module."""
from contextvars import ContextVar, copy_context, Context, Token
import threading

# ─── ContextVar.name ──────────────────────────────────────────────────────────

def test_name():
    v = ContextVar('myvar')
    assert v.name == 'myvar', f"name={v.name!r}"
    print("name ok")

# ─── get() raises LookupError with no default ─────────────────────────────────

def test_get_no_default():
    v = ContextVar('no_default')
    raised = False
    try:
        v.get()
    except LookupError:
        raised = True
    assert raised, "get() with no value/default should raise LookupError"
    print("get no default ok")

# ─── get(default) returns provided default ────────────────────────────────────

def test_get_arg_default():
    v = ContextVar('arg_default')
    assert v.get('fallback') == 'fallback', "get(default) should return fallback"
    print("get arg default ok")

# ─── ContextVar with default= kwarg ───────────────────────────────────────────

def test_var_default():
    v = ContextVar('with_default', default=42)
    assert v.get() == 42, f"got {v.get()}"
    print("var default ok")

# ─── set() and get() ──────────────────────────────────────────────────────────

def test_set_get():
    v = ContextVar('sg')
    v.set('hello')
    assert v.get() == 'hello', f"got {v.get()!r}"
    print("set get ok")

# ─── reset() restores previous value ─────────────────────────────────────────

def test_reset():
    v = ContextVar('reset_var')
    tok = v.set('first')
    assert v.get() == 'first'
    tok2 = v.set('second')
    assert v.get() == 'second'
    v.reset(tok2)
    assert v.get() == 'first', f"after reset tok2: {v.get()!r}"
    v.reset(tok)
    raised = False
    try:
        v.get()
    except LookupError:
        raised = True
    assert raised, "after reset to initial state, should raise LookupError"
    print("reset ok")

# ─── Token.var and Token.old_value ────────────────────────────────────────────

def test_token_attrs():
    v = ContextVar('tok_var')
    tok = v.set(99)
    assert tok.var is v, "token.var should be the ContextVar"
    assert tok.old_value is Token.MISSING, f"first set: old_value should be MISSING, got {tok.old_value!r}"
    tok2 = v.set(100)
    assert tok2.old_value == 99, f"second set: old_value should be 99, got {tok2.old_value!r}"
    print("token attrs ok")

# ─── copy_context() returns independent snapshot ─────────────────────────────

def test_copy_context():
    v = ContextVar('copy_var')
    v.set('original')
    ctx = copy_context()
    v.set('modified')
    # The copy should still see 'original'
    def check():
        assert v.get() == 'original', f"in copy: {v.get()!r}"
    ctx.run(check)
    # Current context still has 'modified'
    assert v.get() == 'modified', f"after copy check: {v.get()!r}"
    print("copy context ok")

# ─── Context.run() isolates changes ──────────────────────────────────────────

def test_context_run_isolation():
    v = ContextVar('iso_var')
    v.set('outer')
    ctx = copy_context()
    def inner():
        v.set('inner')
        assert v.get() == 'inner'
    ctx.run(inner)
    # outer context unchanged
    assert v.get() == 'outer', f"after ctx.run: {v.get()!r}"
    print("context run isolation ok")

# ─── Context mapping interface ────────────────────────────────────────────────

def test_context_mapping():
    v1 = ContextVar('m1')
    v2 = ContextVar('m2')
    v1.set(10)
    v2.set(20)
    ctx = copy_context()

    assert v1 in ctx, "v1 should be in context"
    assert ctx[v1] == 10, f"ctx[v1]={ctx[v1]}"
    assert ctx.get(v1) == 10
    assert ctx.get(v1, 999) == 10
    unknown = ContextVar('unknown')
    assert ctx.get(unknown, -1) == -1
    assert len(ctx) >= 2

    keys = list(ctx.keys())
    assert v1 in keys
    vals = list(ctx.values())
    assert 10 in vals
    items = list(ctx.items())
    assert (v1, 10) in items
    print("context mapping ok")

# ─── Context.copy() ───────────────────────────────────────────────────────────

def test_context_copy():
    v = ContextVar('cc_var')
    v.set('a')
    ctx = copy_context()
    ctx2 = ctx.copy()
    def modify():
        v.set('b')
    ctx.run(modify)
    # ctx2 should still have 'a'
    def check():
        assert v.get() == 'a', f"ctx2 should have 'a', got {v.get()!r}"
    ctx2.run(check)
    print("context copy ok")

# ─── set() as context manager (Python 3.14) ──────────────────────────────────

def test_set_context_manager():
    v = ContextVar('cm_var')
    v.set('before')
    with v.set('inside'):
        assert v.get() == 'inside', f"inside: {v.get()!r}"
    assert v.get() == 'before', f"after: {v.get()!r}"
    print("set context manager ok")

# ─── Token as context manager (Python 3.14) ──────────────────────────────────

def test_token_context_manager():
    v = ContextVar('tok_cm')
    v.set('start')
    tok = v.set('middle')
    with tok:
        pass  # __exit__ resets
    assert v.get() == 'start', f"after tok cm: {v.get()!r}"
    print("token context manager ok")

# ─── Nested contexts ──────────────────────────────────────────────────────────

def test_nested():
    v = ContextVar('nested')
    v.set('root')
    ctx1 = copy_context()
    def level1():
        v.set('l1')
        ctx2 = copy_context()
        def level2():
            v.set('l2')
            assert v.get() == 'l2'
        ctx2.run(level2)
        assert v.get() == 'l1', f"l1 after l2: {v.get()!r}"
    ctx1.run(level1)
    assert v.get() == 'root', f"root after l1: {v.get()!r}"
    print("nested ok")

# ─── Thread isolation ─────────────────────────────────────────────────────────

def test_thread_isolation():
    v = ContextVar('thread_var', default='main')
    v.set('main_value')
    results = {}
    lock = threading.Lock()

    def worker(tid):
        v.set(f'thread_{tid}')
        import time; time.sleep(0.01)
        with lock:
            results[tid] = v.get()

    threads = [threading.Thread(target=worker, args=(i,)) for i in range(3)]
    for t in threads: t.start()
    for t in threads: t.join()

    for tid in range(3):
        assert results[tid] == f'thread_{tid}', f"tid={tid} got {results[tid]!r}"
    # main context unchanged
    assert v.get() == 'main_value'
    print("thread isolation ok")

# ─── Context.run() return value ───────────────────────────────────────────────

def test_context_run_return():
    ctx = copy_context()
    result = ctx.run(lambda: 42)
    assert result == 42, f"result={result}"
    print("context run return ok")

# ─── Context.__iter__ ─────────────────────────────────────────────────────────

def test_context_iter():
    v = ContextVar('iter_var')
    v.set('x')
    ctx = copy_context()
    found = False
    for key in ctx:
        if key is v:
            found = True
    assert found, "iterating context should yield ContextVar keys"
    print("context iter ok")

if __name__ == "__main__":
    test_name()
    test_get_no_default()
    test_get_arg_default()
    test_var_default()
    test_set_get()
    test_reset()
    test_token_attrs()
    test_copy_context()
    test_context_run_isolation()
    test_context_mapping()
    test_context_copy()
    test_set_context_manager()
    test_token_context_manager()
    test_nested()
    test_thread_isolation()
    test_context_run_return()
    test_context_iter()
    print("ALL OK")

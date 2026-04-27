"""Fixture 262 — sys audit events (sys.addaudithook / sys.audit)"""
import sys
import os


# ── 1. Basic hook + audit ──────────────────────────────────────────────────
def test_basic_hook():
    events = []
    def hook(event, args):
        if event == "test262.basic":
            events.append(args)
    sys.addaudithook(hook)
    sys.audit("test262.basic", "hello", 42)
    assert events == [("hello", 42)], repr(events)
    print("test_basic_hook: OK")


# ── 2. audit with no extra args ───────────────────────────────────────────
def test_audit_no_args():
    events = []
    def hook(event, args):
        if event == "test262.noargs":
            events.append(args)
    sys.addaudithook(hook)
    sys.audit("test262.noargs")
    assert events == [()], repr(events)
    print("test_audit_no_args: OK")


# ── 3. args delivered as a tuple ──────────────────────────────────────────
def test_args_is_tuple():
    types = []
    def hook(event, args):
        if event == "test262.argtuple":
            types.append(type(args).__name__)
    sys.addaudithook(hook)
    sys.audit("test262.argtuple", 1, 2, 3)
    assert types == ["tuple"], repr(types)
    print("test_args_is_tuple: OK")


# ── 4. event name delivered as str ────────────────────────────────────────
def test_event_name_is_str():
    names = []
    def hook(event, args):
        if event == "test262.nametype":
            names.append(type(event).__name__)
    sys.addaudithook(hook)
    sys.audit("test262.nametype")
    assert names == ["str"], repr(names)
    print("test_event_name_is_str: OK")


# ── 5. Multiple hooks fire in registration order ──────────────────────────
def test_multiple_hooks_order():
    order = []
    def h1(event, args):
        if event == "test262.order":
            order.append(1)
    def h2(event, args):
        if event == "test262.order":
            order.append(2)
    def h3(event, args):
        if event == "test262.order":
            order.append(3)
    sys.addaudithook(h1)
    sys.addaudithook(h2)
    sys.addaudithook(h3)
    sys.audit("test262.order")
    assert order == [1, 2, 3], repr(order)
    print("test_multiple_hooks_order: OK")


# ── 6. RuntimeError from hook is suppressed; later hooks still run ────────
def test_runtimeerror_suppressed():
    log = []
    def bad(event, args):
        if event == "test262.rerr":
            raise RuntimeError("should suppress")
    def good(event, args):
        if event == "test262.rerr":
            log.append("ran")
    sys.addaudithook(bad)
    sys.addaudithook(good)
    sys.audit("test262.rerr")   # must not raise
    assert log == ["ran"], repr(log)
    print("test_runtimeerror_suppressed: OK")


# ── 7. Non-RuntimeError from hook propagates ─────────────────────────────
def test_exception_propagates():
    def hook(event, args):
        if event == "test262.exc":
            raise ValueError("propagated")
    sys.addaudithook(hook)
    try:
        sys.audit("test262.exc")
        assert False, "expected ValueError"
    except ValueError as e:
        assert str(e) == "propagated"
    print("test_exception_propagates: OK")


# ── 8. Hooks persist across multiple audit calls ──────────────────────────
def test_hooks_persist():
    count = [0]
    def hook(event, args):
        if event == "test262.persist":
            count[0] += 1
    sys.addaudithook(hook)
    sys.audit("test262.persist")
    sys.audit("test262.persist")
    sys.audit("test262.persist")
    assert count[0] == 3, repr(count)
    print("test_hooks_persist: OK")


# ── 9. Arg values preserved exactly ──────────────────────────────────────
def test_args_values():
    received = []
    def hook(event, args):
        if event == "test262.vals":
            received.append(args)
    sys.addaudithook(hook)
    sys.audit("test262.vals", "str", 42, 3.14, True, None)
    assert len(received) == 1
    assert received[0][0] == "str"
    assert received[0][1] == 42
    assert received[0][2] == 3.14
    assert received[0][3] is True
    assert received[0][4] is None
    print("test_args_values: OK")


# ── 10. open() fires 'open' audit event ──────────────────────────────────
def test_open_fires_event():
    paths = []
    def hook(event, args):
        if event == "open":
            paths.append(args[0])
    sys.addaudithook(hook)
    devnull = os.devnull
    try:
        f = open(devnull, "r")
        f.close()
    except OSError:
        pass
    assert devnull in paths, repr(paths)
    print("test_open_fires_event: OK")


# ── 11. open() event args: (path, mode, flags) ───────────────────────────
def test_open_event_args():
    opens = []
    def hook(event, args):
        if event == "open":
            opens.append(args)
    sys.addaudithook(hook)
    devnull = os.devnull
    try:
        f = open(devnull, "w")
        f.close()
    except OSError:
        pass
    found = [a for a in opens if a[0] == devnull and a[1] == "w"]
    assert found, repr(opens)
    assert isinstance(found[0][2], int)
    print("test_open_event_args: OK")


# ── 12. Existing hooks notified when new hook is added ───────────────────
def test_addaudithook_event():
    seen = []
    def first(event, args):
        if event == "sys.addaudithook":
            seen.append("notified")
    sys.addaudithook(first)
    def second(event, args):
        pass
    sys.addaudithook(second)   # first should be notified
    assert "notified" in seen, repr(seen)
    print("test_addaudithook_event: OK")


# ── 13. audit event name is passed verbatim ───────────────────────────────
def test_event_name_verbatim():
    seen = []
    def hook(event, args):
        if event.startswith("test262.verbatim"):
            seen.append(event)
    sys.addaudithook(hook)
    sys.audit("test262.verbatim.foo")
    sys.audit("test262.verbatim.bar")
    assert seen == ["test262.verbatim.foo", "test262.verbatim.bar"], repr(seen)
    print("test_event_name_verbatim: OK")


if __name__ == "__main__":
    test_basic_hook()
    test_audit_no_args()
    test_args_is_tuple()
    test_event_name_is_str()
    test_multiple_hooks_order()
    test_runtimeerror_suppressed()
    test_exception_propagates()
    test_hooks_persist()
    test_args_values()
    test_open_fires_event()
    test_open_event_args()
    test_addaudithook_event()
    test_event_name_verbatim()

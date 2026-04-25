"""Tests for selectors module."""
import selectors
import socket
import threading

# ─── constants ────────────────────────────────────────────────────────────────

def test_constants():
    assert selectors.EVENT_READ == 1
    assert selectors.EVENT_WRITE == 2
    assert isinstance(selectors.EVENT_READ, int)
    assert isinstance(selectors.EVENT_WRITE, int)
    print("constants ok")

# ─── SelectorKey ─────────────────────────────────────────────────────────────

def test_selector_key():
    a, b = socket.socketpair()
    try:
        sel = selectors.DefaultSelector()
        key = sel.register(a, selectors.EVENT_READ, data="mydata")
        assert key.fileobj is a
        assert isinstance(key.fd, int) and key.fd >= 0
        assert key.events == selectors.EVENT_READ
        assert key.data == "mydata"
        sel.close()
    finally:
        a.close()
        b.close()
    print("selector_key ok")

# ─── DefaultSelector readable ────────────────────────────────────────────────

def test_default_selector_readable():
    a, b = socket.socketpair()
    try:
        with selectors.DefaultSelector() as sel:
            sel.register(a, selectors.EVENT_READ)
            # Nothing written — should time out
            ready = sel.select(timeout=0)
            assert ready == [], f"expected empty, got {ready}"
            # Write data then poll
            b.sendall(b'hello')
            ready = sel.select(timeout=1.0)
            assert len(ready) == 1, f"expected 1 event, got {ready}"
            key, events = ready[0]
            assert key.fileobj is a
            assert events & selectors.EVENT_READ
    finally:
        a.close()
        b.close()
    print("default_selector_readable ok")

# ─── DefaultSelector writable ────────────────────────────────────────────────

def test_default_selector_writable():
    a, b = socket.socketpair()
    try:
        with selectors.DefaultSelector() as sel:
            sel.register(a, selectors.EVENT_WRITE)
            ready = sel.select(timeout=0.5)
            assert len(ready) >= 1, f"expected writable, got {ready}"
            key, events = ready[0]
            assert key.fileobj is a
            assert events & selectors.EVENT_WRITE
    finally:
        a.close()
        b.close()
    print("default_selector_writable ok")

# ─── register / unregister / get_key / get_map ───────────────────────────────

def test_register_unregister():
    a, b = socket.socketpair()
    try:
        sel = selectors.DefaultSelector()
        key = sel.register(a, selectors.EVENT_READ | selectors.EVENT_WRITE)
        assert key.events == selectors.EVENT_READ | selectors.EVENT_WRITE

        # get_key returns same key
        k2 = sel.get_key(a)
        assert k2.fd == key.fd

        # get_map contains our fd
        m = sel.get_map()
        assert key.fd in m

        # unregister removes it
        removed = sel.unregister(a)
        assert removed.fd == key.fd
        m2 = sel.get_map()
        assert key.fd not in m2

        # double unregister raises KeyError
        try:
            sel.unregister(a)
            assert False, "expected KeyError"
        except KeyError:
            pass

        sel.close()
    finally:
        a.close()
        b.close()
    print("register_unregister ok")

# ─── modify ──────────────────────────────────────────────────────────────────

def test_modify():
    a, b = socket.socketpair()
    try:
        sel = selectors.DefaultSelector()
        sel.register(a, selectors.EVENT_READ, data=1)

        # modify changes data
        key2 = sel.modify(a, selectors.EVENT_READ, data=2)
        assert key2.data == 2

        # modify changes events
        key3 = sel.modify(a, selectors.EVENT_READ | selectors.EVENT_WRITE)
        assert key3.events == selectors.EVENT_READ | selectors.EVENT_WRITE

        sel.close()
    finally:
        a.close()
        b.close()
    print("modify ok")

# ─── select timeout ──────────────────────────────────────────────────────────

def test_select_timeout():
    a, b = socket.socketpair()
    try:
        with selectors.DefaultSelector() as sel:
            sel.register(a, selectors.EVENT_READ)
            ready = sel.select(timeout=0)
            assert ready == [], f"expected [], got {ready}"
    finally:
        a.close()
        b.close()
    print("select_timeout ok")

# ─── context manager ─────────────────────────────────────────────────────────

def test_context_manager():
    a, b = socket.socketpair()
    try:
        with selectors.DefaultSelector() as sel:
            sel.register(a, selectors.EVENT_READ)
            # selector is usable inside the block
            ready = sel.select(timeout=0)
            assert isinstance(ready, list)
        # after __exit__ the selector is closed; select() should raise
        try:
            sel.select(timeout=0)
            assert False, "expected error after close"
        except (OSError, ValueError, RuntimeError):
            pass
    finally:
        a.close()
        b.close()
    print("context_manager ok")

# ─── EpollSelector (Linux) ────────────────────────────────────────────────────

def test_epoll_selector():
    if not hasattr(selectors, 'EpollSelector'):
        print("epoll_selector ok")
        return

    a, b = socket.socketpair()
    try:
        with selectors.EpollSelector() as sel:
            sel.register(a, selectors.EVENT_READ)
            ready = sel.select(timeout=0)
            assert ready == [], f"expected empty, got {ready}"

            b.sendall(b'epoll')
            ready = sel.select(timeout=0.5)
            assert len(ready) == 1
            key, events = ready[0]
            assert key.fileobj is a
            assert events & selectors.EVENT_READ
    finally:
        a.close()
        b.close()
    print("epoll_selector ok")

# ─── KqueueSelector (macOS/BSD) ──────────────────────────────────────────────

def test_kqueue_selector():
    if not hasattr(selectors, 'KqueueSelector'):
        print("kqueue_selector ok")
        return

    a, b = socket.socketpair()
    try:
        with selectors.KqueueSelector() as sel:
            sel.register(a, selectors.EVENT_READ)
            ready = sel.select(timeout=0)
            assert ready == [], f"expected empty, got {ready}"

            b.sendall(b'kqueue')
            ready = sel.select(timeout=0.5)
            assert len(ready) >= 1
            key, events = ready[0]
            assert key.fileobj is a
            assert events & selectors.EVENT_READ
    finally:
        a.close()
        b.close()
    print("kqueue_selector ok")

if __name__ == "__main__":
    test_constants()
    test_selector_key()
    test_default_selector_readable()
    test_default_selector_writable()
    test_register_unregister()
    test_modify()
    test_select_timeout()
    test_context_manager()
    test_epoll_selector()
    test_kqueue_selector()
    print("ALL OK")

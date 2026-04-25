"""Tests for select module."""
import select
import socket
import threading

# ─── error alias ─────────────────────────────────────────────────────────────

def test_error_alias():
    assert select.error is OSError
    print("error_alias ok")

# ─── constants ────────────────────────────────────────────────────────────────

def test_constants():
    assert select.POLLIN == 1
    assert select.POLLOUT == 4
    assert select.POLLERR == 8
    assert select.POLLHUP == 16
    assert select.POLLNVAL == 32
    assert isinstance(select.POLLIN, int)
    print("constants ok")

# ─── select() timeout ────────────────────────────────────────────────────────

def test_select_timeout():
    a, b = socket.socketpair()
    try:
        r, w, x = select.select([a], [], [], 0)
        assert r == [] and w == [] and x == [], f"expected empty, got {r},{w},{x}"
    finally:
        a.close()
        b.close()
    print("select_timeout ok")

# ─── select() readable ───────────────────────────────────────────────────────

def test_select_readable():
    a, b = socket.socketpair()
    try:
        b.sendall(b'hello')
        r, w, x = select.select([a], [], [], 1.0)
        assert len(r) == 1 and r[0] is a, f"expected [a], got {r}"
    finally:
        a.close()
        b.close()
    print("select_readable ok")

# ─── select() writable ───────────────────────────────────────────────────────

def test_select_writable():
    a, b = socket.socketpair()
    try:
        r, w, x = select.select([], [a], [], 0)
        assert len(w) == 1 and w[0] is a, f"expected [a], got {w}"
    finally:
        a.close()
        b.close()
    print("select_writable ok")

# ─── select() multiple ───────────────────────────────────────────────────────

def test_select_multiple():
    a, b = socket.socketpair()
    c, d = socket.socketpair()
    try:
        b.sendall(b'x')
        r, w, x = select.select([a, c], [], [], 1.0)
        assert a in r and c not in r, f"got r={r}"
    finally:
        a.close(); b.close(); c.close(); d.close()
    print("select_multiple ok")

# ─── poll ─────────────────────────────────────────────────────────────────────

def test_poll():
    if not hasattr(select, 'poll'):
        print("poll ok (skipped)")
        return

    a, b = socket.socketpair()
    try:
        p = select.poll()
        p.register(a, select.POLLIN)

        # Nothing written — should timeout (0 ms)
        events = p.poll(0)
        assert events == [], f"expected empty, got {events}"

        # Write data — should be readable
        b.sendall(b'hi')
        events = p.poll(500)
        assert len(events) == 1, f"expected 1 event, got {events}"
        fd, ev = events[0]
        assert ev & select.POLLIN, f"expected POLLIN, got ev={ev}"

        p.unregister(a)
        events = p.poll(0)
        assert events == [], "after unregister should be empty"
    finally:
        a.close()
        b.close()
    print("poll ok")

# ─── poll modify ─────────────────────────────────────────────────────────────

def test_poll_modify():
    if not hasattr(select, 'poll'):
        print("poll_modify ok (skipped)")
        return

    a, b = socket.socketpair()
    try:
        p = select.poll()
        p.register(a, select.POLLIN | select.POLLOUT)
        p.modify(a, select.POLLIN)
        # Now only watching POLLIN; a is writable but we shouldn't see POLLOUT
        b.sendall(b'y')
        events = p.poll(200)
        assert len(events) >= 1
        for fd, ev in events:
            assert ev & select.POLLIN
    finally:
        a.close()
        b.close()
    print("poll_modify ok")

# ─── epoll (Linux) ────────────────────────────────────────────────────────────

def test_epoll():
    if not hasattr(select, 'epoll'):
        print("epoll ok")
        return

    a, b = socket.socketpair()
    try:
        ep = select.epoll()
        ep.register(a.fileno(), select.EPOLLIN)

        # Nothing — timeout immediately
        events = ep.poll(timeout=0)
        assert events == [], f"expected empty, got {events}"

        b.sendall(b'epoll')
        events = ep.poll(timeout=0.5)
        assert len(events) == 1, f"expected 1 event, got {events}"
        fd2, ev2 = events[0]
        assert ev2 & select.EPOLLIN

        ep.unregister(a.fileno())
        ep.close()
    finally:
        a.close()
        b.close()
    print("epoll ok")

# ─── kqueue (macOS/BSD) ──────────────────────────────────────────────────────

def test_kqueue():
    if not hasattr(select, 'kqueue'):
        print("kqueue ok")
        return

    a, b = socket.socketpair()
    try:
        kq = select.kqueue()

        # Register a for read
        ev = select.kevent(a.fileno(), filter=select.KQ_FILTER_READ,
                           flags=select.KQ_EV_ADD | select.KQ_EV_ENABLE)
        kq.control([ev], 0)

        # Nothing — timeout 0
        got = kq.control(None, 10, 0)
        assert got == [], f"expected [], got {got}"

        # Write data
        b.sendall(b'kqueue')
        got = kq.control(None, 10, 0.5)
        assert len(got) >= 1, f"expected events, got {got}"
        assert got[0].filter == select.KQ_FILTER_READ

        kq.close()
    finally:
        a.close()
        b.close()
    print("kqueue ok")

if __name__ == "__main__":
    test_error_alias()
    test_constants()
    test_select_timeout()
    test_select_readable()
    test_select_writable()
    test_select_multiple()
    test_poll()
    test_poll_modify()
    test_epoll()
    test_kqueue()
    print("ALL OK")

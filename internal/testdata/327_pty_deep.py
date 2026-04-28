import pty

# ── constants ──────────────────────────────────────────────────────────────────
assert pty.CHILD == 0
assert pty.STDIN_FILENO == 0
assert pty.STDOUT_FILENO == 1
assert pty.STDERR_FILENO == 2

# ── all public callable names ──────────────────────────────────────────────────
assert callable(pty.openpty)
assert callable(pty.fork)
assert callable(pty.spawn)
assert callable(pty.close)
assert callable(pty.waitpid)
assert callable(pty.setraw)
assert callable(pty.tcgetattr)
assert callable(pty.tcsetattr)

# ── openpty: either returns (master_fd, slave_fd) or raises OSError ───────────
try:
    result = pty.openpty()
    assert isinstance(result, tuple)
    assert len(result) == 2
    assert isinstance(result[0], int)
    assert isinstance(result[1], int)
    import os
    os.close(result[0])
    os.close(result[1])
except OSError:
    pass

# ── fork and spawn: only callable check (unsafe to call in test context) ──────
assert callable(pty.fork)
assert callable(pty.spawn)

# ── close: raises OSError on invalid fd, or succeeds ─────────────────────────
try:
    pty.close(99999)
except OSError:
    pass

# ── setraw raises on non-tty fd (termios.error is a subclass of OSError in
#    CPython but a plain OSError in goipy -- catch the common base) ────────────
try:
    pty.setraw(0)
except Exception:
    pass

# ── tcgetattr raises on non-tty fd ───────────────────────────────────────────
try:
    pty.tcgetattr(0)
except Exception:
    pass

# ── tcsetattr is callable ─────────────────────────────────────────────────────
assert callable(pty.tcsetattr)

# ── waitpid returns a 2-tuple or raises (no child in test context) ────────────
try:
    r = pty.waitpid(0, 1)  # WNOHANG=1
    assert isinstance(r, tuple) and len(r) == 2
except OSError:
    pass

print('ok')

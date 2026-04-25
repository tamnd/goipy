"""Tests for signal module."""
import signal
import os

# ─── constants ────────────────────────────────────────────────────────────────

def test_constants():
    assert signal.SIG_DFL == 0, f"SIG_DFL={signal.SIG_DFL}"
    assert signal.SIG_IGN == 1, f"SIG_IGN={signal.SIG_IGN}"
    assert isinstance(signal.SIGINT, int) and signal.SIGINT > 0
    assert isinstance(signal.SIGTERM, int) and signal.SIGTERM > 0
    assert isinstance(signal.SIGUSR1, int) and signal.SIGUSR1 > 0
    assert isinstance(signal.SIGUSR2, int) and signal.SIGUSR2 > 0
    assert isinstance(signal.SIGALRM, int) and signal.SIGALRM > 0
    assert signal.ITIMER_REAL == 0
    assert signal.ITIMER_VIRTUAL == 1
    assert signal.ITIMER_PROF == 2
    assert signal.SIG_BLOCK == 1
    assert signal.SIG_UNBLOCK == 2
    assert signal.SIG_SETMASK == 3
    print("constants ok")

# ─── strsignal ────────────────────────────────────────────────────────────────

def test_strsignal():
    desc = signal.strsignal(signal.SIGINT)
    assert isinstance(desc, str) and len(desc) > 0, f"strsignal(SIGINT)={desc!r}"
    desc2 = signal.strsignal(signal.SIGTERM)
    assert isinstance(desc2, str) and len(desc2) > 0
    # Unknown signal returns None or raises ValueError depending on platform
    try:
        none_val = signal.strsignal(999)
        assert none_val is None, f"strsignal(999)={none_val!r}"
    except ValueError:
        pass  # CPython raises ValueError for out-of-range
    print("strsignal ok")

# ─── valid_signals ────────────────────────────────────────────────────────────

def test_valid_signals():
    vs = signal.valid_signals()
    assert signal.SIGINT in vs, f"SIGINT not in valid_signals"
    assert signal.SIGTERM in vs, f"SIGTERM not in valid_signals"
    print("valid_signals ok")

# ─── default_int_handler ──────────────────────────────────────────────────────

def test_default_int_handler():
    try:
        signal.default_int_handler(signal.SIGINT, None)
        assert False, "should have raised KeyboardInterrupt"
    except KeyboardInterrupt:
        pass
    print("default_int_handler ok")

# ─── getsignal default ────────────────────────────────────────────────────────

def test_getsignal_default():
    # SIGTERM starts as SIG_DFL (0)
    h = signal.getsignal(signal.SIGTERM)
    assert h == signal.SIG_DFL or h == 0, f"expected SIG_DFL for SIGTERM, got {h}"
    # SIGINT starts as default_int_handler
    h2 = signal.getsignal(signal.SIGINT)
    assert callable(h2) or h2 == signal.SIG_DFL, f"unexpected SIGINT handler {h2}"
    print("getsignal_default ok")

# ─── signal handler ───────────────────────────────────────────────────────────

def test_signal_handler():
    counter = [0]
    def handler(signum, frame):
        counter[0] += 1

    old = signal.signal(signal.SIGUSR1, handler)
    # old should be SIG_DFL or previous callable
    assert old == signal.SIG_DFL or callable(old) or old == 0

    # raise_signal should invoke the handler synchronously
    signal.raise_signal(signal.SIGUSR1)
    assert counter[0] == 1, f"handler called {counter[0]} times, expected 1"

    # Restore
    signal.signal(signal.SIGUSR1, signal.SIG_DFL)
    print("signal_handler ok")

# ─── SIG_IGN ─────────────────────────────────────────────────────────────────

def test_sig_ign():
    signal.signal(signal.SIGUSR2, signal.SIG_IGN)
    h = signal.getsignal(signal.SIGUSR2)
    assert h == signal.SIG_IGN or h == 1, f"expected SIG_IGN, got {h}"

    # raise_signal with SIG_IGN should not crash or call anything
    signal.raise_signal(signal.SIGUSR2)

    # Restore
    signal.signal(signal.SIGUSR2, signal.SIG_DFL)
    print("sig_ign ok")

# ─── set_wakeup_fd ────────────────────────────────────────────────────────────

def test_set_wakeup_fd():
    # set_wakeup_fd returns the old fd as int
    old = signal.set_wakeup_fd(-1)
    assert isinstance(old, int), f"set_wakeup_fd returned {old!r}"

    # Set and reset
    old2 = signal.set_wakeup_fd(-1)
    assert isinstance(old2, int)
    print("set_wakeup_fd ok")

# ─── alarm ────────────────────────────────────────────────────────────────────

def test_alarm():
    if not hasattr(signal, 'alarm'):
        print("alarm ok")
        return
    # alarm(0) cancels any pending alarm and returns remaining time
    rem = signal.alarm(0)
    assert isinstance(rem, int), f"alarm(0) returned {rem!r}"
    # Set a short alarm and immediately cancel it
    signal.alarm(10)
    rem2 = signal.alarm(0)
    assert isinstance(rem2, int), f"alarm(0) cancel returned {rem2!r}"
    print("alarm ok")

# ─── pthread_sigmask ──────────────────────────────────────────────────────────

def test_pthread_sigmask():
    if not hasattr(signal, 'pthread_sigmask'):
        print("pthread_sigmask ok")
        return
    # Get current mask (block nothing additional) — returns a frozenset/set
    old_mask = signal.pthread_sigmask(signal.SIG_BLOCK, [])

    # Block SIGUSR1, then unblock it
    signal.pthread_sigmask(signal.SIG_BLOCK, [signal.SIGUSR1])
    signal.pthread_sigmask(signal.SIG_UNBLOCK, [signal.SIGUSR1])
    print("pthread_sigmask ok")

if __name__ == "__main__":
    test_constants()
    test_strsignal()
    test_valid_signals()
    test_default_int_handler()
    test_getsignal_default()
    test_signal_handler()
    test_sig_ign()
    test_set_wakeup_fd()
    test_alarm()
    test_pthread_sigmask()
    print("ALL OK")

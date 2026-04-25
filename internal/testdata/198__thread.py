"""Tests for _thread module."""
import _thread
import time

# ─── allocate_lock basic ──────────────────────────────────────────────────────

def test_allocate_lock():
    lock = _thread.allocate_lock()
    assert lock is not None
    assert lock.locked() == False, "new lock should be unlocked"
    print("allocate_lock ok")

# ─── acquire / release ────────────────────────────────────────────────────────

def test_acquire_release():
    lock = _thread.allocate_lock()
    result = lock.acquire()
    assert result == True, f"acquire() should return True, got {result}"
    assert lock.locked() == True
    lock.release()
    assert lock.locked() == False
    print("acquire release ok")

# ─── acquire blocking=False on unlocked ───────────────────────────────────────

def test_acquire_nonblocking_unlocked():
    lock = _thread.allocate_lock()
    got = lock.acquire(False)
    assert got == True, f"acquire(False) on unlocked should return True, got {got}"
    lock.release()
    print("acquire nonblocking unlocked ok")

# ─── acquire blocking=False on locked ────────────────────────────────────────

def test_acquire_nonblocking_locked():
    lock = _thread.allocate_lock()
    lock.acquire()
    got = lock.acquire(False)
    assert got == False, f"acquire(False) on locked should return False, got {got}"
    lock.release()
    print("acquire nonblocking locked ok")

# ─── acquire timeout on locked lock ──────────────────────────────────────────

def test_acquire_timeout():
    lock = _thread.allocate_lock()
    lock.acquire()
    got = lock.acquire(timeout=0.05)
    assert got == False, f"acquire(timeout=0.05) on locked should return False, got {got}"
    lock.release()
    print("acquire timeout ok")

# ─── lock context manager ─────────────────────────────────────────────────────

def test_lock_context_manager():
    lock = _thread.allocate_lock()
    with lock:
        assert lock.locked() == True
    assert lock.locked() == False
    print("lock context manager ok")

# ─── get_ident ────────────────────────────────────────────────────────────────

def test_get_ident():
    ident = _thread.get_ident()
    assert isinstance(ident, int), f"get_ident() should return int, got {type(ident)}"
    assert ident != 0, "get_ident() should return non-zero"
    print("get_ident ok")

# ─── get_native_id ────────────────────────────────────────────────────────────

def test_get_native_id():
    nid = _thread.get_native_id()
    assert isinstance(nid, int), f"get_native_id() should return int, got {type(nid)}"
    assert nid > 0, f"get_native_id() should return positive, got {nid}"
    print("get_native_id ok")

# ─── start_new_thread runs function ──────────────────────────────────────────

def test_start_new_thread():
    results = []
    lock = _thread.allocate_lock()
    lock.acquire()

    def worker(x):
        results.append(x * 2)
        lock.release()

    _thread.start_new_thread(worker, (21,))
    lock.acquire()  # wait for worker to finish
    lock.release()
    assert results == [42], f"results={results}"
    print("start_new_thread ok")

# ─── start_new_thread with kwargs ────────────────────────────────────────────

def test_start_new_thread_kwargs():
    results = []
    done = _thread.allocate_lock()
    done.acquire()

    def worker(a, b=0):
        results.append(a + b)
        done.release()

    _thread.start_new_thread(worker, (10,), {'b': 5})
    done.acquire()
    done.release()
    assert results == [15], f"results={results}"
    print("start_new_thread kwargs ok")

# ─── start_new_thread returns int identifier ─────────────────────────────────

def test_start_new_thread_returns_id():
    done = _thread.allocate_lock()
    done.acquire()

    def worker():
        done.release()

    tid = _thread.start_new_thread(worker, ())
    assert isinstance(tid, int), f"start_new_thread should return int, got {type(tid)}"
    assert tid != 0
    done.acquire()
    done.release()
    print("start_new_thread returns id ok")

# ─── exit() raises SystemExit ────────────────────────────────────────────────

def test_exit():
    raised = False
    try:
        _thread.exit()
    except SystemExit:
        raised = True
    assert raised, "exit() should raise SystemExit"
    print("exit ok")

# ─── error is RuntimeError ────────────────────────────────────────────────────

def test_error():
    assert issubclass(_thread.error, RuntimeError), \
        f"_thread.error should be RuntimeError subclass, got {_thread.error}"
    print("error ok")

# ─── TIMEOUT_MAX ──────────────────────────────────────────────────────────────

def test_timeout_max():
    assert isinstance(_thread.TIMEOUT_MAX, float), \
        f"TIMEOUT_MAX should be float, got {type(_thread.TIMEOUT_MAX)}"
    assert _thread.TIMEOUT_MAX > 1e8, f"TIMEOUT_MAX={_thread.TIMEOUT_MAX}"
    print("TIMEOUT_MAX ok")

# ─── stack_size ───────────────────────────────────────────────────────────────

def test_stack_size():
    sz = _thread.stack_size()
    assert isinstance(sz, int), f"stack_size() should return int, got {type(sz)}"
    prev = _thread.stack_size(0)
    assert isinstance(prev, int)
    print("stack_size ok")

# ─── LockType ─────────────────────────────────────────────────────────────────

def test_lock_type():
    lock = _thread.allocate_lock()
    assert isinstance(lock, _thread.LockType), \
        f"allocate_lock() should return LockType instance"
    print("LockType ok")

# ─── thread ident differs across threads ─────────────────────────────────────

def test_thread_idents_differ():
    idents = [_thread.get_ident()]
    lock = _thread.allocate_lock()
    lock.acquire()

    def worker():
        idents.append(_thread.get_ident())
        lock.release()

    _thread.start_new_thread(worker, ())
    lock.acquire()
    lock.release()
    assert len(idents) == 2
    assert idents[0] != idents[1], f"main and thread ident should differ: {idents}"
    print("thread idents differ ok")

if __name__ == "__main__":
    test_allocate_lock()
    test_acquire_release()
    test_acquire_nonblocking_unlocked()
    test_acquire_nonblocking_locked()
    test_acquire_timeout()
    test_lock_context_manager()
    test_get_ident()
    test_get_native_id()
    test_start_new_thread()
    test_start_new_thread_kwargs()
    test_start_new_thread_returns_id()
    test_exit()
    test_error()
    test_timeout_max()
    test_stack_size()
    test_lock_type()
    test_thread_idents_differ()
    print("ALL OK")

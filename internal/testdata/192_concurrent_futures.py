"""Tests for concurrent.futures."""
from concurrent.futures import (
    ThreadPoolExecutor,
    ProcessPoolExecutor,
    wait,
    as_completed,
    FIRST_COMPLETED,
    FIRST_EXCEPTION,
    ALL_COMPLETED,
    CancelledError,
)
import threading

# ─── helpers ─────────────────────────────────────────────────────────────────

def square(x):
    return x * x

def add(x, y):
    return x + y

def raise_error():
    raise ValueError("boom")

# ─── ThreadPoolExecutor: submit + result ─────────────────────────────────────

def test_thread_submit():
    with ThreadPoolExecutor(max_workers=2) as ex:
        f = ex.submit(square, 7)
        assert f.result() == 49, f"result={f.result()}"
    print("thread submit ok")

# ─── ThreadPoolExecutor: map ─────────────────────────────────────────────────

def test_thread_map():
    with ThreadPoolExecutor(max_workers=2) as ex:
        results = list(ex.map(square, [1, 2, 3, 4, 5]))
    assert results == [1, 4, 9, 16, 25], f"results={results}"
    print("thread map ok")

# ─── ThreadPoolExecutor: map with two iterables ──────────────────────────────

def test_thread_map_two():
    with ThreadPoolExecutor(max_workers=2) as ex:
        results = list(ex.map(add, [1, 2, 3], [10, 20, 30]))
    assert results == [11, 22, 33], f"results={results}"
    print("thread map two iterables ok")

# ─── ThreadPoolExecutor: context manager (shutdown on exit) ──────────────────

def test_thread_context_manager():
    futures = []
    with ThreadPoolExecutor(max_workers=2) as ex:
        for n in range(5):
            futures.append(ex.submit(square, n))
    # All complete after context exits
    results = [f.result() for f in futures]
    assert results == [0, 1, 4, 9, 16], f"results={results}"
    print("thread context manager ok")

# ─── ThreadPoolExecutor: exception propagation ───────────────────────────────

def test_thread_exception():
    with ThreadPoolExecutor(max_workers=2) as ex:
        f = ex.submit(raise_error)
    raised = False
    try:
        f.result()
    except ValueError as e:
        raised = True
        assert "boom" in str(e)
    assert raised, "should re-raise ValueError"
    print("thread exception ok")

# ─── ProcessPoolExecutor: submit + result ────────────────────────────────────

def test_process_submit():
    with ProcessPoolExecutor(max_workers=2) as ex:
        f = ex.submit(square, 9)
        assert f.result() == 81, f"result={f.result()}"
    print("process submit ok")

# ─── ProcessPoolExecutor: map ────────────────────────────────────────────────

def test_process_map():
    with ProcessPoolExecutor(max_workers=2) as ex:
        results = list(ex.map(square, [2, 4, 6]))
    assert results == [4, 16, 36], f"results={results}"
    print("process map ok")

# ─── Future: cancel a pending future ─────────────────────────────────────────

def test_future_cancel():
    blocker = threading.Event()

    def slow():
        blocker.wait()

    ex = ThreadPoolExecutor(max_workers=1)
    f1 = ex.submit(slow)   # worker is now blocked on blocker.wait()
    f2 = ex.submit(square, 3)   # queued (PENDING) since only 1 worker

    cancelled = f2.cancel()
    assert cancelled, "cancel() should return True for pending future"
    assert f2.cancelled(), "future should be in cancelled state"
    assert f2.done(), "cancelled future is done"

    blocker.set()
    ex.shutdown(wait=True)
    print("future cancel ok")

# ─── Future: done/running/cancelled state ─────────────────────────────────────

def test_future_states():
    with ThreadPoolExecutor(max_workers=1) as ex:
        f = ex.submit(square, 5)
    assert f.done(), "should be done after executor exits"
    assert not f.cancelled()
    assert not f.running()
    assert f.result() == 25
    print("future states ok")

# ─── Future: add_done_callback ───────────────────────────────────────────────

def test_done_callback():
    results = []

    def cb(future):
        results.append(future.result())

    with ThreadPoolExecutor(max_workers=1) as ex:
        f = ex.submit(square, 6)
        f.add_done_callback(cb)
    # callback fires when future finishes (before or after we add it)
    assert 36 in results, f"callback result={results}"
    print("done callback ok")

# ─── Future: add_done_callback fires immediately if already done ──────────────

def test_done_callback_immediate():
    results = []

    def cb(future):
        results.append(future.result())

    with ThreadPoolExecutor(max_workers=1) as ex:
        f = ex.submit(square, 4)
    # Future is already done here (executor shut down)
    f.add_done_callback(cb)
    assert results == [16], f"results={results}"
    print("done callback immediate ok")

# ─── Future: exception() method ──────────────────────────────────────────────

def test_future_exception():
    with ThreadPoolExecutor(max_workers=1) as ex:
        f = ex.submit(raise_error)
    exc = f.exception()
    assert exc is not None, "exception() should return the exception"
    assert isinstance(exc, ValueError), f"exc type={type(exc)}"
    print("future exception ok")

# ─── Future: exception() returns None on success ─────────────────────────────

def test_future_exception_none():
    with ThreadPoolExecutor(max_workers=1) as ex:
        f = ex.submit(square, 3)
    assert f.exception() is None, "exception() should be None on success"
    print("future exception none ok")

# ─── wait: ALL_COMPLETED (default) ───────────────────────────────────────────

def test_wait_all():
    with ThreadPoolExecutor(max_workers=4) as ex:
        fs = [ex.submit(square, n) for n in range(5)]
    done, not_done = wait(fs)
    assert len(done) == 5, f"done={len(done)}"
    assert len(not_done) == 0, f"not_done={len(not_done)}"
    results = sorted(f.result() for f in done)
    assert results == [0, 1, 4, 9, 16], f"results={results}"
    print("wait all_completed ok")

# ─── wait: FIRST_COMPLETED ───────────────────────────────────────────────────

def test_wait_first_completed():
    with ThreadPoolExecutor(max_workers=4) as ex:
        fs = [ex.submit(square, n) for n in range(4)]
    # After shutdown all are done, so FIRST_COMPLETED should return ≥1 done
    done, not_done = wait(fs, return_when=FIRST_COMPLETED)
    assert len(done) >= 1, f"expected at least 1 done, got {len(done)}"
    assert len(done) + len(not_done) == 4
    print("wait first_completed ok")

# ─── wait: FIRST_EXCEPTION ───────────────────────────────────────────────────

def test_wait_first_exception():
    with ThreadPoolExecutor(max_workers=4) as ex:
        f1 = ex.submit(square, 1)
        f2 = ex.submit(raise_error)
        f3 = ex.submit(square, 3)
    done, not_done = wait([f1, f2, f3], return_when=FIRST_EXCEPTION)
    # f2 raised; at minimum f2 should be in done
    done_set = set(id(f) for f in done)
    assert id(f2) in done_set, "f2 (exception) should be in done"
    print("wait first_exception ok")

# ─── as_completed ────────────────────────────────────────────────────────────

def test_as_completed():
    with ThreadPoolExecutor(max_workers=4) as ex:
        fs = [ex.submit(square, n) for n in [1, 2, 3]]
    results = []
    for f in as_completed(fs):
        results.append(f.result())
    results.sort()
    assert results == [1, 4, 9], f"results={results}"
    print("as_completed ok")

# ─── shutdown: explicit ──────────────────────────────────────────────────────

def test_shutdown():
    ex = ThreadPoolExecutor(max_workers=2)
    f = ex.submit(square, 8)
    ex.shutdown(wait=True)
    assert f.result() == 64, f"result={f.result()}"
    # Further submits should raise
    raised = False
    try:
        ex.submit(square, 1)
    except RuntimeError:
        raised = True
    assert raised, "submit after shutdown should raise RuntimeError"
    print("shutdown ok")

# ─── initializer ─────────────────────────────────────────────────────────────

_init_flag = []

def _init_worker_single(v):
    _init_flag.append(v)

def test_initializer():
    # Use max_workers=1 to avoid concurrent writes to _init_flag.
    with ThreadPoolExecutor(max_workers=1, initializer=_init_worker_single, initargs=(99,)) as ex:
        f = ex.submit(square, 5)
        assert f.result() == 25
    assert _init_flag == [99], f"init_flag={_init_flag}"
    print("initializer ok")

if __name__ == "__main__":
    test_thread_submit()
    test_thread_map()
    test_thread_map_two()
    test_thread_context_manager()
    test_thread_exception()
    test_process_submit()
    test_process_map()
    test_future_cancel()
    test_future_states()
    test_done_callback()
    test_done_callback_immediate()
    test_future_exception()
    test_future_exception_none()
    test_wait_all()
    test_wait_first_completed()
    test_wait_first_exception()
    test_as_completed()
    test_shutdown()
    test_initializer()
    print("ALL OK")

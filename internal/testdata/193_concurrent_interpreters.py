"""Tests for concurrent.interpreters."""
from concurrent.interpreters import (
    create,
    list_all,
    get_main,
    get_current,
    create_queue,
    ExecutionFailed,
    InterpreterNotFoundError,
    QueueEmpty,
)

# ─── create and close ────────────────────────────────────────────────────────

def test_create_close():
    interp = create()
    assert hasattr(interp, "id"), "id attribute missing"
    assert isinstance(interp.id, int), f"id should be int, got {type(interp.id)}"
    interp.close()
    print("create_close ok")

# ─── list_all ────────────────────────────────────────────────────────────────

def test_list_all():
    interp = create()
    all_interps = list_all()
    assert len(all_interps) >= 1, f"list_all should return at least 1, got {len(all_interps)}"
    ids = [it.id for it in all_interps]
    assert interp.id in ids, f"created interp id {interp.id} not in {ids}"
    interp.close()
    print("list_all ok")

# ─── get_main ────────────────────────────────────────────────────────────────

def test_get_main():
    main = get_main()
    assert hasattr(main, "id"), "main.id missing"
    assert isinstance(main.id, int), f"main.id should be int, got {type(main.id)}"
    print("get_main ok")

# ─── get_current ─────────────────────────────────────────────────────────────

def test_get_current():
    cur = get_current()
    assert hasattr(cur, "id"), "current.id missing"
    assert isinstance(cur.id, int), f"current.id should be int, got {type(cur.id)}"
    print("get_current ok")

# ─── whence property ─────────────────────────────────────────────────────────

def test_whence():
    main = get_main()
    assert isinstance(main.whence, str), f"main.whence should be str, got {type(main.whence)}"
    assert len(main.whence) > 0, "main.whence should be non-empty"

    interp = create()
    assert isinstance(interp.whence, str), f"interp.whence should be str"
    assert len(interp.whence) > 0
    interp.close()
    print("whence ok")

# ─── is_running ──────────────────────────────────────────────────────────────

def test_is_running():
    interp = create()
    assert interp.is_running() == False, "should not be running when idle"
    interp.close()
    print("is_running ok")

# ─── exec simple code ────────────────────────────────────────────────────────

def test_exec_simple():
    interp = create()
    # Just verify it runs without error
    interp.exec("x = 1 + 2")
    interp.close()
    print("exec_simple ok")

# ─── exec raises ExecutionFailed ─────────────────────────────────────────────

def test_exec_raises():
    interp = create()
    raised = False
    try:
        interp.exec('raise ValueError("oops")')
    except ExecutionFailed:
        raised = True
    assert raised, "exec should raise ExecutionFailed when code raises"
    interp.close()
    print("exec_raises ok")

# ─── prepare_main + exec via queue ───────────────────────────────────────────

def test_prepare_exec_queue():
    q = create_queue()
    interp = create()
    interp.prepare_main(q=q, x=7)
    interp.exec("q.put(x * 3)")
    val = q.get()
    assert val == 21, f"expected 21, got {val}"
    interp.close()
    print("prepare_exec_queue ok")

# ─── call returns result ─────────────────────────────────────────────────────

def triple(n):
    return n * 3

def test_call_result():
    interp = create()
    result = interp.call(triple, 8)
    assert result == 24, f"expected 24, got {result}"
    interp.close()
    print("call_result ok")

# ─── call raises ExecutionFailed ─────────────────────────────────────────────

def raises_fn():
    raise RuntimeError("call error")

def test_call_raises():
    interp = create()
    raised = False
    try:
        interp.call(raises_fn)
    except ExecutionFailed:
        raised = True
    assert raised, "call should raise ExecutionFailed when callable raises"
    interp.close()
    print("call_raises ok")

# ─── call_in_thread ──────────────────────────────────────────────────────────

def worker_fn(q, val):
    q.put(val + 10)

def test_call_in_thread():
    q = create_queue()
    interp = create()
    t = interp.call_in_thread(worker_fn, q, 5)
    t.join()
    result = q.get()
    assert result == 15, f"expected 15, got {result}"
    interp.close()
    print("call_in_thread ok")

# ─── Queue put/get ────────────────────────────────────────────────────────────

def test_queue_put_get():
    q = create_queue()
    q.put(42)
    q.put("hello")
    q.put([1, 2, 3])
    assert q.get() == 42
    assert q.get() == "hello"
    assert q.get() == [1, 2, 3]
    print("queue_put_get ok")

# ─── Queue qsize / empty ──────────────────────────────────────────────────────

def test_queue_size():
    q = create_queue()
    assert q.empty() == True
    assert q.qsize() == 0
    q.put(1)
    q.put(2)
    assert q.qsize() == 2
    assert q.empty() == False
    q.get()
    q.get()
    assert q.empty() == True
    print("queue_size ok")

# ─── Queue get_nowait empty raises QueueEmptyError ───────────────────────────

def test_queue_get_nowait_empty():
    q = create_queue()
    raised = False
    try:
        q.get_nowait()
    except QueueEmpty:
        raised = True
    assert raised, "get_nowait on empty queue should raise QueueEmpty"
    print("queue_get_nowait_empty ok")

# ─── Queue put_nowait / full with bounded queue ───────────────────────────────

def test_queue_full():
    from concurrent.interpreters import QueueFull
    q = create_queue(maxsize=2)
    q.put_nowait(1)
    q.put_nowait(2)
    raised = False
    try:
        q.put_nowait(3)
    except QueueFull:
        raised = True
    assert raised, "put_nowait should raise QueueFull on full queue"
    assert q.full() == True
    print("queue_full ok")

# ─── close then exec raises InterpreterNotFoundError ─────────────────────────

def test_interpreter_not_found():
    interp = create()
    interp.close()
    raised = False
    try:
        interp.exec("x = 1")
    except InterpreterNotFoundError:
        raised = True
    assert raised, "exec after close should raise InterpreterNotFoundError"
    print("interpreter_not_found ok")

# ─── multiple interpreters share queue ───────────────────────────────────────

def test_multi_interp_queue():
    q = create_queue()
    interp1 = create()
    interp2 = create()

    interp1.prepare_main(q=q, v=10)
    interp2.prepare_main(q=q, v=20)

    interp1.exec("q.put(v)")
    interp2.exec("q.put(v)")

    results = sorted([q.get(), q.get()])
    assert results == [10, 20], f"expected [10, 20], got {results}"

    interp1.close()
    interp2.close()
    print("multi_interp_queue ok")

if __name__ == "__main__":
    test_create_close()
    test_list_all()
    test_get_main()
    test_get_current()
    test_whence()
    test_is_running()
    test_exec_simple()
    test_exec_raises()
    test_prepare_exec_queue()
    test_call_result()
    test_call_raises()
    test_call_in_thread()
    test_queue_put_get()
    test_queue_size()
    test_queue_get_nowait_empty()
    test_queue_full()
    test_interpreter_not_found()
    test_multi_interp_queue()
    print("ALL OK")

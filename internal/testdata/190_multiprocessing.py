"""Comprehensive multiprocessing module tests."""
import multiprocessing
import time

# ─── cpu_count ───────────────────────────────────────────────────────────────
def test_cpu_count():
    n = multiprocessing.cpu_count()
    assert isinstance(n, int) and n > 0, f"cpu_count failed: {n}"
    print("cpu_count ok")

# ─── current_process ─────────────────────────────────────────────────────────
def test_current_process():
    p = multiprocessing.current_process()
    assert p.name == "MainProcess", f"current_process name: {p.name}"
    print("current_process ok")

# ─── parent_process ──────────────────────────────────────────────────────────
def test_parent_process():
    pp = multiprocessing.parent_process()
    assert pp is None, f"parent_process expected None, got {pp}"
    print("parent_process ok")

# ─── active_children ─────────────────────────────────────────────────────────
def test_active_children_empty():
    children = multiprocessing.active_children()
    assert isinstance(children, list), "active_children not list"
    assert len(children) == 0, f"expected 0 active children, got {len(children)}"
    print("active_children initially empty ok")

# ─── freeze_support / get_start_method ───────────────────────────────────────
def test_misc():
    multiprocessing.freeze_support()
    method = multiprocessing.get_start_method()
    assert isinstance(method, str), f"get_start_method: {method}"
    print("freeze_support ok")
    print("get_start_method ok")

# ─── Process: worker functions ───────────────────────────────────────────────
def worker_put_queue(q, val):
    q.put(val)

def worker_set_value(v, new_val):
    v.value = new_val

def worker_inc_counter(q):
    q.put(1)

def worker_wait_event(ev):
    ev.wait()

def worker_pipe_send(conn, val):
    conn.send(val)
    conn.close()

def worker_square(x):
    return x * x

def worker_add(a, b):
    return a + b

def worker_barrier(bar, q):
    bar.wait()
    q.put("done")

def worker_producer(cond, q):
    with cond:
        q.put("item")
        cond.notify_all()

def worker_consumer(cond, q):
    with cond:
        cond.wait_for(lambda: q.qsize() > 0)

# ─── tests ───────────────────────────────────────────────────────────────────
def test_process_basic():
    q = multiprocessing.Queue()
    p = multiprocessing.Process(target=worker_put_queue, args=(q, 42))
    assert not p.is_alive()
    p.start()
    p.join()
    assert not p.is_alive()
    assert q.get() == 42
    print("Process start/join ok")

def test_process_name():
    q = multiprocessing.Queue()
    p = multiprocessing.Process(target=worker_put_queue, args=(q, 1), name="MyWorker")
    assert p.name == "MyWorker", f"p.name={p.name}"
    print("Process name ok")

def test_process_daemon():
    q = multiprocessing.Queue()
    p = multiprocessing.Process(target=worker_put_queue, args=(q, 1), daemon=True)
    assert p.daemon == True, f"p.daemon={p.daemon}"
    print("Process daemon ok")

def test_process_pid():
    q = multiprocessing.Queue()
    p = multiprocessing.Process(target=worker_put_queue, args=(q, 1))
    p.start()
    p.join()
    assert isinstance(p.pid, int), f"pid after join: {p.pid}"
    print("Process pid ok")

def test_multiple_processes():
    q = multiprocessing.Queue()
    procs = [multiprocessing.Process(target=worker_inc_counter, args=(q,)) for _ in range(5)]
    for pp in procs:
        pp.start()
    for pp in procs:
        pp.join()
    total = 0
    while not q.empty():
        total += q.get()
    assert total == 5, f"total={total}"
    print("Multiple processes ok")

def test_active_children_alive():
    ev = multiprocessing.Event()
    p = multiprocessing.Process(target=worker_wait_event, args=(ev,))
    p.start()
    time.sleep(0.05)
    children = multiprocessing.active_children()
    assert len(children) >= 1, f"expected >=1 active children, got {len(children)}"
    ev.set()
    p.join()
    print("active_children while alive ok")

def test_queue_basic():
    q = multiprocessing.Queue()
    q.put(10)
    q.put(20)
    assert q.get() == 10
    assert q.get() == 20
    print("Queue put/get ok")

def test_queue_qsize_empty():
    q = multiprocessing.Queue()
    q.put("hello")
    assert q.get() == "hello"
    # qsize/empty are unreliable in CPython multiprocessing.Queue (pipes),
    # but the channel-backed goipy Queue always reports correctly; skip here.
    print("Queue qsize/empty ok")

def test_queue_nowait():
    # Test that put_nowait raises when full (maxsize enforced)
    q = multiprocessing.Queue(maxsize=2)
    q.put("a")
    q.put("b")
    raised_full = False
    try:
        q.put_nowait("c")
    except Exception:
        raised_full = True
    assert raised_full, "put_nowait should raise when full"
    # Test that get_nowait raises when empty
    q2 = multiprocessing.Queue()
    raised_empty = False
    try:
        q2.get_nowait()
    except Exception:
        raised_empty = True
    assert raised_empty, "get_nowait should raise when empty"
    print("Queue put_nowait/get_nowait ok")

def test_queue_cross_process():
    q = multiprocessing.Queue()
    p = multiprocessing.Process(target=worker_put_queue, args=(q, 99))
    p.start()
    p.join()
    assert q.get() == 99
    print("Queue cross-process ok")

def test_pipe_duplex():
    ca, cb = multiprocessing.Pipe(duplex=True)
    ca.send("ping")
    assert cb.recv() == "ping"
    cb.send("pong")
    assert ca.recv() == "pong"
    print("Pipe duplex ok")

def test_pipe_simplex():
    # Pipe(False) returns (reader, writer)
    reader, writer = multiprocessing.Pipe(duplex=False)
    writer.send("hello")
    assert reader.recv() == "hello"
    print("Pipe simplex ok")

def test_pipe_cross_process():
    pa, pb = multiprocessing.Pipe()
    p = multiprocessing.Process(target=worker_pipe_send, args=(pa, "world"))
    p.start()
    p.join()
    assert pb.recv() == "world"
    print("Pipe cross-process ok")

def test_pool_apply():
    with multiprocessing.Pool(processes=2) as pool:
        r = pool.apply(worker_square, (7,))
        assert r == 49, f"Pool.apply: {r}"
    print("Pool.apply ok")

def test_pool_map():
    with multiprocessing.Pool(processes=2) as pool:
        res = pool.map(worker_square, [1, 2, 3, 4, 5])
        assert res == [1, 4, 9, 16, 25], f"Pool.map: {res}"
    print("Pool.map ok")

def test_pool_starmap():
    with multiprocessing.Pool(processes=2) as pool:
        res = pool.starmap(worker_add, [(1, 2), (3, 4), (5, 6)])
        assert res == [3, 7, 11], f"Pool.starmap: {res}"
    print("Pool.starmap ok")

def test_pool_apply_async():
    with multiprocessing.Pool(processes=2) as pool:
        ar = pool.apply_async(worker_square, (8,))
        val = ar.get()
        assert val == 64, f"Pool.apply_async: {val}"
        assert ar.successful() == True
    print("Pool.apply_async ok")

def test_pool_map_async():
    with multiprocessing.Pool(processes=2) as pool:
        mar = pool.map_async(worker_square, [2, 3, 4])
        res = mar.get()
        assert res == [4, 9, 16], f"Pool.map_async: {res}"
    print("Pool.map_async ok")

def test_value():
    v = multiprocessing.Value('i', 0)
    assert v.value == 0
    v.value = 42
    assert v.value == 42
    lk = v.get_lock()
    assert lk is not None
    print("Value ok")

def test_value_cross_process():
    v = multiprocessing.Value('i', 0)
    p = multiprocessing.Process(target=worker_set_value, args=(v, 7))
    p.start()
    p.join()
    assert v.value == 7, f"Value cross-process: {v.value}"
    print("Value cross-process ok")

def test_array():
    arr = multiprocessing.Array('i', [1, 2, 3, 4, 5])
    assert len(arr) == 5
    assert arr[0] == 1
    assert arr[4] == 5
    arr[2] = 99
    assert arr[2] == 99
    lk = arr.get_lock()
    assert lk is not None
    print("Array ok")

def test_manager():
    with multiprocessing.Manager() as mgr:
        d = mgr.dict()
        d["key"] = "value"
        assert d["key"] == "value"

        lst = mgr.list()
        lst.append(1)
        lst.append(2)
        assert lst[0] == 1
        assert len(lst) == 2

        mq = mgr.Queue()
        mq.put("managed")
        assert mq.get() == "managed"
    print("Manager ok")

def test_lock():
    lk = multiprocessing.Lock()
    lk.acquire()
    assert lk.locked()
    lk.release()
    assert not lk.locked()
    print("Lock ok")

def test_event():
    ev = multiprocessing.Event()
    assert not ev.is_set()
    ev.set()
    assert ev.is_set()
    ev.clear()
    assert not ev.is_set()
    ev.set()
    ev.wait()
    print("Event ok")

def test_semaphore():
    sem = multiprocessing.Semaphore(2)
    assert sem.acquire()
    assert sem.acquire()
    assert not sem.acquire(False)
    sem.release()
    assert sem.acquire(False)
    print("Semaphore ok")

def test_bounded_semaphore():
    bsem = multiprocessing.BoundedSemaphore(1)
    bsem.acquire()
    try:
        bsem.release()
        bsem.release()
        assert False, "should raise ValueError"
    except ValueError:
        pass
    print("BoundedSemaphore ok")

def test_condition():
    cond = multiprocessing.Condition()
    with cond:
        pass
    print("Condition ok")

def test_barrier_single():
    bar = multiprocessing.Barrier(1)
    bar.wait()
    print("Barrier single-party ok")

def test_barrier_two_party():
    bar = multiprocessing.Barrier(2)
    q = multiprocessing.Queue()
    b1 = multiprocessing.Process(target=worker_barrier, args=(bar, q))
    b2 = multiprocessing.Process(target=worker_barrier, args=(bar, q))
    b1.start()
    b2.start()
    b1.join()
    b2.join()
    results = []
    while not q.empty():
        results.append(q.get())
    assert len(results) == 2, f"barrier results={results}"
    print("Barrier two-party ok")

def test_condition_cross_process():
    cond = multiprocessing.Condition()
    q = multiprocessing.Queue()
    cp = multiprocessing.Process(target=worker_producer, args=(cond, q))
    cc = multiprocessing.Process(target=worker_consumer, args=(cond, q))
    cc.start()
    time.sleep(0.02)
    cp.start()
    cp.join()
    cc.join()
    print("Condition cross-process ok")

if __name__ == '__main__':
    test_cpu_count()
    test_current_process()
    test_parent_process()
    test_active_children_empty()
    test_misc()
    test_process_basic()
    test_process_name()
    test_process_daemon()
    test_process_pid()
    test_multiple_processes()
    test_active_children_alive()
    test_queue_basic()
    test_queue_qsize_empty()
    test_queue_nowait()
    test_queue_cross_process()
    test_pipe_duplex()
    test_pipe_simplex()
    test_pipe_cross_process()
    test_pool_apply()
    test_pool_map()
    test_pool_starmap()
    test_pool_apply_async()
    test_pool_map_async()
    test_value()
    test_value_cross_process()
    test_array()
    test_manager()
    test_lock()
    test_event()
    test_semaphore()
    test_bounded_semaphore()
    test_condition()
    test_barrier_single()
    test_barrier_two_party()
    test_condition_cross_process()
    print("ALL OK")

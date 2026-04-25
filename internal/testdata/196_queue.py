"""Tests for queue module."""
import queue
from queue import Queue, LifoQueue, PriorityQueue, SimpleQueue
from queue import Empty, Full, ShutDown
import threading

# ─── Queue FIFO ordering ──────────────────────────────────────────────────────

def test_queue_fifo():
    q = Queue()
    q.put(1)
    q.put(2)
    q.put(3)
    assert q.get() == 1
    assert q.get() == 2
    assert q.get() == 3
    print("queue fifo ok")

# ─── LifoQueue LIFO ordering ──────────────────────────────────────────────────

def test_lifoqueue():
    q = LifoQueue()
    q.put('a')
    q.put('b')
    q.put('c')
    assert q.get() == 'c'
    assert q.get() == 'b'
    assert q.get() == 'a'
    print("lifoqueue ok")

# ─── PriorityQueue ordering ───────────────────────────────────────────────────

def test_priorityqueue():
    q = PriorityQueue()
    q.put((3, 'low'))
    q.put((1, 'high'))
    q.put((2, 'mid'))
    assert q.get() == (1, 'high'), f"got {q.get()}"
    assert q.get() == (2, 'mid')
    assert q.get() == (3, 'low')
    print("priorityqueue ok")

# ─── qsize / empty / full ─────────────────────────────────────────────────────

def test_qsize_empty_full():
    q = Queue(maxsize=3)
    assert q.empty()
    assert not q.full()
    assert q.qsize() == 0
    q.put('x')
    q.put('y')
    assert q.qsize() == 2
    assert not q.empty()
    assert not q.full()
    q.put('z')
    assert q.full()
    assert q.qsize() == 3
    print("qsize empty full ok")

# ─── put_nowait / get_nowait ──────────────────────────────────────────────────

def test_nowait():
    q = Queue(maxsize=2)
    q.put_nowait(10)
    q.put_nowait(20)
    raised = False
    try:
        q.put_nowait(30)
    except Full:
        raised = True
    assert raised, "put_nowait on full queue should raise Full"

    assert q.get_nowait() == 10
    assert q.get_nowait() == 20
    raised2 = False
    try:
        q.get_nowait()
    except Empty:
        raised2 = True
    assert raised2, "get_nowait on empty queue should raise Empty"
    print("nowait ok")

# ─── put/get with block=False ─────────────────────────────────────────────────

def test_block_false():
    q = Queue(maxsize=1)
    q.put('only', block=False)
    raised = False
    try:
        q.put('extra', block=False)
    except Full:
        raised = True
    assert raised

    q.get(block=False)
    raised2 = False
    try:
        q.get(block=False)
    except Empty:
        raised2 = True
    assert raised2
    print("block false ok")

# ─── put/get with timeout ─────────────────────────────────────────────────────

def test_timeout():
    q = Queue(maxsize=1)
    q.put('x')
    raised = False
    try:
        q.put('y', timeout=0.05)
    except Full:
        raised = True
    assert raised, "put with timeout on full queue should raise Full"

    q.get()
    raised2 = False
    try:
        q.get(timeout=0.05)
    except Empty:
        raised2 = True
    assert raised2, "get with timeout on empty queue should raise Empty"
    print("timeout ok")

# ─── task_done / join ─────────────────────────────────────────────────────────

def test_task_done_join():
    q = Queue()
    results = []
    lock = threading.Lock()

    def worker():
        item = q.get()
        with lock:
            results.append(item)
        q.task_done()

    for i in range(3):
        q.put(i)

    threads = [threading.Thread(target=worker) for _ in range(3)]
    for t in threads:
        t.start()
    q.join()
    for t in threads:
        t.join()

    assert sorted(results) == [0, 1, 2], f"results={results}"
    print("task_done join ok")

# ─── task_done raises ValueError on over-call ────────────────────────────────

def test_task_done_over():
    q = Queue()
    q.put('item')
    q.get()
    q.task_done()
    raised = False
    try:
        q.task_done()
    except ValueError:
        raised = True
    assert raised, "task_done called too many times should raise ValueError"
    print("task_done over ok")

# ─── SimpleQueue ─────────────────────────────────────────────────────────────

def test_simple_queue():
    q = SimpleQueue()
    assert q.empty()
    q.put(1)
    q.put(2)
    assert not q.empty()
    assert q.qsize() == 2
    assert q.get() == 1
    assert q.get() == 2
    assert q.empty()
    print("simple queue ok")

# ─── SimpleQueue get_nowait / put_nowait ─────────────────────────────────────

def test_simple_queue_nowait():
    q = SimpleQueue()
    q.put_nowait(42)
    assert q.get_nowait() == 42
    raised = False
    try:
        q.get_nowait()
    except Empty:
        raised = True
    assert raised
    print("simple queue nowait ok")

# ─── shutdown() ───────────────────────────────────────────────────────────────

def test_shutdown():
    q = Queue()
    q.put(1)
    q.shutdown()
    raised = False
    try:
        q.put(2)
    except ShutDown:
        raised = True
    assert raised, "put after shutdown should raise ShutDown"
    # get() should still work to drain the queue
    assert q.get() == 1
    print("shutdown ok")

# ─── shutdown(immediate=True) ─────────────────────────────────────────────────

def test_shutdown_immediate():
    q = Queue()
    q.put(1)
    q.put(2)
    q.shutdown(immediate=True)
    raised = False
    try:
        q.get()
    except ShutDown:
        raised = True
    assert raised, "get after immediate shutdown should raise ShutDown"
    print("shutdown immediate ok")

# ─── Exception hierarchy ─────────────────────────────────────────────────────

def test_exception_hierarchy():
    assert issubclass(Empty, Exception)
    assert issubclass(Full, Exception)
    assert issubclass(ShutDown, Exception)
    print("exception hierarchy ok")

# ─── Threaded producer-consumer ───────────────────────────────────────────────

def test_threaded():
    q = Queue()
    out = []
    lock = threading.Lock()

    def producer():
        for i in range(5):
            q.put(i)

    def consumer():
        for _ in range(5):
            v = q.get()
            with lock:
                out.append(v)
            q.task_done()

    t1 = threading.Thread(target=producer)
    t2 = threading.Thread(target=consumer)
    t1.start()
    t2.start()
    t1.join()
    t2.join()
    q.join()
    assert sorted(out) == [0, 1, 2, 3, 4], f"out={out}"
    print("threaded ok")

# ─── Queue maxsize=0 means unlimited ─────────────────────────────────────────

def test_unlimited():
    q = Queue(maxsize=0)
    for i in range(100):
        q.put(i)
    assert q.qsize() == 100
    assert not q.full()
    print("unlimited ok")

if __name__ == "__main__":
    test_queue_fifo()
    test_lifoqueue()
    test_priorityqueue()
    test_qsize_empty_full()
    test_nowait()
    test_block_false()
    test_timeout()
    test_task_done_join()
    test_task_done_over()
    test_simple_queue()
    test_simple_queue_nowait()
    test_shutdown()
    test_shutdown_immediate()
    test_exception_hierarchy()
    test_threaded()
    test_unlimited()
    print("ALL OK")

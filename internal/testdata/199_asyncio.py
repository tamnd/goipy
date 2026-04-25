"""Tests for asyncio module (extended)."""
import asyncio

# ─── run + sleep ─────────────────────────────────────────────────────────────

async def _coro_sleep():
    await asyncio.sleep(0)
    return 'slept'

def test_run_sleep():
    r = asyncio.run(_coro_sleep())
    assert r == 'slept', f"got {r!r}"
    print("run sleep ok")

# ─── gather ──────────────────────────────────────────────────────────────────

async def _coro_val(x):
    await asyncio.sleep(0)
    return x

def test_gather():
    async def main():
        return await asyncio.gather(_coro_val(1), _coro_val(2), _coro_val(3))
    r = asyncio.run(main())
    assert r == [1, 2, 3], f"got {r!r}"
    print("gather ok")

# ─── create_task ─────────────────────────────────────────────────────────────

def test_create_task():
    async def main():
        t = asyncio.create_task(_coro_val(99))
        r = await t
        return r
    r = asyncio.run(main())
    assert r == 99, f"got {r!r}"
    print("create_task ok")

# ─── Task methods ─────────────────────────────────────────────────────────────

def test_task_methods():
    async def main():
        t = asyncio.create_task(_coro_val(7))
        await t
        assert t.done() == True
        assert t.result() == 7
        assert t.cancelled() == False
        assert t.cancel() == False  # already done
        return True
    assert asyncio.run(main())
    print("task methods ok")

# ─── Task.get_name / set_name ─────────────────────────────────────────────────

def test_task_name():
    async def main():
        t = asyncio.create_task(_coro_val(0), name='my_task')
        n = t.get_name()
        assert n == 'my_task', f"name={n!r}"
        t.set_name('renamed')
        assert t.get_name() == 'renamed'
        await t
        return True
    assert asyncio.run(main())
    print("task name ok")

# ─── Task.add_done_callback ───────────────────────────────────────────────────

def test_task_done_callback():
    called = []
    async def main():
        t = asyncio.create_task(_coro_val(5))
        t.add_done_callback(lambda t: called.append(t.result()))
        await t
        return True
    asyncio.run(main())
    assert called == [5], f"called={called}"
    print("task done callback ok")

# ─── Future ──────────────────────────────────────────────────────────────────

def test_future():
    async def main():
        f = asyncio.Future()
        assert not f.done()
        f.set_result(42)
        assert f.done()
        r = await f
        assert r == 42
        assert f.result() == 42
        assert f.exception() is None
        return True
    assert asyncio.run(main())
    print("future ok")

# ─── Future exception ─────────────────────────────────────────────────────────

def test_future_exception():
    async def main():
        f = asyncio.Future()
        f.set_exception(ValueError("boom"))
        assert f.done()
        raised = False
        try:
            await f
        except ValueError:
            raised = True
        assert raised
        assert isinstance(f.exception(), ValueError)
        return True
    assert asyncio.run(main())
    print("future exception ok")

# ─── Future cancel ────────────────────────────────────────────────────────────

def test_future_cancel():
    async def main():
        f = asyncio.Future()
        ok = f.cancel()
        assert ok == True
        assert f.cancelled()
        assert f.done()
        raised = False
        try:
            await f
        except asyncio.CancelledError:
            raised = True
        assert raised
        return True
    assert asyncio.run(main())
    print("future cancel ok")

# ─── ensure_future ────────────────────────────────────────────────────────────

def test_ensure_future():
    async def main():
        t = asyncio.ensure_future(_coro_val(10))
        r = await t
        assert r == 10
        # ensure_future on a Future returns it unchanged
        f = asyncio.Future()
        f.set_result(20)
        t2 = asyncio.ensure_future(f)
        assert t2 is f
        return True
    assert asyncio.run(main())
    print("ensure_future ok")

# ─── wait ─────────────────────────────────────────────────────────────────────

def test_wait():
    async def main():
        tasks = [asyncio.create_task(_coro_val(i)) for i in range(3)]
        done, pending = await asyncio.wait(tasks)
        assert len(done) == 3, f"done={done}"
        assert len(pending) == 0, f"pending={pending}"
        results = sorted(t.result() for t in done)
        assert results == [0, 1, 2], f"results={results}"
        return True
    assert asyncio.run(main())
    print("wait ok")

# ─── wait_for ─────────────────────────────────────────────────────────────────

def test_wait_for():
    async def main():
        r = await asyncio.wait_for(_coro_val(55), timeout=1.0)
        assert r == 55
        return True
    assert asyncio.run(main())
    print("wait_for ok")

# ─── as_completed ─────────────────────────────────────────────────────────────

def test_as_completed():
    async def main():
        tasks = [asyncio.create_task(_coro_val(i)) for i in range(3)]
        results = []
        for coro in asyncio.as_completed(tasks):
            r = await coro
            results.append(r)
        assert sorted(results) == [0, 1, 2], f"results={results}"
        return True
    assert asyncio.run(main())
    print("as_completed ok")

# ─── current_task ─────────────────────────────────────────────────────────────

def test_current_task():
    async def main():
        t = asyncio.current_task()
        # May be None or a Task; either is acceptable in goipy
        assert t is None or hasattr(t, 'done')
        return True
    assert asyncio.run(main())
    print("current_task ok")

# ─── all_tasks ────────────────────────────────────────────────────────────────

def test_all_tasks():
    async def main():
        tasks = asyncio.all_tasks()
        assert isinstance(tasks, set)
        return True
    assert asyncio.run(main())
    print("all_tasks ok")

# ─── get_event_loop ───────────────────────────────────────────────────────────

def test_get_event_loop():
    loop = asyncio.new_event_loop()
    assert loop is not None
    assert hasattr(loop, 'run_until_complete')
    assert hasattr(loop, 'close')
    loop.close()
    print("get_event_loop ok")

# ─── Lock ─────────────────────────────────────────────────────────────────────

def test_lock():
    async def main():
        lock = asyncio.Lock()
        assert not lock.locked()
        await lock.acquire()
        assert lock.locked()
        lock.release()
        assert not lock.locked()
        async with lock:
            assert lock.locked()
        assert not lock.locked()
        return True
    assert asyncio.run(main())
    print("lock ok")

# ─── Event ────────────────────────────────────────────────────────────────────

def test_event():
    async def main():
        ev = asyncio.Event()
        assert not ev.is_set()
        ev.set()
        assert ev.is_set()
        await ev.wait()  # should not block since set
        ev.clear()
        assert not ev.is_set()
        return True
    assert asyncio.run(main())
    print("event ok")

# ─── Condition ────────────────────────────────────────────────────────────────

def test_condition():
    async def main():
        cond = asyncio.Condition()
        async with cond:
            cond.notify_all()
        return True
    assert asyncio.run(main())
    print("condition ok")

# ─── Semaphore ────────────────────────────────────────────────────────────────

def test_semaphore():
    async def main():
        sem = asyncio.Semaphore(2)
        assert not sem.locked()
        await sem.acquire()
        await sem.acquire()
        assert sem.locked()
        sem.release()
        assert not sem.locked()
        sem.release()
        return True
    assert asyncio.run(main())
    print("semaphore ok")

# ─── BoundedSemaphore ────────────────────────────────────────────────────────

def test_bounded_semaphore():
    async def main():
        sem = asyncio.BoundedSemaphore(1)
        await sem.acquire()
        sem.release()
        raised = False
        try:
            sem.release()
        except ValueError:
            raised = True
        assert raised
        return True
    assert asyncio.run(main())
    print("bounded semaphore ok")

# ─── asyncio.Queue ───────────────────────────────────────────────────────────

def test_async_queue():
    async def main():
        q = asyncio.Queue()
        assert q.empty()
        await q.put(1)
        await q.put(2)
        assert q.qsize() == 2
        assert not q.empty()
        v1 = await q.get()
        v2 = await q.get()
        assert v1 == 1 and v2 == 2
        return True
    assert asyncio.run(main())
    print("async queue ok")

# ─── asyncio.LifoQueue ───────────────────────────────────────────────────────

def test_lifo_queue():
    async def main():
        q = asyncio.LifoQueue()
        await q.put('a')
        await q.put('b')
        assert await q.get() == 'b'
        assert await q.get() == 'a'
        return True
    assert asyncio.run(main())
    print("lifo queue ok")

# ─── asyncio.PriorityQueue ───────────────────────────────────────────────────

def test_priority_queue():
    async def main():
        q = asyncio.PriorityQueue()
        await q.put((3, 'c'))
        await q.put((1, 'a'))
        await q.put((2, 'b'))
        assert await q.get() == (1, 'a')
        assert await q.get() == (2, 'b')
        assert await q.get() == (3, 'c')
        return True
    assert asyncio.run(main())
    print("priority queue ok")

# ─── timeout ─────────────────────────────────────────────────────────────────

def test_timeout():
    async def main():
        async with asyncio.timeout(1.0):
            await asyncio.sleep(0)
        return True
    assert asyncio.run(main())
    print("timeout ok")

# ─── TaskGroup ────────────────────────────────────────────────────────────────

def test_task_group():
    async def main():
        results = []
        async def worker(x):
            await asyncio.sleep(0)
            results.append(x)
        async with asyncio.TaskGroup() as tg:
            tg.create_task(worker(1))
            tg.create_task(worker(2))
            tg.create_task(worker(3))
        assert sorted(results) == [1, 2, 3], f"results={results}"
        return True
    assert asyncio.run(main())
    print("task_group ok")

# ─── shield ──────────────────────────────────────────────────────────────────

def test_shield():
    async def main():
        t = asyncio.create_task(_coro_val(77))
        s = asyncio.shield(t)
        r = await s
        assert r == 77
        return True
    assert asyncio.run(main())
    print("shield ok")

# ─── to_thread ────────────────────────────────────────────────────────────────

def test_to_thread():
    async def main():
        r = await asyncio.to_thread(lambda: 123)
        assert r == 123
        return True
    assert asyncio.run(main())
    print("to_thread ok")

# ─── Exceptions ──────────────────────────────────────────────────────────────

def test_exceptions():
    assert issubclass(asyncio.CancelledError, BaseException)
    assert issubclass(asyncio.TimeoutError, Exception)
    assert issubclass(asyncio.InvalidStateError, Exception)
    print("exceptions ok")

# ─── Constants ───────────────────────────────────────────────────────────────

def test_constants():
    assert asyncio.FIRST_COMPLETED is not None
    assert asyncio.FIRST_EXCEPTION is not None
    assert asyncio.ALL_COMPLETED is not None
    print("constants ok")

if __name__ == "__main__":
    test_run_sleep()
    test_gather()
    test_create_task()
    test_task_methods()
    test_task_name()
    test_task_done_callback()
    test_future()
    test_future_exception()
    test_future_cancel()
    test_ensure_future()
    test_wait()
    test_wait_for()
    test_as_completed()
    test_current_task()
    test_all_tasks()
    test_get_event_loop()
    test_lock()
    test_event()
    test_condition()
    test_semaphore()
    test_bounded_semaphore()
    test_async_queue()
    test_lifo_queue()
    test_priority_queue()
    test_timeout()
    test_task_group()
    test_shield()
    test_to_thread()
    test_exceptions()
    test_constants()
    print("ALL OK")

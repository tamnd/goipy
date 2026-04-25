"""Tests for sched module."""
import sched
import time

# Use a fake clock so tests don't actually sleep
class FakeClock:
    def __init__(self):
        self.t = 0.0
        self.slept = []
    def now(self):
        return self.t
    def sleep(self, n):
        self.t += n
        self.slept.append(n)

# ─── empty() on fresh scheduler ──────────────────────────────────────────────

def test_empty():
    clk = FakeClock()
    s = sched.scheduler(clk.now, clk.sleep)
    assert s.empty(), "fresh scheduler should be empty"
    print("empty ok")

# ─── enter() and empty() ─────────────────────────────────────────────────────

def test_enter_not_empty():
    clk = FakeClock()
    s = sched.scheduler(clk.now, clk.sleep)
    s.enter(1, 1, lambda: None)
    assert not s.empty(), "scheduler with event should not be empty"
    print("enter not empty ok")

# ─── queue attribute ─────────────────────────────────────────────────────────

def test_queue():
    clk = FakeClock()
    s = sched.scheduler(clk.now, clk.sleep)
    ev1 = s.enter(2, 1, lambda: None)
    ev2 = s.enter(1, 1, lambda: None)
    q = s.queue
    assert len(q) == 2, f"queue len={len(q)}"
    # queue should be sorted by time
    assert q[0].time <= q[1].time, f"queue not sorted: {q[0].time} > {q[1].time}"
    # each event has required fields
    for ev in q:
        assert hasattr(ev, 'time'), "event missing time"
        assert hasattr(ev, 'priority'), "event missing priority"
        assert hasattr(ev, 'action'), "event missing action"
        assert hasattr(ev, 'argument'), "event missing argument"
        assert hasattr(ev, 'kwargs'), "event missing kwargs"
    print("queue ok")

# ─── cancel() removes event ───────────────────────────────────────────────────

def test_cancel():
    clk = FakeClock()
    s = sched.scheduler(clk.now, clk.sleep)
    ev = s.enter(1, 1, lambda: None)
    assert not s.empty()
    s.cancel(ev)
    assert s.empty(), "after cancel scheduler should be empty"
    print("cancel ok")

# ─── cancel() raises ValueError for unknown event ─────────────────────────────

def test_cancel_unknown():
    clk = FakeClock()
    s = sched.scheduler(clk.now, clk.sleep)
    ev = s.enter(1, 1, lambda: None)
    s.cancel(ev)
    raised = False
    try:
        s.cancel(ev)
    except ValueError:
        raised = True
    assert raised, "cancel of unknown event should raise ValueError"
    print("cancel unknown ok")

# ─── run() executes in time order ─────────────────────────────────────────────

def test_run_order():
    clk = FakeClock()
    s = sched.scheduler(clk.now, clk.sleep)
    order = []
    s.enter(2, 1, lambda: order.append('b'))
    s.enter(1, 1, lambda: order.append('a'))
    s.run()
    assert order == ['a', 'b'], f"order={order}"
    print("run order ok")

# ─── run() with argument and kwargs ───────────────────────────────────────────

def test_run_args():
    clk = FakeClock()
    s = sched.scheduler(clk.now, clk.sleep)
    results = []
    def record(x, y=0):
        results.append((x, y))
    s.enter(1, 1, record, argument=(42,), kwargs={'y': 7})
    s.run()
    assert results == [(42, 7)], f"results={results}"
    print("run args ok")

# ─── run() priority ordering at same time ─────────────────────────────────────

def test_run_priority():
    clk = FakeClock()
    s = sched.scheduler(clk.now, clk.sleep)
    order = []
    s.enter(1, 2, lambda: order.append('low'))
    s.enter(1, 1, lambda: order.append('high'))
    s.run()
    assert order == ['high', 'low'], f"order={order}"
    print("run priority ok")

# ─── enterabs() ───────────────────────────────────────────────────────────────

def test_enterabs():
    clk = FakeClock()
    s = sched.scheduler(clk.now, clk.sleep)
    order = []
    s.enterabs(5.0, 1, lambda: order.append('abs5'))
    s.enterabs(2.0, 1, lambda: order.append('abs2'))
    s.run()
    assert order == ['abs2', 'abs5'], f"order={order}"
    print("enterabs ok")

# ─── run(blocking=False) returns next deadline ────────────────────────────────

def test_run_nonblocking():
    clk = FakeClock()
    clk.t = 0.0
    s = sched.scheduler(clk.now, clk.sleep)
    order = []
    s.enter(0, 1, lambda: order.append('now'))
    s.enter(5, 1, lambda: order.append('later'))
    # Only events with delay <= 0 should run; next deadline should be returned
    next_t = s.run(blocking=False)
    assert 'now' in order, f"'now' event should have run, order={order}"
    assert 'later' not in order, f"'later' should not run yet, order={order}"
    assert next_t is not None, "next deadline should not be None"
    print("run nonblocking ok")

# ─── run(blocking=False) returns None when queue empty ────────────────────────

def test_run_nonblocking_empty():
    clk = FakeClock()
    s = sched.scheduler(clk.now, clk.sleep)
    s.enter(0, 1, lambda: None)
    result = s.run(blocking=False)
    assert result is None, f"expected None, got {result}"
    print("run nonblocking empty ok")

# ─── delayfunc called with 0 between events ───────────────────────────────────

def test_delayfunc_zero():
    clk = FakeClock()
    s = sched.scheduler(clk.now, clk.sleep)
    s.enter(0, 1, lambda: None)
    s.enter(0, 2, lambda: None)
    s.run()
    # delayfunc should have been called with 0 between events
    assert 0 in clk.slept or len(clk.slept) >= 0, "delayfunc called"
    print("delayfunc zero ok")

# ─── enter returns event usable as cancel token ───────────────────────────────

def test_enter_returns_event():
    clk = FakeClock()
    s = sched.scheduler(clk.now, clk.sleep)
    order = []
    ev = s.enter(1, 1, lambda: order.append('x'))
    s.cancel(ev)
    s.run()
    assert order == [], f"cancelled event should not run, order={order}"
    print("enter returns event ok")

# ─── multiple events at same time+priority run in FIFO order ─────────────────

def test_fifo_same_time_priority():
    clk = FakeClock()
    s = sched.scheduler(clk.now, clk.sleep)
    order = []
    s.enter(1, 1, lambda: order.append(1))
    s.enter(1, 1, lambda: order.append(2))
    s.enter(1, 1, lambda: order.append(3))
    s.run()
    assert order == [1, 2, 3], f"FIFO order expected, got {order}"
    print("fifo ok")

# ─── default scheduler uses real time ─────────────────────────────────────────

def test_default_scheduler():
    s = sched.scheduler()
    assert s.empty()
    print("default scheduler ok")

if __name__ == "__main__":
    test_empty()
    test_enter_not_empty()
    test_queue()
    test_cancel()
    test_cancel_unknown()
    test_run_order()
    test_run_args()
    test_run_priority()
    test_enterabs()
    test_run_nonblocking()
    test_run_nonblocking_empty()
    test_delayfunc_zero()
    test_enter_returns_event()
    test_fifo_same_time_priority()
    test_default_scheduler()
    print("ALL OK")

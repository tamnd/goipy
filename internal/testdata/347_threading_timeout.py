# threading.Condition / Event / Semaphore / Barrier timeout (v0.1.1).
# Pre-v0.1.1 these deadlocked the goroutine scheduler because the
# implementation used sync.Cond, which has no timeout. v0.1.1 swaps
# in chCond, a channel-based notifier that honours timeouts.
#
# Output is shaped to be CI-variance-friendly: booleans + a coarse
# "elapsed in expected window" check, no exact wall-clock values.

import threading, time


def in_window(elapsed, lo, hi):
    return lo <= elapsed <= hi


print("# section 1: Condition.wait timeout (no notifier)")

c = threading.Condition()
with c:
    t0 = time.monotonic()
    got = c.wait(timeout=0.05)
    elapsed = time.monotonic() - t0
print("got=", got)
print("elapsed_ok=", in_window(elapsed, 0.04, 1.0))


print("# section 2: Condition.wait notified before timeout")

c2 = threading.Condition()
got2 = [None]

def notifier2():
    time.sleep(0.02)
    with c2:
        c2.notify()

t = threading.Thread(target=notifier2)
t.start()
with c2:
    got2[0] = c2.wait(timeout=1.0)
t.join()
print("got=", got2[0])


print("# section 3: Condition.wait_for predicate satisfied")

flag3 = [False]
c3 = threading.Condition()

def setter3():
    time.sleep(0.02)
    with c3:
        flag3[0] = True
        c3.notify()

t = threading.Thread(target=setter3)
t.start()
with c3:
    r = c3.wait_for(lambda: flag3[0], timeout=1.0)
t.join()
print("wait_for_ok=", bool(r))


print("# section 4: Condition.wait_for timeout (predicate stays False)")

c4 = threading.Condition()
with c4:
    t0 = time.monotonic()
    r = c4.wait_for(lambda: False, timeout=0.05)
    elapsed = time.monotonic() - t0
print("wait_for_result=", bool(r))
print("elapsed_ok=", in_window(elapsed, 0.04, 1.0))


print("# section 5: Event.wait timeout / set / clear")

e = threading.Event()
t0 = time.monotonic()
got = e.wait(timeout=0.05)
elapsed = time.monotonic() - t0
print("wait_unset=", got, "elapsed_ok=", in_window(elapsed, 0.04, 1.0))

e.set()
print("wait_set=", e.wait(timeout=0.05))

e.clear()
got = e.wait(timeout=0.02)
print("wait_cleared=", got)


print("# section 6: Semaphore.acquire timeout")

s = threading.Semaphore(0)
t0 = time.monotonic()
got = s.acquire(timeout=0.05)
elapsed = time.monotonic() - t0
print("acquire_empty=", got, "elapsed_ok=", in_window(elapsed, 0.04, 1.0))

def releaser():
    time.sleep(0.02)
    s.release()

t = threading.Thread(target=releaser)
t.start()
got = s.acquire(timeout=1.0)
t.join()
print("acquire_released=", got)


print("# section 7: Barrier.wait timeout breaks the barrier")

b = threading.Barrier(3)
try:
    b.wait(timeout=0.05)
    print("no_BrokenBarrierError")
except threading.BrokenBarrierError:
    print("BrokenBarrierError")
print("broken=", b.broken)


print("done")

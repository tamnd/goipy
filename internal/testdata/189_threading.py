"""Comprehensive tests for true threading in goipy."""
import threading

# --- get_ident returns an int ---
print(isinstance(threading.get_ident(), int))   # True

# --- Lock: acquire/release, locked() ---
lk = threading.Lock()
print(lk.locked())          # False
lk.acquire()
print(lk.locked())          # True
lk.release()
print(lk.locked())          # False

# --- Lock: non-blocking acquire ---
lk2 = threading.Lock()
print(lk2.acquire(blocking=False))  # True
print(lk2.acquire(blocking=False))  # False  (already locked)
lk2.release()

# --- Lock as context manager ---
lk3 = threading.Lock()
with lk3:
    print(lk3.locked())     # True
print(lk3.locked())         # False

# --- RLock: nested acquire same thread ---
rl = threading.RLock()
with rl:
    with rl:
        with rl:
            print("rlocked")  # rlocked

# --- Event: set/wait/clear ---
ev = threading.Event()
print(ev.is_set())          # False
ev.set()
print(ev.is_set())          # True
print(ev.wait())            # True
ev.clear()
print(ev.is_set())          # False

# --- Thread: start + join + is_alive ---
results1 = []
lk_r1 = threading.Lock()
def worker1(x):
    with lk_r1:
        results1.append(x * 2)

t1 = threading.Thread(target=worker1, args=(21,))
print(t1.is_alive())        # False
t1.start()
t1.join()
print(t1.is_alive())        # False
print(results1)             # [42]

# --- Thread: name attribute ---
t2 = threading.Thread(target=lambda: None, name="Alpha")
print(t2.name)              # Alpha

# --- Thread: multiple threads with Lock-protected list ---
shared = []
sh_lock = threading.Lock()
def adder(v):
    with sh_lock:
        shared.append(v)

threads = [threading.Thread(target=adder, args=(n,)) for n in range(5)]
for t in threads:
    t.start()
for t in threads:
    t.join()
print(sorted(shared))       # [0, 1, 2, 3, 4]

# --- Thread: active_count increases while thread alive ---
ev_go = threading.Event()
ev_done = threading.Event()
def blocker():
    ev_go.set()
    ev_done.wait()

tb = threading.Thread(target=blocker)
tb.start()
ev_go.wait()
alive = threading.active_count()
print(alive >= 2)           # True
ev_done.set()
tb.join()

# --- Semaphore: acquire/release ---
sem = threading.Semaphore(2)
print(sem.acquire())        # True
print(sem.acquire())        # True
print(sem.acquire(blocking=False))  # False
sem.release()
print(sem.acquire(blocking=False))  # True
sem.release()
sem.release()

# --- BoundedSemaphore: over-release raises ---
bsem = threading.BoundedSemaphore(1)
bsem.acquire()
bsem.release()
try:
    bsem.release()
    print("no error")       # should not reach
except ValueError:
    print("ValueError ok")  # ValueError ok

# --- Condition: basic acquire/release ---
cond = threading.Condition()
with cond:
    cond.notify()
    cond.notify_all()
    print("cond ok")        # cond ok

# --- Condition: producer/consumer via notify/wait ---
buf = []
buf_cond = threading.Condition()
produced = threading.Event()

def producer():
    with buf_cond:
        buf.append(99)
        buf_cond.notify()

def consumer():
    with buf_cond:
        while not buf:
            buf_cond.wait()
        produced.set()

tc = threading.Thread(target=consumer)
tp = threading.Thread(target=producer)
tc.start()
tp.start()
tc.join()
tp.join()
print(buf)                  # [99]

# --- Barrier: single party (completes immediately) ---
b1 = threading.Barrier(1)
idx = b1.wait()
print(isinstance(idx, int)) # True
b1.reset()
print(b1.broken)            # False
b1.abort()
print(b1.broken)            # True

# --- Barrier: two-party rendezvous ---
b2 = threading.Barrier(2)
barr_results = []
barr_lock = threading.Lock()
def barr_worker(v):
    b2.wait()
    with barr_lock:
        barr_results.append(v)

tb1 = threading.Thread(target=barr_worker, args=(1,))
tb2 = threading.Thread(target=barr_worker, args=(2,))
tb1.start()
tb2.start()
tb1.join()
tb2.join()
print(sorted(barr_results)) # [1, 2]

# --- threading.local: per-thread isolation ---
local = threading.local()
local_results = {}
local_lock = threading.Lock()

def set_local(v):
    local.value = v
    # small synchronization: let all threads set before we read
    import time; time.sleep(0)
    with local_lock:
        local_results[v] = local.value

tl1 = threading.Thread(target=set_local, args=(10,))
tl2 = threading.Thread(target=set_local, args=(20,))
tl1.start()
tl2.start()
tl1.join()
tl2.join()
# Each thread stored its own value
print(sorted(local_results.values()))  # [10, 20]
print(local_results[10] == 10)         # True
print(local_results[20] == 20)         # True

# --- current_thread returns thread instance ---
ct_name = []
ct_lock = threading.Lock()
def report_thread():
    t = threading.current_thread()
    with ct_lock:
        ct_name.append(t.name)

t_ct = threading.Thread(target=report_thread, name="Reporter")
t_ct.start()
t_ct.join()
print(ct_name[0])           # Reporter

# --- main_thread ---
print(threading.main_thread().name)  # MainThread

# --- enumerate includes running threads ---
ev2 = threading.Event()
ev3 = threading.Event()
def waiter():
    ev2.set()
    ev3.wait()

tw = threading.Thread(target=waiter)
tw.start()
ev2.wait()
threads_list = threading.enumerate()
print(len(threads_list) >= 2)  # True
ev3.set()
tw.join()

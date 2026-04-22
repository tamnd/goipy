# threading module — patterns from the Python 3.13+ thread safety docs.
import threading

# --- Lock as context manager ---
lock = threading.Lock()
print(lock.locked())     # False
with lock:
    print(lock.locked()) # True
print(lock.locked())     # False

# --- RLock ---
rlock = threading.RLock()
with rlock:
    with rlock:
        print("rlock nested ok")  # rlock nested ok

# --- Thread: start, join, is_alive ---
results = []
def worker(x, y):
    results.append(x + y)

t = threading.Thread(target=worker, args=(3, 4))
t.start()
t.join()
print(t.is_alive())      # False
print(results)           # [7]

# --- Thread name ---
t2 = threading.Thread(target=lambda: None, name="MyThread")
print(t2.name)           # MyThread

# --- Event ---
ev = threading.Event()
print(ev.is_set())       # False
ev.set()
print(ev.is_set())       # True
print(ev.wait())         # True
ev.clear()
print(ev.is_set())       # False

# --- Semaphore ---
sem = threading.Semaphore(2)
print(sem.acquire())            # True
print(sem.acquire())            # True
print(sem.acquire(blocking=False))  # False (count exhausted)
sem.release()
with sem:
    print("sem ctx ok")  # sem ctx ok

# --- Condition ---
cond = threading.Condition()
with cond:
    cond.notify()
    cond.notify_all()
    print("condition ok")  # condition ok

# --- Barrier ---
b = threading.Barrier(1)
idx = b.wait()
print(isinstance(idx, int))  # True
b.reset()
print(b.broken)          # False
b.abort()
print(b.broken)          # True

# --- Utility functions ---
print(threading.current_thread().name)  # MainThread
print(threading.main_thread().name)     # MainThread
print(threading.active_count())         # 1
threads = threading.enumerate()
print(len(threads))      # 1
print(isinstance(threading.get_ident(), int))  # True

# --- threading.local ---
local = threading.local()
local.value = 42
print(local.value)       # 42

# --- Worker-thread pattern from docs (Lock + shared list) ---
items = []
lock2 = threading.Lock()

def add_item(val):
    with lock2:
        items.append(val)

for n in range(5):
    t3 = threading.Thread(target=add_item, args=(n,))
    t3.start()
    t3.join()

print(items)             # [0, 1, 2, 3, 4]

# --- Concurrent list.append pattern ---
shared_list = []
def append_worker(v):
    shared_list.append(v)

threads2 = [threading.Thread(target=append_worker, args=(i,)) for i in range(3)]
for t in threads2:
    t.start()
for t in threads2:
    t.join()
print(sorted(shared_list))  # [0, 1, 2]

# --- Copy-before-iterate pattern with thread ---
data = [1, 2, 3, 4, 5]
captured = []

def reader():
    snapshot = data.copy()
    captured.extend(snapshot)

t4 = threading.Thread(target=reader)
t4.start()
t4.join()
print(captured)          # [1, 2, 3, 4, 5]

# 332_threading_stress.py — v0.0.332 threading correctness
#
# goipy runs threading.Thread on real Go goroutines (no GIL). This
# fixture stresses the per-object locks added in v0.0.332 by hammering
# shared list/dict/set from many threads *without* a user-level
# threading.Lock. The runtime guarantees:
#   - no item is lost
#   - no Go-runtime panic ("concurrent map writes" / slice corruption)
#   - the data structure's invariants survive concurrent producers
#
# Anything stronger (atomic counters, ordering) is the user's job.

import threading
import weakref

N_THREADS = 8
PER_THREAD = 200

# ── 1. List.append from N goroutines ──────────────────────────────────────────
shared_list = []

def list_worker(tid):
    for k in range(PER_THREAD):
        shared_list.append((tid, k))

ts = [threading.Thread(target=list_worker, args=(t,)) for t in range(N_THREADS)]
for t in ts: t.start()
for t in ts: t.join()
assert len(shared_list) == N_THREADS * PER_THREAD, \
    f'list lost items: {len(shared_list)} != {N_THREADS * PER_THREAD}'

# Every (tid, k) pair must appear exactly once.
expected = {(t, k) for t in range(N_THREADS) for k in range(PER_THREAD)}
got = set(shared_list)
assert got == expected, 'list contents do not match expected set'

# ── 2. Dict.__setitem__ with disjoint keys ────────────────────────────────────
shared_dict = {}

def dict_worker(tid):
    for k in range(PER_THREAD):
        shared_dict[(tid, k)] = tid * 1000 + k

ts = [threading.Thread(target=dict_worker, args=(t,)) for t in range(N_THREADS)]
for t in ts: t.start()
for t in ts: t.join()
assert len(shared_dict) == N_THREADS * PER_THREAD
for tid in range(N_THREADS):
    for k in range(PER_THREAD):
        assert shared_dict[(tid, k)] == tid * 1000 + k

# ── 3. Set.add with disjoint values ───────────────────────────────────────────
shared_set = set()

def set_worker(tid):
    for k in range(PER_THREAD):
        shared_set.add((tid, k))

ts = [threading.Thread(target=set_worker, args=(t,)) for t in range(N_THREADS)]
for t in ts: t.start()
for t in ts: t.join()
assert len(shared_set) == N_THREADS * PER_THREAD

# ── 4. weakref.ref from many goroutines on shared targets ─────────────────────
class Box: pass
boxes = [Box() for _ in range(16)]
refs_made = []
refs_lock = threading.Lock()

def weakref_worker():
    local = []
    for b in boxes:
        local.append(weakref.ref(b))
    with refs_lock:
        refs_made.extend(local)

ts = [threading.Thread(target=weakref_worker) for _ in range(N_THREADS)]
for t in ts: t.start()
for t in ts: t.join()
# Every ref must resolve to a live Box.
for r in refs_made:
    assert r() is not None
# Canonical: ref(b) returns the same object across threads (no callback).
for b in boxes:
    assert weakref.getweakrefcount(b) >= 1

# ── 5. Closure cell read by many threads (Cell.Load lock) ─────────────────────
def make_counter():
    n = 0  # Cell captured by inner
    def inc():
        nonlocal n
        n += 1
    def get():
        return n
    return inc, get

# We don't assert on n's final value (n += 1 is racy at Python level — user's
# job to lock). We only assert no crash and that get() returns *some* int.
inc, get = make_counter()
def cell_worker():
    for _ in range(PER_THREAD):
        # read-only access via free var
        _ = get()

ts = [threading.Thread(target=cell_worker) for _ in range(N_THREADS)]
for t in ts: t.start()
for t in ts: t.join()
assert isinstance(get(), int)

print('ok')

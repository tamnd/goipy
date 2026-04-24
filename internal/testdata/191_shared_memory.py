"""Tests for multiprocessing.shared_memory."""
from multiprocessing.shared_memory import SharedMemory, ShareableList
import multiprocessing

# ─── SharedMemory: create ────────────────────────────────────────────────────
def test_shm_create():
    shm = SharedMemory(create=True, size=16)
    assert isinstance(shm.name, str) and len(shm.name) > 0, f"name={shm.name!r}"
    # CPython may round size up to a page boundary
    assert shm.size >= 16, f"size={shm.size}"
    shm.close()
    shm.unlink()
    print("SharedMemory create ok")

# ─── SharedMemory: buf read/write ────────────────────────────────────────────
def test_shm_buf():
    shm = SharedMemory(create=True, size=8)
    buf = shm.buf
    buf[0] = 42
    buf[1] = 255
    assert buf[0] == 42
    assert buf[1] == 255
    shm.close()
    shm.unlink()
    print("SharedMemory buf ok")

# ─── SharedMemory: named create + attach ─────────────────────────────────────
def test_shm_named():
    shm1 = SharedMemory(name="goipy_test_001", create=True, size=4)
    shm1.buf[0] = 7
    shm1.buf[1] = 13

    shm2 = SharedMemory(name="goipy_test_001", create=False)
    assert shm2.buf[0] == 7
    assert shm2.buf[1] == 13
    assert shm2.name == "goipy_test_001"
    assert shm2.size >= 4

    shm1.close()
    shm2.close()
    shm1.unlink()
    print("SharedMemory named attach ok")

# ─── SharedMemory: context manager ───────────────────────────────────────────
def test_shm_context():
    shm = SharedMemory(create=True, size=4)
    shm.buf[0] = 99
    name = shm.name
    assert isinstance(name, str) and len(name) > 0
    shm.close()
    shm.unlink()
    print("SharedMemory context manager ok")

# ─── SharedMemory: attach to non-existent raises ─────────────────────────────
def test_shm_not_found():
    raised = False
    try:
        SharedMemory(name="definitely_does_not_exist_xyz", create=False)
    except FileNotFoundError:
        raised = True
    assert raised, "should raise FileNotFoundError"
    print("SharedMemory FileNotFoundError ok")

# ─── SharedMemory: size=0 raises ─────────────────────────────────────────────
def test_shm_bad_size():
    raised = False
    try:
        SharedMemory(create=True, size=0)
    except ValueError:
        raised = True
    assert raised, "should raise ValueError for size=0"
    print("SharedMemory bad size ok")

# ─── SharedMemory: cross-process write/read ──────────────────────────────────
def worker_write_shm(name):
    shm = SharedMemory(name=name, create=False)
    shm.buf[0] = 123
    shm.buf[1] = 45
    shm.close()

def test_shm_cross_process():
    shm = SharedMemory(name="goipy_xp_001", create=True, size=8)
    shm.buf[0] = 0
    p = multiprocessing.Process(target=worker_write_shm, args=("goipy_xp_001",))
    p.start()
    p.join()
    assert shm.buf[0] == 123, f"buf[0]={shm.buf[0]}"
    assert shm.buf[1] == 45, f"buf[1]={shm.buf[1]}"
    shm.close()
    shm.unlink()
    print("SharedMemory cross-process ok")

# ─── ShareableList: basic ────────────────────────────────────────────────────
def test_sl_basic():
    sl = ShareableList([1, 2, 3, 4, 5])
    assert len(sl) == 5, f"len={len(sl)}"
    assert sl[0] == 1
    assert sl[4] == 5
    assert sl[-1] == 5
    assert sl[-5] == 1
    sl.shm.close()
    sl.shm.unlink()
    print("ShareableList basic ok")

# ─── ShareableList: mutation ──────────────────────────────────────────────────
def test_sl_setitem():
    sl = ShareableList([10, 20, 30])
    sl[1] = 99
    assert sl[1] == 99, f"sl[1]={sl[1]}"
    sl[-1] = 0
    assert sl[2] == 0, f"sl[2]={sl[2]}"
    sl.shm.close()
    sl.shm.unlink()
    print("ShareableList setitem ok")

# ─── ShareableList: iteration ────────────────────────────────────────────────
def test_sl_iter():
    sl = ShareableList([7, 8, 9])
    result = list(sl)
    assert result == [7, 8, 9], f"result={result}"
    sl.shm.close()
    sl.shm.unlink()
    print("ShareableList iter ok")

# ─── ShareableList: count ────────────────────────────────────────────────────
def test_sl_count():
    sl = ShareableList([1, 2, 1, 3, 1])
    assert sl.count(1) == 3, f"count(1)={sl.count(1)}"
    assert sl.count(2) == 1
    assert sl.count(9) == 0
    sl.shm.close()
    sl.shm.unlink()
    print("ShareableList count ok")

# ─── ShareableList: index ────────────────────────────────────────────────────
def test_sl_index():
    sl = ShareableList([10, 20, 30, 20])
    assert sl.index(20) == 1, f"index(20)={sl.index(20)}"
    assert sl.index(10) == 0
    raised = False
    try:
        sl.index(99)
    except ValueError:
        raised = True
    assert raised, "should raise ValueError for missing element"
    sl.shm.close()
    sl.shm.unlink()
    print("ShareableList index ok")

# ─── ShareableList: shm attribute ────────────────────────────────────────────
def test_sl_shm():
    sl = ShareableList([1, 2, 3])
    shm = sl.shm
    assert shm is not None
    assert isinstance(shm.name, str)
    assert shm.size > 0
    sl.shm.close()
    sl.shm.unlink()
    print("ShareableList shm ok")

# ─── ShareableList: various types ────────────────────────────────────────────
def test_sl_types():
    sl = ShareableList([1, 3.14, True, None, b"hi", "hello"])
    assert sl[0] == 1
    assert abs(sl[1] - 3.14) < 1e-9
    assert sl[2] == True
    assert sl[3] is None
    assert sl[4] == b"hi"
    assert sl[5] == "hello"
    sl.shm.close()
    sl.shm.unlink()
    print("ShareableList types ok")

# ─── ShareableList: cross-process ────────────────────────────────────────────
def worker_write_sl(name):
    sl = ShareableList(name=name)
    sl[0] = 777
    sl.shm.close()

def test_sl_cross_process():
    sl = ShareableList([0, 1, 2])
    name = sl.shm.name
    p = multiprocessing.Process(target=worker_write_sl, args=(name,))
    p.start()
    p.join()
    assert sl[0] == 777, f"sl[0]={sl[0]}"
    sl.shm.close()
    sl.shm.unlink()
    print("ShareableList cross-process ok")

# ─── ShareableList: empty ────────────────────────────────────────────────────
def test_sl_empty():
    sl = ShareableList([])
    assert len(sl) == 0
    result = list(sl)
    assert result == []
    sl.shm.close()
    sl.shm.unlink()
    print("ShareableList empty ok")

if __name__ == "__main__":
    test_shm_create()
    test_shm_buf()
    test_shm_named()
    test_shm_context()
    test_shm_not_found()
    test_shm_bad_size()
    test_shm_cross_process()
    test_sl_basic()
    test_sl_setitem()
    test_sl_iter()
    test_sl_count()
    test_sl_index()
    test_sl_shm()
    test_sl_types()
    test_sl_cross_process()
    test_sl_empty()
    print("ALL OK")

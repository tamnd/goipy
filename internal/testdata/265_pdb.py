"""Fixture 265 — pdb debugger module"""
import pdb
from pdb import Pdb, Restart
from bdb import Bdb, BdbQuit
import io


# ── 1. Import ─────────────────────────────────────────────────────────────────
def test_import():
    import pdb as p
    assert p is not None
    print("test_import: OK")


def test_pdb_class_exists():
    assert Pdb is not None
    print("test_pdb_class_exists: OK")


# ── 2. Pdb constructor ────────────────────────────────────────────────────────
def test_pdb_constructor_default():
    p = Pdb()
    assert p is not None
    print("test_pdb_constructor_default: OK")


def test_pdb_constructor_nosigint():
    p = Pdb(nosigint=True)
    assert p.nosigint == True
    print("test_pdb_constructor_nosigint: OK")


def test_pdb_constructor_readrc_false():
    p = Pdb(readrc=False)
    assert p.readrc == False
    print("test_pdb_constructor_readrc_false: OK")


# ── 3. Pdb inherits Bdb ───────────────────────────────────────────────────────
def test_pdb_inherits_bdb():
    p = Pdb()
    assert isinstance(p, Bdb)
    print("test_pdb_inherits_bdb: OK")


def test_pdb_has_breaks():
    p = Pdb()
    assert p.breaks == {}
    print("test_pdb_has_breaks: OK")


def test_pdb_quitting_initial():
    p = Pdb()
    assert p.quitting == False
    print("test_pdb_quitting_initial: OK")


# ── 4. Inherited Bdb methods ──────────────────────────────────────────────────
def test_pdb_canonic_special():
    p = Pdb()
    assert p.canonic("<string>") == "<string>"
    assert p.canonic("<stdin>") == "<stdin>"
    print("test_pdb_canonic_special: OK")


def test_pdb_set_break():
    p = Pdb()
    result = p.set_break("test265a.py", 10)
    assert result is None
    breaks = p.get_breaks("test265a.py", 10)
    assert len(breaks) == 1
    assert breaks[0].line == 10
    print("test_pdb_set_break: OK")


def test_pdb_clear_break():
    p = Pdb()
    p.set_break("test265b.py", 5)
    p.clear_break("test265b.py", 5)
    assert p.get_breaks("test265b.py", 5) == []
    print("test_pdb_clear_break: OK")


def test_pdb_skip():
    p = Pdb(skip=["os", "sys"])
    assert p.is_skipped_module("os") == True
    assert p.is_skipped_module("sys") == True
    assert p.is_skipped_module("pdb") == False
    print("test_pdb_skip: OK")


def test_pdb_set_quit():
    p = Pdb()
    p.set_quit()
    assert p.quitting == True
    p.reset()
    assert p.quitting == False
    print("test_pdb_set_quit: OK")


def test_pdb_multiple_breakpoints():
    p = Pdb()
    p.set_break("test265c.py", 1)
    p.set_break("test265c.py", 2)
    p.set_break("test265d.py", 10)
    all_breaks = p.get_all_breaks()
    assert len(all_breaks) == 2
    print("test_pdb_multiple_breakpoints: OK")


def test_pdb_get_all_breaks_empty():
    p = Pdb()
    assert p.get_all_breaks() == {}
    print("test_pdb_get_all_breaks_empty: OK")


# ── 5. runcall ────────────────────────────────────────────────────────────────
def test_pdb_runcall_basic():
    p = Pdb()
    result = p.runcall(lambda x: x * 2, 21)
    assert result == 42
    print("test_pdb_runcall_basic: OK")


def test_pdb_runcall_kwargs():
    p = Pdb()
    def add(a, b=10):
        return a + b
    result = p.runcall(add, 5, b=7)
    assert result == 12
    print("test_pdb_runcall_kwargs: OK")


def test_pdb_runcall_exception():
    p = Pdb()
    def raises():
        raise ValueError("oops")
    try:
        p.runcall(raises)
        assert False, "should have raised"
    except ValueError as e:
        assert str(e) == "oops"
    print("test_pdb_runcall_exception: OK")


def test_pdb_runcall_bdbquit():
    p = Pdb()
    def quits():
        raise BdbQuit
    result = p.runcall(quits)
    assert result is None
    print("test_pdb_runcall_bdbquit: OK")


# ── 6. Module-level functions ─────────────────────────────────────────────────
def test_set_trace_noop():
    pdb.set_trace()
    print("test_set_trace_noop: OK")


def test_set_trace_async_noop():
    pdb.set_trace_async()
    print("test_set_trace_async_noop: OK")


def test_runcall_module_basic():
    result = pdb.runcall(lambda x: x + 1, 99)
    assert result == 100
    print("test_runcall_module_basic: OK")


def test_runcall_module_bdbquit():
    def quits():
        raise BdbQuit
    result = pdb.runcall(quits)
    assert result is None
    print("test_runcall_module_bdbquit: OK")


def test_post_mortem_noop():
    pdb.post_mortem()
    print("test_post_mortem_noop: OK")


def test_pm_noop():
    pdb.pm()
    print("test_pm_noop: OK")


def test_run_noop():
    pdb.run("x = 1")
    print("test_run_noop: OK")


def test_runeval_noop():
    pdb.runeval("1 + 1")
    print("test_runeval_noop: OK")


# ── 7. Default backend ────────────────────────────────────────────────────────
def test_get_default_backend():
    backend = pdb.get_default_backend()
    assert isinstance(backend, str)
    assert backend in ('settrace', 'monitoring')
    print("test_get_default_backend: OK")


def test_set_default_backend():
    old = pdb.get_default_backend()
    pdb.set_default_backend('settrace')
    assert pdb.get_default_backend() == 'settrace'
    pdb.set_default_backend('monitoring')
    assert pdb.get_default_backend() == 'monitoring'
    pdb.set_default_backend(old)
    assert pdb.get_default_backend() == old
    print("test_set_default_backend: OK")


# ── 8. Restart exception ──────────────────────────────────────────────────────
def test_restart_exception_exists():
    assert Restart is not None
    print("test_restart_exception_exists: OK")


def test_restart_raise_catch():
    try:
        raise Restart
    except Restart:
        pass
    print("test_restart_raise_catch: OK")


def test_restart_is_exception():
    assert issubclass(Restart, Exception)
    print("test_restart_is_exception: OK")


if __name__ == "__main__":
    test_import()
    test_pdb_class_exists()
    test_pdb_constructor_default()
    test_pdb_constructor_nosigint()
    test_pdb_constructor_readrc_false()
    test_pdb_inherits_bdb()
    test_pdb_has_breaks()
    test_pdb_quitting_initial()
    test_pdb_canonic_special()
    test_pdb_set_break()
    test_pdb_clear_break()
    test_pdb_skip()
    test_pdb_set_quit()
    test_pdb_multiple_breakpoints()
    test_pdb_get_all_breaks_empty()
    test_pdb_runcall_basic()
    test_pdb_runcall_kwargs()
    test_pdb_runcall_exception()
    test_pdb_runcall_bdbquit()
    test_set_trace_noop()
    test_set_trace_async_noop()
    test_runcall_module_basic()
    test_runcall_module_bdbquit()
    test_post_mortem_noop()
    test_pm_noop()
    test_run_noop()
    test_runeval_noop()
    test_get_default_backend()
    test_set_default_backend()
    test_restart_exception_exists()
    test_restart_raise_catch()
    test_restart_is_exception()

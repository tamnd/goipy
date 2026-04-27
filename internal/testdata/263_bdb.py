"""Fixture 263 — bdb debugger base module"""
import bdb
from bdb import Bdb, Breakpoint, BdbQuit
import io


# ── 1. BdbQuit exception ──────────────────────────────────────────────────
def test_bdbquit_raise_catch():
    try:
        raise BdbQuit
    except BdbQuit:
        pass
    print("test_bdbquit_raise_catch: OK")


def test_bdbquit_is_exception():
    assert issubclass(BdbQuit, Exception)
    print("test_bdbquit_is_exception: OK")


# ── 2. Breakpoint creation ────────────────────────────────────────────────
def test_breakpoint_creation():
    bp = Breakpoint("test263a.py", 10)
    assert bp.file == "test263a.py"
    assert bp.line == 10
    assert bp.temporary == False
    assert bp.cond is None
    assert bp.funcname is None
    assert bp.enabled == True
    assert bp.hits == 0
    assert isinstance(bp.number, int)
    assert bp.number >= 1
    print("test_breakpoint_creation: OK")


def test_breakpoint_all_params():
    bp = Breakpoint("test263b.py", 20,
                    temporary=True, cond="x > 0", funcname="my_func")
    assert bp.file == "test263b.py"
    assert bp.line == 20
    assert bp.temporary == True
    assert bp.cond == "x > 0"
    assert bp.funcname == "my_func"
    print("test_breakpoint_all_params: OK")


# ── 3. Breakpoint enable/disable ──────────────────────────────────────────
def test_breakpoint_enable_disable():
    bp = Breakpoint("test263c.py", 5)
    assert bp.enabled == True
    bp.disable()
    assert bp.enabled == False
    bp.enable()
    assert bp.enabled == True
    print("test_breakpoint_enable_disable: OK")


# ── 4. Breakpoint bpformat ────────────────────────────────────────────────
def test_breakpoint_bpformat_basic():
    bp = Breakpoint("test263d.py", 42)
    s = bp.bpformat()
    assert isinstance(s, str)
    assert "42" in s
    assert "test263d.py" in s
    print("test_breakpoint_bpformat_basic: OK")


def test_breakpoint_bpformat_condition():
    bp = Breakpoint("test263e.py", 7, cond="i == 0")
    s = bp.bpformat()
    assert "i == 0" in s
    print("test_breakpoint_bpformat_condition: OK")


def test_breakpoint_bpformat_temporary():
    bp_keep = Breakpoint("test263f.py", 1)
    bp_temp = Breakpoint("test263f.py", 2, temporary=True)
    assert "keep" in bp_keep.bpformat()
    assert "del" in bp_temp.bpformat()
    print("test_breakpoint_bpformat_temporary: OK")


def test_breakpoint_bpformat_disabled():
    bp = Breakpoint("test263g.py", 3)
    bp.disable()
    s = bp.bpformat()
    assert "no" in s
    print("test_breakpoint_bpformat_disabled: OK")


# ── 5. Breakpoint bpprint ─────────────────────────────────────────────────
def test_breakpoint_bpprint():
    bp = Breakpoint("test263h.py", 99)
    buf = io.StringIO()
    bp.bpprint(out=buf)
    s = buf.getvalue()
    assert "99" in s
    assert "test263h.py" in s
    print("test_breakpoint_bpprint: OK")


# ── 6. Breakpoint class-level tracking ───────────────────────────────────
def test_breakpoint_tracking():
    bp = Breakpoint("test263i.py", 11)
    num = bp.number
    assert Breakpoint.bpbynumber[num] is bp
    print("test_breakpoint_tracking: OK")


def test_breakpoint_delete():
    bp = Breakpoint("test263j.py", 77)
    num = bp.number
    assert Breakpoint.bpbynumber[num] is bp
    bp.deleteMe()
    assert Breakpoint.bpbynumber[num] is None
    print("test_breakpoint_delete: OK")


# ── 7. Bdb constructor ────────────────────────────────────────────────────
def test_bdb_constructor():
    b = Bdb()
    assert b.breaks == {}
    assert b.quitting == False
    print("test_bdb_constructor: OK")


def test_bdb_skip():
    b = Bdb(skip=["os", "sys", "re"])
    assert b.is_skipped_module("os") == True
    assert b.is_skipped_module("sys") == True
    assert b.is_skipped_module("re") == True
    assert b.is_skipped_module("bdb") == False
    assert b.is_skipped_module("json") == False
    print("test_bdb_skip: OK")


# ── 8. Bdb.canonic ────────────────────────────────────────────────────────
def test_bdb_canonic_special():
    b = Bdb()
    assert b.canonic("<string>") == "<string>"
    assert b.canonic("<stdin>") == "<stdin>"
    assert b.canonic("<module>") == "<module>"
    print("test_bdb_canonic_special: OK")


def test_bdb_canonic_regular():
    b = Bdb()
    result = b.canonic("somefile.py")
    assert isinstance(result, str)
    assert "somefile.py" in result
    # canonic caches results
    result2 = b.canonic("somefile.py")
    assert result == result2
    print("test_bdb_canonic_regular: OK")


# ── 9. Bdb breakpoint management ──────────────────────────────────────────
def test_bdb_set_break():
    b = Bdb()
    result = b.set_break("test263k.py", 10)
    assert result is None
    breaks = b.get_breaks("test263k.py", 10)
    assert len(breaks) == 1
    assert breaks[0].line == 10
    print("test_bdb_set_break: OK")


def test_bdb_set_break_condition():
    b = Bdb()
    b.set_break("test263l.py", 5, cond="x > 10")
    breaks = b.get_breaks("test263l.py", 5)
    assert len(breaks) == 1
    assert breaks[0].cond == "x > 10"
    print("test_bdb_set_break_condition: OK")


def test_bdb_set_break_temporary():
    b = Bdb()
    b.set_break("test263m.py", 3, temporary=True)
    breaks = b.get_breaks("test263m.py", 3)
    assert breaks[0].temporary == True
    print("test_bdb_set_break_temporary: OK")


def test_bdb_set_break_multiple():
    b = Bdb()
    b.set_break("test263n.py", 1)
    b.set_break("test263n.py", 2)
    b.set_break("test263n.py", 3)
    assert len(b.get_breaks("test263n.py", 1)) == 1
    assert len(b.get_breaks("test263n.py", 2)) == 1
    assert len(b.get_breaks("test263n.py", 3)) == 1
    assert len(b.get_breaks("test263n.py", 4)) == 0
    print("test_bdb_set_break_multiple: OK")


def test_bdb_clear_break():
    b = Bdb()
    b.set_break("test263o.py", 7)
    b.clear_break("test263o.py", 7)
    breaks = b.get_breaks("test263o.py", 7)
    assert breaks == []
    print("test_bdb_clear_break: OK")


def test_bdb_clear_all_breaks():
    b = Bdb()
    b.set_break("test263p.py", 1)
    b.set_break("test263p.py", 2)
    b.set_break("test263q.py", 3)
    b.clear_all_breaks()
    all_breaks = b.get_all_breaks()
    assert all_breaks == {}
    print("test_bdb_clear_all_breaks: OK")


def test_bdb_get_file_breaks():
    b = Bdb()
    b.set_break("test263r.py", 10)
    b.set_break("test263r.py", 20)
    b.set_break("test263s.py", 5)
    fb = b.get_file_breaks("test263r.py")
    assert len(fb) == 2
    lines = sorted(bp.line for bp in fb)
    assert lines == [10, 20]
    print("test_bdb_get_file_breaks: OK")


def test_bdb_get_all_breaks():
    b = Bdb()
    b.set_break("test263t.py", 1)
    b.set_break("test263u.py", 2)
    all_breaks = b.get_all_breaks()
    assert isinstance(all_breaks, dict)
    assert len(all_breaks) == 2
    print("test_bdb_get_all_breaks: OK")


def test_bdb_clear_bpbynumber():
    b = Bdb()
    b.set_break("test263v.py", 15)
    breaks = b.get_breaks("test263v.py", 15)
    num = breaks[0].number
    result = b.clear_bpbynumber(num)
    assert result is None
    assert b.get_breaks("test263v.py", 15) == []
    print("test_bdb_clear_bpbynumber: OK")


def test_bdb_no_break():
    b = Bdb()
    breaks = b.get_breaks("nonexistent263.py", 99)
    assert breaks == []
    print("test_bdb_no_break: OK")


# ── 10. Bdb control methods ───────────────────────────────────────────────
def test_bdb_set_quit():
    b = Bdb()
    assert b.quitting == False
    b.set_quit()
    assert b.quitting == True
    print("test_bdb_set_quit: OK")


def test_bdb_reset():
    b = Bdb()
    b.set_quit()
    assert b.quitting == True
    b.reset()
    assert b.quitting == False
    print("test_bdb_reset: OK")


# ── 11. Bdb.runcall ───────────────────────────────────────────────────────
def test_bdb_runcall_basic():
    b = Bdb()
    result = b.runcall(lambda x, y: x + y, 10, 20)
    assert result == 30
    print("test_bdb_runcall_basic: OK")


def test_bdb_runcall_exception():
    b = Bdb()
    def raises():
        raise ValueError("from func")
    try:
        b.runcall(raises)
        assert False, "should have raised"
    except ValueError as e:
        assert str(e) == "from func"
    print("test_bdb_runcall_exception: OK")


def test_bdb_runcall_bdbquit():
    b = Bdb()
    def quits():
        raise BdbQuit
    result = b.runcall(quits)
    assert result is None   # BdbQuit is swallowed
    print("test_bdb_runcall_bdbquit: OK")


if __name__ == "__main__":
    test_bdbquit_raise_catch()
    test_bdbquit_is_exception()
    test_breakpoint_creation()
    test_breakpoint_all_params()
    test_breakpoint_enable_disable()
    test_breakpoint_bpformat_basic()
    test_breakpoint_bpformat_condition()
    test_breakpoint_bpformat_temporary()
    test_breakpoint_bpformat_disabled()
    test_breakpoint_bpprint()
    test_breakpoint_tracking()
    test_breakpoint_delete()
    test_bdb_constructor()
    test_bdb_skip()
    test_bdb_canonic_special()
    test_bdb_canonic_regular()
    test_bdb_set_break()
    test_bdb_set_break_condition()
    test_bdb_set_break_temporary()
    test_bdb_set_break_multiple()
    test_bdb_clear_break()
    test_bdb_clear_all_breaks()
    test_bdb_get_file_breaks()
    test_bdb_get_all_breaks()
    test_bdb_clear_bpbynumber()
    test_bdb_no_break()
    test_bdb_set_quit()
    test_bdb_reset()
    test_bdb_runcall_basic()
    test_bdb_runcall_exception()
    test_bdb_runcall_bdbquit()

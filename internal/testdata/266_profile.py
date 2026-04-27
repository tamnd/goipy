"""Fixture 266 — profile / cProfile / pstats modules"""
import profile
import cProfile
import pstats
import io
import os
import tempfile


# ── 1. Import ─────────────────────────────────────────────────────────────────
def test_import():
    import profile as p
    import cProfile as cp
    import pstats as ps
    assert p is not None
    assert cp is not None
    assert ps is not None
    print("test_import: OK")


# ── 2. profile.Profile constructor ───────────────────────────────────────────
def test_profile_constructor():
    p = profile.Profile()
    assert p is not None
    print("test_profile_constructor: OK")


def test_cprofile_constructor():
    p = cProfile.Profile()
    assert p is not None
    print("test_cprofile_constructor: OK")


def test_profile_constructor_with_timer():
    p = profile.Profile(timer=None, timeunit=0.0, subcalls=True, builtins=True)
    assert p is not None
    print("test_profile_constructor_with_timer: OK")


# ── 3. runcall ────────────────────────────────────────────────────────────────
def test_profile_runcall_returns_value():
    p = profile.Profile()
    result = p.runcall(lambda x: x * 3, 7)
    assert result == 21
    print("test_profile_runcall_returns_value: OK")


def test_cprofile_runcall_returns_value():
    p = cProfile.Profile()
    result = p.runcall(lambda x: x + 1, 41)
    assert result == 42
    print("test_cprofile_runcall_returns_value: OK")


def test_profile_runcall_with_kwargs():
    p = profile.Profile()
    def add(a, b=0):
        return a + b
    result = p.runcall(add, 3, b=4)
    assert result == 7
    print("test_profile_runcall_with_kwargs: OK")


def test_profile_runcall_exception_propagates():
    p = profile.Profile()
    def raises():
        raise ValueError("oops")
    try:
        p.runcall(raises)
        assert False, "should have raised"
    except ValueError as e:
        assert str(e) == "oops"
    print("test_profile_runcall_exception_propagates: OK")


# ── 4. enable / disable ───────────────────────────────────────────────────────
def test_cprofile_enable_disable():
    p = cProfile.Profile()
    p.enable()
    _ = sum(range(10))
    p.disable()
    print("test_cprofile_enable_disable: OK")


def test_profile_enable_disable():
    p = profile.Profile()
    p.enable()
    p.disable()
    print("test_profile_enable_disable: OK")


# ── 5. context manager ────────────────────────────────────────────────────────
def test_cprofile_context_manager():
    with cProfile.Profile() as p:
        _ = 1 + 1
    assert p is not None
    print("test_cprofile_context_manager: OK")


def test_profile_context_manager():
    with profile.Profile() as p:
        _ = 2 + 2
    assert p is not None
    print("test_profile_context_manager: OK")


# ── 6. create_stats ───────────────────────────────────────────────────────────
def test_profile_create_stats():
    p = profile.Profile()
    p.runcall(lambda: None)
    p.create_stats()
    print("test_profile_create_stats: OK")


# ── 7. print_stats ────────────────────────────────────────────────────────────
def test_profile_print_stats():
    p = profile.Profile()
    p.runcall(lambda x: x, 1)
    p.print_stats()   # writes to stdout, should not crash
    print("test_profile_print_stats: OK")


def test_cprofile_print_stats():
    p = cProfile.Profile()
    p.enable()
    _ = list(range(5))
    p.disable()
    p.print_stats()
    print("test_cprofile_print_stats: OK")


# ── 8. dump_stats ─────────────────────────────────────────────────────────────
def test_profile_dump_stats():
    p = profile.Profile()
    p.runcall(lambda: 42)
    fd, fname = tempfile.mkstemp(suffix=".prof")
    os.close(fd)
    try:
        p.dump_stats(fname)
        assert os.path.exists(fname)
        assert os.path.getsize(fname) > 0
    finally:
        os.unlink(fname)
    print("test_profile_dump_stats: OK")


# ── 9. module-level run / runctx / runcall ────────────────────────────────────
def test_profile_run_noop():
    profile.run("x = 1 + 1")
    print("test_profile_run_noop: OK")


def test_cprofile_run_noop():
    cProfile.run("x = 1")
    print("test_cprofile_run_noop: OK")


def test_profile_runctx_noop():
    profile.runctx("x = 1", {}, {})
    print("test_profile_runctx_noop: OK")


def test_profile_module_runcall():
    result = profile.runcall(lambda x: x * 2, 5)
    assert result == 10
    print("test_profile_module_runcall: OK")


def test_cprofile_module_runcall():
    result = cProfile.runcall(lambda x: x + 10, 5)
    assert result == 15
    print("test_cprofile_module_runcall: OK")


# ── 10. calibrate ─────────────────────────────────────────────────────────────
def test_profile_calibrate():
    p = profile.Profile()
    bias = p.calibrate(10)
    assert isinstance(bias, float)
    print("test_profile_calibrate: OK")


# ── 11. pstats.Stats from Profile ─────────────────────────────────────────────
def test_pstats_from_profile():
    p = cProfile.Profile()
    p.runcall(lambda: sum(range(5)))
    buf = io.StringIO()
    stats = pstats.Stats(p, stream=buf)
    assert stats is not None
    print("test_pstats_from_profile: OK")


def test_pstats_print_stats():
    p = cProfile.Profile()
    p.runcall(lambda: None)
    buf = io.StringIO()
    stats = pstats.Stats(p, stream=buf)
    stats.print_stats()
    s = buf.getvalue()
    assert isinstance(s, str)
    assert len(s) > 0
    print("test_pstats_print_stats: OK")


def test_pstats_print_stats_has_calls():
    p = cProfile.Profile()
    p.runcall(lambda: None)
    buf = io.StringIO()
    stats = pstats.Stats(p, stream=buf)
    stats.print_stats()
    s = buf.getvalue()
    assert "function call" in s
    print("test_pstats_print_stats_has_calls: OK")


# ── 12. Stats chaining ────────────────────────────────────────────────────────
def test_pstats_strip_dirs():
    p = cProfile.Profile()
    buf = io.StringIO()
    stats = pstats.Stats(p, stream=buf)
    result = stats.strip_dirs()
    assert result is stats
    print("test_pstats_strip_dirs: OK")


def test_pstats_sort_stats():
    p = cProfile.Profile()
    p.runcall(lambda: None)
    buf = io.StringIO()
    stats = pstats.Stats(p, stream=buf)
    result = stats.sort_stats("cumulative")
    assert result is stats
    print("test_pstats_sort_stats: OK")


def test_pstats_sort_stats_time():
    p = cProfile.Profile()
    buf = io.StringIO()
    stats = pstats.Stats(p, stream=buf)
    result = stats.sort_stats("time")
    assert result is stats
    print("test_pstats_sort_stats_time: OK")


def test_pstats_sort_stats_calls():
    p = cProfile.Profile()
    buf = io.StringIO()
    stats = pstats.Stats(p, stream=buf)
    result = stats.sort_stats("calls")
    assert result is stats
    print("test_pstats_sort_stats_calls: OK")


def test_pstats_sort_stats_numeric():
    p = cProfile.Profile()
    buf = io.StringIO()
    stats = pstats.Stats(p, stream=buf)
    result = stats.sort_stats(2)  # cumulative
    assert result is stats
    print("test_pstats_sort_stats_numeric: OK")


def test_pstats_reverse_order():
    p = cProfile.Profile()
    buf = io.StringIO()
    stats = pstats.Stats(p, stream=buf)
    result = stats.reverse_order()
    assert result is stats
    print("test_pstats_reverse_order: OK")


def test_pstats_print_callers():
    p = cProfile.Profile()
    buf = io.StringIO()
    stats = pstats.Stats(p, stream=buf)
    result = stats.print_callers()
    assert result is stats
    print("test_pstats_print_callers: OK")


def test_pstats_print_callees():
    p = cProfile.Profile()
    buf = io.StringIO()
    stats = pstats.Stats(p, stream=buf)
    result = stats.print_callees()
    assert result is stats
    print("test_pstats_print_callees: OK")


# ── 13. Stats.add ─────────────────────────────────────────────────────────────
def test_pstats_add():
    p1 = cProfile.Profile()
    p1.runcall(lambda: None)
    p2 = cProfile.Profile()
    p2.runcall(lambda: None)
    buf = io.StringIO()
    stats = pstats.Stats(p1, stream=buf)
    result = stats.add(p2)
    assert result is stats
    print("test_pstats_add: OK")


# ── 14. Stats.dump_stats + load from file ────────────────────────────────────
def test_pstats_dump_load():
    p = cProfile.Profile()
    p.runcall(lambda: sum(range(3)))
    fd, fname = tempfile.mkstemp(suffix=".prof")
    os.close(fd)
    try:
        p.dump_stats(fname)
        buf = io.StringIO()
        stats2 = pstats.Stats(fname, stream=buf)
        stats2.print_stats()
        s = buf.getvalue()
        assert isinstance(s, str)
    finally:
        os.unlink(fname)
    print("test_pstats_dump_load: OK")


# ── 15. get_stats_profile ─────────────────────────────────────────────────────
def test_pstats_get_stats_profile():
    p = cProfile.Profile()
    p.runcall(lambda: 1)
    buf = io.StringIO()
    stats = pstats.Stats(p, stream=buf)
    sp = stats.get_stats_profile()
    assert sp is not None
    assert hasattr(sp, "total_tt")
    assert hasattr(sp, "total_calls")
    print("test_pstats_get_stats_profile: OK")


# ── 16. SortKey enum ──────────────────────────────────────────────────────────
def test_pstats_sort_key_exists():
    from pstats import SortKey
    assert SortKey is not None
    print("test_pstats_sort_key_exists: OK")


def test_pstats_sort_key_cumulative():
    from pstats import SortKey
    assert SortKey.CUMULATIVE == "cumulative"
    print("test_pstats_sort_key_cumulative: OK")


def test_pstats_sort_key_time():
    from pstats import SortKey
    assert SortKey.TIME == "time"
    print("test_pstats_sort_key_time: OK")


def test_pstats_sort_key_calls():
    from pstats import SortKey
    assert SortKey.CALLS == "calls"
    print("test_pstats_sort_key_calls: OK")


def test_pstats_sort_key_filename():
    from pstats import SortKey
    assert SortKey.FILENAME == "filename"
    print("test_pstats_sort_key_filename: OK")


def test_pstats_sort_key_name():
    from pstats import SortKey
    assert SortKey.NAME == "name"
    print("test_pstats_sort_key_name: OK")


def test_pstats_sort_key_in_sort_stats():
    from pstats import SortKey
    p = cProfile.Profile()
    buf = io.StringIO()
    stats = pstats.Stats(p, stream=buf)
    result = stats.sort_stats(SortKey.CUMULATIVE)
    assert result is stats
    print("test_pstats_sort_key_in_sort_stats: OK")


if __name__ == "__main__":
    test_import()
    test_profile_constructor()
    test_cprofile_constructor()
    test_profile_constructor_with_timer()
    test_profile_runcall_returns_value()
    test_cprofile_runcall_returns_value()
    test_profile_runcall_with_kwargs()
    test_profile_runcall_exception_propagates()
    test_cprofile_enable_disable()
    test_profile_enable_disable()
    test_cprofile_context_manager()
    test_profile_context_manager()
    test_profile_create_stats()
    test_profile_print_stats()
    test_cprofile_print_stats()
    test_profile_dump_stats()
    test_profile_run_noop()
    test_cprofile_run_noop()
    test_profile_runctx_noop()
    test_profile_module_runcall()
    test_cprofile_module_runcall()
    test_profile_calibrate()
    test_pstats_from_profile()
    test_pstats_print_stats()
    test_pstats_print_stats_has_calls()
    test_pstats_strip_dirs()
    test_pstats_sort_stats()
    test_pstats_sort_stats_time()
    test_pstats_sort_stats_calls()
    test_pstats_sort_stats_numeric()
    test_pstats_reverse_order()
    test_pstats_print_callers()
    test_pstats_print_callees()
    test_pstats_add()
    test_pstats_dump_load()
    test_pstats_get_stats_profile()
    test_pstats_sort_key_exists()
    test_pstats_sort_key_cumulative()
    test_pstats_sort_key_time()
    test_pstats_sort_key_calls()
    test_pstats_sort_key_filename()
    test_pstats_sort_key_name()
    test_pstats_sort_key_in_sort_stats()

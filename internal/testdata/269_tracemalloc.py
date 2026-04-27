import tracemalloc


def run(name, fn):
    try:
        fn()
        print(f"{name}: OK")
    except Exception as e:
        print(f"{name}: FAIL ({e})")


# ── import ────────────────────────────────────────────────────────────────────

def test_import():
    import tracemalloc as t
    assert t is not None

run("test_import", test_import)


# ── start / stop / is_tracing ─────────────────────────────────────────────────

def test_is_tracing_initially_false():
    tracemalloc.stop()
    assert tracemalloc.is_tracing() == False

run("test_is_tracing_initially_false", test_is_tracing_initially_false)


def test_start_sets_tracing():
    tracemalloc.start()
    assert tracemalloc.is_tracing() == True
    tracemalloc.stop()

run("test_start_sets_tracing", test_start_sets_tracing)


def test_stop_clears_tracing():
    tracemalloc.start()
    tracemalloc.stop()
    assert tracemalloc.is_tracing() == False

run("test_stop_clears_tracing", test_stop_clears_tracing)


def test_start_nframe():
    tracemalloc.start(nframe=5)
    assert tracemalloc.get_traceback_limit() == 5
    tracemalloc.stop()

run("test_start_nframe", test_start_nframe)


def test_start_default_nframe():
    tracemalloc.start()
    assert tracemalloc.get_traceback_limit() >= 1
    tracemalloc.stop()

run("test_start_default_nframe", test_start_default_nframe)


# ── clear_traces ──────────────────────────────────────────────────────────────

def test_clear_traces():
    tracemalloc.start()
    tracemalloc.clear_traces()  # no-op, no crash
    tracemalloc.stop()

run("test_clear_traces", test_clear_traces)


# ── get_traced_memory ─────────────────────────────────────────────────────────

def test_get_traced_memory_returns_tuple():
    result = tracemalloc.get_traced_memory()
    assert isinstance(result, tuple)
    assert len(result) == 2

run("test_get_traced_memory_returns_tuple", test_get_traced_memory_returns_tuple)


def test_get_traced_memory_ints():
    current, peak = tracemalloc.get_traced_memory()
    assert isinstance(current, int)
    assert isinstance(peak, int)
    assert current >= 0
    assert peak >= 0

run("test_get_traced_memory_ints", test_get_traced_memory_ints)


# ── get_tracemalloc_memory ────────────────────────────────────────────────────

def test_get_tracemalloc_memory():
    result = tracemalloc.get_tracemalloc_memory()
    assert isinstance(result, int)
    assert result >= 0

run("test_get_tracemalloc_memory", test_get_tracemalloc_memory)


# ── reset_peak ────────────────────────────────────────────────────────────────

def test_reset_peak():
    tracemalloc.reset_peak()  # no-op, no crash

run("test_reset_peak", test_reset_peak)


# ── get_object_traceback ──────────────────────────────────────────────────────

def test_get_object_traceback_returns_none():
    obj = []
    result = tracemalloc.get_object_traceback(obj)
    assert result is None

run("test_get_object_traceback_returns_none", test_get_object_traceback_returns_none)


# ── take_snapshot ─────────────────────────────────────────────────────────────

def test_take_snapshot_returns_snapshot():
    tracemalloc.start()
    snap = tracemalloc.take_snapshot()
    assert snap is not None
    tracemalloc.stop()

run("test_take_snapshot_returns_snapshot", test_take_snapshot_returns_snapshot)


def test_snapshot_has_traces():
    tracemalloc.start()
    snap = tracemalloc.take_snapshot()
    assert hasattr(snap, "traces")
    tracemalloc.stop()

run("test_snapshot_has_traces", test_snapshot_has_traces)


def test_snapshot_has_traceback_limit():
    tracemalloc.start(nframe=3)
    snap = tracemalloc.take_snapshot()
    assert hasattr(snap, "traceback_limit")
    assert isinstance(snap.traceback_limit, int)
    tracemalloc.stop()

run("test_snapshot_has_traceback_limit", test_snapshot_has_traceback_limit)


def test_snapshot_statistics_empty():
    tracemalloc.start()
    snap = tracemalloc.take_snapshot()
    stats = snap.statistics("lineno")
    assert isinstance(stats, list)
    tracemalloc.stop()

run("test_snapshot_statistics_empty", test_snapshot_statistics_empty)


def test_snapshot_statistics_key_types():
    tracemalloc.start()
    snap = tracemalloc.take_snapshot()
    for key in ("filename", "lineno", "traceback"):
        stats = snap.statistics(key)
        assert isinstance(stats, list)
    tracemalloc.stop()

run("test_snapshot_statistics_key_types", test_snapshot_statistics_key_types)


def test_snapshot_compare_to():
    tracemalloc.start()
    snap1 = tracemalloc.take_snapshot()
    snap2 = tracemalloc.take_snapshot()
    diff = snap2.compare_to(snap1, "lineno")
    assert isinstance(diff, list)
    tracemalloc.stop()

run("test_snapshot_compare_to", test_snapshot_compare_to)


def test_snapshot_filter_traces():
    tracemalloc.start()
    snap = tracemalloc.take_snapshot()
    f = tracemalloc.Filter(True, "*.py")
    snap2 = snap.filter_traces([f])
    assert snap2 is not None
    tracemalloc.stop()

run("test_snapshot_filter_traces", test_snapshot_filter_traces)


def test_snapshot_dump_load():
    import tempfile, os
    tracemalloc.start()
    snap = tracemalloc.take_snapshot()
    tracemalloc.stop()
    with tempfile.NamedTemporaryFile(suffix=".pickle", delete=False) as f:
        path = f.name
    try:
        snap.dump(path)
        snap2 = tracemalloc.Snapshot.load(path)
        assert snap2 is not None
    finally:
        os.unlink(path)

run("test_snapshot_dump_load", test_snapshot_dump_load)


# ── Frame ─────────────────────────────────────────────────────────────────────

def test_frame_constructor():
    f = tracemalloc.Frame("test.py", 42)
    assert f.filename == "test.py"
    assert f.lineno == 42

run("test_frame_constructor", test_frame_constructor)


def test_frame_attributes():
    f = tracemalloc.Frame("foo/bar.py", 10)
    assert isinstance(f.filename, str)
    assert isinstance(f.lineno, int)

run("test_frame_attributes", test_frame_attributes)


# ── Traceback ─────────────────────────────────────────────────────────────────

def test_traceback_constructor():
    tb = tracemalloc.Traceback([])
    assert tb is not None

run("test_traceback_constructor", test_traceback_constructor)


def test_traceback_total_nframe():
    tb = tracemalloc.Traceback([])
    assert tb.total_nframe is None

run("test_traceback_total_nframe", test_traceback_total_nframe)


def test_traceback_format():
    tb = tracemalloc.Traceback([])
    result = tb.format()
    assert isinstance(result, list)

run("test_traceback_format", test_traceback_format)


def test_traceback_format_with_frame():
    f = tracemalloc.Frame("test.py", 5)
    tb = tracemalloc.Traceback([f])
    result = tb.format()
    assert isinstance(result, list)
    assert len(result) >= 1

run("test_traceback_format_with_frame", test_traceback_format_with_frame)


# ── Statistic ─────────────────────────────────────────────────────────────────

def test_statistic_constructor():
    tb = tracemalloc.Traceback([])
    s = tracemalloc.Statistic(tb, 10, 1024)
    assert s.count == 10
    assert s.size == 1024
    assert s.traceback is tb

run("test_statistic_constructor", test_statistic_constructor)


def test_statistic_str():
    tb = tracemalloc.Traceback([])
    s = tracemalloc.Statistic(tb, 5, 512)
    result = str(s)
    assert isinstance(result, str)

run("test_statistic_str", test_statistic_str)


# ── StatisticDiff ─────────────────────────────────────────────────────────────

def test_statistic_diff_constructor():
    tb = tracemalloc.Traceback([])
    d = tracemalloc.StatisticDiff(tb, 10, 1024, 5, 256)
    assert d.count == 10
    assert d.size == 1024
    assert d.count_diff == 5
    assert d.size_diff == 256

run("test_statistic_diff_constructor", test_statistic_diff_constructor)


def test_statistic_diff_str():
    tb = tracemalloc.Traceback([])
    d = tracemalloc.StatisticDiff(tb, 3, 300, -1, -50)
    result = str(d)
    assert isinstance(result, str)

run("test_statistic_diff_str", test_statistic_diff_str)


# ── Filter ────────────────────────────────────────────────────────────────────

def test_filter_constructor():
    f = tracemalloc.Filter(True, "*.py")
    assert f.inclusive == True
    assert f.filename_pattern == "*.py"
    assert f.lineno is None
    assert f.all_frames == False
    assert f.domain is None

run("test_filter_constructor", test_filter_constructor)


def test_filter_with_lineno():
    f = tracemalloc.Filter(False, "test.py", lineno=42)
    assert f.inclusive == False
    assert f.lineno == 42

run("test_filter_with_lineno", test_filter_with_lineno)


def test_filter_all_params():
    f = tracemalloc.Filter(True, "*.py", lineno=10, all_frames=True, domain=0)
    assert f.all_frames == True
    assert f.domain == 0

run("test_filter_all_params", test_filter_all_params)


# ── DomainFilter ──────────────────────────────────────────────────────────────

def test_domain_filter_constructor():
    f = tracemalloc.DomainFilter(True, 0)
    assert f.inclusive == True
    assert f.domain == 0

run("test_domain_filter_constructor", test_domain_filter_constructor)


def test_domain_filter_exclusive():
    f = tracemalloc.DomainFilter(False, 1)
    assert f.inclusive == False
    assert f.domain == 1

run("test_domain_filter_exclusive", test_domain_filter_exclusive)

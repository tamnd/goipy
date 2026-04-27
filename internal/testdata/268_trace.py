import trace


def run(name, fn):
    try:
        fn()
        print(f"{name}: OK")
    except Exception as e:
        print(f"{name}: FAIL ({e})")


# ── import ─────────────────────────────────────────────────────────────────────

def test_import():
    import trace as t
    assert t is not None

run("test_import", test_import)


# ── Trace constructor ──────────────────────────────────────────────────────────

def test_trace_constructor_defaults():
    t = trace.Trace()
    assert t is not None

run("test_trace_constructor_defaults", test_trace_constructor_defaults)


def test_trace_constructor_params():
    t = trace.Trace(count=1, trace=0, countfuncs=1, countcallers=0,
                    ignoremods=(), ignoredirs=(), timing=False)
    assert t is not None

run("test_trace_constructor_params", test_trace_constructor_params)


def test_trace_constructor_infile_none():
    t = trace.Trace(infile=None, outfile=None)
    assert t is not None

run("test_trace_constructor_infile_none", test_trace_constructor_infile_none)


# ── runfunc ────────────────────────────────────────────────────────────────────

def test_runfunc_return_value():
    t = trace.Trace()
    result = t.runfunc(lambda: 42)
    assert result == 42, f"expected 42, got {result}"

run("test_runfunc_return_value", test_runfunc_return_value)


def test_runfunc_with_args():
    t = trace.Trace()
    result = t.runfunc(lambda x, y: x + y, 3, 4)
    assert result == 7, f"expected 7, got {result}"

run("test_runfunc_with_args", test_runfunc_with_args)


def test_runfunc_with_kwargs():
    def add(x, y=10):
        return x + y
    t = trace.Trace()
    result = t.runfunc(add, 5, y=20)
    assert result == 25, f"expected 25, got {result}"

run("test_runfunc_with_kwargs", test_runfunc_with_kwargs)


def test_runfunc_none_return():
    t = trace.Trace()
    result = t.runfunc(lambda: None)
    assert result is None

run("test_runfunc_none_return", test_runfunc_none_return)


def test_runfunc_side_effects():
    calls = []
    t = trace.Trace()
    t.runfunc(lambda: calls.append(1))
    assert calls == [1]

run("test_runfunc_side_effects", test_runfunc_side_effects)


def test_runfunc_exception_propagates():
    t = trace.Trace()
    try:
        t.runfunc(lambda: 1 / 0)
        assert False, "should have raised"
    except ZeroDivisionError:
        pass

run("test_runfunc_exception_propagates", test_runfunc_exception_propagates)


# ── run / runctx stubs ────────────────────────────────────────────────────────

def test_run_stub():
    t = trace.Trace()
    t.run("pass")  # no-op

run("test_run_stub", test_run_stub)


def test_runctx_stub():
    t = trace.Trace()
    t.runctx("pass", globals(), locals())  # no-op

run("test_runctx_stub", test_runctx_stub)


# ── results() ─────────────────────────────────────────────────────────────────

def test_results_returns_object():
    t = trace.Trace()
    r = t.results()
    assert r is not None

run("test_results_returns_object", test_results_returns_object)


def test_results_has_counts():
    t = trace.Trace()
    r = t.results()
    assert hasattr(r, "counts")
    assert isinstance(r.counts, dict)

run("test_results_has_counts", test_results_has_counts)


def test_results_has_calledfuncs():
    t = trace.Trace()
    r = t.results()
    assert hasattr(r, "calledfuncs")
    assert isinstance(r.calledfuncs, dict)

run("test_results_has_calledfuncs", test_results_has_calledfuncs)


def test_results_has_callers():
    t = trace.Trace()
    r = t.results()
    assert hasattr(r, "callers")
    assert isinstance(r.callers, dict)

run("test_results_has_callers", test_results_has_callers)


def test_results_after_runfunc():
    t = trace.Trace(countfuncs=1)
    t.runfunc(lambda: None)
    r = t.results()
    assert r is not None

run("test_results_after_runfunc", test_results_after_runfunc)


# ── CoverageResults ───────────────────────────────────────────────────────────

def test_coverage_results_update():
    t1 = trace.Trace(countfuncs=1)
    t1.runfunc(lambda: None)
    r1 = t1.results()

    t2 = trace.Trace(countfuncs=1)
    t2.runfunc(lambda: None)
    r2 = t2.results()

    r1.update(r2)   # should not crash

run("test_coverage_results_update", test_coverage_results_update)


def test_coverage_results_update_counts():
    r1 = trace.Trace().results()
    r2 = trace.Trace().results()
    r1.update(r2)
    assert isinstance(r1.counts, dict)

run("test_coverage_results_update_counts", test_coverage_results_update_counts)


def test_coverage_results_write_results():
    t = trace.Trace()
    r = t.results()
    r.write_results()   # no-op stub

run("test_coverage_results_write_results", test_coverage_results_write_results)


def test_coverage_results_write_results_kwargs():
    t = trace.Trace()
    r = t.results()
    r.write_results(show_missing=True, summary=False, coverdir=None)

run("test_coverage_results_write_results_kwargs", test_coverage_results_write_results_kwargs)


# ── countfuncs tracking ───────────────────────────────────────────────────────

def test_countfuncs_records_call():
    def my_func():
        return 99
    t = trace.Trace(countfuncs=1)
    t.runfunc(my_func)
    r = t.results()
    # calledfuncs keys are (filename, module, funcname) tuples
    assert len(r.calledfuncs) >= 1

run("test_countfuncs_records_call", test_countfuncs_records_call)


def test_countfuncs_key_is_tuple():
    def my_func():
        pass
    t = trace.Trace(countfuncs=1)
    t.runfunc(my_func)
    r = t.results()
    for key in r.calledfuncs:
        assert isinstance(key, tuple)
        assert len(key) == 3

run("test_countfuncs_key_is_tuple", test_countfuncs_key_is_tuple)


def test_countfuncs_multiple_calls():
    def my_func():
        pass
    t = trace.Trace(countfuncs=1)
    t.runfunc(my_func)
    t.runfunc(my_func)
    r = t.results()
    assert len(r.calledfuncs) >= 1

run("test_countfuncs_multiple_calls", test_countfuncs_multiple_calls)


# ── outfile round-trip ────────────────────────────────────────────────────────

def test_outfile_infile_roundtrip():
    import tempfile, os
    with tempfile.NamedTemporaryFile(suffix=".json", delete=False) as f:
        path = f.name
    try:
        t1 = trace.Trace(countfuncs=1, outfile=path)
        t1.runfunc(lambda: None)
        t1.results()   # writes outfile

        t2 = trace.Trace(countfuncs=1, infile=path)
        r = t2.results()
        assert isinstance(r.calledfuncs, dict)
    finally:
        os.unlink(path)

run("test_outfile_infile_roundtrip", test_outfile_infile_roundtrip)


# ── timing flag ───────────────────────────────────────────────────────────────

def test_timing_flag():
    t = trace.Trace(timing=True)
    result = t.runfunc(lambda: "hello")
    assert result == "hello"

run("test_timing_flag", test_timing_flag)


# ── ignoremods / ignoredirs ────────────────────────────────────────────────────

def test_ignoremods():
    t = trace.Trace(ignoremods=("os", "sys"), ignoredirs=("/usr/lib",))
    result = t.runfunc(lambda: 1 + 1)
    assert result == 2

run("test_ignoremods", test_ignoremods)

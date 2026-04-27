"""Fixture 264 — faulthandler module"""
import faulthandler
import io
import signal as signal_module


# ── 1. import ────────────────────────────────────────────────────────────────
def test_import():
    import faulthandler as fh
    assert fh is not None
    print("test_import: OK")


# ── 2. is_enabled / enable / disable ─────────────────────────────────────────
def test_is_enabled_initial():
    faulthandler.disable()
    assert faulthandler.is_enabled() == False
    print("test_is_enabled_initial: OK")


def test_enable():
    faulthandler.enable()
    assert faulthandler.is_enabled() == True
    faulthandler.disable()
    print("test_enable: OK")


def test_enable_disable():
    faulthandler.enable()
    assert faulthandler.is_enabled() == True
    faulthandler.disable()
    assert faulthandler.is_enabled() == False
    print("test_enable_disable: OK")


def test_enable_twice():
    faulthandler.enable()
    faulthandler.enable()
    assert faulthandler.is_enabled() == True
    faulthandler.disable()
    print("test_enable_twice: OK")


def test_disable_when_not_enabled():
    faulthandler.disable()
    faulthandler.disable()
    assert faulthandler.is_enabled() == False
    print("test_disable_when_not_enabled: OK")


def test_enable_file_kwarg():
    buf = io.StringIO()
    faulthandler.enable(file=buf)
    assert faulthandler.is_enabled() == True
    faulthandler.disable()
    print("test_enable_file_kwarg: OK")


def test_enable_all_threads_false():
    faulthandler.enable(all_threads=False)
    assert faulthandler.is_enabled() == True
    faulthandler.disable()
    print("test_enable_all_threads_false: OK")


# ── 3. dump_traceback ─────────────────────────────────────────────────────────
def test_dump_traceback_default():
    faulthandler.dump_traceback()
    print("test_dump_traceback_default: OK")


def test_dump_traceback_to_stringio():
    buf = io.StringIO()
    faulthandler.dump_traceback(file=buf)
    s = buf.getvalue()
    assert isinstance(s, str)
    assert len(s) > 0
    print("test_dump_traceback_to_stringio: OK")


def test_dump_traceback_has_frame_info():
    buf = io.StringIO()
    faulthandler.dump_traceback(file=buf)
    s = buf.getvalue()
    assert "File" in s
    print("test_dump_traceback_has_frame_info: OK")


def test_dump_traceback_has_filename():
    buf = io.StringIO()
    faulthandler.dump_traceback(file=buf)
    s = buf.getvalue()
    assert "264_faulthandler" in s
    print("test_dump_traceback_has_filename: OK")


def test_dump_traceback_all_threads_false():
    buf = io.StringIO()
    faulthandler.dump_traceback(file=buf, all_threads=False)
    s = buf.getvalue()
    assert isinstance(s, str)
    assert len(s) > 0
    print("test_dump_traceback_all_threads_false: OK")


# ── 4. dump_c_stack ───────────────────────────────────────────────────────────
def test_dump_c_stack():
    buf = io.StringIO()
    faulthandler.dump_c_stack(file=buf)
    print("test_dump_c_stack: OK")


def test_dump_c_stack_default():
    faulthandler.dump_c_stack()
    print("test_dump_c_stack_default: OK")


# ── 5. dump_traceback_later / cancel ─────────────────────────────────────────
def test_dump_traceback_later_cancel():
    buf = io.StringIO()
    faulthandler.dump_traceback_later(60.0, file=buf)
    faulthandler.cancel_dump_traceback_later()
    print("test_dump_traceback_later_cancel: OK")


def test_dump_traceback_later_repeat():
    buf = io.StringIO()
    faulthandler.dump_traceback_later(60.0, repeat=True, file=buf)
    faulthandler.cancel_dump_traceback_later()
    print("test_dump_traceback_later_repeat: OK")


def test_dump_traceback_later_no_exit():
    buf = io.StringIO()
    faulthandler.dump_traceback_later(60.0, exit=False, file=buf)
    faulthandler.cancel_dump_traceback_later()
    print("test_dump_traceback_later_no_exit: OK")


def test_cancel_noop():
    faulthandler.cancel_dump_traceback_later()
    faulthandler.cancel_dump_traceback_later()
    print("test_cancel_noop: OK")


# ── 6. register / unregister ──────────────────────────────────────────────────
def test_register_unregister():
    buf = io.StringIO()
    faulthandler.register(signal_module.SIGUSR1, file=buf)
    result = faulthandler.unregister(signal_module.SIGUSR1)
    assert isinstance(result, bool)
    assert result == True
    print("test_register_unregister: OK")


def test_unregister_not_registered():
    result = faulthandler.unregister(signal_module.SIGUSR2)
    assert result == False
    print("test_unregister_not_registered: OK")


def test_register_all_threads():
    buf = io.StringIO()
    faulthandler.register(signal_module.SIGUSR1, file=buf, all_threads=True)
    result = faulthandler.unregister(signal_module.SIGUSR1)
    assert result == True
    print("test_register_all_threads: OK")


def test_register_chain():
    buf = io.StringIO()
    faulthandler.register(signal_module.SIGUSR1, file=buf, chain=True)
    result = faulthandler.unregister(signal_module.SIGUSR1)
    assert result == True
    print("test_register_chain: OK")


if __name__ == "__main__":
    test_import()
    test_is_enabled_initial()
    test_enable()
    test_enable_disable()
    test_enable_twice()
    test_disable_when_not_enabled()
    test_enable_file_kwarg()
    test_enable_all_threads_false()
    test_dump_traceback_default()
    test_dump_traceback_to_stringio()
    test_dump_traceback_has_frame_info()
    test_dump_traceback_has_filename()
    test_dump_traceback_all_threads_false()
    test_dump_c_stack()
    test_dump_c_stack_default()
    test_dump_traceback_later_cancel()
    test_dump_traceback_later_repeat()
    test_dump_traceback_later_no_exit()
    test_cancel_noop()
    test_register_unregister()
    test_unregister_not_registered()
    test_register_all_threads()
    test_register_chain()

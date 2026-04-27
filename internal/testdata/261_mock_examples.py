"""Fixture 261 — unittest.mock examples (advanced APIs)"""
from unittest.mock import (
    Mock, MagicMock, patch, call, mock_open,
    ANY,
)
import os.path as _ospath


# ── 1. mock_open basic ────────────────────────────────────────────────────────
def test_mock_open_read():
    m = mock_open(read_data="hello world")
    with patch("builtins.open", m):
        with open("somefile.txt") as f:
            data = f.read()
    assert data == "hello world"
    m.assert_called_once_with("somefile.txt")
    print("test_mock_open_read: OK")


def test_mock_open_readline():
    m = mock_open(read_data="line1\nline2\nline3")
    with patch("builtins.open", m):
        with open("f.txt") as f:
            first = f.readline()
            second = f.readline()
    assert first == "line1\n"
    assert second == "line2\n"
    print("test_mock_open_readline: OK")


def test_mock_open_readlines():
    m = mock_open(read_data="a\nb\nc")
    with patch("builtins.open", m):
        with open("f.txt") as f:
            lines = f.readlines()
    assert lines == ["a\n", "b\n", "c"]
    print("test_mock_open_readlines: OK")


def test_mock_open_iteration():
    m = mock_open(read_data="x\ny\nz")
    with patch("builtins.open", m):
        with open("f.txt") as f:
            lines = list(f)
    assert lines == ["x\n", "y\n", "z"]
    print("test_mock_open_iteration: OK")


def test_mock_open_write():
    m = mock_open()
    with patch("builtins.open", m):
        with open("out.txt", "w") as f:
            f.write("data")
    m.assert_called_once_with("out.txt", "w")
    handle = m()
    handle.write.assert_called_once_with("data")
    print("test_mock_open_write: OK")


# ── 2. wraps= ─────────────────────────────────────────────────────────────────
def test_wraps():
    real = lambda x: x * 3
    m = Mock(wraps=real)
    assert m(4) == 12
    assert m(5) == 15
    assert m.call_count == 2
    m.assert_any_call(4)
    print("test_wraps: OK")


def test_wraps_return_value_override():
    real = lambda x: x * 3
    m = Mock(wraps=real, return_value=99)
    # return_value explicitly set: wraps is ignored
    assert m(4) == 99
    print("test_wraps_return_value_override: OK")


# ── 3. spec_set ───────────────────────────────────────────────────────────────
def test_spec_set_attr_access():
    class Spec:
        def method_a(self): pass
        x = 1

    m = Mock(spec_set=Spec)
    _ = m.method_a   # allowed
    try:
        _ = m.nonexistent
        assert False
    except AttributeError:
        pass
    print("test_spec_set_attr_access: OK")


def test_spec_set_attr_set():
    class Spec:
        x = 1
        def method(self): pass

    m = Mock(spec_set=Spec)
    try:
        m.nonexistent = 42
        assert False
    except AttributeError:
        pass
    print("test_spec_set_attr_set: OK")


# ── 4. attach_mock ────────────────────────────────────────────────────────────
def test_attach_mock():
    parent = Mock()
    child = Mock()
    parent.attach_mock(child, "child")
    # After attach, parent.child IS child
    assert parent.child is child
    child(1, 2)
    child.method(3)
    print("test_attach_mock: OK")


# ── 5. patcher.start() / patcher.stop() ──────────────────────────────────────
def test_patcher_start_stop():
    patcher = patch("os.path.exists", return_value=True)
    mock_exists = patcher.start()
    try:
        assert _ospath.exists("/anything") is True
        mock_exists.assert_called_with("/anything")
    finally:
        patcher.stop()
    # After stop, real function restored
    print("test_patcher_start_stop: OK")


# ── 6. Chained call tracking ─────────────────────────────────────────────────
def test_chained_calls():
    m = Mock()
    m.connection.cursor().execute("SELECT 1")
    # cursor() is called once
    assert m.connection.cursor.call_count == 1
    # execute called on cursor()
    cursor_mock = m.connection.cursor()
    cursor_mock.execute.assert_called_with("SELECT 1")
    print("test_chained_calls: OK")


# ── 7. return_value on child mocks ────────────────────────────────────────────
def test_child_return_value():
    m = Mock()
    m.method.return_value = 42
    assert m.method() == 42
    assert m.method(1, 2) == 42
    print("test_child_return_value: OK")


# ── 8. side_effect replacing return_value ────────────────────────────────────
def test_side_effect_replaces_return_value():
    m = Mock(return_value=10, side_effect=lambda: 20)
    assert m() == 20
    print("test_side_effect_replaces_return_value: OK")


# ── 9. magic methods on MagicMock ─────────────────────────────────────────────
def test_magic_mock_int():
    m = MagicMock()
    m.__int__ = Mock(return_value=42)
    assert int(m) == 42
    print("test_magic_mock_int: OK")


def test_magic_mock_add():
    m = MagicMock()
    m.__add__ = Mock(return_value=100)
    result = m + 5
    assert result == 100
    print("test_magic_mock_add: OK")


# ── 10. configure_mock chaining ───────────────────────────────────────────────
def test_configure_mock_chaining():
    m = Mock()
    m.configure_mock(**{"method.return_value": 7})
    # configure_mock with dotted names sets nested attrs
    # Simple case: single attr
    m2 = Mock()
    m2.configure_mock(return_value=5)
    assert m2() == 5
    print("test_configure_mock_chaining: OK")


# ── 11. Mock with no-args call ────────────────────────────────────────────────
def test_mock_no_args():
    m = Mock(return_value="result")
    assert m() == "result"
    m.assert_called_once()
    print("test_mock_no_args: OK")


# ── 12. side_effect list exhaustion → StopIteration ──────────────────────────
def test_side_effect_stopiteration():
    m = Mock(side_effect=[1])
    assert m() == 1
    try:
        m()
        assert False
    except StopIteration:
        pass
    print("test_side_effect_stopiteration: OK")


# ── 13. patch target must exist ───────────────────────────────────────────────
def test_patch_restores_after_exception():
    real_exists = _ospath.exists
    try:
        with patch("os.path.exists", return_value=True):
            raise ValueError("inside patch")
    except ValueError:
        pass
    # Restored even after exception
    assert _ospath.exists is real_exists
    print("test_patch_restores_after_exception: OK")


# ── 14. mock_open context manager calls ───────────────────────────────────────
def test_mock_open_enter_exit():
    m = mock_open(read_data="content")
    with patch("builtins.open", m):
        with open("f.txt") as fh:
            _ = fh.read()
    # __enter__ and __exit__ were called
    handle = m.return_value
    handle.__enter__.assert_called_once()
    handle.__exit__.assert_called()
    print("test_mock_open_enter_exit: OK")


# ── 15. ANY in assert_called_with ─────────────────────────────────────────────
def test_any_in_assert():
    m = Mock()
    m("hello", 42)
    m.assert_called_with(ANY, 42)
    m.assert_called_with("hello", ANY)
    print("test_any_in_assert: OK")


if __name__ == "__main__":
    test_mock_open_read()
    test_mock_open_readline()
    test_mock_open_readlines()
    test_mock_open_iteration()
    test_mock_open_write()
    test_wraps()
    test_wraps_return_value_override()
    test_spec_set_attr_access()
    test_spec_set_attr_set()
    test_attach_mock()
    test_patcher_start_stop()
    test_chained_calls()
    test_child_return_value()
    test_side_effect_replaces_return_value()
    test_magic_mock_int()
    test_magic_mock_add()
    test_configure_mock_chaining()
    test_mock_no_args()
    test_side_effect_stopiteration()
    test_patch_restores_after_exception()
    test_mock_open_enter_exit()
    test_any_in_assert()

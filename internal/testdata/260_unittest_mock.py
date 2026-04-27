"""Fixture 260 — unittest.mock coverage"""
from unittest.mock import (
    Mock, MagicMock, NonCallableMock, patch, call,
    sentinel, ANY, DEFAULT, create_autospec,
)
import unittest.mock as mock_module


# ── 1. Basic Mock ─────────────────────────────────────────────────────────────
def test_basic_mock():
    m = Mock()
    assert not m.called
    assert m.call_count == 0
    result = m(1, 2, key="val")
    assert m.called
    assert m.call_count == 1
    assert m.call_args == call(1, 2, key="val")
    assert m.call_args_list == [call(1, 2, key="val")]
    print("test_basic_mock: OK")


# ── 2. return_value and side_effect ──────────────────────────────────────────
def test_return_value():
    m = Mock(return_value=42)
    assert m() == 42
    assert m(1, 2) == 42
    print("test_return_value: OK")


def test_side_effect_callable():
    m = Mock(side_effect=lambda x: x * 2)
    assert m(3) == 6
    assert m(5) == 10
    print("test_side_effect_callable: OK")


def test_side_effect_exception():
    m = Mock(side_effect=ValueError("boom"))
    try:
        m()
        assert False, "should have raised"
    except ValueError as e:
        assert str(e) == "boom"
    print("test_side_effect_exception: OK")


def test_side_effect_iterable():
    m = Mock(side_effect=[1, 2, 3])
    assert m() == 1
    assert m() == 2
    assert m() == 3
    try:
        m()
        assert False, "should have raised StopIteration"
    except StopIteration:
        pass
    print("test_side_effect_iterable: OK")


# ── 3. Attribute access creates child mocks ───────────────────────────────────
def test_child_mocks():
    m = Mock()
    child = m.some_attr
    assert isinstance(child, Mock)
    child2 = m.some_attr
    assert child is child2  # same child each time
    print("test_child_mocks: OK")


# ── 4. call tracking ─────────────────────────────────────────────────────────
def test_call_tracking():
    m = Mock()
    m(1)
    m(2)
    m(3)
    assert m.call_count == 3
    assert m.call_args_list == [call(1), call(2), call(3)]
    m.reset_mock()
    assert m.call_count == 0
    assert not m.called
    assert m.call_args_list == []
    print("test_call_tracking: OK")


# ── 5. assert helpers ─────────────────────────────────────────────────────────
def test_assert_called():
    m = Mock()
    try:
        m.assert_called()
        assert False
    except AssertionError:
        pass
    m()
    m.assert_called()
    print("test_assert_called: OK")


def test_assert_called_once():
    m = Mock()
    m()
    m.assert_called_once()
    m()
    try:
        m.assert_called_once()
        assert False
    except AssertionError:
        pass
    print("test_assert_called_once: OK")


def test_assert_called_with():
    m = Mock()
    m(1, 2, x=3)
    m.assert_called_with(1, 2, x=3)
    try:
        m.assert_called_with(1, 2)
        assert False
    except AssertionError:
        pass
    print("test_assert_called_with: OK")


def test_assert_called_once_with():
    m = Mock()
    m("hello")
    m.assert_called_once_with("hello")
    m("hello")
    try:
        m.assert_called_once_with("hello")
        assert False
    except AssertionError:
        pass
    print("test_assert_called_once_with: OK")


def test_assert_any_call():
    m = Mock()
    m(1)
    m(2)
    m(3)
    m.assert_any_call(2)
    try:
        m.assert_any_call(99)
        assert False
    except AssertionError:
        pass
    print("test_assert_any_call: OK")


def test_assert_not_called():
    m = Mock()
    m.assert_not_called()
    m()
    try:
        m.assert_not_called()
        assert False
    except AssertionError:
        pass
    print("test_assert_not_called: OK")


def test_assert_has_calls():
    m = Mock()
    m(1)
    m(2)
    m(3)
    m.assert_has_calls([call(1), call(2)])
    m.assert_has_calls([call(2), call(3)])
    try:
        m.assert_has_calls([call(1), call(3)])
        assert False
    except AssertionError:
        pass
    print("test_assert_has_calls: OK")


# ── 6. configure_mock ─────────────────────────────────────────────────────────
def test_configure_mock():
    m = Mock()
    m.configure_mock(return_value=99, name="mymock")
    assert m() == 99
    print("test_configure_mock: OK")


# ── 7. MagicMock ─────────────────────────────────────────────────────────────
def test_magic_mock_str():
    m = MagicMock()
    s = str(m)
    assert isinstance(s, str)
    print("test_magic_mock_str: OK")


def test_magic_mock_len():
    m = MagicMock()
    m.__len__ = Mock(return_value=5)
    assert len(m) == 5
    print("test_magic_mock_len: OK")


def test_magic_mock_iter():
    m = MagicMock()
    m.__iter__ = Mock(return_value=iter([10, 20, 30]))
    result = list(m)
    assert result == [10, 20, 30]
    print("test_magic_mock_iter: OK")


def test_magic_mock_context_manager():
    m = MagicMock()
    m.__enter__ = Mock(return_value="resource")
    m.__exit__ = Mock(return_value=False)
    with m as r:
        assert r == "resource"
    m.__enter__.assert_called_once()
    m.__exit__.assert_called_once()
    print("test_magic_mock_context_manager: OK")


def test_magic_mock_contains():
    m = MagicMock()
    m.__contains__ = Mock(return_value=True)
    assert "x" in m
    print("test_magic_mock_contains: OK")


# ── 8. NonCallableMock ────────────────────────────────────────────────────────
def test_non_callable_mock():
    m = NonCallableMock()
    try:
        m()
        assert False
    except TypeError:
        pass
    child = m.attr
    assert isinstance(child, NonCallableMock)
    print("test_non_callable_mock: OK")


# ── 9. call object ────────────────────────────────────────────────────────────
def test_call_object():
    c = call(1, 2, x=3)
    assert c == call(1, 2, x=3)
    assert c != call(1, 2)
    print("test_call_object: OK")


def test_call_repr():
    c = call(1, "hello")
    r = repr(c)
    assert "call" in r
    print("test_call_repr: OK")


# ── 10. sentinel ─────────────────────────────────────────────────────────────
def test_sentinel():
    s = sentinel.MISSING
    s2 = sentinel.MISSING
    assert s is s2
    s3 = sentinel.DEFAULT
    assert s3 is not s
    assert repr(s) == "sentinel.MISSING"
    print("test_sentinel: OK")


# ── 11. ANY ──────────────────────────────────────────────────────────────────
def test_any():
    assert ANY == 42
    assert ANY == "hello"
    assert ANY == [1, 2, 3]
    assert ANY == ANY
    m = Mock()
    m(1, 2)
    m.assert_called_with(ANY, ANY)
    print("test_any: OK")


# ── 12. patch context manager ─────────────────────────────────────────────────
import os.path as _ospath

def test_patch_context_manager():
    with patch("os.path.exists") as mock_exists:
        mock_exists.return_value = True
        result = _ospath.exists("/fake/path")
        assert result is True
        mock_exists.assert_called_once_with("/fake/path")
    # after patch, real function restored
    print("test_patch_context_manager: OK")


def test_patch_new():
    with patch("os.path.exists", return_value=False) as mock_exists:
        assert _ospath.exists("/anything") is False
    print("test_patch_new: OK")


# ── 13. patch.object ─────────────────────────────────────────────────────────
class MyClass:
    def method(self):
        return "real"


def test_patch_object():
    obj = MyClass()
    with patch.object(MyClass, "method", return_value="mocked") as mock_method:
        assert obj.method() == "mocked"
        mock_method.assert_called_once()
    assert obj.method() == "real"
    print("test_patch_object: OK")


# ── 14. patch.dict ────────────────────────────────────────────────────────────
def test_patch_dict():
    d = {"key": "original"}
    with patch.dict(d, {"key": "patched", "new": "value"}):
        assert d["key"] == "patched"
        assert d["new"] == "value"
    assert d["key"] == "original"
    assert "new" not in d
    print("test_patch_dict: OK")


def test_patch_dict_clear():
    d = {"a": 1, "b": 2}
    with patch.dict(d, {"c": 3}, clear=True):
        assert d == {"c": 3}
    assert d == {"a": 1, "b": 2}
    print("test_patch_dict_clear: OK")


# ── 15. patch as decorator ───────────────────────────────────────────────────
@patch("os.path.exists", return_value=True)
def test_patch_decorator(mock_exists):
    assert _ospath.exists("/test") is True
    mock_exists.assert_called_with("/test")
    print("test_patch_decorator: OK")


# ── 16. create_autospec ──────────────────────────────────────────────────────
def test_create_autospec():
    def real_func(a, b, c=0):
        return a + b + c
    mock_func = create_autospec(real_func, return_value=10)
    assert mock_func(1, 2) == 10
    mock_func.assert_called_once_with(1, 2)
    print("test_create_autospec: OK")


# ── 17. Mock with spec ────────────────────────────────────────────────────────
def test_mock_with_spec():
    class Spec:
        def method_a(self):
            pass
        x = 1

    m = Mock(spec=Spec)
    # valid attribute
    _ = m.method_a
    # invalid attribute raises AttributeError
    try:
        _ = m.nonexistent
        assert False
    except AttributeError:
        pass
    print("test_mock_with_spec: OK")


# ── 18. Mock name ─────────────────────────────────────────────────────────────
def test_mock_name():
    m = Mock(name="my_mock")
    r = repr(m)
    assert "my_mock" in r
    print("test_mock_name: OK")


# ── 19. method access ────────────────────────────────────────────────────────
def test_method_calls():
    m = Mock()
    m.method(1)
    m.method(2)
    assert m.method.call_count == 2
    assert m.method.call_args_list == [call(1), call(2)]
    print("test_method_calls: OK")


# ── 20. patch multiple ────────────────────────────────────────────────────────
def test_patch_multiple():
    with patch("os.path.exists", return_value=True), \
         patch("os.path.isfile", return_value=False):
        assert _ospath.exists("/x") is True
        assert _ospath.isfile("/x") is False
    print("test_patch_multiple: OK")


if __name__ == "__main__":
    test_basic_mock()
    test_return_value()
    test_side_effect_callable()
    test_side_effect_exception()
    test_side_effect_iterable()
    test_child_mocks()
    test_call_tracking()
    test_assert_called()
    test_assert_called_once()
    test_assert_called_with()
    test_assert_called_once_with()
    test_assert_any_call()
    test_assert_not_called()
    test_assert_has_calls()
    test_configure_mock()
    test_magic_mock_str()
    test_magic_mock_len()
    test_magic_mock_iter()
    test_magic_mock_context_manager()
    test_magic_mock_contains()
    test_non_callable_mock()
    test_call_object()
    test_call_repr()
    test_sentinel()
    test_any()
    test_patch_context_manager()
    test_patch_new()
    test_patch_object()
    test_patch_dict()
    test_patch_dict_clear()
    test_patch_decorator()  # decorator auto-injects mock_exists arg
    test_create_autospec()
    test_mock_with_spec()
    test_mock_name()
    test_method_calls()
    test_patch_multiple()

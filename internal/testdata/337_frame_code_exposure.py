"""v0.0.337 frame & code exposure — sys._getframe, frame attrs,
extended code attrs."""
import sys
import types
import inspect


# ── 1. sys._getframe is exposed
fr = sys._getframe()
assert fr is not None
assert isinstance(fr, types.FrameType)
print("getframe ok")


# ── 2. Frame attrs
def f():
    return sys._getframe()

inner = f()
assert inner.f_code.co_name == 'f'
assert isinstance(inner.f_code, types.CodeType)
assert isinstance(inner.f_globals, dict)
assert inner.f_globals['__name__'] == __name__
# PEP 667 / 3.13+ returns FrameLocalsProxy; we ship a dict view for now.
# Just require it has dict-like .keys() / [].
assert hasattr(inner.f_locals, 'keys')
assert inner.f_lasti >= 0
assert inner.f_lineno > 0
print("frame attrs ok")


# ── 3. f_back chain
def g():
    return sys._getframe(1)

fr_g = g()
assert fr_g.f_code.co_name == '<module>'
def h():
    return sys._getframe()
def parent():
    return h()
fr_h = parent()
assert fr_h.f_code.co_name == 'h'
assert fr_h.f_back.f_code.co_name == 'parent'
print("f_back ok")


# ── 4. inspect.currentframe
fr2 = inspect.currentframe()
assert isinstance(fr2, types.FrameType)
print("currentframe ok")


# ── 5. inspect.stack from inside a function
def stack_test():
    s = inspect.stack()
    return [fi.function for fi in s]

names = stack_test()
assert 'stack_test' in names, names
assert '<module>' in names
print("stack ok")


# ── 6. Extended code attributes
def with_args(a, b, *, c=1, **kw):
    return a + b

co = with_args.__code__
assert co.co_name == 'with_args'
assert co.co_qualname == 'with_args'
assert co.co_argcount == 2
assert co.co_kwonlyargcount == 1
assert co.co_filename.endswith('337_frame_code_exposure.py')
assert isinstance(co.co_consts, tuple)
assert isinstance(co.co_names, tuple)
assert isinstance(co.co_varnames, tuple)
assert 'a' in co.co_varnames and 'b' in co.co_varnames
assert isinstance(co.co_flags, int)
print("code attrs ok")


# ── 7. co_lines() iterator yields (start, end, line) triples
spans = list(co.co_lines())
assert len(spans) >= 1
for s in spans:
    assert len(s) == 3
print("co_lines ok")


# ── 8. f_locals reflects current locals for a function frame
def locals_test():
    x = 42
    y = 'hi'
    fr = sys._getframe()
    keys = sorted(fr.f_locals.keys())
    assert 'x' in keys
    assert 'y' in keys

locals_test()
print("f_locals ok")


print("ok")

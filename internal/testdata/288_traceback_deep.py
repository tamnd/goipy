"""Comprehensive traceback test — covers all public API from the Python docs."""
import traceback
import io


# ── FrameSummary: attrs ───────────────────────────────────────────────────────

fs = traceback.FrameSummary("test.py", 42, "myfunc")
print(fs.filename)    # test.py
print(fs.lineno)      # 42
print(fs.name)        # myfunc
print(repr(fs.line))  # '' (linecache returns empty when file not found)

# unpack (4-tuple: filename, lineno, name, line)
fn, ln, nm, line_text = fs
print(fn)             # test.py
print(ln)             # 42
print(nm)             # myfunc
print(repr(line_text))  # '' (empty string)

# subscript
print(fs[0])          # test.py
print(fs[1])          # 42
print(fs[2])          # myfunc

# repr contains filename/lineno/name
r = repr(fs)
print("test.py" in r)   # True
print("42" in r)         # True
print("myfunc" in r)     # True


# ── StackSummary.from_list ────────────────────────────────────────────────────

pairs = [("a.py", 1, "fn1", "x = 1"), ("b.py", 2, "fn2", "y = 2")]
ss = traceback.StackSummary.from_list(pairs)
print(type(ss).__name__)   # StackSummary
print(len(ss))             # 2

# iteration
for frame in ss:
    print(frame.filename, frame.lineno, frame.name)

# subscript
print(ss[0].filename)   # a.py
print(ss[1].filename)   # b.py

# format
lines = ss.format()
print(len(lines))                    # 2
print('"a.py"' in lines[0])         # True (Python uses quoted filenames)
print("x = 1" in lines[0])          # True

# format_frame_summary
ffs = ss.format_frame_summary(ss[0])
print("a.py" in ffs)    # True


# ── format_list with tuples ───────────────────────────────────────────────────

formatted = traceback.format_list(pairs)
print(len(formatted))              # 2
print("a.py" in formatted[0])     # True


# ── extract_stack ─────────────────────────────────────────────────────────────

stack = traceback.extract_stack()
print(isinstance(stack, traceback.StackSummary))   # True
print(len(stack) >= 1)                              # True


# ── format_stack ─────────────────────────────────────────────────────────────

fstack = traceback.format_stack()
print(isinstance(fstack, list))   # True


# ── format_exception_only: two-arg form ──────────────────────────────────────

try:
    raise ValueError("bad value")
except ValueError as e:
    lines = traceback.format_exception_only(type(e), e)
    print(len(lines) >= 1)           # True
    print("ValueError" in lines[0])  # True
    print("bad value" in lines[0])   # True


# ── format_exception_only: one-arg form (Python 3.10+) ───────────────────────

try:
    raise TypeError("wrong type")
except TypeError as e:
    lines = traceback.format_exception_only(e)
    print("TypeError" in lines[0])   # True
    print("wrong type" in lines[0])  # True


# ── format_exc inside except block ───────────────────────────────────────────

try:
    raise RuntimeError("oops")
except RuntimeError:
    s = traceback.format_exc()
    print(isinstance(s, str))        # True
    print(len(s) > 0)                # True
    print("RuntimeError" in s)       # True


# ── format_exception: one-arg form ───────────────────────────────────────────

try:
    raise KeyError("missing")
except KeyError as e:
    lines = traceback.format_exception(e)
    text = "".join(lines)
    print(isinstance(text, str))   # True
    print("KeyError" in text)      # True


# ── TracebackException.from_exception ────────────────────────────────────────

try:
    raise ValueError("test error")
except ValueError as e:
    tbe = traceback.TracebackException.from_exception(e)
    print(tbe.exc_type_str)                          # ValueError
    print(type(tbe.stack).__name__)                  # StackSummary
    lines = list(tbe.format_exception_only())
    print(len(lines) >= 1)                           # True
    print("ValueError" in lines[0])                  # True
    print("test error" in lines[0])                  # True


# ── TracebackException attrs ──────────────────────────────────────────────────

try:
    raise IndexError("out of range")
except IndexError as e:
    tbe = traceback.TracebackException.from_exception(e)
    print(tbe.__cause__ is None)              # True
    print(tbe.__context__ is None)            # True
    print(tbe.__suppress_context__ == False)  # True
    print(tbe.__notes__ is None)              # True


# ── TracebackException.format() ──────────────────────────────────────────────

try:
    raise AttributeError("no attr")
except AttributeError as e:
    tbe = traceback.TracebackException.from_exception(e)
    lines = list(tbe.format())
    text = "".join(lines)
    print("AttributeError" in text)   # True


# ── TracebackException.print(file=buf) ───────────────────────────────────────

try:
    raise NameError("undef")
except NameError as e:
    tbe = traceback.TracebackException.from_exception(e)
    buf = io.StringIO()
    tbe.print(file=buf)
    out = buf.getvalue()
    print("NameError" in out)   # True


# ── chained exception: __cause__ (__suppress_context__ = True) ───────────────

try:
    try:
        raise ValueError("original")
    except ValueError as orig:
        raise TypeError("chained") from orig
except TypeError as te:
    tbe = traceback.TracebackException.from_exception(te)
    print(tbe.__cause__ is not None)   # True
    print(tbe.__suppress_context__)     # True


# ── chained exception: __context__ (implicit) ────────────────────────────────

try:
    try:
        raise ValueError("ctx")
    except ValueError:
        raise TypeError("new error")
except TypeError as te:
    tbe = traceback.TracebackException.from_exception(te)
    print(tbe.__context__ is not None)   # True


# ── print_list(list, file=buf) ────────────────────────────────────────────────

buf = io.StringIO()
pairs2 = [("x.py", 5, "foo", "bar()")]
traceback.print_list(pairs2, file=buf)
out2 = buf.getvalue()
print("x.py" in out2)   # True


# ── print_exception(exc, file=buf) ───────────────────────────────────────────

try:
    raise OSError("io error")
except OSError as e:
    buf3 = io.StringIO()
    traceback.print_exception(e, file=buf3)
    out3 = buf3.getvalue()
    print("OSError" in out3)   # True


# ── clear_frames is a no-op ───────────────────────────────────────────────────

traceback.clear_frames(None)
print("clear_frames ok")


# ── walk_stack / walk_tb return iterables ────────────────────────────────────

ws = traceback.walk_stack(None)
print(ws is not None)   # True

wt = traceback.walk_tb(None)
print(wt is not None)   # True


print("done")

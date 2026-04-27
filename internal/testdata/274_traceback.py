import traceback

# FrameSummary
fs = traceback.FrameSummary("test.py", 42, "myfunc")
print(fs.filename)   # test.py
print(fs.lineno)     # 42
print(fs.name)       # myfunc

# StackSummary.from_list
pairs = [("a.py", 1, "fn1", "x = 1"), ("b.py", 2, "fn2", "y = 2")]
ss = traceback.StackSummary.from_list(pairs)
lines = ss.format()
print(len(lines) >= 2)  # True

# format_list
formatted = traceback.format_list(pairs)
print(len(formatted) >= 2)  # True

# format_exception_only with an exception
try:
    raise ValueError("bad value")
except ValueError as e:
    lines = traceback.format_exception_only(type(e), e)
    print(len(lines) >= 1)   # True
    text = lines[0]
    print("ValueError" in text)  # True

# format_exc inside except block
try:
    raise RuntimeError("oops")
except RuntimeError:
    s = traceback.format_exc()
    print(len(s) > 0)  # True

# extract_stack returns a StackSummary
ss2 = traceback.extract_stack()
print(isinstance(ss2, traceback.StackSummary))  # True

# TracebackException
try:
    raise TypeError("wrong type")
except TypeError as e:
    te = traceback.TracebackException.from_exception(e)
    lines = list(te.format_exception_only())
    print(len(lines) >= 1)    # True
    print("TypeError" in lines[0])  # True

print("done")

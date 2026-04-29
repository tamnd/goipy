# sys.settrace / sys.setprofile + per-frame f_trace

import sys

print("# section 1: settrace round-trip")

print(sys.gettrace())


def t1(frame, event, arg):
    return t1


sys.settrace(t1)
print(sys.gettrace() is t1)
sys.settrace(None)
print(sys.gettrace())


print("# section 2: call/line/return event sequence")

events = []


def tracer(frame, event, arg):
    events.append((frame.f_code.co_name, event))
    return tracer


def f(x):
    a = x + 1
    b = a * 2
    return b


sys.settrace(tracer)
f(5)
sys.settrace(None)
# Filter out anything from the tracer's own teardown frame; only
# events for f() and the driver's <module> matter for parity.
fevents = [e for e in events if e[0] == "f"]
print(fevents)


print("# section 3: per-frame f_trace replacement")

calls = []


def per_frame_trace(frame, event, arg):
    calls.append(("local", frame.f_code.co_name, event))
    return per_frame_trace


def call_handler(frame, event, arg):
    calls.append(("global", frame.f_code.co_name, event))
    if frame.f_code.co_name == "g":
        return per_frame_trace
    return call_handler


def g():
    return 42


sys.settrace(call_handler)
g()
sys.settrace(None)
gcalls = [c for c in calls if c[1] == "g"]
print(gcalls)


print("# section 4: setprofile fires call/return without line")

prof_events = []


def prof(frame, event, arg):
    prof_events.append((frame.f_code.co_name, event))


def h(x):
    return x + 1


sys.setprofile(prof)
h(10)
sys.setprofile(None)
hprof = [e for e in prof_events if e[0] == "h"]
# Profile event order: call then return. No line events.
print(hprof)
print(any(e[1] == "line" for e in prof_events))


print("# section 5: disabling trace mid-execution")

mid_events = []


def mid_tracer(frame, event, arg):
    mid_events.append((frame.f_code.co_name, event))
    return mid_tracer


def runner():
    sys.settrace(None)  # disable from inside traced frame
    return 1


sys.settrace(mid_tracer)
runner()
sys.settrace(None)
runner_events = [e for e in mid_events if e[0] == "runner"]
# At minimum we get the 'call' event before the tracer was cleared.
print(runner_events[0])
print("done")

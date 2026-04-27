import sys.monitoring as monitoring

# Tool ID constants
print(monitoring.DEBUGGER_ID == 0)   # True
print(monitoring.COVERAGE_ID == 1)   # True
print(monitoring.PROFILER_ID == 2)   # True
print(monitoring.OPTIMIZER_ID == 5)  # True

# events namespace
print(monitoring.events.NO_EVENTS == 0)   # True
print(monitoring.events.CALL > 0)         # True
print(monitoring.events.LINE > 0)         # True
print(monitoring.events.PY_RETURN > 0)    # True

# use_tool_id / free_tool_id
monitoring.use_tool_id(monitoring.COVERAGE_ID, "test_tool")
monitoring.free_tool_id(monitoring.COVERAGE_ID)

# set_events / get_events
monitoring.use_tool_id(monitoring.DEBUGGER_ID, "dbg")
monitoring.set_events(monitoring.DEBUGGER_ID, monitoring.events.LINE)
ev = monitoring.get_events(monitoring.DEBUGGER_ID)
print(ev == monitoring.events.LINE)  # True

# set_events to 0
monitoring.set_events(monitoring.DEBUGGER_ID, monitoring.events.NO_EVENTS)
print(monitoring.get_events(monitoring.DEBUGGER_ID) == 0)  # True

# register_callback returns old callback (None initially)
old = monitoring.register_callback(monitoring.DEBUGGER_ID, monitoring.events.LINE, None)
print(old is None or old == monitoring.MISSING)  # True

def my_callback(code, offset):
    pass

monitoring.register_callback(monitoring.DEBUGGER_ID, monitoring.events.LINE, my_callback)
old2 = monitoring.register_callback(monitoring.DEBUGGER_ID, monitoring.events.LINE, None)
print(old2 is my_callback)  # True

monitoring.free_tool_id(monitoring.DEBUGGER_ID)

print("done")

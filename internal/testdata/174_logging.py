"""Tests for the logging module."""
import io
import logging
import os
import sys
import tempfile

# --- level constants ---
print(logging.DEBUG == 10)      # True
print(logging.INFO == 20)       # True
print(logging.WARNING == 30)    # True
print(logging.ERROR == 40)      # True
print(logging.CRITICAL == 50)   # True
print(logging.WARN == 30)       # True
print(logging.FATAL == 50)      # True
print(logging.NOTSET == 0)      # True

# --- getLevelName ---
print(logging.getLevelName(10))  # DEBUG
print(logging.getLevelName(20))  # INFO
print(logging.getLevelName(30))  # WARNING
print(logging.getLevelName(40))  # ERROR
print(logging.getLevelName(50))  # CRITICAL
print(logging.getLevelName(0))   # NOTSET

# int lookup by name
print(logging.getLevelName("DEBUG") == 10)    # True
print(logging.getLevelName("WARNING") == 30)  # True

# --- addLevelName ---
logging.addLevelName(25, "VERBOSE")
print(logging.getLevelName(25))   # VERBOSE
print(logging.getLevelName("VERBOSE") == 25)  # True

# --- getLevelNamesMapping ---
m = logging.getLevelNamesMapping()
print(m["DEBUG"] == 10)    # True
print(m["CRITICAL"] == 50) # True

# --- basicConfig + root logger shortcuts ---
# Use StringIO to capture output
buf = io.StringIO()
logging.basicConfig(stream=buf, level=logging.DEBUG,
                    format="%(levelname)s:%(name)s:%(message)s")
logging.debug("dbg msg")
logging.info("info msg")
logging.warning("warn msg")
logging.error("err msg")
logging.critical("crit msg")
out = buf.getvalue()
lines = [l for l in out.strip().split("\n") if l]
print(lines[0])  # DEBUG:root:dbg msg
print(lines[1])  # INFO:root:info msg
print(lines[2])  # WARNING:root:warn msg
print(lines[3])  # ERROR:root:err msg
print(lines[4])  # CRITICAL:root:crit msg

# --- getLogger identity ---
l1 = logging.getLogger("myapp")
l2 = logging.getLogger("myapp")
print(l1 is l2)   # True

# --- getLogger hierarchy (parent is root) ---
child = logging.getLogger("myapp.sub")
print(child.name)              # myapp.sub
print(child.parent.name)       # myapp (or root if myapp not created first)

# --- logger setLevel / isEnabledFor / getEffectiveLevel ---
logger = logging.getLogger("testlevel")
logger.setLevel(logging.WARNING)
print(logger.isEnabledFor(logging.ERROR))    # True
print(logger.isEnabledFor(logging.DEBUG))    # False
print(logger.getEffectiveLevel() == logging.WARNING)  # True

# --- propagate ---
buf2 = io.StringIO()
parent_logger = logging.getLogger("parent")
parent_logger.setLevel(logging.DEBUG)
sh = logging.StreamHandler(stream=buf2)
sh.setFormatter(logging.Formatter("%(levelname)s:%(name)s:%(message)s"))
parent_logger.addHandler(sh)

child_logger = logging.getLogger("parent.child")
child_logger.setLevel(logging.DEBUG)
child_logger.info("child info")
val2 = buf2.getvalue().strip()
print("parent.child" in val2 or "child" in val2)  # True

# --- propagate=False ---
buf3 = io.StringIO()
isolated = logging.getLogger("isolated")
isolated.setLevel(logging.DEBUG)
isolated.propagate = False
sh2 = logging.StreamHandler(stream=buf3)
sh2.setFormatter(logging.Formatter("%(message)s"))
isolated.addHandler(sh2)
isolated.info("isolated msg")
print(buf3.getvalue().strip())  # isolated msg

# --- disable ---
buf4 = io.StringIO()
logging.basicConfig(stream=buf4, level=logging.DEBUG,
                    format="%(message)s", force=True)
logging.disable(logging.WARNING)
logging.debug("should be suppressed")
logging.info("should be suppressed")
logging.error("should appear")
out4 = buf4.getvalue().strip()
print("suppressed" not in out4)  # True
print("should appear" in out4)   # True
logging.disable(logging.NOTSET)  # re-enable

# --- StreamHandler with StringIO ---
buf5 = io.StringIO()
sh3 = logging.StreamHandler(stream=buf5)
fmt = logging.Formatter("%(levelname)s - %(message)s")
sh3.setFormatter(fmt)
sh3.setLevel(logging.DEBUG)
tmplogger = logging.getLogger("tmplogger")
tmplogger.setLevel(logging.DEBUG)
tmplogger.propagate = False
tmplogger.addHandler(sh3)
tmplogger.warning("test warning")
tmplogger.error("test error %s", "arg")
lines5 = [l for l in buf5.getvalue().strip().split("\n") if l]
print(lines5[0])  # WARNING - test warning
print(lines5[1])  # ERROR - test error arg

# --- FileHandler ---
tmpfile = tempfile.mktemp(suffix=".log")
fh = logging.FileHandler(tmpfile, mode="w")
fh.setFormatter(logging.Formatter("%(levelname)s %(message)s"))
fh.setLevel(logging.DEBUG)
flogger = logging.getLogger("flogger")
flogger.setLevel(logging.DEBUG)
flogger.propagate = False
flogger.addHandler(fh)
flogger.info("file info")
flogger.error("file error")
fh.close()
with open(tmpfile) as f:
    flines = [l.strip() for l in f.readlines()]
print(flines[0])  # INFO file info
print(flines[1])  # ERROR file error
os.remove(tmpfile)

# --- NullHandler ---
nh = logging.NullHandler()
buf6 = io.StringIO()
nlogger = logging.getLogger("nlogger")
nlogger.setLevel(logging.DEBUG)
nlogger.propagate = False
nlogger.addHandler(nh)
nlogger.info("should be discarded")
print(buf6.getvalue() == "")  # True

# --- Formatter ---
f1 = logging.Formatter("%(levelname)s:%(message)s")
rec = logging.makeLogRecord({"levelname": "INFO", "message": "hello", "name": "test",
                              "levelno": 20, "pathname": "", "filename": "",
                              "module": "", "funcName": "", "lineno": 0,
                              "created": 0.0, "msecs": 0.0, "relativeCreated": 0.0,
                              "thread": 0, "threadName": "MainThread",
                              "process": 1, "processName": "MainProcess",
                              "exc_info": None, "exc_text": None, "stack_info": None,
                              "args": None, "msg": "hello"})
print(f1.format(rec))  # INFO:hello

# --- hasHandlers ---
hlogger = logging.getLogger("hlogger")
hlogger.propagate = False
print(hlogger.hasHandlers())  # False
hlogger.addHandler(logging.NullHandler())
print(hlogger.hasHandlers())  # True

# --- getChild ---
parent2 = logging.getLogger("parent2")
kid = parent2.getChild("kid")
print(kid.name)  # parent2.kid

# --- log() with explicit level ---
buf7 = io.StringIO()
llogger = logging.getLogger("llogger")
llogger.setLevel(logging.DEBUG)
llogger.propagate = False
sh4 = logging.StreamHandler(stream=buf7)
sh4.setFormatter(logging.Formatter("%(levelname)s:%(message)s"))
llogger.addHandler(sh4)
llogger.log(logging.INFO, "explicit level")
print(buf7.getvalue().strip())  # INFO:explicit level

# --- setLogRecordFactory / getLogRecordFactory ---
orig = logging.getLogRecordFactory()
logging.setLogRecordFactory(orig)  # no-op round-trip
print(logging.getLogRecordFactory() is orig)  # True

# --- captureWarnings (stub) ---
logging.captureWarnings(True)   # should not raise
logging.captureWarnings(False)
print(True)  # True

# --- Filter ---
buf8 = io.StringIO()
filt_logger = logging.getLogger("filtapp")
filt_logger.setLevel(logging.DEBUG)
filt_logger.propagate = False
sh5 = logging.StreamHandler(stream=buf8)
sh5.setFormatter(logging.Formatter("%(message)s"))
f2 = logging.Filter("filtapp.sub")
sh5.addFilter(f2)
filt_logger.addHandler(sh5)
# Messages from "filtapp" should pass through the filter for "filtapp.sub"?
# Actually Filter("filtapp.sub") only passes records from filtapp.sub.*
# So direct filtapp messages get blocked.
sub_logger = logging.getLogger("filtapp.sub")
sub_logger.setLevel(logging.DEBUG)
sub_logger.propagate = True
sub_logger.info("sub message")
other_logger = logging.getLogger("filtapp.other")
other_logger.setLevel(logging.DEBUG)
other_logger.propagate = True
other_logger.info("other message")
val8 = buf8.getvalue().strip().split("\n")
print(any("sub message" in l for l in val8))    # True
print(not any("other message" in l for l in val8))  # True

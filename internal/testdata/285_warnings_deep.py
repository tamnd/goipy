"""Comprehensive warnings test — covers all public API from the Python docs."""
import warnings
import io
import sys
from warnings import (
    warn, warn_explicit, showwarning, formatwarning,
    filterwarnings, simplefilter, resetwarnings,
    catch_warnings,
    WarningMessage,
)


# ── Warning category hierarchy ───────────────────────────────────────────────

def test_hierarchy():
    print(issubclass(Warning, Exception))            # True
    print(issubclass(UserWarning, Warning))          # True
    print(issubclass(DeprecationWarning, Warning))   # True
    print(issubclass(PendingDeprecationWarning, Warning))  # True
    print(issubclass(RuntimeWarning, Warning))       # True
    print(issubclass(SyntaxWarning, Warning))        # True
    print(issubclass(ResourceWarning, Warning))      # True
    print(issubclass(FutureWarning, Warning))        # True
    print(issubclass(ImportWarning, Warning))        # True
    print(issubclass(UnicodeWarning, Warning))       # True
    print(issubclass(BytesWarning, Warning))         # True
    print('hierarchy ok')

test_hierarchy()


# ── formatwarning ─────────────────────────────────────────────────────────────

def test_formatwarning():
    s = formatwarning("oops", UserWarning, "script.py", 42)
    print(s.strip())   # script.py:42: UserWarning: oops
    print('formatwarning ok')

test_formatwarning()


# ── showwarning writes to file ────────────────────────────────────────────────

def test_showwarning():
    buf = io.StringIO()
    showwarning("hello", UserWarning, "f.py", 10, file=buf)
    print(buf.getvalue().strip())   # f.py:10: UserWarning: hello
    print('showwarning ok')

test_showwarning()


# ── warn() with catch_warnings(record=True) ──────────────────────────────────

def test_warn_record():
    with catch_warnings(record=True) as log:
        simplefilter("always")
        warn("basic message")
        warn("second message", DeprecationWarning)
    print(len(log))                                 # 2
    print(log[0].message)                           # basic message
    print(log[0].category.__name__)                 # UserWarning
    print(log[1].category.__name__)                 # DeprecationWarning
    print('warn_record ok')

test_warn_record()


# ── warn() action: error ──────────────────────────────────────────────────────

def test_warn_error():
    with catch_warnings():
        filterwarnings("error")
        try:
            warn("this should raise")
            print('no raise')
        except UserWarning as e:
            print('caught UserWarning')   # caught UserWarning
    print('warn_error ok')

test_warn_error()


# ── warn() action: ignore ─────────────────────────────────────────────────────

def test_warn_ignore():
    buf = io.StringIO()
    with catch_warnings(record=True) as log:
        simplefilter("ignore")
        warn("ignored message")
    print(len(log))   # 0
    print('warn_ignore ok')

test_warn_ignore()


# ── warn() action: always ─────────────────────────────────────────────────────

def test_warn_always():
    with catch_warnings(record=True) as log:
        simplefilter("always")
        warn("repeat")
        warn("repeat")
        warn("repeat")
    print(len(log))   # 3 — always shows
    print('warn_always ok')

test_warn_always()


# ── warn() action: once ───────────────────────────────────────────────────────

def test_warn_once():
    with catch_warnings(record=True) as log:
        simplefilter("once")
        warn("unique message")
        warn("unique message")   # same text → skip
        warn("different message")
    print(len(log))   # 2
    print('warn_once ok')

test_warn_once()


# ── resetwarnings ─────────────────────────────────────────────────────────────

def test_resetwarnings():
    with catch_warnings():
        filterwarnings("error")
        resetwarnings()
        # after reset no error filter → just record
        with catch_warnings(record=True) as log:
            simplefilter("always")
            warn("after reset")
        print(len(log))   # 1
    print('resetwarnings ok')

test_resetwarnings()


# ── warn_explicit ─────────────────────────────────────────────────────────────

def test_warn_explicit():
    with catch_warnings(record=True) as log:
        simplefilter("always")
        warn_explicit("explicit msg", RuntimeWarning, "myfile.py", 99)
    print(len(log))                     # 1
    print(log[0].message)               # explicit msg
    print(log[0].category.__name__)     # RuntimeWarning
    print(log[0].filename)              # myfile.py
    print(log[0].lineno)                # 99
    print('warn_explicit ok')

test_warn_explicit()


# ── filterwarnings message pattern ───────────────────────────────────────────

def test_filterwarnings_pattern():
    with catch_warnings(record=True) as log:
        simplefilter("always")
        filterwarnings("ignore", message="skip me")
        warn("skip me please")    # matches → ignored
        warn("show this")         # doesn't match → recorded
    print(len(log))   # 1
    print(log[0].message)   # show this
    print('filterwarnings_pattern ok')

test_filterwarnings_pattern()


# ── filterwarnings category ───────────────────────────────────────────────────

def test_filterwarnings_category():
    with catch_warnings(record=True) as log:
        simplefilter("always")
        filterwarnings("error", category=DeprecationWarning)
        warn("fine")   # UserWarning — not caught by error filter
        try:
            warn("dep", DeprecationWarning)
        except DeprecationWarning:
            log.append("caught_dep")
    # log has: WarningMessage("fine") + "caught_dep"
    print(len(log))                    # 2
    print(log[0].message)              # fine
    print(log[1])                      # caught_dep
    print('filterwarnings_category ok')

test_filterwarnings_category()


# ── catch_warnings restores state ────────────────────────────────────────────

def test_catch_warnings_restore():
    original_filter_count = len(warnings.filters)
    with catch_warnings():
        simplefilter("always")
        filterwarnings("ignore")
    # filters should be restored
    print(len(warnings.filters) == original_filter_count)   # True
    print('catch_warnings_restore ok')

test_catch_warnings_restore()


# ── WarningMessage attributes ─────────────────────────────────────────────────

def test_warning_message_attrs():
    with catch_warnings(record=True) as log:
        simplefilter("always")
        warn("check attrs", SyntaxWarning)
    wm = log[0]
    print(wm.message)                # check attrs
    print(wm.category.__name__)      # SyntaxWarning
    print(isinstance(wm.lineno, int))  # True
    print(wm.file is None)           # True
    print(wm.line is None)           # True
    print(wm.source is None)         # True
    print('warning_message_attrs ok')

test_warning_message_attrs()


# ── warn with category class ──────────────────────────────────────────────────

def test_warn_category_kwarg():
    with catch_warnings(record=True) as log:
        simplefilter("always")
        warn("msg", category=FutureWarning)
    print(log[0].category.__name__)   # FutureWarning
    print('warn_category_kwarg ok')

test_warn_category_kwarg()


# ── filters list is accessible ────────────────────────────────────────────────

def test_filters_list():
    with catch_warnings():
        resetwarnings()
        simplefilter("always")
        fl = warnings.filters
        print(isinstance(fl, list))   # True
        print(len(fl) >= 1)           # True
        print(fl[0][0])               # always
    print('filters_list ok')

test_filters_list()


print("done")

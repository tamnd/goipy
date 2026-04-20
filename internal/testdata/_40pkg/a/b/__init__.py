# Deepest package — three-dot relative import reaches up to _40pkg and
# back down to the helper module.
from ..helper import hello as _hello   # ..helper   -> _40pkg.a.helper
from ... import PKG_NAME                # ...        -> _40pkg (parent *after* __init__ runs)

def compute(x):
    return f"{_hello()}|{PKG_NAME}|{x * x}"

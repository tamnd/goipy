# Subpackage __init__ — exercises deeper nesting and relative imports.
from .. import util as _util   # ..  -> _39pkg ; ..util -> _39pkg.util
from . import leaf as _leaf    # .   -> _39pkg.sub

SUB_VERSION = "sub-v1"

def combined(x):
    return _util.double(_leaf.inc(x))

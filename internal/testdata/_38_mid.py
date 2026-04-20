# Middle module: imports the leaf and re-exports part of its surface.
# Also proves that function-level imports resolve via the same machinery
# as module-level imports.

import _38_leaf
from _38_leaf import shout as _shout

LEAF_VERSION = _38_leaf.VERSION

def loud(s):
    return _shout(s)

def lazy_version():
    # Function-local import (deferred until first call). Should hit the
    # sys.modules cache, not re-execute the leaf's body.
    import _38_leaf as l
    return l.VERSION

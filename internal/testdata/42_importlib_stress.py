import importlib

# Repeated import_module returns the cached object (is-identity, not equality).
a1 = importlib.import_module("_39pkg")
a2 = importlib.import_module("_39pkg")
print(a1 is a2)

# import_module + standard `import` share the same sys.modules cache.
import _39pkg
print(a1 is _39pkg)

# Dotted absolute returns the innermost module; compare to raw `import`.
import _39pkg.sub.leaf
leaf_via = importlib.import_module("_39pkg.sub.leaf")
print(leaf_via is _39pkg.sub.leaf)

# Multi-dot relative: `..leaf` with package `_39pkg.sub` reaches siblings.
util_rel = importlib.import_module("..util", package="_39pkg.sub")
print(util_rel.FLAG)
print(util_rel is _39pkg.util)

# Relative `.` with a package returns the package itself.
pkg_self = importlib.import_module(".", package="_39pkg")
print(pkg_self is _39pkg)

# reload returns the same module object.
state = importlib.import_module("_41_state")
reloaded = importlib.reload(state)
print(reloaded is state)

# reload on a package re-runs its __init__ without clobbering submodules.
before_leaf = _39pkg.sub.leaf
importlib.reload(_39pkg)
print(_39pkg.sub.leaf is before_leaf)
print(_39pkg.PACKAGE_MARKER)

# Relative import of a missing sibling raises ImportError.
try:
    importlib.import_module(".no_such_module", package="_39pkg")
except ImportError as e:
    print("rel-missing:", e)

# Typing: non-str name raises some kind of type error (CPython:
# AttributeError when it calls name.startswith; goipy: TypeError).
try:
    importlib.import_module(123)
except (TypeError, AttributeError):
    print("type-error: ok")

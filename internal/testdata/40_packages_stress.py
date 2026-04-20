import _40pkg
from _40pkg import PKG_NAME, compute
from _40pkg.a import GREETING, helper
from _40pkg.a.b import compute as compute2

print(PKG_NAME)
print(GREETING)
print(helper.hello())
print(compute(5))
print(compute is compute2)

# Eager-re-export: `b` and `a` are attributes on the package after __init__.
print(_40pkg.a is _40pkg.a)
print(_40pkg.a.b.__name__)

# Importing a submodule path repeatedly is idempotent and cached.
import _40pkg.a.b as deep1
import _40pkg.a.b as deep2
print(deep1 is deep2)

# Failure surface: attribute access on a non-package submodule.
import _40pkg.a.helper as h
try:
    from _40pkg.a.helper import nope
except ImportError as e:
    print("missing-name:", str(e).split(" (")[0])

# Failure surface: loading a non-existent module.
try:
    import _40pkg.a.does_not_exist  # noqa
except ImportError as e:
    print("missing-module:", e)

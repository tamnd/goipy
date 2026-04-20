import _38_mid
from _38_mid import loud, lazy_version
import _38_leaf  # already loaded transitively via _38_mid

# Transitive module is the same object everywhere
import _38_mid as again
print(_38_mid is again)
print(_38_mid.LEAF_VERSION)
print(_38_mid._38_leaf is _38_leaf)

# Re-export working through the middle module
print(loud("hi"))
print(lazy_version())

# Importing inside a function
def use():
    from _38_leaf import shout, VERSION
    return shout("ping"), VERSION

print(use())

# Shadowing an imported name locally does not affect the module
shout = "local"
print(shout)
print(_38_leaf.shout("ok"))

# Rebinding on the module object is visible elsewhere
_38_leaf.extra = 99
import _38_leaf as leaf2
print(leaf2.extra)

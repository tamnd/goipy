import _39pkg
import _39pkg.util
from _39pkg import banner, util
from _39pkg.sub import leaf, SUB_VERSION, combined
from _39pkg.sub.leaf import inc, tag

print(_39pkg.PACKAGE_MARKER)
print(banner())
print(util.double(7))
print(util.FLAG)

# Package attribute access to submodules
print(_39pkg.util is util)
print(_39pkg.sub.leaf is leaf)

# Imported names
print(inc(41))
print(tag())
print(SUB_VERSION)
print(combined(4))  # leaf.inc(4)=5, util.double(5)=10

# Dotted-module import returns the top-level package when no `from`
import _39pkg.sub.leaf as deep
print(deep is leaf)

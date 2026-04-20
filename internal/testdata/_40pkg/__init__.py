# Eager re-export pattern: top-level package pulls submodules into its
# namespace at import time. PKG_NAME is set *before* submodule imports so
# the deepest package can read it via a circular-but-forward import.
PKG_NAME = "_40pkg"

from . import a
from .a import b
from .a.b import compute

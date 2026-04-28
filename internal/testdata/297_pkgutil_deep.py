"""Comprehensive pkgutil module test — covers all public API from the Python docs."""
import pkgutil
import zipfile
import tempfile
import os


# ── ModuleInfo namedtuple ─────────────────────────────────────────────────────

mi = pkgutil.ModuleInfo(None, "mymod", False)
print(type(mi).__name__ == "ModuleInfo")        # True
print(mi._fields == ("module_finder", "name", "ispkg"))  # True
print(mi.module_finder is None)                 # True
print(mi.name == "mymod")                       # True
print(mi.ispkg == False)                        # True
print(isinstance(mi, tuple))                    # True
print(mi[0] is None)                            # True
print(mi[1] == "mymod")                         # True
print(mi[2] == False)                           # True

mi2 = pkgutil.ModuleInfo(object(), "pkg", True)
print(mi2.ispkg == True)                        # True


# ── simplegeneric ─────────────────────────────────────────────────────────────

@pkgutil.simplegeneric
def myfunc(obj):
    return "default"

print(callable(myfunc))                         # True
print(myfunc(42) == "default")                  # True
print(myfunc("x") == "default")                 # True

@myfunc.register(int)
def myfunc_int(obj):
    return "int"

print(myfunc(42) == "int")                      # True
print(myfunc("hello") == "default")             # True
print(myfunc(3.14) == "default")                # True


# ── extend_path ───────────────────────────────────────────────────────────────

result = pkgutil.extend_path(["/a", "/b"], "mypkg")
print(isinstance(result, list))                 # True
print("/a" in result)                           # True
print("/b" in result)                           # True

# empty path stays empty
result2 = pkgutil.extend_path([], "mypkg")
print(isinstance(result2, list))                # True


# ── get_data ──────────────────────────────────────────────────────────────────

# nonexistent package returns None
result3 = pkgutil.get_data("nonexistent_xyz_pkg", "data.txt")
print(result3 is None)                          # True


# ── read_code ─────────────────────────────────────────────────────────────────

import io
stream = io.BytesIO(b"")
r = pkgutil.read_code(stream)
print(r is None)                                # True


# ── iter_importers ────────────────────────────────────────────────────────────

it = pkgutil.iter_importers()
print(hasattr(it, "__iter__"))                  # True
print(hasattr(it, "__next__"))                  # True


# ── iter_modules / iter_importer_modules / walk_packages with a zip ───────────

tmp = tempfile.mktemp(suffix=".zip")
with zipfile.ZipFile(tmp, "w") as zf:
    zf.writestr("pkga/__init__.py", "")
    zf.writestr("pkga/sub/__init__.py", "")
    zf.writestr("mod1.py", "")
    zf.writestr("mod2.py", "")

try:
    # iter_modules on a zip path
    items = list(pkgutil.iter_modules([tmp]))
    names = sorted(mi.name for mi in items)
    print(isinstance(items, list))              # True
    print(len(items) == 3)                      # True  (pkga, mod1, mod2)
    print("pkga" in names)                      # True
    print("mod1" in names)                      # True
    print("mod2" in names)                      # True

    # ispkg flag
    pkg_item = next(m for m in items if m.name == "pkga")
    print(pkg_item.ispkg == True)               # True
    mod_item = next(m for m in items if m.name == "mod1")
    print(mod_item.ispkg == False)              # True

    # prefix support
    items_p = list(pkgutil.iter_modules([tmp], prefix="X."))
    pnames = sorted(mi.name for mi in items_p)
    print("X.pkga" in pnames)                   # True
    print("X.mod1" in pnames)                   # True

    # all ModuleInfo instances
    print(all(isinstance(m, pkgutil.ModuleInfo) for m in items))  # True

    # get_importer on zip
    imp = pkgutil.get_importer(tmp)
    print(imp is not None)                      # True
    print(type(imp).__name__ == "zipimporter")  # True

    # iter_importer_modules on zip importer
    imp2 = pkgutil.get_importer(tmp)
    pairs = list(pkgutil.iter_importer_modules(imp2))
    pair_names = sorted(n for n, _ in pairs)
    print(isinstance(pairs, list))              # True
    print("pkga" in pair_names)                 # True
    print("mod1" in pair_names)                 # True

    # iter_zipimport_modules
    pairs2 = list(pkgutil.iter_zipimport_modules(imp2))
    pair_names2 = sorted(n for n, _ in pairs2)
    print("pkga" in pair_names2)                # True

    # walk_packages on zip
    walked = list(pkgutil.walk_packages([tmp]))
    wnames = sorted(mi.name for mi in walked)
    print(isinstance(walked, list))             # True
    print("pkga" in wnames)                     # True
    print("mod1" in wnames)                     # True

finally:
    os.unlink(tmp)


# ── iter_modules on empty list ────────────────────────────────────────────────

empty_items = list(pkgutil.iter_modules([]))
print(empty_items == [])                        # True


# ── resolve_name ─────────────────────────────────────────────────────────────

import os.path
r1 = pkgutil.resolve_name("os.path")
print(r1 is os.path)                            # True

r2 = pkgutil.resolve_name("os:path")
print(r2 is os.path)                            # True

import os as _os
r3 = pkgutil.resolve_name("os:getcwd")
print(r3 is _os.getcwd)                         # True

# resolve_name with invalid name raises ValueError
try:
    pkgutil.resolve_name(":bad")
    print(False)
except ValueError:
    print(True)                                  # True

# resolve_name nonexistent raises ModuleNotFoundError
try:
    pkgutil.resolve_name("nonexistent_xyz_module")
    print(False)
except (ModuleNotFoundError, ImportError):
    print(True)                                  # True


print("done")

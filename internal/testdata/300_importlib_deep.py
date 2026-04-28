import importlib
import importlib.util
import importlib.abc
import importlib.machinery
import importlib.resources
import importlib.metadata

# ── importlib (base) ──────────────────────────────────────────────────────────

# import_module returns a module
m = importlib.import_module('os')
print(type(m).__name__)

# import_module with relative name
m2 = importlib.import_module('.path', 'os')
print(type(m2).__name__)

# invalidate_caches returns None
print(importlib.invalidate_caches())

# reload returns the module
import os
m3 = importlib.reload(os)
print(type(m3).__name__)

# ── importlib.util ────────────────────────────────────────────────────────────

# MAGIC_NUMBER is 4 bytes
print(type(importlib.util.MAGIC_NUMBER).__name__)
print(len(importlib.util.MAGIC_NUMBER))

# cache_from_source
c = importlib.util.cache_from_source('/tmp/foo.py')
print('__pycache__' in c)
print(c.endswith('.pyc'))

# cache_from_source with optimization
c2 = importlib.util.cache_from_source('/tmp/foo.py', optimization='1')
print('opt-1' in c2)

# source_from_cache
s = importlib.util.source_from_cache('/tmp/__pycache__/foo.cpython-314.pyc')
print(s)

# source_from_cache ValueError on non-pycache path
try:
    importlib.util.source_from_cache('/tmp/foo.pyc')
    print('no error')
except ValueError:
    print('ValueError')

# resolve_name
print(importlib.util.resolve_name('os', None))
print(importlib.util.resolve_name('.path', 'os'))
try:
    importlib.util.resolve_name('.path', None)
    print('no error')
except ImportError:
    print('ImportError')

# find_spec for known module
spec = importlib.util.find_spec('os')
print(type(spec).__name__)
print(spec.name)

# find_spec for unknown module
print(importlib.util.find_spec('_nonexistent_xyz_importlib_test'))

# source_hash returns 8 bytes
h = importlib.util.source_hash(b'hello world')
print(type(h).__name__)
print(len(h))

# decode_source
print(importlib.util.decode_source(b'hello'))

# spec_from_file_location
spec2 = importlib.util.spec_from_file_location('mymod', '/tmp/mymod.py')
print(type(spec2).__name__)
print(spec2.name)
print(spec2.origin)
print(spec2.has_location)

# spec_from_loader
spec3 = importlib.util.spec_from_loader('mymod3', None)
print(type(spec3).__name__)
print(spec3.name)

# module_from_spec
mod = importlib.util.module_from_spec(spec3)
print(type(mod).__name__)
print(mod.__name__)

# Loader and LazyLoader classes exist
print(callable(importlib.util.Loader))
print(callable(importlib.util.LazyLoader))

# ── importlib.machinery ───────────────────────────────────────────────────────

# ModuleSpec basic
spec4 = importlib.machinery.ModuleSpec('testmod', None)
print(spec4.name)
print(spec4.loader)
print(spec4.origin)
print(spec4.submodule_search_locations)
print(spec4.has_location)
print(repr(spec4.parent))

# ModuleSpec with origin sets has_location
spec5 = importlib.machinery.ModuleSpec('testmod2', None, origin='/tmp/x.py')
print(spec5.has_location)

# ModuleSpec dotted name → parent
spec6 = importlib.machinery.ModuleSpec('pkg.sub', None)
print(repr(spec6.parent))

# ModuleSpec is_package → submodule_search_locations
spec7 = importlib.machinery.ModuleSpec('mypkg', None, is_package=True)
print(type(spec7.submodule_search_locations).__name__)

# Suffixes
print(type(importlib.machinery.SOURCE_SUFFIXES).__name__)
print('.py' in importlib.machinery.SOURCE_SUFFIXES)
print(type(importlib.machinery.BYTECODE_SUFFIXES).__name__)
print('.pyc' in importlib.machinery.BYTECODE_SUFFIXES)
print(type(importlib.machinery.EXTENSION_SUFFIXES).__name__)

# all_suffixes
suf = importlib.machinery.all_suffixes()
print(type(suf).__name__)
print('.py' in suf)
print('.pyc' in suf)

# BuiltinImporter.find_spec for nonexistent → None
bi_spec = importlib.machinery.BuiltinImporter.find_spec('_nonexistent_builtin_xyz', None)
print(bi_spec)

# PathFinder.find_spec for known module → ModuleSpec
pf_spec = importlib.machinery.PathFinder.find_spec('os', None)
print(type(pf_spec).__name__ if pf_spec else None)

# ── importlib.abc ─────────────────────────────────────────────────────────────

print(callable(importlib.abc.Loader))
print(callable(importlib.abc.MetaPathFinder))
print(callable(importlib.abc.PathEntryFinder))
print(callable(importlib.abc.InspectLoader))
print(callable(importlib.abc.ExecutionLoader))

# ── importlib.resources ───────────────────────────────────────────────────────

# files() returns something traversable-like
try:
    f = importlib.resources.files('os')
    print(type(f).__name__ != '')
except Exception:
    print(False)

# ── importlib.metadata ────────────────────────────────────────────────────────

# PackageNotFoundError is an exception class
print(issubclass(importlib.metadata.PackageNotFoundError, Exception))

# version for nonexistent package raises PackageNotFoundError
try:
    importlib.metadata.version('_nonexistent_pkg_xyz_importlib_test')
    print('no error')
except importlib.metadata.PackageNotFoundError:
    print('PackageNotFoundError')

# distributions returns iterable
dists = list(importlib.metadata.distributions())
print(type(dists).__name__)

# packages_distributions returns dict
pd = importlib.metadata.packages_distributions()
print(type(pd).__name__)

# entry_points returns list/sequence
ep = importlib.metadata.entry_points()
print(type(ep).__name__)

print('done')

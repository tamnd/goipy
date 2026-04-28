import modulefinder

# Module class basics
m = modulefinder.Module('mymod', file='/tmp/mymod.py', path=None)
print(m.__name__)
print(m.__file__)
print(m.__path__)
print(repr(m))
print(type(m.globalnames).__name__)
print(type(m.starimports).__name__)

# Module with path list
m2 = modulefinder.Module('pkg', file='/tmp/pkg/__init__.py', path=['/tmp/pkg'])
print(repr(m2))

# Module with no file
m3 = modulefinder.Module('bare')
print(repr(m3))

# packagePathMap and replacePackageMap are dicts
print(type(modulefinder.packagePathMap).__name__)
print(type(modulefinder.replacePackageMap).__name__)

# AddPackagePath
modulefinder.AddPackagePath('mypkg', '/some/path')
print('mypkg' in modulefinder.packagePathMap)
print(modulefinder.packagePathMap['mypkg'])
# Adding same pkg again appends
modulefinder.AddPackagePath('mypkg', '/other/path')
print(len(modulefinder.packagePathMap['mypkg']))

# ReplacePackage
modulefinder.ReplacePackage('oldpkg', 'newpkg')
print(modulefinder.replacePackageMap['oldpkg'])

# ModuleFinder instance
mf = modulefinder.ModuleFinder()
print(type(mf.modules).__name__)
print(type(mf.badmodules).__name__)
print(mf.debug)
print(mf.excludes)

# ModuleFinder with args
mf2 = modulefinder.ModuleFinder(path=['/tmp'], debug=1, excludes=['os'])
print(mf2.debug)
print(mf2.excludes)

# add_module
mod = mf.add_module('testmod')
print(type(mod).__name__)
print('testmod' in mf.modules)
print(mf.modules['testmod'].__name__)

# any_missing on empty badmodules
print(mf.any_missing())
print(mf.any_missing_maybe())

# any_missing with badmodules
mf.badmodules['missing1'] = {'__main__': 1}
mf.badmodules['missing2'] = {'__main__': 1}
missing = mf.any_missing()
print(len(missing))
print('missing1' in missing)
print('missing2' in missing)

(missing2, maybe2) = mf.any_missing_maybe()
print(len(missing2))
print(len(maybe2))

print('done')

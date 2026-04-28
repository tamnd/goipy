import runpy

# run_module and run_path are callable
print(callable(runpy.run_module))
print(callable(runpy.run_path))

# run_module returns a dict
r = runpy.run_module('os.path')
print(type(r).__name__)
print('__name__' in r)

# run_module respects run_name
r2 = runpy.run_module('os.path', run_name='mytest')
print(r2['__name__'])

# run_module with positional init_globals
r3 = runpy.run_module('os.path', {'_test_key': 77})
print(r3.get('_test_key'))

# run_module keyword init_globals
r4 = runpy.run_module('os.path', init_globals={'_kw_key': 88})
print(r4.get('_kw_key'))

# run_module for missing module raises ImportError
try:
    runpy.run_module('_nonexistent_xyz_module_runpy_test')
    print('no error')
except ImportError:
    print('ImportError')

# run_path for missing file raises OSError / FileNotFoundError
try:
    runpy.run_path('/nonexistent_xyz_path_runpy_test_file.py')
    print('no error')
except (OSError, FileNotFoundError):
    print('OSError')

print('done')

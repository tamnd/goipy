import compileall
import tempfile
import os

# ── 1. __all__ ────────────────────────────────────────────────────────────────
print(compileall.__all__ == ['compile_dir', 'compile_file', 'compile_path'])

# ── 2. callable checks ────────────────────────────────────────────────────────
print(callable(compileall.compile_file))
print(callable(compileall.compile_dir))
print(callable(compileall.compile_path))
print(callable(compileall.main))

# ── 3. compile_file with a valid temp .py file ────────────────────────────────
# Both CPython (success) and goipy stub return True.
tmpdir = tempfile.mkdtemp()
pyfile = os.path.join(tmpdir, 'goipy_test_mod.py')
try:
    with open(pyfile, 'w') as f:
        f.write('x = 1\n')
    result = compileall.compile_file(pyfile, quiet=2)
    print(isinstance(result, bool))
    print(result == True)
finally:
    # clean up source
    if os.path.exists(pyfile):
        os.unlink(pyfile)
    # clean up __pycache__ created by CPython
    pycache = os.path.join(tmpdir, '__pycache__')
    if os.path.isdir(pycache):
        for f in os.listdir(pycache):
            os.unlink(os.path.join(pycache, f))
        os.rmdir(pycache)
    os.rmdir(tmpdir)

# ── 4. compile_dir with a valid temp dir ─────────────────────────────────────
tmpdir2 = tempfile.mkdtemp()
pyfile2 = os.path.join(tmpdir2, 'goipy_test_mod2.py')
try:
    with open(pyfile2, 'w') as f:
        f.write('y = 2\n')
    result2 = compileall.compile_dir(tmpdir2, quiet=2)
    print(isinstance(result2, bool))
    print(result2 == True)
finally:
    if os.path.exists(pyfile2):
        os.unlink(pyfile2)
    pycache2 = os.path.join(tmpdir2, '__pycache__')
    if os.path.isdir(pycache2):
        for f in os.listdir(pycache2):
            os.unlink(os.path.join(pycache2, f))
        os.rmdir(pycache2)
    os.rmdir(tmpdir2)

# ── 5. compile_path returns bool ─────────────────────────────────────────────
# Pass maxlevels=0 and quiet=2 to avoid slow compilation of all sys.path entries.
result3 = compileall.compile_path(skip_curdir=1, maxlevels=0, quiet=2)
print(isinstance(result3, bool))

# ── 6. compile_file returns bool ─────────────────────────────────────────────
# Test with a single-arg call using a path that doesn't exist.
# CPython returns True for a nonexistent file with quiet=2 (no compile attempt logged).
result4 = compileall.compile_file('/nonexistent_goipy_test.py', quiet=2)
print(isinstance(result4, bool))

# ── 7. compile_dir returns bool for missing dir ───────────────────────────────
result5 = compileall.compile_dir('/nonexistent_goipy_dir', quiet=2)
print(isinstance(result5, bool))

# ── 8. main is callable and returns None ─────────────────────────────────────
# main() with no args just parses sys.argv; don't call it to avoid side effects.
print(callable(compileall.main))

print('done')

import py_compile

# ── 1. __all__ ────────────────────────────────────────────────────────────────
print(py_compile.__all__ == ['compile', 'main', 'PyCompileError', 'PycInvalidationMode'])

# ── 2. PyCompileError is subclass of Exception ───────────────────────────────
print(issubclass(py_compile.PyCompileError, Exception))

# ── 3. PyCompileError construction — exc_type_name ───────────────────────────
err = py_compile.PyCompileError(Exception, 'bad syntax', 'file.py')
print(err.exc_type_name == 'Exception')

# ── 4. PyCompileError — exc_value ────────────────────────────────────────────
print(err.exc_value == 'bad syntax')

# ── 5. PyCompileError — file ──────────────────────────────────────────────────
print(err.file == 'file.py')

# ── 6. PyCompileError — default msg contains type and value ──────────────────
print('Exception' in err.msg)
print('bad syntax' in err.msg)

# ── 7. str(err) returns msg ───────────────────────────────────────────────────
print(str(err) == err.msg)

# ── 8. PyCompileError with explicit msg ──────────────────────────────────────
err2 = py_compile.PyCompileError(ValueError, 'bad value', 'f.py', 'my custom message')
print(err2.msg == 'my custom message')
print(str(err2) == 'my custom message')

# ── 9. PyCompileError exc_type_name from ValueError ──────────────────────────
print(err2.exc_type_name == 'ValueError')
print(err2.exc_value == 'bad value')
print(err2.file == 'f.py')

# ── 10. PyCompileError catchable as Exception ─────────────────────────────────
try:
    raise py_compile.PyCompileError(Exception, 'oops', 'test.py')
except Exception:
    print(True)

# ── 11. PyCompileError catchable as PyCompileError ───────────────────────────
try:
    raise py_compile.PyCompileError(Exception, 'oops', 'test.py')
except py_compile.PyCompileError:
    print(True)

# ── 12. PyCompileError args accessible ───────────────────────────────────────
err3 = py_compile.PyCompileError(TypeError, 'type mismatch', 'mod.py')
print(err3.exc_type_name == 'TypeError')
print(err3.file == 'mod.py')

# ── 13. PycInvalidationMode.TIMESTAMP ────────────────────────────────────────
print(py_compile.PycInvalidationMode.TIMESTAMP.value == 1)
print(py_compile.PycInvalidationMode.TIMESTAMP.name == 'TIMESTAMP')

# ── 14. PycInvalidationMode.CHECKED_HASH ─────────────────────────────────────
print(py_compile.PycInvalidationMode.CHECKED_HASH.value == 2)
print(py_compile.PycInvalidationMode.CHECKED_HASH.name == 'CHECKED_HASH')

# ── 15. PycInvalidationMode.UNCHECKED_HASH ───────────────────────────────────
print(py_compile.PycInvalidationMode.UNCHECKED_HASH.value == 3)
print(py_compile.PycInvalidationMode.UNCHECKED_HASH.name == 'UNCHECKED_HASH')

# ── 16. PycInvalidationMode iteration ────────────────────────────────────────
members = list(py_compile.PycInvalidationMode)
print(len(members) == 3)
print(members[0].value == 1)
print(members[1].value == 2)
print(members[2].value == 3)
print(members[0].name == 'TIMESTAMP')
print(members[1].name == 'CHECKED_HASH')
print(members[2].name == 'UNCHECKED_HASH')

# ── 17. PycInvalidationMode member identity ───────────────────────────────────
print(py_compile.PycInvalidationMode.TIMESTAMP is members[0])
print(py_compile.PycInvalidationMode.CHECKED_HASH is members[1])
print(py_compile.PycInvalidationMode.UNCHECKED_HASH is members[2])

# ── 18. callable checks ───────────────────────────────────────────────────────
print(callable(py_compile.compile))
print(callable(py_compile.main))

# ── 19. compile() returns None (stub) ────────────────────────────────────────
import tempfile, os
with tempfile.NamedTemporaryFile(suffix='.py', delete=False, mode='w') as f:
    f.write('x = 1\n')
    fname = f.name
result = py_compile.compile(fname)
# goipy stub returns None; CPython returns a .pyc path
print(result is None or isinstance(result, str))
# cleanup
os.unlink(fname)
if result is not None and isinstance(result, str) and os.path.exists(result):
    os.unlink(result)

print('done')

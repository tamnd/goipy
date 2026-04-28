import tabnanny
import tokenize
import io

# ── 1. __all__ ────────────────────────────────────────────────────────────────
print(tabnanny.__all__ == ['check', 'NannyNag', 'process_tokens'])

# ── 2. __version__ ────────────────────────────────────────────────────────────
print(tabnanny.__version__ == '6')

# ── 3. Module-level flags ─────────────────────────────────────────────────────
print(tabnanny.filename_only == 0)
print(tabnanny.verbose == 0)

# ── 4. NannyNag is subclass of Exception ─────────────────────────────────────
print(issubclass(tabnanny.NannyNag, Exception))

# ── 5. NannyNag construction ──────────────────────────────────────────────────
nag = tabnanny.NannyNag(5, 'bad indent', '\tx  y\n')
print(nag.get_lineno() == 5)
print(nag.get_msg() == 'bad indent')
print(nag.get_line() == '\tx  y\n')

# ── 6. NannyNag with different args ──────────────────────────────────────────
nag2 = tabnanny.NannyNag(1, 'token error', 'if True:\n')
print(nag2.get_lineno() == 1)
print(nag2.get_msg() == 'token error')
print(nag2.get_line() == 'if True:\n')

# ── 7. NannyNag is catchable as Exception ────────────────────────────────────
try:
    raise tabnanny.NannyNag(10, 'test', 'line\n')
except Exception as e:
    print(True)

# ── 8. NannyNag is catchable as NannyNag ─────────────────────────────────────
try:
    raise tabnanny.NannyNag(10, 'test', 'line\n')
except tabnanny.NannyNag as e:
    print(e.get_lineno() == 10)
    print(e.get_msg() == 'test')

# ── 9. process_tokens([]) returns None ───────────────────────────────────────
result = tabnanny.process_tokens([])
print(result is None)

# ── 10. process_tokens on clean token list does not raise ────────────────────
src = b'x = 1\ny = 2\n'
tokens = list(tokenize.tokenize(io.BytesIO(src).readline))
try:
    tabnanny.process_tokens(tokens)
    print(True)
except tabnanny.NannyNag:
    print(False)

# ── 11. process_tokens on indented code (no tabs/spaces mix) ─────────────────
src2 = b'if True:\n    x = 1\n    y = 2\n'
tokens2 = list(tokenize.tokenize(io.BytesIO(src2).readline))
try:
    tabnanny.process_tokens(tokens2)
    print(True)
except tabnanny.NannyNag:
    print(False)

# ── 12. callable checks ───────────────────────────────────────────────────────
print(callable(tabnanny.check))
print(callable(tabnanny.process_tokens))
print(callable(tabnanny.format_witnesses))
print(callable(tabnanny.errprint))

# ── 13. Whitespace is a class ─────────────────────────────────────────────────
print(isinstance(tabnanny.Whitespace, type))

# ── 14. check() is callable (don't invoke with nonexistent path to avoid stderr) ─
print(callable(tabnanny.check))

# ── 15. format_witnesses with empty list ──────────────────────────────────────
fw = tabnanny.format_witnesses([])
print(isinstance(fw, str))

# ── 17. NannyNag get_lineno returns int ──────────────────────────────────────
nag3 = tabnanny.NannyNag(42, 'msg', 'line\n')
print(isinstance(nag3.get_lineno(), int))
print(isinstance(nag3.get_msg(), str))
print(isinstance(nag3.get_line(), str))

print('done')

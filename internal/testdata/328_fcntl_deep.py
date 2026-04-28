import fcntl

# ── all constants with exact CPython macOS values ──────────────────────────────
_EXPECTED = {
    'F_DUPFD': 0,
    'F_GETFD': 1,
    'F_SETFD': 2,
    'F_GETFL': 3,
    'F_SETFL': 4,
    'F_GETOWN': 5,
    'F_SETOWN': 6,
    'F_GETLK': 7,
    'F_SETLK': 8,
    'F_SETLKW': 9,
    'F_DUPFD_CLOEXEC': 67,
    'F_RDLCK': 1,
    'F_WRLCK': 3,
    'F_UNLCK': 2,
    'FD_CLOEXEC': 1,
    'LOCK_SH': 1,
    'LOCK_EX': 2,
    'LOCK_NB': 4,
    'LOCK_UN': 8,
    'F_GETNOSIGPIPE': 74,
    'F_SETNOSIGPIPE': 73,
    'F_NOCACHE': 48,
    'F_FULLFSYNC': 51,
    'F_RDAHEAD': 45,
    'F_GETPATH': 50,
    'F_GETLEASE': 107,
    'F_SETLEASE': 106,
    'F_OFD_GETLK': 92,
    'F_OFD_SETLK': 90,
    'F_OFD_SETLKW': 91,
    'FASYNC': 64,
}

for name, expected in _EXPECTED.items():
    actual = getattr(fcntl, name)
    assert actual == expected, f'{name}: expected {expected}, got {actual}'

# ── removed constants must NOT be present ─────────────────────────────────────
for removed in ('F_FREEZE_FS', 'F_PREALLOCATE', 'F_RDADVISE', 'F_SETSIZE', 'F_THAW_FS'):
    assert not hasattr(fcntl, removed), f'unexpected attribute: {removed}'

# ── all functions callable ─────────────────────────────────────────────────────
assert callable(fcntl.fcntl)
assert callable(fcntl.flock)
assert callable(fcntl.ioctl)
assert callable(fcntl.lockf)

# ── fcntl(fd, cmd) returns int (fd 0 = stdin, always open) ───────────────────
try:
    result = fcntl.fcntl(0, fcntl.F_GETFL)
    assert isinstance(result, int), f'fcntl returned {type(result).__name__}'
except OSError:
    pass

try:
    result2 = fcntl.fcntl(0, fcntl.F_GETFD)
    assert isinstance(result2, int)
except OSError:
    pass

# fcntl with explicit int arg also returns int
try:
    result3 = fcntl.fcntl(0, fcntl.F_SETFL, 0)
    assert isinstance(result3, int)
except OSError:
    pass

# ── flock: callable; may raise OSError on non-regular files ──────────────────
try:
    fcntl.flock(0, fcntl.LOCK_SH | fcntl.LOCK_NB)
except OSError:
    pass

# ── ioctl: returns int when called with int arg ────────────────────────────────
try:
    result4 = fcntl.ioctl(0, fcntl.F_GETFL, 0)
    assert isinstance(result4, int)
except OSError:
    pass

# ioctl with bytes arg: returns int or bytes
try:
    result5 = fcntl.ioctl(0, fcntl.F_GETFL, b'\x00\x00\x00\x00')
    assert isinstance(result5, (int, bytes))
except OSError:
    pass

# ── lockf: callable; may raise on non-regular file ───────────────────────────
try:
    fcntl.lockf(0, fcntl.LOCK_SH | fcntl.LOCK_NB)
except OSError:
    pass

print('ok')

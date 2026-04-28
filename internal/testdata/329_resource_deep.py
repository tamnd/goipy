import resource

# ── constants with exact CPython macOS values ──────────────────────────────────
assert resource.RLIM_INFINITY == 9223372036854775807
assert resource.RLIMIT_CPU == 0
assert resource.RLIMIT_FSIZE == 1
assert resource.RLIMIT_DATA == 2
assert resource.RLIMIT_STACK == 3
assert resource.RLIMIT_CORE == 4
assert resource.RLIMIT_RSS == 5
assert resource.RLIMIT_MEMLOCK == 6
assert resource.RLIMIT_NPROC == 7
assert resource.RLIMIT_NOFILE == 8
assert resource.RLIMIT_AS == 5
assert resource.RUSAGE_SELF == 0
assert resource.RUSAGE_CHILDREN == -1

# ── error is an exception class ────────────────────────────────────────────────
assert issubclass(resource.error, Exception)

# ── struct_rusage class ────────────────────────────────────────────────────────
assert hasattr(resource, 'struct_rusage')

# ── getrusage returns struct_rusage with correctly typed fields ────────────────
r = resource.getrusage(resource.RUSAGE_SELF)
assert type(r).__name__ == 'struct_rusage'

# time fields are float
assert isinstance(r.ru_utime, float), f'ru_utime: expected float, got {type(r.ru_utime).__name__}'
assert isinstance(r.ru_stime, float), f'ru_stime: expected float, got {type(r.ru_stime).__name__}'

# counter fields are int
for field in ('ru_maxrss', 'ru_ixrss', 'ru_idrss', 'ru_isrss',
              'ru_minflt', 'ru_majflt', 'ru_nswap', 'ru_inblock',
              'ru_oublock', 'ru_msgsnd', 'ru_msgrcv', 'ru_nsignals',
              'ru_nvcsw', 'ru_nivcsw'):
    v = getattr(r, field)
    assert isinstance(v, int), f'{field}: expected int, got {type(v).__name__}'

# time values are non-negative
assert r.ru_utime >= 0.0
assert r.ru_stime >= 0.0

# ── getrusage(RUSAGE_CHILDREN) works ──────────────────────────────────────────
rc = resource.getrusage(resource.RUSAGE_CHILDREN)
assert isinstance(rc.ru_utime, float)
assert isinstance(rc.ru_maxrss, int)

# ── getrlimit returns 2-tuple of ints ─────────────────────────────────────────
for res in (resource.RLIMIT_NOFILE, resource.RLIMIT_CPU, resource.RLIMIT_STACK):
    rl = resource.getrlimit(res)
    assert isinstance(rl, tuple), f'getrlimit: expected tuple, got {type(rl).__name__}'
    assert len(rl) == 2
    assert isinstance(rl[0], int)
    assert isinstance(rl[1], int)

# ── setrlimit is callable ──────────────────────────────────────────────────────
assert callable(resource.setrlimit)

# ── getpagesize returns positive int ──────────────────────────────────────────
ps = resource.getpagesize()
assert isinstance(ps, int)
assert ps > 0

print('ok')

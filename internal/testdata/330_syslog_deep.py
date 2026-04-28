import syslog

# ── all constants with exact CPython macOS values ──────────────────────────────
_EXPECTED = {
    # priority levels
    'LOG_EMERG':   0,
    'LOG_ALERT':   1,
    'LOG_CRIT':    2,
    'LOG_ERR':     3,
    'LOG_WARNING': 4,
    'LOG_NOTICE':  5,
    'LOG_INFO':    6,
    'LOG_DEBUG':   7,
    # facilities
    'LOG_KERN':       0,
    'LOG_USER':       8,
    'LOG_MAIL':       16,
    'LOG_DAEMON':     24,
    'LOG_AUTH':       32,
    'LOG_SYSLOG':     40,
    'LOG_LPR':        48,
    'LOG_NEWS':       56,
    'LOG_UUCP':       64,
    'LOG_CRON':       72,
    'LOG_AUTHPRIV':   80,
    'LOG_FTP':        88,
    'LOG_NETINFO':    96,
    'LOG_REMOTEAUTH': 104,
    'LOG_INSTALL':    112,
    'LOG_RAS':        120,
    'LOG_LOCAL0':     128,
    'LOG_LOCAL1':     136,
    'LOG_LOCAL2':     144,
    'LOG_LOCAL3':     152,
    'LOG_LOCAL4':     160,
    'LOG_LOCAL5':     168,
    'LOG_LOCAL6':     176,
    'LOG_LOCAL7':     184,
    'LOG_LAUNCHD':    192,
    # openlog options
    'LOG_PID':    1,
    'LOG_CONS':   2,
    'LOG_ODELAY': 4,
    'LOG_NDELAY': 8,
    'LOG_NOWAIT': 16,
    'LOG_PERROR': 32,
}

for name, expected in _EXPECTED.items():
    actual = getattr(syslog, name)
    assert actual == expected, f'{name}: expected {expected}, got {actual}'

# ── all functions callable ─────────────────────────────────────────────────────
assert callable(syslog.syslog)
assert callable(syslog.openlog)
assert callable(syslog.closelog)
assert callable(syslog.setlogmask)
assert callable(syslog.LOG_MASK)
assert callable(syslog.LOG_UPTO)

# ── LOG_MASK(pri) == 1 << pri ─────────────────────────────────────────────────
assert syslog.LOG_MASK(syslog.LOG_EMERG)   == 1    # 1 << 0
assert syslog.LOG_MASK(syslog.LOG_ALERT)   == 2    # 1 << 1
assert syslog.LOG_MASK(syslog.LOG_CRIT)    == 4    # 1 << 2
assert syslog.LOG_MASK(syslog.LOG_ERR)     == 8    # 1 << 3
assert syslog.LOG_MASK(syslog.LOG_WARNING) == 16   # 1 << 4
assert syslog.LOG_MASK(syslog.LOG_NOTICE)  == 32   # 1 << 5
assert syslog.LOG_MASK(syslog.LOG_INFO)    == 64   # 1 << 6
assert syslog.LOG_MASK(syslog.LOG_DEBUG)   == 128  # 1 << 7

# ── LOG_UPTO(pri) == (1 << (pri+1)) - 1 ──────────────────────────────────────
assert syslog.LOG_UPTO(syslog.LOG_EMERG)   == 1    # (1<<1)-1
assert syslog.LOG_UPTO(syslog.LOG_ERR)     == 15   # (1<<4)-1
assert syslog.LOG_UPTO(syslog.LOG_INFO)    == 127  # (1<<7)-1
assert syslog.LOG_UPTO(syslog.LOG_DEBUG)   == 255  # (1<<8)-1

# ── setlogmask returns int (previous mask) ────────────────────────────────────
prev = syslog.setlogmask(syslog.LOG_UPTO(syslog.LOG_INFO))
assert isinstance(prev, int)
# restore
syslog.setlogmask(prev)

# ── syslog() does not raise ───────────────────────────────────────────────────
syslog.syslog(syslog.LOG_INFO, 'goipy test')
syslog.syslog(syslog.LOG_DEBUG, 'debug message')

# ── openlog / closelog do not raise ──────────────────────────────────────────
syslog.openlog('goipy', syslog.LOG_PID, syslog.LOG_USER)
syslog.closelog()

# ── openlog with just ident ───────────────────────────────────────────────────
syslog.openlog('goipy2')
syslog.closelog()

print('ok')

import posix

# ── new constants ──────────────────────────────────────────────────────────────
assert posix.POSIX_SPAWN_OPEN == 0
assert posix.POSIX_SPAWN_CLOSE == 1
assert posix.POSIX_SPAWN_DUP2 == 2

assert posix.PRIO_DARWIN_PROCESS == 4
assert posix.PRIO_DARWIN_THREAD == 3
assert posix.PRIO_DARWIN_BG == 4096
assert posix.PRIO_DARWIN_NONUI == 4097

assert posix.P_ALL == 0
assert posix.P_PID == 1
assert posix.P_PGID == 2

assert posix.ST_RDONLY == 1
assert posix.ST_NOSUID == 2

assert posix.TMP_MAX == 308915776

assert posix.WCONTINUED == 16
assert posix.WEXITED == 4
assert posix.WNOWAIT == 32
assert posix.WSTOPPED == 8

# ── older constants still present ──────────────────────────────────────────────
assert posix.O_RDONLY == 0
assert posix.O_WRONLY == 1
assert posix.O_RDWR == 2
assert posix.F_OK == 0
assert posix.R_OK == 4
assert posix.W_OK == 2
assert posix.X_OK == 1
assert posix.WNOHANG == 1
assert posix.WUNTRACED == 2
assert posix.SEEK_HOLE == 3
assert posix.SEEK_DATA == 4
assert posix.EX_OK == 0
assert posix.SCHED_OTHER == 1
assert posix.SCHED_FIFO == 4
assert posix.SCHED_RR == 2
assert posix.PRIO_PROCESS == 0
assert posix.PRIO_PGRP == 1
assert posix.PRIO_USER == 2
assert posix.TMP_MAX == 308915776

# ── stat_result class ──────────────────────────────────────────────────────────
assert hasattr(posix, 'stat_result')
sr = posix.stat('/')
assert hasattr(sr, 'st_mode')
assert hasattr(sr, 'st_ino')
assert hasattr(sr, 'st_dev')
assert hasattr(sr, 'st_nlink')
assert hasattr(sr, 'st_uid')
assert hasattr(sr, 'st_gid')
assert hasattr(sr, 'st_size')
assert hasattr(sr, 'st_atime')
assert hasattr(sr, 'st_mtime')
assert hasattr(sr, 'st_ctime')
assert isinstance(sr.st_mode, int)
assert isinstance(sr.st_size, int)

lsr = posix.lstat('/')
assert hasattr(lsr, 'st_mode')
assert isinstance(lsr.st_mode, int)

fsr = posix.fstat(0)
assert hasattr(fsr, 'st_mode')
assert isinstance(fsr.st_mode, int)

# ── statvfs_result class ───────────────────────────────────────────────────────
assert hasattr(posix, 'statvfs_result')
svr = posix.statvfs('/')
assert hasattr(svr, 'f_bsize')
assert hasattr(svr, 'f_frsize')
assert hasattr(svr, 'f_blocks')
assert hasattr(svr, 'f_bfree')
assert hasattr(svr, 'f_bavail')
assert hasattr(svr, 'f_files')
assert hasattr(svr, 'f_ffree')
assert hasattr(svr, 'f_favail')
assert hasattr(svr, 'f_flag')
assert hasattr(svr, 'f_namemax')
assert isinstance(svr.f_bsize, int)

# ── uname_result class ─────────────────────────────────────────────────────────
assert hasattr(posix, 'uname_result')
ur = posix.uname()
assert hasattr(ur, 'sysname')
assert hasattr(ur, 'nodename')
assert hasattr(ur, 'release')
assert hasattr(ur, 'version')
assert hasattr(ur, 'machine')
assert isinstance(ur.sysname, str)

# ── times_result class ─────────────────────────────────────────────────────────
assert hasattr(posix, 'times_result')
tr = posix.times()
assert hasattr(tr, 'user')
assert hasattr(tr, 'system')
assert hasattr(tr, 'children_user')
assert hasattr(tr, 'children_system')
assert hasattr(tr, 'elapsed')
assert isinstance(tr.user, float)

# ── terminal_size class ────────────────────────────────────────────────────────
assert hasattr(posix, 'terminal_size')
try:
    ts = posix.get_terminal_size()
    assert hasattr(ts, 'columns')
    assert hasattr(ts, 'lines')
    assert isinstance(ts.columns, int)
    assert isinstance(ts.lines, int)
    assert ts.columns > 0
    assert ts.lines > 0
except OSError:
    pass

# ── waitid_result class ────────────────────────────────────────────────────────
assert hasattr(posix, 'waitid_result')

# ── DirEntry class ─────────────────────────────────────────────────────────────
assert hasattr(posix, 'DirEntry')

# ── fspath ─────────────────────────────────────────────────────────────────────
assert posix.fspath('/tmp') == '/tmp'
assert posix.fspath(b'/tmp') == b'/tmp'

# ── strerror ───────────────────────────────────────────────────────────────────
assert isinstance(posix.strerror(0), str)
assert isinstance(posix.strerror(1), str)
assert len(posix.strerror(1)) > 0

# ── device_encoding ────────────────────────────────────────────────────────────
r = posix.device_encoding(1)
assert r is None or isinstance(r, str)

# ── umask ──────────────────────────────────────────────────────────────────────
r = posix.umask(0o022)
assert isinstance(r, int)
# restore
posix.umask(r)

# ── system ─────────────────────────────────────────────────────────────────────
assert isinstance(posix.system('true'), int)

# ── closerange ─────────────────────────────────────────────────────────────────
assert posix.closerange(100, 100) is None

# ── get_blocking / set_blocking ────────────────────────────────────────────────
assert isinstance(posix.get_blocking(1), bool)
assert posix.set_blocking(1, False) is None

# ── get_inheritable / set_inheritable ─────────────────────────────────────────
assert isinstance(posix.get_inheritable(1), bool)
assert posix.set_inheritable(1, False) is None

# ── major / minor / makedev ───────────────────────────────────────────────────
assert isinstance(posix.major(0), int)
assert isinstance(posix.minor(0), int)
assert isinstance(posix.makedev(0, 0), int)

# ── wait status helpers ────────────────────────────────────────────────────────
assert isinstance(posix.WSTOPSIG(0), int)
assert isinstance(posix.WTERMSIG(0), int)
assert isinstance(posix.WIFEXITED(0), bool)
assert isinstance(posix.WIFSIGNALED(0), bool)
assert isinstance(posix.WIFSTOPPED(0), bool)
assert isinstance(posix.WIFCONTINUED(0), bool)
assert isinstance(posix.WEXITSTATUS(0), int)
assert isinstance(posix.waitstatus_to_exitcode(0), int)

# ── sched helpers ─────────────────────────────────────────────────────────────
assert posix.sched_get_priority_max(posix.SCHED_FIFO) >= 0
assert posix.sched_get_priority_min(posix.SCHED_FIFO) >= 0
assert posix.sched_yield() is None

# ── sync / setpgrp ────────────────────────────────────────────────────────────
assert posix.sync() is None
assert posix.setpgrp() is None

# ── login_tty raises OSError ──────────────────────────────────────────────────
try:
    posix.login_tty(0)
    raised = False
except OSError:
    raised = True
assert raised

# ── pty helpers ───────────────────────────────────────────────────────────────
assert isinstance(posix.posix_openpt(posix.O_RDWR), int)
assert posix.grantpt(3) is None
assert posix.unlockpt(3) is None
assert isinstance(posix.ptsname(3), str)

# ── register_at_fork ──────────────────────────────────────────────────────────
assert callable(posix.register_at_fork)

# ── getgrouplist ──────────────────────────────────────────────────────────────
r = posix.getgrouplist('root', 0)
assert isinstance(r, list)
assert len(r) >= 1

# ── initgroups ────────────────────────────────────────────────────────────────
assert callable(posix.initgroups)

# ── chroot ────────────────────────────────────────────────────────────────────
assert callable(posix.chroot)

# ── posix_spawn / posix_spawnp ────────────────────────────────────────────────
assert callable(posix.posix_spawn)
assert callable(posix.posix_spawnp)

# ── preadv / pwritev / readinto ───────────────────────────────────────────────
assert callable(posix.preadv)
assert callable(posix.pwritev)
assert callable(posix.readinto)

# ── process info ──────────────────────────────────────────────────────────────
assert isinstance(posix.getpid(), int)
assert isinstance(posix.getppid(), int)
assert isinstance(posix.getuid(), int)
assert isinstance(posix.getgid(), int)
assert isinstance(posix.geteuid(), int)
assert isinstance(posix.getegid(), int)
assert isinstance(posix.getlogin(), str)
assert isinstance(posix.getgroups(), list)
la = posix.getloadavg()
assert isinstance(la, tuple) and len(la) == 3

# ── misc ─────────────────────────────────────────────────────────────────────
assert isinstance(posix.getcwd(), str)
assert isinstance(posix.getcwdb(), bytes)
assert isinstance(posix.urandom(4), bytes) and len(posix.urandom(4)) == 4
assert posix.cpu_count() >= 1
assert isinstance(posix.confstr_names, dict)
assert isinstance(posix.sysconf_names, dict)
assert isinstance(posix.pathconf_names, dict)
assert posix.isatty(0) in (True, False)
assert isinstance(posix.ctermid(), str)

print('ok')

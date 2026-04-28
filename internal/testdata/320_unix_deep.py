import posix
import pwd
import grp
import termios
import tty
import pty
import fcntl
import resource
import syslog

# ── posix constants ────────────────────────────────────────────────────────────
print(posix.F_OK == 0)
print(posix.R_OK == 4)
print(posix.W_OK == 2)
print(posix.X_OK == 1)
print(posix.O_RDONLY == 0)
print(posix.O_WRONLY == 1)
print(posix.O_RDWR == 2)
print(posix.WNOHANG == 1)
print(posix.WUNTRACED == 2)
print(posix.EX_OK == 0)
print(posix.EX_USAGE == 64)
print(posix.EX_SOFTWARE == 70)
print(posix.CLD_EXITED == 1)
print(posix.CLD_KILLED == 2)
print(posix.SCHED_OTHER >= 0)
print(posix.SCHED_FIFO >= 0)
print(posix.SCHED_RR >= 0)

# ── posix functions callable ──────────────────────────────────────────────────
print(callable(posix.getpid))
print(callable(posix.getuid))
print(callable(posix.getgid))
print(callable(posix.getcwd))
print(callable(posix.listdir))
print(callable(posix.stat))
print(callable(posix.chmod))
print(callable(posix.chown))
print(callable(posix.link))
print(callable(posix.unlink))
print(callable(posix.rename))
print(callable(posix.mkdir))
print(callable(posix.rmdir))
print(callable(posix.fork))
print(callable(posix.execv))
print(callable(posix.kill))
print(callable(posix.waitpid))
print(callable(posix.open))
print(callable(posix.read))
print(callable(posix.write))
print(callable(posix.close))
print(callable(posix.pipe))
print(callable(posix.dup))
print(callable(posix.lseek))
print(callable(posix.access))
print(callable(posix.urandom))

# ── posix function return types ────────────────────────────────────────────────
print(isinstance(posix.getpid(), int))
print(isinstance(posix.getuid(), int))
print(isinstance(posix.getgid(), int))
print(isinstance(posix.getcwd(), str))
print(isinstance(posix.listdir('/'), list))
print(isinstance(posix.urandom(4), bytes))
print(len(posix.urandom(8)) == 8)

# ── pwd struct_passwd ─────────────────────────────────────────────────────────
print(hasattr(pwd, 'struct_passwd'))
pw = pwd.getpwuid(0)
print(pw.pw_name == 'root')
print(pw.pw_uid == 0)
print(pw.pw_gid == 0)
print(pw.pw_dir == '/var/root')
print(pw.pw_shell == '/bin/sh')

try:
    pwd.getpwnam('_no_such_user_xyz_')
    print(False)
except KeyError:
    print(True)

print(isinstance(pwd.getpwall(), list))

# ── grp struct_group ──────────────────────────────────────────────────────────
print(hasattr(grp, 'struct_group'))
print(callable(grp.getgrgid))
print(callable(grp.getgrnam))
print(callable(grp.getgrall))
print(isinstance(grp.getgrall(), list))

# ── termios constants ──────────────────────────────────────────────────────────
print(termios.TCSANOW == 0)
print(termios.TCSADRAIN == 1)
print(termios.TCSAFLUSH == 2)
print(termios.TCIFLUSH == 1)
print(termios.TCOFLUSH == 2)
print(termios.TCIOFLUSH == 3)
print(termios.VEOF == 0)
print(termios.VINTR == 8)
print(termios.VMIN == 16)
print(termios.VTIME == 17)
print(termios.ECHO == 0x8)
print(termios.ICANON == 0x100)
print(termios.ISIG == 0x80)
print(termios.OPOST == 0x1)
print(termios.CREAD == 0x800)
print(termios.B9600 == 9600)
print(termios.B115200 == 115200)
print(termios.CS8 == 0x300)

# ── termios functions callable ────────────────────────────────────────────────
print(callable(termios.tcgetattr))
print(callable(termios.tcsetattr))
print(callable(termios.tcdrain))
print(callable(termios.tcflush))
print(callable(termios.tcflow))
print(callable(termios.tcsendbreak))
print(callable(termios.tcgetwinsize))
print(callable(termios.tcsetwinsize))

# ── termios tcgetattr structure ────────────────────────────────────────────────
try:
    attr = termios.tcgetattr(0)
    print(isinstance(attr, list))
    print(len(attr) == 7)
    print(isinstance(attr[0], int))
    print(isinstance(attr[6], list))
    print(len(attr[6]) >= 20)
except termios.error:
    print(True)
    print(True)
    print(True)
    print(True)
    print(True)
    attr = [0, 0, 0, 0, 9600, 9600, [0] * 20]

# ── termios tcsetattr / others return None ─────────────────────────────────────
try:
    print(termios.tcsetattr(0, termios.TCSANOW, attr) is None)
except termios.error:
    print(True)
try:
    print(termios.tcdrain(0) is None)
except termios.error:
    print(True)
try:
    print(termios.tcflush(0, termios.TCIOFLUSH) is None)
except termios.error:
    print(True)
try:
    print(termios.tcflow(0, termios.TCOOFF) is None)
except termios.error:
    print(True)
try:
    print(termios.tcsendbreak(0, 0) is None)
except termios.error:
    print(True)

try:
    ws = termios.tcgetwinsize(0)
    print(isinstance(ws, tuple))
    print(len(ws) == 2)
except termios.error:
    print(True)
    print(True)

# ── tty ────────────────────────────────────────────────────────────────────────
print(tty.TCSANOW == 0)
print(tty.TCSAFLUSH == 2)
print(tty.VMIN == 16)
print(callable(tty.setraw))
print(callable(tty.setcbreak))
try:
    print(tty.setraw(0) is None)
except termios.error:
    print(True)
try:
    print(tty.setcbreak(0) is None)
except termios.error:
    print(True)

# ── pty ────────────────────────────────────────────────────────────────────────
print(pty.CHILD == 0)
print(pty.STDIN_FILENO == 0)
print(pty.STDOUT_FILENO == 1)
print(pty.STDERR_FILENO == 2)
print(callable(pty.openpty))
print(callable(pty.fork))
print(callable(pty.spawn))

# ── fcntl constants ────────────────────────────────────────────────────────────
print(fcntl.F_DUPFD == 0)
print(fcntl.F_GETFD == 1)
print(fcntl.F_SETFD == 2)
print(fcntl.F_GETFL == 3)
print(fcntl.F_SETFL == 4)
print(fcntl.F_RDLCK == 1)
print(fcntl.F_WRLCK == 3)
print(fcntl.F_UNLCK == 2)
print(fcntl.FD_CLOEXEC == 1)
print(fcntl.LOCK_SH == 1)
print(fcntl.LOCK_EX == 2)
print(fcntl.LOCK_NB == 4)
print(fcntl.LOCK_UN == 8)

# ── fcntl functions callable ──────────────────────────────────────────────────
print(callable(fcntl.fcntl))
print(callable(fcntl.flock))
print(callable(fcntl.ioctl))
print(callable(fcntl.lockf))
print(fcntl.flock(0, fcntl.LOCK_UN) is None)
r = fcntl.fcntl(0, fcntl.F_GETFL)
print(r is None or isinstance(r, int))
print(fcntl.lockf(0, fcntl.LOCK_UN) is None)

# ── resource constants ────────────────────────────────────────────────────────
print(resource.RLIM_INFINITY > 0)
print(resource.RLIMIT_NOFILE >= 0)
print(resource.RLIMIT_STACK >= 0)
print(resource.RLIMIT_CPU >= 0)
print(resource.RUSAGE_SELF == 0)
print(resource.RUSAGE_CHILDREN == -1)
print(isinstance(resource.RLIM_INFINITY, int))

# ── resource struct_rusage ────────────────────────────────────────────────────
print(hasattr(resource, 'struct_rusage'))
ru = resource.getrusage(resource.RUSAGE_SELF)
print(hasattr(ru, 'ru_utime'))
print(hasattr(ru, 'ru_stime'))
print(hasattr(ru, 'ru_maxrss'))
print(hasattr(ru, 'ru_minflt'))
print(hasattr(ru, 'ru_majflt'))
print(hasattr(ru, 'ru_nvcsw'))
print(hasattr(ru, 'ru_nivcsw'))

# ── resource getrlimit ────────────────────────────────────────────────────────
lim = resource.getrlimit(resource.RLIMIT_NOFILE)
print(isinstance(lim, tuple))
print(len(lim) == 2)
print(resource.setrlimit(resource.RLIMIT_NOFILE, (100, 200)) is None)

# ── resource getpagesize ───────────────────────────────────────────────────────
print(resource.getpagesize() > 0)

# ── syslog constants ───────────────────────────────────────────────────────────
print(syslog.LOG_EMERG == 0)
print(syslog.LOG_ALERT == 1)
print(syslog.LOG_CRIT == 2)
print(syslog.LOG_ERR == 3)
print(syslog.LOG_WARNING == 4)
print(syslog.LOG_NOTICE == 5)
print(syslog.LOG_INFO == 6)
print(syslog.LOG_DEBUG == 7)
print(syslog.LOG_KERN == 0)
print(syslog.LOG_USER == 8)
print(syslog.LOG_DAEMON == 24)
print(syslog.LOG_AUTH == 32)
print(syslog.LOG_SYSLOG == 40)
print(syslog.LOG_LOCAL0 == 128)
print(syslog.LOG_LOCAL7 == 184)
print(syslog.LOG_PID == 1)
print(syslog.LOG_CONS == 2)
print(syslog.LOG_NDELAY == 8)
print(syslog.LOG_PERROR == 32)

# ── syslog functions ───────────────────────────────────────────────────────────
print(callable(syslog.syslog))
print(callable(syslog.openlog))
print(callable(syslog.closelog))
print(callable(syslog.setlogmask))
print(callable(syslog.LOG_MASK))
print(callable(syslog.LOG_UPTO))
print(syslog.syslog(syslog.LOG_INFO, 'test') is None)
print(syslog.openlog('test', syslog.LOG_PID, syslog.LOG_USER) is None)
print(syslog.closelog() is None)
print(isinstance(syslog.setlogmask(0xff), int))
print(syslog.LOG_MASK(syslog.LOG_INFO) == (1 << 6))
print(syslog.LOG_UPTO(syslog.LOG_INFO) == 0x7f)

print('done')

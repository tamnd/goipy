"""Tests for errno module."""
import errno

# errorcode is a dict mapping int -> str
print(isinstance(errno.errorcode, dict))              # True
print(errno.errorcode[errno.EPERM] == 'EPERM')        # True
print(errno.errorcode[errno.ENOENT] == 'ENOENT')      # True
print(errno.errorcode[errno.EINTR] == 'EINTR')        # True

# Basic constants
print(errno.EPERM == 1)       # True
print(errno.ENOENT == 2)      # True
print(errno.ESRCH == 3)       # True
print(errno.EINTR == 4)       # True
print(errno.EIO == 5)         # True
print(errno.EACCES == 13)     # True
print(errno.EINVAL == 22)     # True
print(errno.ENOSYS == 38)     # True

# Aliases: EWOULDBLOCK == EAGAIN, EDEADLOCK == EDEADLK
print(errno.EWOULDBLOCK == errno.EAGAIN)    # True
print(errno.EDEADLOCK == errno.EDEADLK)    # True

# Network errors
print(errno.ENOTSOCK == 88)       # True
print(errno.ECONNREFUSED == 111)  # True
print(errno.ETIMEDOUT == 110)     # True
print(errno.EADDRINUSE == 98)     # True

# errorcode contains many entries
print(len(errno.errorcode) >= 50)   # True

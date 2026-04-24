"""Tests for the platform module."""
import platform

# system() returns a non-empty string
print(isinstance(platform.system(), str) and len(platform.system()) > 0)

# machine() returns a non-empty string
print(isinstance(platform.machine(), str) and len(platform.machine()) > 0)

# node() returns a non-empty string
print(isinstance(platform.node(), str) and len(platform.node()) > 0)

# python_version() returns a version string starting with '3.'
print(platform.python_version().startswith('3.'))

# python_implementation() returns 'CPython'
print(platform.python_implementation() == 'CPython')

# python_version_tuple() is a tuple of length 3
vt = platform.python_version_tuple()
print(isinstance(vt, tuple) and len(vt) == 3)

# uname() returns an object with a system attribute that is a string
u = platform.uname()
print(isinstance(u.system, str))

# platform() returns a non-empty string
print(isinstance(platform.platform(), str) and len(platform.platform()) > 0)

# architecture() returns a tuple of length 2
arch = platform.architecture()
print(isinstance(arch, tuple) and len(arch) == 2)

# win32_ver() returns a tuple of length 4
wv = platform.win32_ver()
print(isinstance(wv, tuple) and len(wv) == 4)

# mac_ver() returns a tuple of length 3
mv = platform.mac_ver()
print(isinstance(mv, tuple) and len(mv) == 3)

# python_build() returns a tuple of length 2
pb = platform.python_build()
print(isinstance(pb, tuple) and len(pb) == 2)

# python_compiler() returns a string
print(isinstance(platform.python_compiler(), str))

# python_branch() returns a string
print(isinstance(platform.python_branch(), str))

# python_revision() returns a string
print(isinstance(platform.python_revision(), str))

# release() returns a string
print(isinstance(platform.release(), str))

# version() returns a string
print(isinstance(platform.version(), str))

# processor() returns a string
print(isinstance(platform.processor(), str))

# uname fields are strings
print(isinstance(u.node, str))
print(isinstance(u.release, str))
print(isinstance(u.version, str))
print(isinstance(u.machine, str))
print(isinstance(u.processor, str))

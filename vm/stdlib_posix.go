package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildPosix() *object.Module {
	m := &object.Module{Name: "posix", Dict: object.NewDict()}
	d := m.Dict

	d.SetStr("error", i.osErr)

	consts := map[string]int64{
		// CLD_* child wait status
		"CLD_EXITED":    1,
		"CLD_KILLED":    2,
		"CLD_DUMPED":    3,
		"CLD_TRAPPED":   4,
		"CLD_STOPPED":   5,
		"CLD_CONTINUED": 6,
		// EX_* exit codes
		"EX_OK":          0,
		"EX_USAGE":       64,
		"EX_DATAERR":     65,
		"EX_NOINPUT":     66,
		"EX_NOUSER":      67,
		"EX_NOHOST":      68,
		"EX_UNAVAILABLE": 69,
		"EX_SOFTWARE":    70,
		"EX_OSERR":       71,
		"EX_OSFILE":      72,
		"EX_CANTCREAT":   73,
		"EX_IOERR":       74,
		"EX_TEMPFAIL":    75,
		"EX_PROTOCOL":    76,
		"EX_NOPERM":      77,
		"EX_CONFIG":      78,
		// F_* lock constants
		"F_LOCK":  1,
		"F_TLOCK": 2,
		"F_ULOCK": 0,
		"F_TEST":  3,
		// access() mode bits
		"F_OK": 0,
		"R_OK": 4,
		"W_OK": 2,
		"X_OK": 1,
		// misc
		"NGROUPS_MAX": 16,
		// O_* open flags
		"O_RDONLY":    0,
		"O_WRONLY":    1,
		"O_RDWR":      2,
		"O_CREAT":     0x200,
		"O_EXCL":      0x800,
		"O_NOCTTY":    0x20000,
		"O_TRUNC":     0x400,
		"O_APPEND":    8,
		"O_NONBLOCK":  4,
		"O_NDELAY":    4,
		"O_DSYNC":     0x400000,
		"O_SYNC":      0x80,
		"O_RSYNC":     0x80,
		"O_DIRECTORY": 0x100000,
		"O_NOFOLLOW":  0x100,
		"O_CLOEXEC":   0x1000000,
		"O_ASYNC":     0x40,
		"O_FSYNC":     0x80,
		"O_NOATIME":   0,
		"O_PATH":      0,
		"O_TMPFILE":   0,
		"O_LARGEFILE": 0,
		"O_ACCMODE":   3,
		"O_SHLOCK":    0x10,
		"O_EXLOCK":    0x20,
		"O_EVTONLY":   0x8000,
		"O_SYMLINK":   0x200000,
		// wait options
		"WNOHANG":   1,
		"WUNTRACED": 2,
		// seek
		"SEEK_SET": 0,
		"SEEK_CUR": 1,
		"SEEK_END": 2,
		"SEEK_HOLE": 3,
		"SEEK_DATA": 4,
		// sched
		"SCHED_OTHER": 1,
		"SCHED_FIFO":  4,
		"SCHED_RR":    2,
		// AT_* flags
		"AT_FDCWD":            -2,
		"AT_SYMLINK_NOFOLLOW": 0x0020,
		"AT_SYMLINK_FOLLOW":   0x0040,
		"AT_REMOVEDIR":        0x0080,
		"AT_EACCESS":          0x0010,
		// POSIX_FADV_* (Linux only, expose as 0 on macOS)
		"POSIX_FADV_NORMAL":     0,
		"POSIX_FADV_SEQUENTIAL": 0,
		"POSIX_FADV_RANDOM":     0,
		"POSIX_FADV_NOREUSE":    0,
		"POSIX_FADV_WILLNEED":   0,
		"POSIX_FADV_DONTNEED":   0,
		// PRIO_*
		"PRIO_PROCESS": 0,
		"PRIO_PGRP":    1,
		"PRIO_USER":    2,
		// RTLD_*
		"RTLD_LAZY":    1,
		"RTLD_NOW":     2,
		"RTLD_GLOBAL":  8,
		"RTLD_LOCAL":   4,
		"RTLD_NODELETE": 0x80,
		"RTLD_NOLOAD":  0x10,
		"RTLD_DEEPBIND": 0,
		// XATTR_* (macOS)
		"XATTR_CREATE":  2,
		"XATTR_REPLACE": 1,
		"XATTR_SIZE_MAX": 65536,
		// MFD_* (Linux memfd, expose as 0)
		"MFD_CLOEXEC":       1,
		"MFD_ALLOW_SEALING": 2,
		// misc extra
		"GRND_NONBLOCK": 1,
		"GRND_RANDOM":   2,
	}
	for name, val := range consts {
		d.SetStr(name, intObj(val))
	}

	noneStub := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		}
	}

	intStub := func(name string, v int64) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return intObj(v), nil
			},
		}
	}

	strStub := func(name string, v string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.Str{V: v}, nil
			},
		}
	}

	// environ dict
	environDict := object.NewDict()
	d.SetStr("environ", &object.Dict{})
	_ = environDict

	// process info
	d.SetStr("getpid", intStub("getpid", 1))
	d.SetStr("getppid", intStub("getppid", 0))
	d.SetStr("getuid", intStub("getuid", 0))
	d.SetStr("geteuid", intStub("geteuid", 0))
	d.SetStr("getgid", intStub("getgid", 0))
	d.SetStr("getegid", intStub("getegid", 0))
	d.SetStr("getpgrp", intStub("getpgrp", 0))
	d.SetStr("getpgid", intStub("getpgid", 0))
	d.SetStr("getsid", intStub("getsid", 0))
	d.SetStr("getgroups", &object.BuiltinFunc{
		Name: "getgroups",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})
	d.SetStr("getresuid", &object.BuiltinFunc{
		Name: "getresuid",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{intObj(0), intObj(0), intObj(0)}}, nil
		},
	})
	d.SetStr("getresgid", &object.BuiltinFunc{
		Name: "getresgid",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{intObj(0), intObj(0), intObj(0)}}, nil
		},
	})
	d.SetStr("getlogin", strStub("getlogin", "root"))
	d.SetStr("getloadavg", &object.BuiltinFunc{
		Name: "getloadavg",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{
				&object.Float{V: 0.0},
				&object.Float{V: 0.0},
				&object.Float{V: 0.0},
			}}, nil
		},
	})

	// cwd / paths
	d.SetStr("getcwd", strStub("getcwd", "/"))
	d.SetStr("getcwdb", &object.BuiltinFunc{
		Name: "getcwdb",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Bytes{V: []byte("/")}, nil
		},
	})
	d.SetStr("chdir", noneStub("chdir"))
	d.SetStr("fchdir", noneStub("fchdir"))
	d.SetStr("listdir", &object.BuiltinFunc{
		Name: "listdir",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})
	d.SetStr("scandir", &object.BuiltinFunc{
		Name: "scandir",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	// file ops
	d.SetStr("open", intStub("open", 3))
	d.SetStr("close", noneStub("close"))
	d.SetStr("read", &object.BuiltinFunc{
		Name: "read",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Bytes{V: []byte{}}, nil
		},
	})
	d.SetStr("write", intStub("write", 0))
	d.SetStr("dup", intStub("dup", 3))
	d.SetStr("dup2", intStub("dup2", 3))
	d.SetStr("lseek", intStub("lseek", 0))
	d.SetStr("fsync", noneStub("fsync"))
	d.SetStr("fdatasync", noneStub("fdatasync"))
	d.SetStr("ftruncate", noneStub("ftruncate"))
	d.SetStr("truncate", noneStub("truncate"))
	d.SetStr("pipe", &object.BuiltinFunc{
		Name: "pipe",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{intObj(3), intObj(4)}}, nil
		},
	})
	d.SetStr("pipe2", &object.BuiltinFunc{
		Name: "pipe2",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{intObj(3), intObj(4)}}, nil
		},
	})
	d.SetStr("isatty", &object.BuiltinFunc{
		Name: "isatty",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		},
	})
	d.SetStr("ttyname", strStub("ttyname", "/dev/tty"))
	d.SetStr("openat", intStub("openat", 3))
	d.SetStr("readv", intStub("readv", 0))
	d.SetStr("writev", intStub("writev", 0))
	d.SetStr("pread", &object.BuiltinFunc{
		Name: "pread",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Bytes{V: []byte{}}, nil
		},
	})
	d.SetStr("pwrite", intStub("pwrite", 0))
	d.SetStr("sendfile", intStub("sendfile", 0))
	d.SetStr("copy_file_range", intStub("copy_file_range", 0))
	d.SetStr("splice", intStub("splice", 0))

	// stat / lstat
	d.SetStr("stat", noneStub("stat"))
	d.SetStr("lstat", noneStub("lstat"))
	d.SetStr("fstat", noneStub("fstat"))
	d.SetStr("stat_result", noneStub("stat_result"))

	// chmod / chown
	d.SetStr("chmod", noneStub("chmod"))
	d.SetStr("fchmod", noneStub("fchmod"))
	d.SetStr("lchmod", noneStub("lchmod"))
	d.SetStr("chown", noneStub("chown"))
	d.SetStr("fchown", noneStub("fchown"))
	d.SetStr("lchown", noneStub("lchown"))
	d.SetStr("chflags", noneStub("chflags"))
	d.SetStr("lchflags", noneStub("lchflags"))

	// link / rename / remove
	d.SetStr("link", noneStub("link"))
	d.SetStr("symlink", noneStub("symlink"))
	d.SetStr("unlink", noneStub("unlink"))
	d.SetStr("remove", noneStub("remove"))
	d.SetStr("rename", noneStub("rename"))
	d.SetStr("renames", noneStub("renames"))
	d.SetStr("replace", noneStub("replace"))
	d.SetStr("readlink", strStub("readlink", ""))
	d.SetStr("mkdir", noneStub("mkdir"))
	d.SetStr("makedirs", noneStub("makedirs"))
	d.SetStr("rmdir", noneStub("rmdir"))
	d.SetStr("removedirs", noneStub("removedirs"))
	d.SetStr("mkfifo", noneStub("mkfifo"))
	d.SetStr("mknod", noneStub("mknod"))

	// access
	d.SetStr("access", &object.BuiltinFunc{
		Name: "access",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		},
	})

	// xattr
	d.SetStr("getxattr", &object.BuiltinFunc{
		Name: "getxattr",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Bytes{V: []byte{}}, nil
		},
	})
	d.SetStr("setxattr", noneStub("setxattr"))
	d.SetStr("removexattr", noneStub("removexattr"))
	d.SetStr("listxattr", &object.BuiltinFunc{
		Name: "listxattr",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	// process control
	d.SetStr("fork", intStub("fork", 0))
	d.SetStr("forkpty", &object.BuiltinFunc{
		Name: "forkpty",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{intObj(0), intObj(-1)}}, nil
		},
	})
	d.SetStr("execv", noneStub("execv"))
	d.SetStr("execve", noneStub("execve"))
	d.SetStr("execvp", noneStub("execvp"))
	d.SetStr("execvpe", noneStub("execvpe"))
	d.SetStr("_execvpe", noneStub("_execvpe"))
	d.SetStr("spawnv", intStub("spawnv", -1))
	d.SetStr("spawnve", intStub("spawnve", -1))
	d.SetStr("spawnvp", intStub("spawnvp", -1))
	d.SetStr("spawnvpe", intStub("spawnvpe", -1))
	d.SetStr("waitpid", &object.BuiltinFunc{
		Name: "waitpid",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{intObj(0), intObj(0)}}, nil
		},
	})
	d.SetStr("wait", &object.BuiltinFunc{
		Name: "wait",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{intObj(0), intObj(0)}}, nil
		},
	})
	d.SetStr("waitid", noneStub("waitid"))
	d.SetStr("wait3", &object.BuiltinFunc{
		Name: "wait3",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{intObj(0), intObj(0), object.None}}, nil
		},
	})
	d.SetStr("wait4", &object.BuiltinFunc{
		Name: "wait4",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{intObj(0), intObj(0), object.None}}, nil
		},
	})
	d.SetStr("WIFEXITED", &object.BuiltinFunc{
		Name: "WIFEXITED",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		},
	})
	d.SetStr("WEXITSTATUS", intStub("WEXITSTATUS", 0))
	d.SetStr("WIFSIGNALED", &object.BuiltinFunc{
		Name: "WIFSIGNALED",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		},
	})
	d.SetStr("WTERMSIG", intStub("WTERMSIG", 0))
	d.SetStr("WIFSTOPPED", &object.BuiltinFunc{
		Name: "WIFSTOPPED",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		},
	})
	d.SetStr("WSTOPSIG", intStub("WSTOPSIG", 0))
	d.SetStr("WIFCONTINUED", &object.BuiltinFunc{
		Name: "WIFCONTINUED",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		},
	})
	d.SetStr("WCOREDUMP", &object.BuiltinFunc{
		Name: "WCOREDUMP",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		},
	})

	// signals
	d.SetStr("kill", noneStub("kill"))
	d.SetStr("killpg", noneStub("killpg"))
	d.SetStr("raise_signal", noneStub("raise_signal"))
	d.SetStr("abort", noneStub("abort"))
	d.SetStr("setpgid", noneStub("setpgid"))
	d.SetStr("setsid", intStub("setsid", 0))
	d.SetStr("setuid", noneStub("setuid"))
	d.SetStr("seteuid", noneStub("seteuid"))
	d.SetStr("setgid", noneStub("setgid"))
	d.SetStr("setegid", noneStub("setegid"))
	d.SetStr("setreuid", noneStub("setreuid"))
	d.SetStr("setregid", noneStub("setregid"))
	d.SetStr("setresuid", noneStub("setresuid"))
	d.SetStr("setresgid", noneStub("setresgid"))
	d.SetStr("setgroups", noneStub("setgroups"))

	// env
	d.SetStr("putenv", noneStub("putenv"))
	d.SetStr("unsetenv", noneStub("unsetenv"))
	d.SetStr("getenv", &object.BuiltinFunc{
		Name: "getenv",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	d.SetStr("getenvb", &object.BuiltinFunc{
		Name: "getenvb",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// misc
	d.SetStr("strerror", strStub("strerror", "Unknown error"))
	d.SetStr("urandom", &object.BuiltinFunc{
		Name: "urandom",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			n := int64(0)
			if len(a) > 0 {
				if v, ok := a[0].(*object.Int); ok {
					n = v.V.Int64()
				}
			}
			return &object.Bytes{V: make([]byte, n)}, nil
		},
	})
	d.SetStr("getrandom", &object.BuiltinFunc{
		Name: "getrandom",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			n := int64(0)
			if len(a) > 0 {
				if v, ok := a[0].(*object.Int); ok {
					n = v.V.Int64()
				}
			}
			return &object.Bytes{V: make([]byte, n)}, nil
		},
	})
	d.SetStr("times", &object.BuiltinFunc{
		Name: "times",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{
				&object.Float{V: 0.0}, &object.Float{V: 0.0},
				&object.Float{V: 0.0}, &object.Float{V: 0.0},
				&object.Float{V: 0.0},
			}}, nil
		},
	})
	d.SetStr("uname", &object.BuiltinFunc{
		Name: "uname",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{
				&object.Str{V: "Darwin"},
				&object.Str{V: "localhost"},
				&object.Str{V: "0.0.0"},
				&object.Str{V: ""},
				&object.Str{V: "arm64"},
			}}, nil
		},
	})
	d.SetStr("uname_result", noneStub("uname_result"))
	d.SetStr("confstr", strStub("confstr", ""))
	d.SetStr("confstr_names", &object.BuiltinFunc{
		Name: "confstr_names",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewDict(), nil
		},
	})
	d.SetStr("sysconf", intStub("sysconf", -1))
	d.SetStr("sysconf_names", &object.BuiltinFunc{
		Name: "sysconf_names",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewDict(), nil
		},
	})
	d.SetStr("pathconf", intStub("pathconf", -1))
	d.SetStr("pathconf_names", &object.BuiltinFunc{
		Name: "pathconf_names",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewDict(), nil
		},
	})
	d.SetStr("fpathconf", intStub("fpathconf", -1))
	d.SetStr("statvfs", noneStub("statvfs"))
	d.SetStr("fstatvfs", noneStub("fstatvfs"))
	d.SetStr("statvfs_result", noneStub("statvfs_result"))
	d.SetStr("getpriority", intStub("getpriority", 0))
	d.SetStr("setpriority", noneStub("setpriority"))
	d.SetStr("nice", intStub("nice", 0))
	d.SetStr("ctermid", strStub("ctermid", "/dev/tty"))
	d.SetStr("cpu_count", intStub("cpu_count", 1))
	d.SetStr("get_terminal_size", &object.BuiltinFunc{
		Name: "get_terminal_size",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{intObj(80), intObj(24)}}, nil
		},
	})
	d.SetStr("terminal_size", noneStub("terminal_size"))
	d.SetStr("SEEK_HOLE", intObj(3))
	d.SetStr("SEEK_DATA", intObj(4))
	d.SetStr("supports_bytes_environ", object.False)
	d.SetStr("supports_dir_fd", &object.List{V: []object.Object{}})
	d.SetStr("supports_effective_ids", &object.List{V: []object.Object{}})
	d.SetStr("supports_fd", &object.List{V: []object.Object{}})
	d.SetStr("supports_follow_symlinks", &object.List{V: []object.Object{}})
	d.SetStr("curdir", &object.Str{V: "."})
	d.SetStr("pardir", &object.Str{V: ".."})
	d.SetStr("sep", &object.Str{V: "/"})
	d.SetStr("altsep", object.None)
	d.SetStr("extsep", &object.Str{V: "."})
	d.SetStr("pathsep", &object.Str{V: ":"})
	d.SetStr("defpath", &object.Str{V: "/bin:/usr/bin"})
	d.SetStr("devnull", &object.Str{V: "/dev/null"})
	d.SetStr("linesep", &object.Str{V: "\n"})
	d.SetStr("name", &object.Str{V: "posix"})

	return m
}

package vm

import (
	"fmt"

	"github.com/tamnd/goipy/object"
)

// posixStatResult builds a stat_result instance with all fields zeroed.
func posixStatResult(cls *object.Class) *object.Instance {
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	for _, f := range []string{
		"st_mode", "st_ino", "st_dev", "st_nlink", "st_uid", "st_gid",
		"st_size", "st_atime", "st_mtime", "st_ctime",
		"st_atime_ns", "st_mtime_ns", "st_ctime_ns",
		"st_blocks", "st_blksize", "st_flags", "st_gen",
		"st_birthtime", "st_birthtime_ns",
	} {
		if f == "st_atime" || f == "st_mtime" || f == "st_ctime" ||
			f == "st_birthtime" {
			inst.Dict.SetStr(f, &object.Float{V: 0.0})
		} else {
			inst.Dict.SetStr(f, intObj(0))
		}
	}
	return inst
}

func (i *Interp) buildPosix() *object.Module {
	m := &object.Module{Name: "posix", Dict: object.NewDict()}
	d := m.Dict

	d.SetStr("error", i.osErr)

	consts := map[string]int64{
		// CLD_* child wait status
		"CLD_EXITED": 1, "CLD_KILLED": 2, "CLD_DUMPED": 3,
		"CLD_TRAPPED": 4, "CLD_STOPPED": 5, "CLD_CONTINUED": 6,
		// EX_* exit codes
		"EX_OK": 0, "EX_USAGE": 64, "EX_DATAERR": 65, "EX_NOINPUT": 66,
		"EX_NOUSER": 67, "EX_NOHOST": 68, "EX_UNAVAILABLE": 69,
		"EX_SOFTWARE": 70, "EX_OSERR": 71, "EX_OSFILE": 72,
		"EX_CANTCREAT": 73, "EX_IOERR": 74, "EX_TEMPFAIL": 75,
		"EX_PROTOCOL": 76, "EX_NOPERM": 77, "EX_CONFIG": 78,
		// F_* lock
		"F_LOCK": 1, "F_TLOCK": 2, "F_ULOCK": 0, "F_TEST": 3,
		// access bits
		"F_OK": 0, "R_OK": 4, "W_OK": 2, "X_OK": 1,
		// misc
		"NGROUPS_MAX": 16,
		"TMP_MAX":     308915776,
		// O_* open flags
		"O_RDONLY": 0, "O_WRONLY": 1, "O_RDWR": 2,
		"O_CREAT": 0x200, "O_EXCL": 0x800, "O_NOCTTY": 0x20000,
		"O_TRUNC": 0x400, "O_APPEND": 8, "O_NONBLOCK": 4,
		"O_NDELAY": 4, "O_DSYNC": 0x400000, "O_SYNC": 0x80,
		"O_RSYNC": 0x80, "O_DIRECTORY": 0x100000, "O_NOFOLLOW": 0x100,
		"O_CLOEXEC": 0x1000000, "O_ASYNC": 0x40, "O_FSYNC": 0x80,
		"O_NOATIME": 0, "O_PATH": 0, "O_TMPFILE": 0, "O_LARGEFILE": 0,
		"O_ACCMODE": 3, "O_SHLOCK": 0x10, "O_EXLOCK": 0x20,
		"O_EVTONLY": 0x8000, "O_SYMLINK": 0x200000,
		"O_EXEC": 0x40000000, "O_SEARCH": 0x40000000,
		"O_NOFOLLOW_ANY": 0x20000000,
		// wait options
		"WNOHANG": 1, "WUNTRACED": 2,
		"WCONTINUED": 16, "WEXITED": 4, "WNOWAIT": 32, "WSTOPPED": 8,
		// P_* idtype constants
		"P_ALL": 0, "P_PID": 1, "P_PGID": 2,
		// seek
		"SEEK_HOLE": 3, "SEEK_DATA": 4,
		// sched
		"SCHED_OTHER": 1, "SCHED_FIFO": 4, "SCHED_RR": 2,
		// AT_* flags
		"AT_FDCWD": -2, "AT_SYMLINK_NOFOLLOW": 0x0020,
		"AT_SYMLINK_FOLLOW": 0x0040, "AT_REMOVEDIR": 0x0080,
		"AT_EACCESS": 0x0010,
		// POSIX_SPAWN_*
		"POSIX_SPAWN_OPEN": 0, "POSIX_SPAWN_CLOSE": 1, "POSIX_SPAWN_DUP2": 2,
		// PRIO_*
		"PRIO_PROCESS": 0, "PRIO_PGRP": 1, "PRIO_USER": 2,
		"PRIO_DARWIN_PROCESS": 4, "PRIO_DARWIN_THREAD": 3,
		"PRIO_DARWIN_BG": 4096, "PRIO_DARWIN_NONUI": 4097,
		// RTLD_*
		"RTLD_LAZY": 1, "RTLD_NOW": 2, "RTLD_GLOBAL": 8, "RTLD_LOCAL": 4,
		"RTLD_NODELETE": 0x80, "RTLD_NOLOAD": 0x10,
		// ST_* statvfs flags
		"ST_RDONLY": 1, "ST_NOSUID": 2,
		// XATTR_*
		"XATTR_CREATE": 2, "XATTR_REPLACE": 1, "XATTR_SIZE_MAX": 65536,
		// GRND_*
		"GRND_NONBLOCK": 1, "GRND_RANDOM": 2,
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
	boolStub := func(name string, v bool) *object.BuiltinFunc {
		b := object.False
		if v {
			b = object.True
		}
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return b, nil
			},
		}
	}
	intArgStub := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if len(a) > 0 {
					if n, ok := a[0].(*object.Int); ok {
						return intObj(n.V.Int64()), nil
					}
				}
				return intObj(0), nil
			},
		}
	}

	// ── named result classes ──────────────────────────────────────────────────

	// stat_result
	statCls := &object.Class{Name: "stat_result", Bases: []*object.Class{}, Dict: object.NewDict()}
	statFields := []string{
		"st_mode", "st_ino", "st_dev", "st_nlink", "st_uid",
		"st_gid", "st_size", "st_atime", "st_mtime", "st_ctime",
	}
	statCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			// init from tuple/list a[1] or positional
			var vals []object.Object
			if len(a) > 1 {
				switch v := a[1].(type) {
				case *object.List:
					vals = v.V
				case *object.Tuple:
					vals = v.V
				}
			}
			for idx, f := range statFields {
				if idx < len(vals) {
					inst.Dict.SetStr(f, vals[idx])
				} else {
					if f == "st_atime" || f == "st_mtime" || f == "st_ctime" {
						inst.Dict.SetStr(f, &object.Float{V: 0.0})
					} else {
						inst.Dict.SetStr(f, intObj(0))
					}
				}
			}
			// extended attrs default to 0
			for _, f := range []string{
				"st_atime_ns", "st_mtime_ns", "st_ctime_ns",
				"st_blocks", "st_blksize", "st_flags", "st_gen",
				"st_birthtime", "st_birthtime_ns",
			} {
				inst.Dict.SetStr(f, intObj(0))
			}
			return object.None, nil
		},
	})
	d.SetStr("stat_result", statCls)

	// statvfs_result
	statvfsCls := &object.Class{Name: "statvfs_result", Bases: []*object.Class{}, Dict: object.NewDict()}
	statvfsFields := []string{
		"f_bsize", "f_frsize", "f_blocks", "f_bfree", "f_bavail",
		"f_files", "f_ffree", "f_favail", "f_flag", "f_namemax",
	}
	statvfsCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			var vals []object.Object
			if len(a) > 1 {
				switch v := a[1].(type) {
				case *object.List:
					vals = v.V
				case *object.Tuple:
					vals = v.V
				}
			}
			for idx, f := range statvfsFields {
				if idx < len(vals) {
					inst.Dict.SetStr(f, vals[idx])
				} else {
					inst.Dict.SetStr(f, intObj(0))
				}
			}
			return object.None, nil
		},
	})
	d.SetStr("statvfs_result", statvfsCls)

	// uname_result
	unameCls := &object.Class{Name: "uname_result", Bases: []*object.Class{}, Dict: object.NewDict()}
	unameFields := []string{"sysname", "nodename", "release", "version", "machine"}
	unameCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			for idx, f := range unameFields {
				if idx+1 < len(a) {
					inst.Dict.SetStr(f, a[idx+1])
				} else {
					inst.Dict.SetStr(f, &object.Str{V: ""})
				}
			}
			return object.None, nil
		},
	})
	d.SetStr("uname_result", unameCls)

	// times_result
	timesCls := &object.Class{Name: "times_result", Bases: []*object.Class{}, Dict: object.NewDict()}
	timesFields := []string{"user", "system", "children_user", "children_system", "elapsed"}
	timesCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			for idx, f := range timesFields {
				if idx+1 < len(a) {
					inst.Dict.SetStr(f, a[idx+1])
				} else {
					inst.Dict.SetStr(f, &object.Float{V: 0.0})
				}
			}
			return object.None, nil
		},
	})
	d.SetStr("times_result", timesCls)

	// terminal_size
	termSizeCls := &object.Class{Name: "terminal_size", Bases: []*object.Class{}, Dict: object.NewDict()}
	termSizeCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			cols := int64(80)
			rows := int64(24)
			if len(a) > 1 {
				switch v := a[1].(type) {
				case *object.Tuple:
					if len(v.V) >= 2 {
						if n, ok2 := v.V[0].(*object.Int); ok2 {
							cols = n.V.Int64()
						}
						if n, ok2 := v.V[1].(*object.Int); ok2 {
							rows = n.V.Int64()
						}
					}
				case *object.List:
					if len(v.V) >= 2 {
						if n, ok2 := v.V[0].(*object.Int); ok2 {
							cols = n.V.Int64()
						}
						if n, ok2 := v.V[1].(*object.Int); ok2 {
							rows = n.V.Int64()
						}
					}
				}
			}
			inst.Dict.SetStr("columns", intObj(cols))
			inst.Dict.SetStr("lines", intObj(rows))
			return object.None, nil
		},
	})
	d.SetStr("terminal_size", termSizeCls)

	// waitid_result
	waitidCls := &object.Class{Name: "waitid_result", Bases: []*object.Class{}, Dict: object.NewDict()}
	waitidCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			for _, f := range []string{"si_pid", "si_uid", "si_signo", "si_status", "si_code"} {
				inst.Dict.SetStr(f, intObj(0))
			}
			return object.None, nil
		},
	})
	d.SetStr("waitid_result", waitidCls)

	// DirEntry stub class
	dirEntryCls := &object.Class{Name: "DirEntry", Bases: []*object.Class{}, Dict: object.NewDict()}
	dirEntryCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				if inst, ok := a[0].(*object.Instance); ok {
					inst.Dict.SetStr("name", &object.Str{V: ""})
					inst.Dict.SetStr("path", &object.Str{V: ""})
				}
			}
			return object.None, nil
		},
	})
	d.SetStr("DirEntry", dirEntryCls)

	// ── helper to build a stat_result instance ────────────────────────────────
	mkStat := func() *object.Instance {
		return posixStatResult(statCls)
	}

	// ── environ dict ──────────────────────────────────────────────────────────
	d.SetStr("environ", object.NewDict())

	// ── process info ──────────────────────────────────────────────────────────
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
				&object.Float{V: 0.0}, &object.Float{V: 0.0}, &object.Float{V: 0.0},
			}}, nil
		},
	})
	d.SetStr("getpriority", intStub("getpriority", 0))
	d.SetStr("setpriority", noneStub("setpriority"))
	d.SetStr("nice", intStub("nice", 0))

	// ── cwd / paths ──────────────────────────────────────────────────────────
	d.SetStr("getcwd", strStub("getcwd", "/"))
	d.SetStr("getcwdb", &object.BuiltinFunc{
		Name: "getcwdb",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Bytes{V: []byte("/")}, nil
		},
	})
	d.SetStr("chdir", noneStub("chdir"))
	d.SetStr("fchdir", noneStub("fchdir"))
	d.SetStr("chroot", noneStub("chroot"))
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

	// ── file ops ─────────────────────────────────────────────────────────────
	d.SetStr("open", intStub("open", 3))
	d.SetStr("close", noneStub("close"))
	d.SetStr("closerange", noneStub("closerange"))
	d.SetStr("read", &object.BuiltinFunc{
		Name: "read",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Bytes{V: []byte{}}, nil
		},
	})
	d.SetStr("readinto", intStub("readinto", 0))
	d.SetStr("write", intStub("write", 0))
	d.SetStr("writev", intStub("writev", 0))
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
	d.SetStr("isatty", boolStub("isatty", false))
	d.SetStr("ttyname", strStub("ttyname", "/dev/tty"))
	d.SetStr("ctermid", strStub("ctermid", "/dev/tty"))
	d.SetStr("openat", intStub("openat", 3))
	d.SetStr("readv", intStub("readv", 0))
	d.SetStr("pread", &object.BuiltinFunc{
		Name: "pread",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			n := int64(0)
			a = mpArgs(a)
			if len(a) > 1 {
				if v, ok := a[1].(*object.Int); ok {
					n = v.V.Int64()
				}
			}
			return &object.Bytes{V: make([]byte, n)}, nil
		},
	})
	d.SetStr("preadv", &object.BuiltinFunc{
		Name: "preadv",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj(0), nil
		},
	})
	d.SetStr("pwrite", intStub("pwrite", 0))
	d.SetStr("pwritev", intStub("pwritev", 0))
	d.SetStr("sendfile", intStub("sendfile", 0))
	d.SetStr("copy_file_range", intStub("copy_file_range", 0))
	d.SetStr("get_blocking", boolStub("get_blocking", false))
	d.SetStr("set_blocking", noneStub("set_blocking"))
	d.SetStr("get_inheritable", boolStub("get_inheritable", false))
	d.SetStr("set_inheritable", noneStub("set_inheritable"))
	d.SetStr("device_encoding", &object.BuiltinFunc{
		Name: "device_encoding",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── stat / lstat ──────────────────────────────────────────────────────────
	d.SetStr("stat", &object.BuiltinFunc{
		Name: "stat",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return mkStat(), nil
		},
	})
	d.SetStr("lstat", &object.BuiltinFunc{
		Name: "lstat",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return mkStat(), nil
		},
	})
	d.SetStr("fstat", &object.BuiltinFunc{
		Name: "fstat",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return mkStat(), nil
		},
	})

	// ── statvfs ───────────────────────────────────────────────────────────────
	mkStatvfs := func() *object.Instance {
		inst := &object.Instance{Class: statvfsCls, Dict: object.NewDict()}
		for _, f := range statvfsFields {
			inst.Dict.SetStr(f, intObj(0))
		}
		return inst
	}
	d.SetStr("statvfs", &object.BuiltinFunc{
		Name: "statvfs",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return mkStatvfs(), nil
		},
	})
	d.SetStr("fstatvfs", &object.BuiltinFunc{
		Name: "fstatvfs",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return mkStatvfs(), nil
		},
	})

	// ── uname ─────────────────────────────────────────────────────────────────
	d.SetStr("uname", &object.BuiltinFunc{
		Name: "uname",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			inst := &object.Instance{Class: unameCls, Dict: object.NewDict()}
			inst.Dict.SetStr("sysname", &object.Str{V: "Darwin"})
			inst.Dict.SetStr("nodename", &object.Str{V: "localhost"})
			inst.Dict.SetStr("release", &object.Str{V: "0.0.0"})
			inst.Dict.SetStr("version", &object.Str{V: ""})
			inst.Dict.SetStr("machine", &object.Str{V: "arm64"})
			return inst, nil
		},
	})

	// ── times ─────────────────────────────────────────────────────────────────
	d.SetStr("times", &object.BuiltinFunc{
		Name: "times",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			inst := &object.Instance{Class: timesCls, Dict: object.NewDict()}
			for _, f := range timesFields {
				inst.Dict.SetStr(f, &object.Float{V: 0.0})
			}
			return inst, nil
		},
	})

	// ── terminal_size / get_terminal_size ─────────────────────────────────────
	d.SetStr("get_terminal_size", &object.BuiltinFunc{
		Name: "get_terminal_size",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			inst := &object.Instance{Class: termSizeCls, Dict: object.NewDict()}
			inst.Dict.SetStr("columns", intObj(80))
			inst.Dict.SetStr("lines", intObj(24))
			return inst, nil
		},
	})

	// ── chmod / chown ─────────────────────────────────────────────────────────
	d.SetStr("chmod", noneStub("chmod"))
	d.SetStr("fchmod", noneStub("fchmod"))
	d.SetStr("lchmod", noneStub("lchmod"))
	d.SetStr("chown", noneStub("chown"))
	d.SetStr("fchown", noneStub("fchown"))
	d.SetStr("lchown", noneStub("lchown"))
	d.SetStr("chflags", noneStub("chflags"))
	d.SetStr("lchflags", noneStub("lchflags"))
	d.SetStr("umask", intArgStub("umask"))

	// ── link / rename / remove ────────────────────────────────────────────────
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
	d.SetStr("utime", noneStub("utime"))
	d.SetStr("sync", noneStub("sync"))

	// ── access ───────────────────────────────────────────────────────────────
	d.SetStr("access", boolStub("access", false))

	// ── xattr ────────────────────────────────────────────────────────────────
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

	// ── process control ───────────────────────────────────────────────────────
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
	d.SetStr("posix_spawn", intStub("posix_spawn", -1))
	d.SetStr("posix_spawnp", intStub("posix_spawnp", -1))
	d.SetStr("spawnv", intStub("spawnv", -1))
	d.SetStr("spawnve", intStub("spawnve", -1))
	d.SetStr("spawnvp", intStub("spawnvp", -1))
	d.SetStr("spawnvpe", intStub("spawnvpe", -1))
	d.SetStr("system", intStub("system", 0))
	d.SetStr("abort", noneStub("abort"))
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
	d.SetStr("waitid", &object.BuiltinFunc{
		Name: "waitid",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
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
	d.SetStr("waitstatus_to_exitcode", intStub("waitstatus_to_exitcode", 0))

	// ── wait status predicates / extractors ──────────────────────────────────
	d.SetStr("WIFEXITED", boolStub("WIFEXITED", true))
	d.SetStr("WEXITSTATUS", intArgStub("WEXITSTATUS"))
	d.SetStr("WIFSIGNALED", boolStub("WIFSIGNALED", false))
	d.SetStr("WTERMSIG", intStub("WTERMSIG", 0))
	d.SetStr("WIFSTOPPED", boolStub("WIFSTOPPED", false))
	d.SetStr("WSTOPSIG", intStub("WSTOPSIG", 0))
	d.SetStr("WIFCONTINUED", boolStub("WIFCONTINUED", false))
	d.SetStr("WCOREDUMP", boolStub("WCOREDUMP", false))

	// ── signals ───────────────────────────────────────────────────────────────
	d.SetStr("kill", noneStub("kill"))
	d.SetStr("killpg", noneStub("killpg"))
	d.SetStr("raise_signal", noneStub("raise_signal"))
	d.SetStr("setpgid", noneStub("setpgid"))
	d.SetStr("setpgrp", noneStub("setpgrp"))
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
	d.SetStr("initgroups", noneStub("initgroups"))
	d.SetStr("getgrouplist", &object.BuiltinFunc{
		Name: "getgrouplist",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			gid := object.Object(intObj(0))
			if len(a) > 1 {
				gid = a[1]
			}
			return &object.List{V: []object.Object{gid}}, nil
		},
	})
	d.SetStr("tcgetpgrp", intStub("tcgetpgrp", 0))
	d.SetStr("tcsetpgrp", noneStub("tcsetpgrp"))
	d.SetStr("login_tty", &object.BuiltinFunc{
		Name: "login_tty",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "[Errno 6] Device not configured")
		},
	})
	d.SetStr("register_at_fork", noneStub("register_at_fork"))

	// ── pty helpers ───────────────────────────────────────────────────────────
	d.SetStr("openpty", &object.BuiltinFunc{
		Name: "openpty",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "[Errno 38] Function not implemented")
		},
	})
	d.SetStr("posix_openpt", intStub("posix_openpt", 3))
	d.SetStr("grantpt", noneStub("grantpt"))
	d.SetStr("unlockpt", noneStub("unlockpt"))
	d.SetStr("ptsname", strStub("ptsname", "/dev/pts/0"))

	// ── env ───────────────────────────────────────────────────────────────────
	d.SetStr("putenv", noneStub("putenv"))
	d.SetStr("unsetenv", noneStub("unsetenv"))
	d.SetStr("getenv", &object.BuiltinFunc{
		Name: "getenv",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── misc ─────────────────────────────────────────────────────────────────
	d.SetStr("strerror", &object.BuiltinFunc{
		Name: "strerror",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			code := int64(0)
			if len(a) > 0 {
				if n, ok := a[0].(*object.Int); ok {
					code = n.V.Int64()
				}
			}
			if code == 0 {
				return &object.Str{V: "Undefined error: 0"}, nil
			}
			return &object.Str{V: fmt.Sprintf("Unknown error %d", code)}, nil
		},
	})
	d.SetStr("urandom", &object.BuiltinFunc{
		Name: "urandom",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
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
			a = mpArgs(a)
			n := int64(0)
			if len(a) > 0 {
				if v, ok := a[0].(*object.Int); ok {
					n = v.V.Int64()
				}
			}
			return &object.Bytes{V: make([]byte, n)}, nil
		},
	})
	d.SetStr("confstr", strStub("confstr", ""))
	d.SetStr("confstr_names", object.NewDict())
	d.SetStr("sysconf", intStub("sysconf", -1))
	d.SetStr("sysconf_names", object.NewDict())
	d.SetStr("pathconf", intStub("pathconf", -1))
	d.SetStr("pathconf_names", object.NewDict())
	d.SetStr("fpathconf", intStub("fpathconf", -1))
	d.SetStr("cpu_count", intStub("cpu_count", 1))
	d.SetStr("ctermid", strStub("ctermid", "/dev/tty"))
	d.SetStr("fspath", &object.BuiltinFunc{
		Name: "fspath",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) > 0 {
				switch v := a[0].(type) {
				case *object.Str:
					return v, nil
				case *object.Bytes:
					return v, nil
				}
				return a[0], nil
			}
			return &object.Str{V: ""}, nil
		},
	})
	d.SetStr("major", intArgStub("major"))
	d.SetStr("minor", intArgStub("minor"))
	d.SetStr("makedev", intStub("makedev", 0))
	d.SetStr("sched_get_priority_max", intStub("sched_get_priority_max", 47))
	d.SetStr("sched_get_priority_min", intStub("sched_get_priority_min", 0))
	d.SetStr("sched_yield", noneStub("sched_yield"))

	return m
}

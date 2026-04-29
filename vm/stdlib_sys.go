package vm

import (
	"io"
	"runtime"

	"github.com/tamnd/goipy/object"
)

// buildSys exposes a read-only view of interpreter state as the `sys`
// module. Writes to sys.argv/sys.path mutate the underlying slices;
// re-assigning the module attribute itself (e.g. `sys.stdout = x`)
// is not supported.
func (i *Interp) buildSys() *object.Module {
	m := &object.Module{Name: "sys", Dict: object.NewDict()}

	argv := &object.List{}
	for _, s := range i.Argv {
		argv.V = append(argv.V, &object.Str{V: s})
	}
	m.Dict.SetStr("argv", argv)

	path := &object.List{}
	for _, s := range i.SearchPath {
		path.V = append(path.V, &object.Str{V: s})
	}
	m.Dict.SetStr("path", path)

	modules := object.NewDict()
	modules.SetStr("sys", m)
	for name, mod := range i.modules {
		modules.SetStr(name, mod)
	}
	m.Dict.SetStr("modules", modules)

	verInfo := &object.Tuple{V: []object.Object{
		object.NewInt(3), object.NewInt(14), object.NewInt(0),
		&object.Str{V: "final"}, object.NewInt(0),
	}}
	m.Dict.SetStr("version_info", verInfo)
	m.Dict.SetStr("version", &object.Str{V: "3.14.0 (goipy)"})
	m.Dict.SetStr("platform", &object.Str{V: runtime.GOOS})
	m.Dict.SetStr("byteorder", &object.Str{V: "little"})
	m.Dict.SetStr("maxsize", object.NewInt(1<<63-1))
	m.Dict.SetStr("executable", &object.Str{V: "goipy"})
	m.Dict.SetStr("implementation", &object.Str{V: "goipy"})

	m.Dict.SetStr("stdout", &object.TextStream{Name: "stdout", W: i.Stdout})
	m.Dict.SetStr("stderr", &object.TextStream{Name: "stderr", W: i.Stderr})

	m.Dict.SetStr("exit", &object.BuiltinFunc{Name: "exit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var code object.Object = object.NewInt(0)
		if len(a) > 0 {
			code = a[0]
		}
		e := &object.Exception{
			Class: i.systemExit,
			Args:  &object.Tuple{V: []object.Object{code}},
		}
		if s, ok := code.(*object.Str); ok {
			e.Msg = s.V
		}
		return nil, e
	}})

	m.Dict.SetStr("_getframe", &object.BuiltinFunc{Name: "_getframe", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// CPython: sys._getframe([depth]) → frame N levels up.
		// depth=0 is the caller's frame. Since this builtin runs during
		// the caller's execution, i.curFrame *is* the caller (the
		// builtin frame is not pushed). depth must be in range.
		depth := 0
		if len(a) >= 1 {
			n, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "an integer is required")
			}
			depth = int(n)
		}
		if depth < 0 {
			return nil, object.Errorf(i.valueErr, "frame index must be non-negative")
		}
		f := i.curFrame
		for k := 0; k < depth && f != nil; k++ {
			f = f.Back
		}
		if f == nil {
			return nil, object.Errorf(i.valueErr, "call stack is not deep enough")
		}
		return f, nil
	}})

	m.Dict.SetStr("exc_info", &object.BuiltinFunc{Name: "exc_info", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		for f := i.curFrame; f != nil; f = f.Back {
			if f.ExcInfo != nil {
				e := f.ExcInfo
				return &object.Tuple{V: []object.Object{e.Class, e, object.None}}, nil
			}
		}
		return &object.Tuple{V: []object.Object{object.None, object.None, object.None}}, nil
	}})

	m.Dict.SetStr("getrecursionlimit", &object.BuiltinFunc{Name: "getrecursionlimit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(i.MaxDepth)), nil
	}})
	m.Dict.SetStr("setrecursionlimit", &object.BuiltinFunc{Name: "setrecursionlimit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "setrecursionlimit() takes exactly one argument")
		}
		n, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "integer expected")
		}
		i.MaxDepth = int(n)
		return object.None, nil
	}})

	// ── sys.flags ─────────────────────────────────────────────────────────────
	// Sequence of 18 flag values matching CPython's sys.flags layout.
	// All are 0/False in goipy (no dev-mode support).
	flagFields := []string{
		"debug", "inspect", "interactive", "optimize", "dont_write_bytecode",
		"no_user_site", "no_site", "ignore_environment", "verbose", "bytes_warning",
		"quiet", "hash_randomization", "isolated", "dev_mode", "utf8_mode",
		"warn_default_encoding", "safe_path", "int_max_str_digits",
	}
	flagVals := []object.Object{
		object.NewInt(0), object.NewInt(0), object.NewInt(0), object.NewInt(0), object.NewInt(0),
		object.NewInt(0), object.NewInt(0), object.NewInt(0), object.NewInt(0), object.NewInt(0),
		object.NewInt(0), object.NewInt(1), object.NewInt(0), object.BoolOf(false), object.NewInt(0),
		object.NewInt(0), object.BoolOf(false), object.NewInt(4300),
	}
	flagsCls := &object.Class{Name: "flags", Dict: object.NewDict()}
	flagsCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			vals, _ := inst.Dict.GetStr("__values__")
			list := vals.(*object.List)
			idx := 0
			if n, ok := a[1].(*object.Int); ok {
				idx = int(n.V.Int64())
			}
			if idx < 0 {
				idx = len(list.V) + idx
			}
			if idx < 0 || idx >= len(list.V) {
				return nil, object.Errorf(i.indexErr, "index out of range")
			}
			return list.V[idx], nil
		},
	})
	flagsCls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(18), nil
		},
	})
	flagsCls.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			vals, _ := inst.Dict.GetStr("__values__")
			list := vals.(*object.List)
			return &object.List{V: append([]object.Object{}, list.V...)}, nil
		},
	})
	flagsCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			fields, _ := inst.Dict.GetStr("__fields__")
			vals, _ := inst.Dict.GetStr("__values__")
			flist := fields.(*object.List)
			vlist := vals.(*object.List)
			var parts []string
			for j, fv := range flist.V {
				fname := fv.(*object.Str).V
				parts = append(parts, fname+"="+object.Repr(vlist.V[j]))
			}
			return &object.Str{V: "sys.flags(" + joinStrings(parts, ", ") + ")"}, nil
		},
	})
	flagsInst := &object.Instance{Class: flagsCls, Dict: object.NewDict()}
	fieldObjs := make([]object.Object, len(flagFields))
	for j, f := range flagFields {
		fieldObjs[j] = &object.Str{V: f}
	}
	flagsInst.Dict.SetStr("__fields__", &object.List{V: fieldObjs})
	flagsInst.Dict.SetStr("__values__", &object.List{V: flagVals})
	for j, fname := range flagFields {
		flagsInst.Dict.SetStr(fname, flagVals[j])
	}
	m.Dict.SetStr("flags", flagsInst)

	// ── sys.warnoptions / sys._xoptions ───────────────────────────────────────
	m.Dict.SetStr("warnoptions", &object.List{V: nil})
	m.Dict.SetStr("_xoptions", object.NewDict())

	// ── sys.maxunicode ────────────────────────────────────────────────────────
	m.Dict.SetStr("maxunicode", object.NewInt(1114111))

	// ── sys.getdefaultencoding ────────────────────────────────────────────────
	m.Dict.SetStr("getdefaultencoding", &object.BuiltinFunc{Name: "getdefaultencoding",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "utf-8"}, nil
		},
	})

	// ── sys.getfilesystemencoding / getfilesystemencodeerrors ─────────────────
	m.Dict.SetStr("getfilesystemencoding", &object.BuiltinFunc{Name: "getfilesystemencoding",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "utf-8"}, nil
		},
	})
	m.Dict.SetStr("getfilesystemencodeerrors", &object.BuiltinFunc{Name: "getfilesystemencodeerrors",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "surrogateescape"}, nil
		},
	})

	// ── sys.intern ────────────────────────────────────────────────────────────
	m.Dict.SetStr("intern", &object.BuiltinFunc{Name: "intern",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "intern() requires 1 argument")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "intern() argument must be str")
			}
			return s, nil
		},
	})

	// ── sys.addaudithook / sys.audit ──────────────────────────────────────────
	m.Dict.SetStr("addaudithook", &object.BuiltinFunc{Name: "addaudithook",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "addaudithook() requires 1 argument")
			}
			hook := a[0]
			interp := ii.(*Interp)
			// Notify existing hooks before adding the new one.
			if err := interp.fireAudit("sys.addaudithook", []object.Object{hook}); err != nil {
				return nil, err
			}
			interp.auditHooks = append(interp.auditHooks, hook)
			return object.None, nil
		},
	})
	m.Dict.SetStr("audit", &object.BuiltinFunc{Name: "audit",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "audit() requires at least 1 argument")
			}
			event := object.Str_(a[0])
			interp := ii.(*Interp)
			return object.None, interp.fireAudit(event, a[1:])
		},
	})

	// ── sys.getsizeof ─────────────────────────────────────────────────────────
	m.Dict.SetStr("getsizeof", &object.BuiltinFunc{Name: "getsizeof",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "getsizeof() requires at least 1 argument")
			}
			// Stub: return a reasonable positive size based on type
			var size int64 = 28
			switch a[0].(type) {
			case *object.Str:
				size = 50
			case *object.List:
				size = 56
			case *object.Dict:
				size = 64
			case *object.Tuple:
				size = 40
			case *object.Float:
				size = 24
			case *object.Bool:
				size = 28
			}
			return object.NewInt(size), nil
		},
	})

	// ── sys.is_finalizing ─────────────────────────────────────────────────────
	m.Dict.SetStr("is_finalizing", &object.BuiltinFunc{Name: "is_finalizing",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(false), nil
		},
	})

	// ── sys.stdin (stub) ──────────────────────────────────────────────────────
	m.Dict.SetStr("stdin", object.None)

	// ── sys path-init attributes ───────────────────────────────────────────────

	m.Dict.SetStr("path_hooks", &object.List{V: nil})
	m.Dict.SetStr("path_importer_cache", object.NewDict())
	m.Dict.SetStr("meta_path", &object.List{V: nil})

	m.Dict.SetStr("prefix", &object.Str{V: ""})
	m.Dict.SetStr("exec_prefix", &object.Str{V: ""})
	m.Dict.SetStr("base_prefix", &object.Str{V: ""})
	m.Dict.SetStr("base_exec_prefix", &object.Str{V: ""})
	m.Dict.SetStr("platlibdir", &object.Str{V: "lib"})

	m.Dict.SetStr("abiflags", &object.Str{V: ""})
	m.Dict.SetStr("float_repr_style", &object.Str{V: "short"})
	m.Dict.SetStr("hexversion", object.NewInt(0x030e00f0))

	// sys.stdlib_module_names — frozenset of CPython 3.14 stdlib module names
	stdlibNames := []string{
		"_thread", "abc", "aifc", "argparse", "array", "ast", "asynchat",
		"asyncio", "asyncore", "atexit", "audioop", "base64", "bdb", "binascii",
		"binhex", "bisect", "builtins", "bz2", "calendar", "cgi", "cgitb",
		"chunk", "cmath", "cmd", "code", "codecs", "codeop", "colorsys",
		"compileall", "concurrent", "configparser", "contextlib", "contextvars",
		"copy", "copyreg", "cProfile", "csv", "ctypes", "curses", "dataclasses",
		"datetime", "dbm", "decimal", "difflib", "dis", "doctest", "email",
		"encodings", "enum", "errno", "faulthandler", "fcntl", "filecmp",
		"fileinput", "fnmatch", "fractions", "ftplib", "functools", "gc",
		"getopt", "getpass", "gettext", "glob", "grp", "gzip", "hashlib",
		"heapq", "hmac", "html", "http", "idlelib", "imaplib", "importlib",
		"inspect", "io", "ipaddress", "itertools", "json", "keyword", "lib2to3",
		"linecache", "locale", "logging", "lzma", "mailbox", "marshal", "math",
		"mimetypes", "mmap", "modulefinder", "multiprocessing", "netrc",
		"numbers", "operator", "optparse", "os", "ossaudiodev", "pathlib",
		"pdb", "pickle", "pickletools", "pipes", "pkgutil", "platform",
		"plistlib", "poplib", "posix", "posixpath", "pprint", "profile",
		"pstats", "pty", "pwd", "py_compile", "pyclbr", "pydoc", "queue",
		"quopri", "random", "re", "readline", "reprlib", "resource", "rlcompleter",
		"runpy", "sched", "secrets", "select", "selectors", "shelve", "shlex",
		"shutil", "signal", "site", "smtpd", "smtplib", "sndhdr", "socket",
		"socketserver", "spwd", "sqlite3", "sre_compile", "sre_constants",
		"sre_parse", "ssl", "stat", "statistics", "string", "stringprep",
		"struct", "subprocess", "sunau", "symtable", "sys", "sysconfig",
		"syslog", "tabnanny", "tarfile", "telnetlib", "tempfile", "termios",
		"test", "textwrap", "threading", "time", "timeit", "tkinter", "token",
		"tokenize", "tomllib", "trace", "traceback", "tracemalloc", "tty",
		"turtle", "turtledemo", "types", "typing", "unicodedata", "unittest",
		"urllib", "uu", "uuid", "venv", "warnings", "wave", "weakref",
		"webbrowser", "wsgiref", "xdrlib", "xml", "xmlrpc", "zipapp",
		"zipfile", "zipimport", "zlib", "zoneinfo",
	}
	stdlibFS := object.NewFrozenset()
	for _, n := range stdlibNames {
		_ = stdlibFS.Add(&object.Str{V: n})
	}
	m.Dict.SetStr("stdlib_module_names", stdlibFS)

	// sys.int_info
	intInfoCls := &object.Class{Name: "int_info", Dict: object.NewDict()}
	intInfoInst := &object.Instance{Class: intInfoCls, Dict: object.NewDict()}
	intInfoInst.Dict.SetStr("bits_per_digit", object.NewInt(30))
	intInfoInst.Dict.SetStr("sizeof_digit", object.NewInt(4))
	intInfoInst.Dict.SetStr("default_max_str_digits", object.NewInt(4300))
	intInfoInst.Dict.SetStr("str_digits_check_threshold", object.NewInt(640))
	m.Dict.SetStr("int_info", intInfoInst)

	// sys.float_info
	floatInfoCls := &object.Class{Name: "float_info", Dict: object.NewDict()}
	floatInfoInst := &object.Instance{Class: floatInfoCls, Dict: object.NewDict()}
	floatInfoInst.Dict.SetStr("max", &object.Float{V: 1.7976931348623157e+308})
	floatInfoInst.Dict.SetStr("max_exp", object.NewInt(1024))
	floatInfoInst.Dict.SetStr("max_10_exp", object.NewInt(308))
	floatInfoInst.Dict.SetStr("min", &object.Float{V: 2.2250738585072014e-308})
	floatInfoInst.Dict.SetStr("min_exp", object.NewInt(-1021))
	floatInfoInst.Dict.SetStr("min_10_exp", object.NewInt(-307))
	floatInfoInst.Dict.SetStr("dig", object.NewInt(15))
	floatInfoInst.Dict.SetStr("mant_dig", object.NewInt(53))
	floatInfoInst.Dict.SetStr("epsilon", &object.Float{V: 2.220446049250313e-16})
	floatInfoInst.Dict.SetStr("radix", object.NewInt(2))
	floatInfoInst.Dict.SetStr("rounds", object.NewInt(1))
	m.Dict.SetStr("float_info", floatInfoInst)

	// sys.hash_info
	hashInfoCls := &object.Class{Name: "hash_info", Dict: object.NewDict()}
	hashInfoInst := &object.Instance{Class: hashInfoCls, Dict: object.NewDict()}
	hashInfoInst.Dict.SetStr("width", object.NewInt(64))
	hashInfoInst.Dict.SetStr("modulus", object.NewInt(2305843009213693951))
	hashInfoInst.Dict.SetStr("inf", object.NewInt(314159))
	hashInfoInst.Dict.SetStr("nan", object.NewInt(0))
	hashInfoInst.Dict.SetStr("imag", object.NewInt(1000003))
	hashInfoInst.Dict.SetStr("algorithm", &object.Str{V: "siphash13"})
	hashInfoInst.Dict.SetStr("hash_bits", object.NewInt(64))
	hashInfoInst.Dict.SetStr("seed_bits", object.NewInt(128))
	m.Dict.SetStr("hash_info", hashInfoInst)

	// sys.thread_info
	threadInfoCls := &object.Class{Name: "thread_info", Dict: object.NewDict()}
	threadInfoInst := &object.Instance{Class: threadInfoCls, Dict: object.NewDict()}
	threadInfoInst.Dict.SetStr("name", &object.Str{V: "pthread"})
	threadInfoInst.Dict.SetStr("lock", &object.Str{V: "mutex+cond"})
	threadInfoInst.Dict.SetStr("version", object.None)
	m.Dict.SetStr("thread_info", threadInfoInst)

	return m
}

// joinStrings joins a slice of strings with a separator.
func joinStrings(parts []string, sep string) string {
	result := ""
	for j, p := range parts {
		if j > 0 {
			result += sep
		}
		result += p
	}
	return result
}

// textStreamAttr dispatches attribute access on a *object.TextStream.
// Only the write-side API is exposed — these streams are not readable.
func textStreamAttr(i *Interp, ts *object.TextStream, name string) (object.Object, bool) {
	switch name {
	case "name":
		return &object.Str{V: "<" + ts.Name + ">"}, true
	case "mode":
		return &object.Str{V: "w"}, true
	case "closed":
		return object.False, true
	case "encoding":
		return &object.Str{V: "utf-8"}, true
	case "write":
		return &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "write() takes exactly one argument")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "write() argument must be str")
			}
			w, ok := ts.W.(io.Writer)
			if !ok {
				return object.NewInt(0), nil
			}
			n, _ := w.Write([]byte(s.V))
			return object.NewInt(int64(n)), nil
		}}, true
	case "flush":
		return &object.BuiltinFunc{Name: "flush", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}, true
	}
	return nil, false
}

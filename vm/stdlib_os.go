package vm

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"

	"github.com/tamnd/goipy/object"
)

// buildOs constructs a minimal os module covering file operations,
// environment access, and path constants.
func (i *Interp) buildOs() *object.Module {
	m := &object.Module{Name: "os", Dict: object.NewDict()}

	// Constants.
	m.Dict.SetStr("sep", &object.Str{V: string(os.PathSeparator)})
	m.Dict.SetStr("linesep", &object.Str{V: linesp()})
	m.Dict.SetStr("devnull", &object.Str{V: os.DevNull})
	m.Dict.SetStr("curdir", &object.Str{V: "."})
	m.Dict.SetStr("pardir", &object.Str{V: ".."})
	m.Dict.SetStr("extsep", &object.Str{V: "."})
	m.Dict.SetStr("altsep", object.None)
	m.Dict.SetStr("name", &object.Str{V: osName()})
	m.Dict.SetStr("pathsep", &object.Str{V: string(os.PathListSeparator)})

	// environ — special mapping that syncs writes to process environment.
	m.Dict.SetStr("environ", i.buildOsEnviron())

	// getcwd() — return current working directory.
	m.Dict.SetStr("getcwd", &object.BuiltinFunc{Name: "getcwd", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		dir, err := os.Getwd()
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Str{V: dir}, nil
	}})

	// getenv(key[, default]) — get environment variable.
	m.Dict.SetStr("getenv", &object.BuiltinFunc{Name: "getenv", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "getenv() requires at least 1 argument")
		}
		key, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "getenv() key must be str")
		}
		val, found := os.LookupEnv(key.V)
		if !found {
			if len(a) >= 2 {
				return a[1], nil
			}
			return object.None, nil
		}
		return &object.Str{V: val}, nil
	}})

	// putenv(key, value) — set environment variable.
	m.Dict.SetStr("putenv", &object.BuiltinFunc{Name: "putenv", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "putenv() requires 2 arguments")
		}
		key, ok1 := a[0].(*object.Str)
		val, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "putenv() arguments must be str")
		}
		os.Setenv(key.V, val.V)
		// keep environ dict in sync
		if envObj, ok := m.Dict.GetStr("environ"); ok {
			if envDict, ok := envObj.(*object.Dict); ok {
				envDict.SetStr(key.V, val)
			}
		}
		return object.None, nil
	}})

	// remove(path) / unlink(path) — delete a file.
	rm := &object.BuiltinFunc{Name: "remove", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "remove() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "remove() path must be str")
		}
		if err := os.Remove(p.V); err != nil {
			if os.IsNotExist(err) {
				return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: '%s'", p.V)
			}
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}}
	m.Dict.SetStr("remove", rm)
	m.Dict.SetStr("unlink", rm)

	// rename(src, dst) — rename a file.
	m.Dict.SetStr("rename", &object.BuiltinFunc{Name: "rename", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "rename() requires 2 arguments")
		}
		src, ok1 := a[0].(*object.Str)
		dst, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "rename() arguments must be str")
		}
		if err := os.Rename(src.V, dst.V); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// mkdir(path[, mode]) — create a directory.
	m.Dict.SetStr("mkdir", &object.BuiltinFunc{Name: "mkdir", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "mkdir() requires at least 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "mkdir() path must be str")
		}
		mode := os.FileMode(0o777)
		if len(a) >= 2 {
			if n, ok := a[1].(*object.Int); ok {
				mode = os.FileMode(n.V.Int64())
			}
		}
		if err := os.Mkdir(p.V, mode); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// makedirs(path[, mode, exist_ok]) — create directory tree.
	m.Dict.SetStr("makedirs", &object.BuiltinFunc{Name: "makedirs", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "makedirs() requires at least 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "makedirs() path must be str")
		}
		mode := os.FileMode(0o777)
		if len(a) >= 2 {
			if n, ok := a[1].(*object.Int); ok {
				mode = os.FileMode(n.V.Int64())
			}
		}
		if err := os.MkdirAll(p.V, mode); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// rmdir(path) — remove an empty directory.
	m.Dict.SetStr("rmdir", &object.BuiltinFunc{Name: "rmdir", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "rmdir() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "rmdir() path must be str")
		}
		if err := os.Remove(p.V); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// listdir([path]) — list directory contents.
	m.Dict.SetStr("listdir", &object.BuiltinFunc{Name: "listdir", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		dir := "."
		if len(a) >= 1 {
			p, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "listdir() path must be str")
			}
			dir = p.V
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		out := make([]object.Object, len(entries))
		for idx, e := range entries {
			out[idx] = &object.Str{V: e.Name()}
		}
		return &object.List{V: out}, nil
	}})

	// stat(path) — minimal stat result.
	m.Dict.SetStr("stat", &object.BuiltinFunc{Name: "stat", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "stat() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "stat() path must be str")
		}
		info, err := os.Stat(p.V)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: '%s'", p.V)
			}
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return osStatResult(i, info), nil
	}})

	// lstat(path) — stat without following symlinks.
	m.Dict.SetStr("lstat", &object.BuiltinFunc{Name: "lstat", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "lstat() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "lstat() path must be str")
		}
		info, err := os.Lstat(p.V)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: '%s'", p.V)
			}
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return osStatResult(i, info), nil
	}})

	// symlink(src, dst) — create a symbolic link.
	m.Dict.SetStr("symlink", &object.BuiltinFunc{Name: "symlink", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "symlink() requires 2 arguments")
		}
		src, ok1 := a[0].(*object.Str)
		dst, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "symlink() arguments must be str")
		}
		if err := os.Symlink(src.V, dst.V); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// readlink(path) — return target of symbolic link.
	m.Dict.SetStr("readlink", &object.BuiltinFunc{Name: "readlink", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "readlink() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "readlink() path must be str")
		}
		target, err := os.Readlink(p.V)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Str{V: target}, nil
	}})

	// getuid() — return current process UID.
	m.Dict.SetStr("getuid", &object.BuiltinFunc{Name: "getuid", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(os.Getuid())), nil
	}})

	// getpid() — return current process ID.
	m.Dict.SetStr("getpid", &object.BuiltinFunc{Name: "getpid", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(os.Getpid())), nil
	}})

	// path sub-module as attribute.
	osPath := i.buildOsPath()
	m.Dict.SetStr("path", osPath)

	return m
}

// osStatResult builds a stat_result-like instance from an os.FileInfo.
func osStatResult(i *Interp, info os.FileInfo) object.Object {
	cls := &object.Class{Name: "stat_result", Bases: nil, Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	mtime := float64(info.ModTime().UnixNano()) / 1e9
	inst.Dict.SetStr("st_size", object.NewInt(info.Size()))
	inst.Dict.SetStr("st_mode", object.NewInt(int64(info.Mode())))
	inst.Dict.SetStr("st_mtime", &object.Float{V: mtime})
	inst.Dict.SetStr("st_atime", &object.Float{V: mtime})
	inst.Dict.SetStr("st_ctime", &object.Float{V: mtime})
	inst.Dict.SetStr("st_nlink", object.NewInt(1))
	inst.Dict.SetStr("st_uid", object.NewInt(0))
	inst.Dict.SetStr("st_gid", object.NewInt(0))
	inst.Dict.SetStr("st_ino", object.NewInt(0))
	inst.Dict.SetStr("st_dev", object.NewInt(0))

	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		inst.Dict.SetStr("st_ino", object.NewInt(int64(sys.Ino)))
		inst.Dict.SetStr("st_dev", object.NewInt(int64(sys.Dev)))
		inst.Dict.SetStr("st_nlink", object.NewInt(int64(sys.Nlink)))
		inst.Dict.SetStr("st_uid", object.NewInt(int64(sys.Uid)))
		inst.Dict.SetStr("st_gid", object.NewInt(int64(sys.Gid)))
		atimeSec, atimeNsec := statAtime(sys)
		ctimeSec, ctimeNsec := statCtime(sys)
		inst.Dict.SetStr("st_atime", &object.Float{V: float64(atimeSec) + float64(atimeNsec)/1e9})
		inst.Dict.SetStr("st_ctime", &object.Float{V: float64(ctimeSec) + float64(ctimeNsec)/1e9})
	}
	return inst
}

// buildOsPath constructs os.path with all standard posixpath functions.
func (i *Interp) buildOsPath() *object.Module {
	m := &object.Module{Name: "os.path", Dict: object.NewDict()}

	// Constants.
	m.Dict.SetStr("sep", &object.Str{V: string(os.PathSeparator)})
	m.Dict.SetStr("curdir", &object.Str{V: "."})
	m.Dict.SetStr("pardir", &object.Str{V: ".."})
	m.Dict.SetStr("extsep", &object.Str{V: "."})
	m.Dict.SetStr("altsep", object.None)
	m.Dict.SetStr("pathsep", &object.Str{V: string(os.PathListSeparator)})
	m.Dict.SetStr("defpath", &object.Str{V: "/bin:/usr/bin"})
	m.Dict.SetStr("devnull", &object.Str{V: os.DevNull})
	m.Dict.SetStr("supports_unicode_filenames", object.BoolOf(runtime.GOOS == "darwin"))

	// isabs(path)
	m.Dict.SetStr("isabs", &object.BuiltinFunc{Name: "isabs", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "isabs", a)
		if err != nil {
			return nil, err
		}
		return object.BoolOf(filepath.IsAbs(p)), nil
	}})

	// abspath(path)
	m.Dict.SetStr("abspath", &object.BuiltinFunc{Name: "abspath", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "abspath", a)
		if err != nil {
			return nil, err
		}
		abs, err2 := filepath.Abs(p)
		if err2 != nil {
			return nil, object.Errorf(i.osErr, "%v", err2)
		}
		return &object.Str{V: abs}, nil
	}})

	// basename(path)
	m.Dict.SetStr("basename", &object.BuiltinFunc{Name: "basename", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "basename", a)
		if err != nil {
			return nil, err
		}
		// Python: p[p.rfind('/')+1:] — returns everything after last slash.
		idx := strings.LastIndex(p, "/")
		return &object.Str{V: p[idx+1:]}, nil
	}})

	// dirname(path)
	m.Dict.SetStr("dirname", &object.BuiltinFunc{Name: "dirname", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "dirname", a)
		if err != nil {
			return nil, err
		}
		head, _ := posixSplit(p)
		return &object.Str{V: head}, nil
	}})

	// split(path) — returns (head, tail)
	m.Dict.SetStr("split", &object.BuiltinFunc{Name: "split", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "split", a)
		if err != nil {
			return nil, err
		}
		head, tail := posixSplit(p)
		return &object.Tuple{V: []object.Object{&object.Str{V: head}, &object.Str{V: tail}}}, nil
	}})

	// splitext(path) — returns (root, ext)
	m.Dict.SetStr("splitext", &object.BuiltinFunc{Name: "splitext", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "splitext", a)
		if err != nil {
			return nil, err
		}
		ext := filepath.Ext(p)
		// Python: dotfiles like ".bar" have no extension
		base := filepath.Base(p)
		if strings.HasPrefix(base, ".") && strings.Count(base, ".") == 1 {
			ext = ""
		}
		root := strings.TrimSuffix(p, ext)
		return &object.Tuple{V: []object.Object{&object.Str{V: root}, &object.Str{V: ext}}}, nil
	}})

	// splitdrive(path) — POSIX: drive is always ""
	m.Dict.SetStr("splitdrive", &object.BuiltinFunc{Name: "splitdrive", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "splitdrive", a)
		if err != nil {
			return nil, err
		}
		return &object.Tuple{V: []object.Object{&object.Str{V: ""}, &object.Str{V: p}}}, nil
	}})

	// splitroot(path) — Python 3.12+, POSIX: (drive, root, tail)
	m.Dict.SetStr("splitroot", &object.BuiltinFunc{Name: "splitroot", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "splitroot", a)
		if err != nil {
			return nil, err
		}
		drive := ""
		root := ""
		tail := p
		if strings.HasPrefix(p, "/") {
			root = "/"
			tail = strings.TrimPrefix(p, "/")
		}
		return &object.Tuple{V: []object.Object{&object.Str{V: drive}, &object.Str{V: root}, &object.Str{V: tail}}}, nil
	}})

	// join(path, *paths) — absolute component resets previous parts
	m.Dict.SetStr("join", &object.BuiltinFunc{Name: "join", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "join() requires at least 1 argument")
		}
		result := ""
		for _, arg := range a {
			s, ok := arg.(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "join() arguments must be str")
			}
			p := s.V
			if filepath.IsAbs(p) {
				result = p
			} else if result == "" || strings.HasSuffix(result, "/") {
				result = result + p
			} else {
				result = result + "/" + p
			}
		}
		return &object.Str{V: result}, nil
	}})

	// normpath(path)
	m.Dict.SetStr("normpath", &object.BuiltinFunc{Name: "normpath", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "normpath", a)
		if err != nil {
			return nil, err
		}
		if p == "" {
			return &object.Str{V: "."}, nil
		}
		return &object.Str{V: filepath.Clean(p)}, nil
	}})

	// normcase(path) — POSIX no-op
	m.Dict.SetStr("normcase", &object.BuiltinFunc{Name: "normcase", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "normcase", a)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: p}, nil
	}})

	// realpath(path[, strict=False])
	m.Dict.SetStr("realpath", &object.BuiltinFunc{Name: "realpath", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "realpath", a)
		if err != nil {
			return nil, err
		}
		rp, err2 := filepath.EvalSymlinks(p)
		if err2 != nil {
			// If symlink resolution fails (e.g. strict=False default), fall back to Abs.
			abs, _ := filepath.Abs(p)
			return &object.Str{V: abs}, nil
		}
		abs, _ := filepath.Abs(rp)
		return &object.Str{V: abs}, nil
	}})

	// relpath(path, start='.')
	m.Dict.SetStr("relpath", &object.BuiltinFunc{Name: "relpath", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "relpath() requires at least 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "relpath() path must be str")
		}
		start := "."
		if len(a) >= 2 {
			s, ok := a[1].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "relpath() start must be str")
			}
			start = s.V
		}
		rel, err2 := filepath.Rel(start, p.V)
		if err2 != nil {
			return nil, object.Errorf(i.osErr, "%v", err2)
		}
		return &object.Str{V: rel}, nil
	}})

	// commonprefix(list) — character-by-character prefix (not path-aware)
	m.Dict.SetStr("commonprefix", &object.BuiltinFunc{Name: "commonprefix", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "commonprefix() requires 1 argument")
		}
		lst, ok := a[0].(*object.List)
		if !ok {
			return nil, object.Errorf(i.typeErr, "commonprefix() argument must be a list")
		}
		if len(lst.V) == 0 {
			return &object.Str{V: ""}, nil
		}
		strs := make([]string, len(lst.V))
		for idx, v := range lst.V {
			s, ok := v.(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "commonprefix() list elements must be str")
			}
			strs[idx] = s.V
		}
		prefix := strs[0]
		for _, s := range strs[1:] {
			for !strings.HasPrefix(s, prefix) {
				prefix = prefix[:len(prefix)-1]
				if prefix == "" {
					return &object.Str{V: ""}, nil
				}
			}
		}
		return &object.Str{V: prefix}, nil
	}})

	// commonpath(paths) — path-aware common ancestor
	m.Dict.SetStr("commonpath", &object.BuiltinFunc{Name: "commonpath", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "commonpath() requires 1 argument")
		}
		var paths []string
		switch v := a[0].(type) {
		case *object.List:
			paths = make([]string, len(v.V))
			for idx, item := range v.V {
				s, ok := item.(*object.Str)
				if !ok {
					return nil, object.Errorf(i.typeErr, "commonpath() elements must be str")
				}
				paths[idx] = s.V
			}
		case *object.Tuple:
			paths = make([]string, len(v.V))
			for idx, item := range v.V {
				s, ok := item.(*object.Str)
				if !ok {
					return nil, object.Errorf(i.typeErr, "commonpath() elements must be str")
				}
				paths[idx] = s.V
			}
		default:
			return nil, object.Errorf(i.typeErr, "commonpath() argument must be a sequence")
		}
		if len(paths) == 0 {
			return nil, object.Errorf(i.valueErr, "commonpath() arg is an empty sequence")
		}
		// Split each path into components and find common prefix of component slices.
		splitParts := func(p string) []string {
			p = filepath.Clean(p)
			return strings.Split(p, "/")
		}
		parts := splitParts(paths[0])
		for _, p := range paths[1:] {
			pp := splitParts(p)
			min := len(parts)
			if len(pp) < min {
				min = len(pp)
			}
			parts = parts[:min]
			for j := 0; j < min; j++ {
				if parts[j] != pp[j] {
					parts = parts[:j]
					break
				}
			}
		}
		if len(parts) == 0 {
			return nil, object.Errorf(i.valueErr, "commonpath() paths have no common ancestor")
		}
		result := strings.Join(parts, "/")
		if result == "" {
			result = "/"
		}
		return &object.Str{V: result}, nil
	}})

	// exists(path)
	m.Dict.SetStr("exists", &object.BuiltinFunc{Name: "exists", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "exists", a)
		if err != nil {
			return nil, err
		}
		_, e := os.Stat(p)
		return object.BoolOf(e == nil), nil
	}})

	// lexists(path) — True even if broken symlink
	m.Dict.SetStr("lexists", &object.BuiltinFunc{Name: "lexists", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "lexists", a)
		if err != nil {
			return nil, err
		}
		_, e := os.Lstat(p)
		return object.BoolOf(e == nil), nil
	}})

	// isfile(path)
	m.Dict.SetStr("isfile", &object.BuiltinFunc{Name: "isfile", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "isfile", a)
		if err != nil {
			return nil, err
		}
		info, e := os.Stat(p)
		return object.BoolOf(e == nil && info.Mode().IsRegular()), nil
	}})

	// isdir(path)
	m.Dict.SetStr("isdir", &object.BuiltinFunc{Name: "isdir", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "isdir", a)
		if err != nil {
			return nil, err
		}
		info, e := os.Stat(p)
		return object.BoolOf(e == nil && info.IsDir()), nil
	}})

	// islink(path)
	m.Dict.SetStr("islink", &object.BuiltinFunc{Name: "islink", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "islink", a)
		if err != nil {
			return nil, err
		}
		info, e := os.Lstat(p)
		return object.BoolOf(e == nil && info.Mode()&os.ModeSymlink != 0), nil
	}})

	// ismount(path)
	m.Dict.SetStr("ismount", &object.BuiltinFunc{Name: "ismount", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "ismount", a)
		if err != nil {
			return nil, err
		}
		return object.BoolOf(osIsMount(p)), nil
	}})

	// isjunction(path) — always False on POSIX
	m.Dict.SetStr("isjunction", &object.BuiltinFunc{Name: "isjunction", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		_, err := osPathStr(i, "isjunction", a)
		if err != nil {
			return nil, err
		}
		return object.False, nil
	}})

	// getsize(path)
	m.Dict.SetStr("getsize", &object.BuiltinFunc{Name: "getsize", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "getsize", a)
		if err != nil {
			return nil, err
		}
		info, e := os.Stat(p)
		if e != nil {
			return nil, object.Errorf(i.osErr, "%v", e)
		}
		return object.NewInt(info.Size()), nil
	}})

	// getmtime(path)
	m.Dict.SetStr("getmtime", &object.BuiltinFunc{Name: "getmtime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "getmtime", a)
		if err != nil {
			return nil, err
		}
		info, e := os.Stat(p)
		if e != nil {
			return nil, object.Errorf(i.osErr, "%v", e)
		}
		return &object.Float{V: float64(info.ModTime().UnixNano()) / 1e9}, nil
	}})

	// getatime(path)
	m.Dict.SetStr("getatime", &object.BuiltinFunc{Name: "getatime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "getatime", a)
		if err != nil {
			return nil, err
		}
		info, e := os.Stat(p)
		if e != nil {
			return nil, object.Errorf(i.osErr, "%v", e)
		}
		atime := osAtime(info)
		return &object.Float{V: atime}, nil
	}})

	// getctime(path)
	m.Dict.SetStr("getctime", &object.BuiltinFunc{Name: "getctime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "getctime", a)
		if err != nil {
			return nil, err
		}
		info, e := os.Stat(p)
		if e != nil {
			return nil, object.Errorf(i.osErr, "%v", e)
		}
		ctime := osCtime(info)
		return &object.Float{V: ctime}, nil
	}})

	// expanduser(path)
	m.Dict.SetStr("expanduser", &object.BuiltinFunc{Name: "expanduser", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "expanduser", a)
		if err != nil {
			return nil, err
		}
		home, _ := os.UserHomeDir()
		result := p
		if strings.HasPrefix(result, "~/") {
			result = home + result[1:]
		} else if result == "~" {
			result = home
		}
		return &object.Str{V: result}, nil
	}})

	// expandvars(path) — expand $VAR and ${VAR}
	m.Dict.SetStr("expandvars", &object.BuiltinFunc{Name: "expandvars", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p, err := osPathStr(i, "expandvars", a)
		if err != nil {
			return nil, err
		}
		result := osExpandVars(p)
		return &object.Str{V: result}, nil
	}})

	// samefile(path1, path2)
	m.Dict.SetStr("samefile", &object.BuiltinFunc{Name: "samefile", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "samefile() requires 2 arguments")
		}
		p1, ok1 := a[0].(*object.Str)
		p2, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "samefile() arguments must be str")
		}
		i1, e1 := os.Stat(p1.V)
		i2, e2 := os.Stat(p2.V)
		if e1 != nil || e2 != nil {
			return nil, object.Errorf(i.osErr, "samefile() stat failed")
		}
		return object.BoolOf(os.SameFile(i1, i2)), nil
	}})

	// samestat(stat1, stat2) — compare two stat_result objects by ino+dev
	m.Dict.SetStr("samestat", &object.BuiltinFunc{Name: "samestat", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "samestat() requires 2 arguments")
		}
		ino1, dev1, ok1 := osStatIno(a[0])
		ino2, dev2, ok2 := osStatIno(a[1])
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "samestat() arguments must be stat_result objects")
		}
		return object.BoolOf(ino1 == ino2 && dev1 == dev2), nil
	}})

	return m
}

// buildOsEnviron returns an _Environ instance whose __setitem__ syncs to the
// real process environment so that os.path.expandvars sees changes.
func (i *Interp) buildOsEnviron() *object.Instance {
	cls := &object.Class{Name: "_Environ", Bases: nil, Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	// Populate from current process env.
	for _, kv := range os.Environ() {
		idx := strings.IndexByte(kv, '=')
		if idx < 0 {
			continue
		}
		inst.Dict.SetStr(kv[:idx], &object.Str{V: kv[idx+1:]})
	}

	cls.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		key, ok1 := a[1].(*object.Str)
		val, ok2 := a[2].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "environ keys and values must be str")
		}
		self.Dict.SetStr(key.V, val)
		os.Setenv(key.V, val.V)
		return object.None, nil
	}})

	cls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		key, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "environ key must be str")
		}
		if v, ok := self.Dict.GetStr(key.V); ok {
			return v, nil
		}
		return nil, object.Errorf(i.keyErr, "'%s'", key.V)
	}})

	cls.Dict.SetStr("__delitem__", &object.BuiltinFunc{Name: "__delitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		key, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "environ key must be str")
		}
		self.Dict.Delete(&object.Str{V: key.V}) //nolint:errcheck
		os.Unsetenv(key.V)
		return object.None, nil
	}})

	cls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "get() requires at least 1 argument")
		}
		key, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "environ key must be str")
		}
		if v, ok := self.Dict.GetStr(key.V); ok {
			return v, nil
		}
		if len(a) >= 3 {
			return a[2], nil
		}
		return object.None, nil
	}})

	return inst
}

// posixSplit splits a path into (head, tail) matching Python's os.path.split.
// Trailing slashes are stripped from head unless head is root.
func posixSplit(p string) (string, string) {
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return "", p
	}
	head, tail := p[:i], p[i+1:]
	// Strip trailing slashes from head, preserving root.
	head = strings.TrimRight(head, "/")
	if head == "" && strings.HasPrefix(p, "/") {
		head = "/"
	}
	return head, tail
}

// osPathStr extracts a single string argument for os.path functions.
func osPathStr(i *Interp, name string, a []object.Object) (string, error) {
	if len(a) < 1 {
		return "", object.Errorf(i.typeErr, "%s() requires 1 argument", name)
	}
	s, ok := a[0].(*object.Str)
	if !ok {
		return "", object.Errorf(i.typeErr, "%s() path must be str", name)
	}
	return s.V, nil
}

// osExpandVars expands $VAR and ${VAR} patterns.
var reExpandBrace = regexp.MustCompile(`\$\{([^}]+)\}`)
var reExpandPlain = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)

func osExpandVars(s string) string {
	// First expand ${VAR} forms.
	s = reExpandBrace.ReplaceAllStringFunc(s, func(m string) string {
		key := m[2 : len(m)-1]
		if v, ok := os.LookupEnv(key); ok {
			return v
		}
		return m
	})
	// Then expand $VAR forms.
	s = reExpandPlain.ReplaceAllStringFunc(s, func(m string) string {
		key := m[1:]
		if v, ok := os.LookupEnv(key); ok {
			return v
		}
		return m
	})
	return s
}

// osIsMount checks if path is a mount point.
func osIsMount(p string) bool {
	if p == "/" {
		return true
	}
	info1, err1 := os.Lstat(p)
	if err1 != nil {
		return false
	}
	parent := filepath.Dir(p)
	info2, err2 := os.Lstat(parent)
	if err2 != nil {
		return false
	}
	s1, ok1 := info1.Sys().(*syscall.Stat_t)
	s2, ok2 := info2.Sys().(*syscall.Stat_t)
	if !ok1 || !ok2 {
		return false
	}
	return s1.Dev != s2.Dev
}

// osStatIno extracts (ino, dev) from a stat_result instance.
func osStatIno(o object.Object) (uint64, uint64, bool) {
	inst, ok := o.(*object.Instance)
	if !ok {
		return 0, 0, false
	}
	inoObj, ok1 := inst.Dict.GetStr("st_ino")
	devObj, ok2 := inst.Dict.GetStr("st_dev")
	if !ok1 || !ok2 {
		return 0, 0, false
	}
	inoInt, ok3 := inoObj.(*object.Int)
	devInt, ok4 := devObj.(*object.Int)
	if !ok3 || !ok4 {
		return 0, 0, false
	}
	return inoInt.V.Uint64(), devInt.V.Uint64(), true
}

func linesp() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}

func osName() string {
	if runtime.GOOS == "windows" {
		return "nt"
	}
	return "posix"
}

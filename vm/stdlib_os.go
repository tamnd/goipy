package vm

import (
	"crypto/rand"
	"errors"
	"io/fs"
	"os"
	"os/user"
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

	// Access-mode constants.
	m.Dict.SetStr("F_OK", object.NewInt(0))
	m.Dict.SetStr("R_OK", object.NewInt(4))
	m.Dict.SetStr("W_OK", object.NewInt(2))
	m.Dict.SetStr("X_OK", object.NewInt(1))

	// Open-flag constants (match syscall values; platform-specific but consistent).
	m.Dict.SetStr("O_RDONLY", object.NewInt(int64(syscall.O_RDONLY)))
	m.Dict.SetStr("O_WRONLY", object.NewInt(int64(syscall.O_WRONLY)))
	m.Dict.SetStr("O_RDWR", object.NewInt(int64(syscall.O_RDWR)))
	m.Dict.SetStr("O_CREAT", object.NewInt(int64(syscall.O_CREAT)))
	m.Dict.SetStr("O_TRUNC", object.NewInt(int64(syscall.O_TRUNC)))
	m.Dict.SetStr("O_APPEND", object.NewInt(int64(syscall.O_APPEND)))
	m.Dict.SetStr("O_EXCL", object.NewInt(int64(syscall.O_EXCL)))

	// Seek constants.
	m.Dict.SetStr("SEEK_SET", object.NewInt(0))
	m.Dict.SetStr("SEEK_CUR", object.NewInt(1))
	m.Dict.SetStr("SEEK_END", object.NewInt(2))

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

	// makedirs(path[, mode=0o777, exist_ok=False]) — create directory tree.
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
			if n, ok2 := a[1].(*object.Int); ok2 {
				mode = os.FileMode(n.V.Int64())
			}
		}
		existOK := false
		if kw != nil {
			if v, ok2 := kw.GetStr("exist_ok"); ok2 {
				existOK = object.Truthy(v)
			}
		}
		// If exist_ok=False and the final path already exists, raise immediately.
		if !existOK {
			if _, serr := os.Lstat(p.V); serr == nil {
				return nil, object.Errorf(i.fileExistsErr, "[Errno 17] File exists: '%s'", p.V)
			}
		}
		if err := os.MkdirAll(p.V, mode); err != nil {
			if errors.Is(err, fs.ErrExist) && existOK {
				return object.None, nil
			}
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

	// close(fd) — close an OS-level file descriptor.
	m.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "close() requires 1 argument")
		}
		n, ok := a[0].(*object.Int)
		if !ok {
			return nil, object.Errorf(i.typeErr, "close() fd must be int")
		}
		_ = tempfileClose(int(n.V.Int64()))
		return object.None, nil
	}})

	// chdir(path) — change current working directory.
	m.Dict.SetStr("chdir", &object.BuiltinFunc{Name: "chdir", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "chdir() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "chdir() path must be str")
		}
		if err := os.Chdir(p.V); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	m.Dict.SetStr("chmod", &object.BuiltinFunc{Name: "chmod", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "chmod() requires path and mode")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "chmod() path must be str")
		}
		mode, ok2 := a[1].(*object.Int)
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "chmod() mode must be int")
		}
		if err := os.Chmod(p.V, fs.FileMode(mode.Int64())); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// replace(src, dst) — atomic rename; dst overwritten if exists.
	m.Dict.SetStr("replace", &object.BuiltinFunc{Name: "replace", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "replace() requires 2 arguments")
		}
		src, ok1 := a[0].(*object.Str)
		dst, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "replace() arguments must be str")
		}
		if err := os.Rename(src.V, dst.V); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// link(src, dst) — create a hard link.
	m.Dict.SetStr("link", &object.BuiltinFunc{Name: "link", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "link() requires 2 arguments")
		}
		src, ok1 := a[0].(*object.Str)
		dst, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "link() arguments must be str")
		}
		if err := os.Link(src.V, dst.V); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// truncate(path, length) — truncate file.
	m.Dict.SetStr("truncate", &object.BuiltinFunc{Name: "truncate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "truncate() requires path and length")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "truncate() path must be str")
		}
		n, ok2 := toInt64(a[1])
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "truncate() length must be int")
		}
		if err := os.Truncate(p.V, n); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// umask(mask) — set file creation mask; returns old mask.
	m.Dict.SetStr("umask", &object.BuiltinFunc{Name: "umask", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "umask() requires 1 argument")
		}
		n, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "umask() argument must be int")
		}
		old := syscall.Umask(int(n))
		return object.NewInt(int64(old)), nil
	}})

	// access(path, mode) — check file accessibility.
	m.Dict.SetStr("access", &object.BuiltinFunc{Name: "access", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "access() requires path and mode")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "access() path must be str")
		}
		mode, ok2 := toInt64(a[1])
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "access() mode must be int")
		}
		err := syscall.Access(p.V, uint32(mode))
		return object.BoolOf(err == nil), nil
	}})

	// walk(top, topdown=True, onerror=None, followlinks=False) — directory tree generator.
	m.Dict.SetStr("walk", &object.BuiltinFunc{Name: "walk", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "walk() requires at least 1 argument")
		}
		top, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "walk() top must be str")
		}
		topdown := true
		if len(a) >= 2 {
			topdown = object.Truthy(a[1])
		} else if kw != nil {
			if v, ok2 := kw.GetStr("topdown"); ok2 {
				topdown = object.Truthy(v)
			}
		}
		ch := make(chan object.Object, 64)
		go func() {
			defer close(ch)
			osWalkDir(top.V, topdown, ch)
		}()
		return &object.Iter{Next: func() (object.Object, bool, error) {
			v, ok := <-ch
			if !ok {
				return nil, false, nil
			}
			return v, true, nil
		}}, nil
	}})

	// scandir(path='.') — returns iterator of DirEntry objects.
	m.Dict.SetStr("scandir", &object.BuiltinFunc{Name: "scandir", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		dir := "."
		if len(a) >= 1 {
			p, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "scandir() path must be str")
			}
			dir = p.V
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(entries) {
				return nil, false, nil
			}
			e := entries[idx]
			idx++
			return osDirEntry(i, dir, e), true, nil
		}}, nil
	}})

	// Process info.
	m.Dict.SetStr("getgid", &object.BuiltinFunc{Name: "getgid", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(os.Getgid())), nil
	}})
	m.Dict.SetStr("getegid", &object.BuiltinFunc{Name: "getegid", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(os.Getegid())), nil
	}})
	m.Dict.SetStr("geteuid", &object.BuiltinFunc{Name: "geteuid", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(os.Geteuid())), nil
	}})
	m.Dict.SetStr("getppid", &object.BuiltinFunc{Name: "getppid", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(os.Getppid())), nil
	}})
	m.Dict.SetStr("getlogin", &object.BuiltinFunc{Name: "getlogin", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		u, err := user.Current()
		if err != nil {
			return nil, object.Errorf(i.osErr, "getlogin: %v", err)
		}
		return &object.Str{V: u.Username}, nil
	}})
	m.Dict.SetStr("cpu_count", &object.BuiltinFunc{Name: "cpu_count", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		n := runtime.NumCPU()
		if n <= 0 {
			return object.None, nil
		}
		return object.NewInt(int64(n)), nil
	}})

	// Misc helpers.
	m.Dict.SetStr("strerror", &object.BuiltinFunc{Name: "strerror", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "strerror() requires 1 argument")
		}
		n, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "strerror() argument must be int")
		}
		s := syscall.Errno(n).Error()
		if len(s) > 0 {
			s = strings.ToUpper(s[:1]) + s[1:]
		}
		return &object.Str{V: s}, nil
	}})

	m.Dict.SetStr("urandom", &object.BuiltinFunc{Name: "urandom", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "urandom() requires 1 argument")
		}
		n, ok := toInt64(a[0])
		if !ok || n < 0 {
			return nil, object.Errorf(i.valueErr, "urandom() size must be non-negative")
		}
		buf := make([]byte, n)
		if _, err := rand.Read(buf); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Bytes{V: buf}, nil
	}})

	m.Dict.SetStr("fsencode", &object.BuiltinFunc{Name: "fsencode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fsencode() requires 1 argument")
		}
		switch v := a[0].(type) {
		case *object.Str:
			return &object.Bytes{V: []byte(v.V)}, nil
		case *object.Bytes:
			return v, nil
		}
		return nil, object.Errorf(i.typeErr, "fsencode() argument must be str or bytes")
	}})

	m.Dict.SetStr("fsdecode", &object.BuiltinFunc{Name: "fsdecode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fsdecode() requires 1 argument")
		}
		switch v := a[0].(type) {
		case *object.Bytes:
			return &object.Str{V: string(v.V)}, nil
		case *object.Str:
			return v, nil
		}
		return nil, object.Errorf(i.typeErr, "fsdecode() argument must be bytes or str")
	}})

	m.Dict.SetStr("get_exec_path", &object.BuiltinFunc{Name: "get_exec_path", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		pathVal := os.Getenv("PATH")
		if len(a) >= 1 {
			if d, ok := a[0].(*object.Dict); ok {
				if v, ok2 := d.GetStr("PATH"); ok2 {
					if s, ok3 := v.(*object.Str); ok3 {
						pathVal = s.V
					}
				}
			}
		}
		parts := strings.Split(pathVal, string(os.PathListSeparator))
		out := make([]object.Object, 0, len(parts))
		for _, p := range parts {
			if p != "" {
				out = append(out, &object.Str{V: p})
			}
		}
		return &object.List{V: out}, nil
	}})

	// Low-level file descriptor operations.
	m.Dict.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "os.open() requires path and flags")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.open() path must be str")
		}
		flags, ok2 := toInt64(a[1])
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "os.open() flags must be int")
		}
		mode := int64(0o777)
		if len(a) >= 3 {
			if n, ok3 := toInt64(a[2]); ok3 {
				mode = n
			}
		}
		fd, err := syscall.Open(p.V, int(flags), uint32(mode))
		if err != nil {
			return nil, object.Errorf(i.osErr, "[Errno %d] %v: '%s'", err.(syscall.Errno), err, p.V)
		}
		return object.NewInt(int64(fd)), nil
	}})

	m.Dict.SetStr("read", &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "os.read() requires fd and n")
		}
		fd, ok1 := toInt64(a[0])
		n, ok2 := toInt64(a[1])
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "os.read() fd and n must be int")
		}
		buf := make([]byte, n)
		nr, err := syscall.Read(int(fd), buf)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Bytes{V: buf[:nr]}, nil
	}})

	m.Dict.SetStr("write", &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "os.write() requires fd and data")
		}
		fd, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.write() fd must be int")
		}
		data, err2 := asBytes(a[1])
		if err2 != nil {
			return nil, err2
		}
		nw, err := syscall.Write(int(fd), data)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.NewInt(int64(nw)), nil
	}})

	m.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.close() requires 1 argument")
		}
		fd, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.close() fd must be int")
		}
		if err := tempfileClose(int(fd)); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	m.Dict.SetStr("lseek", &object.BuiltinFunc{Name: "lseek", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "os.lseek() requires fd, pos, how")
		}
		fd, ok1 := toInt64(a[0])
		pos, ok2 := toInt64(a[1])
		how, ok3 := toInt64(a[2])
		if !ok1 || !ok2 || !ok3 {
			return nil, object.Errorf(i.typeErr, "os.lseek() arguments must be int")
		}
		newpos, err := syscall.Seek(int(fd), pos, int(how))
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.NewInt(newpos), nil
	}})

	m.Dict.SetStr("fstat", &object.BuiltinFunc{Name: "fstat", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.fstat() requires 1 argument")
		}
		fd, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.fstat() fd must be int")
		}
		var st syscall.Stat_t
		if err := syscall.Fstat(int(fd), &st); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return osSyscallStatResult(i, &st), nil
	}})

	m.Dict.SetStr("dup", &object.BuiltinFunc{Name: "dup", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.dup() requires 1 argument")
		}
		fd, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.dup() fd must be int")
		}
		newfd, err := syscall.Dup(int(fd))
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.NewInt(int64(newfd)), nil
	}})

	m.Dict.SetStr("dup2", &object.BuiltinFunc{Name: "dup2", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "os.dup2() requires oldfd and newfd")
		}
		oldfd, ok1 := toInt64(a[0])
		newfd, ok2 := toInt64(a[1])
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "os.dup2() arguments must be int")
		}
		if err := syscall.Dup2(int(oldfd), int(newfd)); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.NewInt(int64(newfd)), nil
	}})

	// path sub-module as attribute.
	osPath := i.buildOsPath()
	m.Dict.SetStr("path", osPath)

	return m
}

// statResultClass is a shared class for all stat_result objects.
// The __getitem__ method supports tuple-like index access [0..9].
var statResultClass = func() *object.Class {
	cls := &object.Class{Name: "stat_result", Bases: nil, Dict: object.NewDict()}
	cls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, nil
		}
		n, ok2 := a[1].(*object.Int)
		if !ok2 {
			return nil, nil
		}
		// st_mode(0) st_ino(1) st_dev(2) st_nlink(3) st_uid(4) st_gid(5) st_size(6) st_atime(7) st_mtime(8) st_ctime(9)
		fields := []string{"st_mode", "st_ino", "st_dev", "st_nlink", "st_uid", "st_gid", "st_size", "st_atime", "st_mtime", "st_ctime"}
		idx := int(n.Int64())
		if idx < 0 {
			idx += len(fields)
		}
		if idx < 0 || idx >= len(fields) {
			return nil, &object.Exception{Msg: "index out of range"}
		}
		v, _ := self.Dict.GetStr(fields[idx])
		return v, nil
	}})
	return cls
}()

// osStatResult builds a stat_result-like instance from an os.FileInfo.
func osStatResult(i *Interp, info os.FileInfo) object.Object {
	inst := &object.Instance{Class: statResultClass, Dict: object.NewDict()}
	mtime := float64(info.ModTime().UnixNano()) / 1e9
	inst.Dict.SetStr("st_size", object.NewInt(info.Size()))
	inst.Dict.SetStr("st_mtime", &object.Float{V: mtime})
	inst.Dict.SetStr("st_atime", &object.Float{V: mtime})
	inst.Dict.SetStr("st_ctime", &object.Float{V: mtime})
	inst.Dict.SetStr("st_nlink", object.NewInt(1))
	inst.Dict.SetStr("st_uid", object.NewInt(0))
	inst.Dict.SetStr("st_gid", object.NewInt(0))
	inst.Dict.SetStr("st_ino", object.NewInt(0))
	inst.Dict.SetStr("st_dev", object.NewInt(0))

	if sys, ok := info.Sys().(*syscall.Stat_t); ok {
		inst.Dict.SetStr("st_mode", object.NewInt(int64(sys.Mode)))
		inst.Dict.SetStr("st_ino", object.NewInt(int64(sys.Ino)))
		inst.Dict.SetStr("st_dev", object.NewInt(int64(sys.Dev)))
		inst.Dict.SetStr("st_nlink", object.NewInt(int64(sys.Nlink)))
		inst.Dict.SetStr("st_uid", object.NewInt(int64(sys.Uid)))
		inst.Dict.SetStr("st_gid", object.NewInt(int64(sys.Gid)))
		atimeSec, atimeNsec := statAtime(sys)
		ctimeSec, ctimeNsec := statCtime(sys)
		inst.Dict.SetStr("st_atime", &object.Float{V: float64(atimeSec) + float64(atimeNsec)/1e9})
		inst.Dict.SetStr("st_ctime", &object.Float{V: float64(ctimeSec) + float64(ctimeNsec)/1e9})
	} else {
		inst.Dict.SetStr("st_mode", object.NewInt(goModeToPosix(info.Mode())))
	}
	return inst
}

// osSyscallStatResult builds a stat_result from a raw syscall.Stat_t (used by fstat).
func osSyscallStatResult(i *Interp, sys *syscall.Stat_t) object.Object {
	inst := &object.Instance{Class: statResultClass, Dict: object.NewDict()}
	inst.Dict.SetStr("st_mode", object.NewInt(int64(sys.Mode)))
	inst.Dict.SetStr("st_ino", object.NewInt(int64(sys.Ino)))
	inst.Dict.SetStr("st_dev", object.NewInt(int64(sys.Dev)))
	inst.Dict.SetStr("st_nlink", object.NewInt(int64(sys.Nlink)))
	inst.Dict.SetStr("st_uid", object.NewInt(int64(sys.Uid)))
	inst.Dict.SetStr("st_gid", object.NewInt(int64(sys.Gid)))
	inst.Dict.SetStr("st_size", object.NewInt(sys.Size))
	atimeSec, atimeNsec := statAtime(sys)
	ctimeSec, ctimeNsec := statCtime(sys)
	mtimeSec, mtimeNsec := statMtime(sys)
	inst.Dict.SetStr("st_atime", &object.Float{V: float64(atimeSec) + float64(atimeNsec)/1e9})
	inst.Dict.SetStr("st_mtime", &object.Float{V: float64(mtimeSec) + float64(mtimeNsec)/1e9})
	inst.Dict.SetStr("st_ctime", &object.Float{V: float64(ctimeSec) + float64(ctimeNsec)/1e9})
	return inst
}

// osDirEntry builds a DirEntry-like instance for os.scandir().
func osDirEntry(i *Interp, dir string, e os.DirEntry) object.Object {
	cls := &object.Class{Name: "DirEntry", Bases: nil, Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	name := e.Name()
	fullPath := filepath.Join(dir, name)
	inst.Dict.SetStr("name", &object.Str{V: name})
	inst.Dict.SetStr("path", &object.Str{V: fullPath})

	cls.Dict.SetStr("is_dir", &object.BuiltinFunc{Name: "is_dir", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		followLinks := true
		if kw != nil {
			if v, ok := kw.GetStr("follow_symlinks"); ok {
				followLinks = object.Truthy(v)
			}
		}
		var info os.FileInfo
		var err error
		if followLinks {
			info, err = os.Stat(fullPath)
		} else {
			info, err = os.Lstat(fullPath)
		}
		return object.BoolOf(err == nil && info.IsDir()), nil
	}})

	cls.Dict.SetStr("is_file", &object.BuiltinFunc{Name: "is_file", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		followLinks := true
		if kw != nil {
			if v, ok := kw.GetStr("follow_symlinks"); ok {
				followLinks = object.Truthy(v)
			}
		}
		var info os.FileInfo
		var err error
		if followLinks {
			info, err = os.Stat(fullPath)
		} else {
			info, err = os.Lstat(fullPath)
		}
		return object.BoolOf(err == nil && info.Mode().IsRegular()), nil
	}})

	cls.Dict.SetStr("is_symlink", &object.BuiltinFunc{Name: "is_symlink", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		info, err := os.Lstat(fullPath)
		return object.BoolOf(err == nil && info.Mode()&os.ModeSymlink != 0), nil
	}})

	cls.Dict.SetStr("stat", &object.BuiltinFunc{Name: "stat", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		followLinks := true
		if kw != nil {
			if v, ok := kw.GetStr("follow_symlinks"); ok {
				followLinks = object.Truthy(v)
			}
		}
		var info os.FileInfo
		var err error
		if followLinks {
			info, err = os.Stat(fullPath)
		} else {
			info, err = os.Lstat(fullPath)
		}
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return osStatResult(i, info), nil
	}})

	cls.Dict.SetStr("inode", &object.BuiltinFunc{Name: "inode", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		info, err := os.Lstat(fullPath)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		if sys, ok := info.Sys().(*syscall.Stat_t); ok {
			return object.NewInt(int64(sys.Ino)), nil
		}
		return object.NewInt(0), nil
	}})

	return inst
}

// osWalkDir recursively walks a directory tree, sending (root, dirs, files) tuples
// to ch. topdown=true sends parent before children.
func osWalkDir(top string, topdown bool, ch chan<- object.Object) {
	entries, err := os.ReadDir(top)
	if err != nil {
		return
	}
	var dirs, files []object.Object
	for _, e := range entries {
		name := &object.Str{V: e.Name()}
		if e.IsDir() {
			dirs = append(dirs, name)
		} else {
			files = append(files, name)
		}
	}
	if dirs == nil {
		dirs = []object.Object{}
	}
	if files == nil {
		files = []object.Object{}
	}
	dirsList := &object.List{V: dirs}
	filesList := &object.List{V: files}
	triple := &object.Tuple{V: []object.Object{
		&object.Str{V: top},
		dirsList,
		filesList,
	}}
	if topdown {
		ch <- triple
	}
	// Recurse into subdirectories (using dirsList.V which caller may have modified).
	for _, d := range dirsList.V {
		if ds, ok := d.(*object.Str); ok {
			osWalkDir(filepath.Join(top, ds.V), topdown, ch)
		}
	}
	if !topdown {
		ch <- triple
	}
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

	cls.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		key, ok := a[1].(*object.Str)
		if !ok {
			return object.False, nil
		}
		_, found := self.Dict.GetStr(key.V)
		return object.BoolOf(found), nil
	}})

	cls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		return object.NewInt(int64(self.Dict.Len())), nil
	}})

	cls.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		ks, _ := self.Dict.Items()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(ks) {
				return nil, false, nil
			}
			k := ks[idx]
			idx++
			return k, true, nil
		}}, nil
	}})

	cls.Dict.SetStr("keys", &object.BuiltinFunc{Name: "keys", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		ks, _ := self.Dict.Items()
		return &object.List{V: ks}, nil
	}})

	cls.Dict.SetStr("values", &object.BuiltinFunc{Name: "values", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		_, vs := self.Dict.Items()
		return &object.List{V: vs}, nil
	}})

	cls.Dict.SetStr("items", &object.BuiltinFunc{Name: "items", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		ks, vs := self.Dict.Items()
		tuples := make([]object.Object, len(ks))
		for j := range ks {
			tuples[j] = &object.Tuple{V: []object.Object{ks[j], vs[j]}}
		}
		return &object.List{V: tuples}, nil
	}})

	cls.Dict.SetStr("copy", &object.BuiltinFunc{Name: "copy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		ks, vs := self.Dict.Items()
		d := object.NewDict()
		for j, k := range ks {
			if ks2, ok := k.(*object.Str); ok {
				d.SetStr(ks2.V, vs[j])
			}
		}
		return d, nil
	}})

	cls.Dict.SetStr("update", &object.BuiltinFunc{Name: "update", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		if len(a) >= 2 {
			switch src := a[1].(type) {
			case *object.Dict:
				srcKs, srcVs := src.Items()
				for j, k := range srcKs {
					ks, ok := k.(*object.Str)
					if !ok {
						continue
					}
					v := srcVs[j]
					self.Dict.SetStr(ks.V, v)
					if s, ok2 := v.(*object.Str); ok2 {
						os.Setenv(ks.V, s.V)
					}
				}
			}
		}
		if kw != nil {
			kwKs, kwVs := kw.Items()
			for j, k := range kwKs {
				ks, ok := k.(*object.Str)
				if !ok {
					continue
				}
				v := kwVs[j]
				self.Dict.SetStr(ks.V, v)
				if s, ok2 := v.(*object.Str); ok2 {
					os.Setenv(ks.V, s.V)
				}
			}
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("setdefault", &object.BuiltinFunc{Name: "setdefault", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "setdefault() requires at least 1 argument")
		}
		key, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "environ key must be str")
		}
		if v, found := self.Dict.GetStr(key.V); found {
			return v, nil
		}
		var def object.Object = object.None
		if len(a) >= 3 {
			def = a[2]
		}
		self.Dict.SetStr(key.V, def)
		if s, ok2 := def.(*object.Str); ok2 {
			os.Setenv(key.V, s.V)
		}
		return def, nil
	}})

	cls.Dict.SetStr("pop", &object.BuiltinFunc{Name: "pop", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "pop() requires at least 1 argument")
		}
		key, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "environ key must be str")
		}
		if v, found := self.Dict.GetStr(key.V); found {
			self.Dict.Delete(&object.Str{V: key.V}) //nolint:errcheck
			os.Unsetenv(key.V)
			return v, nil
		}
		if len(a) >= 3 {
			return a[2], nil
		}
		return nil, object.Errorf(i.keyErr, "'%s'", key.V)
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

// goModeToPosix converts Go's fs.FileMode to POSIX st_mode bits.
// Used as a fallback when syscall.Stat_t is unavailable.
func goModeToPosix(m os.FileMode) int64 {
	perm := int64(m.Perm()) // low 9 bits
	switch {
	case m&os.ModeDir != 0:
		perm |= 0o040000
	case m&os.ModeSymlink != 0:
		perm |= 0o120000
	case m&os.ModeNamedPipe != 0:
		perm |= 0o010000
	case m&os.ModeSocket != 0:
		perm |= 0o140000
	case m&os.ModeDevice != 0:
		if m&os.ModeCharDevice != 0 {
			perm |= 0o020000
		} else {
			perm |= 0o060000
		}
	default:
		perm |= 0o100000 // regular file
	}
	if m&os.ModeSetuid != 0 {
		perm |= 0o4000
	}
	if m&os.ModeSetgid != 0 {
		perm |= 0o2000
	}
	if m&os.ModeSticky != 0 {
		perm |= 0o1000
	}
	return perm
}

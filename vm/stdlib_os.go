package vm

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

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

	// environ — expose as a plain dict snapshot.
	env := object.NewDict()
	for _, kv := range os.Environ() {
		idx := strings.IndexByte(kv, '=')
		if idx < 0 {
			continue
		}
		env.SetStr(kv[:idx], &object.Str{V: kv[idx+1:]})
	}
	m.Dict.SetStr("environ", env)

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
		d := object.NewDict()
		d.SetStr("st_size", object.NewInt(info.Size()))
		d.SetStr("st_mode", object.NewInt(int64(info.Mode())))
		d.SetStr("st_mtime", &object.Float{V: float64(info.ModTime().UnixNano()) / 1e9})
		return d, nil
	}})

	// path sub-module as attribute.
	osPath := i.buildOsPath()
	m.Dict.SetStr("path", osPath)

	return m
}

// buildOsPath constructs os.path with the most common functions.
func (i *Interp) buildOsPath() *object.Module {
	m := &object.Module{Name: "os.path", Dict: object.NewDict()}

	m.Dict.SetStr("sep", &object.Str{V: string(os.PathSeparator)})

	m.Dict.SetStr("join", &object.BuiltinFunc{Name: "join", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.path.join() requires at least 1 argument")
		}
		parts := make([]string, len(a))
		for idx, arg := range a {
			s, ok := arg.(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "os.path.join() arguments must be str")
			}
			parts[idx] = s.V
		}
		return &object.Str{V: filepath.Join(parts...)}, nil
	}})

	m.Dict.SetStr("exists", &object.BuiltinFunc{Name: "exists", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.path.exists() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.path.exists() path must be str")
		}
		_, err := os.Stat(p.V)
		return object.BoolOf(err == nil), nil
	}})

	m.Dict.SetStr("isfile", &object.BuiltinFunc{Name: "isfile", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.path.isfile() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.path.isfile() path must be str")
		}
		info, err := os.Stat(p.V)
		return object.BoolOf(err == nil && info.Mode().IsRegular()), nil
	}})

	m.Dict.SetStr("isdir", &object.BuiltinFunc{Name: "isdir", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.path.isdir() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.path.isdir() path must be str")
		}
		info, err := os.Stat(p.V)
		return object.BoolOf(err == nil && info.IsDir()), nil
	}})

	m.Dict.SetStr("basename", &object.BuiltinFunc{Name: "basename", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.path.basename() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.path.basename() path must be str")
		}
		return &object.Str{V: filepath.Base(p.V)}, nil
	}})

	m.Dict.SetStr("dirname", &object.BuiltinFunc{Name: "dirname", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.path.dirname() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.path.dirname() path must be str")
		}
		return &object.Str{V: filepath.Dir(p.V)}, nil
	}})

	m.Dict.SetStr("abspath", &object.BuiltinFunc{Name: "abspath", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.path.abspath() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.path.abspath() path must be str")
		}
		abs, err := filepath.Abs(p.V)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Str{V: abs}, nil
	}})

	m.Dict.SetStr("split", &object.BuiltinFunc{Name: "split", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.path.split() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.path.split() path must be str")
		}
		dir, file := filepath.Split(p.V)
		return &object.Tuple{V: []object.Object{&object.Str{V: dir}, &object.Str{V: file}}}, nil
	}})

	m.Dict.SetStr("splitext", &object.BuiltinFunc{Name: "splitext", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.path.splitext() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.path.splitext() path must be str")
		}
		ext := filepath.Ext(p.V)
		base := strings.TrimSuffix(p.V, ext)
		return &object.Tuple{V: []object.Object{&object.Str{V: base}, &object.Str{V: ext}}}, nil
	}})

	m.Dict.SetStr("expanduser", &object.BuiltinFunc{Name: "expanduser", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.path.expanduser() requires 1 argument")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.path.expanduser() path must be str")
		}
		home, _ := os.UserHomeDir()
		result := p.V
		if strings.HasPrefix(result, "~/") {
			result = home + result[1:]
		} else if result == "~" {
			result = home
		}
		return &object.Str{V: result}, nil
	}})

	return m
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

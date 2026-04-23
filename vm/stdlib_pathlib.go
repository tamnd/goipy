package vm

import (
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/tamnd/goipy/object"
)

// buildPathlib constructs the pathlib module — PurePosixPath, PurePath,
// Path, and PosixPath (on POSIX) classes matching CPython 3.14 semantics.
func (i *Interp) buildPathlib() *object.Module {
	m := &object.Module{Name: "pathlib", Dict: object.NewDict()}

	pureCls := i.makePurePosixPathClass()
	pathCls := i.makePathClass(pureCls)

	m.Dict.SetStr("PurePosixPath", pureCls)
	m.Dict.SetStr("PurePath", pureCls)
	m.Dict.SetStr("Path", pathCls)
	m.Dict.SetStr("PosixPath", pathCls)

	return m
}

// pathStr extracts the stored "_path" string from a path instance.
func pathStr(inst *object.Instance) string {
	v, ok := inst.Dict.GetStr("_path")
	if !ok {
		return "."
	}
	s, _ := v.(*object.Str)
	if s == nil {
		return "."
	}
	return s.V
}

// newPathInst creates a new instance of cls with the given path string.
func newPathInst(cls *object.Class, p string) *object.Instance {
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("_path", &object.Str{V: p})
	return inst
}

// joinPaths joins a base path with one or more string/path arguments.
func joinPaths(base string, args []object.Object) (string, error) {
	parts := []string{base}
	for _, a := range args {
		switch v := a.(type) {
		case *object.Str:
			parts = append(parts, v.V)
		case *object.Instance:
			parts = append(parts, pathStr(v))
		default:
			return "", object.Errorf(nil, "argument must be str or Path, not '%s'", object.TypeName(a))
		}
	}
	return filepath.Join(parts...), nil
}

// pathSuffixes returns the list of suffixes (e.g. [".tar", ".gz"]).
func pathSuffixes(name string) []string {
	var out []string
	for {
		ext := filepath.Ext(name)
		if ext == "" {
			break
		}
		out = append([]string{ext}, out...)
		name = name[:len(name)-len(ext)]
	}
	return out
}

// pathDrive returns the drive letter/UNC on Windows, empty on POSIX.
func pathDrive(p string) string {
	vol := filepath.VolumeName(p)
	return vol
}

// pathRoot returns "/" if path is absolute else "".
func pathRoot(p string) string {
	if filepath.IsAbs(p) {
		return "/"
	}
	return ""
}

// pathAnchor = drive + root.
func pathAnchor(p string) string {
	return pathDrive(p) + pathRoot(p)
}

// makePurePosixPathClass builds the pure (no I/O) path class.
func (i *Interp) makePurePosixPathClass() *object.Class {
	cls := &object.Class{Name: "PurePosixPath", Dict: object.NewDict()}

	// __new__ / __init__: Path(*parts)
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "missing self")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "self must be instance")
		}
		parts := []string{}
		for _, arg := range a[1:] {
			switch v := arg.(type) {
			case *object.Str:
				parts = append(parts, v.V)
			case *object.Instance:
				parts = append(parts, pathStr(v))
			default:
				return nil, object.Errorf(i.typeErr, "argument must be str or Path")
			}
		}
		p := "."
		if len(parts) > 0 {
			p = filepath.Join(parts...)
		}
		inst.Dict.SetStr("_path", &object.Str{V: p})
		return object.None, nil
	}})

	cls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		// CPython normalizes "." for empty/current
		return &object.Str{V: p}, nil
	}})

	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		return &object.Str{V: inst.Class.Name + "('" + pathStr(inst) + "')"}, nil
	}})

	cls.Dict.SetStr("__fspath__", &object.BuiltinFunc{Name: "__fspath__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		return &object.Str{V: pathStr(inst)}, nil
	}})

	cls.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		other, ok := a[1].(*object.Instance)
		if !ok {
			return object.NotImplemented, nil
		}
		return object.BoolOf(pathStr(inst) == pathStr(other)), nil
	}})

	cls.Dict.SetStr("__lt__", &object.BuiltinFunc{Name: "__lt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		other, ok := a[1].(*object.Instance)
		if !ok {
			return object.NotImplemented, nil
		}
		return object.BoolOf(pathStr(inst) < pathStr(other)), nil
	}})

	cls.Dict.SetStr("__le__", &object.BuiltinFunc{Name: "__le__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		other, ok := a[1].(*object.Instance)
		if !ok {
			return object.NotImplemented, nil
		}
		return object.BoolOf(pathStr(inst) <= pathStr(other)), nil
	}})

	cls.Dict.SetStr("__gt__", &object.BuiltinFunc{Name: "__gt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		other, ok := a[1].(*object.Instance)
		if !ok {
			return object.NotImplemented, nil
		}
		return object.BoolOf(pathStr(inst) > pathStr(other)), nil
	}})

	cls.Dict.SetStr("__ge__", &object.BuiltinFunc{Name: "__ge__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		other, ok := a[1].(*object.Instance)
		if !ok {
			return object.NotImplemented, nil
		}
		return object.BoolOf(pathStr(inst) >= pathStr(other)), nil
	}})

	cls.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		h, err := object.Hash(&object.Str{V: pathStr(inst)})
		if err != nil {
			return nil, err
		}
		return object.NewInt(int64(h)), nil
	}})

	// path / other  →  path.joinpath(other)
	cls.Dict.SetStr("__truediv__", &object.BuiltinFunc{Name: "__truediv__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		joined, err := joinPaths(pathStr(inst), a[1:])
		if err != nil {
			return nil, err
		}
		return newPathInst(inst.Class, joined), nil
	}})

	// other / path
	cls.Dict.SetStr("__rtruediv__", &object.BuiltinFunc{Name: "__rtruediv__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		var prefix string
		switch v := a[1].(type) {
		case *object.Str:
			prefix = v.V
		case *object.Instance:
			prefix = pathStr(v)
		default:
			return object.NotImplemented, nil
		}
		joined := filepath.Join(prefix, pathStr(inst))
		return newPathInst(inst.Class, joined), nil
	}})

	// --- properties ---

	prop := func(fn func(*object.Instance) (object.Object, error)) *object.Property {
		return &object.Property{Fget: &object.BuiltinFunc{Name: "", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return nil, object.Errorf(i.typeErr, "expected instance")
			}
			return fn(inst)
		}}}
	}

	cls.Dict.SetStr("drive", prop(func(inst *object.Instance) (object.Object, error) {
		return &object.Str{V: pathDrive(pathStr(inst))}, nil
	}))

	cls.Dict.SetStr("root", prop(func(inst *object.Instance) (object.Object, error) {
		return &object.Str{V: pathRoot(pathStr(inst))}, nil
	}))

	cls.Dict.SetStr("anchor", prop(func(inst *object.Instance) (object.Object, error) {
		return &object.Str{V: pathAnchor(pathStr(inst))}, nil
	}))

	cls.Dict.SetStr("name", prop(func(inst *object.Instance) (object.Object, error) {
		return &object.Str{V: filepath.Base(pathStr(inst))}, nil
	}))

	cls.Dict.SetStr("suffix", prop(func(inst *object.Instance) (object.Object, error) {
		return &object.Str{V: filepath.Ext(filepath.Base(pathStr(inst)))}, nil
	}))

	cls.Dict.SetStr("suffixes", prop(func(inst *object.Instance) (object.Object, error) {
		suffs := pathSuffixes(filepath.Base(pathStr(inst)))
		out := make([]object.Object, len(suffs))
		for k, s := range suffs {
			out[k] = &object.Str{V: s}
		}
		return &object.List{V: out}, nil
	}))

	cls.Dict.SetStr("stem", prop(func(inst *object.Instance) (object.Object, error) {
		base := filepath.Base(pathStr(inst))
		ext := filepath.Ext(base)
		return &object.Str{V: base[:len(base)-len(ext)]}, nil
	}))

	cls.Dict.SetStr("parent", prop(func(inst *object.Instance) (object.Object, error) {
		p := pathStr(inst)
		dir := filepath.Dir(p)
		return newPathInst(inst.Class, dir), nil
	}))

	cls.Dict.SetStr("parents", prop(func(inst *object.Instance) (object.Object, error) {
		p := pathStr(inst)
		var ps []object.Object
		cur := filepath.Dir(p)
		for {
			ps = append(ps, newPathInst(inst.Class, cur))
			next := filepath.Dir(cur)
			if next == cur {
				break
			}
			cur = next
		}
		return &object.List{V: ps}, nil
	}))

	cls.Dict.SetStr("parts", prop(func(inst *object.Instance) (object.Object, error) {
		p := pathStr(inst)
		var parts []object.Object
		// Split into components
		dir, file := filepath.Split(p)
		// Collect all components
		all := []string{}
		if filepath.IsAbs(p) {
			all = append(all, "/")
			p = p[1:] // strip leading /
		}
		// Split remaining
		for _, part := range strings.Split(p, string(filepath.Separator)) {
			if part != "" {
				all = append(all, part)
			}
		}
		_ = dir
		_ = file
		for _, part := range all {
			parts = append(parts, &object.Str{V: part})
		}
		return &object.Tuple{V: parts}, nil
	}))

	// --- pure methods ---

	cls.Dict.SetStr("as_posix", &object.BuiltinFunc{Name: "as_posix", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := filepath.ToSlash(pathStr(inst))
		return &object.Str{V: p}, nil
	}})

	cls.Dict.SetStr("is_absolute", &object.BuiltinFunc{Name: "is_absolute", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		return object.BoolOf(filepath.IsAbs(pathStr(inst))), nil
	}})

	cls.Dict.SetStr("is_relative_to", &object.BuiltinFunc{Name: "is_relative_to", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "is_relative_to() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		var other string
		switch v := a[1].(type) {
		case *object.Str:
			other = v.V
		case *object.Instance:
			other = pathStr(v)
		default:
			return nil, object.Errorf(i.typeErr, "argument must be str or Path")
		}
		p := pathStr(inst)
		rel, err := filepath.Rel(other, p)
		if err != nil {
			return object.BoolOf(false), nil
		}
		return object.BoolOf(!strings.HasPrefix(rel, "..")), nil
	}})

	cls.Dict.SetStr("relative_to", &object.BuiltinFunc{Name: "relative_to", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "relative_to() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		var other string
		switch v := a[1].(type) {
		case *object.Str:
			other = v.V
		case *object.Instance:
			other = pathStr(v)
		default:
			return nil, object.Errorf(i.typeErr, "argument must be str or Path")
		}
		p := pathStr(inst)
		rel, err := filepath.Rel(other, p)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, object.Errorf(i.valueErr, "'%s' is not relative to '%s'", p, other)
		}
		return newPathInst(inst.Class, rel), nil
	}})

	cls.Dict.SetStr("with_name", &object.BuiltinFunc{Name: "with_name", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "with_name() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		name, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "name must be str")
		}
		p := pathStr(inst)
		dir := filepath.Dir(p)
		return newPathInst(inst.Class, filepath.Join(dir, name.V)), nil
	}})

	cls.Dict.SetStr("with_stem", &object.BuiltinFunc{Name: "with_stem", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "with_stem() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		stem, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "stem must be str")
		}
		p := pathStr(inst)
		dir := filepath.Dir(p)
		ext := filepath.Ext(filepath.Base(p))
		return newPathInst(inst.Class, filepath.Join(dir, stem.V+ext)), nil
	}})

	cls.Dict.SetStr("with_suffix", &object.BuiltinFunc{Name: "with_suffix", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "with_suffix() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		suf, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "suffix must be str")
		}
		p := pathStr(inst)
		dir := filepath.Dir(p)
		base := filepath.Base(p)
		ext := filepath.Ext(base)
		stem := base[:len(base)-len(ext)]
		return newPathInst(inst.Class, filepath.Join(dir, stem+suf.V)), nil
	}})

	cls.Dict.SetStr("with_segments", &object.BuiltinFunc{Name: "with_segments", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		joined, err := joinPaths("", a[1:])
		if err != nil {
			return nil, err
		}
		return newPathInst(inst.Class, joined), nil
	}})

	cls.Dict.SetStr("joinpath", &object.BuiltinFunc{Name: "joinpath", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		joined, err := joinPaths(pathStr(inst), a[1:])
		if err != nil {
			return nil, err
		}
		return newPathInst(inst.Class, joined), nil
	}})

	cls.Dict.SetStr("match", &object.BuiltinFunc{Name: "match", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "match() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		pat, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "pattern must be str")
		}
		p := pathStr(inst)
		matched, err := filepath.Match(pat.V, filepath.Base(p))
		if err != nil {
			return nil, object.Errorf(i.valueErr, "invalid pattern: %v", err)
		}
		if !matched && strings.Contains(pat.V, "/") {
			matched, err = filepath.Match(pat.V, p)
			if err != nil {
				return nil, object.Errorf(i.valueErr, "invalid pattern: %v", err)
			}
		}
		return object.BoolOf(matched), nil
	}})

	cls.Dict.SetStr("full_match", &object.BuiltinFunc{Name: "full_match", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "full_match() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		pat, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "pattern must be str")
		}
		matched, err := filepath.Match(pat.V, pathStr(inst))
		if err != nil {
			return nil, object.Errorf(i.valueErr, "invalid pattern: %v", err)
		}
		return object.BoolOf(matched), nil
	}})

	return cls
}

// makePathClass builds the concrete Path class (adds I/O methods).
func (i *Interp) makePathClass(pureCls *object.Class) *object.Class {
	cls := &object.Class{
		Name:  "PosixPath",
		Bases: []*object.Class{pureCls},
		Dict:  object.NewDict(),
	}

	// Inherit __init__ from pureCls by forwarding.
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		init, ok := pureCls.Dict.GetStr("__init__")
		if !ok {
			return object.None, nil
		}
		bf := init.(*object.BuiltinFunc)
		return bf.Call(nil, a, kw)
	}})

	// Class methods: cwd() and home()
	cwdFn := &object.BuiltinFunc{Name: "cwd", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		dir, err := os.Getwd()
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return newPathInst(cls, dir), nil
	}}
	cls.Dict.SetStr("cwd", cwdFn)

	homeFn := &object.BuiltinFunc{Name: "home", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		u, err := user.Current()
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return newPathInst(cls, u.HomeDir), nil
	}}
	cls.Dict.SetStr("home", homeFn)

	// stat() and lstat()
	mkStatResult := func(info fs.FileInfo) *object.Instance {
		stCls := &object.Class{Name: "stat_result", Dict: object.NewDict()}
		st := &object.Instance{Class: stCls, Dict: object.NewDict()}
		st.Dict.SetStr("st_mode", object.NewInt(int64(info.Mode())))
		st.Dict.SetStr("st_size", object.NewInt(info.Size()))
		st.Dict.SetStr("st_mtime", object.NewInt(info.ModTime().Unix()))
		st.Dict.SetStr("st_atime", object.NewInt(info.ModTime().Unix()))
		st.Dict.SetStr("st_ctime", object.NewInt(info.ModTime().Unix()))
		st.Dict.SetStr("st_nlink", object.NewInt(1))
		st.Dict.SetStr("st_uid", object.NewInt(0))
		st.Dict.SetStr("st_gid", object.NewInt(0))
		st.Dict.SetStr("st_ino", object.NewInt(0))
		st.Dict.SetStr("st_dev", object.NewInt(0))
		return st
	}

	cls.Dict.SetStr("stat", &object.BuiltinFunc{Name: "stat", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		followSymlinks := true
		if kw != nil {
			if v, ok := kw.GetStr("follow_symlinks"); ok {
				followSymlinks = object.Truthy(v)
			}
		}
		var info fs.FileInfo
		var err error
		if followSymlinks {
			info, err = os.Stat(p)
		} else {
			info, err = os.Lstat(p)
		}
		if os.IsNotExist(err) {
			return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: '%s'", p)
		}
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return mkStatResult(info), nil
	}})

	cls.Dict.SetStr("lstat", &object.BuiltinFunc{Name: "lstat", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		info, err := os.Lstat(p)
		if os.IsNotExist(err) {
			return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: '%s'", p)
		}
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return mkStatResult(info), nil
	}})

	// exists / is_file / is_dir / is_symlink / is_mount
	cls.Dict.SetStr("exists", &object.BuiltinFunc{Name: "exists", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		followSymlinks := true
		if kw != nil {
			if v, ok := kw.GetStr("follow_symlinks"); ok {
				followSymlinks = object.Truthy(v)
			}
		}
		var err error
		if followSymlinks {
			_, err = os.Stat(p)
		} else {
			_, err = os.Lstat(p)
		}
		return object.BoolOf(!os.IsNotExist(err) && err == nil), nil
	}})

	cls.Dict.SetStr("is_file", &object.BuiltinFunc{Name: "is_file", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		info, err := os.Stat(pathStr(inst))
		if err != nil {
			return object.BoolOf(false), nil
		}
		return object.BoolOf(info.Mode().IsRegular()), nil
	}})

	cls.Dict.SetStr("is_dir", &object.BuiltinFunc{Name: "is_dir", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		info, err := os.Stat(pathStr(inst))
		if err != nil {
			return object.BoolOf(false), nil
		}
		return object.BoolOf(info.IsDir()), nil
	}})

	cls.Dict.SetStr("is_symlink", &object.BuiltinFunc{Name: "is_symlink", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		info, err := os.Lstat(pathStr(inst))
		if err != nil {
			return object.BoolOf(false), nil
		}
		return object.BoolOf(info.Mode()&os.ModeSymlink != 0), nil
	}})

	cls.Dict.SetStr("is_mount", &object.BuiltinFunc{Name: "is_mount", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// Simplified: a path is a mount point if it's a dir and its parent's device differs.
		// On most POSIX systems, this is expensive to check precisely; return False for non-root.
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		info, err := os.Lstat(p)
		if err != nil || !info.IsDir() {
			return object.BoolOf(false), nil
		}
		// Root is always a mount point
		if p == "/" {
			return object.BoolOf(true), nil
		}
		return object.BoolOf(false), nil
	}})

	cls.Dict.SetStr("is_junction", &object.BuiltinFunc{Name: "is_junction", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(false), nil // POSIX has no junctions
	}})

	cls.Dict.SetStr("is_block_device", &object.BuiltinFunc{Name: "is_block_device", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		info, err := os.Stat(pathStr(inst))
		if err != nil {
			return object.BoolOf(false), nil
		}
		return object.BoolOf(info.Mode()&os.ModeDevice != 0 && info.Mode()&os.ModeCharDevice == 0), nil
	}})

	cls.Dict.SetStr("is_char_device", &object.BuiltinFunc{Name: "is_char_device", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		info, err := os.Stat(pathStr(inst))
		if err != nil {
			return object.BoolOf(false), nil
		}
		return object.BoolOf(info.Mode()&os.ModeCharDevice != 0), nil
	}})

	cls.Dict.SetStr("is_fifo", &object.BuiltinFunc{Name: "is_fifo", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		info, err := os.Stat(pathStr(inst))
		if err != nil {
			return object.BoolOf(false), nil
		}
		return object.BoolOf(info.Mode()&os.ModeNamedPipe != 0), nil
	}})

	cls.Dict.SetStr("is_socket", &object.BuiltinFunc{Name: "is_socket", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		info, err := os.Stat(pathStr(inst))
		if err != nil {
			return object.BoolOf(false), nil
		}
		return object.BoolOf(info.Mode()&os.ModeSocket != 0), nil
	}})

	// resolve() — absolute path resolving symlinks
	cls.Dict.SetStr("resolve", &object.BuiltinFunc{Name: "resolve", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		resolved, err := filepath.EvalSymlinks(abs)
		if err != nil {
			// strict=False (default): return abs even if path doesn't exist
			return newPathInst(cls, abs), nil
		}
		return newPathInst(cls, resolved), nil
	}})

	// absolute() — make absolute without resolving symlinks
	cls.Dict.SetStr("absolute", &object.BuiltinFunc{Name: "absolute", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		abs, err := filepath.Abs(pathStr(inst))
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return newPathInst(cls, abs), nil
	}})

	// expanduser() — expand ~ and ~user
	cls.Dict.SetStr("expanduser", &object.BuiltinFunc{Name: "expanduser", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		if !strings.HasPrefix(p, "~") {
			return newPathInst(cls, p), nil
		}
		u, err := user.Current()
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		if p == "~" {
			return newPathInst(cls, u.HomeDir), nil
		}
		if strings.HasPrefix(p, "~/") {
			return newPathInst(cls, filepath.Join(u.HomeDir, p[2:])), nil
		}
		// ~username/rest — simplified: only support current user
		return newPathInst(cls, p), nil
	}})

	// iterdir()
	cls.Dict.SetStr("iterdir", &object.BuiltinFunc{Name: "iterdir", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		entries, err := os.ReadDir(p)
		if os.IsNotExist(err) {
			return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: '%s'", p)
		}
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
			return newPathInst(cls, filepath.Join(p, e.Name())), true, nil
		}}, nil
	}})

	// glob(pattern)
	cls.Dict.SetStr("glob", &object.BuiltinFunc{Name: "glob", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "glob() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		pat, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "pattern must be str")
		}
		p := pathStr(inst)
		matches, err := filepath.Glob(filepath.Join(p, pat.V))
		if err != nil {
			return nil, object.Errorf(i.valueErr, "invalid pattern: %v", err)
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(matches) {
				return nil, false, nil
			}
			m := matches[idx]
			idx++
			return newPathInst(cls, m), true, nil
		}}, nil
	}})

	// rglob(pattern) — recursive glob
	cls.Dict.SetStr("rglob", &object.BuiltinFunc{Name: "rglob", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "rglob() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		pat, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "pattern must be str")
		}
		p := pathStr(inst)
		var results []string
		_ = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			name := filepath.Base(path)
			matched, e := filepath.Match(pat.V, name)
			if e == nil && matched && path != p {
				results = append(results, path)
			}
			return nil
		})
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(results) {
				return nil, false, nil
			}
			m := results[idx]
			idx++
			return newPathInst(cls, m), true, nil
		}}, nil
	}})

	// walk(top_down=True, on_error=None, follow_symlinks=False)
	cls.Dict.SetStr("walk", &object.BuiltinFunc{Name: "walk", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		type walkEntry struct {
			dir     string
			subdirs []string
			files   []string
		}
		var entries []walkEntry
		_ = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				var subdirs, files []string
				children, _ := os.ReadDir(path)
				for _, c := range children {
					if c.IsDir() {
						subdirs = append(subdirs, c.Name())
					} else {
						files = append(files, c.Name())
					}
				}
				entries = append(entries, walkEntry{dir: path, subdirs: subdirs, files: files})
			}
			return nil
		})
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(entries) {
				return nil, false, nil
			}
			e := entries[idx]
			idx++
			dirPath := newPathInst(cls, e.dir)
			subdirList := make([]object.Object, len(e.subdirs))
			for k, s := range e.subdirs {
				subdirList[k] = &object.Str{V: s}
			}
			fileList := make([]object.Object, len(e.files))
			for k, f := range e.files {
				fileList[k] = &object.Str{V: f}
			}
			tup := &object.Tuple{V: []object.Object{
				dirPath,
				&object.List{V: subdirList},
				&object.List{V: fileList},
			}}
			return tup, true, nil
		}}, nil
	}})

	// open(mode='r', buffering=-1, encoding=None, errors=None, newline=None)
	cls.Dict.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		mode := "r"
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				mode = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("mode"); ok {
				if s, ok := v.(*object.Str); ok {
					mode = s.V
				}
			}
		}
		// Delegate to the vm's open() builtin via the interp
		return i.openFile(p, mode)
	}})

	// read_bytes()
	cls.Dict.SetStr("read_bytes", &object.BuiltinFunc{Name: "read_bytes", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		data, err := os.ReadFile(p)
		if os.IsNotExist(err) {
			return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: '%s'", p)
		}
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Bytes{V: data}, nil
	}})

	// read_text(encoding='utf-8', errors='strict', newline=None)
	cls.Dict.SetStr("read_text", &object.BuiltinFunc{Name: "read_text", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		data, err := os.ReadFile(p)
		if os.IsNotExist(err) {
			return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: '%s'", p)
		}
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Str{V: string(data)}, nil
	}})

	// write_bytes(data)
	cls.Dict.SetStr("write_bytes", &object.BuiltinFunc{Name: "write_bytes", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "write_bytes() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		var data []byte
		switch v := a[1].(type) {
		case *object.Bytes:
			data = v.V
		case *object.Bytearray:
			data = v.V
		default:
			return nil, object.Errorf(i.typeErr, "data must be bytes-like")
		}
		err := os.WriteFile(p, data, 0o666)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.NewInt(int64(len(data))), nil
	}})

	// write_text(data, encoding=None, errors=None, newline=None)
	cls.Dict.SetStr("write_text", &object.BuiltinFunc{Name: "write_text", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "write_text() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		s, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "data must be str")
		}
		err := os.WriteFile(p, []byte(s.V), 0o666)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.NewInt(int64(len(s.V))), nil
	}})

	// touch(mode=0o666, exist_ok=True)
	cls.Dict.SetStr("touch", &object.BuiltinFunc{Name: "touch", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, 0o666)
		if err != nil {
			if os.IsExist(err) {
				return object.None, nil
			}
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		f.Close()
		return object.None, nil
	}})

	// mkdir(mode=0o777, parents=False, exist_ok=False)
	cls.Dict.SetStr("mkdir", &object.BuiltinFunc{Name: "mkdir", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		parents := false
		existOk := false
		if kw != nil {
			if v, ok := kw.GetStr("parents"); ok {
				parents = object.Truthy(v)
			}
			if v, ok := kw.GetStr("exist_ok"); ok {
				existOk = object.Truthy(v)
			}
		}
		var err error
		if parents {
			err = os.MkdirAll(p, 0o777)
		} else {
			err = os.Mkdir(p, 0o777)
		}
		if err != nil {
			if os.IsExist(err) && existOk {
				return object.None, nil
			}
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// chmod(mode, *, follow_symlinks=True)
	cls.Dict.SetStr("chmod", &object.BuiltinFunc{Name: "chmod", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "chmod() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		mode, ok := toInt64(a[1])
		if !ok {
			return nil, object.Errorf(i.typeErr, "mode must be int")
		}
		err := os.Chmod(p, fs.FileMode(mode))
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// unlink(missing_ok=False)
	cls.Dict.SetStr("unlink", &object.BuiltinFunc{Name: "unlink", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		missingOk := false
		if kw != nil {
			if v, ok := kw.GetStr("missing_ok"); ok {
				missingOk = object.Truthy(v)
			}
		}
		err := os.Remove(p)
		if err != nil {
			if os.IsNotExist(err) && missingOk {
				return object.None, nil
			}
			if os.IsNotExist(err) {
				return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: '%s'", p)
			}
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// rmdir()
	cls.Dict.SetStr("rmdir", &object.BuiltinFunc{Name: "rmdir", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		err := os.Remove(p)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// rename(target)
	cls.Dict.SetStr("rename", &object.BuiltinFunc{Name: "rename", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "rename() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		var target string
		switch v := a[1].(type) {
		case *object.Str:
			target = v.V
		case *object.Instance:
			target = pathStr(v)
		default:
			return nil, object.Errorf(i.typeErr, "target must be str or Path")
		}
		err := os.Rename(p, target)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return newPathInst(cls, target), nil
	}})

	// replace(target) — same as rename but overwrites
	cls.Dict.SetStr("replace", &object.BuiltinFunc{Name: "replace", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "replace() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		var target string
		switch v := a[1].(type) {
		case *object.Str:
			target = v.V
		case *object.Instance:
			target = pathStr(v)
		default:
			return nil, object.Errorf(i.typeErr, "target must be str or Path")
		}
		err := os.Rename(p, target)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return newPathInst(cls, target), nil
	}})

	// symlink_to(target, target_is_directory=False)
	cls.Dict.SetStr("symlink_to", &object.BuiltinFunc{Name: "symlink_to", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "symlink_to() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		var target string
		switch v := a[1].(type) {
		case *object.Str:
			target = v.V
		case *object.Instance:
			target = pathStr(v)
		default:
			return nil, object.Errorf(i.typeErr, "target must be str or Path")
		}
		err := os.Symlink(target, p)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// hardlink_to(target)
	cls.Dict.SetStr("hardlink_to", &object.BuiltinFunc{Name: "hardlink_to", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "hardlink_to() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		var target string
		switch v := a[1].(type) {
		case *object.Str:
			target = v.V
		case *object.Instance:
			target = pathStr(v)
		default:
			return nil, object.Errorf(i.typeErr, "target must be str or Path")
		}
		err := os.Link(target, p)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.None, nil
	}})

	// readlink()
	cls.Dict.SetStr("readlink", &object.BuiltinFunc{Name: "readlink", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		target, err := os.Readlink(p)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return newPathInst(cls, target), nil
	}})

	// samefile(other)
	cls.Dict.SetStr("samefile", &object.BuiltinFunc{Name: "samefile", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "samefile() takes 1 argument")
		}
		inst := a[0].(*object.Instance)
		p := pathStr(inst)
		var other string
		switch v := a[1].(type) {
		case *object.Str:
			other = v.V
		case *object.Instance:
			other = pathStr(v)
		default:
			return nil, object.Errorf(i.typeErr, "argument must be str or Path")
		}
		a1, _ := filepath.Abs(p)
		a2, _ := filepath.Abs(other)
		return object.BoolOf(a1 == a2), nil
	}})

	// Inherit pure-path properties and methods from pureCls by delegation.
	// Properties are copied directly so that they work on Path instances.
	purePropNames := []string{
		"drive", "root", "anchor", "name", "suffix", "suffixes", "stem",
		"parent", "parents", "parts",
	}
	for _, pname := range purePropNames {
		v, ok := pureCls.Dict.GetStr(pname)
		if ok {
			cls.Dict.SetStr(pname, v)
		}
	}

	pureMethNames := []string{
		"__str__", "__repr__", "__fspath__", "__eq__", "__lt__", "__le__",
		"__gt__", "__ge__", "__hash__", "__truediv__", "__rtruediv__",
		"as_posix", "is_absolute", "is_relative_to", "relative_to",
		"with_name", "with_stem", "with_suffix", "with_segments",
		"joinpath", "match", "full_match",
	}
	for _, mname := range pureMethNames {
		v, ok := pureCls.Dict.GetStr(mname)
		if ok {
			cls.Dict.SetStr(mname, v)
		}
	}

	return cls
}

// openFile opens a file and returns a file object; delegates to the vm's file
// object creation. We reuse the same pattern as the builtin open().
func (i *Interp) openFile(path, mode string) (object.Object, error) {
	flags := os.O_RDONLY
	create := false
	trunc := false
	append_ := false
	binary := false

	for _, c := range mode {
		switch c {
		case 'r':
			flags = os.O_RDONLY
		case 'w':
			flags = os.O_WRONLY
			create = true
			trunc = true
		case 'a':
			flags = os.O_WRONLY
			create = true
			append_ = true
		case 'x':
			flags = os.O_WRONLY
			create = true
		case '+':
			flags = os.O_RDWR
		case 'b':
			binary = true
		case 't':
			binary = false
		}
	}
	_ = binary
	if create {
		flags |= os.O_CREATE
	}
	if trunc {
		flags |= os.O_TRUNC
	}
	if append_ {
		flags |= os.O_APPEND
	}
	f, err := os.OpenFile(path, flags, 0o666)
	if os.IsNotExist(err) {
		return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: '%s'", path)
	}
	if err != nil {
		return nil, object.Errorf(i.osErr, "%v", err)
	}
	return &object.File{F: f, FilePath: path, Mode: mode, Binary: binary}, nil
}

// buildTempfileOld was the minimal tempfile; the full implementation is in stdlib_tempfile.go.
// Kept here for reference but renamed to avoid collision.
func (i *Interp) buildTempfileOld() *object.Module {
	m := &object.Module{Name: "tempfile", Dict: object.NewDict()}

	// gettempdir() — return the default temp directory
	m.Dict.SetStr("gettempdir", &object.BuiltinFunc{Name: "gettempdir", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: os.TempDir()}, nil
	}})

	// mkdtemp(suffix='', prefix='tmp', dir=None) — create and return a temp dir path
	m.Dict.SetStr("mkdtemp", &object.BuiltinFunc{Name: "mkdtemp", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		suffix := ""
		prefix := "tmp"
		dir := os.TempDir()
		if kw != nil {
			if v, ok := kw.GetStr("suffix"); ok {
				if s, ok := v.(*object.Str); ok {
					suffix = s.V
				}
			}
			if v, ok := kw.GetStr("prefix"); ok {
				if s, ok := v.(*object.Str); ok {
					prefix = s.V
				}
			}
			if v, ok := kw.GetStr("dir"); ok {
				if s, ok := v.(*object.Str); ok {
					dir = s.V
				}
			}
		}
		pattern := prefix + "*" + suffix
		d, err := os.MkdirTemp(dir, pattern)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Str{V: d}, nil
	}})

	// TemporaryDirectory(suffix=None, prefix=None, dir=None, delete=True) — context manager
	m.Dict.SetStr("TemporaryDirectory", &object.BuiltinFunc{Name: "TemporaryDirectory", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		suffix := ""
		prefix := "tmp"
		dir := os.TempDir()
		if kw != nil {
			if v, ok := kw.GetStr("suffix"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					suffix = s.V
				}
			}
			if v, ok := kw.GetStr("prefix"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					prefix = s.V
				}
			}
			if v, ok := kw.GetStr("dir"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					dir = s.V
				}
			}
		}
		pattern := prefix + "*" + suffix
		d, err := os.MkdirTemp(dir, pattern)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		// Return a context manager object
		tdCls := &object.Class{Name: "TemporaryDirectory", Dict: object.NewDict()}
		tdInst := &object.Instance{Class: tdCls, Dict: object.NewDict()}
		tdInst.Dict.SetStr("name", &object.Str{V: d})
		tdCls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: d}, nil
		}})
		tdCls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			os.RemoveAll(d)
			return object.BoolOf(false), nil
		}})
		return tdInst, nil
	}})

	// mkstemp(suffix='', prefix='tmp', dir=None, text=False) — create temp file, return (fd, name)
	m.Dict.SetStr("mkstemp", &object.BuiltinFunc{Name: "mkstemp", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		suffix := ""
		prefix := "tmp"
		dir := os.TempDir()
		if kw != nil {
			if v, ok := kw.GetStr("suffix"); ok {
				if s, ok := v.(*object.Str); ok {
					suffix = s.V
				}
			}
			if v, ok := kw.GetStr("prefix"); ok {
				if s, ok := v.(*object.Str); ok {
					prefix = s.V
				}
			}
			if v, ok := kw.GetStr("dir"); ok {
				if s, ok := v.(*object.Str); ok {
					dir = s.V
				}
			}
		}
		f, err := os.CreateTemp(dir, prefix+"*"+suffix)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		name := f.Name()
		fd := int(f.Fd())
		f.Close()
		return &object.Tuple{V: []object.Object{object.NewInt(int64(fd)), &object.Str{V: name}}}, nil
	}})

	return m
}

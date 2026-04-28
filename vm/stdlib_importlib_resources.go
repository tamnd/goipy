package vm

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildImportlibResources() *object.Module {
	m := &object.Module{Name: "importlib.resources", Dict: object.NewDict()}

	// ── Traversable class ─────────────────────────────────────────────────────

	traversableCls := &object.Class{Name: "Traversable", Dict: object.NewDict()}
	pathCls := &object.Class{Name: "Path", Dict: object.NewDict()}

	// makeTraversable builds a Traversable for the given filesystem path.
	// fsPath may be a directory (package root) or a file (resource).
	var makeTraversable func(fsPath string) *object.Instance
	makeTraversable = func(fsPath string) *object.Instance {
		inst := &object.Instance{Class: traversableCls, Dict: object.NewDict()}
		nameVal := filepath.Base(fsPath)
		inst.Dict.SetStr("name", &object.Str{V: nameVal})
		inst.Dict.SetStr("_path", &object.Str{V: fsPath})

		inst.Dict.SetStr("is_dir", &object.BuiltinFunc{Name: "is_dir",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				info, err := os.Stat(fsPath)
				if err != nil {
					return object.False, nil
				}
				return object.BoolOf(info.IsDir()), nil
			}})

		inst.Dict.SetStr("is_file", &object.BuiltinFunc{Name: "is_file",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				info, err := os.Stat(fsPath)
				if err != nil {
					return object.False, nil
				}
				return object.BoolOf(!info.IsDir()), nil
			}})

		inst.Dict.SetStr("iterdir", &object.BuiltinFunc{Name: "iterdir",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				entries, err := os.ReadDir(fsPath)
				if err != nil {
					return &object.Iter{Next: func() (object.Object, bool, error) {
						return nil, false, nil
					}}, nil
				}
				idx := 0
				return &object.Iter{Next: func() (object.Object, bool, error) {
					if idx >= len(entries) {
						return nil, false, nil
					}
					e := entries[idx]
					idx++
					child := makeTraversable(filepath.Join(fsPath, e.Name()))
					return child, true, nil
				}}, nil
			}})

		inst.Dict.SetStr("joinpath", &object.BuiltinFunc{Name: "joinpath",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				childPath := fsPath
				// skip self if called as bound method
				start := 0
				if len(a) > 0 {
					if a[0] == inst {
						start = 1
					}
				}
				for _, arg := range a[start:] {
					if s, ok := arg.(*object.Str); ok {
						childPath = filepath.Join(childPath, s.V)
					}
				}
				return makeTraversable(childPath), nil
			}})

		inst.Dict.SetStr("open", &object.BuiltinFunc{Name: "open",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				mode := "r"
				encoding := "utf-8"
				// a[0] may be self (Instance) — skip it
				args := a
				if len(args) > 0 {
					if _, ok := args[0].(*object.Instance); ok {
						args = args[1:]
					}
				}
				if len(args) >= 1 {
					if s, ok := args[0].(*object.Str); ok {
						mode = s.V
					}
				}
				if kw != nil {
					if v, ok := kw.GetStr("encoding"); ok && v != object.None {
						if s, ok2 := v.(*object.Str); ok2 {
							encoding = s.V
						}
					}
				}
				data, err := os.ReadFile(fsPath)
				if err != nil {
					return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: %q", fsPath)
				}
				if strings.Contains(mode, "b") {
					return makeBytesCM(data), nil
				}
				_ = encoding
				return makeTextCM(string(data)), nil
			}})

		inst.Dict.SetStr("read_bytes", &object.BuiltinFunc{Name: "read_bytes",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				data, err := os.ReadFile(fsPath)
				if err != nil {
					return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: %q", fsPath)
				}
				return &object.Bytes{V: data}, nil
			}})

		inst.Dict.SetStr("read_text", &object.BuiltinFunc{Name: "read_text",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				data, err := os.ReadFile(fsPath)
				if err != nil {
					return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: %q", fsPath)
				}
				return &object.Str{V: string(data)}, nil
			}})

		return inst
	}

	// makePathInst creates a simple Path-like instance wrapping a filesystem path.
	makePathInst := func(fsPath string) *object.Instance {
		inst := &object.Instance{Class: pathCls, Dict: object.NewDict()}
		inst.Dict.SetStr("_path", &object.Str{V: fsPath})
		inst.Dict.SetStr("name", &object.Str{V: filepath.Base(fsPath)})
		inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.Str{V: fsPath}, nil
			}})
		inst.Dict.SetStr("__fspath__", &object.BuiltinFunc{Name: "__fspath__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.Str{V: fsPath}, nil
			}})
		inst.Dict.SetStr("exists", &object.BuiltinFunc{Name: "exists",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				_, err := os.Stat(fsPath)
				return object.BoolOf(err == nil), nil
			}})
		inst.Dict.SetStr("read_text", &object.BuiltinFunc{Name: "read_text",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				data, err := os.ReadFile(fsPath)
				if err != nil {
					return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file: %q", fsPath)
				}
				return &object.Str{V: string(data)}, nil
			}})
		inst.Dict.SetStr("read_bytes", &object.BuiltinFunc{Name: "read_bytes",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				data, err := os.ReadFile(fsPath)
				if err != nil {
					return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file: %q", fsPath)
				}
				return &object.Bytes{V: data}, nil
			}})
		return inst
	}

	// pkgDir resolves a package/module anchor to its filesystem directory.
	pkgDir := func(anchor object.Object) string {
		var modName string
		switch v := anchor.(type) {
		case *object.Str:
			modName = v.V
		case *object.Module:
			if v.Path != "" {
				return filepath.Dir(v.Path)
			}
			modName = v.Name
		default:
			return ""
		}
		mod, err := i.loadModule(modName)
		if err != nil || mod == nil {
			return ""
		}
		if mod.Path != "" {
			return filepath.Dir(mod.Path)
		}
		return ""
	}

	// ── files(anchor) ─────────────────────────────────────────────────────────

	m.Dict.SetStr("files", &object.BuiltinFunc{Name: "files",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "files() requires an anchor argument")
			}
			dir := pkgDir(a[0])
			if dir == "" {
				// Builtin Go module — return a stub traversable with no entries.
				name := ""
				if s, ok := a[0].(*object.Str); ok {
					name = s.V
				} else if mod, ok := a[0].(*object.Module); ok {
					name = mod.Name
				}
				return makeTraversable("/no_such_dir_goipy_" + name), nil
			}
			return makeTraversable(dir), nil
		}})

	// ── as_file(path) ─────────────────────────────────────────────────────────

	makeCtxMgr := func(enter func() object.Object) *object.Instance {
		ctxCls := &object.Class{Name: "as_file_ctx", Dict: object.NewDict()}
		ctx := &object.Instance{Class: ctxCls, Dict: object.NewDict()}
		ctx.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return enter(), nil
			}})
		ctx.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.False, nil
			}})
		return ctx
	}

	// as_file does NOT use mpArgs — first arg is a Traversable (*object.Instance).
	m.Dict.SetStr("as_file", &object.BuiltinFunc{Name: "as_file",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "as_file() requires a Traversable")
			}
			trav := a[0]
			fsPath := ""
			if inst, ok := trav.(*object.Instance); ok {
				if v, ok2 := inst.Dict.GetStr("_path"); ok2 {
					if s, ok3 := v.(*object.Str); ok3 {
						fsPath = s.V
					}
				}
			}
			fp := fsPath
			return makeCtxMgr(func() object.Object {
				return makePathInst(fp)
			}), nil
		}})

	// ── path(anchor, *path_names) ─────────────────────────────────────────────

	m.Dict.SetStr("path", &object.BuiltinFunc{Name: "path",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "path() requires an anchor")
			}
			dir := pkgDir(a[0])
			parts := []string{dir}
			for _, arg := range a[1:] {
				if s, ok := arg.(*object.Str); ok {
					parts = append(parts, s.V)
				}
			}
			fsPath := filepath.Join(parts...)
			return makeCtxMgr(func() object.Object {
				return makePathInst(fsPath)
			}), nil
		}})

	// ── read_binary(anchor, *path_names) ──────────────────────────────────────

	m.Dict.SetStr("read_binary", &object.BuiltinFunc{Name: "read_binary",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			fsPath := irResolvePath(pkgDir, a)
			data, err := os.ReadFile(fsPath)
			if err != nil {
				return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: %q", fsPath)
			}
			return &object.Bytes{V: data}, nil
		}})

	// ── read_text(anchor, *path_names, encoding, errors) ─────────────────────

	m.Dict.SetStr("read_text", &object.BuiltinFunc{Name: "read_text",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			fsPath := irResolvePath(pkgDir, a)
			data, err := os.ReadFile(fsPath)
			if err != nil {
				return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: %q", fsPath)
			}
			return &object.Str{V: string(data)}, nil
		}})

	// ── open_binary(anchor, *path_names) ─────────────────────────────────────

	m.Dict.SetStr("open_binary", &object.BuiltinFunc{Name: "open_binary",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			fsPath := irResolvePath(pkgDir, a)
			data, err := os.ReadFile(fsPath)
			if err != nil {
				return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: %q", fsPath)
			}
			return makeBytesCM(data), nil
		}})

	// ── open_text(anchor, *path_names, encoding, errors) ─────────────────────

	m.Dict.SetStr("open_text", &object.BuiltinFunc{Name: "open_text",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			fsPath := irResolvePath(pkgDir, a)
			data, err := os.ReadFile(fsPath)
			if err != nil {
				return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: %q", fsPath)
			}
			return makeTextCM(string(data)), nil
		}})

	// ── is_resource(anchor, name) ─────────────────────────────────────────────

	m.Dict.SetStr("is_resource", &object.BuiltinFunc{Name: "is_resource",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			fsPath := irResolvePath(pkgDir, a)
			info, err := os.Stat(fsPath)
			if err != nil {
				return object.False, nil
			}
			return object.BoolOf(!info.IsDir()), nil
		}})

	// ── contents(anchor) ──────────────────────────────────────────────────────

	m.Dict.SetStr("contents", &object.BuiltinFunc{Name: "contents",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return &object.List{V: nil}, nil
			}
			dir := pkgDir(a[0])
			if dir == "" {
				return &object.List{V: nil}, nil
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				return &object.List{V: nil}, nil
			}
			items := make([]object.Object, 0, len(entries))
			for _, e := range entries {
				items = append(items, &object.Str{V: e.Name()})
			}
			return &object.List{V: items}, nil
		}})

	// Type aliases
	m.Dict.SetStr("Package", &object.Str{V: "Package"})
	m.Dict.SetStr("Anchor", &object.Str{V: "Anchor"})
	m.Dict.SetStr("ResourceReader", &object.Class{Name: "ResourceReader", Dict: object.NewDict()})

	return m
}

// ── importlib.resources.abc ───────────────────────────────────────────────────

func (i *Interp) buildImportlibResourcesAbc() *object.Module {
	m := &object.Module{Name: "importlib.resources.abc", Dict: object.NewDict()}
	for _, name := range []string{"Traversable", "TraversableResources", "ResourceReader"} {
		n := name
		cls := &object.Class{Name: n, Dict: object.NewDict()}
		m.Dict.SetStr(n, cls)
	}
	errCls := &object.Class{
		Name:  "TraversalError",
		Dict:  object.NewDict(),
		Bases: []*object.Class{i.exception},
	}
	m.Dict.SetStr("TraversalError", errCls)
	return m
}

// irResolvePath builds a filesystem path from (anchor, *path_names) args.
func irResolvePath(pkgDir func(object.Object) string, a []object.Object) string {
	if len(a) == 0 {
		return ""
	}
	dir := pkgDir(a[0])
	if dir == "" {
		return ""
	}
	parts := []string{dir}
	for _, arg := range a[1:] {
		if s, ok := arg.(*object.Str); ok {
			parts = append(parts, s.V)
		}
	}
	return filepath.Join(parts...)
}

// ── file-like context managers ────────────────────────────────────────────────

// makeBytesCM returns a BytesIO-like context manager.
func makeBytesCM(data []byte) *object.Instance {
	cls := &object.Class{Name: "BytesIO", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	pos := 0
	buf := data

	inst.Dict.SetStr("read", &object.BuiltinFunc{Name: "read",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// skip self if instance
			args := a
			if len(args) > 0 {
				if _, ok := args[0].(*object.Instance); ok {
					args = args[1:]
				}
			}
			n := -1
			if len(args) > 0 {
				if ni, ok := toInt64(args[0]); ok {
					n = int(ni)
				}
			}
			if n < 0 || pos+n > len(buf) {
				result := buf[pos:]
				pos = len(buf)
				return &object.Bytes{V: result}, nil
			}
			result := buf[pos : pos+n]
			pos += n
			return &object.Bytes{V: result}, nil
		}})

	inst.Dict.SetStr("readline", &object.BuiltinFunc{Name: "readline",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if pos >= len(buf) {
				return &object.Bytes{V: nil}, nil
			}
			end := pos
			for end < len(buf) && buf[end] != '\n' {
				end++
			}
			if end < len(buf) {
				end++
			}
			line := buf[pos:end]
			pos = end
			return &object.Bytes{V: line}, nil
		}})

	inst.Dict.SetStr("readlines", &object.BuiltinFunc{Name: "readlines",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			var lines []object.Object
			start := pos
			for pos <= len(buf) {
				if pos == len(buf) || buf[pos] == '\n' {
					end := pos
					if pos < len(buf) {
						end = pos + 1
					}
					if end > start {
						lines = append(lines, &object.Bytes{V: append([]byte(nil), buf[start:end]...)})
					}
					start = end
					pos = end
					if pos >= len(buf) {
						break
					}
				} else {
					pos++
				}
			}
			pos = len(buf)
			return &object.List{V: lines}, nil
		}})

	inst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})

	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		}})

	return inst
}

// makeTextCM returns a StringIO-like context manager.
func makeTextCM(text string) *object.Instance {
	cls := &object.Class{Name: "StringIO", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	pos := 0
	buf := []rune(text)

	inst.Dict.SetStr("read", &object.BuiltinFunc{Name: "read",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			args := a
			if len(args) > 0 {
				if _, ok := args[0].(*object.Instance); ok {
					args = args[1:]
				}
			}
			n := -1
			if len(args) > 0 {
				if ni, ok := toInt64(args[0]); ok {
					n = int(ni)
				}
			}
			if n < 0 || pos+n > len(buf) {
				result := string(buf[pos:])
				pos = len(buf)
				return &object.Str{V: result}, nil
			}
			result := string(buf[pos : pos+n])
			pos += n
			return &object.Str{V: result}, nil
		}})

	inst.Dict.SetStr("readline", &object.BuiltinFunc{Name: "readline",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if pos >= len(buf) {
				return &object.Str{V: ""}, nil
			}
			end := pos
			for end < len(buf) && buf[end] != '\n' {
				end++
			}
			if end < len(buf) {
				end++
			}
			line := string(buf[pos:end])
			pos = end
			return &object.Str{V: line}, nil
		}})

	inst.Dict.SetStr("readlines", &object.BuiltinFunc{Name: "readlines",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			var lines []object.Object
			for _, line := range strings.Split(string(buf[pos:]), "\n") {
				lines = append(lines, &object.Str{V: line + "\n"})
			}
			// Remove the trailing empty line added by Split.
			if len(lines) > 0 {
				if last, ok := lines[len(lines)-1].(*object.Str); ok && last.V == "\n" {
					lines = lines[:len(lines)-1]
				}
			}
			pos = len(buf)
			return &object.List{V: lines}, nil
		}})

	inst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})

	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		}})

	return inst
}

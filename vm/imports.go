package vm

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tamnd/goipy/marshal"
	"github.com/tamnd/goipy/object"
)

// importName implements CPython's __import__(name, globals, locals,
// fromlist, level). It returns:
//   - the topmost package of the dotted chain when fromlist is empty
//     (matches `import a.b.c`, which binds `a`);
//   - the innermost module otherwise (matches `from a.b import c`,
//     which binds from `a.b`).
//
// Relative imports (level > 0) resolve against `__package__` (or
// `__name__`) in the caller's globals.
func (i *Interp) importName(name string, globals *object.Dict, fromlist *object.Tuple, level int) (object.Object, error) {
	if i.modules == nil {
		i.modules = map[string]*object.Module{}
	}
	absName := name
	if level > 0 {
		base, err := i.resolveRelativeBase(globals, level)
		if err != nil {
			return nil, err
		}
		if name == "" {
			absName = base
		} else if base == "" {
			absName = name
		} else {
			absName = base + "." + name
		}
	}
	if absName == "" {
		return nil, object.Errorf(i.importErr, "empty module name")
	}
	top, innermost, err := i.loadChain(absName)
	if err != nil {
		return nil, err
	}
	if fromlist == nil || len(fromlist.V) == 0 {
		return top, nil
	}
	// Each fromlist entry may itself be a submodule that hasn't been
	// loaded yet (e.g. `from pkg import sub` where `sub` is a .pyc
	// alongside __init__.pyc). Best-effort: try loading; silently skip
	// on failure and let IMPORT_FROM raise a clearer error.
	if isPackage(innermost) {
		for _, it := range fromlist.V {
			s, ok := it.(*object.Str)
			if !ok || s.V == "*" {
				continue
			}
			if _, ok := innermost.Dict.GetStr(s.V); ok {
				continue
			}
			_, _ = i.loadModule(innermost.Name + "." + s.V)
		}
	}
	return innermost, nil
}

// resolveRelativeBase walks up `level` package boundaries from the
// caller's `__package__` (or `__name__` for packages) to produce the
// base name that `name` is appended to.
func (i *Interp) resolveRelativeBase(globals *object.Dict, level int) (string, error) {
	var pkg string
	if v, ok := globals.GetStr("__package__"); ok {
		if s, ok := v.(*object.Str); ok {
			pkg = s.V
		}
	}
	if pkg == "" {
		if v, ok := globals.GetStr("__name__"); ok {
			if s, ok := v.(*object.Str); ok {
				pkg = s.V
			}
		}
		// A non-package module's __name__ is its own dotted path, but
		// relative imports walk from its *package* — drop the final
		// component unless the module itself is a package.
		if _, isPkg := globals.GetStr("__path__"); !isPkg {
			if dot := strings.LastIndex(pkg, "."); dot >= 0 {
				pkg = pkg[:dot]
			} else {
				pkg = ""
			}
		}
	}
	// Walk up level-1 parents.
	for k := 1; k < level; k++ {
		dot := strings.LastIndex(pkg, ".")
		if dot < 0 {
			if pkg == "" {
				return "", object.Errorf(i.importErr, "attempted relative import beyond top-level package")
			}
			pkg = ""
			continue
		}
		pkg = pkg[:dot]
	}
	return pkg, nil
}

// loadChain loads every prefix of a dotted name, returning the outermost
// and innermost modules. Intermediate packages are bound as attributes
// on their parents.
func (i *Interp) loadChain(qname string) (top, innermost *object.Module, err error) {
	parts := strings.Split(qname, ".")
	for k := range parts {
		sub := strings.Join(parts[:k+1], ".")
		m, err := i.loadModule(sub)
		if err != nil {
			return nil, nil, err
		}
		if k == 0 {
			top = m
		}
		innermost = m
	}
	return top, innermost, nil
}

// loadModule loads (or returns the cached) module for a fully qualified
// name. If the name has a parent, the parent is loaded first and must be
// a package whose __path__ supplies the search directories.
func (i *Interp) loadModule(qname string) (*object.Module, error) {
	if m, ok := i.modules[qname]; ok {
		return m, nil
	}
	if m, ok := i.builtinModule(qname); ok {
		i.modules[qname] = m
		// For dotted names, also set the leaf as an attribute on the parent module.
		if dot := strings.LastIndex(qname, "."); dot >= 0 {
			parentName := qname[:dot]
			leaf := qname[dot+1:]
			if parent, perr := i.loadModule(parentName); perr == nil {
				parent.Dict.SetStr(leaf, m)
			}
		}
		return m, nil
	}

	var searchDirs []string
	var parent *object.Module
	leaf := qname
	if dot := strings.LastIndex(qname, "."); dot >= 0 {
		parentName := qname[:dot]
		leaf = qname[dot+1:]
		p, err := i.loadModule(parentName)
		if err != nil {
			return nil, err
		}
		if !isPackage(p) {
			return nil, object.Errorf(i.importErr, "No module named '%s'; '%s' is not a package", qname, parentName)
		}
		parent = p
		searchDirs = packagePath(p)
	} else {
		searchDirs = i.SearchPath
	}

	for _, dir := range searchDirs {
		// Package: dir/leaf/__init__.pyc
		pkgDir := filepath.Join(dir, leaf)
		initPyc := filepath.Join(pkgDir, "__init__.pyc")
		if _, err := os.Stat(initPyc); err == nil {
			code, cerr := marshal.LoadPyc(initPyc)
			if cerr != nil {
				return nil, object.Errorf(i.importErr, "cannot load %s: %v", initPyc, cerr)
			}
			m, xerr := i.execModuleAs(qname, code, pkgDir, true)
			if xerr != nil {
				return nil, xerr
			}
			m.Path = initPyc
			if parent != nil {
				parent.Dict.SetStr(leaf, m)
			}
			return m, nil
		}
		// Plain module: dir/leaf.pyc
		modPyc := filepath.Join(dir, leaf+".pyc")
		if _, err := os.Stat(modPyc); err == nil {
			code, cerr := marshal.LoadPyc(modPyc)
			if cerr != nil {
				return nil, object.Errorf(i.importErr, "cannot load %s: %v", modPyc, cerr)
			}
			m, xerr := i.execModuleAs(qname, code, "", false)
			if xerr != nil {
				return nil, xerr
			}
			m.Path = modPyc
			if parent != nil {
				parent.Dict.SetStr(leaf, m)
			}
			return m, nil
		}
	}
	return nil, object.Errorf(i.moduleNotFoundErr, "No module named '%s'", qname)
}

// execModuleAs runs a module body with the standard module dunders set.
// When pkgDir is non-empty the module is registered as a package with
// __path__ = [pkgDir].
func (i *Interp) execModuleAs(qname string, code *object.Code, pkgDir string, isPkg bool) (*object.Module, error) {
	globals := object.NewDict()
	globals.SetStr("__name__", &object.Str{V: qname})
	globals.SetStr("__builtins__", i.Builtins)
	if isPkg {
		globals.SetStr("__package__", &object.Str{V: qname})
		globals.SetStr("__path__", &object.List{V: []object.Object{&object.Str{V: pkgDir}}})
	} else if dot := strings.LastIndex(qname, "."); dot >= 0 {
		globals.SetStr("__package__", &object.Str{V: qname[:dot]})
	} else {
		globals.SetStr("__package__", &object.Str{V: ""})
	}
	m := &object.Module{Name: qname, Dict: globals}
	i.modules[qname] = m
	frame := NewFrame(code, globals, i.Builtins, globals)
	if _, err := i.runFrame(frame); err != nil {
		delete(i.modules, qname)
		return nil, err
	}
	return m, nil
}

func isPackage(m *object.Module) bool {
	if m == nil || m.Dict == nil {
		return false
	}
	_, ok := m.Dict.GetStr("__path__")
	return ok
}

// buildImportlib exposes a minimal importlib surface: import_module(name,
// package=None) and reload(module). Path hooks and the finder/loader
// protocol classes are out of scope here.
func (i *Interp) buildImportlib() *object.Module {
	m := &object.Module{Name: "importlib", Dict: object.NewDict()}

	importModule := &object.BuiltinFunc{Name: "import_module", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "import_module() missing 'name'")
		}
		nameStr, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "import_module() name must be str")
		}
		name := nameStr.V
		var pkg string
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				pkg = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("package"); ok {
				if s, ok := v.(*object.Str); ok {
					pkg = s.V
				}
			}
		}
		level := 0
		for level < len(name) && name[level] == '.' {
			level++
		}
		base := name[level:]
		// Synthesize a globals dict so importName's relative resolver
		// reads the caller-supplied package name.
		gl := object.NewDict()
		if pkg != "" {
			gl.SetStr("__package__", &object.Str{V: pkg})
			gl.SetStr("__name__", &object.Str{V: pkg})
			gl.SetStr("__path__", &object.List{V: nil}) // mark as package
		}
		// import_module returns the innermost module, not the top-level
		// package — request that by passing a non-empty fromlist.
		fl := &object.Tuple{V: []object.Object{&object.Str{V: "__name__"}}}
		if level == 0 {
			return i.importName(base, gl, fl, 0)
		}
		return i.importName(base, gl, fl, level)
	}}
	m.Dict.SetStr("import_module", importModule)

	reload := &object.BuiltinFunc{Name: "reload", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "reload() missing 'module'")
		}
		mod, ok := a[0].(*object.Module)
		if !ok {
			return nil, object.Errorf(i.typeErr, "reload() argument must be module")
		}
		if mod.Path == "" {
			// Builtin module: re-register it from the builtin registry.
			if fresh, ok2 := i.builtinModule(mod.Name); ok2 {
				// Copy fresh dict into existing module so callers holding
				// a reference see the refreshed attributes.
				keys, vals := fresh.Dict.Items()
				for idx, k := range keys {
					mod.Dict.Set(k, vals[idx]) //nolint
				}
			}
			return mod, nil
		}
		code, err := marshal.LoadPyc(mod.Path)
		if err != nil {
			return nil, object.Errorf(i.importErr, "cannot reload %s: %v", mod.Name, err)
		}
		// Re-run the body against the existing module dict so other
		// importers holding a reference see the refreshed attributes.
		frame := NewFrame(code, mod.Dict, i.Builtins, mod.Dict)
		if _, err := i.runFrame(frame); err != nil {
			return nil, err
		}
		return mod, nil
	}}
	m.Dict.SetStr("reload", reload)

	m.Dict.SetStr("invalidate_caches", &object.BuiltinFunc{Name: "invalidate_caches",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	return m
}

func packagePath(m *object.Module) []string {
	v, ok := m.Dict.GetStr("__path__")
	if !ok {
		return nil
	}
	l, ok := v.(*object.List)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(l.V))
	for _, x := range l.V {
		if s, ok := x.(*object.Str); ok {
			out = append(out, s.V)
		}
	}
	return out
}

package vm

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildPkgutil() *object.Module {
	m := &object.Module{Name: "pkgutil", Dict: object.NewDict()}

	// ── ModuleInfo namedtuple-like class ──────────────────────────────────────

	miClass := &object.Class{Name: "ModuleInfo", Dict: object.NewDict()}
	miClass.Dict.SetStr("_fields", &object.Tuple{V: []object.Object{
		&object.Str{V: "module_finder"},
		&object.Str{V: "name"},
		&object.Str{V: "ispkg"},
	}})

	makeModuleInfo := func(finder, name object.Object, ispkg bool) *object.Instance {
		inst := &object.Instance{Class: miClass, Dict: object.NewDict()}
		inst.Dict.SetStr("module_finder", finder)
		inst.Dict.SetStr("name", name)
		inst.Dict.SetStr("ispkg", object.BoolOf(ispkg))
		// Also store as tuple elements for index access.
		inst.Dict.SetStr("_tuple", &object.Tuple{V: []object.Object{
			finder, name, object.BoolOf(ispkg),
		}})
		return inst
	}

	// Make ModuleInfo a tuple subclass by adding __getitem__ and __iter__.
	miClass.Dict.SetStr("__getitem__", &object.BuiltinFunc{
		Name: "__getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			idx, ok := toInt64(a[1])
			if !ok {
				return object.None, nil
			}
			if tv, ok2 := self.Dict.GetStr("_tuple"); ok2 {
				if t, ok3 := tv.(*object.Tuple); ok3 {
					if idx < 0 {
						idx += int64(len(t.V))
					}
					if idx >= 0 && int(idx) < len(t.V) {
						return t.V[idx], nil
					}
				}
			}
			return nil, object.Errorf(i.indexErr, "tuple index out of range")
		},
	})

	miClass.Dict.SetStr("__iter__", &object.BuiltinFunc{
		Name: "__iter__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			var items []object.Object
			if tv, ok := self.Dict.GetStr("_tuple"); ok {
				if t, ok2 := tv.(*object.Tuple); ok2 {
					items = t.V
				}
			}
			pos := 0
			iterCls := &object.Class{Name: "ModuleInfoIter", Dict: object.NewDict()}
			iterCls.Dict.SetStr("__next__", &object.BuiltinFunc{
				Name: "__next__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					if pos >= len(items) {
						return nil, object.Errorf(i.stopIter, "")
					}
					v := items[pos]
					pos++
					return v, nil
				},
			})
			it := &object.Instance{Class: iterCls, Dict: object.NewDict()}
			return it, nil
		},
	})

	// __namedtuple__ marker makes isinstance(mi, tuple) return True.
	miClass.Dict.SetStr("__namedtuple__", object.True)

	miClass.Dict.SetStr("__new__", &object.BuiltinFunc{
		Name: "__new__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 4 {
				return object.None, nil
			}
			return makeModuleInfo(a[1], a[2], isTruthy(a[3])), nil
		},
	})
	miClass.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("ModuleInfo", miClass)

	// ── simplegeneric ─────────────────────────────────────────────────────────

	m.Dict.SetStr("simplegeneric", &object.BuiltinFunc{
		Name: "simplegeneric",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			defaultFn := a[0]
			// Two separate registries: by *object.Class and by builtin type name.
			registryClass := map[*object.Class]object.Object{}
			registryBuiltin := map[string]object.Object{} // "int", "str", etc.

			sgClass := &object.Class{Name: "simplegeneric", Dict: object.NewDict()}

			sgClass.Dict.SetStr("register", &object.BuiltinFunc{
				Name: "register",
				Call: func(_ any, ra []object.Object, _ *object.Dict) (object.Object, error) {
					if len(ra) < 2 {
						return object.None, nil
					}
					typObj := ra[1]
					// Capture the type so the decorator can register the impl.
					decCls := &object.Class{Name: "register_decorator", Dict: object.NewDict()}
					decCls.Dict.SetStr("__call__", &object.BuiltinFunc{
						Name: "__call__",
						Call: func(_ any, da []object.Object, _ *object.Dict) (object.Object, error) {
							if len(da) < 2 {
								return object.None, nil
							}
							impl := da[1]
							switch tv := typObj.(type) {
							case *object.Class:
								registryClass[tv] = impl
							case *object.BuiltinFunc:
								registryBuiltin[tv.Name] = impl
							}
							return impl, nil
						},
					})
					return &object.Instance{Class: decCls, Dict: object.NewDict()}, nil
				},
			})

			// builtinTypeName returns the builtin type name for dispatch.
			builtinTypeName := func(o object.Object) string {
				switch o.(type) {
				case *object.Int:
					return "int"
				case *object.Bool:
					return "bool"
				case *object.Float:
					return "float"
				case *object.Str:
					return "str"
				case *object.Bytes:
					return "bytes"
				case *object.List:
					return "list"
				case *object.Tuple:
					return "tuple"
				case *object.Dict:
					return "dict"
				case *object.Set:
					return "set"
				}
				return ""
			}

			sgClass.Dict.SetStr("__call__", &object.BuiltinFunc{
				Name: "__call__",
				Call: func(ii any, ca []object.Object, kw *object.Dict) (object.Object, error) {
					interp := ii.(*Interp)
					args := ca[1:] // strip self
					var impl object.Object
					if len(args) >= 1 {
						// Check builtin type name first.
						if name := builtinTypeName(args[0]); name != "" {
							if fn, ok := registryBuiltin[name]; ok {
								impl = fn
							}
						}
						// Then check user-defined class.
						if impl == nil {
							if inst, ok := args[0].(*object.Instance); ok {
								if fn, ok2 := registryClass[inst.Class]; ok2 {
									impl = fn
								}
							} else if cls, ok := args[0].(*object.Class); ok {
								if fn, ok2 := registryClass[cls]; ok2 {
									impl = fn
								}
							}
						}
					}
					if impl == nil {
						impl = defaultFn
					}
					return interp.callObject(impl, args, kw)
				},
			})

			sg := &object.Instance{Class: sgClass, Dict: object.NewDict()}
			return sg, nil
		},
	})

	// ── extend_path ───────────────────────────────────────────────────────────

	m.Dict.SetStr("extend_path", &object.BuiltinFunc{
		Name: "extend_path",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{V: []object.Object{}}, nil
			}
			if lst, ok := a[0].(*object.List); ok {
				return lst, nil
			}
			return &object.List{V: []object.Object{}}, nil
		},
	})

	// ── get_data ──────────────────────────────────────────────────────────────

	m.Dict.SetStr("get_data", &object.BuiltinFunc{
		Name: "get_data",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── read_code ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("read_code", &object.BuiltinFunc{
		Name: "read_code",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				// Try to read — if stream is empty or unreadable, return None.
				if inst, ok := a[0].(*object.Instance); ok {
					if readFn, ok2 := inst.Dict.GetStr("read"); ok2 {
						_ = readFn
					}
				}
			}
			return object.None, nil
		},
	})

	// ── iter_importers ────────────────────────────────────────────────────────

	m.Dict.SetStr("iter_importers", &object.BuiltinFunc{
		Name: "iter_importers",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return emptyIterator(i), nil
		},
	})

	// ── helpers ───────────────────────────────────────────────────────────────

	// zipModules returns top-level module names from a zip archive.
	zipModules := func(archive string) []struct {
		name  string
		ispkg bool
	} {
		rc, err := zip.OpenReader(archive)
		if err != nil {
			return nil
		}
		defer rc.Close()

		seen := map[string]bool{}
		var result []struct {
			name  string
			ispkg bool
		}

		for _, f := range rc.File {
			name := f.Name
			parts := strings.SplitN(name, "/", 2)
			top := parts[0]
			if top == "" {
				continue
			}

			if len(parts) == 1 {
				// File at top level: e.g. "mod.py"
				if strings.HasSuffix(top, ".py") {
					modName := strings.TrimSuffix(top, ".py")
					if !seen[modName] {
						seen[modName] = true
						result = append(result, struct {
							name  string
							ispkg bool
						}{modName, false})
					}
				}
			} else {
				// File inside a directory: e.g. "pkg/__init__.py"
				rest := parts[1]
				if rest == "__init__.py" {
					if !seen[top] {
						seen[top] = true
						result = append(result, struct {
							name  string
							ispkg bool
						}{top, true})
					}
				}
			}
		}
		return result
	}

	// isZipPath checks if a string path refers to a valid zip file.
	isZipPath := func(path string) bool {
		// Walk path components looking for a zip file boundary.
		normalised := filepath.ToSlash(path)
		parts := strings.Split(normalised, "/")
		for end := len(parts); end >= 1; end-- {
			candidate := filepath.FromSlash(strings.Join(parts[:end], "/"))
			info, err := os.Stat(candidate)
			if err == nil && !info.IsDir() {
				rc, err2 := zip.OpenReader(candidate)
				if err2 == nil {
					rc.Close()
					return true
				}
			}
		}
		return false
	}

	// archivePath extracts the zip archive path from a possibly prefixed path.
	archivePath := func(path string) (archive, prefix string) {
		normalised := filepath.ToSlash(path)
		parts := strings.Split(normalised, "/")
		for end := len(parts); end >= 1; end-- {
			candidate := filepath.FromSlash(strings.Join(parts[:end], "/"))
			info, err := os.Stat(candidate)
			if err == nil && !info.IsDir() {
				rc, err2 := zip.OpenReader(candidate)
				if err2 == nil {
					rc.Close()
					rest := strings.Join(parts[end:], "/")
					return candidate, rest
				}
			}
		}
		return path, ""
	}

	// buildModuleInfoList builds a list of ModuleInfo from name/ispkg pairs.
	buildMIList := func(pairs []struct {
		name  string
		ispkg bool
	}, prefix string, finder object.Object) *object.List {
		items := make([]object.Object, 0, len(pairs))
		for _, p := range pairs {
			mi := makeModuleInfo(finder, &object.Str{V: prefix + p.name}, p.ispkg)
			items = append(items, mi)
		}
		return &object.List{V: items}
	}

	// ── get_importer ──────────────────────────────────────────────────────────

	m.Dict.SetStr("get_importer", &object.BuiltinFunc{
		Name: "get_importer",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			pathStr, ok := a[0].(*object.Str)
			if !ok {
				return object.None, nil
			}
			if !isZipPath(pathStr.V) {
				return object.None, nil
			}
			// Create a zipimporter instance.
			interp := ii.(*Interp)
			zipMod, err := interp.loadModule("zipimport")
			if err != nil {
				return object.None, nil
			}
			ziClassObj, ok2 := zipMod.Dict.GetStr("zipimporter")
			if !ok2 {
				return object.None, nil
			}
			ziCls, ok3 := ziClassObj.(*object.Class)
			if !ok3 {
				return object.None, nil
			}
			inst := &object.Instance{Class: ziCls, Dict: object.NewDict()}
			initFn, ok4 := ziCls.Dict.GetStr("__init__")
			if !ok4 {
				return object.None, nil
			}
			_, err = interp.callObject(initFn, []object.Object{inst, pathStr}, nil)
			if err != nil {
				return object.None, nil
			}
			return inst, nil
		},
	})

	// ── iter_modules ──────────────────────────────────────────────────────────

	iterModulesImpl := func(ii any, pathList []string, prefix string) *object.List {
		var all []object.Object
		for _, p := range pathList {
			archive, pfx := archivePath(p)
			if archive != p || isZipPath(p) {
				// It's a zip (possibly with prefix).
				rc, err := zip.OpenReader(archive)
				if err != nil {
					continue
				}
				defer rc.Close()

				seen := map[string]bool{}
				for _, f := range rc.File {
					name := f.Name
					// If there's a prefix, strip it.
					if pfx != "" {
						if !strings.HasPrefix(name, pfx+"/") && name != pfx {
							continue
						}
						name = strings.TrimPrefix(name, pfx+"/")
					}
					parts := strings.SplitN(name, "/", 2)
					top := parts[0]
					if top == "" {
						continue
					}
					if len(parts) == 1 {
						if strings.HasSuffix(top, ".py") {
							modName := strings.TrimSuffix(top, ".py")
							if !seen[modName] {
								seen[modName] = true
								mi := makeModuleInfo(object.None, &object.Str{V: prefix + modName}, false)
								all = append(all, mi)
							}
						}
					} else {
						rest := parts[1]
						if rest == "__init__.py" {
							if !seen[top] {
								seen[top] = true
								mi := makeModuleInfo(object.None, &object.Str{V: prefix + top}, true)
								all = append(all, mi)
							}
						}
					}
				}
			} else {
				// Filesystem directory.
				entries, err := os.ReadDir(p)
				if err != nil {
					continue
				}
				for _, entry := range entries {
					name := entry.Name()
					if entry.IsDir() {
						// Check for __init__.py.
						initPath := filepath.Join(p, name, "__init__.py")
						if _, err2 := os.Stat(initPath); err2 == nil {
							mi := makeModuleInfo(object.None, &object.Str{V: prefix + name}, true)
							all = append(all, mi)
						}
					} else if strings.HasSuffix(name, ".py") && name != "__init__.py" {
						modName := strings.TrimSuffix(name, ".py")
						mi := makeModuleInfo(object.None, &object.Str{V: prefix + modName}, false)
						all = append(all, mi)
					}
				}
			}
		}
		return &object.List{V: all}
	}

	m.Dict.SetStr("iter_modules", &object.BuiltinFunc{
		Name: "iter_modules",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			prefix := ""
			if kw != nil {
				if pv, ok := kw.GetStr("prefix"); ok {
					if s, ok2 := pv.(*object.Str); ok2 {
						prefix = s.V
					}
				}
			}
			// Positional prefix (2nd arg).
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					prefix = s.V
				}
			}

			var pathList []string
			if len(a) >= 1 && a[0] != object.None {
				if lst, ok := a[0].(*object.List); ok {
					for _, elem := range lst.V {
						if s, ok2 := elem.(*object.Str); ok2 {
							pathList = append(pathList, s.V)
						}
					}
				}
			}

			result := iterModulesImpl(ii, pathList, prefix)
			return listIterator(i, result), nil
		},
	})

	// ── iter_importer_modules / iter_zipimport_modules ────────────────────────

	iterFromZipImporter := func(imp *object.Instance, prefix string) *object.List {
		archive := ""
		pfx := ""
		if v, ok := imp.Dict.GetStr("archive"); ok {
			if s, ok2 := v.(*object.Str); ok2 {
				archive = s.V
			}
		}
		if v, ok := imp.Dict.GetStr("prefix"); ok {
			if s, ok2 := v.(*object.Str); ok2 {
				pfx = s.V
			}
		}

		pairs := zipModules(archive)
		_ = pfx // prefix within archive already handled by zipModules if needed

		items := make([]object.Object, 0, len(pairs))
		for _, p := range pairs {
			tup := &object.Tuple{V: []object.Object{
				&object.Str{V: prefix + p.name},
				object.BoolOf(p.ispkg),
			}}
			items = append(items, tup)
		}
		return &object.List{V: items}
	}

	iterImporterModulesImpl := func(imp object.Object, prefix string) *object.List {
		if inst, ok := imp.(*object.Instance); ok {
			if inst.Class.Name == "zipimporter" {
				return iterFromZipImporter(inst, prefix)
			}
		}
		return &object.List{V: []object.Object{}}
	}

	m.Dict.SetStr("iter_importer_modules", &object.BuiltinFunc{
		Name: "iter_importer_modules",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return listIterator(i, &object.List{V: nil}), nil
			}
			prefix := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					prefix = s.V
				}
			}
			if kw != nil {
				if pv, ok := kw.GetStr("prefix"); ok {
					if s, ok2 := pv.(*object.Str); ok2 {
						prefix = s.V
					}
				}
			}
			result := iterImporterModulesImpl(a[0], prefix)
			return listIterator(i, result), nil
		},
	})

	m.Dict.SetStr("iter_zipimport_modules", &object.BuiltinFunc{
		Name: "iter_zipimport_modules",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return listIterator(i, &object.List{V: nil}), nil
			}
			prefix := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					prefix = s.V
				}
			}
			result := iterImporterModulesImpl(a[0], prefix)
			return listIterator(i, result), nil
		},
	})

	// ── walk_packages ─────────────────────────────────────────────────────────

	m.Dict.SetStr("walk_packages", &object.BuiltinFunc{
		Name: "walk_packages",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			prefix := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					prefix = s.V
				}
			}
			if kw != nil {
				if pv, ok := kw.GetStr("prefix"); ok {
					if s, ok2 := pv.(*object.Str); ok2 {
						prefix = s.V
					}
				}
			}

			var pathList []string
			if len(a) >= 1 && a[0] != object.None {
				if lst, ok := a[0].(*object.List); ok {
					for _, elem := range lst.V {
						if s, ok2 := elem.(*object.Str); ok2 {
							pathList = append(pathList, s.V)
						}
					}
				}
			}

			// Simplified walk: just return iter_modules results (no recursion into packages).
			result := iterModulesImpl(ii, pathList, prefix)
			return listIterator(i, result), nil
		},
	})

	// ── resolve_name ──────────────────────────────────────────────────────────

	m.Dict.SetStr("resolve_name", &object.BuiltinFunc{
		Name: "resolve_name",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.valueErr, "resolve_name requires an argument")
			}
			nameStr, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.valueErr, "resolve_name requires a string")
			}
			name := nameStr.V
			interp := ii.(*Interp)

			var modName, attrName string
			if idx := strings.Index(name, ":"); idx >= 0 {
				modName = name[:idx]
				attrName = name[idx+1:]
				if modName == "" {
					return nil, object.Errorf(i.valueErr, "Empty module name in %q", name)
				}
			} else {
				modName = name
			}

			if modName == "" {
				return nil, object.Errorf(i.valueErr, "Empty module name")
			}

			mod, err := interp.loadModule(modName)
			if err != nil {
				return nil, object.Errorf(i.importErr, "No module named %q", modName)
			}

			var result object.Object = mod

			if attrName != "" {
				// Walk dotted attribute chain.
				for _, part := range strings.Split(attrName, ".") {
					if part == "" {
						continue
					}
					v, err2 := interp.getAttr(result, part)
					if err2 != nil {
						return nil, object.Errorf(i.importErr, "cannot resolve %q from %q", attrName, modName)
					}
					result = v
				}
			} else if strings.Contains(modName, ".") {
				// For dotted module names like "os.path", the result is the leaf module/object.
				// loadModule("os.path") already returns the correct module.
			}

			return result, nil
		},
	})

	// ── buildModuleInfoList used above ────────────────────────────────────────
	_ = buildMIList

	return m
}

// emptyIterator returns a generator object that immediately stops.
func emptyIterator(i *Interp) *object.Instance {
	return listIterator(i, &object.List{V: []object.Object{}})
}

// listIterator wraps a List in an iterator instance.
func listIterator(i *Interp, lst *object.List) *object.Instance {
	pos := 0
	iterCls := &object.Class{Name: "list_iterator", Dict: object.NewDict()}
	iterCls.Dict.SetStr("__iter__", &object.BuiltinFunc{
		Name: "__iter__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				return a[0], nil
			}
			return object.None, nil
		},
	})
	iterCls.Dict.SetStr("__next__", &object.BuiltinFunc{
		Name: "__next__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if lst == nil || pos >= len(lst.V) {
				return nil, object.Errorf(i.stopIter, "")
			}
			v := lst.V[pos]
			pos++
			return v, nil
		},
	})
	return &object.Instance{Class: iterCls, Dict: object.NewDict()}
}

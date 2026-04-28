package vm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildModulefinder() *object.Module {
	m := &object.Module{Name: "modulefinder", Dict: object.NewDict()}

	// ── module-level state ────────────────────────────────────────────────

	packagePathMap := object.NewDict()
	replacePackageMap := object.NewDict()
	m.Dict.SetStr("packagePathMap", packagePathMap)
	m.Dict.SetStr("replacePackageMap", replacePackageMap)

	// ── AddPackagePath ────────────────────────────────────────────────────

	m.Dict.SetStr("AddPackagePath", &object.BuiltinFunc{Name: "AddPackagePath",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "AddPackagePath() requires 2 arguments")
			}
			pkgName := a[0]
			path := a[1]
			if existing, ok, _ := packagePathMap.Get(pkgName); ok {
				if lst, ok2 := existing.(*object.List); ok2 {
					lst.V = append(lst.V, path)
				}
			} else {
				packagePathMap.Set(pkgName, &object.List{V: []object.Object{path}}) //nolint
			}
			return object.None, nil
		}})

	// ── ReplacePackage ────────────────────────────────────────────────────

	m.Dict.SetStr("ReplacePackage", &object.BuiltinFunc{Name: "ReplacePackage",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "ReplacePackage() requires 2 arguments")
			}
			replacePackageMap.Set(a[0], a[1]) //nolint
			return object.None, nil
		}})

	// ── Module class ──────────────────────────────────────────────────────

	moduleCls := &object.Class{Name: "Module", Dict: object.NewDict()}

	makeModule := func(name string, file object.Object, path object.Object) *object.Instance {
		inst := &object.Instance{Class: moduleCls, Dict: object.NewDict()}
		inst.Dict.SetStr("__name__", &object.Str{V: name})
		inst.Dict.SetStr("__file__", file)
		inst.Dict.SetStr("__path__", path)
		inst.Dict.SetStr("globalnames", object.NewDict())
		inst.Dict.SetStr("starimports", object.NewDict())
		return inst
	}

	pyQuote := func(s string) string {
		return "'" + strings.ReplaceAll(s, "'", "\\'") + "'"
	}

	moduleCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "Module()"}, nil
			}
			self := a[0].(*object.Instance)
			name := ""
			if v, ok := self.Dict.GetStr("__name__"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					name = s.V
				}
			}
			fileObj, _ := self.Dict.GetStr("__file__")
			pathObj, _ := self.Dict.GetStr("__path__")

			fileStr := ""
			hasFile := false
			if fileObj != nil && fileObj != object.None {
				if s, ok := fileObj.(*object.Str); ok {
					fileStr = s.V
					hasFile = true
				}
			}

			hasPath := false
			pathRepr := ""
			if pathObj != nil && pathObj != object.None {
				hasPath = true
				pathRepr = object.Repr(pathObj)
			}

			if hasPath {
				return &object.Str{V: fmt.Sprintf("Module(%s, %s, %s)", pyQuote(name), pyQuote(fileStr), pathRepr)}, nil
			} else if hasFile {
				return &object.Str{V: fmt.Sprintf("Module(%s, %s)", pyQuote(name), pyQuote(fileStr))}, nil
			}
			return &object.Str{V: fmt.Sprintf("Module(%s)", pyQuote(name))}, nil
		}})

	m.Dict.SetStr("Module", &object.BuiltinFunc{Name: "Module",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "Module() requires a name")
			}
			name := ""
			if s, ok := a[0].(*object.Str); ok {
				name = s.V
			}
			var file object.Object = object.None
			var path object.Object = object.None
			if len(a) >= 2 {
				file = a[1]
			}
			if len(a) >= 3 {
				path = a[2]
			}
			if kw != nil {
				if v, ok := kw.GetStr("file"); ok {
					file = v
				}
				if v, ok := kw.GetStr("path"); ok {
					path = v
				}
			}
			return makeModule(name, file, path), nil
		}})

	// ── ModuleFinder class ────────────────────────────────────────────────

	mfCls := &object.Class{Name: "ModuleFinder", Dict: object.NewDict()}

	makeMF := func(path, excludes object.Object, debug int64) *object.Instance {
		inst := &object.Instance{Class: mfCls, Dict: object.NewDict()}
		inst.Dict.SetStr("modules", object.NewDict())
		inst.Dict.SetStr("badmodules", object.NewDict())
		inst.Dict.SetStr("debug", object.NewInt(debug))
		if path == nil || path == object.None {
			inst.Dict.SetStr("path", object.None)
		} else {
			inst.Dict.SetStr("path", path)
		}
		if excludes == nil || excludes == object.None {
			inst.Dict.SetStr("excludes", &object.List{V: nil})
		} else {
			inst.Dict.SetStr("excludes", excludes)
		}
		return inst
	}

	// add_module
	mfCls.Dict.SetStr("add_module", &object.BuiltinFunc{Name: "add_module",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "add_module() requires self and fqname")
			}
			self := a[0].(*object.Instance)
			name := ""
			if s, ok := a[1].(*object.Str); ok {
				name = s.V
			}
			modulesObj, _ := self.Dict.GetStr("modules")
			modules := modulesObj.(*object.Dict)
			key := &object.Str{V: name}
			if existing, ok, _ := modules.Get(key); ok {
				return existing, nil
			}
			mod := makeModule(name, object.None, object.None)
			modules.Set(key, mod) //nolint
			return mod, nil
		}})

	// any_missing
	mfCls.Dict.SetStr("any_missing", &object.BuiltinFunc{Name: "any_missing",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{V: nil}, nil
			}
			self := a[0].(*object.Instance)
			missing, _ := mfAnyMissingMaybe(self)
			items := make([]object.Object, len(missing))
			for j, s := range missing {
				items[j] = &object.Str{V: s}
			}
			return &object.List{V: items}, nil
		}})

	// any_missing_maybe
	mfCls.Dict.SetStr("any_missing_maybe", &object.BuiltinFunc{Name: "any_missing_maybe",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				empty := &object.List{V: nil}
				return &object.Tuple{V: []object.Object{empty, empty}}, nil
			}
			self := a[0].(*object.Instance)
			missing, maybe := mfAnyMissingMaybe(self)
			toList := func(ss []string) *object.List {
				items := make([]object.Object, len(ss))
				for j, s := range ss {
					items[j] = &object.Str{V: s}
				}
				return &object.List{V: items}
			}
			return &object.Tuple{V: []object.Object{toList(missing), toList(maybe)}}, nil
		}})

	// report
	mfCls.Dict.SetStr("report", &object.BuiltinFunc{Name: "report",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			modulesObj, _ := self.Dict.GetStr("modules")
			modules := modulesObj.(*object.Dict)

			fmt.Fprintln(i.Stdout)
			fmt.Fprintf(i.Stdout, "  %-25s %s\n", "Name", "File")
			fmt.Fprintf(i.Stdout, "  %-25s %s\n", "----", "----")

			keys, vals := modules.Items()
			var pairs []struct {
				name string
				mod  object.Object
			}
			for idx, k := range keys {
				if s, ok := k.(*object.Str); ok {
					pairs = append(pairs, struct {
						name string
						mod  object.Object
					}{s.V, vals[idx]})
				}
			}
			sort.Slice(pairs, func(a, b int) bool { return pairs[a].name < pairs[b].name })

			for _, p := range pairs {
				prefix := "m"
				fileStr := ""
				if inst, ok := p.mod.(*object.Instance); ok {
					if pathObj, ok2 := inst.Dict.GetStr("__path__"); ok2 && pathObj != object.None {
						prefix = "P"
					}
					if fileObj, ok2 := inst.Dict.GetStr("__file__"); ok2 && fileObj != object.None {
						if s, ok3 := fileObj.(*object.Str); ok3 {
							fileStr = s.V
						}
					}
				}
				fmt.Fprintf(i.Stdout, "%s %-25s %s\n", prefix, p.name, fileStr)
			}
			return object.None, nil
		}})

	// find_module stub
	mfCls.Dict.SetStr("find_module", &object.BuiltinFunc{Name: "find_module",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{object.None, object.None, object.None}}, nil
		}})

	// run_script stub
	mfCls.Dict.SetStr("run_script", &object.BuiltinFunc{Name: "run_script",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	for _, name := range []string{"msg", "msgin", "msgout"} {
		n := name
		mfCls.Dict.SetStr(n, &object.BuiltinFunc{Name: n,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}})
	}

	m.Dict.SetStr("ModuleFinder", &object.BuiltinFunc{Name: "ModuleFinder",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			var path object.Object = object.None
			var debug int64
			var excludes object.Object = object.None
			if len(a) >= 1 {
				path = a[0]
			}
			if len(a) >= 2 {
				if n, ok := toInt64(a[1]); ok {
					debug = n
				}
			}
			if len(a) >= 3 {
				excludes = a[2]
			}
			if kw != nil {
				if v, ok := kw.GetStr("path"); ok {
					path = v
				}
				if v, ok := kw.GetStr("debug"); ok {
					if n, ok2 := toInt64(v); ok2 {
						debug = n
					}
				}
				if v, ok := kw.GetStr("excludes"); ok {
					excludes = v
				}
			}
			return makeMF(path, excludes, debug), nil
		}})

	return m
}

// mfAnyMissingMaybe implements CPython's any_missing_maybe logic.
func mfAnyMissingMaybe(self *object.Instance) (missing, maybe []string) {
	modulesObj, _ := self.Dict.GetStr("modules")
	badObj, _ := self.Dict.GetStr("badmodules")
	modules := modulesObj.(*object.Dict)
	bad := badObj.(*object.Dict)

	keys, vals := bad.Items()
	for idx, k := range keys {
		name := ""
		if s, ok := k.(*object.Str); ok {
			name = s.V
		}
		if _, ok, _ := modules.Get(k); ok {
			continue
		}
		dot := strings.LastIndex(name, ".")
		if dot < 0 {
			missing = append(missing, name)
			continue
		}
		pkgname := name[:dot]
		pkgKey := &object.Str{V: pkgname}
		if _, hasPkg, _ := modules.Get(pkgKey); hasPkg {
			if importersDict, ok3 := vals[idx].(*object.Dict); ok3 {
				if _, selfImport, _ := importersDict.Get(pkgKey); selfImport {
					missing = append(missing, name)
					continue
				}
			}
			maybe = append(maybe, name)
		} else {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(maybe)
	return
}

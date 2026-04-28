package vm

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tamnd/goipy/object"
)

// ── importlib.metadata ─────────────────────────────────────────────────────────

func (i *Interp) buildImportlibMetadata() *object.Module {
	m := &object.Module{Name: "importlib.metadata", Dict: object.NewDict()}

	// ── PackageNotFoundError ───────────────────────────────────────────────────

	pnfeCls := &object.Class{
		Name:  "PackageNotFoundError",
		Dict:  object.NewDict(),
		Bases: []*object.Class{i.importErr},
	}
	m.Dict.SetStr("PackageNotFoundError", pnfeCls)

	raiseNotFound := func(name string) error {
		return object.Errorf(pnfeCls, "No package metadata was found for %q", name)
	}

	// ── findDistInfoPath ───────────────────────────────────────────────────────

	// findDistInfoPath looks for a <name>-<version>.dist-info dir in i.SearchPath
	// and then the common site-packages locations.
	findDistInfoPath := func(name string) string {
		normalized := strings.ReplaceAll(strings.ToLower(name), "-", "_")
		searchDirs := append([]string(nil), i.SearchPath...)
		searchDirs = append(searchDirs,
			"/opt/homebrew/lib/python3.14/site-packages",
			"/usr/lib/python3.14/site-packages",
			"/usr/local/lib/python3.14/site-packages",
		)
		for _, base := range searchDirs {
			entries, err := os.ReadDir(base)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() || !strings.HasSuffix(e.Name(), ".dist-info") {
					continue
				}
				distName := strings.TrimSuffix(e.Name(), ".dist-info")
				parts := strings.SplitN(distName, "-", 2)
				if len(parts) < 1 {
					continue
				}
				distNorm := strings.ReplaceAll(strings.ToLower(parts[0]), "-", "_")
				if distNorm == normalized {
					return filepath.Join(base, e.Name())
				}
			}
		}
		return ""
	}

	// ── PackageMetadata ────────────────────────────────────────────────────────

	pmCls := &object.Class{Name: "PackageMetadata", Dict: object.NewDict()}

	makePackageMetadata := func(headers map[string][]string, order []string) *object.Instance {
		inst := &object.Instance{Class: pmCls, Dict: object.NewDict()}

		stripSelf := func(a []object.Object) []object.Object {
			if len(a) > 0 {
				if _, ok := a[0].(*object.Instance); ok {
					return a[1:]
				}
			}
			return a
		}

		inst.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = stripSelf(a)
				if len(a) == 0 {
					return object.None, nil
				}
				key := ""
				if s, ok := a[0].(*object.Str); ok {
					key = s.V
				}
				if vs, ok := headers[key]; ok && len(vs) > 0 {
					return &object.Str{V: vs[0]}, nil
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = stripSelf(a)
				if len(a) == 0 {
					return object.False, nil
				}
				key := ""
				if s, ok := a[0].(*object.Str); ok {
					key = s.V
				}
				_, ok := headers[key]
				return object.BoolOf(ok), nil
			}})

		inst.Dict.SetStr("get", &object.BuiltinFunc{Name: "get",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = stripSelf(a)
				if len(a) == 0 {
					return object.None, nil
				}
				key := ""
				if s, ok := a[0].(*object.Str); ok {
					key = s.V
				}
				if vs, ok := headers[key]; ok && len(vs) > 0 {
					return &object.Str{V: vs[0]}, nil
				}
				if len(a) >= 2 {
					return a[1], nil
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("get_all", &object.BuiltinFunc{Name: "get_all",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = stripSelf(a)
				if len(a) == 0 {
					return object.None, nil
				}
				key := ""
				if s, ok := a[0].(*object.Str); ok {
					key = s.V
				}
				vs, ok := headers[key]
				if !ok || len(vs) == 0 {
					if len(a) >= 2 {
						return a[1], nil
					}
					return object.None, nil
				}
				items := make([]object.Object, len(vs))
				for j, v := range vs {
					items[j] = &object.Str{V: v}
				}
				return &object.List{V: items}, nil
			}})

		inst.Dict.SetStr("keys", &object.BuiltinFunc{Name: "keys",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				seen := map[string]bool{}
				var keys []object.Object
				for _, k := range order {
					if !seen[k] {
						seen[k] = true
						keys = append(keys, &object.Str{V: k})
					}
				}
				return &object.List{V: keys}, nil
			}})

		inst.Dict.SetStr("items", &object.BuiltinFunc{Name: "items",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				var pairs []object.Object
				for _, k := range order {
					for _, v := range headers[k] {
						pairs = append(pairs, &object.Tuple{V: []object.Object{
							&object.Str{V: k},
							&object.Str{V: v},
						}})
					}
				}
				return &object.List{V: pairs}, nil
			}})

		return inst
	}

	// ── PackagePath ────────────────────────────────────────────────────────────

	ppCls := &object.Class{Name: "PackagePath", Dict: object.NewDict()}

	makePackagePath := func(path string) *object.Instance {
		inst := &object.Instance{Class: ppCls, Dict: object.NewDict()}
		inst.Dict.SetStr("_path", &object.Str{V: path})
		inst.Dict.SetStr("name", &object.Str{V: filepath.Base(path)})
		inst.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.Str{V: path}, nil
			}})
		inst.Dict.SetStr("read_text", &object.BuiltinFunc{Name: "read_text",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				data, err := os.ReadFile(path)
				if err != nil {
					return nil, object.Errorf(i.fileNotFoundErr, "%v", err)
				}
				return &object.Str{V: string(data)}, nil
			}})
		inst.Dict.SetStr("read_bytes", &object.BuiltinFunc{Name: "read_bytes",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				data, err := os.ReadFile(path)
				if err != nil {
					return nil, object.Errorf(i.fileNotFoundErr, "%v", err)
				}
				return &object.Bytes{V: data}, nil
			}})
		return inst
	}

	// ── EntryPoint ─────────────────────────────────────────────────────────────

	epCls := &object.Class{Name: "EntryPoint", Dict: object.NewDict()}

	makeEntryPoint := func(name, group, value string) *object.Instance {
		inst := &object.Instance{Class: epCls, Dict: object.NewDict()}
		inst.Dict.SetStr("name", &object.Str{V: name})
		inst.Dict.SetStr("group", &object.Str{V: group})
		inst.Dict.SetStr("value", &object.Str{V: value})
		inst.Dict.SetStr("load", &object.BuiltinFunc{Name: "load",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}})
		return inst
	}

	// EntryPoint(name=..., group=..., value=...)
	epCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			getStr := func(posIdx int, key string) string {
				if kw != nil {
					if v, ok2 := kw.GetStr(key); ok2 {
						if s, ok3 := v.(*object.Str); ok3 {
							return s.V
						}
					}
				}
				if len(a) > posIdx {
					if s, ok2 := a[posIdx].(*object.Str); ok2 {
						return s.V
					}
				}
				return ""
			}
			name := getStr(1, "name")
			group := getStr(2, "group")
			value := getStr(3, "value")
			inst.Dict.SetStr("name", &object.Str{V: name})
			inst.Dict.SetStr("group", &object.Str{V: group})
			inst.Dict.SetStr("value", &object.Str{V: value})
			inst.Dict.SetStr("load", &object.BuiltinFunc{Name: "load",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return object.None, nil
				}})
			return object.None, nil
		}})

	// ── EntryPoints ─────────────────────────────────────────────────────────────

	epsCls := &object.Class{Name: "EntryPoints", Dict: object.NewDict()}

	// setupEPInst populates an EntryPoints instance with its list-like methods.
	var setupEPInst func(inst *object.Instance, objs []object.Object)
	setupEPInst = func(inst *object.Instance, objs []object.Object) {
		inst.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.NewInt(int64(len(objs))), nil
			}})

		inst.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				args := a
				if len(args) > 0 {
					if _, ok := args[0].(*object.Instance); ok {
						args = args[1:]
					}
				}
				if len(args) == 0 {
					return nil, object.Errorf(i.indexErr, "index out of range")
				}
				if idx, ok := toInt64(args[0]); ok {
					n := int(idx)
					if n < 0 {
						n += len(objs)
					}
					if n < 0 || n >= len(objs) {
						return nil, object.Errorf(i.indexErr, "list index out of range")
					}
					return objs[n], nil
				}
				return nil, object.Errorf(i.typeErr, "indices must be integers")
			}})

		inst.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				idx := 0
				return &object.Iter{Next: func() (object.Object, bool, error) {
					if idx >= len(objs) {
						return nil, false, nil
					}
					v := objs[idx]
					idx++
					return v, true, nil
				}}, nil
			}})

		inst.Dict.SetStr("select", &object.BuiltinFunc{Name: "select",
			Call: func(_ any, _ []object.Object, kw *object.Dict) (object.Object, error) {
				if kw == nil || kw.Len() == 0 {
					newInst := &object.Instance{Class: epsCls, Dict: object.NewDict()}
					setupEPInst(newInst, objs)
					return newInst, nil
				}
				filters := map[string]string{}
				keys, vals := kw.Items()
				for idx, k := range keys {
					if ks, ok := k.(*object.Str); ok {
						if vs, ok2 := vals[idx].(*object.Str); ok2 {
							filters[ks.V] = vs.V
						}
					}
				}
				var filtered []object.Object
				for _, obj := range objs {
					ep, ok := obj.(*object.Instance)
					if !ok {
						continue
					}
					match := true
					for attr, want := range filters {
						v, ok2 := ep.Dict.GetStr(attr)
						if !ok2 {
							match = false
							break
						}
						if s, ok3 := v.(*object.Str); !ok3 || s.V != want {
							match = false
							break
						}
					}
					if match {
						filtered = append(filtered, ep)
					}
				}
				newInst := &object.Instance{Class: epsCls, Dict: object.NewDict()}
				setupEPInst(newInst, filtered)
				return newInst, nil
			}})
	}

	makeEntryPoints := func(items []*object.Instance) *object.Instance {
		inst := &object.Instance{Class: epsCls, Dict: object.NewDict()}
		objs := make([]object.Object, len(items))
		for j, ep := range items {
			objs[j] = ep
		}
		setupEPInst(inst, objs)
		return inst
	}

	// EntryPoints(iterable) constructor
	epsCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interp.(*Interp)
			if len(a) < 2 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			items, err := iterate(ii, a[1])
			if err != nil {
				return nil, err
			}
			setupEPInst(inst, items)
			return object.None, nil
		}})

	// ── PathDistribution ───────────────────────────────────────────────────────

	distCls := &object.Class{Name: "Distribution", Dict: object.NewDict()}
	pathDistCls := &object.Class{
		Name:  "PathDistribution",
		Dict:  object.NewDict(),
		Bases: []*object.Class{distCls},
	}

	makePathDist := func(distInfoDir string) *object.Instance {
		inst := &object.Instance{Class: pathDistCls, Dict: object.NewDict()}
		inst.Dict.SetStr("_dist_info", &object.Str{V: distInfoDir})

		headers, order := parseRFC822Headers(filepath.Join(distInfoDir, "METADATA"))
		if headers == nil {
			headers, order = parseRFC822Headers(filepath.Join(distInfoDir, "PKG-INFO"))
		}
		if headers == nil {
			headers = map[string][]string{}
		}

		// name — prefer METADATA header, fall back to dir name
		nameStr := ""
		if hs, ok := headers["Name"]; ok && len(hs) > 0 {
			nameStr = hs[0]
		}
		if nameStr == "" {
			base := filepath.Base(distInfoDir)
			parts := strings.SplitN(strings.TrimSuffix(base, ".dist-info"), "-", 2)
			nameStr = parts[0]
		}
		inst.Dict.SetStr("name", &object.Str{V: nameStr})

		// version
		verStr := ""
		if hs, ok := headers["Version"]; ok && len(hs) > 0 {
			verStr = hs[0]
		}
		if verStr == "" {
			base := filepath.Base(distInfoDir)
			parts := strings.SplitN(strings.TrimSuffix(base, ".dist-info"), "-", 2)
			if len(parts) >= 2 {
				verStr = parts[1]
			}
		}
		inst.Dict.SetStr("version", &object.Str{V: verStr})

		// metadata
		inst.Dict.SetStr("metadata", makePackageMetadata(headers, order))

		// requires → list[str] from Requires-Dist, or None
		if reqs, ok := headers["Requires-Dist"]; ok && len(reqs) > 0 {
			items := make([]object.Object, len(reqs))
			for j, r := range reqs {
				items[j] = &object.Str{V: r}
			}
			inst.Dict.SetStr("requires", &object.List{V: items})
		} else {
			inst.Dict.SetStr("requires", object.None)
		}

		// read_text(filename) → str | None
		inst.Dict.SetStr("read_text", &object.BuiltinFunc{Name: "read_text",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				args := a
				if len(args) > 0 {
					if _, ok := args[0].(*object.Instance); ok {
						args = args[1:]
					}
				}
				if len(args) == 0 {
					return object.None, nil
				}
				filename := ""
				if s, ok := args[0].(*object.Str); ok {
					filename = s.V
				}
				data, err := os.ReadFile(filepath.Join(distInfoDir, filename))
				if err != nil {
					return object.None, nil
				}
				return &object.Str{V: string(data)}, nil
			}})

		// files — from RECORD, or None
		recordData, err := os.ReadFile(filepath.Join(distInfoDir, "RECORD"))
		var filesList []object.Object
		if err == nil {
			for _, line := range strings.Split(string(recordData), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				parts := strings.SplitN(line, ",", 2)
				path := parts[0]
				if path != "" {
					filesList = append(filesList, makePackagePath(path))
				}
			}
		}
		if len(filesList) > 0 {
			inst.Dict.SetStr("files", &object.List{V: filesList})
		} else {
			inst.Dict.SetStr("files", object.None)
		}

		// entry_points — from entry_points.txt
		var eps []*object.Instance
		epData, err2 := os.ReadFile(filepath.Join(distInfoDir, "entry_points.txt"))
		if err2 == nil {
			eps = parseEntryPointsTxt(string(epData), makeEntryPoint)
		}
		inst.Dict.SetStr("entry_points", makeEntryPoints(eps))

		// locate_file(path)
		parentDir := filepath.Dir(distInfoDir)
		inst.Dict.SetStr("locate_file", &object.BuiltinFunc{Name: "locate_file",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				args := a
				if len(args) > 0 {
					if _, ok := args[0].(*object.Instance); ok {
						args = args[1:]
					}
				}
				if len(args) == 0 {
					return &object.Str{V: parentDir}, nil
				}
				if s, ok := args[0].(*object.Str); ok {
					return &object.Str{V: filepath.Join(parentDir, s.V)}, nil
				}
				return &object.Str{V: parentDir}, nil
			}})

		return inst
	}

	// Distribution.from_name(name) classmethod
	distCls.Dict.SetStr("from_name", &object.BuiltinFunc{Name: "from_name",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// strip class/instance self
			args := a
			if len(args) > 0 {
				switch args[0].(type) {
				case *object.Class, *object.Instance:
					args = args[1:]
				}
			}
			if len(args) == 0 {
				return nil, raiseNotFound("")
			}
			name := ""
			if s, ok := args[0].(*object.Str); ok {
				name = s.V
			}
			p := findDistInfoPath(name)
			if p == "" {
				return nil, raiseNotFound(name)
			}
			return makePathDist(p), nil
		}})

	// ── searchDistInfoDirs helper ──────────────────────────────────────────────

	allDistInfoPaths := func() []string {
		var result []string
		seen := map[string]bool{}
		searchDirs := append([]string(nil), i.SearchPath...)
		searchDirs = append(searchDirs,
			"/opt/homebrew/lib/python3.14/site-packages",
			"/usr/lib/python3.14/site-packages",
			"/usr/local/lib/python3.14/site-packages",
		)
		for _, base := range searchDirs {
			entries, err := os.ReadDir(base)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() || !strings.HasSuffix(e.Name(), ".dist-info") {
					continue
				}
				fullPath := filepath.Join(base, e.Name())
				if !seen[fullPath] {
					seen[fullPath] = true
					result = append(result, fullPath)
				}
			}
		}
		return result
	}

	// ── Module-level functions ─────────────────────────────────────────────────

	m.Dict.SetStr("distribution", &object.BuiltinFunc{Name: "distribution",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			name := ""
			if len(a) > 0 {
				if s, ok := a[0].(*object.Str); ok {
					name = s.V
				}
			}
			p := findDistInfoPath(name)
			if p == "" {
				return nil, raiseNotFound(name)
			}
			return makePathDist(p), nil
		}})

	m.Dict.SetStr("version", &object.BuiltinFunc{Name: "version",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			name := ""
			if len(a) > 0 {
				if s, ok := a[0].(*object.Str); ok {
					name = s.V
				}
			}
			p := findDistInfoPath(name)
			if p == "" {
				return nil, raiseNotFound(name)
			}
			d := makePathDist(p)
			if v, ok := d.Dict.GetStr("version"); ok {
				return v, nil
			}
			return nil, raiseNotFound(name)
		}})

	m.Dict.SetStr("metadata", &object.BuiltinFunc{Name: "metadata",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			name := ""
			if len(a) > 0 {
				if s, ok := a[0].(*object.Str); ok {
					name = s.V
				}
			}
			p := findDistInfoPath(name)
			if p == "" {
				return nil, raiseNotFound(name)
			}
			d := makePathDist(p)
			if meta, ok := d.Dict.GetStr("metadata"); ok {
				return meta, nil
			}
			return nil, raiseNotFound(name)
		}})

	m.Dict.SetStr("requires", &object.BuiltinFunc{Name: "requires",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			name := ""
			if len(a) > 0 {
				if s, ok := a[0].(*object.Str); ok {
					name = s.V
				}
			}
			p := findDistInfoPath(name)
			if p == "" {
				return nil, raiseNotFound(name)
			}
			d := makePathDist(p)
			if req, ok := d.Dict.GetStr("requires"); ok {
				return req, nil
			}
			return object.None, nil
		}})

	m.Dict.SetStr("files", &object.BuiltinFunc{Name: "files",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			name := ""
			if len(a) > 0 {
				if s, ok := a[0].(*object.Str); ok {
					name = s.V
				}
			}
			p := findDistInfoPath(name)
			if p == "" {
				return nil, raiseNotFound(name)
			}
			d := makePathDist(p)
			if files, ok := d.Dict.GetStr("files"); ok {
				return files, nil
			}
			return object.None, nil
		}})

	m.Dict.SetStr("distributions", &object.BuiltinFunc{Name: "distributions",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			paths := allDistInfoPaths()
			dists := make([]object.Object, len(paths))
			for j, p := range paths {
				dists[j] = makePathDist(p)
			}
			idx := 0
			return &object.Iter{Next: func() (object.Object, bool, error) {
				if idx >= len(dists) {
					return nil, false, nil
				}
				v := dists[idx]
				idx++
				return v, true, nil
			}}, nil
		}})

	m.Dict.SetStr("packages_distributions", &object.BuiltinFunc{Name: "packages_distributions",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			result := object.NewDict()
			for _, distInfoDir := range allDistInfoPaths() {
				base := filepath.Base(distInfoDir)
				distName := strings.TrimSuffix(base, ".dist-info")
				parts := strings.SplitN(distName, "-", 2)
				pkgName := parts[0]

				topLevelData, err := os.ReadFile(filepath.Join(distInfoDir, "top_level.txt"))
				var pkgs []string
				if err == nil {
					for _, line := range strings.Split(string(topLevelData), "\n") {
						line = strings.TrimSpace(line)
						if line != "" {
							pkgs = append(pkgs, line)
						}
					}
				}
				if len(pkgs) == 0 {
					pkgs = []string{pkgName}
				}
				for _, pkg := range pkgs {
					key := &object.Str{V: pkg}
					existing, _, _ := result.Get(key)
					var lst []object.Object
					if el, ok := existing.(*object.List); ok {
						lst = append([]object.Object(nil), el.V...)
					}
					lst = append(lst, &object.Str{V: pkgName})
					_ = result.Set(key, &object.List{V: lst})
				}
			}
			return result, nil
		}})

	m.Dict.SetStr("entry_points", &object.BuiltinFunc{Name: "entry_points",
		Call: func(_ any, _ []object.Object, kw *object.Dict) (object.Object, error) {
			var allEps []*object.Instance
			for _, distInfoDir := range allDistInfoPaths() {
				data, err := os.ReadFile(filepath.Join(distInfoDir, "entry_points.txt"))
				if err != nil {
					continue
				}
				allEps = append(allEps, parseEntryPointsTxt(string(data), makeEntryPoint)...)
			}
			epsInst := makeEntryPoints(allEps)
			if kw != nil && kw.Len() > 0 {
				if selectFn, ok := epsInst.Dict.GetStr("select"); ok {
					return i.callObject(selectFn, nil, kw)
				}
			}
			return epsInst, nil
		}})

	// ── Expose classes ─────────────────────────────────────────────────────────

	m.Dict.SetStr("Distribution", distCls)
	m.Dict.SetStr("PathDistribution", pathDistCls)
	m.Dict.SetStr("EntryPoint", epCls)
	m.Dict.SetStr("EntryPoints", epsCls)
	m.Dict.SetStr("PackagePath", ppCls)
	m.Dict.SetStr("PackageMetadata", pmCls)

	return m
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// parseRFC822Headers reads and parses an RFC 2822-style METADATA file from disk.
// Returns (headers, ordered-key-list) or (nil, nil) on read error.
func parseRFC822Headers(path string) (map[string][]string, []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	headers := map[string][]string{}
	var order []string
	currentKey := ""
	currentVal := ""

	flush := func() {
		if currentKey == "" {
			return
		}
		if _, exists := headers[currentKey]; !exists {
			order = append(order, currentKey)
		}
		headers[currentKey] = append(headers[currentKey], currentVal)
		currentKey = ""
		currentVal = ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			break
		}
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			currentVal += " " + strings.TrimSpace(line)
			continue
		}
		flush()
		if idx := strings.IndexByte(line, ':'); idx >= 0 {
			currentKey = strings.TrimSpace(line[:idx])
			currentVal = strings.TrimSpace(line[idx+1:])
		}
	}
	flush()

	return headers, order
}

// parseEntryPointsTxt parses ini-style entry_points.txt into EntryPoint instances.
func parseEntryPointsTxt(text string, makeEP func(name, group, value string) *object.Instance) []*object.Instance {
	var result []*object.Instance
	currentGroup := ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentGroup = line[1 : len(line)-1]
			continue
		}
		if idx := strings.IndexByte(line, '='); idx >= 0 && currentGroup != "" {
			name := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			result = append(result, makeEP(name, currentGroup, value))
		}
	}
	return result
}

package vm

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildSite() *object.Module {
	m := &object.Module{Name: "site", Dict: object.NewDict()}

	// Platform-appropriate paths
	prefix := "/usr"
	var userBase, userSite string
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		userBase = filepath.Join(home, "Library", "Python", "3.14")
		userSite = filepath.Join(userBase, "lib", "python", "site-packages")
	} else if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			userBase = filepath.Join(appData, "Python", "Python314")
		} else {
			userBase = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "Python", "Python314")
		}
		userSite = filepath.Join(userBase, "site-packages")
	} else {
		home, _ := os.UserHomeDir()
		userBase = filepath.Join(home, ".local")
		userSite = filepath.Join(userBase, "lib", "python3.14", "site-packages")
	}

	m.Dict.SetStr("ENABLE_USER_SITE", object.True)
	m.Dict.SetStr("PREFIXES", &object.List{V: []object.Object{
		&object.Str{V: prefix},
		&object.Str{V: prefix},
	}})
	m.Dict.SetStr("USER_BASE", &object.Str{V: userBase})
	m.Dict.SetStr("USER_SITE", &object.Str{V: userSite})

	// ── getsitepackages ───────────────────────────────────────────────────────

	m.Dict.SetStr("getsitepackages", &object.BuiltinFunc{
		Name: "getsitepackages",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			prefixList := []string{prefix}
			if len(a) >= 1 {
				if lst, ok := a[0].(*object.List); ok {
					prefixList = nil
					for _, item := range lst.V {
						if s, ok2 := item.(*object.Str); ok2 {
							prefixList = append(prefixList, s.V)
						}
					}
				}
			}
			items := []object.Object{}
			for _, p := range prefixList {
				items = append(items,
					&object.Str{V: filepath.Join(p, "lib", "python3.14", "site-packages")},
					&object.Str{V: filepath.Join(p, "lib64", "python3.14", "site-packages")},
				)
			}
			return &object.List{V: items}, nil
		},
	})

	// ── getusersitepackages / getuserbase ─────────────────────────────────────

	m.Dict.SetStr("getusersitepackages", &object.BuiltinFunc{
		Name: "getusersitepackages",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: userSite}, nil
		},
	})

	m.Dict.SetStr("getuserbase", &object.BuiltinFunc{
		Name: "getuserbase",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: userBase}, nil
		},
	})

	// ── makepath ──────────────────────────────────────────────────────────────

	m.Dict.SetStr("makepath", &object.BuiltinFunc{
		Name: "makepath",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			parts := make([]string, 0, len(a))
			for _, arg := range a {
				if s, ok := arg.(*object.Str); ok {
					parts = append(parts, s.V)
				}
			}
			joined := filepath.Join(parts...)
			s := &object.Str{V: joined}
			return &object.Tuple{V: []object.Object{s, s}}, nil
		},
	})

	// ── gethistoryfile ────────────────────────────────────────────────────────

	m.Dict.SetStr("gethistoryfile", &object.BuiltinFunc{
		Name: "gethistoryfile",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			home, _ := os.UserHomeDir()
			return &object.Str{V: filepath.Join(home, ".python_history")}, nil
		},
	})

	// ── check_enableusersite ──────────────────────────────────────────────────

	m.Dict.SetStr("check_enableusersite", &object.BuiltinFunc{
		Name: "check_enableusersite",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		},
	})

	// ── removeduppaths ────────────────────────────────────────────────────────

	m.Dict.SetStr("removeduppaths", &object.BuiltinFunc{
		Name: "removeduppaths",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewSet(), nil
		},
	})

	// ── addsitepackages / addusersitepackages / venv ──────────────────────────

	returnSet := func(a []object.Object) object.Object {
		if len(a) >= 1 {
			if s, ok := a[0].(*object.Set); ok {
				return s
			}
		}
		return object.NewSet()
	}

	m.Dict.SetStr("addsitepackages", &object.BuiltinFunc{
		Name: "addsitepackages",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return returnSet(a), nil
		},
	})

	m.Dict.SetStr("addusersitepackages", &object.BuiltinFunc{
		Name: "addusersitepackages",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return returnSet(a), nil
		},
	})

	m.Dict.SetStr("venv", &object.BuiltinFunc{
		Name: "venv",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return returnSet(a), nil
		},
	})

	// ── no-op stubs ───────────────────────────────────────────────────────────

	noops := []string{
		"addsitedir", "addpackage", "abs_paths", "enablerlcompleter",
		"execsitecustomize", "execusercustomize", "main", "register_readline",
	}
	for _, name := range noops {
		n := name
		m.Dict.SetStr(n, &object.BuiltinFunc{
			Name: n,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		})
	}

	// ── Quitter class (for setquit) ───────────────────────────────────────────

	quitterClass := &object.Class{Name: "Quitter", Dict: object.NewDict()}
	quitterClass.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			name := "quit"
			eof := "Ctrl-D (i.e. EOF)"
			if len(a) >= 1 {
				if inst, ok := a[0].(*object.Instance); ok {
					if n, ok2 := inst.Dict.GetStr("name"); ok2 {
						if s, ok3 := n.(*object.Str); ok3 {
							name = s.V
						}
					}
					if e, ok2 := inst.Dict.GetStr("eof"); ok2 {
						if s, ok3 := e.(*object.Str); ok3 {
							eof = s.V
						}
					}
				}
			}
			return &object.Str{V: "Use " + name + "() or " + eof + " to exit"}, nil
		},
	})
	quitterClass.Dict.SetStr("__call__", &object.BuiltinFunc{
		Name: "__call__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.systemExit, "")
		},
	})

	makeQuitter := func(name, eof string) *object.Instance {
		inst := &object.Instance{Class: quitterClass, Dict: object.NewDict()}
		inst.Dict.SetStr("name", &object.Str{V: name})
		inst.Dict.SetStr("eof", &object.Str{V: eof})
		return inst
	}

	// ── _Printer class (for setcopyright) ────────────────────────────────────

	printerClass := &object.Class{Name: "_Printer", Dict: object.NewDict()}
	printerClass.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			title := "Copyright"
			if len(a) >= 1 {
				if inst, ok := a[0].(*object.Instance); ok {
					if n, ok2 := inst.Dict.GetStr("__name__"); ok2 {
						if s, ok3 := n.(*object.Str); ok3 {
							title = s.V
						}
					}
				}
			}
			return &object.Str{V: title + " information. Type " + title + "() to see the full license text"}, nil
		},
	})
	printerClass.Dict.SetStr("__call__", &object.BuiltinFunc{
		Name: "__call__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	makePrinter := func(name string) *object.Instance {
		inst := &object.Instance{Class: printerClass, Dict: object.NewDict()}
		inst.Dict.SetStr("__name__", &object.Str{V: name})
		return inst
	}

	// ── _Helper class (for sethelper) ─────────────────────────────────────────

	helperClass := &object.Class{Name: "_Helper", Dict: object.NewDict()}
	helperClass.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "Type help() for interactive help, or help(object) for help about object."}, nil
		},
	})
	helperClass.Dict.SetStr("__call__", &object.BuiltinFunc{
		Name: "__call__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── setquit ───────────────────────────────────────────────────────────────

	m.Dict.SetStr("setquit", &object.BuiltinFunc{
		Name: "setquit",
		Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			interp := ii.(*Interp)
			eof := "Ctrl-D (i.e. EOF)"
			if runtime.GOOS == "windows" {
				eof = "Ctrl-Z plus Return"
			}
			interp.Builtins.SetStr("quit", makeQuitter("quit", eof))
			interp.Builtins.SetStr("exit", makeQuitter("exit", eof))
			return object.None, nil
		},
	})

	// ── setcopyright ──────────────────────────────────────────────────────────

	m.Dict.SetStr("setcopyright", &object.BuiltinFunc{
		Name: "setcopyright",
		Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			interp := ii.(*Interp)
			interp.Builtins.SetStr("copyright", makePrinter("copyright"))
			interp.Builtins.SetStr("credits", makePrinter("credits"))
			interp.Builtins.SetStr("license", makePrinter("license"))
			return object.None, nil
		},
	})

	// ── sethelper ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("sethelper", &object.BuiltinFunc{
		Name: "sethelper",
		Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			interp := ii.(*Interp)
			inst := &object.Instance{Class: helperClass, Dict: object.NewDict()}
			interp.Builtins.SetStr("help", inst)
			return object.None, nil
		},
	})

	return m
}

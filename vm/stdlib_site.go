package vm

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildSite() *object.Module {
	m := &object.Module{Name: "site", Dict: object.NewDict()}

	// Determine platform-appropriate paths
	prefix := "/usr"
	userBase := ""
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			userBase = filepath.Join(appData, "Python")
		} else {
			userBase = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "Python")
		}
	} else {
		home, _ := os.UserHomeDir()
		userBase = filepath.Join(home, ".local")
	}
	userSite := ""
	if runtime.GOOS == "windows" {
		userSite = filepath.Join(userBase, "Python314", "site-packages")
	} else {
		userSite = filepath.Join(userBase, "lib", "python3.14", "site-packages")
	}

	m.Dict.SetStr("ENABLE_USER_SITE", object.True)
	m.Dict.SetStr("PREFIXES", &object.List{V: []object.Object{
		&object.Str{V: prefix},
		&object.Str{V: prefix},
	}})
	m.Dict.SetStr("USER_BASE", &object.Str{V: userBase})
	m.Dict.SetStr("USER_SITE", &object.Str{V: userSite})

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

	m.Dict.SetStr("addsitedir", &object.BuiltinFunc{
		Name: "addsitedir",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("addpackage", &object.BuiltinFunc{
		Name: "addpackage",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("addsitepackages", &object.BuiltinFunc{
		Name: "addsitepackages",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("main", &object.BuiltinFunc{
		Name: "main",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("setquit", &object.BuiltinFunc{
		Name: "setquit",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("setcopyright", &object.BuiltinFunc{
		Name: "setcopyright",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("sethelper", &object.BuiltinFunc{
		Name: "sethelper",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("execsitecustomize", &object.BuiltinFunc{
		Name: "execsitecustomize",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("execusercustomize", &object.BuiltinFunc{
		Name: "execusercustomize",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("check_enableusersite", &object.BuiltinFunc{
		Name: "check_enableusersite",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		},
	})

	m.Dict.SetStr("removeduppaths", &object.BuiltinFunc{
		Name: "removeduppaths",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("abs_paths", &object.BuiltinFunc{
		Name: "abs_paths",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	return m
}

package vm

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildSysconfig() *object.Module {
	m := &object.Module{Name: "sysconfig", Dict: object.NewDict()}

	goVersion := runtime.Version()
	_ = goVersion

	getPythonVersion := func() string { return "3.14" }
	getPlatform := func() string {
		switch runtime.GOOS {
		case "darwin":
			return fmt.Sprintf("macosx-14.0-%s", runtime.GOARCH)
		case "windows":
			return fmt.Sprintf("win-%s", runtime.GOARCH)
		default:
			return fmt.Sprintf("linux-%s", runtime.GOARCH)
		}
	}

	schemes := []string{"posix_prefix", "posix_home", "posix_user", "nt", "nt_user", "osx_framework_user"}

	defaultPaths := map[string]string{
		"stdlib":     "/usr/lib/python3.14",
		"platstdlib": "/usr/lib/python3.14",
		"platlib":    "/usr/lib/python3.14/lib-dynload",
		"purelib":    "/usr/lib/python3.14",
		"include":    "/usr/include/python3.14",
		"scripts":    "/usr/bin",
		"data":       "/usr",
	}

	configVars := map[string]string{
		"prefix":          "/usr",
		"exec_prefix":     "/usr",
		"py_version":      "3.14.0",
		"py_version_short": "3.14",
		"base":            "/usr",
		"platbase":        "/usr",
		"installed_base":  "/usr",
		"installed_platbase": "/usr",
		"BINDIR":          "/usr/bin",
		"LIBDIR":          "/usr/lib",
		"INCLUDEDIR":      "/usr/include",
		"PYTHON_VERSION":  "3.14",
		"MULTIARCH":       "",
		"EXT_SUFFIX":      ".cpython-314-x86_64-linux-gnu.so",
		"SOABI":           "cpython-314-x86_64-linux-gnu",
		"SHLIB_SUFFIX":    ".so",
		"CC":              "gcc",
		"CXX":             "g++",
		"CFLAGS":          "-O2",
		"LDFLAGS":         "",
		"VERSION":         "3.14",
		"ABIFLAGS":        "",
		"SIZEOF_LONG":     "8",
		"SIZEOF_VOID_P":   "8",
	}

	m.Dict.SetStr("get_python_version", &object.BuiltinFunc{
		Name: "get_python_version",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: getPythonVersion()}, nil
		},
	})

	m.Dict.SetStr("get_platform", &object.BuiltinFunc{
		Name: "get_platform",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: getPlatform()}, nil
		},
	})

	m.Dict.SetStr("get_scheme_names", &object.BuiltinFunc{
		Name: "get_scheme_names",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			items := make([]object.Object, len(schemes))
			for idx, s := range schemes {
				items[idx] = &object.Str{V: s}
			}
			return &object.Tuple{V: items}, nil
		},
	})

	m.Dict.SetStr("get_default_scheme", &object.BuiltinFunc{
		Name: "get_default_scheme",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if runtime.GOOS == "windows" {
				return &object.Str{V: "nt"}, nil
			}
			return &object.Str{V: "posix_prefix"}, nil
		},
	})

	m.Dict.SetStr("get_paths", &object.BuiltinFunc{
		Name: "get_paths",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			d := object.NewDict()
			for k, v := range defaultPaths {
				d.SetStr(k, &object.Str{V: v})
			}
			return d, nil
		},
	})

	m.Dict.SetStr("get_path", &object.BuiltinFunc{
		Name: "get_path",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			name := ""
			if s, ok := a[0].(*object.Str); ok {
				name = s.V
			}
			if v, ok := defaultPaths[name]; ok {
				return &object.Str{V: v}, nil
			}
			return object.None, nil
		},
	})

	m.Dict.SetStr("get_path_names", &object.BuiltinFunc{
		Name: "get_path_names",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			names := []string{"stdlib", "platstdlib", "platlib", "purelib", "include", "scripts", "data"}
			items := make([]object.Object, len(names))
			for idx, n := range names {
				items[idx] = &object.Str{V: n}
			}
			return &object.Tuple{V: items}, nil
		},
	})

	m.Dict.SetStr("get_config_vars", &object.BuiltinFunc{
		Name: "get_config_vars",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				d := object.NewDict()
				for k, v := range configVars {
					d.SetStr(k, &object.Str{V: v})
				}
				return d, nil
			}
			// With args: return list of values for each key
			items := make([]object.Object, len(a))
			for idx, arg := range a {
				key := ""
				if s, ok := arg.(*object.Str); ok {
					key = s.V
				}
				if v, ok := configVars[key]; ok {
					items[idx] = &object.Str{V: v}
				} else {
					items[idx] = object.None
				}
			}
			return &object.List{V: items}, nil
		},
	})

	m.Dict.SetStr("get_config_var", &object.BuiltinFunc{
		Name: "get_config_var",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			key := ""
			if s, ok := a[0].(*object.Str); ok {
				key = s.V
			}
			if v, ok := configVars[key]; ok {
				return &object.Str{V: v}, nil
			}
			return object.None, nil
		},
	})

	m.Dict.SetStr("get_makefile_filename", &object.BuiltinFunc{
		Name: "get_makefile_filename",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "/usr/lib/python3.14/config-3.14/Makefile"}, nil
		},
	})

	m.Dict.SetStr("get_config_h_filename", &object.BuiltinFunc{
		Name: "get_config_h_filename",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "/usr/include/python3.14/pyconfig.h"}, nil
		},
	})

	m.Dict.SetStr("is_python_build", &object.BuiltinFunc{
		Name: "is_python_build",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		},
	})

	m.Dict.SetStr("parse_config_h", &object.BuiltinFunc{
		Name: "parse_config_h",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewDict(), nil
		},
	})

	_ = strings.HasPrefix
	return m
}

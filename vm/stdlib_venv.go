package vm

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildVenv() *object.Module {
	m := &object.Module{Name: "venv", Dict: object.NewDict()}

	// ── EnvBuilder class ──────────────────────────────────────────────────────

	envBuilderCls := &object.Class{Name: "EnvBuilder", Dict: object.NewDict()}

	envBuilderCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			// defaults
			inst.Dict.SetStr("system_site_packages", &object.Bool{V: false})
			inst.Dict.SetStr("clear", &object.Bool{V: false})
			inst.Dict.SetStr("symlinks", &object.Bool{V: false})
			inst.Dict.SetStr("upgrade", &object.Bool{V: false})
			inst.Dict.SetStr("with_pip", &object.Bool{V: false})
			inst.Dict.SetStr("prompt", object.None)
			inst.Dict.SetStr("upgrade_deps", &object.Bool{V: false})
			inst.Dict.SetStr("scm_ignore_files", &object.List{V: nil})
			if kw != nil {
				ks, vs := kw.Items()
				for idx, k := range ks {
					if s, ok := k.(*object.Str); ok {
						inst.Dict.SetStr(s.V, vs[idx])
					}
				}
			}
			return object.None, nil
		},
	})

	// buildContext creates a SimpleNamespace-like Instance with venv context attrs.
	buildContext := func(envDir string) *object.Instance {
		ctx := &object.Instance{Class: &object.Class{Name: "SimpleNamespace", Dict: object.NewDict()}, Dict: object.NewDict()}
		abs, _ := filepath.Abs(envDir)
		envName := filepath.Base(abs)
		binName := "bin"
		if runtime.GOOS == "windows" {
			binName = "Scripts"
		}
		executable, _ := os.Executable()
		ctx.Dict.SetStr("env_dir", &object.Str{V: abs})
		ctx.Dict.SetStr("env_name", &object.Str{V: envName})
		ctx.Dict.SetStr("prompt", &object.Str{V: "(" + envName + ") "})
		ctx.Dict.SetStr("executable", &object.Str{V: executable})
		ctx.Dict.SetStr("inc_path", &object.Str{V: filepath.Join(abs, "include")})
		ctx.Dict.SetStr("lib_path", &object.Str{V: filepath.Join(abs, "lib")})
		ctx.Dict.SetStr("bin_path", &object.Str{V: filepath.Join(abs, binName)})
		ctx.Dict.SetStr("bin_name", &object.Str{V: binName})
		ctx.Dict.SetStr("env_exe", &object.Str{V: filepath.Join(abs, binName, "python")})
		ctx.Dict.SetStr("env_exec_cmd", &object.Str{V: filepath.Join(abs, binName, "python")})
		return ctx
	}

	noop := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		}
	}

	envBuilderCls.Dict.SetStr("ensure_directories", &object.BuiltinFunc{
		Name: "ensure_directories",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			envDir := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					envDir = s.V
				}
			}
			return buildContext(envDir), nil
		},
	})

	envBuilderCls.Dict.SetStr("create", &object.BuiltinFunc{
		Name: "create",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	envBuilderCls.Dict.SetStr("create_configuration", noop("create_configuration"))
	envBuilderCls.Dict.SetStr("setup_python", noop("setup_python"))
	envBuilderCls.Dict.SetStr("setup_scripts", noop("setup_scripts"))
	envBuilderCls.Dict.SetStr("post_setup", noop("post_setup"))
	envBuilderCls.Dict.SetStr("upgrade_dependencies", noop("upgrade_dependencies"))
	envBuilderCls.Dict.SetStr("install_scripts", noop("install_scripts"))
	envBuilderCls.Dict.SetStr("create_git_ignore_file", noop("create_git_ignore_file"))

	m.Dict.SetStr("EnvBuilder", envBuilderCls)

	// ── module-level create() ─────────────────────────────────────────────────

	m.Dict.SetStr("create", &object.BuiltinFunc{
		Name: "create",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// create(env_dir, **kwargs) → EnvBuilder(**kwargs).create(env_dir)
			_ = a  // env_dir and kwargs ignored (stub)
			return object.None, nil
		},
	})

	return m
}

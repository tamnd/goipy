package vm

import (
	"os"

	"github.com/tamnd/goipy/marshal"
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildRunpy() *object.Module {
	m := &object.Module{Name: "runpy", Dict: object.NewDict()}

	// ── run_module ────────────────────────────────────────────────────────

	m.Dict.SetStr("run_module", &object.BuiltinFunc{Name: "run_module",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "run_module() missing mod_name argument")
			}
			modNameStr, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "run_module() mod_name must be str")
			}
			modName := modNameStr.V

			var initGlobals *object.Dict
			runName := modName

			if len(a) >= 2 && a[1] != object.None {
				if d, ok2 := a[1].(*object.Dict); ok2 {
					initGlobals = d
				}
			}
			if len(a) >= 3 && a[2] != object.None {
				if s, ok2 := a[2].(*object.Str); ok2 {
					runName = s.V
				}
			}
			// alter_sys (a[3]) is intentionally ignored.
			if kw != nil {
				if v, ok2 := kw.GetStr("init_globals"); ok2 && v != object.None {
					if d, ok3 := v.(*object.Dict); ok3 {
						initGlobals = d
					}
				}
				if v, ok2 := kw.GetStr("run_name"); ok2 && v != object.None {
					if s, ok3 := v.(*object.Str); ok3 {
						runName = s.V
					}
				}
			}

			mod, err := i.loadModule(modName)
			if err != nil {
				return nil, object.Errorf(i.importErr, "No module named %q", modName)
			}
			if mod == nil {
				return nil, object.Errorf(i.importErr, "No module named %q", modName)
			}

			// Snapshot the module's dict into a fresh result dict.
			result := object.NewDict()
			keys, vals := mod.Dict.Items()
			for idx, k := range keys {
				result.Set(k, vals[idx]) //nolint
			}

			// Override __name__ with run_name.
			result.SetStr("__name__", &object.Str{V: runName})

			// Inject init_globals on top (overrides module entries).
			if initGlobals != nil {
				ikeys, ivals := initGlobals.Items()
				for idx, k := range ikeys {
					result.Set(k, ivals[idx]) //nolint
				}
			}

			return result, nil
		}})

	// ── run_path ──────────────────────────────────────────────────────────

	m.Dict.SetStr("run_path", &object.BuiltinFunc{Name: "run_path",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "run_path() missing path_name argument")
			}
			pathNameStr, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "run_path() path_name must be str")
			}
			pathName := pathNameStr.V

			var initGlobals *object.Dict
			runName := "<run_path>"

			if len(a) >= 2 && a[1] != object.None {
				if d, ok2 := a[1].(*object.Dict); ok2 {
					initGlobals = d
				}
			}
			if len(a) >= 3 && a[2] != object.None {
				if s, ok2 := a[2].(*object.Str); ok2 {
					runName = s.V
				}
			}
			if kw != nil {
				if v, ok2 := kw.GetStr("init_globals"); ok2 && v != object.None {
					if d, ok3 := v.(*object.Dict); ok3 {
						initGlobals = d
					}
				}
				if v, ok2 := kw.GetStr("run_name"); ok2 && v != object.None {
					if s, ok3 := v.(*object.Str); ok3 {
						runName = s.V
					}
				}
			}

			// Verify path exists.
			if _, err := os.Stat(pathName); err != nil {
				return nil, object.Errorf(i.fileNotFoundErr, "[Errno 2] No such file or directory: %q", pathName)
			}

			// Try to load and execute a .pyc alongside the .py file.
			pycPath := pathName + "c"
			code, err := marshal.LoadPyc(pycPath)
			if err != nil {
				// No compilable .pyc — return minimal namespace.
				result := object.NewDict()
				result.SetStr("__name__", &object.Str{V: runName})
				result.SetStr("__file__", &object.Str{V: pathName})
				if initGlobals != nil {
					ikeys, ivals := initGlobals.Items()
					for idx, k := range ikeys {
						result.Set(k, ivals[idx]) //nolint
					}
				}
				return result, nil
			}

			// Build a fresh globals dict for execution.
			globals := object.NewDict()
			globals.SetStr("__name__", &object.Str{V: runName})
			globals.SetStr("__file__", &object.Str{V: pathName})
			globals.SetStr("__builtins__", i.Builtins)
			if initGlobals != nil {
				ikeys, ivals := initGlobals.Items()
				for idx, k := range ikeys {
					globals.Set(k, ivals[idx]) //nolint
				}
			}

			frame := NewFrame(code, globals, i.Builtins, globals)
			if _, err := i.runFrame(frame); err != nil {
				return nil, err
			}

			return globals, nil
		}})

	return m
}

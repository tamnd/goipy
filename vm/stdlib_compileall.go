package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildCompileall constructs the compileall module with CPython 3.14 API:
// compile_file(), compile_dir(), compile_path(), and main().
func (i *Interp) buildCompileall() *object.Module {
	m := &object.Module{Name: "compileall", Dict: object.NewDict()}

	m.Dict.SetStr("__all__", &object.List{V: []object.Object{
		&object.Str{V: "compile_dir"},
		&object.Str{V: "compile_file"},
		&object.Str{V: "compile_path"},
	}})

	// ── compile_file(fullname, ...) → bool ────────────────────────────────
	// Stub: accepts all arguments, returns True (success).
	// Real CPython compiles the .py file to .pyc; returns False on syntax error.

	m.Dict.SetStr("compile_file", &object.BuiltinFunc{
		Name: "compile_file",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "compile_file() missing fullname argument")
			}
			return object.True, nil
		},
	})

	// ── compile_dir(dir, ...) → bool ──────────────────────────────────────
	// Stub: accepts all arguments, returns True.

	m.Dict.SetStr("compile_dir", &object.BuiltinFunc{
		Name: "compile_dir",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "compile_dir() missing dir argument")
			}
			return object.True, nil
		},
	})

	// ── compile_path(skip_curdir=1, maxlevels=0, ...) → bool ─────────────
	// Stub: returns True.

	m.Dict.SetStr("compile_path", &object.BuiltinFunc{
		Name: "compile_path",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		},
	})

	// ── main() ────────────────────────────────────────────────────────────

	m.Dict.SetStr("main", &object.BuiltinFunc{
		Name: "main",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	return m
}

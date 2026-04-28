package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildCodeop() *object.Module {
	m := &object.Module{Name: "codeop", Dict: object.NewDict()}

	// ── Constants ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("PyCF_DONT_IMPLY_DEDENT", object.NewInt(512))
	m.Dict.SetStr("PyCF_ALLOW_INCOMPLETE_INPUT", object.NewInt(16384))
	m.Dict.SetStr("PyCF_ONLY_AST", object.NewInt(1024))

	// ── Compile class ─────────────────────────────────────────────────────────

	compileClass := &object.Class{Name: "Compile", Dict: object.NewDict()}

	compileClass.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			// PyCF_DONT_IMPLY_DEDENT | PyCF_ALLOW_INCOMPLETE_INPUT
			self.Dict.SetStr("flags", object.NewInt(16896))
			return object.None, nil
		},
	})

	compileClass.Dict.SetStr("__call__", &object.BuiltinFunc{
		Name: "__call__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			source := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					source = s.V
				}
			}
			if isIncompleteSource(source) {
				return nil, object.Errorf(i.syntaxErr, "incomplete input")
			}
			return &object.Code{Name: "<Compile>"}, nil
		},
	})

	m.Dict.SetStr("Compile", compileClass)

	// ── CommandCompiler class ─────────────────────────────────────────────────

	ccClass := &object.Class{Name: "CommandCompiler", Dict: object.NewDict()}

	ccClass.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			compiler := &object.Instance{Class: compileClass, Dict: object.NewDict()}
			compiler.Dict.SetStr("flags", object.NewInt(16896))
			self.Dict.SetStr("compiler", compiler)
			return object.None, nil
		},
	})

	ccClass.Dict.SetStr("__call__", &object.BuiltinFunc{
		Name: "__call__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			source := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					source = s.V
				}
			}
			if kw != nil {
				if sv, ok := kw.GetStr("source"); ok {
					if s, ok2 := sv.(*object.Str); ok2 {
						source = s.V
					}
				}
			}
			if isIncompleteSource(source) {
				return object.None, nil
			}
			return &object.Code{Name: "<CommandCompiler>"}, nil
		},
	})

	m.Dict.SetStr("CommandCompiler", ccClass)

	// ── compile_command ───────────────────────────────────────────────────────

	m.Dict.SetStr("compile_command", &object.BuiltinFunc{
		Name: "compile_command",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			source := ""
			if len(a) >= 1 {
				if s, ok := a[0].(*object.Str); ok {
					source = s.V
				}
			}
			if isIncompleteSource(source) {
				return object.None, nil
			}
			return &object.Code{Name: "<compile_command>"}, nil
		},
	})

	return m
}

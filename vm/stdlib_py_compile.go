package vm

import (
	"fmt"

	"github.com/tamnd/goipy/object"
)

// buildPyCompile constructs the py_compile module with CPython 3.14 API:
// PyCompileError exception, PycInvalidationMode enum, compile(), main().
func (i *Interp) buildPyCompile() *object.Module {
	m := &object.Module{Name: "py_compile", Dict: object.NewDict()}

	m.Dict.SetStr("__all__", &object.List{V: []object.Object{
		&object.Str{V: "compile"},
		&object.Str{V: "main"},
		&object.Str{V: "PyCompileError"},
		&object.Str{V: "PycInvalidationMode"},
	}})

	// ── PyCompileError ────────────────────────────────────────────────────
	// Subclass of Exception. Constructor: PyCompileError(exc_type, exc_value, file, msg='')
	// Exception fast-path stores args in exc.Args; attributes derived via __getattr__.

	pyCompileErrCls := &object.Class{
		Name:  "PyCompileError",
		Dict:  object.NewDict(),
		Bases: []*object.Class{i.exception},
	}

	// excTypeName extracts the class name from an exc_type argument.
	// exc_type is expected to be a *object.Class; fall back to TypeName.
	excTypeName := func(v object.Object) string {
		if cls, ok := v.(*object.Class); ok {
			return cls.Name
		}
		return object.TypeName(v)
	}

	// computeMsg computes the default error message from args.
	computeMsg := func(args []object.Object) string {
		if len(args) < 3 {
			return ""
		}
		typeName := excTypeName(args[0])
		excVal := object.Str_(args[1])
		return fmt.Sprintf("Sorry: %s: %s", typeName, excVal)
	}

	// __getattr__ is called for attribute access when the attribute is not
	// found in exc.Dict or the hardcoded exception fields. It derives
	// exc_type_name, exc_value, file, and msg from exc.Args.
	pyCompileErrCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{
		Name: "__getattr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			attrName, ok := a[1].(*object.Str)
			if !ok {
				return object.None, nil
			}

			var args []object.Object
			switch self := a[0].(type) {
			case *object.Exception:
				if self.Args != nil {
					args = self.Args.V
				}
			case *object.Instance:
				// Fallback for Instance-based construction
				if v, ok2 := self.Dict.GetStr("_pycompile_args"); ok2 {
					if tup, ok3 := v.(*object.Tuple); ok3 {
						args = tup.V
					}
				}
			}

			switch attrName.V {
			case "exc_type_name":
				if len(args) > 0 {
					return &object.Str{V: excTypeName(args[0])}, nil
				}
				return &object.Str{V: ""}, nil
			case "exc_value":
				if len(args) > 1 {
					return args[1], nil
				}
				return &object.Str{V: ""}, nil
			case "file":
				if len(args) > 2 {
					return args[2], nil
				}
				return &object.Str{V: ""}, nil
			case "msg":
				if len(args) > 3 {
					if sv, ok2 := args[3].(*object.Str); ok2 && sv.V != "" {
						return sv, nil
					}
				}
				return &object.Str{V: computeMsg(args)}, nil
			}
			return nil, object.Errorf(i.attrErr, "'PyCompileError' object has no attribute '%s'", attrName.V)
		},
	})

	// __str__ returns the msg attribute.
	pyCompileErrCls.Dict.SetStr("__str__", &object.BuiltinFunc{
		Name: "__str__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "PyCompileError"}, nil
			}
			var args []object.Object
			if exc, ok := a[0].(*object.Exception); ok && exc.Args != nil {
				args = exc.Args.V
			}
			if len(args) > 3 {
				if sv, ok := args[3].(*object.Str); ok && sv.V != "" {
					return sv, nil
				}
			}
			return &object.Str{V: computeMsg(args)}, nil
		},
	})

	m.Dict.SetStr("PyCompileError", pyCompileErrCls)

	// ── PycInvalidationMode ───────────────────────────────────────────────
	// Enum with TIMESTAMP=1, CHECKED_HASH=2, UNCHECKED_HASH=3.

	pimCls := &object.Class{Name: "PycInvalidationMode", Dict: object.NewDict()}

	makeMember := func(name string, val int64) *object.Instance {
		mem := &object.Instance{Class: pimCls, Dict: object.NewDict()}
		mem.Dict.SetStr("_name_", &object.Str{V: name})
		mem.Dict.SetStr("name", &object.Str{V: name})
		mem.Dict.SetStr("_value_", object.NewInt(val))
		mem.Dict.SetStr("value", object.NewInt(val))
		return mem
	}

	tsInst := makeMember("TIMESTAMP", 1)
	chInst := makeMember("CHECKED_HASH", 2)
	ucInst := makeMember("UNCHECKED_HASH", 3)
	allMembers := []object.Object{tsInst, chInst, ucInst}

	pimCls.Dict.SetStr("TIMESTAMP", tsInst)
	pimCls.Dict.SetStr("CHECKED_HASH", chInst)
	pimCls.Dict.SetStr("UNCHECKED_HASH", ucInst)

	pimCls.Dict.SetStr("__iter__", &object.BuiltinFunc{
		Name: "__iter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			items := make([]object.Object, len(allMembers))
			copy(items, allMembers)
			return &object.List{V: items}, nil
		},
	})

	pimCls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "<enum 'PycInvalidationMode'>"}, nil
		},
	})

	// Accessing PycInvalidationMode.TIMESTAMP returns the tsInst instance.
	// We expose it as a class-level attribute (already set above via Dict.SetStr).
	// The class itself is the enum "type"; instances are accessed as attributes.
	pimInst := &object.Instance{Class: pimCls, Dict: object.NewDict()}
	pimInst.Dict.SetStr("TIMESTAMP", tsInst)
	pimInst.Dict.SetStr("CHECKED_HASH", chInst)
	pimInst.Dict.SetStr("UNCHECKED_HASH", ucInst)

	// Wrap in an iterator-supporting instance so list(PycInvalidationMode) works.
	pimInst.Dict.SetStr("__iter__", &object.BuiltinFunc{
		Name: "__iter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			items := make([]object.Object, len(allMembers))
			copy(items, allMembers)
			return &object.List{V: items}, nil
		},
	})

	m.Dict.SetStr("PycInvalidationMode", pimInst)

	// ── compile() ─────────────────────────────────────────────────────────
	// Stub: no real bytecode compiler. Returns None.

	m.Dict.SetStr("compile", &object.BuiltinFunc{
		Name: "compile",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "compile() missing file argument")
			}
			return object.None, nil
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

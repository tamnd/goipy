package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildSymtable constructs the symtable module with the CPython 3.14 API surface.
// symtable.symtable() returns a stub SymbolTable; Symbol is fully implemented.
func (i *Interp) buildSymtable() *object.Module {
	m := &object.Module{Name: "symtable", Dict: object.NewDict()}

	// ── Integer constants ─────────────────────────────────────────────────

	m.Dict.SetStr("CELL", object.NewInt(5))
	m.Dict.SetStr("FREE", object.NewInt(4))
	m.Dict.SetStr("LOCAL", object.NewInt(1))
	m.Dict.SetStr("GLOBAL_EXPLICIT", object.NewInt(2))
	m.Dict.SetStr("GLOBAL_IMPLICIT", object.NewInt(3))
	m.Dict.SetStr("SCOPE_MASK", object.NewInt(15))
	m.Dict.SetStr("SCOPE_OFF", object.NewInt(12))
	m.Dict.SetStr("USE", object.NewInt(16))
	m.Dict.SetStr("DEF_GLOBAL", object.NewInt(1))
	m.Dict.SetStr("DEF_LOCAL", object.NewInt(2))
	m.Dict.SetStr("DEF_PARAM", object.NewInt(4))
	m.Dict.SetStr("DEF_NONLOCAL", object.NewInt(8))
	m.Dict.SetStr("DEF_FREE_CLASS", object.NewInt(64))
	m.Dict.SetStr("DEF_IMPORT", object.NewInt(128))
	m.Dict.SetStr("DEF_BOUND", object.NewInt(134))
	m.Dict.SetStr("DEF_ANNOT", object.NewInt(256))
	m.Dict.SetStr("DEF_COMP_ITER", object.NewInt(512))
	m.Dict.SetStr("DEF_TYPE_PARAM", object.NewInt(1024))
	m.Dict.SetStr("DEF_COMP_CELL", object.NewInt(2048))

	// ── SymbolTableType ───────────────────────────────────────────────────
	// Exposed as a simple namespace object mirroring Python's StrEnum.

	sttCls := &object.Class{Name: "SymbolTableType", Dict: object.NewDict()}
	sttInst := &object.Instance{Class: sttCls, Dict: object.NewDict()}
	for _, pair := range [][2]string{
		{"MODULE", "module"},
		{"FUNCTION", "function"},
		{"CLASS", "class"},
		{"ANNOTATION", "annotation"},
		{"TYPE_ALIAS", "type alias"},
		{"TYPE_PARAMETERS", "type parameters"},
		{"TYPE_VARIABLE", "type variable"},
	} {
		sttInst.Dict.SetStr(pair[0], &object.Str{V: pair[1]})
	}
	m.Dict.SetStr("SymbolTableType", sttInst)

	// ── Symbol class ──────────────────────────────────────────────────────

	// scope constants (mirrors Python's SCOPE_OFF=12, SCOPE_MASK=15)
	const scopeOff = 12
	const scopeMask = 15
	const scopeLocal = 1
	const scopeGlobalExplicit = 2
	const scopeGlobalImplicit = 3
	const scopeFree = 4

	// defUse flags
	const defLocal = 2
	const defParam = 4
	const defNonlocal = 8
	const defFreeClass = 64
	const defImport = 128
	const defAnnot = 256
	const defCompIter = 512
	const defTypeParam = 1024
	const defCompCell = 2048
	const useFlag = 16

	// symFlags reads the flags int64 from an Instance's "_flags" field.
	symFlags := func(self *object.Instance) int64 {
		v, ok := self.Dict.GetStr("_flags")
		if !ok {
			return 0
		}
		iv, ok2 := v.(*object.Int)
		if !ok2 {
			return 0
		}
		return iv.Int64()
	}

	// symScope extracts the scope bits from flags.
	symScope := func(flags int64) int64 {
		return (flags >> scopeOff) & scopeMask
	}

	// makeBoolMethod wraps a boolean check as a BuiltinFunc method.
	makeBoolMethod := func(name string, fn func(flags int64) bool) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) < 1 {
					return object.False, nil
				}
				self := a[0].(*object.Instance)
				flags := symFlags(self)
				if fn(flags) {
					return object.True, nil
				}
				return object.False, nil
			},
		}
	}

	symCls := &object.Class{Name: "Symbol", Dict: object.NewDict()}

	symCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			// positional: name, flags, namespaces
			name := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					name = s.V
				}
			}
			flags := int64(0)
			if len(a) >= 3 {
				if iv, ok := a[2].(*object.Int); ok {
					flags = iv.Int64()
				}
			}
			var namespaces object.Object = &object.Tuple{V: nil}
			if len(a) >= 4 {
				namespaces = a[3]
			}
			// keyword: namespaces=, module_scope=
			if kw != nil {
				if ns, ok := kw.GetStr("namespaces"); ok {
					namespaces = ns
				}
			}
			self.Dict.SetStr("_name", &object.Str{V: name})
			self.Dict.SetStr("_flags", object.NewInt(flags))
			self.Dict.SetStr("_namespaces", namespaces)
			return object.None, nil
		},
	})

	symCls.Dict.SetStr("get_name", &object.BuiltinFunc{
		Name: "get_name",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			self := a[0].(*object.Instance)
			if v, ok := self.Dict.GetStr("_name"); ok {
				return v, nil
			}
			return &object.Str{V: ""}, nil
		},
	})

	symCls.Dict.SetStr("is_referenced", makeBoolMethod("is_referenced", func(f int64) bool {
		return f&useFlag != 0
	}))
	symCls.Dict.SetStr("is_assigned", makeBoolMethod("is_assigned", func(f int64) bool {
		return f&defLocal != 0
	}))
	symCls.Dict.SetStr("is_parameter", makeBoolMethod("is_parameter", func(f int64) bool {
		return f&defParam != 0
	}))
	symCls.Dict.SetStr("is_imported", makeBoolMethod("is_imported", func(f int64) bool {
		return f&defImport != 0
	}))
	symCls.Dict.SetStr("is_nonlocal", makeBoolMethod("is_nonlocal", func(f int64) bool {
		return f&defNonlocal != 0
	}))
	symCls.Dict.SetStr("is_free_class", makeBoolMethod("is_free_class", func(f int64) bool {
		return f&defFreeClass != 0
	}))
	symCls.Dict.SetStr("is_annotated", makeBoolMethod("is_annotated", func(f int64) bool {
		return f&defAnnot != 0
	}))
	symCls.Dict.SetStr("is_comp_iter", makeBoolMethod("is_comp_iter", func(f int64) bool {
		return f&defCompIter != 0
	}))
	symCls.Dict.SetStr("is_comp_cell", makeBoolMethod("is_comp_cell", func(f int64) bool {
		return f&defCompCell != 0
	}))
	symCls.Dict.SetStr("is_type_parameter", makeBoolMethod("is_type_parameter", func(f int64) bool {
		return f&defTypeParam != 0
	}))
	symCls.Dict.SetStr("is_global", makeBoolMethod("is_global", func(f int64) bool {
		sc := symScope(f)
		return sc == scopeGlobalExplicit || sc == scopeGlobalImplicit
	}))
	symCls.Dict.SetStr("is_declared_global", makeBoolMethod("is_declared_global", func(f int64) bool {
		return symScope(f) == scopeGlobalExplicit
	}))
	symCls.Dict.SetStr("is_local", makeBoolMethod("is_local", func(f int64) bool {
		return symScope(f) == scopeLocal
	}))
	symCls.Dict.SetStr("is_free", makeBoolMethod("is_free", func(f int64) bool {
		return symScope(f) == scopeFree
	}))
	symCls.Dict.SetStr("is_namespace", &object.BuiltinFunc{
		Name: "is_namespace",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			self := a[0].(*object.Instance)
			ns, ok := self.Dict.GetStr("_namespaces")
			if !ok {
				return object.False, nil
			}
			switch v := ns.(type) {
			case *object.List:
				if len(v.V) > 0 {
					return object.True, nil
				}
			case *object.Tuple:
				if len(v.V) > 0 {
					return object.True, nil
				}
			}
			return object.False, nil
		},
	})
	symCls.Dict.SetStr("get_namespaces", &object.BuiltinFunc{
		Name: "get_namespaces",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Tuple{V: nil}, nil
			}
			self := a[0].(*object.Instance)
			if ns, ok := self.Dict.GetStr("_namespaces"); ok {
				return ns, nil
			}
			return &object.Tuple{V: nil}, nil
		},
	})
	symCls.Dict.SetStr("get_namespace", &object.BuiltinFunc{
		Name: "get_namespace",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.valueErr, "Symbol.get_namespace(): no namespaces")
			}
			self := a[0].(*object.Instance)
			ns, ok := self.Dict.GetStr("_namespaces")
			if !ok {
				return nil, object.Errorf(i.valueErr, "Symbol.get_namespace(): no namespaces")
			}
			switch v := ns.(type) {
			case *object.List:
				if len(v.V) == 1 {
					return v.V[0], nil
				}
				if len(v.V) == 0 {
					return nil, object.Errorf(i.valueErr, "Symbol.get_namespace(): no namespaces")
				}
				return nil, object.Errorf(i.valueErr, "Symbol.get_namespace(): multiple namespaces")
			case *object.Tuple:
				if len(v.V) == 1 {
					return v.V[0], nil
				}
				if len(v.V) == 0 {
					return nil, object.Errorf(i.valueErr, "Symbol.get_namespace(): no namespaces")
				}
				return nil, object.Errorf(i.valueErr, "Symbol.get_namespace(): multiple namespaces")
			}
			return nil, object.Errorf(i.valueErr, "Symbol.get_namespace(): no namespaces")
		},
	})

	m.Dict.SetStr("Symbol", symCls)

	// ── SymbolTable class ─────────────────────────────────────────────────

	// makeSTMethod creates a simple method that reads a field from the instance Dict.
	makeSTMethod := func(name string, field string, def object.Object) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) < 1 {
					return def, nil
				}
				self := a[0].(*object.Instance)
				if v, ok := self.Dict.GetStr(field); ok {
					return v, nil
				}
				return def, nil
			},
		}
	}

	// makeDefaultSymTab creates an empty module-level SymbolTable instance.
	var stCls *object.Class
	makeDefaultSymTab := func(cls *object.Class) *object.Instance {
		inst := &object.Instance{Class: cls, Dict: object.NewDict()}
		inst.Dict.SetStr("_type", &object.Str{V: "module"})
		inst.Dict.SetStr("_name", &object.Str{V: "top"})
		inst.Dict.SetStr("_lineno", object.NewInt(0))
		inst.Dict.SetStr("_id", object.NewInt(0))
		inst.Dict.SetStr("_optimized", object.False)
		inst.Dict.SetStr("_nested", object.False)
		inst.Dict.SetStr("_symbols", &object.List{V: nil})
		inst.Dict.SetStr("_children", &object.List{V: nil})
		return inst
	}

	stCls = &object.Class{Name: "SymbolTable", Dict: object.NewDict()}

	stCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			// Set defaults; callers who go through symtable() will override
			self.Dict.SetStr("_type", &object.Str{V: "module"})
			self.Dict.SetStr("_name", &object.Str{V: "top"})
			self.Dict.SetStr("_lineno", object.NewInt(0))
			self.Dict.SetStr("_id", object.NewInt(0))
			self.Dict.SetStr("_optimized", object.False)
			self.Dict.SetStr("_nested", object.False)
			self.Dict.SetStr("_symbols", &object.List{V: nil})
			self.Dict.SetStr("_children", &object.List{V: nil})
			return object.None, nil
		},
	})

	stCls.Dict.SetStr("get_type", makeSTMethod("get_type", "_type", &object.Str{V: "module"}))
	stCls.Dict.SetStr("get_name", makeSTMethod("get_name", "_name", &object.Str{V: "top"}))
	stCls.Dict.SetStr("get_lineno", makeSTMethod("get_lineno", "_lineno", object.NewInt(0)))
	stCls.Dict.SetStr("get_id", makeSTMethod("get_id", "_id", object.NewInt(0)))
	stCls.Dict.SetStr("is_optimized", makeSTMethod("is_optimized", "_optimized", object.False))
	stCls.Dict.SetStr("is_nested", makeSTMethod("is_nested", "_nested", object.False))

	stCls.Dict.SetStr("has_children", &object.BuiltinFunc{
		Name: "has_children",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			self := a[0].(*object.Instance)
			if v, ok := self.Dict.GetStr("_children"); ok {
				if l, ok2 := v.(*object.List); ok2 && len(l.V) > 0 {
					return object.True, nil
				}
			}
			return object.False, nil
		},
	})

	stCls.Dict.SetStr("get_symbols", &object.BuiltinFunc{
		Name: "get_symbols",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{V: nil}, nil
			}
			self := a[0].(*object.Instance)
			if v, ok := self.Dict.GetStr("_symbols"); ok {
				return v, nil
			}
			return &object.List{V: nil}, nil
		},
	})

	stCls.Dict.SetStr("get_children", &object.BuiltinFunc{
		Name: "get_children",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{V: nil}, nil
			}
			self := a[0].(*object.Instance)
			if v, ok := self.Dict.GetStr("_children"); ok {
				return v, nil
			}
			return &object.List{V: nil}, nil
		},
	})

	stCls.Dict.SetStr("get_identifiers", &object.BuiltinFunc{
		Name: "get_identifiers",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{V: nil}, nil
			}
			ii := interp.(*Interp)
			self := a[0].(*object.Instance)
			symsVal, ok := self.Dict.GetStr("_symbols")
			if !ok {
				return &object.List{V: nil}, nil
			}
			syms, err := iterate(ii, symsVal)
			if err != nil {
				return &object.List{V: nil}, nil
			}
			var names []object.Object
			for _, sym := range syms {
				if inst2, ok2 := sym.(*object.Instance); ok2 {
					if nm, ok3 := inst2.Dict.GetStr("_name"); ok3 {
						names = append(names, nm)
					}
				}
			}
			return &object.List{V: names}, nil
		},
	})

	stCls.Dict.SetStr("lookup", &object.BuiltinFunc{
		Name: "lookup",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "lookup() requires a name argument")
			}
			ii := interp.(*Interp)
			self := a[0].(*object.Instance)
			name := ""
			if s, ok := a[1].(*object.Str); ok {
				name = s.V
			}
			symsVal, ok := self.Dict.GetStr("_symbols")
			if !ok {
				return nil, object.Errorf(i.keyErr, "%q", name)
			}
			syms, err := iterate(ii, symsVal)
			if err != nil {
				return nil, object.Errorf(i.keyErr, "%q", name)
			}
			for _, sym := range syms {
				if inst2, ok2 := sym.(*object.Instance); ok2 {
					if nm, ok3 := inst2.Dict.GetStr("_name"); ok3 {
						if ns, ok4 := nm.(*object.Str); ok4 && ns.V == name {
							return inst2, nil
						}
					}
				}
			}
			return nil, object.Errorf(i.keyErr, "%q", name)
		},
	})

	m.Dict.SetStr("SymbolTable", stCls)

	// ── Function class (subclass of SymbolTable) ──────────────────────────

	fnCls := &object.Class{
		Name:  "Function",
		Dict:  object.NewDict(),
		Bases: []*object.Class{stCls},
	}
	fnCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			self.Dict.SetStr("_type", &object.Str{V: "function"})
			self.Dict.SetStr("_name", &object.Str{V: "<function>"})
			self.Dict.SetStr("_lineno", object.NewInt(0))
			self.Dict.SetStr("_id", object.NewInt(0))
			self.Dict.SetStr("_optimized", object.True)
			self.Dict.SetStr("_nested", object.False)
			self.Dict.SetStr("_symbols", &object.List{V: nil})
			self.Dict.SetStr("_children", &object.List{V: nil})
			self.Dict.SetStr("_parameters", &object.Tuple{V: nil})
			self.Dict.SetStr("_locals", &object.Tuple{V: nil})
			self.Dict.SetStr("_globals", &object.Tuple{V: nil})
			self.Dict.SetStr("_frees", &object.Tuple{V: nil})
			self.Dict.SetStr("_nonlocals", &object.Tuple{V: nil})
			return object.None, nil
		},
	})
	for _, spec := range []struct {
		method string
		field  string
	}{
		{"get_parameters", "_parameters"},
		{"get_locals", "_locals"},
		{"get_globals", "_globals"},
		{"get_frees", "_frees"},
		{"get_nonlocals", "_nonlocals"},
	} {
		meth := spec.method
		field := spec.field
		fnCls.Dict.SetStr(meth, &object.BuiltinFunc{
			Name: meth,
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) < 1 {
					return &object.Tuple{V: nil}, nil
				}
				self := a[0].(*object.Instance)
				if v, ok := self.Dict.GetStr(field); ok {
					return v, nil
				}
				return &object.Tuple{V: nil}, nil
			},
		})
	}
	m.Dict.SetStr("Function", fnCls)

	// ── Class class (subclass of SymbolTable) ─────────────────────────────

	clsCls := &object.Class{
		Name:  "Class",
		Dict:  object.NewDict(),
		Bases: []*object.Class{stCls},
	}
	clsCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			self.Dict.SetStr("_type", &object.Str{V: "class"})
			self.Dict.SetStr("_name", &object.Str{V: "<class>"})
			self.Dict.SetStr("_lineno", object.NewInt(0))
			self.Dict.SetStr("_id", object.NewInt(0))
			self.Dict.SetStr("_optimized", object.False)
			self.Dict.SetStr("_nested", object.False)
			self.Dict.SetStr("_symbols", &object.List{V: nil})
			self.Dict.SetStr("_children", &object.List{V: nil})
			self.Dict.SetStr("_methods", &object.Tuple{V: nil})
			return object.None, nil
		},
	})
	clsCls.Dict.SetStr("get_methods", &object.BuiltinFunc{
		Name: "get_methods",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Tuple{V: nil}, nil
			}
			self := a[0].(*object.Instance)
			if v, ok := self.Dict.GetStr("_methods"); ok {
				return v, nil
			}
			return &object.Tuple{V: nil}, nil
		},
	})
	m.Dict.SetStr("Class", clsCls)

	// ── symtable() function ───────────────────────────────────────────────

	m.Dict.SetStr("symtable", &object.BuiltinFunc{
		Name: "symtable",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := makeDefaultSymTab(stCls)
			return inst, nil
		},
	})

	// ── SymbolTableFactory ────────────────────────────────────────────────
	// Stub class so that importing the module doesn't break.

	stfCls := &object.Class{Name: "SymbolTableFactory", Dict: object.NewDict()}
	m.Dict.SetStr("SymbolTableFactory", stfCls)

	return m
}

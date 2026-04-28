package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildPyclbr constructs the pyclbr module with the CPython 3.14 API:
// _Object base class, Class and Function subclasses, readmodule/readmodule_ex.
func (i *Interp) buildPyclbr() *object.Module {
	m := &object.Module{Name: "pyclbr", Dict: object.NewDict()}

	m.Dict.SetStr("__all__", &object.List{V: []object.Object{
		&object.Str{V: "readmodule"},
		&object.Str{V: "readmodule_ex"},
		&object.Str{V: "Class"},
		&object.Str{V: "Function"},
	}})

	// ── _Object base class ────────────────────────────────────────────────

	objCls := &object.Class{Name: "_Object", Dict: object.NewDict()}

	// initObject sets all _Object attributes on self from positional args.
	// Signature: (self, module, name, file, lineno, end_lineno, parent)
	initObject := func(self *object.Instance, module, name, file, lineno, endLineno, parent object.Object) {
		if module == nil {
			module = object.None
		}
		if name == nil {
			name = object.None
		}
		if file == nil {
			file = object.None
		}
		if lineno == nil {
			lineno = object.NewInt(0)
		}
		if endLineno == nil {
			endLineno = object.None
		}
		if parent == nil {
			parent = object.None
		}
		self.Dict.SetStr("module", module)
		self.Dict.SetStr("name", name)
		self.Dict.SetStr("file", file)
		self.Dict.SetStr("lineno", lineno)
		self.Dict.SetStr("end_lineno", endLineno)
		self.Dict.SetStr("parent", parent)
		self.Dict.SetStr("children", object.NewDict())

		// If parent is not None, add self to parent.children[name]
		if parent != object.None {
			if parentInst, ok := parent.(*object.Instance); ok {
				if childrenObj, ok2 := parentInst.Dict.GetStr("children"); ok2 {
					if childDict, ok3 := childrenObj.(*object.Dict); ok3 {
						childDict.Set(name, self) //nolint
					}
				}
			}
		}
	}

	objCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		// _Object(module, name, file, lineno, end_lineno, parent)
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			get := func(idx int) object.Object {
				if idx < len(a) {
					return a[idx]
				}
				return nil
			}
			initObject(self, get(1), get(2), get(3), get(4), get(5), get(6))
			return object.None, nil
		},
	})

	objCls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "_Object()"}, nil
			}
			self, ok := a[0].(*object.Instance)
			if !ok {
				return &object.Str{V: "_Object()"}, nil
			}
			name, _ := self.Dict.GetStr("name")
			cls := self.Class.Name
			return &object.Str{V: cls + "(" + object.Str_(name) + ")"}, nil
		},
	})

	m.Dict.SetStr("_Object", objCls)

	// ── Class ─────────────────────────────────────────────────────────────

	clsCls := &object.Class{
		Name:  "Class",
		Dict:  object.NewDict(),
		Bases: []*object.Class{objCls},
	}

	clsCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		// Class(module, name, super_, file, lineno, parent=None, *, end_lineno=None)
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			get := func(idx int) object.Object {
				if idx < len(a) {
					return a[idx]
				}
				return nil
			}
			module := get(1)
			name := get(2)
			superArg := get(3) // super_ positional
			file := get(4)
			lineno := get(5)
			var parent object.Object = object.None
			if len(a) > 6 {
				parent = a[6]
			}
			var endLineno object.Object = object.None

			// Override from keyword args
			if kw != nil {
				if v, ok := kw.GetStr("parent"); ok {
					parent = v
				}
				if v, ok := kw.GetStr("end_lineno"); ok {
					endLineno = v
				}
			}

			initObject(self, module, name, file, lineno, endLineno, parent)

			// super: use the provided list, or empty list
			var superList []object.Object
			if superArg != nil && superArg != object.None {
				items, err := iterate(i, superArg)
				if err == nil {
					superList = items
				}
			}
			if superList == nil {
				superList = []object.Object{}
			}
			self.Dict.SetStr("super", &object.List{V: superList})
			self.Dict.SetStr("methods", object.NewDict())
			return object.None, nil
		},
	})

	m.Dict.SetStr("Class", clsCls)

	// ── Function ──────────────────────────────────────────────────────────

	fnCls := &object.Class{
		Name:  "Function",
		Dict:  object.NewDict(),
		Bases: []*object.Class{objCls},
	}

	fnCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		// Function(module, name, file, lineno, parent=None, is_async=False, *, end_lineno=None)
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			get := func(idx int) object.Object {
				if idx < len(a) {
					return a[idx]
				}
				return nil
			}
			module := get(1)
			name := get(2)
			file := get(3)
			lineno := get(4)
			var parent object.Object = object.None
			if len(a) > 5 {
				parent = a[5]
			}
			var isAsync object.Object = object.False
			if len(a) > 6 {
				isAsync = a[6]
			}
			var endLineno object.Object = object.None

			// Override from keyword args
			if kw != nil {
				if v, ok2 := kw.GetStr("parent"); ok2 {
					parent = v
				}
				if v, ok2 := kw.GetStr("is_async"); ok2 {
					isAsync = v
				}
				if v, ok2 := kw.GetStr("end_lineno"); ok2 {
					endLineno = v
				}
			}

			initObject(self, module, name, file, lineno, endLineno, parent)
			self.Dict.SetStr("is_async", isAsync)

			// If parent is a Class instance, register name→lineno in parent.methods
			if parent != object.None {
				if parentInst, ok2 := parent.(*object.Instance); ok2 {
					if object.IsSubclass(parentInst.Class, clsCls) {
						if methodsObj, ok3 := parentInst.Dict.GetStr("methods"); ok3 {
							if methodDict, ok4 := methodsObj.(*object.Dict); ok4 {
								methodDict.Set(name, lineno) //nolint
							}
						}
					}
				}
			}
			return object.None, nil
		},
	})

	m.Dict.SetStr("Function", fnCls)

	// ── readmodule(module, path=None) ─────────────────────────────────────
	// Returns a dict mapping class names to Class objects.
	// Stub: no real parser available, returns empty dict.

	m.Dict.SetStr("readmodule", &object.BuiltinFunc{
		Name: "readmodule",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "readmodule() missing module argument")
			}
			return object.NewDict(), nil
		},
	})

	// ── readmodule_ex(module, path=None) ──────────────────────────────────
	// Returns a dict mapping names to Class/Function objects.
	// Stub: returns empty dict.

	m.Dict.SetStr("readmodule_ex", &object.BuiltinFunc{
		Name: "readmodule_ex",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "readmodule_ex() missing module argument")
			}
			return object.NewDict(), nil
		},
	})

	return m
}

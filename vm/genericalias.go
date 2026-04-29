package vm

import (
	"strings"

	"github.com/tamnd/goipy/object"
)

// genericAliasCls is the class behind PEP 585 parameterised builtins
// (list[int], dict[str, int], …). Instances carry __origin__ and __args__.
var genericAliasCls *object.Class

// unionTypeCls is the class behind PEP 604 unions (int | str). Instances
// carry __args__ (a flattened, deduped tuple of members).
var unionTypeCls *object.Class

func (i *Interp) initGenericAliasClasses() {
	if genericAliasCls != nil {
		return
	}
	genericAliasCls = &object.Class{Name: "types.GenericAlias", Dict: object.NewDict()}
	unionTypeCls = &object.Class{Name: "types.UnionType", Dict: object.NewDict()}

	// __repr__ → "list[int]" or "dict[str, int]"
	genericAliasCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			origin, _ := inst.Dict.GetStr("__origin__")
			argsObj, _ := inst.Dict.GetStr("__args__")
			parts := []string{}
			if t, ok := argsObj.(*object.Tuple); ok {
				for _, v := range t.V {
					parts = append(parts, typeRepr(v))
				}
			}
			return &object.Str{V: typeRepr(origin) + "[" + strings.Join(parts, ", ") + "]"}, nil
		}})

	// __getitem__ → re-subscription, returns a new alias with the same origin.
	genericAliasCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			origin, _ := inst.Dict.GetStr("__origin__")
			key := a[1]
			var args []object.Object
			if t, ok := key.(*object.Tuple); ok {
				args = t.V
			} else {
				args = []object.Object{key}
			}
			return makeGenericAlias(origin, args), nil
		}})

	// __or__ / __ror__ → build a UnionType.
	genericAliasCls.Dict.SetStr("__or__", &object.BuiltinFunc{Name: "__or__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return makeUnionType([]object.Object{a[0], a[1]}), nil
		}})
	genericAliasCls.Dict.SetStr("__ror__", &object.BuiltinFunc{Name: "__ror__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return makeUnionType([]object.Object{a[1], a[0]}), nil
		}})

	// __mro_entries__(bases) → (origin,) so `class C(list[int])` inherits
	// from list. The base is dropped silently if origin isn't a *object.Class
	// (the existing __build_class__ does that filtering).
	genericAliasCls.Dict.SetStr("__mro_entries__", &object.BuiltinFunc{Name: "__mro_entries__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			origin, _ := inst.Dict.GetStr("__origin__")
			return &object.Tuple{V: []object.Object{origin}}, nil
		}})

	// __call__ → forward to origin so `list[int]([1,2])` builds a list.
	genericAliasCls.Dict.SetStr("__call__", &object.BuiltinFunc{Name: "__call__",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			origin, _ := inst.Dict.GetStr("__origin__")
			return ii.(*Interp).callObject(origin, a[1:], kw)
		}})

	// ── UnionType ────────────────────────────────────────────────────────
	unionTypeCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			argsObj, _ := inst.Dict.GetStr("__args__")
			parts := []string{}
			if t, ok := argsObj.(*object.Tuple); ok {
				for _, v := range t.V {
					parts = append(parts, typeRepr(v))
				}
			}
			return &object.Str{V: strings.Join(parts, " | ")}, nil
		}})
	unionTypeCls.Dict.SetStr("__or__", &object.BuiltinFunc{Name: "__or__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return makeUnionType([]object.Object{a[0], a[1]}), nil
		}})
	unionTypeCls.Dict.SetStr("__ror__", &object.BuiltinFunc{Name: "__ror__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return makeUnionType([]object.Object{a[1], a[0]}), nil
		}})
	unionTypeCls.Dict.SetStr("__instancecheck__", &object.BuiltinFunc{Name: "__instancecheck__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			argsObj, _ := inst.Dict.GetStr("__args__")
			t, _ := argsObj.(*object.Tuple)
			if t == nil {
				return object.BoolOf(false), nil
			}
			for _, m := range t.V {
				if isinstance(a[1], m) {
					return object.BoolOf(true), nil
				}
			}
			return object.BoolOf(false), nil
		}})
}

// makeGenericAlias builds a fresh GenericAlias instance (PEP 585).
func makeGenericAlias(origin object.Object, args []object.Object) *object.Instance {
	inst := &object.Instance{Class: genericAliasCls, Dict: object.NewDict()}
	inst.Dict.SetStr("__origin__", origin)
	inst.Dict.SetStr("__args__", &object.Tuple{V: append([]object.Object{}, args...)})
	return inst
}

// makeUnionType builds a UnionType instance (PEP 604), flattening any
// nested UnionType members and deduping by Python equality + identity.
func makeUnionType(args []object.Object) object.Object {
	flat := unionFlatten(args)
	flat = unionDedupe(flat)
	if len(flat) == 1 {
		return flat[0]
	}
	inst := &object.Instance{Class: unionTypeCls, Dict: object.NewDict()}
	inst.Dict.SetStr("__args__", &object.Tuple{V: flat})
	return inst
}

func unionFlatten(args []object.Object) []object.Object {
	out := make([]object.Object, 0, len(args))
	for _, a := range args {
		if inst, ok := a.(*object.Instance); ok && inst.Class == unionTypeCls {
			if t, ok := inst.Dict.GetStr("__args__"); ok {
				if tup, ok := t.(*object.Tuple); ok {
					out = append(out, tup.V...)
					continue
				}
			}
		}
		out = append(out, a)
	}
	return out
}

func unionDedupe(args []object.Object) []object.Object {
	out := make([]object.Object, 0, len(args))
	for _, a := range args {
		dup := false
		for _, b := range out {
			if a == b || typeRepr(a) == typeRepr(b) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, a)
		}
	}
	return out
}

// typeRepr returns a Python-shaped repr for an object that names a type:
// builtins/classes drop their <class '...'> wrapper.
func typeRepr(o object.Object) string {
	switch v := o.(type) {
	case *object.Class:
		return v.Name
	case *object.BuiltinFunc:
		return v.Name
	case *object.NoneType:
		return "None"
	}
	return object.Repr(o)
}

// isUnionType reports whether o is a UnionType instance.
func isUnionType(o object.Object) bool {
	inst, ok := o.(*object.Instance)
	return ok && inst.Class == unionTypeCls
}

// lookupMROEntries returns o.__mro_entries__ if defined (callable that
// takes the bases tuple and returns the substituted bases). Used by
// __build_class__ to implement PEP 560.
func lookupMROEntries(o object.Object) object.Object {
	switch v := o.(type) {
	case *object.Instance:
		if fn, ok := classLookup(v.Class, "__mro_entries__"); ok {
			return fn
		}
		if fn, ok := v.Dict.GetStr("__mro_entries__"); ok {
			return fn
		}
	case *object.Class:
		if fn, ok := classLookup(v, "__mro_entries__"); ok {
			return fn
		}
	case *object.BuiltinFunc:
		if v.Attrs != nil {
			if fn, ok := v.Attrs.GetStr("__mro_entries__"); ok {
				return fn
			}
		}
	}
	return nil
}

// isTypeLike reports whether o can stand in for a "type" inside a PEP 604
// union: a Class, a builtin type-constructor BuiltinFunc, an existing
// GenericAlias/UnionType, or NoneType (so `int | None` works).
func isTypeLike(o object.Object) bool {
	switch v := o.(type) {
	case *object.Class:
		return true
	case *object.BuiltinFunc:
		if v.Attrs == nil {
			return false
		}
		_, ok := v.Attrs.GetStr("__class_getitem__")
		return ok
	case *object.NoneType:
		return true
	case *object.Instance:
		return v.Class == unionTypeCls || v.Class == genericAliasCls
	}
	return false
}

// isGenericAlias reports whether o is a GenericAlias instance.
func isGenericAlias(o object.Object) bool {
	inst, ok := o.(*object.Instance)
	return ok && inst.Class == genericAliasCls
}

// installClassUnionHooks adds __or__/__ror__ to a Class so user types
// (and the `object` builtin) can participate in PEP 604 unions.
// __class_getitem__ is intentionally NOT installed here: many user
// classes already define their own (typing.Generic, etc.) and overriding
// would break them.
func installClassUnionHooks(cls *object.Class) {
	if _, ok := cls.Dict.GetStr("__or__"); !ok {
		cls.Dict.SetStr("__or__", &object.BuiltinFunc{Name: "__or__",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				return makeUnionType([]object.Object{a[0], a[1]}), nil
			}})
	}
	if _, ok := cls.Dict.GetStr("__ror__"); !ok {
		cls.Dict.SetStr("__ror__", &object.BuiltinFunc{Name: "__ror__",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				return makeUnionType([]object.Object{a[1], a[0]}), nil
			}})
	}
}

// installTypeSubscriptHooks attaches __class_getitem__ and __or__/__ror__
// to a type-like BuiltinFunc so list/dict/tuple/set/frozenset/type
// participate in PEP 585 / 604.
func installTypeSubscriptHooks(bf *object.BuiltinFunc) {
	if bf.Attrs == nil {
		bf.Attrs = object.NewDict()
	}
	bf.Attrs.SetStr("__class_getitem__", &object.BuiltinFunc{Name: "__class_getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			key := a[0]
			var args []object.Object
			if t, ok := key.(*object.Tuple); ok {
				args = t.V
			} else {
				args = []object.Object{key}
			}
			return makeGenericAlias(bf, args), nil
		}})
	bf.Attrs.SetStr("__or__", &object.BuiltinFunc{Name: "__or__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return makeUnionType([]object.Object{bf, a[0]}), nil
		}})
	bf.Attrs.SetStr("__ror__", &object.BuiltinFunc{Name: "__ror__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return makeUnionType([]object.Object{a[0], bf}), nil
		}})
}

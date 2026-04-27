package vm

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/tamnd/goipy/object"
)

// absCacheToken is a global counter incremented on each ABC.register() call.
var absCacheToken int64

// buildAbc creates the abc module.
func (i *Interp) buildAbc() *object.Module {
	m := &object.Module{Name: "abc", Dict: object.NewDict()}

	// ── abstractmethod ────────────────────────────────────────────────────────
	m.Dict.SetStr("abstractmethod", &object.BuiltinFunc{Name: "abstractmethod",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "abstractmethod() requires a function argument")
			}
			fn := a[0]
			abcSetAbstract(fn)
			return fn, nil
		}})

	// ── abstractclassmethod / abstractstaticmethod / abstractproperty (stubs) ─
	makeAbstractWrapper := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name,
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "%s() requires a function argument", name)
				}
				fn := a[0]
				abcSetAbstract(fn)
				return fn, nil
			}}
	}
	m.Dict.SetStr("abstractclassmethod", makeAbstractWrapper("abstractclassmethod"))
	m.Dict.SetStr("abstractstaticmethod", makeAbstractWrapper("abstractstaticmethod"))
	m.Dict.SetStr("abstractproperty", makeAbstractWrapper("abstractproperty"))

	// ── get_cache_token ───────────────────────────────────────────────────────
	m.Dict.SetStr("get_cache_token", &object.BuiltinFunc{Name: "get_cache_token",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(atomic.LoadInt64(&absCacheToken)), nil
		}})

	// ── ABCMeta ───────────────────────────────────────────────────────────────
	abcMetaCls := i.buildABCMeta()
	m.Dict.SetStr("ABCMeta", abcMetaCls)

	// ── ABC base class ────────────────────────────────────────────────────────
	abcCls := &object.Class{Name: "ABC", Dict: object.NewDict()}
	abcCls.Dict.SetStr("__abstractmethods__", object.NewFrozenset())

	// register is a classmethod: a[0] is the class (ABC subclass), a[1] is the subclass to register.
	registerFn := &object.BuiltinFunc{Name: "register",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "register() requires a class argument")
			}
			subcls := a[1]
			if cls, ok := a[0].(*object.Class); ok {
				if c2, ok2 := subcls.(*object.Class); ok2 {
					abcAddRegistered(cls, c2)
					atomic.AddInt64(&absCacheToken, 1)
				}
			}
			return subcls, nil
		}}
	abcCls.Dict.SetStr("register", &object.ClassMethod{Fn: registerFn})

	// __subclasshook__ is also a classmethod.
	abcCls.Dict.SetStr("__subclasshook__", &object.ClassMethod{Fn: &object.BuiltinFunc{Name: "__subclasshook__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NotImplemented, nil
		}}})

	m.Dict.SetStr("ABC", abcCls)

	return m
}

// buildABCMeta creates an ABCMeta class.
func (i *Interp) buildABCMeta() *object.Class {
	cls := &object.Class{Name: "ABCMeta", Dict: object.NewDict()}

	registerMetaFn := &object.BuiltinFunc{Name: "register",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "register() requires a class argument")
			}
			subcls := a[1]
			if cls2, ok := a[0].(*object.Class); ok {
				if c3, ok2 := subcls.(*object.Class); ok2 {
					abcAddRegistered(cls2, c3)
					atomic.AddInt64(&absCacheToken, 1)
				}
			}
			return subcls, nil
		}}
	cls.Dict.SetStr("register", &object.ClassMethod{Fn: registerMetaFn})

	cls.Dict.SetStr("__subclasshook__", &object.ClassMethod{Fn: &object.BuiltinFunc{Name: "__subclasshook__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NotImplemented, nil
		}}})

	cls.Dict.SetStr("__instancecheck__", &object.BuiltinFunc{Name: "__instancecheck__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.False, nil
			}
			abcClass := a[0]
			inst := a[1]
			if c, ok := abcClass.(*object.Class); ok {
				if c.ABCCheck != nil && c.ABCCheck(inst) {
					return object.True, nil
				}
			}
			return object.False, nil
		}})

	cls.Dict.SetStr("__subclasscheck__", &object.BuiltinFunc{Name: "__subclasscheck__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.False, nil
			}
			abcClass := a[0]
			sub := a[1]
			if c, ok := abcClass.(*object.Class); ok {
				if c2, ok2 := sub.(*object.Class); ok2 {
					if object.IsSubclass(c2, c) {
						return object.True, nil
					}
					if regs, ok3 := abcGetRegistered(c); ok3 {
						for _, r := range regs {
							if object.IsSubclass(c2, r) {
								return object.True, nil
							}
						}
					}
				}
			}
			return object.False, nil
		}})

	cls.Dict.SetStr("_dump_registry", &object.BuiltinFunc{Name: "_dump_registry",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 {
				if c, ok := a[0].(*object.Class); ok {
					if regs, ok2 := abcGetRegistered(c); ok2 {
						for _, r := range regs {
							fmt.Printf("  %s\n", r.Name)
						}
					}
				}
			}
			return object.None, nil
		}})

	return cls
}

// abcSetAbstract marks an object as abstract by setting __isabstractmethod__ = True.
func abcSetAbstract(fn object.Object) {
	switch v := fn.(type) {
	case *object.Function:
		if v.Dict == nil {
			v.Dict = object.NewDict()
		}
		v.Dict.SetStr("__isabstractmethod__", object.True)
	case *object.BuiltinFunc:
		if v.Attrs == nil {
			v.Attrs = object.NewDict()
		}
		v.Attrs.SetStr("__isabstractmethod__", object.True)
	case *object.Instance:
		v.Dict.SetStr("__isabstractmethod__", object.True)
	}
}

// abcIsAbstract returns true if fn has __isabstractmethod__ == True.
func abcIsAbstract(fn object.Object) bool {
	var d *object.Dict
	switch v := fn.(type) {
	case *object.Function:
		d = v.Dict
	case *object.BuiltinFunc:
		d = v.Attrs
	case *object.Instance:
		d = v.Dict
	default:
		return false
	}
	if d == nil {
		return false
	}
	val, ok := d.GetStr("__isabstractmethod__")
	if !ok {
		return false
	}
	return isTruthy(val)
}

// abcRegisteredKey is used in class dict to store registered virtual subclasses.
const abcRegisteredKey = "__abc_registered__"

// abcAddRegistered adds c2 as a virtual subclass of cls.
func abcAddRegistered(cls *object.Class, c2 *object.Class) {
	var list []*object.Class
	if existing, ok := cls.Dict.GetStr(abcRegisteredKey); ok {
		if l, ok2 := existing.(*object.List); ok2 {
			for _, v := range l.V {
				if c, ok3 := v.(*object.Class); ok3 {
					list = append(list, c)
				}
			}
		}
	}
	list = append(list, c2)
	objs := make([]object.Object, len(list))
	for j, c := range list {
		objs[j] = c
	}
	cls.Dict.SetStr(abcRegisteredKey, &object.List{V: objs})

	// Wire ABCCheck so isinstance works.
	prevCheck := cls.ABCCheck
	capturedList := list
	cls.ABCCheck = func(o object.Object) bool {
		if inst, ok := o.(*object.Instance); ok {
			for _, r := range capturedList {
				if object.IsSubclass(inst.Class, r) {
					return true
				}
			}
		}
		if prevCheck != nil {
			return prevCheck(o)
		}
		return false
	}
}

// abcGetRegistered returns the virtually registered subclasses for cls.
func abcGetRegistered(cls *object.Class) ([]*object.Class, bool) {
	existing, ok := cls.Dict.GetStr(abcRegisteredKey)
	if !ok {
		return nil, false
	}
	l, ok2 := existing.(*object.List)
	if !ok2 {
		return nil, false
	}
	var out []*object.Class
	for _, v := range l.V {
		if c, ok3 := v.(*object.Class); ok3 {
			out = append(out, c)
		}
	}
	return out, true
}

// abcCollectAbstractMethods returns the names of unimplemented abstract methods on cls.
func abcCollectAbstractMethods(cls *object.Class) []string {
	abstract := map[string]bool{}
	concrete := map[string]bool{}

	var walk func(*object.Class)
	walk = func(c *object.Class) {
		if c == nil {
			return
		}
		keys, vals := c.Dict.Items()
		for idx, k := range keys {
			name, ok := k.(*object.Str)
			if !ok {
				continue
			}
			v := vals[idx]
			if abcIsAbstract(v) {
				if !concrete[name.V] {
					abstract[name.V] = true
				}
			} else {
				concrete[name.V] = true
				delete(abstract, name.V)
			}
		}
		for _, base := range c.Bases {
			walk(base)
		}
	}
	walk(cls)

	var names []string
	for name := range abstract {
		names = append(names, name)
	}
	return names
}

// checkAbstractMethods raises TypeError if cls has unimplemented abstract methods.
func checkAbstractMethods(i *Interp, cls *object.Class) error {
	// Check __abstractmethods__ frozenset/set if explicitly set.
	if v, ok := cls.Dict.GetStr("__abstractmethods__"); ok {
		var names []string
		switch fs := v.(type) {
		case *object.Frozenset:
			for _, item := range fs.Items() {
				if s, ok2 := item.(*object.Str); ok2 {
					names = append(names, s.V)
				}
			}
		case *object.Set:
			for _, item := range fs.Items() {
				if s, ok2 := item.(*object.Str); ok2 {
					names = append(names, s.V)
				}
			}
		}
		if len(names) > 0 {
			return object.Errorf(i.typeErr,
				"Can't instantiate abstract class %s without an implementation for abstract method%s %s",
				cls.Name, pluralS(len(names)), quotedList(names))
		}
		return nil
	}

	// Fallback: scan class MRO for abstract methods.
	names := abcCollectAbstractMethods(cls)
	if len(names) > 0 {
		return object.Errorf(i.typeErr,
			"Can't instantiate abstract class %s without an implementation for abstract method%s %s",
			cls.Name, pluralS(len(names)), quotedList(names))
	}
	return nil
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func quotedList(names []string) string {
	quoted := make([]string, len(names))
	for j, n := range names {
		quoted[j] = "'" + n + "'"
	}
	return strings.Join(quoted, ", ")
}

package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildCollectionsAbc constructs the collections.abc module with all standard
// Abstract Base Classes. isinstance(obj, ABC) works via ABCCheck duck-typing;
// ABC.register(cls) adds virtual subclasses.
func (i *Interp) buildCollectionsAbc() *object.Module {
	m := &object.Module{Name: "collections.abc", Dict: object.NewDict()}

	abc := func(name string, check func(o object.Object) bool) *object.Class {
		// Each ABC keeps a set of registered virtual subclasses.
		registered := []*object.Class{}

		cls := &object.Class{Name: name, Dict: object.NewDict()}
		cls.ABCCheck = func(o object.Object) bool {
			// Virtual subclass check.
			if inst, ok := o.(*object.Instance); ok {
				for _, r := range registered {
					if object.IsSubclass(inst.Class, r) {
						return true
					}
				}
			}
			return check(o)
		}

		cls.Dict.SetStr("register", &object.BuiltinFunc{Name: "register", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "register() takes 1 argument")
			}
			if c, ok := a[0].(*object.Class); ok {
				registered = append(registered, c)
			}
			return a[0], nil
		}})

		cls.Dict.SetStr("__instancecheck__", &object.BuiltinFunc{Name: "__instancecheck__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			return object.BoolOf(cls.ABCCheck(a[0])), nil
		}})

		cls.Dict.SetStr("__subclasscheck__", &object.BuiltinFunc{Name: "__subclasscheck__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			if c, ok := a[0].(*object.Class); ok {
				for _, r := range registered {
					if object.IsSubclass(c, r) {
						return object.True, nil
					}
				}
			}
			return object.False, nil
		}})

		m.Dict.SetStr(name, cls)
		return cls
	}

	// --- helper: does an Instance have a named method? ---
	instHas := func(o object.Object, names ...string) bool {
		inst, ok := o.(*object.Instance)
		if !ok {
			return false
		}
		for _, name := range names {
			if _, ok := inst.Dict.GetStr(name); ok {
				continue
			}
			if inst.Class != nil {
				if _, ok := classLookup(inst.Class, name); ok {
					continue
				}
			}
			return false
		}
		return true
	}

	// ---- Hashable ----
	abc("Hashable", func(o object.Object) bool {
		switch o.(type) {
		case *object.Int, *object.Bool, *object.Float, *object.Str,
			*object.Bytes, *object.Tuple, *object.Frozenset,
			*object.NoneType, *object.Complex:
			return true
		}
		return instHas(o, "__hash__")
	})

	// ---- Callable ----
	abc("Callable", func(o object.Object) bool {
		switch o.(type) {
		case *object.BuiltinFunc, *object.Function, *object.BoundMethod, *object.Class:
			return true
		}
		return instHas(o, "__call__")
	})

	// ---- Iterable ----
	abc("Iterable", func(o object.Object) bool {
		switch o.(type) {
		case *object.List, *object.Tuple, *object.Str, *object.Dict,
			*object.Set, *object.Frozenset, *object.Bytes, *object.Bytearray,
			*object.Range, *object.Iter, *object.Generator,
			*object.Deque, *object.Counter, *object.DefaultDict, *object.OrderedDict:
			return true
		}
		return instHas(o, "__iter__")
	})

	// ---- Iterator ----
	abc("Iterator", func(o object.Object) bool {
		switch o.(type) {
		case *object.Iter, *object.Generator:
			return true
		}
		return instHas(o, "__next__", "__iter__")
	})

	// ---- Reversible ----
	abc("Reversible", func(o object.Object) bool {
		switch o.(type) {
		case *object.List, *object.Tuple, *object.Str,
			*object.Bytes, *object.Bytearray, *object.Range, *object.Deque,
			*object.Dict, *object.Counter, *object.DefaultDict, *object.OrderedDict:
			return true
		}
		return instHas(o, "__reversed__")
	})

	// ---- Generator ----
	abc("Generator", func(o object.Object) bool {
		if _, ok := o.(*object.Generator); ok {
			return true
		}
		return instHas(o, "send", "throw", "__next__", "__iter__")
	})

	// ---- Sized ----
	abc("Sized", func(o object.Object) bool {
		switch o.(type) {
		case *object.List, *object.Tuple, *object.Str, *object.Dict,
			*object.Set, *object.Frozenset, *object.Bytes, *object.Bytearray,
			*object.Range, *object.Deque, *object.Counter, *object.DefaultDict, *object.OrderedDict:
			return true
		}
		return instHas(o, "__len__")
	})

	// ---- Container ----
	abc("Container", func(o object.Object) bool {
		switch o.(type) {
		case *object.List, *object.Tuple, *object.Str, *object.Dict,
			*object.Set, *object.Frozenset, *object.Bytes, *object.Bytearray,
			*object.Deque, *object.Counter, *object.DefaultDict, *object.OrderedDict:
			return true
		}
		return instHas(o, "__contains__")
	})

	// ---- Collection ----
	abc("Collection", func(o object.Object) bool {
		switch o.(type) {
		case *object.List, *object.Tuple, *object.Str, *object.Dict,
			*object.Set, *object.Frozenset, *object.Bytes, *object.Bytearray,
			*object.Deque, *object.Counter, *object.DefaultDict, *object.OrderedDict:
			return true
		}
		return instHas(o, "__contains__", "__iter__", "__len__")
	})

	// ---- Sequence ----
	abc("Sequence", func(o object.Object) bool {
		switch o.(type) {
		case *object.List, *object.Tuple, *object.Str,
			*object.Bytes, *object.Bytearray, *object.Range:
			return true
		}
		return false
	})

	// ---- MutableSequence ----
	abc("MutableSequence", func(o object.Object) bool {
		switch o.(type) {
		case *object.List, *object.Bytearray:
			return true
		}
		return false
	})

	// ---- Set ----
	abc("Set", func(o object.Object) bool {
		switch o.(type) {
		case *object.Set, *object.Frozenset:
			return true
		}
		return false
	})

	// ---- MutableSet ----
	abc("MutableSet", func(o object.Object) bool {
		if _, ok := o.(*object.Set); ok {
			return true
		}
		return false
	})

	// ---- Mapping ----
	abc("Mapping", func(o object.Object) bool {
		switch o.(type) {
		case *object.Dict, *object.Counter, *object.DefaultDict, *object.OrderedDict:
			return true
		}
		return false
	})

	// ---- MutableMapping ----
	abc("MutableMapping", func(o object.Object) bool {
		switch o.(type) {
		case *object.Dict, *object.Counter, *object.DefaultDict, *object.OrderedDict:
			return true
		}
		return false
	})

	// ---- MappingView ----
	abc("MappingView", func(o object.Object) bool {
		return false
	})

	// ---- KeysView ----
	abc("KeysView", func(o object.Object) bool {
		return false
	})

	// ---- ItemsView ----
	abc("ItemsView", func(o object.Object) bool {
		return false
	})

	// ---- ValuesView ----
	abc("ValuesView", func(o object.Object) bool {
		return false
	})

	// ---- Awaitable ----
	abc("Awaitable", func(o object.Object) bool {
		return instHas(o, "__await__")
	})

	// ---- Coroutine ----
	abc("Coroutine", func(o object.Object) bool {
		return false
	})

	// ---- AsyncIterable ----
	abc("AsyncIterable", func(o object.Object) bool {
		return instHas(o, "__aiter__")
	})

	// ---- AsyncIterator ----
	abc("AsyncIterator", func(o object.Object) bool {
		return false
	})

	// ---- AsyncGenerator ----
	abc("AsyncGenerator", func(o object.Object) bool {
		return false
	})

	// ---- Buffer (Python 3.12+) ----
	abc("Buffer", func(o object.Object) bool {
		switch o.(type) {
		case *object.Bytes, *object.Bytearray, *object.Memoryview:
			return true
		}
		return instHas(o, "__buffer__")
	})

	return m
}

package vm

import (
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildTyping() *object.Module {
	m := &object.Module{Name: "typing", Dict: object.NewDict()}

	// ── _GenericAlias ─────────────────────────────────────────────────────────
	// Internal class for parameterised type hints: List[int], Dict[str,int], …
	genericAliasCls := &object.Class{Name: "_GenericAlias", Dict: object.NewDict()}
	makeAlias := func(origin object.Object, args []object.Object) *object.Instance {
		inst := &object.Instance{Class: genericAliasCls, Dict: object.NewDict()}
		inst.Dict.SetStr("__origin__", origin)
		inst.Dict.SetStr("__args__", &object.Tuple{V: append([]object.Object{}, args...)})
		return inst
	}
	// Allow further subscription: Mapping[str, int][str, float] etc.
	genericAliasCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			origin, _ := inst.Dict.GetStr("__origin__")
			key := a[1]
			var newArgs []object.Object
			if t, ok := key.(*object.Tuple); ok {
				newArgs = t.V
			} else {
				newArgs = []object.Object{key}
			}
			return makeAlias(origin, newArgs), nil
		}})
	genericAliasCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			origin, _ := inst.Dict.GetStr("__origin__")
			argsObj, _ := inst.Dict.GetStr("__args__")
			var originName string
			switch o := origin.(type) {
			case *object.Class:
				originName = "typing." + o.Name
			case *object.BuiltinFunc:
				originName = "typing." + o.Name
			default:
				originName = object.Repr(origin)
			}
			if args, ok := argsObj.(*object.Tuple); ok && len(args.V) > 0 {
				parts := make([]string, len(args.V))
				for k, v := range args.V {
					parts[k] = object.Repr(v)
				}
				return &object.Str{V: originName + "[" + strings.Join(parts, ", ") + "]"}, nil
			}
			return &object.Str{V: originName}, nil
		}})

	// ── makeSpecialAlias ──────────────────────────────────────────────────────
	// Creates a subscriptable alias for List, Dict, etc. whose __origin__ is a
	// builtin or class object.
	makeSpecialAlias := func(name string, origin object.Object) *object.Instance {
		cls := &object.Class{Name: name, Dict: object.NewDict()}
		inst := &object.Instance{Class: cls, Dict: object.NewDict()}
		inst.Dict.SetStr("__origin__", origin)
		cls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				key := a[1]
				var args []object.Object
				if t, ok := key.(*object.Tuple); ok {
					args = t.V
				} else {
					args = []object.Object{key}
				}
				return makeAlias(origin, args), nil
			}})
		return inst
	}

	// ── TYPE_CHECKING ─────────────────────────────────────────────────────────
	m.Dict.SetStr("TYPE_CHECKING", object.BoolOf(false))

	// ── Any ──────────────────────────────────────────────────────────────────
	anyCls := &object.Class{Name: "_AnyMeta", Dict: object.NewDict()}
	anyCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "typing.Any"}, nil
		}})
	anyInst := &object.Instance{Class: anyCls, Dict: object.NewDict()}
	m.Dict.SetStr("Any", anyInst)

	// ── Union ─────────────────────────────────────────────────────────────────
	// Union is a *object.Class so `get_origin(x) is Union` works via identity.
	unionCls := &object.Class{Name: "Union", Dict: object.NewDict()}
	unionCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "<class 'typing.Union'>"}, nil
		}})
	unionCls.Dict.SetStr("__class_getitem__", &object.BuiltinFunc{Name: "__class_getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			key := a[1]
			var args []object.Object
			if t, ok := key.(*object.Tuple); ok {
				args = t.V
			} else {
				args = []object.Object{key}
			}
			return makeAlias(unionCls, args), nil
		}})
	m.Dict.SetStr("Union", unionCls)

	// ── Optional ─────────────────────────────────────────────────────────────
	// Optional[T] = Union[T, type(None)]
	optionalCls := &object.Class{Name: "Optional", Dict: object.NewDict()}
	optionalCls.Dict.SetStr("__class_getitem__", &object.BuiltinFunc{Name: "__class_getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			t := a[1]
			noneType, _ := i.Builtins.GetStr("type")
			_ = noneType
			// type(None) — we use object.None and look up its class indirectly
			// by creating a sentinel *object.Class named "NoneType".
			noneTypeCls := &object.Class{Name: "NoneType", Dict: object.NewDict()}
			noneTypeCls.ABCCheck = func(o object.Object) bool {
				_, ok := o.(*object.NoneType)
				return ok
			}
			return makeAlias(unionCls, []object.Object{t, noneTypeCls}), nil
		}})
	m.Dict.SetStr("Optional", optionalCls)

	// ── Generic ───────────────────────────────────────────────────────────────
	// Used as base class: class Stack(Generic[T]). The parameterised form
	// Generic[T] is a *object.Instance which goipy's __build_class__ ignores
	// as a base (only *object.Class bases are kept).
	genericCls := &object.Class{Name: "Generic", Dict: object.NewDict()}
	genericCls.Dict.SetStr("__class_getitem__", &object.BuiltinFunc{Name: "__class_getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			key := a[1]
			var args []object.Object
			if t, ok := key.(*object.Tuple); ok {
				args = t.V
			} else {
				args = []object.Object{key}
			}
			return makeAlias(genericCls, args), nil
		}})
	genericCls.Dict.SetStr("__init_subclass__", &object.BuiltinFunc{Name: "__init_subclass__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	m.Dict.SetStr("Generic", genericCls)

	// ── Protocol ─────────────────────────────────────────────────────────────
	protocolCls := &object.Class{Name: "Protocol", Dict: object.NewDict()}
	protocolCls.Dict.SetStr("__class_getitem__", &object.BuiltinFunc{Name: "__class_getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			key := a[1]
			var args []object.Object
			if t, ok := key.(*object.Tuple); ok {
				args = t.V
			} else {
				args = []object.Object{key}
			}
			return makeAlias(protocolCls, args), nil
		}})
	protocolCls.Dict.SetStr("__init_subclass__", &object.BuiltinFunc{Name: "__init_subclass__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	m.Dict.SetStr("Protocol", protocolCls)

	// ── runtime_checkable ────────────────────────────────────────────────────
	m.Dict.SetStr("runtime_checkable", &object.BuiltinFunc{Name: "runtime_checkable",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			cls, ok := a[0].(*object.Class)
			if !ok {
				return a[0], nil
			}
			// Collect all non-dunder methods defined directly in this class.
			var methodNames []string
			keys, _ := cls.Dict.Items()
			for _, k := range keys {
				s, ok := k.(*object.Str)
				if !ok {
					continue
				}
				if strings.HasPrefix(s.V, "__") && strings.HasSuffix(s.V, "__") {
					continue
				}
				methodNames = append(methodNames, s.V)
			}
			names := methodNames
			cls.ABCCheck = func(o object.Object) bool {
				inst, ok := o.(*object.Instance)
				if !ok {
					return false
				}
				for _, m := range names {
					if _, ok := classLookup(inst.Class, m); !ok {
						return false
					}
				}
				return true
			}
			return cls, nil
		}})

	// ── TypeVar ───────────────────────────────────────────────────────────────
	typeVarCls := &object.Class{Name: "TypeVar", Dict: object.NewDict()}
	typeVarCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "TypeVar() requires a name")
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return nil, object.Errorf(i.typeErr, "TypeVar self must be instance")
			}
			name, ok2 := a[1].(*object.Str)
			if !ok2 {
				return nil, object.Errorf(i.typeErr, "TypeVar name must be a string")
			}
			constraints := make([]object.Object, 0)
			for _, c := range a[2:] {
				constraints = append(constraints, c)
			}
			var bound object.Object = object.None
			var covariant, contravariant object.Object = object.BoolOf(false), object.BoolOf(false)
			if kw != nil {
				if v, ok := kw.GetStr("bound"); ok {
					bound = v
				}
				if v, ok := kw.GetStr("covariant"); ok {
					covariant = v
				}
				if v, ok := kw.GetStr("contravariant"); ok {
					contravariant = v
				}
			}
			inst.Dict.SetStr("__name__", name)
			inst.Dict.SetStr("__constraints__", &object.Tuple{V: constraints})
			inst.Dict.SetStr("__bound__", bound)
			inst.Dict.SetStr("__covariant__", covariant)
			inst.Dict.SetStr("__contravariant__", contravariant)
			return object.None, nil
		}})
	typeVarCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			name, _ := inst.Dict.GetStr("__name__")
			if s, ok := name.(*object.Str); ok {
				return &object.Str{V: "~" + s.V}, nil
			}
			return &object.Str{V: "~T"}, nil
		}})
	m.Dict.SetStr("TypeVar", typeVarCls)

	// ── Subscriptable type aliases ────────────────────────────────────────────
	// Each one wraps its corresponding builtin or a placeholder class.
	listBuiltin, _ := i.Builtins.GetStr("list")
	dictBuiltin, _ := i.Builtins.GetStr("dict")
	tupleBuiltin, _ := i.Builtins.GetStr("tuple")
	setBuiltin, _ := i.Builtins.GetStr("set")
	frozensetBuiltin, _ := i.Builtins.GetStr("frozenset")

	m.Dict.SetStr("List", makeSpecialAlias("List", listBuiltin))
	m.Dict.SetStr("Dict", makeSpecialAlias("Dict", dictBuiltin))
	m.Dict.SetStr("Tuple", makeSpecialAlias("Tuple", tupleBuiltin))
	m.Dict.SetStr("Set", makeSpecialAlias("Set", setBuiltin))
	m.Dict.SetStr("FrozenSet", makeSpecialAlias("FrozenSet", frozensetBuiltin))

	// ABC-based aliases use placeholder classes.
	sequenceCls := &object.Class{Name: "Sequence", Dict: object.NewDict()}
	mappingCls := &object.Class{Name: "Mapping", Dict: object.NewDict()}
	iterableCls := &object.Class{Name: "Iterable", Dict: object.NewDict()}
	iteratorCls := &object.Class{Name: "Iterator", Dict: object.NewDict()}
	callableCls := &object.Class{Name: "Callable", Dict: object.NewDict()}
	typeCls := &object.Class{Name: "Type", Dict: object.NewDict()}
	m.Dict.SetStr("Sequence", makeSpecialAlias("Sequence", sequenceCls))
	m.Dict.SetStr("Mapping", makeSpecialAlias("Mapping", mappingCls))
	m.Dict.SetStr("Iterable", makeSpecialAlias("Iterable", iterableCls))
	m.Dict.SetStr("Iterator", makeSpecialAlias("Iterator", iteratorCls))
	m.Dict.SetStr("Callable", makeSpecialAlias("Callable", callableCls))
	m.Dict.SetStr("Type", makeSpecialAlias("Type", typeCls))

	// Other common aliases
	classVarCls := &object.Class{Name: "ClassVar", Dict: object.NewDict()}
	finalFormCls := &object.Class{Name: "Final", Dict: object.NewDict()}
	literalCls := &object.Class{Name: "Literal", Dict: object.NewDict()}
	annotatedCls := &object.Class{Name: "Annotated", Dict: object.NewDict()}
	m.Dict.SetStr("ClassVar", makeSpecialAlias("ClassVar", classVarCls))
	m.Dict.SetStr("Final", makeSpecialAlias("Final", finalFormCls))
	m.Dict.SetStr("Literal", makeSpecialAlias("Literal", literalCls))
	m.Dict.SetStr("Annotated", makeSpecialAlias("Annotated", annotatedCls))

	// Collection aliases from collections.abc
	for _, name := range []string{
		"Deque", "DefaultDict", "OrderedDict", "Counter", "ChainMap",
		"Generator", "AsyncGenerator", "Coroutine",
		"AsyncIterable", "AsyncIterator", "AsyncContextManager",
		"ContextManager", "MutableMapping", "MutableSequence", "MutableSet",
		"AbstractSet", "Collection", "Reversible", "Container",
		"ItemsView", "KeysView", "ValuesView", "MappingView",
		"Awaitable", "Sized", "Hashable",
	} {
		nameCopy := name
		ph := &object.Class{Name: nameCopy, Dict: object.NewDict()}
		m.Dict.SetStr(nameCopy, makeSpecialAlias(nameCopy, ph))
	}

	// ── get_origin / get_args ─────────────────────────────────────────────────
	m.Dict.SetStr("get_origin", &object.BuiltinFunc{Name: "get_origin",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if inst, ok := a[0].(*object.Instance); ok {
				if inst.Class == genericAliasCls {
					if v, ok := inst.Dict.GetStr("__origin__"); ok {
						return v, nil
					}
				}
			}
			return object.None, nil
		}})
	m.Dict.SetStr("get_args", &object.BuiltinFunc{Name: "get_args",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if inst, ok := a[0].(*object.Instance); ok {
				if inst.Class == genericAliasCls {
					if v, ok := inst.Dict.GetStr("__args__"); ok {
						return v, nil
					}
				}
			}
			return &object.Tuple{V: nil}, nil
		}})

	// ── Utility decorators ────────────────────────────────────────────────────
	m.Dict.SetStr("cast", &object.BuiltinFunc{Name: "cast",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			return a[1], nil
		}})
	m.Dict.SetStr("overload", &object.BuiltinFunc{Name: "overload",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return a[0], nil
		}})
	m.Dict.SetStr("final", &object.BuiltinFunc{Name: "final",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			fn := a[0]
			switch f := fn.(type) {
			case *object.Function:
				if f.Dict == nil {
					f.Dict = object.NewDict()
				}
				f.Dict.SetStr("__final__", object.BoolOf(true))
			case *object.Class:
				f.Dict.SetStr("__final__", object.BoolOf(true))
			case *object.Instance:
				f.Dict.SetStr("__final__", object.BoolOf(true))
			}
			return fn, nil
		}})
	m.Dict.SetStr("no_type_check", &object.BuiltinFunc{Name: "no_type_check",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return a[0], nil
		}})
	m.Dict.SetStr("no_type_check_decorator", &object.BuiltinFunc{Name: "no_type_check_decorator",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return a[0], nil
		}})

	// ── NewType ───────────────────────────────────────────────────────────────
	m.Dict.SetStr("NewType", &object.BuiltinFunc{Name: "NewType",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "NewType() requires name and type")
			}
			supertype := a[1]
			// Return a BuiltinFunc that calls the supertype constructor.
			bf := &object.BuiltinFunc{Name: "NewType",
				Call: func(_ any, inner []object.Object, kw *object.Dict) (object.Object, error) {
					return i.callObject(supertype, inner, kw)
				}}
			if bf.Attrs == nil {
				bf.Attrs = object.NewDict()
			}
			bf.Attrs.SetStr("__supertype__", supertype)
			return bf, nil
		}})

	// ── getClassAnnotations helper ────────────────────────────────────────────
	getAnnotationFields := func(cls *object.Class) ([]string, error) {
		// Python 3.14: __annotate_func__ is a lazy annotation function
		if annotateFn, ok := cls.Dict.GetStr("__annotate_func__"); ok {
			result, err := i.callObject(annotateFn, []object.Object{object.NewInt(1)}, nil)
			if err != nil {
				return nil, err
			}
			if d, ok := result.(*object.Dict); ok {
				keys, _ := d.Items()
				names := make([]string, 0, len(keys))
				for _, k := range keys {
					if s, ok := k.(*object.Str); ok {
						names = append(names, s.V)
					}
				}
				return names, nil
			}
		}
		// Fallback: __annotations__ dict (SETUP_ANNOTATIONS path, pre-3.14 .pyc)
		if annObj, ok := cls.Dict.GetStr("__annotations__"); ok {
			if d, ok := annObj.(*object.Dict); ok {
				keys, _ := d.Items()
				names := make([]string, 0, len(keys))
				for _, k := range keys {
					if s, ok := k.(*object.Str); ok {
						names = append(names, s.V)
					}
				}
				return names, nil
			}
		}
		return nil, nil
	}

	// applyNamedTupleMethods copies all namedtuple machinery from a newly
	// created namedtuple class into dst.
	applyNamedTupleMethods := func(dst *object.Class, fields []string, defaults []object.Object) {
		src := i.makeNamedTuple(dst.Name, fields, defaults)
		keys, vals := src.Dict.Items()
		for idx, k := range keys {
			dst.Dict.Set(k, vals[idx])
		}
	}

	// ── NamedTuple ────────────────────────────────────────────────────────────
	namedTupleCls := &object.Class{Name: "NamedTuple", Dict: object.NewDict()}
	// __new__ handles two cases:
	// 1. Functional form: NamedTuple('Point', [('x', int), ('y', int)])
	//    → a[0]=NamedTupleCls, a[1]=str  → returns new Class (not instance)
	// 2. Subclass instantiation: Color(255, 0, 128)
	//    → a[0]=Color (not namedTupleCls), returns a plain *object.Instance
	//      so that goipy continues to call Color.__init__(inst, ...)
	namedTupleCls.Dict.SetStr("__new__", &object.BuiltinFunc{Name: "__new__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "missing class argument")
			}
			cls, ok := a[0].(*object.Class)
			if !ok {
				return nil, object.Errorf(i.typeErr, "__new__ first arg must be a class")
			}
			// Functional form: NamedTuple('Name', [...])
			if cls == namedTupleCls && len(a) >= 2 {
				if typeName, ok := a[1].(*object.Str); ok {
					var fieldNames []string
					var fieldsObj object.Object
					if len(a) >= 3 {
						fieldsObj = a[2]
					}
					if fieldsObj != nil {
						items, err := iterate(i, fieldsObj)
						if err != nil {
							return nil, err
						}
						for _, item := range items {
							var fieldName string
							switch f := item.(type) {
							case *object.Tuple:
								if len(f.V) >= 1 {
									if s, ok := f.V[0].(*object.Str); ok {
										fieldName = s.V
									}
								}
							case *object.List:
								if len(f.V) >= 1 {
									if s, ok := f.V[0].(*object.Str); ok {
										fieldName = s.V
									}
								}
							}
							if fieldName != "" {
								fieldNames = append(fieldNames, fieldName)
							}
						}
					}
					return i.makeNamedTuple(typeName.V, fieldNames, nil), nil
				}
			}
			// Subclass instantiation path: Color(255, 0, 128) etc.
			// Return a plain instance; goipy will then call cls.__init__(inst, ...).
			return &object.Instance{Class: cls, Dict: object.NewDict()}, nil
		}})
	// Class form: class Color(NamedTuple): r:int; g:int; b:int
	namedTupleCls.Dict.SetStr("__init_subclass__", &object.BuiltinFunc{Name: "__init_subclass__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			cls, ok := a[0].(*object.Class)
			if !ok {
				return object.None, nil
			}
			fields, err := getAnnotationFields(cls)
			if err != nil {
				return nil, err
			}
			// Collect defaults: fields that have values in the class dict.
			var defaults []object.Object
			foundDefault := false
			for _, name := range fields {
				if v, ok := cls.Dict.GetStr(name); ok {
					defaults = append(defaults, v)
					foundDefault = true
				} else if foundDefault {
					// Fields without defaults cannot follow fields with defaults.
					return nil, object.Errorf(i.typeErr,
						"non-default argument '%s' follows default argument", name)
				}
			}
			applyNamedTupleMethods(cls, fields, defaults)
			return object.None, nil
		}})
	m.Dict.SetStr("NamedTuple", namedTupleCls)

	// ── TypedDict ─────────────────────────────────────────────────────────────
	typedDictCls := &object.Class{Name: "TypedDict", Dict: object.NewDict()}
	typedDictCls.Dict.SetStr("__new__", &object.BuiltinFunc{Name: "__new__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// Functional form: TypedDict('Movie', {'name': str, 'year': int})
			if len(a) >= 2 {
				if typeName, ok := a[1].(*object.Str); ok {
					tdCls := &object.Class{Name: typeName.V, Dict: object.NewDict()}
					tdCls.Dict.SetStr("__is_typeddict__", object.BoolOf(true))
					tdCls.Dict.SetStr("__new__", &object.BuiltinFunc{Name: "__new__",
						Call: func(_ any, inner []object.Object, innerKw *object.Dict) (object.Object, error) {
							d := object.NewDict()
							if innerKw != nil {
								ks, vs := innerKw.Items()
								for idx, k := range ks {
									d.Set(k, vs[idx])
								}
							}
							return d, nil
						}})
					return tdCls, nil
				}
			}
			return nil, object.Errorf(i.typeErr, "TypedDict() requires a typename")
		}})
	typedDictCls.Dict.SetStr("__init_subclass__", &object.BuiltinFunc{Name: "__init_subclass__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			cls, ok := a[0].(*object.Class)
			if !ok {
				return object.None, nil
			}
			// Mark as TypedDict and install a __new__ that returns a plain dict.
			cls.Dict.SetStr("__is_typeddict__", object.BoolOf(true))
			cls.Dict.SetStr("__new__", &object.BuiltinFunc{Name: "__new__",
				Call: func(_ any, inner []object.Object, innerKw *object.Dict) (object.Object, error) {
					d := object.NewDict()
					if innerKw != nil {
						ks, vs := innerKw.Items()
						for idx, k := range ks {
							d.Set(k, vs[idx])
						}
					}
					return d, nil
				}})
			return object.None, nil
		}})
	m.Dict.SetStr("TypedDict", typedDictCls)

	// ── is_typeddict ──────────────────────────────────────────────────────────
	m.Dict.SetStr("is_typeddict", &object.BuiltinFunc{Name: "is_typeddict",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if cls, ok := a[0].(*object.Class); ok {
				if _, hasMarker := cls.Dict.GetStr("__is_typeddict__"); hasMarker {
					return object.BoolOf(true), nil
				}
			}
			return object.BoolOf(false), nil
		}})

	// ── Special forms (NoReturn, Never, Self, LiteralString, …) ─────────────
	for _, name := range []string{
		"NoReturn", "Never", "Self", "LiteralString", "Text",
		"TypeAlias", "TypeGuard", "TypeIs", "ReadOnly", "Required", "NotRequired",
		"ParamSpecArgs", "ParamSpecKwargs", "Concatenate", "Unpack", "Reveal",
		"ForwardRef",
	} {
		nameCopy := name
		sfCls := &object.Class{Name: nameCopy, Dict: object.NewDict()}
		sfCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.Str{V: "typing." + nameCopy}, nil
			}})
		sfInst := &object.Instance{Class: sfCls, Dict: object.NewDict()}
		m.Dict.SetStr(nameCopy, sfInst)
	}

	// ── AnyStr ───────────────────────────────────────────────────────────────
	strBuiltin, _ := i.Builtins.GetStr("str")
	bytesBuiltin, _ := i.Builtins.GetStr("bytes")
	anyStrInst := &object.Instance{Class: typeVarCls, Dict: object.NewDict()}
	anyStrInst.Dict.SetStr("__name__", &object.Str{V: "AnyStr"})
	anyStrInst.Dict.SetStr("__constraints__", &object.Tuple{V: []object.Object{strBuiltin, bytesBuiltin}})
	anyStrInst.Dict.SetStr("__bound__", object.None)
	anyStrInst.Dict.SetStr("__covariant__", object.BoolOf(false))
	anyStrInst.Dict.SetStr("__contravariant__", object.BoolOf(false))
	m.Dict.SetStr("AnyStr", anyStrInst)

	// ── IO / BinaryIO / TextIO ────────────────────────────────────────────────
	for _, name := range []string{"IO", "BinaryIO", "TextIO"} {
		cls := &object.Class{Name: name, Dict: object.NewDict()}
		m.Dict.SetStr(name, cls)
	}

	// ── ParamSpec / TypeVarTuple ──────────────────────────────────────────────
	for _, name := range []string{"ParamSpec", "TypeVarTuple"} {
		nameCopy := name
		paramCls := &object.Class{Name: nameCopy, Dict: object.NewDict()}
		paramCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) >= 2 {
					inst := a[0].(*object.Instance)
					if s, ok := a[1].(*object.Str); ok {
						inst.Dict.SetStr("__name__", s)
					}
				}
				return object.None, nil
			}})
		m.Dict.SetStr(nameCopy, paramCls)
	}

	// ── TypeAliasType ─────────────────────────────────────────────────────────
	typeAliasTypeCls := &object.Class{Name: "TypeAliasType", Dict: object.NewDict()}
	m.Dict.SetStr("TypeAliasType", typeAliasTypeCls)

	// ── Miscellaneous one-shot functions ─────────────────────────────────────
	m.Dict.SetStr("get_type_hints", &object.BuiltinFunc{Name: "get_type_hints",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.NewDict(), nil
			}
			var annObj object.Object
			switch o := a[0].(type) {
			case *object.Class:
				annObj, _ = o.Dict.GetStr("__annotations__")
			case *object.Function:
				if o.Annotations != nil {
					annObj = o.Annotations
				}
			case *object.Instance:
				annObj, _ = o.Class.Dict.GetStr("__annotations__")
			}
			if d, ok := annObj.(*object.Dict); ok {
				return d, nil
			}
			return object.NewDict(), nil
		}})

	m.Dict.SetStr("assert_never", &object.BuiltinFunc{Name: "assert_never",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.typeErr, "assert_never called with %s", object.Repr(a[0]))
		}})
	m.Dict.SetStr("assert_type", &object.BuiltinFunc{Name: "assert_type",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				return a[0], nil
			}
			return object.None, nil
		}})
	m.Dict.SetStr("reveal_type", &object.BuiltinFunc{Name: "reveal_type",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				return a[0], nil
			}
			return object.None, nil
		}})
	m.Dict.SetStr("dataclass_transform", &object.BuiltinFunc{Name: "dataclass_transform",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			// Returns a decorator that returns the class/function unchanged.
			return &object.BuiltinFunc{Name: "dataclass_transform",
				Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
					return a[0], nil
				}}, nil
		}})
	m.Dict.SetStr("override", &object.BuiltinFunc{Name: "override",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return a[0], nil
		}})
	m.Dict.SetStr("get_overloads", &object.BuiltinFunc{Name: "get_overloads",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: nil}, nil
		}})
	m.Dict.SetStr("clear_overloads", &object.BuiltinFunc{Name: "clear_overloads",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	m.Dict.SetStr("get_protocol_members", &object.BuiltinFunc{Name: "get_protocol_members",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if cls, ok := a[0].(*object.Class); ok {
				var members []object.Object
				keys, _ := cls.Dict.Items()
				for _, k := range keys {
					if s, ok := k.(*object.Str); ok {
						if !strings.HasPrefix(s.V, "__") {
							members = append(members, s)
						}
					}
				}
				return &object.Tuple{V: members}, nil
			}
			return &object.Tuple{V: nil}, nil
		}})
	m.Dict.SetStr("is_protocol", &object.BuiltinFunc{Name: "is_protocol",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if cls, ok := a[0].(*object.Class); ok {
				for _, base := range cls.Bases {
					if base == protocolCls {
						return object.BoolOf(true), nil
					}
				}
			}
			return object.BoolOf(false), nil
		}})
	m.Dict.SetStr("evaluate_forward_ref", &object.BuiltinFunc{Name: "evaluate_forward_ref",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				return a[0], nil
			}
			return object.None, nil
		}})

	// NoDefault sentinel
	noDefaultCls := &object.Class{Name: "NoDefault", Dict: object.NewDict()}
	noDefaultInst := &object.Instance{Class: noDefaultCls, Dict: object.NewDict()}
	m.Dict.SetStr("NoDefault", noDefaultInst)

	return m
}

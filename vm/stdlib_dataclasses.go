package vm

import (
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildDataclasses() *object.Module {
	m := &object.Module{Name: "dataclasses", Dict: object.NewDict()}

	// ── MISSING sentinel ──────────────────────────────────────────────────────
	missClass := &object.Class{Name: "_MISSING_TYPE", Dict: object.NewDict()}
	missing := &object.Instance{Class: missClass, Dict: object.NewDict()}
	m.Dict.SetStr("MISSING", missing)
	// HAS_DEFAULT_FACTORY is its own sentinel in CPython; aliasing MISSING is close enough.
	m.Dict.SetStr("HAS_DEFAULT_FACTORY", missing)

	// ── KW_ONLY sentinel ─────────────────────────────────────────────────────
	kwOnlyClass := &object.Class{Name: "_KW_ONLY_TYPE", Dict: object.NewDict()}
	kwOnlySentinel := &object.Instance{Class: kwOnlyClass, Dict: object.NewDict()}
	m.Dict.SetStr("KW_ONLY", kwOnlySentinel)

	// ── FrozenInstanceError ───────────────────────────────────────────────────
	frozenErr := &object.Class{
		Name:  "FrozenInstanceError",
		Bases: []*object.Class{i.exception},
		Dict:  object.NewDict(),
	}
	m.Dict.SetStr("FrozenInstanceError", frozenErr)

	// ── InitVar class ─────────────────────────────────────────────────────────
	initVarCls := &object.Class{Name: "InitVar", Dict: object.NewDict()}
	m.Dict.SetStr("InitVar", initVarCls)

	// ── Field class ───────────────────────────────────────────────────────────
	fieldCls := dcMakeFieldClass()
	m.Dict.SetStr("Field", fieldCls)

	// ── field() constructor ───────────────────────────────────────────────────
	m.Dict.SetStr("field", &object.BuiltinFunc{Name: "field",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			interp := interpFrom(ii)
			if interp == nil {
				interp = i
			}
			return dcMakeField(interp, fieldCls, missing, a, kw)
		}})

	// ── is_dataclass(obj) ─────────────────────────────────────────────────────
	m.Dict.SetStr("is_dataclass", &object.BuiltinFunc{Name: "is_dataclass",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.False, nil
			}
			return object.BoolOf(dcIsDataclass(a[0])), nil
		}})

	// ── fields(obj) ───────────────────────────────────────────────────────────
	m.Dict.SetStr("fields", &object.BuiltinFunc{Name: "fields",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "fields() takes 1 argument")
			}
			flds, ok := dcGetFields(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "fields() called on a non-dataclass")
			}
			out := make([]object.Object, len(flds))
			for k, f := range flds {
				out[k] = f
			}
			return &object.Tuple{V: out}, nil
		}})

	// ── asdict(obj, *, dict_factory=dict) ────────────────────────────────────
	m.Dict.SetStr("asdict", &object.BuiltinFunc{Name: "asdict",
		Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			if len(a) == 0 {
				return nil, object.Errorf(ii.typeErr, "asdict() takes 1 argument")
			}
			seen := map[uintptr]bool{}
			return dcAsdict(ii, a[0], seen)
		}})

	// ── astuple(obj, *, tuple_factory=tuple) ─────────────────────────────────
	m.Dict.SetStr("astuple", &object.BuiltinFunc{Name: "astuple",
		Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			if len(a) == 0 {
				return nil, object.Errorf(ii.typeErr, "astuple() takes 1 argument")
			}
			seen := map[uintptr]bool{}
			return dcAstuple(ii, a[0], seen)
		}})

	// ── replace(obj, **changes) ───────────────────────────────────────────────
	m.Dict.SetStr("replace", &object.BuiltinFunc{Name: "replace",
		Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			if len(a) == 0 {
				return nil, object.Errorf(ii.typeErr, "replace() takes at least 1 argument")
			}
			return dcReplace(ii, a[0], kw, missing)
		}})

	// ── make_dataclass(cls_name, fields, *, bases=(), ...) ───────────────────
	m.Dict.SetStr("make_dataclass", &object.BuiltinFunc{Name: "make_dataclass",
		Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			if len(a) < 2 {
				return nil, object.Errorf(ii.typeErr, "make_dataclass() requires cls_name and fields")
			}
			name := ""
			if s, ok := a[0].(*object.Str); ok {
				name = s.V
			}
			var bases []*object.Class
			if kw != nil {
				if b, ok := kw.GetStr("bases"); ok {
					if bt, ok2 := b.(*object.Tuple); ok2 {
						for _, bv := range bt.V {
							if cls, ok3 := bv.(*object.Class); ok3 {
								bases = append(bases, cls)
							}
						}
					}
				}
			}
			cls := &object.Class{Name: name, Bases: bases, Dict: object.NewDict()}
			fieldList, _ := a[1].(*object.List)
			if fieldList == nil {
				if ft, ok := a[1].(*object.Tuple); ok {
					fieldList = &object.List{V: ft.V}
				}
			}
			if fieldList != nil {
				dcBuildAnnotations(ii, cls, fieldList.V, fieldCls, missing)
			}
			doFrozen := dcKwBool(kw, "frozen", false)
			doOrder := dcKwBool(kw, "order", false)
			doInit := dcKwBool(kw, "init", true)
			doRepr := dcKwBool(kw, "repr", true)
			doEq := dcKwBool(kw, "eq", true)
			doMatchArgs := dcKwBool(kw, "match_args", true)
			doKwOnly := dcKwBool(kw, "kw_only", false)
			doUnsafeHash := dcKwBool(kw, "unsafe_hash", false)
			return dcProcessClass(ii, cls, fieldCls, missing, frozenErr, kwOnlySentinel,
				doFrozen, doOrder, doInit, doRepr, doEq, doMatchArgs, doKwOnly, doUnsafeHash), nil
		}})

	// ── @dataclass decorator ──────────────────────────────────────────────────
	var dcDecorator object.Object
	dcDecorator = &object.BuiltinFunc{Name: "dataclass",
		Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			// Called as @dataclass (no parens) → a[0] is the class.
			if len(a) == 1 {
				if cls, ok := a[0].(*object.Class); ok {
					return dcProcessClass(ii, cls, fieldCls, missing, frozenErr, kwOnlySentinel,
						false, false, true, true, true, true, false, false), nil
				}
			}
			// Called as @dataclass(...) → return a decorator.
			doInit := dcKwBool(kw, "init", true)
			doRepr := dcKwBool(kw, "repr", true)
			doEq := dcKwBool(kw, "eq", true)
			doOrder := dcKwBool(kw, "order", false)
			frozen := dcKwBool(kw, "frozen", false)
			doMatchArgs := dcKwBool(kw, "match_args", true)
			doKwOnly := dcKwBool(kw, "kw_only", false)
			doUnsafeHash := dcKwBool(kw, "unsafe_hash", false)
			_ = dcKwBool(kw, "slots", false)
			return &object.BuiltinFunc{Name: "dataclass_decorator",
				Call: func(interp2 any, a2 []object.Object, _ *object.Dict) (object.Object, error) {
					ii2 := interpFrom(interp2)
					if ii2 == nil {
						ii2 = ii
					}
					if len(a2) == 0 {
						return nil, object.Errorf(ii2.typeErr, "dataclass decorator requires a class")
					}
					cls, ok := a2[0].(*object.Class)
					if !ok {
						return nil, object.Errorf(ii2.typeErr, "dataclass() applied to non-class")
					}
					return dcProcessClass(ii2, cls, fieldCls, missing, frozenErr, kwOnlySentinel,
						frozen, doOrder, doInit, doRepr, doEq, doMatchArgs, doKwOnly, doUnsafeHash), nil
				}}, nil
		}}
	_ = dcDecorator
	m.Dict.SetStr("dataclass", dcDecorator)

	return m
}

// ── Field class ──────────────────────────────────────────────────────────────

func dcMakeFieldClass() *object.Class {
	cls := &object.Class{Name: "Field", Dict: object.NewDict()}
	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return &object.Str{V: "Field()"}, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return &object.Str{V: "Field()"}, nil
			}
			name := ""
			if nv, ok2 := inst.Dict.GetStr("name"); ok2 {
				if s, ok3 := nv.(*object.Str); ok3 {
					name = s.V
				}
			}
			return &object.Str{V: "Field(name=" + name + ")"}, nil
		}})
	return cls
}

func dcMakeField(ii *Interp, fieldCls *object.Class, missing *object.Instance, a []object.Object, kw *object.Dict) (*object.Instance, error) {
	f := &object.Instance{Class: fieldCls, Dict: object.NewDict()}
	f.Dict.SetStr("name", &object.Str{V: ""})

	def := object.Object(missing)
	defFactory := object.Object(missing)
	repr := object.Object(object.True)
	hash := object.Object(object.None)
	init := object.Object(object.True)
	compare := object.Object(object.True)
	metadata := object.Object(object.None)
	kwOnly := object.Object(object.False)

	if kw != nil {
		if v, ok := kw.GetStr("default"); ok {
			def = v
		}
		if v, ok := kw.GetStr("default_factory"); ok {
			defFactory = v
		}
		if v, ok := kw.GetStr("repr"); ok {
			repr = v
		}
		if v, ok := kw.GetStr("hash"); ok {
			hash = v
		}
		if v, ok := kw.GetStr("init"); ok {
			init = v
		}
		if v, ok := kw.GetStr("compare"); ok {
			compare = v
		}
		if v, ok := kw.GetStr("metadata"); ok {
			metadata = v
		}
		if v, ok := kw.GetStr("kw_only"); ok {
			kwOnly = v
		}
	}

	// Raise ValueError if both default and default_factory are set.
	if def != object.Object(missing) && defFactory != object.Object(missing) {
		if ii != nil {
			return nil, object.Errorf(ii.valueErr, "cannot specify both default and default_factory")
		}
	}

	f.Dict.SetStr("default", def)
	f.Dict.SetStr("default_factory", defFactory)
	f.Dict.SetStr("repr", repr)
	f.Dict.SetStr("hash", hash)
	f.Dict.SetStr("init", init)
	f.Dict.SetStr("compare", compare)
	f.Dict.SetStr("metadata", metadata)
	f.Dict.SetStr("kw_only", kwOnly)
	return f, nil
}

// ── dataclass params ──────────────────────────────────────────────────────────

func dcMakeParams(frozen, order, init, matchArgs, kwOnly, unsafeHash bool) *object.Instance {
	cls := &object.Class{Name: "_DataclassParams", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("frozen", object.BoolOf(frozen))
	inst.Dict.SetStr("order", object.BoolOf(order))
	inst.Dict.SetStr("init", object.BoolOf(init))
	inst.Dict.SetStr("match_args", object.BoolOf(matchArgs))
	inst.Dict.SetStr("kw_only", object.BoolOf(kwOnly))
	inst.Dict.SetStr("unsafe_hash", object.BoolOf(unsafeHash))
	return inst
}

// ── process class ─────────────────────────────────────────────────────────────

func dcProcessClass(ii *Interp, cls *object.Class, fieldCls *object.Class, missing *object.Instance,
	frozenErr *object.Class, kwOnlySentinel *object.Instance,
	frozen, order, doInit, doRepr, doEq, matchArgs, classKwOnly, unsafeHash bool) *object.Class {

	// Build __dataclass_fields__ from class annotations.
	fields := object.NewDict()
	cls.Dict.SetStr("__dataclass_fields__", fields)
	cls.Dict.SetStr("__dataclass_params__", dcMakeParams(frozen, order, doInit, matchArgs, classKwOnly, unsafeHash))

	// Collect annotations. Python 3.14 uses __annotate_func__ (lazy evaluation);
	// fallback to __annotations__ for older .pyc or pre-3.14 paths.
	annoDict := dcGetAnnotations(ii, cls)
	var ownFieldOrder []string

	// Track whether we've passed a KW_ONLY sentinel in the annotation list.
	seenKwOnly := classKwOnly

	if annoDict != nil {
		keys, vals := annoDict.Items()
		for k, key := range keys {
			ks, ok2 := key.(*object.Str)
			if !ok2 {
				continue
			}
			fname := ks.V
			typeVal := vals[k]

			// KW_ONLY sentinel field: marks all subsequent fields as kw_only.
			if kwOnlySentinel != nil {
				if typeInst, ok3 := typeVal.(*object.Instance); ok3 && typeInst == kwOnlySentinel {
					seenKwOnly = true
					continue
				}
				// Also match any instance of the KW_ONLY class.
				if typeInst, ok3 := typeVal.(*object.Instance); ok3 &&
					typeInst.Class == kwOnlySentinel.Class {
					seenKwOnly = true
					continue
				}
			}

			ownFieldOrder = append(ownFieldOrder, fname)

			// Check if class has a default/field() value for this name.
			var defVal object.Object = missing
			var defFactory object.Object = missing

			existing, hasDef := cls.Dict.GetStr(fname)
			var fobj *object.Instance
			if hasDef {
				if fInst, ok3 := existing.(*object.Instance); ok3 && fInst.Class == fieldCls {
					// It's a field() descriptor — use it directly, just set name/type.
					fobj = fInst
					if d, ok4 := fInst.Dict.GetStr("default"); ok4 {
						defVal = d
					}
					if df, ok4 := fInst.Dict.GetStr("default_factory"); ok4 {
						defFactory = df
					}
					cls.Dict.SetStr(fname, missing)
				} else if existing != object.Object(missing) {
					defVal = existing
				}
			}
			if fobj == nil {
				fobj, _ = dcMakeField(ii, fieldCls, missing, nil, nil)
			}
			fobj.Dict.SetStr("name", &object.Str{V: fname})
			fobj.Dict.SetStr("type", typeVal)
			fobj.Dict.SetStr("default", defVal)
			fobj.Dict.SetStr("default_factory", defFactory)
			// Apply per-field or class-level kw_only.
			if seenKwOnly {
				fobj.Dict.SetStr("kw_only", object.True)
			}
			fields.Set(key, fobj)
		}
	}

	// ── Inheritance: prepend fields from dataclass base classes ───────────────
	// CPython rule: fields from base classes come first, in left-to-right MRO order.
	// Child fields override same-named base fields.
	var inheritedOrder []string
	for _, base := range cls.Bases {
		fo, ok := base.Dict.GetStr("__dataclass_field_order__")
		if !ok {
			continue
		}
		baseOrder, ok2 := fo.(*object.List)
		if !ok2 {
			continue
		}
		fd, ok3 := base.Dict.GetStr("__dataclass_fields__")
		if !ok3 {
			continue
		}
		baseFields, ok4 := fd.(*object.Dict)
		if !ok4 {
			continue
		}
		for _, nameObj := range baseOrder.V {
			ns, ok5 := nameObj.(*object.Str)
			if !ok5 {
				continue
			}
			// Child overrides: skip if child defined same name.
			childOverrides := false
			for _, cn := range ownFieldOrder {
				if cn == ns.V {
					childOverrides = true
					break
				}
			}
			if childOverrides {
				continue
			}
			// Skip if already added from a prior base.
			alreadyAdded := false
			for _, prev := range inheritedOrder {
				if prev == ns.V {
					alreadyAdded = true
					break
				}
			}
			if alreadyAdded {
				continue
			}
			fobj, ok6 := baseFields.GetStr(ns.V)
			if !ok6 {
				continue
			}
			inheritedOrder = append(inheritedOrder, ns.V)
			fields.Set(nameObj, fobj)
		}
	}

	// Final field order: inherited first, then own.
	fieldOrder := append(inheritedOrder, ownFieldOrder...)

	// Store field order for __init__ and __repr__.
	orderList := make([]object.Object, len(fieldOrder))
	for k, n := range fieldOrder {
		orderList[k] = &object.Str{V: n}
	}
	cls.Dict.SetStr("__dataclass_field_order__", &object.List{V: orderList})

	// ── __match_args__ ────────────────────────────────────────────────────────
	if matchArgs {
		var matchArgsSlice []object.Object
		for _, n := range fieldOrder {
			fobj, ok := dcLookupField(fields, n)
			if !ok {
				continue
			}
			isKwOnly := false
			if kwOnlyV, ok2 := fobj.Dict.GetStr("kw_only"); ok2 {
				if b, ok3 := kwOnlyV.(*object.Bool); ok3 {
					isKwOnly = b.V
				}
			}
			if !isKwOnly {
				matchArgsSlice = append(matchArgsSlice, &object.Str{V: n})
			}
		}
		cls.Dict.SetStr("__match_args__", &object.Tuple{V: matchArgsSlice})
	}

	// ── __init__ ──────────────────────────────────────────────────────────────
	if doInit {
		cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
			Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
				execII := interpFrom(interp)
				if execII == nil {
					execII = ii
				}
				if len(a) == 0 {
					return object.None, nil
				}
				self, ok := a[0].(*object.Instance)
				if !ok {
					return object.None, nil
				}
				argIdx := 1 // a[0] = self

				// Sort fields into declaration order.
				sortedFields := dcSortedFields(fields, fieldOrder)

				for _, sf := range sortedFields {
					fname := sf.name
					fobj := sf.fobj
					if fobj == nil {
						continue
					}
					// Skip fields with init=False.
					if initFlag, ok2 := fobj.Dict.GetStr("init"); ok2 {
						if b, ok3 := initFlag.(*object.Bool); ok3 && !b.V {
							defVal, _ := fobj.Dict.GetStr("default")
							if defVal != nil && defVal != object.Object(missing) {
								self.Dict.SetStr(fname, defVal)
							} else if df, ok4 := fobj.Dict.GetStr("default_factory"); ok4 && df != object.Object(missing) {
								val, err := execII.callObject(df, nil, nil)
								if err != nil {
									return nil, err
								}
								self.Dict.SetStr(fname, val)
							}
							continue
						}
					}
					// Determine if this field is kw_only.
					isKwOnly := false
					if kwOnlyV, ok2 := fobj.Dict.GetStr("kw_only"); ok2 {
						if b, ok3 := kwOnlyV.(*object.Bool); ok3 {
							isKwOnly = b.V
						}
					}

					var val object.Object
					// Check kwargs first.
					if kw != nil {
						if v, ok2 := kw.GetStr(fname); ok2 {
							val = v
						}
					}
					// Positional args (only for non-kw_only fields).
					if val == nil && !isKwOnly && argIdx < len(a) {
						val = a[argIdx]
						argIdx++
					}
					if val == nil {
						// Use default.
						defVal, hasDef := fobj.Dict.GetStr("default")
						if hasDef && defVal != object.Object(missing) {
							val = defVal
						} else if df, ok2 := fobj.Dict.GetStr("default_factory"); ok2 && df != object.Object(missing) {
							v, err := execII.callObject(df, nil, nil)
							if err != nil {
								return nil, err
							}
							val = v
						} else {
							return nil, object.Errorf(execII.typeErr, "__init__() missing argument: '%s'", fname)
						}
					}
					self.Dict.SetStr(fname, val)
				}
				// Call __post_init__ if defined.
				if postFn, found := classLookup(cls, "__post_init__"); found {
					_, err := execII.callObject(postFn, []object.Object{self}, nil)
					if err != nil {
						return nil, err
					}
				}
				return object.None, nil
			}})
	}

	// ── __repr__ ──────────────────────────────────────────────────────────────
	if doRepr {
		cls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) == 0 {
					return &object.Str{V: cls.Name + "()"}, nil
				}
				self, ok := a[0].(*object.Instance)
				if !ok {
					return &object.Str{V: cls.Name + "()"}, nil
				}
				var parts []string
				for _, n := range fieldOrder {
					fobj, hasFld := dcLookupField(fields, n)
					if !hasFld {
						continue
					}
					if rv, ok2 := fobj.Dict.GetStr("repr"); ok2 {
						if b, ok3 := rv.(*object.Bool); ok3 && !b.V {
							continue
						}
					}
					val, ok2 := self.Dict.GetStr(n)
					if !ok2 {
						continue
					}
					parts = append(parts, n+"="+object.Repr(val))
				}
				return &object.Str{V: cls.Name + "(" + strings.Join(parts, ", ") + ")"}, nil
			}})
	}

	// ── __eq__ ────────────────────────────────────────────────────────────────
	if doEq {
		cls.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) < 2 {
					return object.False, nil
				}
				selfInst, ok := a[0].(*object.Instance)
				otherInst, ok2 := a[1].(*object.Instance)
				if !ok || !ok2 || selfInst.Class != otherInst.Class {
					return object.False, nil
				}
				for _, n := range fieldOrder {
					fobj, hasFld := dcLookupField(fields, n)
					if !hasFld {
						continue
					}
					if cv, ok3 := fobj.Dict.GetStr("compare"); ok3 {
						if b, ok4 := cv.(*object.Bool); ok4 && !b.V {
							continue
						}
					}
					v1, _ := selfInst.Dict.GetStr(n)
					v2, _ := otherInst.Dict.GetStr(n)
					eq, _ := object.Eq(v1, v2)
					if !eq {
						return object.False, nil
					}
				}
				return object.True, nil
			}})
	}

	// ── ordering (__lt__, __le__, __gt__, __ge__) ─────────────────────────────
	if order {
		makeCmp := func(opName string) *object.BuiltinFunc {
			return &object.BuiltinFunc{Name: opName,
				Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
					if len(a) < 2 {
						return object.False, nil
					}
					selfInst, ok := a[0].(*object.Instance)
					otherInst, ok2 := a[1].(*object.Instance)
					if !ok || !ok2 || selfInst.Class != otherInst.Class {
						return object.NotImplemented, nil
					}
					for _, n := range fieldOrder {
						fobj, hasFld := dcLookupField(fields, n)
						if !hasFld {
							continue
						}
						if cv, ok3 := fobj.Dict.GetStr("compare"); ok3 {
							if b, ok4 := cv.(*object.Bool); ok4 && !b.V {
								continue
							}
						}
						v1, _ := selfInst.Dict.GetStr(n)
						v2, _ := otherInst.Dict.GetStr(n)
						eq, _ := object.Eq(v1, v2)
						if !eq {
							switch opName {
							case "__lt__":
								r, _ := ii.compare(v1, v2, cmpLT)
								return r, nil
							case "__le__":
								r, _ := ii.compare(v1, v2, cmpLE)
								return r, nil
							case "__gt__":
								r, _ := ii.compare(v1, v2, cmpGT)
								return r, nil
							case "__ge__":
								r, _ := ii.compare(v1, v2, cmpGE)
								return r, nil
							}
						}
					}
					switch opName {
					case "__lt__", "__gt__":
						return object.False, nil
					default:
						return object.True, nil
					}
				}}
		}
		cls.Dict.SetStr("__lt__", makeCmp("__lt__"))
		cls.Dict.SetStr("__le__", makeCmp("__le__"))
		cls.Dict.SetStr("__gt__", makeCmp("__gt__"))
		cls.Dict.SetStr("__ge__", makeCmp("__ge__"))
	}

	// ── frozen: override __setattr__ / __delattr__ ────────────────────────────
	if frozen {
		cls.Dict.SetStr("__setattr__", &object.BuiltinFunc{Name: "__setattr__",
			Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
				execII := interpFrom(interp)
				if execII == nil {
					execII = ii
				}
				return nil, object.Errorf(frozenErr, "cannot assign to field of frozen instance")
			}})
		cls.Dict.SetStr("__delattr__", &object.BuiltinFunc{Name: "__delattr__",
			Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
				execII := interpFrom(interp)
				if execII == nil {
					execII = ii
				}
				return nil, object.Errorf(frozenErr, "cannot delete field of frozen instance")
			}})
	}

	// ── __hash__ (frozen or unsafe_hash) ─────────────────────────────────────
	if frozen || (unsafeHash && doEq) {
		cls.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) == 0 {
					return object.NewInt(0), nil
				}
				self, ok := a[0].(*object.Instance)
				if !ok {
					return object.NewInt(0), nil
				}
				var h uint64
				for _, n := range fieldOrder {
					fobj, ok2 := dcLookupField(fields, n)
					if !ok2 {
						continue
					}
					// Respect hash=False on individual fields.
					if hv, ok3 := fobj.Dict.GetStr("hash"); ok3 {
						if b, ok4 := hv.(*object.Bool); ok4 && !b.V {
							continue
						}
					}
					if v, ok2b := self.Dict.GetStr(n); ok2b {
						hval, _ := object.Hash(v)
						h = h*31 + hval
					}
				}
				return object.NewInt(int64(h)), nil
			}})
	}

	return cls
}

// dcSortedFields returns fields in declaration order for use in __init__.
func dcSortedFields(fields *object.Dict, fieldOrder []string) []struct {
	name string
	fobj *object.Instance
} {
	fkeys, fvals := fields.Items()
	fieldMap := make(map[string]*object.Instance, len(fkeys))
	for k, fk := range fkeys {
		if ks, ok := fk.(*object.Str); ok {
			if fInst, ok2 := fvals[k].(*object.Instance); ok2 {
				fieldMap[ks.V] = fInst
			}
		}
	}
	result := make([]struct {
		name string
		fobj *object.Instance
	}, len(fieldOrder))
	for idx, n := range fieldOrder {
		result[idx].name = n
		result[idx].fobj = fieldMap[n]
	}
	return result
}

// dcBuildAnnotations builds __annotations__ in a class dict from a list of
// (name,) / (name, type) / (name, type, default) specs (for make_dataclass).
func dcBuildAnnotations(_ *Interp, cls *object.Class, specs []object.Object, _ *object.Class, missing *object.Instance) {
	anno := object.NewDict()
	cls.Dict.SetStr("__annotations__", anno)
	for _, spec := range specs {
		var fname, ftype string
		var defVal object.Object = missing
		switch sv := spec.(type) {
		case *object.Str:
			fname = sv.V
			ftype = "typing.Any"
		case *object.Tuple:
			if len(sv.V) >= 1 {
				if s, ok := sv.V[0].(*object.Str); ok {
					fname = s.V
				}
			}
			if len(sv.V) >= 2 {
				if s, ok := sv.V[1].(*object.Str); ok {
					ftype = s.V
				}
			}
			if len(sv.V) >= 3 {
				defVal = sv.V[2]
			}
		}
		if fname == "" {
			continue
		}
		anno.Set(&object.Str{V: fname}, &object.Str{V: ftype})
		if defVal != missing {
			cls.Dict.SetStr(fname, defVal)
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// dcGetAnnotations returns the annotations dict for a class.
// Handles Python 3.14's __annotate_func__ lazy evaluation as well as the
// older __annotations__ dict set by SETUP_ANNOTATIONS.
func dcGetAnnotations(ii *Interp, cls *object.Class) *object.Dict {
	if annotateFn, ok := cls.Dict.GetStr("__annotate_func__"); ok {
		result, err := ii.callObject(annotateFn, []object.Object{object.NewInt(1)}, nil)
		if err == nil {
			if d, ok2 := result.(*object.Dict); ok2 {
				return d
			}
		}
	}
	if annObj, ok := cls.Dict.GetStr("__annotations__"); ok {
		if d, ok2 := annObj.(*object.Dict); ok2 {
			return d
		}
	}
	return nil
}

func dcIsDataclass(o object.Object) bool {
	switch v := o.(type) {
	case *object.Class:
		_, ok := v.Dict.GetStr("__dataclass_fields__")
		return ok
	case *object.Instance:
		_, ok := v.Class.Dict.GetStr("__dataclass_fields__")
		return ok
	}
	return false
}

func dcGetFields(o object.Object) ([]*object.Instance, bool) {
	var fieldsDict *object.Dict
	var orderList *object.List

	switch v := o.(type) {
	case *object.Class:
		fd, ok := v.Dict.GetStr("__dataclass_fields__")
		if !ok {
			return nil, false
		}
		fieldsDict, _ = fd.(*object.Dict)
		ol, _ := v.Dict.GetStr("__dataclass_field_order__")
		orderList, _ = ol.(*object.List)
	case *object.Instance:
		fd, ok := v.Class.Dict.GetStr("__dataclass_fields__")
		if !ok {
			return nil, false
		}
		fieldsDict, _ = fd.(*object.Dict)
		ol, _ := v.Class.Dict.GetStr("__dataclass_field_order__")
		orderList, _ = ol.(*object.List)
	default:
		return nil, false
	}

	if fieldsDict == nil {
		return nil, true
	}

	var order []string
	if orderList != nil {
		for _, ov := range orderList.V {
			if s, ok := ov.(*object.Str); ok {
				order = append(order, s.V)
			}
		}
	}

	out := make([]*object.Instance, 0, len(order))
	for _, n := range order {
		fobj, ok := fieldsDict.GetStr(n)
		if !ok {
			continue
		}
		if finst, ok2 := fobj.(*object.Instance); ok2 {
			out = append(out, finst)
		}
	}
	return out, true
}

func dcLookupField(fields *object.Dict, name string) (*object.Instance, bool) {
	fobj, ok := fields.GetStr(name)
	if !ok {
		return nil, false
	}
	finst, ok2 := fobj.(*object.Instance)
	return finst, ok2
}

func dcAsdict(ii *Interp, o object.Object, seen map[uintptr]bool) (object.Object, error) {
	inst, ok := o.(*object.Instance)
	if !ok || !dcIsDataclass(o) {
		return o, nil
	}
	fields, _ := dcGetFields(inst)
	d := object.NewDict()
	for _, f := range fields {
		nameObj, _ := f.Dict.GetStr("name")
		ns, ok2 := nameObj.(*object.Str)
		if !ok2 {
			continue
		}
		val, _ := inst.Dict.GetStr(ns.V)
		conv, err := dcAsdictConvert(ii, val, seen)
		if err != nil {
			return nil, err
		}
		d.Set(nameObj, conv)
	}
	return d, nil
}

func dcAsdictConvert(ii *Interp, o object.Object, seen map[uintptr]bool) (object.Object, error) {
	if dcIsDataclass(o) {
		return dcAsdict(ii, o, seen)
	}
	switch v := o.(type) {
	case *object.List:
		out := make([]object.Object, len(v.V))
		for k, item := range v.V {
			conv, err := dcAsdictConvert(ii, item, seen)
			if err != nil {
				return nil, err
			}
			out[k] = conv
		}
		return &object.List{V: out}, nil
	case *object.Tuple:
		out := make([]object.Object, len(v.V))
		for k, item := range v.V {
			conv, err := dcAsdictConvert(ii, item, seen)
			if err != nil {
				return nil, err
			}
			out[k] = conv
		}
		return &object.Tuple{V: out}, nil
	case *object.Dict:
		nd := object.NewDict()
		keys, vals := v.Items()
		for k, key := range keys {
			kc, err := dcAsdictConvert(ii, key, seen)
			if err != nil {
				return nil, err
			}
			vc, err := dcAsdictConvert(ii, vals[k], seen)
			if err != nil {
				return nil, err
			}
			nd.Set(kc, vc)
		}
		return nd, nil
	}
	return o, nil
}

func dcAstuple(ii *Interp, o object.Object, seen map[uintptr]bool) (object.Object, error) {
	inst, ok := o.(*object.Instance)
	if !ok || !dcIsDataclass(o) {
		return o, nil
	}
	fields, _ := dcGetFields(inst)
	out := make([]object.Object, 0, len(fields))
	for _, f := range fields {
		nameObj, _ := f.Dict.GetStr("name")
		ns, ok2 := nameObj.(*object.Str)
		if !ok2 {
			continue
		}
		val, _ := inst.Dict.GetStr(ns.V)
		conv, err := dcAstupleConvert(ii, val, seen)
		if err != nil {
			return nil, err
		}
		out = append(out, conv)
	}
	return &object.Tuple{V: out}, nil
}

func dcAstupleConvert(ii *Interp, o object.Object, seen map[uintptr]bool) (object.Object, error) {
	if dcIsDataclass(o) {
		return dcAstuple(ii, o, seen)
	}
	switch v := o.(type) {
	case *object.List:
		out := make([]object.Object, len(v.V))
		for k, item := range v.V {
			conv, err := dcAstupleConvert(ii, item, seen)
			if err != nil {
				return nil, err
			}
			out[k] = conv
		}
		return &object.List{V: out}, nil
	case *object.Tuple:
		out := make([]object.Object, len(v.V))
		for k, item := range v.V {
			conv, err := dcAstupleConvert(ii, item, seen)
			if err != nil {
				return nil, err
			}
			out[k] = conv
		}
		return &object.Tuple{V: out}, nil
	}
	return o, nil
}

func dcReplace(ii *Interp, o object.Object, changes *object.Dict, missing *object.Instance) (object.Object, error) {
	inst, ok := o.(*object.Instance)
	if !ok || !dcIsDataclass(o) {
		return nil, object.Errorf(ii.typeErr, "replace() argument must be a dataclass instance")
	}
	newInst := &object.Instance{Class: inst.Class, Dict: object.NewDict()}
	fields, _ := dcGetFields(inst)
	for _, f := range fields {
		nameObj, _ := f.Dict.GetStr("name")
		ns, ok2 := nameObj.(*object.Str)
		if !ok2 {
			continue
		}
		val, _ := inst.Dict.GetStr(ns.V)
		newInst.Dict.SetStr(ns.V, val)
	}
	if changes != nil {
		keys, vals := changes.Items()
		for k, key := range keys {
			if ks, ok2 := key.(*object.Str); ok2 {
				newInst.Dict.SetStr(ks.V, vals[k])
			}
		}
	}
	return newInst, nil
}

func dcKwBool(kw *object.Dict, name string, dflt bool) bool {
	if kw == nil {
		return dflt
	}
	v, ok := kw.GetStr(name)
	if !ok {
		return dflt
	}
	b, ok2 := v.(*object.Bool)
	if !ok2 {
		return dflt
	}
	return b.V
}

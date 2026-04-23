package vm

import (
	"math/big"
	"strings"

	"github.com/tamnd/goipy/object"
)

// buildEnum constructs the `enum` module: Enum, IntEnum, StrEnum, Flag,
// IntFlag, auto, unique, and the functional Enum() API.
func (i *Interp) buildEnum() *object.Module {
	m := &object.Module{Name: "enum", Dict: object.NewDict()}

	// ---- auto sentinel class ----
	// auto() instances carry no data; the enum processor detects them by class.
	autoClass := &object.Class{Name: "auto", Dict: object.NewDict()}
	autoClass.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	m.Dict.SetStr("auto", autoClass)

	isAutoSentinel := func(o object.Object) bool {
		if inst, ok := o.(*object.Instance); ok {
			return inst.Class == autoClass
		}
		return false
	}

	// ---- forward declarations for base class pointers ----
	var (
		enumClass    *object.Class
		intEnumClass *object.Class
		strEnumClass *object.Class
		flagClass    *object.Class
		intFlagClass *object.Class
	)

	// ---- helper: get _name_ / _value_ from an enum member instance ----
	memberName := func(inst *object.Instance) string {
		if v, ok := inst.Dict.GetStr("_name_"); ok {
			if s, ok := v.(*object.Str); ok {
				return s.V
			}
		}
		return "?"
	}
	memberValue := func(inst *object.Instance) object.Object {
		if v, ok := inst.Dict.GetStr("_value_"); ok {
			return v
		}
		return object.None
	}

	// ---- generate auto() value ----
	genAuto := func(baseType, name string, count int) object.Object {
		switch baseType {
		case "StrEnum":
			return &object.Str{V: strings.ToLower(name)}
		case "Flag", "IntFlag":
			return object.NewInt(int64(1) << uint(count))
		default: // Enum, IntEnum
			return object.NewInt(int64(count + 1))
		}
	}

	// ---- detectBaseType: walk cls.Bases to find which enum base it inherits ----
	detectBaseType := func(cls *object.Class) string {
		for _, base := range cls.Bases {
			if base == intFlagClass {
				return "IntFlag"
			}
			if base == flagClass {
				return "Flag"
			}
			if base == intEnumClass {
				return "IntEnum"
			}
			if base == strEnumClass {
				return "StrEnum"
			}
		}
		return "Enum"
	}

	// ---- processEnumSubclass: transform class dict into enum members ----
	processEnumSubclass := func(cls *object.Class, baseType string) error {
		// Collect candidate member entries in dict insertion order.
		type entry struct {
			name string
			val  object.Object
		}
		keys, vals := cls.Dict.Items()
		var defs []entry
		for idx, k := range keys {
			ks, ok := k.(*object.Str)
			if !ok {
				continue
			}
			name := ks.V
			if strings.HasPrefix(name, "_") {
				continue
			}
			val := vals[idx]
			// Skip methods/functions/classes
			switch val.(type) {
			case *object.Function, *object.BuiltinFunc, *object.BoundMethod, *object.Class:
				continue
			}
			defs = append(defs, entry{name: name, val: val})
		}

		members := make([]*object.Instance, 0, len(defs))
		memberNames := make([]string, 0, len(defs))
		valMap := map[string]*object.Instance{}
		memberMap := object.NewDict()

		autoCount := 0
		for _, def := range defs {
			val := def.val
			if isAutoSentinel(val) {
				val = genAuto(baseType, def.name, autoCount)
				autoCount++
			} else {
				autoCount++
			}
			valKey := object.EnumValueKey(val)

			var mem *object.Instance
			if existing, ok := valMap[valKey]; ok {
				// Alias: same value as an existing canonical member.
				mem = existing
			} else {
				mem = &object.Instance{Class: cls, Dict: object.NewDict()}
				mem.Dict.SetStr("_name_", &object.Str{V: def.name})
				mem.Dict.SetStr("name", &object.Str{V: def.name})
				mem.Dict.SetStr("_value_", val)
				mem.Dict.SetStr("value", val)
				members = append(members, mem)
				memberNames = append(memberNames, def.name)
				valMap[valKey] = mem
			}
			memberMap.SetStr(def.name, mem)
			cls.Dict.SetStr(def.name, mem)
		}

		cls.EnumData = &object.EnumData{
			Members:     members,
			MemberMap:   memberMap,
			ValMap:      valMap,
			MemberNames: memberNames,
			BaseType:    baseType,
		}
		cls.Dict.SetStr("__members__", memberMap)
		return nil
	}

	// ---- enumLookupByValue: Color(1) → Color.RED ----
	enumLookupByValue := func(cls *object.Class, val object.Object) (object.Object, error) {
		key := object.EnumValueKey(val)
		if mem, ok := cls.EnumData.ValMap[key]; ok {
			return mem, nil
		}
		return nil, object.Errorf(i.valueErr, "%s is not a valid %s", object.Repr(val), cls.Name)
	}

	// ---- enumFunctionalCreate: Enum('Name', members) ----
	enumFunctionalCreate := func(baseCls *object.Class, name string, membersObj object.Object, baseType string) (object.Object, error) {
		newCls := &object.Class{Name: name, Bases: []*object.Class{baseCls}, Dict: object.NewDict()}

		switch mv := membersObj.(type) {
		case *object.List:
			// List of names (strings) or (name, value) tuples.
			autoIdx := 0
			for _, item := range mv.V {
				switch it := item.(type) {
				case *object.Str:
					newCls.Dict.SetStr(it.V, object.NewInt(int64(autoIdx+1)))
					autoIdx++
				case *object.Tuple:
					if len(it.V) == 2 {
						if ns, ok := it.V[0].(*object.Str); ok {
							newCls.Dict.SetStr(ns.V, it.V[1])
						}
					}
				}
			}
		case *object.Str:
			// Space- or comma-separated string of names.
			raw := mv.V
			raw = strings.ReplaceAll(raw, ",", " ")
			parts := strings.Fields(raw)
			for idx, p := range parts {
				newCls.Dict.SetStr(p, object.NewInt(int64(idx+1)))
			}
		case *object.Dict:
			// Dict of {name: value}.
			keys, vals := mv.Items()
			for idx, k := range keys {
				if ks, ok := k.(*object.Str); ok {
					newCls.Dict.SetStr(ks.V, vals[idx])
				}
			}
		default:
			return nil, object.Errorf(i.typeErr, "Enum() second argument must be a list, string, or dict")
		}

		if err := processEnumSubclass(newCls, baseType); err != nil {
			return nil, err
		}
		return newCls, nil
	}

	// ---- Enum base class ----
	enumClass = &object.Class{Name: "Enum", Dict: object.NewDict()}

	// __new__: handles both value lookup (Color(1)) and functional creation (Enum('Name', [...]))
	enumClass.Dict.SetStr("__new__", &object.BuiltinFunc{Name: "__new__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "Enum.__new__ requires cls")
		}
		cls, ok := a[0].(*object.Class)
		if !ok {
			return nil, object.Errorf(i.typeErr, "Enum.__new__: first arg must be a class")
		}
		if len(a) >= 2 {
			if cls.EnumData != nil {
				// Subclass value lookup: Color(1)
				return enumLookupByValue(cls, a[1])
			}
			// Functional API: Enum('Animal', [...]) — first arg is cls=Enum, a[1]=name, a[2]=members
			if nameStr, ok := a[1].(*object.Str); ok && len(a) >= 3 {
				baseType := detectBaseType(cls)
				if cls == enumClass {
					baseType = "Enum"
				} else if cls == intEnumClass {
					baseType = "IntEnum"
				} else if cls == strEnumClass {
					baseType = "StrEnum"
				} else if cls == flagClass {
					baseType = "Flag"
				} else if cls == intFlagClass {
					baseType = "IntFlag"
				}
				return enumFunctionalCreate(cls, nameStr.V, a[2], baseType)
			}
		}
		// Direct instantiation — create a bare instance (should not normally happen for users).
		return &object.Instance{Class: cls, Dict: object.NewDict()}, nil
	}})

	// __init__: no-op (members are created by __new__ / __init_subclass__)
	enumClass.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// __init_subclass__: transforms the new class dict into enum members
	enumClass.Dict.SetStr("__init_subclass__", &object.BuiltinFunc{Name: "__init_subclass__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		cls, ok := a[0].(*object.Class)
		if !ok {
			return object.None, nil
		}
		// Skip if this IS one of our own base classes (prevents processing IntEnum etc.)
		if cls == intEnumClass || cls == strEnumClass || cls == flagClass || cls == intFlagClass {
			return object.None, nil
		}
		baseType := detectBaseType(cls)
		if err := processEnumSubclass(cls, baseType); err != nil {
			return nil, err
		}
		return object.None, nil
	}})

	// __repr__: <Color.RED: 1>
	enumClass.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "<enum>"}, nil
		}
		return &object.Str{V: "<" + inst.Class.Name + "." + memberName(inst) + ": " + object.Repr(memberValue(inst)) + ">"}, nil
	}})

	// __str__: Color.RED
	enumClass.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "?"}, nil
		}
		return &object.Str{V: inst.Class.Name + "." + memberName(inst)}, nil
	}})

	// __eq__: identity comparison
	enumClass.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return &object.Bool{V: false}, nil
		}
		return &object.Bool{V: a[0] == a[1]}, nil
	}})

	// __ne__
	enumClass.Dict.SetStr("__ne__", &object.BuiltinFunc{Name: "__ne__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return &object.Bool{V: true}, nil
		}
		return &object.Bool{V: a[0] != a[1]}, nil
	}})

	// __hash__: identity-based
	enumClass.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		return object.NewInt(int64(objectID(a[0]) >> 4)), nil
	}})

	m.Dict.SetStr("Enum", enumClass)

	// ---- IntEnum ----
	intEnumClass = &object.Class{Name: "IntEnum", Bases: []*object.Class{enumClass}, Dict: object.NewDict()}

	// __repr__: <Number.ONE: 1> (same as Enum)
	// Inherited from enumClass.

	// __str__: returns the int value as string ("1" not "Number.ONE")
	intEnumClass.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "?"}, nil
		}
		return &object.Str{V: object.Str_(memberValue(inst))}, nil
	}})

	// __int__
	intEnumClass.Dict.SetStr("__int__", &object.BuiltinFunc{Name: "__int__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		n, ok2 := toInt64(memberValue(inst))
		if !ok2 {
			return object.NewInt(0), nil
		}
		return object.NewInt(n), nil
	}})

	// __eq__: compare value with int
	intEnumClass.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return &object.Bool{V: false}, nil
		}
		if a[0] == a[1] {
			return &object.Bool{V: true}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Bool{V: false}, nil
		}
		eq, err := object.Eq(memberValue(inst), a[1])
		if err != nil {
			return &object.Bool{V: false}, nil
		}
		return &object.Bool{V: eq}, nil
	}})

	// __hash__: value-based for int compatibility
	intEnumClass.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		n, _ := toInt64(memberValue(inst))
		return object.NewInt(n), nil
	}})

	cmpIntEnum := func(a []object.Object) (int64, int64, bool) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return 0, 0, false
		}
		lv, ok1 := toInt64(memberValue(inst))
		rv, ok2 := toInt64(a[1])
		if !ok1 {
			return 0, 0, false
		}
		if !ok2 {
			if inst2, ok3 := a[1].(*object.Instance); ok3 {
				rv, ok2 = toInt64(memberValue(inst2))
			}
		}
		return lv, rv, ok2
	}

	intEnumClass.Dict.SetStr("__lt__", &object.BuiltinFunc{Name: "__lt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		lv, rv, ok := cmpIntEnum(a)
		if !ok {
			return object.NotImplemented, nil
		}
		return &object.Bool{V: lv < rv}, nil
	}})
	intEnumClass.Dict.SetStr("__le__", &object.BuiltinFunc{Name: "__le__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		lv, rv, ok := cmpIntEnum(a)
		if !ok {
			return object.NotImplemented, nil
		}
		return &object.Bool{V: lv <= rv}, nil
	}})
	intEnumClass.Dict.SetStr("__gt__", &object.BuiltinFunc{Name: "__gt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		lv, rv, ok := cmpIntEnum(a)
		if !ok {
			return object.NotImplemented, nil
		}
		return &object.Bool{V: lv > rv}, nil
	}})
	intEnumClass.Dict.SetStr("__ge__", &object.BuiltinFunc{Name: "__ge__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		lv, rv, ok := cmpIntEnum(a)
		if !ok {
			return object.NotImplemented, nil
		}
		return &object.Bool{V: lv >= rv}, nil
	}})

	// __add__: returns plain int
	intEnumClass.Dict.SetStr("__add__", &object.BuiltinFunc{Name: "__add__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "unsupported operand type")
		}
		lv, ok1 := toInt64(memberValue(inst))
		rv, ok2 := toInt64(a[1])
		if !ok1 || !ok2 {
			// try big.Int
			lbi := new(big.Int)
			if n, ok := toInt64(memberValue(inst)); ok {
				lbi.SetInt64(n)
			}
			rbi := new(big.Int)
			if n, ok := toInt64(a[1]); ok {
				rbi.SetInt64(n)
			}
			result := new(big.Int).Add(lbi, rbi)
			ri := object.Int{}
			ri.V = *result
			return &ri, nil
		}
		return object.NewInt(lv + rv), nil
	}})

	m.Dict.SetStr("IntEnum", intEnumClass)

	// ---- StrEnum ----
	strEnumClass = &object.Class{Name: "StrEnum", Bases: []*object.Class{enumClass}, Dict: object.NewDict()}

	// __str__: returns the string value ("happy" not "Mood.HAPPY")
	strEnumClass.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: ""}, nil
		}
		return &object.Str{V: object.Str_(memberValue(inst))}, nil
	}})

	// __eq__: compare value with str
	strEnumClass.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return &object.Bool{V: false}, nil
		}
		if a[0] == a[1] {
			return &object.Bool{V: true}, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Bool{V: false}, nil
		}
		eq, err := object.Eq(memberValue(inst), a[1])
		if err != nil {
			return &object.Bool{V: false}, nil
		}
		return &object.Bool{V: eq}, nil
	}})

	// __hash__
	strEnumClass.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		s, _ := memberValue(inst).(*object.Str)
		if s == nil {
			return object.NewInt(0), nil
		}
		// simple string hash
		h := int64(0)
		for _, r := range s.V {
			h = h*31 + int64(r)
		}
		return object.NewInt(h), nil
	}})

	m.Dict.SetStr("StrEnum", strEnumClass)

	// ---- Flag ----
	flagClass = &object.Class{Name: "Flag", Bases: []*object.Class{enumClass}, Dict: object.NewDict()}

	// __repr__: <Perm.READ: 1> or <Perm.READ|WRITE: 3>
	flagClass.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "<flag>"}, nil
		}
		return &object.Str{V: "<" + inst.Class.Name + "." + memberName(inst) + ": " + object.Repr(memberValue(inst)) + ">"}, nil
	}})

	// __str__: Perm.READ|WRITE (no angle brackets)
	flagClass.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "?"}, nil
		}
		return &object.Str{V: inst.Class.Name + "." + memberName(inst)}, nil
	}})

	// __or__: bitwise OR to create composite flags
	flagClass.Dict.SetStr("__or__", &object.BuiltinFunc{Name: "__or__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		lhs, ok1 := a[0].(*object.Instance)
		rhs, ok2 := a[1].(*object.Instance)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "unsupported operand types for |")
		}
		lv, _ := toInt64(memberValue(lhs))
		rv, _ := toInt64(memberValue(rhs))
		combined := lv | rv
		cls := lhs.Class

		valKey := object.EnumValueKey(object.NewInt(combined))
		if cls.EnumData != nil {
			if mem, ok := cls.EnumData.ValMap[valKey]; ok {
				return mem, nil
			}
		}
		// Build composite member.
		composite := &object.Instance{Class: cls, Dict: object.NewDict()}
		var names []string
		if cls.EnumData != nil {
			for _, mem := range cls.EnumData.Members {
				mInt, _ := toInt64(memberValue(mem))
				if mInt != 0 && (combined&mInt) == mInt {
					names = append(names, memberName(mem))
				}
			}
		}
		combinedVal := object.NewInt(combined)
		combinedName := strings.Join(names, "|")
		composite.Dict.SetStr("_name_", &object.Str{V: combinedName})
		composite.Dict.SetStr("name", &object.Str{V: combinedName})
		composite.Dict.SetStr("_value_", combinedVal)
		composite.Dict.SetStr("value", combinedVal)
		return composite, nil
	}})

	// __and__: bitwise AND
	flagClass.Dict.SetStr("__and__", &object.BuiltinFunc{Name: "__and__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		lhs, ok1 := a[0].(*object.Instance)
		rhs, ok2 := a[1].(*object.Instance)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "unsupported operand types for &")
		}
		lv, _ := toInt64(memberValue(lhs))
		rv, _ := toInt64(memberValue(rhs))
		combined := lv & rv
		cls := lhs.Class
		valKey := object.EnumValueKey(object.NewInt(combined))
		if cls.EnumData != nil {
			if mem, ok := cls.EnumData.ValMap[valKey]; ok {
				return mem, nil
			}
		}
		// Return zero-value instance.
		zero := &object.Instance{Class: cls, Dict: object.NewDict()}
		zero.Dict.SetStr("_name_", &object.Str{V: ""})
		zero.Dict.SetStr("_value_", object.NewInt(combined))
		return zero, nil
	}})

	// __contains__: Perm.READ in rw  (rw is a Flag instance)
	flagClass.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self, ok1 := a[0].(*object.Instance)
		needle, ok2 := a[1].(*object.Instance)
		if !ok1 || !ok2 {
			return &object.Bool{V: false}, nil
		}
		sv, _ := toInt64(memberValue(self))
		nv, _ := toInt64(memberValue(needle))
		if nv == 0 {
			return &object.Bool{V: false}, nil
		}
		return &object.Bool{V: (sv & nv) == nv}, nil
	}})

	// __bool__
	flagClass.Dict.SetStr("__bool__", &object.BuiltinFunc{Name: "__bool__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Bool{V: false}, nil
		}
		n, _ := toInt64(memberValue(inst))
		return &object.Bool{V: n != 0}, nil
	}})

	m.Dict.SetStr("Flag", flagClass)

	// ---- IntFlag ----
	intFlagClass = &object.Class{Name: "IntFlag", Bases: []*object.Class{flagClass, intEnumClass}, Dict: object.NewDict()}

	// __str__: int value (like IntEnum)
	intFlagClass.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "?"}, nil
		}
		return &object.Str{V: object.Str_(memberValue(inst))}, nil
	}})

	m.Dict.SetStr("IntFlag", intFlagClass)

	// ---- @unique decorator ----
	m.Dict.SetStr("unique", &object.BuiltinFunc{Name: "unique", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "unique() requires an enum class")
		}
		cls, ok := a[0].(*object.Class)
		if !ok || cls.EnumData == nil {
			return a[0], nil
		}
		// Check for duplicate values: valMap size < member definitions count indicates aliases.
		keys, vals := cls.EnumData.MemberMap.Items()
		seen := map[string]string{} // valueKey -> first name
		for idx, k := range keys {
			ks, ok := k.(*object.Str)
			if !ok {
				continue
			}
			mem, ok := vals[idx].(*object.Instance)
			if !ok {
				continue
			}
			valKey := object.EnumValueKey(memberValue(mem))
			if firstName, exists := seen[valKey]; exists {
				return nil, object.Errorf(i.valueErr, "duplicate values found in %s: %s -> %s", cls.Name, ks.V, firstName)
			}
			seen[valKey] = ks.V
		}
		return cls, nil
	}})

	// ---- module-level aliases ----
	m.Dict.SetStr("EnumMeta", enumClass)
	m.Dict.SetStr("EnumType", enumClass)

	return m
}

package object

// EnumData holds the runtime metadata for an enum class (Color, Number, etc.).
// Non-nil only on classes created as enum subclasses.
type EnumData struct {
	Members     []*Instance          // canonical members in definition order
	MemberMap   *Dict                // all names -> member (including aliases)
	ValMap      map[string]*Instance // enumValueKey(value) -> canonical member
	MemberNames []string             // canonical member names in definition order
	BaseType    string               // "Enum", "IntEnum", "StrEnum", "Flag", "IntFlag"
}

// EnumValueKey produces a stable string key for an enum member value,
// used to de-duplicate entries by value in ValMap.
func EnumValueKey(o Object) string {
	switch v := o.(type) {
	case *Bool:
		if v.V {
			return "i1"
		}
		return "i0"
	case *Int:
		return "i" + v.V.String()
	case *Str:
		return "s" + v.V
	default:
		return "r" + Repr(o)
	}
}

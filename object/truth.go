package object

// Truthy reports Python truth value of o.
func Truthy(o Object) bool {
	switch v := o.(type) {
	case nil, *NoneType:
		return false
	case *Bool:
		return v.V
	case *Int:
		return v.V.Sign() != 0
	case *Float:
		return v.V != 0
	case *Complex:
		return v.Real != 0 || v.Imag != 0
	case *Str:
		return v.V != ""
	case *Bytes:
		return len(v.V) != 0
	case *Bytearray:
		return len(v.V) != 0
	case *Memoryview:
		return v.Stop-v.Start != 0
	case *Tuple:
		return len(v.V) != 0
	case *List:
		return len(v.V) != 0
	case *Dict:
		return v.Len() != 0
	case *Set:
		return v.Len() != 0
	case *Frozenset:
		return v.Len() != 0
	case *Range:
		if v.Step > 0 {
			return v.Stop > v.Start
		}
		if v.Step < 0 {
			return v.Stop < v.Start
		}
		return false
	}
	return true
}

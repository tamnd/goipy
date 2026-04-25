package vm

import "github.com/tamnd/goipy/object"

// toBytes extracts a byte slice from a Python bytes/bytearray/str object.
func toBytes(obj object.Object) []byte {
	switch v := obj.(type) {
	case *object.Bytes:
		return v.V
	case *object.Bytearray:
		return v.V
	case *object.Str:
		return []byte(v.V)
	}
	return nil
}

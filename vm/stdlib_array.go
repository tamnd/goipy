package vm

import (
	"encoding/binary"
	"fmt"
	"math"
	"github.com/tamnd/goipy/object"
)

// buildArray constructs the array module exposing array.array and array.typecodes.
func (i *Interp) buildArray() *object.Module {
	m := &object.Module{Name: "array", Dict: object.NewDict()}

	// typecodes matches Python 3.14 on this platform.
	m.Dict.SetStr("typecodes", &object.Str{V: "bBuwhHiIlLqQfd"})

	arrayCls := &object.Class{Name: "array", Dict: object.NewDict()}

	constructor := &object.BuiltinFunc{Name: "array", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "array() requires at least 1 argument (typecode)")
		}
		tc, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "array() argument 1 must be a unicode character, not '%s'", object.TypeName(a[0]))
		}
		if len(tc.V) != 1 {
			return nil, object.Errorf(i.valueErr, "array typecode must be a single character")
		}
		tcode := tc.V
		if !isValidTypecode(tcode) {
			return nil, object.Errorf(i.valueErr, "bad typecode (must be b, B, u, h, H, i, I, l, L, q, Q, f or d)")
		}
		arr := &object.PyArray{Typecode: tcode}
		if len(a) >= 2 {
			// initializer: iterable of values OR bytes
			init := a[1]
			if b2, ok2 := init.(*object.Bytes); ok2 {
				if err := arrayFromBytes(i, arr, b2.V); err != nil {
					return nil, err
				}
			} else if ba, ok2 := init.(*object.Bytearray); ok2 {
				if err := arrayFromBytes(i, arr, ba.V); err != nil {
					return nil, err
				}
			} else {
				items, err := iterate(i, init)
				if err != nil {
					return nil, err
				}
				for _, x := range items {
					v, err := arrayValidate(tcode, x)
					if err != nil {
						return nil, object.Errorf(i.typeErr, "%s", err.Error())
					}
					arr.V = append(arr.V, v)
				}
			}
		}
		return arr, nil
	}}
	// Make isinstance(x, array.array) work via ABCCheck.
	arrayCls.ABCCheck = func(o object.Object) bool {
		_, ok := o.(*object.PyArray)
		return ok
	}
	m.Dict.SetStr("array", arrayCls)
	// Also expose the constructor under the class dict so array.array(…) works.
	arrayCls.Dict.SetStr("__call__", constructor)
	// Make array.array(…) callable by using the constructor as the module-level value.
	// We register a BuiltinFunc that also satisfies isinstance checks via the class.
	m.Dict.SetStr("array", &object.BuiltinFunc{Name: "array", Call: constructor.Call})

	return m
}

func isValidTypecode(tc string) bool {
	switch tc {
	case "b", "B", "h", "H", "i", "I", "l", "L", "q", "Q", "f", "d":
		return true
	}
	return false
}

// arrayValidate coerces v into the correct Python type for typecode tc.
// Returns TypeError/OverflowError as a Go error (caller wraps into object.Errorf).
func arrayValidate(tc string, v object.Object) (object.Object, error) {
	switch tc {
	case "b":
		n, ok := toInt64(v)
		if !ok {
			return nil, fmt.Errorf("an integer is required")
		}
		if n < -128 || n > 127 {
			return nil, fmt.Errorf("signed char is greater than maximum")
		}
		return object.NewInt(n), nil
	case "B":
		n, ok := toInt64(v)
		if !ok {
			return nil, fmt.Errorf("an integer is required")
		}
		if n < 0 || n > 255 {
			return nil, fmt.Errorf("unsigned byte is greater than maximum")
		}
		return object.NewInt(n), nil
	case "h":
		n, ok := toInt64(v)
		if !ok {
			return nil, fmt.Errorf("an integer is required")
		}
		if n < -32768 || n > 32767 {
			return nil, fmt.Errorf("signed short is out of range")
		}
		return object.NewInt(n), nil
	case "H":
		n, ok := toInt64(v)
		if !ok {
			return nil, fmt.Errorf("an integer is required")
		}
		if n < 0 || n > 65535 {
			return nil, fmt.Errorf("unsigned short is out of range")
		}
		return object.NewInt(n), nil
	case "i":
		n, ok := toInt64(v)
		if !ok {
			return nil, fmt.Errorf("an integer is required")
		}
		if n < math.MinInt32 || n > math.MaxInt32 {
			return nil, fmt.Errorf("signed int is out of range")
		}
		return object.NewInt(n), nil
	case "I":
		n, ok := toInt64(v)
		if !ok {
			return nil, fmt.Errorf("an integer is required")
		}
		if n < 0 || n > math.MaxUint32 {
			return nil, fmt.Errorf("unsigned int is out of range")
		}
		return object.NewInt(n), nil
	case "l":
		bi, ok := toBigInt(v)
		if !ok {
			return nil, fmt.Errorf("an integer is required")
		}
		n := bi.Int64()
		return object.NewInt(n), nil
	case "L":
		bi, ok := toBigInt(v)
		if !ok {
			return nil, fmt.Errorf("an integer is required")
		}
		if bi.Sign() < 0 {
			return nil, fmt.Errorf("unsigned long is out of range")
		}
		return object.IntFromBig(bi), nil
	case "q":
		bi, ok := toBigInt(v)
		if !ok {
			return nil, fmt.Errorf("an integer is required")
		}
		return object.NewInt(bi.Int64()), nil
	case "Q":
		bi, ok := toBigInt(v)
		if !ok {
			return nil, fmt.Errorf("an integer is required")
		}
		if bi.Sign() < 0 {
			return nil, fmt.Errorf("unsigned long long is out of range")
		}
		// Preserve the full uint64 range using big.Int storage.
		return object.IntFromBig(bi), nil
	case "f":
		_, _, isFloat, ok := asIntOrFloat(v)
		if !ok {
			return nil, fmt.Errorf("a real number is required")
		}
		_ = isFloat
		ibig, fv, isF, _ := asIntOrFloat(v)
		var f64 float64
		if isF {
			f64 = fv
		} else {
			f64 = toFloat(ibig, 0, false)
		}
		// Round-trip through float32 to match Python's 'f' typecode behavior.
		f32 := float32(f64)
		return &object.Float{V: float64(f32)}, nil
	case "d":
		ibig, fv, isF, ok := asIntOrFloat(v)
		if !ok {
			return nil, fmt.Errorf("a real number is required")
		}
		var f64 float64
		if isF {
			f64 = fv
		} else {
			f64 = toFloat(ibig, 0, false)
		}
		return &object.Float{V: f64}, nil
	}
	return nil, fmt.Errorf("bad typecode")
}

// arrayItemBytes serializes one element into raw bytes (little-endian / native).
func arrayItemBytes(tc string, v object.Object) []byte {
	sz := object.ArrayItemSize(tc)
	buf := make([]byte, sz)
	switch tc {
	case "b":
		n, _ := toInt64(v)
		buf[0] = byte(int8(n))
	case "B":
		n, _ := toInt64(v)
		buf[0] = byte(n)
	case "h":
		n, _ := toInt64(v)
		binary.NativeEndian.PutUint16(buf, uint16(int16(n)))
	case "H":
		n, _ := toInt64(v)
		binary.NativeEndian.PutUint16(buf, uint16(n))
	case "i":
		n, _ := toInt64(v)
		binary.NativeEndian.PutUint32(buf, uint32(int32(n)))
	case "I":
		n, _ := toInt64(v)
		binary.NativeEndian.PutUint32(buf, uint32(n))
	case "l":
		n, _ := toInt64(v)
		binary.NativeEndian.PutUint64(buf, uint64(n))
	case "L":
		n, _ := toInt64(v)
		binary.NativeEndian.PutUint64(buf, uint64(n))
	case "q":
		n, _ := toInt64(v)
		binary.NativeEndian.PutUint64(buf, uint64(n))
	case "Q":
		n, _ := toInt64(v)
		binary.NativeEndian.PutUint64(buf, uint64(n))
	case "f":
		f := v.(*object.Float).V
		binary.NativeEndian.PutUint32(buf, math.Float32bits(float32(f)))
	case "d":
		f := v.(*object.Float).V
		binary.NativeEndian.PutUint64(buf, math.Float64bits(f))
	}
	return buf
}

// arrayFromBytes appends elements decoded from raw bytes b into arr.
func arrayFromBytes(i *Interp, arr *object.PyArray, b []byte) error {
	sz := object.ArrayItemSize(arr.Typecode)
	if len(b)%sz != 0 {
		return object.Errorf(i.valueErr, "bytes length not a multiple of item size")
	}
	for off := 0; off < len(b); off += sz {
		chunk := b[off : off+sz]
		var v object.Object
		switch arr.Typecode {
		case "b":
			v = object.NewInt(int64(int8(chunk[0])))
		case "B":
			v = object.NewInt(int64(chunk[0]))
		case "h":
			v = object.NewInt(int64(int16(binary.NativeEndian.Uint16(chunk))))
		case "H":
			v = object.NewInt(int64(binary.NativeEndian.Uint16(chunk)))
		case "i":
			v = object.NewInt(int64(int32(binary.NativeEndian.Uint32(chunk))))
		case "I":
			v = object.NewInt(int64(binary.NativeEndian.Uint32(chunk)))
		case "l":
			v = object.NewInt(int64(binary.NativeEndian.Uint64(chunk)))
		case "L":
			v = object.NewInt(int64(binary.NativeEndian.Uint64(chunk)))
		case "q":
			v = object.NewInt(int64(binary.NativeEndian.Uint64(chunk)))
		case "Q":
			v = object.NewInt(int64(binary.NativeEndian.Uint64(chunk)))
		case "f":
			v = &object.Float{V: float64(math.Float32frombits(binary.NativeEndian.Uint32(chunk)))}
		case "d":
			v = &object.Float{V: math.Float64frombits(binary.NativeEndian.Uint64(chunk))}
		}
		arr.V = append(arr.V, v)
	}
	return nil
}

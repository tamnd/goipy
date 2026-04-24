package vm

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"math"
	"math/big"
	"net/url"
	"strconv"
	"strings"

	"github.com/tamnd/goipy/object"
)

// --- struct module ---------------------------------------------------------

type structItem struct {
	kind byte
	n    int // repeat count, or byte length for 's'
}

// parseStructFormat splits a format string into a byte order + item list.
// Unknown characters or stray digits raise. The grammar is permissive about
// whitespace, matching CPython.
func parseStructFormat(fmt string) (binary.ByteOrder, bool, []structItem, error) {
	var order binary.ByteOrder = binary.LittleEndian
	native := false
	items := make([]structItem, 0, len(fmt))
	i := 0
	if i < len(fmt) {
		switch fmt[i] {
		case '<':
			order = binary.LittleEndian
			i++
		case '>', '!':
			order = binary.BigEndian
			i++
		case '=':
			order = binary.LittleEndian
			i++
		case '@':
			order = binary.LittleEndian
			native = true
			i++
		default:
			native = true
		}
	}
	_ = native
	for i < len(fmt) {
		c := fmt[i]
		if c == ' ' || c == '\t' {
			i++
			continue
		}
		n := 1
		if c >= '0' && c <= '9' {
			n = 0
			for i < len(fmt) && fmt[i] >= '0' && fmt[i] <= '9' {
				n = n*10 + int(fmt[i]-'0')
				i++
			}
			if i >= len(fmt) {
				return nil, false, nil, fmt2err("repeat count given without format specifier")
			}
			c = fmt[i]
		}
		i++
		switch c {
		case 'x', 'c', 'b', 'B', '?', 'h', 'H', 'i', 'I', 'l', 'L', 'q', 'Q', 'e', 'f', 'd':
			for k := 0; k < n; k++ {
				items = append(items, structItem{kind: c, n: 1})
			}
		case 's', 'p':
			items = append(items, structItem{kind: c, n: n})
		default:
			return nil, false, nil, fmt2err("bad char in struct format: " + string(c))
		}
	}
	return order, native, items, nil
}

func fmt2err(msg string) error { return fmt.Errorf("%s", msg) }

func structItemSize(it structItem) int {
	switch it.kind {
	case 'x', 'c', 'b', 'B', '?':
		return 1
	case 'h', 'H', 'e':
		return 2
	case 'i', 'I', 'l', 'L', 'f':
		return 4
	case 'q', 'Q', 'd':
		return 8
	case 's', 'p':
		return it.n
	}
	return 0
}

// float16Encode converts float32 to IEEE 754 half-precision bits.
func float16Encode(f float32) uint16 {
	b := math.Float32bits(f)
	sign := uint16(b>>31) << 15
	exp := int((b>>23)&0xFF) - 127
	mantissa := b & 0x7FFFFF
	if exp == 128 { // inf/nan
		return sign | 0x7C00 | uint16(mantissa>>13)
	}
	exp16 := exp + 15
	if exp16 >= 31 {
		return sign | 0x7C00 // overflow → inf
	}
	if exp16 <= 0 {
		if exp16 < -10 {
			return sign // underflow → zero
		}
		m := uint16((mantissa | 0x800000) >> (14 - exp16))
		return sign | m
	}
	return sign | uint16(exp16)<<10 | uint16(mantissa>>13)
}

// float16Decode converts IEEE 754 half-precision bits to float32.
func float16Decode(b uint16) float32 {
	sign := uint32(b>>15) << 31
	exp := uint32(b>>10) & 0x1F
	mantissa := uint32(b & 0x3FF)
	var bits uint32
	switch exp {
	case 31: // inf/nan
		bits = sign | 0x7F800000 | mantissa<<13
	case 0: // zero/subnormal
		if mantissa == 0 {
			bits = sign
		} else {
			e, m := uint32(0), mantissa
			for m&0x400 == 0 {
				m <<= 1
				e++
			}
			bits = sign | (127-15-e+1)<<23 | (m&0x3FF)<<13
		}
	default:
		bits = sign | (exp+127-15)<<23 | mantissa<<13
	}
	return math.Float32frombits(bits)
}

func structTotalSize(items []structItem) int {
	n := 0
	for _, it := range items {
		n += structItemSize(it)
	}
	return n
}

func (i *Interp) buildStruct() *object.Module {
	m := &object.Module{Name: "struct", Dict: object.NewDict()}

	// struct.error is the canonical exception for all struct errors.
	errCls := &object.Class{Name: "error", Dict: object.NewDict(), Bases: []*object.Class{i.exception}}
	m.Dict.SetStr("error", errCls)
	se := func(msg string, a ...any) error { return object.Errorf(errCls, msg, a...) }

	parseFormat := func(fmtStr string) (binary.ByteOrder, []structItem, error) {
		order, _, items, err := parseStructFormat(fmtStr)
		if err != nil {
			return nil, nil, se("%s", err.Error())
		}
		return order, items, nil
	}

	m.Dict.SetStr("calcsize", &object.BuiltinFunc{Name: "calcsize", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, se("calcsize() argument must be str")
		}
		_, items, err := parseFormat(s.V)
		if err != nil {
			return nil, err
		}
		return object.NewInt(int64(structTotalSize(items))), nil
	}})

	m.Dict.SetStr("pack", &object.BuiltinFunc{Name: "pack", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, se("pack() missing format")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, se("pack() format must be str")
		}
		order, items, err := parseFormat(s.V)
		if err != nil {
			return nil, err
		}
		out, err := structPack(errCls, order, items, a[1:])
		if err != nil {
			return nil, err
		}
		return &object.Bytes{V: out}, nil
	}})

	m.Dict.SetStr("pack_into", &object.BuiltinFunc{Name: "pack_into", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, se("pack_into() requires format, buffer, offset")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, se("pack_into() format must be str")
		}
		ba, ok2 := a[1].(*object.Bytearray)
		if !ok2 {
			return nil, se("pack_into() buffer must be a writable bytes-like object")
		}
		off, ok3 := toInt64(a[2])
		if !ok3 {
			return nil, se("pack_into() offset must be int")
		}
		order, items, err := parseFormat(s.V)
		if err != nil {
			return nil, err
		}
		packed, err := structPack(errCls, order, items, a[3:])
		if err != nil {
			return nil, err
		}
		start := int(off)
		if start < 0 || start+len(packed) > len(ba.V) {
			return nil, se("pack_into requires a buffer of at least %d bytes", start+len(packed))
		}
		copy(ba.V[start:], packed)
		return object.None, nil
	}})

	unpackFn := func(name string, exact bool) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, se("%s() missing format or buffer", name)
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, se("%s() format must be str", name)
			}
			data, err := asBytes(a[1])
			if err != nil {
				return nil, se("%s() buffer must be bytes-like", name)
			}
			offset := 0
			if len(a) >= 3 {
				n, ok2 := toInt64(a[2])
				if !ok2 {
					return nil, se("%s() offset must be int", name)
				}
				offset = int(n)
			}
			order, items, err2 := parseFormat(s.V)
			if err2 != nil {
				return nil, err2
			}
			need := structTotalSize(items)
			if exact && offset == 0 && len(data) != need {
				return nil, se("unpack requires a buffer of %d bytes", need)
			}
			if offset < 0 || offset+need > len(data) {
				return nil, se("%s requires a buffer of at least %d bytes for unpacking", name, need)
			}
			vals, err3 := structUnpack(order, items, data[offset:offset+need])
			if err3 != nil {
				return nil, err3
			}
			return &object.Tuple{V: vals}, nil
		}}
	}
	m.Dict.SetStr("unpack", unpackFn("unpack", true))
	m.Dict.SetStr("unpack_from", unpackFn("unpack_from", false))

	m.Dict.SetStr("iter_unpack", &object.BuiltinFunc{Name: "iter_unpack", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, se("iter_unpack() missing format or buffer")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, se("iter_unpack() format must be str")
		}
		data, err := asBytes(a[1])
		if err != nil {
			return nil, se("iter_unpack() buffer must be bytes-like")
		}
		order, items, err2 := parseFormat(s.V)
		if err2 != nil {
			return nil, err2
		}
		itemSize := structTotalSize(items)
		if itemSize == 0 {
			return nil, se("iter_unpack() format size is 0")
		}
		if len(data)%itemSize != 0 {
			return nil, se("iterative unpacking requires a buffer whose length is a multiple of %d", itemSize)
		}
		buf := append([]byte(nil), data...)
		pos := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if pos >= len(buf) {
				return nil, false, nil
			}
			vals, err3 := structUnpack(order, items, buf[pos:pos+itemSize])
			if err3 != nil {
				return nil, false, err3
			}
			pos += itemSize
			return &object.Tuple{V: vals}, true, nil
		}}, nil
	}})

	// Struct class — compiled format object.
	structCls := &object.Class{Name: "Struct", Dict: object.NewDict()}
	structCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		fmtStr, ok2 := a[1].(*object.Str)
		if !ok2 {
			return nil, se("Struct() format must be str")
		}
		_, items, err := parseFormat(fmtStr.V)
		if err != nil {
			return nil, err
		}
		sz := structTotalSize(items)

		// Store format and size in instance dict.
		self.Dict.SetStr("format", fmtStr)
		self.Dict.SetStr("size", object.NewInt(int64(sz)))

		// All methods stored in instance dict to avoid AttrCacheClassValue issue.
		self.Dict.SetStr("pack", &object.BuiltinFunc{Name: "pack", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			order, its, err2 := parseFormat(fmtStr.V)
			if err2 != nil {
				return nil, err2
			}
			out, err3 := structPack(errCls, order, its, args)
			if err3 != nil {
				return nil, err3
			}
			return &object.Bytes{V: out}, nil
		}})

		self.Dict.SetStr("unpack", &object.BuiltinFunc{Name: "unpack", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if len(args) < 1 {
				return nil, se("unpack() missing buffer")
			}
			data, err2 := asBytes(args[0])
			if err2 != nil {
				return nil, se("unpack() buffer must be bytes-like")
			}
			if len(data) != sz {
				return nil, se("unpack requires a buffer of %d bytes", sz)
			}
			order, its, err3 := parseFormat(fmtStr.V)
			if err3 != nil {
				return nil, err3
			}
			vals, err4 := structUnpack(order, its, data)
			if err4 != nil {
				return nil, err4
			}
			return &object.Tuple{V: vals}, nil
		}})

		self.Dict.SetStr("unpack_from", &object.BuiltinFunc{Name: "unpack_from", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if len(args) < 1 {
				return nil, se("unpack_from() missing buffer")
			}
			data, err2 := asBytes(args[0])
			if err2 != nil {
				return nil, se("unpack_from() buffer must be bytes-like")
			}
			offset := 0
			if len(args) >= 2 {
				n, ok3 := toInt64(args[1])
				if !ok3 {
					return nil, se("unpack_from() offset must be int")
				}
				offset = int(n)
			}
			if offset < 0 || offset+sz > len(data) {
				return nil, se("unpack_from requires a buffer of at least %d bytes", sz)
			}
			order, its, err3 := parseFormat(fmtStr.V)
			if err3 != nil {
				return nil, err3
			}
			vals, err4 := structUnpack(order, its, data[offset:offset+sz])
			if err4 != nil {
				return nil, err4
			}
			return &object.Tuple{V: vals}, nil
		}})

		self.Dict.SetStr("pack_into", &object.BuiltinFunc{Name: "pack_into", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if len(args) < 2 {
				return nil, se("pack_into() requires buffer and offset")
			}
			ba, ok3 := args[0].(*object.Bytearray)
			if !ok3 {
				return nil, se("pack_into() buffer must be a writable bytes-like object")
			}
			off, ok4 := toInt64(args[1])
			if !ok4 {
				return nil, se("pack_into() offset must be int")
			}
			order, its, err2 := parseFormat(fmtStr.V)
			if err2 != nil {
				return nil, err2
			}
			packed, err3 := structPack(errCls, order, its, args[2:])
			if err3 != nil {
				return nil, err3
			}
			start := int(off)
			if start < 0 || start+len(packed) > len(ba.V) {
				return nil, se("pack_into requires a buffer of at least %d bytes", start+len(packed))
			}
			copy(ba.V[start:], packed)
			return object.None, nil
		}})

		self.Dict.SetStr("iter_unpack", &object.BuiltinFunc{Name: "iter_unpack", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if len(args) < 1 {
				return nil, se("iter_unpack() missing buffer")
			}
			data, err2 := asBytes(args[0])
			if err2 != nil {
				return nil, se("iter_unpack() buffer must be bytes-like")
			}
			if sz == 0 {
				return nil, se("iter_unpack() format size is 0")
			}
			if len(data)%sz != 0 {
				return nil, se("iterative unpacking requires a buffer whose length is a multiple of %d", sz)
			}
			buf := append([]byte(nil), data...)
			pos := 0
			order, its, err3 := parseFormat(fmtStr.V)
			if err3 != nil {
				return nil, err3
			}
			return &object.Iter{Next: func() (object.Object, bool, error) {
				if pos >= len(buf) {
					return nil, false, nil
				}
				vals, err4 := structUnpack(order, its, buf[pos:pos+sz])
				if err4 != nil {
					return nil, false, err4
				}
				pos += sz
				return &object.Tuple{V: vals}, true, nil
			}}, nil
		}})

		return object.None, nil
	}})
	m.Dict.SetStr("Struct", structCls)

	return m
}

// argAsInt coerces an int-compatible operand for struct packing.
func argAsInt(o object.Object) (int64, bool) {
	if b, ok := o.(*object.Bool); ok {
		if b.V {
			return 1, true
		}
		return 0, true
	}
	return toInt64(o)
}

// argAsUint64 extracts the low 64 bits from any int-like operand. Big ints
// above MaxInt64 still pack correctly for format chars like 'Q'.
func argAsUint64(o object.Object) (uint64, bool) {
	if b, ok := o.(*object.Bool); ok {
		if b.V {
			return 1, true
		}
		return 0, true
	}
	if i, ok := o.(*object.Int); ok {
		return new(big.Int).And(&i.V, new(big.Int).SetUint64(^uint64(0))).Uint64(), true
	}
	return 0, false
}

func structPack(errCls *object.Class, order binary.ByteOrder, items []structItem, args []object.Object) ([]byte, error) {
	se := func(msg string, a ...any) error {
		return object.Errorf(errCls, msg, a...)
	}
	need := 0
	for _, it := range items {
		if it.kind != 'x' {
			need++
		}
	}
	if len(args) != need {
		return nil, se("pack expected %d items for packing (got %d)", need, len(args))
	}
	out := make([]byte, 0, structTotalSize(items))
	ai := 0
	for _, it := range items {
		switch it.kind {
		case 'x':
			out = append(out, 0)
		case 'c':
			b, err := asBytes(args[ai])
			if err != nil || len(b) != 1 {
				return nil, se("char format requires a bytes object of length 1")
			}
			out = append(out, b[0])
			ai++
		case 'b':
			n, ok := argAsInt(args[ai])
			if !ok {
				return nil, se("required argument is not an integer")
			}
			if n < -128 || n > 127 {
				return nil, se("byte format requires -128 <= number <= 127")
			}
			out = append(out, byte(int8(n)))
			ai++
		case 'B':
			n, ok := argAsInt(args[ai])
			if !ok {
				return nil, se("required argument is not an integer")
			}
			if n < 0 || n > 255 {
				return nil, se("ubyte format requires 0 <= number <= 255")
			}
			out = append(out, byte(n))
			ai++
		case '?':
			n, ok := argAsInt(args[ai])
			if !ok {
				return nil, se("required argument is not an integer")
			}
			out = append(out, byte(n))
			ai++
		case 'h':
			n, ok := argAsInt(args[ai])
			if !ok {
				return nil, se("required argument is not an integer")
			}
			if n < -32768 || n > 32767 {
				return nil, se("short format requires -32768 <= number <= 32767")
			}
			var buf [2]byte
			order.PutUint16(buf[:], uint16(int16(n)))
			out = append(out, buf[:]...)
			ai++
		case 'H':
			n, ok := argAsInt(args[ai])
			if !ok {
				return nil, se("required argument is not an integer")
			}
			if n < 0 || n > 65535 {
				return nil, se("ushort format requires 0 <= number <= 65535")
			}
			var buf [2]byte
			order.PutUint16(buf[:], uint16(n))
			out = append(out, buf[:]...)
			ai++
		case 'i', 'l':
			n, ok := argAsInt(args[ai])
			if !ok {
				return nil, se("required argument is not an integer")
			}
			if n < -2147483648 || n > 2147483647 {
				return nil, se("int format requires -2147483648 <= number <= 2147483647")
			}
			var buf [4]byte
			order.PutUint32(buf[:], uint32(int32(n)))
			out = append(out, buf[:]...)
			ai++
		case 'I', 'L':
			u, ok := argAsUint64(args[ai])
			if !ok {
				return nil, se("required argument is not an integer")
			}
			var buf [4]byte
			order.PutUint32(buf[:], uint32(u))
			out = append(out, buf[:]...)
			ai++
		case 'q':
			n, ok := argAsInt(args[ai])
			if !ok {
				return nil, se("required argument is not an integer")
			}
			var buf [8]byte
			order.PutUint64(buf[:], uint64(n))
			out = append(out, buf[:]...)
			ai++
		case 'Q':
			u, ok := argAsUint64(args[ai])
			if !ok {
				return nil, se("required argument is not an integer")
			}
			var buf [8]byte
			order.PutUint64(buf[:], u)
			out = append(out, buf[:]...)
			ai++
		case 'e':
			f, ok := toFloat64(args[ai])
			if !ok {
				return nil, se("required argument is not a float")
			}
			var buf [2]byte
			order.PutUint16(buf[:], float16Encode(float32(f)))
			out = append(out, buf[:]...)
			ai++
		case 'f':
			f, ok := toFloat64(args[ai])
			if !ok {
				return nil, se("required argument is not a float")
			}
			var buf [4]byte
			order.PutUint32(buf[:], math.Float32bits(float32(f)))
			out = append(out, buf[:]...)
			ai++
		case 'd':
			f, ok := toFloat64(args[ai])
			if !ok {
				return nil, se("required argument is not a float")
			}
			var buf [8]byte
			order.PutUint64(buf[:], math.Float64bits(f))
			out = append(out, buf[:]...)
			ai++
		case 's':
			b, err := asBytes(args[ai])
			if err != nil {
				return nil, se("s format requires a bytes-like object")
			}
			if len(b) > it.n {
				b = b[:it.n]
			}
			out = append(out, b...)
			for k := len(b); k < it.n; k++ {
				out = append(out, 0)
			}
			ai++
		case 'p':
			b, err := asBytes(args[ai])
			if err != nil {
				return nil, se("p format requires a bytes-like object")
			}
			maxLen := it.n - 1
			if maxLen < 0 {
				maxLen = 0
			}
			if len(b) > maxLen {
				b = b[:maxLen]
			}
			out = append(out, byte(len(b)))
			out = append(out, b...)
			for k := len(b) + 1; k < it.n; k++ {
				out = append(out, 0)
			}
			ai++
		}
	}
	return out, nil
}

func structUnpack(order binary.ByteOrder, items []structItem, data []byte) ([]object.Object, error) {
	out := make([]object.Object, 0, len(items))
	p := 0
	for _, it := range items {
		switch it.kind {
		case 'x':
			p++
		case 'c':
			out = append(out, &object.Bytes{V: []byte{data[p]}})
			p++
		case 'b':
			out = append(out, object.NewInt(int64(int8(data[p]))))
			p++
		case 'B':
			out = append(out, object.NewInt(int64(data[p])))
			p++
		case '?':
			out = append(out, object.BoolOf(data[p] != 0))
			p++
		case 'h':
			out = append(out, object.NewInt(int64(int16(order.Uint16(data[p:])))))
			p += 2
		case 'H':
			out = append(out, object.NewInt(int64(order.Uint16(data[p:]))))
			p += 2
		case 'i', 'l':
			out = append(out, object.NewInt(int64(int32(order.Uint32(data[p:])))))
			p += 4
		case 'I', 'L':
			out = append(out, object.NewInt(int64(order.Uint32(data[p:]))))
			p += 4
		case 'q':
			out = append(out, object.NewInt(int64(order.Uint64(data[p:]))))
			p += 8
		case 'Q':
			u := order.Uint64(data[p:])
			out = append(out, newIntU64(u))
			p += 8
		case 'e':
			bits := order.Uint16(data[p:])
			out = append(out, &object.Float{V: float64(float16Decode(bits))})
			p += 2
		case 'f':
			bits := order.Uint32(data[p:])
			out = append(out, &object.Float{V: float64(math.Float32frombits(bits))})
			p += 4
		case 'd':
			bits := order.Uint64(data[p:])
			out = append(out, &object.Float{V: math.Float64frombits(bits)})
			p += 8
		case 's':
			out = append(out, &object.Bytes{V: append([]byte(nil), data[p:p+it.n]...)})
			p += it.n
		case 'p':
			length := int(data[p])
			if length > it.n-1 {
				length = it.n - 1
			}
			out = append(out, &object.Bytes{V: append([]byte(nil), data[p+1:p+1+length]...)})
			p += it.n
		}
	}
	return out, nil
}

// toFloat64 coerces a numeric to float64 for struct packing.
func toFloat64(o object.Object) (float64, bool) {
	switch v := o.(type) {
	case *object.Float:
		return v.V, true
	case *object.Int:
		f, _ := new(big.Float).SetInt(&v.V).Float64()
		return f, true
	case *object.Bool:
		if v.V {
			return 1.0, true
		}
		return 0.0, true
	}
	return 0, false
}

// --- csv module ------------------------------------------------------------

func excelDialect() *object.CSVDialect {
	return &object.CSVDialect{
		Delimiter:      ',',
		Quotechar:      '"',
		Doublequote:    true,
		Lineterminator: "\r\n",
		Quoting:        0,
	}
}

func excelTabDialect() *object.CSVDialect {
	d := excelDialect()
	d.Delimiter = '\t'
	return d
}

// resolveDialect picks a dialect from a positional/keyword arg. Kwargs can
// override individual fields.
func resolveDialect(base *object.CSVDialect, dialectArg object.Object, kw *object.Dict) *object.CSVDialect {
	d := *base
	if dialectArg != nil {
		if s, ok := dialectArg.(*object.Str); ok {
			if s.V == "excel-tab" {
				d = *excelTabDialect()
			}
		}
	}
	if kw != nil {
		if v, ok := kw.GetStr("delimiter"); ok {
			if s, ok := v.(*object.Str); ok && len(s.V) > 0 {
				d.Delimiter = s.V[0]
			}
		}
		if v, ok := kw.GetStr("quotechar"); ok {
			if s, ok := v.(*object.Str); ok && len(s.V) > 0 {
				d.Quotechar = s.V[0]
			}
		}
		if v, ok := kw.GetStr("lineterminator"); ok {
			if s, ok := v.(*object.Str); ok {
				d.Lineterminator = s.V
			}
		}
		if v, ok := kw.GetStr("quoting"); ok {
			if n, ok := toInt64(v); ok {
				d.Quoting = int(n)
			}
		}
		if v, ok := kw.GetStr("doublequote"); ok {
			if b, ok := v.(*object.Bool); ok {
				d.Doublequote = b.V
			}
		}
	}
	return &d
}

func (i *Interp) buildCsv() *object.Module {
	m := &object.Module{Name: "csv", Dict: object.NewDict()}

	m.Dict.SetStr("QUOTE_MINIMAL", object.NewInt(0))
	m.Dict.SetStr("QUOTE_ALL", object.NewInt(1))
	m.Dict.SetStr("QUOTE_NONNUMERIC", object.NewInt(2))
	m.Dict.SetStr("QUOTE_NONE", object.NewInt(3))

	m.Dict.SetStr("reader", &object.BuiltinFunc{Name: "reader", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "reader() missing source")
		}
		lines, err := i.iterStrings(a[0])
		if err != nil {
			return nil, err
		}
		var dialect object.Object
		if len(a) >= 2 {
			dialect = a[1]
		}
		d := resolveDialect(excelDialect(), dialect, kw)
		r := csv.NewReader(strings.NewReader(strings.Join(lines, "\n")))
		r.Comma = rune(d.Delimiter)
		r.LazyQuotes = true
		r.FieldsPerRecord = -1
		rows, err := r.ReadAll()
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		return &object.CSVReader{Rows: rows, Dialect: d}, nil
	}})

	m.Dict.SetStr("writer", &object.BuiltinFunc{Name: "writer", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "writer() missing target")
		}
		var dialect object.Object
		if len(a) >= 2 {
			dialect = a[1]
		}
		d := resolveDialect(excelDialect(), dialect, kw)
		return &object.CSVWriter{Target: a[0], Dialect: d}, nil
	}})

	m.Dict.SetStr("DictReader", &object.BuiltinFunc{Name: "DictReader", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "DictReader() missing source")
		}
		lines, err := i.iterStrings(a[0])
		if err != nil {
			return nil, err
		}
		d := resolveDialect(excelDialect(), nil, kw)
		r := csv.NewReader(strings.NewReader(strings.Join(lines, "\n")))
		r.Comma = rune(d.Delimiter)
		r.LazyQuotes = true
		r.FieldsPerRecord = -1
		rows, err := r.ReadAll()
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		var fieldnames []string
		if len(a) >= 2 {
			if l, ok := a[1].(*object.List); ok {
				for _, v := range l.V {
					if s, ok := v.(*object.Str); ok {
						fieldnames = append(fieldnames, s.V)
					}
				}
			}
		}
		if fieldnames == nil && len(rows) > 0 {
			fieldnames = rows[0]
			rows = rows[1:]
		}
		// Build list of dicts lazily via a Python iter.
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(rows) {
				return nil, false, nil
			}
			row := rows[idx]
			idx++
			dd := object.NewDict()
			for k, name := range fieldnames {
				if k < len(row) {
					dd.SetStr(name, &object.Str{V: row[k]})
				} else {
					dd.SetStr(name, object.None)
				}
			}
			return dd, true, nil
		}}, nil
	}})

	m.Dict.SetStr("DictWriter", &object.BuiltinFunc{Name: "DictWriter", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "DictWriter() missing target or fieldnames")
		}
		var fieldnames []string
		if l, ok := a[1].(*object.List); ok {
			for _, v := range l.V {
				if s, ok := v.(*object.Str); ok {
					fieldnames = append(fieldnames, s.V)
				}
			}
		}
		d := resolveDialect(excelDialect(), nil, kw)
		w := &object.CSVWriter{Target: a[0], Dialect: d}
		return &object.CSVDictWriter{Writer: w, Fieldnames: fieldnames}, nil
	}})

	return m
}

// iterStrings collects an iterable of str (e.g. a list of CSV lines or a
// StringIO) into a flat []string. StringIO is special-cased to read the
// whole buffer.
func (i *Interp) iterStrings(o object.Object) ([]string, error) {
	if sio, ok := o.(*object.StringIO); ok {
		return strings.Split(string(sio.V), "\n"), nil
	}
	it, err := i.getIter(o)
	if err != nil {
		return nil, err
	}
	var out []string
	for {
		v, ok, err := it.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		s, ok := v.(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "csv iterator yielded a non-str")
		}
		out = append(out, s.V)
	}
	return out, nil
}

func csvFormatRow(row []string, d *object.CSVDialect) (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Comma = rune(d.Delimiter)
	if err := w.Write(row); err != nil {
		return "", err
	}
	w.Flush()
	out := buf.String()
	// Go's encoding/csv uses \n; strip it so we can append our own terminator.
	out = strings.TrimRight(out, "\r\n")
	return out, nil
}

func csvWriteLine(i *Interp, target object.Object, line string) error {
	wr, err := i.getAttr(target, "write")
	if err != nil {
		return err
	}
	_, err = i.callObject(wr, []object.Object{&object.Str{V: line}}, nil)
	return err
}

func pyToCsvCell(v object.Object) string {
	switch x := v.(type) {
	case *object.Str:
		return x.V
	case *object.Int:
		return x.V.String()
	case *object.Float:
		return formatFloatSimple(x.V)
	case *object.Bool:
		if x.V {
			return "True"
		}
		return "False"
	case *object.NoneType:
		return ""
	}
	return object.Repr(v)
}

func formatFloatSimple(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatFloat(f, 'f', 1, 64)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// csvReaderAttr dispatches attribute access on a *object.CSVReader.
func csvReaderAttr(i *Interp, r *object.CSVReader, name string) (object.Object, bool) {
	switch name {
	case "line_num":
		return object.NewInt(int64(r.LineNo)), true
	case "dialect":
		dd := object.NewDict()
		dd.SetStr("delimiter", &object.Str{V: string(r.Dialect.Delimiter)})
		dd.SetStr("quotechar", &object.Str{V: string(r.Dialect.Quotechar)})
		dd.SetStr("lineterminator", &object.Str{V: r.Dialect.Lineterminator})
		return dd, true
	case "__iter__":
		return &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return r, nil
		}}, true
	case "__next__":
		return &object.BuiltinFunc{Name: "__next__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if r.Pos >= len(r.Rows) {
				exc := object.NewException(i.stopIter, "")
				return nil, exc
			}
			row := r.Rows[r.Pos]
			r.Pos++
			r.LineNo++
			v := make([]object.Object, len(row))
			for k, s := range row {
				v[k] = &object.Str{V: s}
			}
			return &object.List{V: v}, nil
		}}, true
	}
	return nil, false
}

// csvWriterAttr dispatches attribute access on a *object.CSVWriter.
func csvWriterAttr(i *Interp, w *object.CSVWriter, name string) (object.Object, bool) {
	switch name {
	case "dialect":
		dd := object.NewDict()
		dd.SetStr("delimiter", &object.Str{V: string(w.Dialect.Delimiter)})
		dd.SetStr("quotechar", &object.Str{V: string(w.Dialect.Quotechar)})
		dd.SetStr("lineterminator", &object.Str{V: w.Dialect.Lineterminator})
		return dd, true
	case "writerow":
		return &object.BuiltinFunc{Name: "writerow", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "writerow() missing row")
			}
			row, err := pyRowToStrings(i, a[0])
			if err != nil {
				return nil, err
			}
			line, err := csvFormatRow(row, w.Dialect)
			if err != nil {
				return nil, err
			}
			full := line + w.Dialect.Lineterminator
			if err := csvWriteLine(i, w.Target, full); err != nil {
				return nil, err
			}
			return object.NewInt(int64(len(full))), nil
		}}, true
	case "writerows":
		return &object.BuiltinFunc{Name: "writerows", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "writerows() missing rows")
			}
			it, err := i.getIter(a[0])
			if err != nil {
				return nil, err
			}
			for {
				v, ok, err := it.Next()
				if err != nil {
					return nil, err
				}
				if !ok {
					break
				}
				row, err := pyRowToStrings(i, v)
				if err != nil {
					return nil, err
				}
				line, err := csvFormatRow(row, w.Dialect)
				if err != nil {
					return nil, err
				}
				if err := csvWriteLine(i, w.Target, line+w.Dialect.Lineterminator); err != nil {
					return nil, err
				}
			}
			return object.None, nil
		}}, true
	}
	return nil, false
}

// csvDictWriterAttr dispatches attribute access on *object.CSVDictWriter.
func csvDictWriterAttr(i *Interp, dw *object.CSVDictWriter, name string) (object.Object, bool) {
	switch name {
	case "fieldnames":
		v := make([]object.Object, len(dw.Fieldnames))
		for k, n := range dw.Fieldnames {
			v[k] = &object.Str{V: n}
		}
		return &object.List{V: v}, true
	case "writeheader":
		return &object.BuiltinFunc{Name: "writeheader", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			line, err := csvFormatRow(dw.Fieldnames, dw.Writer.Dialect)
			if err != nil {
				return nil, err
			}
			full := line + dw.Writer.Dialect.Lineterminator
			if err := csvWriteLine(i, dw.Writer.Target, full); err != nil {
				return nil, err
			}
			return object.NewInt(int64(len(full))), nil
		}}, true
	case "writerow":
		return &object.BuiltinFunc{Name: "writerow", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "writerow() missing row")
			}
			d, ok := a[0].(*object.Dict)
			if !ok {
				return nil, object.Errorf(i.typeErr, "writerow() row must be dict")
			}
			row := dictRowToStrings(d, dw.Fieldnames)
			line, err := csvFormatRow(row, dw.Writer.Dialect)
			if err != nil {
				return nil, err
			}
			full := line + dw.Writer.Dialect.Lineterminator
			if err := csvWriteLine(i, dw.Writer.Target, full); err != nil {
				return nil, err
			}
			return object.NewInt(int64(len(full))), nil
		}}, true
	case "writerows":
		return &object.BuiltinFunc{Name: "writerows", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "writerows() missing rows")
			}
			it, err := i.getIter(a[0])
			if err != nil {
				return nil, err
			}
			for {
				v, ok, err := it.Next()
				if err != nil {
					return nil, err
				}
				if !ok {
					break
				}
				d, ok := v.(*object.Dict)
				if !ok {
					return nil, object.Errorf(i.typeErr, "writerows() row must be dict")
				}
				row := dictRowToStrings(d, dw.Fieldnames)
				line, err := csvFormatRow(row, dw.Writer.Dialect)
				if err != nil {
					return nil, err
				}
				if err := csvWriteLine(i, dw.Writer.Target, line+dw.Writer.Dialect.Lineterminator); err != nil {
					return nil, err
				}
			}
			return object.None, nil
		}}, true
	}
	return nil, false
}

func dictRowToStrings(d *object.Dict, fieldnames []string) []string {
	out := make([]string, len(fieldnames))
	for i, name := range fieldnames {
		if v, ok := d.GetStr(name); ok {
			out[i] = pyToCsvCell(v)
		}
	}
	return out
}

func pyRowToStrings(i *Interp, o object.Object) ([]string, error) {
	it, err := i.getIter(o)
	if err != nil {
		return nil, err
	}
	var out []string
	for {
		v, ok, err := it.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		out = append(out, pyToCsvCell(v))
	}
	return out, nil
}

// --- urllib.parse module ---------------------------------------------------

func (i *Interp) buildUrllib() *object.Module {
	m := &object.Module{Name: "urllib", Dict: object.NewDict()}
	m.Dict.SetStr("__path__", &object.List{V: []object.Object{&object.Str{V: "<builtin>"}}})
	return m
}

func (i *Interp) buildUrllibParse() *object.Module {
	m := &object.Module{Name: "urllib.parse", Dict: object.NewDict()}
	m.Dict.SetStr("__package__", &object.Str{V: "urllib"})

	m.Dict.SetStr("urlparse", &object.BuiltinFunc{Name: "urlparse", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "urlparse")
		if err != nil {
			return nil, err
		}
		return parseURL(s, true), nil
	}})
	m.Dict.SetStr("urlsplit", &object.BuiltinFunc{Name: "urlsplit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "urlsplit")
		if err != nil {
			return nil, err
		}
		return parseURL(s, false), nil
	}})
	m.Dict.SetStr("urlunparse", &object.BuiltinFunc{Name: "urlunparse", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "urlunparse() missing parts")
		}
		parts, err := extractURLParts(a[0], true)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: buildURL(parts, true)}, nil
	}})
	m.Dict.SetStr("urlunsplit", &object.BuiltinFunc{Name: "urlunsplit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "urlunsplit() missing parts")
		}
		parts, err := extractURLParts(a[0], false)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: buildURL(parts, false)}, nil
	}})
	m.Dict.SetStr("urljoin", &object.BuiltinFunc{Name: "urljoin", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "urljoin() requires base and url")
		}
		bs, _ := a[0].(*object.Str)
		rs, _ := a[1].(*object.Str)
		if bs == nil || rs == nil {
			return nil, object.Errorf(i.typeErr, "urljoin() arguments must be str")
		}
		base, err := url.Parse(bs.V)
		if err != nil {
			return &object.Str{V: rs.V}, nil
		}
		ref, err := url.Parse(rs.V)
		if err != nil {
			return &object.Str{V: rs.V}, nil
		}
		return &object.Str{V: base.ResolveReference(ref).String()}, nil
	}})
	m.Dict.SetStr("quote", &object.BuiltinFunc{Name: "quote", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "quote")
		if err != nil {
			return nil, err
		}
		safe := "/"
		if len(a) >= 2 {
			if ss, ok := a[1].(*object.Str); ok {
				safe = ss.V
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("safe"); ok {
				if ss, ok := v.(*object.Str); ok {
					safe = ss.V
				}
			}
		}
		return &object.Str{V: pctEncode(s, safe, false)}, nil
	}})
	m.Dict.SetStr("quote_plus", &object.BuiltinFunc{Name: "quote_plus", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "quote_plus")
		if err != nil {
			return nil, err
		}
		safe := ""
		if len(a) >= 2 {
			if ss, ok := a[1].(*object.Str); ok {
				safe = ss.V
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("safe"); ok {
				if ss, ok := v.(*object.Str); ok {
					safe = ss.V
				}
			}
		}
		return &object.Str{V: pctEncode(s, safe, true)}, nil
	}})
	m.Dict.SetStr("unquote", &object.BuiltinFunc{Name: "unquote", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "unquote")
		if err != nil {
			return nil, err
		}
		out, _ := url.QueryUnescape(strings.ReplaceAll(s, "+", "%2B"))
		return &object.Str{V: out}, nil
	}})
	m.Dict.SetStr("unquote_plus", &object.BuiltinFunc{Name: "unquote_plus", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "unquote_plus")
		if err != nil {
			return nil, err
		}
		out, _ := url.QueryUnescape(s)
		return &object.Str{V: out}, nil
	}})
	m.Dict.SetStr("urlencode", &object.BuiltinFunc{Name: "urlencode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "urlencode() missing mapping")
		}
		doseq := false
		if kw != nil {
			if v, ok := kw.GetStr("doseq"); ok {
				if b, ok := v.(*object.Bool); ok {
					doseq = b.V
				}
			}
		}
		pairs, err := extractQSPairs(i, a[0])
		if err != nil {
			return nil, err
		}
		var parts []string
		for _, p := range pairs {
			k := pctEncode(anyToStr(p[0]), "", true)
			v := p[1]
			if doseq {
				if lst, ok := v.(*object.List); ok {
					for _, vv := range lst.V {
						parts = append(parts, k+"="+pctEncode(anyToStr(vv), "", true))
					}
					continue
				}
				if tup, ok := v.(*object.Tuple); ok {
					for _, vv := range tup.V {
						parts = append(parts, k+"="+pctEncode(anyToStr(vv), "", true))
					}
					continue
				}
			}
			parts = append(parts, k+"="+pctEncode(anyToStr(v), "", true))
		}
		return &object.Str{V: strings.Join(parts, "&")}, nil
	}})
	m.Dict.SetStr("parse_qs", &object.BuiltinFunc{Name: "parse_qs", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "parse_qs")
		if err != nil {
			return nil, err
		}
		out := object.NewDict()
		for _, pair := range splitQS(s) {
			k, v, ok := splitKV(pair)
			if !ok || k == "" {
				continue
			}
			kdec, _ := url.QueryUnescape(k)
			vdec, _ := url.QueryUnescape(v)
			kObj := &object.Str{V: kdec}
			if existing, ok := out.GetStr(kdec); ok {
				if lst, ok := existing.(*object.List); ok {
					lst.V = append(lst.V, &object.Str{V: vdec})
				}
				continue
			}
			out.Set(kObj, &object.List{V: []object.Object{&object.Str{V: vdec}}})
		}
		return out, nil
	}})
	m.Dict.SetStr("parse_qsl", &object.BuiltinFunc{Name: "parse_qsl", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, err := stringArg(i, a, "parse_qsl")
		if err != nil {
			return nil, err
		}
		out := &object.List{}
		for _, pair := range splitQS(s) {
			k, v, ok := splitKV(pair)
			if !ok || k == "" {
				continue
			}
			kdec, _ := url.QueryUnescape(k)
			vdec, _ := url.QueryUnescape(v)
			out.V = append(out.V, &object.Tuple{V: []object.Object{&object.Str{V: kdec}, &object.Str{V: vdec}}})
		}
		return out, nil
	}})

	return m
}

func stringArg(i *Interp, a []object.Object, fn string) (string, error) {
	if len(a) == 0 {
		return "", object.Errorf(i.typeErr, "%s() missing argument", fn)
	}
	s, ok := a[0].(*object.Str)
	if !ok {
		return "", object.Errorf(i.typeErr, "%s() argument must be str", fn)
	}
	return s.V, nil
}

// parseURL splits a URL into scheme/netloc/path/params/query/fragment. When
// withParams is false (urlsplit), the path retains its ';params' suffix.
func parseURL(s string, withParams bool) *object.URLParseResult {
	r := &object.URLParseResult{}
	rest := s
	// scheme
	if idx := strings.Index(rest, ":"); idx > 0 {
		head := rest[:idx]
		if isValidScheme(head) {
			r.Scheme = strings.ToLower(head)
			rest = rest[idx+1:]
		}
	}
	// fragment
	if idx := strings.Index(rest, "#"); idx >= 0 {
		r.Fragment = rest[idx+1:]
		rest = rest[:idx]
	}
	// query
	if idx := strings.Index(rest, "?"); idx >= 0 {
		r.Query = rest[idx+1:]
		rest = rest[:idx]
	}
	// netloc
	if strings.HasPrefix(rest, "//") {
		rest = rest[2:]
		sep := strings.IndexAny(rest, "/")
		if sep < 0 {
			r.Netloc = rest
			rest = ""
		} else {
			r.Netloc = rest[:sep]
			rest = rest[sep:]
		}
	}
	// params
	if withParams {
		if idx := strings.LastIndex(rest, ";"); idx >= 0 {
			r.Params = rest[idx+1:]
			rest = rest[:idx]
		}
	}
	r.Path = rest
	return r
}

func isValidScheme(s string) bool {
	if len(s) == 0 {
		return false
	}
	first := s[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')) {
		return false
	}
	for _, c := range s[1:] {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '+' || c == '-' || c == '.':
		default:
			return false
		}
	}
	return true
}

// extractURLParts pulls the 6 (or 5) string components out of a tuple,
// list, or URLParseResult.
func extractURLParts(o object.Object, withParams bool) ([]string, error) {
	if r, ok := o.(*object.URLParseResult); ok {
		if withParams {
			return []string{r.Scheme, r.Netloc, r.Path, r.Params, r.Query, r.Fragment}, nil
		}
		return []string{r.Scheme, r.Netloc, r.Path, r.Query, r.Fragment}, nil
	}
	var seq []object.Object
	switch v := o.(type) {
	case *object.Tuple:
		seq = v.V
	case *object.List:
		seq = v.V
	default:
		return nil, fmt.Errorf("expected tuple or list")
	}
	out := make([]string, len(seq))
	for i, v := range seq {
		if s, ok := v.(*object.Str); ok {
			out[i] = s.V
		}
	}
	return out, nil
}

func buildURL(parts []string, withParams bool) string {
	var scheme, netloc, path, params, query, fragment string
	if withParams && len(parts) >= 6 {
		scheme, netloc, path, params, query, fragment = parts[0], parts[1], parts[2], parts[3], parts[4], parts[5]
	} else if !withParams && len(parts) >= 5 {
		scheme, netloc, path, query, fragment = parts[0], parts[1], parts[2], parts[3], parts[4]
	}
	var b strings.Builder
	if scheme != "" {
		b.WriteString(scheme)
		b.WriteByte(':')
	}
	if netloc != "" || (scheme != "" && strings.HasPrefix(path, "/")) {
		b.WriteString("//")
		b.WriteString(netloc)
	}
	b.WriteString(path)
	if params != "" {
		b.WriteByte(';')
		b.WriteString(params)
	}
	if query != "" {
		b.WriteByte('?')
		b.WriteString(query)
	}
	if fragment != "" {
		b.WriteByte('#')
		b.WriteString(fragment)
	}
	return b.String()
}

// pctEncode percent-encodes a string. When plus is true, spaces become '+'
// rather than %20. Characters in safe are passed through literally.
func pctEncode(s, safe string, plus bool) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEncode(c, safe, plus) {
			if plus && c == ' ' {
				b.WriteByte('+')
				continue
			}
			fmt.Fprintf(&b, "%%%02X", c)
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func shouldEncode(c byte, safe string, plus bool) bool {
	switch {
	case c >= 'a' && c <= 'z':
		return false
	case c >= 'A' && c <= 'Z':
		return false
	case c >= '0' && c <= '9':
		return false
	case c == '-' || c == '_' || c == '.' || c == '~':
		return false
	}
	if !plus && c == ' ' {
		// quote() does NOT treat space as safe by default
		return true
	}
	for i := 0; i < len(safe); i++ {
		if safe[i] == c {
			return false
		}
	}
	return true
}

func extractQSPairs(i *Interp, o object.Object) ([][2]object.Object, error) {
	var out [][2]object.Object
	if d, ok := o.(*object.Dict); ok {
		keys, vals := d.Items()
		// Preserve insertion order (Dict.Items does already).
		for k, key := range keys {
			ks, _ := key.(*object.Str)
			if ks == nil {
				continue
			}
			out = append(out, [2]object.Object{&object.Str{V: ks.V}, vals[k]})
		}
		return out, nil
	}
	// Fall through: iterate a list/tuple of 2-tuples.
	it, err := i.getIter(o)
	if err != nil {
		return nil, err
	}
	for {
		v, ok, err := it.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		tup, ok := v.(*object.Tuple)
		if !ok || len(tup.V) < 2 {
			lst, ok := v.(*object.List)
			if !ok || len(lst.V) < 2 {
				return nil, object.Errorf(i.valueErr, "not a sequence of 2-tuples")
			}
			out = append(out, [2]object.Object{lst.V[0], lst.V[1]})
			continue
		}
		out = append(out, [2]object.Object{tup.V[0], tup.V[1]})
	}
	return out, nil
}

func anyToStr(o object.Object) string {
	if s, ok := o.(*object.Str); ok {
		return s.V
	}
	if i, ok := o.(*object.Int); ok {
		return i.V.String()
	}
	if f, ok := o.(*object.Float); ok {
		return formatFloatSimple(f.V)
	}
	if b, ok := o.(*object.Bool); ok {
		if b.V {
			return "True"
		}
		return "False"
	}
	return object.Repr(o)
}

func splitQS(s string) []string {
	if s == "" {
		return nil
	}
	// Split on both '&' and ';' the way parse_qs does by default.
	f := func(c rune) bool { return c == '&' || c == ';' }
	return strings.FieldsFunc(s, f)
}

func splitKV(pair string) (string, string, bool) {
	idx := strings.Index(pair, "=")
	if idx < 0 {
		return pair, "", true
	}
	return pair[:idx], pair[idx+1:], true
}

// urlParseResultAttr dispatches attributes for URLParseResult.
func urlParseResultAttr(r *object.URLParseResult, name string) (object.Object, bool) {
	switch name {
	case "scheme":
		return &object.Str{V: r.Scheme}, true
	case "netloc":
		return &object.Str{V: r.Netloc}, true
	case "path":
		return &object.Str{V: r.Path}, true
	case "params":
		return &object.Str{V: r.Params}, true
	case "query":
		return &object.Str{V: r.Query}, true
	case "fragment":
		return &object.Str{V: r.Fragment}, true
	case "hostname":
		host := r.Netloc
		if at := strings.LastIndex(host, "@"); at >= 0 {
			host = host[at+1:]
		}
		if colon := strings.Index(host, ":"); colon >= 0 {
			host = host[:colon]
		}
		return &object.Str{V: strings.ToLower(host)}, true
	case "port":
		host := r.Netloc
		if at := strings.LastIndex(host, "@"); at >= 0 {
			host = host[at+1:]
		}
		if colon := strings.Index(host, ":"); colon >= 0 {
			if n, err := strconv.Atoi(host[colon+1:]); err == nil {
				return object.NewInt(int64(n)), true
			}
		}
		return object.None, true
	}
	return nil, false
}

// urlParseResultGetItem supports tuple-style integer indexing.
func urlParseResultGetItem(r *object.URLParseResult, idx int) (object.Object, bool) {
	fields := []string{r.Scheme, r.Netloc, r.Path, r.Params, r.Query, r.Fragment}
	if idx < 0 {
		idx += len(fields)
	}
	if idx < 0 || idx >= len(fields) {
		return nil, false
	}
	return &object.Str{V: fields[idx]}, true
}

// newIntU64 wraps a uint64 as *object.Int, preserving values above MaxInt64.
func newIntU64(u uint64) *object.Int {
	return object.IntFromBig(new(big.Int).SetUint64(u))
}

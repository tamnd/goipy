package vm

import (
	"encoding/binary"
	"errors"
	"math"
	"math/big"
	"strconv"

	"github.com/tamnd/goipy/object"
)

var errMarshalEOF = errors.New("EOF read where object expected")
var errMarshalTrunc = errors.New("truncated marshal data")

// Marshal type codes (CPython Python/marshal.c).
const (
	mNull        = '0' // dict terminator
	mNone        = 'N'
	mFalse       = 'F'
	mTrue        = 'T'
	mStopIter    = 'S'
	mEllipsis    = '.'
	mInt         = 'i' // 32-bit signed LE
	mLong        = 'l' // arbitrary precision
	mFloat       = 'f' // old: string float (v0-1)
	mBinaryFloat = 'g' // IEEE 754 double LE
	mComplex     = 'x' // old: string complex
	mBinComplex  = 'y' // two IEEE 754 doubles
	mString      = 's' // bytes (4-byte len)
	mInterned    = 't' // interned string (old)
	mRef         = 'r' // object table reference
	mUnicode     = 'u' // str, 4-byte len, UTF-8
	mTuple       = '(' // 4-byte count
	mSmallTuple  = ')' // 1-byte count
	mList        = '['
	mDict        = '{'
	mSet         = '<'
	FrozenSet    = '>'
	mAscii       = 'a' // str 4-byte len, ASCII
	mAsciiIntern = 'A' // str 4-byte len, ASCII interned
	mShortAscii  = 'z' // str 1-byte len, ASCII
	mShortAsciiI = 'Z' // str 1-byte len, ASCII interned
	mFlagRef     = 0x80
)

// ── writer ───────────────────────────────────────────────────────────────────

type marshalWriter struct {
	buf []byte
}

func (w *marshalWriter) byte1(b byte) { w.buf = append(w.buf, b) }

func (w *marshalWriter) uint32(v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	w.buf = append(w.buf, b[:]...)
}

func (w *marshalWriter) int32(v int32) { w.uint32(uint32(v)) }

func (w *marshalWriter) uint64(v uint64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	w.buf = append(w.buf, b[:]...)
}

func (w *marshalWriter) uint16(v uint16) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	w.buf = append(w.buf, b[:]...)
}

func (w *marshalWriter) writeObject(obj object.Object) error {
	switch v := obj.(type) {
	case *object.NoneType:
		w.byte1(mNone)
	case *object.EllipsisType:
		w.byte1(mEllipsis)
	case *object.Bool:
		if v.V {
			w.byte1(mTrue)
		} else {
			w.byte1(mFalse)
		}
	case *object.Int:
		return w.writeInt(v)
	case *object.Float:
		w.byte1(mBinaryFloat)
		w.uint64(math.Float64bits(v.V))
	case *object.Complex:
		w.byte1(mBinComplex)
		w.uint64(math.Float64bits(v.Real))
		w.uint64(math.Float64bits(v.Imag))
	case *object.Str:
		return w.writeStr(v.V)
	case *object.Bytes:
		w.byte1(mString)
		w.uint32(uint32(len(v.V)))
		w.buf = append(w.buf, v.V...)
	case *object.Bytearray:
		w.byte1(mString)
		w.uint32(uint32(len(v.V)))
		w.buf = append(w.buf, v.V...)
	case *object.Tuple:
		return w.writeTuple(v.V)
	case *object.List:
		w.byte1(mList)
		w.uint32(uint32(len(v.V)))
		for _, elem := range v.V {
			if err := w.writeObject(elem); err != nil {
				return err
			}
		}
	case *object.Dict:
		w.byte1(mDict)
		ks, vs := v.Items()
		for idx, k := range ks {
			if err := w.writeObject(k); err != nil {
				return err
			}
			if err := w.writeObject(vs[idx]); err != nil {
				return err
			}
		}
		w.byte1(mNull)
	case *object.Set:
		w.byte1(mSet)
		items := v.Items()
		w.uint32(uint32(len(items)))
		for _, elem := range items {
			if err := w.writeObject(elem); err != nil {
				return err
			}
		}
	case *object.Frozenset:
		w.byte1(FrozenSet)
		items := v.Items()
		w.uint32(uint32(len(items)))
		for _, elem := range items {
			if err := w.writeObject(elem); err != nil {
				return err
			}
		}
	default:
		return errors.New("unmarshallable object: " + object.TypeName(obj))
	}
	return nil
}

func (w *marshalWriter) writeInt(v *object.Int) error {
	// Use TYPE_INT for values that fit in int32, TYPE_LONG otherwise.
	if v.V.IsInt64() {
		n := v.V.Int64()
		if n >= math.MinInt32 && n <= math.MaxInt32 {
			w.byte1(mInt)
			w.int32(int32(n))
			return nil
		}
	}
	// TYPE_LONG: encode in 15-bit digits.
	w.byte1(mLong)
	abs := new(big.Int).Abs(&v.V)
	var digits []uint16
	mask := big.NewInt(0x7fff)
	tmp := new(big.Int).Set(abs)
	for tmp.Sign() > 0 {
		d := new(big.Int).And(tmp, mask)
		digits = append(digits, uint16(d.Int64()))
		tmp.Rsh(tmp, 15)
	}
	nDigits := int32(len(digits))
	if v.V.Sign() < 0 {
		nDigits = -nDigits
	}
	w.int32(nDigits)
	for _, d := range digits {
		w.uint16(d)
	}
	return nil
}

func (w *marshalWriter) writeStr(s string) error {
	utf8 := []byte(s)
	if len(utf8) <= 255 {
		w.byte1(mShortAsciiI)
		w.byte1(byte(len(utf8)))
		w.buf = append(w.buf, utf8...)
	} else {
		w.byte1(mAsciiIntern)
		w.uint32(uint32(len(utf8)))
		w.buf = append(w.buf, utf8...)
	}
	return nil
}

func (w *marshalWriter) writeTuple(elems []object.Object) error {
	if len(elems) <= 255 {
		w.byte1(mSmallTuple)
		w.byte1(byte(len(elems)))
	} else {
		w.byte1(mTuple)
		w.uint32(uint32(len(elems)))
	}
	for _, elem := range elems {
		if err := w.writeObject(elem); err != nil {
			return err
		}
	}
	return nil
}

// ── reader ───────────────────────────────────────────────────────────────────

type marshalReader struct {
	data []byte
	pos  int
	refs []object.Object // object reference table (FLAG_REF)
}

func (r *marshalReader) readByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, errMarshalEOF
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

func (r *marshalReader) readUint32() (uint32, error) {
	if r.pos+4 > len(r.data) {
		return 0, errMarshalTrunc
	}
	v := binary.LittleEndian.Uint32(r.data[r.pos:])
	r.pos += 4
	return v, nil
}

func (r *marshalReader) readInt32() (int32, error) {
	v, err := r.readUint32()
	return int32(v), err
}

func (r *marshalReader) readUint64() (uint64, error) {
	if r.pos+8 > len(r.data) {
		return 0, errMarshalTrunc
	}
	v := binary.LittleEndian.Uint64(r.data[r.pos:])
	r.pos += 8
	return v, nil
}

func (r *marshalReader) readUint16() (uint16, error) {
	if r.pos+2 > len(r.data) {
		return 0, errMarshalTrunc
	}
	v := binary.LittleEndian.Uint16(r.data[r.pos:])
	r.pos += 2
	return v, nil
}

func (r *marshalReader) readBytes(n int) ([]byte, error) {
	if r.pos+n > len(r.data) {
		return nil, errMarshalTrunc
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}

func (r *marshalReader) reserveRef() int {
	idx := len(r.refs)
	r.refs = append(r.refs, nil)
	return idx
}

func (r *marshalReader) setRef(idx int, obj object.Object) {
	r.refs[idx] = obj
}

func (r *marshalReader) readObject() (object.Object, error) {
	code, err := r.readByte()
	if err != nil {
		return nil, err
	}
	hasRef := (code & mFlagRef) != 0
	typeCode := code &^ mFlagRef

	var refIdx int
	if hasRef {
		refIdx = r.reserveRef()
	}

	var obj object.Object
	switch typeCode {
	case mNull:
		return nil, nil // sentinel — caller checks
	case mNone:
		obj = object.None
	case mEllipsis:
		obj = object.Ellipsis
	case mTrue:
		obj = object.True
	case mFalse:
		obj = object.False
	case mStopIter:
		obj = object.None // substitute None for StopIteration
	case mInt:
		n, e2 := r.readInt32()
		if e2 != nil {
			return nil, e2
		}
		obj = object.IntFromInt64(int64(n))
	case mLong:
		o, e2 := r.readLong()
		if e2 != nil {
			return nil, e2
		}
		obj = o
	case mBinaryFloat:
		bits, e2 := r.readUint64()
		if e2 != nil {
			return nil, e2
		}
		obj = &object.Float{V: math.Float64frombits(bits)}
	case mFloat: // old string-encoded float
		length, e2 := r.readByte()
		if e2 != nil {
			return nil, e2
		}
		raw, e3 := r.readBytes(int(length))
		if e3 != nil {
			return nil, e3
		}
		var f float64
		_, err2 := marshalSscanf(string(raw), &f)
		if err2 != nil {
			return nil, object.Errorf(nil, "ValueError: bad float: %s", string(raw))
		}
		obj = &object.Float{V: f}
	case mBinComplex:
		rbits, e2 := r.readUint64()
		if e2 != nil {
			return nil, e2
		}
		ibits, e3 := r.readUint64()
		if e3 != nil {
			return nil, e3
		}
		obj = &object.Complex{Real: math.Float64frombits(rbits), Imag: math.Float64frombits(ibits)}
	case mComplex: // old string-encoded complex
		rl, e2 := r.readByte()
		if e2 != nil {
			return nil, e2
		}
		rraw, e3 := r.readBytes(int(rl))
		if e3 != nil {
			return nil, e3
		}
		il, e4 := r.readByte()
		if e4 != nil {
			return nil, e4
		}
		iraw, e5 := r.readBytes(int(il))
		if e5 != nil {
			return nil, e5
		}
		var re, im float64
		marshalSscanf(string(rraw), &re)
		marshalSscanf(string(iraw), &im)
		obj = &object.Complex{Real: re, Imag: im}
	case mString:
		n, e2 := r.readUint32()
		if e2 != nil {
			return nil, e2
		}
		b, e3 := r.readBytes(int(n))
		if e3 != nil {
			return nil, e3
		}
		cp := make([]byte, len(b))
		copy(cp, b)
		obj = &object.Bytes{V: cp}
	case mInterned, mUnicode, mAscii, mAsciiIntern: // 4-byte length strings
		n, e2 := r.readUint32()
		if e2 != nil {
			return nil, e2
		}
		b, e3 := r.readBytes(int(n))
		if e3 != nil {
			return nil, e3
		}
		obj = &object.Str{V: string(b)}
	case mShortAscii, mShortAsciiI: // 1-byte length strings
		length, e2 := r.readByte()
		if e2 != nil {
			return nil, e2
		}
		b, e3 := r.readBytes(int(length))
		if e3 != nil {
			return nil, e3
		}
		obj = &object.Str{V: string(b)}
	case mSmallTuple:
		count, e2 := r.readByte()
		if e2 != nil {
			return nil, e2
		}
		elems, e3 := r.readNObjects(int(count))
		if e3 != nil {
			return nil, e3
		}
		obj = &object.Tuple{V: elems}
	case mTuple:
		count, e2 := r.readUint32()
		if e2 != nil {
			return nil, e2
		}
		elems, e3 := r.readNObjects(int(count))
		if e3 != nil {
			return nil, e3
		}
		obj = &object.Tuple{V: elems}
	case mList:
		count, e2 := r.readUint32()
		if e2 != nil {
			return nil, e2
		}
		elems, e3 := r.readNObjects(int(count))
		if e3 != nil {
			return nil, e3
		}
		obj = &object.List{V: elems}
	case mDict:
		d := object.NewDict()
		for {
			k, e2 := r.readObject()
			if e2 != nil {
				return nil, e2
			}
			if k == nil { // mNull sentinel
				break
			}
			v2, e3 := r.readObject()
			if e3 != nil {
				return nil, e3
			}
			if err2 := d.Set(k, v2); err2 != nil {
				return nil, err2
			}
		}
		obj = d
	case mSet:
		count, e2 := r.readUint32()
		if e2 != nil {
			return nil, e2
		}
		s := object.NewSet()
		for n := uint32(0); n < count; n++ {
			elem, e3 := r.readObject()
			if e3 != nil {
				return nil, e3
			}
			if err2 := s.Add(elem); err2 != nil {
				return nil, err2
			}
		}
		obj = s
	case FrozenSet:
		count, e2 := r.readUint32()
		if e2 != nil {
			return nil, e2
		}
		fs := object.NewFrozenset()
		for n := uint32(0); n < count; n++ {
			elem, e3 := r.readObject()
			if e3 != nil {
				return nil, e3
			}
			if err2 := fs.Add(elem); err2 != nil {
				return nil, err2
			}
		}
		obj = fs
	case mRef:
		idx, e2 := r.readUint32()
		if e2 != nil {
			return nil, e2
		}
		if int(idx) >= len(r.refs) {
			return nil, errors.New("bad marshal ref index")
		}
		return r.refs[idx], nil
	default:
		return nil, errors.New("bad marshal type code")
	}

	if hasRef {
		r.setRef(refIdx, obj)
	}
	return obj, nil
}

func (r *marshalReader) readNObjects(n int) ([]object.Object, error) {
	elems := make([]object.Object, n)
	for i := 0; i < n; i++ {
		o, err := r.readObject()
		if err != nil {
			return nil, err
		}
		elems[i] = o
	}
	return elems, nil
}

func (r *marshalReader) readLong() (object.Object, error) {
	n, err := r.readInt32()
	if err != nil {
		return nil, err
	}
	sign := int32(1)
	numDigits := int(n)
	if numDigits < 0 {
		sign = -1
		numDigits = -numDigits
	}
	result := new(big.Int)
	for i := 0; i < numDigits; i++ {
		d, e2 := r.readUint16()
		if e2 != nil {
			return nil, e2
		}
		tmp := new(big.Int).SetUint64(uint64(d))
		tmp.Lsh(tmp, uint(15*i))
		result.Or(result, tmp)
	}
	if sign < 0 {
		result.Neg(result)
	}
	return object.IntFromBig(result), nil
}

// marshalSscanf parses a float from a string (handles "inf", "-inf", "nan").
func marshalSscanf(s string, f *float64) (int, error) {
	switch s {
	case "inf", "+inf":
		*f = math.Inf(1)
	case "-inf":
		*f = math.Inf(-1)
	case "nan", "-nan", "+nan":
		*f = math.NaN()
	default:
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, err
		}
		*f = v
	}
	return 1, nil
}

// ── module builder ────────────────────────────────────────────────────────────

func (i *Interp) buildMarshal() *object.Module {
	m := &object.Module{Name: "marshal", Dict: object.NewDict()}

	m.Dict.SetStr("version", object.IntFromInt64(4))

	marshalDo := func(obj object.Object, _ int) ([]byte, error) {
		w := &marshalWriter{}
		if err := w.writeObject(obj); err != nil {
			return nil, object.Errorf(i.valueErr, "%v", err)
		}
		return w.buf, nil
	}

	marshalRead := func(data []byte) (object.Object, error) {
		if len(data) == 0 {
			return nil, object.Errorf(i.eofErr, "EOF read where object expected")
		}
		r := &marshalReader{data: data}
		obj, err := r.readObject()
		if err != nil {
			if errors.Is(err, errMarshalEOF) || errors.Is(err, errMarshalTrunc) {
				return nil, object.Errorf(i.eofErr, "%s", err.Error())
			}
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		return obj, nil
	}

	// marshal.dumps(value, version=4)
	m.Dict.SetStr("dumps", &object.BuiltinFunc{Name: "dumps", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "dumps() requires value")
		}
		ver := 4
		if len(args) >= 2 {
			if vi, ok := args[1].(*object.Int); ok {
				ver = int(vi.Int64())
			}
		}
		data, err := marshalDo(args[0], ver)
		if err != nil {
			return nil, err
		}
		return &object.Bytes{V: data}, nil
	}})

	// marshal.loads(bytes_or_bytearray)
	m.Dict.SetStr("loads", &object.BuiltinFunc{Name: "loads", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "loads() requires bytes-like object")
		}
		var data []byte
		switch v := args[0].(type) {
		case *object.Bytes:
			data = v.V
		case *object.Bytearray:
			data = v.V
		default:
			return nil, object.Errorf(i.typeErr, "loads() argument must be bytes-like")
		}
		return marshalRead(data)
	}})

	// marshal.dump(value, file, version=4)
	m.Dict.SetStr("dump", &object.BuiltinFunc{Name: "dump", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "dump() requires value and file")
		}
		ver := 4
		if len(args) >= 3 {
			if vi, ok := args[2].(*object.Int); ok {
				ver = int(vi.Int64())
			}
		}
		data, err := marshalDo(args[0], ver)
		if err != nil {
			return nil, err
		}
		writeFn, err2 := i.getAttr(args[1], "write")
		if err2 != nil {
			return nil, err2
		}
		_, err3 := i.callObject(writeFn, []object.Object{&object.Bytes{V: data}}, nil)
		return object.None, err3
	}})

	// marshal.load(file)
	m.Dict.SetStr("load", &object.BuiltinFunc{Name: "load", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "load() requires file")
		}
		readFn, err := i.getAttr(args[0], "read")
		if err != nil {
			return nil, err
		}
		// Read up to 4 MB at once; marshal objects are small in practice.
		chunk, err2 := i.callObject(readFn, []object.Object{object.IntFromInt64(4 * 1024 * 1024)}, nil)
		if err2 != nil {
			return nil, err2
		}
		var data []byte
		switch v := chunk.(type) {
		case *object.Bytes:
			data = v.V
		case *object.Bytearray:
			data = v.V
		default:
			return nil, object.Errorf(i.typeErr, "file.read() must return bytes")
		}
		return marshalRead(data)
	}})

	return m
}

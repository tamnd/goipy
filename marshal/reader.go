// Package marshal decodes the CPython marshal stream produced by .pyc files.
// Only the subset of type tags actually emitted by python3.14 -m py_compile is
// implemented; legacy tags (TYPE_FLOAT ascii, TYPE_COMPLEX) are supported for
// completeness.
package marshal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/tamnd/goipy/object"
)

// Type tags.
const (
	TYPE_NULL             = '0'
	TYPE_NONE             = 'N'
	TYPE_FALSE            = 'F'
	TYPE_TRUE             = 'T'
	TYPE_STOPITER         = 'S'
	TYPE_ELLIPSIS         = '.'
	TYPE_INT              = 'i'
	TYPE_INT64            = 'I'
	TYPE_FLOAT            = 'f'
	TYPE_BINARY_FLOAT     = 'g'
	TYPE_COMPLEX          = 'x'
	TYPE_BINARY_COMPLEX   = 'y'
	TYPE_LONG             = 'l'
	TYPE_STRING           = 's'
	TYPE_INTERNED         = 't'
	TYPE_REF              = 'r'
	TYPE_TUPLE            = '('
	TYPE_LIST             = '['
	TYPE_DICT             = '{'
	TYPE_CODE             = 'c'
	TYPE_UNICODE          = 'u'
	TYPE_UNKNOWN          = '?'
	TYPE_SET              = '<'
	TYPE_FROZENSET        = '>'
	TYPE_ASCII            = 'a'
	TYPE_ASCII_INTERNED   = 'A'
	TYPE_SMALL_TUPLE      = ')'
	TYPE_SHORT_ASCII      = 'z'
	TYPE_SHORT_ASCII_INTERNED = 'Z'
	TYPE_SLICE            = ':'

	FLAG_REF = 0x80
)

// Reader decodes a marshal stream.
type Reader struct {
	buf  []byte
	pos  int
	refs []object.Object
}

// Unmarshal decodes a single object from b.
func Unmarshal(b []byte) (object.Object, error) {
	r := &Reader{buf: b}
	return r.ReadObject()
}

func (r *Reader) need(n int) error {
	if r.pos+n > len(r.buf) {
		return fmt.Errorf("marshal: unexpected EOF (need %d, have %d)", n, len(r.buf)-r.pos)
	}
	return nil
}

func (r *Reader) readByte() (byte, error) {
	if err := r.need(1); err != nil {
		return 0, err
	}
	b := r.buf[r.pos]
	r.pos++
	return b, nil
}

func (r *Reader) readU32() (uint32, error) {
	if err := r.need(4); err != nil {
		return 0, err
	}
	v := binary.LittleEndian.Uint32(r.buf[r.pos:])
	r.pos += 4
	return v, nil
}

func (r *Reader) readI32() (int32, error) {
	v, err := r.readU32()
	return int32(v), err
}

func (r *Reader) readU64() (uint64, error) {
	if err := r.need(8); err != nil {
		return 0, err
	}
	v := binary.LittleEndian.Uint64(r.buf[r.pos:])
	r.pos += 8
	return v, nil
}

func (r *Reader) readBytes(n int) ([]byte, error) {
	if err := r.need(n); err != nil {
		return nil, err
	}
	b := r.buf[r.pos : r.pos+n]
	r.pos += n
	out := make([]byte, n)
	copy(out, b)
	return out, nil
}

func (r *Reader) reserveRef(flag bool) int {
	if !flag {
		return -1
	}
	r.refs = append(r.refs, nil)
	return len(r.refs) - 1
}

func (r *Reader) setRef(idx int, o object.Object) object.Object {
	if idx >= 0 {
		r.refs[idx] = o
	}
	return o
}

// ReadObject decodes the next object.
func (r *Reader) ReadObject() (object.Object, error) {
	t, err := r.readByte()
	if err != nil {
		return nil, err
	}
	flag := t&FLAG_REF != 0
	t &^= FLAG_REF

	switch t {
	case TYPE_NULL:
		return nil, nil
	case TYPE_NONE:
		return object.None, nil
	case TYPE_FALSE:
		return object.False, nil
	case TYPE_TRUE:
		return object.True, nil
	case TYPE_STOPITER:
		return nil, errors.New("marshal: TYPE_STOPITER not supported")
	case TYPE_ELLIPSIS:
		return object.None, nil // we don't model Ellipsis separately
	case TYPE_INT:
		v, err := r.readI32()
		if err != nil {
			return nil, err
		}
		o := &object.Int{V: big.NewInt(int64(v))}
		idx := r.reserveRef(flag)
		return r.setRef(idx, o), nil
	case TYPE_INT64:
		v, err := r.readU64()
		if err != nil {
			return nil, err
		}
		o := &object.Int{V: new(big.Int).SetInt64(int64(v))}
		idx := r.reserveRef(flag)
		return r.setRef(idx, o), nil
	case TYPE_LONG:
		return r.readLong(flag)
	case TYPE_BINARY_FLOAT:
		v, err := r.readU64()
		if err != nil {
			return nil, err
		}
		o := &object.Float{V: math.Float64frombits(v)}
		idx := r.reserveRef(flag)
		return r.setRef(idx, o), nil
	case TYPE_FLOAT:
		n, err := r.readByte()
		if err != nil {
			return nil, err
		}
		b, err := r.readBytes(int(n))
		if err != nil {
			return nil, err
		}
		var f float64
		_, _ = fmt.Sscan(string(b), &f)
		o := &object.Float{V: f}
		idx := r.reserveRef(flag)
		return r.setRef(idx, o), nil
	case TYPE_BINARY_COMPLEX:
		re, _ := r.readU64()
		im, err := r.readU64()
		if err != nil {
			return nil, err
		}
		o := &object.Complex{
			Real: math.Float64frombits(re),
			Imag: math.Float64frombits(im),
		}
		idx := r.reserveRef(flag)
		return r.setRef(idx, o), nil
	case TYPE_STRING:
		n, err := r.readU32()
		if err != nil {
			return nil, err
		}
		b, err := r.readBytes(int(n))
		if err != nil {
			return nil, err
		}
		o := &object.Bytes{V: b}
		idx := r.reserveRef(flag)
		return r.setRef(idx, o), nil
	case TYPE_UNICODE, TYPE_INTERNED, TYPE_ASCII, TYPE_ASCII_INTERNED:
		n, err := r.readU32()
		if err != nil {
			return nil, err
		}
		b, err := r.readBytes(int(n))
		if err != nil {
			return nil, err
		}
		o := &object.Str{V: string(b)}
		idx := r.reserveRef(flag)
		return r.setRef(idx, o), nil
	case TYPE_SHORT_ASCII, TYPE_SHORT_ASCII_INTERNED:
		n, err := r.readByte()
		if err != nil {
			return nil, err
		}
		b, err := r.readBytes(int(n))
		if err != nil {
			return nil, err
		}
		o := &object.Str{V: string(b)}
		idx := r.reserveRef(flag)
		return r.setRef(idx, o), nil
	case TYPE_TUPLE:
		n, err := r.readU32()
		if err != nil {
			return nil, err
		}
		return r.readTuple(int(n), flag)
	case TYPE_SMALL_TUPLE:
		n, err := r.readByte()
		if err != nil {
			return nil, err
		}
		return r.readTuple(int(n), flag)
	case TYPE_LIST:
		n, err := r.readU32()
		if err != nil {
			return nil, err
		}
		tup, err := r.readTuple(int(n), flag)
		if err != nil {
			return nil, err
		}
		return &object.List{V: tup.(*object.Tuple).V}, nil
	case TYPE_DICT:
		o := object.NewDict()
		idx := r.reserveRef(flag)
		r.setRef(idx, o)
		for {
			k, err := r.ReadObject()
			if err != nil {
				return nil, err
			}
			if k == nil {
				break
			}
			v, err := r.ReadObject()
			if err != nil {
				return nil, err
			}
			if err := o.Set(k, v); err != nil {
				return nil, err
			}
		}
		return o, nil
	case TYPE_SET, TYPE_FROZENSET:
		n, err := r.readU32()
		if err != nil {
			return nil, err
		}
		o := object.NewSet()
		idx := r.reserveRef(flag)
		r.setRef(idx, o)
		for i := uint32(0); i < n; i++ {
			x, err := r.ReadObject()
			if err != nil {
				return nil, err
			}
			if err := o.Add(x); err != nil {
				return nil, err
			}
		}
		return o, nil
	case TYPE_SLICE:
		start, err := r.ReadObject()
		if err != nil {
			return nil, err
		}
		stop, err := r.ReadObject()
		if err != nil {
			return nil, err
		}
		step, err := r.ReadObject()
		if err != nil {
			return nil, err
		}
		o := &object.Slice{Start: start, Stop: stop, Step: step}
		idx := r.reserveRef(flag)
		return r.setRef(idx, o), nil
	case TYPE_CODE:
		return r.readCode(flag)
	case TYPE_REF:
		ix, err := r.readU32()
		if err != nil {
			return nil, err
		}
		if int(ix) >= len(r.refs) {
			return nil, fmt.Errorf("marshal: bad TYPE_REF index %d", ix)
		}
		return r.refs[ix], nil
	}
	return nil, fmt.Errorf("marshal: unknown type byte %q (0x%02x) at pos %d", t, t, r.pos-1)
}

func (r *Reader) readTuple(n int, flag bool) (object.Object, error) {
	t := &object.Tuple{V: make([]object.Object, n)}
	idx := r.reserveRef(flag)
	r.setRef(idx, t)
	for i := 0; i < n; i++ {
		o, err := r.ReadObject()
		if err != nil {
			return nil, err
		}
		t.V[i] = o
	}
	return t, nil
}

func (r *Reader) readLong(flag bool) (object.Object, error) {
	n, err := r.readI32()
	if err != nil {
		return nil, err
	}
	neg := false
	size := int(n)
	if n < 0 {
		neg = true
		size = int(-n)
	}
	// shorts LE base 2^15
	result := new(big.Int)
	shift := uint(0)
	for i := 0; i < size; i++ {
		if err := r.need(2); err != nil {
			return nil, err
		}
		d := uint16(r.buf[r.pos]) | uint16(r.buf[r.pos+1])<<8
		r.pos += 2
		chunk := new(big.Int).SetUint64(uint64(d))
		chunk.Lsh(chunk, shift)
		result.Or(result, chunk)
		shift += 15
	}
	if neg {
		result.Neg(result)
	}
	o := &object.Int{V: result}
	idx := r.reserveRef(flag)
	return r.setRef(idx, o), nil
}

func (r *Reader) readCode(flag bool) (object.Object, error) {
	c := &object.Code{}
	idx := r.reserveRef(flag)
	r.setRef(idx, c)

	var err error
	var v int32
	if v, err = r.readI32(); err != nil {
		return nil, err
	}
	c.ArgCount = int(v)
	if v, err = r.readI32(); err != nil {
		return nil, err
	}
	c.PosOnlyArgCount = int(v)
	if v, err = r.readI32(); err != nil {
		return nil, err
	}
	c.KwOnlyArgCount = int(v)
	if v, err = r.readI32(); err != nil {
		return nil, err
	}
	c.Stacksize = int(v)
	if v, err = r.readI32(); err != nil {
		return nil, err
	}
	c.Flags = int(v)

	code, err := r.ReadObject()
	if err != nil {
		return nil, err
	}
	if b, ok := code.(*object.Bytes); ok {
		c.Bytecode = b.V
	} else {
		return nil, fmt.Errorf("marshal: co_code is %T", code)
	}

	consts, err := r.ReadObject()
	if err != nil {
		return nil, err
	}
	if t, ok := consts.(*object.Tuple); ok {
		c.Consts = t.V
	}

	names, err := r.ReadObject()
	if err != nil {
		return nil, err
	}
	if t, ok := names.(*object.Tuple); ok {
		c.Names = make([]string, len(t.V))
		for i, x := range t.V {
			c.Names[i] = x.(*object.Str).V
		}
	}

	localsplusnames, err := r.ReadObject()
	if err != nil {
		return nil, err
	}
	if t, ok := localsplusnames.(*object.Tuple); ok {
		c.LocalsPlusNames = make([]string, len(t.V))
		for i, x := range t.V {
			c.LocalsPlusNames[i] = x.(*object.Str).V
		}
	}

	kinds, err := r.ReadObject()
	if err != nil {
		return nil, err
	}
	if b, ok := kinds.(*object.Bytes); ok {
		c.LocalsPlusKinds = b.V
	}

	filename, err := r.ReadObject()
	if err != nil {
		return nil, err
	}
	if s, ok := filename.(*object.Str); ok {
		c.Filename = s.V
	}

	name, err := r.ReadObject()
	if err != nil {
		return nil, err
	}
	if s, ok := name.(*object.Str); ok {
		c.Name = s.V
	}

	qname, err := r.ReadObject()
	if err != nil {
		return nil, err
	}
	if s, ok := qname.(*object.Str); ok {
		c.QualName = s.V
	}

	if v, err = r.readI32(); err != nil {
		return nil, err
	}
	c.FirstLineNo = int(v)

	lt, err := r.ReadObject()
	if err != nil {
		return nil, err
	}
	if b, ok := lt.(*object.Bytes); ok {
		c.LineTable = b.V
	}

	et, err := r.ReadObject()
	if err != nil {
		return nil, err
	}
	if b, ok := et.(*object.Bytes); ok {
		c.ExceptionTable = b.V
	}

	// Derive cell/free lists from localspluskinds.
	for i, k := range c.LocalsPlusKinds {
		switch {
		case k&object.FastFree != 0:
			c.NFrees++
			c.FreeVars = append(c.FreeVars, c.LocalsPlusNames[i])
		case k&object.FastCell != 0:
			c.NCells++
			c.CellVars = append(c.CellVars, c.LocalsPlusNames[i])
		default:
			c.NLocals++
		}
	}

	return c, nil
}

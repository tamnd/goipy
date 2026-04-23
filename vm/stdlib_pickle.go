package vm

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/tamnd/goipy/object"
)

// ---- opcode constants -------------------------------------------------------

const (
	opMark           = '('
	opStop           = '.'
	opNone           = 'N'
	opBinInt1        = 'K'
	opBinInt2        = 'M'
	opBinInt         = 'J'
	opLong1          = 0x8a
	opLong4          = 0x8b
	opBFloat         = 'G'
	opBinUnicode     = 'X'
	opShortBinUni    = 0x8c
	opBinUni8        = 0x8d
	opBinBytes       = 'B'
	opShortBinBytes  = 'C'
	opBinBytes8      = 0x8e
	opBinString      = 'T' // proto 1 bytes-as-string
	opShortBinString = 'U'
	opEmptyList      = ']'
	opEmptyTuple     = ')'
	opEmptyDict      = '}'
	opEmptySet       = 0x8f
	opFrozenSet      = 0x91
	opAddItems       = 0x90
	opAppend         = 'a'
	opAppends        = 'e'
	opSetItem        = 's'
	opSetItems       = 'u'
	opTuple1         = 0x85
	opTuple2         = 0x86
	opTuple3         = 0x87
	opTuple          = 't'
	opList           = 'l'
	opDict           = 'd'
	opGlobal         = 'c'
	opStackGlobal    = 0x93
	opReduce         = 'R'
	opBinPut         = 'q'
	opLongBinPut     = 'r'
	opBinGet         = 'h'
	opLongBinGet     = 'j'
	opMemoize        = 0x94
	opFrame          = 0x95
	opProto          = 0x80
	opTrue           = 0x88
	opFalse          = 0x89
	opNewObj         = 0x81
	opNewObjEx       = 0x92
	opInt            = 'I'  // text int
	opLong           = 'L'  // text long
	opFloat          = 'F'  // text float
	opString         = 'S'  // text string
	opUnicode        = 'V'  // text unicode
	opGet            = 'g'  // text get
	opPut            = 'p'  // text put
	opBuild          = 'b'
	opInst           = 'i'
	opObj            = 'o'
)

// ---- module builder ---------------------------------------------------------

func (i *Interp) buildPickle() *object.Module {
	m := &object.Module{Name: "pickle", Dict: object.NewDict()}

	pickleErr := &object.Class{Name: "PickleError", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	picklingErr := &object.Class{Name: "PicklingError", Bases: []*object.Class{pickleErr}, Dict: object.NewDict()}
	unpicklingErr := &object.Class{Name: "UnpicklingError", Bases: []*object.Class{pickleErr}, Dict: object.NewDict()}
	m.Dict.SetStr("PickleError", pickleErr)
	m.Dict.SetStr("PicklingError", picklingErr)
	m.Dict.SetStr("UnpicklingError", unpicklingErr)

	m.Dict.SetStr("DEFAULT_PROTOCOL", object.NewInt(2))
	m.Dict.SetStr("HIGHEST_PROTOCOL", object.NewInt(5))
	m.Dict.SetStr("format_version", &object.Str{V: "5.0"})
	compatible := []string{"1.0", "1.1", "1.2", "1.3", "2.0", "3.0", "4.0", "5.0"}
	compatList := make([]object.Object, len(compatible))
	for idx, s := range compatible {
		compatList[idx] = &object.Str{V: s}
	}
	m.Dict.SetStr("compatible_formats", &object.List{V: compatList})

	// dumps(obj, protocol=None, *, fix_imports=True, buffer_callback=None)
	m.Dict.SetStr("dumps", &object.BuiltinFunc{Name: "dumps", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "dumps() requires obj")
		}
		proto := 2
		if len(a) >= 2 {
			if a[1] != object.None {
				if pn, ok := a[1].(*object.Int); ok {
					proto = int(pn.Int64())
				}
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("protocol"); ok && v != object.None {
				if pn, ok2 := v.(*object.Int); ok2 {
					proto = int(pn.Int64())
				}
			}
		}
		if proto < 0 || proto > 5 {
			return nil, object.Errorf(i.valueErr, "pickle protocol %d not supported", proto)
		}
		data, err := pickleSerialize(a[0], proto)
		if err != nil {
			return nil, object.NewException(picklingErr, err.Error())
		}
		return &object.Bytes{V: data}, nil
	}})

	// loads(data, /, *, fix_imports=True, encoding='ASCII', errors='strict', buffers=())
	m.Dict.SetStr("loads", &object.BuiltinFunc{Name: "loads", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "loads() requires data")
		}
		var data []byte
		switch v := a[0].(type) {
		case *object.Bytes:
			data = v.V
		case *object.Bytearray:
			data = v.V
		default:
			return nil, object.Errorf(i.typeErr, "loads() data must be bytes or bytearray")
		}
		obj, err := pickleDeserialize(data)
		if err != nil {
			return nil, object.NewException(unpicklingErr, err.Error())
		}
		return obj, nil
	}})

	// dump(obj, file, protocol=None, ...)
	m.Dict.SetStr("dump", &object.BuiltinFunc{Name: "dump", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "dump() requires obj and file")
		}
		proto := 2
		if len(a) >= 3 && a[2] != object.None {
			if pn, ok := a[2].(*object.Int); ok {
				proto = int(pn.Int64())
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("protocol"); ok && v != object.None {
				if pn, ok2 := v.(*object.Int); ok2 {
					proto = int(pn.Int64())
				}
			}
		}
		data, err := pickleSerialize(a[0], proto)
		if err != nil {
			return nil, object.NewException(picklingErr, err.Error())
		}
		writeFn, err := i.getAttr(a[1], "write")
		if err != nil {
			return nil, err
		}
		_, err = i.callObject(writeFn, []object.Object{&object.Bytes{V: data}}, nil)
		return object.None, err
	}})

	// load(file, *, fix_imports=True, encoding='ASCII', errors='strict', buffers=())
	m.Dict.SetStr("load", &object.BuiltinFunc{Name: "load", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "load() requires file")
		}
		readFn, err := i.getAttr(a[0], "read")
		if err != nil {
			return nil, err
		}
		// Read all remaining bytes by reading a large chunk.
		// To support sequential loads from the same file, we use a peek/read approach.
		chunk, err := i.callObject(readFn, []object.Object{object.NewInt(1 << 20)}, nil)
		if err != nil {
			return nil, err
		}
		var data []byte
		switch v := chunk.(type) {
		case *object.Bytes:
			data = v.V
		default:
			return nil, object.Errorf(i.typeErr, "file.read() must return bytes")
		}
		// Parse one pickle object, then seek back remaining bytes.
		obj, consumed, err2 := pickleDeserializeN(data)
		if err2 != nil {
			return nil, object.NewException(unpicklingErr, err2.Error())
		}
		// Seek back the unconsumed bytes.
		if len(data)-consumed > 0 {
			seekFn, serr := i.getAttr(a[0], "seek")
			if serr == nil {
				tellFn, terr := i.getAttr(a[0], "tell")
				if terr == nil {
					pos, _ := i.callObject(tellFn, nil, nil)
					if posInt, ok := pos.(*object.Int); ok {
						newPos := posInt.Int64() - int64(len(data)) + int64(consumed)
						_, _ = i.callObject(seekFn, []object.Object{object.NewInt(newPos)}, nil)
					}
				}
			}
		}
		return obj, nil
	}})

	return m
}

// ---- serializer -------------------------------------------------------------

func pickleSerialize(obj object.Object, proto int) ([]byte, error) {
	var payload bytes.Buffer
	p := &pickler{buf: &payload, proto: proto}
	if err := p.dump(obj); err != nil {
		return nil, err
	}
	payload.WriteByte(opStop)

	var out bytes.Buffer
	out.WriteByte(opProto)
	// Use at most protocol 4 encoding internally (proto 5 = same as 4 for our purposes)
	effectiveProto := proto
	if effectiveProto > 4 {
		effectiveProto = 4
	}
	out.WriteByte(byte(effectiveProto))

	if effectiveProto >= 4 {
		// Wrap payload in a FRAME.
		frameData := payload.Bytes()
		out.WriteByte(opFrame)
		var size [8]byte
		binary.LittleEndian.PutUint64(size[:], uint64(len(frameData)))
		out.Write(size[:])
	}
	out.Write(payload.Bytes())
	return out.Bytes(), nil
}

type pickler struct {
	buf   *bytes.Buffer
	proto int
	memo  map[uint64]int
}

func (p *pickler) dump(obj object.Object) error {
	switch v := obj.(type) {
	case *object.NoneType:
		p.buf.WriteByte(opNone)

	case *object.Bool:
		if v.V {
			p.buf.WriteByte(opTrue)
		} else {
			p.buf.WriteByte(opFalse)
		}

	case *object.Int:
		p.writeInt(v)

	case *object.Float:
		p.buf.WriteByte(opBFloat)
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], math.Float64bits(v.V))
		p.buf.Write(b[:])

	case *object.Str:
		p.writeStr(v.V)

	case *object.Bytes:
		p.writeBytes(v.V)

	case *object.Bytearray:
		p.writeBytes(v.V)

	case *object.Tuple:
		if err := p.writeTuple(v.V); err != nil {
			return err
		}

	case *object.List:
		if err := p.writeList(v.V); err != nil {
			return err
		}

	case *object.Dict:
		if err := p.writeDict(v); err != nil {
			return err
		}

	case *object.Set:
		if err := p.writeSet(v.Items()); err != nil {
			return err
		}

	case *object.Frozenset:
		if err := p.writeFrozenset(v.Items()); err != nil {
			return err
		}

	default:
		return &pickleTypeError{obj}
	}
	return nil
}

type pickleTypeError struct{ obj object.Object }

func (e *pickleTypeError) Error() string {
	return "cannot pickle " + object.TypeName(e.obj)
}

func (p *pickler) writeInt(n *object.Int) {
	if n.IsInt64() {
		v := n.Int64()
		switch {
		case v >= 0 && v <= 0xff:
			p.buf.WriteByte(opBinInt1)
			p.buf.WriteByte(byte(v))
		case v >= 0 && v <= 0xffff:
			p.buf.WriteByte(opBinInt2)
			p.buf.WriteByte(byte(v))
			p.buf.WriteByte(byte(v >> 8))
		case v >= -(1<<31) && v <= (1<<31)-1:
			p.buf.WriteByte(opBinInt)
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], uint32(int32(v)))
			p.buf.Write(b[:])
		default:
			p.writeLong(n.Big())
		}
	} else {
		p.writeLong(n.Big())
	}
}

// writeLong serializes a big.Int using LONG1 opcode.
func (p *pickler) writeLong(n *big.Int) {
	// Convert to little-endian two's complement bytes.
	b := pickleEncodeLong(n)
	if len(b) <= 0xff {
		p.buf.WriteByte(opLong1)
		p.buf.WriteByte(byte(len(b)))
	} else {
		p.buf.WriteByte(opLong4)
		var sz [4]byte
		binary.LittleEndian.PutUint32(sz[:], uint32(len(b)))
		p.buf.Write(sz[:])
	}
	p.buf.Write(b)
}

// pickleEncodeLong converts *big.Int to little-endian two's complement bytes.
func pickleEncodeLong(n *big.Int) []byte {
	if n.Sign() == 0 {
		return []byte{0}
	}
	neg := n.Sign() < 0
	var abs big.Int
	if neg {
		abs.Neg(n)
		abs.Sub(&abs, big.NewInt(1))
	} else {
		abs.Set(n)
	}
	// Get bytes in big-endian, then reverse to LE.
	b := abs.Bytes()
	// Reverse to little-endian.
	for lo, hi := 0, len(b)-1; lo < hi; lo, hi = lo+1, hi-1 {
		b[lo], b[hi] = b[hi], b[lo]
	}
	if neg {
		// Invert all bits (two's complement from ~abs-1 + 1, but we did abs-1 already).
		for i := range b {
			b[i] ^= 0xff
		}
	}
	// Ensure high bit doesn't cause sign ambiguity.
	if neg {
		if b[len(b)-1]&0x80 == 0 {
			b = append(b, 0xff)
		}
	} else {
		if b[len(b)-1]&0x80 != 0 {
			b = append(b, 0x00)
		}
	}
	return b
}

func (p *pickler) writeStr(s string) {
	utf8 := []byte(s)
	if p.proto >= 4 && len(utf8) <= 0xff {
		p.buf.WriteByte(opShortBinUni)
		p.buf.WriteByte(byte(len(utf8)))
	} else {
		p.buf.WriteByte(opBinUnicode)
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], uint32(len(utf8)))
		p.buf.Write(b[:])
	}
	p.buf.Write(utf8)
}

func (p *pickler) writeBytes(data []byte) {
	if p.proto >= 3 {
		if len(data) <= 0xff {
			p.buf.WriteByte(opShortBinBytes)
			p.buf.WriteByte(byte(len(data)))
		} else {
			p.buf.WriteByte(opBinBytes)
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], uint32(len(data)))
			p.buf.Write(b[:])
		}
		p.buf.Write(data)
	} else {
		// Protocol < 3: encode bytes using _codecs.encode trick with latin-1.
		// We write a GLOBAL for _codecs.encode, then args, then REDUCE.
		p.buf.WriteByte(opGlobal)
		p.buf.WriteString("_codecs\nencode\n")
		// Push args: (latin-1-decoded string, 'latin-1')
		latin1 := make([]byte, 0, len(data)*2)
		for _, b := range data {
			if b < 0x80 {
				latin1 = append(latin1, b)
			} else {
				// Encode as UTF-8 representation of the latin-1 code point.
				r := rune(b)
				encoded := []byte(string(r))
				latin1 = append(latin1, encoded...)
			}
		}
		p.buf.WriteByte(opBinUnicode)
		var sz [4]byte
		binary.LittleEndian.PutUint32(sz[:], uint32(len(latin1)))
		p.buf.Write(sz[:])
		p.buf.Write(latin1)
		p.writeStr("latin-1")
		p.buf.WriteByte(opTuple2)
		p.buf.WriteByte(opReduce)
	}
}

func (p *pickler) writeTuple(items []object.Object) error {
	n := len(items)
	if n == 0 {
		p.buf.WriteByte(opEmptyTuple)
		return nil
	}
	if n <= 3 {
		for _, item := range items {
			if err := p.dump(item); err != nil {
				return err
			}
		}
		switch n {
		case 1:
			p.buf.WriteByte(opTuple1)
		case 2:
			p.buf.WriteByte(opTuple2)
		case 3:
			p.buf.WriteByte(opTuple3)
		}
		return nil
	}
	p.buf.WriteByte(opMark)
	for _, item := range items {
		if err := p.dump(item); err != nil {
			return err
		}
	}
	p.buf.WriteByte(opTuple)
	return nil
}

func (p *pickler) writeList(items []object.Object) error {
	p.buf.WriteByte(opEmptyList)
	if len(items) == 0 {
		return nil
	}
	p.buf.WriteByte(opMark)
	for _, item := range items {
		if err := p.dump(item); err != nil {
			return err
		}
	}
	p.buf.WriteByte(opAppends)
	return nil
}

func (p *pickler) writeDict(d *object.Dict) error {
	p.buf.WriteByte(opEmptyDict)
	keys, vals := d.Items()
	if len(keys) == 0 {
		return nil
	}
	p.buf.WriteByte(opMark)
	for idx := range keys {
		if err := p.dump(keys[idx]); err != nil {
			return err
		}
		if err := p.dump(vals[idx]); err != nil {
			return err
		}
	}
	p.buf.WriteByte(opSetItems)
	return nil
}

func (p *pickler) writeSet(items []object.Object) error {
	if p.proto >= 4 {
		p.buf.WriteByte(opEmptySet)
		if len(items) == 0 {
			return nil
		}
		p.buf.WriteByte(opMark)
		for _, item := range items {
			if err := p.dump(item); err != nil {
				return err
			}
		}
		p.buf.WriteByte(opAddItems)
		return nil
	}
	// Protocol < 4: use GLOBAL __builtin__ set + REDUCE.
	p.buf.WriteByte(opGlobal)
	p.buf.WriteString("__builtin__\nset\n")
	if err := p.writeList(items); err != nil {
		return err
	}
	p.buf.WriteByte(opTuple1)
	p.buf.WriteByte(opReduce)
	return nil
}

func (p *pickler) writeFrozenset(items []object.Object) error {
	if p.proto >= 4 {
		p.buf.WriteByte(opMark)
		for _, item := range items {
			if err := p.dump(item); err != nil {
				return err
			}
		}
		p.buf.WriteByte(opFrozenSet)
		return nil
	}
	// Protocol < 4: use GLOBAL __builtin__ frozenset + REDUCE.
	p.buf.WriteByte(opGlobal)
	p.buf.WriteString("__builtin__\nfrozenset\n")
	if err := p.writeList(items); err != nil {
		return err
	}
	p.buf.WriteByte(opTuple1)
	p.buf.WriteByte(opReduce)
	return nil
}

// ---- deserializer -----------------------------------------------------------

func pickleDeserialize(data []byte) (object.Object, error) {
	obj, _, err := pickleDeserializeN(data)
	return obj, err
}

// pickleDeserializeN returns the deserialized object and number of bytes consumed.
func pickleDeserializeN(data []byte) (object.Object, int, error) {
	u := &unpickler{
		data: data,
		pos:  0,
		memo: map[int]object.Object{},
	}
	obj, err := u.load()
	return obj, u.pos, err
}

type markType struct{}

var markSentinel = &markType{}

type unpickler struct {
	data  []byte
	pos   int
	stack []object.Object // stack can hold markSentinel
	memo  map[int]object.Object
}

func (u *unpickler) readByte() (byte, error) {
	if u.pos >= len(u.data) {
		return 0, &pickleEOF{}
	}
	b := u.data[u.pos]
	u.pos++
	return b, nil
}

func (u *unpickler) readN(n int) ([]byte, error) {
	if u.pos+n > len(u.data) {
		return nil, &pickleEOF{}
	}
	b := u.data[u.pos : u.pos+n]
	u.pos += n
	return b, nil
}

func (u *unpickler) readLine() (string, error) {
	start := u.pos
	for u.pos < len(u.data) && u.data[u.pos] != '\n' {
		u.pos++
	}
	s := string(u.data[start:u.pos])
	if u.pos < len(u.data) {
		u.pos++ // skip \n
	}
	return s, nil
}

func (u *unpickler) push(o object.Object) { u.stack = append(u.stack, o) }

func (u *unpickler) pop() (object.Object, error) {
	if len(u.stack) == 0 {
		return nil, &pickleErr{"pickle stack underflow"}
	}
	v := u.stack[len(u.stack)-1]
	u.stack = u.stack[:len(u.stack)-1]
	return v, nil
}

func (u *unpickler) peek() (object.Object, error) {
	if len(u.stack) == 0 {
		return nil, &pickleErr{"pickle stack underflow"}
	}
	return u.stack[len(u.stack)-1], nil
}

// popMark pops items above the most recent MARK sentinel.
func (u *unpickler) popMark() ([]object.Object, error) {
	markIdx := -1
	for idx := len(u.stack) - 1; idx >= 0; idx-- {
		if u.stack[idx] == markSentinel {
			markIdx = idx
			break
		}
	}
	if markIdx < 0 {
		return nil, &pickleErr{"MARK not found on stack"}
	}
	items := make([]object.Object, len(u.stack)-markIdx-1)
	copy(items, u.stack[markIdx+1:])
	u.stack = u.stack[:markIdx]
	return items, nil
}

func (u *unpickler) load() (object.Object, error) {
	for {
		op, err := u.readByte()
		if err != nil {
			return nil, err
		}
		switch op {
		case opProto:
			if _, err := u.readByte(); err != nil { // protocol version
				return nil, err
			}

		case opFrame:
			if _, err := u.readN(8); err != nil { // frame size (ignore)
				return nil, err
			}

		case opStop:
			return u.pop()

		case opNone:
			u.push(object.None)

		case opTrue:
			u.push(object.True)

		case opFalse:
			u.push(object.False)

		case opBinInt1:
			b, err := u.readByte()
			if err != nil {
				return nil, err
			}
			u.push(object.NewInt(int64(b)))

		case opBinInt2:
			b, err := u.readN(2)
			if err != nil {
				return nil, err
			}
			u.push(object.NewInt(int64(binary.LittleEndian.Uint16(b))))

		case opBinInt:
			b, err := u.readN(4)
			if err != nil {
				return nil, err
			}
			u.push(object.NewInt(int64(int32(binary.LittleEndian.Uint32(b)))))

		case opLong1:
			n, err := u.readByte()
			if err != nil {
				return nil, err
			}
			b, err := u.readN(int(n))
			if err != nil {
				return nil, err
			}
			u.push(object.IntFromBig(pickleDecodeLong(b)))

		case opLong4:
			sb, err := u.readN(4)
			if err != nil {
				return nil, err
			}
			n := int(binary.LittleEndian.Uint32(sb))
			b, err := u.readN(n)
			if err != nil {
				return nil, err
			}
			u.push(object.IntFromBig(pickleDecodeLong(b)))

		case opBFloat:
			b, err := u.readN(8)
			if err != nil {
				return nil, err
			}
			f := math.Float64frombits(binary.BigEndian.Uint64(b))
			u.push(&object.Float{V: f})

		case opBinUnicode:
			b, err := u.readN(4)
			if err != nil {
				return nil, err
			}
			n := int(binary.LittleEndian.Uint32(b))
			s, err := u.readN(n)
			if err != nil {
				return nil, err
			}
			u.push(&object.Str{V: string(s)})

		case opShortBinUni:
			n, err := u.readByte()
			if err != nil {
				return nil, err
			}
			s, err := u.readN(int(n))
			if err != nil {
				return nil, err
			}
			u.push(&object.Str{V: string(s)})

		case opBinUni8:
			b, err := u.readN(8)
			if err != nil {
				return nil, err
			}
			n := int(binary.LittleEndian.Uint64(b))
			s, err := u.readN(n)
			if err != nil {
				return nil, err
			}
			u.push(&object.Str{V: string(s)})

		case opShortBinBytes:
			n, err := u.readByte()
			if err != nil {
				return nil, err
			}
			b, err := u.readN(int(n))
			if err != nil {
				return nil, err
			}
			cp := make([]byte, len(b))
			copy(cp, b)
			u.push(&object.Bytes{V: cp})

		case opBinBytes:
			sb, err := u.readN(4)
			if err != nil {
				return nil, err
			}
			n := int(binary.LittleEndian.Uint32(sb))
			b, err := u.readN(n)
			if err != nil {
				return nil, err
			}
			cp := make([]byte, len(b))
			copy(cp, b)
			u.push(&object.Bytes{V: cp})

		case opBinBytes8:
			sb, err := u.readN(8)
			if err != nil {
				return nil, err
			}
			n := int(binary.LittleEndian.Uint64(sb))
			b, err := u.readN(n)
			if err != nil {
				return nil, err
			}
			cp := make([]byte, len(b))
			copy(cp, b)
			u.push(&object.Bytes{V: cp})

		case opBinString: // proto 1 bytes-as-string (4-byte len)
			sb, err := u.readN(4)
			if err != nil {
				return nil, err
			}
			n := int(binary.LittleEndian.Uint32(sb))
			b, err := u.readN(n)
			if err != nil {
				return nil, err
			}
			cp := make([]byte, len(b))
			copy(cp, b)
			u.push(&object.Bytes{V: cp})

		case opShortBinString: // proto 1 bytes-as-string (1-byte len)
			n, err := u.readByte()
			if err != nil {
				return nil, err
			}
			b, err := u.readN(int(n))
			if err != nil {
				return nil, err
			}
			cp := make([]byte, len(b))
			copy(cp, b)
			u.push(&object.Bytes{V: cp})

		case opEmptyList:
			u.push(&object.List{V: nil})

		case opEmptyTuple:
			u.push(&object.Tuple{V: nil})

		case opEmptyDict:
			u.push(object.NewDict())

		case opEmptySet:
			u.push(object.NewSet())

		case opMark:
			u.stack = append(u.stack, markSentinel)

		case opTuple1:
			v, err := u.pop()
			if err != nil {
				return nil, err
			}
			u.push(&object.Tuple{V: []object.Object{v}})

		case opTuple2:
			b, err := u.pop()
			if err != nil {
				return nil, err
			}
			a, err := u.pop()
			if err != nil {
				return nil, err
			}
			u.push(&object.Tuple{V: []object.Object{a, b}})

		case opTuple3:
			c, err := u.pop()
			if err != nil {
				return nil, err
			}
			b, err := u.pop()
			if err != nil {
				return nil, err
			}
			a, err := u.pop()
			if err != nil {
				return nil, err
			}
			u.push(&object.Tuple{V: []object.Object{a, b, c}})

		case opTuple:
			items, err := u.popMark()
			if err != nil {
				return nil, err
			}
			u.push(&object.Tuple{V: items})

		case opList:
			items, err := u.popMark()
			if err != nil {
				return nil, err
			}
			u.push(&object.List{V: items})

		case opDict:
			items, err := u.popMark()
			if err != nil {
				return nil, err
			}
			d := object.NewDict()
			for j := 0; j+1 < len(items); j += 2 {
				_ = d.Set(items[j], items[j+1])
			}
			u.push(d)

		case opAppend:
			val, err := u.pop()
			if err != nil {
				return nil, err
			}
			lst, err := u.peek()
			if err != nil {
				return nil, err
			}
			if l, ok := lst.(*object.List); ok {
				l.V = append(l.V, val)
			}

		case opAppends:
			items, err := u.popMark()
			if err != nil {
				return nil, err
			}
			lst, err := u.peek()
			if err != nil {
				return nil, err
			}
			if l, ok := lst.(*object.List); ok {
				l.V = append(l.V, items...)
			}

		case opSetItem:
			val, err := u.pop()
			if err != nil {
				return nil, err
			}
			key, err := u.pop()
			if err != nil {
				return nil, err
			}
			top, err := u.peek()
			if err != nil {
				return nil, err
			}
			if d, ok := top.(*object.Dict); ok {
				_ = d.Set(key, val)
			}

		case opSetItems:
			items, err := u.popMark()
			if err != nil {
				return nil, err
			}
			top, err := u.peek()
			if err != nil {
				return nil, err
			}
			if d, ok := top.(*object.Dict); ok {
				for j := 0; j+1 < len(items); j += 2 {
					_ = d.Set(items[j], items[j+1])
				}
			}

		case opAddItems:
			items, err := u.popMark()
			if err != nil {
				return nil, err
			}
			top, err := u.peek()
			if err != nil {
				return nil, err
			}
			if s, ok := top.(*object.Set); ok {
				for _, item := range items {
					_ = s.Add(item)
				}
			}

		case opFrozenSet:
			items, err := u.popMark()
			if err != nil {
				return nil, err
			}
			fs := object.NewFrozenset()
			for _, item := range items {
				_ = fs.Add(item)
			}
			u.push(fs)

		case opGlobal:
			modName, err := u.readLine()
			if err != nil {
				return nil, err
			}
			name, err := u.readLine()
			if err != nil {
				return nil, err
			}
			g, err := u.resolveGlobal(modName, name)
			if err != nil {
				return nil, err
			}
			u.push(g)

		case opStackGlobal:
			name, err := u.pop()
			if err != nil {
				return nil, err
			}
			mod, err := u.pop()
			if err != nil {
				return nil, err
			}
			modStr, _ := mod.(*object.Str)
			nameStr, _ := name.(*object.Str)
			if modStr == nil || nameStr == nil {
				return nil, &pickleErr{"STACK_GLOBAL: expected str"}
			}
			g, err := u.resolveGlobal(modStr.V, nameStr.V)
			if err != nil {
				return nil, err
			}
			u.push(g)

		case opReduce:
			args, err := u.pop()
			if err != nil {
				return nil, err
			}
			callable, err := u.pop()
			if err != nil {
				return nil, err
			}
			result, err := u.callReduceCallable(callable, args)
			if err != nil {
				return nil, err
			}
			u.push(result)

		case opBinPut:
			n, err := u.readByte()
			if err != nil {
				return nil, err
			}
			top, err := u.peek()
			if err != nil {
				return nil, err
			}
			u.memo[int(n)] = top

		case opLongBinPut:
			b, err := u.readN(4)
			if err != nil {
				return nil, err
			}
			n := int(binary.LittleEndian.Uint32(b))
			top, err := u.peek()
			if err != nil {
				return nil, err
			}
			u.memo[n] = top

		case opBinGet:
			n, err := u.readByte()
			if err != nil {
				return nil, err
			}
			obj, ok := u.memo[int(n)]
			if !ok {
				return nil, &pickleErr{"memo key not found"}
			}
			u.push(obj)

		case opLongBinGet:
			b, err := u.readN(4)
			if err != nil {
				return nil, err
			}
			n := int(binary.LittleEndian.Uint32(b))
			obj, ok := u.memo[n]
			if !ok {
				return nil, &pickleErr{"memo key not found"}
			}
			u.push(obj)

		case opMemoize:
			top, err := u.peek()
			if err != nil {
				return nil, err
			}
			u.memo[len(u.memo)] = top

		case opInt: // text format
			line, err := u.readLine()
			if err != nil {
				return nil, err
			}
			line = strings.TrimSpace(line)
			if line == "01" {
				u.push(object.True)
			} else if line == "00" {
				u.push(object.False)
			} else {
				var n big.Int
				if _, ok := n.SetString(line, 10); !ok {
					return nil, &pickleErr{"bad INT: " + line}
				}
				u.push(object.IntFromBig(&n))
			}

		case opLong: // text format
			line, err := u.readLine()
			if err != nil {
				return nil, err
			}
			line = strings.TrimSuffix(strings.TrimSpace(line), "L")
			var n big.Int
			if _, ok := n.SetString(line, 10); !ok {
				return nil, &pickleErr{"bad LONG: " + line}
			}
			u.push(object.IntFromBig(&n))

		case opFloat: // text format
			line, err := u.readLine()
			if err != nil {
				return nil, err
			}
			f, err := parseFloat64(strings.TrimSpace(line))
			if err != nil {
				return nil, &pickleErr{"bad FLOAT: " + line}
			}
			u.push(&object.Float{V: f})

		case opUnicode: // text format
			line, err := u.readLine()
			if err != nil {
				return nil, err
			}
			s, err := decodeUnicodeEscapes(line)
			if err != nil {
				return nil, err
			}
			u.push(&object.Str{V: s})

		case opString: // text format (protocol 0)
			line, err := u.readLine()
			if err != nil {
				return nil, err
			}
			s := unquotePickleString(line)
			u.push(&object.Bytes{V: []byte(s)})

		case opGet: // text format
			line, err := u.readLine()
			if err != nil {
				return nil, err
			}
			idx := 0
			for _, c := range line {
				idx = idx*10 + int(c-'0')
			}
			obj, ok := u.memo[idx]
			if !ok {
				return nil, &pickleErr{"memo not found"}
			}
			u.push(obj)

		case opPut: // text format
			line, err := u.readLine()
			if err != nil {
				return nil, err
			}
			idx := 0
			for _, c := range line {
				idx = idx*10 + int(c-'0')
			}
			top, err := u.peek()
			if err != nil {
				return nil, err
			}
			u.memo[idx] = top

		case opBuild:
			// BUILD: pop state and apply to the object below it (for __setstate__).
			// For our purposes, skip complex object reconstruction.
			_, err := u.pop() // pop state
			if err != nil {
				return nil, err
			}

		case opNewObj:
			args, err := u.pop()
			if err != nil {
				return nil, err
			}
			cls, err := u.pop()
			if err != nil {
				return nil, err
			}
			result, err := u.callReduceCallable(cls, args)
			if err != nil {
				return nil, err
			}
			u.push(result)

		default:
			return nil, &pickleErr{"unsupported opcode: " + string([]byte{op})}
		}
	}
}

// resolveGlobal maps (module, name) → a callable sentinel for REDUCE.
func (u *unpickler) resolveGlobal(mod, name string) (object.Object, error) {
	key := mod + "\x00" + name
	switch key {
	case "__builtin__\x00set", "builtins\x00set", "_builtins_\x00set", "__builtins__\x00set":
		return &pickleGlobal{kind: "set"}, nil
	case "__builtin__\x00frozenset", "builtins\x00frozenset", "__builtins__\x00frozenset":
		return &pickleGlobal{kind: "frozenset"}, nil
	case "__builtin__\x00bytes", "builtins\x00bytes":
		return &pickleGlobal{kind: "bytes"}, nil
	case "_codecs\x00encode":
		return &pickleGlobal{kind: "_codecs_encode"}, nil
	default:
		return nil, &pickleErr{"cannot unpickle global " + mod + "." + name}
	}
}

// pickleGlobal is a sentinel for well-known builtins used in REDUCE.
type pickleGlobal struct{ kind string }

func (u *unpickler) callReduceCallable(callable, args object.Object) (object.Object, error) {
	if g, ok := callable.(*pickleGlobal); ok {
		tup, _ := args.(*object.Tuple)
		switch g.kind {
		case "set":
			s := object.NewSet()
			if tup != nil && len(tup.V) == 1 {
				if lst, ok := tup.V[0].(*object.List); ok {
					for _, item := range lst.V {
						_ = s.Add(item)
					}
				}
			}
			return s, nil
		case "frozenset":
			fs := object.NewFrozenset()
			if tup != nil && len(tup.V) == 1 {
				if lst, ok := tup.V[0].(*object.List); ok {
					for _, item := range lst.V {
						_ = fs.Add(item)
					}
				}
			}
			return fs, nil
		case "bytes":
			if tup != nil && len(tup.V) == 1 {
				switch v := tup.V[0].(type) {
				case *object.Bytes:
					return v, nil
				case *object.Str:
					return &object.Bytes{V: []byte(v.V)}, nil
				case *object.List:
					b := make([]byte, len(v.V))
					for idx, item := range v.V {
						if n, ok := item.(*object.Int); ok {
							b[idx] = byte(n.Int64())
						}
					}
					return &object.Bytes{V: b}, nil
				}
			}
			return &object.Bytes{V: nil}, nil
		case "_codecs_encode":
			// _codecs.encode(str, 'latin-1') → bytes
			if tup != nil && len(tup.V) == 2 {
				s, ok1 := tup.V[0].(*object.Str)
				enc, ok2 := tup.V[1].(*object.Str)
				if ok1 && ok2 && enc.V == "latin-1" {
					// Each rune in the string is a latin-1 code point.
					b := make([]byte, 0, len(s.V))
					for _, r := range s.V {
						b = append(b, byte(r))
					}
					return &object.Bytes{V: b}, nil
				}
			}
			return &object.Bytes{V: nil}, nil
		}
	}
	return nil, &pickleErr{"cannot REDUCE " + object.TypeName(callable)}
}

// ---- helpers ----------------------------------------------------------------

// pickleDecodeLong decodes little-endian two's complement bytes to *big.Int.
func pickleDecodeLong(b []byte) *big.Int {
	n := len(b)
	if n == 0 {
		return new(big.Int)
	}
	neg := b[n-1]&0x80 != 0
	if neg {
		// Invert and add 1 (two's complement).
		inv := make([]byte, n)
		copy(inv, b)
		for i := range inv {
			inv[i] ^= 0xff
		}
		// Reverse to big-endian.
		for lo, hi := 0, len(inv)-1; lo < hi; lo, hi = lo+1, hi-1 {
			inv[lo], inv[hi] = inv[hi], inv[lo]
		}
		result := new(big.Int).SetBytes(inv)
		result.Add(result, big.NewInt(1))
		result.Neg(result)
		return result
	}
	// Positive: reverse to big-endian.
	cp := make([]byte, n)
	copy(cp, b)
	for lo, hi := 0, len(cp)-1; lo < hi; lo, hi = lo+1, hi-1 {
		cp[lo], cp[hi] = cp[hi], cp[lo]
	}
	return new(big.Int).SetBytes(cp)
}

func parseFloat64(s string) (float64, error) {
	switch s {
	case "inf", "Infinity":
		return math.Inf(1), nil
	case "-inf", "-Infinity":
		return math.Inf(-1), nil
	case "nan", "NaN":
		return math.NaN(), nil
	}
	return strconv.ParseFloat(s, 64)
}

func decodeUnicodeEscapes(s string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			i++
			continue
		}
		i++
		if i >= len(s) {
			break
		}
		switch s[i] {
		case 'u':
			if i+5 <= len(s) {
				r := rune(0)
				for _, c := range s[i+1 : i+5] {
					r = r*16 + hexDigit(byte(c))
				}
				b.WriteRune(r)
				i += 5
			}
		case 'U':
			if i+9 <= len(s) {
				r := rune(0)
				for _, c := range s[i+1 : i+9] {
					r = r*16 + hexDigit(byte(c))
				}
				b.WriteRune(r)
				i += 9
			}
		case 'n':
			b.WriteByte('\n')
			i++
		case 'r':
			b.WriteByte('\r')
			i++
		case 't':
			b.WriteByte('\t')
			i++
		case '\\':
			b.WriteByte('\\')
			i++
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String(), nil
}

func hexDigit(c byte) rune {
	switch {
	case c >= '0' && c <= '9':
		return rune(c - '0')
	case c >= 'a' && c <= 'f':
		return rune(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return rune(c-'A') + 10
	}
	return 0
}

func unquotePickleString(s string) string {
	if len(s) >= 2 && (s[0] == '\'' || s[0] == '"') {
		s = s[1 : len(s)-1]
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			case 'x':
				if i+2 < len(s) {
					b.WriteByte(byte(hexDigit(s[i+1])<<4 | hexDigit(s[i+2])))
					i += 2
				}
			default:
				b.WriteByte(s[i])
			}
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

type pickleErr struct{ msg string }
type pickleEOF struct{}

func (e *pickleErr) Error() string { return e.msg }
func (e *pickleEOF) Error() string { return "unexpected end of pickle data" }

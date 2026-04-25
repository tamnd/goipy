package vm

import (
	"bufio"
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/tamnd/goipy/object"
)

// --- io module: StringIO and BytesIO ---

// buildIO exposes `io.StringIO`, `io.BytesIO`, constants and helpers.
func (i *Interp) buildIO() *object.Module {
	m := &object.Module{Name: "io", Dict: object.NewDict()}

	// Constants.
	m.Dict.SetStr("DEFAULT_BUFFER_SIZE", object.NewInt(8192))
	m.Dict.SetStr("SEEK_SET", object.NewInt(0))
	m.Dict.SetStr("SEEK_CUR", object.NewInt(1))
	m.Dict.SetStr("SEEK_END", object.NewInt(2))

	// UnsupportedOperation inherits from OSError and ValueError.
	unsupported := &object.Class{
		Name:  "UnsupportedOperation",
		Bases: []*object.Class{i.osErr, i.valueErr},
		Dict:  object.NewDict(),
	}
	m.Dict.SetStr("UnsupportedOperation", unsupported)

	// open() is the same as the built-in open().
	if v, ok := i.Builtins.GetStr("open"); ok {
		m.Dict.SetStr("open", v)
	}

	m.Dict.SetStr("StringIO", &object.BuiltinFunc{Name: "StringIO", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		sio := &object.StringIO{}
		if len(a) >= 1 && a[0] != object.None {
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "initial_value must be str")
			}
			sio.V = []byte(s.V)
		}
		return sio, nil
	}})

	m.Dict.SetStr("BytesIO", &object.BuiltinFunc{Name: "BytesIO", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		bio := &object.BytesIO{}
		if len(a) >= 1 && a[0] != object.None {
			switch v := a[0].(type) {
			case *object.Bytes:
				bio.V = append([]byte(nil), v.V...)
			case *object.Bytearray:
				bio.V = append([]byte(nil), v.V...)
			default:
				return nil, object.Errorf(i.typeErr, "initial_value must be bytes")
			}
		}
		return bio, nil
	}})

	return m
}

// stringIOAttr dispatches attribute access on a *object.StringIO.
func stringIOAttr(i *Interp, sio *object.StringIO, name string) (object.Object, bool) {
	switch name {
	case "closed":
		return object.BoolOf(sio.Closed), true
	case "write":
		return &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if sio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "write() argument must be str")
			}
			// At EOF we simply extend; otherwise overlay and possibly extend.
			if sio.Pos >= len(sio.V) {
				sio.V = append(sio.V, s.V...)
				sio.Pos = len(sio.V)
			} else {
				end := sio.Pos + len(s.V)
				if end > len(sio.V) {
					sio.V = append(sio.V[:sio.Pos], s.V...)
				} else {
					copy(sio.V[sio.Pos:], s.V)
				}
				sio.Pos = end
			}
			return object.NewInt(int64(len([]rune(s.V)))), nil
		}}, true
	case "read":
		return &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if sio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			n := -1
			if len(a) >= 1 {
				if v, ok := toInt64(a[0]); ok {
					n = int(v)
				}
			}
			start := sio.Pos
			end := len(sio.V)
			if n >= 0 && start+n < end {
				end = start + n
			}
			sio.Pos = end
			return &object.Str{V: string(sio.V[start:end])}, nil
		}}, true
	case "readline":
		return &object.BuiltinFunc{Name: "readline", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if sio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			start := sio.Pos
			end := start
			for end < len(sio.V) && sio.V[end] != '\n' {
				end++
			}
			if end < len(sio.V) {
				end++ // include the newline
			}
			sio.Pos = end
			return &object.Str{V: string(sio.V[start:end])}, nil
		}}, true
	case "readlines":
		return &object.BuiltinFunc{Name: "readlines", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if sio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			out := []object.Object{}
			for sio.Pos < len(sio.V) {
				start := sio.Pos
				end := start
				for end < len(sio.V) && sio.V[end] != '\n' {
					end++
				}
				if end < len(sio.V) {
					end++
				}
				out = append(out, &object.Str{V: string(sio.V[start:end])})
				sio.Pos = end
			}
			return &object.List{V: out}, nil
		}}, true
	case "writelines":
		return &object.BuiltinFunc{Name: "writelines", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "writelines() missing arg")
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
				s, ok := v.(*object.Str)
				if !ok {
					return nil, object.Errorf(i.typeErr, "writelines() requires str items")
				}
				sio.V = append(sio.V, s.V...)
				sio.Pos = len(sio.V)
			}
			return object.None, nil
		}}, true
	case "getvalue":
		return &object.BuiltinFunc{Name: "getvalue", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: string(sio.V)}, nil
		}}, true
	case "seek":
		return &object.BuiltinFunc{Name: "seek", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return ioSeek(&sio.Pos, len(sio.V), a)
		}}, true
	case "tell":
		return &object.BuiltinFunc{Name: "tell", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(sio.Pos)), nil
		}}, true
	case "close":
		return &object.BuiltinFunc{Name: "close", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			sio.Closed = true
			return object.None, nil
		}}, true
	case "truncate":
		return &object.BuiltinFunc{Name: "truncate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			size := sio.Pos
			if len(a) >= 1 && a[0] != object.None {
				if v, ok := toInt64(a[0]); ok {
					size = int(v)
				}
			}
			if size < len(sio.V) {
				sio.V = sio.V[:size]
			}
			return object.NewInt(int64(size)), nil
		}}, true
	case "readable", "writable", "seekable":
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}}, true
	case "__iter__":
		return &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return sio, nil
		}}, true
	case "__next__":
		return &object.BuiltinFunc{Name: "__next__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if sio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			if sio.Pos >= len(sio.V) {
				return nil, object.Errorf(i.stopIter, "")
			}
			start := sio.Pos
			end := start
			for end < len(sio.V) && sio.V[end] != '\n' {
				end++
			}
			if end < len(sio.V) {
				end++ // include the newline
			}
			sio.Pos = end
			return &object.Str{V: string(sio.V[start:end])}, nil
		}}, true
	}
	return nil, false
}

// bytesIOAttr mirrors stringIOAttr for the bytes variant.
func bytesIOAttr(i *Interp, bio *object.BytesIO, name string) (object.Object, bool) {
	switch name {
	case "closed":
		return object.BoolOf(bio.Closed), true
	case "write":
		return &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if bio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			var data []byte
			switch v := a[0].(type) {
			case *object.Bytes:
				data = v.V
			case *object.Bytearray:
				data = v.V
			default:
				return nil, object.Errorf(i.typeErr, "write() argument must be bytes")
			}
			if bio.Pos >= len(bio.V) {
				bio.V = append(bio.V, data...)
				bio.Pos = len(bio.V)
			} else {
				end := bio.Pos + len(data)
				if end > len(bio.V) {
					bio.V = append(bio.V[:bio.Pos], data...)
				} else {
					copy(bio.V[bio.Pos:], data)
				}
				bio.Pos = end
			}
			return object.NewInt(int64(len(data))), nil
		}}, true
	case "read":
		return &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if bio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			n := -1
			if len(a) >= 1 {
				if v, ok := toInt64(a[0]); ok {
					n = int(v)
				}
			}
			start := bio.Pos
			end := len(bio.V)
			if n >= 0 && start+n < end {
				end = start + n
			}
			bio.Pos = end
			return &object.Bytes{V: append([]byte(nil), bio.V[start:end]...)}, nil
		}}, true
	case "getvalue":
		return &object.BuiltinFunc{Name: "getvalue", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Bytes{V: append([]byte(nil), bio.V...)}, nil
		}}, true
	case "seek":
		return &object.BuiltinFunc{Name: "seek", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return ioSeek(&bio.Pos, len(bio.V), a)
		}}, true
	case "tell":
		return &object.BuiltinFunc{Name: "tell", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(bio.Pos)), nil
		}}, true
	case "readline":
		return &object.BuiltinFunc{Name: "readline", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if bio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			if bio.Pos >= len(bio.V) {
				return &object.Bytes{V: []byte{}}, nil
			}
			end := bio.Pos
			for end < len(bio.V) && bio.V[end] != '\n' {
				end++
			}
			if end < len(bio.V) {
				end++ // include newline
			}
			line := append([]byte(nil), bio.V[bio.Pos:end]...)
			bio.Pos = end
			return &object.Bytes{V: line}, nil
		}}, true
	case "readlines":
		return &object.BuiltinFunc{Name: "readlines", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if bio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			var out []object.Object
			for bio.Pos < len(bio.V) {
				end := bio.Pos
				for end < len(bio.V) && bio.V[end] != '\n' {
					end++
				}
				if end < len(bio.V) {
					end++
				}
				out = append(out, &object.Bytes{V: append([]byte(nil), bio.V[bio.Pos:end]...)})
				bio.Pos = end
			}
			return &object.List{V: out}, nil
		}}, true
	case "writelines":
		return &object.BuiltinFunc{Name: "writelines", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if bio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "writelines() requires 1 argument")
			}
			it, ierr := i.getIter(a[0])
			if ierr != nil {
				return nil, ierr
			}
			for {
				item, ok, nerr := it.Next()
				if nerr != nil {
					return nil, nerr
				}
				if !ok {
					break
				}
				data, e2 := asBytes(item)
				if e2 != nil {
					return nil, e2
				}
				if bio.Pos >= len(bio.V) {
					bio.V = append(bio.V, data...)
					bio.Pos = len(bio.V)
				} else {
					end := bio.Pos + len(data)
					if end > len(bio.V) {
						bio.V = append(bio.V[:bio.Pos], data...)
					} else {
						copy(bio.V[bio.Pos:], data)
					}
					bio.Pos = end
				}
			}
			return object.None, nil
		}}, true
	case "truncate":
		return &object.BuiltinFunc{Name: "truncate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if bio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			size := bio.Pos
			if len(a) >= 1 && a[0] != object.None {
				if v, ok := toInt64(a[0]); ok {
					size = int(v)
				}
			}
			if size < len(bio.V) {
				bio.V = bio.V[:size]
			}
			return object.NewInt(int64(size)), nil
		}}, true
	case "readable", "writable", "seekable":
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}}, true
	case "__iter__":
		return &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return bio, nil
		}}, true
	case "__next__":
		return &object.BuiltinFunc{Name: "__next__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if bio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			if bio.Pos >= len(bio.V) {
				return nil, object.Errorf(i.stopIter, "")
			}
			end := bio.Pos
			for end < len(bio.V) && bio.V[end] != '\n' {
				end++
			}
			if end < len(bio.V) {
				end++
			}
			line := append([]byte(nil), bio.V[bio.Pos:end]...)
			bio.Pos = end
			return &object.Bytes{V: line}, nil
		}}, true
	case "close":
		return &object.BuiltinFunc{Name: "close", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			bio.Closed = true
			return object.None, nil
		}}, true
	}
	return nil, false
}

// ioSeek implements the shared seek(offset, whence=0) semantics. Whence 0
// is absolute, 1 is relative to cursor, 2 is relative to end.
func ioSeek(pos *int, size int, a []object.Object) (object.Object, error) {
	if len(a) < 1 {
		return nil, nil
	}
	offset, _ := toInt64(a[0])
	whence := int64(0)
	if len(a) >= 2 {
		if v, ok := toInt64(a[1]); ok {
			whence = v
		}
	}
	var target int64
	switch whence {
	case 0:
		target = offset
	case 1:
		target = int64(*pos) + offset
	case 2:
		target = int64(size) + offset
	}
	if target < 0 {
		target = 0
	}
	*pos = int(target)
	return object.NewInt(target), nil
}

// --- base64 module ---

type a85Error struct{ msg string }

func (e *a85Error) Error() string { return e.msg }

// a85encodeBytes encodes src using ASCII85 encoding.
func a85encodeBytes(src []byte, foldspaces bool, wrapcol int, pad bool, adobe bool) []byte {
	if pad && len(src)%4 != 0 {
		padded := make([]byte, len(src)+4-len(src)%4)
		copy(padded, src)
		src = padded
	}

	var raw []byte

	for len(src) > 0 {
		var b [4]byte
		n := copy(b[:], src)
		src = src[n:]

		if n == 4 {
			val := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
			if val == 0 {
				raw = append(raw, 'z')
				continue
			}
			if foldspaces && val == 0x20202020 {
				raw = append(raw, 'y')
				continue
			}
			var enc [5]byte
			for j := 4; j >= 0; j-- {
				enc[j] = byte(val%85) + '!'
				val /= 85
			}
			raw = append(raw, enc[:]...)
		} else {
			// Partial group: pad with zeros, encode 5 chars, keep n+1.
			var full [4]byte
			copy(full[:], b[:n])
			val := uint32(full[0])<<24 | uint32(full[1])<<16 | uint32(full[2])<<8 | uint32(full[3])
			var enc [5]byte
			for j := 4; j >= 0; j-- {
				enc[j] = byte(val%85) + '!'
				val /= 85
			}
			raw = append(raw, enc[:n+1]...)
		}
	}

	var out []byte
	if adobe {
		out = append(out, '<', '~')
	}

	if wrapcol > 0 {
		col := 0
		if adobe {
			col = 2
		}
		for _, c := range raw {
			if col == wrapcol {
				out = append(out, '\n')
				col = 0
			}
			out = append(out, c)
			col++
		}
	} else {
		out = append(out, raw...)
	}

	if adobe {
		out = append(out, '~', '>')
	}

	return out
}

func a85decodeBytes(src []byte, foldspaces bool, adobe bool, ignorechars []byte) ([]byte, error) {
	if adobe {
		trimmed := bytes.TrimSpace(src)
		if !bytes.HasPrefix(trimmed, []byte("<~")) || !bytes.HasSuffix(trimmed, []byte("~>")) {
			return nil, &a85Error{"Adobe Ascii85 string must start with <~ and end with ~>"}
		}
		src = trimmed[2 : len(trimmed)-2]
	}

	ignoreSet := make(map[byte]bool, len(ignorechars))
	for _, c := range ignorechars {
		ignoreSet[c] = true
	}

	var out []byte
	var group [5]byte
	groupLen := 0

	flush := func(n int) error {
		// n is the number of chars in the group (2..5); output n-1 bytes.
		// Pad with 'u' to 5 chars.
		tmp := group
		for j := n; j < 5; j++ {
			tmp[j] = 'u'
		}
		val := uint32(tmp[0]-'!')*85*85*85*85 +
			uint32(tmp[1]-'!')*85*85*85 +
			uint32(tmp[2]-'!')*85*85 +
			uint32(tmp[3]-'!')*85 +
			uint32(tmp[4]-'!')
		b := [4]byte{byte(val >> 24), byte(val >> 16), byte(val >> 8), byte(val)}
		out = append(out, b[:n-1]...)
		return nil
	}

	for i := 0; i < len(src); i++ {
		c := src[i]

		if ignoreSet[c] {
			continue
		}

		if c == 'z' {
			if groupLen != 0 {
				return nil, &a85Error{"z inside Ascii85 5-tuple"}
			}
			out = append(out, 0, 0, 0, 0)
			continue
		}

		if foldspaces && c == 'y' {
			if groupLen != 0 {
				return nil, &a85Error{"y inside Ascii85 5-tuple"}
			}
			out = append(out, 0x20, 0x20, 0x20, 0x20)
			continue
		}

		if c < '!' || c > 'u' {
			return nil, &a85Error{"Non-Ascii85 digit found: " + string([]byte{c})}
		}

		group[groupLen] = c
		groupLen++

		if groupLen == 5 {
			val := uint32(group[0]-'!')*85*85*85*85 +
				uint32(group[1]-'!')*85*85*85 +
				uint32(group[2]-'!')*85*85 +
				uint32(group[3]-'!')*85 +
				uint32(group[4]-'!')
			out = append(out, byte(val>>24), byte(val>>16), byte(val>>8), byte(val))
			groupLen = 0
		}
	}

	if groupLen == 1 {
		return nil, &a85Error{"Ascii85 encoded byte sequences must be multiple of 5 in length, padded with '!' chars"}
	}

	if groupLen > 0 {
		if err := flush(groupLen); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// b85alphabet is the Base85 alphabet used by git/mercurial (RFC 1924 variant).
const b85alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!#$%&()*+-;<=>?@^_`{|}~"

func b85encodeBytes(src []byte, pad bool) []byte {
	alpha := []byte(b85alphabet)
	// Build group count.
	nGroups := (len(src) + 3) / 4
	if nGroups == 0 {
		return []byte{}
	}
	out := make([]byte, 0, nGroups*5)
	for len(src) > 0 {
		var b [4]byte
		n := copy(b[:], src)
		src = src[n:]
		val := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
		var chunk [5]byte
		for j := 4; j >= 0; j-- {
			chunk[j] = alpha[val%85]
			val /= 85
		}
		if n < 4 && !pad {
			// Emit only n+1 chars for partial group.
			out = append(out, chunk[:n+1]...)
		} else {
			out = append(out, chunk[:]...)
		}
	}
	return out
}

func b85decodeBytes(src []byte) ([]byte, error) {
	// Build reverse lookup.
	var rev [256]int
	for i := range rev {
		rev[i] = -1
	}
	for i, c := range []byte(b85alphabet) {
		rev[c] = i
	}

	// Match CPython: pad with '~' (value 84, the max) to multiple of 5,
	// decode all groups, then strip the padding bytes from the end.
	padding := (5 - len(src)%5) % 5
	padded := make([]byte, len(src)+padding)
	copy(padded, src)
	for j := len(src); j < len(padded); j++ {
		padded[j] = '~'
	}

	out := make([]byte, 0, len(padded)/5*4)
	for i := 0; i < len(padded); i += 5 {
		chunk := padded[i : i+5]
		var val uint32
		for j := 0; j < 5; j++ {
			idx := rev[chunk[j]]
			if idx < 0 {
				return nil, &a85Error{"base85: invalid character"}
			}
			val = val*85 + uint32(idx)
		}
		out = append(out, byte(val>>24), byte(val>>16), byte(val>>8), byte(val))
	}

	if padding > 0 {
		out = out[:len(out)-padding]
	}
	return out, nil
}

func (i *Interp) buildBase64() *object.Module {
	m := &object.Module{Name: "base64", Dict: object.NewDict()}

	// b64encode(s, altchars=None)
	m.Dict.SetStr("b64encode", &object.BuiltinFunc{Name: "b64encode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "b64encode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		result := []byte(base64.StdEncoding.EncodeToString(data))
		// Check altchars kwarg or positional arg[1].
		var altchars []byte
		if len(a) >= 2 && a[1] != object.None {
			altchars, err = asBytes(a[1])
			if err != nil {
				return nil, err
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("altchars"); ok && v != object.None {
				altchars, err = asBytes(v)
				if err != nil {
					return nil, err
				}
			}
		}
		if len(altchars) >= 2 {
			for idx, c := range result {
				if c == '+' {
					result[idx] = altchars[0]
				} else if c == '/' {
					result[idx] = altchars[1]
				}
			}
		}
		return &object.Bytes{V: result}, nil
	}})

	// b64decode(s, altchars=None, validate=False)
	m.Dict.SetStr("b64decode", &object.BuiltinFunc{Name: "b64decode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "b64decode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}

		var altchars []byte
		validate := false

		if len(a) >= 2 && a[1] != object.None {
			altchars, err = asBytes(a[1])
			if err != nil {
				return nil, err
			}
		}
		if len(a) >= 3 {
			validate = object.Truthy(a[2])
		}
		if kw != nil {
			if v, ok := kw.GetStr("altchars"); ok && v != object.None {
				altchars, err = asBytes(v)
				if err != nil {
					return nil, err
				}
			}
			if v, ok := kw.GetStr("validate"); ok {
				validate = object.Truthy(v)
			}
		}

		// Apply reverse altchars mapping.
		s := make([]byte, len(data))
		copy(s, data)
		if len(altchars) >= 2 {
			for idx, c := range s {
				if c == altchars[0] {
					s[idx] = '+'
				} else if c == altchars[1] {
					s[idx] = '/'
				}
			}
		}

		if validate {
			// In validate mode, non-base64 chars (except padding) raise an error.
			for _, c := range s {
				if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
					return nil, object.Errorf(i.valueErr, "Non-base64 digit found")
				}
			}
		} else {
			// Strip whitespace.
			s = bytes.Map(func(r rune) rune {
				if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
					return -1
				}
				return r
			}, s)
		}

		out, err2 := base64.StdEncoding.DecodeString(string(s))
		if err2 != nil {
			return nil, object.Errorf(i.valueErr, "base64: %s", err2.Error())
		}
		return &object.Bytes{V: out}, nil
	}})

	// urlsafe_b64encode(s)
	m.Dict.SetStr("urlsafe_b64encode", &object.BuiltinFunc{Name: "urlsafe_b64encode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "urlsafe_b64encode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		return &object.Bytes{V: []byte(base64.URLEncoding.EncodeToString(data))}, nil
	}})

	// urlsafe_b64decode(s) — adds missing padding before decoding
	m.Dict.SetStr("urlsafe_b64decode", &object.BuiltinFunc{Name: "urlsafe_b64decode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "urlsafe_b64decode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		s := make([]byte, len(data))
		copy(s, data)
		// Add missing padding.
		switch len(s) % 4 {
		case 2:
			s = append(s, '=', '=')
		case 3:
			s = append(s, '=')
		}
		out, err2 := base64.URLEncoding.DecodeString(string(s))
		if err2 != nil {
			return nil, object.Errorf(i.valueErr, "base64: %s", err2.Error())
		}
		return &object.Bytes{V: out}, nil
	}})

	// b32encode(s)
	m.Dict.SetStr("b32encode", &object.BuiltinFunc{Name: "b32encode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "b32encode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		return &object.Bytes{V: []byte(base32.StdEncoding.EncodeToString(data))}, nil
	}})

	// b32decode(s, casefold=False, map01=None)
	m.Dict.SetStr("b32decode", &object.BuiltinFunc{Name: "b32decode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "b32decode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		casefold := false
		var map01 []byte

		if len(a) >= 2 {
			casefold = object.Truthy(a[1])
		}
		if len(a) >= 3 && a[2] != object.None {
			map01, err = asBytes(a[2])
			if err != nil {
				return nil, err
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("casefold"); ok {
				casefold = object.Truthy(v)
			}
			if v, ok := kw.GetStr("map01"); ok && v != object.None {
				map01, err = asBytes(v)
				if err != nil {
					return nil, err
				}
			}
		}

		s := make([]byte, len(data))
		copy(s, data)

		if len(map01) >= 1 {
			for idx, c := range s {
				if c == '0' {
					s[idx] = 'O'
				} else if c == '1' {
					s[idx] = map01[0]
				}
			}
		}

		if casefold {
			s = bytes.ToUpper(s)
		}

		out, err2 := base32.StdEncoding.DecodeString(string(s))
		if err2 != nil {
			return nil, object.Errorf(i.valueErr, "base32: %s", err2.Error())
		}
		return &object.Bytes{V: out}, nil
	}})

	// b32hexencode(s) — extended-hex alphabet (0-9A-V)
	m.Dict.SetStr("b32hexencode", &object.BuiltinFunc{Name: "b32hexencode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "b32hexencode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		return &object.Bytes{V: []byte(base32.HexEncoding.EncodeToString(data))}, nil
	}})

	// b32hexdecode(s, casefold=False)
	m.Dict.SetStr("b32hexdecode", &object.BuiltinFunc{Name: "b32hexdecode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "b32hexdecode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		casefold := false
		if len(a) >= 2 {
			casefold = object.Truthy(a[1])
		}
		if kw != nil {
			if v, ok := kw.GetStr("casefold"); ok {
				casefold = object.Truthy(v)
			}
		}
		s := make([]byte, len(data))
		copy(s, data)
		if casefold {
			s = bytes.ToUpper(s)
		}
		out, err2 := base32.HexEncoding.DecodeString(string(s))
		if err2 != nil {
			return nil, object.Errorf(i.valueErr, "base32hex: %s", err2.Error())
		}
		return &object.Bytes{V: out}, nil
	}})

	// b16encode(s)
	m.Dict.SetStr("b16encode", &object.BuiltinFunc{Name: "b16encode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "b16encode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		return &object.Bytes{V: []byte(strings.ToUpper(hex.EncodeToString(data)))}, nil
	}})

	// b16decode(s, casefold=False)
	m.Dict.SetStr("b16decode", &object.BuiltinFunc{Name: "b16decode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "b16decode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		casefold := false
		if len(a) >= 2 {
			casefold = object.Truthy(a[1])
		}
		if kw != nil {
			if v, ok := kw.GetStr("casefold"); ok {
				casefold = object.Truthy(v)
			}
		}
		s := string(data)
		if casefold {
			s = strings.ToUpper(s)
		}
		out, err2 := hex.DecodeString(s)
		if err2 != nil {
			return nil, object.Errorf(i.valueErr, "base16: %s", err2.Error())
		}
		return &object.Bytes{V: out}, nil
	}})

	// encodebytes(s) — b64encode with newline every 76 output chars + trailing newline
	m.Dict.SetStr("encodebytes", &object.BuiltinFunc{Name: "encodebytes", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "encodebytes requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		encoded := base64.StdEncoding.EncodeToString(data)
		var out []byte
		for len(encoded) > 76 {
			out = append(out, []byte(encoded[:76])...)
			out = append(out, '\n')
			encoded = encoded[76:]
		}
		out = append(out, []byte(encoded)...)
		out = append(out, '\n')
		return &object.Bytes{V: out}, nil
	}})

	// decodebytes(s) — b64decode ignoring non-base64 chars (incl. newlines)
	m.Dict.SetStr("decodebytes", &object.BuiltinFunc{Name: "decodebytes", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "decodebytes requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		// Keep only valid base64 characters.
		s := bytes.Map(func(r rune) rune {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' {
				return r
			}
			return -1
		}, data)
		out, err2 := base64.StdEncoding.DecodeString(string(s))
		if err2 != nil {
			return nil, object.Errorf(i.valueErr, "base64: %s", err2.Error())
		}
		return &object.Bytes{V: out}, nil
	}})

	// a85encode(b, *, foldspaces=False, wrapcol=0, pad=False, adobe=False)
	m.Dict.SetStr("a85encode", &object.BuiltinFunc{Name: "a85encode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "a85encode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		foldspaces := false
		wrapcol := 0
		pad := false
		adobe := false
		if kw != nil {
			if v, ok := kw.GetStr("foldspaces"); ok {
				foldspaces = object.Truthy(v)
			}
			if v, ok := kw.GetStr("wrapcol"); ok {
				if n, ok2 := toInt64(v); ok2 {
					wrapcol = int(n)
				}
			}
			if v, ok := kw.GetStr("pad"); ok {
				pad = object.Truthy(v)
			}
			if v, ok := kw.GetStr("adobe"); ok {
				adobe = object.Truthy(v)
			}
		}
		return &object.Bytes{V: a85encodeBytes(data, foldspaces, wrapcol, pad, adobe)}, nil
	}})

	// a85decode(b, *, foldspaces=False, adobe=False, ignorechars=b' \t\n\r\v')
	m.Dict.SetStr("a85decode", &object.BuiltinFunc{Name: "a85decode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "a85decode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		foldspaces := false
		adobe := false
		ignorechars := []byte(" \t\n\r\v")
		if kw != nil {
			if v, ok := kw.GetStr("foldspaces"); ok {
				foldspaces = object.Truthy(v)
			}
			if v, ok := kw.GetStr("adobe"); ok {
				adobe = object.Truthy(v)
			}
			if v, ok := kw.GetStr("ignorechars"); ok {
				ignorechars, err = asBytes(v)
				if err != nil {
					return nil, err
				}
			}
		}
		out, err2 := a85decodeBytes(data, foldspaces, adobe, ignorechars)
		if err2 != nil {
			return nil, object.Errorf(i.valueErr, "a85decode: %s", err2.Error())
		}
		return &object.Bytes{V: out}, nil
	}})

	// b85encode(b, pad=False)
	m.Dict.SetStr("b85encode", &object.BuiltinFunc{Name: "b85encode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "b85encode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		pad := false
		if len(a) >= 2 {
			pad = object.Truthy(a[1])
		}
		if kw != nil {
			if v, ok := kw.GetStr("pad"); ok {
				pad = object.Truthy(v)
			}
		}
		return &object.Bytes{V: b85encodeBytes(data, pad)}, nil
	}})

	// b85decode(b)
	m.Dict.SetStr("b85decode", &object.BuiltinFunc{Name: "b85decode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "b85decode requires input")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		out, err2 := b85decodeBytes(data)
		if err2 != nil {
			return nil, object.Errorf(i.valueErr, "b85decode: %s", err2.Error())
		}
		return &object.Bytes{V: out}, nil
	}})

	// standard_b64encode / standard_b64decode are aliases.
	if v, ok := m.Dict.GetStr("b64encode"); ok {
		m.Dict.SetStr("standard_b64encode", v)
	}
	if v, ok := m.Dict.GetStr("b64decode"); ok {
		m.Dict.SetStr("standard_b64decode", v)
	}

	return m
}

// --- textwrap module ---

// twOpts mirrors Python's TextWrapper constructor parameters.
type twOpts struct {
	width            int
	initialIndent    string
	subsequentIndent string
	expandTabs       bool
	tabsize          int
	breakLongWords   bool
	maxLines         int    // 0 = unlimited
	placeholder      string // default " [...]"
}

func defaultTWOpts() twOpts {
	return twOpts{
		width:          70,
		expandTabs:     true,
		tabsize:        8,
		breakLongWords: true,
		placeholder:    " [...]",
	}
}

// parseTWOpts reads positional arg[1] as width and keyword args.
func parseTWOpts(a []object.Object, kw *object.Dict) twOpts {
	o := defaultTWOpts()
	if len(a) >= 2 {
		if n, ok := toInt64(a[1]); ok {
			o.width = int(n)
		}
	}
	if kw == nil {
		return o
	}
	if v, ok := kw.GetStr("width"); ok {
		if n, ok := toInt64(v); ok {
			o.width = int(n)
		}
	}
	if v, ok := kw.GetStr("initial_indent"); ok {
		if s, ok := v.(*object.Str); ok {
			o.initialIndent = s.V
		}
	}
	if v, ok := kw.GetStr("subsequent_indent"); ok {
		if s, ok := v.(*object.Str); ok {
			o.subsequentIndent = s.V
		}
	}
	if v, ok := kw.GetStr("expand_tabs"); ok {
		o.expandTabs = object.Truthy(v)
	}
	if v, ok := kw.GetStr("tabsize"); ok {
		if n, ok := toInt64(v); ok {
			o.tabsize = int(n)
		}
	}
	if v, ok := kw.GetStr("break_long_words"); ok {
		o.breakLongWords = object.Truthy(v)
	}
	if v, ok := kw.GetStr("max_lines"); ok {
		if n, ok := toInt64(v); ok {
			o.maxLines = int(n)
		}
	}
	if v, ok := kw.GetStr("placeholder"); ok {
		if s, ok := v.(*object.Str); ok {
			o.placeholder = s.V
		}
	}
	return o
}

func (i *Interp) buildTextwrap() *object.Module {
	m := &object.Module{Name: "textwrap", Dict: object.NewDict()}

	m.Dict.SetStr("dedent", &object.BuiltinFunc{Name: "dedent", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "dedent() argument must be str")
		}
		return &object.Str{V: dedent(s.V)}, nil
	}})

	m.Dict.SetStr("indent", &object.BuiltinFunc{Name: "indent", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "indent() missing args")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "indent() text must be str")
		}
		prefix, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "indent() prefix must be str")
		}
		var predicate object.Object
		if kw != nil {
			if v, ok := kw.GetStr("predicate"); ok {
				predicate = v
			}
		}
		result, err := i.indentText(s.V, prefix.V, predicate)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: result}, nil
	}})

	m.Dict.SetStr("wrap", &object.BuiltinFunc{Name: "wrap", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "wrap() missing text")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "wrap() text must be str")
		}
		opts := parseTWOpts(a, kw)
		lines := wrapFull(s.V, opts)
		out := make([]object.Object, len(lines))
		for k, l := range lines {
			out[k] = &object.Str{V: l}
		}
		return &object.List{V: out}, nil
	}})

	m.Dict.SetStr("fill", &object.BuiltinFunc{Name: "fill", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fill() missing text")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "fill() text must be str")
		}
		opts := parseTWOpts(a, kw)
		return &object.Str{V: strings.Join(wrapFull(s.V, opts), "\n")}, nil
	}})

	m.Dict.SetStr("shorten", &object.BuiltinFunc{Name: "shorten", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "shorten() missing text")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "shorten() text must be str")
		}
		opts := parseTWOpts(a, kw)
		return &object.Str{V: i.shortenText(s.V, opts)}, nil
	}})

	// TextWrapper class
	twClass := &object.Class{Name: "TextWrapper"}
	twClass.Dict = object.NewDict()
	twClass.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		// defaults
		opts := defaultTWOpts()
		// parse kw only (no positional width for TextWrapper)
		if kw != nil {
			if v, ok2 := kw.GetStr("width"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					opts.width = int(n)
				}
			}
			if v, ok2 := kw.GetStr("initial_indent"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					opts.initialIndent = s.V
				}
			}
			if v, ok2 := kw.GetStr("subsequent_indent"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					opts.subsequentIndent = s.V
				}
			}
			if v, ok2 := kw.GetStr("expand_tabs"); ok2 {
				opts.expandTabs = object.Truthy(v)
			}
			if v, ok2 := kw.GetStr("tabsize"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					opts.tabsize = int(n)
				}
			}
			if v, ok2 := kw.GetStr("break_long_words"); ok2 {
				opts.breakLongWords = object.Truthy(v)
			}
			if v, ok2 := kw.GetStr("max_lines"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					opts.maxLines = int(n)
				}
			}
			if v, ok2 := kw.GetStr("placeholder"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					opts.placeholder = s.V
				}
			}
		}
		self.Dict.SetStr("width", &object.Int{V: *new(big.Int).SetInt64(int64(opts.width))})
		self.Dict.SetStr("initial_indent", &object.Str{V: opts.initialIndent})
		self.Dict.SetStr("subsequent_indent", &object.Str{V: opts.subsequentIndent})
		self.Dict.SetStr("expand_tabs", object.BoolOf(opts.expandTabs))
		self.Dict.SetStr("tabsize", &object.Int{V: *new(big.Int).SetInt64(int64(opts.tabsize))})
		self.Dict.SetStr("break_long_words", object.BoolOf(opts.breakLongWords))
		if opts.maxLines > 0 {
			self.Dict.SetStr("max_lines", &object.Int{V: *new(big.Int).SetInt64(int64(opts.maxLines))})
		} else {
			self.Dict.SetStr("max_lines", object.None)
		}
		self.Dict.SetStr("placeholder", &object.Str{V: opts.placeholder})
		return object.None, nil
	}})
	twClass.Dict.SetStr("wrap", &object.BuiltinFunc{Name: "wrap", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "TextWrapper.wrap() missing text")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "TextWrapper.wrap() bad self")
		}
		text, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "TextWrapper.wrap() text must be str")
		}
		opts := twOptsFromInstance(self)
		lines := wrapFull(text.V, opts)
		out := make([]object.Object, len(lines))
		for k, l := range lines {
			out[k] = &object.Str{V: l}
		}
		return &object.List{V: out}, nil
	}})
	twClass.Dict.SetStr("fill", &object.BuiltinFunc{Name: "fill", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "TextWrapper.fill() missing text")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "TextWrapper.fill() bad self")
		}
		text, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "TextWrapper.fill() text must be str")
		}
		opts := twOptsFromInstance(self)
		return &object.Str{V: strings.Join(wrapFull(text.V, opts), "\n")}, nil
	}})
	m.Dict.SetStr("TextWrapper", twClass)

	return m
}

// twOptsFromInstance reads twOpts back from a TextWrapper instance's __dict__.
func twOptsFromInstance(self *object.Instance) twOpts {
	o := defaultTWOpts()
	if v, ok := self.Dict.GetStr("width"); ok {
		if n, ok2 := toInt64(v); ok2 {
			o.width = int(n)
		}
	}
	if v, ok := self.Dict.GetStr("initial_indent"); ok {
		if s, ok2 := v.(*object.Str); ok2 {
			o.initialIndent = s.V
		}
	}
	if v, ok := self.Dict.GetStr("subsequent_indent"); ok {
		if s, ok2 := v.(*object.Str); ok2 {
			o.subsequentIndent = s.V
		}
	}
	if v, ok := self.Dict.GetStr("expand_tabs"); ok {
		o.expandTabs = object.Truthy(v)
	}
	if v, ok := self.Dict.GetStr("tabsize"); ok {
		if n, ok2 := toInt64(v); ok2 {
			o.tabsize = int(n)
		}
	}
	if v, ok := self.Dict.GetStr("break_long_words"); ok {
		o.breakLongWords = object.Truthy(v)
	}
	if v, ok := self.Dict.GetStr("max_lines"); ok {
		if n, ok2 := toInt64(v); ok2 {
			o.maxLines = int(n)
		}
	}
	if v, ok := self.Dict.GetStr("placeholder"); ok {
		if s, ok2 := v.(*object.Str); ok2 {
			o.placeholder = s.V
		}
	}
	return o
}

// dedent strips the longest common leading whitespace from every non-blank
// line. Blank lines are normalized to empty, matching CPython's behavior.
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	prefix := ""
	set := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		if !set {
			prefix = indent
			set = true
		} else {
			prefix = commonPrefix(prefix, indent)
		}
	}
	if prefix == "" {
		// Still blank-line-normalize.
		for k, line := range lines {
			if strings.TrimSpace(line) == "" {
				lines[k] = ""
			}
		}
		return strings.Join(lines, "\n")
	}
	for k, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[k] = ""
		} else if strings.HasPrefix(line, prefix) {
			lines[k] = line[len(prefix):]
		}
	}
	return strings.Join(lines, "\n")
}

func commonPrefix(a, b string) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for k := 0; k < n; k++ {
		if a[k] != b[k] {
			return a[:k]
		}
	}
	return a[:n]
}

// splitlinesKeepends mimics Python's str.splitlines(keepends=True).
// A trailing newline does NOT produce a trailing empty element.
func splitlinesKeepends(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.SplitAfter(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// expandTabsStr expands tab characters to spaces using given tabsize.
func expandTabsStr(s string, tabsize int) string {
	if tabsize <= 0 {
		return strings.ReplaceAll(s, "\t", "")
	}
	var b strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			spaces := tabsize - col%tabsize
			b.WriteString(strings.Repeat(" ", spaces))
			col += spaces
		} else if r == '\n' || r == '\r' {
			b.WriteRune(r)
			col = 0
		} else {
			b.WriteRune(r)
			col++
		}
	}
	return b.String()
}

// indentText prepends prefix to lines selected by predicate (default: non-blank).
// predicate may be nil (use default) or a Python callable object.
func (i *Interp) indentText(text, prefix string, predicate object.Object) (string, error) {
	lines := splitlinesKeepends(text)
	var b strings.Builder
	for _, line := range lines {
		add := false
		if predicate == nil || predicate == object.None {
			add = strings.TrimSpace(line) != ""
		} else {
			res, err := i.callObject(predicate, []object.Object{&object.Str{V: line}}, nil)
			if err != nil {
				return "", err
			}
			add = object.Truthy(res)
		}
		if add {
			b.WriteString(prefix)
		}
		b.WriteString(line)
	}
	return b.String(), nil
}

// wrapFull is a full CPython-compatible text wrapper.
func wrapFull(text string, o twOpts) []string {
	if o.expandTabs {
		ts := o.tabsize
		if ts <= 0 {
			ts = 8
		}
		text = expandTabsStr(text, ts)
	}

	// CPython preserves leading whitespace on the first line (drop_whitespace
	// only drops leading whitespace on lines after the first).
	leadWS := ""
	trimmed := strings.TrimLeft(text, " \t")
	if len(trimmed) < len(text) {
		leadWS = text[:len(text)-len(trimmed)]
		text = trimmed
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	ph := o.placeholder
	if ph == "" {
		ph = " [...]"
	}
	width := o.width
	if width < 1 {
		width = 1
	}

	// Work on a mutable copy so we can split long words inline.
	ws := make([]string, len(words))
	copy(ws, words)

	var lines []string
	// Leading whitespace merges with initial_indent (applies to first line only).
	indent := leadWS + o.initialIndent

	wi := 0
	for wi < len(ws) {
		lineIndent := indent
		lineW := lineIndent
		isLast := o.maxLines > 0 && len(lines) == o.maxLines-1
		startWi := wi

		// Fill this line greedily.
		for wi < len(ws) {
			w := ws[wi]
			sep := " "
			if lineW == lineIndent {
				sep = ""
			}
			fits := len(lineW)+len(sep)+len(w) <= width
			if fits {
				lineW += sep + w
				wi++
			} else if lineW == lineIndent {
				// First word on line; it's too long.
				if o.breakLongWords {
					rem := width - len(lineIndent)
					if rem < 1 {
						rem = 1
					}
					lineW += w[:rem]
					tail := w[rem:]
					if tail == "" {
						wi++
					} else {
						// Splice remainder back into ws.
						ws = append(ws[:wi], append([]string{tail}, ws[wi+1:]...)...)
					}
				} else {
					lineW += w
					wi++
				}
				break
			} else {
				break // word doesn't fit; flush line
			}
		}

		// If this is the last allowed line and there are still words left,
		// truncate with placeholder.
		if isLast && wi < len(ws) {
			actualPh := ph
			if lineW == lineIndent {
				// Nothing on line yet: strip leading spaces from placeholder.
				actualPh = strings.TrimLeft(ph, " ")
			}
			// Backtrack words until placeholder fits.
			for len(lineW)+len(actualPh) > width && lineW != lineIndent {
				idx := strings.LastIndex(lineW[len(lineIndent):], " ")
				if idx < 0 {
					lineW = lineIndent
					actualPh = strings.TrimLeft(ph, " ")
					break
				}
				lineW = lineW[:len(lineIndent)+idx]
				if lineW == lineIndent {
					actualPh = strings.TrimLeft(ph, " ")
				}
			}
			lineW += actualPh
		}

		if lineW != lineIndent || wi > startWi {
			lines = append(lines, lineW)
		} else {
			break // safety: no progress
		}

		indent = o.subsequentIndent
		if isLast {
			break
		}
	}

	return lines
}

// shortenText collapses whitespace and wraps to a single line with placeholder.
func (i *Interp) shortenText(text string, o twOpts) string {
	text = strings.Join(strings.Fields(text), " ")
	o.maxLines = 1
	lines := wrapFull(text, o)
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

// fileAttr dispatches attribute/method access on an *object.File (from open()).
func fileAttr(i *Interp, f *object.File, name string) (object.Object, bool) {
	osf := f.F.(*os.File)
	switch name {
	case "name":
		return &object.Str{V: f.FilePath}, true
	case "mode":
		return &object.Str{V: f.Mode}, true
	case "closed":
		return object.BoolOf(f.Closed), true
	case "read":
		return &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if f.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			var data []byte
			var err error
			if len(a) >= 1 {
				n, ok := a[0].(*object.Int)
				if !ok {
					return nil, object.Errorf(i.typeErr, "read() argument must be int")
				}
				sz := n.V.Int64()
				if sz < 0 {
					data, err = io.ReadAll(osf)
				} else {
					data = make([]byte, sz)
					nr, e := osf.Read(data)
					data = data[:nr]
					err = e
					if err == io.EOF {
						err = nil
					}
				}
			} else {
				data, err = io.ReadAll(osf)
			}
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			if f.Binary {
				return &object.Bytes{V: data}, nil
			}
			return &object.Str{V: string(data)}, nil
		}}, true
	case "readline":
		return &object.BuiltinFunc{Name: "readline", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if f.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			rd := bufio.NewReader(osf)
			line, err := rd.ReadString('\n')
			if err != nil && err != io.EOF {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			if f.Binary {
				return &object.Bytes{V: []byte(line)}, nil
			}
			return &object.Str{V: line}, nil
		}}, true
	case "readlines":
		return &object.BuiltinFunc{Name: "readlines", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if f.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			data, err := io.ReadAll(osf)
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			var out []object.Object
			remaining := data
			for len(remaining) > 0 {
				idx := strings.IndexByte(string(remaining), '\n')
				var line []byte
				if idx < 0 {
					line = remaining
					remaining = nil
				} else {
					line = remaining[:idx+1]
					remaining = remaining[idx+1:]
				}
				if f.Binary {
					out = append(out, &object.Bytes{V: append([]byte(nil), line...)})
				} else {
					out = append(out, &object.Str{V: string(line)})
				}
			}
			return &object.List{V: out}, nil
		}}, true
	case "write":
		return &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if f.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "write() requires 1 argument")
			}
			var data []byte
			if f.Binary {
				switch v := a[0].(type) {
				case *object.Bytes:
					data = v.V
				case *object.Bytearray:
					data = v.V
				default:
					return nil, object.Errorf(i.typeErr, "write() argument must be bytes-like in binary mode")
				}
			} else {
				s, ok := a[0].(*object.Str)
				if !ok {
					return nil, object.Errorf(i.typeErr, "write() argument must be str")
				}
				data = []byte(s.V)
			}
			n, err := osf.Write(data)
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			return object.NewInt(int64(n)), nil
		}}, true
	case "writelines":
		return &object.BuiltinFunc{Name: "writelines", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if f.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "writelines() requires 1 argument")
			}
			it, ierr := i.getIter(a[0])
			if ierr != nil {
				return nil, ierr
			}
			for {
				item, ok2, nerr := it.Next()
				if nerr != nil {
					return nil, nerr
				}
				if !ok2 {
					break
				}
				var data []byte
				if f.Binary {
					b, ok3 := item.(*object.Bytes)
					if !ok3 {
						return nil, object.Errorf(i.typeErr, "writelines() items must be bytes in binary mode")
					}
					data = b.V
				} else {
					s, ok3 := item.(*object.Str)
					if !ok3 {
						return nil, object.Errorf(i.typeErr, "writelines() items must be str")
					}
					data = []byte(s.V)
				}
				if _, werr := osf.Write(data); werr != nil {
					return nil, object.Errorf(i.osErr, "%v", werr)
				}
			}
			return object.None, nil
		}}, true
	case "seek":
		return &object.BuiltinFunc{Name: "seek", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if f.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "seek() requires at least 1 argument")
			}
			offset, _ := toInt64(a[0])
			whence := 0
			if len(a) >= 2 {
				if w, ok2 := toInt64(a[1]); ok2 {
					whence = int(w)
				}
			}
			pos, err := osf.Seek(offset, whence)
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			return object.NewInt(pos), nil
		}}, true
	case "tell":
		return &object.BuiltinFunc{Name: "tell", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if f.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			pos, err := osf.Seek(0, io.SeekCurrent)
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			return object.NewInt(pos), nil
		}}, true
	case "truncate":
		return &object.BuiltinFunc{Name: "truncate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if f.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			var size int64
			if len(a) >= 1 && a[0] != object.None {
				v, ok2 := toInt64(a[0])
				if !ok2 {
					return nil, object.Errorf(i.typeErr, "truncate() argument must be int")
				}
				size = v
			} else {
				var err error
				size, err = osf.Seek(0, io.SeekCurrent)
				if err != nil {
					return nil, object.Errorf(i.osErr, "%v", err)
				}
			}
			if err := osf.Truncate(size); err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			return object.NewInt(size), nil
		}}, true
	case "fileno":
		return &object.BuiltinFunc{Name: "fileno", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if f.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			return object.NewInt(int64(osf.Fd())), nil
		}}, true
	case "isatty":
		return &object.BuiltinFunc{Name: "isatty", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		}}, true
	case "readable":
		return &object.BuiltinFunc{Name: "readable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			m := f.Mode
			return object.BoolOf(m == "" || strings.ContainsAny(m, "r+")), nil
		}}, true
	case "writable":
		return &object.BuiltinFunc{Name: "writable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(strings.ContainsAny(f.Mode, "wxa+")), nil
		}}, true
	case "seekable":
		return &object.BuiltinFunc{Name: "seekable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}}, true
	case "encoding":
		if f.Binary {
			return object.None, true
		}
		return &object.Str{V: "utf-8"}, true
	case "errors":
		if f.Binary {
			return object.None, true
		}
		return &object.Str{V: "strict"}, true
	case "__iter__":
		return &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return f, nil
		}}, true
	case "__next__":
		return &object.BuiltinFunc{Name: "__next__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if f.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			rd := bufio.NewReader(osf)
			line, err := rd.ReadString('\n')
			if err == io.EOF && line == "" {
				return nil, object.Errorf(i.stopIter, "")
			}
			if err != nil && err != io.EOF {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			if f.Binary {
				return &object.Bytes{V: []byte(line)}, nil
			}
			return &object.Str{V: line}, nil
		}}, true
	case "flush":
		return &object.BuiltinFunc{Name: "flush", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}, true
	case "close":
		return &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if !f.Closed {
				f.Closed = true
				osf.Close()
			}
			return object.None, nil
		}}, true
	case "__enter__":
		return &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return f, nil
		}}, true
	case "__exit__":
		return &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if !f.Closed {
				f.Closed = true
				osf.Close()
			}
			return object.False, nil
		}}, true
	}
	return nil, false
}

// fileIter returns an *object.Iter that reads one line at a time from f.
// It is used by getIter when a *object.File appears in a for-loop.
func (i *Interp) fileIter(f *object.File) *object.Iter {
	osf := f.F.(*os.File)
	rd := bufio.NewReader(osf)
	return &object.Iter{Next: func() (object.Object, bool, error) {
		if f.Closed {
			return nil, false, nil
		}
		line, err := rd.ReadString('\n')
		if err == io.EOF && line == "" {
			return nil, false, nil
		}
		if err != nil && err != io.EOF {
			return nil, false, object.Errorf(i.osErr, "%v", err)
		}
		if f.Binary {
			return &object.Bytes{V: []byte(line)}, true, nil
		}
		return &object.Str{V: line}, true, nil
	}}
}

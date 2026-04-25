package vm

import (
	"bytes"
	"fmt"
	"io"
	"mime/quotedprintable"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildQuopri() *object.Module {
	m := &object.Module{Name: "quopri", Dict: object.NewDict()}

	// encodestring(s, quotetabs=False, header=False) -> bytes
	m.Dict.SetStr("encodestring", &object.BuiltinFunc{Name: "encodestring", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "encodestring() missing required argument: 's'")
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "encodestring() argument 1 must be bytes-like")
		}
		quotetabs := false
		header := false
		if len(args) >= 2 {
			quotetabs = isTruthy(args[1])
		} else if kw != nil {
			if v, ok := kw.GetStr("quotetabs"); ok {
				quotetabs = isTruthy(v)
			}
		}
		if len(args) >= 3 {
			header = isTruthy(args[2])
		} else if kw != nil {
			if v, ok := kw.GetStr("header"); ok {
				header = isTruthy(v)
			}
		}
		return &object.Bytes{V: qpEncode(data, quotetabs, header)}, nil
	}})

	// decodestring(s, header=False) -> bytes
	m.Dict.SetStr("decodestring", &object.BuiltinFunc{Name: "decodestring", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "decodestring() missing required argument: 's'")
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "decodestring() argument 1 must be bytes-like")
		}
		header := false
		if len(args) >= 2 {
			header = isTruthy(args[1])
		} else if kw != nil {
			if v, ok := kw.GetStr("header"); ok {
				header = isTruthy(v)
			}
		}
		out, err := qpDecode(data, header)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "decodestring() decode error: %v", err)
		}
		return &object.Bytes{V: out}, nil
	}})

	// encode(input, output, quotetabs, header=False)
	m.Dict.SetStr("encode", &object.BuiltinFunc{Name: "encode", Call: func(interp any, args []object.Object, kw *object.Dict) (object.Object, error) {
		if len(args) < 3 {
			return nil, object.Errorf(i.typeErr, "encode() requires input, output, and quotetabs arguments")
		}
		ii := interp.(*Interp)

		// Read from input file-like object.
		data, err := qpReadFilelike(ii, args[0])
		if err != nil {
			return nil, err
		}

		quotetabs := isTruthy(args[2])
		header := false
		if len(args) >= 4 {
			header = isTruthy(args[3])
		} else if kw != nil {
			if v, ok := kw.GetStr("header"); ok {
				header = isTruthy(v)
			}
		}

		encoded := qpEncode(data, quotetabs, header)

		// Write to output file-like object.
		if err := qpWriteFilelike(ii, args[1], encoded); err != nil {
			return nil, err
		}
		return object.None, nil
	}})

	// decode(input, output, header=False)
	m.Dict.SetStr("decode", &object.BuiltinFunc{Name: "decode", Call: func(interp any, args []object.Object, kw *object.Dict) (object.Object, error) {
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "decode() requires input and output arguments")
		}
		ii := interp.(*Interp)

		// Read from input file-like object.
		data, err := qpReadFilelike(ii, args[0])
		if err != nil {
			return nil, err
		}

		header := false
		if len(args) >= 3 {
			header = isTruthy(args[2])
		} else if kw != nil {
			if v, ok := kw.GetStr("header"); ok {
				header = isTruthy(v)
			}
		}

		decoded, err := qpDecode(data, header)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "decode() error: %v", err)
		}

		// Write to output file-like object.
		if err := qpWriteFilelike(ii, args[1], decoded); err != nil {
			return nil, err
		}
		return object.None, nil
	}})

	return m
}

// isTruthy returns whether an object is truthy.
func isTruthy(o object.Object) bool {
	switch v := o.(type) {
	case *object.Bool:
		return v == object.True
	case *object.Int:
		return v.Int64() != 0
	case *object.NoneType:
		return false
	}
	return true
}

// qpReadFilelike reads all bytes from a Python file-like object.
func qpReadFilelike(ii *Interp, obj object.Object) ([]byte, error) {
	readFn, err := ii.getAttr(obj, "read")
	if err != nil {
		return nil, object.Errorf(nil, "encode/decode: input has no read() method")
	}
	result, err := ii.callObject(readFn, nil, nil)
	if err != nil {
		return nil, err
	}
	switch v := result.(type) {
	case *object.Bytes:
		return v.V, nil
	case *object.Bytearray:
		return v.V, nil
	case *object.Str:
		return []byte(v.V), nil
	}
	return nil, object.Errorf(nil, "encode/decode: read() did not return bytes")
}

// qpWriteFilelike writes bytes to a Python file-like object.
func qpWriteFilelike(ii *Interp, obj object.Object, data []byte) error {
	writeFn, err := ii.getAttr(obj, "write")
	if err != nil {
		return object.Errorf(nil, "encode/decode: output has no write() method")
	}
	_, err = ii.callObject(writeFn, []object.Object{&object.Bytes{V: data}}, nil)
	return err
}

// qpEncode encodes data using quoted-printable encoding.
// quotetabs=True encodes embedded spaces and tabs.
// header=True encodes spaces as underscores and underscores as =5F.
func qpEncode(input []byte, quotetabs, header bool) []byte {
	const maxLine = 76

	var out bytes.Buffer

	// Split into lines preserving line endings.
	lines := qpSplitLines(input)

	for lineIdx, lineBytes := range lines {
		// Separate the trailing newline from line content.
		eol := []byte(nil)
		content := lineBytes
		if len(content) > 0 && content[len(content)-1] == '\n' {
			if len(content) > 1 && content[len(content)-2] == '\r' {
				eol = []byte("\r\n")
				content = content[:len(content)-2]
			} else {
				eol = []byte("\n")
				content = content[:len(content)-1]
			}
		}

		// Build the encoded tokens for this line.
		// We collect tokens then handle soft line wrapping.
		lineLen := 0

		emitToken := func(tok []byte) {
			// If adding this token would exceed maxLine, emit a soft break first.
			// A soft break is '=' followed by '\n' (or '\r\n' to match EOL).
			// The soft break itself counts as 1 char ('=') on the current line.
			if lineLen+len(tok) > maxLine {
				out.WriteByte('=')
				out.Write([]byte("\n"))
				lineLen = 0
			}
			out.Write(tok)
			lineLen += len(tok)
		}

		for idx := 0; idx < len(content); idx++ {
			b := content[idx]
			var tok []byte

			if header && b == ' ' {
				tok = []byte{'_'}
			} else if header && b == '_' {
				tok = []byte("=5F")
			} else if b == '=' {
				tok = []byte("=3D")
			} else if b == '\t' || b == ' ' {
				// Check if this is a trailing whitespace character.
				isTrailing := true
				for j := idx + 1; j < len(content); j++ {
					if content[j] != ' ' && content[j] != '\t' {
						isTrailing = false
						break
					}
				}
				// Encode if trailing OR if quotetabs is set.
				if isTrailing || quotetabs {
					tok = []byte(fmt.Sprintf("=%02X", b))
				} else {
					tok = []byte{b}
				}
			} else if b < 0x21 || b > 0x7E {
				tok = []byte(fmt.Sprintf("=%02X", b))
			} else {
				tok = []byte{b}
			}

			emitToken(tok)
		}

		// Emit the line ending (or nothing for the last line if no newline).
		if eol != nil {
			out.Write(eol)
		} else if lineIdx < len(lines)-1 {
			// No trailing newline recorded but there are more lines - shouldn't happen
			// with splitLines, but be safe.
			out.WriteByte('\n')
		}
	}

	return out.Bytes()
}

// qpSplitLines splits input into lines, keeping the line ending attached to each line.
// The last segment may or may not have a newline.
func qpSplitLines(input []byte) [][]byte {
	if len(input) == 0 {
		return nil
	}
	var lines [][]byte
	start := 0
	for i := 0; i < len(input); i++ {
		if input[i] == '\n' {
			lines = append(lines, input[start:i+1])
			start = i + 1
		}
	}
	if start < len(input) {
		lines = append(lines, input[start:])
	}
	return lines
}

// qpDecode decodes quoted-printable encoded data.
// header=True decodes underscores as spaces.
func qpDecode(input []byte, header bool) ([]byte, error) {
	if header {
		// Replace underscores with spaces before QP decoding.
		input = bytes.ReplaceAll(input, []byte{'_'}, []byte{' '})
	}
	r := quotedprintable.NewReader(bytes.NewReader(input))
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return out, nil
}

package marshal

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/tamnd/goipy/object"
)

// Magic314 is the 4-byte magic for CPython 3.14 .pyc files.
var Magic314 = [4]byte{0x2b, 0x0e, 0x0d, 0x0a}

// LoadPyc reads the .pyc at path and returns the top-level code object.
func LoadPyc(path string) (*object.Code, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Decode(b)
}

// Decode parses a .pyc file (header + marshal stream).
func Decode(b []byte) (*object.Code, error) {
	if len(b) < 16 {
		return nil, fmt.Errorf("pyc: too short (%d bytes)", len(b))
	}
	if b[0] != Magic314[0] || b[1] != Magic314[1] || b[2] != Magic314[2] || b[3] != Magic314[3] {
		return nil, fmt.Errorf("pyc: unsupported magic %02x%02x%02x%02x (need 3.14 = %02x%02x%02x%02x)",
			b[0], b[1], b[2], b[3], Magic314[0], Magic314[1], Magic314[2], Magic314[3])
	}
	_ = binary.LittleEndian.Uint32(b[4:8]) // flags; ignore
	// Skip 16-byte header (magic + flags + 8 bytes validation).
	o, err := Unmarshal(b[16:])
	if err != nil {
		return nil, err
	}
	c, ok := o.(*object.Code)
	if !ok {
		return nil, fmt.Errorf("pyc: top-level object is %T, want code", o)
	}
	return c, nil
}

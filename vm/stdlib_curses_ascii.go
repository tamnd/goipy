package vm

import (
	"unicode"

	"github.com/tamnd/goipy/object"
)

// asciiOrd extracts an integer ordinal from an int, str, or bytes value.
func asciiOrd(v object.Object) (int, bool) {
	switch x := v.(type) {
	case *object.Int:
		if n, ok := toInt64(x); ok {
			return int(n), true
		}
	case *object.Str:
		if len(x.V) > 0 {
			r := []rune(x.V)
			return int(r[0]), true
		}
	case *object.Bytes:
		if len(x.V) > 0 {
			return int(x.V[0]), true
		}
	}
	return 0, false
}

// buildCursesAscii constructs the curses.ascii submodule.
func (i *Interp) buildCursesAscii() *object.Module {
	m := &object.Module{Name: "curses.ascii", Dict: object.NewDict()}

	// --- Constants ---
	consts := map[string]int64{
		"NUL": 0, "SOH": 1, "STX": 2, "ETX": 3, "EOT": 4, "ENQ": 5, "ACK": 6,
		"BEL": 7, "BS": 8, "TAB": 9, "HT": 9, "LF": 10, "NL": 10, "VT": 11,
		"FF": 12, "CR": 13, "SO": 14, "SI": 15, "DLE": 16, "DC1": 17, "DC2": 18,
		"DC3": 19, "DC4": 20, "NAK": 21, "SYN": 22, "ETB": 23, "CAN": 24,
		"EM": 25, "SUB": 26, "ESC": 27, "FS": 28, "GS": 29, "RS": 30, "US": 31,
		"SP": 32, "DEL": 127, "EOF": -1,
	}
	for k, v := range consts {
		m.Dict.SetStr(k, object.NewInt(v))
	}

	// Helper: make a bool classification function.
	boolFn := func(name string, fn func(int) bool) {
		m.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.False, nil
			}
			c, ok := asciiOrd(a[0])
			if !ok {
				return object.False, nil
			}
			return object.BoolOf(fn(c)), nil
		}})
	}

	boolFn("isalnum", func(c int) bool {
		r := rune(c)
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	})
	boolFn("isalpha", func(c int) bool {
		return unicode.IsLetter(rune(c))
	})
	boolFn("isascii", func(c int) bool {
		return c >= 0 && c <= 127
	})
	boolFn("isblank", func(c int) bool {
		return c == ' ' || c == '\t'
	})
	boolFn("iscntrl", func(c int) bool {
		return (c >= 0 && c <= 31) || c == 127
	})
	boolFn("isdigit", func(c int) bool {
		return c >= '0' && c <= '9'
	})
	boolFn("isgraph", func(c int) bool {
		return c > 32 && c < 127
	})
	boolFn("islower", func(c int) bool {
		return c >= 'a' && c <= 'z'
	})
	boolFn("isprint", func(c int) bool {
		return c >= 32 && c < 127
	})
	boolFn("ispunct", func(c int) bool {
		r := rune(c)
		return unicode.IsPunct(r) || unicode.IsSymbol(r)
	})
	boolFn("isspace", func(c int) bool {
		return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
	})
	boolFn("isupper", func(c int) bool {
		return c >= 'A' && c <= 'Z'
	})
	boolFn("isxdigit", func(c int) bool {
		return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
	})

	// toascii(c) → c & 0x7F
	m.Dict.SetStr("toascii", &object.BuiltinFunc{Name: "toascii", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		c, _ := asciiOrd(a[0])
		return object.NewInt(int64(c & 0x7F)), nil
	}})

	// alt(c) → c | 0x80
	m.Dict.SetStr("alt", &object.BuiltinFunc{Name: "alt", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		c, _ := asciiOrd(a[0])
		return object.NewInt(int64(c | 0x80)), nil
	}})

	// ctrl(c) → c & 0x1F
	m.Dict.SetStr("ctrl", &object.BuiltinFunc{Name: "ctrl", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		c, _ := asciiOrd(a[0])
		return object.NewInt(int64(c & 0x1F)), nil
	}})

	// unctrl(c) → str representation
	m.Dict.SetStr("unctrl", &object.BuiltinFunc{Name: "unctrl", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		c, _ := asciiOrd(a[0])
		if c < 32 {
			return &object.Str{V: "^" + string(rune('A'+c-1))}, nil
		}
		if c == 127 {
			return &object.Str{V: "^?"}, nil
		}
		return &object.Str{V: string(rune(c))}, nil
	}})

	// ascii(c) → int ordinal
	m.Dict.SetStr("ascii", &object.BuiltinFunc{Name: "ascii", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		c, _ := asciiOrd(a[0])
		return object.NewInt(int64(c)), nil
	}})

	return m
}

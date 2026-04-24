package vm

import (
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

	// Pure ASCII classification helpers (not Unicode-aware).
	isdigit := func(c int) bool { return c >= 48 && c <= 57 }
	isupper := func(c int) bool { return c >= 65 && c <= 90 }
	islower := func(c int) bool { return c >= 97 && c <= 122 }
	isalpha := func(c int) bool { return isupper(c) || islower(c) }
	isalnum := func(c int) bool { return isalpha(c) || isdigit(c) }
	isgraph := func(c int) bool { return c >= 33 && c <= 126 }

	// Helper: register a bool classification function from a predicate.
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

	boolFn("isalnum", isalnum)
	boolFn("isalpha", isalpha)
	boolFn("isascii", func(c int) bool { return c >= 0 && c <= 127 })
	boolFn("isblank", func(c int) bool { return c == 32 || c == 9 })
	boolFn("iscntrl", func(c int) bool { return (c >= 0 && c <= 31) || c == 127 })
	boolFn("isdigit", isdigit)
	boolFn("isgraph", isgraph)
	boolFn("islower", islower)
	boolFn("isprint", func(c int) bool { return c >= 32 && c <= 126 })
	boolFn("ispunct", func(c int) bool { return isgraph(c) && !isalnum(c) })
	boolFn("isspace", func(c int) bool {
		return c == 32 || c == 9 || c == 10 || c == 11 || c == 12 || c == 13
	})
	boolFn("isupper", isupper)
	boolFn("isxdigit", func(c int) bool {
		return isdigit(c) || (c >= 97 && c <= 102) || (c >= 65 && c <= 70)
	})

	// ascii(c) → int ordinal
	m.Dict.SetStr("ascii", &object.BuiltinFunc{Name: "ascii", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		c, _ := asciiOrd(a[0])
		return object.NewInt(int64(c)), nil
	}})

	// toascii(c) → c & 0x7F
	m.Dict.SetStr("toascii", &object.BuiltinFunc{Name: "toascii", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		c, _ := asciiOrd(a[0])
		return object.NewInt(int64(c & 0x7F)), nil
	}})

	// ctrl(c) → c & 0x1F
	m.Dict.SetStr("ctrl", &object.BuiltinFunc{Name: "ctrl", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		c, _ := asciiOrd(a[0])
		return object.NewInt(int64(c & 0x1F)), nil
	}})

	// alt(c) → c | 0x80
	m.Dict.SetStr("alt", &object.BuiltinFunc{Name: "alt", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(0), nil
		}
		c, _ := asciiOrd(a[0])
		return object.NewInt(int64(c | 0x80)), nil
	}})

	// unctrl(c) → printable string representation
	m.Dict.SetStr("unctrl", &object.BuiltinFunc{Name: "unctrl", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		c, _ := asciiOrd(a[0])
		var s string
		switch {
		case c >= 32 && c <= 126:
			// printable
			s = string(rune(c))
		case c == 127:
			// DEL
			s = "^?"
		case c >= 0 && c < 32:
			// control character: c+64 gives the corresponding uppercase letter
			s = "^" + string(rune(c+64))
		case c > 127:
			// high-bit set: prefix with '!' and recurse on low 7 bits
			s = "!" + string(rune(c&0x7F))
		default:
			s = "?"
		}
		return &object.Str{V: s}, nil
	}})

	return m
}

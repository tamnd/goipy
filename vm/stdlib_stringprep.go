package vm

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/bidi"
	"golang.org/x/text/unicode/norm"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildStringprep() *object.Module {
	m := &object.Module{Name: "stringprep", Dict: object.NewDict()}

	// Helper: extract single rune from argument, including WTF-8 surrogates.
	getChar := func(a []object.Object, name string) (rune, error) {
		if len(a) < 1 {
			return 0, object.Errorf(i.typeErr, "%s() missing argument", name)
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return 0, object.Errorf(i.typeErr, "%s() argument must be str", name)
		}
		r, sz := decodeRuneWTF8(s.V)
		if r == utf8.RuneError && sz <= 1 {
			return 0, object.Errorf(i.typeErr, "%s() argument must be a unicode character", name)
		}
		return r, nil
	}

	boolFn := func(name string, pred func(rune) bool) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			r, err := getChar(a, name)
			if err != nil {
				return nil, err
			}
			return object.BoolOf(pred(r)), nil
		}}
	}

	// in_table_a1: unassigned code points in Unicode 3.2
	m.Dict.SetStr("in_table_a1", boolFn("in_table_a1", inTableA1))

	// in_table_b1: commonly mapped to nothing
	m.Dict.SetStr("in_table_b1", boolFn("in_table_b1", inTableB1))

	// map_table_b2: case-folding with NFKC
	m.Dict.SetStr("map_table_b2", &object.BuiltinFunc{Name: "map_table_b2", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "map_table_b2() missing argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "map_table_b2() argument must be str")
		}
		return &object.Str{V: mapTableB2(s.V)}, nil
	}})

	// map_table_b3: case-folding without normalization
	m.Dict.SetStr("map_table_b3", &object.BuiltinFunc{Name: "map_table_b3", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "map_table_b3() missing argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "map_table_b3() argument must be str")
		}
		return &object.Str{V: mapTableB3(s.V)}, nil
	}})

	// in_table_c11: ASCII space (U+0020 only)
	m.Dict.SetStr("in_table_c11", boolFn("in_table_c11", func(r rune) bool { return r == 0x0020 }))

	// in_table_c12: non-ASCII space (category Zs, not U+0020)
	m.Dict.SetStr("in_table_c12", boolFn("in_table_c12", func(r rune) bool {
		return r != 0x0020 && unicode.Is(unicode.Zs, r)
	}))

	// in_table_c11_c12: any space (category Zs)
	m.Dict.SetStr("in_table_c11_c12", boolFn("in_table_c11_c12", func(r rune) bool {
		return unicode.Is(unicode.Zs, r)
	}))

	// in_table_c21: ASCII control characters
	m.Dict.SetStr("in_table_c21", boolFn("in_table_c21", func(r rune) bool {
		return r < 128 && unicode.Is(unicode.Cc, r)
	}))

	// in_table_c22: non-ASCII control characters + special set
	m.Dict.SetStr("in_table_c22", boolFn("in_table_c22", func(r rune) bool {
		return inTableC22(r)
	}))

	// in_table_c21_c22: all control characters
	m.Dict.SetStr("in_table_c21_c22", boolFn("in_table_c21_c22", func(r rune) bool {
		return unicode.Is(unicode.Cc, r) || inC22Specials(r)
	}))

	// in_table_c3: private use
	m.Dict.SetStr("in_table_c3", boolFn("in_table_c3", func(r rune) bool {
		return unicode.Is(unicode.Co, r)
	}))

	// in_table_c4: non-character code points
	m.Dict.SetStr("in_table_c4", boolFn("in_table_c4", func(r rune) bool {
		c := int(r)
		if c < 0xFDD0 {
			return false
		}
		if c < 0xFDF0 {
			return true
		}
		return (c&0xFFFF) == 0xFFFE || (c&0xFFFF) == 0xFFFF
	}))

	// in_table_c5: surrogate codes
	m.Dict.SetStr("in_table_c5", boolFn("in_table_c5", func(r rune) bool {
		return unicode.Is(unicode.Cs, r)
	}))

	// in_table_c6: inappropriate for plain text (U+FFF9–U+FFFD)
	m.Dict.SetStr("in_table_c6", boolFn("in_table_c6", func(r rune) bool {
		return r >= 0xFFF9 && r <= 0xFFFD
	}))

	// in_table_c7: inappropriate for canonical representation (U+2FF0–U+2FFB)
	m.Dict.SetStr("in_table_c7", boolFn("in_table_c7", func(r rune) bool {
		return r >= 0x2FF0 && r <= 0x2FFB
	}))

	// in_table_c8: change display properties or deprecated
	// {832, 833, 8206, 8207} + range(8234,8239) + range(8298,8304)
	m.Dict.SetStr("in_table_c8", boolFn("in_table_c8", func(r rune) bool {
		return inTableC8(r)
	}))

	// in_table_c9: tagging characters {E0001} + range(E0020,E0080)
	m.Dict.SetStr("in_table_c9", boolFn("in_table_c9", func(r rune) bool {
		return r == 0xE0001 || (r >= 0xE0020 && r <= 0xE007F)
	}))

	// in_table_d1: characters with bidi property R or AL
	m.Dict.SetStr("in_table_d1", boolFn("in_table_d1", func(r rune) bool {
		p, _ := bidi.LookupRune(r)
		c := p.Class()
		return c == bidi.R || c == bidi.AL
	}))

	// in_table_d2: characters with bidi property L
	m.Dict.SetStr("in_table_d2", boolFn("in_table_d2", func(r rune) bool {
		p, _ := bidi.LookupRune(r)
		return p.Class() == bidi.L
	}))

	return m
}

// decodeRuneWTF8 decodes the first rune from s, accepting WTF-8 surrogates
// (lone surrogate codepoints that CPython serialises as 3-byte CESU-8 sequences).
func decodeRuneWTF8(s string) (rune, int) {
	if len(s) >= 3 {
		b := s[0:3]
		if b[0] == 0xED && b[1]&0xE0 == 0xA0 && b[2]&0xC0 == 0x80 {
			r := rune(b[0]&0x0F)<<12 | rune(b[1]&0x3F)<<6 | rune(b[2]&0x3F)
			if r >= 0xD800 && r <= 0xDFFF {
				return r, 3
			}
		}
	}
	return utf8.DecodeRuneInString(s)
}

// inTableA1: RFC 3454 Table A.1 — unassigned code points in Unicode 3.2.
func inTableA1(r rune) bool {
	i := sort.Search(len(tableA1), func(k int) bool { return tableA1[k][1] >= r })
	return i < len(tableA1) && r >= tableA1[i][0]
}

// inTableB1: RFC 3454 Table B.1 — commonly mapped to nothing.
// Set: {0x00AD, 0x034F, 0x1806, 0x180B-0x180D, 0x200B-0x200D, 0x2060,
//
//	0xFE00-0xFE0F, 0xFEFF}
func inTableB1(r rune) bool {
	switch {
	case r == 0x00AD: // SOFT HYPHEN
		return true
	case r == 0x034F: // COMBINING GRAPHEME JOINER
		return true
	case r == 0x1806: // MONGOLIAN TODO SOFT HYPHEN
		return true
	case r >= 0x180B && r <= 0x180D: // VARIATION SELECTORS
		return true
	case r >= 0x200B && r <= 0x200D: // ZERO WIDTH SPACE/NON-JOINER/JOINER
		return true
	case r == 0x2060: // WORD JOINER
		return true
	case r >= 0xFE00 && r <= 0xFE0F: // VARIATION SELECTORS
		return true
	case r == 0xFEFF: // ZERO WIDTH NO-BREAK SPACE
		return true
	}
	return false
}

// mapTableB3: RFC 3454 Table B.3 — case folding, no normalization.
func mapTableB3(ch string) string {
	if ch == "" {
		return ch
	}
	r, _ := utf8.DecodeRuneInString(ch)
	if v, ok := b3Exceptions[r]; ok {
		return v
	}
	return strings.ToLower(ch)
}

// mapTableB2: RFC 3454 Table B.2 — case folding with NFKC.
func mapTableB2(ch string) string {
	al := mapTableB3(ch)
	b := norm.NFKC.String(al)
	var bl strings.Builder
	for _, c := range b {
		bl.WriteString(mapTableB3(string(c)))
	}
	c := norm.NFKC.String(bl.String())
	if b != c {
		return c
	}
	return al
}

// c22_specials: {1757, 1807, 6158, 8204, 8205, 8232, 8233, 65279}
//   - range(8288, 8292) + range(8298, 8304)
//   - range(65529, 65533) + range(119155, 119163)
func inC22Specials(r rune) bool {
	switch r {
	case 1757, 1807, 6158, 8204, 8205, 8232, 8233, 65279:
		return true
	}
	return (r >= 8288 && r < 8292) ||
		(r >= 8298 && r < 8304) ||
		(r >= 65529 && r < 65533) ||
		(r >= 119155 && r < 119163)
}

func inTableC22(r rune) bool {
	if r < 128 {
		return false
	}
	if unicode.Is(unicode.Cc, r) {
		return true
	}
	return inC22Specials(r)
}

// inTableC8: {832, 833, 8206, 8207} + range(8234,8239) + range(8298,8304)
func inTableC8(r rune) bool {
	switch r {
	case 832, 833, 8206, 8207:
		return true
	}
	return (r >= 8234 && r < 8239) || (r >= 8298 && r < 8304)
}

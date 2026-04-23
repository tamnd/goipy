package vm

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/bidi"
	"golang.org/x/text/unicode/norm"
	"golang.org/x/text/unicode/runenames"
	"golang.org/x/text/width"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildUnicodedata() *object.Module {
	m := &object.Module{Name: "unicodedata", Dict: object.NewDict()}

	m.Dict.SetStr("unidata_version", &object.Str{V: "16.0.0"})

	// name(chr[, default])
	m.Dict.SetStr("name", &object.BuiltinFunc{Name: "name", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "name() missing argument")
		}
		r, err := singleRune(i, a[0], "name")
		if err != nil {
			return nil, err
		}
		n := unicodeName(r)
		if n == "" {
			if len(a) >= 2 {
				return a[1], nil
			}
			return nil, object.Errorf(i.valueErr, "no such name")
		}
		return &object.Str{V: n}, nil
	}})

	// lookup(name)
	m.Dict.SetStr("lookup", &object.BuiltinFunc{Name: "lookup", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "lookup() missing argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "lookup() argument must be str")
		}
		r, ok2 := unicodeLookup(s.V)
		if !ok2 {
			return nil, object.Errorf(i.keyErr, "undefined character name '%s'", s.V)
		}
		return &object.Str{V: string(r)}, nil
	}})

	// decimal(chr[, default])
	m.Dict.SetStr("decimal", &object.BuiltinFunc{Name: "decimal", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "decimal() missing argument")
		}
		r, err := singleRune(i, a[0], "decimal")
		if err != nil {
			return nil, err
		}
		if v, ok := decimalValue(r); ok {
			return object.NewInt(int64(v)), nil
		}
		if len(a) >= 2 {
			return a[1], nil
		}
		return nil, object.Errorf(i.valueErr, "not a decimal")
	}})

	// digit(chr[, default])
	m.Dict.SetStr("digit", &object.BuiltinFunc{Name: "digit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "digit() missing argument")
		}
		r, err := singleRune(i, a[0], "digit")
		if err != nil {
			return nil, err
		}
		if v, ok := digitValue(r); ok {
			return object.NewInt(int64(v)), nil
		}
		if len(a) >= 2 {
			return a[1], nil
		}
		return nil, object.Errorf(i.valueErr, "not a digit")
	}})

	// numeric(chr[, default])
	m.Dict.SetStr("numeric", &object.BuiltinFunc{Name: "numeric", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "numeric() missing argument")
		}
		r, err := singleRune(i, a[0], "numeric")
		if err != nil {
			return nil, err
		}
		if v, ok := numericValue(r); ok {
			return &object.Float{V: v}, nil
		}
		if len(a) >= 2 {
			return a[1], nil
		}
		return nil, object.Errorf(i.valueErr, "not a numeric character")
	}})

	// category(chr)
	m.Dict.SetStr("category", &object.BuiltinFunc{Name: "category", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "category() missing argument")
		}
		r, err := singleRune(i, a[0], "category")
		if err != nil {
			return nil, err
		}
		return &object.Str{V: unicodeCategory(r)}, nil
	}})

	// bidirectional(chr)
	m.Dict.SetStr("bidirectional", &object.BuiltinFunc{Name: "bidirectional", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "bidirectional() missing argument")
		}
		r, err := singleRune(i, a[0], "bidirectional")
		if err != nil {
			return nil, err
		}
		return &object.Str{V: unicodeBidi(r)}, nil
	}})

	// combining(chr)
	m.Dict.SetStr("combining", &object.BuiltinFunc{Name: "combining", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "combining() missing argument")
		}
		r, err := singleRune(i, a[0], "combining")
		if err != nil {
			return nil, err
		}
		ccc := norm.NFD.Properties([]byte(string(r))).CCC()
		return object.NewInt(int64(ccc)), nil
	}})

	// east_asian_width(chr)
	m.Dict.SetStr("east_asian_width", &object.BuiltinFunc{Name: "east_asian_width", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "east_asian_width() missing argument")
		}
		r, err := singleRune(i, a[0], "east_asian_width")
		if err != nil {
			return nil, err
		}
		k := width.LookupRune(r).Kind()
		names := [...]string{"N", "A", "W", "Na", "F", "H"}
		if int(k) < len(names) {
			return &object.Str{V: names[k]}, nil
		}
		return &object.Str{V: "N"}, nil
	}})

	// mirrored(chr)
	m.Dict.SetStr("mirrored", &object.BuiltinFunc{Name: "mirrored", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "mirrored() missing argument")
		}
		r, err := singleRune(i, a[0], "mirrored")
		if err != nil {
			return nil, err
		}
		if isMirrored(r) {
			return object.NewInt(1), nil
		}
		return object.NewInt(0), nil
	}})

	// decomposition(chr)
	m.Dict.SetStr("decomposition", &object.BuiltinFunc{Name: "decomposition", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "decomposition() missing argument")
		}
		r, err := singleRune(i, a[0], "decomposition")
		if err != nil {
			return nil, err
		}
		return &object.Str{V: unicodeDecomposition(r)}, nil
	}})

	// normalize(form, unistr)
	m.Dict.SetStr("normalize", &object.BuiltinFunc{Name: "normalize", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "normalize() requires 2 arguments")
		}
		form, ok1 := a[0].(*object.Str)
		text, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "normalize() requires str arguments")
		}
		f, err := normForm(i, form.V)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: f.String(text.V)}, nil
	}})

	// is_normalized(form, unistr)
	m.Dict.SetStr("is_normalized", &object.BuiltinFunc{Name: "is_normalized", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "is_normalized() requires 2 arguments")
		}
		form, ok1 := a[0].(*object.Str)
		text, ok2 := a[1].(*object.Str)
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "is_normalized() requires str arguments")
		}
		f, err := normForm(i, form.V)
		if err != nil {
			return nil, err
		}
		return object.BoolOf(f.IsNormal([]byte(text.V))), nil
	}})

	return m
}

// singleRune extracts a single rune from an object.Str, erroring if it's not
// exactly one Unicode character.
func singleRune(i *Interp, v object.Object, fnName string) (rune, error) {
	s, ok := v.(*object.Str)
	if !ok {
		return 0, object.Errorf(i.typeErr, "%s() argument must be a unicode character, not '%T'", fnName, v)
	}
	r, sz := utf8.DecodeRuneInString(s.V)
	if r == utf8.RuneError && sz <= 1 {
		return 0, object.Errorf(i.typeErr, "%s() argument must be a unicode character", fnName)
	}
	if len(s.V) != sz {
		return 0, object.Errorf(i.typeErr, "%s() argument must be a unicode character, not a string", fnName)
	}
	return r, nil
}

// normForm maps a Python normalization form name to norm.Form.
func normForm(i *Interp, name string) (norm.Form, error) {
	switch name {
	case "NFC":
		return norm.NFC, nil
	case "NFD":
		return norm.NFD, nil
	case "NFKC":
		return norm.NFKC, nil
	case "NFKD":
		return norm.NFKD, nil
	}
	return norm.NFC, object.Errorf(i.valueErr, "invalid normalization form")
}

// --- name / lookup ---

var (
	nameMapOnce sync.Once
	nameToRune  map[string]rune
)

func buildNameMap() {
	nameMapOnce.Do(func() {
		nameToRune = make(map[string]rune, 40000)
		for r := rune(0); r <= 0x10FFFF; r++ {
			n := runenames.Name(r)
			if n != "" && !strings.HasPrefix(n, "<") {
				nameToRune[n] = r
			}
		}
	})
}

// unicodeName returns the Unicode name for rune r, or "" if it has no name.
func unicodeName(r rune) string {
	// Hangul syllables: algorithmic name
	if r >= 0xAC00 && r <= 0xD7A3 {
		return hangulSyllableName(r)
	}
	// CJK unified ideographs: "CJK UNIFIED IDEOGRAPH-XXXX"
	if isCJKUnified(r) {
		return fmt.Sprintf("CJK UNIFIED IDEOGRAPH-%X", r)
	}
	n := runenames.Name(r)
	if n == "" || strings.HasPrefix(n, "<") {
		return ""
	}
	return n
}

// unicodeLookup finds a rune by its Unicode name.
func unicodeLookup(name string) (rune, bool) {
	// CJK UNIFIED IDEOGRAPH-XXXX
	if strings.HasPrefix(name, "CJK UNIFIED IDEOGRAPH-") {
		hex := name[len("CJK UNIFIED IDEOGRAPH-"):]
		var cp int
		if _, err := fmt.Sscanf(hex, "%X", &cp); err == nil {
			r := rune(cp)
			if isCJKUnified(r) {
				return r, true
			}
		}
	}
	// Hangul syllables
	if strings.HasPrefix(name, "HANGUL SYLLABLE ") {
		if r, ok := lookupHangul(name[len("HANGUL SYLLABLE "):]); ok {
			return r, true
		}
	}
	buildNameMap()
	r, ok := nameToRune[name]
	return r, ok
}

var cjkRanges = [][2]rune{
	{0x3400, 0x4DBF},   // Extension A
	{0x4E00, 0x9FFF},   // CJK Unified Ideographs
	{0x20000, 0x2A6DF}, // Extension B
	{0x2A700, 0x2B73F}, // Extension C
	{0x2B740, 0x2B81F}, // Extension D
	{0x2B820, 0x2CEAF}, // Extension E
	{0x2CEB0, 0x2EBEF}, // Extension F
	{0x30000, 0x3134F}, // Extension G
	{0x31350, 0x323AF}, // Extension H
	{0xF900, 0xFAFF},   // CJK Compatibility Ideographs
	{0x2F800, 0x2FA1F}, // CJK Compatibility Ideographs Supplement
}

func isCJKUnified(r rune) bool {
	for _, ran := range cjkRanges {
		if r >= ran[0] && r <= ran[1] {
			return true
		}
	}
	return false
}

// Hangul syllable decomposition constants.
const (
	hangulBase = 0xAC00
	jamoTCount = 28
	jamoVCount = 21
	jamoNCount = jamoVCount * jamoTCount
)

var jamoLeadNames = []string{
	"G", "GG", "N", "D", "DD", "R", "M", "B", "BB", "S", "SS",
	"", "J", "JJ", "C", "K", "T", "P", "H",
}
var jamoVowelNames = []string{
	"A", "AE", "YA", "YAE", "EO", "E", "YEO", "YE", "O", "WA", "WAE",
	"OE", "YO", "U", "WEO", "WE", "WI", "YU", "EU", "YI", "I",
}
var jamoTrailNames = []string{
	"", "G", "GG", "GS", "N", "NJ", "NH", "D", "L", "LG", "LM",
	"LB", "LS", "LT", "LP", "LH", "M", "B", "BS", "S", "SS", "NG",
	"J", "C", "K", "T", "P", "H",
}

func hangulSyllableName(r rune) string {
	idx := int(r - hangulBase)
	l := idx / jamoNCount
	v := (idx % jamoNCount) / jamoTCount
	t := idx % jamoTCount
	return "HANGUL SYLLABLE " + jamoLeadNames[l] + jamoVowelNames[v] + jamoTrailNames[t]
}

func lookupHangul(tail string) (rune, bool) {
	for l, ln := range jamoLeadNames {
		if !strings.HasPrefix(tail, ln) {
			continue
		}
		rest := tail[len(ln):]
		for v, vn := range jamoVowelNames {
			if !strings.HasPrefix(rest, vn) {
				continue
			}
			rest2 := rest[len(vn):]
			for t, tn := range jamoTrailNames {
				if rest2 == tn {
					r := hangulBase + rune(l*jamoNCount+v*jamoTCount+t)
					return r, true
				}
			}
		}
	}
	return 0, false
}

// --- decimal / digit / numeric ---

func decimalValue(r rune) (int, bool) {
	i := sort.Search(len(decimalBlocks), func(k int) bool {
		return decimalBlocks[k][1] >= r
	})
	if i < len(decimalBlocks) && r >= decimalBlocks[i][0] && r <= decimalBlocks[i][1] {
		return int(r - decimalBlocks[i][0]), true
	}
	return 0, false
}

func digitValue(r rune) (int, bool) {
	if v, ok := decimalValue(r); ok {
		return v, true
	}
	v, ok := digitExtra[r]
	return v, ok
}

func numericValue(r rune) (float64, bool) {
	if v, ok := digitValue(r); ok {
		return float64(v), true
	}
	v, ok := numericExtra[r]
	return v, ok
}

// --- category ---

func unicodeCategory(r rune) string {
	switch {
	case unicode.Is(unicode.Lu, r):
		return "Lu"
	case unicode.Is(unicode.Ll, r):
		return "Ll"
	case unicode.Is(unicode.Lt, r):
		return "Lt"
	case unicode.Is(unicode.Lm, r):
		return "Lm"
	case unicode.Is(unicode.Lo, r):
		return "Lo"
	case unicode.Is(unicode.Mn, r):
		return "Mn"
	case unicode.Is(unicode.Mc, r):
		return "Mc"
	case unicode.Is(unicode.Me, r):
		return "Me"
	case unicode.Is(unicode.Nd, r):
		return "Nd"
	case unicode.Is(unicode.Nl, r):
		return "Nl"
	case unicode.Is(unicode.No, r):
		return "No"
	case unicode.Is(unicode.Pc, r):
		return "Pc"
	case unicode.Is(unicode.Pd, r):
		return "Pd"
	case unicode.Is(unicode.Ps, r):
		return "Ps"
	case unicode.Is(unicode.Pe, r):
		return "Pe"
	case unicode.Is(unicode.Pi, r):
		return "Pi"
	case unicode.Is(unicode.Pf, r):
		return "Pf"
	case unicode.Is(unicode.Po, r):
		return "Po"
	case unicode.Is(unicode.Sm, r):
		return "Sm"
	case unicode.Is(unicode.Sc, r):
		return "Sc"
	case unicode.Is(unicode.Sk, r):
		return "Sk"
	case unicode.Is(unicode.So, r):
		return "So"
	case unicode.Is(unicode.Zs, r):
		return "Zs"
	case unicode.Is(unicode.Zl, r):
		return "Zl"
	case unicode.Is(unicode.Zp, r):
		return "Zp"
	case unicode.Is(unicode.Cc, r):
		return "Cc"
	case unicode.Is(unicode.Cf, r):
		return "Cf"
	case unicode.Is(unicode.Cs, r):
		return "Cs"
	case unicode.Is(unicode.Co, r):
		return "Co"
	default:
		return "Cn"
	}
}

// --- bidirectional ---

var bidiNames = [...]string{
	"L", "R", "EN", "ES", "ET", "AN", "CS", "B", "S", "WS", "ON",
	"BN", "NSM", "AL", "LRE", "LRO", "RLE", "RLO", "PDF",
	"LRI", "RLI", "FSI", "PDI",
}

func unicodeBidi(r rune) string {
	p, _ := bidi.LookupRune(r)
	c := int(p.Class())
	if c < len(bidiNames) {
		return bidiNames[c]
	}
	return ""
}

// --- mirrored ---

func isMirrored(r rune) bool {
	i := sort.Search(len(mirroredRanges), func(k int) bool {
		return mirroredRanges[k][1] >= r
	})
	return i < len(mirroredRanges) && r >= mirroredRanges[i][0]
}

// --- decomposition ---

func unicodeDecomposition(r rune) string {
	b := []byte(string(r))

	// Canonical decomposition via NFD.
	dNFD := norm.NFD.Properties(b).Decomposition()
	if len(dNFD) > 0 {
		return runeSliceToHex(dNFD)
	}

	// Compatibility decomposition: look up type tag, then use NFKD bytes.
	tag, hasTag := unicodeCompatType[r]
	if !hasTag {
		return ""
	}
	dNFKD := norm.NFKD.Properties(b).Decomposition()
	if len(dNFKD) == 0 {
		return ""
	}
	return "<" + tag + "> " + runeSliceToHex(dNFKD)
}

func runeSliceToHex(b []byte) string {
	var parts []string
	for len(b) > 0 {
		r, sz := utf8.DecodeRune(b)
		parts = append(parts, fmt.Sprintf("%04X", r))
		b = b[sz:]
	}
	return strings.Join(parts, " ")
}

package object

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// Repr returns Python-style repr.
func Repr(o Object) string {
	if _, ok := o.(*Instance); ok && InstanceReprHook != nil {
		if s, handled := InstanceReprHook(o); handled {
			return s
		}
	}
	switch v := o.(type) {
	case nil:
		return "None"
	case *NoneType:
		return "None"
	case *EllipsisType:
		return "Ellipsis"
	case *NotImplementedType:
		return "NotImplemented"
	case *Bool:
		if v.V {
			return "True"
		}
		return "False"
	case *Int:
		return v.V.String()
	case *Float:
		return formatFloat(v.V)
	case *Complex:
		return formatComplex(v.Real, v.Imag)
	case *Str:
		return pyStrRepr(v.V)
	case *Bytes:
		return "b" + pyBytesQuote(v.V)
	case *Bytearray:
		return "bytearray(b" + pyBytesQuote(v.V) + ")"
	case *Memoryview:
		return "<memory>"
	case *Tuple:
		parts := make([]string, len(v.V))
		for i, x := range v.V {
			parts[i] = Repr(x)
		}
		if len(parts) == 1 {
			return "(" + parts[0] + ",)"
		}
		return "(" + strings.Join(parts, ", ") + ")"
	case *List:
		parts := make([]string, len(v.V))
		for i, x := range v.V {
			parts[i] = Repr(x)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *Dict:
		parts := make([]string, 0, v.Len())
		for i, k := range v.keys {
			parts = append(parts, Repr(k)+": "+Repr(v.vals[i]))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case *Set:
		if v.Len() == 0 {
			return "set()"
		}
		parts := make([]string, len(v.items))
		for i, x := range v.items {
			parts[i] = Repr(x)
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case *Frozenset:
		if v.Len() == 0 {
			return "frozenset()"
		}
		parts := make([]string, len(v.items))
		for i, x := range v.items {
			parts[i] = Repr(x)
		}
		return "frozenset({" + strings.Join(parts, ", ") + "})"
	case *Range:
		if v.Step == 1 {
			return "range(" + strconv.FormatInt(v.Start, 10) + ", " + strconv.FormatInt(v.Stop, 10) + ")"
		}
		return "range(" + strconv.FormatInt(v.Start, 10) + ", " + strconv.FormatInt(v.Stop, 10) + ", " + strconv.FormatInt(v.Step, 10) + ")"
	case *Function:
		return "<function " + v.QualName + ">"
	case *BuiltinFunc:
		return "<built-in function " + v.Name + ">"
	case *BoundMethod:
		return "<bound method>"
	case *Class:
		return "<class '" + v.Name + "'>"
	case *Instance:
		return "<" + v.Class.Name + " object>"
	case *Module:
		return "<module '" + v.Name + "'>"
	case *Slice:
		return "slice(" + Repr(v.Start) + ", " + Repr(v.Stop) + ", " + Repr(v.Step) + ")"
	case *Deque:
		parts := make([]string, len(v.V))
		for i, x := range v.V {
			parts[i] = Repr(x)
		}
		if v.MaxLen >= 0 {
			return "deque([" + strings.Join(parts, ", ") + "], maxlen=" + strconv.Itoa(v.MaxLen) + ")"
		}
		return "deque([" + strings.Join(parts, ", ") + "])"
	case *Counter:
		parts := make([]string, 0, v.D.Len())
		for i, k := range v.D.keys {
			parts = append(parts, Repr(k)+": "+Repr(v.D.vals[i]))
		}
		return "Counter({" + strings.Join(parts, ", ") + "})"
	case *DefaultDict:
		parts := make([]string, 0, v.D.Len())
		for i, k := range v.D.keys {
			parts = append(parts, Repr(k)+": "+Repr(v.D.vals[i]))
		}
		factoryRepr := "None"
		if v.Factory != nil {
			factoryRepr = Repr(v.Factory)
		}
		return "defaultdict(" + factoryRepr + ", {" + strings.Join(parts, ", ") + "})"
	case *OrderedDict:
		if v.D.Len() == 0 {
			return "OrderedDict()"
		}
		parts := make([]string, 0, v.D.Len())
		for i, k := range v.D.keys {
			parts = append(parts, "("+Repr(k)+", "+Repr(v.D.vals[i])+")")
		}
		return "OrderedDict([" + strings.Join(parts, ", ") + "])"
	case *StringIO:
		return fmt.Sprintf("<_io.StringIO object at 0x%p>", v)
	case *BytesIO:
		return fmt.Sprintf("<_io.BytesIO object at 0x%p>", v)
	case *Hasher:
		return fmt.Sprintf("<%s _hashlib.HASH object>", v.Name)
	case *CSVReader:
		return fmt.Sprintf("<_csv.reader object at 0x%p>", v)
	case *CSVWriter:
		return fmt.Sprintf("<_csv.writer object at 0x%p>", v)
	case *CSVDictWriter:
		return fmt.Sprintf("<csv.DictWriter object at 0x%p>", v)
	case *URLParseResult:
		return fmt.Sprintf("ParseResult(scheme=%s, netloc=%s, path=%s, params=%s, query=%s, fragment=%s)",
			Repr(&Str{V: v.Scheme}), Repr(&Str{V: v.Netloc}), Repr(&Str{V: v.Path}),
			Repr(&Str{V: v.Params}), Repr(&Str{V: v.Query}), Repr(&Str{V: v.Fragment}))
	case *UUID:
		return "UUID('" + uuidHyphenated(v.Bytes) + "')"
	case *Pattern:
		return "re.compile(" + Repr(&Str{V: v.Pattern}) + ")"
	case *Match:
		s, e := -1, -1
		if len(v.Offsets) >= 2 {
			s, e = v.Offsets[0], v.Offsets[1]
		}
		val := ""
		if s >= 0 && e >= 0 && e <= len(v.String) {
			val = v.String[s:e]
		}
		return fmt.Sprintf("<re.Match object; span=(%d, %d), match=%s>", s, e, Repr(&Str{V: val}))
	case *Exception:
		name := "Exception"
		if v.Class != nil {
			name = v.Class.Name
		}
		if v.Args == nil || len(v.Args.V) == 0 {
			return name + "()"
		}
		parts := make([]string, len(v.Args.V))
		for i, x := range v.Args.V {
			parts[i] = Repr(x)
		}
		return name + "(" + strings.Join(parts, ", ") + ")"
	}
	return "<?>"
}

// Str returns Python-style str().
// pyBytesQuote formats a bytes literal body the way CPython does: single
// quotes by default, double quotes if the payload contains a single quote
// but no double quote. Non-printable and non-ASCII bytes are escaped.
func pyBytesQuote(v []byte) string {
	hasSingle := false
	hasDouble := false
	for _, c := range v {
		if c == '\'' {
			hasSingle = true
		}
		if c == '"' {
			hasDouble = true
		}
	}
	quote := byte('\'')
	if hasSingle && !hasDouble {
		quote = '"'
	}
	const hex = "0123456789abcdef"
	buf := make([]byte, 0, len(v)+2)
	buf = append(buf, quote)
	for _, c := range v {
		switch {
		case c == quote:
			buf = append(buf, '\\', c)
		case c == '\\':
			buf = append(buf, '\\', '\\')
		case c == '\t':
			buf = append(buf, '\\', 't')
		case c == '\n':
			buf = append(buf, '\\', 'n')
		case c == '\r':
			buf = append(buf, '\\', 'r')
		case c < 0x20 || c >= 0x7f:
			buf = append(buf, '\\', 'x', hex[c>>4], hex[c&0xf])
		default:
			buf = append(buf, c)
		}
	}
	buf = append(buf, quote)
	return string(buf)
}

// uuidHyphenated renders a UUID byte array as 8-4-4-4-12 hex.
func uuidHyphenated(b [16]byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 36)
	j := 0
	for i := 0; i < 16; i++ {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out[j] = '-'
			j++
		}
		out[j] = hex[b[i]>>4]
		out[j+1] = hex[b[i]&0x0f]
		j += 2
	}
	return string(out)
}

func Str_(o Object) string {
	if _, ok := o.(*Instance); ok && InstanceStrHook != nil {
		if s, handled := InstanceStrHook(o); handled {
			return s
		}
	}
	switch v := o.(type) {
	case *Str:
		return v.V
	case *UUID:
		return uuidHyphenated(v.Bytes)
	case *Bytes:
		// str(b'...') returns the repr in Python, but we'll be lenient.
		return "b" + pyBytesQuote(v.V)
	case *Bytearray:
		return "bytearray(b" + pyBytesQuote(v.V) + ")"
	case *Exception:
		// str(exc): single arg → that arg's str; no args → empty; many → tuple repr.
		if v.Args == nil || len(v.Args.V) == 0 {
			return ""
		}
		if len(v.Args.V) == 1 {
			return Str_(v.Args.V[0])
		}
		return Repr(v.Args)
	}
	return Repr(o)
}

// formatComplex renders a Python-style complex repr:
//   - bare imag form when real is +0.0 (e.g. "2j", "0j")
//   - parenthesised "(r+ij)" / "(r-ij)" otherwise, including signed-zero
//     real like "(-0-2j)".
func formatComplex(re, im float64) string {
	if re == 0 && !isNegZero(re) {
		return formatComplexComponent(im) + "j"
	}
	sign := "+"
	imStr := formatComplexComponent(im)
	if len(imStr) > 0 && imStr[0] == '-' {
		sign = "-"
		imStr = imStr[1:]
	}
	return "(" + formatComplexComponent(re) + sign + imStr + "j)"
}

func formatComplexComponent(f float64) string {
	if math.IsInf(f, 1) {
		return "inf"
	}
	if math.IsInf(f, -1) {
		return "-inf"
	}
	if math.IsNaN(f) {
		return "nan"
	}
	s := strconv.FormatFloat(f, 'g', -1, 64)
	return s
}

func isNegZero(f float64) bool {
	return f == 0 && math.Signbit(f)
}

func formatFloat(f float64) string {
	if math.IsInf(f, 1) {
		return "inf"
	}
	if math.IsInf(f, -1) {
		return "-inf"
	}
	if math.IsNaN(f) {
		return "nan"
	}
	if f == 0 {
		if math.Signbit(f) {
			return "-0.0"
		}
		return "0.0"
	}
	// Python's float repr: shortest round-trip digits, scientific notation
	// only when decimal-point position <= -4 or > 16. Go's 'g' switches at
	// exponent >= 6 with prec=-1, so do the conversion ourselves.
	shortest := strconv.FormatFloat(f, 'e', -1, 64)
	mant, exp := splitSciParts(shortest)
	digits := strings.Replace(mant, ".", "", 1)
	dp := exp + 1 // position of decimal point after the first digit shift
	if dp <= -4 || dp > 16 {
		// Scientific: format "d.ddde±NN" with min 2-digit exponent.
		s := strconv.FormatFloat(f, 'e', -1, 64)
		return fixupSciExp(s)
	}
	// Fixed: render digits with decimal point at dp.
	neg := ""
	if digits[0] == '-' {
		neg = "-"
		digits = digits[1:]
	}
	var out string
	switch {
	case dp <= 0:
		out = "0." + strings.Repeat("0", -dp) + digits
	case dp >= len(digits):
		out = digits + strings.Repeat("0", dp-len(digits)) + ".0"
	default:
		out = digits[:dp] + "." + digits[dp:]
	}
	return neg + out
}

// splitSciParts splits "m.mmme±NN" into mantissa and integer exponent.
func splitSciParts(s string) (string, int) {
	i := strings.IndexAny(s, "eE")
	if i < 0 {
		return s, 0
	}
	e, _ := strconv.Atoi(s[i+1:])
	return s[:i], e
}

// fixupSciExp rewrites Go's "1e+09" to Python's "1e+09" (same) but ensures
// at least 2 exponent digits and adds ".0" to bare integer mantissa.
func fixupSciExp(s string) string {
	i := strings.IndexAny(s, "eE")
	if i < 0 {
		return s
	}
	mant, exp := s[:i], s[i+1:]
	if !strings.Contains(mant, ".") {
		mant += ".0"
	}
	sign := "+"
	if exp[0] == '+' || exp[0] == '-' {
		sign = string(exp[0])
		exp = exp[1:]
	}
	if len(exp) < 2 {
		exp = "0" + exp
	}
	return mant + "e" + sign + exp
}

// pyStrRepr formats a string using Python repr() rules: single quotes
// unless the string contains single quotes and no double quotes, then
// double quotes; with Python-style escapes.
func pyStrRepr(s string) string {
	hasSingle := strings.ContainsRune(s, '\'')
	hasDouble := strings.ContainsRune(s, '"')
	quote := byte('\'')
	if hasSingle && !hasDouble {
		quote = '"'
	}
	var b strings.Builder
	b.WriteByte(quote)
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case rune(quote):
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			if r < 0x20 || r == 0x7f {
				b.WriteString(`\x`)
				b.WriteString(strconv.FormatInt(int64(r), 16))
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte(quote)
	return b.String()
}

// AsBig returns a big.Int copy of an Int.
func AsBig(i *Int) *big.Int { return new(big.Int).Set(&i.V) }

package object

import (
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

func Str_(o Object) string {
	if _, ok := o.(*Instance); ok && InstanceStrHook != nil {
		if s, handled := InstanceStrHook(o); handled {
			return s
		}
	}
	switch v := o.(type) {
	case *Str:
		return v.V
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
	// Python prints floats with minimum digits to round-trip, and at least
	// one decimal (e.g. "1.0" not "1").
	s := strconv.FormatFloat(f, 'g', 17, 64)
	// Round-trip minimal.
	s2 := strconv.FormatFloat(f, 'g', -1, 64)
	if p, err := strconv.ParseFloat(s2, 64); err == nil && p == f {
		s = s2
	}
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
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
func AsBig(i *Int) *big.Int { return new(big.Int).Set(i.V) }

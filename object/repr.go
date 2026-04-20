package object

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// Repr returns Python-style repr.
func Repr(o Object) string {
	switch v := o.(type) {
	case nil:
		return "None"
	case *NoneType:
		return "None"
	case *Bool:
		if v.V {
			return "True"
		}
		return "False"
	case *Int:
		return v.V.String()
	case *Float:
		return formatFloat(v.V)
	case *Str:
		return pyStrRepr(v.V)
	case *Bytes:
		return "b" + strconv.Quote(string(v.V))
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
	}
	return "<?>"
}

// Str returns Python-style str().
func Str_(o Object) string {
	switch v := o.(type) {
	case *Str:
		return v.V
	case *Bytes:
		// str(b'...') returns the repr in Python, but we'll be lenient.
		return "b" + strconv.Quote(string(v.V))
	}
	return Repr(o)
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

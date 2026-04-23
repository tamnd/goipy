package object

// Interpolation represents a single interpolated value in a PEP 750 template
// string (t"..."). It corresponds to string.templatelib.Interpolation.
// Conversion is "" for no conversion, "s", "r", or "a" otherwise.
type Interpolation struct {
	Value      Object
	Expression string
	Conversion string
	FormatSpec string
}

// Template is the result of a PEP 750 t"..." literal. It corresponds to
// string.templatelib.Template. Strings and Interpolations interleave:
// len(Strings) == len(Interpolations)+1.
type Template struct {
	Strings        []*Str
	Interpolations []*Interpolation
}

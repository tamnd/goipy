package object

// Property is the object produced by @property. Fget is the getter; Fset and
// Fdel are optional setter/deleter replacements.
type Property struct {
	Fget Object
	Fset Object
	Fdel Object
}

// ClassMethod turns its wrapped callable into a method bound to the class,
// not the instance.
type ClassMethod struct {
	Fn Object
}

// StaticMethod returns its wrapped callable unbound.
type StaticMethod struct {
	Fn Object
}

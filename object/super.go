package object

// Super represents a `super(StartCls, Self)` proxy. Attribute access on a
// Super walks the MRO of `Self`'s class starting strictly after StartCls.
// Methods returned by the lookup are bound to Self so calling looks like a
// regular bound-method call.
type Super struct {
	StartCls *Class
	Self     Object
}

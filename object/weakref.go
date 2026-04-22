package object

// PyWeakRef is a weak reference to an object.
// In this VM objects are never GC-collected during execution, so the
// reference stays alive for the life of the script. Callback is invoked
// when the referent is explicitly "killed" (not supported yet).
// TypeName overrides the default "ReferenceType" (e.g. "WeakMethod").
type PyWeakRef struct {
	Target   Object // nil → dead reference
	Callback Object // callable or nil
	TypeName string // if non-empty, overrides "ReferenceType"
}

// PyProxy is a transparent proxy to an object.
// All attribute access is forwarded to Target by the VM.
type PyProxy struct {
	Target   Object
	Callable bool // true for CallableProxyType
}

// PyFinalizer is a cleanup callback registered with weakref.finalize().
type PyFinalizer struct {
	Target Object
	Fn     Object
	Args   []Object
	Kwargs *Dict
	Alive  bool
	Atexit bool
}

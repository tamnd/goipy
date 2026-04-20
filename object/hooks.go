package object

// Hooks that route user-defined __dunder__ methods on Instance back into the
// interpreter. The vm package installs these at startup so the object package
// can stay free of VM-level dependencies while still honoring user methods
// from Hash/Eq/Repr/Str/Truthy.
var (
	InstanceReprHook   func(Object) (string, bool)
	InstanceStrHook    func(Object) (string, bool)
	InstanceTruthyHook func(Object) (bool, bool)
	InstanceHashHook   func(Object) (uint64, bool, error)
	InstanceEqHook     func(Object, Object) (bool, bool, error)
)

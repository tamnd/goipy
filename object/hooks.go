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
	// TypeNameHook lets the vm package register additional Go-level
	// types (e.g. *vm.Frame) so TypeName returns the user-facing
	// Python type name instead of the Go reflection name. Returns
	// "" if the type is unknown.
	TypeNameHook func(Object) string
)

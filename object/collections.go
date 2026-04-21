package object

// Deque is a double-ended queue backed by a slice. MaxLen < 0 means
// unbounded; when MaxLen >= 0 and Append would exceed it, the oldest
// element (the opposite end) is dropped.
type Deque struct {
	V      []Object
	MaxLen int
}

// Counter is a Dict-backed mapping from hashable to counts, exposing
// most_common, subtract, elements, total via getAttr in vm.
type Counter struct {
	D *Dict
}

// DefaultDict wraps a Dict with a default-factory callable. Access via
// getitem invokes the factory when the key is missing and stores the
// resulting value back into D.
type DefaultDict struct {
	D       *Dict
	Factory Object // callable or None
}

// OrderedDict wraps a Dict (which is itself already insertion-ordered)
// and exposes move_to_end / popitem(last=bool) through getAttr.
type OrderedDict struct {
	D *Dict
}

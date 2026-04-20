// Package object implements the goipy Python object model.
//
// Object is any Go value — we use interface{} and dispatch on concrete types
// instead of a vtable interface. The trade-off: adding a new operation means
// extending the relevant helper rather than implementing an interface method,
// which keeps concrete types cheap (a string wrapper needs no methods) and
// puts all numeric specialisation in one place.
package object

import (
	"fmt"
	"math/big"
)

// Object is any Python-level value visible to bytecode.
type Object = any

// Singletons.
var (
	None  = &NoneType{}
	True  = &Bool{V: true}
	False = &Bool{V: false}
)

type NoneType struct{}

func (*NoneType) String() string { return "None" }

type Bool struct{ V bool }

func BoolOf(b bool) *Bool {
	if b {
		return True
	}
	return False
}

// Int wraps arbitrary-precision integer.
type Int struct{ V *big.Int }

func NewInt(n int64) *Int { return &Int{V: big.NewInt(n)} }

// Float.
type Float struct{ V float64 }

// Complex is Python's complex number, a pair of IEEE-754 doubles.
type Complex struct{ Real, Imag float64 }

// Str holds a UTF-8 string. Indexing is by rune; we cache the rune slice on
// first indexed access.
type Str struct {
	V     string
	runes []rune // populated lazily
}

func (s *Str) Runes() []rune {
	if s.runes == nil && s.V != "" {
		s.runes = []rune(s.V)
	}
	return s.runes
}

// Bytes.
type Bytes struct{ V []byte }

// Bytearray is Python's mutable bytes.
type Bytearray struct{ V []byte }

// Memoryview is a shared view over a bytes/bytearray buffer. Start/stop
// describe a contiguous Python slice of the backing buffer (step=1 only).
// Readonly is true when the backing object is immutable (Bytes).
type Memoryview struct {
	Backing  Object // *Bytes or *Bytearray (pointer so mutations are visible)
	Start    int
	Stop     int
	Readonly bool
}

// Bytes returns a snapshot of the current view. Not an alias — callers that
// need to mutate should go through MV.Set/MV.Buf instead.
func (m *Memoryview) Bytes() []byte {
	raw := mvRaw(m)
	return append([]byte(nil), raw[m.Start:m.Stop]...)
}

// Buf returns the live slice underlying this view. Mutating it will affect
// the backing buffer. Nil if backing has shrunk below Stop.
func (m *Memoryview) Buf() []byte {
	raw := mvRaw(m)
	if m.Stop > len(raw) {
		return nil
	}
	return raw[m.Start:m.Stop]
}

func mvRaw(m *Memoryview) []byte {
	switch b := m.Backing.(type) {
	case *Bytes:
		return b.V
	case *Bytearray:
		return b.V
	}
	return nil
}

// Tuple is immutable.
type Tuple struct{ V []Object }

// List is mutable.
type List struct{ V []Object }

// Set backed by a slice + map of hash→indices for equality-based membership.
type Set struct {
	items []Object
	index map[uint64][]int
}

func NewSet() *Set { return &Set{index: map[uint64][]int{}} }

// Frozenset is an immutable, hashable set. Shares the Set layout but is a
// distinct type so TypeName, repr, and hashability differ.
type Frozenset struct {
	items []Object
	index map[uint64][]int
}

func NewFrozenset() *Frozenset { return &Frozenset{index: map[uint64][]int{}} }

// Dict: insertion-ordered.
type Dict struct {
	keys  []Object
	vals  []Object
	index map[string]int // fast path for str keys
	oHash map[uint64][]int
}

func NewDict() *Dict {
	return &Dict{index: map[string]int{}, oHash: map[uint64][]int{}}
}

// Len returns the number of entries.
func (d *Dict) Len() int { return len(d.keys) }

// Items returns key and value slices (caller must not mutate).
func (d *Dict) Items() ([]Object, []Object) { return d.keys, d.vals }

// Code mirrors CPython co_* fields we need.
type Code struct {
	ArgCount        int
	PosOnlyArgCount int
	KwOnlyArgCount  int
	Stacksize       int
	Flags           int
	Bytecode        []byte
	Consts          []Object
	Names           []string
	LocalsPlusNames []string
	LocalsPlusKinds []byte
	Filename        string
	Name            string
	QualName        string
	FirstLineNo     int
	LineTable       []byte
	ExceptionTable  []byte

	// Derived:
	NLocals  int // count of "fast" locals (bit CO_FAST_LOCAL = 0x20)
	NCells   int // cell slots (CO_FAST_CELL = 0x40)
	NFrees   int // free slots (CO_FAST_FREE = 0x80)
	CellVars []string
	FreeVars []string
}

const (
	FastLocal  = 0x20
	FastCell   = 0x40
	FastFree   = 0x80
	FastHidden = 0x10
	FastArg    = 0x01 // generic argument marker (includes posonly/kwonly)
)

// Function is a user-defined Python function.
type Function struct {
	Code     *Code
	Globals  *Dict
	Defaults *Tuple
	KwDefaults *Dict
	Closure  *Tuple // of *Cell
	Name     string
	QualName string
	Doc      Object
	Module   Object
	Annotations Object
	Dict     *Dict
}

// Cell is a shared storage slot used for closures.
type Cell struct {
	V   Object
	Set bool
}

// BuiltinFunc wraps a Go callable exposed as a Python builtin.
type BuiltinFunc struct {
	Name  string
	Call  func(interp any, args []Object, kwargs *Dict) (Object, error)
}

// BoundMethod binds self to a function/builtin.
type BoundMethod struct {
	Self Object
	Fn   Object
}

// Slice represents a slice object.
type Slice struct{ Start, Stop, Step Object }

// Module is a Python module.
type Module struct {
	Name string
	Dict *Dict
	// Path is the filesystem path of the .pyc the module was loaded
	// from, or "" for built-in modules. Used by importlib.reload.
	Path string
}

// Class is a minimal user-defined class object.
type Class struct {
	Name  string
	Bases []*Class
	Dict  *Dict
	// MRO computed lazily
	mro []*Class
}

// Instance of a user class.
type Instance struct {
	Class *Class
	Dict  *Dict
}

// Range represents the built-in range object.
type Range struct {
	Start, Stop, Step int64
}

// Iter wraps any stateful iterator.
type Iter struct {
	Next func() (Object, bool, error) // value, ok, error (ok=false = StopIteration)
}

// TypeName returns a short Python-style type name for o.
func TypeName(o Object) string {
	switch v := o.(type) {
	case nil:
		return "NoneType"
	case *NoneType:
		return "NoneType"
	case *Bool:
		return "bool"
	case *Int:
		return "int"
	case *Float:
		return "float"
	case *Complex:
		return "complex"
	case *Str:
		return "str"
	case *Bytes:
		return "bytes"
	case *Bytearray:
		return "bytearray"
	case *Memoryview:
		return "memoryview"
	case *Tuple:
		return "tuple"
	case *List:
		return "list"
	case *Dict:
		return "dict"
	case *Set:
		return "set"
	case *Frozenset:
		return "frozenset"
	case *Slice:
		return "slice"
	case *Range:
		return "range"
	case *Function, *BuiltinFunc, *BoundMethod:
		return "function"
	case *Code:
		return "code"
	case *Class:
		return v.Name
	case *Instance:
		return v.Class.Name
	case *Module:
		return "module"
	case *Iter:
		return "iterator"
	}
	return fmt.Sprintf("%T", o)
}

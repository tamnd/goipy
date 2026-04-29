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
	"sync"
	"sync/atomic"
)

// Object is any Python-level value visible to bytecode.
type Object = any

// Singletons.
var (
	None           = &NoneType{}
	True           = &Bool{V: true}
	False          = &Bool{V: false}
	Ellipsis       = &EllipsisType{}
	NotImplemented = &NotImplementedType{}
)

type NoneType struct{}

func (*NoneType) String() string { return "None" }

type EllipsisType struct{}

func (*EllipsisType) String() string { return "Ellipsis" }

type NotImplementedType struct{}

func (*NotImplementedType) String() string { return "NotImplemented" }

type Bool struct{ V bool }

func BoolOf(b bool) *Bool {
	if b {
		return True
	}
	return False
}

// Int holds a Python integer. V is a value-embedded big.Int so creating an
// Int is one heap allocation containing both the object header and the
// big.Int body, instead of three (Int header + big.Int header + nat slice).
// Callers passing *big.Int to the big API should use &ai.V.
type Int struct {
	V big.Int
	I int64
}

// Big returns a pointer to the embedded big.Int for APIs that need one.
func (n *Int) Big() *big.Int { return &n.V }

// IsInt64 reports whether n fits in an int64.
func (n *Int) IsInt64() bool { return n.V.IsInt64() }

// Int64 returns n as an int64. Caller must have verified IsInt64.
func (n *Int) Int64() int64 { return n.V.Int64() }

// smallIntCache covers the range Python's CPython caches internally. Hot
// loops like `for i in range(...)` and boolean int promotion hit this path
// heavily; pre-building the values makes those zero-allocation.
const (
	smallIntMin = -5
	smallIntMax = 256
)

var smallIntCache [smallIntMax - smallIntMin + 1]*Int

func init() {
	for i := range smallIntCache {
		n := int64(smallIntMin + i)
		x := &Int{I: n}
		x.V.SetInt64(n)
		smallIntCache[i] = x
	}
}

// IntFromInt64 returns a cached *Int for values in [-5, 256] and a freshly
// allocated one otherwise.
func IntFromInt64(n int64) *Int {
	if n >= smallIntMin && n <= smallIntMax {
		return smallIntCache[n-smallIntMin]
	}
	x := &Int{I: n}
	x.V.SetInt64(n)
	return x
}

// IntFromBig wraps a *big.Int into an *Int, hitting the small-int cache when
// the value fits so callers do not accidentally hold two copies of e.g. 0 or
// 1 that share a representation.
func IntFromBig(b *big.Int) *Int {
	if b.IsInt64() {
		n := b.Int64()
		if n >= smallIntMin && n <= smallIntMax {
			return smallIntCache[n-smallIntMin]
		}
		x := &Int{I: n}
		x.V.Set(b)
		return x
	}
	x := &Int{}
	x.V.Set(b)
	return x
}

func NewInt(n int64) *Int { return IntFromInt64(n) }

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
// views counts active memoryview objects backed by this bytearray; while
// views > 0, resize operations must raise BufferError. Atomic so that
// memoryview.release() and resize-attempt checks from different goroutines
// observe a coherent count.
type Bytearray struct {
	V     []byte
	views atomic.Int32
}

// AddView increments the active memoryview count.
func (b *Bytearray) AddView() { b.views.Add(1) }

// DropView decrements the active memoryview count if positive. Returns
// true when a view was actually dropped.
func (b *Bytearray) DropView() bool {
	for {
		n := b.views.Load()
		if n <= 0 {
			return false
		}
		if b.views.CompareAndSwap(n, n-1) {
			return true
		}
	}
}

// HasViews reports whether any memoryview is currently backed by this bytearray.
func (b *Bytearray) HasViews() bool { return b.views.Load() > 0 }

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

// List is mutable. Mu guards V for concurrent goroutine access in
// no-GIL threading. Direct V access is permitted on a freshly
// constructed list before it is published to other goroutines; once
// shared, callers must use Append/Extend/SetItem or take the lock
// themselves.
type List struct {
	Mu sync.Mutex
	V  []Object
}

// Append appends o to l under Mu. Used by LIST_APPEND opcode and
// other concurrent producers.
func (l *List) Append(o Object) {
	l.Mu.Lock()
	l.V = append(l.V, o)
	l.Mu.Unlock()
}

// Extend appends items to l under Mu.
func (l *List) Extend(items []Object) {
	l.Mu.Lock()
	l.V = append(l.V, items...)
	l.Mu.Unlock()
}

// Snapshot returns a copy of l.V taken under Mu so iterators see a
// stable view even if writers append concurrently.
func (l *List) Snapshot() []Object {
	l.Mu.Lock()
	out := make([]Object, len(l.V))
	copy(out, l.V)
	l.Mu.Unlock()
	return out
}

// Len returns len(l.V) under Mu.
func (l *List) Len() int {
	l.Mu.Lock()
	n := len(l.V)
	l.Mu.Unlock()
	return n
}

// Set backed by a slice + map of hash→indices for equality-based membership.
// mu protects items/index for concurrent goroutine access in no-GIL threading.
type Set struct {
	mu    sync.RWMutex
	items []Object
	index map[uint64][]int
}

func NewSet() *Set { return &Set{index: map[uint64][]int{}} }

// Frozenset is an immutable, hashable set. Shares the Set layout but is a
// distinct type so TypeName, repr, and hashability differ. mu guards the
// brief construction window before the value is exposed; once frozen, only
// reads occur and the lock is uncontended.
type Frozenset struct {
	mu    sync.RWMutex
	items []Object
	index map[uint64][]int
}

func NewFrozenset() *Frozenset { return &Frozenset{index: map[uint64][]int{}} }

// Dict: insertion-ordered. Deletes leave tombstones in keys/vals and are
// compacted when >50% of slots are dead; this keeps Delete O(1) amortized
// instead of O(n) per call.
type Dict struct {
	mu    sync.RWMutex   // protects all fields for concurrent goroutine access
	keys  []Object
	vals  []Object
	index map[string]int // fast path for str keys
	oHash map[uint64][]int
	dead  int // count of tombstones in keys/vals
}

// deletedEntry marks a tombstone slot in Dict.keys after a deletion.
type deletedEntry struct{}

var deletedKey Object = &deletedEntry{}

func NewDict() *Dict {
	return &Dict{}
}

// Len returns the number of live entries.
func (d *Dict) Len() int {
	d.mu.RLock()
	n := len(d.keys) - d.dead
	d.mu.RUnlock()
	return n
}

// Items returns a snapshot of key and value slices of live entries.
func (d *Dict) Items() ([]Object, []Object) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	ks := make([]Object, 0, len(d.keys)-d.dead)
	vs := make([]Object, 0, len(d.vals)-d.dead)
	for i, k := range d.keys {
		if k == deletedKey {
			continue
		}
		ks = append(ks, k)
		vs = append(vs, d.vals[i])
	}
	return ks, vs
}

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

	// FramePool holds a single reusable *vm.Frame (stored as any to avoid
	// a cycle between vm and object). The VM grabs it on call and returns
	// it on successful return; reentrant calls simply bypass the pool and
	// allocate a fresh frame. Replacing the old map[*Code]*Frame lookup
	// with a direct field removes a hashmap probe from every Python call.
	FramePool any
	// KwSlot caches kwname → Fast[] slot index for bindArgs. Built on
	// first keyword call; stable for the lifetime of the Code.
	KwSlot map[string]int
	// AttrCache is populated lazily by LOAD_ATTR dispatch. Indexed by the
	// bytecode IP of the LOAD_ATTR op (startIP). Zero entry means not yet
	// specialized; a non-zero Cls is a guard — if the instance's class
	// changed, the entry is invalid and the slow path runs + repopulates.
	AttrCache []AttrCacheEntry
	// Mu protects AttrCache and KwSlot for concurrent access from goroutine threads.
	Mu sync.Mutex
}

// AttrCacheEntry is a one-shot inline cache for LOAD_ATTR on instances.
type AttrCacheEntry struct {
	Cls  *Class
	Val  Object // for KindClassVal / KindClassMethod (raw Function to bind)
	Kind uint8  // 0 empty, 1 inst-dict hit, 2 class value (non-descriptor), 3 unbound method
}

const (
	AttrCacheEmpty       uint8 = 0
	AttrCacheInstDict    uint8 = 1
	AttrCacheClassValue  uint8 = 2
	AttrCacheClassMethod uint8 = 3
)

const (
	FastLocal  = 0x20
	FastCell   = 0x40
	FastFree   = 0x80
	FastHidden = 0x10
	FastArg    = 0x01 // generic argument marker (includes posonly/kwonly)
)

// Function is a user-defined Python function.
type Function struct {
	Code        *Code
	Globals     *Dict
	Defaults    *Tuple
	KwDefaults  *Dict
	Closure     *Tuple // of *Cell
	Name        string
	QualName    string
	Doc         Object
	Module      Object
	Annotations Object
	// Annotate is the PEP 649 lazy annotation function (Python 3.14
	// MAKE_FUNCTION + SET_FUNCTION_ATTRIBUTE flag 0x10). When
	// Annotations is unset, calling Annotate(1) materialises the
	// annotation dict; the result is cached into Annotations.
	Annotate Object
	Dict     *Dict
}

// Cell is a shared storage slot used for closures. Mu protects V/Set
// for closures captured by goroutine-backed threads (threading.Thread
// targets reading nonlocal state).
type Cell struct {
	Mu  sync.Mutex
	V   Object
	Set bool
}

// Load returns the cell's value and whether it is set, under Mu.
func (c *Cell) Load() (Object, bool) {
	c.Mu.Lock()
	v, ok := c.V, c.Set
	c.Mu.Unlock()
	return v, ok
}

// Store assigns v to the cell and marks it set, under Mu.
func (c *Cell) Store(v Object) {
	c.Mu.Lock()
	c.V = v
	c.Set = true
	c.Mu.Unlock()
}

// Unset clears the cell (DELETE_DEREF) under Mu.
func (c *Cell) Unset() {
	c.Mu.Lock()
	c.V = nil
	c.Set = false
	c.Mu.Unlock()
}

// BuiltinFunc wraps a Go callable exposed as a Python builtin.
type BuiltinFunc struct {
	Name string
	Call func(interp any, args []Object, kwargs *Dict) (Object, error)
	// Attrs holds extra attributes set on the builtin (e.g.
	// functools.lru_cache's wrapper exposes cache_info/cache_clear).
	Attrs *Dict
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

	// Slots is the list of own __slots__ names declared by this class
	// (excluding '__dict__' and '__weakref__'). Nil if the class did
	// not declare __slots__. NoDict, when true, suppresses __dict__
	// on instances and rejects assignment to non-slot names. The
	// default (false) preserves the legacy "every instance has a
	// dict" behaviour.
	Slots  []string
	NoDict bool

	// Mu protects MethodCache for concurrent access from goroutine threads.
	Mu sync.Mutex

	// MethodCache memoises classLookup(name) walks. A single global epoch
	// is bumped whenever any class is mutated (ClassEpoch()); stale entries
	// are ignored. Populated by the VM; safe to leave nil.
	MethodCache map[string]MethodCacheEntry

	// ABCCheck is set on abstract base classes (collections.abc). When
	// non-nil, isinstance() calls it before the normal MRO walk.
	ABCCheck func(o Object) bool

	// EnumData is non-nil for enum subclasses (Color, Number, etc.).
	// It holds the member list, value map, and iteration order.
	EnumData *EnumData
}

// MethodCacheEntry stores one cached classLookup result.
type MethodCacheEntry struct {
	Val   Object
	Found bool
	Epoch uint64
}

// classEpoch is bumped on any class dict mutation; MethodCache entries
// carry the epoch at which they were computed. Atomic so concurrent
// goroutines (no-GIL threads) can read/bump without tearing.
var classEpoch atomic.Uint64

func init() { classEpoch.Store(1) }

// ClassEpoch returns the current epoch.
func ClassEpoch() uint64 { return classEpoch.Load() }

// BumpClassEpoch invalidates every class method cache.
func BumpClassEpoch() { classEpoch.Add(1) }

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
	case *EllipsisType:
		return "ellipsis"
	case *NotImplementedType:
		return "NotImplementedType"
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
	case *PyArray:
		return "array"
	case *PyWeakRef:
		if v.TypeName != "" {
			return v.TypeName
		}
		return "ReferenceType"
	case *PyProxy:
		if v.Callable {
			return "CallableProxyType"
		}
		return "ProxyType"
	case *PyFinalizer:
		return "finalize"
	case *Deque:
		return "collections.deque"
	case *Counter:
		return "Counter"
	case *DefaultDict:
		return "defaultdict"
	case *OrderedDict:
		return "OrderedDict"
	case *Pattern:
		return "re.Pattern"
	case *Match:
		return "re.Match"
	case *StringIO:
		return "_io.StringIO"
	case *BytesIO:
		return "_io.BytesIO"
	case *TextStream:
		return "_io.TextIOWrapper"
	case *Traceback:
		return "traceback"
	case *TracebackFrame:
		return "frame"
	case *Hasher:
		return v.Name
	case *CSVReader:
		return "_csv.reader"
	case *CSVWriter:
		return "_csv.writer"
	case *CSVDictWriter:
		return "csv.DictWriter"
	case *CSVDictReader:
		return "csv.DictReader"
	case *CSVDialectObj:
		return "csv.Dialect"
	case *ConfigParserObj:
		return "configparser.ConfigParser"
	case *SectionProxyObj:
		return "configparser.SectionProxy"
	case *URLParseResult:
		if v.IsSplit {
			return "SplitResult"
		}
		return "ParseResult"
	case *UUID:
		return "UUID"
	case *SequenceMatcher:
		return "SequenceMatcher"
	case *File:
		return "_io.TextIOWrapper"
	case *Interpolation:
		return "Interpolation"
	case *Template:
		return "Template"
	}
	if TypeNameHook != nil {
		if name := TypeNameHook(o); name != "" {
			return name
		}
	}
	return fmt.Sprintf("%T", o)
}

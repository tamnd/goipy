package vm

import "github.com/tamnd/goipy/object"

const (
	inlineFastCap  = 8
	inlineStackCap = 16
)

// Frame is a single Python call frame.
type Frame struct {
	Code     *object.Code
	Globals  *object.Dict
	Builtins *object.Dict
	Locals   *object.Dict // for module-level / class-body frames (LOAD_NAME)
	Fast     []object.Object
	Cells    []*object.Cell // cell + free slots, length = NCells+NFrees
	Stack    []object.Object
	SP       int
	IP       int
	// LastIP is the start offset of the most recently executed opcode;
	// set by dispatch right before propagating an unhandled exception so
	// traceback nodes can resolve the right source line.
	LastIP int
	Back   *Frame
	// ExcInfo holds the most recent handled exception for re-raise.
	ExcInfo *object.Exception
	// Pending exception used by exception handler dispatch.
	curExc *object.Exception
	// Yielded carries the value produced by the most recent YIELD_VALUE;
	// consumed by the generator driver on resume.
	Yielded object.Object

	// YieldIP is the bytecode offset of the most recent YIELD_VALUE opcode;
	// used by throwGenerator to re-enter at the right exception-table entry.
	YieldIP int
	// PendingThrow, if non-nil, is an exception injected by throwGenerator.
	// Dispatch checks it at the start of each iteration and redirects to handleErr.
	PendingThrow error

	// LocalTrace is the per-frame trace function returned by the global
	// trace's 'call' event (or set explicitly by f_trace = fn). Fires for
	// 'line'/'return'/'exception' on this frame only.
	LocalTrace object.Object
	// LastLine is the source line of the last 'line' event fired; -1 means
	// no line event has fired yet (so the very first dispatched line emits
	// one).
	LastLine int

	// Inline buffers avoid a separate heap allocation for Fast/Stack on
	// small frames (most Python functions fit). Fast/Stack reference
	// these when the required size fits; otherwise they point at a
	// freshly allocated slice.
	fastInline  [inlineFastCap]object.Object
	stackInline [inlineStackCap]object.Object
}

// frameLocalsView returns a snapshot dict of f's local variables. For
// module/class-body frames f.Locals is the authoritative dict and is
// returned as-is; for function frames the view materialises Fast slots
// keyed by their LocalsPlusNames entries (PEP 667 calls this
// FrameLocalsProxy in 3.13+; goipy ships the simpler dict view for now).
func frameLocalsView(f *Frame) *object.Dict {
	if f == nil {
		return object.NewDict()
	}
	if f.Locals != nil && f.Code != nil && (f.Code.Flags&0x02) == 0 {
		// Bit 0x02 = CO_NEWLOCALS clear → module/class body. Return the
		// real locals dict so writes are observable.
		return f.Locals
	}
	d := object.NewDict()
	if f.Code == nil {
		return d
	}
	for k, n := range f.Code.LocalsPlusNames {
		if k >= len(f.Code.LocalsPlusKinds) {
			break
		}
		kind := f.Code.LocalsPlusKinds[k]
		if kind&object.FastHidden != 0 {
			continue
		}
		if k >= len(f.Fast) {
			break
		}
		v := f.Fast[k]
		if v == nil {
			continue
		}
		d.SetStr(n, v)
	}
	// Cell + free vars carry observable values too.
	for k, c := range f.Cells {
		if c == nil {
			continue
		}
		v, ok := c.Load()
		if !ok {
			continue
		}
		var name string
		if k < len(f.Code.CellVars) {
			name = f.Code.CellVars[k]
		} else if k-len(f.Code.CellVars) < len(f.Code.FreeVars) {
			name = f.Code.FreeVars[k-len(f.Code.CellVars)]
		}
		if name != "" {
			d.SetStr(name, v)
		}
	}
	return d
}

// NewFrame builds a fresh frame for code.
func NewFrame(code *object.Code, globals, builtins, locals *object.Dict) *Frame {
	f := &Frame{
		Code:     code,
		Globals:  globals,
		Builtins: builtins,
		Locals:   locals,
	}
	nFast := len(code.LocalsPlusNames)
	if nFast <= inlineFastCap {
		f.Fast = f.fastInline[:nFast]
	} else {
		f.Fast = make([]object.Object, nFast)
	}
	nStack := code.Stacksize + 8
	if nStack <= inlineStackCap {
		f.Stack = f.stackInline[:nStack]
	} else {
		f.Stack = make([]object.Object, nStack)
	}
	// Pre-allocate cell slots for MAKE_CELL.
	if code.NCells+code.NFrees > 0 {
		f.Cells = make([]*object.Cell, code.NCells+code.NFrees)
	}
	return f
}

func (f *Frame) push(o object.Object) {
	// Stack is preallocated to code.Stacksize+8 in NewFrame, but defensive
	// growth is kept for pathological compilers that under-report depth.
	if f.SP >= len(f.Stack) {
		f.Stack = append(f.Stack, o)
	} else {
		f.Stack[f.SP] = o
	}
	f.SP++
}

func (f *Frame) pop() object.Object {
	f.SP--
	return f.Stack[f.SP]
}

func (f *Frame) top() object.Object       { return f.Stack[f.SP-1] }
func (f *Frame) peek(n int) object.Object { return f.Stack[f.SP-1-n] }
func (f *Frame) setTop(o object.Object)   { f.Stack[f.SP-1] = o }
func (f *Frame) setPeek(n int, o object.Object) {
	f.Stack[f.SP-1-n] = o
}

package vm

import "github.com/tamnd/goipy/object"

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
	Back     *Frame
	// ExcInfo holds the most recent handled exception for re-raise.
	ExcInfo *object.Exception
	// Pending exception used by exception handler dispatch.
	curExc *object.Exception
}

// NewFrame builds a fresh frame for code.
func NewFrame(code *object.Code, globals, builtins, locals *object.Dict) *Frame {
	f := &Frame{
		Code:     code,
		Globals:  globals,
		Builtins: builtins,
		Locals:   locals,
		Fast:     make([]object.Object, len(code.LocalsPlusNames)),
		Stack:    make([]object.Object, code.Stacksize+8),
	}
	// Pre-allocate cell slots for MAKE_CELL.
	if code.NCells+code.NFrees > 0 {
		f.Cells = make([]*object.Cell, code.NCells+code.NFrees)
	}
	return f
}

func (f *Frame) push(o object.Object) {
	if f.SP >= len(f.Stack) {
		f.Stack = append(f.Stack, o)
	} else {
		f.Stack[f.SP] = o
	}
	f.SP++
}

func (f *Frame) pop() object.Object {
	f.SP--
	o := f.Stack[f.SP]
	f.Stack[f.SP] = nil
	return o
}

func (f *Frame) top() object.Object        { return f.Stack[f.SP-1] }
func (f *Frame) peek(n int) object.Object  { return f.Stack[f.SP-1-n] }
func (f *Frame) setTop(o object.Object)    { f.Stack[f.SP-1] = o }
func (f *Frame) setPeek(n int, o object.Object) {
	f.Stack[f.SP-1-n] = o
}

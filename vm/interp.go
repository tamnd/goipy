// Package vm implements the goipy bytecode interpreter.
package vm

import (
	"fmt"
	"io"
	"os"

	"github.com/tamnd/goipy/object"
)

// Interp is a Python-3.14 bytecode interpreter.
type Interp struct {
	Builtins *object.Dict
	Stdout   io.Writer
	Stderr   io.Writer

	// Cached exception classes (set by builtins).
	baseExc,
	exception,
	typeErr,
	valueErr,
	nameErr,
	keyErr,
	indexErr,
	attrErr,
	zeroDivErr,
	runtimeErr,
	stopIter,
	notImpl,
	overflowErr,
	lookupErr,
	arithErr,
	assertErr,
	importErr,
	recursionErr *object.Class

	MaxDepth  int
	callDepth int

	// modules is the minimal sys.modules equivalent used by IMPORT_NAME
	// until the full import machinery lands in PR #7.
	modules map[string]*object.Module
}

// New builds a fresh interpreter.
func New() *Interp {
	i := &Interp{
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		MaxDepth: 500,
	}
	i.initBuiltins()
	return i
}

// Run executes a code object as module-level code.
func (i *Interp) Run(code *object.Code) (object.Object, error) {
	globals := object.NewDict()
	globals.SetStr("__name__", &object.Str{V: "__main__"})
	globals.SetStr("__builtins__", i.Builtins)
	frame := NewFrame(code, globals, i.Builtins, globals)
	return i.runFrame(frame)
}

// RunPyc executes the top-level code of a decoded .pyc.
func (i *Interp) RunPyc(code *object.Code) error {
	_, err := i.Run(code)
	return err
}

func (i *Interp) runFrame(f *Frame) (object.Object, error) {
	if i.callDepth >= i.MaxDepth {
		return nil, object.Errorf(i.recursionErr, "maximum recursion depth exceeded")
	}
	i.callDepth++
	defer func() { i.callDepth-- }()
	return i.dispatch(f)
}

// errUnsupported returns a NotImplementedError for a given opcode number.
func (i *Interp) errUnsupported(op uint8, name string) error {
	return object.Errorf(i.notImpl, "opcode %s (%d) not implemented", name, op)
}

// Sprintf helper (avoids fmt import littering other files).
var _ = fmt.Sprintf

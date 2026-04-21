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

	// modules is the sys.modules equivalent used by IMPORT_NAME.
	modules map[string]*object.Module

	// SearchPath is the list of directories searched for `.pyc` files when
	// resolving `import <name>`. The main entry point typically seeds this
	// with the directory of the script being executed. Built-in modules
	// (e.g. asyncio) resolve before the search path is consulted.
	SearchPath []string

	// Argv is exposed as sys.argv. The main entry point seeds it with the
	// script path followed by remaining command-line arguments.
	Argv []string

	// curFrame is the innermost executing frame; sys.exc_info walks this
	// chain via Frame.Back to find the active exception.
	curFrame *Frame

	// framePool caches at most one reusable *Frame per *Code for
	// non-generator calls. Hot loops like `for _ in range(N): inc(x)`
	// hit the same Code repeatedly; pooling saves the per-call Fast +
	// Stack slice allocations.
	framePool map[*object.Code]*Frame
}

// New builds a fresh interpreter.
func New() *Interp {
	i := &Interp{
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		MaxDepth:  500,
		framePool: make(map[*object.Code]*Frame, 64),
	}
	i.initBuiltins()
	i.installDunderHooks()
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
	f.Back = i.curFrame
	i.curFrame = f
	defer func() {
		i.callDepth--
		i.curFrame = f.Back
	}()
	return i.dispatch(f)
}

// extendTraceback prepends a new traceback node for the frame f. The
// innermost frame is the head of the linked list; each frame the
// exception unwinds through adds a node.
func extendTraceback(e *object.Exception, f *Frame) {
	tb := &object.Traceback{
		Code:     f.Code,
		Lasti:    f.LastIP,
		Lineno:   f.Code.LineForOffset(f.LastIP),
		FuncName: f.Code.Name,
		Next:     e.Traceback,
	}
	e.Traceback = tb
}

// errUnsupported returns a NotImplementedError for a given opcode number.
func (i *Interp) errUnsupported(op uint8, name string) error {
	return object.Errorf(i.notImpl, "opcode %s (%d) not implemented", name, op)
}

// Sprintf helper (avoids fmt import littering other files).
var _ = fmt.Sprintf

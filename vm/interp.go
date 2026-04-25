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
	recursionErr,
	eofErr,
	osErr,
	fileNotFoundErr,
	fileExistsErr,
	stopAsyncIter,
	// New exception classes.
	floatErr,
	bufferErr,
	memoryErr,
	unboundLocalErr,
	moduleNotFoundErr,
	systemExit,
	keyboardInterrupt,
	generatorExit,
	connectionErr,
	syntaxErr,
	systemErr,
	unicodeErr,
	unicodeDecodeErr,
	unicodeEncodeErr,
	warningClass,
	baseExcGroup,
	reErr *object.Class

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

	// logStates holds per-module logState for logging/logging.config sharing.
	logStates map[string]*logState

	// ctxStack is the per-goroutine context variable stack (contextvars module).
	// Each entry is an active Context; the last element is the current context.
	ctxStack []*cvContext
}

// New builds a fresh interpreter.
func New() *Interp {
	i := &Interp{
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		MaxDepth: 500,
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

// threadCopy returns a shallow copy of i suitable for running in a new goroutine.
// It shares all read-only state (Builtins, exception classes, Stdout/Stderr) but
// gets its own module map so concurrent imports don't race on map writes.
func (i *Interp) threadCopy() *Interp {
	tc := *i
	tc.callDepth = 0
	tc.curFrame = nil
	tc.modules = make(map[string]*object.Module, len(i.modules))
	for k, v := range i.modules {
		tc.modules[k] = v
	}
	if i.logStates != nil {
		tc.logStates = make(map[string]*logState, len(i.logStates))
		for k, v := range i.logStates {
			tc.logStates[k] = v
		}
	}
	// Each goroutine gets its own copy of the context stack so threads
	// can't see each other's ContextVar mutations.
	if len(i.ctxStack) > 0 {
		tc.ctxStack = make([]*cvContext, len(i.ctxStack))
		for k, ctx := range i.ctxStack {
			tc.ctxStack[k] = ctx.clone()
		}
	} else {
		tc.ctxStack = nil
	}
	return &tc
}

func (i *Interp) runFrame(f *Frame) (object.Object, error) {
	if i.callDepth >= i.MaxDepth {
		return nil, object.Errorf(i.recursionErr, "maximum recursion depth exceeded")
	}
	i.callDepth++
	f.Back = i.curFrame
	i.curFrame = f
	r, err := i.dispatch(f)
	i.callDepth--
	i.curFrame = f.Back
	return r, err
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

// SystemExitCode returns the integer exit code if err is a SystemExit
// exception, and ok=true. Otherwise ok=false.
func (i *Interp) SystemExitCode(err error) (code int, ok bool) {
	e, isExc := err.(*object.Exception)
	if !isExc || e.Class != i.systemExit {
		return 0, false
	}
	if e.Args != nil && len(e.Args.V) == 1 {
		if n, okN := toInt64(e.Args.V[0]); okN {
			return int(n), true
		}
		// Non-integer arg (e.g. a string) → print to stderr, exit 1.
		return 1, true
	}
	return 0, true
}

// Sprintf helper (avoids fmt import littering other files).
var _ = fmt.Sprintf

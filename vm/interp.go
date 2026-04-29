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
	excGroup,
	reErr,
	jsonDecodeErr *object.Class

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

	// warnState holds per-Interp warnings module state.
	warnState *warnState

	// ctxStack is the per-goroutine context variable stack (contextvars module).
	// Each entry is an active Context; the last element is the current context.
	ctxStack []*cvContext

	// auditHooks holds callables registered via sys.addaudithook.
	// Hooks are permanent and cannot be removed.
	auditHooks []object.Object

	// traceFn is the global trace function set by sys.settrace; nil/None
	// disables tracing. Called with (frame, 'call', None) on frame entry;
	// the return value becomes the per-frame local trace.
	traceFn object.Object
	// profileFn is the global profile function set by sys.setprofile; like
	// traceFn but only fires call/return — never line.
	profileFn object.Object
	// inTrace guards against re-entry: while a trace/profile callback is
	// running, further events are suppressed so the tracer's own dispatch
	// doesn't infinite-loop.
	inTrace bool
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

// CallObject invokes callable with positional and keyword args.
// It is the public counterpart of the internal callObject method,
// intended for use by native extension modules outside this package.
func (i *Interp) CallObject(callable object.Object, args []object.Object, kwargs *object.Dict) (object.Object, error) {
	return i.callObject(callable, args, kwargs)
}

// RegisterModule pre-registers a native module so that
// `import <name>` resolves to mod without needing a .pyc file.
// This is the public replacement for the old NativeModules field.
func (i *Interp) RegisterModule(name string, mod *object.Module) {
	if i.modules == nil {
		i.modules = make(map[string]*object.Module)
	}
	i.modules[name] = mod
}

// NativeModules is a convenience setter: it registers all entries in the
// map as native modules. Keys are module names, values are factory functions
// called immediately with i to build each module.
func (i *Interp) SetNativeModules(factories map[string]func(*Interp) *object.Module) {
	for name, fn := range factories {
		i.RegisterModule(name, fn(i))
	}
}

// Run executes a code object as module-level code.
func (i *Interp) Run(code *object.Code) (object.Object, error) {
	globals := object.NewDict()
	globals.SetStr("__name__", &object.Str{V: "__main__"})
	globals.SetStr("__doc__", object.None)
	globals.SetStr("__annotations__", object.NewDict())
	globals.SetStr("__builtins__", i.Builtins)
	globals.SetStr("__spec__", object.None)
	globals.SetStr("__loader__", object.None)
	globals.SetStr("__package__", object.None)
	// Register the live namespace as __main__ so `import __main__` works.
	if i.modules == nil {
		i.modules = map[string]*object.Module{}
	}
	i.modules["__main__"] = &object.Module{Name: "__main__", Dict: globals}
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
	// Snapshot audit hooks so goroutine inherits existing hooks but new hooks
	// added in the goroutine don't leak back to the parent.
	if len(i.auditHooks) > 0 {
		tc.auditHooks = make([]object.Object, len(i.auditHooks))
		copy(tc.auditHooks, i.auditHooks)
	}
	return &tc
}

// fireAudit calls every registered audit hook with (event, args-tuple).
// RuntimeError raised by a hook is suppressed; any other exception propagates
// immediately, aborting remaining hooks.
func (i *Interp) fireAudit(event string, args []object.Object) error {
	if len(i.auditHooks) == 0 {
		return nil
	}
	argTuple := &object.Tuple{V: args}
	eventStr := &object.Str{V: event}
	hookArgs := []object.Object{eventStr, argTuple}
	for _, hook := range i.auditHooks {
		_, err := i.callObject(hook, hookArgs, nil)
		if err != nil {
			if exc, ok := err.(*object.Exception); ok {
				if object.IsSubclass(exc.Class, i.runtimeErr) {
					continue
				}
			}
			return err
		}
	}
	return nil
}

func (i *Interp) runFrame(f *Frame) (object.Object, error) {
	if i.callDepth >= i.MaxDepth {
		return nil, object.Errorf(i.recursionErr, "maximum recursion depth exceeded")
	}
	i.callDepth++
	f.Back = i.curFrame
	i.curFrame = f
	i.fireCallEvent(f)
	r, err := i.dispatch(f)
	if err == nil {
		i.fireReturnEvent(f, r)
	} else if err != errYielded {
		i.fireExceptionEvent(f, err)
	}
	i.callDepth--
	i.curFrame = f.Back
	return r, err
}

// fireCallEvent invokes the global trace/profile function with a 'call'
// event when a frame begins execution. The result of traceFn(...) is
// stashed on the frame as the per-frame local trace function used for
// 'line'/'return'/'exception' events.
func (i *Interp) fireCallEvent(f *Frame) {
	if i.inTrace {
		return
	}
	// Seed LastLine to the def line so the first body line transition
	// fires a 'line' event, matching CPython 3.11+ which never reports
	// the RESUME opcode's def-line as a separate event.
	if f.Code != nil {
		f.LastLine = f.Code.FirstLineNo
	} else {
		f.LastLine = -1
	}
	if i.traceFn != nil {
		i.inTrace = true
		r, err := i.callObject(i.traceFn, []object.Object{f, &object.Str{V: "call"}, object.None}, nil)
		i.inTrace = false
		if err == nil && r != nil {
			if _, isNone := r.(*object.NoneType); !isNone {
				f.LocalTrace = r
			}
		}
	}
	if i.profileFn != nil {
		i.inTrace = true
		i.callObject(i.profileFn, []object.Object{f, &object.Str{V: "call"}, object.None}, nil)
		i.inTrace = false
	}
}

// fireReturnEvent dispatches the 'return' event on the per-frame trace
// and the global profile function, in that order — matching CPython.
func (i *Interp) fireReturnEvent(f *Frame, retval object.Object) {
	if i.inTrace {
		return
	}
	arg := retval
	if arg == nil {
		arg = object.None
	}
	if f.LocalTrace != nil {
		i.inTrace = true
		i.callObject(f.LocalTrace, []object.Object{f, &object.Str{V: "return"}, arg}, nil)
		i.inTrace = false
	}
	if i.profileFn != nil {
		i.inTrace = true
		i.callObject(i.profileFn, []object.Object{f, &object.Str{V: "return"}, arg}, nil)
		i.inTrace = false
	}
}

// fireExceptionEvent dispatches the 'exception' event with a
// (type, value, None) tuple matching CPython's signature.
func (i *Interp) fireExceptionEvent(f *Frame, err error) {
	if i.inTrace || f.LocalTrace == nil {
		return
	}
	exc, ok := err.(*object.Exception)
	if !ok {
		return
	}
	arg := &object.Tuple{V: []object.Object{exc.Class, exc, object.None}}
	i.inTrace = true
	i.callObject(f.LocalTrace, []object.Object{f, &object.Str{V: "exception"}, arg}, nil)
	i.inTrace = false
}

// fireLineEvent fires when execution crosses a source-line boundary.
// Called from the dispatch loop only when f.LocalTrace != nil and the
// line actually changes.
func (i *Interp) fireLineEvent(f *Frame) {
	if i.inTrace || f.LocalTrace == nil {
		return
	}
	i.inTrace = true
	r, err := i.callObject(f.LocalTrace, []object.Object{f, &object.Str{V: "line"}, object.None}, nil)
	i.inTrace = false
	// Per CPython: if local trace returns None, disable further line events
	// on this frame; if it returns a different callable, replace the local
	// trace.
	if err == nil && r != nil {
		if _, isNone := r.(*object.NoneType); isNone {
			f.LocalTrace = nil
		} else {
			f.LocalTrace = r
		}
	}
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

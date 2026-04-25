package vm

import (
	"fmt"
	osExec "os/exec"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/tamnd/goipy/marshal"
	"github.com/tamnd/goipy/object"
)

// ─── Sub-interpreter state ───────────────────────────────────────────────────

type ciInterpState struct {
	id      int64
	whence  string
	main    *object.Dict // __main__ namespace used by exec()
	closed  atomic.Bool
	running atomic.Int32
}

var (
	ciRegistry    sync.Map    // int64 → *ciInterpState
	ciIDCounter   atomic.Int64
	ciMainID      int64 = -1 // set on first buildConcurrentInterpreters call
	ciMainIDOnce  sync.Once
)

// ciQueueState backs a cross-interpreter Queue.
type ciQueueState struct {
	id      int64
	ch      chan object.Object
	maxsize int
}

var (
	ciQueueRegistry  sync.Map // int64 → *ciQueueState
	ciQueueIDCounter atomic.Int64
)

// buildConcurrentInterpreters constructs the concurrent.interpreters module.
func (i *Interp) buildConcurrentInterpreters() *object.Module {
	m := &object.Module{Name: "concurrent.interpreters", Dict: object.NewDict()}

	// Register the main interpreter on first module load.
	ciMainIDOnce.Do(func() {
		ciMainID = ciIDCounter.Add(1) - 1 // start at 0
		main := object.NewDict()
		main.SetStr("__builtins__", i.Builtins)
		main.SetStr("__name__", &object.Str{V: "__main__"})
		st := &ciInterpState{id: ciMainID, whence: "runtime init", main: main}
		ciRegistry.Store(ciMainID, st)
	})

	// ─── Exception classes ──────────────────────────────────────────────────
	interpErr  := &object.Class{Name: "InterpreterError", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	notFoundErr := &object.Class{Name: "InterpreterNotFoundError", Bases: []*object.Class{interpErr}, Dict: object.NewDict()}
	execFailed := &object.Class{Name: "ExecutionFailed", Bases: []*object.Class{interpErr}, Dict: object.NewDict()}
	notShareableErr := &object.Class{Name: "NotShareableError", Bases: []*object.Class{i.typeErr}, Dict: object.NewDict()}
	queueEmptyErr := &object.Class{Name: "QueueEmpty", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	queueFullErr  := &object.Class{Name: "QueueFull", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}

	m.Dict.SetStr("InterpreterError", interpErr)
	m.Dict.SetStr("InterpreterNotFoundError", notFoundErr)
	m.Dict.SetStr("ExecutionFailed", execFailed)
	m.Dict.SetStr("NotShareableError", notShareableErr)
	m.Dict.SetStr("QueueEmpty", queueEmptyErr)
	m.Dict.SetStr("QueueFull", queueFullErr)

	// ─── create() ──────────────────────────────────────────────────────────
	m.Dict.SetStr("create", &object.BuiltinFunc{Name: "create",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			id := ciIDCounter.Add(1) - 1
			main := object.NewDict()
			main.SetStr("__builtins__", i.Builtins)
			main.SetStr("__name__", &object.Str{V: "__main__"})
			st := &ciInterpState{id: id, whence: "interpreter.create()"}
			st.main = main
			ciRegistry.Store(id, st)
			return i.makeCIInterp(st, notFoundErr, execFailed), nil
		}})

	// ─── list_all() ────────────────────────────────────────────────────────
	m.Dict.SetStr("list_all", &object.BuiltinFunc{Name: "list_all",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			var out []object.Object
			ciRegistry.Range(func(k, v any) bool {
				st := v.(*ciInterpState)
				if !st.closed.Load() {
					out = append(out, i.makeCIInterp(st, notFoundErr, execFailed))
				}
				return true
			})
			return &object.List{V: out}, nil
		}})

	// ─── get_main() ────────────────────────────────────────────────────────
	m.Dict.SetStr("get_main", &object.BuiltinFunc{Name: "get_main",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			v, ok := ciRegistry.Load(ciMainID)
			if !ok {
				return nil, object.Errorf(notFoundErr, "main interpreter not found")
			}
			return i.makeCIInterp(v.(*ciInterpState), notFoundErr, execFailed), nil
		}})

	// ─── get_current() ─────────────────────────────────────────────────────
	// In goipy all user code runs in a single interpreter context.
	m.Dict.SetStr("get_current", &object.BuiltinFunc{Name: "get_current",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			v, ok := ciRegistry.Load(ciMainID)
			if !ok {
				return nil, object.Errorf(notFoundErr, "current interpreter not found")
			}
			return i.makeCIInterp(v.(*ciInterpState), notFoundErr, execFailed), nil
		}})

	// ─── create_queue(maxsize=0) ────────────────────────────────────────────
	m.Dict.SetStr("create_queue", &object.BuiltinFunc{Name: "create_queue",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			maxsize := 0
			if len(a) > 0 {
				if n, ok := toInt64(a[0]); ok {
					maxsize = int(n)
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("maxsize"); ok {
					if n, ok2 := toInt64(v); ok2 {
						maxsize = int(n)
					}
				}
			}
			return i.makeCIQueue(maxsize, queueEmptyErr, queueFullErr), nil
		}})

	return m
}

// makeCIInterp builds an Interpreter instance from a ciInterpState.
func (i *Interp) makeCIInterp(st *ciInterpState, notFoundErr, execFailed *object.Class) *object.Instance {
	cls := &object.Class{Name: "Interpreter", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	inst.Dict.SetStr("id", object.NewInt(st.id))
	inst.Dict.SetStr("whence", &object.Str{V: st.whence})

	cls.Dict.SetStr("is_running", &object.BuiltinFunc{Name: "is_running",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(st.running.Load() > 0), nil
		}})

	// close()
	cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			st.closed.Store(true)
			ciRegistry.Delete(st.id)
			return object.None, nil
		}})

	// prepare_main(ns=None, **kwargs)
	cls.Dict.SetStr("prepare_main", &object.BuiltinFunc{Name: "prepare_main",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if st.closed.Load() {
				return nil, object.Errorf(notFoundErr, "interpreter already closed")
			}
			// Merge ns dict if provided.
			if len(a) > 0 {
				if d, ok := a[0].(*object.Dict); ok {
					ks, vs := d.Items()
					for idx, kobj := range ks {
						if s, ok2 := kobj.(*object.Str); ok2 {
							st.main.SetStr(s.V, vs[idx])
						}
					}
				}
			}
			// Merge kwargs.
			if kw != nil {
				ks, vs := kw.Items()
				for idx, kobj := range ks {
					if s, ok2 := kobj.(*object.Str); ok2 {
						st.main.SetStr(s.V, vs[idx])
					}
				}
			}
			return object.None, nil
		}})

	// exec(code, /, dedent=True)
	cls.Dict.SetStr("exec", &object.BuiltinFunc{Name: "exec",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if st.closed.Load() {
				return nil, object.Errorf(notFoundErr, "interpreter already closed")
			}
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "exec() requires code argument")
			}
			src, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "exec() argument must be str")
			}
			dedent := true
			if len(a) > 1 {
				if b, ok2 := a[1].(*object.Bool); ok2 {
					dedent = b.V
				}
			}
			if kw != nil {
				if v, ok2 := kw.GetStr("dedent"); ok2 {
					if b, ok3 := v.(*object.Bool); ok3 {
						dedent = b.V
					}
				}
			}
			code, err := ciCompileSource(src.V, dedent)
			if err != nil {
				return nil, object.Errorf(execFailed, "compile error: %s", err)
			}
			st.running.Add(1)
			wi := i.threadCopy()
			wi.modules = i.modules // share module cache
			frame := NewFrame(code, st.main, wi.Builtins, st.main)
			_, runErr := wi.runFrame(frame)
			st.running.Add(-1)
			if runErr != nil {
				return nil, object.Errorf(execFailed, "%s", runErr)
			}
			return object.None, nil
		}})

	// call(callable, /, *args, **kwargs)
	cls.Dict.SetStr("call", &object.BuiltinFunc{Name: "call",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if st.closed.Load() {
				return nil, object.Errorf(notFoundErr, "interpreter already closed")
			}
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "call() requires a callable")
			}
			fn := a[0]
			fnArgs := append([]object.Object(nil), a[1:]...)
			wi := i.threadCopy()
			st.running.Add(1)
			val, runErr := wi.callObject(fn, fnArgs, kw)
			st.running.Add(-1)
			if runErr != nil {
				return nil, object.Errorf(execFailed, "%s", runErr)
			}
			return val, nil
		}})

	// call_in_thread(callable, /, *args, **kwargs) → thread-like object
	cls.Dict.SetStr("call_in_thread", &object.BuiltinFunc{Name: "call_in_thread",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if st.closed.Load() {
				return nil, object.Errorf(notFoundErr, "interpreter already closed")
			}
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "call_in_thread() requires a callable")
			}
			fn := a[0]
			fnArgs := append([]object.Object(nil), a[1:]...)
			return i.makeCIThread(st, fn, fnArgs, kw, execFailed), nil
		}})

	return inst
}

// makeCIThread starts a goroutine and returns a Thread-like instance with join/is_alive.
func (i *Interp) makeCIThread(st *ciInterpState, fn object.Object, args []object.Object, kw *object.Dict, execFailed *object.Class) *object.Instance {
	cls := &object.Class{Name: "Thread", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	var wg sync.WaitGroup
	var alive int32 = 1

	wg.Add(1)
	wi := i.threadCopy()
	go func() {
		defer func() {
			atomic.StoreInt32(&alive, 0)
			st.running.Add(-1)
			wg.Done()
		}()
		st.running.Add(1)
		wi.callObject(fn, args, kw) //nolint
	}()

	cls.Dict.SetStr("join", &object.BuiltinFunc{Name: "join",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			wg.Wait()
			return object.None, nil
		}})

	cls.Dict.SetStr("is_alive", &object.BuiltinFunc{Name: "is_alive",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(atomic.LoadInt32(&alive) == 1), nil
		}})

	return inst
}

// makeCIQueue builds a cross-interpreter Queue backed by a Go channel.
func (i *Interp) makeCIQueue(maxsize int, queueEmptyErr, queueFullErr *object.Class) *object.Instance {
	cap := 1000
	if maxsize > 0 {
		cap = maxsize
	}
	qst := &ciQueueState{
		id:      ciQueueIDCounter.Add(1) - 1,
		ch:      make(chan object.Object, cap),
		maxsize: maxsize,
	}
	ciQueueRegistry.Store(qst.id, qst)

	cls := &object.Class{Name: "Queue", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("id", object.NewInt(qst.id))

	cls.Dict.SetStr("put", &object.BuiltinFunc{Name: "put",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "put() requires item argument")
			}
			item := a[0]
			block := true
			if len(a) > 1 {
				if b, ok := a[1].(*object.Bool); ok {
					block = b.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("block"); ok {
					if b, ok2 := v.(*object.Bool); ok2 {
						block = b.V
					}
				}
			}
			if block {
				qst.ch <- item
			} else {
				select {
				case qst.ch <- item:
				default:
					return nil, object.Errorf(queueFullErr, "queue is full")
				}
			}
			return object.None, nil
		}})

	cls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			block := true
			if len(a) > 0 {
				if b, ok := a[0].(*object.Bool); ok {
					block = b.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("block"); ok {
					if b, ok2 := v.(*object.Bool); ok2 {
						block = b.V
					}
				}
			}
			if block {
				return <-qst.ch, nil
			}
			select {
			case v := <-qst.ch:
				return v, nil
			default:
				return nil, object.Errorf(queueEmptyErr, "queue is empty")
			}
		}})

	cls.Dict.SetStr("put_nowait", &object.BuiltinFunc{Name: "put_nowait",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "put_nowait() requires item argument")
			}
			select {
			case qst.ch <- a[0]:
			default:
				return nil, object.Errorf(queueFullErr, "queue is full")
			}
			return object.None, nil
		}})

	cls.Dict.SetStr("get_nowait", &object.BuiltinFunc{Name: "get_nowait",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			select {
			case v := <-qst.ch:
				return v, nil
			default:
				return nil, object.Errorf(queueEmptyErr, "queue is empty")
			}
		}})

	cls.Dict.SetStr("empty", &object.BuiltinFunc{Name: "empty",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(len(qst.ch) == 0), nil
		}})

	cls.Dict.SetStr("full", &object.BuiltinFunc{Name: "full",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if qst.maxsize <= 0 {
				return object.False, nil
			}
			return object.BoolOf(len(qst.ch) >= qst.maxsize), nil
		}})

	cls.Dict.SetStr("qsize", &object.BuiltinFunc{Name: "qsize",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(len(qst.ch))), nil
		}})

	return inst
}

// ciCompileSource compiles a Python source string to a code object by shelling
// out to Python (which is required for the exec() method).
func ciCompileSource(src string, dedent bool) (*object.Code, error) {
	python := ciPythonCmd()
	var script string
	if dedent {
		script = fmt.Sprintf(`import marshal, sys, textwrap
code = compile(textwrap.dedent(%s), '<string>', 'exec')
sys.stdout.buffer.write(marshal.dumps(code))
`, strconv.Quote(src))
	} else {
		script = fmt.Sprintf(`import marshal, sys
code = compile(%s, '<string>', 'exec')
sys.stdout.buffer.write(marshal.dumps(code))
`, strconv.Quote(src))
	}
	cmd := osExec.Command(python, "-c", script)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*osExec.ExitError); ok {
			return nil, fmt.Errorf("%s", string(exitErr.Stderr))
		}
		return nil, err
	}
	obj, err := marshal.Unmarshal(out)
	if err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	code, ok := obj.(*object.Code)
	if !ok {
		return nil, fmt.Errorf("expected code object, got %T", obj)
	}
	return code, nil
}

// ciPythonCmd returns the first available Python 3 binary name in PATH.
func ciPythonCmd() string {
	for _, name := range []string{"python3.14", "python3", "python"} {
		if _, err := osExec.LookPath(name); err == nil {
			return name
		}
	}
	return "python3"
}

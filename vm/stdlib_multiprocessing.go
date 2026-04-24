package vm

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tamnd/goipy/object"
)

var (
	mpPIDCounter       int64    // monotonically increasing process ID
	mpAlive            int64    // count of live non-main processes
	mpCurrentProcesses sync.Map // int64(goroutineID) → *object.Instance
)

// mpArgs strips the leading self argument from a BuiltinFunc args slice.
// When methods are stored in a Class.Dict and looked up via LOAD_ATTR on an
// Instance, goipy prepends self to args via BoundMethod dispatch. All class
// methods defined in multiprocessing must call mpArgs to get the real args.
func mpArgs(a []object.Object) []object.Object {
	if len(a) > 0 {
		if _, ok := a[0].(*object.Instance); ok {
			return a[1:]
		}
	}
	return a
}

// buildMultiprocessing constructs the multiprocessing module backed by real
// goroutines. "Processes" are goroutines sharing the same OS process.
func (i *Interp) buildMultiprocessing() *object.Module {
	m := &object.Module{Name: "multiprocessing", Dict: object.NewDict()}

	mainGID := goroutineID()
	mainProc := i.makeMPProcess("MainProcess", mainGID, nil, nil, nil, false)

	// --- sync primitives (delegate to threading) ---
	threading := i.buildThreading()
	for _, name := range []string{"Lock", "RLock", "Event", "Condition", "Semaphore", "BoundedSemaphore", "Barrier"} {
		if v, ok := threading.Dict.GetStr(name); ok {
			m.Dict.SetStr(name, v)
		}
	}

	// --- Process ---
	m.Dict.SetStr("Process", &object.BuiltinFunc{Name: "Process", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		var target object.Object
		var pargs *object.Tuple
		var pkwargs *object.Dict
		name := ""
		daemon := false
		if kw != nil {
			if v, ok := kw.GetStr("target"); ok {
				target = v
			}
			if v, ok := kw.GetStr("args"); ok {
				switch t := v.(type) {
				case *object.Tuple:
					pargs = t
				case *object.List:
					pargs = &object.Tuple{V: t.V}
				}
			}
			if v, ok := kw.GetStr("kwargs"); ok {
				if d, ok2 := v.(*object.Dict); ok2 {
					pkwargs = d
				}
			}
			if v, ok := kw.GetStr("name"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					name = s.V
				}
			}
			if v, ok := kw.GetStr("daemon"); ok {
				if b, ok2 := v.(*object.Bool); ok2 {
					daemon = b.V
				}
			}
		}
		n := atomic.AddInt64(&mpPIDCounter, 1)
		if name == "" {
			name = fmt.Sprintf("Process-%d", n)
		}
		return i.makeMPProcess(name, 0, target, pargs, pkwargs, daemon), nil
	}})

	// --- Queue ---
	m.Dict.SetStr("Queue", &object.BuiltinFunc{Name: "Queue", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		maxsize := 0
		if len(args) > 0 {
			if n, ok := args[0].(*object.Int); ok && n.IsInt64() {
				maxsize = int(n.Int64())
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("maxsize"); ok {
				if n, ok2 := v.(*object.Int); ok2 && n.IsInt64() {
					maxsize = int(n.Int64())
				}
			}
		}
		return i.makeMPQueue(maxsize), nil
	}})

	// --- JoinableQueue (alias for Queue) ---
	m.Dict.SetStr("JoinableQueue", &object.BuiltinFunc{Name: "JoinableQueue", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		maxsize := 0
		if len(args) > 0 {
			if n, ok := args[0].(*object.Int); ok && n.IsInt64() {
				maxsize = int(n.Int64())
			}
		}
		return i.makeMPQueue(maxsize), nil
	}})

	// --- Pipe ---
	m.Dict.SetStr("Pipe", &object.BuiltinFunc{Name: "Pipe", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		duplex := true
		if len(args) > 0 {
			if b, ok := args[0].(*object.Bool); ok {
				duplex = b.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("duplex"); ok {
				if b, ok2 := v.(*object.Bool); ok2 {
					duplex = b.V
				}
			}
		}
		return i.makeMPPipe(duplex)
	}})

	// --- Pool ---
	m.Dict.SetStr("Pool", &object.BuiltinFunc{Name: "Pool", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		nproc := runtime.NumCPU()
		if len(args) > 0 {
			if n, ok := args[0].(*object.Int); ok && n.IsInt64() && n.Int64() > 0 {
				nproc = int(n.Int64())
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("processes"); ok {
				if n, ok2 := v.(*object.Int); ok2 && n.IsInt64() && n.Int64() > 0 {
					nproc = int(n.Int64())
				}
			}
		}
		var initializer object.Object
		if kw != nil {
			if v, ok := kw.GetStr("initializer"); ok {
				initializer = v
			}
		}
		initArgs := &object.Tuple{}
		if kw != nil {
			if v, ok := kw.GetStr("initargs"); ok {
				switch t := v.(type) {
				case *object.Tuple:
					initArgs = t
				case *object.List:
					initArgs = &object.Tuple{V: t.V}
				}
			}
		}
		return i.makeMPPool(nproc, initializer, initArgs), nil
	}})

	// --- Value ---
	m.Dict.SetStr("Value", &object.BuiltinFunc{Name: "Value", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		var val object.Object = object.None
		if len(args) >= 2 {
			val = args[1]
		}
		return i.makeMPValue(val), nil
	}})

	// --- Array ---
	m.Dict.SetStr("Array", &object.BuiltinFunc{Name: "Array", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		var init []object.Object
		if len(args) >= 2 {
			switch t := args[1].(type) {
			case *object.List:
				init = t.V
			case *object.Tuple:
				init = t.V
			}
		}
		return i.makeMPArray(init), nil
	}})

	// --- Manager ---
	m.Dict.SetStr("Manager", &object.BuiltinFunc{Name: "Manager", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return i.makeMPManager(), nil
	}})

	// --- utility functions ---

	m.Dict.SetStr("cpu_count", &object.BuiltinFunc{Name: "cpu_count", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(runtime.NumCPU())), nil
	}})

	m.Dict.SetStr("current_process", &object.BuiltinFunc{Name: "current_process", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		gid := goroutineID()
		if v, ok := mpCurrentProcesses.Load(gid); ok {
			return v.(*object.Instance), nil
		}
		return mainProc, nil
	}})

	m.Dict.SetStr("parent_process", &object.BuiltinFunc{Name: "parent_process", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		gid := goroutineID()
		if v, ok := mpCurrentProcesses.Load(gid); ok {
			inst := v.(*object.Instance)
			if pv, ok2 := inst.Dict.GetStr("_parent"); ok2 {
				return pv, nil
			}
		}
		return object.None, nil
	}})

	m.Dict.SetStr("active_children", &object.BuiltinFunc{Name: "active_children", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var out []object.Object
		mpCurrentProcesses.Range(func(_, v any) bool {
			inst := v.(*object.Instance)
			if av, ok := inst.Dict.GetStr("_alive"); ok {
				if n, ok2 := av.(*object.Int); ok2 && n.Int64() == 1 {
					out = append(out, inst)
				}
			}
			return true
		})
		return &object.List{V: out}, nil
	}})

	m.Dict.SetStr("freeze_support", &object.BuiltinFunc{Name: "freeze_support", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	m.Dict.SetStr("get_start_method", &object.BuiltinFunc{Name: "get_start_method", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "fork"}, nil
	}})

	m.Dict.SetStr("set_start_method", &object.BuiltinFunc{Name: "set_start_method", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	m.Dict.SetStr("get_context", &object.BuiltinFunc{Name: "get_context", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return m, nil
	}})

	m.Dict.SetStr("allow_connection_pickling", &object.BuiltinFunc{Name: "allow_connection_pickling", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// --- exception classes ---
	for _, name := range []string{"ProcessError", "AuthenticationError", "BufferTooShort", "TimeoutError"} {
		cls := &object.Class{Name: name, Dict: object.NewDict()}
		m.Dict.SetStr(name, cls)
	}

	_ = mainProc
	return m
}

// ─── Process ────────────────────────────────────────────────────────────────

func (i *Interp) makeMPProcess(name string, gid int64, target object.Object, args *object.Tuple, kwargs *object.Dict, daemon bool) *object.Instance {
	cls := &object.Class{Name: "Process", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	var alive atomic.Int32 // 0=not started, 1=running, 2=done
	var wg sync.WaitGroup
	var exitcode atomic.Int32
	exitcode.Store(-1)

	daemonVal := object.False
	if daemon {
		daemonVal = object.True
	}
	inst.Dict.SetStr("name", &object.Str{V: name})
	inst.Dict.SetStr("daemon", daemonVal)
	inst.Dict.SetStr("_alive", object.NewInt(0))
	inst.Dict.SetStr("_parent", object.None)
	inst.Dict.SetStr("pid", object.None)
	inst.Dict.SetStr("exitcode", object.None)
	inst.Dict.SetStr("authkey", &object.Bytes{})
	inst.Dict.SetStr("sentinel", object.None)

	cls.Dict.SetStr("start", &object.BuiltinFunc{Name: "start", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if alive.Load() != 0 {
			return object.None, object.Errorf(i.assertErr, "process already started")
		}
		n := atomic.AddInt64(&mpPIDCounter, 1)
		inst.Dict.SetStr("pid", object.NewInt(n))
		inst.Dict.SetStr("_alive", object.NewInt(1))
		alive.Store(1)
		atomic.AddInt64(&mpAlive, 1)
		wg.Add(1)

		ti := i.threadCopy()
		go func() {
			gid2 := goroutineID()
			mpCurrentProcesses.Store(gid2, inst)
			defer func() {
				mpCurrentProcesses.Delete(gid2)
				inst.Dict.SetStr("_alive", object.NewInt(0))
				alive.Store(2)
				atomic.AddInt64(&mpAlive, -1)
				wg.Done()
			}()
			code := int32(0)
			if target != nil {
				var callArgs []object.Object
				if args != nil {
					callArgs = args.V
				}
				if _, err := ti.callObject(target, callArgs, kwargs); err != nil {
					code = 1
				}
			}
			exitcode.Store(code)
		}()
		return object.None, nil
	}})

	cls.Dict.SetStr("join", &object.BuiltinFunc{Name: "join", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		timeout := -1.0
		if len(a) > 0 {
			if f, ok := a[0].(*object.Float); ok {
				timeout = f.V
			} else if n, ok2 := a[0].(*object.Int); ok2 && n.IsInt64() {
				timeout = float64(n.Int64())
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("timeout"); ok {
				if f, ok2 := v.(*object.Float); ok2 {
					timeout = f.V
				} else if n, ok2 := v.(*object.Int); ok2 {
					timeout = float64(n.Int64())
				}
			}
		}
		if timeout < 0 {
			wg.Wait()
		} else {
			done := make(chan struct{})
			go func() { wg.Wait(); close(done) }()
			select {
			case <-done:
			case <-time.After(time.Duration(timeout * float64(time.Second))):
			}
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("is_alive", &object.BuiltinFunc{Name: "is_alive", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(alive.Load() == 1), nil
	}})

	cls.Dict.SetStr("terminate", &object.BuiltinFunc{Name: "terminate", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	cls.Dict.SetStr("kill", &object.BuiltinFunc{Name: "kill", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// exitcode via __getattr__ (only for "exitcode" key — other attrs resolved normally)
	cls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok && s.V == "exitcode" {
				if alive.Load() == 0 {
					return object.None, nil
				}
				return object.NewInt(int64(exitcode.Load())), nil
			}
		}
		return object.None, nil
	}})

	if gid != 0 {
		inst.Dict.SetStr("pid", object.NewInt(gid))
	}

	return inst
}

func (i *Interp) makeCurrentMainProc() *object.Instance {
	gid := goroutineID()
	if v, ok := mpCurrentProcesses.Load(gid); ok {
		return v.(*object.Instance)
	}
	return i.makeMPProcess("MainProcess", gid, nil, nil, nil, false)
}

// ─── Queue ──────────────────────────────────────────────────────────────────

type mpQueueState struct {
	ch      chan object.Object
	maxsize int
	closed  atomic.Bool
}

func (i *Interp) makeMPQueue(maxsize int) *object.Instance {
	cls := &object.Class{Name: "Queue", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	var ch chan object.Object
	if maxsize > 0 {
		ch = make(chan object.Object, maxsize)
	} else {
		ch = make(chan object.Object, 1<<20)
	}
	qs := &mpQueueState{ch: ch, maxsize: maxsize}

	cls.Dict.SetStr("put", &object.BuiltinFunc{Name: "put", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if len(a) == 0 {
			return object.None, object.Errorf(i.typeErr, "put() missing argument")
		}
		item := a[0]
		block := true
		timeout := -1.0
		if len(a) >= 2 {
			if b, ok := a[1].(*object.Bool); ok {
				block = b.V
			}
		}
		if len(a) >= 3 {
			if f, ok := a[2].(*object.Float); ok {
				timeout = f.V
			} else if n, ok2 := a[2].(*object.Int); ok2 {
				timeout = float64(n.Int64())
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("block"); ok {
				if b, ok2 := v.(*object.Bool); ok2 {
					block = b.V
				}
			}
			if v, ok := kw.GetStr("timeout"); ok {
				if f, ok2 := v.(*object.Float); ok2 {
					timeout = f.V
				} else if n, ok2 := v.(*object.Int); ok2 {
					timeout = float64(n.Int64())
				}
			}
		}
		if !block {
			select {
			case ch <- item:
				return object.None, nil
			default:
				return object.None, object.Errorf(i.runtimeErr, "queue.Full")
			}
		}
		if timeout >= 0 {
			select {
			case ch <- item:
				return object.None, nil
			case <-time.After(time.Duration(timeout * float64(time.Second))):
				return object.None, object.Errorf(i.runtimeErr, "queue.Full")
			}
		}
		ch <- item
		return object.None, nil
	}})

	cls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		block := true
		timeout := -1.0
		if len(a) >= 1 {
			if b, ok := a[0].(*object.Bool); ok {
				block = b.V
			}
		}
		if len(a) >= 2 {
			if f, ok := a[1].(*object.Float); ok {
				timeout = f.V
			} else if n, ok2 := a[1].(*object.Int); ok2 {
				timeout = float64(n.Int64())
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("block"); ok {
				if b, ok2 := v.(*object.Bool); ok2 {
					block = b.V
				}
			}
			if v, ok := kw.GetStr("timeout"); ok {
				if f, ok2 := v.(*object.Float); ok2 {
					timeout = f.V
				} else if n, ok2 := v.(*object.Int); ok2 {
					timeout = float64(n.Int64())
				}
			}
		}
		if !block {
			select {
			case v := <-ch:
				return v, nil
			default:
				return object.None, object.Errorf(i.runtimeErr, "queue.Empty")
			}
		}
		if timeout >= 0 {
			select {
			case v := <-ch:
				return v, nil
			case <-time.After(time.Duration(timeout * float64(time.Second))):
				return object.None, object.Errorf(i.runtimeErr, "queue.Empty")
			}
		}
		v := <-ch
		return v, nil
	}})

	cls.Dict.SetStr("put_nowait", &object.BuiltinFunc{Name: "put_nowait", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if len(a) == 0 {
			return object.None, object.Errorf(i.typeErr, "put_nowait() missing argument")
		}
		select {
		case ch <- a[0]:
			return object.None, nil
		default:
			return object.None, object.Errorf(i.runtimeErr, "queue.Full")
		}
	}})

	cls.Dict.SetStr("get_nowait", &object.BuiltinFunc{Name: "get_nowait", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		select {
		case v := <-ch:
			return v, nil
		default:
			return object.None, object.Errorf(i.runtimeErr, "queue.Empty")
		}
	}})

	cls.Dict.SetStr("empty", &object.BuiltinFunc{Name: "empty", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(len(ch) == 0), nil
	}})

	cls.Dict.SetStr("full", &object.BuiltinFunc{Name: "full", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if qs.maxsize <= 0 {
			return object.False, nil
		}
		return object.BoolOf(len(ch) >= qs.maxsize), nil
	}})

	cls.Dict.SetStr("qsize", &object.BuiltinFunc{Name: "qsize", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(len(ch))), nil
	}})

	cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if qs.closed.CompareAndSwap(false, true) {
			close(ch)
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("join_thread", &object.BuiltinFunc{Name: "join_thread", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	cls.Dict.SetStr("cancel_join_thread", &object.BuiltinFunc{Name: "cancel_join_thread", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	cls.Dict.SetStr("task_done", &object.BuiltinFunc{Name: "task_done", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	cls.Dict.SetStr("join", &object.BuiltinFunc{Name: "join", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	return inst
}

// ─── Pipe / Connection ──────────────────────────────────────────────────────

func (i *Interp) makeMPPipe(duplex bool) (object.Object, error) {
	ch1 := make(chan object.Object, 1024)
	ch2 := make(chan object.Object, 1024)

	var conn1, conn2 *object.Instance
	if duplex {
		// conn1: sends on ch1, recvs on ch2
		// conn2: sends on ch2, recvs on ch1
		conn1 = i.makeMPConnection(ch1, ch2)
		conn2 = i.makeMPConnection(ch2, ch1)
	} else {
		// Pipe(False) returns (reader, writer) — conn1=reader, conn2=writer
		conn1 = i.makeMPConnection(nil, ch1) // read-only
		conn2 = i.makeMPConnection(ch1, nil) // write-only
	}
	return &object.Tuple{V: []object.Object{conn1, conn2}}, nil
}

func (i *Interp) makeMPConnection(send, recv chan object.Object) *object.Instance {
	cls := &object.Class{Name: "Connection", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	var closed atomic.Bool

	cls.Dict.SetStr("send", &object.BuiltinFunc{Name: "send", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if closed.Load() {
			return object.None, object.Errorf(i.osErr, "handle is closed")
		}
		if send == nil {
			return object.None, object.Errorf(i.osErr, "connection is read-only")
		}
		if len(a) == 0 {
			return object.None, object.Errorf(i.typeErr, "send() missing argument")
		}
		send <- a[0]
		return object.None, nil
	}})

	cls.Dict.SetStr("recv", &object.BuiltinFunc{Name: "recv", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if closed.Load() {
			return object.None, object.Errorf(i.osErr, "handle is closed")
		}
		if recv == nil {
			return object.None, object.Errorf(i.osErr, "connection is write-only")
		}
		v, ok := <-recv
		if !ok {
			return object.None, object.Errorf(i.eofErr, "EOFError")
		}
		return v, nil
	}})

	cls.Dict.SetStr("send_bytes", &object.BuiltinFunc{Name: "send_bytes", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if closed.Load() {
			return object.None, object.Errorf(i.osErr, "handle is closed")
		}
		if send == nil {
			return object.None, object.Errorf(i.osErr, "connection is read-only")
		}
		if len(a) == 0 {
			return object.None, object.Errorf(i.typeErr, "send_bytes() missing argument")
		}
		send <- a[0]
		return object.None, nil
	}})

	cls.Dict.SetStr("recv_bytes", &object.BuiltinFunc{Name: "recv_bytes", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if closed.Load() {
			return object.None, object.Errorf(i.osErr, "handle is closed")
		}
		if recv == nil {
			return object.None, object.Errorf(i.osErr, "connection is write-only")
		}
		v, ok := <-recv
		if !ok {
			return object.None, object.Errorf(i.eofErr, "EOFError")
		}
		return v, nil
	}})

	cls.Dict.SetStr("poll", &object.BuiltinFunc{Name: "poll", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if recv == nil {
			return object.False, nil
		}
		timeout := 0.0
		if len(a) > 0 {
			if f, ok := a[0].(*object.Float); ok {
				timeout = f.V
			} else if n, ok2 := a[0].(*object.Int); ok2 {
				timeout = float64(n.Int64())
			}
		}
		if timeout == 0 {
			return object.BoolOf(len(recv) > 0), nil
		}
		select {
		case v := <-recv:
			recv <- v
			return object.True, nil
		case <-time.After(time.Duration(timeout * float64(time.Second))):
			return object.False, nil
		}
	}})

	cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		closed.Store(true)
		return object.None, nil
	}})

	cls.Dict.SetStr("fileno", &object.BuiltinFunc{Name: "fileno", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, object.Errorf(i.osErr, "no file descriptor")
	}})

	return inst
}

// ─── Pool ───────────────────────────────────────────────────────────────────

type poolJob struct {
	fn     object.Object
	args   []object.Object
	kwargs *object.Dict
	result chan poolResult
}

type poolResult struct {
	val object.Object
	err error
}

type mpPoolState struct {
	jobs   chan poolJob
	wg     sync.WaitGroup
	closed atomic.Bool
}

func (i *Interp) makeMPPool(nproc int, initializer object.Object, initArgs *object.Tuple) *object.Instance {
	cls := &object.Class{Name: "Pool", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	jobs := make(chan poolJob, nproc*4)
	ps := &mpPoolState{jobs: jobs}

	for w := 0; w < nproc; w++ {
		ps.wg.Add(1)
		wi := i.threadCopy()
		go func() {
			defer ps.wg.Done()
			if initializer != nil {
				var ia []object.Object
				if initArgs != nil {
					ia = initArgs.V
				}
				wi.callObject(initializer, ia, nil) //nolint
			}
			for job := range jobs {
				v, err := wi.callObject(job.fn, job.args, job.kwargs)
				job.result <- poolResult{val: v, err: err}
			}
		}()
	}

	cls.Dict.SetStr("apply", &object.BuiltinFunc{Name: "apply", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if ps.closed.Load() {
			return object.None, object.Errorf(i.valueErr, "Pool not running")
		}
		if len(a) == 0 {
			return object.None, object.Errorf(i.typeErr, "apply() requires func argument")
		}
		fn := a[0]
		var fargs []object.Object
		var fkwargs *object.Dict
		if len(a) >= 2 {
			switch t := a[1].(type) {
			case *object.Tuple:
				fargs = t.V
			case *object.List:
				fargs = t.V
			}
		}
		if len(a) >= 3 {
			if d, ok := a[2].(*object.Dict); ok {
				fkwargs = d
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("args"); ok {
				switch t := v.(type) {
				case *object.Tuple:
					fargs = t.V
				case *object.List:
					fargs = t.V
				}
			}
			if v, ok := kw.GetStr("kwds"); ok {
				if d, ok2 := v.(*object.Dict); ok2 {
					fkwargs = d
				}
			}
		}
		ch := make(chan poolResult, 1)
		jobs <- poolJob{fn: fn, args: fargs, kwargs: fkwargs, result: ch}
		r := <-ch
		return r.val, r.err
	}})

	cls.Dict.SetStr("apply_async", &object.BuiltinFunc{Name: "apply_async", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if ps.closed.Load() {
			return object.None, object.Errorf(i.valueErr, "Pool not running")
		}
		if len(a) == 0 {
			return object.None, object.Errorf(i.typeErr, "apply_async() requires func argument")
		}
		fn := a[0]
		var fargs []object.Object
		var fkwargs *object.Dict
		if len(a) >= 2 {
			switch t := a[1].(type) {
			case *object.Tuple:
				fargs = t.V
			case *object.List:
				fargs = t.V
			}
		}
		if len(a) >= 3 {
			if d, ok := a[2].(*object.Dict); ok {
				fkwargs = d
			}
		}
		var callback, errCallback object.Object
		if kw != nil {
			if v, ok := kw.GetStr("args"); ok {
				switch t := v.(type) {
				case *object.Tuple:
					fargs = t.V
				case *object.List:
					fargs = t.V
				}
			}
			if v, ok := kw.GetStr("callback"); ok {
				callback = v
			}
			if v, ok := kw.GetStr("error_callback"); ok {
				errCallback = v
			}
		}
		ch := make(chan poolResult, 1)
		jobs <- poolJob{fn: fn, args: fargs, kwargs: fkwargs, result: ch}
		return i.makeAsyncResult(ch, callback, errCallback), nil
	}})

	cls.Dict.SetStr("map", &object.BuiltinFunc{Name: "map", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if ps.closed.Load() {
			return object.None, object.Errorf(i.valueErr, "Pool not running")
		}
		if len(a) < 2 {
			return object.None, object.Errorf(i.typeErr, "map() requires func and iterable")
		}
		fn := a[0]
		items, err := i.iterToSlice(a[1])
		if err != nil {
			return nil, err
		}
		chs := make([]chan poolResult, len(items))
		for k, item := range items {
			chs[k] = make(chan poolResult, 1)
			jobs <- poolJob{fn: fn, args: []object.Object{item}, result: chs[k]}
		}
		out := make([]object.Object, len(items))
		for k, ch := range chs {
			r := <-ch
			if r.err != nil {
				return nil, r.err
			}
			out[k] = r.val
		}
		return &object.List{V: out}, nil
	}})

	cls.Dict.SetStr("map_async", &object.BuiltinFunc{Name: "map_async", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if ps.closed.Load() {
			return object.None, object.Errorf(i.valueErr, "Pool not running")
		}
		if len(a) < 2 {
			return object.None, object.Errorf(i.typeErr, "map_async() requires func and iterable")
		}
		fn := a[0]
		items, err := i.iterToSlice(a[1])
		if err != nil {
			return nil, err
		}
		var callback, errCallback object.Object
		if kw != nil {
			if v, ok := kw.GetStr("callback"); ok {
				callback = v
			}
			if v, ok := kw.GetStr("error_callback"); ok {
				errCallback = v
			}
		}
		chs := make([]chan poolResult, len(items))
		for k, item := range items {
			chs[k] = make(chan poolResult, 1)
			jobs <- poolJob{fn: fn, args: []object.Object{item}, result: chs[k]}
		}
		aggCh := make(chan poolResult, 1)
		go func() {
			out := make([]object.Object, len(items))
			for k, ch := range chs {
				r := <-ch
				if r.err != nil {
					aggCh <- r
					return
				}
				out[k] = r.val
			}
			aggCh <- poolResult{val: &object.List{V: out}}
		}()
		return i.makeAsyncResult(aggCh, callback, errCallback), nil
	}})

	cls.Dict.SetStr("imap", &object.BuiltinFunc{Name: "imap", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if ps.closed.Load() {
			return object.None, object.Errorf(i.valueErr, "Pool not running")
		}
		if len(a) < 2 {
			return object.None, object.Errorf(i.typeErr, "imap() requires func and iterable")
		}
		fn := a[0]
		items, err := i.iterToSlice(a[1])
		if err != nil {
			return nil, err
		}
		chs := make([]chan poolResult, len(items))
		for k, item := range items {
			chs[k] = make(chan poolResult, 1)
			jobs <- poolJob{fn: fn, args: []object.Object{item}, result: chs[k]}
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(chs) {
				return nil, false, nil
			}
			r := <-chs[idx]
			idx++
			return r.val, true, r.err
		}}, nil
	}})

	cls.Dict.SetStr("imap_unordered", &object.BuiltinFunc{Name: "imap_unordered", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if ps.closed.Load() {
			return object.None, object.Errorf(i.valueErr, "Pool not running")
		}
		if len(a) < 2 {
			return object.None, object.Errorf(i.typeErr, "imap_unordered() requires func and iterable")
		}
		fn := a[0]
		items, err := i.iterToSlice(a[1])
		if err != nil {
			return nil, err
		}
		merged := make(chan poolResult, len(items))
		for _, item := range items {
			ch := make(chan poolResult, 1)
			jobs <- poolJob{fn: fn, args: []object.Object{item}, result: ch}
			go func(c chan poolResult) {
				r := <-c
				merged <- r
			}(ch)
		}
		remaining := len(items)
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if remaining <= 0 {
				return nil, false, nil
			}
			r := <-merged
			remaining--
			return r.val, true, r.err
		}}, nil
	}})

	cls.Dict.SetStr("starmap", &object.BuiltinFunc{Name: "starmap", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if ps.closed.Load() {
			return object.None, object.Errorf(i.valueErr, "Pool not running")
		}
		if len(a) < 2 {
			return object.None, object.Errorf(i.typeErr, "starmap() requires func and iterable")
		}
		fn := a[0]
		items, err := i.iterToSlice(a[1])
		if err != nil {
			return nil, err
		}
		chs := make([]chan poolResult, len(items))
		for k, item := range items {
			var callArgs []object.Object
			switch t := item.(type) {
			case *object.Tuple:
				callArgs = t.V
			case *object.List:
				callArgs = t.V
			default:
				callArgs = []object.Object{item}
			}
			chs[k] = make(chan poolResult, 1)
			jobs <- poolJob{fn: fn, args: callArgs, result: chs[k]}
		}
		out := make([]object.Object, len(items))
		for k, ch := range chs {
			r := <-ch
			if r.err != nil {
				return nil, r.err
			}
			out[k] = r.val
		}
		return &object.List{V: out}, nil
	}})

	cls.Dict.SetStr("starmap_async", &object.BuiltinFunc{Name: "starmap_async", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if ps.closed.Load() {
			return object.None, object.Errorf(i.valueErr, "Pool not running")
		}
		if len(a) < 2 {
			return object.None, object.Errorf(i.typeErr, "starmap_async() requires func and iterable")
		}
		fn := a[0]
		items, err := i.iterToSlice(a[1])
		if err != nil {
			return nil, err
		}
		var callback object.Object
		if kw != nil {
			if v, ok := kw.GetStr("callback"); ok {
				callback = v
			}
		}
		chs := make([]chan poolResult, len(items))
		for k, item := range items {
			var callArgs []object.Object
			switch t := item.(type) {
			case *object.Tuple:
				callArgs = t.V
			case *object.List:
				callArgs = t.V
			default:
				callArgs = []object.Object{item}
			}
			chs[k] = make(chan poolResult, 1)
			jobs <- poolJob{fn: fn, args: callArgs, result: chs[k]}
		}
		aggCh := make(chan poolResult, 1)
		go func() {
			out := make([]object.Object, len(items))
			for k, ch := range chs {
				r := <-ch
				if r.err != nil {
					aggCh <- r
					return
				}
				out[k] = r.val
			}
			aggCh <- poolResult{val: &object.List{V: out}}
		}()
		return i.makeAsyncResult(aggCh, callback, nil), nil
	}})

	cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if ps.closed.CompareAndSwap(false, true) {
			close(jobs)
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("terminate", &object.BuiltinFunc{Name: "terminate", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if ps.closed.CompareAndSwap(false, true) {
			close(jobs)
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("join", &object.BuiltinFunc{Name: "join", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		ps.wg.Wait()
		return object.None, nil
	}})

	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return inst, nil
	}})

	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if ps.closed.CompareAndSwap(false, true) {
			close(jobs)
		}
		ps.wg.Wait()
		return object.False, nil
	}})

	return inst
}

func (i *Interp) makeAsyncResult(ch chan poolResult, callback, errCallback object.Object) *object.Instance {
	cls := &object.Class{Name: "AsyncResult", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	var cached *poolResult
	var mu sync.Mutex

	getResult := func() poolResult {
		mu.Lock()
		defer mu.Unlock()
		if cached != nil {
			return *cached
		}
		r := <-ch
		cached = &r
		if r.err == nil && callback != nil {
			i.callObject(callback, []object.Object{r.val}, nil) //nolint
		}
		if r.err != nil && errCallback != nil {
			i.callObject(errCallback, []object.Object{&object.Str{V: r.err.Error()}}, nil) //nolint
		}
		return r
	}

	cls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		r := getResult()
		return r.val, r.err
	}})

	cls.Dict.SetStr("wait", &object.BuiltinFunc{Name: "wait", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		getResult()
		return object.None, nil
	}})

	cls.Dict.SetStr("ready", &object.BuiltinFunc{Name: "ready", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Lock()
		defer mu.Unlock()
		if cached != nil {
			return object.True, nil
		}
		select {
		case r := <-ch:
			cached = &r
			return object.True, nil
		default:
			return object.False, nil
		}
	}})

	cls.Dict.SetStr("successful", &object.BuiltinFunc{Name: "successful", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Lock()
		defer mu.Unlock()
		if cached == nil {
			return object.None, object.Errorf(i.valueErr, "AsyncResult not ready")
		}
		return object.BoolOf(cached.err == nil), nil
	}})

	_ = inst
	return inst
}

// iterToSlice converts a Python iterable to a Go slice.
func (i *Interp) iterToSlice(obj object.Object) ([]object.Object, error) {
	switch t := obj.(type) {
	case *object.List:
		return t.V, nil
	case *object.Tuple:
		return t.V, nil
	case *object.Range:
		var out []object.Object
		for v := t.Start; v < t.Stop; v += t.Step {
			out = append(out, object.NewInt(v))
		}
		return out, nil
	}
	return nil, object.Errorf(i.typeErr, "not iterable")
}

// ─── Value ──────────────────────────────────────────────────────────────────

func (i *Interp) makeMPValue(initial object.Object) *object.Instance {
	cls := &object.Class{Name: "Value", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	var mu sync.Mutex
	var val object.Object = initial
	lockInst := i.makeLock()

	// Store value in inst.Dict so normal attribute read finds it;
	// __setattr__ intercepts writes to update the mutex-protected copy.
	inst.Dict.SetStr("value", val)

	cls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				switch s.V {
				case "value":
					mu.Lock()
					v := val
					mu.Unlock()
					return v, nil
				case "get_lock":
					return &object.BuiltinFunc{Name: "get_lock", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
						return lockInst, nil
					}}, nil
				}
			}
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("__setattr__", &object.BuiltinFunc{Name: "__setattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 3 {
			if s, ok := a[1].(*object.Str); ok && s.V == "value" {
				mu.Lock()
				val = a[2]
				mu.Unlock()
				inst.Dict.SetStr("value", a[2])
			}
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("get_lock", &object.BuiltinFunc{Name: "get_lock", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return lockInst, nil
	}})

	return inst
}

// ─── Array ──────────────────────────────────────────────────────────────────

func (i *Interp) makeMPArray(initial []object.Object) *object.Instance {
	cls := &object.Class{Name: "Array", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	var mu sync.RWMutex
	data := make([]object.Object, len(initial))
	copy(data, initial)
	lockInst := i.makeLock()

	cls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.RLock()
		n := len(data)
		mu.RUnlock()
		return object.NewInt(int64(n)), nil
	}})

	cls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if len(a) < 1 {
			return object.None, object.Errorf(i.indexErr, "IndexError")
		}
		n, ok := a[0].(*object.Int)
		if !ok {
			return object.None, object.Errorf(i.typeErr, "indices must be integers")
		}
		mu.RLock()
		defer mu.RUnlock()
		idx := int(n.Int64())
		if idx < 0 {
			idx += len(data)
		}
		if idx < 0 || idx >= len(data) {
			return object.None, object.Errorf(i.indexErr, "array index out of range")
		}
		return data[idx], nil
	}})

	cls.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		if len(a) < 2 {
			return object.None, object.Errorf(i.indexErr, "IndexError")
		}
		n, ok := a[0].(*object.Int)
		if !ok {
			return object.None, object.Errorf(i.typeErr, "indices must be integers")
		}
		mu.Lock()
		defer mu.Unlock()
		idx := int(n.Int64())
		if idx < 0 {
			idx += len(data)
		}
		if idx < 0 || idx >= len(data) {
			return object.None, object.Errorf(i.indexErr, "array index out of range")
		}
		data[idx] = a[1]
		return object.None, nil
	}})

	cls.Dict.SetStr("get_lock", &object.BuiltinFunc{Name: "get_lock", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return lockInst, nil
	}})

	_ = inst
	return inst
}

// ─── Manager ────────────────────────────────────────────────────────────────

func (i *Interp) makeMPManager() *object.Instance {
	cls := &object.Class{Name: "SyncManager", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	cls.Dict.SetStr("start", &object.BuiltinFunc{Name: "start", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	cls.Dict.SetStr("shutdown", &object.BuiltinFunc{Name: "shutdown", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	cls.Dict.SetStr("dict", &object.BuiltinFunc{Name: "dict", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		d := object.NewDict()
		a = mpArgs(a)
		if len(a) > 0 {
			if src, ok := a[0].(*object.Dict); ok {
				keys, vals := src.Items()
				for k, key := range keys {
					d.Set(key, vals[k]) //nolint
				}
			}
		}
		return d, nil
	}})

	cls.Dict.SetStr("list", &object.BuiltinFunc{Name: "list", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		var v []object.Object
		if len(a) > 0 {
			switch t := a[0].(type) {
			case *object.List:
				v = append([]object.Object{}, t.V...)
			case *object.Tuple:
				v = append([]object.Object{}, t.V...)
			}
		}
		return &object.List{V: v}, nil
	}})

	cls.Dict.SetStr("Value", &object.BuiltinFunc{Name: "Value", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		var val object.Object = object.None
		if len(a) >= 2 {
			val = a[1]
		}
		return i.makeMPValue(val), nil
	}})

	cls.Dict.SetStr("Array", &object.BuiltinFunc{Name: "Array", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		var init []object.Object
		if len(a) >= 2 {
			switch t := a[1].(type) {
			case *object.List:
				init = t.V
			case *object.Tuple:
				init = t.V
			}
		}
		return i.makeMPArray(init), nil
	}})

	cls.Dict.SetStr("Queue", &object.BuiltinFunc{Name: "Queue", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		a = mpArgs(a)
		maxsize := 0
		if len(a) > 0 {
			if n, ok := a[0].(*object.Int); ok && n.IsInt64() {
				maxsize = int(n.Int64())
			}
		}
		return i.makeMPQueue(maxsize), nil
	}})

	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return inst, nil
	}})

	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})

	return inst
}

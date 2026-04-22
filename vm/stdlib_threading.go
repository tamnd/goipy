package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildThreading constructs a minimal threading module for goipy.
// Since the interpreter is single-goroutine, Thread.start() runs the
// target synchronously. Lock/RLock are context-manager no-ops.
func (i *Interp) buildThreading() *object.Module {
	m := &object.Module{Name: "threading", Dict: object.NewDict()}

	mainThread := i.makeThread("MainThread")

	// Lock
	m.Dict.SetStr("Lock", &object.BuiltinFunc{Name: "Lock", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return i.makeLock(false), nil
	}})

	// RLock
	m.Dict.SetStr("RLock", &object.BuiltinFunc{Name: "RLock", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return i.makeLock(true), nil
	}})

	// Thread(target, args=(), kwargs={}, daemon=None, name=None)
	m.Dict.SetStr("Thread", &object.BuiltinFunc{Name: "Thread", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		var target object.Object
		var args *object.Tuple
		var kwargs *object.Dict
		name := "Thread"

		if kw != nil {
			if v, ok := kw.GetStr("target"); ok {
				target = v
			}
			if v, ok := kw.GetStr("args"); ok {
				if t, ok2 := v.(*object.Tuple); ok2 {
					args = t
				} else if l, ok2 := v.(*object.List); ok2 {
					args = &object.Tuple{V: l.V}
				}
			}
			if v, ok := kw.GetStr("kwargs"); ok {
				if d, ok2 := v.(*object.Dict); ok2 {
					kwargs = d
				}
			}
			if v, ok := kw.GetStr("name"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					name = s.V
				}
			}
		}
		if args == nil {
			args = &object.Tuple{}
		}
		return i.makeThreadObj(name, target, args, kwargs), nil
	}})

	// Event
	m.Dict.SetStr("Event", &object.BuiltinFunc{Name: "Event", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return i.makeEvent(), nil
	}})

	// Semaphore(value=1)
	m.Dict.SetStr("Semaphore", &object.BuiltinFunc{Name: "Semaphore", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		val := int64(1)
		if len(a) > 0 {
			if n, ok := toInt64(a[0]); ok {
				val = n
			}
		}
		return i.makeSemaphore(val), nil
	}})

	// BoundedSemaphore(value=1)
	m.Dict.SetStr("BoundedSemaphore", &object.BuiltinFunc{Name: "BoundedSemaphore", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		val := int64(1)
		if len(a) > 0 {
			if n, ok := toInt64(a[0]); ok {
				val = n
			}
		}
		return i.makeSemaphore(val), nil
	}})

	// Condition(lock=None)
	m.Dict.SetStr("Condition", &object.BuiltinFunc{Name: "Condition", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return i.makeCondition(), nil
	}})

	// Barrier(parties, action=None, timeout=None)
	m.Dict.SetStr("Barrier", &object.BuiltinFunc{Name: "Barrier", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		parties := int64(1)
		if len(a) > 0 {
			if n, ok := toInt64(a[0]); ok {
				parties = n
			}
		}
		return i.makeBarrier(parties), nil
	}})

	// current_thread() / currentThread()
	ctFn := &object.BuiltinFunc{Name: "current_thread", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return mainThread, nil
	}}
	m.Dict.SetStr("current_thread", ctFn)
	m.Dict.SetStr("currentThread", ctFn)

	// main_thread()
	m.Dict.SetStr("main_thread", &object.BuiltinFunc{Name: "main_thread", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return mainThread, nil
	}})

	// active_count() / activeCount()
	acFn := &object.BuiltinFunc{Name: "active_count", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(1), nil
	}}
	m.Dict.SetStr("active_count", acFn)
	m.Dict.SetStr("activeCount", acFn)

	// enumerate()
	m.Dict.SetStr("enumerate", &object.BuiltinFunc{Name: "enumerate", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.List{V: []object.Object{mainThread}}, nil
	}})

	// local()
	m.Dict.SetStr("local", &object.BuiltinFunc{Name: "local", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		inst := &object.Instance{Dict: object.NewDict()}
		return inst, nil
	}})

	// get_ident()
	m.Dict.SetStr("get_ident", &object.BuiltinFunc{Name: "get_ident", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(0), nil
	}})

	// get_native_id()
	m.Dict.SetStr("get_native_id", &object.BuiltinFunc{Name: "get_native_id", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(0), nil
	}})

	// settrace / setprofile — no-ops
	m.Dict.SetStr("settrace", &object.BuiltinFunc{Name: "settrace", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	m.Dict.SetStr("setprofile", &object.BuiltinFunc{Name: "setprofile", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// TIMEOUT_MAX
	m.Dict.SetStr("TIMEOUT_MAX", &object.Float{V: 1e308})

	return m
}

// makeThread creates a Thread-like instance with name/ident attributes only.
func (i *Interp) makeThread(name string) *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	inst.Dict.SetStr("name", &object.Str{V: name})
	inst.Dict.SetStr("daemon", object.False)
	inst.Dict.SetStr("ident", object.NewInt(0))
	inst.Dict.SetStr("native_id", object.NewInt(0))
	return inst
}

// makeThreadObj creates a Thread instance with start/join/is_alive methods.
func (i *Interp) makeThreadObj(name string, target object.Object, args *object.Tuple, kwargs *object.Dict) *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	inst.Dict.SetStr("name", &object.Str{V: name})
	inst.Dict.SetStr("daemon", object.False)
	inst.Dict.SetStr("ident", object.None)
	inst.Dict.SetStr("native_id", object.None)

	inst.Dict.SetStr("start", &object.BuiltinFunc{Name: "start", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		inst.Dict.SetStr("ident", object.NewInt(0))
		inst.Dict.SetStr("native_id", object.NewInt(0))
		if target == nil {
			return object.None, nil
		}
		argSlice := args.V
		_, err := i.callObject(target, argSlice, kwargs)
		return object.None, err
	}})

	inst.Dict.SetStr("join", &object.BuiltinFunc{Name: "join", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	inst.Dict.SetStr("is_alive", &object.BuiltinFunc{Name: "is_alive", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})

	inst.Dict.SetStr("run", &object.BuiltinFunc{Name: "run", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if target == nil {
			return object.None, nil
		}
		_, err := i.callObject(target, args.V, kwargs)
		return object.None, err
	}})

	return inst
}

// makeLock creates a Lock or RLock instance (context manager no-op).
func (i *Interp) makeLock(reentrant bool) *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	locked := false

	inst.Dict.SetStr("acquire", &object.BuiltinFunc{Name: "acquire", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		locked = true
		return object.True, nil
	}})

	inst.Dict.SetStr("release", &object.BuiltinFunc{Name: "release", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		locked = false
		return object.None, nil
	}})

	inst.Dict.SetStr("locked", &object.BuiltinFunc{Name: "locked", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(locked), nil
	}})

	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		locked = true
		return inst, nil
	}})

	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		locked = false
		return object.False, nil
	}})

	_ = reentrant
	return inst
}

// makeEvent creates an Event instance.
func (i *Interp) makeEvent() *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	set := false

	inst.Dict.SetStr("is_set", &object.BuiltinFunc{Name: "is_set", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(set), nil
	}})

	inst.Dict.SetStr("set", &object.BuiltinFunc{Name: "set", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		set = true
		return object.None, nil
	}})

	inst.Dict.SetStr("clear", &object.BuiltinFunc{Name: "clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		set = false
		return object.None, nil
	}})

	// wait(timeout=None) — in sequential execution, returns is_set immediately
	inst.Dict.SetStr("wait", &object.BuiltinFunc{Name: "wait", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(set), nil
	}})

	return inst
}

// makeSemaphore creates a Semaphore instance.
func (i *Interp) makeSemaphore(value int64) *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	count := value

	inst.Dict.SetStr("acquire", &object.BuiltinFunc{Name: "acquire", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if count > 0 {
			count--
			return object.True, nil
		}
		return object.False, nil
	}})

	inst.Dict.SetStr("release", &object.BuiltinFunc{Name: "release", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		n := int64(1)
		if len(a) > 0 {
			if v, ok := toInt64(a[0]); ok {
				n = v
			}
		}
		count += n
		return object.None, nil
	}})

	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if count > 0 {
			count--
		}
		return inst, nil
	}})

	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		count++
		return object.False, nil
	}})

	return inst
}

// makeCondition creates a Condition instance.
func (i *Interp) makeCondition() *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	lock := i.makeLock(false)

	inst.Dict.SetStr("acquire", &object.BuiltinFunc{Name: "acquire", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if fn, ok := lock.Dict.GetStr("acquire"); ok {
			return i.callObject(fn, nil, nil)
		}
		return object.True, nil
	}})

	inst.Dict.SetStr("release", &object.BuiltinFunc{Name: "release", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if fn, ok := lock.Dict.GetStr("release"); ok {
			return i.callObject(fn, nil, nil)
		}
		return object.None, nil
	}})

	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if fn, ok := lock.Dict.GetStr("acquire"); ok {
			i.callObject(fn, nil, nil) //nolint
		}
		return inst, nil
	}})

	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if fn, ok := lock.Dict.GetStr("release"); ok {
			i.callObject(fn, nil, nil) //nolint
		}
		return object.False, nil
	}})

	inst.Dict.SetStr("wait", &object.BuiltinFunc{Name: "wait", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.True, nil
	}})

	inst.Dict.SetStr("notify", &object.BuiltinFunc{Name: "notify", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	inst.Dict.SetStr("notify_all", &object.BuiltinFunc{Name: "notify_all", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	inst.Dict.SetStr("notifyAll", &object.BuiltinFunc{Name: "notifyAll", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	inst.Dict.SetStr("wait_for", &object.BuiltinFunc{Name: "wait_for", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			r, err := i.callObject(a[0], nil, nil)
			if err != nil {
				return nil, err
			}
			return r, nil
		}
		return object.True, nil
	}})

	return inst
}

// makeBarrier creates a Barrier instance.
func (i *Interp) makeBarrier(parties int64) *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	broken := false
	nWaiting := int64(0)

	inst.Dict.SetStr("parties", object.NewInt(parties))
	inst.Dict.SetStr("n_waiting", object.NewInt(nWaiting))
	inst.Dict.SetStr("broken", object.BoolOf(broken))

	inst.Dict.SetStr("wait", &object.BuiltinFunc{Name: "wait", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if broken {
			return nil, object.Errorf(i.runtimeErr, "BrokenBarrierError")
		}
		return object.NewInt(0), nil
	}})

	inst.Dict.SetStr("reset", &object.BuiltinFunc{Name: "reset", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		broken = false
		nWaiting = 0
		inst.Dict.SetStr("broken", object.False)
		inst.Dict.SetStr("n_waiting", object.NewInt(0))
		return object.None, nil
	}})

	inst.Dict.SetStr("abort", &object.BuiltinFunc{Name: "abort", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		broken = true
		inst.Dict.SetStr("broken", object.True)
		return object.None, nil
	}})

	return inst
}

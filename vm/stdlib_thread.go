package vm

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/tamnd/goipy/object"
)

// buildThread constructs the _thread module (low-level threading API).
func (i *Interp) buildThread() *object.Module {
	m := &object.Module{Name: "_thread", Dict: object.NewDict()}

	// error = RuntimeError
	m.Dict.SetStr("error", i.runtimeErr)

	// TIMEOUT_MAX — very large value; matches threading.TIMEOUT_MAX on most platforms
	m.Dict.SetStr("TIMEOUT_MAX", &object.Float{V: 1e308})

	// LockType class — we build one lock to extract its class, then expose the class.
	lockCls := &object.Class{Name: "lock", Dict: object.NewDict()}
	m.Dict.SetStr("LockType", lockCls)

	// allocate_lock() → new lock instance
	m.Dict.SetStr("allocate_lock", &object.BuiltinFunc{Name: "allocate_lock",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return i.makeThreadLock(lockCls), nil
		}})

	// get_ident() → goroutine ID
	m.Dict.SetStr("get_ident", &object.BuiltinFunc{Name: "get_ident",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(goroutineID()), nil
		}})

	// get_native_id() → same as get_ident in goipy
	m.Dict.SetStr("get_native_id", &object.BuiltinFunc{Name: "get_native_id",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(goroutineID()), nil
		}})

	// start_new_thread(function, args[, kwargs]) → thread identifier
	m.Dict.SetStr("start_new_thread", &object.BuiltinFunc{Name: "start_new_thread",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr,
					"start_new_thread() requires function and args")
			}
			fn := a[0]

			// Convert args tuple/list to []object.Object.
			var posArgs []object.Object
			switch t := a[1].(type) {
			case *object.Tuple:
				posArgs = append([]object.Object(nil), t.V...)
			case *object.List:
				posArgs = append([]object.Object(nil), t.V...)
			case *object.NoneType:
				posArgs = nil
			default:
				return nil, object.Errorf(i.typeErr,
					"start_new_thread() args must be a tuple")
			}

			// Optional kwargs dict.
			var kwDict *object.Dict
			if len(a) >= 3 {
				if d, ok := a[2].(*object.Dict); ok {
					kwDict = d
				}
			}

			// Channel to receive the goroutine ID from the new thread.
			idCh := make(chan int64, 1)
			wi := i.threadCopy()
			go func() {
				idCh <- goroutineID()
				// Silently swallow all errors (including SystemExit) per CPython spec.
				wi.callObject(fn, posArgs, kwDict) //nolint
			}()
			tid := <-idCh
			return object.NewInt(tid), nil
		}})

	// exit() → raise SystemExit
	m.Dict.SetStr("exit", &object.BuiltinFunc{Name: "exit",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.NewException(i.systemExit, "")
		}})

	// stack_size([size]) → always returns 0 (goroutines have dynamic stacks)
	var stackSz int64
	m.Dict.SetStr("stack_size", &object.BuiltinFunc{Name: "stack_size",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			prev := atomic.LoadInt64(&stackSz)
			if len(a) > 0 {
				if n, ok := toInt64(a[0]); ok {
					atomic.StoreInt64(&stackSz, n)
				}
			}
			return object.NewInt(prev), nil
		}})

	// interrupt_main() — no-op in goipy (no signal handling)
	m.Dict.SetStr("interrupt_main", &object.BuiltinFunc{Name: "interrupt_main",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	return m
}

// makeThreadLock creates a _thread lock with acquire(blocking, timeout), release(), locked().
func (i *Interp) makeThreadLock(cls *object.Class) *object.Instance {
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	var mu sync.Mutex
	var isLocked int32

	doAcquire := func(blocking bool, timeout time.Duration) bool {
		if !blocking {
			if mu.TryLock() {
				atomic.StoreInt32(&isLocked, 1)
				return true
			}
			return false
		}
		if timeout < 0 {
			// Blocking indefinitely.
			mu.Lock()
			atomic.StoreInt32(&isLocked, 1)
			return true
		}
		// Blocking with timeout: poll via TryLock.
		deadline := time.Now().Add(timeout)
		for {
			if mu.TryLock() {
				atomic.StoreInt32(&isLocked, 1)
				return true
			}
			if time.Now().After(deadline) {
				return false
			}
			time.Sleep(time.Millisecond)
		}
	}

	inst.Dict.SetStr("acquire", &object.BuiltinFunc{Name: "acquire",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			blocking := true
			timeout := time.Duration(-1)

			if len(a) > 0 {
				blocking = object.Truthy(a[0])
			}
			if len(a) > 1 {
				if f, ok := toFloat64(a[1]); ok && f >= 0 {
					timeout = time.Duration(f * float64(time.Second))
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("blocking"); ok {
					blocking = object.Truthy(v)
				}
				if v, ok := kw.GetStr("timeout"); ok && v != object.None {
					if f, ok2 := toFloat64(v); ok2 && f >= 0 {
						timeout = time.Duration(f * float64(time.Second))
					}
				}
			}

			return object.BoolOf(doAcquire(blocking, timeout)), nil
		}})

	inst.Dict.SetStr("release", &object.BuiltinFunc{Name: "release",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			atomic.StoreInt32(&isLocked, 0)
			mu.Unlock()
			return object.None, nil
		}})

	inst.Dict.SetStr("locked", &object.BuiltinFunc{Name: "locked",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(atomic.LoadInt32(&isLocked) == 1), nil
		}})

	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			mu.Lock()
			atomic.StoreInt32(&isLocked, 1)
			return inst, nil
		}})

	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			atomic.StoreInt32(&isLocked, 0)
			mu.Unlock()
			return object.False, nil
		}})

	return inst
}

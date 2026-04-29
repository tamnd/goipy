package vm

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tamnd/goipy/object"
)

// goroutineID returns the current goroutine's numeric ID parsed from
// the "goroutine N [" prefix that runtime.Stack always prints first.
func goroutineID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	b := buf[:n]
	if idx := bytes.IndexByte(b, ' '); idx >= 0 {
		b = b[idx+1:]
		if idx2 := bytes.IndexByte(b, ' '); idx2 >= 0 {
			id, _ := strconv.ParseInt(string(b[:idx2]), 10, 64)
			return id
		}
	}
	return 0
}

var (
	threadIDCounter int64       // monotonically increasing ident
	threadAlive     int64       // active thread count (not counting main)
	currentThreads  sync.Map    // int64(goroutineID) → *object.Instance
	threadLocalData sync.Map    // [threadLocalKey] → object.Object
)

type threadLocalKey struct {
	inst *object.Instance
	gid  int64
}

// chCond is a channel-backed condition variable that supports timed
// waits, unlike sync.Cond. Each Wait() registers a fresh channel; the
// FIFO of channels is woken in order by Notify(n). Caller must hold
// extMu when calling Wait — Wait drops it for the duration of the
// sleep and re-acquires it on return.
type chCond struct {
	mu      sync.Mutex
	waiters []chan struct{}
}

// Wait drops extMu, waits up to timeout (negative = no timeout) for
// a Notify, then re-acquires extMu. Returns true on signal, false on
// timeout.
func (c *chCond) Wait(extMu *sync.Mutex, timeout time.Duration) bool {
	ch := make(chan struct{}, 1)
	c.mu.Lock()
	c.waiters = append(c.waiters, ch)
	c.mu.Unlock()
	extMu.Unlock()
	var ok bool
	if timeout < 0 {
		<-ch
		ok = true
	} else {
		select {
		case <-ch:
			ok = true
		case <-time.After(timeout):
			c.mu.Lock()
			for j, w := range c.waiters {
				if w == ch {
					c.waiters = append(c.waiters[:j], c.waiters[j+1:]...)
					break
				}
			}
			c.mu.Unlock()
			select {
			case <-ch:
				ok = true
			default:
				ok = false
			}
		}
	}
	extMu.Lock()
	return ok
}

// Notify wakes up to n waiters in FIFO order. n<=0 wakes all.
func (c *chCond) Notify(n int) {
	c.mu.Lock()
	if n <= 0 || n > len(c.waiters) {
		n = len(c.waiters)
	}
	for j := 0; j < n; j++ {
		select {
		case c.waiters[j] <- struct{}{}:
		default:
		}
	}
	c.waiters = c.waiters[n:]
	c.mu.Unlock()
}

// Broadcast wakes all current waiters.
func (c *chCond) Broadcast() { c.Notify(0) }

// parseTimeout converts a Python timeout argument to a time.Duration.
// None, missing, or negative values map to -1 (no timeout). Float
// seconds are converted to nanoseconds.
func parseTimeout(o object.Object) time.Duration {
	if o == nil || o == object.None {
		return -1
	}
	if f, ok := toFloat64(o); ok {
		if f < 0 {
			return -1
		}
		return time.Duration(f * float64(time.Second))
	}
	return -1
}

// buildThreading constructs the threading module backed by real goroutines
// and Go sync primitives. Each Thread.start() spawns a real goroutine;
// join() waits via sync.WaitGroup.
func (i *Interp) buildThreading() *object.Module {
	m := &object.Module{Name: "threading", Dict: object.NewDict()}

	mainGID := goroutineID()
	mainThread := i.makeThread("MainThread", mainGID)
	// Do NOT store mainThread in currentThreads — it's per-interpreter and
	// currentThreads is global, so stale entries would leak across tests.

	// Lock
	m.Dict.SetStr("Lock", &object.BuiltinFunc{Name: "Lock", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return i.makeLock(), nil
	}})

	// RLock
	m.Dict.SetStr("RLock", &object.BuiltinFunc{Name: "RLock", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return i.makeRLock(), nil
	}})

	// Thread(target=…, args=(), kwargs={}, daemon=None, name=None)
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
				switch t := v.(type) {
				case *object.Tuple:
					args = t
				case *object.List:
					args = &object.Tuple{V: t.V}
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
	m.Dict.SetStr("Semaphore", &object.BuiltinFunc{Name: "Semaphore", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		val := int64(1)
		if len(a) > 0 {
			if n, ok := toInt64(a[0]); ok {
				val = n
			}
		}
		return i.makeSemaphore(val, -1), nil
	}})

	// BoundedSemaphore(value=1)
	m.Dict.SetStr("BoundedSemaphore", &object.BuiltinFunc{Name: "BoundedSemaphore", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		val := int64(1)
		if len(a) > 0 {
			if n, ok := toInt64(a[0]); ok {
				val = n
			}
		}
		return i.makeSemaphore(val, val), nil
	}})

	// Condition(lock=None)
	m.Dict.SetStr("Condition", &object.BuiltinFunc{Name: "Condition", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return i.makeCondition(), nil
	}})

	// BrokenBarrierError — subclass of RuntimeError per CPython.
	brokenBarrierErr := &object.Class{Name: "BrokenBarrierError", Bases: []*object.Class{i.runtimeErr}, Dict: object.NewDict()}
	m.Dict.SetStr("BrokenBarrierError", brokenBarrierErr)

	// Barrier(parties, action=None, timeout=None)
	m.Dict.SetStr("Barrier", &object.BuiltinFunc{Name: "Barrier", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		parties := int64(1)
		if len(a) > 0 {
			if n, ok := toInt64(a[0]); ok {
				parties = n
			}
		}
		return i.makeBarrier(parties, brokenBarrierErr), nil
	}})

	// current_thread() / currentThread()
	// If the caller is a spawned thread, it's in currentThreads; otherwise main.
	ctFn := &object.BuiltinFunc{Name: "current_thread", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		gid := goroutineID()
		if v, ok := currentThreads.Load(gid); ok {
			return v.(*object.Instance), nil
		}
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
		return object.NewInt(1 + atomic.LoadInt64(&threadAlive)), nil
	}}
	m.Dict.SetStr("active_count", acFn)
	m.Dict.SetStr("activeCount", acFn)

	// enumerate() returns main thread plus all live spawned threads.
	m.Dict.SetStr("enumerate", &object.BuiltinFunc{Name: "enumerate", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		threads := []object.Object{mainThread}
		currentThreads.Range(func(k, v any) bool {
			threads = append(threads, v.(*object.Instance))
			return true
		})
		return &object.List{V: threads}, nil
	}})

	// local()
	m.Dict.SetStr("local", &object.BuiltinFunc{Name: "local", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return i.makeLocal(), nil
	}})

	// get_ident()
	m.Dict.SetStr("get_ident", &object.BuiltinFunc{Name: "get_ident", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(goroutineID()), nil
	}})

	// get_native_id()
	m.Dict.SetStr("get_native_id", &object.BuiltinFunc{Name: "get_native_id", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(goroutineID()), nil
	}})

	// settrace / setprofile — no-ops
	noop := &object.BuiltinFunc{Name: "noop", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}}
	m.Dict.SetStr("settrace", noop)
	m.Dict.SetStr("setprofile", noop)

	// TIMEOUT_MAX
	m.Dict.SetStr("TIMEOUT_MAX", &object.Float{V: 1e308})

	return m
}

// makeThread creates a named Thread instance (no start/join — used for main thread).
func (i *Interp) makeThread(name string, gid int64) *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	inst.Dict.SetStr("name", &object.Str{V: name})
	inst.Dict.SetStr("daemon", object.False)
	inst.Dict.SetStr("ident", object.NewInt(gid))
	inst.Dict.SetStr("native_id", object.NewInt(gid))
	return inst
}

// makeThreadObj creates a Thread instance with start/join/is_alive backed by
// a real goroutine and sync.WaitGroup.
func (i *Interp) makeThreadObj(name string, target object.Object, args *object.Tuple, kwargs *object.Dict) *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	inst.Dict.SetStr("name", &object.Str{V: name})
	inst.Dict.SetStr("daemon", object.False)
	inst.Dict.SetStr("ident", object.None)
	inst.Dict.SetStr("native_id", object.None)

	var wg sync.WaitGroup
	var alive int32 // 0=not started, 1=running, 2=done
	var started int32

	inst.Dict.SetStr("start", &object.BuiltinFunc{Name: "start", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if !atomic.CompareAndSwapInt32(&started, 0, 1) {
			return nil, object.Errorf(i.runtimeErr, "thread already started")
		}
		ident := atomic.AddInt64(&threadIDCounter, 1)
		inst.Dict.SetStr("ident", object.NewInt(ident))
		inst.Dict.SetStr("native_id", object.NewInt(ident))
		atomic.StoreInt32(&alive, 1)
		atomic.AddInt64(&threadAlive, 1)
		wg.Add(1)
		ti := i.threadCopy()
		go func() {
			gid := goroutineID()
			currentThreads.Store(gid, inst)
			defer func() {
				currentThreads.Delete(gid)
				atomic.StoreInt32(&alive, 2)
				atomic.AddInt64(&threadAlive, -1)
				wg.Done()
			}()
			if target != nil {
				ti.callObject(target, args.V, kwargs) //nolint
			}
		}()
		return object.None, nil
	}})

	inst.Dict.SetStr("join", &object.BuiltinFunc{Name: "join", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		wg.Wait()
		return object.None, nil
	}})

	inst.Dict.SetStr("is_alive", &object.BuiltinFunc{Name: "is_alive", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(atomic.LoadInt32(&alive) == 1), nil
	}})

	inst.Dict.SetStr("run", &object.BuiltinFunc{Name: "run", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if target != nil {
			_, err := i.callObject(target, args.V, kwargs)
			return object.None, err
		}
		return object.None, nil
	}})

	return inst
}

// makeLock creates a Lock backed by a real sync.Mutex.
func (i *Interp) makeLock() *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	var mu sync.Mutex
	var isLocked int32

	inst.Dict.SetStr("acquire", &object.BuiltinFunc{Name: "acquire", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		blocking := true
		if kw != nil {
			if v, ok := kw.GetStr("blocking"); ok {
				blocking = object.Truthy(v)
			}
		}
		if len(a) > 0 {
			blocking = object.Truthy(a[0])
		}
		if blocking {
			mu.Lock()
			atomic.StoreInt32(&isLocked, 1)
			return object.True, nil
		}
		// Non-blocking: TryLock
		if mu.TryLock() {
			atomic.StoreInt32(&isLocked, 1)
			return object.True, nil
		}
		return object.False, nil
	}})

	inst.Dict.SetStr("release", &object.BuiltinFunc{Name: "release", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		atomic.StoreInt32(&isLocked, 0)
		mu.Unlock()
		return object.None, nil
	}})

	inst.Dict.SetStr("locked", &object.BuiltinFunc{Name: "locked", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(atomic.LoadInt32(&isLocked) == 1), nil
	}})

	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Lock()
		atomic.StoreInt32(&isLocked, 1)
		return inst, nil
	}})

	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		atomic.StoreInt32(&isLocked, 0)
		mu.Unlock()
		return object.False, nil
	}})

	return inst
}

// rLockState holds the reentrancy bookkeeping for RLock.
type rLockState struct {
	mu    sync.Mutex
	cond  *sync.Cond
	owner int64 // goroutine ID, 0 = unlocked
	depth int
}

// makeRLock creates an RLock backed by a real reentrant mutex.
func (i *Interp) makeRLock() *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	s := &rLockState{}
	s.cond = sync.NewCond(&s.mu)

	acquire := func(blocking bool) bool {
		gid := goroutineID()
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.owner == gid {
			// Reentrant: same goroutine
			s.depth++
			return true
		}
		if blocking {
			for s.owner != 0 {
				s.cond.Wait()
			}
			s.owner = gid
			s.depth = 1
			return true
		}
		if s.owner == 0 {
			s.owner = gid
			s.depth = 1
			return true
		}
		return false
	}

	release := func() {
		gid := goroutineID()
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.owner != gid {
			return
		}
		s.depth--
		if s.depth == 0 {
			s.owner = 0
			s.cond.Signal()
		}
	}

	inst.Dict.SetStr("acquire", &object.BuiltinFunc{Name: "acquire", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		blocking := true
		if kw != nil {
			if v, ok := kw.GetStr("blocking"); ok {
				blocking = object.Truthy(v)
			}
		}
		if len(a) > 0 {
			blocking = object.Truthy(a[0])
		}
		return object.BoolOf(acquire(blocking)), nil
	}})

	inst.Dict.SetStr("release", &object.BuiltinFunc{Name: "release", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		release()
		return object.None, nil
	}})

	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		acquire(true)
		return inst, nil
	}})

	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		release()
		return object.False, nil
	}})

	return inst
}

// makeEvent creates an Event backed by chCond so wait(timeout) is
// honoured.
func (i *Interp) makeEvent() *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	var mu sync.Mutex
	var cond chCond
	flag := false

	inst.Dict.SetStr("is_set", &object.BuiltinFunc{Name: "is_set", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Lock()
		defer mu.Unlock()
		return object.BoolOf(flag), nil
	}})

	inst.Dict.SetStr("isSet", &object.BuiltinFunc{Name: "isSet", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Lock()
		defer mu.Unlock()
		return object.BoolOf(flag), nil
	}})

	inst.Dict.SetStr("set", &object.BuiltinFunc{Name: "set", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Lock()
		flag = true
		cond.Broadcast()
		mu.Unlock()
		return object.None, nil
	}})

	inst.Dict.SetStr("clear", &object.BuiltinFunc{Name: "clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Lock()
		flag = false
		mu.Unlock()
		return object.None, nil
	}})

	inst.Dict.SetStr("wait", &object.BuiltinFunc{Name: "wait", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		var timeoutArg object.Object
		if len(a) > 0 {
			timeoutArg = a[0]
		} else if kw != nil {
			if v, ok := kw.GetStr("timeout"); ok {
				timeoutArg = v
			}
		}
		timeout := parseTimeout(timeoutArg)
		mu.Lock()
		defer mu.Unlock()
		if flag {
			return object.True, nil
		}
		if timeout < 0 {
			for !flag {
				cond.Wait(&mu, -1)
			}
			return object.True, nil
		}
		deadline := time.Now().Add(timeout)
		for !flag {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return object.BoolOf(flag), nil
			}
			cond.Wait(&mu, remaining)
		}
		return object.True, nil
	}})

	return inst
}

// makeSemaphore creates a Semaphore (maxVal < 0 means unbounded).
func (i *Interp) makeSemaphore(value, maxVal int64) *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	var mu sync.Mutex
	var cond chCond
	count := value

	inst.Dict.SetStr("acquire", &object.BuiltinFunc{Name: "acquire", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		blocking := true
		var timeoutArg object.Object
		if kw != nil {
			if v, ok := kw.GetStr("blocking"); ok {
				blocking = object.Truthy(v)
			}
			if v, ok := kw.GetStr("timeout"); ok {
				timeoutArg = v
			}
		}
		if len(a) > 0 {
			blocking = object.Truthy(a[0])
		}
		if len(a) > 1 {
			timeoutArg = a[1]
		}
		timeout := parseTimeout(timeoutArg)
		mu.Lock()
		defer mu.Unlock()
		if !blocking {
			if count > 0 {
				count--
				return object.True, nil
			}
			return object.False, nil
		}
		if timeout < 0 {
			for count <= 0 {
				cond.Wait(&mu, -1)
			}
			count--
			return object.True, nil
		}
		deadline := time.Now().Add(timeout)
		for count <= 0 {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return object.False, nil
			}
			cond.Wait(&mu, remaining)
		}
		count--
		return object.True, nil
	}})

	inst.Dict.SetStr("release", &object.BuiltinFunc{Name: "release", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n := int64(1)
		if len(a) > 0 {
			if v, ok := toInt64(a[0]); ok {
				n = v
			}
		}
		mu.Lock()
		if maxVal >= 0 && count+n > maxVal {
			mu.Unlock()
			return nil, object.Errorf(i.valueErr, "semaphore released too many times")
		}
		count += n
		cond.Notify(int(n))
		mu.Unlock()
		return object.None, nil
	}})

	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Lock()
		for count <= 0 {
			cond.Wait(&mu, -1)
		}
		count--
		mu.Unlock()
		return inst, nil
	}})

	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Lock()
		count++
		cond.Notify(1)
		mu.Unlock()
		return object.False, nil
	}})

	return inst
}

// makeCondition creates a Condition variable backed by chCond so
// wait(timeout) and wait_for(predicate, timeout) are honoured.
func (i *Interp) makeCondition() *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	var mu sync.Mutex
	var cond chCond

	inst.Dict.SetStr("acquire", &object.BuiltinFunc{Name: "acquire", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Lock()
		return object.True, nil
	}})

	inst.Dict.SetStr("release", &object.BuiltinFunc{Name: "release", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Unlock()
		return object.None, nil
	}})

	inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Lock()
		return inst, nil
	}})

	inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Unlock()
		return object.False, nil
	}})

	inst.Dict.SetStr("wait", &object.BuiltinFunc{Name: "wait", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		var timeoutArg object.Object
		if len(a) > 0 {
			timeoutArg = a[0]
		} else if kw != nil {
			if v, ok := kw.GetStr("timeout"); ok {
				timeoutArg = v
			}
		}
		timeout := parseTimeout(timeoutArg)
		ok := cond.Wait(&mu, timeout)
		return object.BoolOf(ok), nil
	}})

	inst.Dict.SetStr("notify", &object.BuiltinFunc{Name: "notify", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n := 1
		if len(a) > 0 {
			if v, ok := toInt64(a[0]); ok {
				n = int(v)
			}
		}
		cond.Notify(n)
		return object.None, nil
	}})

	inst.Dict.SetStr("notify_all", &object.BuiltinFunc{Name: "notify_all", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		cond.Broadcast()
		return object.None, nil
	}})

	inst.Dict.SetStr("notifyAll", &object.BuiltinFunc{Name: "notifyAll", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		cond.Broadcast()
		return object.None, nil
	}})

	inst.Dict.SetStr("wait_for", &object.BuiltinFunc{Name: "wait_for", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		caller := i
		if ii != nil {
			if ci, ok := ii.(*Interp); ok {
				caller = ci
			}
		}
		if len(a) == 0 {
			return object.True, nil
		}
		var timeoutArg object.Object
		if len(a) > 1 {
			timeoutArg = a[1]
		} else if kw != nil {
			if v, ok := kw.GetStr("timeout"); ok {
				timeoutArg = v
			}
		}
		timeout := parseTimeout(timeoutArg)
		var deadline time.Time
		if timeout >= 0 {
			deadline = time.Now().Add(timeout)
		}
		var last object.Object = object.False
		for {
			r, err := caller.callObject(a[0], nil, nil)
			if err != nil {
				return nil, err
			}
			last = r
			if object.Truthy(r) {
				return r, nil
			}
			if timeout < 0 {
				cond.Wait(&mu, -1)
				continue
			}
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return last, nil
			}
			cond.Wait(&mu, remaining)
		}
	}})

	return inst
}

// makeBarrier creates a Barrier for parties goroutines, backed by
// chCond so wait(timeout) is honoured.
func (i *Interp) makeBarrier(parties int64, brokenBarrierErr *object.Class) *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}
	var mu sync.Mutex
	var cond chCond
	var waiting int64
	var broken bool
	var generation int64

	inst.Dict.SetStr("parties", object.NewInt(parties))
	inst.Dict.SetStr("n_waiting", object.NewInt(0))
	inst.Dict.SetStr("broken", object.False)

	inst.Dict.SetStr("wait", &object.BuiltinFunc{Name: "wait", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		var timeoutArg object.Object
		if len(a) > 0 {
			timeoutArg = a[0]
		} else if kw != nil {
			if v, ok := kw.GetStr("timeout"); ok {
				timeoutArg = v
			}
		}
		timeout := parseTimeout(timeoutArg)

		mu.Lock()
		defer mu.Unlock()
		if broken {
			return nil, object.Errorf(brokenBarrierErr, "BrokenBarrierError")
		}
		myIndex := waiting
		waiting++
		inst.Dict.SetStr("n_waiting", object.NewInt(waiting))
		myGen := generation
		if waiting == parties {
			generation++
			waiting = 0
			inst.Dict.SetStr("n_waiting", object.NewInt(0))
			cond.Broadcast()
			return object.NewInt(myIndex), nil
		}
		var deadline time.Time
		if timeout >= 0 {
			deadline = time.Now().Add(timeout)
		}
		for !broken && generation == myGen {
			if timeout < 0 {
				cond.Wait(&mu, -1)
				continue
			}
			remaining := time.Until(deadline)
			if remaining <= 0 {
				broken = true
				inst.Dict.SetStr("broken", object.True)
				cond.Broadcast()
				return nil, object.Errorf(brokenBarrierErr, "BrokenBarrierError")
			}
			cond.Wait(&mu, remaining)
		}
		if broken {
			return nil, object.Errorf(brokenBarrierErr, "BrokenBarrierError")
		}
		return object.NewInt(myIndex), nil
	}})

	inst.Dict.SetStr("reset", &object.BuiltinFunc{Name: "reset", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Lock()
		broken = false
		waiting = 0
		generation++
		inst.Dict.SetStr("broken", object.False)
		inst.Dict.SetStr("n_waiting", object.NewInt(0))
		cond.Broadcast()
		mu.Unlock()
		return object.None, nil
	}})

	inst.Dict.SetStr("abort", &object.BuiltinFunc{Name: "abort", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		mu.Lock()
		broken = true
		inst.Dict.SetStr("broken", object.True)
		cond.Broadcast()
		mu.Unlock()
		return object.None, nil
	}})

	return inst
}

// makeLocal creates a threading.local() object backed by sync.Map.
// Each goroutine sees its own independent attribute namespace.
func (i *Interp) makeLocal() *object.Instance {
	inst := &object.Instance{Dict: object.NewDict()}

	getDict := func() *object.Dict {
		gid := goroutineID()
		key := threadLocalKey{inst: inst, gid: gid}
		v, _ := threadLocalData.LoadOrStore(key, object.NewDict())
		return v.(*object.Dict)
	}

	cls := &object.Class{Name: "local", Dict: object.NewDict()}

	cls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		d := getDict()
		if v, ok := d.GetStr(name); ok {
			return v, nil
		}
		return nil, object.Errorf(i.attrErr, "'local' object has no attribute '%s'", name)
	}})

	cls.Dict.SetStr("__setattr__", &object.BuiltinFunc{Name: "__setattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		getDict().SetStr(name, a[2])
		return object.None, nil
	}})

	inst.Class = cls
	return inst
}

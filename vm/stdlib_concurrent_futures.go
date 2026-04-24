package vm

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/tamnd/goipy/object"
)

// ─── Future state constants ──────────────────────────────────────────────────

const (
	cfStatePending   int32 = 0
	cfStateRunning   int32 = 1
	cfStateCancelled int32 = 2
	cfStateFinished  int32 = 3
)

// cfFutureState holds the mutable internal state of a Future.
type cfFutureState struct {
	state     atomic.Int32
	mu        sync.Mutex
	resultVal object.Object
	resultErr error
	done      chan struct{} // closed when CANCELLED or FINISHED
	callbacks []cfDoneCallback
}

type cfDoneCallback struct {
	fn   object.Object
	inst *object.Instance
}

func newCFFutureState() *cfFutureState {
	return &cfFutureState{done: make(chan struct{})}
}

// cfFutureRegistry maps *object.Instance → *cfFutureState.
var cfFutureRegistry sync.Map

// cfJob is sent from Executor.submit to a worker goroutine.
type cfJob struct {
	f    *cfFutureState
	inst *object.Instance
	fn   object.Object
	args []object.Object
	kw   *object.Dict
}

// cfExecutorState holds pool bookkeeping.
type cfExecutorState struct {
	jobs      chan cfJob
	wg        sync.WaitGroup
	shutdown  atomic.Bool
	mu        sync.Mutex // protect submit vs shutdown race
	closeOnce sync.Once
}

func (ps *cfExecutorState) trySubmit(job cfJob) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.shutdown.Load() {
		return false
	}
	ps.jobs <- job
	return true
}

func (ps *cfExecutorState) doClose() {
	ps.closeOnce.Do(func() { close(ps.jobs) })
}

// buildConcurrentFutures constructs the concurrent.futures module.
func (i *Interp) buildConcurrentFutures() *object.Module {
	m := &object.Module{Name: "concurrent.futures", Dict: object.NewDict()}

	// ─── Exception classes ─────────────────────────────────────────────────
	cancelledErr := &object.Class{Name: "CancelledError", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	cfTimeoutErr := &object.Class{Name: "TimeoutError", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	brokenExec   := &object.Class{Name: "BrokenExecutor", Bases: []*object.Class{i.runtimeErr}, Dict: object.NewDict()}
	invalidState := &object.Class{Name: "InvalidStateError", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}

	m.Dict.SetStr("CancelledError", cancelledErr)
	m.Dict.SetStr("TimeoutError", cfTimeoutErr)
	m.Dict.SetStr("BrokenExecutor", brokenExec)
	m.Dict.SetStr("InvalidStateError", invalidState)

	// ─── Constants ─────────────────────────────────────────────────────────
	m.Dict.SetStr("FIRST_COMPLETED", &object.Str{V: "FIRST_COMPLETED"})
	m.Dict.SetStr("FIRST_EXCEPTION", &object.Str{V: "FIRST_EXCEPTION"})
	m.Dict.SetStr("ALL_COMPLETED", &object.Str{V: "ALL_COMPLETED"})

	// ─── ThreadPoolExecutor ────────────────────────────────────────────────
	m.Dict.SetStr("ThreadPoolExecutor", &object.BuiltinFunc{Name: "ThreadPoolExecutor",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			maxWorkers, initializer, initArgs := cfParseExecutorArgs(a, kw)
			return i.makeCFExecutor("ThreadPoolExecutor", maxWorkers, initializer, initArgs,
				cancelledErr, cfTimeoutErr, invalidState), nil
		}})

	// ─── ProcessPoolExecutor ───────────────────────────────────────────────
	m.Dict.SetStr("ProcessPoolExecutor", &object.BuiltinFunc{Name: "ProcessPoolExecutor",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			maxWorkers, initializer, initArgs := cfParseExecutorArgs(a, kw)
			return i.makeCFExecutor("ProcessPoolExecutor", maxWorkers, initializer, initArgs,
				cancelledErr, cfTimeoutErr, invalidState), nil
		}})

	// ─── wait(fs, timeout=None, return_when=ALL_COMPLETED) ─────────────────
	m.Dict.SetStr("wait", &object.BuiltinFunc{Name: "wait",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "wait() requires futures argument")
			}
			futsObj := a[0]
			timeout := cfParseTimeout(a[1:], kw, "timeout")
			returnWhen := "ALL_COMPLETED"
			if kw != nil {
				if v, ok := kw.GetStr("return_when"); ok {
					if s, ok2 := v.(*object.Str); ok2 {
						returnWhen = s.V
					}
				}
			}
			insts, states, err := i.cfCollectFutures(futsObj)
			if err != nil {
				return nil, err
			}
			n := len(states)
			if n == 0 {
				empty := &object.List{V: []object.Object{}}
				return &object.Tuple{V: []object.Object{empty, empty}}, nil
			}

			completionCh := make(chan int, n)
			for k, f := range states {
				capK := k
				capF := f
				go func() {
					<-capF.done
					completionCh <- capK
				}()
			}

			doneSeen := make([]bool, n)
			doneCnt := 0
			for k, f := range states {
				st := f.state.Load()
				if st == cfStateCancelled || st == cfStateFinished {
					doneSeen[k] = true
					doneCnt++
				}
			}

			var deadlineCh <-chan time.Time
			if timeout >= 0 {
				deadlineCh = time.After(timeout)
			}

			timedOut := false
		waitLoop:
			for doneCnt < n {
				switch returnWhen {
				case "FIRST_COMPLETED":
					if doneCnt > 0 {
						break waitLoop
					}
				case "FIRST_EXCEPTION":
					for k, f := range states {
						if doneSeen[k] && f.state.Load() == cfStateFinished {
							f.mu.Lock()
							hasErr := f.resultErr != nil
							f.mu.Unlock()
							if hasErr {
								break waitLoop
							}
						}
					}
					if doneCnt == n {
						break waitLoop
					}
				default: // ALL_COMPLETED
					if doneCnt == n {
						break waitLoop
					}
				}

				if deadlineCh != nil {
					select {
					case k := <-completionCh:
						doneSeen[k] = true
						doneCnt++
					case <-deadlineCh:
						timedOut = true
						break waitLoop
					}
				} else {
					k := <-completionCh
					doneSeen[k] = true
					doneCnt++
				}
				_ = timedOut
			}

			var doneList, notDoneList []object.Object
			for k, inst := range insts {
				if doneSeen[k] {
					doneList = append(doneList, inst)
				} else {
					notDoneList = append(notDoneList, inst)
				}
			}
			if doneList == nil {
				doneList = []object.Object{}
			}
			if notDoneList == nil {
				notDoneList = []object.Object{}
			}
			return &object.Tuple{V: []object.Object{
				&object.List{V: doneList},
				&object.List{V: notDoneList},
			}}, nil
		}})

	// ─── as_completed(fs, timeout=None) ───────────────────────────────────
	m.Dict.SetStr("as_completed", &object.BuiltinFunc{Name: "as_completed",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "as_completed() requires futures argument")
			}
			insts, states, err := i.cfCollectFutures(a[0])
			if err != nil {
				return nil, err
			}
			n := len(states)
			ch := make(chan *object.Instance, n+1)
			for k := range states {
				capInst := insts[k]
				capF := states[k]
				go func() {
					<-capF.done
					ch <- capInst
				}()
			}
			remaining := n
			return &object.Iter{Next: func() (object.Object, bool, error) {
				if remaining == 0 {
					return nil, false, nil
				}
				remaining--
				inst := <-ch
				return inst, true, nil
			}}, nil
		}})

	return m
}

// makeCFExecutor builds a ThreadPoolExecutor or ProcessPoolExecutor instance.
func (i *Interp) makeCFExecutor(
	name string,
	maxWorkers int,
	initializer object.Object,
	initArgs *object.Tuple,
	cancelledErr, timeoutErr, invalidStateErr *object.Class,
) *object.Instance {
	cls := &object.Class{Name: name, Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	ps := &cfExecutorState{
		jobs: make(chan cfJob, maxWorkers*4),
	}

	for w := 0; w < maxWorkers; w++ {
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
			for job := range ps.jobs {
				if !job.f.state.CompareAndSwap(cfStatePending, cfStateRunning) {
					continue // future was cancelled
				}
				val, err := wi.callObject(job.fn, job.args, job.kw)
				job.f.mu.Lock()
				job.f.resultVal = val
				job.f.resultErr = err
				job.f.state.Store(cfStateFinished)
				cbs := append([]cfDoneCallback(nil), job.f.callbacks...)
				job.f.callbacks = nil
				job.f.mu.Unlock()
				close(job.f.done)
				for _, cb := range cbs {
					wi.callObject(cb.fn, []object.Object{cb.inst}, nil) //nolint
				}
			}
		}()
	}

	// submit(fn, /, *args, **kwargs)
	cls.Dict.SetStr("submit", &object.BuiltinFunc{Name: "submit",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "submit() requires a callable")
			}
			fn := a[0]
			fnArgs := append([]object.Object(nil), a[1:]...)
			f, finst := i.makeCFFuture(cancelledErr, timeoutErr, invalidStateErr)
			if !ps.trySubmit(cfJob{f: f, inst: finst, fn: fn, args: fnArgs, kw: kw}) {
				return nil, object.Errorf(i.runtimeErr, "cannot schedule new futures after shutdown")
			}
			return finst, nil
		}})

	// map(fn, *iterables, timeout=None, chunksize=1)
	cls.Dict.SetStr("map", &object.BuiltinFunc{Name: "map",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "map() requires a callable")
			}
			fn := a[0]
			timeout := cfParseTimeout(nil, kw, "timeout")

			// Collect items from all iterables; zip if multiple.
			var zipped [][]object.Object
			for _, iterable := range a[1:] {
				sl, err := i.iterToSlice(iterable)
				if err != nil {
					return nil, err
				}
				zipped = append(zipped, sl)
			}
			if len(zipped) == 0 {
				return &object.Iter{Next: func() (object.Object, bool, error) {
					return nil, false, nil
				}}, nil
			}
			nItems := len(zipped[0])
			for _, col := range zipped[1:] {
				if len(col) < nItems {
					nItems = len(col)
				}
			}

			futs := make([]*cfFutureState, nItems)
			finsts := make([]*object.Instance, nItems)
			for k := 0; k < nItems; k++ {
				callArgs := make([]object.Object, len(zipped))
				for col, items := range zipped {
					callArgs[col] = items[k]
				}
				f, finst := i.makeCFFuture(cancelledErr, timeoutErr, invalidStateErr)
				futs[k] = f
				finsts[k] = finst
				ps.trySubmit(cfJob{f: f, inst: finst, fn: fn, args: callArgs})
			}

			idx := 0
			return &object.Iter{Next: func() (object.Object, bool, error) {
				if idx >= len(futs) {
					return nil, false, nil
				}
				f := futs[idx]
				idx++
				if timeout >= 0 {
					select {
					case <-f.done:
					case <-time.After(timeout):
						return nil, false, object.Errorf(timeoutErr, "futures did not complete within timeout")
					}
				} else {
					<-f.done
				}
				if f.state.Load() == cfStateCancelled {
					return nil, false, object.Errorf(cancelledErr, "Future was cancelled")
				}
				f.mu.Lock()
				val, err := f.resultVal, f.resultErr
				f.mu.Unlock()
				if err != nil {
					return nil, false, err
				}
				return val, true, nil
			}}, nil
		}})

	// shutdown(wait=True, *, cancel_futures=False)
	cls.Dict.SetStr("shutdown", &object.BuiltinFunc{Name: "shutdown",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			waitArg := true
			cancelFutures := false
			if len(a) > 0 {
				if b, ok := a[0].(*object.Bool); ok {
					waitArg = b.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("wait"); ok {
					if b, ok2 := v.(*object.Bool); ok2 {
						waitArg = b.V
					}
				}
				if v, ok := kw.GetStr("cancel_futures"); ok {
					if b, ok2 := v.(*object.Bool); ok2 {
						cancelFutures = b.V
					}
				}
			}
			ps.mu.Lock()
			ps.shutdown.Store(true)
			ps.mu.Unlock()
			if cancelFutures {
			drain:
				for {
					select {
					case job := <-ps.jobs:
						i.cfCancelFuture(job.f, job.inst, cancelledErr)
					default:
						break drain
					}
				}
			}
			ps.doClose()
			if waitArg {
				ps.wg.Wait()
			}
			return object.None, nil
		}})

	// __enter__
	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})

	// __exit__ — calls shutdown(wait=True)
	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ps.mu.Lock()
			ps.shutdown.Store(true)
			ps.mu.Unlock()
			ps.doClose()
			ps.wg.Wait()
			return object.False, nil
		}})

	return inst
}

// makeCFFuture creates a new Future instance and registers it.
func (i *Interp) makeCFFuture(
	cancelledErr, timeoutErr, invalidStateErr *object.Class,
) (*cfFutureState, *object.Instance) {
	f := newCFFutureState()
	cls := &object.Class{Name: "Future", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	cfFutureRegistry.Store(inst, f)

	cls.Dict.SetStr("cancel", &object.BuiltinFunc{Name: "cancel",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(i.cfCancelFuture(f, inst, cancelledErr)), nil
		}})

	cls.Dict.SetStr("cancelled", &object.BuiltinFunc{Name: "cancelled",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(f.state.Load() == cfStateCancelled), nil
		}})

	cls.Dict.SetStr("running", &object.BuiltinFunc{Name: "running",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(f.state.Load() == cfStateRunning), nil
		}})

	cls.Dict.SetStr("done", &object.BuiltinFunc{Name: "done",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			st := f.state.Load()
			return object.BoolOf(st == cfStateCancelled || st == cfStateFinished), nil
		}})

	cls.Dict.SetStr("result", &object.BuiltinFunc{Name: "result",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			timeout := cfParseTimeout(a, kw, "timeout")
			if !cfWaitDone(f, timeout) {
				return nil, object.Errorf(timeoutErr, "future did not complete within timeout")
			}
			if f.state.Load() == cfStateCancelled {
				return nil, object.Errorf(cancelledErr, "Future was cancelled")
			}
			f.mu.Lock()
			val, err := f.resultVal, f.resultErr
			f.mu.Unlock()
			return val, err
		}})

	cls.Dict.SetStr("exception", &object.BuiltinFunc{Name: "exception",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			timeout := cfParseTimeout(a, kw, "timeout")
			if !cfWaitDone(f, timeout) {
				return nil, object.Errorf(timeoutErr, "future did not complete within timeout")
			}
			if f.state.Load() == cfStateCancelled {
				return nil, object.Errorf(cancelledErr, "Future was cancelled")
			}
			f.mu.Lock()
			err := f.resultErr
			f.mu.Unlock()
			if err == nil {
				return object.None, nil
			}
			if exc, ok := err.(*object.Exception); ok {
				return exc, nil
			}
			return &object.Str{V: err.Error()}, nil
		}})

	cls.Dict.SetStr("add_done_callback", &object.BuiltinFunc{Name: "add_done_callback",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "add_done_callback() requires a callable")
			}
			fn := a[0]
			f.mu.Lock()
			st := f.state.Load()
			isDone := st == cfStateCancelled || st == cfStateFinished
			if !isDone {
				f.callbacks = append(f.callbacks, cfDoneCallback{fn: fn, inst: inst})
			}
			f.mu.Unlock()
			if isDone {
				i.callObject(fn, []object.Object{inst}, nil) //nolint
			}
			return object.None, nil
		}})

	return f, inst
}

// cfCancelFuture transitions future from PENDING to CANCELLED.
func (i *Interp) cfCancelFuture(f *cfFutureState, inst *object.Instance, cancelledErr *object.Class) bool {
	if !f.state.CompareAndSwap(cfStatePending, cfStateCancelled) {
		return false
	}
	f.mu.Lock()
	cbs := append([]cfDoneCallback(nil), f.callbacks...)
	f.callbacks = nil
	f.mu.Unlock()
	close(f.done)
	for _, cb := range cbs {
		i.callObject(cb.fn, []object.Object{cb.inst}, nil) //nolint
	}
	return true
}

// cfWaitDone blocks until the future's done channel closes. Returns false on timeout.
func cfWaitDone(f *cfFutureState, timeout time.Duration) bool {
	if timeout < 0 {
		<-f.done
		return true
	}
	select {
	case <-f.done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// cfParseTimeout extracts a timeout value from positional args or keyword args.
// Returns -1 (no timeout) if timeout is None or not present.
func cfParseTimeout(a []object.Object, kw *object.Dict, kwName string) time.Duration {
	var secs float64 = -1
	if len(a) > 0 {
		switch t := a[0].(type) {
		case *object.Float:
			secs = t.V
		case *object.Int:
			if t.IsInt64() {
				secs = float64(t.Int64())
			}
		}
	}
	if kw != nil {
		if v, ok := kw.GetStr(kwName); ok {
			switch t := v.(type) {
			case *object.Float:
				secs = t.V
			case *object.Int:
				if t.IsInt64() {
					secs = float64(t.Int64())
				}
			}
		}
	}
	if secs < 0 {
		return -1
	}
	return time.Duration(secs * float64(time.Second))
}

// cfParseExecutorArgs extracts max_workers, initializer, initargs from constructor args.
func cfParseExecutorArgs(a []object.Object, kw *object.Dict) (int, object.Object, *object.Tuple) {
	maxWorkers := 4
	var initializer object.Object
	var initArgs *object.Tuple

	if len(a) > 0 {
		if _, isNone := a[0].(*object.NoneType); !isNone {
			if n, ok := toInt64(a[0]); ok && n > 0 {
				maxWorkers = int(n)
			}
		}
	}
	if kw != nil {
		if v, ok := kw.GetStr("max_workers"); ok {
			if _, isNone := v.(*object.NoneType); !isNone {
				if n, ok2 := toInt64(v); ok2 && n > 0 {
					maxWorkers = int(n)
				}
			}
		}
		if v, ok := kw.GetStr("initializer"); ok {
			initializer = v
		}
		if v, ok := kw.GetStr("initargs"); ok {
			switch t := v.(type) {
			case *object.Tuple:
				initArgs = t
			case *object.List:
				initArgs = &object.Tuple{V: t.V}
			}
		}
	}
	return maxWorkers, initializer, initArgs
}

// cfCollectFutures extracts Instance pointers and their cfFutureState from an iterable.
func (i *Interp) cfCollectFutures(obj object.Object) ([]*object.Instance, []*cfFutureState, error) {
	var items []object.Object
	switch t := obj.(type) {
	case *object.List:
		items = t.V
	case *object.Tuple:
		items = t.V
	default:
		return nil, nil, object.Errorf(i.typeErr, "expected list of Future instances")
	}
	insts := make([]*object.Instance, 0, len(items))
	states := make([]*cfFutureState, 0, len(items))
	for _, item := range items {
		inst, ok := item.(*object.Instance)
		if !ok {
			return nil, nil, object.Errorf(i.typeErr, "expected Future, got %s", object.TypeName(item))
		}
		v, ok2 := cfFutureRegistry.Load(inst)
		if !ok2 {
			return nil, nil, object.Errorf(i.typeErr, "object is not a Future")
		}
		insts = append(insts, inst)
		states = append(states, v.(*cfFutureState))
	}
	return insts, states, nil
}

package vm

import (
	"container/heap"
	"sync"
	"time"

	"github.com/tamnd/goipy/object"
)

// ─── queue state ─────────────────────────────────────────────────────────────

type qState struct {
	mu         sync.Mutex
	notEmpty   *sync.Cond
	notFull    *sync.Cond
	allDone    *sync.Cond
	items      []object.Object
	maxsize    int
	unfinished int
	shutdown   bool
	shutImm    bool // immediate shutdown
}

func newQState(maxsize int) *qState {
	qs := &qState{maxsize: maxsize}
	qs.notEmpty = sync.NewCond(&qs.mu)
	qs.notFull = sync.NewCond(&qs.mu)
	qs.allDone = sync.NewCond(&qs.mu)
	return qs
}

func (qs *qState) isFull() bool {
	return qs.maxsize > 0 && len(qs.items) >= qs.maxsize
}

// ─── priority queue heap ─────────────────────────────────────────────────────

type pqHeap struct {
	items []object.Object
	less  func(a, b object.Object) bool
}

func (h pqHeap) Len() int            { return len(h.items) }
func (h pqHeap) Less(a, b int) bool  { return h.less(h.items[a], h.items[b]) }
func (h pqHeap) Swap(a, b int)       { h.items[a], h.items[b] = h.items[b], h.items[a] }
func (h *pqHeap) Push(x any)         { h.items = append(h.items, x.(object.Object)) }
func (h *pqHeap) Pop() any {
	old := h.items
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	h.items = old[:n-1]
	return x
}

// ─── buildQueue ──────────────────────────────────────────────────────────────

func (i *Interp) buildQueue() *object.Module {
	m := &object.Module{Name: "queue", Dict: object.NewDict()}

	emptyErr := &object.Class{Name: "Empty", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	fullErr := &object.Class{Name: "Full", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	shutdownErr := &object.Class{Name: "ShutDown", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}

	m.Dict.SetStr("Empty", emptyErr)
	m.Dict.SetStr("Full", fullErr)
	m.Dict.SetStr("ShutDown", shutdownErr)

	// ─── Queue ────────────────────────────────────────────────────────────
	m.Dict.SetStr("Queue", &object.BuiltinFunc{Name: "Queue",
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
			qs := newQState(maxsize)
			return i.qFifoInstance(qs, emptyErr, fullErr, shutdownErr), nil
		}})

	// ─── LifoQueue ────────────────────────────────────────────────────────
	m.Dict.SetStr("LifoQueue", &object.BuiltinFunc{Name: "LifoQueue",
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
			qs := newQState(maxsize)
			return i.qLifoInstance(qs, emptyErr, fullErr, shutdownErr), nil
		}})

	// ─── PriorityQueue ────────────────────────────────────────────────────
	m.Dict.SetStr("PriorityQueue", &object.BuiltinFunc{Name: "PriorityQueue",
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
			return i.qPriorityInstance(maxsize, emptyErr, fullErr, shutdownErr), nil
		}})

	// ─── SimpleQueue ──────────────────────────────────────────────────────
	m.Dict.SetStr("SimpleQueue", &object.BuiltinFunc{Name: "SimpleQueue",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return i.qSimpleInstance(emptyErr), nil
		}})

	return m
}

// ─── shared put/get/task_done/join/shutdown helpers ───────────────────────────

// qWaitTimeout waits on cond until pred() is true or timeout expires.
// Returns true if pred() became true, false on timeout.
// Must be called with qs.mu held; releases and re-acquires as needed.
func qWaitTimeout(cond *sync.Cond, pred func() bool, timeout time.Duration) bool {
	if pred() {
		return true
	}
	if timeout <= 0 {
		return false
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-time.After(timeout):
			cond.Broadcast()
		case <-done:
		}
	}()
	deadline := time.Now().Add(timeout)
	for !pred() {
		if time.Now().After(deadline) {
			close(done)
			return false
		}
		cond.Wait()
	}
	close(done)
	return true
}

// qPut implements the put() logic for FIFO/LIFO queues.
func (i *Interp) qPut(qs *qState, item object.Object, block bool, timeout time.Duration, fullErr, shutdownErr *object.Class) error {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	if qs.shutdown {
		return object.Errorf(shutdownErr, "Queue.put() on a shut-down queue")
	}

	if qs.isFull() {
		if !block {
			return object.Errorf(fullErr, "Queue full")
		}
		if timeout > 0 {
			ok := qWaitTimeout(qs.notFull, func() bool { return !qs.isFull() || qs.shutdown }, timeout)
			if qs.shutdown {
				return object.Errorf(shutdownErr, "Queue.put() on a shut-down queue")
			}
			if !ok {
				return object.Errorf(fullErr, "Queue.put() timed out")
			}
		} else {
			for qs.isFull() && !qs.shutdown {
				qs.notFull.Wait()
			}
			if qs.shutdown {
				return object.Errorf(shutdownErr, "Queue.put() on a shut-down queue")
			}
		}
	}

	qs.items = append(qs.items, item)
	qs.unfinished++
	qs.notEmpty.Signal()
	return nil
}

// qGetFifo pops the front item (FIFO).
func (i *Interp) qGetFifo(qs *qState, block bool, timeout time.Duration, emptyErr, shutdownErr *object.Class) (object.Object, error) {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	if qs.shutImm {
		return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down queue")
	}

	if len(qs.items) == 0 {
		if qs.shutdown {
			return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down empty queue")
		}
		if !block {
			return nil, object.Errorf(emptyErr, "Queue empty")
		}
		if timeout > 0 {
			ok := qWaitTimeout(qs.notEmpty, func() bool { return len(qs.items) > 0 || qs.shutdown || qs.shutImm }, timeout)
			if qs.shutImm {
				return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down queue")
			}
			if qs.shutdown && len(qs.items) == 0 {
				return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down empty queue")
			}
			if !ok {
				return nil, object.Errorf(emptyErr, "Queue.get() timed out")
			}
		} else {
			for len(qs.items) == 0 && !qs.shutdown && !qs.shutImm {
				qs.notEmpty.Wait()
			}
			if qs.shutImm {
				return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down queue")
			}
			if qs.shutdown && len(qs.items) == 0 {
				return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down empty queue")
			}
		}
	}

	item := qs.items[0]
	qs.items = qs.items[1:]
	qs.notFull.Signal()
	return item, nil
}

// qGetLifo pops the back item (LIFO).
func (i *Interp) qGetLifo(qs *qState, block bool, timeout time.Duration, emptyErr, shutdownErr *object.Class) (object.Object, error) {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	if qs.shutImm {
		return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down queue")
	}

	if len(qs.items) == 0 {
		if qs.shutdown {
			return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down empty queue")
		}
		if !block {
			return nil, object.Errorf(emptyErr, "Queue empty")
		}
		if timeout > 0 {
			ok := qWaitTimeout(qs.notEmpty, func() bool { return len(qs.items) > 0 || qs.shutdown || qs.shutImm }, timeout)
			if qs.shutImm {
				return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down queue")
			}
			if qs.shutdown && len(qs.items) == 0 {
				return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down empty queue")
			}
			if !ok {
				return nil, object.Errorf(emptyErr, "Queue.get() timed out")
			}
		} else {
			for len(qs.items) == 0 && !qs.shutdown && !qs.shutImm {
				qs.notEmpty.Wait()
			}
			if qs.shutImm {
				return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down queue")
			}
			if qs.shutdown && len(qs.items) == 0 {
				return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down empty queue")
			}
		}
	}

	n := len(qs.items)
	item := qs.items[n-1]
	qs.items = qs.items[:n-1]
	qs.notFull.Signal()
	return item, nil
}

func (i *Interp) qTaskDone(qs *qState) error {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	qs.unfinished--
	if qs.unfinished < 0 {
		qs.unfinished = 0
		return object.Errorf(i.valueErr, "task_done() called too many times")
	}
	if qs.unfinished == 0 {
		qs.allDone.Broadcast()
	}
	return nil
}

func (i *Interp) qJoin(qs *qState) {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	for qs.unfinished > 0 {
		qs.allDone.Wait()
	}
}

func (i *Interp) qShutdown(qs *qState, immediate bool) {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	qs.shutdown = true
	if immediate {
		qs.shutImm = true
		qs.items = nil
		qs.unfinished = 0
		qs.allDone.Broadcast()
	}
	qs.notFull.Broadcast()
	qs.notEmpty.Broadcast()
}

// parsePutGetArgs parses block/timeout from positional+kw args.
// Positional: (item, block=True, timeout=None) for put;
//             (block=True, timeout=None) for get.
// Returns block bool and timeout duration (-1 = no timeout).
func parsePutArgs(a []object.Object, kw *object.Dict) (item object.Object, block bool, timeout time.Duration) {
	block = true
	timeout = -1
	if len(a) > 0 {
		item = a[0]
	}
	if len(a) > 1 {
		if b, ok := a[1].(*object.Bool); ok {
			block = b.V
		}
	}
	if len(a) > 2 {
		if f, ok := toFloat64(a[2]); ok && f >= 0 {
			timeout = time.Duration(f * float64(time.Second))
		}
	}
	if kw != nil {
		if v, ok := kw.GetStr("block"); ok {
			if b, ok2 := v.(*object.Bool); ok2 {
				block = b.V
			}
		}
		if v, ok := kw.GetStr("timeout"); ok && v != object.None {
			if f, ok2 := toFloat64(v); ok2 && f >= 0 {
				timeout = time.Duration(f * float64(time.Second))
			}
		}
	}
	return
}

func parseGetArgs(a []object.Object, kw *object.Dict) (block bool, timeout time.Duration) {
	block = true
	timeout = -1
	if len(a) > 0 {
		if b, ok := a[0].(*object.Bool); ok {
			block = b.V
		}
	}
	if len(a) > 1 {
		if f, ok := toFloat64(a[1]); ok && f >= 0 {
			timeout = time.Duration(f * float64(time.Second))
		}
	}
	if kw != nil {
		if v, ok := kw.GetStr("block"); ok {
			if b, ok2 := v.(*object.Bool); ok2 {
				block = b.V
			}
		}
		if v, ok := kw.GetStr("timeout"); ok && v != object.None {
			if f, ok2 := toFloat64(v); ok2 && f >= 0 {
				timeout = time.Duration(f * float64(time.Second))
			}
		}
	}
	return
}

// ─── attachQueueMethods attaches the common Queue methods (put/get/etc.) ─────

func (i *Interp) attachQueueMethods(
	cls *object.Class, inst *object.Instance, qs *qState,
	getFn func(block bool, timeout time.Duration) (object.Object, error),
	emptyErr, fullErr, shutdownErr *object.Class,
) {
	cls.Dict.SetStr("put", &object.BuiltinFunc{Name: "put",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			item, block, timeout := parsePutArgs(a, kw)
			if item == nil {
				return nil, object.Errorf(i.typeErr, "put() requires an item")
			}
			return object.None, i.qPut(qs, item, block, timeout, fullErr, shutdownErr)
		}})

	cls.Dict.SetStr("put_nowait", &object.BuiltinFunc{Name: "put_nowait",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			item, _, _ := parsePutArgs(a, kw)
			if item == nil {
				return nil, object.Errorf(i.typeErr, "put_nowait() requires an item")
			}
			return object.None, i.qPut(qs, item, false, -1, fullErr, shutdownErr)
		}})

	cls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			block, timeout := parseGetArgs(a, kw)
			return getFn(block, timeout)
		}})

	cls.Dict.SetStr("get_nowait", &object.BuiltinFunc{Name: "get_nowait",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return getFn(false, -1)
		}})

	cls.Dict.SetStr("qsize", &object.BuiltinFunc{Name: "qsize",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			qs.mu.Lock()
			n := len(qs.items)
			qs.mu.Unlock()
			return object.NewInt(int64(n)), nil
		}})

	cls.Dict.SetStr("empty", &object.BuiltinFunc{Name: "empty",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			qs.mu.Lock()
			n := len(qs.items)
			qs.mu.Unlock()
			return object.BoolOf(n == 0), nil
		}})

	cls.Dict.SetStr("full", &object.BuiltinFunc{Name: "full",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			qs.mu.Lock()
			full := qs.isFull()
			qs.mu.Unlock()
			return object.BoolOf(full), nil
		}})

	cls.Dict.SetStr("task_done", &object.BuiltinFunc{Name: "task_done",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, i.qTaskDone(qs)
		}})

	cls.Dict.SetStr("join", &object.BuiltinFunc{Name: "join",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			i.qJoin(qs)
			return object.None, nil
		}})

	cls.Dict.SetStr("shutdown", &object.BuiltinFunc{Name: "shutdown",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			immediate := false
			if len(a) > 0 {
				if b, ok := a[0].(*object.Bool); ok {
					immediate = b.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("immediate"); ok {
					if b, ok2 := v.(*object.Bool); ok2 {
						immediate = b.V
					}
				}
			}
			i.qShutdown(qs, immediate)
			return object.None, nil
		}})
}

// ─── FIFO Queue instance ─────────────────────────────────────────────────────

func (i *Interp) qFifoInstance(qs *qState, emptyErr, fullErr, shutdownErr *object.Class) *object.Instance {
	cls := &object.Class{Name: "Queue", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	getFn := func(block bool, timeout time.Duration) (object.Object, error) {
		return i.qGetFifo(qs, block, timeout, emptyErr, shutdownErr)
	}
	i.attachQueueMethods(cls, inst, qs, getFn, emptyErr, fullErr, shutdownErr)
	return inst
}

// ─── LIFO Queue instance ─────────────────────────────────────────────────────

func (i *Interp) qLifoInstance(qs *qState, emptyErr, fullErr, shutdownErr *object.Class) *object.Instance {
	cls := &object.Class{Name: "LifoQueue", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	getFn := func(block bool, timeout time.Duration) (object.Object, error) {
		return i.qGetLifo(qs, block, timeout, emptyErr, shutdownErr)
	}
	i.attachQueueMethods(cls, inst, qs, getFn, emptyErr, fullErr, shutdownErr)
	return inst
}

// ─── PriorityQueue instance ───────────────────────────────────────────────────

func (i *Interp) qPriorityInstance(maxsize int, emptyErr, fullErr, shutdownErr *object.Class) *object.Instance {
	ph := &pqHeap{
		less: func(a, b object.Object) bool {
			res, err := i.lt(a, b)
			if err != nil {
				return false
			}
			return res
		},
	}
	heap.Init(ph)

	var mu sync.Mutex
	notEmpty := sync.NewCond(&mu)
	notFull := sync.NewCond(&mu)
	allDone := sync.NewCond(&mu)
	var unfinished int
	var shutdown, shutImm bool

	isFull := func() bool { return maxsize > 0 && ph.Len() >= maxsize }

	cls := &object.Class{Name: "PriorityQueue", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	cls.Dict.SetStr("put", &object.BuiltinFunc{Name: "put",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			item, block, timeout := parsePutArgs(a, kw)
			if item == nil {
				return nil, object.Errorf(i.typeErr, "put() requires an item")
			}
			mu.Lock()
			defer mu.Unlock()
			if shutdown {
				return nil, object.Errorf(shutdownErr, "Queue.put() on a shut-down queue")
			}
			if isFull() {
				if !block {
					return nil, object.Errorf(fullErr, "Queue full")
				}
				if timeout > 0 {
					qWaitTimeout(notFull, func() bool { return !isFull() || shutdown }, timeout)
					if shutdown {
						return nil, object.Errorf(shutdownErr, "Queue.put() on a shut-down queue")
					}
					if isFull() {
						return nil, object.Errorf(fullErr, "Queue.put() timed out")
					}
				} else {
					for isFull() && !shutdown {
						notFull.Wait()
					}
					if shutdown {
						return nil, object.Errorf(shutdownErr, "Queue.put() on a shut-down queue")
					}
				}
			}
			heap.Push(ph, item)
			unfinished++
			notEmpty.Signal()
			return object.None, nil
		}})

	cls.Dict.SetStr("put_nowait", &object.BuiltinFunc{Name: "put_nowait",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			item, _, _ := parsePutArgs(a, kw)
			if item == nil {
				return nil, object.Errorf(i.typeErr, "put_nowait() requires an item")
			}
			mu.Lock()
			defer mu.Unlock()
			if shutdown {
				return nil, object.Errorf(shutdownErr, "Queue.put() on a shut-down queue")
			}
			if isFull() {
				return nil, object.Errorf(fullErr, "Queue full")
			}
			heap.Push(ph, item)
			unfinished++
			notEmpty.Signal()
			return object.None, nil
		}})

	doGet := func(block bool, timeout time.Duration) (object.Object, error) {
		mu.Lock()
		defer mu.Unlock()
		if shutImm {
			return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down queue")
		}
		if ph.Len() == 0 {
			if shutdown {
				return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down empty queue")
			}
			if !block {
				return nil, object.Errorf(emptyErr, "Queue empty")
			}
			if timeout > 0 {
				ok := qWaitTimeout(notEmpty, func() bool { return ph.Len() > 0 || shutdown || shutImm }, timeout)
				if shutImm {
					return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down queue")
				}
				if shutdown && ph.Len() == 0 {
					return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down empty queue")
				}
				if !ok {
					return nil, object.Errorf(emptyErr, "Queue.get() timed out")
				}
			} else {
				for ph.Len() == 0 && !shutdown && !shutImm {
					notEmpty.Wait()
				}
				if shutImm {
					return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down queue")
				}
				if shutdown && ph.Len() == 0 {
					return nil, object.Errorf(shutdownErr, "Queue.get() on a shut-down empty queue")
				}
			}
		}
		item := heap.Pop(ph).(object.Object)
		notFull.Signal()
		return item, nil
	}

	cls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			block, timeout := parseGetArgs(a, kw)
			return doGet(block, timeout)
		}})

	cls.Dict.SetStr("get_nowait", &object.BuiltinFunc{Name: "get_nowait",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return doGet(false, -1)
		}})

	cls.Dict.SetStr("qsize", &object.BuiltinFunc{Name: "qsize",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			mu.Lock()
			n := ph.Len()
			mu.Unlock()
			return object.NewInt(int64(n)), nil
		}})

	cls.Dict.SetStr("empty", &object.BuiltinFunc{Name: "empty",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			mu.Lock()
			n := ph.Len()
			mu.Unlock()
			return object.BoolOf(n == 0), nil
		}})

	cls.Dict.SetStr("full", &object.BuiltinFunc{Name: "full",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			mu.Lock()
			full := isFull()
			mu.Unlock()
			return object.BoolOf(full), nil
		}})

	cls.Dict.SetStr("task_done", &object.BuiltinFunc{Name: "task_done",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			mu.Lock()
			defer mu.Unlock()
			unfinished--
			if unfinished < 0 {
				unfinished = 0
				return nil, object.Errorf(i.valueErr, "task_done() called too many times")
			}
			if unfinished == 0 {
				allDone.Broadcast()
			}
			return object.None, nil
		}})

	cls.Dict.SetStr("join", &object.BuiltinFunc{Name: "join",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			mu.Lock()
			defer mu.Unlock()
			for unfinished > 0 {
				allDone.Wait()
			}
			return object.None, nil
		}})

	cls.Dict.SetStr("shutdown", &object.BuiltinFunc{Name: "shutdown",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			immediate := false
			if len(a) > 0 {
				if b, ok := a[0].(*object.Bool); ok {
					immediate = b.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("immediate"); ok {
					if b, ok2 := v.(*object.Bool); ok2 {
						immediate = b.V
					}
				}
			}
			mu.Lock()
			defer mu.Unlock()
			shutdown = true
			if immediate {
				shutImm = true
				ph.items = nil
				unfinished = 0
				allDone.Broadcast()
			}
			notFull.Broadcast()
			notEmpty.Broadcast()
			return object.None, nil
		}})

	return inst
}

// ─── SimpleQueue instance ─────────────────────────────────────────────────────

func (i *Interp) qSimpleInstance(emptyErr *object.Class) *object.Instance {
	var mu sync.Mutex
	notEmpty := sync.NewCond(&mu)
	var items []object.Object

	cls := &object.Class{Name: "SimpleQueue", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	cls.Dict.SetStr("put", &object.BuiltinFunc{Name: "put",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			item, _, _ := parsePutArgs(a, kw)
			if item == nil {
				return nil, object.Errorf(i.typeErr, "put() requires an item")
			}
			mu.Lock()
			items = append(items, item)
			notEmpty.Signal()
			mu.Unlock()
			return object.None, nil
		}})

	cls.Dict.SetStr("put_nowait", &object.BuiltinFunc{Name: "put_nowait",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			item, _, _ := parsePutArgs(a, kw)
			if item == nil {
				return nil, object.Errorf(i.typeErr, "put_nowait() requires an item")
			}
			mu.Lock()
			items = append(items, item)
			notEmpty.Signal()
			mu.Unlock()
			return object.None, nil
		}})

	doGet := func(block bool, timeout time.Duration) (object.Object, error) {
		mu.Lock()
		defer mu.Unlock()
		if len(items) == 0 {
			if !block {
				return nil, object.Errorf(emptyErr, "SimpleQueue empty")
			}
			if timeout > 0 {
				ok := qWaitTimeout(notEmpty, func() bool { return len(items) > 0 }, timeout)
				if !ok {
					return nil, object.Errorf(emptyErr, "SimpleQueue.get() timed out")
				}
			} else {
				for len(items) == 0 {
					notEmpty.Wait()
				}
			}
		}
		item := items[0]
		items = items[1:]
		return item, nil
	}

	cls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			block, timeout := parseGetArgs(a, kw)
			return doGet(block, timeout)
		}})

	cls.Dict.SetStr("get_nowait", &object.BuiltinFunc{Name: "get_nowait",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return doGet(false, -1)
		}})

	cls.Dict.SetStr("qsize", &object.BuiltinFunc{Name: "qsize",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			mu.Lock()
			n := len(items)
			mu.Unlock()
			return object.NewInt(int64(n)), nil
		}})

	cls.Dict.SetStr("empty", &object.BuiltinFunc{Name: "empty",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			mu.Lock()
			n := len(items)
			mu.Unlock()
			return object.BoolOf(n == 0), nil
		}})

	return inst
}

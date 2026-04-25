package vm

import (
	"container/heap"
	"sort"
	"sync"

	"github.com/tamnd/goipy/object"
)

// ─── event ───────────────────────────────────────────────────────────────────

type scEvent struct {
	id       int64
	t        float64
	priority int64
	seq      int64 // insertion order; breaks ties for FIFO
	action   object.Object
	argument object.Object // tuple of positional args
	kwargs   object.Object // dict or None
	heapIdx  int           // managed by heap.Interface
}

// ─── heap ────────────────────────────────────────────────────────────────────

type scHeap []*scEvent

func (h scHeap) Len() int { return len(h) }
func (h scHeap) Less(a, b int) bool {
	ea, eb := h[a], h[b]
	if ea.t != eb.t {
		return ea.t < eb.t
	}
	if ea.priority != eb.priority {
		return ea.priority < eb.priority
	}
	return ea.seq < eb.seq
}
func (h scHeap) Swap(a, b int) {
	h[a], h[b] = h[b], h[a]
	h[a].heapIdx = a
	h[b].heapIdx = b
}
func (h *scHeap) Push(x any) {
	ev := x.(*scEvent)
	ev.heapIdx = len(*h)
	*h = append(*h, ev)
}
func (h *scHeap) Pop() any {
	old := *h
	n := len(old)
	ev := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	ev.heapIdx = -1
	return ev
}

// ─── scheduler state ─────────────────────────────────────────────────────────

type scSchedulerState struct {
	timefunc  object.Object
	delayfunc object.Object
	h         scHeap
	mu        sync.Mutex
	nextID    int64
	nextSeq   int64
	// tokenMap maps event id → *scEvent (for cancel by token).
	tokenMap sync.Map
}

func (ss *scSchedulerState) push(t float64, priority int64, action, argument, kwargs object.Object) *scEvent {
	ss.mu.Lock()
	ss.nextID++
	ss.nextSeq++
	ev := &scEvent{
		id:       ss.nextID,
		t:        t,
		priority: priority,
		seq:      ss.nextSeq,
		action:   action,
		argument: argument,
		kwargs:   kwargs,
	}
	heap.Push(&ss.h, ev)
	ss.mu.Unlock()
	ss.tokenMap.Store(ev.id, ev)
	return ev
}

func (ss *scSchedulerState) remove(id int64) bool {
	v, ok := ss.tokenMap.Load(id)
	if !ok {
		return false
	}
	ev := v.(*scEvent)
	ss.mu.Lock()
	idx := ev.heapIdx
	inHeap := idx >= 0 && idx < len(ss.h) && ss.h[idx] == ev
	if inHeap {
		heap.Remove(&ss.h, idx)
	}
	ss.mu.Unlock()
	if !inHeap {
		return false
	}
	ss.tokenMap.Delete(id)
	return true
}

// ─── buildSched ──────────────────────────────────────────────────────────────

func (i *Interp) buildSched() *object.Module {
	m := &object.Module{Name: "sched", Dict: object.NewDict()}

	// Grab default timefunc / delayfunc from the time module.
	var defaultTimefunc, defaultDelayfunc object.Object
	if timeMod, ok := i.builtinModule("time"); ok {
		if v, ok2 := timeMod.Dict.GetStr("monotonic"); ok2 {
			defaultTimefunc = v
		}
		if v, ok2 := timeMod.Dict.GetStr("sleep"); ok2 {
			defaultDelayfunc = v
		}
	}

	m.Dict.SetStr("scheduler", &object.BuiltinFunc{Name: "scheduler",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			tf := defaultTimefunc
			df := defaultDelayfunc

			if len(a) > 0 && a[0] != object.None {
				tf = a[0]
			}
			if len(a) > 1 && a[1] != object.None {
				df = a[1]
			}
			if kw != nil {
				if v, ok := kw.GetStr("timefunc"); ok && v != object.None {
					tf = v
				}
				if v, ok := kw.GetStr("delayfunc"); ok && v != object.None {
					df = v
				}
			}
			if tf == nil || df == nil {
				return nil, object.Errorf(i.runtimeErr, "sched: time module unavailable")
			}

			ss := &scSchedulerState{timefunc: tf, delayfunc: df}
			heap.Init(&ss.h)
			return i.scSchedulerInstance(ss), nil
		}})

	return m
}

func (i *Interp) scSchedulerInstance(ss *scSchedulerState) *object.Instance {
	cls := &object.Class{Name: "scheduler", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	callTimefunc := func() (float64, error) {
		res, err := i.callObject(ss.timefunc, nil, nil)
		if err != nil {
			return 0, err
		}
		if f, ok := toFloat64(res); ok {
			return f, nil
		}
		return 0, object.Errorf(i.typeErr, "timefunc must return a number")
	}

	callDelayfunc := func(n float64) error {
		_, err := i.callObject(ss.delayfunc, []object.Object{&object.Float{V: n}}, nil)
		return err
	}

	// makeToken creates a Python instance representing an event.
	// The token stores the event id so cancel() can look up the *scEvent.
	makeToken := func(ev *scEvent) *object.Instance {
		tokCls := &object.Class{Name: "Event", Dict: object.NewDict()}
		tok := &object.Instance{Class: tokCls, Dict: object.NewDict()}
		tok.Dict.SetStr("time",     &object.Float{V: ev.t})
		tok.Dict.SetStr("priority", object.NewInt(ev.priority))
		tok.Dict.SetStr("action",   ev.action)
		arg := ev.argument
		if arg == nil {
			arg = &object.Tuple{V: []object.Object{}}
		}
		tok.Dict.SetStr("argument", arg)
		kw := ev.kwargs
		if kw == nil {
			kw = object.None
		}
		tok.Dict.SetStr("kwargs",   kw)
		tok.Dict.SetStr("_id",      object.NewInt(ev.id))
		return tok
	}

	getID := func(tok object.Object) (int64, bool) {
		inst2, ok := tok.(*object.Instance)
		if !ok {
			return 0, false
		}
		v, ok2 := inst2.Dict.GetStr("_id")
		if !ok2 {
			return 0, false
		}
		n, ok3 := toInt64(v)
		return n, ok3
	}

	// parseEventArgs: a[0]=t/delay, a[1]=priority, a[2]=action, [a[3]=argument, a[4]=kwargs]
	parseEventArgs := func(a []object.Object, kw *object.Dict, firstName string) (
		tOrDelay float64, priority int64, action, argument, kwargs object.Object, err error,
	) {
		argument = &object.Tuple{V: []object.Object{}}
		kwargs = object.None
		if len(a) < 3 {
			err = object.Errorf(i.typeErr, "%s(), priority, action are required", firstName)
			return
		}
		f, ok := toFloat64(a[0])
		if !ok {
			if n, ok2 := toInt64(a[0]); ok2 {
				f = float64(n)
			} else {
				err = object.Errorf(i.typeErr, "%s must be numeric", firstName)
				return
			}
		}
		tOrDelay = f
		if p, ok2 := toInt64(a[1]); ok2 {
			priority = p
		} else if p2, ok3 := toFloat64(a[1]); ok3 {
			priority = int64(p2)
		} else {
			err = object.Errorf(i.typeErr, "priority must be numeric")
			return
		}
		action = a[2]
		if len(a) > 3 {
			argument = a[3]
		}
		if len(a) > 4 {
			kwargs = a[4]
		}
		if kw != nil {
			if v, ok3 := kw.GetStr("argument"); ok3 {
				argument = v
			}
			if v, ok3 := kw.GetStr("kwargs"); ok3 {
				kwargs = v
			}
		}
		return
	}

	callAction := func(ev *scEvent) error {
		var posArgs []object.Object
		switch t := ev.argument.(type) {
		case *object.Tuple:
			posArgs = append([]object.Object(nil), t.V...)
		case *object.List:
			posArgs = append([]object.Object(nil), t.V...)
		}
		var kwDict *object.Dict
		if d, ok := ev.kwargs.(*object.Dict); ok {
			kwDict = d
		}
		_, err := i.callObject(ev.action, posArgs, kwDict)
		return err
	}

	// ─── enterabs ─────────────────────────────────────────────────────────
	cls.Dict.SetStr("enterabs", &object.BuiltinFunc{Name: "enterabs",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			t, priority, action, argument, kwargs, err := parseEventArgs(a, kw, "time")
			if err != nil {
				return nil, err
			}
			ev := ss.push(t, priority, action, argument, kwargs)
			return makeToken(ev), nil
		}})

	// ─── enter ────────────────────────────────────────────────────────────
	cls.Dict.SetStr("enter", &object.BuiltinFunc{Name: "enter",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			delay, priority, action, argument, kwargs, err := parseEventArgs(a, kw, "delay")
			if err != nil {
				return nil, err
			}
			now, err2 := callTimefunc()
			if err2 != nil {
				return nil, err2
			}
			ev := ss.push(now+delay, priority, action, argument, kwargs)
			return makeToken(ev), nil
		}})

	// ─── cancel ───────────────────────────────────────────────────────────
	cls.Dict.SetStr("cancel", &object.BuiltinFunc{Name: "cancel",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "cancel() requires an event argument")
			}
			id, ok := getID(a[0])
			if !ok || !ss.remove(id) {
				return nil, object.Errorf(i.valueErr, "event not in queue")
			}
			return object.None, nil
		}})

	// ─── empty ────────────────────────────────────────────────────────────
	cls.Dict.SetStr("empty", &object.BuiltinFunc{Name: "empty",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ss.mu.Lock()
			n := len(ss.h)
			ss.mu.Unlock()
			return object.BoolOf(n == 0), nil
		}})

	// ─── run ──────────────────────────────────────────────────────────────
	cls.Dict.SetStr("run", &object.BuiltinFunc{Name: "run",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			blocking := true
			if len(a) > 0 {
				if b, ok := a[0].(*object.Bool); ok {
					blocking = b.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("blocking"); ok {
					if b, ok2 := v.(*object.Bool); ok2 {
						blocking = b.V
					}
				}
			}

			for {
				ss.mu.Lock()
				if len(ss.h) == 0 {
					ss.mu.Unlock()
					return object.None, nil
				}
				ev := ss.h[0]
				ss.mu.Unlock()

				now, err := callTimefunc()
				if err != nil {
					return nil, err
				}

				delay := ev.t - now
				if delay > 0 {
					if !blocking {
						return &object.Float{V: ev.t}, nil
					}
					if err2 := callDelayfunc(delay); err2 != nil {
						return nil, err2
					}
					continue
				}

				// Pop the due event.
				ss.mu.Lock()
				if len(ss.h) == 0 || ss.h[0] != ev {
					ss.mu.Unlock()
					continue
				}
				heap.Pop(&ss.h)
				ss.mu.Unlock()
				ss.tokenMap.Delete(ev.id)

				// Yield to other threads (CPython spec).
				if err3 := callDelayfunc(0); err3 != nil {
					return nil, err3
				}

				if err4 := callAction(ev); err4 != nil {
					return nil, err4
				}

				if !blocking {
					ss.mu.Lock()
					empty := len(ss.h) == 0
					var nextT float64
					if !empty {
						nextT = ss.h[0].t
					}
					ss.mu.Unlock()
					if empty {
						return object.None, nil
					}
					now2, _ := callTimefunc()
					if nextT > now2 {
						return &object.Float{V: nextT}, nil
					}
					// More events due — keep processing.
				}
			}
		}})

	// ─── queue (read-only property) ───────────────────────────────────────
	cls.Dict.SetStr("queue", &object.Property{Fget: &object.BuiltinFunc{Name: "queue",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ss.mu.Lock()
			snapshot := make([]*scEvent, len(ss.h))
			copy(snapshot, ss.h)
			ss.mu.Unlock()
			sort.Slice(snapshot, func(a, b int) bool {
				ea, eb := snapshot[a], snapshot[b]
				if ea.t != eb.t {
					return ea.t < eb.t
				}
				if ea.priority != eb.priority {
					return ea.priority < eb.priority
				}
				return ea.seq < eb.seq
			})
			items := make([]object.Object, len(snapshot))
			for idx, ev := range snapshot {
				evCls := &object.Class{Name: "Event", Dict: object.NewDict()}
				evInst := &object.Instance{Class: evCls, Dict: object.NewDict()}
				evInst.Dict.SetStr("time",     &object.Float{V: ev.t})
				evInst.Dict.SetStr("priority", object.NewInt(ev.priority))
				evInst.Dict.SetStr("action",   ev.action)
				arg := ev.argument
				if arg == nil {
					arg = &object.Tuple{V: []object.Object{}}
				}
				evInst.Dict.SetStr("argument", arg)
				kw := ev.kwargs
				if kw == nil {
					kw = object.None
				}
				evInst.Dict.SetStr("kwargs", kw)
				items[idx] = evInst
			}
			return &object.List{V: items}, nil
		}}})

	return inst
}

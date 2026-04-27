package vm

import (
	"sync"

	"github.com/tamnd/goipy/object"
)

type atexitHandler struct {
	fn   object.Object
	args []object.Object
	kw   *object.Dict
}

type atexitState struct {
	mu       sync.Mutex
	handlers []atexitHandler
}

func (i *Interp) buildAtexit() *object.Module {
	m := &object.Module{Name: "atexit", Dict: object.NewDict()}
	state := &atexitState{}

	m.Dict.SetStr("register", &object.BuiltinFunc{
		Name: "register",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "register() requires at least one argument")
			}
			fn := a[0]
			args := make([]object.Object, len(a)-1)
			copy(args, a[1:])
			state.mu.Lock()
			state.handlers = append(state.handlers, atexitHandler{fn: fn, args: args, kw: kw})
			state.mu.Unlock()
			return fn, nil
		},
	})

	m.Dict.SetStr("unregister", &object.BuiltinFunc{
		Name: "unregister",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			target := a[0]
			state.mu.Lock()
			defer state.mu.Unlock()
			filtered := state.handlers[:0]
			for _, h := range state.handlers {
				eq, _ := objectEqual(h.fn, target)
				if !eq {
					filtered = append(filtered, h)
				}
			}
			state.handlers = filtered
			return object.None, nil
		},
	})

	// _run_exitfuncs calls handlers in LIFO order — used by tests and by the
	// interpreter shutdown path. Matches CPython: after running the snapshot,
	// the entire handler list is cleared (including any registered during the run).
	runExitFuncs := func(ii any) {
		state.mu.Lock()
		handlers := make([]atexitHandler, len(state.handlers))
		copy(handlers, state.handlers)
		state.mu.Unlock()
		interp := ii.(*Interp)
		for idx := len(handlers) - 1; idx >= 0; idx-- {
			h := handlers[idx]
			interp.callObject(h.fn, h.args, h.kw) //nolint
		}
		// Clear all handlers after the run (CPython semantics: wipes anything
		// registered during the run too).
		state.mu.Lock()
		state.handlers = state.handlers[:0]
		state.mu.Unlock()
	}

	m.Dict.SetStr("_run_exitfuncs", &object.BuiltinFunc{
		Name: "_run_exitfuncs",
		Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			runExitFuncs(ii)
			return object.None, nil
		},
	})

	// _clear clears all registered handlers (for testing).
	m.Dict.SetStr("_clear", &object.BuiltinFunc{
		Name: "_clear",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			state.mu.Lock()
			state.handlers = state.handlers[:0]
			state.mu.Unlock()
			return object.None, nil
		},
	})

	// _ncallbacks returns the number of registered handlers (for testing).
	m.Dict.SetStr("_ncallbacks", &object.BuiltinFunc{
		Name: "_ncallbacks",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			state.mu.Lock()
			n := len(state.handlers)
			state.mu.Unlock()
			return object.NewInt(int64(n)), nil
		},
	})

	return m
}

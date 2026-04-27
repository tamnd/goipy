package vm

import (
	"io"
	"os"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildContextlib() *object.Module {
	m := &object.Module{Name: "contextlib", Dict: object.NewDict()}

	// ── AbstractContextManager ────────────────────────────────────────────────
	absCM := &object.Class{Name: "AbstractContextManager", Dict: object.NewDict()}
	absCM.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 {
				return a[0], nil
			}
			return object.None, nil
		}})
	absCM.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	m.Dict.SetStr("AbstractContextManager", absCM)

	// ── AbstractAsyncContextManager ───────────────────────────────────────────
	absACM := &object.Class{Name: "AbstractAsyncContextManager", Dict: object.NewDict()}
	absACM.Dict.SetStr("__aenter__", &object.BuiltinFunc{Name: "__aenter__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 {
				return a[0], nil
			}
			return object.None, nil
		}})
	absACM.Dict.SetStr("__aexit__", &object.BuiltinFunc{Name: "__aexit__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	m.Dict.SetStr("AbstractAsyncContextManager", absACM)

	// ── suppress(*exceptions) ────────────────────────────────────────────────
	m.Dict.SetStr("suppress", &object.BuiltinFunc{Name: "suppress",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			classes := make([]*object.Class, 0, len(a))
			for _, arg := range a {
				if cls, ok := arg.(*object.Class); ok {
					classes = append(classes, cls)
				}
			}
			cls := &object.Class{Name: "_SuppressContext", Dict: object.NewDict()}
			cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
				Call: func(_ any, a2 []object.Object, _ *object.Dict) (object.Object, error) {
					if len(a2) > 0 {
						return a2[0], nil
					}
					return object.None, nil
				}})
			cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
				Call: func(_ any, a2 []object.Object, _ *object.Dict) (object.Object, error) {
					// a2[0]=self a2[1]=exc_type a2[2]=exc_val a2[3]=tb
					if len(a2) < 2 || a2[1] == object.None {
						return object.False, nil
					}
					excCls, ok := a2[1].(*object.Class)
					if !ok {
						return object.False, nil
					}
					for _, cls2 := range classes {
						if object.IsSubclass(excCls, cls2) {
							return object.True, nil
						}
					}
					return object.False, nil
				}})
			return &object.Instance{Class: cls, Dict: object.NewDict()}, nil
		}})

	// ── closing(thing) ───────────────────────────────────────────────────────
	m.Dict.SetStr("closing", &object.BuiltinFunc{Name: "closing",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			var thing object.Object = object.None
			if len(a) > 0 {
				thing = a[0]
			}
			cls := &object.Class{Name: "closing", Dict: object.NewDict()}
			inst := &object.Instance{Class: cls, Dict: object.NewDict()}
			inst.Dict.SetStr("thing", thing)
			cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return thing, nil
				}})
			cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					closeMethod, getErr := ii.getAttr(thing, "close")
					if getErr == nil {
						_, _ = ii.callObject(closeMethod, nil, nil)
					}
					return object.False, nil
				}})
			return inst, nil
		}})

	// ── nullcontext(enter_result=None) ───────────────────────────────────────
	m.Dict.SetStr("nullcontext", &object.BuiltinFunc{Name: "nullcontext",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			var enterResult object.Object = object.None
			if kw != nil {
				if v, ok := kw.GetStr("enter_result"); ok {
					enterResult = v
				}
			}
			if len(a) > 0 {
				enterResult = a[0]
			}
			cls := &object.Class{Name: "nullcontext", Dict: object.NewDict()}
			inst := &object.Instance{Class: cls, Dict: object.NewDict()}
			inst.Dict.SetStr("enter_result", enterResult)
			cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return enterResult, nil
				}})
			cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return object.False, nil
				}})
			return inst, nil
		}})

	// ── contextmanager decorator ─────────────────────────────────────────────
	m.Dict.SetStr("contextmanager", &object.BuiltinFunc{Name: "contextmanager",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			if len(a) == 0 {
				return nil, object.Errorf(ii.typeErr, "contextmanager() requires a function argument")
			}
			fn := a[0]
			return &object.BuiltinFunc{Name: "contextmanager_wrapper",
				Call: func(interp2 any, args []object.Object, kw *object.Dict) (object.Object, error) {
					ii2 := interpFrom(interp2)
					if ii2 == nil {
						ii2 = ii
					}
					gen, err := ii2.callObject(fn, args, kw)
					if err != nil {
						return nil, err
					}
					genObj, ok := gen.(*object.Generator)
					if !ok {
						return nil, object.Errorf(ii2.typeErr, "contextmanager: function must be a generator function")
					}
					return ctxlibMakeGenCtxMgr(genObj, ii2), nil
				}}, nil
		}})

	// ── asynccontextmanager decorator ────────────────────────────────────────
	m.Dict.SetStr("asynccontextmanager", &object.BuiltinFunc{Name: "asynccontextmanager",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			if len(a) == 0 {
				return nil, object.Errorf(ii.typeErr, "asynccontextmanager() requires a function argument")
			}
			fn := a[0]
			return &object.BuiltinFunc{Name: "asynccontextmanager_wrapper",
				Call: func(interp2 any, args []object.Object, kw *object.Dict) (object.Object, error) {
					ii2 := interpFrom(interp2)
					if ii2 == nil {
						ii2 = ii
					}
					gen, err := ii2.callObject(fn, args, kw)
					if err != nil {
						return nil, err
					}
					genObj, ok := gen.(*object.Generator)
					if !ok {
						return nil, object.Errorf(ii2.typeErr, "asynccontextmanager: function must be a generator function")
					}
					return ctxlibMakeGenCtxMgr(genObj, ii2), nil
				}}, nil
		}})

	// ── redirect_stdout(new_target) ──────────────────────────────────────────
	m.Dict.SetStr("redirect_stdout", &object.BuiltinFunc{Name: "redirect_stdout",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			var target object.Object = object.None
			if len(a) > 0 {
				target = a[0]
			}
			return ctxlibRedirect(i, "stdout", target), nil
		}})

	// ── redirect_stderr(new_target) ──────────────────────────────────────────
	m.Dict.SetStr("redirect_stderr", &object.BuiltinFunc{Name: "redirect_stderr",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			var target object.Object = object.None
			if len(a) > 0 {
				target = a[0]
			}
			return ctxlibRedirect(i, "stderr", target), nil
		}})

	// ── chdir(path) ──────────────────────────────────────────────────────────
	m.Dict.SetStr("chdir", &object.BuiltinFunc{Name: "chdir",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			var path string
			if len(a) > 0 {
				if s, ok := a[0].(*object.Str); ok {
					path = s.V
				}
			}
			cls := &object.Class{Name: "chdir", Dict: object.NewDict()}
			inst := &object.Instance{Class: cls, Dict: object.NewDict()}
			inst.Dict.SetStr("path", &object.Str{V: path})
			var oldDir string
			cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					d, err2 := os.Getwd()
					if err2 == nil {
						oldDir = d
					}
					if path != "" {
						_ = os.Chdir(path)
					}
					return object.None, nil
				}})
			cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					if oldDir != "" {
						_ = os.Chdir(oldDir)
					}
					return object.False, nil
				}})
			return inst, nil
		}})

	// ── ExitStack ────────────────────────────────────────────────────────────
	m.Dict.SetStr("ExitStack", &object.BuiltinFunc{Name: "ExitStack",
		Call: func(interp any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			return ctxlibMakeExitStack(ii), nil
		}})

	// ── AsyncExitStack ───────────────────────────────────────────────────────
	m.Dict.SetStr("AsyncExitStack", &object.BuiltinFunc{Name: "AsyncExitStack",
		Call: func(interp any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			return ctxlibMakeExitStack(ii), nil
		}})

	// ── contextdecorator ─────────────────────────────────────────────────────
	ctxDecoratorCls := &object.Class{Name: "ContextDecorator", Dict: object.NewDict()}
	ctxDecoratorCls.Dict.SetStr("__call__", &object.BuiltinFunc{Name: "__call__",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			if len(a) < 2 {
				return object.None, nil
			}
			fn := a[1]
			return &object.BuiltinFunc{Name: "decorator_wrapper",
				Call: func(interp2 any, args []object.Object, kw *object.Dict) (object.Object, error) {
					ii2 := interpFrom(interp2)
					if ii2 == nil {
						ii2 = ii
					}
					return ii2.callObject(fn, args, kw)
				}}, nil
		}})
	m.Dict.SetStr("ContextDecorator", ctxDecoratorCls)

	// ── SUPPRESS sentinel ─────────────────────────────────────────────────────
	m.Dict.SetStr("SUPPRESS", &object.Str{V: "<no value>"})

	return m
}

// ctxlibMakeGenCtxMgr creates a _GeneratorContextManager instance.
func ctxlibMakeGenCtxMgr(gen *object.Generator, ii *Interp) *object.Instance {
	cls := &object.Class{Name: "_GeneratorContextManager", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("gen", gen)

	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			// Advance to the yield point and return the yielded value.
			v, err := ii.resumeGenerator(gen, object.None)
			if err != nil {
				if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, ii.stopIter) {
					return nil, object.Errorf(ii.runtimeErr, "generator didn't yield")
				}
				return nil, err
			}
			return v, nil
		}})

	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// a[0]=self, a[1]=exc_type, a[2]=exc_val, a[3]=tb
			noExc := len(a) < 2 || a[1] == object.None
			if noExc {
				// Drive generator to completion; expect StopIteration.
				_, err := ii.resumeGenerator(gen, object.None)
				if err != nil {
					if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, ii.stopIter) {
						return object.False, nil
					}
					return object.False, err
				}
				return object.False, object.Errorf(ii.runtimeErr, "generator didn't stop")
			}
			// Exception occurred — throw it into the generator.
			var excVal *object.Exception
			if len(a) >= 3 {
				if ev, ok := a[2].(*object.Exception); ok {
					excVal = ev
				}
			}
			if excVal == nil && len(a) >= 2 {
				if cls, ok := a[1].(*object.Class); ok {
					excVal = object.NewException(cls, "")
				}
			}
			if excVal == nil {
				return object.False, nil
			}
			_, throwErr := ii.throwGenerator(gen, excVal)
			if throwErr != nil {
				if exc, ok := throwErr.(*object.Exception); ok && object.IsSubclass(exc.Class, ii.stopIter) {
					// StopIteration means the generator handled the exception and exited.
					return object.True, nil
				}
				// The generator re-raised (or raised something else).
				if throwErr == excVal || throwErr == error(excVal) {
					return object.False, nil
				}
				return object.False, throwErr
			}
			// Generator yielded again — this is an error.
			return object.False, object.Errorf(ii.runtimeErr, "generator didn't stop after throw()")
		}})

	return inst
}

// ctxlibRedirect creates a redirect_stdout / redirect_stderr context manager.
func ctxlibRedirect(i *Interp, stream string, target object.Object) *object.Instance {
	cls := &object.Class{Name: "_RedirectStream", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("_new_target", target)

	var oldWriter io.Writer

	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			// Redirect the interpreter's Go-level stream to the Python target.
			if sio, ok := target.(*object.StringIO); ok {
				adapter := &stringIOWriter{sio: sio}
				if stream == "stdout" {
					oldWriter = i.Stdout
					i.Stdout = adapter
				} else {
					oldWriter = i.Stderr
					i.Stderr = adapter
				}
			}
			// Also update sys.stdout / sys.stderr for Python code that reads it.
			if sysModule, ok := i.modules["sys"]; ok {
				sysModule.Dict.SetStr(stream, target)
			}
			return target, nil
		}})
	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if oldWriter != nil {
				if stream == "stdout" {
					i.Stdout = oldWriter
				} else {
					i.Stderr = oldWriter
				}
				oldWriter = nil
			}
			return object.False, nil
		}})
	return inst
}

// ctxlibMakeExitStack creates an ExitStack instance.
func ctxlibMakeExitStack(ii *Interp) *object.Instance {
	// exitCB is callable with signature (exc_type, exc_val, tb) → bool|None.
	// cleanCB is a plain thunk with pre-bound args, never suppresses.
	type cbKind int
	const (
		cbExit  cbKind = iota // push / enter_context
		cbClean               // callback(fn, *args)
	)
	type cb struct {
		kind cbKind
		fn   object.Object
		args []object.Object // only used for cbClean
	}
	var cbs []cb

	cls := &object.Class{Name: "ExitStack", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	// runCallbacks invokes all registered callbacks in LIFO order.
	// Returns true if any exit callback suppressed the current exception.
	runCallbacks := func(excType, excVal, tb object.Object) bool {
		suppressed := false
		for idx := len(cbs) - 1; idx >= 0; idx-- {
			c := cbs[idx]
			switch c.kind {
			case cbExit:
				res, _ := ii.callObject(c.fn, []object.Object{excType, excVal, tb}, nil)
				if isTruthy(res) {
					suppressed = true
					excType = object.None
					excVal = object.None
					tb = object.None
				}
			case cbClean:
				_, _ = ii.callObject(c.fn, c.args, nil)
			}
		}
		return suppressed
	}

	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})

	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// a[0]=self a[1]=exc_type a[2]=exc_val a[3]=tb
			excType := object.Object(object.None)
			excVal := object.Object(object.None)
			tb := object.Object(object.None)
			if len(a) >= 2 {
				excType = a[1]
			}
			if len(a) >= 3 {
				excVal = a[2]
			}
			if len(a) >= 4 {
				tb = a[3]
			}
			return object.BoolOf(runCallbacks(excType, excVal, tb)), nil
		}})

	cls.Dict.SetStr("enter_context", &object.BuiltinFunc{Name: "enter_context",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// a[0]=self a[1]=cm
			if len(a) < 2 {
				return object.None, nil
			}
			cm := a[1]
			enterMethod, getErr := ii.getAttr(cm, "__enter__")
			if getErr != nil {
				return object.None, nil
			}
			result, err := ii.callObject(enterMethod, nil, nil)
			if err != nil {
				return nil, err
			}
			exitMethod, getErr2 := ii.getAttr(cm, "__exit__")
			if getErr2 == nil {
				cbs = append(cbs, cb{kind: cbExit, fn: exitMethod})
			}
			return result, nil
		}})

	cls.Dict.SetStr("callback", &object.BuiltinFunc{Name: "callback",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// a[0]=self a[1]=fn a[2..]=extra_args
			if len(a) < 2 {
				return object.None, nil
			}
			fn := a[1]
			extra := append([]object.Object(nil), a[2:]...)
			cbs = append(cbs, cb{kind: cbClean, fn: fn, args: extra})
			return fn, nil
		}})

	cls.Dict.SetStr("push", &object.BuiltinFunc{Name: "push",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// a[0]=self a[1]=exit_fn  — called with (exc_type, exc_val, tb)
			if len(a) < 2 {
				return object.None, nil
			}
			fn := a[1]
			cbs = append(cbs, cb{kind: cbExit, fn: fn})
			return fn, nil
		}})

	cls.Dict.SetStr("pop_all", &object.BuiltinFunc{Name: "pop_all",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			newStack := ctxlibMakeExitStack(ii)
			cbs = nil
			return newStack, nil
		}})

	cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			runCallbacks(object.None, object.None, object.None)
			cbs = nil
			return object.None, nil
		}})

	return inst
}


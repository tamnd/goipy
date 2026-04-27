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
	// Default __enter__ returns self; subclasses may override.
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
	// __class_getitem__ for Generic support (returns a generic alias stub).
	absCM.Dict.SetStr("__class_getitem__", &object.BuiltinFunc{Name: "__class_getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return absCM, nil
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
	absACM.Dict.SetStr("__class_getitem__", &object.BuiltinFunc{Name: "__class_getitem__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return absACM, nil
		}})
	m.Dict.SetStr("AbstractAsyncContextManager", absACM)

	// ── ContextDecorator ─────────────────────────────────────────────────────
	// Mixin that makes a context manager usable as a function decorator.
	// Subclasses inherit __call__ which wraps the decorated function in `with self:`.
	ctxDecoratorCls := &object.Class{Name: "ContextDecorator", Dict: object.NewDict()}
	ctxDecoratorCls.Dict.SetStr("_recreate_cm", &object.BuiltinFunc{Name: "_recreate_cm",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// Default: return self (same instance reused each call).
			if len(a) > 0 {
				return a[0], nil
			}
			return object.None, nil
		}})
	ctxDecoratorCls.Dict.SetStr("__call__", &object.BuiltinFunc{Name: "__call__",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			// a[0]=self (the CM instance), a[1]=fn to decorate
			if len(a) < 2 {
				return object.None, nil
			}
			selfCM := a[0]
			fn := a[1]
			return ctxlibWrapWithCM(ii, selfCM, fn), nil
		}})
	m.Dict.SetStr("ContextDecorator", ctxDecoratorCls)

	// ── AsyncContextDecorator (Python 3.10+) ─────────────────────────────────
	asyncCtxDecCls := &object.Class{Name: "AsyncContextDecorator", Dict: object.NewDict()}
	asyncCtxDecCls.Dict.SetStr("__call__", &object.BuiltinFunc{Name: "__call__",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			if len(a) < 2 {
				return object.None, nil
			}
			selfCM := a[0]
			fn := a[1]
			return ctxlibWrapWithCM(ii, selfCM, fn), nil
		}})
	m.Dict.SetStr("AsyncContextDecorator", asyncCtxDecCls)

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
			inst := &object.Instance{Class: cls, Dict: object.NewDict()}
			// Python 3.12+: exceptions attribute holds the suppressed exception.
			inst.Dict.SetStr("exceptions", &object.List{V: []object.Object{}})

			cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
				Call: func(_ any, a2 []object.Object, _ *object.Dict) (object.Object, error) {
					// Reset exceptions on entry.
					inst.Dict.SetStr("exceptions", &object.List{V: []object.Object{}})
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
							// Record the suppressed exception.
							if len(a2) >= 3 && a2[2] != object.None {
								if exList, ok2 := inst.Dict.GetStr("exceptions"); ok2 {
									if lst, ok3 := exList.(*object.List); ok3 {
										lst.V = append(lst.V, a2[2])
									}
								}
							}
							return object.True, nil
						}
					}
					return object.False, nil
				}})
			return inst, nil
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

	// ── aclosing(athing) (Python 3.10+) ─────────────────────────────────────
	// Async version of closing — calls athing.aclose() on exit.
	m.Dict.SetStr("aclosing", &object.BuiltinFunc{Name: "aclosing",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			var thing object.Object = object.None
			if len(a) > 0 {
				thing = a[0]
			}
			cls := &object.Class{Name: "aclosing", Dict: object.NewDict()}
			inst := &object.Instance{Class: cls, Dict: object.NewDict()}
			inst.Dict.SetStr("thing", thing)
			cls.Dict.SetStr("__aenter__", &object.BuiltinFunc{Name: "__aenter__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return ctxlibOneShotIter(ii, thing), nil
				}})
			cls.Dict.SetStr("__aexit__", &object.BuiltinFunc{Name: "__aexit__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					// Try aclose() first, fall back to close().
					if acm, err := ii.getAttr(thing, "aclose"); err == nil {
						coro, cerr := ii.callObject(acm, nil, nil)
						if cerr == nil {
							if gen, ok := coro.(*object.Generator); ok {
								_, _ = ii.driveCoroutine(gen)
							}
						}
					} else if cm, err2 := ii.getAttr(thing, "close"); err2 == nil {
						_, _ = ii.callObject(cm, nil, nil)
					}
					return ctxlibOneShotIter(ii, object.False), nil
				}})
			// Also supply synchronous enter/exit so it can be used in non-async code.
			cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return thing, nil
				}})
			cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					if cm, err := ii.getAttr(thing, "close"); err == nil {
						_, _ = ii.callObject(cm, nil, nil)
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
			enter := &object.BuiltinFunc{Name: "__enter__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return enterResult, nil
				}}
			exit := &object.BuiltinFunc{Name: "__exit__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return object.False, nil
				}}
			cls.Dict.SetStr("__enter__", enter)
			cls.Dict.SetStr("__exit__", exit)
			// Async variants (Python 3.10+): same semantics.
			cls.Dict.SetStr("__aenter__", &object.BuiltinFunc{Name: "__aenter__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return enterResult, nil
				}})
			cls.Dict.SetStr("__aexit__", &object.BuiltinFunc{Name: "__aexit__",
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
			var wrapper *object.BuiltinFunc
			wrapper = &object.BuiltinFunc{Name: "contextmanager_wrapper",
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
					cm := ctxlibMakeGenCtxMgr(genObj, ii2)
					// Store wrapper + original args so the CM can be recreated for decorator use.
					cm.Dict.SetStr("_cm_wrapper", wrapper)
					argsCopy := make([]object.Object, len(args))
					copy(argsCopy, args)
					cm.Dict.SetStr("_cm_args", &object.Tuple{V: argsCopy})
					return cm, nil
				}}
			return wrapper, nil
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
			return ctxlibMakeExitStackWith(ii, &[]exitStackCb{}, true), nil
		}})

	// ── SUPPRESS sentinel ─────────────────────────────────────────────────────
	m.Dict.SetStr("SUPPRESS", &object.Str{V: "<no value>"})

	return m
}

// ctxlibWrapWithCM returns a BuiltinFunc that, when called, wraps the call
// to fn inside a `with selfCM:` block. Used by ContextDecorator.__call__ and
// _GeneratorContextManager.__call__.
func ctxlibWrapWithCM(ii *Interp, selfCM object.Object, fn object.Object) *object.BuiltinFunc {
	return &object.BuiltinFunc{Name: "decorator_wrapper",
		Call: func(interp2 any, args []object.Object, kw *object.Dict) (object.Object, error) {
			ii2 := interpFrom(interp2)
			if ii2 == nil {
				ii2 = ii
			}
			// If selfCM has _cm_wrapper, recreate the CM fresh for this call.
			cm := selfCM
			if inst, ok := selfCM.(*object.Instance); ok {
				if wrapperObj, ok2 := inst.Dict.GetStr("_cm_wrapper"); ok2 {
					var cmArgs []object.Object
					if argsObj, ok3 := inst.Dict.GetStr("_cm_args"); ok3 {
						if t, ok4 := argsObj.(*object.Tuple); ok4 {
							cmArgs = t.V
						}
					}
					freshCM, err := ii2.callObject(wrapperObj, cmArgs, nil)
					if err != nil {
						return nil, err
					}
					cm = freshCM
				}
			}
			// __enter__
			enterMethod, err := ii2.getAttr(cm, "__enter__")
			if err != nil {
				return nil, err
			}
			_, err = ii2.callObject(enterMethod, nil, nil)
			if err != nil {
				return nil, err
			}
			// call fn
			result, fnErr := ii2.callObject(fn, args, kw)
			// __exit__
			exitMethod, exitGetErr := ii2.getAttr(cm, "__exit__")
			if exitGetErr == nil {
				exitArgs := []object.Object{object.None, object.None, object.None}
				if fnErr != nil {
					if exc, ok := fnErr.(*object.Exception); ok {
						exitArgs = []object.Object{exc.Class, fnErr, object.None}
					}
				}
				suppressed, _ := ii2.callObject(exitMethod, exitArgs, nil)
				if fnErr != nil && isTruthy(suppressed) {
					return object.None, nil
				}
			}
			if fnErr != nil {
				return nil, fnErr
			}
			return result, nil
		}}
}

// ctxlibOneShotIter returns a one-shot Iter that resolves to val when awaited.
func ctxlibOneShotIter(ii *Interp, val object.Object) *object.Iter {
	done := false
	return &object.Iter{Next: func() (object.Object, bool, error) {
		if done {
			return nil, false, nil
		}
		done = true
		exc := &object.Exception{Class: ii.stopIter, Args: &object.Tuple{V: []object.Object{val}}}
		return nil, false, exc
	}}
}

// ctxlibMakeGenCtxMgr creates a _GeneratorContextManager instance.
func ctxlibMakeGenCtxMgr(gen *object.Generator, ii *Interp) *object.Instance {
	cls := &object.Class{Name: "_GeneratorContextManager", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("gen", gen)

	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
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
			noExc := len(a) < 2 || a[1] == object.None
			if noExc {
				_, err := ii.resumeGenerator(gen, object.None)
				if err != nil {
					if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, ii.stopIter) {
						return object.False, nil
					}
					return object.False, err
				}
				return object.False, object.Errorf(ii.runtimeErr, "generator didn't stop")
			}
			var excVal *object.Exception
			if len(a) >= 3 {
				if ev, ok := a[2].(*object.Exception); ok {
					excVal = ev
				}
			}
			if excVal == nil && len(a) >= 2 {
				if cls2, ok := a[1].(*object.Class); ok {
					excVal = object.NewException(cls2, "")
				}
			}
			if excVal == nil {
				return object.False, nil
			}
			_, throwErr := ii.throwGenerator(gen, excVal)
			if throwErr != nil {
				if exc, ok := throwErr.(*object.Exception); ok && object.IsSubclass(exc.Class, ii.stopIter) {
					return object.True, nil
				}
				if throwErr == excVal || throwErr == error(excVal) {
					return object.False, nil
				}
				return object.False, throwErr
			}
			return object.False, object.Errorf(ii.runtimeErr, "generator didn't stop after throw()")
		}})

	// __aenter__ / __aexit__ delegate to the sync methods, wrapping results for await.
	cls.Dict.SetStr("__aenter__", &object.BuiltinFunc{Name: "__aenter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			v, err := ii.resumeGenerator(gen, object.None)
			if err != nil {
				if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, ii.stopIter) {
					return nil, object.Errorf(ii.runtimeErr, "generator didn't yield")
				}
				return nil, err
			}
			return ctxlibOneShotIter(ii, v), nil
		}})

	cls.Dict.SetStr("__aexit__", &object.BuiltinFunc{Name: "__aexit__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			noExc := len(a) < 2 || a[1] == object.None
			if noExc {
				_, err := ii.resumeGenerator(gen, object.None)
				if err != nil {
					if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, ii.stopIter) {
						return ctxlibOneShotIter(ii, object.False), nil
					}
					return nil, err
				}
				return nil, object.Errorf(ii.runtimeErr, "generator didn't stop")
			}
			var excVal *object.Exception
			if len(a) >= 3 {
				if ev, ok := a[2].(*object.Exception); ok {
					excVal = ev
				}
			}
			if excVal == nil && len(a) >= 2 {
				if cls2, ok := a[1].(*object.Class); ok {
					excVal = object.NewException(cls2, "")
				}
			}
			if excVal == nil {
				return ctxlibOneShotIter(ii, object.False), nil
			}
			_, throwErr := ii.throwGenerator(gen, excVal)
			if throwErr != nil {
				if exc, ok := throwErr.(*object.Exception); ok && object.IsSubclass(exc.Class, ii.stopIter) {
					return ctxlibOneShotIter(ii, object.True), nil
				}
				if throwErr == excVal || throwErr == error(excVal) {
					return ctxlibOneShotIter(ii, object.False), nil
				}
				return nil, throwErr
			}
			return nil, object.Errorf(ii.runtimeErr, "generator didn't stop after throw()")
		}})

	// __call__ makes the _GeneratorContextManager usable as a function decorator.
	// It recreates the generator context manager for each invocation of the
	// decorated function (via the stored _cm_wrapper and _cm_args).
	cls.Dict.SetStr("__call__", &object.BuiltinFunc{Name: "__call__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// a[0]=self, a[1]=fn to decorate
			if len(a) < 2 {
				return object.None, nil
			}
			selfInst := inst
			fn := a[1]
			return ctxlibWrapWithCM(ii, selfInst, fn), nil
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

type exitStackCbKind int

const (
	exitStackCbExit  exitStackCbKind = iota
	exitStackCbClean
)

type exitStackCb struct {
	kind exitStackCbKind
	fn   object.Object
	args []object.Object
	kw   *object.Dict
}

// ctxlibMakeExitStack creates an ExitStack instance.
func ctxlibMakeExitStack(ii *Interp) *object.Instance {
	return ctxlibMakeExitStackWith(ii, &[]exitStackCb{}, false)
}

func ctxlibMakeExitStackWith(ii *Interp, cbsPtr *[]exitStackCb, async bool) *object.Instance {
	name := "ExitStack"
	if async {
		name = "AsyncExitStack"
	}
	cls := &object.Class{Name: name, Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	runCallbacks := func(excType, excVal, tb object.Object) bool {
		suppressed := false
		cbs := *cbsPtr
		for idx := len(cbs) - 1; idx >= 0; idx-- {
			c := cbs[idx]
			switch c.kind {
			case exitStackCbExit:
				res, _ := ii.callObject(c.fn, []object.Object{excType, excVal, tb}, nil)
				if isTruthy(res) {
					suppressed = true
					excType = object.None
					excVal = object.None
					tb = object.None
				}
			case exitStackCbClean:
				_, _ = ii.callObject(c.fn, c.args, c.kw)
			}
		}
		*cbsPtr = nil
		return suppressed
	}

	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})

	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
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

	// __aenter__ / __aexit__ for AsyncExitStack
	cls.Dict.SetStr("__aenter__", &object.BuiltinFunc{Name: "__aenter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return ctxlibOneShotIter(ii, inst), nil
		}})
	cls.Dict.SetStr("__aexit__", &object.BuiltinFunc{Name: "__aexit__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
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
			result := object.BoolOf(runCallbacks(excType, excVal, tb))
			return ctxlibOneShotIter(ii, result), nil
		}})

	cls.Dict.SetStr("enter_context", &object.BuiltinFunc{Name: "enter_context",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
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
				*cbsPtr = append(*cbsPtr, exitStackCb{kind: exitStackCbExit, fn: exitMethod})
			}
			return result, nil
		}})

	cls.Dict.SetStr("enter_async_context", &object.BuiltinFunc{Name: "enter_async_context",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			cm := a[1]
			enterMethod, getErr := ii.getAttr(cm, "__aenter__")
			if getErr != nil {
				// fallback to sync
				enterMethod, getErr = ii.getAttr(cm, "__enter__")
				if getErr != nil {
					return object.None, nil
				}
			}
			result, err := ii.callObject(enterMethod, nil, nil)
			if err != nil {
				return nil, err
			}
			exitMethod, getErr2 := ii.getAttr(cm, "__aexit__")
			if getErr2 != nil {
				exitMethod, getErr2 = ii.getAttr(cm, "__exit__")
			}
			if getErr2 == nil {
				*cbsPtr = append(*cbsPtr, exitStackCb{kind: exitStackCbExit, fn: exitMethod})
			}
			return result, nil
		}})

	cls.Dict.SetStr("callback", &object.BuiltinFunc{Name: "callback",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			fn := a[1]
			extra := append([]object.Object(nil), a[2:]...)
			*cbsPtr = append(*cbsPtr, exitStackCb{kind: exitStackCbClean, fn: fn, args: extra, kw: kw})
			return fn, nil
		}})

	cls.Dict.SetStr("push", &object.BuiltinFunc{Name: "push",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			fn := a[1]
			if exitMethod, err := ii.getAttr(fn, "__exit__"); err == nil {
				*cbsPtr = append(*cbsPtr, exitStackCb{kind: exitStackCbExit, fn: exitMethod})
			} else {
				*cbsPtr = append(*cbsPtr, exitStackCb{kind: exitStackCbExit, fn: fn})
			}
			return fn, nil
		}})

	cls.Dict.SetStr("pop_all", &object.BuiltinFunc{Name: "pop_all",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			// Transfer our callbacks to a new stack and clear ours.
			transferred := append([]exitStackCb(nil), *cbsPtr...)
			*cbsPtr = nil
			return ctxlibMakeExitStackWith(ii, &transferred, async), nil
		}})

	cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			runCallbacks(object.None, object.None, object.None)
			return object.None, nil
		}})

	return inst
}

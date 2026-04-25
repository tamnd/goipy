package vm

import (
	"sync"
	"sync/atomic"

	"github.com/tamnd/goipy/object"
)

// ─── cvContext: the Go-level context map ─────────────────────────────────────

type cvContext struct {
	mu      sync.RWMutex
	vals    map[int64]object.Object // ContextVar.id → value
	varObjs map[int64]object.Object // ContextVar.id → Python ContextVar instance
}

func newCvContext() *cvContext {
	return &cvContext{
		vals:    make(map[int64]object.Object),
		varObjs: make(map[int64]object.Object),
	}
}

func (c *cvContext) clone() *cvContext {
	c.mu.RLock()
	defer c.mu.RUnlock()
	nc := &cvContext{
		vals:    make(map[int64]object.Object, len(c.vals)),
		varObjs: make(map[int64]object.Object, len(c.varObjs)),
	}
	for k, v := range c.vals {
		nc.vals[k] = v
	}
	for k, v := range c.varObjs {
		nc.varObjs[k] = v
	}
	return nc
}

// ─── ContextVar id counter ───────────────────────────────────────────────────

var cvIDCounter atomic.Int64

// ─── currentCtx returns the active context for the given interpreter ─────────

func cvCurrentCtx(ii *Interp) *cvContext {
	if len(ii.ctxStack) == 0 {
		root := newCvContext()
		ii.ctxStack = append(ii.ctxStack, root)
	}
	return ii.ctxStack[len(ii.ctxStack)-1]
}

// interpFrom casts the interp any arg to *Interp.
func interpFrom(interp any) *Interp {
	if ii, ok := interp.(*Interp); ok {
		return ii
	}
	return nil
}

// ─── buildContextvars ────────────────────────────────────────────────────────

func (i *Interp) buildContextvars() *object.Module {
	m := &object.Module{Name: "contextvars", Dict: object.NewDict()}

	// Token.MISSING sentinel.
	missingCls := &object.Class{Name: "_MissingType", Dict: object.NewDict()}
	missing := &object.Instance{Class: missingCls, Dict: object.NewDict()}

	// ─── ContextVar ───────────────────────────────────────────────────────
	m.Dict.SetStr("ContextVar", &object.BuiltinFunc{Name: "ContextVar",
		Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			if len(a) == 0 {
				return nil, object.Errorf(ii.typeErr, "ContextVar() requires name argument")
			}
			nameStr, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(ii.typeErr, "ContextVar name must be a str")
			}
			var defaultVal object.Object
			if kw != nil {
				if v, ok2 := kw.GetStr("default"); ok2 {
					defaultVal = v
				}
			}
			id := cvIDCounter.Add(1)
			return cvMakeVar(id, nameStr.V, defaultVal, missing), nil
		}})

	// ─── copy_context() ───────────────────────────────────────────────────
	m.Dict.SetStr("copy_context", &object.BuiltinFunc{Name: "copy_context",
		Call: func(interp any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			ctx := cvCurrentCtx(ii).clone()
			return cvMakeContext(ctx, missing, ii), nil
		}})

	// ─── Context (empty constructor) ──────────────────────────────────────
	m.Dict.SetStr("Context", &object.BuiltinFunc{Name: "Context",
		Call: func(interp any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interpFrom(interp)
			if ii == nil {
				ii = i
			}
			return cvMakeContext(newCvContext(), missing, ii), nil
		}})

	// ─── Token class (for isinstance / Token.MISSING) ─────────────────────
	tokCls := &object.Class{Name: "Token", Dict: object.NewDict()}
	tokCls.Dict.SetStr("MISSING", missing)
	m.Dict.SetStr("Token", tokCls)

	return m
}

// ─── ContextVar instance ──────────────────────────────────────────────────────

// cvMakeVar builds a Python ContextVar instance.
func cvMakeVar(id int64, name string, defaultVal object.Object, missing *object.Instance) *object.Instance {
	cls := &object.Class{Name: "ContextVar", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	inst.Dict.SetStr("name",   &object.Str{V: name})
	inst.Dict.SetStr("_cv_id", object.NewInt(id))

	// get([default]) — uses the calling interpreter's context stack
	cls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get",
		Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
			ii, ok := interp.(*Interp)
			if !ok {
				return nil, object.Errorf(nil, "get: no interpreter")
			}
			a = mpArgs(a)
			ctx := cvCurrentCtx(ii)
			ctx.mu.RLock()
			val, found := ctx.vals[id]
			ctx.mu.RUnlock()
			if found {
				return val, nil
			}
			if len(a) > 0 {
				return a[0], nil
			}
			if defaultVal != nil {
				return defaultVal, nil
			}
			return nil, object.Errorf(ii.lookupErr, "ContextVar '%s' has no value", name)
		}})

	// set(value) — uses calling interpreter's context; also a context manager
	cls.Dict.SetStr("set", &object.BuiltinFunc{Name: "set",
		Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
			ii, ok := interp.(*Interp)
			if !ok {
				return nil, object.Errorf(nil, "set: no interpreter")
			}
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(ii.typeErr, "set() requires a value")
			}
			newVal := a[0]
			ctx := cvCurrentCtx(ii)

			ctx.mu.Lock()
			oldVal, hadOld := ctx.vals[id]
			ctx.vals[id] = newVal
			ctx.varObjs[id] = inst
			ctx.mu.Unlock()

			var oldObj object.Object
			if hadOld {
				oldObj = oldVal
			} else {
				oldObj = missing
			}
			return cvMakeToken(inst, oldObj, id, missing, ii), nil
		}})

	// reset(token) — uses calling interpreter's context
	cls.Dict.SetStr("reset", &object.BuiltinFunc{Name: "reset",
		Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
			ii, ok := interp.(*Interp)
			if !ok {
				return nil, object.Errorf(nil, "reset: no interpreter")
			}
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(ii.typeErr, "reset() requires a token")
			}
			tok, ok2 := a[0].(*object.Instance)
			if !ok2 {
				return nil, object.Errorf(ii.typeErr, "reset() argument must be a Token")
			}
			return object.None, cvApplyReset(tok, id, missing, ii)
		}})

	return inst
}

// cvApplyReset restores the context variable using the token on the given interp.
func cvApplyReset(tok *object.Instance, varID int64, missing *object.Instance, ii *Interp) error {
	usedObj, ok := tok.Dict.GetStr("_used")
	if ok {
		if b, ok2 := usedObj.(*object.Bool); ok2 && b.V {
			return object.Errorf(ii.runtimeErr, "Token has already been used once")
		}
	}
	tok.Dict.SetStr("_used", object.True)

	oldValObj, hasOld := tok.Dict.GetStr("old_value")
	if !hasOld {
		return nil
	}

	ctx := cvCurrentCtx(ii)
	ctx.mu.Lock()
	if oldValObj == missing {
		delete(ctx.vals, varID)
		delete(ctx.varObjs, varID)
	} else {
		ctx.vals[varID] = oldValObj
	}
	ctx.mu.Unlock()
	return nil
}

// ─── Token instance ───────────────────────────────────────────────────────────

func cvMakeToken(varInst *object.Instance, oldVal object.Object, varID int64, missing *object.Instance, ii *Interp) *object.Instance {
	cls := &object.Class{Name: "Token", Dict: object.NewDict()}
	tok := &object.Instance{Class: cls, Dict: object.NewDict()}

	tok.Dict.SetStr("var",       varInst)
	tok.Dict.SetStr("old_value", oldVal)
	tok.Dict.SetStr("_var_id",   object.NewInt(varID))
	tok.Dict.SetStr("_used",     object.False)
	cls.Dict.SetStr("MISSING",   missing)

	// Context manager: __enter__ returns self, __exit__ resets.
	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return tok, nil
		}})
	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
		Call: func(interp any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			execII, ok := interp.(*Interp)
			if !ok {
				execII = ii
			}
			return object.False, cvApplyReset(tok, varID, missing, execII)
		}})

	return tok
}

// ─── Context instance ────────────────────────────────────────────────────────

func cvMakeContext(ctx *cvContext, missing *object.Instance, buildInterp *Interp) *object.Instance {
	cls := &object.Class{Name: "Context", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}

	// Helper: extract ContextVar id from a Python object.
	getVarID := func(v object.Object) (int64, bool) {
		vi, ok := v.(*object.Instance)
		if !ok {
			return 0, false
		}
		idObj, ok2 := vi.Dict.GetStr("_cv_id")
		if !ok2 {
			return 0, false
		}
		n, ok3 := toInt64(idObj)
		return n, ok3
	}

	// run(callable, *args, **kwargs) — pushes ctx onto the calling interp's stack
	cls.Dict.SetStr("run", &object.BuiltinFunc{Name: "run",
		Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
			ii, ok := interp.(*Interp)
			if !ok {
				ii = buildInterp
			}
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(ii.typeErr, "Context.run() requires a callable")
			}
			callable := a[0]
			callArgs := append([]object.Object(nil), a[1:]...)

			// Push a working copy of ctx; changes propagate back to ctx on exit.
			child := ctx.clone()
			ii.ctxStack = append(ii.ctxStack, child)
			result, err := ii.callObject(callable, callArgs, kw)
			ii.ctxStack = ii.ctxStack[:len(ii.ctxStack)-1]

			// Merge child's changes back into ctx (per CPython: run() changes are
			// visible in the context object after the call).
			ctx.mu.Lock()
			child.mu.RLock()
			for k, v := range child.vals {
				ctx.vals[k] = v
			}
			for k, v := range child.varObjs {
				ctx.varObjs[k] = v
			}
			child.mu.RUnlock()
			ctx.mu.Unlock()

			return result, err
		}})

	// copy()
	cls.Dict.SetStr("copy", &object.BuiltinFunc{Name: "copy",
		Call: func(interp any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ii, ok := interp.(*Interp)
			if !ok {
				ii = buildInterp
			}
			return cvMakeContext(ctx.clone(), missing, ii), nil
		}})

	// __contains__ (var in ctx)
	cls.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return object.False, nil
			}
			vid, ok := getVarID(a[0])
			if !ok {
				return object.False, nil
			}
			ctx.mu.RLock()
			_, found := ctx.vals[vid]
			ctx.mu.RUnlock()
			return object.BoolOf(found), nil
		}})

	// __getitem__ (ctx[var])
	cls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii, ok := interp.(*Interp)
			if !ok {
				ii = buildInterp
			}
			a = mpArgs(a)
			if len(a) == 0 {
				return nil, object.Errorf(ii.typeErr, "Context.__getitem__ requires a key")
			}
			vid, ok2 := getVarID(a[0])
			if !ok2 {
				return nil, object.Errorf(ii.keyErr, "not a ContextVar")
			}
			ctx.mu.RLock()
			val, found := ctx.vals[vid]
			ctx.mu.RUnlock()
			if !found {
				return nil, object.Errorf(ii.keyErr, "ContextVar not set in context")
			}
			return val, nil
		}})

	// get(var[, default])
	cls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) == 0 {
				return object.None, nil
			}
			vid, ok := getVarID(a[0])
			if !ok {
				return object.None, nil
			}
			ctx.mu.RLock()
			val, found := ctx.vals[vid]
			ctx.mu.RUnlock()
			if found {
				return val, nil
			}
			if len(a) > 1 {
				return a[1], nil
			}
			return object.None, nil
		}})

	// __len__
	cls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ctx.mu.RLock()
			n := len(ctx.vals)
			ctx.mu.RUnlock()
			return object.NewInt(int64(n)), nil
		}})

	// __iter__
	cls.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ctx.mu.RLock()
			keys := make([]object.Object, 0, len(ctx.varObjs))
			for _, v := range ctx.varObjs {
				keys = append(keys, v)
			}
			ctx.mu.RUnlock()
			idx := 0
			return &object.Iter{Next: func() (object.Object, bool, error) {
				if idx >= len(keys) {
					return nil, false, nil
				}
				v := keys[idx]
				idx++
				return v, true, nil
			}}, nil
		}})

	// keys()
	cls.Dict.SetStr("keys", &object.BuiltinFunc{Name: "keys",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ctx.mu.RLock()
			keys := make([]object.Object, 0, len(ctx.varObjs))
			for _, v := range ctx.varObjs {
				keys = append(keys, v)
			}
			ctx.mu.RUnlock()
			return &object.List{V: keys}, nil
		}})

	// values()
	cls.Dict.SetStr("values", &object.BuiltinFunc{Name: "values",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ctx.mu.RLock()
			vals := make([]object.Object, 0, len(ctx.vals))
			for _, v := range ctx.vals {
				vals = append(vals, v)
			}
			ctx.mu.RUnlock()
			return &object.List{V: vals}, nil
		}})

	// items()
	cls.Dict.SetStr("items", &object.BuiltinFunc{Name: "items",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ctx.mu.RLock()
			items := make([]object.Object, 0, len(ctx.vals))
			for id, val := range ctx.vals {
				varObj, ok2 := ctx.varObjs[id]
				if !ok2 {
					continue
				}
				items = append(items, &object.Tuple{V: []object.Object{varObj, val}})
			}
			ctx.mu.RUnlock()
			return &object.List{V: items}, nil
		}})

	return inst
}

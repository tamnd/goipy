package vm

import (
	"strings"

	"github.com/tamnd/goipy/object"
	"github.com/tamnd/goipy/op"
)

// callObject invokes callable with positional and keyword args.
func (i *Interp) callObject(callable object.Object, args []object.Object, kwargs *object.Dict) (object.Object, error) {
	switch fn := callable.(type) {
	case *object.BuiltinFunc:
		return fn.Call(i, args, kwargs)
	case *object.Function:
		return i.callFunction(fn, args, kwargs)
	case *object.BoundMethod:
		newArgs := make([]object.Object, 0, len(args)+1)
		newArgs = append(newArgs, fn.Self)
		newArgs = append(newArgs, args...)
		return i.callObject(fn.Fn, newArgs, kwargs)
	case *object.Class:
		// Exception subclasses produce *object.Exception directly.
		if object.IsSubclass(fn, i.baseExc) {
			exc := &object.Exception{Class: fn, Args: &object.Tuple{V: append([]object.Object{}, args...)}}
			if len(args) == 1 {
				if s, ok := args[0].(*object.Str); ok {
					exc.Msg = s.V
				} else {
					exc.Msg = object.Str_(args[0])
				}
			}
			return exc, nil
		}
		inst := &object.Instance{Class: fn, Dict: object.NewDict()}
		if init, ok := classLookup(fn, "__init__"); ok {
			initArgs := append([]object.Object{inst}, args...)
			if _, err := i.callObject(init, initArgs, kwargs); err != nil {
				return nil, err
			}
		}
		return inst, nil
	case *object.Instance:
		if r, ok, err := i.callInstanceDunder(fn, "__call__", args...); ok {
			return r, err
		}
	}
	return nil, object.Errorf(i.typeErr, "'%s' object is not callable", object.TypeName(callable))
}

// callFunctionFast is a hot-path variant of callFunction used when the caller
// has already verified that fn takes exactly positional args, no defaults or
// kwargs are involved, and there is no *args/**kwargs. self may be nil.
// It skips bindArgs entirely and copies directly into Fast.
func (i *Interp) callFunctionFast(fn *object.Function, self object.Object, args []object.Object) (object.Object, error) {
	code := fn.Code
	isGen := code.Flags&(CO_GENERATOR|CO_COROUTINE|CO_ITERABLE_COROUTINE) != 0
	var frame *Frame
	if !isGen {
		if f, ok := i.framePool[code]; ok {
			delete(i.framePool, code)
			frame = f
			frame.Globals = fn.Globals
			frame.Builtins = i.Builtins
			// Fast slice: clear any stale entries so GC can reclaim and
			// so unset slots read as nil. `clear` compiles to memclr.
			clear(frame.Fast)
			frame.SP = 0
			frame.IP = 0
			frame.LastIP = 0
			frame.Back = nil
			frame.ExcInfo = nil
			frame.curExc = nil
			frame.Yielded = nil
			// Cells may have been captured by escaped closures; allocate fresh.
			if code.NCells+code.NFrees > 0 {
				frame.Cells = make([]*object.Cell, code.NCells+code.NFrees)
			} else {
				frame.Cells = nil
			}
		}
	}
	if frame == nil {
		frame = NewFrame(code, fn.Globals, i.Builtins, nil)
	}
	if code.Flags&CO_OPTIMIZED != 0 {
		frame.Locals = nil
	} else {
		frame.Locals = object.NewDict()
	}
	off := 0
	if self != nil {
		frame.Fast[0] = self
		off = 1
	}
	copy(frame.Fast[off:off+len(args)], args)
	if fn.Closure != nil {
		base := code.NLocals + code.NCells
		for k, cell := range fn.Closure.V {
			if c, ok := cell.(*object.Cell); ok {
				frame.Fast[base+k] = c
			}
		}
	}
	if isGen {
		return &object.Generator{Name: fn.Name, Frame: frame}, nil
	}
	r, err := i.runFrame(frame)
	// Return frame to pool unless something kept a reference (generator
	// branch above returned early; tracebacks can retain frames via
	// exception info — skip pooling when err != nil to be safe).
	if err == nil {
		if _, exists := i.framePool[code]; !exists {
			i.framePool[code] = frame
		}
	}
	return r, err
}

// isFastCallable reports whether fn can be invoked through callFunctionFast
// given nArgsTotal (including any bound self): same arity, no *args/**kwargs,
// no keyword-only args.
func isFastCallable(fn *object.Function, nArgsTotal int) bool {
	code := fn.Code
	if code.Flags&(CO_VARARGS|CO_VARKWDS) != 0 {
		return false
	}
	if code.KwOnlyArgCount != 0 {
		return false
	}
	return nArgsTotal == code.ArgCount
}

// callFunction builds a fresh frame for fn and runs it.
func (i *Interp) callFunction(fn *object.Function, args []object.Object, kwargs *object.Dict) (object.Object, error) {
	code := fn.Code
	frame := NewFrame(code, fn.Globals, i.Builtins, nil)
	if code.Flags&CO_OPTIMIZED != 0 {
		frame.Locals = nil // function uses fast locals, no locals dict
	} else {
		frame.Locals = object.NewDict()
	}
	// Bind arguments.
	if err := i.bindArgs(fn, frame, args, kwargs); err != nil {
		return nil, err
	}
	if code.Flags&(CO_GENERATOR|CO_COROUTINE|CO_ITERABLE_COROUTINE) != 0 {
		return &object.Generator{Name: fn.Name, Frame: frame}, nil
	}
	return i.runFrame(frame)
}

// CO_OPTIMIZED etc. mirror the CPython co_flags bits we need.
const (
	CO_OPTIMIZED = 0x0001
	CO_NEWLOCALS = 0x0002
	CO_VARARGS   = 0x0004
	CO_VARKWDS   = 0x0008
	CO_NESTED    = 0x0010
	CO_GENERATOR         = 0x0020
	CO_COROUTINE         = 0x0080
	CO_ITERABLE_COROUTINE = 0x0100
	CO_ASYNC_GENERATOR   = 0x0200
)

func (i *Interp) bindArgs(fn *object.Function, frame *Frame, args []object.Object, kwargs *object.Dict) error {
	code := fn.Code
	narg := code.ArgCount
	nkwonly := code.KwOnlyArgCount
	hasVarargs := code.Flags&CO_VARARGS != 0
	hasVarkwargs := code.Flags&CO_VARKWDS != 0

	// argSlot[i] lives in frame.Fast[i] for i in [0, narg+nkwonly), then *args,
	// then **kwargs.
	given := args
	// positionals
	nPos := narg
	if len(given) > nPos && !hasVarargs {
		return object.Errorf(i.typeErr,
			"%s() takes %d positional arguments but %d were given", fn.Name, narg, len(given))
	}
	usedPos := nPos
	if len(given) < usedPos {
		usedPos = len(given)
	}
	for k := 0; k < usedPos; k++ {
		frame.Fast[k] = given[k]
	}
	extraPos := given[usedPos:]
	// *args
	varargIdx := narg + nkwonly
	if hasVarargs {
		frame.Fast[varargIdx] = &object.Tuple{V: append([]object.Object{}, extraPos...)}
	}
	// **kwargs
	varkwIdx := varargIdx
	if hasVarargs {
		varkwIdx++
	}
	varkw := object.NewDict()
	if hasVarkwargs {
		frame.Fast[varkwIdx] = varkw
	}
	// kwargs
	if kwargs != nil {
		keys, vals := kwargs.Items()
		for k, key := range keys {
			name := key.(*object.Str).V
			slot := -1
			for ai := 0; ai < narg+nkwonly; ai++ {
				if code.LocalsPlusNames[ai] == name {
					slot = ai
					break
				}
			}
			if slot >= 0 {
				if frame.Fast[slot] != nil {
					return object.Errorf(i.typeErr,
						"%s() got multiple values for argument '%s'", fn.Name, name)
				}
				frame.Fast[slot] = vals[k]
			} else if hasVarkwargs {
				varkw.SetStr(name, vals[k])
			} else {
				return object.Errorf(i.typeErr,
					"%s() got an unexpected keyword argument '%s'", fn.Name, name)
			}
		}
	}
	// defaults for positionals
	if fn.Defaults != nil {
		defaults := fn.Defaults.V
		nDef := len(defaults)
		// defaults apply to last nDef positionals
		for k := 0; k < nDef; k++ {
			slot := narg - nDef + k
			if frame.Fast[slot] == nil {
				frame.Fast[slot] = defaults[k]
			}
		}
	}
	// kw-only defaults
	if fn.KwDefaults != nil {
		for k := narg; k < narg+nkwonly; k++ {
			name := code.LocalsPlusNames[k]
			if frame.Fast[k] == nil {
				if v, ok := fn.KwDefaults.GetStr(name); ok {
					frame.Fast[k] = v
				}
			}
		}
	}
	// Check required args
	for k := 0; k < narg+nkwonly; k++ {
		if frame.Fast[k] == nil {
			return object.Errorf(i.typeErr,
				"%s() missing required argument: '%s'", fn.Name, code.LocalsPlusNames[k])
		}
	}
	// Closure: copy free cells into Fast at the free-var slots.
	if fn.Closure != nil {
		base := code.NLocals + code.NCells
		for k, cell := range fn.Closure.V {
			if c, ok := cell.(*object.Cell); ok {
				frame.Fast[base+k] = c
			}
		}
	}
	return nil
}

// --- intrinsics ---

func (i *Interp) intrinsic1(idx int, v object.Object) (object.Object, error) {
	switch idx {
	case op.INTRINSIC_1_INVALID:
		return nil, object.Errorf(i.runtimeErr, "invalid intrinsic")
	case op.INTRINSIC_PRINT:
		// Used by REPL for auto-print; we ignore.
		return v, nil
	case op.INTRINSIC_UNARY_POSITIVE:
		if inst, ok := v.(*object.Instance); ok {
			if r, ok, err := i.callInstanceDunder(inst, "__pos__"); ok {
				return r, err
			}
		}
		switch x := v.(type) {
		case *object.Int, *object.Float:
			return x, nil
		case *object.Bool:
			if x.V {
				return object.NewInt(1), nil
			}
			return object.NewInt(0), nil
		}
		return nil, object.Errorf(i.typeErr, "bad operand for unary +")
	case op.INTRINSIC_LIST_TO_TUPLE:
		if l, ok := v.(*object.List); ok {
			return &object.Tuple{V: append([]object.Object{}, l.V...)}, nil
		}
		return nil, object.Errorf(i.typeErr, "expected list")
	case op.INTRINSIC_STOPITERATION_ERROR:
		// PEP 479: if a generator body lets a StopIteration leak, convert
		// it to RuntimeError. Any other exception passes through untouched
		// for the following RERAISE to propagate.
		if exc, ok := v.(*object.Exception); ok && object.IsSubclass(exc.Class, i.stopIter) {
			return object.NewException(i.runtimeErr, "generator raised StopIteration"), nil
		}
		return v, nil
	}
	return nil, object.Errorf(i.notImpl, "intrinsic %d not implemented", idx)
}

func (i *Interp) intrinsic2(idx int, a, b object.Object) (object.Object, error) {
	return nil, object.Errorf(i.notImpl, "intrinsic2 %d not implemented", idx)
}

// --- methods ---

func strMethod(s *object.Str, name string) (object.Object, bool) {
	switch name {
	case "upper":
		return &object.BuiltinFunc{Name: "upper", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: strings.ToUpper(s.V)}, nil
		}}, true
	case "lower":
		return &object.BuiltinFunc{Name: "lower", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: strings.ToLower(s.V)}, nil
		}}, true
	case "strip":
		return &object.BuiltinFunc{Name: "strip", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: strings.TrimSpace(s.V)}, nil
		}}, true
	case "split":
		return &object.BuiltinFunc{Name: "split", Call: func(i_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			sep := ""
			if len(a) > 0 {
				if ss, ok := a[0].(*object.Str); ok {
					sep = ss.V
				}
			}
			var parts []string
			if sep == "" {
				parts = strings.Fields(s.V)
			} else {
				parts = strings.Split(s.V, sep)
			}
			out := make([]object.Object, len(parts))
			for k, p := range parts {
				out[k] = &object.Str{V: p}
			}
			return &object.List{V: out}, nil
		}}, true
	case "join":
		return &object.BuiltinFunc{Name: "join", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(ii.(*Interp).typeErr, "join expects 1 argument")
			}
			items, err := iterate(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			parts := make([]string, len(items))
			for k, x := range items {
				sx, ok := x.(*object.Str)
				if !ok {
					return nil, object.Errorf(ii.(*Interp).typeErr, "join requires str")
				}
				parts[k] = sx.V
			}
			return &object.Str{V: strings.Join(parts, s.V)}, nil
		}}, true
	case "replace":
		return &object.BuiltinFunc{Name: "replace", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(ii.(*Interp).typeErr, "replace needs 2 args")
			}
			old := a[0].(*object.Str).V
			new_ := a[1].(*object.Str).V
			return &object.Str{V: strings.ReplaceAll(s.V, old, new_)}, nil
		}}, true
	case "startswith":
		return &object.BuiltinFunc{Name: "startswith", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.False, nil
			}
			return object.BoolOf(strings.HasPrefix(s.V, a[0].(*object.Str).V)), nil
		}}, true
	case "endswith":
		return &object.BuiltinFunc{Name: "endswith", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.False, nil
			}
			return object.BoolOf(strings.HasSuffix(s.V, a[0].(*object.Str).V)), nil
		}}, true
	case "find":
		return &object.BuiltinFunc{Name: "find", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.NewInt(-1), nil
			}
			idx := strings.Index(s.V, a[0].(*object.Str).V)
			return object.NewInt(int64(idx)), nil
		}}, true
	case "count":
		return &object.BuiltinFunc{Name: "count", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.NewInt(0), nil
			}
			return object.NewInt(int64(strings.Count(s.V, a[0].(*object.Str).V))), nil
		}}, true
	case "format":
		return &object.BuiltinFunc{Name: "format", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: s.V}, nil
		}}, true
	}
	return nil, false
}

// Shared package-level BuiltinFuncs for hot list methods. These receive self
// as args[0] so they can be wrapped in a BoundMethod without allocating a
// per-list closure. The CALL dispatch has a fast path that forwards
// BoundMethod{Fn:*BuiltinFunc} without allocating a new args slice.
var sharedListAppend = &object.BuiltinFunc{Name: "append", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
	l := a[0].(*object.List)
	l.V = append(l.V, a[1])
	return object.None, nil
}}

func listMethod(l *object.List, name string) (object.Object, bool) {
	switch name {
	case "append":
		return &object.BoundMethod{Self: l, Fn: sharedListAppend}, true
	case "extend":
		return &object.BuiltinFunc{Name: "extend", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			items, err := iterate(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			l.V = append(l.V, items...)
			return object.None, nil
		}}, true
	case "pop":
		return &object.BuiltinFunc{Name: "pop", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			idx := len(l.V) - 1
			if len(a) > 0 {
				if n, ok := toInt64(a[0]); ok {
					idx = int(n)
					if idx < 0 {
						idx += len(l.V)
					}
				}
			}
			if idx < 0 || idx >= len(l.V) {
				return nil, object.Errorf(ii.(*Interp).indexErr, "pop index out of range")
			}
			v := l.V[idx]
			l.V = append(l.V[:idx], l.V[idx+1:]...)
			return v, nil
		}}, true
	case "insert":
		return &object.BuiltinFunc{Name: "insert", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			n, _ := toInt64(a[0])
			idx := int(n)
			if idx < 0 {
				idx += len(l.V)
			}
			if idx < 0 {
				idx = 0
			}
			if idx > len(l.V) {
				idx = len(l.V)
			}
			l.V = append(l.V[:idx], append([]object.Object{a[1]}, l.V[idx:]...)...)
			return object.None, nil
		}}, true
	case "remove":
		return &object.BuiltinFunc{Name: "remove", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			for k, x := range l.V {
				eq, err := object.Eq(x, a[0])
				if err != nil {
					return nil, err
				}
				if eq {
					l.V = append(l.V[:k], l.V[k+1:]...)
					return object.None, nil
				}
			}
			return nil, object.Errorf(ii.(*Interp).valueErr, "list.remove(x): x not in list")
		}}, true
	case "clear":
		return &object.BuiltinFunc{Name: "clear", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			l.V = l.V[:0]
			return object.None, nil
		}}, true
	case "reverse":
		return &object.BuiltinFunc{Name: "reverse", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			for k, j := 0, len(l.V)-1; k < j; k, j = k+1, j-1 {
				l.V[k], l.V[j] = l.V[j], l.V[k]
			}
			return object.None, nil
		}}, true
	case "sort":
		return &object.BuiltinFunc{Name: "sort", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, sortList(ii.(*Interp), l.V, false)
		}}, true
	case "copy":
		return &object.BuiltinFunc{Name: "copy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			cp := make([]object.Object, len(l.V))
			copy(cp, l.V)
			return &object.List{V: cp}, nil
		}}, true
	case "count":
		return &object.BuiltinFunc{Name: "count", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			n := int64(0)
			for _, x := range l.V {
				eq, err := object.Eq(x, a[0])
				if err != nil {
					return nil, err
				}
				if eq {
					n++
				}
			}
			return object.NewInt(n), nil
		}}, true
	case "index":
		return &object.BuiltinFunc{Name: "index", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			for k, x := range l.V {
				eq, err := object.Eq(x, a[0])
				if err != nil {
					return nil, err
				}
				if eq {
					return object.NewInt(int64(k)), nil
				}
			}
			return nil, object.Errorf(ii.(*Interp).valueErr, "not in list")
		}}, true
	}
	return nil, false
}

func dictMethod(d *object.Dict, name string) (object.Object, bool) {
	switch name {
	case "get":
		return &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.None, nil
			}
			v, ok, err := d.Get(a[0])
			if err != nil {
				return nil, err
			}
			if !ok {
				if len(a) > 1 {
					return a[1], nil
				}
				return object.None, nil
			}
			return v, nil
		}}, true
	case "setdefault":
		return &object.BuiltinFunc{Name: "setdefault", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			v, ok, err := d.Get(a[0])
			if err != nil {
				return nil, err
			}
			if ok {
				return v, nil
			}
			var def object.Object = object.None
			if len(a) > 1 {
				def = a[1]
			}
			if err := d.Set(a[0], def); err != nil {
				return nil, err
			}
			return def, nil
		}}, true
	case "keys":
		return &object.BuiltinFunc{Name: "keys", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			keys, _ := d.Items()
			out := make([]object.Object, len(keys))
			copy(out, keys)
			return &object.List{V: out}, nil
		}}, true
	case "values":
		return &object.BuiltinFunc{Name: "values", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			_, vals := d.Items()
			out := make([]object.Object, len(vals))
			copy(out, vals)
			return &object.List{V: out}, nil
		}}, true
	case "items":
		return &object.BuiltinFunc{Name: "items", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			keys, vals := d.Items()
			out := make([]object.Object, len(keys))
			for k := range keys {
				out[k] = &object.Tuple{V: []object.Object{keys[k], vals[k]}}
			}
			return &object.List{V: out}, nil
		}}, true
	case "update":
		return &object.BuiltinFunc{Name: "update", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 1 {
				if sd, ok := a[0].(*object.Dict); ok {
					ks, vs := sd.Items()
					for k, key := range ks {
						if err := d.Set(key, vs[k]); err != nil {
							return nil, err
						}
					}
				}
			}
			return object.None, nil
		}}, true
	case "pop":
		return &object.BuiltinFunc{Name: "pop", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(ii.(*Interp).typeErr, "pop expected at least 1 argument")
			}
			v, ok, err := d.Get(a[0])
			if err != nil {
				return nil, err
			}
			if ok {
				_, _ = d.Delete(a[0])
				return v, nil
			}
			if len(a) > 1 {
				return a[1], nil
			}
			return nil, object.Errorf(ii.(*Interp).keyErr, "%s", object.Repr(a[0]))
		}}, true
	case "clear":
		return &object.BuiltinFunc{Name: "clear", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			keys, _ := d.Items()
			for _, k := range keys {
				_, _ = d.Delete(k)
			}
			return object.None, nil
		}}, true
	case "copy":
		return &object.BuiltinFunc{Name: "copy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			nd := object.NewDict()
			keys, vals := d.Items()
			for k, key := range keys {
				_ = nd.Set(key, vals[k])
			}
			return nd, nil
		}}, true
	}
	return nil, false
}

// bytesSubMethod returns read-only methods common to bytes and bytearray.
// `mutable` controls whether replace/split return bytearray or bytes.
func bytesSubMethod(data func() []byte, mutable bool, name string) (object.Object, bool) {
	wrap := func(b []byte) object.Object {
		if mutable {
			return &object.Bytearray{V: b}
		}
		return &object.Bytes{V: b}
	}
	needleBytes := func(o object.Object) ([]byte, bool) {
		return bytesBytesOrArray(o)
	}
	switch name {
	case "count":
		return &object.BuiltinFunc{Name: "count", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			nd, ok := needleBytes(a[0])
			if !ok {
				return object.NewInt(0), nil
			}
			hay := data()
			if len(nd) == 0 {
				return object.NewInt(int64(len(hay) + 1)), nil
			}
			n := 0
			for i := 0; i+len(nd) <= len(hay); {
				if bytesEqAt(hay, i, nd) {
					n++
					i += len(nd)
				} else {
					i++
				}
			}
			return object.NewInt(int64(n)), nil
		}}, true
	case "find":
		return &object.BuiltinFunc{Name: "find", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			nd, _ := needleBytes(a[0])
			return object.NewInt(int64(bytesIndex(data(), nd))), nil
		}}, true
	case "index":
		return &object.BuiltinFunc{Name: "index", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			nd, _ := needleBytes(a[0])
			idx := bytesIndex(data(), nd)
			if idx < 0 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "subsection not found")
			}
			return object.NewInt(int64(idx)), nil
		}}, true
	case "startswith":
		return &object.BuiltinFunc{Name: "startswith", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			nd, ok := needleBytes(a[0])
			if !ok {
				return object.False, nil
			}
			hay := data()
			if len(nd) > len(hay) {
				return object.False, nil
			}
			return object.BoolOf(bytesEqAt(hay, 0, nd)), nil
		}}, true
	case "endswith":
		return &object.BuiltinFunc{Name: "endswith", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			nd, ok := needleBytes(a[0])
			if !ok {
				return object.False, nil
			}
			hay := data()
			if len(nd) > len(hay) {
				return object.False, nil
			}
			return object.BoolOf(bytesEqAt(hay, len(hay)-len(nd), nd)), nil
		}}, true
	case "replace":
		return &object.BuiltinFunc{Name: "replace", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			old, _ := needleBytes(a[0])
			new_, _ := needleBytes(a[1])
			return wrap(bytesReplace(data(), old, new_)), nil
		}}, true
	case "split":
		return &object.BuiltinFunc{Name: "split", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			var sep []byte
			if len(a) > 0 {
				sep, _ = needleBytes(a[0])
			}
			parts := bytesSplit(data(), sep)
			out := make([]object.Object, len(parts))
			for k, p := range parts {
				out[k] = wrap(p)
			}
			return &object.List{V: out}, nil
		}}, true
	}
	return nil, false
}

func bytesEqAt(hay []byte, i int, needle []byte) bool {
	if i < 0 || i+len(needle) > len(hay) {
		return false
	}
	for j := range needle {
		if hay[i+j] != needle[j] {
			return false
		}
	}
	return true
}

func bytesIndex(hay, needle []byte) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		if bytesEqAt(hay, i, needle) {
			return i
		}
	}
	return -1
}

func bytesReplace(hay, old, new_ []byte) []byte {
	if len(old) == 0 {
		return append([]byte(nil), hay...)
	}
	out := make([]byte, 0, len(hay))
	for i := 0; i < len(hay); {
		if bytesEqAt(hay, i, old) {
			out = append(out, new_...)
			i += len(old)
		} else {
			out = append(out, hay[i])
			i++
		}
	}
	return out
}

func bytesSplit(hay, sep []byte) [][]byte {
	if len(sep) == 0 {
		// split on whitespace runs
		var parts [][]byte
		i := 0
		for i < len(hay) {
			for i < len(hay) && isAsciiSpace(hay[i]) {
				i++
			}
			if i >= len(hay) {
				break
			}
			j := i
			for j < len(hay) && !isAsciiSpace(hay[j]) {
				j++
			}
			parts = append(parts, append([]byte(nil), hay[i:j]...))
			i = j
		}
		return parts
	}
	var parts [][]byte
	i := 0
	for i <= len(hay) {
		idx := bytesIndex(hay[i:], sep)
		if idx < 0 {
			parts = append(parts, append([]byte(nil), hay[i:]...))
			break
		}
		parts = append(parts, append([]byte(nil), hay[i:i+idx]...))
		i += idx + len(sep)
	}
	return parts
}

func isAsciiSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

func memoryviewAttr(mv *object.Memoryview, name string) (object.Object, bool) {
	switch name {
	case "readonly":
		return object.BoolOf(mv.Readonly), true
	case "obj":
		return mv.Backing, true
	case "nbytes":
		return object.NewInt(int64(mv.Stop - mv.Start)), true
	case "format":
		return &object.Str{V: "B"}, true
	case "itemsize":
		return object.NewInt(1), true
	case "tobytes":
		return &object.BuiltinFunc{Name: "tobytes", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Bytes{V: mv.Bytes()}, nil
		}}, true
	case "tolist":
		return &object.BuiltinFunc{Name: "tolist", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			buf := mv.Buf()
			out := make([]object.Object, len(buf))
			for k, b := range buf {
				out[k] = object.NewInt(int64(b))
			}
			return &object.List{V: out}, nil
		}}, true
	case "release":
		return &object.BuiltinFunc{Name: "release", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}, true
	}
	return nil, false
}

func bytearrayMethod(ba *object.Bytearray, name string) (object.Object, bool) {
	if m, ok := bytesSubMethod(func() []byte { return ba.V }, true, name); ok {
		return m, true
	}
	switch name {
	case "clear":
		return &object.BuiltinFunc{Name: "clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ba.V = ba.V[:0]
			return object.None, nil
		}}, true
	case "reverse":
		return &object.BuiltinFunc{Name: "reverse", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			for i, j := 0, len(ba.V)-1; i < j; i, j = i+1, j-1 {
				ba.V[i], ba.V[j] = ba.V[j], ba.V[i]
			}
			return object.None, nil
		}}, true
	case "insert":
		return &object.BuiltinFunc{Name: "insert", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			n, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(ii.(*Interp).typeErr, "bytearray indices must be integers")
			}
			v, ok := toInt64(a[1])
			if !ok || v < 0 || v > 255 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "byte must be in range(0, 256)")
			}
			idx := int(n)
			L := len(ba.V)
			if idx < 0 {
				idx += L
			}
			if idx < 0 {
				idx = 0
			}
			if idx > L {
				idx = L
			}
			ba.V = append(ba.V, 0)
			copy(ba.V[idx+1:], ba.V[idx:])
			ba.V[idx] = byte(v)
			return object.None, nil
		}}, true
	case "remove":
		return &object.BuiltinFunc{Name: "remove", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			v, ok := toInt64(a[0])
			if !ok || v < 0 || v > 255 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "byte must be in range(0, 256)")
			}
			b := byte(v)
			for idx, x := range ba.V {
				if x == b {
					ba.V = append(ba.V[:idx], ba.V[idx+1:]...)
					return object.None, nil
				}
			}
			return nil, object.Errorf(ii.(*Interp).valueErr, "value not found in bytearray")
		}}, true
	case "append":
		return &object.BuiltinFunc{Name: "append", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			n, ok := toInt64(a[0])
			if !ok || n < 0 || n > 255 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "byte must be in range(0, 256)")
			}
			ba.V = append(ba.V, byte(n))
			return object.None, nil
		}}, true
	case "extend":
		return &object.BuiltinFunc{Name: "extend", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if bb, ok := bytesBytesOrArray(a[0]); ok {
				ba.V = append(ba.V, bb...)
				return object.None, nil
			}
			items, err := iterate(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			for _, x := range items {
				n, ok := toInt64(x)
				if !ok || n < 0 || n > 255 {
					return nil, object.Errorf(ii.(*Interp).valueErr, "byte must be in range(0, 256)")
				}
				ba.V = append(ba.V, byte(n))
			}
			return object.None, nil
		}}, true
	case "pop":
		return &object.BuiltinFunc{Name: "pop", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			idx := len(ba.V) - 1
			if len(a) > 0 {
				n, ok := toInt64(a[0])
				if !ok {
					return nil, object.Errorf(ii.(*Interp).typeErr, "bytearray indices must be integers")
				}
				idx = int(n)
				if idx < 0 {
					idx += len(ba.V)
				}
			}
			if idx < 0 || idx >= len(ba.V) {
				return nil, object.Errorf(ii.(*Interp).indexErr, "pop from empty bytearray")
			}
			v := ba.V[idx]
			ba.V = append(ba.V[:idx], ba.V[idx+1:]...)
			return object.NewInt(int64(v)), nil
		}}, true
	case "decode":
		return &object.BuiltinFunc{Name: "decode", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: string(ba.V)}, nil
		}}, true
	case "hex":
		return &object.BuiltinFunc{Name: "hex", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			const digits = "0123456789abcdef"
			buf := make([]byte, 0, 2*len(ba.V))
			for _, c := range ba.V {
				buf = append(buf, digits[c>>4], digits[c&0xf])
			}
			return &object.Str{V: string(buf)}, nil
		}}, true
	}
	return nil, false
}

func setMethod(s *object.Set, name string) (object.Object, bool) {
	switch name {
	case "add":
		return &object.BuiltinFunc{Name: "add", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, s.Add(a[0])
		}}, true
	case "discard":
		return &object.BuiltinFunc{Name: "discard", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}, true
	case "copy":
		return &object.BuiltinFunc{Name: "copy", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			out := object.NewSet()
			for _, x := range s.Items() {
				_ = out.Add(x)
			}
			return out, nil
		}}, true
	}
	if m, ok := setQueryMethod(s, name); ok {
		return m, true
	}
	return nil, false
}

func frozensetMethod(s *object.Frozenset, name string) (object.Object, bool) {
	switch name {
	case "copy":
		return &object.BuiltinFunc{Name: "copy", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return s, nil
		}}, true
	}
	if m, ok := setQueryMethod(s, name); ok {
		return m, true
	}
	return nil, false
}

// setQueryMethod returns methods common to set and frozenset: issubset,
// issuperset, isdisjoint, union, intersection, difference, symmetric_difference.
// Mutating variants (update, intersection_update, ...) are not included.
func setQueryMethod(self object.Object, name string) (object.Object, bool) {
	switch name {
	case "issubset":
		return &object.BuiltinFunc{Name: "issubset", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			other, err := materializeSet(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			for _, x := range setItems(self) {
				if !setContains(other, x) {
					return object.False, nil
				}
			}
			return object.True, nil
		}}, true
	case "issuperset":
		return &object.BuiltinFunc{Name: "issuperset", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			other, err := materializeSet(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			for _, x := range setItems(other) {
				if !setContains(self, x) {
					return object.False, nil
				}
			}
			return object.True, nil
		}}, true
	case "isdisjoint":
		return &object.BuiltinFunc{Name: "isdisjoint", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			other, err := materializeSet(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			for _, x := range setItems(self) {
				if setContains(other, x) {
					return object.False, nil
				}
			}
			return object.True, nil
		}}, true
	case "union":
		return &object.BuiltinFunc{Name: "union", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return setReduce(ii.(*Interp), self, a, "|")
		}}, true
	case "intersection":
		return &object.BuiltinFunc{Name: "intersection", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return setReduce(ii.(*Interp), self, a, "&")
		}}, true
	case "difference":
		return &object.BuiltinFunc{Name: "difference", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return setReduce(ii.(*Interp), self, a, "-")
		}}, true
	case "symmetric_difference":
		return &object.BuiltinFunc{Name: "symmetric_difference", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			other, err := materializeSet(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			return setBitop(self, other, "^"), nil
		}}, true
	}
	return nil, false
}

// materializeSet coerces any iterable into a Set for use by set operations.
func materializeSet(i *Interp, o object.Object) (object.Object, error) {
	if isSetLike(o) {
		return o, nil
	}
	items, err := iterate(i, o)
	if err != nil {
		return nil, err
	}
	s := object.NewSet()
	for _, x := range items {
		if err := s.Add(x); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// setReduce folds op across self and each arg, preserving self's type.
func setReduce(i *Interp, self object.Object, args []object.Object, op string) (object.Object, error) {
	result := self
	for _, a := range args {
		other, err := materializeSet(i, a)
		if err != nil {
			return nil, err
		}
		result = setBitop(result, other, op)
	}
	if len(args) == 0 {
		// Return a copy of self to avoid aliasing.
		return setBitop(self, self, "|"), nil
	}
	return result, nil
}

func tupleMethod(t *object.Tuple, name string) (object.Object, bool) {
	switch name {
	case "count":
		return &object.BuiltinFunc{Name: "count", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			n := int64(0)
			for _, x := range t.V {
				eq, err := object.Eq(x, a[0])
				if err != nil {
					return nil, err
				}
				if eq {
					n++
				}
			}
			return object.NewInt(n), nil
		}}, true
	case "index":
		return &object.BuiltinFunc{Name: "index", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			for k, x := range t.V {
				eq, err := object.Eq(x, a[0])
				if err != nil {
					return nil, err
				}
				if eq {
					return object.NewInt(int64(k)), nil
				}
			}
			return nil, object.Errorf(ii.(*Interp).valueErr, "not in tuple")
		}}, true
	}
	return nil, false
}

// sortList sorts a slice in place using the interpreter's lt.
func sortList(i *Interp, items []object.Object, reverse bool) error {
	return sortListKey(i, items, nil, reverse)
}

// sortListKey sorts items in place, optionally using key(x) for comparisons.
// Stable (insertion sort) and error-propagating — keeping this hand-rolled
// avoids the awkwardness of Go's sort package where the less function can't
// return an error.
func sortListKey(i *Interp, items []object.Object, key object.Object, reverse bool) error {
	keys := items
	if key != nil {
		keys = make([]object.Object, len(items))
		for k, v := range items {
			kv, err := i.callObject(key, []object.Object{v}, nil)
			if err != nil {
				return err
			}
			keys[k] = kv
		}
	}
	for k := 1; k < len(items); k++ {
		for j := k; j > 0; j-- {
			var less bool
			var err error
			if reverse {
				less, err = i.lt(keys[j-1], keys[j])
			} else {
				less, err = i.lt(keys[j], keys[j-1])
			}
			if err != nil {
				return err
			}
			if !less {
				break
			}
			items[j-1], items[j] = items[j], items[j-1]
			if key != nil {
				keys[j-1], keys[j] = keys[j], keys[j-1]
			}
		}
	}
	return nil
}

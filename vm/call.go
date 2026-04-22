package vm

import (
	"math"
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"

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
			ret, err := i.callObject(init, initArgs, kwargs)
			if err != nil {
				return nil, err
			}
			if ret != nil && ret != object.None {
				if _, isNone := ret.(*object.NoneType); !isNone {
					return nil, object.Errorf(i.typeErr, "__init__() should return None, not '%s'", object.TypeName(ret))
				}
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
		if f, ok := code.FramePool.(*Frame); ok && f != nil {
			code.FramePool = nil
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
	if err == nil && code.FramePool == nil {
		code.FramePool = frame
	}
	return r, err
}

// callFunctionFastKw is the hot-path variant for CALL_KW when the callable
// is a *Function with exactly ArgCount+KwOnlyArgCount slots and no *args
// / **kwargs. pos holds positional args (self included if bound), and
// kwnames/kwvals describe keyword args. Skips building a kwargs dict.
func (i *Interp) callFunctionFastKw(fn *object.Function, pos []object.Object, kwnames []object.Object, kwvals []object.Object) (object.Object, error) {
	code := fn.Code
	isGen := code.Flags&(CO_GENERATOR|CO_COROUTINE|CO_ITERABLE_COROUTINE) != 0
	var frame *Frame
	if !isGen {
		if f, ok := code.FramePool.(*Frame); ok && f != nil {
			code.FramePool = nil
			frame = f
			frame.Globals = fn.Globals
			frame.Builtins = i.Builtins
			clear(frame.Fast)
			frame.SP = 0
			frame.IP = 0
			frame.LastIP = 0
			frame.Back = nil
			frame.ExcInfo = nil
			frame.curExc = nil
			frame.Yielded = nil
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
	narg := code.ArgCount
	nkwonly := code.KwOnlyArgCount
	if len(pos) > narg {
		return nil, object.Errorf(i.typeErr, "%s() takes %d positional arguments but %d were given", fn.Name, narg, len(pos))
	}
	copy(frame.Fast[:len(pos)], pos)
	// Populate per-Code kwname→slot map lazily.
	slotMap := code.KwSlot
	if slotMap == nil {
		slotMap = make(map[string]int, narg+nkwonly)
		for k := 0; k < narg+nkwonly; k++ {
			slotMap[code.LocalsPlusNames[k]] = k
		}
		code.KwSlot = slotMap
	}
	for k, nameObj := range kwnames {
		name := nameObj.(*object.Str).V
		slot, ok := slotMap[name]
		if !ok {
			return nil, object.Errorf(i.typeErr, "%s() got an unexpected keyword argument '%s'", fn.Name, name)
		}
		if frame.Fast[slot] != nil {
			return nil, object.Errorf(i.typeErr, "%s() got multiple values for argument '%s'", fn.Name, name)
		}
		frame.Fast[slot] = kwvals[k]
	}
	// defaults for positionals
	if fn.Defaults != nil {
		defaults := fn.Defaults.V
		nDef := len(defaults)
		for k := 0; k < nDef; k++ {
			slot := narg - nDef + k
			if frame.Fast[slot] == nil {
				frame.Fast[slot] = defaults[k]
			}
		}
	}
	if fn.KwDefaults != nil {
		for k := narg; k < narg+nkwonly; k++ {
			if frame.Fast[k] == nil {
				if v, ok := fn.KwDefaults.GetStr(code.LocalsPlusNames[k]); ok {
					frame.Fast[k] = v
				}
			}
		}
	}
	for k := 0; k < narg+nkwonly; k++ {
		if frame.Fast[k] == nil {
			return nil, object.Errorf(i.typeErr, "%s() missing required argument: '%s'", fn.Name, code.LocalsPlusNames[k])
		}
	}
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
	if err == nil && code.FramePool == nil {
		code.FramePool = frame
	}
	return r, err
}

// isFastKwCallable reports whether fn + (nPos, nKw) can be routed through
// callFunctionFastKw.
func isFastKwCallable(fn *object.Function, nPos int) bool {
	code := fn.Code
	if code.Flags&(CO_VARARGS|CO_VARKWDS) != 0 {
		return false
	}
	return nPos <= code.ArgCount
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

// strOptStr extracts optional first arg as string; returns "" if absent or None.
func strOptStr(a []object.Object) (string, bool) {
	if len(a) == 0 {
		return "", false
	}
	if a[0] == object.None {
		return "", false
	}
	if ss, ok := a[0].(*object.Str); ok {
		return ss.V, true
	}
	return "", false
}

// strRuneIndex returns the rune-index of byte-index b in s, or -1.
func strRuneIndex(s string, b int) int {
	if b < 0 {
		return -1
	}
	return utf8.RuneCountInString(s[:b])
}

// strSplitFields splits like Python str.split() with no separator.
func strSplitFields(s string, maxsplit int) []string {
	if maxsplit == 0 {
		return strings.Fields(s)
	}
	var parts []string
	n := 0
	i := 0
	for i < len(s) {
		// skip leading whitespace
		for i < len(s) && unicode.IsSpace(rune(s[i])) {
			i++
		}
		if i >= len(s) {
			break
		}
		if maxsplit > 0 && n >= maxsplit {
			parts = append(parts, s[i:])
			break
		}
		j := i
		for j < len(s) && !unicode.IsSpace(rune(s[j])) {
			j++
		}
		parts = append(parts, s[i:j])
		n++
		i = j
	}
	return parts
}

func strMethod(s *object.Str, name string) (object.Object, bool) {
	str := func(v string) object.Object { return &object.Str{V: v} }
	switch name {
	// --- case ---
	case "upper":
		return &object.BuiltinFunc{Name: "upper", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return str(strings.ToUpper(s.V)), nil
		}}, true
	case "lower":
		return &object.BuiltinFunc{Name: "lower", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return str(strings.ToLower(s.V)), nil
		}}, true
	case "title":
		return &object.BuiltinFunc{Name: "title", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			// Python title: uppercase after any non-cased character.
			prev := true
			var b strings.Builder
			for _, r := range s.V {
				if unicode.IsLetter(r) {
					if prev {
						b.WriteRune(unicode.ToUpper(r))
					} else {
						b.WriteRune(unicode.ToLower(r))
					}
					prev = false
				} else {
					b.WriteRune(r)
					prev = !unicode.IsLetter(r)
				}
			}
			return str(b.String()), nil
		}}, true
	case "swapcase":
		return &object.BuiltinFunc{Name: "swapcase", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			var b strings.Builder
			for _, r := range s.V {
				if unicode.IsUpper(r) {
					b.WriteRune(unicode.ToLower(r))
				} else if unicode.IsLower(r) {
					b.WriteRune(unicode.ToUpper(r))
				} else {
					b.WriteRune(r)
				}
			}
			return str(b.String()), nil
		}}, true
	case "casefold":
		return &object.BuiltinFunc{Name: "casefold", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return str(strings.ToLower(s.V)), nil
		}}, true
	case "capitalize":
		return &object.BuiltinFunc{Name: "capitalize", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if s.V == "" {
				return str(""), nil
			}
			r, size := utf8.DecodeRuneInString(s.V)
			return str(string(unicode.ToUpper(r)) + strings.ToLower(s.V[size:])), nil
		}}, true
	// --- predicates ---
	case "isalpha":
		return &object.BuiltinFunc{Name: "isalpha", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if s.V == "" {
				return object.False, nil
			}
			for _, r := range s.V {
				if !unicode.IsLetter(r) {
					return object.False, nil
				}
			}
			return object.True, nil
		}}, true
	case "isdigit":
		return &object.BuiltinFunc{Name: "isdigit", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if s.V == "" {
				return object.False, nil
			}
			for _, r := range s.V {
				if !unicode.IsDigit(r) {
					return object.False, nil
				}
			}
			return object.True, nil
		}}, true
	case "isnumeric":
		return &object.BuiltinFunc{Name: "isnumeric", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if s.V == "" {
				return object.False, nil
			}
			for _, r := range s.V {
				if !unicode.Is(unicode.N, r) {
					return object.False, nil
				}
			}
			return object.True, nil
		}}, true
	case "isdecimal":
		return &object.BuiltinFunc{Name: "isdecimal", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if s.V == "" {
				return object.False, nil
			}
			for _, r := range s.V {
				// Decimal digits: Unicode category Nd
				if !unicode.Is(unicode.Nd, r) {
					return object.False, nil
				}
			}
			return object.True, nil
		}}, true
	case "isspace":
		return &object.BuiltinFunc{Name: "isspace", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if s.V == "" {
				return object.False, nil
			}
			for _, r := range s.V {
				if !unicode.IsSpace(r) {
					return object.False, nil
				}
			}
			return object.True, nil
		}}, true
	case "isalnum":
		return &object.BuiltinFunc{Name: "isalnum", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if s.V == "" {
				return object.False, nil
			}
			for _, r := range s.V {
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
					return object.False, nil
				}
			}
			return object.True, nil
		}}, true
	case "isupper":
		return &object.BuiltinFunc{Name: "isupper", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			hasCased := false
			for _, r := range s.V {
				if unicode.IsLower(r) {
					return object.False, nil
				}
				if unicode.IsUpper(r) {
					hasCased = true
				}
			}
			return object.BoolOf(hasCased), nil
		}}, true
	case "islower":
		return &object.BuiltinFunc{Name: "islower", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			hasCased := false
			for _, r := range s.V {
				if unicode.IsUpper(r) {
					return object.False, nil
				}
				if unicode.IsLower(r) {
					hasCased = true
				}
			}
			return object.BoolOf(hasCased), nil
		}}, true
	case "istitle":
		return &object.BuiltinFunc{Name: "istitle", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			hasCased := false
			prevCased := false
			for _, r := range s.V {
				if unicode.IsUpper(r) {
					if prevCased {
						return object.False, nil
					}
					hasCased = true
					prevCased = true
				} else if unicode.IsLower(r) {
					if !prevCased {
						return object.False, nil
					}
					hasCased = true
					prevCased = true
				} else {
					prevCased = false
				}
			}
			return object.BoolOf(hasCased), nil
		}}, true
	case "isidentifier":
		return &object.BuiltinFunc{Name: "isidentifier", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if s.V == "" {
				return object.False, nil
			}
			for i, r := range s.V {
				if i == 0 {
					if !unicode.IsLetter(r) && r != '_' {
						return object.False, nil
					}
				} else {
					if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
						return object.False, nil
					}
				}
			}
			return object.True, nil
		}}, true
	case "isprintable":
		return &object.BuiltinFunc{Name: "isprintable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			for _, r := range s.V {
				if !unicode.IsPrint(r) {
					return object.False, nil
				}
			}
			return object.True, nil
		}}, true
	case "isascii":
		return &object.BuiltinFunc{Name: "isascii", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(isASCII(s.V)), nil
		}}, true
	// --- strip ---
	case "strip":
		return &object.BuiltinFunc{Name: "strip", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if chars, ok := strOptStr(a); ok {
				return str(strings.Trim(s.V, chars)), nil
			}
			return str(strings.TrimSpace(s.V)), nil
		}}, true
	case "lstrip":
		return &object.BuiltinFunc{Name: "lstrip", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if chars, ok := strOptStr(a); ok {
				return str(strings.TrimLeft(s.V, chars)), nil
			}
			return str(strings.TrimLeftFunc(s.V, unicode.IsSpace)), nil
		}}, true
	case "rstrip":
		return &object.BuiltinFunc{Name: "rstrip", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if chars, ok := strOptStr(a); ok {
				return str(strings.TrimRight(s.V, chars)), nil
			}
			return str(strings.TrimRightFunc(s.V, unicode.IsSpace)), nil
		}}, true
	// --- split ---
	case "split":
		return &object.BuiltinFunc{Name: "split", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			maxsplit := -1
			sep, hasSep := strOptStr(a)
			if len(a) >= 2 {
				if n, ok := toInt64(a[1]); ok {
					maxsplit = int(n)
				}
			}
			var parts []string
			if !hasSep {
				parts = strSplitFields(s.V, maxsplit)
			} else if maxsplit < 0 {
				parts = strings.Split(s.V, sep)
			} else {
				parts = strings.SplitN(s.V, sep, maxsplit+1)
			}
			out := make([]object.Object, len(parts))
			for k, p := range parts {
				out[k] = str(p)
			}
			return &object.List{V: out}, nil
		}}, true
	case "rsplit":
		return &object.BuiltinFunc{Name: "rsplit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			maxsplit := -1
			sep, hasSep := strOptStr(a)
			if len(a) >= 2 {
				if n, ok := toInt64(a[1]); ok {
					maxsplit = int(n)
				}
			}
			var parts []string
			if !hasSep {
				parts = strSplitFields(s.V, maxsplit)
			} else if maxsplit < 0 {
				parts = strings.Split(s.V, sep)
			} else {
				parts = strings.SplitAfterN(reverseStr(s.V), reverseStr(sep), maxsplit+1)
				for i, p := range parts {
					parts[i] = reverseStr(strings.TrimSuffix(p, reverseStr(sep)))
				}
				reverseSlice(parts)
			}
			out := make([]object.Object, len(parts))
			for k, p := range parts {
				out[k] = str(p)
			}
			return &object.List{V: out}, nil
		}}, true
	case "splitlines":
		return &object.BuiltinFunc{Name: "splitlines", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			keepends := false
			if len(a) > 0 {
				keepends = object.Truthy(a[0])
			}
			parts := strSplitLines(s.V, keepends)
			out := make([]object.Object, len(parts))
			for k, p := range parts {
				out[k] = str(p)
			}
			return &object.List{V: out}, nil
		}}, true
	case "partition":
		return &object.BuiltinFunc{Name: "partition", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(ii.(*Interp).typeErr, "partition requires 1 argument")
			}
			sep := a[0].(*object.Str).V
			idx := strings.Index(s.V, sep)
			if idx < 0 {
				return &object.Tuple{V: []object.Object{str(s.V), str(""), str("")}}, nil
			}
			return &object.Tuple{V: []object.Object{str(s.V[:idx]), str(sep), str(s.V[idx+len(sep):])}}, nil
		}}, true
	case "rpartition":
		return &object.BuiltinFunc{Name: "rpartition", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(ii.(*Interp).typeErr, "rpartition requires 1 argument")
			}
			sep := a[0].(*object.Str).V
			idx := strings.LastIndex(s.V, sep)
			if idx < 0 {
				return &object.Tuple{V: []object.Object{str(""), str(""), str(s.V)}}, nil
			}
			return &object.Tuple{V: []object.Object{str(s.V[:idx]), str(sep), str(s.V[idx+len(sep):])}}, nil
		}}, true
	// --- join ---
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
					return nil, object.Errorf(ii.(*Interp).typeErr, "sequence item %d: expected str instance, %s found", k, object.TypeName(x))
				}
				parts[k] = sx.V
			}
			return str(strings.Join(parts, s.V)), nil
		}}, true
	// --- search ---
	case "find":
		return &object.BuiltinFunc{Name: "find", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.NewInt(-1), nil
			}
			sub := a[0].(*object.Str).V
			byteIdx := strings.Index(s.V, sub)
			return object.NewInt(int64(strRuneIndex(s.V, byteIdx))), nil
		}}, true
	case "rfind":
		return &object.BuiltinFunc{Name: "rfind", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.NewInt(-1), nil
			}
			sub := a[0].(*object.Str).V
			byteIdx := strings.LastIndex(s.V, sub)
			return object.NewInt(int64(strRuneIndex(s.V, byteIdx))), nil
		}}, true
	case "index":
		return &object.BuiltinFunc{Name: "index", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "substring not found")
			}
			sub := a[0].(*object.Str).V
			byteIdx := strings.Index(s.V, sub)
			if byteIdx < 0 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "substring not found")
			}
			return object.NewInt(int64(strRuneIndex(s.V, byteIdx))), nil
		}}, true
	case "rindex":
		return &object.BuiltinFunc{Name: "rindex", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "substring not found")
			}
			sub := a[0].(*object.Str).V
			byteIdx := strings.LastIndex(s.V, sub)
			if byteIdx < 0 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "substring not found")
			}
			return object.NewInt(int64(strRuneIndex(s.V, byteIdx))), nil
		}}, true
	case "count":
		return &object.BuiltinFunc{Name: "count", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.NewInt(0), nil
			}
			return object.NewInt(int64(strings.Count(s.V, a[0].(*object.Str).V))), nil
		}}, true
	case "startswith":
		return &object.BuiltinFunc{Name: "startswith", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.False, nil
			}
			switch v := a[0].(type) {
			case *object.Str:
				return object.BoolOf(strings.HasPrefix(s.V, v.V)), nil
			case *object.Tuple:
				for _, x := range v.V {
					if ss, ok := x.(*object.Str); ok && strings.HasPrefix(s.V, ss.V) {
						return object.True, nil
					}
				}
				return object.False, nil
			}
			return object.BoolOf(strings.HasPrefix(s.V, a[0].(*object.Str).V)), nil
		}}, true
	case "endswith":
		return &object.BuiltinFunc{Name: "endswith", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.False, nil
			}
			switch v := a[0].(type) {
			case *object.Str:
				return object.BoolOf(strings.HasSuffix(s.V, v.V)), nil
			case *object.Tuple:
				for _, x := range v.V {
					if ss, ok := x.(*object.Str); ok && strings.HasSuffix(s.V, ss.V) {
						return object.True, nil
					}
				}
				return object.False, nil
			}
			return object.BoolOf(strings.HasSuffix(s.V, a[0].(*object.Str).V)), nil
		}}, true
	// --- replace ---
	case "replace":
		return &object.BuiltinFunc{Name: "replace", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(ii.(*Interp).typeErr, "replace needs 2 args")
			}
			old := a[0].(*object.Str).V
			new_ := a[1].(*object.Str).V
			n := -1
			if len(a) >= 3 {
				if v, ok := toInt64(a[2]); ok {
					n = int(v)
				}
			}
			if n < 0 {
				return str(strings.ReplaceAll(s.V, old, new_)), nil
			}
			return str(strings.Replace(s.V, old, new_, n)), nil
		}}, true
	case "removeprefix":
		return &object.BuiltinFunc{Name: "removeprefix", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return str(s.V), nil
			}
			prefix := a[0].(*object.Str).V
			return str(strings.TrimPrefix(s.V, prefix)), nil
		}}, true
	case "removesuffix":
		return &object.BuiltinFunc{Name: "removesuffix", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return str(s.V), nil
			}
			suffix := a[0].(*object.Str).V
			return str(strings.TrimSuffix(s.V, suffix)), nil
		}}, true
	// --- padding ---
	case "center":
		return &object.BuiltinFunc{Name: "center", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return str(s.V), nil
			}
			width, _ := toInt64(a[0])
			fill := " "
			if len(a) >= 2 {
				if fs, ok := a[1].(*object.Str); ok {
					fill = fs.V
				}
			}
			return str(strCenter(s.V, int(width), []rune(fill)[0])), nil
		}}, true
	case "ljust":
		return &object.BuiltinFunc{Name: "ljust", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return str(s.V), nil
			}
			width, _ := toInt64(a[0])
			fill := ' '
			if len(a) >= 2 {
				if fs, ok := a[1].(*object.Str); ok && len([]rune(fs.V)) == 1 {
					fill = []rune(fs.V)[0]
				}
			}
			runes := []rune(s.V)
			pad := int(width) - len(runes)
			if pad <= 0 {
				return str(s.V), nil
			}
			return str(string(runes) + strings.Repeat(string(fill), pad)), nil
		}}, true
	case "rjust":
		return &object.BuiltinFunc{Name: "rjust", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return str(s.V), nil
			}
			width, _ := toInt64(a[0])
			fill := ' '
			if len(a) >= 2 {
				if fs, ok := a[1].(*object.Str); ok && len([]rune(fs.V)) == 1 {
					fill = []rune(fs.V)[0]
				}
			}
			runes := []rune(s.V)
			pad := int(width) - len(runes)
			if pad <= 0 {
				return str(s.V), nil
			}
			return str(strings.Repeat(string(fill), pad) + string(runes)), nil
		}}, true
	case "zfill":
		return &object.BuiltinFunc{Name: "zfill", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return str(s.V), nil
			}
			width, _ := toInt64(a[0])
			runes := []rune(s.V)
			pad := int(width) - len(runes)
			if pad <= 0 {
				return str(s.V), nil
			}
			prefix := ""
			body := s.V
			if len(runes) > 0 && (runes[0] == '+' || runes[0] == '-') {
				prefix = string(runes[0])
				body = string(runes[1:])
			}
			return str(prefix + strings.Repeat("0", pad) + body), nil
		}}, true
	case "expandtabs":
		return &object.BuiltinFunc{Name: "expandtabs", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			tabsize := 8
			if len(a) > 0 {
				if n, ok := toInt64(a[0]); ok {
					tabsize = int(n)
				}
			}
			var b strings.Builder
			col := 0
			for _, r := range s.V {
				if r == '\t' {
					if tabsize > 0 {
						spaces := tabsize - col%tabsize
						b.WriteString(strings.Repeat(" ", spaces))
						col += spaces
					}
				} else if r == '\n' || r == '\r' {
					b.WriteRune(r)
					col = 0
				} else {
					b.WriteRune(r)
					col++
				}
			}
			return str(b.String()), nil
		}}, true
	// --- translate ---
	case "translate":
		return &object.BuiltinFunc{Name: "translate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return str(s.V), nil
			}
			table, ok := a[0].(*object.Dict)
			if !ok {
				return str(s.V), nil
			}
			var b strings.Builder
			for _, r := range s.V {
				key := object.NewInt(int64(r))
				if v, ok2, _ := table.Get(key); ok2 {
					switch mv := v.(type) {
					case *object.NoneType:
						// delete
					case *object.Int:
						b.WriteRune(rune(mv.Int64()))
					case *object.Str:
						b.WriteString(mv.V)
					}
				} else {
					b.WriteRune(r)
				}
			}
			return str(b.String()), nil
		}}, true
	case "maketrans":
		return &object.BuiltinFunc{Name: "maketrans", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			d := object.NewDict()
			if len(a) == 1 {
				if src, ok := a[0].(*object.Dict); ok {
					// dict form: keys can be ints or single-char strs
					ks, vs := src.Items()
					for k, key := range ks {
						var ikey object.Object
						switch kv := key.(type) {
						case *object.Int:
							ikey = kv
						case *object.Str:
							if rs := []rune(kv.V); len(rs) == 1 {
								ikey = object.NewInt(int64(rs[0]))
							}
						}
						if ikey != nil {
							_ = d.Set(ikey, vs[k])
						}
					}
				}
			} else if len(a) >= 2 {
				x := []rune(a[0].(*object.Str).V)
				y := []rune(a[1].(*object.Str).V)
				for k, r := range x {
					var val object.Object = object.None
					if k < len(y) {
						val = object.NewInt(int64(y[k]))
					}
					_ = d.Set(object.NewInt(int64(r)), val)
				}
				if len(a) >= 3 {
					for _, r := range []rune(a[2].(*object.Str).V) {
						_ = d.Set(object.NewInt(int64(r)), object.None)
					}
				}
			}
			return d, nil
		}}, true
	// --- encode ---
	case "encode":
		return &object.BuiltinFunc{Name: "encode", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Bytes{V: []byte(s.V)}, nil
		}}, true
	// --- format ---
	case "format":
		return &object.BuiltinFunc{Name: "format", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			result, err := strFormat(ii.(*Interp), s.V, a, kw)
			if err != nil {
				return nil, err
			}
			return str(result), nil
		}}, true
	case "format_map":
		return &object.BuiltinFunc{Name: "format_map", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return str(s.V), nil
			}
			mapping, ok := a[0].(*object.Dict)
			if !ok {
				return str(s.V), nil
			}
			result, err := strFormat(ii.(*Interp), s.V, nil, mapping)
			if err != nil {
				return nil, err
			}
			return str(result), nil
		}}, true
	}
	return nil, false
}

func hexNibble(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

func reverseStr(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func reverseSlice(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func strCenter(s string, width int, fill rune) string {
	runes := []rune(s)
	pad := width - len(runes)
	if pad <= 0 {
		return s
	}
	left := pad / 2
	right := pad - left
	return strings.Repeat(string(fill), left) + s + strings.Repeat(string(fill), right)
}

// strSplitLines splits s on line boundaries, optionally keeping the endings.
func strSplitLines(s string, keepends bool) []string {
	var parts []string
	i := 0
	for i < len(s) {
		j := i
		for j < len(s) && s[j] != '\n' && s[j] != '\r' && s[j] != '\f' &&
			s[j] != '\v' && s[j] != '\x1c' && s[j] != '\x1d' && s[j] != '\x1e' && s[j] != '\x85' {
			j++
		}
		if j >= len(s) {
			if j > i {
				parts = append(parts, s[i:j])
			}
			break
		}
		end := j
		if s[j] == '\r' && j+1 < len(s) && s[j+1] == '\n' {
			j += 2
		} else {
			j++
		}
		if keepends {
			parts = append(parts, s[i:j])
		} else {
			parts = append(parts, s[i:end])
		}
		i = j
	}
	return parts
}

// strFormat is a minimal implementation of str.format() / str.format_map().
// It handles positional {0}, {1} and keyword {name} fields without format specs.
func strFormat(i *Interp, tmpl string, args []object.Object, kwargs *object.Dict) (string, error) {
	var b strings.Builder
	autoIdx := 0
	j := 0
	for j < len(tmpl) {
		if tmpl[j] == '{' {
			if j+1 < len(tmpl) && tmpl[j+1] == '{' {
				b.WriteByte('{')
				j += 2
				continue
			}
			end := strings.IndexByte(tmpl[j:], '}')
			if end < 0 {
				b.WriteByte('{')
				j++
				continue
			}
			field := tmpl[j+1 : j+end]
			j += end + 1
			// strip format spec after ':'
			if colon := strings.IndexByte(field, ':'); colon >= 0 {
				field = field[:colon]
			}
			// strip conversion after '!'
			if bang := strings.IndexByte(field, '!'); bang >= 0 {
				field = field[:bang]
			}
			var val object.Object
			if field == "" {
				if autoIdx < len(args) {
					val = args[autoIdx]
					autoIdx++
				} else {
					val = object.None
				}
			} else if n, ok2 := parseInt(field); ok2 {
				if int(n) < len(args) {
					val = args[n]
				} else {
					val = object.None
				}
			} else if kwargs != nil {
				v, ok2 := kwargs.GetStr(field)
				if ok2 {
					val = v
				} else {
					return "", object.Errorf(i.keyErr, "KeyError: '%s'", field)
				}
			} else {
				return "", object.Errorf(i.keyErr, "KeyError: '%s'", field)
			}
			b.WriteString(object.Str_(val))
		} else if tmpl[j] == '}' && j+1 < len(tmpl) && tmpl[j+1] == '}' {
			b.WriteByte('}')
			j += 2
		} else {
			b.WriteByte(tmpl[j])
			j++
		}
	}
	return b.String(), nil
}

func parseInt(s string) (int64, bool) {
	n := int64(0)
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int64(c-'0')
	}
	return n, true
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
		return &object.BuiltinFunc{Name: "sort", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			reverse := false
			var key object.Object
			if kw != nil {
				if v, ok := kw.GetStr("reverse"); ok {
					reverse = object.Truthy(v)
				}
				if v, ok := kw.GetStr("key"); ok && v != object.None {
					key = v
				}
			}
			if key != nil {
				return object.None, sortListKey(ii.(*Interp), l.V, key, reverse)
			}
			return object.None, sortList(ii.(*Interp), l.V, reverse)
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
	case "popitem":
		return &object.BuiltinFunc{Name: "popitem", Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			keys, vals := d.Items()
			if len(keys) == 0 {
				return nil, object.Errorf(ii.(*Interp).keyErr, "dictionary is empty")
			}
			k := keys[len(keys)-1]
			v := vals[len(vals)-1]
			_, _ = d.Delete(k)
			return &object.Tuple{V: []object.Object{k, v}}, nil
		}}, true
	case "fromkeys":
		return &object.BuiltinFunc{Name: "fromkeys", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.NewDict(), nil
			}
			var val object.Object = object.None
			if len(a) >= 2 {
				val = a[1]
			}
			items, err := iterate(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			nd := object.NewDict()
			for _, k := range items {
				if err := nd.Set(k, val); err != nil {
					return nil, err
				}
			}
			return nd, nil
		}}, true
	case "__or__":
		return &object.BuiltinFunc{Name: "__or__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			other, ok := a[0].(*object.Dict)
			if !ok {
				return object.NotImplemented, nil
			}
			nd := object.NewDict()
			keys, vals := d.Items()
			for k, key := range keys {
				_ = nd.Set(key, vals[k])
			}
			oks, ovs := other.Items()
			for k, key := range oks {
				_ = nd.Set(key, ovs[k])
			}
			return nd, nil
		}}, true
	case "__ior__":
		return &object.BuiltinFunc{Name: "__ior__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			other, ok := a[0].(*object.Dict)
			if !ok {
				return object.NotImplemented, nil
			}
			oks, ovs := other.Items()
			for k, key := range oks {
				_ = d.Set(key, ovs[k])
			}
			return d, nil
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
	case "rfind":
		return &object.BuiltinFunc{Name: "rfind", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			nd, _ := needleBytes(a[0])
			hay := data()
			idx := -1
			for i := 0; i+len(nd) <= len(hay); i++ {
				if bytesEqAt(hay, i, nd) {
					idx = i
				}
			}
			return object.NewInt(int64(idx)), nil
		}}, true
	case "rindex":
		return &object.BuiltinFunc{Name: "rindex", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			nd, _ := needleBytes(a[0])
			hay := data()
			idx := -1
			for i := 0; i+len(nd) <= len(hay); i++ {
				if bytesEqAt(hay, i, nd) {
					idx = i
				}
			}
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
	case "join":
		return &object.BuiltinFunc{Name: "join", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return wrap(nil), nil
			}
			items, err := iterate(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			var parts [][]byte
			for _, x := range items {
				b, ok := bytesBytesOrArray(x)
				if !ok {
					return nil, object.Errorf(ii.(*Interp).typeErr, "sequence item must be bytes-like")
				}
				parts = append(parts, b)
			}
			sep := data()
			result := bytesJoin(parts, sep)
			return wrap(result), nil
		}}, true
	case "strip":
		return &object.BuiltinFunc{Name: "strip", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			d := data()
			var chars []byte
			if len(a) > 0 {
				chars, _ = bytesBytesOrArray(a[0])
			}
			return wrap(bytesTrim(d, chars, true, true)), nil
		}}, true
	case "lstrip":
		return &object.BuiltinFunc{Name: "lstrip", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			d := data()
			var chars []byte
			if len(a) > 0 {
				chars, _ = bytesBytesOrArray(a[0])
			}
			return wrap(bytesTrim(d, chars, true, false)), nil
		}}, true
	case "rstrip":
		return &object.BuiltinFunc{Name: "rstrip", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			d := data()
			var chars []byte
			if len(a) > 0 {
				chars, _ = bytesBytesOrArray(a[0])
			}
			return wrap(bytesTrim(d, chars, false, true)), nil
		}}, true
	case "upper":
		return &object.BuiltinFunc{Name: "upper", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			d := data()
			out := make([]byte, len(d))
			for i, c := range d {
				if c >= 'a' && c <= 'z' {
					out[i] = c - 32
				} else {
					out[i] = c
				}
			}
			return wrap(out), nil
		}}, true
	case "lower":
		return &object.BuiltinFunc{Name: "lower", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			d := data()
			out := make([]byte, len(d))
			for i, c := range d {
				if c >= 'A' && c <= 'Z' {
					out[i] = c + 32
				} else {
					out[i] = c
				}
			}
			return wrap(out), nil
		}}, true
	case "center":
		return &object.BuiltinFunc{Name: "center", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			d := data()
			if len(a) == 0 {
				return wrap(d), nil
			}
			width, _ := toInt64(a[0])
			fill := byte(' ')
			if len(a) >= 2 {
				if fb, ok := bytesBytesOrArray(a[1]); ok && len(fb) == 1 {
					fill = fb[0]
				}
			}
			return wrap(bytesPad(d, int(width), fill, "center")), nil
		}}, true
	case "ljust":
		return &object.BuiltinFunc{Name: "ljust", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			d := data()
			if len(a) == 0 {
				return wrap(d), nil
			}
			width, _ := toInt64(a[0])
			fill := byte(' ')
			if len(a) >= 2 {
				if fb, ok := bytesBytesOrArray(a[1]); ok && len(fb) == 1 {
					fill = fb[0]
				}
			}
			return wrap(bytesPad(d, int(width), fill, "right")), nil
		}}, true
	case "rjust":
		return &object.BuiltinFunc{Name: "rjust", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			d := data()
			if len(a) == 0 {
				return wrap(d), nil
			}
			width, _ := toInt64(a[0])
			fill := byte(' ')
			if len(a) >= 2 {
				if fb, ok := bytesBytesOrArray(a[1]); ok && len(fb) == 1 {
					fill = fb[0]
				}
			}
			return wrap(bytesPad(d, int(width), fill, "left")), nil
		}}, true
	case "zfill":
		return &object.BuiltinFunc{Name: "zfill", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			d := data()
			if len(a) == 0 {
				return wrap(d), nil
			}
			width, _ := toInt64(a[0])
			return wrap(bytesPad(d, int(width), '0', "right_fill_left")), nil
		}}, true
	case "hex":
		return &object.BuiltinFunc{Name: "hex", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			const digits = "0123456789abcdef"
			d := data()
			buf := make([]byte, 0, 2*len(d))
			for _, c := range d {
				buf = append(buf, digits[c>>4], digits[c&0xf])
			}
			return &object.Str{V: string(buf)}, nil
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

func bytesJoin(parts [][]byte, sep []byte) []byte {
	if len(parts) == 0 {
		return nil
	}
	total := len(sep) * (len(parts) - 1)
	for _, p := range parts {
		total += len(p)
	}
	out := make([]byte, 0, total)
	for i, p := range parts {
		if i > 0 {
			out = append(out, sep...)
		}
		out = append(out, p...)
	}
	return out
}

// bytesTrim trims ASCII whitespace or the given chars from left/right.
func bytesTrim(d, chars []byte, left, right bool) []byte {
	isChar := func(c byte) bool {
		if chars == nil {
			return isAsciiSpace(c)
		}
		for _, ch := range chars {
			if c == ch {
				return true
			}
		}
		return false
	}
	start, end := 0, len(d)
	if left {
		for start < end && isChar(d[start]) {
			start++
		}
	}
	if right {
		for end > start && isChar(d[end-1]) {
			end--
		}
	}
	return append([]byte(nil), d[start:end]...)
}

// bytesPad pads d to width using fill byte.
// mode: "left" = rjust, "right" = ljust, "center" = center, "right_fill_left" = zfill.
func bytesPad(d []byte, width int, fill byte, mode string) []byte {
	pad := width - len(d)
	if pad <= 0 {
		return append([]byte(nil), d...)
	}
	switch mode {
	case "left":
		out := make([]byte, width)
		copy(out[pad:], d)
		for i := 0; i < pad; i++ {
			out[i] = fill
		}
		return out
	case "right":
		out := make([]byte, width)
		copy(out, d)
		for i := len(d); i < width; i++ {
			out[i] = fill
		}
		return out
	case "center":
		left := pad / 2
		right := pad - left
		out := make([]byte, width)
		for i := 0; i < left; i++ {
			out[i] = fill
		}
		copy(out[left:], d)
		for i := left + len(d); i < width; i++ {
			_ = right
			out[i] = fill
		}
		return out
	case "right_fill_left":
		// zfill: preserve sign prefix if present
		out := make([]byte, width)
		if len(d) > 0 && (d[0] == '+' || d[0] == '-') {
			out[0] = d[0]
			for i := 1; i <= pad; i++ {
				out[i] = fill
			}
			copy(out[pad+1:], d[1:])
		} else {
			for i := 0; i < pad; i++ {
				out[i] = fill
			}
			copy(out[pad:], d)
		}
		return out
	}
	return append([]byte(nil), d...)
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
			if ba, ok := mv.Backing.(*object.Bytearray); ok && ba.Views > 0 {
				ba.Views--
			}
			return object.None, nil
		}}, true
	case "shape":
		n := int64(mv.Stop - mv.Start)
		return &object.Tuple{V: []object.Object{object.NewInt(n)}}, true
	case "ndim":
		return object.NewInt(1), true
	case "strides":
		return &object.Tuple{V: []object.Object{object.NewInt(1)}}, true
	case "suboffsets":
		return &object.Tuple{}, true
	case "c_contiguous":
		return object.True, true
	case "f_contiguous":
		return object.True, true
	case "contiguous":
		return object.True, true
	case "cast":
		return &object.BuiltinFunc{Name: "cast", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// Returns a new memoryview with the requested format; for
			// simple byte buffers format "B" or "b" or "c" is fine.
			return mv, nil
		}}, true
	}
	return nil, false
}

func bytesMethod(b *object.Bytes, name string) (object.Object, bool) {
	if m, ok := bytesSubMethod(func() []byte { return b.V }, false, name); ok {
		return m, true
	}
	switch name {
	case "decode":
		return &object.BuiltinFunc{Name: "decode", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: string(b.V)}, nil
		}}, true
	}
	return nil, false
}

func bytearrayMethod(interp *Interp, ba *object.Bytearray, name string) (object.Object, bool) {
	if m, ok := bytesSubMethod(func() []byte { return ba.V }, true, name); ok {
		return m, true
	}
	bufErr := func() error {
		return object.Errorf(interp.bufferErr, "Existing exports of data: object cannot be re-sized")
	}
	switch name {
	case "clear":
		return &object.BuiltinFunc{Name: "clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if ba.Views > 0 {
				return nil, bufErr()
			}
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
			if ba.Views > 0 {
				return nil, bufErr()
			}
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
			if ba.Views > 0 {
				return nil, bufErr()
			}
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
			if ba.Views > 0 {
				return nil, bufErr()
			}
			n, ok := toInt64(a[0])
			if !ok || n < 0 || n > 255 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "byte must be in range(0, 256)")
			}
			ba.V = append(ba.V, byte(n))
			return object.None, nil
		}}, true
	case "extend":
		return &object.BuiltinFunc{Name: "extend", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if ba.Views > 0 {
				return nil, bufErr()
			}
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
			if ba.Views > 0 {
				return nil, bufErr()
			}
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
	case "copy":
		return &object.BuiltinFunc{Name: "copy", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			out := make([]byte, len(ba.V))
			copy(out, ba.V)
			return &object.Bytearray{V: out}, nil
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
			s.Discard(a[0])
			return object.None, nil
		}}, true
	case "remove":
		return &object.BuiltinFunc{Name: "remove", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ok, err := s.Contains(a[0])
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, object.Errorf(ii.(*Interp).keyErr, "%s", object.Repr(a[0]))
			}
			s.Discard(a[0])
			return object.None, nil
		}}, true
	case "pop":
		return &object.BuiltinFunc{Name: "pop", Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			items := s.Items()
			if len(items) == 0 {
				return nil, object.Errorf(ii.(*Interp).keyErr, "pop from an empty set")
			}
			v := items[0]
			s.Discard(v)
			return v, nil
		}}, true
	case "clear":
		return &object.BuiltinFunc{Name: "clear", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			items := append([]object.Object(nil), s.Items()...)
			for _, x := range items {
				s.Discard(x)
			}
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
	case "update":
		return &object.BuiltinFunc{Name: "update", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			for _, arg := range a {
				other, err := materializeSet(ii.(*Interp), arg)
				if err != nil {
					return nil, err
				}
				for _, x := range setItems(other) {
					if err := s.Add(x); err != nil {
						return nil, err
					}
				}
			}
			return object.None, nil
		}}, true
	case "intersection_update":
		return &object.BuiltinFunc{Name: "intersection_update", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			result, err := setReduce(ii.(*Interp), s, a, "&")
			if err != nil {
				return nil, err
			}
			items := append([]object.Object(nil), s.Items()...)
			for _, x := range items {
				s.Discard(x)
			}
			for _, x := range setItems(result) {
				_ = s.Add(x)
			}
			return object.None, nil
		}}, true
	case "difference_update":
		return &object.BuiltinFunc{Name: "difference_update", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			for _, arg := range a {
				other, err := materializeSet(ii.(*Interp), arg)
				if err != nil {
					return nil, err
				}
				for _, x := range setItems(other) {
					s.Discard(x)
				}
			}
			return object.None, nil
		}}, true
	case "symmetric_difference_update":
		return &object.BuiltinFunc{Name: "symmetric_difference_update", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.None, nil
			}
			other, err := materializeSet(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			result := setBitop(s, other, "^")
			items := append([]object.Object(nil), s.Items()...)
			for _, x := range items {
				s.Discard(x)
			}
			for _, x := range setItems(result) {
				_ = s.Add(x)
			}
			return object.None, nil
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

// intMethod returns methods on int objects.
func intMethod(n *object.Int, name string) (object.Object, bool) {
	switch name {
	case "bit_length":
		return &object.BuiltinFunc{Name: "bit_length", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(n.V.BitLen())), nil
		}}, true
	case "bit_count":
		return &object.BuiltinFunc{Name: "bit_count", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			abs := new(big.Int).Abs(&n.V)
			return object.NewInt(int64(abs.BitLen() - int(abs.BitLen()) + popcount(abs))), nil
		}}, true
	case "to_bytes":
		return &object.BuiltinFunc{Name: "to_bytes", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			length := int64(1)
			byteorder := "big"
			signed := false
			if len(a) >= 1 {
				length, _ = toInt64(a[0])
			}
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					byteorder = s.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("signed"); ok {
					signed = object.Truthy(v)
				}
			}
			var src *big.Int
			if signed && n.V.Sign() < 0 {
				// two's complement
				mod := new(big.Int).Lsh(big.NewInt(1), uint(length*8))
				src = new(big.Int).Add(&n.V, mod)
			} else {
				src = new(big.Int).Set(&n.V)
			}
			b := make([]byte, length)
			raw := src.Bytes()
			copy(b[int(length)-len(raw):], raw)
			if byteorder == "little" {
				for l, r := 0, len(b)-1; l < r; l, r = l+1, r-1 {
					b[l], b[r] = b[r], b[l]
				}
			}
			return &object.Bytes{V: b}, nil
		}}, true
	case "conjugate":
		return &object.BuiltinFunc{Name: "conjugate", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return n, nil
		}}, true
	case "as_integer_ratio":
		return &object.BuiltinFunc{Name: "as_integer_ratio", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{n, object.NewInt(1)}}, nil
		}}, true
	}
	return nil, false
}

// popcount counts set bits in a big.Int.
func popcount(n *big.Int) int {
	count := 0
	for _, w := range n.Bits() {
		count += bits64(uint64(w))
	}
	return count
}

func bits64(x uint64) int {
	count := 0
	for x != 0 {
		count += int(x & 1)
		x >>= 1
	}
	return count
}

// floatMethod returns methods on float objects.
func floatMethod(f *object.Float, name string) (object.Object, bool) {
	switch name {
	case "is_integer":
		return &object.BuiltinFunc{Name: "is_integer", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(!math.IsInf(f.V, 0) && !math.IsNaN(f.V) && f.V == math.Trunc(f.V)), nil
		}}, true
	case "as_integer_ratio":
		return &object.BuiltinFunc{Name: "as_integer_ratio", Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if math.IsInf(f.V, 0) {
				return nil, object.Errorf(ii.(*Interp).valueErr, "cannot convert infinity to integer ratio")
			}
			if math.IsNaN(f.V) {
				return nil, object.Errorf(ii.(*Interp).valueErr, "cannot convert NaN to integer ratio")
			}
			num, den := floatRatio(f.V)
			return &object.Tuple{V: []object.Object{
				object.IntFromBig(num),
				object.IntFromBig(den),
			}}, nil
		}}, true
	case "conjugate":
		return &object.BuiltinFunc{Name: "conjugate", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return f, nil
		}}, true
	case "hex":
		return &object.BuiltinFunc{Name: "hex", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: floatHex(f.V)}, nil
		}}, true
	}
	return nil, false
}

// floatRatio returns (numerator, denominator) as *big.Int for a finite float.
func floatRatio(f float64) (*big.Int, *big.Int) {
	if f == 0 {
		return big.NewInt(0), big.NewInt(1)
	}
	sign := 1
	if f < 0 {
		sign = -1
		f = -f
	}
	frac, exp := math.Frexp(f) // f = frac * 2^exp, 0.5 <= frac < 1
	// frac has 52 mantissa bits; multiply by 2^53 to get integer
	const shift = 53
	mantissa := uint64(frac * (1 << shift))
	num := new(big.Int).SetUint64(mantissa)
	den := new(big.Int).SetUint64(1)
	e := exp - shift
	if e >= 0 {
		num.Lsh(num, uint(e))
	} else {
		den.Lsh(den, uint(-e))
	}
	// reduce
	gcd := new(big.Int).GCD(nil, nil, num, den)
	num.Div(num, gcd)
	den.Div(den, gcd)
	if sign < 0 {
		num.Neg(num)
	}
	return num, den
}

// floatHex returns Python-style hex representation of a float.
func floatHex(f float64) string {
	if math.IsInf(f, 1) {
		return "inf"
	}
	if math.IsInf(f, -1) {
		return "-inf"
	}
	if math.IsNaN(f) {
		return "nan"
	}
	sign := ""
	if math.Signbit(f) {
		sign = "-"
		f = -f
	}
	if f == 0 {
		return sign + "0x0.0000000000000p+0"
	}
	frac, exp := math.Frexp(f)
	// frac in [0.5, 1.0); adjust so mantissa is displayed as 0x1.xxxxp+e
	exp--
	frac *= 2
	// frac now in [1.0, 2.0); strip the leading 1
	mantissa := frac - 1.0
	// 52 hex digits of the mantissa (13 hex digits = 52 bits)
	m := uint64(mantissa * (1 << 52))
	hexStr := "0000000000000"
	buf := make([]byte, 13)
	const hexChars = "0123456789abcdef"
	for i := 12; i >= 0; i-- {
		buf[i] = hexChars[m&0xf]
		m >>= 4
	}
	hexStr = string(buf)
	// trim trailing zeros but keep at least one digit
	hexStr = strings.TrimRight(hexStr, "0")
	if hexStr == "" {
		hexStr = "0"
	}
	expSign := "+"
	if exp < 0 {
		expSign = "-"
		exp = -exp
	}
	return sign + "0x1." + hexStr + "p" + expSign + string([]byte{byte('0' + exp/100%10), byte('0' + exp/10%10), byte('0' + exp%10)})
}

// rangeMethod returns methods and attributes for range objects.
func rangeMethod(r *object.Range, name string) (object.Object, bool) {
	switch name {
	case "count":
		return &object.BuiltinFunc{Name: "count", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return object.NewInt(0), nil
			}
			v, ok := toInt64(a[0])
			if !ok {
				return object.NewInt(0), nil
			}
			if rangeContains(r, v) {
				return object.NewInt(1), nil
			}
			return object.NewInt(0), nil
		}}, true
	case "index":
		return &object.BuiltinFunc{Name: "index", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "0 is not in range")
			}
			v, ok := toInt64(a[0])
			if !ok || !rangeContains(r, v) {
				return nil, object.Errorf(ii.(*Interp).valueErr, "%d is not in range", v)
			}
			return object.NewInt((v - r.Start) / r.Step), nil
		}}, true
	}
	return nil, false
}

func rangeContains(r *object.Range, v int64) bool {
	if r.Step > 0 {
		return v >= r.Start && v < r.Stop && (v-r.Start)%r.Step == 0
	}
	if r.Step < 0 {
		return v <= r.Start && v > r.Stop && (v-r.Start)%r.Step == 0
	}
	return false
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

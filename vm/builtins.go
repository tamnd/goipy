package vm

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/tamnd/goipy/object"
)

// initBuiltins creates the interpreter's builtin namespace and exception
// classes.
func (i *Interp) initBuiltins() {
	b := object.NewDict()
	i.Builtins = b

	// Exception classes.
	mk := func(name string, bases ...*object.Class) *object.Class {
		c := &object.Class{Name: name, Dict: object.NewDict()}
		c.Bases = bases
		b.SetStr(name, c)
		return c
	}
	i.baseExc = mk("BaseException")
	i.exception = mk("Exception", i.baseExc)

	// ArithmeticError family.
	i.arithErr = mk("ArithmeticError", i.exception)
	i.floatErr = mk("FloatingPointError", i.arithErr)
	i.overflowErr = mk("OverflowError", i.arithErr)
	i.zeroDivErr = mk("ZeroDivisionError", i.arithErr)

	// Simple Exception subclasses.
	i.assertErr = mk("AssertionError", i.exception)
	i.attrErr = mk("AttributeError", i.exception)
	i.bufferErr = mk("BufferError", i.exception)
	i.eofErr = mk("EOFError", i.exception)
	i.memoryErr = mk("MemoryError", i.exception)

	// ImportError family.
	i.importErr = mk("ImportError", i.exception)
	i.moduleNotFoundErr = mk("ModuleNotFoundError", i.importErr)

	// LookupError family.
	i.lookupErr = mk("LookupError", i.exception)
	i.indexErr = mk("IndexError", i.lookupErr)
	i.keyErr = mk("KeyError", i.lookupErr)

	// NameError family.
	i.nameErr = mk("NameError", i.exception)
	i.unboundLocalErr = mk("UnboundLocalError", i.nameErr)

	// OSError family.
	i.osErr = mk("OSError", i.exception)
	mk("IOError", i.osErr)
	mk("EnvironmentError", i.osErr)
	mk("WindowsError", i.osErr)
	mk("BlockingIOError", i.osErr)
	mk("ChildProcessError", i.osErr)
	i.connectionErr = mk("ConnectionError", i.osErr)
	mk("BrokenPipeError", i.connectionErr, i.osErr)
	mk("ConnectionAbortedError", i.connectionErr)
	mk("ConnectionRefusedError", i.connectionErr)
	mk("ConnectionResetError", i.connectionErr)
	i.fileNotFoundErr = mk("FileNotFoundError", i.osErr)
	i.fileExistsErr = mk("FileExistsError", i.osErr)
	mk("InterruptedError", i.osErr)
	mk("IsADirectoryError", i.osErr)
	mk("NotADirectoryError", i.osErr)
	mk("PermissionError", i.osErr)
	mk("ProcessLookupError", i.osErr)
	mk("TimeoutError", i.osErr)

	// Other Exception subclasses.
	mk("ReferenceError", i.exception)
	i.runtimeErr = mk("RuntimeError", i.exception)
	i.notImpl = mk("NotImplementedError", i.runtimeErr)
	i.recursionErr = mk("RecursionError", i.runtimeErr)
	i.stopAsyncIter = mk("StopAsyncIteration", i.exception)
	i.stopIter = mk("StopIteration", i.exception)

	// SyntaxError family.
	i.syntaxErr = mk("SyntaxError", i.exception)
	indentErr := mk("IndentationError", i.syntaxErr)
	mk("TabError", indentErr)

	// SystemError.
	i.systemErr = mk("SystemError", i.exception)

	// TypeError / ValueError.
	i.typeErr = mk("TypeError", i.exception)
	i.valueErr = mk("ValueError", i.exception)

	// UnicodeError family (subclass of ValueError).
	i.unicodeErr = mk("UnicodeError", i.valueErr)
	i.unicodeDecodeErr = mk("UnicodeDecodeError", i.unicodeErr)
	i.unicodeEncodeErr = mk("UnicodeEncodeError", i.unicodeErr)
	mk("UnicodeTranslateError", i.unicodeErr)

	// Warning hierarchy (subclass of Exception).
	i.warningClass = mk("Warning", i.exception)
	mk("DeprecationWarning", i.warningClass)
	mk("PendingDeprecationWarning", i.warningClass)
	mk("RuntimeWarning", i.warningClass)
	mk("SyntaxWarning", i.warningClass)
	mk("UserWarning", i.warningClass)
	mk("FutureWarning", i.warningClass)
	mk("ImportWarning", i.warningClass)
	mk("UnicodeWarning", i.warningClass)
	mk("BytesWarning", i.warningClass)
	mk("ResourceWarning", i.warningClass)
	mk("EncodingWarning", i.warningClass)

	// BaseException-level (not caught by bare except Exception).
	i.systemExit = mk("SystemExit", i.baseExc)
	i.keyboardInterrupt = mk("KeyboardInterrupt", i.baseExc)
	i.generatorExit = mk("GeneratorExit", i.baseExc)

	// ExceptionGroup (Python 3.11+): inherits Exception AND BaseExceptionGroup.
	i.baseExcGroup = mk("BaseExceptionGroup", i.baseExc)
	mk("ExceptionGroup", i.exception, i.baseExcGroup)

	// Singletons & types.
	b.SetStr("None", object.None)
	b.SetStr("True", object.True)
	b.SetStr("False", object.False)
	b.SetStr("Ellipsis", object.Ellipsis)
	b.SetStr("NotImplemented", object.NotImplemented)

	// Constructors / types.
	intFromBytes := &object.BuiltinFunc{Name: "from_bytes", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if len(a) == 0 {
			return object.NewInt(0), nil
		}
		var raw []byte
		switch v := a[0].(type) {
		case *object.Bytes:
			raw = v.V
		case *object.Bytearray:
			raw = v.V
		default:
			items, err := iterate(in, a[0])
			if err != nil {
				return nil, err
			}
			raw = make([]byte, len(items))
			for i, x := range items {
				n, ok := toInt64(x)
				if !ok {
					return nil, object.Errorf(in.typeErr, "int.from_bytes: not an integer")
				}
				raw[i] = byte(n)
			}
		}
		byteorder := "big"
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				byteorder = s.V
			}
		}
		signed := false
		if kw != nil {
			if v, ok := kw.GetStr("signed"); ok {
				signed = object.Truthy(v)
			}
		}
		b2 := append([]byte(nil), raw...)
		if byteorder == "little" {
			for l, r := 0, len(b2)-1; l < r; l, r = l+1, r-1 {
				b2[l], b2[r] = b2[r], b2[l]
			}
		}
		n := new(big.Int).SetBytes(b2)
		if signed && len(b2) > 0 && b2[0]&0x80 != 0 {
			mod := new(big.Int).Lsh(big.NewInt(1), uint(len(b2)*8))
			n.Sub(n, mod)
		}
		return object.IntFromBig(n), nil
	}}
	intAttrs := object.NewDict()
	intAttrs.SetStr("from_bytes", intFromBytes)

	b.SetStr("int", &object.BuiltinFunc{Name: "int", Attrs: intAttrs, Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if len(a) == 0 {
			return object.NewInt(0), nil
		}
		if inst, ok := a[0].(*object.Instance); ok {
			for _, name := range [2]string{"__int__", "__index__"} {
				if r, ok, err := in.callInstanceDunder(inst, name); ok {
					if err != nil {
						return nil, err
					}
					if _, ok := r.(*object.Int); ok {
						return r, nil
					}
					return nil, object.Errorf(in.typeErr, "%s returned non-int", name)
				}
			}
		}
		switch v := a[0].(type) {
		case *object.Int:
			return v, nil
		case *object.Bool:
			if v.V {
				return object.NewInt(1), nil
			}
			return object.NewInt(0), nil
		case *object.Float:
			n := new(big.Int)
			new(big.Float).SetFloat64(v.V).Int(n)
			return object.IntFromBig(n), nil
		case *object.Str:
			base := 10
			if len(a) > 1 {
				if n, ok := toInt64(a[1]); ok {
					base = int(n)
				}
			}
			if kw != nil {
				if bv, ok := kw.GetStr("base"); ok {
					if n, ok := toInt64(bv); ok {
						base = int(n)
					}
				}
			}
			n, ok := new(big.Int).SetString(strings.TrimSpace(v.V), base)
			if !ok {
				return nil, object.Errorf(in.valueErr, "invalid literal for int()")
			}
			return object.IntFromBig(n), nil
		}
		return nil, object.Errorf(in.typeErr, "int() argument must be str or int")
	}})
	floatFromHex := &object.BuiltinFunc{Name: "fromhex", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(ii.(*Interp).typeErr, "float.fromhex() takes exactly one argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(ii.(*Interp).typeErr, "float.fromhex() argument must be str")
		}
		clean := strings.TrimSpace(s.V)
		// Normalise to Go hex-float format (prepend 0x after optional sign)
		withHex := clean
		lower := strings.ToLower(clean)
		if !strings.HasPrefix(lower, "0x") && !strings.HasPrefix(lower, "+0x") && !strings.HasPrefix(lower, "-0x") {
			if strings.HasPrefix(clean, "+") || strings.HasPrefix(clean, "-") {
				withHex = string(clean[0]) + "0x" + clean[1:]
			} else {
				withHex = "0x" + clean
			}
		}
		f, err := strconv.ParseFloat(withHex, 64)
		if err != nil {
			// Fall back to decimal parsing
			f, err = strconv.ParseFloat(clean, 64)
			if err != nil {
				return nil, object.Errorf(ii.(*Interp).valueErr, "could not convert string to float: %s", s.V)
			}
		}
		return &object.Float{V: f}, nil
	}}
	floatAttrs := object.NewDict()
	floatAttrs.SetStr("fromhex", floatFromHex)
	b.SetStr("float", &object.BuiltinFunc{Name: "float", Attrs: floatAttrs, Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if len(a) == 0 {
			return &object.Float{V: 0}, nil
		}
		if inst, ok := a[0].(*object.Instance); ok {
			if r, ok, err := in.callInstanceDunder(inst, "__float__"); ok {
				if err != nil {
					return nil, err
				}
				if _, ok := r.(*object.Float); ok {
					return r, nil
				}
				return nil, object.Errorf(in.typeErr, "__float__ returned non-float")
			}
		}
		switch v := a[0].(type) {
		case *object.Float:
			return v, nil
		case *object.Int:
			f, _ := new(big.Float).SetInt(&v.V).Float64()
			return &object.Float{V: f}, nil
		case *object.Bool:
			if v.V {
				return &object.Float{V: 1}, nil
			}
			return &object.Float{V: 0}, nil
		case *object.Str:
			f, err := strconv.ParseFloat(strings.TrimSpace(v.V), 64)
			if err != nil {
				return nil, object.Errorf(in.valueErr, "could not convert to float")
			}
			return &object.Float{V: f}, nil
		}
		return nil, object.Errorf(in.typeErr, "float() bad arg")
	}})
	strMaketrans := &object.BuiltinFunc{Name: "maketrans", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		d := object.NewDict()
		if len(a) == 1 {
			if src, ok := a[0].(*object.Dict); ok {
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
	}}
	strAttrs := object.NewDict()
	strAttrs.SetStr("maketrans", strMaketrans)
	b.SetStr("str", &object.BuiltinFunc{Name: "str", Attrs: strAttrs, Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return &object.Str{V: ""}, nil
		}
		// For exception subclasses with a custom __str__, call it via the interpreter.
		if exc, ok := a[0].(*object.Exception); ok && exc.Class != nil {
			if strFn, found := classLookup(exc.Class, "__str__"); found {
				interp := ii.(*Interp)
				if result, err := interp.callObject(strFn, []object.Object{exc}, nil); err == nil {
					if s, ok := result.(*object.Str); ok {
						return s, nil
					}
				}
			}
		}
		return &object.Str{V: object.Str_(a[0])}, nil
	}})
	b.SetStr("bool", &object.BuiltinFunc{Name: "bool", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return object.False, nil
		}
		return object.BoolOf(object.Truthy(a[0])), nil
	}})
	bytesFromHex := &object.BuiltinFunc{Name: "fromhex", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(ii.(*Interp).typeErr, "bytes.fromhex() takes exactly one argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(ii.(*Interp).typeErr, "bytes.fromhex() argument must be str")
		}
		raw := strings.ReplaceAll(s.V, " ", "")
		if len(raw)%2 != 0 {
			return nil, object.Errorf(ii.(*Interp).valueErr, "non-hexadecimal number found in fromhex() arg")
		}
		out := make([]byte, len(raw)/2)
		for i := 0; i < len(raw); i += 2 {
			hi := hexNibble(raw[i])
			lo := hexNibble(raw[i+1])
			if hi < 0 || lo < 0 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "non-hexadecimal number found in fromhex() arg")
			}
			out[i/2] = byte(hi<<4 | lo)
		}
		return &object.Bytes{V: out}, nil
	}}
	bytesAttrs := object.NewDict()
	bytesAttrs.SetStr("fromhex", bytesFromHex)
	b.SetStr("bytes", &object.BuiltinFunc{Name: "bytes", Attrs: bytesAttrs, Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return &object.Bytes{V: nil}, nil
		}
		if mv, ok := a[0].(*object.Memoryview); ok {
			return &object.Bytes{V: mv.Bytes()}, nil
		}
		if bb, ok := bytesBytesOrArray(a[0]); ok {
			cp := make([]byte, len(bb))
			copy(cp, bb)
			return &object.Bytes{V: cp}, nil
		}
		if n, ok := toInt64(a[0]); ok {
			return &object.Bytes{V: make([]byte, n)}, nil
		}
		items, err := iterate(ii.(*Interp), a[0])
		if err != nil {
			return nil, err
		}
		out := make([]byte, len(items))
		for k, x := range items {
			n, ok := toInt64(x)
			if !ok || n < 0 || n > 255 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "bytes must be in range(0, 256)")
			}
			out[k] = byte(n)
		}
		return &object.Bytes{V: out}, nil
	}})
	b.SetStr("memoryview", &object.BuiltinFunc{Name: "memoryview", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(ii.(*Interp).typeErr, "memoryview() takes exactly one argument")
		}
		switch src := a[0].(type) {
		case *object.Bytes:
			return &object.Memoryview{Backing: src, Start: 0, Stop: len(src.V), Readonly: true}, nil
		case *object.Bytearray:
			src.Views++
			return &object.Memoryview{Backing: src, Start: 0, Stop: len(src.V), Readonly: false}, nil
		case *object.Memoryview:
			return src, nil
		}
		return nil, object.Errorf(ii.(*Interp).typeErr, "memoryview: a bytes-like object is required")
	}})
	bytearrayFromHex := &object.BuiltinFunc{Name: "fromhex", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(ii.(*Interp).typeErr, "bytearray.fromhex() takes exactly one argument")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(ii.(*Interp).typeErr, "bytearray.fromhex() argument must be str")
		}
		raw := strings.ReplaceAll(s.V, " ", "")
		if len(raw)%2 != 0 {
			return nil, object.Errorf(ii.(*Interp).valueErr, "non-hexadecimal number found in fromhex() arg")
		}
		out := make([]byte, len(raw)/2)
		for i := 0; i < len(raw); i += 2 {
			hi := hexNibble(raw[i])
			lo := hexNibble(raw[i+1])
			if hi < 0 || lo < 0 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "non-hexadecimal number found in fromhex() arg")
			}
			out[i/2] = byte(hi<<4 | lo)
		}
		return &object.Bytearray{V: out}, nil
	}}
	bytearrayAttrs := object.NewDict()
	bytearrayAttrs.SetStr("fromhex", bytearrayFromHex)
	b.SetStr("bytearray", &object.BuiltinFunc{Name: "bytearray", Attrs: bytearrayAttrs, Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return &object.Bytearray{V: []byte{}}, nil
		}
		if bb, ok := bytesBytesOrArray(a[0]); ok {
			cp := make([]byte, len(bb))
			copy(cp, bb)
			return &object.Bytearray{V: cp}, nil
		}
		if n, ok := toInt64(a[0]); ok {
			return &object.Bytearray{V: make([]byte, n)}, nil
		}
		items, err := iterate(ii.(*Interp), a[0])
		if err != nil {
			return nil, err
		}
		out := make([]byte, len(items))
		for k, x := range items {
			n, ok := toInt64(x)
			if !ok || n < 0 || n > 255 {
				return nil, object.Errorf(ii.(*Interp).valueErr, "bytes must be in range(0, 256)")
			}
			out[k] = byte(n)
		}
		return &object.Bytearray{V: out}, nil
	}})
	b.SetStr("list", &object.BuiltinFunc{Name: "list", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return &object.List{}, nil
		}
		items, err := iterate(ii.(*Interp), a[0])
		if err != nil {
			return nil, err
		}
		return &object.List{V: items}, nil
	}})
	b.SetStr("tuple", &object.BuiltinFunc{Name: "tuple", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return &object.Tuple{}, nil
		}
		items, err := iterate(ii.(*Interp), a[0])
		if err != nil {
			return nil, err
		}
		return &object.Tuple{V: items}, nil
	}})
	dictFromkeys := &object.BuiltinFunc{Name: "fromkeys", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if len(a) == 0 {
			return nil, object.Errorf(in.typeErr, "dict.fromkeys() requires at least one argument")
		}
		var val object.Object = object.None
		if len(a) >= 2 {
			val = a[1]
		}
		d := object.NewDict()
		items, err := iterate(in, a[0])
		if err != nil {
			return nil, err
		}
		for _, k := range items {
			if err := d.Set(k, val); err != nil {
				return nil, err
			}
		}
		return d, nil
	}}
	dictAttrs := object.NewDict()
	dictAttrs.SetStr("fromkeys", dictFromkeys)
	b.SetStr("dict", &object.BuiltinFunc{Name: "dict", Attrs: dictAttrs, Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		d := object.NewDict()
		if len(a) > 0 {
			var src *object.Dict
			switch x := a[0].(type) {
			case *object.Dict:
				src = x
			case *object.DefaultDict:
				src = x.D
			case *object.OrderedDict:
				src = x.D
			case *object.Counter:
				src = x.D
			}
			if src != nil {
				keys, vals := src.Items()
				for k, key := range keys {
					_ = d.Set(key, vals[k])
				}
			} else {
				items, err := iterate(in, a[0])
				if err != nil {
					return nil, err
				}
				for _, pair := range items {
					var k, v object.Object
					switch p := pair.(type) {
					case *object.Tuple:
						if len(p.V) != 2 {
							return nil, object.Errorf(in.valueErr, "dict update sequence element has length %d; 2 required", len(p.V))
						}
						k, v = p.V[0], p.V[1]
					case *object.List:
						if len(p.V) != 2 {
							return nil, object.Errorf(in.valueErr, "dict update sequence element has length %d; 2 required", len(p.V))
						}
						k, v = p.V[0], p.V[1]
					default:
						return nil, object.Errorf(in.typeErr, "cannot convert dictionary update sequence element to 2-tuple")
					}
					if err := d.Set(k, v); err != nil {
						return nil, err
					}
				}
			}
		}
		if kw != nil {
			keys, vals := kw.Items()
			for k, key := range keys {
				_ = d.Set(key, vals[k])
			}
		}
		return d, nil
	}})
	b.SetStr("set", &object.BuiltinFunc{Name: "set", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s := object.NewSet()
		if len(a) > 0 {
			items, err := iterate(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			for _, x := range items {
				_ = s.Add(x)
			}
		}
		return s, nil
	}})
	b.SetStr("frozenset", &object.BuiltinFunc{Name: "frozenset", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s := object.NewFrozenset()
		if len(a) > 0 {
			items, err := iterate(ii.(*Interp), a[0])
			if err != nil {
				return nil, err
			}
			for _, x := range items {
				if err := s.Add(x); err != nil {
					return nil, err
				}
			}
		}
		return s, nil
	}})

	// Core functions.
	b.SetStr("print", &object.BuiltinFunc{Name: "print", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		sep := " "
		end := "\n"
		if kw != nil {
			if v, ok := kw.GetStr("sep"); ok {
				if s, ok := v.(*object.Str); ok {
					sep = s.V
				}
			}
			if v, ok := kw.GetStr("end"); ok {
				if s, ok := v.(*object.Str); ok {
					end = s.V
				}
			}
		}
		parts := make([]string, len(a))
		for k, x := range a {
			parts[k] = object.Str_(x)
		}
		fmt.Fprint(in.Stdout, strings.Join(parts, sep)+end)
		return object.None, nil
	}})
	b.SetStr("repr", &object.BuiltinFunc{Name: "repr", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return &object.Str{V: ""}, nil
		}
		interp := ii.(*Interp)
		// For exception subclasses with a custom __repr__, call it via the interpreter.
		if exc, ok := a[0].(*object.Exception); ok && exc.Class != nil {
			if reprFn, found := classLookup(exc.Class, "__repr__"); found {
				if result, err := interp.callObject(reprFn, []object.Object{exc}, nil); err == nil {
					if s, ok := result.(*object.Str); ok {
						return s, nil
					}
				}
			}
		}
		// For regular instances, call __repr__ if defined on the class.
		if inst, ok := a[0].(*object.Instance); ok {
			if r, ok2, err := interp.callInstanceDunder(inst, "__repr__"); ok2 && err == nil {
				if s, ok3 := r.(*object.Str); ok3 {
					return s, nil
				}
			}
		}
		return &object.Str{V: object.Repr(a[0])}, nil
	}})
	b.SetStr("len", &object.BuiltinFunc{Name: "len", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, err := ii.(*Interp).length(a[0])
		if err != nil {
			return nil, err
		}
		return object.NewInt(n), nil
	}})
	b.SetStr("range", &object.BuiltinFunc{Name: "range", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		var start, stop, step int64 = 0, 0, 1
		switch len(a) {
		case 1:
			n, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(in.typeErr, "range arg must be int")
			}
			stop = n
		case 2:
			n, _ := toInt64(a[0])
			m, _ := toInt64(a[1])
			start, stop = n, m
		case 3:
			n, _ := toInt64(a[0])
			m, _ := toInt64(a[1])
			s, _ := toInt64(a[2])
			start, stop, step = n, m, s
		default:
			return nil, object.Errorf(in.typeErr, "range requires 1..3 args")
		}
		if step == 0 {
			return nil, object.Errorf(in.valueErr, "range step cannot be zero")
		}
		return &object.Range{Start: start, Stop: stop, Step: step}, nil
	}})
	b.SetStr("iter", &object.BuiltinFunc{Name: "iter", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return ii.(*Interp).getIter(a[0])
	}})
	b.SetStr("next", &object.BuiltinFunc{Name: "next", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if gen, ok := a[0].(*object.Generator); ok {
			v, err := in.resumeGenerator(gen, object.None)
			if err != nil {
				if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, in.stopIter) {
					if len(a) > 1 {
						return a[1], nil
					}
				}
				return nil, err
			}
			return v, nil
		}
		if inst, ok := a[0].(*object.Instance); ok {
			r, handled, err := in.callInstanceDunder(inst, "__next__")
			if !handled {
				return nil, object.Errorf(in.typeErr, "next() arg is not an iterator")
			}
			if err != nil {
				if exc, eok := err.(*object.Exception); eok && exc.Class != nil && object.IsSubclass(exc.Class, in.stopIter) {
					if len(a) > 1 {
						return a[1], nil
					}
					return nil, err
				}
				return nil, err
			}
			return r, nil
		}
		it, ok := a[0].(*object.Iter)
		if !ok {
			return nil, object.Errorf(in.typeErr, "next() arg is not an iterator")
		}
		v, ok2, err := it.Next()
		if err != nil {
			return nil, err
		}
		if !ok2 {
			if len(a) > 1 {
				return a[1], nil
			}
			return nil, object.Errorf(in.stopIter, "")
		}
		return v, nil
	}})
	b.SetStr("pow", &object.BuiltinFunc{Name: "pow", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if len(a) < 2 || len(a) > 3 {
			return nil, object.Errorf(in.typeErr, "pow() takes 2 or 3 arguments")
		}
		if len(a) == 3 {
			// 3-arg form: integers only, modular exponentiation.
			base, ok1 := toBigInt(a[0])
			exp, ok2 := toBigInt(a[1])
			mod, ok3 := toBigInt(a[2])
			if !ok1 || !ok2 || !ok3 {
				return nil, object.Errorf(in.typeErr, "pow() 3rd argument requires integers")
			}
			if mod.Sign() == 0 {
				return nil, object.Errorf(in.valueErr, "pow() 3rd argument cannot be 0")
			}
			return object.IntFromBig(new(big.Int).Exp(base, exp, mod)), nil
		}
		return in.pow(a[0], a[1])
	}})
	b.SetStr("format", &object.BuiltinFunc{Name: "format", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		spec := ""
		if len(a) > 1 {
			if s, ok := a[1].(*object.Str); ok {
				spec = s.V
			}
		}
		if r, ok, err := in.instanceFormat(a[0], spec); ok {
			if err != nil {
				return nil, err
			}
			return &object.Str{V: r}, nil
		}
		if spec == "" {
			return &object.Str{V: object.Str_(a[0])}, nil
		}
		s, err := formatValue(a[0], spec)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: s}, nil
	}})
	b.SetStr("ascii", &object.BuiltinFunc{Name: "ascii", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: asciiRepr(a[0])}, nil
	}})
	b.SetStr("slice", &object.BuiltinFunc{Name: "slice", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		switch len(a) {
		case 1:
			return &object.Slice{Start: object.None, Stop: a[0], Step: object.None}, nil
		case 2:
			return &object.Slice{Start: a[0], Stop: a[1], Step: object.None}, nil
		case 3:
			return &object.Slice{Start: a[0], Stop: a[1], Step: a[2]}, nil
		}
		return nil, object.Errorf(ii.(*Interp).typeErr, "slice expected 1..3 arguments")
	}})
	b.SetStr("dir", &object.BuiltinFunc{Name: "dir", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.List{V: dirOf(a)}, nil
	}})
	b.SetStr("delattr", &object.BuiltinFunc{Name: "delattr", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		name, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(in.typeErr, "delattr: attribute name must be string")
		}
		var d *object.Dict
		switch obj := a[0].(type) {
		case *object.Instance:
			d = obj.Dict
		case *object.Class:
			d = obj.Dict
		case *object.Module:
			d = obj.Dict
		default:
			return nil, object.Errorf(in.attrErr, "'%s' object has no attribute '%s'", object.TypeName(a[0]), name.V)
		}
		ok2, err := d.Delete(&object.Str{V: name.V})
		if err != nil {
			return nil, err
		}
		if !ok2 {
			return nil, object.Errorf(in.attrErr, "%s", name.V)
		}
		return object.None, nil
	}})
	b.SetStr("abs", &object.BuiltinFunc{Name: "abs", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if inst, ok := a[0].(*object.Instance); ok {
			if r, ok, err := in.callInstanceDunder(inst, "__abs__"); ok {
				return r, err
			}
		}
		switch v := a[0].(type) {
		case *object.Int:
			return object.IntFromBig(new(big.Int).Abs(&v.V)), nil
		case *object.Float:
			if v.V < 0 {
				return &object.Float{V: -v.V}, nil
			}
			return v, nil
		case *object.Bool:
			if v.V {
				return object.NewInt(1), nil
			}
			return object.NewInt(0), nil
		case *object.Complex:
			return &object.Float{V: math.Hypot(v.Real, v.Imag)}, nil
		}
		return nil, object.Errorf(in.typeErr, "bad abs() arg")
	}})
	b.SetStr("complex", &object.BuiltinFunc{Name: "complex", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		re, im := 0.0, 0.0
		if len(a) >= 1 {
			r, i2, ok := asComplex(a[0])
			if !ok {
				return nil, object.Errorf(in.typeErr, "complex() first argument must be a number, not '%s'", object.TypeName(a[0]))
			}
			re, im = r, i2
		}
		if len(a) >= 2 {
			br, bi, ok := asComplex(a[1])
			if !ok {
				return nil, object.Errorf(in.typeErr, "complex() second argument must be a number, not '%s'", object.TypeName(a[1]))
			}
			// complex(r, i): result is (r.real - i.imag) + (r.imag + i.real)j
			re, im = re-bi, im+br
		}
		return &object.Complex{Real: re, Imag: im}, nil
	}})
	b.SetStr("min", &object.BuiltinFunc{Name: "min", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return reduceMinMax(ii.(*Interp), a, true)
	}})
	b.SetStr("max", &object.BuiltinFunc{Name: "max", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return reduceMinMax(ii.(*Interp), a, false)
	}})
	b.SetStr("sum", &object.BuiltinFunc{Name: "sum", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		items, err := iterate(in, a[0])
		if err != nil {
			return nil, err
		}
		var acc object.Object = object.NewInt(0)
		if len(a) > 1 {
			acc = a[1]
		}
		for _, x := range items {
			acc, err = in.add(acc, x)
			if err != nil {
				return nil, err
			}
		}
		return acc, nil
	}})
	b.SetStr("sorted", &object.BuiltinFunc{Name: "sorted", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		items, err := iterate(in, a[0])
		if err != nil {
			return nil, err
		}
		cp := make([]object.Object, len(items))
		copy(cp, items)
		reverse := false
		var key object.Object
		if kw != nil {
			if v, ok := kw.GetStr("reverse"); ok {
				reverse = object.Truthy(v)
			}
			if v, ok := kw.GetStr("key"); ok {
				if _, isNone := v.(*object.NoneType); !isNone {
					key = v
				}
			}
		}
		if err := sortListKey(in, cp, key, reverse); err != nil {
			return nil, err
		}
		return &object.List{V: cp}, nil
	}})
	b.SetStr("reversed", &object.BuiltinFunc{Name: "reversed", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		items, err := iterate(ii.(*Interp), a[0])
		if err != nil {
			return nil, err
		}
		out := make([]object.Object, len(items))
		for k := range items {
			out[k] = items[len(items)-1-k]
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(out) {
				return nil, false, nil
			}
			r := out[idx]
			idx++
			return r, true, nil
		}}, nil
	}})
	b.SetStr("enumerate", &object.BuiltinFunc{Name: "enumerate", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		it, err := in.getIter(a[0])
		if err != nil {
			return nil, err
		}
		start := int64(0)
		if len(a) > 1 {
			if n, ok := toInt64(a[1]); ok {
				start = n
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("start"); ok {
				if n, ok := toInt64(v); ok {
					start = n
				}
			}
		}
		return &object.Iter{Next: func() (object.Object, bool, error) {
			v, ok, err := it.Next()
			if err != nil || !ok {
				return nil, ok, err
			}
			t := &object.Tuple{V: []object.Object{object.NewInt(start), v}}
			start++
			return t, true, nil
		}}, nil
	}})
	b.SetStr("zip", &object.BuiltinFunc{Name: "zip", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		iters := make([]*object.Iter, len(a))
		for k, x := range a {
			it, err := in.getIter(x)
			if err != nil {
				return nil, err
			}
			iters[k] = it
		}
		return &object.Iter{Next: func() (object.Object, bool, error) {
			items := make([]object.Object, len(iters))
			for k, it := range iters {
				v, ok, err := it.Next()
				if err != nil {
					return nil, false, err
				}
				if !ok {
					return nil, false, nil
				}
				items[k] = v
			}
			return &object.Tuple{V: items}, true, nil
		}}, nil
	}})
	b.SetStr("map", &object.BuiltinFunc{Name: "map", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		fn := a[0]
		iters := make([]*object.Iter, len(a)-1)
		for k, x := range a[1:] {
			it, err := in.getIter(x)
			if err != nil {
				return nil, err
			}
			iters[k] = it
		}
		return &object.Iter{Next: func() (object.Object, bool, error) {
			args := make([]object.Object, len(iters))
			for k, it := range iters {
				v, ok, err := it.Next()
				if err != nil || !ok {
					return nil, ok, err
				}
				args[k] = v
			}
			r, err := in.callObject(fn, args, nil)
			if err != nil {
				return nil, false, err
			}
			return r, true, nil
		}}, nil
	}})
	b.SetStr("filter", &object.BuiltinFunc{Name: "filter", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		fn := a[0]
		it, err := in.getIter(a[1])
		if err != nil {
			return nil, err
		}
		return &object.Iter{Next: func() (object.Object, bool, error) {
			for {
				v, ok, err := it.Next()
				if err != nil || !ok {
					return nil, ok, err
				}
				keep := false
				if _, isNone := fn.(*object.NoneType); isNone {
					keep = object.Truthy(v)
				} else {
					r, err := in.callObject(fn, []object.Object{v}, nil)
					if err != nil {
						return nil, false, err
					}
					keep = object.Truthy(r)
				}
				if keep {
					return v, true, nil
				}
			}
		}}, nil
	}})
	b.SetStr("any", &object.BuiltinFunc{Name: "any", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		items, err := iterate(ii.(*Interp), a[0])
		if err != nil {
			return nil, err
		}
		for _, x := range items {
			if object.Truthy(x) {
				return object.True, nil
			}
		}
		return object.False, nil
	}})
	b.SetStr("all", &object.BuiltinFunc{Name: "all", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		items, err := iterate(ii.(*Interp), a[0])
		if err != nil {
			return nil, err
		}
		for _, x := range items {
			if !object.Truthy(x) {
				return object.False, nil
			}
		}
		return object.True, nil
	}})
	b.SetStr("isinstance", &object.BuiltinFunc{Name: "isinstance", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(isinstance(a[0], a[1])), nil
	}})
	b.SetStr("issubclass", &object.BuiltinFunc{Name: "issubclass", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		ca, okA := a[0].(*object.Class)
		// Second arg may be a tuple of classes.
		if tup, ok := a[1].(*object.Tuple); ok {
			for _, tt := range tup.V {
				if c2, ok2 := tt.(*object.Class); ok2 && okA {
					if object.IsSubclass(ca, c2) {
						return object.True, nil
					}
					if regsT, ok3 := abcGetRegistered(c2); ok3 {
						for _, r := range regsT {
							if object.IsSubclass(ca, r) {
								return object.True, nil
							}
						}
					}
				}
			}
			return object.False, nil
		}
		cb, okB := a[1].(*object.Class)
		if !okA || !okB {
			return object.False, nil
		}
		// Structural subclass check.
		if object.IsSubclass(ca, cb) {
			return object.True, nil
		}
		// Check __subclasshook__ on the parent class.
		if hookVal, ok2 := classLookup(cb, "__subclasshook__"); ok2 {
			var hookResult object.Object
			var hookErr error
			switch hv := hookVal.(type) {
			case *object.ClassMethod:
				hookResult, hookErr = in.callObject(hv.Fn, []object.Object{cb, ca}, nil)
			case *object.BoundMethod:
				hookResult, hookErr = in.callObject(hv.Fn, []object.Object{hv.Self, ca}, nil)
			default:
				hookResult, hookErr = in.callObject(hookVal, []object.Object{ca}, nil)
			}
			if hookErr == nil && hookResult != object.NotImplemented && hookResult != nil {
				return object.BoolOf(isTruthy(hookResult)), nil
			}
		}
		// ABC virtual subclass registry check.
		if regs, ok3 := abcGetRegistered(cb); ok3 {
			for _, r := range regs {
				if object.IsSubclass(ca, r) {
					return object.True, nil
				}
			}
		}
		return object.False, nil
	}})
	b.SetStr("type", &object.BuiltinFunc{Name: "type", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if inst, ok := a[0].(*object.Instance); ok {
			return inst.Class, nil
		}
		if exc, ok := a[0].(*object.Exception); ok {
			return exc.Class, nil
		}
		// Fallback: synthesize a bare class with just the name; good enough
		// for `type(x).__name__` on builtin types.
		return &object.Class{Name: object.TypeName(a[0]), Dict: object.NewDict()}, nil
	}})
	b.SetStr("id", &object.BuiltinFunc{Name: "id", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(fmt.Sprintf("%p", a[0])[2] ^ 0x42)), nil
	}})
	b.SetStr("hash", &object.BuiltinFunc{Name: "hash", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		h, err := object.Hash(a[0])
		if err != nil {
			return nil, object.Errorf(ii.(*Interp).typeErr, "unhashable type: '%s'", object.TypeName(a[0]))
		}
		return object.NewInt(int64(h)), nil
	}})
	b.SetStr("ord", &object.BuiltinFunc{Name: "ord", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(ii.(*Interp).typeErr, "ord expects str")
		}
		rs := s.Runes()
		if len(rs) != 1 {
			return nil, object.Errorf(ii.(*Interp).typeErr, "ord expects single char")
		}
		return object.NewInt(int64(rs[0])), nil
	}})
	b.SetStr("chr", &object.BuiltinFunc{Name: "chr", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, _ := toInt64(a[0])
		return &object.Str{V: string(rune(n))}, nil
	}})
	b.SetStr("hex", &object.BuiltinFunc{Name: "hex", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, _ := toBigInt(a[0])
		if n.Sign() < 0 {
			return &object.Str{V: "-0x" + new(big.Int).Abs(n).Text(16)}, nil
		}
		return &object.Str{V: "0x" + n.Text(16)}, nil
	}})
	b.SetStr("oct", &object.BuiltinFunc{Name: "oct", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, _ := toBigInt(a[0])
		if n.Sign() < 0 {
			return &object.Str{V: "-0o" + new(big.Int).Abs(n).Text(8)}, nil
		}
		return &object.Str{V: "0o" + n.Text(8)}, nil
	}})
	b.SetStr("bin", &object.BuiltinFunc{Name: "bin", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n, _ := toBigInt(a[0])
		if n.Sign() < 0 {
			return &object.Str{V: "-0b" + new(big.Int).Abs(n).Text(2)}, nil
		}
		return &object.Str{V: "0b" + n.Text(2)}, nil
	}})
	b.SetStr("round", &object.BuiltinFunc{Name: "round", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if inst, ok := a[0].(*object.Instance); ok {
			args := []object.Object{}
			if len(a) > 1 {
				args = append(args, a[1])
			}
			if r, ok, err := in.callInstanceDunder(inst, "__round__", args...); ok {
				return r, err
			}
		}
		ndigits := 0
		if len(a) > 1 {
			if n, ok := toInt64(a[1]); ok {
				ndigits = int(n)
			}
		}
		switch v := a[0].(type) {
		case *object.Float:
			mult := 1.0
			for k := 0; k < ndigits; k++ {
				mult *= 10
			}
			for k := 0; k < -ndigits; k++ {
				mult /= 10
			}
			r := v.V * mult
			// Python's banker's rounding; we approximate with math.Round.
			if r >= 0 {
				r = float64(int64(r + 0.5))
			} else {
				r = float64(int64(r - 0.5))
			}
			return &object.Float{V: r / mult}, nil
		case *object.Int:
			return v, nil
		case *object.Bool:
			if v.V {
				return object.NewInt(1), nil
			}
			return object.NewInt(0), nil
		}
		return nil, nil
	}})
	b.SetStr("getattr", &object.BuiltinFunc{Name: "getattr", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		name := a[1].(*object.Str).V
		v, err := ii.(*Interp).getAttr(a[0], name)
		if err != nil {
			if len(a) > 2 {
				return a[2], nil
			}
			return nil, err
		}
		return v, nil
	}})
	b.SetStr("setattr", &object.BuiltinFunc{Name: "setattr", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, ii.(*Interp).setAttr(a[0], a[1].(*object.Str).V, a[2])
	}})
	b.SetStr("hasattr", &object.BuiltinFunc{Name: "hasattr", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		_, err := ii.(*Interp).getAttr(a[0], a[1].(*object.Str).V)
		return object.BoolOf(err == nil), nil
	}})
	b.SetStr("property", &object.BuiltinFunc{Name: "property", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		p := &object.Property{}
		if len(a) > 0 {
			p.Fget = a[0]
		}
		if len(a) > 1 {
			p.Fset = a[1]
		}
		if len(a) > 2 {
			p.Fdel = a[2]
		}
		return p, nil
	}})
	b.SetStr("classmethod", &object.BuiltinFunc{Name: "classmethod", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.ClassMethod{Fn: a[0]}, nil
	}})
	b.SetStr("staticmethod", &object.BuiltinFunc{Name: "staticmethod", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.StaticMethod{Fn: a[0]}, nil
	}})
	// super is a stub — LOAD_SUPER_ATTR handles the real work by receiving
	// (super, __class__, self) on the stack.
	b.SetStr("super", &object.BuiltinFunc{Name: "super", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	b.SetStr("callable", &object.BuiltinFunc{Name: "callable", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		switch v := a[0].(type) {
		case *object.BuiltinFunc, *object.Function, *object.BoundMethod, *object.Class:
			return object.True, nil
		case *object.Instance:
			if _, ok := v.Class.Dict.GetStr("__call__"); ok {
				return object.True, nil
			}
		}
		return object.False, nil
	}})
	b.SetStr("divmod", &object.BuiltinFunc{Name: "divmod", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if inst, ok := a[0].(*object.Instance); ok {
			if r, ok, err := in.callInstanceDunder(inst, "__divmod__", a[1]); ok {
				return r, err
			}
		}
		if inst, ok := a[1].(*object.Instance); ok {
			if r, ok, err := in.callInstanceDunder(inst, "__rdivmod__", a[0]); ok {
				return r, err
			}
		}
		q, err := in.floordiv(a[0], a[1])
		if err != nil {
			return nil, err
		}
		r, err := in.mod(a[0], a[1])
		if err != nil {
			return nil, err
		}
		return &object.Tuple{V: []object.Object{q, r}}, nil
	}})
	// object — the root base class.
	objectClass := &object.Class{Name: "object", Dict: object.NewDict()}
	b.SetStr("object", objectClass)

	// globals() — return the calling frame's global namespace.
	b.SetStr("globals", &object.BuiltinFunc{Name: "globals", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if in.curFrame == nil {
			return object.NewDict(), nil
		}
		return in.curFrame.Globals, nil
	}})

	// locals() — snapshot of the calling frame's local namespace.
	b.SetStr("locals", &object.BuiltinFunc{Name: "locals", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		return frameLocals(in.curFrame), nil
	}})

	// vars([obj]) — obj.__dict__ or locals() if no argument.
	b.SetStr("vars", &object.BuiltinFunc{Name: "vars", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if len(a) == 0 {
			return frameLocals(in.curFrame), nil
		}
		switch v := a[0].(type) {
		case *object.Instance:
			return v.Dict, nil
		case *object.Module:
			return v.Dict, nil
		case *object.Class:
			return v.Dict, nil
		default:
			return nil, object.Errorf(in.typeErr, "vars() argument must have __dict__ attribute")
		}
	}})

	// input([prompt]) — read one line from stdin.
	b.SetStr("input", &object.BuiltinFunc{Name: "input", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if len(a) >= 1 {
			fmt.Fprint(in.Stdout, object.Str_(a[0]))
		}
		var line []byte
		buf := make([]byte, 1)
		for {
			_, err := os.Stdin.Read(buf)
			if err != nil {
				if len(line) == 0 {
					return nil, object.Errorf(in.eofErr, "EOF when reading a line")
				}
				break
			}
			if buf[0] == '\n' {
				break
			}
			line = append(line, buf[0])
		}
		return &object.Str{V: string(line)}, nil
	}})

	// aiter(obj) — call obj.__aiter__().
	b.SetStr("aiter", &object.BuiltinFunc{Name: "aiter", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if len(a) < 1 {
			return nil, object.Errorf(in.typeErr, "aiter() requires 1 argument")
		}
		fn, err := in.getAttr(a[0], "__aiter__")
		if err != nil {
			return nil, object.Errorf(in.typeErr, "'%.100s' object is not an async iterable", object.TypeName(a[0]))
		}
		return in.callObject(fn, nil, nil)
	}})

	// anext(obj[, default]) — call obj.__anext__(); return default on StopAsyncIteration.
	b.SetStr("anext", &object.BuiltinFunc{Name: "anext", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		if len(a) < 1 {
			return nil, object.Errorf(in.typeErr, "anext() requires at least 1 argument")
		}
		fn, err := in.getAttr(a[0], "__anext__")
		if err != nil {
			return nil, object.Errorf(in.typeErr, "'%.100s' object is not an async iterator", object.TypeName(a[0]))
		}
		r, callErr := in.callObject(fn, nil, nil)
		if callErr != nil {
			exc, ok := callErr.(*object.Exception)
			if ok && len(a) >= 2 && object.IsSubclass(exc.Class, in.stopAsyncIter) {
				return a[1], nil
			}
			return nil, callErr
		}
		return r, nil
	}})

	// breakpoint() — no debugger in goipy; write a notice and return None.
	b.SetStr("breakpoint", &object.BuiltinFunc{Name: "breakpoint", Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		fmt.Fprintln(in.Stderr, "breakpoint() is a no-op in goipy")
		return object.None, nil
	}})

	// help([obj]) — pydoc not available; print a stub notice to stderr.
	b.SetStr("help", &object.BuiltinFunc{Name: "help", Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		fmt.Fprintln(in.Stderr, "help() not available in goipy")
		return object.None, nil
	}})

	// open(file, mode='r', ...) — basic file I/O.
	b.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		in := ii.(*Interp)
		return builtinOpen(in, a, kw)
	}})

	b.SetStr("__build_class__", &object.BuiltinFunc{Name: "__build_class__", Call: func(ii any, a []object.Object, kwds *object.Dict) (object.Object, error) {
		// args: func, name, *bases, **kwds
		in := ii.(*Interp)
		fn := a[0].(*object.Function)
		name := a[1].(*object.Str).V
		var bases []*object.Class
		for _, b := range a[2:] {
			if cls, ok := b.(*object.Class); ok {
				bases = append(bases, cls)
			}
		}
		// Execute class body in a fresh dict.
		ns := object.NewDict()
		frame := NewFrame(fn.Code, fn.Globals, in.Builtins, ns)
		if _, err := in.runFrame(frame); err != nil {
			return nil, err
		}
		cls := &object.Class{Name: name, Bases: bases, Dict: ns}
		// Populate __class__ cell so zero-arg super() in methods resolves.
		if v, ok := ns.GetStr("__classcell__"); ok {
			if c, ok := v.(*object.Cell); ok {
				c.V = cls
				c.Set = true
			}
		}
		// __set_name__: notify descriptors of the attribute name they were
		// bound to.
		keys, vals := ns.Items()
		for idx, k := range keys {
			kstr, ok := k.(*object.Str)
			if !ok {
				continue
			}
			if cp, ok := vals[idx].(*cachedProperty); ok {
				cp.name = kstr.V
				continue
			}
			dinst, ok := vals[idx].(*object.Instance)
			if !ok {
				continue
			}
			if fn, ok := classLookup(dinst.Class, "__set_name__"); ok {
				if _, err := in.callObject(fn, []object.Object{dinst, cls, &object.Str{V: kstr.V}}, nil); err != nil {
					return nil, err
				}
			}
		}
		// __init_subclass__: walk MRO of bases (excluding cls itself) and
		// invoke the first __init_subclass__ found, treated as an implicit
		// classmethod. Class-definition kwargs (Class Foo(Base, tag=...)) are
		// forwarded as-is.
		for _, base := range bases {
			if fn, ok := classLookup(base, "__init_subclass__"); ok {
				if _, err := in.callObject(fn, []object.Object{cls}, kwds); err != nil {
					return nil, err
				}
				break
			}
		}
		return cls, nil
	}})
}

func reduceMinMax(in *Interp, a []object.Object, isMin bool) (object.Object, error) {
	var items []object.Object
	if len(a) == 1 {
		its, err := iterate(in, a[0])
		if err != nil {
			return nil, err
		}
		items = its
	} else {
		items = a
	}
	if len(items) == 0 {
		return nil, object.Errorf(in.valueErr, "min/max of empty sequence")
	}
	best := items[0]
	for _, x := range items[1:] {
		less, err := in.lt(x, best)
		if err != nil {
			return nil, err
		}
		if isMin && less {
			best = x
		}
		if !isMin && !less {
			eq, _ := object.Eq(x, best)
			if !eq {
				best = x
			}
		}
	}
	return best, nil
}

func isinstance(o, t object.Object) bool {
	if cls, ok := t.(*object.Class); ok {
		// ABC structural check: if the class defines ABCCheck, try it first.
		if cls.ABCCheck != nil && cls.ABCCheck(o) {
			return true
		}
		if inst, ok := o.(*object.Instance); ok {
			return object.IsSubclass(inst.Class, cls)
		}
		if e, ok := o.(*object.Exception); ok {
			return object.IsSubclass(e.Class, cls)
		}
		return false
	}
	if tup, ok := t.(*object.Tuple); ok {
		for _, tt := range tup.V {
			if isinstance(o, tt) {
				return true
			}
		}
	}
	// check by type name for builtin types
	if s, ok := t.(*object.Str); ok {
		return object.TypeName(o) == s.V
	}
	if bf, ok := t.(*object.BuiltinFunc); ok {
		return matchBuiltinType(o, bf.Name)
	}
	return false
}

// asciiRepr returns Repr(o) with any non-ASCII character escaped as \xHH or
// \uHHHH. Mirrors Python's ascii() builtin.
func asciiRepr(o object.Object) string {
	r := object.Repr(o)
	var buf strings.Builder
	for _, c := range r {
		switch {
		case c < 0x80:
			buf.WriteRune(c)
		case c <= 0xff:
			fmt.Fprintf(&buf, "\\x%02x", c)
		case c <= 0xffff:
			fmt.Fprintf(&buf, "\\u%04x", c)
		default:
			fmt.Fprintf(&buf, "\\U%08x", c)
		}
	}
	return buf.String()
}

// dictStrKeys returns all string-keyed names from d.
func dictStrKeys(d *object.Dict) []string {
	keys, _ := d.Items()
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if s, ok := k.(*object.Str); ok {
			out = append(out, s.V)
		}
	}
	return out
}

// dirOf returns a sorted list of attribute names for an object. With no
// argument, returns the empty list (we don't expose the caller's frame).
func dirOf(args []object.Object) []object.Object {
	if len(args) == 0 {
		return nil
	}
	var names []string
	switch v := args[0].(type) {
	case *object.Module:
		names = dictStrKeys(v.Dict)
	case *object.Instance:
		names = append(names, dictStrKeys(v.Dict)...)
		for c := v.Class; c != nil; {
			names = append(names, dictStrKeys(c.Dict)...)
			if len(c.Bases) == 0 {
				break
			}
			c = c.Bases[0]
		}
	case *object.Class:
		names = dictStrKeys(v.Dict)
	case *object.Dict:
		names = dictStrKeys(v)
	}
	seen := map[string]bool{}
	out := []object.Object{}
	sorted := make([]string, 0, len(names))
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			sorted = append(sorted, n)
		}
	}
	sortStrings(sorted)
	for _, n := range sorted {
		out = append(out, &object.Str{V: n})
	}
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// frameLocals builds a snapshot dict of the locals visible in frame f. For
// function frames the live values come from f.Fast; for module/class frames
// they come from f.Locals.
func frameLocals(f *Frame) *object.Dict {
	d := object.NewDict()
	if f == nil {
		return d
	}
	// Fast locals (function frames).
	for idx, name := range f.Code.LocalsPlusNames {
		if idx < len(f.Fast) && f.Fast[idx] != nil {
			d.SetStr(name, f.Fast[idx])
		}
	}
	// Locals dict (module / class body frames).
	if f.Locals != nil && f.Locals != f.Globals {
		keys, vals := f.Locals.Items()
		for i, k := range keys {
			if s, ok := k.(*object.Str); ok {
				d.SetStr(s.V, vals[i])
			}
		}
	}
	return d
}

// builtinOpen implements open(file, mode='r', ...).
func builtinOpen(in *Interp, a []object.Object, kw *object.Dict) (object.Object, error) {
	if len(a) < 1 {
		return nil, object.Errorf(in.typeErr, "open() requires at least 1 argument")
	}
	path, ok := a[0].(*object.Str)
	if !ok {
		return nil, object.Errorf(in.typeErr, "open() path must be str")
	}

	mode := "r"
	if len(a) >= 2 {
		ms, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(in.typeErr, "open() mode must be str")
		}
		mode = ms.V
	} else if kw != nil {
		if v, ok2 := kw.GetStr("mode"); ok2 {
			if ms, ok3 := v.(*object.Str); ok3 {
				mode = ms.V
			}
		}
	}

	// Audit event: open(path, mode, flags)
	if err := in.fireAudit("open", []object.Object{path, &object.Str{V: mode}, object.NewInt(0)}); err != nil {
		return nil, err
	}

	binary := strings.ContainsRune(mode, 'b')

	var flag int
	switch {
	case strings.ContainsRune(mode, 'w'):
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	case strings.ContainsRune(mode, 'a'):
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	case strings.ContainsRune(mode, '+'):
		flag = os.O_RDWR | os.O_CREATE
	default:
		flag = os.O_RDONLY
	}

	f, err := os.OpenFile(path.V, flag, 0o666)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, object.Errorf(in.fileNotFoundErr, "[Errno 2] No such file or directory: '%s'", path.V)
		}
		return nil, object.Errorf(in.osErr, "%v", err)
	}
	return &object.File{F: f, FilePath: path.V, Mode: mode, Binary: binary}, nil
}

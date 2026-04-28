package vm

import (
	"math/big"

	"github.com/tamnd/goipy/object"
)

// buildPickletools constructs the pickletools module with CPython 3.14 API.
func (i *Interp) buildPickletools() *object.Module {
	m := &object.Module{Name: "pickletools", Dict: object.NewDict()}
	d := m.Dict

	// ── StackObject class ─────────────────────────────────────────────────────

	stackObjCls := &object.Class{
		Name:  "StackObject",
		Bases: []*object.Class{},
		Dict:  object.NewDict(),
	}
	stackObjCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			if inst, ok := a[0].(*object.Instance); ok {
				if len(a) > 1 {
					inst.Dict.SetStr("name", a[1])
				}
				if len(a) > 2 {
					inst.Dict.SetStr("doc", a[2])
				}
			}
			return object.None, nil
		},
	})
	d.SetStr("StackObject", stackObjCls)

	// ── ArgumentDescriptor class ──────────────────────────────────────────────

	argDescCls := &object.Class{
		Name:  "ArgumentDescriptor",
		Bases: []*object.Class{},
		Dict:  object.NewDict(),
	}
	argDescCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			if inst, ok := a[0].(*object.Instance); ok {
				if len(a) > 1 {
					inst.Dict.SetStr("name", a[1])
				}
				if len(a) > 2 {
					inst.Dict.SetStr("n", a[2])
				}
				inst.Dict.SetStr("reader", object.None)
				inst.Dict.SetStr("doc", &object.Str{V: ""})
			}
			return object.None, nil
		},
	})
	d.SetStr("ArgumentDescriptor", argDescCls)

	// ── OpcodeInfo class ──────────────────────────────────────────────────────

	opcodeInfoCls := &object.Class{
		Name:  "OpcodeInfo",
		Bases: []*object.Class{},
		Dict:  object.NewDict(),
	}
	opcodeInfoCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			if inst, ok := a[0].(*object.Instance); ok {
				names := []string{"name", "code", "arg", "stack_before", "stack_after", "proto", "doc"}
				for idx, nm := range names {
					if idx+1 < len(a) {
						inst.Dict.SetStr(nm, a[idx+1])
					} else {
						inst.Dict.SetStr(nm, object.None)
					}
				}
			}
			return object.None, nil
		},
	})
	d.SetStr("OpcodeInfo", opcodeInfoCls)

	// ── _Example class ────────────────────────────────────────────────────────

	exampleCls := &object.Class{
		Name:  "_Example",
		Bases: []*object.Class{},
		Dict:  object.NewDict(),
	}
	d.SetStr("_Example", exampleCls)

	// ── StackObject singletons ────────────────────────────────────────────────

	mkSO := func(name, doc string) *object.Instance {
		inst := &object.Instance{Class: stackObjCls, Dict: object.NewDict()}
		inst.Dict.SetStr("name", &object.Str{V: name})
		inst.Dict.SetStr("doc", &object.Str{V: doc})
		return inst
	}

	soIntOrBool := mkSO("int_or_bool", "A Python integer or boolean object.")
	soInt := mkSO("int", "A Python integer object.")
	soBytesOrStr := mkSO("bytes_or_str", "A Python bytes or (Unicode) string object.")
	soBytes := mkSO("bytes", "A Python bytes object.")
	soBytearray := mkSO("bytearray", "A Python bytearray object.")
	soBuffer := mkSO("buffer", "A Python buffer-like object.")
	soNone := mkSO("None", "The Python None object.")
	soBool := mkSO("bool", "A Python boolean object.")
	soStr := mkSO("str", "A Python (Unicode) string object.")
	soFloat := mkSO("float", "A Python float object.")
	soList := mkSO("list", "A Python list object.")
	soAny := mkSO("any", "Any kind of object whatsoever.")
	soMark := mkSO("mark", "'The mark' is a unique object.")
	soStackslice := mkSO("stackslice", "An object representing a contiguous slice of the stack.")
	soTuple := mkSO("tuple", "A Python tuple object.")
	soDict := mkSO("dict", "A Python dict object.")
	soSet := mkSO("set", "A Python set object.")
	soFrozenset := mkSO("frozenset", "A Python frozenset object.")

	soList_ := func(names ...string) *object.List {
		objs := make([]object.Object, len(names))
		soMap := map[string]*object.Instance{
			"int_or_bool": soIntOrBool, "int": soInt, "bytes_or_str": soBytesOrStr,
			"bytes": soBytes, "bytearray": soBytearray, "buffer": soBuffer,
			"None": soNone, "bool": soBool, "str": soStr, "float": soFloat,
			"list": soList, "any": soAny, "mark": soMark, "stackslice": soStackslice,
			"tuple": soTuple, "dict": soDict, "set": soSet, "frozenset": soFrozenset,
		}
		for k, n := range names {
			objs[k] = soMap[n]
		}
		return &object.List{V: objs}
	}

	// ── ArgumentDescriptor singletons ────────────────────────────────────────

	mkAD := func(name string, n int) *object.Instance {
		inst := &object.Instance{Class: argDescCls, Dict: object.NewDict()}
		inst.Dict.SetStr("name", &object.Str{V: name})
		inst.Dict.SetStr("n", intObj(int64(n)))
		inst.Dict.SetStr("reader", object.None)
		inst.Dict.SetStr("doc", &object.Str{V: ""})
		return inst
	}

	adDecimalnlShort := mkAD("decimalnl_short", -1)
	adDecimalnlLong := mkAD("decimalnl_long", -1)
	adInt4 := mkAD("int4", 4)
	adUint1 := mkAD("uint1", 1)
	adUint2 := mkAD("uint2", 2)
	adUint4 := mkAD("uint4", 4)
	adUint8 := mkAD("uint8", 8)
	adLong1 := mkAD("long1", -1)
	adLong4 := mkAD("long4", -1)
	adString1 := mkAD("string1", -1)
	adString4 := mkAD("string4", -1)
	adStringnl := mkAD("stringnl", -1)
	adStringnlNoescape := mkAD("stringnl_noescape", -1)
	adStringnlNoescapePair := mkAD("stringnl_noescape_pair", -1)
	adBytes1 := mkAD("bytes1", -1)
	adBytes4 := mkAD("bytes4", -1)
	adBytes8 := mkAD("bytes8", -1)
	adBytearray8 := mkAD("bytearray8", -1)
	adFloat8 := mkAD("float8", 8)
	adFloatnl := mkAD("floatnl", -1)
	adUnicodestring1 := mkAD("unicodestring1", -1)
	adUnicodestring4 := mkAD("unicodestring4", -1)
	adUnicodestring8 := mkAD("unicodestring8", -1)
	adUnicodestringnl := mkAD("unicodestringnl", -1)

	// ── opcodes list ──────────────────────────────────────────────────────────

	mkOI := func(name, code string, arg object.Object, sb, sa *object.List, proto int, doc string) *object.Instance {
		inst := &object.Instance{Class: opcodeInfoCls, Dict: object.NewDict()}
		inst.Dict.SetStr("name", &object.Str{V: name})
		inst.Dict.SetStr("code", &object.Str{V: code})
		if arg == nil {
			inst.Dict.SetStr("arg", object.None)
		} else {
			inst.Dict.SetStr("arg", arg)
		}
		inst.Dict.SetStr("stack_before", sb)
		inst.Dict.SetStr("stack_after", sa)
		inst.Dict.SetStr("proto", intObj(int64(proto)))
		inst.Dict.SetStr("doc", &object.Str{V: doc})
		return inst
	}

	nil_ := object.Object(nil) // for readability below
	_ = nil_

	opcodeList := []object.Object{
		mkOI("INT", "I", adDecimalnlShort, soList_(), soList_("int_or_bool"), 0, "Push an integer or bool."),
		mkOI("BININT", "J", adInt4, soList_(), soList_("int"), 1, "Push a four-byte signed integer."),
		mkOI("BININT1", "K", adUint1, soList_(), soList_("int"), 1, "Push a one-byte unsigned integer."),
		mkOI("BININT2", "M", adUint2, soList_(), soList_("int"), 1, "Push a two-byte unsigned integer."),
		mkOI("LONG", "L", adDecimalnlLong, soList_(), soList_("int"), 0, "Push a long integer."),
		mkOI("LONG1", "\x8a", adLong1, soList_(), soList_("int"), 2, "Long integer using one-byte length."),
		mkOI("LONG4", "\x8b", adLong4, soList_(), soList_("int"), 2, "Long integer using four-byte length."),
		mkOI("STRING", "S", adStringnl, soList_(), soList_("bytes_or_str"), 0, "Push a Python string object."),
		mkOI("BINSTRING", "T", adString4, soList_(), soList_("bytes_or_str"), 1, "Push a Python string object."),
		mkOI("SHORT_BINSTRING", "U", adString1, soList_(), soList_("bytes_or_str"), 1, "Push a Python string object."),
		mkOI("BINBYTES", "B", adBytes4, soList_(), soList_("bytes"), 3, "Push a Python bytes object."),
		mkOI("SHORT_BINBYTES", "C", adBytes1, soList_(), soList_("bytes"), 3, "Push a Python bytes object."),
		mkOI("BINBYTES8", "\x8e", adBytes8, soList_(), soList_("bytes"), 4, "Push a Python bytes object."),
		mkOI("BYTEARRAY8", "\x96", adBytearray8, soList_(), soList_("bytearray"), 5, "Push a Python bytearray object."),
		mkOI("NEXT_BUFFER", "\x97", nil, soList_(), soList_("buffer"), 5, "Push an out-of-band buffer object."),
		mkOI("READONLY_BUFFER", "\x98", nil, soList_("buffer"), soList_("buffer"), 5, "Make an existing out-of-band buffer object read-only."),
		mkOI("NONE", "N", nil, soList_(), soList_("None"), 0, "Push None."),
		mkOI("NEWTRUE", "\x88", nil, soList_(), soList_("bool"), 2, "Push True."),
		mkOI("NEWFALSE", "\x89", nil, soList_(), soList_("bool"), 2, "Push False."),
		mkOI("UNICODE", "V", adUnicodestringnl, soList_(), soList_("str"), 0, "Push a Python Unicode string object."),
		mkOI("SHORT_BINUNICODE", "\x8c", adUnicodestring1, soList_(), soList_("str"), 4, "Push a Python Unicode string object."),
		mkOI("BINUNICODE", "X", adUnicodestring4, soList_(), soList_("str"), 1, "Push a Python Unicode string object."),
		mkOI("BINUNICODE8", "\x8d", adUnicodestring8, soList_(), soList_("str"), 4, "Push a Python Unicode string object."),
		mkOI("FLOAT", "F", adFloatnl, soList_(), soList_("float"), 0, "Newline-terminated decimal float literal."),
		mkOI("BINFLOAT", "G", adFloat8, soList_(), soList_("float"), 1, "Float stored in binary."),
		mkOI("EMPTY_LIST", "]", nil, soList_(), soList_("list"), 1, "Push an empty list."),
		mkOI("APPEND", "a", nil, soList_("list", "any"), soList_("list"), 0, "Append an object to a list."),
		mkOI("APPENDS", "e", nil, soList_("list", "mark", "stackslice"), soList_("list"), 1, "Extend a list by a slice of the stack."),
		mkOI("LIST", "l", nil, soList_("mark", "stackslice"), soList_("list"), 0, "Build a list out of the topmost stack slice."),
		mkOI("EMPTY_TUPLE", ")", nil, soList_(), soList_("tuple"), 1, "Push an empty tuple."),
		mkOI("TUPLE", "t", nil, soList_("mark", "stackslice"), soList_("tuple"), 0, "Build a tuple out of the topmost stack slice."),
		mkOI("TUPLE1", "\x85", nil, soList_("any"), soList_("tuple"), 2, "Build a one-tuple out of the topmost item."),
		mkOI("TUPLE2", "\x86", nil, soList_("any", "any"), soList_("tuple"), 2, "Build a two-tuple out of the top two items."),
		mkOI("TUPLE3", "\x87", nil, soList_("any", "any", "any"), soList_("tuple"), 2, "Build a three-tuple out of the top three items."),
		mkOI("EMPTY_DICT", "}", nil, soList_(), soList_("dict"), 1, "Push an empty dict."),
		mkOI("DICT", "d", nil, soList_("mark", "stackslice"), soList_("dict"), 0, "Build a dict out of the topmost stack slice."),
		mkOI("SETITEM", "s", nil, soList_("dict", "any", "any"), soList_("dict"), 0, "Add a key+value pair to an existing dict."),
		mkOI("SETITEMS", "u", nil, soList_("dict", "mark", "stackslice"), soList_("dict"), 1, "Add an arbitrary number of key+value pairs to an existing dict."),
		mkOI("EMPTY_SET", "\x8f", nil, soList_(), soList_("set"), 4, "Push an empty set."),
		mkOI("ADDITEMS", "\x90", nil, soList_("set", "mark", "stackslice"), soList_("set"), 4, "Add an arbitrary number of items to an existing set."),
		mkOI("FROZENSET", "\x91", nil, soList_("mark", "stackslice"), soList_("frozenset"), 4, "Build a frozenset out of the topmost stack slice."),
		mkOI("POP", "0", nil, soList_("any"), soList_(), 0, "Discard the top stack item."),
		mkOI("DUP", "2", nil, soList_("any"), soList_("any", "any"), 0, "Push the top stack item onto the stack again."),
		mkOI("MARK", "(", nil, soList_(), soList_("mark"), 0, "Push markobject onto the stack."),
		mkOI("POP_MARK", "1", nil, soList_("mark", "stackslice"), soList_(), 1, "Pop all the stack objects at and above the topmost markobject."),
		mkOI("GET", "g", adDecimalnlShort, soList_(), soList_("any"), 0, "Read an object from the memo and push it on the stack."),
		mkOI("BINGET", "h", adUint1, soList_(), soList_("any"), 1, "Read an object from the memo and push it on the stack."),
		mkOI("LONG_BINGET", "j", adUint4, soList_(), soList_("any"), 1, "Read an object from the memo and push it on the stack."),
		mkOI("PUT", "p", adDecimalnlShort, soList_(), soList_(), 0, "Store the stack top into the memo."),
		mkOI("BINPUT", "q", adUint1, soList_(), soList_(), 1, "Store the stack top into the memo."),
		mkOI("LONG_BINPUT", "r", adUint4, soList_(), soList_(), 1, "Store the stack top into the memo."),
		mkOI("MEMOIZE", "\x94", nil, soList_("any"), soList_("any"), 4, "Store the top of the stack in memo."),
		mkOI("EXT1", "\x82", adUint1, soList_(), soList_("any"), 2, "Extension code."),
		mkOI("EXT2", "\x83", adUint2, soList_(), soList_("any"), 2, "Extension code."),
		mkOI("EXT4", "\x84", adInt4, soList_(), soList_("any"), 2, "Extension code."),
		mkOI("GLOBAL", "c", adStringnlNoescapePair, soList_(), soList_("any"), 0, "Push a global object (find it in a module)."),
		mkOI("STACK_GLOBAL", "\x93", nil, soList_("str", "str"), soList_("any"), 4, "Push a global object (find it in a module); 2-arg form."),
		mkOI("REDUCE", "R", nil, soList_("any", "any"), soList_("any"), 0, "Push an object built from a callable and an argument tuple."),
		mkOI("BUILD", "b", nil, soList_("any", "any"), soList_("any"), 0, "Call __setstate__ or __dict__.update()."),
		mkOI("INST", "i", adStringnlNoescapePair, soList_("mark", "stackslice"), soList_("any"), 0, "Build a class instance."),
		mkOI("OBJ", "o", nil, soList_("mark", "any", "stackslice"), soList_("any"), 1, "Build a class instance."),
		mkOI("NEWOBJ", "\x81", nil, soList_("any", "any"), soList_("any"), 2, "Build an object by applying cls.__new__ to argtuple."),
		mkOI("NEWOBJ_EX", "\x92", nil, soList_("any", "any", "any"), soList_("any"), 4, "Build an object by applying cls.__new__ to argtuple."),
		mkOI("PROTO", "\x80", adUint1, soList_(), soList_(), 2, "Protocol version indicator."),
		mkOI("STOP", ".", nil, soList_("any"), soList_(), 0, "Stop the unpickling machine."),
		mkOI("FRAME", "\x95", adUint8, soList_(), soList_(), 4, "Indicate the beginning of a new frame."),
		mkOI("PERSID", "P", adStringnlNoescape, soList_(), soList_("any"), 0, "Push an object identified by a persistent ID."),
		mkOI("BINPERSID", "Q", nil, soList_("any"), soList_("any"), 1, "Push an object identified by a persistent ID."),
	}

	d.SetStr("opcodes", &object.List{V: opcodeList})

	// ── decode_long(data) → int ───────────────────────────────────────────────

	d.SetStr("decode_long", &object.BuiltinFunc{
		Name: "decode_long",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return &object.Int{}, nil
			}
			var data []byte
			switch v := a[0].(type) {
			case *object.Bytes:
				data = v.V
			default:
				return &object.Int{}, nil
			}
			if len(data) == 0 {
				return &object.Int{}, nil
			}
			// little-endian two's complement
			n := new(big.Int)
			for idx := len(data) - 1; idx >= 0; idx-- {
				n.Lsh(n, 8)
				n.Or(n, new(big.Int).SetInt64(int64(data[idx])))
			}
			// sign extend: if high bit set, subtract 2^(8*len)
			if data[len(data)-1]&0x80 != 0 {
				shift := new(big.Int).Lsh(big.NewInt(1), uint(8*len(data)))
				n.Sub(n, shift)
			}
			return &object.Int{V: *n}, nil
		},
	})

	// ── genops(pickle) → list of (OpcodeInfo, arg, pos) ──────────────────────
	// Stub: returns empty list (no pickle parser in goipy)

	d.SetStr("genops", &object.BuiltinFunc{
		Name: "genops",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	// ── dis(pickle, out=None, ...) → None ─────────────────────────────────────

	d.SetStr("dis", &object.BuiltinFunc{
		Name: "dis",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── optimize(p) → bytes ───────────────────────────────────────────────────
	// Stub: returns input bytes unchanged

	d.SetStr("optimize", &object.BuiltinFunc{
		Name: "optimize",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) > 0 {
				if b, ok := a[0].(*object.Bytes); ok {
					return b, nil
				}
			}
			return &object.Bytes{V: []byte{}}, nil
		},
	})

	// ── __all__ ───────────────────────────────────────────────────────────────

	d.SetStr("__all__", &object.List{V: []object.Object{
		&object.Str{V: "dis"}, &object.Str{V: "genops"}, &object.Str{V: "optimize"},
	}})

	// ── reader stubs ──────────────────────────────────────────────────────────
	// All read_* functions exist and accept a file-like arg; stubs return sensible zero values.

	readerStubs := []string{
		"read_uint1", "read_uint2", "read_uint4", "read_uint8",
		"read_int4", "read_bytes1", "read_bytes4", "read_bytes8",
		"read_bytearray8", "read_string1", "read_string4", "read_stringnl",
		"read_stringnl_noescape", "read_stringnl_noescape_pair",
		"read_float8", "read_floatnl", "read_decimalnl_short",
		"read_decimalnl_long", "read_long1", "read_long4",
		"read_unicodestring1", "read_unicodestring4", "read_unicodestring8",
		"read_unicodestringnl",
	}
	for _, nm := range readerStubs {
		name := nm // capture
		d.SetStr(name, &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.Bytes{V: []byte{}}, nil
			},
		})
	}

	return m
}

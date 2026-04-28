package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildDis constructs the dis module with CPython 3.14 API.
func (i *Interp) buildDis() *object.Module {
	m := &object.Module{Name: "dis", Dict: object.NewDict()}
	d := m.Dict

	// ── constants ────────────────────────────────────────────────────────────

	d.SetStr("HAVE_ARGUMENT", intObj(43))
	d.SetStr("EXTENDED_ARG", intObj(69))

	d.SetStr("cmp_op", &object.Tuple{V: []object.Object{
		&object.Str{V: "<"},
		&object.Str{V: "<="},
		&object.Str{V: "=="},
		&object.Str{V: "!="},
		&object.Str{V: ">"},
		&object.Str{V: ">="},
	}})

	// ── opmap: opname → opcode (CPython 3.14 exact) ──────────────────────────

	opPairs := [][2]interface{}{
		{"CACHE", 0}, {"BINARY_SLICE", 1}, {"BUILD_TEMPLATE", 2},
		{"CALL_FUNCTION_EX", 4}, {"CHECK_EG_MATCH", 5}, {"CHECK_EXC_MATCH", 6},
		{"CLEANUP_THROW", 7}, {"DELETE_SUBSCR", 8}, {"END_FOR", 9},
		{"END_SEND", 10}, {"EXIT_INIT_CHECK", 11}, {"FORMAT_SIMPLE", 12},
		{"FORMAT_WITH_SPEC", 13}, {"GET_AITER", 14}, {"GET_ANEXT", 15},
		{"GET_ITER", 16}, {"RESERVED", 17}, {"GET_LEN", 18},
		{"GET_YIELD_FROM_ITER", 19}, {"INTERPRETER_EXIT", 20},
		{"LOAD_BUILD_CLASS", 21}, {"LOAD_LOCALS", 22}, {"MAKE_FUNCTION", 23},
		{"MATCH_KEYS", 24}, {"MATCH_MAPPING", 25}, {"MATCH_SEQUENCE", 26},
		{"NOP", 27}, {"NOT_TAKEN", 28}, {"POP_EXCEPT", 29},
		{"POP_ITER", 30}, {"POP_TOP", 31}, {"PUSH_EXC_INFO", 32},
		{"PUSH_NULL", 33}, {"RETURN_GENERATOR", 34}, {"RETURN_VALUE", 35},
		{"SETUP_ANNOTATIONS", 36}, {"STORE_SLICE", 37}, {"STORE_SUBSCR", 38},
		{"TO_BOOL", 39}, {"UNARY_INVERT", 40}, {"UNARY_NEGATIVE", 41},
		{"UNARY_NOT", 42}, {"WITH_EXCEPT_START", 43}, {"BINARY_OP", 44},
		{"BUILD_INTERPOLATION", 45}, {"BUILD_LIST", 46}, {"BUILD_MAP", 47},
		{"BUILD_SET", 48}, {"BUILD_SLICE", 49}, {"BUILD_STRING", 50},
		{"BUILD_TUPLE", 51}, {"CALL", 52}, {"CALL_INTRINSIC_1", 53},
		{"CALL_INTRINSIC_2", 54}, {"CALL_KW", 55}, {"COMPARE_OP", 56},
		{"CONTAINS_OP", 57}, {"CONVERT_VALUE", 58}, {"COPY", 59},
		{"COPY_FREE_VARS", 60}, {"DELETE_ATTR", 61}, {"DELETE_DEREF", 62},
		{"DELETE_FAST", 63}, {"DELETE_GLOBAL", 64}, {"DELETE_NAME", 65},
		{"DICT_MERGE", 66}, {"DICT_UPDATE", 67}, {"END_ASYNC_FOR", 68},
		{"EXTENDED_ARG", 69}, {"FOR_ITER", 70}, {"GET_AWAITABLE", 71},
		{"IMPORT_FROM", 72}, {"IMPORT_NAME", 73}, {"IS_OP", 74},
		{"JUMP_BACKWARD", 75}, {"JUMP_BACKWARD_NO_INTERRUPT", 76},
		{"JUMP_FORWARD", 77}, {"LIST_APPEND", 78}, {"LIST_EXTEND", 79},
		{"LOAD_ATTR", 80}, {"LOAD_COMMON_CONSTANT", 81}, {"LOAD_CONST", 82},
		{"LOAD_DEREF", 83}, {"LOAD_FAST", 84}, {"LOAD_FAST_AND_CLEAR", 85},
		{"LOAD_FAST_BORROW", 86}, {"LOAD_FAST_BORROW_LOAD_FAST_BORROW", 87},
		{"LOAD_FAST_CHECK", 88}, {"LOAD_FAST_LOAD_FAST", 89},
		{"LOAD_FROM_DICT_OR_DEREF", 90}, {"LOAD_FROM_DICT_OR_GLOBALS", 91},
		{"LOAD_GLOBAL", 92}, {"LOAD_NAME", 93}, {"LOAD_SMALL_INT", 94},
		{"LOAD_SPECIAL", 95}, {"LOAD_SUPER_ATTR", 96}, {"MAKE_CELL", 97},
		{"MAP_ADD", 98}, {"MATCH_CLASS", 99}, {"POP_JUMP_IF_FALSE", 100},
		{"POP_JUMP_IF_NONE", 101}, {"POP_JUMP_IF_NOT_NONE", 102},
		{"POP_JUMP_IF_TRUE", 103}, {"RAISE_VARARGS", 104}, {"RERAISE", 105},
		{"SEND", 106}, {"SET_ADD", 107}, {"SET_FUNCTION_ATTRIBUTE", 108},
		{"SET_UPDATE", 109}, {"STORE_ATTR", 110}, {"STORE_DEREF", 111},
		{"STORE_FAST", 112}, {"STORE_FAST_LOAD_FAST", 113},
		{"STORE_FAST_STORE_FAST", 114}, {"STORE_GLOBAL", 115},
		{"STORE_NAME", 116}, {"SWAP", 117}, {"UNPACK_EX", 118},
		{"UNPACK_SEQUENCE", 119}, {"YIELD_VALUE", 120},
		{"RESUME", 128},
		{"INSTRUMENTED_END_FOR", 234}, {"INSTRUMENTED_POP_ITER", 235},
		{"INSTRUMENTED_END_SEND", 236}, {"INSTRUMENTED_FOR_ITER", 237},
		{"INSTRUMENTED_INSTRUCTION", 238}, {"INSTRUMENTED_JUMP_FORWARD", 239},
		{"INSTRUMENTED_NOT_TAKEN", 240}, {"INSTRUMENTED_POP_JUMP_IF_TRUE", 241},
		{"INSTRUMENTED_POP_JUMP_IF_FALSE", 242},
		{"INSTRUMENTED_POP_JUMP_IF_NONE", 243},
		{"INSTRUMENTED_POP_JUMP_IF_NOT_NONE", 244},
		{"INSTRUMENTED_RESUME", 245}, {"INSTRUMENTED_RETURN_VALUE", 246},
		{"INSTRUMENTED_YIELD_VALUE", 247}, {"INSTRUMENTED_END_ASYNC_FOR", 248},
		{"INSTRUMENTED_LOAD_SUPER_ATTR", 249}, {"INSTRUMENTED_CALL", 250},
		{"INSTRUMENTED_CALL_KW", 251}, {"INSTRUMENTED_CALL_FUNCTION_EX", 252},
		{"INSTRUMENTED_JUMP_BACKWARD", 253}, {"INSTRUMENTED_LINE", 254},
		{"ENTER_EXECUTOR", 255}, {"ANNOTATIONS_PLACEHOLDER", 256},
		{"JUMP", 257}, {"JUMP_IF_FALSE", 258}, {"JUMP_IF_TRUE", 259},
		{"JUMP_NO_INTERRUPT", 260}, {"LOAD_CLOSURE", 261}, {"POP_BLOCK", 262},
		{"SETUP_CLEANUP", 263}, {"SETUP_FINALLY", 264}, {"SETUP_WITH", 265},
		{"STORE_FAST_MAYBE_NULL", 266},
	}

	opmapDict := object.NewDict()
	for _, p := range opPairs {
		opmapDict.SetStr(p[0].(string), intObj(int64(p[1].(int))))
	}
	d.SetStr("opmap", opmapDict)

	// ── opname: list indexed by opcode number (267 elements) ─────────────────

	opnameSlice := make([]object.Object, 267)
	for idx := range opnameSlice {
		opnameSlice[idx] = &object.Str{V: "<" + itoa(idx) + ">"}
	}
	for _, p := range opPairs {
		opnameSlice[p[1].(int)] = &object.Str{V: p[0].(string)}
	}
	d.SetStr("opname", &object.List{V: opnameSlice})

	// ── opcode category lists ─────────────────────────────────────────────────

	makeIntList := func(nums []int) *object.List {
		objs := make([]object.Object, len(nums))
		for k, n := range nums {
			objs[k] = intObj(int64(n))
		}
		return &object.List{V: objs}
	}

	d.SetStr("hasarg", makeIntList([]int{
		44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60,
		61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77,
		78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94,
		95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109,
		110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 128, 237, 239,
		241, 242, 243, 244, 245, 247, 248, 249, 250, 251, 253, 255, 257, 258,
		259, 260, 261, 263, 264, 265, 266,
	}))

	d.SetStr("hasconst", makeIntList([]int{82}))
	d.SetStr("hasname", makeIntList([]int{61, 64, 65, 72, 73, 80, 91, 92, 93, 96, 110, 115, 116, 249}))
	d.SetStr("hasjump", makeIntList([]int{68, 70, 75, 76, 77, 100, 101, 102, 103, 106, 237, 248, 257, 258, 259, 260}))
	d.SetStr("hasjrel", makeIntList([]int{68, 70, 75, 76, 77, 100, 101, 102, 103, 106, 237, 248, 257, 258, 259, 260}))
	d.SetStr("hasjabs", makeIntList([]int{}))
	d.SetStr("hasfree", makeIntList([]int{62, 90, 97, 111}))
	d.SetStr("haslocal", makeIntList([]int{63, 83, 84, 85, 86, 87, 88, 89, 112, 113, 114, 261, 266}))
	d.SetStr("hasexc", makeIntList([]int{263, 264, 265}))
	d.SetStr("hascompare", makeIntList([]int{56}))

	// ── COMPILER_FLAG_NAMES ───────────────────────────────────────────────────

	cfnDict := object.NewDict()
	cfnPairs := [][2]interface{}{
		{1, "OPTIMIZED"}, {2, "NEWLOCALS"}, {4, "VARARGS"}, {8, "VARKEYWORDS"},
		{16, "NESTED"}, {32, "GENERATOR"}, {64, "NOFREE"}, {128, "COROUTINE"},
		{256, "ITERABLE_COROUTINE"}, {512, "ASYNC_GENERATOR"},
		{67108864, "HAS_DOCSTRING"}, {134217728, "METHOD"},
	}
	for _, p := range cfnPairs {
		cfnDict.Set(intObj(int64(p[0].(int))), &object.Str{V: p[1].(string)}) //nolint
	}
	d.SetStr("COMPILER_FLAG_NAMES", cfnDict)

	// ── Positions namedtuple-like class ───────────────────────────────────────

	posFields := []object.Object{
		&object.Str{V: "lineno"}, &object.Str{V: "end_lineno"},
		&object.Str{V: "col_offset"}, &object.Str{V: "end_col_offset"},
	}
	posCls := &object.Class{
		Name:  "Positions",
		Bases: []*object.Class{},
		Dict:  object.NewDict(),
	}
	posCls.Dict.SetStr("_fields", &object.Tuple{V: posFields})
	posCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			if inst, ok := a[0].(*object.Instance); ok {
				names := []string{"lineno", "end_lineno", "col_offset", "end_col_offset"}
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
	posCls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "Positions()"}, nil
			}
			if inst, ok := a[0].(*object.Instance); ok {
				ln, _ := inst.Dict.GetStr("lineno")
				eln, _ := inst.Dict.GetStr("end_lineno")
				co, _ := inst.Dict.GetStr("col_offset")
				eco, _ := inst.Dict.GetStr("end_col_offset")
				s := "Positions(lineno=" + object.Repr(ln) +
					", end_lineno=" + object.Repr(eln) +
					", col_offset=" + object.Repr(co) +
					", end_col_offset=" + object.Repr(eco) + ")"
				return &object.Str{V: s}, nil
			}
			return &object.Str{V: "Positions()"}, nil
		},
	})
	d.SetStr("Positions", posCls)

	// ── Instruction namedtuple-like class ─────────────────────────────────────

	instrFieldNames := []string{
		"opname", "opcode", "arg", "argval", "argrepr",
		"offset", "start_offset", "starts_line", "line_number",
		"label", "positions", "cache_info",
	}
	instrFieldObjs := make([]object.Object, len(instrFieldNames))
	for k, nm := range instrFieldNames {
		instrFieldObjs[k] = &object.Str{V: nm}
	}
	instrCls := &object.Class{
		Name:  "Instruction",
		Bases: []*object.Class{},
		Dict:  object.NewDict(),
	}
	instrCls.Dict.SetStr("_fields", &object.Tuple{V: instrFieldObjs})
	instrCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			for idx, nm := range instrFieldNames {
				if idx+1 < len(a) {
					inst.Dict.SetStr(nm, a[idx+1])
				} else if kw != nil {
					if v, ok2 := kw.GetStr(nm); ok2 {
						inst.Dict.SetStr(nm, v)
						continue
					}
					inst.Dict.SetStr(nm, object.None)
				} else {
					inst.Dict.SetStr(nm, object.None)
				}
			}
			return object.None, nil
		},
	})
	instrCls.Dict.SetStr("_replace", &object.BuiltinFunc{
		Name: "_replace",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			newInst := &object.Instance{Class: instrCls, Dict: object.NewDict()}
			for _, nm := range instrFieldNames {
				if v, ok2 := inst.Dict.GetStr(nm); ok2 {
					newInst.Dict.SetStr(nm, v)
				}
			}
			if kw != nil {
				ks, vs := kw.Items()
				for idx, k := range ks {
					if ks2, ok2 := k.(*object.Str); ok2 {
						newInst.Dict.SetStr(ks2.V, vs[idx])
					}
				}
			}
			return newInst, nil
		},
	})
	d.SetStr("Instruction", instrCls)

	// ── Bytecode class ────────────────────────────────────────────────────────

	bytecodeCls := &object.Class{
		Name:  "Bytecode",
		Bases: []*object.Class{},
		Dict:  object.NewDict(),
	}
	bytecodeCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			var codeobj object.Object = object.None
			if len(a) > 1 {
				codeobj = a[1]
			}
			inst.Dict.SetStr("codeobj", codeobj)
			inst.Dict.SetStr("first_line", object.None)
			return object.None, nil
		},
	})
	bytecodeCls.Dict.SetStr("__iter__", &object.BuiltinFunc{
		Name: "__iter__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})
	bytecodeCls.Dict.SetStr("dis", &object.BuiltinFunc{
		Name: "dis",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	d.SetStr("Bytecode", bytecodeCls)

	// ── stub functions ────────────────────────────────────────────────────────

	d.SetStr("code_info", &object.BuiltinFunc{
		Name: "code_info",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: ""}, nil
		},
	})

	d.SetStr("show_code", &object.BuiltinFunc{
		Name: "show_code",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	d.SetStr("dis", &object.BuiltinFunc{
		Name: "dis",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	d.SetStr("disassemble", &object.BuiltinFunc{
		Name: "disassemble",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	d.SetStr("distb", &object.BuiltinFunc{
		Name: "distb",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	if v, ok := d.GetStr("disassemble"); ok {
		d.SetStr("disco", v)
	}

	d.SetStr("get_instructions", &object.BuiltinFunc{
		Name: "get_instructions",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	d.SetStr("findlinestarts", &object.BuiltinFunc{
		Name: "findlinestarts",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	d.SetStr("findlabels", &object.BuiltinFunc{
		Name: "findlabels",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	d.SetStr("stack_effect", &object.BuiltinFunc{
		Name: "stack_effect",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj(0), nil
		},
	})

	// ── __all__ ───────────────────────────────────────────────────────────────

	d.SetStr("__all__", &object.List{V: []object.Object{
		&object.Str{V: "code_info"}, &object.Str{V: "dis"},
		&object.Str{V: "disassemble"}, &object.Str{V: "distb"},
		&object.Str{V: "disco"}, &object.Str{V: "findlinestarts"},
		&object.Str{V: "findlabels"}, &object.Str{V: "show_code"},
		&object.Str{V: "get_instructions"}, &object.Str{V: "Instruction"},
		&object.Str{V: "Bytecode"}, &object.Str{V: "cmp_op"},
		&object.Str{V: "stack_effect"}, &object.Str{V: "hascompare"},
		&object.Str{V: "opname"}, &object.Str{V: "opmap"},
		&object.Str{V: "HAVE_ARGUMENT"}, &object.Str{V: "EXTENDED_ARG"},
		&object.Str{V: "hasarg"}, &object.Str{V: "hasconst"},
		&object.Str{V: "hasname"}, &object.Str{V: "hasjump"},
		&object.Str{V: "hasjrel"}, &object.Str{V: "hasjabs"},
		&object.Str{V: "hasfree"}, &object.Str{V: "haslocal"},
		&object.Str{V: "hasexc"},
	}})

	return m
}

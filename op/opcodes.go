// Code generated from python3.14 dis.opname / opcode. DO NOT EDIT.
// Regenerate with: go generate ./op
//go:generate sh -c "python3.14 ../internal/gen/gen_opcodes.py > opcodes.go"

package op

// Python 3.14 opcode numbers. Values > 255 are compiler-only pseudo-opcodes
// and are not included here because they cannot appear in a real bytecode
// stream.
const (
	CACHE                             = 0
	BINARY_SLICE                      = 1
	BUILD_TEMPLATE                    = 2
	BINARY_OP_INPLACE_ADD_UNICODE     = 3
	CALL_FUNCTION_EX                  = 4
	CHECK_EG_MATCH                    = 5
	CHECK_EXC_MATCH                   = 6
	CLEANUP_THROW                     = 7
	DELETE_SUBSCR                     = 8
	END_FOR                           = 9
	END_SEND                          = 10
	EXIT_INIT_CHECK                   = 11
	FORMAT_SIMPLE                     = 12
	FORMAT_WITH_SPEC                  = 13
	GET_AITER                         = 14
	GET_ANEXT                         = 15
	GET_ITER                          = 16
	RESERVED                          = 17
	GET_LEN                           = 18
	GET_YIELD_FROM_ITER               = 19
	INTERPRETER_EXIT                  = 20
	LOAD_BUILD_CLASS                  = 21
	LOAD_LOCALS                       = 22
	MAKE_FUNCTION                     = 23
	MATCH_KEYS                        = 24
	MATCH_MAPPING                     = 25
	MATCH_SEQUENCE                    = 26
	NOP                               = 27
	NOT_TAKEN                         = 28
	POP_EXCEPT                        = 29
	POP_ITER                          = 30
	POP_TOP                           = 31
	PUSH_EXC_INFO                     = 32
	PUSH_NULL                         = 33
	RETURN_GENERATOR                  = 34
	RETURN_VALUE                      = 35
	SETUP_ANNOTATIONS                 = 36
	STORE_SLICE                       = 37
	STORE_SUBSCR                      = 38
	TO_BOOL                           = 39
	UNARY_INVERT                      = 40
	UNARY_NEGATIVE                    = 41
	UNARY_NOT                         = 42
	WITH_EXCEPT_START                 = 43
	BINARY_OP                         = 44
	BUILD_INTERPOLATION               = 45
	BUILD_LIST                        = 46
	BUILD_MAP                         = 47
	BUILD_SET                         = 48
	BUILD_SLICE                       = 49
	BUILD_STRING                      = 50
	BUILD_TUPLE                       = 51
	CALL                              = 52
	CALL_INTRINSIC_1                  = 53
	CALL_INTRINSIC_2                  = 54
	CALL_KW                           = 55
	COMPARE_OP                        = 56
	CONTAINS_OP                       = 57
	CONVERT_VALUE                     = 58
	COPY                              = 59
	COPY_FREE_VARS                    = 60
	DELETE_ATTR                       = 61
	DELETE_DEREF                      = 62
	DELETE_FAST                       = 63
	DELETE_GLOBAL                     = 64
	DELETE_NAME                       = 65
	DICT_MERGE                        = 66
	DICT_UPDATE                       = 67
	END_ASYNC_FOR                     = 68
	EXTENDED_ARG                      = 69
	FOR_ITER                          = 70
	GET_AWAITABLE                     = 71
	IMPORT_FROM                       = 72
	IMPORT_NAME                       = 73
	IS_OP                             = 74
	JUMP_BACKWARD                     = 75
	JUMP_BACKWARD_NO_INTERRUPT        = 76
	JUMP_FORWARD                      = 77
	LIST_APPEND                       = 78
	LIST_EXTEND                       = 79
	LOAD_ATTR                         = 80
	LOAD_COMMON_CONSTANT              = 81
	LOAD_CONST                        = 82
	LOAD_DEREF                        = 83
	LOAD_FAST                         = 84
	LOAD_FAST_AND_CLEAR               = 85
	LOAD_FAST_BORROW                  = 86
	LOAD_FAST_BORROW_LOAD_FAST_BORROW = 87
	LOAD_FAST_CHECK                   = 88
	LOAD_FAST_LOAD_FAST               = 89
	LOAD_FROM_DICT_OR_DEREF           = 90
	LOAD_FROM_DICT_OR_GLOBALS         = 91
	LOAD_GLOBAL                       = 92
	LOAD_NAME                         = 93
	LOAD_SMALL_INT                    = 94
	LOAD_SPECIAL                      = 95
	LOAD_SUPER_ATTR                   = 96
	MAKE_CELL                         = 97
	MAP_ADD                           = 98
	MATCH_CLASS                       = 99
	POP_JUMP_IF_FALSE                 = 100
	POP_JUMP_IF_NONE                  = 101
	POP_JUMP_IF_NOT_NONE              = 102
	POP_JUMP_IF_TRUE                  = 103
	RAISE_VARARGS                     = 104
	RERAISE                           = 105
	SEND                              = 106
	SET_ADD                           = 107
	SET_FUNCTION_ATTRIBUTE            = 108
	SET_UPDATE                        = 109
	STORE_ATTR                        = 110
	STORE_DEREF                       = 111
	STORE_FAST                        = 112
	STORE_FAST_LOAD_FAST              = 113
	STORE_FAST_STORE_FAST             = 114
	STORE_GLOBAL                      = 115
	STORE_NAME                        = 116
	SWAP                              = 117
	UNPACK_EX                         = 118
	UNPACK_SEQUENCE                   = 119
	YIELD_VALUE                       = 120
	RESUME                            = 128
)

// Specialized / instrumented opcodes we map onto their generic form.
const (
	BINARY_OP_ADD_FLOAT                       = 129
	BINARY_OP_ADD_INT                         = 130
	BINARY_OP_ADD_UNICODE                     = 131
	BINARY_OP_EXTEND                          = 132
	BINARY_OP_MULTIPLY_FLOAT                  = 133
	BINARY_OP_MULTIPLY_INT                    = 134
	BINARY_OP_SUBSCR_DICT                     = 135
	BINARY_OP_SUBSCR_GETITEM                  = 136
	BINARY_OP_SUBSCR_LIST_INT                 = 137
	BINARY_OP_SUBSCR_LIST_SLICE               = 138
	BINARY_OP_SUBSCR_STR_INT                  = 139
	BINARY_OP_SUBSCR_TUPLE_INT                = 140
	BINARY_OP_SUBTRACT_FLOAT                  = 141
	BINARY_OP_SUBTRACT_INT                    = 142
	CALL_ALLOC_AND_ENTER_INIT                 = 143
	CALL_BOUND_METHOD_EXACT_ARGS              = 144
	CALL_BOUND_METHOD_GENERAL                 = 145
	CALL_BUILTIN_CLASS                        = 146
	CALL_BUILTIN_FAST                         = 147
	CALL_BUILTIN_FAST_WITH_KEYWORDS           = 148
	CALL_BUILTIN_O                            = 149
	CALL_ISINSTANCE                           = 150
	CALL_KW_BOUND_METHOD                      = 151
	CALL_KW_NON_PY                            = 152
	CALL_KW_PY                                = 153
	CALL_LEN                                  = 154
	CALL_LIST_APPEND                          = 155
	CALL_METHOD_DESCRIPTOR_FAST               = 156
	CALL_METHOD_DESCRIPTOR_FAST_WITH_KEYWORDS = 157
	CALL_METHOD_DESCRIPTOR_NOARGS             = 158
	CALL_METHOD_DESCRIPTOR_O                  = 159
	CALL_NON_PY_GENERAL                       = 160
	CALL_PY_EXACT_ARGS                        = 161
	CALL_PY_GENERAL                           = 162
	CALL_STR_1                                = 163
	CALL_TUPLE_1                              = 164
	CALL_TYPE_1                               = 165
	COMPARE_OP_FLOAT                          = 166
	COMPARE_OP_INT                            = 167
	COMPARE_OP_STR                            = 168
	CONTAINS_OP_DICT                          = 169
	CONTAINS_OP_SET                           = 170
	FOR_ITER_GEN                              = 171
	FOR_ITER_LIST                             = 172
	FOR_ITER_RANGE                            = 173
	FOR_ITER_TUPLE                            = 174
	JUMP_BACKWARD_JIT                         = 175
	JUMP_BACKWARD_NO_JIT                      = 176
	LOAD_ATTR_CLASS                           = 177
	LOAD_ATTR_CLASS_WITH_METACLASS_CHECK      = 178
	LOAD_ATTR_GETATTRIBUTE_OVERRIDDEN         = 179
	LOAD_ATTR_INSTANCE_VALUE                  = 180
	LOAD_ATTR_METHOD_LAZY_DICT                = 181
	LOAD_ATTR_METHOD_NO_DICT                  = 182
	LOAD_ATTR_METHOD_WITH_VALUES              = 183
	LOAD_ATTR_MODULE                          = 184
	LOAD_ATTR_NONDESCRIPTOR_NO_DICT           = 185
	LOAD_ATTR_NONDESCRIPTOR_WITH_VALUES       = 186
	LOAD_ATTR_PROPERTY                        = 187
	LOAD_ATTR_SLOT                            = 188
	LOAD_ATTR_WITH_HINT                       = 189
	LOAD_CONST_IMMORTAL                       = 190
	LOAD_CONST_MORTAL                         = 191
	LOAD_GLOBAL_BUILTIN                       = 192
	LOAD_GLOBAL_MODULE                        = 193
	LOAD_SUPER_ATTR_ATTR                      = 194
	LOAD_SUPER_ATTR_METHOD                    = 195
	RESUME_CHECK                              = 196
	SEND_GEN                                  = 197
	STORE_ATTR_INSTANCE_VALUE                 = 198
	STORE_ATTR_SLOT                           = 199
	STORE_ATTR_WITH_HINT                      = 200
	STORE_SUBSCR_DICT                         = 201
	STORE_SUBSCR_LIST_INT                     = 202
	TO_BOOL_ALWAYS_TRUE                       = 203
	TO_BOOL_BOOL                              = 204
	TO_BOOL_INT                               = 205
	TO_BOOL_LIST                              = 206
	TO_BOOL_NONE                              = 207
	TO_BOOL_STR                               = 208
	UNPACK_SEQUENCE_LIST                      = 209
	UNPACK_SEQUENCE_TUPLE                     = 210
	UNPACK_SEQUENCE_TWO_TUPLE                 = 211
	INSTRUMENTED_END_FOR                      = 234
	INSTRUMENTED_POP_ITER                     = 235
	INSTRUMENTED_END_SEND                     = 236
	INSTRUMENTED_FOR_ITER                     = 237
	INSTRUMENTED_INSTRUCTION                  = 238
	INSTRUMENTED_JUMP_FORWARD                 = 239
	INSTRUMENTED_NOT_TAKEN                    = 240
	INSTRUMENTED_POP_JUMP_IF_TRUE             = 241
	INSTRUMENTED_POP_JUMP_IF_FALSE            = 242
	INSTRUMENTED_POP_JUMP_IF_NONE             = 243
	INSTRUMENTED_POP_JUMP_IF_NOT_NONE         = 244
	INSTRUMENTED_RESUME                       = 245
	INSTRUMENTED_RETURN_VALUE                 = 246
	INSTRUMENTED_YIELD_VALUE                  = 247
	INSTRUMENTED_END_ASYNC_FOR                = 248
	INSTRUMENTED_LOAD_SUPER_ATTR              = 249
	INSTRUMENTED_CALL                         = 250
	INSTRUMENTED_CALL_KW                      = 251
	INSTRUMENTED_CALL_FUNCTION_EX             = 252
	INSTRUMENTED_JUMP_BACKWARD                = 253
	INSTRUMENTED_LINE                         = 254
	ENTER_EXECUTOR                            = 255
)

// Cache holds the number of CACHE entries (each 2 bytes) that follow an
// instruction. Opcodes not listed have zero cache entries.
var Cache = [256]uint8{
	LOAD_GLOBAL:          4,
	BINARY_OP:            5,
	UNPACK_SEQUENCE:      1,
	COMPARE_OP:           1,
	CONTAINS_OP:          1,
	FOR_ITER:             1,
	LOAD_SUPER_ATTR:      1,
	LOAD_ATTR:            9,
	STORE_ATTR:           4,
	CALL:                 3,
	CALL_KW:              3,
	STORE_SUBSCR:         1,
	SEND:                 1,
	JUMP_BACKWARD:        1,
	TO_BOOL:              3,
	POP_JUMP_IF_TRUE:     1,
	POP_JUMP_IF_FALSE:    1,
	POP_JUMP_IF_NONE:     1,
	POP_JUMP_IF_NOT_NONE: 1,
}

// Name returns the mnemonic of an opcode (or "OP<n>" if unknown).
func Name(op uint8) string {
	if n, ok := names[op]; ok {
		return n
	}
	return "OP?"
}

var names = map[uint8]string{
	CACHE: "CACHE", BINARY_SLICE: "BINARY_SLICE", BUILD_TEMPLATE: "BUILD_TEMPLATE",
	BINARY_OP_INPLACE_ADD_UNICODE: "BINARY_OP_INPLACE_ADD_UNICODE",
	CALL_FUNCTION_EX:              "CALL_FUNCTION_EX", CHECK_EG_MATCH: "CHECK_EG_MATCH",
	CHECK_EXC_MATCH: "CHECK_EXC_MATCH", CLEANUP_THROW: "CLEANUP_THROW",
	DELETE_SUBSCR: "DELETE_SUBSCR", END_FOR: "END_FOR", END_SEND: "END_SEND",
	EXIT_INIT_CHECK: "EXIT_INIT_CHECK", FORMAT_SIMPLE: "FORMAT_SIMPLE",
	FORMAT_WITH_SPEC: "FORMAT_WITH_SPEC", GET_AITER: "GET_AITER",
	GET_ANEXT: "GET_ANEXT", GET_ITER: "GET_ITER", RESERVED: "RESERVED",
	GET_LEN: "GET_LEN", GET_YIELD_FROM_ITER: "GET_YIELD_FROM_ITER",
	INTERPRETER_EXIT: "INTERPRETER_EXIT", LOAD_BUILD_CLASS: "LOAD_BUILD_CLASS",
	LOAD_LOCALS: "LOAD_LOCALS", MAKE_FUNCTION: "MAKE_FUNCTION",
	MATCH_KEYS: "MATCH_KEYS", MATCH_MAPPING: "MATCH_MAPPING",
	MATCH_SEQUENCE: "MATCH_SEQUENCE", NOP: "NOP", NOT_TAKEN: "NOT_TAKEN",
	POP_EXCEPT: "POP_EXCEPT", POP_ITER: "POP_ITER", POP_TOP: "POP_TOP",
	PUSH_EXC_INFO: "PUSH_EXC_INFO", PUSH_NULL: "PUSH_NULL",
	RETURN_GENERATOR: "RETURN_GENERATOR", RETURN_VALUE: "RETURN_VALUE",
	SETUP_ANNOTATIONS: "SETUP_ANNOTATIONS", STORE_SLICE: "STORE_SLICE",
	STORE_SUBSCR: "STORE_SUBSCR", TO_BOOL: "TO_BOOL", UNARY_INVERT: "UNARY_INVERT",
	UNARY_NEGATIVE: "UNARY_NEGATIVE", UNARY_NOT: "UNARY_NOT",
	WITH_EXCEPT_START: "WITH_EXCEPT_START", BINARY_OP: "BINARY_OP",
	BUILD_INTERPOLATION: "BUILD_INTERPOLATION", BUILD_LIST: "BUILD_LIST",
	BUILD_MAP: "BUILD_MAP", BUILD_SET: "BUILD_SET", BUILD_SLICE: "BUILD_SLICE",
	BUILD_STRING: "BUILD_STRING", BUILD_TUPLE: "BUILD_TUPLE", CALL: "CALL",
	CALL_INTRINSIC_1: "CALL_INTRINSIC_1", CALL_INTRINSIC_2: "CALL_INTRINSIC_2",
	CALL_KW: "CALL_KW", COMPARE_OP: "COMPARE_OP", CONTAINS_OP: "CONTAINS_OP",
	CONVERT_VALUE: "CONVERT_VALUE", COPY: "COPY", COPY_FREE_VARS: "COPY_FREE_VARS",
	DELETE_ATTR: "DELETE_ATTR", DELETE_DEREF: "DELETE_DEREF",
	DELETE_FAST: "DELETE_FAST", DELETE_GLOBAL: "DELETE_GLOBAL",
	DELETE_NAME: "DELETE_NAME", DICT_MERGE: "DICT_MERGE", DICT_UPDATE: "DICT_UPDATE",
	END_ASYNC_FOR: "END_ASYNC_FOR", EXTENDED_ARG: "EXTENDED_ARG", FOR_ITER: "FOR_ITER",
	GET_AWAITABLE: "GET_AWAITABLE", IMPORT_FROM: "IMPORT_FROM", IMPORT_NAME: "IMPORT_NAME",
	IS_OP: "IS_OP", JUMP_BACKWARD: "JUMP_BACKWARD",
	JUMP_BACKWARD_NO_INTERRUPT: "JUMP_BACKWARD_NO_INTERRUPT", JUMP_FORWARD: "JUMP_FORWARD",
	LIST_APPEND: "LIST_APPEND", LIST_EXTEND: "LIST_EXTEND", LOAD_ATTR: "LOAD_ATTR",
	LOAD_COMMON_CONSTANT: "LOAD_COMMON_CONSTANT", LOAD_CONST: "LOAD_CONST",
	LOAD_DEREF: "LOAD_DEREF", LOAD_FAST: "LOAD_FAST",
	LOAD_FAST_AND_CLEAR: "LOAD_FAST_AND_CLEAR", LOAD_FAST_BORROW: "LOAD_FAST_BORROW",
	LOAD_FAST_BORROW_LOAD_FAST_BORROW: "LOAD_FAST_BORROW_LOAD_FAST_BORROW",
	LOAD_FAST_CHECK:                   "LOAD_FAST_CHECK", LOAD_FAST_LOAD_FAST: "LOAD_FAST_LOAD_FAST",
	LOAD_FROM_DICT_OR_DEREF: "LOAD_FROM_DICT_OR_DEREF",
	LOAD_FROM_DICT_OR_GLOBALS: "LOAD_FROM_DICT_OR_GLOBALS",
	LOAD_GLOBAL:               "LOAD_GLOBAL", LOAD_NAME: "LOAD_NAME",
	LOAD_SMALL_INT:  "LOAD_SMALL_INT", LOAD_SPECIAL: "LOAD_SPECIAL",
	LOAD_SUPER_ATTR: "LOAD_SUPER_ATTR", MAKE_CELL: "MAKE_CELL",
	MAP_ADD:         "MAP_ADD", MATCH_CLASS: "MATCH_CLASS",
	POP_JUMP_IF_FALSE: "POP_JUMP_IF_FALSE", POP_JUMP_IF_NONE: "POP_JUMP_IF_NONE",
	POP_JUMP_IF_NOT_NONE: "POP_JUMP_IF_NOT_NONE", POP_JUMP_IF_TRUE: "POP_JUMP_IF_TRUE",
	RAISE_VARARGS: "RAISE_VARARGS", RERAISE: "RERAISE", SEND: "SEND",
	SET_ADD: "SET_ADD", SET_FUNCTION_ATTRIBUTE: "SET_FUNCTION_ATTRIBUTE",
	SET_UPDATE: "SET_UPDATE", STORE_ATTR: "STORE_ATTR", STORE_DEREF: "STORE_DEREF",
	STORE_FAST: "STORE_FAST", STORE_FAST_LOAD_FAST: "STORE_FAST_LOAD_FAST",
	STORE_FAST_STORE_FAST: "STORE_FAST_STORE_FAST", STORE_GLOBAL: "STORE_GLOBAL",
	STORE_NAME: "STORE_NAME", SWAP: "SWAP", UNPACK_EX: "UNPACK_EX",
	UNPACK_SEQUENCE: "UNPACK_SEQUENCE", YIELD_VALUE: "YIELD_VALUE",
	RESUME: "RESUME", LOAD_CONST_IMMORTAL: "LOAD_CONST_IMMORTAL",
	LOAD_CONST_MORTAL: "LOAD_CONST_MORTAL", RESUME_CHECK: "RESUME_CHECK",
	JUMP_BACKWARD_JIT: "JUMP_BACKWARD_JIT", JUMP_BACKWARD_NO_JIT: "JUMP_BACKWARD_NO_JIT",
	ENTER_EXECUTOR: "ENTER_EXECUTOR", INSTRUMENTED_LINE: "INSTRUMENTED_LINE",
}

// BINARY_OP oparg codes (Python 3.14 opcode._nb_ops indices).
const (
	NB_ADD                    = 0
	NB_AND                    = 1
	NB_FLOOR_DIVIDE           = 2
	NB_LSHIFT                 = 3
	NB_MATRIX_MULTIPLY        = 4
	NB_MULTIPLY               = 5
	NB_REMAINDER              = 6
	NB_OR                     = 7
	NB_POWER                  = 8
	NB_RSHIFT                 = 9
	NB_SUBTRACT               = 10
	NB_TRUE_DIVIDE            = 11
	NB_XOR                    = 12
	NB_INPLACE_ADD            = 13
	NB_INPLACE_AND            = 14
	NB_INPLACE_FLOOR_DIVIDE   = 15
	NB_INPLACE_LSHIFT         = 16
	NB_INPLACE_MATRIX_MULTIPLY = 17
	NB_INPLACE_MULTIPLY       = 18
	NB_INPLACE_REMAINDER      = 19
	NB_INPLACE_OR             = 20
	NB_INPLACE_POWER          = 21
	NB_INPLACE_RSHIFT         = 22
	NB_INPLACE_SUBTRACT       = 23
	NB_INPLACE_TRUE_DIVIDE    = 24
	NB_INPLACE_XOR            = 25
	NB_SUBSCR                 = 26
)

// COMPARE_OP arg encoding (3.14): high bits encode the comparison type, low 4
// bits (mask 0xF, then top 5 give cmp index from 0..5; layout is oparg>>5 for
// index per CPython 3.14 compile.c). Helper: Compare returns (op, castBool).
// oparg layout: oparg = (mask << 5) | (cmp_op << 4) is NOT correct; 3.14 uses
// oparg = (cmp_op << 5) | flag; we only need cmp_op = oparg >> 5.
func CompareOp(oparg uint32) int { return int(oparg >> 5) }

// Intrinsic 1 and 2 function indices.
const (
	INTRINSIC_1_INVALID       = 0
	INTRINSIC_PRINT           = 1
	INTRINSIC_IMPORT_STAR     = 2
	INTRINSIC_STOPITERATION_ERROR = 3
	INTRINSIC_ASYNC_GEN_WRAP  = 4
	INTRINSIC_UNARY_POSITIVE  = 5
	INTRINSIC_LIST_TO_TUPLE   = 6
	INTRINSIC_TYPEVAR         = 7
	INTRINSIC_PARAMSPEC       = 8
	INTRINSIC_TYPEVARTUPLE    = 9
	INTRINSIC_SUBSCRIPT_GENERIC = 10
	INTRINSIC_TYPEALIAS       = 11
)

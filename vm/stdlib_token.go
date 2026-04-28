package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildToken constructs the token module with the full CPython 3.14 token
// constants, tok_name dict, EXACT_TOKEN_TYPES dict, and ISEOF/ISTERMINAL/
// ISNONTERMINAL helper functions.
func (i *Interp) buildToken() *object.Module {
	m := &object.Module{Name: "token", Dict: object.NewDict()}

	// ── Token integer constants ───────────────────────────────────────────

	tokenConsts := []struct {
		name string
		val  int64
	}{
		{"ENDMARKER", 0},
		{"NAME", 1},
		{"NUMBER", 2},
		{"STRING", 3},
		{"NEWLINE", 4},
		{"INDENT", 5},
		{"DEDENT", 6},
		{"LPAR", 7},
		{"RPAR", 8},
		{"LSQB", 9},
		{"RSQB", 10},
		{"COLON", 11},
		{"COMMA", 12},
		{"SEMI", 13},
		{"PLUS", 14},
		{"MINUS", 15},
		{"STAR", 16},
		{"SLASH", 17},
		{"VBAR", 18},
		{"AMPER", 19},
		{"LESS", 20},
		{"GREATER", 21},
		{"EQUAL", 22},
		{"DOT", 23},
		{"PERCENT", 24},
		{"LBRACE", 25},
		{"RBRACE", 26},
		{"EQEQUAL", 27},
		{"NOTEQUAL", 28},
		{"LESSEQUAL", 29},
		{"GREATEREQUAL", 30},
		{"TILDE", 31},
		{"CIRCUMFLEX", 32},
		{"LEFTSHIFT", 33},
		{"RIGHTSHIFT", 34},
		{"DOUBLESTAR", 35},
		{"PLUSEQUAL", 36},
		{"MINEQUAL", 37},
		{"STAREQUAL", 38},
		{"SLASHEQUAL", 39},
		{"PERCENTEQUAL", 40},
		{"AMPEREQUAL", 41},
		{"VBAREQUAL", 42},
		{"CIRCUMFLEXEQUAL", 43},
		{"LEFTSHIFTEQUAL", 44},
		{"RIGHTSHIFTEQUAL", 45},
		{"DOUBLESTAREQUAL", 46},
		{"DOUBLESLASH", 47},
		{"DOUBLESLASHEQUAL", 48},
		{"AT", 49},
		{"ATEQUAL", 50},
		{"RARROW", 51},
		{"ELLIPSIS", 52},
		{"COLONEQUAL", 53},
		{"EXCLAMATION", 54},
		{"OP", 55},
		{"TYPE_IGNORE", 56},
		{"TYPE_COMMENT", 57},
		{"SOFT_KEYWORD", 58},
		{"FSTRING_START", 59},
		{"FSTRING_MIDDLE", 60},
		{"FSTRING_END", 61},
		{"TSTRING_START", 62},
		{"TSTRING_MIDDLE", 63},
		{"TSTRING_END", 64},
		{"COMMENT", 65},
		{"NL", 66},
		{"ERRORTOKEN", 67},
		{"ENCODING", 68},
		{"N_TOKENS", 69},
		{"NT_OFFSET", 256},
	}

	for _, c := range tokenConsts {
		m.Dict.SetStr(c.name, object.NewInt(c.val))
	}

	// ── tok_name dict ─────────────────────────────────────────────────────

	tokName := object.NewDict()
	for _, c := range tokenConsts {
		tokName.Set(object.NewInt(c.val), &object.Str{V: c.name}) //nolint
	}
	m.Dict.SetStr("tok_name", tokName)

	// ── EXACT_TOKEN_TYPES dict ────────────────────────────────────────────

	exactTypes := object.NewDict()
	for _, pair := range [][2]interface{}{
		{"!", 54}, {"!=", 28}, {"%", 24}, {"%=", 40},
		{"&", 19}, {"&=", 41}, {"(", 7}, {")", 8},
		{"*", 16}, {"**", 35}, {"**=", 46}, {"*=", 38},
		{"+", 14}, {"+=", 36}, {",", 12}, {"-", 15},
		{"-=", 37}, {"->", 51}, {".", 23}, {"...", 52},
		{"/", 17}, {"//", 47}, {"//=", 48}, {"/=", 39},
		{":", 11}, {":=", 53}, {";", 13}, {"<", 20},
		{"<<", 33}, {"<<=", 44}, {"<=", 29}, {"=", 22},
		{"==", 27}, {">", 21}, {">=", 30}, {">>", 34},
		{">>=", 45}, {"@", 49}, {"@=", 50}, {"[", 9},
		{"]", 10}, {"^", 32}, {"^=", 43}, {"{", 25},
		{"|", 18}, {"|=", 42}, {"}", 26}, {"~", 31},
	} {
		k := &object.Str{V: pair[0].(string)}
		v := object.NewInt(int64(pair[1].(int)))
		exactTypes.Set(k, v) //nolint
	}
	m.Dict.SetStr("EXACT_TOKEN_TYPES", exactTypes)

	// ── Helper functions ──────────────────────────────────────────────────

	const ntOffset = int64(256)

	m.Dict.SetStr("ISEOF", &object.BuiltinFunc{
		Name: "ISEOF",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "ISEOF() missing argument")
			}
			x, ok := a[0].(*object.Int)
			if !ok {
				return nil, object.Errorf(i.typeErr, "ISEOF() argument must be int")
			}
			if x.Int64() == 0 {
				return object.True, nil
			}
			return object.False, nil
		},
	})

	m.Dict.SetStr("ISTERMINAL", &object.BuiltinFunc{
		Name: "ISTERMINAL",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "ISTERMINAL() missing argument")
			}
			x, ok := a[0].(*object.Int)
			if !ok {
				return nil, object.Errorf(i.typeErr, "ISTERMINAL() argument must be int")
			}
			if x.Int64() < ntOffset {
				return object.True, nil
			}
			return object.False, nil
		},
	})

	m.Dict.SetStr("ISNONTERMINAL", &object.BuiltinFunc{
		Name: "ISNONTERMINAL",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "ISNONTERMINAL() missing argument")
			}
			x, ok := a[0].(*object.Int)
			if !ok {
				return nil, object.Errorf(i.typeErr, "ISNONTERMINAL() argument must be int")
			}
			if x.Int64() >= ntOffset {
				return object.True, nil
			}
			return object.False, nil
		},
	})

	return m
}

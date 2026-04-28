package vm

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/tamnd/goipy/object"
)

// buildTokenize constructs the tokenize module with CPython 3.14 API.
func (i *Interp) buildTokenize() *object.Module {
	m := &object.Module{Name: "tokenize", Dict: object.NewDict()}

	// ── Re-export all token constants ─────────────────────────────────────

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

	// ── Re-export tok_name ────────────────────────────────────────────────

	tokName := object.NewDict()
	for _, c := range tokenConsts {
		tokName.Set(object.NewInt(c.val), &object.Str{V: c.name}) //nolint
	}
	m.Dict.SetStr("tok_name", tokName)

	// ── Re-export EXACT_TOKEN_TYPES ───────────────────────────────────────

	exactTypes := object.NewDict()
	exactTypesMap := map[string]int64{
		"!": 54, "!=": 28, "%": 24, "%=": 40,
		"&": 19, "&=": 41, "(": 7, ")": 8,
		"*": 16, "**": 35, "**=": 46, "*=": 38,
		"+": 14, "+=": 36, ",": 12, "-": 15,
		"-=": 37, "->": 51, ".": 23, "...": 52,
		"/": 17, "//": 47, "//=": 48, "/=": 39,
		":": 11, ":=": 53, ";": 13, "<": 20,
		"<<": 33, "<<=": 44, "<=": 29, "=": 22,
		"==": 27, ">": 21, ">=": 30, ">>": 34,
		">>=": 45, "@": 49, "@=": 50, "[": 9,
		"]": 10, "^": 32, "^=": 43, "{": 25,
		"|": 18, "|=": 42, "}": 26, "~": 31,
	}
	for op, code := range exactTypesMap {
		k := &object.Str{V: op}
		v := object.NewInt(code)
		exactTypes.Set(k, v) //nolint
	}
	m.Dict.SetStr("EXACT_TOKEN_TYPES", exactTypes)

	// ── Re-export ISEOF/ISTERMINAL/ISNONTERMINAL ──────────────────────────

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

	// ── BOM_UTF8 and tabsize ──────────────────────────────────────────────

	m.Dict.SetStr("BOM_UTF8", &object.Bytes{V: []byte{0xef, 0xbb, 0xbf}})
	m.Dict.SetStr("tabsize", object.NewInt(8))

	// ── Pattern strings ───────────────────────────────────────────────────

	m.Dict.SetStr("Whitespace", &object.Str{V: `[ \f\t]*`})
	m.Dict.SetStr("Comment", &object.Str{V: `#[^\r\n]*`})
	m.Dict.SetStr("Ignore", &object.Str{V: `[ \f\t]*(?:#[^\r\n]*)?\n*`})
	m.Dict.SetStr("Name", &object.Str{V: `\w+`})
	m.Dict.SetStr("Hexnumber", &object.Str{V: `0[xX](?:_?[0-9a-fA-F])+`})
	m.Dict.SetStr("Binnumber", &object.Str{V: `0[bB](?:_?[01])+`})
	m.Dict.SetStr("Octnumber", &object.Str{V: `0[oO](?:_?[0-7])+`})
	m.Dict.SetStr("Decnumber", &object.Str{V: `(?:0(?:_?0)*|[1-9](?:_?[0-9])*)`})
	m.Dict.SetStr("Intnumber", &object.Str{V: `0[xX](?:_?[0-9a-fA-F])+|0[bB](?:_?[01])+|0[oO](?:_?[0-7])+|(?:0(?:_?0)*|[1-9](?:_?[0-9])*)`})
	m.Dict.SetStr("Exponent", &object.Str{V: `[eE][-+]?[0-9](?:_?[0-9])*`})
	m.Dict.SetStr("Pointfloat", &object.Str{V: `(?:[0-9](?:_?[0-9])*)?\.(?:[0-9](?:_?[0-9])*)?(?:[eE][-+]?[0-9](?:_?[0-9])*)?|[0-9](?:_?[0-9])*[eE][-+]?[0-9](?:_?[0-9])*`})
	m.Dict.SetStr("Expfloat", &object.Str{V: `[0-9](?:_?[0-9])*[eE][-+]?[0-9](?:_?[0-9])*`})
	m.Dict.SetStr("Floatnumber", &object.Str{V: `(?:[0-9](?:_?[0-9])*)?\.(?:[0-9](?:_?[0-9])*)?(?:[eE][-+]?[0-9](?:_?[0-9])*)?|[0-9](?:_?[0-9])*[eE][-+]?[0-9](?:_?[0-9])*`})
	m.Dict.SetStr("Imagnumber", &object.Str{V: `[0-9](?:_?[0-9])*[jJ]|(?:[0-9](?:_?[0-9])*)?\.(?:[0-9](?:_?[0-9])*)?(?:[eE][-+]?[0-9](?:_?[0-9])*)?[jJ]`})
	m.Dict.SetStr("Number", &object.Str{V: `(?:[0-9](?:_?[0-9])*)?\.(?:[0-9](?:_?[0-9])*)?(?:[eE][-+]?[0-9](?:_?[0-9])*)?[jJ]?|[0-9](?:_?[0-9])*(?:[eE][-+]?[0-9](?:_?[0-9])*[jJ]?|[jJ])?|0[xX](?:_?[0-9a-fA-F])+|0[bB](?:_?[01])+|0[oO](?:_?[0-7])+`})
	m.Dict.SetStr("Special", &object.Str{V: `\r?\n|\r|\.\.\.|[()[\]{}]`})
	m.Dict.SetStr("Funny", &object.Str{V: `\r?\n|\r|\.\.\.|[()[\]{}]|[-+*/%&@|^=<>!~:]+`})

	// ── TokenError exception ──────────────────────────────────────────────

	tokErrClass := &object.Class{
		Name:  "TokenError",
		Dict:  object.NewDict(),
		Bases: []*object.Class{i.exception},
	}
	m.Dict.SetStr("TokenError", tokErrClass)

	// ── TokenInfo namedtuple ──────────────────────────────────────────────

	// TokenInfo(type, string, start, end, line) with .exact_type property
	tiCls := &object.Class{Name: "TokenInfo", Dict: object.NewDict()}

	fields := []string{"type", "string", "start", "end", "line"}
	fieldObjs := make([]object.Object, len(fields))
	for j, f := range fields {
		fieldObjs[j] = &object.Str{V: f}
	}
	tiCls.Dict.SetStr("_fields", &object.Tuple{V: fieldObjs})

	tiCls.Dict.SetStr("__new__", &object.BuiltinFunc{
		Name: "__new__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// a[0] = cls, a[1..] = positional args
			args := a
			if len(args) > 0 {
				if _, ok := args[0].(*object.Class); ok {
					args = args[1:]
				}
			}
			inst := &object.Instance{Class: tiCls, Dict: object.NewDict()}
			for j, f := range fields {
				if j < len(args) {
					inst.Dict.SetStr(f, args[j])
				} else {
					inst.Dict.SetStr(f, object.None)
				}
			}
			if kw != nil {
				ks, vs := kw.Items()
				for j, k := range ks {
					if ks2, ok := k.(*object.Str); ok {
						inst.Dict.SetStr(ks2.V, vs[j])
					}
				}
			}
			return inst, nil
		},
	})

	tiCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			args := a[1:]
			for j, f := range fields {
				if j < len(args) {
					self.Dict.SetStr(f, args[j])
				}
			}
			if kw != nil {
				ks, vs := kw.Items()
				for j, k := range ks {
					if ks2, ok := k.(*object.Str); ok {
						self.Dict.SetStr(ks2.V, vs[j])
					}
				}
			}
			return object.None, nil
		},
	})

	// exact_type property: for OP (55), look up exact token type; otherwise return type
	tiCls.Dict.SetStr("exact_type", &object.Property{
		Fget: &object.BuiltinFunc{
			Name: "exact_type",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) < 1 {
					return nil, object.Errorf(i.typeErr, "exact_type requires self")
				}
				self, ok := a[0].(*object.Instance)
				if !ok {
					return nil, object.Errorf(i.typeErr, "exact_type requires TokenInfo instance")
				}
				typ, _ := self.Dict.GetStr("type")
				typInt, ok := typ.(*object.Int)
				if !ok {
					return object.NewInt(0), nil
				}
				// If type is OP (55), look up exact type from string
				if typInt.Int64() == 55 {
					strVal, _ := self.Dict.GetStr("string")
					if sv, ok := strVal.(*object.Str); ok {
						if code, found := exactTypesMap[sv.V]; found {
							return object.NewInt(code), nil
						}
					}
					return typInt, nil
				}
				return typInt, nil
			},
		},
	})

	tiCls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "TokenInfo()"}, nil
			}
			self, ok := a[0].(*object.Instance)
			if !ok {
				return &object.Str{V: "TokenInfo()"}, nil
			}
			parts := make([]string, len(fields))
			for j, f := range fields {
				v, _ := self.Dict.GetStr(f)
				parts[j] = fmt.Sprintf("%s=%v", f, v)
			}
			return &object.Str{V: "TokenInfo(" + strings.Join(parts, ", ") + ")"}, nil
		},
	})

	m.Dict.SetStr("TokenInfo", tiCls)

	// ── makeTokenInfo helper ──────────────────────────────────────────────

	makeTokenInfo := func(typ int64, str string, sl, sc, el, ec int, line string) *object.Instance {
		inst := &object.Instance{Class: tiCls, Dict: object.NewDict()}
		inst.Dict.SetStr("type", object.NewInt(typ))
		inst.Dict.SetStr("string", &object.Str{V: str})
		start := &object.Tuple{V: []object.Object{object.NewInt(int64(sl)), object.NewInt(int64(sc))}}
		end := &object.Tuple{V: []object.Object{object.NewInt(int64(el)), object.NewInt(int64(ec))}}
		inst.Dict.SetStr("start", start)
		inst.Dict.SetStr("end", end)
		inst.Dict.SetStr("line", &object.Str{V: line})
		return inst
	}

	// ── goTokenize: minimal Python tokenizer ─────────────────────────────

	goTokenize := func(source string, emitEncoding bool, encoding string) ([]*object.Instance, error) {
		var tokens []*object.Instance

		if emitEncoding {
			tokens = append(tokens, makeTokenInfo(68, encoding, 0, 0, 0, 0, ""))
		}

		lines := strings.SplitAfter(source, "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		// Three-char and two-char operators (order matters: longest match first)
		threeCharOps := []string{"**=", "//=", "<<=", ">>=", "..."}
		twoCharOps := []string{"==", "!=", "<=", ">=", "<<", ">>", "**", "//", "->", ":=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "@="}

		for lineIdx, line := range lines {
			lineNum := lineIdx + 1
			col := 0
			runes := []rune(line)
			n := len(runes)

			for col < n {
				// Skip whitespace (non-leading)
				if runes[col] == ' ' || runes[col] == '\t' || runes[col] == '\f' {
					col++
					continue
				}

				// Comment
				if runes[col] == '#' {
					start := col
					for col < n && runes[col] != '\n' && runes[col] != '\r' {
						col++
					}
					tokens = append(tokens, makeTokenInfo(65, string(runes[start:col]), lineNum, start, lineNum, col, line))
					continue
				}

				// Newline
				if runes[col] == '\n' || runes[col] == '\r' {
					end := col + 1
					if runes[col] == '\r' && end < n && runes[end] == '\n' {
						end++
					}
					tokens = append(tokens, makeTokenInfo(4, string(runes[col:end]), lineNum, col, lineNum, end, line))
					col = end
					continue
				}

				// Number
				if unicode.IsDigit(runes[col]) || (runes[col] == '.' && col+1 < n && unicode.IsDigit(runes[col+1])) {
					start := col
					// Check hex/bin/oct
					if runes[col] == '0' && col+1 < n {
						next := runes[col+1]
						if next == 'x' || next == 'X' {
							col += 2
							for col < n && (unicode.IsDigit(runes[col]) || (runes[col] >= 'a' && runes[col] <= 'f') || (runes[col] >= 'A' && runes[col] <= 'F') || runes[col] == '_') {
								col++
							}
							tokens = append(tokens, makeTokenInfo(2, string(runes[start:col]), lineNum, start, lineNum, col, line))
							continue
						}
						if next == 'b' || next == 'B' {
							col += 2
							for col < n && (runes[col] == '0' || runes[col] == '1' || runes[col] == '_') {
								col++
							}
							tokens = append(tokens, makeTokenInfo(2, string(runes[start:col]), lineNum, start, lineNum, col, line))
							continue
						}
						if next == 'o' || next == 'O' {
							col += 2
							for col < n && (runes[col] >= '0' && runes[col] <= '7' || runes[col] == '_') {
								col++
							}
							tokens = append(tokens, makeTokenInfo(2, string(runes[start:col]), lineNum, start, lineNum, col, line))
							continue
						}
					}
					// Decimal / float
					for col < n && (unicode.IsDigit(runes[col]) || runes[col] == '_') {
						col++
					}
					isFloat := false
					if col < n && runes[col] == '.' {
						isFloat = true
						col++
						for col < n && (unicode.IsDigit(runes[col]) || runes[col] == '_') {
							col++
						}
					}
					if col < n && (runes[col] == 'e' || runes[col] == 'E') {
						isFloat = true
						col++
						if col < n && (runes[col] == '+' || runes[col] == '-') {
							col++
						}
						for col < n && (unicode.IsDigit(runes[col]) || runes[col] == '_') {
							col++
						}
					}
					if col < n && (runes[col] == 'j' || runes[col] == 'J') {
						col++
					}
					_ = isFloat
					tokens = append(tokens, makeTokenInfo(2, string(runes[start:col]), lineNum, start, lineNum, col, line))
					continue
				}

				// String
				if runes[col] == '"' || runes[col] == '\'' ||
					(col+1 < n && (runes[col+1] == '"' || runes[col+1] == '\'') &&
						(runes[col] == 'r' || runes[col] == 'R' || runes[col] == 'b' || runes[col] == 'B' ||
							runes[col] == 'u' || runes[col] == 'U' || runes[col] == 'f' || runes[col] == 'F')) ||
					(col+2 < n && (runes[col+2] == '"' || runes[col+2] == '\'') &&
						(runes[col] == 'r' || runes[col] == 'R' || runes[col] == 'b' || runes[col] == 'B' ||
							runes[col] == 'f' || runes[col] == 'F') &&
						(runes[col+1] == 'r' || runes[col+1] == 'R' || runes[col+1] == 'b' || runes[col+1] == 'B')) {

					start := col
					// Skip prefix
					for col < n && (runes[col] == 'r' || runes[col] == 'R' || runes[col] == 'b' || runes[col] == 'B' ||
						runes[col] == 'u' || runes[col] == 'U' || runes[col] == 'f' || runes[col] == 'F') {
						col++
					}
					if col >= n {
						tokens = append(tokens, makeTokenInfo(67, string(runes[start:col]), lineNum, start, lineNum, col, line))
						continue
					}
					quote := runes[col]
					col++
					// Check for triple quote
					triple := false
					if col+1 < n && runes[col] == quote && runes[col+1] == quote {
						triple = true
						col += 2
					}
					for col < n {
						ch := runes[col]
						if ch == '\\' {
							col += 2
							continue
						}
						if triple {
							if ch == quote && col+2 < n && runes[col+1] == quote && runes[col+2] == quote {
								col += 3
								break
							}
						} else {
							if ch == quote {
								col++
								break
							}
							if ch == '\n' || ch == '\r' {
								break
							}
						}
						col++
					}
					tokens = append(tokens, makeTokenInfo(3, string(runes[start:col]), lineNum, start, lineNum, col, line))
					continue
				}

				// Name / identifier
				if unicode.IsLetter(runes[col]) || runes[col] == '_' {
					start := col
					for col < n && (unicode.IsLetter(runes[col]) || unicode.IsDigit(runes[col]) || runes[col] == '_') {
						col++
					}
					tokens = append(tokens, makeTokenInfo(1, string(runes[start:col]), lineNum, start, lineNum, col, line))
					continue
				}

				// Operators (longest match first)
				matched := false
				if col+3 <= n {
					s3 := string(runes[col : col+3])
					for _, op := range threeCharOps {
						if s3 == op {
							tokens = append(tokens, makeTokenInfo(55, s3, lineNum, col, lineNum, col+3, line))
							col += 3
							matched = true
							break
						}
					}
				}
				if !matched && col+2 <= n {
					s2 := string(runes[col : col+2])
					for _, op := range twoCharOps {
						if s2 == op {
							tokens = append(tokens, makeTokenInfo(55, s2, lineNum, col, lineNum, col+2, line))
							col += 2
							matched = true
							break
						}
					}
				}
				if !matched {
					s1 := string(runes[col : col+1])
					opChars := "!%&()*+,-./:;<=>?@[]^{|}~"
					if strings.ContainsRune(opChars, runes[col]) {
						tokens = append(tokens, makeTokenInfo(55, s1, lineNum, col, lineNum, col+1, line))
					} else {
						tokens = append(tokens, makeTokenInfo(67, s1, lineNum, col, lineNum, col+1, line))
					}
					col++
				}
			}
		}

		// ENDMARKER
		endLine := len(lines) + 1
		tokens = append(tokens, makeTokenInfo(0, "", endLine, 0, endLine, 0, ""))
		return tokens, nil
	}

	// ── detect_encoding ───────────────────────────────────────────────────

	detectEncoding := func(ii *Interp, readlineFn object.Object) (string, []object.Object, error) {
		// Read up to 2 lines to find coding declaration or BOM
		var rawLines []object.Object
		encoding := "utf-8"
		bom := false

		for k := 0; k < 2; k++ {
			line, err := ii.callObject(readlineFn, nil, nil)
			if err != nil {
				break
			}
			b, ok := line.(*object.Bytes)
			if !ok || len(b.V) == 0 {
				break
			}
			rawLines = append(rawLines, b)
			// Check BOM on first line
			if k == 0 && len(b.V) >= 3 && b.V[0] == 0xef && b.V[1] == 0xbb && b.V[2] == 0xbf {
				encoding = "utf-8-sig"
				bom = true
				_ = bom
				continue
			}
			// Check for coding: declaration
			lineStr := string(b.V)
			for _, prefix := range []string{"# -*- coding:", "# coding:", "# coding ="} {
				if idx := strings.Index(lineStr, prefix); idx >= 0 {
					rest := lineStr[idx+len(prefix):]
					rest = strings.TrimSpace(rest)
					if i2 := strings.IndexAny(rest, " \t\r\n*"); i2 >= 0 {
						rest = rest[:i2]
					}
					rest = strings.TrimSuffix(rest, " -*-")
					rest = strings.TrimSpace(rest)
					if rest != "" {
						encoding = strings.ToLower(rest)
					}
					break
				}
			}
		}

		return encoding, rawLines, nil
	}

	// ── tokenize(readline) ────────────────────────────────────────────────

	m.Dict.SetStr("tokenize", &object.BuiltinFunc{
		Name: "tokenize",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := i
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(ii.typeErr, "tokenize() missing argument")
			}
			readlineFn := a[0]

			// Detect encoding by reading first lines
			encoding, firstLines, err := detectEncoding(ii, readlineFn)
			if err != nil {
				return nil, err
			}

			// Collect remaining bytes
			var sb strings.Builder
			// Replay first lines
			for _, fl := range firstLines {
				if b, ok := fl.(*object.Bytes); ok {
					sb.Write(b.V)
				}
			}
			// Read rest
			for {
				line, err2 := ii.callObject(readlineFn, nil, nil)
				if err2 != nil {
					break
				}
				if b, ok := line.(*object.Bytes); ok {
					if len(b.V) == 0 {
						break
					}
					sb.Write(b.V)
				} else {
					break
				}
			}

			source := sb.String()
			// Strip UTF-8 BOM if present
			if strings.HasPrefix(source, "\xef\xbb\xbf") {
				source = source[3:]
			}

			toks, err := goTokenize(source, true, encoding)
			if err != nil {
				return nil, err
			}

			objs := make([]object.Object, len(toks))
			for j, t := range toks {
				objs[j] = t
			}
			return &object.List{V: objs}, nil
		},
	})

	// ── generate_tokens(readline) ─────────────────────────────────────────

	m.Dict.SetStr("generate_tokens", &object.BuiltinFunc{
		Name: "generate_tokens",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := i
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(ii.typeErr, "generate_tokens() missing argument")
			}
			readlineFn := a[0]

			var sb strings.Builder
			for {
				line, err2 := ii.callObject(readlineFn, nil, nil)
				if err2 != nil {
					break
				}
				if s, ok := line.(*object.Str); ok {
					if s.V == "" {
						break
					}
					sb.WriteString(s.V)
				} else {
					break
				}
			}

			toks, err := goTokenize(sb.String(), false, "")
			if err != nil {
				return nil, err
			}

			objs := make([]object.Object, len(toks))
			for j, t := range toks {
				objs[j] = t
			}
			return &object.List{V: objs}, nil
		},
	})

	// ── untokenize(iterable) ──────────────────────────────────────────────

	m.Dict.SetStr("untokenize", &object.BuiltinFunc{
		Name: "untokenize",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := i
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(ii.typeErr, "untokenize() missing argument")
			}
			items, err := iterate(ii, a[0])
			if err != nil {
				return nil, err
			}

			var sb strings.Builder
			prevRow, prevCol := 1, 0

			for _, item := range items {
				switch t := item.(type) {
				case *object.Instance:
					// Full TokenInfo: use position info
					typObj, _ := t.Dict.GetStr("type")
					strObj, _ := t.Dict.GetStr("string")
					startObj, _ := t.Dict.GetStr("start")
					if typInt, ok := typObj.(*object.Int); ok {
						if typInt.Int64() == 0 { // ENDMARKER
							continue
						}
					}
					str := ""
					if sv, ok := strObj.(*object.Str); ok {
						str = sv.V
					}
					if tup, ok := startObj.(*object.Tuple); ok && len(tup.V) >= 2 {
						rowObj, colObj := tup.V[0], tup.V[1]
						row, col := prevRow, prevCol
						if ri, ok := rowObj.(*object.Int); ok {
							row = int(ri.Int64())
						}
						if ci, ok := colObj.(*object.Int); ok {
							col = int(ci.Int64())
						}
						// Add newlines to reach target row
						for row > prevRow {
							sb.WriteString("\n")
							prevRow++
							prevCol = 0
						}
						// Add spaces to reach target column
						for col > prevCol {
							sb.WriteString(" ")
							prevCol++
						}
					}
					sb.WriteString(str)
					prevRow += strings.Count(str, "\n")
					if lastNL := strings.LastIndex(str, "\n"); lastNL >= 0 {
						prevCol = len(str) - lastNL - 1
					} else {
						prevCol += len(str)
					}
				case *object.Tuple:
					// 2-tuple (type, string) or 5-tuple
					if len(t.V) >= 2 {
						typObj := t.V[0]
						strObj := t.V[1]
						if typInt, ok := typObj.(*object.Int); ok {
							if typInt.Int64() == 0 { // ENDMARKER
								continue
							}
						}
						str := ""
						if sv, ok := strObj.(*object.Str); ok {
							str = sv.V
						}
						if len(t.V) >= 3 {
							// Has position info
							if tup, ok := t.V[2].(*object.Tuple); ok && len(tup.V) >= 2 {
								row, col := prevRow, prevCol
								if ri, ok := tup.V[0].(*object.Int); ok {
									row = int(ri.Int64())
								}
								if ci, ok := tup.V[1].(*object.Int); ok {
									col = int(ci.Int64())
								}
								for row > prevRow {
									sb.WriteString("\n")
									prevRow++
									prevCol = 0
								}
								for col > prevCol {
									sb.WriteString(" ")
									prevCol++
								}
							}
						} else {
							// 2-tuple: just space-join
							if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "\n") && !strings.HasSuffix(sb.String(), " ") {
								sb.WriteString(" ")
							}
						}
						sb.WriteString(str)
						prevRow += strings.Count(str, "\n")
						if lastNL := strings.LastIndex(str, "\n"); lastNL >= 0 {
							prevCol = len(str) - lastNL - 1
						} else {
							prevCol += len(str)
						}
					}
				}
			}
			return &object.Str{V: sb.String()}, nil
		},
	})

	// ── detect_encoding(readline) ─────────────────────────────────────────

	m.Dict.SetStr("detect_encoding", &object.BuiltinFunc{
		Name: "detect_encoding",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := i
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(ii.typeErr, "detect_encoding() missing argument")
			}
			encoding, rawLines, err := detectEncoding(ii, a[0])
			if err != nil {
				return nil, err
			}
			lineObjs := make([]object.Object, len(rawLines))
			copy(lineObjs, rawLines)
			result := &object.Tuple{V: []object.Object{
				&object.Str{V: encoding},
				&object.List{V: lineObjs},
			}}
			return result, nil
		},
	})

	// ── open(filename) ────────────────────────────────────────────────────

	m.Dict.SetStr("open", &object.BuiltinFunc{
		Name: "open",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "open() missing argument")
			}
			// Return a basic file-like object stub
			return &object.Instance{Class: &object.Class{Name: "TextIOWrapper", Dict: object.NewDict()}, Dict: object.NewDict()}, nil
		},
	})

	// ── Untokenizer class ─────────────────────────────────────────────────

	untokCls := &object.Class{Name: "Untokenizer", Dict: object.NewDict()}
	untokCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			self.Dict.SetStr("tokens", &object.List{V: nil})
			self.Dict.SetStr("prev_row", object.NewInt(1))
			self.Dict.SetStr("prev_col", object.NewInt(0))
			self.Dict.SetStr("encoding", &object.Str{V: "utf-8"})
			return object.None, nil
		},
	})
	untokCls.Dict.SetStr("untokenize", &object.BuiltinFunc{
		Name: "untokenize",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := i
			if len(a) < 2 {
				return &object.Str{V: ""}, nil
			}
			items, err := iterate(ii, a[1])
			if err != nil {
				return nil, err
			}
			var sb strings.Builder
			for _, item := range items {
				if tup, ok := item.(*object.Tuple); ok && len(tup.V) >= 2 {
					if sv, ok := tup.V[1].(*object.Str); ok {
						if sb.Len() > 0 {
							sb.WriteString(" ")
						}
						sb.WriteString(sv.V)
					}
				}
			}
			return &object.Str{V: sb.String()}, nil
		},
	})
	untokCls.Dict.SetStr("compat", &object.BuiltinFunc{
		Name: "compat",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	m.Dict.SetStr("Untokenizer", untokCls)

	return m
}

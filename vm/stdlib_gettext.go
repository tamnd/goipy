package vm

import (
	"encoding/binary"
	"strings"

	"github.com/tamnd/goipy/object"
)

// ── Plural-form expression evaluator ──────────────────────────────────────────
// Evaluates simple C-like plural-form expressions (gettext Plural-Forms header).

type pluralExprEval struct {
	s   string
	pos int
	n   int64
}

func evalPluralExpr(expr string, n int64) int64 {
	e := &pluralExprEval{s: strings.TrimSpace(expr), n: n}
	return e.parseTernary()
}

func (e *pluralExprEval) skipWS() {
	for e.pos < len(e.s) && (e.s[e.pos] == ' ' || e.s[e.pos] == '\t') {
		e.pos++
	}
}

func (e *pluralExprEval) peek(k int) string {
	e.skipWS()
	if e.pos+k > len(e.s) {
		return ""
	}
	return e.s[e.pos : e.pos+k]
}

func (e *pluralExprEval) parseTernary() int64 {
	cond := e.parseOr()
	e.skipWS()
	if e.pos < len(e.s) && e.s[e.pos] == '?' {
		e.pos++
		a := e.parseTernary()
		e.skipWS()
		if e.pos < len(e.s) && e.s[e.pos] == ':' {
			e.pos++
		}
		b := e.parseTernary()
		if cond != 0 {
			return a
		}
		return b
	}
	return cond
}

func (e *pluralExprEval) parseOr() int64 {
	left := e.parseAnd()
	for e.peek(2) == "||" {
		e.pos += 2
		right := e.parseAnd()
		if left != 0 || right != 0 {
			left = 1
		} else {
			left = 0
		}
	}
	return left
}

func (e *pluralExprEval) parseAnd() int64 {
	left := e.parseEq()
	for e.peek(2) == "&&" {
		e.pos += 2
		right := e.parseEq()
		if left != 0 && right != 0 {
			left = 1
		} else {
			left = 0
		}
	}
	return left
}

func (e *pluralExprEval) parseEq() int64 {
	left := e.parseRel()
	for {
		switch e.peek(2) {
		case "==":
			e.pos += 2
			r := e.parseRel()
			if left == r {
				left = 1
			} else {
				left = 0
			}
		case "!=":
			e.pos += 2
			r := e.parseRel()
			if left != r {
				left = 1
			} else {
				left = 0
			}
		default:
			return left
		}
	}
}

func (e *pluralExprEval) parseRel() int64 {
	left := e.parseAdd()
	for {
		switch e.peek(2) {
		case ">=":
			e.pos += 2
			if left >= e.parseAdd() {
				left = 1
			} else {
				left = 0
			}
		case "<=":
			e.pos += 2
			if left <= e.parseAdd() {
				left = 1
			} else {
				left = 0
			}
		default:
			switch e.peek(1) {
			case ">":
				e.pos++
				if left > e.parseAdd() {
					left = 1
				} else {
					left = 0
				}
			case "<":
				e.pos++
				if left < e.parseAdd() {
					left = 1
				} else {
					left = 0
				}
			default:
				return left
			}
		}
	}
}

func (e *pluralExprEval) parseAdd() int64 {
	left := e.parseMul()
	for {
		switch e.peek(1) {
		case "+":
			e.pos++
			left += e.parseMul()
		case "-":
			e.pos++
			left -= e.parseMul()
		default:
			return left
		}
	}
}

func (e *pluralExprEval) parseMul() int64 {
	left := e.parseUnary()
	for {
		switch e.peek(1) {
		case "*":
			e.pos++
			left *= e.parseUnary()
		case "/":
			e.pos++
			r := e.parseUnary()
			if r != 0 {
				left /= r
			}
		case "%":
			e.pos++
			r := e.parseUnary()
			if r != 0 {
				left %= r
			}
		default:
			return left
		}
	}
}

func (e *pluralExprEval) parseUnary() int64 {
	e.skipWS()
	if e.pos < len(e.s) {
		switch e.s[e.pos] {
		case '!':
			e.pos++
			if e.parseUnary() == 0 {
				return 1
			}
			return 0
		case '-':
			e.pos++
			return -e.parseUnary()
		}
	}
	return e.parsePrimary()
}

func (e *pluralExprEval) parsePrimary() int64 {
	e.skipWS()
	if e.pos >= len(e.s) {
		return 0
	}
	if e.s[e.pos] == '(' {
		e.pos++
		v := e.parseTernary()
		e.skipWS()
		if e.pos < len(e.s) && e.s[e.pos] == ')' {
			e.pos++
		}
		return v
	}
	// Variable 'n'
	if e.s[e.pos] == 'n' {
		e.pos++
		return e.n
	}
	// Integer literal
	var val int64
	for e.pos < len(e.s) && e.s[e.pos] >= '0' && e.s[e.pos] <= '9' {
		val = val*10 + int64(e.s[e.pos]-'0')
		e.pos++
	}
	return val
}

// ── gettext module builder ────────────────────────────────────────────────────

func (i *Interp) buildGettext() *object.Module {
	m := &object.Module{Name: "gettext", Dict: object.NewDict()}

	// ── Module-level state ───────────────────────────────────────────────────
	currentDomain := "messages"
	localeDirs := map[string]string{}

	// ── readBytesFromFP: read all bytes from a file-like object ───────────────
	readBytesFromFP := func(fp object.Object) ([]byte, error) {
		switch f := fp.(type) {
		case *object.BytesIO:
			return f.V[f.Pos:], nil
		case *object.StringIO:
			return f.V[f.Pos:], nil
		}
		// Call fp.read() method
		readFn, err := i.getAttr(fp, "read")
		if err != nil {
			return nil, err
		}
		result, err2 := i.callObject(readFn, nil, nil)
		if err2 != nil {
			return nil, err2
		}
		switch r := result.(type) {
		case *object.Bytes:
			return r.V, nil
		case *object.Str:
			return []byte(r.V), nil
		}
		return nil, nil
	}

	// ── parseMOBytes: parse a GNU .mo binary, return catalog + metadata ───────
	type moResult struct {
		catalog     map[string]string // orig → trans (raw bytes as string)
		charset     string
		info        map[string]string
		pluralExpr  string
		nplurals    int64
	}

	parseMOBytes := func(buf []byte) (*moResult, error) {
		if len(buf) < 28 {
			return nil, object.Errorf(i.valueErr, "gettext: .mo file too short")
		}
		var bo binary.ByteOrder
		magic := binary.LittleEndian.Uint32(buf[0:4])
		if magic == 0x950412de {
			bo = binary.LittleEndian
		} else if magic == 0xde120495 {
			bo = binary.BigEndian
		} else {
			return nil, object.Errorf(i.valueErr, "gettext: bad .mo magic")
		}
		count := int(bo.Uint32(buf[8:12]))
		origOff := int(bo.Uint32(buf[12:16]))
		transOff := int(bo.Uint32(buf[16:20]))

		getString := func(tableOff, idx int) (string, bool) {
			off := tableOff + idx*8
			if off+8 > len(buf) {
				return "", false
			}
			length := int(bo.Uint32(buf[off : off+4]))
			start := int(bo.Uint32(buf[off+4 : off+8]))
			end := start + length
			if end > len(buf) {
				return "", false
			}
			return string(buf[start:end]), true
		}

		res := &moResult{
			catalog:    make(map[string]string),
			info:       make(map[string]string),
			pluralExpr: "(n != 1)",
			nplurals:   2,
		}

		for idx := 0; idx < count; idx++ {
			orig, ok1 := getString(origOff, idx)
			trans, ok2 := getString(transOff, idx)
			if !ok1 || !ok2 {
				continue
			}
			if orig == "" {
				// Parse metadata header
				for _, line := range strings.Split(trans, "\n") {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					colonIdx := strings.Index(line, ":")
					if colonIdx < 0 {
						continue
					}
					key := strings.ToLower(strings.TrimSpace(line[:colonIdx]))
					val := strings.TrimSpace(line[colonIdx+1:])
					res.info[key] = val
					if key == "content-type" {
						// Extract charset=... from Content-Type value
						for _, part := range strings.Split(val, ";") {
							part = strings.TrimSpace(part)
							if strings.HasPrefix(strings.ToLower(part), "charset=") {
								res.charset = strings.TrimSpace(part[8:])
							}
						}
					}
					if key == "plural-forms" {
						for _, part := range strings.Split(val, ";") {
							part = strings.TrimSpace(part)
							if strings.HasPrefix(part, "plural=") {
								res.pluralExpr = strings.TrimSpace(part[7:])
							}
							if strings.HasPrefix(part, "nplurals=") {
								np := strings.TrimSpace(part[9:])
								var npv int64
								for _, c := range np {
									if c >= '0' && c <= '9' {
										npv = npv*10 + int64(c-'0')
									}
								}
								if npv > 0 {
									res.nplurals = npv
								}
							}
						}
					}
				}
				continue
			}
			res.catalog[orig] = trans
		}
		return res, nil
	}

	// ── buildTranslationsClass: shared factory for NullTranslations and GNU ──
	// Returns a class; the caller fills in _parse if needed.
	buildNullTranslations := func() *object.Class {
		cls := &object.Class{Name: "NullTranslations", Dict: object.NewDict()}

		cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				inst := a[0].(*object.Instance)
				inst.Dict.SetStr("_catalog", object.NewDict())
				inst.Dict.SetStr("_info", object.NewDict())
				inst.Dict.SetStr("_charset", object.None)
				inst.Dict.SetStr("_fallback", object.None)
				inst.Dict.SetStr("_plural_expr", &object.Str{V: "(n != 1)"})
				inst.Dict.SetStr("_nplurals", object.NewInt(2))
				// fp argument
				var fp object.Object
				if len(a) >= 2 {
					fp = a[1]
				} else if kw != nil {
					if v, ok := kw.GetStr("fp"); ok {
						fp = v
					}
				}
				if fp != nil {
					if _, isNone := fp.(*object.NoneType); !isNone {
						parseFn, ok := classLookup(inst.Class, "_parse")
						if ok {
							_, err := i.callObject(parseFn, []object.Object{inst, fp}, nil)
							if err != nil {
								return nil, err
							}
						}
					}
				}
				return object.None, nil
			}})

		cls.Dict.SetStr("_parse", &object.BuiltinFunc{Name: "_parse",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}})

		cls.Dict.SetStr("add_fallback", &object.BuiltinFunc{Name: "add_fallback",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				inst := a[0].(*object.Instance)
				if len(a) >= 2 {
					// Walk to end of chain and append
					cur := inst
					for {
						fb, _ := cur.Dict.GetStr("_fallback")
						if _, isNone := fb.(*object.NoneType); isNone {
							cur.Dict.SetStr("_fallback", a[1])
							break
						}
						if next, ok := fb.(*object.Instance); ok {
							cur = next
						} else {
							break
						}
					}
				}
				return object.None, nil
			}})

		cls.Dict.SetStr("gettext", &object.BuiltinFunc{Name: "gettext",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				inst := a[0].(*object.Instance)
				msg, ok := a[1].(*object.Str)
				if !ok {
					return a[1], nil
				}
				cat, _ := inst.Dict.GetStr("_catalog")
				if d, ok := cat.(*object.Dict); ok {
					if v, ok := d.GetStr(msg.V); ok {
						return v, nil
					}
				}
				// Try fallback
				if fb, ok := inst.Dict.GetStr("_fallback"); ok {
					if fbInst, ok := fb.(*object.Instance); ok {
						if fn, ok := classLookup(fbInst.Class, "gettext"); ok {
							return i.callObject(fn, []object.Object{fbInst, msg}, nil)
						}
					}
				}
				return msg, nil
			}})

		cls.Dict.SetStr("ngettext", &object.BuiltinFunc{Name: "ngettext",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				inst := a[0].(*object.Instance)
				msg1, ok1 := a[1].(*object.Str)
				msg2, ok2 := a[2].(*object.Str)
				if !ok1 || !ok2 {
					return a[1], nil
				}
				nObj := a[3]
				n, _ := toInt64(nObj)

				// Look up in catalog: key = msg1 + \x00 + msg2
				key := msg1.V + "\x00" + msg2.V
				cat, _ := inst.Dict.GetStr("_catalog")
				if d, ok := cat.(*object.Dict); ok {
					if tv, ok := d.GetStr(key); ok {
						if transStr, ok := tv.(*object.Str); ok {
							parts := strings.Split(transStr.V, "\x00")
							pluralExprStr := "(n != 1)"
							if pv, ok := inst.Dict.GetStr("_plural_expr"); ok {
								if ps, ok := pv.(*object.Str); ok {
									pluralExprStr = ps.V
								}
							}
							idx := int(evalPluralExpr(pluralExprStr, n))
							if idx < 0 {
								idx = 0
							}
							if idx >= len(parts) {
								idx = len(parts) - 1
							}
							return &object.Str{V: parts[idx]}, nil
						}
					}
				}
				// Try fallback
				if fb, ok := inst.Dict.GetStr("_fallback"); ok {
					if fbInst, ok := fb.(*object.Instance); ok {
						if fn, ok := classLookup(fbInst.Class, "ngettext"); ok {
							return i.callObject(fn, []object.Object{fbInst, msg1, msg2, nObj}, nil)
						}
					}
				}
				// Default English rule
				if n == 1 {
					return msg1, nil
				}
				return msg2, nil
			}})

		cls.Dict.SetStr("pgettext", &object.BuiltinFunc{Name: "pgettext",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				inst := a[0].(*object.Instance)
				ctx, ok1 := a[1].(*object.Str)
				msg, ok2 := a[2].(*object.Str)
				if !ok1 || !ok2 {
					if len(a) >= 3 {
						return a[2], nil
					}
					return object.None, nil
				}
				key := ctx.V + "\x04" + msg.V
				cat, _ := inst.Dict.GetStr("_catalog")
				if d, ok := cat.(*object.Dict); ok {
					if v, ok := d.GetStr(key); ok {
						return v, nil
					}
				}
				// Try fallback
				if fb, ok := inst.Dict.GetStr("_fallback"); ok {
					if fbInst, ok := fb.(*object.Instance); ok {
						if fn, ok := classLookup(fbInst.Class, "pgettext"); ok {
							return i.callObject(fn, []object.Object{fbInst, ctx, msg}, nil)
						}
					}
				}
				return msg, nil
			}})

		cls.Dict.SetStr("npgettext", &object.BuiltinFunc{Name: "npgettext",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				inst := a[0].(*object.Instance)
				ctx, ok1 := a[1].(*object.Str)
				msg1, ok2 := a[2].(*object.Str)
				msg2, ok3 := a[3].(*object.Str)
				if !ok1 || !ok2 || !ok3 {
					return object.None, nil
				}
				nObj := a[4]
				n, _ := toInt64(nObj)

				key := ctx.V + "\x04" + msg1.V + "\x00" + msg2.V
				cat, _ := inst.Dict.GetStr("_catalog")
				if d, ok := cat.(*object.Dict); ok {
					if tv, ok := d.GetStr(key); ok {
						if transStr, ok := tv.(*object.Str); ok {
							parts := strings.Split(transStr.V, "\x00")
							pluralExprStr := "(n != 1)"
							if pv, ok := inst.Dict.GetStr("_plural_expr"); ok {
								if ps, ok := pv.(*object.Str); ok {
									pluralExprStr = ps.V
								}
							}
							idx := int(evalPluralExpr(pluralExprStr, n))
							if idx < 0 {
								idx = 0
							}
							if idx >= len(parts) {
								idx = len(parts) - 1
							}
							return &object.Str{V: parts[idx]}, nil
						}
					}
				}
				// Try fallback
				if fb, ok := inst.Dict.GetStr("_fallback"); ok {
					if fbInst, ok := fb.(*object.Instance); ok {
						if fn, ok := classLookup(fbInst.Class, "npgettext"); ok {
							return i.callObject(fn, []object.Object{fbInst, ctx, msg1, msg2, nObj}, nil)
						}
					}
				}
				if n == 1 {
					return msg1, nil
				}
				return msg2, nil
			}})

		cls.Dict.SetStr("charset", &object.BuiltinFunc{Name: "charset",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				inst := a[0].(*object.Instance)
				if v, ok := inst.Dict.GetStr("_charset"); ok {
					return v, nil
				}
				return object.None, nil
			}})

		cls.Dict.SetStr("info", &object.BuiltinFunc{Name: "info",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				inst := a[0].(*object.Instance)
				if v, ok := inst.Dict.GetStr("_info"); ok {
					return v, nil
				}
				return object.NewDict(), nil
			}})

		cls.Dict.SetStr("install", &object.BuiltinFunc{Name: "install",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				inst := a[0].(*object.Instance)
				gettextFn, ok := classLookup(inst.Class, "gettext")
				if !ok {
					return object.None, nil
				}
				capturedInst := inst
				capturedFn := gettextFn
				i.Builtins.SetStr("_", &object.BuiltinFunc{Name: "_",
					Call: func(_ any, inner []object.Object, _ *object.Dict) (object.Object, error) {
						return i.callObject(capturedFn, append([]object.Object{capturedInst}, inner...), nil)
					}})
				return object.None, nil
			}})

		return cls
	}

	// ── NullTranslations ─────────────────────────────────────────────────────
	nullTransCls := buildNullTranslations()
	m.Dict.SetStr("NullTranslations", nullTransCls)

	// ── GNUTranslations ──────────────────────────────────────────────────────
	// Builds on NullTranslations: same class layout, overrides __init__ to
	// call the Go-side .mo parser, then delegates all lookups to the same
	// method set (catalog is already populated).
	gnuTransCls := buildNullTranslations()
	gnuTransCls.Name = "GNUTranslations"

	// Override __init__ to parse the fp argument.
	gnuTransCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("_catalog", object.NewDict())
			inst.Dict.SetStr("_info", object.NewDict())
			inst.Dict.SetStr("_charset", object.None)
			inst.Dict.SetStr("_fallback", object.None)
			inst.Dict.SetStr("_plural_expr", &object.Str{V: "(n != 1)"})
			inst.Dict.SetStr("_nplurals", object.NewInt(2))

			var fp object.Object
			if len(a) >= 2 {
				fp = a[1]
			} else if kw != nil {
				if v, ok := kw.GetStr("fp"); ok {
					fp = v
				}
			}
			if fp == nil {
				return object.None, nil
			}
			if _, isNone := fp.(*object.NoneType); isNone {
				return object.None, nil
			}

			buf, err := readBytesFromFP(fp)
			if err != nil {
				return nil, err
			}
			if len(buf) == 0 {
				return object.None, nil
			}

			res, err := parseMOBytes(buf)
			if err != nil {
				return nil, err
			}

			cat := object.NewDict()
			for orig, trans := range res.catalog {
				cat.Set(&object.Str{V: orig}, &object.Str{V: trans})
			}
			inst.Dict.SetStr("_catalog", cat)

			if res.charset != "" {
				inst.Dict.SetStr("_charset", &object.Str{V: res.charset})
			}
			infod := object.NewDict()
			for k, v := range res.info {
				infod.Set(&object.Str{V: k}, &object.Str{V: v})
			}
			inst.Dict.SetStr("_info", infod)
			inst.Dict.SetStr("_plural_expr", &object.Str{V: res.pluralExpr})
			inst.Dict.SetStr("_nplurals", object.NewInt(res.nplurals))

			return object.None, nil
		}})

	m.Dict.SetStr("GNUTranslations", gnuTransCls)
	m.Dict.SetStr("Catalog", gnuTransCls)

	// ── Module-level functions ────────────────────────────────────────────────

	m.Dict.SetStr("gettext", &object.BuiltinFunc{Name: "gettext",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			return a[0], nil
		}})
	m.Dict.SetStr("ngettext", &object.BuiltinFunc{Name: "ngettext",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return object.None, nil
			}
			n, _ := toInt64(a[2])
			if n == 1 {
				return a[0], nil
			}
			return a[1], nil
		}})
	m.Dict.SetStr("pgettext", &object.BuiltinFunc{Name: "pgettext",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			return a[1], nil
		}})
	m.Dict.SetStr("npgettext", &object.BuiltinFunc{Name: "npgettext",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 4 {
				return object.None, nil
			}
			n, _ := toInt64(a[3])
			if n == 1 {
				return a[1], nil
			}
			return a[2], nil
		}})
	m.Dict.SetStr("dgettext", &object.BuiltinFunc{Name: "dgettext",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			return a[1], nil
		}})
	m.Dict.SetStr("dngettext", &object.BuiltinFunc{Name: "dngettext",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 4 {
				return object.None, nil
			}
			n, _ := toInt64(a[3])
			if n == 1 {
				return a[1], nil
			}
			return a[2], nil
		}})
	m.Dict.SetStr("dpgettext", &object.BuiltinFunc{Name: "dpgettext",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return object.None, nil
			}
			return a[2], nil
		}})
	m.Dict.SetStr("dnpgettext", &object.BuiltinFunc{Name: "dnpgettext",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 5 {
				return object.None, nil
			}
			n, _ := toInt64(a[4])
			if n == 1 {
				return a[2], nil
			}
			return a[3], nil
		}})

	m.Dict.SetStr("textdomain", &object.BuiltinFunc{Name: "textdomain",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			var arg object.Object
			if len(a) >= 1 {
				arg = a[0]
			} else if kw != nil {
				arg, _ = kw.GetStr("domain")
			}
			if arg != nil {
				if _, isNone := arg.(*object.NoneType); !isNone {
					if s, ok := arg.(*object.Str); ok {
						currentDomain = s.V
					}
				}
			}
			return &object.Str{V: currentDomain}, nil
		}})

	m.Dict.SetStr("bindtextdomain", &object.BuiltinFunc{Name: "bindtextdomain",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			domain, ok := a[0].(*object.Str)
			if !ok {
				return object.None, nil
			}
			if len(a) >= 2 {
				if dir, ok := a[1].(*object.Str); ok {
					localeDirs[domain.V] = dir.V
				}
			}
			if dir, ok := localeDirs[domain.V]; ok {
				return &object.Str{V: dir}, nil
			}
			return object.None, nil
		}})

	m.Dict.SetStr("bind_textdomain_codeset", &object.BuiltinFunc{Name: "bind_textdomain_codeset",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	m.Dict.SetStr("find", &object.BuiltinFunc{Name: "find",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// allFlag = True returns list, else single path or None
			allFlag := false
			if kw != nil {
				if v, ok := kw.GetStr("all"); ok {
					allFlag = object.Truthy(v)
				}
			}
			if allFlag {
				return &object.List{V: nil}, nil
			}
			return object.None, nil
		}})

	m.Dict.SetStr("translation", &object.BuiltinFunc{Name: "translation",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// If fallback=True, return a NullTranslations
			fallback := false
			if kw != nil {
				if v, ok := kw.GetStr("fallback"); ok {
					fallback = object.Truthy(v)
				}
			}
			if fallback {
				inst := &object.Instance{Class: nullTransCls, Dict: object.NewDict()}
				cat := object.NewDict()
				inst.Dict.SetStr("_catalog", cat)
				inst.Dict.SetStr("_info", object.NewDict())
				inst.Dict.SetStr("_charset", object.None)
				inst.Dict.SetStr("_fallback", object.None)
				inst.Dict.SetStr("_plural_expr", &object.Str{V: "(n != 1)"})
				inst.Dict.SetStr("_nplurals", object.NewInt(2))
				return inst, nil
			}
			return nil, object.Errorf(i.osErr, "No translation file found")
		}})

	m.Dict.SetStr("install", &object.BuiltinFunc{Name: "install",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			// Install passthrough _ in builtins
			i.Builtins.SetStr("_", &object.BuiltinFunc{Name: "_",
				Call: func(_ any, inner []object.Object, _ *object.Dict) (object.Object, error) {
					if len(inner) >= 1 {
						return inner[0], nil
					}
					return object.None, nil
				}})
			return object.None, nil
		}})

	m.Dict.SetStr("c2py", &object.BuiltinFunc{Name: "c2py",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			var expr string
			if len(a) >= 1 {
				if s, ok := a[0].(*object.Str); ok {
					expr = s.V
				}
			}
			capturedExpr := expr
			return &object.BuiltinFunc{Name: "plural",
				Call: func(_ any, inner []object.Object, _ *object.Dict) (object.Object, error) {
					if len(inner) < 1 {
						return object.NewInt(0), nil
					}
					n, _ := toInt64(inner[0])
					return object.NewInt(evalPluralExpr(capturedExpr, n)), nil
				}}, nil
		}})

	return m
}

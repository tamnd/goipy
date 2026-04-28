package vm

import (
	"strings"

	"github.com/tamnd/goipy/object"
)

// buildTabnanny constructs the tabnanny module with the CPython 3.14 API:
// NannyNag exception, process_tokens(), check(), and module attributes.
func (i *Interp) buildTabnanny() *object.Module {
	m := &object.Module{Name: "tabnanny", Dict: object.NewDict()}

	// ── Module attributes ─────────────────────────────────────────────────

	m.Dict.SetStr("filename_only", object.NewInt(0))
	m.Dict.SetStr("verbose", object.NewInt(0))
	m.Dict.SetStr("__version__", &object.Str{V: "6"})

	allList := &object.List{V: []object.Object{
		&object.Str{V: "check"},
		&object.Str{V: "NannyNag"},
		&object.Str{V: "process_tokens"},
	}}
	m.Dict.SetStr("__all__", allList)

	// ── Whitespace class (internal helper, exposed publicly) ──────────────

	wsCls := &object.Class{Name: "Whitespace", Dict: object.NewDict()}
	wsCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			self.Dict.SetStr("indent", a[1])
			return object.None, nil
		},
	})
	wsCls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "Whitespace()"}, nil
		},
	})
	m.Dict.SetStr("Whitespace", wsCls)

	// ── NannyNag exception ────────────────────────────────────────────────

	nagCls := &object.Class{
		Name:  "NannyNag",
		Dict:  object.NewDict(),
		Bases: []*object.Class{i.exception},
	}

	nagCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			if len(a) > 1 {
				self.Dict.SetStr("_lineno", a[1])
			} else {
				self.Dict.SetStr("_lineno", object.NewInt(0))
			}
			if len(a) > 2 {
				self.Dict.SetStr("_msg", a[2])
			} else {
				self.Dict.SetStr("_msg", &object.Str{V: ""})
			}
			if len(a) > 3 {
				self.Dict.SetStr("_line", a[3])
			} else {
				self.Dict.SetStr("_line", &object.Str{V: ""})
			}
			return object.None, nil
		},
	})

	// Exception subclasses produce *object.Exception whose args are stored in
	// exc.Args. NannyNag(lineno, msg, line) → args[0]=lineno, [1]=msg, [2]=line.
	nagArg := func(a []object.Object, idx int, def object.Object) object.Object {
		if len(a) < 1 {
			return def
		}
		// self may be *object.Exception (from exception subclass fast path)
		if exc, ok := a[0].(*object.Exception); ok {
			if exc.Args != nil && idx < len(exc.Args.V) {
				return exc.Args.V[idx]
			}
			return def
		}
		// fallback: *object.Instance with Dict fields
		if inst, ok := a[0].(*object.Instance); ok {
			keys := []string{"_lineno", "_msg", "_line"}
			if idx < len(keys) {
				if v, ok2 := inst.Dict.GetStr(keys[idx]); ok2 {
					return v
				}
			}
		}
		return def
	}

	nagCls.Dict.SetStr("get_lineno", &object.BuiltinFunc{
		Name: "get_lineno",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return nagArg(a, 0, object.NewInt(0)), nil
		},
	})

	nagCls.Dict.SetStr("get_msg", &object.BuiltinFunc{
		Name: "get_msg",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return nagArg(a, 1, &object.Str{V: ""}), nil
		},
	})

	nagCls.Dict.SetStr("get_line", &object.BuiltinFunc{
		Name: "get_line",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return nagArg(a, 2, &object.Str{V: ""}), nil
		},
	})

	nagCls.Dict.SetStr("__str__", &object.BuiltinFunc{
		Name: "__str__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			v := nagArg(a, 1, &object.Str{V: "NannyNag"})
			if sv, ok := v.(*object.Str); ok {
				return sv, nil
			}
			return &object.Str{V: object.Str_(v)}, nil
		},
	})

	m.Dict.SetStr("NannyNag", nagCls)

	// ── process_tokens(tokens) ────────────────────────────────────────────
	// Inspects INDENT/DEDENT tokens for ambiguous whitespace. In this
	// implementation we perform a simplified check: mixed tabs and spaces
	// on the same indent level raises NannyNag.

	m.Dict.SetStr("process_tokens", &object.BuiltinFunc{
		Name: "process_tokens",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := i
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(ii.typeErr, "process_tokens() missing argument")
			}
			tokens, err := iterate(ii, a[0])
			if err != nil {
				return nil, err
			}

			// Check for INDENT tokens with mixed tabs/spaces (ambiguous indent)
			// Token type INDENT=5; we inspect the string field.
			for _, tok := range tokens {
				inst, ok := tok.(*object.Instance)
				if !ok {
					continue
				}
				typObj, _ := inst.Dict.GetStr("type")
				typInt, ok := typObj.(*object.Int)
				if !ok {
					continue
				}
				if typInt.Int64() != 5 { // INDENT
					continue
				}
				strObj, _ := inst.Dict.GetStr("string")
				sv, ok := strObj.(*object.Str)
				if !ok {
					continue
				}
				indent := sv.V
				// Ambiguous: contains both tabs and spaces
				hasTabs := strings.ContainsRune(indent, '\t')
				hasSpaces := strings.ContainsAny(indent, " ")
				if hasTabs && hasSpaces {
					linenoObj, _ := inst.Dict.GetStr("start")
					lineno := int64(0)
					if tup, ok := linenoObj.(*object.Tuple); ok && len(tup.V) >= 1 {
						if li, ok := tup.V[0].(*object.Int); ok {
							lineno = li.Int64()
						}
					}
					lineObj, _ := inst.Dict.GetStr("line")
					lineStr := ""
					if ls, ok := lineObj.(*object.Str); ok {
						lineStr = ls.V
					}
					exc := &object.Exception{
						Class: nagCls,
						Args: &object.Tuple{V: []object.Object{
							object.NewInt(lineno),
							&object.Str{V: "inconsistent use of tabs and spaces in indentation"},
							&object.Str{V: lineStr},
						}},
						Msg: "inconsistent use of tabs and spaces in indentation",
					}
					return nil, exc
				}
			}
			return object.None, nil
		},
	})

	// ── check(file_or_dir) ────────────────────────────────────────────────

	m.Dict.SetStr("check", &object.BuiltinFunc{
		Name: "check",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "check() missing argument")
			}
			// Stub: accept the argument, return None
			return object.None, nil
		},
	})

	// ── format_witnesses(witnesses) ───────────────────────────────────────

	m.Dict.SetStr("format_witnesses", &object.BuiltinFunc{
		Name: "format_witnesses",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			// Return a comma-separated string of witness representations
			items, err := iterate(i, a[0])
			if err != nil {
				return &object.Str{V: ""}, nil
			}
			parts := make([]string, 0, len(items))
			for _, item := range items {
				parts = append(parts, object.Str_(item))
			}
			return &object.Str{V: strings.Join(parts, ", ")}, nil
		},
	})

	// ── errprint(*args) ───────────────────────────────────────────────────

	m.Dict.SetStr("errprint", &object.BuiltinFunc{
		Name: "errprint",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			_ = a
			return object.None, nil
		},
	})

	return m
}

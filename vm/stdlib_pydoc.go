package vm

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildPydoc() *object.Module {
	m := &object.Module{Name: "pydoc", Dict: object.NewDict()}

	// ── ErrorDuringImport exception ───────────────────────────────────────────
	// Exception subclasses in goipy are *object.Exception, not *object.Instance.
	errorDuringImport := &object.Class{
		Name:  "ErrorDuringImport",
		Bases: []*object.Class{i.exception},
		Dict:  object.NewDict(),
	}
	errorDuringImport.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			exc, ok := a[0].(*object.Exception)
			if !ok {
				return object.None, nil
			}
			filename := ""
			if len(a) >= 2 {
				if s, ok2 := a[1].(*object.Str); ok2 {
					filename = s.V
				}
			}
			msg := fmt.Sprintf("problem in %s", filename)
			exc.Msg = msg
			exc.Args = &object.Tuple{V: []object.Object{&object.Str{V: msg}}}
			if exc.Dict == nil {
				exc.Dict = object.NewDict()
			}
			exc.Dict.SetStr("filename", &object.Str{V: filename})
			return object.None, nil
		},
	})
	m.Dict.SetStr("ErrorDuringImport", errorDuringImport)

	// ── getdoc ────────────────────────────────────────────────────────────────
	getdoc := &object.BuiltinFunc{
		Name: "getdoc",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			obj := a[0]
			doc := pydocGetDocStr(obj)
			return &object.Str{V: pydocCleanDoc(doc)}, nil
		},
	}
	m.Dict.SetStr("getdoc", getdoc)

	// ── describe ──────────────────────────────────────────────────────────────
	m.Dict.SetStr("describe", &object.BuiltinFunc{
		Name: "describe",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "object"}, nil
			}
			return &object.Str{V: pydocDescribe(a[0])}, nil
		},
	})

	// ── splitdoc ──────────────────────────────────────────────────────────────
	m.Dict.SetStr("splitdoc", &object.BuiltinFunc{
		Name: "splitdoc",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			doc := ""
			if len(a) >= 1 {
				if s, ok := a[0].(*object.Str); ok {
					doc = s.V
				}
			}
			synopsis, body := pydocSplitDoc(doc)
			return &object.Tuple{V: []object.Object{
				&object.Str{V: synopsis},
				&object.Str{V: body},
			}}, nil
		},
	})

	// ── plain ─────────────────────────────────────────────────────────────────
	rePlain := regexp.MustCompile(`.\x08`)
	m.Dict.SetStr("plain", &object.BuiltinFunc{
		Name: "plain",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return &object.Str{V: ""}, nil
			}
			return &object.Str{V: rePlain.ReplaceAllString(s.V, "")}, nil
		},
	})

	// ── stripid ───────────────────────────────────────────────────────────────
	// Match " at 0x<lowercase-hex>{6,14}>" and replace with ">" to strip address.
	reStripID := regexp.MustCompile(` at 0x[0-9a-f]{6,14}>`)
	m.Dict.SetStr("stripid", &object.BuiltinFunc{
		Name: "stripid",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return &object.Str{V: ""}, nil
			}
			result := reStripID.ReplaceAllString(s.V, ">")
			return &object.Str{V: result}, nil
		},
	})

	// ── replace ───────────────────────────────────────────────────────────────
	m.Dict.SetStr("replace", &object.BuiltinFunc{
		Name: "replace",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return &object.Str{V: ""}, nil
			}
			result := s.V
			// Process pairs
			for idx := 1; idx+1 < len(a); idx += 2 {
				old, ok1 := a[idx].(*object.Str)
				new_, ok2 := a[idx+1].(*object.Str)
				if ok1 && ok2 {
					result = strings.ReplaceAll(result, old.V, new_.V)
				}
			}
			return &object.Str{V: result}, nil
		},
	})

	// ── isdata ────────────────────────────────────────────────────────────────
	m.Dict.SetStr("isdata", &object.BuiltinFunc{
		Name: "isdata",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.BoolOf(false), nil
			}
			return object.BoolOf(pydocIsData(a[0])), nil
		},
	})

	// ── visiblename ───────────────────────────────────────────────────────────
	// Dunder names excluded from visible set (matches CPython pydoc._is_bound_method skip list)
	pydocExcludedDunders := map[string]bool{
		"__all__": true, "__builtins__": true, "__cached__": true,
		"__doc__": true, "__file__": true, "__loader__": true,
		"__name__": true, "__package__": true, "__path__": true,
		"__spec__": true,
	}
	m.Dict.SetStr("visiblename", &object.BuiltinFunc{
		Name: "visiblename",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.BoolOf(false), nil
			}
			nameStr, ok := a[0].(*object.Str)
			if !ok {
				return object.BoolOf(false), nil
			}
			name := nameStr.V

			// Collect all/extra arg
			var allNames []string
			if len(a) >= 2 {
				switch v := a[1].(type) {
				case *object.List:
					for _, item := range v.V {
						if s, ok2 := item.(*object.Str); ok2 {
							allNames = append(allNames, s.V)
						}
					}
				case *object.Tuple:
					for _, item := range v.V {
						if s, ok2 := item.(*object.Str); ok2 {
							allNames = append(allNames, s.V)
						}
					}
				}
			}
			if kw != nil {
				if allArg, ok2 := kw.GetStr("all"); ok2 {
					switch v := allArg.(type) {
					case *object.List:
						for _, item := range v.V {
							if s, ok3 := item.(*object.Str); ok3 {
								allNames = append(allNames, s.V)
							}
						}
					}
				}
			}

			if pydocExcludedDunders[name] {
				return object.BoolOf(false), nil
			}
			// Private names starting with underscore
			if strings.HasPrefix(name, "_") && !strings.HasPrefix(name, "__") {
				// Only visible if listed in all
				for _, n := range allNames {
					if n == name {
						return object.BoolOf(true), nil
					}
				}
				return object.BoolOf(false), nil
			}
			// Dunder names that are not in the excluded set
			if strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__") {
				return object.NewInt(1), nil // CPython returns 1 (truthy int) for dunders
			}
			return object.BoolOf(true), nil
		},
	})

	// ── ispath ────────────────────────────────────────────────────────────────
	m.Dict.SetStr("ispath", &object.BuiltinFunc{
		Name: "ispath",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.BoolOf(false), nil
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return object.BoolOf(false), nil
			}
			return object.BoolOf(strings.ContainsRune(s.V, '/')), nil
		},
	})

	// ── cram ─────────────────────────────────────────────────────────────────
	m.Dict.SetStr("cram", &object.BuiltinFunc{
		Name: "cram",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				if len(a) == 1 {
					if s, ok := a[0].(*object.Str); ok {
						return s, nil
					}
				}
				return &object.Str{V: ""}, nil
			}
			text := ""
			if s, ok := a[0].(*object.Str); ok {
				text = s.V
			}
			maxlen := 0
			if n, ok := a[1].(*object.Int); ok {
				maxlen = int(n.V.Int64())
			}
			if len(text) <= maxlen {
				return &object.Str{V: text}, nil
			}
			pre := 0
			if maxlen-3 > 0 {
				pre = (maxlen - 3) / 2
			}
			suffix := 0
			if maxlen-3-pre > 0 {
				suffix = maxlen - 3 - pre
			}
			result := text[:pre] + "..." + text[len(text)-suffix:]
			return &object.Str{V: result}, nil
		},
	})

	// ── Repr class ────────────────────────────────────────────────────────────
	reprCls := &object.Class{Name: "Repr", Dict: object.NewDict()}
	reprCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("maxlevel", object.NewInt(6))
			inst.Dict.SetStr("maxstring", object.NewInt(30))
			inst.Dict.SetStr("maxother", object.NewInt(30))
			inst.Dict.SetStr("maxlist", object.NewInt(6))
			inst.Dict.SetStr("maxtuple", object.NewInt(6))
			inst.Dict.SetStr("maxset", object.NewInt(6))
			inst.Dict.SetStr("maxfrozenset", object.NewInt(6))
			inst.Dict.SetStr("maxdict", object.NewInt(4))
			inst.Dict.SetStr("maxlong", object.NewInt(40))
			return object.None, nil
		},
	})
	reprCls.Dict.SetStr("repr", &object.BuiltinFunc{
		Name: "repr",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return &object.Str{V: ""}, nil
			}
			inst := a[0].(*object.Instance)
			obj := a[1]
			maxStr := 30
			if v, ok := inst.Dict.GetStr("maxstring"); ok {
				if n, ok2 := v.(*object.Int); ok2 {
					maxStr = int(n.V.Int64())
				}
			}
			maxOther := 30
			if v, ok := inst.Dict.GetStr("maxother"); ok {
				if n, ok2 := v.(*object.Int); ok2 {
					maxOther = int(n.V.Int64())
				}
			}
			r := object.Repr(obj)
			limit := maxOther
			if _, ok := obj.(*object.Str); ok {
				limit = maxStr + 2 // account for quotes
			}
			if len(r) > limit {
				r = r[:limit-3] + "..."
			}
			return &object.Str{V: r}, nil
		},
	})
	m.Dict.SetStr("Repr", reprCls)

	// ── HTMLRepr class (alias/stub) ───────────────────────────────────────────
	htmlReprCls := &object.Class{Name: "HTMLRepr", Bases: []*object.Class{reprCls}, Dict: object.NewDict()}
	m.Dict.SetStr("HTMLRepr", htmlReprCls)

	// ── Doc base class ────────────────────────────────────────────────────────
	docCls := &object.Class{Name: "Doc", Dict: object.NewDict()}
	m.Dict.SetStr("Doc", docCls)

	// ── TextDoc ───────────────────────────────────────────────────────────────
	textDocCls := &object.Class{Name: "TextDoc", Bases: []*object.Class{docCls}, Dict: object.NewDict()}
	textDocCls.Dict.SetStr("document", &object.BuiltinFunc{
		Name: "document",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return &object.Str{V: ""}, nil
			}
			obj := a[1]
			name := pydocObjectName(obj)
			doc := pydocGetDocStr(obj)
			result := fmt.Sprintf("%s\n\n%s", name, pydocCleanDoc(doc))
			return &object.Str{V: result}, nil
		},
	})
	m.Dict.SetStr("TextDoc", textDocCls)

	// ── _PlainTextDoc ─────────────────────────────────────────────────────────
	plainTextDocCls := &object.Class{Name: "_PlainTextDoc", Bases: []*object.Class{textDocCls}, Dict: object.NewDict()}
	m.Dict.SetStr("_PlainTextDoc", plainTextDocCls)

	// ── HTMLDoc ───────────────────────────────────────────────────────────────
	htmlDocCls := &object.Class{Name: "HTMLDoc", Bases: []*object.Class{docCls}, Dict: object.NewDict()}
	m.Dict.SetStr("HTMLDoc", htmlDocCls)

	// ── text, html, plaintext instances ──────────────────────────────────────
	textInst := &object.Instance{Class: textDocCls, Dict: object.NewDict()}
	htmlInst := &object.Instance{Class: htmlDocCls, Dict: object.NewDict()}
	plainInst := &object.Instance{Class: plainTextDocCls, Dict: object.NewDict()}
	m.Dict.SetStr("text", textInst)
	m.Dict.SetStr("html", htmlInst)
	m.Dict.SetStr("plaintext", plainInst)

	// ── render_doc ────────────────────────────────────────────────────────────
	m.Dict.SetStr("render_doc", &object.BuiltinFunc{
		Name: "render_doc",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			obj := a[0]
			name := pydocObjectName(obj)
			doc := pydocCleanDoc(pydocGetDocStr(obj))
			desc := pydocDescribe(obj)
			var sb strings.Builder
			sb.WriteString("Python Library Documentation: ")
			sb.WriteString(desc)
			sb.WriteString("\n\n")
			sb.WriteString(name)
			if doc != "" {
				sb.WriteString("\n\n")
				sb.WriteString(doc)
			}
			sb.WriteString("\n")
			return &object.Str{V: sb.String()}, nil
		},
	})

	// ── doc ───────────────────────────────────────────────────────────────────
	m.Dict.SetStr("doc", &object.BuiltinFunc{
		Name: "doc",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			obj := a[0]
			name := pydocObjectName(obj)
			doc := pydocCleanDoc(pydocGetDocStr(obj))
			desc := pydocDescribe(obj)
			var sb strings.Builder
			sb.WriteString("Python Library Documentation: ")
			sb.WriteString(desc)
			sb.WriteString("\n\n")
			sb.WriteString(name)
			if doc != "" {
				sb.WriteString("\n\n")
				sb.WriteString(doc)
			}
			sb.WriteString("\n")
			text := sb.String()

			// Check output kwarg
			var output object.Object
			if kw != nil {
				if v, ok2 := kw.GetStr("output"); ok2 {
					output = v
				}
			}
			if len(a) >= 3 {
				output = a[2]
			}

			if output != nil && output != object.None {
				// Write to output (StringIO or similar)
				writeFn, err := i.getAttr(output, "write")
				if err == nil {
					_, _ = i.callObject(writeFn, []object.Object{&object.Str{V: text}}, nil)
				}
				return object.None, nil
			}
			// Write to stdout
			fmt.Fprint(i.Stdout, text)
			return object.None, nil
		},
	})

	// ── Helper class ──────────────────────────────────────────────────────────
	helperCls := &object.Class{Name: "Helper", Dict: object.NewDict()}
	helperCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	helperCls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return &object.Str{V: "<pydoc.Helper instance>"}, nil
		},
	})
	helperCls.Dict.SetStr("__call__", &object.BuiltinFunc{
		Name: "__call__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	m.Dict.SetStr("Helper", helperCls)

	// pydoc.help is a Helper instance
	helpInst := &object.Instance{Class: helperCls, Dict: object.NewDict()}
	m.Dict.SetStr("help", helpInst)

	// ── locate ────────────────────────────────────────────────────────────────
	m.Dict.SetStr("locate", &object.BuiltinFunc{
		Name: "locate",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			path, ok := a[0].(*object.Str)
			if !ok {
				return object.None, nil
			}
			obj, err := pydocLocate(i, path.V)
			if err != nil || obj == nil {
				return object.None, nil
			}
			return obj, nil
		},
	})

	// ── safeimport ────────────────────────────────────────────────────────────
	m.Dict.SetStr("safeimport", &object.BuiltinFunc{
		Name: "safeimport",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			path, ok := a[0].(*object.Str)
			if !ok {
				return object.None, nil
			}
			mod, err := i.loadModule(path.V)
			if err != nil {
				return object.None, nil
			}
			return mod, nil
		},
	})

	// ── isdata ── (already set above) ─────────────────────────────────────────
	// ── ispackage stub ────────────────────────────────────────────────────────
	m.Dict.SetStr("ispackage", &object.BuiltinFunc{
		Name: "ispackage",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.BoolOf(false), nil
		},
	})

	// ── allmethods ────────────────────────────────────────────────────────────
	m.Dict.SetStr("allmethods", &object.BuiltinFunc{
		Name: "allmethods",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NewDict(), nil
			}
			cls, ok := a[0].(*object.Class)
			if !ok {
				return object.NewDict(), nil
			}
			result := object.NewDict()
			pydocCollectMethods(cls, result)
			return result, nil
		},
	})

	// ── sort_attributes ───────────────────────────────────────────────────────
	m.Dict.SetStr("sort_attributes", &object.BuiltinFunc{
		Name: "sort_attributes",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// No-op stub — just return None
			return object.None, nil
		},
	})

	// ── classify_class_attrs ──────────────────────────────────────────────────
	m.Dict.SetStr("classify_class_attrs", &object.BuiltinFunc{
		Name: "classify_class_attrs",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return &object.List{V: nil}, nil
		},
	})

	// ── pathdirs ──────────────────────────────────────────────────────────────
	m.Dict.SetStr("pathdirs", &object.BuiltinFunc{
		Name: "pathdirs",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return &object.List{V: nil}, nil
		},
	})

	// ── pager / getpager / get_pager stubs ───────────────────────────────────
	noopPager := &object.BuiltinFunc{
		Name: "pager",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	}
	m.Dict.SetStr("pager", noopPager)
	m.Dict.SetStr("getpager", &object.BuiltinFunc{
		Name: "getpager",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return noopPager, nil
		},
	})
	m.Dict.SetStr("get_pager", &object.BuiltinFunc{
		Name: "get_pager",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return noopPager, nil
		},
	})

	// ── format_exception_only ─────────────────────────────────────────────────
	m.Dict.SetStr("format_exception_only", &object.BuiltinFunc{
		Name: "format_exception_only",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			typeName := "Exception"
			msg := ""
			if len(a) >= 1 {
				if cls, ok := a[0].(*object.Class); ok {
					typeName = cls.Name
				}
			}
			if len(a) >= 2 {
				msg = object.Str_(a[1])
			}
			line := typeName + ": " + msg + "\n"
			return &object.List{V: []object.Object{&object.Str{V: line}}}, nil
		},
	})

	// ── resolve ───────────────────────────────────────────────────────────────
	m.Dict.SetStr("resolve", &object.BuiltinFunc{
		Name: "resolve",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Tuple{V: []object.Object{object.None, &object.Str{V: ""}}}, nil
			}
			obj := a[0]
			if path, ok := obj.(*object.Str); ok {
				// Locate by path
				located, err := pydocLocate(i, path.V)
				if err == nil && located != nil {
					return &object.Tuple{V: []object.Object{located, &object.Str{V: path.V}}}, nil
				}
				return &object.Tuple{V: []object.Object{object.None, &object.Str{V: path.V}}}, nil
			}
			name := pydocObjectName(obj)
			return &object.Tuple{V: []object.Object{obj, &object.Str{V: name}}}, nil
		},
	})

	// ── visiblename is already set above ─────────────────────────────────────
	// ── ispath is already set above ──────────────────────────────────────────

	// ── apropos stub ─────────────────────────────────────────────────────────
	m.Dict.SetStr("apropos", &object.BuiltinFunc{
		Name: "apropos",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── writedoc / writedocs stubs ────────────────────────────────────────────
	m.Dict.SetStr("writedoc", &object.BuiltinFunc{
		Name: "writedoc",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	m.Dict.SetStr("writedocs", &object.BuiltinFunc{
		Name: "writedocs",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	return m
}

// ── Helper functions ──────────────────────────────────────────────────────────

func pydocGetDocStr(obj object.Object) string {
	switch v := obj.(type) {
	case *object.Function:
		if v.Doc != nil && v.Doc != object.None {
			if s, ok := v.Doc.(*object.Str); ok {
				return s.V
			}
		}
		// Fallback: check Dict
		if v.Dict != nil {
			if d, ok := v.Dict.GetStr("__doc__"); ok {
				if s, ok2 := d.(*object.Str); ok2 {
					return s.V
				}
			}
		}
	case *object.Class:
		if d, ok := v.Dict.GetStr("__doc__"); ok {
			if s, ok2 := d.(*object.Str); ok2 {
				return s.V
			}
		}
	case *object.Instance:
		// Use class docstring
		if d, ok := classLookup(v.Class, "__doc__"); ok {
			if s, ok2 := d.(*object.Str); ok2 {
				return s.V
			}
		}
	case *object.Module:
		if d, ok := v.Dict.GetStr("__doc__"); ok {
			if s, ok2 := d.(*object.Str); ok2 {
				return s.V
			}
		}
	case *object.BuiltinFunc:
		if v.Attrs != nil {
			if d, ok := v.Attrs.GetStr("__doc__"); ok {
				if s, ok2 := d.(*object.Str); ok2 {
					return s.V
				}
			}
		}
	}
	return ""
}

// pydocCleanDoc strips common leading whitespace from a docstring.
func pydocCleanDoc(doc string) string {
	if doc == "" {
		return ""
	}
	lines := strings.Split(doc, "\n")
	if len(lines) == 1 {
		return strings.TrimSpace(lines[0])
	}
	// Find minimum indent of non-empty lines after first
	minIndent := -1
	for _, line := range lines[1:] {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(trimmed)
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}
	var out []string
	out = append(out, strings.TrimSpace(lines[0]))
	for _, line := range lines[1:] {
		if minIndent > 0 && len(line) >= minIndent {
			out = append(out, line[minIndent:])
		} else {
			out = append(out, strings.TrimRight(line, " \t"))
		}
	}
	// Strip trailing blank lines
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

// pydocSplitDoc splits a docstring into (synopsis, body).
func pydocSplitDoc(doc string) (string, string) {
	stripped := strings.TrimSpace(doc)
	if stripped == "" {
		return "", ""
	}
	lines := strings.Split(stripped, "\n")
	if len(lines) == 1 {
		return strings.TrimSpace(lines[0]), ""
	}
	if len(lines) >= 2 && strings.TrimSpace(lines[1]) == "" {
		rest := strings.TrimSpace(strings.Join(lines[2:], "\n"))
		return strings.TrimSpace(lines[0]), rest
	}
	return "", stripped
}

// pydocDescribe returns a human-readable description of an object.
func pydocDescribe(obj object.Object) string {
	switch v := obj.(type) {
	case *object.Function:
		return "function " + v.Name
	case *object.BuiltinFunc:
		return "built-in function " + v.Name
	case *object.Class:
		return "class " + v.Name
	case *object.Module:
		return "module " + v.Name
	case *object.Instance:
		return v.Class.Name
	case *object.NoneType:
		return "NoneType"
	case *object.Bool:
		return "bool"
	case *object.Int:
		return "int"
	case *object.Float:
		return "float"
	case *object.Str:
		return "str"
	case *object.Bytes:
		return "bytes"
	case *object.List:
		return "list"
	case *object.Dict:
		return "dict"
	case *object.Tuple:
		return "tuple"
	case *object.Set:
		return "set"
	case *object.Frozenset:
		return "frozenset"
	}
	return object.TypeName(obj)
}

// pydocObjectName returns the name used to identify an object in docs.
func pydocObjectName(obj object.Object) string {
	switch v := obj.(type) {
	case *object.Function:
		return v.Name
	case *object.BuiltinFunc:
		return v.Name
	case *object.Class:
		return v.Name
	case *object.Module:
		return v.Name
	}
	return object.TypeName(obj)
}

// pydocIsData returns true iff object is a data value (not callable/module/class).
func pydocIsData(obj object.Object) bool {
	switch obj.(type) {
	case *object.Function, *object.BuiltinFunc, *object.Class, *object.Module,
		*object.BoundMethod:
		return false
	}
	return true
}

// pydocLocate walks a dotted path, importing modules and getting attributes.
func pydocLocate(i *Interp, path string) (object.Object, error) {
	parts := strings.SplitN(path, ".", 2)
	mod, err := i.loadModule(parts[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 1 {
		return mod, nil
	}
	// Walk remaining attrs
	var cur object.Object = mod
	for _, part := range strings.Split(parts[1], ".") {
		next, err2 := i.getAttr(cur, part)
		if err2 != nil {
			return nil, err2
		}
		cur = next
	}
	return cur, nil
}

// pydocCollectMethods collects all methods from a class and its bases.
func pydocCollectMethods(cls *object.Class, result *object.Dict) {
	for _, base := range cls.Bases {
		pydocCollectMethods(base, result)
	}
	keys, vals := cls.Dict.Items()
	for idx, k := range keys {
		switch vals[idx].(type) {
		case *object.Function, *object.BuiltinFunc:
			result.Set(k, vals[idx])
		}
	}
}

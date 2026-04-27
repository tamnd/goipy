package vm

import (
	"fmt"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildDoctest() *object.Module {
	m := &object.Module{Name: "doctest", Dict: object.NewDict()}

	// ── Option flag constants ──────────────────────────────────────────────────
	m.Dict.SetStr("DONT_ACCEPT_TRUE_FOR_1", object.NewInt(1))
	m.Dict.SetStr("DONT_ACCEPT_BLANKLINE", object.NewInt(2))
	m.Dict.SetStr("NORMALIZE_WHITESPACE", object.NewInt(4))
	m.Dict.SetStr("ELLIPSIS", object.NewInt(8))
	m.Dict.SetStr("SKIP", object.NewInt(16))
	m.Dict.SetStr("IGNORE_EXCEPTION_DETAIL", object.NewInt(32))
	m.Dict.SetStr("REPORT_UDIFF", object.NewInt(64))
	m.Dict.SetStr("REPORT_CDIFF", object.NewInt(128))
	m.Dict.SetStr("REPORT_NDIFF", object.NewInt(256))
	m.Dict.SetStr("REPORT_ONLY_FIRST_FAILURE", object.NewInt(512))
	m.Dict.SetStr("FAIL_FAST", object.NewInt(1024))
	m.Dict.SetStr("BLANKLINE_MARKER", &object.Str{V: "<BLANKLINE>"})
	m.Dict.SetStr("ELLIPSIS_MARKER", &object.Str{V: "..."})

	// ── register_optionflag ────────────────────────────────────────────────────
	optionFlagRegistry := map[string]int64{}
	var nextFlagBit int64 = 2048
	m.Dict.SetStr("register_optionflag", &object.BuiltinFunc{
		Name: "register_optionflag",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "register_optionflag() requires a name")
			}
			name := ""
			if s, ok := a[0].(*object.Str); ok {
				name = s.V
			}
			if v, ok := optionFlagRegistry[name]; ok {
				return object.NewInt(v), nil
			}
			v := nextFlagBit
			nextFlagBit <<= 1
			optionFlagRegistry[name] = v
			return object.NewInt(v), nil
		},
	})

	// ── Example class ──────────────────────────────────────────────────────────
	exampleCls := &object.Class{Name: "Example", Dict: object.NewDict()}
	exampleCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, object.Errorf(i.typeErr, "Example() requires source and want")
			}
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("source", a[1])
			inst.Dict.SetStr("want", a[2])
			var lineno object.Object = object.NewInt(0)
			var indent object.Object = object.NewInt(0)
			var options object.Object = object.NewDict()
			if len(a) >= 4 {
				lineno = a[3]
			}
			if len(a) >= 5 {
				indent = a[4]
			}
			if kw != nil {
				if v, ok := kw.GetStr("lineno"); ok {
					lineno = v
				}
				if v, ok := kw.GetStr("indent"); ok {
					indent = v
				}
				if v, ok := kw.GetStr("options"); ok {
					options = v
				}
			}
			inst.Dict.SetStr("lineno", lineno)
			inst.Dict.SetStr("indent", indent)
			inst.Dict.SetStr("options", options)
			return object.None, nil
		},
	})
	exampleCls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			srcStr := ""
			if v, ok := inst.Dict.GetStr("source"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					srcStr = s.V
				}
			}
			wantStr := ""
			if v, ok := inst.Dict.GetStr("want"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					wantStr = s.V
				}
			}
			return &object.Str{V: fmt.Sprintf("<Example %q => %q>", srcStr, wantStr)}, nil
		},
	})
	m.Dict.SetStr("Example", exampleCls)

	// ── DocTest class ──────────────────────────────────────────────────────────
	docTestCls := &object.Class{Name: "DocTest", Dict: object.NewDict()}
	docTestCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 7 {
				return nil, object.Errorf(i.typeErr, "DocTest() requires examples, globs, name, filename, lineno, docstring")
			}
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("examples", a[1])
			inst.Dict.SetStr("globs", a[2])
			inst.Dict.SetStr("name", a[3])
			inst.Dict.SetStr("filename", a[4])
			inst.Dict.SetStr("lineno", a[5])
			inst.Dict.SetStr("docstring", a[6])
			return object.None, nil
		},
	})
	docTestCls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			name := ""
			filename := ""
			var lineno int64
			numEx := 0
			if v, ok := inst.Dict.GetStr("name"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					name = s.V
				}
			}
			if v, ok := inst.Dict.GetStr("filename"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					filename = s.V
				}
			}
			if v, ok := inst.Dict.GetStr("lineno"); ok {
				if n, ok2 := toInt64(v); ok2 {
					lineno = n
				}
			}
			if v, ok := inst.Dict.GetStr("examples"); ok {
				if lst, ok2 := v.(*object.List); ok2 {
					numEx = len(lst.V)
				}
			}
			return &object.Str{V: fmt.Sprintf("<DocTest %s from %s:%d (%d examples)>", name, filename, lineno, numEx)}, nil
		},
	})
	m.Dict.SetStr("DocTest", docTestCls)

	// parseExamples parses a docstring text into Example instances.
	parseExamples := func(text string) []*object.Instance {
		var results []*object.Instance
		lines := strings.Split(text, "\n")
		n := len(lines)
		idx := 0
		for idx < n {
			line := lines[idx]
			stripped := strings.TrimLeft(line, " \t")
			if !strings.HasPrefix(stripped, ">>> ") && stripped != ">>>" {
				idx++
				continue
			}
			indent := len(line) - len(stripped)
			lineNo := idx
			var srcParts []string
			for idx < n {
				l := lines[idx]
				t := strings.TrimLeft(l, " \t")
				if strings.HasPrefix(t, ">>> ") {
					srcParts = append(srcParts, strings.TrimPrefix(t, ">>> "))
					idx++
				} else if t == ">>>" {
					srcParts = append(srcParts, "")
					idx++
				} else if strings.HasPrefix(t, "... ") {
					srcParts = append(srcParts, strings.TrimPrefix(t, "... "))
					idx++
				} else if t == "..." {
					srcParts = append(srcParts, "")
					idx++
				} else {
					break
				}
			}
			var wantParts []string
			for idx < n {
				l := lines[idx]
				t := strings.TrimSpace(l)
				if t == "" {
					break
				}
				if strings.HasPrefix(t, ">>> ") || t == ">>>" {
					break
				}
				wantParts = append(wantParts, t)
				idx++
			}
			src := strings.Join(srcParts, "\n") + "\n"
			want := ""
			if len(wantParts) > 0 {
				want = strings.Join(wantParts, "\n") + "\n"
			}
			ex := &object.Instance{Class: exampleCls, Dict: object.NewDict()}
			ex.Dict.SetStr("source", &object.Str{V: src})
			ex.Dict.SetStr("want", &object.Str{V: want})
			ex.Dict.SetStr("lineno", object.NewInt(int64(lineNo)))
			ex.Dict.SetStr("indent", object.NewInt(int64(indent)))
			ex.Dict.SetStr("options", object.NewDict())
			results = append(results, ex)
		}
		return results
	}

	// makeDocTest constructs a DocTest instance from parsed examples.
	makeDocTest := func(text string, globs *object.Dict, name, filename string, lineno int64) *object.Instance {
		exs := parseExamples(text)
		items := make([]object.Object, len(exs))
		for j, ex := range exs {
			items[j] = ex
		}
		dt := &object.Instance{Class: docTestCls, Dict: object.NewDict()}
		dt.Dict.SetStr("examples", &object.List{V: items})
		dt.Dict.SetStr("globs", globs)
		dt.Dict.SetStr("name", &object.Str{V: name})
		dt.Dict.SetStr("filename", &object.Str{V: filename})
		dt.Dict.SetStr("lineno", object.NewInt(lineno))
		dt.Dict.SetStr("docstring", &object.Str{V: text})
		return dt
	}

	// ── DocTestParser class ────────────────────────────────────────────────────
	docTestParserCls := &object.Class{Name: "DocTestParser", Dict: object.NewDict()}
	docTestParserCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	docTestParserCls.Dict.SetStr("get_examples", &object.BuiltinFunc{
		Name: "get_examples",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			text := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					text = s.V
				}
			}
			exs := parseExamples(text)
			items := make([]object.Object, len(exs))
			for j, ex := range exs {
				items[j] = ex
			}
			return &object.List{V: items}, nil
		},
	})
	docTestParserCls.Dict.SetStr("parse", &object.BuiltinFunc{
		Name: "parse",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			text := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					text = s.V
				}
			}
			exs := parseExamples(text)
			// Return interleaved: empty-string, Example, empty-string, Example, ...
			items := make([]object.Object, 0, len(exs)*2+1)
			items = append(items, &object.Str{V: ""})
			for _, ex := range exs {
				items = append(items, ex)
				items = append(items, &object.Str{V: ""})
			}
			return &object.List{V: items}, nil
		},
	})
	docTestParserCls.Dict.SetStr("get_doctest", &object.BuiltinFunc{
		Name: "get_doctest",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			text := ""
			globs := object.NewDict()
			name := "<string>"
			filename := "<string>"
			var lineno int64
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					text = s.V
				}
			}
			if len(a) >= 3 {
				if d, ok := a[2].(*object.Dict); ok {
					globs = d
				}
			}
			if len(a) >= 4 {
				if s, ok := a[3].(*object.Str); ok {
					name = s.V
				}
			}
			if len(a) >= 5 {
				if s, ok := a[4].(*object.Str); ok {
					filename = s.V
				}
			}
			if len(a) >= 6 {
				if n, ok := toInt64(a[5]); ok {
					lineno = n
				}
			}
			return makeDocTest(text, globs, name, filename, lineno), nil
		},
	})
	m.Dict.SetStr("DocTestParser", docTestParserCls)

	// ── TestResults namedtuple ─────────────────────────────────────────────────
	testResultsCls := i.makeNamedTuple("TestResults", []string{"failed", "attempted"}, nil)
	m.Dict.SetStr("TestResults", testResultsCls)

	newTestResults := func(failed, attempted int64) *object.Instance {
		tr := &object.Instance{Class: testResultsCls, Dict: object.NewDict()}
		tr.Dict.SetStr("failed", object.NewInt(failed))
		tr.Dict.SetStr("attempted", object.NewInt(attempted))
		return tr
	}

	// ── OutputChecker class ────────────────────────────────────────────────────
	outputCheckerCls := &object.Class{Name: "OutputChecker", Dict: object.NewDict()}
	outputCheckerCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	outputCheckerCls.Dict.SetStr("check_output", &object.BuiltinFunc{
		Name: "check_output",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 4 {
				return object.BoolOf(false), nil
			}
			want, got := "", ""
			if s, ok := a[1].(*object.Str); ok {
				want = s.V
			}
			if s, ok := a[2].(*object.Str); ok {
				got = s.V
			}
			var flags int64
			if n, ok := toInt64(a[3]); ok {
				flags = n
			}
			return object.BoolOf(doctestCheckOutput(want, got, flags)), nil
		},
	})
	outputCheckerCls.Dict.SetStr("output_difference", &object.BuiltinFunc{
		Name: "output_difference",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			want, got := "", ""
			if len(a) >= 2 {
				if inst, ok := a[1].(*object.Instance); ok {
					if v, ok2 := inst.Dict.GetStr("want"); ok2 {
						if s, ok3 := v.(*object.Str); ok3 {
							want = s.V
						}
					}
				}
			}
			if len(a) >= 3 {
				if s, ok := a[2].(*object.Str); ok {
					got = s.V
				}
			}
			return &object.Str{V: doctestOutputDiff(want, got)}, nil
		},
	})
	m.Dict.SetStr("OutputChecker", outputCheckerCls)

	// ── DocTestRunner class ────────────────────────────────────────────────────
	docTestRunnerCls := &object.Class{Name: "DocTestRunner", Dict: object.NewDict()}
	docTestRunnerCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("_total_failed", object.NewInt(0))
			inst.Dict.SetStr("_total_attempted", object.NewInt(0))
			return object.None, nil
		},
	})
	docTestRunnerCls.Dict.SetStr("run", &object.BuiltinFunc{
		Name: "run",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			attempted := int64(0)
			if len(a) >= 2 {
				if inst, ok := a[1].(*object.Instance); ok {
					if v, ok2 := inst.Dict.GetStr("examples"); ok2 {
						if lst, ok3 := v.(*object.List); ok3 {
							attempted = int64(len(lst.V))
						}
					}
				}
			}
			if len(a) >= 1 {
				if self, ok := a[0].(*object.Instance); ok {
					prev := int64(0)
					if v, ok2 := self.Dict.GetStr("_total_attempted"); ok2 {
						if n, ok3 := toInt64(v); ok3 {
							prev = n
						}
					}
					self.Dict.SetStr("_total_attempted", object.NewInt(prev+attempted))
				}
			}
			return newTestResults(0, attempted), nil
		},
	})
	docTestRunnerCls.Dict.SetStr("summarize", &object.BuiltinFunc{
		Name: "summarize",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return newTestResults(0, 0), nil
			}
			self := a[0].(*object.Instance)
			totalAttempted, totalFailed := int64(0), int64(0)
			if v, ok := self.Dict.GetStr("_total_attempted"); ok {
				if n, ok2 := toInt64(v); ok2 {
					totalAttempted = n
				}
			}
			if v, ok := self.Dict.GetStr("_total_failed"); ok {
				if n, ok2 := toInt64(v); ok2 {
					totalFailed = n
				}
			}
			return newTestResults(totalFailed, totalAttempted), nil
		},
	})
	m.Dict.SetStr("DocTestRunner", docTestRunnerCls)

	// ── DocTestFinder class ────────────────────────────────────────────────────
	docTestFinderCls := &object.Class{Name: "DocTestFinder", Dict: object.NewDict()}
	docTestFinderCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	docTestFinderCls.Dict.SetStr("find", &object.BuiltinFunc{
		Name: "find",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return &object.List{V: []object.Object{}}, nil
			}
			obj := a[1]
			name := ""
			if kw != nil {
				if v, ok := kw.GetStr("name"); ok {
					if s, ok2 := v.(*object.Str); ok2 {
						name = s.V
					}
				}
			}
			docStr := ""
			switch o := obj.(type) {
			case *object.Function:
				if o.Doc != nil {
					if s, ok := o.Doc.(*object.Str); ok {
						docStr = s.V
					}
				}
				if name == "" {
					name = o.Name
				}
			case *object.Class:
				if v, ok := o.Dict.GetStr("__doc__"); ok {
					if s, ok2 := v.(*object.Str); ok2 {
						docStr = s.V
					}
				}
				if name == "" {
					name = o.Name
				}
			case *object.Module:
				if v, ok := o.Dict.GetStr("__doc__"); ok {
					if s, ok2 := v.(*object.Str); ok2 {
						docStr = s.V
					}
				}
				if name == "" {
					name = o.Name
				}
			}
			dt := makeDocTest(docStr, object.NewDict(), name, "<doctest>", 0)
			return &object.List{V: []object.Object{dt}}, nil
		},
	})
	m.Dict.SetStr("DocTestFinder", docTestFinderCls)

	// ── Exceptions ────────────────────────────────────────────────────────────
	// Exception subclasses in goipy don't call __init__ — callObject creates
	// *object.Exception directly and stores constructor args in exc.Args.
	// Use __getattr__ to serve the named fields from positional Args.
	docTestFailureCls := &object.Class{
		Name:  "DocTestFailure",
		Bases: []*object.Class{i.exception},
		Dict:  object.NewDict(),
	}
	docTestFailureCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{
		Name: "__getattr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			exc := a[0].(*object.Exception)
			name := ""
			if s, ok := a[1].(*object.Str); ok {
				name = s.V
			}
			if exc.Args != nil {
				switch name {
				case "test":
					if len(exc.Args.V) >= 1 {
						return exc.Args.V[0], nil
					}
				case "example":
					if len(exc.Args.V) >= 2 {
						return exc.Args.V[1], nil
					}
				case "got":
					if len(exc.Args.V) >= 3 {
						return exc.Args.V[2], nil
					}
				}
			}
			return nil, object.Errorf(i.attrErr, "'DocTestFailure' object has no attribute '%s'", name)
		},
	})
	m.Dict.SetStr("DocTestFailure", docTestFailureCls)

	unexpectedExcCls := &object.Class{
		Name:  "UnexpectedException",
		Bases: []*object.Class{i.exception},
		Dict:  object.NewDict(),
	}
	unexpectedExcCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{
		Name: "__getattr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			exc := a[0].(*object.Exception)
			name := ""
			if s, ok := a[1].(*object.Str); ok {
				name = s.V
			}
			if exc.Args != nil {
				switch name {
				case "test":
					if len(exc.Args.V) >= 1 {
						return exc.Args.V[0], nil
					}
				case "example":
					if len(exc.Args.V) >= 2 {
						return exc.Args.V[1], nil
					}
				case "exc_info":
					if len(exc.Args.V) >= 3 {
						return exc.Args.V[2], nil
					}
				}
			}
			return nil, object.Errorf(i.attrErr, "'UnexpectedException' object has no attribute '%s'", name)
		},
	})
	m.Dict.SetStr("UnexpectedException", unexpectedExcCls)

	// ── DebugRunner (same as DocTestRunner; raises DocTestFailure on failure) ──
	debugRunnerCls := &object.Class{Name: "DebugRunner", Dict: object.NewDict()}
	if v, ok := docTestRunnerCls.Dict.GetStr("__init__"); ok {
		debugRunnerCls.Dict.SetStr("__init__", v)
	}
	if v, ok := docTestRunnerCls.Dict.GetStr("run"); ok {
		debugRunnerCls.Dict.SetStr("run", v)
	}
	if v, ok := docTestRunnerCls.Dict.GetStr("summarize"); ok {
		debugRunnerCls.Dict.SetStr("summarize", v)
	}
	m.Dict.SetStr("DebugRunner", debugRunnerCls)

	// ── Module-level functions ─────────────────────────────────────────────────
	m.Dict.SetStr("testmod", &object.BuiltinFunc{
		Name: "testmod",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return newTestResults(0, 0), nil
		},
	})

	m.Dict.SetStr("testfile", &object.BuiltinFunc{
		Name: "testfile",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return newTestResults(0, 0), nil
		},
	})

	m.Dict.SetStr("run_docstring_examples", &object.BuiltinFunc{
		Name: "run_docstring_examples",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("script_from_examples", &object.BuiltinFunc{
		Name: "script_from_examples",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			docstr := ""
			if s, ok := a[0].(*object.Str); ok {
				docstr = s.V
			}
			return &object.Str{V: doctestScriptFromExamples(docstr, parseExamples)}, nil
		},
	})

	m.Dict.SetStr("debug", &object.BuiltinFunc{
		Name: "debug",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("debug_script", &object.BuiltinFunc{
		Name: "debug_script",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	return m
}

// doctestCheckOutput checks whether got matches want under optionflags.
func doctestCheckOutput(want, got string, flags int64) bool {
	const (
		ellipsis           int64 = 8
		normalizeWhitespace int64 = 4
	)
	if flags&normalizeWhitespace != 0 {
		want = strings.Join(strings.Fields(want), " ")
		got = strings.Join(strings.Fields(got), " ")
	}
	if flags&ellipsis != 0 {
		return doctestEllipsisMatch(want, got)
	}
	return want == got
}

// doctestEllipsisMatch checks if got matches want where '...' is a wildcard.
func doctestEllipsisMatch(want, got string) bool {
	parts := strings.Split(want, "...")
	if len(parts) == 1 {
		return want == got
	}
	remaining := got
	for pi, part := range parts {
		switch {
		case pi == 0:
			if !strings.HasPrefix(remaining, part) {
				return false
			}
			remaining = remaining[len(part):]
		case pi == len(parts)-1:
			if !strings.HasSuffix(remaining, part) {
				return false
			}
		default:
			idx := strings.Index(remaining, part)
			if idx < 0 {
				return false
			}
			remaining = remaining[idx+len(part):]
		}
	}
	return true
}

// doctestOutputDiff formats a human-readable difference between want and got.
func doctestOutputDiff(want, got string) string {
	var sb strings.Builder
	sb.WriteString("Expected:\n")
	for _, line := range strings.Split(strings.TrimSuffix(want, "\n"), "\n") {
		sb.WriteString("    " + line + "\n")
	}
	sb.WriteString("Got:\n")
	for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		sb.WriteString("    " + line + "\n")
	}
	return sb.String()
}

// doctestScriptFromExamples converts a docstring to a runnable Python script.
func doctestScriptFromExamples(docstr string, parseExs func(string) []*object.Instance) string {
	exs := parseExs(docstr)
	var sb strings.Builder
	for _, ex := range exs {
		src, want := "", ""
		if v, ok := ex.Dict.GetStr("source"); ok {
			if s, ok2 := v.(*object.Str); ok2 {
				src = s.V
			}
		}
		if v, ok := ex.Dict.GetStr("want"); ok {
			if s, ok2 := v.(*object.Str); ok2 {
				want = s.V
			}
		}
		sb.WriteString(src)
		if want != "" {
			sb.WriteString("# Expected:\n")
			for _, wline := range strings.Split(strings.TrimSuffix(want, "\n"), "\n") {
				sb.WriteString("## " + wline + "\n")
			}
		}
	}
	return sb.String()
}

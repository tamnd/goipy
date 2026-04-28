package vm

import (
	"fmt"
	"io"
	"strings"

	"github.com/tamnd/goipy/object"
)

// tracebackWriteToFile writes text to the `file` kwarg (*object.StringIO) if
// present, otherwise to interp.Stderr.
func tracebackWriteToFile(ii any, kw *object.Dict, text string) {
	if kw != nil {
		if f, ok := kw.GetStr("file"); ok && f != object.None {
			if sio, ok2 := f.(*object.StringIO); ok2 {
				sio.V = append(sio.V, []byte(text)...)
				return
			}
		}
	}
	fmt.Fprint(ii.(*Interp).Stderr, text)
}

func (i *Interp) buildTraceback() *object.Module {
	m := &object.Module{Name: "traceback", Dict: object.NewDict()}

	// ── FrameSummary ─────────────────────────────────────────────────────────

	frameSummaryCls := &object.Class{Name: "FrameSummary", Dict: object.NewDict()}
	frameSummaryCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			filename := object.Object(&object.Str{V: "<unknown>"})
			var lineno object.Object = object.NewInt(0)
			name := object.Object(&object.Str{V: "<unknown>"})
			if len(a) >= 2 {
				filename = a[1]
			}
			if len(a) >= 3 {
				lineno = a[2]
			}
			if len(a) >= 4 {
				name = a[3]
			}
			if kw != nil {
				if v, ok := kw.GetStr("filename"); ok {
					filename = v
				}
				if v, ok := kw.GetStr("lineno"); ok {
					lineno = v
				}
				if v, ok := kw.GetStr("name"); ok {
					name = v
				}
			}
			inst.Dict.SetStr("filename", filename)
			inst.Dict.SetStr("lineno", lineno)
			inst.Dict.SetStr("name", name)
			// line defaults to empty string (matches linecache behaviour for missing files)
			inst.Dict.SetStr("line", &object.Str{V: ""})
			inst.Dict.SetStr("end_lineno", object.None)
			inst.Dict.SetStr("colno", object.None)
			inst.Dict.SetStr("end_colno", object.None)
			return object.None, nil
		},
	})

	fsItems := func(inst *object.Instance) []object.Object {
		fn, _ := inst.Dict.GetStr("filename")
		ln, _ := inst.Dict.GetStr("lineno")
		nm, _ := inst.Dict.GetStr("name")
		line, _ := inst.Dict.GetStr("line")
		return []object.Object{fn, ln, nm, line}
	}

	// __iter__ yields (filename, lineno, name, line) — 4 items like a namedtuple
	frameSummaryCls.Dict.SetStr("__iter__", &object.BuiltinFunc{
		Name: "__iter__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			items := fsItems(inst)
			idx := 0
			return &object.Iter{
				Next: func() (object.Object, bool, error) {
					if idx >= len(items) {
						return nil, false, nil
					}
					v := items[idx]
					idx++
					return v, true, nil
				},
			}, nil
		},
	})

	// __getitem__ supports index access 0-3
	frameSummaryCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{
		Name: "__getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			n, ok := toInt64(a[1])
			if !ok {
				return nil, object.Errorf(i.typeErr, "indices must be integers")
			}
			items := fsItems(inst)
			if n < 0 {
				n += int64(len(items))
			}
			if n < 0 || n >= int64(len(items)) {
				return nil, object.Errorf(i.indexErr, "FrameSummary index out of range")
			}
			return items[n], nil
		},
	})

	frameSummaryCls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			fn, _ := inst.Dict.GetStr("filename")
			ln, _ := inst.Dict.GetStr("lineno")
			nm, _ := inst.Dict.GetStr("name")
			return &object.Str{V: fmt.Sprintf("<FrameSummary file %v, line %v in %v>", fn, ln, nm)}, nil
		},
	})
	m.Dict.SetStr("FrameSummary", frameSummaryCls)

	// ── StackSummary ──────────────────────────────────────────────────────────

	stackSummaryCls := &object.Class{Name: "StackSummary", Dict: object.NewDict()}

	// formatFrameEntry formats a single frame as a string using Python's quoted format.
	formatFrameEntry := func(entry object.Object) string {
		switch v := entry.(type) {
		case *object.Instance:
			fn, _ := v.Dict.GetStr("filename")
			ln, _ := v.Dict.GetStr("lineno")
			nm, _ := v.Dict.GetStr("name")
			s := fmt.Sprintf("  File %q, line %v, in %v\n", object.Str_(fn), object.Str_(ln), object.Str_(nm))
			if lineObj, ok := v.Dict.GetStr("line"); ok {
				if ls, ok2 := lineObj.(*object.Str); ok2 && ls.V != "" {
					s += fmt.Sprintf("    %s\n", strings.TrimRight(ls.V, "\n"))
				}
			}
			return s
		case *object.Tuple:
			if len(v.V) >= 3 {
				fn := object.Str_(v.V[0])
				ln := object.Str_(v.V[1])
				nm := object.Str_(v.V[2])
				s := fmt.Sprintf("  File %q, line %v, in %v\n", fn, ln, nm)
				if len(v.V) >= 4 {
					if ls, ok := v.V[3].(*object.Str); ok && ls.V != "" {
						s += fmt.Sprintf("    %s\n", strings.TrimRight(ls.V, "\n"))
					}
				}
				return s
			}
		}
		return ""
	}

	// tupleToFrameSummary converts a raw (filename, lineno, name, line) tuple
	// to a FrameSummary instance.
	tupleToFrameSummary := func(entry object.Object) *object.Instance {
		fs := &object.Instance{Class: frameSummaryCls, Dict: object.NewDict()}
		switch v := entry.(type) {
		case *object.Instance:
			// already a FrameSummary
			return v
		case *object.Tuple:
			if len(v.V) >= 1 {
				fs.Dict.SetStr("filename", v.V[0])
			} else {
				fs.Dict.SetStr("filename", &object.Str{V: "<unknown>"})
			}
			if len(v.V) >= 2 {
				fs.Dict.SetStr("lineno", v.V[1])
			} else {
				fs.Dict.SetStr("lineno", object.NewInt(0))
			}
			if len(v.V) >= 3 {
				fs.Dict.SetStr("name", v.V[2])
			} else {
				fs.Dict.SetStr("name", &object.Str{V: "<unknown>"})
			}
			if len(v.V) >= 4 {
				fs.Dict.SetStr("line", v.V[3])
			} else {
				fs.Dict.SetStr("line", &object.Str{V: ""})
			}
		default:
			fs.Dict.SetStr("filename", &object.Str{V: "<unknown>"})
			fs.Dict.SetStr("lineno", object.NewInt(0))
			fs.Dict.SetStr("name", &object.Str{V: "<unknown>"})
			fs.Dict.SetStr("line", &object.Str{V: ""})
		}
		fs.Dict.SetStr("end_lineno", object.None)
		fs.Dict.SetStr("colno", object.None)
		fs.Dict.SetStr("end_colno", object.None)
		return fs
	}

	stackSummaryFrames := func(inst *object.Instance) []object.Object {
		if fObj, ok := inst.Dict.GetStr("_frames"); ok {
			if lst, ok2 := fObj.(*object.List); ok2 {
				return lst.V
			}
		}
		return nil
	}

	stackSummaryCls.Dict.SetStr("extract", &object.BuiltinFunc{
		Name: "extract",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Instance{Class: stackSummaryCls, Dict: object.NewDict()}, nil
		},
	})

	stackSummaryCls.Dict.SetStr("from_list", &object.BuiltinFunc{
		Name: "from_list",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := &object.Instance{Class: stackSummaryCls, Dict: object.NewDict()}
			var listArg object.Object
			for _, arg := range a {
				if _, ok := arg.(*object.List); ok {
					listArg = arg
					break
				}
			}
			if listArg != nil {
				// Convert raw tuples to FrameSummary instances
				raw := listArg.(*object.List)
				converted := make([]object.Object, len(raw.V))
				for idx, entry := range raw.V {
					converted[idx] = tupleToFrameSummary(entry)
				}
				inst.Dict.SetStr("_frames", &object.List{V: converted})
			}
			return inst, nil
		},
	})

	stackSummaryCls.Dict.SetStr("__iter__", &object.BuiltinFunc{
		Name: "__iter__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			frames := stackSummaryFrames(inst)
			idx := 0
			return &object.Iter{
				Next: func() (object.Object, bool, error) {
					if idx >= len(frames) {
						return nil, false, nil
					}
					v := frames[idx]
					idx++
					return v, true, nil
				},
			}, nil
		},
	})

	stackSummaryCls.Dict.SetStr("__len__", &object.BuiltinFunc{
		Name: "__len__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			return object.NewInt(int64(len(stackSummaryFrames(inst)))), nil
		},
	})

	stackSummaryCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{
		Name: "__getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			frames := stackSummaryFrames(inst)
			n, ok := toInt64(a[1])
			if !ok {
				return nil, object.Errorf(i.typeErr, "indices must be integers")
			}
			if n < 0 {
				n += int64(len(frames))
			}
			if n < 0 || n >= int64(len(frames)) {
				return nil, object.Errorf(i.indexErr, "StackSummary index out of range")
			}
			return frames[n], nil
		},
	})

	stackSummaryCls.Dict.SetStr("format", &object.BuiltinFunc{
		Name: "format",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			var lines []object.Object
			for _, fSumObj := range stackSummaryFrames(inst) {
				s := formatFrameEntry(fSumObj)
				if s != "" {
					lines = append(lines, &object.Str{V: s})
				}
			}
			return &object.List{V: lines}, nil
		},
	})

	// format_frame_summary(frame_summary) → str
	stackSummaryCls.Dict.SetStr("format_frame_summary", &object.BuiltinFunc{
		Name: "format_frame_summary",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// a[0] = self (StackSummary), a[1] = frame_summary
			var entry object.Object
			if len(a) >= 2 {
				entry = a[1]
			} else if len(a) >= 1 {
				entry = a[0]
			}
			s := formatFrameEntry(entry)
			return &object.Str{V: s}, nil
		},
	})

	m.Dict.SetStr("StackSummary", stackSummaryCls)

	// ── extract_stack ─────────────────────────────────────────────────────────

	extractStack := func(ii any) *object.Instance {
		interp := ii.(*Interp)
		var frames []object.Object
		for f := interp.curFrame; f != nil; f = f.Back {
			lineno := f.Code.LineForOffset(f.LastIP)
			if lineno == 0 {
				lineno = f.Code.FirstLineNo
			}
			fs := &object.Instance{Class: frameSummaryCls, Dict: object.NewDict()}
			fs.Dict.SetStr("filename", &object.Str{V: f.Code.Filename})
			fs.Dict.SetStr("lineno", object.NewInt(int64(lineno)))
			fs.Dict.SetStr("name", &object.Str{V: f.Code.Name})
			fs.Dict.SetStr("line", &object.Str{V: ""})
			fs.Dict.SetStr("end_lineno", object.None)
			fs.Dict.SetStr("colno", object.None)
			fs.Dict.SetStr("end_colno", object.None)
			frames = append(frames, fs)
		}
		// reverse (most recent call last)
		for l, r := 0, len(frames)-1; l < r; l, r = l+1, r-1 {
			frames[l], frames[r] = frames[r], frames[l]
		}
		ss := &object.Instance{Class: stackSummaryCls, Dict: object.NewDict()}
		ss.Dict.SetStr("_frames", &object.List{V: frames})
		return ss
	}

	m.Dict.SetStr("extract_stack", &object.BuiltinFunc{
		Name: "extract_stack",
		Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return extractStack(ii), nil
		},
	})

	m.Dict.SetStr("extract_tb", &object.BuiltinFunc{
		Name: "extract_tb",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ss := &object.Instance{Class: stackSummaryCls, Dict: object.NewDict()}
			ss.Dict.SetStr("_frames", &object.List{V: nil})
			return ss, nil
		},
	})

	// ── format_list ───────────────────────────────────────────────────────────

	m.Dict.SetStr("format_list", &object.BuiltinFunc{
		Name: "format_list",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{V: nil}, nil
			}
			switch v := a[0].(type) {
			case *object.Instance:
				// StackSummary — call its format()
				fmtFn, ok2 := stackSummaryCls.Dict.GetStr("format")
				if !ok2 {
					return &object.List{V: nil}, nil
				}
				return i.callObject(fmtFn, []object.Object{v}, nil)
			case *object.List:
				items := []object.Object{}
				for _, entry := range v.V {
					s := formatFrameEntry(entry)
					if s != "" {
						items = append(items, &object.Str{V: s})
					}
				}
				return &object.List{V: items}, nil
			}
			return &object.List{V: nil}, nil
		},
	})

	// ── format helpers ────────────────────────────────────────────────────────

	fmtExcOnly := func(excObj object.Object) []string {
		if excObj == nil || excObj == object.None {
			return nil
		}
		switch e := excObj.(type) {
		case *object.Exception:
			msg := ""
			if e.Args != nil && len(e.Args.V) > 0 {
				msg = fmt.Sprintf("%v", e.Args.V[0])
			}
			if msg != "" {
				return []string{fmt.Sprintf("%s: %s\n", e.Class.Name, msg)}
			}
			return []string{e.Class.Name + "\n"}
		case *object.Instance:
			clsName := e.Class.Name
			if msgObj, ok := e.Dict.GetStr("args"); ok {
				if tpl, ok2 := msgObj.(*object.Tuple); ok2 && len(tpl.V) > 0 {
					return []string{fmt.Sprintf("%s: %v\n", clsName, tpl.V[0])}
				}
			}
			return []string{clsName + "\n"}
		}
		return []string{fmt.Sprintf("%v\n", excObj)}
	}

	excTypeName := func(excObj object.Object) string {
		switch e := excObj.(type) {
		case *object.Exception:
			return e.Class.Name
		case *object.Instance:
			return e.Class.Name
		}
		return "Exception"
	}

	// resolveExcArg handles both 1-arg (Python 3.10+) and 2-arg forms.
	// 1-arg: format_exception_only(exc)
	// 2-arg: format_exception_only(exc_type, exc_value)
	resolveExcArg := func(a []object.Object) object.Object {
		if len(a) >= 2 && a[1] != object.None {
			return a[1]
		}
		if len(a) >= 1 {
			return a[0]
		}
		return object.None
	}

	m.Dict.SetStr("format_exception_only", &object.BuiltinFunc{
		Name: "format_exception_only",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			excObj := resolveExcArg(a)
			lines := fmtExcOnly(excObj)
			items := make([]object.Object, len(lines))
			for idx, l := range lines {
				items[idx] = &object.Str{V: l}
			}
			return &object.List{V: items}, nil
		},
	})

	m.Dict.SetStr("format_tb", &object.BuiltinFunc{
		Name: "format_tb",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: nil}, nil
		},
	})

	fmtException := func(excObj object.Object) []string {
		lines := []string{"Traceback (most recent call last):\n"}
		lines = append(lines, fmtExcOnly(excObj)...)
		return lines
	}

	m.Dict.SetStr("format_exception", &object.BuiltinFunc{
		Name: "format_exception",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			excObj := resolveExcArg(a)
			lines := fmtException(excObj)
			items := make([]object.Object, len(lines))
			for idx, l := range lines {
				items[idx] = &object.Str{V: l}
			}
			return &object.List{V: items}, nil
		},
	})

	writeExc := func(ii any, excObj object.Object, kw *object.Dict) {
		var sb strings.Builder
		for _, l := range fmtException(excObj) {
			sb.WriteString(l)
		}
		tracebackWriteToFile(ii, kw, sb.String())
	}

	getCurrentExc := func(ii any) object.Object {
		for f := ii.(*Interp).curFrame; f != nil; f = f.Back {
			if f.ExcInfo != nil {
				return f.ExcInfo
			}
		}
		return object.None
	}

	m.Dict.SetStr("format_exc", &object.BuiltinFunc{
		Name: "format_exc",
		Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			exc := getCurrentExc(ii)
			if exc == object.None {
				return &object.Str{V: "NoneType: None\n"}, nil
			}
			var sb strings.Builder
			for _, l := range fmtException(exc) {
				sb.WriteString(l)
			}
			return &object.Str{V: sb.String()}, nil
		},
	})

	m.Dict.SetStr("print_exc", &object.BuiltinFunc{
		Name: "print_exc",
		Call: func(ii any, _ []object.Object, kw *object.Dict) (object.Object, error) {
			writeExc(ii, getCurrentExc(ii), kw)
			return object.None, nil
		},
	})

	m.Dict.SetStr("print_exception", &object.BuiltinFunc{
		Name: "print_exception",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			excObj := resolveExcArg(a)
			writeExc(ii, excObj, kw)
			return object.None, nil
		},
	})

	m.Dict.SetStr("print_tb", &object.BuiltinFunc{
		Name: "print_tb",
		Call: func(ii any, _ []object.Object, kw *object.Dict) (object.Object, error) {
			tracebackWriteToFile(ii, kw, "Traceback (most recent call last):\n")
			return object.None, nil
		},
	})

	m.Dict.SetStr("print_stack", &object.BuiltinFunc{
		Name: "print_stack",
		Call: func(ii any, _ []object.Object, kw *object.Dict) (object.Object, error) {
			interp := ii.(*Interp)
			var sb strings.Builder
			sb.WriteString("Stack (most recent call last):\n")
			for f := interp.curFrame; f != nil; f = f.Back {
				lineno := f.Code.LineForOffset(f.LastIP)
				if lineno == 0 {
					lineno = f.Code.FirstLineNo
				}
				sb.WriteString(fmt.Sprintf("  File %q, line %d, in %s\n", f.Code.Filename, lineno, f.Code.Name))
			}
			tracebackWriteToFile(ii, kw, sb.String())
			return object.None, nil
		},
	})

	m.Dict.SetStr("print_last", &object.BuiltinFunc{
		Name: "print_last",
		Call: func(ii any, _ []object.Object, kw *object.Dict) (object.Object, error) {
			writeExc(ii, getCurrentExc(ii), kw)
			return object.None, nil
		},
	})

	m.Dict.SetStr("format_stack", &object.BuiltinFunc{
		Name: "format_stack",
		Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ss := extractStack(ii)
			fmtFn, _ := stackSummaryCls.Dict.GetStr("format")
			return i.callObject(fmtFn, []object.Object{ss}, nil)
		},
	})

	m.Dict.SetStr("clear_frames", &object.BuiltinFunc{
		Name: "clear_frames",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("walk_stack", &object.BuiltinFunc{
		Name: "walk_stack",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Iter{Next: func() (object.Object, bool, error) { return nil, false, nil }}, nil
		},
	})

	m.Dict.SetStr("walk_tb", &object.BuiltinFunc{
		Name: "walk_tb",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Iter{Next: func() (object.Object, bool, error) { return nil, false, nil }}, nil
		},
	})

	m.Dict.SetStr("print_list", &object.BuiltinFunc{
		Name: "print_list",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			var sb strings.Builder
			switch v := a[0].(type) {
			case *object.List:
				for _, entry := range v.V {
					if s := formatFrameEntry(entry); s != "" {
						sb.WriteString(s)
					}
				}
			case *object.Instance:
				// StackSummary: format and print
				fmtFn, _ := stackSummaryCls.Dict.GetStr("format")
				res, err := i.callObject(fmtFn, []object.Object{v}, nil)
				if err != nil {
					return nil, err
				}
				if lst, ok := res.(*object.List); ok {
					for _, item := range lst.V {
						sb.WriteString(fmt.Sprintf("%v", item))
					}
				}
			}
			tracebackWriteToFile(ii, kw, sb.String())
			return object.None, nil
		},
	})

	// ── TracebackException ────────────────────────────────────────────────────

	tbExcCls := &object.Class{Name: "TracebackException", Dict: object.NewDict()}

	// buildTBE creates a TracebackException instance from an exception object.
	var buildTBE func(excObj object.Object) *object.Instance
	buildTBE = func(excObj object.Object) *object.Instance {
		inst := &object.Instance{Class: tbExcCls, Dict: object.NewDict()}
		emptyStack := &object.Instance{Class: stackSummaryCls, Dict: object.NewDict()}
		emptyStack.Dict.SetStr("_frames", &object.List{V: nil})
		inst.Dict.SetStr("stack", emptyStack)
		inst.Dict.SetStr("__notes__", object.None)
		inst.Dict.SetStr("exceptions", object.None)
		inst.Dict.SetStr("filename", object.None)
		inst.Dict.SetStr("lineno", object.None)
		inst.Dict.SetStr("end_lineno", object.None)
		inst.Dict.SetStr("text", object.None)
		inst.Dict.SetStr("offset", object.None)
		inst.Dict.SetStr("end_offset", object.None)
		inst.Dict.SetStr("msg", &object.Str{V: ""})

		typeName := "Exception"
		var cause, ctx object.Object = object.None, object.None
		suppressCtx := false

		switch e := excObj.(type) {
		case *object.Exception:
			typeName = e.Class.Name
			if e.Cause != nil {
				cause = buildTBE(e.Cause)
				suppressCtx = true
			}
			if e.Ctx != nil {
				ctx = buildTBE(e.Ctx)
			}
		case *object.Instance:
			typeName = e.Class.Name
		}

		inst.Dict.SetStr("exc_type_str", &object.Str{V: typeName})
		inst.Dict.SetStr("__cause__", cause)
		inst.Dict.SetStr("__context__", ctx)
		inst.Dict.SetStr("__suppress_context__", object.BoolOf(suppressCtx))
		inst.Dict.SetStr("_exc", excObj)
		return inst
	}

	tbExcCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			emptyStack := &object.Instance{Class: stackSummaryCls, Dict: object.NewDict()}
			emptyStack.Dict.SetStr("_frames", &object.List{V: nil})
			inst.Dict.SetStr("stack", emptyStack)
			inst.Dict.SetStr("__cause__", object.None)
			inst.Dict.SetStr("__context__", object.None)
			inst.Dict.SetStr("__suppress_context__", object.BoolOf(false))
			inst.Dict.SetStr("__notes__", object.None)
			inst.Dict.SetStr("exceptions", object.None)
			inst.Dict.SetStr("exc_type_str", &object.Str{V: "Exception"})
			inst.Dict.SetStr("filename", object.None)
			inst.Dict.SetStr("lineno", object.None)
			inst.Dict.SetStr("end_lineno", object.None)
			inst.Dict.SetStr("text", object.None)
			inst.Dict.SetStr("offset", object.None)
			inst.Dict.SetStr("end_offset", object.None)
			inst.Dict.SetStr("msg", &object.Str{V: ""})
			if len(a) >= 3 {
				inst.Dict.SetStr("_exc", a[2])
			}
			return object.None, nil
		},
	})

	tbExcCls.Dict.SetStr("from_exception", &object.BuiltinFunc{
		Name: "from_exception",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			var excObj object.Object = object.None
			if len(a) >= 2 {
				excObj = a[1]
			} else if len(a) >= 1 {
				excObj = a[0]
			}
			return buildTBE(excObj), nil
		},
	})

	tbExcCls.Dict.SetStr("format", &object.BuiltinFunc{
		Name: "format",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			lines := []object.Object{&object.Str{V: "Traceback (most recent call last):\n"}}
			if excObj, ok := inst.Dict.GetStr("_exc"); ok {
				for _, l := range fmtExcOnly(excObj) {
					lines = append(lines, &object.Str{V: l})
				}
			} else if tsObj, ok2 := inst.Dict.GetStr("exc_type_str"); ok2 {
				lines = append(lines, &object.Str{V: fmt.Sprintf("%v\n", tsObj)})
			}
			return &object.List{V: lines}, nil
		},
	})

	tbExcCls.Dict.SetStr("format_exception_only", &object.BuiltinFunc{
		Name: "format_exception_only",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			var lines []string
			if excObj, ok := inst.Dict.GetStr("_exc"); ok {
				lines = fmtExcOnly(excObj)
			} else if tsObj, ok2 := inst.Dict.GetStr("exc_type_str"); ok2 {
				lines = []string{fmt.Sprintf("%v\n", tsObj)}
			}
			items := make([]object.Object, len(lines))
			for idx, l := range lines {
				items[idx] = &object.Str{V: l}
			}
			return &object.List{V: items}, nil
		},
	})

	tbExcCls.Dict.SetStr("print", &object.BuiltinFunc{
		Name: "print",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			var sb strings.Builder
			sb.WriteString("Traceback (most recent call last):\n")
			excObj, _ := inst.Dict.GetStr("_exc")
			for _, l := range fmtExcOnly(excObj) {
				sb.WriteString(l)
			}
			tracebackWriteToFile(ii, kw, sb.String())
			return object.None, nil
		},
	})

	m.Dict.SetStr("TracebackException", tbExcCls)

	// ── exc_type_str helper on TracebackException (exposed via format/print) ──
	_ = excTypeName // used above
	_ = io.Discard  // ensure io import is used

	return m
}

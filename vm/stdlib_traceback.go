package vm

import (
	"fmt"
	"io"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildTraceback() *object.Module {
	m := &object.Module{Name: "traceback", Dict: object.NewDict()}

	// ── FrameSummary class ────────────────────────────────────────────────────

	frameSummaryCls := &object.Class{Name: "FrameSummary", Dict: object.NewDict()}
	frameSummaryCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			var filename, name object.Object
			filename = &object.Str{V: "<unknown>"}
			name = &object.Str{V: "<unknown>"}
			var lineno object.Object = object.NewInt(0)
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
			inst.Dict.SetStr("line", object.None)
			inst.Dict.SetStr("end_lineno", object.None)
			inst.Dict.SetStr("colno", object.None)
			inst.Dict.SetStr("end_colno", object.None)
			return object.None, nil
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

	// ── StackSummary class ────────────────────────────────────────────────────

	stackSummaryCls := &object.Class{Name: "StackSummary", Dict: object.NewDict()}

	// extract returns a StackSummary from the current frame chain
	stackSummaryCls.Dict.SetStr("extract", &object.BuiltinFunc{
		Name: "extract",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// a[0] is class (or first item of frame_gen in positional form)
			// We just return an empty StackSummary; real extraction needs frame iteration
			return &object.Instance{Class: stackSummaryCls, Dict: object.NewDict()}, nil
		},
	})
	stackSummaryCls.Dict.SetStr("from_list", &object.BuiltinFunc{
		Name: "from_list",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := &object.Instance{Class: stackSummaryCls, Dict: object.NewDict()}
			// a[0] may be the class (bound-method call) or the list (unbound call)
			var listArg object.Object
			for _, arg := range a {
				if _, ok := arg.(*object.List); ok {
					listArg = arg
					break
				}
			}
			if listArg != nil {
				inst.Dict.SetStr("_frames", listArg)
			}
			return inst, nil
		},
	})
	formatFrameEntry := func(entry object.Object) string {
		switch v := entry.(type) {
		case *object.Instance:
			fn, _ := v.Dict.GetStr("filename")
			ln, _ := v.Dict.GetStr("lineno")
			nm, _ := v.Dict.GetStr("name")
			s := fmt.Sprintf("  File %v, line %v, in %v\n", fn, ln, nm)
			if lineObj, ok := v.Dict.GetStr("line"); ok && lineObj != object.None {
				s += fmt.Sprintf("    %v\n", lineObj)
			}
			return s
		case *object.Tuple:
			if len(v.V) >= 3 {
				fn := fmt.Sprintf("%v", v.V[0])
				ln := fmt.Sprintf("%v", v.V[1])
				nm := fmt.Sprintf("%v", v.V[2])
				s := fmt.Sprintf("  File %v, line %v, in %v\n", fn, ln, nm)
				if len(v.V) >= 4 && v.V[3] != object.None {
					s += fmt.Sprintf("    %v\n", v.V[3])
				}
				return s
			}
		}
		return ""
	}

	stackSummaryCls.Dict.SetStr("format", &object.BuiltinFunc{
		Name: "format",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			var lines []object.Object
			if fObj, ok := inst.Dict.GetStr("_frames"); ok {
				if lst, ok2 := fObj.(*object.List); ok2 {
					for _, fSumObj := range lst.V {
						s := formatFrameEntry(fSumObj)
						if s != "" {
							lines = append(lines, &object.Str{V: s})
						}
					}
				}
			}
			return &object.List{V: lines}, nil
		},
	})
	m.Dict.SetStr("StackSummary", stackSummaryCls)

	// ── extract_stack: walk curFrame chain ────────────────────────────────────

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
			fs.Dict.SetStr("line", object.None)
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
			return []string{fmt.Sprintf("%s: %s\n", e.Class.Name, msg)}
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

	m.Dict.SetStr("format_exception_only", &object.BuiltinFunc{
		Name: "format_exception_only",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			var excObj object.Object
			if len(a) >= 1 {
				excObj = a[0]
			}
			if len(a) >= 2 {
				excObj = a[1]
			}
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
				// raw list of (filename, lineno, name, line) tuples
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

	fmtException := func(excObj object.Object) []string {
		lines := []string{"Traceback (most recent call last):\n"}
		lines = append(lines, fmtExcOnly(excObj)...)
		return lines
	}

	m.Dict.SetStr("format_exception", &object.BuiltinFunc{
		Name: "format_exception",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			var excObj object.Object
			if len(a) >= 1 {
				excObj = a[0]
			}
			if len(a) >= 2 && a[1] != object.None {
				excObj = a[1]
			}
			lines := fmtException(excObj)
			items := make([]object.Object, len(lines))
			for idx, l := range lines {
				items[idx] = &object.Str{V: l}
			}
			return &object.List{V: items}, nil
		},
	})

	// writeExc writes a formatted exception to w (defaults to i.Stderr)
	writeExc := func(ii any, excObj object.Object, w io.Writer) {
		if w == nil {
			w = ii.(*Interp).Stderr
		}
		for _, l := range fmtException(excObj) {
			fmt.Fprint(w, l)
		}
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
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			var w io.Writer = ii.(*Interp).Stderr
			writeExc(ii, getCurrentExc(ii), w)
			return object.None, nil
		},
	})

	m.Dict.SetStr("print_exception", &object.BuiltinFunc{
		Name: "print_exception",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			var excObj object.Object
			if len(a) >= 1 {
				excObj = a[0]
			}
			writeExc(ii, excObj, nil)
			return object.None, nil
		},
	})

	m.Dict.SetStr("print_tb", &object.BuiltinFunc{
		Name: "print_tb",
		Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			fmt.Fprintf(ii.(*Interp).Stderr, "Traceback (most recent call last):\n")
			return object.None, nil
		},
	})

	m.Dict.SetStr("print_stack", &object.BuiltinFunc{
		Name: "print_stack",
		Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			interp := ii.(*Interp)
			fmt.Fprintf(interp.Stderr, "Stack (most recent call last):\n")
			for f := interp.curFrame; f != nil; f = f.Back {
				lineno := f.Code.LineForOffset(f.LastIP)
				if lineno == 0 {
					lineno = f.Code.FirstLineNo
				}
				fmt.Fprintf(interp.Stderr, "  File %q, line %d, in %s\n", f.Code.Filename, lineno, f.Code.Name)
			}
			return object.None, nil
		},
	})

	m.Dict.SetStr("print_last", &object.BuiltinFunc{
		Name: "print_last",
		Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			writeExc(ii, getCurrentExc(ii), nil)
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
			return &object.List{V: nil}, nil
		},
	})

	m.Dict.SetStr("walk_tb", &object.BuiltinFunc{
		Name: "walk_tb",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: nil}, nil
		},
	})

	m.Dict.SetStr("print_list", &object.BuiltinFunc{
		Name: "print_list",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			if lst, ok := a[0].(*object.List); ok {
				for _, item := range lst.V {
					fmt.Fprintf(ii.(*Interp).Stderr, "%v", item)
				}
			}
			return object.None, nil
		},
	})

	// ── TracebackException class ──────────────────────────────────────────────

	tbExcCls := &object.Class{Name: "TracebackException", Dict: object.NewDict()}
	tbExcCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("stack", &object.Instance{Class: stackSummaryCls, Dict: object.NewDict()})
			inst.Dict.SetStr("__cause__", object.None)
			inst.Dict.SetStr("__context__", object.None)
			inst.Dict.SetStr("__suppress_context__", &object.Bool{V: false})
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
			// store exc value for formatting
			if len(a) >= 3 {
				inst.Dict.SetStr("_exc", a[2])
			}
			return object.None, nil
		},
	})
	tbExcCls.Dict.SetStr("from_exception", &object.BuiltinFunc{
		Name: "from_exception",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := &object.Instance{Class: tbExcCls, Dict: object.NewDict()}
			inst.Dict.SetStr("stack", &object.Instance{Class: stackSummaryCls, Dict: object.NewDict()})
			inst.Dict.SetStr("__cause__", object.None)
			inst.Dict.SetStr("__context__", object.None)
			inst.Dict.SetStr("__suppress_context__", &object.Bool{V: false})
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
			if len(a) >= 2 {
				inst.Dict.SetStr("_exc", a[1])
			} else if len(a) >= 1 {
				inst.Dict.SetStr("_exc", a[0])
			}
			return inst, nil
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
			}
			return &object.List{V: lines}, nil
		},
	})
	tbExcCls.Dict.SetStr("format_exception_only", &object.BuiltinFunc{
		Name: "format_exception_only",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			excObj, _ := inst.Dict.GetStr("_exc")
			lines := fmtExcOnly(excObj)
			items := make([]object.Object, len(lines))
			for idx, l := range lines {
				items[idx] = &object.Str{V: l}
			}
			return &object.List{V: items}, nil
		},
	})
	tbExcCls.Dict.SetStr("print", &object.BuiltinFunc{
		Name: "print",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			excObj, _ := inst.Dict.GetStr("_exc")
			writeExc(ii, excObj, nil)
			return object.None, nil
		},
	})
	m.Dict.SetStr("TracebackException", tbExcCls)

	return m
}

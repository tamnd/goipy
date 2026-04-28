package vm

import (
	"fmt"
	"strings"

	"github.com/tamnd/goipy/object"
)

// isIncompleteSource returns true when source clearly needs more input.
func isIncompleteSource(source string) bool {
	s := strings.TrimRight(source, " \t\n\r")
	if strings.HasSuffix(s, ":") || strings.HasSuffix(s, "\\") {
		return true
	}
	open := 0
	for _, c := range source {
		switch c {
		case '(', '[', '{':
			open++
		case ')', ']', '}':
			open--
		}
	}
	return open > 0
}

func (i *Interp) buildCode() *object.Module {
	m := &object.Module{Name: "code", Dict: object.NewDict()}

	// ── CommandCompiler class ─────────────────────────────────────────────────

	ccClass := &object.Class{Name: "CommandCompiler", Dict: object.NewDict()}
	ccClass.Dict.SetStr("__call__", &object.BuiltinFunc{
		Name: "__call__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			source := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					source = s.V
				}
			}
			if isIncompleteSource(source) {
				return object.None, nil
			}
			return &object.Code{Name: "<compile_command>"}, nil
		},
	})

	makeCC := func() *object.Instance {
		return &object.Instance{Class: ccClass, Dict: object.NewDict()}
	}

	// ── compile_command ───────────────────────────────────────────────────────

	m.Dict.SetStr("compile_command", &object.BuiltinFunc{
		Name: "compile_command",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			source := ""
			if len(a) >= 1 {
				if s, ok := a[0].(*object.Str); ok {
					source = s.V
				}
			}
			if isIncompleteSource(source) {
				return object.None, nil
			}
			return &object.Code{Name: "<compile_command>"}, nil
		},
	})

	// ── Quitter class ─────────────────────────────────────────────────────────

	quitterClass := &object.Class{Name: "Quitter", Dict: object.NewDict()}
	quitterClass.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			self.Dict.SetStr("name", a[1])
			return object.None, nil
		},
	})
	quitterClass.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			name := "quit"
			if len(a) >= 1 {
				if inst, ok := a[0].(*object.Instance); ok {
					if n, ok2 := inst.Dict.GetStr("name"); ok2 {
						if s, ok3 := n.(*object.Str); ok3 {
							name = s.V
						}
					}
				}
			}
			return &object.Str{V: "Use " + name + " or Ctrl-D (i.e. EOF) to exit"}, nil
		},
	})
	quitterClass.Dict.SetStr("__call__", &object.BuiltinFunc{
		Name: "__call__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.systemExit, "")
		},
	})

	m.Dict.SetStr("Quitter", quitterClass)

	// ── InteractiveInterpreter class ──────────────────────────────────────────

	iiClass := &object.Class{Name: "InteractiveInterpreter", Dict: object.NewDict()}

	iiClass.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			var locals *object.Dict
			if len(a) >= 2 && a[1] != object.None {
				if d, ok := a[1].(*object.Dict); ok {
					locals = d
				}
			}
			if locals == nil {
				locals = object.NewDict()
				locals.SetStr("__name__", &object.Str{V: "__console__"})
				locals.SetStr("__doc__", object.None)
			}
			self.Dict.SetStr("locals", locals)
			self.Dict.SetStr("compile", makeCC())
			return object.None, nil
		},
	})

	iiClass.Dict.SetStr("runsource", &object.BuiltinFunc{
		Name: "runsource",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			source := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					source = s.V
				}
			}
			return object.BoolOf(isIncompleteSource(source)), nil
		},
	})

	iiClass.Dict.SetStr("runcode", &object.BuiltinFunc{
		Name: "runcode",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	iiClass.Dict.SetStr("showsyntaxerror", &object.BuiltinFunc{
		Name: "showsyntaxerror",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	iiClass.Dict.SetStr("showtraceback", &object.BuiltinFunc{
		Name: "showtraceback",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	iiClass.Dict.SetStr("write", &object.BuiltinFunc{
		Name: "write",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					interp := ii.(*Interp)
					fmt.Fprint(interp.Stderr, s.V)
				}
			}
			return object.None, nil
		},
	})

	m.Dict.SetStr("InteractiveInterpreter", iiClass)

	// ── InteractiveConsole class ──────────────────────────────────────────────

	icClass := &object.Class{Name: "InteractiveConsole", Dict: object.NewDict()}
	icClass.Bases = []*object.Class{iiClass}

	icClass.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)

			// Initialise InteractiveInterpreter fields
			var locals *object.Dict
			if len(a) >= 2 && a[1] != object.None {
				if d, ok := a[1].(*object.Dict); ok {
					locals = d
				}
			}
			if locals == nil {
				locals = object.NewDict()
				locals.SetStr("__name__", &object.Str{V: "__console__"})
				locals.SetStr("__doc__", object.None)
			}
			self.Dict.SetStr("locals", locals)
			self.Dict.SetStr("compile", makeCC())

			// filename
			filename := "<console>"
			if len(a) >= 3 {
				if s, ok := a[2].(*object.Str); ok {
					filename = s.V
				}
			}
			if kw != nil {
				if fv, ok := kw.GetStr("filename"); ok {
					if s, ok2 := fv.(*object.Str); ok2 {
						filename = s.V
					}
				}
			}
			self.Dict.SetStr("filename", &object.Str{V: filename})

			// local_exit
			var localExit object.Object = object.False
			if kw != nil {
				if lv, ok := kw.GetStr("local_exit"); ok {
					localExit = lv
				}
			}
			self.Dict.SetStr("local_exit", localExit)
			self.Dict.SetStr("buffer", &object.List{V: []object.Object{}})
			return object.None, nil
		},
	})

	icClass.Dict.SetStr("resetbuffer", &object.BuiltinFunc{
		Name: "resetbuffer",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				if self, ok := a[0].(*object.Instance); ok {
					self.Dict.SetStr("buffer", &object.List{V: []object.Object{}})
				}
			}
			return object.None, nil
		},
	})

	icClass.Dict.SetStr("push", &object.BuiltinFunc{
		Name: "push",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.False, nil
			}
			self := a[0].(*object.Instance)
			line := ""
			if s, ok := a[1].(*object.Str); ok {
				line = s.V
			}

			// Append line to buffer
			bufObj, _ := self.Dict.GetStr("buffer")
			buf, _ := bufObj.(*object.List)
			if buf == nil {
				buf = &object.List{}
			}
			buf.V = append(buf.V, &object.Str{V: line})
			self.Dict.SetStr("buffer", buf)

			// Join buffer and check completeness
			parts := make([]string, len(buf.V))
			for idx, v := range buf.V {
				if s, ok := v.(*object.Str); ok {
					parts[idx] = s.V
				}
			}
			joined := strings.Join(parts, "\n")
			incomplete := isIncompleteSource(joined)

			if !incomplete {
				self.Dict.SetStr("buffer", &object.List{V: []object.Object{}})
			}
			return object.BoolOf(incomplete), nil
		},
	})

	icClass.Dict.SetStr("raw_input", &object.BuiltinFunc{
		Name: "raw_input",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: ""}, nil
		},
	})

	icClass.Dict.SetStr("interact", &object.BuiltinFunc{
		Name: "interact",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// Inherit InteractiveInterpreter methods
	for _, name := range []string{"runsource", "runcode", "showsyntaxerror", "showtraceback", "write"} {
		if fn, ok := iiClass.Dict.GetStr(name); ok {
			icClass.Dict.SetStr(name, fn)
		}
	}

	m.Dict.SetStr("InteractiveConsole", icClass)

	// ── interact module-level function ────────────────────────────────────────

	m.Dict.SetStr("interact", &object.BuiltinFunc{
		Name: "interact",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("CommandCompiler", ccClass)

	return m
}

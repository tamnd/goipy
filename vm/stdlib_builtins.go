package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildBuiltins returns the "builtins" module whose __dict__ is the
// interpreter's live builtin namespace.  All built-in names are already
// present; we only add a handful of module-identity attributes and the
// callable __import__ / exec / eval / compile that are proper builtins in
// CPython but were omitted from initBuiltins because they need the import
// machinery or a bytecode compiler.
func (i *Interp) buildBuiltins() *object.Module {
	b := i.Builtins

	// Module identity.
	if _, ok := b.GetStr("__name__"); !ok {
		b.SetStr("__name__", &object.Str{V: "builtins"})
	}
	if _, ok := b.GetStr("__doc__"); !ok {
		b.SetStr("__doc__", &object.Str{V: "Built-in functions, exceptions, and other objects."})
	}
	if _, ok := b.GetStr("__package__"); !ok {
		b.SetStr("__package__", object.None)
	}
	if _, ok := b.GetStr("__loader__"); !ok {
		b.SetStr("__loader__", object.None)
	}
	if _, ok := b.GetStr("__spec__"); !ok {
		b.SetStr("__spec__", object.None)
	}

	// __import__(name, globals=None, locals=None, fromlist=(), level=0)
	// Thin wrapper around the interpreter's existing import machinery.
	if _, ok := b.GetStr("__import__"); !ok {
		b.SetStr("__import__", &object.BuiltinFunc{
			Name: "__import__",
			Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
				interp := ii.(*Interp)
				if len(a) < 1 {
					return nil, object.Errorf(interp.typeErr, "__import__() requires at least 1 argument")
				}
				nameStr, ok := a[0].(*object.Str)
				if !ok {
					return nil, object.Errorf(interp.typeErr, "__import__() argument 1 must be str")
				}
				var globals *object.Dict
				if len(a) >= 2 {
					if d, ok2 := a[1].(*object.Dict); ok2 {
						globals = d
					}
				}
				var fromlist *object.Tuple
				if len(a) >= 4 {
					if t, ok2 := a[3].(*object.Tuple); ok2 {
						fromlist = t
					} else if lst, ok2 := a[3].(*object.List); ok2 {
						fromlist = &object.Tuple{V: lst.V}
					}
				}
				level := 0
				if len(a) >= 5 {
					if v, ok2 := toInt64(a[4]); ok2 {
						level = int(v)
					}
				}
				if kw != nil {
					if v, ok2 := kw.GetStr("fromlist"); ok2 {
						if t, ok3 := v.(*object.Tuple); ok3 {
							fromlist = t
						} else if lst, ok3 := v.(*object.List); ok3 {
							fromlist = &object.Tuple{V: lst.V}
						}
					}
					if v, ok2 := kw.GetStr("level"); ok2 {
						if n, ok3 := toInt64(v); ok3 {
							level = int(n)
						}
					}
				}
				return interp.importName(nameStr.V, globals, fromlist, level)
			},
		})
	}

	// exec(source_or_code, globals=None, locals=None)
	// Executes an already-compiled *object.Code object.  String source
	// is not supported (goipy has no Go-side Python compiler).
	if _, ok := b.GetStr("exec"); !ok {
		b.SetStr("exec", &object.BuiltinFunc{
			Name: "exec",
			Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
				interp := ii.(*Interp)
				if len(a) < 1 {
					return nil, object.Errorf(interp.typeErr, "exec() requires at least 1 argument")
				}
				code, ok2 := a[0].(*object.Code)
				if !ok2 {
					// String source compilation requires a Python parser;
					// goipy ships .pyc-only. Raise SyntaxError so user code
					// that catches it (the documented compile-time class)
					// runs the right except branch.
					return nil, object.Errorf(interp.syntaxErr,
						"exec() with string source requires a Python parser; goipy only runs compiled .pyc")
				}
				var globals *object.Dict
				if len(a) >= 2 {
					if d, ok3 := a[1].(*object.Dict); ok3 {
						globals = d
					}
				}
				if globals == nil {
					globals = object.NewDict()
					globals.SetStr("__builtins__", interp.Builtins)
				}
				var locals *object.Dict
				if len(a) >= 3 {
					if d, ok3 := a[2].(*object.Dict); ok3 {
						locals = d
					}
				}
				if locals == nil {
					locals = globals
				}
				frame := NewFrame(code, globals, interp.Builtins, locals)
				_, err := interp.runFrame(frame)
				if err != nil {
					return nil, err
				}
				return object.None, nil
			},
		})
	}

	// eval(expression, globals=None, locals=None)
	// Evaluates an already-compiled *object.Code object and returns its result.
	if _, ok := b.GetStr("eval"); !ok {
		b.SetStr("eval", &object.BuiltinFunc{
			Name: "eval",
			Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
				interp := ii.(*Interp)
				if len(a) < 1 {
					return nil, object.Errorf(interp.typeErr, "eval() requires at least 1 argument")
				}
				code, ok2 := a[0].(*object.Code)
				if !ok2 {
					return nil, object.Errorf(interp.syntaxErr,
						"eval() with string expression requires a Python parser; goipy only runs compiled .pyc")
				}
				var globals *object.Dict
				if len(a) >= 2 {
					if d, ok3 := a[1].(*object.Dict); ok3 {
						globals = d
					}
				}
				if globals == nil {
					globals = object.NewDict()
					globals.SetStr("__builtins__", interp.Builtins)
				}
				var locals *object.Dict
				if len(a) >= 3 {
					if d, ok3 := a[2].(*object.Dict); ok3 {
						locals = d
					}
				}
				if locals == nil {
					locals = globals
				}
				frame := NewFrame(code, globals, interp.Builtins, locals)
				result, err := interp.runFrame(frame)
				if err != nil {
					return nil, err
				}
				if result == nil {
					return object.None, nil
				}
				return result, nil
			},
		})
	}

	// compile(source, filename, mode, flags=0, dont_inherit=False, optimize=-1)
	// goipy has no Go-side Python compiler; raises SyntaxError, matching
	// the class user code expects from compile() on bad input.
	if _, ok := b.GetStr("compile"); !ok {
		b.SetStr("compile", &object.BuiltinFunc{
			Name: "compile",
			Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				interp := ii.(*Interp)
				return nil, object.Errorf(interp.syntaxErr,
					"compile() requires a Python parser; goipy only runs compiled .pyc")
			},
		})
	}

	return &object.Module{Name: "builtins", Dict: b}
}

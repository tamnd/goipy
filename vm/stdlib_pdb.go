package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildPdb() *object.Module {
	m := &object.Module{Name: "pdb", Dict: object.NewDict()}

	// Resolve bdb module so Pdb inherits the exact same class pointer that Python
	// code sees when it does `from bdb import Bdb`.
	bdbMod, _ := i.loadModule("bdb")

	var bdbCls *object.Class
	var bdbQuitCls *object.Class
	if bdbMod != nil {
		if v, ok := bdbMod.Dict.GetStr("Bdb"); ok {
			bdbCls, _ = v.(*object.Class)
		}
		if v, ok := bdbMod.Dict.GetStr("BdbQuit"); ok {
			bdbQuitCls, _ = v.(*object.Class)
		}
	}

	// ── Restart exception ──────────────────────────────────────────────────
	// Raised to restart a pdb session (e.g. via the "restart" command).
	restartCls := &object.Class{
		Name:  "Restart",
		Dict:  object.NewDict(),
		Bases: []*object.Class{i.exception},
	}
	m.Dict.SetStr("Restart", restartCls)

	// ── Default backend ────────────────────────────────────────────────────
	defaultBackend := "monitoring"

	// ── Pdb class ──────────────────────────────────────────────────────────
	pdbCls := &object.Class{Name: "Pdb", Dict: object.NewDict()}
	if bdbCls != nil {
		pdbCls.Bases = []*object.Class{bdbCls}
	}

	// Pdb.__init__(self, completekey='tab', stdin=None, stdout=None,
	//              skip=None, nosigint=False, readrc=True,
	//              mode=None, backend=None, colorize=False)
	pdbCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)

			// Delegate to Bdb.__init__ for breaks/quitting/_fncache/_skip.
			if bdbCls != nil {
				if v, ok := bdbCls.Dict.GetStr("__init__"); ok {
					if bdbInit, ok2 := v.(*object.BuiltinFunc); ok2 {
						bdbKw := object.NewDict()
						if kw != nil {
							if sv, ok3 := kw.GetStr("skip"); ok3 {
								bdbKw.SetStr("skip", sv)
							}
						}
						if _, err := bdbInit.Call(nil, []object.Object{inst}, bdbKw); err != nil {
							return nil, err
						}
					}
				}
			}

			// Pdb-specific attributes.
			inst.Dict.SetStr("nosigint", object.BoolOf(false))
			inst.Dict.SetStr("readrc", object.BoolOf(true))
			inst.Dict.SetStr("colorize", object.BoolOf(false))
			inst.Dict.SetStr("mode", object.None)
			inst.Dict.SetStr("stdin", object.None)
			inst.Dict.SetStr("stdout", object.None)

			if kw != nil {
				for _, attr := range []string{
					"nosigint", "readrc", "colorize", "mode", "stdin", "stdout",
				} {
					if v, ok := kw.GetStr(attr); ok {
						inst.Dict.SetStr(attr, v)
					}
				}
			}
			return object.None, nil
		},
	})

	// Pdb.set_trace() — no-op (goipy has no sys.settrace)
	pdbCls.Dict.SetStr("set_trace", &object.BuiltinFunc{
		Name: "set_trace",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("Pdb", pdbCls)

	// ── newPdb: create and initialize a Pdb instance ───────────────────────
	newPdb := func(ii any) (*object.Instance, error) {
		interp := ii.(*Interp)
		inst := &object.Instance{Class: pdbCls, Dict: object.NewDict()}
		initFn, err := interp.getAttr(inst, "__init__")
		if err != nil {
			return nil, err
		}
		if _, err = interp.callObject(initFn, []object.Object{}, nil); err != nil {
			return nil, err
		}
		return inst, nil
	}

	// catchBdbQuit wraps err: if it's a BdbQuit, return (None, nil); else pass through.
	catchBdbQuit := func(err error) (object.Object, error) {
		if bdbQuitCls == nil {
			return nil, err
		}
		if exc, ok := err.(*object.Exception); ok {
			if object.IsSubclass(exc.Class, bdbQuitCls) {
				return object.None, nil
			}
		}
		return nil, err
	}

	// ── set_trace(*, header=None, commands=None) ───────────────────────────
	m.Dict.SetStr("set_trace", &object.BuiltinFunc{
		Name: "set_trace",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── set_trace_async(*, header=None, commands=None) ─────────────────────
	m.Dict.SetStr("set_trace_async", &object.BuiltinFunc{
		Name: "set_trace_async",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── runcall(function, *args, **kwds) ───────────────────────────────────
	m.Dict.SetStr("runcall", &object.BuiltinFunc{
		Name: "runcall",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "runcall() requires function argument")
			}
			interp := ii.(*Interp)
			fn := a[0]
			fnArgs := a[1:]

			pdbInst, err := newPdb(ii)
			if err != nil {
				return nil, err
			}
			runcallMethod, err := interp.getAttr(pdbInst, "runcall")
			if err != nil {
				return nil, err
			}
			result, rerr := interp.callObject(runcallMethod, append([]object.Object{fn}, fnArgs...), kw)
			if rerr != nil {
				return catchBdbQuit(rerr)
			}
			return result, nil
		},
	})

	// ── run(statement, globals=None, locals=None) ──────────────────────────
	// Stub: executing a string statement requires compile/exec support.
	m.Dict.SetStr("run", &object.BuiltinFunc{
		Name: "run",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── runeval(expression, globals=None, locals=None) ─────────────────────
	m.Dict.SetStr("runeval", &object.BuiltinFunc{
		Name: "runeval",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── post_mortem(t=None) ────────────────────────────────────────────────
	m.Dict.SetStr("post_mortem", &object.BuiltinFunc{
		Name: "post_mortem",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── pm() ──────────────────────────────────────────────────────────────
	m.Dict.SetStr("pm", &object.BuiltinFunc{
		Name: "pm",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── get_default_backend() ──────────────────────────────────────────────
	m.Dict.SetStr("get_default_backend", &object.BuiltinFunc{
		Name: "get_default_backend",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: defaultBackend}, nil
		},
	})

	// ── set_default_backend(backend) ──────────────────────────────────────
	m.Dict.SetStr("set_default_backend", &object.BuiltinFunc{
		Name: "set_default_backend",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "set_default_backend() requires backend argument")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "set_default_backend() backend must be a string")
			}
			if s.V != "settrace" && s.V != "monitoring" {
				return nil, object.Errorf(i.valueErr, "set_default_backend() backend must be 'settrace' or 'monitoring'")
			}
			defaultBackend = s.V
			return object.None, nil
		},
	})

	return m
}

package vm

import (
	"io"
	"runtime"

	"github.com/tamnd/goipy/object"
)

// buildSys exposes a read-only view of interpreter state as the `sys`
// module. Writes to sys.argv/sys.path mutate the underlying slices;
// re-assigning the module attribute itself (e.g. `sys.stdout = x`)
// is not supported.
func (i *Interp) buildSys() *object.Module {
	m := &object.Module{Name: "sys", Dict: object.NewDict()}

	argv := &object.List{}
	for _, s := range i.Argv {
		argv.V = append(argv.V, &object.Str{V: s})
	}
	m.Dict.SetStr("argv", argv)

	path := &object.List{}
	for _, s := range i.SearchPath {
		path.V = append(path.V, &object.Str{V: s})
	}
	m.Dict.SetStr("path", path)

	modules := object.NewDict()
	modules.SetStr("sys", m)
	for name, mod := range i.modules {
		modules.SetStr(name, mod)
	}
	m.Dict.SetStr("modules", modules)

	verInfo := &object.Tuple{V: []object.Object{
		object.NewInt(3), object.NewInt(14), object.NewInt(0),
		&object.Str{V: "final"}, object.NewInt(0),
	}}
	m.Dict.SetStr("version_info", verInfo)
	m.Dict.SetStr("version", &object.Str{V: "3.14.0 (goipy)"})
	m.Dict.SetStr("platform", &object.Str{V: runtime.GOOS})
	m.Dict.SetStr("byteorder", &object.Str{V: "little"})
	m.Dict.SetStr("maxsize", object.NewInt(1<<63-1))
	m.Dict.SetStr("executable", &object.Str{V: "goipy"})
	m.Dict.SetStr("implementation", &object.Str{V: "goipy"})

	m.Dict.SetStr("stdout", &object.TextStream{Name: "stdout", W: i.Stdout})
	m.Dict.SetStr("stderr", &object.TextStream{Name: "stderr", W: i.Stderr})

	m.Dict.SetStr("exc_info", &object.BuiltinFunc{Name: "exc_info", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		for f := i.curFrame; f != nil; f = f.Back {
			if f.ExcInfo != nil {
				e := f.ExcInfo
				return &object.Tuple{V: []object.Object{e.Class, e, object.None}}, nil
			}
		}
		return &object.Tuple{V: []object.Object{object.None, object.None, object.None}}, nil
	}})

	m.Dict.SetStr("getrecursionlimit", &object.BuiltinFunc{Name: "getrecursionlimit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(i.MaxDepth)), nil
	}})
	m.Dict.SetStr("setrecursionlimit", &object.BuiltinFunc{Name: "setrecursionlimit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "setrecursionlimit() takes exactly one argument")
		}
		n, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "integer expected")
		}
		i.MaxDepth = int(n)
		return object.None, nil
	}})

	return m
}

// textStreamAttr dispatches attribute access on a *object.TextStream.
// Only the write-side API is exposed — these streams are not readable.
func textStreamAttr(i *Interp, ts *object.TextStream, name string) (object.Object, bool) {
	switch name {
	case "name":
		return &object.Str{V: "<" + ts.Name + ">"}, true
	case "mode":
		return &object.Str{V: "w"}, true
	case "closed":
		return object.False, true
	case "encoding":
		return &object.Str{V: "utf-8"}, true
	case "write":
		return &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) != 1 {
				return nil, object.Errorf(i.typeErr, "write() takes exactly one argument")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "write() argument must be str")
			}
			w, ok := ts.W.(io.Writer)
			if !ok {
				return object.NewInt(0), nil
			}
			n, _ := w.Write([]byte(s.V))
			return object.NewInt(int64(n)), nil
		}}, true
	case "flush":
		return &object.BuiltinFunc{Name: "flush", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}, true
	}
	return nil, false
}

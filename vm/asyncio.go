package vm

import (
	"os"
	"path/filepath"

	"github.com/tamnd/goipy/marshal"
	"github.com/tamnd/goipy/object"
)

// importModule resolves a top-level module name. Lookup order:
//  1. sys.modules cache (i.modules)
//  2. built-in registry (asyncio, …)
//  3. filesystem: {name}.pyc on i.SearchPath
//
// 7a only resolves top-level single-module imports; packages and
// relative imports are the subject of 7b.
func (i *Interp) importModule(name string) (*object.Module, error) {
	if i.modules == nil {
		i.modules = map[string]*object.Module{}
	}
	if m, ok := i.modules[name]; ok {
		return m, nil
	}
	if m, ok := i.builtinModule(name); ok {
		i.modules[name] = m
		return m, nil
	}
	for _, dir := range i.SearchPath {
		path := filepath.Join(dir, name+".pyc")
		if _, err := os.Stat(path); err != nil {
			continue
		}
		code, err := marshal.LoadPyc(path)
		if err != nil {
			return nil, object.Errorf(i.importErr, "cannot load %s: %v", path, err)
		}
		m, err := i.execModule(name, code)
		if err != nil {
			return nil, err
		}
		i.modules[name] = m
		return m, nil
	}
	return nil, object.Errorf(i.importErr, "No module named '%s'", name)
}

// builtinModule returns VM-provided modules that don't need a .pyc.
func (i *Interp) builtinModule(name string) (*object.Module, bool) {
	switch name {
	case "asyncio":
		return i.buildAsyncio(), true
	}
	return nil, false
}

// execModule runs a module-level code object in its own globals and
// returns a Module whose dict is the resulting namespace. The module is
// inserted into sys.modules before execution so that circular imports
// see a partially-initialized module rather than re-entering the loader.
func (i *Interp) execModule(name string, code *object.Code) (*object.Module, error) {
	globals := object.NewDict()
	globals.SetStr("__name__", &object.Str{V: name})
	globals.SetStr("__builtins__", i.Builtins)
	m := &object.Module{Name: name, Dict: globals}
	i.modules[name] = m
	frame := NewFrame(code, globals, i.Builtins, globals)
	if _, err := i.runFrame(frame); err != nil {
		delete(i.modules, name)
		return nil, err
	}
	return m, nil
}

// buildAsyncio constructs a minimal asyncio module: run(coro), sleep(t),
// gather(*coros). The runtime has no real event loop; coroutines are
// driven synchronously to completion. This is enough for single-file
// async scripts that don't depend on concurrent I/O.
func (i *Interp) buildAsyncio() *object.Module {
	m := &object.Module{Name: "asyncio", Dict: object.NewDict()}

	run := &object.BuiltinFunc{Name: "run", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "asyncio.run() missing coroutine")
		}
		return i.driveCoroutine(a[0])
	}}
	m.Dict.SetStr("run", run)

	sleep := &object.BuiltinFunc{Name: "sleep", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var result object.Object = object.None
		if len(a) > 1 {
			result = a[1]
		}
		done := false
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if done {
				return nil, false, nil
			}
			done = true
			// Terminate immediately: SEND sees the iterator exhausted and
			// turns it into StopIteration(result), which becomes the await
			// expression's value.
			exc := object.NewException(i.stopIter, "")
			exc.Args = &object.Tuple{V: []object.Object{result}}
			return nil, false, exc
		}}, nil
	}}
	m.Dict.SetStr("sleep", sleep)

	gather := &object.BuiltinFunc{Name: "gather", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// No real concurrency — drive each awaitable to completion and
		// collect the results in order.
		results := make([]object.Object, len(a))
		for k, c := range a {
			v, err := i.driveCoroutine(c)
			if err != nil {
				return nil, err
			}
			results[k] = v
		}
		// gather() must itself be awaitable. Wrap as a one-shot iter that
		// immediately produces the results list via StopIteration.
		done := false
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if done {
				return nil, false, nil
			}
			done = true
			exc := object.NewException(i.stopIter, "")
			exc.Args = &object.Tuple{V: []object.Object{&object.List{V: results}}}
			return nil, false, exc
		}}, nil
	}}
	m.Dict.SetStr("gather", gather)

	return m
}

// driveCoroutine runs an awaitable (coroutine / generator / iter) to
// completion by repeatedly sending None. Returns the final value (the
// StopIteration .value) or any unhandled exception.
func (i *Interp) driveCoroutine(awaitable object.Object) (object.Object, error) {
	switch x := awaitable.(type) {
	case *object.Generator:
		for {
			_, err := i.resumeGenerator(x, object.None)
			if err != nil {
				if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, i.stopIter) {
					if exc.Args != nil && len(exc.Args.V) > 0 {
						return exc.Args.V[0], nil
					}
					return object.None, nil
				}
				return nil, err
			}
			// Yielded (no one to deliver to — keep driving).
		}
	case *object.Iter:
		for {
			v, ok, err := x.Next()
			if err != nil {
				if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, i.stopIter) {
					if exc.Args != nil && len(exc.Args.V) > 0 {
						return exc.Args.V[0], nil
					}
					return object.None, nil
				}
				return nil, err
			}
			if !ok {
				return object.None, nil
			}
			_ = v
		}
	}
	return nil, object.Errorf(i.typeErr, "cannot drive %s as a coroutine", object.TypeName(awaitable))
}

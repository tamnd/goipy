package vm

import (
	"github.com/tamnd/goipy/object"
)

// builtinModule returns VM-provided modules that don't need a .pyc.
func (i *Interp) builtinModule(name string) (*object.Module, bool) {
	switch name {
	case "asyncio":
		return i.buildAsyncio(), true
	case "importlib":
		return i.buildImportlib(), true
	case "functools":
		return i.buildFunctools(), true
	case "itertools":
		return i.buildItertools(), true
	case "collections":
		return i.buildCollections(), true
	case "operator":
		return i.buildOperator(), true
	case "math":
		return i.buildMath(), true
	case "heapq":
		return i.buildHeapq(), true
	case "bisect":
		return i.buildBisect(), true
	case "random":
		return i.buildRandom(), true
	case "json":
		return i.buildJSON(), true
	case "re":
		return i.buildRe(), true
	case "string":
		return i.buildStringMod(), true
	case "copy":
		return i.buildCopy(), true
	case "io":
		return i.buildIO(), true
	case "hashlib":
		return i.buildHashlib(), true
	case "base64":
		return i.buildBase64(), true
	case "textwrap":
		return i.buildTextwrap(), true
	case "struct":
		return i.buildStruct(), true
	case "csv":
		return i.buildCsv(), true
	case "urllib":
		return i.buildUrllib(), true
	case "urllib.parse":
		return i.buildUrllibParse(), true
	case "zlib":
		return i.buildZlib(), true
	case "binascii":
		return i.buildBinascii(), true
	case "hmac":
		return i.buildHmac(), true
	case "secrets":
		return i.buildSecrets(), true
	case "uuid":
		return i.buildUUID(), true
	}
	return nil, false
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

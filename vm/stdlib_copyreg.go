package vm

import (
	"sync"

	"github.com/tamnd/goipy/object"
)

// Global extension registry (module, name) <-> code
var (
	copyregMu         sync.Mutex
	copyregByCode     = map[int64][2]string{}       // code -> (module, name)
	copyregByName     = map[[2]string]int64{}        // (module, name) -> code
	copyregCache      = map[int64]object.Object{}    // code -> resolved object (cleared by clear_extension_cache)
)

func (i *Interp) buildCopyreg() *object.Module {
	m := &object.Module{Name: "copyreg", Dict: object.NewDict()}

	// dispatch_table: a plain Python dict mapping type -> reduce_fn
	dispatchTable := object.NewDict()
	m.Dict.SetStr("dispatch_table", dispatchTable)

	// pickle(ob_type, pickle_function, constructor_ob=None)
	pickleFunc := &object.BuiltinFunc{Name: "pickle", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "pickle() requires at least 2 arguments")
		}
		obType := args[0]
		pickleFn := args[1]
		var constructorOb object.Object
		if len(args) >= 3 {
			constructorOb = args[2]
		}
		if kwargs != nil {
			if v, ok := kwargs.GetStr("constructor_ob"); ok {
				constructorOb = v
			}
		}

		// Validate pickleFn is callable
		if !i.isCallable(pickleFn) {
			return nil, object.Errorf(i.typeErr, "reduction functions must be callable")
		}
		// Validate constructorOb if provided
		if constructorOb != nil && constructorOb != object.None {
			if !i.isCallable(constructorOb) {
				return nil, object.Errorf(i.typeErr, "constructors must be callable")
			}
		}

		// Register in dispatch_table
		if err := dispatchTable.Set(obType, pickleFn); err != nil {
			return nil, err
		}
		return object.None, nil
	}}
	m.Dict.SetStr("pickle", pickleFunc)

	// constructor(object) — validates callability
	constructorFunc := &object.BuiltinFunc{Name: "constructor", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "constructor() requires 1 argument")
		}
		if !i.isCallable(args[0]) {
			return nil, object.Errorf(i.typeErr, "constructors must be callable")
		}
		return object.None, nil
	}}
	m.Dict.SetStr("constructor", constructorFunc)

	// add_extension(module, name, code)
	addExtFunc := &object.BuiltinFunc{Name: "add_extension", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) < 3 {
			return nil, object.Errorf(i.typeErr, "add_extension() requires 3 arguments")
		}
		modStr, ok1 := args[0].(*object.Str)
		nameStr, ok2 := args[1].(*object.Str)
		codeInt, ok3 := args[2].(*object.Int)
		if !ok1 || !ok2 || !ok3 {
			return nil, object.Errorf(i.typeErr, "add_extension() requires (str, str, int)")
		}
		code := codeInt.Int64()
		if code <= 0 || code >= (1<<31) {
			return nil, object.Errorf(i.valueErr, "code out of range")
		}
		key := [2]string{modStr.V, nameStr.V}

		copyregMu.Lock()
		defer copyregMu.Unlock()

		if existCode, exists := copyregByName[key]; exists {
			if existCode != code {
				return nil, object.Errorf(i.valueErr,
					"(%s, %s) is already registered with code %d", modStr.V, nameStr.V, existCode)
			}
			// Idempotent for exact same triple
			return object.None, nil
		}
		if existKey, exists := copyregByCode[code]; exists {
			if existKey != key {
				return nil, object.Errorf(i.valueErr,
					"code %d is already in use for %s.%s", code, existKey[0], existKey[1])
			}
			return object.None, nil
		}
		copyregByCode[code] = key
		copyregByName[key] = code
		return object.None, nil
	}}
	m.Dict.SetStr("add_extension", addExtFunc)

	// remove_extension(module, name, code)
	removeExtFunc := &object.BuiltinFunc{Name: "remove_extension", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) < 3 {
			return nil, object.Errorf(i.typeErr, "remove_extension() requires 3 arguments")
		}
		modStr, ok1 := args[0].(*object.Str)
		nameStr, ok2 := args[1].(*object.Str)
		codeInt, ok3 := args[2].(*object.Int)
		if !ok1 || !ok2 || !ok3 {
			return nil, object.Errorf(i.typeErr, "remove_extension() requires (str, str, int)")
		}
		code := codeInt.Int64()
		key := [2]string{modStr.V, nameStr.V}

		copyregMu.Lock()
		defer copyregMu.Unlock()

		existCode, nameExists := copyregByName[key]
		existKey, codeExists := copyregByCode[code]

		if !nameExists && !codeExists {
			return nil, object.Errorf(i.valueErr,
				"(%s, %s, %d) is not registered", modStr.V, nameStr.V, code)
		}
		if nameExists && existCode != code {
			return nil, object.Errorf(i.valueErr,
				"(%s, %s) is registered with code %d, not %d", modStr.V, nameStr.V, existCode, code)
		}
		if codeExists && existKey != key {
			return nil, object.Errorf(i.valueErr,
				"code %d is registered for %s.%s, not %s.%s", code, existKey[0], existKey[1], modStr.V, nameStr.V)
		}

		delete(copyregByName, key)
		delete(copyregByCode, code)
		delete(copyregCache, code)
		return object.None, nil
	}}
	m.Dict.SetStr("remove_extension", removeExtFunc)

	// clear_extension_cache() — clears only the resolved-object cache
	clearCacheFunc := &object.BuiltinFunc{Name: "clear_extension_cache", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		copyregMu.Lock()
		defer copyregMu.Unlock()
		copyregCache = map[int64]object.Object{}
		return object.None, nil
	}}
	m.Dict.SetStr("clear_extension_cache", clearCacheFunc)

	return m
}

// isCallable returns true if obj can be called.
func (i *Interp) isCallable(obj object.Object) bool {
	switch obj.(type) {
	case *object.BuiltinFunc, *object.Function, *object.Class:
		return true
	}
	return false
}

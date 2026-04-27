//go:build windows

package vm

import "github.com/tamnd/goipy/object"

func (i *Interp) buildFaulthandler() *object.Module {
	m := &object.Module{Name: "faulthandler", Dict: object.NewDict()}
	var enabled bool

	m.Dict.SetStr("enable", &object.BuiltinFunc{Name: "enable",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			enabled = true
			return object.None, nil
		},
	})
	m.Dict.SetStr("disable", &object.BuiltinFunc{Name: "disable",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			enabled = false
			return object.None, nil
		},
	})
	m.Dict.SetStr("is_enabled", &object.BuiltinFunc{Name: "is_enabled",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(enabled), nil
		},
	})
	m.Dict.SetStr("dump_traceback", &object.BuiltinFunc{Name: "dump_traceback",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	m.Dict.SetStr("dump_c_stack", &object.BuiltinFunc{Name: "dump_c_stack",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	m.Dict.SetStr("dump_traceback_later", &object.BuiltinFunc{Name: "dump_traceback_later",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	m.Dict.SetStr("cancel_dump_traceback_later", &object.BuiltinFunc{Name: "cancel_dump_traceback_later",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	m.Dict.SetStr("register", &object.BuiltinFunc{Name: "register",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	m.Dict.SetStr("unregister", &object.BuiltinFunc{Name: "unregister",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(false), nil
		},
	})
	return m
}

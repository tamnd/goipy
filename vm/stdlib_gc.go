package vm

import (
	"runtime"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildGc() *object.Module {
	m := &object.Module{Name: "gc", Dict: object.NewDict()}

	type gcState struct {
		enabled    bool
		thresholds [3]int64
	}
	state := &gcState{enabled: true, thresholds: [3]int64{700, 10, 10}}

	m.Dict.SetStr("enable", &object.BuiltinFunc{
		Name: "enable",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			state.enabled = true
			return object.None, nil
		},
	})

	m.Dict.SetStr("disable", &object.BuiltinFunc{
		Name: "disable",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			state.enabled = false
			return object.None, nil
		},
	})

	m.Dict.SetStr("isenabled", &object.BuiltinFunc{
		Name: "isenabled",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if state.enabled {
				return object.True, nil
			}
			return object.False, nil
		},
	})

	m.Dict.SetStr("collect", &object.BuiltinFunc{
		Name: "collect",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			runtime.GC()
			return object.NewInt(0), nil
		},
	})

	m.Dict.SetStr("get_count", &object.BuiltinFunc{
		Name: "get_count",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			items := []object.Object{
				object.NewInt(0),
				object.NewInt(0),
				object.NewInt(0),
			}
			return &object.Tuple{V: items}, nil
		},
	})

	m.Dict.SetStr("get_threshold", &object.BuiltinFunc{
		Name: "get_threshold",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			items := []object.Object{
				object.NewInt(state.thresholds[0]),
				object.NewInt(state.thresholds[1]),
				object.NewInt(state.thresholds[2]),
			}
			return &object.Tuple{V: items}, nil
		},
	})

	m.Dict.SetStr("set_threshold", &object.BuiltinFunc{
		Name: "set_threshold",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				if v, ok := toInt64(a[0]); ok {
					state.thresholds[0] = v
				}
			}
			if len(a) >= 2 {
				if v, ok := toInt64(a[1]); ok {
					state.thresholds[1] = v
				}
			}
			if len(a) >= 3 {
				if v, ok := toInt64(a[2]); ok {
					state.thresholds[2] = v
				}
			}
			return object.None, nil
		},
	})

	m.Dict.SetStr("get_objects", &object.BuiltinFunc{
		Name: "get_objects",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	m.Dict.SetStr("is_tracked", &object.BuiltinFunc{
		Name: "is_tracked",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		},
	})

	m.Dict.SetStr("is_finalized", &object.BuiltinFunc{
		Name: "is_finalized",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		},
	})

	m.Dict.SetStr("freeze", &object.BuiltinFunc{
		Name: "freeze",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("unfreeze", &object.BuiltinFunc{
		Name: "unfreeze",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("get_freeze_count", &object.BuiltinFunc{
		Name: "get_freeze_count",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(0), nil
		},
	})

	m.Dict.SetStr("get_referents", &object.BuiltinFunc{
		Name: "get_referents",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	m.Dict.SetStr("get_referrers", &object.BuiltinFunc{
		Name: "get_referrers",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	m.Dict.SetStr("set_debug", &object.BuiltinFunc{
		Name: "set_debug",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("get_debug", &object.BuiltinFunc{
		Name: "get_debug",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(0), nil
		},
	})

	m.Dict.SetStr("callbacks", &object.List{V: []object.Object{}})

	// GC debug flags
	m.Dict.SetStr("DEBUG_STATS", object.NewInt(1))
	m.Dict.SetStr("DEBUG_COLLECTABLE", object.NewInt(2))
	m.Dict.SetStr("DEBUG_UNCOLLECTABLE", object.NewInt(4))
	m.Dict.SetStr("DEBUG_SAVEALL", object.NewInt(32))
	m.Dict.SetStr("DEBUG_LEAK", object.NewInt(38))

	return m
}

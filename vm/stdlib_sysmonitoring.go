package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildSysMonitoring() *object.Module {
	m := &object.Module{Name: "sys.monitoring", Dict: object.NewDict()}

	// Tool IDs
	m.Dict.SetStr("DEBUGGER_ID", object.NewInt(0))
	m.Dict.SetStr("COVERAGE_ID", object.NewInt(1))
	m.Dict.SetStr("PROFILER_ID", object.NewInt(2))
	m.Dict.SetStr("OPTIMIZER_ID", object.NewInt(5))

	// MISSING sentinel
	missingType := &object.Class{Name: "_Missing", Dict: object.NewDict()}
	missingInst := &object.Instance{Class: missingType, Dict: object.NewDict()}
	m.Dict.SetStr("MISSING", missingInst)

	// events namespace
	eventsClass := &object.Class{Name: "events", Dict: object.NewDict()}
	events := map[string]int64{
		"BRANCH":              1 << 0,
		"CALL":                1 << 1,
		"C_RAISE":             1 << 2,
		"C_RETURN":            1 << 3,
		"EXCEPTION_HANDLED":   1 << 4,
		"INSTRUCTION":         1 << 5,
		"JUMP":                1 << 6,
		"LINE":                1 << 7,
		"NO_EVENTS":           0,
		"PY_RESUME":           1 << 8,
		"PY_RETURN":           1 << 9,
		"PY_START":            1 << 10,
		"PY_THROW":            1 << 11,
		"PY_UNWIND":           1 << 12,
		"PY_YIELD":            1 << 13,
		"RAISE":               1 << 14,
		"RERAISE":             1 << 15,
		"STOP_ITERATION":      1 << 16,
	}
	for name, val := range events {
		eventsClass.Dict.SetStr(name, object.NewInt(val))
	}
	m.Dict.SetStr("events", eventsClass)

	// Per-tool state: events bitmask and callbacks
	type toolState struct {
		events    int64
		callbacks map[int64]object.Object
	}
	toolStates := map[int64]*toolState{}
	getOrCreate := func(toolID int64) *toolState {
		if ts, ok := toolStates[toolID]; ok {
			return ts
		}
		ts := &toolState{callbacks: map[int64]object.Object{}}
		toolStates[toolID] = ts
		return ts
	}

	m.Dict.SetStr("use_tool_id", &object.BuiltinFunc{
		Name: "use_tool_id",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			id, _ := toInt64(a[0])
			getOrCreate(id)
			return object.None, nil
		},
	})

	m.Dict.SetStr("free_tool_id", &object.BuiltinFunc{
		Name: "free_tool_id",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			id, _ := toInt64(a[0])
			delete(toolStates, id)
			return object.None, nil
		},
	})

	m.Dict.SetStr("get_tool", &object.BuiltinFunc{
		Name: "get_tool",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("set_events", &object.BuiltinFunc{
		Name: "set_events",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			id, _ := toInt64(a[0])
			ev, _ := toInt64(a[1])
			getOrCreate(id).events = ev
			return object.None, nil
		},
	})

	m.Dict.SetStr("get_events", &object.BuiltinFunc{
		Name: "get_events",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NewInt(0), nil
			}
			id, _ := toInt64(a[0])
			return object.NewInt(getOrCreate(id).events), nil
		},
	})

	m.Dict.SetStr("set_local_events", &object.BuiltinFunc{
		Name: "set_local_events",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("get_local_events", &object.BuiltinFunc{
		Name: "get_local_events",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(0), nil
		},
	})

	m.Dict.SetStr("register_callback", &object.BuiltinFunc{
		Name: "register_callback",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return object.None, nil
			}
			id, _ := toInt64(a[0])
			ev, _ := toInt64(a[1])
			ts := getOrCreate(id)
			old, ok := ts.callbacks[ev]
			if !ok {
				old = object.None
			}
			if a[2] == object.None {
				delete(ts.callbacks, ev)
			} else {
				ts.callbacks[ev] = a[2]
			}
			return old, nil
		},
	})

	m.Dict.SetStr("get_local_events", &object.BuiltinFunc{
		Name: "get_local_events",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(0), nil
		},
	})

	m.Dict.SetStr("_all_events", &object.BuiltinFunc{
		Name: "_all_events",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewDict(), nil
		},
	})

	return m
}

package vm

import (
	"math/big"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildReadline() *object.Module {
	m := &object.Module{Name: "readline", Dict: object.NewDict()}

	// In-memory state
	var (
		history       []string
		historyLength = -1
		completer     object.Object
		startupHook   object.Object
		preInputHook  object.Object
		displayHook   object.Object
		lineBuffer    string
		completerDelims = "\t\n !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	)

	_ = startupHook
	_ = preInputHook
	_ = displayHook

	m.Dict.SetStr("backend", &object.Str{V: "editline"})

	// --- Init file ---
	m.Dict.SetStr("parse_and_bind", &object.BuiltinFunc{Name: "parse_and_bind",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	m.Dict.SetStr("read_init_file", &object.BuiltinFunc{Name: "read_init_file",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "[Errno -1] Unknown error: -1")
		}})

	// --- Line buffer ---
	m.Dict.SetStr("get_line_buffer", &object.BuiltinFunc{Name: "get_line_buffer",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: lineBuffer}, nil
		}})

	m.Dict.SetStr("insert_text", &object.BuiltinFunc{Name: "insert_text",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 {
				if s, ok := a[0].(*object.Str); ok {
					lineBuffer += s.V
				}
			}
			return object.None, nil
		}})

	m.Dict.SetStr("redisplay", &object.BuiltinFunc{Name: "redisplay",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	// --- History file ---
	m.Dict.SetStr("read_history_file", &object.BuiltinFunc{Name: "read_history_file",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	m.Dict.SetStr("write_history_file", &object.BuiltinFunc{Name: "write_history_file",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	m.Dict.SetStr("append_history_file", &object.BuiltinFunc{Name: "append_history_file",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	m.Dict.SetStr("get_history_length", &object.BuiltinFunc{Name: "get_history_length",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj(int64(historyLength)), nil
		}})

	m.Dict.SetStr("set_history_length", &object.BuiltinFunc{Name: "set_history_length",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 {
				if n, ok := toInt(a[0]); ok {
					historyLength = int(n)
				}
			}
			return object.None, nil
		}})

	// --- History list ---
	m.Dict.SetStr("clear_history", &object.BuiltinFunc{Name: "clear_history",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			history = history[:0]
			return object.None, nil
		}})

	m.Dict.SetStr("get_current_history_length", &object.BuiltinFunc{Name: "get_current_history_length",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj(int64(len(history))), nil
		}})

	m.Dict.SetStr("get_history_item", &object.BuiltinFunc{Name: "get_history_item",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "get_history_item() missing argument")
			}
			n, ok := toInt(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "get_history_item() argument must be int")
			}
			idx := int(n) - 1 // 1-based
			if idx < 0 || idx >= len(history) {
				return object.None, nil
			}
			return &object.Str{V: history[idx]}, nil
		}})

	m.Dict.SetStr("remove_history_item", &object.BuiltinFunc{Name: "remove_history_item",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "remove_history_item() missing argument")
			}
			n, ok := toInt(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "remove_history_item() argument must be int")
			}
			idx := int(n) // 0-based
			if idx < 0 || idx >= len(history) {
				return nil, object.Errorf(i.valueErr, "remove_history_item(): index out of range")
			}
			history = append(history[:idx], history[idx+1:]...)
			return object.None, nil
		}})

	m.Dict.SetStr("replace_history_item", &object.BuiltinFunc{Name: "replace_history_item",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "replace_history_item() requires 2 arguments")
			}
			n, ok := toInt(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "replace_history_item() first argument must be int")
			}
			s, ok2 := a[1].(*object.Str)
			if !ok2 {
				return nil, object.Errorf(i.typeErr, "replace_history_item() second argument must be str")
			}
			idx := int(n) // 0-based
			if idx < 0 || idx >= len(history) {
				return nil, object.Errorf(i.valueErr, "replace_history_item(): index out of range")
			}
			history[idx] = s.V
			return object.None, nil
		}})

	m.Dict.SetStr("add_history", &object.BuiltinFunc{Name: "add_history",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 {
				if s, ok := a[0].(*object.Str); ok {
					history = append(history, s.V)
				}
			}
			return object.None, nil
		}})

	m.Dict.SetStr("set_auto_history", &object.BuiltinFunc{Name: "set_auto_history",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	// --- Startup hooks ---
	m.Dict.SetStr("set_startup_hook", &object.BuiltinFunc{Name: "set_startup_hook",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 && a[0] != object.None {
				startupHook = a[0]
			} else {
				startupHook = nil
			}
			return object.None, nil
		}})

	m.Dict.SetStr("set_pre_input_hook", &object.BuiltinFunc{Name: "set_pre_input_hook",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 && a[0] != object.None {
				preInputHook = a[0]
			} else {
				preInputHook = nil
			}
			return object.None, nil
		}})

	// --- Completion ---
	m.Dict.SetStr("set_completer", &object.BuiltinFunc{Name: "set_completer",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 && a[0] != object.None {
				completer = a[0]
			} else {
				completer = nil
			}
			return object.None, nil
		}})

	m.Dict.SetStr("get_completer", &object.BuiltinFunc{Name: "get_completer",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if completer == nil {
				return object.None, nil
			}
			return completer, nil
		}})

	m.Dict.SetStr("get_completion_type", &object.BuiltinFunc{Name: "get_completion_type",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj(0), nil
		}})

	m.Dict.SetStr("get_begidx", &object.BuiltinFunc{Name: "get_begidx",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj(0), nil
		}})

	m.Dict.SetStr("get_endidx", &object.BuiltinFunc{Name: "get_endidx",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj(0), nil
		}})

	m.Dict.SetStr("set_completer_delims", &object.BuiltinFunc{Name: "set_completer_delims",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 {
				if s, ok := a[0].(*object.Str); ok {
					completerDelims = s.V
				}
			}
			return object.None, nil
		}})

	m.Dict.SetStr("get_completer_delims", &object.BuiltinFunc{Name: "get_completer_delims",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: completerDelims}, nil
		}})

	m.Dict.SetStr("set_completion_display_matches_hook", &object.BuiltinFunc{Name: "set_completion_display_matches_hook",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 && a[0] != object.None {
				displayHook = a[0]
			} else {
				displayHook = nil
			}
			return object.None, nil
		}})

	return m
}

func intObj(n int64) *object.Int {
	return &object.Int{V: *new(big.Int).SetInt64(n)}
}

// toInt extracts an int64 value from an object.Int or object.Bool.
func toInt(o object.Object) (int64, bool) {
	switch v := o.(type) {
	case *object.Int:
		return v.V.Int64(), true
	case *object.Bool:
		if v.V {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

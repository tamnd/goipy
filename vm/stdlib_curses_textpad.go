package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildCursesTextpad constructs the curses.textpad submodule.
func (i *Interp) buildCursesTextpad() *object.Module {
	m := &object.Module{Name: "curses.textpad", Dict: object.NewDict()}

	// rectangle(win, uly, ulx, lry, lrx) → None
	m.Dict.SetStr("rectangle", &object.BuiltinFunc{Name: "rectangle", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// Textbox class
	textboxCls := &object.Class{Name: "Textbox", Dict: object.NewDict()}

	// __init__(self, win, insert_mode=False) → None
	textboxCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		// store the window reference
		if len(a) >= 2 {
			self.Dict.SetStr("win", a[1])
		}
		var insertMode object.Object = object.False
		if len(a) >= 3 {
			insertMode = a[2]
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("insert_mode"); ok2 {
				insertMode = v
			}
		}
		self.Dict.SetStr("insert_mode", insertMode)

		// edit(validate=None) → ''
		self.Dict.SetStr("edit", &object.BuiltinFunc{Name: "edit", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: ""}, nil
		}})

		// do_command(ch) → None
		self.Dict.SetStr("do_command", &object.BuiltinFunc{Name: "do_command", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

		// gather() → ''
		self.Dict.SetStr("gather", &object.BuiltinFunc{Name: "gather", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: ""}, nil
		}})

		return object.None, nil
	}})

	m.Dict.SetStr("Textbox", textboxCls)

	return m
}

package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildCursesPanel constructs the curses.panel submodule.
func (i *Interp) buildCursesPanel() *object.Module {
	m := &object.Module{Name: "curses.panel", Dict: object.NewDict()}

	// panel class
	panelCls := &object.Class{Name: "panel", Dict: object.NewDict()}

	// makePanelInst creates a panel instance wrapping the given window.
	makePanelInst := func(win object.Object) *object.Instance {
		inst := &object.Instance{Class: panelCls, Dict: object.NewDict()}

		none := func(name string) *object.BuiltinFunc {
			return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}}
		}

		inst.Dict.SetStr("bottom", none("bottom"))
		inst.Dict.SetStr("top", none("top"))
		inst.Dict.SetStr("above", &object.BuiltinFunc{Name: "above", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
		inst.Dict.SetStr("below", &object.BuiltinFunc{Name: "below", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
		inst.Dict.SetStr("window", &object.BuiltinFunc{Name: "window", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return win, nil
		}})
		inst.Dict.SetStr("replace", none("replace"))
		inst.Dict.SetStr("set_userptr", none("set_userptr"))
		inst.Dict.SetStr("userptr", &object.BuiltinFunc{Name: "userptr", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
		inst.Dict.SetStr("show", none("show"))
		inst.Dict.SetStr("hide", none("hide"))
		inst.Dict.SetStr("hidden", &object.BuiltinFunc{Name: "hidden", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		}})
		inst.Dict.SetStr("move", none("move"))

		return inst
	}

	// new_panel(win) → panel
	m.Dict.SetStr("new_panel", &object.BuiltinFunc{Name: "new_panel", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var win object.Object = object.None
		if len(a) >= 1 {
			win = a[0]
		}
		return makePanelInst(win), nil
	}})

	// top_panel() → None (no panel stack in stub)
	m.Dict.SetStr("top_panel", &object.BuiltinFunc{Name: "top_panel", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// bottom_panel() → None
	m.Dict.SetStr("bottom_panel", &object.BuiltinFunc{Name: "bottom_panel", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// update_panels() → None
	m.Dict.SetStr("update_panels", &object.BuiltinFunc{Name: "update_panels", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	m.Dict.SetStr("panel", panelCls)

	return m
}

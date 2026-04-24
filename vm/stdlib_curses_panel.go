package vm

import (
	"fmt"

	"github.com/tamnd/goipy/object"
)

// panelState holds the mutable state for one panel instance.
type panelState struct {
	win     object.Object
	userptr object.Object
	hidden  bool
	y, x    int
}

// buildCursesPanel constructs the curses.panel submodule.
func (i *Interp) buildCursesPanel() *object.Module {
	m := &object.Module{Name: "curses.panel", Dict: object.NewDict()}

	// Global panel stack: index 0 = bottom, last = top.
	var stack []*object.Instance

	panelCls := &object.Class{Name: "panel", Dict: object.NewDict()}

	// makePanel creates a new panel instance wrapping win and appends it to
	// the top of the stack.
	makePanel := func(win object.Object) *object.Instance {
		ps := &panelState{win: win, userptr: object.None}

		// Read initial position from window's getbegyx if available.
		if winInst, ok := win.(*object.Instance); ok {
			if fn, ok2 := winInst.Dict.GetStr("getbegyx"); ok2 {
				if bf, ok3 := fn.(*object.BuiltinFunc); ok3 {
					if res, err := bf.Call(nil, []object.Object{winInst}, nil); err == nil {
						if tup, ok4 := res.(*object.Tuple); ok4 && len(tup.V) == 2 {
							y, _ := toInt64(tup.V[0])
							x, _ := toInt64(tup.V[1])
							ps.y, ps.x = int(y), int(x)
						}
					}
				}
			}
		}

		inst := &object.Instance{Class: panelCls, Dict: object.NewDict()}

		// findIdx returns the index of inst in the stack, or -1.
		findIdx := func() int {
			for k, p := range stack {
				if p == inst {
					return k
				}
			}
			return -1
		}

		// bottom() — move this panel to index 0.
		inst.Dict.SetStr("bottom", &object.BuiltinFunc{Name: "bottom", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			idx := findIdx()
			if idx > 0 {
				// Remove from current position then prepend.
				rest := make([]*object.Instance, 0, len(stack))
				rest = append(rest, inst)
				rest = append(rest, stack[:idx]...)
				rest = append(rest, stack[idx+1:]...)
				stack = rest
			}
			return object.None, nil
		}})

		// top() — move this panel to last index.
		inst.Dict.SetStr("top", &object.BuiltinFunc{Name: "top", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			idx := findIdx()
			if idx >= 0 && idx < len(stack)-1 {
				rest := make([]*object.Instance, 0, len(stack))
				rest = append(rest, stack[:idx]...)
				rest = append(rest, stack[idx+1:]...)
				rest = append(rest, inst)
				stack = rest
			}
			return object.None, nil
		}})

		// above() — panel directly above this one, or None.
		inst.Dict.SetStr("above", &object.BuiltinFunc{Name: "above", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			idx := findIdx()
			if idx >= 0 && idx < len(stack)-1 {
				return stack[idx+1], nil
			}
			return object.None, nil
		}})

		// below() — panel directly below this one, or None.
		inst.Dict.SetStr("below", &object.BuiltinFunc{Name: "below", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			idx := findIdx()
			if idx > 0 {
				return stack[idx-1], nil
			}
			return object.None, nil
		}})

		// window() — return the associated curses window.
		inst.Dict.SetStr("window", &object.BuiltinFunc{Name: "window", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return ps.win, nil
		}})

		// replace(win) — replace the window. a[0]=new window (no self for inst-dict funcs).
		inst.Dict.SetStr("replace", &object.BuiltinFunc{Name: "replace", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				ps.win = a[0]
			}
			return object.None, nil
		}})

		// set_userptr(obj) — store user pointer. a[0]=obj (no self for inst-dict funcs).
		inst.Dict.SetStr("set_userptr", &object.BuiltinFunc{Name: "set_userptr", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				ps.userptr = a[0]
			}
			return object.None, nil
		}})

		// userptr() — retrieve user pointer.
		inst.Dict.SetStr("userptr", &object.BuiltinFunc{Name: "userptr", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return ps.userptr, nil
		}})

		// show() — make panel visible.
		inst.Dict.SetStr("show", &object.BuiltinFunc{Name: "show", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ps.hidden = false
			return object.None, nil
		}})

		// hide() — make panel invisible.
		inst.Dict.SetStr("hide", &object.BuiltinFunc{Name: "hide", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ps.hidden = true
			return object.None, nil
		}})

		// hidden() — return whether the panel is hidden.
		inst.Dict.SetStr("hidden", &object.BuiltinFunc{Name: "hidden", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(ps.hidden), nil
		}})

		// move(y, x) — move panel position. a[0]=y, a[1]=x (no self for inst-dict funcs).
		inst.Dict.SetStr("move", &object.BuiltinFunc{Name: "move", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 2 {
				y, _ := toInt64(a[0])
				x, _ := toInt64(a[1])
				ps.y, ps.x = int(y), int(x)
				// Call win.mvwin(y, x) if available.
				if winInst, ok2 := ps.win.(*object.Instance); ok2 {
					if fn, ok3 := winInst.Dict.GetStr("mvwin"); ok3 {
						if bf, ok4 := fn.(*object.BuiltinFunc); ok4 {
							bf.Call(nil, []object.Object{winInst, a[0], a[1]}, nil) //nolint:errcheck
						}
					}
				}
			}
			return object.None, nil
		}})

		// Add to top of stack.
		stack = append(stack, inst)
		return inst
	}

	// new_panel(win) — create and return a new panel wrapping win.
	m.Dict.SetStr("new_panel", &object.BuiltinFunc{Name: "new_panel", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("new_panel() requires a window argument")
		}
		return makePanel(a[0]), nil
	}})

	// top_panel() — return the topmost visible panel, or topmost overall.
	m.Dict.SetStr("top_panel", &object.BuiltinFunc{Name: "top_panel", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		for k := len(stack) - 1; k >= 0; k-- {
			p := stack[k]
			if fn, ok := p.Dict.GetStr("hidden"); ok {
				if bf, ok2 := fn.(*object.BuiltinFunc); ok2 {
					res, _ := bf.Call(nil, []object.Object{p}, nil)
					if !object.Truthy(res) {
						return p, nil
					}
				}
			}
		}
		// No visible panel — return topmost if any.
		if len(stack) > 0 {
			return stack[len(stack)-1], nil
		}
		return object.None, nil
	}})

	// bottom_panel() — return the bottommost panel.
	m.Dict.SetStr("bottom_panel", &object.BuiltinFunc{Name: "bottom_panel", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if len(stack) > 0 {
			return stack[0], nil
		}
		return object.None, nil
	}})

	// update_panels() — no-op stub.
	m.Dict.SetStr("update_panels", &object.BuiltinFunc{Name: "update_panels", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	m.Dict.SetStr("panel", panelCls)
	return m
}

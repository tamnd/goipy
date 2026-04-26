package vm

import (
	"github.com/tamnd/goipy/object"
)

// webbrowserLaunch reports that the browser was "opened" without actually
// spawning a system browser process. Goipy is a scripting/testing interpreter;
// launching the OS browser from CI or test suites is never desirable.
func webbrowserLaunch(_ string) bool {
	return true
}

// ── webbrowser ────────────────────────────────────────────────────────────────

func (i *Interp) buildWebbrowser() *object.Module {
	m := &object.Module{Name: "webbrowser", Dict: object.NewDict()}

	// Error exception
	wbErrCls := &object.Class{
		Name:  "Error",
		Dict:  object.NewDict(),
		Bases: []*object.Class{i.exception},
	}
	m.Dict.SetStr("Error", wbErrCls)

	// Controller class
	ctrlCls := &object.Class{Name: "Controller", Dict: object.NewDict()}

	ctrlCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "Controller.__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		self := a[0].(*object.Instance)
		if len(a) > 1 {
			self.Dict.SetStr("name", a[1])
		} else {
			self.Dict.SetStr("name", &object.Str{V: ""})
		}
		return object.None, nil
	}})

	openURL := func(a []object.Object) object.Object {
		url := ""
		if len(a) > 1 {
			if s, ok := a[1].(*object.Str); ok {
				url = s.V
			}
		}
		if webbrowserLaunch(url) {
			return object.True
		}
		return object.False
	}

	ctrlCls.Dict.SetStr("open", &object.BuiltinFunc{Name: "Controller.open", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return openURL(a), nil
	}})
	ctrlCls.Dict.SetStr("open_new", &object.BuiltinFunc{Name: "Controller.open_new", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return openURL(a), nil
	}})
	ctrlCls.Dict.SetStr("open_new_tab", &object.BuiltinFunc{Name: "Controller.open_new_tab", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return openURL(a), nil
	}})

	m.Dict.SetStr("Controller", ctrlCls)

	// Internal registry stored on the module
	registry := object.NewDict()
	preferred := &object.List{V: []object.Object{}}
	m.Dict.SetStr("_registry", registry)
	m.Dict.SetStr("_preferred", preferred)

	// register(name, constructor, instance=None, *, preferred=False)
	m.Dict.SetStr("register", &object.BuiltinFunc{Name: "register", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "register() requires at least 2 positional arguments")
		}
		nameStr := ""
		if s, ok := a[0].(*object.Str); ok {
			nameStr = s.V
		}
		var inst object.Object
		if len(a) > 2 && a[2] != object.None {
			inst = a[2]
		} else if a[1] != object.None {
			interp := ii.(*Interp)
			var err error
			inst, err = interp.callObject(a[1], nil, nil)
			if err != nil {
				return nil, err
			}
		}
		if inst != nil {
			registry.SetStr(nameStr, inst)
		}
		isPreferred := false
		if kw != nil {
			if pv, ok := kw.GetStr("preferred"); ok {
				isPreferred = pv != object.False && pv != object.None
			}
		}
		if isPreferred {
			newV := make([]object.Object, 0, len(preferred.V)+1)
			newV = append(newV, &object.Str{V: nameStr})
			newV = append(newV, preferred.V...)
			preferred.V = newV
		}
		return object.None, nil
	}})

	// get(using=None) -> controller
	m.Dict.SetStr("get", &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		using := object.Object(object.None)
		if len(a) > 0 {
			using = a[0]
		}
		if using != object.None {
			nameStr := ""
			if s, ok := using.(*object.Str); ok {
				nameStr = s.V
			}
			if inst, ok := registry.GetStr(nameStr); ok && inst != object.None {
				return inst, nil
			}
			return nil, object.Errorf(wbErrCls, "%s not found", nameStr)
		}
		for _, n := range preferred.V {
			if ns, ok := n.(*object.Str); ok {
				if inst, ok2 := registry.GetStr(ns.V); ok2 && inst != object.None {
					return inst, nil
				}
			}
		}
		ctrl := &object.Instance{Class: ctrlCls, Dict: object.NewDict()}
		ctrl.Dict.SetStr("name", &object.Str{V: "default"})
		return ctrl, nil
	}})

	// Module-level open/open_new/open_new_tab: try preferred controller first,
	// then fall back to OS launcher.
	wbOpen := func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		url := ""
		if len(a) > 0 {
			if s, ok := a[0].(*object.Str); ok {
				url = s.V
			}
		}
		for _, n := range preferred.V {
			if ns, ok := n.(*object.Str); ok {
				if inst, ok2 := registry.GetStr(ns.V); ok2 && inst != object.None {
					interp := ii.(*Interp)
					fn, err := interp.getAttr(inst, "open")
					if err == nil {
						res, err2 := interp.callObject(fn, []object.Object{&object.Str{V: url}}, nil)
						if err2 != nil {
							return object.False, nil
						}
						return res, nil
					}
				}
			}
		}
		if webbrowserLaunch(url) {
			return object.True, nil
		}
		return object.False, nil
	}

	m.Dict.SetStr("open", &object.BuiltinFunc{Name: "open", Call: wbOpen})
	m.Dict.SetStr("open_new", &object.BuiltinFunc{Name: "open_new", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return wbOpen(ii, a, kw)
	}})
	m.Dict.SetStr("open_new_tab", &object.BuiltinFunc{Name: "open_new_tab", Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return wbOpen(ii, a, kw)
	}})

	return m
}

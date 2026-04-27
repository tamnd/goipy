package vm

import (
	"fmt"
	"strings"
	"sync"

	"github.com/tamnd/goipy/object"
)

// ─────────────────────────────────────────────────────────────────────────────
// call class — represents a recorded call (args, kwargs)
// ─────────────────────────────────────────────────────────────────────────────

func buildCallClass(i *Interp) *object.Class {
	cls := &object.Class{Name: "call", Dict: object.NewDict()}

	makeCall := func(args []object.Object, kwargs *object.Dict) *object.Instance {
		inst := &object.Instance{Class: cls, Dict: object.NewDict()}
		inst.Dict.SetStr("_args", &object.Tuple{V: args})
		if kwargs == nil {
			kwargs = object.NewDict()
		}
		inst.Dict.SetStr("_kwargs", kwargs)
		return inst
	}

	cls.Dict.SetStr("__new__", &object.BuiltinFunc{
		Name: "__new__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// a[0] is the class, skip it
			var args []object.Object
			if len(a) > 1 {
				args = a[1:]
			}
			return makeCall(args, kw), nil
		},
	})

	cls.Dict.SetStr("__eq__", &object.BuiltinFunc{
		Name: "__eq__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.BoolOf(false), nil
			}
			self, ok1 := a[0].(*object.Instance)
			other, ok2 := a[1].(*object.Instance)
			if !ok1 || !ok2 || self.Class != cls || other.Class != cls {
				return object.BoolOf(false), nil
			}
			sa, _ := self.Dict.GetStr("_args")
			oa, _ := other.Dict.GetStr("_args")
			sk, _ := self.Dict.GetStr("_kwargs")
			ok, _ := other.Dict.GetStr("_kwargs")
			argsEq, _ := object.Eq(sa, oa)
			kwargsEq, _ := object.Eq(sk, ok)
			return object.BoolOf(argsEq && kwargsEq), nil
		},
	})

	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "call()"}, nil
			}
			self := a[0].(*object.Instance)
			argsObj, _ := self.Dict.GetStr("_args")
			kwargsObj, _ := self.Dict.GetStr("_kwargs")
			var parts []string
			if t, ok := argsObj.(*object.Tuple); ok {
				for _, v := range t.V {
					parts = append(parts, object.Repr(v))
				}
			}
			if d, ok := kwargsObj.(*object.Dict); ok {
				keys, vals := d.Items()
				for idx, k := range keys {
					ks := object.Str_(k)
					parts = append(parts, fmt.Sprintf("%s=%s", ks, object.Repr(vals[idx])))
				}
			}
			return &object.Str{V: "call(" + strings.Join(parts, ", ") + ")"}, nil
		},
	})

	// __getattr__ — call.method_name(args) for method_calls comparison
	cls.Dict.SetStr("__getattr__", &object.BuiltinFunc{
		Name: "__getattr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			name := object.Str_(a[1])
			builder := &object.BuiltinFunc{
				Name: name,
				Call: func(_ any, bArgs []object.Object, bKw *object.Dict) (object.Object, error) {
					inst := makeCall(bArgs, bKw)
					inst.Dict.SetStr("_name", &object.Str{V: name})
					return inst, nil
				},
			}
			return builder, nil
		},
	})

	return cls
}

// ─────────────────────────────────────────────────────────────────────────────
// ANY — equality wildcard
// ─────────────────────────────────────────────────────────────────────────────

func buildAnyObject(i *Interp) object.Object {
	cls := &object.Class{Name: "_ANY", Dict: object.NewDict()}
	cls.Dict.SetStr("__eq__", &object.BuiltinFunc{
		Name: "__eq__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(true), nil
		},
	})
	cls.Dict.SetStr("__ne__", &object.BuiltinFunc{
		Name: "__ne__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(false), nil
		},
	})
	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "<ANY>"}, nil
		},
	})
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	return inst
}

// ─────────────────────────────────────────────────────────────────────────────
// DEFAULT sentinel
// ─────────────────────────────────────────────────────────────────────────────

func buildDefaultObject() object.Object {
	cls := &object.Class{Name: "_DEFAULT", Dict: object.NewDict()}
	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "DEFAULT"}, nil
		},
	})
	return &object.Instance{Class: cls, Dict: object.NewDict()}
}

// ─────────────────────────────────────────────────────────────────────────────
// sentinel — unique named singletons
// ─────────────────────────────────────────────────────────────────────────────

func buildSentinel() object.Object {
	var mu sync.Mutex
	registry := map[string]object.Object{}

	sentCls := &object.Class{Name: "_SentinelObject", Dict: object.NewDict()}
	sentCls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "sentinel.?"}, nil
			}
			inst := a[0].(*object.Instance)
			n, _ := inst.Dict.GetStr("_name")
			return &object.Str{V: "sentinel." + object.Str_(n)}, nil
		},
	})

	factoryCls := &object.Class{Name: "_Sentinel", Dict: object.NewDict()}
	factoryCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{
		Name: "__getattr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			name := object.Str_(a[1])
			mu.Lock()
			defer mu.Unlock()
			if obj, ok := registry[name]; ok {
				return obj, nil
			}
			obj := &object.Instance{Class: sentCls, Dict: object.NewDict()}
			obj.Dict.SetStr("_name", &object.Str{V: name})
			registry[name] = obj
			return obj, nil
		},
	})
	return &object.Instance{Class: factoryCls, Dict: object.NewDict()}
}

// ─────────────────────────────────────────────────────────────────────────────
// Mock / MagicMock / NonCallableMock
// ─────────────────────────────────────────────────────────────────────────────

// mockCallEq compares two call records (args tuple + kwargs dict)
func mockCallEq(a, b *object.Instance) bool {
	sa, _ := a.Dict.GetStr("_args")
	oa, _ := b.Dict.GetStr("_args")
	sk, _ := a.Dict.GetStr("_kwargs")
	ok, _ := b.Dict.GetStr("_kwargs")
	argsEq, _ := object.Eq(sa, oa)
	kwargsEq, _ := object.Eq(sk, ok)
	return argsEq && kwargsEq
}

// isExceptionClass returns true if cls is an exception class
func isExceptionClass(cls *object.Class) bool {
	if cls == nil {
		return false
	}
	for _, b := range cls.Bases {
		if b != nil && (b.Name == "Exception" || b.Name == "BaseException" || isExceptionClass(b)) {
			return true
		}
	}
	return false
}

// mockSpecHasAttr checks if an object's spec has a given attribute
func mockSpecHasAttr(spec object.Object, name string) bool {
	switch s := spec.(type) {
	case *object.Class:
		_, ok := s.Dict.GetStr(name)
		return ok
	case *object.Instance:
		if s.Class != nil {
			_, ok := classLookup(s.Class, name)
			return ok
		}
	case *object.Module:
		_, ok := s.Dict.GetStr(name)
		return ok
	}
	return false
}

// mockInitInstance initialises the standard state fields on a new Mock instance.
func mockInitInstance(inst *object.Instance, callable bool) {
	inst.Dict.SetStr("called", object.BoolOf(false))
	inst.Dict.SetStr("call_count", object.NewInt(0))
	inst.Dict.SetStr("call_args", object.None)
	inst.Dict.SetStr("call_args_list", &object.List{})
	inst.Dict.SetStr("method_calls", &object.List{})
	inst.Dict.SetStr("mock_calls", &object.List{})
	inst.Dict.SetStr("_children", object.NewDict())
	inst.Dict.SetStr("_spec", object.None)
	inst.Dict.SetStr("_callable", object.BoolOf(callable))
	inst.Dict.SetStr("return_value", object.None)
	inst.Dict.SetStr("_return_value_set", object.BoolOf(false))
	inst.Dict.SetStr("side_effect", object.None)
	inst.Dict.SetStr("_name", object.None)
	inst.Dict.SetStr("_side_effect_idx", object.NewInt(0))
}

// buildMockCore builds the core Mock/NonCallableMock class.
func buildMockCore(i *Interp, callCls *object.Class, callable bool, className string) *object.Class {
	cls := &object.Class{Name: className, Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			mockInitInstance(inst, callable)
			if kw != nil {
				if rv, ok := kw.GetStr("return_value"); ok {
					inst.Dict.SetStr("return_value", rv)
					inst.Dict.SetStr("_return_value_set", object.BoolOf(true))
				}
				if se, ok := kw.GetStr("side_effect"); ok {
					inst.Dict.SetStr("side_effect", se)
				}
				if nm, ok := kw.GetStr("name"); ok {
					inst.Dict.SetStr("_name", nm)
				}
				if sp, ok := kw.GetStr("spec"); ok {
					inst.Dict.SetStr("_spec", sp)
				}
			}
			return object.None, nil
		},
	})

	cls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "<Mock>"}, nil
			}
			inst := a[0].(*object.Instance)
			nameObj, _ := inst.Dict.GetStr("_name")
			name := object.Str_(nameObj)
			if name == "" || name == "None" {
				return &object.Str{V: fmt.Sprintf("<%s id='%p'>", className, inst)}, nil
			}
			return &object.Str{V: fmt.Sprintf("<%s name='%s' id='%p'>", className, name, inst)}, nil
		},
	})

	// __getattr__ — returns child mocks, checking spec
	cls.Dict.SetStr("__getattr__", &object.BuiltinFunc{
		Name: "__getattr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			name := object.Str_(a[1])

			// Check spec
			specObj, _ := inst.Dict.GetStr("_spec")
			if specObj != nil && specObj != object.None {
				if !mockSpecHasAttr(specObj, name) {
					return nil, object.Errorf(i.attrErr, "'Mock' object has no attribute '%s'", name)
				}
			}

			// Return or create child mock
			childrenObj, _ := inst.Dict.GetStr("_children")
			cd, _ := childrenObj.(*object.Dict)
			if cd == nil {
				cd = object.NewDict()
				inst.Dict.SetStr("_children", cd)
			}
			if child, ok2 := cd.GetStr(name); ok2 {
				return child, nil
			}
			child := &object.Instance{Class: cls, Dict: object.NewDict()}
			mockInitInstance(child, callable)
			child.Dict.SetStr("_name", &object.Str{V: name})
			cd.SetStr(name, child)
			return child, nil
		},
	})

	// __setattr__
	cls.Dict.SetStr("__setattr__", &object.BuiltinFunc{
		Name: "__setattr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			name := object.Str_(a[1])
			val := a[2]
			if name == "return_value" {
				inst.Dict.SetStr("_return_value_set", object.BoolOf(true))
			}
			// If the name corresponds to a child mock, update children dict too
			childrenObj, _ := inst.Dict.GetStr("_children")
			if cd, ok := childrenObj.(*object.Dict); ok {
				if _, exists := cd.GetStr(name); exists {
					cd.SetStr(name, val)
					return object.None, nil
				}
			}
			inst.Dict.SetStr(name, val)
			return object.None, nil
		},
	})

	// __call__
	if callable {
		cls.Dict.SetStr("__call__", &object.BuiltinFunc{
			Name: "__call__",
			Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) < 1 {
					return object.None, nil
				}
				inst := a[0].(*object.Instance)
				args := a[1:]

				// Record call
				inst.Dict.SetStr("called", object.BoolOf(true))
				cntObj, _ := inst.Dict.GetStr("call_count")
				n := int64(0)
				if ni, ok := cntObj.(*object.Int); ok {
					n = ni.Int64()
				}
				inst.Dict.SetStr("call_count", object.NewInt(n+1))

				callInst := &object.Instance{Class: callCls, Dict: object.NewDict()}
				callInst.Dict.SetStr("_args", &object.Tuple{V: args})
				if kw == nil {
					kw = object.NewDict()
				}
				callInst.Dict.SetStr("_kwargs", kw)

				inst.Dict.SetStr("call_args", callInst)
				if v, ok2 := inst.Dict.GetStr("call_args_list"); ok2 {
					if lst, ok3 := v.(*object.List); ok3 {
						lst.V = append(lst.V, callInst)
					}
				}
				if v, ok2 := inst.Dict.GetStr("mock_calls"); ok2 {
					if lst, ok3 := v.(*object.List); ok3 {
						lst.V = append(lst.V, callInst)
					}
				}

				// side_effect takes priority
				seObj, _ := inst.Dict.GetStr("side_effect")
				if seObj != nil && seObj != object.None {
					switch se := seObj.(type) {
					case *object.List:
						idxObj, _ := inst.Dict.GetStr("_side_effect_idx")
						idx := int64(0)
						if idxI, ok2 := idxObj.(*object.Int); ok2 {
							idx = idxI.Int64()
						}
						if int(idx) >= len(se.V) {
							return nil, object.Errorf(i.stopIter, "")
						}
						val := se.V[idx]
						inst.Dict.SetStr("_side_effect_idx", object.NewInt(idx+1))
						if exc, ok2 := val.(*object.Exception); ok2 {
							return nil, exc
						}
						if excCls, ok2 := val.(*object.Class); ok2 {
							if isExceptionClass(excCls) {
								return nil, object.Errorf(excCls, "")
							}
						}
						if instV, ok2 := val.(*object.Instance); ok2 {
							if isExceptionClass(instV.Class) {
								return nil, object.Errorf(instV.Class, "%s", object.Str_(instV))
							}
						}
						return val, nil
					case *object.Tuple:
						idxObj, _ := inst.Dict.GetStr("_side_effect_idx")
						idx := int64(0)
						if idxI, ok2 := idxObj.(*object.Int); ok2 {
							idx = idxI.Int64()
						}
						if int(idx) >= len(se.V) {
							return nil, object.Errorf(i.stopIter, "")
						}
						val := se.V[idx]
						inst.Dict.SetStr("_side_effect_idx", object.NewInt(idx+1))
						return val, nil
					case *object.BuiltinFunc:
						return se.Call(ii, args, kw)
					case *object.Function:
						interp := ii.(*Interp)
						return interp.callFunction(se, args, kw)
					case *object.Instance:
						interp := ii.(*Interp)
						return interp.callObject(se, args, kw)
					default:
						if exc, ok2 := seObj.(*object.Exception); ok2 {
							return nil, exc
						}
						if excCls, ok2 := seObj.(*object.Class); ok2 {
							if isExceptionClass(excCls) {
								return nil, object.Errorf(excCls, "")
							}
						}
					}
				}

				rvObj, _ := inst.Dict.GetStr("return_value")
				if rvObj != nil {
					return rvObj, nil
				}
				return object.None, nil
			},
		})
	} else {
		cls.Dict.SetStr("__call__", &object.BuiltinFunc{
			Name: "__call__",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				return nil, object.Errorf(i.typeErr, "Mock object is not callable")
			},
		})
	}

	// reset_mock
	cls.Dict.SetStr("reset_mock", &object.BuiltinFunc{
		Name: "reset_mock",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("called", object.BoolOf(false))
			inst.Dict.SetStr("call_count", object.NewInt(0))
			inst.Dict.SetStr("call_args", object.None)
			inst.Dict.SetStr("call_args_list", &object.List{})
			inst.Dict.SetStr("method_calls", &object.List{})
			inst.Dict.SetStr("mock_calls", &object.List{})
			inst.Dict.SetStr("_side_effect_idx", object.NewInt(0))
			return object.None, nil
		},
	})

	// configure_mock
	cls.Dict.SetStr("configure_mock", &object.BuiltinFunc{
		Name: "configure_mock",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 || kw == nil {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			keys, vals := kw.Items()
			for idx, k := range keys {
				ks := object.Str_(k)
				v := vals[idx]
				if ks == "return_value" {
					inst.Dict.SetStr("return_value", v)
					inst.Dict.SetStr("_return_value_set", object.BoolOf(true))
				} else {
					inst.Dict.SetStr(ks, v)
				}
			}
			return object.None, nil
		},
	})

	// assert_called
	cls.Dict.SetStr("assert_called", &object.BuiltinFunc{
		Name: "assert_called",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			calledObj, _ := inst.Dict.GetStr("called")
			if !object.Truthy(calledObj) {
				return nil, object.Errorf(i.assertErr, "Expected mock to have been called")
			}
			return object.None, nil
		},
	})

	// assert_not_called
	cls.Dict.SetStr("assert_not_called", &object.BuiltinFunc{
		Name: "assert_not_called",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			calledObj, _ := inst.Dict.GetStr("called")
			if object.Truthy(calledObj) {
				cntObj, _ := inst.Dict.GetStr("call_count")
				return nil, object.Errorf(i.assertErr, "Expected mock not to have been called. Called %s times.", object.Str_(cntObj))
			}
			return object.None, nil
		},
	})

	// assert_called_once
	cls.Dict.SetStr("assert_called_once", &object.BuiltinFunc{
		Name: "assert_called_once",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			cntObj, _ := inst.Dict.GetStr("call_count")
			cnt := int64(0)
			if ci, ok := cntObj.(*object.Int); ok {
				cnt = ci.Int64()
			}
			if cnt != 1 {
				return nil, object.Errorf(i.assertErr, "Expected mock to have been called once. Called %d times.", cnt)
			}
			return object.None, nil
		},
	})

	// assert_called_with
	cls.Dict.SetStr("assert_called_with", &object.BuiltinFunc{
		Name: "assert_called_with",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			expectedArgs := a[1:]
			callArgsObj, _ := inst.Dict.GetStr("call_args")
			if callArgsObj == nil || callArgsObj == object.None {
				return nil, object.Errorf(i.assertErr, "Expected call not found.")
			}
			callArg, ok := callArgsObj.(*object.Instance)
			if !ok {
				return nil, object.Errorf(i.assertErr, "Expected call not found.")
			}
			expectedInst := &object.Instance{Class: callCls, Dict: object.NewDict()}
			expectedInst.Dict.SetStr("_args", &object.Tuple{V: expectedArgs})
			if kw == nil {
				kw = object.NewDict()
			}
			expectedInst.Dict.SetStr("_kwargs", kw)
			if !mockCallEq(callArg, expectedInst) {
				return nil, object.Errorf(i.assertErr, "Expected call not found.")
			}
			return object.None, nil
		},
	})

	// assert_called_once_with
	cls.Dict.SetStr("assert_called_once_with", &object.BuiltinFunc{
		Name: "assert_called_once_with",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			cntObj, _ := inst.Dict.GetStr("call_count")
			cnt := int64(0)
			if ci, ok := cntObj.(*object.Int); ok {
				cnt = ci.Int64()
			}
			if cnt != 1 {
				return nil, object.Errorf(i.assertErr, "Expected to be called once. Called %d times.", cnt)
			}
			expectedArgs := a[1:]
			callArgsObj, _ := inst.Dict.GetStr("call_args")
			if callArgsObj == nil || callArgsObj == object.None {
				return nil, object.Errorf(i.assertErr, "Expected call not found.")
			}
			callArg, ok := callArgsObj.(*object.Instance)
			if !ok {
				return nil, object.Errorf(i.assertErr, "Expected call not found.")
			}
			expectedInst := &object.Instance{Class: callCls, Dict: object.NewDict()}
			expectedInst.Dict.SetStr("_args", &object.Tuple{V: expectedArgs})
			if kw == nil {
				kw = object.NewDict()
			}
			expectedInst.Dict.SetStr("_kwargs", kw)
			if !mockCallEq(callArg, expectedInst) {
				return nil, object.Errorf(i.assertErr, "Expected call not found.")
			}
			return object.None, nil
		},
	})

	// assert_any_call
	cls.Dict.SetStr("assert_any_call", &object.BuiltinFunc{
		Name: "assert_any_call",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			expectedArgs := a[1:]
			expectedInst := &object.Instance{Class: callCls, Dict: object.NewDict()}
			expectedInst.Dict.SetStr("_args", &object.Tuple{V: expectedArgs})
			if kw == nil {
				kw = object.NewDict()
			}
			expectedInst.Dict.SetStr("_kwargs", kw)

			callListObj, _ := inst.Dict.GetStr("call_args_list")
			if lst, ok := callListObj.(*object.List); ok {
				for _, c := range lst.V {
					if ci, ok2 := c.(*object.Instance); ok2 {
						if mockCallEq(ci, expectedInst) {
							return object.None, nil
						}
					}
				}
			}
			return nil, object.Errorf(i.assertErr, "mock was not called with the expected arguments")
		},
	})

	// assert_has_calls — calls appear as subsequence
	cls.Dict.SetStr("assert_has_calls", &object.BuiltinFunc{
		Name: "assert_has_calls",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			expectedListObj := a[1]

			var expected []*object.Instance
			switch el := expectedListObj.(type) {
			case *object.List:
				for _, v := range el.V {
					if ci, ok := v.(*object.Instance); ok {
						expected = append(expected, ci)
					}
				}
			case *object.Tuple:
				for _, v := range el.V {
					if ci, ok := v.(*object.Instance); ok {
						expected = append(expected, ci)
					}
				}
			}

			callListObj, _ := inst.Dict.GetStr("call_args_list")
			var actual []*object.Instance
			if lst, ok := callListObj.(*object.List); ok {
				for _, v := range lst.V {
					if ci, ok2 := v.(*object.Instance); ok2 {
						actual = append(actual, ci)
					}
				}
			}

			// Find expected as subsequence of actual
			j := 0
			for _, exp := range expected {
				found := false
				for j < len(actual) {
					if mockCallEq(actual[j], exp) {
						j++
						found = true
						break
					}
					j++
				}
				if !found {
					return nil, object.Errorf(i.assertErr, "Calls not found in mock_calls")
				}
			}
			return object.None, nil
		},
	})

	return cls
}

// buildMagicMockClassFromBase adds magic methods on top of Mock.
func buildMagicMockClassFromBase(i *Interp, baseCls *object.Class) *object.Class {
	cls := &object.Class{Name: "MagicMock", Bases: []*object.Class{baseCls}, Dict: object.NewDict()}

	cls.Dict.SetStr("__str__", &object.BuiltinFunc{
		Name: "__str__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			inst := a[0].(*object.Instance)
			return &object.Str{V: fmt.Sprintf("<MagicMock id='%p'>", inst)}, nil
		},
	})

	cls.Dict.SetStr("__bool__", &object.BuiltinFunc{
		Name: "__bool__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(true), nil
		},
	})

	return cls
}

// ─────────────────────────────────────────────────────────────────────────────
// patch infrastructure
// ─────────────────────────────────────────────────────────────────────────────

// mockMakeInstance creates a new Mock instance for use inside a patch.
func mockMakeInstance(mockCls *object.Class, patchInst *object.Instance) *object.Instance {
	m := &object.Instance{Class: mockCls, Dict: object.NewDict()}
	mockInitInstance(m, true)
	rvObj, hasRV := patchInst.Dict.GetStr("_return_value")
	if hasRV && rvObj != nil && rvObj != object.None {
		m.Dict.SetStr("return_value", rvObj)
		m.Dict.SetStr("_return_value_set", object.BoolOf(true))
	}
	return m
}

// mockPatchNameEnter: patch("a.b.c") → import "a.b", replace "c"
func mockPatchNameEnter(interp *Interp, inst *object.Instance, mockCls *object.Class) (object.Object, error) {
	targetObj, _ := inst.Dict.GetStr("_target")
	target := object.Str_(targetObj)

	dot := strings.LastIndex(target, ".")
	if dot < 0 {
		return nil, object.Errorf(interp.attrErr, "patch: target must be 'pkg.attr', got %q", target)
	}
	modName := target[:dot]
	attrName := target[dot+1:]

	mod, err := interp.loadModule(modName)
	if err != nil {
		return nil, err
	}
	orig, _ := mod.Dict.GetStr(attrName)
	inst.Dict.SetStr("_original", orig)
	inst.Dict.SetStr("_target_mod", mod)
	inst.Dict.SetStr("_attr_name", &object.Str{V: attrName})

	newObj, hasNew := inst.Dict.GetStr("_new")
	if !hasNew || newObj == nil || newObj == object.None {
		newObj = mockMakeInstance(mockCls, inst)
		inst.Dict.SetStr("_mock", newObj)
	}
	mod.Dict.SetStr(attrName, newObj)
	return newObj, nil
}

func mockPatchNameExit(inst *object.Instance) {
	modObj, _ := inst.Dict.GetStr("_target_mod")
	attrNameObj, _ := inst.Dict.GetStr("_attr_name")
	origObj, _ := inst.Dict.GetStr("_original")
	if mod, ok := modObj.(*object.Module); ok {
		attrName := object.Str_(attrNameObj)
		if origObj != nil {
			mod.Dict.SetStr(attrName, origObj)
		}
	}
}

// mockPatchObjectEnter: patch.object(obj, attr)
func mockPatchObjectEnter(interp *Interp, inst *object.Instance, mockCls *object.Class) (object.Object, error) {
	targetObj, _ := inst.Dict.GetStr("_target_obj")
	attrNameObj, _ := inst.Dict.GetStr("_attr_name")
	attrName := object.Str_(attrNameObj)

	var orig object.Object
	switch t := targetObj.(type) {
	case *object.Class:
		orig, _ = t.Dict.GetStr(attrName)
	case *object.Instance:
		orig, _ = t.Dict.GetStr(attrName)
	case *object.Module:
		orig, _ = t.Dict.GetStr(attrName)
	}
	inst.Dict.SetStr("_original", orig)

	newObj, hasNew := inst.Dict.GetStr("_new")
	if !hasNew || newObj == nil || newObj == object.None {
		newObj = mockMakeInstance(mockCls, inst)
		inst.Dict.SetStr("_mock", newObj)
	}

	switch t := targetObj.(type) {
	case *object.Class:
		t.Dict.SetStr(attrName, newObj)
		object.BumpClassEpoch()
	case *object.Instance:
		t.Dict.SetStr(attrName, newObj)
	case *object.Module:
		t.Dict.SetStr(attrName, newObj)
	}
	return newObj, nil
}

func mockPatchObjectExit(inst *object.Instance) {
	targetObj, _ := inst.Dict.GetStr("_target_obj")
	attrNameObj, _ := inst.Dict.GetStr("_attr_name")
	origObj, _ := inst.Dict.GetStr("_original")
	attrName := object.Str_(attrNameObj)

	switch t := targetObj.(type) {
	case *object.Class:
		if origObj != nil {
			t.Dict.SetStr(attrName, origObj)
		} else {
			t.Dict.Delete(&object.Str{V: attrName})
		}
		object.BumpClassEpoch()
	case *object.Instance:
		if origObj != nil {
			t.Dict.SetStr(attrName, origObj)
		}
	case *object.Module:
		if origObj != nil {
			t.Dict.SetStr(attrName, origObj)
		}
	}
}

// mockPatchDictEnter: patch.dict(d, values, clear=False)
func mockPatchDictEnter(interp *Interp, inst *object.Instance) (object.Object, error) {
	targetObj, _ := inst.Dict.GetStr("_target_dict")
	d, ok := targetObj.(*object.Dict)
	if !ok {
		return nil, object.Errorf(interp.typeErr, "patch.dict: target must be a dict")
	}

	// Snapshot current state
	snapshot := object.NewDict()
	keys, vals := d.Items()
	for idx, k := range keys {
		snapshot.Set(k, vals[idx])
	}
	inst.Dict.SetStr("_dict_snapshot", snapshot)

	clearObj, _ := inst.Dict.GetStr("_clear")
	if object.Truthy(clearObj) {
		d.Clear()
	}

	newValsObj, _ := inst.Dict.GetStr("_new_vals")
	if newDict, ok2 := newValsObj.(*object.Dict); ok2 {
		nkeys, nvals := newDict.Items()
		for idx, k := range nkeys {
			d.Set(k, nvals[idx])
		}
	}
	return targetObj, nil
}

func mockPatchDictExit(inst *object.Instance) {
	targetObj, _ := inst.Dict.GetStr("_target_dict")
	snapshotObj, _ := inst.Dict.GetStr("_dict_snapshot")
	d, ok := targetObj.(*object.Dict)
	if !ok {
		return
	}
	snapshot, ok := snapshotObj.(*object.Dict)
	if !ok {
		return
	}
	d.Clear()
	skeys, svals := snapshot.Items()
	for idx, k := range skeys {
		d.Set(k, svals[idx])
	}
}

// buildPatcherClass builds the _MockPatcher context-manager/decorator class.
func buildPatcherClass(i *Interp, mockCls *object.Class) *object.Class {
	cls := &object.Class{Name: "_MockPatcher", Dict: object.NewDict()}

	doEnter := func(interp *Interp, inst *object.Instance) (object.Object, error) {
		patchType, _ := inst.Dict.GetStr("_patch_type")
		switch object.Str_(patchType) {
		case "dict":
			return mockPatchDictEnter(interp, inst)
		case "object":
			return mockPatchObjectEnter(interp, inst, mockCls)
		default:
			return mockPatchNameEnter(interp, inst, mockCls)
		}
	}

	doExit := func(inst *object.Instance) {
		patchType, _ := inst.Dict.GetStr("_patch_type")
		switch object.Str_(patchType) {
		case "dict":
			mockPatchDictExit(inst)
		case "object":
			mockPatchObjectExit(inst)
		default:
			mockPatchNameExit(inst)
		}
	}

	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{
		Name: "__enter__",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			return doEnter(ii.(*Interp), a[0].(*object.Instance))
		},
	})

	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{
		Name: "__exit__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				if inst, ok := a[0].(*object.Instance); ok {
					doExit(inst)
				}
			}
			return object.BoolOf(false), nil
		},
	})

	// __call__ — use as decorator: @patch(...) wraps the function
	cls.Dict.SetStr("__call__", &object.BuiltinFunc{
		Name: "__call__",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			patcherInst := a[0].(*object.Instance)
			fn := a[1]

			wrapper := &object.BuiltinFunc{
				Name: "patched",
				Call: func(ii2 any, wArgs []object.Object, wKw *object.Dict) (object.Object, error) {
					interp2 := ii2.(*Interp)
					mockObj, err := doEnter(interp2, patcherInst)
					if err != nil {
						return nil, err
					}
					newArgs := append(wArgs, mockObj)
					var result object.Object
					var callErr error
					switch f := fn.(type) {
					case *object.BuiltinFunc:
						result, callErr = f.Call(ii2, newArgs, wKw)
					case *object.Function:
						result, callErr = interp2.callFunction(f, newArgs, wKw)
					default:
						result, callErr = interp2.callObject(fn, newArgs, wKw)
					}
					doExit(patcherInst)
					return result, callErr
				},
			}
			return wrapper, nil
		},
	})

	return cls
}

// ─────────────────────────────────────────────────────────────────────────────
// buildUnittestMock — main entry point
// ─────────────────────────────────────────────────────────────────────────────

func (i *Interp) buildUnittestMock() *object.Module {
	m := &object.Module{Name: "unittest.mock", Dict: object.NewDict()}

	callCls := buildCallClass(i)
	m.Dict.SetStr("call", callCls)

	anyObj := buildAnyObject(i)
	m.Dict.SetStr("ANY", anyObj)

	defaultObj := buildDefaultObject()
	m.Dict.SetStr("DEFAULT", defaultObj)

	sentObj := buildSentinel()
	m.Dict.SetStr("sentinel", sentObj)

	nonCallCls := buildMockCore(i, callCls, false, "NonCallableMock")
	m.Dict.SetStr("NonCallableMock", nonCallCls)

	mockCls := buildMockCore(i, callCls, true, "Mock")
	m.Dict.SetStr("Mock", mockCls)

	magicMockCls := buildMagicMockClassFromBase(i, mockCls)
	m.Dict.SetStr("MagicMock", magicMockCls)

	patcherCls := buildPatcherClass(i, mockCls)

	// patch function — returns a _MockPatcher
	patchFn := &object.BuiltinFunc{
		Name: "patch",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			args := mpArgs(a)
			if len(args) < 1 {
				return nil, object.Errorf(i.typeErr, "patch() requires at least 1 argument")
			}
			target := object.Str_(args[0])
			var newObj object.Object = object.None
			if len(args) > 1 {
				newObj = args[1]
			}
			var rvObj object.Object
			if kw != nil {
				if rv, ok := kw.GetStr("return_value"); ok {
					rvObj = rv
				}
				if nv, ok := kw.GetStr("new"); ok {
					newObj = nv
				}
			}
			inst := &object.Instance{Class: patcherCls, Dict: object.NewDict()}
			inst.Dict.SetStr("_patch_type", &object.Str{V: "name"})
			inst.Dict.SetStr("_target", &object.Str{V: target})
			inst.Dict.SetStr("_new", newObj)
			if rvObj != nil {
				inst.Dict.SetStr("_return_value", rvObj)
			} else {
				inst.Dict.SetStr("_return_value", object.None)
			}
			return inst, nil
		},
	}

	// patch.object
	patchObjectFn := &object.BuiltinFunc{
		Name: "object",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			args := mpArgs(a)
			if len(args) < 2 {
				return nil, object.Errorf(i.typeErr, "patch.object() requires target and attribute")
			}
			targetObj := args[0]
			attrName := object.Str_(args[1])
			var newObj object.Object = object.None
			if len(args) > 2 {
				newObj = args[2]
			}
			var rvObj object.Object
			if kw != nil {
				if rv, ok := kw.GetStr("return_value"); ok {
					rvObj = rv
				}
				if nv, ok := kw.GetStr("new"); ok {
					newObj = nv
				}
			}
			inst := &object.Instance{Class: patcherCls, Dict: object.NewDict()}
			inst.Dict.SetStr("_patch_type", &object.Str{V: "object"})
			inst.Dict.SetStr("_target_obj", targetObj)
			inst.Dict.SetStr("_attr_name", &object.Str{V: attrName})
			inst.Dict.SetStr("_new", newObj)
			if rvObj != nil {
				inst.Dict.SetStr("_return_value", rvObj)
			} else {
				inst.Dict.SetStr("_return_value", object.None)
			}
			return inst, nil
		},
	}

	// patch.dict
	patchDictFn := &object.BuiltinFunc{
		Name: "dict",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			args := mpArgs(a)
			if len(args) < 1 {
				return nil, object.Errorf(i.typeErr, "patch.dict() requires target dict")
			}
			targetObj := args[0]
			var newVals *object.Dict
			if len(args) > 1 {
				if d2, ok := args[1].(*object.Dict); ok {
					newVals = d2
				}
			}
			// remaining kwargs are new values if no positional dict
			if newVals == nil && kw != nil {
				clearVal := false
				if cv, ok := kw.GetStr("clear"); ok {
					clearVal = object.Truthy(cv)
					kw.Delete(&object.Str{V: "clear"})
					inst := &object.Instance{Class: patcherCls, Dict: object.NewDict()}
					inst.Dict.SetStr("_patch_type", &object.Str{V: "dict"})
					inst.Dict.SetStr("_target_dict", targetObj)
					inst.Dict.SetStr("_new_vals", kw)
					inst.Dict.SetStr("_clear", object.BoolOf(clearVal))
					return inst, nil
				}
				newVals = kw
				kw = nil
			}
			if newVals == nil {
				newVals = object.NewDict()
			}
			clearVal := false
			if kw != nil {
				if cv, ok := kw.GetStr("clear"); ok {
					clearVal = object.Truthy(cv)
				}
			}
			inst := &object.Instance{Class: patcherCls, Dict: object.NewDict()}
			inst.Dict.SetStr("_patch_type", &object.Str{V: "dict"})
			inst.Dict.SetStr("_target_dict", targetObj)
			inst.Dict.SetStr("_new_vals", newVals)
			inst.Dict.SetStr("_clear", object.BoolOf(clearVal))
			return inst, nil
		},
	}

	// Expose patch as a callable class instance with .object and .dict attrs
	patchWrapCls := &object.Class{Name: "_PatchCallable", Dict: object.NewDict()}
	patchWrapCls.Dict.SetStr("__call__", patchFn)
	patchWrapCls.Dict.SetStr("object", patchObjectFn)
	patchWrapCls.Dict.SetStr("dict", patchDictFn)
	patchObj := &object.Instance{Class: patchWrapCls, Dict: object.NewDict()}
	m.Dict.SetStr("patch", patchObj)

	// create_autospec
	m.Dict.SetStr("create_autospec", &object.BuiltinFunc{
		Name: "create_autospec",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			args := mpArgs(a)
			if len(args) < 1 {
				return nil, object.Errorf(i.typeErr, "create_autospec() requires spec argument")
			}
			spec := args[0]
			mi := &object.Instance{Class: mockCls, Dict: object.NewDict()}
			mockInitInstance(mi, true)
			mi.Dict.SetStr("_spec", spec)
			if kw != nil {
				if rv, ok := kw.GetStr("return_value"); ok {
					mi.Dict.SetStr("return_value", rv)
					mi.Dict.SetStr("_return_value_set", object.BoolOf(true))
				}
			}
			return mi, nil
		},
	})

	m.Dict.SetStr("PropertyMock", buildMockCore(i, callCls, true, "PropertyMock"))

	return m
}

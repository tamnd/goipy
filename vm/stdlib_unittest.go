package vm

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildUnittest() *object.Module {
	m := &object.Module{Name: "unittest", Dict: object.NewDict()}

	// ── SkipTest exception ────────────────────────────────────────────────────
	skipTestCls := &object.Class{
		Name:  "SkipTest",
		Bases: []*object.Class{i.exception},
		Dict:  object.NewDict(),
	}
	m.Dict.SetStr("SkipTest", skipTestCls)

	// ── helpers ───────────────────────────────────────────────────────────────

	// assertFail raises AssertionError with an optional message prefix.
	assertFail := func(defaultMsg string, userMsg object.Object) error {
		msg := defaultMsg
		if userMsg != nil && userMsg != object.None {
			if s, ok := userMsg.(*object.Str); ok {
				msg = s.V + " : " + defaultMsg
			}
		}
		return object.Errorf(i.assertErr, "%s", msg)
	}

	// pyEq wraps object.Eq to return a plain bool (errors → false).
	pyEq := func(a, b object.Object) bool {
		eq, _ := object.Eq(a, b)
		return eq
	}

	// userMsg extracts optional message from args/kwargs.
	userMsgArg := func(a []object.Object, pos int, kw *object.Dict) object.Object {
		if kw != nil {
			if v, ok := kw.GetStr("msg"); ok {
				return v
			}
		}
		if len(a) > pos {
			return a[pos]
		}
		return nil
	}

	// ── TestResult class ──────────────────────────────────────────────────────
	testResultCls := &object.Class{Name: "TestResult", Dict: object.NewDict()}
	makeTestResult := func() *object.Instance {
		r := &object.Instance{Class: testResultCls, Dict: object.NewDict()}
		r.Dict.SetStr("errors", &object.List{})
		r.Dict.SetStr("failures", &object.List{})
		r.Dict.SetStr("skipped", &object.List{})
		r.Dict.SetStr("expectedFailures", &object.List{})
		r.Dict.SetStr("unexpectedSuccesses", &object.List{})
		r.Dict.SetStr("collectedDurations", &object.List{})
		r.Dict.SetStr("testsRun", object.NewInt(0))
		r.Dict.SetStr("shouldStop", object.BoolOf(false))
		r.Dict.SetStr("buffer", object.BoolOf(false))
		r.Dict.SetStr("failfast", object.BoolOf(false))
		r.Dict.SetStr("tb_locals", object.BoolOf(false))
		return r
	}
	testResultCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			blank := makeTestResult()
			// copy all blank fields
			keys, vals := blank.Dict.Items()
			for k, key := range keys {
				if ks, ok := key.(*object.Str); ok {
					inst.Dict.SetStr(ks.V, vals[k])
				}
			}
			return object.None, nil
		},
	})
	testResultCls.Dict.SetStr("wasSuccessful", &object.BuiltinFunc{
		Name: "wasSuccessful",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			errList, _ := self.Dict.GetStr("errors")
			failList, _ := self.Dict.GetStr("failures")
			noErrors := true
			if l, ok := errList.(*object.List); ok && len(l.V) > 0 {
				noErrors = false
			}
			if l, ok := failList.(*object.List); ok && len(l.V) > 0 {
				noErrors = false
			}
			return object.BoolOf(noErrors), nil
		},
	})
	testResultCls.Dict.SetStr("stop", &object.BuiltinFunc{
		Name: "stop",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 1 {
				a[0].(*object.Instance).Dict.SetStr("shouldStop", object.BoolOf(true))
			}
			return object.None, nil
		},
	})
	testResultCls.Dict.SetStr("startTest", &object.BuiltinFunc{
		Name: "startTest",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			prev := int64(0)
			if v, ok := self.Dict.GetStr("testsRun"); ok {
				if n, ok2 := toInt64(v); ok2 {
					prev = n
				}
			}
			self.Dict.SetStr("testsRun", object.NewInt(prev+1))
			return object.None, nil
		},
	})
	testResultCls.Dict.SetStr("stopTest", &object.BuiltinFunc{
		Name: "stopTest",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	testResultCls.Dict.SetStr("startTestRun", &object.BuiltinFunc{
		Name: "startTestRun",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	testResultCls.Dict.SetStr("stopTestRun", &object.BuiltinFunc{
		Name: "stopTestRun",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	testResultCls.Dict.SetStr("addSuccess", &object.BuiltinFunc{
		Name: "addSuccess",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	appendToResultList := func(listName string, item object.Object) func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			if v, ok := self.Dict.GetStr(listName); ok {
				if l, ok2 := v.(*object.List); ok2 {
					// build a tuple (test, info)
					info := object.Object(object.None)
					if item == nil && len(a) >= 3 {
						info = a[2]
					} else if item == nil && len(a) >= 2 {
						info = a[1]
					}
					pair := &object.Tuple{V: []object.Object{a[1], info}}
					l.V = append(l.V, pair)
				}
			}
			return object.None, nil
		}
	}
	testResultCls.Dict.SetStr("addError", &object.BuiltinFunc{
		Name: "addError",
		Call: appendToResultList("errors", nil),
	})
	testResultCls.Dict.SetStr("addFailure", &object.BuiltinFunc{
		Name: "addFailure",
		Call: appendToResultList("failures", nil),
	})
	testResultCls.Dict.SetStr("addSkip", &object.BuiltinFunc{
		Name: "addSkip",
		Call: appendToResultList("skipped", nil),
	})
	testResultCls.Dict.SetStr("addExpectedFailure", &object.BuiltinFunc{
		Name: "addExpectedFailure",
		Call: appendToResultList("expectedFailures", nil),
	})
	testResultCls.Dict.SetStr("addUnexpectedSuccess", &object.BuiltinFunc{
		Name: "addUnexpectedSuccess",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			if v, ok := self.Dict.GetStr("unexpectedSuccesses"); ok {
				if l, ok2 := v.(*object.List); ok2 {
					if len(a) >= 2 {
						l.V = append(l.V, a[1])
					}
				}
			}
			return object.None, nil
		},
	})
	testResultCls.Dict.SetStr("addSubTest", &object.BuiltinFunc{
		Name: "addSubTest",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	testResultCls.Dict.SetStr("addDuration", &object.BuiltinFunc{
		Name: "addDuration",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	m.Dict.SetStr("TestResult", testResultCls)

	// callResultMethod calls a named method on a TestResult instance.
	callResultMethod := func(result *object.Instance, method string, args ...object.Object) {
		if fn, ok := classLookup(result.Class, method); ok {
			callArgs := make([]object.Object, 0, len(args)+1)
			callArgs = append(callArgs, result)
			callArgs = append(callArgs, args...)
			i.callObject(fn, callArgs, nil) //nolint:errcheck
		}
	}

	// ── assertRaises context-manager helper ───────────────────────────────────
	assertRaisesCtxCls := &object.Class{Name: "_AssertRaisesContext", Dict: object.NewDict()}
	assertRaisesCtxCls.Dict.SetStr("__enter__", &object.BuiltinFunc{
		Name: "__enter__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return a[0], nil
		},
	})
	assertRaisesCtxCls.Dict.SetStr("__exit__", &object.BuiltinFunc{
		Name: "__exit__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			excCls := a[1] // None if no exception
			var expectedCls *object.Class
			if v, ok := self.Dict.GetStr("_expected"); ok {
				expectedCls, _ = v.(*object.Class)
			}
			expectedName := "Exception"
			if expectedCls != nil {
				expectedName = expectedCls.Name
			}
			// No exception was raised?
			if _, isNone := excCls.(*object.NoneType); isNone || excCls == nil {
				return nil, object.Errorf(i.assertErr, "%s not raised", expectedName)
			}
			// Exception raised — check type.
			if raisedCls, ok := excCls.(*object.Class); ok {
				if expectedCls != nil && object.IsSubclass(raisedCls, expectedCls) {
					return object.BoolOf(true), nil // suppress
				}
			}
			return object.BoolOf(false), nil // re-raise
		},
	})

	// ── TestCase class ────────────────────────────────────────────────────────
	testCaseCls := &object.Class{Name: "TestCase", Dict: object.NewDict()}
	testCaseCls.Dict.SetStr("failureException", i.assertErr)
	testCaseCls.Dict.SetStr("longMessage", object.BoolOf(true))
	testCaseCls.Dict.SetStr("maxDiff", object.NewInt(640))

	testCaseCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			methodName := "runTest"
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					methodName = s.V
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("methodName"); ok {
					if s, ok2 := v.(*object.Str); ok2 {
						methodName = s.V
					}
				}
			}
			inst.Dict.SetStr("_testMethodName", &object.Str{V: methodName})
			inst.Dict.SetStr("_cleanups", &object.List{})
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("setUp", &object.BuiltinFunc{
		Name: "setUp",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("tearDown", &object.BuiltinFunc{
		Name: "tearDown",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("setUpClass", &object.BuiltinFunc{
		Name: "setUpClass",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("tearDownClass", &object.BuiltinFunc{
		Name: "tearDownClass",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("countTestCases", &object.BuiltinFunc{
		Name: "countTestCases",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(1), nil
		},
	})
	testCaseCls.Dict.SetStr("id", &object.BuiltinFunc{
		Name: "id",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			moduleName := "__main__"
			if v, ok := self.Class.Dict.GetStr("__module__"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					moduleName = s.V
				}
			}
			methodName := "runTest"
			if v, ok := self.Dict.GetStr("_testMethodName"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					methodName = s.V
				}
			}
			return &object.Str{V: moduleName + "." + self.Class.Name + "." + methodName}, nil
		},
	})
	testCaseCls.Dict.SetStr("shortDescription", &object.BuiltinFunc{
		Name: "shortDescription",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("skipTest", &object.BuiltinFunc{
		Name: "skipTest",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			reason := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					reason = s.V
				}
			}
			return nil, object.Errorf(skipTestCls, "%s", reason)
		},
	})
	testCaseCls.Dict.SetStr("fail", &object.BuiltinFunc{
		Name: "fail",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			msg := "test failed"
			if len(a) >= 2 && a[1] != object.None {
				if s, ok := a[1].(*object.Str); ok {
					msg = s.V
				}
			}
			return nil, object.Errorf(i.assertErr, "%s", msg)
		},
	})
	testCaseCls.Dict.SetStr("defaultTestResult", &object.BuiltinFunc{
		Name: "defaultTestResult",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return makeTestResult(), nil
		},
	})
	testCaseCls.Dict.SetStr("addCleanup", &object.BuiltinFunc{
		Name: "addCleanup",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			if v, ok := self.Dict.GetStr("_cleanups"); ok {
				if l, ok2 := v.(*object.List); ok2 {
					l.V = append(l.V, a[1])
				}
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("doCleanups", &object.BuiltinFunc{
		Name: "doCleanups",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.BoolOf(true), nil
			}
			self := a[0].(*object.Instance)
			if v, ok := self.Dict.GetStr("_cleanups"); ok {
				if l, ok2 := v.(*object.List); ok2 {
					for j := len(l.V) - 1; j >= 0; j-- {
						i.callObject(l.V[j], nil, nil) //nolint:errcheck
					}
					l.V = nil
				}
			}
			return object.BoolOf(true), nil
		},
	})
	testCaseCls.Dict.SetStr("subTest", &object.BuiltinFunc{
		Name: "subTest",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// Return a trivial no-op context manager.
			ctx := &object.Instance{Class: &object.Class{Name: "_SubTestContext", Dict: object.NewDict()}, Dict: object.NewDict()}
			ctx.Class.Dict.SetStr("__enter__", &object.BuiltinFunc{
				Name: "__enter__",
				Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
					return a[0], nil
				},
			})
			ctx.Class.Dict.SetStr("__exit__", &object.BuiltinFunc{
				Name: "__exit__",
				Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
					return object.BoolOf(false), nil
				},
			})
			return ctx, nil
		},
	})

	// run — execute the test method and record the result.
	testCaseCls.Dict.SetStr("run", &object.BuiltinFunc{
		Name: "run",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			var result *object.Instance
			if len(a) >= 2 {
				result, _ = a[1].(*object.Instance)
			}
			if result == nil {
				result = makeTestResult()
			}

			callResultMethod(result, "startTest", self)

			methodName := "runTest"
			if v, ok := self.Dict.GetStr("_testMethodName"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					methodName = s.V
				}
			}

			// Check for __unittest_skip__ on the method.
			if method, ok := classLookup(self.Class, methodName); ok {
				if fn, ok2 := method.(*object.Function); ok2 {
					if fn.Dict != nil {
						if v, ok3 := fn.Dict.GetStr("__unittest_skip__"); ok3 {
							if object.Truthy(v) {
								reason := ""
								if rv, ok4 := fn.Dict.GetStr("__unittest_skip_why__"); ok4 {
									if s, ok5 := rv.(*object.Str); ok5 {
										reason = s.V
									}
								}
								callResultMethod(result, "addSkip", self, &object.Str{V: reason})
								callResultMethod(result, "stopTest", self)
								return result, nil
							}
						}
					}
				}
			}

			// setUp
			if setUpFn, ok := classLookup(self.Class, "setUp"); ok {
				if _, err := i.callObject(setUpFn, []object.Object{self}, nil); err != nil {
					if exc, ok2 := err.(*object.Exception); ok2 {
						errInfo := &object.Tuple{V: []object.Object{exc.Class, exc, object.None}}
						callResultMethod(result, "addError", self, errInfo)
					}
					callResultMethod(result, "stopTest", self)
					return result, nil
				}
			}

			// Run the test method.
			if method, ok := classLookup(self.Class, methodName); ok {
				_, err := i.callObject(method, []object.Object{self}, nil)
				if err != nil {
					if exc, ok2 := err.(*object.Exception); ok2 {
						errInfo := &object.Tuple{V: []object.Object{exc.Class, exc, object.None}}
						if object.IsSubclass(exc.Class, skipTestCls) {
							reason := exc.Msg
							callResultMethod(result, "addSkip", self, &object.Str{V: reason})
						} else if object.IsSubclass(exc.Class, i.assertErr) {
							callResultMethod(result, "addFailure", self, errInfo)
						} else {
							callResultMethod(result, "addError", self, errInfo)
						}
					}
				} else {
					callResultMethod(result, "addSuccess", self)
				}
			} else {
				errInfo := &object.Tuple{V: []object.Object{i.attrErr, object.Errorf(i.attrErr, "no test method: %s", methodName), object.None}}
				callResultMethod(result, "addError", self, errInfo)
			}

			// tearDown (best effort)
			if tearDownFn, ok := classLookup(self.Class, "tearDown"); ok {
				i.callObject(tearDownFn, []object.Object{self}, nil) //nolint:errcheck
			}

			callResultMethod(result, "stopTest", self)
			return result, nil
		},
	})
	testCaseCls.Dict.SetStr("debug", &object.BuiltinFunc{
		Name: "debug",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			methodName := "runTest"
			if v, ok := self.Dict.GetStr("_testMethodName"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					methodName = s.V
				}
			}
			if method, ok := classLookup(self.Class, methodName); ok {
				_, err := i.callObject(method, []object.Object{self}, nil)
				if err != nil {
					return nil, err
				}
			}
			return object.None, nil
		},
	})

	// ── Assertion methods ──────────────────────────────────────────────────────
	testCaseCls.Dict.SetStr("assertEqual", &object.BuiltinFunc{
		Name: "assertEqual",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertEqual: missing args", nil)
			}
			if !pyEq(a[1], a[2]) {
				msg := fmt.Sprintf("%s != %s", object.Repr(a[1]), object.Repr(a[2]))
				return nil, assertFail(msg, userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertNotEqual", &object.BuiltinFunc{
		Name: "assertNotEqual",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertNotEqual: missing args", nil)
			}
			if pyEq(a[1], a[2]) {
				msg := fmt.Sprintf("%s == %s", object.Repr(a[1]), object.Repr(a[2]))
				return nil, assertFail(msg, userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertTrue", &object.BuiltinFunc{
		Name: "assertTrue",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, assertFail("assertTrue: missing expr", nil)
			}
			if !object.Truthy(a[1]) {
				return nil, assertFail(fmt.Sprintf("%s is not true", object.Repr(a[1])), userMsgArg(a, 2, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertFalse", &object.BuiltinFunc{
		Name: "assertFalse",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, assertFail("assertFalse: missing expr", nil)
			}
			if object.Truthy(a[1]) {
				return nil, assertFail(fmt.Sprintf("%s is not false", object.Repr(a[1])), userMsgArg(a, 2, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertIs", &object.BuiltinFunc{
		Name: "assertIs",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertIs: missing args", nil)
			}
			if a[1] != a[2] {
				return nil, assertFail(fmt.Sprintf("%s is not %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertIsNot", &object.BuiltinFunc{
		Name: "assertIsNot",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertIsNot: missing args", nil)
			}
			if a[1] == a[2] {
				return nil, assertFail(fmt.Sprintf("unexpectedly identical: %s", object.Repr(a[1])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertIsNone", &object.BuiltinFunc{
		Name: "assertIsNone",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, assertFail("assertIsNone: missing expr", nil)
			}
			if _, ok := a[1].(*object.NoneType); !ok {
				return nil, assertFail(fmt.Sprintf("%s is not None", object.Repr(a[1])), userMsgArg(a, 2, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertIsNotNone", &object.BuiltinFunc{
		Name: "assertIsNotNone",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, assertFail("assertIsNotNone: missing expr", nil)
			}
			if _, ok := a[1].(*object.NoneType); ok {
				return nil, assertFail("unexpectedly None", userMsgArg(a, 2, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertIn", &object.BuiltinFunc{
		Name: "assertIn",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertIn: missing args", nil)
			}
			found, err := i.pyContains(a[2], a[1])
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, assertFail(fmt.Sprintf("%s not found in %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertNotIn", &object.BuiltinFunc{
		Name: "assertNotIn",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertNotIn: missing args", nil)
			}
			found, err := i.pyContains(a[2], a[1])
			if err != nil {
				return nil, err
			}
			if found {
				return nil, assertFail(fmt.Sprintf("%s unexpectedly found in %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertIsInstance", &object.BuiltinFunc{
		Name: "assertIsInstance",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertIsInstance: missing args", nil)
			}
			if !isinstance(a[1], a[2]) {
				return nil, assertFail(fmt.Sprintf("%s is not an instance of %s", object.Repr(a[1]), object.TypeName(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertNotIsInstance", &object.BuiltinFunc{
		Name: "assertNotIsInstance",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertNotIsInstance: missing args", nil)
			}
			if isinstance(a[1], a[2]) {
				return nil, assertFail(fmt.Sprintf("%s is an instance of %s", object.Repr(a[1]), object.TypeName(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertGreater", &object.BuiltinFunc{
		Name: "assertGreater",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertGreater: missing args", nil)
			}
			r, err := i.compare(a[1], a[2], cmpGT)
			if err != nil {
				return nil, err
			}
			if !object.Truthy(r) {
				return nil, assertFail(fmt.Sprintf("%s not greater than %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertGreaterEqual", &object.BuiltinFunc{
		Name: "assertGreaterEqual",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertGreaterEqual: missing args", nil)
			}
			r, err := i.compare(a[1], a[2], cmpGE)
			if err != nil {
				return nil, err
			}
			if !object.Truthy(r) {
				return nil, assertFail(fmt.Sprintf("%s not greater than or equal to %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertLess", &object.BuiltinFunc{
		Name: "assertLess",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertLess: missing args", nil)
			}
			r, err := i.compare(a[1], a[2], cmpLT)
			if err != nil {
				return nil, err
			}
			if !object.Truthy(r) {
				return nil, assertFail(fmt.Sprintf("%s not less than %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertLessEqual", &object.BuiltinFunc{
		Name: "assertLessEqual",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertLessEqual: missing args", nil)
			}
			r, err := i.compare(a[1], a[2], cmpLE)
			if err != nil {
				return nil, err
			}
			if !object.Truthy(r) {
				return nil, assertFail(fmt.Sprintf("%s not less than or equal to %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertAlmostEqual", &object.BuiltinFunc{
		Name: "assertAlmostEqual",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertAlmostEqual: missing args", nil)
			}
			fa, okA := unitTestToFloat(a[1])
			fb, okB := unitTestToFloat(a[2])
			if !okA || !okB {
				if pyEq(a[1], a[2]) {
					return object.None, nil
				}
				return nil, assertFail(fmt.Sprintf("%s != %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			places := 7
			if kw != nil {
				if v, ok := kw.GetStr("places"); ok {
					if n, ok2 := toInt64(v); ok2 {
						places = int(n)
					}
				}
			}
			diff := math.Abs(fa - fb)
			tol := math.Pow(10, float64(-places))
			if diff > tol {
				return nil, assertFail(fmt.Sprintf("%v != %v within %d places", fa, fb, places), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertNotAlmostEqual", &object.BuiltinFunc{
		Name: "assertNotAlmostEqual",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertNotAlmostEqual: missing args", nil)
			}
			fa, okA := unitTestToFloat(a[1])
			fb, okB := unitTestToFloat(a[2])
			if !okA || !okB {
				if !pyEq(a[1], a[2]) {
					return object.None, nil
				}
				return nil, assertFail(fmt.Sprintf("%s == %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			places := 7
			if kw != nil {
				if v, ok := kw.GetStr("places"); ok {
					if n, ok2 := toInt64(v); ok2 {
						places = int(n)
					}
				}
			}
			diff := math.Abs(fa - fb)
			tol := math.Pow(10, float64(-places))
			if diff <= tol {
				return nil, assertFail(fmt.Sprintf("%v == %v within %d places", fa, fb, places), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertRegex", &object.BuiltinFunc{
		Name: "assertRegex",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertRegex: missing args", nil)
			}
			text := object.Str_(a[1])
			pat := object.Str_(a[2])
			re, err := regexp.Compile(pat)
			if err != nil {
				return nil, object.Errorf(i.valueErr, "invalid regex: %s", err)
			}
			if !re.MatchString(text) {
				return nil, assertFail(fmt.Sprintf("%q does not match %q", text, pat), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertNotRegex", &object.BuiltinFunc{
		Name: "assertNotRegex",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertNotRegex: missing args", nil)
			}
			text := object.Str_(a[1])
			pat := object.Str_(a[2])
			re, err := regexp.Compile(pat)
			if err != nil {
				return nil, object.Errorf(i.valueErr, "invalid regex: %s", err)
			}
			if re.MatchString(text) {
				return nil, assertFail(fmt.Sprintf("%q matches %q", text, pat), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertCountEqual", &object.BuiltinFunc{
		Name: "assertCountEqual",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertCountEqual: missing args", nil)
			}
			if !unitTestCountEqual(a[1], a[2]) {
				return nil, assertFail(fmt.Sprintf("sequences %s and %s are not equal when counted", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertMultiLineEqual", &object.BuiltinFunc{
		Name: "assertMultiLineEqual",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertMultiLineEqual: missing args", nil)
			}
			s1, ok1 := a[1].(*object.Str)
			s2, ok2 := a[2].(*object.Str)
			if !ok1 || !ok2 {
				return nil, assertFail("assertMultiLineEqual: both args must be str", nil)
			}
			if s1.V != s2.V {
				return nil, assertFail(fmt.Sprintf("multi-line strings differ:\n%s\n!=\n%s", s1.V, s2.V), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertSequenceEqual", &object.BuiltinFunc{
		Name: "assertSequenceEqual",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertSequenceEqual: missing args", nil)
			}
			if !pyEq(a[1], a[2]) {
				return nil, assertFail(fmt.Sprintf("%s != %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertListEqual", &object.BuiltinFunc{
		Name: "assertListEqual",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertListEqual: missing args", nil)
			}
			if !pyEq(a[1], a[2]) {
				return nil, assertFail(fmt.Sprintf("lists differ: %s != %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertTupleEqual", &object.BuiltinFunc{
		Name: "assertTupleEqual",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertTupleEqual: missing args", nil)
			}
			if !pyEq(a[1], a[2]) {
				return nil, assertFail(fmt.Sprintf("tuples differ: %s != %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertSetEqual", &object.BuiltinFunc{
		Name: "assertSetEqual",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertSetEqual: missing args", nil)
			}
			if !pyEq(a[1], a[2]) {
				return nil, assertFail(fmt.Sprintf("sets differ: %s != %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertDictEqual", &object.BuiltinFunc{
		Name: "assertDictEqual",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertDictEqual: missing args", nil)
			}
			if !pyEq(a[1], a[2]) {
				return nil, assertFail(fmt.Sprintf("dicts differ: %s != %s", object.Repr(a[1]), object.Repr(a[2])), userMsgArg(a, 3, kw))
			}
			return object.None, nil
		},
	})

	// assertRaises — context manager.
	testCaseCls.Dict.SetStr("assertRaises", &object.BuiltinFunc{
		Name: "assertRaises",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, assertFail("assertRaises: missing exception class", nil)
			}
			expectedCls, ok := a[1].(*object.Class)
			if !ok {
				return nil, assertFail("assertRaises: first arg must be an exception class", nil)
			}
			// Called as context manager (no callable): return context manager object.
			if len(a) == 2 && (kw == nil || kw.Len() == 0) {
				ctx := &object.Instance{Class: assertRaisesCtxCls, Dict: object.NewDict()}
				ctx.Dict.SetStr("_expected", expectedCls)
				return ctx, nil
			}
			// Called directly: assertRaises(exc, callable, *args)
			if len(a) >= 3 {
				args := a[3:]
				_, err := i.callObject(a[2], args, kw)
				if err == nil {
					return nil, assertFail(fmt.Sprintf("%s not raised", expectedCls.Name), nil)
				}
				if exc, ok2 := err.(*object.Exception); ok2 {
					if !object.IsSubclass(exc.Class, expectedCls) {
						return nil, err
					}
					return object.None, nil
				}
				return nil, err
			}
			return object.None, nil
		},
	})
	testCaseCls.Dict.SetStr("assertRaisesRegex", &object.BuiltinFunc{
		Name: "assertRaisesRegex",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, assertFail("assertRaisesRegex: missing args", nil)
			}
			expectedCls, _ := a[1].(*object.Class)
			ctx := &object.Instance{Class: assertRaisesCtxCls, Dict: object.NewDict()}
			ctx.Dict.SetStr("_expected", expectedCls)
			return ctx, nil
		},
	})
	testCaseCls.Dict.SetStr("assertWarns", &object.BuiltinFunc{
		Name: "assertWarns",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// stub: return a no-op context manager
			return unitTestNoOpCtx(), nil
		},
	})
	testCaseCls.Dict.SetStr("assertWarnsRegex", &object.BuiltinFunc{
		Name: "assertWarnsRegex",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return unitTestNoOpCtx(), nil
		},
	})
	testCaseCls.Dict.SetStr("assertLogs", &object.BuiltinFunc{
		Name: "assertLogs",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return unitTestNoOpCtx(), nil
		},
	})
	testCaseCls.Dict.SetStr("assertNoLogs", &object.BuiltinFunc{
		Name: "assertNoLogs",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return unitTestNoOpCtx(), nil
		},
	})
	testCaseCls.Dict.SetStr("addTypeEqualityFunc", &object.BuiltinFunc{
		Name: "addTypeEqualityFunc",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	m.Dict.SetStr("TestCase", testCaseCls)

	// ── FunctionTestCase ──────────────────────────────────────────────────────
	funcTestCaseCls := &object.Class{Name: "FunctionTestCase", Dict: object.NewDict()}
	funcTestCaseCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("_testFunc", a[1])
			inst.Dict.SetStr("_setUp", object.None)
			inst.Dict.SetStr("_tearDown", object.None)
			if len(a) >= 3 {
				inst.Dict.SetStr("_setUp", a[2])
			}
			if len(a) >= 4 {
				inst.Dict.SetStr("_tearDown", a[3])
			}
			return object.None, nil
		},
	})
	funcTestCaseCls.Dict.SetStr("countTestCases", &object.BuiltinFunc{
		Name: "countTestCases",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(1), nil
		},
	})
	funcTestCaseCls.Dict.SetStr("run", &object.BuiltinFunc{
		Name: "run",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			var result *object.Instance
			if len(a) >= 2 {
				result, _ = a[1].(*object.Instance)
			}
			if result == nil {
				result = makeTestResult()
			}
			callResultMethod(result, "startTest", self)
			fn, _ := self.Dict.GetStr("_testFunc")
			if fn != nil {
				_, err := i.callObject(fn, nil, nil)
				if err != nil {
					if exc, ok := err.(*object.Exception); ok {
						errInfo := &object.Tuple{V: []object.Object{exc.Class, exc, object.None}}
						callResultMethod(result, "addError", self, errInfo)
					}
				} else {
					callResultMethod(result, "addSuccess", self)
				}
			}
			callResultMethod(result, "stopTest", self)
			return result, nil
		},
	})
	m.Dict.SetStr("FunctionTestCase", funcTestCaseCls)

	// ── TestSuite class ────────────────────────────────────────────────────────
	testSuiteCls := &object.Class{Name: "TestSuite", Dict: object.NewDict()}
	testSuiteCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("_tests", &object.List{})
			if len(a) >= 2 {
				tests := a[1]
				if l, ok := tests.(*object.List); ok {
					inst.Dict.SetStr("_tests", l)
				} else if t, ok := tests.(*object.Tuple); ok {
					inst.Dict.SetStr("_tests", &object.List{V: append([]object.Object{}, t.V...)})
				}
			}
			return object.None, nil
		},
	})
	testSuiteCls.Dict.SetStr("addTest", &object.BuiltinFunc{
		Name: "addTest",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			if v, ok := self.Dict.GetStr("_tests"); ok {
				if l, ok2 := v.(*object.List); ok2 {
					l.V = append(l.V, a[1])
				}
			}
			return object.None, nil
		},
	})
	testSuiteCls.Dict.SetStr("addTests", &object.BuiltinFunc{
		Name: "addTests",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			if v, ok := self.Dict.GetStr("_tests"); ok {
				if l, ok2 := v.(*object.List); ok2 {
					switch tests := a[1].(type) {
					case *object.List:
						l.V = append(l.V, tests.V...)
					case *object.Tuple:
						l.V = append(l.V, tests.V...)
					case *object.Instance:
						// another suite
						if tv, ok3 := tests.Dict.GetStr("_tests"); ok3 {
							if tl, ok4 := tv.(*object.List); ok4 {
								l.V = append(l.V, tl.V...)
							}
						}
					}
				}
			}
			return object.None, nil
		},
	})
	testSuiteCls.Dict.SetStr("countTestCases", &object.BuiltinFunc{
		Name: "countTestCases",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NewInt(0), nil
			}
			total := unitTestCountSuite(a[0].(*object.Instance), i)
			return object.NewInt(int64(total)), nil
		},
	})
	testSuiteCls.Dict.SetStr("__iter__", &object.BuiltinFunc{
		Name: "__iter__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Iter{Next: func() (object.Object, bool, error) { return nil, false, nil }}, nil
			}
			self := a[0].(*object.Instance)
			var items []object.Object
			if v, ok := self.Dict.GetStr("_tests"); ok {
				if l, ok2 := v.(*object.List); ok2 {
					items = l.V
				}
			}
			idx := 0
			return &object.Iter{Next: func() (object.Object, bool, error) {
				if idx >= len(items) {
					return nil, false, nil
				}
				v := items[idx]
				idx++
				return v, true, nil
			}}, nil
		},
	})
	testSuiteCls.Dict.SetStr("run", &object.BuiltinFunc{
		Name: "run",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return makeTestResult(), nil
			}
			self := a[0].(*object.Instance)
			var result *object.Instance
			if len(a) >= 2 {
				result, _ = a[1].(*object.Instance)
			}
			if result == nil {
				result = makeTestResult()
			}
			// Run each test in _tests.
			if v, ok := self.Dict.GetStr("_tests"); ok {
				if l, ok2 := v.(*object.List); ok2 {
					for _, test := range l.V {
						if runMethod, ok3 := i.getAttr(test, "run"); ok3 == nil {
							i.callObject(runMethod, []object.Object{test, result}, nil) //nolint:errcheck
						}
					}
				}
			}
			return result, nil
		},
	})
	testSuiteCls.Dict.SetStr("debug", &object.BuiltinFunc{
		Name: "debug",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	m.Dict.SetStr("TestSuite", testSuiteCls)

	// ── TestLoader class ──────────────────────────────────────────────────────
	testLoaderCls := &object.Class{Name: "TestLoader", Dict: object.NewDict()}
	testLoaderCls.Dict.SetStr("testMethodPrefix", &object.Str{V: "test"})
	testLoaderCls.Dict.SetStr("sortTestMethodsUsing", object.None)
	testLoaderCls.Dict.SetStr("suiteClass", testSuiteCls)
	testLoaderCls.Dict.SetStr("testNamePatterns", object.None)
	testLoaderCls.Dict.SetStr("errors", &object.List{})
	testLoaderCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("errors", &object.List{})
			return object.None, nil
		},
	})
	testLoaderCls.Dict.SetStr("getTestCaseNames", &object.BuiltinFunc{
		Name: "getTestCaseNames",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return &object.List{}, nil
			}
			cls, ok := a[1].(*object.Class)
			if !ok {
				return &object.List{}, nil
			}
			var names []string
			// Collect all test* method names from MRO.
			keys, _ := cls.Dict.Items()
			for _, k := range keys {
				if ks, ok2 := k.(*object.Str); ok2 {
					if strings.HasPrefix(ks.V, "test") {
						names = append(names, ks.V)
					}
				}
			}
			sort.Strings(names)
			items := make([]object.Object, len(names))
			for j, n := range names {
				items[j] = &object.Str{V: n}
			}
			return &object.List{V: items}, nil
		},
	})
	testLoaderCls.Dict.SetStr("loadTestsFromTestCase", &object.BuiltinFunc{
		Name: "loadTestsFromTestCase",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return &object.Instance{Class: testSuiteCls, Dict: object.NewDict()}, nil
			}
			cls, ok := a[1].(*object.Class)
			if !ok {
				return &object.Instance{Class: testSuiteCls, Dict: object.NewDict()}, nil
			}
			// Find test methods.
			var names []string
			keys, _ := cls.Dict.Items()
			for _, k := range keys {
				if ks, ok2 := k.(*object.Str); ok2 {
					if strings.HasPrefix(ks.V, "test") {
						names = append(names, ks.V)
					}
				}
			}
			sort.Strings(names)
			// Build suite.
			suite := &object.Instance{Class: testSuiteCls, Dict: object.NewDict()}
			tests := make([]object.Object, 0, len(names))
			for _, name := range names {
				// Create TestCase instance for each method.
				tc := &object.Instance{Class: cls, Dict: object.NewDict()}
				tc.Dict.SetStr("_testMethodName", &object.Str{V: name})
				tc.Dict.SetStr("_cleanups", &object.List{})
				tests = append(tests, tc)
			}
			suite.Dict.SetStr("_tests", &object.List{V: tests})
			return suite, nil
		},
	})
	testLoaderCls.Dict.SetStr("loadTestsFromModule", &object.BuiltinFunc{
		Name: "loadTestsFromModule",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			suite := &object.Instance{Class: testSuiteCls, Dict: object.NewDict()}
			suite.Dict.SetStr("_tests", &object.List{})
			return suite, nil
		},
	})
	testLoaderCls.Dict.SetStr("loadTestsFromName", &object.BuiltinFunc{
		Name: "loadTestsFromName",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			suite := &object.Instance{Class: testSuiteCls, Dict: object.NewDict()}
			suite.Dict.SetStr("_tests", &object.List{})
			return suite, nil
		},
	})
	testLoaderCls.Dict.SetStr("loadTestsFromNames", &object.BuiltinFunc{
		Name: "loadTestsFromNames",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			suite := &object.Instance{Class: testSuiteCls, Dict: object.NewDict()}
			suite.Dict.SetStr("_tests", &object.List{})
			return suite, nil
		},
	})
	testLoaderCls.Dict.SetStr("discover", &object.BuiltinFunc{
		Name: "discover",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			suite := &object.Instance{Class: testSuiteCls, Dict: object.NewDict()}
			suite.Dict.SetStr("_tests", &object.List{})
			return suite, nil
		},
	})
	m.Dict.SetStr("TestLoader", testLoaderCls)

	// ── TextTestResult ─────────────────────────────────────────────────────────
	textTestResultCls := &object.Class{
		Name:  "TextTestResult",
		Bases: []*object.Class{testResultCls},
		Dict:  object.NewDict(),
	}
	makeTextTestResult := func(stream, descriptions, verbosity object.Object) *object.Instance {
		r := makeTestResult()
		r.Class = textTestResultCls
		r.Dict.SetStr("stream", stream)
		r.Dict.SetStr("descriptions", descriptions)
		r.Dict.SetStr("verbosity", verbosity)
		return r
	}
	textTestResultCls.Dict.SetStr("printErrors", &object.BuiltinFunc{
		Name: "printErrors",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	textTestResultCls.Dict.SetStr("getDescription", &object.BuiltinFunc{
		Name: "getDescription",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 2 {
				return &object.Str{V: object.TypeName(a[1])}, nil
			}
			return &object.Str{V: "test"}, nil
		},
	})
	m.Dict.SetStr("TextTestResult", textTestResultCls)

	// ── TextTestRunner ─────────────────────────────────────────────────────────
	textTestRunnerCls := &object.Class{Name: "TextTestRunner", Dict: object.NewDict()}
	textTestRunnerCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			stream := object.Object(object.None)
			descriptions := object.Object(object.BoolOf(true))
			verbosity := object.Object(object.NewInt(1))
			failfast := object.Object(object.BoolOf(false))
			if kw != nil {
				if v, ok := kw.GetStr("stream"); ok {
					stream = v
				}
				if v, ok := kw.GetStr("descriptions"); ok {
					descriptions = v
				}
				if v, ok := kw.GetStr("verbosity"); ok {
					verbosity = v
				}
				if v, ok := kw.GetStr("failfast"); ok {
					failfast = v
				}
			}
			inst.Dict.SetStr("stream", stream)
			inst.Dict.SetStr("descriptions", descriptions)
			inst.Dict.SetStr("verbosity", verbosity)
			inst.Dict.SetStr("failfast", failfast)
			return object.None, nil
		},
	})
	textTestRunnerCls.Dict.SetStr("run", &object.BuiltinFunc{
		Name: "run",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return makeTestResult(), nil
			}
			self := a[0].(*object.Instance)
			test := a[1]
			stream, _ := self.Dict.GetStr("stream")
			descriptions, _ := self.Dict.GetStr("descriptions")
			verbosity, _ := self.Dict.GetStr("verbosity")
			if stream == nil {
				stream = object.None
			}
			if descriptions == nil {
				descriptions = object.BoolOf(true)
			}
			if verbosity == nil {
				verbosity = object.NewInt(1)
			}
			result := makeTextTestResult(stream, descriptions, verbosity)
			// Run the test (suite or case).
			if runMethod, err := i.getAttr(test, "run"); err == nil {
				i.callObject(runMethod, []object.Object{test, result}, nil) //nolint:errcheck
			}
			return result, nil
		},
	})
	m.Dict.SetStr("TextTestRunner", textTestRunnerCls)

	// ── Skip decorators ────────────────────────────────────────────────────────
	// skip(reason) — sets __unittest_skip__ and __unittest_skip_why__ on fn.
	m.Dict.SetStr("skip", &object.BuiltinFunc{
		Name: "skip",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "skip() requires a reason")
			}
			reason := ""
			if s, ok := a[0].(*object.Str); ok {
				reason = s.V
			}
			// Return a decorator.
			return &object.BuiltinFunc{
				Name: "skip_decorator",
				Call: func(_ any, da []object.Object, _ *object.Dict) (object.Object, error) {
					if len(da) < 1 {
						return object.None, nil
					}
					fn := da[0]
					// Set attributes on fn (works for *object.Function via the setAttr fix).
					i.setAttr(fn, "__unittest_skip__", object.BoolOf(true))       //nolint:errcheck
					i.setAttr(fn, "__unittest_skip_why__", &object.Str{V: reason}) //nolint:errcheck
					return fn, nil
				},
			}, nil
		},
	})
	m.Dict.SetStr("skipIf", &object.BuiltinFunc{
		Name: "skipIf",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "skipIf() requires condition and reason")
			}
			condition := object.Truthy(a[0])
			reason := ""
			if s, ok := a[1].(*object.Str); ok {
				reason = s.V
			}
			return &object.BuiltinFunc{
				Name: "skipIf_decorator",
				Call: func(_ any, da []object.Object, _ *object.Dict) (object.Object, error) {
					if len(da) < 1 {
						return object.None, nil
					}
					fn := da[0]
					if condition {
						i.setAttr(fn, "__unittest_skip__", object.BoolOf(true))       //nolint:errcheck
						i.setAttr(fn, "__unittest_skip_why__", &object.Str{V: reason}) //nolint:errcheck
					}
					return fn, nil
				},
			}, nil
		},
	})
	m.Dict.SetStr("skipUnless", &object.BuiltinFunc{
		Name: "skipUnless",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "skipUnless() requires condition and reason")
			}
			condition := object.Truthy(a[0])
			reason := ""
			if s, ok := a[1].(*object.Str); ok {
				reason = s.V
			}
			return &object.BuiltinFunc{
				Name: "skipUnless_decorator",
				Call: func(_ any, da []object.Object, _ *object.Dict) (object.Object, error) {
					if len(da) < 1 {
						return object.None, nil
					}
					fn := da[0]
					if !condition {
						i.setAttr(fn, "__unittest_skip__", object.BoolOf(true))       //nolint:errcheck
						i.setAttr(fn, "__unittest_skip_why__", &object.Str{V: reason}) //nolint:errcheck
					}
					return fn, nil
				},
			}, nil
		},
	})
	m.Dict.SetStr("expectedFailure", &object.BuiltinFunc{
		Name: "expectedFailure",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			fn := a[0]
			i.setAttr(fn, "__unittest_expecting_failure__", object.BoolOf(true)) //nolint:errcheck
			return fn, nil
		},
	})

	// ── main() ────────────────────────────────────────────────────────────────
	m.Dict.SetStr("main", &object.BuiltinFunc{
		Name: "main",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── defaultTestLoader ─────────────────────────────────────────────────────
	defaultLoader := &object.Instance{Class: testLoaderCls, Dict: object.NewDict()}
	defaultLoader.Dict.SetStr("errors", &object.List{})
	m.Dict.SetStr("defaultTestLoader", defaultLoader)

	return m
}

// ── package-level helpers ─────────────────────────────────────────────────────

func unitTestToFloat(o object.Object) (float64, bool) {
	switch v := o.(type) {
	case *object.Int:
		f, _ := v.V.Float64()
		return f, true
	case *object.Float:
		return v.V, true
	case *object.Bool:
		if v.V {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func unitTestCountSuite(suite *object.Instance, i *Interp) int {
	v, ok := suite.Dict.GetStr("_tests")
	if !ok {
		return 0
	}
	l, ok := v.(*object.List)
	if !ok {
		return 0
	}
	total := 0
	for _, test := range l.V {
		if sub, ok2 := test.(*object.Instance); ok2 {
			if countFn, ok3 := classLookup(sub.Class, "countTestCases"); ok3 {
				r, err := i.callObject(countFn, []object.Object{sub}, nil)
				if err == nil {
					if n, ok4 := toInt64(r); ok4 {
						total += int(n)
					}
				}
				continue
			}
		}
		total++
	}
	return total
}

func unitTestCountEqual(a, b object.Object) bool {
	var asSlice []object.Object
	switch v := a.(type) {
	case *object.List:
		asSlice = v.V
	case *object.Tuple:
		asSlice = v.V
	}
	var bsSlice []object.Object
	switch v := b.(type) {
	case *object.List:
		bsSlice = v.V
	case *object.Tuple:
		bsSlice = v.V
	}
	if len(asSlice) != len(bsSlice) {
		return false
	}
	counts := make(map[string]int)
	for _, x := range asSlice {
		counts[object.Repr(x)]++
	}
	for _, x := range bsSlice {
		counts[object.Repr(x)]--
		if counts[object.Repr(x)] < 0 {
			return false
		}
	}
	return true
}

func unitTestNoOpCtx() *object.Instance {
	cls := &object.Class{Name: "_NoOpCtx", Dict: object.NewDict()}
	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{
		Name: "__enter__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return a[0], nil
		},
	})
	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{
		Name: "__exit__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.BoolOf(false), nil
		},
	})
	return &object.Instance{Class: cls, Dict: object.NewDict()}
}

// pyContains checks whether container contains elem using Python membership rules.
func (i *Interp) pyContains(container, elem object.Object) (bool, error) {
	switch c := container.(type) {
	case *object.List:
		for _, v := range c.V {
			if eq, _ := object.Eq(elem, v); eq {
				return true, nil
			}
		}
		return false, nil
	case *object.Tuple:
		for _, v := range c.V {
			if eq, _ := object.Eq(elem, v); eq {
				return true, nil
			}
		}
		return false, nil
	case *object.Dict:
		_, ok := c.GetStr(object.Str_(elem))
		return ok, nil
	case *object.Set:
		for _, v := range c.Items() {
			if eq, _ := object.Eq(elem, v); eq {
				return true, nil
			}
		}
		return false, nil
	case *object.Str:
		if es, ok := elem.(*object.Str); ok {
			return strings.Contains(c.V, es.V), nil
		}
		return false, nil
	}
	return false, nil
}

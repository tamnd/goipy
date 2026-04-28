package vm

import "github.com/tamnd/goipy/object"

func (i *Interp) buildFuture() *object.Module {
	m := &object.Module{Name: "__future__", Dict: object.NewDict()}

	// ── CO_* constants ────────────────────────────────────────────────────────
	m.Dict.SetStr("CO_NESTED", object.NewInt(16))
	m.Dict.SetStr("CO_GENERATOR_ALLOWED", object.NewInt(0))
	m.Dict.SetStr("CO_FUTURE_DIVISION", object.NewInt(131072))
	m.Dict.SetStr("CO_FUTURE_ABSOLUTE_IMPORT", object.NewInt(262144))
	m.Dict.SetStr("CO_FUTURE_WITH_STATEMENT", object.NewInt(524288))
	m.Dict.SetStr("CO_FUTURE_PRINT_FUNCTION", object.NewInt(1048576))
	m.Dict.SetStr("CO_FUTURE_UNICODE_LITERALS", object.NewInt(2097152))
	m.Dict.SetStr("CO_FUTURE_BARRY_AS_BDFL", object.NewInt(4194304))
	m.Dict.SetStr("CO_FUTURE_GENERATOR_STOP", object.NewInt(8388608))
	m.Dict.SetStr("CO_FUTURE_ANNOTATIONS", object.NewInt(16777216))

	// ── _Feature class ────────────────────────────────────────────────────────

	featureCls := &object.Class{Name: "_Feature", Dict: object.NewDict()}

	featureCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// a[0]=self, a[1]=optionalRelease, a[2]=mandatoryRelease, a[3]=compiler_flag
			inst := a[0].(*object.Instance)
			var opt, mand, flag object.Object = object.None, object.None, object.NewInt(0)
			if len(a) >= 2 {
				opt = a[1]
			}
			if len(a) >= 3 {
				mand = a[2]
			}
			if len(a) >= 4 {
				flag = a[3]
			}
			inst.Dict.SetStr("_optional", opt)
			inst.Dict.SetStr("_mandatory", mand)
			inst.Dict.SetStr("compiler_flag", flag)
			return object.None, nil
		},
	})

	featureCls.Dict.SetStr("getOptionalRelease", &object.BuiltinFunc{
		Name: "getOptionalRelease",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			v, _ := inst.Dict.GetStr("_optional")
			return v, nil
		},
	})

	featureCls.Dict.SetStr("getMandatoryRelease", &object.BuiltinFunc{
		Name: "getMandatoryRelease",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			v, _ := inst.Dict.GetStr("_mandatory")
			return v, nil
		},
	})

	featureCls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			flag, _ := inst.Dict.GetStr("compiler_flag")
			return &object.Str{V: "_Feature(" + object.Repr(flag) + ")"}, nil
		},
	})

	m.Dict.SetStr("_Feature", featureCls)

	// relTuple builds a 5-element tuple representing a release version.
	relTuple := func(major, minor, micro int, level string, serial int) *object.Tuple {
		return &object.Tuple{V: []object.Object{
			object.NewInt(int64(major)),
			object.NewInt(int64(minor)),
			object.NewInt(int64(micro)),
			&object.Str{V: level},
			object.NewInt(int64(serial)),
		}}
	}

	// mkFeature creates a _Feature instance.
	mkFeature := func(opt *object.Tuple, mand object.Object, flag int64) *object.Instance {
		inst := &object.Instance{Class: featureCls, Dict: object.NewDict()}
		inst.Dict.SetStr("_optional", opt)
		inst.Dict.SetStr("_mandatory", mand)
		inst.Dict.SetStr("compiler_flag", object.NewInt(flag))
		return inst
	}

	// ── feature instances (in definition order matching all_feature_names) ────

	m.Dict.SetStr("nested_scopes", mkFeature(
		relTuple(2, 1, 0, "beta", 1),
		relTuple(2, 2, 0, "alpha", 0),
		16, // CO_NESTED
	))

	m.Dict.SetStr("generators", mkFeature(
		relTuple(2, 2, 0, "alpha", 1),
		relTuple(2, 3, 0, "final", 0),
		0,
	))

	m.Dict.SetStr("division", mkFeature(
		relTuple(2, 2, 0, "alpha", 2),
		relTuple(3, 0, 0, "alpha", 0),
		131072, // CO_FUTURE_DIVISION
	))

	m.Dict.SetStr("absolute_import", mkFeature(
		relTuple(2, 5, 0, "alpha", 1),
		relTuple(3, 0, 0, "alpha", 0),
		262144, // CO_FUTURE_ABSOLUTE_IMPORT
	))

	m.Dict.SetStr("with_statement", mkFeature(
		relTuple(2, 5, 0, "alpha", 1),
		relTuple(2, 6, 0, "alpha", 0),
		524288, // CO_FUTURE_WITH_STATEMENT
	))

	m.Dict.SetStr("print_function", mkFeature(
		relTuple(2, 6, 0, "alpha", 2),
		relTuple(3, 0, 0, "alpha", 0),
		1048576, // CO_FUTURE_PRINT_FUNCTION
	))

	m.Dict.SetStr("unicode_literals", mkFeature(
		relTuple(2, 6, 0, "alpha", 2),
		relTuple(3, 0, 0, "alpha", 0),
		2097152, // CO_FUTURE_UNICODE_LITERALS
	))

	m.Dict.SetStr("barry_as_FLUFL", mkFeature(
		relTuple(3, 1, 0, "alpha", 2),
		relTuple(4, 0, 0, "alpha", 0),
		4194304, // CO_FUTURE_BARRY_AS_BDFL
	))

	m.Dict.SetStr("generator_stop", mkFeature(
		relTuple(3, 5, 0, "beta", 1),
		relTuple(3, 7, 0, "alpha", 0),
		8388608, // CO_FUTURE_GENERATOR_STOP
	))

	m.Dict.SetStr("annotations", mkFeature(
		relTuple(3, 7, 0, "beta", 1),
		object.None, // mandatory = None (not yet required)
		16777216,    // CO_FUTURE_ANNOTATIONS
	))

	// ── all_feature_names ─────────────────────────────────────────────────────

	featureNames := []object.Object{
		&object.Str{V: "nested_scopes"},
		&object.Str{V: "generators"},
		&object.Str{V: "division"},
		&object.Str{V: "absolute_import"},
		&object.Str{V: "with_statement"},
		&object.Str{V: "print_function"},
		&object.Str{V: "unicode_literals"},
		&object.Str{V: "barry_as_FLUFL"},
		&object.Str{V: "generator_stop"},
		&object.Str{V: "annotations"},
	}
	m.Dict.SetStr("all_feature_names", &object.List{V: featureNames})

	// ── __all__ ───────────────────────────────────────────────────────────────

	m.Dict.SetStr("__all__", &object.List{V: featureNames})

	return m
}

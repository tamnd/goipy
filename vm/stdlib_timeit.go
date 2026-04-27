package vm

import (
	"time"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildTimeit() *object.Module {
	m := &object.Module{Name: "timeit", Dict: object.NewDict()}

	// default_timer: returns elapsed seconds since module init (monotonic)
	epoch := time.Now()
	defaultTimer := &object.BuiltinFunc{
		Name: "default_timer",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Float{V: time.Since(epoch).Seconds()}, nil
		},
	}
	m.Dict.SetStr("default_timer", defaultTimer)

	// ── Timer class ───────────────────────────────────────────────────────────

	timerCls := &object.Class{Name: "Timer", Dict: object.NewDict()}

	timerCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			var stmt, setup, timer object.Object
			stmt = &object.Str{V: "pass"}
			setup = &object.Str{V: "pass"}
			timer = defaultTimer
			if len(a) >= 2 {
				stmt = a[1]
			}
			if len(a) >= 3 {
				setup = a[2]
			}
			if len(a) >= 4 {
				timer = a[3]
			}
			if kw != nil {
				if v, ok := kw.GetStr("stmt"); ok {
					stmt = v
				}
				if v, ok := kw.GetStr("setup"); ok {
					setup = v
				}
				if v, ok := kw.GetStr("timer"); ok {
					timer = v
				}
			}
			inst.Dict.SetStr("_stmt", stmt)
			inst.Dict.SetStr("_setup", setup)
			inst.Dict.SetStr("_timer", timer)
			return object.None, nil
		},
	})

	// runOnce executes stmt: callable → callObject, string → no-op.
	runOnce := func(ii any, stmt object.Object) error {
		if _, ok := stmt.(*object.Str); ok {
			return nil
		}
		_, err := ii.(*Interp).callObject(stmt, nil, nil)
		return err
	}

	timerCls.Dict.SetStr("timeit", &object.BuiltinFunc{
		Name: "timeit",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			number := int64(1000000)
			if len(a) >= 2 {
				if n, ok := toInt64(a[1]); ok {
					number = n
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("number"); ok {
					if n, ok2 := toInt64(v); ok2 {
						number = n
					}
				}
			}
			stmtObj, _ := inst.Dict.GetStr("_stmt")
			setupObj, _ := inst.Dict.GetStr("_setup")
			if err := runOnce(ii, setupObj); err != nil {
				return nil, err
			}
			start := time.Now()
			var j int64
			for j = 0; j < number; j++ {
				if err := runOnce(ii, stmtObj); err != nil {
					return nil, err
				}
			}
			return &object.Float{V: time.Since(start).Seconds()}, nil
		},
	})

	timerCls.Dict.SetStr("repeat", &object.BuiltinFunc{
		Name: "repeat",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			repeat := int64(5)
			number := int64(1000000)
			if len(a) >= 2 {
				if n, ok := toInt64(a[1]); ok {
					repeat = n
				}
			}
			if len(a) >= 3 {
				if n, ok := toInt64(a[2]); ok {
					number = n
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("repeat"); ok {
					if n, ok2 := toInt64(v); ok2 {
						repeat = n
					}
				}
				if v, ok := kw.GetStr("number"); ok {
					if n, ok2 := toInt64(v); ok2 {
						number = n
					}
				}
			}
			timeitFn, _ := timerCls.Dict.GetStr("timeit")
			results := make([]object.Object, 0, repeat)
			var r int64
			for r = 0; r < repeat; r++ {
				res, err := ii.(*Interp).callObject(timeitFn, []object.Object{inst, object.NewInt(number)}, nil)
				if err != nil {
					return nil, err
				}
				results = append(results, res)
			}
			return &object.List{V: results}, nil
		},
	})

	timerCls.Dict.SetStr("autorange", &object.BuiltinFunc{
		Name: "autorange",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			var callback object.Object
			if len(a) >= 2 && a[1] != object.None {
				callback = a[1]
			}
			if kw != nil {
				if v, ok := kw.GetStr("callback"); ok && v != object.None {
					callback = v
				}
			}
			timeitFn, _ := timerCls.Dict.GetStr("timeit")
			interp := ii.(*Interp)
			// CPython sequence: 1, 2, 5, 10, 20, 50, 100, ...
			number := int64(1)
			step := 0
			for {
				res, err := interp.callObject(timeitFn, []object.Object{inst, object.NewInt(number)}, nil)
				if err != nil {
					return nil, err
				}
				timeTaken := 0.0
				if f, ok := res.(*object.Float); ok {
					timeTaken = f.V
				}
				if callback != nil {
					interp.callObject(callback, []object.Object{object.NewInt(number), &object.Float{V: timeTaken}}, nil) //nolint
				}
				if timeTaken >= 0.2 || number > 1_000_000_000 {
					return &object.Tuple{V: []object.Object{object.NewInt(number), &object.Float{V: timeTaken}}}, nil
				}
				// advance: 1→2→5→10→20→50→100→...
				step++
				switch step % 3 {
				case 1:
					number *= 2
				case 2:
					number = number * 5 / 2
				case 0:
					number *= 2
				}
			}
		},
	})

	timerCls.Dict.SetStr("print_exc", &object.BuiltinFunc{
		Name: "print_exc",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("Timer", timerCls)

	// ── Module-level helpers ──────────────────────────────────────────────────

	newTimerInst := func(stmt, setup, timer object.Object) *object.Instance {
		inst := &object.Instance{Class: timerCls, Dict: object.NewDict()}
		inst.Dict.SetStr("_stmt", stmt)
		inst.Dict.SetStr("_setup", setup)
		if timer == nil || timer == object.None {
			inst.Dict.SetStr("_timer", defaultTimer)
		} else {
			inst.Dict.SetStr("_timer", timer)
		}
		return inst
	}

	timeitFn, _ := timerCls.Dict.GetStr("timeit")
	repeatFn, _ := timerCls.Dict.GetStr("repeat")

	m.Dict.SetStr("timeit", &object.BuiltinFunc{
		Name: "timeit",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			stmt := object.Object(&object.Str{V: "pass"})
			setup := object.Object(&object.Str{V: "pass"})
			var timer object.Object
			number := int64(1000000)
			if len(a) >= 1 {
				stmt = a[0]
			}
			if len(a) >= 2 {
				setup = a[1]
			}
			if len(a) >= 3 {
				timer = a[2]
			}
			if kw != nil {
				if v, ok := kw.GetStr("stmt"); ok {
					stmt = v
				}
				if v, ok := kw.GetStr("setup"); ok {
					setup = v
				}
				if v, ok := kw.GetStr("timer"); ok {
					timer = v
				}
				if v, ok := kw.GetStr("number"); ok {
					if n, ok2 := toInt64(v); ok2 {
						number = n
					}
				}
			}
			inst := newTimerInst(stmt, setup, timer)
			return ii.(*Interp).callObject(timeitFn, []object.Object{inst, object.NewInt(number)}, nil)
		},
	})

	m.Dict.SetStr("repeat", &object.BuiltinFunc{
		Name: "repeat",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			stmt := object.Object(&object.Str{V: "pass"})
			setup := object.Object(&object.Str{V: "pass"})
			var timer object.Object
			repeat := int64(5)
			number := int64(1000000)
			if len(a) >= 1 {
				stmt = a[0]
			}
			if len(a) >= 2 {
				setup = a[1]
			}
			if len(a) >= 3 {
				timer = a[2]
			}
			if kw != nil {
				if v, ok := kw.GetStr("stmt"); ok {
					stmt = v
				}
				if v, ok := kw.GetStr("setup"); ok {
					setup = v
				}
				if v, ok := kw.GetStr("timer"); ok {
					timer = v
				}
				if v, ok := kw.GetStr("repeat"); ok {
					if n, ok2 := toInt64(v); ok2 {
						repeat = n
					}
				}
				if v, ok := kw.GetStr("number"); ok {
					if n, ok2 := toInt64(v); ok2 {
						number = n
					}
				}
			}
			inst := newTimerInst(stmt, setup, timer)
			return ii.(*Interp).callObject(repeatFn, []object.Object{inst, object.NewInt(repeat), object.NewInt(number)}, nil)
		},
	})

	return m
}

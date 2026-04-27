package vm

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildTracemalloc() *object.Module {
	m := &object.Module{Name: "tracemalloc", Dict: object.NewDict()}

	// module-level state (no actual malloc hooking in goipy)
	type tmState struct {
		tracing bool
		nframe  int
	}
	state := &tmState{nframe: 1}

	// ── Frame class ───────────────────────────────────────────────────────────

	frameCls := &object.Class{Name: "Frame", Dict: object.NewDict()}
	frameCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			var filename object.Object = &object.Str{V: "<unknown>"}
			var lineno object.Object = object.NewInt(0)
			if len(a) >= 2 {
				filename = a[1]
			}
			if len(a) >= 3 {
				lineno = a[2]
			}
			if kw != nil {
				if v, ok := kw.GetStr("filename"); ok {
					filename = v
				}
				if v, ok := kw.GetStr("lineno"); ok {
					lineno = v
				}
			}
			inst.Dict.SetStr("filename", filename)
			inst.Dict.SetStr("lineno", lineno)
			return object.None, nil
		},
	})
	frameCls.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			fn, _ := inst.Dict.GetStr("filename")
			ln, _ := inst.Dict.GetStr("lineno")
			return &object.Str{V: fmt.Sprintf("<Frame filename=%v lineno=%v>", fn, ln)}, nil
		},
	})
	m.Dict.SetStr("Frame", frameCls)

	// ── Traceback class ───────────────────────────────────────────────────────

	tracebackCls := &object.Class{Name: "Traceback", Dict: object.NewDict()}
	tracebackCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			frames := &object.List{V: nil}
			if len(a) >= 2 {
				if lst, ok := a[1].(*object.List); ok {
					frames = lst
				} else if tpl, ok := a[1].(*object.Tuple); ok {
					frames = &object.List{V: tpl.V}
				}
			}
			inst.Dict.SetStr("_frames", frames)
			inst.Dict.SetStr("total_nframe", object.None)
			return object.None, nil
		},
	})
	tracebackCls.Dict.SetStr("format", &object.BuiltinFunc{
		Name: "format",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			framesObj, _ := inst.Dict.GetStr("_frames")
			lst, ok := framesObj.(*object.List)
			if !ok || len(lst.V) == 0 {
				return &object.List{V: nil}, nil
			}
			lines := make([]object.Object, 0, len(lst.V))
			for _, fobj := range lst.V {
				finst, ok := fobj.(*object.Instance)
				if !ok {
					continue
				}
				fnObj, _ := finst.Dict.GetStr("filename")
				lnObj, _ := finst.Dict.GetStr("lineno")
				fn := "<unknown>"
				if s, ok2 := fnObj.(*object.Str); ok2 {
					fn = s.V
				}
				ln := int64(0)
				if n, ok2 := toInt64(lnObj); ok2 {
					ln = n
				}
				line := fmt.Sprintf(`  File "%s", line %d`, fn, ln)
				lines = append(lines, &object.Str{V: line})
			}
			return &object.List{V: lines}, nil
		},
	})
	tracebackCls.Dict.SetStr("__len__", &object.BuiltinFunc{
		Name: "__len__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			framesObj, _ := inst.Dict.GetStr("_frames")
			if lst, ok := framesObj.(*object.List); ok {
				return object.NewInt(int64(len(lst.V))), nil
			}
			return object.NewInt(0), nil
		},
	})
	m.Dict.SetStr("Traceback", tracebackCls)

	// helper: build an empty Traceback instance
	emptyTraceback := func() *object.Instance {
		tb := &object.Instance{Class: tracebackCls, Dict: object.NewDict()}
		tb.Dict.SetStr("_frames", &object.List{V: nil})
		tb.Dict.SetStr("total_nframe", object.None)
		return tb
	}

	// ── Trace class ───────────────────────────────────────────────────────────

	traceCls2 := &object.Class{Name: "Trace", Dict: object.NewDict()}
	traceCls2.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			var size object.Object = object.NewInt(0)
			var domain object.Object = object.NewInt(0)
			var tb object.Object = emptyTraceback()
			if len(a) >= 2 {
				tb = a[1]
			}
			if len(a) >= 3 {
				size = a[2]
			}
			if len(a) >= 4 {
				domain = a[3]
			}
			inst.Dict.SetStr("traceback", tb)
			inst.Dict.SetStr("size", size)
			inst.Dict.SetStr("domain", domain)
			return object.None, nil
		},
	})
	m.Dict.SetStr("Trace", traceCls2)

	// ── Statistic class ───────────────────────────────────────────────────────

	statisticCls := &object.Class{Name: "Statistic", Dict: object.NewDict()}
	statisticCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			var tb object.Object = emptyTraceback()
			var count object.Object = object.NewInt(0)
			var size object.Object = object.NewInt(0)
			if len(a) >= 2 {
				tb = a[1]
			}
			if len(a) >= 3 {
				count = a[2]
			}
			if len(a) >= 4 {
				size = a[3]
			}
			inst.Dict.SetStr("traceback", tb)
			inst.Dict.SetStr("count", count)
			inst.Dict.SetStr("size", size)
			return object.None, nil
		},
	})
	statisticCls.Dict.SetStr("__str__", &object.BuiltinFunc{
		Name: "__str__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			count, _ := inst.Dict.GetStr("count")
			size, _ := inst.Dict.GetStr("size")
			return &object.Str{V: fmt.Sprintf("<traceback>: size=%v, count=%v", size, count)}, nil
		},
	})
	m.Dict.SetStr("Statistic", statisticCls)

	// ── StatisticDiff class ───────────────────────────────────────────────────

	statisticDiffCls := &object.Class{Name: "StatisticDiff", Dict: object.NewDict()}
	statisticDiffCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			var tb object.Object = emptyTraceback()
			var count object.Object = object.NewInt(0)
			var size object.Object = object.NewInt(0)
			var countDiff object.Object = object.NewInt(0)
			var sizeDiff object.Object = object.NewInt(0)
			if len(a) >= 2 {
				tb = a[1]
			}
			if len(a) >= 3 {
				count = a[2]
			}
			if len(a) >= 4 {
				size = a[3]
			}
			if len(a) >= 5 {
				countDiff = a[4]
			}
			if len(a) >= 6 {
				sizeDiff = a[5]
			}
			inst.Dict.SetStr("traceback", tb)
			inst.Dict.SetStr("count", count)
			inst.Dict.SetStr("size", size)
			inst.Dict.SetStr("count_diff", countDiff)
			inst.Dict.SetStr("size_diff", sizeDiff)
			return object.None, nil
		},
	})
	statisticDiffCls.Dict.SetStr("__str__", &object.BuiltinFunc{
		Name: "__str__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			count, _ := inst.Dict.GetStr("count")
			size, _ := inst.Dict.GetStr("size")
			cd, _ := inst.Dict.GetStr("count_diff")
			sd, _ := inst.Dict.GetStr("size_diff")
			return &object.Str{V: fmt.Sprintf("<traceback>: size=%v (%v), count=%v (%v)", size, sd, count, cd)}, nil
		},
	})
	m.Dict.SetStr("StatisticDiff", statisticDiffCls)

	// ── Snapshot class ────────────────────────────────────────────────────────

	snapshotCls := &object.Class{Name: "Snapshot", Dict: object.NewDict()}
	snapshotCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			traces := &object.Tuple{V: nil}
			var tbLimit object.Object = object.NewInt(int64(state.nframe))
			if len(a) >= 2 {
				if t, ok := a[1].(*object.Tuple); ok {
					traces = t
				}
			}
			if len(a) >= 3 {
				tbLimit = a[2]
			}
			inst.Dict.SetStr("traces", traces)
			inst.Dict.SetStr("traceback_limit", tbLimit)
			return object.None, nil
		},
	})
	snapshotCls.Dict.SetStr("statistics", &object.BuiltinFunc{
		Name: "statistics",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: nil}, nil
		},
	})
	snapshotCls.Dict.SetStr("compare_to", &object.BuiltinFunc{
		Name: "compare_to",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: nil}, nil
		},
	})
	snapshotCls.Dict.SetStr("filter_traces", &object.BuiltinFunc{
		Name: "filter_traces",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// return a new empty Snapshot with same traceback_limit
			newSnap := &object.Instance{Class: snapshotCls, Dict: object.NewDict()}
			var tbLimit object.Object = object.NewInt(int64(state.nframe))
			if len(a) >= 1 {
				if inst, ok := a[0].(*object.Instance); ok {
					if v, ok2 := inst.Dict.GetStr("traceback_limit"); ok2 {
						tbLimit = v
					}
				}
			}
			newSnap.Dict.SetStr("traces", &object.Tuple{V: nil})
			newSnap.Dict.SetStr("traceback_limit", tbLimit)
			return newSnap, nil
		},
	})
	snapshotCls.Dict.SetStr("dump", &object.BuiltinFunc{
		Name: "dump",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			path, ok := a[1].(*object.Str)
			if !ok {
				return object.None, nil
			}
			os.WriteFile(path.V, []byte(`{}`), 0o644) //nolint
			return object.None, nil
		},
	})
	snapshotCls.Dict.SetStr("load", &object.BuiltinFunc{
		Name: "load",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// classmethod pattern: first arg may be class or instance
			var path string
			for _, arg := range a {
				if s, ok := arg.(*object.Str); ok {
					path = s.V
					break
				}
			}
			// read file to validate it exists (ignore content — always empty)
			if path != "" {
				data, err := os.ReadFile(path)
				if err != nil {
					return nil, object.Errorf(i.valueErr, "could not load snapshot: %v", err)
				}
				var tmp map[string]any
				json.Unmarshal(data, &tmp) //nolint
			}
			snap := &object.Instance{Class: snapshotCls, Dict: object.NewDict()}
			snap.Dict.SetStr("traces", &object.Tuple{V: nil})
			snap.Dict.SetStr("traceback_limit", object.NewInt(1))
			return snap, nil
		},
	})
	m.Dict.SetStr("Snapshot", snapshotCls)

	// ── Filter class ──────────────────────────────────────────────────────────

	filterCls := &object.Class{Name: "Filter", Dict: object.NewDict()}
	filterCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			var inclusive object.Object = &object.Bool{V: true}
			var filenamePattern object.Object = &object.Str{V: "*"}
			var lineno object.Object = object.None
			var allFrames object.Object = &object.Bool{V: false}
			var domain object.Object = object.None
			if len(a) >= 2 {
				inclusive = a[1]
			}
			if len(a) >= 3 {
				filenamePattern = a[2]
			}
			if len(a) >= 4 {
				lineno = a[3]
			}
			if len(a) >= 5 {
				allFrames = a[4]
			}
			if len(a) >= 6 {
				domain = a[5]
			}
			if kw != nil {
				if v, ok := kw.GetStr("inclusive"); ok {
					inclusive = v
				}
				if v, ok := kw.GetStr("filename_pattern"); ok {
					filenamePattern = v
				}
				if v, ok := kw.GetStr("lineno"); ok {
					lineno = v
				}
				if v, ok := kw.GetStr("all_frames"); ok {
					allFrames = v
				}
				if v, ok := kw.GetStr("domain"); ok {
					domain = v
				}
			}
			inst.Dict.SetStr("inclusive", inclusive)
			inst.Dict.SetStr("filename_pattern", filenamePattern)
			inst.Dict.SetStr("lineno", lineno)
			inst.Dict.SetStr("all_frames", allFrames)
			inst.Dict.SetStr("domain", domain)
			return object.None, nil
		},
	})
	m.Dict.SetStr("Filter", filterCls)

	// ── DomainFilter class ────────────────────────────────────────────────────

	domainFilterCls := &object.Class{Name: "DomainFilter", Dict: object.NewDict()}
	domainFilterCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			var inclusive object.Object = &object.Bool{V: true}
			var domain object.Object = object.NewInt(0)
			if len(a) >= 2 {
				inclusive = a[1]
			}
			if len(a) >= 3 {
				domain = a[2]
			}
			if kw != nil {
				if v, ok := kw.GetStr("inclusive"); ok {
					inclusive = v
				}
				if v, ok := kw.GetStr("domain"); ok {
					domain = v
				}
			}
			inst.Dict.SetStr("inclusive", inclusive)
			inst.Dict.SetStr("domain", domain)
			return object.None, nil
		},
	})
	m.Dict.SetStr("DomainFilter", domainFilterCls)

	// ── Module-level functions ─────────────────────────────────────────────────

	m.Dict.SetStr("start", &object.BuiltinFunc{
		Name: "start",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			nframe := 1
			if len(a) >= 1 {
				if n, ok := toInt64(a[0]); ok && n >= 1 {
					nframe = int(n)
				}
			}
			if kw != nil {
				if v, ok := kw.GetStr("nframe"); ok {
					if n, ok2 := toInt64(v); ok2 && n >= 1 {
						nframe = int(n)
					}
				}
			}
			state.tracing = true
			state.nframe = nframe
			return object.None, nil
		},
	})

	m.Dict.SetStr("stop", &object.BuiltinFunc{
		Name: "stop",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			state.tracing = false
			return object.None, nil
		},
	})

	m.Dict.SetStr("is_tracing", &object.BuiltinFunc{
		Name: "is_tracing",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Bool{V: state.tracing}, nil
		},
	})

	m.Dict.SetStr("clear_traces", &object.BuiltinFunc{
		Name: "clear_traces",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("get_traced_memory", &object.BuiltinFunc{
		Name: "get_traced_memory",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{object.NewInt(0), object.NewInt(0)}}, nil
		},
	})

	m.Dict.SetStr("get_tracemalloc_memory", &object.BuiltinFunc{
		Name: "get_tracemalloc_memory",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(0), nil
		},
	})

	m.Dict.SetStr("get_traceback_limit", &object.BuiltinFunc{
		Name: "get_traceback_limit",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(state.nframe)), nil
		},
	})

	m.Dict.SetStr("reset_peak", &object.BuiltinFunc{
		Name: "reset_peak",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("get_object_traceback", &object.BuiltinFunc{
		Name: "get_object_traceback",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("take_snapshot", &object.BuiltinFunc{
		Name: "take_snapshot",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			snap := &object.Instance{Class: snapshotCls, Dict: object.NewDict()}
			snap.Dict.SetStr("traces", &object.Tuple{V: nil})
			snap.Dict.SetStr("traceback_limit", object.NewInt(int64(state.nframe)))
			return snap, nil
		},
	})

	return m
}

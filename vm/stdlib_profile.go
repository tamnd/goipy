package vm

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/tamnd/goipy/object"
)

// profileEntry is the internal record for a single profiled call.
type profileEntry struct {
	Filename string  `json:"f"`
	Lineno   int     `json:"l"`
	Funcname string  `json:"n"`
	NCalls   int     `json:"c"`
	Tottime  float64 `json:"t"`
	Cumtime  float64 `json:"u"`
}

// ── shared helpers ────────────────────────────────────────────────────────────

// profileGetEntries retrieves the _entries list stored on a Profile instance.
func profileGetEntries(inst *object.Instance) []profileEntry {
	v, ok := inst.Dict.GetStr("_entries_json")
	if !ok || v == nil {
		return nil
	}
	s, ok := v.(*object.Str)
	if !ok {
		return nil
	}
	var entries []profileEntry
	json.Unmarshal([]byte(s.V), &entries) //nolint
	return entries
}

// profileSetEntries serialises entries into the instance dict.
func profileSetEntries(inst *object.Instance, entries []profileEntry) {
	data, _ := json.Marshal(entries)
	inst.Dict.SetStr("_entries_json", &object.Str{V: string(data)})
}

// profilePrintTable writes the stats table to w.
func profilePrintTable(w io.Writer, entries []profileEntry, totalTime float64) {
	totalCalls := 0
	for _, e := range entries {
		totalCalls += e.NCalls
	}
	fmt.Fprintf(w, "         %d function calls in %.6f seconds\n\n", totalCalls, totalTime)
	fmt.Fprintf(w, "   ncalls  tottime  percall  cumtime  percall filename:lineno(function)\n")
	for _, e := range entries {
		percallTot := 0.0
		if e.NCalls > 0 {
			percallTot = e.Tottime / float64(e.NCalls)
		}
		percallCum := 0.0
		if e.NCalls > 0 {
			percallCum = e.Cumtime / float64(e.NCalls)
		}
		fmt.Fprintf(w, "   %6d  %.6f  %.6f  %.6f  %.6f %s:%d(%s)\n",
			e.NCalls, e.Tottime, percallTot, e.Cumtime, percallCum,
			e.Filename, e.Lineno, e.Funcname)
	}
}

// profileResolveWriter extracts an io.Writer from a file object or falls back to stdout.
func profileResolveWriter(ii any, fileObj object.Object) io.Writer {
	if ts, ok := fileObj.(*object.TextStream); ok {
		if w, ok2 := ts.W.(io.Writer); ok2 {
			return w
		}
	}
	interp := ii.(*Interp)
	if fileObj != nil && fileObj != object.None {
		writeMethod, err := interp.getAttr(fileObj, "write")
		if err == nil {
			return &pythonWriter{interp: interp, writeMethod: writeMethod}
		}
	}
	return interp.Stdout
}

type pythonWriter struct {
	interp      *Interp
	writeMethod object.Object
}

func (pw *pythonWriter) Write(p []byte) (int, error) {
	pw.interp.callObject(pw.writeMethod, []object.Object{&object.Str{V: string(p)}}, nil) //nolint
	return len(p), nil
}

// ── Profile class builder ─────────────────────────────────────────────────────

func buildProfileClass(i *Interp, name string) *object.Class {
	cls := &object.Class{Name: name, Dict: object.NewDict()}

	// Profile.__init__(self, timer=None, timeunit=0.0, subcalls=True, builtins=True)
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("_enabled", object.BoolOf(false))
			inst.Dict.SetStr("_start", object.NewInt(0))
			inst.Dict.SetStr("_total_tt", &object.Float{V: 0})
			profileSetEntries(inst, nil)
			return object.None, nil
		},
	})

	// enable()
	cls.Dict.SetStr("enable", &object.BuiltinFunc{
		Name: "enable",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("_enabled", object.BoolOf(true))
			inst.Dict.SetStr("_start_ns", object.NewInt(time.Now().UnixNano()))
			return object.None, nil
		},
	})

	// disable()
	cls.Dict.SetStr("disable", &object.BuiltinFunc{
		Name: "disable",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			inst.Dict.SetStr("_enabled", object.BoolOf(false))
			// accumulate total time
			if startObj, ok := inst.Dict.GetStr("_start_ns"); ok {
				if startInt, ok2 := startObj.(*object.Int); ok2 {
					elapsed := float64(time.Now().UnixNano()-startInt.Int64()) / 1e9
					if ttObj, ok3 := inst.Dict.GetStr("_total_tt"); ok3 {
						if ttF, ok4 := ttObj.(*object.Float); ok4 {
							elapsed += ttF.V
						}
					}
					inst.Dict.SetStr("_total_tt", &object.Float{V: elapsed})
				}
			}
			return object.None, nil
		},
	})

	// create_stats() — finalise stats (no-op if already done)
	cls.Dict.SetStr("create_stats", &object.BuiltinFunc{
		Name: "create_stats",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// runcall(func, *args, **kwds)
	cls.Dict.SetStr("runcall", &object.BuiltinFunc{
		Name: "runcall",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "runcall() requires function argument")
			}
			inst := a[0].(*object.Instance)
			fn := a[1]
			fnArgs := a[2:]
			interp := ii.(*Interp)

			start := time.Now()
			result, err := interp.callObject(fn, fnArgs, kw)
			elapsed := time.Since(start).Seconds()

			// Record entry
			entries := profileGetEntries(inst)
			filename := "<profile>"
			lineno := 0
			funcname := "<unknown>"
			if fn2, ok := fn.(*object.Function); ok {
				filename = fn2.Code.Filename
				lineno = fn2.Code.FirstLineNo
				funcname = fn2.Code.Name
			} else if bf, ok := fn.(*object.BuiltinFunc); ok {
				funcname = bf.Name
			}
			entries = append(entries, profileEntry{
				Filename: filename, Lineno: lineno, Funcname: funcname,
				NCalls: 1, Tottime: elapsed, Cumtime: elapsed,
			})
			profileSetEntries(inst, entries)

			// Accumulate total time
			totalTT := elapsed
			if ttObj, ok := inst.Dict.GetStr("_total_tt"); ok {
				if ttF, ok2 := ttObj.(*object.Float); ok2 {
					totalTT += ttF.V
				}
			}
			inst.Dict.SetStr("_total_tt", &object.Float{V: totalTT})

			if err != nil {
				return nil, err
			}
			return result, nil
		},
	})

	// run(cmd) — stub (string execution not wired)
	cls.Dict.SetStr("run", &object.BuiltinFunc{
		Name: "run",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// runctx(cmd, globals, locals) — stub
	cls.Dict.SetStr("runctx", &object.BuiltinFunc{
		Name: "runctx",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// print_stats(sort=-1) — writes to stderr to avoid polluting captured stdout
	cls.Dict.SetStr("print_stats", &object.BuiltinFunc{
		Name: "print_stats",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			interp := ii.(*Interp)
			entries := profileGetEntries(inst)
			totalTT := 0.0
			if ttObj, ok := inst.Dict.GetStr("_total_tt"); ok {
				if ttF, ok2 := ttObj.(*object.Float); ok2 {
					totalTT = ttF.V
				}
			}
			w := io.Writer(interp.Stderr)
			profilePrintTable(w, entries, totalTT)
			return object.None, nil
		},
	})

	// dump_stats(filename)
	cls.Dict.SetStr("dump_stats", &object.BuiltinFunc{
		Name: "dump_stats",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "dump_stats() requires filename argument")
			}
			inst := a[0].(*object.Instance)
			filename := object.Str_(a[1])
			entries := profileGetEntries(inst)
			totalTT := 0.0
			if ttObj, ok := inst.Dict.GetStr("_total_tt"); ok {
				if ttF, ok2 := ttObj.(*object.Float); ok2 {
					totalTT = ttF.V
				}
			}
			data := map[string]any{
				"entries":  entries,
				"total_tt": totalTT,
			}
			b, err2 := json.Marshal(data)
			if err2 != nil {
				return nil, object.Errorf(i.runtimeErr, "dump_stats: marshal error: %v", err2)
			}
			if err2 = os.WriteFile(filename, b, 0o644); err2 != nil {
				return nil, object.Errorf(i.osErr, "dump_stats: %v", err2)
			}
			return object.None, nil
		},
	})

	// calibrate(m) — profile module only; returns 0.0 stub
	cls.Dict.SetStr("calibrate", &object.BuiltinFunc{
		Name: "calibrate",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Float{V: 0.0}, nil
		},
	})

	// __enter__() — enable + return self
	cls.Dict.SetStr("__enter__", &object.BuiltinFunc{
		Name: "__enter__",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			enableFn, _ := cls.Dict.GetStr("enable")
			if ef, ok := enableFn.(*object.BuiltinFunc); ok {
				ef.Call(ii, []object.Object{inst}, nil) //nolint
			}
			return inst, nil
		},
	})

	// __exit__() — disable
	cls.Dict.SetStr("__exit__", &object.BuiltinFunc{
		Name: "__exit__",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			disableFn, _ := cls.Dict.GetStr("disable")
			if df, ok := disableFn.(*object.BuiltinFunc); ok {
				df.Call(ii, []object.Object{inst}, nil) //nolint
			}
			return object.BoolOf(false), nil
		},
	})

	return cls
}

// ── profile module ────────────────────────────────────────────────────────────

func (i *Interp) buildProfile() *object.Module {
	m := &object.Module{Name: "profile", Dict: object.NewDict()}
	profileCls := buildProfileClass(i, "Profile")
	m.Dict.SetStr("Profile", profileCls)

	addModuleFuncs(i, m, profileCls)
	return m
}

// ── cProfile module ───────────────────────────────────────────────────────────

func (i *Interp) buildCProfile() *object.Module {
	m := &object.Module{Name: "cProfile", Dict: object.NewDict()}
	profileCls := buildProfileClass(i, "Profile")
	m.Dict.SetStr("Profile", profileCls)

	addModuleFuncs(i, m, profileCls)
	return m
}

// addModuleFuncs adds run/runctx/runcall at module level.
func addModuleFuncs(i *Interp, m *object.Module, profileCls *object.Class) {
	// run(command, filename=None, sort=-1) — stub
	m.Dict.SetStr("run", &object.BuiltinFunc{
		Name: "run",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// runctx(command, globals, locals, filename=None, sort=-1) — stub
	m.Dict.SetStr("runctx", &object.BuiltinFunc{
		Name: "runctx",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// runcall(func, *args, **kwds)
	m.Dict.SetStr("runcall", &object.BuiltinFunc{
		Name: "runcall",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "runcall() requires function argument")
			}
			interp := ii.(*Interp)
			inst := &object.Instance{Class: profileCls, Dict: object.NewDict()}
			// init
			if initFn, ok := profileCls.Dict.GetStr("__init__"); ok {
				if bf, ok2 := initFn.(*object.BuiltinFunc); ok2 {
					bf.Call(ii, []object.Object{inst}, nil) //nolint
				}
			}
			// runcall with inst as self
			runcallFn, _ := profileCls.Dict.GetStr("runcall")
			if rf, ok := runcallFn.(*object.BuiltinFunc); ok {
				return rf.Call(ii, append([]object.Object{inst}, a...), kw)
			}
			// fallback: direct call
			return interp.callObject(a[0], a[1:], kw)
		},
	})
}

// ── pstats module ─────────────────────────────────────────────────────────────

func (i *Interp) buildPstats() *object.Module {
	m := &object.Module{Name: "pstats", Dict: object.NewDict()}

	// ── SortKey ───────────────────────────────────────────────────────────
	sortKeyCls := &object.Class{Name: "SortKey", Dict: object.NewDict()}
	for _, pair := range [][2]string{
		{"CALLS", "calls"}, {"CUMULATIVE", "cumulative"}, {"FILENAME", "filename"},
		{"LINE", "line"}, {"NAME", "name"}, {"NFL", "nfl"},
		{"PCALLS", "pcalls"}, {"STDNAME", "stdname"}, {"TIME", "time"},
	} {
		val := &object.Str{V: pair[1]}
		sortKeyCls.Dict.SetStr(pair[0], val)
		m.Dict.SetStr("SortKey", sortKeyCls)
	}
	m.Dict.SetStr("SortKey", sortKeyCls)

	// ── Stats class ───────────────────────────────────────────────────────
	statsCls := &object.Class{Name: "Stats", Dict: object.NewDict()}

	// Stats.__init__(self, *filenames_or_profile, stream=sys.stdout)
	statsCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "Stats() requires at least self")
			}
			inst := a[0].(*object.Instance)

			// stream kwarg
			var streamObj object.Object = &object.TextStream{Name: "stdout", W: ii.(*Interp).Stdout}
			if kw != nil {
				if sv, ok := kw.GetStr("stream"); ok {
					streamObj = sv
				}
			}
			inst.Dict.SetStr("_stream", streamObj)

			// Collect entries from Profile instances or filenames.
			var allEntries []profileEntry
			totalTT := 0.0

			for _, src := range a[1:] {
				switch v := src.(type) {
				case *object.Instance:
					// Profile object
					entries := profileGetEntries(v)
					allEntries = append(allEntries, entries...)
					if ttObj, ok2 := v.Dict.GetStr("_total_tt"); ok2 {
						if ttF, ok3 := ttObj.(*object.Float); ok3 {
							totalTT += ttF.V
						}
					}
				case *object.Str:
					// filename — load from JSON file
					b, err2 := os.ReadFile(v.V)
					if err2 != nil {
						return nil, object.Errorf(i.osErr, "Stats: cannot read %s: %v", v.V, err2)
					}
					var data map[string]any
					if err2 = json.Unmarshal(b, &data); err2 != nil {
						return nil, object.Errorf(i.valueErr, "Stats: invalid profile file %s", v.V)
					}
					if enc, ok2 := data["entries"]; ok2 {
						if re, err3 := json.Marshal(enc); err3 == nil {
							var entries []profileEntry
							json.Unmarshal(re, &entries) //nolint
							allEntries = append(allEntries, entries...)
						}
					}
					if tt, ok2 := data["total_tt"].(float64); ok2 {
						totalTT += tt
					}
				}
			}

			profileSetEntries(inst, allEntries)
			inst.Dict.SetStr("_total_tt", &object.Float{V: totalTT})
			inst.Dict.SetStr("_sort_key", &object.Str{V: "stdname"})
			inst.Dict.SetStr("_reversed", object.BoolOf(false))
			return object.None, nil
		},
	})

	// add(*filenames_or_profiles) — merge more stats into self; returns self
	statsCls.Dict.SetStr("add", &object.BuiltinFunc{
		Name: "add",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "add() requires self")
			}
			inst := a[0].(*object.Instance)
			existing := profileGetEntries(inst)
			totalTT := 0.0
			if ttObj, ok := inst.Dict.GetStr("_total_tt"); ok {
				if ttF, ok2 := ttObj.(*object.Float); ok2 {
					totalTT = ttF.V
				}
			}
			for _, src := range a[1:] {
				switch v := src.(type) {
				case *object.Instance:
					existing = append(existing, profileGetEntries(v)...)
					if ttObj, ok2 := v.Dict.GetStr("_total_tt"); ok2 {
						if ttF, ok3 := ttObj.(*object.Float); ok3 {
							totalTT += ttF.V
						}
					}
				}
			}
			profileSetEntries(inst, existing)
			inst.Dict.SetStr("_total_tt", &object.Float{V: totalTT})
			return inst, nil
		},
	})

	// strip_dirs() — remove directory paths from filenames; returns self
	statsCls.Dict.SetStr("strip_dirs", &object.BuiltinFunc{
		Name: "strip_dirs",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			entries := profileGetEntries(inst)
			for idx := range entries {
				parts := strings.Split(entries[idx].Filename, "/")
				entries[idx].Filename = parts[len(parts)-1]
			}
			profileSetEntries(inst, entries)
			return inst, nil
		},
	})

	// sort_stats(*keys) — sort entries by key; returns self
	statsCls.Dict.SetStr("sort_stats", &object.BuiltinFunc{
		Name: "sort_stats",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			key := "stdname"
			if len(a) >= 2 {
				switch k := a[1].(type) {
				case *object.Str:
					key = k.V
				case *object.Int:
					switch k.Int64() {
					case 0:
						key = "calls"
					case 1:
						key = "time"
					case 2:
						key = "cumulative"
					default:
						key = "stdname"
					}
				}
			}
			inst.Dict.SetStr("_sort_key", &object.Str{V: key})
			// Apply sort
			entries := profileGetEntries(inst)
			sortProfileEntries(entries, key)
			profileSetEntries(inst, entries)
			return inst, nil
		},
	})

	// reverse_order() — reverse current sort order; returns self
	statsCls.Dict.SetStr("reverse_order", &object.BuiltinFunc{
		Name: "reverse_order",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			entries := profileGetEntries(inst)
			for left, right := 0, len(entries)-1; left < right; left, right = left+1, right-1 {
				entries[left], entries[right] = entries[right], entries[left]
			}
			profileSetEntries(inst, entries)
			return inst, nil
		},
	})

	// print_stats(*restrictions) — print table to stream
	statsCls.Dict.SetStr("print_stats", &object.BuiltinFunc{
		Name: "print_stats",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			entries := profileGetEntries(inst)
			totalTT := 0.0
			if ttObj, ok := inst.Dict.GetStr("_total_tt"); ok {
				if ttF, ok2 := ttObj.(*object.Float); ok2 {
					totalTT = ttF.V
				}
			}
			streamObj, _ := inst.Dict.GetStr("_stream")
			w := profileResolveWriter(ii, streamObj)
			profilePrintTable(w, entries, totalTT)
			return inst, nil
		},
	})

	// print_callers(*restrictions) — stub; writes header only
	statsCls.Dict.SetStr("print_callers", &object.BuiltinFunc{
		Name: "print_callers",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			streamObj, _ := inst.Dict.GetStr("_stream")
			w := profileResolveWriter(ii, streamObj)
			fmt.Fprintf(w, "   Function                                          was called by...\n")
			return inst, nil
		},
	})

	// print_callees(*restrictions) — stub; writes header only
	statsCls.Dict.SetStr("print_callees", &object.BuiltinFunc{
		Name: "print_callees",
		Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			streamObj, _ := inst.Dict.GetStr("_stream")
			w := profileResolveWriter(ii, streamObj)
			fmt.Fprintf(w, "   Function                                          called...\n")
			return inst, nil
		},
	})

	// dump_stats(filename) — write stats to file
	statsCls.Dict.SetStr("dump_stats", &object.BuiltinFunc{
		Name: "dump_stats",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "dump_stats() requires filename argument")
			}
			inst := a[0].(*object.Instance)
			filename := object.Str_(a[1])
			entries := profileGetEntries(inst)
			totalTT := 0.0
			if ttObj, ok := inst.Dict.GetStr("_total_tt"); ok {
				if ttF, ok2 := ttObj.(*object.Float); ok2 {
					totalTT = ttF.V
				}
			}
			data := map[string]any{"entries": entries, "total_tt": totalTT}
			b, _ := json.Marshal(data)
			if err := os.WriteFile(filename, b, 0o644); err != nil {
				return nil, object.Errorf(i.osErr, "dump_stats: %v", err)
			}
			return object.None, nil
		},
	})

	// get_stats_profile() — returns a simple object with total_tt, total_calls, etc.
	statsCls.Dict.SetStr("get_stats_profile", &object.BuiltinFunc{
		Name: "get_stats_profile",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			entries := profileGetEntries(inst)
			totalCalls := 0
			for _, e := range entries {
				totalCalls += e.NCalls
			}
			totalTT := 0.0
			if ttObj, ok := inst.Dict.GetStr("_total_tt"); ok {
				if ttF, ok2 := ttObj.(*object.Float); ok2 {
					totalTT = ttF.V
				}
			}
			result := &object.Instance{Class: statsProfileCls, Dict: object.NewDict()}
			result.Dict.SetStr("total_tt", &object.Float{V: totalTT})
			result.Dict.SetStr("total_calls", object.NewInt(int64(totalCalls)))
			result.Dict.SetStr("prim_calls", object.NewInt(int64(totalCalls)))
			return result, nil
		},
	})

	m.Dict.SetStr("Stats", statsCls)
	return m
}

// statsProfileCls is a lightweight result class for get_stats_profile().
var statsProfileCls = &object.Class{Name: "StatsProfile", Dict: object.NewDict()}

// sortProfileEntries sorts entries in-place by key.
func sortProfileEntries(entries []profileEntry, key string) {
	sort.SliceStable(entries, func(a, b int) bool {
		ea, eb := entries[a], entries[b]
		switch key {
		case "calls", "ncalls":
			return ea.NCalls > eb.NCalls
		case "cumulative", "cumtime":
			return ea.Cumtime > eb.Cumtime
		case "time", "tottime":
			return ea.Tottime > eb.Tottime
		case "filename", "file", "module":
			return ea.Filename < eb.Filename
		case "name":
			return ea.Funcname < eb.Funcname
		case "line":
			return ea.Lineno < eb.Lineno
		default: // stdname, nfl
			ka := fmt.Sprintf("%s:%d(%s)", ea.Filename, ea.Lineno, ea.Funcname)
			kb := fmt.Sprintf("%s:%d(%s)", eb.Filename, eb.Lineno, eb.Funcname)
			return ka < kb
		}
	})
}

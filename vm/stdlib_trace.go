package vm

import (
	"encoding/json"
	"os"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildTrace() *object.Module {
	m := &object.Module{Name: "trace", Dict: object.NewDict()}

	// ── CoverageResults class ─────────────────────────────────────────────────

	coverCls := &object.Class{Name: "CoverageResults", Dict: object.NewDict()}

	// helpers: (filename, module, funcname) → 1 stored as JSON map[string]int
	type calledKey struct {
		F string `json:"f"`
		M string `json:"m"`
		N string `json:"n"`
	}
	type coverData struct {
		CalledFuncs []calledKey `json:"cf"`
	}

	loadCoverData := func(inst *object.Instance) coverData {
		v, ok := inst.Dict.GetStr("_data_json")
		if !ok {
			return coverData{}
		}
		s, ok := v.(*object.Str)
		if !ok {
			return coverData{}
		}
		var d coverData
		json.Unmarshal([]byte(s.V), &d) //nolint
		return d
	}

	saveCoverData := func(inst *object.Instance, d coverData) {
		b, _ := json.Marshal(d)
		inst.Dict.SetStr("_data_json", &object.Str{V: string(b)})
	}

	// Build a Python dict from calledfuncs: keys are 3-tuples (file, mod, name)
	buildCalledFuncsDict := func(d coverData) *object.Dict {
		dict := object.NewDict()
		for _, k := range d.CalledFuncs {
			key := &object.Tuple{V: []object.Object{
				&object.Str{V: k.F},
				&object.Str{V: k.M},
				&object.Str{V: k.N},
			}}
			dict.Set(key, object.NewInt(1))
		}
		return dict
	}

	coverCls.Dict.SetStr("update", &object.BuiltinFunc{
		Name: "update",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			other, ok := a[1].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			selfData := loadCoverData(self)
			otherData := loadCoverData(other)
			// merge calledfuncs (deduplicate)
			seen := map[calledKey]bool{}
			for _, k := range selfData.CalledFuncs {
				seen[k] = true
			}
			for _, k := range otherData.CalledFuncs {
				if !seen[k] {
					selfData.CalledFuncs = append(selfData.CalledFuncs, k)
					seen[k] = true
				}
			}
			saveCoverData(self, selfData)
			return object.None, nil
		},
	})

	coverCls.Dict.SetStr("write_results", &object.BuiltinFunc{
		Name: "write_results",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// __getattr__ to expose counts/calledfuncs/callers as Python dicts
	coverCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{
		Name: "__getattr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			attr, ok := a[1].(*object.Str)
			if !ok {
				return object.None, nil
			}
			switch attr.V {
			case "counts":
				return object.NewDict(), nil
			case "callers":
				return object.NewDict(), nil
			case "calledfuncs":
				d := loadCoverData(inst)
				return buildCalledFuncsDict(d), nil
			}
			return nil, object.Errorf(i.attrErr, "'CoverageResults' object has no attribute '%s'", attr.V)
		},
	})

	m.Dict.SetStr("CoverageResults", coverCls)

	// ── Trace class ───────────────────────────────────────────────────────────

	traceCls := &object.Class{Name: "Trace", Dict: object.NewDict()}

	traceCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			// defaults
			inst.Dict.SetStr("count", object.NewInt(1))
			inst.Dict.SetStr("trace", object.NewInt(1))
			inst.Dict.SetStr("countfuncs", object.NewInt(0))
			inst.Dict.SetStr("countcallers", object.NewInt(0))
			inst.Dict.SetStr("timing", &object.Bool{V: false})
			inst.Dict.SetStr("infile", object.None)
			inst.Dict.SetStr("outfile", object.None)
			// positional
			positional := []string{"count", "trace", "countfuncs", "countcallers",
				"ignoremods", "ignoredirs", "infile", "outfile", "timing"}
			for idx, name := range positional {
				if len(a) >= idx+2 {
					inst.Dict.SetStr(name, a[idx+1])
				}
			}
			// keyword
			if kw != nil {
				ks, vs := kw.Items()
				for idx, k := range ks {
					if s, ok := k.(*object.Str); ok {
						inst.Dict.SetStr(s.V, vs[idx])
					}
				}
			}
			// _data holds accumulated CoverageResults data as JSON
			inst.Dict.SetStr("_data_json", &object.Str{V: `{"cf":[]}`})
			// load infile if given
			if infileObj, ok := inst.Dict.GetStr("infile"); ok && infileObj != object.None {
				if s, ok2 := infileObj.(*object.Str); ok2 {
					if raw, err := os.ReadFile(s.V); err == nil {
						inst.Dict.SetStr("_data_json", &object.Str{V: string(raw)})
					}
				}
			}
			return object.None, nil
		},
	})

	traceCls.Dict.SetStr("run", &object.BuiltinFunc{
		Name: "run",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	traceCls.Dict.SetStr("runctx", &object.BuiltinFunc{
		Name: "runctx",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	traceCls.Dict.SetStr("runfunc", &object.BuiltinFunc{
		Name: "runfunc",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			inst := a[0].(*object.Instance)
			fn := a[1]
			fnArgs := a[2:]
			interp := ii.(*Interp)

			// record call if countfuncs is set
			countfuncsObj, _ := inst.Dict.GetStr("countfuncs")
			doCount := false
			if n, ok := toInt64(countfuncsObj); ok && n != 0 {
				doCount = true
			} else if b, ok := countfuncsObj.(*object.Bool); ok && b.V {
				doCount = true
			}
			if doCount {
				filename := "<unknown>"
				module := "<unknown>"
				funcname := "<unknown>"
				switch f := fn.(type) {
				case *object.Function:
					filename = f.Code.Filename
					funcname = f.Code.Name
					module = funcname
				case *object.BuiltinFunc:
					funcname = f.Name
					module = f.Name
				}
				type calledKey struct {
					F string `json:"f"`
					M string `json:"m"`
					N string `json:"n"`
				}
				type coverData struct {
					CalledFuncs []calledKey `json:"cf"`
				}
				// load existing
				var d coverData
				if v, ok := inst.Dict.GetStr("_data_json"); ok {
					if s, ok2 := v.(*object.Str); ok2 {
						json.Unmarshal([]byte(s.V), &d) //nolint
					}
				}
				// append if not already present
				newKey := calledKey{F: filename, M: module, N: funcname}
				found := false
				for _, k := range d.CalledFuncs {
					if k == newKey {
						found = true
						break
					}
				}
				if !found {
					d.CalledFuncs = append(d.CalledFuncs, newKey)
				}
				b, _ := json.Marshal(d)
				inst.Dict.SetStr("_data_json", &object.Str{V: string(b)})
			}

			return interp.callObject(fn, fnArgs, kw)
		},
	})

	traceCls.Dict.SetStr("results", &object.BuiltinFunc{
		Name: "results",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := a[0].(*object.Instance)
			// build CoverageResults instance
			result := &object.Instance{Class: coverCls, Dict: object.NewDict()}
			// copy _data_json
			if v, ok := inst.Dict.GetStr("_data_json"); ok {
				result.Dict.SetStr("_data_json", v)
			}
			// write outfile if set
			if outfileObj, ok := inst.Dict.GetStr("outfile"); ok && outfileObj != object.None {
				if s, ok2 := outfileObj.(*object.Str); ok2 {
					if v, ok3 := inst.Dict.GetStr("_data_json"); ok3 {
						if js, ok4 := v.(*object.Str); ok4 {
							os.WriteFile(s.V, []byte(js.V), 0o644) //nolint
						}
					}
				}
			}
			return result, nil
		},
	})

	m.Dict.SetStr("Trace", traceCls)

	return m
}

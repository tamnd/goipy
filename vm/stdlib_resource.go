package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildResource() *object.Module {
	m := &object.Module{Name: "resource", Dict: object.NewDict()}
	d := m.Dict

	d.SetStr("error", i.osErr)

	const RLIM_INFINITY = int64(^uint64(0) >> 1)

	consts := map[string]int64{
		"RLIM_INFINITY":    RLIM_INFINITY,
		"RLIMIT_CPU":       0,
		"RLIMIT_FSIZE":     1,
		"RLIMIT_DATA":      2,
		"RLIMIT_STACK":     3,
		"RLIMIT_CORE":      4,
		"RLIMIT_RSS":       5,
		"RLIMIT_MEMLOCK":   6,
		"RLIMIT_NPROC":     7,
		"RLIMIT_NOFILE":    8,
		"RLIMIT_AS":        5,
		"RUSAGE_SELF":      0,
		"RUSAGE_CHILDREN":  -1,
	}
	for name, val := range consts {
		d.SetStr(name, intObj(val))
	}

	ruCls := &object.Class{
		Name:  "struct_rusage",
		Bases: []*object.Class{},
		Dict:  object.NewDict(),
	}
	ruFields := []string{
		"ru_utime", "ru_stime", "ru_maxrss", "ru_ixrss", "ru_idrss",
		"ru_isrss", "ru_minflt", "ru_majflt", "ru_nswap", "ru_inblock",
		"ru_oublock", "ru_msgsnd", "ru_msgrcv", "ru_nsignals",
		"ru_nvcsw", "ru_nivcsw",
	}
	ruCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			for idx, f := range ruFields {
				if idx+1 < len(a) {
					inst.Dict.SetStr(f, a[idx+1])
				} else if f == "ru_utime" || f == "ru_stime" {
					inst.Dict.SetStr(f, &object.Float{V: 0.0})
				} else {
					inst.Dict.SetStr(f, intObj(0))
				}
			}
			return object.None, nil
		},
	})
	d.SetStr("struct_rusage", ruCls)

	mkRusage := func() *object.Instance {
		inst := &object.Instance{Class: ruCls, Dict: object.NewDict()}
		for _, f := range ruFields {
			// ru_utime and ru_stime are float in CPython; all others are int
			if f == "ru_utime" || f == "ru_stime" {
				inst.Dict.SetStr(f, &object.Float{V: 0.0})
			} else {
				inst.Dict.SetStr(f, intObj(0))
			}
		}
		return inst
	}

	d.SetStr("getrusage", &object.BuiltinFunc{
		Name: "getrusage",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return mkRusage(), nil
		},
	})

	d.SetStr("getrlimit", &object.BuiltinFunc{
		Name: "getrlimit",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{intObj(0), intObj(RLIM_INFINITY)}}, nil
		},
	})

	d.SetStr("setrlimit", &object.BuiltinFunc{
		Name: "setrlimit",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	d.SetStr("getpagesize", &object.BuiltinFunc{
		Name: "getpagesize",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj(4096), nil
		},
	})

	return m
}

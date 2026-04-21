package vm

import (
	"time"

	"github.com/tamnd/goipy/object"
)

// buildTime exposes a small slice of Python's `time` module. `perf_counter`
// is the workhorse for benchmarks — returns a monotonic float in seconds.
// `time()` returns Unix wall time. `sleep()` blocks the goroutine.
func (i *Interp) buildTime() *object.Module {
	m := &object.Module{Name: "time", Dict: object.NewDict()}

	origin := time.Now()

	perf := &object.BuiltinFunc{Name: "perf_counter", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Float{V: time.Since(origin).Seconds()}, nil
	}}
	m.Dict.SetStr("perf_counter", perf)
	m.Dict.SetStr("perf_counter_ns", &object.BuiltinFunc{Name: "perf_counter_ns", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(time.Since(origin).Nanoseconds()), nil
	}})
	m.Dict.SetStr("monotonic", perf)
	m.Dict.SetStr("monotonic_ns", &object.BuiltinFunc{Name: "monotonic_ns", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(time.Since(origin).Nanoseconds()), nil
	}})

	m.Dict.SetStr("time", &object.BuiltinFunc{Name: "time", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Float{V: float64(time.Now().UnixNano()) / 1e9}, nil
	}})
	m.Dict.SetStr("time_ns", &object.BuiltinFunc{Name: "time_ns", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(time.Now().UnixNano()), nil
	}})

	m.Dict.SetStr("sleep", &object.BuiltinFunc{Name: "sleep", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) != 1 {
			return nil, object.Errorf(i.typeErr, "sleep() takes exactly one argument")
		}
		secs, ok := toFloat64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "sleep() argument must be numeric")
		}
		if secs > 0 {
			time.Sleep(time.Duration(secs * float64(time.Second)))
		}
		return object.None, nil
	}})

	return m
}

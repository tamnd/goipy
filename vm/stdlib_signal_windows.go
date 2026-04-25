//go:build windows

package vm

import (
	ossignal "os/signal"
	"syscall"

	"github.com/tamnd/goipy/object"
)

const sigDFL = int64(0)
const sigIGN = int64(1)

// sigCtx satisfies any platform files that reference it; on Windows it is empty.
type sigCtx struct{}

func (i *Interp) buildSignal() *object.Module {
	m := &object.Module{Name: "signal", Dict: object.NewDict()}

	setInt := func(name string, val int) {
		m.Dict.SetStr(name, object.NewInt(int64(val)))
	}

	m.Dict.SetStr("SIG_DFL", object.NewInt(sigDFL))
	m.Dict.SetStr("SIG_IGN", object.NewInt(sigIGN))

	setInt("SIGINT", int(syscall.SIGINT))
	setInt("SIGTERM", int(syscall.SIGTERM))
	setInt("SIGABRT", int(syscall.SIGABRT))

	m.Dict.SetStr("signal", &object.BuiltinFunc{Name: "signal", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	m.Dict.SetStr("getsignal", &object.BuiltinFunc{Name: "getsignal", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(sigDFL), nil
	}})

	m.Dict.SetStr("raise_signal", &object.BuiltinFunc{Name: "raise_signal", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		n, ok := toInt64(a[0])
		if !ok {
			return object.None, nil
		}
		ossignal.Reset(syscall.Signal(n))
		return object.None, nil
	}})

	m.Dict.SetStr("set_wakeup_fd", &object.BuiltinFunc{Name: "set_wakeup_fd", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(-1), nil
	}})

	m.Dict.SetStr("pause", &object.BuiltinFunc{Name: "pause", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return nil, object.Errorf(i.osErr, "signal.pause: not supported on Windows")
	}})

	return m
}

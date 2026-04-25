//go:build darwin

package vm

import (
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) extendSignalModule(m *object.Module, _ sigCtx) {
	setInt := func(name string, val int) {
		m.Dict.SetStr(name, object.NewInt(int64(val)))
	}

	// macOS/BSD-specific signal constants
	setInt("SIGEMT", 7)
	setInt("SIGINFO", 29)
	setInt("SIGIO", 23)

	// ── alarm(seconds) — simulated via goroutine ──────────────────────────────
	// Go's unix package does not expose Alarm on darwin; emulate via time.AfterFunc
	// sending SIGALRM to the process. Does NOT track remaining time accurately.
	var alarmMu sync.Mutex
	var alarmTimer *time.Timer

	m.Dict.SetStr("alarm", &object.BuiltinFunc{Name: "alarm",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			secs := uint(0)
			if len(a) > 0 {
				if n, ok := toInt64(a[0]); ok {
					secs = uint(n)
				}
			}
			alarmMu.Lock()
			if alarmTimer != nil {
				alarmTimer.Stop()
				alarmTimer = nil
			}
			if secs > 0 {
				alarmTimer = time.AfterFunc(time.Duration(secs)*time.Second, func() {
					syscall.Kill(os.Getpid(), syscall.SIGALRM) //nolint
				})
			}
			alarmMu.Unlock()
			return object.NewInt(0), nil // remaining not tracked on darwin
		}})

	// ── getitimer/setitimer — not available on darwin without CGo ─────────────
	stubTuple := &object.Tuple{V: []object.Object{&object.Float{V: 0}, &object.Float{V: 0}}}
	m.Dict.SetStr("getitimer", &object.BuiltinFunc{Name: "getitimer",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return stubTuple, nil
		}})
	m.Dict.SetStr("setitimer", &object.BuiltinFunc{Name: "setitimer",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return stubTuple, nil
		}})

	// ── pthread_sigmask — returns empty frozenset on darwin ───────────────────
	m.Dict.SetStr("pthread_sigmask", &object.BuiltinFunc{Name: "pthread_sigmask",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewFrozenset(), nil
		}})

	// ── sigpending — returns empty frozenset on darwin ────────────────────────
	m.Dict.SetStr("sigpending", &object.BuiltinFunc{Name: "sigpending",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewFrozenset(), nil
		}})

	// ── sigwait — not available without CGo on darwin ─────────────────────────
	m.Dict.SetStr("sigwait", &object.BuiltinFunc{Name: "sigwait",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.runtimeErr, "sigwait not available on this platform")
		}})
}

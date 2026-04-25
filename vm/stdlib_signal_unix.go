//go:build unix

package vm

import (
	"fmt"
	"os"
	ossignal "os/signal"
	"sync"
	"syscall"

	"github.com/tamnd/goipy/object"
	"golang.org/x/sys/unix"
)

const sigDFL = int64(0)
const sigIGN = int64(1)

// signalState holds the process-wide Python signal handler registry.
// Signals are process-wide so this is shared across all Interp instances.
type signalState struct {
	mu       sync.Mutex
	handlers map[int]object.Object  // signum → Python handler
	channels map[int]chan os.Signal  // signum → active Notify channel
	wakeupFd int                    // set_wakeup_fd target, -1 if none
}

var globalSigState = &signalState{
	handlers: make(map[int]object.Object),
	channels: make(map[int]chan os.Signal),
	wakeupFd: -1,
}

// sigCtx bundles helpers for extendSignalModule platform files.
type sigCtx struct {
	errCls       *object.Class
	itimerErrCls *object.Class
}

func (i *Interp) buildSignal() *object.Module {
	m := &object.Module{Name: "signal", Dict: object.NewDict()}
	errCls := i.osErr

	setInt := func(name string, val int) {
		m.Dict.SetStr(name, object.NewInt(int64(val)))
	}

	// ── Special handler sentinels ─────────────────────────────────────────────
	m.Dict.SetStr("SIG_DFL", object.NewInt(sigDFL))
	m.Dict.SetStr("SIG_IGN", object.NewInt(sigIGN))

	// ── Sigmask how constants ─────────────────────────────────────────────────
	setInt("SIG_BLOCK", 1)
	setInt("SIG_UNBLOCK", 2)
	setInt("SIG_SETMASK", 3)

	// ── Interval timer constants ──────────────────────────────────────────────
	setInt("ITIMER_REAL", 0)
	setInt("ITIMER_VIRTUAL", 1)
	setInt("ITIMER_PROF", 2)

	// ── Signal constants (common across all Unix) ─────────────────────────────
	setInt("SIGHUP", int(syscall.SIGHUP))
	setInt("SIGINT", int(syscall.SIGINT))
	setInt("SIGQUIT", int(syscall.SIGQUIT))
	setInt("SIGILL", int(syscall.SIGILL))
	setInt("SIGTRAP", int(syscall.SIGTRAP))
	setInt("SIGABRT", int(syscall.SIGABRT))
	setInt("SIGIOT", int(syscall.SIGABRT)) // SIGIOT == SIGABRT on most Unix
	setInt("SIGFPE", int(syscall.SIGFPE))
	setInt("SIGKILL", int(syscall.SIGKILL))
	setInt("SIGSEGV", int(syscall.SIGSEGV))
	setInt("SIGPIPE", int(syscall.SIGPIPE))
	setInt("SIGALRM", int(syscall.SIGALRM))
	setInt("SIGTERM", int(syscall.SIGTERM))
	setInt("SIGURG", int(syscall.SIGURG))
	setInt("SIGSTOP", int(syscall.SIGSTOP))
	setInt("SIGTSTP", int(syscall.SIGTSTP))
	setInt("SIGCONT", int(syscall.SIGCONT))
	setInt("SIGCHLD", int(syscall.SIGCHLD))
	setInt("SIGTTIN", int(syscall.SIGTTIN))
	setInt("SIGTTOU", int(syscall.SIGTTOU))
	setInt("SIGXCPU", int(syscall.SIGXCPU))
	setInt("SIGXFSZ", int(syscall.SIGXFSZ))
	setInt("SIGVTALRM", int(syscall.SIGVTALRM))
	setInt("SIGPROF", int(syscall.SIGPROF))
	setInt("SIGWINCH", int(syscall.SIGWINCH))
	setInt("SIGUSR1", int(syscall.SIGUSR1))
	setInt("SIGUSR2", int(syscall.SIGUSR2))
	setInt("SIGBUS", int(unix.SIGBUS))

	// ── ItimerError ───────────────────────────────────────────────────────────
	itimerErrCls := &object.Class{Name: "ItimerError", Dict: object.NewDict()}
	m.Dict.SetStr("ItimerError", itimerErrCls)

	// ── default_int_handler ───────────────────────────────────────────────────
	defaultIntHandler := &object.BuiltinFunc{Name: "default_int_handler",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.keyboardInterrupt, "Keyboard interrupt")
		}}
	m.Dict.SetStr("default_int_handler", defaultIntHandler)

	// Initialize SIGINT handler to default_int_handler (like CPython).
	globalSigState.mu.Lock()
	if _, exists := globalSigState.handlers[int(syscall.SIGINT)]; !exists {
		globalSigState.handlers[int(syscall.SIGINT)] = defaultIntHandler
	}
	globalSigState.mu.Unlock()

	// ── signal(signum, handler) ───────────────────────────────────────────────
	m.Dict.SetStr("signal", &object.BuiltinFunc{Name: "signal",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "signal() requires signum and handler")
			}
			signum, ok := sigExtract(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "signal() invalid signum")
			}
			handler := a[1]

			globalSigState.mu.Lock()
			old, exists := globalSigState.handlers[signum]
			if !exists {
				old = object.NewInt(sigDFL)
			}

			// Stop any existing Notify channel for this signal.
			if ch, ok2 := globalSigState.channels[signum]; ok2 {
				ossignal.Stop(ch)
				close(ch)
				delete(globalSigState.channels, signum)
			}

			if n, isInt := toInt64(handler); isInt && (n == sigDFL || n == sigIGN) {
				globalSigState.handlers[signum] = handler
				if n == sigIGN {
					ossignal.Ignore(syscall.Signal(signum))
				} else {
					ossignal.Reset(syscall.Signal(signum))
				}
				globalSigState.mu.Unlock()
			} else {
				globalSigState.handlers[signum] = handler
				ch := make(chan os.Signal, 16)
				globalSigState.channels[signum] = ch
				globalSigState.mu.Unlock()

				ossignal.Notify(ch, syscall.Signal(signum))
				go func(sig int, ch chan os.Signal) {
					for range ch {
						globalSigState.mu.Lock()
						h := globalSigState.handlers[sig]
						wfd := globalSigState.wakeupFd
						globalSigState.mu.Unlock()
						if h != nil {
							if _, isInt2 := toInt64(h); !isInt2 {
								i.callObject(h, []object.Object{object.NewInt(int64(sig)), object.None}, nil) //nolint
							}
						}
						if wfd >= 0 {
							unix.Write(wfd, []byte{byte(sig)}) //nolint
						}
					}
				}(signum, ch)
			}
			return old, nil
		}})

	// ── getsignal(signum) ─────────────────────────────────────────────────────
	m.Dict.SetStr("getsignal", &object.BuiltinFunc{Name: "getsignal",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "getsignal() requires signum")
			}
			signum, ok := sigExtract(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "getsignal() invalid signum")
			}
			globalSigState.mu.Lock()
			h, exists := globalSigState.handlers[signum]
			globalSigState.mu.Unlock()
			if !exists {
				return object.NewInt(sigDFL), nil
			}
			return h, nil
		}})

	// ── raise_signal(signum) ──────────────────────────────────────────────────
	m.Dict.SetStr("raise_signal", &object.BuiltinFunc{Name: "raise_signal",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "raise_signal() requires signum")
			}
			signum, ok := sigExtract(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "raise_signal() invalid signum")
			}

			globalSigState.mu.Lock()
			h := globalSigState.handlers[signum]
			globalSigState.mu.Unlock()

			if h != nil {
				if n, isInt := toInt64(h); isInt {
					if n == sigIGN {
						return object.None, nil
					}
					// SIG_DFL: fall through to OS kill.
				} else {
					// Python callable: call synchronously.
					_, err := i.callObject(h, []object.Object{object.NewInt(int64(signum)), object.None}, nil)
					return object.None, err
				}
			}
			// Default/unregistered: send actual OS signal.
			if err := unix.Kill(os.Getpid(), syscall.Signal(signum)); err != nil {
				return nil, object.Errorf(errCls, "raise_signal: %v", err)
			}
			return object.None, nil
		}})

	// ── strsignal(signum) ─────────────────────────────────────────────────────
	m.Dict.SetStr("strsignal", &object.BuiltinFunc{Name: "strsignal",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "strsignal() requires signum")
			}
			n, ok := toInt64(a[0])
			if !ok {
				return object.None, nil
			}
			sig := syscall.Signal(n)
			desc := sig.String()
			// Go returns "signal N" for unknown signals.
			if desc == fmt.Sprintf("signal %d", n) {
				return object.None, nil
			}
			return &object.Str{V: desc}, nil
		}})

	// ── valid_signals() ───────────────────────────────────────────────────────
	m.Dict.SetStr("valid_signals", &object.BuiltinFunc{Name: "valid_signals",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			fs := object.NewFrozenset()
			for n := 1; n < 64; n++ {
				sig := syscall.Signal(n)
				desc := sig.String()
				if desc != fmt.Sprintf("signal %d", n) {
					fs.Add(object.NewInt(int64(n))) //nolint
				}
			}
			return fs, nil
		}})

	// ── set_wakeup_fd(fd, *, warn_on_full_buffer=True) ────────────────────────
	m.Dict.SetStr("set_wakeup_fd", &object.BuiltinFunc{Name: "set_wakeup_fd",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "set_wakeup_fd() requires fd")
			}
			fd, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "set_wakeup_fd() fd must be int")
			}
			globalSigState.mu.Lock()
			old := globalSigState.wakeupFd
			globalSigState.wakeupFd = int(fd)
			globalSigState.mu.Unlock()
			return object.NewInt(int64(old)), nil
		}})

	// ── pause() ───────────────────────────────────────────────────────────────
	m.Dict.SetStr("pause", &object.BuiltinFunc{Name: "pause",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ch := make(chan os.Signal, 1)
			ossignal.Notify(ch)
			defer ossignal.Stop(ch)
			<-ch
			return object.None, nil
		}})

	// ── siginterrupt(signum, flag) ────────────────────────────────────────────
	m.Dict.SetStr("siginterrupt", &object.BuiltinFunc{Name: "siginterrupt",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "siginterrupt() requires signum and flag")
			}
			return object.None, nil // SA_RESTART is not directly settable from Go
		}})

	// ── pthread_kill(thread_id, signum) ───────────────────────────────────────
	m.Dict.SetStr("pthread_kill", &object.BuiltinFunc{Name: "pthread_kill",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "pthread_kill() requires thread_id and signum")
			}
			signum, ok := sigExtract(a[1])
			if !ok {
				return nil, object.Errorf(i.typeErr, "pthread_kill() invalid signum")
			}
			// In goipy, goroutines don't have pthread_t IDs accessible from Python.
			// Deliver to the whole process instead.
			if err := unix.Kill(os.Getpid(), syscall.Signal(signum)); err != nil {
				return nil, object.Errorf(errCls, "pthread_kill: %v", err)
			}
			return object.None, nil
		}})

	ctx := sigCtx{errCls: errCls, itimerErrCls: itimerErrCls}
	i.extendSignalModule(m, ctx)

	return m
}

// sigExtract converts a Python object to a signal number.
func sigExtract(obj object.Object) (int, bool) {
	n, ok := toInt64(obj)
	if !ok {
		return 0, false
	}
	if n <= 0 || n > 255 {
		return 0, false
	}
	return int(n), true
}

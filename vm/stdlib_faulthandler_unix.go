//go:build unix

package vm

import (
	"context"
	"fmt"
	"io"
	"os"
	ossignal "os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildFaulthandler() *object.Module {
	m := &object.Module{Name: "faulthandler", Dict: object.NewDict()}

	var mu sync.Mutex
	var enabled bool
	var enableCh chan os.Signal
	var laterCancel context.CancelFunc
	registeredSigs := make(map[syscall.Signal]chan os.Signal)

	stderrStream := func() object.Object {
		return &object.TextStream{Name: "stderr", W: i.Stderr}
	}

	resolveFile := func(kw *object.Dict) object.Object {
		if kw != nil {
			if f, ok := kw.GetStr("file"); ok {
				return f
			}
		}
		return stderrStream()
	}

	// extractWriter gets an io.Writer from a file object without calling Python.
	extractWriter := func(fileObj object.Object) io.Writer {
		if ts, ok := fileObj.(*object.TextStream); ok {
			if w, ok2 := ts.W.(io.Writer); ok2 {
				return w
			}
		}
		return i.Stderr
	}

	// writeToFile writes s to fileObj, using Python write() when necessary.
	writeToFile := func(ii any, fileObj object.Object, s string) {
		if ts, ok := fileObj.(*object.TextStream); ok {
			if w, ok2 := ts.W.(io.Writer); ok2 {
				w.Write([]byte(s)) //nolint
				return
			}
		}
		interp := ii.(*Interp)
		writeMethod, err := interp.getAttr(fileObj, "write")
		if err != nil {
			return
		}
		interp.callObject(writeMethod, []object.Object{&object.Str{V: s}}, nil) //nolint
	}

	// dumpCurrentThread walks curFrame chain and writes "File ..., line N, in func" lines.
	dumpCurrentThread := func(ii any, fileObj object.Object) {
		interp := ii.(*Interp)
		writeToFile(ii, fileObj, "Current thread 0x0000000000000001 (most recent call first):\n")
		for f := interp.curFrame; f != nil; f = f.Back {
			lineno := f.Code.LineForOffset(f.IP)
			if lineno == 0 {
				lineno = f.Code.FirstLineNo
			}
			writeToFile(ii, fileObj, fmt.Sprintf("  File \"%s\", line %d, in %s\n",
				f.Code.Filename, lineno, f.Code.Name))
		}
	}

	// ── enable ─────────────────────────────────────────────────────────────
	m.Dict.SetStr("enable", &object.BuiltinFunc{Name: "enable",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			mu.Lock()
			defer mu.Unlock()
			if enabled {
				return object.None, nil
			}
			enabled = true
			ch := make(chan os.Signal, 1)
			ossignal.Notify(ch, syscall.SIGSEGV, syscall.SIGFPE, syscall.SIGABRT,
				syscall.SIGBUS, syscall.SIGILL)
			enableCh = ch
			return object.None, nil
		},
	})

	// ── disable ────────────────────────────────────────────────────────────
	m.Dict.SetStr("disable", &object.BuiltinFunc{Name: "disable",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			mu.Lock()
			defer mu.Unlock()
			if !enabled {
				return object.None, nil
			}
			enabled = false
			if enableCh != nil {
				ossignal.Stop(enableCh)
				enableCh = nil
			}
			return object.None, nil
		},
	})

	// ── is_enabled ─────────────────────────────────────────────────────────
	m.Dict.SetStr("is_enabled", &object.BuiltinFunc{Name: "is_enabled",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			mu.Lock()
			defer mu.Unlock()
			return object.BoolOf(enabled), nil
		},
	})

	// ── dump_traceback ──────────────────────────────────────────────────────
	m.Dict.SetStr("dump_traceback", &object.BuiltinFunc{Name: "dump_traceback",
		Call: func(ii any, _ []object.Object, kw *object.Dict) (object.Object, error) {
			fileObj := resolveFile(kw)
			dumpCurrentThread(ii, fileObj)
			return object.None, nil
		},
	})

	// ── dump_c_stack ────────────────────────────────────────────────────────
	// Go has no C stack to display; this is intentionally a no-op.
	m.Dict.SetStr("dump_c_stack", &object.BuiltinFunc{Name: "dump_c_stack",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ── dump_traceback_later ────────────────────────────────────────────────
	m.Dict.SetStr("dump_traceback_later", &object.BuiltinFunc{Name: "dump_traceback_later",
		Call: func(ii any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "dump_traceback_later() requires timeout argument")
			}
			var secs float64
			switch v := a[0].(type) {
			case *object.Int:
				secs = float64(v.Int64())
			case *object.Float:
				secs = v.V
			default:
				return nil, object.Errorf(i.typeErr, "dump_traceback_later() timeout must be a number")
			}
			fileObj := resolveFile(kw)
			w := extractWriter(fileObj)
			repeat := false
			exitAfter := false
			if kw != nil {
				if rv, ok := kw.GetStr("repeat"); ok {
					repeat = object.Truthy(rv)
				}
				if ev, ok := kw.GetStr("exit"); ok {
					exitAfter = object.Truthy(ev)
				}
			}

			mu.Lock()
			if laterCancel != nil {
				laterCancel()
			}
			ctx, cancelFn := context.WithCancel(context.Background())
			laterCancel = cancelFn
			mu.Unlock()

			d := time.Duration(float64(time.Second) * secs)
			go func() {
				for {
					select {
					case <-time.After(d):
						fmt.Fprintf(w, "Current thread 0x0000000000000001 (most recent call first):\n")
						if exitAfter {
							os.Exit(1)
						}
						if !repeat {
							return
						}
					case <-ctx.Done():
						return
					}
				}
			}()
			return object.None, nil
		},
	})

	// ── cancel_dump_traceback_later ─────────────────────────────────────────
	m.Dict.SetStr("cancel_dump_traceback_later", &object.BuiltinFunc{Name: "cancel_dump_traceback_later",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			mu.Lock()
			defer mu.Unlock()
			if laterCancel != nil {
				laterCancel()
				laterCancel = nil
			}
			return object.None, nil
		},
	})

	// ── register ────────────────────────────────────────────────────────────
	m.Dict.SetStr("register", &object.BuiltinFunc{Name: "register",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "register() requires signum argument")
			}
			n, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "register() signum must be an int")
			}
			sig := syscall.Signal(n)
			mu.Lock()
			if _, exists := registeredSigs[sig]; !exists {
				ch := make(chan os.Signal, 1)
				ossignal.Notify(ch, sig)
				registeredSigs[sig] = ch
			}
			mu.Unlock()
			return object.None, nil
		},
	})

	// ── unregister ──────────────────────────────────────────────────────────
	m.Dict.SetStr("unregister", &object.BuiltinFunc{Name: "unregister",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "unregister() requires signum argument")
			}
			n, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "unregister() signum must be an int")
			}
			sig := syscall.Signal(n)
			mu.Lock()
			ch, exists := registeredSigs[sig]
			if exists {
				ossignal.Stop(ch)
				delete(registeredSigs, sig)
			}
			mu.Unlock()
			return object.BoolOf(exists), nil
		},
	})

	return m
}

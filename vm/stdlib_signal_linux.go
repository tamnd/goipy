//go:build linux && !arm64

package vm

import (
	"unsafe"

	"github.com/tamnd/goipy/object"
	"golang.org/x/sys/unix"
)

func (i *Interp) extendSignalModule(m *object.Module, ctx sigCtx) {
	errCls := ctx.errCls
	itimerErrCls := ctx.itimerErrCls

	setInt := func(name string, val int) {
		m.Dict.SetStr(name, object.NewInt(int64(val)))
	}

	// Linux-specific signal constants
	setInt("SIGIO", int(unix.SIGIO))
	setInt("SIGPWR", int(unix.SIGPWR))
	setInt("SIGSTKFLT", int(unix.SIGSTKFLT))

	// ── alarm(seconds) ────────────────────────────────────────────────────────
	m.Dict.SetStr("alarm", &object.BuiltinFunc{Name: "alarm",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			secs := uint(0)
			if len(a) > 0 {
				if n, ok := toInt64(a[0]); ok {
					secs = uint(n)
				}
			}
			rem, err := unix.Alarm(secs)
			if err != nil {
				return nil, object.Errorf(errCls, "alarm: %v", err)
			}
			return object.NewInt(int64(rem)), nil
		}})

	// ── getitimer(which) → (value, interval) ─────────────────────────────────
	m.Dict.SetStr("getitimer", &object.BuiltinFunc{Name: "getitimer",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "getitimer() requires which")
			}
			which, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "getitimer() which must be int")
			}
			it, err := unix.Getitimer(unix.ItimerWhich(which))
			if err != nil {
				return nil, object.Errorf(itimerErrCls, "getitimer: %v", err)
			}
			return &object.Tuple{V: []object.Object{
				&object.Float{V: itimervalToFloat(it.Value)},
				&object.Float{V: itimervalToFloat(it.Interval)},
			}}, nil
		}})

	// ── setitimer(which, seconds, interval=0) ─────────────────────────────────
	m.Dict.SetStr("setitimer", &object.BuiltinFunc{Name: "setitimer",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "setitimer() requires which and seconds")
			}
			which, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "setitimer() which must be int")
			}
			sec, ok2 := toFloat64(a[1])
			if !ok2 {
				return nil, object.Errorf(i.typeErr, "setitimer() seconds must be numeric")
			}
			interval := 0.0
			if len(a) > 2 {
				if f, ok3 := toFloat64(a[2]); ok3 {
					interval = f
				}
			}
			it := unix.Itimerval{
				Value:    floatToTimeval(sec),
				Interval: floatToTimeval(interval),
			}
			old, err := unix.Setitimer(unix.ItimerWhich(which), it)
			if err != nil {
				return nil, object.Errorf(itimerErrCls, "setitimer: %v", err)
			}
			return &object.Tuple{V: []object.Object{
				&object.Float{V: itimervalToFloat(old.Value)},
				&object.Float{V: itimervalToFloat(old.Interval)},
			}}, nil
		}})

	// ── pthread_sigmask(how, mask) → frozenset ────────────────────────────────
	m.Dict.SetStr("pthread_sigmask", &object.BuiltinFunc{Name: "pthread_sigmask",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "pthread_sigmask() requires how and mask")
			}
			how, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "pthread_sigmask() how must be int")
			}
			var newset unix.Sigset_t
			for _, obj := range pyListToSlice(a[1]) {
				n, ok2 := toInt64(obj)
				if !ok2 {
					continue
				}
				word := int(n) / 64
				bit := uint(n) % 64
				if word < len(newset.Val) {
					newset.Val[word] |= 1 << bit
				}
			}
			var oldset unix.Sigset_t
			if err := unix.PthreadSigmask(int(how), &newset, &oldset); err != nil {
				return nil, object.Errorf(errCls, "pthread_sigmask: %v", err)
			}
			fs := object.NewFrozenset()
			for word := 0; word < len(oldset.Val); word++ {
				v := oldset.Val[word]
				for bit := 0; bit < 64 && v != 0; bit++ {
					if v&(1<<uint(bit)) != 0 {
						signum := word*64 + bit
						if signum > 0 {
							fs.Add(object.NewInt(int64(signum))) //nolint
						}
					}
				}
			}
			return fs, nil
		}})

	// ── sigpending() → frozenset ──────────────────────────────────────────────
	// unix.Sigpending is not available in golang.org/x/sys; use rt_sigpending(2).
	m.Dict.SetStr("sigpending", &object.BuiltinFunc{Name: "sigpending",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			var set unix.Sigset_t
			_, _, errno := unix.RawSyscall(unix.SYS_RT_SIGPENDING,
				uintptr(unsafe.Pointer(&set)),
				uintptr(unsafe.Sizeof(set)), 0)
			if errno != 0 {
				return nil, object.Errorf(errCls, "sigpending: %v", errno)
			}
			fs := object.NewFrozenset()
			for word := 0; word < len(set.Val); word++ {
				v := set.Val[word]
				for bit := 0; bit < 64 && v != 0; bit++ {
					if v&(1<<uint(bit)) != 0 {
						signum := word*64 + bit
						if signum > 0 {
							fs.Add(object.NewInt(int64(signum))) //nolint
						}
					}
				}
			}
			return fs, nil
		}})

	// ── sigwait(sigset) → signum ──────────────────────────────────────────────
	// unix.Sigwait is not in golang.org/x/sys; implement via rt_sigtimedwait(2).
	m.Dict.SetStr("sigwait", &object.BuiltinFunc{Name: "sigwait",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "sigwait() requires sigset")
			}
			var set unix.Sigset_t
			for _, obj := range pyListToSlice(a[0]) {
				n, ok2 := toInt64(obj)
				if !ok2 {
					continue
				}
				word := int(n) / 64
				bit := uint(n) % 64
				if word < len(set.Val) {
					set.Val[word] |= 1 << bit
				}
			}
			// rt_sigtimedwait with NULL timeout blocks until a signal arrives.
			r1, _, errno := unix.RawSyscall6(unix.SYS_RT_SIGTIMEDWAIT,
				uintptr(unsafe.Pointer(&set)),
				0, 0, uintptr(unsafe.Sizeof(set)), 0, 0)
			if errno != 0 {
				return nil, object.Errorf(errCls, "sigwait: %v", errno)
			}
			return object.NewInt(int64(r1)), nil
		}})
}

// itimervalToFloat converts a unix.Timeval to seconds as float64.
func itimervalToFloat(tv unix.Timeval) float64 {
	return float64(tv.Sec) + float64(tv.Usec)*1e-6
}

// floatToTimeval converts seconds (float64) to unix.Timeval.
func floatToTimeval(sec float64) unix.Timeval {
	s := int64(sec)
	us := int64((sec - float64(s)) * 1e6)
	return unix.Timeval{Sec: s, Usec: us}
}

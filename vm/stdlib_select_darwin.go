//go:build darwin

package vm

import (
	"time"

	"github.com/tamnd/goipy/object"
	"golang.org/x/sys/unix"
)

// extendSelectModule adds kqueue/kevent objects and KQ* constants on macOS/BSD.
func (i *Interp) extendSelectModule(m *object.Module, errCls *object.Class) {
	setInt := func(name string, val int) {
		m.Dict.SetStr(name, object.NewInt(int64(val)))
	}

	// KQ_FILTER_* constants
	setInt("KQ_FILTER_READ", int(unix.EVFILT_READ))
	setInt("KQ_FILTER_WRITE", int(unix.EVFILT_WRITE))
	setInt("KQ_FILTER_AIO", int(unix.EVFILT_AIO))
	setInt("KQ_FILTER_VNODE", int(unix.EVFILT_VNODE))
	setInt("KQ_FILTER_PROC", int(unix.EVFILT_PROC))
	setInt("KQ_FILTER_SIGNAL", int(unix.EVFILT_SIGNAL))
	setInt("KQ_FILTER_TIMER", int(unix.EVFILT_TIMER))

	// KQ_EV_* constants
	setInt("KQ_EV_ADD", int(unix.EV_ADD))
	setInt("KQ_EV_DELETE", int(unix.EV_DELETE))
	setInt("KQ_EV_ENABLE", int(unix.EV_ENABLE))
	setInt("KQ_EV_DISABLE", int(unix.EV_DISABLE))
	setInt("KQ_EV_ONESHOT", int(unix.EV_ONESHOT))
	setInt("KQ_EV_CLEAR", int(unix.EV_CLEAR))
	setInt("KQ_EV_EOF", int(unix.EV_EOF))
	setInt("KQ_EV_ERROR", int(unix.EV_ERROR))

	// KQ_NOTE_* constants (vnode filter flags)
	setInt("KQ_NOTE_DELETE", int(unix.NOTE_DELETE))
	setInt("KQ_NOTE_WRITE", int(unix.NOTE_WRITE))
	setInt("KQ_NOTE_EXTEND", int(unix.NOTE_EXTEND))
	setInt("KQ_NOTE_ATTRIB", int(unix.NOTE_ATTRIB))
	setInt("KQ_NOTE_LINK", int(unix.NOTE_LINK))
	setInt("KQ_NOTE_RENAME", int(unix.NOTE_RENAME))
	setInt("KQ_NOTE_REVOKE", int(unix.NOTE_REVOKE))
	// Proc filter flags
	setInt("KQ_NOTE_EXIT", int(unix.NOTE_EXIT))
	setInt("KQ_NOTE_FORK", int(unix.NOTE_FORK))
	setInt("KQ_NOTE_EXEC", int(unix.NOTE_EXEC))
	setInt("KQ_NOTE_TRACK", int(unix.NOTE_TRACK))
	setInt("KQ_NOTE_CHILD", int(unix.NOTE_CHILD))
	setInt("KQ_NOTE_TRACKERR", int(unix.NOTE_TRACKERR))
	// Read/Write filter lowat flag
	setInt("KQ_NOTE_LOWAT", int(unix.NOTE_LOWAT))

	// ── kevent class ──────────────────────────────────────────────────────

	keventCls := &object.Class{Name: "kevent", Dict: object.NewDict()}

	makeKevent := func(ident, filter, flags, fflags, data, udata int64) *object.Instance {
		inst := &object.Instance{Class: keventCls, Dict: object.NewDict()}
		inst.Dict.SetStr("ident", object.NewInt(ident))
		inst.Dict.SetStr("filter", object.NewInt(filter))
		inst.Dict.SetStr("flags", object.NewInt(flags))
		inst.Dict.SetStr("fflags", object.NewInt(fflags))
		inst.Dict.SetStr("data", object.NewInt(data))
		inst.Dict.SetStr("udata", object.NewInt(udata))
		return inst
	}

	m.Dict.SetStr("kevent", &object.BuiltinFunc{Name: "kevent",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "kevent() requires ident")
			}
			ident, _ := toInt64(a[0])
			filter := int64(unix.EVFILT_READ)
			flags := int64(unix.EV_ADD)
			fflags := int64(0)
			data := int64(0)
			udata := int64(0)
			if len(a) > 1 {
				filter, _ = toInt64(a[1])
			}
			if len(a) > 2 {
				flags, _ = toInt64(a[2])
			}
			if len(a) > 3 {
				fflags, _ = toInt64(a[3])
			}
			if len(a) > 4 {
				data, _ = toInt64(a[4])
			}
			if len(a) > 5 {
				udata, _ = toInt64(a[5])
			}
			if kw != nil {
				if v, ok := kw.GetStr("filter"); ok {
					filter, _ = toInt64(v)
				}
				if v, ok := kw.GetStr("flags"); ok {
					flags, _ = toInt64(v)
				}
				if v, ok := kw.GetStr("fflags"); ok {
					fflags, _ = toInt64(v)
				}
				if v, ok := kw.GetStr("data"); ok {
					data, _ = toInt64(v)
				}
				if v, ok := kw.GetStr("udata"); ok {
					udata, _ = toInt64(v)
				}
			}
			return makeKevent(ident, filter, flags, fflags, data, udata), nil
		}})

	// ── kqueue object ─────────────────────────────────────────────────────

	kqueueCls := &object.Class{Name: "kqueue", Dict: object.NewDict()}

	makeKqueue := func(kqfd int) *object.Instance {
		closed := false
		inst := &object.Instance{Class: kqueueCls, Dict: object.NewDict()}

		inst.Dict.SetStr("fileno", &object.BuiltinFunc{Name: "fileno",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.NewInt(int64(kqfd)), nil
			}})

		inst.Dict.SetStr("closed", object.False)

		inst.Dict.SetStr("control", &object.BuiltinFunc{Name: "control",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) < 2 {
					return nil, object.Errorf(i.typeErr, "control() requires changelist and max_events")
				}
				maxEvents := 0
				if n, ok := toInt64(a[1]); ok {
					maxEvents = int(n)
				}
				timeoutNs := int64(-1) // block forever
				if len(a) > 2 && a[2] != object.None {
					if f, ok := toFloat64(a[2]); ok {
						timeoutNs = int64(f * float64(time.Second))
					}
				}

				// Build changes slice from changelist (list/tuple of kevent instances, or None)
				var changes []unix.Kevent_t
				if a[0] != object.None {
					objs := pyListToSlice(a[0])
					for _, obj := range objs {
						if inst2, ok := obj.(*object.Instance); ok {
							kev, err := keventFromInst(inst2)
							if err != nil {
								return nil, object.Errorf(errCls, "control: %v", err)
							}
							changes = append(changes, kev)
						}
					}
				}

				if maxEvents == 0 && len(changes) > 0 {
					// Submit changes only
					_, err := unix.Kevent(kqfd, changes, nil, nil)
					if err != nil {
						return nil, object.Errorf(errCls, "kqueue control: %v", err)
					}
					return &object.List{V: nil}, nil
				}

				events := make([]unix.Kevent_t, maxEvents)
				var ts *unix.Timespec
				if timeoutNs >= 0 {
					tsVal := unix.NsecToTimespec(timeoutNs)
					ts = &tsVal
				}
				n, err := unix.Kevent(kqfd, changes, events, ts)
				if err != nil && err != unix.EINTR {
					return nil, object.Errorf(errCls, "kqueue control: %v", err)
				}

				var out []object.Object
				for _, ev := range events[:n] {
					out = append(out, makeKevent(
						int64(ev.Ident),
						int64(ev.Filter),
						int64(ev.Flags),
						int64(ev.Fflags),
						int64(ev.Data),
						0, // Udata is *byte on darwin; not representable as int
					))
				}
				return &object.List{V: out}, nil
			}})

		inst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if !closed {
					unix.Close(kqfd) //nolint
					closed = true
					inst.Dict.SetStr("closed", object.True)
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return inst, nil
			}})

		inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				unix.Close(kqfd) //nolint
				return object.False, nil
			}})

		return inst
	}

	m.Dict.SetStr("kqueue", &object.BuiltinFunc{Name: "kqueue",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			kqfd, err := unix.Kqueue()
			if err != nil {
				return nil, object.Errorf(errCls, "kqueue(): %v", err)
			}
			return makeKqueue(kqfd), nil
		}})

	// kqueue.fromfd
	m.Dict.SetStr("kqueue_fromfd", &object.BuiltinFunc{Name: "kqueue_fromfd",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "kqueue_fromfd() requires fd")
			}
			fd, ok := i.getSockFd(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "kqueue_fromfd() argument has no valid fd")
			}
			return makeKqueue(fd), nil
		}})
}

// keventFromInst extracts a unix.Kevent_t from a Python kevent instance.
func keventFromInst(inst *object.Instance) (unix.Kevent_t, error) {
	get := func(name string) int64 {
		if v, ok := inst.Dict.GetStr(name); ok {
			if n, ok2 := toInt64(v); ok2 {
				return n
			}
		}
		return 0
	}
	kev := unix.Kevent_t{
		Ident:  uint64(get("ident")),
		Filter: int16(get("filter")),
		Flags:  uint16(get("flags")),
		Fflags: uint32(get("fflags")),
		Data:   int64(get("data")),
		Udata:  nil, // *byte on darwin; Python udata value ignored
	}
	return kev, nil
}

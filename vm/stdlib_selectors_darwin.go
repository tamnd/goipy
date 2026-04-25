//go:build darwin

package vm

import (
	"time"

	"github.com/tamnd/goipy/object"
	"golang.org/x/sys/unix"
)

// extendSelectorsModule adds KqueueSelector and sets DefaultSelector on macOS/BSD.
func (i *Interp) extendSelectorsModule(m *object.Module, ctx selectorCtx) {
	errCls := ctx.errCls

	makeKqueueSelector := func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		kqfd, err := unix.Kqueue()
		if err != nil {
			return nil, object.Errorf(errCls, "KqueueSelector: %v", err)
		}

		st := newSelectorState()

		// Each fd may need two kevent registrations (read + write).
		// We submit changes lazily via the changelist on the next Kevent call.
		buildChanges := func(fd int, events int64, flags uint16) []unix.Kevent_t {
			var changes []unix.Kevent_t
			if events&selectorEventRead != 0 {
				changes = append(changes, unix.Kevent_t{
					Ident:  uint64(fd),
					Filter: unix.EVFILT_READ,
					Flags:  flags,
				})
			}
			if events&selectorEventWrite != 0 {
				changes = append(changes, unix.Kevent_t{
					Ident:  uint64(fd),
					Filter: unix.EVFILT_WRITE,
					Flags:  flags,
				})
			}
			return changes
		}

		hooks := selectorHooks{
			onRegister: func(fd int, events int64) error {
				changes := buildChanges(fd, events, unix.EV_ADD|unix.EV_ENABLE)
				_, err := unix.Kevent(kqfd, changes, nil, nil)
				return err
			},
			onUnregister: func(fd int) {
				changes := buildChanges(fd, selectorEventRead|selectorEventWrite, unix.EV_DELETE)
				unix.Kevent(kqfd, changes, nil, nil) //nolint
			},
			onModify: func(fd int, newEvents int64) error {
				// Remove both filters, then re-add only the requested ones.
				del := buildChanges(fd, selectorEventRead|selectorEventWrite, unix.EV_DELETE)
				unix.Kevent(kqfd, del, nil, nil) //nolint
				add := buildChanges(fd, newEvents, unix.EV_ADD|unix.EV_ENABLE)
				_, err := unix.Kevent(kqfd, add, nil, nil)
				return err
			},
			selectFn: func(st *selectorState, timeoutMs int) ([]object.Object, error) {
				evts := make([]unix.Kevent_t, 64)
				var ts *unix.Timespec
				if timeoutMs >= 0 {
					nsec := int64(timeoutMs) * int64(time.Millisecond)
					tsVal := unix.NsecToTimespec(nsec)
					ts = &tsVal
				}
				n, err := unix.Kevent(kqfd, nil, evts, ts)
				if err != nil && err != unix.EINTR {
					return nil, object.Errorf(errCls, "KqueueSelector.select: %v", err)
				}
				st.mu.RLock()
				defer st.mu.RUnlock()
				// Merge multiple events for the same fd into one result.
				merged := map[int32]int64{}
				for _, ev := range evts[:n] {
					fd := int32(ev.Ident)
					if _, exists := st.keys[fd]; !exists {
						continue
					}
					if ev.Filter == unix.EVFILT_READ {
						merged[fd] |= selectorEventRead
					} else if ev.Filter == unix.EVFILT_WRITE {
						merged[fd] |= selectorEventWrite
					}
				}
				var out []object.Object
				for fd, sev := range merged {
					key, exists := st.keys[fd]
					if !exists {
						continue
					}
					out = append(out, &object.Tuple{V: []object.Object{key, object.NewInt(sev)}})
				}
				return out, nil
			},
			closeFn: func() {
				unix.Close(kqfd) //nolint
			},
		}

		return ctx.buildBase("KqueueSelector", st, hooks), nil
	}

	m.Dict.SetStr("KqueueSelector", &object.BuiltinFunc{Name: "KqueueSelector", Call: makeKqueueSelector})
	m.Dict.SetStr("DefaultSelector", &object.BuiltinFunc{Name: "DefaultSelector", Call: makeKqueueSelector})
}

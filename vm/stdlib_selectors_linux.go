//go:build linux

package vm

import (
	"github.com/tamnd/goipy/object"
	"golang.org/x/sys/unix"
)

// extendSelectorsModule adds EpollSelector and sets DefaultSelector on Linux.
func (i *Interp) extendSelectorsModule(m *object.Module, ctx selectorCtx) {
	errCls := ctx.errCls

	makeEpollSelector := func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		efd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
		if err != nil {
			return nil, object.Errorf(errCls, "EpollSelector: %v", err)
		}

		st := newSelectorState()

		eventsToEpoll := func(events int64) uint32 {
			var mask uint32
			if events&selectorEventRead != 0 {
				mask |= unix.EPOLLIN
			}
			if events&selectorEventWrite != 0 {
				mask |= unix.EPOLLOUT
			}
			return mask
		}

		hooks := selectorHooks{
			onRegister: func(fd int, events int64) error {
				ev := unix.EpollEvent{Events: eventsToEpoll(events), Fd: int32(fd)}
				return unix.EpollCtl(efd, unix.EPOLL_CTL_ADD, fd, &ev)
			},
			onUnregister: func(fd int) {
				unix.EpollCtl(efd, unix.EPOLL_CTL_DEL, fd, nil) //nolint
			},
			onModify: func(fd int, newEvents int64) error {
				ev := unix.EpollEvent{Events: eventsToEpoll(newEvents), Fd: int32(fd)}
				return unix.EpollCtl(efd, unix.EPOLL_CTL_MOD, fd, &ev)
			},
			selectFn: func(st *selectorState, timeoutMs int) ([]object.Object, error) {
				evts := make([]unix.EpollEvent, 64)
				n, err := unix.EpollWait(efd, evts, timeoutMs)
				if err != nil && err != unix.EINTR {
					return nil, object.Errorf(errCls, "EpollSelector.select: %v", err)
				}
				st.mu.RLock()
				defer st.mu.RUnlock()
				var out []object.Object
				for _, ev := range evts[:n] {
					key, exists := st.keys[ev.Fd]
					if !exists {
						continue
					}
					var sev int64
					if ev.Events&(unix.EPOLLIN|unix.EPOLLHUP|unix.EPOLLRDHUP) != 0 {
						sev |= selectorEventRead
					}
					if ev.Events&unix.EPOLLOUT != 0 {
						sev |= selectorEventWrite
					}
					if sev != 0 {
						out = append(out, &object.Tuple{V: []object.Object{key, object.NewInt(sev)}})
					}
				}
				return out, nil
			},
			closeFn: func() {
				unix.Close(efd) //nolint
			},
		}

		return ctx.buildBase("EpollSelector", st, hooks), nil
	}

	m.Dict.SetStr("EpollSelector", &object.BuiltinFunc{Name: "EpollSelector", Call: makeEpollSelector})
	m.Dict.SetStr("DefaultSelector", &object.BuiltinFunc{Name: "DefaultSelector", Call: makeEpollSelector})
}

//go:build linux

package vm

import (
	"sync"

	"github.com/tamnd/goipy/object"
	"golang.org/x/sys/unix"
)

func init() { selectPollrdhup = int16(unix.POLLRDHUP) }

// extendSelectModule adds the epoll object and EPOLL* constants on Linux.
func (i *Interp) extendSelectModule(m *object.Module, errCls *object.Class) {
	setInt := func(name string, val int) {
		m.Dict.SetStr(name, object.NewInt(int64(val)))
	}
	setInt("EPOLLIN", int(unix.EPOLLIN))
	setInt("EPOLLOUT", int(unix.EPOLLOUT))
	setInt("EPOLLERR", int(unix.EPOLLERR))
	setInt("EPOLLHUP", int(unix.EPOLLHUP))
	setInt("EPOLLET", int(unix.EPOLLET))
	setInt("EPOLLONESHOT", int(unix.EPOLLONESHOT))
	setInt("EPOLLRDHUP", int(unix.EPOLLRDHUP))
	setInt("EPOLLEXCLUSIVE", int(unix.EPOLLEXCLUSIVE))
	setInt("EPOLLPRI", int(unix.EPOLLPRI))
	setInt("EPOLLRDNORM", int(unix.EPOLLRDNORM))
	setInt("EPOLLRDBAND", int(unix.EPOLLRDBAND))
	setInt("EPOLLWRNORM", int(unix.EPOLLWRNORM))
	setInt("EPOLLWRBAND", int(unix.EPOLLWRBAND))
	setInt("EPOLLMSG", int(unix.EPOLLMSG))
	setInt("EPOLL_CLOEXEC", int(unix.EPOLL_CLOEXEC))

	epollCls := &object.Class{Name: "epoll", Dict: object.NewDict()}

	makeEpoll := func(efd int) *object.Instance {
		var mu sync.Mutex
		closed := false
		inst := &object.Instance{Class: epollCls, Dict: object.NewDict()}

		inst.Dict.SetStr("fileno", &object.BuiltinFunc{Name: "fileno",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.NewInt(int64(efd)), nil
			}})

		inst.Dict.SetStr("closed", object.False)

		inst.Dict.SetStr("register", &object.BuiltinFunc{Name: "register",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "register() requires fd")
				}
				fd, ok := i.getSockFd(a[0])
				if !ok {
					return nil, object.Errorf(i.typeErr, "register() argument has no valid fd")
				}
				events := uint32(unix.EPOLLIN | unix.EPOLLOUT)
				if len(a) > 1 {
					if n, ok2 := toInt64(a[1]); ok2 {
						events = uint32(n)
					}
				}
				ev := unix.EpollEvent{Events: events, Fd: int32(fd)}
				mu.Lock()
				defer mu.Unlock()
				if closed {
					return nil, object.Errorf(errCls, "epoll is closed")
				}
				if err := unix.EpollCtl(efd, unix.EPOLL_CTL_ADD, fd, &ev); err != nil {
					return nil, object.Errorf(errCls, "epoll register: %v", err)
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("modify", &object.BuiltinFunc{Name: "modify",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) < 2 {
					return nil, object.Errorf(i.typeErr, "modify() requires fd and eventmask")
				}
				fd, ok := i.getSockFd(a[0])
				if !ok {
					return nil, object.Errorf(i.typeErr, "modify() argument has no valid fd")
				}
				n, _ := toInt64(a[1])
				ev := unix.EpollEvent{Events: uint32(n), Fd: int32(fd)}
				mu.Lock()
				defer mu.Unlock()
				if err := unix.EpollCtl(efd, unix.EPOLL_CTL_MOD, fd, &ev); err != nil {
					return nil, object.Errorf(errCls, "epoll modify: %v", err)
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("unregister", &object.BuiltinFunc{Name: "unregister",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "unregister() requires fd")
				}
				fd, ok := i.getSockFd(a[0])
				if !ok {
					return nil, object.Errorf(i.typeErr, "unregister() argument has no valid fd")
				}
				mu.Lock()
				defer mu.Unlock()
				unix.EpollCtl(efd, unix.EPOLL_CTL_DEL, fd, nil) //nolint
				return object.None, nil
			}})

		inst.Dict.SetStr("poll", &object.BuiltinFunc{Name: "poll",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				timeoutMs := -1
				maxEvents := 64
				tvArg := object.Object(object.None)
				if len(a) > 0 {
					tvArg = a[0]
				}
				if kw != nil {
					if v, ok2 := kw.GetStr("timeout"); ok2 {
						tvArg = v
					}
					if v, ok2 := kw.GetStr("maxevents"); ok2 {
						if n, ok3 := toInt64(v); ok3 {
							maxEvents = int(n)
						}
					}
				}
				if tvArg != object.None {
					if f, ok2 := toFloat64(tvArg); ok2 {
						ms := int(f * 1000)
						if ms < 0 {
							ms = 0
						}
						timeoutMs = ms
					}
				}

				evts := make([]unix.EpollEvent, maxEvents)
				mu.Lock()
				isClose := closed
				mu.Unlock()
				if isClose {
					return nil, object.Errorf(errCls, "epoll is closed")
				}
				n, err := unix.EpollWait(efd, evts, timeoutMs)
				if err != nil && err != unix.EINTR {
					return nil, object.Errorf(errCls, "epoll poll: %v", err)
				}
				var out []object.Object
				for _, ev := range evts[:n] {
					out = append(out, &object.Tuple{V: []object.Object{
						object.NewInt(int64(ev.Fd)),
						object.NewInt(int64(ev.Events)),
					}})
				}
				return &object.List{V: out}, nil
			}})

		inst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				mu.Lock()
				defer mu.Unlock()
				if !closed {
					unix.Close(efd) //nolint
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
				unix.Close(efd) //nolint
				return object.False, nil
			}})

		return inst
	}

	m.Dict.SetStr("epoll", &object.BuiltinFunc{Name: "epoll",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			efd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
			if err != nil {
				return nil, object.Errorf(errCls, "epoll(): %v", err)
			}
			return makeEpoll(efd), nil
		}})

	// epoll.fromfd exposed as a module-level function for convenience.
	m.Dict.SetStr("epoll_fromfd", &object.BuiltinFunc{Name: "epoll_fromfd",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "epoll_fromfd() requires fd")
			}
			fd, ok := i.getSockFd(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "epoll_fromfd() argument has no valid fd")
			}
			return makeEpoll(fd), nil
		}})
}

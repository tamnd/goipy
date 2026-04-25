//go:build unix

package vm

import (
	"net"
	"sync"
	"time"

	"github.com/tamnd/goipy/object"
	"golang.org/x/sys/unix"
)


// pollrdhup is 0x2000 on Linux; defined here as fallback for non-Linux Unix.
// Overridden in stdlib_select_linux.go with the real unix.POLLRDHUP.
var selectPollrdhup = int16(0x2000)

// pollState holds the registered fds for a Python poll object.
type pollState struct {
	mu   sync.Mutex
	fds  map[int32]int16        // fd → event mask
	objs map[int32]object.Object // fd → original Python object
}

// extendSelectModule is declared in platform-specific files:
//   stdlib_select_linux.go  (linux)
//   stdlib_select_darwin.go (darwin)
//   stdlib_select_other.go  (unix && !linux && !darwin)
// Each provides its own implementation (or no-op).

// buildSelect constructs the select module.
func (i *Interp) buildSelect() *object.Module {
	m := &object.Module{Name: "select", Dict: object.NewDict()}

	errCls := i.osErr
	m.Dict.SetStr("error", errCls)

	// ── poll event constants ───────────────────────────────────────────────

	setInt := func(name string, val int) {
		m.Dict.SetStr(name, object.NewInt(int64(val)))
	}
	setInt("POLLIN", int(unix.POLLIN))
	setInt("POLLPRI", int(unix.POLLPRI))
	setInt("POLLOUT", int(unix.POLLOUT))
	setInt("POLLERR", int(unix.POLLERR))
	setInt("POLLHUP", int(unix.POLLHUP))
	setInt("POLLNVAL", int(unix.POLLNVAL))
	setInt("POLLRDHUP", int(selectPollrdhup))
	setInt("PIPE_BUF", 512) // POSIX minimum; real value platform-specific

	// ── select.select() ───────────────────────────────────────────────────

	m.Dict.SetStr("select", &object.BuiltinFunc{Name: "select",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, object.Errorf(i.typeErr, "select() requires at least 3 arguments")
			}
			rObjs := pyListToSlice(a[0])
			wObjs := pyListToSlice(a[1])
			xObjs := pyListToSlice(a[2])

			timeoutMs := -1
			var tvArg object.Object
			if len(a) > 3 {
				tvArg = a[3]
			}
			if kw != nil {
				if v, ok := kw.GetStr("timeout"); ok {
					tvArg = v
				}
			}
			if tvArg != nil && tvArg != object.None {
				if f, ok := toFloat64(tvArg); ok {
					ms := int(f * 1000)
					if ms < 0 {
						ms = 0
					}
					timeoutMs = ms
				}
			}

			// Build a deduplicated PollFd slice. Each fd may appear in
			// multiple lists; combine their event masks.
			type fdMeta struct {
				pfdIdx int
				rObj   object.Object
				wObj   object.Object
				xObj   object.Object
			}
			var pfds []unix.PollFd
			byFd := map[int32]*fdMeta{}

			addToList := func(obj object.Object, mask int16, setObj func(*fdMeta, object.Object)) {
				fd, ok := i.getSockFd(obj)
				if !ok {
					return
				}
				key := int32(fd)
				mm, exists := byFd[key]
				if !exists {
					mm = &fdMeta{pfdIdx: len(pfds)}
					pfds = append(pfds, unix.PollFd{Fd: key})
					byFd[key] = mm
				}
				pfds[mm.pfdIdx].Events |= mask
				setObj(mm, obj)
			}
			for _, obj := range rObjs {
				addToList(obj, unix.POLLIN, func(mm *fdMeta, o object.Object) {
					if mm.rObj == nil { mm.rObj = o }
				})
			}
			for _, obj := range wObjs {
				addToList(obj, unix.POLLOUT, func(mm *fdMeta, o object.Object) {
					if mm.wObj == nil { mm.wObj = o }
				})
			}
			for _, obj := range xObjs {
				addToList(obj, unix.POLLPRI|unix.POLLERR, func(mm *fdMeta, o object.Object) {
					if mm.xObj == nil { mm.xObj = o }
				})
			}

			if len(pfds) == 0 {
				if timeoutMs > 0 {
					time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
				}
				return emptySelectResult(), nil
			}

			_, err := unix.Poll(pfds, timeoutMs)
			if err != nil && err != unix.EINTR {
				return nil, object.Errorf(errCls, "select: %v", err)
			}

			var outR, outW, outX []object.Object
			for _, pfd := range pfds {
				mm := byFd[pfd.Fd]
				if mm == nil {
					continue
				}
				rev := pfd.Revents
				if rev&(unix.POLLIN|unix.POLLHUP|selectPollrdhup) != 0 && mm.rObj != nil {
					outR = append(outR, mm.rObj)
				}
				if rev&unix.POLLOUT != 0 && mm.wObj != nil {
					outW = append(outW, mm.wObj)
				}
				if rev&(unix.POLLERR|unix.POLLPRI) != 0 && mm.xObj != nil {
					outX = append(outX, mm.xObj)
				}
			}

			return &object.Tuple{V: []object.Object{
				&object.List{V: outR},
				&object.List{V: outW},
				&object.List{V: outX},
			}}, nil
		}})

	// ── poll object ───────────────────────────────────────────────────────

	pollCls := &object.Class{Name: "poll", Dict: object.NewDict()}

	m.Dict.SetStr("poll", &object.BuiltinFunc{Name: "poll",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ps := &pollState{
				fds:  make(map[int32]int16),
				objs: make(map[int32]object.Object),
			}
			inst := &object.Instance{Class: pollCls, Dict: object.NewDict()}

			inst.Dict.SetStr("register", &object.BuiltinFunc{Name: "register",
				Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
					if len(a) == 0 {
						return nil, object.Errorf(i.typeErr, "register() requires fd")
					}
					fd, ok := i.getSockFd(a[0])
					if !ok {
						return nil, object.Errorf(i.typeErr, "register() argument has no valid fd")
					}
					mask := int16(unix.POLLIN | unix.POLLPRI | unix.POLLOUT)
					if len(a) > 1 {
						if n, ok2 := toInt64(a[1]); ok2 {
							mask = int16(n)
						}
					}
					ps.mu.Lock()
					ps.fds[int32(fd)] = mask
					ps.objs[int32(fd)] = a[0]
					ps.mu.Unlock()
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
					ps.mu.Lock()
					if _, exists := ps.fds[int32(fd)]; !exists {
						ps.mu.Unlock()
						return nil, object.Errorf(errCls, "modify: fd not registered")
					}
					ps.fds[int32(fd)] = int16(n)
					ps.mu.Unlock()
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
					ps.mu.Lock()
					delete(ps.fds, int32(fd))
					delete(ps.objs, int32(fd))
					ps.mu.Unlock()
					return object.None, nil
				}})

			inst.Dict.SetStr("poll", &object.BuiltinFunc{Name: "poll",
				Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
					timeoutMs := -1
					if len(a) > 0 && a[0] != object.None {
						if f, ok := toFloat64(a[0]); ok {
							timeoutMs = int(f)
							if timeoutMs < 0 {
								timeoutMs = -1
							}
						}
					}
					ps.mu.Lock()
					pfds := make([]unix.PollFd, 0, len(ps.fds))
					for fd, mask := range ps.fds {
						pfds = append(pfds, unix.PollFd{Fd: fd, Events: mask})
					}
					ps.mu.Unlock()

					if len(pfds) == 0 {
						if timeoutMs > 0 {
							time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
						}
						return &object.List{V: nil}, nil
					}

					_, err := unix.Poll(pfds, timeoutMs)
					if err != nil && err != unix.EINTR {
						return nil, object.Errorf(errCls, "poll: %v", err)
					}

					var out []object.Object
					for _, pfd := range pfds {
						if pfd.Revents != 0 {
							out = append(out, &object.Tuple{V: []object.Object{
								object.NewInt(int64(pfd.Fd)),
								object.NewInt(int64(pfd.Revents)),
							}})
						}
					}
					return &object.List{V: out}, nil
				}})

			return inst, nil
		}})

	// Platform-specific additions (epoll on Linux, kqueue on macOS/BSD).
	i.extendSelectModule(m, errCls)

	return m
}

var _ net.Conn   // ensure net imported
var _ = time.Now // ensure time imported

//go:build windows

package vm

import (
	"time"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildSelectors() *object.Module {
	m := &object.Module{Name: "selectors", Dict: object.NewDict()}
	errCls := i.osErr

	m.Dict.SetStr("EVENT_READ", object.NewInt(selectorEventRead))
	m.Dict.SetStr("EVENT_WRITE", object.NewInt(selectorEventWrite))

	keyCls := &object.Class{Name: "SelectorKey", Dict: object.NewDict()}
	m.Dict.SetStr("SelectorKey", keyCls)

	makeSelectorKey := func(fileobj object.Object, fd int, events int64, data object.Object) *object.Instance {
		if data == nil {
			data = object.None
		}
		inst := &object.Instance{Class: keyCls, Dict: object.NewDict()}
		inst.Dict.SetStr("fileobj", fileobj)
		inst.Dict.SetStr("fd", object.NewInt(int64(fd)))
		inst.Dict.SetStr("events", object.NewInt(events))
		inst.Dict.SetStr("data", data)
		return inst
	}

	buildBase := func(className string, st *selectorState, hooks selectorHooks) *object.Instance {
		cls := &object.Class{Name: className, Dict: object.NewDict()}
		inst := &object.Instance{Class: cls, Dict: object.NewDict()}

		inst.Dict.SetStr("register", &object.BuiltinFunc{Name: "register",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "register() requires fileobj")
				}
				fd, ok := i.getSockFd(a[0])
				if !ok {
					return nil, object.Errorf(i.typeErr, "register() argument has no valid fd")
				}
				events := selectorEventRead
				if len(a) > 1 {
					if n, ok2 := toInt64(a[1]); ok2 {
						events = n
					}
				}
				var data object.Object = object.None
				if len(a) > 2 {
					data = a[2]
				}
				if kw != nil {
					if v, ok2 := kw.GetStr("data"); ok2 {
						data = v
					}
					if v, ok2 := kw.GetStr("events"); ok2 {
						if n, ok3 := toInt64(v); ok3 {
							events = n
						}
					}
				}
				if events == 0 || events&^(selectorEventRead|selectorEventWrite) != 0 {
					return nil, object.Errorf(i.valueErr, "invalid events: %d", events)
				}
				st.mu.Lock()
				defer st.mu.Unlock()
				if st.closed {
					return nil, object.Errorf(errCls, "selector is closed")
				}
				if _, exists := st.keys[int32(fd)]; exists {
					return nil, object.Errorf(i.keyErr, "%d is already registered", fd)
				}
				if hooks.onRegister != nil {
					if err := hooks.onRegister(fd, events); err != nil {
						return nil, object.Errorf(errCls, "register: %v", err)
					}
				}
				key := makeSelectorKey(a[0], fd, events, data)
				st.keys[int32(fd)] = key
				return key, nil
			}})

		inst.Dict.SetStr("unregister", &object.BuiltinFunc{Name: "unregister",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "unregister() requires fileobj")
				}
				fd, ok := i.getSockFd(a[0])
				if !ok {
					return nil, object.Errorf(i.typeErr, "unregister() argument has no valid fd")
				}
				st.mu.Lock()
				defer st.mu.Unlock()
				key, exists := st.keys[int32(fd)]
				if !exists {
					return nil, object.Errorf(i.keyErr, "%d is not registered", fd)
				}
				if hooks.onUnregister != nil {
					hooks.onUnregister(fd)
				}
				delete(st.keys, int32(fd))
				return key, nil
			}})

		inst.Dict.SetStr("modify", &object.BuiltinFunc{Name: "modify",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "modify() requires fileobj")
				}
				fd, ok := i.getSockFd(a[0])
				if !ok {
					return nil, object.Errorf(i.typeErr, "modify() argument has no valid fd")
				}
				events := selectorEventRead
				if len(a) > 1 {
					if n, ok2 := toInt64(a[1]); ok2 {
						events = n
					}
				}
				var data object.Object = object.None
				if len(a) > 2 {
					data = a[2]
				}
				if kw != nil {
					if v, ok2 := kw.GetStr("data"); ok2 {
						data = v
					}
				}
				if events == 0 || events&^(selectorEventRead|selectorEventWrite) != 0 {
					return nil, object.Errorf(i.valueErr, "invalid events: %d", events)
				}
				st.mu.Lock()
				defer st.mu.Unlock()
				if _, exists := st.keys[int32(fd)]; !exists {
					return nil, object.Errorf(i.keyErr, "%d is not registered", fd)
				}
				if hooks.onModify != nil {
					if err := hooks.onModify(fd, events); err != nil {
						return nil, object.Errorf(errCls, "modify: %v", err)
					}
				}
				key := makeSelectorKey(a[0], fd, events, data)
				st.keys[int32(fd)] = key
				return key, nil
			}})

		inst.Dict.SetStr("select", &object.BuiltinFunc{Name: "select",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				st.mu.RLock()
				closed := st.closed
				st.mu.RUnlock()
				if closed {
					return nil, object.Errorf(errCls, "selector is closed")
				}
				timeoutMs := -1
				var tvArg object.Object
				if len(a) > 0 {
					tvArg = a[0]
				}
				if kw != nil {
					if v, ok2 := kw.GetStr("timeout"); ok2 {
						tvArg = v
					}
				}
				if tvArg != nil && tvArg != object.None {
					if f, ok2 := toFloat64(tvArg); ok2 {
						if f >= 0 {
							ms := int(f * 1000)
							if ms < 0 {
								ms = 0
							}
							timeoutMs = ms
						}
					}
				}
				out, err := hooks.selectFn(st, timeoutMs)
				if err != nil {
					return nil, err
				}
				if out == nil {
					out = []object.Object{}
				}
				return &object.List{V: out}, nil
			}})

		inst.Dict.SetStr("get_key", &object.BuiltinFunc{Name: "get_key",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "get_key() requires fileobj")
				}
				fd, ok := i.getSockFd(a[0])
				if !ok {
					return nil, object.Errorf(i.typeErr, "get_key() argument has no valid fd")
				}
				st.mu.RLock()
				key, exists := st.keys[int32(fd)]
				st.mu.RUnlock()
				if !exists {
					return nil, object.Errorf(i.keyErr, "%d is not registered", fd)
				}
				return key, nil
			}})

		inst.Dict.SetStr("get_map", &object.BuiltinFunc{Name: "get_map",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				st.mu.RLock()
				defer st.mu.RUnlock()
				d := object.NewDict()
				for fd, key := range st.keys {
					d.Set(object.NewInt(int64(fd)), key)
				}
				return d, nil
			}})

		doClose := func() {
			st.mu.Lock()
			defer st.mu.Unlock()
			if !st.closed {
				st.closed = true
				if hooks.closeFn != nil {
					hooks.closeFn()
				}
			}
		}

		inst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				doClose()
				return object.None, nil
			}})

		inst.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return inst, nil
			}})

		inst.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				doClose()
				return object.False, nil
			}})

		return inst
	}

	// wsaSelectFn implements the select logic using WSAPoll for Windows sockets.
	wsaSelectFn := func(st *selectorState, timeoutMs int) ([]object.Object, error) {
		st.mu.RLock()
		pfds := make([]wsaPollFd, 0, len(st.keys))
		fdKeys := make(map[uintptr]*object.Instance, len(st.keys))
		for fd, key := range st.keys {
			var mask int16
			if ev, ok2 := key.Dict.GetStr("events"); ok2 {
				if n, ok3 := toInt64(ev); ok3 {
					if n&selectorEventRead != 0 {
						mask |= wsaPOLLIN
					}
					if n&selectorEventWrite != 0 {
						mask |= wsaPOLLOUT
					}
				}
			}
			pfds = append(pfds, wsaPollFd{fd: uintptr(fd), events: mask})
			fdKeys[uintptr(fd)] = key
		}
		st.mu.RUnlock()

		if len(pfds) == 0 {
			if timeoutMs > 0 {
				time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
			}
			return nil, nil
		}

		n := wsaPoll(pfds, int32(timeoutMs))
		if n < 0 {
			return nil, object.Errorf(errCls, "select: WSAPoll failed")
		}

		var out []object.Object
		for _, pfd := range pfds {
			if pfd.revents == 0 {
				continue
			}
			key, exists := fdKeys[pfd.fd]
			if !exists {
				continue
			}
			var evts int64
			if pfd.revents&wsaPOLLIN != 0 {
				evts |= selectorEventRead
			}
			if pfd.revents&wsaPOLLOUT != 0 {
				evts |= selectorEventWrite
			}
			if evts != 0 {
				out = append(out, &object.Tuple{V: []object.Object{key, object.NewInt(evts)}})
			}
		}
		return out, nil
	}

	makeSelectSelector := func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		st := newSelectorState()
		return buildBase("SelectSelector", st, selectorHooks{selectFn: wsaSelectFn}), nil
	}

	m.Dict.SetStr("SelectSelector", &object.BuiltinFunc{Name: "SelectSelector", Call: makeSelectSelector})
	m.Dict.SetStr("DefaultSelector", &object.BuiltinFunc{Name: "DefaultSelector", Call: makeSelectSelector})

	return m
}

// extendSelectorsModule is a no-op on Windows (no epoll/kqueue).
func (i *Interp) extendSelectorsModule(_ *object.Module, _ selectorCtx) {}

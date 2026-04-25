//go:build unix

package vm

import (
	"sync"
	"time"

	"github.com/tamnd/goipy/object"
	"golang.org/x/sys/unix"
)

const selectorEventRead = int64(1)
const selectorEventWrite = int64(2)

// selectorState is the shared fd→SelectorKey registry for all selector types.
type selectorState struct {
	mu     sync.RWMutex
	keys   map[int32]*object.Instance // fd → SelectorKey
	closed bool
}

func newSelectorState() *selectorState {
	return &selectorState{keys: make(map[int32]*object.Instance)}
}

// selectorHooks are called by the base selector on fd lifecycle events.
type selectorHooks struct {
	onRegister   func(fd int, events int64) error // called after key stored
	onUnregister func(fd int)                     // called before key removed
	onModify     func(fd int, newEvents int64) error
	selectFn     func(st *selectorState, timeoutMs int) ([]object.Object, error)
	closeFn      func()
}

// selectorCtx bundles the helpers that platform files need to build selectors.
type selectorCtx struct {
	errCls          *object.Class
	makeSelectorKey func(fileobj object.Object, fd int, events int64, data object.Object) *object.Instance
	buildBase       func(className string, st *selectorState, hooks selectorHooks) *object.Instance
}

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

	// buildBase constructs a selector instance with common register/unregister/
	// modify/select/close/get_key/get_map/__enter__/__exit__ methods.
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

	// pollSelectFn builds the select() implementation for Poll/Select selectors.
	pollSelectFn := func(st *selectorState, timeoutMs int) ([]object.Object, error) {
		st.mu.RLock()
		pfds := make([]unix.PollFd, 0, len(st.keys))
		for fd, key := range st.keys {
			var mask int16
			if ev, ok2 := key.Dict.GetStr("events"); ok2 {
				if n, ok3 := toInt64(ev); ok3 {
					if n&selectorEventRead != 0 {
						mask |= unix.POLLIN
					}
					if n&selectorEventWrite != 0 {
						mask |= unix.POLLOUT
					}
				}
			}
			pfds = append(pfds, unix.PollFd{Fd: fd, Events: mask})
		}
		st.mu.RUnlock()

		if len(pfds) == 0 {
			if timeoutMs > 0 {
				time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
			}
			return nil, nil
		}

		_, err := unix.Poll(pfds, timeoutMs)
		if err != nil && err != unix.EINTR {
			return nil, object.Errorf(errCls, "select: %v", err)
		}

		st.mu.RLock()
		defer st.mu.RUnlock()
		var out []object.Object
		for _, pfd := range pfds {
			if pfd.Revents == 0 {
				continue
			}
			key, exists := st.keys[pfd.Fd]
			if !exists {
				continue
			}
			var evts int64
			if pfd.Revents&(unix.POLLIN|unix.POLLHUP|selectPollrdhup) != 0 {
				evts |= selectorEventRead
			}
			if pfd.Revents&unix.POLLOUT != 0 {
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
		return buildBase("SelectSelector", st, selectorHooks{selectFn: pollSelectFn}), nil
	}
	makePollSelector := func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		st := newSelectorState()
		return buildBase("PollSelector", st, selectorHooks{selectFn: pollSelectFn}), nil
	}

	m.Dict.SetStr("SelectSelector", &object.BuiltinFunc{Name: "SelectSelector", Call: makeSelectSelector})
	m.Dict.SetStr("PollSelector", &object.BuiltinFunc{Name: "PollSelector", Call: makePollSelector})

	ctx := selectorCtx{
		errCls:          errCls,
		makeSelectorKey: makeSelectorKey,
		buildBase:       buildBase,
	}
	i.extendSelectorsModule(m, ctx)

	return m
}

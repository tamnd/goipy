//go:build windows

package vm

import (
	"syscall"
	"time"
	"unsafe"

	"github.com/tamnd/goipy/object"
)

var ws2_32 = syscall.NewLazyDLL("ws2_32.dll")
var wsaPollProc = ws2_32.NewProc("WSAPoll")

type wsaPollFd struct {
	fd      uintptr
	events  int16
	revents int16
}

const (
	wsaPOLLIN  = int16(0x0001)
	wsaPOLLOUT = int16(0x0004)
	wsaPOLLERR = int16(0x0008)
	wsaPOLLHUP = int16(0x0010)
)

// wsaPoll wraps the Winsock2 WSAPoll function.
func wsaPoll(fds []wsaPollFd, timeoutMs int32) int {
	if len(fds) == 0 {
		return 0
	}
	r, _, _ := wsaPollProc.Call(
		uintptr(unsafe.Pointer(&fds[0])),
		uintptr(len(fds)),
		uintptr(timeoutMs),
	)
	return int(int32(r))
}

func (i *Interp) buildSelect() *object.Module {
	m := &object.Module{Name: "select", Dict: object.NewDict()}
	errCls := i.osErr
	m.Dict.SetStr("error", errCls)

	setInt := func(name string, val int) {
		m.Dict.SetStr(name, object.NewInt(int64(val)))
	}
	setInt("POLLIN", int(wsaPOLLIN))
	setInt("POLLOUT", int(wsaPOLLOUT))
	setInt("POLLERR", int(wsaPOLLERR))
	setInt("POLLHUP", int(wsaPOLLHUP))

	m.Dict.SetStr("select", &object.BuiltinFunc{Name: "select",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return nil, object.Errorf(i.typeErr, "select() requires at least 3 arguments")
			}
			rObjs := pyListToSlice(a[0])
			wObjs := pyListToSlice(a[1])
			xObjs := pyListToSlice(a[2])

			timeoutMs := int32(-1)
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
				if f, ok := toFloat64(tvArg); ok && f >= 0 {
					ms := int32(f * 1000)
					if ms < 0 {
						ms = 0
					}
					timeoutMs = ms
				}
			}

			type entry struct {
				obj   object.Object
				which int // 0=read, 1=write, 2=except
			}

			var entries []entry
			var pfds []wsaPollFd
			fdIdx := map[uintptr]int{}

			addFd := func(obj object.Object, events int16, which int) {
				fd, ok := i.getSockFd(obj)
				if !ok || fd < 0 {
					return
				}
				key := uintptr(fd)
				if idx, exists := fdIdx[key]; exists {
					pfds[idx].events |= events
					return
				}
				fdIdx[key] = len(pfds)
				pfds = append(pfds, wsaPollFd{fd: key, events: events})
				entries = append(entries, entry{obj, which})
			}

			for _, obj := range rObjs {
				addFd(obj, wsaPOLLIN, 0)
			}
			for _, obj := range wObjs {
				addFd(obj, wsaPOLLOUT, 1)
			}
			for _, obj := range xObjs {
				addFd(obj, wsaPOLLERR|wsaPOLLHUP, 2)
			}

			if len(pfds) == 0 {
				if timeoutMs > 0 {
					time.Sleep(time.Duration(timeoutMs) * time.Millisecond)
				}
				return emptySelectResult(), nil
			}

			n := wsaPoll(pfds, timeoutMs)
			if n < 0 {
				return nil, object.Errorf(errCls, "select: WSAPoll failed")
			}

			var outR, outW, outX []object.Object
			for idx, pfd := range pfds {
				rev := pfd.revents
				if rev == 0 {
					continue
				}
				e := entries[idx]
				switch e.which {
				case 0:
					if rev&wsaPOLLIN != 0 {
						outR = append(outR, e.obj)
					}
				case 1:
					if rev&wsaPOLLOUT != 0 {
						outW = append(outW, e.obj)
					}
				case 2:
					if rev&(wsaPOLLERR|wsaPOLLHUP) != 0 {
						outX = append(outX, e.obj)
					}
				}
			}

			return &object.Tuple{V: []object.Object{
				&object.List{V: outR},
				&object.List{V: outW},
				&object.List{V: outX},
			}}, nil
		}})

	return m
}

package vm

import "github.com/tamnd/goipy/object"

// getSockFd extracts the OS file descriptor from a Python socket instance,
// a plain integer, or any object whose Dict contains "fileno".
func (i *Interp) getSockFd(obj object.Object) (int, bool) {
	if n, ok := toInt64(obj); ok {
		return int(n), n >= 0
	}
	if st := sockStateOf(obj); st != nil {
		st.mu.RLock()
		conn, ln := st.conn, st.listener
		st.mu.RUnlock()
		if conn != nil {
			if fd := connFd(conn); fd >= 0 {
				return fd, true
			}
		}
		if ln != nil {
			if fd := connFd(ln); fd >= 0 {
				return fd, true
			}
		}
	}
	if inst, ok := obj.(*object.Instance); ok {
		if fn, ok2 := inst.Dict.GetStr("fileno"); ok2 {
			if r, err := i.callObject(fn, nil, nil); err == nil {
				if n, ok3 := toInt64(r); ok3 && n >= 0 {
					return int(n), true
				}
			}
		}
	}
	return -1, false
}

func emptySelectResult() object.Object {
	return &object.Tuple{V: []object.Object{
		&object.List{V: nil},
		&object.List{V: nil},
		&object.List{V: nil},
	}}
}

// pyListToSlice converts a Python list/tuple to []object.Object.
func pyListToSlice(obj object.Object) []object.Object {
	switch v := obj.(type) {
	case *object.List:
		return v.V
	case *object.Tuple:
		return v.V
	}
	return nil
}

package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildFcntl() *object.Module {
	m := &object.Module{Name: "fcntl", Dict: object.NewDict()}
	d := m.Dict

	consts := map[string]int64{
		// basic F_ commands
		"F_DUPFD":         0,
		"F_GETFD":         1,
		"F_SETFD":         2,
		"F_GETFL":         3,
		"F_SETFL":         4,
		"F_GETOWN":        5,
		"F_SETOWN":        6,
		"F_GETLK":         7,
		"F_SETLK":         8,
		"F_SETLKW":        9,
		"F_DUPFD_CLOEXEC": 67,
		// lock types
		"F_RDLCK": 1,
		"F_WRLCK": 3,
		"F_UNLCK": 2,
		// fd flags
		"FD_CLOEXEC": 1,
		// flock() operations
		"LOCK_SH": 1,
		"LOCK_EX": 2,
		"LOCK_NB": 4,
		"LOCK_UN": 8,
		// macOS-specific
		"F_GETNOSIGPIPE": 74,
		"F_SETNOSIGPIPE": 73,
		"F_NOCACHE":      48,
		"F_FULLFSYNC":    51,
		"F_RDAHEAD":      45,
		"F_GETPATH":      50,
		// lease / OFD (macOS 10.x+)
		"F_GETLEASE":  107,
		"F_SETLEASE":  106,
		"F_OFD_GETLK": 92,
		"F_OFD_SETLK": 90,
		"F_OFD_SETLKW": 91,
		// misc
		"FASYNC": 64,
	}
	for name, val := range consts {
		d.SetStr(name, intObj(val))
	}

	// fcntl(fd, cmd[, arg]) -> int
	d.SetStr("fcntl", &object.BuiltinFunc{
		Name: "fcntl",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj(0), nil
		},
	})

	// flock(fd, operation) -> None  (may raise OSError)
	d.SetStr("flock", &object.BuiltinFunc{
		Name: "flock",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	// ioctl(fd, request[, arg[, mutate_flag]]) -> int or bytes
	d.SetStr("ioctl", &object.BuiltinFunc{
		Name: "ioctl",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			// if third arg is bytes/bytearray, return bytes; otherwise int
			if len(a) >= 3 {
				switch a[2].(type) {
				case *object.Bytes:
					return &object.Bytes{V: []byte{}}, nil
				}
			}
			return intObj(0), nil
		},
	})

	// lockf(fd, cmd[, len[, start[, whence]]]) -> None
	d.SetStr("lockf", &object.BuiltinFunc{
		Name: "lockf",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	return m
}

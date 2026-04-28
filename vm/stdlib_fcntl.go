package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildFcntl() *object.Module {
	m := &object.Module{Name: "fcntl", Dict: object.NewDict()}
	d := m.Dict

	consts := map[string]int64{
		"F_DUPFD":      0,
		"F_GETFD":      1,
		"F_SETFD":      2,
		"F_GETFL":      3,
		"F_SETFL":      4,
		"F_GETOWN":     5,
		"F_SETOWN":     6,
		"F_GETLK":      7,
		"F_SETLK":      8,
		"F_SETLKW":     9,
		"F_RDLCK":      1,
		"F_WRLCK":      3,
		"F_UNLCK":      2,
		"FD_CLOEXEC":   1,
		"LOCK_SH":      1,
		"LOCK_EX":      2,
		"LOCK_NB":      4,
		"LOCK_UN":      8,
		"F_DUPFD_CLOEXEC": 67,
		"F_GETNOSIGPIPE": 74,
		"F_SETNOSIGPIPE": 73,
		"F_NOCACHE":    48,
		"F_PREALLOCATE": 42,
		"F_SETSIZE":    43,
		"F_RDADVISE":   44,
		"F_RDAHEAD":    60,
		"F_FULLFSYNC":  51,
		"F_FREEZE_FS":  53,
		"F_THAW_FS":    54,
		"FASYNC":       0x40,
	}
	for name, val := range consts {
		d.SetStr(name, intObj(val))
	}

	noneStub := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		}
	}

	d.SetStr("fcntl", noneStub("fcntl"))
	d.SetStr("flock", noneStub("flock"))
	d.SetStr("ioctl", noneStub("ioctl"))
	d.SetStr("lockf", noneStub("lockf"))

	return m
}

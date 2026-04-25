//go:build windows

package vm

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/tamnd/goipy/object"
)

func registerOsSyscalls(i *Interp, m *object.Module) {
	// umask is not available on Windows; return the conventional POSIX default.
	m.Dict.SetStr("umask", &object.BuiltinFunc{Name: "umask", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(0o022), nil
	}})

	// access: check file existence via os.Stat.
	m.Dict.SetStr("access", &object.BuiltinFunc{Name: "access", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "access() requires path and mode")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "access() path must be str")
		}
		_, err := os.Stat(p.V)
		return object.BoolOf(err == nil), nil
	}})

	// open: returns a Windows HANDLE cast to int64.
	m.Dict.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "os.open() requires path and flags")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.open() path must be str")
		}
		flags, ok2 := toInt64(a[1])
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "os.open() flags must be int")
		}
		mode := int64(0o666)
		if len(a) >= 3 {
			if n, ok3 := toInt64(a[2]); ok3 {
				mode = n
			}
		}
		fd, err := syscall.Open(p.V, int(flags), uint32(mode))
		if err != nil {
			return nil, object.Errorf(i.osErr, "[Errno %d] %v: '%s'", err.(syscall.Errno), err, p.V)
		}
		return object.NewInt(int64(fd)), nil
	}})

	// read: syscall.Read uses Windows HANDLE.
	m.Dict.SetStr("read", &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "os.read() requires fd and n")
		}
		fd, ok1 := toInt64(a[0])
		n, ok2 := toInt64(a[1])
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "os.read() fd and n must be int")
		}
		buf := make([]byte, n)
		nr, err := syscall.Read(syscall.Handle(fd), buf)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Bytes{V: buf[:nr]}, nil
	}})

	// write: syscall.Write uses Windows HANDLE.
	m.Dict.SetStr("write", &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "os.write() requires fd and data")
		}
		fd, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.write() fd must be int")
		}
		data, err2 := asBytes(a[1])
		if err2 != nil {
			return nil, err2
		}
		nw, err := syscall.Write(syscall.Handle(fd), data)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.NewInt(int64(nw)), nil
	}})

	// lseek: syscall.Seek uses Windows HANDLE.
	m.Dict.SetStr("lseek", &object.BuiltinFunc{Name: "lseek", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "os.lseek() requires fd, pos, how")
		}
		fd, ok1 := toInt64(a[0])
		pos, ok2 := toInt64(a[1])
		how, ok3 := toInt64(a[2])
		if !ok1 || !ok2 || !ok3 {
			return nil, object.Errorf(i.typeErr, "os.lseek() arguments must be int")
		}
		newpos, err := syscall.Seek(syscall.Handle(fd), pos, int(how))
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.NewInt(newpos), nil
	}})

	// fstat: use os.NewFile to get file info from the HANDLE.
	m.Dict.SetStr("fstat", &object.BuiltinFunc{Name: "fstat", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.fstat() requires 1 argument")
		}
		fd, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.fstat() fd must be int")
		}
		f := os.NewFile(uintptr(fd), "")
		if f == nil {
			return nil, object.Errorf(i.osErr, "fstat: invalid fd")
		}
		info, err := f.Stat()
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		inst := &object.Instance{Class: statResultClass, Dict: object.NewDict()}
		inst.Dict.SetStr("st_mode", object.NewInt(goModeToPosix(info.Mode())))
		inst.Dict.SetStr("st_ino", object.NewInt(0))
		inst.Dict.SetStr("st_dev", object.NewInt(0))
		inst.Dict.SetStr("st_nlink", object.NewInt(1))
		inst.Dict.SetStr("st_uid", object.NewInt(0))
		inst.Dict.SetStr("st_gid", object.NewInt(0))
		inst.Dict.SetStr("st_size", object.NewInt(info.Size()))
		mt := float64(info.ModTime().UnixNano()) / 1e9
		inst.Dict.SetStr("st_atime", &object.Float{V: mt})
		inst.Dict.SetStr("st_mtime", &object.Float{V: mt})
		inst.Dict.SetStr("st_ctime", &object.Float{V: mt})
		return inst, nil
	}})

	// dup: duplicate a HANDLE using DuplicateHandle.
	m.Dict.SetStr("dup", &object.BuiltinFunc{Name: "dup", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.dup() requires 1 argument")
		}
		fd, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.dup() fd must be int")
		}
		proc, err := windows.GetCurrentProcess()
		if err != nil {
			return nil, object.Errorf(i.osErr, "dup: %v", err)
		}
		var newHandle windows.Handle
		if err = windows.DuplicateHandle(proc, windows.Handle(fd), proc, &newHandle, 0, false, windows.DUPLICATE_SAME_ACCESS); err != nil {
			return nil, object.Errorf(i.osErr, "dup: %v", err)
		}
		return object.NewInt(int64(newHandle)), nil
	}})

	// dup2: not available on Windows with HANDLE semantics.
	m.Dict.SetStr("dup2", &object.BuiltinFunc{Name: "dup2", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return nil, object.Errorf(i.osErr, "os.dup2: not supported on Windows")
	}})
}

func osDup2(_, _ int) error {
	return syscall.EWINDOWS
}

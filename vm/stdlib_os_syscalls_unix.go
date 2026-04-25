//go:build unix

package vm

import (
	"syscall"

	"github.com/tamnd/goipy/object"
)

func registerOsSyscalls(i *Interp, m *object.Module) {
	m.Dict.SetStr("umask", &object.BuiltinFunc{Name: "umask", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "umask() requires 1 argument")
		}
		n, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "umask() argument must be int")
		}
		old := syscall.Umask(int(n))
		return object.NewInt(int64(old)), nil
	}})

	m.Dict.SetStr("access", &object.BuiltinFunc{Name: "access", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "access() requires path and mode")
		}
		p, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "access() path must be str")
		}
		mode, ok2 := toInt64(a[1])
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "access() mode must be int")
		}
		err := syscall.Access(p.V, uint32(mode))
		return object.BoolOf(err == nil), nil
	}})

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
		mode := int64(0o777)
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
		nr, err := syscall.Read(int(fd), buf)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Bytes{V: buf[:nr]}, nil
	}})

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
		nw, err := syscall.Write(int(fd), data)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.NewInt(int64(nw)), nil
	}})

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
		newpos, err := syscall.Seek(int(fd), pos, int(how))
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.NewInt(newpos), nil
	}})

	m.Dict.SetStr("fstat", &object.BuiltinFunc{Name: "fstat", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.fstat() requires 1 argument")
		}
		fd, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.fstat() fd must be int")
		}
		var st syscall.Stat_t
		if err := syscall.Fstat(int(fd), &st); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return osSyscallStatResult(i, &st), nil
	}})

	m.Dict.SetStr("dup", &object.BuiltinFunc{Name: "dup", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "os.dup() requires 1 argument")
		}
		fd, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "os.dup() fd must be int")
		}
		newfd, err := syscall.Dup(int(fd))
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.NewInt(int64(newfd)), nil
	}})

	m.Dict.SetStr("dup2", &object.BuiltinFunc{Name: "dup2", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "os.dup2() requires oldfd and newfd")
		}
		oldfd, ok1 := toInt64(a[0])
		newfd, ok2 := toInt64(a[1])
		if !ok1 || !ok2 {
			return nil, object.Errorf(i.typeErr, "os.dup2() arguments must be int")
		}
		if err := osDup2(int(oldfd), int(newfd)); err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return object.NewInt(int64(newfd)), nil
	}})
}

func osSyscallStatResult(i *Interp, sys *syscall.Stat_t) object.Object {
	inst := &object.Instance{Class: statResultClass, Dict: object.NewDict()}
	inst.Dict.SetStr("st_mode", object.NewInt(int64(sys.Mode)))
	inst.Dict.SetStr("st_ino", object.NewInt(int64(sys.Ino)))
	inst.Dict.SetStr("st_dev", object.NewInt(int64(sys.Dev)))
	inst.Dict.SetStr("st_nlink", object.NewInt(int64(sys.Nlink)))
	inst.Dict.SetStr("st_uid", object.NewInt(int64(sys.Uid)))
	inst.Dict.SetStr("st_gid", object.NewInt(int64(sys.Gid)))
	inst.Dict.SetStr("st_size", object.NewInt(sys.Size))
	atimeSec, atimeNsec := statAtime(sys)
	ctimeSec, ctimeNsec := statCtime(sys)
	mtimeSec, mtimeNsec := statMtime(sys)
	inst.Dict.SetStr("st_atime", &object.Float{V: float64(atimeSec) + float64(atimeNsec)/1e9})
	inst.Dict.SetStr("st_mtime", &object.Float{V: float64(mtimeSec) + float64(mtimeNsec)/1e9})
	inst.Dict.SetStr("st_ctime", &object.Float{V: float64(ctimeSec) + float64(ctimeNsec)/1e9})
	return inst
}

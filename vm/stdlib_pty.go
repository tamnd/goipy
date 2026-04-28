package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildPty() *object.Module {
	m := &object.Module{Name: "pty", Dict: object.NewDict()}
	d := m.Dict

	d.SetStr("CHILD", intObj(0))
	d.SetStr("STDIN_FILENO", intObj(0))
	d.SetStr("STDOUT_FILENO", intObj(1))
	d.SetStr("STDERR_FILENO", intObj(2))

	osErrStub := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return nil, object.Errorf(i.osErr, "[Errno 38] Function not implemented")
			},
		}
	}
	noneStub := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		}
	}

	// own functions -- all require a real pty device so raise in stub context
	d.SetStr("openpty", osErrStub("openpty"))
	d.SetStr("fork", osErrStub("fork"))
	d.SetStr("spawn", osErrStub("spawn"))

	// re-exports from os
	d.SetStr("close", noneStub("close"))
	d.SetStr("waitpid", &object.BuiltinFunc{
		Name: "waitpid",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Tuple{V: []object.Object{intObj(0), intObj(0)}}, nil
		},
	})

	// re-exports from tty -- raise on non-tty fd
	d.SetStr("setraw", &object.BuiltinFunc{
		Name: "setraw",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "[Errno 25] Inappropriate ioctl for device")
		},
	})
	d.SetStr("tcgetattr", &object.BuiltinFunc{
		Name: "tcgetattr",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "[Errno 25] Inappropriate ioctl for device")
		},
	})
	d.SetStr("tcsetattr", noneStub("tcsetattr"))

	return m
}

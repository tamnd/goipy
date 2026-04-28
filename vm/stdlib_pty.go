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

	errStub := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return nil, object.Errorf(i.osErr, "[Errno 38] Function not implemented")
			},
		}
	}

	d.SetStr("openpty", errStub("openpty"))
	d.SetStr("fork", errStub("fork"))
	d.SetStr("spawn", errStub("spawn"))

	return m
}

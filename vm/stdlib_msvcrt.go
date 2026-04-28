package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildMsvcrt() *object.Module {
	m := &object.Module{Name: "msvcrt", Dict: object.NewDict()}
	d := m.Dict

	d.SetStr("CRT_ASSEMBLY_VERSION", &object.Str{V: "14.0.0.0"})
	d.SetStr("LK_UNLCK", intObj(0))
	d.SetStr("LK_LOCK", intObj(1))
	d.SetStr("LK_NBLCK", intObj(2))
	d.SetStr("LK_RLCK", intObj(3))
	d.SetStr("LK_NBRLCK", intObj(4))
	d.SetStr("SEM_FAILCRITICALERRORS", intObj(0x0001))
	d.SetStr("SEM_NOALIGNMENTFAULTEXCEPT", intObj(0x0004))
	d.SetStr("SEM_NOGPFAULTERRORBOX", intObj(0x0002))
	d.SetStr("SEM_NOOPENFILEERRORBOX", intObj(0x8000))

	noneStub := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		}
	}
	intStub := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return intObj(0), nil
			},
		}
	}

	d.SetStr("locking", noneStub("locking"))
	d.SetStr("setmode", intStub("setmode"))
	d.SetStr("open_osfhandle", intStub("open_osfhandle"))
	d.SetStr("get_osfhandle", intStub("get_osfhandle"))
	d.SetStr("kbhit", &object.BuiltinFunc{
		Name: "kbhit",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		},
	})
	d.SetStr("getch", &object.BuiltinFunc{
		Name: "getch",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Bytes{V: []byte{}}, nil
		},
	})
	d.SetStr("getwch", &object.BuiltinFunc{
		Name: "getwch",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: ""}, nil
		},
	})
	d.SetStr("getche", &object.BuiltinFunc{
		Name: "getche",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Bytes{V: []byte{}}, nil
		},
	})
	d.SetStr("getwche", &object.BuiltinFunc{
		Name: "getwche",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: ""}, nil
		},
	})
	d.SetStr("putch", noneStub("putch"))
	d.SetStr("putwch", noneStub("putwch"))
	d.SetStr("ungetch", noneStub("ungetch"))
	d.SetStr("ungetwch", noneStub("ungetwch"))
	d.SetStr("heapmin", noneStub("heapmin"))
	d.SetStr("SetErrorMode", intStub("SetErrorMode"))
	d.SetStr("GetErrorMode", intStub("GetErrorMode"))

	return m
}

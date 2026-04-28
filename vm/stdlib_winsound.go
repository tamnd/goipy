package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildWinsound() *object.Module {
	m := &object.Module{Name: "winsound", Dict: object.NewDict()}
	d := m.Dict

	d.SetStr("SND_APPLICATION", intObj(128))
	d.SetStr("SND_FILENAME", intObj(131072))
	d.SetStr("SND_ALIAS", intObj(65536))
	d.SetStr("SND_LOOP", intObj(8))
	d.SetStr("SND_MEMORY", intObj(4))
	d.SetStr("SND_PURGE", intObj(64))
	d.SetStr("SND_ASYNC", intObj(1))
	d.SetStr("SND_NODEFAULT", intObj(2))
	d.SetStr("SND_NOSTOP", intObj(16))
	d.SetStr("SND_NOWAIT", intObj(8192))
	d.SetStr("SND_SENTRY", intObj(524288))
	d.SetStr("SND_SYNC", intObj(0))
	d.SetStr("SND_SYSTEM", intObj(2097152))

	d.SetStr("MB_ICONASTERISK", intObj(64))
	d.SetStr("MB_ICONEXCLAMATION", intObj(48))
	d.SetStr("MB_ICONHAND", intObj(16))
	d.SetStr("MB_ICONQUESTION", intObj(32))
	d.SetStr("MB_OK", intObj(0))
	d.SetStr("MB_ICONERROR", intObj(16))
	d.SetStr("MB_ICONINFORMATION", intObj(64))
	d.SetStr("MB_ICONSTOP", intObj(16))
	d.SetStr("MB_ICONWARNING", intObj(48))

	noneStub := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		}
	}

	d.SetStr("Beep", noneStub("Beep"))
	d.SetStr("PlaySound", noneStub("PlaySound"))
	d.SetStr("MessageBeep", noneStub("MessageBeep"))

	return m
}

package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildTty() *object.Module {
	m := &object.Module{Name: "tty", Dict: object.NewDict()}
	d := m.Dict

	for name, val := range termiosConstants() {
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

	d.SetStr("setraw", noneStub("setraw"))
	d.SetStr("setcbreak", noneStub("setcbreak"))

	d.SetStr("tcgetattr", &object.BuiltinFunc{
		Name: "tcgetattr",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			cc := make([]object.Object, 20)
			for k := range cc {
				cc[k] = intObj(0)
			}
			return &object.List{V: []object.Object{
				intObj(0), intObj(0), intObj(0), intObj(0),
				intObj(9600), intObj(9600),
				&object.List{V: cc},
			}}, nil
		},
	})
	d.SetStr("tcsetattr", noneStub("tcsetattr"))
	d.SetStr("tcdrain", noneStub("tcdrain"))
	d.SetStr("tcflush", noneStub("tcflush"))
	d.SetStr("tcflow", noneStub("tcflow"))
	d.SetStr("tcsendbreak", noneStub("tcsendbreak"))

	return m
}

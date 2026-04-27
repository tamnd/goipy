package vm

import (
	"github.com/tamnd/goipy/object"
)

const ensurepipVersion = "24.0"

func (i *Interp) buildEnsurepip() *object.Module {
	m := &object.Module{Name: "ensurepip", Dict: object.NewDict()}

	m.Dict.SetStr("version", &object.BuiltinFunc{
		Name: "version",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: ensurepipVersion}, nil
		},
	})

	m.Dict.SetStr("bootstrap", &object.BuiltinFunc{
		Name: "bootstrap",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			altinstall := false
			defaultPip := false
			if kw != nil {
				if v, ok := kw.GetStr("altinstall"); ok {
					if b, ok2 := v.(*object.Bool); ok2 {
						altinstall = b.V
					}
				}
				if v, ok := kw.GetStr("default_pip"); ok {
					if b, ok2 := v.(*object.Bool); ok2 {
						defaultPip = b.V
					}
				}
			}
			if altinstall && defaultPip {
				return nil, object.Errorf(i.valueErr, "altinstall and default_pip are mutually exclusive")
			}
			return object.None, nil
		},
	})

	m.Dict.SetStr("_bundled_packages", &object.BuiltinFunc{
		Name: "_bundled_packages",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			d := object.NewDict()
			d.SetStr("pip", &object.Str{V: ensurepipVersion})
			return d, nil
		},
	})

	return m
}

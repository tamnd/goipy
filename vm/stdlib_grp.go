package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildGrp() *object.Module {
	m := &object.Module{Name: "grp", Dict: object.NewDict()}
	d := m.Dict

	groupCls := &object.Class{
		Name:  "struct_group",
		Bases: []*object.Class{},
		Dict:  object.NewDict(),
	}
	groupCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			fields := []string{"gr_name", "gr_passwd", "gr_gid", "gr_mem"}
			for idx, f := range fields {
				if idx+1 < len(a) {
					inst.Dict.SetStr(f, a[idx+1])
				} else {
					inst.Dict.SetStr(f, object.None)
				}
			}
			return object.None, nil
		},
	})
	d.SetStr("struct_group", groupCls)

	notFound := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return nil, object.Errorf(i.keyErr, "%s(): group not found", name)
			},
		}
	}

	d.SetStr("getgrgid", notFound("getgrgid"))
	d.SetStr("getgrnam", notFound("getgrnam"))
	d.SetStr("getgrall", &object.BuiltinFunc{
		Name: "getgrall",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	return m
}

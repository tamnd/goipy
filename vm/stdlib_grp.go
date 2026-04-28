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

	mkGroup := func(name, passwd string, gid int, mem []string) *object.Instance {
		inst := &object.Instance{Class: groupCls, Dict: object.NewDict()}
		inst.Dict.SetStr("gr_name", &object.Str{V: name})
		inst.Dict.SetStr("gr_passwd", &object.Str{V: passwd})
		inst.Dict.SetStr("gr_gid", intObj(int64(gid)))
		members := make([]object.Object, len(mem))
		for idx, s := range mem {
			members[idx] = &object.Str{V: s}
		}
		inst.Dict.SetStr("gr_mem", &object.List{V: members})
		return inst
	}

	// stub database: gid 0 = wheel (mirrors macOS)
	wheelEntry := mkGroup("wheel", "*", 0, []string{"root"})

	d.SetStr("getgrgid", &object.BuiltinFunc{
		Name: "getgrgid",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) > 0 {
				if n, ok := a[0].(*object.Int); ok && n.V.Int64() == 0 {
					return mkGroup("wheel", "*", 0, []string{"root"}), nil
				}
			}
			return nil, object.Errorf(i.keyErr, "getgrgid(): gid not found")
		},
	})

	d.SetStr("getgrnam", &object.BuiltinFunc{
		Name: "getgrnam",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) > 0 {
				if s, ok := a[0].(*object.Str); ok && s.V == "wheel" {
					return mkGroup("wheel", "*", 0, []string{"root"}), nil
				}
			}
			return nil, object.Errorf(i.keyErr, "getgrnam(): group not found")
		},
	})

	d.SetStr("getgrall", &object.BuiltinFunc{
		Name: "getgrall",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{wheelEntry}}, nil
		},
	})

	return m
}

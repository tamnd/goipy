package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildPwd() *object.Module {
	m := &object.Module{Name: "pwd", Dict: object.NewDict()}
	d := m.Dict

	passwdCls := &object.Class{
		Name:  "struct_passwd",
		Bases: []*object.Class{},
		Dict:  object.NewDict(),
	}
	passwdCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			fields := []string{"pw_name", "pw_passwd", "pw_uid", "pw_gid", "pw_gecos", "pw_dir", "pw_shell"}
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
	d.SetStr("struct_passwd", passwdCls)

	mkPasswd := func(name, passwd string, uid, gid int, gecos, dir, shell string) *object.Instance {
		inst := &object.Instance{Class: passwdCls, Dict: object.NewDict()}
		inst.Dict.SetStr("pw_name", &object.Str{V: name})
		inst.Dict.SetStr("pw_passwd", &object.Str{V: passwd})
		inst.Dict.SetStr("pw_uid", intObj(int64(uid)))
		inst.Dict.SetStr("pw_gid", intObj(int64(gid)))
		inst.Dict.SetStr("pw_gecos", &object.Str{V: gecos})
		inst.Dict.SetStr("pw_dir", &object.Str{V: dir})
		inst.Dict.SetStr("pw_shell", &object.Str{V: shell})
		return inst
	}

	d.SetStr("getpwuid", &object.BuiltinFunc{
		Name: "getpwuid",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) > 0 {
				if n, ok := a[0].(*object.Int); ok && n.V.Int64() == 0 {
					return mkPasswd("root", "*", 0, 0, "System Administrator", "/var/root", "/bin/sh"), nil
				}
			}
			return nil, object.Errorf(i.keyErr, "getpwuid(): uid not found")
		},
	})

	d.SetStr("getpwnam", &object.BuiltinFunc{
		Name: "getpwnam",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.keyErr, "getpwnam(): name not found")
		},
	})

	d.SetStr("getpwall", &object.BuiltinFunc{
		Name: "getpwall",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	return m
}

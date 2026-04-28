package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildMsilib() *object.Module {
	m := &object.Module{Name: "msilib", Dict: object.NewDict()}
	d := m.Dict

	// constants
	d.SetStr("AMD64", object.False)
	d.SetStr("Win64", object.False)
	d.SetStr("datasizemask", intObj(0x00FF))
	d.SetStr("type_valid", intObj(0x0100))
	d.SetStr("type_localizable", intObj(0x0200))
	d.SetStr("typemask", intObj(0x0C00))
	d.SetStr("type_long", intObj(0x0000))
	d.SetStr("type_short", intObj(0x0400))
	d.SetStr("type_string", intObj(0x0C00))
	d.SetStr("type_binary", intObj(0x0800))
	d.SetStr("type_nullable", intObj(0x1000))
	d.SetStr("type_key", intObj(0x2000))
	d.SetStr("knownbits", intObj(0x3FFF))

	mkStubClass := func(name string) *object.Class {
		cls := &object.Class{
			Name:  name,
			Bases: []*object.Class{},
			Dict:  object.NewDict(),
		}
		cls.Dict.SetStr("__init__", &object.BuiltinFunc{
			Name: "__init__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		})
		return cls
	}

	d.SetStr("Table", mkStubClass("Table"))
	d.SetStr("CAB", mkStubClass("CAB"))
	d.SetStr("Directory", mkStubClass("Directory"))
	d.SetStr("Binary", mkStubClass("Binary"))
	d.SetStr("Feature", mkStubClass("Feature"))
	d.SetStr("Control", mkStubClass("Control"))
	d.SetStr("RadioButtonGroup", mkStubClass("RadioButtonGroup"))
	d.SetStr("Dialog", mkStubClass("Dialog"))
	d.SetStr("_Unspecified", mkStubClass("_Unspecified"))

	noneStub := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			},
		}
	}

	d.SetStr("change_sequence", noneStub("change_sequence"))
	d.SetStr("add_data", noneStub("add_data"))
	d.SetStr("add_stream", noneStub("add_stream"))
	d.SetStr("init_database", noneStub("init_database"))
	d.SetStr("add_tables", noneStub("add_tables"))

	d.SetStr("make_id", &object.BuiltinFunc{
		Name: "make_id",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			a = mpArgs(a)
			if len(a) > 0 {
				return a[0], nil
			}
			return &object.Str{V: ""}, nil
		},
	})

	d.SetStr("gen_uuid", &object.BuiltinFunc{
		Name: "gen_uuid",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "{00000000-0000-0000-0000-000000000000}"}, nil
		},
	})

	return m
}

package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildWinreg() *object.Module {
	m := &object.Module{Name: "winreg", Dict: object.NewDict()}
	d := m.Dict

	// HKEY root constants (unsigned 32-bit values stored as Python ints)
	d.SetStr("HKEY_CLASSES_ROOT", intObj(2147483648))
	d.SetStr("HKEY_CURRENT_USER", intObj(2147483649))
	d.SetStr("HKEY_LOCAL_MACHINE", intObj(2147483650))
	d.SetStr("HKEY_USERS", intObj(2147483651))
	d.SetStr("HKEY_PERFORMANCE_DATA", intObj(2147483652))
	d.SetStr("HKEY_CURRENT_CONFIG", intObj(2147483653))
	d.SetStr("HKEY_DYN_DATA", intObj(2147483654))

	// KEY access constants
	d.SetStr("KEY_ALL_ACCESS", intObj(983103))
	d.SetStr("KEY_WRITE", intObj(131078))
	d.SetStr("KEY_READ", intObj(131097))
	d.SetStr("KEY_EXECUTE", intObj(131097))
	d.SetStr("KEY_QUERY_VALUE", intObj(1))
	d.SetStr("KEY_SET_VALUE", intObj(2))
	d.SetStr("KEY_CREATE_SUB_KEY", intObj(4))
	d.SetStr("KEY_ENUMERATE_SUB_KEYS", intObj(8))
	d.SetStr("KEY_NOTIFY", intObj(16))
	d.SetStr("KEY_CREATE_LINK", intObj(32))
	d.SetStr("KEY_WOW64_64KEY", intObj(256))
	d.SetStr("KEY_WOW64_32KEY", intObj(512))

	// REG value type constants (documented)
	d.SetStr("REG_NONE", intObj(0))
	d.SetStr("REG_SZ", intObj(1))
	d.SetStr("REG_EXPAND_SZ", intObj(2))
	d.SetStr("REG_BINARY", intObj(3))
	d.SetStr("REG_DWORD", intObj(4))
	d.SetStr("REG_DWORD_LITTLE_ENDIAN", intObj(4))
	d.SetStr("REG_DWORD_BIG_ENDIAN", intObj(5))
	d.SetStr("REG_LINK", intObj(6))
	d.SetStr("REG_MULTI_SZ", intObj(7))
	d.SetStr("REG_RESOURCE_LIST", intObj(8))
	d.SetStr("REG_FULL_RESOURCE_DESCRIPTOR", intObj(9))
	d.SetStr("REG_RESOURCE_REQUIREMENTS_LIST", intObj(10))
	d.SetStr("REG_QWORD", intObj(11))
	d.SetStr("REG_QWORD_LITTLE_ENDIAN", intObj(11))

	// REG undocumented constants (present on real Windows and in typeshed)
	d.SetStr("REG_CREATED_NEW_KEY", intObj(1))
	d.SetStr("REG_LEGAL_CHANGE_FILTER", intObj(268435471))
	d.SetStr("REG_LEGAL_OPTION", intObj(31))
	d.SetStr("REG_NOTIFY_CHANGE_ATTRIBUTES", intObj(2))
	d.SetStr("REG_NOTIFY_CHANGE_LAST_SET", intObj(4))
	d.SetStr("REG_NOTIFY_CHANGE_NAME", intObj(1))
	d.SetStr("REG_NOTIFY_CHANGE_SECURITY", intObj(8))
	d.SetStr("REG_NO_LAZY_FLUSH", intObj(4))
	d.SetStr("REG_OPENED_EXISTING_KEY", intObj(2))
	d.SetStr("REG_OPTION_BACKUP_RESTORE", intObj(4))
	d.SetStr("REG_OPTION_CREATE_LINK", intObj(2))
	d.SetStr("REG_OPTION_NON_VOLATILE", intObj(0))
	d.SetStr("REG_OPTION_OPEN_LINK", intObj(8))
	d.SetStr("REG_OPTION_RESERVED", intObj(0))
	d.SetStr("REG_OPTION_VOLATILE", intObj(1))
	d.SetStr("REG_REFRESH_HIVE", intObj(2))
	d.SetStr("REG_WHOLE_HIVE_VOLATILE", intObj(1))

	// error = OSError
	d.SetStr("error", i.osErr)

	// HKEYType class
	hkeyCls := &object.Class{
		Name:  "HKEYType",
		Bases: []*object.Class{},
		Dict:  object.NewDict(),
	}
	hkeyCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if inst, ok := a[0].(*object.Instance); ok {
				inst.Dict.SetStr("handle", intObj(0))
			}
			return object.None, nil
		},
	})
	hkeyCls.Dict.SetStr("Close", &object.BuiltinFunc{
		Name: "Close",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	hkeyCls.Dict.SetStr("Detach", &object.BuiltinFunc{
		Name: "Detach",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj(0), nil
		},
	})
	hkeyCls.Dict.SetStr("__bool__", &object.BuiltinFunc{
		Name: "__bool__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		},
	})
	hkeyCls.Dict.SetStr("__int__", &object.BuiltinFunc{
		Name: "__int__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj(0), nil
		},
	})
	hkeyCls.Dict.SetStr("__hash__", &object.BuiltinFunc{
		Name: "__hash__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj(0), nil
		},
	})
	hkeyCls.Dict.SetStr("__enter__", &object.BuiltinFunc{
		Name: "__enter__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 {
				return a[0], nil
			}
			return object.None, nil
		},
	})
	hkeyCls.Dict.SetStr("__exit__", &object.BuiltinFunc{
		Name: "__exit__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	d.SetStr("HKEYType", hkeyCls)

	// All registry functions raise OSError (Windows registry not available)
	osErrStub := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return nil, object.Errorf(i.osErr, "%s: Windows registry not available", name)
			},
		}
	}

	for _, name := range []string{
		"CloseKey", "ConnectRegistry", "CreateKey", "CreateKeyEx",
		"DeleteKey", "DeleteKeyEx", "DeleteValue", "EnumKey", "EnumValue",
		"ExpandEnvironmentStrings", "FlushKey", "LoadKey",
		"OpenKey", "OpenKeyEx", "QueryInfoKey", "QueryValue", "QueryValueEx",
		"SaveKey", "SetValue", "SetValueEx",
		"DisableReflectionKey", "EnableReflectionKey", "QueryReflectionKey",
	} {
		d.SetStr(name, osErrStub(name))
	}

	return m
}

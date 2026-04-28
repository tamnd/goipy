package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildSyslog() *object.Module {
	m := &object.Module{Name: "syslog", Dict: object.NewDict()}
	d := m.Dict

	consts := map[string]int64{
		// priority levels
		"LOG_EMERG":   0,
		"LOG_ALERT":   1,
		"LOG_CRIT":    2,
		"LOG_ERR":     3,
		"LOG_WARNING": 4,
		"LOG_NOTICE":  5,
		"LOG_INFO":    6,
		"LOG_DEBUG":   7,
		// facilities
		"LOG_KERN":       0,
		"LOG_USER":       8,
		"LOG_MAIL":       16,
		"LOG_DAEMON":     24,
		"LOG_AUTH":       32,
		"LOG_SYSLOG":     40,
		"LOG_LPR":        48,
		"LOG_NEWS":       56,
		"LOG_UUCP":       64,
		"LOG_CRON":       72,
		"LOG_AUTHPRIV":   80,
		"LOG_FTP":        88,
		"LOG_NETINFO":    96,
		"LOG_REMOTEAUTH": 104,
		"LOG_INSTALL":    112,
		"LOG_RAS":        120,
		"LOG_LOCAL0":     128,
		"LOG_LOCAL1":     136,
		"LOG_LOCAL2":     144,
		"LOG_LOCAL3":     152,
		"LOG_LOCAL4":     160,
		"LOG_LOCAL5":     168,
		"LOG_LOCAL6":     176,
		"LOG_LOCAL7":     184,
		"LOG_LAUNCHD":    192,
		// openlog options
		"LOG_PID":    1,
		"LOG_CONS":   2,
		"LOG_ODELAY": 4,
		"LOG_NDELAY": 8,
		"LOG_NOWAIT": 16,
		"LOG_PERROR": 32,
	}
	for name, val := range consts {
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

	d.SetStr("syslog", noneStub("syslog"))
	d.SetStr("openlog", noneStub("openlog"))
	d.SetStr("closelog", noneStub("closelog"))

	d.SetStr("setlogmask", &object.BuiltinFunc{
		Name: "setlogmask",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return intObj(0), nil
		},
	})

	d.SetStr("LOG_MASK", &object.BuiltinFunc{
		Name: "LOG_MASK",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 {
				if n, ok := a[0].(*object.Int); ok {
					v := n.V.Int64()
					return intObj(1 << v), nil
				}
			}
			return intObj(0), nil
		},
	})

	d.SetStr("LOG_UPTO", &object.BuiltinFunc{
		Name: "LOG_UPTO",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 {
				if n, ok := a[0].(*object.Int); ok {
					v := n.V.Int64()
					return intObj((1<<(v+1)) - 1), nil
				}
			}
			return intObj(0), nil
		},
	})

	return m
}

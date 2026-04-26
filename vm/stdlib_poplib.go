package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildPoplib constructs the poplib module matching CPython 3.14's API
// surface. No real POP3 connections are made; all network methods are stubs.
func (i *Interp) buildPoplib() *object.Module {
	m := &object.Module{Name: "poplib", Dict: object.NewDict()}

	// ── Constants ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("POP3_PORT", object.NewInt(110))
	m.Dict.SetStr("POP3_SSL_PORT", object.NewInt(995))

	// ── Exception ─────────────────────────────────────────────────────────────

	errorProtoCls := &object.Class{Name: "error_proto", Dict: object.NewDict(), Bases: []*object.Class{i.exception}}
	m.Dict.SetStr("error_proto", errorProtoCls)

	// ── POP3 class ────────────────────────────────────────────────────────────

	pop3Cls := &object.Class{Name: "POP3", Dict: object.NewDict()}

	noop := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}
	}

	pop3Cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		host := ""
		if len(a) > 1 {
			if s, ok2 := a[1].(*object.Str); ok2 {
				host = s.V
			}
		} else if kw != nil {
			if v, ok2 := kw.GetStr("host"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					host = s.V
				}
			}
		}
		port := object.Object(object.NewInt(110))
		if len(a) > 2 {
			port = a[2]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("port"); ok2 {
				port = v
			}
		}
		timeout := object.Object(object.None)
		if len(a) > 3 {
			timeout = a[3]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("timeout"); ok2 {
				timeout = v
			}
		}
		inst.Dict.SetStr("host", &object.Str{V: host})
		inst.Dict.SetStr("port", port)
		inst.Dict.SetStr("timeout", timeout)
		inst.Dict.SetStr("sock", object.None)
		inst.Dict.SetStr("file", object.None)
		inst.Dict.SetStr("welcome", object.None)
		inst.Dict.SetStr("debugging", object.NewInt(0))
		inst.Dict.SetStr("timestamp", &object.Str{V: ""})
		return object.None, nil
	}})

	pop3Cls.Dict.SetStr("set_debuglevel", &object.BuiltinFunc{Name: "set_debuglevel", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		if inst, ok := a[0].(*object.Instance); ok {
			inst.Dict.SetStr("debugging", a[1])
		}
		return object.None, nil
	}})

	pop3Cls.Dict.SetStr("getwelcome", &object.BuiltinFunc{Name: "getwelcome", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		if v, ok2 := inst.Dict.GetStr("welcome"); ok2 {
			return v, nil
		}
		return object.None, nil
	}})

	// user(username) → '+OK'
	pop3Cls.Dict.SetStr("user", &object.BuiltinFunc{Name: "user", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "+OK"}, nil
	}})

	// pass_(pswd) → ('+OK', [], 0)
	pop3Cls.Dict.SetStr("pass_", &object.BuiltinFunc{Name: "pass_", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "+OK"},
			&object.List{V: nil},
			object.NewInt(0),
		}}, nil
	}})

	// stat() → (message_count, mailbox_size)
	pop3Cls.Dict.SetStr("stat", &object.BuiltinFunc{Name: "stat", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{object.NewInt(0), object.NewInt(0)}}, nil
	}})

	// list(which=None) → (response, ['mesg_num octets', ...], octets)
	pop3Cls.Dict.SetStr("list", &object.BuiltinFunc{Name: "list", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "+OK 0 messages"},
			&object.List{V: nil},
			object.NewInt(0),
		}}, nil
	}})

	// retr(which) → (response, ['line', ...], octets)
	pop3Cls.Dict.SetStr("retr", &object.BuiltinFunc{Name: "retr", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "+OK"},
			&object.List{V: nil},
			object.NewInt(0),
		}}, nil
	}})

	// top(which, howmuch) → (response, ['line', ...], octets)
	pop3Cls.Dict.SetStr("top", &object.BuiltinFunc{Name: "top", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "+OK"},
			&object.List{V: nil},
			object.NewInt(0),
		}}, nil
	}})

	// dele(which) → '+OK'
	pop3Cls.Dict.SetStr("dele", &object.BuiltinFunc{Name: "dele", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "+OK"}, nil
	}})

	// noop() → '+OK'
	pop3Cls.Dict.SetStr("noop", &object.BuiltinFunc{Name: "noop", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "+OK"}, nil
	}})

	// rset() → '+OK'
	pop3Cls.Dict.SetStr("rset", &object.BuiltinFunc{Name: "rset", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "+OK"}, nil
	}})

	// quit() → '+OK Goodbye'
	pop3Cls.Dict.SetStr("quit", &object.BuiltinFunc{Name: "quit", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "+OK Goodbye"}, nil
	}})

	// uidl(which=None) → (response, ['mesgnum uid', ...], octets)
	pop3Cls.Dict.SetStr("uidl", &object.BuiltinFunc{Name: "uidl", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{
			&object.Str{V: "+OK"},
			&object.List{V: nil},
			object.NewInt(0),
		}}, nil
	}})

	// apop(user, secret) → '+OK'
	pop3Cls.Dict.SetStr("apop", &object.BuiltinFunc{Name: "apop", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "+OK"}, nil
	}})

	// rpop(user) → '+OK'
	pop3Cls.Dict.SetStr("rpop", &object.BuiltinFunc{Name: "rpop", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "+OK"}, nil
	}})

	// capa() → {} (empty dict of capabilities)
	pop3Cls.Dict.SetStr("capa", &object.BuiltinFunc{Name: "capa", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewDict(), nil
	}})

	// utf8() → '+OK'
	pop3Cls.Dict.SetStr("utf8", &object.BuiltinFunc{Name: "utf8", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "+OK"}, nil
	}})

	// stls(context=None) → '+OK'
	pop3Cls.Dict.SetStr("stls", &object.BuiltinFunc{Name: "stls", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "+OK"}, nil
	}})

	// close() — no-op
	pop3Cls.Dict.SetStr("close", noop("close"))

	// __enter__ / __exit__ context manager
	pop3Cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			return a[0], nil
		}
		return object.None, nil
	}})
	pop3Cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	m.Dict.SetStr("POP3", pop3Cls)

	// ── POP3_SSL ──────────────────────────────────────────────────────────────

	pop3SSLCls := &object.Class{Name: "POP3_SSL", Bases: []*object.Class{pop3Cls}, Dict: object.NewDict()}
	pop3SSLCls.Dict.SetStr("default_port", object.NewInt(995))
	m.Dict.SetStr("POP3_SSL", pop3SSLCls)

	return m
}

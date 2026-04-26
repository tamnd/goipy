package vm

import (
	"strings"

	"github.com/tamnd/goipy/object"
)

// buildSmtplib constructs the smtplib module matching CPython 3.14's API.
// No real SMTP connections are made; all network methods are stubs.
func (i *Interp) buildSmtplib() *object.Module {
	m := &object.Module{Name: "smtplib", Dict: object.NewDict()}

	// ── Constants ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("SMTP_PORT", object.NewInt(25))
	m.Dict.SetStr("SMTP_SSL_PORT", object.NewInt(465))
	m.Dict.SetStr("LMTP_PORT", object.NewInt(2003))

	// ── Exception hierarchy ───────────────────────────────────────────────────

	mkExc := func(name string, bases ...*object.Class) *object.Class {
		return &object.Class{Name: name, Dict: object.NewDict(), Bases: bases}
	}

	// SMTPException(OSError)
	smtpExcCls := mkExc("SMTPException", i.osErr)
	m.Dict.SetStr("SMTPException", smtpExcCls)

	// SMTPServerDisconnected(SMTPException)
	smtpServerDiscCls := mkExc("SMTPServerDisconnected", smtpExcCls)
	m.Dict.SetStr("SMTPServerDisconnected", smtpServerDiscCls)

	// SMTPNotSupportedError(SMTPException)
	smtpNotSuppCls := mkExc("SMTPNotSupportedError", smtpExcCls)
	m.Dict.SetStr("SMTPNotSupportedError", smtpNotSuppCls)

	// SMTPRecipientsRefused(SMTPException)  — .recipients = Args[0]
	smtpRecipRefCls := mkExc("SMTPRecipientsRefused", smtpExcCls)
	smtpRecipRefCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		name, ok := a[1].(*object.Str)
		if !ok {
			return object.None, nil
		}
		exc, ok2 := a[0].(*object.Exception)
		if !ok2 {
			return object.None, nil
		}
		if name.V == "recipients" {
			if exc.Args != nil && len(exc.Args.V) > 0 {
				return exc.Args.V[0], nil
			}
			return object.NewDict(), nil
		}
		return nil, object.Errorf(i.attrErr, "'SMTPRecipientsRefused' object has no attribute '%s'", name.V)
	}})
	m.Dict.SetStr("SMTPRecipientsRefused", smtpRecipRefCls)

	// SMTPResponseException(SMTPException)  — .smtp_code = Args[0], .smtp_error = Args[1]
	smtpRespExcCls := mkExc("SMTPResponseException", smtpExcCls)
	smtpRespExcCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		name, ok := a[1].(*object.Str)
		if !ok {
			return object.None, nil
		}
		exc, ok2 := a[0].(*object.Exception)
		if !ok2 {
			return object.None, nil
		}
		switch name.V {
		case "smtp_code":
			if exc.Args != nil && len(exc.Args.V) > 0 {
				return exc.Args.V[0], nil
			}
			return object.NewInt(0), nil
		case "smtp_error":
			if exc.Args != nil && len(exc.Args.V) > 1 {
				return exc.Args.V[1], nil
			}
			return &object.Bytes{V: nil}, nil
		}
		return nil, object.Errorf(i.attrErr, "'SMTPResponseException' object has no attribute '%s'", name.V)
	}})
	m.Dict.SetStr("SMTPResponseException", smtpRespExcCls)

	// SMTPSenderRefused(SMTPResponseException)  — also .sender = Args[2]
	smtpSenderRefCls := mkExc("SMTPSenderRefused", smtpRespExcCls)
	smtpSenderRefCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		name, ok := a[1].(*object.Str)
		if !ok {
			return object.None, nil
		}
		exc, ok2 := a[0].(*object.Exception)
		if !ok2 {
			return object.None, nil
		}
		switch name.V {
		case "smtp_code":
			if exc.Args != nil && len(exc.Args.V) > 0 {
				return exc.Args.V[0], nil
			}
			return object.NewInt(0), nil
		case "smtp_error":
			if exc.Args != nil && len(exc.Args.V) > 1 {
				return exc.Args.V[1], nil
			}
			return &object.Bytes{V: nil}, nil
		case "sender":
			if exc.Args != nil && len(exc.Args.V) > 2 {
				return exc.Args.V[2], nil
			}
			return &object.Str{V: ""}, nil
		}
		return nil, object.Errorf(i.attrErr, "'SMTPSenderRefused' object has no attribute '%s'", name.V)
	}})
	m.Dict.SetStr("SMTPSenderRefused", smtpSenderRefCls)

	// SMTPDataError(SMTPResponseException)
	smtpDataErrCls := mkExc("SMTPDataError", smtpRespExcCls)
	smtpDataErrCls.Dict.SetStr("__getattr__", func() object.Object { v, _ := smtpRespExcCls.Dict.GetStr("__getattr__"); return v }())
	m.Dict.SetStr("SMTPDataError", smtpDataErrCls)

	// SMTPConnectError(SMTPResponseException)
	smtpConnErrCls := mkExc("SMTPConnectError", smtpRespExcCls)
	smtpConnErrCls.Dict.SetStr("__getattr__", func() object.Object { v, _ := smtpRespExcCls.Dict.GetStr("__getattr__"); return v }())
	m.Dict.SetStr("SMTPConnectError", smtpConnErrCls)

	// SMTPHeloError(SMTPResponseException)
	smtpHeloErrCls := mkExc("SMTPHeloError", smtpRespExcCls)
	smtpHeloErrCls.Dict.SetStr("__getattr__", func() object.Object { v, _ := smtpRespExcCls.Dict.GetStr("__getattr__"); return v }())
	m.Dict.SetStr("SMTPHeloError", smtpHeloErrCls)

	// SMTPAuthenticationError(SMTPResponseException)
	smtpAuthErrCls := mkExc("SMTPAuthenticationError", smtpRespExcCls)
	smtpAuthErrCls.Dict.SetStr("__getattr__", func() object.Object { v, _ := smtpRespExcCls.Dict.GetStr("__getattr__"); return v }())
	m.Dict.SetStr("SMTPAuthenticationError", smtpAuthErrCls)

	// ── SMTP class ────────────────────────────────────────────────────────────

	smtpCls := &object.Class{Name: "SMTP", Dict: object.NewDict()}

	noop := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}
	}

	codeResp := func(code int64, msg string) *object.Tuple {
		return &object.Tuple{V: []object.Object{
			object.NewInt(code),
			&object.Bytes{V: []byte(msg)},
		}}
	}

	codeRespFn := func(name string, code int64, msg string) *object.BuiltinFunc {
		resp := codeResp(code, msg)
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return resp, nil
		}}
	}

	smtpCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
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
		port := object.Object(object.NewInt(0))
		if len(a) > 2 {
			port = a[2]
		}
		timeout := object.Object(object.None)
		if len(a) > 4 {
			timeout = a[4]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("timeout"); ok2 {
				timeout = v
			}
		}

		inst.Dict.SetStr("_host", &object.Str{V: host})
		inst.Dict.SetStr("port", port)
		inst.Dict.SetStr("timeout", timeout)
		inst.Dict.SetStr("esmtp_features", object.NewDict())
		inst.Dict.SetStr("does_esmtp", object.False)
		inst.Dict.SetStr("helo_resp", object.None)
		inst.Dict.SetStr("ehlo_resp", object.None)
		inst.Dict.SetStr("debuglevel", object.NewInt(0))
		inst.Dict.SetStr("local_hostname", &object.Str{V: "localhost"})
		return object.None, nil
	}})

	smtpCls.Dict.SetStr("set_debuglevel", &object.BuiltinFunc{Name: "set_debuglevel", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		if inst, ok := a[0].(*object.Instance); ok {
			inst.Dict.SetStr("debuglevel", a[1])
		}
		return object.None, nil
	}})

	smtpCls.Dict.SetStr("has_extn", &object.BuiltinFunc{Name: "has_extn", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.False, nil
		}
		name := ""
		if s, ok2 := a[1].(*object.Str); ok2 {
			name = strings.ToLower(s.V)
		}
		featsObj, ok2 := inst.Dict.GetStr("esmtp_features")
		if !ok2 {
			return object.False, nil
		}
		feats, ok3 := featsObj.(*object.Dict)
		if !ok3 {
			return object.False, nil
		}
		if _, found := feats.GetStr(name); found {
			return object.True, nil
		}
		return object.False, nil
	}})

	smtpCls.Dict.SetStr("close", noop("close"))

	smtpCls.Dict.SetStr("connect", &object.BuiltinFunc{Name: "connect", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			if inst, ok := a[0].(*object.Instance); ok {
				if len(a) > 1 {
					if s, ok2 := a[1].(*object.Str); ok2 {
						inst.Dict.SetStr("_host", s)
					}
				}
			}
		}
		return codeResp(220, "Mock SMTP ready"), nil
	}})

	smtpCls.Dict.SetStr("helo", &object.BuiltinFunc{Name: "helo", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			if inst, ok := a[0].(*object.Instance); ok {
				inst.Dict.SetStr("helo_resp", &object.Bytes{V: []byte("OK")})
			}
		}
		return codeResp(250, "OK"), nil
	}})

	smtpCls.Dict.SetStr("ehlo", &object.BuiltinFunc{Name: "ehlo", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			if inst, ok := a[0].(*object.Instance); ok {
				inst.Dict.SetStr("ehlo_resp", &object.Bytes{V: []byte("OK")})
				inst.Dict.SetStr("does_esmtp", object.True)
			}
		}
		return codeResp(250, "OK"), nil
	}})

	smtpCls.Dict.SetStr("ehlo_or_helo_if_needed", noop("ehlo_or_helo_if_needed"))
	smtpCls.Dict.SetStr("starttls", codeRespFn("starttls", 220, "Ready to start TLS"))
	smtpCls.Dict.SetStr("login", codeRespFn("login", 235, "Authentication successful"))
	smtpCls.Dict.SetStr("auth", &object.BuiltinFunc{Name: "auth", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return codeResp(235, ""), nil
	}})
	smtpCls.Dict.SetStr("auth_cram_md5", noop("auth_cram_md5"))
	smtpCls.Dict.SetStr("auth_plain", noop("auth_plain"))
	smtpCls.Dict.SetStr("auth_login", noop("auth_login"))
	smtpCls.Dict.SetStr("sendmail", &object.BuiltinFunc{Name: "sendmail", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewDict(), nil
	}})
	smtpCls.Dict.SetStr("send_message", &object.BuiltinFunc{Name: "send_message", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewDict(), nil
	}})
	smtpCls.Dict.SetStr("quit", codeRespFn("quit", 221, "Bye"))
	smtpCls.Dict.SetStr("noop", codeRespFn("noop", 250, "OK"))
	smtpCls.Dict.SetStr("rset", codeRespFn("rset", 250, "Flushed"))
	smtpCls.Dict.SetStr("verify", codeRespFn("verify", 250, "OK"))
	smtpCls.Dict.SetStr("expn", codeRespFn("expn", 250, "OK"))
	smtpCls.Dict.SetStr("help", &object.BuiltinFunc{Name: "help", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Bytes{V: []byte("OK")}, nil
	}})
	smtpCls.Dict.SetStr("docmd", codeRespFn("docmd", 250, "OK"))

	// __enter__ / __exit__
	smtpCls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			return a[0], nil
		}
		return object.None, nil
	}})
	smtpCls.Dict.SetStr("__exit__", noop("__exit__"))

	m.Dict.SetStr("SMTP", smtpCls)

	// ── SMTP_SSL ──────────────────────────────────────────────────────────────

	smtpSSLCls := &object.Class{Name: "SMTP_SSL", Bases: []*object.Class{smtpCls}, Dict: object.NewDict()}
	m.Dict.SetStr("SMTP_SSL", smtpSSLCls)

	// ── LMTP ─────────────────────────────────────────────────────────────────

	lmtpCls := &object.Class{Name: "LMTP", Bases: []*object.Class{smtpCls}, Dict: object.NewDict()}
	m.Dict.SetStr("LMTP", lmtpCls)

	return m
}

package vm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tamnd/goipy/object"
)

// buildFtplib constructs the ftplib module matching CPython 3.14's API
// surface. No real FTP connections are made; all network methods are stubs.
func (i *Interp) buildFtplib() *object.Module {
	m := &object.Module{Name: "ftplib", Dict: object.NewDict()}

	// ── Constants ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("FTP_PORT", object.NewInt(21))
	m.Dict.SetStr("MAXLINE", object.NewInt(8192))
	m.Dict.SetStr("MSG_OOB", object.NewInt(1))
	m.Dict.SetStr("CRLF", &object.Str{V: "\r\n"})
	m.Dict.SetStr("B_CRLF", &object.Bytes{V: []byte("\r\n")})

	// ── Exception hierarchy ───────────────────────────────────────────────────

	mkExc := func(name string, bases ...*object.Class) *object.Class {
		return &object.Class{Name: name, Dict: object.NewDict(), Bases: bases}
	}

	errorCls := mkExc("Error", i.exception)
	m.Dict.SetStr("Error", errorCls)

	errorReplyCls := mkExc("error_reply", errorCls)
	m.Dict.SetStr("error_reply", errorReplyCls)

	errorTempCls := mkExc("error_temp", errorCls)
	m.Dict.SetStr("error_temp", errorTempCls)

	errorPermCls := mkExc("error_perm", errorCls)
	m.Dict.SetStr("error_perm", errorPermCls)

	errorProtoCls := mkExc("error_proto", errorCls)
	m.Dict.SetStr("error_proto", errorProtoCls)

	// all_errors = (Error, OSError, EOFError)
	var eofErrCls object.Object = object.None
	if v, ok := i.Builtins.GetStr("EOFError"); ok {
		eofErrCls = v
	}
	allErrors := &object.Tuple{V: []object.Object{errorCls, i.osErr, eofErrCls}}
	m.Dict.SetStr("all_errors", allErrors)

	// ── parse functions ───────────────────────────────────────────────────────

	// parse150(resp) — extracts size from '150 … (N bytes)' or returns None.
	re150 := regexp.MustCompile(`(?i)150 .* \((\d+) bytes\)`)
	m.Dict.SetStr("parse150", &object.BuiltinFunc{Name: "parse150", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "parse150() requires 1 argument")
		}
		resp, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "parse150() argument must be str")
		}
		if !strings.HasPrefix(resp.V, "150") {
			return nil, object.NewException(errorReplyCls, resp.V)
		}
		m := re150.FindStringSubmatch(resp.V)
		if m == nil {
			return object.None, nil
		}
		n, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return object.None, nil
		}
		return object.NewInt(n), nil
	}})

	// parse227(resp) — extracts (host, port) from PASV response.
	re227 := regexp.MustCompile(`(\d+),(\d+),(\d+),(\d+),(\d+),(\d+)`)
	m.Dict.SetStr("parse227", &object.BuiltinFunc{Name: "parse227", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "parse227() requires 1 argument")
		}
		resp, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "parse227() argument must be str")
		}
		if !strings.HasPrefix(resp.V, "227") {
			return nil, object.NewException(errorReplyCls, resp.V)
		}
		match := re227.FindStringSubmatch(resp.V)
		if match == nil {
			return nil, object.NewException(errorProtoCls, resp.V)
		}
		parts := match[1:]
		host := parts[0] + "." + parts[1] + "." + parts[2] + "." + parts[3]
		p1, _ := strconv.ParseInt(parts[4], 10, 64)
		p2, _ := strconv.ParseInt(parts[5], 10, 64)
		port := (p1 << 8) + p2
		return &object.Tuple{V: []object.Object{&object.Str{V: host}, object.NewInt(port)}}, nil
	}})

	// parse229(resp, peer) — extracts (host, port) from EPSV response.
	m.Dict.SetStr("parse229", &object.BuiltinFunc{Name: "parse229", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "parse229() requires 2 arguments")
		}
		resp, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "parse229() first argument must be str")
		}
		if !strings.HasPrefix(resp.V, "229") {
			return nil, object.NewException(errorReplyCls, resp.V)
		}
		s := resp.V
		left := strings.IndexByte(s, '(')
		if left < 0 {
			return nil, object.NewException(errorProtoCls, s)
		}
		right := strings.IndexByte(s[left+1:], ')')
		if right < 0 {
			return nil, object.NewException(errorProtoCls, s)
		}
		right += left + 1
		if s[left+1] != s[right-1] {
			return nil, object.NewException(errorProtoCls, s)
		}
		inner := s[left+1 : right]
		delim := string(inner[0])
		parts := strings.Split(inner, delim)
		if len(parts) != 5 {
			return nil, object.NewException(errorProtoCls, s)
		}
		port, _ := strconv.ParseInt(parts[3], 10, 64)
		// peer[0] is the host
		host := ""
		if peer, ok2 := a[1].(*object.Tuple); ok2 && len(peer.V) > 0 {
			if h, ok3 := peer.V[0].(*object.Str); ok3 {
				host = h.V
			}
		} else if peer, ok2 := a[1].(*object.List); ok2 && len(peer.V) > 0 {
			if h, ok3 := peer.V[0].(*object.Str); ok3 {
				host = h.V
			}
		}
		return &object.Tuple{V: []object.Object{&object.Str{V: host}, object.NewInt(port)}}, nil
	}})

	// parse257(resp) — extracts directory name from PWD/MKD response.
	m.Dict.SetStr("parse257", &object.BuiltinFunc{Name: "parse257", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "parse257() requires 1 argument")
		}
		resp, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "parse257() argument must be str")
		}
		s := resp.V
		if !strings.HasPrefix(s, "257") {
			return nil, object.NewException(errorReplyCls, s)
		}
		if len(s) < 5 || s[3:5] != ` "` {
			return &object.Str{V: ""}, nil
		}
		var dirname strings.Builder
		idx := 5
		n := len(s)
		for idx < n {
			c := s[idx]
			idx++
			if c == '"' {
				if idx >= n || s[idx] != '"' {
					break
				}
				idx++
			}
			dirname.WriteByte(c)
		}
		return &object.Str{V: dirname.String()}, nil
	}})

	// print_line(line) — prints to stdout.
	m.Dict.SetStr("print_line", &object.BuiltinFunc{Name: "print_line", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		line := ""
		if len(a) > 0 {
			line = object.Str_(a[0])
		}
		fmt.Fprintln(i.Stdout, line)
		return object.None, nil
	}})

	// ── FTP class ─────────────────────────────────────────────────────────────

	ftpCls := &object.Class{Name: "FTP", Dict: object.NewDict()}

	// Class-level defaults
	ftpCls.Dict.SetStr("host", &object.Str{V: ""})
	ftpCls.Dict.SetStr("port", object.NewInt(21))
	ftpCls.Dict.SetStr("maxline", object.NewInt(8192))
	ftpCls.Dict.SetStr("sock", object.None)
	ftpCls.Dict.SetStr("file", object.None)
	ftpCls.Dict.SetStr("welcome", object.None)
	ftpCls.Dict.SetStr("passiveserver", object.True)
	ftpCls.Dict.SetStr("trust_server_pasv_ipv4_address", object.False)
	ftpCls.Dict.SetStr("debugging", object.NewInt(0))

	noop := func(name string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}}
	}

	ftpCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst := a[0].(*object.Instance)
		// positional: host, user, passwd, acct, timeout, source_address
		host := ""
		if len(a) > 1 {
			if s, ok := a[1].(*object.Str); ok {
				host = s.V
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("host"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					host = s.V
				}
			}
		}
		timeout := object.Object(object.None)
		if len(a) > 4 {
			timeout = a[4]
		} else if kw != nil {
			if v, ok := kw.GetStr("timeout"); ok {
				timeout = v
			}
		}
		encoding := "utf-8"
		if kw != nil {
			if v, ok := kw.GetStr("encoding"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					encoding = s.V
				}
			}
		}

		inst.Dict.SetStr("host", &object.Str{V: host})
		inst.Dict.SetStr("port", object.NewInt(21))
		inst.Dict.SetStr("timeout", timeout)
		inst.Dict.SetStr("encoding", &object.Str{V: encoding})
		inst.Dict.SetStr("sock", object.None)
		inst.Dict.SetStr("file", object.None)
		inst.Dict.SetStr("welcome", object.None)
		inst.Dict.SetStr("passiveserver", object.True)
		inst.Dict.SetStr("trust_server_pasv_ipv4_address", object.False)
		inst.Dict.SetStr("debugging", object.NewInt(0))
		return object.None, nil
	}})

	ftpCls.Dict.SetStr("set_debuglevel", &object.BuiltinFunc{Name: "set_debuglevel", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		if inst, ok := a[0].(*object.Instance); ok {
			inst.Dict.SetStr("debugging", a[1])
		}
		return object.None, nil
	}})
	// debug is an alias for set_debuglevel
	if fn, ok := ftpCls.Dict.GetStr("set_debuglevel"); ok {
		ftpCls.Dict.SetStr("debug", fn)
	}

	ftpCls.Dict.SetStr("set_pasv", &object.BuiltinFunc{Name: "set_pasv", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		if inst, ok := a[0].(*object.Instance); ok {
			val := a[1]
			if val != object.False && val != object.None {
				inst.Dict.SetStr("passiveserver", object.True)
			} else {
				inst.Dict.SetStr("passiveserver", object.False)
			}
		}
		return object.None, nil
	}})

	ftpCls.Dict.SetStr("sanitize", &object.BuiltinFunc{Name: "sanitize", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return &object.Str{V: "''"}, nil
		}
		s := object.Str_(a[1])
		lower5 := strings.ToLower(s)
		if strings.HasPrefix(lower5, "pass ") {
			trimmed := strings.TrimRight(s, "\r\n")
			s = s[:5] + strings.Repeat("*", len(trimmed)-5) + s[len(trimmed):]
		}
		return &object.Str{V: "'" + s + "'"}, nil
	}})

	ftpCls.Dict.SetStr("getwelcome", &object.BuiltinFunc{Name: "getwelcome", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
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

	// connect(host='', port=0, timeout=-999, source_address=None) — stub
	ftpCls.Dict.SetStr("connect", &object.BuiltinFunc{Name: "connect", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			if inst, ok := a[0].(*object.Instance); ok {
				if len(a) > 1 {
					if s, ok2 := a[1].(*object.Str); ok2 {
						inst.Dict.SetStr("host", s)
					}
				}
				if len(a) > 2 && a[2] != object.None {
					if n, ok2 := toInt64(a[2]); ok2 && n > 0 {
						inst.Dict.SetStr("port", object.NewInt(n))
					}
				}
				inst.Dict.SetStr("welcome", &object.Str{V: "220 Mock FTP server ready"})
			}
		}
		return &object.Str{V: "220 Mock FTP server ready"}, nil
	}})

	// login(user='', passwd='', acct='') — stub
	ftpCls.Dict.SetStr("login", noop("login"))

	// close() — stub
	ftpCls.Dict.SetStr("close", noop("close"))

	// quit() — sends QUIT, closes; stub returns goodbye
	ftpCls.Dict.SetStr("quit", &object.BuiltinFunc{Name: "quit", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "221 Goodbye"}, nil
	}})

	// sendcmd(cmd) — stub
	ftpCls.Dict.SetStr("sendcmd", &object.BuiltinFunc{Name: "sendcmd", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "200 OK"}, nil
	}})

	// voidcmd(cmd) — stub
	ftpCls.Dict.SetStr("voidcmd", &object.BuiltinFunc{Name: "voidcmd", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "200 OK"}, nil
	}})

	// voidresp() — stub
	ftpCls.Dict.SetStr("voidresp", &object.BuiltinFunc{Name: "voidresp", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "200 OK"}, nil
	}})

	// getresp() — stub
	ftpCls.Dict.SetStr("getresp", &object.BuiltinFunc{Name: "getresp", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "200 OK"}, nil
	}})

	// getline() — stub
	ftpCls.Dict.SetStr("getline", &object.BuiltinFunc{Name: "getline", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: ""}, nil
	}})

	// getmultiline() — stub
	ftpCls.Dict.SetStr("getmultiline", &object.BuiltinFunc{Name: "getmultiline", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "200 OK"}, nil
	}})

	// putline(line) — stub
	ftpCls.Dict.SetStr("putline", noop("putline"))

	// putcmd(line) — stub
	ftpCls.Dict.SetStr("putcmd", noop("putcmd"))

	// pwd() — stub
	ftpCls.Dict.SetStr("pwd", &object.BuiltinFunc{Name: "pwd", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "/"}, nil
	}})

	// cwd(dirname) — stub
	ftpCls.Dict.SetStr("cwd", noop("cwd"))

	// mkd(dirname) — stub, returns dirname
	ftpCls.Dict.SetStr("mkd", &object.BuiltinFunc{Name: "mkd", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 1 {
			return a[1], nil
		}
		return &object.Str{V: ""}, nil
	}})

	// rmd(dirname) — stub
	ftpCls.Dict.SetStr("rmd", noop("rmd"))

	// delete(filename) — stub
	ftpCls.Dict.SetStr("delete", noop("delete"))

	// rename(fromname, toname) — stub
	ftpCls.Dict.SetStr("rename", noop("rename"))

	// size(filename) — stub, returns None
	ftpCls.Dict.SetStr("size", &object.BuiltinFunc{Name: "size", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// abort() — stub
	ftpCls.Dict.SetStr("abort", noop("abort"))

	// acct(password) — stub
	ftpCls.Dict.SetStr("acct", noop("acct"))

	// nlst(*args) — stub, returns empty list
	ftpCls.Dict.SetStr("nlst", &object.BuiltinFunc{Name: "nlst", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.List{V: nil}, nil
	}})

	// dir(*args) — stub
	ftpCls.Dict.SetStr("dir", noop("dir"))

	// mlsd(path='', facts=()) — stub, returns empty iterator
	ftpCls.Dict.SetStr("mlsd", &object.BuiltinFunc{Name: "mlsd", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		items := []object.Object{}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(items) {
				return nil, false, nil
			}
			v := items[idx]
			idx++
			return v, true, nil
		}}, nil
	}})

	// makeport() — stub
	ftpCls.Dict.SetStr("makeport", &object.BuiltinFunc{Name: "makeport", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// makepasv() — stub
	ftpCls.Dict.SetStr("makepasv", &object.BuiltinFunc{Name: "makepasv", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{&object.Str{V: "127.0.0.1"}, object.NewInt(0)}}, nil
	}})

	// sendport(host, port) — stub
	ftpCls.Dict.SetStr("sendport", noop("sendport"))

	// sendeprt(host, port) — stub
	ftpCls.Dict.SetStr("sendeprt", noop("sendeprt"))

	// ntransfercmd(cmd, rest=None) — stub
	ftpCls.Dict.SetStr("ntransfercmd", &object.BuiltinFunc{Name: "ntransfercmd", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{object.None, object.None}}, nil
	}})

	// transfercmd(cmd, rest=None) — stub
	ftpCls.Dict.SetStr("transfercmd", &object.BuiltinFunc{Name: "transfercmd", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// retrbinary(cmd, callback, blocksize=8192, rest=None) — stub
	ftpCls.Dict.SetStr("retrbinary", noop("retrbinary"))

	// retrlines(cmd, callback=None) — stub
	ftpCls.Dict.SetStr("retrlines", noop("retrlines"))

	// storbinary(cmd, fp, blocksize=8192, callback=None, rest=None) — stub
	ftpCls.Dict.SetStr("storbinary", noop("storbinary"))

	// storlines(cmd, fp, callback=None) — stub
	ftpCls.Dict.SetStr("storlines", noop("storlines"))

	// __enter__ / __exit__ context manager
	ftpCls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			return a[0], nil
		}
		return object.None, nil
	}})
	ftpCls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	m.Dict.SetStr("FTP", ftpCls)

	// ── FTP_TLS ───────────────────────────────────────────────────────────────

	ftpTLSCls := &object.Class{Name: "FTP_TLS", Bases: []*object.Class{ftpCls}, Dict: object.NewDict()}
	ftpTLSCls.Dict.SetStr("auth", noop("auth"))
	ftpTLSCls.Dict.SetStr("ccc", noop("ccc"))
	ftpTLSCls.Dict.SetStr("prot_c", noop("prot_c"))
	ftpTLSCls.Dict.SetStr("prot_p", noop("prot_p"))
	m.Dict.SetStr("FTP_TLS", ftpTLSCls)

	return m
}

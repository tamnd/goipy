package vm

import (
	"fmt"
	"os"
	"strings"

	"github.com/tamnd/goipy/object"
)

// buildNetrc registers the netrc module.
func (i *Interp) buildNetrc() *object.Module {
	m := &object.Module{Name: "netrc", Dict: object.NewDict()}

	// NetrcParseError exception class.
	parseErrCls := &object.Class{
		Name:  "NetrcParseError",
		Bases: []*object.Class{i.exception},
		Dict:  object.NewDict(),
	}
	parseErrCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		self.Dict.SetStr("msg", a[1])
		self.Dict.SetStr("args", &object.Tuple{V: a[1:]})
		var fn, ln object.Object = object.None, object.None
		if len(a) >= 3 {
			fn = a[2]
		}
		if len(a) >= 4 {
			ln = a[3]
		}
		if kw != nil {
			if v, ok := kw.GetStr("filename"); ok {
				fn = v
			}
			if v, ok := kw.GetStr("lineno"); ok {
				ln = v
			}
		}
		self.Dict.SetStr("filename", fn)
		self.Dict.SetStr("lineno", ln)
		return object.None, nil
	}})
	parseErrCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{}, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{}, nil
		}
		msg := ""
		fn := "None"
		ln := "None"
		if v, ok := self.Dict.GetStr("msg"); ok {
			msg = object.Str_(v)
		}
		if v, ok := self.Dict.GetStr("filename"); ok {
			fn = object.Str_(v)
		}
		if v, ok := self.Dict.GetStr("lineno"); ok {
			ln = object.Str_(v)
		}
		return &object.Str{V: fmt.Sprintf("%s (%s, line %s)", msg, fn, ln)}, nil
	}})
	m.Dict.SetStr("NetrcParseError", parseErrCls)

	// netrc class.
	netrcCls := &object.Class{Name: "netrc", Dict: object.NewDict()}

	netrcCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}

		// Determine file path.
		var filePath string
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				filePath = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("file"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					filePath = s.V
				}
			}
		}
		if filePath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, object.Errorf(i.valueErr, "could not determine home directory")
			}
			filePath = home + "/.netrc"
		}

		// Read file.
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, object.Errorf(i.osErr, "Cannot open file: %s", err.Error())
		}

		hosts, macros, parseErr := netrcParse(string(data), filePath)
		if parseErr != nil {
			exc := object.NewException(parseErrCls, parseErr.Error())
			exc.Dict = object.NewDict()
			exc.Dict.SetStr("msg", &object.Str{V: parseErr.msg})
			exc.Dict.SetStr("filename", &object.Str{V: parseErr.filename})
			exc.Dict.SetStr("lineno", object.NewInt(int64(parseErr.lineno)))
			return nil, exc
		}

		// Build hosts dict: {machine: (login, account, password)}.
		hostsDict := object.NewDict()
		for machine, e := range hosts {
			tup := &object.Tuple{V: []object.Object{
				&object.Str{V: e.login},
				&object.Str{V: e.account},
				&object.Str{V: e.password},
			}}
			hostsDict.SetStr(machine, tup)
		}

		// Build macros dict: {name: [line, ...]}.
		macrosDict := object.NewDict()
		for name, lines := range macros {
			items := make([]object.Object, len(lines))
			for k, l := range lines {
				items[k] = &object.Str{V: l}
			}
			macrosDict.SetStr(name, &object.List{V: items})
		}

		self.Dict.SetStr("hosts", hostsDict)
		self.Dict.SetStr("macros", macrosDict)
		return object.None, nil
	}})

	netrcCls.Dict.SetStr("authenticators", &object.BuiltinFunc{Name: "authenticators", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		host, ok2 := a[1].(*object.Str)
		if !ok2 {
			return object.None, nil
		}
		hostsRaw, ok3 := self.Dict.GetStr("hosts")
		if !ok3 {
			return object.None, nil
		}
		hostsDict, ok4 := hostsRaw.(*object.Dict)
		if !ok4 {
			return object.None, nil
		}
		if v, ok5, _ := hostsDict.Get(&object.Str{V: host.V}); ok5 {
			return v, nil
		}
		if v, ok5, _ := hostsDict.Get(&object.Str{V: "default"}); ok5 {
			return v, nil
		}
		return object.None, nil
	}})

	netrcCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{}, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{}, nil
		}
		hostsRaw, _ := self.Dict.GetStr("hosts")
		macrosRaw, _ := self.Dict.GetStr("macros")
		hostsDict, _ := hostsRaw.(*object.Dict)
		macrosDict, _ := macrosRaw.(*object.Dict)

		var b strings.Builder
		if hostsDict != nil {
			hkeys, hvals := hostsDict.Items()
			for k, kobj := range hkeys {
				host := object.Str_(kobj)
				tup, ok2 := hvals[k].(*object.Tuple)
				if !ok2 || len(tup.V) < 3 {
					continue
				}
				login := object.Str_(tup.V[0])
				account := object.Str_(tup.V[1])
				password := object.Str_(tup.V[2])
				b.WriteString(fmt.Sprintf("machine %s\n\tlogin %s\n", host, login))
				if account != "" {
					b.WriteString(fmt.Sprintf("\taccount %s\n", account))
				}
				b.WriteString(fmt.Sprintf("\tpassword %s\n", password))
			}
		}
		if macrosDict != nil {
			mkeys, mvals := macrosDict.Items()
			for k, kobj := range mkeys {
				name := object.Str_(kobj)
				b.WriteString(fmt.Sprintf("macdef %s\n", name))
				if lines, ok2 := mvals[k].(*object.List); ok2 {
					for _, l := range lines.V {
						b.WriteString(object.Str_(l))
					}
				}
				b.WriteByte('\n')
			}
		}
		return &object.Str{V: b.String()}, nil
	}})

	m.Dict.SetStr("netrc", netrcCls)
	return m
}

// ── parser types ──────────────────────────────────────────────────────────

type netrcParseError struct {
	msg      string
	filename string
	lineno   int
}

func (e *netrcParseError) Error() string {
	return fmt.Sprintf("%s (%s, line %d)", e.msg, e.filename, e.lineno)
}

type netrcEntry struct {
	login, account, password string
}

// hostOrder stores machine names in parse order (package-level, set by netrcParse).
// This is reset on each call to netrcParse so it's not goroutine-safe but
// tests run sequentially.
var hostOrder []string

// netrcParse parses netrc content and returns hosts map and macros map.
func netrcParse(content, filename string) (map[string]netrcEntry, map[string][]string, *netrcParseError) {
	hostOrder = nil
	hosts := make(map[string]netrcEntry)
	macros := make(map[string][]string)
	lex := &netrcLex{s: content, lineno: 1}

	for {
		savedLineno := lex.lineno
		tt := lex.token()
		if tt == "" {
			break
		}
		if strings.HasPrefix(tt, "#") {
			if lex.lineno == savedLineno {
				lex.readline()
			}
			continue
		}

		var entryname string
		switch tt {
		case "machine":
			entryname = lex.token()
		case "default":
			entryname = "default"
		case "macdef":
			entryname = lex.token()
			if entryname == "" {
				return nil, nil, &netrcParseError{"missing macdef name", filename, lex.lineno}
			}
			var lines []string
			for {
				line := lex.readline()
				if line == "" {
					return nil, nil, &netrcParseError{"Macro definition missing null line terminator.", filename, lex.lineno}
				}
				if line == "\n" {
					break
				}
				lines = append(lines, line)
			}
			macros[entryname] = lines
			continue
		default:
			return nil, nil, &netrcParseError{fmt.Sprintf("bad toplevel token '%s'", tt), filename, lex.lineno}
		}

		if entryname == "" {
			return nil, nil, &netrcParseError{fmt.Sprintf("missing '%s' name", tt), filename, lex.lineno}
		}

		login, account, password := "", "", ""
		if _, exists := hosts[entryname]; !exists {
			hostOrder = append(hostOrder, entryname)
		}
		hosts[entryname] = netrcEntry{}

		for {
			prevLineno := lex.lineno
			tok := lex.token()
			if strings.HasPrefix(tok, "#") {
				if lex.lineno == prevLineno {
					lex.readline()
				}
				continue
			}
			if tok == "" || tok == "machine" || tok == "default" || tok == "macdef" {
				hosts[entryname] = netrcEntry{login, account, password}
				lex.pushback(tok)
				break
			}
			switch tok {
			case "login", "user":
				login = lex.token()
			case "account":
				account = lex.token()
			case "password":
				password = lex.token()
			default:
				return nil, nil, &netrcParseError{fmt.Sprintf("bad follower token '%s'", tok), filename, lex.lineno}
			}
		}
	}
	return hosts, macros, nil
}

// ── lexer ─────────────────────────────────────────────────────────────────

type netrcLex struct {
	s        string
	pos      int
	lineno   int
	pushbuf  []string
}

func (l *netrcLex) pushback(tok string) {
	l.pushbuf = append([]string{tok}, l.pushbuf...)
}

// readline reads from the current position to the end of the line (consuming the '\n').
func (l *netrcLex) readline() string {
	if l.pos >= len(l.s) {
		return ""
	}
	start := l.pos
	for l.pos < len(l.s) && l.s[l.pos] != '\n' {
		l.pos++
	}
	line := l.s[start:l.pos]
	if l.pos < len(l.s) {
		l.lineno++
		l.pos++ // consume '\n'
		return line + "\n"
	}
	return line
}

// token returns the next whitespace-delimited token, handling quoted strings and backslash.
func (l *netrcLex) token() string {
	if len(l.pushbuf) > 0 {
		tok := l.pushbuf[0]
		l.pushbuf = l.pushbuf[1:]
		return tok
	}
	// Skip whitespace.
	for l.pos < len(l.s) {
		c := l.s[l.pos]
		if c == ' ' || c == '\t' || c == '\r' {
			l.pos++
		} else if c == '\n' {
			l.lineno++
			l.pos++
		} else {
			break
		}
	}
	if l.pos >= len(l.s) {
		return ""
	}
	c := l.s[l.pos]

	// Quoted string.
	if c == '"' {
		l.pos++
		var b strings.Builder
		for l.pos < len(l.s) {
			c2 := l.s[l.pos]
			l.pos++
			if c2 == '"' {
				return b.String()
			}
			if c2 == '\\' && l.pos < len(l.s) {
				c2 = l.s[l.pos]
				l.pos++
			}
			if c2 == '\n' {
				l.lineno++
			}
			b.WriteByte(c2)
		}
		return b.String()
	}

	// Unquoted token — may start with backslash.
	var b strings.Builder
	if c == '\\' {
		l.pos++
		if l.pos < len(l.s) {
			c = l.s[l.pos]
			l.pos++
		}
	} else {
		l.pos++
	}
	if c == '\n' {
		l.lineno++
	}
	b.WriteByte(c)

	for l.pos < len(l.s) {
		c2 := l.s[l.pos]
		if c2 == ' ' || c2 == '\t' || c2 == '\r' {
			l.pos++
			break
		}
		if c2 == '\n' {
			l.lineno++
			l.pos++
			break
		}
		if c2 == '\\' {
			l.pos++
			if l.pos < len(l.s) {
				c2 = l.s[l.pos]
				l.pos++
			}
		} else {
			l.pos++
		}
		if c2 == '\n' {
			l.lineno++
		}
		b.WriteByte(c2)
	}
	return b.String()
}

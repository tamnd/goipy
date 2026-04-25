package vm

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tamnd/goipy/object"
)

// ── mbox state ───────────────────────────────────────────────────────────────

type mboxEntry struct {
	start, end int64
	deleted    bool
}

type mboxState struct {
	mu      sync.Mutex
	path    string
	toc     []mboxEntry // index is the int key; deleted slot means removed
	pending bool
	closed  bool
}

// parseMboxFile reads an mbox file and builds a TOC.
func parseMboxFile(path string) ([]mboxEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var toc []mboxEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	var pos int64
	var msgStart int64 = -1
	var msgStartValid bool

	for scanner.Scan() {
		line := scanner.Text()
		lineLen := int64(len(line) + 1) // +1 for \n
		if strings.HasPrefix(line, "From ") {
			if msgStartValid {
				toc = append(toc, mboxEntry{start: msgStart, end: pos})
			}
			msgStart = pos
			msgStartValid = true
		}
		pos += lineLen
	}
	if msgStartValid {
		toc = append(toc, mboxEntry{start: msgStart, end: pos})
	}
	return toc, scanner.Err()
}

// readMboxMessage reads the raw bytes for a single mbox message entry.
func readMboxMessage(path string, entry mboxEntry) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := f.Seek(entry.start, 0); err != nil {
		return nil, err
	}
	size := entry.end - entry.start
	if size <= 0 {
		return []byte{}, nil
	}
	buf := make([]byte, size)
	_, err = f.Read(buf)
	return buf, err
}

// stripFromLine removes the leading "From ..." envelope line from a raw mbox message.
func stripFromLine(raw string) string {
	if strings.HasPrefix(raw, "From ") {
		idx := strings.Index(raw, "\n")
		if idx >= 0 {
			return raw[idx+1:]
		}
	}
	return raw
}

// mboxFlush rewrites the mbox file, omitting deleted entries.
func mboxFlush(st *mboxState) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.pending {
		return nil
	}

	// Read all live messages
	var messages [][]byte
	for _, e := range st.toc {
		if e.deleted {
			continue
		}
		raw, err := readMboxMessage(st.path, e)
		if err != nil {
			return err
		}
		messages = append(messages, raw)
	}

	// Rewrite file
	f, err := os.Create(st.path)
	if err != nil {
		return err
	}
	defer f.Close()

	var newTOC []mboxEntry
	var pos int64
	for _, raw := range messages {
		start := pos
		if _, err2 := f.Write(raw); err2 != nil {
			return err2
		}
		end := pos + int64(len(raw))
		newTOC = append(newTOC, mboxEntry{start: start, end: end})
		pos = end
	}
	st.toc = newTOC
	st.pending = false
	return nil
}

// mboxSerializeMessage converts a Python object to mbox format string.
func mboxSerializeMessage(msg object.Object) string {
	var raw string
	switch v := msg.(type) {
	case *object.Str:
		raw = v.V
	case *object.Bytes:
		raw = string(v.V)
	case *object.Instance:
		raw = emailAsString(v)
	default:
		raw = object.Repr(msg)
	}

	// Ensure it starts with "From " line
	if !strings.HasPrefix(raw, "From ") {
		fromLine := fmt.Sprintf("From MAILER-DAEMON %s\n", time.Now().Format(time.UnixDate))
		raw = fromLine + raw
	}

	// Ensure trailing newline
	if !strings.HasSuffix(raw, "\n") {
		raw += "\n"
	}
	return raw
}

// mboxParseMessage parses a raw mbox message (with "From " line) into a Message instance.
func (i *Interp) mboxParseMessage(raw string, cls *object.Class) (*object.Instance, error) {
	content := stripFromLine(raw)
	return i.emailParseString(content, cls)
}

// ── Maildir state ─────────────────────────────────────────────────────────────

type maildirState struct {
	mu        sync.Mutex
	path      string
	factory   object.Object
	toc       map[string]string // key → relative path ("new/fname" or "cur/fname")
	tocLoaded bool
	closed    bool
}

func (s *maildirState) loadTOC() {
	if s.tocLoaded {
		return
	}
	if s.toc == nil {
		s.toc = make(map[string]string)
	}
	for _, sub := range []string{"new", "cur"} {
		dir := filepath.Join(s.path, sub)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			key := name
			if idx := strings.Index(name, ":"); idx >= 0 {
				key = name[:idx]
			}
			s.toc[key] = sub + "/" + name
		}
	}
	s.tocLoaded = true
}

func maildirKey() string {
	return fmt.Sprintf("%d.%d.goipy", time.Now().UnixNano(), os.Getpid())
}

// maildirSerializeMessage converts a Python object to a flat RFC 2822 string.
func maildirSerializeMessage(msg object.Object) string {
	switch v := msg.(type) {
	case *object.Str:
		return v.V
	case *object.Bytes:
		return string(v.V)
	case *object.Instance:
		return emailAsString(v)
	default:
		return object.Repr(msg)
	}
}

// ── buildMailbox ─────────────────────────────────────────────────────────────

func (i *Interp) buildMailbox() *object.Module {
	m := &object.Module{Name: "mailbox", Dict: object.NewDict()}

	// ── Exception hierarchy ──────────────────────────────────────────────────
	mailboxErrClass := &object.Class{Name: "Error", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	noSuchClass := &object.Class{Name: "NoSuchMailboxError", Bases: []*object.Class{mailboxErrClass}, Dict: object.NewDict()}
	notEmptyClass := &object.Class{Name: "NotEmptyError", Bases: []*object.Class{mailboxErrClass}, Dict: object.NewDict()}
	extClashClass := &object.Class{Name: "ExternalClashError", Bases: []*object.Class{mailboxErrClass}, Dict: object.NewDict()}
	formatErrClass := &object.Class{Name: "FormatError", Bases: []*object.Class{mailboxErrClass}, Dict: object.NewDict()}

	m.Dict.SetStr("Error", mailboxErrClass)
	m.Dict.SetStr("NoSuchMailboxError", noSuchClass)
	m.Dict.SetStr("NotEmptyError", notEmptyClass)
	m.Dict.SetStr("ExternalClashError", extClashClass)
	m.Dict.SetStr("FormatError", formatErrClass)

	// ── mailbox.Message class ────────────────────────────────────────────────
	msgCls := i.buildMailboxMessageClass()
	m.Dict.SetStr("Message", msgCls)

	// ── mailbox.mboxMessage class ────────────────────────────────────────────
	mboxMsgCls := i.buildMboxMessageClass(msgCls)
	m.Dict.SetStr("mboxMessage", mboxMsgCls)

	// ── mailbox.MaildirMessage class ─────────────────────────────────────────
	maildirMsgCls := i.buildMaildirMessageClass(msgCls)
	m.Dict.SetStr("MaildirMessage", maildirMsgCls)

	// ── mailbox.mbox class ───────────────────────────────────────────────────
	mboxCls := i.buildMboxClass(mboxMsgCls, mailboxErrClass, noSuchClass)
	m.Dict.SetStr("mbox", mboxCls)

	// ── mailbox.Maildir class ────────────────────────────────────────────────
	maildirCls := i.buildMaildirClass(maildirMsgCls, mailboxErrClass, noSuchClass, notEmptyClass)
	m.Dict.SetStr("Maildir", maildirCls)

	return m
}

// ── mailbox.Message class ─────────────────────────────────────────────────────

func (i *Interp) buildMailboxMessageClass() *object.Class {
	cls := &object.Class{Name: "Message", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			emailInitInstance(self)
			self.Dict.SetStr("_subdir", &object.Str{V: ""})
			self.Dict.SetStr("_date", &object.Float{V: 0})
			self.Dict.SetStr("_info", &object.Str{V: ""})
			if len(a) >= 2 && a[1] != object.None {
				initMailboxMessageFrom(i, self, a[1])
			}
			return object.None, nil
		}})

	addEmailMethods(i, cls)
	addMailboxMessageMethods(i, cls)

	return cls
}

// addEmailMethods copies the standard email.message.Message methods onto a class.
func addEmailMethods(i *Interp, cls *object.Class) {
	cls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			name, _ := a[1].(*object.Str)
			if name == nil {
				return object.None, nil
			}
			if v, ok := emailHeaderGet(self, name.V); ok {
				return &object.Str{V: v}, nil
			}
			return object.None, nil
		}})
	cls.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 3 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			name := object.Str_(a[1])
			value := object.Str_(a[2])
			emailHeaderSet(self, name, value)
			return object.None, nil
		}})
	cls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			name := object.Str_(a[1])
			var def object.Object = object.None
			if len(a) >= 3 {
				def = a[2]
			}
			if v, ok := emailHeaderGet(self, name); ok {
				return &object.Str{V: v}, nil
			}
			return def, nil
		}})
	cls.Dict.SetStr("get_payload", &object.BuiltinFunc{Name: "get_payload",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			pl, ok := self.Dict.GetStr("_payload")
			if !ok {
				return object.None, nil
			}
			return pl, nil
		}})
	cls.Dict.SetStr("set_payload", &object.BuiltinFunc{Name: "set_payload",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			a[0].(*object.Instance).Dict.SetStr("_payload", a[1])
			return object.None, nil
		}})
	cls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			return &object.Str{V: emailAsString(a[0].(*object.Instance))}, nil
		}})
}

// addMailboxMessageMethods adds subdir/date/info/flags methods.
func addMailboxMessageMethods(i *Interp, cls *object.Class) {
	cls.Dict.SetStr("get_subdir", &object.BuiltinFunc{Name: "get_subdir",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			v, ok := self.Dict.GetStr("_subdir")
			if !ok {
				return &object.Str{V: ""}, nil
			}
			return v, nil
		}})
	cls.Dict.SetStr("set_subdir", &object.BuiltinFunc{Name: "set_subdir",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			a[0].(*object.Instance).Dict.SetStr("_subdir", a[1])
			return object.None, nil
		}})
	cls.Dict.SetStr("get_flags", &object.BuiltinFunc{Name: "get_flags",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			v, ok := self.Dict.GetStr("_flags")
			if !ok {
				return &object.Str{V: ""}, nil
			}
			return v, nil
		}})
	cls.Dict.SetStr("set_flags", &object.BuiltinFunc{Name: "set_flags",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			a[0].(*object.Instance).Dict.SetStr("_flags", a[1])
			return object.None, nil
		}})
	cls.Dict.SetStr("add_flag", &object.BuiltinFunc{Name: "add_flag",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			flag := object.Str_(a[1])
			cur := ""
			if v, ok := self.Dict.GetStr("_flags"); ok {
				cur = object.Str_(v)
			}
			for _, c := range flag {
				if !strings.ContainsRune(cur, c) {
					cur += string(c)
				}
			}
			self.Dict.SetStr("_flags", &object.Str{V: cur})
			return object.None, nil
		}})
	cls.Dict.SetStr("remove_flag", &object.BuiltinFunc{Name: "remove_flag",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			flag := object.Str_(a[1])
			cur := ""
			if v, ok := self.Dict.GetStr("_flags"); ok {
				cur = object.Str_(v)
			}
			for _, c := range flag {
				cur = strings.ReplaceAll(cur, string(c), "")
			}
			self.Dict.SetStr("_flags", &object.Str{V: cur})
			return object.None, nil
		}})
	cls.Dict.SetStr("get_date", &object.BuiltinFunc{Name: "get_date",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			v, ok := self.Dict.GetStr("_date")
			if !ok {
				return &object.Float{V: 0}, nil
			}
			return v, nil
		}})
	cls.Dict.SetStr("set_date", &object.BuiltinFunc{Name: "set_date",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			a[0].(*object.Instance).Dict.SetStr("_date", a[1])
			return object.None, nil
		}})
	cls.Dict.SetStr("get_info", &object.BuiltinFunc{Name: "get_info",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			self := a[0].(*object.Instance)
			v, ok := self.Dict.GetStr("_info")
			if !ok {
				return &object.Str{V: ""}, nil
			}
			return v, nil
		}})
	cls.Dict.SetStr("set_info", &object.BuiltinFunc{Name: "set_info",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			a[0].(*object.Instance).Dict.SetStr("_info", a[1])
			return object.None, nil
		}})
}

// initMailboxMessageFrom populates a Message instance from a str/bytes source.
func initMailboxMessageFrom(i *Interp, inst *object.Instance, src object.Object) {
	var raw string
	switch v := src.(type) {
	case *object.Str:
		raw = v.V
	case *object.Bytes:
		raw = string(v.V)
	case *object.Instance:
		if hdrs := emailGetHeaders(v); hdrs != nil {
			for _, item := range hdrs.V {
				if t, ok := item.(*object.Tuple); ok && len(t.V) == 2 {
					emailHeaderSet(inst, object.Str_(t.V[0]), object.Str_(t.V[1]))
				}
			}
		}
		if pl, ok := v.Dict.GetStr("_payload"); ok {
			inst.Dict.SetStr("_payload", pl)
		}
		return
	default:
		return
	}
	parsed, err := i.emailParseString(raw, inst.Class)
	if err != nil {
		return
	}
	if hdrs := emailGetHeaders(parsed); hdrs != nil {
		for _, item := range hdrs.V {
			if t, ok := item.(*object.Tuple); ok && len(t.V) == 2 {
				emailHeaderSet(inst, object.Str_(t.V[0]), object.Str_(t.V[1]))
			}
		}
	}
	if pl, ok := parsed.Dict.GetStr("_payload"); ok {
		inst.Dict.SetStr("_payload", pl)
	}
}

// ── mailbox.mboxMessage class ─────────────────────────────────────────────────

func (i *Interp) buildMboxMessageClass(baseCls *object.Class) *object.Class {
	cls := &object.Class{Name: "mboxMessage", Bases: []*object.Class{baseCls}, Dict: object.NewDict()}

	cls.Dict.SetStr("get_from", &object.BuiltinFunc{Name: "get_from",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			self := a[0].(*object.Instance)
			v, ok := self.Dict.GetStr("_from")
			if !ok {
				return &object.Str{V: ""}, nil
			}
			return v, nil
		}})
	cls.Dict.SetStr("set_from", &object.BuiltinFunc{Name: "set_from",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			from := object.Str_(a[1])
			if len(a) >= 3 && a[2] != object.None {
				switch tv := a[2].(type) {
				case *object.Bool:
					if tv.V {
						from += " " + time.Now().Format(time.UnixDate)
					}
				case *object.Float:
					t := time.Unix(int64(tv.V), 0)
					from += " " + t.Format(time.UnixDate)
				}
			}
			self.Dict.SetStr("_from", &object.Str{V: from})
			return object.None, nil
		}})

	// Flags via Status/X-Status headers
	cls.Dict.SetStr("get_flags", &object.BuiltinFunc{Name: "get_flags",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			self := a[0].(*object.Instance)
			flags := ""
			if v, ok := emailHeaderGet(self, "Status"); ok {
				flags += v
			}
			if v, ok := emailHeaderGet(self, "X-Status"); ok {
				flags += v
			}
			return &object.Str{V: flags}, nil
		}})
	cls.Dict.SetStr("set_flags", &object.BuiltinFunc{Name: "set_flags",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			flags := object.Str_(a[1])
			emailHeaderDel(self, "Status")
			emailHeaderDel(self, "X-Status")
			status, xstatus := splitMboxFlags(flags)
			if status != "" {
				emailHeaderSet(self, "Status", status)
			}
			if xstatus != "" {
				emailHeaderSet(self, "X-Status", xstatus)
			}
			return object.None, nil
		}})
	cls.Dict.SetStr("add_flag", &object.BuiltinFunc{Name: "add_flag",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			flag := object.Str_(a[1])
			cur := ""
			if v, ok := emailHeaderGet(self, "Status"); ok {
				cur += v
			}
			if v, ok := emailHeaderGet(self, "X-Status"); ok {
				cur += v
			}
			for _, c := range flag {
				if !strings.ContainsRune(cur, c) {
					cur += string(c)
				}
			}
			emailHeaderDel(self, "Status")
			emailHeaderDel(self, "X-Status")
			status, xstatus := splitMboxFlags(cur)
			if status != "" {
				emailHeaderSet(self, "Status", status)
			}
			if xstatus != "" {
				emailHeaderSet(self, "X-Status", xstatus)
			}
			return object.None, nil
		}})
	cls.Dict.SetStr("remove_flag", &object.BuiltinFunc{Name: "remove_flag",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			flag := object.Str_(a[1])
			cur := ""
			if v, ok := emailHeaderGet(self, "Status"); ok {
				cur += v
			}
			if v, ok := emailHeaderGet(self, "X-Status"); ok {
				cur += v
			}
			for _, c := range flag {
				cur = strings.ReplaceAll(cur, string(c), "")
			}
			emailHeaderDel(self, "Status")
			emailHeaderDel(self, "X-Status")
			status, xstatus := splitMboxFlags(cur)
			if status != "" {
				emailHeaderSet(self, "Status", status)
			}
			if xstatus != "" {
				emailHeaderSet(self, "X-Status", xstatus)
			}
			return object.None, nil
		}})

	return cls
}

func splitMboxFlags(flags string) (status, xstatus string) {
	for _, c := range flags {
		switch c {
		case 'R', 'O':
			status += string(c)
		default:
			xstatus += string(c)
		}
	}
	return
}

// ── mailbox.MaildirMessage class ──────────────────────────────────────────────

func (i *Interp) buildMaildirMessageClass(baseCls *object.Class) *object.Class {
	cls := &object.Class{Name: "MaildirMessage", Bases: []*object.Class{baseCls}, Dict: object.NewDict()}

	cls.Dict.SetStr("get_subdir", &object.BuiltinFunc{Name: "get_subdir",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "new"}, nil
			}
			self := a[0].(*object.Instance)
			v, ok := self.Dict.GetStr("_subdir")
			if !ok {
				return &object.Str{V: "new"}, nil
			}
			return v, nil
		}})
	cls.Dict.SetStr("set_subdir", &object.BuiltinFunc{Name: "set_subdir",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			sub := object.Str_(a[1])
			if sub != "new" && sub != "cur" {
				return nil, object.Errorf(i.valueErr, "invalid subdir: %s", sub)
			}
			self.Dict.SetStr("_subdir", &object.Str{V: sub})
			return object.None, nil
		}})
	cls.Dict.SetStr("get_info", &object.BuiltinFunc{Name: "get_info",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			self := a[0].(*object.Instance)
			v, ok := self.Dict.GetStr("_info")
			if !ok {
				return &object.Str{V: ""}, nil
			}
			return v, nil
		}})
	cls.Dict.SetStr("set_info", &object.BuiltinFunc{Name: "set_info",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			a[0].(*object.Instance).Dict.SetStr("_info", a[1])
			return object.None, nil
		}})
	cls.Dict.SetStr("get_flags", &object.BuiltinFunc{Name: "get_flags",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			self := a[0].(*object.Instance)
			info := ""
			if v, ok := self.Dict.GetStr("_info"); ok {
				info = object.Str_(v)
			}
			if idx := strings.Index(info, "2,"); idx >= 0 {
				return &object.Str{V: info[idx+2:]}, nil
			}
			return &object.Str{V: ""}, nil
		}})
	cls.Dict.SetStr("set_flags", &object.BuiltinFunc{Name: "set_flags",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			flags := object.Str_(a[1])
			self.Dict.SetStr("_info", &object.Str{V: "2," + flags})
			return object.None, nil
		}})
	cls.Dict.SetStr("add_flag", &object.BuiltinFunc{Name: "add_flag",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			flag := object.Str_(a[1])
			info := ""
			if v, ok := self.Dict.GetStr("_info"); ok {
				info = object.Str_(v)
			}
			cur := ""
			if idx := strings.Index(info, "2,"); idx >= 0 {
				cur = info[idx+2:]
			}
			for _, c := range flag {
				if !strings.ContainsRune(cur, c) {
					cur += string(c)
				}
			}
			self.Dict.SetStr("_info", &object.Str{V: "2," + cur})
			return object.None, nil
		}})
	cls.Dict.SetStr("remove_flag", &object.BuiltinFunc{Name: "remove_flag",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			flag := object.Str_(a[1])
			info := ""
			if v, ok := self.Dict.GetStr("_info"); ok {
				info = object.Str_(v)
			}
			cur := ""
			if idx := strings.Index(info, "2,"); idx >= 0 {
				cur = info[idx+2:]
			}
			for _, c := range flag {
				cur = strings.ReplaceAll(cur, string(c), "")
			}
			self.Dict.SetStr("_info", &object.Str{V: "2," + cur})
			return object.None, nil
		}})
	cls.Dict.SetStr("get_date", &object.BuiltinFunc{Name: "get_date",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Float{V: 0}, nil
			}
			self := a[0].(*object.Instance)
			v, ok := self.Dict.GetStr("_date")
			if !ok {
				return &object.Float{V: 0}, nil
			}
			return v, nil
		}})
	cls.Dict.SetStr("set_date", &object.BuiltinFunc{Name: "set_date",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			a[0].(*object.Instance).Dict.SetStr("_date", a[1])
			return object.None, nil
		}})

	return cls
}

// ── mailbox.mbox class (instance created per __init__ call) ──────────────────

func (i *Interp) buildMboxClass(msgCls *object.Class, mailboxErr, noSuchErr *object.Class) *object.Class {
	cls := &object.Class{Name: "mbox", Dict: object.NewDict()}

	bf := func(name string, fn func(any, []object.Object, *object.Dict) (object.Object, error)) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: fn}
	}

	// __new__ is not used; __init__ creates a per-instance mboxState and wires
	// all methods into the instance dict (same pattern as shelve/bz2).
	cls.Dict.SetStr("__init__", bf("__init__", func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "mbox() requires path argument")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "mbox.__init__: self must be Instance")
		}
		path := object.Str_(a[1])
		create := true
		if len(a) >= 4 {
			if b, ok2 := a[3].(*object.Bool); ok2 {
				create = b.V
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("create"); ok2 {
				if b, ok3 := v.(*object.Bool); ok3 {
					create = b.V
				}
			}
		}

		if create {
			f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
			if err != nil {
				return nil, object.Errorf(i.osErr, "mbox: %v", err)
			}
			f.Close()
		} else {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return nil, object.Errorf(noSuchErr, "no such mailbox: %s", path)
			}
		}

		toc, err := parseMboxFile(path)
		if err != nil {
			return nil, object.Errorf(i.osErr, "mbox parse: %v", err)
		}

		st := &mboxState{path: path, toc: toc}
		i.wireMboxInstance(self, st, msgCls, mailboxErr, noSuchErr)
		return object.None, nil
	}))

	return cls
}

// wireMboxInstance places all mbox methods into an instance's own dict,
// closing over the mboxState st.
func (i *Interp) wireMboxInstance(self *object.Instance, st *mboxState, msgCls *object.Class, mailboxErr, noSuchErr *object.Class) {
	d := self.Dict

	checkOpen := func() error {
		if st.closed {
			return object.Errorf(i.valueErr, "I/O operation on closed mailbox")
		}
		return nil
	}

	getIntKey := func(obj object.Object) (int, error) {
		kInt, ok := obj.(*object.Int)
		if !ok {
			return 0, object.Errorf(i.typeErr, "mbox keys must be int")
		}
		return int(kInt.Int64()), nil
	}

	bf := func(name string, fn func(any, []object.Object, *object.Dict) (object.Object, error)) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: fn}
	}

	// add(message) -> int key
	d.SetStr("add", bf("add", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "add() requires message")
		}
		raw := mboxSerializeMessage(a[0])

		st.mu.Lock()
		defer st.mu.Unlock()

		f, err := os.OpenFile(st.path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return nil, object.Errorf(i.osErr, "mbox add: %v", err)
		}
		defer f.Close()

		info, err2 := f.Stat()
		if err2 != nil {
			return nil, object.Errorf(i.osErr, "mbox stat: %v", err2)
		}
		start := info.Size()
		if _, err3 := f.WriteString(raw); err3 != nil {
			return nil, object.Errorf(i.osErr, "mbox write: %v", err3)
		}
		end := start + int64(len(raw))

		key := len(st.toc)
		st.toc = append(st.toc, mboxEntry{start: start, end: end})
		return object.IntFromInt64(int64(key)), nil
	}))

	// __getitem__(key) -> Message
	d.SetStr("__getitem__", bf("__getitem__", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "__getitem__ requires key")
		}
		k, err := getIntKey(a[0])
		if err != nil {
			return nil, err
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		if k < 0 || k >= len(st.toc) || st.toc[k].deleted {
			return nil, object.Errorf(i.keyErr, "%d", k)
		}
		raw, err2 := readMboxMessage(st.path, st.toc[k])
		if err2 != nil {
			return nil, object.Errorf(i.osErr, "mbox read: %v", err2)
		}
		return i.mboxParseMessage(string(raw), msgCls)
	}))

	// get(key, default=None)
	d.SetStr("get", bf("get", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var def object.Object = object.None
		if len(a) >= 2 {
			def = a[1]
		}
		if len(a) < 1 {
			return def, nil
		}
		k, err := getIntKey(a[0])
		if err != nil {
			return def, nil
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		if k < 0 || k >= len(st.toc) || st.toc[k].deleted {
			return def, nil
		}
		raw, err2 := readMboxMessage(st.path, st.toc[k])
		if err2 != nil {
			return def, nil
		}
		msg, err3 := i.mboxParseMessage(string(raw), msgCls)
		if err3 != nil {
			return def, nil
		}
		return msg, nil
	}))

	// get_message(key) -> mboxMessage
	d.SetStr("get_message", bf("get_message", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "get_message() requires key")
		}
		k, err := getIntKey(a[0])
		if err != nil {
			return nil, err
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		if k < 0 || k >= len(st.toc) || st.toc[k].deleted {
			return nil, object.Errorf(i.keyErr, "%d", k)
		}
		raw, err2 := readMboxMessage(st.path, st.toc[k])
		if err2 != nil {
			return nil, object.Errorf(i.osErr, "mbox read: %v", err2)
		}
		return i.mboxParseMessage(string(raw), msgCls)
	}))

	// get_string(key) -> str
	d.SetStr("get_string", bf("get_string", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "get_string() requires key")
		}
		k, err := getIntKey(a[0])
		if err != nil {
			return nil, err
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		if k < 0 || k >= len(st.toc) || st.toc[k].deleted {
			return nil, object.Errorf(i.keyErr, "%d", k)
		}
		raw, err2 := readMboxMessage(st.path, st.toc[k])
		if err2 != nil {
			return nil, object.Errorf(i.osErr, "mbox read: %v", err2)
		}
		return &object.Str{V: stripFromLine(string(raw))}, nil
	}))

	// get_bytes(key) -> bytes
	d.SetStr("get_bytes", bf("get_bytes", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "get_bytes() requires key")
		}
		k, err := getIntKey(a[0])
		if err != nil {
			return nil, err
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		if k < 0 || k >= len(st.toc) || st.toc[k].deleted {
			return nil, object.Errorf(i.keyErr, "%d", k)
		}
		raw, err2 := readMboxMessage(st.path, st.toc[k])
		if err2 != nil {
			return nil, object.Errorf(i.osErr, "mbox read: %v", err2)
		}
		stripped := stripFromLine(string(raw))
		return &object.Bytes{V: []byte(stripped)}, nil
	}))

	// __setitem__(key, message)
	d.SetStr("__setitem__", bf("__setitem__", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "__setitem__ requires key and message")
		}
		k, err := getIntKey(a[0])
		if err != nil {
			return nil, err
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		if k < 0 || k >= len(st.toc) || st.toc[k].deleted {
			return nil, object.Errorf(i.keyErr, "%d", k)
		}
		raw := mboxSerializeMessage(a[1])
		f, err2 := os.OpenFile(st.path, os.O_APPEND|os.O_WRONLY, 0600)
		if err2 != nil {
			return nil, object.Errorf(i.osErr, "mbox setitem: %v", err2)
		}
		defer f.Close()
		info, _ := f.Stat()
		start := info.Size()
		f.WriteString(raw)
		end := start + int64(len(raw))
		st.toc[k] = mboxEntry{start: start, end: end}
		return object.None, nil
	}))

	// remove(key)
	d.SetStr("remove", bf("remove", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "remove() requires key")
		}
		k, err := getIntKey(a[0])
		if err != nil {
			return nil, err
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		if k < 0 || k >= len(st.toc) || st.toc[k].deleted {
			return nil, object.Errorf(i.keyErr, "%d", k)
		}
		st.toc[k].deleted = true
		st.pending = true
		return object.None, nil
	}))

	// __delitem__(key)
	d.SetStr("__delitem__", bf("__delitem__", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if err := checkOpen(); err != nil {
			return nil, err
		}
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "__delitem__ requires key")
		}
		k, err := getIntKey(a[0])
		if err != nil {
			return nil, err
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		if k < 0 || k >= len(st.toc) || st.toc[k].deleted {
			return nil, object.Errorf(i.keyErr, "%d", k)
		}
		st.toc[k].deleted = true
		st.pending = true
		return object.None, nil
	}))

	// discard(key) -- no error if absent
	d.SetStr("discard", bf("discard", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		k, err := getIntKey(a[0])
		if err != nil {
			return object.None, nil
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		if k >= 0 && k < len(st.toc) && !st.toc[k].deleted {
			st.toc[k].deleted = true
			st.pending = true
		}
		return object.None, nil
	}))

	// __contains__(key) -> bool
	d.SetStr("__contains__", bf("__contains__", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.BoolOf(false), nil
		}
		k, err := getIntKey(a[0])
		if err != nil {
			return object.BoolOf(false), nil
		}
		st.mu.Lock()
		defer st.mu.Unlock()
		if k >= 0 && k < len(st.toc) && !st.toc[k].deleted {
			return object.BoolOf(true), nil
		}
		return object.BoolOf(false), nil
	}))

	// __len__() -> int
	d.SetStr("__len__", bf("__len__", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		st.mu.Lock()
		defer st.mu.Unlock()
		count := 0
		for _, e := range st.toc {
			if !e.deleted {
				count++
			}
		}
		return object.IntFromInt64(int64(count)), nil
	}))

	// keys() -> list of int
	d.SetStr("keys", bf("keys", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		st.mu.Lock()
		defer st.mu.Unlock()
		var ks []object.Object
		for k, e := range st.toc {
			if !e.deleted {
				ks = append(ks, object.IntFromInt64(int64(k)))
			}
		}
		return &object.List{V: ks}, nil
	}))

	// values() -> list of Message
	d.SetStr("values", bf("values", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		st.mu.Lock()
		entries := make([]mboxEntry, len(st.toc))
		copy(entries, st.toc)
		path := st.path
		st.mu.Unlock()
		var msgs []object.Object
		for _, e := range entries {
			if e.deleted {
				continue
			}
			raw, err := readMboxMessage(path, e)
			if err != nil {
				continue
			}
			msg, err2 := i.mboxParseMessage(string(raw), msgCls)
			if err2 != nil {
				continue
			}
			msgs = append(msgs, msg)
		}
		return &object.List{V: msgs}, nil
	}))

	// items() -> list of (key, Message)
	d.SetStr("items", bf("items", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		st.mu.Lock()
		entries := make([]mboxEntry, len(st.toc))
		copy(entries, st.toc)
		path := st.path
		st.mu.Unlock()
		var items []object.Object
		for k, e := range entries {
			if e.deleted {
				continue
			}
			raw, err := readMboxMessage(path, e)
			if err != nil {
				continue
			}
			msg, err2 := i.mboxParseMessage(string(raw), msgCls)
			if err2 != nil {
				continue
			}
			items = append(items, &object.Tuple{V: []object.Object{
				object.IntFromInt64(int64(k)), msg,
			}})
		}
		return &object.List{V: items}, nil
	}))

	// __iter__()
	d.SetStr("__iter__", bf("__iter__", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		st.mu.Lock()
		entries := make([]mboxEntry, len(st.toc))
		copy(entries, st.toc)
		path := st.path
		st.mu.Unlock()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			for idx < len(entries) {
				e := entries[idx]
				idx++
				if e.deleted {
					continue
				}
				raw, err := readMboxMessage(path, e)
				if err != nil {
					continue
				}
				msg, err2 := i.mboxParseMessage(string(raw), msgCls)
				if err2 != nil {
					continue
				}
				return msg, true, nil
			}
			return nil, false, nil
		}}, nil
	}))

	// flush()
	d.SetStr("flush", bf("flush", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, mboxFlush(st)
	}))

	// lock() / unlock()
	d.SetStr("lock", bf("lock", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}))
	d.SetStr("unlock", bf("unlock", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}))

	// close()
	d.SetStr("close", bf("close", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if st.closed {
			return object.None, nil
		}
		if err := mboxFlush(st); err != nil {
			return nil, object.Errorf(i.osErr, "mbox close: %v", err)
		}
		st.closed = true
		return object.None, nil
	}))

	// clear()
	d.SetStr("clear", bf("clear", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		st.mu.Lock()
		defer st.mu.Unlock()
		for k := range st.toc {
			st.toc[k].deleted = true
		}
		if err := os.Truncate(st.path, 0); err != nil {
			return nil, object.Errorf(i.osErr, "mbox clear: %v", err)
		}
		st.toc = nil
		st.pending = false
		return object.None, nil
	}))

	// __enter__
	d.SetStr("__enter__", bf("__enter__", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return self, nil
	}))

	// __exit__
	d.SetStr("__exit__", bf("__exit__", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if !st.closed {
			mboxFlush(st)
			st.closed = true
		}
		return object.None, nil
	}))
}

// ── mailbox.Maildir class ─────────────────────────────────────────────────────

func (i *Interp) buildMaildirClass(msgCls *object.Class, mailboxErr, noSuchErr, notEmptyErr *object.Class) *object.Class {
	cls := &object.Class{Name: "Maildir", Dict: object.NewDict()}

	bf := func(name string, fn func(any, []object.Object, *object.Dict) (object.Object, error)) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: fn}
	}

	cls.Dict.SetStr("__init__", bf("__init__", func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "Maildir() requires dirname")
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "Maildir.__init__: self must be Instance")
		}
		path := object.Str_(a[1])
		var factory object.Object = object.None
		create := true

		if len(a) >= 3 {
			factory = a[2]
		}
		if len(a) >= 4 {
			if b, ok2 := a[3].(*object.Bool); ok2 {
				create = b.V
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("factory"); ok2 {
				factory = v
			}
			if v, ok2 := kw.GetStr("create"); ok2 {
				if b, ok3 := v.(*object.Bool); ok3 {
					create = b.V
				}
			}
		}

		if create {
			for _, sub := range []string{"", "new", "cur", "tmp"} {
				dir := filepath.Join(path, sub)
				if err := os.MkdirAll(dir, 0700); err != nil {
					return nil, object.Errorf(i.osErr, "Maildir: %v", err)
				}
			}
		} else {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return nil, object.Errorf(noSuchErr, "no such mailbox: %s", path)
			}
		}

		st := &maildirState{path: path, factory: factory}
		i.wireMaildirInstance(self, st, cls, msgCls, noSuchErr, notEmptyErr)
		return object.None, nil
	}))

	return cls
}

// wireMaildirInstance places all Maildir methods into an instance's own dict.
func (i *Interp) wireMaildirInstance(self *object.Instance, st *maildirState, cls, msgCls *object.Class, noSuchErr, notEmptyErr *object.Class) {
	d := self.Dict

	bf := func(name string, fn func(any, []object.Object, *object.Dict) (object.Object, error)) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: fn}
	}

	ensureTOC := func() {
		st.mu.Lock()
		st.loadTOC()
		st.mu.Unlock()
	}
	_ = ensureTOC

	// add(message) -> key (str)
	d.SetStr("add", bf("add", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "add() requires message")
		}
		key := maildirKey()
		fname := filepath.Join(st.path, "new", key)
		raw := maildirSerializeMessage(a[0])
		if err := os.WriteFile(fname, []byte(raw), 0600); err != nil {
			return nil, object.Errorf(i.osErr, "Maildir add: %v", err)
		}
		st.mu.Lock()
		if st.toc == nil {
			st.toc = make(map[string]string)
		}
		st.toc[key] = "new/" + key
		st.mu.Unlock()
		return &object.Str{V: key}, nil
	}))

	// __getitem__(key) -> Message
	d.SetStr("__getitem__", bf("__getitem__", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "__getitem__ requires key")
		}
		key := object.Str_(a[0])
		st.mu.Lock()
		st.loadTOC()
		relPath, ok := st.toc[key]
		st.mu.Unlock()
		if !ok {
			return nil, object.Errorf(i.keyErr, "%s", key)
		}
		raw, err := os.ReadFile(filepath.Join(st.path, relPath))
		if err != nil {
			return nil, object.Errorf(i.osErr, "Maildir read: %v", err)
		}
		inst, err2 := i.emailParseString(string(raw), msgCls)
		if err2 != nil {
			return nil, err2
		}
		sub := "new"
		if strings.HasPrefix(relPath, "cur/") {
			sub = "cur"
		}
		inst.Dict.SetStr("_subdir", &object.Str{V: sub})
		return inst, nil
	}))

	// get(key, default=None)
	d.SetStr("get", bf("get", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var def object.Object = object.None
		if len(a) >= 2 {
			def = a[1]
		}
		if len(a) < 1 {
			return def, nil
		}
		key := object.Str_(a[0])
		st.mu.Lock()
		st.loadTOC()
		relPath, ok := st.toc[key]
		st.mu.Unlock()
		if !ok {
			return def, nil
		}
		raw, err := os.ReadFile(filepath.Join(st.path, relPath))
		if err != nil {
			return def, nil
		}
		inst, err2 := i.emailParseString(string(raw), msgCls)
		if err2 != nil {
			return def, nil
		}
		return inst, nil
	}))

	// get_message(key)
	d.SetStr("get_message", bf("get_message", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "get_message() requires key")
		}
		key := object.Str_(a[0])
		st.mu.Lock()
		st.loadTOC()
		relPath, ok := st.toc[key]
		st.mu.Unlock()
		if !ok {
			return nil, object.Errorf(i.keyErr, "%s", key)
		}
		raw, err := os.ReadFile(filepath.Join(st.path, relPath))
		if err != nil {
			return nil, object.Errorf(i.osErr, "Maildir read: %v", err)
		}
		return i.emailParseString(string(raw), msgCls)
	}))

	// get_string(key) -> str
	d.SetStr("get_string", bf("get_string", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "get_string() requires key")
		}
		key := object.Str_(a[0])
		st.mu.Lock()
		st.loadTOC()
		relPath, ok := st.toc[key]
		st.mu.Unlock()
		if !ok {
			return nil, object.Errorf(i.keyErr, "%s", key)
		}
		raw, err := os.ReadFile(filepath.Join(st.path, relPath))
		if err != nil {
			return nil, object.Errorf(i.osErr, "Maildir read: %v", err)
		}
		return &object.Str{V: string(raw)}, nil
	}))

	// get_bytes(key) -> bytes
	d.SetStr("get_bytes", bf("get_bytes", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "get_bytes() requires key")
		}
		key := object.Str_(a[0])
		st.mu.Lock()
		st.loadTOC()
		relPath, ok := st.toc[key]
		st.mu.Unlock()
		if !ok {
			return nil, object.Errorf(i.keyErr, "%s", key)
		}
		raw, err := os.ReadFile(filepath.Join(st.path, relPath))
		if err != nil {
			return nil, object.Errorf(i.osErr, "Maildir read: %v", err)
		}
		return &object.Bytes{V: raw}, nil
	}))

	// __setitem__(key, message)
	d.SetStr("__setitem__", bf("__setitem__", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "__setitem__ requires key and message")
		}
		key := object.Str_(a[0])
		st.mu.Lock()
		st.loadTOC()
		relPath, ok := st.toc[key]
		st.mu.Unlock()
		if !ok {
			return nil, object.Errorf(i.keyErr, "%s", key)
		}
		raw := maildirSerializeMessage(a[1])
		if err := os.WriteFile(filepath.Join(st.path, relPath), []byte(raw), 0600); err != nil {
			return nil, object.Errorf(i.osErr, "Maildir setitem: %v", err)
		}
		return object.None, nil
	}))

	// remove(key)
	d.SetStr("remove", bf("remove", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "remove() requires key")
		}
		key := object.Str_(a[0])
		st.mu.Lock()
		st.loadTOC()
		relPath, ok := st.toc[key]
		if ok {
			delete(st.toc, key)
		}
		st.mu.Unlock()
		if !ok {
			return nil, object.Errorf(i.keyErr, "%s", key)
		}
		os.Remove(filepath.Join(st.path, relPath))
		return object.None, nil
	}))

	// __delitem__(key)
	d.SetStr("__delitem__", bf("__delitem__", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "__delitem__ requires key")
		}
		key := object.Str_(a[0])
		st.mu.Lock()
		st.loadTOC()
		relPath, ok := st.toc[key]
		if ok {
			delete(st.toc, key)
		}
		st.mu.Unlock()
		if !ok {
			return nil, object.Errorf(i.keyErr, "%s", key)
		}
		os.Remove(filepath.Join(st.path, relPath))
		return object.None, nil
	}))

	// discard(key) -- no error if absent
	d.SetStr("discard", bf("discard", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		key := object.Str_(a[0])
		st.mu.Lock()
		st.loadTOC()
		relPath, ok := st.toc[key]
		if ok {
			delete(st.toc, key)
		}
		st.mu.Unlock()
		if ok {
			os.Remove(filepath.Join(st.path, relPath))
		}
		return object.None, nil
	}))

	// __contains__(key) -> bool
	d.SetStr("__contains__", bf("__contains__", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.BoolOf(false), nil
		}
		key := object.Str_(a[0])
		st.mu.Lock()
		st.loadTOC()
		_, ok := st.toc[key]
		st.mu.Unlock()
		return object.BoolOf(ok), nil
	}))

	// __len__() -> int
	d.SetStr("__len__", bf("__len__", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		st.mu.Lock()
		st.loadTOC()
		n := len(st.toc)
		st.mu.Unlock()
		return object.IntFromInt64(int64(n)), nil
	}))

	// keys() -> list
	d.SetStr("keys", bf("keys", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		st.mu.Lock()
		st.loadTOC()
		ks := make([]object.Object, 0, len(st.toc))
		for k := range st.toc {
			ks = append(ks, &object.Str{V: k})
		}
		st.mu.Unlock()
		return &object.List{V: ks}, nil
	}))

	// values() -> list of Message
	d.SetStr("values", bf("values", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		st.mu.Lock()
		st.loadTOC()
		toc := make(map[string]string, len(st.toc))
		for k, v := range st.toc {
			toc[k] = v
		}
		st.mu.Unlock()
		var msgs []object.Object
		for _, relPath := range toc {
			raw, err := os.ReadFile(filepath.Join(st.path, relPath))
			if err != nil {
				continue
			}
			msg, err2 := i.emailParseString(string(raw), msgCls)
			if err2 != nil {
				continue
			}
			msgs = append(msgs, msg)
		}
		return &object.List{V: msgs}, nil
	}))

	// items() -> list of (key, Message)
	d.SetStr("items", bf("items", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		st.mu.Lock()
		st.loadTOC()
		toc := make(map[string]string, len(st.toc))
		for k, v := range st.toc {
			toc[k] = v
		}
		st.mu.Unlock()
		var items []object.Object
		for k, relPath := range toc {
			raw, err := os.ReadFile(filepath.Join(st.path, relPath))
			if err != nil {
				continue
			}
			msg, err2 := i.emailParseString(string(raw), msgCls)
			if err2 != nil {
				continue
			}
			items = append(items, &object.Tuple{V: []object.Object{&object.Str{V: k}, msg}})
		}
		return &object.List{V: items}, nil
	}))

	// __iter__()
	d.SetStr("__iter__", bf("__iter__", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		st.mu.Lock()
		st.loadTOC()
		relPaths := make([]string, 0, len(st.toc))
		for _, v := range st.toc {
			relPaths = append(relPaths, v)
		}
		st.mu.Unlock()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			for idx < len(relPaths) {
				rp := relPaths[idx]
				idx++
				raw, err := os.ReadFile(filepath.Join(st.path, rp))
				if err != nil {
					continue
				}
				msg, err2 := i.emailParseString(string(raw), msgCls)
				if err2 != nil {
					continue
				}
				return msg, true, nil
			}
			return nil, false, nil
		}}, nil
	}))

	// flush() -- noop
	d.SetStr("flush", bf("flush", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}))

	// lock / unlock -- noop
	d.SetStr("lock", bf("lock", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}))
	d.SetStr("unlock", bf("unlock", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}))

	// close()
	d.SetStr("close", bf("close", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}))

	// clean() -- remove tmp files older than 36h
	d.SetStr("clean", bf("clean", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		cutoff := time.Now().Add(-36 * time.Hour)
		tmpDir := filepath.Join(st.path, "tmp")
		entries, err := os.ReadDir(tmpDir)
		if err != nil {
			return object.None, nil
		}
		for _, e := range entries {
			info, err2 := e.Info()
			if err2 != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(tmpDir, e.Name()))
			}
		}
		return object.None, nil
	}))

	// list_folders() -> list
	d.SetStr("list_folders", bf("list_folders", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		entries, err := os.ReadDir(st.path)
		if err != nil {
			return &object.List{}, nil
		}
		var folders []object.Object
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), ".") {
				folders = append(folders, &object.Str{V: e.Name()[1:]})
			}
		}
		return &object.List{V: folders}, nil
	}))

	// get_folder(folder) -> Maildir
	d.SetStr("get_folder", bf("get_folder", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "get_folder() requires folder name")
		}
		folder := object.Str_(a[0])
		folderPath := filepath.Join(st.path, "."+folder)
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			return nil, object.Errorf(noSuchErr, "no such folder: %s", folder)
		}
		folderSt := &maildirState{path: folderPath, factory: st.factory}
		inst := &object.Instance{Class: cls, Dict: object.NewDict()}
		i.wireMaildirInstance(inst, folderSt, cls, msgCls, noSuchErr, notEmptyErr)
		return inst, nil
	}))

	// add_folder(folder) -> Maildir
	d.SetStr("add_folder", bf("add_folder", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "add_folder() requires folder name")
		}
		folder := object.Str_(a[0])
		folderPath := filepath.Join(st.path, "."+folder)
		for _, sub := range []string{"", "new", "cur", "tmp"} {
			dir := filepath.Join(folderPath, sub)
			if err := os.MkdirAll(dir, 0700); err != nil {
				return nil, object.Errorf(i.osErr, "add_folder: %v", err)
			}
		}
		folderSt := &maildirState{path: folderPath, factory: st.factory}
		inst := &object.Instance{Class: cls, Dict: object.NewDict()}
		i.wireMaildirInstance(inst, folderSt, cls, msgCls, noSuchErr, notEmptyErr)
		return inst, nil
	}))

	// remove_folder(folder)
	d.SetStr("remove_folder", bf("remove_folder", func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "remove_folder() requires folder name")
		}
		folder := object.Str_(a[0])
		folderPath := filepath.Join(st.path, "."+folder)
		for _, sub := range []string{"new", "cur"} {
			entries, err := os.ReadDir(filepath.Join(folderPath, sub))
			if err == nil && len(entries) > 0 {
				return nil, object.Errorf(notEmptyErr, "folder not empty: %s", folder)
			}
		}
		os.RemoveAll(folderPath)
		return object.None, nil
	}))

	// __enter__ / __exit__
	d.SetStr("__enter__", bf("__enter__", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return self, nil
	}))
	d.SetStr("__exit__", bf("__exit__", func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}))
}

package vm

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildLoggingHandlers() *object.Module {
	m := &object.Module{Name: "logging.handlers", Dict: object.NewDict()}

	ls, loggingMod, err := i.ensureLogState()
	if err != nil {
		// Return empty module on error; logging module isn't loaded yet.
		return m
	}

	handlerCls := getLoggingClass(loggingMod, "Handler")
	fileHandlerCls := getLoggingClass(loggingMod, "FileHandler")
	if handlerCls == nil || fileHandlerCls == nil {
		return m
	}

	// --- BaseRotatingHandler ---
	baseRotCls := i.buildBaseRotatingHandlerClass(ls, handlerCls, fileHandlerCls)
	m.Dict.SetStr("BaseRotatingHandler", baseRotCls)

	// --- RotatingFileHandler ---
	rotCls := i.buildRotatingFileHandlerClass(ls, handlerCls, fileHandlerCls, baseRotCls)
	m.Dict.SetStr("RotatingFileHandler", rotCls)

	// --- TimedRotatingFileHandler ---
	timedCls := i.buildTimedRotatingFileHandlerClass(ls, handlerCls, fileHandlerCls, baseRotCls)
	m.Dict.SetStr("TimedRotatingFileHandler", timedCls)

	// --- WatchedFileHandler (alias of FileHandler on non-Unix) ---
	watchedCls := i.buildWatchedFileHandlerClass(ls, handlerCls, fileHandlerCls)
	m.Dict.SetStr("WatchedFileHandler", watchedCls)

	// --- BufferingHandler ---
	bufCls := i.buildBufferingHandlerClass(ls, handlerCls)
	m.Dict.SetStr("BufferingHandler", bufCls)

	// --- MemoryHandler ---
	memCls := i.buildMemoryHandlerClass(ls, handlerCls, bufCls)
	m.Dict.SetStr("MemoryHandler", memCls)

	// --- QueueHandler ---
	qhCls := i.buildQueueHandlerClass(ls, handlerCls)
	m.Dict.SetStr("QueueHandler", qhCls)

	// --- QueueListener ---
	qlCls := i.buildQueueListenerClass(ls)
	m.Dict.SetStr("QueueListener", qlCls)

	// --- Stub handlers ---
	for _, name := range []string{"SocketHandler", "DatagramHandler",
		"NTEventLogHandler", "SMTPHandler", "HTTPHandler"} {
		name := name
		stubCls := i.buildStubHandlerClass(name, handlerCls)
		m.Dict.SetStr(name, stubCls)
	}

	// --- SysLogHandler: stub + syslog facility/priority class attributes ---
	syslogCls := i.buildStubHandlerClass("SysLogHandler", handlerCls)
	syslogConsts := map[string]int64{
		// Priorities
		"LOG_EMERG": 0, "LOG_ALERT": 1, "LOG_CRIT": 2, "LOG_ERR": 3,
		"LOG_WARNING": 4, "LOG_NOTICE": 5, "LOG_INFO": 6, "LOG_DEBUG": 7,
		// Facilities
		"LOG_KERN": 0, "LOG_USER": 1, "LOG_MAIL": 2, "LOG_DAEMON": 3,
		"LOG_AUTH": 4, "LOG_LPR": 6, "LOG_NEWS": 7, "LOG_UUCP": 8,
		"LOG_CRON": 9, "LOG_SYSLOG": 15,
		"LOG_LOCAL0": 16, "LOG_LOCAL1": 17, "LOG_LOCAL2": 18, "LOG_LOCAL3": 19,
		"LOG_LOCAL4": 20, "LOG_LOCAL5": 21, "LOG_LOCAL6": 22, "LOG_LOCAL7": 23,
	}
	for k, v := range syslogConsts {
		syslogCls.Dict.SetStr(k, object.NewInt(v))
	}
	m.Dict.SetStr("SysLogHandler", syslogCls)
	m.Dict.SetStr("DEFAULT_TCP_LOGGING_PORT", object.NewInt(9020))
	m.Dict.SetStr("DEFAULT_UDP_LOGGING_PORT", object.NewInt(9021))
	m.Dict.SetStr("DEFAULT_HTTP_LOGGING_PORT", object.NewInt(9022))
	m.Dict.SetStr("DEFAULT_SOAP_LOGGING_PORT", object.NewInt(9023))
	m.Dict.SetStr("SYSLOG_UDP_PORT", object.NewInt(514))

	_ = ls
	return m
}

// handlerMethodsFromBase copies the standard handler methods from handlerCls.
func handlerMethodsFromBase(cls, handlerCls *object.Class) {
	for _, name := range []string{"setLevel", "setFormatter", "format", "handle",
		"flush", "close", "addFilter", "removeFilter", "filter"} {
		cls.Dict.SetStr(name, mustGetStr(handlerCls.Dict, name))
	}
}

// newBaseHandlerDict initialises the standard handler instance fields.
func newBaseHandlerDict() *object.Dict {
	d := object.NewDict()
	d.SetStr("level", object.NewInt(logNOTSET))
	d.SetStr("formatter", object.None)
	d.SetStr("filters", &object.List{V: nil})
	return d
}

// openRotFile opens a file for rotation handlers (mode a or w).
func openRotFile(filename, mode string) (*os.File, error) {
	flag := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	if mode == "w" {
		flag = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
	}
	return os.OpenFile(filename, flag, 0644)
}

// makeRotFileEmit builds the _emit BuiltinFunc that writes to a *os.File pointer.
// The pointer itself is indirected so doRollover can swap the underlying file.
func makeRotFileEmit(fp **os.File) *object.BuiltinFunc {
	return &object.BuiltinFunc{Name: "_emit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if *fp == nil {
			return object.None, nil
		}
		msg := ""
		if len(a) >= 1 {
			if s, ok := a[0].(*object.Str); ok {
				msg = s.V
			}
		}
		(*fp).WriteString(msg + "\n") //nolint
		return object.None, nil
	}}
}

// --- BaseRotatingHandler ---

func (i *Interp) buildBaseRotatingHandlerClass(ls *logState, handlerCls, fileHandlerCls *object.Class) *object.Class {
	cls := &object.Class{Name: "BaseRotatingHandler", Bases: []*object.Class{handlerCls}, Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, fmt.Errorf("BaseRotatingHandler requires filename")
		}
		self := a[0].(*object.Instance)
		filename, ok := a[1].(*object.Str)
		if !ok {
			return nil, fmt.Errorf("BaseRotatingHandler: filename must be str")
		}
		mode := "a"
		if len(a) >= 3 {
			if s, ok2 := a[2].(*object.Str); ok2 {
				mode = s.V
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("mode"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					mode = s.V
				}
			}
		}
		f, err := openRotFile(filename.V, mode)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		self.Dict = newBaseHandlerDict()
		self.Dict.SetStr("baseFilename", filename)
		self.Dict.SetStr("mode", &object.Str{V: mode})
		fp := f
		self.Dict.SetStr("_emit", makeRotFileEmit(&fp))
		self.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if fp != nil {
				fp.Close()
			}
			return object.None, nil
		}})
		return object.None, nil
	}})
	handlerMethodsFromBase(cls, handlerCls)
	return cls
}

// --- RotatingFileHandler ---

func (i *Interp) buildRotatingFileHandlerClass(ls *logState, handlerCls, fileHandlerCls, baseRotCls *object.Class) *object.Class {
	cls := &object.Class{Name: "RotatingFileHandler", Bases: []*object.Class{handlerCls}, Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, fmt.Errorf("RotatingFileHandler requires filename")
		}
		self := a[0].(*object.Instance)
		filename, ok := a[1].(*object.Str)
		if !ok {
			return nil, fmt.Errorf("RotatingFileHandler: filename must be str")
		}

		mode := "a"
		maxBytes := int64(0)
		backupCount := int64(0)

		if len(a) >= 3 {
			if s, ok2 := a[2].(*object.Str); ok2 {
				mode = s.V
			}
		}
		if len(a) >= 4 {
			if n, ok2 := toInt64(a[3]); ok2 {
				maxBytes = n
			}
		}
		if len(a) >= 5 {
			if n, ok2 := toInt64(a[4]); ok2 {
				backupCount = n
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("mode"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					mode = s.V
				}
			}
			if v, ok2 := kw.GetStr("maxBytes"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					maxBytes = n
				}
			}
			if v, ok2 := kw.GetStr("backupCount"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					backupCount = n
				}
			}
		}

		f, err := openRotFile(filename.V, mode)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}

		self.Dict = newBaseHandlerDict()
		self.Dict.SetStr("baseFilename", filename)
		self.Dict.SetStr("mode", &object.Str{V: mode})
		self.Dict.SetStr("maxBytes", object.NewInt(maxBytes))
		self.Dict.SetStr("backupCount", object.NewInt(backupCount))

		fp := f
		self.Dict.SetStr("_emit", makeRotFileEmit(&fp))

		// doRollover implementation: rename .log.N → .log.N+1, then .log → .log.1
		doRollover := func() error {
			if fp != nil {
				fp.Close()
				fp = nil
			}
			bc := int(backupCount)
			fname := filename.V
			if bc > 0 {
				for k := bc; k >= 1; k-- {
					src := fmt.Sprintf("%s.%d", fname, k)
					dst := fmt.Sprintf("%s.%d", fname, k+1)
					if _, err2 := os.Stat(src); err2 == nil {
						if k == bc {
							os.Remove(dst) //nolint
						}
						os.Rename(src, dst) //nolint
					}
				}
				os.Rename(fname, fname+".1") //nolint
			}
			nf, err2 := openRotFile(fname, "w")
			if err2 != nil {
				return err2
			}
			fp = nf
			return nil
		}

		self.Dict.SetStr("doRollover", &object.BuiltinFunc{Name: "doRollover", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, doRollover()
		}})

		self.Dict.SetStr("shouldRollover", &object.BuiltinFunc{Name: "shouldRollover", Call: func(_ any, a2 []object.Object, _ *object.Dict) (object.Object, error) {
			if maxBytes <= 0 {
				return object.False, nil
			}
			if fp == nil {
				return object.False, nil
			}
			info, err2 := fp.Stat()
			if err2 != nil {
				return object.False, nil
			}
			// Message length approximation: get formatted message from record if possible.
			msgLen := int64(0)
			if len(a2) >= 2 {
				if rec, ok2 := a2[1].(*object.Instance); ok2 {
					if mv, ok3 := rec.Dict.GetStr("message"); ok3 {
						if ms, ok4 := mv.(*object.Str); ok4 {
							msgLen = int64(len(ms.V))
						}
					}
				}
			}
			return object.BoolOf(info.Size()+msgLen >= maxBytes), nil
		}})

		self.Dict.SetStr("emit", &object.BuiltinFunc{Name: "emit", Call: func(_ any, a2 []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a2) < 2 {
				return object.None, nil
			}
			rec := a2[1].(*object.Instance)
			// shouldRollover check
			if maxBytes > 0 && fp != nil {
				info, _ := fp.Stat()
				if info != nil && info.Size() >= maxBytes {
					if err2 := doRollover(); err2 != nil {
						return object.None, nil
					}
				}
			}
			msg := formatRecord(ls, a2[0].(*object.Instance), rec)
			if fp != nil {
				fp.WriteString(msg + "\n") //nolint
			}
			return object.None, nil
		}})

		self.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if fp != nil {
				fp.Close()
				fp = nil
			}
			return object.None, nil
		}})

		return object.None, nil
	}})

	handlerMethodsFromBase(cls, handlerCls)
	return cls
}

// --- TimedRotatingFileHandler ---

func (i *Interp) buildTimedRotatingFileHandlerClass(ls *logState, handlerCls, fileHandlerCls, baseRotCls *object.Class) *object.Class {
	cls := &object.Class{Name: "TimedRotatingFileHandler", Bases: []*object.Class{handlerCls}, Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, fmt.Errorf("TimedRotatingFileHandler requires filename")
		}
		self := a[0].(*object.Instance)
		filename, ok := a[1].(*object.Str)
		if !ok {
			return nil, fmt.Errorf("TimedRotatingFileHandler: filename must be str")
		}

		when := "h"
		interval := int64(1)
		backupCount := int64(0)
		utc := false

		if len(a) >= 3 {
			if s, ok2 := a[2].(*object.Str); ok2 {
				when = s.V
			}
		}
		if len(a) >= 4 {
			if n, ok2 := toInt64(a[3]); ok2 {
				interval = n
			}
		}
		if len(a) >= 5 {
			if n, ok2 := toInt64(a[4]); ok2 {
				backupCount = n
			}
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("when"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					when = s.V
				}
			}
			if v, ok2 := kw.GetStr("interval"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					interval = n
				}
			}
			if v, ok2 := kw.GetStr("backupCount"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					backupCount = n
				}
			}
			if v, ok2 := kw.GetStr("utc"); ok2 {
				utc = object.Truthy(v)
			}
		}

		// Compute rollover interval in seconds.
		var rolloverSecs int64
		whenUp := strings.ToUpper(when)
		switch whenUp {
		case "S":
			rolloverSecs = interval
		case "M":
			rolloverSecs = interval * 60
		case "H":
			rolloverSecs = interval * 3600
		case "D", "MIDNIGHT":
			rolloverSecs = interval * 86400
		default:
			rolloverSecs = interval * 3600
		}

		now := time.Now()
		if utc {
			now = now.UTC()
		}
		rolloverAt := now.Unix() + rolloverSecs

		f, err := openRotFile(filename.V, "a")
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}

		self.Dict = newBaseHandlerDict()
		self.Dict.SetStr("baseFilename", filename)
		self.Dict.SetStr("when", &object.Str{V: whenUp})
		self.Dict.SetStr("interval", object.NewInt(rolloverSecs))
		self.Dict.SetStr("backupCount", object.NewInt(backupCount))
		self.Dict.SetStr("rolloverAt", object.NewInt(rolloverAt))
		self.Dict.SetStr("utc", object.BoolOf(utc))

		fp := f

		self.Dict.SetStr("_emit", makeRotFileEmit(&fp))

		doRollover := func() error {
			if fp != nil {
				fp.Close()
				fp = nil
			}
			fname := filename.V
			// Suffix format: .YYYY-MM-DD_HH-MM-SS
			t := time.Now()
			if utc {
				t = t.UTC()
			}
			suffix := t.Format("2006-01-02_15-04-05")
			dst := fname + "." + suffix
			os.Rename(fname, dst) //nolint

			// Delete old backups if backupCount > 0.
			if backupCount > 0 {
				// List files with same prefix + dot
				dir := "."
				base := fname
				if idx := strings.LastIndex(fname, "/"); idx >= 0 {
					dir = fname[:idx]
					base = fname[idx+1:]
				}
				entries, err2 := os.ReadDir(dir)
				if err2 == nil {
					var oldFiles []string
					for _, e := range entries {
						n := e.Name()
						if strings.HasPrefix(n, base+".") {
							oldFiles = append(oldFiles, n)
						}
					}
					if int64(len(oldFiles)) > backupCount {
						for _, of := range oldFiles[:int64(len(oldFiles))-backupCount] {
							os.Remove(dir + "/" + of) //nolint
						}
					}
				}
			}

			nf, err2 := openRotFile(fname, "a")
			if err2 != nil {
				return err2
			}
			fp = nf
			// Compute next rollover.
			now2 := time.Now()
			if utc {
				now2 = now2.UTC()
			}
			rolloverAt = now2.Unix() + rolloverSecs
			self.Dict.SetStr("rolloverAt", object.NewInt(rolloverAt))
			return nil
		}

		self.Dict.SetStr("doRollover", &object.BuiltinFunc{Name: "doRollover", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, doRollover()
		}})

		self.Dict.SetStr("shouldRollover", &object.BuiltinFunc{Name: "shouldRollover", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			t := time.Now()
			if utc {
				t = t.UTC()
			}
			return object.BoolOf(t.Unix() >= rolloverAt), nil
		}})

		self.Dict.SetStr("emit", &object.BuiltinFunc{Name: "emit", Call: func(_ any, a2 []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a2) < 2 {
				return object.None, nil
			}
			rec := a2[1].(*object.Instance)
			t := time.Now()
			if utc {
				t = t.UTC()
			}
			if t.Unix() >= rolloverAt {
				doRollover() //nolint
			}
			msg := formatRecord(ls, a2[0].(*object.Instance), rec)
			if fp != nil {
				fp.WriteString(msg + "\n") //nolint
			}
			return object.None, nil
		}})

		self.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if fp != nil {
				fp.Close()
				fp = nil
			}
			return object.None, nil
		}})

		return object.None, nil
	}})

	handlerMethodsFromBase(cls, handlerCls)
	return cls
}

// --- WatchedFileHandler (simplified: behaves like FileHandler on non-Unix) ---

func (i *Interp) buildWatchedFileHandlerClass(ls *logState, handlerCls, fileHandlerCls *object.Class) *object.Class {
	cls := &object.Class{Name: "WatchedFileHandler", Bases: []*object.Class{handlerCls}, Dict: object.NewDict()}
	// Reuse FileHandler's __init__ by delegating.
	initFn, _ := fileHandlerCls.Dict.GetStr("__init__")
	cls.Dict.SetStr("__init__", initFn)
	cls.Dict.SetStr("reopenIfNeeded", &object.BuiltinFunc{Name: "reopenIfNeeded", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	handlerMethodsFromBase(cls, handlerCls)
	return cls
}

// --- BufferingHandler ---

func (i *Interp) buildBufferingHandlerClass(ls *logState, handlerCls *object.Class) *object.Class {
	cls := &object.Class{Name: "BufferingHandler", Bases: []*object.Class{handlerCls}, Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("BufferingHandler requires capacity")
		}
		self := a[0].(*object.Instance)
		var capVal object.Object
		if len(a) >= 2 {
			capVal = a[1]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("capacity"); ok2 {
				capVal = v
			}
		}
		if capVal == nil {
			return nil, fmt.Errorf("BufferingHandler requires capacity")
		}
		capacity, ok := toInt64(capVal)
		if !ok {
			return nil, fmt.Errorf("BufferingHandler: capacity must be int")
		}
		self.Dict = newBaseHandlerDict()
		self.Dict.SetStr("capacity", object.NewInt(capacity))
		self.Dict.SetStr("buffer", &object.List{V: nil})
		self.Dict.SetStr("emit", &object.BuiltinFunc{Name: "emit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self2 := a[0].(*object.Instance)
			rec := a[1]
			if buf, ok := self2.Dict.GetStr("buffer"); ok {
				if lst, ok2 := buf.(*object.List); ok2 {
					lst.V = append(lst.V, rec)
					cap2, _ := toInt64(mustGetStr(self2.Dict, "capacity"))
					if int64(len(lst.V)) >= cap2 {
						lst.V = nil
					}
				}
			}
			return object.None, nil
		}})
		return object.None, nil
	}})

	cls.Dict.SetStr("shouldFlush", &object.BuiltinFunc{Name: "shouldFlush", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.False, nil
		}
		self := a[0].(*object.Instance)
		cap, _ := toInt64(mustGetStr(self.Dict, "capacity"))
		buf, _ := self.Dict.GetStr("buffer")
		if lst, ok := buf.(*object.List); ok {
			return object.BoolOf(int64(len(lst.V)) >= cap), nil
		}
		return object.False, nil
	}})

	cls.Dict.SetStr("flush", &object.BuiltinFunc{Name: "flush", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if buf, ok := self.Dict.GetStr("buffer"); ok {
			if lst, ok2 := buf.(*object.List); ok2 {
				lst.V = nil
			}
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("close", mustGetStr(handlerCls.Dict, "close"))
	handlerMethodsFromBase(cls, handlerCls)
	return cls
}

// --- MemoryHandler ---

func (i *Interp) buildMemoryHandlerClass(ls *logState, handlerCls, bufCls *object.Class) *object.Class {
	cls := &object.Class{Name: "MemoryHandler", Bases: []*object.Class{handlerCls}, Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, fmt.Errorf("MemoryHandler requires capacity")
		}
		self := a[0].(*object.Instance)
		var capVal object.Object
		if len(a) >= 2 {
			capVal = a[1]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("capacity"); ok2 {
				capVal = v
			}
		}
		if capVal == nil {
			return nil, fmt.Errorf("MemoryHandler requires capacity")
		}
		capacity, ok := toInt64(capVal)
		if !ok {
			return nil, fmt.Errorf("MemoryHandler: capacity must be int")
		}

		flushLevel := int64(logERROR)
		var target object.Object = object.None
		flushOnClose := true

		if len(a) >= 3 {
			if n, ok2 := toInt64(a[2]); ok2 {
				flushLevel = n
			}
		}
		if len(a) >= 4 {
			target = a[3]
		}
		if len(a) >= 5 {
			flushOnClose = object.Truthy(a[4])
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("flushLevel"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					flushLevel = n
				}
			}
			if v, ok2 := kw.GetStr("target"); ok2 {
				target = v
			}
			if v, ok2 := kw.GetStr("flushOnClose"); ok2 {
				flushOnClose = object.Truthy(v)
			}
		}

		self.Dict = newBaseHandlerDict()
		self.Dict.SetStr("capacity", object.NewInt(capacity))
		self.Dict.SetStr("flushLevel", object.NewInt(flushLevel))
		self.Dict.SetStr("target", target)
		self.Dict.SetStr("flushOnClose", object.BoolOf(flushOnClose))
		self.Dict.SetStr("buffer", &object.List{V: nil})
		self.Dict.SetStr("emit", &object.BuiltinFunc{Name: "emit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self2 := a[0].(*object.Instance)
			rec := a[1].(*object.Instance)
			buf, _ := self2.Dict.GetStr("buffer")
			lst, _ := buf.(*object.List)
			if lst != nil {
				lst.V = append(lst.V, rec)
			}
			cap2, _ := toInt64(mustGetStr(self2.Dict, "capacity"))
			fl2, _ := toInt64(mustGetStr(self2.Dict, "flushLevel"))
			recLevel := int64(0)
			if lv, ok2 := rec.Dict.GetStr("levelno"); ok2 {
				recLevel, _ = toInt64(lv)
			}
			if lst != nil && (int64(len(lst.V)) >= cap2 || recLevel >= fl2) {
				i.memoryHandlerFlush(ls, self2, lst)
			}
			return object.None, nil
		}})
		return object.None, nil
	}})

	cls.Dict.SetStr("setTarget", &object.BuiltinFunc{Name: "setTarget", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		self.Dict.SetStr("target", a[1])
		return object.None, nil
	}})

	cls.Dict.SetStr("shouldFlush", &object.BuiltinFunc{Name: "shouldFlush", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		self := a[0].(*object.Instance)
		rec, ok := a[1].(*object.Instance)
		if !ok {
			return object.False, nil
		}
		cap, _ := toInt64(mustGetStr(self.Dict, "capacity"))
		fl, _ := toInt64(mustGetStr(self.Dict, "flushLevel"))
		buf, _ := self.Dict.GetStr("buffer")
		bufLen := int64(0)
		if lst, ok2 := buf.(*object.List); ok2 {
			bufLen = int64(len(lst.V))
		}
		recLevel := int64(0)
		if lv, ok2 := rec.Dict.GetStr("levelno"); ok2 {
			recLevel, _ = toInt64(lv)
		}
		return object.BoolOf(bufLen >= cap || recLevel >= fl), nil
	}})

	cls.Dict.SetStr("flush", &object.BuiltinFunc{Name: "flush", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		buf, _ := self.Dict.GetStr("buffer")
		if lst, ok := buf.(*object.List); ok {
			i.memoryHandlerFlush(ls, self, lst)
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if foc, ok := self.Dict.GetStr("flushOnClose"); ok && object.Truthy(foc) {
			if buf, ok2 := self.Dict.GetStr("buffer"); ok2 {
				if lst, ok3 := buf.(*object.List); ok3 {
					i.memoryHandlerFlush(ls, self, lst)
				}
			}
		}
		return object.None, nil
	}})

	handlerMethodsFromBase(cls, handlerCls)
	return cls
}

// memoryHandlerFlush sends buffered records to target and clears the buffer.
func (i *Interp) memoryHandlerFlush(ls *logState, self *object.Instance, buf *object.List) {
	if buf == nil || len(buf.V) == 0 {
		return
	}
	tgt, _ := self.Dict.GetStr("target")
	if tgt != nil && tgt != object.None {
		if hdlr, ok := tgt.(*object.Instance); ok {
			for _, rec := range buf.V {
				if r, ok2 := rec.(*object.Instance); ok2 {
					i.handlerHandle(ls, hdlr, r)
				}
			}
		}
	}
	buf.V = nil
}

// --- QueueHandler ---

func (i *Interp) buildQueueHandlerClass(ls *logState, handlerCls *object.Class) *object.Class {
	cls := &object.Class{Name: "QueueHandler", Bases: []*object.Class{handlerCls}, Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, fmt.Errorf("QueueHandler requires queue")
		}
		self := a[0].(*object.Instance)
		self.Dict = newBaseHandlerDict()
		self.Dict.SetStr("queue", a[1])
		self.Dict.SetStr("listener", object.None)
		// emit is set on the instance so handlerHandle's instance-emit check fires.
		self.Dict.SetStr("emit", &object.BuiltinFunc{Name: "emit", Call: func(_ any, a2 []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a2) < 2 {
				return object.None, nil
			}
			self2 := a2[0].(*object.Instance)
			rec := a2[1].(*object.Instance)
			// prepare: merge message
			msg := ""
			if mv, ok2 := rec.Dict.GetStr("msg"); ok2 {
				if ms, ok3 := mv.(*object.Str); ok3 {
					msg = ms.V
				}
			}
			if argsV, ok2 := rec.Dict.GetStr("args"); ok2 && argsV != object.None {
				if tpl, ok3 := argsV.(*object.Tuple); ok3 {
					msg = logFormatMsg(msg, tpl.V)
				}
			}
			rec.Dict.SetStr("message", &object.Str{V: msg})
			rec.Dict.SetStr("msg", &object.Str{V: msg})
			rec.Dict.SetStr("args", object.None)
			// enqueue via put_nowait (supports Python-defined queues)
			if qv, ok2 := self2.Dict.GetStr("queue"); ok2 {
				if q, ok3 := qv.(*object.Instance); ok3 {
					fn, _ := q.Dict.GetStr("put_nowait")
					if fn == nil {
						fn, _ = classLookup(q.Class, "put_nowait")
					}
					if fn != nil {
						i.callObject(fn, []object.Object{q, rec}, nil) //nolint
					}
				}
			}
			return object.None, nil
		}})
		return object.None, nil
	}})

	cls.Dict.SetStr("prepare", &object.BuiltinFunc{Name: "prepare", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		rec, ok := a[1].(*object.Instance)
		if !ok {
			return a[1], nil
		}
		msg := ""
		if mv, ok2 := rec.Dict.GetStr("msg"); ok2 {
			if ms, ok3 := mv.(*object.Str); ok3 {
				msg = ms.V
			}
		}
		if argsV, ok2 := rec.Dict.GetStr("args"); ok2 && argsV != object.None {
			if tpl, ok3 := argsV.(*object.Tuple); ok3 && len(tpl.V) > 0 {
				msg = logFormatMsg(msg, tpl.V)
			}
		}
		rec.Dict.SetStr("message", &object.Str{V: msg})
		rec.Dict.SetStr("msg", &object.Str{V: msg})
		rec.Dict.SetStr("args", object.None)
		return rec, nil
	}})

	cls.Dict.SetStr("enqueue", &object.BuiltinFunc{Name: "enqueue", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		rec := a[1]
		if qv, ok := self.Dict.GetStr("queue"); ok {
			if q, ok2 := qv.(*object.Instance); ok2 {
				fn, _ := q.Dict.GetStr("put_nowait")
				if fn == nil {
					fn, _ = classLookup(q.Class, "put_nowait")
				}
				if fn != nil {
					i.callObject(fn, []object.Object{q, rec}, nil) //nolint
				}
			}
		}
		return object.None, nil
	}})

	handlerMethodsFromBase(cls, handlerCls)
	return cls
}

// --- QueueListener ---
// Synchronous implementation: start() marks running, stop() drains the queue.
// No goroutines — safe for the single-threaded goipy interpreter.

func (i *Interp) buildQueueListenerClass(ls *logState) *object.Class {
	cls := &object.Class{Name: "QueueListener", Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, fmt.Errorf("QueueListener requires queue")
		}
		self := a[0].(*object.Instance)
		queue := a[1]
		handlers := a[2:]
		respectLevel := false
		if kw != nil {
			if v, ok := kw.GetStr("respect_handler_level"); ok {
				respectLevel = object.Truthy(v)
			}
		}
		self.Dict.SetStr("queue", queue)
		self.Dict.SetStr("handlers", &object.List{V: handlers})
		self.Dict.SetStr("respect_handler_level", object.BoolOf(respectLevel))
		self.Dict.SetStr("_running", object.False)
		return object.None, nil
	}})

	// handle dispatches a record to all handlers.
	handleFn := &object.BuiltinFunc{Name: "handle", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		rec, ok := a[1].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		respectLevel := false
		if rv, ok2 := self.Dict.GetStr("respect_handler_level"); ok2 {
			respectLevel = object.Truthy(rv)
		}
		if hv, ok2 := self.Dict.GetStr("handlers"); ok2 {
			if lst, ok3 := hv.(*object.List); ok3 {
				for _, h := range lst.V {
					if hdlr, ok4 := h.(*object.Instance); ok4 {
						if respectLevel {
							hlevel := int64(logNOTSET)
							if lv, ok5 := hdlr.Dict.GetStr("level"); ok5 {
								hlevel, _ = toInt64(lv)
							}
							reclevel := int64(logNOTSET)
							if lv, ok5 := rec.Dict.GetStr("levelno"); ok5 {
								reclevel, _ = toInt64(lv)
							}
							if reclevel < hlevel {
								continue
							}
						}
						i.handlerHandle(ls, hdlr, rec)
					}
				}
			}
		}
		return object.None, nil
	}}
	cls.Dict.SetStr("handle", handleFn)

	// start: mark running.
	cls.Dict.SetStr("start", &object.BuiltinFunc{Name: "start", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if rv, ok := self.Dict.GetStr("_running"); ok && object.Truthy(rv) {
			return nil, object.Errorf(i.runtimeErr, "QueueListener already running")
		}
		self.Dict.SetStr("_running", object.True)
		return object.None, nil
	}})

	// stop: drain all records from queue synchronously, then mark stopped.
	cls.Dict.SetStr("stop", &object.BuiltinFunc{Name: "stop", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		// Drain queue synchronously.
		i.queueListenerDrain(self, handleFn)
		self.Dict.SetStr("_running", object.False)
		return object.None, nil
	}})

	cls.Dict.SetStr("dequeue", &object.BuiltinFunc{Name: "dequeue", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		return i.queueGet(self), nil
	}})

	return cls
}

// queueListenerDrain pulls all items from the queue and dispatches them.
func (i *Interp) queueListenerDrain(self *object.Instance, handleFn *object.BuiltinFunc) {
	for {
		item := i.queueGetNowait(self)
		if item == nil {
			break
		}
		if item == object.None {
			break // sentinel
		}
		if rec, ok := item.(*object.Instance); ok {
			handleFn.Call(nil, []object.Object{self, rec}, nil) //nolint
		}
	}
}

// queueGetNowait calls queue.get_nowait(); returns nil on empty/error.
func (i *Interp) queueGetNowait(self *object.Instance) object.Object {
	qv, ok := self.Dict.GetStr("queue")
	if !ok {
		return nil
	}
	q, ok := qv.(*object.Instance)
	if !ok {
		return nil
	}
	fn, _ := q.Dict.GetStr("get_nowait")
	if fn == nil {
		fn, _ = classLookup(q.Class, "get_nowait")
	}
	if fn == nil {
		return nil
	}
	item, err := i.callObject(fn, []object.Object{q}, nil)
	if err != nil {
		return nil // empty queue raises, caught here
	}
	return item
}

// queueGet calls queue.get(); returns None on error.
func (i *Interp) queueGet(self *object.Instance) object.Object {
	qv, ok := self.Dict.GetStr("queue")
	if !ok {
		return object.None
	}
	q, ok := qv.(*object.Instance)
	if !ok {
		return object.None
	}
	if fn, ok2 := q.Dict.GetStr("get"); ok2 {
		if bf, ok3 := fn.(*object.BuiltinFunc); ok3 {
			item, err := bf.Call(nil, []object.Object{q}, nil)
			if err != nil {
				return object.None
			}
			return item
		}
	}
	return object.None
}

// --- Stub handlers ---

func (i *Interp) buildStubHandlerClass(name string, handlerCls *object.Class) *object.Class {
	cls := &object.Class{Name: name, Bases: []*object.Class{handlerCls}, Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		self.Dict = newBaseHandlerDict()
		self.Dict.SetStr("_emit", &object.BuiltinFunc{Name: "_emit", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
		self.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
		return object.None, nil
	}})
	cls.Dict.SetStr("emit", &object.BuiltinFunc{Name: "emit", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	handlerMethodsFromBase(cls, handlerCls)
	return cls
}

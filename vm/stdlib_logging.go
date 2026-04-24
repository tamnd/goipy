package vm

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/tamnd/goipy/object"
)

// mustGetStr returns the value from a Dict or nil if missing. Used to share
// methods between handler class and its subclasses.
func mustGetStr(d *object.Dict, key string) object.Object {
	if v, ok := d.GetStr(key); ok {
		return v
	}
	return object.None
}

// --- level constants ---

const (
	logNOTSET   = 0
	logDEBUG    = 10
	logINFO     = 20
	logWARNING  = 30
	logERROR    = 40
	logCRITICAL = 50
)

var defaultLevelNames = map[int]string{
	logCRITICAL: "CRITICAL",
	logERROR:    "ERROR",
	logWARNING:  "WARNING",
	logINFO:     "INFO",
	logDEBUG:    "DEBUG",
	logNOTSET:   "NOTSET",
}

// logState holds per-module mutable state so we don't need global vars.
type logState struct {
	loggers        map[string]*object.Instance // name → Logger instance
	levelNames     map[int]string
	levelNamesRev  map[string]int
	manager        *object.Instance // root logger
	disabledLevel  int
	logRecordFactory object.Object
	loggerClass    object.Object
	startTime      float64
}

func newLogState() *logState {
	ls := &logState{
		loggers:       map[string]*object.Instance{},
		levelNames:    map[int]string{},
		levelNamesRev: map[string]int{},
		startTime:     float64(time.Now().UnixNano()) / 1e9,
	}
	for k, v := range defaultLevelNames {
		ls.levelNames[k] = v
		ls.levelNamesRev[v] = k
	}
	return ls
}

func (ls *logState) getLevelName(level int) string {
	if n, ok := ls.levelNames[level]; ok {
		return n
	}
	return fmt.Sprintf("Level %d", level)
}

func (ls *logState) getLevelNum(name string) (int, bool) {
	if n, ok := ls.levelNamesRev[name]; ok {
		return n, true
	}
	return 0, false
}

// --- buildLogging ---

func (i *Interp) buildLogging() *object.Module {
	m := &object.Module{Name: "logging", Dict: object.NewDict()}
	ls := newLogState()
	// Register ls so that logging.config can access it.
	if i.logStates == nil {
		i.logStates = map[string]*logState{}
	}
	i.logStates["logging"] = ls

	// --- level constants ---
	m.Dict.SetStr("CRITICAL", object.NewInt(logCRITICAL))
	m.Dict.SetStr("FATAL", object.NewInt(logCRITICAL))
	m.Dict.SetStr("ERROR", object.NewInt(logERROR))
	m.Dict.SetStr("WARNING", object.NewInt(logWARNING))
	m.Dict.SetStr("WARN", object.NewInt(logWARNING))
	m.Dict.SetStr("INFO", object.NewInt(logINFO))
	m.Dict.SetStr("DEBUG", object.NewInt(logDEBUG))
	m.Dict.SetStr("NOTSET", object.NewInt(logNOTSET))

	// --- class builders ---
	filterCls := i.buildFilterClass(ls)
	filtererCls := i.buildFiltererClass(ls)
	handlerCls := i.buildHandlerClass(ls, filtererCls)
	streamHandlerCls := i.buildStreamHandlerClass(ls, handlerCls)
	fileHandlerCls := i.buildFileHandlerClass(ls, handlerCls)
	nullHandlerCls := i.buildNullHandlerClass(handlerCls)
	formatterCls := i.buildLoggingFormatterClass(ls)
	logRecordCls := i.buildLogRecordClass(ls)
	loggerCls := i.buildLoggerClass(ls, filtererCls, logRecordCls)

	ls.loggerClass = loggerCls
	ls.logRecordFactory = &object.BuiltinFunc{Name: "LogRecord", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return makeLogRecordInst(ls, logRecordCls, a, kw)
	}}

	m.Dict.SetStr("Filter", filterCls)
	m.Dict.SetStr("Filterer", filtererCls)
	m.Dict.SetStr("Handler", handlerCls)
	m.Dict.SetStr("StreamHandler", streamHandlerCls)
	m.Dict.SetStr("FileHandler", fileHandlerCls)
	m.Dict.SetStr("NullHandler", nullHandlerCls)
	m.Dict.SetStr("Formatter", formatterCls)
	m.Dict.SetStr("LogRecord", logRecordCls)
	m.Dict.SetStr("Logger", loggerCls)

	// create root logger
	rootLogger := newLoggerInst(ls, loggerCls, "root", logWARNING)
	ls.loggers["root"] = rootLogger
	ls.loggers[""] = rootLogger
	ls.manager = rootLogger
	m.Dict.SetStr("root", rootLogger)

	// --- getLevelName / addLevelName ---
	m.Dict.SetStr("getLevelName", &object.BuiltinFunc{Name: "getLevelName", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "getLevelName() requires 1 argument")
		}
		if n, ok := toInt64(a[0]); ok {
			return &object.Str{V: ls.getLevelName(int(n))}, nil
		}
		if s, ok := a[0].(*object.Str); ok {
			if n, ok2 := ls.getLevelNum(s.V); ok2 {
				return object.NewInt(int64(n)), nil
			}
			return &object.Str{V: "Level " + s.V}, nil
		}
		return nil, object.Errorf(i.typeErr, "getLevelName() argument must be int or str")
	}})

	m.Dict.SetStr("getLevelNamesMapping", &object.BuiltinFunc{Name: "getLevelNamesMapping", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		d := object.NewDict()
		for k, v := range ls.levelNamesRev {
			d.SetStr(k, object.NewInt(int64(v)))
		}
		return d, nil
	}})

	m.Dict.SetStr("addLevelName", &object.BuiltinFunc{Name: "addLevelName", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "addLevelName() requires 2 arguments")
		}
		n, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "addLevelName() level must be int")
		}
		s, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "addLevelName() name must be str")
		}
		ls.levelNames[int(n)] = s.V
		ls.levelNamesRev[s.V] = int(n)
		return object.None, nil
	}})

	// --- getLogger ---
	m.Dict.SetStr("getLogger", &object.BuiltinFunc{Name: "getLogger", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		name := ""
		if len(a) >= 1 && a[0] != object.None {
			if s, ok := a[0].(*object.Str); ok {
				name = s.V
			}
		}
		return i.getOrCreateLogger(ls, loggerCls, name), nil
	}})

	m.Dict.SetStr("getLoggerClass", &object.BuiltinFunc{Name: "getLoggerClass", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return ls.loggerClass, nil
	}})

	m.Dict.SetStr("setLoggerClass", &object.BuiltinFunc{Name: "setLoggerClass", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 1 {
			ls.loggerClass = a[0]
		}
		return object.None, nil
	}})

	m.Dict.SetStr("getLogRecordFactory", &object.BuiltinFunc{Name: "getLogRecordFactory", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return ls.logRecordFactory, nil
	}})

	m.Dict.SetStr("setLogRecordFactory", &object.BuiltinFunc{Name: "setLogRecordFactory", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 1 {
			ls.logRecordFactory = a[0]
		}
		return object.None, nil
	}})

	// --- disable ---
	m.Dict.SetStr("disable", &object.BuiltinFunc{Name: "disable", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		level := logCRITICAL
		if len(a) >= 1 {
			if n, ok := toInt64(a[0]); ok {
				level = int(n)
			}
		}
		ls.disabledLevel = level
		return object.None, nil
	}})

	// --- basicConfig ---
	m.Dict.SetStr("basicConfig", &object.BuiltinFunc{Name: "basicConfig", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		root := ls.loggers["root"]
		// Check if root already has handlers (idempotent unless force=True).
		force := false
		if kw != nil {
			if v, ok := kw.GetStr("force"); ok {
				force = object.Truthy(v)
			}
		}
		handlersVal, hasHandlers := root.Dict.GetStr("handlers")
		if hasHandlers {
			if lst, ok := handlersVal.(*object.List); ok && len(lst.V) > 0 && !force {
				return object.None, nil
			}
		}
		// Clear existing handlers if force.
		if force {
			root.Dict.SetStr("handlers", &object.List{V: nil})
		}

		level := logWARNING
		fmtStr := "%(levelname)s:%(name)s:%(message)s"
		datefmt := ""
		var stream object.Object = object.None
		filename := ""
		filemode := "a"

		if kw != nil {
			if v, ok := kw.GetStr("level"); ok {
				if n, ok2 := toInt64(v); ok2 {
					level = int(n)
				}
			}
			if v, ok := kw.GetStr("format"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					fmtStr = s.V
				}
			}
			if v, ok := kw.GetStr("datefmt"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					datefmt = s.V
				}
			}
			if v, ok := kw.GetStr("stream"); ok {
				stream = v
			}
			if v, ok := kw.GetStr("filename"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					filename = s.V
				}
			}
			if v, ok := kw.GetStr("filemode"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					filemode = s.V
				}
			}
		}

		// Set root level.
		root.Dict.SetStr("level", object.NewInt(int64(level)))

		// Build formatter.
		fmtInst := newFormatterInst(formatterCls, fmtStr, datefmt)

		// Build handler.
		var hdlr *object.Instance
		if filename != "" {
			flag := os.O_APPEND | os.O_CREATE | os.O_WRONLY
			if filemode == "w" {
				flag = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
			}
			f, err := os.OpenFile(filename, flag, 0644)
			if err != nil {
				return nil, object.Errorf(i.osErr, "basicConfig: %v", err)
			}
			hdlr = newFileHandlerInst(handlerCls, fileHandlerCls, f)
		} else {
			var w interface{ Write([]byte) (int, error) }
			if stream != object.None {
				switch v := stream.(type) {
				case *object.StringIO:
					w = &stringIOWriter{v}
				case *object.BytesIO:
					w = &bytesIOWriter{v}
				case *object.File:
					w = v.F.(*os.File)
				case *object.TextStream:
					if wr, ok := v.W.(interface{ Write([]byte) (int, error) }); ok {
						w = wr
					}
				}
			}
			if w == nil {
				w = os.Stderr
			}
			hdlr = newStreamHandlerInst(handlerCls, streamHandlerCls, w)
		}
		hdlr.Dict.SetStr("formatter", fmtInst)
		hdlr.Dict.SetStr("level", object.NewInt(logNOTSET))

		existing, _ := root.Dict.GetStr("handlers")
		if lst, ok := existing.(*object.List); ok {
			lst.V = append(lst.V, hdlr)
		} else {
			root.Dict.SetStr("handlers", &object.List{V: []object.Object{hdlr}})
		}
		return object.None, nil
	}})

	// --- makeLogRecord ---
	m.Dict.SetStr("makeLogRecord", &object.BuiltinFunc{Name: "makeLogRecord", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		rec := newEmptyLogRecord(logRecordCls)
		if len(a) >= 1 {
			if d, ok := a[0].(*object.Dict); ok {
				keys, vals := d.Items()
				for k, key := range keys {
					if ks, ok2 := key.(*object.Str); ok2 {
						rec.Dict.SetStr(ks.V, vals[k])
					}
				}
			}
		}
		return rec, nil
	}})

	// --- captureWarnings ---
	m.Dict.SetStr("captureWarnings", &object.BuiltinFunc{Name: "captureWarnings", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// --- module-level convenience functions (delegate to root logger) ---
	for _, spec := range []struct {
		name  string
		level int
	}{
		{"debug", logDEBUG},
		{"info", logINFO},
		{"warning", logWARNING},
		{"warn", logWARNING},
		{"error", logERROR},
		{"critical", logCRITICAL},
		{"fatal", logCRITICAL},
	} {
		spec := spec
		m.Dict.SetStr(spec.name, &object.BuiltinFunc{Name: spec.name, Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			root := ls.loggers["root"]
			return i.loggerLog(ls, root, spec.level, a, kw)
		}})
	}
	m.Dict.SetStr("exception", &object.BuiltinFunc{Name: "exception", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		root := ls.loggers["root"]
		return i.loggerLog(ls, root, logERROR, a, kw)
	}})
	m.Dict.SetStr("log", &object.BuiltinFunc{Name: "log", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "log() requires at least 2 arguments")
		}
		level, ok := toInt64(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "log() level must be int")
		}
		root := ls.loggers["root"]
		return i.loggerLog(ls, root, int(level), a[1:], kw)
	}})

	return m
}

// --- helpers ---

// stringIOWriter / bytesIOWriter wrap in-memory streams as io.Writer.
type stringIOWriter struct{ sio *object.StringIO }

func (w *stringIOWriter) Write(b []byte) (int, error) {
	if w.sio.Closed {
		return 0, fmt.Errorf("I/O on closed file")
	}
	w.sio.V = append(w.sio.V, b...)
	w.sio.Pos = len(w.sio.V)
	return len(b), nil
}

type bytesIOWriter struct{ bio *object.BytesIO }

func (w *bytesIOWriter) Write(b []byte) (int, error) {
	if w.bio.Closed {
		return 0, fmt.Errorf("I/O on closed file")
	}
	w.bio.V = append(w.bio.V, b...)
	w.bio.Pos = len(w.bio.V)
	return len(b), nil
}

// getOrCreateLogger returns the Logger for `name`, creating the hierarchy if needed.
func (i *Interp) getOrCreateLogger(ls *logState, cls *object.Class, name string) *object.Instance {
	key := name
	if name == "" {
		key = "root"
	}
	if lg, ok := ls.loggers[key]; ok {
		return lg
	}
	lg := newLoggerInst(ls, cls, name, logNOTSET)
	// Set parent: find closest ancestor.
	parent := ls.loggers["root"]
	parts := strings.Split(name, ".")
	for k := len(parts) - 1; k > 0; k-- {
		ancestor := strings.Join(parts[:k], ".")
		if anc, ok := ls.loggers[ancestor]; ok {
			parent = anc
			break
		}
	}
	lg.Dict.SetStr("parent", parent)
	ls.loggers[key] = lg
	return lg
}

// newLoggerInst creates a Logger instance.
func newLoggerInst(ls *logState, cls *object.Class, name string, level int) *object.Instance {
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("name", &object.Str{V: name})
	inst.Dict.SetStr("level", object.NewInt(int64(level)))
	inst.Dict.SetStr("propagate", object.True)
	inst.Dict.SetStr("handlers", &object.List{V: nil})
	inst.Dict.SetStr("disabled", object.False)
	inst.Dict.SetStr("parent", object.None)
	inst.Dict.SetStr("filters", &object.List{V: nil})
	return inst
}

// loggerGetEffectiveLevel walks the parent chain to find the first set level.
func loggerGetEffectiveLevel(lg *object.Instance) int {
	cur := lg
	for cur != nil {
		if v, ok := cur.Dict.GetStr("level"); ok {
			if n, ok2 := toInt64(v); ok2 && n != logNOTSET {
				return int(n)
			}
		}
		if p, ok := cur.Dict.GetStr("parent"); ok {
			if pi, ok2 := p.(*object.Instance); ok2 {
				cur = pi
			} else {
				break
			}
		} else {
			break
		}
	}
	return logWARNING
}

// loggerLog emits a log message at the given level.
func (i *Interp) loggerLog(ls *logState, lg *object.Instance, level int, a []object.Object, kw *object.Dict) (object.Object, error) {
	if len(a) < 1 {
		return nil, object.Errorf(i.typeErr, "log message required")
	}
	// Check global disabled.
	if level <= ls.disabledLevel && ls.disabledLevel > 0 {
		return object.None, nil
	}
	// Check level.
	if level < loggerGetEffectiveLevel(lg) {
		return object.None, nil
	}
	// Check disabled flag.
	if v, ok := lg.Dict.GetStr("disabled"); ok && object.Truthy(v) {
		return object.None, nil
	}

	msg := ""
	if s, ok := a[0].(*object.Str); ok {
		msg = s.V
	} else {
		msg = object.Repr(a[0])
	}
	// Apply % formatting if extra args provided.
	if len(a) > 1 {
		args := a[1:]
		msg = logFormatMsg(msg, args)
	}

	name := "root"
	if v, ok := lg.Dict.GetStr("name"); ok {
		if s, ok2 := v.(*object.Str); ok2 {
			name = s.V
		}
	}

	// Build record.
	rec := newSimpleLogRecord(ls, level, name, msg)

	// Call handlers.
	i.callHandlers(ls, lg, rec)
	return object.None, nil
}

// callHandlers walks the parent chain calling each handler.
func (i *Interp) callHandlers(ls *logState, lg *object.Instance, rec *object.Instance) {
	cur := lg
	for cur != nil {
		// Call handlers on this logger.
		if v, ok := cur.Dict.GetStr("handlers"); ok {
			if lst, ok2 := v.(*object.List); ok2 {
				for _, h := range lst.V {
					if hi, ok3 := h.(*object.Instance); ok3 {
						i.handlerHandle(ls, hi, rec)
					}
				}
			}
		}
		// Check propagate.
		if v, ok := cur.Dict.GetStr("propagate"); ok && !object.Truthy(v) {
			break
		}
		// Walk to parent.
		if p, ok := cur.Dict.GetStr("parent"); ok {
			if pi, ok2 := p.(*object.Instance); ok2 {
				cur = pi
			} else {
				break
			}
		} else {
			break
		}
	}
}

// handlerHandle calls a handler's emit if the record level passes and filters allow it.
func (i *Interp) handlerHandle(ls *logState, hdlr *object.Instance, rec *object.Instance) {
	// Check handler level.
	if v, ok := hdlr.Dict.GetStr("level"); ok {
		if n, ok2 := toInt64(v); ok2 {
			recLevel := 0
			if rv, ok3 := rec.Dict.GetStr("levelno"); ok3 {
				if rn, ok4 := toInt64(rv); ok4 {
					recLevel = int(rn)
				}
			}
			if recLevel < int(n) {
				return
			}
		}
	}
	// Run handler's filter chain.
	if !i.runFilters(hdlr, rec) {
		return
	}
	// Instance-level emit (buffering/queue handlers set this in __init__).
	if emitV, ok := hdlr.Dict.GetStr("emit"); ok {
		if fn, ok2 := emitV.(*object.BuiltinFunc); ok2 {
			fn.Call(nil, []object.Object{hdlr, rec}, nil) //nolint
			return
		}
	}
	// Format message and call _emit (StreamHandler, FileHandler, etc.).
	msg := formatRecord(ls, hdlr, rec)
	if emit, ok := hdlr.Dict.GetStr("_emit"); ok {
		if fn, ok2 := emit.(*object.BuiltinFunc); ok2 {
			fn.Call(nil, []object.Object{&object.Str{V: msg}}, nil) //nolint
		}
	}
}

// runFilters checks all filters on a Filterer instance. Returns true if the record passes.
func (i *Interp) runFilters(filterer *object.Instance, rec *object.Instance) bool {
	v, ok := filterer.Dict.GetStr("filters")
	if !ok {
		return true
	}
	lst, ok := v.(*object.List)
	if !ok {
		return true
	}
	for _, f := range lst.V {
		fi, ok := f.(*object.Instance)
		if !ok {
			continue
		}
		fm, ok := fi.Class.Dict.GetStr("filter")
		if !ok {
			continue
		}
		fn, ok := fm.(*object.BuiltinFunc)
		if !ok {
			continue
		}
		result, err := fn.Call(nil, []object.Object{fi, rec}, nil)
		if err != nil {
			return false
		}
		if !object.Truthy(result) {
			return false
		}
	}
	return true
}

// formatRecord uses the handler's formatter (if any) to format the record.
func formatRecord(ls *logState, hdlr *object.Instance, rec *object.Instance) string {
	fmtStr := "%(levelname)s:%(name)s:%(message)s"
	datefmt := ""
	if v, ok := hdlr.Dict.GetStr("formatter"); ok {
		if fi, ok2 := v.(*object.Instance); ok2 {
			if fv, ok3 := fi.Dict.GetStr("_fmt"); ok3 {
				if fs, ok4 := fv.(*object.Str); ok4 {
					fmtStr = fs.V
				}
			}
			if dv, ok3 := fi.Dict.GetStr("_datefmt"); ok3 {
				if ds, ok4 := dv.(*object.Str); ok4 {
					datefmt = ds.V
				}
			}
		}
	}
	return applyFmt(fmtStr, datefmt, rec)
}

// applyFmt substitutes %(key)s/%(key)d/%(key)f in fmtStr from record fields.
var fmtDirectiveRe = regexp.MustCompile(`%\((\w+)\)[#0\- +]?(\*|\d+)?\.?(\*|\d+)?[diouxXeEfFgGcrsab%]`)

func applyFmt(fmtStr, datefmt string, rec *object.Instance) string {
	result := fmtDirectiveRe.ReplaceAllStringFunc(fmtStr, func(match string) string {
		// Extract key from %(key)s form.
		inner := match[2 : strings.Index(match, ")")]
		conv := match[len(match)-1]
		val, ok := rec.Dict.GetStr(inner)
		if !ok {
			if inner == "asctime" {
				// format current time
				if datefmt != "" {
					return strftime(datefmt, time.Now())
				}
				return time.Now().Format("2006-01-02 15:04:05,000")
			}
			return match
		}
		switch conv {
		case 'd', 'i':
			if n, ok2 := toInt64(val); ok2 {
				return fmt.Sprintf("%d", n)
			}
		case 'f':
			if f, ok2 := toFloat64(val); ok2 {
				return fmt.Sprintf("%f", f)
			}
		}
		// default: string
		return anyToStr(val)
	})
	return result
}

// logFormatMsg applies % formatting on the message string.
func logFormatMsg(msg string, args []object.Object) string {
	if len(args) == 0 {
		return msg
	}
	// Simple substitution: replace each %s/%d with the next arg.
	result := &strings.Builder{}
	argIdx := 0
	for j := 0; j < len(msg); j++ {
		if msg[j] != '%' || j+1 >= len(msg) || argIdx >= len(args) {
			result.WriteByte(msg[j])
			continue
		}
		j++
		switch msg[j] {
		case 's':
			result.WriteString(anyToStr(args[argIdx]))
			argIdx++
		case 'd':
			if n, ok := toInt64(args[argIdx]); ok {
				result.WriteString(fmt.Sprintf("%d", n))
			} else {
				result.WriteString(anyToStr(args[argIdx]))
			}
			argIdx++
		case 'f':
			if f, ok := toFloat64(args[argIdx]); ok {
				result.WriteString(fmt.Sprintf("%f", f))
			} else {
				result.WriteString(anyToStr(args[argIdx]))
			}
			argIdx++
		case 'r':
			result.WriteString(object.Repr(args[argIdx]))
			argIdx++
		case '%':
			result.WriteByte('%')
		default:
			result.WriteByte('%')
			result.WriteByte(msg[j])
		}
	}
	return result.String()
}

// newSimpleLogRecord creates a LogRecord with the most important fields set.
func newSimpleLogRecord(ls *logState, level int, name, msg string) *object.Instance {
	cls := &object.Class{Name: "LogRecord", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	now := time.Now()
	created := float64(now.UnixNano()) / 1e9
	inst.Dict.SetStr("name", &object.Str{V: name})
	inst.Dict.SetStr("msg", &object.Str{V: msg})
	inst.Dict.SetStr("args", &object.Tuple{V: nil})
	inst.Dict.SetStr("levelno", object.NewInt(int64(level)))
	inst.Dict.SetStr("levelname", &object.Str{V: ls.getLevelName(level)})
	inst.Dict.SetStr("pathname", &object.Str{V: ""})
	inst.Dict.SetStr("filename", &object.Str{V: ""})
	inst.Dict.SetStr("module", &object.Str{V: ""})
	inst.Dict.SetStr("funcName", &object.Str{V: ""})
	inst.Dict.SetStr("lineno", object.NewInt(0))
	inst.Dict.SetStr("created", &object.Float{V: created})
	inst.Dict.SetStr("msecs", &object.Float{V: float64(now.UnixNano()/1e6) - float64(now.Unix())*1e3})
	inst.Dict.SetStr("relativeCreated", &object.Float{V: (created - ls.startTime) * 1e3})
	inst.Dict.SetStr("thread", object.NewInt(0))
	inst.Dict.SetStr("threadName", &object.Str{V: "MainThread"})
	inst.Dict.SetStr("process", object.NewInt(int64(os.Getpid())))
	inst.Dict.SetStr("processName", &object.Str{V: "MainProcess"})
	inst.Dict.SetStr("exc_info", object.None)
	inst.Dict.SetStr("exc_text", object.None)
	inst.Dict.SetStr("stack_info", object.None)
	inst.Dict.SetStr("message", &object.Str{V: msg})
	return inst
}

func newEmptyLogRecord(cls *object.Class) *object.Instance {
	return &object.Instance{Class: cls, Dict: object.NewDict()}
}

// --- class builders ---

func (i *Interp) buildFilterClass(ls *logState) *object.Class {
	cls := &object.Class{Name: "Filter", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		name := ""
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				name = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("name"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					name = s.V
				}
			}
		}
		self.Dict.SetStr("name", &object.Str{V: name})
		self.Dict.SetStr("nlen", object.NewInt(int64(len(name))))
		return object.None, nil
	}})
	cls.Dict.SetStr("filter", &object.BuiltinFunc{Name: "filter", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.True, nil
		}
		self := a[0].(*object.Instance)
		rec := a[1].(*object.Instance)
		nv, _ := self.Dict.GetStr("name")
		nlen, _ := self.Dict.GetStr("nlen")
		name := ""
		if s, ok := nv.(*object.Str); ok {
			name = s.V
		}
		n, _ := toInt64(nlen)
		if n == 0 {
			return object.True, nil
		}
		recName := ""
		if rv, ok := rec.Dict.GetStr("name"); ok {
			if rs, ok2 := rv.(*object.Str); ok2 {
				recName = rs.V
			}
		}
		if recName == name || strings.HasPrefix(recName, name+".") {
			return object.True, nil
		}
		return object.False, nil
	}})
	return cls
}

func (i *Interp) buildFiltererClass(ls *logState) *object.Class {
	cls := &object.Class{Name: "Filterer", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		self.Dict.SetStr("filters", &object.List{V: nil})
		return object.None, nil
	}})
	cls.Dict.SetStr("addFilter", &object.BuiltinFunc{Name: "addFilter", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		f := a[1]
		if v, ok := self.Dict.GetStr("filters"); ok {
			if lst, ok2 := v.(*object.List); ok2 {
				lst.V = append(lst.V, f)
			}
		}
		return object.None, nil
	}})
	cls.Dict.SetStr("removeFilter", &object.BuiltinFunc{Name: "removeFilter", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		target := a[1]
		if v, ok := self.Dict.GetStr("filters"); ok {
			if lst, ok2 := v.(*object.List); ok2 {
				out := lst.V[:0]
				for _, x := range lst.V {
					if x != target {
						out = append(out, x)
					}
				}
				lst.V = out
			}
		}
		return object.None, nil
	}})
	cls.Dict.SetStr("filter", &object.BuiltinFunc{Name: "filter", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.True, nil
		}
		self := a[0].(*object.Instance)
		rec := a[1]
		if v, ok := self.Dict.GetStr("filters"); ok {
			if lst, ok2 := v.(*object.List); ok2 {
				for _, f := range lst.V {
					if fi, ok3 := f.(*object.Instance); ok3 {
						if fm, ok4 := fi.Class.Dict.GetStr("filter"); ok4 {
							if fn, ok5 := fm.(*object.BuiltinFunc); ok5 {
								result, _ := fn.Call(nil, []object.Object{fi, rec}, nil)
								if !object.Truthy(result) {
									return object.False, nil
								}
							}
						}
					}
				}
			}
		}
		return object.True, nil
	}})
	return cls
}

func (i *Interp) buildHandlerClass(ls *logState, filtererCls *object.Class) *object.Class {
	cls := &object.Class{Name: "Handler", Bases: []*object.Class{filtererCls}, Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		level := logNOTSET
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok {
				level = int(n)
			}
		}
		self.Dict.SetStr("level", object.NewInt(int64(level)))
		self.Dict.SetStr("formatter", object.None)
		self.Dict.SetStr("filters", &object.List{V: nil})
		return object.None, nil
	}})
	cls.Dict.SetStr("setLevel", &object.BuiltinFunc{Name: "setLevel", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if n, ok := toInt64(a[1]); ok {
			self.Dict.SetStr("level", object.NewInt(n))
		} else if s, ok2 := a[1].(*object.Str); ok2 {
			if n2, ok3 := ls.getLevelNum(s.V); ok3 {
				self.Dict.SetStr("level", object.NewInt(int64(n2)))
			}
		}
		return object.None, nil
	}})
	cls.Dict.SetStr("setFormatter", &object.BuiltinFunc{Name: "setFormatter", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		a[0].(*object.Instance).Dict.SetStr("formatter", a[1])
		return object.None, nil
	}})
	cls.Dict.SetStr("format", &object.BuiltinFunc{Name: "format", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return &object.Str{V: ""}, nil
		}
		self := a[0].(*object.Instance)
		rec := a[1].(*object.Instance)
		msg := formatRecord(ls, self, rec)
		return &object.Str{V: msg}, nil
	}})
	cls.Dict.SetStr("handle", &object.BuiltinFunc{Name: "handle", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		rec := a[1].(*object.Instance)
		i.handlerHandle(ls, self, rec)
		return object.None, nil
	}})
	cls.Dict.SetStr("emit", &object.BuiltinFunc{Name: "emit", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	cls.Dict.SetStr("flush", &object.BuiltinFunc{Name: "flush", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	cls.Dict.SetStr("addFilter", mustGetStr(filtererCls.Dict, "addFilter"))
	cls.Dict.SetStr("removeFilter", mustGetStr(filtererCls.Dict, "removeFilter"))
	cls.Dict.SetStr("filter", mustGetStr(filtererCls.Dict, "filter"))
	return cls
}

func newStreamHandlerInst(handlerCls, streamHandlerCls *object.Class, w interface{ Write([]byte) (int, error) }) *object.Instance {
	inst := &object.Instance{Class: streamHandlerCls, Dict: object.NewDict()}
	inst.Dict.SetStr("level", object.NewInt(logNOTSET))
	inst.Dict.SetStr("formatter", object.None)
	inst.Dict.SetStr("filters", &object.List{V: nil})
	inst.Dict.SetStr("_emit", &object.BuiltinFunc{Name: "_emit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		msg := ""
		if len(a) >= 1 {
			if s, ok := a[0].(*object.Str); ok {
				msg = s.V
			}
		}
		w.Write([]byte(msg + "\n")) //nolint
		return object.None, nil
	}})
	return inst
}

func (i *Interp) buildStreamHandlerClass(ls *logState, handlerCls *object.Class) *object.Class {
	cls := &object.Class{Name: "StreamHandler", Bases: []*object.Class{handlerCls}, Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		self.Dict.SetStr("level", object.NewInt(logNOTSET))
		self.Dict.SetStr("formatter", object.None)
		self.Dict.SetStr("filters", &object.List{V: nil})

		var w interface{ Write([]byte) (int, error) } = os.Stderr
		var stream object.Object = object.None
		if len(a) >= 2 && a[1] != object.None {
			stream = a[1]
		}
		if kw != nil {
			if v, ok := kw.GetStr("stream"); ok && v != object.None {
				stream = v
			}
		}
		if stream != object.None {
			switch v := stream.(type) {
			case *object.StringIO:
				w = &stringIOWriter{v}
			case *object.BytesIO:
				w = &bytesIOWriter{v}
			case *object.File:
				w = v.F.(*os.File)
			case *object.TextStream:
				if wr, ok := v.W.(interface{ Write([]byte) (int, error) }); ok {
					w = wr
				}
			}
		}
		wCapture := w
		self.Dict.SetStr("_emit", &object.BuiltinFunc{Name: "_emit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			msg := ""
			if len(a) >= 1 {
				if s, ok := a[0].(*object.Str); ok {
					msg = s.V
				}
			}
			wCapture.Write([]byte(msg + "\n")) //nolint
			return object.None, nil
		}})
		return object.None, nil
	}})
	cls.Dict.SetStr("setLevel", mustGetStr(handlerCls.Dict, "setLevel"))
	cls.Dict.SetStr("setFormatter", mustGetStr(handlerCls.Dict, "setFormatter"))
	cls.Dict.SetStr("format", mustGetStr(handlerCls.Dict, "format"))
	cls.Dict.SetStr("handle", mustGetStr(handlerCls.Dict, "handle"))
	cls.Dict.SetStr("flush", mustGetStr(handlerCls.Dict, "flush"))
	cls.Dict.SetStr("close", mustGetStr(handlerCls.Dict, "close"))
	cls.Dict.SetStr("addFilter", mustGetStr(handlerCls.Dict, "addFilter"))
	cls.Dict.SetStr("removeFilter", mustGetStr(handlerCls.Dict, "removeFilter"))
	cls.Dict.SetStr("filter", mustGetStr(handlerCls.Dict, "filter"))
	return cls
}

func newFileHandlerInst(handlerCls, fileHandlerCls *object.Class, f *os.File) *object.Instance {
	inst := &object.Instance{Class: fileHandlerCls, Dict: object.NewDict()}
	inst.Dict.SetStr("level", object.NewInt(logNOTSET))
	inst.Dict.SetStr("formatter", object.None)
	inst.Dict.SetStr("filters", &object.List{V: nil})
	inst.Dict.SetStr("_emit", &object.BuiltinFunc{Name: "_emit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		msg := ""
		if len(a) >= 1 {
			if s, ok := a[0].(*object.Str); ok {
				msg = s.V
			}
		}
		f.WriteString(msg + "\n") //nolint
		return object.None, nil
	}})
	return inst
}

func (i *Interp) buildFileHandlerClass(ls *logState, handlerCls *object.Class) *object.Class {
	cls := &object.Class{Name: "FileHandler", Bases: []*object.Class{handlerCls}, Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, fmt.Errorf("FileHandler requires filename")
		}
		self := a[0].(*object.Instance)
		filename, ok := a[1].(*object.Str)
		if !ok {
			return nil, fmt.Errorf("FileHandler: filename must be str")
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
		flag := os.O_APPEND | os.O_CREATE | os.O_WRONLY
		if mode == "w" {
			flag = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
		}
		f, err := os.OpenFile(filename.V, flag, 0644)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		self.Dict.SetStr("level", object.NewInt(logNOTSET))
		self.Dict.SetStr("formatter", object.None)
		self.Dict.SetStr("filters", &object.List{V: nil})
		self.Dict.SetStr("baseFilename", filename)
		fCapture := f
		self.Dict.SetStr("_emit", &object.BuiltinFunc{Name: "_emit", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			msg := ""
			if len(a) >= 1 {
				if s, ok2 := a[0].(*object.Str); ok2 {
					msg = s.V
				}
			}
			fCapture.WriteString(msg + "\n") //nolint
			return object.None, nil
		}})
		self.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			fCapture.Close()
			return object.None, nil
		}})
		return object.None, nil
	}})
	cls.Dict.SetStr("setLevel", mustGetStr(handlerCls.Dict, "setLevel"))
	cls.Dict.SetStr("setFormatter", mustGetStr(handlerCls.Dict, "setFormatter"))
	cls.Dict.SetStr("format", mustGetStr(handlerCls.Dict, "format"))
	cls.Dict.SetStr("handle", mustGetStr(handlerCls.Dict, "handle"))
	cls.Dict.SetStr("flush", mustGetStr(handlerCls.Dict, "flush"))
	cls.Dict.SetStr("addFilter", mustGetStr(handlerCls.Dict, "addFilter"))
	cls.Dict.SetStr("removeFilter", mustGetStr(handlerCls.Dict, "removeFilter"))
	cls.Dict.SetStr("filter", mustGetStr(handlerCls.Dict, "filter"))
	return cls
}

func (i *Interp) buildNullHandlerClass(handlerCls *object.Class) *object.Class {
	cls := &object.Class{Name: "NullHandler", Bases: []*object.Class{handlerCls}, Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 1 {
			self := a[0].(*object.Instance)
			self.Dict.SetStr("level", object.NewInt(logNOTSET))
			self.Dict.SetStr("formatter", object.None)
			self.Dict.SetStr("filters", &object.List{V: nil})
			// _emit does nothing.
			self.Dict.SetStr("_emit", &object.BuiltinFunc{Name: "_emit", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}})
		}
		return object.None, nil
	}})
	cls.Dict.SetStr("emit", &object.BuiltinFunc{Name: "emit", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	cls.Dict.SetStr("handle", &object.BuiltinFunc{Name: "handle", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	cls.Dict.SetStr("setLevel", mustGetStr(handlerCls.Dict, "setLevel"))
	cls.Dict.SetStr("setFormatter", mustGetStr(handlerCls.Dict, "setFormatter"))
	cls.Dict.SetStr("flush", mustGetStr(handlerCls.Dict, "flush"))
	cls.Dict.SetStr("close", mustGetStr(handlerCls.Dict, "close"))
	cls.Dict.SetStr("addFilter", mustGetStr(handlerCls.Dict, "addFilter"))
	cls.Dict.SetStr("removeFilter", mustGetStr(handlerCls.Dict, "removeFilter"))
	cls.Dict.SetStr("filter", mustGetStr(handlerCls.Dict, "filter"))
	return cls
}

func newFormatterInst(cls *object.Class, fmtStr, datefmt string) *object.Instance {
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("_fmt", &object.Str{V: fmtStr})
	inst.Dict.SetStr("_datefmt", &object.Str{V: datefmt})
	return inst
}

func (i *Interp) buildLoggingFormatterClass(ls *logState) *object.Class {
	cls := &object.Class{Name: "Formatter", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		fmtStr := "%(levelname)s:%(name)s:%(message)s"
		datefmt := ""
		if len(a) >= 2 && a[1] != object.None {
			if s, ok := a[1].(*object.Str); ok {
				fmtStr = s.V
			}
		}
		if len(a) >= 3 && a[2] != object.None {
			if s, ok := a[2].(*object.Str); ok {
				datefmt = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("fmt"); ok && v != object.None {
				if s, ok2 := v.(*object.Str); ok2 {
					fmtStr = s.V
				}
			}
			if v, ok := kw.GetStr("datefmt"); ok && v != object.None {
				if s, ok2 := v.(*object.Str); ok2 {
					datefmt = s.V
				}
			}
		}
		self.Dict.SetStr("_fmt", &object.Str{V: fmtStr})
		self.Dict.SetStr("_datefmt", &object.Str{V: datefmt})
		return object.None, nil
	}})
	cls.Dict.SetStr("format", &object.BuiltinFunc{Name: "format", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return &object.Str{V: ""}, nil
		}
		self := a[0].(*object.Instance)
		rec, ok := a[1].(*object.Instance)
		if !ok {
			return &object.Str{V: ""}, nil
		}
		fmtStr := "%(levelname)s:%(name)s:%(message)s"
		datefmt := ""
		if v, ok2 := self.Dict.GetStr("_fmt"); ok2 {
			if s, ok3 := v.(*object.Str); ok3 {
				fmtStr = s.V
			}
		}
		if v, ok2 := self.Dict.GetStr("_datefmt"); ok2 {
			if s, ok3 := v.(*object.Str); ok3 {
				datefmt = s.V
			}
		}
		return &object.Str{V: applyFmt(fmtStr, datefmt, rec)}, nil
	}})
	cls.Dict.SetStr("formatTime", &object.BuiltinFunc{Name: "formatTime", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		datefmt := ""
		if len(a) >= 3 && a[2] != object.None {
			if s, ok := a[2].(*object.Str); ok {
				datefmt = s.V
			}
		}
		now := time.Now()
		if datefmt != "" {
			return &object.Str{V: strftime(datefmt, now)}, nil
		}
		return &object.Str{V: now.Format("2006-01-02 15:04:05,000")}, nil
	}})
	cls.Dict.SetStr("formatException", &object.BuiltinFunc{Name: "formatException", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: ""}, nil
	}})
	cls.Dict.SetStr("formatStack", &object.BuiltinFunc{Name: "formatStack", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: ""}, nil
	}})
	return cls
}

func (i *Interp) buildLogRecordClass(ls *logState) *object.Class {
	cls := &object.Class{Name: "LogRecord", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return makeLogRecordInst(ls, cls, a, kw)
	}})
	cls.Dict.SetStr("getMessage", &object.BuiltinFunc{Name: "getMessage", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		self := a[0].(*object.Instance)
		if v, ok := self.Dict.GetStr("message"); ok {
			return v, nil
		}
		if v, ok := self.Dict.GetStr("msg"); ok {
			return v, nil
		}
		return &object.Str{V: ""}, nil
	}})
	return cls
}

func makeLogRecordInst(ls *logState, cls *object.Class, a []object.Object, kw *object.Dict) (object.Object, error) {
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	get := func(idx int, kwName string, def object.Object) object.Object {
		if len(a) > idx && a[idx] != object.None {
			return a[idx]
		}
		if kw != nil {
			if v, ok := kw.GetStr(kwName); ok {
				return v
			}
		}
		return def
	}
	name := get(1, "name", &object.Str{V: "root"})
	level := get(2, "level", object.NewInt(0))
	pathname := get(3, "pathname", &object.Str{V: ""})
	lineno := get(4, "lineno", object.NewInt(0))
	msg := get(5, "msg", &object.Str{V: ""})
	args := get(6, "args", object.None)
	excInfo := get(7, "exc_info", object.None)

	msgStr := ""
	if s, ok := msg.(*object.Str); ok {
		msgStr = s.V
	}
	lvl := 0
	if n, ok := toInt64(level); ok {
		lvl = int(n)
	}
	pth := ""
	if s, ok := pathname.(*object.Str); ok {
		pth = s.V
	}
	filename := pth
	if idx := strings.LastIndex(pth, "/"); idx >= 0 {
		filename = pth[idx+1:]
	}
	module := filename
	if idx := strings.LastIndex(filename, "."); idx >= 0 {
		module = filename[:idx]
	}

	now := time.Now()
	created := float64(now.UnixNano()) / 1e9

	inst.Dict.SetStr("name", name)
	inst.Dict.SetStr("msg", msg)
	inst.Dict.SetStr("args", args)
	inst.Dict.SetStr("levelno", object.NewInt(int64(lvl)))
	inst.Dict.SetStr("levelname", &object.Str{V: ls.getLevelName(lvl)})
	inst.Dict.SetStr("pathname", pathname)
	inst.Dict.SetStr("filename", &object.Str{V: filename})
	inst.Dict.SetStr("module", &object.Str{V: module})
	inst.Dict.SetStr("funcName", &object.Str{V: ""})
	inst.Dict.SetStr("lineno", lineno)
	inst.Dict.SetStr("created", &object.Float{V: created})
	inst.Dict.SetStr("msecs", &object.Float{V: float64(now.Nanosecond() / 1e6)})
	inst.Dict.SetStr("relativeCreated", &object.Float{V: (created - ls.startTime) * 1e3})
	inst.Dict.SetStr("thread", object.NewInt(0))
	inst.Dict.SetStr("threadName", &object.Str{V: "MainThread"})
	inst.Dict.SetStr("process", object.NewInt(int64(os.Getpid())))
	inst.Dict.SetStr("processName", &object.Str{V: "MainProcess"})
	inst.Dict.SetStr("exc_info", excInfo)
	inst.Dict.SetStr("exc_text", object.None)
	inst.Dict.SetStr("stack_info", object.None)
	inst.Dict.SetStr("message", &object.Str{V: msgStr})
	_ = args
	return object.None, nil
}

func (i *Interp) buildLoggerClass(ls *logState, filtererCls *object.Class, logRecordCls *object.Class) *object.Class {
	cls := &object.Class{Name: "Logger", Bases: []*object.Class{filtererCls}, Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		name, _ := a[1].(*object.Str)
		level := logNOTSET
		if len(a) >= 3 {
			if n, ok := toInt64(a[2]); ok {
				level = int(n)
			}
		}
		nameStr := ""
		if name != nil {
			nameStr = name.V
		}
		self.Dict.SetStr("name", &object.Str{V: nameStr})
		self.Dict.SetStr("level", object.NewInt(int64(level)))
		self.Dict.SetStr("propagate", object.True)
		self.Dict.SetStr("handlers", &object.List{V: nil})
		self.Dict.SetStr("disabled", object.False)
		self.Dict.SetStr("parent", object.None)
		self.Dict.SetStr("filters", &object.List{V: nil})
		return object.None, nil
	}})

	cls.Dict.SetStr("setLevel", &object.BuiltinFunc{Name: "setLevel", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if n, ok := toInt64(a[1]); ok {
			self.Dict.SetStr("level", object.NewInt(n))
		} else if s, ok2 := a[1].(*object.Str); ok2 {
			if n2, ok3 := ls.getLevelNum(s.V); ok3 {
				self.Dict.SetStr("level", object.NewInt(int64(n2)))
			}
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("getEffectiveLevel", &object.BuiltinFunc{Name: "getEffectiveLevel", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.NewInt(logWARNING), nil
		}
		self := a[0].(*object.Instance)
		return object.NewInt(int64(loggerGetEffectiveLevel(self))), nil
	}})

	cls.Dict.SetStr("isEnabledFor", &object.BuiltinFunc{Name: "isEnabledFor", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		self := a[0].(*object.Instance)
		level, _ := toInt64(a[1])
		if int(level) <= ls.disabledLevel && ls.disabledLevel > 0 {
			return object.False, nil
		}
		return object.BoolOf(int(level) >= loggerGetEffectiveLevel(self)), nil
	}})

	cls.Dict.SetStr("addHandler", &object.BuiltinFunc{Name: "addHandler", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		hdlr := a[1]
		if v, ok := self.Dict.GetStr("handlers"); ok {
			if lst, ok2 := v.(*object.List); ok2 {
				lst.V = append(lst.V, hdlr)
			}
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("removeHandler", &object.BuiltinFunc{Name: "removeHandler", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		target := a[1]
		if v, ok := self.Dict.GetStr("handlers"); ok {
			if lst, ok2 := v.(*object.List); ok2 {
				out := lst.V[:0]
				for _, x := range lst.V {
					if x != target {
						out = append(out, x)
					}
				}
				lst.V = out
			}
		}
		return object.None, nil
	}})

	cls.Dict.SetStr("hasHandlers", &object.BuiltinFunc{Name: "hasHandlers", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.False, nil
		}
		cur := a[0].(*object.Instance)
		for cur != nil {
			if v, ok := cur.Dict.GetStr("handlers"); ok {
				if lst, ok2 := v.(*object.List); ok2 && len(lst.V) > 0 {
					return object.True, nil
				}
			}
			// Stop if propagate is False.
			if pv, ok := cur.Dict.GetStr("propagate"); ok && !object.Truthy(pv) {
				break
			}
			if p, ok := cur.Dict.GetStr("parent"); ok {
				if pi, ok2 := p.(*object.Instance); ok2 {
					cur = pi
				} else {
					break
				}
			} else {
				break
			}
		}
		return object.False, nil
	}})

	cls.Dict.SetStr("getChild", &object.BuiltinFunc{Name: "getChild", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		suffix, ok := a[1].(*object.Str)
		if !ok {
			return object.None, nil
		}
		name := ""
		if v, ok2 := self.Dict.GetStr("name"); ok2 {
			if s, ok3 := v.(*object.Str); ok3 {
				name = s.V
			}
		}
		childName := suffix.V
		if name != "" && name != "root" {
			childName = name + "." + suffix.V
		}
		return i.getOrCreateLogger(ls, cls, childName), nil
	}})

	logFn := func(level int) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: "log", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			return i.loggerLog(ls, self, level, a[1:], kw)
		}}
	}

	cls.Dict.SetStr("debug", logFn(logDEBUG))
	cls.Dict.SetStr("info", logFn(logINFO))
	cls.Dict.SetStr("warning", logFn(logWARNING))
	cls.Dict.SetStr("warn", logFn(logWARNING))
	cls.Dict.SetStr("error", logFn(logERROR))
	cls.Dict.SetStr("critical", logFn(logCRITICAL))
	cls.Dict.SetStr("fatal", logFn(logCRITICAL))
	cls.Dict.SetStr("exception", logFn(logERROR))

	cls.Dict.SetStr("log", &object.BuiltinFunc{Name: "log", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		level, ok := toInt64(a[1])
		if !ok {
			return nil, object.Errorf(i.typeErr, "log() level must be int")
		}
		return i.loggerLog(ls, self, int(level), a[2:], kw)
	}})

	cls.Dict.SetStr("addFilter", mustGetStr(filtererCls.Dict, "addFilter"))
	cls.Dict.SetStr("removeFilter", mustGetStr(filtererCls.Dict, "removeFilter"))
	cls.Dict.SetStr("filter", mustGetStr(filtererCls.Dict, "filter"))

	return cls
}

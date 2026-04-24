package vm

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildLoggingConfig() *object.Module {
	m := &object.Module{Name: "logging.config", Dict: object.NewDict()}

	m.Dict.SetStr("DEFAULT_LOGGING_CONFIG_PORT", object.NewInt(9030))

	m.Dict.SetStr("dictConfig", &object.BuiltinFunc{Name: "dictConfig", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "dictConfig() requires 1 argument")
		}
		cfg, ok := a[0].(*object.Dict)
		if !ok {
			return nil, object.Errorf(i.typeErr, "dictConfig() argument must be dict")
		}
		return object.None, i.applyDictConfig(cfg)
	}})

	m.Dict.SetStr("fileConfig", &object.BuiltinFunc{Name: "fileConfig", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fileConfig() requires fname argument")
		}
		fname, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "fileConfig() fname must be str")
		}
		disableExisting := true
		if kw != nil {
			if v, ok2 := kw.GetStr("disable_existing_loggers"); ok2 {
				disableExisting = object.Truthy(v)
			}
		}
		return object.None, i.applyFileConfig(fname.V, disableExisting)
	}})

	// listen: stub returning a sentinel
	m.Dict.SetStr("listen", &object.BuiltinFunc{Name: "listen", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		// Return a sentinel tuple so callers can check "is not None".
		return &object.Tuple{V: []object.Object{&object.Str{V: "logging.config.listener"}}}, nil
	}})

	m.Dict.SetStr("stopListening", &object.BuiltinFunc{Name: "stopListening", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	return m
}

// ensureLogState loads the logging module if needed and returns its logState.
func (i *Interp) ensureLogState() (*logState, *object.Module, error) {
	mod, err := i.loadModule("logging")
	if err != nil {
		return nil, nil, fmt.Errorf("logging.config: cannot load logging: %w", err)
	}
	if i.logStates == nil {
		return nil, nil, fmt.Errorf("logging.config: logStates not initialised")
	}
	ls, ok := i.logStates["logging"]
	if !ok {
		return nil, nil, fmt.Errorf("logging.config: logging state not found")
	}
	return ls, mod, nil
}

// getLoggingClass retrieves a class from the logging module dict by name.
func getLoggingClass(mod *object.Module, name string) *object.Class {
	v, ok := mod.Dict.GetStr(name)
	if !ok {
		return nil
	}
	if cls, ok := v.(*object.Class); ok {
		return cls
	}
	return nil
}

// lcfgStr extracts a string from a Dict key, returns "" if missing or not string.
func lcfgStr(d *object.Dict, key string) string {
	if d == nil {
		return ""
	}
	v, ok := d.GetStr(key)
	if !ok {
		return ""
	}
	if s, ok := v.(*object.Str); ok {
		return s.V
	}
	return ""
}

// lcfgBool extracts a bool with a default.
func lcfgBool(d *object.Dict, key string, def bool) bool {
	if d == nil {
		return def
	}
	v, ok := d.GetStr(key)
	if !ok {
		return def
	}
	return object.Truthy(v)
}

// lcfgDict extracts a nested *object.Dict from d[key].
func lcfgDict(d *object.Dict, key string) *object.Dict {
	if d == nil {
		return nil
	}
	v, ok := d.GetStr(key)
	if !ok {
		return nil
	}
	if sub, ok := v.(*object.Dict); ok {
		return sub
	}
	return nil
}

// lcfgStrList extracts a []string from d[key] where the value is a *object.List of *object.Str.
func lcfgStrList(d *object.Dict, key string) []string {
	if d == nil {
		return nil
	}
	v, ok := d.GetStr(key)
	if !ok {
		return nil
	}
	lst, ok := v.(*object.List)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(lst.V))
	for _, el := range lst.V {
		if s, ok := el.(*object.Str); ok {
			out = append(out, s.V)
		}
	}
	return out
}

// resolveLevelStr converts a level string like "DEBUG" or "20" to an int.
func resolveLevelStr(ls *logState, s string) int {
	if s == "" {
		return logNOTSET
	}
	if n, ok := ls.getLevelNum(s); ok {
		return n
	}
	// Try numeric.
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err == nil {
		return n
	}
	return logNOTSET
}

// resolveStream maps "ext://sys.stdout" / "ext://sys.stderr" to a writer.
func (i *Interp) resolveStream(s string) interface{ Write([]byte) (int, error) } {
	switch s {
	case "ext://sys.stdout":
		return i.Stdout
	case "ext://sys.stderr", "":
		return i.Stderr
	default:
		return i.Stderr
	}
}

// applyDictConfig is the core of dictConfig.
func (i *Interp) applyDictConfig(cfg *object.Dict) error {
	ls, loggingMod, err := i.ensureLogState()
	if err != nil {
		return err
	}
	loggerCls, ok := ls.loggerClass.(*object.Class)
	if !ok {
		return fmt.Errorf("logging.config: logger class not found")
	}

	// version check
	if vv, ok := cfg.GetStr("version"); ok {
		if n, ok2 := toInt64(vv); !ok2 || n != 1 {
			return fmt.Errorf("logging.config: unsupported config version")
		}
	}

	// incremental: only update levels
	if lcfgBool(cfg, "incremental", false) {
		return i.applyIncrementalConfig(cfg, ls, loggerCls)
	}

	disableExisting := lcfgBool(cfg, "disable_existing_loggers", true)

	// --- formatters ---
	fmtterCls := getLoggingClass(loggingMod, "Formatter")
	formatters := map[string]*object.Instance{}
	if fmtDict := lcfgDict(cfg, "formatters"); fmtDict != nil {
		ks, vs := fmtDict.Items()
		for idx, kobj := range ks {
			k, ok := kobj.(*object.Str)
			if !ok {
				continue
			}
			sub, ok := vs[idx].(*object.Dict)
			if !ok {
				continue
			}
			fmtStr := lcfgStr(sub, "format")
			if fmtStr == "" {
				fmtStr = "%(levelname)s:%(name)s:%(message)s"
			}
			datefmt := lcfgStr(sub, "datefmt")
			if fmtterCls != nil {
				formatters[k.V] = newFormatterInst(fmtterCls, fmtStr, datefmt)
			}
		}
	}

	// --- filters ---
	filterCls := getLoggingClass(loggingMod, "Filter")
	filters := map[string]*object.Instance{}
	if fltDict := lcfgDict(cfg, "filters"); fltDict != nil {
		ks, vs := fltDict.Items()
		for idx, kobj := range ks {
			k, ok := kobj.(*object.Str)
			if !ok {
				continue
			}
			sub, ok := vs[idx].(*object.Dict)
			if !ok {
				continue
			}
			name := lcfgStr(sub, "name")
			if filterCls != nil {
				inst := &object.Instance{Class: filterCls, Dict: object.NewDict()}
				inst.Dict.SetStr("name", &object.Str{V: name})
				inst.Dict.SetStr("nlen", object.NewInt(int64(len(name))))
				filters[k.V] = inst
			}
		}
	}

	// --- handlers ---
	handlerCls := getLoggingClass(loggingMod, "Handler")
	streamHCls := getLoggingClass(loggingMod, "StreamHandler")
	fileHCls := getLoggingClass(loggingMod, "FileHandler")
	nullHCls := getLoggingClass(loggingMod, "NullHandler")
	handlers := map[string]*object.Instance{}
	if hdlDict := lcfgDict(cfg, "handlers"); hdlDict != nil {
		ks, vs := hdlDict.Items()
		for idx, kobj := range ks {
			k, ok := kobj.(*object.Str)
			if !ok {
				continue
			}
			sub, ok := vs[idx].(*object.Dict)
			if !ok {
				continue
			}
			hdlr := i.buildHandlerFromDict(sub, ls, handlerCls, streamHCls, fileHCls, nullHCls)
			if hdlr == nil {
				continue
			}
			// set level
			if lvlStr := lcfgStr(sub, "level"); lvlStr != "" {
				hdlr.Dict.SetStr("level", object.NewInt(int64(resolveLevelStr(ls, lvlStr))))
			}
			// set formatter
			if fmtID := lcfgStr(sub, "formatter"); fmtID != "" {
				if fi, ok := formatters[fmtID]; ok {
					hdlr.Dict.SetStr("formatter", fi)
				}
			}
			// add filters
			for _, fid := range lcfgStrList(sub, "filters") {
				if fi, ok := filters[fid]; ok {
					if lst, ok2 := hdlr.Dict.GetStr("filters"); ok2 {
						if l, ok3 := lst.(*object.List); ok3 {
							l.V = append(l.V, fi)
						}
					}
				}
			}
			handlers[k.V] = hdlr
		}
	}

	// collect logger names in new config
	configuredLoggers := map[string]bool{"root": true, "": true}

	// --- root logger ---
	if rootDict := lcfgDict(cfg, "root"); rootDict != nil {
		root := i.getOrCreateLogger(ls, loggerCls, "")
		i.applyLoggerConfig(root, rootDict, ls, handlers, filters)
	}

	// --- named loggers ---
	if lgDict := lcfgDict(cfg, "loggers"); lgDict != nil {
		ks, vs := lgDict.Items()
		for idx, kobj := range ks {
			k, ok := kobj.(*object.Str)
			if !ok {
				continue
			}
			sub, ok := vs[idx].(*object.Dict)
			if !ok {
				continue
			}
			lg := i.getOrCreateLogger(ls, loggerCls, k.V)
			i.applyLoggerConfig(lg, sub, ls, handlers, filters)
			configuredLoggers[k.V] = true
		}
	}

	// --- disable_existing_loggers ---
	if disableExisting {
		for name, lg := range ls.loggers {
			if configuredLoggers[name] {
				continue
			}
			lg.Dict.SetStr("disabled", object.True)
		}
	}

	return nil
}

// applyIncrementalConfig handles incremental=True: only update levels.
func (i *Interp) applyIncrementalConfig(cfg *object.Dict, ls *logState, loggerCls *object.Class) error {
	if rootDict := lcfgDict(cfg, "root"); rootDict != nil {
		root := i.getOrCreateLogger(ls, loggerCls, "")
		if lvl := lcfgStr(rootDict, "level"); lvl != "" {
			root.Dict.SetStr("level", object.NewInt(int64(resolveLevelStr(ls, lvl))))
		}
	}
	if lgDict := lcfgDict(cfg, "loggers"); lgDict != nil {
		ks, vs := lgDict.Items()
		for idx, kobj := range ks {
			k, ok := kobj.(*object.Str)
			if !ok {
				continue
			}
			sub, ok := vs[idx].(*object.Dict)
			if !ok {
				continue
			}
			lg := i.getOrCreateLogger(ls, loggerCls, k.V)
			if lvl := lcfgStr(sub, "level"); lvl != "" {
				lg.Dict.SetStr("level", object.NewInt(int64(resolveLevelStr(ls, lvl))))
			}
			if v2, ok2 := sub.GetStr("propagate"); ok2 {
				lg.Dict.SetStr("propagate", v2)
			}
		}
	}
	return nil
}

// applyLoggerConfig sets level, handlers, propagate, filters on a logger instance.
// Handler and filter lists are replaced (not appended), matching CPython's dictConfig behavior.
func (i *Interp) applyLoggerConfig(lg *object.Instance, cfg *object.Dict, ls *logState, handlers map[string]*object.Instance, filters map[string]*object.Instance) {
	if lvl := lcfgStr(cfg, "level"); lvl != "" {
		lg.Dict.SetStr("level", object.NewInt(int64(resolveLevelStr(ls, lvl))))
	}
	if v, ok := cfg.GetStr("propagate"); ok {
		lg.Dict.SetStr("propagate", v)
	}
	// Replace (not append) handler list.
	if _, hasCfgHandlers := cfg.GetStr("handlers"); hasCfgHandlers {
		newList := &object.List{}
		for _, hid := range lcfgStrList(cfg, "handlers") {
			if h, ok := handlers[hid]; ok {
				newList.V = append(newList.V, h)
			}
		}
		lg.Dict.SetStr("handlers", newList)
	}
	// Replace filter list.
	if _, hasCfgFilters := cfg.GetStr("filters"); hasCfgFilters {
		newFList := &object.List{}
		for _, fid := range lcfgStrList(cfg, "filters") {
			if fi, ok := filters[fid]; ok {
				newFList.V = append(newFList.V, fi)
			}
		}
		lg.Dict.SetStr("filters", newFList)
	}
}

// buildHandlerFromDict creates a handler instance from a handler config dict.
func (i *Interp) buildHandlerFromDict(sub *object.Dict, ls *logState, handlerCls, streamHCls, fileHCls, nullHCls *object.Class) *object.Instance {
	className := lcfgStr(sub, "class")
	// Normalize: strip "logging." prefix.
	short := className
	if strings.HasPrefix(short, "logging.") {
		short = short[8:]
	}

	switch short {
	case "StreamHandler":
		streamVal := lcfgStr(sub, "stream")
		w := i.resolveStream(streamVal)
		if streamHCls == nil || handlerCls == nil {
			return nil
		}
		return newStreamHandlerInst(handlerCls, streamHCls, w)

	case "FileHandler":
		filename := lcfgStr(sub, "filename")
		if filename == "" {
			return nil
		}
		mode := lcfgStr(sub, "mode")
		if mode == "" {
			mode = "a"
		}
		flag := os.O_APPEND | os.O_CREATE | os.O_WRONLY
		if mode == "w" {
			flag = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
		}
		f, err := os.OpenFile(filename, flag, 0644)
		if err != nil {
			return nil
		}
		if fileHCls == nil || handlerCls == nil {
			return nil
		}
		return newFileHandlerInst(handlerCls, fileHCls, f)

	case "NullHandler":
		if nullHCls == nil {
			return nil
		}
		inst := &object.Instance{Class: nullHCls, Dict: object.NewDict()}
		inst.Dict.SetStr("level", object.NewInt(logNOTSET))
		inst.Dict.SetStr("formatter", object.None)
		inst.Dict.SetStr("filters", &object.List{V: nil})
		inst.Dict.SetStr("_emit", &object.BuiltinFunc{Name: "_emit", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
		return inst
	}
	return nil
}

// iniFile holds parsed sections from a logging .ini file.
type iniFile struct {
	sections map[string]map[string]string
}

func parseIniFile(path string) (*iniFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ini := &iniFile{sections: map[string]map[string]string{}}
	var cur map[string]string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r\n")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			sec := trimmed[1 : len(trimmed)-1]
			cur = map[string]string{}
			ini.sections[sec] = cur
			continue
		}
		if cur == nil {
			continue
		}
		idx := strings.IndexAny(trimmed, "=:")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:idx])
		val := strings.TrimSpace(trimmed[idx+1:])
		cur[key] = val
	}
	return ini, scanner.Err()
}

func (ini *iniFile) get(section, key string) string {
	s, ok := ini.sections[section]
	if !ok {
		return ""
	}
	return s[key]
}

// applyFileConfig parses an .ini logging config file and applies it.
func (i *Interp) applyFileConfig(fname string, disableExisting bool) error {
	ls, loggingMod, err := i.ensureLogState()
	if err != nil {
		return err
	}
	loggerCls, ok := ls.loggerClass.(*object.Class)
	if !ok {
		return fmt.Errorf("logging.config: logger class not found")
	}

	ini, err := parseIniFile(fname)
	if err != nil {
		return err
	}

	handlerCls := getLoggingClass(loggingMod, "Handler")
	streamHCls := getLoggingClass(loggingMod, "StreamHandler")
	fileHCls := getLoggingClass(loggingMod, "FileHandler")
	nullHCls := getLoggingClass(loggingMod, "NullHandler")
	fmtterCls := getLoggingClass(loggingMod, "Formatter")
	filterCls := getLoggingClass(loggingMod, "Filter")

	// --- parse formatter keys ---
	formatters := map[string]*object.Instance{}
	for _, fk := range splitKeys(ini.get("formatters", "keys")) {
		sec := "formatter_" + fk
		if fk == "root" {
			sec = "formatter_root"
		}
		fmtStr := ini.get(sec, "format")
		if fmtStr == "" {
			fmtStr = "%(levelname)s:%(name)s:%(message)s"
		}
		datefmt := ini.get(sec, "datefmt")
		if fmtterCls != nil {
			formatters[fk] = newFormatterInst(fmtterCls, fmtStr, datefmt)
		}
	}

	// --- parse filter keys ---
	filters := map[string]*object.Instance{}
	for _, fk := range splitKeys(ini.get("filters", "keys")) {
		sec := "filter_" + fk
		name := ini.get(sec, "name")
		if filterCls != nil {
			inst := &object.Instance{Class: filterCls, Dict: object.NewDict()}
			inst.Dict.SetStr("name", &object.Str{V: name})
			inst.Dict.SetStr("nlen", object.NewInt(int64(len(name))))
			filters[fk] = inst
		}
	}

	// --- parse handler keys ---
	handlers := map[string]*object.Instance{}
	for _, hk := range splitKeys(ini.get("handlers", "keys")) {
		sec := "handler_" + hk
		hdlr := i.buildHandlerFromIni(ini, sec, ls, handlerCls, streamHCls, fileHCls, nullHCls)
		if hdlr == nil {
			continue
		}
		// set level
		if lvl := ini.get(sec, "level"); lvl != "" && lvl != "NOTSET" {
			hdlr.Dict.SetStr("level", object.NewInt(int64(resolveLevelStr(ls, lvl))))
		}
		// set formatter
		if fk := ini.get(sec, "formatter"); fk != "" {
			if fi, ok := formatters[fk]; ok {
				hdlr.Dict.SetStr("formatter", fi)
			}
		}
		handlers[hk] = hdlr
	}

	configuredLoggers := map[string]bool{"root": true, "": true}

	// --- parse logger keys ---
	for _, lk := range splitKeys(ini.get("loggers", "keys")) {
		sec := "logger_" + lk
		var lg *object.Instance
		if lk == "root" {
			lg = i.getOrCreateLogger(ls, loggerCls, "")
		} else {
			qualname := ini.get(sec, "qualname")
			if qualname == "" {
				qualname = lk
			}
			lg = i.getOrCreateLogger(ls, loggerCls, qualname)
			configuredLoggers[qualname] = true
		}
		// set level
		if lvl := ini.get(sec, "level"); lvl != "" {
			lg.Dict.SetStr("level", object.NewInt(int64(resolveLevelStr(ls, lvl))))
		}
		// set propagate
		if prop := ini.get(sec, "propagate"); prop != "" {
			switch prop {
			case "0", "false", "False":
				lg.Dict.SetStr("propagate", object.False)
			default:
				lg.Dict.SetStr("propagate", object.True)
			}
		}
		// replace handlers (not append), matching CPython fileConfig behavior
		newHList := &object.List{}
		for _, hk := range splitKeys(ini.get(sec, "handlers")) {
			if h, ok := handlers[hk]; ok {
				newHList.V = append(newHList.V, h)
			}
		}
		lg.Dict.SetStr("handlers", newHList)
		_ = filters
	}

	if disableExisting {
		for name, lg := range ls.loggers {
			if configuredLoggers[name] {
				continue
			}
			lg.Dict.SetStr("disabled", object.True)
		}
	}

	return nil
}

// buildHandlerFromIni creates a handler from an ini section.
func (i *Interp) buildHandlerFromIni(ini *iniFile, sec string, ls *logState, handlerCls, streamHCls, fileHCls, nullHCls *object.Class) *object.Instance {
	className := ini.get(sec, "class")
	short := className
	if strings.HasPrefix(short, "logging.") {
		short = short[8:]
	}
	argsStr := strings.TrimSpace(ini.get(sec, "args"))

	switch short {
	case "StreamHandler":
		w := i.parseStreamArg(argsStr)
		if streamHCls == nil || handlerCls == nil {
			return nil
		}
		return newStreamHandlerInst(handlerCls, streamHCls, w)

	case "FileHandler":
		filename, mode := i.parseFileArgs(argsStr)
		if filename == "" {
			return nil
		}
		if mode == "" {
			mode = "a"
		}
		flag := os.O_APPEND | os.O_CREATE | os.O_WRONLY
		if mode == "w" {
			flag = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
		}
		f, err := os.OpenFile(filename, flag, 0644)
		if err != nil {
			return nil
		}
		if fileHCls == nil || handlerCls == nil {
			return nil
		}
		return newFileHandlerInst(handlerCls, fileHCls, f)

	case "NullHandler":
		if nullHCls == nil {
			return nil
		}
		inst := &object.Instance{Class: nullHCls, Dict: object.NewDict()}
		inst.Dict.SetStr("level", object.NewInt(logNOTSET))
		inst.Dict.SetStr("formatter", object.None)
		inst.Dict.SetStr("filters", &object.List{V: nil})
		inst.Dict.SetStr("_emit", &object.BuiltinFunc{Name: "_emit", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
		return inst
	}
	return nil
}

// parseStreamArg parses fileConfig handler args like "(sys.stdout,)" to a writer.
func (i *Interp) parseStreamArg(args string) interface{ Write([]byte) (int, error) } {
	if strings.Contains(args, "sys.stdout") {
		return i.Stdout
	}
	return i.Stderr
}

// parseFileArgs parses fileConfig handler args like "('file.log', 'w')" to filename+mode.
func (i *Interp) parseFileArgs(args string) (filename, mode string) {
	// Strip outer parens.
	s := strings.TrimSpace(args)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	s = strings.TrimRight(s, ",")
	parts := strings.SplitN(s, ",", 2)
	filename = unquoteSimple(strings.TrimSpace(parts[0]))
	if len(parts) >= 2 {
		mode = unquoteSimple(strings.TrimSpace(parts[1]))
	}
	return
}

// unquoteSimple strips single or double quotes from a quoted string literal.
func unquoteSimple(s string) string {
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// splitKeys splits a comma-separated keys string, trimming whitespace.
func splitKeys(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

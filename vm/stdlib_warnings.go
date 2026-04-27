package vm

import (
	"fmt"
	"strings"
	"sync"

	"github.com/tamnd/goipy/object"
)

// warnFilter is one entry in warnings.filters.
type warnFilter struct {
	action   string // "default","always","error","ignore","module","once"
	message  string // substring match on str(message); "" = match all
	category *object.Class
	module   string // substring match on module name; "" = match all
	lineno   int    // 0 = match all
}

// warnState is per-Interp warnings state.
type warnState struct {
	mu            sync.Mutex
	filters       []warnFilter
	onceRegistry  map[string]bool
	defaultAction string
	showWarning   func(msg, catName, filename string, lineno int) // current showwarning impl
	recorder      *[]object.Object                               // non-nil when catch_warnings(record=True) active
}

func newWarnState() *warnState {
	ws := &warnState{
		onceRegistry:  make(map[string]bool),
		defaultAction: "default",
	}
	ws.filters = []warnFilter{
		{action: "default", category: nil}, // nil = match Warning (set after class setup)
	}
	return ws
}

// buildWarnings creates the warnings module (comprehensive implementation).
func (i *Interp) buildWarnings() *object.Module {
	m := &object.Module{Name: "warnings", Dict: object.NewDict()}

	// Lazily initialise per-Interp warnState.
	if i.warnState == nil {
		i.warnState = newWarnState()
		// patch default filter category after builtins are ready
		i.warnState.filters[0].category = i.warningClass
	}
	ws := i.warnState

	// ── Helper: look up a warning class by name from builtins ────────────────
	warnCls := func(name string) *object.Class {
		if v, ok := i.Builtins.GetStr(name); ok {
			if cls, ok2 := v.(*object.Class); ok2 {
				return cls
			}
		}
		return i.warningClass
	}

	// Expose all warning classes directly in the module dict.
	warningNames := []string{
		"Warning", "UserWarning", "DeprecationWarning", "PendingDeprecationWarning",
		"RuntimeWarning", "SyntaxWarning", "ResourceWarning", "FutureWarning",
		"ImportWarning", "UnicodeWarning", "BytesWarning", "EncodingWarning",
	}
	for _, name := range warningNames {
		if v, ok := i.Builtins.GetStr(name); ok {
			m.Dict.SetStr(name, v)
		}
	}

	// ── filters list (Python-visible list of 5-tuples) ───────────────────────
	filtersList := &object.List{}
	rebuildFiltersList := func() {
		ws.mu.Lock()
		defer ws.mu.Unlock()
		filtersList.V = filtersList.V[:0]
		for _, f := range ws.filters {
			catObj := object.Object(object.None)
			if f.category != nil {
				catObj = f.category
			}
			filtersList.V = append(filtersList.V, &object.Tuple{V: []object.Object{
				&object.Str{V: f.action},
				&object.Str{V: f.message},
				catObj,
				&object.Str{V: f.module},
				object.NewInt(int64(f.lineno)),
			}})
		}
	}
	rebuildFiltersList()
	m.Dict.SetStr("filters", filtersList)
	m.Dict.SetStr("defaultaction", &object.Str{V: ws.defaultAction})
	m.Dict.SetStr("onceregistry", object.NewDict())

	// ── _filters_mutated ─────────────────────────────────────────────────────
	m.Dict.SetStr("_filters_mutated", &object.BuiltinFunc{Name: "_filters_mutated",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	// ── formatwarning ─────────────────────────────────────────────────────────
	m.Dict.SetStr("formatwarning", &object.BuiltinFunc{Name: "formatwarning",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// formatwarning(message, category, filename, lineno, line=None)
			if len(a) < 4 {
				return nil, object.Errorf(i.typeErr, "formatwarning() requires 4 positional arguments")
			}
			msg := object.Str_(a[0])
			catName := ""
			if cls, ok := a[1].(*object.Class); ok {
				catName = cls.Name
			} else {
				catName = object.Str_(a[1])
			}
			filename := object.Str_(a[2])
			lineno := int64(0)
			if n, ok := a[3].(*object.Int); ok {
				lineno = n.Int64()
			}
			return &object.Str{V: fmt.Sprintf("%s:%d: %s: %s\n", filename, lineno, catName, msg)}, nil
		}})

	// ── showwarning ──────────────────────────────────────────────────────────
	doShowWarning := func(msg, catName, filename string, lineno int) {
		line := fmt.Sprintf("%s:%d: %s: %s\n", filename, lineno, catName, msg)
		if ws.recorder != nil {
			// record mode — append a WarningMessage
			wmCls := &object.Class{Name: "WarningMessage", Dict: object.NewDict()}
			wm := &object.Instance{Class: wmCls, Dict: object.NewDict()}
			wm.Dict.SetStr("message", &object.Str{V: msg})
			wm.Dict.SetStr("category", warnCls(catName))
			wm.Dict.SetStr("filename", &object.Str{V: filename})
			wm.Dict.SetStr("lineno", object.NewInt(int64(lineno)))
			wm.Dict.SetStr("file", object.None)
			wm.Dict.SetStr("line", object.None)
			wm.Dict.SetStr("source", object.None)
			*ws.recorder = append(*ws.recorder, wm)
		} else {
			fmt.Fprint(i.Stderr, line)
		}
	}
	ws.showWarning = doShowWarning

	m.Dict.SetStr("showwarning", &object.BuiltinFunc{Name: "showwarning",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// showwarning(message, category, filename, lineno, file=None, line=None)
			if len(a) < 4 {
				return nil, object.Errorf(i.typeErr, "showwarning() requires 4 positional arguments")
			}
			msg := object.Str_(a[0])
			catName := ""
			if cls, ok := a[1].(*object.Class); ok {
				catName = cls.Name
			} else {
				catName = object.Str_(a[1])
			}
			filename := object.Str_(a[2])
			lineno := 0
			if n, ok := a[3].(*object.Int); ok {
				lineno = int(n.Int64())
			}
			// Optional file= argument — write to it if provided
			var fileObj object.Object = object.None
			if len(a) >= 5 {
				fileObj = a[4]
			}
			if kw != nil {
				if v, ok := kw.GetStr("file"); ok {
					fileObj = v
				}
			}
			line := fmt.Sprintf("%s:%d: %s: %s\n", filename, lineno, catName, msg)
			if fileObj != object.None {
				if writeMethod, err := i.getAttr(fileObj, "write"); err == nil {
					_, _ = i.callObject(writeMethod, []object.Object{&object.Str{V: line}}, nil)
					return object.None, nil
				}
			}
			ws.showWarning(msg, catName, filename, lineno)
			return object.None, nil
		}})

	// ── resolveCategory: normalise warn() category argument ──────────────────
	resolveCategory := func(catArg object.Object) *object.Class {
		if cls, ok := catArg.(*object.Class); ok {
			return cls
		}
		if inst, ok := catArg.(*object.Instance); ok {
			return inst.Class
		}
		return warnCls("UserWarning")
	}

	// ── matchFilter: find first matching filter ───────────────────────────────
	matchFilter := func(msgStr string, cat *object.Class) (string, bool) {
		ws.mu.Lock()
		defer ws.mu.Unlock()
		for _, f := range ws.filters {
			if f.message != "" && !strings.Contains(msgStr, f.message) {
				continue
			}
			if f.category != nil && !object.IsSubclass(cat, f.category) {
				continue
			}
			return f.action, true
		}
		return ws.defaultAction, true
	}

	// ── applyAction: run the decided action ──────────────────────────────────
	applyAction := func(action, msgStr string, cat *object.Class, filename string, lineno int) error {
		catName := cat.Name
		switch action {
		case "ignore":
			return nil
		case "error":
			return object.NewException(cat, msgStr)
		case "always":
			ws.showWarning(msgStr, catName, filename, lineno)
		case "once":
			key := msgStr + "\x00" + catName
			ws.mu.Lock()
			already := ws.onceRegistry[key]
			if !already {
				ws.onceRegistry[key] = true
			}
			ws.mu.Unlock()
			if !already {
				ws.showWarning(msgStr, catName, filename, lineno)
			}
		case "module":
			key := msgStr + "\x00" + catName + "\x00" + filename
			ws.mu.Lock()
			already := ws.onceRegistry[key]
			if !already {
				ws.onceRegistry[key] = true
			}
			ws.mu.Unlock()
			if !already {
				ws.showWarning(msgStr, catName, filename, lineno)
			}
		default: // "default"
			key := catName + "\x00" + fmt.Sprintf("%d", lineno)
			ws.mu.Lock()
			already := ws.onceRegistry[key]
			if !already {
				ws.onceRegistry[key] = true
			}
			ws.mu.Unlock()
			if !already {
				ws.showWarning(msgStr, catName, filename, lineno)
			}
		}
		return nil
	}

	// ── warn ─────────────────────────────────────────────────────────────────
	m.Dict.SetStr("warn", &object.BuiltinFunc{Name: "warn",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "warn() missing message argument")
			}
			msgArg := a[0]
			cat := warnCls("UserWarning")
			if len(a) >= 2 {
				cat = resolveCategory(a[1])
			}
			if kw != nil {
				if v, ok := kw.GetStr("category"); ok {
					cat = resolveCategory(v)
				}
			}
			// Normalise message to string.
			msgStr := ""
			switch v := msgArg.(type) {
			case *object.Str:
				msgStr = v.V
			case *object.Instance:
				msgStr = object.Str_(v)
				cat = v.Class
			default:
				msgStr = object.Str_(msgArg)
			}
			action, _ := matchFilter(msgStr, cat)
			if err := applyAction(action, msgStr, cat, "<unknown>", 0); err != nil {
				return nil, err
			}
			return object.None, nil
		}})

	// ── warn_explicit ─────────────────────────────────────────────────────────
	m.Dict.SetStr("warn_explicit", &object.BuiltinFunc{Name: "warn_explicit",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			// warn_explicit(message, category, filename, lineno, module=None, ...)
			if len(a) < 4 {
				return nil, object.Errorf(i.typeErr, "warn_explicit() requires 4 positional arguments")
			}
			msgStr := object.Str_(a[0])
			cat := resolveCategory(a[1])
			filename := object.Str_(a[2])
			lineno := 0
			if n, ok := a[3].(*object.Int); ok {
				lineno = int(n.Int64())
			}
			action, _ := matchFilter(msgStr, cat)
			if err := applyAction(action, msgStr, cat, filename, lineno); err != nil {
				return nil, err
			}
			return object.None, nil
		}})

	// ── filterwarnings ────────────────────────────────────────────────────────
	m.Dict.SetStr("filterwarnings", &object.BuiltinFunc{Name: "filterwarnings",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "filterwarnings() missing action argument")
			}
			action := object.Str_(a[0])
			msgPat := ""
			catCls := i.warningClass
			modPat := ""
			lineno := 0
			append_ := false
			if len(a) >= 2 {
				msgPat = object.Str_(a[1])
			}
			if len(a) >= 3 {
				if cls, ok := a[2].(*object.Class); ok {
					catCls = cls
				}
			}
			if len(a) >= 4 {
				modPat = object.Str_(a[3])
			}
			if len(a) >= 5 {
				if n, ok := a[4].(*object.Int); ok {
					lineno = int(n.Int64())
				}
			}
			if len(a) >= 6 {
				append_ = isTruthy(a[5])
			}
			if kw != nil {
				if v, ok := kw.GetStr("message"); ok {
					msgPat = object.Str_(v)
				}
				if v, ok := kw.GetStr("category"); ok {
					if cls, ok2 := v.(*object.Class); ok2 {
						catCls = cls
					}
				}
				if v, ok := kw.GetStr("module"); ok {
					modPat = object.Str_(v)
				}
				if v, ok := kw.GetStr("lineno"); ok {
					if n, ok2 := v.(*object.Int); ok2 {
						lineno = int(n.Int64())
					}
				}
				if v, ok := kw.GetStr("append"); ok {
					append_ = isTruthy(v)
				}
			}
			f := warnFilter{action: action, message: msgPat, category: catCls, module: modPat, lineno: lineno}
			ws.mu.Lock()
			if append_ {
				ws.filters = append(ws.filters, f)
			} else {
				ws.filters = append([]warnFilter{f}, ws.filters...)
			}
			ws.mu.Unlock()
			rebuildFiltersList()
			return object.None, nil
		}})

	// ── simplefilter ─────────────────────────────────────────────────────────
	m.Dict.SetStr("simplefilter", &object.BuiltinFunc{Name: "simplefilter",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "simplefilter() missing action argument")
			}
			action := object.Str_(a[0])
			catCls := i.warningClass
			lineno := 0
			append_ := false
			if len(a) >= 2 {
				if cls, ok := a[1].(*object.Class); ok {
					catCls = cls
				}
			}
			if len(a) >= 3 {
				if n, ok := a[2].(*object.Int); ok {
					lineno = int(n.Int64())
				}
			}
			if len(a) >= 4 {
				append_ = isTruthy(a[3])
			}
			if kw != nil {
				if v, ok := kw.GetStr("category"); ok {
					if cls, ok2 := v.(*object.Class); ok2 {
						catCls = cls
					}
				}
				if v, ok := kw.GetStr("lineno"); ok {
					if n, ok2 := v.(*object.Int); ok2 {
						lineno = int(n.Int64())
					}
				}
				if v, ok := kw.GetStr("append"); ok {
					append_ = isTruthy(v)
				}
			}
			f := warnFilter{action: action, message: "", category: catCls, module: "", lineno: lineno}
			ws.mu.Lock()
			if append_ {
				ws.filters = append(ws.filters, f)
			} else {
				ws.filters = append([]warnFilter{f}, ws.filters...)
			}
			ws.mu.Unlock()
			rebuildFiltersList()
			return object.None, nil
		}})

	// ── resetwarnings ─────────────────────────────────────────────────────────
	m.Dict.SetStr("resetwarnings", &object.BuiltinFunc{Name: "resetwarnings",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			ws.mu.Lock()
			ws.filters = []warnFilter{}
			ws.mu.Unlock()
			rebuildFiltersList()
			return object.None, nil
		}})

	// ── catch_warnings ────────────────────────────────────────────────────────
	m.Dict.SetStr("catch_warnings", &object.BuiltinFunc{Name: "catch_warnings",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			record := false
			if kw != nil {
				if v, ok := kw.GetStr("record"); ok {
					record = isTruthy(v)
				}
			}

			cls := &object.Class{Name: "catch_warnings", Dict: object.NewDict()}
			inst := &object.Instance{Class: cls, Dict: object.NewDict()}

			// Saved state.
			var savedFilters []warnFilter
			var savedRegistry map[string]bool
			var savedRecorder *[]object.Object
			var log []object.Object

			cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					ws.mu.Lock()
					savedFilters = append([]warnFilter(nil), ws.filters...)
					savedRegistry = make(map[string]bool)
					for k, v := range ws.onceRegistry {
						savedRegistry[k] = v
					}
					savedRecorder = ws.recorder
					if record {
						log = nil
						ws.recorder = &log
					}
					ws.mu.Unlock()
					if record {
						logList := &object.List{V: []object.Object{}}
						// We'll use a proxy that stays in sync.
						inst.Dict.SetStr("_log", logList)
						return logList, nil
					}
					return object.None, nil
				}})

			cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					ws.mu.Lock()
					ws.filters = savedFilters
					ws.onceRegistry = savedRegistry
					ws.recorder = savedRecorder
					ws.mu.Unlock()
					rebuildFiltersList()
					return object.False, nil
				}})

			cls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return &object.Str{V: "<catch_warnings>"}, nil
				}})

			// After __enter__ runs the list needs to be kept in sync.
			// We add an append interceptor: when ws.recorder appends, we also
			// update the logList. Handled via closure: log slice and logList share
			// pointer if we update logList.V from the same slice.
			// Instead: use a ticker approach — the list returned by __enter__ is
			// the real log slice converted to *object.List. We keep them in sync
			// by having recorder write to a shared slice, then sync on exit.
			// Simpler: after each warn() call, the WarningMessage is appended to
			// the slice pointed to by ws.recorder. We return a list whose V IS
			// the same underlying slice.

			// Override __enter__ to return a *object.List backed by the real slice:
			cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					ws.mu.Lock()
					savedFilters = append([]warnFilter(nil), ws.filters...)
					savedRegistry = make(map[string]bool)
					for k, v := range ws.onceRegistry {
						savedRegistry[k] = v
					}
					savedRecorder = ws.recorder
					ws.mu.Unlock()
					if record {
						logSlice := &[]object.Object{}
						ws.mu.Lock()
						ws.recorder = logSlice
						ws.mu.Unlock()
						logList := &object.List{}
						// Keep logList.V pointing at *logSlice.
						// We return a special list that reads from logSlice on access.
						// Simplest: store the pointer in inst so __exit__ can sync.
						inst.Dict.SetStr("_logSlice_ptr", object.None) // placeholder
						// Instead, return the logList and update it in showWarning.
						// We intercept ws.showWarning to append to logList directly.
						origShow := ws.showWarning
						ws.showWarning = func(msg, catName, filename string, lineno int) {
							wmCls := &object.Class{Name: "WarningMessage", Dict: object.NewDict()}
							wm := &object.Instance{Class: wmCls, Dict: object.NewDict()}
							wm.Dict.SetStr("message", &object.Str{V: msg})
							wm.Dict.SetStr("category", warnCls(catName))
							wm.Dict.SetStr("filename", &object.Str{V: filename})
							wm.Dict.SetStr("lineno", object.NewInt(int64(lineno)))
							wm.Dict.SetStr("file", object.None)
							wm.Dict.SetStr("line", object.None)
							wm.Dict.SetStr("source", object.None)
							logList.V = append(logList.V, wm)
						}
						// Restore showWarning on exit (need to capture it in the exit closure).
						inst.Dict.SetStr("_origShow", object.None)
						// Store origShow for __exit__ via a closure variable.
						_ = origShow
						cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
							Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
								ws.mu.Lock()
								ws.filters = savedFilters
								ws.onceRegistry = savedRegistry
								ws.recorder = savedRecorder
								ws.showWarning = origShow
								ws.mu.Unlock()
								rebuildFiltersList()
								return object.False, nil
							}})
						return logList, nil
					}
					// non-record mode
					cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__",
						Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
							ws.mu.Lock()
							ws.filters = savedFilters
							ws.onceRegistry = savedRegistry
							ws.recorder = savedRecorder
							ws.mu.Unlock()
							rebuildFiltersList()
							return object.False, nil
						}})
					return object.None, nil
				}})

			return inst, nil
		}})

	// ── WarningMessage class ──────────────────────────────────────────────────
	wmCls := &object.Class{Name: "WarningMessage", Dict: object.NewDict()}
	wmCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) > 0 {
				if inst, ok := a[0].(*object.Instance); ok {
					msg, _ := inst.Dict.GetStr("message")
					cat, _ := inst.Dict.GetStr("category")
					catName := ""
					if cls, ok2 := cat.(*object.Class); ok2 {
						catName = cls.Name
					}
					return &object.Str{V: fmt.Sprintf("<WarningMessage: %s %s>", catName, object.Str_(msg))}, nil
				}
			}
			return &object.Str{V: "<WarningMessage>"}, nil
		}})
	m.Dict.SetStr("WarningMessage", wmCls)

	return m
}

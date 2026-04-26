package vm

import (
	"github.com/tamnd/goipy/object"
)

// builtinModule returns VM-provided modules that don't need a .pyc.
func (i *Interp) builtinModule(name string) (*object.Module, bool) {
	switch name {
	case "asyncio":
		return i.buildAsyncio(), true
	case "importlib":
		return i.buildImportlib(), true
	case "functools":
		return i.buildFunctools(), true
	case "itertools":
		return i.buildItertools(), true
	case "array":
		return i.buildArray(), true
	case "weakref":
		return i.buildWeakref(), true
	case "collections":
		return i.buildCollections(), true
	case "collections.abc":
		return i.buildCollectionsAbc(), true
	case "operator":
		return i.buildOperator(), true
	case "math":
		return i.buildMath(), true
	case "heapq":
		return i.buildHeapq(), true
	case "bisect":
		return i.buildBisect(), true
	case "random":
		return i.buildRandom(), true
	case "json":
		return i.buildJSON(), true
	case "re":
		return i.buildRe(), true
	case "string":
		return i.buildStringMod(), true
	case "copy":
		return i.buildCopy(), true
	case "io":
		return i.buildIO(), true
	case "hashlib":
		return i.buildHashlib(), true
	case "base64":
		return i.buildBase64(), true
	case "textwrap":
		return i.buildTextwrap(), true
	case "unicodedata":
		return i.buildUnicodedata(), true
	case "stringprep":
		return i.buildStringprep(), true
	case "struct":
		return i.buildStruct(), true
	case "csv":
		return i.buildCsv(), true
	case "urllib":
		return i.buildUrllib(), true
	case "urllib.parse":
		return i.buildUrllibParse(), true
	case "urllib.error":
		return i.buildUrllibError(), true
	case "urllib.request":
		return i.buildUrllibRequest(), true
	case "urllib.robotparser":
		return i.buildUrllibRobotParser(), true
	case "urllib.response":
		return i.buildUrllib(), true
	case "zlib":
		return i.buildZlib(), true
	case "binascii":
		return i.buildBinascii(), true
	case "hmac":
		return i.buildHmac(), true
	case "secrets":
		return i.buildSecrets(), true
	case "uuid":
		return i.buildUuid(), true
	case "configparser":
		return i.buildConfigParser(), true
	case "tomllib":
		return i.buildTomllib(), true
	case "netrc":
		return i.buildNetrc(), true
	case "difflib":
		return i.buildDifflib(), true
	case "shlex":
		return i.buildShlex(), true
	case "gzip":
		return i.buildGzip(), true
	case "bz2":
		return i.buildBz2(), true
	case "lzma":
		return i.buildLzma(), true
	case "zipfile":
		return i.buildZipfile(), true
	case "tarfile":
		return i.buildTarfile(), true
	case "compression":
		return i.buildCompression(), true
	case "compression.zstd":
		return i.buildCompressionZstd(), true
	case "fnmatch":
		return i.buildFnmatch(), true
	case "glob":
		return i.buildGlob(), true
	case "statistics":
		return i.buildStatistics(), true
	case "calendar":
		return i.buildCalendar(), true
	case "plistlib":
		return i.buildPlistlib(), true
	case "pprint":
		return i.buildPprint(), true
	case "reprlib":
		return i.buildReprlib(), true
	case "enum":
		return i.buildEnum(), true
	case "graphlib":
		return i.buildGraphlib(), true
	case "numbers":
		return i.buildNumbers(), true
	case "cmath":
		return i.buildCmath(), true
	case "ftplib":
		return i.buildFtplib(), true
	case "poplib":
		return i.buildPoplib(), true
	case "imaplib":
		return i.buildImaplib(), true
	case "smtplib":
		return i.buildSmtplib(), true
	case "socketserver":
		return i.buildSocketserver(), true
	case "http.server":
		return i.buildHttpServer(), true
	case "http":
		return i.buildHttp(), true
	case "http.client":
		return i.buildHttpClient(), true
	case "html":
		return i.buildHtml(), true
	case "html.parser":
		return i.buildHtmlParser(), true
	case "html.entities":
		return i.buildHtmlEntities(), true
	case "sys":
		return i.buildSys(), true
	case "time":
		return i.buildTime(), true
	case "os":
		return i.buildOs(), true
	case "os.path":
		return i.buildOsPath(), true
	case "warnings":
		return i.buildWarnings(), true
	case "threading":
		return i.buildThreading(), true
	case "multiprocessing":
		return i.buildMultiprocessing(), true
	case "multiprocessing.shared_memory":
		return i.buildSharedMemory(), true
	case "concurrent":
		// Namespace package for concurrent.futures / concurrent.interpreters.
		m := &object.Module{Name: "concurrent", Dict: object.NewDict()}
		m.Dict.SetStr("__path__", &object.List{V: []object.Object{&object.Str{V: ""}}})
		m.Dict.SetStr("__package__", &object.Str{V: "concurrent"})
		return m, true
	case "concurrent.futures":
		return i.buildConcurrentFutures(), true
	case "concurrent.interpreters":
		return i.buildConcurrentInterpreters(), true
	case "subprocess":
		return i.buildSubprocess(), true
	case "sched":
		return i.buildSched(), true
	case "queue":
		return i.buildQueue(), true
	case "contextvars":
		return i.buildContextvars(), true
	case "_thread":
		return i.buildThread(), true
	case "string.templatelib":
		return i.buildTemplatelib(), true
	case "cmd":
		return i.buildCmd(), true
	case "readline":
		return i.buildReadline(), true
	case "rlcompleter":
		return i.buildRlcompleter(), true
	case "codecs":
		return i.buildCodecs(), true
	case "datetime":
		return i.buildDatetime(), true
	case "zoneinfo":
		return i.buildZoneinfo(), true
	case "types":
		return i.buildTypes(), true
	case "decimal":
		return i.buildDecimal(), true
	case "fractions":
		return i.buildFractions(), true
	case "pathlib":
		return i.buildPathlib(), true
	case "tempfile":
		return i.buildTempfile(), true
	case "stat":
		return i.buildStat(), true
	case "filecmp":
		return i.buildFilecmp(), true
	case "linecache":
		return i.buildLinecache(), true
	case "shutil":
		return i.buildShutil(), true
	case "pickle":
		return i.buildPickle(), true
	case "copyreg":
		return i.buildCopyreg(), true
	case "shelve":
		return i.buildShelve(), true
	case "marshal":
		return i.buildMarshal(), true
	case "dbm":
		return i.buildDbm(), true
	case "dbm.sqlite3":
		return i.buildDbmSqlite3(), true
	case "sqlite3":
		return i.buildSqlite3(), true
	case "logging":
		return i.buildLogging(), true
	case "logging.config":
		return i.buildLoggingConfig(), true
	case "logging.handlers":
		return i.buildLoggingHandlers(), true
	case "platform":
		return i.buildPlatform(), true
	case "errno":
		return i.buildErrno(), true
	case "ctypes":
		return i.buildCtypes(), true
	case "argparse":
		return i.buildArgparse(), true
	case "optparse":
		return i.buildOptparse(), true
	case "getpass":
		return i.buildGetpass(), true
	case "fileinput":
		return i.buildFileinput(), true
	case "curses":
		return i.buildCurses(), true
	case "curses.ascii":
		return i.buildCursesAscii(), true
	case "curses.textpad":
		return i.buildCursesTextpad(), true
	case "curses.panel":
		return i.buildCursesPanel(), true
	case "socket":
		return i.buildSocket(), true
	case "ssl":
		return i.buildSSL(), true
	case "select":
		return i.buildSelect(), true
	case "selectors":
		return i.buildSelectors(), true
	case "signal":
		return i.buildSignal(), true
	case "mmap":
		return i.buildMmap(), true
	case "email":
		return i.buildEmail(), true
	case "email.message":
		return i.buildEmailMessage(), true
	case "email.mime":
		return i.buildEmailMime(), true
	case "email.mime.base":
		return i.buildEmailMimeBase(), true
	case "email.mime.nonmultipart":
		return i.buildEmailMimeNonMultipart(), true
	case "email.mime.multipart":
		return i.buildEmailMimeMultipart(), true
	case "email.mime.text":
		return i.buildEmailMimeText(), true
	case "email.mime.application":
		return i.buildEmailMimeApplication(), true
	case "email.mime.image":
		return i.buildEmailMimeImage(), true
	case "email.mime.audio":
		return i.buildEmailMimeAudio(), true
	case "email.mime.message":
		return i.buildEmailMimeMessage(), true
	case "email.utils":
		return i.buildEmailUtils(), true
	case "email.header":
		return i.buildEmailHeader(), true
	case "email.encoders":
		return i.buildEmailEncoders(), true
	case "email.errors":
		return i.buildEmailErrors(), true
	case "email.generator":
		return i.buildEmailGenerator(), true
	case "email.parser":
		return i.buildEmailParser(), true
	case "email.policy":
		return i.buildEmailPolicy(), true
	case "email.charset":
		return i.buildEmailCharset(), true
	case "email.headerregistry":
		return &object.Module{Name: "email.headerregistry", Dict: object.NewDict()}, true
	case "mailbox":
		return i.buildMailbox(), true
	case "mimetypes":
		return i.buildMimetypes(), true
	case "quopri":
		return i.buildQuopri(), true
	case "xml":
		return i.buildXml(), true
	case "xml.etree":
		return i.buildXmlEtree(), true
	case "xml.etree.ElementTree":
		return i.buildXmlElementTree(), true
	case "xml.sax":
		return i.buildXmlSax(), true
	case "xml.sax.handler":
		return i.buildXmlSaxHandler(), true
	case "xml.sax.saxutils":
		return i.buildXmlSaxUtils(), true
	case "xml.sax.xmlreader":
		return i.buildXmlSaxXmlReader(), true
	case "xml.dom":
		return i.buildXmlDom(), true
	case "xml.dom.minidom":
		return i.buildXmlDomMinidom(), true
	case "xml.dom.pulldom":
		return i.buildXmlDomPulldom(), true
	case "xml.parsers":
		return i.buildXmlParsers(), true
	case "xml.parsers.expat":
		return i.buildXmlParsersExpat(), true
	case "pyexpat":
		return i.buildPyexpat(), true
	case "pyexpat.errors":
		m, _ := i.loadModule("pyexpat")
		if sub, ok := m.Dict.GetStr("errors"); ok {
			if sm, ok2 := sub.(*object.Module); ok2 {
				return sm, true
			}
		}
		return nil, false
	case "pyexpat.model":
		m, _ := i.loadModule("pyexpat")
		if sub, ok := m.Dict.GetStr("model"); ok {
			if sm, ok2 := sub.(*object.Module); ok2 {
				return sm, true
			}
		}
		return nil, false
	case "webbrowser":
		return i.buildWebbrowser(), true
	case "wsgiref":
		return i.buildWsgiref(), true
	case "wsgiref.util":
		return i.buildWsgirefUtil(), true
	case "wsgiref.headers":
		return i.buildWsgirefHeaders(), true
	case "wsgiref.simple_server":
		return i.buildWsgirefSimpleServer(), true
	case "wsgiref.handlers":
		return i.buildWsgirefHandlers(), true
	case "wsgiref.validate":
		return i.buildWsgirefValidate(), true
	case "wsgiref.types":
		return i.buildWsgirefTypes(), true
	}
	return nil, false
}

// buildAsyncio constructs the asyncio module with the full CPython 3.14 API
// surface. The runtime has no real event loop; coroutines are driven
// synchronously to completion.
func (i *Interp) buildAsyncio() *object.Module {
	m := &object.Module{Name: "asyncio", Dict: object.NewDict()}

	// makeIter creates a one-shot awaitable that resolves to val.
	makeIter := func(val object.Object) *object.Iter {
		done := false
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if done {
				return nil, false, nil
			}
			done = true
			exc := object.NewException(i.stopIter, "")
			exc.Args = &object.Tuple{V: []object.Object{val}}
			return nil, false, exc
		}}
	}

	// makeErrIter creates a one-shot awaitable that raises err when driven.
	makeErrIter := func(err error) *object.Iter {
		done := false
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if done {
				return nil, false, nil
			}
			done = true
			return nil, false, err
		}}
	}

	// ── exception classes ─────────────────────────────────────────────────

	cancelledErrCls := &object.Class{
		Name:  "CancelledError",
		Dict:  object.NewDict(),
		Bases: []*object.Class{i.baseExc},
	}
	m.Dict.SetStr("CancelledError", cancelledErrCls)

	timeoutErrCls := &object.Class{
		Name:  "TimeoutError",
		Dict:  object.NewDict(),
		Bases: []*object.Class{i.exception},
	}
	m.Dict.SetStr("TimeoutError", timeoutErrCls)

	invalidStateErrCls := &object.Class{
		Name:  "InvalidStateError",
		Dict:  object.NewDict(),
		Bases: []*object.Class{i.exception},
	}
	m.Dict.SetStr("InvalidStateError", invalidStateErrCls)

	// ── constants ─────────────────────────────────────────────────────────

	m.Dict.SetStr("FIRST_COMPLETED", &object.Str{V: "FIRST_COMPLETED"})
	m.Dict.SetStr("FIRST_EXCEPTION", &object.Str{V: "FIRST_EXCEPTION"})
	m.Dict.SetStr("ALL_COMPLETED", &object.Str{V: "ALL_COMPLETED"})

	// ── Future ────────────────────────────────────────────────────────────

	futureCls := &object.Class{Name: "Future", Dict: object.NewDict()}

	type futureState struct {
		done      bool
		cancelled bool
		result    object.Object
		excVal    error
		callbacks []object.Object
	}

	makeFuture := func() *object.Instance {
		st := &futureState{}
		inst := &object.Instance{Class: futureCls, Dict: object.NewDict()}

		runCallbacks := func() {
			for _, cb := range st.callbacks {
				i.callObject(cb, []object.Object{inst}, nil) //nolint
			}
			st.callbacks = nil
		}

		inst.Dict.SetStr("done", &object.BuiltinFunc{Name: "done",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.BoolOf(st.done), nil
			}})

		inst.Dict.SetStr("cancelled", &object.BuiltinFunc{Name: "cancelled",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.BoolOf(st.cancelled), nil
			}})

		inst.Dict.SetStr("result", &object.BuiltinFunc{Name: "result",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if st.cancelled {
					return nil, object.NewException(cancelledErrCls, "")
				}
				if !st.done {
					return nil, object.NewException(invalidStateErrCls, "result is not ready")
				}
				if st.excVal != nil {
					return nil, st.excVal
				}
				if st.result == nil {
					return object.None, nil
				}
				return st.result, nil
			}})

		inst.Dict.SetStr("exception", &object.BuiltinFunc{Name: "exception",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if st.cancelled {
					return nil, object.NewException(cancelledErrCls, "")
				}
				if !st.done {
					return nil, object.NewException(invalidStateErrCls, "result is not ready")
				}
				if st.excVal == nil {
					return object.None, nil
				}
				if exc, ok := st.excVal.(*object.Exception); ok {
					return exc, nil
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("set_result", &object.BuiltinFunc{Name: "set_result",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if st.done {
					return nil, object.NewException(invalidStateErrCls, "result is already set")
				}
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "set_result() requires 1 argument")
				}
				st.result = a[0]
				st.done = true
				runCallbacks()
				return object.None, nil
			}})

		inst.Dict.SetStr("set_exception", &object.BuiltinFunc{Name: "set_exception",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if st.done {
					return nil, object.NewException(invalidStateErrCls, "result is already set")
				}
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "set_exception() requires 1 argument")
				}
				switch e := a[0].(type) {
				case *object.Exception:
					st.excVal = e
				case *object.Class:
					st.excVal = object.NewException(e, "")
				default:
					return nil, object.Errorf(i.typeErr, "exception must be an exception instance or type")
				}
				st.done = true
				runCallbacks()
				return object.None, nil
			}})

		inst.Dict.SetStr("cancel", &object.BuiltinFunc{Name: "cancel",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if st.done {
					return object.False, nil
				}
				st.done = true
				st.cancelled = true
				runCallbacks()
				return object.True, nil
			}})

		inst.Dict.SetStr("add_done_callback", &object.BuiltinFunc{Name: "add_done_callback",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "add_done_callback() requires 1 argument")
				}
				if st.done {
					i.callObject(a[0], []object.Object{inst}, nil) //nolint
				} else {
					st.callbacks = append(st.callbacks, a[0])
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("remove_done_callback", &object.BuiltinFunc{Name: "remove_done_callback",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "remove_done_callback() requires 1 argument")
				}
				cb := a[0]
				var newCbs []object.Object
				removed := 0
				for _, c := range st.callbacks {
					if c == cb {
						removed++
					} else {
						newCbs = append(newCbs, c)
					}
				}
				st.callbacks = newCbs
				return object.NewInt(int64(removed)), nil
			}})

		inst.Dict.SetStr("__await__", &object.BuiltinFunc{Name: "__await__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if st.cancelled {
					return makeErrIter(object.NewException(cancelledErrCls, "")), nil
				}
				if st.excVal != nil {
					return makeErrIter(st.excVal), nil
				}
				result := st.result
				if result == nil {
					result = object.None
				}
				return makeIter(result), nil
			}})

		return inst
	}

	m.Dict.SetStr("Future", &object.BuiltinFunc{Name: "Future",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return makeFuture(), nil
		}})

	// ── Task ──────────────────────────────────────────────────────────────

	taskCls := &object.Class{Name: "Task", Dict: object.NewDict()}

	type taskState struct {
		done      bool
		cancelled bool
		result    object.Object
		excVal    error
		name      string
	}

	var createTask func(coro object.Object, name string) (object.Object, error)
	createTask = func(coro object.Object, name string) (object.Object, error) {
		st := &taskState{name: name}
		inst := &object.Instance{Class: taskCls, Dict: object.NewDict()}

		result, err := i.driveCoroutine(coro)
		if err != nil {
			if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, cancelledErrCls) {
				st.cancelled = true
			} else {
				st.excVal = err
			}
		} else {
			st.result = result
		}
		st.done = true

		inst.Dict.SetStr("done", &object.BuiltinFunc{Name: "done",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.True, nil
			}})

		inst.Dict.SetStr("cancelled", &object.BuiltinFunc{Name: "cancelled",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.BoolOf(st.cancelled), nil
			}})

		inst.Dict.SetStr("result", &object.BuiltinFunc{Name: "result",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if st.cancelled {
					return nil, object.NewException(cancelledErrCls, "")
				}
				if st.excVal != nil {
					return nil, st.excVal
				}
				if st.result == nil {
					return object.None, nil
				}
				return st.result, nil
			}})

		inst.Dict.SetStr("exception", &object.BuiltinFunc{Name: "exception",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if st.cancelled {
					return nil, object.NewException(cancelledErrCls, "")
				}
				if st.excVal == nil {
					return object.None, nil
				}
				if exc, ok := st.excVal.(*object.Exception); ok {
					return exc, nil
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("cancel", &object.BuiltinFunc{Name: "cancel",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.False, nil
			}})

		inst.Dict.SetStr("get_name", &object.BuiltinFunc{Name: "get_name",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return &object.Str{V: st.name}, nil
			}})

		inst.Dict.SetStr("set_name", &object.BuiltinFunc{Name: "set_name",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if len(a) > 0 {
					if s, ok := a[0].(*object.Str); ok {
						st.name = s.V
					}
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("add_done_callback", &object.BuiltinFunc{Name: "add_done_callback",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "add_done_callback() requires 1 argument")
				}
				i.callObject(a[0], []object.Object{inst}, nil) //nolint
				return object.None, nil
			}})

		inst.Dict.SetStr("remove_done_callback", &object.BuiltinFunc{Name: "remove_done_callback",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.NewInt(0), nil
			}})

		inst.Dict.SetStr("__await__", &object.BuiltinFunc{Name: "__await__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if st.cancelled {
					return makeErrIter(object.NewException(cancelledErrCls, "")), nil
				}
				if st.excVal != nil {
					return makeErrIter(st.excVal), nil
				}
				result := st.result
				if result == nil {
					result = object.None
				}
				return makeIter(result), nil
			}})

		return inst, nil
	}

	m.Dict.SetStr("create_task", &object.BuiltinFunc{Name: "create_task",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "create_task() requires a coroutine")
			}
			name := ""
			if kw != nil {
				if v, ok := kw.GetStr("name"); ok && v != object.None {
					if s, ok2 := v.(*object.Str); ok2 {
						name = s.V
					}
				}
			}
			return createTask(a[0], name)
		}})

	// ── run ───────────────────────────────────────────────────────────────

	m.Dict.SetStr("run", &object.BuiltinFunc{Name: "run",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "asyncio.run() missing coroutine")
			}
			return i.driveCoroutine(a[0])
		}})

	// ── sleep ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("sleep", &object.BuiltinFunc{Name: "sleep",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			var result object.Object = object.None
			if len(a) > 1 {
				result = a[1]
			}
			return makeIter(result), nil
		}})

	// ── gather ────────────────────────────────────────────────────────────

	m.Dict.SetStr("gather", &object.BuiltinFunc{Name: "gather",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			results := make([]object.Object, len(a))
			for k, c := range a {
				v, err := i.driveCoroutine(c)
				if err != nil {
					return nil, err
				}
				results[k] = v
			}
			return makeIter(&object.List{V: results}), nil
		}})

	// ── ensure_future ─────────────────────────────────────────────────────

	m.Dict.SetStr("ensure_future", &object.BuiltinFunc{Name: "ensure_future",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "ensure_future() requires an argument")
			}
			switch aw := a[0].(type) {
			case *object.Instance:
				return aw, nil
			default:
				return createTask(aw, "")
			}
		}})

	// ── wait ──────────────────────────────────────────────────────────────

	m.Dict.SetStr("wait", &object.BuiltinFunc{Name: "wait",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "wait() requires awaitables")
			}
			var tasks []object.Object
			switch t := a[0].(type) {
			case *object.List:
				tasks = t.V
			case *object.Tuple:
				tasks = t.V
			case *object.Set:
				tasks = t.Items()
			default:
				tasks = a
			}

			doneSet := object.NewSet()
			for _, t := range tasks {
				switch x := t.(type) {
				case *object.Instance:
					_ = doneSet.Add(x)
				default:
					task, err := createTask(t, "")
					if err != nil {
						return nil, err
					}
					_ = doneSet.Add(task)
				}
			}
			pendingSet := object.NewSet()

			return makeIter(&object.Tuple{V: []object.Object{doneSet, pendingSet}}), nil
		}})

	// ── wait_for ──────────────────────────────────────────────────────────

	m.Dict.SetStr("wait_for", &object.BuiltinFunc{Name: "wait_for",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "wait_for() requires an awaitable")
			}
			result, err := i.driveCoroutine(a[0])
			if err != nil {
				return nil, err
			}
			return makeIter(result), nil
		}})

	// ── as_completed ──────────────────────────────────────────────────────

	m.Dict.SetStr("as_completed", &object.BuiltinFunc{Name: "as_completed",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "as_completed() requires awaitables")
			}
			var tasks []object.Object
			switch t := a[0].(type) {
			case *object.List:
				tasks = t.V
			case *object.Tuple:
				tasks = t.V
			default:
				tasks = a
			}

			results := make([]object.Object, 0, len(tasks))
			for _, t := range tasks {
				switch x := t.(type) {
				case *object.Instance:
					if resFn, ok := x.Dict.GetStr("result"); ok {
						r, err := i.callObject(resFn, []object.Object{x}, nil)
						if err != nil {
							return nil, err
						}
						results = append(results, r)
					} else {
						results = append(results, object.None)
					}
				default:
					r, err := i.driveCoroutine(t)
					if err != nil {
						return nil, err
					}
					results = append(results, r)
				}
			}

			idx := 0
			return &object.Iter{Next: func() (object.Object, bool, error) {
				if idx >= len(results) {
					return nil, false, nil
				}
				r := results[idx]
				idx++
				return makeIter(r), true, nil
			}}, nil
		}})

	// ── current_task / all_tasks ──────────────────────────────────────────

	m.Dict.SetStr("current_task", &object.BuiltinFunc{Name: "current_task",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	m.Dict.SetStr("all_tasks", &object.BuiltinFunc{Name: "all_tasks",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewSet(), nil
		}})

	// ── event loop ────────────────────────────────────────────────────────

	makeEventLoop := func() *object.Instance {
		loopCls := &object.Class{Name: "EventLoop", Dict: object.NewDict()}
		inst := &object.Instance{Class: loopCls, Dict: object.NewDict()}
		closed := false

		inst.Dict.SetStr("run_until_complete", &object.BuiltinFunc{Name: "run_until_complete",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if len(a) == 0 {
					return nil, object.Errorf(i.typeErr, "run_until_complete() requires a coroutine")
				}
				return i.driveCoroutine(a[0])
			}})

		inst.Dict.SetStr("call_soon", &object.BuiltinFunc{Name: "call_soon",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if len(a) > 0 {
					var args []object.Object
					if len(a) > 1 {
						args = a[1:]
					}
					i.callObject(a[0], args, nil) //nolint
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("call_later", &object.BuiltinFunc{Name: "call_later",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if len(a) > 1 {
					var args []object.Object
					if len(a) > 2 {
						args = a[2:]
					}
					i.callObject(a[1], args, nil) //nolint
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				closed = true
				return object.None, nil
			}})

		inst.Dict.SetStr("is_closed", &object.BuiltinFunc{Name: "is_closed",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.BoolOf(closed), nil
			}})

		inst.Dict.SetStr("is_running", &object.BuiltinFunc{Name: "is_running",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.False, nil
			}})

		inst.Dict.SetStr("run_forever", &object.BuiltinFunc{Name: "run_forever",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}})

		inst.Dict.SetStr("stop", &object.BuiltinFunc{Name: "stop",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}})

		return inst
	}

	m.Dict.SetStr("new_event_loop", &object.BuiltinFunc{Name: "new_event_loop",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return makeEventLoop(), nil
		}})

	m.Dict.SetStr("get_event_loop", &object.BuiltinFunc{Name: "get_event_loop",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return makeEventLoop(), nil
		}})

	m.Dict.SetStr("get_running_loop", &object.BuiltinFunc{Name: "get_running_loop",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return makeEventLoop(), nil
		}})

	// ── Lock ──────────────────────────────────────────────────────────────

	m.Dict.SetStr("Lock", &object.BuiltinFunc{Name: "Lock",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			lockCls := &object.Class{Name: "Lock", Dict: object.NewDict()}
			inst := &object.Instance{Class: lockCls, Dict: object.NewDict()}
			locked := false

			inst.Dict.SetStr("locked", &object.BuiltinFunc{Name: "locked",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return object.BoolOf(locked), nil
				}})

			inst.Dict.SetStr("acquire", &object.BuiltinFunc{Name: "acquire",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					locked = true
					return makeIter(object.True), nil
				}})

			inst.Dict.SetStr("release", &object.BuiltinFunc{Name: "release",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					locked = false
					return object.None, nil
				}})

			inst.Dict.SetStr("__aenter__", &object.BuiltinFunc{Name: "__aenter__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					locked = true
					return makeIter(inst), nil
				}})

			inst.Dict.SetStr("__aexit__", &object.BuiltinFunc{Name: "__aexit__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					locked = false
					return makeIter(object.False), nil
				}})

			return inst, nil
		}})

	// ── Event ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("Event", &object.BuiltinFunc{Name: "Event",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			evCls := &object.Class{Name: "Event", Dict: object.NewDict()}
			inst := &object.Instance{Class: evCls, Dict: object.NewDict()}
			isSet := false

			inst.Dict.SetStr("is_set", &object.BuiltinFunc{Name: "is_set",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return object.BoolOf(isSet), nil
				}})

			inst.Dict.SetStr("set", &object.BuiltinFunc{Name: "set",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					isSet = true
					return object.None, nil
				}})

			inst.Dict.SetStr("clear", &object.BuiltinFunc{Name: "clear",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					isSet = false
					return object.None, nil
				}})

			inst.Dict.SetStr("wait", &object.BuiltinFunc{Name: "wait",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return makeIter(object.BoolOf(isSet)), nil
				}})

			return inst, nil
		}})

	// ── Condition ─────────────────────────────────────────────────────────

	m.Dict.SetStr("Condition", &object.BuiltinFunc{Name: "Condition",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			condCls := &object.Class{Name: "Condition", Dict: object.NewDict()}
			inst := &object.Instance{Class: condCls, Dict: object.NewDict()}

			inst.Dict.SetStr("acquire", &object.BuiltinFunc{Name: "acquire",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return makeIter(object.True), nil
				}})

			inst.Dict.SetStr("release", &object.BuiltinFunc{Name: "release",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return object.None, nil
				}})

			inst.Dict.SetStr("notify", &object.BuiltinFunc{Name: "notify",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return object.None, nil
				}})

			inst.Dict.SetStr("notify_all", &object.BuiltinFunc{Name: "notify_all",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return object.None, nil
				}})

			inst.Dict.SetStr("wait", &object.BuiltinFunc{Name: "wait",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return makeIter(object.True), nil
				}})

			inst.Dict.SetStr("__aenter__", &object.BuiltinFunc{Name: "__aenter__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return makeIter(inst), nil
				}})

			inst.Dict.SetStr("__aexit__", &object.BuiltinFunc{Name: "__aexit__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return makeIter(object.False), nil
				}})

			return inst, nil
		}})

	// ── Semaphore / BoundedSemaphore ──────────────────────────────────────

	makeSemaphore := func(value int, bounded bool) (*object.Instance, error) {
		clsName := "Semaphore"
		if bounded {
			clsName = "BoundedSemaphore"
		}
		semCls := &object.Class{Name: clsName, Dict: object.NewDict()}
		inst := &object.Instance{Class: semCls, Dict: object.NewDict()}
		count := value
		initial := value

		inst.Dict.SetStr("locked", &object.BuiltinFunc{Name: "locked",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.BoolOf(count == 0), nil
			}})

		inst.Dict.SetStr("acquire", &object.BuiltinFunc{Name: "acquire",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if count > 0 {
					count--
				}
				return makeIter(object.True), nil
			}})

		inst.Dict.SetStr("release", &object.BuiltinFunc{Name: "release",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if bounded && count >= initial {
					return nil, object.Errorf(i.valueErr, "BoundedSemaphore released too many times")
				}
				count++
				return object.None, nil
			}})

		inst.Dict.SetStr("__aenter__", &object.BuiltinFunc{Name: "__aenter__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if count > 0 {
					count--
				}
				return makeIter(object.None), nil
			}})

		inst.Dict.SetStr("__aexit__", &object.BuiltinFunc{Name: "__aexit__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				count++
				return makeIter(object.False), nil
			}})

		return inst, nil
	}

	m.Dict.SetStr("Semaphore", &object.BuiltinFunc{Name: "Semaphore",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			value := 1
			if len(a) > 0 {
				if n, ok := toInt64(a[0]); ok {
					value = int(n)
				}
			}
			return makeSemaphore(value, false)
		}})

	m.Dict.SetStr("BoundedSemaphore", &object.BuiltinFunc{Name: "BoundedSemaphore",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			value := 1
			if len(a) > 0 {
				if n, ok := toInt64(a[0]); ok {
					value = int(n)
				}
			}
			return makeSemaphore(value, true)
		}})

	// ── async Queue / LifoQueue / PriorityQueue ───────────────────────────

	makeAsyncQueue := func(qtype string) *object.Instance {
		qCls := &object.Class{Name: qtype, Dict: object.NewDict()}
		inst := &object.Instance{Class: qCls, Dict: object.NewDict()}
		var items []object.Object

		putItem := func(item object.Object) error {
			if qtype == "PriorityQueue" {
				pos := len(items)
				for j, existing := range items {
					less, err := i.lt(item, existing)
					if err != nil {
						return err
					}
					if less {
						pos = j
						break
					}
				}
				items = append(items, nil)
				copy(items[pos+1:], items[pos:])
				items[pos] = item
			} else {
				items = append(items, item)
			}
			return nil
		}

		popItem := func() object.Object {
			if qtype == "LifoQueue" {
				item := items[len(items)-1]
				items = items[:len(items)-1]
				return item
			}
			item := items[0]
			items = items[1:]
			return item
		}

		inst.Dict.SetStr("empty", &object.BuiltinFunc{Name: "empty",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.BoolOf(len(items) == 0), nil
			}})

		inst.Dict.SetStr("full", &object.BuiltinFunc{Name: "full",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.False, nil
			}})

		inst.Dict.SetStr("qsize", &object.BuiltinFunc{Name: "qsize",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.NewInt(int64(len(items))), nil
			}})

		inst.Dict.SetStr("put", &object.BuiltinFunc{Name: "put",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if len(a) > 0 {
					if err := putItem(a[0]); err != nil {
						return nil, err
					}
				}
				return makeIter(object.None), nil
			}})

		inst.Dict.SetStr("get", &object.BuiltinFunc{Name: "get",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if len(items) == 0 {
					return nil, object.Errorf(i.exception, "get() from empty queue")
				}
				return makeIter(popItem()), nil
			}})

		inst.Dict.SetStr("put_nowait", &object.BuiltinFunc{Name: "put_nowait",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				a = mpArgs(a)
				if len(a) > 0 {
					if err := putItem(a[0]); err != nil {
						return nil, err
					}
				}
				return object.None, nil
			}})

		inst.Dict.SetStr("get_nowait", &object.BuiltinFunc{Name: "get_nowait",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				if len(items) == 0 {
					return nil, object.Errorf(i.exception, "get() from empty queue")
				}
				return popItem(), nil
			}})

		inst.Dict.SetStr("join", &object.BuiltinFunc{Name: "join",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return makeIter(object.None), nil
			}})

		inst.Dict.SetStr("task_done", &object.BuiltinFunc{Name: "task_done",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return object.None, nil
			}})

		return inst
	}

	m.Dict.SetStr("Queue", &object.BuiltinFunc{Name: "Queue",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return makeAsyncQueue("Queue"), nil
		}})

	m.Dict.SetStr("LifoQueue", &object.BuiltinFunc{Name: "LifoQueue",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return makeAsyncQueue("LifoQueue"), nil
		}})

	m.Dict.SetStr("PriorityQueue", &object.BuiltinFunc{Name: "PriorityQueue",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return makeAsyncQueue("PriorityQueue"), nil
		}})

	// ── timeout context manager ───────────────────────────────────────────

	makeTimeoutCtx := func() *object.Instance {
		tmCls := &object.Class{Name: "Timeout", Dict: object.NewDict()}
		inst := &object.Instance{Class: tmCls, Dict: object.NewDict()}

		inst.Dict.SetStr("__aenter__", &object.BuiltinFunc{Name: "__aenter__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return makeIter(inst), nil
			}})

		inst.Dict.SetStr("__aexit__", &object.BuiltinFunc{Name: "__aexit__",
			Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
				return makeIter(object.False), nil
			}})

		return inst
	}

	m.Dict.SetStr("timeout", &object.BuiltinFunc{Name: "timeout",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return makeTimeoutCtx(), nil
		}})

	m.Dict.SetStr("timeout_at", &object.BuiltinFunc{Name: "timeout_at",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return makeTimeoutCtx(), nil
		}})

	// ── TaskGroup ─────────────────────────────────────────────────────────

	m.Dict.SetStr("TaskGroup", &object.BuiltinFunc{Name: "TaskGroup",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			tgCls := &object.Class{Name: "TaskGroup", Dict: object.NewDict()}
			inst := &object.Instance{Class: tgCls, Dict: object.NewDict()}

			inst.Dict.SetStr("create_task", &object.BuiltinFunc{Name: "create_task",
				Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
					a = mpArgs(a)
					if len(a) == 0 {
						return nil, object.Errorf(i.typeErr, "create_task() requires a coroutine")
					}
					name := ""
					if kw != nil {
						if v, ok := kw.GetStr("name"); ok && v != object.None {
							if s, ok2 := v.(*object.Str); ok2 {
								name = s.V
							}
						}
					}
					return createTask(a[0], name)
				}})

			inst.Dict.SetStr("__aenter__", &object.BuiltinFunc{Name: "__aenter__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return makeIter(inst), nil
				}})

			inst.Dict.SetStr("__aexit__", &object.BuiltinFunc{Name: "__aexit__",
				Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
					return makeIter(object.False), nil
				}})

			return inst, nil
		}})

	// ── shield ────────────────────────────────────────────────────────────

	m.Dict.SetStr("shield", &object.BuiltinFunc{Name: "shield",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "shield() requires an argument")
			}
			return a[0], nil
		}})

	// ── to_thread ─────────────────────────────────────────────────────────

	m.Dict.SetStr("to_thread", &object.BuiltinFunc{Name: "to_thread",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "to_thread() requires a function")
			}
			var args []object.Object
			if len(a) > 1 {
				args = a[1:]
			}
			result, err := i.callObject(a[0], args, kw)
			if err != nil {
				return nil, err
			}
			return makeIter(result), nil
		}})

	return m
}

// driveCoroutine runs an awaitable (coroutine / generator / iter) to
// completion by repeatedly sending None. Returns the final value (the
// StopIteration .value) or any unhandled exception.
func (i *Interp) driveCoroutine(awaitable object.Object) (object.Object, error) {
	switch x := awaitable.(type) {
	case *object.Generator:
		for {
			_, err := i.resumeGenerator(x, object.None)
			if err != nil {
				if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, i.stopIter) {
					if exc.Args != nil && len(exc.Args.V) > 0 {
						return exc.Args.V[0], nil
					}
					return object.None, nil
				}
				return nil, err
			}
			// Yielded (no one to deliver to — keep driving).
		}
	case *object.Iter:
		for {
			v, ok, err := x.Next()
			if err != nil {
				if exc, ok2 := err.(*object.Exception); ok2 && object.IsSubclass(exc.Class, i.stopIter) {
					if exc.Args != nil && len(exc.Args.V) > 0 {
						return exc.Args.V[0], nil
					}
					return object.None, nil
				}
				return nil, err
			}
			if !ok {
				return object.None, nil
			}
			_ = v
		}
	}
	return nil, object.Errorf(i.typeErr, "cannot drive %s as a coroutine", object.TypeName(awaitable))
}

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
	case "zlib":
		return i.buildZlib(), true
	case "binascii":
		return i.buildBinascii(), true
	case "hmac":
		return i.buildHmac(), true
	case "secrets":
		return i.buildSecrets(), true
	case "uuid":
		return i.buildUUID(), true
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
	case "html":
		return i.buildHtml(), true
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
	case "string.templatelib":
		return i.buildTemplatelib(), true
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
	}
	return nil, false
}

// buildAsyncio constructs a minimal asyncio module: run(coro), sleep(t),
// gather(*coros). The runtime has no real event loop; coroutines are
// driven synchronously to completion. This is enough for single-file
// async scripts that don't depend on concurrent I/O.
func (i *Interp) buildAsyncio() *object.Module {
	m := &object.Module{Name: "asyncio", Dict: object.NewDict()}

	run := &object.BuiltinFunc{Name: "run", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "asyncio.run() missing coroutine")
		}
		return i.driveCoroutine(a[0])
	}}
	m.Dict.SetStr("run", run)

	sleep := &object.BuiltinFunc{Name: "sleep", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		var result object.Object = object.None
		if len(a) > 1 {
			result = a[1]
		}
		done := false
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if done {
				return nil, false, nil
			}
			done = true
			// Terminate immediately: SEND sees the iterator exhausted and
			// turns it into StopIteration(result), which becomes the await
			// expression's value.
			exc := object.NewException(i.stopIter, "")
			exc.Args = &object.Tuple{V: []object.Object{result}}
			return nil, false, exc
		}}, nil
	}}
	m.Dict.SetStr("sleep", sleep)

	gather := &object.BuiltinFunc{Name: "gather", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// No real concurrency — drive each awaitable to completion and
		// collect the results in order.
		results := make([]object.Object, len(a))
		for k, c := range a {
			v, err := i.driveCoroutine(c)
			if err != nil {
				return nil, err
			}
			results[k] = v
		}
		// gather() must itself be awaitable. Wrap as a one-shot iter that
		// immediately produces the results list via StopIteration.
		done := false
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if done {
				return nil, false, nil
			}
			done = true
			exc := object.NewException(i.stopIter, "")
			exc.Args = &object.Tuple{V: []object.Object{&object.List{V: results}}}
			return nil, false, exc
		}}, nil
	}}
	m.Dict.SetStr("gather", gather)

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
				if exc, ok := err.(*object.Exception); ok && object.IsSubclass(exc.Class, i.stopIter) {
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

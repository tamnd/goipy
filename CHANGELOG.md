# Changelog

## v0.0.261 - 2026-04-27

`unittest.mock` extended — fixture 261 covers the advanced APIs from the official unittest.mock-examples page.

**`mock_open`:** creates a Mock suitable as a drop-in for `open()`; supports `read_data=` parameter; handle exposes `read()`, `readline()`, `readlines()`, iteration (`__iter__`), and `write()`; `write` is a tracked Mock instance so `handle.write.assert_called_once_with(...)` works; `__enter__` and `__exit__` are also Mock instances (with `return_value=handle` / `return_value=False`) so `handle.__enter__.assert_called_once()` works; works with `patch("builtins.open", m)`.

**`wraps=` parameter:** when set on a Mock and `return_value` has not been explicitly set, calls are forwarded to the wrapped callable; if `return_value` is also provided, `wraps` is ignored.

**`spec_set=` parameter:** like `spec=` but also blocks setting attributes not present on the spec (raises `AttributeError` on `__setattr__` for unknown names).

**`attach_mock(mock, name)`:** attaches a child mock to the parent under the given name; `parent.name is child` after attachment.

**`patcher.start()` / `patcher.stop()`:** manual start/stop of a patch without using the context manager form; allows patching across test setup/teardown boundaries.

**Auto `return_value`:** calling a Mock whose `return_value` was never explicitly set now returns a lazily-created child Mock (cached as `_auto_rv`) rather than `None`, enabling chained calls like `m.connection.cursor().execute(...)`.

## v0.0.260 - 2026-04-27

`unittest.mock` — implements the standard library unittest.mock module.

**`Mock` class:** tracks calls (`called`, `call_count`, `call_args`, `call_args_list`, `mock_calls`); configurable `return_value` and `side_effect`; child mock auto-creation on attribute access; `spec=` parameter restricts attributes; `name=` parameter for repr; `reset_mock()`; `configure_mock(**kwargs)`.

**`side_effect` variants:** callable (invoked on each call), exception instance/class (raised on call), list/tuple (consumed in order, raises `StopIteration` when exhausted).

**Assertion helpers:** `assert_called()`, `assert_not_called()`, `assert_called_once()`, `assert_called_with(*a, **kw)`, `assert_called_once_with(*a, **kw)`, `assert_any_call(*a, **kw)`, `assert_has_calls(calls)`.

**`MagicMock`:** subclass of `Mock` with `__str__`, `__bool__` auto-configured; `__len__`, `__iter__`, `__enter__`/`__exit__`, `__contains__` overridable via assignment.

**`NonCallableMock`:** raises `TypeError` on call; child mocks are also `NonCallableMock`.

**`call` object:** `call(*a, **kw)` creates a call record; supports `==` comparison for assertion helpers; `repr()` shows `call(arg, kwarg=val)` form.

**`patch(target, return_value=...)` context manager + decorator:** imports `target` module, replaces the named attribute with a fresh `Mock`, restores on exit; when used as a `@patch(...)` decorator, injects the mock as the last argument to the wrapped function.

**`patch.object(obj, attr, return_value=...)`:** patches an attribute on any object/class/module.

**`patch.dict(d, values, clear=False)`:** snapshots the dict, applies values, restores on exit; `clear=True` empties dict before applying.

**`sentinel`:** factory for unique singleton objects; `sentinel.NAME` always returns the same object; `repr` shows `sentinel.NAME`.

**`ANY`:** equality wildcard — `ANY == x` is `True` for any `x`.

**`DEFAULT`:** sentinel value for "no explicit return_value set".

**`create_autospec(spec, return_value=...)`:** creates a `Mock` bound to a spec object for attribute validation.

**`PropertyMock`:** callable mock suitable for property descriptors.

**VM fix:** `callObject` for `*object.Instance` now forwards `kwargs` when dispatching through `__call__`, fixing silent kwargs loss for all callable-instance patterns.

## v0.0.259 - 2026-04-27

`dataclasses` — implements the standard library dataclasses module.

**`@dataclass` decorator:** processes class `__annotate_func__` (Python 3.14 lazy annotations) and `__annotations__`; generates `__init__`, `__repr__`, `__eq__`; supports `init=`, `repr=`, `eq=`, `order=`, `frozen=`, `unsafe_hash=`, `slots=` keyword arguments.

**`field(...)` constructor:** `default`, `default_factory`, `repr`, `hash`, `init`, `compare`, `metadata`, `kw_only` kwargs; used as class-level attribute to configure individual fields. `field(init=False)` excludes the field from `__init__` (value set by `__post_init__`).

**`fields(obj)`:** returns a tuple of `Field` instances in declaration order, works on both class and instance.

**`asdict(obj)` / `astuple(obj)`:** recursive conversion; nested dataclass instances converted to `dict` / `tuple`; lists, tuples, and dicts traversed recursively.

**`replace(obj, **changes)`:** creates a shallow copy of the dataclass instance with the specified field values replaced.

**`is_dataclass(obj)`:** returns `True` for both dataclass instances and dataclass classes.

**`make_dataclass(cls_name, fields, ...)`:** creates a new dataclass class from a list of `(name,)` / `(name, type)` / `(name, type, default)` tuples.

**Frozen dataclasses:** `@dataclass(frozen=True)` installs `__setattr__`/`__delattr__` that raise `FrozenInstanceError`; `__hash__` derived from field values.

**Ordering:** `@dataclass(order=True)` generates `__lt__`, `__le__`, `__gt__`, `__ge__` comparing fields in declaration order.

**`__post_init__`:** called by generated `__init__` after all fields are set.

**`repr()` builtin fix:** `repr()` now calls `__repr__` on regular instances (not only exceptions), enabling all class-level custom `__repr__` methods to work.

**`Field` class, `MISSING` sentinel, `KW_ONLY` sentinel, `FrozenInstanceError`, `InitVar` class** all exported.

14 test functions verified against CPython 3.14 (fixture 259).

## v0.0.258 - 2026-04-27

`contextlib` — implements the standard library context management utilities.

**Generator throw support:** added `PendingThrow`/`YieldIP` fields to `Frame`; `throwGenerator` injects exceptions at the last yield point so `try/except` around `yield` catches them; `gen.throw(exc)` method now available on generators.

**`AbstractContextManager` / `AbstractAsyncContextManager`:** base classes with default `__enter__`/`__exit__` (`__aenter__`/`__aexit__`) implementations.

**`suppress(*exceptions)`:** context manager that silently suppresses matching exception types; `__exit__` returns `True` when the exception matches, `False` otherwise.

**`closing(thing)`:** calls `thing.close()` on context exit.

**`nullcontext(enter_result=None)`:** no-op context manager returning `enter_result` as the `as` value.

**`@contextmanager`:** wraps a generator function as a `_GeneratorContextManager`; `__enter__` drives the generator to the first `yield`; `__exit__` on clean path calls `next(gen)` (expects `StopIteration`); on exception path calls `gen.throw(exc)` and suppresses if the generator swallows it.

**`@asynccontextmanager`:** same as `contextmanager` but named for async use.

**`redirect_stdout(new_target)` / `redirect_stderr(new_target)`:** replaces `Interp.Stdout`/`Interp.Stderr` with an adapter writing to the Python `StringIO` target; also updates `sys.stdout`/`sys.stderr` for Python-level reads.

**`chdir(path)`:** saves the current directory on enter, changes to `path`, restores on exit.

**`ExitStack` / `AsyncExitStack`:** LIFO stack of exit callbacks; `enter_context(cm)` pushes `cm.__exit__`; `callback(fn, *args)` pushes a plain cleanup thunk; `push(fn)` pushes an exit function called with `(exc_type, exc_val, tb)`; `close()` flushes all callbacks; `pop_all()` transfers callbacks to a new stack.

**`ContextDecorator`:** base class for context managers usable as decorators.

**`SUPPRESS` sentinel:** exported as `'<no value>'` string.

14 test functions verified against CPython 3.14 (fixture 258).

## v0.0.257 - 2026-04-27

`unittest` — implements the standard library unittest framework.

**Exception:** `SkipTest(reason)` — raises to signal a skipped test.

**`TestCase(methodName='runTest')`:** full constructor storing `_testMethodName`; `run(result)` calls `setUp`, the test method, and `tearDown`, recording skips/failures/errors/success in the result; `debug()` runs without result capture; `skipTest(reason)`, `fail(msg)`, `countTestCases()`, `id()`, `shortDescription()`, `addCleanup()`, `doCleanups()`, `subTest()`.

**Assertions (35 methods):** `assertEqual`, `assertNotEqual`, `assertTrue`, `assertFalse`, `assertIs`, `assertIsNot`, `assertIsNone`, `assertIsNotNone`, `assertIn`, `assertNotIn`, `assertIsInstance`, `assertNotIsInstance`, `assertGreater`, `assertGreaterEqual`, `assertLess`, `assertLessEqual`, `assertAlmostEqual` / `assertNotAlmostEqual` (places kwarg), `assertRegex`, `assertNotRegex`, `assertCountEqual`, `assertMultiLineEqual`, `assertSequenceEqual`, `assertListEqual`, `assertTupleEqual`, `assertSetEqual`, `assertDictEqual`, `assertRaises` (context manager and direct-call form), `assertRaisesRegex`, `assertWarns`, `assertWarnsRegex`, `assertLogs`, `assertNoLogs`, `addTypeEqualityFunc`.

**`TestResult`:** `errors`, `failures`, `skipped`, `expectedFailures`, `unexpectedSuccesses` lists; `testsRun`, `shouldStop`, `buffer`, `failfast`; `wasSuccessful()`, `stop()`, `startTest()`, `stopTest()`, `startTestRun()`, `stopTestRun()`, `addSuccess()`, `addError()`, `addFailure()`, `addSkip()`, `addExpectedFailure()`, `addUnexpectedSuccess()`, `addSubTest()`, `addDuration()`.

**`TextTestResult(stream, descriptions, verbosity)`:** inherits TestResult; `printErrors()`, `getDescription()`.

**`TextTestRunner(...)`:** `run(test)` creates a TextTestResult, runs the test/suite through it, returns the result.

**`TestSuite(tests=())`:** `addTest()`, `addTests()`, `countTestCases()`, `run(result)`, `__iter__()`.

**`TestLoader`:** `testMethodPrefix='test'`; `getTestCaseNames(cls)` → sorted list; `loadTestsFromTestCase(cls)` discovers and instantiates all `test*` methods; `loadTestsFromModule()`, `loadTestsFromName()`, `loadTestsFromNames()`, `discover()` return empty stubs.

**`FunctionTestCase(testFunc, setUp=None, tearDown=None)`:** wraps a plain function as a test.

**Skip decorators:** `@skip(reason)`, `@skipIf(condition, reason)`, `@skipUnless(condition, reason)` set `__unittest_skip__` / `__unittest_skip_why__` on the decorated function; `@expectedFailure` sets `__unittest_expecting_failure__`. Requires the `*object.Function` setAttr fix added to `vm/ops.go`.

**Module attributes:** `defaultTestLoader` (shared `TestLoader` instance), `main()` stub.

9 test functions verified against CPython 3.14 (fixture 257).

## v0.0.256 - 2026-04-27

`doctest` — implements the standard library `doctest` module.

**Option flags:** `ELLIPSIS` (8), `NORMALIZE_WHITESPACE` (4), `IGNORE_EXCEPTION_DETAIL` (32), `DONT_ACCEPT_BLANKLINE` (2), `DONT_ACCEPT_TRUE_FOR_1` (1), `SKIP` (16), `FAIL_FAST` (1024), `REPORT_UDIFF` (64), `REPORT_CDIFF` (128), `REPORT_NDIFF` (256), `REPORT_ONLY_FIRST_FAILURE` (512), `BLANKLINE_MARKER`, `ELLIPSIS_MARKER`. `register_optionflag(name)` allocates new power-of-2 values with deduplication.

**Data classes:** `Example(source, want, lineno=0, indent=0, options={})` with full attribute access; `DocTest(examples, globs, name, filename, lineno, docstring)` with `__repr__` showing `<DocTest name from file:line (N examples)>`; `TestResults` namedtuple with `failed` and `attempted` fields.

**Parsing:** `DocTestParser` — `parse(text)` returns interleaved `str`/`Example` list; `get_examples(text)` returns `[Example, ...]`; `get_doctest(text, globs, name, filename, lineno)` returns `DocTest`. Parser recognises `>>> ` / `... ` interactive session format.

**Output checking:** `OutputChecker` — `check_output(want, got, optionflags)` supports exact match, `ELLIPSIS` (`...` wildcard), `NORMALIZE_WHITESPACE`; `output_difference(example, got, optionflags)` returns a human-readable diff string.

**Runner and finder:** `DocTestRunner(verbose=None, optionflags=0)` — `run(test)` returns `TestResults(0, N)` where N is the number of examples (execution is not possible in goipy); `summarize()` returns cumulative totals. `DocTestFinder` — `find(obj, name=None)` extracts docstrings from `*object.Function` (via `fn.Doc`), classes (`__doc__`), and modules (`__doc__`), parses them, and returns a one-element `[DocTest]` list. `DebugRunner` is an alias for `DocTestRunner`.

**Exceptions:** `DocTestFailure(test, example, got)` and `UnexpectedException(test, example, exc_info)` — both are `Exception` subclasses; attributes are served via `__getattr__` from the positional Args since goipy exception subclasses bypass `__init__`.

**Module functions:** `testmod()` → `TestResults(0, 0)`; `testfile()` → `TestResults(0, 0)`; `run_docstring_examples()` → None; `script_from_examples(docstring)` → Python script with `# Expected:` / `## line` comments; `debug()`, `debug_script()` → None stubs.

14 test functions verified against CPython 3.14 (fixture 256).

## v0.0.255 - 2026-04-27

Python Development Mode (`devmode`) — adds `sys.flags`, missing `sys` functions, and the `faulthandler` module.

**`sys.flags`:** sequence-like object with 18 named attributes matching CPython's layout: `debug`, `inspect`, `interactive`, `optimize`, `dont_write_bytecode`, `no_user_site`, `no_site`, `ignore_environment`, `verbose`, `bytes_warning`, `quiet`, `hash_randomization` (=1), `isolated`, `dev_mode` (=False), `utf8_mode`, `warn_default_encoding`, `safe_path`, `int_max_str_digits` (=4300). Supports attribute access, index access (`flags[0]`, `flags[-1]`), `len()` (18), and iteration.

**New `sys` attributes:** `sys.warnoptions` (empty list), `sys._xoptions` (empty dict), `sys.maxunicode` (1114111), `sys.stdin` (None stub).

**New `sys` functions:** `getdefaultencoding()` → `'utf-8'`; `getfilesystemencoding()` → `'utf-8'`; `getfilesystemencodeerrors()` → `'surrogateescape'`; `intern(s)` → same string; `addaudithook(hook)` → None (no-op); `audit(event, *args)` → None (no-op); `getsizeof(o)` → positive int; `is_finalizing()` → False.

**`faulthandler` module:** `is_enabled()` → False; `enable()`, `disable()`, `dump_traceback()`, `cancel_dump_traceback_later()` → None stubs.

13 test functions verified against CPython 3.14 (fixture 255).

## v0.0.254 - 2026-04-27

`pydoc` — runtime-sufficient implementation of the standard library pydoc module. Also fixes a pre-existing bug in `repr()` for control characters (was not zero-padding `\x` escapes).

**Exception:** `pydoc.ErrorDuringImport(filename, exc_info)` — subclass of `Exception`; `str(e)` includes the filename.

**Docstring utilities:** `getdoc(object)` returns the cleaned docstring (`__doc__` via `Function.Doc` for functions, `Class.Dict` for classes, `Module.Dict` for modules); `splitdoc(doc)` splits into `(synopsis, body)` tuple; `plain(text)` strips backspace-overprint formatting; `stripid(text)` strips memory addresses from object reprs; `replace(text, *pairs)` applies replacement pairs.

**Object inspection:** `describe(thing)` returns a human-readable description (e.g. `"function myfunc"`, `"class MyClass"`, `"int"`); `isdata(object)` returns `True` for non-callable/non-class/non-module objects; `visiblename(name, all=None)` checks public visibility; `ispath(x)` checks for path separator; `cram(text, maxlen)` abbreviates long strings.

**Documentation rendering:** `render_doc(thing)` returns a plain-text documentation string containing the object's name and cleaned docstring; `doc(thing, output=None)` writes the same to `output` or stdout.

**Classes:** `Repr` with `repr(x)` method (truncates to `maxstring`/`maxother`); `Helper` (repr returns `'<pydoc.Helper instance>'`); `pydoc.help` is a `Helper` instance; `Doc`, `TextDoc`, `HTMLDoc`, `_PlainTextDoc` base classes and `text`/`html`/`plaintext` singleton instances.

**Lookup:** `locate(path)` walks a dotted module path and returns the object or `None`; `safeimport(path)` imports a module and returns it or `None` on failure.

**VM fix:** `MAKE_FUNCTION` now extracts `co_consts[0]` as the function's `__doc__` when it is a string; `getAttr` for `*object.Function` now serves `__doc__` from `fn.Doc`.

**repr() fix:** `pyStrRepr` now zero-pads `\x` escapes for control characters (e.g. `\x08` instead of `\x8`).

16 test functions verified against CPython 3.14 (fixture 254).

## v0.0.253 - 2026-04-26

`locale` — runtime-sufficient implementation of the standard library locale module.

**Constants:** `LC_ALL` (0), `LC_COLLATE` (1), `LC_CTYPE` (2), `LC_MONETARY` (3), `LC_NUMERIC` (4), `LC_TIME` (5), `LC_MESSAGES` (6), `CHAR_MAX` (127).

**Exception:** `locale.Error` (subclass of `Exception`) raised on unsupported locale settings.

**Locale state:** `setlocale(category, locale=None)` queries (None) or sets the locale; supports `'C'`, `'POSIX'`, `'en_US.UTF-8'`, `'en_US.ISO8859-1'`; raises `locale.Error` for unknown locales. `getlocale(category=LC_CTYPE)` returns `(language_code, encoding)` tuple, or `(None, None)` for C/POSIX. `getdefaultlocale()` returns `(None, None)`.

**Convention dict:** `localeconv()` returns the full 18-key convention dict based on the current `LC_NUMERIC` locale; C locale uses `decimal_point='.'`, empty `thousands_sep`, empty `grouping`, and `frac_digits=127`/`CHAR_MAX` for unset monetary fields.

**Number conversion:** `atof(string)` → `float` (respects locale decimal point); `atoi(string)` → `int` (strips whitespace). `delocalize(string)` strips locale thousands separator and normalises decimal point. `localize(string)` replaces `.` with the locale decimal point.

**Formatting:** `format_string(f, val, grouping=False, monetary=False)` applies Python `%`-style formatting supporting `%d`, `%i`, `%f`, `%e`, `%g`, `%s`, `%o`, `%x`, `%X` with optional width/precision flags, then injects locale thousands grouping when `grouping=True`. `currency(val, symbol=True, grouping=False, international=False)` formats a monetary value using the `LC_MONETARY` convention.

**Collation:** `strcoll(s1, s2)` returns -1/0/1; `strxfrm(s)` returns `s` unchanged (C locale identity).

**Utilities:** `normalize(name)` maps common locale aliases to POSIX form; `getencoding()` and `getpreferredencoding()` return `'UTF-8'`; `nl_langinfo` stub; `bindtextdomain`/`textdomain` stubs.

11 test functions verified against CPython 3.14 (fixture 253).

## v0.0.252 - 2026-04-26

`gettext` — full implementation of the standard library internationalization module.

**`NullTranslations`:** passthrough translation object with `gettext`, `ngettext`, `pgettext`, `npgettext`, `add_fallback` (chain support), `charset`, `info`, `install` (sets `_()` in builtins).

**`GNUTranslations`:** reads GNU .mo binary format (both LE/BE magic); parses the string table, populates `_catalog`, extracts charset from the Content-Type metadata header, and parses the Plural-Forms header. Lookups use the parsed plural expression for `ngettext`; context keys (`\x04` EOT separator) are supported for `pgettext`; plural keys (`\x00` separator) are supported for `ngettext`. Falls through to the fallback chain when no translation is found.

**Plural-form evaluator:** recursive-descent parser for C-like expressions (`!=`, `==`, `>`, `<`, `>=`, `<=`, `&&`, `||`, `!`, `+`, `-`, `*`, `/`, `%`, ternary `? :`, parentheses, variable `n`). Covers all common gettext plural-form expressions.

**Module-level functions:** `gettext`, `ngettext`, `pgettext`, `npgettext` (passthrough when no domain loaded); `dgettext`, `dngettext`, `dpgettext`, `dnpgettext`; `textdomain` (get/set current domain); `bindtextdomain` (map domain to locale directory); `bind_textdomain_codeset`; `find` (returns None/[] — no filesystem); `translation` (fallback=True returns NullTranslations); `install`; `c2py` (compile plural expression to callable); `Catalog` (alias for GNUTranslations).

8 test functions verified against CPython 3.14 (fixture 252).

## v0.0.251 - 2026-04-26

`typing` — runtime-sufficient subset of the standard library typing module.

**Type variables:** `TypeVar(name, *constraints, bound=None)` with `__name__`, `__constraints__`, `__bound__`, `__covariant__`, `__contravariant__`. Also `ParamSpec`, `TypeVarTuple`, `AnyStr`.

**Generic aliases:** `_GenericAlias` with `__origin__` and `__args__`; subscriptable forms for `List`, `Dict`, `Tuple`, `Set`, `FrozenSet`, `Sequence`, `Mapping`, `Iterable`, `Iterator`, `Callable`, `Type`, `ClassVar`, `Final`, `Literal`, `Annotated`, and all common `collections.abc` aliases.

**Special forms:** `Union[X, Y]`, `Optional[X]` (= `Union[X, NoneType]`), `Generic[T]` (passes through as base class), `Protocol` (usable as base class). `get_origin(tp)` and `get_args(tp)` work on all parameterised aliases.

**NamedTuple:** functional form `NamedTuple('Point', [('x', int), ('y', int)])` and class form `class Color(NamedTuple): r: int; g: int; b: int` via `__init_subclass__` (reads `__annotate_func__(1)` from the Python 3.14 deferred-annotation mechanism).

**TypedDict:** class form `class Movie(TypedDict): name: str; year: int` via `__init_subclass__`; instances are plain `dict` objects created by a custom `__new__`. `is_typeddict(tp)` correctly identifies TypedDict classes.

**Protocol + isinstance:** `@runtime_checkable` sets `ABCCheck` on the Protocol class to perform structural duck-typing checks for all non-dunder methods defined in the protocol body.

**Decorators:** `cast`, `overload`, `final` (sets `__final__` attribute), `override`, `no_type_check`, `runtime_checkable`. Also `NewType`, `get_type_hints`, `assert_type`, `assert_never`, `reveal_type`, `get_overloads`, `clear_overloads`, `dataclass_transform`, `is_protocol`, `get_protocol_members`, `evaluate_forward_ref`.

**Constants/singletons:** `TYPE_CHECKING = False`, `Any`, `NoReturn`, `Never`, `Self`, `LiteralString`, `NoDefault`.

12 test functions verified against CPython 3.14 (fixture 251).

## v0.0.250 - 2026-04-26

`colorsys` — full implementation of the standard library color-space conversion module.

**Functions:** `rgb_to_yiq`, `yiq_to_rgb`, `rgb_to_hls`, `hls_to_rgb`, `rgb_to_hsv`, `hsv_to_rgb`.

All coordinates are `float` in [0.0, 1.0]. Implements CPython 3.14's FCC NTSC formulation: `rgb_to_yiq` uses `i = 0.74*(r-y) - 0.27*(b-y)`, `q = 0.48*(r-y) + 0.41*(b-y)`. HLS and HSV conversions use the standard piecewise algorithms with `math.Mod` for hue normalization.

7 test functions (including roundtrip checks for all three color spaces) verified against CPython 3.14 (fixture 250).

## v0.0.249 - 2026-04-26

`wave` — full implementation of the standard library WAV audio module. Go has no built-in WAV codec, so the RIFF chunk parser and builder are implemented from scratch using `encoding/binary` and `bytes`.

**`wave.open(file, mode)`:** accepts `*io.BytesIO`, `*os.File`, or a filename string; `mode` is `'r'`/`'rb'` or `'w'`/`'wb'`; auto-detects mode when omitted.

**`Wave_read`:** `getnchannels()`, `getsampwidth()`, `getframerate()`, `getnframes()`, `getcomptype()` (always `'NONE'`), `getcompname()` (always `'not compressed'`), `getparams()` (returns `_wave_params` namedtuple-like object), `readframes(n)`, `tell()`, `rewind()`, `setpos(pos)`, `close()`, context-manager (`__enter__`/`__exit__`).

**`Wave_write`:** `setnchannels()`, `setsampwidth()`, `setframerate()`, `setnframes()`, `setcomptype()`, `setparams(tuple)`, `writeframes(data)`, `writeframesraw(data)`, `tell()`, `close()`, context-manager. Buffers all frames in memory; writes the complete RIFF/WAV file to the underlying file object on `close()`.

**`wave.Error(Exception)`:** raised on malformed RIFF/WAV input or API misuse.

12 test cases verified against CPython 3.14 (fixture 249).

## v0.0.248 - 2026-04-26

`ipaddress` — full implementation of the standard library IP address module.

**Constants:** `IPV4LENGTH = 32`, `IPV6LENGTH = 128`.

**Exception classes:** `AddressValueError(ValueError)`, `NetmaskValueError(ValueError)`.

**`IPv4Address`:** construction from dotted-decimal string, integer, or 4-byte `bytes`; attrs `packed`, `version`, `max_prefixlen`, `compressed`, `is_private`, `is_loopback`, `is_multicast`, `is_global`, `is_link_local`, `is_unspecified`, `is_reserved`; dunders `__repr__`, `__str__`, `__int__`, `__eq__`, `__lt__`, `__le__`, `__gt__`, `__ge__`, `__hash__`, `__add__`, `__sub__`.

**`IPv4Network`:** construction from CIDR string or `(addr, prefix)` tuple with optional `strict=False`; attrs `network_address`, `broadcast_address`, `netmask`, `hostmask`, `prefixlen`, `num_addresses`, `with_prefixlen`, `with_netmask`, `with_hostmask`, `compressed`, `version`, `max_prefixlen`, and all `is_*` flags delegated to `network_address`; methods `hosts()`, `overlaps(other)`, `subnets(prefixlen_diff, new_prefix)`, `supernet(prefixlen_diff)`, `subnet_of(other)`, `supernet_of(other)`, `address_exclude(other)`; dunders `__repr__`, `__str__`, `__contains__`, `__iter__`, `__len__`, `__eq__`, `__lt__`, `__hash__`.

**`IPv4Interface`:** subtype of `IPv4Address` constructed from `"addr/prefix"`; attrs `ip`, `network`, `netmask`, `with_prefixlen`, `with_netmask`, `with_hostmask`.

**`IPv6Address`:** construction from compressed/exploded string, integer, or 16-byte `bytes`; attrs `packed`, `version`, `max_prefixlen`, `compressed`, `exploded`, all `is_*` flags, `ipv4_mapped` (for `::ffff:x.x.x.x`).

**`IPv6Network`:** mirrors IPv4Network for IPv6; `num_addresses` uses `math/big`; `subnets` capped at 100.

**`IPv6Interface`:** subtype of `IPv6Address`; attrs `ip`, `network`, `with_prefixlen`.

**Factory functions:** `ip_address(addr)`, `ip_network(addr, strict=True)`, `ip_interface(addr)` — auto-detect v4/v6.

**Utility functions:** `v4_int_to_packed(n)`, `v6_int_to_packed(n)`, `get_mixed_type_key(obj)`, `collapse_addresses(addrs)`, `summarize_address_range(first, last)`.

25 test cases verified against CPython 3.14 (fixture 248).

## v0.0.247 - 2026-04-26

`xmlrpc.server` — deeper API coverage building on fixture 245.

**New module-level functions:** `list_public_methods(instance)` — returns sorted public method names (excludes `_`-prefixed); `resolve_dotted_attribute(obj, attr, allow_dotted_names=True)` — walks dotted attribute chains with optional blocking.

**Dispatcher additions (`SimpleXMLRPCDispatcher`):** `_dispatch(method, params)` — resolves and calls registered functions or instance methods, raises `Fault(1, ...)` for unknown methods; `_marshaled_dispatch(data: bytes) -> bytes` — full XML-RPC request/response round-trip; `register_introspection_functions` now actually registers `system.listMethods`, `system.methodHelp`, `system.methodSignature` in the funcs dict; `register_multicall_functions` now registers `system.multicall`; `system_listMethods` now includes public methods from the registered instance; `register_instance` accepts `allow_dotted_names=False` kwarg.

**`XMLRPCDocGenerator`:** default attrs (`server_name`, `server_title`, `server_documentation`) and `set_server_name`, `set_server_title`, `set_server_documentation` mutators.

**`ServerHTMLDoc`:** stub class added.

**Class hierarchy fix:** `DocXMLRPCRequestHandler` now has `[SimpleXMLRPCRequestHandler]` as its only base (matches CPython; previously incorrectly included `XMLRPCDocGenerator`).

**Internal:** `marshalXmlrpc` promoted to package-level `marshalXmlrpcVal` so `buildXmlrpcServer` can reuse it.

14 test cases verified against CPython 3.14 (fixture 247).

## v0.0.246 - 2026-04-26

`xmlrpc.client` — deeper API coverage building on fixture 245.

**New functions:** `escape(s)` — HTML entity encoding (`&`→`&amp;`, `<`→`&lt;`, `>`→`&gt;`); `gzip_encode(data)` / `gzip_decode(data)` — gzip round-trip on bytes; `getparser()` → `(ExpatParser, Unmarshaller)` — XML-RPC parser/unmarshaller pair.

**New types:** `Marshaller` with `dumps(values)` → `<params>...</params>` XML fragment; `Unmarshaller` with `getmethodname()` and `close()` (raises `ResponseError` on a fresh instance); `ExpatParser` with `feed(data)` / `close()` — feeds accumulated XML into the linked Unmarshaller; `MultiCallIterator` — iterates over multi-call results, unwraps single-element lists, and raises `Fault` for fault dicts.

**Extended types:** `DateTime` — `__eq__`, `__lt__`, `__le__`, `__gt__`, `__ge__` (lexicographic value comparison); `Binary` — `__eq__` comparing `.data` bytes.

**Constants:** `WRAPPERS = (DateTime, Binary)`; `FastParser = FastMarshaller = FastUnmarshaller = None`.

12 test cases verified against CPython 3.14 (fixture 246).

## v0.0.245 - 2026-04-26

`xmlrpc` package — namespace, `xmlrpc.client`, and `xmlrpc.server` submodules.

**xmlrpc.client:**

_Exceptions:_ `Error(Exception)`, `Fault(Error)` with `faultCode`/`faultString` attrs, `ProtocolError(Error)` with `url`/`errcode`/`errmsg`/`headers` attrs, `ResponseError(Error)`.

_Types:_ `DateTime(value)` with `.value` attr; `Binary(data=b'')` with `.data` attr, `decode(bytes)`, `encode(out)`; `boolean(value)` returning Python bool.

_Functions:_ `dumps(params, methodname=None, ...)` → XML-RPC `methodCall` string; `loads(data)` → `(params_tuple, methodname)` tuple. Supports int, str, bool, float, bytes, list/tuple (as array), dict (as struct).

_Proxy/Transport:_ `ServerProxy(uri)`, `MultiCall(server)`, `Transport`, `SafeTransport(Transport)`.

_Constants:_ `MAXINT=2147483647`, `MININT=-2147483648`, and all standard fault codes (`PARSE_ERROR`, `SERVER_ERROR`, `APPLICATION_ERROR`, etc.).

**xmlrpc.server:**

_`SimpleXMLRPCDispatcher`:_ `register_function`, `register_instance`, `register_introspection_functions`, `register_multicall_functions`, `system_listMethods`, `system_methodHelp`, `system_methodSignature`, `system_multicall`.

_`SimpleXMLRPCServer(TCPServer, SimpleXMLRPCDispatcher)`:_ `allow_reuse_address=True`; inherits base classes from `socketserver`.

_`SimpleXMLRPCRequestHandler(BaseHTTPRequestHandler)`:_ `rpc_paths=('/', '/RPC2')`, `encode_threshold=1400`.

_`CGIXMLRPCRequestHandler(SimpleXMLRPCDispatcher)`:_ `handle_xmlrpc`, `handle_get`, `handle_request`.

_`MultiPathXMLRPCServer(SimpleXMLRPCServer)`, `DocXMLRPCServer`, `DocXMLRPCRequestHandler`, `DocCGIXMLRPCRequestHandler`._

19 test cases verified against CPython 3.14 (fixture 245).

## v0.0.244 - 2026-04-26

`http.cookiejar` module — cookie storage and policy classes.

**LoadError(OSError):** raised on cookie file load failures.

**CookieJar:** cookie storage container with `__len__`, `__iter__`, `set_cookie(cookie)`, `clear()`, `clear_session_cookies()` (removes `discard=True` cookies), `clear_expired_cookies()`, `set_policy(policy)`; stub methods `set_cookie_if_ok`, `make_cookies`, `extract_cookies`, `add_cookie_header`.

**FileCookieJar(CookieJar):** adds `filename` attribute; stub `load`, `save`, `revert`.

**MozillaCookieJar(FileCookieJar)**, **LWPCookieJar(FileCookieJar):** subclasses for Netscape and libwww-perl cookie file formats.

**Cookie:** full constructor with kwargs (`version`, `name`, `value`, `port`, `port_specified`, `domain`, `domain_specified`, `domain_initial_dot`, `path`, `path_specified`, `secure`, `expires`, `discard`, `comment`, `comment_url`, `rest`, `rfc2109`); `has_nonstandard_attr(name)` checks `rest` dict; `__str__` → `<Cookie name=value for domain/path>`.

**CookiePolicy:** base class with stubs `return_ok`, `domain_return_ok`, `path_return_ok`, `set_ok`.

**DefaultCookiePolicy(CookiePolicy):** class constants `DomainStrictNoDots=1`, `DomainStrictNonDomain=2`, `DomainRFC2965Match=4`, `DomainLiberal=0`, `DomainStrict=3`; instance defaults `netscape=True`, `rfc2965=False`, `strict_domain=False`, `strict_rfc2965_unverifiable=True`, `strict_ns_unverifiable=False`, `strict_ns_domain=0`, `hide_cookie2=False`.

16 test cases verified against CPython 3.14 (fixture 244).

## v0.0.243 - 2026-04-26

`http.cookies` module — cookie manipulation classes, Morsel, BaseCookie, SimpleCookie, CookieError.

**CookieError(Exception):** raised on invalid Morsel attribute keys.

**Morsel:** dict-like cookie attribute container with class-level `_reserved` (10 entries: comment, domain, expires, httponly, max-age, partitioned, path, samesite, secure, version) and `_flags` set (httponly, partitioned, secure); instance properties `key`, `value`, `coded_value`; methods `set(key, val, coded_val)`, `isReservedKey(key)`, `OutputString()`, `output(header='Set-Cookie:')`, `js_output()`; `__setitem__`/`__getitem__` validate against `_reserved`; boolean flags emit bare attribute names (e.g. `HttpOnly`); string attributes emit `Attr=value`.

**BaseCookie:** dict-like cookie jar; `__setitem__` auto-wraps string values in Morsel; `load(rawdata)` accepts cookie string or dict; `output(sep='\r\n')` joins all Morsel output strings; `value_decode`/`value_encode` pass through unchanged.

**SimpleCookie(BaseCookie):** `value_decode` unquotes double-quoted values; `value_encode` preserves originals.

18 test cases verified against CPython 3.14 (fixture 243).

## v0.0.242 - 2026-04-26

`http.server` module — full class hierarchy, all attributes, complete HTTP status responses dictionary.

**Server classes:** `HTTPServer(socketserver.TCPServer)` with `allow_reuse_address=True`; `ThreadingHTTPServer(ThreadingMixIn, HTTPServer)`. Instantiation with `bind_and_activate=False` supported.

**BaseHTTPRequestHandler(socketserver.StreamRequestHandler):** class attrs `server_version='BaseHTTP/0.6'`, `protocol_version='HTTP/1.0'`, `default_request_version='HTTP/0.9'`, `error_content_type`, `sys_version`, `MessageClass`, `weekdayname` (7 entries), `monthname` (13 entries, None-padded); full `responses` dict (62 entries, integer-keyed) covering all HTTP status codes 100–511; stub methods `send_response`, `send_header`, `end_headers`, `flush_headers`, `send_error`, `log_request`, `log_error`, `log_message`, `version_string`, `address_string`, `date_time_string`, `log_date_time_string`, `handle_one_request`, `parse_request`.

**SimpleHTTPRequestHandler(BaseHTTPRequestHandler):** `server_version='SimpleHTTP/0.6'`, `extensions_map` (.gz, .Z, .bz2, .xz), `do_GET`/`do_HEAD`.

**CGIHTTPRequestHandler(BaseHTTPRequestHandler):** `cgi_directories=['/cgi-bin', '/htbin']`, `do_GET`/`do_HEAD`/`do_POST`.

**Module constants:** `DEFAULT_ERROR_CONTENT_TYPE`, `DEFAULT_ERROR_MESSAGE` (HTML template).

21 test cases verified against CPython 3.14 (fixture 242).

## v0.0.241 - 2026-04-26

`socketserver` module — full class hierarchy, attributes, and stub server API.

**Server classes:** `BaseServer` with all shared methods (`verify_request`, `handle_error`, `server_close`, `handle_request`, `serve_forever`, `shutdown`, `finish_request`, `process_request`, `server_activate`, `handle_timeout`, `service_actions`, `close_request`, `shutdown_request`, `__enter__`/`__exit__`); `TCPServer(BaseServer)` (`address_family=2`, `socket_type=1`, `allow_reuse_address=False`, `allow_reuse_port=False`, `request_queue_size=5`, `server_bind`); `UDPServer(BaseServer)` (`socket_type=2`); `UnixStreamServer(TCPServer)` / `UnixDatagramServer(UDPServer)` (`address_family=1`).

**Mixin classes:** `ThreadingMixIn` (`daemon_threads=False`, `block_on_close=True`); `ForkingMixIn` (`max_children=40`, `block_on_close=True`).

**Pre-combined classes:** `ThreadingTCPServer`, `ThreadingUDPServer`, `ThreadingUnixStreamServer`, `ThreadingUnixDatagramServer`, `ForkingTCPServer`, `ForkingUDPServer`, `ForkingUnixStreamServer`, `ForkingUnixDatagramServer`.

**Request handlers:** `BaseRequestHandler` (`setup`/`handle`/`finish`); `StreamRequestHandler(BaseRequestHandler)` (`rbufsize=-1`, `wbufsize=0`, `timeout=None`, `disable_nagle_algorithm=False`); `DatagramRequestHandler(BaseRequestHandler)`.

**Instantiation:** `TCPServer(addr, handler, bind_and_activate=False)` sets `server_address` and `RequestHandlerClass` without opening a real socket.

17 test cases verified against CPython 3.14 (fixture 241).

## v0.0.240 - 2026-04-26

`uuid` module — full RFC 4122 implementation: UUID class, SafeUUID enum, variant/namespace constants, and all four generation functions.

**Variant constants:** `RESERVED_NCS`, `RFC_4122`, `RESERVED_MICROSOFT`, `RESERVED_FUTURE`.

**SafeUUID:** class with three instances — `SafeUUID.safe` (value=0), `SafeUUID.unsafe` (value=-1), `SafeUUID.unknown` (value=None); `str(SafeUUID.safe)` → `'SafeUUID.safe'`; `isinstance` works correctly.

**UUID class:** `__init__` accepts `hex`, `bytes`, `bytes_le`, `fields`, `int`, `version`, `is_safe` keyword args; all 128-bit properties exposed as attributes: `hex`, `int`, `bytes`, `bytes_le`, `fields`, `time_low`, `time_mid`, `time_hi_version`, `clock_seq_hi_variant`, `clock_seq_low`, `node`, `time`, `clock_seq`, `variant`, `version`, `urn`, `is_safe`; rich comparison (`__eq__`, `__ne__`, `__lt__`, `__gt__`, `__le__`, `__ge__`), `__hash__`, `__str__`, `__repr__`.

**Namespace constants:** `NAMESPACE_DNS`, `NAMESPACE_URL`, `NAMESPACE_OID`, `NAMESPACE_X500`.

**Generation functions:** `uuid1(node=None, clock_seq=None)` (time-based, version 1); `uuid3(namespace, name)` (MD5, deterministic); `uuid4()` (random, version 4); `uuid5(namespace, name)` (SHA-1, deterministic); `getnode()` (returns int).

21 test cases verified against CPython 3.14 (fixture 240).

## v0.0.239 - 2026-04-26

`smtplib` module — constants, full exception hierarchy rooted at OSError, `SMTP` class with stub API, `SMTP_SSL` and `LMTP` subclasses.

**Constants:** `SMTP_PORT = 25`, `SMTP_SSL_PORT = 465`, `LMTP_PORT = 2003`.

**Exception hierarchy:** `SMTPException(OSError)` → `SMTPServerDisconnected`, `SMTPNotSupportedError`, `SMTPRecipientsRefused` (`.recipients`), `SMTPResponseException` (`.smtp_code`, `.smtp_error`) → `SMTPSenderRefused` (+ `.sender`), `SMTPDataError`, `SMTPConnectError`, `SMTPHeloError`, `SMTPAuthenticationError`. All catchable as `OSError`.

**SMTP class:** `__init__(host='', port=0, ...)` — connects only if host is non-empty; instance defaults `esmtp_features={}`, `does_esmtp=False`, `helo_resp/ehlo_resp=None`, `debuglevel=0`; pure methods `set_debuglevel()`, `has_extn()`, `close()`; stub network methods `connect`, `helo`, `ehlo`, `ehlo_or_helo_if_needed`, `starttls`, `login`, `auth`, `sendmail`, `send_message`, `quit`, `noop`, `rset`, `verify`, `expn`, `help`, `docmd`; context manager.

**SMTP_SSL(SMTP)** and **LMTP(SMTP)** subclasses.

14 test cases verified against CPython 3.14 (fixture 239).

## v0.0.238 - 2026-04-26

`imaplib` module — constants, nested exception hierarchy, utility functions, `IMAP4` class with full stub API, `IMAP4_SSL` and `IMAP4_stream` subclasses.

**Constants:** `IMAP4_PORT = 143`, `IMAP4_SSL_PORT = 993`, `Debug = 0`.

**Utility functions:** `Int2AP(num)` base-16 bytes using A-P chars; `ParseFlags(resp)` extracts flag tuple from IMAP response; `Time2Internaldate(date_time)` formats a Unix timestamp as an IMAP4 internal date string; `Internaldate2tuple(resp)` parses an IMAP4 date string to a 9-element tuple (returns `None` on mismatch).

**IMAP4 nested exceptions:** `IMAP4.error(Exception)` → `IMAP4.abort` → `IMAP4.readonly`; catchable at any level.

**IMAP4 class:** `__init__(host='', port=143, timeout=None)`; `set_debuglevel()`; full stub network API covering all RFC 2060/2195/2086/2087/2342/3501 commands: `login`, `logout`, `capability`, `noop`, `select`, `examine`, `search`, `fetch`, `store`, `copy`, `move`, `expunge`, `append`, `check`, `close`, `list`, `lsub`, `create`, `delete`, `rename`, `subscribe`, `unsubscribe`, `status`, `sort`, `thread`, `uid`, `getacl/setacl/deleteacl/myrights`, `getquota/setquota/getquotaroot`, `namespace`, `enable`, `xatom`, `getannotation/setannotation`, `proxyauth`, `starttls`, `partial`, `recent`, `response`; context manager.

**IMAP4_SSL(IMAP4):** `default_port = 993`. **IMAP4_stream(IMAP4):** no extra methods.

10 test cases verified against CPython 3.14 (fixture 238).

## v0.0.237 - 2026-04-26

`poplib` module — constants, `error_proto` exception, `POP3` class with full stub API, `POP3_SSL` subclass.

**Constants:** `POP3_PORT = 110`, `POP3_SSL_PORT = 995`.

**Exception:** `error_proto(Exception)` — raised on POP3 protocol errors.

**POP3 class:** `__init__(host, port=110, timeout=None)` stores attrs without connecting; `set_debuglevel()`, `getwelcome()`; stub network methods: `user()`, `pass_()`, `stat()`, `list()`, `retr()`, `top()`, `dele()`, `noop()`, `rset()`, `quit()`, `uidl()`, `apop()`, `rpop()`, `capa()`, `utf8()`, `stls()`, `close()`; context manager.

**POP3_SSL(POP3):** `default_port = 995`; inherits all POP3 methods.

6 test cases verified against CPython 3.14 (fixture 237).

## v0.0.236 - 2026-04-26

`ftplib` module deep coverage — constants, exception hierarchy, parse functions, FTP class, FTP_TLS subclass.

**Constants:** `FTP_PORT = 21`, `MAXLINE = 8192`, `MSG_OOB = 1`, `CRLF = "\r\n"`, `B_CRLF = b"\r\n"`.

**Exception hierarchy:** `Error(Exception)` → `error_reply`, `error_temp`, `error_perm`, `error_proto`. `all_errors = (Error, OSError, EOFError)`.

**Parse functions:** `parse150(resp)` extracts file size from 150 response (None if absent, raises `error_reply` if non-150); `parse227(resp)` extracts `(host, port)` from PASV response (raises `error_proto` on bad format); `parse229(resp, peer)` extracts host/port from EPSV response; `parse257(resp)` extracts directory name from MKD/PWD response (returns `""` for non-compliant, raises `error_reply` if not 257). `print_line(line)` prints to stdout.

**FTP class:** class-level defaults (`host`, `port`, `sock`, `file`, `welcome`, `passiveserver`, `trust_server_pasv_ipv4_address`, `debugging`, `maxline`); pure methods `set_debuglevel()`/`debug()`, `set_pasv()`, `sanitize()` (masks PASS password), `getwelcome()`; context manager (`__enter__`/`__exit__`); network stubs (no real TCP): `connect()`, `login()`, `close()`, `quit()`, `sendcmd()`, `voidcmd()`, `pwd()`, `cwd()`, `mkd()`, `rmd()`, `delete()`, `rename()`, `size()`, `abort()`, `nlst()`, `dir()`, `retrbinary()`, `retrlines()`, `storbinary()`, `storlines()`, `acct()`.

**FTP_TLS(FTP):** stub subclass with `auth()`, `ccc()`, `prot_c()`, `prot_p()`.

30 new test fixtures (fixture 236).

## v0.0.235 - 2026-04-26

`http.client` deep coverage — exception hierarchy, HTTPMessage, parse_headers, HTTPConnection, HTTPSConnection, responses dict, and status-code integer re-exports.

**Exception hierarchy (14 classes):**
`HTTPException(Exception)` → `NotConnected`, `InvalidURL`, `UnknownProtocol` (`.version`), `UnknownTransferEncoding`, `UnimplementedFileMode`, `IncompleteRead` (`.partial`, `.expected`, custom `repr()`), `ImproperConnectionState` → `CannotSendRequest`, `CannotSendHeader`, `ResponseNotReady`; `BadStatusLine` (`.line`); `LineTooLong` (computed message from line_type arg); `RemoteDisconnected(ConnectionResetError, BadStatusLine)` — catchable as `OSError`, `ConnectionResetError`, `BadStatusLine`, and `HTTPException`.

**HTTPMessage:** case-insensitive ordered multi-value header container; `get()`, `get_all()`, `get_content_type()`, `keys()`, `values()`, `items()`, `__contains__`, `__len__`, `__getitem__`, `__setitem__`.

**parse_headers(fp):** reads lines from a BytesIO/file-like object until blank line; returns `HTTPMessage`.

**HTTPConnection(host, port, timeout):** `.host`, `.port`, `.timeout`, `.debuglevel`; `set_debuglevel()`, `set_tunnel()`, `connect()`, `close()`, `send()`, `request()`, `putheader()`, `putrequest()`, `endheaders()` (stubs — no real TCP); `getresponse()` raises `ResponseNotReady`.

**HTTPSConnection(HTTPConnection):** `default_port = 443`; inherits all connection methods.

**responses:** dict mapping integer status codes to phrase strings for all IANA codes.

**Status code re-exports:** all `http.HTTPStatus` names exported as plain integers (`OK = 200`, `NOT_FOUND = 404`, …) including CPython 3.14 aliases (`RANGE_NOT_SATISFIABLE`, `UNPROCESSABLE_CONTENT`, `CONTENT_TOO_LARGE`, `URI_TOO_LONG`).

33 new test fixtures (fixture 235).

## v0.0.234 - 2026-04-26

`http` module deep coverage — `HTTPStatus` and `HTTPMethod` enums.

`http.HTTPStatus` is an IntEnum with all standard IANA HTTP status codes (1xx–5xx). Each member carries `.value` (int), `.phrase` (short description string), and `.description` (long description string). Boolean properties — `.is_informational`, `.is_success`, `.is_redirection`, `.is_client_error`, `.is_server_error`, `.is_integer` — are set directly on each member instance. Ordering (`<`, `<=`, `>`, `>=`) and equality with plain integers work as expected for IntEnum. Construction by value (`HTTPStatus(200)`) and subscript by name (`HTTPStatus["OK"]`) are supported. Iteration yields all members in definition order.

`http.HTTPMethod` is a StrEnum (Python 3.11+) with nine standard methods: CONNECT, DELETE, GET, HEAD, OPTIONS, PATCH, POST, PUT, TRACE. Each member carries `.value` (the method string), `.name`, and `.description`. `str(HTTPMethod.GET) == "GET"` and equality with plain strings both work. Construction by value (`HTTPMethod("GET")`) and subscript by name (`HTTPMethod["GET"]`) are supported.

32 new test fixtures (fixture 234).

## v0.0.233 - 2026-04-26

`urllib.robotparser` deep coverage — correct return types, `mtime`/`modified`, full-URL `can_fetch`, first-match-wins semantics.

`crawl_delay(agent)` now returns `int` (was string); values that are not valid integers (e.g. `1.5`) correctly return `None`, matching CPython 3.14.

`request_rate(agent)` now returns a `RequestRate(requests, seconds)` instance with integer `.requests` and `.seconds` attributes (was string); `repr()` shows `RequestRate(requests=N, seconds=M)`. `RequestRate` is exported from the module.

`mtime()` now returns `0` before any call to `parse()` or `modified()`, and the current Unix timestamp (float) after calling either — `parse()` internally calls `modified()`, exactly as CPython does.

`modified()` was previously a no-op; it now updates the stored mtime to the current time.

`can_fetch(useragent, url)` now extracts the path from full HTTP/HTTPS URLs before matching, so `can_fetch('*', 'http://example.com/private/')` and `can_fetch('*', '/private/')` behave identically.

**Matching semantics fixed:** the engine now uses **first-match wins** (CPython 3.14 behaviour) instead of longest-match. Rules are evaluated in document order and the first matching rule determines the result. Empty `Disallow:` is correctly converted to an allow-all rule.

**Webbrowser fix:** `webbrowserLaunch()` no longer spawns a real OS browser process. The three tests in fixture 227 (`test_open_returns_bool`, etc.) that previously caused Safari/Chrome to open `http://example.com` now run silently in CI and on developer machines.

24 new test fixtures (fixture 233). Fixture 229 updated: `rfp.crawl_delay("mybot") == "10"` → `== 10`.

## v0.0.232 - 2026-04-26

`urllib.error` deep coverage — complete attribute set, response methods, and exception-hierarchy fixes.

`URLError`: `reason` works for both string and exception-instance arguments; `str()` now calls the class `__str__` (returns `<urlopen error …>`) rather than the raw first arg; `repr()` now calls the class `__repr__` for exception subclasses; `issubclass(URLError, OSError)` and catch-as-`OSError` confirmed.

`HTTPError`: added `filename` attribute (alias for `url`); new methods — `read([n])` delegates to `fp.read()`; `getcode()` returns the integer status code; `geturl()` returns the request URL; `info()` returns the `hdrs` mapping; `close()` closes the file-pointer; `__enter__`/`__exit__` context-manager support; `__repr__` now returns `<HTTPError code: 'msg'>` format; `isinstance(e, URLError)` and `isinstance(e, OSError)` confirmed; catchable as both.

`ContentTooShortError`: `reason` and `content` attributes confirmed; `str()` returns `<urlopen error …>` via inherited `URLError.__str__`; hierarchy confirmed.

**VM fix:** `str()` builtin now calls the class `__str__` for `*object.Exception` subclasses (previously bypassed it, returning the raw first arg). Same fix for `repr()` and class-defined `__repr__`.

24 new test fixtures (fixture 232).

## v0.0.231 - 2026-04-26

`urllib.parse` deep coverage — complete `ParseResult`/`SplitResult` attribute set, bytes variants, and improved query-string functions.

`ParseResult` and `SplitResult`: added `username` and `password` attributes (decoded from the `user:pass@host` portion of `netloc`); `geturl()` method reconstructs the full URL from components; `encode(encoding='ascii')` returns a `ParseResultBytes`/`SplitResultBytes` instance with all fields as `bytes`; `SplitResult.__repr__` now renders as `SplitResult(scheme=…, netloc=…, path=…, query=…, fragment=…)` (5 fields, no `params`), while `ParseResult.__repr__` retains its 6-field format; `SplitResult[i]` indexing returns 5 elements.

`ParseResultBytes` / `SplitResultBytes` / `DefragResultBytes`: full bytes-valued result classes returned when the input to `urlparse`/`urlsplit`/`urldefrag` is `bytes`; `decode(encoding='ascii')` converts back to the str result class; `geturl()` returns `bytes`.

`DefragResult`: added `__getitem__` (index 0 → url, 1 → fragment), `__len__` (always 2), `__iter__` (yields url then fragment), and `encode()` → `DefragResultBytes`.

`urlparse` / `urlsplit`: new `scheme=''` kwarg used as the default scheme when the URL has none; new `allow_fragments=True` kwarg (False folds the fragment into path/query).

`urljoin`: `allow_fragments=True` kwarg strips the fragment from the result when False.

`urlencode`: `quote_via` kwarg accepts any callable (e.g. `urllib.parse.quote`) used for percent-encoding values — the default remains `quote_plus`; `safe=''` kwarg forwarded to `quote_via`.

`parse_qs` / `parse_qsl`: `keep_blank_values=False` kwarg retains params with empty values when True; `separator='&'` kwarg sets the field delimiter (single char); `max_num_fields=None` kwarg raises `ValueError` when exceeded.

Exported classes: `ParseResult`, `SplitResult`, `ParseResultBytes`, `SplitResultBytes`, `DefragResult`, `DefragResultBytes` are all accessible as `urllib.parse.*`.

24 test fixtures (fixture 231).

## v0.0.230 - 2026-04-26

`urllib.request` deep coverage — complete handler hierarchy, real `OpenerDirector`, `urlretrieve`, and response methods.

`Request` class: full attribute set including `full_url` (parses scheme/host/selector), `type`, `host`, `origin_req_host`, `selector`, `data`, `unverifiable`, `_method`; `get_method()` returns POST when data is set; `set_proxy(host, type)`; `add_header`/`has_header`/`get_header`/`remove_header` with capitalize-key semantics; `add_unredirected_header` (stored separately, not sent on redirect); `header_items()` merges both header maps; state stored per-instance in a package-level `sync.Map`.

`HTTPPasswordMgr` hierarchy: `HTTPPasswordMgr` — `add_password(realm, uri, user, passwd)`, `find_user_password(realm, authuri)` with longest-URI-prefix matching; `HTTPPasswordMgrWithDefaultRealm` — falls back to `realm=None` entry when no exact match; `HTTPPasswordMgrWithPriorAuth` — adds `update_authenticated(uri, flag)` and `is_authenticated(authuri)`. All three handle `realm=None` by canonicalising to an empty-string key.

`Handler` classes with real logic: `HTTPDefaultErrorHandler.http_error_default` raises `HTTPError`; `HTTPErrorProcessor.http_response`/`https_response` passes 2xx through and calls `opener.error` for others; `HTTPRedirectHandler.redirect_request` builds a new `Request` for the redirect target and implements `http_error_301/302/303/307/308`; `HTTPCookieProcessor.__init__(cookiejar=None)` stores the jar; `ProxyHandler.__init__(proxies=None)` stores the proxy map; `AbstractBasicAuthHandler.http_error_auth_reqed` extracts realm from `WWW-Authenticate`, fetches credentials from the password manager, encodes them as Base64, and re-opens via the parent opener; `HTTPBasicAuthHandler.http_error_401` / `ProxyBasicAuthHandler.http_error_407`; `AbstractDigestAuthHandler` / `HTTPDigestAuthHandler` / `ProxyDigestAuthHandler` stubs; `AbstractHTTPHandler.do_request_` adds `Content-Type`/`Content-Length` when `data` is set; `AbstractHTTPHandler.do_open` performs the real HTTP request via Go `net/http`; `HTTPHandler.http_open` / `HTTPSHandler.https_open`; `FileHandler.open_local_file`/`file_open` reads local files; `DataHandler.data_open` full RFC 2397 (plain and base64); `UnknownHandler.unknown_open` raises `URLError`; `CacheFTPHandler.setTimeout`/`setMaxConns` stubs.

`addinfourl` response object: `read(n=-1)`, `readline()`, `readlines()`, `close()`, `geturl()`, `info()`, `getcode()`, `__enter__`/`__exit__` context-manager support.

`OpenerDirector`: complete `add_handler` (scans class dict for `protocol_open`, `http_error_NNN`, `protocol_request`, `protocol_response` methods); `open(url, data, timeout)` runs three-stage dispatch (request processors → open handlers → response processors); `error(proto, *args)` dispatches through `http_error_NNN` chains then `http_error_default`; `build_opener(*handlers)` installs the default handler set.

`urlretrieve(url, filename=None, reporthook=None, data=None)` downloads to a temp file when no filename is given, calls `reporthook(block, blocksize, totalsize)` every 8192 bytes, and tracks temp files in a `sync.Map`. `urlcleanup()` removes all tracked temp files.

24 test fixtures (fixture 230).

## v0.0.229 - 2026-04-26

`urllib` package — urllib.parse additions plus three new sub-modules.

`urllib.parse` additions: `urldefrag(url)` returns a `DefragResult` instance with `.url` and `.fragment` attributes; `quote_from_bytes(bytes, safe='/')` percent-encodes a bytes object; `unquote_to_bytes(str)` percent-decodes a string to bytes; `unwrap(url)` strips leading `<`/`>` angle brackets and `URL:` prefix.

`urllib.error`: `URLError(reason)` (base: `OSError`) with `.reason`; `HTTPError(url, code, msg, hdrs, fp)` (base: `URLError`) with `.url`, `.code`, `.reason`, `.headers`, `.hdrs`, `.fp`; `ContentTooShortError(msg, content)` with `.content`. Attribute access on all three is backed by the constructor `Args` tuple via `__getattr__`; `__str__` gives human-readable representations (`<urlopen error …>` / `HTTP Error N: …`).

`urllib.request`: `Request(url, data=None, headers={}, …, method=None)` with `.full_url`, `.type`, `.host`, `.selector`, `.data`, `get_method()`, `add_header`, `has_header`, `get_header`, `remove_header`, `add_unredirected_header`, `header_items` (headers stored with capitalize-key semantics via a `sync.Map`); `urlopen(url, data=None)` supports `data:` URIs (RFC 2397 — plain percent-encoded and base64 variants) and HTTP/HTTPS via Go `net/http`; returns `addinfourl` with `read(n)`, `readline`, `readlines`, `close`, `url`, `status`, `code`, `headers`, `geturl`, `info`, `getcode`, and context-manager support; `build_opener(*handlers)` → `OpenerDirector` (with `open`, `add_handler`, `error`); `install_opener(opener)`; `OpenerDirector`; full set of 19 handler class stubs (`HTTPDefaultErrorHandler`, `HTTPRedirectHandler`, `HTTPCookieProcessor`, `ProxyHandler`, `HTTPPasswordMgr`, `HTTPPasswordMgrWithDefaultRealm`, `HTTPPasswordMgrWithPriorAuth`, `HTTPBasicAuthHandler`, `ProxyBasicAuthHandler`, `HTTPDigestAuthHandler`, `ProxyDigestAuthHandler`, `HTTPHandler`, `HTTPSHandler`, `FileHandler`, `DataHandler`, `FTPHandler`, `CacheFTPHandler`, `UnknownHandler`, `HTTPErrorProcessor`); `pathname2url`, `url2pathname`, `getproxies`, `urlretrieve`, `urlcleanup`.

`urllib.robotparser`: `RobotFileParser(url='')` — `set_url`, `read` (fetches via HTTP), `parse(lines)` (groups `User-agent`/`Allow`/`Disallow`/`Crawl-delay`/`Sitemap` directives), `can_fetch(agent, path)` (longest-match-wins with Allow precedence over Disallow, wildcard `*` fallback), `crawl_delay(agent)`, `request_rate(agent)`, `site_maps()`, `mtime()`, `modified()`. Internal state stored in a `sync.Map` keyed by instance pointer.

VM fix: added `__getattr__` fallback support for `*object.Exception` instances in `getAttr`, mirroring the existing `*object.Instance` behaviour.

18 test fixtures cover all sub-modules (fixture 229).

## v0.0.228 - 2026-04-26

`wsgiref` package — all six sub-modules.

`wsgiref.util`: `guess_scheme`, `setup_testing_defaults`, `request_uri`, `application_uri`, `shift_path_info`, `is_hop_by_hop`, and `FileWrapper` (iterable file wrapper with configurable block size). `wsgiref.headers`: `Headers` class — case-insensitive HTTP header mapping over a `(name, value)` list; `__getitem__`/`__setitem__`/`__delitem__`/`__contains__`/`__len__`/`__iter__`, `get`, `setdefault`, `keys`, `values`, `items`, `get_all`, `add_header` (MIME params via kwargs), `__bytes__` (formatted HTTP header block). `wsgiref.simple_server`: `demo_app`, `make_server` (binds a real TCP listener via Go `net.Listen`), `WSGIServer` (`set_app`/`get_app`/`server_close`/`serve_forever`/`handle_request`), `WSGIRequestHandler`. `wsgiref.handlers`: `SimpleHandler` whose `run(app)` calls the WSGI app, then writes a complete `HTTP/1.0` response (status line, headers, blank line, body) to the `stdout` file-like object via Python's write protocol; `BaseHandler`, `BaseCGIHandler`, `CGIHandler`, `IISCGIHandler`, `read_environ` stubs. `wsgiref.validate`: `validator` pass-through wrapper. `wsgiref.types`: runtime `None` stubs for all PEP 3333 type aliases.

21 test fixtures cover all sub-modules (fixture 228).

## v0.0.227 - 2026-04-26

`webbrowser` module.

Adds the `webbrowser` module with `open(url, new=0, autoraise=True)`, `open_new(url)`, `open_new_tab(url)`, `get(using=None)`, and `register(name, constructor, instance=None, *, preferred=False)`. The `Error` exception class and the `Controller` base class (with `name` attribute and `open`/`open_new`/`open_new_tab` methods) are exposed. The built-in `Controller` delegates to the OS default browser (`open` on macOS, `xdg-open` on Linux, `cmd /c start` on Windows). `register` stores controllers in a per-module registry; `preferred=True` prepends the name to the preferred list so `get()` and module-level `open` dispatch through it first.

11 test fixtures cover: `Error` class hierarchy, `Controller` instantiation, `register`+`get` roundtrip, `get` error for unknown browser, module-level callables, `open`/`open_new`/`open_new_tab` return types, default `get()`, preferred browser dispatch, and constructor-based registration (fixture 227).

## v0.0.226 - 2026-04-26

`pyexpat` module deep coverage per Python 3.14.

Adds the `pyexpat` module (the C extension underlying `xml.parsers.expat`) as a first-class goipy module. `ParserCreate([encoding[, namespace_separator]])` returns an `xmlparser` instance (exposed as `XMLParserType`). The full handler set is supported: `StartElementHandler`, `EndElementHandler`, `CharacterDataHandler`, `ProcessingInstructionHandler`, `CommentHandler`, `XmlDeclHandler`, `StartCdataSectionHandler`/`EndCdataSectionHandler`, `StartDoctypeDeclHandler`/`EndDoctypeDeclHandler`, `NotationDeclHandler`, `StartNamespaceDeclHandler`/`EndNamespaceDeclHandler`, and stubs for the remaining handlers. Parse errors raise `ExpatError` with `lineno`, `offset`, and `code` attributes; the `Error*` and `Current*` position attributes are updated accordingly. `ordered_attributes=True` mode produces a flat `[name, val, …]` list instead of a dict. Incremental parsing (`isfinal=False`) buffers data until the final call. `ExternalEntityParserCreate`, `SetBase`/`GetBase`, `SetParamEntityParsing`, `UseForeignDTD`, `GetInputContext` are all implemented. The `errors` sub-module exports all 44 `XML_ERROR_*` string constants plus `codes` and `messages` dicts. The `model` sub-module exports `XML_CTYPE_*` and `XML_CQUANT_*` integer constants. `pyexpat.errors` and `pyexpat.model` are also importable as sub-packages.

25 test fixtures cover all of the above (fixture 226).

## v0.0.225 - 2026-04-26

`xml.sax.xmlreader` deep coverage per Python 3.14.

Rewrites `XMLReader` with a proper `__init__` that initialises four default handler slots (`_cont_handler`, `_dtd_handler`, `_ent_handler`, `_err_handler`) from `xml.sax.handler`, adds `getDTDHandler`/`setDTDHandler`/`getEntityResolver`/`setEntityResolver`, and makes `setLocale` raise `SAXNotSupportedException` and `getFeature`/`setFeature`/`getProperty`/`setProperty` raise `SAXNotRecognizedException` with the feature/property name in the message. Rewrites `IncrementalParser` with `__init__(bufsize=65536)`, abstract `feed`/`close`/`reset`/`prepareParser` methods (raise `NotImplementedError`), and a concrete `parse(source)` that reads from the source's byte/char stream in `_bufsize`-sized chunks. Adds `values()` and `get(name, alternative=None)` to `AttributesImpl`, and fixes `getQNameByName`/`getNameByQName` to raise `KeyError` when the name is missing. Fully implements `AttributesNSImpl(attrs, qnames)` with NS tuple keys: all access/lookup methods, `getValueByQName`/`getNameByQName`/`getQNameByName` via reverse scan, and `copy()` returning a new `AttributesNSImpl`.

21 test fixtures cover all of the above (fixture 225).

## v0.0.224 - 2026-04-26

`xml.sax.saxutils` deep coverage per Python 3.14.

Fixes `escape` to process `&` → `>` → `<` in the correct order (ampersand first, matching CPython exactly). Fixes `unescape` ordering: `&lt;`/`&gt;` decoded first, custom entities next, `&amp;` last — so `&amp;lt;` correctly becomes `&lt;` rather than `<`. Adds whitespace escaping to `quoteattr` (`\n` → `&#10;`, `\r` → `&#13;`, `\t` → `&#9;`). Rewrites `XMLGenerator` with proper `short_empty_elements` support (`_pending_start_element` flag for `<br/>` collapse), correct attribute quoting via `quoteattr`-style (single-quote wrap when value contains `"`), truthy guard on `characters`/`ignorableWhitespace`, and `flush()` in `endDocument`. Rewrites `XMLFilterBase` with `_parent`, all four handler get/set pairs, and full delegation from `ContentHandler`/`ErrorHandler`/`DTDHandler`/`EntityResolver`. Adds `prepare_input_source` (str → `InputSource`, bytes → `BytesIO`-backed, `InputSource` pass-through).

21 test fixtures cover every function and edge case listed above (fixture 224).

## v0.0.223 - 2026-04-26

`xml.sax.handler` deep coverage per Python 3.14.

Adds the missing `version = "2.0beta"` module attribute. Fixes `all_properties` ordering to match Python exactly: `dom-node` now appears before `declaration-handler` in the list. Fixes `ErrorHandler.warning` to call `print(exception)` rather than silently do nothing, matching Python's base-class implementation.

14 test fixtures verify every constant value and ordering, all five handler base classes, `EntityResolver.resolveEntity` return value, `ErrorHandler` raise/no-op semantics, and `LexicalHandler` no-op stubs (fixture 223).

## v0.0.222 - 2026-04-25

Full `xml.sax` coverage per Python 3.14: the push-based SAX2 parser, all four handler base classes, saxutils helpers, and the xmlreader types.

`xml.sax` — `SAXException`, `SAXParseException`, `SAXNotRecognizedException`, `SAXNotSupportedException`, `SAXReaderNotAvailable` are all proper exception subclasses. `parseString` and `parse` (file-like, bytes, or path) dispatch `startDocument`, `startElement`, `endElement`, `characters`, `processingInstruction`, `endDocument` to any `ContentHandler`. `make_parser()` returns an `ExpatParser` with the full get/set handler API: `setContentHandler`, `getContentHandler`, `setErrorHandler`, `getErrorHandler`, `setDTDHandler`, `getDTDHandler`, `setEntityResolver`, `getEntityResolver`, `setFeature`, `getFeature`, `setProperty`, `getProperty`, `reset`.

`xml.sax.handler` — `ContentHandler` (13 no-op methods), `DTDHandler`, `EntityResolver`, `ErrorHandler` (error/fatalError raise, warning is no-op), `LexicalHandler` (comment, startDTD, endDTD, startCDATA, endCDATA). All six `feature_*` and six `property_*` string constants; `all_features` and `all_properties` lists.

`xml.sax.saxutils` — `escape`, `unescape`, `quoteattr`; `XMLGenerator` writes `<?xml ...?>`, elements, and escaped text to any file-like; `XMLFilterBase` stub.

`xml.sax.xmlreader` — `XMLReader`, `IncrementalParser`, `Locator` (`getColumnNumber`/`getLineNumber` return −1), `InputSource` (full getter/setter set), `AttributesImpl` (keys, items, copy, getQNames, getValueByQName, getQNameByName, getNameByQName, __len__, __getitem__, __contains__), `AttributesNSImpl`.

Also fixes exception method dispatch: methods defined on exception subclass dicts are now returned as bound methods, making user-defined methods on exception instances callable without panicking.

23 test fixtures cover all of the above (fixture 222).

## v0.0.221 - 2026-04-25

This release completes `xml.dom.pulldom` with a working pull API, real DOM nodes, and `expandNode`.

Previously the module had event constants and `parseString`/`parse` that returned a stream object, but that stream was basically a wrapper around a pre-built list with no way to drive it from Python code. The iteration worked by returning the list itself, which meant `for event, node in stream:` happened to work, but `getEvent()`, `reset()`, and `expandNode()` did nothing.

`getEvent()` is now the primary pull interface — it returns the next `(event, node)` tuple and advances the cursor, or returns `None` when exhausted. `reset()` resets the cursor to the beginning so the stream can be replayed. `__iter__` returns `self` (proper iterator protocol) and `__next__` raises `StopIteration` when done.

`expandNode(node)` is now real. After you receive a `START_ELEMENT` event, calling `expandNode(node)` consumes the remaining events for that element's subtree and builds the full DOM tree under `node`. Once done, `node.childNodes.length` is correct, `node.firstChild` and `node.lastChild` work, text children have `.data`, and nested elements are fully populated.

Nodes in the event stream are now real minidom nodes backed by `domNodeState`, so all the minidom APIs (`getAttribute`, `childNodes`, `firstChild`, `lastChild`, `data`, `target`) work on them without `expandNode`.

`START_ELEMENT` and `END_ELEMENT` events for the same tag now return the same node instance. `START_DOCUMENT` and `END_DOCUMENT` return a real Document node. `default_bufsize = 8192` is present. `PullDOM` class added (stub).

18 new test fixtures cover all of the above (fixture 221).

## v0.0.220 - 2026-04-25

This release completes the `xml.dom.minidom` serialization layer and wires up several properties that were missing or returning wrong values.

`writexml` is now the real serialization core, matching CPython's rules exactly: an element with zero children emits a self-closing tag, an element with exactly one `Text` child emits it inline (no indentation on the text), and everything else gets the indent/addindent/newl treatment with each child on its own line. `toxml` and `toprettyxml` both call through to `writexml` and support the `encoding` and `standalone` keyword arguments. Passing `encoding` returns `bytes` with the right `encoding="..."` declaration; `standalone=True/False` inserts `standalone="yes"/"no"`. `Document.writexml` also accepts those extra keywords.

`Attr` objects now carry the full set of properties CPython minidom exposes: `specified` is always `False`, `ownerElement` points back to the element that owns the attribute, `localName` strips any namespace prefix, `prefix` and `namespaceURI` are `None` for unqualified attrs.

`Document.doctype` is a live property that scans the document children for a `DocumentType` node and returns it, or `None` if there is none. `Document.implementation` returns the `DOMImplementation` singleton. The `Document` class now works as a context manager so `with minidom.parseString(...) as doc:` works.

`Node.localName` is now set for all node types, not just elements. Non-element, non-attribute nodes get `None`.

12 new test fixtures cover all these features (fixture 220).

## v0.0.219 - 2026-04-25

This release fills in the parts of `xml.dom` and `xml.dom.minidom` that were stubs before.

The main additions: `firstChild`, `lastChild`, `previousSibling`, and `nextSibling` are now live properties that reflect the current tree instead of being hardcoded `None`. `cloneNode(deep)` does a real structural copy. `normalize()` merges adjacent text nodes. The `CharacterData` interface -- `substringData`, `appendData`, `insertData`, `deleteData`, `replaceData`, `length`, and direct `data` assignment -- is wired up on `Text`, `Comment`, and `CDATASection`. `Text.splitText(offset)` splits a text node in place and inserts the tail into the parent. `createDocumentFragment` now returns a proper `DocumentFragment` node instead of the wrong class. `importNode(node, deep)` clones a node into the target document. `NamedNodeMap.getNamedItemNS`, `setNamedItemNS`, and `removeNamedItemNS` are real implementations. `Element.hasAttributeNS` is available. `DOMImplementation.createDocument` returns a full working `Document`.

13 new test fixtures cover these features (fixture 219).

## v0.0.218.1 - 2026-04-25

Patch release adding Windows support. The stdlib modules that used Unix-only syscalls (`os`, `signal`, `socket`, `shutil`, `time`, `mmap`, `select`, `selectors`) now have Windows-compatible implementations. The release workflow now builds and ships a `windows/amd64` binary alongside the four Unix targets.

## v0.0.218 - 2026-04-25

First release of goipy. The version number tracks the number of merged PRs; v0.0.218 means 218 test fixtures passing against CPython 3.14 bytecode.

### What is goipy

A pure-Go interpreter for CPython 3.14 `.pyc` files. No cgo, no libpython, no subprocess. You compile your Python with `python3.14 -m py_compile`, ship the `.pyc` alongside a single static binary, and goipy runs it. Works on Linux, macOS, and Windows.

### Core language

The bytecode eval loop covers the full CPython 3.14 instruction set: LOAD/STORE/DELETE for all scopes (local, global, deref, fast), CALL/CALL_INTRINSIC variants, MAKE_FUNCTION with closures and defaults, all comprehension forms, generators (sync and async), match/case with all pattern types, with/async-with, exception groups, and the full set of binary/unary/comparison ops.

Class creation goes through `__init_subclass__`, `__set_name__`, and `__class_cell__`. Descriptors (`__get__`, `__set__`, `__delete__`) work on both instances and classes. `super()` works with zero and two arguments.

### Standard library

The stdlib covers about 100 modules:

**Built-in types:** `int`, `float`, `complex`, `str`, `bytes`, `bytearray`, `list`, `tuple`, `dict`, `set`, `frozenset`, `bool`, `memoryview`, `range`, `slice` -- all with their full CPython method surfaces.

**Numerics:** `math`, `cmath`, `decimal`, `fractions`, `statistics`, `random`, `struct`

**Text:** `re` (backed by Go `regexp/syntax`), `string`, `textwrap`, `difflib`, `unicodedata`, `stringprep`, `readline`

**Data structures:** `collections` (deque, OrderedDict, Counter, ChainMap, namedtuple, defaultdict), `heapq`, `bisect`, `array`, `enum`, `dataclasses`, `copy`

**Serialization:** `json` (full CPython 3.14 encoder/decoder), `csv`, `configparser`, `tomllib`, `plistlib`, `pickle`, `shelve`, `marshal`

**Binary/encoding:** `base64`, `binascii`, `codecs`, `quopri`, `uu`

**OS and files:** `os`, `os.path`, `sys`, `pathlib`, `io`, `shutil`, `glob`, `fnmatch`, `tempfile`, `fileinput`, `stat`, `mmap`, `gzip`, `bz2`, `lzma`, `zlib`, `zipfile`, `tarfile`, `zstd`

**Networking and IPC:** `socket`, `ssl` (backed by Go `crypto/tls`), `select`, `selectors`, `signal`, `subprocess`, `mmap`

**Concurrency:** `threading`, `multiprocessing`, `multiprocessing.shared_memory`, `concurrent.futures`, `concurrent.interpreters`, `asyncio` (full CPython 3.14 API), `queue`, `sched`, `contextvars`, `_thread`

**Internet and email:** `email` (message, mime types, utils, header, encoders, generator, parser), `mailbox` (mbox, Maildir), `mimetypes`, `html.parser`, `html.entities`

**XML:** `xml.etree.ElementTree` (full API including QName, TreeBuilder, iterparse, canonicalize), `xml.dom` (NodeList, NamedNodeMap, Attr, DOMImplementation, 15 DOMException subclasses, NS methods), `xml.dom.minidom`, `xml.sax`, `xml.parsers.expat`

**Development tools:** `argparse`, `optparse`, `getpass`, `logging`, `unittest`, `pdb`, `traceback`, `inspect`, `dis`, `ast`, `tokenize`, `typing`, `abc`

**Functional:** `functools` (full coverage), `itertools` (full coverage), `operator` (full coverage)

**Other:** `datetime`, `time`, `calendar`, `hashlib`, `hmac`, `secrets`, `uuid`, `locale`, `gettext`, `importlib`, `pkgutil`, `zipimport`, `ctypes`, `errno`, `curses` (textpad, ascii, panel), `cmd`, `gc`, `weakref`, `contextlib`, `warnings`

### Threading model

`threading.Thread` maps to goroutines. The GIL is not emulated; Go's runtime scheduler manages goroutines. `threading.Lock`, `RLock`, `Condition`, `Semaphore`, `Event`, and `Barrier` all map to Go sync primitives. `multiprocessing` forks OS processes via `os/exec`.

### Binaries

Each release ships pre-built binaries for:

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`

All binaries are statically linked (CGO_ENABLED=0) and built with `-ldflags="-s -w"`.

## v0.0.217 - 2026-04-25

`xml.etree.ElementTree` extended: `QName`, `TreeBuilder`, `iterparse` with
`start`/`end`/`start-ns`/`end-ns` events, `canonicalize`.

## v0.0.216 - 2026-04-25

Full `xml` package: `xml.etree.ElementTree` (parse, Element, SubElement,
tostring, fromstring), `xml.sax` (ContentHandler, parse, parseString),
`xml.dom` (base classes, constants), `xml.parsers.expat`.

## v0.0.215 - 2026-04-25

`html.entities` extended to full CPython 3.14 spec: all 2231 named character
references in `html5` and `name2codepoint`/`codepoint2name` mappings.

## v0.0.214 - 2026-04-25

`html.parser` extended to full CPython 3.14 spec: `handle_starttag`,
`handle_endtag`, `handle_startendtag`, `handle_data`, `handle_comment`,
`handle_decl`, `handle_pi`, `handle_entityref`, `unknown_decl`.
`convert_charrefs`, `CDATA_CONTENT_ELEMENTS`.

## v0.0.213 - 2026-04-25

`html`: `unescape`, `escape`. `html.parser`: `HTMLParser`. `html.entities`:
`html5`, `name2codepoint`, `codepoint2name`.

## v0.0.212 - 2026-04-25

`quopri`: `encode`, `decode`, `encodestring`, `decodestring`. Quoted-printable
encoding per RFC 2045. `header` mode flag.

## v0.0.211 - 2026-04-25

`binascii` extended to full CPython 3.14 spec: `b2a_qp`, `a2b_qp`,
`b2a_hqx`, `a2b_hqx`, `rlecode_hqx`, `rledecode_hqx`, `crc_hqx`,
`b2a_uu`, `a2b_uu`, `b2a_base64` with `newline` param, `Error`, `Incomplete`.

## v0.0.210 - 2026-04-25

`base64` extended to full CPython 3.14 spec: `encodebytes`, `decodebytes`,
`b85encode`, `b85decode`, `a85encode`, `a85decode`, `b16encode`, `b16decode`,
`b32encode`, `b32decode`, `b32hexencode`, `b32hexdecode`, `urlsafe_b64encode`,
`urlsafe_b64decode`.

## v0.0.209 - 2026-04-25

`mimetypes`: `guess_type`, `guess_extension`, `guess_all_extensions`,
`add_type`, `init`, `read_mime_types`, `MimeTypes`. Common MIME types
pre-loaded.

## v0.0.208 - 2026-04-25

`mailbox`: `mbox`, `Maildir`, `MaildirMessage`, `mboxMessage`. `add`,
`remove`, `update`, `flush`, `lock`, `unlock`. `NoSuchMailboxError`.

## v0.0.207 - 2026-04-25

`json` extended to full CPython 3.14 feature set: `JSONDecodeError` with
`pos`/`lineno`/`colno`, `JSONEncoder` with `check_circular`, `allow_nan`,
`sort_keys`, `default` hook. `JSONDecoder` with `object_hook`,
`object_pairs_hook`, `parse_float`, `parse_int`, `parse_constant`.

## v0.0.206 - 2026-04-25

`email` package: `email.message.Message`, MIME classes (`MIMEText`,
`MIMEMultipart`, `MIMEBase`, `MIMEImage`, `MIMEAudio`, `MIMEApplication`).
`email.utils`, `email.header`, `email.encoders`, `email.generator`,
`email.parser`.

## v0.0.205 - 2026-04-25

`mmap`: file-backed and anonymous mappings. `ACCESS_READ`, `ACCESS_WRITE`,
`ACCESS_COPY`, `ACCESS_NONE`. `seek`, `tell`, `read`, `write`, `readline`,
`find`, `rfind`, `flush`, `resize`, `close`. Slice assignment.

## v0.0.204 - 2026-04-25

`signal`: `signal`, `getsignal`, `raise_signal`, `strsignal`, `valid_signals`,
`pause`, `alarm`, `setitimer`, `getitimer`. Handler registry and OS-level
signal delivery. `SIG_DFL`, `SIG_IGN`.

## v0.0.203 - 2026-04-25

`selectors`: `DefaultSelector`, `EpollSelector`, `KqueueSelector`,
`PollSelector`, `SelectSelector`. `register`, `unregister`, `modify`,
`select`, `get_key`, `get_map`. `EVENT_READ`, `EVENT_WRITE`.

## v0.0.202 - 2026-04-25

`select`: `select`, `poll`, `epoll`, `kqueue`, `kevent`. `POLLIN`, `POLLOUT`,
`POLLERR`, `POLLHUP`, `POLLNVAL`. `epoll` edge-triggered and level-triggered.

## v0.0.201 - 2026-04-25

`ssl` backed by Go `crypto/tls`: `SSLContext`, `wrap_socket`, `SSLSocket`,
`SSLError`. `PROTOCOL_TLS_CLIENT`, `PROTOCOL_TLS_SERVER`. Certificate
loading, hostname verification, `check_hostname`, `verify_mode`.

For releases v0.0.001 through v0.0.200, see the [changelog/](changelog/) folder.

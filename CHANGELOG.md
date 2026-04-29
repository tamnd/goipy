# Changelog

Individual release notes live in [`changelog/`](changelog/).

| Version | Date | Summary |
|---------|------|---------|
| [v0.0.346](changelog/0.0.346.md) | 2026-04-29 | `pickle.PickleBuffer` + `bytes(__bytes__)` + compile/exec/eval raise SyntaxError; eager `buildBuiltins()`; closes v0.1.0 tail, fixture 346 |
| [v0.0.345](changelog/0.0.345.md) | 2026-04-29 | `sys.settrace` / `sys.setprofile` -- per-frame `f_trace`, call/line/return/exception events, fixture 345 |
| [v0.0.344](changelog/0.0.344.md) | 2026-04-29 | `iter(callable, sentinel)` + `slice.indices(n)` + `functools._CacheInfo` namedtuple, fixture 344 |
| [v0.0.343](changelog/0.0.343.md) | 2026-04-29 | generator/coroutine/async-gen introspection -- `gi_*`/`cr_*`/`ag_*` (frame/running/code/suspended/await/origin), fixture 343 |
| [v0.0.342](changelog/0.0.342.md) | 2026-04-29 | explicit `super(C, inst)` + metaclass `__instancecheck__`/`__subclasscheck__` -- `Class.Metaclass`, fixture 342 |
| [v0.0.341](changelog/0.0.341.md) | 2026-04-29 | encoding handlers + `bytes.hex(sep, group)` -- `str.encode` honors `errors=`, `namereplace`, proper UnicodeEncodeError/DecodeError, fixture 341 |
| [v0.0.340](changelog/0.0.340.md) | 2026-04-29 | exception API -- `add_note`/`__notes__`, `with_traceback`, `__suppress_context__`, BaseExceptionGroup `split`/`subgroup`/`derive`, fixture 340 |
| [v0.0.339](changelog/0.0.339.md) | 2026-04-29 | async generators -- `__aiter__`/`__anext__`/`asend`/`athrow`/`aclose`, async for + comprehension, fixture 339 |
| [v0.0.338](changelog/0.0.338.md) | 2026-04-29 | PEP 585 / 604 / 560 -- `list[int]`, `int | str`, `__mro_entries__` substitution, fixture 338 |
| [v0.0.337](changelog/0.0.337.md) | 2026-04-29 | frame & code exposure -- `sys._getframe`, `f_*` attrs, `co_qualname`/`co_consts`/`co_lines()`, `inspect` chain walkers, fixture 337 |
| [v0.0.336](changelog/0.0.336.md) | 2026-04-29 | hash invariant -- CPython numeric hash (Mersenne P=2^61-1), `int`/`float`/`Fraction`/`Decimal`/`complex` collide, fixture 336 |
| [v0.0.335](changelog/0.0.335.md) | 2026-04-29 | function introspection -- `__code__`, `__defaults__`, `__closure__`, `__globals__`, PEP 649 lazy `__annotate__`, BoundMethod attrs, fixture 335 |
| [v0.0.334](changelog/0.0.334.md) | 2026-04-28 | `subprocess` shell handling -- shell=True positional args, `executable=` kwarg, `COMSPEC` on Windows, fixture 334 |
| [v0.0.333](changelog/0.0.333.md) | 2026-04-28 | `__slots__` semantics -- setattr/delattr gating, `__dict__` hiding, MRO-aware filtering, fixture 333 |
| [v0.0.332](changelog/0.0.332.md) | 2026-04-29 | threading correctness -- per-object locks on List/Set/Cell/Bytearray, atomic classEpoch, weakref table mutex, fixture 332 |
| [v0.0.331](changelog/0.0.331.md) | 2026-04-28 | v0.1.0 prep -- C3 MRO, all intrinsic1/2 cases, 20 specialized opcodes routed, ExceptionGroup split, PEP 695 closure fix |
| [v0.0.330](changelog/0.0.330.md) | 2026-04-28 | syslog deep coverage -- 7 missing macOS facility constants and fixture 330 |
| [v0.0.329](changelog/0.0.329.md) | 2026-04-28 | resource deep coverage -- ru_utime/ru_stime float fix and fixture 329 |
| [v0.0.328](changelog/0.0.328.md) | 2026-04-28 | fcntl deep coverage -- 6 new constants, 5 removed, F_RDAHEAD fix, fcntl/ioctl return int |
| [v0.0.327](changelog/0.0.327.md) | 2026-04-28 | pty deep coverage -- close, waitpid, setraw, tcgetattr, tcsetattr re-exports |
| [v0.0.326](changelog/0.0.326.md) | 2026-04-28 | tty deep coverage -- cfmakeraw, cfmakecbreak, struct-index constants, fixture 326 |
| [v0.0.325](changelog/0.0.325.md) | 2026-04-28 | termios deep coverage -- 4 constant fixes, remove 7 fake constants, fixture 325 |
| [v0.0.324](changelog/0.0.324.md) | 2026-04-28 | grp deep coverage -- getgrgid/getgrnam/getgrall fixes and fixture 324 |
| [v0.0.323](changelog/0.0.323.md) | 2026-04-28 | pwd deep coverage -- getpwnam/getpwall fixes and fixture 323 |
| [v0.0.322](changelog/0.0.322.md) | 2026-04-28 | posix deep coverage -- named result classes, 20+ new constants, 30+ new functions |
| [v0.0.321](changelog/0.0.321.md) | 2026-04-28 | shlex class, posix=False fix, and fixture 321 deep coverage |
| [v0.0.320](changelog/0.0.320.md) | 2026-04-28 | Unix modules (posix, pwd, grp, termios, tty, pty, fcntl, resource, syslog) -- fixture 320 |
| [v0.0.319](changelog/0.0.319.md) | 2026-04-28 | winsound dedicated fixture 319 -- full API coverage |
| [v0.0.318](changelog/0.0.318.md) | 2026-04-28 | winreg dedicated fixture 318 -- full API coverage plus missing constants |
| [v0.0.317](changelog/0.0.317.md) | 2026-04-28 | msvcrt dedicated fixture 317 -- full API coverage |
| [v0.0.316](changelog/0.0.316.md) | 2026-04-28 | Windows modules (msvcrt, winreg, winsound, msilib) -- fixture 316 |
| [v0.0.315](changelog/0.0.315.md) | 2026-04-28 | `pickletools` module — fixture 315 for https://docs.python.org/3/library/pick... |
| [v0.0.314](changelog/0.0.314.md) | 2026-04-28 | `dis` module — fixture 314 for https://docs.python.org/3/library/dis.html. Fu... |
| [v0.0.313](changelog/0.0.313.md) | 2026-04-28 | `compileall` module — fixture 313 for https://docs.python.org/3/library/compi... |
| [v0.0.312](changelog/0.0.312.md) | 2026-04-28 | `py_compile` module — fixture 312 for https://docs.python.org/3/library/py_co... |
| [v0.0.311](changelog/0.0.311.md) | 2026-04-28 | `pyclbr` module — fixture 311 for https://docs.python.org/3/library/pyclbr.ht... |
| [v0.0.310](changelog/0.0.310.md) | 2026-04-28 | `tabnanny` module — fixture 310 for https://docs.python.org/3/library/tabnann... |
| [v0.0.309](changelog/0.0.309.md) | 2026-04-28 | `tokenize` module — fixture 309 for https://docs.python.org/3/library/tokeniz... |
| [v0.0.308](changelog/0.0.308.md) | 2026-04-28 | `keyword` module — fixture 308 for https://docs.python.org/3/library/keyword.... |
| [v0.0.307](changelog/0.0.307.md) | 2026-04-28 | `token` module — fixture 307 for https://docs.python.org/3/library/token.html... |
| [v0.0.306](changelog/0.0.306.md) | 2026-04-28 | `symtable` module — fixture 306 for https://docs.python.org/3/library/symtabl... |
| [v0.0.305](changelog/0.0.305.md) | 2026-04-28 | `ast` module — fixture 305 for https://docs.python.org/3/library/ast.html. Fu... |
| [v0.0.304](changelog/0.0.304.md) | 2026-04-28 | `sys` path initialization attributes — fixture 304 for https://docs.python.or... |
| [v0.0.303](changelog/0.0.303.md) | 2026-04-28 | `importlib.metadata` comprehensive — fixture 303 for https://docs.python.org/... |
| [v0.0.302](changelog/0.0.302.md) | 2026-04-28 | `importlib.resources.abc` comprehensive — fixture 302 for https://docs.python... |
| [v0.0.301](changelog/0.0.301.md) | 2026-04-28 | `importlib.resources` comprehensive — fixture 301 for https://docs.python.org... |
| [v0.0.300](changelog/0.0.300.md) | 2026-04-28 | `importlib` family — first comprehensive fixture (300) for https://docs.pytho... |
| [v0.0.299](changelog/0.0.299.md) | 2026-04-28 | `runpy` — first fixture (299) for https://docs.python.org/3/library/runpy.htm... |
| [v0.0.298](changelog/0.0.298.md) | 2026-04-28 | `modulefinder` — first fixture (298) for https://docs.python.org/3/library/mo... |
| [v0.0.297](changelog/0.0.297.md) | 2026-04-28 | `pkgutil` — first fixture (297) for https://docs.python.org/3/library/pkgutil... |
| [v0.0.296](changelog/0.0.296.md) | 2026-04-28 | `zipimport` — first fixture (296) for https://docs.python.org/3/library/zipim... |
| [v0.0.295](changelog/0.0.295.md) | 2026-04-28 | `codeop` — first fixture (295) for https://docs.python.org/3/library/codeop.h... |
| [v0.0.294](changelog/0.0.294.md) | 2026-04-28 | `code` — first fixture (294) for https://docs.python.org/3/library/code.html.... |
| [v0.0.293](changelog/0.0.293.md) | 2026-04-28 | `site` — comprehensive deep-coverage fixture (293) for https://docs.python.or... |
| [v0.0.292](changelog/0.0.292.md) | 2026-04-28 | `annotationlib` — comprehensive deep-coverage fixture (292) for https://docs.... |
| [v0.0.291](changelog/0.0.291.md) | 2026-04-28 | `inspect` — comprehensive deep-coverage fixture (291) for https://docs.python... |
| [v0.0.290](changelog/0.0.290.md) | 2026-04-28 | `gc` — comprehensive deep-coverage fixture (290) for https://docs.python.org/... |
| [v0.0.289](changelog/0.0.289.md) | 2026-04-28 | `__future__` — full coverage of https://docs.python.org/3/library/__future__.... |
| [v0.0.288](changelog/0.0.288.md) | 2026-04-28 | `traceback` — comprehensive deep-coverage fixture (288) for https://docs.pyth... |
| [v0.0.287](changelog/0.0.287.md) | 2026-04-28 | `atexit` — comprehensive deep-coverage fixture (287) for https://docs.python.... |
| [v0.0.286](changelog/0.0.286.md) | 2026-04-28 | `abc` — full coverage of https://docs.python.org/3/library/abc.html. New modu... |
| [v0.0.285](changelog/0.0.285.md) | 2026-04-28 | `warnings` — comprehensive coverage of https://docs.python.org/3/library/warn... |
| [v0.0.284](changelog/0.0.284.md) | 2026-04-28 | `contextlib` — comprehensive coverage of https://docs.python.org/3/library/co... |
| [v0.0.283](changelog/0.0.283.md) | 2026-04-28 | `dataclasses` — comprehensive coverage of https://docs.python.org/3/library/d... |
| [v0.0.282](changelog/0.0.282.md) | 2026-04-28 | `__main__` — implements `import __main__` from https://docs.python.org/3/libr... |
| [v0.0.281](changelog/0.0.281.md) | 2026-04-28 | `builtins` — implements `import builtins` from https://docs.python.org/3/libr... |
| [v0.0.280](changelog/0.0.280.md) | 2026-04-27 | `sys.monitoring` — implements the Python 3.12+ monitoring API from https://do... |
| [v0.0.279](changelog/0.0.279.md) | 2026-04-27 | `site` — implements the site-customization module from https://docs.python.or... |
| [v0.0.278](changelog/0.0.278.md) | 2026-04-27 | `annotationlib` — implements the Python 3.14 annotation utilities from https:... |
| [v0.0.277](changelog/0.0.277.md) | 2026-04-27 | `inspect` — implements the inspection module from https://docs.python.org/3/l... |
| [v0.0.276](changelog/0.0.276.md) | 2026-04-27 | `sysconfig` — implements the build configuration module from https://docs.pyt... |
| [v0.0.275](changelog/0.0.275.md) | 2026-04-27 | `gc` — implements the garbage collector interface from https://docs.python.or... |
| [v0.0.274](changelog/0.0.274.md) | 2026-04-27 | `traceback` — implements the traceback utilities from https://docs.python.org... |
| [v0.0.273](changelog/0.0.273.md) | 2026-04-27 | `atexit` — implements the exit handler registration module from https://docs.... |
| [v0.0.272](changelog/0.0.272.md) | 2026-04-28 | `zipapp` — implements the Python executable zip archive module from https://d... |
| [v0.0.271](changelog/0.0.271.md) | 2026-04-27 | `venv` — implements the Python virtual environment creation module from https... |
| [v0.0.270](changelog/0.0.270.md) | 2026-04-27 | `ensurepip` — implements the pip bootstrap module from https://docs.python.or... |
| [v0.0.269](changelog/0.0.269.md) | 2026-04-27 | `tracemalloc` — implements the Python memory allocation tracer module from ht... |
| [v0.0.268](changelog/0.0.268.md) | 2026-04-27 | `trace` — implements the Python execution tracing and coverage module from ht... |
| [v0.0.267](changelog/0.0.267.md) | 2026-04-27 | `timeit` — implements the Python micro-benchmarking module from https://docs.... |
| [v0.0.266](changelog/0.0.266.md) | 2026-04-27 | `profile` / `cProfile` / `pstats` — implements the Python profiler suite from... |
| [v0.0.265](changelog/0.0.265.md) | 2026-04-27 | `pdb` — implements the Python interactive debugger module from https://docs.p... |
| [v0.0.264](changelog/0.0.264.md) | 2026-04-27 | `faulthandler` — implements the Python fault handler module from https://docs... |
| [v0.0.263](changelog/0.0.263.md) | 2026-04-27 | `bdb` — implements the Python debugger base infrastructure from https://docs.... |
| [v0.0.262](changelog/0.0.262.md) | 2026-04-27 | `sys` audit events — implements the audit hook mechanism from https://docs.py... |
| [v0.0.261](changelog/0.0.261.md) | 2026-04-27 | `unittest.mock` extended — fixture 261 covers the advanced APIs from the offi... |
| [v0.0.260](changelog/0.0.260.md) | 2026-04-27 | `unittest.mock` — implements the standard library unittest.mock module. |
| [v0.0.259](changelog/0.0.259.md) | 2026-04-27 | `dataclasses` — implements the standard library dataclasses module. |
| [v0.0.258](changelog/0.0.258.md) | 2026-04-27 | `contextlib` — implements the standard library context management utilities. |
| [v0.0.257](changelog/0.0.257.md) | 2026-04-27 | `unittest` — implements the standard library unittest framework. |
| [v0.0.256](changelog/0.0.256.md) | 2026-04-27 | `doctest` — implements the standard library `doctest` module. |
| [v0.0.255](changelog/0.0.255.md) | 2026-04-27 | Python Development Mode (`devmode`) — adds `sys.flags`, missing `sys` functio... |
| [v0.0.254](changelog/0.0.254.md) | 2026-04-27 | `pydoc` — runtime-sufficient implementation of the standard library pydoc mod... |
| [v0.0.253](changelog/0.0.253.md) | 2026-04-26 | `locale` — runtime-sufficient implementation of the standard library locale m... |
| [v0.0.252](changelog/0.0.252.md) | 2026-04-26 | `gettext` — full implementation of the standard library internationalization ... |
| [v0.0.251](changelog/0.0.251.md) | 2026-04-26 | `typing` — runtime-sufficient subset of the standard library typing module. |
| [v0.0.250](changelog/0.0.250.md) | 2026-04-26 | `colorsys` — full implementation of the standard library color-space conversi... |
| [v0.0.249](changelog/0.0.249.md) | 2026-04-26 | `wave` — full implementation of the standard library WAV audio module. Go has... |
| [v0.0.248](changelog/0.0.248.md) | 2026-04-26 | `ipaddress` — full implementation of the standard library IP address module. |
| [v0.0.247](changelog/0.0.247.md) | 2026-04-26 | `xmlrpc.server` — deeper API coverage building on fixture 245. |
| [v0.0.246](changelog/0.0.246.md) | 2026-04-26 | `xmlrpc.client` — deeper API coverage building on fixture 245. |
| [v0.0.245](changelog/0.0.245.md) | 2026-04-26 | `xmlrpc` package — namespace, `xmlrpc.client`, and `xmlrpc.server` submodules. |
| [v0.0.244](changelog/0.0.244.md) | 2026-04-26 | `http.cookiejar` module — cookie storage and policy classes. |
| [v0.0.243](changelog/0.0.243.md) | 2026-04-26 | `http.cookies` module — cookie manipulation classes, Morsel, BaseCookie, Simp... |
| [v0.0.242](changelog/0.0.242.md) | 2026-04-26 | `http.server` module — full class hierarchy, all attributes, complete HTTP st... |
| [v0.0.241](changelog/0.0.241.md) | 2026-04-26 | `socketserver` module — full class hierarchy, attributes, and stub server API. |
| [v0.0.240](changelog/0.0.240.md) | 2026-04-26 | `uuid` module — full RFC 4122 implementation: UUID class, SafeUUID enum, vari... |
| [v0.0.239](changelog/0.0.239.md) | 2026-04-26 | `smtplib` module — constants, full exception hierarchy rooted at OSError, `SM... |
| [v0.0.238](changelog/0.0.238.md) | 2026-04-26 | `imaplib` module — constants, nested exception hierarchy, utility functions, ... |
| [v0.0.237](changelog/0.0.237.md) | 2026-04-26 | `poplib` module — constants, `error_proto` exception, `POP3` class with full ... |
| [v0.0.236](changelog/0.0.236.md) | 2026-04-26 | `ftplib` module deep coverage — constants, exception hierarchy, parse functio... |
| [v0.0.235](changelog/0.0.235.md) | 2026-04-26 | `http.client` deep coverage — exception hierarchy, HTTPMessage, parse_headers... |
| [v0.0.234](changelog/0.0.234.md) | 2026-04-26 | `http` module deep coverage — `HTTPStatus` and `HTTPMethod` enums. |
| [v0.0.233](changelog/0.0.233.md) | 2026-04-26 | `urllib.robotparser` deep coverage — correct return types, `mtime`/`modified`... |
| [v0.0.232](changelog/0.0.232.md) | 2026-04-26 | `urllib.error` deep coverage — complete attribute set, response methods, and ... |
| [v0.0.231](changelog/0.0.231.md) | 2026-04-26 | `urllib.parse` deep coverage — complete `ParseResult`/`SplitResult` attribute... |
| [v0.0.230](changelog/0.0.230.md) | 2026-04-26 | `urllib.request` deep coverage — complete handler hierarchy, real `OpenerDire... |
| [v0.0.229](changelog/0.0.229.md) | 2026-04-26 | `urllib` package — urllib.parse additions plus three new sub-modules. |
| [v0.0.228](changelog/0.0.228.md) | 2026-04-26 | `wsgiref` package — all six sub-modules. |
| [v0.0.227](changelog/0.0.227.md) | 2026-04-26 | `webbrowser` module. |
| [v0.0.226](changelog/0.0.226.md) | 2026-04-26 | `pyexpat` module deep coverage per Python 3.14. |
| [v0.0.225](changelog/0.0.225.md) | 2026-04-26 | `xml.sax.xmlreader` deep coverage per Python 3.14. |
| [v0.0.224](changelog/0.0.224.md) | 2026-04-26 | `xml.sax.saxutils` deep coverage per Python 3.14. |
| [v0.0.223](changelog/0.0.223.md) | 2026-04-26 | `xml.sax.handler` deep coverage per Python 3.14. |
| [v0.0.222](changelog/0.0.222.md) | 2026-04-25 | Full `xml.sax` coverage per Python 3.14: the push-based SAX2 parser, all four... |
| [v0.0.221](changelog/0.0.221.md) | 2026-04-25 | This release completes `xml.dom.pulldom` with a working pull API, real DOM no... |
| [v0.0.220](changelog/0.0.220.md) | 2026-04-25 | This release completes the `xml.dom.minidom` serialization layer and wires up... |
| [v0.0.219](changelog/0.0.219.md) | 2026-04-25 | This release fills in the parts of `xml.dom` and `xml.dom.minidom` that were ... |
| [v0.0.218](changelog/0.0.218.md) | 2026-04-25 | First release of goipy. The version number tracks the number of merged PRs; v... |
| [v0.0.217](changelog/0.0.217.md) | 2026-04-25 | `xml.etree.ElementTree` extended: `QName`, `TreeBuilder`, `iterparse` with |
| [v0.0.216](changelog/0.0.216.md) | 2026-04-25 | Full `xml` package: `xml.etree.ElementTree` (parse, Element, SubElement, |
| [v0.0.215](changelog/0.0.215.md) | 2026-04-25 | `html.entities` extended to full CPython 3.14 spec: all 2231 named character |
| [v0.0.214](changelog/0.0.214.md) | 2026-04-25 | `html.parser` extended to full CPython 3.14 spec: `handle_starttag`, |
| [v0.0.213](changelog/0.0.213.md) | 2026-04-25 | `html`: `unescape`, `escape`. `html.parser`: `HTMLParser`. `html.entities`: |
| [v0.0.212](changelog/0.0.212.md) | 2026-04-25 | `quopri`: `encode`, `decode`, `encodestring`, `decodestring`. Quoted-printable |
| [v0.0.211](changelog/0.0.211.md) | 2026-04-25 | `binascii` extended to full CPython 3.14 spec: `b2a_qp`, `a2b_qp`, |
| [v0.0.210](changelog/0.0.210.md) | 2026-04-25 | `base64` extended to full CPython 3.14 spec: `encodebytes`, `decodebytes`, |
| [v0.0.209](changelog/0.0.209.md) | 2026-04-25 | `mimetypes`: `guess_type`, `guess_extension`, `guess_all_extensions`, |
| [v0.0.208](changelog/0.0.208.md) | 2026-04-25 | `mailbox`: `mbox`, `Maildir`, `MaildirMessage`, `mboxMessage`. `add`, |
| [v0.0.207](changelog/0.0.207.md) | 2026-04-25 | `json` extended to full CPython 3.14 feature set: `JSONDecodeError` with |
| [v0.0.206](changelog/0.0.206.md) | 2026-04-25 | `email` package: `email.message.Message`, MIME classes (`MIMEText`, |
| [v0.0.205](changelog/0.0.205.md) | 2026-04-25 | `mmap`: file-backed and anonymous mappings. `ACCESS_READ`, `ACCESS_WRITE`, |
| [v0.0.204](changelog/0.0.204.md) | 2026-04-25 | `signal`: `signal`, `getsignal`, `raise_signal`, `strsignal`, `valid_signals`, |
| [v0.0.203](changelog/0.0.203.md) | 2026-04-25 | `selectors`: `DefaultSelector`, `EpollSelector`, `KqueueSelector`, |
| [v0.0.202](changelog/0.0.202.md) | 2026-04-25 | `select`: `select`, `poll`, `epoll`, `kqueue`, `kevent`. `POLLIN`, `POLLOUT`, |
| [v0.0.201](changelog/0.0.201.md) | 2026-04-25 | `ssl` backed by Go `crypto/tls`: `SSLContext`, `wrap_socket`, `SSLSocket`, |
| [v0.0.200](changelog/0.0.200.md) | 2026-04-25 | `socket`: TCP/UDP backed by Go `net`. `socketpair` via `syscall.Socketpair`. |
| [v0.0.199](changelog/0.0.199.md) | 2026-04-25 | `asyncio` full CPython 3.14 API: `Task`, `Future`, `create_task`, |
| [v0.0.198](changelog/0.0.198.md) | 2026-04-25 | `_thread`: `start_new_thread`, `allocate_lock`, `get_ident`, `exit`, |
| [v0.0.197](changelog/0.0.197.md) | 2026-04-25 | `contextvars`: `ContextVar`, `Token`, `Context`, `copy_context`. Per-goroutine |
| [v0.0.196](changelog/0.0.196.md) | 2026-04-25 | `queue`: `Queue`, `LifoQueue`, `PriorityQueue`, `SimpleQueue`. `task_done`, |
| [v0.0.195](changelog/0.0.195.md) | 2026-04-25 | `sched`: `scheduler` backed by `container/heap`. `enter`, `enterabs`, |
| [v0.0.194](changelog/0.0.194.md) | 2026-04-25 | `subprocess`: `run`, `Popen`, `call`, `check_call`, `check_output`, |
| [v0.0.193](changelog/0.0.193.md) | 2026-04-25 | `concurrent.interpreters`: `Interpreter`, `Queue`. `exec`, `call`, |
| [v0.0.192](changelog/0.0.192.md) | 2026-04-25 | `concurrent.futures`: `ThreadPoolExecutor`, `ProcessPoolExecutor`, `Future`, |
| [v0.0.191](changelog/0.0.191.md) | 2026-04-25 | `multiprocessing.shared_memory`: `SharedMemory`, `ShareableList`. |
| [v0.0.190](changelog/0.0.190.md) | 2026-04-25 | `multiprocessing`: goroutine-backed `Process`, `Queue`, `Pipe`, `Pool` |
| [v0.0.189](changelog/0.0.189.md) | 2026-04-25 | `threading` rewritten with real goroutines. `Thread` maps to `goroutine`. |
| [v0.0.188](changelog/0.0.188.md) | 2026-04-24 | `cmd`: `Cmd` base class. `cmdloop`, `onecmd`, `parseline`, `emptyline`, |
| [v0.0.187](changelog/0.0.187.md) | 2026-04-24 | `curses.panel`: `new_panel`, `top_panel`, `bottom_panel`, `update_panels`, |
| [v0.0.186](changelog/0.0.186.md) | 2026-04-24 | `curses.ascii`: all 32 classification functions (`isalpha`, `isdigit`, |
| [v0.0.185](changelog/0.0.185.md) | 2026-04-24 | `curses.textpad`: `Textbox` with all editing commands (Ctrl-A through Ctrl-Z). |
| [v0.0.184](changelog/0.0.184.md) | 2026-04-24 | `curses`: `initscr`, `newwin`, `wrapper`. Window methods: `addstr`, `addch`, |
| [v0.0.183](changelog/0.0.183.md) | 2026-04-24 | `fileinput`: `input`, `filename`, `fileno`, `lineno`, `filelineno`, |
| [v0.0.182](changelog/0.0.182.md) | 2026-04-24 | `getpass`: `getpass`, `getuser`. Falls back to `input` when no TTY is present. |
| [v0.0.181](changelog/0.0.181.md) | 2026-04-24 | `optparse`: `OptionParser`, `add_option`, `parse_args`. `store`, `store_true`, |
| [v0.0.180](changelog/0.0.180.md) | 2026-04-24 | `argparse`: `ArgumentParser`, `add_argument`, `parse_args`, `parse_known_args`. |
| [v0.0.179](changelog/0.0.179.md) | 2026-04-24 | `ctypes`: `c_int`, `c_uint`, `c_long`, `c_ulong`, `c_char`, `c_char_p`, |
| [v0.0.178](changelog/0.0.178.md) | 2026-04-24 | `errno`: all standard POSIX error constants (`EPERM`, `ENOENT`, `EACCES`, |
| [v0.0.177](changelog/0.0.177.md) | 2026-04-24 | `platform`: `system`, `node`, `release`, `version`, `machine`, `processor`, |
| [v0.0.176](changelog/0.0.176.md) | 2026-04-24 | `logging.handlers`: `RotatingFileHandler`, `TimedRotatingFileHandler`, |
| [v0.0.175](changelog/0.0.175.md) | 2026-04-24 | `logging.config`: `dictConfig`, `fileConfig`, `listen`, `stopListening`. |
| [v0.0.174](changelog/0.0.174.md) | 2026-04-24 | `logging` full coverage: `Logger`, `Handler`, `Formatter`, `Filter`, |
| [v0.0.173](changelog/0.0.173.md) | 2026-04-24 | `time` full coverage: `struct_time`, `strftime`, `strptime`, `localtime`, |
| [v0.0.172](changelog/0.0.172.md) | 2026-04-24 | `io` full coverage: `BytesIO` all methods, `StringIO` all methods, |
| [v0.0.171](changelog/0.0.171.md) | 2026-04-24 | `os` full coverage: `walk`, `scandir` (with `DirEntry`), file-descriptor |
| [v0.0.170](changelog/0.0.170.md) | 2026-04-24 | `secrets` full coverage: correct error messages, `choice`, `randbits`, |
| [v0.0.169](changelog/0.0.169.md) | 2026-04-24 | `hmac` full coverage: SHA-3 digestmod, `block_size`, `copy`, callable |
| [v0.0.168](changelog/0.0.168.md) | 2026-04-24 | `hashlib` full coverage: `md5`, `sha1`, `sha224`, `sha256`, `sha384`, |
| [v0.0.167](changelog/0.0.167.md) | 2026-04-24 | `plistlib`: `loads`, `dumps`, `load`, `dump`. XML and binary plist formats. |
| [v0.0.166](changelog/0.0.166.md) | 2026-04-24 | `netrc`: `netrc`, `netrc_entry`. Parses `.netrc` format; `authenticators`, |
| [v0.0.165](changelog/0.0.165.md) | 2026-04-24 | `tomllib`: TOML v1.0 parser. Tables, arrays of tables, inline tables, |
| [v0.0.164](changelog/0.0.164.md) | 2026-04-24 | `configparser`: `ConfigParser`, `RawConfigParser`, `SafeConfigParser`, |
| [v0.0.163](changelog/0.0.163.md) | 2026-04-24 | `csv` full coverage: `reader`, `writer`, `DictReader`, `DictWriter`, |
| [v0.0.162](changelog/0.0.162.md) | 2026-04-24 | `zstd` (`compression.zstd`): `compress`, `decompress`, `ZstdCompressor`, |
| [v0.0.161](changelog/0.0.161.md) | 2026-04-24 | `tarfile`: `open`, `TarFile`, `TarInfo`. Read/write/append modes. |
| [v0.0.160](changelog/0.0.160.md) | 2026-04-24 | `zipfile`: `ZipFile`, `ZipInfo`, `is_zipfile`, `Path`. Read, write, append |
| [v0.0.159](changelog/0.0.159.md) | 2026-04-24 | `lzma`: `LZMAFile`, `LZMACompressor`, `LZMADecompressor`, `compress`, |
| [v0.0.158](changelog/0.0.158.md) | 2026-04-24 | `bz2`: `BZ2File`, `BZ2Compressor`, `BZ2Decompressor`, `compress`, `decompress`. |
| [v0.0.157](changelog/0.0.157.md) | 2026-04-24 | `gzip`: `open`, `GzipFile`, `compress`, `decompress`. Read/write modes, |
| [v0.0.156](changelog/0.0.156.md) | 2026-04-24 | `zlib` full coverage: `compress`, `decompress`, `compressobj`, `decompressobj`, |
| [v0.0.155](changelog/0.0.155.md) | 2026-04-24 | `sqlite3`: `connect`, `Connection`, `Cursor`, `Row`. DDL, DML, `executemany`, |
| [v0.0.154](changelog/0.0.154.md) | 2026-04-24 | `dbm`: `open`, `whichdb`, `ndbm`/`gdbm`/`dumb` backends via Go's |
| [v0.0.153](changelog/0.0.153.md) | 2026-04-24 | `marshal`: `dumps`, `loads`, `dump`, `load`. All CPython 3.14 marshal codes |
| [v0.0.152](changelog/0.0.152.md) | 2026-04-24 | `shelve`: `open`, `Shelf`, `DbfilenameShelf`, `BsdDbShelf`. Key-based |
| [v0.0.151](changelog/0.0.151.md) | 2026-04-24 | `copyreg`: `dispatch_table`, `pickle`, `constructor`. Used by `pickle` to |
| [v0.0.150](changelog/0.0.150.md) | 2026-04-24 | `pickle`: `dumps`, `loads`, `Pickler`, `Unpickler`, protocols 0-5, |
| [v0.0.149](changelog/0.0.149.md) | 2026-04-24 | `shutil`: `copy`, `copy2`, `copyfile`, `copyfileobj`, `copymode`, `copystat`, |
| [v0.0.148](changelog/0.0.148.md) | 2026-04-23 | `linecache`: `getline`, `getlines`, `checkcache`, `clearcache`, `lazycache`. |
| [v0.0.147](changelog/0.0.147.md) | 2026-04-23 | `fnmatch`: `fnmatch`, `fnmatchcase`, `filter`, `translate`. Character class |
| [v0.0.146](changelog/0.0.146.md) | 2026-04-23 | `glob`: `glob`, `iglob`, `escape`, `translate`. Recursive `**` pattern. |
| [v0.0.145](changelog/0.0.145.md) | 2026-04-23 | `tempfile`: `NamedTemporaryFile`, `TemporaryFile`, `SpooledTemporaryFile`, |
| [v0.0.144](changelog/0.0.144.md) | 2026-04-23 | `filecmp`: `cmp`, `cmpfiles`, `dircmp`. `os.chdir` added. |
| [v0.0.143](changelog/0.0.143.md) | 2026-04-23 | `stat`: all `S_IS*` predicates, `S_IMODE`, `S_IFMT`, `filemode`, `UF_*`, |
| [v0.0.142](changelog/0.0.142.md) | 2026-04-23 | `os.path` complete: `join`, `split`, `dirname`, `basename`, `exists`, |
| [v0.0.141](changelog/0.0.141.md) | 2026-04-23 | `pathlib`: `Path`, `PurePath`, `PurePosixPath`, `PureWindowsPath`. Full |
| [v0.0.140](changelog/0.0.140.md) | 2026-04-23 | `operator` complete: all comparison, arithmetic, bitwise, and item-access |
| [v0.0.139](changelog/0.0.139.md) | 2026-04-23 | `functools` complete: `reduce`, `partial`, `partialmethod`, `lru_cache`, |
| [v0.0.138](changelog/0.0.138.md) | 2026-04-23 | `itertools` complete: `batched`, `pairwise`, `takewhile`, `dropwhile`, |
| [v0.0.137](changelog/0.0.137.md) | 2026-04-23 | `statistics` rewritten to full Python 3.14 spec: `fmean`, `geometric_mean`, |
| [v0.0.136](changelog/0.0.136.md) | 2026-04-23 | `random`: `Random` class, `SystemRandom`, all distribution functions |
| [v0.0.135](changelog/0.0.135.md) | 2026-04-23 | `fractions`: `Fraction` with exact rational arithmetic, `limit_denominator`, |
| [v0.0.134](changelog/0.0.134.md) | 2026-04-23 | `decimal`: `Decimal`, `Context`, `getcontext`, `setcontext`, |
| [v0.0.133](changelog/0.0.133.md) | 2026-04-23 | `cmath`: `phase`, `polar`, `rect`, `exp`, `log`, `log10`, `sqrt`, `sin`, |
| [v0.0.132](changelog/0.0.132.md) | 2026-04-23 | `math` full coverage: `comb`, `perm`, `isqrt`, `ulp`, `nextafter`, `dist`, |
| [v0.0.131](changelog/0.0.131.md) | 2026-04-23 | `numbers`: abstract numeric tower: `Complex`, `Real`, `Rational`, `Integral`. |
| [v0.0.130](changelog/0.0.130.md) | 2026-04-23 | `graphlib`: `TopologicalSorter` with `prepare`, `get_ready`, `done`, |
| [v0.0.129](changelog/0.0.129.md) | 2026-04-23 | `enum`: `Enum`, `IntEnum`, `Flag`, `IntFlag`, `StrEnum`, `auto`, `unique`, |
| [v0.0.128](changelog/0.0.128.md) | 2026-04-23 | `reprlib`: `Repr` class with customizable limits; `aRepr` singleton; |
| [v0.0.127](changelog/0.0.127.md) | 2026-04-23 | `pprint`: `PrettyPrinter` with indent, width, depth, compact, sort_dicts; |
| [v0.0.126](changelog/0.0.126.md) | 2026-04-23 | `copy`: `copy`, `deepcopy` with cycle detection and `__copy__`/`__deepcopy__` |
| [v0.0.125](changelog/0.0.125.md) | 2026-04-22 | `types`: `SimpleNamespace`, `MappingProxyType`, `ModuleType`, `FunctionType`, |
| [v0.0.124](changelog/0.0.124.md) | 2026-04-22 | `weakref`: `ref`, `proxy`, `WeakValueDictionary`, `WeakKeyDictionary`, |
| [v0.0.123](changelog/0.0.123.md) | 2026-04-22 | `array`: typed homogeneous sequences. All type codes (`b`, `B`, `u`, `h`, |
| [v0.0.122](changelog/0.0.122.md) | 2026-04-22 | `bisect` full coverage: `bisect_left`, `bisect_right`, `insort_left`, |
| [v0.0.121](changelog/0.0.121.md) | 2026-04-22 | `heapq` full coverage: `heapify`, `heapreplace`, `heappushpop`, `merge`, |
| [v0.0.120](changelog/0.0.120.md) | 2026-04-22 | `collections.abc`: `isinstance` checks for `Mapping`, `Sequence`, `Set`, |
| [v0.0.119](changelog/0.0.119.md) | 2026-04-22 | `collections` full stdlib coverage: `deque` rotate/maxlen, `OrderedDict` |
| [v0.0.118](changelog/0.0.118.md) | 2026-04-22 | `calendar`: full coverage including `TextCalendar`, `HTMLCalendar`, |
| [v0.0.117](changelog/0.0.117.md) | 2026-04-22 | `zoneinfo`: `ZoneInfo`, `available_timezones`, `ZoneInfoNotFoundError`. Uses |
| [v0.0.116](changelog/0.0.116.md) | 2026-04-22 | `datetime`: `date`, `time`, `datetime`, `timedelta`, `timezone`, `tzinfo`. |
| [v0.0.115](changelog/0.0.115.md) | 2026-04-22 | `codecs`: `encode`, `decode`, `lookup`, `open`, incremental encoder/decoder, |
| [v0.0.114](changelog/0.0.114.md) | 2026-04-22 | `struct` full coverage: all format codes (`b`, `B`, `h`, `H`, `i`, `I`, `l`, |
| [v0.0.113](changelog/0.0.113.md) | 2026-04-22 | `rlcompleter`: `Completer` class with `complete` method, attribute and keyword |
| [v0.0.112](changelog/0.0.112.md) | 2026-04-22 | `readline`: `get_line_buffer`, `insert_text`, `redisplay`, `parse_and_bind`, |
| [v0.0.111](changelog/0.0.111.md) | 2026-04-22 | `stringprep` (RFC 3454): all 31 tables (`in_table_b1` through `in_table_d2`), |
| [v0.0.110](changelog/0.0.110.md) | 2026-04-22 | `unicodedata`: `lookup`, `name`, `category`, `bidirectional`, `combining`, |
| [v0.0.109](changelog/0.0.109.md) | 2026-04-22 | `textwrap` full coverage: `TextWrapper` constructor options, `fill`, `wrap`, |
| [v0.0.108](changelog/0.0.108.md) | 2026-04-22 | `difflib` full coverage: `SequenceMatcher` with `get_matching_blocks`, |
| [v0.0.107](changelog/0.0.107.md) | 2026-04-22 | `re` expanded: `Pattern.fullmatch`, `re.split`, `re.subn`, `re.escape`, |
| [v0.0.104](changelog/0.0.104.md) | 2026-04-22 | `re` module: full flag set (`IGNORECASE`, `MULTILINE`, `DOTALL`, `VERBOSE`, |
| [v0.0.103](changelog/0.0.103.md) | 2026-04-22 | `string.Formatter` extended tests. `string.Template` with custom delimiters. |
| [v0.0.100](changelog/0.0.100.md) | 2026-04-22 | `string` module: `Formatter` (full format spec mini-language), `Template` |
| [v0.0.099](changelog/0.0.099.md) | 2026-04-22 | Thread-safety fixtures: concurrent access to `list`, `dict`, `set`, |
| [v0.0.093](changelog/0.0.093.md) | 2026-04-22 | Complete built-in exception hierarchy per docs.python.org/3/library/exceptions. |
| [v0.0.091](changelog/0.0.091.md) | 2026-04-22 | Full `stdtypes` coverage from docs.python.org: all `str`, `int`, `float`, |
| [v0.0.086](changelog/0.0.086.md) | 2026-04-22 | All built-in constants: `False`, `True`, `None`, `NotImplemented`, |
| [v0.0.085](changelog/0.0.085.md) | 2026-04-22 | Seven missing CPython 3.14 opcodes implemented. `__init__` return-value check. |
| [v0.0.080](changelog/0.0.080.md) | 2026-04-22 | Missing built-ins added: `globals`, `locals`, `vars`, `object`, `input`, |
| [v0.0.078](changelog/0.0.078.md) | 2026-04-21 | Line table decoder for CPython 3.14 bytecode. `traceback`: format_exc, |
| [v0.0.076](changelog/0.0.076.md) | 2026-04-21 | `statistics`: mean, median, mode, stdev, variance, pstdev, NormalDist. |
| [v0.0.074](changelog/0.0.074.md) | 2026-04-21 | `difflib`: SequenceMatcher, `ndiff`, `unified_diff`, `context_diff`, HtmlDiff. |
| [v0.0.072](changelog/0.0.072.md) | 2026-04-21 | `binascii`: hexlify, unhexlify, b2a/a2b variants. `uuid`: UUID1/UUID4/UUID5. |
| [v0.0.070](changelog/0.0.070.md) | 2026-04-21 | `struct`: `pack`, `unpack`, `calcsize`, all format codes. `csv`: reader, |
| [v0.0.068](changelog/0.0.068.md) | 2026-04-21 | `io`: `StringIO`, `BytesIO`, `FileIO`. `hashlib`: MD5, SHA-1, SHA-256, |
| [v0.0.066](changelog/0.0.066.md) | 2026-04-21 | `json`: encoder, decoder, `loads`, `dumps`, indent/separators/sort_keys. |
| [v0.0.064](changelog/0.0.064.md) | 2026-04-21 | `math`: full function set including `comb`, `perm`, `isqrt`, `ulp`, `dist`. |
| [v0.0.062](changelog/0.0.062.md) | 2026-04-21 | `collections`: `deque`, `OrderedDict`, `Counter`, `ChainMap`, `namedtuple`, |
| [v0.0.060](changelog/0.0.060.md) | 2026-04-21 | `functools`: `reduce`, `partial`, `lru_cache`, `cached_property`, |
| [v0.0.058](changelog/0.0.058.md) | 2026-04-21 | Descriptor protocol and class-creation hooks: `__init_subclass__`, |
| [v0.0.056](changelog/0.0.056.md) | 2026-04-21 | Complete dunder surface: in-place operators (`__iadd__` etc.), reflected |
| [v0.0.054](changelog/0.0.054.md) | 2026-04-21 | `__dunder__` methods on user classes: `__repr__`, `__str__`, `__bool__`, |
| [v0.0.052](changelog/0.0.052.md) | 2026-04-21 | Long-tail builtins: `pow`, `format`, `ascii`, `slice`, `dir`, `delattr`, |
| [v0.0.050](changelog/0.0.050.md) | 2026-04-21 | `memoryview` over `bytes` and `bytearray`: slicing, format, shape, strides, |
| [v0.0.048](changelog/0.0.048.md) | 2026-04-21 | `bytearray`: mutable byte sequences, slice assignment, `append`, `extend`, |
| [v0.0.046](changelog/0.0.046.md) | 2026-04-21 | `frozenset` as a distinct hashable type: set operations, `issubset`, |
| [v0.0.044](changelog/0.0.044.md) | 2026-04-21 | `complex` type: arithmetic, `abs`, `conjugate`, polar/rectangular conversion. |
| [v0.0.042](changelog/0.0.042.md) | 2026-04-21 | `importlib`: `import_module`, `reload`, and `find_spec`. |
| [v0.0.040](changelog/0.0.040.md) | 2026-04-21 | Package imports: `__init__.py`, sub-packages, and relative `from pkg import x` |
| [v0.0.038](changelog/0.0.038.md) | 2026-04-21 | Filesystem imports: single-module `.pyc` loading, relative imports, and |
| [v0.0.035](changelog/0.0.035.md) | 2026-04-20 | `async def` / `await`, async generators, `async for`, and `async with`. A |
| [v0.0.032](changelog/0.0.032.md) | 2026-04-20 | Generator stress tests and deep generator chains. `yield from` across nested |
| [v0.0.030](changelog/0.0.030.md) | 2026-04-20 | Format spec polish: width, fill, align, sign, `z`, `#`, `0`, precision, and |
| [v0.0.028](changelog/0.0.028.md) | 2026-04-20 | Pattern matching edge cases: guard conditions, nested patterns, `as`-patterns, |
| [v0.0.026](changelog/0.0.026.md) | 2026-04-20 | Descriptor stress tests. The `__class__` cell is now plumbed correctly so |
| [v0.0.025](changelog/0.0.025.md) | 2026-04-20 | `with` statements, `super()`, descriptor protocol (`__get__`/`__set__`/ |
| [v0.0.010](changelog/0.0.010.md) | 2026-04-20 | Initial release. The eval loop covers the core CPython 3.14 instruction set. |

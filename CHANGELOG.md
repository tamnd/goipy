# Changelog

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

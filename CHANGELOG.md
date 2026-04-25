# Changelog

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

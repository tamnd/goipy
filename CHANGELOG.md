# Changelog

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

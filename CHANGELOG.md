# Changelog

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

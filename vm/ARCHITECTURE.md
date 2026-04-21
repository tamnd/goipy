# goipy VM Architecture

goipy is a pure-Go interpreter for CPython 3.14 bytecode. It reads a `.pyc`,
reconstructs the `code` object, and walks the bytecode. There is no JIT, no
specialisation, and no reference-counting. Everything runs on Go's garbage
collector.

This document describes the VM core. Standard-library modules (`vm/stdlib_*.go`)
are out of scope here — they're regular builtins registered in
`vm/asyncio.go:builtinModule`.

## Entry points

All program execution enters through `*Interp` in `vm/interp.go`:

- `Run(code *Code)` creates a top-level frame whose `globals` dict is fresh and
  whose `__builtins__` is the interpreter's builtin dict. It returns the final
  stack value (normally `None`).
- `RunPyc(code *Code)` is the same path with a plain error return — used by
  the test harness and `cmd/goipy`.
- `runFrame(f *Frame)` increments `callDepth`, guards against
  `MaxDepth` (default 500) via `RecursionError`, and delegates to
  `dispatch(f)`.

The interpreter caches one pointer per builtin exception class
(`typeErr`, `valueErr`, … — see `vm/interp.go:19-36`) so they can be raised
without a dict lookup.

## Frame layout

`vm.Frame` (`vm/frame.go`) is the single monolithic call-frame struct. It is
stack-allocated in Go's sense (one `NewFrame` per Python call) but always lives
on the Go heap — there is no fast-path for non-escaping frames.

```
Frame {
    Code           *Code        // compiled bytecode + metadata
    Globals        *Dict        // module globals (also Frame.Back.Globals if nested)
    Builtins       *Dict
    Locals         *Dict        // module/class bodies; nil for optimized functions
    Fast           []Object     // one slot per LocalsPlusNames entry
    Cells          []*Cell      // closure cells + free vars (NCells+NFrees)
    Stack          []Object     // value stack; capacity = Stacksize+8
    SP, IP         int
    Back           *Frame       // caller frame (unused at runtime; reserved)
    ExcInfo        *Exception   // most recent handled exception (for bare `raise`)
    Yielded        Object       // set by YIELD_VALUE, read by generator driver
}
```

`Fast` is flat — the CPython 3.11+ "localsplus" idea. Closure cells go into
`Cells`, not `Fast`; `MAKE_CELL idx` wraps `Fast[idx]` into a `*Cell` and moves
it into `Cells`.

Module-level and class-body frames are built with `Locals == Globals` (class
body frames get their own dict); `LOAD_NAME` / `STORE_NAME` consult it.
Function frames built by `callFunction` set `Locals = nil` when the code has
`CO_OPTIMIZED`, which forces LOAD_NAME to fall through to globals.

## Dispatch loop

`vm/dispatch.go:dispatch` is one big `for { switch opcode { ... } }`. Each
iteration:

1. Read the opcode at `code[f.IP]`, oparg at `code[f.IP+1]`.
2. OR the `EXTENDED_ARG` carry into oparg; clear the carry.
3. Remember `startIP` (used if we land on an exception handler).
4. Advance `IP` past opcode+arg, plus `2 * op.Cache[opcode]` for CPython's
   inline cache slots (we never read them — we just skip).
5. Execute the case.
6. On error, `goto handleErr`: walk the frame's exception table, trim the
   stack, push the exception, and resume at the handler's target.

The `op` package (`op/opcodes.go`) has all 266 opcode constants from CPython
3.14 plus a `Cache[op]` table for inline-cache widths. `dispatch.go` currently
implements 88 of them; the rest fall through to
`default → NotImplementedError`. The ~30 opcodes a real Python program
actually emits are all covered — the gap is specialised variants, the
monitoring/instrumentation opcodes, and niche match-statement helpers.

### What's covered

- Stack: `POP_TOP`, `PUSH_NULL`, `COPY`, `SWAP`
- Loads: `LOAD_CONST` (plus `IMMORTAL`/`MORTAL` variants), `LOAD_SMALL_INT`,
  `LOAD_NAME`, `LOAD_GLOBAL` (+ `BUILTIN`/`MODULE`), `LOAD_FAST` (+ `BORROW`,
  `CHECK`, `AND_CLEAR`), `LOAD_DEREF`, `LOAD_FROM_DICT_OR_{DEREF,GLOBALS}`,
  `LOAD_LOCALS`, `LOAD_COMMON_CONSTANT`, the paired `LOAD_FAST_LOAD_FAST`
  style fused ops, `MAKE_CELL`, `COPY_FREE_VARS`
- Stores/deletes: `STORE_NAME`, `STORE_GLOBAL`, `STORE_FAST` (+ fused),
  `STORE_DEREF`, `DELETE_*`, `STORE_ATTR`, `STORE_SUBSCR`, `STORE_SLICE`
- Binary/unary: `BINARY_OP` (dispatches through `vm/binops.go`), the
  type-specialised `BINARY_OP_*_INT/FLOAT/UNICODE` variants, `BINARY_SLICE`,
  `UNARY_NEGATIVE/NOT/INVERT`, `TO_BOOL` and family
- Compare: `COMPARE_OP` (+ typed variants), `IS_OP`, `CONTAINS_OP`
- Build: `BUILD_TUPLE/LIST/SET/MAP/SLICE/STRING`
- Container mutation: `LIST_APPEND/EXTEND`, `SET_ADD/UPDATE`, `MAP_ADD`,
  `DICT_UPDATE/MERGE`, `GET_LEN`
- Pattern matching: `MATCH_MAPPING`, `MATCH_SEQUENCE`, `MATCH_KEYS`,
  `MATCH_CLASS`
- Iter: `GET_ITER`, `FOR_ITER` (+ `LIST`/`TUPLE`/`RANGE`), `END_FOR`,
  `POP_ITER`
- Control: `JUMP_FORWARD`, `JUMP_BACKWARD` (+ `JIT` variants treated as
  plain jumps), `POP_JUMP_IF_{TRUE,FALSE,NONE,NOT_NONE}`, `JUMP_BACKWARD_NO_INTERRUPT`
- Unpacking: `UNPACK_SEQUENCE` (+ typed variants), `UNPACK_EX`
- Calls: `CALL` and every specialised variant, `CALL_KW`,
  `CALL_FUNCTION_EX`, `CALL_INTRINSIC_1/2`
- Return/yield: `RETURN_VALUE`, `RETURN_GENERATOR`, `YIELD_VALUE`, `SEND`,
  `END_SEND`, `GET_YIELD_FROM_ITER`, `CLEANUP_THROW`
- Functions & classes: `MAKE_FUNCTION`, `SET_FUNCTION_ATTRIBUTE`,
  `LOAD_BUILD_CLASS`
- Format: `CONVERT_VALUE`, `FORMAT_SIMPLE`, `FORMAT_WITH_SPEC`
- Import: `IMPORT_NAME`, `IMPORT_FROM`
- Exceptions: `RAISE_VARARGS`, `RERAISE`, `PUSH_EXC_INFO`, `POP_EXCEPT`,
  `CHECK_EXC_MATCH`, `WITH_EXCEPT_START`, `LOAD_SUPER_ATTR`,
  `LOAD_SPECIAL`, `CHECK_EG_MATCH` (non-group fallback)
- Async: `GET_AWAITABLE` (pass-through)

### What's missing or stubbed

- Exception groups — `CHECK_EG_MATCH` accepts `except*` syntax but treats it
  like `except` (no `.split()`, no `ExceptionGroup` unwrap).
- `__await__` on arbitrary objects — `GET_AWAITABLE` only handles
  `Generator`/`Iter` directly.
- `intrinsic2` — no cases implemented yet (no real program needed it).
- Monitoring/instrumentation — `INSTRUMENTED_*` opcodes are treated as NOP.
- `co_linetable` is stored on `Code` but never decoded; tracebacks have no
  line numbers. See gap spec.

## Exception handling

Exception handling is zero-cost (CPython 3.11+ style). The bytecode carries
a compact `co_exceptiontable` blob; `vm/exctable.go:decodeExceptionTable`
unpacks it into `excEntry{Start, End, Target, Depth, Lasti}` once per
`dispatch` call.

On error (`err != nil`):

1. If `err` is not `*object.Exception`, it's a Go-side bug → propagate out.
2. `findHandler(startIP)` searches linearly for the entry whose range
   covers the instruction that faulted.
3. The stack is trimmed to `handler.Depth` and the exception is pushed.
4. If `handler.Lasti`, the originating IP is pushed first (used by
   `RERAISE` to restore context).
5. `IP = handler.Target`, and dispatch resumes.

`*object.Exception` holds `Class`, `Args`, `Cause` (`raise X from Y`),
`Ctx` (implicit context), and `Msg`. Built-in classes are registered in
`vm/builtins.go`; `IsSubclass` walks the `Bases` chain directly — no C3
linearisation yet for exception types (it isn't needed because the
exception hierarchy is a tree).

Bare `raise` in an `except` block reads from `f.ExcInfo`, which is set by
`PUSH_EXC_INFO` and cleared by `POP_EXCEPT`.

## Object model

`object/object.go` defines a set of concrete Go types (`*Int`, `*Str`,
`*List`, `*Dict`, `*Tuple`, `*Function`, `*Class`, `*Instance`, …) used as
Python values. Dispatch is by Go type switch, not a vtable — adding an
operation means extending the relevant helper function.

- **Integers** are `*big.Int`. No int64 fast path yet.
- **Floats** are `float64`.
- **Strings** cache a `[]rune` on first indexed access (`vm/binops.go`
  subscript path).
- **Dicts** are insertion-ordered and have a fast path for string keys plus
  an equality chain for arbitrary hashable objects.
- **Classes** store `Bases []*Class` and a `Dict` of members. MRO is
  computed lazily but not C3-linearised — `classLookup` in `vm/dunder.go`
  does a depth-first walk of bases.
- **Instances** always use a `__dict__`; `__slots__` is not honoured.
- **Descriptors** work for functions (→ `BoundMethod`) and for any object
  exposing `__get__` via the dunder dispatch in `vm/dunder.go`.
- **Dunders** implemented (see `vm/dunder.go`): `__init__`, `__call__`,
  `__repr__`, `__str__`, `__eq__`, `__ne__`, `__lt__`, `__le__`, `__gt__`,
  `__ge__`, `__add__`/`__radd__`, `__mul__`/`__rmul__`, `__sub__`,
  `__truediv__`, `__getitem__`/`__setitem__`, `__iter__`, `__next__`,
  `__contains__`, `__len__`, `__bool__`, `__hash__`, `__int__`,
  `__float__`, `__index__`, `__invert__`, `__enter__`/`__exit__`,
  `__aenter__`/`__aexit__`, `__getattr__`/`__setattr__`.
- **Metaclasses** — not supported. `type(cls)` returns a placeholder; you
  can't override `__new__` on a metaclass to intercept class creation.

`vm/builtins.go` populates `Interp.Builtins` with the builtin functions
(`print`, `len`, `range`, …) and exception classes, then
`installDunderHooks` wires a few internal hooks (e.g. `Str_` for `*Instance`
so formatting finds `__str__`).

## Imports

`IMPORT_NAME` calls `importName(name, globals, fromlist, level)` in
`vm/imports.go`:

1. Resolve relative (`level > 0`) against `__package__`/`__name__`.
2. `loadChain` walks every prefix of the dotted name — `a`, then `a.b`,
   then `a.b.c` — calling `loadModule` for each so parents are populated.
3. `loadModule` consults `Interp.modules` (the `sys.modules` analogue),
   then `builtinModule(name)` for VM-provided stdlib, then every directory
   in `SearchPath` (or the parent package's `__path__`) for
   `leaf/__init__.pyc` or `leaf.pyc`.
4. `execModuleAs` runs the module body with `__name__`, `__package__`,
   `__builtins__`, and (for packages) `__path__` and `__file__` wired on
   the module's globals.

`IMPORT_FROM` just reads an attribute from the topmost module on the
stack, with a fallback that triggers another `loadModule` for
`pkg.submodule` accesses.

`sys.modules` is not exposed yet; `importlib.import_module` and
`importlib.reload` are the only programmatic entry points. There is no
finder/loader protocol, no `.py` source handling, no `*.so` / C extensions.

## Functions, generators, coroutines

`callObject` in `vm/call.go` is the universal callable dispatcher:

- `*BuiltinFunc` → call Go closure directly.
- `*Function` → `callFunction` builds a fresh frame, calls `bindArgs`
  (handles positional, keyword, `*args`, `**kwargs`, defaults, kw-only,
  pos-only), and either runs the frame or wraps it in a
  `*Generator` when any of `CO_GENERATOR | CO_COROUTINE |
  CO_ITERABLE_COROUTINE` is set.
- `*BoundMethod` → prepend `self`, recurse.
- `*Class` → if it's an exception subclass, construct `*Exception`
  directly; otherwise create `*Instance`, run `__init__`.
- `*Instance` → look up `__call__` dunder.

Generators live in `vm/generator.go`. `resumeGenerator` pushes the sent
value, runs the frame until either `YIELD_VALUE` (signalled by the
sentinel `errYielded` with the value stored on the frame) or
`RETURN_VALUE`. `close()` throws `GeneratorExit`; `.send()` / `.__next__`
resume. The `Generator` object also satisfies the iterator protocol so
`for x in gen` works.

Coroutines reuse the same machinery — `async def` compiles with
`CO_COROUTINE` instead of `CO_GENERATOR` and `driveCoroutine` (in
`vm/asyncio.go`) pumps it to completion synchronously. Real concurrency
isn't a goal.

## Test harness

`vm/interp_test.go:TestFixtures` sweeps `internal/testdata/*.pyc`. For each
fixture:

1. Load the `.pyc` through `marshal.LoadPyc`.
2. `Interp.Run` the top-level code.
3. Capture stdout, compare byte-for-byte to `<name>.expected.txt`.

Fixtures are numbered 01–76 and stored as paired `.py` / `.pyc` /
`.expected.txt`. The `.pyc` is compiled by the host's `python3.14`; the
expected output is captured by running the same `.py` through CPython.
Eighty-one fixtures cover:

- Core language (01–20): arithmetic, strings, lists, dicts, classes,
  exceptions, `with`, super, descriptors, f-strings, match/case, generators,
  walrus, comprehensions, unpacking.
- Stress & depth runs (21–35) for each core area.
- Async basics + stress (33–35).
- Imports (36–42): basic, transitive, packages, `importlib`.
- Advanced types (43–50): complex, frozenset, bytearray, memoryview.
- Builtins/dunders (51–58).
- Stdlib batches (59–76): each new module PR adds a basics fixture plus a
  ~80-scenario stress fixture.

## File map

| file | responsibility |
| --- | --- |
| `vm/interp.go` | `Interp` struct, `Run`, `runFrame`, recursion guard |
| `vm/frame.go` | `Frame`, push/pop helpers, stack & fast-locals layout |
| `vm/dispatch.go` | the main interpreter switch |
| `vm/ops.go` | non-op shared helpers (attribute access, getitem, getiter) |
| `vm/binops.go` | `BINARY_OP`, subscript, slice, numeric coercions |
| `vm/compare.go` | `COMPARE_OP` |
| `vm/call.go` | `callObject`, `callFunction`, `bindArgs`, intrinsics |
| `vm/generator.go` | generator/coroutine drive, `send`/`throw`/`close` |
| `vm/dunder.go` | dunder dispatch, descriptors, class lookup |
| `vm/exctable.go` | `co_exceptiontable` decoder + handler search |
| `vm/imports.go` | `__import__`, `loadChain`, `importlib` stub |
| `vm/builtins.go` | builtin functions, exception classes |
| `vm/asyncio.go` | `builtinModule` dispatch, tiny asyncio surface |
| `vm/stdlib_*.go` | individual standard-library module builders |
| `op/opcodes.go` | CPython 3.14 opcode constants + inline-cache widths |
| `object/*.go` | runtime object types, Repr/Str, dict/set internals |
| `marshal/` | `.pyc` reader (magic, header, code-object unmarshal) |
| `cmd/goipy/` | minimal CLI that runs a `.pyc` |

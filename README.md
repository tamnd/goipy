## goipy

goipy is a pure-Go interpreter for CPython 3.14 bytecode. Point it
at a `.pyc` produced by `python3.14 -m py_compile` and it runs the
program inside a Go process, on Go's garbage collector, as a plain
Go binary. Pure-Python scripts that avoid native extensions are the
sweet spot.

If you only want to try it, jump to [Quick start](#quick-start).
The rest of this file walks through how the interpreter is put
together and why each piece is shaped the way it is.

---

### Table of contents

1. [Motivation](#motivation)
2. [What goipy covers](#what-goipy-covers)
3. [Quick start](#quick-start)
4. [A worked example](#a-worked-example)
5. [Architecture overview](#architecture-overview)
6. [The pipeline, stage by stage](#the-pipeline-stage-by-stage)
   1. [Stage 1: loading a `.pyc`](#stage-1-loading-a-pyc)
   2. [Stage 2: unmarshalling the code object](#stage-2-unmarshalling-the-code-object)
   3. [Stage 3: the interpreter loop](#stage-3-the-interpreter-loop)
   4. [Stage 4: the built-in modules](#stage-4-the-built-in-modules)
7. [What works today](#what-works-today)
8. [Performance](#performance)
9. [Using goipy from Go](#using-goipy-from-go)
10. [Diagnostics and tracebacks](#diagnostics-and-tracebacks)
11. [Testing philosophy](#testing-philosophy)
12. [Design decisions and trade-offs](#design-decisions-and-trade-offs)
13. [What is missing and why](#what-is-missing-and-why)
14. [Extending goipy](#extending-goipy)
15. [Project layout](#project-layout)
16. [FAQ](#faq)
17. [A short history of the project](#a-short-history-of-the-project)
18. [License](#license)

---

### Motivation

Embedding Python inside a Go service usually means one of three
things. You ship CPython alongside the binary and trust that the
target box has the right shared libraries, headers, and Python
version. You link against `libpython` via cgo and inherit the
build-time complexity of two language toolchains plus the runtime
hazards of mixing two GCs. Or you run Python as a subprocess and
pay the cost of pipes for every call.

Each of those has a place. None of them gives you a single Go
binary that can execute user-supplied Python end-to-end on its own.

goipy fills that fourth spot. It loads compiled Python
(`.pyc`) and runs the bytecode inside a Go process with Go's
garbage collector, Go's stack, Go's concurrency primitives. The
price is peak speed and access to C extensions. In return you get a
portable artifact: one binary, one language toolchain, one kind of
crash dump. Python stays the guest; Go stays the host.

The audience this is built for: services that want user-pluggable
logic, sandboxes for auditable `.pyc` payloads, CLI tools that
accept small Python scripts as configuration, and benchmarks of "how
close can pure Go get to CPython on pure-Python workloads". If you
need numpy, reach for cgo and gopy; if you need peak speed, run
CPython. Otherwise, keep reading.

### What goipy covers

goipy is three things bolted together. A reader for CPython 3.14's
`.pyc` format. A Python object model in Go. An interpreter that
walks the bytecode through a classic switch-based dispatch loop.
Around that core sits a standard-library subset implemented as Go
functions registered as modules. The whole package is importable
from Go, so you can drive execution from your own programs.

The scope is deliberate. Compilation stays CPython's job; goipy
starts from the `.pyc`. Dispatch stays simple, switch-based, and
single-threaded. Native extensions (`PyObject *` and cgo) stay
outside the project. Stdlib coverage is a moving subset, driven by
what real scripts use.

Think of it the way you think of MicroPython or RustPython: a
from-scratch interpreter aimed at a specific use case, trading
ecosystem reach for a small, embeddable runtime.

### Quick start

Requirements: Go 1.26 and Python 3.14 on your `PATH`.

```bash
cat > hello.py <<'EOF'
name = "goipy"
print(f"hello from {name}")
print(sum(range(10)))
EOF

python3.14 -m py_compile hello.py
go run ./cmd/goipy __pycache__/hello.cpython-314.pyc
```

Expected output:

```
hello from goipy
45
```

The CLI has one job: take a path to a compiled `.pyc` and execute
it. Flags and config files are intentionally absent; the single
positional argument is the whole interface.

Run the test suite and benchmarks:

```bash
go test ./...        # unit tests against testdata/*.pyc
bench/run.sh         # full benchmark sweep against python3.14
```

### A worked example

Take a Python program that exercises a handful of features the
interpreter has to get right at once:

```python
# primes.py
from math import isqrt


def is_prime(n: int) -> bool:
    if n < 2:
        return False
    for i in range(2, isqrt(n) + 1):
        if n % i == 0:
            return False
    return True


primes = [n for n in range(2, 30) if is_prime(n)]
print(primes)
print(sum(primes))
```

Compile it and run the `.pyc` under goipy:

```bash
python3.14 -m py_compile primes.py
go run ./cmd/goipy __pycache__/primes.cpython-314.pyc
```

You get:

```
[2, 3, 5, 7, 11, 13, 17, 19, 23, 29]
129
```

Under the hood:

- `marshal` reads the `.pyc` header (magic, timestamp, hash, flags)
  and then unmarshals the top-level code object, which is a tree of
  `object.Code` values with constants, names, and the raw bytecode
  slice attached.
- `vm` creates a fresh module-level frame, binds `__name__` to
  `__main__`, resolves `from math import isqrt` via the built-in
  `math` module, and starts the dispatch loop.
- The list comprehension is its own code object; the interpreter
  recurses into it with a new frame, accumulates the list, and
  returns it back to the caller's operand stack.
- `print` is the built-in in `vm/builtins.go`; it writes to
  `Interp.Stdout`, which defaults to `os.Stdout` but can be
  redirected.

Every value on the stack is a `goipy/object` type. The resulting
binary links only Go packages; `go build` is the whole toolchain.

### Architecture overview

The project has five Go packages, a CLI, and a benchmark harness.

```
object/      Python object model: Int, Str, Dict, Class, Exception, ...
op/          Opcode table generated from CPython 3.14's opcode.py
marshal/     .pyc reader and marshal-format decoder
vm/          The interpreter: dispatch loop, call machinery, stdlib modules
cmd/goipy/   CLI entry point
bench/       Benchmark cases and CPython comparison runner
```

The data flow is linear:

```
hello.pyc
    |
    v   marshal.LoadPyc
*object.Code
    |
    v   vm.New().Run
Interpreter loop (vm/dispatch.go)
    |
    v   Interp.Stdout
program output
```

There is one interpreter struct, one frame struct, one dispatch
switch. `vm/interp.go` holds the interpreter; `vm/frame.go` holds
the call frame; `vm/dispatch.go` is the big switch that implements
every opcode.

### The pipeline, stage by stage

#### Stage 1: loading a `.pyc`

A `.pyc` is a short binary header followed by a marshalled code
object. The header is magic number plus flags plus either a
timestamp or a source hash, depending on how the file was compiled.
goipy only needs the magic to confirm it is looking at CPython 3.14
bytecode; every other header field is accepted and ignored. If the
magic differs, `marshal.LoadPyc` returns an error that names the
offending number.

#### Stage 2: unmarshalling the code object

CPython's marshal format is a tagged stream. Each value starts with
a type byte (sometimes with a REF flag on the top bit) and the
decoder dispatches on that byte: `TYPE_INT`, `TYPE_STRING`,
`TYPE_TUPLE`, `TYPE_CODE`, and about two dozen others. The format
includes a reference table so that repeated values (a string like
`"__init__"` appearing in dozens of code objects) are serialised
once and referenced thereafter.

Decoded values land in `object/`. A `*object.Code` has the bytecode
as a `[]byte`, constants as `[]object.Object`, variable-name arrays
(`varnames`, `names`, `cellvars`, `freevars`), the line-number
table, the exception table, and the handful of metadata fields that
CPython tracks (positional argument count, posonly count, kwonly
count, flags, the qualified name, and so on). Nested code objects
(functions, comprehensions, class bodies) are themselves
`*object.Code` sitting inside the enclosing one's consts.

#### Stage 3: the interpreter loop

`vm/dispatch.go` is the hot path. The dispatch loop owns:

- **The operand stack.** A `[]object.Object` on the frame. Every
  Python operation pushes and pops values here.
- **The instruction pointer.** An index into the frame's code bytes.
  Python 3.14 uses fixed 2-byte instructions with optional prefix
  `EXTENDED_ARG` words for oparg values larger than 8 bits.
- **Exception handling.** Rather than stack unwinding in Go, the
  dispatch loop reads the code object's exception table to find a
  handler PC for the current instruction. `RAISE_VARARGS` and the
  `RERAISE` opcodes walk that table; matched handlers get the
  exception on top of their stack.
- **Call machinery.** `CALL`, `CALL_FUNCTION_EX`, `CALL_KW`, and
  friends route into `vm/call.go`, which handles binding arguments
  to the callee's frame (positional, keyword, defaults, `*args`,
  `**kwargs`), creating a new `Frame`, and recursing into
  `runFrame`.
- **Attribute access.** `LOAD_ATTR` and `STORE_ATTR` go through
  `vm/dunder.go`, which implements the descriptor protocol and
  walks the MRO. goipy implements C3 linearisation in
  `object/class.go` so `super()` behaves.

Each function call is a fresh Go call frame. Recursion in the guest
Python translates directly to recursion in the host Go.
`Interp.MaxDepth` caps that at 500 by default and raises
`RecursionError` if the guest blows past it.

Generators, coroutines, and `async`/`await` *do* use a different
path: `vm/generator.go` builds a goroutine and a channel pair,
suspends the guest at `YIELD_VALUE`, and resumes on the next
`send`. This is the one place goipy uses real concurrency, and it
costs a goroutine per live generator.

#### Stage 4: the built-in modules

Modules that would be written in C in CPython (`math`, `re`,
`struct`, `json`, `hashlib`, `asyncio`, `sys`, and a long tail of
others) live in `vm/stdlib_*.go`. Each registers a handful of
builtin functions by wrapping Go stdlib calls and marshalling
arguments to and from `object.Object`. `vm/asyncio.go` is a
partial event loop that talks to Go's channels to drive
`asyncio.run` and friends.

User-level `.py` modules work too: `IMPORT_NAME` walks
`Interp.SearchPath` looking for a matching `.pyc`, decodes it via
`marshal`, runs it in a fresh module frame, caches the result in
`sys.modules`, and binds the module object into the caller's
globals.

### What works today

| Area                                         | Status  | Notes                                                      |
|----------------------------------------------|---------|------------------------------------------------------------|
| `int`, `float`, `bool`, `bytes`, `str`       | works   | `int` is arbitrary precision via `math/big`                |
| `list`, `tuple`, `dict`, `set`, `frozenset`  | works   | Dicts are insertion-ordered                                |
| Control flow, comprehensions                 | works   | `for`, `while`, `break`, `continue`                        |
| Functions, closures, decorators              | works   | Positional, keyword, `*args`, `**kwargs`, defaults         |
| Classes, MRO, `super()`                      | works   | C3 linearisation, memoised attribute lookup                |
| Exceptions, traceback                        | works   | `try/except/finally`, chained `raise`, exception groups    |
| Generators, `yield`, `yield from`            | works   | Exercised in the bench suite                               |
| `async`, `await`, `asyncio.run`              | works   | Driver covers `asyncio.sleep` and `gather`                 |
| `with` / `async with`                        | works   | `__enter__`/`__exit__`, `__aenter__`/`__aexit__`           |
| `match` statement                            | works   | Class, sequence, mapping patterns, guards                  |
| `import` of `.pyc` on disk                   | works   | `SearchPath` seeded from the entry script's directory      |
| Stdlib subset                                | partial | `sys`, `math`, `time`, `io`, `json`, `re`, `hashlib`, ~20  |
| C extensions                                 | out     | No `PyObject*` ABI; not planned                            |

The stdlib column is the one that moves most often. See
`vm/stdlib_*.go` for the current coverage, and the per-module test
files under `vm/*_test.go`.

### Performance

Captured 2026-04-21 on Apple M4, Go 1.26.2 against CPython 3.14.4.
All 24 cases produce byte-identical output against CPython.
Regenerate with `bench/run.sh`.

| Case               | CPython (ms) | goipy (ms) | ratio |
|--------------------|-------------:|-----------:|------:|
| arith_bigint       |        0.011 |      0.025 |  2.3x |
| arith_float        |       20.208 |     45.379 |  2.2x |
| arith_int          |       56.981 |    221.974 |  3.9x |
| async_drive        |        6.283 |     27.885 |  4.4x |
| call_closure       |       59.050 |    185.999 |  3.1x |
| call_kwargs        |       34.093 |    116.899 |  3.4x |
| call_plain         |       53.433 |    181.139 |  3.4x |
| class_attrs        |       18.541 |    112.161 |  6.0x |
| class_method       |       25.636 |     92.094 |  3.6x |
| class_mro          |       13.125 |     49.551 |  3.8x |
| coll_dict          |        6.499 |     31.325 |  4.8x |
| coll_list          |        8.052 |     31.856 |  4.0x |
| coll_set           |        4.946 |     28.851 |  5.8x |
| ctrl_fib_recursive |        8.543 |     36.991 |  4.3x |
| ctrl_for_range     |      131.172 |    439.422 |  3.3x |
| ctrl_while         |     8990.356 |   5897.821 |  0.7x |
| gen_pipeline       |       11.113 |     40.946 |  3.7x |
| gen_yield          |       15.991 |     56.486 |  3.5x |
| real_fib_iter      |        0.269 |      0.724 |  2.7x |
| real_nqueens       |       12.184 |     44.305 |  3.6x |
| real_wordcount     |       12.483 |     46.696 |  3.7x |
| str_concat         |        0.161 |      1.374 |  8.5x |
| str_format         |       16.720 |     38.545 |  2.3x |
| str_join_split     |        3.897 |     16.088 |  4.1x |

Across 23 of the 24 cases goipy lands between 2x and 6x slower than
CPython. `ctrl_while` is the one outlier where goipy wins: it
computes a bignum Fibonacci large enough that Go's `math/big`
outperforms CPython's `PyLong` once the values pass a few thousand
bits. Full methodology and per-case commentary live in
[`bench/RESULTS.md`](bench/RESULTS.md).

### Using goipy from Go

```go
package main

import (
    "fmt"
    "os"

    "github.com/tamnd/goipy/marshal"
    "github.com/tamnd/goipy/object"
    "github.com/tamnd/goipy/vm"
)

func main() {
    code, err := marshal.LoadPyc("hello.cpython-314.pyc")
    if err != nil {
        panic(err)
    }
    i := vm.New()
    i.Stdout = os.Stdout
    i.Argv = []string{"hello.pyc"}
    if _, err := i.Run(code); err != nil {
        if e, ok := err.(*object.Exception); ok {
            fmt.Fprint(os.Stderr, vm.FormatException(e))
        } else {
            fmt.Fprintln(os.Stderr, err)
        }
        os.Exit(1)
    }
}
```

A few points worth knowing:

- `vm.New()` is cheap. Reuse the interpreter across `Run` calls to
  keep the builtin dict and cached exception classes warm.
- Set `i.SearchPath` to a list of directories if the guest script
  does `import foo` and `foo.pyc` lives on disk. Built-in modules
  (`asyncio`, `math`, etc.) resolve before the search path is
  consulted.
- The return value of `Run` is the top-level value on the operand
  stack when the guest code returned, which is normally Python
  `None`. Discard it unless you need the module's globals, which
  you can read from `Run`'s result side-channel.
- `vm.FormatException` renders a Python-style traceback string,
  matching CPython's formatting for the common cases.
- `i.Stdout` and `i.Stderr` are `io.Writer`. Point them at
  `bytes.Buffer` for tests, or at a file, or at a logger.

### Diagnostics and tracebacks

Every uncaught exception in the guest surfaces as a Go error that
assertion-casts to `*object.Exception`. The exception carries a
linked list of frames, each annotated with the line number extracted
from the code object's line-number table (`co_linetable`), plus the
source file name.

`vm.FormatException` produces the familiar multi-line format:

```
Traceback (most recent call last):
  File "primes.py", line 9, in <module>
    print(primes[30])
          ~~~~~~~^^^^
IndexError: list index out of range
```

The `~` underline comes from CPython's position table. goipy decodes
that table in `object/linetable.go` and replays it in
`vm/traceback.go`. Custom formatters can walk the frame chain
directly: every `*object.Exception` exposes its traceback root via
`exc.Traceback`.

Plain Go errors (a panic in a stdlib shim, a malformed `.pyc`, an
out-of-memory condition) come back outside the `*object.Exception`
type. Treat those as bugs in goipy itself and report them.

### Testing philosophy

Three layers of tests live side by side.

**Unit tests** under `vm/*_test.go` and `object/*_test.go` target
individual opcodes, dunder hooks, object methods, and stdlib shims.
They tend to build a small `object.Code` by hand and run it, or
compile a one-line Python snippet at test time.

**Bytecode fixtures** under `testdata/` are full `.pyc` files with
their expected stdout. A test harness compiles each fixture with
`python3.14 -m py_compile`, runs it under goipy, and diffs the
output against the committed expected file. When Python's compiler
changes the shape of bytecode between patch versions, the fixtures
get regenerated; that regeneration is the point at which goipy's
decoder sees new opcodes.

**Benchmark snapshot** under `bench/`. The runner compiles every
case under `bench/cases/`, runs it under CPython and under goipy,
and checks that the outputs are byte-identical before timing.
Output divergence fails the whole sweep: a fast interpreter that
prints the wrong answer is a bug, not a speedup. Timings land in
`bench/RESULTS.md` for code review.

The combination means that opcode-level changes are caught by unit
tests, integration bugs are caught by the fixtures, and semantic
drift relative to CPython is caught by the benchmark harness
running the same code on both interpreters.

### Design decisions and trade-offs

**Bytecode input, not source.** goipy starts from `.pyc`. The
parser is a pile of work CPython already did, and CPython's output
is a stable, documented artifact per version. Pinning to 3.14
bytecode matches the project's scope.

**Pure Go.** The compiled artifact is a plain Go binary. That
scopes out `libpython`, C extensions, and the `PyObject *` ABI, and
along with them CPython's peephole optimiser and specialising
interpreter. The cost is peak performance. The payoff is that
`go build` is the whole toolchain.

**`math/big` for `int`.** Python ints are unbounded; Go's `int64`
tops out at 2^63. Using `math/big` on every `int` value costs
allocation on arithmetic and buys semantic parity with CPython on
integer workloads, which matters more than shaving nanoseconds off
the small-int case.

**Classic switch dispatch.** One big `switch` in `vm/dispatch.go`
covers every opcode. Go's current toolchain exposes neither
computed goto nor threaded dispatch; the switch is readable and
honest. A future revision can explore other forms once Go offers
the primitive.

**Goroutine per generator.** `yield` suspends by sending on a
channel; `next()` resumes by receiving. This maps Python's
resumable coroutines cleanly onto Go's scheduler. The cost is one
goroutine per live generator, which is fine at typical scale.

**Stdlib shims in Go.** Every built-in module forwards to the
closest Go standard-library equivalent. `hashlib.sha256` wraps
`crypto/sha256`; `re` wraps `regexp` with a small shim for
Python-flavoured syntax; `json` wraps `encoding/json` with type
adaptors. Reusing Go's stdlib keeps goipy's surface small and the
behaviour close to what Go already ships.

### What is missing and why

- **The C extension ABI.** Anything that ships a `PyObject *` stays
  outside goipy by design.
- **`async` beyond asyncio basics.** `asyncio.sleep`, `gather`, and
  `run` work today. `asyncio.subprocess`, `asyncio.streams`, and
  the network stack are still on the to-do list.
- **The tail of the stdlib.** Around 20 modules are implemented;
  CPython ships hundreds. Each new module is additive and fairly
  mechanical; prioritisation follows user demand.
- **Unicode edge cases.** Go's `unicode` and `strings` packages
  cover the common paths. Python has corners around normalisation
  and case-folding that still want systematic verification.
- **Native performance.** goipy is a straightforward switch-based
  interpreter; CPython 3.14 is a specialising adaptive interpreter
  tuned over three decades. Closing the gap beyond a small constant
  factor would take real design work.
- **Signal handling, threads, and the GIL.** goipy runs the guest
  single-threaded by default. Real `threading.Thread` semantics
  clash with Go's scheduler in subtle ways that need design work
  before landing.

Treat each item as a candidate future direction rather than a
commitment.

### Extending goipy

Adding a new opcode:

1. Check `op/` to make sure the opcode number is correct for Python
   3.14. `op/` is generated from CPython's `opcode.py`, so mismatch
   means the source was out of date when the table was produced.
2. Add the case to the dispatch switch in `vm/dispatch.go`. Keep
   the case body focused; complex helpers go into separate
   functions in the same file or an adjacent one.
3. Add a unit test in `vm/interp_test.go` that compiles a one-line
   Python snippet exercising the opcode and checks the result.
4. Run the benchmark suite. The timing pages are a safety net
   against accidentally pessimising an operator.

Adding a new stdlib module:

1. Create `vm/stdlib_<name>.go` and register a module object via
   `i.addStdlibModule(...)`.
2. Wrap the Go stdlib function you want to expose, converting
   arguments to and from `object.Object`.
3. Add a test in `vm/stdlib_<name>_test.go` that drives the module
   from a compiled Python snippet.
4. If the module needs new Python-level behaviour, not a direct Go
   forward, add that logic to `vm/` and keep the module file thin.

Every change that touches the dispatch loop or a high-traffic
object method should include benchmark numbers from the full
`bench/RESULTS.md` sweep. A single microbenchmark in isolation is
not enough to land a VM change.

### Project layout

```
cmd/goipy/           CLI: load a .pyc and run it.
marshal/             .pyc header reader and marshal-format decoder.
op/                  Opcode table, regenerated from CPython's opcode.py.
object/              Python object model: Int, Float, Str, List, Dict, Class,
                     Exception, Module, Code, Frame, Generator, descriptors.
vm/interp.go         Interp struct; Run, RunPyc, module frame setup.
vm/frame.go          Call frame layout.
vm/dispatch.go       The opcode dispatch switch.
vm/call.go           Function-call machinery: argument binding, defaults,
                     *args/**kwargs.
vm/dunder.go         Attribute access, MRO walk, descriptor protocol.
vm/generator.go      Generators and coroutines (goroutine + channel based).
vm/exctable.go       Exception-table decoding and handler lookup.
vm/traceback.go      Python-style traceback formatting.
vm/stdlib_*.go       Built-in modules (math, re, hashlib, asyncio, ...).
vm/ARCHITECTURE.md   Low-level design notes for the VM.
testdata/            .pyc fixtures with expected stdout.
bench/               Benchmark cases, CPython comparison, results table.
```

### FAQ

**Why `.pyc` as input, not `.py`?**

Parsing is a distinct project with its own failure modes. CPython
already parses Python correctly and exposes the result as a stable
bytecode format. Accepting `.pyc` lets goipy focus on execution.

**Can I run scripts without compiling them first?**

Through the CLI, you feed it a `.pyc`. Programmatically, you can
shell out to `python3.14 -m py_compile` from Go, or build your own
helper that invokes CPython in a subprocess and hands the resulting
`.pyc` bytes to `marshal.LoadPyc`.

**How does goipy handle threads?**

It runs single-threaded. `threading.Thread` is out of scope for
now; guests that depend on real OS threads should run under
CPython.

**How does `async` work without an event loop?**

`vm/asyncio.go` implements a minimal event loop on top of Go
channels. `asyncio.run` drives it; `asyncio.sleep` and `gather`
cooperate. The surface is small on purpose.

**Why is `arith_int` 3.9x slower than CPython?**

`math/big` allocates for almost every operation, whereas CPython
keeps a cache of small ints and avoids allocation on the hot path.
Using tagged pointers or a small-int fast path would close most of
the gap and is a natural future direction.

**What Python version do I need?**

CPython 3.14. The opcode table is pinned to 3.14, and the `.pyc`
magic check enforces it.

**Does `import` of `.py` files work?**

Indirectly. Compile each module to `.pyc` first and place it under
a directory listed in `Interp.SearchPath`. On-the-fly compilation
is out of scope; pre-compile is the workflow.

**Is this secure enough to run untrusted code?**

goipy has a smaller attack surface than CPython: no C extensions,
no JIT. It is unaudited, and it has had no hardening pass against
adversarial bytecode. Treat hostile `.pyc` as capable of exhausting
memory or CPU even while staying within the guest.

### A short history of the project

goipy started as a learning exercise: read CPython's marshal format
and pretty-print a `.pyc`. That grew into a partial decoder, then a
handful of opcodes, then an interpreter that could run straight-line
arithmetic. Once control flow and function calls worked, the project
picked up classes, then exceptions, then generators, then
`async`/`await`, adding each layer once the lower one was stable.

Standard-library modules came in waves (math/heapq/bisect, then
json/re, then hashlib/base64, then a long tail), each landed with
tests and fixtures. The most recent rounds of work have been about
closing gaps in behaviour that only show up under real programs:
line-table decoding so tracebacks read right, exception groups,
`match` pattern matching, and a benchmark harness that compares
goipy's output against CPython on a varied suite.

The project is now past the "does it run?" stage and into the "how
does it behave on real programs?" stage. The direction from here is
more stdlib coverage and a closer look at the performance gap.

### License

MIT. `.pyc` input files remain under the PSF license that covers
CPython bytecode output.

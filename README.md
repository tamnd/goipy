<h1 align="center">goipy</h1>

<p align="center">
  <b>A pure-Go interpreter for CPython 3.14 bytecode.</b><br>
  <sub>Ship one binary. Run user Python inside your Go process, on Go's GC, with Go's concurrency.</sub>
</p>

<p align="center">
  <a href="#quick-start">Quick start</a> ·
  <a href="#embed-in-go">Embed</a> ·
  <a href="#what-works">Features</a> ·
  <a href="#performance">Performance</a> ·
  <a href="ARCHITECTURE.md">Architecture</a> ·
  <a href="#faq">FAQ</a>
</p>

---

```sh
python3.14 -m py_compile hello.py
go run ./cmd/goipy __pycache__/hello.cpython-314.pyc
```

Single Go binary. No cgo. No shared libraries. No Python on the
target box. Feed it a `.pyc`, get Python output.

## Why

Embedding Python in a Go service usually means shipping CPython
alongside the binary, linking `libpython` through cgo, or driving
Python as a subprocess. Each works; none gives you a single Go
binary that executes user-supplied Python end-to-end.

goipy fills that gap. Compilation stays CPython's job; goipy
starts from the `.pyc` and runs the bytecode through a switch-based
dispatch loop with Python objects modelled as Go values. The price
is peak speed and C extensions. The payoff is a portable artifact:
one binary, one toolchain, one kind of crash dump.

Good fits: services with user-pluggable logic, auditable `.pyc`
payloads, CLI tools that accept small Python scripts as config.

## Quick start

Requires **Go 1.26** and **Python 3.14**.

```sh
cat > hello.py <<'EOF'
name = "goipy"
print(f"hello from {name}")
print(sum(range(10)))
EOF

python3.14 -m py_compile hello.py
go run ./cmd/goipy __pycache__/hello.cpython-314.pyc
```

Output:

```
hello from goipy
45
```

Tests and benchmarks:

```sh
go test ./...        # unit + fixture tests
bench/run.sh         # diff vs CPython on 24 cases, then time each
```

The benchmark runner fails the sweep if any case produces output
that differs from CPython. Fast is worthless if the answer is wrong.

## Embed in Go

```go
import (
    "os"

    "github.com/tamnd/goipy/marshal"
    "github.com/tamnd/goipy/object"
    "github.com/tamnd/goipy/vm"
)

func runPyc(path string) error {
    code, err := marshal.LoadPyc(path)
    if err != nil {
        return err
    }
    i := vm.New()
    i.Stdout = os.Stdout
    i.SearchPath = []string{"./pymodules"} // for IMPORT_NAME
    _, err = i.Run(code)
    if e, ok := err.(*object.Exception); ok {
        os.Stderr.WriteString(vm.FormatException(e))
    }
    return err
}
```

- `vm.New()` is cheap; reuse the interpreter across runs to keep
  the builtin dict warm.
- `i.Stdout` / `i.Stderr` are `io.Writer`. Point them at a buffer
  for tests, or at a logger.
- `*object.Exception` carries a full Python traceback. `vm.FormatException`
  renders it in the familiar multi-line format, position underlines
  and all.

## What works

| Area                                         | Status  | Notes                                                      |
|----------------------------------------------|---------|------------------------------------------------------------|
| `int`, `float`, `bool`, `bytes`, `str`       | works   | `int` is arbitrary precision via `math/big`                |
| `list`, `tuple`, `dict`, `set`, `frozenset`  | works   | Dicts are insertion-ordered                                |
| Control flow, comprehensions                 | works   | `for`, `while`, `break`, `continue`                        |
| Functions, closures, decorators              | works   | Positional, keyword, `*args`, `**kwargs`, defaults         |
| Classes, MRO, `super()`                      | works   | C3 linearisation                                           |
| Exceptions, tracebacks                       | works   | `try/except/finally`, chained `raise`, exception groups    |
| Generators, `yield`, `yield from`            | works   | Goroutine + channel per generator                          |
| `async` / `await` / `asyncio.run`            | works   | Covers `asyncio.sleep`, `gather`                           |
| `with` / `async with`                        | works   | `__enter__`/`__exit__`, `__aenter__`/`__aexit__`           |
| `match` statement                            | works   | Class, sequence, mapping patterns, guards                  |
| `import` of `.pyc` on disk                   | works   | `Interp.SearchPath` for module resolution                  |
| Stdlib subset                                | partial | ~20 modules: `sys`, `math`, `time`, `io`, `json`, `re`, `hashlib`, &hellip; |
| C extensions                                 | out     | No `PyObject*` ABI; out of scope                           |

Stdlib coverage moves the most often. See `vm/stdlib_*.go` for the
current set.

## Performance

Captured 2026-04-21 on Apple M4, Go 1.26.2 vs CPython 3.14.4. All
24 cases produce byte-identical output against CPython.

| Case               | CPython (ms) | goipy (ms) | ratio |
|--------------------|-------------:|-----------:|------:|
| arith_bigint       |        0.011 |      0.025 |  2.3x |
| arith_float        |       20.208 |     45.379 |  2.2x |
| arith_int          |       56.981 |    221.974 |  3.9x |
| call_plain         |       53.433 |    181.139 |  3.4x |
| class_attrs        |       18.541 |    112.161 |  6.0x |
| ctrl_for_range     |      131.172 |    439.422 |  3.3x |
| ctrl_while         |     8990.356 |   5897.821 |  0.7x |
| gen_yield          |       15.991 |     56.486 |  3.5x |
| real_nqueens       |       12.184 |     44.305 |  3.6x |
| real_wordcount     |       12.483 |     46.696 |  3.7x |

23 of 24 cases land between **2x and 6x** slower than CPython.
`ctrl_while` wins because it runs a bignum Fibonacci large enough
that Go's `math/big` overtakes CPython's `PyLong`. Full table and
methodology: [`bench/RESULTS.md`](bench/RESULTS.md).

## Project layout

```
cmd/goipy/      CLI: load a .pyc and run it
marshal/        .pyc header + marshal-format decoder
op/             Opcode table, regenerated from CPython's opcode.py
object/         Python object model: Int, Str, Dict, Class, Exception, ...
vm/interp.go    Interp struct; Run, RunPyc, module frame setup
vm/dispatch.go  The opcode dispatch switch
vm/call.go      Argument binding, *args, **kwargs, defaults
vm/generator.go Generators and coroutines (goroutine + channel)
vm/stdlib_*.go  Built-in modules (math, re, hashlib, asyncio, ...)
testdata/       .pyc fixtures with expected stdout
bench/          Benchmark cases and CPython comparison runner
```

## FAQ

**Why `.pyc` input?** CPython already parses Python correctly and
emits a stable per-version bytecode. Starting from `.pyc` lets
goipy focus on execution.

**Can I feed it a `.py`?** Compile first: `python3.14 -m py_compile`.
On-the-fly compilation is out of scope.

**Threads?** Single-threaded by default. `threading.Thread` is out
of scope; workloads that need real OS threads belong under CPython.

**How does async work?** `vm/asyncio.go` runs a minimal event loop
on Go channels. `asyncio.run`, `sleep`, and `gather` cooperate.

**Safe for untrusted code?** Smaller attack surface than CPython
(no C extensions, no JIT), but unaudited. Treat hostile `.pyc` as
capable of exhausting memory or CPU even while staying within the
guest.

**Why is `arith_int` 3.9x slower?** `math/big` allocates on every
op; CPython caches small ints. A tagged-pointer small-int path
would close most of that gap and is a natural future direction.

More in [ARCHITECTURE.md &sect; FAQ](ARCHITECTURE.md#faq).

## Learn more

- [ARCHITECTURE.md](ARCHITECTURE.md) &mdash; pipeline stages, object
  model, dispatch internals, stdlib shim strategy, extension guide.
- [`bench/RESULTS.md`](bench/RESULTS.md) &mdash; per-case commentary
  and methodology.
- [`testdata/`](testdata/) &mdash; `.pyc` fixtures with expected stdout.

## License

MIT. `.pyc` input files remain under the PSF license that covers
CPython bytecode output.

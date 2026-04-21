# goipy

A pure-Go interpreter that runs CPython 3.14 `.pyc` files. No cgo, no bundled libpython, no JIT.

goipy loads a `.pyc` produced by `python3.14 -m py_compile` and executes its bytecode in a Go process. The scope is deliberate: CPython 3.14 bytecode only, no parser, no C extensions. If the code runs with `python3.14` and does not reach into numpy or other native modules, it is a candidate for goipy.

## Quick start

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

## Why goipy exists

The driving motivation is deployment: ship a single Go binary that runs user-supplied Python without bundling CPython and without cgo. That forecloses `pip install numpy`, but it also forecloses the Python-version-skew and "we needed libpython3.14.so.1.0 on the box" class of incident. goipy trades peak speed for that simplicity. If you are running heavy numeric code or need the full stdlib, reach for cgo + gopy or run CPython as a subprocess. If you want user-pluggable logic inside a Go service, light scripting, or a sandbox for `.pyc` you can audit, goipy is the project.

## What works today

| Area | Status | Notes |
|---|---|---|
| `int`, `float`, `bool`, `bytes`, `str` | works | `int` is arbitrary precision via `math/big` |
| `list`, `tuple`, `dict`, `set`, `frozenset` | works | ordered dict, insertion-stable |
| Control flow, comprehensions | works | `for`, `while`, `break`, `continue` |
| Functions, closures, decorators | works | positional, keyword, `*args`, `**kwargs`, defaults |
| Classes, MRO, `super()` | works | C3 linearisation, memoised attribute lookup |
| Exceptions, traceback | works | `try/except/finally`, chained `raise`, exception groups |
| Generators, `yield`, `yield from` | works | exercised in the bench suite |
| `async`, `await`, `asyncio.run` | works | driver covers `asyncio.sleep` and `gather` |
| `with` / `async with` | works | `__enter__` / `__exit__`, `__aenter__` / `__aexit__` |
| `match` statement | works | class, sequence, mapping patterns, guards |
| `import` of `.pyc` on disk | works | `SearchPath` seeded from the entry script's directory |
| Stdlib subset | partial | `sys`, `math`, `time`, `io`, `json`, `re`, `hashlib`, `heapq`, `collections`, `random`, `struct`, `base64`, `asyncio`, and ~20 others under `vm/stdlib_*.go` |
| C extensions | no | no `PyObject*` ABI, no plans |

## Performance

Captured 2026-04-21 on Apple M4, Go 1.26.2 vs CPython 3.14.4. All 24 cases produce byte-identical output. Regenerate with `bench/run.sh`.

| Case | CPython (ms) | goipy (ms) | ratio |
|---|---:|---:|---:|
| arith_bigint | 0.011 | 0.025 | 2.3x |
| arith_float | 20.208 | 45.379 | 2.2x |
| arith_int | 56.981 | 221.974 | 3.9x |
| async_drive | 6.283 | 27.885 | 4.4x |
| call_closure | 59.050 | 185.999 | 3.1x |
| call_kwargs | 34.093 | 116.899 | 3.4x |
| call_plain | 53.433 | 181.139 | 3.4x |
| class_attrs | 18.541 | 112.161 | 6.0x |
| class_method | 25.636 | 92.094 | 3.6x |
| class_mro | 13.125 | 49.551 | 3.8x |
| coll_dict | 6.499 | 31.325 | 4.8x |
| coll_list | 8.052 | 31.856 | 4.0x |
| coll_set | 4.946 | 28.851 | 5.8x |
| ctrl_fib_recursive | 8.543 | 36.991 | 4.3x |
| ctrl_for_range | 131.172 | 439.422 | 3.3x |
| ctrl_while | 8990.356 | 5897.821 | 0.7x |
| gen_pipeline | 11.113 | 40.946 | 3.7x |
| gen_yield | 15.991 | 56.486 | 3.5x |
| real_fib_iter | 0.269 | 0.724 | 2.7x |
| real_nqueens | 12.184 | 44.305 | 3.6x |
| real_wordcount | 12.483 | 46.696 | 3.7x |
| str_concat | 0.161 | 1.374 | 8.5x |
| str_format | 16.720 | 38.545 | 2.3x |
| str_join_split | 3.897 | 16.088 | 4.1x |

goipy runs 2.2x to 6x slower than CPython on every case except `ctrl_while`, which is a bignum Fibonacci where Go's `math/big` outperforms `PyLong` once the integers grow past a few thousand bits. Full methodology and per-case observations live in [`bench/RESULTS.md`](bench/RESULTS.md).

## Using goipy from Go

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

`vm.New()` is cheap; reuse the interpreter across `Run` calls to keep the builtin dict and cached exception classes warm. Set `i.SearchPath` to a list of directories if the script does `import foo` and `foo.pyc` lives on disk. The return value of `Run` is the module object for the executed code; discard it unless you need to pull globals back out.

## Project layout

- `object/` is the Python object model (`Int`, `Str`, `Dict`, `Class`, `Exception`, ...).
- `op/` is the opcode table generated from CPython 3.14's `opcode.py`.
- `marshal/` reads `.pyc` files and decodes the 3.14 marshal format into `object.Code`.
- `vm/` is the interpreter. `dispatch.go` holds the main switch, `call.go` handles function invocation, `stdlib_*.go` implement the built-in modules.
- `cmd/goipy/` is the CLI entry point.
- `bench/` is the benchmark harness and results.

## Contributing

```bash
go test ./...        # unit tests
bench/run.sh         # full benchmark sweep against python3.14
```

Opcode handling lives in `vm/dispatch.go`. Adding an opcode means extending that switch and, for new specialised forms, regenerating `op/` from the CPython source. Performance-motivated commits should include before-and-after numbers from the full `bench/RESULTS.md` sweep; a single microbenchmark in isolation is not enough to land a VM change. Design notes and historical decisions live under `notes/Spec/0900/`.

## License

MIT. `.pyc` input files remain under the PSF license that covers CPython bytecode output.

# bench

Speed comparison between `goipy` and CPython 3.14 on a fixed set of
self-contained Python programs.

## Running

```sh
bench/run.sh          # 3 runs per interpreter (default), prints Markdown table
RUNS=5 bench/run.sh   # more samples
PYTHON=python3.14 bench/run.sh  # override interpreter path
```

The script:

1. Rebuilds `goipy` from source into `bench/.bin/goipy`.
2. For each `bench/cases/*.py`, compiles it to `.pyc` with `python3.14` and
   runs the `.pyc` under both interpreters.
3. Diffs the `CHECKSUM:` line byte-for-byte between the two interpreters
   before publishing a time. A mismatched case prints `⚠ mismatch` and its
   ratio is suppressed — it is unsafe to compare workloads when the
   results disagree.
4. Samples `TIME_MS:` `$RUNS` times per interpreter and reports the median.

## Case shape

Every case follows the same template so the harness can treat them
uniformly:

```python
import time

def main():
    ...
    return checksum  # deterministic int/str

t0 = time.perf_counter()
cs = main()
t1 = time.perf_counter()
print(f"CHECKSUM:{cs}")
print(f"TIME_MS:{(t1 - t0) * 1000:.3f}")
```

`TIME_MS` is the in-process wall time around `main()` only — interpreter
startup and import cost are excluded from the reported figure. Startup
cost is a separate dimension; it is out of scope for this suite.

## What the cases exercise

| Group | Cases | What it measures |
|---|---|---|
| arith | `arith_int`, `arith_float`, `arith_bigint` | integer, float, and bignum arithmetic loops |
| ctrl | `ctrl_for_range`, `ctrl_while`, `ctrl_fib_recursive` | loop + branch overhead |
| coll | `coll_list`, `coll_dict`, `coll_set` | container ops including mid-sequence deletes |
| str | `str_concat`, `str_join_split`, `str_format` | string building and f-string formatting |
| call | `call_plain`, `call_kwargs`, `call_closure` | function call dispatch, kw handling, closures |
| class | `class_attrs`, `class_method`, `class_mro` | attribute lookup and MRO traversal |
| gen | `gen_yield`, `gen_pipeline` | generator resume and pipelined generators |
| async | `async_drive` | coroutine dispatch under `asyncio.run` |
| real | `real_fib_iter`, `real_nqueens`, `real_wordcount` | mixed realistic workloads |

Each case is self-contained: no imports beyond `time` (and `asyncio` for
the async case). All inputs are hard-coded constants so runs are
deterministic.

## Reading the ratio

The `ratio` column is `goipy_median / cpython_median`. `7.0x` means goipy
took 7× as long as CPython on that case. A value below `1.0x` means
goipy was faster — only `ctrl_while` currently hits that, because it
does one 1M-iteration bignum loop and Go's `math/big` beats CPython's
`PyLong` on very wide integers.

Ratios in the 3×–15× range are typical for an interpreter that has no
JIT, no inline caches specialized for hot dispatch, and no specialized
instructions for common patterns — CPython has all three. The spec for
this suite is at `~/notes/Spec/0900/0972_goipy_benchmark.md`.

See `RESULTS.md` for the latest numbers.

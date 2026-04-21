# Results — goipy vs CPython 3.14

Captured on 2026-04-21. Regenerate with `bench/run.sh`.

- Host: Apple M4, Darwin 24.6.0 (arm64)
- Go: 1.26.2
- CPython: 3.14.4
- Runs per interpreter per case: 3 (median reported)

CHECKSUM status: **24/24 cases matched** byte-for-byte.

| Case | CPython 3.14 (ms) | goipy (ms) | ratio |
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

## Observations

- Uniform drop across every case vs. the prior snapshot. The biggest
  single win was `call_kwargs` at 10.8× → 3.4×, from a dedicated
  `CALL_KW` fast path that skips building a kwargs dict and resolves
  keyword names through a per-`Code` slot map.
- `ctrl_fib_recursive` went 14.8× → 4.3× on the back of three
  independent changes: embedding `big.Int` by value in `object.Int`,
  moving the per-code frame pool off a shared map and onto `Code`
  itself, and removing the `defer` from `runFrame`.
- `class_*` ratios (9–12× → 3.6–6.0×) pick up the same call-path
  wins plus a global-epoch `classLookup` memoisation — `LOAD_ATTR` on
  a class now hits an O(1) entry instead of walking the MRO.
- `arith_float` and `arith_bigint` land at 2.2× / 2.3× after converting
  the specialized `BINARY_OP_*` variants to write results back into
  the stack in place instead of popping twice and pushing.
- `ctrl_while` stays faster than CPython (0.7×). The workload is a
  bignum Fibonacci step; Go's `math/big` still beats `PyLong` on very
  wide integers once the mantissa grows to thousands of bits.
- `str_concat` remains 8.5× but its absolute time is under 1.5 ms;
  not a practical bottleneck and deliberately deferred.

## Method notes

- Each `.py` is compiled with `python3.14 -c "py_compile.compile(...)"`
  into a `.pyc` and both interpreters run the same `.pyc`. This keeps
  parsing and compilation out of the timed section.
- Timing is done inside the program with `time.perf_counter()` around
  `main()`. Interpreter startup is not counted here; it is a separate
  dimension.
- Every case prints a `CHECKSUM:` line; the harness compares the two
  strings byte-for-byte. Any mismatch is flagged in the output and the
  ratio is suppressed. Comparing times of programs that disagree on
  their result is worse than useless.

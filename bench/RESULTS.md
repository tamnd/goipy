# Results — goipy vs CPython 3.14

Captured on 2026-04-21. Regenerate with `bench/run.sh`.

- Host: Apple M4, Darwin 24.6.0 (arm64)
- Go: 1.26.2
- CPython: 3.14.4
- Runs per interpreter per case: 3 (median reported)

CHECKSUM status: **24/24 cases matched** byte-for-byte.

| Case | CPython 3.14 (ms) | goipy (ms) | ratio |
|---|---:|---:|---:|
| arith_bigint | 0.010 | 0.048 | 4.8x |
| arith_float | 20.331 | 122.560 | 6.0x |
| arith_int | 56.684 | 407.505 | 7.2x |
| async_drive | 6.553 | 43.257 | 6.6x |
| call_closure | 59.460 | 461.949 | 7.8x |
| call_kwargs | 35.682 | 383.641 | 10.8x |
| call_plain | 52.714 | 531.603 | 10.1x |
| class_attrs | 18.739 | 170.465 | 9.1x |
| class_method | 25.353 | 259.457 | 10.2x |
| class_mro | 13.239 | 162.569 | 12.3x |
| coll_dict | 6.662 | 41.833 | 6.3x |
| coll_list | 7.886 | 56.396 | 7.2x |
| coll_set | 4.992 | 33.056 | 6.6x |
| ctrl_fib_recursive | 8.545 | 126.641 | 14.8x |
| ctrl_for_range | 130.474 | 995.045 | 7.6x |
| ctrl_while | 9045.271 | 3357.178 | 0.4x |
| gen_pipeline | 10.912 | 66.771 | 6.1x |
| gen_yield | 15.819 | 100.317 | 6.3x |
| real_fib_iter | 0.267 | 0.625 | 2.3x |
| real_nqueens | 12.104 | 75.211 | 6.2x |
| real_wordcount | 12.758 | 94.413 | 7.4x |
| str_concat | 0.165 | 1.508 | 9.1x |
| str_format | 16.544 | 49.519 | 3.0x |
| str_join_split | 3.933 | 18.713 | 4.8x |

## Observations

- Call dispatch (`call_*`, `class_method`) is the hottest path by ratio,
  10×–12× slower. Candidate targets for inline caches or specialized
  opcodes before any other optimization work.
- `ctrl_fib_recursive` is 14.8× because it is nothing but call and
  compare; it falls out of the call-dispatch cost above.
- `ctrl_while` is faster on goipy (0.4×) because the workload is 1M
  iterations of a bignum Fibonacci step. Go's `math/big` outperforms
  CPython's `PyLong` on very wide integers, and that dominates once the
  mantissa is thousands of bits long.
- `str_format` lands at 3.0× — the f-string path is already in good
  shape. `str_concat` is 9.1× but its absolute time is under 2 ms; not
  a practical bottleneck.
- `coll_dict` landed at 6.3× after fixing an O(n²) regression in
  `object.Dict.Delete` (was 3843× before the fix — it rebuilt the index
  map on every delete instead of using tombstones).

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

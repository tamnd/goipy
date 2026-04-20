# goipy — pure-Go CPython 3.14 bytecode interpreter

`goipy` runs `.pyc` files produced by CPython 3.14 without cgo and without
an embedded CPython distribution. See
[spec 0968](../../notes/Spec/0900/0968_go_python_bytecode_interpreter.md)
for design details.

## Quick start

```bash
python3.14 -m py_compile hello.py
go run ./cmd/goipy run hello.cpython-314.pyc
```

## Status

v0.1 — arithmetic, strings, lists/tuples/dicts/sets, control flow,
functions, closures, simple classes, exceptions.

Not yet: generators, `async`/`await`, `with`-statement, pattern matching,
C extensions.

## License

MIT.

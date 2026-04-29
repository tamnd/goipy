# pickle.PickleBuffer + compile()/exec(str)/eval(str) raise SyntaxError

print("# section 1: PickleBuffer construct/bytes/raw/release")

import pickle

buf = pickle.PickleBuffer(b"hello world")
print(type(buf).__name__)
print(bytes(buf))
print(len(bytes(buf)))

raw = buf.raw()
print(bytes(raw))

# release() invalidates the buffer; subsequent bytes() raises ValueError.
buf.release()
try:
    bytes(buf)
except ValueError as e:
    print("ValueError:", e)


print("# section 2: PickleBuffer wraps bytearray and memoryview too")

ba = bytearray(b"abc")
buf2 = pickle.PickleBuffer(ba)
print(bytes(buf2))

mv = memoryview(b"xyz")
buf3 = pickle.PickleBuffer(mv)
print(bytes(buf3))


print("# section 3: PickleBuffer namespace presence")

print(isinstance(buf, pickle.PickleBuffer))


print("# section 4: compile()/exec(str)/eval(str) error class is SyntaxError")

# Invalid Python: both CPython and goipy raise SyntaxError. This tests
# the *class* (SyntaxError, not the legacy NotImplementedError) — user
# code that does `try: compile(...) except SyntaxError:` works.
try:
    compile("if 1", "<s>", "eval")
except SyntaxError:
    print("compile -> SyntaxError")

try:
    exec("def f(:")
except SyntaxError:
    print("exec -> SyntaxError")

try:
    eval("a b c")
except SyntaxError:
    print("eval -> SyntaxError")


print("done")

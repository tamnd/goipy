import marshal
import io

# ===== module attributes =====
print(isinstance(marshal.version, int))    # True

# ===== dumps returns bytes =====
d = marshal.dumps(42)
print(isinstance(d, bytes))               # True
print(len(d) > 0)                         # True

# ===== round-trips: None / bool =====
print(marshal.loads(marshal.dumps(None)) is None)    # True
print(marshal.loads(marshal.dumps(True)) is True)    # True
print(marshal.loads(marshal.dumps(False)) is False)  # True

# ===== round-trips: int =====
for n in [0, 1, -1, 127, 128, 255, 256, -128, -256,
          2**15, 2**16, 2**31 - 1, -(2**31),
          2**31, -(2**31 + 1), 2**40, -(2**40), 2**62]:
    assert marshal.loads(marshal.dumps(n)) == n, f'int fail: {n}'
print('int round-trips ok')               # int round-trips ok

# ===== round-trips: float =====
for f in [0.0, 1.0, -1.0, 3.14, -0.5, 1e308, float('inf'), float('-inf')]:
    assert marshal.loads(marshal.dumps(f)) == f, f'float fail: {f}'
print('float round-trips ok')             # float round-trips ok

# ===== round-trips: complex =====
for c in [0j, 1+2j, -1-1j, 3.14+2.71j]:
    assert marshal.loads(marshal.dumps(c)) == c, f'complex fail: {c}'
print('complex round-trips ok')           # complex round-trips ok

# ===== round-trips: str =====
for s in ['', 'hello', 'unicode \u00e9', 'a' * 256, '\n\t\r']:
    assert marshal.loads(marshal.dumps(s)) == s, f'str fail: {repr(s)}'
print('str round-trips ok')               # str round-trips ok

# ===== round-trips: bytes =====
for b in [b'', b'hello', b'\x00\xff\x80', b'x' * 300]:
    assert marshal.loads(marshal.dumps(b)) == b, f'bytes fail: {b}'
print('bytes round-trips ok')             # bytes round-trips ok

# ===== round-trips: tuple =====
for t in [(), (1,), (1, 2), (1, 2, 3), tuple(range(256)), (1, 'a', None, True)]:
    assert marshal.loads(marshal.dumps(t)) == t, f'tuple fail: {t}'
print('tuple round-trips ok')             # tuple round-trips ok

# ===== round-trips: list =====
for lst in [[], [1, 2, 3], ['a', 'b'], [None, True, False]]:
    assert marshal.loads(marshal.dumps(lst)) == lst, f'list fail: {lst}'
print('list round-trips ok')              # list round-trips ok

# ===== round-trips: dict =====
for d in [{}, {'a': 1}, {'x': 1, 'y': 2}, {'nested': [1, 2]}]:
    assert marshal.loads(marshal.dumps(d)) == d, f'dict fail: {d}'
print('dict round-trips ok')              # dict round-trips ok

# ===== round-trips: set =====
s = marshal.loads(marshal.dumps({1, 2, 3}))
print(type(s).__name__)                   # set
print(sorted(s))                          # [1, 2, 3]

s2 = marshal.loads(marshal.dumps(set()))
print(type(s2).__name__)                  # set
print(len(s2))                            # 0

# ===== round-trips: frozenset =====
fs = marshal.loads(marshal.dumps(frozenset([4, 5, 6])))
print(type(fs).__name__)                  # frozenset
print(sorted(fs))                         # [4, 5, 6]

fs2 = marshal.loads(marshal.dumps(frozenset()))
print(type(fs2).__name__)                 # frozenset
print(len(fs2))                           # 0

# ===== round-trips: Ellipsis =====
print(marshal.loads(marshal.dumps(...)) is ...)   # True

# ===== nested structure =====
nested = {'a': [1, (2, 3), {'b': True}], 'c': None}
print(marshal.loads(marshal.dumps(nested)))       # {'a': [1, (2, 3), {'b': True}], 'c': None}

# ===== version parameter (all produce loadable output) =====
for ver in range(5):
    d = marshal.dumps([1, 'x', None, True], ver)
    assert isinstance(d, bytes)
    assert marshal.loads(d) == [1, 'x', None, True]
print('all versions ok')                  # all versions ok

# ===== loads ignores extra bytes =====
d = marshal.dumps(42) + b'\x00\x00garbage'
print(marshal.loads(d))                   # 42

# ===== loads with bytearray =====
d = marshal.dumps(99)
print(marshal.loads(bytearray(d)))        # 99

# ===== EOFError on empty input =====
try:
    marshal.loads(b'')
except EOFError:
    print('EOFError empty')               # EOFError empty

# ===== ValueError on unsupported type =====
try:
    marshal.dumps(object())
except ValueError:
    print('ValueError unsupported')       # ValueError unsupported

# ===== dump / load via BytesIO =====
buf = io.BytesIO()
marshal.dump({'key': 'value', 'n': 99}, buf)
buf.seek(0)
print(marshal.load(buf))                  # {'key': 'value', 'n': 99}

buf2 = io.BytesIO()
marshal.dump([1, 2, 3], buf2)
buf2.seek(0)
print(marshal.load(buf2))                 # [1, 2, 3]

# ===== dump with version =====
buf3 = io.BytesIO()
marshal.dump(42, buf3, 4)
buf3.seek(0)
print(marshal.load(buf3))                 # 42

print('done')                             # done

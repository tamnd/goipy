import pickle
import io

# ===== constants =====
print(isinstance(pickle.DEFAULT_PROTOCOL, int))  # True
print(isinstance(pickle.HIGHEST_PROTOCOL, int))  # True
print(pickle.DEFAULT_PROTOCOL <= pickle.HIGHEST_PROTOCOL)  # True
print(pickle.format_version)                     # 5.0 (or similar)
print(isinstance(pickle.compatible_formats, list))  # True

# ===== exception hierarchy =====
print(issubclass(pickle.PickleError, Exception))      # True
print(issubclass(pickle.PicklingError, pickle.PickleError))    # True
print(issubclass(pickle.UnpicklingError, pickle.PickleError))  # True

# ===== dumps() / loads() round-trips =====

# None
print(pickle.loads(pickle.dumps(None)) is None)       # True

# bool
print(pickle.loads(pickle.dumps(True)) is True)       # True
print(pickle.loads(pickle.dumps(False)) is False)     # True

# int
print(pickle.loads(pickle.dumps(0)))                  # 0
print(pickle.loads(pickle.dumps(42)))                 # 42
print(pickle.loads(pickle.dumps(-1)))                 # -1
print(pickle.loads(pickle.dumps(255)))                # 255
print(pickle.loads(pickle.dumps(256)))                # 256
print(pickle.loads(pickle.dumps(65535)))              # 65535
print(pickle.loads(pickle.dumps(65536)))              # 65536
print(pickle.loads(pickle.dumps(-128)))               # -128
print(pickle.loads(pickle.dumps(2**31 - 1)))          # 2147483647
print(pickle.loads(pickle.dumps(-(2**31))))           # -2147483648
print(pickle.loads(pickle.dumps(2**40)))              # 1099511627776
print(pickle.loads(pickle.dumps(-(2**40))))           # -1099511627776

# float
print(pickle.loads(pickle.dumps(3.14)))               # 3.14
print(pickle.loads(pickle.dumps(-0.5)))               # -0.5
print(pickle.loads(pickle.dumps(0.0)))                # 0.0
print(pickle.loads(pickle.dumps(float('inf'))))       # inf
print(pickle.loads(pickle.dumps(float('-inf'))))      # -inf

# str
print(pickle.loads(pickle.dumps('')))                 # (empty)
print(pickle.loads(pickle.dumps('hello')))            # hello
print(pickle.loads(pickle.dumps('unicode \u00e9')))   # unicode é
print(pickle.loads(pickle.dumps('with\nnewline')))    # with\nnewline (via repr-like)

# bytes
print(pickle.loads(pickle.dumps(b'')))                # b''
print(pickle.loads(pickle.dumps(b'hello')))           # b'hello'
print(pickle.loads(pickle.dumps(b'\x00\xff\x80')))    # b'\x00\xff\x80'

# tuple
print(pickle.loads(pickle.dumps(())))                 # ()
print(pickle.loads(pickle.dumps((1,))))               # (1,)
print(pickle.loads(pickle.dumps((1, 2))))             # (1, 2)
print(pickle.loads(pickle.dumps((1, 2, 3))))          # (1, 2, 3)
print(pickle.loads(pickle.dumps((1, 'a', None))))     # (1, 'a', None)

# list
print(pickle.loads(pickle.dumps([])))                 # []
print(pickle.loads(pickle.dumps([1, 2, 3])))          # [1, 2, 3]
print(pickle.loads(pickle.dumps(['a', 'b'])))         # ['a', 'b']

# dict
print(pickle.loads(pickle.dumps({})))                 # {}
print(pickle.loads(pickle.dumps({'a': 1})))           # {'a': 1}
print(pickle.loads(pickle.dumps({'x': 1, 'y': 2})))  # {'x': 1, 'y': 2}

# set — use sorted for determinism
s = pickle.loads(pickle.dumps({1, 2, 3}))
print(type(s).__name__)                               # set
print(sorted(s))                                      # [1, 2, 3]

s2 = pickle.loads(pickle.dumps(set()))
print(type(s2).__name__)                              # set
print(len(s2))                                        # 0

# frozenset
fs = pickle.loads(pickle.dumps(frozenset([1, 2, 3])))
print(type(fs).__name__)                              # frozenset
print(sorted(fs))                                     # [1, 2, 3]

fs2 = pickle.loads(pickle.dumps(frozenset()))
print(type(fs2).__name__)                             # frozenset
print(len(fs2))                                       # 0

# nested
nested = {'a': [1, (2, 3), {'b': True}], 'c': None}
print(pickle.loads(pickle.dumps(nested)))             # {'a': [1, (2, 3), {'b': True}], 'c': None}

# ===== protocol argument =====
for proto in range(pickle.HIGHEST_PROTOCOL + 1):
    d = pickle.dumps([1, 'x', None, True], protocol=proto)
    assert isinstance(d, bytes)
    assert pickle.loads(d) == [1, 'x', None, True]
print('all protocols ok')                             # all protocols ok

# protocol=None uses default
d = pickle.dumps(42, protocol=None)
print(pickle.loads(d))                               # 42

# ===== dump() / load() with file =====
buf = io.BytesIO()
pickle.dump({'key': 'value', 'n': 99}, buf)
buf.seek(0)
print(pickle.load(buf))                              # {'key': 'value', 'n': 99}

buf2 = io.BytesIO()
pickle.dump([1, 2, 3], buf2)
buf2.seek(0)
print(pickle.load(buf2))                             # [1, 2, 3]

# ===== multiple objects in one stream (sequential) =====
buf3 = io.BytesIO()
pickle.dump(1, buf3)
pickle.dump('hello', buf3)
pickle.dump([True, False], buf3)
buf3.seek(0)
print(pickle.load(buf3))                             # 1
print(pickle.load(buf3))                             # hello
print(pickle.load(buf3))                             # [True, False]

# ===== dumps returns bytes =====
d = pickle.dumps('test')
print(isinstance(d, bytes))                           # True
print(len(d) > 0)                                     # True

# ===== PickleError is catchable =====
try:
    raise pickle.PicklingError('test error')
except pickle.PickleError as e:
    print('caught PickleError')                       # caught PickleError

try:
    raise pickle.UnpicklingError('bad data')
except pickle.UnpicklingError as e:
    print('caught UnpicklingError')                   # caught UnpicklingError

# ===== loads with bytes and bytearray =====
d = pickle.dumps(42)
print(pickle.loads(bytearray(d)))                     # 42

print('done')

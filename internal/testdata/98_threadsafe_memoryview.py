# Memoryview operations from the Python 3.13+ thread safety docs.

# --- From immutable bytes ---
b = b"hello"
mv = memoryview(b)
print(mv.readonly)       # True
print(mv.format)         # B
print(mv.itemsize)       # 1
print(mv.ndim)           # 1
print(mv.shape)          # (5,)
print(mv.strides)        # (1,)
print(mv.suboffsets)     # ()
print(mv.nbytes)         # 5
print(mv.c_contiguous)   # True
print(mv.f_contiguous)   # True
print(mv.contiguous)     # True
print(bytes(mv.obj))     # b'hello'

# reading
print(mv[1])             # 101  (ord('e'))
print(bytes(mv[1:4]))    # b'ell'
print(bytes(mv))         # b'hello'
print(mv.tobytes())      # b'hello'
print(mv.tolist())       # [104, 101, 108, 108, 111]
mv.release()

# --- From mutable bytearray ---
ba = bytearray(b"world")
mv2 = memoryview(ba)
print(mv2.readonly)      # False
print(mv2[0])            # 119  (ord('w'))

# write through the view
mv2[0] = 87              # ord('W')
print(ba[0:5])           # bytearray(b'World')
mv2[1:3] = b"OR"
print(ba)                # bytearray(b'WORld')

print(mv2.tobytes())     # b'WORld'
mv2.release()

# --- BufferError: resize while view is active ---
ba2 = bytearray(b"abc")
mv3 = memoryview(ba2)
try:
    ba2.append(100)
    print("no error")
except BufferError:
    print("BufferError raised")  # BufferError raised
mv3.release()

# --- After release: view is usable but ba is resizable ---
ba2.append(100)
print(ba2)               # bytearray(b'abcd')

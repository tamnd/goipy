import struct

# --- error exception ---
print(issubclass(struct.error, Exception))
try:
    struct.pack('z', 1)
except struct.error:
    print('bad char caught')

try:
    struct.unpack('<i', b'\x00\x00')
except struct.error:
    print('short buffer caught')

try:
    struct.pack('<b', 200)
except struct.error:
    print('overflow caught')

# --- calcsize ---
print(struct.calcsize('<i'))
print(struct.calcsize('>ihb'))
print(struct.calcsize('5s'))
print(struct.calcsize('<eH'))

# --- pack / unpack round-trips ---
# integers
print(struct.unpack('<i', struct.pack('<i', -123)))
print(struct.unpack('>H', struct.pack('>H', 1)))
print(struct.unpack('<Q', struct.pack('<Q', 2**63)))
print(struct.unpack('>q', struct.pack('>q', -1)))

# bool
print(struct.unpack('<??', struct.pack('<??', True, False)))

# char
print(struct.unpack('<c', struct.pack('<c', b'A')))

# pad byte (x) — consumes no argument
data = struct.pack('<xHx', 0xABCD)
print(struct.calcsize('<xHx'))
print(struct.unpack('<xHx', data))

# strings
print(struct.pack('<5s', b'hello'))
print(struct.unpack('<5s', b'hi\x00\x00\x00'))

# Pascal string p
p_data = struct.pack('<4p', b'hi')
print(len(p_data))
print(struct.unpack('<4p', p_data))

# float / double
print(struct.unpack('<f', struct.pack('<f', 1.5)))
print(struct.unpack('<d', struct.pack('<d', 3.14159)))

# half-float e
e_data = struct.pack('<e', 1.0)
print(len(e_data))
print(struct.unpack('<e', e_data))

e_data2 = struct.pack('>e', -2.0)
print(struct.unpack('>e', e_data2))

# --- pack_into ---
buf = bytearray(10)
struct.pack_into('<I', buf, 2, 0xDEADBEEF)
print(bytes(buf[2:6]))
print(struct.unpack('<I', buf[2:6]))

# pack_into at offset 0
struct.pack_into('>H', buf, 0, 0x1234)
print(buf[0], buf[1])

# --- unpack_from ---
data = b'\x00\x00' + struct.pack('<I', 0xDEADBEEF)
print(struct.unpack_from('<I', data, 2))

# --- iter_unpack ---
buf = struct.pack('<iii', 1, 2, 3)
it = struct.iter_unpack('<i', buf)
print(next(it))
print(next(it))
print(next(it))
try:
    next(it)
except StopIteration:
    print('iter exhausted')

# iter_unpack collect all
results = list(struct.iter_unpack('<H', struct.pack('<HHHH', 10, 20, 30, 40)))
print(results)

# iter_unpack wrong buffer size
try:
    list(struct.iter_unpack('<i', b'\x00\x00\x00'))
except struct.error:
    print('iter bad size caught')

# --- Struct class ---
s = struct.Struct('<iH')
print(s.format)
print(s.size)

data = s.pack(1000, 65535)
print(s.unpack(data))
print(s.unpack_from(b'\xff\xff' + data, 2))

# pack_into via Struct
buf = bytearray(s.size + 4)
s.pack_into(buf, 2, -1, 0)
print(struct.unpack('<iH', bytes(buf[2:2+s.size])))

# iter_unpack via Struct
s2 = struct.Struct('<B')
results = list(s2.iter_unpack(b'\x01\x02\x03'))
print(results)

print('done')

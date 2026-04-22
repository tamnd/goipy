import array

# ===== array.typecodes =====
print(array.typecodes)

# ===== construction =====
# integer typecodes
a = array.array('i', [1, 2, 3])
print(a)
print(a.typecode)
print(a.itemsize)

b = array.array('b')
print(b)
print(b.typecode)
print(b.itemsize)

# from list initializer
a = array.array('h', [10, 20, 30])
print(list(a))

# float typecodes
f = array.array('f', [1.0, 2.5, 3.0])
print(f.typecode)
print(f.itemsize)
print(list(f))

d = array.array('d', [1.1, 2.2, 3.3])
print(d.typecode)
print(d.itemsize)

# unsigned types
u = array.array('B', [0, 100, 255])
print(list(u))

u = array.array('H', [0, 1000, 65535])
print(list(u))

u = array.array('I', [0, 1, 2**32 - 1])
print(list(u))

# large integer types
q = array.array('q', [-1, 0, 2**62])
print(list(q))

Q = array.array('Q', [0, 1, 2**63])
print(list(Q))

# ===== itemsize for each typecode =====
for tc in 'bBhHiIlLqQfd':
    print(tc, array.array(tc).itemsize)

# ===== append =====
a = array.array('i')
a.append(10)
a.append(20)
a.append(30)
print(list(a))  # [10, 20, 30]

# ===== extend =====
a = array.array('i', [1, 2])
a.extend([3, 4, 5])
print(list(a))  # [1, 2, 3, 4, 5]

# extend from another array
b = array.array('i', [6, 7])
a.extend(b)
print(list(a))  # [1, 2, 3, 4, 5, 6, 7]

# ===== fromlist =====
a = array.array('i', [1])
a.fromlist([2, 3, 4])
print(list(a))  # [1, 2, 3, 4]

# ===== insert =====
a = array.array('i', [1, 2, 3])
a.insert(1, 10)
print(list(a))  # [1, 10, 2, 3]

a.insert(0, 99)
print(list(a))  # [99, 1, 10, 2, 3]

a.insert(100, 42)   # past end → appended
print(list(a))  # [99, 1, 10, 2, 3, 42]

a.insert(-1, 77)    # negative index
print(list(a))  # [99, 1, 10, 2, 3, 77, 42]

# ===== pop =====
a = array.array('i', [10, 20, 30, 40])
print(a.pop())       # 40  (last by default)
print(a.pop(0))      # 10  (first)
print(a.pop(1))      # 30  (index 1)
print(list(a))       # [20]

# pop from empty raises IndexError
try:
    array.array('i').pop()
except IndexError:
    print("IndexError: pop from empty array")

# ===== remove =====
a = array.array('i', [1, 2, 3, 2, 4])
a.remove(2)
print(list(a))  # [1, 3, 2, 4]  — removes first occurrence

# remove missing raises ValueError
try:
    a.remove(99)
except ValueError:
    print("ValueError: x not in array")

# ===== count =====
a = array.array('i', [1, 2, 3, 2, 2, 4])
print(a.count(2))   # 3
print(a.count(5))   # 0

# ===== index =====
a = array.array('i', [10, 20, 30, 20, 40])
print(a.index(20))      # 1
print(a.index(20, 2))   # 3  (search from index 2)
print(a.index(40))      # 4

# index missing raises ValueError
try:
    a.index(99)
except ValueError:
    print("ValueError: not in array")

# ===== reverse =====
a = array.array('i', [1, 2, 3, 4, 5])
a.reverse()
print(list(a))  # [5, 4, 3, 2, 1]

# ===== subscript / index access =====
a = array.array('i', [10, 20, 30, 40])
print(a[0])    # 10
print(a[-1])   # 40
print(a[2])    # 30

# slicing returns array
s = a[1:3]
print(type(s).__name__)   # array
print(list(s))            # [20, 30]
print(s.typecode)         # i

# ===== setitem =====
a = array.array('i', [1, 2, 3])
a[1] = 99
print(list(a))  # [1, 99, 3]

a[-1] = 77
print(list(a))  # [1, 99, 77]

# ===== delitem =====
a = array.array('i', [10, 20, 30, 40])
del a[1]
print(list(a))  # [10, 30, 40]

del a[-1]
print(list(a))  # [10, 30]

# ===== len =====
a = array.array('i', [1, 2, 3, 4, 5])
print(len(a))  # 5
print(len(array.array('d')))  # 0

# ===== iteration =====
a = array.array('i', [1, 2, 3])
total = 0
for x in a:
    total += x
print(total)  # 6

# list/tuple conversion
print(list(array.array('i', [9, 8, 7])))  # [9, 8, 7]
print(tuple(array.array('b', [-1, 0, 1])))  # (-1, 0, 1)

# ===== in operator =====
a = array.array('i', [1, 2, 3, 4, 5])
print(3 in a)   # True
print(9 in a)   # False

# ===== tobytes / frombytes =====
a = array.array('b', [1, 2, 3])
raw = a.tobytes()
print(raw)  # b'\x01\x02\x03'
print(len(raw))  # 3

b = array.array('b')
b.frombytes(raw)
print(list(b))  # [1, 2, 3]

# round-trip for 'i' (4 bytes each)
a = array.array('i', [1, 2, 3])
raw = a.tobytes()
print(len(raw))  # 12

b = array.array('i')
b.frombytes(raw)
print(list(b))  # [1, 2, 3]

# round-trip for 'd' (8 bytes each)
a = array.array('d', [1.5, 2.5])
raw = a.tobytes()
print(len(raw))  # 16

b = array.array('d')
b.frombytes(raw)
print(list(b))  # [1.5, 2.5]

# ===== tolist =====
a = array.array('i', [10, 20, 30])
lst = a.tolist()
print(lst)           # [10, 20, 30]
print(type(lst).__name__)  # list

# ===== buffer_info =====
a = array.array('i', [1, 2, 3])
addr, length = a.buffer_info()
print(length)          # 3
print(isinstance(addr, int))  # True

# ===== repr =====
print(repr(array.array('i')))           # array('i')
print(repr(array.array('i', [1, 2])))  # array('i', [1, 2])
print(repr(array.array('d', [1.5])))   # array('d', [1.5])

# ===== TypeError on wrong value type =====
try:
    array.array('i', [1, 'hello'])
except TypeError:
    print("TypeError: integer required for 'i'")

# ===== overflow error =====
try:
    array.array('b', [200])  # 200 > 127
except (OverflowError, TypeError, ValueError):
    print("error: signed char overflow")

try:
    array.array('B', [-1])  # negative unsigned
except (OverflowError, TypeError, ValueError):
    print("error: unsigned char underflow")

# ===== unknown typecode =====
try:
    array.array('x')
except ValueError:
    print("ValueError: bad typecode")

# ===== byteswap =====
a = array.array('b', [1, 2, 3])   # 1-byte: byteswap is no-op
a.byteswap()
print(list(a))  # [1, 2, 3]

# ===== construction from bytes =====
raw = bytes([5, 0, 0, 0, 10, 0, 0, 0])  # two little-endian int32: 5 and 10
a = array.array('i', raw)
print(list(a))  # [5, 10]

print('done')

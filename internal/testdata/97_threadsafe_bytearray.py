# Bytearray operations from the Python 3.13+ thread safety docs.

ba = bytearray(b"hello world")

# --- Reads ---
print(len(ba))           # 11
print(ba + b" !")        # bytearray(b'hello world !')
print(ba == bytearray(b"hello world"))  # True
print(ba < bytearray(b"zzz"))           # True
print(ba[0])             # 104
print(ba[0:5])           # bytearray(b'hello')

# --- Writes ---
ba[0] = 72
print(ba[0:5])           # bytearray(b'Hello')
ba[6:11] = b"earth"
print(ba)                # bytearray(b'Hello earth')

# --- Mutating methods ---
ba2 = bytearray(b"abc")
ba2.append(100)
print(ba2)               # bytearray(b'abcd')
ba2.extend(b"ef")
print(ba2)               # bytearray(b'abcdef')
ba2.insert(0, 90)
print(ba2)               # bytearray(b'Zabcdef')

print(ba2.pop())         # 102  (ord('f'))
print(ba2.pop(0))        # 90   (ord('Z'))
print(ba2)               # bytearray(b'abcde')

ba3 = bytearray(b"abcba")
ba3.remove(ord("b"))
print(ba3)               # bytearray(b'acba')
ba3.reverse()
print(ba3)               # bytearray(b'abca')
ba3.clear()
print(ba3)               # bytearray(b'')

# --- New-object ops ---
ba4 = bytearray(b"hi")
print(ba4.copy())        # bytearray(b'hi')
print(ba4 * 3)           # bytearray(b'hihihi')
print(ord("h") in ba4)  # True

# --- String-like methods ---
ba5 = bytearray(b"  hello  ")
print(ba5.find(b"ell"))         # 3
print(ba5.replace(b"hello", b"world"))  # bytearray(b'  world  ')
parts = bytearray(b"a,b,c").split(b",")
print(parts)             # [bytearray(b'a'), bytearray(b'b'), bytearray(b'c')]
print(bytearray(b"caf\xc3\xa9").decode("utf-8"))  # café

# --- Safe pattern: copy-before-iterate ---
shared = bytearray(b"hello")
snapshot = shared.copy()
total = sum(snapshot)
print(total)             # 532 (h+e+l+l+o = 104+101+108+108+111)

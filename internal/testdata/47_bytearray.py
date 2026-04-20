# bytearray: mutable bytes.

ba = bytearray(b"hello")
print(ba)
print(type(ba).__name__)
print(len(ba))
print(ba[0], ba[-1])
print(ba[1:4])
print(ba == b"hello")
print(b"hello" == ba)

# Mutation via item assignment.
ba[0] = 72
print(ba)
ba.append(33)
print(ba)
ba.extend(b" world")
print(ba)
print(ba.pop())
print(ba.pop(0))
print(ba)

# Constructors.
print(bytearray())
print(bytearray(3))
print(bytearray([65, 66, 67]))

# Iteration yields ints.
for b in bytearray(b"AB"):
    print(b)

# Membership: int matches a byte, bytes/bytearray match a subsequence.
print(65 in bytearray(b"AB"))
print(b"A" in bytearray(b"AB"))
print(bytearray(b"B") in bytearray(b"AB"))

# isinstance discrimination.
print(isinstance(bytearray(), bytearray))
print(isinstance(bytearray(), bytes))
print(isinstance(b"", bytearray))

# Concatenation: result type follows the left operand.
print(type(b"a" + bytearray(b"b")).__name__)
print(type(bytearray(b"a") + b"b").__name__)

# hex / decode.
print(bytearray(b"AB").hex())
print(bytearray(b"hi").decode())

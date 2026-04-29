"""v0.0.341 encoding error handlers (C1) and bytes.hex grouping (C2)."""


# ── 1. ascii + replace
print('a-中-b'.encode('ascii', 'replace'))


# ── 2. ascii + ignore
print('a-中-b'.encode('ascii', 'ignore'))


# ── 3. ascii + xmlcharrefreplace
print('中'.encode('ascii', 'xmlcharrefreplace'))


# ── 4. ascii + backslashreplace
print('中'.encode('ascii', 'backslashreplace'))


# ── 5. ascii + namereplace (uses unicode name)
print('中'.encode('ascii', 'namereplace'))


# ── 6. ascii + strict raises
try:
    '中'.encode('ascii')
    print('no error')
except UnicodeEncodeError as e:
    print('UnicodeEncodeError raised')


# ── 7. latin-1 round trip for U+00FF byte
print('ÿ'.encode('latin-1'))


# ── 8. utf-8 default encoding
print('hi'.encode())


# ── 9. encoding= keyword
print('hi'.encode(encoding='utf-8'))


# ── 10. bytes.decode round trips
print(b'hello'.decode('utf-8'))
print(b'hello'.decode('ascii'))


# ── 11. bytes.hex with separator
print(b'\xaa\xbb\xcc\xdd'.hex())
print(b'\xaa\xbb\xcc\xdd'.hex('-'))
print(b'\xaa\xbb\xcc\xdd'.hex(':', 2))
print(b'\xaa\xbb\xcc\xdd'.hex('-', -2))


# ── 12. bytearray.hex with separator
ba = bytearray(b'\x01\x02\x03\x04\x05')
print(ba.hex(' '))
print(ba.hex(' ', 2))
print(ba.hex(' ', -2))


# ── 13. bytes.hex on empty bytes
print(repr(b''.hex('-')))

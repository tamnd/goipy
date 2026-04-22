import codecs

# --- BOM constants ---
print(codecs.BOM_UTF8)
print(codecs.BOM_UTF16_BE)
print(codecs.BOM_UTF16_LE)
print(codecs.BOM_UTF32_BE)
print(codecs.BOM_UTF32_LE)

# --- encode / decode: text encodings ---
print(codecs.encode('hello', 'utf-8'))
print(codecs.decode(b'hello', 'utf-8'))
print(codecs.encode('hello', 'ascii'))
print(codecs.decode(b'hello', 'ascii'))
print(codecs.encode('\xe9', 'latin-1'))
print(codecs.decode(b'\xe9', 'latin-1'))

# utf-8 alias normalisation
print(codecs.encode('hi', 'utf_8'))
print(codecs.encode('hi', 'UTF-8'))

# --- encode / decode: binary transforms ---
print(codecs.encode(b'hello', 'hex_codec'))
print(codecs.decode(b'68656c6c6f', 'hex_codec'))
print(codecs.encode(b'hello', 'base64_codec').strip())
print(codecs.decode(b'aGVsbG8=\n', 'base64_codec'))

# rot_13 (text transform)
print(codecs.encode('Hello, World!', 'rot_13'))
print(codecs.decode('Uryyb, Jbeyq!', 'rot_13'))

# --- error handlers ---
print(codecs.encode('caf\u00e9', 'ascii', 'ignore'))
print(codecs.encode('caf\u00e9', 'ascii', 'replace'))
print(codecs.encode('caf\u00e9', 'ascii', 'xmlcharrefreplace'))
print(codecs.encode('caf\u00e9', 'ascii', 'backslashreplace'))

print(codecs.decode(b'\xff\xfe', 'ascii', 'ignore'))
print(codecs.decode(b'\xff\xfe', 'ascii', 'replace'))

# --- lookup ---
ci = codecs.lookup('utf-8')
print(ci.name)
print(callable(ci.encode))
print(callable(ci.decode))

ci2 = codecs.lookup('ascii')
print(ci2.name)

# lookup normalises name
print(codecs.lookup('UTF_8').name)
print(codecs.lookup('utf8').name)

# --- getencoder / getdecoder ---
enc = codecs.getencoder('utf-8')
print(enc('hello'))

dec = codecs.getdecoder('utf-8')
print(dec(b'hello'))

# --- register_error / lookup_error ---
codecs.register_error('myerr', codecs.ignore_errors)
handler = codecs.lookup_error('myerr')
print(handler is codecs.ignore_errors)

# Built-in error handlers accessible as attributes
print(callable(codecs.strict_errors))
print(callable(codecs.ignore_errors))
print(callable(codecs.replace_errors))
print(callable(codecs.backslashreplace_errors))
print(callable(codecs.xmlcharrefreplace_errors))

# --- lookup_error for built-ins ---
print(codecs.lookup_error('strict') is codecs.strict_errors)
print(codecs.lookup_error('ignore') is codecs.ignore_errors)
print(codecs.lookup_error('replace') is codecs.replace_errors)

# --- iterencode / iterdecode ---
chunks = list(codecs.iterencode(['hello', ' ', 'world'], 'utf-8'))
print(chunks)

chunks2 = list(codecs.iterdecode([b'hel', b'lo'], 'utf-8'))
print(chunks2)

# --- charmap_build ---
mapping = codecs.charmap_build('abcde')
print(mapping[ord('a')])
print(mapping[ord('e')])

print('done')

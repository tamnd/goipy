import unicodedata

# unidata_version
print(unicodedata.unidata_version)

# --- name ---
print(unicodedata.name('A'))
print(unicodedata.name('a'))
print(unicodedata.name('1'))
print(unicodedata.name('\u00e9'))
print(unicodedata.name('\u4e2d'))
print(unicodedata.name('\U0001f600'))
print(unicodedata.name('\x00', 'no name'))
try:
    unicodedata.name('\x00')
except ValueError as e:
    print(type(e).__name__)

# --- lookup ---
print(unicodedata.lookup('LATIN CAPITAL LETTER A'))
print(unicodedata.lookup('LATIN SMALL LETTER E WITH ACUTE'))
try:
    unicodedata.lookup('NONEXISTENT CHARACTER')
except KeyError:
    print('KeyError')

# --- decimal ---
print(unicodedata.decimal('9'))
print(unicodedata.decimal('\u0660'))
print(unicodedata.decimal('a', None))
print(unicodedata.decimal('\u06f5'))

# --- digit ---
print(unicodedata.digit('9'))
print(unicodedata.digit('\u00b2'))
print(unicodedata.digit('a', None))

# --- numeric ---
print(unicodedata.numeric('9'))
print(unicodedata.numeric('\u00bd'))
print(unicodedata.numeric('\u00b2'))
print(unicodedata.numeric('a', None))

# --- category ---
print(unicodedata.category('A'))
print(unicodedata.category('a'))
print(unicodedata.category('1'))
print(unicodedata.category(' '))
print(unicodedata.category('!'))
print(unicodedata.category('\u0300'))
print(unicodedata.category('\u200b'))
print(unicodedata.category('\x00'))

# --- bidirectional ---
print(unicodedata.bidirectional('A'))
print(unicodedata.bidirectional('1'))
print(unicodedata.bidirectional('\u0660'))
print(unicodedata.bidirectional('\u200f'))
print(unicodedata.bidirectional('\x00'))

# --- combining ---
print(unicodedata.combining('\u0300'))
print(unicodedata.combining('a'))
print(unicodedata.combining('\u0308'))

# --- east_asian_width ---
print(unicodedata.east_asian_width('A'))
print(unicodedata.east_asian_width('\u4e2d'))
print(unicodedata.east_asian_width('\uff01'))

# --- mirrored ---
print(unicodedata.mirrored('('))
print(unicodedata.mirrored('A'))
print(unicodedata.mirrored('['))

# --- decomposition ---
print(unicodedata.decomposition('\u00e9'))
print(unicodedata.decomposition('a'))
print(unicodedata.decomposition('\u00bc'))
print(unicodedata.decomposition('\ufb01'))

# --- normalize ---
print(repr(unicodedata.normalize('NFC', 'e\u0301')))
print(repr(unicodedata.normalize('NFD', '\u00e9')))
print(repr(unicodedata.normalize('NFKC', '\uff41')))
print(repr(unicodedata.normalize('NFKD', '\u2126')))
print(repr(unicodedata.normalize('NFC', 'hello')))

# --- is_normalized ---
print(unicodedata.is_normalized('NFC', 'hello'))
print(unicodedata.is_normalized('NFC', '\u00e9'))
print(unicodedata.is_normalized('NFD', '\u00e9'))
print(unicodedata.is_normalized('NFC', 'e\u0301'))

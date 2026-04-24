import plistlib
import datetime
import io

# --- basic string round-trip XML ---
data = plistlib.dumps({'key': 'value'})
print(isinstance(data, bytes))
result = plistlib.loads(data)
print(result['key'])

# --- int round-trip ---
data = plistlib.dumps({'n': 42})
result = plistlib.loads(data)
print(result['n'])

# --- float round-trip ---
data = plistlib.dumps({'f': 3.14})
result = plistlib.loads(data)
print(round(result['f'], 2))

# --- bool round-trip ---
data = plistlib.dumps({'t': True, 'f': False})
result = plistlib.loads(data)
print(result['t'], result['f'])

# --- bytes round-trip ---
data = plistlib.dumps({'b': b'\x00\x01\x02'})
result = plistlib.loads(data)
print(result['b'])

# --- list round-trip ---
data = plistlib.dumps({'lst': [1, 2, 3]})
result = plistlib.loads(data)
print(result['lst'])

# --- nested dict ---
data = plistlib.dumps({'outer': {'inner': 'hello'}})
result = plistlib.loads(data)
print(result['outer']['inner'])

# --- sort_keys=True (default) ---
data = plistlib.dumps({'z': 1, 'a': 2})
result = plistlib.loads(data)
print(sorted(result.keys()))

# --- sort_keys=False ---
data = plistlib.dumps({'z': 1, 'a': 2}, sort_keys=False)
result = plistlib.loads(data)
print(set(result.keys()) == {'z', 'a'})

# --- empty dict ---
data = plistlib.dumps({})
result = plistlib.loads(data)
print(result)

# --- empty list ---
data = plistlib.dumps([])
result = plistlib.loads(data)
print(result)

# --- string with XML special chars ---
data = plistlib.dumps({'s': '<hello & "world">'})
result = plistlib.loads(data)
print(result['s'])

# --- datetime naive ---
dt = datetime.datetime(2024, 3, 15, 10, 30, 0)
data = plistlib.dumps({'dt': dt})
result = plistlib.loads(data)
print(result['dt'].year, result['dt'].month, result['dt'].day)
print(result['dt'].hour, result['dt'].minute, result['dt'].second)

# --- datetime aware ---
dt = datetime.datetime(2024, 3, 15, 10, 30, 0, tzinfo=datetime.timezone.utc)
data = plistlib.dumps({'dt': dt})
result = plistlib.loads(data, aware_datetime=True)
print(result['dt'].tzinfo is not None)

# --- FMT_XML explicit ---
data = plistlib.dumps({'x': 1}, fmt=plistlib.FMT_XML)
result = plistlib.loads(data, fmt=plistlib.FMT_XML)
print(result['x'])

# --- FMT_BINARY round-trip ---
data = plistlib.dumps({'x': 1}, fmt=plistlib.FMT_BINARY)
print(data[:8])
result = plistlib.loads(data, fmt=plistlib.FMT_BINARY)
print(result['x'])

# --- binary: bool ---
data = plistlib.dumps({'t': True, 'f': False}, fmt=plistlib.FMT_BINARY)
result = plistlib.loads(data)
print(result['t'], result['f'])

# --- binary: string ---
data = plistlib.dumps({'s': 'hello'}, fmt=plistlib.FMT_BINARY)
result = plistlib.loads(data)
print(result['s'])

# --- binary: bytes ---
data = plistlib.dumps({'b': b'\xDE\xAD\xBE\xEF'}, fmt=plistlib.FMT_BINARY)
result = plistlib.loads(data)
print(result['b'])

# --- binary: float ---
data = plistlib.dumps({'f': 2.718}, fmt=plistlib.FMT_BINARY)
result = plistlib.loads(data)
print(round(result['f'], 3))

# --- binary: list ---
data = plistlib.dumps([1, 2, 3], fmt=plistlib.FMT_BINARY)
result = plistlib.loads(data)
print(result)

# --- binary: nested ---
data = plistlib.dumps({'a': {'b': [1, 2]}}, fmt=plistlib.FMT_BINARY)
result = plistlib.loads(data)
print(result['a']['b'])

# --- load/dump with file objects ---
buf = io.BytesIO()
plistlib.dump({'k': 'v'}, buf)
buf.seek(0)
result = plistlib.load(buf)
print(result['k'])

# --- UID ---
uid = plistlib.UID(42)
print(uid.data)
print(repr(uid))

# --- UID round-trip binary ---
data = plistlib.dumps({'u': plistlib.UID(7)}, fmt=plistlib.FMT_BINARY)
result = plistlib.loads(data)
print(result['u'].data)

# --- InvalidFileException ---
try:
    plistlib.loads(b'not a plist')
except plistlib.InvalidFileException as e:
    print(type(e).__name__)

# --- tuple serialized as array ---
data = plistlib.dumps({'t': (1, 2, 3)})
result = plistlib.loads(data)
print(result['t'])

# --- large integer ---
data = plistlib.dumps({'n': 2**32})
result = plistlib.loads(data)
print(result['n'])

# --- negative integer ---
data = plistlib.dumps({'n': -1})
result = plistlib.loads(data)
print(result['n'])

# --- bytearray ---
data = plistlib.dumps({'b': bytearray(b'\x01\x02')})
result = plistlib.loads(data)
print(result['b'])

# --- FMT constants have value attribute ---
print(plistlib.FMT_XML.value)
print(plistlib.FMT_BINARY.value)

# --- FMT constants repr ---
print(repr(plistlib.FMT_XML))
print(repr(plistlib.FMT_BINARY))

# --- unicode string XML ---
data = plistlib.dumps({'u': 'héllo'})
result = plistlib.loads(data)
print(result['u'])

# --- unicode string binary ---
data = plistlib.dumps({'u': 'héllo'}, fmt=plistlib.FMT_BINARY)
result = plistlib.loads(data)
print(result['u'])

# --- datetime binary round-trip ---
dt = datetime.datetime(2001, 1, 1, 0, 0, 0)
data = plistlib.dumps({'dt': dt}, fmt=plistlib.FMT_BINARY)
result = plistlib.loads(data)
print(result['dt'].year, result['dt'].month, result['dt'].day)

# --- binary: integer boundary values ---
for n in [0, 255, 256, 65535, 65536, 2**31-1, 2**32-1]:
    data = plistlib.dumps(n, fmt=plistlib.FMT_BINARY)
    result = plistlib.loads(data)
    print(result)

# --- XML output contains expected tags ---
data = plistlib.dumps({'key': 'val'})
xml = data.decode()
print('<dict>' in xml)
print('<key>key</key>' in xml)
print('<string>val</string>' in xml)

# --- auto-detect XML format ---
xml_data = plistlib.dumps({'k': 'v'}, fmt=plistlib.FMT_XML)
result = plistlib.loads(xml_data)  # fmt=None by default, auto-detect
print(result['k'])

# --- auto-detect binary format ---
bin_data = plistlib.dumps({'k': 'v'}, fmt=plistlib.FMT_BINARY)
result = plistlib.loads(bin_data)  # auto-detect
print(result['k'])

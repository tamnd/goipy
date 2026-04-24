import tomllib
import io
import math

# --- loads: basic string ---
d = tomllib.loads('title = "TOML Example"')
print(d['title'])

# --- integers ---
d = tomllib.loads('a = 99\nb = -1\nc = 0\n')
print(d['a'], d['b'], d['c'])

# --- integer bases ---
d = tomllib.loads('h = 0xDEAD\no = 0o17\nb = 0b11010110\n')
print(d['h'], d['o'], d['b'])

# --- underscores in integers ---
d = tomllib.loads('n = 1_000_000\n')
print(d['n'])

# --- float ---
d = tomllib.loads('f = 3.14\ng = -2.0\nh = 6.626e-34\n')
print(d['f'], d['g'], d['h'])

# --- special floats ---
d = tomllib.loads('a = inf\nb = -inf\nc = +inf\n')
print(d['a'], d['b'], d['c'])
print(math.isnan(tomllib.loads('x = nan')['x']))

# --- booleans ---
d = tomllib.loads('t = true\nf = false\n')
print(d['t'], d['f'])

# --- literal string (no escapes) ---
d = tomllib.loads(r"s = 'C:\Users\n'")
print(d['s'])

# --- basic string escapes ---
d = tomllib.loads('s = "\\t\\n\\\\"\n')
print(repr(d['s']))

# --- unicode escape ---
d = tomllib.loads(r's = "α\U0001F600"')
print(d['s'])

# --- multiline basic string (leading newline stripped) ---
d = tomllib.loads('s = """\nline1\nline2\n"""')
print(repr(d['s']))

# --- multiline literal string ---
d = tomllib.loads("s = '''\nline1\nline2\n'''")
print(repr(d['s']))

# --- multiline basic backslash continuation ---
d = tomllib.loads('s = """\\\n  hello \\\n  world\\\n"""')
print(d['s'])

# --- array ---
d = tomllib.loads('a = [1, 2, 3]\n')
print(d['a'])

# --- nested array ---
d = tomllib.loads('a = [[1, 2], [3, 4]]\n')
print(d['a'])

# --- mixed array ---
d = tomllib.loads('a = [1, 2.0, "three"]\n')
print(d['a'])

# --- inline table ---
d = tomllib.loads('pt = {x = 1, y = 2}\n')
print(sorted(d['pt'].items()))

# --- table ---
d = tomllib.loads('[server]\nhost = "localhost"\nport = 8080\n')
print(d['server']['host'], d['server']['port'])

# --- dotted key ---
d = tomllib.loads('a.b = 1\n')
print(d['a']['b'])

# --- table with dotted header ---
d = tomllib.loads('[a.b]\nc = 99\n')
print(d['a']['b']['c'])

# --- top-level and table coexist ---
d = tomllib.loads('name = "test"\n[meta]\nv = 1\n')
print(d['name'], d['meta']['v'])

# --- array of tables ---
d = tomllib.loads('[[fruits]]\nname = "apple"\n[[fruits]]\nname = "banana"\n')
print([f['name'] for f in d['fruits']])

# --- comment ---
d = tomllib.loads('# comment\na = 1 # inline\n')
print(d['a'])

# --- quoted key ---
d = tomllib.loads('"127.0.0.1" = "host"\n')
print(d['127.0.0.1'])

# --- empty string key ---
d = tomllib.loads('"" = "empty"\n')
print(d[''])

# --- offset datetime ---
d = tomllib.loads('dt = 1979-05-27T07:32:00Z\n')
print(type(d['dt']).__name__, d['dt'].year, d['dt'].month, d['dt'].day)
print(d['dt'].tzinfo is not None)

# --- offset datetime with +offset ---
d = tomllib.loads('dt = 1979-05-27T07:32:00+05:30\n')
print(type(d['dt']).__name__, d['dt'].tzinfo is not None)

# --- local datetime ---
d = tomllib.loads('dt = 1979-05-27T07:32:00\n')
print(type(d['dt']).__name__, d['dt'].tzinfo is None)

# --- local date ---
d = tomllib.loads('d = 1979-05-27\n')
print(type(d['d']).__name__, d['d'].year, d['d'].month, d['d'].day)

# --- local time ---
d = tomllib.loads('t = 07:32:00\n')
print(type(d['t']).__name__, d['t'].hour, d['t'].minute, d['t'].second)

# --- time with fractional seconds ---
d = tomllib.loads('t = 00:00:00.999999\n')
print(d['t'].microsecond)

# --- load() with BytesIO ---
bio = io.BytesIO(b'key = "value"\n')
d = tomllib.load(bio)
print(d['key'])

# --- TOMLDecodeError ---
try:
    tomllib.loads('key = @\n')
except tomllib.TOMLDecodeError:
    print('TOMLDecodeError')

# --- duplicate key ---
try:
    tomllib.loads('a = 1\na = 2\n')
except tomllib.TOMLDecodeError:
    print('duplicate key')

# --- duplicate table ---
try:
    tomllib.loads('[a]\n[a]\n')
except tomllib.TOMLDecodeError:
    print('duplicate table')

# --- missing section before key is fine at top level ---
d = tomllib.loads('x = 1\n')
print(d['x'])

# --- float with underscore ---
d = tomllib.loads('f = 9_224_617.445_991_228\n')
print(type(d['f']).__name__)

# --- array of tables with sub-keys ---
toml_str = '''
[[products]]
name = "Hammer"
sku = 738594937

[[products]]
name = "Nail"
sku = 284758393
'''
d = tomllib.loads(toml_str)
print(len(d['products']))
print(d['products'][0]['name'])
print(d['products'][1]['sku'])

# --- nested tables ---
d = tomllib.loads('[a]\n[a.b]\nc = 1\n')
print(d['a']['b']['c'])

# --- multiline array with comments ---
d = tomllib.loads('a = [\n  1, # one\n  2, # two\n  3,\n]\n')
print(d['a'])

# --- integer +prefix ---
d = tomllib.loads('n = +42\n')
print(d['n'])

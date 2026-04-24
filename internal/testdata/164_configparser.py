import configparser
import io

# --- constants ---
print(configparser.DEFAULTSECT)
print(configparser.MAX_INTERPOLATION_DEPTH)

# --- basic read_string, sections, options, get ---
cfg = configparser.ConfigParser()
cfg.read_string('[sec1]\nkey1 = value1\nkey2 = value2\n[sec2]\nkeyA = valueA\n')
print(cfg.sections())
print(cfg.has_section('sec1'))
print(cfg.has_section('sec3'))
print(sorted(cfg.options('sec1')))
print(cfg.get('sec1', 'key1'))
print(cfg.has_option('sec1', 'key1'))
print(cfg.has_option('sec1', 'missing'))

# --- type converters ---
cfg2 = configparser.ConfigParser()
cfg2.read_string('[n]\nival=42\nfval=3.14\nbval=yes\nb2=off\n')
print(cfg2.getint('n', 'ival'))
print(cfg2.getfloat('n', 'fval'))
print(cfg2.getboolean('n', 'bval'))
print(cfg2.getboolean('n', 'b2'))

# --- DEFAULT section inheritance ---
cfg3 = configparser.ConfigParser()
cfg3.read_string('[DEFAULT]\nport=8080\n[production]\nserver=prod.example.com\n[testing]\nport=9090\n')
print(cfg3.get('production', 'port'))
print(cfg3.get('testing', 'port'))

# --- defaults() ---
cfg4 = configparser.ConfigParser(defaults={'foo': 'bar'})
print(cfg4.defaults()['foo'])

# --- mapping protocol: section access ---
cfg5 = configparser.ConfigParser()
cfg5.read_string('[mysec]\nopt1 = hello\nopt2 = world\n')
sec = cfg5['mysec']
print(sec['opt1'])
print('opt1' in sec)
print('missing' in sec)
print(sorted(sec.keys()))

# --- set / remove ---
cfg6 = configparser.ConfigParser()
cfg6.add_section('s')
cfg6.set('s', 'key', 'val')
print(cfg6.get('s', 'key'))
print(cfg6.remove_option('s', 'key'))
print(cfg6.remove_option('s', 'missing'))
print(cfg6.remove_section('s'))
print(cfg6.remove_section('missing'))

# --- write ---
cfg7 = configparser.ConfigParser()
cfg7.read_string('[sec1]\nk1 = v1\nk2 = v2\n')
buf = io.StringIO()
cfg7.write(buf)
print(repr(buf.getvalue()))

# --- items(section) ---
cfg8 = configparser.ConfigParser()
cfg8.read_string('[s]\na = 1\nb = 2\n')
print(sorted(cfg8.items('s')))

# --- fallback ---
cfg9 = configparser.ConfigParser()
cfg9.read_string('[s]\na=1\n')
print(cfg9.get('s', 'missing', fallback='default'))
print(cfg9.getint('s', 'missing', fallback=99))
print(cfg9.getboolean('s', 'missing', fallback=False))

# --- exceptions ---
cfg10 = configparser.ConfigParser()
cfg10.read_string('[s]\na=1\n')
try:
    cfg10.get('s', 'missing')
except configparser.NoOptionError:
    print('NoOptionError')
try:
    cfg10.get('missing_sec', 'a')
except configparser.NoSectionError:
    print('NoSectionError')
try:
    cfg10.add_section('s2')
    cfg10.add_section('s2')
except configparser.DuplicateSectionError:
    print('DuplicateSectionError')

# --- BasicInterpolation ---
cfg11 = configparser.ConfigParser()
cfg11.read_string('[paths]\nhome = /home/user\nwork = %(home)s/work\n')
print(cfg11.get('paths', 'work'))
print(cfg11.get('paths', 'work', raw=True))

# --- RawConfigParser (no interpolation) ---
raw = configparser.RawConfigParser()
raw.read_string('[raw]\nkey = %(no_interp)s\n')
print(raw.get('raw', 'key'))

# --- allow_no_value ---
cfg12 = configparser.ConfigParser(allow_no_value=True)
cfg12.read_string('[flags]\ndebug\nverbose = yes\n')
print(cfg12.get('flags', 'debug'))
print(cfg12.getboolean('flags', 'verbose'))

# --- optionxform lowercases keys ---
cfg13 = configparser.ConfigParser()
cfg13.read_string('[s]\nMyKey = val\n')
print(cfg13.get('s', 'mykey'))

# --- read_dict ---
cfg14 = configparser.ConfigParser()
cfg14.read_dict({'section1': {'key1': 'val1', 'key2': 'val2'}})
print(cfg14.get('section1', 'key1'))
print(cfg14.sections())

# --- BOOLEAN_STATES ---
print(sorted(configparser.ConfigParser.BOOLEAN_STATES.items()))

# --- DuplicateOptionError ---
try:
    cfg15 = configparser.ConfigParser(strict=True)
    cfg15.read_string('[s]\na=1\na=2\n')
except configparser.DuplicateOptionError:
    print('DuplicateOptionError')

# --- MissingSectionHeaderError ---
try:
    cfg16 = configparser.ConfigParser()
    cfg16.read_string('key = val\n')
except configparser.MissingSectionHeaderError:
    print('MissingSectionHeaderError')

# --- SectionProxy items() ---
cfg17 = configparser.ConfigParser()
cfg17.read_string('[s]\nx=1\ny=2\n')
print(sorted(cfg17['s'].items()))

# --- write with DEFAULT section ---
cfg18 = configparser.ConfigParser(defaults={'base': 'val'})
cfg18.add_section('sec')
cfg18.set('sec', 'key', 'myval')
buf18 = io.StringIO()
cfg18.write(buf18)
print(repr(buf18.getvalue()))

# --- read_file ---
buf_file = io.StringIO('[file_sec]\nfoo = bar\n')
cfg19 = configparser.ConfigParser()
cfg19.read_file(buf_file)
print(cfg19.get('file_sec', 'foo'))

# --- DEFAULT via mapping ---
cfg20 = configparser.ConfigParser()
cfg20.read_string('[DEFAULT]\ndefkey = defval\n[s]\n')
print(cfg20['s']['defkey'])

# --- ExtendedInterpolation ---
cfgExt = configparser.ConfigParser(interpolation=configparser.ExtendedInterpolation())
cfgExt.read_string('[base]\npath = /home\n[user]\nhome = ${base:path}/user\nwork = ${home}/work\n')
print(cfgExt.get('user', 'home'))
print(cfgExt.get('user', 'work'))

# --- inline_comment_prefixes ---
cfg21 = configparser.ConfigParser(inline_comment_prefixes=(';',))
cfg21.read_string('[s]\na = value ; inline comment\n')
print(cfg21.get('s', 'a'))

# --- write(space_around_delimiters=False) ---
cfg22 = configparser.ConfigParser()
cfg22.read_string('[s]\nkey = val\n')
buf22 = io.StringIO()
cfg22.write(buf22, space_around_delimiters=False)
print(repr(buf22.getvalue()))

# --- InterpolationMissingOptionError ---
try:
    cfg23 = configparser.ConfigParser()
    cfg23.read_string('[s]\na = %(missing)s\n')
    cfg23.get('s', 'a')
except configparser.InterpolationMissingOptionError:
    print('InterpolationMissingOptionError')

# --- custom delimiters ---
cfg24 = configparser.ConfigParser(delimiters=('=',))
cfg24.read_string('[s]\na = 1\n')
print(cfg24.get('s', 'a'))

# --- custom comment_prefixes ---
cfg25 = configparser.ConfigParser(comment_prefixes=('#',))
cfg25.read_string('[s]\na = 1\n; not a comment = val\n')
print(cfg25.get('s', '; not a comment'))

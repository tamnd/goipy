import netrc
import os
import tempfile

# Helper: write a temp netrc file and return its path.
def write_netrc(content):
    path = os.path.join(tempfile.gettempdir(), '_goipy_test_netrc_tmp')
    with open(path, 'w') as f:
        f.write(content)
    return path

# --- basic machine entry ---
path = write_netrc('machine example.com login user1 password pass1\n')
try:
    n = netrc.netrc(path)
    print(sorted(n.hosts.keys()))
    auth = n.authenticators('example.com')
    print(auth[0], auth[1], auth[2])
finally:
    os.unlink(path)

# --- multiple machines ---
path = write_netrc(
    'machine host1 login a password b\n'
    'machine host2 login c account d password e\n'
)
try:
    n = netrc.netrc(path)
    print(len(n.hosts))
    auth = n.authenticators('host1')
    print(auth[0], auth[1], auth[2])
    auth = n.authenticators('host2')
    print(auth[0], auth[1], auth[2])
finally:
    os.unlink(path)

# --- default fallback ---
path = write_netrc(
    'machine known.com login user password pass\n'
    'default login anon password guest\n'
)
try:
    n = netrc.netrc(path)
    auth = n.authenticators('unknown.com')
    print(auth[0], auth[1], auth[2])
    auth = n.authenticators('known.com')
    print(auth[0], auth[1], auth[2])
finally:
    os.unlink(path)

# --- no matching host, no default ---
path = write_netrc('machine only.com login u password p\n')
try:
    n = netrc.netrc(path)
    print(n.authenticators('missing.com'))
finally:
    os.unlink(path)

# --- comments ---
path = write_netrc(
    '# this is a comment\n'
    'machine ex.com login u # inline ignored\n'
    'password p\n'
)
try:
    n = netrc.netrc(path)
    auth = n.authenticators('ex.com')
    print(auth[0], auth[2])
finally:
    os.unlink(path)

# --- quoted values ---
path = write_netrc('machine q.com login "my user" password "my pass"\n')
try:
    n = netrc.netrc(path)
    auth = n.authenticators('q.com')
    print(auth[0], auth[2])
finally:
    os.unlink(path)

# --- macdef ---
path = write_netrc(
    'macdef upload\n'
    'binary\n'
    'put file.txt\n'
    '\n'
    'machine m.com login u password p\n'
)
try:
    n = netrc.netrc(path)
    print(list(n.macros.keys()))
    print(n.macros['upload'])
    auth = n.authenticators('m.com')
    print(auth[0], auth[2])
finally:
    os.unlink(path)

# --- user synonym for login ---
path = write_netrc('machine s.com user alice password secret\n')
try:
    n = netrc.netrc(path)
    auth = n.authenticators('s.com')
    print(auth[0], auth[2])
finally:
    os.unlink(path)

# --- NetrcParseError on bad toplevel token ---
path = write_netrc('badtoken\n')
try:
    n = netrc.netrc(path)
except netrc.NetrcParseError as e:
    print(type(e).__name__)
    print(e.msg)
    print(e.lineno)
finally:
    os.unlink(path)

# --- NetrcParseError bad follower token ---
path = write_netrc('machine bad.com unknown_field value\n')
try:
    n = netrc.netrc(path)
except netrc.NetrcParseError as e:
    print(type(e).__name__)
finally:
    os.unlink(path)

# --- __repr__ contains machine entries ---
path = write_netrc('machine r.com login ru password rp\n')
try:
    n = netrc.netrc(path)
    rep = repr(n)
    print('machine r.com' in rep)
    print('login ru' in rep)
    print('password rp' in rep)
finally:
    os.unlink(path)

# --- empty file ---
path = write_netrc('')
try:
    n = netrc.netrc(path)
    print(n.hosts)
    print(n.macros)
finally:
    os.unlink(path)

# --- multiline macro ---
path = write_netrc(
    'macdef init\n'
    'cd /pub\n'
    'mget *.gz\n'
    '\n'
)
try:
    n = netrc.netrc(path)
    print(n.macros['init'])
finally:
    os.unlink(path)

# --- account field ---
path = write_netrc('machine a.com login u account acc password p\n')
try:
    n = netrc.netrc(path)
    auth = n.authenticators('a.com')
    print(auth[0], auth[1], auth[2])
finally:
    os.unlink(path)

# --- multiple machines, authenticators for each ---
path = write_netrc(
    'machine x.com login xu password xp\n'
    'machine y.com login yu password yp\n'
    'machine z.com login zu password zp\n'
)
try:
    n = netrc.netrc(path)
    for host in ['x.com', 'y.com', 'z.com']:
        auth = n.authenticators(host)
        print(auth[0], auth[2])
finally:
    os.unlink(path)

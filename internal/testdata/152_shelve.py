import shelve
import os
import tempfile

# ===== module attributes =====
print(hasattr(shelve, 'open'))           # True
print(hasattr(shelve, 'Shelf'))          # True
print(hasattr(shelve, 'DEFAULT_PROTOCOL'))  # True
print(isinstance(shelve.DEFAULT_PROTOCOL, int))  # True

# ===== basic open / set / get / close =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'test')

    db = shelve.open(path)
    db['x'] = 42
    db['y'] = 'hello'
    db['z'] = [1, 2, 3]
    db.close()

    db2 = shelve.open(path)
    print(db2['x'])       # 42
    print(db2['y'])       # hello
    print(db2['z'])       # [1, 2, 3]
    db2.close()

# ===== context manager =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'cm')
    with shelve.open(path) as db:
        db['a'] = True
        db['b'] = None
    with shelve.open(path) as db:
        print(db['a'])           # True
        print(db['b'] is None)   # True

# ===== contains / len / del =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'cl')
    with shelve.open(path) as db:
        db['p'] = 10
        db['q'] = 20
        print('p' in db)   # True
        print('r' in db)   # False
        print(len(db))     # 2
        del db['p']
        print('p' in db)   # False
        print(len(db))     # 1

# ===== all picklable types =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'types')
    with shelve.open(path) as db:
        db['none']   = None
        db['bool_t'] = True
        db['bool_f'] = False
        db['int']    = -999
        db['big']    = 2**40
        db['float']  = 3.14
        db['str']    = 'unicode \u00e9'
        db['bytes']  = b'\x00\xff'
        db['list']   = [1, 'a', None]
        db['tuple']  = (1, 2, 3)
        db['dict']   = {'k': 'v'}
        db['nested'] = {'a': [1, (2,), {'b': True}]}

    with shelve.open(path) as db:
        print(db['none'] is None)    # True
        print(db['bool_t'] is True)  # True
        print(db['bool_f'] is False) # True
        print(db['int'])             # -999
        print(db['big'])             # 1099511627776
        print(db['float'])           # 3.14
        print(db['str'])             # unicode é
        print(db['bytes'])           # b'\x00\xff'
        print(db['list'])            # [1, 'a', None]
        print(db['tuple'])           # (1, 2, 3)
        print(db['dict'])            # {'k': 'v'}
        print(db['nested'])          # {'a': [1, (2,), {'b': True}]}

# ===== set and frozenset =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'sets')
    with shelve.open(path) as db:
        db['s'] = {1, 2, 3}
        db['fs'] = frozenset([4, 5])
    with shelve.open(path) as db:
        print(type(db['s']).__name__)       # set
        print(sorted(db['s']))              # [1, 2, 3]
        print(type(db['fs']).__name__)      # frozenset
        print(sorted(db['fs']))             # [4, 5]

# ===== keys / values / items =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'kvi')
    with shelve.open(path) as db:
        db['one'] = 1
        db['two'] = 2
        print(sorted(db.keys()))            # ['one', 'two']
        print(sorted(db.values()))          # [1, 2]
        print(sorted(db.items()))           # [('one', 1), ('two', 2)]

# ===== iteration =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'iter')
    with shelve.open(path) as db:
        db['a'] = 1
        db['b'] = 2
        print(sorted(db))                   # ['a', 'b']

# ===== get / pop / setdefault / update =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'methods')
    with shelve.open(path) as db:
        db['m'] = 99
        print(db.get('m'))           # 99
        print(db.get('missing'))     # None
        print(db.get('missing', 0))  # 0

        v = db.pop('m')
        print(v)                     # 99
        print('m' in db)             # False

        print(db.setdefault('new', 42))  # 42
        print(db['new'])                 # 42
        print(db.setdefault('new', 99))  # 42 (already set)

        db.update({'aa': 1, 'bb': 2})
        print(sorted(db.keys()))         # ['aa', 'bb', 'new']

# ===== pop with / without default =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'pop')
    with shelve.open(path) as db:
        print(db.pop('nope', 'default'))  # default
        try:
            db.pop('nope')
        except KeyError:
            print('KeyError on pop missing')  # KeyError on pop missing

# ===== KeyError on missing get via [] =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'kerr')
    with shelve.open(path) as db:
        try:
            _ = db['no_such']
        except KeyError:
            print('KeyError missing key')     # KeyError missing key

# ===== flag='n' always starts fresh =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'flagn')
    with shelve.open(path, flag='c') as db:
        db['old'] = 'data'
    with shelve.open(path, flag='n') as db:
        print(len(db))       # 0
        print('old' in db)   # False

# ===== flag='r' read-only =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'flagr')
    with shelve.open(path, flag='c') as db:
        db['k'] = 'v'
    with shelve.open(path, flag='r') as db:
        print(db['k'])       # v
        try:
            db['k'] = 'new'
        except Exception:
            print('error on read-only write')  # error on read-only write

# ===== sync() =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'sync')
    db = shelve.open(path)
    db['s'] = 'synced'
    db.sync()
    db['s2'] = 'also'
    db.close()
    with shelve.open(path) as db2:
        print(db2['s'])      # synced
        print(db2['s2'])     # also

# ===== writeback=True =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'wb')
    with shelve.open(path, writeback=True) as db:
        db['lst'] = [1, 2, 3]
        db['lst'].append(4)
    with shelve.open(path) as db2:
        print(db2['lst'])    # [1, 2, 3, 4]

# ===== operations on closed shelf raise ValueError =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'closed')
    db = shelve.open(path)
    db.close()
    try:
        _ = db['x']
    except ValueError:
        print('ValueError on closed get')   # ValueError on closed get
    try:
        db['x'] = 1
    except ValueError:
        print('ValueError on closed set')   # ValueError on closed set
    try:
        len(db)
    except ValueError:
        print('ValueError on closed len')   # ValueError on closed len

# ===== multiple shelves simultaneously =====
with tempfile.TemporaryDirectory() as tmpdir:
    pa = os.path.join(tmpdir, 'a')
    pb = os.path.join(tmpdir, 'b')
    with shelve.open(pa) as da:
        with shelve.open(pb) as db:
            da['x'] = 1
            db['x'] = 2
    with shelve.open(pa) as da:
        with shelve.open(pb) as db:
            print(da['x'])   # 1
            print(db['x'])   # 2

# ===== protocol kwarg =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'proto')
    with shelve.open(path, protocol=2) as db:
        db['v'] = [1, 2, 3]
    with shelve.open(path) as db:
        print(db['v'])       # [1, 2, 3]

# ===== del raises KeyError on missing =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'del')
    with shelve.open(path) as db:
        try:
            del db['nope']
        except KeyError:
            print('KeyError del missing')    # KeyError del missing

print('done')                               # done

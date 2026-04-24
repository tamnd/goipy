import dbm
import dbm.sqlite3
import os
import tempfile

# ===== module attributes =====
print(hasattr(dbm, 'open'))        # True
print(hasattr(dbm, 'whichdb'))     # True
print(hasattr(dbm, 'error'))       # True
print(isinstance(dbm.error, tuple))  # True
print(len(dbm.error) >= 1)         # True

# ===== dbm.sqlite3 submodule =====
print(hasattr(dbm.sqlite3, 'open'))   # True
print(hasattr(dbm.sqlite3, 'error'))  # True

# ===== basic open / set / get / close =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'test')

    db = dbm.open(path, 'c')
    db['key'] = 'value'
    db['num'] = b'42'
    db.close()

    db2 = dbm.open(path, 'r')
    print(db2['key'])           # b'value'
    print(db2['num'])           # b'42'
    db2.close()

# ===== context manager =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'cm')
    with dbm.open(path, 'c') as db:
        db['a'] = 'alpha'
        db['b'] = b'beta'
    with dbm.open(path, 'r') as db:
        print(db['a'])          # b'alpha'
        print(db['b'])          # b'beta'

# ===== str and bytes keys are interchangeable =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'keys')
    with dbm.open(path, 'c') as db:
        db['str_key'] = 'str_val'
        db[b'bytes_key'] = b'bytes_val'
    with dbm.open(path, 'r') as db:
        print(db['str_key'])         # b'str_val'
        print(db[b'bytes_key'])      # b'bytes_val'
        print(db[b'str_key'])        # b'str_val'  (str key encoded same as bytes)
        print('str_key' in db)       # True
        print(b'bytes_key' in db)    # True
        print('missing' in db)       # False

# ===== keys() returns list of bytes =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'kl')
    with dbm.open(path, 'c') as db:
        db['x'] = '1'
        db['y'] = '2'
        ks = db.keys()
        print(isinstance(ks, list))  # True
        print(sorted(ks))            # [b'x', b'y']

# ===== get() =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'get')
    with dbm.open(path, 'c') as db:
        db['k'] = 'v'
        print(db.get('k'))           # b'v'
        print(db.get('missing'))     # None
        print(db.get('missing', b'default'))  # b'default'

# ===== setdefault() =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'sd')
    with dbm.open(path, 'c') as db:
        db['existing'] = 'value'
        r1 = db.setdefault('existing', b'other')
        r2 = db.setdefault('new_key', b'created')
        print(r1)   # b'value'
        print(r2)   # b'created'
        print(db['new_key'])  # b'created'

# ===== del key =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'del')
    with dbm.open(path, 'c') as db:
        db['d'] = 'data'
        del db['d']
        print('d' in db)     # False
        try:
            del db['missing']
        except KeyError:
            print('KeyError del missing')  # KeyError del missing

# ===== clear() =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'cl')
    with dbm.open(path, 'c') as db:
        db['a'] = '1'
        db['b'] = '2'
        db.clear()
        print(db.keys())     # []

# ===== flag='n' starts fresh =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'flagn')
    with dbm.open(path, 'c') as db:
        db['old'] = 'data'
    with dbm.open(path, 'n') as db:
        print(db.keys())     # []

# ===== flag='r' prevents writes =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'flagr')
    with dbm.open(path, 'c') as db:
        db['k'] = 'v'
    with dbm.open(path, 'r') as db:
        print(db['k'])       # b'v'
        try:
            db['new'] = 'val'
        except Exception:
            print('error on read-only write')  # error on read-only write

# ===== persist across close/reopen =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'persist')
    with dbm.open(path, 'c') as db:
        db['p1'] = 'first'
        db['p2'] = 'second'
    with dbm.open(path, 'r') as db:
        print(db['p1'])      # b'first'
        print(db['p2'])      # b'second'

# ===== KeyError on missing key =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'kerr')
    with dbm.open(path, 'c') as db:
        try:
            _ = db['no_such']
        except KeyError:
            print('KeyError missing')      # KeyError missing

# ===== operations on closed db raise error =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'closed')
    db = dbm.open(path, 'c')
    db.close()
    try:
        _ = db['x']
    except Exception:
        print('error on closed get')       # error on closed get
    try:
        db['x'] = 'y'
    except Exception:
        print('error on closed set')       # error on closed set

# ===== whichdb =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'wdb')

    # missing file → None
    print(dbm.whichdb(path) is None)      # True

    # after creating → 'dbm.sqlite3'
    with dbm.open(path, 'c') as db:
        db['x'] = '1'
    print(dbm.whichdb(path))              # dbm.sqlite3

    # non-dbm file → ''
    other = os.path.join(tmpdir, 'notdbm')
    with open(other, 'w') as f:
        f.write('garbage')
    print(dbm.whichdb(other))             # (empty string or None)

# ===== dbm.sqlite3 direct open =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'sq')
    with dbm.sqlite3.open(path, 'c') as db:
        db['sq_key'] = 'sq_val'
    with dbm.sqlite3.open(path, 'r') as db:
        print(db['sq_key'])               # b'sq_val'
        print(db.keys())                  # [b'sq_key']

# ===== update via write flag =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'upd')
    with dbm.open(path, 'c') as db:
        db['orig'] = 'first'
    with dbm.open(path, 'w') as db:
        db['orig'] = 'updated'
        db['new'] = 'added'
    with dbm.open(path, 'r') as db:
        print(db['orig'])     # b'updated'
        print(db['new'])      # b'added'

print('done')                 # done

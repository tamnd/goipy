import tempfile
import os

# ===== gettempdir / gettempdirb =====
d = tempfile.gettempdir()
print(isinstance(d, str))          # True
print(os.path.isdir(d))            # True

db = tempfile.gettempdirb()
print(isinstance(db, bytes))       # True
print(db == d.encode())            # True

# ===== gettempprefix / gettempprefixb =====
print(tempfile.gettempprefix())    # tmp
print(tempfile.gettempprefixb())   # b'tmp'

# ===== gettempdir caches in tempdir =====
print(isinstance(tempfile.tempdir, str))  # True  (set by gettempdir() above)
old_td = tempfile.tempdir
tempfile.tempdir = tempfile.gettempdir()
print(isinstance(tempfile.tempdir, str))  # True
tempfile.tempdir = None            # reset

# ===== mktemp (deprecated but callable) =====
t = tempfile.mktemp()
print(isinstance(t, str))          # True
print(not os.path.exists(t))       # True  (deleted immediately)

t2 = tempfile.mktemp(suffix='.bak', prefix='my_')
print(os.path.basename(t2).startswith('my_'))   # True
print(os.path.basename(t2).endswith('.bak'))     # True

# ===== mkstemp =====
fd, path = tempfile.mkstemp()
print(isinstance(fd, int))         # True
print(os.path.isfile(path))        # True
os.close(fd)
os.unlink(path)

fd2, path2 = tempfile.mkstemp(suffix='.txt', prefix='test_')
print(path2.endswith('.txt'))                       # True
print(os.path.basename(path2).startswith('test_'))  # True
os.close(fd2)
os.unlink(path2)

# ===== mkdtemp =====
dp = tempfile.mkdtemp()
print(os.path.isdir(dp))           # True
os.rmdir(dp)

dp2 = tempfile.mkdtemp(suffix='_dir', prefix='mydir_')
print(os.path.basename(dp2).startswith('mydir_'))  # True
print(os.path.basename(dp2).endswith('_dir'))       # True
os.rmdir(dp2)

# ===== TemporaryFile (binary, default) =====
with tempfile.TemporaryFile() as f:
    print(f.writable())            # True
    print(f.readable())            # True
    print(f.seekable())            # True
    print(f.closed)                # False
    f.write(b'hello world')
    f.seek(0)
    print(f.read())                # b'hello world'
    print(f.tell())                # 11
    f.seek(6)
    print(f.read())                # b'world'
    f.seek(0, 2)                   # seek to end
    print(f.tell())                # 11
    f.seek(-5, 2)                  # seek 5 from end
    print(f.read())                # b'world'
print(f.closed)                    # True

# ===== TemporaryFile (text mode) =====
with tempfile.TemporaryFile(mode='w+') as f:
    f.write('hello text')
    f.seek(0)
    print(f.read())                # hello text

# ===== NamedTemporaryFile (binary, delete=True default) =====
with tempfile.NamedTemporaryFile() as f:
    saved_name = f.name
    print(isinstance(f.name, str))     # True
    print(os.path.isfile(f.name))      # True
    f.write(b'named data')
    f.seek(0)
    print(f.read())                    # b'named data'
    print(f.closed)                    # False
print(os.path.exists(saved_name))      # False  (deleted on close)

# ===== NamedTemporaryFile (suffix/prefix) =====
with tempfile.NamedTemporaryFile(suffix='.dat', prefix='pfx_') as f:
    print(os.path.basename(f.name).startswith('pfx_'))  # True
    print(f.name.endswith('.dat'))                       # True

# ===== NamedTemporaryFile (delete=False) =====
f = tempfile.NamedTemporaryFile(delete=False)
kept_name = f.name
f.write(b'keep me')
f.close()
print(os.path.isfile(kept_name))       # True
os.unlink(kept_name)

# ===== NamedTemporaryFile (text mode) =====
with tempfile.NamedTemporaryFile(mode='w+') as f:
    f.write('text mode')
    f.seek(0)
    print(f.read())                    # text mode

# ===== SpooledTemporaryFile (binary) =====
with tempfile.SpooledTemporaryFile() as f:
    print(f.writable())                # True
    print(f.readable())                # True
    print(f.seekable())                # True
    print(f.closed)                    # False
    f.write(b'spooled')
    f.seek(0)
    print(f.read())                    # b'spooled'
    print(f.tell())                    # 7
    f.seek(0)
    f.write(b'SPOOLED')
    f.seek(0)
    print(f.read())                    # b'SPOOLED'
    f.rollover()
    f.seek(0)
    print(f.read())                    # b'SPOOLED'
print(f.closed)                        # True

# ===== SpooledTemporaryFile (text mode) =====
with tempfile.SpooledTemporaryFile(mode='w+') as f:
    f.write('text spool')
    f.seek(0)
    print(f.read())                    # text spool

# ===== SpooledTemporaryFile max_size param =====
with tempfile.SpooledTemporaryFile(max_size=10) as f:
    f.write(b'small')
    f.seek(0)
    print(f.read())                    # b'small'

# ===== TemporaryDirectory (context manager) =====
with tempfile.TemporaryDirectory() as d:
    print(os.path.isdir(d))           # True
    td_name = d
print(os.path.exists(td_name))        # False

# ===== TemporaryDirectory .name =====
td = tempfile.TemporaryDirectory()
print(os.path.isdir(td.name))         # True
td_path = td.name
td.cleanup()
print(os.path.exists(td_path))        # False

# ===== TemporaryDirectory suffix/prefix =====
td2 = tempfile.TemporaryDirectory(suffix='_sfx', prefix='pre_')
print(os.path.basename(td2.name).startswith('pre_'))  # True
print(os.path.basename(td2.name).endswith('_sfx'))    # True
td2.cleanup()

# ===== TemporaryDirectory delete=False =====
# delete=False: cleanup() still deletes; __exit__ does NOT delete.
td3 = tempfile.TemporaryDirectory(delete=False)
td3_path = td3.name
print(os.path.isdir(td3_path))        # True
td3.cleanup()                         # always deletes
print(os.path.exists(td3_path))       # False (cleanup() always deletes)

# delete=False context manager: __exit__ does not clean up
with tempfile.TemporaryDirectory(delete=False) as td4_path:
    pass
print(os.path.isdir(td4_path))        # True (not deleted by __exit__)
os.rmdir(td4_path)

print('done')

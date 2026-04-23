import pathlib
import tempfile

# ===== PurePosixPath basics =====
pp = pathlib.PurePosixPath('/usr/local/bin/python')
print(str(pp))               # /usr/local/bin/python
print(pp.name)               # python
print(pp.stem)               # python
print(pp.suffix)             # (empty — no extension)
print(list(pp.suffixes))     # []
print(str(pp.parent))        # /usr/local/bin
print(pp.drive)              # (empty on POSIX)
print(pp.root)               # /
print(pp.anchor)             # /
print(pp.parts)              # ('/', 'usr', 'local', 'bin', 'python')
print(pp.is_absolute())      # True

# relative path
pr = pathlib.PurePosixPath('a/b/c')
print(str(pr))               # a/b/c
print(pr.is_absolute())      # False
print(pr.parts)              # ('a', 'b', 'c')

# PurePosixPath repr
print(repr(pathlib.PurePosixPath('/tmp')))   # PurePosixPath('/tmp')

# ===== Multiple suffixes =====
pt = pathlib.PurePosixPath('/home/user/archive.tar.gz')
print(pt.name)               # archive.tar.gz
print(pt.stem)               # archive.tar
print(pt.suffix)             # .gz
print(list(pt.suffixes))     # ['.tar', '.gz']

# ===== with_name / with_stem / with_suffix =====
p = pathlib.PurePosixPath('/home/user/file.txt')
print(str(p.with_name('new.txt')))    # /home/user/new.txt
print(str(p.with_stem('new')))        # /home/user/new.txt
print(str(p.with_suffix('.md')))      # /home/user/file.md
print(str(p.with_suffix('')))         # /home/user/file

# ===== joinpath / / operator =====
p2 = pathlib.PurePosixPath('/home') / 'user' / 'docs'
print(str(p2))               # /home/user/docs
p3 = pathlib.PurePosixPath('/a/b').joinpath('c', 'd')
print(str(p3))               # /a/b/c/d

# ===== relative_to / is_relative_to =====
p4 = pathlib.PurePosixPath('/a/b/c/d')
print(str(p4.relative_to('/a/b')))   # c/d
print(p4.is_relative_to('/a/b'))     # True
print(p4.is_relative_to('/x'))       # False

# ===== Equality and comparison =====
pa = pathlib.PurePosixPath('/a/b')
pb = pathlib.PurePosixPath('/a/b')
pc = pathlib.PurePosixPath('/a/c')
print(pa == pb)   # True
print(pa == pc)   # False
print(pa < pc)    # True
print(pa > pc)    # False

# ===== as_posix =====
print(pathlib.PurePosixPath('/home/user/file.txt').as_posix())   # /home/user/file.txt

# ===== parents =====
parents = list(pathlib.PurePosixPath('/a/b/c/d').parents)
print([str(p) for p in parents])   # ['/a/b/c', '/a/b', '/a', '/']

# ===== Path (PosixPath on POSIX) =====
pth = pathlib.Path('/tmp/hello.txt')
print(type(pth).__name__)    # PosixPath
print(repr(pth))             # PosixPath('/tmp/hello.txt')

# ===== match =====
p = pathlib.PurePosixPath('/a/b/file.py')
print(p.match('*.py'))       # True
print(p.match('*.txt'))      # False

# ===== File I/O (tempdir) =====
with tempfile.TemporaryDirectory() as tmpdir:
    base = pathlib.Path(tmpdir)
    print(base.is_dir())     # True
    print(base.exists())     # True

    # write_text / read_text
    f = base / 'hello.txt'
    n = f.write_text('hello world')
    print(n)                 # 11
    print(f.read_text())     # hello world

    # exists, is_file, is_dir
    print(f.exists())        # True
    print(f.is_file())       # True
    print(f.is_dir())        # False

    # write_bytes / read_bytes
    bf = base / 'data.bin'
    bf.write_bytes(b'abc')
    print(bf.read_bytes())   # b'abc'

    # touch
    t = base / 'empty.txt'
    t.touch()
    print(t.exists())        # True
    print(t.is_file())       # True

    # mkdir / is_dir
    sub = base / 'subdir'
    sub.mkdir()
    print(sub.is_dir())      # True

    # mkdir parents=True
    nested = base / 'a' / 'b' / 'c'
    nested.mkdir(parents=True)
    print(nested.is_dir())   # True

    # mkdir exist_ok=True
    sub.mkdir(exist_ok=True)
    print(sub.is_dir())      # True

    # iterdir (sorted for determinism)
    items = sorted([p.name for p in base.iterdir()])
    print(items)             # ['a', 'data.bin', 'empty.txt', 'hello.txt', 'subdir']

    # glob
    (base / 'x.py').write_text('x')
    (base / 'y.py').write_text('y')
    g = sorted([p.name for p in base.glob('*.py')])
    print(g)                 # ['x.py', 'y.py']

    # rglob
    (sub / 'z.py').write_text('z')
    rg = sorted([p.name for p in base.rglob('*.py')])
    print(rg)                # ['x.py', 'y.py', 'z.py']

    # clean up sub for rmdir
    (sub / 'z.py').unlink()
    sub.rmdir()
    print(sub.exists())      # False

    # rename
    f2 = f.rename(base / 'hello2.txt')
    print(f2.name)           # hello2.txt
    print((base / 'hello.txt').exists())  # False
    print((base / 'hello2.txt').exists()) # True

    # unlink
    f2.unlink()
    print(f2.exists())       # False

    # unlink missing_ok
    f2.unlink(missing_ok=True)  # no error

    # unlink missing raises FileNotFoundError
    try:
        f2.unlink()
    except FileNotFoundError:
        print('FileNotFoundError ok')   # FileNotFoundError ok

    # stat
    st = bf.stat()
    print(type(st).__name__)         # stat_result
    print(hasattr(st, 'st_mode'))    # True
    print(hasattr(st, 'st_size'))    # True
    print(st.st_size)                # 3

    # resolve / absolute
    rel = pathlib.Path(tmpdir + '/../' + tmpdir.split('/')[-1])
    abs_p = rel.absolute()
    print(abs_p.is_absolute())       # True

    # with_segments
    ws = pathlib.PurePosixPath('/a/b').with_segments('/x', 'y')
    print(str(ws))                   # /x/y

    # walk
    (sub2 := base / 'walkdir').mkdir()
    (sub2 / 'file1.txt').write_text('1')
    (sub2 / 'inner').mkdir()
    (sub2 / 'inner' / 'file2.txt').write_text('2')
    walked = []
    for dirpath, subdirs, files in sub2.walk():
        walked.append((dirpath.name, sorted(subdirs), sorted(files)))
    walked.sort(key=lambda x: x[0])
    for entry in walked:
        print(entry)
        # ('inner', [], ['file2.txt'])
        # ('walkdir', ['inner'], ['file1.txt'])

# ===== cwd() and home() =====
import os
cwd = pathlib.Path.cwd()
print(cwd == pathlib.Path(os.getcwd()))   # True

home = pathlib.Path.home()
print(home.is_absolute())   # True
print(home.is_dir())        # True

# expanduser
p_home = pathlib.Path('~')
expanded = p_home.expanduser()
print(expanded.is_absolute())  # True

# ===== is_symlink =====
with tempfile.TemporaryDirectory() as tmpdir:
    base = pathlib.Path(tmpdir)
    target = base / 'target.txt'
    target.write_text('hi')
    link = base / 'link.txt'
    link.symlink_to(target)
    print(link.is_symlink())     # True
    print(target.is_symlink())   # False
    print(link.readlink().name)  # target.txt

print('done')

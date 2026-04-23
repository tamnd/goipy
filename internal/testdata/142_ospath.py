import os
import os.path
import tempfile

# ===== Constants =====
print(os.path.sep)           # /
print(os.path.curdir)        # .
print(os.path.pardir)        # ..
print(os.path.extsep)        # .
print(os.path.altsep)        # None
print(os.path.pathsep)       # :
print(os.path.defpath)       # /bin:/usr/bin
print(os.path.devnull)       # /dev/null
print(isinstance(os.path.supports_unicode_filenames, bool))  # True

# ===== isabs =====
print(os.path.isabs('/a/b'))    # True
print(os.path.isabs('a/b'))     # False
print(os.path.isabs(''))        # False

# ===== abspath =====
print(os.path.isabs(os.path.abspath('.')))     # True
print(os.path.isabs(os.path.abspath('/tmp')))  # True

# ===== basename =====
print(os.path.basename('/foo/bar/baz.txt'))  # baz.txt
print(repr(os.path.basename('/foo/bar/')))   # ''
print(repr(os.path.basename('')))            # ''
print(os.path.basename('file.txt'))          # file.txt

# ===== dirname =====
print(os.path.dirname('/foo/bar/baz.txt'))   # /foo/bar
print(os.path.dirname('/foo/bar/'))          # /foo/bar
print(repr(os.path.dirname('file.txt')))     # ''
print(os.path.dirname('/'))                  # /

# ===== split =====
print(os.path.split('/foo/bar/baz.txt'))     # ('/foo/bar', 'baz.txt')
print(os.path.split('/foo/bar/'))            # ('/foo/bar', '')
print(os.path.split(''))                     # ('', '')
print(os.path.split('file.txt'))             # ('', 'file.txt')

# ===== splitext =====
print(os.path.splitext('/foo/bar.txt'))      # ('/foo/bar', '.txt')
print(os.path.splitext('/foo/bar'))          # ('/foo/bar', '')
print(os.path.splitext('/foo/.bar'))         # ('/foo/.bar', '')
print(os.path.splitext('/foo/bar.tar.gz'))   # ('/foo/bar.tar', '.gz')

# ===== splitdrive =====
print(os.path.splitdrive('/foo/bar'))        # ('', '/foo/bar')
print(os.path.splitdrive('foo/bar'))         # ('', 'foo/bar')
print(os.path.splitdrive(''))               # ('', '')

# ===== splitroot =====
print(os.path.splitroot('/foo/bar'))         # ('', '/', 'foo/bar')
print(os.path.splitroot('foo/bar'))          # ('', '', 'foo/bar')
print(os.path.splitroot('/'))                # ('', '/', '')
print(os.path.splitroot(''))                 # ('', '', '')

# ===== join — absolute resets previous components =====
print(os.path.join('/a', 'b', 'c'))          # /a/b/c
print(os.path.join('/a', '/b'))              # /b
print(os.path.join('/a/b', 'c', '/d'))       # /d
print(os.path.join('a', 'b'))                # a/b
print(os.path.join('a', ''))                # a/
print(os.path.join('', 'b'))                # b
print(os.path.join('/a', 'b/'))             # /a/b/

# ===== normpath =====
print(os.path.normpath('/a/b/../c'))         # /a/c
print(os.path.normpath('/a/./b'))            # /a/b
print(os.path.normpath('a//b'))              # a/b
print(os.path.normpath('/'))                 # /
print(os.path.normpath(''))                  # .
print(os.path.normpath('a/b/../../c'))       # c

# ===== normcase =====
print(os.path.normcase('/FOO/Bar'))          # /FOO/Bar  (POSIX: no-op)
print(repr(os.path.normcase('')))            # ''

# ===== commonprefix =====
print(os.path.commonprefix(['/usr/lib', '/usr/local/lib']))  # /usr/l
print(os.path.commonprefix(['/usr/lib', '/usr']))            # /usr
print(repr(os.path.commonprefix(['abc', 'xyz'])))            # ''
print(repr(os.path.commonprefix([])))                        # ''
print(os.path.commonprefix(['abc']))                         # abc

# ===== commonpath =====
print(os.path.commonpath(['/usr/lib', '/usr/local/lib']))    # /usr
print(os.path.commonpath(['/usr/lib', '/usr']))              # /usr
print(os.path.commonpath(['a/b/c', 'a/b/d']))                # a/b

# ===== relpath =====
print(os.path.relpath('/a/b/c', '/a'))      # b/c
print(os.path.relpath('/a/b/c', '/a/b'))    # c
print(os.path.relpath('/a/b', '/a/b/c'))    # ..
print(os.path.relpath('/a/b/c', '/x/y'))    # ../../a/b/c

# ===== expandvars =====
os.environ['_GOIPY_TEST'] = 'hello'
print(os.path.expandvars('$_GOIPY_TEST world'))       # hello world
print(os.path.expandvars('${_GOIPY_TEST}!'))          # hello!
print(os.path.expandvars('no_var'))                   # no_var
print(os.path.expandvars('$_NONEXISTENT_VAR_XYZ'))    # $_NONEXISTENT_VAR_XYZ

# ===== expanduser =====
home = os.path.expanduser('~')
print(os.path.isabs(home))                            # True
print(os.path.expanduser('~/foo') == home + '/foo')   # True
print(os.path.expanduser('/no/expand'))               # /no/expand

# ===== filesystem tests (tempdir) =====
with tempfile.TemporaryDirectory() as tmpdir:
    f = os.path.join(tmpdir, 'file.txt')
    with open(f, 'w') as fp:
        fp.write('hello')

    # exists / isfile / isdir / lexists
    print(os.path.exists(f))              # True
    print(os.path.exists(f + '_nope'))    # False
    print(os.path.isfile(f))              # True
    print(os.path.isdir(f))              # False
    print(os.path.isdir(tmpdir))          # True
    print(os.path.lexists(f))             # True
    print(os.path.lexists(f + '_nope'))   # False

    # getsize
    print(os.path.getsize(f))             # 5

    # timestamps > 0
    print(os.path.getmtime(f) > 0)        # True
    print(os.path.getatime(f) > 0)        # True
    print(os.path.getctime(f) > 0)        # True

    # samefile
    print(os.path.samefile(f, f))         # True
    f2 = os.path.join(tmpdir, 'other.txt')
    with open(f2, 'w') as fp:
        fp.write('world')
    print(os.path.samefile(f, f2))        # False

    # samestat
    st1 = os.stat(f)
    st2 = os.stat(f)
    st3 = os.stat(f2)
    print(os.path.samestat(st1, st2))     # True
    print(os.path.samestat(st1, st3))     # False

    # islink / symlink
    link = os.path.join(tmpdir, 'link.txt')
    os.symlink(f, link)
    print(os.path.islink(link))            # True
    print(os.path.islink(f))              # False
    print(os.path.lexists(link))           # True

    # realpath follows symlinks
    rp = os.path.realpath(link)
    print(os.path.isabs(rp))              # True
    print(os.path.exists(rp))             # True

    # ismount
    print(os.path.ismount(tmpdir))        # False
    print(os.path.ismount('/'))           # True

    # isjunction — always False on POSIX
    print(os.path.isjunction(f))          # False

print('done')

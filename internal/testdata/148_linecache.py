import linecache
import os
import tempfile

# ===== basic getline() =====
with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, 'sample.txt')
    with open(path, 'w') as f:
        f.write('line one\nline two\nline three\n')

    print(repr(linecache.getline(path, 1)))    # 'line one\n'
    print(repr(linecache.getline(path, 2)))    # 'line two\n'
    print(repr(linecache.getline(path, 3)))    # 'line three\n'

    # out-of-bounds line numbers
    print(repr(linecache.getline(path, 0)))    # ''
    print(repr(linecache.getline(path, 4)))    # ''
    print(repr(linecache.getline(path, -1)))   # ''
    print(repr(linecache.getline(path, 999)))  # ''

    # missing file — never raises
    print(repr(linecache.getline('/no/such/file.py', 1)))  # ''
    print(repr(linecache.getline('', 1)))                  # ''

    # ===== file without trailing newline =====
    path2 = os.path.join(tmpdir, 'no_newline.txt')
    with open(path2, 'w') as f:
        f.write('only line')
    print(repr(linecache.getline(path2, 1)))   # 'only line'
    print(repr(linecache.getline(path2, 2)))   # ''

    # ===== empty file =====
    path3 = os.path.join(tmpdir, 'empty.txt')
    with open(path3, 'w') as f:
        pass
    print(repr(linecache.getline(path3, 1)))   # ''

    # ===== caching: same result on repeated calls =====
    r1 = linecache.getline(path, 1)
    r2 = linecache.getline(path, 1)
    print(r1 == r2)                            # True

    # ===== clearcache() =====
    linecache.clearcache()
    # after clear, getline still works (re-reads from disk)
    print(repr(linecache.getline(path, 2)))    # 'line two\n'

    # ===== checkcache() — file unchanged, cache stays valid =====
    linecache.getline(path, 1)   # populate cache
    linecache.checkcache(path)
    print(repr(linecache.getline(path, 1)))    # 'line one\n'

    # checkcache with None checks all entries
    linecache.checkcache()
    print(repr(linecache.getline(path, 3)))    # 'line three\n'

    # ===== checkcache() detects file changes =====
    linecache.clearcache()
    linecache.getline(path, 1)                 # cache it
    with open(path, 'w') as f:
        f.write('updated one\nupdated two\n')
    linecache.checkcache(path)                 # invalidate stale entry
    print(repr(linecache.getline(path, 1)))    # 'updated one\n'
    print(repr(linecache.getline(path, 2)))    # 'updated two\n'
    print(repr(linecache.getline(path, 3)))    # ''

    # ===== lazycache() =====
    lazy_path = os.path.join(tmpdir, 'lazy.txt')
    with open(lazy_path, 'w') as f:
        f.write('alpha\nbeta\ngamma\n')
    # lazycache registers metadata without reading disk
    linecache.lazycache(lazy_path, {})
    # getline still works after lazycache
    print(repr(linecache.getline(lazy_path, 1)))  # 'alpha\n'
    print(repr(linecache.getline(lazy_path, 2)))  # 'beta\n'
    print(repr(linecache.getline(lazy_path, 3)))  # 'gamma\n'

    # lazycache on non-existent file — getline returns ''
    linecache.lazycache(os.path.join(tmpdir, 'ghost.txt'), {})
    print(repr(linecache.getline(os.path.join(tmpdir, 'ghost.txt'), 1)))  # ''

    # ===== multiline, mixed content =====
    path4 = os.path.join(tmpdir, 'mixed.txt')
    with open(path4, 'w') as f:
        f.write('# comment\ncode = 42\n\nresult = code + 1\n')
    print(repr(linecache.getline(path4, 1)))   # '# comment\n'
    print(repr(linecache.getline(path4, 2)))   # 'code = 42\n'
    print(repr(linecache.getline(path4, 3)))   # '\n'
    print(repr(linecache.getline(path4, 4)))   # 'result = code + 1\n'
    print(repr(linecache.getline(path4, 5)))   # ''

print('done')

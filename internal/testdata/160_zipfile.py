import os
import tempfile
import zipfile

with tempfile.TemporaryDirectory() as tmpdir:
    path = os.path.join(tmpdir, "a.zip")

    # ===== write STORED + DEFLATE =====
    with zipfile.ZipFile(path, 'w', zipfile.ZIP_DEFLATED) as zf:
        zf.writestr('hello.txt', b'hello deflate')
        zf.writestr('stored.bin', b'stored bytes', compress_type=zipfile.ZIP_STORED)
        zf.writestr('from_str.txt', 'string payload')

    # ===== read back =====
    with zipfile.ZipFile(path, 'r') as zf:
        print(sorted(zf.namelist()))                   # ['from_str.txt','hello.txt','stored.bin']
        print(zf.read('hello.txt'))                    # b'hello deflate'
        print(zf.read('stored.bin'))                   # b'stored bytes'
        print(zf.read('from_str.txt'))                 # b'string payload'
        with zf.open('hello.txt') as fp:
            print(fp.read())                           # b'hello deflate'

        info = zf.getinfo('stored.bin')
        print(info.filename)                           # stored.bin
        print(info.file_size)                          # 12
        print(info.compress_type == zipfile.ZIP_STORED)# True
        print(info.is_dir())                           # False

        info2 = zf.getinfo('hello.txt')
        print(info2.compress_type == zipfile.ZIP_DEFLATED)  # True
        print(info2.file_size)                         # 13

        print(len(zf.infolist()))                      # 3
        print(zf.testzip() is None)                    # True

        print(sorted(zf.namelist()))                   # same

    # ===== BZIP2 round-trip =====
    bz_path = os.path.join(tmpdir, "bz.zip")
    with zipfile.ZipFile(bz_path, 'w', zipfile.ZIP_BZIP2) as zf:
        zf.writestr('bz.txt', b'bzip2 payload ' * 5)
    with zipfile.ZipFile(bz_path, 'r') as zf:
        print(zf.read('bz.txt') == b'bzip2 payload ' * 5)   # True
        print(zf.getinfo('bz.txt').compress_type == zipfile.ZIP_BZIP2)  # True

    # ===== LZMA round-trip =====
    lz_path = os.path.join(tmpdir, "lz.zip")
    with zipfile.ZipFile(lz_path, 'w', zipfile.ZIP_LZMA) as zf:
        zf.writestr('lz.txt', b'lzma payload here')
    with zipfile.ZipFile(lz_path, 'r') as zf:
        print(zf.read('lz.txt') == b'lzma payload here')    # True
        print(zf.getinfo('lz.txt').compress_type == zipfile.ZIP_LZMA)  # True

    # ===== directories =====
    dir_path = os.path.join(tmpdir, "dir.zip")
    with zipfile.ZipFile(dir_path, 'w') as zf:
        zf.writestr('mydir/', b'')
        zf.writestr('mydir/file.txt', b'in dir')
    with zipfile.ZipFile(dir_path, 'r') as zf:
        print(zf.getinfo('mydir/').is_dir())           # True
        print(zf.getinfo('mydir/file.txt').is_dir())   # False

    # ===== extract / extractall =====
    out_dir = os.path.join(tmpdir, "out")
    os.mkdir(out_dir)
    with zipfile.ZipFile(path, 'r') as zf:
        zf.extractall(out_dir)
    print(os.path.exists(os.path.join(out_dir, 'hello.txt')))   # True
    with open(os.path.join(out_dir, 'hello.txt'), 'rb') as fp:
        print(fp.read())                               # b'hello deflate'

    out2 = os.path.join(tmpdir, "out2")
    os.mkdir(out2)
    with zipfile.ZipFile(path, 'r') as zf:
        p = zf.extract('stored.bin', out2)
    with open(os.path.join(out2, 'stored.bin'), 'rb') as fp:
        print(fp.read())                               # b'stored bytes'

    # ===== is_zipfile =====
    print(zipfile.is_zipfile(path))                    # True
    garbage = os.path.join(tmpdir, "garbage.bin")
    with open(garbage, 'wb') as fp:
        fp.write(b'not a zip')
    print(zipfile.is_zipfile(garbage))                 # False

    # ===== BadZipFile on invalid input =====
    try:
        zipfile.ZipFile(garbage, 'r')
    except zipfile.BadZipFile:
        print('BadZipFile raised')                     # BadZipFile raised

    # BadZipfile alias
    print(zipfile.BadZipfile is zipfile.BadZipFile)    # True

    # ===== append mode =====
    app_path = os.path.join(tmpdir, "app.zip")
    with zipfile.ZipFile(app_path, 'w') as zf:
        zf.writestr('first.txt', b'first')
    with zipfile.ZipFile(app_path, 'a') as zf:
        zf.writestr('second.txt', b'second')
    with zipfile.ZipFile(app_path, 'r') as zf:
        print(sorted(zf.namelist()))                   # ['first.txt','second.txt']
        print(zf.read('first.txt'))                    # b'first'
        print(zf.read('second.txt'))                   # b'second'

    # ===== ZipInfo construction =====
    zi = zipfile.ZipInfo('a/b.txt')
    print(zi.filename)                                 # a/b.txt
    print(zi.is_dir())                                 # False
    zi2 = zipfile.ZipInfo('d/')
    print(zi2.is_dir())                                # True

    # ===== writestr with ZipInfo =====
    zi_path = os.path.join(tmpdir, "zi.zip")
    with zipfile.ZipFile(zi_path, 'w') as zf:
        info = zipfile.ZipInfo('via_info.txt')
        zf.writestr(info, b'with zipinfo')
    with zipfile.ZipFile(zi_path, 'r') as zf:
        print(zf.read('via_info.txt'))                 # b'with zipinfo'

    # ===== constants =====
    print(zipfile.ZIP_STORED, zipfile.ZIP_DEFLATED, zipfile.ZIP_BZIP2, zipfile.ZIP_LZMA)

print('done')

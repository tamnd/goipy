import io
import os
import tarfile
import tempfile

with tempfile.TemporaryDirectory() as tmpdir:
    # ===== seed some source files =====
    src = os.path.join(tmpdir, "src")
    os.mkdir(src)
    with open(os.path.join(src, "a.txt"), "wb") as f:
        f.write(b"hello from a")
    with open(os.path.join(src, "b.txt"), "wb") as f:
        f.write(b"second file body")
    os.mkdir(os.path.join(src, "sub"))
    with open(os.path.join(src, "sub", "c.txt"), "wb") as f:
        f.write(b"nested")

    # ===== write uncompressed via addfile =====
    path = os.path.join(tmpdir, "plain.tar")
    with tarfile.open(path, "w") as tf:
        ti = tarfile.TarInfo("hello.txt")
        data = b"hello addfile"
        ti.size = len(data)
        tf.addfile(ti, io.BytesIO(data))

        ti2 = tarfile.TarInfo("second.bin")
        data2 = b"second via addfile"
        ti2.size = len(data2)
        tf.addfile(ti2, io.BytesIO(data2))

    with tarfile.open(path, "r") as tf:
        print(sorted(tf.getnames()))                     # ['hello.txt', 'second.bin']
        print(tf.getmember("hello.txt").name)            # hello.txt
        print(tf.getmember("hello.txt").size)            # 13
        print(tf.extractfile("hello.txt").read())        # b'hello addfile'
        print(tf.extractfile("second.bin").read())       # b'second via addfile'
        members = tf.getmembers()
        print(len(members))                              # 2
        print(members[0].isfile())                       # True
        print(members[0].isdir())                        # False

    # ===== write with add() walking a directory =====
    path2 = os.path.join(tmpdir, "tree.tar")
    with tarfile.open(path2, "w") as tf:
        tf.add(src, arcname="root")
    with tarfile.open(path2, "r") as tf:
        names = sorted(tf.getnames())
        print(names)                                     # ['root', 'root/a.txt', 'root/b.txt', 'root/sub', 'root/sub/c.txt']
        print(tf.extractfile("root/a.txt").read())       # b'hello from a'
        print(tf.extractfile("root/sub/c.txt").read())   # b'nested'
        root = tf.getmember("root")
        print(root.isdir())                              # True

    # ===== compressed round-trips =====
    for comp in ("gz", "bz2", "xz"):
        cp = os.path.join(tmpdir, "comp." + comp + ".tar")
        with tarfile.open(cp, "w:" + comp) as tf:
            ti = tarfile.TarInfo("data.txt")
            payload = ("payload for " + comp + " " * 10).encode() * 5
            ti.size = len(payload)
            tf.addfile(ti, io.BytesIO(payload))
        with tarfile.open(cp, "r:" + comp) as tf:
            print(tf.getnames())                         # ['data.txt']
            print(tf.extractfile("data.txt").read() == payload)  # True

    # ===== autodetect with r:* =====
    auto_path = os.path.join(tmpdir, "auto.tar.gz")
    with tarfile.open(auto_path, "w:gz") as tf:
        ti = tarfile.TarInfo("auto.txt")
        payload = b"autodetect me"
        ti.size = len(payload)
        tf.addfile(ti, io.BytesIO(payload))
    with tarfile.open(auto_path, "r:*") as tf:
        print(tf.extractfile("auto.txt").read())         # b'autodetect me'

    # ===== extractall =====
    out_dir = os.path.join(tmpdir, "out")
    os.mkdir(out_dir)
    with tarfile.open(path2, "r") as tf:
        tf.extractall(out_dir)
    with open(os.path.join(out_dir, "root", "a.txt"), "rb") as f:
        print(f.read())                                  # b'hello from a'
    with open(os.path.join(out_dir, "root", "sub", "c.txt"), "rb") as f:
        print(f.read())                                  # b'nested'

    # ===== iteration =====
    with tarfile.open(path, "r") as tf:
        iter_names = sorted([m.name for m in tf])
        print(iter_names)                                # ['hello.txt', 'second.bin']

    # ===== is_tarfile =====
    print(tarfile.is_tarfile(path))                      # True
    print(tarfile.is_tarfile(auto_path))                 # True
    garbage = os.path.join(tmpdir, "garbage.bin")
    with open(garbage, "wb") as f:
        f.write(b"not a tar")
    print(tarfile.is_tarfile(garbage))                   # False

    # ===== ReadError on garbage =====
    try:
        tarfile.open(garbage, "r")
    except tarfile.ReadError:
        print("ReadError raised")                        # ReadError raised
    except tarfile.TarError:
        print("ReadError raised")

    # ===== TarInfo standalone =====
    ti = tarfile.TarInfo("zz.txt")
    print(ti.name)                                       # zz.txt
    print(ti.isfile())                                   # True
    print(ti.isdir())                                    # False

    # ===== directory TarInfo via addfile =====
    dpath = os.path.join(tmpdir, "dir.tar")
    with tarfile.open(dpath, "w") as tf:
        di = tarfile.TarInfo("mydir")
        di.type = tarfile.DIRTYPE
        tf.addfile(di)
        fi = tarfile.TarInfo("mydir/x.txt")
        data = b"inside"
        fi.size = len(data)
        tf.addfile(fi, io.BytesIO(data))
    with tarfile.open(dpath, "r") as tf:
        print(tf.getmember("mydir").isdir())             # True
        print(tf.extractfile("mydir/x.txt").read())      # b'inside'

print('done')

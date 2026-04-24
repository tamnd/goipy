import shutil
import os
import stat
import tempfile

# ===== helpers =====
def make_tree(base):
    """Create a small directory tree under base."""
    os.makedirs(os.path.join(base, 'sub'))
    os.makedirs(os.path.join(base, 'sub', 'deep'))
    for p, data in [
        ('a.txt',          'hello\n'),
        ('b.txt',          'world\n'),
        ('sub/c.txt',      'sub file\n'),
        ('sub/deep/d.txt', 'deep file\n'),
    ]:
        with open(os.path.join(base, p), 'w') as f:
            f.write(data)

with tempfile.TemporaryDirectory() as tmp:

    # ===== copyfileobj() =====
    import io
    src_buf = io.BytesIO(b'binary content here')
    dst_buf = io.BytesIO()
    shutil.copyfileobj(src_buf, dst_buf)
    print(dst_buf.getvalue())                           # b'binary content here'

    src_buf2 = io.StringIO('text content')
    dst_buf2 = io.StringIO()
    shutil.copyfileobj(src_buf2, dst_buf2)
    print(dst_buf2.getvalue())                          # text content

    # ===== copyfile() =====
    src = os.path.join(tmp, 'src.txt')
    dst = os.path.join(tmp, 'dst.txt')
    with open(src, 'w') as f:
        f.write('copyfile test\n')
    shutil.copyfile(src, dst)
    with open(dst) as f:
        print(f.read())                                 # copyfile test\n

    # SameFileError when src == dst
    try:
        shutil.copyfile(src, src)
    except shutil.SameFileError:
        print('SameFileError')                          # SameFileError

    # ===== copymode() =====
    mode_src = os.path.join(tmp, 'mode_src.txt')
    mode_dst = os.path.join(tmp, 'mode_dst.txt')
    with open(mode_src, 'w') as f: f.write('x')
    with open(mode_dst, 'w') as f: f.write('y')
    os.chmod(mode_src, 0o644)
    os.chmod(mode_dst, 0o600)
    shutil.copymode(mode_src, mode_dst)
    m = stat.S_IMODE(os.stat(mode_dst).st_mode)
    print(oct(m))                                       # 0o644

    # ===== copystat() =====
    stat_src = os.path.join(tmp, 'stat_src.txt')
    stat_dst = os.path.join(tmp, 'stat_dst.txt')
    with open(stat_src, 'w') as f: f.write('stat test')
    with open(stat_dst, 'w') as f: f.write('dst')
    os.chmod(stat_src, 0o755)
    shutil.copystat(stat_src, stat_dst)
    m2 = stat.S_IMODE(os.stat(stat_dst).st_mode)
    print(oct(m2))                                      # 0o755

    # ===== copy() — copies content + mode =====
    copy_src = os.path.join(tmp, 'copy_src.txt')
    copy_dst = os.path.join(tmp, 'copy_dst.txt')
    with open(copy_src, 'w') as f: f.write('copy content\n')
    os.chmod(copy_src, 0o644)
    shutil.copy(copy_src, copy_dst)
    with open(copy_dst) as f:
        print(f.read())                                 # copy content\n
    print(oct(stat.S_IMODE(os.stat(copy_dst).st_mode))) # 0o644

    # copy to existing directory — places file inside
    copy_dir = os.path.join(tmp, 'copy_dir')
    os.makedirs(copy_dir)
    result = shutil.copy(copy_src, copy_dir)
    print(os.path.basename(result))                     # copy_src.txt

    # ===== copy2() — copies content + full stat =====
    c2_src = os.path.join(tmp, 'c2_src.txt')
    c2_dst = os.path.join(tmp, 'c2_dst.txt')
    with open(c2_src, 'w') as f: f.write('copy2 content\n')
    os.chmod(c2_src, 0o600)
    shutil.copy2(c2_src, c2_dst)
    with open(c2_dst) as f:
        print(f.read())                                 # copy2 content\n
    print(oct(stat.S_IMODE(os.stat(c2_dst).st_mode)))  # 0o600

    # ===== copytree() =====
    tree_src = os.path.join(tmp, 'tree_src')
    tree_dst = os.path.join(tmp, 'tree_dst')
    make_tree(tree_src)
    shutil.copytree(tree_src, tree_dst)
    # all files exist
    for p in ['a.txt', 'b.txt', 'sub/c.txt', 'sub/deep/d.txt']:
        print(os.path.exists(os.path.join(tree_dst, p)))  # True x4

    # contents preserved
    with open(os.path.join(tree_dst, 'a.txt')) as f:
        print(f.read())                                 # hello\n
    with open(os.path.join(tree_dst, 'sub', 'deep', 'd.txt')) as f:
        print(f.read())                                 # deep file\n

    # dirs_exist_ok=True allows dst to exist
    shutil.copytree(tree_src, tree_dst, dirs_exist_ok=True)
    print(os.path.exists(os.path.join(tree_dst, 'a.txt')))  # True

    # ignore_patterns
    tree_dst2 = os.path.join(tmp, 'tree_dst2')
    shutil.copytree(tree_src, tree_dst2, ignore=shutil.ignore_patterns('*.txt'))
    print(os.path.exists(os.path.join(tree_dst2, 'a.txt')))  # False
    print(os.path.isdir(os.path.join(tree_dst2, 'sub')))     # True

    # ===== rmtree() =====
    rm_dir = os.path.join(tmp, 'rm_dir')
    make_tree(rm_dir)
    print(os.path.isdir(rm_dir))                        # True
    shutil.rmtree(rm_dir)
    print(os.path.exists(rm_dir))                       # False

    # ignore_errors=True — no exception for missing dir
    shutil.rmtree('/nonexistent_path_xyz', ignore_errors=True)
    print('rmtree ignore_errors ok')                    # rmtree ignore_errors ok

    # ===== move() =====
    mv_src = os.path.join(tmp, 'mv_src.txt')
    mv_dst = os.path.join(tmp, 'mv_dst.txt')
    with open(mv_src, 'w') as f: f.write('move me\n')
    shutil.move(mv_src, mv_dst)
    print(os.path.exists(mv_src))                       # False
    with open(mv_dst) as f:
        print(f.read())                                 # move me\n

    # move directory
    mv_dir_src = os.path.join(tmp, 'mv_dir_src')
    mv_dir_dst = os.path.join(tmp, 'mv_dir_dst')
    make_tree(mv_dir_src)
    shutil.move(mv_dir_src, mv_dir_dst)
    print(os.path.exists(mv_dir_src))                   # False
    print(os.path.isdir(mv_dir_dst))                    # True
    print(os.path.exists(os.path.join(mv_dir_dst, 'a.txt')))  # True

    # move into existing directory
    mv_into_src = os.path.join(tmp, 'mv_into_src.txt')
    mv_into_dir = os.path.join(tmp, 'mv_into_dir')
    with open(mv_into_src, 'w') as f: f.write('into dir\n')
    os.makedirs(mv_into_dir)
    result2 = shutil.move(mv_into_src, mv_into_dir)
    print(os.path.basename(result2))                    # mv_into_src.txt
    print(os.path.exists(mv_into_src))                  # False

    # ===== disk_usage() =====
    du = shutil.disk_usage(tmp)
    print(du.total > 0)                                 # True
    print(du.used > 0)                                  # True
    print(du.free > 0)                                  # True
    print(du.total >= du.used + du.free)                # True

    # named tuple fields
    print(isinstance(du.total, int))                    # True
    print(isinstance(du.used, int))                     # True
    print(isinstance(du.free, int))                     # True

    # ===== which() =====
    result_ls = shutil.which('ls')
    print(result_ls is not None)                        # True
    print(os.path.isabs(result_ls))                     # True

    result_missing = shutil.which('no_such_cmd_xyz_abc')
    print(result_missing is None)                       # True

    # ===== get_terminal_size() =====
    ts = shutil.get_terminal_size()
    print(isinstance(ts.columns, int))                  # True
    print(isinstance(ts.lines, int))                    # True
    print(ts.columns > 0)                               # True
    print(ts.lines > 0)                                 # True

    ts2 = shutil.get_terminal_size(fallback=(100, 50))
    print(ts2.columns > 0)                              # True
    print(ts2.lines > 0)                                # True

    # ===== ignore_patterns() =====
    ip = shutil.ignore_patterns('*.txt', '*.py')
    # Returns a function that, given (dir, names), returns set of names to ignore
    names = ['a.txt', 'b.py', 'c.md', 'd.txt', 'e.go']
    ignored = ip('/some/dir', names)
    print(sorted(ignored))                              # ['a.txt', 'b.py', 'd.txt']

    ip2 = shutil.ignore_patterns('sub*')
    ignored2 = ip2('/dir', ['sub', 'sub2', 'other', 'subdir'])
    print(sorted(ignored2))                             # ['sub', 'sub2', 'subdir']

    # ===== get_archive_formats() =====
    fmts = shutil.get_archive_formats()
    print(isinstance(fmts, list))                       # True
    names_only = [f[0] for f in fmts]
    print('zip' in names_only)                          # True
    print('tar' in names_only)                          # True
    print('gztar' in names_only)                        # True

    # ===== get_unpack_formats() =====
    ufmts = shutil.get_unpack_formats()
    print(isinstance(ufmts, list))                      # True
    unames = [f[0] for f in ufmts]
    print('zip' in unames)                              # True
    print('tar' in unames)                              # True

    # ===== make_archive() + unpack_archive() =====
    arch_src = os.path.join(tmp, 'arch_src')
    os.makedirs(arch_src)
    with open(os.path.join(arch_src, 'hello.txt'), 'w') as f:
        f.write('archive content\n')
    with open(os.path.join(arch_src, 'world.txt'), 'w') as f:
        f.write('more content\n')

    arch_base = os.path.join(tmp, 'myarchive')
    arch_path = shutil.make_archive(arch_base, 'zip', root_dir=arch_src)
    print(os.path.exists(arch_path))                    # True
    print(arch_path.endswith('.zip'))                   # True

    arch_extract = os.path.join(tmp, 'arch_extract')
    os.makedirs(arch_extract)
    shutil.unpack_archive(arch_path, arch_extract)
    print(os.path.exists(os.path.join(arch_extract, 'hello.txt')))  # True
    print(os.path.exists(os.path.join(arch_extract, 'world.txt')))  # True
    with open(os.path.join(arch_extract, 'hello.txt')) as f:
        print(f.read())                                 # archive content\n

    # tar archive
    tar_base = os.path.join(tmp, 'mytar')
    tar_path = shutil.make_archive(tar_base, 'gztar', root_dir=arch_src)
    print(os.path.exists(tar_path))                     # True
    print(tar_path.endswith('.tar.gz'))                 # True

    tar_extract = os.path.join(tmp, 'tar_extract')
    os.makedirs(tar_extract)
    shutil.unpack_archive(tar_path, tar_extract)
    print(os.path.exists(os.path.join(tar_extract, 'hello.txt')))   # True
    with open(os.path.join(tar_extract, 'hello.txt')) as f:
        print(f.read())                                 # archive content\n

    # unpack with explicit format
    arch_extract2 = os.path.join(tmp, 'arch_extract2')
    os.makedirs(arch_extract2)
    shutil.unpack_archive(arch_path, arch_extract2, format='zip')
    print(os.path.exists(os.path.join(arch_extract2, 'hello.txt')))  # True

    # ===== Error / SameFileError / ReadError attributes =====
    print(issubclass(shutil.SameFileError, OSError))    # True
    print(issubclass(shutil.Error, OSError))            # True

print('done')

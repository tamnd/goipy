import filecmp
import os
import tempfile

# ===== Constants =====
print(filecmp.BUFSIZE)             # 8192
print(filecmp.DEFAULT_IGNORES)     # the list

with tempfile.TemporaryDirectory() as tmpdir:
    os.chdir(tmpdir)

    # Build directory tree:
    #   a/  same.txt  diff.txt  left_only.txt  subdir/sub.txt  subdir/sub_diff.txt
    #   b/  same.txt  diff.txt  right_only.txt subdir/sub.txt  subdir/sub_diff.txt
    os.makedirs('a/subdir')
    os.makedirs('b/subdir')

    for path, content in [
        ('a/same.txt',         'identical\n'),
        ('b/same.txt',         'identical\n'),
        ('a/diff.txt',         'version A\n'),
        ('b/diff.txt',         'version B\n'),
        ('a/left_only.txt',    'left\n'),
        ('b/right_only.txt',   'right\n'),
        ('a/subdir/sub.txt',   'sub\n'),
        ('b/subdir/sub.txt',   'sub\n'),
        ('a/subdir/sub_diff.txt', 'aaa\n'),
        ('b/subdir/sub_diff.txt', 'bbb\n'),
    ]:
        with open(path, 'w') as f:
            f.write(content)

    # ===== cmp() =====
    print(filecmp.cmp('a/same.txt', 'b/same.txt'))         # True  (shallow: same sig)
    print(filecmp.cmp('a/same.txt', 'b/same.txt', False))  # True  (content)
    print(filecmp.cmp('a/diff.txt', 'b/diff.txt'))         # False (different sizes)
    print(filecmp.cmp('a/diff.txt', 'b/diff.txt', False))  # False (different content)
    print(filecmp.cmp('a/same.txt', 'a/same.txt'))         # True  (same file)

    # shallow=True but different mtime with same content → still True via content check
    # (both size and content match, cache miss forces do_cmp)

    # ===== clear_cache() =====
    filecmp.clear_cache()
    print(filecmp.cmp('a/same.txt', 'b/same.txt'))         # True after cache clear

    # ===== cmpfiles() =====
    match, mismatch, errors = filecmp.cmpfiles(
        'a', 'b', ['same.txt', 'diff.txt'])
    print(sorted(match))      # ['same.txt']
    print(sorted(mismatch))   # ['diff.txt']
    print(sorted(errors))     # []

    # cmpfiles — missing file → errors
    match2, mismatch2, errors2 = filecmp.cmpfiles(
        'a', 'b', ['same.txt', 'missing.txt'])
    print(sorted(match2))     # ['same.txt']
    print(sorted(mismatch2))  # []
    print(sorted(errors2))    # ['missing.txt']

    # cmpfiles — empty common list
    m3, mm3, e3 = filecmp.cmpfiles('a', 'b', [])
    print(m3, mm3, e3)        # [] [] []

    # ===== dircmp =====
    dc = filecmp.dircmp('a', 'b')

    print(sorted(dc.left_list))     # ['diff.txt', 'left_only.txt', 'same.txt', 'subdir']
    print(sorted(dc.right_list))    # ['diff.txt', 'right_only.txt', 'same.txt', 'subdir']
    print(sorted(dc.common))        # ['diff.txt', 'same.txt', 'subdir']
    print(sorted(dc.left_only))     # ['left_only.txt']
    print(sorted(dc.right_only))    # ['right_only.txt']
    print(sorted(dc.common_dirs))   # ['subdir']
    print(sorted(dc.common_files))  # ['diff.txt', 'same.txt']
    print(sorted(dc.common_funny))  # []
    print(sorted(dc.same_files))    # ['same.txt']
    print(sorted(dc.diff_files))    # ['diff.txt']
    print(sorted(dc.funny_files))   # []
    print(sorted(dc.subdirs.keys()))  # ['subdir']

    # subdirs entry is a dircmp instance
    sub_dc = dc.subdirs['subdir']
    print(type(sub_dc).__name__)          # dircmp
    print(sorted(sub_dc.same_files))      # ['sub.txt']
    print(sorted(sub_dc.diff_files))      # ['sub_diff.txt']
    print(sorted(sub_dc.left_list))       # ['sub.txt', 'sub_diff.txt']

    # ===== dircmp.report() =====
    dc.report()
    # diff a b
    # Only in a : ['left_only.txt']
    # Only in b : ['right_only.txt']
    # Identical files : ['same.txt']
    # Differing files : ['diff.txt']
    # Common subdirectories : ['subdir']

    # ===== report_partial_closure() =====
    print()
    dc.report_partial_closure()
    # diff a b
    # ...
    # (blank line)
    # diff a/subdir b/subdir
    # Identical files : ['sub.txt']
    # Differing files : ['sub_diff.txt']

    # ===== report_full_closure() — same as partial here (one level) =====
    print()
    dc2 = filecmp.dircmp('a', 'b')
    dc2.report_full_closure()

    # ===== dircmp ignore parameter =====
    dc3 = filecmp.dircmp('a', 'b', ignore=['diff.txt'])
    print(sorted(dc3.common_files))   # ['same.txt']
    print(sorted(dc3.left_only))      # ['left_only.txt']

    # ===== dircmp hide parameter =====
    dc4 = filecmp.dircmp('a', 'b', hide=['same.txt', '.', '..'])
    print(sorted(dc4.left_list))      # ['diff.txt', 'left_only.txt', 'subdir']

    # ===== dircmp shallow=False =====
    dc5 = filecmp.dircmp('a', 'b', shallow=False)
    print(sorted(dc5.same_files))     # ['same.txt']
    print(sorted(dc5.diff_files))     # ['diff.txt']

    # ===== DEFAULT_IGNORES filtering =====
    os.makedirs('a/__pycache__')
    os.makedirs('b/__pycache__')
    os.makedirs('a/.git')
    os.makedirs('b/.git')
    dc6 = filecmp.dircmp('a', 'b')
    print('__pycache__' not in dc6.left_list)   # True (filtered by DEFAULT_IGNORES)
    print('.git' not in dc6.left_list)           # True

print('done')

import glob
import os
import re
import tempfile

# ===== escape() =====
print(glob.escape('hello'))           # hello
print(glob.escape('[test]'))          # \[test\]
print(glob.escape('*.txt'))           # \*.txt
print(glob.escape('file?.py'))        # file\?.py
print(glob.escape('/path/to/[dir]'))  # /path/to/\[dir\]
print(glob.escape('no_special'))      # no_special

# ===== translate() — Python 3.12+ =====
# Returns a regex that matches the same strings as the glob pattern
tr = glob.translate('*.txt')
print(isinstance(tr, str))            # True
print(bool(re.fullmatch(tr, 'hello.txt')))   # True
print(bool(re.fullmatch(tr, 'hello.py')))    # False
print(bool(re.fullmatch(tr, '.hidden.txt'))) # False (hidden not matched)

tr2 = glob.translate('*.txt', include_hidden=True)
print(bool(re.fullmatch(tr2, '.hidden.txt'))) # True

tr3 = glob.translate('**/*.txt', recursive=True)
print(isinstance(tr3, str))           # True

# ===== build test tree =====
with tempfile.TemporaryDirectory() as tmpdir:
    os.chdir(tmpdir)

    # Create structure:
    # a.txt  b.txt  c.py  .hidden  .hidden.txt
    # sub/d.txt  sub/e.txt  sub/.sub_hidden
    # sub/deep/f.txt
    # sub2/g.txt
    os.makedirs('sub/deep')
    os.makedirs('sub2')
    for path in ['a.txt', 'b.txt', 'c.py', '.hidden', '.hidden.txt',
                  'sub/d.txt', 'sub/e.txt', 'sub/.sub_hidden',
                  'sub/deep/f.txt', 'sub2/g.txt']:
        with open(path, 'w') as fh:
            fh.write(path)

    # ===== glob.glob basic =====
    print(sorted(glob.glob('*.txt')))            # ['a.txt', 'b.txt']
    print(sorted(glob.glob('*.py')))             # ['c.py']
    print(sorted(glob.glob('*')))                # ['a.txt', 'b.txt', 'c.py', 'sub', 'sub2']
    print(sorted(glob.glob('?.txt')))            # ['a.txt', 'b.txt']
    print(sorted(glob.glob('sub/*.txt')))        # ['sub/d.txt', 'sub/e.txt']

    # ===== hidden file handling =====
    print(sorted(glob.glob('.*')))               # ['.hidden', '.hidden.txt']
    print(sorted(glob.glob('.*.txt')))           # ['.hidden.txt']
    print(sorted(glob.glob('*.txt')))            # ['a.txt', 'b.txt']  (no hidden)
    print(sorted(glob.glob('*.txt', include_hidden=True)))  # ['.hidden.txt', 'a.txt', 'b.txt']

    # ===== directory-only (trailing slash) =====
    dirs = sorted(glob.glob('*/'))
    print(dirs)                                  # ['sub/', 'sub2/']

    # ===== no matches =====
    print(glob.glob('no_match_*'))               # []
    print(glob.glob('*.xyz'))                    # []

    # ===== character class =====
    print(sorted(glob.glob('[ab].txt')))         # ['a.txt', 'b.txt']
    print(sorted(glob.glob('[!a]*.txt')))        # ['b.txt']

    # ===== recursive ** =====
    print(sorted(glob.glob('**/*.txt', recursive=True)))
    # ['a.txt', 'b.txt', 'sub/d.txt', 'sub/e.txt', 'sub/deep/f.txt', 'sub2/g.txt']

    print(sorted(glob.glob('**', recursive=True)))
    # all files and dirs recursively (non-hidden)

    # ** with recursive=False → ** acts like * (one dir level)
    print(sorted(glob.glob('**/*.txt', recursive=False)))
    # ['sub/d.txt', 'sub/e.txt', 'sub2/g.txt']

    # ===== root_dir =====
    orig = os.getcwd()
    os.chdir('/')  # change away from tmpdir
    print(sorted(glob.glob('*.txt', root_dir=tmpdir)))       # ['a.txt', 'b.txt']
    print(sorted(glob.glob('sub/*.txt', root_dir=tmpdir)))   # ['sub/d.txt', 'sub/e.txt']
    os.chdir(tmpdir)

    # ===== iglob =====
    it = glob.iglob('*.txt')
    results = sorted(it)
    print(results)                               # ['a.txt', 'b.txt']

    it2 = glob.iglob('**/*.txt', recursive=True)
    print(sorted(it2))
    # ['a.txt', 'b.txt', 'sub/d.txt', 'sub/e.txt', 'sub/deep/f.txt', 'sub2/g.txt']

    # ===== absolute path pattern =====
    abs_results = glob.glob(os.path.join(tmpdir, '*.txt'))
    print(len(abs_results) == 2)                 # True
    print(all(r.endswith('.txt') for r in abs_results))  # True

    # ===== pattern with no directory component =====
    print(sorted(glob.glob('c.py')))             # ['c.py']  exact match
    print(sorted(glob.glob('nonexistent.txt')))  # []

    # ===== ** alone matches everything recursively =====
    all_items = glob.glob('**', recursive=True)
    print('sub' in all_items)                    # True
    print('sub/deep' in all_items)               # True
    print('sub/d.txt' in all_items)              # True

print('done')

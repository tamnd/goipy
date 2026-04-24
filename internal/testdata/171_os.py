"""Tests for the extended os module."""
import os
import tempfile

tmp = tempfile.mkdtemp()

# --- makedirs exist_ok ---
sub = os.path.join(tmp, "a", "b")
os.makedirs(sub, exist_ok=False)
os.makedirs(sub, exist_ok=True)  # should not raise
try:
    os.makedirs(sub, exist_ok=False)
except FileExistsError:
    print("makedirs exist_ok=False raises FileExistsError")

# --- replace ---
src = os.path.join(tmp, "src.txt")
dst = os.path.join(tmp, "dst.txt")
with open(src, "w") as f:
    f.write("hello")
with open(dst, "w") as f:
    f.write("world")
os.replace(src, dst)
print(os.path.exists(dst))   # True
print(os.path.exists(src))   # False

# --- link ---
lnk = os.path.join(tmp, "lnk.txt")
os.link(dst, lnk)
print(os.stat(lnk).st_nlink >= 2)  # True

# --- truncate ---
os.truncate(dst, 3)
with open(dst) as f:
    print(f.read())  # hel

# --- access ---
print(os.access(dst, os.F_OK))  # True
print(os.access(dst, os.R_OK))  # True
print(os.access(dst, os.W_OK))  # True
print(os.access("/no/such/path", os.F_OK))  # False

# --- umask ---
old = os.umask(0o022)
restored = os.umask(old)
print(restored == 0o022)  # True

# --- process info ---
print(os.getgid() >= 0)    # True
print(os.getegid() >= 0)   # True
print(os.geteuid() >= 0)   # True
print(os.getppid() > 0)    # True

# --- cpu_count ---
cc = os.cpu_count()
print(cc is None or cc > 0)  # True

# --- strerror ---
print(os.strerror(2))   # No such file or directory
print(os.strerror(13))  # Permission denied

# --- urandom ---
data = os.urandom(16)
print(len(data) == 16)  # True
print(isinstance(data, bytes))  # True

# --- fsencode / fsdecode ---
enc = os.fsencode("hello")
print(enc == b"hello")  # True
dec = os.fsdecode(b"hello")
print(dec == "hello")  # True

# --- get_exec_path ---
path = os.get_exec_path()
print(isinstance(path, list))   # True
print(len(path) > 0)            # True

# --- walk ---
walkdir = os.path.join(tmp, "walk")
os.makedirs(os.path.join(walkdir, "sub"), exist_ok=True)
with open(os.path.join(walkdir, "f1.txt"), "w") as f:
    f.write("x")
with open(os.path.join(walkdir, "sub", "f2.txt"), "w") as f:
    f.write("y")

roots = []
for root, dirs, files in os.walk(walkdir):
    roots.append(root)
print(len(roots) == 2)  # True — walkdir and walkdir/sub

# --- scandir ---
entries = list(os.scandir(walkdir))
names = sorted(e.name for e in entries)
print(names)  # ['f1.txt', 'sub']
for e in entries:
    if e.name == "sub":
        print(e.is_dir())   # True
        print(e.is_file())  # False
    if e.name == "f1.txt":
        print(e.is_file())  # True
        print(e.is_dir())   # False

# --- stat_result indexing ---
st = os.stat(dst)
print(st[6] == st.st_size)   # True
print(st[4] == st.st_uid)    # True

# --- low-level fd ops ---
fdpath = os.path.join(tmp, "fd_test.bin")
fd = os.open(fdpath, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
n = os.write(fd, b"hello world")
print(n == 11)  # True
os.close(fd)

fd2 = os.open(fdpath, os.O_RDONLY)
buf = os.read(fd2, 5)
print(buf == b"hello")  # True
pos = os.lseek(fd2, 0, os.SEEK_SET)
print(pos == 0)  # True
all_data = os.read(fd2, 100)
print(all_data == b"hello world")  # True
fst = os.fstat(fd2)
print(fst.st_size == 11)  # True
os.close(fd2)

# --- dup / dup2 ---
fd3 = os.open(fdpath, os.O_RDONLY)
fd4 = os.dup(fd3)
print(fd4 != fd3)  # True
buf3 = os.read(fd3, 3)
buf4 = os.read(fd4, 3)
print(buf3 == b"hel")  # True
print(buf4 == b"hel")  # True  (independent position)
os.close(fd3)
os.close(fd4)

# --- environ ---
os.environ["_TEST_KEY"] = "abc"
print("_TEST_KEY" in os.environ)  # True
print(os.environ["_TEST_KEY"])    # abc
del os.environ["_TEST_KEY"]
print("_TEST_KEY" in os.environ)  # False

keys = list(os.environ.keys())
print(len(keys) > 0)   # True
vals = list(os.environ.values())
print(len(vals) > 0)   # True
items = list(os.environ.items())
print(len(items) > 0)  # True

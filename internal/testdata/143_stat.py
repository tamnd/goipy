import stat
import os
import tempfile

# ===== ST_* index constants =====
print(stat.ST_MODE)    # 0
print(stat.ST_INO)     # 1
print(stat.ST_DEV)     # 2
print(stat.ST_NLINK)   # 3
print(stat.ST_UID)     # 4
print(stat.ST_GID)     # 5
print(stat.ST_SIZE)    # 6
print(stat.ST_ATIME)   # 7
print(stat.ST_MTIME)   # 8
print(stat.ST_CTIME)   # 9

# ===== S_IF* file type constants =====
print(oct(stat.S_IFDIR))   # 0o40000
print(oct(stat.S_IFCHR))   # 0o20000
print(oct(stat.S_IFBLK))   # 0o60000
print(oct(stat.S_IFREG))   # 0o100000
print(oct(stat.S_IFIFO))   # 0o10000
print(oct(stat.S_IFLNK))   # 0o120000
print(oct(stat.S_IFSOCK))  # 0o140000
print(oct(stat.S_IFDOOR))  # 0o0
print(oct(stat.S_IFPORT))  # 0o0
print(oct(stat.S_IFWHT))   # 0o160000

# ===== S_IFMT() / S_IMODE() — functions, not constants =====
print(oct(stat.S_IFMT(0o100644)))   # 0o100000
print(oct(stat.S_IFMT(0o040755)))   # 0o40000
print(oct(stat.S_IFMT(0o120777)))   # 0o120000
print(oct(stat.S_IMODE(0o100644)))  # 0o644
print(oct(stat.S_IMODE(0o040755)))  # 0o755
print(oct(stat.S_IMODE(0o104755)))  # 0o4755

# ===== Type-testing functions =====
print(stat.S_ISDIR(0o040755))    # True
print(stat.S_ISDIR(0o100644))    # False

print(stat.S_ISCHR(0o020600))    # True
print(stat.S_ISCHR(0o100644))    # False

print(stat.S_ISBLK(0o060600))    # True
print(stat.S_ISBLK(0o100644))    # False

print(stat.S_ISREG(0o100644))    # True
print(stat.S_ISREG(0o040755))    # False

print(stat.S_ISFIFO(0o010644))   # True
print(stat.S_ISFIFO(0o100644))   # False

print(stat.S_ISLNK(0o120777))    # True
print(stat.S_ISLNK(0o100644))    # False

print(stat.S_ISSOCK(0o140777))   # True
print(stat.S_ISSOCK(0o100644))   # False

print(stat.S_ISDOOR(0o150777))   # False  (S_IFDOOR=0 on POSIX)
print(stat.S_ISDOOR(0o100644))   # False

print(stat.S_ISPORT(0o160777))   # False  (S_IFPORT=0 on POSIX)
print(stat.S_ISPORT(0o100644))   # False

print(stat.S_ISWHT(0o160000))    # True
print(stat.S_ISWHT(0o100644))    # False

# ===== Permission bit constants =====
print(oct(stat.S_ISUID))   # 0o4000
print(oct(stat.S_ISGID))   # 0o2000
print(oct(stat.S_ISVTX))   # 0o1000

print(oct(stat.S_IRWXU))   # 0o700
print(oct(stat.S_IRUSR))   # 0o400
print(oct(stat.S_IWUSR))   # 0o200
print(oct(stat.S_IXUSR))   # 0o100

print(oct(stat.S_IRWXG))   # 0o70
print(oct(stat.S_IRGRP))   # 0o40
print(oct(stat.S_IWGRP))   # 0o20
print(oct(stat.S_IXGRP))   # 0o10

print(oct(stat.S_IRWXO))   # 0o7
print(oct(stat.S_IROTH))   # 0o4
print(oct(stat.S_IWOTH))   # 0o2
print(oct(stat.S_IXOTH))   # 0o1

# ===== Legacy aliases =====
print(stat.S_IREAD == stat.S_IRUSR)    # True
print(stat.S_IWRITE == stat.S_IWUSR)   # True
print(stat.S_IEXEC == stat.S_IXUSR)    # True
print(stat.S_ENFMT == stat.S_ISGID)    # True

# ===== UF_* user flags =====
print(hex(stat.UF_NODUMP))      # 0x1
print(hex(stat.UF_IMMUTABLE))   # 0x2
print(hex(stat.UF_APPEND))      # 0x4
print(hex(stat.UF_OPAQUE))      # 0x8
print(hex(stat.UF_NOUNLINK))    # 0x10
print(hex(stat.UF_COMPRESSED))  # 0x800
print(hex(stat.UF_HIDDEN))      # 0x8000
print(hex(stat.UF_TRACKED))     # 0x40
print(hex(stat.UF_DATAVAULT))   # 0x80
print(hex(stat.UF_SETTABLE))    # 0xffff

# ===== SF_* superuser flags =====
print(hex(stat.SF_ARCHIVED))    # 0x10000
print(hex(stat.SF_IMMUTABLE))   # 0x20000
print(hex(stat.SF_APPEND))      # 0x40000
print(hex(stat.SF_NOUNLINK))    # 0x100000
print(hex(stat.SF_SNAPSHOT))    # 0x200000
print(hex(stat.SF_FIRMLINK))    # 0x800000
print(hex(stat.SF_RESTRICTED))  # 0x80000
print(hex(stat.SF_SUPPORTED))   # 0x9f0000
print(hex(stat.SF_DATALESS))    # 0x40000000
print(hex(stat.SF_SYNTHETIC))   # 0xc0000000
print(hex(stat.SF_SETTABLE))    # 0x3fff0000

# ===== FILE_ATTRIBUTE_* Windows constants (cross-platform) =====
print(stat.FILE_ATTRIBUTE_ARCHIVE)              # 32
print(stat.FILE_ATTRIBUTE_COMPRESSED)           # 2048
print(stat.FILE_ATTRIBUTE_DEVICE)               # 64
print(stat.FILE_ATTRIBUTE_DIRECTORY)            # 16
print(stat.FILE_ATTRIBUTE_ENCRYPTED)            # 16384
print(stat.FILE_ATTRIBUTE_HIDDEN)               # 2
print(stat.FILE_ATTRIBUTE_INTEGRITY_STREAM)     # 32768
print(stat.FILE_ATTRIBUTE_NORMAL)               # 128
print(stat.FILE_ATTRIBUTE_NOT_CONTENT_INDEXED)  # 8192
print(stat.FILE_ATTRIBUTE_NO_SCRUB_DATA)        # 131072
print(stat.FILE_ATTRIBUTE_OFFLINE)              # 4096
print(stat.FILE_ATTRIBUTE_READONLY)             # 1
print(stat.FILE_ATTRIBUTE_REPARSE_POINT)        # 1024
print(stat.FILE_ATTRIBUTE_SPARSE_FILE)          # 512
print(stat.FILE_ATTRIBUTE_SYSTEM)               # 4
print(stat.FILE_ATTRIBUTE_TEMPORARY)            # 256
print(stat.FILE_ATTRIBUTE_VIRTUAL)              # 65536

# ===== filemode() =====
print(stat.filemode(0o100644))   # -rw-r--r--
print(stat.filemode(0o100755))   # -rwxr-xr-x
print(stat.filemode(0o040755))   # drwxr-xr-x
print(stat.filemode(0o120777))   # lrwxrwxrwx
print(stat.filemode(0o140777))   # srwxrwxrwx
print(stat.filemode(0o010644))   # prw-r--r--
print(stat.filemode(0o060640))   # brw-r-----
print(stat.filemode(0o020640))   # crw-r-----
print(stat.filemode(0o104755))   # -rwsr-xr-x  (setuid + x)
print(stat.filemode(0o102755))   # -rwxr-sr-x  (setgid + x)
print(stat.filemode(0o042755))   # drwxr-sr-x  (setgid dir + x)
print(stat.filemode(0o041777))   # drwxrwxrwt  (sticky + writable)
print(stat.filemode(0o041755))   # drwxr-xr-t  (sticky)
print(stat.filemode(0o044755))   # drwsr-xr-x  (setuid + no x → S)
print(stat.filemode(0o000000))   # ----------

# ===== Using stat with real files =====
with tempfile.TemporaryDirectory() as tmpdir:
    f = os.path.join(tmpdir, 'test.txt')
    with open(f, 'w') as fp:
        fp.write('hello')

    st = os.stat(f)
    mode = st.st_mode

    print(stat.S_ISREG(mode))                    # True
    print(stat.S_ISDIR(mode))                     # False
    print(isinstance(stat.filemode(mode), str))   # True
    print(stat.filemode(mode)[0])                 # -

    link = os.path.join(tmpdir, 'link')
    os.symlink(f, link)
    lst = os.lstat(link)
    print(stat.S_ISLNK(lst.st_mode))    # True
    print(stat.S_ISREG(lst.st_mode))    # False

    d = os.path.join(tmpdir, 'subdir')
    os.mkdir(d)
    dst = os.stat(d)
    print(stat.S_ISDIR(dst.st_mode))            # True
    print(stat.filemode(dst.st_mode)[0])         # d

print('done')

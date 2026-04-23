package vm

import (
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildStat() *object.Module {
	m := &object.Module{Name: "stat", Dict: object.NewDict()}

	// ===== ST_* index constants =====
	m.Dict.SetStr("ST_MODE", object.NewInt(0))
	m.Dict.SetStr("ST_INO", object.NewInt(1))
	m.Dict.SetStr("ST_DEV", object.NewInt(2))
	m.Dict.SetStr("ST_NLINK", object.NewInt(3))
	m.Dict.SetStr("ST_UID", object.NewInt(4))
	m.Dict.SetStr("ST_GID", object.NewInt(5))
	m.Dict.SetStr("ST_SIZE", object.NewInt(6))
	m.Dict.SetStr("ST_ATIME", object.NewInt(7))
	m.Dict.SetStr("ST_MTIME", object.NewInt(8))
	m.Dict.SetStr("ST_CTIME", object.NewInt(9))

	// ===== S_IF* file type constants =====
	m.Dict.SetStr("S_IFDIR", object.NewInt(0o040000))
	m.Dict.SetStr("S_IFCHR", object.NewInt(0o020000))
	m.Dict.SetStr("S_IFBLK", object.NewInt(0o060000))
	m.Dict.SetStr("S_IFREG", object.NewInt(0o100000))
	m.Dict.SetStr("S_IFIFO", object.NewInt(0o010000))
	m.Dict.SetStr("S_IFLNK", object.NewInt(0o120000))
	m.Dict.SetStr("S_IFSOCK", object.NewInt(0o140000))
	m.Dict.SetStr("S_IFDOOR", object.NewInt(0))       // Solaris; 0 on POSIX
	m.Dict.SetStr("S_IFPORT", object.NewInt(0))       // Solaris event port; 0 on POSIX
	m.Dict.SetStr("S_IFWHT", object.NewInt(0o160000)) // BSD whiteout

	// ===== Permission bit constants =====
	m.Dict.SetStr("S_ISUID", object.NewInt(0o4000))
	m.Dict.SetStr("S_ISGID", object.NewInt(0o2000))
	m.Dict.SetStr("S_ISVTX", object.NewInt(0o1000))

	m.Dict.SetStr("S_IRWXU", object.NewInt(0o700))
	m.Dict.SetStr("S_IRUSR", object.NewInt(0o400))
	m.Dict.SetStr("S_IWUSR", object.NewInt(0o200))
	m.Dict.SetStr("S_IXUSR", object.NewInt(0o100))

	m.Dict.SetStr("S_IRWXG", object.NewInt(0o070))
	m.Dict.SetStr("S_IRGRP", object.NewInt(0o040))
	m.Dict.SetStr("S_IWGRP", object.NewInt(0o020))
	m.Dict.SetStr("S_IXGRP", object.NewInt(0o010))

	m.Dict.SetStr("S_IRWXO", object.NewInt(0o007))
	m.Dict.SetStr("S_IROTH", object.NewInt(0o004))
	m.Dict.SetStr("S_IWOTH", object.NewInt(0o002))
	m.Dict.SetStr("S_IXOTH", object.NewInt(0o001))

	// Legacy aliases
	m.Dict.SetStr("S_IREAD", object.NewInt(0o400))  // = S_IRUSR
	m.Dict.SetStr("S_IWRITE", object.NewInt(0o200)) // = S_IWUSR
	m.Dict.SetStr("S_IEXEC", object.NewInt(0o100))  // = S_IXUSR
	m.Dict.SetStr("S_ENFMT", object.NewInt(0o2000)) // = S_ISGID

	// ===== UF_* user flags (BSD/macOS) =====
	m.Dict.SetStr("UF_NODUMP", object.NewInt(0x00000001))
	m.Dict.SetStr("UF_IMMUTABLE", object.NewInt(0x00000002))
	m.Dict.SetStr("UF_APPEND", object.NewInt(0x00000004))
	m.Dict.SetStr("UF_OPAQUE", object.NewInt(0x00000008))
	m.Dict.SetStr("UF_NOUNLINK", object.NewInt(0x00000010))
	m.Dict.SetStr("UF_COMPRESSED", object.NewInt(0x00000020))
	m.Dict.SetStr("UF_TRACKED", object.NewInt(0x00000040))
	m.Dict.SetStr("UF_DATAVAULT", object.NewInt(0x00000080))
	m.Dict.SetStr("UF_HIDDEN", object.NewInt(0x00008000))
	m.Dict.SetStr("UF_SETTABLE", object.NewInt(0x0000ffff))

	// ===== SF_* superuser flags (BSD/macOS) =====
	m.Dict.SetStr("SF_ARCHIVED", object.NewInt(0x00010000))
	m.Dict.SetStr("SF_IMMUTABLE", object.NewInt(0x00020000))
	m.Dict.SetStr("SF_APPEND", object.NewInt(0x00040000))
	m.Dict.SetStr("SF_RESTRICTED", object.NewInt(0x00080000))
	m.Dict.SetStr("SF_NOUNLINK", object.NewInt(0x00100000))
	m.Dict.SetStr("SF_SNAPSHOT", object.NewInt(0x00200000))
	m.Dict.SetStr("SF_FIRMLINK", object.NewInt(0x00800000))
	m.Dict.SetStr("SF_DATALESS", object.NewInt(0x40000000))
	m.Dict.SetStr("SF_SYNTHETIC", object.NewInt(0xc0000000))
	m.Dict.SetStr("SF_SUPPORTED", object.NewInt(0x009f0000))
	m.Dict.SetStr("SF_SETTABLE", object.NewInt(0x3fff0000))

	// ===== FILE_ATTRIBUTE_* Windows constants (cross-platform) =====
	m.Dict.SetStr("FILE_ATTRIBUTE_ARCHIVE", object.NewInt(32))
	m.Dict.SetStr("FILE_ATTRIBUTE_COMPRESSED", object.NewInt(2048))
	m.Dict.SetStr("FILE_ATTRIBUTE_DEVICE", object.NewInt(64))
	m.Dict.SetStr("FILE_ATTRIBUTE_DIRECTORY", object.NewInt(16))
	m.Dict.SetStr("FILE_ATTRIBUTE_ENCRYPTED", object.NewInt(16384))
	m.Dict.SetStr("FILE_ATTRIBUTE_HIDDEN", object.NewInt(2))
	m.Dict.SetStr("FILE_ATTRIBUTE_INTEGRITY_STREAM", object.NewInt(32768))
	m.Dict.SetStr("FILE_ATTRIBUTE_NORMAL", object.NewInt(128))
	m.Dict.SetStr("FILE_ATTRIBUTE_NOT_CONTENT_INDEXED", object.NewInt(8192))
	m.Dict.SetStr("FILE_ATTRIBUTE_NO_SCRUB_DATA", object.NewInt(131072))
	m.Dict.SetStr("FILE_ATTRIBUTE_OFFLINE", object.NewInt(4096))
	m.Dict.SetStr("FILE_ATTRIBUTE_READONLY", object.NewInt(1))
	m.Dict.SetStr("FILE_ATTRIBUTE_REPARSE_POINT", object.NewInt(1024))
	m.Dict.SetStr("FILE_ATTRIBUTE_SPARSE_FILE", object.NewInt(512))
	m.Dict.SetStr("FILE_ATTRIBUTE_SYSTEM", object.NewInt(4))
	m.Dict.SetStr("FILE_ATTRIBUTE_TEMPORARY", object.NewInt(256))
	m.Dict.SetStr("FILE_ATTRIBUTE_VIRTUAL", object.NewInt(65536))

	// ===== S_IFMT(mode) — extract file type bits =====
	m.Dict.SetStr("S_IFMT", &object.BuiltinFunc{Name: "S_IFMT", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		mode, err := statModeArg(i, "S_IFMT", a)
		if err != nil {
			return nil, err
		}
		return object.NewInt(mode & 0o170000), nil
	}})

	// ===== S_IMODE(mode) — extract permission bits =====
	m.Dict.SetStr("S_IMODE", &object.BuiltinFunc{Name: "S_IMODE", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		mode, err := statModeArg(i, "S_IMODE", a)
		if err != nil {
			return nil, err
		}
		return object.NewInt(mode & 0o7777), nil
	}})

	// ===== Type-testing functions =====
	statIsType := func(name string, ifConst int64) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			mode, err := statModeArg(i, name, a)
			if err != nil {
				return nil, err
			}
			if ifConst == 0 {
				return object.False, nil
			}
			return object.BoolOf(mode&0o170000 == ifConst), nil
		}}
	}

	m.Dict.SetStr("S_ISDIR", statIsType("S_ISDIR", 0o040000))
	m.Dict.SetStr("S_ISCHR", statIsType("S_ISCHR", 0o020000))
	m.Dict.SetStr("S_ISBLK", statIsType("S_ISBLK", 0o060000))
	m.Dict.SetStr("S_ISREG", statIsType("S_ISREG", 0o100000))
	m.Dict.SetStr("S_ISFIFO", statIsType("S_ISFIFO", 0o010000))
	m.Dict.SetStr("S_ISLNK", statIsType("S_ISLNK", 0o120000))
	m.Dict.SetStr("S_ISSOCK", statIsType("S_ISSOCK", 0o140000))
	m.Dict.SetStr("S_ISDOOR", statIsType("S_ISDOOR", 0)) // always False on POSIX
	m.Dict.SetStr("S_ISPORT", statIsType("S_ISPORT", 0)) // always False on POSIX
	m.Dict.SetStr("S_ISWHT", statIsType("S_ISWHT", 0o160000))

	// ===== filemode(mode) =====
	m.Dict.SetStr("filemode", &object.BuiltinFunc{Name: "filemode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		mode, err := statModeArg(i, "filemode", a)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: statFilemode(mode)}, nil
	}})

	return m
}

// statModeArg extracts a single integer mode argument.
func statModeArg(i *Interp, name string, a []object.Object) (int64, error) {
	if len(a) < 1 {
		return 0, object.Errorf(i.typeErr, "%s() requires 1 argument", name)
	}
	n, ok := a[0].(*object.Int)
	if !ok {
		return 0, object.Errorf(i.typeErr, "%s() argument must be int", name)
	}
	return n.V.Int64(), nil
}

// statFilemode returns a 10-character mode string like "-rwxr-xr-x".
// Matches CPython's stat.filemode() / os.stat result formatting.
func statFilemode(mode int64) string {
	buf := [10]byte{}

	// File type character.
	switch mode & 0o170000 {
	case 0o120000:
		buf[0] = 'l' // symbolic link
	case 0o140000:
		buf[0] = 's' // socket
	case 0o060000:
		buf[0] = 'b' // block device
	case 0o020000:
		buf[0] = 'c' // character device
	case 0o040000:
		buf[0] = 'd' // directory
	case 0o010000:
		buf[0] = 'p' // named pipe (FIFO)
	case 0o100000:
		buf[0] = '-' // regular file
	case 0o160000:
		buf[0] = 'w' // BSD whiteout
	default:
		buf[0] = '?'
	}

	// User read / write / execute+setuid
	if mode&0o400 != 0 {
		buf[1] = 'r'
	} else {
		buf[1] = '-'
	}
	if mode&0o200 != 0 {
		buf[2] = 'w'
	} else {
		buf[2] = '-'
	}
	switch {
	case mode&0o4100 == 0o4100: // setuid + execute
		buf[3] = 's'
	case mode&0o4000 != 0: // setuid, no execute
		buf[3] = 'S'
	case mode&0o100 != 0:
		buf[3] = 'x'
	default:
		buf[3] = '-'
	}

	// Group read / write / execute+setgid
	if mode&0o040 != 0 {
		buf[4] = 'r'
	} else {
		buf[4] = '-'
	}
	if mode&0o020 != 0 {
		buf[5] = 'w'
	} else {
		buf[5] = '-'
	}
	switch {
	case mode&0o2010 == 0o2010: // setgid + execute
		buf[6] = 's'
	case mode&0o2000 != 0: // setgid, no execute
		buf[6] = 'S'
	case mode&0o010 != 0:
		buf[6] = 'x'
	default:
		buf[6] = '-'
	}

	// Other read / write / execute+sticky
	if mode&0o004 != 0 {
		buf[7] = 'r'
	} else {
		buf[7] = '-'
	}
	if mode&0o002 != 0 {
		buf[8] = 'w'
	} else {
		buf[8] = '-'
	}
	switch {
	case mode&0o1001 == 0o1001: // sticky + execute
		buf[9] = 't'
	case mode&0o1000 != 0: // sticky, no execute
		buf[9] = 'T'
	case mode&0o001 != 0:
		buf[9] = 'x'
	default:
		buf[9] = '-'
	}

	return string(buf[:])
}

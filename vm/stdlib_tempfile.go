package vm

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildTempfile() *object.Module {
	m := &object.Module{Name: "tempfile", Dict: object.NewDict()}

	// tempdir — module-level override for default temp directory.
	m.Dict.SetStr("tempdir", object.None)

	getTempBase := func() string {
		if v, ok := m.Dict.GetStr("tempdir"); ok && v != object.None {
			if s, ok := v.(*object.Str); ok && s.V != "" {
				return s.V
			}
		}
		dir := os.TempDir()
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			dir = resolved
		}
		return dir
	}

	// parseTempSPD extracts suffix, prefix, dir from positional/keyword args.
	// Positional order: suffix, prefix, dir.
	parseTempSPD := func(a []object.Object, kw *object.Dict) (suffix, prefix, dir string) {
		prefix = "tmp"
		dir = getTempBase()
		if len(a) >= 1 && a[0] != object.None {
			if s, ok := a[0].(*object.Str); ok {
				suffix = s.V
			}
		}
		if len(a) >= 2 && a[1] != object.None {
			if s, ok := a[1].(*object.Str); ok {
				prefix = s.V
			}
		}
		if len(a) >= 3 && a[2] != object.None {
			if s, ok := a[2].(*object.Str); ok {
				dir = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("suffix"); ok && v != object.None {
				if s, ok := v.(*object.Str); ok {
					suffix = s.V
				}
			}
			if v, ok := kw.GetStr("prefix"); ok && v != object.None {
				if s, ok := v.(*object.Str); ok {
					prefix = s.V
				}
			}
			if v, ok := kw.GetStr("dir"); ok && v != object.None {
				if s, ok := v.(*object.Str); ok {
					dir = s.V
				}
			}
		}
		return
	}

	// gettempdir() — return the default temp directory as str.
	// Caches the result in tempdir (matching CPython behaviour).
	m.Dict.SetStr("gettempdir", &object.BuiltinFunc{Name: "gettempdir", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		dir := getTempBase()
		// Resolve symlinks to match CPython on macOS (/tmp → /private/tmp).
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			dir = resolved
		}
		m.Dict.SetStr("tempdir", &object.Str{V: dir})
		return &object.Str{V: dir}, nil
	}})

	// gettempdirb() — return the default temp directory as bytes.
	// Caches the result in tempdir.
	m.Dict.SetStr("gettempdirb", &object.BuiltinFunc{Name: "gettempdirb", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		dir := getTempBase()
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			dir = resolved
		}
		m.Dict.SetStr("tempdir", &object.Str{V: dir})
		return &object.Bytes{V: []byte(dir)}, nil
	}})

	// gettempprefix() — return default file prefix as str.
	m.Dict.SetStr("gettempprefix", &object.BuiltinFunc{Name: "gettempprefix", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "tmp"}, nil
	}})

	// gettempprefixb() — return default file prefix as bytes.
	m.Dict.SetStr("gettempprefixb", &object.BuiltinFunc{Name: "gettempprefixb", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Bytes{V: []byte("tmp")}, nil
	}})

	// mktemp(suffix='', prefix='tmp', dir=None) — DEPRECATED. Create, remove, return path.
	m.Dict.SetStr("mktemp", &object.BuiltinFunc{Name: "mktemp", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		suffix, prefix, dir := parseTempSPD(a, kw)
		f, err := os.CreateTemp(dir, prefix+"*"+suffix)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		name := f.Name()
		f.Close()
		os.Remove(name)
		return &object.Str{V: name}, nil
	}})

	// mkstemp(suffix=None, prefix=None, dir=None, text=False) — return (fd, path).
	// The OS-level fd is kept open; caller must os.close(fd) and os.unlink(path).
	m.Dict.SetStr("mkstemp", &object.BuiltinFunc{Name: "mkstemp", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		suffix, prefix, dir := parseTempSPD(a, kw)
		f, err := os.CreateTemp(dir, prefix+"*"+suffix)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		name := f.Name()
		fd := int64(f.Fd())
		// Keep file open; caller is responsible for os.close(fd).
		// Prevent GC finalizer from closing until caller does so.
		_ = f
		return &object.Tuple{V: []object.Object{object.NewInt(fd), &object.Str{V: name}}}, nil
	}})

	// mkdtemp(suffix=None, prefix=None, dir=None) — create temp dir, return path.
	m.Dict.SetStr("mkdtemp", &object.BuiltinFunc{Name: "mkdtemp", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		suffix, prefix, dir := parseTempSPD(a, kw)
		d, err := os.MkdirTemp(dir, prefix+"*"+suffix)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		return &object.Str{V: d}, nil
	}})

	// makeTempFileInst builds a file-like instance backed by an *os.File.
	// All methods are on the class dict (per-instance class, since each captures different state).
	makeTempFileInst := func(clsName string, f *os.File, binary bool, deleteOnClose bool) *object.Instance {
		cls := &object.Class{Name: clsName, Dict: object.NewDict()}
		inst := &object.Instance{Class: cls, Dict: object.NewDict()}
		inst.Dict.SetStr("name", &object.Str{V: f.Name()})
		if binary {
			inst.Dict.SetStr("mode", &object.Str{V: "w+b"})
		} else {
			inst.Dict.SetStr("mode", &object.Str{V: "w+"})
		}
		inst.Dict.SetStr("closed", object.False)

		closed := false

		doClose := func() {
			if closed {
				return
			}
			closed = true
			f.Close()
			if deleteOnClose {
				os.Remove(f.Name())
			}
			inst.Dict.SetStr("closed", object.True)
		}

		// Note: methods below receive self as a[0] (bound-method prepend).
		// Actual user arguments start at a[1].

		cls.Dict.SetStr("write", &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "write() requires 1 argument")
			}
			var data []byte
			switch v := a[1].(type) {
			case *object.Str:
				data = []byte(v.V)
			case *object.Bytes:
				data = v.V
			default:
				return nil, object.Errorf(i.typeErr, "write() argument must be str or bytes")
			}
			n, err := f.Write(data)
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			return object.NewInt(int64(n)), nil
		}})

		cls.Dict.SetStr("read", &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			var (
				data []byte
				err  error
			)
			// a[0]=self; a[1]=size (optional)
			if len(a) >= 2 && a[1] != object.None {
				if n, ok := a[1].(*object.Int); ok {
					sz := n.V.Int64()
					if sz < 0 {
						data, err = io.ReadAll(f)
					} else {
						data = make([]byte, sz)
						nr, e := f.Read(data)
						data = data[:nr]
						if e == io.EOF {
							e = nil
						}
						err = e
					}
				}
			} else {
				data, err = io.ReadAll(f)
			}
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			if binary {
				return &object.Bytes{V: data}, nil
			}
			return &object.Str{V: string(data)}, nil
		}})

		cls.Dict.SetStr("seek", &object.BuiltinFunc{Name: "seek", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "seek() requires 1 argument")
			}
			n, ok := a[1].(*object.Int)
			if !ok {
				return nil, object.Errorf(i.typeErr, "seek() offset must be int")
			}
			whence := 0
			if len(a) >= 3 {
				if w, ok := a[2].(*object.Int); ok {
					whence = int(w.V.Int64())
				}
			}
			newPos, err := f.Seek(n.V.Int64(), whence)
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			return object.NewInt(newPos), nil
		}})

		cls.Dict.SetStr("tell", &object.BuiltinFunc{Name: "tell", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			pos, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			return object.NewInt(pos), nil
		}})

		cls.Dict.SetStr("flush", &object.BuiltinFunc{Name: "flush", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			if err := f.Sync(); err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			return object.None, nil
		}})

		cls.Dict.SetStr("truncate", &object.BuiltinFunc{Name: "truncate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			// a[0]=self; a[1]=size (optional)
			var size int64 = -1
			if len(a) >= 2 && a[1] != object.None {
				if n, ok := a[1].(*object.Int); ok {
					size = n.V.Int64()
				}
			}
			if size < 0 {
				var err error
				size, err = f.Seek(0, io.SeekCurrent)
				if err != nil {
					return nil, object.Errorf(i.osErr, "%v", err)
				}
			}
			if err := f.Truncate(size); err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			return object.NewInt(size), nil
		}})

		cls.Dict.SetStr("fileno", &object.BuiltinFunc{Name: "fileno", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			return object.NewInt(int64(f.Fd())), nil
		}})

		cls.Dict.SetStr("readable", &object.BuiltinFunc{Name: "readable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})
		cls.Dict.SetStr("writable", &object.BuiltinFunc{Name: "writable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})
		cls.Dict.SetStr("seekable", &object.BuiltinFunc{Name: "seekable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})

		cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			doClose()
			return object.None, nil
		}})

		cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})

		cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			doClose()
			return object.False, nil
		}})

		return inst
	}

	// TemporaryFile(mode='w+b', ...) — file-like object, auto-deleted on close.
	m.Dict.SetStr("TemporaryFile", &object.BuiltinFunc{Name: "TemporaryFile", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		mode := "w+b"
		if len(a) >= 1 {
			if s, ok := a[0].(*object.Str); ok {
				mode = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("mode"); ok {
				if s, ok := v.(*object.Str); ok {
					mode = s.V
				}
			}
		}
		binary := strings.Contains(mode, "b")

		suffix := ""
		prefix := "tmp"
		dir := getTempBase()
		if kw != nil {
			if v, ok := kw.GetStr("suffix"); ok && v != object.None {
				if s, ok := v.(*object.Str); ok {
					suffix = s.V
				}
			}
			if v, ok := kw.GetStr("prefix"); ok && v != object.None {
				if s, ok := v.(*object.Str); ok {
					prefix = s.V
				}
			}
			if v, ok := kw.GetStr("dir"); ok && v != object.None {
				if s, ok := v.(*object.Str); ok {
					dir = s.V
				}
			}
		}

		f, err := os.CreateTemp(dir, prefix+"*"+suffix)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		inst := makeTempFileInst("TemporaryFile", f, binary, true)
		return inst, nil
	}})

	// NamedTemporaryFile(mode='w+b', ..., delete=True, delete_on_close=True)
	m.Dict.SetStr("NamedTemporaryFile", &object.BuiltinFunc{Name: "NamedTemporaryFile", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		mode := "w+b"
		if len(a) >= 1 {
			if s, ok := a[0].(*object.Str); ok {
				mode = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("mode"); ok {
				if s, ok := v.(*object.Str); ok {
					mode = s.V
				}
			}
		}
		binary := strings.Contains(mode, "b")

		suffix := ""
		prefix := "tmp"
		dir := getTempBase()
		delete_ := true

		if kw != nil {
			if v, ok := kw.GetStr("suffix"); ok && v != object.None {
				if s, ok := v.(*object.Str); ok {
					suffix = s.V
				}
			}
			if v, ok := kw.GetStr("prefix"); ok && v != object.None {
				if s, ok := v.(*object.Str); ok {
					prefix = s.V
				}
			}
			if v, ok := kw.GetStr("dir"); ok && v != object.None {
				if s, ok := v.(*object.Str); ok {
					dir = s.V
				}
			}
			if v, ok := kw.GetStr("delete"); ok {
				delete_ = object.Truthy(v)
			}
		}

		f, err := os.CreateTemp(dir, prefix+"*"+suffix)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}
		inst := makeTempFileInst("NamedTemporaryFile", f, binary, delete_)
		// .file — the underlying real file object (exposed as an *object.File)
		inst.Dict.SetStr("file", &object.File{F: f, FilePath: f.Name(), Mode: mode, Binary: binary})
		return inst, nil
	}})

	// SpooledTemporaryFile(max_size=0, mode='w+b', ...) — in-memory until rollover.
	// This implementation stays entirely in memory (rollover is a no-op).
	m.Dict.SetStr("SpooledTemporaryFile", &object.BuiltinFunc{Name: "SpooledTemporaryFile", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		mode := "w+b"
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				mode = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("mode"); ok {
				if s, ok := v.(*object.Str); ok {
					mode = s.V
				}
			}
		}
		binary := strings.Contains(mode, "b")

		cls := &object.Class{Name: "SpooledTemporaryFile", Dict: object.NewDict()}
		inst := &object.Instance{Class: cls, Dict: object.NewDict()}
		inst.Dict.SetStr("closed", object.False)

		data := make([]byte, 0, 64)
		var pos int64
		closed := false

		doWrite := func(buf []byte) int64 {
			end := int(pos) + len(buf)
			if end > len(data) {
				data = append(data, make([]byte, end-len(data))...)
			}
			copy(data[pos:], buf)
			pos += int64(len(buf))
			return int64(len(buf))
		}

		// Note: methods receive self as a[0] (bound-method prepend). User args at a[1]+.

		cls.Dict.SetStr("write", &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "write() requires 1 argument")
			}
			var buf []byte
			switch v := a[1].(type) {
			case *object.Str:
				buf = []byte(v.V)
			case *object.Bytes:
				buf = v.V
			default:
				return nil, object.Errorf(i.typeErr, "write() argument must be str or bytes")
			}
			return object.NewInt(doWrite(buf)), nil
		}})

		cls.Dict.SetStr("read", &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			start := pos
			end := int64(len(data))
			// a[0]=self; a[1]=size (optional)
			if len(a) >= 2 && a[1] != object.None {
				if n, ok := a[1].(*object.Int); ok {
					sz := n.V.Int64()
					if sz >= 0 {
						e := pos + sz
						if e < end {
							end = e
						}
					}
				}
			}
			if start > end {
				start = end
			}
			chunk := make([]byte, end-start)
			copy(chunk, data[start:end])
			pos = end
			if binary {
				return &object.Bytes{V: chunk}, nil
			}
			return &object.Str{V: string(chunk)}, nil
		}})

		cls.Dict.SetStr("seek", &object.BuiltinFunc{Name: "seek", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "seek() requires 1 argument")
			}
			off := a[1].(*object.Int).V.Int64()
			whence := 0
			if len(a) >= 3 {
				if w, ok := a[2].(*object.Int); ok {
					whence = int(w.V.Int64())
				}
			}
			switch whence {
			case 0:
				pos = off
			case 1:
				pos += off
			case 2:
				pos = int64(len(data)) + off
			}
			if pos < 0 {
				pos = 0
			}
			return object.NewInt(pos), nil
		}})

		cls.Dict.SetStr("tell", &object.BuiltinFunc{Name: "tell", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			return object.NewInt(pos), nil
		}})

		cls.Dict.SetStr("flush", &object.BuiltinFunc{Name: "flush", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

		cls.Dict.SetStr("truncate", &object.BuiltinFunc{Name: "truncate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			// a[0]=self; a[1]=size (optional)
			size := pos
			if len(a) >= 2 && a[1] != object.None {
				if n, ok := a[1].(*object.Int); ok {
					size = n.V.Int64()
				}
			}
			if size < int64(len(data)) {
				data = data[:size]
			}
			return object.NewInt(size), nil
		}})

		cls.Dict.SetStr("readable", &object.BuiltinFunc{Name: "readable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})
		cls.Dict.SetStr("writable", &object.BuiltinFunc{Name: "writable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})
		cls.Dict.SetStr("seekable", &object.BuiltinFunc{Name: "seekable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})

		// rollover() — in this implementation always in-memory, no-op.
		cls.Dict.SetStr("rollover", &object.BuiltinFunc{Name: "rollover", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

		cls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			closed = true
			inst.Dict.SetStr("closed", object.True)
			return object.None, nil
		}})

		cls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return inst, nil
		}})

		cls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			closed = true
			inst.Dict.SetStr("closed", object.True)
			return object.False, nil
		}})

		return inst, nil
	}})

	// TemporaryDirectory(suffix=None, prefix=None, dir=None, ignore_cleanup_errors=False, *, delete=True)
	m.Dict.SetStr("TemporaryDirectory", &object.BuiltinFunc{Name: "TemporaryDirectory", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		suffix, prefix, dir := parseTempSPD(a, kw)
		ignoreCleanupErrors := false
		deleteDir := true
		if kw != nil {
			if v, ok := kw.GetStr("ignore_cleanup_errors"); ok {
				ignoreCleanupErrors = object.Truthy(v)
			}
			if v, ok := kw.GetStr("delete"); ok {
				deleteDir = object.Truthy(v)
			}
		}
		_ = ignoreCleanupErrors

		d, err := os.MkdirTemp(dir, prefix+"*"+suffix)
		if err != nil {
			return nil, object.Errorf(i.osErr, "%v", err)
		}

		tdCls := &object.Class{Name: "TemporaryDirectory", Dict: object.NewDict()}
		tdInst := &object.Instance{Class: tdCls, Dict: object.NewDict()}
		tdInst.Dict.SetStr("name", &object.Str{V: d})

		// cleanup() always removes the directory regardless of delete flag.
		tdCls.Dict.SetStr("cleanup", &object.BuiltinFunc{Name: "cleanup", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			os.RemoveAll(d)
			return object.None, nil
		}})

		tdCls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: d}, nil
		}})

		// __exit__ only removes when delete=True (default).
		tdCls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if deleteDir {
				os.RemoveAll(d)
			}
			return object.False, nil
		}})

		return tdInst, nil
	}})

	return m
}

// tempfileClose is a best-effort close of an OS-level file descriptor.
// Used by os.close() to support the pattern: fd, path = mkstemp(); os.close(fd).
func tempfileClose(fd int) error {
	return syscall.Close(fd)
}

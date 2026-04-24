package vm

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"strings"
	"time"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildGzip() *object.Module {
	m := &object.Module{Name: "gzip", Dict: object.NewDict()}

	// ── Constants ─────────────────────────────────────────────────────────────
	setInt := func(name string, v int) { m.Dict.SetStr(name, object.NewInt(int64(v))) }
	// READ and WRITE are mode strings in Python 3.14+
	m.Dict.SetStr("READ", &object.Str{V: "rb"})
	m.Dict.SetStr("WRITE", &object.Str{V: "wb"})
	setInt("READ_BUFFER_SIZE", 131072)
	setInt("FTEXT", 1)
	setInt("FHCRC", 2)
	setInt("FEXTRA", 4)
	setInt("FNAME", 8)
	setInt("FCOMMENT", 16)

	// ── Exceptions ────────────────────────────────────────────────────────────
	badGzip := &object.Class{
		Name:  "BadGzipFile",
		Bases: []*object.Class{i.osErr},
		Dict:  object.NewDict(),
	}
	m.Dict.SetStr("BadGzipFile", badGzip)

	// ── Helpers ───────────────────────────────────────────────────────────────

	// gzipDecompress decompresses one or more concatenated gzip members.
	gzipDecompress := func(data []byte) ([]byte, error) {
		var out []byte
		src := bytes.NewReader(data)
		for src.Len() > 0 {
			r, err := gzip.NewReader(src)
			if err != nil {
				return nil, object.Errorf(badGzip, "%v", err)
			}
			part, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				return nil, object.Errorf(badGzip, "%v", err)
			}
			out = append(out, part...)
		}
		return out, nil
	}

	// mtimeFromArgs extracts the mtime keyword argument (may be None or int/float).
	mtimeFromArgs := func(args []object.Object, pos int, kwargs *object.Dict) time.Time {
		var v object.Object
		if pos < len(args) {
			v = args[pos]
		} else if kwargs != nil {
			v, _ = kwargs.GetStr("mtime")
		}
		if v == nil || v == object.None {
			return gzipZeroTime // default: epoch 0 for reproducibility
		}
		if n, ok := toInt64(v); ok {
			return time.Unix(n, 0)
		}
		if f, ok := v.(*object.Float); ok {
			sec := int64(f.V)
			return time.Unix(sec, 0)
		}
		return gzipZeroTime
	}

	// ── compress() ────────────────────────────────────────────────────────────
	m.Dict.SetStr("compress", &object.BuiltinFunc{Name: "compress", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "compress() missing data")
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		level := gzip.DefaultCompression
		if len(args) >= 2 {
			if n, ok := toInt64(args[1]); ok {
				level = int(n)
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("compresslevel"); ok {
				if n, ok2 := toInt64(v); ok2 {
					level = int(n)
				}
			}
		}
		mtime := mtimeFromArgs(args, 2, kwargs)
		var buf bytes.Buffer
		w, err := gzip.NewWriterLevel(&buf, level)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%v", err)
		}
		w.ModTime = mtime
		if _, err := w.Write(data); err != nil {
			return nil, object.Errorf(badGzip, "%v", err)
		}
		if err := w.Close(); err != nil {
			return nil, object.Errorf(badGzip, "%v", err)
		}
		return &object.Bytes{V: buf.Bytes()}, nil
	}})

	// ── decompress() ──────────────────────────────────────────────────────────
	m.Dict.SetStr("decompress", &object.BuiltinFunc{Name: "decompress", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "decompress() missing data")
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		out, err := gzipDecompress(data)
		if err != nil {
			return nil, err
		}
		return &object.Bytes{V: out}, nil
	}})

	// ── GzipFile class ────────────────────────────────────────────────────────
	gzipFileCls := i.buildGzipFileClass(badGzip, gzipDecompress)
	m.Dict.SetStr("GzipFile", gzipFileCls)

	// ── open() ────────────────────────────────────────────────────────────────
	m.Dict.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "open() requires filename")
		}
		filename, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "open() filename must be str")
		}
		mode := "rb"
		if len(args) >= 2 {
			if s, ok2 := args[1].(*object.Str); ok2 {
				mode = s.V
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("mode"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					mode = s.V
				}
			}
		}
		level := gzip.DefaultCompression
		if len(args) >= 3 {
			if n, ok2 := toInt64(args[2]); ok2 {
				level = int(n)
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("compresslevel"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					level = int(n)
				}
			}
		}

		encoding := "utf-8"
		if kwargs != nil {
			if v, ok2 := kwargs.GetStr("encoding"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					encoding = s.V
				}
			}
		}

		binaryMode := strings.TrimSuffix(mode, "t")
		if !strings.HasSuffix(binaryMode, "b") {
			binaryMode += "b"
		}
		isText := strings.HasSuffix(mode, "t")

		inst, err := i.newGzipFileInstance(gzipFileCls, badGzip, gzipDecompress, filename.V, binaryMode, level, gzipZeroTime)
		if err != nil {
			return nil, err
		}

		if !isText {
			return inst, nil
		}

		// Wrap for text mode
		return i.wrapGzipFileAsText(inst, encoding), nil
	}})

	return m
}

// buildGzipFileClass constructs the GzipFile class object.
func (i *Interp) buildGzipFileClass(badGzip *object.Class, gzipDecompress func([]byte) ([]byte, error)) *object.Class {
	cls := &object.Class{Name: "GzipFile", Dict: object.NewDict()}

	init := &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		// args[0]=self, then positional: filename, mode, compresslevel, fileobj, mtime
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "GzipFile() missing arguments")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "GzipFile() self must be instance")
		}

		filename := ""
		mode := "rb"
		level := gzip.DefaultCompression
		var mtime time.Time = gzipZeroTime

		if len(args) >= 2 {
			if s, ok2 := args[1].(*object.Str); ok2 {
				filename = s.V
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("filename"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					filename = s.V
				}
			}
		}
		if len(args) >= 3 {
			if s, ok2 := args[2].(*object.Str); ok2 {
				mode = s.V
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("mode"); ok2 {
				if s, ok3 := v.(*object.Str); ok3 {
					mode = s.V
				}
			}
		}
		if len(args) >= 4 {
			if n, ok2 := toInt64(args[3]); ok2 {
				level = int(n)
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("compresslevel"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					level = int(n)
				}
			}
		}
		if kwargs != nil {
			if v, ok2 := kwargs.GetStr("mtime"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					mtime = time.Unix(n, 0)
				}
			}
		}

		return nil, i.initGzipFileInstance(self, badGzip, gzipDecompress, filename, mode, level, mtime)
	}}
	cls.Dict.SetStr("__init__", init)

	return cls
}

// newGzipFileInstance creates and fully initialises a GzipFile instance.
func (i *Interp) newGzipFileInstance(cls *object.Class, badGzip *object.Class, gzipDecompress func([]byte) ([]byte, error), filename, mode string, level int, mtime time.Time) (*object.Instance, error) {
	d := object.NewDict()
	inst := &object.Instance{Class: cls, Dict: d}
	if err := i.initGzipFileInstance(inst, badGzip, gzipDecompress, filename, mode, level, mtime); err != nil {
		return nil, err
	}
	return inst, nil
}

// initGzipFileInstance wires up all methods and state on an already-allocated instance.
func (i *Interp) initGzipFileInstance(inst *object.Instance, badGzip *object.Class, gzipDecompress func([]byte) ([]byte, error), filename, mode string, level int, mtime time.Time) error {
	d := inst.Dict
	isWrite := strings.Contains(mode, "w") || strings.Contains(mode, "a")
	isAppend := strings.Contains(mode, "a")
	closed := false
	var pos int64

	// ── Write mode ────────────────────────────────────────────────────────────
	if isWrite {
		openFlag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		if isAppend {
			openFlag = os.O_CREATE | os.O_APPEND | os.O_WRONLY
		}
		osFile, err := os.OpenFile(filename, openFlag, 0644)
		if err != nil {
			return object.Errorf(i.osErr, "%v", err)
		}
		gzW, err := gzip.NewWriterLevel(osFile, level)
		if err != nil {
			osFile.Close()
			return object.Errorf(i.valueErr, "%v", err)
		}
		gzW.ModTime = mtime

		d.SetStr("mode", &object.Str{V: "wb"})
		d.SetStr("name", &object.Str{V: filename})
		d.SetStr("closed", object.False)
		d.SetStr("mtime", object.None)

		checkOpen := func() error {
			if closed {
				return object.Errorf(i.valueErr, "write to closed GzipFile")
			}
			return nil
		}

		writeFn := &object.BuiltinFunc{Name: "write", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if len(args) == 0 {
				return object.NewInt(0), nil
			}
			data, err := asBytes(args[0])
			if err != nil {
				return nil, err
			}
			n, err := gzW.Write(data)
			if err != nil {
				return nil, object.Errorf(badGzip, "%v", err)
			}
			pos += int64(n)
			return object.NewInt(int64(n)), nil
		}}
		d.SetStr("write", writeFn)

		flushFn := &object.BuiltinFunc{Name: "flush", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if err := gzW.Flush(); err != nil {
				return nil, object.Errorf(badGzip, "%v", err)
			}
			return object.None, nil
		}}
		d.SetStr("flush", flushFn)

		closeFn := &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return object.None, nil
			}
			closed = true
			d.SetStr("closed", object.True)
			if err := gzW.Close(); err != nil {
				osFile.Close()
				return nil, object.Errorf(badGzip, "%v", err)
			}
			osFile.Close()
			return object.None, nil
		}}
		d.SetStr("close", closeFn)

		tellFn := &object.BuiltinFunc{Name: "tell", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(pos), nil
		}}
		d.SetStr("tell", tellFn)

		readFn := &object.BuiltinFunc{Name: "read", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "read() on write-mode GzipFile")
		}}
		d.SetStr("read", readFn)
		d.SetStr("read1", readFn)
		d.SetStr("readline", readFn)

		seekFn := &object.BuiltinFunc{Name: "seek", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "seek() on write-mode GzipFile")
		}}
		d.SetStr("seek", seekFn)

		d.SetStr("readable", &object.BuiltinFunc{Name: "readable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		}})
		d.SetStr("writable", &object.BuiltinFunc{Name: "writable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})
		d.SetStr("seekable", &object.BuiltinFunc{Name: "seekable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})
		d.SetStr("peek", &object.BuiltinFunc{Name: "peek", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "peek() on write-mode GzipFile")
		}})

	} else {
		// ── Read mode ─────────────────────────────────────────────────────────
		raw, err := os.ReadFile(filename)
		if err != nil {
			return object.Errorf(i.osErr, "%v", err)
		}
		decompressed, err := gzipDecompress(raw)
		if err != nil {
			return err
		}

		d.SetStr("mode", &object.Str{V: "rb"})
		d.SetStr("name", &object.Str{V: filename})
		d.SetStr("closed", object.False)
		d.SetStr("mtime", object.None)

		checkOpen := func() error {
			if closed {
				return object.Errorf(i.valueErr, "read from closed GzipFile")
			}
			return nil
		}

		readFn := &object.BuiltinFunc{Name: "read", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			size := -1
			if len(args) >= 1 {
				if n, ok := toInt64(args[0]); ok {
					size = int(n)
				}
			}
			remaining := decompressed[pos:]
			if size < 0 || size >= len(remaining) {
				pos = int64(len(decompressed))
				return &object.Bytes{V: append([]byte{}, remaining...)}, nil
			}
			chunk := remaining[:size]
			pos += int64(size)
			return &object.Bytes{V: append([]byte{}, chunk...)}, nil
		}}
		d.SetStr("read", readFn)
		d.SetStr("read1", readFn)

		peekFn := &object.BuiltinFunc{Name: "peek", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			n := 1
			if len(args) >= 1 {
				if v, ok := toInt64(args[0]); ok {
					n = int(v)
				}
			}
			remaining := decompressed[pos:]
			if n > len(remaining) {
				n = len(remaining)
			}
			return &object.Bytes{V: append([]byte{}, remaining[:n]...)}, nil
		}}
		d.SetStr("peek", peekFn)

		readlineFn := &object.BuiltinFunc{Name: "readline", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			remaining := decompressed[pos:]
			if len(remaining) == 0 {
				return &object.Bytes{V: []byte{}}, nil
			}
			nl := bytes.IndexByte(remaining, '\n')
			var line []byte
			if nl < 0 {
				line = append([]byte{}, remaining...)
				pos = int64(len(decompressed))
			} else {
				line = append([]byte{}, remaining[:nl+1]...)
				pos += int64(nl + 1)
			}
			return &object.Bytes{V: line}, nil
		}}
		d.SetStr("readline", readlineFn)

		seekFn := &object.BuiltinFunc{Name: "seek", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if len(args) == 0 {
				return object.NewInt(pos), nil
			}
			offset, ok := toInt64(args[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "seek offset must be int")
			}
			whence := int64(0)
			if len(args) >= 2 {
				if w, ok2 := toInt64(args[1]); ok2 {
					whence = w
				}
			}
			var newPos int64
			switch whence {
			case 0:
				newPos = offset
			case 1:
				newPos = pos + offset
			case 2:
				newPos = int64(len(decompressed)) + offset
			}
			if newPos < 0 {
				newPos = 0
			}
			if newPos > int64(len(decompressed)) {
				newPos = int64(len(decompressed))
			}
			pos = newPos
			return object.NewInt(pos), nil
		}}
		d.SetStr("seek", seekFn)

		tellFn := &object.BuiltinFunc{Name: "tell", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(pos), nil
		}}
		d.SetStr("tell", tellFn)

		closeFn := &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if !closed {
				closed = true
				d.SetStr("closed", object.True)
			}
			return object.None, nil
		}}
		d.SetStr("close", closeFn)

		writeFn := &object.BuiltinFunc{Name: "write", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "write() on read-mode GzipFile")
		}}
		d.SetStr("write", writeFn)
		d.SetStr("flush", &object.BuiltinFunc{Name: "flush", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

		d.SetStr("readable", &object.BuiltinFunc{Name: "readable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})
		d.SetStr("writable", &object.BuiltinFunc{Name: "writable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		}})
		d.SetStr("seekable", &object.BuiltinFunc{Name: "seekable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})
	}

	// ── Shared (both modes) ───────────────────────────────────────────────────
	d.SetStr("fileno", &object.BuiltinFunc{Name: "fileno", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return nil, object.Errorf(i.osErr, "GzipFile does not support fileno()")
	}})

	d.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return inst, nil
	}})
	d.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if closeFn, ok := d.GetStr("close"); ok {
			if bf, ok2 := closeFn.(*object.BuiltinFunc); ok2 {
				bf.Call(nil, nil, nil) //nolint:errcheck
			}
		}
		return object.False, nil
	}})

	return nil
}

// wrapGzipFileAsText wraps a binary GzipFile instance with text encode/decode.
func (i *Interp) wrapGzipFileAsText(binInst *object.Instance, encoding string) *object.Instance {
	cls := &object.Class{Name: "GzipFile", Dict: object.NewDict()}
	d := object.NewDict()
	wrapper := &object.Instance{Class: cls, Dict: d}

	getMethod := func(name string) *object.BuiltinFunc {
		if v, ok := binInst.Dict.GetStr(name); ok {
			if bf, ok2 := v.(*object.BuiltinFunc); ok2 {
				return bf
			}
		}
		return nil
	}
	callNoArgs := func(name string) (object.Object, error) {
		if bf := getMethod(name); bf != nil {
			return bf.Call(nil, nil, nil)
		}
		return object.None, nil
	}

	// read → decode bytes to str
	d.SetStr("read", &object.BuiltinFunc{Name: "read", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		bf := getMethod("read")
		if bf == nil {
			return &object.Str{V: ""}, nil
		}
		result, err := bf.Call(nil, args, nil)
		if err != nil {
			return nil, err
		}
		if b, ok := result.(*object.Bytes); ok {
			return &object.Str{V: string(b.V)}, nil
		}
		return result, nil
	}})

	// write → encode str to bytes
	d.SetStr("write", &object.BuiltinFunc{Name: "write", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		bf := getMethod("write")
		if bf == nil || len(args) == 0 {
			return object.NewInt(0), nil
		}
		var data object.Object
		switch v := args[0].(type) {
		case *object.Str:
			_ = encoding
			data = &object.Bytes{V: []byte(v.V)}
		default:
			data = args[0]
		}
		return bf.Call(nil, []object.Object{data}, nil)
	}})

	for _, name := range []string{"close", "flush", "readable", "writable", "seekable", "tell", "seek", "__enter__", "__exit__"} {
		d.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
			if name == "__enter__" {
				return wrapper, nil
			}
			return callNoArgs(name)
		}})
	}
	d.SetStr("closed", object.False)

	return wrapper
}

// ── package-level ─────────────────────────────────────────────────────────────

// gzipZeroTime is the deterministic zero mtime written by gzip.compress.
var gzipZeroTime = time.Time{}

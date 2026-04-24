package vm

import (
	"bytes"
	stdbzip2 "compress/bzip2"
	"io"
	"os"
	"strings"

	"github.com/tamnd/goipy/lib/bzip2"
	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildBz2() *object.Module {
	m := &object.Module{Name: "bz2", Dict: object.NewDict()}

	// bz2Decompress decompresses one or more concatenated bzip2 streams.
	bz2Decompress := func(data []byte) ([]byte, error) {
		var out []byte
		src := bytes.NewReader(data)
		for src.Len() > 0 {
			r := stdbzip2.NewReader(src)
			part, err := io.ReadAll(r)
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			out = append(out, part...)
		}
		return out, nil
	}

	// compress(data, compresslevel=9)
	m.Dict.SetStr("compress", &object.BuiltinFunc{Name: "compress", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "compress() missing data")
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		level := 9
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
		if level < 1 || level > 9 {
			return nil, object.Errorf(i.valueErr, "compresslevel must be between 1 and 9")
		}
		return &object.Bytes{V: bzip2.Encode(data, level)}, nil
	}})

	// decompress(data)
	m.Dict.SetStr("decompress", &object.BuiltinFunc{Name: "decompress", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "decompress() missing data")
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		out, err := bz2Decompress(data)
		if err != nil {
			return nil, err
		}
		return &object.Bytes{V: out}, nil
	}})

	m.Dict.SetStr("BZ2Compressor", i.buildBZ2CompressorClass())
	m.Dict.SetStr("BZ2Decompressor", i.buildBZ2DecompressorClass())

	bz2FileCls := i.buildBZ2FileClass(bz2Decompress)
	m.Dict.SetStr("BZ2File", bz2FileCls)

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
		level := 9
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

		binaryMode := strings.TrimSuffix(mode, "t")
		if !strings.HasSuffix(binaryMode, "b") {
			binaryMode += "b"
		}
		isText := strings.HasSuffix(mode, "t")

		inst, err := i.newBZ2FileInstance(bz2FileCls, bz2Decompress, filename.V, binaryMode, level)
		if err != nil {
			return nil, err
		}
		if !isText {
			return inst, nil
		}
		return i.wrapBZ2FileAsText(inst), nil
	}})

	return m
}

// ── BZ2Compressor ─────────────────────────────────────────────────────────

func (i *Interp) buildBZ2CompressorClass() *object.Class {
	cls := &object.Class{Name: "BZ2Compressor", Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "BZ2Compressor() missing self")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "BZ2Compressor() self must be instance")
		}
		level := 9
		if len(args) >= 2 {
			if n, ok2 := toInt64(args[1]); ok2 {
				level = int(n)
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("compresslevel"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					level = int(n)
				}
			}
		}
		if level < 1 || level > 9 {
			return nil, object.Errorf(i.valueErr, "compresslevel must be between 1 and 9")
		}
		return nil, i.initBZ2Compressor(self, level)
	}})

	return cls
}

// initBZ2Compressor buffers all input until flush(); bzip2 is a block-oriented
// format so there's no natural streaming boundary. On flush() we encode the
// whole buffer and return the full stream.
func (i *Interp) initBZ2Compressor(inst *object.Instance, level int) error {
	var buf []byte
	flushed := false
	d := inst.Dict

	d.SetStr("compress", &object.BuiltinFunc{Name: "compress", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if flushed {
			return nil, object.Errorf(i.valueErr, "compressor already flushed")
		}
		if len(args) == 0 {
			return &object.Bytes{V: []byte{}}, nil
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		buf = append(buf, data...)
		// bzip2 emits nothing until the block closes; mirror that.
		return &object.Bytes{V: []byte{}}, nil
	}})

	d.SetStr("flush", &object.BuiltinFunc{Name: "flush", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if flushed {
			return nil, object.Errorf(i.valueErr, "compressor already flushed")
		}
		flushed = true
		out := bzip2.Encode(buf, level)
		buf = nil
		return &object.Bytes{V: out}, nil
	}})

	return nil
}

// ── BZ2Decompressor ───────────────────────────────────────────────────────

func (i *Interp) buildBZ2DecompressorClass() *object.Class {
	cls := &object.Class{Name: "BZ2Decompressor", Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "BZ2Decompressor() missing self")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "BZ2Decompressor() self must be instance")
		}
		return nil, i.initBZ2Decompressor(self)
	}})

	return cls
}

func (i *Interp) initBZ2Decompressor(inst *object.Instance) error {
	d := inst.Dict
	var accumulated []byte
	var outBuf []byte
	eof := false

	d.SetStr("eof", object.False)
	d.SetStr("unused_data", &object.Bytes{V: []byte{}})
	d.SetStr("needs_input", object.True)

	d.SetStr("decompress", &object.BuiltinFunc{Name: "decompress", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if eof && len(args) > 0 {
			if data, err := asBytes(args[0]); err == nil && len(data) > 0 {
				return nil, object.Errorf(i.eofErr, "end of stream already reached")
			}
		}
		maxLength := -1
		if len(args) >= 2 {
			if n, ok := toInt64(args[1]); ok {
				maxLength = int(n)
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("max_length"); ok {
				if n, ok2 := toInt64(v); ok2 {
					maxLength = int(n)
				}
			}
		}
		if len(args) >= 1 {
			data, err := asBytes(args[0])
			if err != nil {
				return nil, err
			}
			accumulated = append(accumulated, data...)
		}
		// Attempt full decompression of the first bzip2 stream. The Go
		// decoder treats trailing bytes as an attempted next stream and
		// errors if the magic doesn't match, over-reading the underlying
		// reader by a few bytes in the process. When that happens we
		// probe shorter prefixes to find the exact stream end so we can
		// expose trailing bytes via unused_data.
		if !eof && len(accumulated) > 0 {
			innerRd := bytes.NewReader(accumulated)
			r := stdbzip2.NewReader(innerRd)
			data, rerr := io.ReadAll(r)
			if rerr == nil {
				outBuf = append(outBuf, data...)
				eof = true
				d.SetStr("eof", object.True)
			} else if len(data) > 0 {
				consumed := len(accumulated) - innerRd.Len()
				streamEnd := -1
				for k := consumed; k > 0 && k >= consumed-16; k-- {
					tryRd := bytes.NewReader(accumulated[:k])
					tryData, tryErr := io.ReadAll(stdbzip2.NewReader(tryRd))
					if tryErr == nil && bytes.Equal(tryData, data) {
						streamEnd = k
						break
					}
				}
				if streamEnd >= 0 {
					outBuf = append(outBuf, data...)
					eof = true
					d.SetStr("eof", object.True)
					if streamEnd < len(accumulated) {
						unused := append([]byte{}, accumulated[streamEnd:]...)
						d.SetStr("unused_data", &object.Bytes{V: unused})
					}
				}
			}
		}
		var chunk []byte
		if maxLength >= 0 && maxLength < len(outBuf) {
			chunk = append([]byte{}, outBuf[:maxLength]...)
			outBuf = outBuf[maxLength:]
		} else {
			chunk = append([]byte{}, outBuf...)
			outBuf = outBuf[:0]
		}
		if len(outBuf) > 0 {
			d.SetStr("needs_input", object.False)
		} else {
			d.SetStr("needs_input", object.True)
		}
		return &object.Bytes{V: chunk}, nil
	}})

	return nil
}

// ── BZ2File ───────────────────────────────────────────────────────────────

func (i *Interp) buildBZ2FileClass(bz2Decompress func([]byte) ([]byte, error)) *object.Class {
	cls := &object.Class{Name: "BZ2File", Dict: object.NewDict()}

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "BZ2File() missing self")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "BZ2File() self must be instance")
		}
		filename := ""
		mode := "rb"
		level := 9
		if len(args) >= 2 {
			if s, ok2 := args[1].(*object.Str); ok2 {
				filename = s.V
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
		if kwargs != nil {
			if v, ok2 := kwargs.GetStr("compresslevel"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					level = int(n)
				}
			}
		}
		return nil, i.initBZ2FileInstance(self, bz2Decompress, filename, mode, level)
	}})

	return cls
}

func (i *Interp) newBZ2FileInstance(cls *object.Class, bz2Decompress func([]byte) ([]byte, error), filename, mode string, level int) (*object.Instance, error) {
	d := object.NewDict()
	inst := &object.Instance{Class: cls, Dict: d}
	if err := i.initBZ2FileInstance(inst, bz2Decompress, filename, mode, level); err != nil {
		return nil, err
	}
	return inst, nil
}

func (i *Interp) initBZ2FileInstance(inst *object.Instance, bz2Decompress func([]byte) ([]byte, error), filename, mode string, level int) error {
	d := inst.Dict
	isWrite := strings.Contains(mode, "w") || strings.Contains(mode, "a") || strings.Contains(mode, "x")
	isAppend := strings.Contains(mode, "a")
	isExclusive := strings.Contains(mode, "x")
	closed := false
	var pos int64

	if isWrite {
		openFlag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		if isAppend {
			openFlag = os.O_CREATE | os.O_APPEND | os.O_WRONLY
		}
		if isExclusive {
			openFlag = os.O_CREATE | os.O_EXCL | os.O_WRONLY
		}
		osFile, err := os.OpenFile(filename, openFlag, 0644)
		if err != nil {
			return object.Errorf(i.osErr, "%v", err)
		}
		// Buffer all writes; encode on close (bzip2 needs the full block).
		var writeBuf []byte

		d.SetStr("mode", &object.Str{V: "wb"})
		d.SetStr("name", &object.Str{V: filename})
		d.SetStr("closed", object.False)

		checkOpen := func() error {
			if closed {
				return object.Errorf(i.valueErr, "write to closed BZ2File")
			}
			return nil
		}

		d.SetStr("write", &object.BuiltinFunc{Name: "write", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
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
			writeBuf = append(writeBuf, data...)
			pos += int64(len(data))
			return object.NewInt(int64(len(data))), nil
		}})

		d.SetStr("writelines", &object.BuiltinFunc{Name: "writelines", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			if len(args) == 0 {
				return object.None, nil
			}
			var items []object.Object
			switch lst := args[0].(type) {
			case *object.List:
				items = lst.V
			case *object.Tuple:
				items = lst.V
			}
			for _, item := range items {
				data, err := asBytes(item)
				if err != nil {
					return nil, err
				}
				writeBuf = append(writeBuf, data...)
				pos += int64(len(data))
			}
			return object.None, nil
		}})

		d.SetStr("flush", &object.BuiltinFunc{Name: "flush", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

		d.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if closed {
				return object.None, nil
			}
			closed = true
			d.SetStr("closed", object.True)
			encoded := bzip2.Encode(writeBuf, level)
			if _, err := osFile.Write(encoded); err != nil {
				osFile.Close()
				return nil, object.Errorf(i.osErr, "%v", err)
			}
			osFile.Close()
			return object.None, nil
		}})

		d.SetStr("tell", &object.BuiltinFunc{Name: "tell", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(pos), nil
		}})

		readErr := &object.BuiltinFunc{Name: "read", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "read on write-mode BZ2File")
		}}
		d.SetStr("read", readErr)
		d.SetStr("read1", readErr)
		d.SetStr("readline", readErr)
		d.SetStr("readlines", readErr)
		d.SetStr("readinto", readErr)
		d.SetStr("peek", readErr)
		d.SetStr("seek", &object.BuiltinFunc{Name: "seek", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "seek on write-mode BZ2File")
		}})
		d.SetStr("readable", &object.BuiltinFunc{Name: "readable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		}})
		d.SetStr("writable", &object.BuiltinFunc{Name: "writable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.True, nil
		}})
		d.SetStr("seekable", &object.BuiltinFunc{Name: "seekable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		}})

	} else {
		raw, err := os.ReadFile(filename)
		if err != nil {
			return object.Errorf(i.osErr, "%v", err)
		}
		decompressed, err := bz2Decompress(raw)
		if err != nil {
			return err
		}

		d.SetStr("mode", &object.Str{V: "rb"})
		d.SetStr("name", &object.Str{V: filename})
		d.SetStr("closed", object.False)

		checkOpen := func() error {
			if closed {
				return object.Errorf(i.valueErr, "read from closed BZ2File")
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

		d.SetStr("peek", &object.BuiltinFunc{Name: "peek", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
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
		}})

		d.SetStr("readline", &object.BuiltinFunc{Name: "readline", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
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
		}})

		d.SetStr("readlines", &object.BuiltinFunc{Name: "readlines", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if err := checkOpen(); err != nil {
				return nil, err
			}
			var lines []object.Object
			for {
				remaining := decompressed[pos:]
				if len(remaining) == 0 {
					break
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
				lines = append(lines, &object.Bytes{V: line})
			}
			return &object.List{V: lines}, nil
		}})

		d.SetStr("seek", &object.BuiltinFunc{Name: "seek", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
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
		}})

		d.SetStr("tell", &object.BuiltinFunc{Name: "tell", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(pos), nil
		}})

		d.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			if !closed {
				closed = true
				d.SetStr("closed", object.True)
			}
			return object.None, nil
		}})

		d.SetStr("write", &object.BuiltinFunc{Name: "write", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "write on read-mode BZ2File")
		}})
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

	d.SetStr("fileno", &object.BuiltinFunc{Name: "fileno", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return nil, object.Errorf(i.osErr, "BZ2File does not support fileno()")
	}})

	d.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return inst, nil
	}})
	d.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if closeFn, ok := d.GetStr("close"); ok {
			if bf, ok2 := closeFn.(*object.BuiltinFunc); ok2 {
				bf.Call(nil, nil, nil) //nolint:errcheck
			}
		}
		return object.False, nil
	}})

	return nil
}

// wrapBZ2FileAsText wraps a binary BZ2File with UTF-8 encode/decode on IO.
func (i *Interp) wrapBZ2FileAsText(binInst *object.Instance) *object.Instance {
	cls := &object.Class{Name: "BZ2File", Dict: object.NewDict()}
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

	d.SetStr("write", &object.BuiltinFunc{Name: "write", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		bf := getMethod("write")
		if bf == nil || len(args) == 0 {
			return object.NewInt(0), nil
		}
		var data object.Object
		switch v := args[0].(type) {
		case *object.Str:
			data = &object.Bytes{V: []byte(v.V)}
		default:
			data = args[0]
		}
		return bf.Call(nil, []object.Object{data}, nil)
	}})

	for _, name := range []string{"close", "flush", "readable", "writable", "seekable", "tell", "seek", "readline", "__enter__", "__exit__"} {
		d.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
			if name == "__enter__" {
				return wrapper, nil
			}
			bf := getMethod(name)
			if bf == nil {
				return object.None, nil
			}
			return bf.Call(nil, args, nil)
		}})
	}
	d.SetStr("closed", object.False)

	return wrapper
}

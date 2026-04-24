package vm

import (
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/tamnd/goipy/lib/zstd"
	"github.com/tamnd/goipy/object"
)

// Python `compression.zstd` stdlib module (new in 3.14).
//
// The bytes we emit are valid RFC 8878 zstd frames with XXH64
// content checksums; any compliant decoder (CPython's included) can
// consume them. Our decoder handles Raw and RLE blocks only, which
// is sufficient for round-trips against our own encoder.

func (i *Interp) buildCompression() *object.Module {
	m := &object.Module{Name: "compression", Dict: object.NewDict()}
	m.Dict.SetStr("zstd", i.buildCompressionZstd())
	return m
}

func (i *Interp) buildCompressionZstd() *object.Module {
	m := &object.Module{Name: "compression.zstd", Dict: object.NewDict()}

	zstdErr := &object.Class{Name: "ZstdError", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	m.Dict.SetStr("ZstdError", zstdErr)

	m.Dict.SetStr("COMPRESSION_LEVEL_DEFAULT", object.NewInt(3))
	m.Dict.SetStr("zstd_version", &object.Str{V: "1.5.7"})
	m.Dict.SetStr("zstd_version_info", &object.Tuple{V: []object.Object{
		object.NewInt(1), object.NewInt(5), object.NewInt(7),
	}})

	m.Dict.SetStr("compress", &object.BuiltinFunc{Name: "compress", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "compress() missing data")
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		return &object.Bytes{V: zstd.Encode(data)}, nil
	}})

	m.Dict.SetStr("decompress", &object.BuiltinFunc{Name: "decompress", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "decompress() missing data")
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		out, derr := zstd.DecodeAll(data)
		if derr != nil {
			return nil, object.Errorf(zstdErr, "%v", derr)
		}
		return &object.Bytes{V: out}, nil
	}})

	compressorCls := i.buildZstdCompressorClass(zstdErr)
	m.Dict.SetStr("ZstdCompressor", compressorCls)

	decompressorCls := i.buildZstdDecompressorClass(zstdErr)
	m.Dict.SetStr("ZstdDecompressor", decompressorCls)

	dictCls := i.buildZstdDictClass()
	m.Dict.SetStr("ZstdDict", dictCls)

	fileCls := i.buildZstdFileClass(zstdErr)
	m.Dict.SetStr("ZstdFile", fileCls)

	m.Dict.SetStr("open", &object.BuiltinFunc{Name: "open", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "open() missing file")
		}
		name, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "open() file must be str")
		}
		mode := "rb"
		if len(args) >= 2 {
			if s, ok := args[1].(*object.Str); ok {
				mode = s.V
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("mode"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					mode = s.V
				}
			}
		}
		return i.openZstdFile(name.V, mode, fileCls, zstdErr)
	}})

	// Parameter enums — expose as simple classes with int members.
	m.Dict.SetStr("CompressionParameter", buildZstdEnum("CompressionParameter",
		[]string{"compression_level", "checksum_flag", "window_log"}))
	m.Dict.SetStr("DecompressionParameter", buildZstdEnum("DecompressionParameter",
		[]string{"window_log_max"}))
	m.Dict.SetStr("Strategy", buildZstdEnum("Strategy",
		[]string{"fast", "dfast", "greedy", "lazy", "lazy2", "btlazy2", "btopt", "btultra", "btultra2"}))

	// Module-level finalize_dict / train_dict stubs.
	m.Dict.SetStr("train_dict", &object.BuiltinFunc{Name: "train_dict", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) >= 2 {
			if n, ok := toInt64(args[1]); ok {
				return &object.Bytes{V: make([]byte, int(n))}, nil
			}
		}
		return &object.Bytes{V: nil}, nil
	}})
	m.Dict.SetStr("finalize_dict", &object.BuiltinFunc{Name: "finalize_dict", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) >= 1 {
			if b, ok := args[0].(*object.Bytes); ok {
				return b, nil
			}
		}
		return &object.Bytes{V: nil}, nil
	}})

	return m
}

// ── ZstdCompressor ────────────────────────────────────────────────────────

const (
	zstdModeContinue   = 0
	zstdModeFlushBlock = 1
	zstdModeFlushFrame = 2
)

type zstdCompressorState struct {
	buf bytes.Buffer
}

func (i *Interp) buildZstdCompressorClass(zstdErr *object.Class) *object.Class {
	cls := &object.Class{Name: "ZstdCompressor", Dict: object.NewDict()}
	cls.Dict.SetStr("CONTINUE", object.NewInt(zstdModeContinue))
	cls.Dict.SetStr("FLUSH_BLOCK", object.NewInt(zstdModeFlushBlock))
	cls.Dict.SetStr("FLUSH_FRAME", object.NewInt(zstdModeFlushFrame))

	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "ZstdCompressor() missing self")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "ZstdCompressor() self must be instance")
		}
		st := &zstdCompressorState{}
		d := self.Dict

		d.SetStr("compress", &object.BuiltinFunc{Name: "compress", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "compress() missing data")
			}
			data, err := asBytes(a[0])
			if err != nil {
				return nil, err
			}
			mode := zstdModeContinue
			if len(a) >= 2 {
				if n, ok2 := toInt64(a[1]); ok2 {
					mode = int(n)
				}
			}
			st.buf.Write(data)
			if mode == zstdModeFlushFrame {
				out := zstd.Encode(st.buf.Bytes())
				st.buf.Reset()
				return &object.Bytes{V: out}, nil
			}
			return &object.Bytes{V: nil}, nil
		}})

		d.SetStr("flush", &object.BuiltinFunc{Name: "flush", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			out := zstd.Encode(st.buf.Bytes())
			st.buf.Reset()
			return &object.Bytes{V: out}, nil
		}})

		d.SetStr("CONTINUE", object.NewInt(zstdModeContinue))
		d.SetStr("FLUSH_BLOCK", object.NewInt(zstdModeFlushBlock))
		d.SetStr("FLUSH_FRAME", object.NewInt(zstdModeFlushFrame))
		return nil, nil
	}})
	return cls
}

// ── ZstdDecompressor ──────────────────────────────────────────────────────

type zstdDecompressorState struct {
	pending    []byte // accumulated input not yet decoded
	overflow   []byte // already-decoded bytes that didn't fit under max_length
	unused     []byte // bytes after a complete frame
	eof        bool
	needsInput bool
}

func (i *Interp) buildZstdDecompressorClass(zstdErr *object.Class) *object.Class {
	cls := &object.Class{Name: "ZstdDecompressor", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "ZstdDecompressor() missing self")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "ZstdDecompressor() self must be instance")
		}
		st := &zstdDecompressorState{needsInput: true}
		d := self.Dict

		refresh := func() {
			d.SetStr("eof", object.BoolOf(st.eof))
			d.SetStr("needs_input", object.BoolOf(st.needsInput))
			d.SetStr("unused_data", &object.Bytes{V: append([]byte(nil), st.unused...)})
		}
		refresh()

		d.SetStr("decompress", &object.BuiltinFunc{Name: "decompress", Call: func(_ any, a []object.Object, kwargs *object.Dict) (object.Object, error) {
			if len(a) == 0 {
				return nil, object.Errorf(i.typeErr, "decompress() missing data")
			}
			data, err := asBytes(a[0])
			if err != nil {
				return nil, err
			}
			maxLen := -1
			if len(a) >= 2 {
				if n, ok2 := toInt64(a[1]); ok2 {
					maxLen = int(n)
				}
			} else if kwargs != nil {
				if v, ok2 := kwargs.GetStr("max_length"); ok2 {
					if n, ok3 := toInt64(v); ok3 {
						maxLen = int(n)
					}
				}
			}

			st.pending = append(st.pending, data...)
			var out []byte
			if len(st.overflow) > 0 {
				out = st.overflow
				st.overflow = nil
			}

			// Try to decode one complete frame from pending.
			if !st.eof {
				if dec, n, derr := zstd.Decode(st.pending); derr == nil {
					out = append(out, dec...)
					st.unused = append([]byte(nil), st.pending[n:]...)
					st.pending = nil
					st.eof = true
					st.needsInput = false
				} else {
					// Truncated / not enough input: keep pending and
					// wait for more.
					st.needsInput = true
				}
			}

			// Apply max_length.
			if maxLen >= 0 && len(out) > maxLen {
				st.overflow = append([]byte(nil), out[maxLen:]...)
				out = out[:maxLen]
				st.needsInput = false
			}
			refresh()
			return &object.Bytes{V: out}, nil
		}})
		return nil, nil
	}})
	return cls
}

// ── ZstdDict ──────────────────────────────────────────────────────────────

func (i *Interp) buildZstdDictClass() *object.Class {
	cls := &object.Class{Name: "ZstdDict", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "ZstdDict() requires dict_content")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "ZstdDict() self must be instance")
		}
		content, _ := asBytes(args[1])
		self.Dict.SetStr("dict_content", &object.Bytes{V: content})
		self.Dict.SetStr("dict_id", object.NewInt(0))
		return nil, nil
	}})
	return cls
}

// ── ZstdFile ──────────────────────────────────────────────────────────────

func (i *Interp) buildZstdFileClass(zstdErr *object.Class) *object.Class {
	cls := &object.Class{Name: "ZstdFile", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "ZstdFile() requires file")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "ZstdFile() self must be instance")
		}
		name, ok := args[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "ZstdFile() file must be str")
		}
		mode := "rb"
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
		return nil, i.initZstdFileInstance(self, name.V, mode, zstdErr)
	}})
	return cls
}

func (i *Interp) openZstdFile(name, mode string, fileCls, zstdErr *object.Class) (object.Object, error) {
	inst := &object.Instance{Class: fileCls, Dict: object.NewDict()}
	if err := i.initZstdFileInstance(inst, name, mode, zstdErr); err != nil {
		return nil, err
	}
	if strings.Contains(mode, "t") {
		return i.wrapZstdFileText(inst, mode), nil
	}
	return inst, nil
}

func (i *Interp) initZstdFileInstance(inst *object.Instance, name, mode string, zstdErr *object.Class) error {
	d := inst.Dict
	isRead := !strings.ContainsAny(mode, "wxa")
	d.SetStr("name", &object.Str{V: name})
	d.SetStr("mode", &object.Str{V: strings.ReplaceAll(mode, "t", "b")})
	d.SetStr("closed", object.False)

	var data []byte
	var pos int
	var wbuf bytes.Buffer
	if isRead {
		raw, err := os.ReadFile(name)
		if err != nil {
			return object.Errorf(zstdErr, "%v", err)
		}
		dec, derr := zstd.DecodeAll(raw)
		if derr != nil {
			return object.Errorf(zstdErr, "%v", derr)
		}
		data = dec
	}

	close := func() {
		if v, ok := d.GetStr("closed"); ok && v == object.True {
			return
		}
		d.SetStr("closed", object.True)
		if !isRead {
			if err := os.WriteFile(name, zstd.Encode(wbuf.Bytes()), 0644); err != nil {
				// Silent on close — mirror CPython behaviour (exception at
				// close is rare for compressed files).
				_ = err
			}
		}
	}

	d.SetStr("read", &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if !isRead {
			return nil, object.Errorf(i.valueErr, "file not open for reading")
		}
		size := -1
		if len(a) >= 1 {
			if n, ok := toInt64(a[0]); ok {
				size = int(n)
			}
		}
		remaining := data[pos:]
		if size < 0 || size >= len(remaining) {
			pos = len(data)
			return &object.Bytes{V: append([]byte{}, remaining...)}, nil
		}
		chunk := remaining[:size]
		pos += size
		return &object.Bytes{V: append([]byte{}, chunk...)}, nil
	}})

	d.SetStr("readline", &object.BuiltinFunc{Name: "readline", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if !isRead {
			return nil, object.Errorf(i.valueErr, "file not open for reading")
		}
		remaining := data[pos:]
		if len(remaining) == 0 {
			return &object.Bytes{V: []byte{}}, nil
		}
		nl := bytes.IndexByte(remaining, '\n')
		var line []byte
		if nl < 0 {
			line = append([]byte{}, remaining...)
			pos = len(data)
		} else {
			line = append([]byte{}, remaining[:nl+1]...)
			pos += nl + 1
		}
		return &object.Bytes{V: line}, nil
	}})

	d.SetStr("readlines", &object.BuiltinFunc{Name: "readlines", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if !isRead {
			return nil, object.Errorf(i.valueErr, "file not open for reading")
		}
		var lines []object.Object
		for pos < len(data) {
			remaining := data[pos:]
			nl := bytes.IndexByte(remaining, '\n')
			var line []byte
			if nl < 0 {
				line = append([]byte{}, remaining...)
				pos = len(data)
			} else {
				line = append([]byte{}, remaining[:nl+1]...)
				pos += nl + 1
			}
			lines = append(lines, &object.Bytes{V: line})
		}
		return &object.List{V: lines}, nil
	}})

	d.SetStr("write", &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if isRead {
			return nil, object.Errorf(i.valueErr, "file not open for writing")
		}
		if len(a) == 0 {
			return object.NewInt(0), nil
		}
		b, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		wbuf.Write(b)
		return object.NewInt(int64(len(b))), nil
	}})

	d.SetStr("seek", &object.BuiltinFunc{Name: "seek", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if !isRead {
			return nil, object.Errorf(i.valueErr, "file not seekable")
		}
		if len(a) == 0 {
			return object.NewInt(int64(pos)), nil
		}
		off, _ := toInt64(a[0])
		whence := int64(0)
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok {
				whence = n
			}
		}
		switch whence {
		case 0:
			pos = int(off)
		case 1:
			pos += int(off)
		case 2:
			pos = len(data) + int(off)
		}
		if pos < 0 {
			pos = 0
		}
		if pos > len(data) {
			pos = len(data)
		}
		return object.NewInt(int64(pos)), nil
	}})

	d.SetStr("tell", &object.BuiltinFunc{Name: "tell", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.NewInt(int64(pos)), nil
	}})

	d.SetStr("peek", &object.BuiltinFunc{Name: "peek", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if !isRead {
			return &object.Bytes{V: nil}, nil
		}
		size := 1
		if len(a) >= 1 {
			if n, ok := toInt64(a[0]); ok {
				size = int(n)
			}
		}
		remaining := data[pos:]
		if size > len(remaining) {
			size = len(remaining)
		}
		return &object.Bytes{V: append([]byte{}, remaining[:size]...)}, nil
	}})

	d.SetStr("readable", &object.BuiltinFunc{Name: "readable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(isRead), nil
	}})
	d.SetStr("writable", &object.BuiltinFunc{Name: "writable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(!isRead), nil
	}})
	d.SetStr("seekable", &object.BuiltinFunc{Name: "seekable", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.BoolOf(isRead), nil
	}})

	d.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		close()
		return object.None, nil
	}})
	d.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return inst, nil
	}})
	d.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		close()
		return object.False, nil
	}})

	_ = io.EOF
	return nil
}

// wrapZstdFileText wraps a binary ZstdFile instance as a text-mode
// file by layering read/write over UTF-8.
func (i *Interp) wrapZstdFileText(bin *object.Instance, mode string) *object.Instance {
	cls := &object.Class{Name: "ZstdTextFile", Dict: object.NewDict()}
	d := object.NewDict()
	inst := &object.Instance{Class: cls, Dict: d}

	callBin := func(name string, args []object.Object) (object.Object, error) {
		fn, ok := bin.Dict.GetStr(name)
		if !ok {
			return nil, object.Errorf(i.typeErr, "binary method %s missing", name)
		}
		bf, ok2 := fn.(*object.BuiltinFunc)
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "binary %s not callable", name)
		}
		return bf.Call(nil, args, nil)
	}

	d.SetStr("read", &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		res, err := callBin("read", a)
		if err != nil {
			return nil, err
		}
		if b, ok := res.(*object.Bytes); ok {
			return &object.Str{V: string(b.V)}, nil
		}
		return object.None, nil
	}})
	d.SetStr("write", &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return object.NewInt(0), nil
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "write() argument must be str")
		}
		return callBin("write", []object.Object{&object.Bytes{V: []byte(s.V)}})
	}})
	d.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return callBin("close", a)
	}})
	d.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return inst, nil
	}})
	d.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return callBin("close", nil)
	}})
	return inst
}

// buildZstdEnum produces a lightweight class with named int members
// used as an enum-like placeholder.
func buildZstdEnum(name string, members []string) *object.Class {
	cls := &object.Class{Name: name, Dict: object.NewDict()}
	for idx, m := range members {
		cls.Dict.SetStr(m, object.NewInt(int64(idx)))
	}
	return cls
}

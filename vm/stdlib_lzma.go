package vm

import (
	"bytes"
	"os"
	"strings"

	"github.com/tamnd/goipy/lib/lzma"
	"github.com/tamnd/goipy/object"
)

// Python `lzma` stdlib module.
//
// Formats:
//   FORMAT_AUTO  (0) — decompressor auto-detect between XZ and ALONE.
//   FORMAT_XZ    (1) — default. Full .xz container.
//   FORMAT_ALONE (2) — legacy .lzma (13-byte header + raw LZMA).
//   FORMAT_RAW   (3) — raw LZMA stream, filters must be provided.
//
// The underlying lib/lzma uses a literal-only LZMA encoder so output is
// a valid XZ / .lzma stream but without meaningful compression. Any
// conforming LZMA decoder accepts our output.

const (
	lzmaFormatAuto  = 0
	lzmaFormatXZ    = 1
	lzmaFormatAlone = 2
	lzmaFormatRaw   = 3

	lzmaCheckNone    = 0
	lzmaCheckCRC32   = 1
	lzmaCheckCRC64   = 4
	lzmaCheckSHA256  = 10
	lzmaCheckIDMax   = 15
	lzmaCheckUnknown = 16

	lzmaPresetDefault = 6
	lzmaPresetExtreme = 0x80000000
)

func (i *Interp) buildLzma() *object.Module {
	m := &object.Module{Name: "lzma", Dict: object.NewDict()}

	// LZMAError — subclass of OSError per CPython.
	lzmaErrCls := &object.Class{Name: "LZMAError", Bases: []*object.Class{i.osErr}, Dict: object.NewDict()}
	m.Dict.SetStr("LZMAError", lzmaErrCls)

	// Constants.
	for name, v := range map[string]int64{
		"FORMAT_AUTO": lzmaFormatAuto, "FORMAT_XZ": lzmaFormatXZ,
		"FORMAT_ALONE": lzmaFormatAlone, "FORMAT_RAW": lzmaFormatRaw,
		"CHECK_NONE": lzmaCheckNone, "CHECK_CRC32": lzmaCheckCRC32,
		"CHECK_CRC64": lzmaCheckCRC64, "CHECK_SHA256": lzmaCheckSHA256,
		"CHECK_ID_MAX": lzmaCheckIDMax, "CHECK_UNKNOWN": lzmaCheckUnknown,
		"PRESET_DEFAULT":  lzmaPresetDefault,
		"PRESET_EXTREME":  lzmaPresetExtreme,
		"FILTER_LZMA1":    0x4000000000000001,
		"FILTER_LZMA2":    0x21,
		"FILTER_DELTA":    0x03,
		"FILTER_X86":      0x04,
		"FILTER_POWERPC":  0x05,
		"FILTER_IA64":     0x06,
		"FILTER_ARM":      0x07,
		"FILTER_ARMTHUMB": 0x08,
		"FILTER_SPARC":    0x09,
		"FILTER_ARM64":    0x0A,
		"MF_HC3":          0x03, "MF_HC4": 0x04, "MF_BT2": 0x12, "MF_BT3": 0x13, "MF_BT4": 0x14,
		"MODE_FAST":   1,
		"MODE_NORMAL": 2,
	} {
		m.Dict.SetStr(name, object.NewInt(v))
	}

	raiseLZMAError := func(msg string) error {
		exc := object.NewException(lzmaErrCls, msg)
		return exc
	}

	resolveCheck := func(format int, checkArg int64) lzma.CheckID {
		if checkArg == -1 {
			if format == lzmaFormatXZ {
				return lzma.CheckCRC64
			}
			return lzma.CheckNone
		}
		return lzma.CheckID(checkArg)
	}

	// compress(data, format=FORMAT_XZ, check=-1, preset=None, filters=None)
	m.Dict.SetStr("compress", &object.BuiltinFunc{Name: "compress", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "compress() missing data")
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		format := lzmaFormatXZ
		checkArg := int64(-1)
		if len(args) >= 2 {
			if n, ok := toInt64(args[1]); ok {
				format = int(n)
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("format"); ok {
				if n, ok2 := toInt64(v); ok2 {
					format = int(n)
				}
			}
		}
		if len(args) >= 3 {
			if n, ok := toInt64(args[2]); ok {
				checkArg = n
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("check"); ok {
				if n, ok2 := toInt64(v); ok2 {
					checkArg = n
				}
			}
		}
		switch format {
		case lzmaFormatXZ:
			return &object.Bytes{V: lzma.EncodeXZ(data, resolveCheck(format, checkArg))}, nil
		case lzmaFormatAlone:
			return &object.Bytes{V: lzma.EncodeAlone(data)}, nil
		case lzmaFormatRaw:
			return nil, raiseLZMAError("FORMAT_RAW requires a filter chain")
		default:
			return nil, raiseLZMAError("invalid format")
		}
	}})

	// decompress(data, format=FORMAT_AUTO, memlimit=None, filters=None)
	m.Dict.SetStr("decompress", &object.BuiltinFunc{Name: "decompress", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "decompress() missing data")
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		format := lzmaFormatAuto
		if len(args) >= 2 {
			if n, ok := toInt64(args[1]); ok {
				format = int(n)
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("format"); ok {
				if n, ok2 := toInt64(v); ok2 {
					format = int(n)
				}
			}
		}
		out, err := lzmaDecompressAll(data, format)
		if err != nil {
			return nil, raiseLZMAError(err.Error())
		}
		return &object.Bytes{V: out}, nil
	}})

	// is_check_supported(check_id)
	m.Dict.SetStr("is_check_supported", &object.BuiltinFunc{Name: "is_check_supported", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return object.False, nil
		}
		n, ok := toInt64(args[0])
		if !ok {
			return object.False, nil
		}
		switch n {
		case lzmaCheckNone, lzmaCheckCRC32, lzmaCheckCRC64, lzmaCheckSHA256:
			return object.True, nil
		}
		return object.False, nil
	}})

	m.Dict.SetStr("LZMACompressor", i.buildLZMACompressorClass(raiseLZMAError))
	m.Dict.SetStr("LZMADecompressor", i.buildLZMADecompressorClass(raiseLZMAError))

	lzmaFileCls := i.buildLZMAFileClass(raiseLZMAError)
	m.Dict.SetStr("LZMAFile", lzmaFileCls)

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
		format := lzmaFormatXZ
		if kwargs != nil {
			if v, ok2 := kwargs.GetStr("format"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					format = int(n)
				}
			}
		}
		checkArg := int64(-1)
		if kwargs != nil {
			if v, ok2 := kwargs.GetStr("check"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					checkArg = n
				}
			}
		}
		binaryMode := strings.TrimSuffix(mode, "t")
		if !strings.HasSuffix(binaryMode, "b") {
			binaryMode += "b"
		}
		isText := strings.HasSuffix(mode, "t")

		inst, err := i.newLZMAFileInstance(lzmaFileCls, filename.V, binaryMode, format, resolveCheck(format, checkArg))
		if err != nil {
			return nil, err
		}
		if !isText {
			return inst, nil
		}
		return i.wrapLZMAFileAsText(inst), nil
	}})

	return m
}

// lzmaDecompressAll handles one or more concatenated streams per `format`.
func lzmaDecompressAll(data []byte, format int) ([]byte, error) {
	switch format {
	case lzmaFormatAuto:
		if len(data) >= 6 && bytes.Equal(data[:6], []byte{0xFD, '7', 'z', 'X', 'Z', 0x00}) {
			out, _, err := lzma.DecodeXZ(data)
			return out, err
		}
		// Fallback: FORMAT_ALONE single stream.
		return lzma.DecodeAlone(data)
	case lzmaFormatXZ:
		out, _, err := lzma.DecodeXZ(data)
		return out, err
	case lzmaFormatAlone:
		return lzma.DecodeAlone(data)
	case lzmaFormatRaw:
		return nil, errLZMARawUnsup
	}
	return nil, errLZMABadFormat
}

var (
	errLZMARawUnsup  = &simpleErr{"FORMAT_RAW requires a filter chain"}
	errLZMABadFormat = &simpleErr{"invalid format"}
)

type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }

// ── LZMACompressor ────────────────────────────────────────────────────────

func (i *Interp) buildLZMACompressorClass(raiseLZMAError func(string) error) *object.Class {
	cls := &object.Class{Name: "LZMACompressor", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "LZMACompressor() missing self")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "LZMACompressor() self must be instance")
		}
		format := lzmaFormatXZ
		checkArg := int64(-1)
		if len(args) >= 2 {
			if n, ok2 := toInt64(args[1]); ok2 {
				format = int(n)
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("format"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					format = int(n)
				}
			}
		}
		if len(args) >= 3 {
			if n, ok2 := toInt64(args[2]); ok2 {
				checkArg = n
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("check"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					checkArg = n
				}
			}
		}
		return nil, i.initLZMACompressor(self, format, checkArg, raiseLZMAError)
	}})
	return cls
}

func (i *Interp) initLZMACompressor(inst *object.Instance, format int, checkArg int64, raiseLZMAError func(string) error) error {
	var buf []byte
	flushed := false
	d := inst.Dict

	check := lzma.CheckCRC64
	if checkArg == -1 {
		if format != lzmaFormatXZ {
			check = lzma.CheckNone
		}
	} else {
		check = lzma.CheckID(checkArg)
	}

	d.SetStr("compress", &object.BuiltinFunc{Name: "compress", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if flushed {
			return nil, raiseLZMAError("compressor already flushed")
		}
		if len(args) == 0 {
			return &object.Bytes{V: []byte{}}, nil
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		buf = append(buf, data...)
		return &object.Bytes{V: []byte{}}, nil
	}})

	d.SetStr("flush", &object.BuiltinFunc{Name: "flush", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		if flushed {
			return nil, raiseLZMAError("compressor already flushed")
		}
		flushed = true
		var out []byte
		switch format {
		case lzmaFormatXZ:
			out = lzma.EncodeXZ(buf, check)
		case lzmaFormatAlone:
			out = lzma.EncodeAlone(buf)
		default:
			return nil, raiseLZMAError("unsupported format")
		}
		buf = nil
		return &object.Bytes{V: out}, nil
	}})

	return nil
}

// ── LZMADecompressor ──────────────────────────────────────────────────────

func (i *Interp) buildLZMADecompressorClass(raiseLZMAError func(string) error) *object.Class {
	cls := &object.Class{Name: "LZMADecompressor", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "LZMADecompressor() missing self")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "LZMADecompressor() self must be instance")
		}
		format := lzmaFormatAuto
		if len(args) >= 2 {
			if n, ok2 := toInt64(args[1]); ok2 {
				format = int(n)
			}
		} else if kwargs != nil {
			if v, ok2 := kwargs.GetStr("format"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					format = int(n)
				}
			}
		}
		return nil, i.initLZMADecompressor(self, format, raiseLZMAError)
	}})
	return cls
}

func (i *Interp) initLZMADecompressor(inst *object.Instance, format int, raiseLZMAError func(string) error) error {
	d := inst.Dict
	var accumulated []byte
	var outBuf []byte
	eof := false

	d.SetStr("eof", object.False)
	d.SetStr("unused_data", &object.Bytes{V: []byte{}})
	d.SetStr("needs_input", object.True)
	d.SetStr("check", object.NewInt(lzmaCheckUnknown))

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
		if !eof && len(accumulated) > 0 {
			isXZ := format == lzmaFormatXZ || (format == lzmaFormatAuto && len(accumulated) >= 6 && bytes.Equal(accumulated[:6], []byte{0xFD, '7', 'z', 'X', 'Z', 0x00}))
			if isXZ {
				// Try to decode just ONE stream, then expose trailing
				// bytes as unused_data. DecodeXZ greedily consumes all
				// concatenated streams; we only want the first.
				out, consumed, err := lzma.DecodeOneXZStream(accumulated)
				switch {
				case err == nil:
					outBuf = append(outBuf, out...)
					eof = true
					d.SetStr("eof", object.True)
					d.SetStr("check", object.NewInt(lzmaCheckCRC64))
					if consumed < len(accumulated) {
						unused := append([]byte{}, accumulated[consumed:]...)
						d.SetStr("unused_data", &object.Bytes{V: unused})
					}
				case lzma.IsShortInputErr(err):
					// Need more bytes — not a fatal error.
				default:
					return nil, raiseLZMAError(err.Error())
				}
			} else {
				out, err := lzma.DecodeAlone(accumulated)
				switch {
				case err == nil:
					outBuf = append(outBuf, out...)
					eof = true
					d.SetStr("eof", object.True)
					d.SetStr("check", object.NewInt(lzmaCheckNone))
				case lzma.IsShortInputErr(err):
					// wait for more
				default:
					return nil, raiseLZMAError(err.Error())
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

// ── LZMAFile ──────────────────────────────────────────────────────────────

func (i *Interp) buildLZMAFileClass(raiseLZMAError func(string) error) *object.Class {
	cls := &object.Class{Name: "LZMAFile", Dict: object.NewDict()}
	cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "LZMAFile() missing self")
		}
		self, ok := args[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "LZMAFile() self must be instance")
		}
		filename := ""
		mode := "rb"
		format := lzmaFormatXZ
		check := lzma.CheckCRC64
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
			if v, ok2 := kwargs.GetStr("format"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					format = int(n)
				}
			}
			if v, ok2 := kwargs.GetStr("check"); ok2 {
				if n, ok3 := toInt64(v); ok3 {
					if n == -1 {
						if format != lzmaFormatXZ {
							check = lzma.CheckNone
						}
					} else {
						check = lzma.CheckID(n)
					}
				}
			}
		}
		return nil, i.initLZMAFileInstance(self, filename, mode, format, check, raiseLZMAError)
	}})
	return cls
}

func (i *Interp) newLZMAFileInstance(cls *object.Class, filename, mode string, format int, check lzma.CheckID) (*object.Instance, error) {
	d := object.NewDict()
	inst := &object.Instance{Class: cls, Dict: d}
	if err := i.initLZMAFileInstance(inst, filename, mode, format, check, func(msg string) error {
		return object.Errorf(i.osErr, "%s", msg)
	}); err != nil {
		return nil, err
	}
	return inst, nil
}

func (i *Interp) initLZMAFileInstance(inst *object.Instance, filename, mode string, format int, check lzma.CheckID, raiseLZMAError func(string) error) error {
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
		var writeBuf []byte

		d.SetStr("mode", &object.Str{V: "wb"})
		d.SetStr("name", &object.Str{V: filename})
		d.SetStr("closed", object.False)

		checkOpen := func() error {
			if closed {
				return raiseLZMAError("write to closed LZMAFile")
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
			var encoded []byte
			switch format {
			case lzmaFormatAlone:
				encoded = lzma.EncodeAlone(writeBuf)
			default:
				encoded = lzma.EncodeXZ(writeBuf, check)
			}
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
			return nil, object.Errorf(i.osErr, "read on write-mode LZMAFile")
		}}
		d.SetStr("read", readErr)
		d.SetStr("read1", readErr)
		d.SetStr("readline", readErr)
		d.SetStr("readlines", readErr)
		d.SetStr("readinto", readErr)
		d.SetStr("peek", readErr)
		d.SetStr("seek", &object.BuiltinFunc{Name: "seek", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.osErr, "seek on write-mode LZMAFile")
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
		decompressed, err := lzmaDecompressAll(raw, format)
		if err != nil {
			return raiseLZMAError(err.Error())
		}

		d.SetStr("mode", &object.Str{V: "rb"})
		d.SetStr("name", &object.Str{V: filename})
		d.SetStr("closed", object.False)

		checkOpen := func() error {
			if closed {
				return raiseLZMAError("read from closed LZMAFile")
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
			return nil, object.Errorf(i.osErr, "write on read-mode LZMAFile")
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
		return nil, object.Errorf(i.osErr, "LZMAFile does not support fileno()")
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

// wrapLZMAFileAsText overlays UTF-8 encode/decode on a binary LZMAFile.
func (i *Interp) wrapLZMAFileAsText(binInst *object.Instance) *object.Instance {
	cls := &object.Class{Name: "LZMAFile", Dict: object.NewDict()}
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

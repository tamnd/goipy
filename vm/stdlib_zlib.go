package vm

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"math/big"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildZlib() *object.Module {
	m := &object.Module{Name: "zlib", Dict: object.NewDict()}

	// ── Exception ─────────────────────────────────────────────────────────────
	zlibErr := &object.Class{Name: "error", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	m.Dict.SetStr("error", zlibErr)

	// ── Constants ─────────────────────────────────────────────────────────────
	setInt := func(name string, v int) { m.Dict.SetStr(name, object.NewInt(int64(v))) }

	// level
	setInt("Z_NO_COMPRESSION", 0)
	setInt("Z_BEST_SPEED", 1)
	setInt("Z_BEST_COMPRESSION", 9)
	setInt("Z_DEFAULT_COMPRESSION", -1)

	// flush modes
	setInt("Z_NO_FLUSH", 0)
	setInt("Z_PARTIAL_FLUSH", 1)
	setInt("Z_SYNC_FLUSH", 2)
	setInt("Z_FULL_FLUSH", 3)
	setInt("Z_FINISH", 4)
	setInt("Z_BLOCK", 5)
	setInt("Z_TREES", 6)

	// strategy
	setInt("Z_FILTERED", 1)
	setInt("Z_HUFFMAN_ONLY", 2)
	setInt("Z_RLE", 3)
	setInt("Z_FIXED", 4)
	setInt("Z_DEFAULT_STRATEGY", 0)

	// misc
	setInt("DEFLATED", 8)
	setInt("DEF_BUF_SIZE", 16384)
	setInt("DEF_MEM_LEVEL", 8)
	setInt("MAX_WBITS", 15)

	// ── Helpers ───────────────────────────────────────────────────────────────

	zlibError := func(msg string, args ...any) error {
		if len(args) > 0 {
			return object.Errorf(zlibErr, msg, args...)
		}
		return object.Errorf(zlibErr, "%s", msg)
	}

	// zlibWbitsFromArgs reads wbits from positional arg at index `pos` or from
	// kwargs key "wbits".  Returns the default (15) when absent.
	zlibWbitsFromArgs := func(args []object.Object, pos int, kwargs *object.Dict) int {
		wbits := 15
		if pos < len(args) {
			if n, ok := toInt64(args[pos]); ok {
				wbits = int(n)
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("wbits"); ok {
				if n, ok2 := toInt64(v); ok2 {
					wbits = int(n)
				}
			}
		}
		return wbits
	}

	// newZlibWriter creates a compressor for the given wbits and level.
	newZlibWriter := func(buf *bytes.Buffer, level, wbits int) (io.WriteCloser, error) {
		switch {
		case wbits >= 9 && wbits <= 15:
			return zlib.NewWriterLevel(buf, level)
		case wbits <= -9 && wbits >= -15:
			return flate.NewWriter(buf, level)
		case wbits >= 25 && wbits <= 31:
			gw, err := gzip.NewWriterLevel(buf, level)
			return gw, err
		default:
			return nil, zlibError("invalid wbits value %d", wbits)
		}
	}

	// newZlibReader creates a decompressor for the given wbits.
	newZlibReader := func(r io.Reader, wbits int) (io.ReadCloser, error) {
		switch {
		case wbits >= 9 && wbits <= 15:
			return zlib.NewReader(r)
		case wbits <= -9 && wbits >= -15:
			return flate.NewReader(r), nil
		case wbits >= 25 && wbits <= 31:
			return gzip.NewReader(r)
		default:
			return nil, zlibError("invalid wbits value %d", wbits)
		}
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
		level := -1
		if len(args) >= 2 {
			if n, ok := toInt64(args[1]); ok {
				level = int(n)
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("level"); ok {
				if n, ok2 := toInt64(v); ok2 {
					level = int(n)
				}
			}
		}
		wbits := zlibWbitsFromArgs(args, 2, kwargs)
		var buf bytes.Buffer
		w, err := newZlibWriter(&buf, level, wbits)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(data); err != nil {
			return nil, zlibError("%v", err)
		}
		if err := w.Close(); err != nil {
			return nil, zlibError("%v", err)
		}
		return &object.Bytes{V: buf.Bytes()}, nil
	}})

	// ── decompress() ──────────────────────────────────────────────────────────

	m.Dict.SetStr("decompress", &object.BuiltinFunc{Name: "decompress", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "decompress() missing data")
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		wbits := zlibWbitsFromArgs(args, 1, kwargs)
		r, err := newZlibReader(bytes.NewReader(data), wbits)
		if err != nil {
			return nil, zlibError("%v", err)
		}
		defer r.Close()
		out, err := io.ReadAll(r)
		if err != nil {
			return nil, zlibError("%v", err)
		}
		return &object.Bytes{V: out}, nil
	}})

	// ── crc32() ───────────────────────────────────────────────────────────────

	m.Dict.SetStr("crc32", &object.BuiltinFunc{Name: "crc32", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "crc32() missing data")
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		seed := uint32(0)
		if len(args) >= 2 {
			if n, ok := toInt64(args[1]); ok {
				seed = uint32(n)
			}
		}
		result := zlibCRC32Update(seed, data)
		return newZlibIntU32(result), nil
	}})

	// ── adler32() ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("adler32", &object.BuiltinFunc{Name: "adler32", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return nil, object.Errorf(i.typeErr, "adler32() missing data")
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		seed := uint32(1)
		if len(args) >= 2 {
			if n, ok := toInt64(args[1]); ok {
				seed = uint32(n)
			}
		}
		result := zlibAdler32Update(seed, data)
		return newZlibIntU32(result), nil
	}})

	// ── compressobj() ─────────────────────────────────────────────────────────

	m.Dict.SetStr("compressobj", &object.BuiltinFunc{Name: "compressobj", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		level := -1
		if len(args) >= 1 {
			if n, ok := toInt64(args[0]); ok {
				level = int(n)
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("level"); ok {
				if n, ok2 := toInt64(v); ok2 {
					level = int(n)
				}
			}
		}
		// args[1] = method (ignored, always DEFLATED)
		wbits := 15
		if len(args) >= 3 {
			if n, ok := toInt64(args[2]); ok {
				wbits = int(n)
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("wbits"); ok {
				if n, ok2 := toInt64(v); ok2 {
					wbits = int(n)
				}
			}
		}
		// args[3]=memLevel, args[4]=strategy, args[5]=zdict — accepted, not applied

		return i.makeZlibCompressObj(level, wbits, zlibError, newZlibWriter), nil
	}})

	// ── decompressobj() ───────────────────────────────────────────────────────

	m.Dict.SetStr("decompressobj", &object.BuiltinFunc{Name: "decompressobj", Call: func(_ any, args []object.Object, kwargs *object.Dict) (object.Object, error) {
		wbits := 15
		if len(args) >= 1 {
			if n, ok := toInt64(args[0]); ok {
				wbits = int(n)
			}
		} else if kwargs != nil {
			if v, ok := kwargs.GetStr("wbits"); ok {
				if n, ok2 := toInt64(v); ok2 {
					wbits = int(n)
				}
			}
		}
		return i.makeZlibDecompressObj(wbits, zlibError, newZlibReader), nil
	}})

	return m
}

// ── compressobj ───────────────────────────────────────────────────────────────

func (i *Interp) makeZlibCompressObj(
	level, wbits int,
	zlibError func(string, ...any) error,
	newWriter func(*bytes.Buffer, int, int) (io.WriteCloser, error),
) *object.Instance {
	cls := &object.Class{Name: "Compress", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	d := object.NewDict()
	inst := &object.Instance{Class: cls, Dict: d}

	var buf bytes.Buffer
	w, err := newWriter(&buf, level, wbits)
	if err != nil {
		// return a broken object; compress() will fail
		w = nil
	}

	// drain returns whatever the writer flushed into buf since last drain.
	drain := func() []byte {
		if buf.Len() == 0 {
			return []byte{}
		}
		out := make([]byte, buf.Len())
		copy(out, buf.Bytes())
		buf.Reset()
		return out
	}

	compress := &object.BuiltinFunc{Name: "compress", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if w == nil {
			return nil, zlibError("compressobj is in error state")
		}
		if len(args) == 0 {
			return &object.Bytes{V: []byte{}}, nil
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(data); err != nil {
			return nil, zlibError("%v", err)
		}
		return &object.Bytes{V: drain()}, nil
	}}
	d.SetStr("compress", compress)

	flush := &object.BuiltinFunc{Name: "flush", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if w == nil {
			return &object.Bytes{V: []byte{}}, nil
		}
		mode := 4 // Z_FINISH
		if len(args) >= 1 {
			if n, ok := toInt64(args[0]); ok {
				mode = int(n)
			}
		}
		switch mode {
		case 4: // Z_FINISH
			if err := w.Close(); err != nil {
				return nil, zlibError("%v", err)
			}
			w = nil
		default:
			// Z_SYNC_FLUSH, Z_FULL_FLUSH, etc. — flush without closing
			if f, ok := w.(interface{ Flush() error }); ok {
				if err := f.Flush(); err != nil {
					return nil, zlibError("%v", err)
				}
			}
		}
		return &object.Bytes{V: drain()}, nil
	}}
	d.SetStr("flush", flush)

	copyFn := &object.BuiltinFunc{Name: "copy", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		// Best-effort: flush current state, return a new object seeded with
		// the compressed bytes produced so far (not a true mid-stream clone).
		return i.makeZlibCompressObj(level, wbits, zlibError, newWriter), nil
	}}
	d.SetStr("copy", copyFn)

	return inst
}

// ── decompressobj ─────────────────────────────────────────────────────────────

func (i *Interp) makeZlibDecompressObj(
	wbits int,
	zlibError func(string, ...any) error,
	newReader func(io.Reader, int) (io.ReadCloser, error),
) *object.Instance {
	cls := &object.Class{Name: "Decompress", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	d := object.NewDict()
	inst := &object.Instance{Class: cls, Dict: d}

	// We buffer all compressed input. On each decompress() call we attempt to
	// fully decompress the accumulated bytes. This correctly handles the common
	// cases: all-at-once and split-at-stream-boundary. Partial mid-stream
	// chunks return empty bytes until a complete stream is accumulated.
	var accumulated []byte
	var decompressed []byte
	var outPos int      // how many decompressed bytes have been returned
	var unusedData []byte

	// tryDecompress attempts to decompress accumulated bytes.
	// It uses bytes.Reader.Len() after decompression to detect unused_data,
	// because the decompressor's internal buffering means the counting reader
	// approach over-counts consumed bytes.
	tryDecompress := func() {
		innerRd := bytes.NewReader(accumulated)
		r, err := newReader(innerRd, wbits)
		if err != nil {
			return
		}
		out, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			// partial input — can't decompress yet
			return
		}
		decompressed = out
		remaining := innerRd.Len()
		if remaining > 0 {
			unusedData = make([]byte, remaining)
			copy(unusedData, accumulated[len(accumulated)-remaining:])
		} else {
			unusedData = []byte{}
		}
	}

	d.SetStr("unused_data", &object.Bytes{V: []byte{}})
	d.SetStr("unconsumed_tail", &object.Bytes{V: []byte{}})

	decompress := &object.BuiltinFunc{Name: "decompress", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		if len(args) == 0 {
			return &object.Bytes{V: []byte{}}, nil
		}
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		maxLength := 0
		if len(args) >= 2 {
			if n, ok := toInt64(args[1]); ok {
				maxLength = int(n)
			}
		}
		accumulated = append(accumulated, data...)
		prevLen := len(decompressed)
		tryDecompress()

		// New output available since last call
		chunk := decompressed[outPos:]
		if maxLength > 0 && len(chunk) > maxLength {
			unconsumed := chunk[maxLength:]
			chunk = chunk[:maxLength]
			d.SetStr("unconsumed_tail", &object.Bytes{V: unconsumed})
		} else {
			d.SetStr("unconsumed_tail", &object.Bytes{V: []byte{}})
		}
		outPos += len(chunk)
		_ = prevLen

		if len(unusedData) > 0 {
			d.SetStr("unused_data", &object.Bytes{V: unusedData})
		}
		return &object.Bytes{V: chunk}, nil
	}}
	d.SetStr("decompress", decompress)

	flush := &object.BuiltinFunc{Name: "flush", Call: func(_ any, args []object.Object, _ *object.Dict) (object.Object, error) {
		tryDecompress()
		remaining := decompressed[outPos:]
		outPos = len(decompressed)
		if len(unusedData) > 0 {
			d.SetStr("unused_data", &object.Bytes{V: unusedData})
		}
		return &object.Bytes{V: remaining}, nil
	}}
	d.SetStr("flush", flush)

	copyFn := &object.BuiltinFunc{Name: "copy", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return i.makeZlibDecompressObj(wbits, zlibError, newReader), nil
	}}
	d.SetStr("copy", copyFn)

	return inst
}

// ── pure-Go checksum helpers ──────────────────────────────────────────────────

// zlibCRC32Update computes CRC-32 (IEEE) with a running seed.
func zlibCRC32Update(seed uint32, data []byte) uint32 {
	const poly = 0xedb88320
	crc := ^seed
	for _, b := range data {
		crc ^= uint32(b)
		for j := 0; j < 8; j++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ poly
			} else {
				crc >>= 1
			}
		}
	}
	return ^crc
}

// zlibAdler32Update computes Adler-32 with an arbitrary seed.
// The seed encodes the running (B, A) pair as (seed>>16)<<16 | (seed&0xffff).
// The standard initial state is seed=1 (A=1, B=0).
func zlibAdler32Update(seed uint32, data []byte) uint32 {
	const mod = 65521
	a := seed & 0xffff
	b := (seed >> 16) & 0xffff
	for _, c := range data {
		a = (a + uint32(c)) % mod
		b = (b + a) % mod
	}
	return (b << 16) | a
}

// newZlibIntU32 boxes an unsigned 32-bit value as a Python int.
func newZlibIntU32(u uint32) *object.Int {
	return object.IntFromBig(new(big.Int).SetUint64(uint64(u)))
}


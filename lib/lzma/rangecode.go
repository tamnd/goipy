package lzma

import "io"

// LZMA range coder. Follows Igor Pavlov's reference LZMA SDK.
//
// Encoder maintains (low uint64, rng uint32). Normalise while
// rng < topValue = 1<<24 by shifting out high bytes. The carry path
// uses a cache byte plus a cache-size counter so that an eventual
// +1 ripple into already-produced output can be flushed at close
// time.

const (
	topValue = uint32(1) << 24
	numBits  = 11
	bitModel = uint32(1) << numBits // 2048
	moveBits = 5
)

type rangeEncoder struct {
	w         io.Writer
	low       uint64
	rng       uint32
	cache     byte
	cacheSize uint64
	out       []byte
	err       error
}

func newRangeEncoder(w io.Writer) *rangeEncoder {
	return &rangeEncoder{w: w, rng: 0xFFFFFFFF, cacheSize: 1}
}

func (e *rangeEncoder) shiftLow() {
	// If low's top byte is stable (no carry possible), flush cached
	// byte + any pending 0xFF fillers. If carry is set (bit 32), ripple
	// +1 through the cached byte and fillers.
	if e.low < 0xFF000000 || e.low >= 0x100000000 {
		carry := byte(e.low >> 32)
		out := e.cache + carry
		e.emit(out)
		for i := uint64(0); i < e.cacheSize-1; i++ {
			e.emit(0xFF + carry)
		}
		e.cache = byte((e.low >> 24) & 0xFF)
		e.cacheSize = 1
	} else {
		e.cacheSize++
	}
	e.low = (e.low << 8) & 0xFFFFFFFF
}

func (e *rangeEncoder) emit(b byte) {
	if e.err != nil {
		return
	}
	e.out = append(e.out, b)
}

func (e *rangeEncoder) normalize() {
	if e.rng < topValue {
		e.rng <<= 8
		e.shiftLow()
	}
}

// encodeBit encodes one context-sensitive bit, updating the probability.
func (e *rangeEncoder) encodeBit(prob *uint16, bit uint32) {
	p := uint32(*prob)
	bound := (e.rng >> numBits) * p
	if bit == 0 {
		e.rng = bound
		*prob = uint16(p + ((bitModel - p) >> moveBits))
	} else {
		e.low += uint64(bound)
		e.rng -= bound
		*prob = uint16(p - (p >> moveBits))
	}
	e.normalize()
}

// encodeDirectBits writes count raw bits, MSB first, without probability.
func (e *rangeEncoder) encodeDirectBits(v uint32, count int) {
	for i := count - 1; i >= 0; i-- {
		e.rng >>= 1
		e.low += uint64(e.rng) & uint64(-int64((v>>uint(i))&1))
		e.normalize()
	}
}

func (e *rangeEncoder) flush() error {
	for i := 0; i < 5; i++ {
		e.shiftLow()
	}
	if e.err != nil {
		return e.err
	}
	_, err := e.w.Write(e.out)
	e.out = nil
	return err
}

// ── decoder ───────────────────────────────────────────────────────────────

type rangeDecoder struct {
	src  []byte
	pos  int
	code uint32
	rng  uint32
}

func newRangeDecoder(src []byte) (*rangeDecoder, error) {
	if len(src) < 5 {
		return nil, errShort
	}
	if src[0] != 0 {
		return nil, errCorrupt
	}
	d := &rangeDecoder{src: src, pos: 5, rng: 0xFFFFFFFF}
	d.code = uint32(src[1])<<24 | uint32(src[2])<<16 | uint32(src[3])<<8 | uint32(src[4])
	return d, nil
}

func (d *rangeDecoder) normalize() error {
	if d.rng < topValue {
		if d.pos >= len(d.src) {
			return errShort
		}
		d.rng <<= 8
		d.code = (d.code << 8) | uint32(d.src[d.pos])
		d.pos++
	}
	return nil
}

func (d *rangeDecoder) decodeBit(prob *uint16) (uint32, error) {
	p := uint32(*prob)
	bound := (d.rng >> numBits) * p
	var bit uint32
	if d.code < bound {
		d.rng = bound
		*prob = uint16(p + ((bitModel - p) >> moveBits))
		bit = 0
	} else {
		d.code -= bound
		d.rng -= bound
		*prob = uint16(p - (p >> moveBits))
		bit = 1
	}
	if err := d.normalize(); err != nil {
		return 0, err
	}
	return bit, nil
}

func (d *rangeDecoder) decodeDirectBits(count int) (uint32, error) {
	var v uint32
	for i := 0; i < count; i++ {
		d.rng >>= 1
		d.code -= d.rng
		// If code went "negative" (top bit set in int32 view), the bit is 0.
		mask := uint32(0) - (d.code >> 31)
		d.code += d.rng & mask
		v = (v << 1) | (1 - (mask & 1))
		if err := d.normalize(); err != nil {
			return 0, err
		}
	}
	return v, nil
}

// finished returns true if the decoder has consumed enough bytes and
// the code buffer is drained. Not used for explicit EOS — LZMA end-of-
// stream is signalled via the MATCH escape distance instead.
func (d *rangeDecoder) finished() bool { return d.code == 0 }

package bzip2

// bitWriter buffers bits MSB-first into a byte slice.
type bitWriter struct {
	buf     []byte
	current uint64
	nBits   uint
}

func (w *bitWriter) writeBits(value uint64, nBits uint) {
	w.current = (w.current << nBits) | (value & ((1 << nBits) - 1))
	w.nBits += nBits
	for w.nBits >= 8 {
		w.nBits -= 8
		w.buf = append(w.buf, byte(w.current>>w.nBits))
		w.current &= (1 << w.nBits) - 1
	}
}

// flushByte pads the current partial byte with zero bits and emits it.
func (w *bitWriter) flushByte() {
	if w.nBits > 0 {
		w.buf = append(w.buf, byte(w.current<<(8-w.nBits)))
		w.current = 0
		w.nBits = 0
	}
}

func (w *bitWriter) bytes() []byte { return w.buf }

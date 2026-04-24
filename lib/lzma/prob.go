package lzma

// Probability tree helpers. LZMA codes multi-bit values (alphabet size
// up to 256 for literals) by walking a binary tree whose internal nodes
// each hold one 11-bit probability. For an N-bit value the tree has
// 2^N - 1 nodes stored in index [1 .. 2^N-1] (index 0 unused).

type bitTree struct {
	probs []uint16
	bits  int
}

func newBitTree(bits int) *bitTree {
	n := 1 << uint(bits)
	t := &bitTree{probs: make([]uint16, n), bits: bits}
	for i := range t.probs {
		t.probs[i] = uint16(bitModel / 2)
	}
	return t
}

func (t *bitTree) reset() {
	for i := range t.probs {
		t.probs[i] = uint16(bitModel / 2)
	}
}

func (t *bitTree) encode(e *rangeEncoder, v uint32) {
	idx := uint32(1)
	for i := t.bits - 1; i >= 0; i-- {
		bit := (v >> uint(i)) & 1
		e.encodeBit(&t.probs[idx], bit)
		idx = (idx << 1) | bit
	}
}

func (t *bitTree) decode(d *rangeDecoder) (uint32, error) {
	idx := uint32(1)
	for i := 0; i < t.bits; i++ {
		b, err := d.decodeBit(&t.probs[idx])
		if err != nil {
			return 0, err
		}
		idx = (idx << 1) | b
	}
	return idx - (1 << uint(t.bits)), nil
}

// encodeReverse / decodeReverse walk the tree LSB first (used for the
// "aligned bits" sub-coder on large distances).
func (t *bitTree) encodeReverse(e *rangeEncoder, v uint32) {
	idx := uint32(1)
	for i := 0; i < t.bits; i++ {
		bit := (v >> uint(i)) & 1
		e.encodeBit(&t.probs[idx], bit)
		idx = (idx << 1) | bit
	}
}

func (t *bitTree) decodeReverse(d *rangeDecoder) (uint32, error) {
	idx := uint32(1)
	var v uint32
	for i := 0; i < t.bits; i++ {
		b, err := d.decodeBit(&t.probs[idx])
		if err != nil {
			return 0, err
		}
		idx = (idx << 1) | b
		v |= b << uint(i)
	}
	return v, nil
}

// encodeReverseFixed is used by the match-finder's "direct bits" path
// for distance slots that include a reversed sub-coder of a fixed set
// of probabilities rooted elsewhere.
func encodeReverseFixed(e *rangeEncoder, probs []uint16, v uint32, bits int) {
	idx := uint32(1)
	for i := 0; i < bits; i++ {
		bit := (v >> uint(i)) & 1
		e.encodeBit(&probs[idx-1], bit)
		idx = (idx << 1) | bit
	}
}

func decodeReverseFixed(d *rangeDecoder, probs []uint16, bits int) (uint32, error) {
	idx := uint32(1)
	var v uint32
	for i := 0; i < bits; i++ {
		b, err := d.decodeBit(&probs[idx-1])
		if err != nil {
			return 0, err
		}
		idx = (idx << 1) | b
		v |= b << uint(i)
	}
	return v, nil
}

func fillProbs(p []uint16) {
	for i := range p {
		p[i] = uint16(bitModel / 2)
	}
}

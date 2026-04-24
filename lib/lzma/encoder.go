package lzma

import "bytes"

// Literal-only LZMA encoder. Emits a valid raw LZMA range-coded stream
// where every input byte becomes one LIT packet (IsMatch bit = 0, then
// an 8-bit context-tree literal). No MATCH, REP, or SHORTREP packets.
// No end-of-stream marker — the surrounding container (LZMA2 chunk
// header or .lzma 13-byte header) carries the exact uncompressed size.
//
// State tracking: because we only emit literals starting from state 0,
// the state machine stays at state 0 forever (LIT transition from any
// of {0,1,2,3} → 0). The "previous packet was a match" delta-literal
// branch is therefore unreachable — simple literal coding suffices.

// encodeLZMA range-codes src under props and returns the produced bytes.
func encodeLZMA(src []byte, props Properties) []byte {
	var buf bytes.Buffer
	enc := newRangeEncoder(&buf)

	// isMatch[state][posState] — prob of "packet starts with match".
	// We always encode 0 (literal) but the probability still updates,
	// matching what the decoder will see.
	var isMatch [numStates][1 << 4]uint16
	for i := range isMatch {
		fillProbs(isMatch[i][:])
	}

	// Literal coder probs: one 0x300-entry table per literal context.
	// Layout: probs[(ctx * 0x300) + treeIdx].
	numCtx := 1 << uint(props.LC+props.LP)
	litProbs := make([]uint16, numCtx*0x300)
	fillProbs(litProbs)

	posMask := props.posMask()
	lpMask := props.lpMask()
	lcShift := uint(8 - props.LC)

	var prev byte
	state := 0
	_ = state // only ever 0 — retained for future extension.

	for i, b := range src {
		posState := uint32(i) & posMask
		// IsMatch bit = 0 (literal).
		enc.encodeBit(&isMatch[0][posState], 0)

		// Literal context index.
		ctx := ((uint32(i) & lpMask) << uint(props.LC)) | (uint32(prev) >> lcShift)
		probs := litProbs[int(ctx)*0x300 : int(ctx)*0x300+0x300]

		// Straight literal path (no previous match → no delta-literal).
		sym := uint32(1)
		for bit := 7; bit >= 0; bit-- {
			v := uint32(b>>uint(bit)) & 1
			enc.encodeBit(&probs[sym], v)
			sym = (sym << 1) | v
		}

		prev = b
	}

	_ = enc.flush()
	return buf.Bytes()
}

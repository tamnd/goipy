package lzma

// decodeLZMA reverses encodeLZMA: range-decodes `size` literal packets
// from src under props. Returns the uncompressed bytes and the number
// of source bytes consumed. If the stream contains any non-literal
// packet (IsMatch bit = 1), returns errUnsup — matches are outside the
// scope of our literal-only encoder pairing.

func decodeLZMA(src []byte, props Properties, size int64) ([]byte, int, error) {
	dec, err := newRangeDecoder(src)
	if err != nil {
		return nil, 0, err
	}

	var isMatch [numStates][1 << 4]uint16
	for i := range isMatch {
		fillProbs(isMatch[i][:])
	}

	numCtx := 1 << uint(props.LC+props.LP)
	litProbs := make([]uint16, numCtx*0x300)
	fillProbs(litProbs)

	posMask := props.posMask()
	lpMask := props.lpMask()
	lcShift := uint(8 - props.LC)

	out := make([]byte, 0, size)
	var prev byte

	for int64(len(out)) < size {
		i := uint32(len(out))
		posState := i & posMask
		bit, err := dec.decodeBit(&isMatch[0][posState])
		if err != nil {
			return nil, 0, err
		}
		if bit != 0 {
			return nil, 0, errUnsup
		}

		ctx := ((i & lpMask) << uint(props.LC)) | (uint32(prev) >> lcShift)
		probs := litProbs[int(ctx)*0x300 : int(ctx)*0x300+0x300]

		sym := uint32(1)
		for sym < 0x100 {
			b, err := dec.decodeBit(&probs[sym])
			if err != nil {
				return nil, 0, err
			}
			sym = (sym << 1) | b
		}
		out = append(out, byte(sym&0xFF))
		prev = byte(sym & 0xFF)
	}

	return out, dec.pos, nil
}

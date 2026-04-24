package lzma

// LZMA2 framing. Our encoder emits:
//   - first chunk:   control 0xE0 (state + props + dict reset)
//   - subsequent:    control 0xC0 (state + props reset, no dict reset)
//   - terminator:    control 0x00
//
// Uncompressed chunks (0x01, 0x02) are decoded on input for robustness
// but never produced by our encoder.
//
// Per chunk the max uncompressed payload is 2 MiB (21 bits) and max
// compressed payload is 64 KiB (16 bits). To stay under the compressed
// cap with literal-only LZMA (expansion ≤ 1 byte + few bits per input
// byte), we split at 1<<15 = 32 KiB uncompressed.

const lzma2UncompressedChunk = 1 << 15

func encodeLZMA2(src []byte, props Properties) []byte {
	var out []byte
	first := true
	for i := 0; i < len(src); {
		end := i + lzma2UncompressedChunk
		if end > len(src) {
			end = len(src)
		}
		chunkU := src[i:end]
		chunkC := encodeLZMA(chunkU, props)

		uSize := len(chunkU) - 1 // encoded as size-1
		cSize := len(chunkC) - 1

		var control byte
		if first {
			control = 0xE0 // state + props + dict reset
		} else {
			control = 0xC0 // state + props reset
		}
		control |= byte(uSize >> 16) // high 5 bits of uncompressed size

		out = append(out,
			control,
			byte(uSize>>8), byte(uSize),
			byte(cSize>>8), byte(cSize),
			props.propByte(),
		)
		out = append(out, chunkC...)

		first = false
		i = end
	}
	out = append(out, 0x00) // end of LZMA2 stream
	return out
}

func decodeLZMA2(src []byte) ([]byte, int, error) {
	var out []byte
	props := defaultProperties()
	pos := 0
	propsSet := false
	for pos < len(src) {
		c := src[pos]
		pos++
		if c == 0x00 {
			return out, pos, nil
		}
		switch {
		case c == 0x01 || c == 0x02:
			if pos+2 > len(src) {
				return nil, 0, errShort
			}
			size := int(src[pos])<<8 | int(src[pos+1])
			size++
			pos += 2
			if pos+size > len(src) {
				return nil, 0, errShort
			}
			out = append(out, src[pos:pos+size]...)
			pos += size
			if c == 0x01 {
				propsSet = false // dict reset — keep decoder fresh
			}
		case c >= 0x80:
			if pos+4 > len(src) {
				return nil, 0, errShort
			}
			uSize := (int(c&0x1F) << 16) | (int(src[pos]) << 8) | int(src[pos+1])
			uSize++
			cSize := (int(src[pos+2]) << 8) | int(src[pos+3])
			cSize++
			pos += 4
			mode := c & 0xE0
			if mode == 0xC0 || mode == 0xE0 {
				if pos >= len(src) {
					return nil, 0, errShort
				}
				p, err := decodePropByte(src[pos])
				if err != nil {
					return nil, 0, err
				}
				props = p
				propsSet = true
				pos++
			} else if !propsSet {
				return nil, 0, errCorrupt
			}
			if pos+cSize > len(src) {
				return nil, 0, errShort
			}
			chunkC := src[pos : pos+cSize]
			pos += cSize
			chunkU, _, err := decodeLZMA(chunkC, props, int64(uSize))
			if err != nil {
				return nil, 0, err
			}
			out = append(out, chunkU...)
		default:
			return nil, 0, errCorrupt
		}
	}
	return nil, 0, errShort
}

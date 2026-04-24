package lzma

// XZ multibyte varint: little-endian base-128, up to 9 bytes, minimal
// form required (no trailing-zero continuation byte).

func encodeVarint(v uint64) []byte {
	if v == 0 {
		return []byte{0}
	}
	var buf [9]byte
	n := 0
	for v >= 0x80 {
		buf[n] = byte(v) | 0x80
		v >>= 7
		n++
		if n == 9 {
			break
		}
	}
	buf[n] = byte(v)
	n++
	return buf[:n]
}

func decodeVarint(data []byte) (uint64, int, error) {
	var v uint64
	for i := 0; i < 9 && i < len(data); i++ {
		b := data[i]
		if i == 8 && b == 0 {
			return 0, 0, errCorrupt
		}
		v |= uint64(b&0x7F) << (7 * uint(i))
		if b&0x80 == 0 {
			if i > 0 && b == 0 {
				return 0, 0, errCorrupt
			}
			return v, i + 1, nil
		}
	}
	return 0, 0, errCorrupt
}

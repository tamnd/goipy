package lzma

import (
	"crypto/sha256"
	"hash/crc32"
)

// crc32IEEE wraps stdlib CRC-32 (IEEE) matching XZ's check ID 1.
func crc32IEEE(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

// CRC-64 ECMA-182 with reflected polynomial 0xC96C5795_7F412614 (the
// variant XZ uses for check ID 4).
var crc64Table [256]uint64

func init() {
	const poly = uint64(0xC96C5795_7F412614)
	for i := 0; i < 256; i++ {
		c := uint64(i)
		for j := 0; j < 8; j++ {
			if c&1 != 0 {
				c = (c >> 1) ^ poly
			} else {
				c >>= 1
			}
		}
		crc64Table[i] = c
	}
}

func crc64ECMA(data []byte) uint64 {
	var c uint64 = 0xFFFFFFFF_FFFFFFFF
	for _, b := range data {
		c = crc64Table[byte(c)^b] ^ (c >> 8)
	}
	return c ^ 0xFFFFFFFF_FFFFFFFF
}

func sha256Sum(data []byte) [32]byte {
	return sha256.Sum256(data)
}

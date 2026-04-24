// Package zstd provides a minimal RFC 8878 Zstandard codec.
//
// This file implements XXH64, the checksum zstd stores at the end of
// frames when the Content_Checksum_flag is set. Only the low 32 bits
// are written to the frame — the full 64-bit digest is returned here
// so callers can also use it standalone.
package zstd

import "encoding/binary"

const (
	xxh64Prime1 uint64 = 0x9E3779B185EBCA87
	xxh64Prime2 uint64 = 0xC2B2AE3D27D4EB4F
	xxh64Prime3 uint64 = 0x165667B19E3779F9
	xxh64Prime4 uint64 = 0x85EBCA77C2B2AE63
	xxh64Prime5 uint64 = 0x27D4EB2F165667C5
)

func xxh64Round(acc, input uint64) uint64 {
	acc += input * xxh64Prime2
	acc = (acc << 31) | (acc >> 33)
	return acc * xxh64Prime1
}

func xxh64MergeRound(acc, v uint64) uint64 {
	v = xxh64Round(0, v)
	acc ^= v
	return acc*xxh64Prime1 + xxh64Prime4
}

// XXH64 computes the 64-bit XXH64 digest of data with seed 0.
func XXH64(data []byte) uint64 {
	length := uint64(len(data))
	var h uint64
	p := 0
	if len(data) >= 32 {
		// Force runtime arithmetic so Go allows the wrapping.
		one := uint64(1)
		v1 := xxh64Prime1*one + xxh64Prime2
		v2 := xxh64Prime2
		var v3 uint64
		v4 := -(xxh64Prime1 * one)
		for len(data)-p >= 32 {
			v1 = xxh64Round(v1, binary.LittleEndian.Uint64(data[p:]))
			p += 8
			v2 = xxh64Round(v2, binary.LittleEndian.Uint64(data[p:]))
			p += 8
			v3 = xxh64Round(v3, binary.LittleEndian.Uint64(data[p:]))
			p += 8
			v4 = xxh64Round(v4, binary.LittleEndian.Uint64(data[p:]))
			p += 8
		}
		h = rotl64(v1, 1) + rotl64(v2, 7) + rotl64(v3, 12) + rotl64(v4, 18)
		h = xxh64MergeRound(h, v1)
		h = xxh64MergeRound(h, v2)
		h = xxh64MergeRound(h, v3)
		h = xxh64MergeRound(h, v4)
	} else {
		h = xxh64Prime5
	}
	h += length

	for len(data)-p >= 8 {
		k := xxh64Round(0, binary.LittleEndian.Uint64(data[p:]))
		h ^= k
		h = rotl64(h, 27)*xxh64Prime1 + xxh64Prime4
		p += 8
	}
	if len(data)-p >= 4 {
		h ^= uint64(binary.LittleEndian.Uint32(data[p:])) * xxh64Prime1
		h = rotl64(h, 23)*xxh64Prime2 + xxh64Prime3
		p += 4
	}
	for p < len(data) {
		h ^= uint64(data[p]) * xxh64Prime5
		h = rotl64(h, 11) * xxh64Prime1
		p++
	}

	h ^= h >> 33
	h *= xxh64Prime2
	h ^= h >> 29
	h *= xxh64Prime3
	h ^= h >> 32
	return h
}

func rotl64(x uint64, r uint) uint64 {
	return (x << r) | (x >> (64 - r))
}

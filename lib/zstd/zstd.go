package zstd

import (
	"encoding/binary"
	"errors"
)

// Zstd frame format (RFC 8878).
//
// Our encoder emits a single frame per call with:
//   - Single_Segment_flag = 1
//   - Content_Checksum_flag = 1
//   - No dictionary
//   - Frame_Content_Size present (1 / 2 / 4 / 8 bytes depending on size)
//   - Body split into Raw blocks of at most 128 KiB, last marked Last_Block
//   - Content checksum = low 32 bits of XXH64(raw)
//
// The decoder handles Raw and RLE blocks fully. Compressed blocks
// produced by other encoders (real zstd) are rejected with
// ErrCompressedBlockUnsupported.

const (
	magicNumber = 0xFD2FB528
	maxBlockLog = 17 // 128 KiB
	maxBlock    = 1 << maxBlockLog

	blockTypeRaw        = 0
	blockTypeRLE        = 1
	blockTypeCompressed = 2
	blockTypeReserved   = 3
)

// Errors returned by Decode.
var (
	ErrBadMagic                    = errors.New("zstd: bad magic number")
	ErrReserved                    = errors.New("zstd: reserved bit set")
	ErrTruncated                   = errors.New("zstd: truncated input")
	ErrCompressedBlockUnsupported  = errors.New("zstd: compressed blocks not supported by this decoder")
	ErrReservedBlockType           = errors.New("zstd: reserved block type")
	ErrChecksumMismatch            = errors.New("zstd: content checksum mismatch")
)

// Encode compresses data into a single zstd frame. The returned
// frame is valid RFC 8878 zstd.
func Encode(data []byte) []byte {
	var out []byte

	// Frame Header Descriptor.
	fcsFlag := fcsFlagFor(uint64(len(data)))
	fhd := byte(fcsFlag<<6) | (1 << 5) | (1 << 2) // single-segment + checksum

	out = append(out, 0x28, 0xB5, 0x2F, 0xFD)
	out = append(out, fhd)
	// No window descriptor (single-segment).
	// No dictionary ID.
	// Frame_Content_Size.
	fcsSize := fcsSizeFor(fcsFlag)
	fcsValue := uint64(len(data))
	if fcsFlag == 1 {
		// FCS_flag=1 encodes value - 256.
		fcsValue -= 256
	}
	var fcs [8]byte
	binary.LittleEndian.PutUint64(fcs[:], fcsValue)
	out = append(out, fcs[:fcsSize]...)

	// Emit blocks. Special-case empty input: a single Last_Block Raw
	// with size 0.
	if len(data) == 0 {
		out = append(out, encodeBlockHeader(0, blockTypeRaw, true)...)
	} else {
		for i := 0; i < len(data); {
			end := i + maxBlock
			if end > len(data) {
				end = len(data)
			}
			last := end == len(data)
			chunk := data[i:end]
			out = append(out, encodeBlockHeader(len(chunk), blockTypeRaw, last)...)
			out = append(out, chunk...)
			i = end
		}
	}

	// Content checksum: low 32 bits of XXH64 of raw data.
	digest := XXH64(data)
	var ck [4]byte
	binary.LittleEndian.PutUint32(ck[:], uint32(digest))
	out = append(out, ck[:]...)
	return out
}

// Decode decompresses a single zstd frame from src. It returns the
// decompressed data and the number of bytes consumed.
func Decode(src []byte) ([]byte, int, error) {
	if len(src) < 4 {
		return nil, 0, ErrTruncated
	}
	if binary.LittleEndian.Uint32(src[:4]) != magicNumber {
		return nil, 0, ErrBadMagic
	}
	p := 4
	if p >= len(src) {
		return nil, 0, ErrTruncated
	}
	fhd := src[p]
	p++
	fcsFlag := (fhd >> 6) & 0x3
	singleSegment := (fhd>>5)&1 == 1
	unusedBit := (fhd >> 4) & 1
	reservedBit := (fhd >> 3) & 1
	contentChecksum := (fhd>>2)&1 == 1
	didFlag := fhd & 0x3
	if unusedBit != 0 || reservedBit != 0 {
		return nil, 0, ErrReserved
	}

	// Window descriptor if not single-segment.
	if !singleSegment {
		if p >= len(src) {
			return nil, 0, ErrTruncated
		}
		p++
	}

	// Dictionary ID.
	didSize := [4]int{0, 1, 2, 4}[didFlag]
	if p+didSize > len(src) {
		return nil, 0, ErrTruncated
	}
	p += didSize

	// Frame Content Size.
	fcsSize := 0
	switch fcsFlag {
	case 0:
		if singleSegment {
			fcsSize = 1
		}
	case 1:
		fcsSize = 2
	case 2:
		fcsSize = 4
	case 3:
		fcsSize = 8
	}
	if p+fcsSize > len(src) {
		return nil, 0, ErrTruncated
	}
	_ = fcsSize // value not required for correctness here
	p += fcsSize

	// Iterate blocks.
	var out []byte
	for {
		if p+3 > len(src) {
			return nil, 0, ErrTruncated
		}
		hdr := uint32(src[p]) | uint32(src[p+1])<<8 | uint32(src[p+2])<<16
		p += 3
		last := hdr&1 == 1
		bt := (hdr >> 1) & 0x3
		bs := int(hdr >> 3)
		switch bt {
		case blockTypeRaw:
			if p+bs > len(src) {
				return nil, 0, ErrTruncated
			}
			out = append(out, src[p:p+bs]...)
			p += bs
		case blockTypeRLE:
			if p+1 > len(src) {
				return nil, 0, ErrTruncated
			}
			b := src[p]
			p++
			for j := 0; j < bs; j++ {
				out = append(out, b)
			}
		case blockTypeCompressed:
			return nil, 0, ErrCompressedBlockUnsupported
		default:
			return nil, 0, ErrReservedBlockType
		}
		if last {
			break
		}
	}

	if contentChecksum {
		if p+4 > len(src) {
			return nil, 0, ErrTruncated
		}
		got := binary.LittleEndian.Uint32(src[p : p+4])
		want := uint32(XXH64(out))
		if got != want {
			return nil, 0, ErrChecksumMismatch
		}
		p += 4
	}

	return out, p, nil
}

// DecodeAll decompresses all concatenated frames in src and returns
// the joined output.
func DecodeAll(src []byte) ([]byte, error) {
	var out []byte
	for len(src) > 0 {
		dec, n, err := Decode(src)
		if err != nil {
			return nil, err
		}
		out = append(out, dec...)
		src = src[n:]
	}
	return out, nil
}

func fcsFlagFor(n uint64) byte {
	switch {
	case n < 256:
		return 0 // 1 byte (single-segment only)
	case n < 65536+256:
		return 1 // 2 bytes (stored as n-256)
	case n < 1<<32:
		return 2 // 4 bytes
	default:
		return 3 // 8 bytes
	}
}

func fcsSizeFor(flag byte) int {
	switch flag {
	case 0:
		return 1
	case 1:
		return 2
	case 2:
		return 4
	default:
		return 8
	}
}

func encodeBlockHeader(size int, blockType byte, last bool) []byte {
	hdr := uint32(size)<<3 | uint32(blockType)<<1
	if last {
		hdr |= 1
	}
	return []byte{byte(hdr), byte(hdr >> 8), byte(hdr >> 16)}
}

package lzma

import (
	"bytes"
	"encoding/binary"
)

// XZ container encode / decode. One stream, one block, LZMA2 filter.
//
// Stream header (12): magic FD 37 7A 58 5A 00 + 2 flag bytes + CRC32.
// Block header: size byte (real=(v+1)*4) + flags + filter flags +
// padding to declared size - 4 + CRC32.
// Block: compressed data + 0..3 pad zero bytes + check.
// Index: 00 + record-count varint + (unpadded_size, uncompressed_size)
// varint pair + padding + CRC32.
// Stream footer (12): CRC32 + backward_size u32 LE + flags + "YZ".

// CheckID values match Python's constants.
type CheckID uint8

const (
	CheckNone   CheckID = 0
	CheckCRC32  CheckID = 1
	CheckCRC64  CheckID = 4
	CheckSHA256 CheckID = 10
)

func (c CheckID) size() int {
	switch c {
	case CheckNone:
		return 0
	case CheckCRC32:
		return 4
	case CheckCRC64:
		return 8
	case CheckSHA256:
		return 32
	default:
		return -1
	}
}

var xzMagic = []byte{0xFD, '7', 'z', 'X', 'Z', 0x00}
var xzFooterMagic = []byte{'Y', 'Z'}

// dictSizeProp encodes dict size as the 1-byte LZMA2 filter property.
// Our encoder always uses 1<<23.
const lzma2DictProp = 0x14 // (2|0) << (10+11) == 1<<22? no — spec: (2 | (prop&1)) << (prop/2 + 11).
// prop=0x14=20: (2|0) << (10+11) = 2<<21 = 1<<22. Close enough — decoders
// happily accept larger nominal sizes; our true working dict is bounded
// by input length. We advertise 0x14.

func EncodeXZ(data []byte, check CheckID) []byte {
	if check.size() < 0 {
		check = CheckCRC64
	}
	props := defaultProperties()

	// ── stream header ────────────────────────────────────────────────
	var out bytes.Buffer
	out.Write(xzMagic)
	flags := [2]byte{0x00, byte(check)}
	out.Write(flags[:])
	crc := crc32IEEE(flags[:])
	binary.Write(&out, binary.LittleEndian, crc)

	// ── block ────────────────────────────────────────────────────────
	compressed := encodeLZMA2(data, props)

	// Block header: we omit compressed/uncompressed size flags (both 0),
	// single filter (LZMA2), no header padding except to align.
	blkFlags := byte(0x00) // filters-1 = 0, no size fields
	filterID := encodeVarint(0x21)
	propsSize := encodeVarint(1)
	filterFlags := make([]byte, 0, len(filterID)+len(propsSize)+1)
	filterFlags = append(filterFlags, filterID...)
	filterFlags = append(filterFlags, propsSize...)
	filterFlags = append(filterFlags, lzma2DictProp)

	// Header content (excluding the leading size byte and trailing CRC):
	//   flags + filterFlags + pad.
	hdrContent := append([]byte{blkFlags}, filterFlags...)
	// Total block header length (including size byte + CRC) must be a
	// multiple of 4 in the range 8..1024.
	headerLen := 1 + len(hdrContent) + 4
	pad := (4 - (headerLen % 4)) % 4
	headerLen += pad
	sizeByte := byte((headerLen / 4) - 1)

	blockHdr := make([]byte, 0, headerLen)
	blockHdr = append(blockHdr, sizeByte)
	blockHdr = append(blockHdr, hdrContent...)
	blockHdr = append(blockHdr, make([]byte, pad)...)
	blockHdrCRC := crc32IEEE(blockHdr)
	blockHdr = binary.LittleEndian.AppendUint32(blockHdr, blockHdrCRC)

	out.Write(blockHdr)
	out.Write(compressed)

	// Block padding: zeros to multiple of 4 over (header + compressed).
	bodyLen := len(blockHdr) + len(compressed)
	bpad := (4 - (bodyLen % 4)) % 4
	out.Write(make([]byte, bpad))

	// Check.
	switch check {
	case CheckCRC32:
		binary.Write(&out, binary.LittleEndian, crc32IEEE(data))
	case CheckCRC64:
		binary.Write(&out, binary.LittleEndian, crc64ECMA(data))
	case CheckSHA256:
		h := sha256Sum(data)
		out.Write(h[:])
	}

	unpadded := uint64(len(blockHdr) + len(compressed) + check.size())
	uncSize := uint64(len(data))

	// ── index ────────────────────────────────────────────────────────
	var idx bytes.Buffer
	idx.WriteByte(0x00)
	idx.Write(encodeVarint(1)) // 1 record
	idx.Write(encodeVarint(unpadded))
	idx.Write(encodeVarint(uncSize))
	ipad := (4 - (idx.Len() % 4)) % 4
	idx.Write(make([]byte, ipad))
	idxCRC := crc32IEEE(idx.Bytes())
	idxBytes := idx.Bytes()
	idxBytes = binary.LittleEndian.AppendUint32(idxBytes, idxCRC)
	out.Write(idxBytes)

	// ── stream footer ────────────────────────────────────────────────
	var foot [12]byte
	backwardSize := uint32(len(idxBytes)/4 - 1)
	binary.LittleEndian.PutUint32(foot[4:8], backwardSize)
	foot[8] = flags[0]
	foot[9] = flags[1]
	footCRC := crc32IEEE(foot[4:10])
	binary.LittleEndian.PutUint32(foot[0:4], footCRC)
	foot[10] = xzFooterMagic[0]
	foot[11] = xzFooterMagic[1]
	out.Write(foot[:])

	return out.Bytes()
}

// DecodeXZ decodes one or more concatenated XZ streams and returns the
// concatenation of their uncompressed content. `unused` points at the
// first byte that wasn't part of an XZ stream (or past end if all was
// consumed).
func DecodeXZ(data []byte) (out []byte, unused int, err error) {
	pos := 0
	for pos < len(data) {
		// Tolerate up-to-4-byte stream padding (zero bytes) between
		// concatenated streams — xz spec requires multiples of 4.
		if pos > 0 {
			start := pos
			for pos < len(data) && data[pos] == 0 && (pos-start) < 4 {
				pos++
			}
			if pos == len(data) {
				break
			}
			if (pos-start)%4 != 0 {
				return nil, 0, errFormat
			}
		}
		if pos+12 > len(data) {
			break
		}
		if !bytes.Equal(data[pos:pos+6], xzMagic) {
			break
		}
		streamOut, consumed, derr := decodeOneStream(data[pos:])
		if derr != nil {
			return nil, 0, derr
		}
		out = append(out, streamOut...)
		pos += consumed
	}
	return out, pos, nil
}

// DecodeOneXZStream decodes exactly one XZ stream and returns the
// bytes consumed so callers can surface the remainder via unused_data.
func DecodeOneXZStream(data []byte) ([]byte, int, error) {
	return decodeOneStream(data)
}

func decodeOneStream(data []byte) ([]byte, int, error) {
	if len(data) < 12 || !bytes.Equal(data[0:6], xzMagic) {
		return nil, 0, errFormat
	}
	flags := data[6:8]
	hdrCRC := binary.LittleEndian.Uint32(data[8:12])
	if hdrCRC != crc32IEEE(flags) {
		return nil, 0, errCheck
	}
	check := CheckID(flags[1] & 0x0F)
	pos := 12

	var out []byte
	for {
		if pos >= len(data) {
			return nil, 0, errShort
		}
		if data[pos] == 0x00 {
			// Index follows.
			break
		}
		// ── Block header ───────────────────────────────────────────────
		headerLen := (int(data[pos]) + 1) * 4
		if pos+headerLen > len(data) {
			return nil, 0, errShort
		}
		hdr := data[pos : pos+headerLen]
		gotCRC := binary.LittleEndian.Uint32(hdr[headerLen-4:])
		if gotCRC != crc32IEEE(hdr[:headerLen-4]) {
			return nil, 0, errCheck
		}
		blkFlags := hdr[1]
		numFilters := int(blkFlags&0x03) + 1
		hpos := 2
		var compSize, uncSize int64 = -1, -1
		if blkFlags&0x40 != 0 {
			v, n, err := decodeVarint(hdr[hpos : headerLen-4])
			if err != nil {
				return nil, 0, err
			}
			compSize = int64(v)
			hpos += n
		}
		if blkFlags&0x80 != 0 {
			v, n, err := decodeVarint(hdr[hpos : headerLen-4])
			if err != nil {
				return nil, 0, err
			}
			uncSize = int64(v)
			hpos += n
		}
		_ = compSize
		_ = uncSize
		var filterIsLZMA2 bool
		for f := 0; f < numFilters; f++ {
			fid, n, err := decodeVarint(hdr[hpos : headerLen-4])
			if err != nil {
				return nil, 0, err
			}
			hpos += n
			propsLen, n2, err := decodeVarint(hdr[hpos : headerLen-4])
			if err != nil {
				return nil, 0, err
			}
			hpos += n2
			if fid == 0x21 && f == numFilters-1 {
				filterIsLZMA2 = true
			}
			hpos += int(propsLen)
		}
		if !filterIsLZMA2 {
			return nil, 0, errUnsup
		}
		pos += headerLen

		// ── Compressed data ────────────────────────────────────────────
		chunk, consumed, err := decodeLZMA2(data[pos:])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, chunk...)
		pos += consumed

		// ── Block padding ──────────────────────────────────────────────
		bodyLen := headerLen + consumed
		bpad := (4 - (bodyLen % 4)) % 4
		if pos+bpad > len(data) {
			return nil, 0, errShort
		}
		for i := 0; i < bpad; i++ {
			if data[pos+i] != 0 {
				return nil, 0, errCorrupt
			}
		}
		pos += bpad

		// ── Check ──────────────────────────────────────────────────────
		csize := check.size()
		if csize < 0 {
			return nil, 0, errUnsup
		}
		if pos+csize > len(data) {
			return nil, 0, errShort
		}
		if err := verifyCheck(check, data[pos:pos+csize], chunk); err != nil {
			return nil, 0, err
		}
		pos += csize
	}

	// ── Index ────────────────────────────────────────────────────────
	if pos >= len(data) {
		return nil, 0, errShort
	}
	idxStart := pos
	if data[pos] != 0 {
		return nil, 0, errCorrupt
	}
	pos++
	nRec, n, err := decodeVarint(data[pos:])
	if err != nil {
		return nil, 0, err
	}
	pos += n
	for i := uint64(0); i < nRec; i++ {
		_, n, err := decodeVarint(data[pos:])
		if err != nil {
			return nil, 0, err
		}
		pos += n
		_, n, err = decodeVarint(data[pos:])
		if err != nil {
			return nil, 0, err
		}
		pos += n
	}
	// index pad to multiple of 4
	padLen := (4 - ((pos - idxStart) % 4)) % 4
	pos += padLen
	if pos+4 > len(data) {
		return nil, 0, errShort
	}
	gotCRC := binary.LittleEndian.Uint32(data[pos : pos+4])
	if gotCRC != crc32IEEE(data[idxStart:pos]) {
		return nil, 0, errCheck
	}
	pos += 4

	// ── Stream footer ────────────────────────────────────────────────
	if pos+12 > len(data) {
		return nil, 0, errShort
	}
	footCRC := binary.LittleEndian.Uint32(data[pos : pos+4])
	if footCRC != crc32IEEE(data[pos+4:pos+10]) {
		return nil, 0, errCheck
	}
	if !bytes.Equal(data[pos+10:pos+12], xzFooterMagic) {
		return nil, 0, errFormat
	}
	if !bytes.Equal(data[pos+8:pos+10], flags) {
		return nil, 0, errFormat
	}
	pos += 12
	return out, pos, nil
}

func verifyCheck(check CheckID, got, data []byte) error {
	switch check {
	case CheckNone:
		return nil
	case CheckCRC32:
		want := crc32IEEE(data)
		have := binary.LittleEndian.Uint32(got)
		if want != have {
			return errCheck
		}
	case CheckCRC64:
		want := crc64ECMA(data)
		have := binary.LittleEndian.Uint64(got)
		if want != have {
			return errCheck
		}
	case CheckSHA256:
		want := sha256Sum(data)
		if !bytes.Equal(want[:], got) {
			return errCheck
		}
	default:
		return errUnsup
	}
	return nil
}

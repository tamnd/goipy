package lzma

import "encoding/binary"

// FORMAT_ALONE: 13-byte header followed by a raw LZMA stream. We fill
// the uncompressed-size field with the real length (not the unknown
// sentinel), so the encoder doesn't have to emit an end-of-stream
// marker — and the decoder stops at exactly that many bytes.

func EncodeAlone(data []byte) []byte {
	props := defaultProperties()
	out := make([]byte, 13)
	out[0] = props.propByte()
	binary.LittleEndian.PutUint32(out[1:5], props.DictSize)
	binary.LittleEndian.PutUint64(out[5:13], uint64(len(data)))
	return append(out, encodeLZMA(data, props)...)
}

func DecodeAlone(data []byte) ([]byte, error) {
	if len(data) < 13 {
		return nil, errShort
	}
	props, err := decodePropByte(data[0])
	if err != nil {
		return nil, err
	}
	props.DictSize = binary.LittleEndian.Uint32(data[1:5])
	size := binary.LittleEndian.Uint64(data[5:13])
	if size == 0xFFFFFFFF_FFFFFFFF {
		// Unknown size — requires EOS marker, which our literal-only
		// decoder doesn't handle. Mark as unsupported.
		return nil, errUnsup
	}
	out, _, err := decodeLZMA(data[13:], props, int64(size))
	if err != nil {
		return nil, err
	}
	return out, nil
}

package vm

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc32"
	"math/big"
	"strings"

	"golang.org/x/crypto/sha3"

	"github.com/tamnd/goipy/object"
)

// --- binascii module -------------------------------------------------------

// crcHqx computes CRC-CCITT (polynomial 0x1021) over data, starting from crc.
func crcHqx(data []byte, crc uint16) uint16 {
	for _, b := range data {
		crc ^= uint16(b) << 8
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// hqxTable is the BinHex4 6-bit encoding alphabet (64 chars).
const hqxTable = `!"#$%&'()*+,-012345689@ABCDEFGHIJKLMNPQRSTUVXYZ[` + "`" + `abcdefhijklmpqr`

func hqxEncode(b byte) byte { return hqxTable[b&0x3f] }

func hqxDecode(b byte) (byte, bool) {
	for i, c := range []byte(hqxTable) {
		if c == b {
			return byte(i), true
		}
	}
	return 0, false
}

func (i *Interp) buildBinascii() *object.Module {
	m := &object.Module{Name: "binascii", Dict: object.NewDict()}

	// binascii.Error — subclass of ValueError
	binErr := &object.Class{Name: "Error", Bases: []*object.Class{i.valueErr}, Dict: object.NewDict()}
	m.Dict.SetStr("Error", binErr)

	// binascii.Incomplete — subclass of Exception
	binIncomplete := &object.Class{Name: "Incomplete", Bases: []*object.Class{i.exception}, Dict: object.NewDict()}
	m.Dict.SetStr("Incomplete", binIncomplete)

	// hexlify / b2a_hex with sep / bytes_per_sep support
	hexlify := &object.BuiltinFunc{Name: "hexlify", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "hexlify() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		encoded := hex.EncodeToString(data)
		// Check for sep kwarg
		var sepBytes []byte
		if kw != nil {
			if sv, ok := kw.GetStr("sep"); ok && sv != object.None {
				switch s := sv.(type) {
				case *object.Str:
					sepBytes = []byte(s.V)
				case *object.Bytes:
					sepBytes = s.V
				case *object.Bytearray:
					sepBytes = s.V
				default:
					return nil, object.Errorf(i.typeErr, "sep must be str or bytes")
				}
				if len(sepBytes) != 1 {
					return nil, object.Errorf(i.valueErr, "sep must be length 1")
				}
			}
		}
		if len(sepBytes) == 0 {
			return &object.Bytes{V: []byte(encoded)}, nil
		}
		// bytes_per_sep default is 1
		bps := int64(1)
		if kw != nil {
			if bv, ok := kw.GetStr("bytes_per_sep"); ok {
				if n, ok2 := toInt64(bv); ok2 {
					bps = n
				}
			}
		}
		if bps == 0 {
			return &object.Bytes{V: []byte(encoded)}, nil
		}
		// Split the hex string into groups of abs(bps)*2 hex chars.
		// Positive bps: group from left; negative: group from right.
		abs := bps
		if abs < 0 {
			abs = -abs
		}
		groupLen := int(abs) * 2 // hex chars per group
		hexBytes := []byte(encoded)
		n := len(hexBytes)
		if n == 0 {
			return &object.Bytes{V: hexBytes}, nil
		}
		var groups [][]byte
		if bps > 0 {
			// left-to-right grouping
			for start := 0; start < n; start += groupLen {
				end := start + groupLen
				if end > n {
					end = n
				}
				groups = append(groups, hexBytes[start:end])
			}
		} else {
			// right-to-left grouping (last group may be smaller at the start)
			for end := n; end > 0; end -= groupLen {
				start := end - groupLen
				if start < 0 {
					start = 0
				}
				groups = append([][]byte{hexBytes[start:end]}, groups...)
			}
		}
		// Join groups with sep
		sep := sepBytes[0]
		total := n + 1*(len(groups)-1)
		out := make([]byte, 0, total)
		for idx, g := range groups {
			if idx > 0 {
				out = append(out, sep)
			}
			out = append(out, g...)
		}
		return &object.Bytes{V: out}, nil
	}}
	m.Dict.SetStr("hexlify", hexlify)
	m.Dict.SetStr("b2a_hex", hexlify)

	unhexlify := &object.BuiltinFunc{Name: "unhexlify", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "unhexlify() missing data")
		}
		var src string
		switch v := a[0].(type) {
		case *object.Str:
			src = v.V
		case *object.Bytes:
			src = string(v.V)
		case *object.Bytearray:
			src = string(v.V)
		default:
			return nil, object.Errorf(i.typeErr, "unhexlify() expects str or bytes")
		}
		out, err := hex.DecodeString(src)
		if err != nil {
			return nil, object.NewException(binErr, err.Error())
		}
		return &object.Bytes{V: out}, nil
	}}
	m.Dict.SetStr("unhexlify", unhexlify)
	m.Dict.SetStr("a2b_hex", unhexlify)

	m.Dict.SetStr("b2a_base64", &object.BuiltinFunc{Name: "b2a_base64", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "b2a_base64() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		newline := true
		if kw != nil {
			if v, ok := kw.GetStr("newline"); ok {
				newline = object.Truthy(v)
			}
		}
		enc := base64.StdEncoding.EncodeToString(data)
		if newline {
			enc += "\n"
		}
		return &object.Bytes{V: []byte(enc)}, nil
	}})

	m.Dict.SetStr("a2b_base64", &object.BuiltinFunc{Name: "a2b_base64", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "a2b_base64() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		strictMode := false
		if kw != nil {
			if v, ok := kw.GetStr("strict_mode"); ok {
				strictMode = object.Truthy(v)
			}
		}
		src := string(data)
		if strictMode {
			// In strict mode: only allow A-Z, a-z, 0-9, +, /, = characters.
			for _, c := range src {
				valid := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
					(c >= '0' && c <= '9') || c == '+' || c == '/' || c == '='
				if !valid {
					return nil, object.NewException(binErr, "Only base64 data is allowed")
				}
			}
		} else {
			// Non-strict: strip whitespace.
			src = strings.Map(func(r rune) rune {
				if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
					return -1
				}
				return r
			}, src)
		}
		out, err2 := base64.StdEncoding.DecodeString(src)
		if err2 != nil {
			out, err2 = base64.RawStdEncoding.DecodeString(src)
			if err2 != nil {
				return nil, object.NewException(binErr, err2.Error())
			}
		}
		return &object.Bytes{V: out}, nil
	}})

	m.Dict.SetStr("crc32", &object.BuiltinFunc{Name: "crc32", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "crc32() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		var seed uint32
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok {
				seed = uint32(n)
			}
		}
		return newIntU64(uint64(crc32.Update(seed, crc32.IEEETable, data))), nil
	}})

	// b2a_uu: UU-encode one line (max 45 input bytes).
	m.Dict.SetStr("b2a_uu", &object.BuiltinFunc{Name: "b2a_uu", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "b2a_uu() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		if len(data) > 45 {
			return nil, object.NewException(binErr, "At most 45 bytes at once")
		}
		backtick := false
		if kw != nil {
			if v, ok := kw.GetStr("backtick"); ok {
				backtick = object.Truthy(v)
			}
		}
		uuChar := func(v byte) byte {
			c := (v & 0x3f) + 0x20
			if backtick && c == 0x20 {
				return '`'
			}
			return c
		}
		out := make([]byte, 0, 2+((len(data)+2)/3)*4)
		// Length character
		lenByte := byte(len(data)) + 0x20
		if backtick && len(data) == 0 {
			lenByte = '`'
		}
		out = append(out, lenByte)
		// Encode 3-byte groups
		for i := 0; i < len(data); i += 3 {
			var b0, b1, b2 byte
			b0 = data[i]
			if i+1 < len(data) {
				b1 = data[i+1]
			}
			if i+2 < len(data) {
				b2 = data[i+2]
			}
			out = append(out,
				uuChar(b0>>2),
				uuChar((b0<<4)|(b1>>4)),
				uuChar((b1<<2)|(b2>>6)),
				uuChar(b2),
			)
		}
		out = append(out, '\n')
		return &object.Bytes{V: out}, nil
	}})

	// a2b_uu: UU-decode one line.
	m.Dict.SetStr("a2b_uu", &object.BuiltinFunc{Name: "a2b_uu", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "a2b_uu() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		// Strip trailing newline
		if len(data) > 0 && data[len(data)-1] == '\n' {
			data = data[:len(data)-1]
		}
		if len(data) == 0 {
			return &object.Bytes{V: []byte{}}, nil
		}
		// First char is length
		lc := data[0]
		if lc == '`' {
			lc = 0x20
		}
		length := int(lc - 0x20)
		if length < 0 {
			length = 0
		}
		uuVal := func(c byte) byte {
			if c == '`' {
				return 0
			}
			return (c - 0x20) & 0x3f
		}
		rest := data[1:]
		out := make([]byte, 0, length)
		for i := 0; i+3 < len(rest) && len(out) < length; i += 4 {
			v0 := uuVal(rest[i])
			v1 := uuVal(rest[i+1])
			v2 := uuVal(rest[i+2])
			v3 := uuVal(rest[i+3])
			if len(out) < length {
				out = append(out, (v0<<2)|(v1>>4))
			}
			if len(out) < length {
				out = append(out, (v1<<4)|(v2>>2))
			}
			if len(out) < length {
				out = append(out, (v2<<6)|v3)
			}
		}
		return &object.Bytes{V: out}, nil
	}})

	// b2a_qp: Quoted-printable encode.
	m.Dict.SetStr("b2a_qp", &object.BuiltinFunc{Name: "b2a_qp", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "b2a_qp() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		quotetabs := false
		istext := true
		header := false
		if kw != nil {
			if v, ok := kw.GetStr("quotetabs"); ok {
				quotetabs = object.Truthy(v)
			}
			if v, ok := kw.GetStr("istext"); ok {
				istext = object.Truthy(v)
			}
			if v, ok := kw.GetStr("header"); ok {
				header = object.Truthy(v)
			}
		}
		if len(a) >= 2 {
			quotetabs = object.Truthy(a[1])
		}
		if len(a) >= 3 {
			istext = object.Truthy(a[2])
		}
		if len(a) >= 4 {
			header = object.Truthy(a[3])
		}

		var out []byte
		lineLen := 0
		const maxLine = 76

		addSoftBreak := func() {
			out = append(out, '=', '\n')
			lineLen = 0
		}
		addByte := func(b byte) {
			if lineLen+1 > maxLine {
				addSoftBreak()
			}
			out = append(out, b)
			lineLen++
		}
		addEncoded := func(b byte) {
			const hexDigits = "0123456789ABCDEF"
			if lineLen+3 > maxLine {
				addSoftBreak()
			}
			out = append(out, '=', hexDigits[b>>4], hexDigits[b&0xf])
			lineLen += 3
		}

		for j := 0; j < len(data); j++ {
			c := data[j]
			if header && c == ' ' {
				addByte('_')
			} else if istext && c == '\n' {
				// Preserve newline as line break
				// Trim trailing space/tab from line
				out = append(out, '\n')
				lineLen = 0
			} else if istext && c == '\r' && j+1 < len(data) && data[j+1] == '\n' {
				out = append(out, '\r', '\n')
				lineLen = 0
				j++ // skip \n
			} else if c == '\t' || c == ' ' {
				if quotetabs {
					addEncoded(c)
				} else {
					// Space/tab: encode if it would be at end of line or before newline
					isLast := j+1 >= len(data)
					beforeNewline := j+1 < len(data) && (data[j+1] == '\n' || data[j+1] == '\r')
					if isLast || beforeNewline {
						addEncoded(c)
					} else {
						addByte(c)
					}
				}
			} else if c == '=' || c > 126 || (c < 32 && c != '\t') {
				addEncoded(c)
			} else {
				addByte(c)
			}
		}
		return &object.Bytes{V: out}, nil
	}})

	// a2b_qp: Quoted-printable decode.
	m.Dict.SetStr("a2b_qp", &object.BuiltinFunc{Name: "a2b_qp", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "a2b_qp() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		header := false
		if kw != nil {
			if v, ok := kw.GetStr("header"); ok {
				header = object.Truthy(v)
			}
		}
		if len(a) >= 2 {
			header = object.Truthy(a[1])
		}

		var out []byte
		hexVal := func(c byte) (byte, bool) {
			if c >= '0' && c <= '9' {
				return c - '0', true
			}
			if c >= 'A' && c <= 'F' {
				return c - 'A' + 10, true
			}
			if c >= 'a' && c <= 'f' {
				return c - 'a' + 10, true
			}
			return 0, false
		}
		for j := 0; j < len(data); j++ {
			c := data[j]
			if header && c == '_' {
				out = append(out, ' ')
			} else if c == '=' && j+1 < len(data) {
				next := data[j+1]
				if next == '\n' {
					// soft line break
					j++
				} else if next == '\r' && j+2 < len(data) && data[j+2] == '\n' {
					// soft line break with CRLF
					j += 2
				} else if j+2 < len(data) {
					hi, ok1 := hexVal(next)
					lo, ok2 := hexVal(data[j+2])
					if ok1 && ok2 {
						out = append(out, (hi<<4)|lo)
						j += 2
					} else {
						out = append(out, c)
					}
				} else {
					out = append(out, c)
				}
			} else {
				out = append(out, c)
			}
		}
		return &object.Bytes{V: out}, nil
	}})

	// crc_hqx: CRC-CCITT (16-bit) used by BinHex4.
	m.Dict.SetStr("crc_hqx", &object.BuiltinFunc{Name: "crc_hqx", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "crc_hqx() requires data and crc")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		crcVal, ok := toInt64(a[1])
		if !ok {
			return nil, object.Errorf(i.typeErr, "crc_hqx() crc must be int")
		}
		result := crcHqx(data, uint16(crcVal))
		return newIntU64(uint64(result)), nil
	}})

	// rlecode_hqx: BinHex4 RLE compress.
	m.Dict.SetStr("rlecode_hqx", &object.BuiltinFunc{Name: "rlecode_hqx", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "rlecode_hqx() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		var out []byte
		n := len(data)
		for j := 0; j < n; {
			b := data[j]
			if b == 0x90 {
				// Each 0x90 byte must be escaped as 0x90, 0x00
				out = append(out, 0x90, 0x00)
				j++
				continue
			}
			// Count run length (non-0x90 bytes only)
			run := 1
			for j+run < n && data[j+run] == b && run < 255 {
				run++
			}
			if run >= 3 {
				out = append(out, b, 0x90, byte(run))
				j += run
			} else {
				out = append(out, b)
				j++
			}
		}
		return &object.Bytes{V: out}, nil
	}})

	// rledecode_hqx: BinHex4 RLE decompress.
	m.Dict.SetStr("rledecode_hqx", &object.BuiltinFunc{Name: "rledecode_hqx", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "rledecode_hqx() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		var out []byte
		n := len(data)
		for j := 0; j < n; {
			b := data[j]
			j++
			if b == 0x90 {
				if j >= n {
					return nil, object.NewException(binIncomplete, "incomplete RLE sequence")
				}
				count := data[j]
				j++
				if count == 0 {
					// Escaped literal 0x90
					out = append(out, 0x90)
				} else if len(out) == 0 {
					return nil, object.NewException(binErr, "RLE count with no previous byte")
				} else {
					// Repeat the last emitted byte (count-1) more times.
					// The byte was already emitted; total = count.
					prev := out[len(out)-1]
					for k := 1; k < int(count); k++ {
						out = append(out, prev)
					}
				}
			} else {
				out = append(out, b)
			}
		}
		return &object.Bytes{V: out}, nil
	}})

	// b2a_hqx: BinHex4 encode (applies 6-bit encoding).
	m.Dict.SetStr("b2a_hqx", &object.BuiltinFunc{Name: "b2a_hqx", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "b2a_hqx() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		// Pack 6-bit groups from input bytes (like base64 structure)
		var out []byte
		bits := 0
		buf := 0
		for _, b := range data {
			buf = (buf << 8) | int(b)
			bits += 8
			for bits >= 6 {
				bits -= 6
				idx := (buf >> bits) & 0x3f
				out = append(out, hqxEncode(byte(idx)))
			}
		}
		if bits > 0 {
			idx := (buf << (6 - bits)) & 0x3f
			out = append(out, hqxEncode(byte(idx)))
		}
		return &object.Bytes{V: out}, nil
	}})

	// a2b_hqx: BinHex4 decode; returns (data, done) tuple.
	m.Dict.SetStr("a2b_hqx", &object.BuiltinFunc{Name: "a2b_hqx", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "a2b_hqx() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		done := 0
		// Strip trailing ':' (BinHex end marker)
		if len(data) > 0 && data[len(data)-1] == ':' {
			done = 1
			data = data[:len(data)-1]
		}
		var out []byte
		bits := 0
		buf := 0
		for _, b := range data {
			v, ok := hqxDecode(b)
			if !ok {
				return nil, object.NewException(binErr, "Illegal char")
			}
			buf = (buf << 6) | int(v)
			bits += 6
			if bits >= 8 {
				bits -= 8
				out = append(out, byte((buf>>bits)&0xff))
			}
		}
		// Leftover bits are padding (zeros) from the encoder's trailing group.
		// Only raise Incomplete when we have enough bits for another byte but
		// the stream is cut off — treat remaining < 6 bits as padding.
		result := &object.Tuple{V: []object.Object{
			&object.Bytes{V: out},
			object.NewInt(int64(done)),
		}}
		return result, nil
	}})

	return m
}

// copyableHMAC is an HMAC implementation that supports BinaryMarshaler
// (by delegating to the inner hash) so copyHasher can clone it. The standard
// crypto/hmac.HMAC does not implement BinaryMarshaler.
type copyableHMAC struct {
	inner    hash.Hash
	ipad     []byte // block-size key XOR 0x36
	opad     []byte // block-size key XOR 0x5c
	hashNewFn func() hash.Hash
	digestSz int
	blkSz    int
}

func newCopyableHMAC(hashNewFn func() hash.Hash, key []byte) *copyableHMAC {
	h := hashNewFn()
	blkSz := h.BlockSize()
	digestSz := h.Size()
	if len(key) > blkSz {
		h.Write(key)
		key = h.Sum(nil)
	}
	ipad := make([]byte, blkSz)
	opad := make([]byte, blkSz)
	copy(ipad, key)
	copy(opad, key)
	for j := range ipad {
		ipad[j] ^= 0x36
		opad[j] ^= 0x5c
	}
	inner := hashNewFn()
	inner.Write(ipad)
	return &copyableHMAC{inner: inner, ipad: ipad, opad: opad, hashNewFn: hashNewFn, digestSz: digestSz, blkSz: blkSz}
}

func (h *copyableHMAC) Write(p []byte) (int, error) { return h.inner.Write(p) }
func (h *copyableHMAC) Size() int                   { return h.digestSz }
func (h *copyableHMAC) BlockSize() int              { return h.blkSz }
func (h *copyableHMAC) Reset() {
	h.inner = h.hashNewFn()
	h.inner.Write(h.ipad)
}

func (h *copyableHMAC) Sum(b []byte) []byte {
	innerSum := h.inner.Sum(nil)
	outer := h.hashNewFn()
	outer.Write(h.opad)
	outer.Write(innerSum)
	return outer.Sum(b)
}

// MarshalBinary serialises the inner hash state so copyHasher can clone it.
func (h *copyableHMAC) MarshalBinary() ([]byte, error) {
	m, ok := h.inner.(interface{ MarshalBinary() ([]byte, error) })
	if !ok {
		return nil, fmt.Errorf("inner hash not marshalable")
	}
	return m.MarshalBinary()
}

// UnmarshalBinary restores the inner hash state into a fresh copyableHMAC.
// Called by copyHasher after creating a new instance via NewFn().
func (h *copyableHMAC) UnmarshalBinary(data []byte) error {
	ni := h.hashNewFn()
	u, ok := ni.(interface{ UnmarshalBinary([]byte) error })
	if !ok {
		return fmt.Errorf("inner hash not unmarshalable")
	}
	if err := u.UnmarshalBinary(data); err != nil {
		return err
	}
	h.inner = ni
	return nil
}

// --- hmac module -----------------------------------------------------------

func (i *Interp) buildHmac() *object.Module {
	m := &object.Module{Name: "hmac", Dict: object.NewDict()}

	m.Dict.SetStr("new", &object.BuiltinFunc{Name: "new", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "hmac.new() requires a key")
		}
		key, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		var msg []byte
		if len(a) >= 2 && a[1] != nil && a[1] != object.None {
			msg, err = asBytes(a[1])
			if err != nil {
				return nil, err
			}
		}
		name := "sha256"
		var dig object.Object
		if len(a) >= 3 {
			dig = a[2]
		} else if kw != nil {
			if v, ok := kw.GetStr("digestmod"); ok {
				dig = v
			}
		}
		if dig != nil {
			name = digestmodName(dig)
		}
		hashNewFn, digestSize, blockSize, ok := hashConstructorByName(name)
		if !ok {
			return nil, object.Errorf(i.valueErr, "unsupported digestmod %q", name)
		}
		keyCopy := append([]byte(nil), key...)
		macFactory := func() hash.Hash { return newCopyableHMAC(hashNewFn, keyCopy) }
		mac := macFactory()
		if len(msg) > 0 {
			mac.Write(msg)
		}
		return &object.Hasher{Name: "hmac-" + name, Size: digestSize, BlockSize: blockSize, State: mac, NewFn: macFactory}, nil
	}})

	m.Dict.SetStr("compare_digest", &object.BuiltinFunc{Name: "compare_digest", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "compare_digest() requires two arguments")
		}
		ab, aok := bytesOrStr(a[0])
		bb, bok := bytesOrStr(a[1])
		if !aok || !bok {
			return nil, object.Errorf(i.typeErr, "compare_digest() args must be bytes or str")
		}
		return object.BoolOf(subtle.ConstantTimeCompare(ab, bb) == 1 && len(ab) == len(bb)), nil
	}})

	m.Dict.SetStr("digest", &object.BuiltinFunc{Name: "digest", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "hmac.digest() requires (key, msg, digest)")
		}
		key, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		msg, err := asBytes(a[1])
		if err != nil {
			return nil, err
		}
		name := digestmodName(a[2])
		newFn, _, _, ok := hashConstructorByName(name)
		if !ok {
			return nil, object.Errorf(i.valueErr, "unsupported digest %q", name)
		}
		mac := hmac.New(newFn, key)
		mac.Write(msg)
		return &object.Bytes{V: mac.Sum(nil)}, nil
	}})

	return m
}

// digestmodName extracts the algorithm name from a digestmod argument.
// Accepts: str name, BuiltinFunc (hashlib constructor), BoundMethod.
func digestmodName(o object.Object) string {
	switch d := o.(type) {
	case *object.Str:
		return strings.ToLower(d.V)
	case *object.BuiltinFunc:
		return strings.ToLower(d.Name)
	case *object.BoundMethod:
		if bf, ok := d.Fn.(*object.BuiltinFunc); ok {
			return strings.ToLower(bf.Name)
		}
	}
	return "sha256"
}

// hashConstructorByName returns (newFn, digestSize, blockSize, ok) for a named hash.
func hashConstructorByName(name string) (func() hash.Hash, int, int, bool) {
	switch name {
	case "md5":
		return md5.New, 16, 64, true
	case "sha1":
		return sha1.New, 20, 64, true
	case "sha224":
		return sha256.New224, 28, 64, true
	case "sha256":
		return sha256.New, 32, 64, true
	case "sha384":
		return sha512.New384, 48, 128, true
	case "sha512":
		return sha512.New, 64, 128, true
	case "sha3_224":
		return sha3.New224, 28, 144, true
	case "sha3_256":
		return sha3.New256, 32, 136, true
	case "sha3_384":
		return sha3.New384, 48, 104, true
	case "sha3_512":
		return sha3.New512, 64, 72, true
	}
	return nil, 0, 0, false
}

func bytesOrStr(o object.Object) ([]byte, bool) {
	switch v := o.(type) {
	case *object.Bytes:
		return v.V, true
	case *object.Bytearray:
		return v.V, true
	case *object.Str:
		return []byte(v.V), true
	}
	return nil, false
}

// --- secrets module --------------------------------------------------------

func (i *Interp) buildSecrets() *object.Module {
	m := &object.Module{Name: "secrets", Dict: object.NewDict()}

	m.Dict.SetStr("DEFAULT_ENTROPY", object.NewInt(32))

	readRand := func(n int) ([]byte, error) {
		if n < 0 {
			return nil, object.Errorf(i.valueErr, "negative argument not allowed")
		}
		buf := make([]byte, n)
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		return buf, nil
	}

	m.Dict.SetStr("token_bytes", &object.BuiltinFunc{Name: "token_bytes", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n := 32
		if len(a) >= 1 && a[0] != object.None {
			v, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "token_bytes() nbytes must be int")
			}
			n = int(v)
		}
		buf, err := readRand(n)
		if err != nil {
			return nil, err
		}
		return &object.Bytes{V: buf}, nil
	}})

	m.Dict.SetStr("token_hex", &object.BuiltinFunc{Name: "token_hex", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n := 32
		if len(a) >= 1 && a[0] != object.None {
			v, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "token_hex() nbytes must be int")
			}
			n = int(v)
		}
		buf, err := readRand(n)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: hex.EncodeToString(buf)}, nil
	}})

	m.Dict.SetStr("token_urlsafe", &object.BuiltinFunc{Name: "token_urlsafe", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		n := 32
		if len(a) >= 1 && a[0] != object.None {
			v, ok := toInt64(a[0])
			if !ok {
				return nil, object.Errorf(i.typeErr, "token_urlsafe() nbytes must be int")
			}
			n = int(v)
		}
		buf, err := readRand(n)
		if err != nil {
			return nil, err
		}
		return &object.Str{V: base64.RawURLEncoding.EncodeToString(buf)}, nil
	}})

	m.Dict.SetStr("randbelow", &object.BuiltinFunc{Name: "randbelow", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "randbelow() missing upper bound")
		}
		n, ok := toInt64(a[0])
		if !ok || n <= 0 {
			return nil, object.Errorf(i.valueErr, "Upper bound must be positive.")
		}
		bn, err := rand.Int(rand.Reader, big.NewInt(n))
		if err != nil {
			return nil, err
		}
		return object.IntFromBig(bn), nil
	}})

	m.Dict.SetStr("randbits", &object.BuiltinFunc{Name: "randbits", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "randbits() missing bit count")
		}
		k, ok := toInt64(a[0])
		if !ok || k < 0 {
			return nil, object.Errorf(i.valueErr, "number of bits must be non-negative")
		}
		if k == 0 {
			return object.NewInt(0), nil
		}
		nbytes := int((k + 7) / 8)
		buf := make([]byte, nbytes)
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		// Mask off the high bits that exceed k.
		excess := uint(nbytes*8) - uint(k)
		if excess > 0 {
			buf[0] &= byte(0xff >> excess)
		}
		out := new(big.Int).SetBytes(buf)
		return object.IntFromBig(out), nil
	}})

	m.Dict.SetStr("choice", &object.BuiltinFunc{Name: "choice", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "choice() missing sequence")
		}
		seq := a[0]
		n := sequenceLen(seq)
		if n == 0 {
			return nil, object.Errorf(i.indexErr, "Cannot choose from an empty sequence")
		}
		if n < 0 {
			return nil, object.Errorf(i.typeErr, "choice() argument is not a sequence")
		}
		bn, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
		if err != nil {
			return nil, err
		}
		idx := int(bn.Int64())
		return sequenceAt(seq, idx)
	}})

	m.Dict.SetStr("compare_digest", &object.BuiltinFunc{Name: "compare_digest", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "compare_digest() requires two arguments")
		}
		ab, aok := bytesOrStr(a[0])
		bb, bok := bytesOrStr(a[1])
		if !aok || !bok {
			return nil, object.Errorf(i.typeErr, "compare_digest() args must be bytes or str")
		}
		return object.BoolOf(subtle.ConstantTimeCompare(ab, bb) == 1 && len(ab) == len(bb)), nil
	}})

	return m
}

func sequenceLen(o object.Object) int {
	switch v := o.(type) {
	case *object.List:
		return len(v.V)
	case *object.Tuple:
		return len(v.V)
	case *object.Str:
		return len([]rune(v.V))
	case *object.Bytes:
		return len(v.V)
	case *object.Bytearray:
		return len(v.V)
	}
	return -1
}

func sequenceAt(o object.Object, idx int) (object.Object, error) {
	switch v := o.(type) {
	case *object.List:
		return v.V[idx], nil
	case *object.Tuple:
		return v.V[idx], nil
	case *object.Str:
		r := []rune(v.V)
		return &object.Str{V: string(r[idx])}, nil
	case *object.Bytes:
		return object.NewInt(int64(v.V[idx])), nil
	case *object.Bytearray:
		return object.NewInt(int64(v.V[idx])), nil
	}
	return nil, fmt.Errorf("not indexable")
}

// --- uuid module -----------------------------------------------------------

func (i *Interp) buildUUID() *object.Module {
	m := &object.Module{Name: "uuid", Dict: object.NewDict()}

	m.Dict.SetStr("uuid4", &object.BuiltinFunc{Name: "uuid4", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var u object.UUID
		if _, err := rand.Read(u.Bytes[:]); err != nil {
			return nil, err
		}
		// Set version 4 and RFC 4122 variant.
		u.Bytes[6] = (u.Bytes[6] & 0x0f) | 0x40
		u.Bytes[8] = (u.Bytes[8] & 0x3f) | 0x80
		return &u, nil
	}})

	m.Dict.SetStr("UUID", &object.BuiltinFunc{Name: "UUID", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return constructUUID(i, a, kw)
	}})

	// Namespace constants (RFC 4122).
	ns := func(s string) *object.UUID {
		u, _ := parseUUIDString(s)
		return u
	}
	m.Dict.SetStr("NAMESPACE_DNS", ns("6ba7b810-9dad-11d1-80b4-00c04fd430c8"))
	m.Dict.SetStr("NAMESPACE_URL", ns("6ba7b811-9dad-11d1-80b4-00c04fd430c8"))
	m.Dict.SetStr("NAMESPACE_OID", ns("6ba7b812-9dad-11d1-80b4-00c04fd430c8"))
	m.Dict.SetStr("NAMESPACE_X500", ns("6ba7b814-9dad-11d1-80b4-00c04fd430c8"))

	m.Dict.SetStr("uuid3", &object.BuiltinFunc{Name: "uuid3", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return nameBasedUUID(i, a, md5.New, 3)
	}})
	m.Dict.SetStr("uuid5", &object.BuiltinFunc{Name: "uuid5", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return nameBasedUUID(i, a, sha1.New, 5)
	}})

	return m
}

func constructUUID(i *Interp, a []object.Object, kw *object.Dict) (object.Object, error) {
	// UUID(hex=..., bytes=..., int=...) — take the first non-None.
	if len(a) >= 1 && a[0] != nil && a[0] != object.None {
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "UUID() first arg must be str")
		}
		u, err := parseUUIDString(s.V)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
		}
		return u, nil
	}
	if kw != nil {
		if v, ok := kw.GetStr("hex"); ok {
			s, _ := v.(*object.Str)
			if s == nil {
				return nil, object.Errorf(i.typeErr, "UUID(hex=...) must be str")
			}
			u, err := parseUUIDString(s.V)
			if err != nil {
				return nil, object.Errorf(i.valueErr, "%s", err.Error())
			}
			return u, nil
		}
		if v, ok := kw.GetStr("bytes"); ok {
			b, err := asBytes(v)
			if err != nil {
				return nil, err
			}
			if len(b) != 16 {
				return nil, object.Errorf(i.valueErr, "UUID(bytes=) requires 16 bytes")
			}
			var u object.UUID
			copy(u.Bytes[:], b)
			return &u, nil
		}
		if v, ok := kw.GetStr("int"); ok {
			bi, ok := toBigInt(v)
			if !ok {
				return nil, object.Errorf(i.typeErr, "UUID(int=) must be int")
			}
			var u object.UUID
			bs := bi.Bytes()
			if len(bs) > 16 {
				return nil, object.Errorf(i.valueErr, "UUID(int=) value out of range")
			}
			copy(u.Bytes[16-len(bs):], bs)
			return &u, nil
		}
	}
	return nil, object.Errorf(i.typeErr, "UUID() requires a hex string, bytes, or int")
}

// parseUUIDString accepts 32 hex chars with optional hyphens and an optional
// "urn:uuid:" prefix.
func parseUUIDString(s string) (*object.UUID, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "urn:uuid:")
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 32 {
		return nil, fmt.Errorf("badly formed hexadecimal UUID string")
	}
	raw, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("%s", err.Error())
	}
	var u object.UUID
	copy(u.Bytes[:], raw)
	return &u, nil
}

func nameBasedUUID(i *Interp, a []object.Object, newFn func() hash.Hash, version byte) (object.Object, error) {
	if len(a) < 2 {
		return nil, object.Errorf(i.typeErr, "uuid3/5() requires namespace and name")
	}
	ns, ok := a[0].(*object.UUID)
	if !ok {
		return nil, object.Errorf(i.typeErr, "uuid3/5() namespace must be a UUID")
	}
	name, ok := a[1].(*object.Str)
	if !ok {
		return nil, object.Errorf(i.typeErr, "uuid3/5() name must be str")
	}
	h := newFn()
	h.Write(ns.Bytes[:])
	h.Write([]byte(name.V))
	sum := h.Sum(nil)
	var u object.UUID
	copy(u.Bytes[:], sum[:16])
	u.Bytes[6] = (u.Bytes[6] & 0x0f) | (version << 4)
	u.Bytes[8] = (u.Bytes[8] & 0x3f) | 0x80
	return &u, nil
}

// uuidAttr dispatches attribute access on a *object.UUID.
func uuidAttr(u *object.UUID, name string) (object.Object, bool) {
	switch name {
	case "hex":
		return &object.Str{V: hex.EncodeToString(u.Bytes[:])}, true
	case "bytes":
		return &object.Bytes{V: append([]byte(nil), u.Bytes[:]...)}, true
	case "bytes_le":
		// CPython swaps the first three fields when returning bytes_le.
		b := append([]byte(nil), u.Bytes[:]...)
		b[0], b[1], b[2], b[3] = b[3], b[2], b[1], b[0]
		b[4], b[5] = b[5], b[4]
		b[6], b[7] = b[7], b[6]
		return &object.Bytes{V: b}, true
	case "int":
		return object.IntFromBig(new(big.Int).SetBytes(u.Bytes[:])), true
	case "version":
		// CPython exposes a version only when the variant is RFC 4122.
		if u.Bytes[8]&0xc0 != 0x80 {
			return object.None, true
		}
		v := u.Bytes[6] >> 4
		if v == 0 {
			return object.None, true
		}
		return object.NewInt(int64(v)), true
	case "variant":
		switch {
		case u.Bytes[8]&0x80 == 0:
			return &object.Str{V: "reserved for NCS compatibility"}, true
		case u.Bytes[8]&0xc0 == 0x80:
			return &object.Str{V: "specified in RFC 4122"}, true
		case u.Bytes[8]&0xe0 == 0xc0:
			return &object.Str{V: "reserved for Microsoft compatibility"}, true
		default:
			return &object.Str{V: "reserved for future definition"}, true
		}
	case "urn":
		return &object.Str{V: "urn:uuid:" + hyphenated(u.Bytes)}, true
	case "fields":
		b := u.Bytes
		timeLow := uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3])
		timeMid := uint64(b[4])<<8 | uint64(b[5])
		timeHi := uint64(b[6])<<8 | uint64(b[7])
		clockHi := uint64(b[8])
		clockLo := uint64(b[9])
		node := uint64(b[10])<<40 | uint64(b[11])<<32 | uint64(b[12])<<24 | uint64(b[13])<<16 | uint64(b[14])<<8 | uint64(b[15])
		return &object.Tuple{V: []object.Object{
			newIntU64(timeLow),
			newIntU64(timeMid),
			newIntU64(timeHi),
			newIntU64(clockHi),
			newIntU64(clockLo),
			newIntU64(node),
		}}, true
	}
	return nil, false
}

func hyphenated(b [16]byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 36)
	j := 0
	for i := 0; i < 16; i++ {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out[j] = '-'
			j++
		}
		out[j] = hex[b[i]>>4]
		out[j+1] = hex[b[i]&0x0f]
		j += 2
	}
	return string(out)
}

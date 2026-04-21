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

	"github.com/tamnd/goipy/object"
)

// --- binascii module -------------------------------------------------------

func (i *Interp) buildBinascii() *object.Module {
	m := &object.Module{Name: "binascii", Dict: object.NewDict()}

	hexlify := &object.BuiltinFunc{Name: "hexlify", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "hexlify() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		return &object.Bytes{V: []byte(hex.EncodeToString(data))}, nil
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
			return nil, object.Errorf(i.valueErr, "%s", err.Error())
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
				if b, ok := v.(*object.Bool); ok {
					newline = b.V
				}
			}
		}
		enc := base64.StdEncoding.EncodeToString(data)
		if newline {
			enc += "\n"
		}
		return &object.Bytes{V: []byte(enc)}, nil
	}})

	m.Dict.SetStr("a2b_base64", &object.BuiltinFunc{Name: "a2b_base64", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) == 0 {
			return nil, object.Errorf(i.typeErr, "a2b_base64() missing data")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, err
		}
		// CPython's a2b_base64 accepts with or without trailing newline and
		// is lenient about whitespace.
		src := strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
				return -1
			}
			return r
		}, string(data))
		out, err := base64.StdEncoding.DecodeString(src)
		if err != nil {
			// Try raw (no padding).
			out, err = base64.RawStdEncoding.DecodeString(src)
			if err != nil {
				return nil, object.Errorf(i.valueErr, "%s", err.Error())
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

	return m
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
			switch d := dig.(type) {
			case *object.Str:
				name = strings.ToLower(d.V)
			case *object.BuiltinFunc:
				name = strings.ToLower(d.Name)
			}
		}
		newFn, size, ok := hashConstructorByName(name)
		if !ok {
			return nil, object.Errorf(i.valueErr, "unsupported digestmod %q", name)
		}
		mac := hmac.New(newFn, key)
		if len(msg) > 0 {
			mac.Write(msg)
		}
		return &object.Hasher{Name: "hmac-" + name, Size: size, State: mac}, nil
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
		name := "sha256"
		switch d := a[2].(type) {
		case *object.Str:
			name = strings.ToLower(d.V)
		case *object.BuiltinFunc:
			name = strings.ToLower(d.Name)
		}
		newFn, _, ok := hashConstructorByName(name)
		if !ok {
			return nil, object.Errorf(i.valueErr, "unsupported digest %q", name)
		}
		mac := hmac.New(newFn, key)
		mac.Write(msg)
		return &object.Bytes{V: mac.Sum(nil)}, nil
	}})

	return m
}

func hashConstructorByName(name string) (func() hash.Hash, int, bool) {
	switch name {
	case "md5":
		return md5.New, 16, true
	case "sha1":
		return sha1.New, 20, true
	case "sha224":
		return sha256.New224, 28, true
	case "sha256":
		return sha256.New, 32, true
	case "sha384":
		return sha512.New384, 48, true
	case "sha512":
		return sha512.New, 64, true
	}
	return nil, 0, false
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

	readRand := func(n int) ([]byte, error) {
		if n < 0 {
			return nil, object.Errorf(i.valueErr, "negative argument")
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
			return nil, object.Errorf(i.valueErr, "randbelow() upper bound must be positive int")
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
			return nil, object.Errorf(i.valueErr, "randbits() k must be non-negative")
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
		if n <= 0 {
			return nil, object.Errorf(i.valueErr, "choice() empty sequence")
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

package vm

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/tamnd/goipy/object"
)

// buildUuid constructs the uuid module matching CPython 3.14's API.
func (i *Interp) buildUuid() *object.Module {
	m := &object.Module{Name: "uuid", Dict: object.NewDict()}

	// ── Variant constants ─────────────────────────────────────────────────────

	m.Dict.SetStr("RESERVED_NCS", &object.Str{V: "reserved for NCS compatibility"})
	m.Dict.SetStr("RFC_4122", &object.Str{V: "specified in RFC 4122"})
	m.Dict.SetStr("RESERVED_MICROSOFT", &object.Str{V: "reserved for Microsoft compatibility"})
	m.Dict.SetStr("RESERVED_FUTURE", &object.Str{V: "reserved for future definition"})

	// ── SafeUUID ──────────────────────────────────────────────────────────────

	safeUUIDCls := &object.Class{Name: "SafeUUID", Dict: object.NewDict()}

	makeSafeMember := func(name string, val object.Object) *object.Instance {
		inst := &object.Instance{Class: safeUUIDCls, Dict: object.NewDict()}
		inst.Dict.SetStr("value", val)
		inst.Dict.SetStr("name", &object.Str{V: name})
		return inst
	}
	safeUUIDCls.Dict.SetStr("safe", makeSafeMember("safe", object.NewInt(0)))
	safeUUIDCls.Dict.SetStr("unsafe", makeSafeMember("unsafe", object.NewInt(-1)))
	safeUUIDCls.Dict.SetStr("unknown", makeSafeMember("unknown", object.None))

	safeUUIDCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "SafeUUID"}, nil
		}
		if n, ok2 := inst.Dict.GetStr("name"); ok2 {
			if s, ok3 := n.(*object.Str); ok3 {
				return &object.Str{V: "SafeUUID." + s.V}, nil
			}
		}
		return &object.Str{V: "SafeUUID"}, nil
	}})
	safeUUIDCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "<SafeUUID>"}, nil
		}
		nameStr, valStr := "unknown", "None"
		if n, ok2 := inst.Dict.GetStr("name"); ok2 {
			if s, ok3 := n.(*object.Str); ok3 {
				nameStr = s.V
			}
		}
		if v, ok2 := inst.Dict.GetStr("value"); ok2 {
			valStr = object.Repr(v)
		}
		return &object.Str{V: fmt.Sprintf("<SafeUUID.%s: %s>", nameStr, valStr)}, nil
	}})

	m.Dict.SetStr("SafeUUID", safeUUIDCls)

	safeUnknown, _ := safeUUIDCls.Dict.GetStr("unknown")

	// ── UUID helpers ──────────────────────────────────────────────────────────

	// uuidToBytes converts a big.Int to a 16-byte big-endian UUID.
	uuidToBytes := func(n *big.Int) [16]byte {
		var b [16]byte
		nb := n.Bytes()
		if len(nb) <= 16 {
			copy(b[16-len(nb):], nb)
		}
		return b
	}

	// uuidStr formats 16 bytes as "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx".
	uuidStr := func(b [16]byte) string {
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			binary.BigEndian.Uint32(b[0:4]),
			binary.BigEndian.Uint16(b[4:6]),
			binary.BigEndian.Uint16(b[6:8]),
			binary.BigEndian.Uint16(b[8:10]),
			b[10:16])
	}

	// computeVariant returns the variant string for byte 8 of the UUID.
	computeVariant := func(b8 byte) string {
		switch {
		case b8&0x80 == 0:
			return "reserved for NCS compatibility"
		case b8&0xc0 == 0x80:
			return "specified in RFC 4122"
		case b8&0xe0 == 0xc0:
			return "reserved for Microsoft compatibility"
		default:
			return "reserved for future definition"
		}
	}

	// populateUUID fills all UUID properties into inst.Dict from a 128-bit int.
	populateUUID := func(inst *object.Instance, n *big.Int, isSafe object.Object) {
		b := uuidToBytes(n)

		// int
		nCopy := new(big.Int).Set(n)
		inst.Dict.SetStr("int", &object.Int{V: *nCopy})

		// hex
		inst.Dict.SetStr("hex", &object.Str{V: fmt.Sprintf("%032x", n)})

		// bytes (big-endian)
		bcopy := make([]byte, 16)
		copy(bcopy, b[:])
		inst.Dict.SetStr("bytes", &object.Bytes{V: bcopy})

		// bytes_le (first 3 components little-endian, last 2 big-endian)
		ble := make([]byte, 16)
		copy(ble, b[:])
		ble[0], ble[1], ble[2], ble[3] = b[3], b[2], b[1], b[0]
		ble[4], ble[5] = b[5], b[4]
		ble[6], ble[7] = b[7], b[6]
		inst.Dict.SetStr("bytes_le", &object.Bytes{V: ble})

		// fields
		timelow := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
		timemid := uint16(b[4])<<8 | uint16(b[5])
		timehiver := uint16(b[6])<<8 | uint16(b[7])
		cshivar := b[8]
		cslow := b[9]
		node := uint64(b[10])<<40 | uint64(b[11])<<32 | uint64(b[12])<<24 |
			uint64(b[13])<<16 | uint64(b[14])<<8 | uint64(b[15])
		inst.Dict.SetStr("fields", &object.Tuple{V: []object.Object{
			object.NewInt(int64(timelow)),
			object.NewInt(int64(timemid)),
			object.NewInt(int64(timehiver)),
			object.NewInt(int64(cshivar)),
			object.NewInt(int64(cslow)),
			object.NewInt(int64(node)),
		}})
		inst.Dict.SetStr("time_low", object.NewInt(int64(timelow)))
		inst.Dict.SetStr("time_mid", object.NewInt(int64(timemid)))
		inst.Dict.SetStr("time_hi_version", object.NewInt(int64(timehiver)))
		inst.Dict.SetStr("clock_seq_hi_variant", object.NewInt(int64(cshivar)))
		inst.Dict.SetStr("clock_seq_low", object.NewInt(int64(cslow)))
		inst.Dict.SetStr("node", object.NewInt(int64(node)))

		// time: ((int>>64)&0x0fff)<<48 | ((int>>80)&0xffff)<<32 | int>>96
		timeVal := new(big.Int)
		tmp := new(big.Int)
		// (n >> 64) & 0x0fff
		tmp.Rsh(n, 64)
		tmp.And(tmp, big.NewInt(0x0fff))
		timeVal.Lsh(tmp, 48)
		// (n >> 80) & 0xffff
		tmp2 := new(big.Int)
		tmp2.Rsh(n, 80)
		tmp2.And(tmp2, big.NewInt(0xffff))
		tmp2.Lsh(tmp2, 32)
		timeVal.Or(timeVal, tmp2)
		// n >> 96
		tmp3 := new(big.Int)
		tmp3.Rsh(n, 96)
		timeVal.Or(timeVal, tmp3)
		inst.Dict.SetStr("time", &object.Int{V: *timeVal})

		// clock_seq: (n>>48)&0x3fff
		cs := new(big.Int)
		cs.Rsh(n, 48)
		cs.And(cs, big.NewInt(0x3fff))
		inst.Dict.SetStr("clock_seq", &object.Int{V: *cs})

		// variant
		inst.Dict.SetStr("variant", &object.Str{V: computeVariant(cshivar)})

		// version (only for RFC 4122)
		if cshivar&0xc0 == 0x80 {
			inst.Dict.SetStr("version", object.NewInt(int64(b[6]>>4)))
		} else {
			inst.Dict.SetStr("version", object.None)
		}

		// urn
		inst.Dict.SetStr("urn", &object.Str{V: "urn:uuid:" + uuidStr(b)})

		// is_safe
		if isSafe == nil {
			isSafe = safeUnknown
		}
		inst.Dict.SetStr("is_safe", isSafe)
	}

	// parseUUIDHex parses a UUID hex string (with or without hyphens, braces, urn prefix).
	reUUIDHex := regexp.MustCompile(`^[0-9a-f]{32}$`)
	parseUUIDHex := func(s string) (*big.Int, error) {
		s = strings.TrimSpace(s)
		s = strings.ToLower(s)
		s = strings.TrimPrefix(s, "urn:uuid:")
		s = strings.ReplaceAll(s, "-", "")
		s = strings.ReplaceAll(s, "{", "")
		s = strings.ReplaceAll(s, "}", "")
		if !reUUIDHex.MatchString(s) {
			return nil, fmt.Errorf("badly formed hexadecimal UUID string")
		}
		n := new(big.Int)
		n.SetString(s, 16)
		return n, nil
	}

	// ── UUID class ────────────────────────────────────────────────────────────

	uuidCls := &object.Class{Name: "UUID", Dict: object.NewDict()}

	uuidCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}

		// Collect arguments
		var hexArg, bytesArg, bytesLeArg, fieldsArg, intArg, versionArg object.Object
		isSafe := safeUnknown

		// First positional arg is hex string
		if len(a) > 1 {
			hexArg = a[1]
		}
		if kw != nil {
			if v, ok2 := kw.GetStr("hex"); ok2 {
				hexArg = v
			}
			if v, ok2 := kw.GetStr("bytes"); ok2 {
				bytesArg = v
			}
			if v, ok2 := kw.GetStr("bytes_le"); ok2 {
				bytesLeArg = v
			}
			if v, ok2 := kw.GetStr("fields"); ok2 {
				fieldsArg = v
			}
			if v, ok2 := kw.GetStr("int"); ok2 {
				intArg = v
			}
			if v, ok2 := kw.GetStr("version"); ok2 {
				versionArg = v
			}
			if v, ok2 := kw.GetStr("is_safe"); ok2 {
				isSafe = v
			}
		}

		// Count provided arguments
		var n *big.Int
		provided := 0
		for _, v := range []object.Object{hexArg, bytesArg, bytesLeArg, fieldsArg, intArg} {
			if v != nil {
				provided++
			}
		}
		if provided != 1 {
			return nil, object.Errorf(i.typeErr, "one of the hex, bytes, bytes_le, fields, or int arguments must be given")
		}

		var parseErr error
		switch {
		case hexArg != nil:
			s, ok2 := hexArg.(*object.Str)
			if !ok2 {
				return nil, object.Errorf(i.typeErr, "hex must be a string")
			}
			n, parseErr = parseUUIDHex(s.V)
			if parseErr != nil {
				return nil, object.Errorf(i.valueErr, "%s", parseErr.Error())
			}

		case bytesArg != nil:
			b, ok2 := bytesArg.(*object.Bytes)
			if !ok2 || len(b.V) != 16 {
				return nil, object.Errorf(i.valueErr, "bytes is not a 16-char string")
			}
			n = new(big.Int)
			n.SetBytes(b.V)

		case bytesLeArg != nil:
			b, ok2 := bytesLeArg.(*object.Bytes)
			if !ok2 || len(b.V) != 16 {
				return nil, object.Errorf(i.valueErr, "bytes_le is not a 16-char string")
			}
			// Convert little-endian components to big-endian
			be := make([]byte, 16)
			copy(be, b.V)
			be[0], be[1], be[2], be[3] = b.V[3], b.V[2], b.V[1], b.V[0]
			be[4], be[5] = b.V[5], b.V[4]
			be[6], be[7] = b.V[7], b.V[6]
			n = new(big.Int)
			n.SetBytes(be)

		case fieldsArg != nil:
			tup, ok2 := fieldsArg.(*object.Tuple)
			if !ok2 || len(tup.V) != 6 {
				return nil, object.Errorf(i.valueErr, "fields is not a 6-tuple")
			}
			f := make([]int64, 6)
			for idx, fv := range tup.V {
				fi, ok3 := toInt64(fv)
				if !ok3 {
					return nil, object.Errorf(i.valueErr, "field %d must be int", idx)
				}
				f[idx] = fi
			}
			// fields: time_low(32), time_mid(16), time_hi(16), clk_hi(8), clk_lo(8), node(48)
			n = new(big.Int)
			n.Or(n, new(big.Int).Lsh(big.NewInt(f[0]), 96))
			n.Or(n, new(big.Int).Lsh(big.NewInt(f[1]), 80))
			n.Or(n, new(big.Int).Lsh(big.NewInt(f[2]), 64))
			n.Or(n, new(big.Int).Lsh(big.NewInt(f[3]), 56))
			n.Or(n, new(big.Int).Lsh(big.NewInt(f[4]), 48))
			n.Or(n, big.NewInt(f[5]))

		case intArg != nil:
			iv, ok2 := intArg.(*object.Int)
			if !ok2 {
				return nil, object.Errorf(i.typeErr, "int must be int")
			}
			n = new(big.Int).Set(&iv.V)
		}

		// Apply version if specified
		if versionArg != nil && versionArg != object.None {
			ver, ok2 := toInt64(versionArg)
			if !ok2 || ver < 1 || ver > 5 {
				return nil, object.Errorf(i.valueErr, "illegal version number")
			}
			// Set RFC 4122 variant
			mask1 := new(big.Int).Lsh(big.NewInt(0xc000), 48)
			n.AndNot(n, mask1)
			n.Or(n, new(big.Int).Lsh(big.NewInt(0x8000), 48))
			// Set version
			mask2 := new(big.Int).Lsh(big.NewInt(0xf000), 64)
			n.AndNot(n, mask2)
			n.Or(n, new(big.Int).Lsh(big.NewInt(ver), 76))
		}

		populateUUID(inst, n, isSafe)
		return object.None, nil
	}})

	uuidCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: ""}, nil
		}
		if h, ok2 := inst.Dict.GetStr("hex"); ok2 {
			if s, ok3 := h.(*object.Str); ok3 && len(s.V) == 32 {
				return &object.Str{V: s.V[:8] + "-" + s.V[8:12] + "-" + s.V[12:16] + "-" + s.V[16:20] + "-" + s.V[20:]}, nil
			}
		}
		return &object.Str{V: ""}, nil
	}})

	uuidCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Str{V: "UUID('')"}, nil
		}
		if h, ok2 := inst.Dict.GetStr("hex"); ok2 {
			if s, ok3 := h.(*object.Str); ok3 && len(s.V) == 32 {
				h32 := s.V
				dash := h32[:8] + "-" + h32[8:12] + "-" + h32[12:16] + "-" + h32[16:20] + "-" + h32[20:]
				return &object.Str{V: "UUID('" + dash + "')"}, nil
			}
		}
		return &object.Str{V: "UUID('')"}, nil
	}})

	uuidCls.Dict.SetStr("__hash__", &object.BuiltinFunc{Name: "__hash__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.NewInt(0), nil
		}
		if v, ok2 := inst.Dict.GetStr("int"); ok2 {
			if iv, ok3 := v.(*object.Int); ok3 {
				cp := new(big.Int).Set(&iv.V)
				return &object.Int{V: *cp}, nil
			}
		}
		return object.NewInt(0), nil
	}})

	cmpUUID := func(a []object.Object) (ai, bi *big.Int, ok bool) {
		if len(a) < 2 {
			return nil, nil, false
		}
		ia, ok1 := a[0].(*object.Instance)
		ib, ok2 := a[1].(*object.Instance)
		if !ok1 || !ok2 {
			return nil, nil, false
		}
		va, ok3 := ia.Dict.GetStr("int")
		vb, ok4 := ib.Dict.GetStr("int")
		if !ok3 || !ok4 {
			return nil, nil, false
		}
		iva, ok5 := va.(*object.Int)
		ivb, ok6 := vb.(*object.Int)
		if !ok5 || !ok6 {
			return nil, nil, false
		}
		return &iva.V, &ivb.V, true
	}

	uuidCls.Dict.SetStr("__eq__", &object.BuiltinFunc{Name: "__eq__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		ai, bi, ok := cmpUUID(a)
		if !ok {
			return object.False, nil
		}
		if ai.Cmp(bi) == 0 {
			return object.True, nil
		}
		return object.False, nil
	}})
	uuidCls.Dict.SetStr("__ne__", &object.BuiltinFunc{Name: "__ne__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		ai, bi, ok := cmpUUID(a)
		if !ok {
			return object.True, nil
		}
		if ai.Cmp(bi) != 0 {
			return object.True, nil
		}
		return object.False, nil
	}})
	uuidCls.Dict.SetStr("__lt__", &object.BuiltinFunc{Name: "__lt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		ai, bi, ok := cmpUUID(a)
		if !ok {
			return object.False, nil
		}
		if ai.Cmp(bi) < 0 {
			return object.True, nil
		}
		return object.False, nil
	}})
	uuidCls.Dict.SetStr("__gt__", &object.BuiltinFunc{Name: "__gt__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		ai, bi, ok := cmpUUID(a)
		if !ok {
			return object.False, nil
		}
		if ai.Cmp(bi) > 0 {
			return object.True, nil
		}
		return object.False, nil
	}})
	uuidCls.Dict.SetStr("__le__", &object.BuiltinFunc{Name: "__le__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		ai, bi, ok := cmpUUID(a)
		if !ok {
			return object.False, nil
		}
		if ai.Cmp(bi) <= 0 {
			return object.True, nil
		}
		return object.False, nil
	}})
	uuidCls.Dict.SetStr("__ge__", &object.BuiltinFunc{Name: "__ge__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		ai, bi, ok := cmpUUID(a)
		if !ok {
			return object.False, nil
		}
		if ai.Cmp(bi) >= 0 {
			return object.True, nil
		}
		return object.False, nil
	}})

	m.Dict.SetStr("UUID", uuidCls)

	// helper: make a UUID object from a big.Int
	newUUID := func(n *big.Int) *object.Instance {
		inst := &object.Instance{Class: uuidCls, Dict: object.NewDict()}
		populateUUID(inst, n, safeUnknown)
		return inst
	}

	// ── Namespace constants ───────────────────────────────────────────────────

	for name, hex := range map[string]string{
		"NAMESPACE_DNS":  "6ba7b8109dad11d180b400c04fd430c8",
		"NAMESPACE_URL":  "6ba7b8119dad11d180b400c04fd430c8",
		"NAMESPACE_OID":  "6ba7b8129dad11d180b400c04fd430c8",
		"NAMESPACE_X500": "6ba7b8149dad11d180b400c04fd430c8",
	} {
		n := new(big.Int)
		n.SetString(hex, 16)
		m.Dict.SetStr(name, newUUID(n))
	}

	// ── uuid-generation functions ─────────────────────────────────────────────

	// uuid4 — random
	m.Dict.SetStr("uuid4", &object.BuiltinFunc{Name: "uuid4", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			return nil, object.Errorf(i.osErr, "uuid4: %v", err)
		}
		b[6] = (b[6] & 0x0f) | 0x40 // version 4
		b[8] = (b[8] & 0x3f) | 0x80 // variant RFC 4122
		n := new(big.Int)
		n.SetBytes(b[:])
		return newUUID(n), nil
	}})

	// uuid3 — MD5 based
	getNamespaceBytes := func(ns object.Object) ([]byte, bool) {
		inst, ok := ns.(*object.Instance)
		if !ok {
			return nil, false
		}
		b, ok2 := inst.Dict.GetStr("bytes")
		if !ok2 {
			return nil, false
		}
		bv, ok3 := b.(*object.Bytes)
		if !ok3 {
			return nil, false
		}
		return bv.V, true
	}

	m.Dict.SetStr("uuid3", &object.BuiltinFunc{Name: "uuid3", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "uuid3() requires 2 arguments")
		}
		nsBytes, ok := getNamespaceBytes(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "namespace must be a UUID")
		}
		name := ""
		switch v := a[1].(type) {
		case *object.Str:
			name = v.V
		case *object.Bytes:
			name = string(v.V)
		}
		data := append(nsBytes, []byte(name)...)
		sum := md5.Sum(data)
		h := sum[:]
		h[6] = (h[6] & 0x0f) | 0x30 // version 3
		h[8] = (h[8] & 0x3f) | 0x80 // variant RFC 4122
		n := new(big.Int)
		n.SetBytes(h)
		return newUUID(n), nil
	}})

	// uuid5 — SHA-1 based
	m.Dict.SetStr("uuid5", &object.BuiltinFunc{Name: "uuid5", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "uuid5() requires 2 arguments")
		}
		nsBytes, ok := getNamespaceBytes(a[0])
		if !ok {
			return nil, object.Errorf(i.typeErr, "namespace must be a UUID")
		}
		name := ""
		switch v := a[1].(type) {
		case *object.Str:
			name = v.V
		case *object.Bytes:
			name = string(v.V)
		}
		data := append(nsBytes, []byte(name)...)
		sum := sha1.Sum(data)
		h := sum[:16] // take first 16 bytes
		h[6] = (h[6] & 0x0f) | 0x50 // version 5
		h[8] = (h[8] & 0x3f) | 0x80 // variant RFC 4122
		n := new(big.Int)
		n.SetBytes(h)
		return newUUID(n), nil
	}})

	// uuid1 — time-based
	m.Dict.SetStr("uuid1", &object.BuiltinFunc{Name: "uuid1", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		// UUID epoch offset: Oct 15, 1582 to Jan 1, 1970 in 100-ns intervals
		const uuidEpoch = int64(0x01b21dd213814000)
		t := time.Now().UnixNano()/100 + uuidEpoch

		timelow := uint32(t & 0xffffffff)
		timemid := uint16((t >> 32) & 0xffff)
		timehiver := uint16((t>>48)&0x0fff) | 0x1000 // version 1

		// Random clock sequence
		var csBytes [2]byte
		rand.Read(csBytes[:])
		clockSeq := (uint16(csBytes[0]&0x3f) << 8) | uint16(csBytes[1])
		clockSeq |= 0x8000 // variant RFC 4122

		// Random node
		var nodeBytes [6]byte
		rand.Read(nodeBytes[:])
		nodeBytes[0] |= 0x01 // set multicast bit for random node

		var b [16]byte
		binary.BigEndian.PutUint32(b[0:], timelow)
		binary.BigEndian.PutUint16(b[4:], timemid)
		binary.BigEndian.PutUint16(b[6:], timehiver)
		binary.BigEndian.PutUint16(b[8:], clockSeq)
		copy(b[10:], nodeBytes[:])

		n := new(big.Int)
		n.SetBytes(b[:])
		return newUUID(n), nil
	}})

	// getnode — returns an int (MAC address or random)
	m.Dict.SetStr("getnode", &object.BuiltinFunc{Name: "getnode", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		var b [6]byte
		rand.Read(b[:])
		b[0] |= 0x01
		node := uint64(b[0])<<40 | uint64(b[1])<<32 | uint64(b[2])<<24 |
			uint64(b[3])<<16 | uint64(b[4])<<8 | uint64(b[5])
		return object.NewInt(int64(node)), nil
	}})

	return m
}

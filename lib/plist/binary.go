package plist

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"time"
	"unicode/utf16"
)

// Apple's reference epoch: Jan 1 2001 00:00:00 UTC
var appleEpoch = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

// ── Binary plist writer ───────────────────────────────────────────────────────
//
// Strategy: two passes.
//   Pass 1 (flatten): DFS-walk the value tree, assign each object an integer
//                     index. The root gets index 0. Each container stores the
//                     indices of its elements.
//   Pass 2 (emit):    Serialize each object in index order, recording its byte
//                     offset for the offset table.

type bplistObj struct {
	v        Value    // the actual value (leaf or container)
	children []int    // child indices (for array/dict)
	keys     []int    // key indices (for dict)
	sortKeys bool     // applies only to dict
}

type binaryWriter struct {
	objects  []bplistObj
	idxMap   []int  // maps flat index → object-list index (identity for now)
	sortKeys bool
}

func dumpBinary(v Value, sortKeys bool) ([]byte, error) {
	w := &binaryWriter{sortKeys: sortKeys}
	w.flatten(v)

	n := len(w.objects)
	refSize := refSizeFor(n)

	// Emit all objects, track offsets.
	var buf bytes.Buffer
	buf.WriteString("bplist00")
	offsets := make([]int, n)
	for idx := range w.objects {
		offsets[idx] = buf.Len()
		if err := w.emit(&buf, idx, refSize); err != nil {
			return nil, err
		}
	}

	// Offset table.
	offsetTableStart := buf.Len()
	offSize := offsetTableBytes(offsetTableStart)
	for _, off := range offsets {
		writeUintN(&buf, uint64(off), offSize)
	}

	// Trailer (32 bytes).
	trailer := make([]byte, 32)
	trailer[6] = byte(offSize)
	trailer[7] = byte(refSize)
	binary.BigEndian.PutUint64(trailer[8:], uint64(n))
	binary.BigEndian.PutUint64(trailer[16:], 0) // top object = 0
	binary.BigEndian.PutUint64(trailer[24:], uint64(offsetTableStart))
	buf.Write(trailer)
	return buf.Bytes(), nil
}

// flatten assigns indices to all objects via BFS so that the root is at index 0.
func (w *binaryWriter) flatten(v Value) int {
	idx := len(w.objects)
	w.objects = append(w.objects, bplistObj{v: v, sortKeys: w.sortKeys})

	switch x := v.(type) {
	case []interface{}:
		children := make([]int, len(x))
		for ci, child := range x {
			children[ci] = w.flatten(child)
		}
		w.objects[idx].children = children
	case map[string]interface{}:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		if w.sortKeys {
			sort.Strings(keys)
		}
		kIdxs := make([]int, len(keys))
		vIdxs := make([]int, len(keys))
		for ki, k := range keys {
			kIdxs[ki] = w.flatten(k)
			vIdxs[ki] = w.flatten(x[k])
		}
		w.objects[idx].keys = kIdxs
		w.objects[idx].children = vIdxs
	}
	return idx
}

func (w *binaryWriter) emit(buf *bytes.Buffer, idx int, refSize int) error {
	obj := w.objects[idx]
	v := obj.v
	switch x := v.(type) {
	case bool:
		if x {
			buf.WriteByte(0x09)
		} else {
			buf.WriteByte(0x08)
		}
	case int64:
		emitInt(buf, x)
	case uint64:
		if x <= math.MaxInt64 {
			emitInt(buf, int64(x))
		} else {
			buf.WriteByte(0x13)
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, x)
			buf.Write(b)
		}
	case float64:
		buf.WriteByte(0x23)
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, math.Float64bits(x))
		buf.Write(b)
	case time.Time:
		buf.WriteByte(0x33)
		secs := x.UTC().Sub(appleEpoch).Seconds()
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, math.Float64bits(secs))
		buf.Write(b)
	case []byte:
		writeCountTag(buf, 0x40, len(x))
		buf.Write(x)
	case string:
		if isASCII(x) {
			writeCountTag(buf, 0x50, len(x))
			buf.WriteString(x)
		} else {
			u16 := utf16.Encode([]rune(x))
			writeCountTag(buf, 0x60, len(u16))
			b := make([]byte, len(u16)*2)
			for i, r := range u16 {
				binary.BigEndian.PutUint16(b[i*2:], r)
			}
			buf.Write(b)
		}
	case UID:
		n := x.Data
		switch {
		case n <= 0xFF:
			buf.WriteByte(0x80)
			buf.WriteByte(byte(n))
		case n <= 0xFFFF:
			buf.WriteByte(0x81)
			b := make([]byte, 2)
			binary.BigEndian.PutUint16(b, uint16(n))
			buf.Write(b)
		case n <= 0xFFFFFFFF:
			buf.WriteByte(0x83)
			b := make([]byte, 4)
			binary.BigEndian.PutUint32(b, uint32(n))
			buf.Write(b)
		default:
			buf.WriteByte(0x87)
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, n)
			buf.Write(b)
		}
	case []interface{}:
		writeCountTag(buf, 0xA0, len(obj.children))
		for _, ci := range obj.children {
			writeUintN(buf, uint64(ci), refSize)
		}
	case map[string]interface{}:
		writeCountTag(buf, 0xD0, len(obj.keys))
		for _, ki := range obj.keys {
			writeUintN(buf, uint64(ki), refSize)
		}
		for _, vi := range obj.children {
			writeUintN(buf, uint64(vi), refSize)
		}
	case nil:
		return fmt.Errorf("plistlib: cannot serialize None")
	default:
		return fmt.Errorf("plistlib: unsupported type %T", v)
	}
	return nil
}

func emitInt(buf *bytes.Buffer, n int64) {
	switch {
	case n >= 0 && n <= 0xFF:
		buf.WriteByte(0x10)
		buf.WriteByte(byte(n))
	case n >= 0 && n <= 0xFFFF:
		buf.WriteByte(0x11)
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, uint16(n))
		buf.Write(b)
	case n >= 0 && n <= 0xFFFFFFFF:
		buf.WriteByte(0x12)
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(n))
		buf.Write(b)
	default:
		buf.WriteByte(0x13)
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(n))
		buf.Write(b)
	}
}

func writeCountTag(buf *bytes.Buffer, typeNibble byte, count int) {
	if count < 15 {
		buf.WriteByte(typeNibble | byte(count))
	} else {
		buf.WriteByte(typeNibble | 0x0F)
		emitInt(buf, int64(count))
	}
}

func writeUintN(buf *bytes.Buffer, v uint64, size int) {
	b := make([]byte, size)
	switch size {
	case 1:
		b[0] = byte(v)
	case 2:
		binary.BigEndian.PutUint16(b, uint16(v))
	case 4:
		binary.BigEndian.PutUint32(b, uint32(v))
	case 8:
		binary.BigEndian.PutUint64(b, v)
	}
	buf.Write(b)
}

func refSizeFor(n int) int {
	switch {
	case n <= 0xFF:
		return 1
	case n <= 0xFFFF:
		return 2
	default:
		return 4
	}
}

func offsetTableBytes(max int) int {
	switch {
	case max < 256:
		return 1
	case max < 65536:
		return 2
	default:
		return 4
	}
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 0x7E {
			return false
		}
	}
	return true
}

// ── Binary plist reader ───────────────────────────────────────────────────────

type binaryReader struct {
	data    []byte
	offsets []uint64
	objRefs int
}

func parseBinary(data []byte) (Value, error) {
	if len(data) < 40 {
		return nil, &InvalidFileError{Msg: "binary plist too short"}
	}
	if string(data[:8]) != "bplist00" {
		return nil, &InvalidFileError{Msg: "not a binary plist"}
	}
	trailer := data[len(data)-32:]
	offsetSize := int(trailer[6])
	objRefSize := int(trailer[7])
	numObjects := int(binary.BigEndian.Uint64(trailer[8:]))
	topObject := int(binary.BigEndian.Uint64(trailer[16:]))
	offsetTableOffset := int(binary.BigEndian.Uint64(trailer[24:]))

	if offsetSize < 1 || offsetSize > 8 || objRefSize < 1 || objRefSize > 8 {
		return nil, &InvalidFileError{Msg: "invalid binary plist trailer"}
	}

	r := &binaryReader{data: data, objRefs: objRefSize}
	r.offsets = make([]uint64, numObjects)
	tableEnd := offsetTableOffset + numObjects*offsetSize
	if tableEnd > len(data)-32 {
		return nil, &InvalidFileError{Msg: "offset table out of bounds"}
	}
	for i := 0; i < numObjects; i++ {
		off := offsetTableOffset + i*offsetSize
		r.offsets[i] = readUintN(data[off:], offsetSize)
	}
	return r.readObject(topObject, 0)
}

func readUintN(b []byte, size int) uint64 {
	switch size {
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(binary.BigEndian.Uint16(b))
	case 3:
		return uint64(b[0])<<16 | uint64(b[1])<<8 | uint64(b[2])
	case 4:
		return uint64(binary.BigEndian.Uint32(b))
	case 8:
		return binary.BigEndian.Uint64(b)
	}
	return 0
}

func (r *binaryReader) readObject(idx int, depth int) (Value, error) {
	if depth > 1000 {
		return nil, &InvalidFileError{Msg: "binary plist: nesting too deep"}
	}
	if idx < 0 || idx >= len(r.offsets) {
		return nil, &InvalidFileError{Msg: fmt.Sprintf("object index %d out of range", idx)}
	}
	off := int(r.offsets[idx])
	if off >= len(r.data) {
		return nil, &InvalidFileError{Msg: "object offset out of bounds"}
	}
	marker := r.data[off]
	typeNibble := marker >> 4
	info := marker & 0x0F

	switch typeNibble {
	case 0x0:
		switch info {
		case 0x08:
			return false, nil
		case 0x09:
			return true, nil
		case 0x00:
			return nil, nil
		}
		return nil, &InvalidFileError{Msg: fmt.Sprintf("unknown singleton 0x%02x", marker)}
	case 0x1:
		size := 1 << info
		if off+1+size > len(r.data) {
			return nil, &InvalidFileError{Msg: "int out of bounds"}
		}
		b := r.data[off+1 : off+1+size]
		if size == 16 {
			return binary.BigEndian.Uint64(b[8:]), nil
		}
		u := readUintN(b, size)
		if size < 8 {
			return int64(u), nil
		}
		if u > math.MaxInt64 {
			return u, nil
		}
		return int64(u), nil
	case 0x2:
		size := 1 << info
		if off+1+size > len(r.data) {
			return nil, &InvalidFileError{Msg: "real out of bounds"}
		}
		b := r.data[off+1 : off+1+size]
		if size == 4 {
			bits := binary.BigEndian.Uint32(b)
			return float64(math.Float32frombits(bits)), nil
		}
		bits := binary.BigEndian.Uint64(b)
		return math.Float64frombits(bits), nil
	case 0x3:
		if off+9 > len(r.data) {
			return nil, &InvalidFileError{Msg: "date out of bounds"}
		}
		bits := binary.BigEndian.Uint64(r.data[off+1 : off+9])
		secs := math.Float64frombits(bits)
		dur := time.Duration(secs * float64(time.Second))
		return appleEpoch.Add(dur).UTC(), nil
	case 0x4:
		count, skip, err := r.readCount(off, info)
		if err != nil {
			return nil, err
		}
		start := off + 1 + skip
		if start+count > len(r.data) {
			return nil, &InvalidFileError{Msg: "data out of bounds"}
		}
		b := make([]byte, count)
		copy(b, r.data[start:start+count])
		return b, nil
	case 0x5:
		count, skip, err := r.readCount(off, info)
		if err != nil {
			return nil, err
		}
		start := off + 1 + skip
		if start+count > len(r.data) {
			return nil, &InvalidFileError{Msg: "ASCII string out of bounds"}
		}
		return string(r.data[start : start+count]), nil
	case 0x6:
		count, skip, err := r.readCount(off, info)
		if err != nil {
			return nil, err
		}
		start := off + 1 + skip
		byteCount := count * 2
		if start+byteCount > len(r.data) {
			return nil, &InvalidFileError{Msg: "UTF-16 string out of bounds"}
		}
		u16 := make([]uint16, count)
		for i := 0; i < count; i++ {
			u16[i] = binary.BigEndian.Uint16(r.data[start+i*2:])
		}
		return string(utf16.Decode(u16)), nil
	case 0x8:
		size := int(info) + 1
		if off+1+size > len(r.data) {
			return nil, &InvalidFileError{Msg: "UID out of bounds"}
		}
		v := readUintN(r.data[off+1:], size)
		return UID{Data: v}, nil
	case 0xA:
		count, skip, err := r.readCount(off, info)
		if err != nil {
			return nil, err
		}
		start := off + 1 + skip
		items := make([]interface{}, count)
		for j := 0; j < count; j++ {
			refOff := start + j*r.objRefs
			if refOff+r.objRefs > len(r.data) {
				return nil, &InvalidFileError{Msg: "array ref out of bounds"}
			}
			refIdx := int(readUintN(r.data[refOff:], r.objRefs))
			v, err := r.readObject(refIdx, depth+1)
			if err != nil {
				return nil, err
			}
			items[j] = v
		}
		return items, nil
	case 0xD:
		count, skip, err := r.readCount(off, info)
		if err != nil {
			return nil, err
		}
		start := off + 1 + skip
		m := map[string]interface{}{}
		for j := 0; j < count; j++ {
			keyRefOff := start + j*r.objRefs
			valRefOff := start + (count+j)*r.objRefs
			if keyRefOff+r.objRefs > len(r.data) || valRefOff+r.objRefs > len(r.data) {
				return nil, &InvalidFileError{Msg: "dict ref out of bounds"}
			}
			keyIdx := int(readUintN(r.data[keyRefOff:], r.objRefs))
			valIdx := int(readUintN(r.data[valRefOff:], r.objRefs))
			kv, err := r.readObject(keyIdx, depth+1)
			if err != nil {
				return nil, err
			}
			vv, err := r.readObject(valIdx, depth+1)
			if err != nil {
				return nil, err
			}
			ks, ok := kv.(string)
			if !ok {
				return nil, &InvalidFileError{Msg: "dict key is not a string"}
			}
			m[ks] = vv
		}
		return m, nil
	}
	return nil, &InvalidFileError{Msg: fmt.Sprintf("unknown object type 0x%02x", marker)}
}

func (r *binaryReader) readCount(off int, info byte) (int, int, error) {
	if info != 0x0F {
		return int(info), 0, nil
	}
	if off+1 >= len(r.data) {
		return 0, 0, &InvalidFileError{Msg: "unexpected EOF reading count"}
	}
	cMarker := r.data[off+1]
	if cMarker>>4 != 0x1 {
		return 0, 0, &InvalidFileError{Msg: "expected int for count"}
	}
	size := 1 << (cMarker & 0x0F)
	if off+2+size > len(r.data) {
		return 0, 0, &InvalidFileError{Msg: "count out of bounds"}
	}
	count := int(readUintN(r.data[off+2:], size))
	return count, 1 + size, nil
}

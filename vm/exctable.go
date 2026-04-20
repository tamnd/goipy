package vm

// excEntry is a decoded row from co_exceptiontable.
// Offsets are in bytes (instruction offset × 2).
type excEntry struct {
	Start, End int // [Start, End) byte range this entry protects
	Target     int // target byte offset
	Depth      int // stack depth to restore
	Lasti      bool
}

// decodeExceptionTable decodes CPython's co_exceptiontable. The wire format
// matches the reference dis.py parser: a sequence of 4 varints per entry
// (start, length, target, dl), where each varint is 6-bit big-endian chunks
// with bit 6 (0x40) as the continuation flag. Bit 7 (0x80) marks the first
// byte of an entry and is ignored by the decoder.
func decodeExceptionTable(b []byte) []excEntry {
	var out []excEntry
	pos := 0
	for pos < len(b) {
		var start, length, target, dl int
		start, pos = readVarint(b, pos)
		length, pos = readVarint(b, pos)
		target, pos = readVarint(b, pos)
		dl, pos = readVarint(b, pos)
		out = append(out, excEntry{
			Start:  start * 2,
			End:    (start + length) * 2,
			Target: target * 2,
			Depth:  dl >> 1,
			Lasti:  dl&1 != 0,
		})
	}
	return out
}

func readVarint(b []byte, pos int) (int, int) {
	c := b[pos]
	pos++
	val := int(c & 63)
	for c&64 != 0 {
		c = b[pos]
		pos++
		val = (val << 6) | int(c&63)
	}
	return val, pos
}

// findHandler returns the matching exception table entry for a given byte
// offset, or nil.
func findHandler(table []excEntry, ipBytes int) *excEntry {
	// In CPython, only one entry matches at a time (non-overlapping
	// innermost first). We just scan linearly.
	for i := range table {
		e := &table[i]
		if ipBytes >= e.Start && ipBytes < e.End {
			return e
		}
	}
	return nil
}

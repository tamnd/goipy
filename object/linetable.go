package object

// LineForOffset returns the source line that contains the bytecode at ip
// (byte offset into Code.Bytecode). Returns 0 when the table is missing or
// the offset falls outside every entry.
//
// CPython 3.11+ encodes the location table (PEP 626 + PEP 657) as a stream
// of variable-length entries. Each entry starts with a byte with the high
// bit set:
//
//	1 kkkk lll
//
// where kkkk is a 4-bit code and lll is length-1 in "code units"
// (2 bytes each). Entries are consecutive and together span the whole
// bytecode.
//
// Codes:
//   0-9  — SHORT_FORM: line_delta=0, one data byte of column info
//   10/11/12 — ONE_LINE_{0,1,2}: line_delta = code-10, two data bytes
//   13   — NO_COLUMNS: one svarint line_delta
//   14   — LONG: svarint line_delta, varint end_line_delta, varint col, varint end_col
//   15   — NONE: no data (represents "no location available")
//
// We only need the line_delta; column bytes are skipped.
func (c *Code) LineForOffset(ip int) int {
	if c == nil || len(c.LineTable) == 0 {
		return c.FirstLineNo
	}
	line := c.FirstLineNo
	codeUnit := 0           // index in code units (one unit = 2 bytes)
	targetUnit := ip / 2
	p := 0
	for p < len(c.LineTable) {
		b := c.LineTable[p]
		if b&0x80 == 0 {
			// Desync — bail out with whatever line we have.
			return line
		}
		code := int((b >> 3) & 0x0F)
		length := int(b&0x07) + 1 // in code units
		p++

		var dLine int
		switch {
		case code == 15:
			// NONE — no line info; keep last known line.
			dLine = 0
		case code == 14:
			dLine = scanSvarint(c.LineTable, &p)
			// end_line_delta, col, end_col — discarded.
			scanVarint(c.LineTable, &p)
			scanVarint(c.LineTable, &p)
			scanVarint(c.LineTable, &p)
		case code == 13:
			dLine = scanSvarint(c.LineTable, &p)
		case code >= 10 && code <= 12:
			dLine = code - 10
			p += 2 // column / end column
		default: // 0-9
			dLine = 0
			p += 1 // column byte
		}
		line += dLine
		next := codeUnit + length
		if targetUnit >= codeUnit && targetUnit < next {
			if code == 15 {
				return 0
			}
			return line
		}
		codeUnit = next
	}
	return line
}

// scanVarint reads a base-64 varint (6 data bits per byte, continuation bit
// 0x40). Advances *p past the last byte consumed.
func scanVarint(buf []byte, p *int) int {
	if *p >= len(buf) {
		return 0
	}
	read := buf[*p]
	val := int(read & 0x3F)
	shift := 0
	for read&0x40 != 0 {
		*p++
		if *p >= len(buf) {
			return val
		}
		shift += 6
		read = buf[*p]
		val |= int(read&0x3F) << shift
	}
	*p++
	return val
}

// scanSvarint reads a signed varint (low bit is sign, rest is magnitude).
func scanSvarint(buf []byte, p *int) int {
	uval := scanVarint(buf, p)
	if uval&1 != 0 {
		return -(uval >> 1)
	}
	return uval >> 1
}

// Package bzip2 provides a pure-Go bzip2 encoder.
//
// Go's standard library (compress/bzip2) only supports decompression.
// This package implements the encoder side so goipy's bz2 stdlib module
// can compress without external dependencies. The output is a valid
// bzip2 stream accepted by any standard decoder, including
// compress/bzip2.
//
// Algorithms:
//
//   - BWT: doubling-suffix-array with counting-sort radix passes
//     (O(n log n)), adapted for cyclic rotations.
//   - MTF + RLE2: fused single pass emitting the RUNA/RUNB/shifted-MTF/
//     EOB alphabet.
//   - Huffman: classical two-way merge with tree-rotation length
//     limiting to 20 bits, canonical code assignment.
//   - Selectors: K-means-style iterative assignment of 50-symbol groups
//     to 2..6 Huffman tables.
package bzip2

const (
	blockMagic = uint64(0x314159265359)
	endMagic   = uint64(0x177245385090)
	groupSize  = 50
)

// Encode compresses data into a complete bzip2 stream at the given
// level (1..9). Level is clamped into range. The block size is
// level * 100000 bytes, matching the libbzip2 convention.
func Encode(data []byte, level int) []byte {
	if level < 1 {
		level = 1
	}
	if level > 9 {
		level = 9
	}
	w := &bitWriter{}
	w.writeBits('B', 8)
	w.writeBits('Z', 8)
	w.writeBits('h', 8)
	w.writeBits(uint64('0'+level), 8)

	if len(data) == 0 {
		w.writeBits(endMagic, 48)
		w.writeBits(0, 32)
		w.flushByte()
		return w.bytes()
	}

	blockSize := level * 100000
	var combined uint32
	for i := 0; i < len(data); i += blockSize {
		end := i + blockSize
		if end > len(data) {
			end = len(data)
		}
		block := data[i:end]
		crc := blockCRC(block)
		combined = ((combined << 1) | (combined >> 31)) ^ crc
		encodeBlock(w, block, crc)
	}

	w.writeBits(endMagic, 48)
	w.writeBits(uint64(combined), 32)
	w.flushByte()
	return w.bytes()
}

// encodeBlock emits one bzip2 block: header + symbol map + trees +
// encoded symbols.
func encodeBlock(w *bitWriter, block []byte, crc uint32) {
	rle := rle1(block)
	transformed, origPtr := bwt(rle)

	// Symbol map: track which of 0..255 actually appear.
	present := [256]bool{}
	for _, b := range rle {
		present[b] = true
	}
	var symMap []byte
	for i := range 256 {
		if present[i] {
			symMap = append(symMap, byte(i))
		}
	}
	if len(symMap) == 0 {
		symMap = append(symMap, 0)
	}

	syms, alphaSize := mtfAndRLE2(transformed, symMap)

	numTrees := pickNumTrees(len(syms))
	selectors, lensPerTree := refineSelectors(syms, alphaSize, numTrees, groupSize)

	codesPerTree := make([][]uint32, numTrees)
	for t := range codesPerTree {
		codesPerTree[t] = canonicalCodes(lensPerTree[t])
	}

	// ── Block header ─────────────────────────────────────────────────
	w.writeBits(blockMagic, 48)
	w.writeBits(uint64(crc), 32)
	w.writeBits(0, 1)               // not randomised
	w.writeBits(uint64(origPtr), 24)

	// Symbol map: 16-bit mapBig + up to 16 × 16-bit submaps.
	var mapBig uint16
	mapSmall := make([]uint16, 16)
	for _, b := range symMap {
		g := int(b) >> 4
		mapBig |= 1 << uint(15-g)
		mapSmall[g] |= 1 << uint(15-(int(b)&0xF))
	}
	w.writeBits(uint64(mapBig), 16)
	for g := 0; g < 16; g++ {
		if mapBig&(1<<uint(15-g)) != 0 {
			w.writeBits(uint64(mapSmall[g]), 16)
		}
	}

	numGroups := len(selectors)
	w.writeBits(uint64(numTrees), 3)
	w.writeBits(uint64(numGroups), 15)

	// Selectors are MTF-encoded over [0, numTrees), then written in
	// unary: N → N "1" bits followed by a "0".
	mtfList := make([]int, numTrees)
	for i := range mtfList {
		mtfList[i] = i
	}
	for _, sel := range selectors {
		pos := 0
		for pos < numTrees && mtfList[pos] != sel {
			pos++
		}
		// MTF: move sel to front.
		if pos > 0 {
			v := mtfList[pos]
			copy(mtfList[1:pos+1], mtfList[:pos])
			mtfList[0] = v
		}
		for range pos {
			w.writeBits(1, 1)
		}
		w.writeBits(0, 1)
	}

	// Emit each Huffman tree's code lengths.
	for t := 0; t < numTrees; t++ {
		lens := lensPerTree[t]
		cur := lens[0]
		w.writeBits(uint64(cur), 5)
		for _, l := range lens {
			for cur < l {
				w.writeBits(0b10, 2)
				cur++
			}
			for cur > l {
				w.writeBits(0b11, 2)
				cur--
			}
			w.writeBits(0, 1)
		}
	}

	// Emit the encoded symbols, switching tables per group.
	for g := 0; g < numGroups; g++ {
		t := selectors[g]
		lens := lensPerTree[t]
		codes := codesPerTree[t]
		start := g * groupSize
		end := start + groupSize
		if end > len(syms) {
			end = len(syms)
		}
		for _, s := range syms[start:end] {
			w.writeBits(uint64(codes[s]), uint(lens[s]))
		}
	}
}

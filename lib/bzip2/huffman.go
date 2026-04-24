package bzip2

import "sort"

const maxHuffmanLen = 20

// buildHuffman returns per-symbol code lengths, each in 1..maxLen.
// Zero-frequency symbols receive a real length too (bzip2 requires the
// tree to cover the entire alphabet).
func buildHuffman(freqs []int, maxLen int) []int {
	n := len(freqs)
	w := make([]int, n)
	for i, f := range freqs {
		if f == 0 {
			w[i] = 1
		} else {
			w[i] = f
		}
	}
	for {
		lens := huffmanLengths(w)
		over := 0
		for _, l := range lens {
			if l > over {
				over = l
			}
		}
		if over <= maxLen {
			return lens
		}
		// Squash weights (keep ≥1) and retry; this eventually drives the
		// max depth down.
		for i := range w {
			w[i] = (w[i] >> 1) | 1
		}
	}
}

// Classic Huffman: repeatedly merge the two lowest-weight active nodes.
// We keep the active list sorted by weight with binary-search insertion.
type hufNode struct {
	weight int
	parent int
}

func huffmanLengths(weights []int) []int {
	n := len(weights)
	if n == 0 {
		return nil
	}
	if n == 1 {
		return []int{1}
	}
	nodes := make([]hufNode, 0, 2*n)
	for _, w := range weights {
		nodes = append(nodes, hufNode{weight: w, parent: -1})
	}
	active := make([]int, n)
	for i := range active {
		active[i] = i
	}
	sort.SliceStable(active, func(a, b int) bool {
		return nodes[active[a]].weight < nodes[active[b]].weight
	})

	for len(active) > 1 {
		a := active[0]
		b := active[1]
		active = active[2:]
		parent := len(nodes)
		nodes = append(nodes, hufNode{weight: nodes[a].weight + nodes[b].weight, parent: -1})
		nodes[a].parent = parent
		nodes[b].parent = parent
		pw := nodes[parent].weight
		pos := sort.Search(len(active), func(i int) bool { return nodes[active[i]].weight > pw })
		active = append(active, 0)
		copy(active[pos+1:], active[pos:])
		active[pos] = parent
	}

	lens := make([]int, n)
	for i := range n {
		depth := 0
		for p := nodes[i].parent; p != -1; p = nodes[p].parent {
			depth++
		}
		if depth == 0 {
			depth = 1
		}
		lens[i] = depth
	}
	return lens
}

// canonicalCodes builds canonical Huffman codes from per-symbol lengths.
// Codes are assigned in (length-ascending, symbol-ascending) order.
func canonicalCodes(lens []int) []uint32 {
	n := len(lens)
	codes := make([]uint32, n)
	bySym := make([]int, n)
	for i := range bySym {
		bySym[i] = i
	}
	sort.SliceStable(bySym, func(a, b int) bool {
		if lens[bySym[a]] != lens[bySym[b]] {
			return lens[bySym[a]] < lens[bySym[b]]
		}
		return bySym[a] < bySym[b]
	})
	var code uint32
	prevLen := 0
	for _, s := range bySym {
		if lens[s] == 0 {
			continue
		}
		if prevLen == 0 {
			code = 0
			prevLen = lens[s]
		} else if lens[s] > prevLen {
			code <<= uint(lens[s] - prevLen)
			prevLen = lens[s]
		}
		codes[s] = code
		code++
	}
	return codes
}

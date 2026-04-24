package bzip2

// pickNumTrees chooses how many Huffman tables to use based on the
// number of post-RLE2 symbols, mirroring the libbzip2 reference ladder.
func pickNumTrees(numSyms int) int {
	switch {
	case numSyms < 200:
		return 2
	case numSyms < 600:
		return 3
	case numSyms < 1200:
		return 4
	case numSyms < 2400:
		return 5
	default:
		return 6
	}
}

// refineSelectors runs a short K-means-style refinement to assign each
// 50-symbol group to the Huffman table that encodes it most cheaply.
// It returns the per-group table index plus the final per-tree code
// lengths.
//
// Starting point: round-robin.  Iteration: build each tree from its
// currently-assigned groups, then re-assign each group to whichever
// tree would encode it in fewest bits.  Three iterations matches the
// classic libbzip2 behaviour well.
func refineSelectors(syms []uint16, alphaSize, numTrees, groupSize int) (selectors []int, lensPerTree [][]int) {
	numGroups := (len(syms) + groupSize - 1) / groupSize
	if numGroups == 0 {
		numGroups = 1
	}
	selectors = make([]int, numGroups)
	for g := range selectors {
		selectors[g] = g % numTrees
	}

	lensPerTree = make([][]int, numTrees)
	for iter := 0; iter < 3; iter++ {
		// Build frequency tables per tree based on current assignment.
		freqsPerTree := make([][]int, numTrees)
		for t := range freqsPerTree {
			freqsPerTree[t] = make([]int, alphaSize)
		}
		for g := 0; g < numGroups; g++ {
			start := g * groupSize
			end := start + groupSize
			if end > len(syms) {
				end = len(syms)
			}
			f := freqsPerTree[selectors[g]]
			for _, s := range syms[start:end] {
				f[int(s)]++
			}
		}
		// Derive Huffman lengths for each tree. For trees with no
		// symbols (possible early in iteration) fall back to the
		// first tree's lengths.
		emptyFallback := -1
		for t := 0; t < numTrees; t++ {
			any := false
			for _, f := range freqsPerTree[t] {
				if f > 0 {
					any = true
					break
				}
			}
			if any {
				lensPerTree[t] = buildHuffman(freqsPerTree[t], maxHuffmanLen)
				if emptyFallback < 0 {
					emptyFallback = t
				}
			}
		}
		for t := 0; t < numTrees; t++ {
			if lensPerTree[t] == nil {
				if emptyFallback >= 0 {
					lensPerTree[t] = append([]int(nil), lensPerTree[emptyFallback]...)
				} else {
					// Absolute fallback: uniform lengths.
					lensPerTree[t] = uniformLengths(alphaSize)
				}
			}
		}

		// Re-assign each group to the cheapest tree.
		for g := 0; g < numGroups; g++ {
			start := g * groupSize
			end := start + groupSize
			if end > len(syms) {
				end = len(syms)
			}
			bestT := 0
			bestCost := -1
			for t := 0; t < numTrees; t++ {
				cost := 0
				for _, s := range syms[start:end] {
					cost += lensPerTree[t][int(s)]
				}
				if bestCost < 0 || cost < bestCost {
					bestCost = cost
					bestT = t
				}
			}
			selectors[g] = bestT
		}
	}

	return selectors, lensPerTree
}

func uniformLengths(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = 15
	}
	return out
}

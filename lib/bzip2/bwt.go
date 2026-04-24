package bzip2

// Burrows-Wheeler transform on cyclic rotations of the input.
//
// We use the doubling-suffix-array technique with counting-sort radix
// passes — O(n log n) time overall. For input length n:
//
//  1. rank[i] = int(data[i]).
//  2. For k = 1, 2, 4, ... until all ranks are distinct or k >= n:
//       sort indices by (rank[i], rank[(i+k) mod n]) using two stable
//       counting-sort passes (first by the secondary key, then by the
//       primary key).  Then re-derive ranks from the sorted order.
//  3. The sorted index array is the BWT sort order of cyclic rotations.
//
// This is dramatically faster than the naive O(n² log n) sort and
// handles degenerate inputs (all identical bytes, periodic inputs) as
// well as random inputs.

func bwt(data []byte) (transformed []byte, origPtr int) {
	n := len(data)
	if n == 0 {
		return nil, 0
	}
	if n == 1 {
		return []byte{data[0]}, 0
	}

	idx := make([]int, n)
	tmp := make([]int, n)
	rank := make([]int, n)
	newRank := make([]int, n)
	secondary := make([]int, n)
	for i := range idx {
		idx[i] = i
		rank[i] = int(data[i])
	}

	// First pass: sort by rank[i] alone (single-key). Counting sort on
	// bytes 0..255 — our initial alphabet.
	countingSortBy(idx, tmp, rank, 256)
	newRank[idx[0]] = 0
	for i := 1; i < n; i++ {
		newRank[idx[i]] = newRank[idx[i-1]]
		if rank[idx[i]] != rank[idx[i-1]] {
			newRank[idx[i]]++
		}
	}
	copy(rank, newRank)
	if rank[idx[n-1]] == n-1 {
		// Already fully sorted after byte pass (rare, needs all distinct).
		return emitBWT(data, idx), findOrigPtr(idx)
	}

	for k := 1; k < n; k *= 2 {
		// Build secondary key for each index: rank[(i+k) mod n].
		for i := 0; i < n; i++ {
			secondary[i] = rank[(i+k)%n]
		}
		// Stable counting sort: first by secondary, then by primary.
		// Alphabet size for ranks is at most n.
		countingSortBy(idx, tmp, secondary, n)
		countingSortBy(idx, tmp, rank, n)

		newRank[idx[0]] = 0
		for i := 1; i < n; i++ {
			newRank[idx[i]] = newRank[idx[i-1]]
			a := idx[i-1]
			b := idx[i]
			if rank[a] != rank[b] || secondary[a] != secondary[b] {
				newRank[idx[i]]++
			}
		}
		copy(rank, newRank)
		if rank[idx[n-1]] == n-1 {
			break
		}
	}

	return emitBWT(data, idx), findOrigPtr(idx)
}

func emitBWT(data []byte, idx []int) []byte {
	n := len(data)
	out := make([]byte, n)
	for i, s := range idx {
		out[i] = data[(s+n-1)%n]
	}
	return out
}

func findOrigPtr(idx []int) int {
	for i, s := range idx {
		if s == 0 {
			return i
		}
	}
	return 0
}

// countingSortBy stably sorts idx by key[idx[i]] using a counting sort
// with bucket alphabet size `alphabet`.
func countingSortBy(idx, tmp, key []int, alphabet int) {
	n := len(idx)
	count := make([]int, alphabet+1)
	for _, i := range idx {
		count[key[i]+1]++
	}
	for i := 1; i < len(count); i++ {
		count[i] += count[i-1]
	}
	for _, i := range idx {
		k := key[i]
		tmp[count[k]] = i
		count[k]++
	}
	copy(idx[:n], tmp[:n])
}

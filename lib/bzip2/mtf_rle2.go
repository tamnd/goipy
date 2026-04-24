package bzip2

// mtfAndRLE2 fuses the Move-to-Front transform with zero-run encoding.
// Output alphabet:
//
//	0     RUNA              ) bijective base-2 encoding of zero-run length:
//	1     RUNB              )   loop: run--;  emit (run&1); run >>= 1.
//	2..N  shifted MTF values 1..N-1
//	N+1   EOB
//
// where N = len(symMap) (number of distinct bytes in the block).
func mtfAndRLE2(data []byte, symMap []byte) (syms []uint16, alphaSize int) {
	n := len(symMap)
	list := make([]byte, n)
	copy(list, symMap)

	syms = make([]uint16, 0, len(data)+1)

	emitRun := func(run int) {
		for run > 0 {
			run--
			syms = append(syms, uint16(run&1))
			run >>= 1
		}
	}

	zeroRun := 0
	for _, b := range data {
		// Find index of b in list.
		idx := 0
		for idx < n && list[idx] != b {
			idx++
		}
		// Move-to-front.
		if idx > 0 {
			c := list[idx]
			copy(list[1:idx+1], list[:idx])
			list[0] = c
		}
		if idx == 0 {
			zeroRun++
		} else {
			if zeroRun > 0 {
				emitRun(zeroRun)
				zeroRun = 0
			}
			syms = append(syms, uint16(idx)+1)
		}
	}
	if zeroRun > 0 {
		emitRun(zeroRun)
	}
	alphaSize = n + 2
	syms = append(syms, uint16(alphaSize-1))
	return
}

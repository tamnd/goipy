package bzip2

// rle1 applies bzip2's input-level run-length encoding: any run of 4+
// identical bytes is emitted as 4 literal copies followed by a count
// byte (0..251 meaning 0..251 *additional* copies beyond the first 4).
// A run is capped at 255 total; longer runs emit further groups on the
// next iteration.
func rle1(data []byte) []byte {
	out := make([]byte, 0, len(data))
	i := 0
	for i < len(data) {
		j := i + 1
		for j < len(data) && data[j] == data[i] && j-i < 255 {
			j++
		}
		run := j - i
		if run >= 4 {
			out = append(out, data[i], data[i], data[i], data[i], byte(run-4))
		} else {
			for range run {
				out = append(out, data[i])
			}
		}
		i = j
	}
	return out
}

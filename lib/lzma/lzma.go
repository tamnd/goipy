// Package lzma implements a pure-Go LZMA / LZMA2 / XZ encoder and
// decoder.
//
// Scope: this implementation emits valid XZ / .lzma / raw LZMA streams
// using a literal-only encoder. Every byte becomes one LIT packet —
// the output is conformant (any LZMA decoder, including liblzma and
// xz-utils, will decode it) but achieves little to no compression.
// The decoder accepts the subset our encoder produces (all literals);
// if a stream contains MATCH/REP packets it returns errUnsup.
//
// The code is structured so a future richer match-finder + optimal
// parser can slot in without changing the container / framing layers.
package lzma

// Properties holds the three LZMA coding parameters plus the dict size.
// Encoder always uses the canonical (lc=3, lp=0, pb=2, dict=8 MiB)
// combination; Properties is kept separate so the decoder can honour
// external streams that chose different values.
type Properties struct {
	LC, LP, PB int
	DictSize   uint32
}

func defaultProperties() Properties {
	return Properties{LC: 3, LP: 0, PB: 2, DictSize: 1 << 23}
}

// propByte encodes the LZMA properties triple into the one-byte form
// used by FORMAT_ALONE and LZMA2.
func (p Properties) propByte() byte {
	return byte((p.PB*5+p.LP)*9 + p.LC)
}

func decodePropByte(b byte) (Properties, error) {
	if b >= 225 {
		return Properties{}, errCorrupt
	}
	d := int(b)
	lc := d % 9
	d /= 9
	lp := d % 5
	pb := d / 5
	if lc+lp > 4 {
		return Properties{}, errCorrupt
	}
	return Properties{LC: lc, LP: lp, PB: pb, DictSize: 1 << 23}, nil
}

// Number of LZMA state-machine states. Only state 0 (literal-after-
// literal) is ever reached by our encoder.
const numStates = 12

// Position/literal-context masks derived from Properties.
func (p Properties) posMask() uint32 { return (1 << uint(p.PB)) - 1 }
func (p Properties) lpMask() uint32  { return (1 << uint(p.LP)) - 1 }

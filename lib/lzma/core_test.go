package lzma

import (
	"bytes"
	"testing"
)

func TestLZMARoundTrip(t *testing.T) {
	cases := [][]byte{
		[]byte("hello world"),
		bytes.Repeat([]byte("abcdefgh"), 200),
		{},
		{0},
		func() []byte {
			b := make([]byte, 4000)
			for i := range b {
				b[i] = byte(i * 7)
			}
			return b
		}(),
	}
	props := defaultProperties()
	for _, in := range cases {
		enc := encodeLZMA(in, props)
		out, _, err := decodeLZMA(enc, props, int64(len(in)))
		if err != nil {
			t.Fatalf("decode len=%d: %v", len(in), err)
		}
		if !bytes.Equal(out, in) {
			t.Fatalf("mismatch len=%d: got %v want %v", len(in), out[:min(8, len(out))], in[:min(8, len(in))])
		}
	}
}

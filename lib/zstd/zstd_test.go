package zstd

import (
	"bytes"
	"testing"
)

func TestXXH64Vectors(t *testing.T) {
	// Known reference vectors (seed=0).
	cases := []struct {
		in   string
		want uint64
	}{
		{"", 0xEF46DB3751D8E999},
		{"a", 0xD24EC4F1A98C6E5B},
		{"abc", 0x44BC2CF5AD770999},
		{"heiho", 0x84D3884810D6C41C},
	}
	for _, c := range cases {
		got := XXH64([]byte(c.in))
		if got != c.want {
			t.Errorf("XXH64(%q) = %#x, want %#x", c.in, got, c.want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	payloads := [][]byte{
		nil,
		[]byte("hello"),
		bytes.Repeat([]byte("abcdefgh"), 100),
		bytes.Repeat([]byte("x"), 1<<18), // > 128 KiB → multiple blocks
	}
	for _, p := range payloads {
		enc := Encode(p)
		dec, n, err := Decode(enc)
		if err != nil {
			t.Fatalf("decode err: %v", err)
		}
		if n != len(enc) {
			t.Errorf("consumed %d, want %d", n, len(enc))
		}
		if !bytes.Equal(dec, p) {
			t.Errorf("round-trip mismatch: got %d bytes, want %d", len(dec), len(p))
		}
	}
}

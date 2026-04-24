package bzip2

import (
	"bytes"
	stdbzip2 "compress/bzip2"
	"io"
	"strings"
	"testing"
)

func roundTrip(t *testing.T, in []byte, level int) {
	t.Helper()
	compressed := Encode(in, level)
	if !bytes.HasPrefix(compressed, []byte("BZh")) {
		t.Fatalf("missing BZh magic: %x", compressed[:min(4, len(compressed))])
	}
	r := stdbzip2.NewReader(bytes.NewReader(compressed))
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("decode level=%d len=%d: %v", level, len(in), err)
	}
	if !bytes.Equal(got, in) {
		t.Fatalf("round-trip mismatch level=%d len=%d\ngot  %q\nwant %q", level, len(in), got, in)
	}
}

func TestRoundTripSmall(t *testing.T) {
	roundTrip(t, []byte("hello world"), 9)
}

func TestRoundTripRepeats(t *testing.T) {
	roundTrip(t, bytes.Repeat([]byte("hello bz2 world "), 50), 9)
}

func TestRoundTripSingleByte(t *testing.T) {
	roundTrip(t, []byte("A"), 9)
}

func TestRoundTripAllSame(t *testing.T) {
	roundTrip(t, bytes.Repeat([]byte{0x42}, 400), 9)
}

func TestRoundTripBinary(t *testing.T) {
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}
	roundTrip(t, data, 9)
}

func TestRoundTripLevels(t *testing.T) {
	data := []byte(strings.Repeat("the quick brown fox jumps over the lazy dog\n", 30))
	for lvl := 1; lvl <= 9; lvl++ {
		roundTrip(t, data, lvl)
	}
}

func TestRoundTripMultiBlock(t *testing.T) {
	// Slightly larger than a level-1 block (100k) to force 2 blocks.
	data := make([]byte, 120000)
	for i := range data {
		data[i] = byte((i * 7) & 0xFF)
	}
	roundTrip(t, data, 1)
}

func TestRoundTripLarge(t *testing.T) {
	// One full level-9 block of semi-compressible text.
	var b bytes.Buffer
	for b.Len() < 900000 {
		b.WriteString("the quick brown fox jumps over the lazy dog. ")
	}
	roundTrip(t, b.Bytes()[:900000], 9)
}

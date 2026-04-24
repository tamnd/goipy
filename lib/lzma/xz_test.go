package lzma

import (
	"bytes"
	"testing"
)

func TestXZRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"short", []byte("hello world")},
		{"medium", bytes.Repeat([]byte("abcdefgh"), 500)},
		{"binary", func() []byte {
			b := make([]byte, 65537)
			for i := range b {
				b[i] = byte(i * 31)
			}
			return b
		}()},
	}
	for _, tc := range cases {
		for _, check := range []CheckID{CheckNone, CheckCRC32, CheckCRC64, CheckSHA256} {
			enc := EncodeXZ(tc.data, check)
			if !bytes.HasPrefix(enc, xzMagic) {
				t.Fatalf("%s check=%d: bad magic %x", tc.name, check, enc[:6])
			}
			out, consumed, err := DecodeXZ(enc)
			if err != nil {
				t.Fatalf("%s check=%d: decode: %v", tc.name, check, err)
			}
			if consumed != len(enc) {
				t.Fatalf("%s check=%d: consumed %d of %d", tc.name, check, consumed, len(enc))
			}
			if !bytes.Equal(out, tc.data) {
				t.Fatalf("%s check=%d: data mismatch (got %d bytes, want %d)", tc.name, check, len(out), len(tc.data))
			}
		}
	}
}

func TestXZMultiStream(t *testing.T) {
	a := EncodeXZ([]byte("stream1"), CheckCRC64)
	b := EncodeXZ([]byte("stream2"), CheckCRC64)
	combined := append(append([]byte{}, a...), b...)
	out, _, err := DecodeXZ(combined)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(out, []byte("stream1stream2")) {
		t.Fatalf("got %q", out)
	}
}

func TestAloneRoundTrip(t *testing.T) {
	data := bytes.Repeat([]byte("xyz"), 400)
	enc := EncodeAlone(data)
	out, err := DecodeAlone(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Fatalf("mismatch")
	}
}

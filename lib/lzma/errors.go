package lzma

import "errors"

var (
	errShort   = errors.New("lzma: unexpected end of input")
	errCorrupt = errors.New("lzma: corrupt data")
	errFormat  = errors.New("lzma: bad stream format")
	errCheck   = errors.New("lzma: integrity check failed")
	errUnsup   = errors.New("lzma: unsupported feature")
)

// IsShortInputErr reports whether err is the "need more bytes" sentinel
// (distinguishable so streaming callers can wait for more input instead
// of surfacing it as a fatal decode error).
func IsShortInputErr(err error) bool { return err == errShort }

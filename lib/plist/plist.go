// Package plist implements Apple Property List (plist) parsing and generation.
// It supports XML plist (bplist00 magic) and binary plist formats.
package plist

import (
	"bytes"
	"fmt"
	"time"
)

// FMT constants mirror Python's plistlib.FMT_XML / FMT_BINARY.
const (
	FMT_XML    = 1
	FMT_BINARY = 2
)

// UID wraps an integer for NSKeyedArchiver encoded data.
type UID struct {
	Data uint64
}

// InvalidFileError is returned when the plist data cannot be parsed.
type InvalidFileError struct {
	Msg string
}

func (e *InvalidFileError) Error() string { return e.Msg }

// Value is any Go value that can appear in a plist:
//
//	string, int64, uint64, float64, bool, []byte, time.Time, UID,
//	[]interface{}, map[string]interface{}
type Value = interface{}

// Loads parses plist data (bytes). fmt may be 0 (auto-detect), FMT_XML or FMT_BINARY.
func Loads(data []byte, fmt int) (Value, error) {
	if fmt == FMT_XML || (fmt == 0 && isXML(data)) {
		return parseXML(data)
	}
	if fmt == FMT_BINARY || (fmt == 0 && isBinary(data)) {
		return parseBinary(data)
	}
	if fmt == 0 {
		// Try XML first, then binary.
		v, err := parseXML(data)
		if err == nil {
			return v, nil
		}
		return parseBinary(data)
	}
	return nil, &InvalidFileError{Msg: fmt_sprintf("unsupported format %d", fmt)}
}

// Dumps serializes v to plist bytes. fmt must be FMT_XML or FMT_BINARY.
// If sortKeys is true, dict keys are sorted. If skipKeys is true, non-string
// dict keys are silently skipped (they cannot occur in Go, so this is a no-op
// at this layer).
func Dumps(v Value, fmtID int, sortKeys bool) ([]byte, error) {
	switch fmtID {
	case FMT_XML, 0:
		return dumpXML(v, sortKeys)
	case FMT_BINARY:
		return dumpBinary(v, sortKeys)
	}
	return nil, fmt.Errorf("unsupported format %d", fmtID)
}

// helpers

func isXML(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	return bytes.HasPrefix(trimmed, []byte("<?xml")) ||
		bytes.HasPrefix(trimmed, []byte("<plist"))
}

func isBinary(data []byte) bool {
	return len(data) >= 8 && string(data[:8]) == "bplist00"
}

func fmt_sprintf(f string, args ...interface{}) string {
	return fmt.Sprintf(f, args...)
}

// CoerceTime converts a Go time.Time to a plist date value (UTC).
func CoerceTime(t time.Time) time.Time {
	return t.UTC()
}

package object

// UUID is a 128-bit RFC 4122 identifier. Stored as raw bytes; version and
// variant are derived on access. We keep the bytes canonical (network-byte
// order) so conversions to hex/int/fields are direct slicing/shifts.
type UUID struct {
	Bytes [16]byte
}

// SequenceMatcher is difflib's similarity-scoring object. We keep the two
// inputs as strings and compute ratios on demand.
type SequenceMatcher struct {
	A, B string
}

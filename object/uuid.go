package object

// UUID is a 128-bit RFC 4122 identifier. Stored as raw bytes; version and
// variant are derived on access. We keep the bytes canonical (network-byte
// order) so conversions to hex/int/fields are direct slicing/shifts.
type UUID struct {
	Bytes [16]byte
}

// SequenceMatcher is difflib's similarity-scoring object.
// When SeqA/SeqB are non-nil, the matcher operates on those sequences;
// otherwise A and B are used as character sequences.
type SequenceMatcher struct {
	A, B     string
	SeqA, SeqB []Object // non-nil when operating on lists
}

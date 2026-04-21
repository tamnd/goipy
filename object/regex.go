package object

import "regexp"

// Pattern is a compiled regular expression. Exposed to Python via the `re`
// module; backed by Go's regexp (RE2), so pattern-level backreferences are
// unsupported and will fail at compile time.
type Pattern struct {
	Pattern string
	Regexp  *regexp.Regexp
	Flags   int64
}

// Match represents a single match against a subject string. Offsets are
// flat [start, end, g1_start, g1_end, ...] indices into String (-1 for
// unmatched optional groups), rebased onto the original string when the
// caller supplied a pos/endpos window.
type Match struct {
	Pattern *Pattern
	String  string
	Offsets []int
	Pos     int
	Endpos  int
}

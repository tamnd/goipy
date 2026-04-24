package object

// StringIO is a minimal in-memory text buffer mirroring io.StringIO. Pos
// tracks the read cursor (writes always append to V — StringIO in CPython
// overwrites on seek, but for the common append-then-read flow we don't
// need that complexity).
type StringIO struct {
	V      []byte
	Pos    int
	Closed bool
}

// BytesIO is the bytes counterpart. Shares the same shape so callers can
// reuse most of the method dispatch.
type BytesIO struct {
	V      []byte
	Pos    int
	Closed bool
}

// Hasher wraps a hash algorithm (md5/sha1/sha256/...). The VM treats this
// as opaque — methods live in the stdlib layer.
type Hasher struct {
	Name      string
	Size      int       // digest_size in bytes (0 for SHAKE variable-length)
	BlockSize int       // block_size in bytes
	IsShake   bool      // true for SHAKE-128/256 (variable-length digest)
	// State is a hash.Hash but kept as `any` to avoid pulling `hash` into
	// the object package's import graph. The stdlib module type-asserts.
	State any
	// NewFn creates a fresh hash of the same type (for copy). Kept as `any`.
	NewFn any
}

// TextStream is a thin wrapper around an io.Writer for sys.stdout /
// sys.stderr. W is kept as `any` to avoid pulling `io` into the object
// package's import graph; the stdlib module type-asserts to io.Writer.
type TextStream struct {
	Name string // "stdout" / "stderr"
	W    any
}

// File wraps an *os.File for the built-in open() function. F is kept as `any`
// to avoid pulling `os` into the object package's import graph; the vm layer
// type-asserts to *os.File.
type File struct {
	F        any    // *os.File
	FilePath string // value of the `name` attribute
	Mode     string // e.g. "r", "wb"
	Binary   bool   // true for "b" modes
	Closed   bool
}

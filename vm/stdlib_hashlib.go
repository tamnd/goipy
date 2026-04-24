package vm

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/crypto/sha3"

	"github.com/tamnd/goipy/object"
)

// buildHashlib constructs the hashlib module, replacing the minimal version
// in stdlib_io.go. It registers all algorithms_guaranteed algorithms plus
// helpers like new() and file_digest().
func (i *Interp) buildHashlib() *object.Module {
	m := &object.Module{Name: "hashlib", Dict: object.NewDict()}

	type algSpec struct {
		name      string
		newFn     func() hash.Hash
		digestSize int
		blockSize  int
		isShake   bool
	}

	algs := []algSpec{
		{"md5", md5.New, 16, 64, false},
		{"sha1", sha1.New, 20, 64, false},
		{"sha224", sha256.New224, 28, 64, false},
		{"sha256", sha256.New, 32, 64, false},
		{"sha384", sha512.New384, 48, 128, false},
		{"sha512", sha512.New, 64, 128, false},
		{"sha3_224", sha3.New224, 28, 144, false},
		{"sha3_256", sha3.New256, 32, 136, false},
		{"sha3_384", sha3.New384, 48, 104, false},
		{"sha3_512", sha3.New512, 64, 72, false},
		// SHAKE — digest_size is 0 (variable); block_size is the rate in bytes
		{"shake_128", func() hash.Hash { return sha3.NewShake128() }, 0, 168, true},
		{"shake_256", func() hash.Hash { return sha3.NewShake256() }, 0, 136, true},
	}

	// Build the guaranteed set as a frozenset string for printing, and a Go
	// slice for membership checks.
	guaranteedNames := make([]object.Object, 0, len(algs)+2)
	for _, a := range algs {
		guaranteedNames = append(guaranteedNames, &object.Str{V: a.name})
	}
	guaranteedNames = append(guaranteedNames, &object.Str{V: "blake2b"})
	guaranteedNames = append(guaranteedNames, &object.Str{V: "blake2s"})

	guaranteed := object.NewFrozenset()
	for _, n := range guaranteedNames {
		guaranteed.Add(n)
	}
	m.Dict.SetStr("algorithms_guaranteed", guaranteed)
	m.Dict.SetStr("algorithms_available", guaranteed) // same set for us

	// Register each standard algorithm as a top-level function.
	for _, a := range algs {
		a := a // capture
		m.Dict.SetStr(a.name, &object.BuiltinFunc{Name: a.name, Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
			return i.newHasher(a.name, a.newFn, a.digestSize, a.blockSize, a.isShake, args, kw)
		}})
	}

	// blake2b
	m.Dict.SetStr("blake2b", &object.BuiltinFunc{Name: "blake2b", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		digestSize := 64
		var key, salt, person []byte
		if kw != nil {
			if v, ok := kw.GetStr("digest_size"); ok {
				if n, ok2 := v.(*object.Int); ok2 {
					digestSize = int(n.Int64())
				}
			}
			if v, ok := kw.GetStr("key"); ok {
				if b, ok2 := v.(*object.Bytes); ok2 {
					key = b.V
				}
			}
			if v, ok := kw.GetStr("salt"); ok {
				if b, ok2 := v.(*object.Bytes); ok2 {
					salt = b.V
				}
			}
			if v, ok := kw.GetStr("person"); ok {
				if b, ok2 := v.(*object.Bytes); ok2 {
					person = b.V
				}
			}
		}
		// blake2b digest_size from positional arg[1] not supported; always kwarg.
		newFn, err := blake2bFactory(digestSize, key, salt, person)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "blake2b: %s", err.Error())
		}
		h := &object.Hasher{
			Name: "blake2b", Size: digestSize, BlockSize: 128, IsShake: false,
			State: newFn(), NewFn: newFn,
		}
		if len(args) >= 1 {
			data, err := asBytes(args[0])
			if err != nil {
				return nil, err
			}
			h.State.(hash.Hash).Write(data)
		}
		return h, nil
	}})

	// blake2s
	m.Dict.SetStr("blake2s", &object.BuiltinFunc{Name: "blake2s", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		digestSize := 32
		var key, salt, person []byte
		if kw != nil {
			if v, ok := kw.GetStr("digest_size"); ok {
				if n, ok2 := v.(*object.Int); ok2 {
					digestSize = int(n.Int64())
				}
			}
			if v, ok := kw.GetStr("key"); ok {
				if b, ok2 := v.(*object.Bytes); ok2 {
					key = b.V
				}
			}
			if v, ok := kw.GetStr("salt"); ok {
				if b, ok2 := v.(*object.Bytes); ok2 {
					salt = b.V
				}
			}
			if v, ok := kw.GetStr("person"); ok {
				if b, ok2 := v.(*object.Bytes); ok2 {
					person = b.V
				}
			}
		}
		newFn, err := blake2sFactory(digestSize, key, salt, person)
		if err != nil {
			return nil, object.Errorf(i.valueErr, "blake2s: %s", err.Error())
		}
		h := &object.Hasher{
			Name: "blake2s", Size: digestSize, BlockSize: 64, IsShake: false,
			State: newFn(), NewFn: newFn,
		}
		if len(args) >= 1 {
			data, err := asBytes(args[0])
			if err != nil {
				return nil, err
			}
			h.State.(hash.Hash).Write(data)
		}
		return h, nil
	}})

	// new(name, data=b'', *, usedforsecurity=True)
	m.Dict.SetStr("new", &object.BuiltinFunc{Name: "new", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		if len(args) < 1 {
			return nil, object.Errorf(i.typeErr, "new() requires an algorithm name")
		}
		s, ok := args[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "new() algorithm must be str")
		}
		name := s.V
		// Remaining positional args become the data arg.
		dataArgs := args[1:]
		// Forward to the per-algorithm function.
		fn, found := m.Dict.GetStr(name)
		if !found {
			return nil, object.Errorf(i.valueErr, "unsupported hash type %s", name)
		}
		bf, ok2 := fn.(*object.BuiltinFunc)
		if !ok2 {
			return nil, object.Errorf(i.valueErr, "unsupported hash type %s", name)
		}
		return bf.Call(nil, dataArgs, kw)
	}})

	// file_digest(fileobj, digest) — Python 3.11+
	m.Dict.SetStr("file_digest", &object.BuiltinFunc{Name: "file_digest", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "file_digest() requires fileobj and digest arguments")
		}
		// digest arg: string name or callable that returns a hash.
		var h *object.Hasher
		switch d := args[1].(type) {
		case *object.Str:
			hObj, err := i.callObject(dictMustGet(m.Dict, "new"), []object.Object{d}, nil)
			if err != nil {
				return nil, err
			}
			h = hObj.(*object.Hasher)
		case *object.BuiltinFunc:
			hObj, err := i.callObject(d, nil, nil)
			if err != nil {
				return nil, err
			}
			h = hObj.(*object.Hasher)
		default:
			return nil, object.Errorf(i.typeErr, "file_digest() digest must be str or callable")
		}
		// Read chunks from fileobj.
		readFn, err := i.getAttr(args[0], "read")
		if err != nil {
			return nil, object.Errorf(i.typeErr, "file_digest() fileobj must have read()")
		}
		const chunkSize = 8192
		for {
			chunk, err := i.callObject(readFn, []object.Object{object.NewInt(chunkSize)}, nil)
			if err != nil {
				return nil, err
			}
			var data []byte
			switch v := chunk.(type) {
			case *object.Bytes:
				data = v.V
			case *object.Str:
				data = []byte(v.V)
			default:
				return nil, object.Errorf(i.typeErr, "read() must return bytes")
			}
			if len(data) == 0 {
				break
			}
			h.State.(hash.Hash).Write(data)
		}
		return h, nil
	}})

	// compare_digest — constant-time comparison
	m.Dict.SetStr("compare_digest", &object.BuiltinFunc{Name: "compare_digest", Call: func(_ any, args []object.Object, kw *object.Dict) (object.Object, error) {
		if len(args) < 2 {
			return nil, object.Errorf(i.typeErr, "compare_digest() requires two arguments")
		}
		var a, b []byte
		switch v := args[0].(type) {
		case *object.Bytes:
			a = v.V
		case *object.Str:
			a = []byte(v.V)
		default:
			return nil, object.Errorf(i.typeErr, "compare_digest() arguments must be str or bytes")
		}
		switch v := args[1].(type) {
		case *object.Bytes:
			b = v.V
		case *object.Str:
			b = []byte(v.V)
		default:
			return nil, object.Errorf(i.typeErr, "compare_digest() arguments must be str or bytes")
		}
		return object.BoolOf(hmac.Equal(a, b)), nil
	}})

	return m
}

// newHasher builds a *object.Hasher using the given factory and optionally
// feeds initial data from args[0]. usedforsecurity kwarg is silently ignored.
func (i *Interp) newHasher(name string, newFn func() hash.Hash, digestSize, blockSize int, isShake bool, args []object.Object, kw *object.Dict) (*object.Hasher, error) {
	h := &object.Hasher{
		Name: name, Size: digestSize, BlockSize: blockSize, IsShake: isShake,
		State: newFn(), NewFn: func() hash.Hash { return newFn() },
	}
	if len(args) >= 1 {
		data, err := asBytes(args[0])
		if err != nil {
			return nil, err
		}
		h.State.(hash.Hash).Write(data)
	}
	return h, nil
}

// hasherAttr dispatches attribute access on a *object.Hasher.
func hasherAttr(i *Interp, h *object.Hasher, name string) (object.Object, bool) {
	switch name {
	case "name":
		return &object.Str{V: h.Name}, true
	case "digest_size":
		return object.NewInt(int64(h.Size)), true
	case "block_size":
		return object.NewInt(int64(h.BlockSize)), true
	case "update":
		return &object.BuiltinFunc{Name: "update", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "update() missing arg")
			}
			data, err := asBytes(a[0])
			if err != nil {
				return nil, err
			}
			h.State.(hash.Hash).Write(data)
			return object.None, nil
		}}, true
	case "hexdigest":
		return &object.BuiltinFunc{Name: "hexdigest", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if h.IsShake {
				// SHAKE: hexdigest(length) is required.
				length := 32 // default
				if len(a) >= 1 {
					if n, ok := a[0].(*object.Int); ok {
						length = int(n.Int64())
					}
				}
				sh, ok := h.State.(interface{ Read([]byte) (int, error) })
				if !ok {
					return nil, object.Errorf(i.typeErr, "shake state does not support Read")
				}
				buf := make([]byte, length)
				// Clone for non-destructive read: use io.Reader directly.
				cloned := cloneShake(h)
				if cr, ok2 := cloned.(interface{ Read([]byte) (int, error) }); ok2 {
					cr.Read(buf)
				} else {
					sh.Read(buf)
				}
				return &object.Str{V: hex.EncodeToString(buf)}, nil
			}
			sum := h.State.(hash.Hash).Sum(nil)
			return &object.Str{V: hex.EncodeToString(sum)}, nil
		}}, true
	case "digest":
		return &object.BuiltinFunc{Name: "digest", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if h.IsShake {
				length := 32
				if len(a) >= 1 {
					if n, ok := a[0].(*object.Int); ok {
						length = int(n.Int64())
					}
				}
				buf := make([]byte, length)
				cloned := cloneShake(h)
				if cr, ok2 := cloned.(interface{ Read([]byte) (int, error) }); ok2 {
					cr.Read(buf)
				} else {
					h.State.(interface{ Read([]byte) (int, error) }).Read(buf)
				}
				return &object.Bytes{V: buf}, nil
			}
			sum := h.State.(hash.Hash).Sum(nil)
			return &object.Bytes{V: sum}, nil
		}}, true
	case "copy":
		return &object.BuiltinFunc{Name: "copy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return copyHasher(h)
		}}, true
	}
	return nil, false
}

// copyHasher clones a Hasher using BinaryMarshaler or a fresh-state fallback.
func copyHasher(h *object.Hasher) (*object.Hasher, error) {
	// Try BinaryMarshaler path (works for crypto/* hashes).
	if m, ok := h.State.(interface{ MarshalBinary() ([]byte, error) }); ok {
		data, err := m.MarshalBinary()
		if err == nil && h.NewFn != nil {
			newFn := h.NewFn.(func() hash.Hash)
			nh := newFn()
			if u, ok2 := nh.(interface{ UnmarshalBinary([]byte) error }); ok2 {
				if err2 := u.UnmarshalBinary(data); err2 == nil {
					return &object.Hasher{
						Name: h.Name, Size: h.Size, BlockSize: h.BlockSize,
						IsShake: h.IsShake, State: nh, NewFn: h.NewFn,
					}, nil
				}
			}
		}
	}
	// SHAKE / SHA3 fallback: clone via CloneHash interface.
	if c, ok := h.State.(interface{ Clone() sha3.ShakeHash }); ok {
		_ = c
	}
	// Last resort: re-run via sha3.ShakeHash if it exposes Clone.
	return nil, object.Errorf(nil, "copy() unsupported for %s", h.Name)
}

// cloneShake returns a copy of the underlying SHAKE state so we can call
// Read without consuming the original.
func cloneShake(h *object.Hasher) any {
	if m, ok := h.State.(interface{ MarshalBinary() ([]byte, error) }); ok {
		data, err := m.MarshalBinary()
		if err == nil && h.NewFn != nil {
			newFn := h.NewFn.(func() hash.Hash)
			nh := newFn()
			if u, ok2 := nh.(interface{ UnmarshalBinary([]byte) error }); ok2 {
				if err2 := u.UnmarshalBinary(data); err2 == nil {
					return nh
				}
			}
		}
	}
	return h.State
}

// blake2bFactory returns a func() hash.Hash for BLAKE2b with the given params.
func blake2bFactory(digestSize int, key, salt, person []byte) (func() hash.Hash, error) {
	// Validate digest size.
	if digestSize < 1 || digestSize > 64 {
		return nil, io.EOF // will be wrapped
	}
	// Pad salt and person to 16 bytes each (blake2b requirement).
	var s16, p16 [16]byte
	copy(s16[:], salt)
	copy(p16[:], person)
	// Validate: blake2b.New requires key len ≤ 64.
	return func() hash.Hash {
		var h hash.Hash
		var err error
		if len(key) > 0 {
			h, err = blake2b.New(digestSize, key)
		} else {
			h, err = blake2b.New(digestSize, nil)
		}
		if err != nil {
			// Fallback: unsalted/unpersoned.
			h, _ = blake2b.New(digestSize, nil)
		}
		return h
	}, nil
}

// blake2sFactory returns a func() hash.Hash for BLAKE2s with the given params.
// golang.org/x/crypto/blake2s only provides New256 (32 bytes) and New128 (16 bytes, keyed only).
func blake2sFactory(digestSize int, key, salt, person []byte) (func() hash.Hash, error) {
	switch digestSize {
	case 32:
		return func() hash.Hash {
			h, _ := blake2s.New256(key)
			return h
		}, nil
	case 16:
		if len(key) == 0 {
			return nil, fmt.Errorf("BLAKE2s: digest_size=16 requires a key")
		}
		return func() hash.Hash {
			h, _ := blake2s.New128(key)
			return h
		}, nil
	default:
		return nil, fmt.Errorf("BLAKE2s: unsupported digest size %d (only 16 with key, or 32)", digestSize)
	}
}

// asBytes coerces a Python bytes/bytearray/str object to a raw byte slice.
func asBytes(o object.Object) ([]byte, error) {
	switch v := o.(type) {
	case *object.Bytes:
		return v.V, nil
	case *object.Bytearray:
		return v.V, nil
	case *object.Str:
		return []byte(v.V), nil
	}
	return nil, object.Errorf(nil, "expected bytes-like, got %s", object.TypeName(o))
}

// dictMustGet panics if key is not in dict (used only in internal setup code).
func dictMustGet(d *object.Dict, key string) object.Object {
	v, ok := d.GetStr(key)
	if !ok {
		panic("dict missing key: " + key)
	}
	return v
}

package vm

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"hash"
	"strings"

	"github.com/tamnd/goipy/object"
)

// --- io module: StringIO and BytesIO ---

// buildIO exposes `io.StringIO` and `io.BytesIO` constructors. The returned
// objects are plain *object.StringIO / *object.BytesIO; method dispatch
// happens through getAttr hooks (see stringIOAttr / bytesIOAttr).
func (i *Interp) buildIO() *object.Module {
	m := &object.Module{Name: "io", Dict: object.NewDict()}

	m.Dict.SetStr("StringIO", &object.BuiltinFunc{Name: "StringIO", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		sio := &object.StringIO{}
		if len(a) >= 1 {
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "initial_value must be str")
			}
			sio.V = []byte(s.V)
		}
		return sio, nil
	}})

	m.Dict.SetStr("BytesIO", &object.BuiltinFunc{Name: "BytesIO", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		bio := &object.BytesIO{}
		if len(a) >= 1 {
			switch v := a[0].(type) {
			case *object.Bytes:
				bio.V = append([]byte(nil), v.V...)
			case *object.Bytearray:
				bio.V = append([]byte(nil), v.V...)
			default:
				return nil, object.Errorf(i.typeErr, "initial_value must be bytes")
			}
		}
		return bio, nil
	}})

	return m
}

// stringIOAttr dispatches attribute access on a *object.StringIO.
func stringIOAttr(i *Interp, sio *object.StringIO, name string) (object.Object, bool) {
	switch name {
	case "closed":
		return object.BoolOf(sio.Closed), true
	case "write":
		return &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if sio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "write() argument must be str")
			}
			// At EOF we simply extend; otherwise overlay and possibly extend.
			if sio.Pos >= len(sio.V) {
				sio.V = append(sio.V, s.V...)
				sio.Pos = len(sio.V)
			} else {
				end := sio.Pos + len(s.V)
				if end > len(sio.V) {
					sio.V = append(sio.V[:sio.Pos], s.V...)
				} else {
					copy(sio.V[sio.Pos:], s.V)
				}
				sio.Pos = end
			}
			return object.NewInt(int64(len([]rune(s.V)))), nil
		}}, true
	case "read":
		return &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if sio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			n := -1
			if len(a) >= 1 {
				if v, ok := toInt64(a[0]); ok {
					n = int(v)
				}
			}
			start := sio.Pos
			end := len(sio.V)
			if n >= 0 && start+n < end {
				end = start + n
			}
			sio.Pos = end
			return &object.Str{V: string(sio.V[start:end])}, nil
		}}, true
	case "readline":
		return &object.BuiltinFunc{Name: "readline", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if sio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			start := sio.Pos
			end := start
			for end < len(sio.V) && sio.V[end] != '\n' {
				end++
			}
			if end < len(sio.V) {
				end++ // include the newline
			}
			sio.Pos = end
			return &object.Str{V: string(sio.V[start:end])}, nil
		}}, true
	case "readlines":
		return &object.BuiltinFunc{Name: "readlines", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if sio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			out := []object.Object{}
			for sio.Pos < len(sio.V) {
				start := sio.Pos
				end := start
				for end < len(sio.V) && sio.V[end] != '\n' {
					end++
				}
				if end < len(sio.V) {
					end++
				}
				out = append(out, &object.Str{V: string(sio.V[start:end])})
				sio.Pos = end
			}
			return &object.List{V: out}, nil
		}}, true
	case "writelines":
		return &object.BuiltinFunc{Name: "writelines", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "writelines() missing arg")
			}
			it, err := i.getIter(a[0])
			if err != nil {
				return nil, err
			}
			for {
				v, ok, err := it.Next()
				if err != nil {
					return nil, err
				}
				if !ok {
					break
				}
				s, ok := v.(*object.Str)
				if !ok {
					return nil, object.Errorf(i.typeErr, "writelines() requires str items")
				}
				sio.V = append(sio.V, s.V...)
				sio.Pos = len(sio.V)
			}
			return object.None, nil
		}}, true
	case "getvalue":
		return &object.BuiltinFunc{Name: "getvalue", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: string(sio.V)}, nil
		}}, true
	case "seek":
		return &object.BuiltinFunc{Name: "seek", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return ioSeek(&sio.Pos, len(sio.V), a)
		}}, true
	case "tell":
		return &object.BuiltinFunc{Name: "tell", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(sio.Pos)), nil
		}}, true
	case "close":
		return &object.BuiltinFunc{Name: "close", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			sio.Closed = true
			return object.None, nil
		}}, true
	case "truncate":
		return &object.BuiltinFunc{Name: "truncate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			size := sio.Pos
			if len(a) >= 1 {
				if v, ok := toInt64(a[0]); ok {
					size = int(v)
				}
			}
			if size < len(sio.V) {
				sio.V = sio.V[:size]
			}
			return object.NewInt(int64(size)), nil
		}}, true
	}
	return nil, false
}

// bytesIOAttr mirrors stringIOAttr for the bytes variant.
func bytesIOAttr(i *Interp, bio *object.BytesIO, name string) (object.Object, bool) {
	switch name {
	case "closed":
		return object.BoolOf(bio.Closed), true
	case "write":
		return &object.BuiltinFunc{Name: "write", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if bio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			var data []byte
			switch v := a[0].(type) {
			case *object.Bytes:
				data = v.V
			case *object.Bytearray:
				data = v.V
			default:
				return nil, object.Errorf(i.typeErr, "write() argument must be bytes")
			}
			if bio.Pos >= len(bio.V) {
				bio.V = append(bio.V, data...)
				bio.Pos = len(bio.V)
			} else {
				end := bio.Pos + len(data)
				if end > len(bio.V) {
					bio.V = append(bio.V[:bio.Pos], data...)
				} else {
					copy(bio.V[bio.Pos:], data)
				}
				bio.Pos = end
			}
			return object.NewInt(int64(len(data))), nil
		}}, true
	case "read":
		return &object.BuiltinFunc{Name: "read", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if bio.Closed {
				return nil, object.Errorf(i.valueErr, "I/O operation on closed file")
			}
			n := -1
			if len(a) >= 1 {
				if v, ok := toInt64(a[0]); ok {
					n = int(v)
				}
			}
			start := bio.Pos
			end := len(bio.V)
			if n >= 0 && start+n < end {
				end = start + n
			}
			bio.Pos = end
			return &object.Bytes{V: append([]byte(nil), bio.V[start:end]...)}, nil
		}}, true
	case "getvalue":
		return &object.BuiltinFunc{Name: "getvalue", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Bytes{V: append([]byte(nil), bio.V...)}, nil
		}}, true
	case "seek":
		return &object.BuiltinFunc{Name: "seek", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return ioSeek(&bio.Pos, len(bio.V), a)
		}}, true
	case "tell":
		return &object.BuiltinFunc{Name: "tell", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(int64(bio.Pos)), nil
		}}, true
	case "close":
		return &object.BuiltinFunc{Name: "close", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			bio.Closed = true
			return object.None, nil
		}}, true
	}
	return nil, false
}

// ioSeek implements the shared seek(offset, whence=0) semantics. Whence 0
// is absolute, 1 is relative to cursor, 2 is relative to end.
func ioSeek(pos *int, size int, a []object.Object) (object.Object, error) {
	if len(a) < 1 {
		return nil, nil
	}
	offset, _ := toInt64(a[0])
	whence := int64(0)
	if len(a) >= 2 {
		if v, ok := toInt64(a[1]); ok {
			whence = v
		}
	}
	var target int64
	switch whence {
	case 0:
		target = offset
	case 1:
		target = int64(*pos) + offset
	case 2:
		target = int64(size) + offset
	}
	if target < 0 {
		target = 0
	}
	*pos = int(target)
	return object.NewInt(target), nil
}

// --- hashlib module ---

func (i *Interp) buildHashlib() *object.Module {
	m := &object.Module{Name: "hashlib", Dict: object.NewDict()}

	mk := func(name string, newFn func() hash.Hash, size int) {
		m.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			h := &object.Hasher{Name: name, Size: size, State: newFn()}
			if len(a) >= 1 {
				data, err := asBytes(a[0])
				if err != nil {
					return nil, err
				}
				h.State.(hash.Hash).Write(data)
			}
			return h, nil
		}})
	}
	mk("md5", md5.New, 16)
	mk("sha1", sha1.New, 20)
	mk("sha224", sha256.New224, 28)
	mk("sha256", sha256.New, 32)
	mk("sha384", sha512.New384, 48)
	mk("sha512", sha512.New, 64)

	// new(name, data=b'') dispatches by name.
	m.Dict.SetStr("new", &object.BuiltinFunc{Name: "new", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "new() needs algorithm name")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "new() algorithm must be str")
		}
		name := strings.ToLower(s.V)
		var newFn func() hash.Hash
		var size int
		switch name {
		case "md5":
			newFn, size = md5.New, 16
		case "sha1":
			newFn, size = sha1.New, 20
		case "sha224":
			newFn, size = sha256.New224, 28
		case "sha256":
			newFn, size = sha256.New, 32
		case "sha384":
			newFn, size = sha512.New384, 48
		case "sha512":
			newFn, size = sha512.New, 64
		default:
			return nil, object.Errorf(i.valueErr, "unsupported hash %q", name)
		}
		h := &object.Hasher{Name: name, Size: size, State: newFn()}
		if len(a) >= 2 {
			data, err := asBytes(a[1])
			if err != nil {
				return nil, err
			}
			h.State.(hash.Hash).Write(data)
		}
		return h, nil
	}})

	m.Dict.SetStr("algorithms_available", &object.Set{})
	m.Dict.SetStr("algorithms_guaranteed", &object.Set{})
	return m
}

// hasherAttr dispatches attribute access on a *object.Hasher.
func hasherAttr(i *Interp, h *object.Hasher, name string) (object.Object, bool) {
	switch name {
	case "name":
		return &object.Str{V: h.Name}, true
	case "digest_size":
		return object.NewInt(int64(h.Size)), true
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
			sum := h.State.(hash.Hash).Sum(nil)
			return &object.Str{V: hex.EncodeToString(sum)}, nil
		}}, true
	case "digest":
		return &object.BuiltinFunc{Name: "digest", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			sum := h.State.(hash.Hash).Sum(nil)
			return &object.Bytes{V: sum}, nil
		}}, true
	case "copy":
		return &object.BuiltinFunc{Name: "copy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			// Go's hash.Hash doesn't expose a generic Clone; the
			// BinaryMarshaler path works for the crypto/ hashes we use.
			if m, ok := h.State.(interface {
				MarshalBinary() ([]byte, error)
			}); ok {
				data, err := m.MarshalBinary()
				if err == nil {
					var nh hash.Hash
					switch h.Name {
					case "md5":
						nh = md5.New()
					case "sha1":
						nh = sha1.New()
					case "sha224":
						nh = sha256.New224()
					case "sha256":
						nh = sha256.New()
					case "sha384":
						nh = sha512.New384()
					case "sha512":
						nh = sha512.New()
					}
					if u, ok := nh.(interface {
						UnmarshalBinary([]byte) error
					}); ok {
						if err := u.UnmarshalBinary(data); err == nil {
							return &object.Hasher{Name: h.Name, Size: h.Size, State: nh}, nil
						}
					}
				}
			}
			return nil, object.Errorf(i.valueErr, "copy() unsupported for %s", h.Name)
		}}, true
	}
	return nil, false
}

// asBytes coerces Str/Bytes/Bytearray to a raw byte slice. Strings encode
// as UTF-8, matching what CPython does when feeding str to a hash after an
// explicit .encode() — but most callers pass bytes.
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

// --- base64 module ---

func (i *Interp) buildBase64() *object.Module {
	m := &object.Module{Name: "base64", Dict: object.NewDict()}

	enc := func(name string, fn func([]byte) string) {
		m.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "%s requires input", name)
			}
			data, err := asBytes(a[0])
			if err != nil {
				return nil, err
			}
			return &object.Bytes{V: []byte(fn(data))}, nil
		}})
	}
	dec := func(name string, fn func(string) ([]byte, error)) {
		m.Dict.SetStr(name, &object.BuiltinFunc{Name: name, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "%s requires input", name)
			}
			var s string
			switch v := a[0].(type) {
			case *object.Str:
				s = v.V
			case *object.Bytes:
				s = string(v.V)
			case *object.Bytearray:
				s = string(v.V)
			default:
				return nil, object.Errorf(i.typeErr, "%s argument must be str or bytes", name)
			}
			out, err := fn(s)
			if err != nil {
				return nil, object.Errorf(i.valueErr, "base64: %s", err.Error())
			}
			return &object.Bytes{V: out}, nil
		}})
	}

	enc("b64encode", base64.StdEncoding.EncodeToString)
	dec("b64decode", base64.StdEncoding.DecodeString)
	enc("urlsafe_b64encode", base64.URLEncoding.EncodeToString)
	dec("urlsafe_b64decode", base64.URLEncoding.DecodeString)
	enc("b32encode", base32.StdEncoding.EncodeToString)
	dec("b32decode", base32.StdEncoding.DecodeString)
	enc("b16encode", func(b []byte) string { return strings.ToUpper(hex.EncodeToString(b)) })
	dec("b16decode", func(s string) ([]byte, error) { return hex.DecodeString(strings.ToLower(s)) })

	// standard_b64encode / standard_b64decode are aliases.
	if v, ok := m.Dict.GetStr("b64encode"); ok {
		m.Dict.SetStr("standard_b64encode", v)
	}
	if v, ok := m.Dict.GetStr("b64decode"); ok {
		m.Dict.SetStr("standard_b64decode", v)
	}

	return m
}

// --- textwrap module ---

func (i *Interp) buildTextwrap() *object.Module {
	m := &object.Module{Name: "textwrap", Dict: object.NewDict()}

	m.Dict.SetStr("dedent", &object.BuiltinFunc{Name: "dedent", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "dedent() argument must be str")
		}
		return &object.Str{V: dedent(s.V)}, nil
	}})

	m.Dict.SetStr("indent", &object.BuiltinFunc{Name: "indent", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "indent() missing args")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "indent() text must be str")
		}
		prefix, ok := a[1].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "indent() prefix must be str")
		}
		return &object.Str{V: indent(s.V, prefix.V)}, nil
	}})

	m.Dict.SetStr("wrap", &object.BuiltinFunc{Name: "wrap", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "wrap() missing text")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "wrap() text must be str")
		}
		width := 70
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok {
				width = int(n)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("width"); ok {
				if n, ok := toInt64(v); ok {
					width = int(n)
				}
			}
		}
		lines := wrap(s.V, width)
		out := make([]object.Object, len(lines))
		for k, l := range lines {
			out[k] = &object.Str{V: l}
		}
		return &object.List{V: out}, nil
	}})

	m.Dict.SetStr("fill", &object.BuiltinFunc{Name: "fill", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fill() missing text")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "fill() text must be str")
		}
		width := 70
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok {
				width = int(n)
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("width"); ok {
				if n, ok := toInt64(v); ok {
					width = int(n)
				}
			}
		}
		return &object.Str{V: strings.Join(wrap(s.V, width), "\n")}, nil
	}})

	m.Dict.SetStr("shorten", &object.BuiltinFunc{Name: "shorten", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "shorten() missing text")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "shorten() text must be str")
		}
		width := 70
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok {
				width = int(n)
			}
		}
		placeholder := " [...]"
		if kw != nil {
			if v, ok := kw.GetStr("width"); ok {
				if n, ok := toInt64(v); ok {
					width = int(n)
				}
			}
			if v, ok := kw.GetStr("placeholder"); ok {
				if p, ok := v.(*object.Str); ok {
					placeholder = p.V
				}
			}
		}
		return &object.Str{V: shorten(s.V, width, placeholder)}, nil
	}})

	return m
}

// dedent strips the longest common leading whitespace from every non-blank
// line. Blank lines are normalized to empty, matching CPython's behavior.
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	prefix := ""
	set := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		if !set {
			prefix = indent
			set = true
		} else {
			prefix = commonPrefix(prefix, indent)
		}
	}
	if prefix == "" {
		// Still blank-line-normalize.
		for k, line := range lines {
			if strings.TrimSpace(line) == "" {
				lines[k] = ""
			}
		}
		return strings.Join(lines, "\n")
	}
	for k, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[k] = ""
		} else if strings.HasPrefix(line, prefix) {
			lines[k] = line[len(prefix):]
		}
	}
	return strings.Join(lines, "\n")
}

func commonPrefix(a, b string) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for k := 0; k < n; k++ {
		if a[k] != b[k] {
			return a[:k]
		}
	}
	return a[:n]
}

// indent prepends prefix to each non-empty line (CPython default).
func indent(text, prefix string) string {
	lines := strings.SplitAfter(text, "\n")
	var b strings.Builder
	for _, line := range lines {
		if strings.TrimSpace(strings.TrimRight(line, "\n")) != "" {
			b.WriteString(prefix)
		}
		b.WriteString(line)
	}
	return b.String()
}

// wrap splits text into lines no wider than width, breaking on whitespace.
// Words longer than width are split (CPython's default break_long_words).
// Empty input returns an empty list.
func wrap(text string, width int) []string {
	if width <= 0 {
		width = 1
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			lines = append(lines, cur.String())
			cur.Reset()
		}
	}
	for _, w := range words {
		// Break oversized words across lines by emitting width-sized slices.
		for len(w) > width {
			if cur.Len() > 0 {
				remaining := width - cur.Len() - 1
				if remaining > 0 {
					cur.WriteByte(' ')
					cur.WriteString(w[:remaining])
					w = w[remaining:]
				}
				flush()
			}
			cur.WriteString(w[:width])
			w = w[width:]
			flush()
		}
		if w == "" {
			continue
		}
		if cur.Len() == 0 {
			cur.WriteString(w)
			continue
		}
		if cur.Len()+1+len(w) > width {
			flush()
			cur.WriteString(w)
			continue
		}
		cur.WriteByte(' ')
		cur.WriteString(w)
	}
	flush()
	return lines
}

// shorten collapses whitespace then trims to width, appending placeholder
// when the result would otherwise exceed width.
func shorten(text string, width int, placeholder string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= width {
		return text
	}
	// Walk back word-by-word until the result + placeholder fits.
	words := strings.Split(text, " ")
	for n := len(words); n > 0; n-- {
		candidate := strings.Join(words[:n], " ") + placeholder
		if len(candidate) <= width {
			return candidate
		}
	}
	return placeholder
}

package vm

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"

	"github.com/tamnd/goipy/object"
)

// BOM constants matching CPython.
var (
	bomUTF8    = []byte{0xEF, 0xBB, 0xBF}
	bomUTF16BE = []byte{0xFE, 0xFF}
	bomUTF16LE = []byte{0xFF, 0xFE}
	bomUTF32BE = []byte{0x00, 0x00, 0xFE, 0xFF}
	bomUTF32LE = []byte{0xFF, 0xFE, 0x00, 0x00}
)

// normaliseEncoding maps CPython alias names to a canonical key.
func normaliseEncoding(name string) string {
	n := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, "-", "_"), " ", "_"))
	switch n {
	case "utf_8", "utf8", "u8":
		return "utf-8"
	case "ascii", "us_ascii", "us", "646", "ansi_x3.4_1968":
		return "ascii"
	case "latin_1", "latin1", "iso_8859_1", "iso8859_1", "iso_8859_1_", "8859",
		"cp819", "csisolatin1", "iso8859", "iso_ir_100", "l1":
		return "latin-1"
	case "utf_16", "utf16", "u16":
		return "utf-16"
	case "utf_16_be", "utf16_be":
		return "utf-16-be"
	case "utf_16_le", "utf16_le":
		return "utf-16-le"
	case "utf_32", "utf32", "u32":
		return "utf-32"
	case "utf_32_be", "utf32_be":
		return "utf-32-be"
	case "utf_32_le", "utf32_le":
		return "utf-32-le"
	case "hex_codec", "hex":
		return "hex_codec"
	case "base64_codec", "base64", "base_64":
		return "base64_codec"
	case "rot_13", "rot13":
		return "rot_13"
	}
	return n
}

// rot13Byte applies ROT-13 to a single ASCII letter.
func rot13Byte(b byte) byte {
	switch {
	case b >= 'A' && b <= 'Z':
		return 'A' + (b-'A'+13)%26
	case b >= 'a' && b <= 'z':
		return 'a' + (b-'a'+13)%26
	}
	return b
}

func rot13String(s string) string {
	out := make([]byte, len(s))
	for i := range s {
		out[i] = rot13Byte(s[i])
	}
	return string(out)
}

// applyEncodeErrors applies the error handler name when a byte can't be encoded.
func applyEncodeErrors(errName, s string) ([]byte, error) {
	var out []byte
	for _, r := range s {
		if r > 0x7F {
			switch errName {
			case "ignore":
				// skip
			case "replace":
				out = append(out, '?')
			case "xmlcharrefreplace":
				out = append(out, []byte("&#"+codecsItoA(int(r))+";")...)
			case "backslashreplace":
				if r <= 0xFF {
					out = append(out, []byte(`\x`+hex.EncodeToString([]byte{byte(r)}))...)
				} else if r <= 0xFFFF {
					out = append(out, []byte(`\u`+zeroPad(hex.EncodeToString([]byte{byte(r >> 8), byte(r)}), 4))...)
				} else {
					b := []byte{byte(r >> 24), byte(r >> 16), byte(r >> 8), byte(r)}
					out = append(out, []byte(`\U`+zeroPad(hex.EncodeToString(b), 8))...)
				}
			default:
				return nil, nil // caller raises UnicodeEncodeError
			}
		} else {
			out = append(out, byte(r))
		}
	}
	return out, nil
}

func codecsItoA(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}

func zeroPad(s string, n int) string {
	for len(s) < n {
		s = "0" + s
	}
	return s
}

// applyDecodeErrors handles bytes that can't be decoded as ASCII.
func applyDecodeErrors(errName string, data []byte) (string, error) {
	var out strings.Builder
	for _, b := range data {
		if b > 0x7F {
			switch errName {
			case "ignore":
				// skip
			case "replace":
				out.WriteRune(utf8.RuneError)
			default:
				return "", nil // caller raises
			}
		} else {
			out.WriteByte(b)
		}
	}
	return out.String(), nil
}

// codecsEncode encodes obj using the given codec.
func (i *Interp) codecsEncode(obj object.Object, enc, errName string) (object.Object, error) {
	enc = normaliseEncoding(enc)
	switch enc {
	case "utf-8":
		s, ok := obj.(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "'%s' does not support encoding of '%s'", enc, object.TypeName(obj))
		}
		return &object.Bytes{V: []byte(s.V)}, nil

	case "ascii":
		s, ok := obj.(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "'%s' does not support encoding of '%s'", enc, object.TypeName(obj))
		}
		out, err := applyEncodeErrors(errName, s.V)
		if err != nil || out == nil {
			if errName == "strict" || out == nil {
				return nil, object.Errorf(i.unicodeErr, "'ascii' codec can't encode characters in string")
			}
		}
		return &object.Bytes{V: out}, nil

	case "latin-1":
		s, ok := obj.(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "'%s' does not support encoding of '%s'", enc, object.TypeName(obj))
		}
		enc2 := charmap.ISO8859_1.NewEncoder()
		b, err := enc2.Bytes([]byte(s.V))
		if err != nil && errName == "strict" {
			return nil, object.Errorf(i.unicodeErr, "'latin-1' codec can't encode character")
		}
		if err != nil {
			return &object.Bytes{V: []byte(s.V)}, nil // fallback
		}
		return &object.Bytes{V: b}, nil

	case "utf-16":
		s, ok := obj.(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "'%s' does not support encoding of '%s'", enc, object.TypeName(obj))
		}
		enc2 := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder()
		b, _ := enc2.Bytes([]byte(s.V))
		return &object.Bytes{V: b}, nil

	case "utf-16-be":
		s, ok := obj.(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "'%s' does not support encoding of '%s'", enc, object.TypeName(obj))
		}
		enc2 := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewEncoder()
		b, _ := enc2.Bytes([]byte(s.V))
		return &object.Bytes{V: b}, nil

	case "utf-16-le":
		s, ok := obj.(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "'%s' does not support encoding of '%s'", enc, object.TypeName(obj))
		}
		enc2 := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder()
		b, _ := enc2.Bytes([]byte(s.V))
		return &object.Bytes{V: b}, nil

	case "hex_codec":
		data, err := asBytes(obj)
		if err != nil {
			return nil, object.Errorf(i.typeErr, "hex_codec encoder requires bytes-like")
		}
		return &object.Bytes{V: []byte(hex.EncodeToString(data))}, nil

	case "base64_codec":
		data, err := asBytes(obj)
		if err != nil {
			return nil, object.Errorf(i.typeErr, "base64_codec encoder requires bytes-like")
		}
		return &object.Bytes{V: []byte(base64.StdEncoding.EncodeToString(data))}, nil

	case "rot_13":
		s, ok := obj.(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "rot_13 encoder requires str")
		}
		return &object.Str{V: rot13String(s.V)}, nil
	}
	return nil, object.Errorf(i.lookupErr, "unknown encoding: %s", enc)
}

// codecsDecode decodes obj using the given codec.
func (i *Interp) codecsDecode(obj object.Object, enc, errName string) (object.Object, error) {
	enc = normaliseEncoding(enc)
	switch enc {
	case "utf-8":
		data, err := asBytes(obj)
		if err != nil {
			return nil, object.Errorf(i.typeErr, "'%s' does not support decoding of '%s'", enc, object.TypeName(obj))
		}
		if errName == "strict" {
			if !utf8.Valid(data) {
				return nil, object.Errorf(i.unicodeErr, "'utf-8' codec can't decode bytes")
			}
		}
		return &object.Str{V: string(data)}, nil

	case "ascii":
		data, err := asBytes(obj)
		if err != nil {
			return nil, object.Errorf(i.typeErr, "'%s' does not support decoding of '%s'", enc, object.TypeName(obj))
		}
		s, err2 := applyDecodeErrors(errName, data)
		if err2 != nil || (s == "" && errName == "strict") {
			for _, b := range data {
				if b > 0x7F {
					return nil, object.Errorf(i.unicodeErr, "'ascii' codec can't decode byte 0x%02x", b)
				}
			}
		}
		if s == "" && errName == "strict" {
			return &object.Str{V: string(data)}, nil
		}
		return &object.Str{V: s}, nil

	case "latin-1":
		data, err := asBytes(obj)
		if err != nil {
			return nil, object.Errorf(i.typeErr, "'%s' does not support decoding of '%s'", enc, object.TypeName(obj))
		}
		dec := charmap.ISO8859_1.NewDecoder()
		b, err2 := dec.Bytes(data)
		if err2 != nil {
			return nil, object.Errorf(i.unicodeErr, "'latin-1' codec can't decode")
		}
		return &object.Str{V: string(b)}, nil

	case "utf-16":
		data, err := asBytes(obj)
		if err != nil {
			return nil, object.Errorf(i.typeErr, "'%s' does not support decoding of '%s'", enc, object.TypeName(obj))
		}
		dec := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()
		b, err2 := dec.Bytes(data)
		if err2 != nil {
			return nil, object.Errorf(i.unicodeErr, "'utf-16' codec can't decode")
		}
		return &object.Str{V: string(b)}, nil

	case "utf-16-be":
		data, err := asBytes(obj)
		if err != nil {
			return nil, object.Errorf(i.typeErr, "'%s' does not support decoding of '%s'", enc, object.TypeName(obj))
		}
		dec := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
		b, err2 := dec.Bytes(data)
		if err2 != nil {
			return nil, object.Errorf(i.unicodeErr, "'utf-16-be' codec can't decode")
		}
		return &object.Str{V: string(b)}, nil

	case "utf-16-le":
		data, err := asBytes(obj)
		if err != nil {
			return nil, object.Errorf(i.typeErr, "'%s' does not support decoding of '%s'", enc, object.TypeName(obj))
		}
		dec := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		b, err2 := dec.Bytes(data)
		if err2 != nil {
			return nil, object.Errorf(i.unicodeErr, "'utf-16-le' codec can't decode")
		}
		return &object.Str{V: string(b)}, nil

	case "hex_codec":
		data, err := asBytes(obj)
		if err != nil {
			return nil, object.Errorf(i.typeErr, "hex_codec decoder requires bytes-like")
		}
		b, err2 := hex.DecodeString(string(data))
		if err2 != nil {
			return nil, object.Errorf(i.valueErr, "hex_codec: invalid hex data")
		}
		return &object.Bytes{V: b}, nil

	case "base64_codec":
		data, err := asBytes(obj)
		if err != nil {
			return nil, object.Errorf(i.typeErr, "base64_codec decoder requires bytes-like")
		}
		b, err2 := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
		if err2 != nil {
			return nil, object.Errorf(i.valueErr, "base64_codec: invalid base64 data")
		}
		return &object.Bytes{V: b}, nil

	case "rot_13":
		s, ok := obj.(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "rot_13 decoder requires str")
		}
		return &object.Str{V: rot13String(s.V)}, nil
	}
	return nil, object.Errorf(i.lookupErr, "unknown encoding: %s", enc)
}

func (i *Interp) buildCodecs() *object.Module {
	m := &object.Module{Name: "codecs", Dict: object.NewDict()}

	// --- BOM constants ---
	m.Dict.SetStr("BOM_UTF8", &object.Bytes{V: bomUTF8})
	m.Dict.SetStr("BOM_UTF16_BE", &object.Bytes{V: bomUTF16BE})
	m.Dict.SetStr("BOM_UTF16_LE", &object.Bytes{V: bomUTF16LE})
	m.Dict.SetStr("BOM_UTF32_BE", &object.Bytes{V: bomUTF32BE})
	m.Dict.SetStr("BOM_UTF32_LE", &object.Bytes{V: bomUTF32LE})
	m.Dict.SetStr("BOM_UTF16", &object.Bytes{V: bomUTF16LE}) // platform little-endian
	m.Dict.SetStr("BOM_UTF32", &object.Bytes{V: bomUTF32LE})
	m.Dict.SetStr("BOM_BE", &object.Bytes{V: bomUTF16BE})
	m.Dict.SetStr("BOM_LE", &object.Bytes{V: bomUTF16LE})
	m.Dict.SetStr("BOM", &object.Bytes{V: bomUTF16LE})

	// --- Built-in error handler functions ---
	strictErrors := &object.BuiltinFunc{Name: "strict_errors", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) > 0 {
			return nil, a[0].(error)
		}
		return nil, object.Errorf(i.unicodeErr, "strict error handler called")
	}}
	ignoreErrors := &object.BuiltinFunc{Name: "ignore_errors", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{&object.Str{V: ""}, object.NewInt(0)}}, nil
	}}
	replaceErrors := &object.BuiltinFunc{Name: "replace_errors", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{&object.Str{V: "\uFFFD"}, object.NewInt(0)}}, nil
	}}
	backslashReplaceErrors := &object.BuiltinFunc{Name: "backslashreplace_errors", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{&object.Str{V: "?"}, object.NewInt(0)}}, nil
	}}
	xmlcharrefErrors := &object.BuiltinFunc{Name: "xmlcharrefreplace_errors", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Tuple{V: []object.Object{&object.Str{V: ""}, object.NewInt(0)}}, nil
	}}
	m.Dict.SetStr("strict_errors", strictErrors)
	m.Dict.SetStr("ignore_errors", ignoreErrors)
	m.Dict.SetStr("replace_errors", replaceErrors)
	m.Dict.SetStr("backslashreplace_errors", backslashReplaceErrors)
	m.Dict.SetStr("xmlcharrefreplace_errors", xmlcharrefErrors)

	// error handler registry
	errorRegistry := map[string]*object.BuiltinFunc{
		"strict":            strictErrors,
		"ignore":            ignoreErrors,
		"replace":           replaceErrors,
		"backslashreplace":  backslashReplaceErrors,
		"xmlcharrefreplace": xmlcharrefErrors,
	}

	m.Dict.SetStr("register_error", &object.BuiltinFunc{Name: "register_error", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "register_error() requires name and handler")
		}
		name, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "register_error() name must be str")
		}
		fn, ok2 := a[1].(*object.BuiltinFunc)
		if !ok2 {
			return nil, object.Errorf(i.typeErr, "register_error() handler must be callable")
		}
		errorRegistry[name.V] = fn
		return object.None, nil
	}})

	m.Dict.SetStr("lookup_error", &object.BuiltinFunc{Name: "lookup_error", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "lookup_error() requires name")
		}
		name, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "lookup_error() name must be str")
		}
		if fn, ok2 := errorRegistry[name.V]; ok2 {
			return fn, nil
		}
		return nil, object.Errorf(i.lookupErr, "unknown error handler name '%s'", name.V)
	}})

	// --- CodecInfo namedtuple-like class ---
	codecInfoCls := &object.Class{Name: "CodecInfo", Dict: object.NewDict()}

	makeCodecInfo := func(name string) *object.Instance {
		ci := &object.Instance{Class: codecInfoCls, Dict: object.NewDict()}
		ci.Dict.SetStr("name", &object.Str{V: name})

		ci.Dict.SetStr("encode", &object.BuiltinFunc{Name: "encode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "encode() missing input")
			}
			errMode := "strict"
			if kw != nil {
				if v, ok := kw.GetStr("errors"); ok {
					if s, ok2 := v.(*object.Str); ok2 {
						errMode = s.V
					}
				}
			}
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					errMode = s.V
				}
			}
			result, err := i.codecsEncode(a[0], name, errMode)
			if err != nil {
				return nil, err
			}
			var sz int
			switch v := a[0].(type) {
			case *object.Str:
				sz = len([]rune(v.V))
			case *object.Bytes:
				sz = len(v.V)
			}
			return &object.Tuple{V: []object.Object{result, object.NewInt(int64(sz))}}, nil
		}})

		ci.Dict.SetStr("decode", &object.BuiltinFunc{Name: "decode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "decode() missing input")
			}
			errMode := "strict"
			if kw != nil {
				if v, ok := kw.GetStr("errors"); ok {
					if s, ok2 := v.(*object.Str); ok2 {
						errMode = s.V
					}
				}
			}
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					errMode = s.V
				}
			}
			result, err := i.codecsDecode(a[0], name, errMode)
			if err != nil {
				return nil, err
			}
			var sz int
			switch v := a[0].(type) {
			case *object.Bytes:
				sz = len(v.V)
			case *object.Bytearray:
				sz = len(v.V)
			}
			return &object.Tuple{V: []object.Object{result, object.NewInt(int64(sz))}}, nil
		}})

		return ci
	}

	// --- lookup ---
	m.Dict.SetStr("lookup", &object.BuiltinFunc{Name: "lookup", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "lookup() requires encoding name")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "lookup() argument must be str")
		}
		norm := normaliseEncoding(s.V)
		switch norm {
		case "utf-8", "ascii", "latin-1", "utf-16", "utf-16-be", "utf-16-le",
			"utf-32", "utf-32-be", "utf-32-le", "hex_codec", "base64_codec", "rot_13":
			return makeCodecInfo(norm), nil
		}
		return nil, object.Errorf(i.lookupErr, "unknown encoding: %s", s.V)
	}})

	// --- encode ---
	m.Dict.SetStr("encode", &object.BuiltinFunc{Name: "encode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "encode() missing obj")
		}
		enc := "utf-8"
		errMode := "strict"
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				enc = s.V
			}
		}
		if len(a) >= 3 {
			if s, ok := a[2].(*object.Str); ok {
				errMode = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("encoding"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					enc = s.V
				}
			}
			if v, ok := kw.GetStr("errors"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					errMode = s.V
				}
			}
		}
		return i.codecsEncode(a[0], enc, errMode)
	}})

	// --- decode ---
	m.Dict.SetStr("decode", &object.BuiltinFunc{Name: "decode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "decode() missing obj")
		}
		enc := "utf-8"
		errMode := "strict"
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				enc = s.V
			}
		}
		if len(a) >= 3 {
			if s, ok := a[2].(*object.Str); ok {
				errMode = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("encoding"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					enc = s.V
				}
			}
			if v, ok := kw.GetStr("errors"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					errMode = s.V
				}
			}
		}
		return i.codecsDecode(a[0], enc, errMode)
	}})

	// --- getencoder / getdecoder ---
	m.Dict.SetStr("getencoder", &object.BuiltinFunc{Name: "getencoder", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "getencoder() requires encoding name")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "getencoder() argument must be str")
		}
		encName := s.V
		ci := makeCodecInfo(normaliseEncoding(encName))
		fn, _ := ci.Dict.GetStr("encode")
		return fn, nil
	}})

	m.Dict.SetStr("getdecoder", &object.BuiltinFunc{Name: "getdecoder", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "getdecoder() requires encoding name")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "getdecoder() argument must be str")
		}
		encName := s.V
		ci := makeCodecInfo(normaliseEncoding(encName))
		fn, _ := ci.Dict.GetStr("decode")
		return fn, nil
	}})

	// --- iterencode ---
	m.Dict.SetStr("iterencode", &object.BuiltinFunc{Name: "iterencode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "iterencode() requires iterator and encoding")
		}
		enc := ""
		if s, ok := a[1].(*object.Str); ok {
			enc = s.V
		}
		errMode := "strict"
		if len(a) >= 3 {
			if s, ok := a[2].(*object.Str); ok {
				errMode = s.V
			}
		}
		// Collect items from iterator
		var items []object.Object
		switch v := a[0].(type) {
		case *object.List:
			items = v.V
		case *object.Tuple:
			items = v.V
		default:
			items = []object.Object{a[0]}
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(items) {
				return nil, false, nil
			}
			result, err := i.codecsEncode(items[idx], enc, errMode)
			if err != nil {
				return nil, false, err
			}
			idx++
			return result, true, nil
		}}, nil
	}})

	// --- iterdecode ---
	m.Dict.SetStr("iterdecode", &object.BuiltinFunc{Name: "iterdecode", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "iterdecode() requires iterator and encoding")
		}
		enc := ""
		if s, ok := a[1].(*object.Str); ok {
			enc = s.V
		}
		errMode := "strict"
		if len(a) >= 3 {
			if s, ok := a[2].(*object.Str); ok {
				errMode = s.V
			}
		}
		var items []object.Object
		switch v := a[0].(type) {
		case *object.List:
			items = v.V
		case *object.Tuple:
			items = v.V
		default:
			items = []object.Object{a[0]}
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(items) {
				return nil, false, nil
			}
			result, err := i.codecsDecode(items[idx], enc, errMode)
			if err != nil {
				return nil, false, err
			}
			idx++
			return result, true, nil
		}}, nil
	}})

	// --- charmap_build ---
	m.Dict.SetStr("charmap_build", &object.BuiltinFunc{Name: "charmap_build", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "charmap_build() requires a string")
		}
		s, ok := a[0].(*object.Str)
		if !ok {
			return nil, object.Errorf(i.typeErr, "charmap_build() argument must be str")
		}
		d := object.NewDict()
		for idx, r := range s.V {
			key := &object.Int{}
			key.V.SetInt64(int64(r))
			_ = d.Set(key, object.NewInt(int64(idx)))
		}
		return d, nil
	}})

	// --- register (no-op) ---
	m.Dict.SetStr("register", &object.BuiltinFunc{Name: "register", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// --- unregister (no-op) ---
	m.Dict.SetStr("unregister", &object.BuiltinFunc{Name: "unregister", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	m.Dict.SetStr("CodecInfo", codecInfoCls)

	return m
}

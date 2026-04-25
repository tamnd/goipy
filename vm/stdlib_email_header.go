package vm

import (
	"encoding/base64"
	"mime/quotedprintable"
	"strings"

	"github.com/tamnd/goipy/object"
)

// ── email.header submodule ────────────────────────────────────────────────────

func (i *Interp) buildEmailHeader() *object.Module {
	m := &object.Module{Name: "email.header", Dict: object.NewDict()}

	// decode_header: decode RFC 2047 encoded words.
	// Returns list of (decoded_bytes_or_str, charset_or_None) tuples.
	m.Dict.SetStr("decode_header", &object.BuiltinFunc{Name: "decode_header",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{}, nil
			}
			header := object.Str_(a[0])
			return decodeRFC2047Header(header), nil
		}})

	// Header class
	headerCls := &object.Class{Name: "Header", Dict: object.NewDict()}
	headerCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			self.Dict.SetStr("_chunks", &object.List{})
			if len(a) >= 2 && a[1] != object.None {
				s := &object.Tuple{V: []object.Object{a[1], object.None}}
				if len(a) >= 3 {
					s.V[1] = a[2]
				}
				self.Dict.GetStr("_chunks")
				lst, _ := self.Dict.GetStr("_chunks")
				lst.(*object.List).V = append(lst.(*object.List).V, s)
			}
			return object.None, nil
		}})
	headerCls.Dict.SetStr("append", &object.BuiltinFunc{Name: "append",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			charset := object.Object(object.None)
			if len(a) >= 3 {
				charset = a[2]
			}
			lst, _ := self.Dict.GetStr("_chunks")
			if l, ok := lst.(*object.List); ok {
				l.V = append(l.V, &object.Tuple{V: []object.Object{a[1], charset}})
			}
			return object.None, nil
		}})
	headerCls.Dict.SetStr("encode", &object.BuiltinFunc{Name: "encode",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{}, nil
			}
			self := a[0].(*object.Instance)
			lst, _ := self.Dict.GetStr("_chunks")
			if l, ok := lst.(*object.List); ok {
				var parts []string
				for _, item := range l.V {
					t, ok2 := item.(*object.Tuple)
					if !ok2 || len(t.V) < 1 {
						continue
					}
					parts = append(parts, object.Str_(t.V[0]))
				}
				return &object.Str{V: strings.Join(parts, " ")}, nil
			}
			return &object.Str{}, nil
		}})
	headerCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{}, nil
			}
			self := a[0].(*object.Instance)
			lst, _ := self.Dict.GetStr("_chunks")
			if l, ok := lst.(*object.List); ok {
				var parts []string
				for _, item := range l.V {
					t, ok2 := item.(*object.Tuple)
					if !ok2 || len(t.V) < 1 {
						continue
					}
					parts = append(parts, object.Str_(t.V[0]))
				}
				return &object.Str{V: strings.Join(parts, " ")}, nil
			}
			return &object.Str{}, nil
		}})

	m.Dict.SetStr("Header", headerCls)

	// make_header: reverse of decode_header
	m.Dict.SetStr("make_header", &object.BuiltinFunc{Name: "make_header",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			inst := &object.Instance{Class: headerCls, Dict: object.NewDict()}
			inst.Dict.SetStr("_chunks", &object.List{})
			if len(a) >= 1 {
				if lst, ok := a[0].(*object.List); ok {
					chunks, _ := inst.Dict.GetStr("_chunks")
					cl := chunks.(*object.List)
					for _, item := range lst.V {
						if t, ok2 := item.(*object.Tuple); ok2 {
							cl.V = append(cl.V, t)
						}
					}
				}
			}
			return inst, nil
		}})

	return m
}

// decodeRFC2047Header decodes RFC 2047 encoded words from a header string.
// Returns *object.List of (bytes_or_str, charset) tuples.
func decodeRFC2047Header(header string) *object.List {
	var out []object.Object
	remaining := header
	for {
		start := strings.Index(remaining, "=?")
		if start < 0 {
			// Plain text remainder
			if plain := strings.TrimSpace(remaining); plain != "" {
				out = append(out, &object.Tuple{V: []object.Object{
					&object.Str{V: plain}, object.None,
				}})
			}
			break
		}
		// Plain text before encoded word
		if start > 0 {
			plain := strings.TrimSpace(remaining[:start])
			if plain != "" {
				out = append(out, &object.Tuple{V: []object.Object{
					&object.Str{V: plain}, object.None,
				}})
			}
		}
		// Find end of encoded word
		rest := remaining[start+2:]
		end := strings.Index(rest, "?=")
		if end < 0 {
			// Malformed; treat rest as plain
			out = append(out, &object.Tuple{V: []object.Object{
				&object.Str{V: remaining[start:]}, object.None,
			}})
			break
		}
		encoded := rest[:end]
		remaining = rest[end+2:]

		// Parse =?charset?encoding?text?=
		parts := strings.SplitN(encoded, "?", 3)
		if len(parts) != 3 {
			out = append(out, &object.Tuple{V: []object.Object{
				&object.Str{V: "=?" + encoded + "?="}, object.None,
			}})
			continue
		}
		charset := parts[0]
		encoding := strings.ToLower(parts[1])
		text := parts[2]

		var decoded []byte
		switch encoding {
		case "b":
			// Base64
			dec, err := base64.StdEncoding.DecodeString(text)
			if err != nil {
				dec, _ = base64.RawStdEncoding.DecodeString(text)
			}
			decoded = dec
		case "q":
			// Quoted-printable (with _ as space)
			text = strings.ReplaceAll(text, "_", " ")
			r := quotedprintable.NewReader(strings.NewReader(text))
			buf := make([]byte, 4096)
			n, _ := r.Read(buf)
			decoded = buf[:n]
		default:
			decoded = []byte(text)
		}
		out = append(out, &object.Tuple{V: []object.Object{
			&object.Bytes{V: decoded}, &object.Str{V: charset},
		}})
	}
	if len(out) == 0 {
		out = []object.Object{&object.Tuple{V: []object.Object{
			&object.Str{V: header}, object.None,
		}}}
	}
	return &object.List{V: out}
}

// ── email.encoders submodule ──────────────────────────────────────────────────

func (i *Interp) buildEmailEncoders() *object.Module {
	m := &object.Module{Name: "email.encoders", Dict: object.NewDict()}

	m.Dict.SetStr("encode_base64", &object.BuiltinFunc{Name: "encode_base64",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			msg, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			pl, ok2 := msg.Dict.GetStr("_payload")
			if !ok2 {
				return object.None, nil
			}
			var raw []byte
			switch v := pl.(type) {
			case *object.Bytes:
				raw = v.V
			case *object.Str:
				raw = []byte(v.V)
			default:
				return object.None, nil
			}
			encoded := base64.StdEncoding.EncodeToString(raw)
			msg.Dict.SetStr("_payload", &object.Str{V: encoded})
			emailHeaderDel(msg, "Content-Transfer-Encoding")
			emailHeaderSet(msg, "Content-Transfer-Encoding", "base64")
			return object.None, nil
		}})

	m.Dict.SetStr("encode_quopri", &object.BuiltinFunc{Name: "encode_quopri",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			msg, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			pl, ok2 := msg.Dict.GetStr("_payload")
			if !ok2 {
				return object.None, nil
			}
			var raw []byte
			switch v := pl.(type) {
			case *object.Bytes:
				raw = v.V
			case *object.Str:
				raw = []byte(v.V)
			default:
				return object.None, nil
			}
			var sb strings.Builder
			w := quotedprintable.NewWriter(&sb)
			w.Write(raw) //nolint
			w.Close()    //nolint
			msg.Dict.SetStr("_payload", &object.Str{V: sb.String()})
			emailHeaderDel(msg, "Content-Transfer-Encoding")
			emailHeaderSet(msg, "Content-Transfer-Encoding", "quoted-printable")
			return object.None, nil
		}})

	m.Dict.SetStr("encode_7or8bit", &object.BuiltinFunc{Name: "encode_7or8bit",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			msg, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			pl, _ := msg.Dict.GetStr("_payload")
			is7bit := true
			var raw []byte
			switch v := pl.(type) {
			case *object.Bytes:
				raw = v.V
			case *object.Str:
				raw = []byte(v.V)
			}
			for _, b := range raw {
				if b > 127 {
					is7bit = false
					break
				}
			}
			cte := "7bit"
			if !is7bit {
				cte = "8bit"
			}
			emailHeaderDel(msg, "Content-Transfer-Encoding")
			emailHeaderSet(msg, "Content-Transfer-Encoding", cte)
			return object.None, nil
		}})

	m.Dict.SetStr("encode_noop", &object.BuiltinFunc{Name: "encode_noop",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})

	return m
}

// ── email.generator submodule ─────────────────────────────────────────────────

func (i *Interp) buildEmailGenerator() *object.Module {
	m := &object.Module{Name: "email.generator", Dict: object.NewDict()}

	buildGeneratorClass := func(name string, asBytes bool) *object.Class {
		cls := &object.Class{Name: name, Dict: object.NewDict()}
		cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) < 2 {
					return object.None, nil
				}
				self := a[0].(*object.Instance)
				self.Dict.SetStr("_fp", a[1])
				return object.None, nil
			}})
		cls.Dict.SetStr("flatten", &object.BuiltinFunc{Name: "flatten",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) < 2 {
					return object.None, nil
				}
				self := a[0].(*object.Instance)
				msg, ok := a[1].(*object.Instance)
				if !ok {
					return object.None, nil
				}
				text := emailAsString(msg)
				fp, ok2 := self.Dict.GetStr("_fp")
				if !ok2 {
					return object.None, nil
				}
				if asBytes {
					switch f := fp.(type) {
					case *object.BytesIO:
						f.V = append(f.V, []byte(text)...)
					case *object.StringIO:
						f.V = append(f.V, []byte(text)...)
					}
				} else {
					switch f := fp.(type) {
					case *object.StringIO:
						f.V = append(f.V, []byte(text)...)
					case *object.BytesIO:
						f.V = append(f.V, []byte(text)...)
					}
				}
				return object.None, nil
			}})
		cls.Dict.SetStr("write", &object.BuiltinFunc{Name: "write",
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) < 2 {
					return object.None, nil
				}
				self := a[0].(*object.Instance)
				fp, ok := self.Dict.GetStr("_fp")
				if !ok {
					return object.None, nil
				}
				s := object.Str_(a[1])
				if sio, ok2 := fp.(*object.StringIO); ok2 {
					sio.V = append(sio.V, []byte(s)...)
				}
				return object.None, nil
			}})
		cls.Dict.SetStr("clone", &object.BuiltinFunc{Name: "clone",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) < 2 {
					return object.None, nil
				}
				inst := &object.Instance{Class: cls, Dict: object.NewDict()}
				inst.Dict.SetStr("_fp", a[1])
				return inst, nil
			}})
		return cls
	}

	generatorCls := buildGeneratorClass("Generator", false)
	bytesGeneratorCls := buildGeneratorClass("BytesGenerator", true)

	decodedGeneratorCls := &object.Class{Name: "DecodedGenerator", Dict: object.NewDict()}
	decodedGeneratorCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			a[0].(*object.Instance).Dict.SetStr("_fp", a[1])
			return object.None, nil
		}})

	m.Dict.SetStr("Generator", generatorCls)
	m.Dict.SetStr("BytesGenerator", bytesGeneratorCls)
	m.Dict.SetStr("DecodedGenerator", decodedGeneratorCls)
	return m
}

// ── email.parser submodule ────────────────────────────────────────────────────

func (i *Interp) buildEmailParser() *object.Module {
	m := &object.Module{Name: "email.parser", Dict: object.NewDict()}
	msgCls := i.getEmailMessageClass()

	buildParserClass := func(name string) *object.Class {
		cls := &object.Class{Name: name, Dict: object.NewDict()}
		cls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) < 1 {
					return object.None, nil
				}
				self := a[0].(*object.Instance)
				self.Dict.SetStr("_class", msgCls)
				if kw != nil {
					if v, ok := kw.GetStr("_class"); ok && v != object.None {
						self.Dict.SetStr("_class", v)
					}
				}
				if len(a) >= 2 && a[1] != object.None {
					self.Dict.SetStr("_class", a[1])
				}
				return object.None, nil
			}})
		cls.Dict.SetStr("parsestr", &object.BuiltinFunc{Name: "parsestr",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) < 2 {
					return nil, object.Errorf(i.typeErr, "parsestr() requires text")
				}
				self := a[0].(*object.Instance)
				text, ok := a[1].(*object.Str)
				if !ok {
					return nil, object.Errorf(i.typeErr, "parsestr() requires str")
				}
				cls := msgCls
				if v, ok2 := self.Dict.GetStr("_class"); ok2 {
					if c, ok3 := v.(*object.Class); ok3 {
						cls = c
					}
				}
				return i.emailParseString(text.V, cls)
			}})
		cls.Dict.SetStr("parsebytes", &object.BuiltinFunc{Name: "parsebytes",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) < 2 {
					return nil, object.Errorf(i.typeErr, "parsebytes() requires bytes")
				}
				self := a[0].(*object.Instance)
				var raw string
				switch v := a[1].(type) {
				case *object.Bytes:
					raw = string(v.V)
				case *object.Str:
					raw = v.V
				default:
					return nil, object.Errorf(i.typeErr, "parsebytes() requires bytes")
				}
				cls := msgCls
				if v, ok2 := self.Dict.GetStr("_class"); ok2 {
					if c, ok3 := v.(*object.Class); ok3 {
						cls = c
					}
				}
				return i.emailParseString(raw, cls)
			}})
		cls.Dict.SetStr("parse", &object.BuiltinFunc{Name: "parse",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) < 2 {
					return nil, object.Errorf(i.typeErr, "parse() requires a file")
				}
				self := a[0].(*object.Instance)
				var raw string
				switch v := a[1].(type) {
				case *object.StringIO:
					raw = string(v.V)
				case *object.BytesIO:
					raw = string(v.V)
				default:
					return nil, object.Errorf(i.typeErr, "parse(): unsupported file type")
				}
				cls := msgCls
				if v, ok2 := self.Dict.GetStr("_class"); ok2 {
					if c, ok3 := v.(*object.Class); ok3 {
						cls = c
					}
				}
				return i.emailParseString(raw, cls)
			}})
		return cls
	}

	parserCls := buildParserClass("Parser")
	bytesParserCls := buildParserClass("BytesParser")
	headerParserCls := buildParserClass("HeaderParser")
	bytesHeaderParserCls := buildParserClass("BytesHeaderParser")

	// FeedParser — incremental
	feedParserCls := &object.Class{Name: "FeedParser", Dict: object.NewDict()}
	feedParserCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			self.Dict.SetStr("_buf", &object.Str{V: ""})
			return object.None, nil
		}})
	feedParserCls.Dict.SetStr("feed", &object.BuiltinFunc{Name: "feed",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			cur := ""
			if v, ok := self.Dict.GetStr("_buf"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					cur = s.V
				}
			}
			cur += object.Str_(a[1])
			self.Dict.SetStr("_buf", &object.Str{V: cur})
			return object.None, nil
		}})
	feedParserCls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			cur := ""
			if v, ok := self.Dict.GetStr("_buf"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					cur = s.V
				}
			}
			return i.emailParseString(cur, msgCls)
		}})

	bytesFeedParserCls := &object.Class{Name: "BytesFeedParser", Bases: []*object.Class{feedParserCls}, Dict: object.NewDict()}

	m.Dict.SetStr("Parser", parserCls)
	m.Dict.SetStr("BytesParser", bytesParserCls)
	m.Dict.SetStr("HeaderParser", headerParserCls)
	m.Dict.SetStr("BytesHeaderParser", bytesHeaderParserCls)
	m.Dict.SetStr("FeedParser", feedParserCls)
	m.Dict.SetStr("BytesFeedParser", bytesFeedParserCls)

	// headerregistry stub
	m.Dict.SetStr("headerregistry", &object.Module{
		Name: "email.headerregistry", Dict: object.NewDict(),
	})

	// Convenience: expose Address and Group
	addressCls := &object.Class{Name: "Address", Dict: object.NewDict()}
	addressCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			self.Dict.SetStr("display_name", &object.Str{V: ""})
			self.Dict.SetStr("username", &object.Str{V: ""})
			self.Dict.SetStr("domain", &object.Str{V: ""})
			if kw != nil {
				dictIterStr(kw, func(k string, v object.Object) {
					self.Dict.SetStr(k, v)
				})
			}
			return object.None, nil
		}})

	return m
}

package plist

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
`
const xmlFooter = `</plist>
`

// plist datetime format (ISO 8601 with 'Z' suffix, no fractional seconds).
const plistDateFmt = "2006-01-02T15:04:05Z"

// ── XML writer ────────────────────────────────────────────────────────────────

func dumpXML(v Value, sortKeys bool) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(xmlHeader)
	if err := writeXMLValue(&b, v, sortKeys, 0); err != nil {
		return nil, err
	}
	b.WriteString(xmlFooter)
	return b.Bytes(), nil
}

func writeXMLValue(b *bytes.Buffer, v Value, sortKeys bool, depth int) error {
	indent := strings.Repeat("\t", depth)
	switch x := v.(type) {
	case bool:
		if x {
			b.WriteString(indent + "<true/>\n")
		} else {
			b.WriteString(indent + "<false/>\n")
		}
	case int64:
		b.WriteString(indent + "<integer>" + strconv.FormatInt(x, 10) + "</integer>\n")
	case uint64:
		b.WriteString(indent + "<integer>" + strconv.FormatUint(x, 10) + "</integer>\n")
	case float64:
		b.WriteString(indent + "<real>" + formatFloat(x) + "</real>\n")
	case string:
		b.WriteString(indent + "<string>" + xmlEscape(x) + "</string>\n")
	case []byte:
		encoded := base64.StdEncoding.EncodeToString(x)
		// wrap at 76 chars like Python does
		b.WriteString(indent + "<data>\n")
		for len(encoded) > 0 {
			n := 76
			if n > len(encoded) {
				n = len(encoded)
			}
			b.WriteString(indent + "\t" + encoded[:n] + "\n")
			encoded = encoded[n:]
		}
		b.WriteString(indent + "</data>\n")
	case time.Time:
		b.WriteString(indent + "<date>" + x.UTC().Format(plistDateFmt) + "</date>\n")
	case UID:
		// UID is not representable in XML plist; raise error like CPython.
		return fmt.Errorf("plistlib: UID cannot be serialized to XML plist")
	case []interface{}:
		b.WriteString(indent + "<array>\n")
		for _, item := range x {
			if err := writeXMLValue(b, item, sortKeys, depth+1); err != nil {
				return err
			}
		}
		b.WriteString(indent + "</array>\n")
	case map[string]interface{}:
		b.WriteString(indent + "<dict>\n")
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		if sortKeys {
			sort.Strings(keys)
		}
		for _, k := range keys {
			b.WriteString(indent + "\t<key>" + xmlEscape(k) + "</key>\n")
			if err := writeXMLValue(b, x[k], sortKeys, depth+1); err != nil {
				return err
			}
		}
		b.WriteString(indent + "</dict>\n")
	case nil:
		return fmt.Errorf("plistlib: cannot serialize None")
	default:
		return fmt.Errorf("plistlib: unsupported type %T", v)
	}
	return nil
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func formatFloat(f float64) string {
	if math.IsInf(f, 1) {
		return "inf"
	}
	if math.IsInf(f, -1) {
		return "-inf"
	}
	if math.IsNaN(f) {
		return "nan"
	}
	s := strconv.FormatFloat(f, 'g', -1, 64)
	return s
}

// ── XML parser ────────────────────────────────────────────────────────────────

// Minimal hand-written XML plist parser (no external deps, no encoding/xml).
// We only need to handle the small subset used by plists.

type xmlParser struct {
	data []byte
	pos  int
}

func parseXML(data []byte) (Value, error) {
	p := &xmlParser{data: data}
	// skip XML header and DOCTYPE
	if err := p.skipProlog(); err != nil {
		return nil, err
	}
	// expect <plist ...>
	tag, _, err := p.readOpenTag()
	if err != nil {
		return nil, &InvalidFileError{Msg: "missing <plist> element: " + err.Error()}
	}
	if tag != "plist" {
		return nil, &InvalidFileError{Msg: "expected <plist>, got <" + tag + ">"}
	}
	p.skipWS()
	v, err := p.readValue()
	if err != nil {
		return nil, err
	}
	p.skipWS()
	// expect </plist>
	if err2 := p.readCloseTag("plist"); err2 != nil {
		return nil, &InvalidFileError{Msg: "missing </plist>: " + err2.Error()}
	}
	return v, nil
}

func (p *xmlParser) skipProlog() error {
	for {
		p.skipWS()
		if p.pos >= len(p.data) {
			return nil
		}
		if !p.peek("<?") && !p.peek("<!") {
			return nil
		}
		// skip until closing > (handling nested <> in DOCTYPE is tricky, use bracket depth)
		if p.peek("<!") {
			depth := 0
			for p.pos < len(p.data) {
				c := p.data[p.pos]
				p.pos++
				if c == '<' {
					depth++
				} else if c == '>' {
					depth--
					if depth <= 0 {
						break
					}
				}
			}
		} else {
			// <?xml ... ?>
			for p.pos < len(p.data) {
				if p.peek("?>") {
					p.pos += 2
					break
				}
				p.pos++
			}
		}
	}
}

func (p *xmlParser) peek(s string) bool {
	return p.pos+len(s) <= len(p.data) && string(p.data[p.pos:p.pos+len(s)]) == s
}

func (p *xmlParser) skipWS() {
	for p.pos < len(p.data) && isWS(p.data[p.pos]) {
		p.pos++
	}
}

func isWS(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// readOpenTag reads <tagname attr="..." ...> and returns the tag name and attrs.
func (p *xmlParser) readOpenTag() (string, map[string]string, error) {
	p.skipWS()
	if p.pos >= len(p.data) || p.data[p.pos] != '<' {
		return "", nil, fmt.Errorf("expected '<' at pos %d", p.pos)
	}
	p.pos++ // consume '<'
	name := p.readName()
	attrs := map[string]string{}
	for {
		p.skipWS()
		if p.pos >= len(p.data) {
			return "", nil, fmt.Errorf("unexpected EOF in tag")
		}
		c := p.data[p.pos]
		if c == '>' {
			p.pos++
			return name, attrs, nil
		}
		if c == '/' {
			// self-closing />
			p.pos++
			if p.pos < len(p.data) && p.data[p.pos] == '>' {
				p.pos++
			}
			return name + "/", attrs, nil
		}
		// attribute
		attrName := p.readName()
		p.skipWS()
		if p.pos < len(p.data) && p.data[p.pos] == '=' {
			p.pos++
			p.skipWS()
			val, err := p.readAttrValue()
			if err != nil {
				return "", nil, err
			}
			attrs[attrName] = val
		}
	}
}

func (p *xmlParser) readName() string {
	start := p.pos
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if isWS(c) || c == '>' || c == '=' || c == '/' || c == '"' || c == '\'' {
			break
		}
		p.pos++
	}
	return string(p.data[start:p.pos])
}

func (p *xmlParser) readAttrValue() (string, error) {
	if p.pos >= len(p.data) {
		return "", fmt.Errorf("expected attribute value")
	}
	quote := p.data[p.pos]
	if quote != '"' && quote != '\'' {
		return "", fmt.Errorf("expected quote in attribute value")
	}
	p.pos++
	start := p.pos
	for p.pos < len(p.data) && p.data[p.pos] != quote {
		p.pos++
	}
	v := string(p.data[start:p.pos])
	if p.pos < len(p.data) {
		p.pos++ // consume closing quote
	}
	return v, nil
}

func (p *xmlParser) readCloseTag(name string) error {
	p.skipWS()
	if !p.peek("</") {
		return fmt.Errorf("expected </%s>", name)
	}
	p.pos += 2
	p.skipWS()
	n := p.readName()
	if n != name {
		return fmt.Errorf("expected </%s>, got </%s>", name, n)
	}
	p.skipWS()
	if p.pos < len(p.data) && p.data[p.pos] == '>' {
		p.pos++
	}
	return nil
}

func (p *xmlParser) readTextContent() string {
	start := p.pos
	for p.pos < len(p.data) && p.data[p.pos] != '<' {
		p.pos++
	}
	raw := string(p.data[start:p.pos])
	// decode XML entities
	raw = strings.ReplaceAll(raw, "&amp;", "&")
	raw = strings.ReplaceAll(raw, "&lt;", "<")
	raw = strings.ReplaceAll(raw, "&gt;", ">")
	raw = strings.ReplaceAll(raw, "&apos;", "'")
	raw = strings.ReplaceAll(raw, "&quot;", "\"")
	return raw
}

func (p *xmlParser) readValue() (Value, error) {
	p.skipWS()
	if p.pos >= len(p.data) {
		return nil, &InvalidFileError{Msg: "unexpected EOF"}
	}
	if !p.peek("<") {
		return nil, &InvalidFileError{Msg: fmt.Sprintf("expected '<' at pos %d", p.pos)}
	}

	tag, _, err := p.readOpenTag()
	if err != nil {
		return nil, &InvalidFileError{Msg: "bad tag: " + err.Error()}
	}

	switch tag {
	case "true/":
		return true, nil
	case "false/":
		return false, nil
	case "string":
		text := p.readTextContent()
		if err := p.readCloseTag("string"); err != nil {
			return nil, &InvalidFileError{Msg: err.Error()}
		}
		return text, nil
	case "integer":
		text := strings.TrimSpace(p.readTextContent())
		if err := p.readCloseTag("integer"); err != nil {
			return nil, &InvalidFileError{Msg: err.Error()}
		}
		// try signed first, then unsigned
		if iv, e := strconv.ParseInt(text, 10, 64); e == nil {
			return iv, nil
		}
		if uv, e := strconv.ParseUint(text, 10, 64); e == nil {
			return uv, nil
		}
		return nil, &InvalidFileError{Msg: "invalid integer: " + text}
	case "real":
		text := strings.TrimSpace(p.readTextContent())
		if err := p.readCloseTag("real"); err != nil {
			return nil, &InvalidFileError{Msg: err.Error()}
		}
		f, e := strconv.ParseFloat(text, 64)
		if e != nil {
			return nil, &InvalidFileError{Msg: "invalid real: " + text}
		}
		return f, nil
	case "data":
		raw := p.readTextContent()
		if err := p.readCloseTag("data"); err != nil {
			return nil, &InvalidFileError{Msg: err.Error()}
		}
		// strip whitespace from base64
		raw = strings.Join(strings.Fields(raw), "")
		decoded, e := base64.StdEncoding.DecodeString(raw)
		if e != nil {
			return nil, &InvalidFileError{Msg: "invalid base64 data: " + e.Error()}
		}
		return decoded, nil
	case "date":
		text := strings.TrimSpace(p.readTextContent())
		if err := p.readCloseTag("date"); err != nil {
			return nil, &InvalidFileError{Msg: err.Error()}
		}
		t, e := time.Parse(plistDateFmt, text)
		if e != nil {
			return nil, &InvalidFileError{Msg: "invalid date: " + text}
		}
		return t.UTC(), nil
	case "array":
		var items []interface{}
		for {
			p.skipWS()
			if p.peek("</") {
				break
			}
			item, err := p.readValue()
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		if items == nil {
			items = []interface{}{}
		}
		if err := p.readCloseTag("array"); err != nil {
			return nil, &InvalidFileError{Msg: err.Error()}
		}
		return items, nil
	case "dict":
		m := map[string]interface{}{}
		for {
			p.skipWS()
			if p.peek("</") {
				break
			}
			// read <key>...</key>
			keyTag, _, err := p.readOpenTag()
			if err != nil {
				return nil, &InvalidFileError{Msg: "dict: " + err.Error()}
			}
			if keyTag != "key" {
				return nil, &InvalidFileError{Msg: "dict: expected <key>, got <" + keyTag + ">"}
			}
			keyText := p.readTextContent()
			keyText = strings.ReplaceAll(keyText, "&amp;", "&")
			keyText = strings.ReplaceAll(keyText, "&lt;", "<")
			keyText = strings.ReplaceAll(keyText, "&gt;", ">")
			if err := p.readCloseTag("key"); err != nil {
				return nil, &InvalidFileError{Msg: err.Error()}
			}
			// read value
			val, err := p.readValue()
			if err != nil {
				return nil, err
			}
			m[keyText] = val
		}
		if err := p.readCloseTag("dict"); err != nil {
			return nil, &InvalidFileError{Msg: err.Error()}
		}
		return m, nil
	default:
		return nil, &InvalidFileError{Msg: "unknown plist tag: <" + tag + ">"}
	}
}

package vm

import (
	"fmt"
	"strings"
	"sync"

	"github.com/tamnd/goipy/object"
)

// buildHtmlEntities returns the html.entities submodule.
func (i *Interp) buildHtmlEntities() *object.Module {
	m := &object.Module{Name: "html.entities", Dict: object.NewDict()}

	// name2codepoint: dict mapping entity name (without &;) to codepoint int
	n2cp := object.NewDict()
	for name, val := range htmlEntities {
		if len([]rune(val)) == 1 {
			cp := int64([]rune(val)[0])
			n2cp.SetStr(name, object.IntFromInt64(cp))
		}
	}
	m.Dict.SetStr("name2codepoint", n2cp)

	// codepoint2name: reverse mapping — for duplicates, last writer wins, then
	// we override with known-canonical names.
	cp2n := object.NewDict()
	// Use Items() to iterate all entries in n2cp.
	n2cpKeys, n2cpVals := n2cp.Items()
	for idx, k := range n2cpKeys {
		ks, ok := k.(*object.Str)
		if !ok {
			continue
		}
		_ = ks
		cp2n.Set(n2cpVals[idx], &object.Str{V: ks.V})
	}
	// Override known conflicts with canonical CPython names
	overrides := map[int64]string{
		38: "amp", 60: "lt", 62: "gt", 34: "quot",
	}
	for cp, name := range overrides {
		cp2n.Set(object.IntFromInt64(cp), &object.Str{V: name})
	}
	m.Dict.SetStr("codepoint2name", cp2n)

	// html5: dict with semicolon-terminated keys → unicode string
	// We use our htmlEntities table (all HTML4 entries) with ";" suffix.
	h5 := object.NewDict()
	for name, val := range htmlEntities {
		h5.SetStr(name+";", &object.Str{V: val})
	}
	m.Dict.SetStr("html5", h5)

	return m
}

// htmlParserState holds the mutable state for one HTMLParser instance.
type htmlParserState struct {
	buf             string
	line            int
	col             int
	lastStartTag    string
	convertCharrefs bool
}

// buildHtmlParser returns the html.parser submodule containing HTMLParser class.
func (i *Interp) buildHtmlParser() *object.Module {
	m := &object.Module{Name: "html.parser", Dict: object.NewDict()}

	htmlParserCls := &object.Class{Name: "HTMLParser", Dict: object.NewDict()}

	// __init__
	htmlParserCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		convertCharrefs := true
		if kw != nil {
			if v, ok2 := kw.GetStr("convert_charrefs"); ok2 {
				if b, ok3 := v.(*object.Bool); ok3 {
					convertCharrefs = b.V
				}
			}
		}
		state := &htmlParserState{
			buf:             "",
			line:            1,
			col:             0,
			convertCharrefs: convertCharrefs,
		}
		self.Dict.SetStr("convert_charrefs", boolObj(convertCharrefs))
		// Store state in the registry keyed by instance pointer.
		htmlParserStateMap.Store(self, state)
		return object.None, nil
	}})

	// reset
	htmlParserCls.Dict.SetStr("reset", &object.BuiltinFunc{Name: "reset", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		if st := getHtmlParserState(self); st != nil {
			st.buf = ""
			st.line = 1
			st.col = 0
			st.lastStartTag = ""
		}
		return object.None, nil
	}})

	// getpos
	htmlParserCls.Dict.SetStr("getpos", &object.BuiltinFunc{Name: "getpos", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Tuple{V: []object.Object{object.IntFromInt64(1), object.IntFromInt64(0)}}, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return &object.Tuple{V: []object.Object{object.IntFromInt64(1), object.IntFromInt64(0)}}, nil
		}
		st := getHtmlParserState(self)
		if st == nil {
			return &object.Tuple{V: []object.Object{object.IntFromInt64(1), object.IntFromInt64(0)}}, nil
		}
		return &object.Tuple{V: []object.Object{
			object.IntFromInt64(int64(st.line)),
			object.IntFromInt64(int64(st.col)),
		}}, nil
	}})

	// get_starttag_text
	htmlParserCls.Dict.SetStr("get_starttag_text", &object.BuiltinFunc{Name: "get_starttag_text", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		st := getHtmlParserState(self)
		if st == nil || st.lastStartTag == "" {
			return object.None, nil
		}
		return &object.Str{V: st.lastStartTag}, nil
	}})

	// error (deprecated)
	htmlParserCls.Dict.SetStr("error", &object.BuiltinFunc{Name: "error", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		msg := "html parser error"
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				msg = s.V
			}
		}
		return nil, fmt.Errorf("AssertionError: %s", msg)
	}})

	// Default no-op callback stubs. User subclasses override these.
	for _, name := range []string{
		"handle_starttag", "handle_endtag", "handle_startendtag",
		"handle_data", "handle_comment", "handle_entityref",
		"handle_charref", "handle_decl", "handle_pi", "unknown_decl",
	} {
		n := name
		htmlParserCls.Dict.SetStr(n, &object.BuiltinFunc{Name: n, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	}

	// handle_startendtag: default calls handle_starttag then handle_endtag
	htmlParserCls.Dict.SetStr("handle_startendtag", &object.BuiltinFunc{Name: "handle_startendtag", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		ii := interp.(*Interp)
		self := a[0]
		tag := a[1]
		attrs := a[2]
		if fn, err := ii.getAttr(self, "handle_starttag"); err == nil {
			if _, err2 := ii.callObject(fn, []object.Object{tag, attrs}, nil); err2 != nil {
				return nil, err2
			}
		}
		if fn, err := ii.getAttr(self, "handle_endtag"); err == nil {
			if _, err2 := ii.callObject(fn, []object.Object{tag}, nil); err2 != nil {
				return nil, err2
			}
		}
		return object.None, nil
	}})

	// close — flush remaining buffer as raw data
	htmlParserCls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		ii := interp.(*Interp)
		st := getHtmlParserState(self)
		if st == nil || st.buf == "" {
			return object.None, nil
		}
		// Flush any remaining buffer as raw data.
		remainder := st.buf
		st.buf = ""
		if remainder != "" {
			convertCharrefs := st.convertCharrefs
			if err := htmlFireText(ii, self, remainder, convertCharrefs); err != nil {
				return nil, err
			}
		}
		return object.None, nil
	}})

	// feed — the main parsing method
	htmlParserCls.Dict.SetStr("feed", &object.BuiltinFunc{Name: "feed", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		data, ok2 := a[1].(*object.Str)
		if !ok2 {
			return object.None, nil
		}
		ii := interp.(*Interp)

		st := getHtmlParserState(self)
		if st == nil {
			// Create default state if __init__ wasn't called (rare but safe)
			st = &htmlParserState{line: 1, col: 0, convertCharrefs: true}
			htmlParserStateMap.Store(self, st)
		}

		return object.None, htmlFeed(ii, self, data.V, st)
	}})

	m.Dict.SetStr("HTMLParser", htmlParserCls)
	return m
}

// htmlParserStateMap maps *object.Instance → *htmlParserState.
// Uses sync.Map for safety across goroutine threads.
var htmlParserStateMap htmlStateRegistry

type htmlStateRegistry struct {
	m sync.Map
}

func (r *htmlStateRegistry) Store(k *object.Instance, v *htmlParserState) {
	r.m.Store(k, v)
}

func (r *htmlStateRegistry) Load(k *object.Instance) *htmlParserState {
	v, ok := r.m.Load(k)
	if !ok {
		return nil
	}
	return v.(*htmlParserState)
}

func getHtmlParserState(self *object.Instance) *htmlParserState {
	return htmlParserStateMap.Load(self)
}

// boolObj converts a Go bool to *object.Bool.
func boolObj(b bool) *object.Bool {
	if b {
		return object.True
	}
	return object.False
}

// htmlCallbackStr calls a named method on self with a single string argument.
func htmlCallbackStr(ii *Interp, self *object.Instance, method, arg string) error {
	fn, err := ii.getAttr(self, method)
	if err != nil {
		return nil
	}
	_, err = ii.callObject(fn, []object.Object{&object.Str{V: arg}}, nil)
	return err
}

// htmlFireText emits text data, handling convert_charrefs and entity/charref
// callbacks as appropriate.
func htmlFireText(ii *Interp, self *object.Instance, text string, convertCharrefs bool) error {
	if text == "" {
		return nil
	}
	if convertCharrefs {
		// Decode references inline and emit single handle_data.
		decoded := htmlUnescape(text)
		return htmlCallbackStr(ii, self, "handle_data", decoded)
	}
	// convert_charrefs=False: scan for & references and fire entityref/charref.
	return htmlFireTextSegments(ii, self, text)
}

// htmlFireTextSegments scans text for & references when convert_charrefs=False.
// It fires handle_entityref, handle_charref, and handle_data as appropriate.
func htmlFireTextSegments(ii *Interp, self *object.Instance, text string) error {
	for len(text) > 0 {
		amp := strings.IndexByte(text, '&')
		if amp < 0 {
			return htmlCallbackStr(ii, self, "handle_data", text)
		}
		// Emit text before '&'.
		if amp > 0 {
			if err := htmlCallbackStr(ii, self, "handle_data", text[:amp]); err != nil {
				return err
			}
		}
		text = text[amp+1:] // skip the '&'

		// Find the ';' that terminates the reference.
		semi := strings.IndexByte(text, ';')
		if semi < 0 {
			// No terminating ';' — treat '&' + rest as literal data.
			return htmlCallbackStr(ii, self, "handle_data", "&"+text)
		}
		ref := text[:semi]
		text = text[semi+1:] // skip past ';'

		if ref == "" {
			// bare '&;' — treat as data
			if err := htmlCallbackStr(ii, self, "handle_data", "&;"); err != nil {
				return err
			}
			continue
		}

		if ref[0] == '#' {
			// Numeric character reference: &#NNN; or &#xNNN;
			numPart := ref[1:]
			if err := htmlCallbackStr(ii, self, "handle_charref", numPart); err != nil {
				return err
			}
		} else {
			// Named entity reference: &name;
			if err := htmlCallbackStr(ii, self, "handle_entityref", ref); err != nil {
				return err
			}
		}
	}
	return nil
}

// htmlAdvancePos updates line/col tracking as we advance through scanned characters.
func htmlAdvancePos(st *htmlParserState, s string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			st.line++
			st.col = 0
		} else {
			st.col++
		}
	}
}

// htmlFeed parses the HTML string and calls callbacks on the Python instance.
// It appends input to st.buf and processes complete tokens, leaving any
// incomplete token in st.buf for the next feed() call.
func htmlFeed(ii *Interp, self *object.Instance, input string, st *htmlParserState) error {
	convertCharrefs := st.convertCharrefs
	st.buf += input
	s := st.buf

	pos := 0
	n := len(s)

	for pos < n {
		// Find the next '<'.
		lt := strings.IndexByte(s[pos:], '<')
		if lt < 0 {
			// All remaining text — no incomplete tag possible.
			if err := htmlFireText(ii, self, s[pos:], convertCharrefs); err != nil {
				return err
			}
			htmlAdvancePos(st, s[pos:])
			pos = n
			break
		}

		// Emit text before '<'.
		if lt > 0 {
			chunk := s[pos : pos+lt]
			if err := htmlFireText(ii, self, chunk, convertCharrefs); err != nil {
				return err
			}
			htmlAdvancePos(st, chunk)
			pos += lt
		}

		// pos is now at '<'.
		rest := s[pos:]

		// Try to consume a complete token.
		consumed, err := htmlTryConsumeToken(ii, self, rest, st, convertCharrefs)
		if err != nil {
			return err
		}
		if consumed < 0 {
			// Incomplete token — stop; leave from pos onwards in buffer.
			break
		}
		pos += consumed
	}

	st.buf = s[pos:]
	return nil
}

// htmlTryConsumeToken tries to parse and handle a single HTML token starting
// at '<' in s.  Returns the number of bytes consumed, or -1 if the token
// is incomplete (no closing delimiter found yet).
func htmlTryConsumeToken(ii *Interp, self *object.Instance, s string, st *htmlParserState, convertCharrefs bool) (int, error) {
	// s[0] == '<' guaranteed by caller.

	// Comment: <!-- ... -->
	if strings.HasPrefix(s, "<!--") {
		end := strings.Index(s[4:], "-->")
		if end < 0 {
			return -1, nil // incomplete
		}
		comment := s[4 : 4+end]
		tokenRaw := s[:4+end+3]
		htmlAdvancePos(st, tokenRaw)
		if err := htmlCallbackStr(ii, self, "handle_comment", comment); err != nil {
			return 0, err
		}
		return 4 + end + 3, nil
	}

	// Processing instruction: <?...?>
	if strings.HasPrefix(s, "<?") {
		end := strings.Index(s[2:], "?>")
		if end < 0 {
			return -1, nil // incomplete
		}
		piData := s[2 : 2+end]
		tokenRaw := s[:2+end+2]
		htmlAdvancePos(st, tokenRaw)
		if err := htmlCallbackStr(ii, self, "handle_pi", piData); err != nil {
			return 0, err
		}
		return 2 + end + 2, nil
	}

	// CDATA section: <![CDATA[...]]>  (or any <![...]]>)
	if strings.HasPrefix(s, "<![") {
		end := strings.Index(s[3:], "]]>")
		if end < 0 {
			return -1, nil // incomplete
		}
		// unknown_decl receives content between "<![" and "]]>"
		cdataContent := s[3 : 3+end]
		// CPython passes "CDATA[content" for <![CDATA[content]]>
		// but more generally passes everything between "<![" and "]]>".
		tokenRaw := s[:3+end+3]
		htmlAdvancePos(st, tokenRaw)
		if err := htmlCallbackStr(ii, self, "unknown_decl", cdataContent); err != nil {
			return 0, err
		}
		return 3 + end + 3, nil
	}

	// Declaration: <!...>  (DOCTYPE, etc.) — but not <!--
	if strings.HasPrefix(s, "<!") && len(s) > 2 && s[2] != '-' {
		end := strings.IndexByte(s[2:], '>')
		if end < 0 {
			return -1, nil // incomplete
		}
		declContent := s[2 : 2+end]
		tokenRaw := s[:2+end+1]
		htmlAdvancePos(st, tokenRaw)
		if isStandardDecl(declContent) {
			if err := htmlCallbackStr(ii, self, "handle_decl", declContent); err != nil {
				return 0, err
			}
		} else {
			if err := htmlCallbackStr(ii, self, "unknown_decl", declContent); err != nil {
				return 0, err
			}
		}
		return 2 + end + 1, nil
	}

	// End tag: </tag>
	if strings.HasPrefix(s, "</") {
		end := strings.IndexByte(s[2:], '>')
		if end < 0 {
			return -1, nil // incomplete
		}
		tagName := strings.ToLower(strings.TrimSpace(s[2 : 2+end]))
		tokenRaw := s[:2+end+1]
		htmlAdvancePos(st, tokenRaw)
		fn, err := ii.getAttr(self, "handle_endtag")
		if err == nil {
			if _, err2 := ii.callObject(fn, []object.Object{&object.Str{V: tagName}}, nil); err2 != nil {
				return 0, err2
			}
		}
		return 2 + end + 1, nil
	}

	// Start tag or self-closing: <tag ...> or <tag .../>
	// Tag name must start with a letter, '_', or ':' (not digit per spec,
	// but tags like <h1> start with 'h' which is a letter).
	if len(s) > 1 && (isLetter(s[1]) || s[1] == '_' || s[1] == ':') {
		tagEnd, selfClose, rawTag, tagName, attrs, ok := parseStartTag(s)
		if !ok {
			// Incomplete — no closing '>'.
			return -1, nil
		}
		if tagEnd < 0 {
			// Malformed — treat '<' as literal data.
			return 1, htmlFireText(ii, self, "<", convertCharrefs)
		}

		// Update position tracking to start of this tag.
		st.lastStartTag = rawTag

		// Build attrs list: boolean attrs get None value.
		attrsList := make([]object.Object, len(attrs))
		for idx, av := range attrs {
			var valObj object.Object
			if av[1] == "\x00" {
				// Sentinel for boolean attribute.
				valObj = object.None
			} else {
				attrVal := av[1]
				if convertCharrefs {
					attrVal = htmlUnescape(attrVal)
				}
				valObj = &object.Str{V: attrVal}
			}
			attrsList[idx] = &object.Tuple{V: []object.Object{
				&object.Str{V: av[0]},
				valObj,
			}}
		}
		attrsObj := &object.List{V: attrsList}

		// Update line/col to position of this tag.
		htmlAdvancePos(st, rawTag)

		if selfClose {
			fn, err := ii.getAttr(self, "handle_startendtag")
			if err == nil {
				if _, err2 := ii.callObject(fn, []object.Object{
					&object.Str{V: tagName},
					attrsObj,
				}, nil); err2 != nil {
					return 0, err2
				}
			}
		} else {
			fn, err := ii.getAttr(self, "handle_starttag")
			if err == nil {
				if _, err2 := ii.callObject(fn, []object.Object{
					&object.Str{V: tagName},
					attrsObj,
				}, nil); err2 != nil {
					return 0, err2
				}
			}
		}
		return tagEnd, nil
	}

	// Not a recognized tag start — treat '<' as literal data.
	if err := htmlFireText(ii, self, "<", convertCharrefs); err != nil {
		return 0, err
	}
	htmlAdvancePos(st, "<")
	return 1, nil
}

// isStandardDecl returns true if a declaration content looks like DOCTYPE or similar.
func isStandardDecl(s string) bool {
	up := strings.ToUpper(strings.TrimSpace(s))
	return strings.HasPrefix(up, "DOCTYPE") ||
		strings.HasPrefix(up, "ELEMENT") ||
		strings.HasPrefix(up, "ATTLIST") ||
		strings.HasPrefix(up, "NOTATION") ||
		strings.HasPrefix(up, "ENTITY")
}

// isLetter returns true if b is an ASCII letter.
func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// parseStartTag parses a start tag starting with '<' in s.
// Returns (consumed, selfClose, rawTag, tagName, attrs, ok) where:
//   - consumed == -1 and ok == true  → malformed (not a valid tag)
//   - ok == false                    → incomplete (no closing '>' yet)
//   - ok == true and consumed >= 0   → successfully parsed
//
// Boolean attributes are encoded with value "\x00" (NUL sentinel).
func parseStartTag(s string) (consumed int, selfClose bool, rawTag, tagName string, attrs [][2]string, ok bool) {
	// s starts with '<'
	if len(s) < 2 {
		return -1, false, "", "", nil, false // incomplete
	}
	i := 1
	n := len(s)

	// tag name: letters, digits, '_', '-', ':', '.'
	for i < n && !isSpace(s[i]) && s[i] != '>' && s[i] != '/' {
		i++
	}
	if i >= n {
		return -1, false, "", "", nil, false // incomplete — no '>' seen yet
	}
	tagName = strings.ToLower(s[1:i])
	if tagName == "" {
		return -1, false, "", "", nil, true // malformed
	}

	// parse attributes
	for i < n {
		// skip whitespace
		for i < n && isSpace(s[i]) {
			i++
		}
		if i >= n {
			return -1, false, "", "", nil, false // incomplete
		}
		if s[i] == '>' {
			i++
			return i, false, s[:i], tagName, attrs, true
		}
		if s[i] == '/' {
			i++
			if i >= n {
				return -1, false, "", "", nil, false // incomplete
			}
			if s[i] == '>' {
				i++
				return i, true, s[:i], tagName, attrs, true
			}
			// lone '/' inside tag — skip
			continue
		}
		// attribute name
		attrStart := i
		for i < n && s[i] != '=' && s[i] != '>' && s[i] != '/' && !isSpace(s[i]) {
			i++
		}
		attrName := strings.ToLower(s[attrStart:i])
		if attrName == "" {
			if i >= n {
				return -1, false, "", "", nil, false // incomplete
			}
			i++ // skip unknown char
			continue
		}
		// skip whitespace
		for i < n && isSpace(s[i]) {
			i++
		}
		if i >= n {
			// boolean attr but tag not closed yet
			return -1, false, "", "", nil, false
		}
		if s[i] != '=' {
			// boolean attribute — use NUL sentinel so caller can set None
			attrs = append(attrs, [2]string{attrName, "\x00"})
			continue
		}
		i++ // skip '='
		// skip whitespace
		for i < n && isSpace(s[i]) {
			i++
		}
		if i >= n {
			return -1, false, "", "", nil, false // incomplete
		}
		var attrVal string
		if s[i] == '"' {
			i++
			end := strings.IndexByte(s[i:], '"')
			if end < 0 {
				return -1, false, "", "", nil, false // incomplete — no closing quote
			}
			attrVal = s[i : i+end]
			i = i + end + 1
		} else if s[i] == '\'' {
			i++
			end := strings.IndexByte(s[i:], '\'')
			if end < 0 {
				return -1, false, "", "", nil, false // incomplete
			}
			attrVal = s[i : i+end]
			i = i + end + 1
		} else {
			// unquoted value
			start := i
			for i < n && !isSpace(s[i]) && s[i] != '>' && s[i] != '/' {
				i++
			}
			attrVal = s[start:i]
		}
		attrs = append(attrs, [2]string{attrName, attrVal})
	}
	// Reached end of s without finding '>'
	return -1, false, "", "", nil, false // incomplete
}

// isSpace returns true if b is ASCII whitespace.
func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

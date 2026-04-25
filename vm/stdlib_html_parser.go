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
	buf            string
	line           int
	col            int
	lastStartTag   string
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

	// close
	htmlParserCls.Dict.SetStr("close", &object.BuiltinFunc{Name: "close", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// Process any buffered data. For our implementation the buffer is always
		// flushed during feed(), so this is a no-op.
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
		convertCharrefs := true
		if st != nil {
			convertCharrefs = st.convertCharrefs
		}

		return object.None, htmlFeed(ii, self, data.V, convertCharrefs, st)
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

// htmlFeed parses the HTML string and calls callbacks on the Python instance.
func htmlFeed(ii *Interp, self *object.Instance, input string, convertCharrefs bool, st *htmlParserState) error {
	pos := 0
	n := len(input)

	fireData := func(text string) error {
		if text == "" {
			return nil
		}
		if convertCharrefs {
			text = htmlUnescape(text)
		}
		fn, err := ii.getAttr(self, "handle_data")
		if err != nil {
			return nil
		}
		_, err = ii.callObject(fn, []object.Object{&object.Str{V: text}}, nil)
		return err
	}

	for pos < n {
		// Find next '<'
		lt := strings.IndexByte(input[pos:], '<')
		if lt < 0 {
			// rest is text
			return fireData(input[pos:])
		}
		// text before '<'
		if lt > 0 {
			if err := fireData(input[pos : pos+lt]); err != nil {
				return err
			}
		}
		pos += lt
		// now at '<'

		rest := input[pos:]

		// Comment: <!-- ... -->
		if strings.HasPrefix(rest, "<!--") {
			end := strings.Index(rest[4:], "-->")
			if end < 0 {
				// unterminated — treat rest as text
				if err := fireData(rest); err != nil {
					return err
				}
				pos = n
				continue
			}
			comment := rest[4 : 4+end]
			fn, err := ii.getAttr(self, "handle_comment")
			if err == nil {
				if _, err2 := ii.callObject(fn, []object.Object{&object.Str{V: comment}}, nil); err2 != nil {
					return err2
				}
			}
			pos += 4 + end + 3
			continue
		}

		// Processing instruction: <?...?>
		if strings.HasPrefix(rest, "<?") {
			end := strings.Index(rest[2:], "?>")
			if end < 0 {
				if err := fireData(rest); err != nil {
					return err
				}
				pos = n
				continue
			}
			piData := rest[2 : 2+end]
			fn, err := ii.getAttr(self, "handle_pi")
			if err == nil {
				if _, err2 := ii.callObject(fn, []object.Object{&object.Str{V: piData}}, nil); err2 != nil {
					return err2
				}
			}
			pos += 2 + end + 2
			continue
		}

		// Declaration: <!...>  (DOCTYPE, CDATA, etc.)
		if strings.HasPrefix(rest, "<!") && len(rest) > 2 && rest[2] != '-' {
			end := strings.IndexByte(rest[2:], '>')
			if end < 0 {
				if err := fireData(rest); err != nil {
					return err
				}
				pos = n
				continue
			}
			declContent := rest[2 : 2+end]
			var fn object.Object
			var getErr error
			if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(declContent)), "DOCTYPE") ||
				strings.HasPrefix(strings.ToUpper(strings.TrimSpace(declContent)), "[") ||
				isStandardDecl(declContent) {
				fn, getErr = ii.getAttr(self, "handle_decl")
			} else {
				fn, getErr = ii.getAttr(self, "unknown_decl")
			}
			if getErr == nil {
				if _, err2 := ii.callObject(fn, []object.Object{&object.Str{V: declContent}}, nil); err2 != nil {
					return err2
				}
			}
			pos += 2 + end + 1
			continue
		}

		// End tag: </tag>
		if strings.HasPrefix(rest, "</") {
			end := strings.IndexByte(rest[2:], '>')
			if end < 0 {
				if err := fireData(rest); err != nil {
					return err
				}
				pos = n
				continue
			}
			tagName := strings.ToLower(strings.TrimSpace(rest[2 : 2+end]))
			fn, err := ii.getAttr(self, "handle_endtag")
			if err == nil {
				if _, err2 := ii.callObject(fn, []object.Object{&object.Str{V: tagName}}, nil); err2 != nil {
					return err2
				}
			}
			pos += 2 + end + 1
			continue
		}

		// Start tag or self-closing: <tag ...> or <tag .../>
		if len(rest) > 1 && (rest[1] == '_' || rest[1] == ':' || isLetter(rest[1])) {
			tagEnd, selfClose, rawTag, tagName, attrs := parseStartTag(rest)
			if tagEnd < 0 {
				// malformed — output as text
				if err := fireData("<"); err != nil {
					return err
				}
				pos++
				continue
			}
			if st != nil {
				st.lastStartTag = rawTag
			}
			// Build attrs list
			attrsList := make([]object.Object, len(attrs))
			for idx, av := range attrs {
				attrVal := av[1]
				if convertCharrefs {
					attrVal = htmlUnescape(attrVal)
				}
				attrsList[idx] = &object.Tuple{V: []object.Object{
					&object.Str{V: av[0]},
					&object.Str{V: attrVal},
				}}
			}
			attrsObj := &object.List{V: attrsList}

			if selfClose {
				fn, err := ii.getAttr(self, "handle_startendtag")
				if err == nil {
					if _, err2 := ii.callObject(fn, []object.Object{
						&object.Str{V: tagName},
						attrsObj,
					}, nil); err2 != nil {
						return err2
					}
				}
			} else {
				fn, err := ii.getAttr(self, "handle_starttag")
				if err == nil {
					if _, err2 := ii.callObject(fn, []object.Object{
						&object.Str{V: tagName},
						attrsObj,
					}, nil); err2 != nil {
						return err2
					}
				}
			}
			pos += tagEnd
			continue
		}

		// Not a recognized tag — treat '<' as literal text
		if err := fireData("<"); err != nil {
			return err
		}
		pos++
	}
	return nil
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
// Returns (consumed, selfClose, rawTag, tagName, attrs) where consumed is -1 on failure.
func parseStartTag(s string) (consumed int, selfClose bool, rawTag, tagName string, attrs [][2]string) {
	// s starts with '<'
	if len(s) < 2 {
		return -1, false, "", "", nil
	}
	// find the matching '>'
	// We need to handle quoted attribute values
	i := 1
	n := len(s)

	// tag name
	for i < n && !isSpace(s[i]) && s[i] != '>' && s[i] != '/' {
		i++
	}
	if i >= n {
		return -1, false, "", "", nil
	}
	tagName = strings.ToLower(s[1:i])
	if tagName == "" {
		return -1, false, "", "", nil
	}

	// parse attributes
	for i < n {
		// skip whitespace
		for i < n && isSpace(s[i]) {
			i++
		}
		if i >= n {
			return -1, false, "", "", nil
		}
		if s[i] == '>' {
			i++
			end := i
			return end, false, s[:end], tagName, attrs
		}
		if s[i] == '/' {
			i++
			if i < n && s[i] == '>' {
				i++
				return i, true, s[:i], tagName, attrs
			}
			// lone '/' — skip
			continue
		}
		// attribute name
		attrStart := i
		for i < n && s[i] != '=' && s[i] != '>' && s[i] != '/' && !isSpace(s[i]) {
			i++
		}
		attrName := strings.ToLower(s[attrStart:i])
		if attrName == "" {
			i++
			continue
		}
		// skip whitespace
		for i < n && isSpace(s[i]) {
			i++
		}
		if i >= n {
			attrs = append(attrs, [2]string{attrName, ""})
			return -1, false, "", "", nil
		}
		if s[i] != '=' {
			// boolean attribute
			attrs = append(attrs, [2]string{attrName, ""})
			continue
		}
		i++ // skip '='
		// skip whitespace
		for i < n && isSpace(s[i]) {
			i++
		}
		if i >= n {
			attrs = append(attrs, [2]string{attrName, ""})
			return -1, false, "", "", nil
		}
		var attrVal string
		if s[i] == '"' {
			i++
			end := strings.IndexByte(s[i:], '"')
			if end < 0 {
				// unterminated
				attrVal = s[i:]
				i = n
			} else {
				attrVal = s[i : i+end]
				i = i + end + 1
			}
		} else if s[i] == '\'' {
			i++
			end := strings.IndexByte(s[i:], '\'')
			if end < 0 {
				attrVal = s[i:]
				i = n
			} else {
				attrVal = s[i : i+end]
				i = i + end + 1
			}
		} else {
			// unquoted
			start := i
			for i < n && !isSpace(s[i]) && s[i] != '>' && s[i] != '/' {
				i++
			}
			attrVal = s[start:i]
		}
		attrs = append(attrs, [2]string{attrName, attrVal})
	}
	return -1, false, "", "", nil
}

// isSpace returns true if b is ASCII whitespace.
func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

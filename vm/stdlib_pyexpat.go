package vm

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/tamnd/goipy/object"
)

// ── pyexpat ───────────────────────────────────────────────────────────────────

// pyexpatErrorMessages maps expat error code → message string.
var pyexpatErrorMessages = map[int]string{
	1:  "out of memory",
	2:  "syntax error",
	3:  "no element found",
	4:  "not well-formed (invalid token)",
	5:  "unclosed token",
	6:  "partial character",
	7:  "mismatched tag",
	8:  "duplicate attribute",
	9:  "junk after document element",
	10: "illegal parameter entity reference",
	11: "undefined entity",
	12: "recursive entity reference",
	13: "asynchronous entity",
	14: "reference to invalid character number",
	15: "reference to binary entity",
	16: "reference to external entity in attribute",
	17: "XML or text declaration not at start of entity",
	18: "unknown encoding",
	19: "encoding specified in XML declaration is incorrect",
	20: "unclosed CDATA section",
	21: "error in processing external entity reference",
	22: "document is not standalone",
	23: "unexpected parser state - please send a bug report",
	24: "entity declared in parameter entity",
	25: "requested feature requires XML_DTD support in Expat",
	26: "cannot change setting once parsing has begun",
	27: "unbound prefix",
	28: "must not undeclare prefix",
	29: "incomplete markup in parameter entity",
	30: "XML declaration not well-formed",
	31: "text declaration not well-formed",
	32: "illegal character(s) in public id",
	33: "parser suspended",
	34: "parser not suspended",
	35: "parsing aborted",
	36: "parsing finished",
	37: "cannot suspend in external parameter entity",
	38: "reserved prefix (xml) must not be undeclared or bound to another namespace name",
	39: "reserved prefix (xmlns) must not be declared or undeclared",
	40: "prefix must not be bound to one of the reserved namespace names",
	41: "invalid argument",
	42: "a successful prior call to function XML_GetBuffer is required",
	43: "limit on input amplification factor (from DTD and entities) breached",
	44: "parser not started",
}

// pyexpatErrorCodes maps message string → error code (reverse of above).
var pyexpatErrorCodes = func() map[string]int {
	m := make(map[string]int, len(pyexpatErrorMessages))
	for code, msg := range pyexpatErrorMessages {
		m[msg] = code
	}
	return m
}()

// pyexpatErrorByName maps XML_ERROR_* constant name → message string.
var pyexpatErrorByName = map[string]string{
	"XML_ERROR_NO_MEMORY":                   "out of memory",
	"XML_ERROR_SYNTAX":                      "syntax error",
	"XML_ERROR_NO_ELEMENTS":                 "no element found",
	"XML_ERROR_INVALID_TOKEN":               "not well-formed (invalid token)",
	"XML_ERROR_UNCLOSED_TOKEN":              "unclosed token",
	"XML_ERROR_PARTIAL_CHAR":                "partial character",
	"XML_ERROR_TAG_MISMATCH":                "mismatched tag",
	"XML_ERROR_DUPLICATE_ATTRIBUTE":         "duplicate attribute",
	"XML_ERROR_JUNK_AFTER_DOC_ELEMENT":      "junk after document element",
	"XML_ERROR_PARAM_ENTITY_REF":            "illegal parameter entity reference",
	"XML_ERROR_UNDEFINED_ENTITY":            "undefined entity",
	"XML_ERROR_RECURSIVE_ENTITY_REF":        "recursive entity reference",
	"XML_ERROR_ASYNC_ENTITY":                "asynchronous entity",
	"XML_ERROR_BAD_CHAR_REF":                "reference to invalid character number",
	"XML_ERROR_BINARY_ENTITY_REF":           "reference to binary entity",
	"XML_ERROR_ATTRIBUTE_EXTERNAL_ENTITY_REF": "reference to external entity in attribute",
	"XML_ERROR_MISPLACED_XML_PI":            "XML or text declaration not at start of entity",
	"XML_ERROR_UNKNOWN_ENCODING":            "unknown encoding",
	"XML_ERROR_INCORRECT_ENCODING":          "encoding specified in XML declaration is incorrect",
	"XML_ERROR_UNCLOSED_CDATA_SECTION":      "unclosed CDATA section",
	"XML_ERROR_EXTERNAL_ENTITY_HANDLING":    "error in processing external entity reference",
	"XML_ERROR_NOT_STANDALONE":              "document is not standalone",
	"XML_ERROR_UNEXPECTED_STATE":            "unexpected parser state - please send a bug report",
	"XML_ERROR_ENTITY_DECLARED_IN_PE":       "entity declared in parameter entity",
	"XML_ERROR_FEATURE_REQUIRES_XML_DTD":    "requested feature requires XML_DTD support in Expat",
	"XML_ERROR_CANT_CHANGE_FEATURE_ONCE_PARSING": "cannot change setting once parsing has begun",
	"XML_ERROR_UNBOUND_PREFIX":              "unbound prefix",
	"XML_ERROR_UNDECLARING_PREFIX":          "must not undeclare prefix",
	"XML_ERROR_INCOMPLETE_PE":               "incomplete markup in parameter entity",
	"XML_ERROR_XML_DECL":                    "XML declaration not well-formed",
	"XML_ERROR_TEXT_DECL":                   "text declaration not well-formed",
	"XML_ERROR_PUBLICID":                    "illegal character(s) in public id",
	"XML_ERROR_SUSPENDED":                   "parser suspended",
	"XML_ERROR_NOT_SUSPENDED":               "parser not suspended",
	"XML_ERROR_ABORTED":                     "parsing aborted",
	"XML_ERROR_FINISHED":                    "parsing finished",
	"XML_ERROR_SUSPEND_PE":                  "cannot suspend in external parameter entity",
	"XML_ERROR_RESERVED_PREFIX_XML":         "reserved prefix (xml) must not be undeclared or bound to another namespace name",
	"XML_ERROR_RESERVED_PREFIX_XMLNS":       "reserved prefix (xmlns) must not be declared or undeclared",
	"XML_ERROR_RESERVED_NAMESPACE_URI":      "prefix must not be bound to one of the reserved namespace names",
	"XML_ERROR_INVALID_ARGUMENT":            "invalid argument",
	"XML_ERROR_NO_BUFFER":                   "a successful prior call to function XML_GetBuffer is required",
	"XML_ERROR_AMPLIFICATION_LIMIT_BREACH":  "limit on input amplification factor (from DTD and entities) breached",
	"XML_ERROR_NOT_STARTED":                 "parser not started",
}

// pyexpatState holds per-parser-instance state.
type pyexpatState struct {
	handlers         map[string]object.Object
	nsSep            string // namespace separator, "" = no namespace mode
	buf              []byte // buffered data for incremental parsing
	orderedAttrs     bool
}

type pyexpatRegistry struct{ m sync.Map }

func (r *pyexpatRegistry) Store(k *object.Instance, v *pyexpatState) { r.m.Store(k, v) }
func (r *pyexpatRegistry) Load(k *object.Instance) *pyexpatState {
	v, ok := r.m.Load(k)
	if !ok {
		return nil
	}
	return v.(*pyexpatState)
}

var pyexpatMap pyexpatRegistry

// pyexpatHandlerNames is the full list of handler names.
var pyexpatHandlerNames = []string{
	"StartElementHandler", "EndElementHandler", "CharacterDataHandler",
	"ProcessingInstructionHandler", "CommentHandler",
	"StartCdataSectionHandler", "EndCdataSectionHandler",
	"DefaultHandler", "DefaultHandlerExpand",
	"XmlDeclHandler",
	"StartDoctypeDeclHandler", "EndDoctypeDeclHandler",
	"NotationDeclHandler", "UnparsedEntityDeclHandler",
	"EntityDeclHandler", "ElementDeclHandler", "AttlistDeclHandler",
	"StartNamespaceDeclHandler", "EndNamespaceDeclHandler",
	"NotStandaloneHandler", "ExternalEntityRefHandler",
	"SkippedEntityHandler",
}

func (i *Interp) buildPyexpat() *object.Module {
	m := &object.Module{Name: "pyexpat", Dict: object.NewDict()}

	// ExpatError / error
	expatErrCls := &object.Class{Name: "ExpatError", Dict: object.NewDict(), Bases: []*object.Class{i.exception}}
	m.Dict.SetStr("ExpatError", expatErrCls)
	m.Dict.SetStr("error", expatErrCls)

	// Module constants
	m.Dict.SetStr("EXPAT_VERSION", &object.Str{V: "expat_2.5.0"})
	m.Dict.SetStr("native_encoding", &object.Str{V: "UTF-8"})
	m.Dict.SetStr("version_info", &object.Tuple{V: []object.Object{
		object.IntFromInt64(2), object.IntFromInt64(5), object.IntFromInt64(0),
	}})
	m.Dict.SetStr("XML_PARAM_ENTITY_PARSING_NEVER", object.IntFromInt64(0))
	m.Dict.SetStr("XML_PARAM_ENTITY_PARSING_UNLESS_STANDALONE", object.IntFromInt64(1))
	m.Dict.SetStr("XML_PARAM_ENTITY_PARSING_ALWAYS", object.IntFromInt64(2))

	// features list
	featureItems := []object.Object{
		&object.Tuple{V: []object.Object{&object.Str{V: "sizeof(XML_Char)"}, object.IntFromInt64(1)}},
		&object.Tuple{V: []object.Object{&object.Str{V: "sizeof(XML_LChar)"}, object.IntFromInt64(1)}},
		&object.Tuple{V: []object.Object{&object.Str{V: "XML_DTD"}, object.IntFromInt64(0)}},
		&object.Tuple{V: []object.Object{&object.Str{V: "XML_CONTEXT_BYTES"}, object.IntFromInt64(1024)}},
		&object.Tuple{V: []object.Object{&object.Str{V: "XML_NS"}, object.IntFromInt64(0)}},
	}
	m.Dict.SetStr("features", &object.List{V: featureItems})

	// ErrorString(errno) -> str or None
	m.Dict.SetStr("ErrorString", &object.BuiltinFunc{Name: "ErrorString", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		code := int64(0)
		if n, ok := a[0].(*object.Int); ok {
			code = n.Int64()
		}
		if code == 0 {
			return object.None, nil
		}
		if msg, ok := pyexpatErrorMessages[int(code)]; ok {
			return &object.Str{V: msg}, nil
		}
		return object.None, nil
	}})

	// errors sub-module
	errMod := &object.Module{Name: "pyexpat.errors", Dict: object.NewDict()}
	codesDict := object.NewDict()
	messagesDict := object.NewDict()
	for constName, msg := range pyexpatErrorByName {
		errMod.Dict.SetStr(constName, &object.Str{V: msg})
		code := pyexpatErrorCodes[msg]
		codesDict.Set(&object.Str{V: msg}, object.IntFromInt64(int64(code)))
		messagesDict.Set(object.IntFromInt64(int64(code)), &object.Str{V: msg})
	}
	errMod.Dict.SetStr("codes", codesDict)
	errMod.Dict.SetStr("messages", messagesDict)
	m.Dict.SetStr("errors", errMod)

	// model sub-module
	modelMod := &object.Module{Name: "pyexpat.model", Dict: object.NewDict()}
	modelMod.Dict.SetStr("XML_CTYPE_EMPTY", object.IntFromInt64(1))
	modelMod.Dict.SetStr("XML_CTYPE_ANY", object.IntFromInt64(2))
	modelMod.Dict.SetStr("XML_CTYPE_MIXED", object.IntFromInt64(3))
	modelMod.Dict.SetStr("XML_CTYPE_NAME", object.IntFromInt64(4))
	modelMod.Dict.SetStr("XML_CTYPE_CHOICE", object.IntFromInt64(5))
	modelMod.Dict.SetStr("XML_CTYPE_SEQ", object.IntFromInt64(6))
	modelMod.Dict.SetStr("XML_CQUANT_NONE", object.IntFromInt64(0))
	modelMod.Dict.SetStr("XML_CQUANT_OPT", object.IntFromInt64(1))
	modelMod.Dict.SetStr("XML_CQUANT_REP", object.IntFromInt64(2))
	modelMod.Dict.SetStr("XML_CQUANT_PLUS", object.IntFromInt64(3))
	m.Dict.SetStr("model", modelMod)

	// xmlparser class
	xmlparserCls := &object.Class{Name: "xmlparser", Dict: object.NewDict()}
	m.Dict.SetStr("XMLParserType", xmlparserCls)

	pyexpatNewInst := func(nsSep string) *object.Instance {
		inst := &object.Instance{Class: xmlparserCls, Dict: object.NewDict()}
		st := &pyexpatState{
			handlers: make(map[string]object.Object, len(pyexpatHandlerNames)),
			nsSep:    nsSep,
		}
		for _, h := range pyexpatHandlerNames {
			st.handlers[h] = object.None
		}
		pyexpatMap.Store(inst, st)
		inst.Dict.SetStr("buffer_size", object.IntFromInt64(8192))
		inst.Dict.SetStr("buffer_text", object.False)
		inst.Dict.SetStr("buffer_used", object.IntFromInt64(0))
		inst.Dict.SetStr("ordered_attributes", object.False)
		inst.Dict.SetStr("specified_attributes", object.False)
		inst.Dict.SetStr("namespace_prefixes", object.False)
		inst.Dict.SetStr("intern", object.NewDict())
		inst.Dict.SetStr("ErrorCode", object.IntFromInt64(0))
		inst.Dict.SetStr("ErrorLineNumber", object.IntFromInt64(1))
		inst.Dict.SetStr("ErrorColumnNumber", object.IntFromInt64(0))
		inst.Dict.SetStr("ErrorByteIndex", object.IntFromInt64(-1))
		inst.Dict.SetStr("CurrentLineNumber", object.IntFromInt64(1))
		inst.Dict.SetStr("CurrentColumnNumber", object.IntFromInt64(0))
		inst.Dict.SetStr("CurrentByteIndex", object.IntFromInt64(-1))
		return inst
	}

	// __setattr__ — capture handler/attr assignments
	xmlparserCls.Dict.SetStr("__setattr__", &object.BuiltinFunc{Name: "__setattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		name := ""
		if s, ok2 := a[1].(*object.Str); ok2 {
			name = s.V
		}
		val := a[2]
		self.Dict.SetStr(name, val)
		if st := pyexpatMap.Load(self); st != nil {
			if _, isHandler := st.handlers[name]; isHandler {
				st.handlers[name] = val
			}
			if name == "ordered_attributes" {
				st.orderedAttrs = val == object.True
			}
		}
		return object.None, nil
	}})

	// __getattr__
	xmlparserCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		if v, ok := self.Dict.GetStr(name); ok {
			return v, nil
		}
		if st := pyexpatMap.Load(self); st != nil {
			if v, ok := st.handlers[name]; ok {
				return v, nil
			}
		}
		return object.None, nil
	}})

	// Parse(data, isfinal=False)
	xmlparserCls.Dict.SetStr("Parse", &object.BuiltinFunc{Name: "Parse", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.IntFromInt64(1), nil
		}
		self := a[0].(*object.Instance)
		data, _ := asBytes(a[1])
		isFinal := false
		if len(a) >= 3 {
			isFinal = a[2] != object.False && a[2] != object.None
			if n, ok := a[2].(*object.Int); ok {
				isFinal = n.Int64() != 0
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("isfinal"); ok {
				isFinal = v != object.False && v != object.None
				if n, ok2 := v.(*object.Int); ok2 {
					isFinal = n.Int64() != 0
				}
			}
		}
		st := pyexpatMap.Load(self)
		if st == nil {
			return object.IntFromInt64(1), nil
		}
		st.buf = append(st.buf, data...)
		if !isFinal {
			return object.IntFromInt64(1), nil
		}
		ii := interp.(*Interp)
		parseData := st.buf
		st.buf = nil
		return object.IntFromInt64(1), i.pyexpatDoParse(ii, self, st, parseData, expatErrCls)
	}})

	// ParseFile(file)
	xmlparserCls.Dict.SetStr("ParseFile", &object.BuiltinFunc{Name: "ParseFile", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.IntFromInt64(1), nil
		}
		self := a[0].(*object.Instance)
		ii := interp.(*Interp)
		fn, err := ii.getAttr(a[1], "read")
		if err != nil {
			return object.IntFromInt64(1), nil
		}
		res, rerr := ii.callObject(fn, nil, nil)
		if rerr != nil {
			return nil, rerr
		}
		data, _ := asBytes(res)
		st := pyexpatMap.Load(self)
		if st == nil {
			return object.IntFromInt64(1), nil
		}
		return object.IntFromInt64(1), i.pyexpatDoParse(ii, self, st, data, expatErrCls)
	}})

	// SetBase / GetBase
	xmlparserCls.Dict.SetStr("SetBase", &object.BuiltinFunc{Name: "SetBase", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			if self, ok := a[0].(*object.Instance); ok {
				self.Dict.SetStr("_base", a[1])
			}
		}
		return object.None, nil
	}})
	xmlparserCls.Dict.SetStr("GetBase", &object.BuiltinFunc{Name: "GetBase", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		if self, ok := a[0].(*object.Instance); ok {
			if v, ok2 := self.Dict.GetStr("_base"); ok2 {
				return v, nil
			}
		}
		return object.None, nil
	}})

	// GetInputContext → b""
	xmlparserCls.Dict.SetStr("GetInputContext", &object.BuiltinFunc{Name: "GetInputContext", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Bytes{V: []byte{}}, nil
	}})

	// ExternalEntityParserCreate(context, encoding=None) → xmlparser
	xmlparserCls.Dict.SetStr("ExternalEntityParserCreate", &object.BuiltinFunc{Name: "ExternalEntityParserCreate", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		nsSep := ""
		if len(a) >= 1 {
			if parent, ok := a[0].(*object.Instance); ok {
				if st := pyexpatMap.Load(parent); st != nil {
					nsSep = st.nsSep
				}
			}
		}
		sub := pyexpatNewInst(nsSep)
		// Copy handlers from parent
		if len(a) >= 1 {
			if parent, ok := a[0].(*object.Instance); ok {
				if pst := pyexpatMap.Load(parent); pst != nil {
					if sst := pyexpatMap.Load(sub); sst != nil {
						for k, v := range pst.handlers {
							sst.handlers[k] = v
						}
					}
				}
			}
		}
		return sub, nil
	}})

	// SetParamEntityParsing(flag) → 1
	xmlparserCls.Dict.SetStr("SetParamEntityParsing", &object.BuiltinFunc{Name: "SetParamEntityParsing", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.IntFromInt64(1), nil
	}})

	// UseForeignDTD([flag]) → None
	xmlparserCls.Dict.SetStr("UseForeignDTD", &object.BuiltinFunc{Name: "UseForeignDTD", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// SetReparseDeferralEnabled / GetReparseDeferralEnabled (stubs, Python 3.12+)
	xmlparserCls.Dict.SetStr("SetReparseDeferralEnabled", &object.BuiltinFunc{Name: "SetReparseDeferralEnabled", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	xmlparserCls.Dict.SetStr("GetReparseDeferralEnabled", &object.BuiltinFunc{Name: "GetReparseDeferralEnabled", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.True, nil
	}})

	// ParserCreate(encoding=None, namespace_separator=None) → xmlparser
	m.Dict.SetStr("ParserCreate", &object.BuiltinFunc{Name: "ParserCreate", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		nsSep := ""
		// a[0]=encoding, a[1]=namespace_separator
		if len(a) >= 2 && a[1] != object.None {
			if s, ok := a[1].(*object.Str); ok {
				nsSep = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("namespace_separator"); ok && v != object.None {
				if s, ok2 := v.(*object.Str); ok2 {
					nsSep = s.V
				}
			}
		}
		return pyexpatNewInst(nsSep), nil
	}})

	return m
}

// pyexpatDoParse is the core parse engine.
func (i *Interp) pyexpatDoParse(ii *Interp, self *object.Instance, st *pyexpatState, data []byte, expatErrCls *object.Class) error {
	callH := func(name string, args []object.Object) error {
		h := st.handlers[name]
		if h == nil || h == object.None {
			// Also check instance dict (handler set directly)
			if v, ok := self.Dict.GetStr(name); ok && v != object.None {
				h = v
			}
		}
		if h == nil || h == object.None {
			return nil
		}
		_, err := ii.callObject(h, args, nil)
		return err
	}

	// Build newline index for line/column computation.
	newlines := pyexpatNewlineIndex(data)

	lineCol := func(offset int64) (line, col int64) {
		// binary search newlines
		lo, hi := 0, len(newlines)
		for lo < hi {
			mid := (lo + hi) / 2
			if newlines[mid] < offset {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		line = int64(lo + 1) // 1-based
		lineStart := int64(0)
		if lo > 0 {
			lineStart = newlines[lo-1] + 1
		}
		col = offset - lineStart
		return
	}

	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false

	prevOffset := int64(0)

	// Track pending namespace declarations per element depth.
	type nsDecl struct{ prefix, uri string }
	nsStack := [][]nsDecl{} // per-element pending decls

	for {
		tokStart := prevOffset
		tok, err := dec.Token()
		prevOffset = dec.InputOffset()

		if err == io.EOF {
			break
		}
		if err != nil {
			// Parse error — compute position and raise ExpatError.
			errLine, errCol := lineCol(tokStart)
			errCode := pyexpatErrorCodeFromGoErr(err)
			errMsg := pyexpatErrorMessages[errCode]
			if errMsg == "" {
				errMsg = "syntax error"
			}
			msg := fmt.Sprintf("%s: line %d, column %d", errMsg, errLine, errCol)
			exc := &object.Exception{
				Class: expatErrCls,
				Args:  &object.Tuple{V: []object.Object{&object.Str{V: msg}}},
				Dict:  object.NewDict(),
			}
			exc.Dict.SetStr("lineno", object.IntFromInt64(errLine))
			exc.Dict.SetStr("offset", object.IntFromInt64(errCol))
			exc.Dict.SetStr("code", object.IntFromInt64(int64(errCode)))
			// Update parser error attrs
			self.Dict.SetStr("ErrorCode", object.IntFromInt64(int64(errCode)))
			self.Dict.SetStr("ErrorLineNumber", object.IntFromInt64(errLine))
			self.Dict.SetStr("ErrorColumnNumber", object.IntFromInt64(errCol))
			self.Dict.SetStr("ErrorByteIndex", object.IntFromInt64(tokStart))
			return exc
		}

		// Update current position attributes.
		curLine, curCol := lineCol(tokStart)
		self.Dict.SetStr("CurrentLineNumber", object.IntFromInt64(curLine))
		self.Dict.SetStr("CurrentColumnNumber", object.IntFromInt64(curCol))
		self.Dict.SetStr("CurrentByteIndex", object.IntFromInt64(tokStart))

		switch t := tok.(type) {
		case xml.ProcInst:
			if t.Target == "xml" {
				// XML declaration
				ver, enc, sa := pyexpatParseXmlDecl(t.Inst)
				if err2 := callH("XmlDeclHandler", []object.Object{
					&object.Str{V: ver},
					strOrNone(enc),
					object.IntFromInt64(int64(sa)),
				}); err2 != nil {
					return err2
				}
			} else {
				if err2 := callH("ProcessingInstructionHandler", []object.Object{
					&object.Str{V: t.Target},
					&object.Str{V: string(t.Inst)},
				}); err2 != nil {
					return err2
				}
			}

		case xml.Directive:
			// DOCTYPE or other
			s := strings.TrimSpace(string(t))
			if strings.HasPrefix(s, "DOCTYPE") {
				name, sysid, pubid, hasInternal := pyexpatParseDoctype(s)
				hasInternalInt := int64(0)
				if hasInternal {
					hasInternalInt = 1
				}
				// Fire StartDoctypeDeclHandler
				if err2 := callH("StartDoctypeDeclHandler", []object.Object{
					&object.Str{V: name},
					strOrNone(sysid),
					strOrNone(pubid),
					object.IntFromInt64(hasInternalInt),
				}); err2 != nil {
					return err2
				}
				// Parse notation declarations inside the internal subset
				if hasInternal {
					if err2 := pyexpatParseInternalSubset(ii, callH, s); err2 != nil {
						return err2
					}
				}
				if err2 := callH("EndDoctypeDeclHandler", nil); err2 != nil {
					return err2
				}
			}

		case xml.StartElement:
			// Collect namespace declarations first
			var elemNS []nsDecl
			if st.nsSep != "" {
				for _, attr := range t.Attr {
					if attr.Name.Space == "xmlns" {
						// xmlns:prefix="uri"
						elemNS = append(elemNS, nsDecl{attr.Name.Local, attr.Value})
					} else if attr.Name.Space == "" && attr.Name.Local == "xmlns" {
						// default namespace
						elemNS = append(elemNS, nsDecl{"", attr.Value})
					}
				}
				for _, ns := range elemNS {
					if err2 := callH("StartNamespaceDeclHandler", []object.Object{
						strOrNone(ns.prefix),
						&object.Str{V: ns.uri},
					}); err2 != nil {
						return err2
					}
				}
			}
			nsStack = append(nsStack, elemNS)

			// Build element name
			elemName := pyexpatElemName(t.Name, st.nsSep)

			// Build attrs
			var attrsArg object.Object
			if st.orderedAttrs {
				var flat []object.Object
				for _, attr := range t.Attr {
					if st.nsSep != "" && (attr.Name.Space == "xmlns" ||
						(attr.Name.Space == "" && attr.Name.Local == "xmlns")) {
						continue // skip xmlns decls from attrs in NS mode
					}
					aname := pyexpatElemName(attr.Name, st.nsSep)
					flat = append(flat, &object.Str{V: aname}, &object.Str{V: attr.Value})
				}
				attrsArg = &object.List{V: flat}
			} else {
				d := object.NewDict()
				for _, attr := range t.Attr {
					if st.nsSep != "" && (attr.Name.Space == "xmlns" ||
						(attr.Name.Space == "" && attr.Name.Local == "xmlns")) {
						continue
					}
					aname := pyexpatElemName(attr.Name, st.nsSep)
					d.SetStr(aname, &object.Str{V: attr.Value})
				}
				attrsArg = d
			}

			if err2 := callH("StartElementHandler", []object.Object{
				&object.Str{V: elemName}, attrsArg,
			}); err2 != nil {
				return err2
			}

		case xml.EndElement:
			elemName := pyexpatElemName(t.Name, st.nsSep)
			if err2 := callH("EndElementHandler", []object.Object{
				&object.Str{V: elemName},
			}); err2 != nil {
				return err2
			}
			// Fire EndNamespaceDeclHandler for this element's decls (reverse order)
			if st.nsSep != "" && len(nsStack) > 0 {
				elemNS := nsStack[len(nsStack)-1]
				nsStack = nsStack[:len(nsStack)-1]
				for idx := len(elemNS) - 1; idx >= 0; idx-- {
					if err2 := callH("EndNamespaceDeclHandler", []object.Object{
						strOrNone(elemNS[idx].prefix),
					}); err2 != nil {
						return err2
					}
				}
			} else if len(nsStack) > 0 {
				nsStack = nsStack[:len(nsStack)-1]
			}

		case xml.CharData:
			// Detect CDATA section by checking raw bytes at tokStart
			isCDATA := tokStart+8 < int64(len(data)) &&
				string(data[tokStart:tokStart+9]) == "<![CDATA["
			text := string(t)
			if isCDATA {
				if err2 := callH("StartCdataSectionHandler", nil); err2 != nil {
					return err2
				}
				if err2 := callH("CharacterDataHandler", []object.Object{
					&object.Str{V: text},
				}); err2 != nil {
					return err2
				}
				if err2 := callH("EndCdataSectionHandler", nil); err2 != nil {
					return err2
				}
			} else {
				if err2 := callH("CharacterDataHandler", []object.Object{
					&object.Str{V: text},
				}); err2 != nil {
					return err2
				}
			}

		case xml.Comment:
			if err2 := callH("CommentHandler", []object.Object{
				&object.Str{V: string(t)},
			}); err2 != nil {
				return err2
			}
		}
	}

	return nil
}

// pyexpatNewlineIndex builds a sorted list of newline byte offsets.
func pyexpatNewlineIndex(data []byte) []int64 {
	var nl []int64
	for idx, b := range data {
		if b == '\n' {
			nl = append(nl, int64(idx))
		}
	}
	return nl
}

// pyexpatParseXmlDecl parses the content of <?xml ...?> → version, encoding, standalone.
// standalone: 1=yes, 0=no, -1=not specified.
func pyexpatParseXmlDecl(inst []byte) (version, encoding string, standalone int) {
	s := string(inst)
	standalone = -1
	getAttr := func(key string) string {
		for _, q := range []byte{'"', '\''} {
			prefix := key + "=" + string(q)
			if idx := strings.Index(s, prefix); idx >= 0 {
				rest := s[idx+len(prefix):]
				if end := strings.IndexByte(rest, q); end >= 0 {
					return rest[:end]
				}
			}
		}
		return ""
	}
	version = getAttr("version")
	encoding = getAttr("encoding")
	sa := getAttr("standalone")
	switch sa {
	case "yes":
		standalone = 1
	case "no":
		standalone = 0
	}
	return
}

// pyexpatParseDoctype parses a DOCTYPE directive string.
func pyexpatParseDoctype(s string) (name, sysid, pubid string, hasInternal bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "DOCTYPE") {
		return
	}
	s = strings.TrimSpace(s[7:])
	// get element name (first token)
	idx := strings.IndexAny(s, " \t\r\n[")
	if idx < 0 {
		name = s
		return
	}
	name = s[:idx]
	rest := strings.TrimSpace(s[idx:])
	if strings.Contains(rest, "[") {
		hasInternal = true
	}
	if strings.HasPrefix(rest, "SYSTEM") {
		rest = strings.TrimSpace(rest[6:])
		sysid = pyexpatReadQuoted(rest)
	} else if strings.HasPrefix(rest, "PUBLIC") {
		rest = strings.TrimSpace(rest[6:])
		pubid = pyexpatReadQuoted(rest)
		if end := strings.IndexAny(rest, "\"'"); end >= 0 {
			q := rest[end]
			rest2 := rest[end+1:]
			if e2 := strings.IndexByte(rest2, q); e2 >= 0 {
				rest = strings.TrimSpace(rest2[e2+1:])
				sysid = pyexpatReadQuoted(rest)
			}
		}
	}
	return
}

func pyexpatReadQuoted(s string) string {
	if len(s) == 0 {
		return ""
	}
	for _, q := range []byte{'"', '\''} {
		if idx := strings.IndexByte(s, q); idx >= 0 {
			rest := s[idx+1:]
			if end := strings.IndexByte(rest, q); end >= 0 {
				return rest[:end]
			}
		}
	}
	return ""
}

// pyexpatParseInternalSubset fires NotationDeclHandler for NOTATION entries.
func pyexpatParseInternalSubset(ii *Interp, callH func(string, []object.Object) error, directive string) error {
	// Find the internal subset between [ and ]
	start := strings.Index(directive, "[")
	end := strings.LastIndex(directive, "]")
	if start < 0 || end <= start {
		return nil
	}
	subset := directive[start+1 : end]
	// Scan for <!NOTATION ...>
	for {
		idx := strings.Index(subset, "<!NOTATION")
		if idx < 0 {
			break
		}
		rest := strings.TrimSpace(subset[idx+10:])
		// name
		parts := strings.Fields(rest)
		if len(parts) == 0 {
			break
		}
		notName := parts[0]
		rest2 := strings.TrimSpace(rest[len(notName):])
		var sysid, pubid string
		if strings.HasPrefix(rest2, "SYSTEM") {
			rest2 = strings.TrimSpace(rest2[6:])
			sysid = pyexpatReadQuoted(rest2)
		} else if strings.HasPrefix(rest2, "PUBLIC") {
			rest2 = strings.TrimSpace(rest2[6:])
			pubid = pyexpatReadQuoted(rest2)
		}
		if err2 := callH("NotationDeclHandler", []object.Object{
			&object.Str{V: notName},
			object.None, // base
			strOrNone(sysid),
			strOrNone(pubid),
		}); err2 != nil {
			return err2
		}
		subset = subset[idx+10:]
	}
	return nil
}

// pyexpatElemName builds the element/attr name for pyexpat output.
// In namespace mode, it's "{uri}{sep}{local}"; otherwise "prefix:local" or "local".
func pyexpatElemName(n xml.Name, nsSep string) string {
	if nsSep != "" {
		if n.Space != "" {
			return n.Space + nsSep + n.Local
		}
		return n.Local
	}
	return n.Local
}

// pyexpatErrorCodeFromGoErr maps a Go XML parse error to an expat error code.
func pyexpatErrorCodeFromGoErr(err error) int {
	if err == nil {
		return 0
	}
	s := err.Error()
	if strings.Contains(s, "unexpected EOF") || strings.Contains(s, "unclosed") {
		return 5 // XML_ERROR_UNCLOSED_TOKEN
	}
	if strings.Contains(s, "mismatched") {
		return 7 // XML_ERROR_TAG_MISMATCH
	}
	if strings.Contains(s, "syntax") {
		return 2 // XML_ERROR_SYNTAX
	}
	return 4 // XML_ERROR_INVALID_TOKEN
}

// strOrNone returns a *object.Str if s != "", else object.None.
func strOrNone(s string) object.Object {
	if s == "" {
		return object.None
	}
	return &object.Str{V: s}
}

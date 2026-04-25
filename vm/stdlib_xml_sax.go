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

// ── xml.sax ──────────────────────────────────────────────────────────────────

func (i *Interp) buildXmlSax() *object.Module {
	m := &object.Module{Name: "xml.sax", Dict: object.NewDict()}

	// SAXException
	saxExcCls := &object.Class{Name: "SAXException", Dict: object.NewDict(), Bases: []*object.Class{i.exception}}
	saxExcGetDict := func(a []object.Object) *object.Dict {
		if len(a) < 1 {
			return nil
		}
		switch v := a[0].(type) {
		case *object.Instance:
			return v.Dict
		case *object.Exception:
			if v.Dict == nil {
				v.Dict = object.NewDict()
			}
			return v.Dict
		}
		return nil
	}
	saxExcCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		d := saxExcGetDict(a)
		if d == nil || len(a) < 2 {
			return object.None, nil
		}
		if s, ok := a[1].(*object.Str); ok {
			d.SetStr("_message", s)
			d.SetStr("args", &object.Tuple{V: []object.Object{s}})
		}
		if len(a) >= 3 {
			d.SetStr("_exception", a[2])
		} else {
			d.SetStr("_exception", object.None)
		}
		return object.None, nil
	}})
	saxExcCls.Dict.SetStr("getMessage", &object.BuiltinFunc{Name: "getMessage", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		d := saxExcGetDict(a)
		if d == nil {
			return &object.Str{V: ""}, nil
		}
		if v, ok := d.GetStr("_message"); ok {
			return v, nil
		}
		if len(a) >= 1 {
			if e, ok := a[0].(*object.Exception); ok {
				if e.Msg != "" {
					return &object.Str{V: e.Msg}, nil
				}
				if e.Args != nil && len(e.Args.V) >= 1 {
					if s, ok := e.Args.V[0].(*object.Str); ok {
						return s, nil
					}
				}
			}
		}
		return &object.Str{V: ""}, nil
	}})
	saxExcCls.Dict.SetStr("getException", &object.BuiltinFunc{Name: "getException", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		d := saxExcGetDict(a)
		if d == nil {
			return object.None, nil
		}
		if v, ok := d.GetStr("_exception"); ok {
			return v, nil
		}
		return object.None, nil
	}})
	saxExcCls.Dict.SetStr("__str__", &object.BuiltinFunc{Name: "__str__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		d := saxExcGetDict(a)
		if d != nil {
			if v, ok := d.GetStr("_message"); ok {
				if s, ok := v.(*object.Str); ok {
					return s, nil
				}
			}
		}
		if len(a) >= 1 {
			if e, ok := a[0].(*object.Exception); ok {
				if e.Msg != "" {
					return &object.Str{V: e.Msg}, nil
				}
				if e.Args != nil && len(e.Args.V) >= 1 {
					if s, ok := e.Args.V[0].(*object.Str); ok {
						return s, nil
					}
				}
			}
		}
		return &object.Str{V: "SAXException"}, nil
	}})
	m.Dict.SetStr("SAXException", saxExcCls)

	// SAXParseException
	saxParseExcCls := &object.Class{Name: "SAXParseException", Dict: object.NewDict(), Bases: []*object.Class{saxExcCls}}
	saxParseExcCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		d := saxExcGetDict(a)
		if d == nil || len(a) < 2 {
			return object.None, nil
		}
		if s, ok := a[1].(*object.Str); ok {
			d.SetStr("_message", s)
		}
		// a[2] = exception, a[3] = locator
		locator := object.Object(object.None)
		if len(a) >= 4 {
			locator = a[3]
		} else if len(a) >= 3 {
			locator = a[2]
		}
		lineNo := object.Object(object.IntFromInt64(-1))
		colNo := object.Object(object.IntFromInt64(-1))
		sysId := object.Object(object.None)
		pubId := object.Object(object.None)
		if locator != object.None {
			if loc, ok := locator.(*object.Instance); ok {
				if v, ok2 := loc.Dict.GetStr("_lineNumber"); ok2 {
					lineNo = v
				}
				if v, ok2 := loc.Dict.GetStr("_columnNumber"); ok2 {
					colNo = v
				}
				if v, ok2 := loc.Dict.GetStr("_systemId"); ok2 {
					sysId = v
				}
				if v, ok2 := loc.Dict.GetStr("_publicId"); ok2 {
					pubId = v
				}
			}
		}
		d.SetStr("_lineNumber", lineNo)
		d.SetStr("_columnNumber", colNo)
		d.SetStr("_systemId", sysId)
		d.SetStr("_publicId", pubId)
		return object.None, nil
	}})
	saxParseGetLocator := func(a []object.Object) *object.Instance {
		if len(a) < 1 {
			return nil
		}
		var args *object.Tuple
		switch v := a[0].(type) {
		case *object.Exception:
			args = v.Args
		}
		if args == nil || len(args.V) < 3 {
			return nil
		}
		if loc, ok := args.V[2].(*object.Instance); ok {
			return loc
		}
		return nil
	}
	saxParseExcCls.Dict.SetStr("getLineNumber", &object.BuiltinFunc{Name: "getLineNumber", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if loc := saxParseGetLocator(a); loc != nil {
			fn, err := interp.(*Interp).getAttr(loc, "getLineNumber")
			if err == nil {
				return interp.(*Interp).callObject(fn, []object.Object{loc}, nil)
			}
		}
		return object.IntFromInt64(-1), nil
	}})
	saxParseExcCls.Dict.SetStr("getColumnNumber", &object.BuiltinFunc{Name: "getColumnNumber", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if loc := saxParseGetLocator(a); loc != nil {
			fn, err := interp.(*Interp).getAttr(loc, "getColumnNumber")
			if err == nil {
				return interp.(*Interp).callObject(fn, []object.Object{loc}, nil)
			}
		}
		return object.IntFromInt64(-1), nil
	}})
	saxParseExcCls.Dict.SetStr("getSystemId", &object.BuiltinFunc{Name: "getSystemId", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if loc := saxParseGetLocator(a); loc != nil {
			fn, err := interp.(*Interp).getAttr(loc, "getSystemId")
			if err == nil {
				return interp.(*Interp).callObject(fn, []object.Object{loc}, nil)
			}
		}
		return object.None, nil
	}})
	saxParseExcCls.Dict.SetStr("getPublicId", &object.BuiltinFunc{Name: "getPublicId", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if loc := saxParseGetLocator(a); loc != nil {
			fn, err := interp.(*Interp).getAttr(loc, "getPublicId")
			if err == nil {
				return interp.(*Interp).callObject(fn, []object.Object{loc}, nil)
			}
		}
		return object.None, nil
	}})
	m.Dict.SetStr("SAXParseException", saxParseExcCls)

	// SAXNotRecognizedException, SAXNotSupportedException, SAXReaderNotAvailable
	saxNotRecCls := &object.Class{Name: "SAXNotRecognizedException", Dict: object.NewDict(), Bases: []*object.Class{saxExcCls}}
	saxNotSupCls := &object.Class{Name: "SAXNotSupportedException", Dict: object.NewDict(), Bases: []*object.Class{saxExcCls}}
	saxNotAvailCls := &object.Class{Name: "SAXReaderNotAvailable", Dict: object.NewDict(), Bases: []*object.Class{saxNotSupCls}}
	m.Dict.SetStr("SAXNotRecognizedException", saxNotRecCls)
	m.Dict.SetStr("SAXNotSupportedException", saxNotSupCls)
	m.Dict.SetStr("SAXReaderNotAvailable", saxNotAvailCls)

	// make_parser()
	m.Dict.SetStr("make_parser", &object.BuiltinFunc{Name: "make_parser", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return i.makeSAXParser(saxParseExcCls), nil
	}})

	// parse(source, handler, error_handler=...)
	m.Dict.SetStr("parse", &object.BuiltinFunc{Name: "parse", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		ii := interp.(*Interp)
		var data []byte
		switch v := a[0].(type) {
		case *object.Str:
			raw, err := etReadFile(v.V)
			if err != nil {
				return nil, object.Errorf(ii.valueErr, "parse: cannot read %q: %v", v.V, err)
			}
			data = raw
		case *object.Bytes:
			data = v.V
		case *object.Bytearray:
			data = v.V
		default:
			fn, err := ii.getAttr(a[0], "read")
			if err == nil {
				res, rerr := ii.callObject(fn, nil, nil)
				if rerr != nil {
					return nil, rerr
				}
				data, _ = asBytes(res)
			}
		}
		handler := a[1]
		return object.None, i.saxParse(data, handler, saxParseExcCls)
	}})

	// parseString(string, handler, error_handler=...)
	m.Dict.SetStr("parseString", &object.BuiltinFunc{Name: "parseString", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "parseString() requires bytes or str")
		}
		handler := a[1]
		return object.None, i.saxParse(data, handler, saxParseExcCls)
	}})

	// handler submodule reference
	m.Dict.SetStr("handler", &object.Module{Name: "xml.sax.handler", Dict: object.NewDict()})
	m.Dict.SetStr("saxutils", &object.Module{Name: "xml.sax.saxutils", Dict: object.NewDict()})
	m.Dict.SetStr("xmlreader", &object.Module{Name: "xml.sax.xmlreader", Dict: object.NewDict()})

	return m
}

// saxParse runs a SAX parse of data, calling Python handler methods.
func (i *Interp) saxParse(data []byte, handler object.Object, saxParseExcCls *object.Class) error {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false

	// Build AttributesImpl factory
	attrsCls := i.buildSaxAttrsClass()

	saxCallHandler := func(name string, args []object.Object) error {
		fn, err := i.getAttr(handler, name)
		if err != nil {
			return nil // not overridden, skip
		}
		_, err = i.callObject(fn, args, nil)
		return err
	}

	// startDocument
	if err := saxCallHandler("startDocument", []object.Object{}); err != nil {
		return err
	}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return object.Errorf(saxParseExcCls, "SAX parse error: %v", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := t.Name.Local
			attrsInst := &object.Instance{Class: attrsCls, Dict: object.NewDict()}
			attrMap := make(map[string]string)
			for _, a2 := range t.Attr {
				attrName := a2.Name.Local
				if a2.Name.Space != "" {
					attrName = "{" + a2.Name.Space + "}" + a2.Name.Local
				}
				attrMap[attrName] = a2.Value
			}
			saxAttrsStateMap.Store(attrsInst, attrMap)
			if err2 := saxCallHandler("startElement", []object.Object{&object.Str{V: name}, attrsInst}); err2 != nil {
				return err2
			}
		case xml.EndElement:
			name := t.Name.Local
			if err2 := saxCallHandler("endElement", []object.Object{&object.Str{V: name}}); err2 != nil {
				return err2
			}
		case xml.CharData:
			text := string(t)
			if err2 := saxCallHandler("characters", []object.Object{&object.Str{V: text}}); err2 != nil {
				return err2
			}
		case xml.Comment:
			// ignore
		case xml.ProcInst:
			if err2 := saxCallHandler("processingInstruction", []object.Object{
				&object.Str{V: t.Target},
				&object.Str{V: string(t.Inst)},
			}); err2 != nil {
				return err2
			}
		}
	}

	// endDocument
	return saxCallHandler("endDocument", []object.Object{})
}

// ── SAX Attrs ─────────────────────────────────────────────────────────────────

type saxAttrsRegistry struct{ m sync.Map }

func (r *saxAttrsRegistry) Store(k *object.Instance, v map[string]string) { r.m.Store(k, v) }
func (r *saxAttrsRegistry) Load(k *object.Instance) map[string]string {
	v, ok := r.m.Load(k)
	if !ok {
		return nil
	}
	return v.(map[string]string)
}

var saxAttrsStateMap saxAttrsRegistry

func (i *Interp) buildSaxAttrsClass() *object.Class {
	cls := &object.Class{Name: "AttributesImpl", Dict: object.NewDict()}

	cls.Dict.SetStr("getNames", &object.BuiltinFunc{Name: "getNames", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		if m2 == nil {
			return &object.List{V: nil}, nil
		}
		items := make([]object.Object, 0, len(m2))
		for k := range m2 {
			items = append(items, &object.Str{V: k})
		}
		return &object.List{V: items}, nil
	}})

	cls.Dict.SetStr("getType", &object.BuiltinFunc{Name: "getType", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "CDATA"}, nil
	}})

	cls.Dict.SetStr("getValue", &object.BuiltinFunc{Name: "getValue", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		if m2 == nil {
			return object.None, nil
		}
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		if v, ok := m2[key]; ok {
			return &object.Str{V: v}, nil
		}
		return nil, object.Errorf(i.keyErr, "attribute %q not found", key)
	}})

	cls.Dict.SetStr("getLength", &object.BuiltinFunc{Name: "getLength", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.IntFromInt64(0), nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		if m2 == nil {
			return object.IntFromInt64(0), nil
		}
		return object.IntFromInt64(int64(len(m2))), nil
	}})

	cls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.IntFromInt64(0), nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		if m2 == nil {
			return object.IntFromInt64(0), nil
		}
		return object.IntFromInt64(int64(len(m2))), nil
	}})

	cls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		if m2 == nil {
			return nil, object.Errorf(i.keyErr, "key not found")
		}
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		if v, ok := m2[key]; ok {
			return &object.Str{V: v}, nil
		}
		return nil, object.Errorf(i.keyErr, "key %q not found", key)
	}})

	cls.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		if m2 == nil {
			return object.False, nil
		}
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		_, ok := m2[key]
		return object.BoolOf(ok), nil
	}})

	cls.Dict.SetStr("keys", &object.BuiltinFunc{Name: "keys", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		if m2 == nil {
			return &object.List{V: nil}, nil
		}
		items := make([]object.Object, 0, len(m2))
		for k := range m2 {
			items = append(items, &object.Str{V: k})
		}
		return &object.List{V: items}, nil
	}})

	cls.Dict.SetStr("items", &object.BuiltinFunc{Name: "items", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		if m2 == nil {
			return &object.List{V: nil}, nil
		}
		items := make([]object.Object, 0, len(m2))
		for k, v := range m2 {
			items = append(items, &object.Tuple{V: []object.Object{&object.Str{V: k}, &object.Str{V: v}}})
		}
		return &object.List{V: items}, nil
	}})

	cls.Dict.SetStr("getQNameByName", &object.BuiltinFunc{Name: "getQNameByName", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		return a[1], nil
	}})

	cls.Dict.SetStr("getNameByQName", &object.BuiltinFunc{Name: "getNameByQName", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		return a[1], nil
	}})

	cls.Dict.SetStr("getValueByQName", &object.BuiltinFunc{Name: "getValueByQName", Call: func(ii any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		if m2 == nil {
			return nil, object.Errorf(i.keyErr, "key not found")
		}
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		if v, ok := m2[key]; ok {
			return &object.Str{V: v}, nil
		}
		return nil, object.Errorf(i.keyErr, "key %q not found", key)
	}})

	cls.Dict.SetStr("getQNames", &object.BuiltinFunc{Name: "getQNames", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		if m2 == nil {
			return &object.List{V: nil}, nil
		}
		items := make([]object.Object, 0, len(m2))
		for k := range m2 {
			items = append(items, &object.Str{V: k})
		}
		return &object.List{V: items}, nil
	}})

	cls.Dict.SetStr("copy", &object.BuiltinFunc{Name: "copy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Instance{Class: cls, Dict: object.NewDict()}, nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		newMap := make(map[string]string)
		for k, v := range m2 {
			newMap[k] = v
		}
		newInst := &object.Instance{Class: cls, Dict: object.NewDict()}
		saxAttrsStateMap.Store(newInst, newMap)
		return newInst, nil
	}})

	cls.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		if m2 == nil {
			return &object.List{V: nil}, nil
		}
		items := make([]object.Object, 0, len(m2))
		for k := range m2 {
			items = append(items, &object.Str{V: k})
		}
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(items) {
				return nil, false, nil
			}
			v := items[idx]
			idx++
			return v, true, nil
		}}, nil
	}})

	return cls
}

// ── xml.sax.handler ───────────────────────────────────────────────────────────

func (i *Interp) buildXmlSaxHandler() *object.Module {
	m := &object.Module{Name: "xml.sax.handler", Dict: object.NewDict()}

	// Constants
	m.Dict.SetStr("feature_namespaces", &object.Str{V: "http://xml.org/sax/features/namespaces"})
	m.Dict.SetStr("feature_namespace_prefixes", &object.Str{V: "http://xml.org/sax/features/namespace-prefixes"})
	m.Dict.SetStr("feature_string_interning", &object.Str{V: "http://xml.org/sax/features/string-interning"})
	m.Dict.SetStr("feature_validation", &object.Str{V: "http://xml.org/sax/features/validation"})
	m.Dict.SetStr("feature_external_ges", &object.Str{V: "http://xml.org/sax/features/external-general-entities"})
	m.Dict.SetStr("feature_external_pes", &object.Str{V: "http://xml.org/sax/features/external-parameter-entities"})
	m.Dict.SetStr("property_lexical_handler", &object.Str{V: "http://xml.org/sax/properties/lexical-handler"})
	m.Dict.SetStr("property_declaration_handler", &object.Str{V: "http://xml.org/sax/properties/declaration-handler"})
	m.Dict.SetStr("property_dom_node", &object.Str{V: "http://xml.org/sax/properties/dom-node"})
	m.Dict.SetStr("property_xml_string", &object.Str{V: "http://xml.org/sax/properties/xml-string"})
	m.Dict.SetStr("property_encoding", &object.Str{V: "http://www.python.org/sax/properties/encoding"})
	m.Dict.SetStr("property_interning_dict", &object.Str{V: "http://www.python.org/sax/properties/interning-dict"})

	// ContentHandler base class
	contentHandlerCls := &object.Class{Name: "ContentHandler", Dict: object.NewDict()}
	noopMethods := []string{
		"startDocument", "endDocument",
		"startPrefixMapping", "endPrefixMapping",
		"startElement", "endElement",
		"startElementNS", "endElementNS",
		"characters", "ignorableWhitespace",
		"processingInstruction", "skippedEntity",
		"setDocumentLocator",
	}
	for _, name := range noopMethods {
		n := name
		contentHandlerCls.Dict.SetStr(n, &object.BuiltinFunc{Name: n, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	}
	m.Dict.SetStr("ContentHandler", contentHandlerCls)

	// DTDHandler base class
	dtdHandlerCls := &object.Class{Name: "DTDHandler", Dict: object.NewDict()}
	for _, name := range []string{"notationDecl", "unparsedEntityDecl"} {
		n := name
		dtdHandlerCls.Dict.SetStr(n, &object.BuiltinFunc{Name: n, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	}
	m.Dict.SetStr("DTDHandler", dtdHandlerCls)

	// EntityResolver
	entityResolverCls := &object.Class{Name: "EntityResolver", Dict: object.NewDict()}
	entityResolverCls.Dict.SetStr("resolveEntity", &object.BuiltinFunc{Name: "resolveEntity", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 3 {
			return a[2], nil
		}
		return object.None, nil
	}})
	m.Dict.SetStr("EntityResolver", entityResolverCls)

	// ErrorHandler
	errorHandlerCls := &object.Class{Name: "ErrorHandler", Dict: object.NewDict()}
	errorHandlerCls.Dict.SetStr("error", &object.BuiltinFunc{Name: "error", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			if exc, ok := a[1].(*object.Exception); ok {
				return nil, exc
			}
			return nil, object.Errorf(i.exception, "%v", a[1])
		}
		return nil, object.Errorf(i.exception, "error")
	}})
	errorHandlerCls.Dict.SetStr("fatalError", &object.BuiltinFunc{Name: "fatalError", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			if exc, ok := a[1].(*object.Exception); ok {
				return nil, exc
			}
			return nil, object.Errorf(i.exception, "%v", a[1])
		}
		return nil, object.Errorf(i.exception, "fatalError")
	}})
	errorHandlerCls.Dict.SetStr("warning", &object.BuiltinFunc{Name: "warning", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			ii := interp.(*Interp)
			fmt.Fprint(ii.Stdout, object.Str_(a[1])+"\n")
		}
		return object.None, nil
	}})
	m.Dict.SetStr("ErrorHandler", errorHandlerCls)

	// LexicalHandler base class (no-op stubs)
	lexHandlerCls := &object.Class{Name: "LexicalHandler", Dict: object.NewDict()}
	for _, name := range []string{"comment", "startDTD", "endDTD", "startCDATA", "endCDATA"} {
		n := name
		lexHandlerCls.Dict.SetStr(n, &object.BuiltinFunc{Name: n, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	}
	m.Dict.SetStr("LexicalHandler", lexHandlerCls)

	// all_features / all_properties lists
	features := []object.Object{
		&object.Str{V: "http://xml.org/sax/features/namespaces"},
		&object.Str{V: "http://xml.org/sax/features/namespace-prefixes"},
		&object.Str{V: "http://xml.org/sax/features/string-interning"},
		&object.Str{V: "http://xml.org/sax/features/validation"},
		&object.Str{V: "http://xml.org/sax/features/external-general-entities"},
		&object.Str{V: "http://xml.org/sax/features/external-parameter-entities"},
	}
	props := []object.Object{
		&object.Str{V: "http://xml.org/sax/properties/lexical-handler"},
		&object.Str{V: "http://xml.org/sax/properties/dom-node"},
		&object.Str{V: "http://xml.org/sax/properties/declaration-handler"},
		&object.Str{V: "http://xml.org/sax/properties/xml-string"},
		&object.Str{V: "http://www.python.org/sax/properties/encoding"},
		&object.Str{V: "http://www.python.org/sax/properties/interning-dict"},
	}
	m.Dict.SetStr("all_features", &object.List{V: features})
	m.Dict.SetStr("all_properties", &object.List{V: props})
	m.Dict.SetStr("version", &object.Str{V: "2.0beta"})

	return m
}

// ── xml.sax.saxutils ─────────────────────────────────────────────────────────

func (i *Interp) buildXmlSaxUtils() *object.Module {
	m := &object.Module{Name: "xml.sax.saxutils", Dict: object.NewDict()}

	// escape(data, entities={})
	m.Dict.SetStr("escape", &object.BuiltinFunc{Name: "escape", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		s := ""
		if sv, ok := a[0].(*object.Str); ok {
			s = sv.V
		}
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		// extra entities
		if len(a) >= 2 {
			if d, ok := a[1].(*object.Dict); ok {
				ks, vs := d.Items()
				for idx2, k := range ks {
					if ks2, ok2 := k.(*object.Str); ok2 {
						if vs2, ok3 := vs[idx2].(*object.Str); ok3 {
							s = strings.ReplaceAll(s, ks2.V, vs2.V)
						}
					}
				}
			}
		}
		return &object.Str{V: s}, nil
	}})

	// unescape(data, entities={})
	m.Dict.SetStr("unescape", &object.BuiltinFunc{Name: "unescape", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		s := ""
		if sv, ok := a[0].(*object.Str); ok {
			s = sv.V
		}
		// extra entities first (before standard ones)
		if len(a) >= 2 {
			if d, ok := a[1].(*object.Dict); ok {
				ks, vs := d.Items()
				for idx2, k := range ks {
					if ks2, ok2 := k.(*object.Str); ok2 {
						if vs2, ok3 := vs[idx2].(*object.Str); ok3 {
							s = strings.ReplaceAll(s, ks2.V, vs2.V)
						}
					}
				}
			}
		}
		s = strings.ReplaceAll(s, "&amp;", "&")
		s = strings.ReplaceAll(s, "&lt;", "<")
		s = strings.ReplaceAll(s, "&gt;", ">")
		return &object.Str{V: s}, nil
	}})

	// quoteattr(data, entities={})
	m.Dict.SetStr("quoteattr", &object.BuiltinFunc{Name: "quoteattr", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: `""`}, nil
		}
		s := ""
		if sv, ok := a[0].(*object.Str); ok {
			s = sv.V
		}
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		if strings.ContainsRune(s, '"') && !strings.ContainsRune(s, '\'') {
			return &object.Str{V: "'" + s + "'"}, nil
		}
		s = strings.ReplaceAll(s, "\"", "&quot;")
		return &object.Str{V: "\"" + s + "\""}, nil
	}})

	// XMLGenerator class (ContentHandler subclass that writes XML)
	xmlGenCls := &object.Class{Name: "XMLGenerator", Dict: object.NewDict()}

	xmlGenCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		out := object.Object(object.None)
		encoding := "iso-8859-1"
		shortEmpty := false
		if len(a) >= 2 {
			out = a[1]
		}
		if len(a) >= 3 {
			if s, ok := a[2].(*object.Str); ok {
				encoding = s.V
			}
		}
		if len(a) >= 4 {
			shortEmpty = object.Truthy(a[3])
		}
		if kw != nil {
			if v, ok := kw.GetStr("out"); ok {
				out = v
			}
			if v, ok := kw.GetStr("encoding"); ok {
				if s, ok := v.(*object.Str); ok {
					encoding = s.V
				}
			}
			if v, ok := kw.GetStr("short_empty_elements"); ok {
				shortEmpty = object.Truthy(v)
			}
		}
		self.Dict.SetStr("_out", out)
		self.Dict.SetStr("_encoding", &object.Str{V: encoding})
		self.Dict.SetStr("_short_empty_elements", object.BoolOf(shortEmpty))
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("startDocument", &object.BuiltinFunc{Name: "startDocument", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		out, _ := self.Dict.GetStr("_out")
		enc := "iso-8859-1"
		if v, ok := self.Dict.GetStr("_encoding"); ok {
			if s, ok := v.(*object.Str); ok {
				enc = s.V
			}
		}
		content := `<?xml version="1.0" encoding="` + enc + `"?>` + "\n"
		saxGenWrite(interp.(*Interp), out, content)
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("endDocument", &object.BuiltinFunc{Name: "endDocument", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("startElement", &object.BuiltinFunc{Name: "startElement", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		out, _ := self.Dict.GetStr("_out")
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		var buf strings.Builder
		buf.WriteByte('<')
		buf.WriteString(name)
		// attrs
		if attrsInst, ok := a[2].(*object.Instance); ok {
			m2 := saxAttrsStateMap.Load(attrsInst)
			for k, v := range m2 {
				buf.WriteByte(' ')
				buf.WriteString(k)
				buf.WriteString(`="`)
				buf.WriteString(xmlEscapeAttr(v))
				buf.WriteByte('"')
			}
		} else if d, ok := a[2].(*object.Dict); ok {
			ks, vs := d.Items()
			for idx2, k := range ks {
				if ks2, ok2 := k.(*object.Str); ok2 {
					buf.WriteByte(' ')
					buf.WriteString(ks2.V)
					buf.WriteString(`="`)
					if vs2, ok3 := vs[idx2].(*object.Str); ok3 {
						buf.WriteString(xmlEscapeAttr(vs2.V))
					}
					buf.WriteByte('"')
				}
			}
		}
		buf.WriteByte('>')
		saxGenWrite(interp.(*Interp), out, buf.String())
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("endElement", &object.BuiltinFunc{Name: "endElement", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		out, _ := self.Dict.GetStr("_out")
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		saxGenWrite(interp.(*Interp), out, "</"+name+">")
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("characters", &object.BuiltinFunc{Name: "characters", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		out, _ := self.Dict.GetStr("_out")
		content := ""
		if s, ok := a[1].(*object.Str); ok {
			content = s.V
		}
		saxGenWrite(interp.(*Interp), out, xmlEscapeText(content))
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("ignorableWhitespace", &object.BuiltinFunc{Name: "ignorableWhitespace", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		out, _ := self.Dict.GetStr("_out")
		content := ""
		if s, ok := a[1].(*object.Str); ok {
			content = s.V
		}
		saxGenWrite(interp.(*Interp), out, content)
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("processingInstruction", &object.BuiltinFunc{Name: "processingInstruction", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		out, _ := self.Dict.GetStr("_out")
		target := ""
		data := ""
		if s, ok := a[1].(*object.Str); ok {
			target = s.V
		}
		if s, ok := a[2].(*object.Str); ok {
			data = s.V
		}
		saxGenWrite(interp.(*Interp), out, "<?"+target+" "+data+"?>")
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("setDocumentLocator", &object.BuiltinFunc{Name: "setDocumentLocator", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			a[0].(*object.Instance).Dict.SetStr("_locator", a[1])
		}
		return object.None, nil
	}})
	xmlGenCls.Dict.SetStr("startPrefixMapping", &object.BuiltinFunc{Name: "startPrefixMapping", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	xmlGenCls.Dict.SetStr("endPrefixMapping", &object.BuiltinFunc{Name: "endPrefixMapping", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	// startElementNS / endElementNS stubs
	xmlGenCls.Dict.SetStr("startElementNS", &object.BuiltinFunc{Name: "startElementNS", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	xmlGenCls.Dict.SetStr("endElementNS", &object.BuiltinFunc{Name: "endElementNS", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	m.Dict.SetStr("XMLGenerator", xmlGenCls)

	// XMLFilterBase stub
	xmlFilterCls := &object.Class{Name: "XMLFilterBase", Dict: object.NewDict()}
	xmlFilterCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			a[0].(*object.Instance).Dict.SetStr("_parent", a[1])
		}
		return object.None, nil
	}})
	m.Dict.SetStr("XMLFilterBase", xmlFilterCls)

	return m
}

func saxGenWrite(ii *Interp, out object.Object, content string) {
	if out == nil || out == object.None {
		return
	}
	fn, err := ii.getAttr(out, "write")
	if err != nil {
		return
	}
	ii.callObject(fn, []object.Object{&object.Str{V: content}}, nil)
}

// ── xml.sax.xmlreader ────────────────────────────────────────────────────────

func (i *Interp) buildXmlSaxXmlReader() *object.Module {
	m := &object.Module{Name: "xml.sax.xmlreader", Dict: object.NewDict()}

	// XMLReader
	xmlReaderCls := &object.Class{Name: "XMLReader", Dict: object.NewDict()}
	for _, name := range []string{
		"parse", "getContentHandler", "setContentHandler",
		"getErrorHandler", "setErrorHandler",
		"getFeature", "setFeature",
		"getProperty", "setProperty",
	} {
		n := name
		xmlReaderCls.Dict.SetStr(n, &object.BuiltinFunc{Name: n, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	}
	m.Dict.SetStr("XMLReader", xmlReaderCls)

	// IncrementalParser
	incParserCls := &object.Class{Name: "IncrementalParser", Dict: object.NewDict()}
	for _, name := range []string{"feed", "close", "prepareParser", "reset"} {
		n := name
		incParserCls.Dict.SetStr(n, &object.BuiltinFunc{Name: n, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	}
	m.Dict.SetStr("IncrementalParser", incParserCls)

	// Locator
	locatorCls := &object.Class{Name: "Locator", Dict: object.NewDict()}
	for _, name := range []string{"getPublicId", "getSystemId"} {
		n := name
		locatorCls.Dict.SetStr(n, &object.BuiltinFunc{Name: n, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	}
	locatorCls.Dict.SetStr("getColumnNumber", &object.BuiltinFunc{Name: "getColumnNumber", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.IntFromInt64(-1), nil
	}})
	locatorCls.Dict.SetStr("getLineNumber", &object.BuiltinFunc{Name: "getLineNumber", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.IntFromInt64(-1), nil
	}})
	m.Dict.SetStr("Locator", locatorCls)

	// InputSource
	inputSourceCls := &object.Class{Name: "InputSource", Dict: object.NewDict()}
	inputSourceCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if len(a) >= 2 {
			self.Dict.SetStr("_system_id", a[1])
		} else {
			self.Dict.SetStr("_system_id", object.None)
		}
		self.Dict.SetStr("_byte_stream", object.None)
		self.Dict.SetStr("_char_stream", object.None)
		self.Dict.SetStr("_encoding", object.None)
		self.Dict.SetStr("_public_id", object.None)
		return object.None, nil
	}})
	for _, pair := range [][2]string{
		{"setByteStream", "_byte_stream"}, {"getByteStream", "_byte_stream"},
		{"setCharacterStream", "_char_stream"}, {"getCharacterStream", "_char_stream"},
		{"setEncoding", "_encoding"}, {"getEncoding", "_encoding"},
		{"setSystemId", "_system_id"}, {"getSystemId", "_system_id"},
		{"setPublicId", "_public_id"}, {"getPublicId", "_public_id"},
	} {
		method, attr := pair[0], pair[1]
		m2 := method
		a2 := attr
		if strings.HasPrefix(m2, "set") {
			inputSourceCls.Dict.SetStr(m2, &object.BuiltinFunc{Name: m2, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) >= 2 {
					a[0].(*object.Instance).Dict.SetStr(a2, a[1])
				}
				return object.None, nil
			}})
		} else {
			inputSourceCls.Dict.SetStr(m2, &object.BuiltinFunc{Name: m2, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) < 1 {
					return object.None, nil
				}
				if v, ok := a[0].(*object.Instance).Dict.GetStr(a2); ok {
					return v, nil
				}
				return object.None, nil
			}})
		}
	}
	m.Dict.SetStr("InputSource", inputSourceCls)

	// AttributesImpl(attrs_dict)
	attrsCls := i.buildSaxAttrsClass()
	attrsCls2 := &object.Class{Name: "AttributesImpl", Dict: object.NewDict()}
	attrsCls2.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		m3 := make(map[string]string)
		if d, ok := a[1].(*object.Dict); ok {
			ks, vs := d.Items()
			for idx2, k := range ks {
				if ks2, ok2 := k.(*object.Str); ok2 {
					if vs2, ok3 := vs[idx2].(*object.Str); ok3 {
						m3[ks2.V] = vs2.V
					}
				}
			}
		}
		saxAttrsStateMap.Store(self, m3)
		return object.None, nil
	}})
	// Copy all methods from attrsCls
	for _, name := range []string{
		"getNames", "getType", "getValue", "getLength",
		"__len__", "__getitem__", "__contains__", "__iter__",
		"keys", "items", "copy",
		"getQNameByName", "getNameByQName", "getValueByQName", "getQNames",
	} {
		if v, ok := attrsCls.Dict.GetStr(name); ok {
			attrsCls2.Dict.SetStr(name, v)
		}
	}
	m.Dict.SetStr("AttributesImpl", attrsCls2)

	// AttributesNSImpl stub
	attrsNSCls := &object.Class{Name: "AttributesNSImpl", Dict: object.NewDict()}
	attrsNSCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	m.Dict.SetStr("AttributesNSImpl", attrsNSCls)

	return m
}

// makeSAXParser returns a simple ExpatParser-like object for xml.sax.make_parser().
func (i *Interp) makeSAXParser(saxParseExcCls *object.Class) *object.Instance {
	cls := &object.Class{Name: "ExpatParser", Dict: object.NewDict()}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	inst.Dict.SetStr("_handler", object.None)
	inst.Dict.SetStr("_error_handler", object.None)

	cls.Dict.SetStr("setContentHandler", &object.BuiltinFunc{Name: "setContentHandler", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			a[0].(*object.Instance).Dict.SetStr("_handler", a[1])
		}
		return object.None, nil
	}})
	cls.Dict.SetStr("getContentHandler", &object.BuiltinFunc{Name: "getContentHandler", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		if v, ok := a[0].(*object.Instance).Dict.GetStr("_handler"); ok {
			return v, nil
		}
		return object.None, nil
	}})
	cls.Dict.SetStr("parse", &object.BuiltinFunc{Name: "parse", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		var data []byte
		switch v := a[1].(type) {
		case *object.Bytes:
			data = v.V
		case *object.Str:
			data = []byte(v.V)
		default:
			ii := interp.(*Interp)
			fn, err := ii.getAttr(a[1], "read")
			if err == nil {
				res, rerr := ii.callObject(fn, nil, nil)
				if rerr != nil {
					return nil, rerr
				}
				data, _ = asBytes(res)
			}
		}
		handler, _ := self.Dict.GetStr("_handler")
		return object.None, i.saxParse(data, handler, saxParseExcCls)
	}})
	cls.Dict.SetStr("setErrorHandler", &object.BuiltinFunc{Name: "setErrorHandler", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			a[0].(*object.Instance).Dict.SetStr("_error_handler", a[1])
		}
		return object.None, nil
	}})
	cls.Dict.SetStr("getErrorHandler", &object.BuiltinFunc{Name: "getErrorHandler", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		if v, ok := a[0].(*object.Instance).Dict.GetStr("_error_handler"); ok {
			return v, nil
		}
		return object.None, nil
	}})
	cls.Dict.SetStr("setDTDHandler", &object.BuiltinFunc{Name: "setDTDHandler", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			a[0].(*object.Instance).Dict.SetStr("_dtd_handler", a[1])
		}
		return object.None, nil
	}})
	cls.Dict.SetStr("getDTDHandler", &object.BuiltinFunc{Name: "getDTDHandler", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		if v, ok := a[0].(*object.Instance).Dict.GetStr("_dtd_handler"); ok {
			return v, nil
		}
		return object.None, nil
	}})
	cls.Dict.SetStr("setEntityResolver", &object.BuiltinFunc{Name: "setEntityResolver", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			a[0].(*object.Instance).Dict.SetStr("_entity_resolver", a[1])
		}
		return object.None, nil
	}})
	cls.Dict.SetStr("getEntityResolver", &object.BuiltinFunc{Name: "getEntityResolver", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		if v, ok := a[0].(*object.Instance).Dict.GetStr("_entity_resolver"); ok {
			return v, nil
		}
		return object.None, nil
	}})
	cls.Dict.SetStr("reset", &object.BuiltinFunc{Name: "reset", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	cls.Dict.SetStr("setProperty", &object.BuiltinFunc{Name: "setProperty", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	cls.Dict.SetStr("getProperty", &object.BuiltinFunc{Name: "getProperty", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	cls.Dict.SetStr("setFeature", &object.BuiltinFunc{Name: "setFeature", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	cls.Dict.SetStr("getFeature", &object.BuiltinFunc{Name: "getFeature", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})
	return inst
}

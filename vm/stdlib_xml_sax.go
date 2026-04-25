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
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		if m2 != nil {
			if _, ok := m2[key]; ok {
				return a[1], nil
			}
		}
		return nil, object.Errorf(i.keyErr, "%q", key)
	}})

	cls.Dict.SetStr("getNameByQName", &object.BuiltinFunc{Name: "getNameByQName", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		if m2 != nil {
			if _, ok := m2[key]; ok {
				return a[1], nil
			}
		}
		return nil, object.Errorf(i.keyErr, "%q", key)
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

	cls.Dict.SetStr("values", &object.BuiltinFunc{Name: "values", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		if m2 == nil {
			return &object.List{V: nil}, nil
		}
		items := make([]object.Object, 0, len(m2))
		for _, v := range m2 {
			items = append(items, &object.Str{V: v})
		}
		return &object.List{V: items}, nil
	}})

	cls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		m2 := saxAttrsStateMap.Load(self)
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		if m2 != nil {
			if v, ok := m2[key]; ok {
				return &object.Str{V: v}, nil
			}
		}
		if len(a) >= 3 {
			return a[2], nil
		}
		return object.None, nil
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

// saxutilsApplyEntities applies string replacements from a Python dict.
func saxutilsApplyEntities(s string, d *object.Dict) string {
	if d == nil {
		return s
	}
	ks, vs := d.Items()
	for idx, k := range ks {
		if ks2, ok := k.(*object.Str); ok {
			if vs2, ok := vs[idx].(*object.Str); ok {
				s = strings.ReplaceAll(s, ks2.V, vs2.V)
			}
		}
	}
	return s
}

// saxutilsEscape matches Python's xml.sax.saxutils.escape order: & > < then entities.
func saxutilsEscape(s string, entities *object.Dict) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	return saxutilsApplyEntities(s, entities)
}

// saxutilsQuoteattrStr quotes an already-escaped string as an XML attribute value.
func saxutilsQuoteattrStr(s string) string {
	hasDouble := strings.ContainsRune(s, '"')
	hasSingle := strings.ContainsRune(s, '\'')
	if hasDouble {
		if hasSingle {
			return `"` + strings.ReplaceAll(s, `"`, "&quot;") + `"`
		}
		return "'" + s + "'"
	}
	return `"` + s + `"`
}

func (i *Interp) buildXmlSaxUtils() *object.Module {
	m := &object.Module{Name: "xml.sax.saxutils", Dict: object.NewDict()}

	// escape(data, entities={})
	m.Dict.SetStr("escape", &object.BuiltinFunc{Name: "escape", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		s, _ := a[0].(*object.Str)
		if s == nil {
			return &object.Str{V: ""}, nil
		}
		var ents *object.Dict
		if len(a) >= 2 {
			ents, _ = a[1].(*object.Dict)
		}
		return &object.Str{V: saxutilsEscape(s.V, ents)}, nil
	}})

	// unescape(data, entities={})
	// Order: &lt;→< , &gt;→> , custom entities, &amp;→& (last)
	m.Dict.SetStr("unescape", &object.BuiltinFunc{Name: "unescape", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		sv, _ := a[0].(*object.Str)
		if sv == nil {
			return &object.Str{V: ""}, nil
		}
		s := sv.V
		s = strings.ReplaceAll(s, "&lt;", "<")
		s = strings.ReplaceAll(s, "&gt;", ">")
		if len(a) >= 2 {
			if d, ok := a[1].(*object.Dict); ok {
				s = saxutilsApplyEntities(s, d)
			}
		}
		s = strings.ReplaceAll(s, "&amp;", "&")
		return &object.Str{V: s}, nil
	}})

	// quoteattr(data, entities={})
	// Escapes &, >, <, \n→&#10;, \r→&#13;, \t→&#9; then quotes.
	m.Dict.SetStr("quoteattr", &object.BuiltinFunc{Name: "quoteattr", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: `""`}, nil
		}
		sv, _ := a[0].(*object.Str)
		if sv == nil {
			return &object.Str{V: `""`}, nil
		}
		var userEnts *object.Dict
		if len(a) >= 2 {
			userEnts, _ = a[1].(*object.Dict)
		}
		// Build merged entity map: user + whitespace specials
		wsEnts := object.NewDict()
		wsEnts.SetStr("\n", &object.Str{V: "&#10;"})
		wsEnts.SetStr("\r", &object.Str{V: "&#13;"})
		wsEnts.SetStr("\t", &object.Str{V: "&#9;"})
		if userEnts != nil {
			ks, vs := userEnts.Items()
			for idx, k := range ks {
				wsEnts.Set(k, vs[idx])
			}
		}
		s := saxutilsEscape(sv.V, wsEnts)
		return &object.Str{V: saxutilsQuoteattrStr(s)}, nil
	}})

	// ── XMLGenerator ────────────────────────────────────────────────────────

	xmlGenCls := &object.Class{Name: "XMLGenerator", Dict: object.NewDict()}

	// helpers that operate on an XMLGenerator instance
	xmlGenGetOut := func(self *object.Instance) object.Object {
		if v, ok := self.Dict.GetStr("_out"); ok {
			return v
		}
		return object.None
	}
	xmlGenShortEmpty := func(self *object.Instance) bool {
		if v, ok := self.Dict.GetStr("_short_empty_elements"); ok {
			return object.Truthy(v)
		}
		return false
	}
	xmlGenPending := func(self *object.Instance) bool {
		if v, ok := self.Dict.GetStr("_pending_start_element"); ok {
			return object.Truthy(v)
		}
		return false
	}
	xmlGenFlushPending := func(ii *Interp, self *object.Instance) {
		if xmlGenPending(self) {
			saxGenWrite(ii, xmlGenGetOut(self), ">")
			self.Dict.SetStr("_pending_start_element", object.False)
		}
	}

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
		self.Dict.SetStr("_pending_start_element", object.False)
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("startDocument", &object.BuiltinFunc{Name: "startDocument", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		enc := "iso-8859-1"
		if v, ok := self.Dict.GetStr("_encoding"); ok {
			if s, ok := v.(*object.Str); ok {
				enc = s.V
			}
		}
		saxGenWrite(interp.(*Interp), xmlGenGetOut(self), `<?xml version="1.0" encoding="`+enc+`"?>`+"\n")
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("endDocument", &object.BuiltinFunc{Name: "endDocument", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		out := xmlGenGetOut(self)
		if out == nil || out == object.None {
			return object.None, nil
		}
		fn, err := interp.(*Interp).getAttr(out, "flush")
		if err == nil {
			interp.(*Interp).callObject(fn, []object.Object{out}, nil)
		}
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("startElement", &object.BuiltinFunc{Name: "startElement", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		ii := interp.(*Interp)
		self := a[0].(*object.Instance)
		xmlGenFlushPending(ii, self)
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		var buf strings.Builder
		buf.WriteByte('<')
		buf.WriteString(name)
		writeAttrs := func(k, v string) {
			buf.WriteByte(' ')
			buf.WriteString(k)
			buf.WriteByte('=')
			buf.WriteString(saxutilsQuoteattrStr(saxutilsEscape(v, nil)))
		}
		if attrsInst, ok := a[2].(*object.Instance); ok {
			m2 := saxAttrsStateMap.Load(attrsInst)
			for k, v := range m2 {
				writeAttrs(k, v)
			}
		} else if d, ok := a[2].(*object.Dict); ok {
			ks, vs := d.Items()
			for idx2, k := range ks {
				if ks2, ok2 := k.(*object.Str); ok2 {
					v := ""
					if vs2, ok3 := vs[idx2].(*object.Str); ok3 {
						v = vs2.V
					}
					writeAttrs(ks2.V, v)
				}
			}
		}
		if xmlGenShortEmpty(self) {
			self.Dict.SetStr("_pending_start_element", object.True)
			saxGenWrite(ii, xmlGenGetOut(self), buf.String())
		} else {
			buf.WriteByte('>')
			saxGenWrite(ii, xmlGenGetOut(self), buf.String())
		}
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("endElement", &object.BuiltinFunc{Name: "endElement", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		ii := interp.(*Interp)
		self := a[0].(*object.Instance)
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		if xmlGenPending(self) {
			saxGenWrite(ii, xmlGenGetOut(self), "/>")
			self.Dict.SetStr("_pending_start_element", object.False)
		} else {
			saxGenWrite(ii, xmlGenGetOut(self), "</"+name+">")
		}
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("characters", &object.BuiltinFunc{Name: "characters", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		ii := interp.(*Interp)
		self := a[0].(*object.Instance)
		content := ""
		if s, ok := a[1].(*object.Str); ok {
			content = s.V
		}
		if content == "" {
			return object.None, nil
		}
		xmlGenFlushPending(ii, self)
		saxGenWrite(ii, xmlGenGetOut(self), xmlEscapeText(content))
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("ignorableWhitespace", &object.BuiltinFunc{Name: "ignorableWhitespace", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		ii := interp.(*Interp)
		self := a[0].(*object.Instance)
		content := ""
		if s, ok := a[1].(*object.Str); ok {
			content = s.V
		}
		if content == "" {
			return object.None, nil
		}
		xmlGenFlushPending(ii, self)
		saxGenWrite(ii, xmlGenGetOut(self), content)
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("processingInstruction", &object.BuiltinFunc{Name: "processingInstruction", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		ii := interp.(*Interp)
		self := a[0].(*object.Instance)
		xmlGenFlushPending(ii, self)
		target, data2 := "", ""
		if s, ok := a[1].(*object.Str); ok {
			target = s.V
		}
		if s, ok := a[2].(*object.Str); ok {
			data2 = s.V
		}
		saxGenWrite(ii, xmlGenGetOut(self), "<?"+target+" "+data2+"?>")
		return object.None, nil
	}})

	xmlGenCls.Dict.SetStr("setDocumentLocator", &object.BuiltinFunc{Name: "setDocumentLocator", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			a[0].(*object.Instance).Dict.SetStr("_locator", a[1])
		}
		return object.None, nil
	}})
	for _, n := range []string{"startPrefixMapping", "endPrefixMapping", "startElementNS", "endElementNS", "skippedEntity"} {
		nm := n
		xmlGenCls.Dict.SetStr(nm, &object.BuiltinFunc{Name: nm, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		}})
	}

	m.Dict.SetStr("XMLGenerator", xmlGenCls)

	// ── XMLFilterBase ────────────────────────────────────────────────────────

	xmlFilterCls := &object.Class{Name: "XMLFilterBase", Dict: object.NewDict()}

	xmlFilterCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if len(a) >= 2 {
			self.Dict.SetStr("_parent", a[1])
		} else {
			self.Dict.SetStr("_parent", object.None)
		}
		self.Dict.SetStr("_cont_handler", object.None)
		self.Dict.SetStr("_err_handler", object.None)
		self.Dict.SetStr("_dtd_handler", object.None)
		self.Dict.SetStr("_ent_handler", object.None)
		return object.None, nil
	}})

	xmlFilterCls.Dict.SetStr("getParent", &object.BuiltinFunc{Name: "getParent", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		if v, ok := a[0].(*object.Instance).Dict.GetStr("_parent"); ok {
			return v, nil
		}
		return object.None, nil
	}})
	xmlFilterCls.Dict.SetStr("setParent", &object.BuiltinFunc{Name: "setParent", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			a[0].(*object.Instance).Dict.SetStr("_parent", a[1])
		}
		return object.None, nil
	}})

	// Handler get/set pairs
	for _, pair := range [][2]string{
		{"setContentHandler", "_cont_handler"}, {"getContentHandler", "_cont_handler"},
		{"setErrorHandler", "_err_handler"}, {"getErrorHandler", "_err_handler"},
		{"setDTDHandler", "_dtd_handler"}, {"getDTDHandler", "_dtd_handler"},
		{"setEntityResolver", "_ent_handler"}, {"getEntityResolver", "_ent_handler"},
	} {
		method, attr := pair[0], pair[1]
		m2, a2 := method, attr
		if strings.HasPrefix(m2, "set") {
			xmlFilterCls.Dict.SetStr(m2, &object.BuiltinFunc{Name: m2, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) >= 2 {
					a[0].(*object.Instance).Dict.SetStr(a2, a[1])
				}
				return object.None, nil
			}})
		} else {
			xmlFilterCls.Dict.SetStr(m2, &object.BuiltinFunc{Name: m2, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
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

	// XMLReader delegation to _parent
	for _, n := range []string{"getFeature", "setFeature", "getProperty", "setProperty"} {
		nm := n
		xmlFilterCls.Dict.SetStr(nm, &object.BuiltinFunc{Name: nm, Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			parent, ok := self.Dict.GetStr("_parent")
			if !ok || parent == object.None {
				return object.None, nil
			}
			ii := interp.(*Interp)
			fn, err := ii.getAttr(parent, nm)
			if err != nil {
				return object.None, nil
			}
			return ii.callObject(fn, a[1:], nil)
		}})
	}

	m.Dict.SetStr("XMLFilterBase", xmlFilterCls)

	// prepare_input_source(source, base="")
	m.Dict.SetStr("prepare_input_source", &object.BuiltinFunc{Name: "prepare_input_source", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inputSourceCls := &object.Class{Name: "InputSource", Dict: object.NewDict()}
		src := a[0]
		switch v := src.(type) {
		case *object.Str:
			inst := &object.Instance{Class: inputSourceCls, Dict: object.NewDict()}
			inst.Dict.SetStr("_system_id", v)
			inst.Dict.SetStr("_public_id", object.None)
			inst.Dict.SetStr("_byte_stream", object.None)
			inst.Dict.SetStr("_char_stream", object.None)
			inst.Dict.SetStr("_encoding", object.None)
			return inst, nil
		case *object.Instance:
			return v, nil
		}
		return object.None, nil
	}})

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

// saxNSAttrsState holds NS-aware attribute data for AttributesNSImpl instances.
type saxNSAttrsState struct {
	attrKeys  [][2]string // (ns_uri, local_name) pairs
	attrVals  []string    // attribute values parallel to attrKeys
	qnameKeys [][2]string // (ns_uri, local_name) pairs for qnames
	qnameVals []string    // qname strings parallel to qnameKeys
}

var saxNSAttrsStateMap sync.Map

// ── xml.sax.xmlreader ────────────────────────────────────────────────────────

func (i *Interp) buildXmlSaxXmlReader() *object.Module {
	m := &object.Module{Name: "xml.sax.xmlreader", Dict: object.NewDict()}

	// SAX exceptions are also exported from xmlreader
	saxExcCls := &object.Class{Name: "SAXException", Dict: object.NewDict(), Bases: []*object.Class{i.exception}}
	saxNotRecCls := &object.Class{Name: "SAXNotRecognizedException", Dict: object.NewDict(), Bases: []*object.Class{saxExcCls}}
	saxNotSupCls := &object.Class{Name: "SAXNotSupportedException", Dict: object.NewDict(), Bases: []*object.Class{saxExcCls}}
	m.Dict.SetStr("SAXNotRecognizedException", saxNotRecCls)
	m.Dict.SetStr("SAXNotSupportedException", saxNotSupCls)

	// xmlReaderHandlerInit initialises the four default handler slots on self.
	xmlReaderHandlerInit := func(interp any, self *object.Instance) {
		ii := interp.(*Interp)
		makeInst := func(clsName string) object.Object {
			hm, err := ii.loadModule("xml.sax.handler")
			if err != nil {
				return object.None
			}
			cv, ok := hm.Dict.GetStr(clsName)
			if !ok {
				return object.None
			}
			cls, ok := cv.(*object.Class)
			if !ok {
				return object.None
			}
			return &object.Instance{Class: cls, Dict: object.NewDict()}
		}
		self.Dict.SetStr("_cont_handler", makeInst("ContentHandler"))
		self.Dict.SetStr("_dtd_handler", makeInst("DTDHandler"))
		self.Dict.SetStr("_ent_handler", makeInst("EntityResolver"))
		self.Dict.SetStr("_err_handler", makeInst("ErrorHandler"))
	}

	// Handler get/set helper for XMLReader instance dict.
	xmlReaderHandlerGet := func(key string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: "get" + key[1:], Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			if self, ok := a[0].(*object.Instance); ok {
				if v, ok2 := self.Dict.GetStr(key); ok2 {
					return v, nil
				}
			}
			return object.None, nil
		}}
	}
	xmlReaderHandlerSet := func(key string) *object.BuiltinFunc {
		return &object.BuiltinFunc{Name: "set" + key[1:], Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) >= 2 {
				if self, ok := a[0].(*object.Instance); ok {
					self.Dict.SetStr(key, a[1])
				}
			}
			return object.None, nil
		}}
	}

	// XMLReader
	xmlReaderCls := &object.Class{Name: "XMLReader", Dict: object.NewDict()}
	xmlReaderCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		xmlReaderHandlerInit(interp, self)
		return object.None, nil
	}})
	xmlReaderCls.Dict.SetStr("parse", &object.BuiltinFunc{Name: "parse", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return nil, object.Errorf(i.notImpl, "This method must be implemented!")
	}})
	xmlReaderCls.Dict.SetStr("setLocale", &object.BuiltinFunc{Name: "setLocale", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return nil, object.Errorf(saxNotSupCls, "Locale support not implemented")
	}})
	for _, key := range []string{"Feature", "Property"} {
		k := key
		xmlReaderCls.Dict.SetStr("get"+k, &object.BuiltinFunc{Name: "get" + k, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			name := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					name = s.V
				}
			}
			return nil, object.Errorf(saxNotRecCls, "%s '%s' not recognized", k, name)
		}})
		xmlReaderCls.Dict.SetStr("set"+k, &object.BuiltinFunc{Name: "set" + k, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			name := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					name = s.V
				}
			}
			return nil, object.Errorf(saxNotRecCls, "%s '%s' not recognized", k, name)
		}})
	}
	for _, pair := range [][2]string{
		{"getContentHandler", "_cont_handler"}, {"setContentHandler", "_cont_handler"},
		{"getDTDHandler", "_dtd_handler"}, {"setDTDHandler", "_dtd_handler"},
		{"getEntityResolver", "_ent_handler"}, {"setEntityResolver", "_ent_handler"},
		{"getErrorHandler", "_err_handler"}, {"setErrorHandler", "_err_handler"},
	} {
		name, key := pair[0], pair[1]
		if strings.HasPrefix(name, "get") {
			xmlReaderCls.Dict.SetStr(name, xmlReaderHandlerGet(key))
		} else {
			xmlReaderCls.Dict.SetStr(name, xmlReaderHandlerSet(key))
		}
	}
	m.Dict.SetStr("XMLReader", xmlReaderCls)

	// IncrementalParser(bufsize=65536) – subclass of XMLReader
	incParserCls := &object.Class{Name: "IncrementalParser", Dict: object.NewDict(), Bases: []*object.Class{xmlReaderCls}}
	incParserCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		bufsize := int64(65536)
		if len(a) >= 2 {
			if n, ok := a[1].(*object.Int); ok {
				bufsize = n.Int64()
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("bufsize"); ok {
				if n, ok2 := v.(*object.Int); ok2 {
					bufsize = n.Int64()
				}
			}
		}
		self.Dict.SetStr("_bufsize", object.IntFromInt64(bufsize))
		xmlReaderHandlerInit(interp, self)
		return object.None, nil
	}})
	// Abstract methods — raise NotImplementedError
	for _, pair := range [][2]string{
		{"feed", "This method must be implemented!"},
		{"close", "This method must be implemented!"},
		{"reset", "This method must be implemented!"},
		{"prepareParser", "prepareParser must be overridden!"},
	} {
		name, msg := pair[0], pair[1]
		n, ms := name, msg
		incParserCls.Dict.SetStr(n, &object.BuiltinFunc{Name: n, Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.notImpl, "%s", ms)
		}})
	}
	// Concrete parse() — reads source in bufsize chunks, calls feed()/close()
	incParserCls.Dict.SetStr("parse", &object.BuiltinFunc{Name: "parse", Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		ii := interp.(*Interp)
		self := a[0].(*object.Instance)
		src := a[1]

		// get bufsize
		bufsize := int64(65536)
		if v, ok := self.Dict.GetStr("_bufsize"); ok {
			if n, ok2 := v.(*object.Int); ok2 {
				bufsize = n.Int64()
			}
		}

		// get byteStream or charStream
		var stream object.Object
		for _, attr := range []string{"getCharacterStream", "getByteStream"} {
			fn, err := ii.getAttr(src, attr)
			if err != nil {
				continue
			}
			res, err := ii.callObject(fn, nil, nil)
			if err != nil || res == object.None {
				continue
			}
			stream = res
			break
		}
		if stream == nil {
			return object.None, nil
		}

		// read stream in chunks
		readFn, err := ii.getAttr(stream, "read")
		if err != nil {
			return object.None, nil
		}
		feedFn, err2 := ii.getAttr(self, "feed")
		if err2 != nil {
			return object.None, nil
		}

		for {
			chunk, rerr := ii.callObject(readFn, []object.Object{object.IntFromInt64(bufsize)}, nil)
			if rerr != nil {
				return nil, rerr
			}
			// empty bytes/str means EOF
			empty := false
			switch v := chunk.(type) {
			case *object.Bytes:
				empty = len(v.V) == 0
			case *object.Str:
				empty = len(v.V) == 0
			default:
				empty = chunk == object.None
			}
			if empty {
				break
			}
			if _, ferr := ii.callObject(feedFn, []object.Object{chunk}, nil); ferr != nil {
				return nil, ferr
			}
		}

		closeFn, cerr := ii.getAttr(self, "close")
		if cerr == nil {
			if _, cerr2 := ii.callObject(closeFn, nil, nil); cerr2 != nil {
				return nil, cerr2
			}
		}
		return object.None, nil
	}})
	// Inherit XMLReader handler get/set methods
	for _, pair := range [][2]string{
		{"getContentHandler", "_cont_handler"}, {"setContentHandler", "_cont_handler"},
		{"getDTDHandler", "_dtd_handler"}, {"setDTDHandler", "_dtd_handler"},
		{"getEntityResolver", "_ent_handler"}, {"setEntityResolver", "_ent_handler"},
		{"getErrorHandler", "_err_handler"}, {"setErrorHandler", "_err_handler"},
	} {
		name, key := pair[0], pair[1]
		if strings.HasPrefix(name, "get") {
			incParserCls.Dict.SetStr(name, xmlReaderHandlerGet(key))
		} else {
			incParserCls.Dict.SetStr(name, xmlReaderHandlerSet(key))
		}
	}
	incParserCls.Dict.SetStr("setLocale", &object.BuiltinFunc{Name: "setLocale", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return nil, object.Errorf(saxNotSupCls, "Locale support not implemented")
	}})
	for _, key := range []string{"Feature", "Property"} {
		k := key
		incParserCls.Dict.SetStr("get"+k, &object.BuiltinFunc{Name: "get" + k, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			name := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					name = s.V
				}
			}
			return nil, object.Errorf(saxNotRecCls, "%s '%s' not recognized", k, name)
		}})
		incParserCls.Dict.SetStr("set"+k, &object.BuiltinFunc{Name: "set" + k, Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			name := ""
			if len(a) >= 2 {
				if s, ok := a[1].(*object.Str); ok {
					name = s.V
				}
			}
			return nil, object.Errorf(saxNotRecCls, "%s '%s' not recognized", k, name)
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

	// AttributesImpl(attrs_dict) — full implementation with values()/get()
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
	for _, name := range []string{
		"getNames", "getType", "getValue", "getLength",
		"__len__", "__getitem__", "__contains__", "__iter__",
		"keys", "items", "values", "get", "copy",
		"getQNameByName", "getNameByQName", "getValueByQName", "getQNames",
	} {
		if v, ok := attrsCls.Dict.GetStr(name); ok {
			attrsCls2.Dict.SetStr(name, v)
		}
	}
	m.Dict.SetStr("AttributesImpl", attrsCls2)

	// AttributesNSImpl(attrs, qnames) — NS-aware full implementation
	attrsNSCls := &object.Class{Name: "AttributesNSImpl", Dict: object.NewDict()}

	// helper: extract [2]string key from a Python tuple (ns_uri, local_name)
	nsTupleKey := func(obj object.Object) ([2]string, bool) {
		t, ok := obj.(*object.Tuple)
		if !ok || len(t.V) < 2 {
			return [2]string{}, false
		}
		ns, ok1 := t.V[0].(*object.Str)
		local, ok2 := t.V[1].(*object.Str)
		if !ok1 || !ok2 {
			return [2]string{}, false
		}
		return [2]string{ns.V, local.V}, true
	}

	attrsNSCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := &saxNSAttrsState{}
		if ad, ok := a[1].(*object.Dict); ok {
			ks, vs := ad.Items()
			for idx2, k := range ks {
				key, ok2 := nsTupleKey(k)
				if !ok2 {
					continue
				}
				val := ""
				if s, ok3 := vs[idx2].(*object.Str); ok3 {
					val = s.V
				}
				st.attrKeys = append(st.attrKeys, key)
				st.attrVals = append(st.attrVals, val)
			}
		}
		if qd, ok := a[2].(*object.Dict); ok {
			ks, vs := qd.Items()
			for idx2, k := range ks {
				key, ok2 := nsTupleKey(k)
				if !ok2 {
					continue
				}
				qname := ""
				if s, ok3 := vs[idx2].(*object.Str); ok3 {
					qname = s.V
				}
				st.qnameKeys = append(st.qnameKeys, key)
				st.qnameVals = append(st.qnameVals, qname)
			}
		}
		saxNSAttrsStateMap.Store(self, st)
		return object.None, nil
	}})

	nsLoad := func(a []object.Object) *saxNSAttrsState {
		if len(a) < 1 {
			return nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return nil
		}
		v, _ := saxNSAttrsStateMap.Load(self)
		if v == nil {
			return nil
		}
		return v.(*saxNSAttrsState)
	}

	attrsNSCls.Dict.SetStr("getLength", &object.BuiltinFunc{Name: "getLength", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		st := nsLoad(a)
		if st == nil {
			return object.IntFromInt64(0), nil
		}
		return object.IntFromInt64(int64(len(st.attrKeys))), nil
	}})
	attrsNSCls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		st := nsLoad(a)
		if st == nil {
			return object.IntFromInt64(0), nil
		}
		return object.IntFromInt64(int64(len(st.attrKeys))), nil
	}})
	attrsNSCls.Dict.SetStr("getType", &object.BuiltinFunc{Name: "getType", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "CDATA"}, nil
	}})
	attrsNSCls.Dict.SetStr("getValue", &object.BuiltinFunc{Name: "getValue", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		st := nsLoad(a)
		if st == nil {
			return nil, object.Errorf(i.keyErr, "key not found")
		}
		key, ok := nsTupleKey(a[1])
		if !ok {
			return nil, object.Errorf(i.keyErr, "key not found")
		}
		for idx2, k := range st.attrKeys {
			if k == key {
				return &object.Str{V: st.attrVals[idx2]}, nil
			}
		}
		return nil, object.Errorf(i.keyErr, "key not found")
	}})
	attrsNSCls.Dict.SetStr("getNames", &object.BuiltinFunc{Name: "getNames", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		st := nsLoad(a)
		if st == nil {
			return &object.List{V: nil}, nil
		}
		items := make([]object.Object, len(st.attrKeys))
		for idx2, k := range st.attrKeys {
			items[idx2] = &object.Tuple{V: []object.Object{&object.Str{V: k[0]}, &object.Str{V: k[1]}}}
		}
		return &object.List{V: items}, nil
	}})
	attrsNSCls.Dict.SetStr("getQNames", &object.BuiltinFunc{Name: "getQNames", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		st := nsLoad(a)
		if st == nil {
			return &object.List{V: nil}, nil
		}
		items := make([]object.Object, len(st.qnameVals))
		for idx2, q := range st.qnameVals {
			items[idx2] = &object.Str{V: q}
		}
		return &object.List{V: items}, nil
	}})
	attrsNSCls.Dict.SetStr("getQNameByName", &object.BuiltinFunc{Name: "getQNameByName", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		st := nsLoad(a)
		if st == nil {
			return nil, object.Errorf(i.keyErr, "key not found")
		}
		key, ok := nsTupleKey(a[1])
		if !ok {
			return nil, object.Errorf(i.keyErr, "key not found")
		}
		for idx2, k := range st.qnameKeys {
			if k == key {
				return &object.Str{V: st.qnameVals[idx2]}, nil
			}
		}
		return nil, object.Errorf(i.keyErr, "key not found")
	}})
	attrsNSCls.Dict.SetStr("getNameByQName", &object.BuiltinFunc{Name: "getNameByQName", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		st := nsLoad(a)
		if st == nil {
			return nil, object.Errorf(i.keyErr, "key not found")
		}
		qname := ""
		if s, ok := a[1].(*object.Str); ok {
			qname = s.V
		}
		for idx2, q := range st.qnameVals {
			if q == qname {
				k := st.qnameKeys[idx2]
				return &object.Tuple{V: []object.Object{&object.Str{V: k[0]}, &object.Str{V: k[1]}}}, nil
			}
		}
		return nil, object.Errorf(i.keyErr, "%q", qname)
	}})
	attrsNSCls.Dict.SetStr("getValueByQName", &object.BuiltinFunc{Name: "getValueByQName", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		st := nsLoad(a)
		if st == nil {
			return nil, object.Errorf(i.keyErr, "key not found")
		}
		qname := ""
		if s, ok := a[1].(*object.Str); ok {
			qname = s.V
		}
		for idx2, q := range st.qnameVals {
			if q == qname {
				k := st.qnameKeys[idx2]
				for idx3, ak := range st.attrKeys {
					if ak == k {
						return &object.Str{V: st.attrVals[idx3]}, nil
					}
				}
			}
		}
		return nil, object.Errorf(i.keyErr, "%q", qname)
	}})
	attrsNSCls.Dict.SetStr("keys", &object.BuiltinFunc{Name: "keys", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		st := nsLoad(a)
		if st == nil {
			return &object.List{V: nil}, nil
		}
		items := make([]object.Object, len(st.attrKeys))
		for idx2, k := range st.attrKeys {
			items[idx2] = &object.Tuple{V: []object.Object{&object.Str{V: k[0]}, &object.Str{V: k[1]}}}
		}
		return &object.List{V: items}, nil
	}})
	attrsNSCls.Dict.SetStr("values", &object.BuiltinFunc{Name: "values", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		st := nsLoad(a)
		if st == nil {
			return &object.List{V: nil}, nil
		}
		items := make([]object.Object, len(st.attrVals))
		for idx2, v := range st.attrVals {
			items[idx2] = &object.Str{V: v}
		}
		return &object.List{V: items}, nil
	}})
	attrsNSCls.Dict.SetStr("items", &object.BuiltinFunc{Name: "items", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		st := nsLoad(a)
		if st == nil {
			return &object.List{V: nil}, nil
		}
		items := make([]object.Object, len(st.attrKeys))
		for idx2, k := range st.attrKeys {
			key := &object.Tuple{V: []object.Object{&object.Str{V: k[0]}, &object.Str{V: k[1]}}}
			val := &object.Str{V: st.attrVals[idx2]}
			items[idx2] = &object.Tuple{V: []object.Object{key, val}}
		}
		return &object.List{V: items}, nil
	}})
	attrsNSCls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		st := nsLoad(a)
		if st != nil {
			key, ok := nsTupleKey(a[1])
			if ok {
				for idx2, k := range st.attrKeys {
					if k == key {
						return &object.Str{V: st.attrVals[idx2]}, nil
					}
				}
			}
		}
		if len(a) >= 3 {
			return a[2], nil
		}
		return object.None, nil
	}})
	attrsNSCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		st := nsLoad(a)
		if st != nil {
			key, ok := nsTupleKey(a[1])
			if ok {
				for idx2, k := range st.attrKeys {
					if k == key {
						return &object.Str{V: st.attrVals[idx2]}, nil
					}
				}
			}
		}
		return nil, object.Errorf(i.keyErr, "key not found")
	}})
	attrsNSCls.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		st := nsLoad(a)
		if st == nil {
			return object.False, nil
		}
		key, ok := nsTupleKey(a[1])
		if !ok {
			return object.False, nil
		}
		for _, k := range st.attrKeys {
			if k == key {
				return object.True, nil
			}
		}
		return object.False, nil
	}})
	attrsNSCls.Dict.SetStr("copy", &object.BuiltinFunc{Name: "copy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		st := nsLoad(a)
		newInst := &object.Instance{Class: attrsNSCls, Dict: object.NewDict()}
		if st != nil {
			newSt := &saxNSAttrsState{
				attrKeys:  append([][2]string{}, st.attrKeys...),
				attrVals:  append([]string{}, st.attrVals...),
				qnameKeys: append([][2]string{}, st.qnameKeys...),
				qnameVals: append([]string{}, st.qnameVals...),
			}
			saxNSAttrsStateMap.Store(newInst, newSt)
		}
		return newInst, nil
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

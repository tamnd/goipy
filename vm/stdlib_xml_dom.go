package vm

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"sync"

	"github.com/tamnd/goipy/object"
)

// ── domNode: Go-side state for minidom nodes ──────────────────────────────────

const (
	domELEMENT_NODE                = 1
	domATTRIBUTE_NODE              = 2
	domTEXT_NODE                   = 3
	domCDATA_SECTION_NODE          = 4
	domENTITY_REFERENCE_NODE       = 5
	domENTITY_NODE                 = 6
	domPROCESSING_INSTRUCTION_NODE = 7
	domCOMMENT_NODE                = 8
	domDOCUMENT_NODE               = 9
	domDOCUMENT_TYPE_NODE          = 10
	domDOCUMENT_FRAGMENT_NODE      = 11
	domNOTATION_NODE               = 12
)

type domAttr struct{ name, value string }

type domNodeState struct {
	nodeType  int
	nodeName  string
	nodeValue string
	parent    *object.Instance
	children  []*object.Instance
	attrs     []domAttr
	ownerDoc  *object.Instance
	// for text/comment/pi
	data   string
	target string // for PI
}

type domNodeRegistry struct{ m sync.Map }

func (r *domNodeRegistry) Store(k *object.Instance, v *domNodeState) { r.m.Store(k, v) }
func (r *domNodeRegistry) Load(k *object.Instance) *domNodeState {
	v, ok := r.m.Load(k)
	if !ok {
		return nil
	}
	return v.(*domNodeState)
}

var domNodeMap domNodeRegistry

func getDomNode(inst *object.Instance) *domNodeState {
	return domNodeMap.Load(inst)
}

// ── xml.dom ───────────────────────────────────────────────────────────────────

func (i *Interp) buildXmlDom() *object.Module {
	m := &object.Module{Name: "xml.dom", Dict: object.NewDict()}
	m.Dict.SetStr("__path__", &object.List{V: []object.Object{&object.Str{V: ""}}})
	m.Dict.SetStr("__package__", &object.Str{V: "xml.dom"})

	// Node type constants
	m.Dict.SetStr("ELEMENT_NODE", object.IntFromInt64(1))
	m.Dict.SetStr("ATTRIBUTE_NODE", object.IntFromInt64(2))
	m.Dict.SetStr("TEXT_NODE", object.IntFromInt64(3))
	m.Dict.SetStr("CDATA_SECTION_NODE", object.IntFromInt64(4))
	m.Dict.SetStr("ENTITY_REFERENCE_NODE", object.IntFromInt64(5))
	m.Dict.SetStr("ENTITY_NODE", object.IntFromInt64(6))
	m.Dict.SetStr("PROCESSING_INSTRUCTION_NODE", object.IntFromInt64(7))
	m.Dict.SetStr("COMMENT_NODE", object.IntFromInt64(8))
	m.Dict.SetStr("DOCUMENT_NODE", object.IntFromInt64(9))
	m.Dict.SetStr("DOCUMENT_TYPE_NODE", object.IntFromInt64(10))
	m.Dict.SetStr("DOCUMENT_FRAGMENT_NODE", object.IntFromInt64(11))
	m.Dict.SetStr("NOTATION_NODE", object.IntFromInt64(12))

	// Error code constants
	m.Dict.SetStr("INDEX_SIZE_ERR", object.IntFromInt64(1))
	m.Dict.SetStr("DOMSTRING_SIZE_ERR", object.IntFromInt64(2))
	m.Dict.SetStr("HIERARCHY_REQUEST_ERR", object.IntFromInt64(3))
	m.Dict.SetStr("WRONG_DOCUMENT_ERR", object.IntFromInt64(4))
	m.Dict.SetStr("INVALID_CHARACTER_ERR", object.IntFromInt64(5))
	m.Dict.SetStr("NO_DATA_ALLOWED_ERR", object.IntFromInt64(6))
	m.Dict.SetStr("NO_MODIFICATION_ALLOWED_ERR", object.IntFromInt64(7))
	m.Dict.SetStr("NOT_FOUND_ERR", object.IntFromInt64(8))
	m.Dict.SetStr("NOT_SUPPORTED_ERR", object.IntFromInt64(9))
	m.Dict.SetStr("INUSE_ATTRIBUTE_ERR", object.IntFromInt64(10))
	m.Dict.SetStr("INVALID_STATE_ERR", object.IntFromInt64(11))
	m.Dict.SetStr("SYNTAX_ERR", object.IntFromInt64(12))
	m.Dict.SetStr("INVALID_MODIFICATION_ERR", object.IntFromInt64(13))
	m.Dict.SetStr("NAMESPACE_ERR", object.IntFromInt64(14))
	m.Dict.SetStr("INVALID_ACCESS_ERR", object.IntFromInt64(15))

	// DOMException
	domExcCls := &object.Class{Name: "DOMException", Dict: object.NewDict(), Bases: []*object.Class{i.exception}}
	domExcCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		self.Dict.SetStr("code", object.IntFromInt64(0))
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok {
				self.Dict.SetStr("code", object.IntFromInt64(n))
			}
		}
		return object.None, nil
	}})
	m.Dict.SetStr("DOMException", domExcCls)

	// Abstract Node class
	nodeCls := i.buildDomNodeClass(domExcCls)
	m.Dict.SetStr("Node", nodeCls)

	return m
}

func (i *Interp) buildDomNodeClass(domExcCls *object.Class) *object.Class {
	cls := &object.Class{Name: "Node", Dict: object.NewDict()}
	// Constants on class
	cls.Dict.SetStr("ELEMENT_NODE", object.IntFromInt64(1))
	cls.Dict.SetStr("ATTRIBUTE_NODE", object.IntFromInt64(2))
	cls.Dict.SetStr("TEXT_NODE", object.IntFromInt64(3))
	cls.Dict.SetStr("CDATA_SECTION_NODE", object.IntFromInt64(4))
	cls.Dict.SetStr("ENTITY_REFERENCE_NODE", object.IntFromInt64(5))
	cls.Dict.SetStr("ENTITY_NODE", object.IntFromInt64(6))
	cls.Dict.SetStr("PROCESSING_INSTRUCTION_NODE", object.IntFromInt64(7))
	cls.Dict.SetStr("COMMENT_NODE", object.IntFromInt64(8))
	cls.Dict.SetStr("DOCUMENT_NODE", object.IntFromInt64(9))
	cls.Dict.SetStr("DOCUMENT_TYPE_NODE", object.IntFromInt64(10))
	cls.Dict.SetStr("DOCUMENT_FRAGMENT_NODE", object.IntFromInt64(11))
	cls.Dict.SetStr("NOTATION_NODE", object.IntFromInt64(12))
	return cls
}

// ── xml.dom.minidom ───────────────────────────────────────────────────────────

func (i *Interp) buildXmlDomMinidom() *object.Module {
	m := &object.Module{Name: "xml.dom.minidom", Dict: object.NewDict()}

	docCls, elemCls, textCls, commentCls, piCls, attrCls, cdataCls, docTypeCls := i.buildMinidomClasses()

	// parse(file, parser=None) -> Document
	m.Dict.SetStr("parse", &object.BuiltinFunc{Name: "parse", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "parse() requires at least 1 argument")
		}
		var data []byte
		switch v := a[0].(type) {
		case *object.Str:
			var err error
			data, err = etReadFile(v.V)
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
		case *object.Bytes:
			data = v.V
		case *object.Bytearray:
			data = v.V
		default:
			ii := interp.(*Interp)
			fn, ferr := ii.getAttr(a[0], "read")
			if ferr == nil {
				res, rerr := ii.callObject(fn, nil, nil)
				if rerr != nil {
					return nil, rerr
				}
				data, _ = asBytes(res)
			}
		}
		return minidomParseBytes(data, docCls, elemCls, textCls, commentCls, piCls, attrCls, cdataCls, docTypeCls)
	}})

	// parseString(string, parser=None) -> Document
	m.Dict.SetStr("parseString", &object.BuiltinFunc{Name: "parseString", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "parseString() requires at least 1 argument")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "parseString() requires bytes or str")
		}
		return minidomParseBytes(data, docCls, elemCls, textCls, commentCls, piCls, attrCls, cdataCls, docTypeCls)
	}})

	// Document class
	m.Dict.SetStr("Document", docCls)
	m.Dict.SetStr("Element", elemCls)
	m.Dict.SetStr("Text", textCls)
	m.Dict.SetStr("Comment", commentCls)
	m.Dict.SetStr("ProcessingInstruction", piCls)
	m.Dict.SetStr("Attr", attrCls)
	m.Dict.SetStr("CDATASection", cdataCls)

	return m
}

// buildMinidomClasses creates all the minidom node classes.
func (i *Interp) buildMinidomClasses() (docCls, elemCls, textCls, commentCls, piCls, attrCls, cdataCls, docTypeCls *object.Class) {
	// Node base
	nodeCls := &object.Class{Name: "Node", Dict: object.NewDict()}
	i.installDomNodeMethods(nodeCls)

	// Element
	elemCls = &object.Class{Name: "Element", Dict: object.NewDict(), Bases: []*object.Class{nodeCls}}
	i.installDomNodeMethods(elemCls)
	i.installDomElementMethods(elemCls)

	// Text
	textCls = &object.Class{Name: "Text", Dict: object.NewDict(), Bases: []*object.Class{nodeCls}}
	i.installDomNodeMethods(textCls)

	// Comment
	commentCls = &object.Class{Name: "Comment", Dict: object.NewDict(), Bases: []*object.Class{nodeCls}}
	i.installDomNodeMethods(commentCls)

	// ProcessingInstruction
	piCls = &object.Class{Name: "ProcessingInstruction", Dict: object.NewDict(), Bases: []*object.Class{nodeCls}}
	i.installDomNodeMethods(piCls)

	// Attr
	attrCls = &object.Class{Name: "Attr", Dict: object.NewDict(), Bases: []*object.Class{nodeCls}}
	i.installDomNodeMethods(attrCls)

	// CDATASection
	cdataCls = &object.Class{Name: "CDATASection", Dict: object.NewDict(), Bases: []*object.Class{textCls}}
	i.installDomNodeMethods(cdataCls)

	// DocumentType
	docTypeCls = &object.Class{Name: "DocumentType", Dict: object.NewDict(), Bases: []*object.Class{nodeCls}}
	i.installDomNodeMethods(docTypeCls)

	// Document
	docCls = &object.Class{Name: "Document", Dict: object.NewDict(), Bases: []*object.Class{nodeCls}}
	i.installDomNodeMethods(docCls)
	i.installDomDocumentMethods(docCls, elemCls, textCls, commentCls, piCls, attrCls, cdataCls, docTypeCls)

	return
}

func (i *Interp) installDomNodeMethods(cls *object.Class) {
	// nodeType, nodeName, nodeValue as instance attributes (set on create)

	// childNodes property (list)
	cls.Dict.SetStr("childNodes", &object.List{V: nil})

	// appendChild(newChild)
	cls.Dict.SetStr("appendChild", &object.BuiltinFunc{Name: "appendChild", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		child, ok2 := a[1].(*object.Instance)
		if !ok || !ok2 {
			return object.None, nil
		}
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		childSt := getDomNode(child)
		if childSt != nil {
			childSt.parent = self
		}
		st.children = append(st.children, child)
		child.Dict.SetStr("parentNode", self)
		// Update childNodes list
		self.Dict.SetStr("childNodes", domChildNodesList(st))
		return child, nil
	}})

	// removeChild(oldChild)
	cls.Dict.SetStr("removeChild", &object.BuiltinFunc{Name: "removeChild", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		child, ok2 := a[1].(*object.Instance)
		if !ok || !ok2 {
			return object.None, nil
		}
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		for idx2, c := range st.children {
			if c == child {
				st.children = append(st.children[:idx2], st.children[idx2+1:]...)
				self.Dict.SetStr("childNodes", domChildNodesList(st))
				child.Dict.SetStr("parentNode", object.None)
				return child, nil
			}
		}
		return child, nil
	}})

	// insertBefore(newChild, refChild)
	cls.Dict.SetStr("insertBefore", &object.BuiltinFunc{Name: "insertBefore", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		newChild, ok2 := a[1].(*object.Instance)
		refChild, ok3 := a[2].(*object.Instance)
		if !ok || !ok2 || !ok3 {
			return object.None, nil
		}
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		for idx2, c := range st.children {
			if c == refChild {
				st.children = append(st.children[:idx2], append([]*object.Instance{newChild}, st.children[idx2:]...)...)
				self.Dict.SetStr("childNodes", domChildNodesList(st))
				return newChild, nil
			}
		}
		st.children = append(st.children, newChild)
		self.Dict.SetStr("childNodes", domChildNodesList(st))
		return newChild, nil
	}})

	// replaceChild(newChild, oldChild)
	cls.Dict.SetStr("replaceChild", &object.BuiltinFunc{Name: "replaceChild", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		newChild, ok2 := a[1].(*object.Instance)
		oldChild, ok3 := a[2].(*object.Instance)
		if !ok || !ok2 || !ok3 {
			return object.None, nil
		}
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		for idx2, c := range st.children {
			if c == oldChild {
				st.children[idx2] = newChild
				self.Dict.SetStr("childNodes", domChildNodesList(st))
				return oldChild, nil
			}
		}
		return oldChild, nil
	}})

	// hasChildNodes()
	cls.Dict.SetStr("hasChildNodes", &object.BuiltinFunc{Name: "hasChildNodes", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.False, nil
		}
		self := a[0].(*object.Instance)
		st := getDomNode(self)
		if st == nil {
			return object.False, nil
		}
		return object.BoolOf(len(st.children) > 0), nil
	}})

	// cloneNode(deep)
	cls.Dict.SetStr("cloneNode", &object.BuiltinFunc{Name: "cloneNode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// normalize()
	cls.Dict.SetStr("normalize", &object.BuiltinFunc{Name: "normalize", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// isSupported()
	cls.Dict.SetStr("isSupported", &object.BuiltinFunc{Name: "isSupported", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.False, nil
	}})

	// hasAttributes()
	cls.Dict.SetStr("hasAttributes", &object.BuiltinFunc{Name: "hasAttributes", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.False, nil
		}
		self := a[0].(*object.Instance)
		st := getDomNode(self)
		if st == nil {
			return object.False, nil
		}
		return object.BoolOf(len(st.attrs) > 0), nil
	}})

	// toxml(encoding=None) -> str
	cls.Dict.SetStr("toxml", &object.BuiltinFunc{Name: "toxml", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		self := a[0].(*object.Instance)
		var buf strings.Builder
		domSerialize(&buf, self, false, "", 0)
		return &object.Str{V: buf.String()}, nil
	}})

	// toprettyxml(indent='\t', newl='\n', encoding=None)
	cls.Dict.SetStr("toprettyxml", &object.BuiltinFunc{Name: "toprettyxml", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		self := a[0].(*object.Instance)
		indent := "\t"
		if kw != nil {
			if v, ok := kw.GetStr("indent"); ok {
				if s, ok := v.(*object.Str); ok {
					indent = s.V
				}
			}
		}
		var buf strings.Builder
		domSerialize(&buf, self, true, indent, 0)
		return &object.Str{V: buf.String()}, nil
	}})

	// unlink() - for compatibility
	cls.Dict.SetStr("unlink", &object.BuiltinFunc{Name: "unlink", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
}

func (i *Interp) installDomElementMethods(cls *object.Class) {
	// getAttribute(name) -> str
	cls.Dict.SetStr("getAttribute", &object.BuiltinFunc{Name: "getAttribute", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return &object.Str{V: ""}, nil
		}
		self := a[0].(*object.Instance)
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		st := getDomNode(self)
		if st == nil {
			return &object.Str{V: ""}, nil
		}
		for _, attr := range st.attrs {
			if attr.name == name {
				return &object.Str{V: attr.value}, nil
			}
		}
		return &object.Str{V: ""}, nil
	}})

	// setAttribute(name, value)
	cls.Dict.SetStr("setAttribute", &object.BuiltinFunc{Name: "setAttribute", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		name := ""
		val := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		if s, ok := a[2].(*object.Str); ok {
			val = s.V
		}
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		for idx2 := range st.attrs {
			if st.attrs[idx2].name == name {
				st.attrs[idx2].value = val
				return object.None, nil
			}
		}
		st.attrs = append(st.attrs, domAttr{name, val})
		return object.None, nil
	}})

	// removeAttribute(name)
	cls.Dict.SetStr("removeAttribute", &object.BuiltinFunc{Name: "removeAttribute", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		for idx2, attr := range st.attrs {
			if attr.name == name {
				st.attrs = append(st.attrs[:idx2], st.attrs[idx2+1:]...)
				return object.None, nil
			}
		}
		return object.None, nil
	}})

	// hasAttribute(name) -> bool
	cls.Dict.SetStr("hasAttribute", &object.BuiltinFunc{Name: "hasAttribute", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		self := a[0].(*object.Instance)
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		st := getDomNode(self)
		if st == nil {
			return object.False, nil
		}
		for _, attr := range st.attrs {
			if attr.name == name {
				return object.True, nil
			}
		}
		return object.False, nil
	}})

	// getElementsByTagName(name) -> list
	cls.Dict.SetStr("getElementsByTagName", &object.BuiltinFunc{Name: "getElementsByTagName", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		var results []*object.Instance
		domGetByTagName(self, name, &results)
		items := make([]object.Object, len(results))
		for idx2, r := range results {
			items[idx2] = r
		}
		return &object.List{V: items}, nil
	}})
}

func (i *Interp) installDomDocumentMethods(docCls, elemCls, textCls, commentCls, piCls, attrCls, cdataCls, docTypeCls *object.Class) {
	// createElement(tagName) -> Element
	docCls.Dict.SetStr("createElement", &object.BuiltinFunc{Name: "createElement", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "createElement() requires 1 argument")
		}
		doc := a[0].(*object.Instance)
		tagName := ""
		if s, ok := a[1].(*object.Str); ok {
			tagName = s.V
		}
		inst := &object.Instance{Class: elemCls, Dict: object.NewDict()}
		st := &domNodeState{
			nodeType: domELEMENT_NODE,
			nodeName: tagName,
			ownerDoc: doc,
		}
		domNodeMap.Store(inst, st)
		domSyncNodeDict(inst, st)
		return inst, nil
	}})

	// createTextNode(data) -> Text
	docCls.Dict.SetStr("createTextNode", &object.BuiltinFunc{Name: "createTextNode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "createTextNode() requires 1 argument")
		}
		doc := a[0].(*object.Instance)
		data := ""
		if s, ok := a[1].(*object.Str); ok {
			data = s.V
		}
		inst := &object.Instance{Class: textCls, Dict: object.NewDict()}
		st := &domNodeState{
			nodeType:  domTEXT_NODE,
			nodeName:  "#text",
			nodeValue: data,
			data:      data,
			ownerDoc:  doc,
		}
		domNodeMap.Store(inst, st)
		domSyncNodeDict(inst, st)
		return inst, nil
	}})

	// createComment(data) -> Comment
	docCls.Dict.SetStr("createComment", &object.BuiltinFunc{Name: "createComment", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "createComment() requires 1 argument")
		}
		doc := a[0].(*object.Instance)
		data := ""
		if s, ok := a[1].(*object.Str); ok {
			data = s.V
		}
		inst := &object.Instance{Class: commentCls, Dict: object.NewDict()}
		st := &domNodeState{
			nodeType:  domCOMMENT_NODE,
			nodeName:  "#comment",
			nodeValue: data,
			data:      data,
			ownerDoc:  doc,
		}
		domNodeMap.Store(inst, st)
		domSyncNodeDict(inst, st)
		return inst, nil
	}})

	// createProcessingInstruction(target, data) -> PI
	docCls.Dict.SetStr("createProcessingInstruction", &object.BuiltinFunc{Name: "createProcessingInstruction", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		doc := a[0].(*object.Instance)
		target := ""
		data := ""
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				target = s.V
			}
		}
		if len(a) >= 3 {
			if s, ok := a[2].(*object.Str); ok {
				data = s.V
			}
		}
		inst := &object.Instance{Class: piCls, Dict: object.NewDict()}
		st := &domNodeState{
			nodeType: domPROCESSING_INSTRUCTION_NODE,
			nodeName: target,
			data:     data,
			target:   target,
			ownerDoc: doc,
		}
		domNodeMap.Store(inst, st)
		domSyncNodeDict(inst, st)
		return inst, nil
	}})

	// createAttribute(name) -> Attr
	docCls.Dict.SetStr("createAttribute", &object.BuiltinFunc{Name: "createAttribute", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		doc := a[0].(*object.Instance)
		name := ""
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				name = s.V
			}
		}
		inst := &object.Instance{Class: attrCls, Dict: object.NewDict()}
		st := &domNodeState{
			nodeType: domATTRIBUTE_NODE,
			nodeName: name,
			ownerDoc: doc,
		}
		domNodeMap.Store(inst, st)
		domSyncNodeDict(inst, st)
		return inst, nil
	}})

	// getElementsByTagName(name) -> list
	docCls.Dict.SetStr("getElementsByTagName", &object.BuiltinFunc{Name: "getElementsByTagName", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		// Get root element
		rootObj, ok := self.Dict.GetStr("documentElement")
		if !ok || rootObj == object.None {
			return &object.List{V: nil}, nil
		}
		root, ok := rootObj.(*object.Instance)
		if !ok {
			return &object.List{V: nil}, nil
		}
		var results []*object.Instance
		domGetByTagName(root, name, &results)
		items := make([]object.Object, len(results))
		for idx2, r := range results {
			items[idx2] = r
		}
		return &object.List{V: items}, nil
	}})

	// getElementById(id) -> Element or None
	docCls.Dict.SetStr("getElementById", &object.BuiltinFunc{Name: "getElementById", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		id := ""
		if s, ok := a[1].(*object.Str); ok {
			id = s.V
		}
		rootObj, ok := self.Dict.GetStr("documentElement")
		if !ok || rootObj == object.None {
			return object.None, nil
		}
		root, ok := rootObj.(*object.Instance)
		if !ok {
			return object.None, nil
		}
		result := domGetByID(root, id)
		if result == nil {
			return object.None, nil
		}
		return result, nil
	}})

	// toxml(encoding=None) -> str
	docCls.Dict.SetStr("toxml", &object.BuiltinFunc{Name: "toxml", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		self := a[0].(*object.Instance)
		var buf strings.Builder
		domSerialize(&buf, self, false, "", 0)
		return &object.Str{V: buf.String()}, nil
	}})

	// toprettyxml
	docCls.Dict.SetStr("toprettyxml", &object.BuiltinFunc{Name: "toprettyxml", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		self := a[0].(*object.Instance)
		indent := "\t"
		if kw != nil {
			if v, ok := kw.GetStr("indent"); ok {
				if s, ok := v.(*object.Str); ok {
					indent = s.V
				}
			}
		}
		var buf strings.Builder
		domSerialize(&buf, self, true, indent, 0)
		return &object.Str{V: buf.String()}, nil
	}})

	// appendChild for Document (adds to top-level)
	docCls.Dict.SetStr("appendChild", &object.BuiltinFunc{Name: "appendChild", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		child, ok := a[1].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		st := getDomNode(self)
		if st != nil {
			st.children = append(st.children, child)
		}
		child.Dict.SetStr("parentNode", self)
		// Set documentElement if it's an element
		childSt := getDomNode(child)
		if childSt != nil && childSt.nodeType == domELEMENT_NODE {
			self.Dict.SetStr("documentElement", child)
		}
		return child, nil
	}})
}

// domSyncNodeDict copies Go state to inst.Dict.
func domSyncNodeDict(inst *object.Instance, st *domNodeState) {
	inst.Dict.SetStr("nodeType", object.IntFromInt64(int64(st.nodeType)))
	inst.Dict.SetStr("nodeName", &object.Str{V: st.nodeName})
	if st.nodeValue != "" {
		inst.Dict.SetStr("nodeValue", &object.Str{V: st.nodeValue})
	} else {
		inst.Dict.SetStr("nodeValue", object.None)
	}
	if st.nodeType == domELEMENT_NODE {
		inst.Dict.SetStr("tagName", &object.Str{V: st.nodeName})
	}
	if st.data != "" {
		inst.Dict.SetStr("data", &object.Str{V: st.data})
	}
	if st.target != "" {
		inst.Dict.SetStr("target", &object.Str{V: st.target})
	}
	inst.Dict.SetStr("parentNode", object.None)
	inst.Dict.SetStr("childNodes", &object.List{V: nil})
	inst.Dict.SetStr("firstChild", object.None)
	inst.Dict.SetStr("lastChild", object.None)
	inst.Dict.SetStr("previousSibling", object.None)
	inst.Dict.SetStr("nextSibling", object.None)
	if st.ownerDoc != nil {
		inst.Dict.SetStr("ownerDocument", st.ownerDoc)
	} else {
		inst.Dict.SetStr("ownerDocument", object.None)
	}
	inst.Dict.SetStr("namespaceURI", object.None)
	inst.Dict.SetStr("prefix", object.None)
	if st.nodeType == domELEMENT_NODE {
		inst.Dict.SetStr("localName", &object.Str{V: st.nodeName})
	}
	// attributes as NamedNodeMap (simple dict)
	if st.nodeType == domELEMENT_NODE {
		attrMap := object.NewDict()
		for _, a := range st.attrs {
			attrMap.SetStr(a.name, &object.Str{V: a.value})
		}
		inst.Dict.SetStr("attributes", attrMap)
	}
}

func domChildNodesList(st *domNodeState) *object.List {
	items := make([]object.Object, len(st.children))
	for idx2, c := range st.children {
		items[idx2] = c
	}
	return &object.List{V: items}
}

// domSerialize serializes DOM node to string builder.
func domSerialize(buf *strings.Builder, inst *object.Instance, pretty bool, indent string, level int) {
	st := getDomNode(inst)
	if st == nil {
		// Check if it's a Document by looking for documentElement
		if docElem, ok := inst.Dict.GetStr("documentElement"); ok && docElem != object.None {
			if docInst, ok := docElem.(*object.Instance); ok {
				// It's a Document node
				docSt := getDomNode(inst)
				if docSt == nil {
					// Write preamble
					if !pretty {
						buf.WriteString(`<?xml version="1.0" ?>`)
					} else {
						buf.WriteString(`<?xml version="1.0" ?>` + "\n")
					}
					domSerialize(buf, docInst, pretty, indent, level)
					return
				}
			}
		}
		return
	}

	pfx := ""
	if pretty {
		pfx = strings.Repeat(indent, level)
	}

	switch st.nodeType {
	case domDOCUMENT_NODE:
		if !pretty {
			buf.WriteString(`<?xml version="1.0" ?>`)
		} else {
			buf.WriteString(`<?xml version="1.0" ?>` + "\n")
		}
		for _, child := range st.children {
			domSerialize(buf, child, pretty, indent, level)
		}
	case domELEMENT_NODE:
		if pretty {
			buf.WriteString(pfx)
		}
		buf.WriteByte('<')
		buf.WriteString(st.nodeName)
		for _, attr := range st.attrs {
			buf.WriteByte(' ')
			buf.WriteString(attr.name)
			buf.WriteString(`="`)
			buf.WriteString(xmlEscapeAttr(attr.value))
			buf.WriteByte('"')
		}
		if len(st.children) == 0 {
			buf.WriteString("/>")
		} else {
			buf.WriteByte('>')
			if pretty {
				buf.WriteByte('\n')
			}
			for _, child := range st.children {
				domSerialize(buf, child, pretty, indent, level+1)
			}
			if pretty {
				buf.WriteString(pfx)
			}
			buf.WriteString("</")
			buf.WriteString(st.nodeName)
			buf.WriteByte('>')
		}
		if pretty {
			buf.WriteByte('\n')
		}
	case domTEXT_NODE:
		if pretty {
			buf.WriteString(pfx)
		}
		buf.WriteString(xmlEscapeText(st.data))
		if pretty {
			buf.WriteByte('\n')
		}
	case domCDATA_SECTION_NODE:
		if pretty {
			buf.WriteString(pfx)
		}
		buf.WriteString("<![CDATA[")
		buf.WriteString(st.data)
		buf.WriteString("]]>")
		if pretty {
			buf.WriteByte('\n')
		}
	case domCOMMENT_NODE:
		if pretty {
			buf.WriteString(pfx)
		}
		buf.WriteString("<!--")
		buf.WriteString(st.data)
		buf.WriteString("-->")
		if pretty {
			buf.WriteByte('\n')
		}
	case domPROCESSING_INSTRUCTION_NODE:
		if pretty {
			buf.WriteString(pfx)
		}
		buf.WriteString("<?")
		buf.WriteString(st.target)
		if st.data != "" {
			buf.WriteByte(' ')
			buf.WriteString(st.data)
		}
		buf.WriteString("?>")
		if pretty {
			buf.WriteByte('\n')
		}
	}
}

func domGetByTagName(inst *object.Instance, name string, results *[]*object.Instance) {
	st := getDomNode(inst)
	if st == nil {
		return
	}
	for _, child := range st.children {
		childSt := getDomNode(child)
		if childSt == nil {
			continue
		}
		if childSt.nodeType == domELEMENT_NODE && (name == "*" || childSt.nodeName == name) {
			*results = append(*results, child)
		}
		domGetByTagName(child, name, results)
	}
}

func domGetByID(inst *object.Instance, id string) *object.Instance {
	st := getDomNode(inst)
	if st == nil {
		return nil
	}
	if st.nodeType == domELEMENT_NODE {
		for _, attr := range st.attrs {
			if attr.name == "id" && attr.value == id {
				return inst
			}
		}
	}
	for _, child := range st.children {
		if result := domGetByID(child, id); result != nil {
			return result
		}
	}
	return nil
}

// minidomParseBytes parses XML bytes into a minidom Document.
func minidomParseBytes(data []byte, docCls, elemCls, textCls, commentCls, piCls, attrCls, cdataCls, docTypeCls *object.Class) (object.Object, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false

	docInst := &object.Instance{Class: docCls, Dict: object.NewDict()}
	docSt := &domNodeState{
		nodeType: domDOCUMENT_NODE,
		nodeName: "#document",
	}
	domNodeMap.Store(docInst, docSt)
	domSyncNodeDict(docInst, docSt)

	type stackEntry struct {
		inst *object.Instance
		st   *domNodeState
	}
	var stack []stackEntry
	stack = append(stack, stackEntry{docInst, docSt})
	var docRoot *object.Instance

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, object.Errorf(nil, "minidom parse error: %v", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			parent := stack[len(stack)-1]
			inst := &object.Instance{Class: elemCls, Dict: object.NewDict()}
			st := &domNodeState{
				nodeType: domELEMENT_NODE,
				nodeName: t.Name.Local,
				ownerDoc: docInst,
			}
			for _, attr := range t.Attr {
				st.attrs = append(st.attrs, domAttr{attr.Name.Local, attr.Value})
			}
			domNodeMap.Store(inst, st)
			domSyncNodeDict(inst, st)
			inst.Dict.SetStr("parentNode", parent.inst)
			parent.st.children = append(parent.st.children, inst)
			parent.inst.Dict.SetStr("childNodes", domChildNodesList(parent.st))
			if len(stack) == 1 {
				docRoot = inst
				docInst.Dict.SetStr("documentElement", inst)
			}
			stack = append(stack, stackEntry{inst, st})
		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			text := string(t)
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				inst := &object.Instance{Class: textCls, Dict: object.NewDict()}
				st := &domNodeState{
					nodeType:  domTEXT_NODE,
					nodeName:  "#text",
					nodeValue: text,
					data:      text,
					ownerDoc:  docInst,
				}
				domNodeMap.Store(inst, st)
				domSyncNodeDict(inst, st)
				inst.Dict.SetStr("parentNode", parent.inst)
				parent.st.children = append(parent.st.children, inst)
				parent.inst.Dict.SetStr("childNodes", domChildNodesList(parent.st))
			}
		case xml.Comment:
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				inst := &object.Instance{Class: commentCls, Dict: object.NewDict()}
				st := &domNodeState{
					nodeType: domCOMMENT_NODE,
					nodeName: "#comment",
					data:     string(t),
					ownerDoc: docInst,
				}
				domNodeMap.Store(inst, st)
				domSyncNodeDict(inst, st)
				parent.st.children = append(parent.st.children, inst)
				parent.inst.Dict.SetStr("childNodes", domChildNodesList(parent.st))
			}
		case xml.ProcInst:
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				inst := &object.Instance{Class: piCls, Dict: object.NewDict()}
				st := &domNodeState{
					nodeType: domPROCESSING_INSTRUCTION_NODE,
					nodeName: t.Target,
					target:   t.Target,
					data:     string(t.Inst),
					ownerDoc: docInst,
				}
				domNodeMap.Store(inst, st)
				domSyncNodeDict(inst, st)
				parent.st.children = append(parent.st.children, inst)
				parent.inst.Dict.SetStr("childNodes", domChildNodesList(parent.st))
			}
		}
	}
	_ = docRoot
	return docInst, nil
}

// ── xml.dom.pulldom ───────────────────────────────────────────────────────────

func (i *Interp) buildXmlDomPulldom() *object.Module {
	m := &object.Module{Name: "xml.dom.pulldom", Dict: object.NewDict()}

	// Event constants
	m.Dict.SetStr("START_ELEMENT", &object.Str{V: "START_ELEMENT"})
	m.Dict.SetStr("END_ELEMENT", &object.Str{V: "END_ELEMENT"})
	m.Dict.SetStr("COMMENT", &object.Str{V: "COMMENT"})
	m.Dict.SetStr("START_DOCUMENT", &object.Str{V: "START_DOCUMENT"})
	m.Dict.SetStr("END_DOCUMENT", &object.Str{V: "END_DOCUMENT"})
	m.Dict.SetStr("CHARACTERS", &object.Str{V: "CHARACTERS"})
	m.Dict.SetStr("PROCESSING_INSTRUCTION", &object.Str{V: "PROCESSING_INSTRUCTION"})
	m.Dict.SetStr("IGNORABLE_WHITESPACE", &object.Str{V: "IGNORABLE_WHITESPACE"})

	// DOMEventStream class
	domesCls := &object.Class{Name: "DOMEventStream", Dict: object.NewDict()}
	domesCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			a[0].(*object.Instance).Dict.SetStr("_events", a[1])
		}
		return object.None, nil
	}})
	domesCls.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		if v, ok := self.Dict.GetStr("_events"); ok {
			return v, nil
		}
		return &object.List{V: nil}, nil
	}})
	domesCls.Dict.SetStr("expandNode", &object.BuiltinFunc{Name: "expandNode", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})
	m.Dict.SetStr("DOMEventStream", domesCls)

	// SAX2DOM class (stub)
	sax2domCls := &object.Class{Name: "SAX2DOM", Dict: object.NewDict()}
	m.Dict.SetStr("SAX2DOM", sax2domCls)

	// parse(stream, parser=None, bufsize=None) -> DOMEventStream
	m.Dict.SetStr("parse", &object.BuiltinFunc{Name: "parse", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "parse() requires at least 1 argument")
		}
		var data []byte
		switch v := a[0].(type) {
		case *object.Str:
			var err error
			data, err = etReadFile(v.V)
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
		case *object.Bytes:
			data = v.V
		case *object.Bytearray:
			data = v.V
		default:
			ii := interp.(*Interp)
			fn, ferr := ii.getAttr(a[0], "read")
			if ferr == nil {
				res, rerr := ii.callObject(fn, nil, nil)
				if rerr != nil {
					return nil, rerr
				}
				data, _ = asBytes(res)
			}
		}
		events, err := pulldomParse(data)
		if err != nil {
			return nil, err
		}
		inst := &object.Instance{Class: domesCls, Dict: object.NewDict()}
		inst.Dict.SetStr("_events", &object.List{V: events})
		return inst, nil
	}})

	// parseString(string, parser=None) -> DOMEventStream
	m.Dict.SetStr("parseString", &object.BuiltinFunc{Name: "parseString", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "parseString() requires at least 1 argument")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "parseString() requires bytes or str")
		}
		events, perr := pulldomParse(data)
		if perr != nil {
			return nil, perr
		}
		inst := &object.Instance{Class: domesCls, Dict: object.NewDict()}
		inst.Dict.SetStr("_events", &object.List{V: events})
		return inst, nil
	}})

	return m
}

// pulldomParse parses XML bytes and returns a list of (event, node) tuples.
func pulldomParse(data []byte) ([]object.Object, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false

	var events []object.Object

	// minimal node classes for pulldom
	elemCls := &object.Class{Name: "Element", Dict: object.NewDict()}
	textCls := &object.Class{Name: "Text", Dict: object.NewDict()}
	commentCls := &object.Class{Name: "Comment", Dict: object.NewDict()}
	piCls := &object.Class{Name: "ProcessingInstruction", Dict: object.NewDict()}

	// START_DOCUMENT
	docNode := object.None
	events = append(events, &object.Tuple{V: []object.Object{&object.Str{V: "START_DOCUMENT"}, docNode}})

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, object.Errorf(nil, "pulldom parse error: %v", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			inst := &object.Instance{Class: elemCls, Dict: object.NewDict()}
			inst.Dict.SetStr("tagName", &object.Str{V: t.Name.Local})
			inst.Dict.SetStr("nodeName", &object.Str{V: t.Name.Local})
			inst.Dict.SetStr("nodeType", object.IntFromInt64(domELEMENT_NODE))
			attrDict := object.NewDict()
			for _, attr := range t.Attr {
				attrDict.SetStr(attr.Name.Local, &object.Str{V: attr.Value})
			}
			inst.Dict.SetStr("attributes", attrDict)
			events = append(events, &object.Tuple{V: []object.Object{
				&object.Str{V: "START_ELEMENT"}, inst,
			}})
		case xml.EndElement:
			inst := &object.Instance{Class: elemCls, Dict: object.NewDict()}
			inst.Dict.SetStr("tagName", &object.Str{V: t.Name.Local})
			inst.Dict.SetStr("nodeName", &object.Str{V: t.Name.Local})
			inst.Dict.SetStr("nodeType", object.IntFromInt64(domELEMENT_NODE))
			events = append(events, &object.Tuple{V: []object.Object{
				&object.Str{V: "END_ELEMENT"}, inst,
			}})
		case xml.CharData:
			text := string(t)
			inst := &object.Instance{Class: textCls, Dict: object.NewDict()}
			inst.Dict.SetStr("data", &object.Str{V: text})
			inst.Dict.SetStr("nodeType", object.IntFromInt64(domTEXT_NODE))
			if strings.TrimSpace(text) == "" {
				events = append(events, &object.Tuple{V: []object.Object{
					&object.Str{V: "IGNORABLE_WHITESPACE"}, inst,
				}})
			} else {
				events = append(events, &object.Tuple{V: []object.Object{
					&object.Str{V: "CHARACTERS"}, inst,
				}})
			}
		case xml.Comment:
			inst := &object.Instance{Class: commentCls, Dict: object.NewDict()}
			inst.Dict.SetStr("data", &object.Str{V: string(t)})
			inst.Dict.SetStr("nodeType", object.IntFromInt64(domCOMMENT_NODE))
			events = append(events, &object.Tuple{V: []object.Object{
				&object.Str{V: "COMMENT"}, inst,
			}})
		case xml.ProcInst:
			inst := &object.Instance{Class: piCls, Dict: object.NewDict()}
			inst.Dict.SetStr("target", &object.Str{V: t.Target})
			inst.Dict.SetStr("data", &object.Str{V: string(t.Inst)})
			inst.Dict.SetStr("nodeType", object.IntFromInt64(domPROCESSING_INSTRUCTION_NODE))
			events = append(events, &object.Tuple{V: []object.Object{
				&object.Str{V: "PROCESSING_INSTRUCTION"}, inst,
			}})
		}
	}

	// END_DOCUMENT
	events = append(events, &object.Tuple{V: []object.Object{&object.Str{V: "END_DOCUMENT"}, docNode}})

	return events, nil
}

// ── xml.parsers ───────────────────────────────────────────────────────────────

func (i *Interp) buildXmlParsers() *object.Module {
	m := &object.Module{Name: "xml.parsers", Dict: object.NewDict()}
	m.Dict.SetStr("__path__", &object.List{V: []object.Object{&object.Str{V: ""}}})
	m.Dict.SetStr("__package__", &object.Str{V: "xml.parsers"})
	return m
}

// ── xml.parsers.expat ─────────────────────────────────────────────────────────

type expatParserState struct {
	handlers map[string]object.Object
}

type expatParserRegistry struct{ m sync.Map }

func (r *expatParserRegistry) Store(k *object.Instance, v *expatParserState) { r.m.Store(k, v) }
func (r *expatParserRegistry) Load(k *object.Instance) *expatParserState {
	v, ok := r.m.Load(k)
	if !ok {
		return nil
	}
	return v.(*expatParserState)
}

var expatParserMap expatParserRegistry

func (i *Interp) buildXmlParsersExpat() *object.Module {
	m := &object.Module{Name: "xml.parsers.expat", Dict: object.NewDict()}

	// ExpatError
	expatErrCls := &object.Class{Name: "ExpatError", Dict: object.NewDict(), Bases: []*object.Class{i.exception}}
	m.Dict.SetStr("ExpatError", expatErrCls)
	m.Dict.SetStr("error", expatErrCls)

	// Constants
	m.Dict.SetStr("EXPAT_VERSION", &object.Str{V: "expat_2.5.0"})
	m.Dict.SetStr("native_encoding", &object.Str{V: "UTF-8"})

	// errors module-like object
	errorsModule := &object.Module{Name: "xml.parsers.expat.errors", Dict: object.NewDict()}
	errorsModule.Dict.SetStr("XML_ERROR_NONE", &object.Str{V: ""})
	errorsModule.Dict.SetStr("XML_ERROR_UNDEFINED_ENTITY", &object.Str{V: "undefined entity"})
	m.Dict.SetStr("errors", errorsModule)

	// ErrorString(errno) -> str
	m.Dict.SetStr("ErrorString", &object.BuiltinFunc{Name: "ErrorString", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Str{V: "error"}, nil
	}})

	// ExpatBuilderNS class stub
	expatBuilderNSCls := &object.Class{Name: "ExpatBuilderNS", Dict: object.NewDict()}
	m.Dict.SetStr("ExpatBuilderNS", expatBuilderNSCls)

	// ExpatParser class
	expatCls := &object.Class{Name: "ExpatParser", Dict: object.NewDict()}

	handlerNames := []string{
		"StartElementHandler", "EndElementHandler", "CharacterDataHandler",
		"ProcessingInstructionHandler", "CommentHandler",
		"StartCdataSectionHandler", "EndCdataSectionHandler",
		"DefaultHandler", "XmlDeclHandler",
		"StartDoctypeDeclHandler", "EndDoctypeDeclHandler",
		"NotationDeclHandler", "UnparsedEntityDeclHandler",
	}

	// __init__
	expatCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := &expatParserState{handlers: make(map[string]object.Object)}
		for _, h := range handlerNames {
			st.handlers[h] = object.None
		}
		expatParserMap.Store(self, st)
		self.Dict.SetStr("returns_unicode", object.True)
		self.Dict.SetStr("ErrorLineNumber", object.IntFromInt64(0))
		self.Dict.SetStr("ErrorColumnNumber", object.IntFromInt64(0))
		self.Dict.SetStr("ErrorByteIndex", object.IntFromInt64(0))
		self.Dict.SetStr("ErrorCode", object.IntFromInt64(0))
		self.Dict.SetStr("CurrentLineNumber", object.IntFromInt64(0))
		self.Dict.SetStr("CurrentColumnNumber", object.IntFromInt64(0))
		self.Dict.SetStr("CurrentByteIndex", object.IntFromInt64(0))
		return object.None, nil
	}})

	// __setattr__ to capture handler assignments
	expatCls.Dict.SetStr("__setattr__", &object.BuiltinFunc{Name: "__setattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		val := a[2]
		self.Dict.SetStr(name, val)
		// Also store in handler map
		st := expatParserMap.Load(self)
		if st != nil {
			st.handlers[name] = val
		}
		return object.None, nil
	}})

	// __getattr__ for handler names (return from dict)
	expatCls.Dict.SetStr("__getattr__", &object.BuiltinFunc{Name: "__getattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
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
		st := expatParserMap.Load(self)
		if st != nil {
			if v, ok := st.handlers[name]; ok {
				return v, nil
			}
		}
		return object.None, nil
	}})

	// Parse(data, isfinal=False)
	expatCls.Dict.SetStr("Parse", &object.BuiltinFunc{Name: "Parse", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.IntFromInt64(1), nil
		}
		self := a[0].(*object.Instance)
		data, _ := asBytes(a[1])
		st := expatParserMap.Load(self)
		if st == nil {
			return object.IntFromInt64(1), nil
		}
		ii := interp.(*Interp)
		return object.IntFromInt64(1), expatParse(ii, self, st, data)
	}})

	// ParseFile(file)
	expatCls.Dict.SetStr("ParseFile", &object.BuiltinFunc{Name: "ParseFile", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.IntFromInt64(1), nil
		}
		self := a[0].(*object.Instance)
		var data []byte
		ii := interp.(*Interp)
		fn, err := ii.getAttr(a[1], "read")
		if err == nil {
			res, rerr := ii.callObject(fn, nil, nil)
			if rerr != nil {
				return nil, rerr
			}
			data, _ = asBytes(res)
		}
		st := expatParserMap.Load(self)
		if st == nil {
			return object.IntFromInt64(1), nil
		}
		return object.IntFromInt64(1), expatParse(ii, self, st, data)
	}})

	// SetBase, GetBase stubs
	expatCls.Dict.SetStr("SetBase", &object.BuiltinFunc{Name: "SetBase", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) >= 2 {
			a[0].(*object.Instance).Dict.SetStr("_base", a[1])
		}
		return object.None, nil
	}})
	expatCls.Dict.SetStr("GetBase", &object.BuiltinFunc{Name: "GetBase", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		if v, ok := a[0].(*object.Instance).Dict.GetStr("_base"); ok {
			return v, nil
		}
		return object.None, nil
	}})

	// GetInputContext stub
	expatCls.Dict.SetStr("GetInputContext", &object.BuiltinFunc{Name: "GetInputContext", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return &object.Bytes{V: nil}, nil
	}})

	// ParserCreate(encoding=None, namespace_separator=None) -> ExpatParser
	m.Dict.SetStr("ParserCreate", &object.BuiltinFunc{Name: "ParserCreate", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		inst := &object.Instance{Class: expatCls, Dict: object.NewDict()}
		st := &expatParserState{handlers: make(map[string]object.Object)}
		for _, h := range handlerNames {
			st.handlers[h] = object.None
		}
		expatParserMap.Store(inst, st)
		inst.Dict.SetStr("returns_unicode", object.True)
		inst.Dict.SetStr("ErrorLineNumber", object.IntFromInt64(0))
		inst.Dict.SetStr("ErrorColumnNumber", object.IntFromInt64(0))
		inst.Dict.SetStr("ErrorByteIndex", object.IntFromInt64(0))
		inst.Dict.SetStr("ErrorCode", object.IntFromInt64(0))
		inst.Dict.SetStr("CurrentLineNumber", object.IntFromInt64(0))
		inst.Dict.SetStr("CurrentColumnNumber", object.IntFromInt64(0))
		inst.Dict.SetStr("CurrentByteIndex", object.IntFromInt64(0))
		return inst, nil
	}})

	return m
}

// expatParse runs the expat-style parsing, calling Go handlers.
func expatParse(ii *Interp, self *object.Instance, st *expatParserState, data []byte) error {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false

	callHandler := func(name string, args []object.Object) error {
		// First check instance dict (for handler set via __setattr__)
		h, ok := self.Dict.GetStr(name)
		if !ok || h == object.None {
			// Fall back to st.handlers
			h = st.handlers[name]
		}
		if h == nil || h == object.None {
			return nil
		}
		_, err := ii.callObject(h, args, nil)
		return err
	}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil // ignore parse errors in expat stub
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := t.Name.Local
			// Build attrs dict
			attrsDict := object.NewDict()
			for _, attr := range t.Attr {
				attrsDict.SetStr(attr.Name.Local, &object.Str{V: attr.Value})
			}
			if err2 := callHandler("StartElementHandler", []object.Object{
				&object.Str{V: name}, attrsDict,
			}); err2 != nil {
				return err2
			}
		case xml.EndElement:
			name := t.Name.Local
			if err2 := callHandler("EndElementHandler", []object.Object{
				&object.Str{V: name},
			}); err2 != nil {
				return err2
			}
		case xml.CharData:
			text := string(t)
			if err2 := callHandler("CharacterDataHandler", []object.Object{
				&object.Str{V: text},
			}); err2 != nil {
				return err2
			}
		case xml.Comment:
			if err2 := callHandler("CommentHandler", []object.Object{
				&object.Str{V: string(t)},
			}); err2 != nil {
				return err2
			}
		case xml.ProcInst:
			if err2 := callHandler("ProcessingInstructionHandler", []object.Object{
				&object.Str{V: t.Target},
				&object.Str{V: string(t.Inst)},
			}); err2 != nil {
				return err2
			}
		}
	}
	return nil
}


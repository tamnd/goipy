package vm

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"sync"

	"github.com/tamnd/goipy/object"
)

// ── shared DOM class singletons ───────────────────────────────────────────────

var (
	domSharedOnce            sync.Once
	domSharedAttrCls         *object.Class
	domSharedNodeListCls     *object.Class
	domSharedNamedNodeMapCls *object.Class
	domSharedImplInst        *object.Instance
)

func ensureDomSharedClasses(i *Interp) {
	domSharedOnce.Do(func() {
		domSharedAttrCls, domSharedNodeListCls, domSharedNamedNodeMapCls = i.buildDomSharedClasses()
	})
}

// nodeListState: dynamic NodeList backed by owner's children list.
type nodeListState struct{ owner *object.Instance }

var nodeListRegistry struct{ m sync.Map }

func getNLState(inst *object.Instance) *nodeListState {
	v, ok := nodeListRegistry.m.Load(inst)
	if !ok {
		return nil
	}
	return v.(*nodeListState)
}

// namedNodeMapState: dynamic NamedNodeMap backed by owner's attrs slice.
type namedNodeMapState struct{ owner *object.Instance }

var namedNodeMapRegistry struct{ m sync.Map }

func getNNMState(inst *object.Instance) *namedNodeMapState {
	v, ok := namedNodeMapRegistry.m.Load(inst)
	if !ok {
		return nil
	}
	return v.(*namedNodeMapState)
}

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
	// for DocumentType
	publicId string
	systemId string
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

// ── buildDomSharedClasses ─────────────────────────────────────────────────────

func (i *Interp) buildDomSharedClasses() (attrCls, nodeListCls, namedNodeMapCls *object.Class) {
	// ── Attr ──────────────────────────────────────────────────────────────────
	attrCls = &object.Class{Name: "Attr", Dict: object.NewDict()}
	attrCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		self.Dict.SetStr("name", &object.Str{V: ""})
		self.Dict.SetStr("nodeName", &object.Str{V: ""})
		self.Dict.SetStr("value", &object.Str{V: ""})
		self.Dict.SetStr("nodeValue", &object.Str{V: ""})
		self.Dict.SetStr("nodeType", object.IntFromInt64(domATTRIBUTE_NODE))
		self.Dict.SetStr("ownerElement", object.None)
		return object.None, nil
	}})

	makeDomAttr := func(name, value string) *object.Instance {
		inst := &object.Instance{Class: attrCls, Dict: object.NewDict()}
		inst.Dict.SetStr("name", &object.Str{V: name})
		inst.Dict.SetStr("nodeName", &object.Str{V: name})
		inst.Dict.SetStr("value", &object.Str{V: value})
		inst.Dict.SetStr("nodeValue", &object.Str{V: value})
		inst.Dict.SetStr("nodeType", object.IntFromInt64(domATTRIBUTE_NODE))
		inst.Dict.SetStr("ownerElement", object.None)
		inst.Dict.SetStr("specified", object.False)
		inst.Dict.SetStr("prefix", object.None)
		inst.Dict.SetStr("namespaceURI", object.None)
		local := name
		if idx := strings.LastIndex(name, ":"); idx >= 0 {
			local = name[idx+1:]
		}
		inst.Dict.SetStr("localName", &object.Str{V: local})
		return inst
	}

	// ── NodeList ──────────────────────────────────────────────────────────────
	nodeListCls = &object.Class{Name: "NodeList", Dict: object.NewDict()}

	getChildren := func(inst *object.Instance) []*object.Instance {
		st := getNLState(inst)
		if st == nil {
			return nil
		}
		ownerSt := getDomNode(st.owner)
		if ownerSt == nil {
			return nil
		}
		return ownerSt.children
	}

	nodeListCls.Dict.SetStr("item", &object.BuiltinFunc{Name: "item", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		ch := getChildren(a[0].(*object.Instance))
		n, ok := toInt64(a[1])
		if !ok {
			return object.None, nil
		}
		idx := int(n)
		if idx < 0 || idx >= len(ch) {
			return object.None, nil
		}
		return ch[idx], nil
	}})

	nodeListCls.Dict.SetStr("length", &object.Property{Fget: &object.BuiltinFunc{Name: "length", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.IntFromInt64(0), nil
		}
		return object.IntFromInt64(int64(len(getChildren(a[0].(*object.Instance))))), nil
	}}})

	nodeListCls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.IntFromInt64(0), nil
		}
		return object.IntFromInt64(int64(len(getChildren(a[0].(*object.Instance))))), nil
	}})

	nodeListCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		ch := getChildren(a[0].(*object.Instance))
		n, ok := toInt64(a[1])
		if !ok {
			return nil, object.Errorf(i.indexErr, "list index out of range")
		}
		idx := int(n)
		if idx < 0 {
			idx += len(ch)
		}
		if idx < 0 || idx >= len(ch) {
			return nil, object.Errorf(i.indexErr, "list index out of range")
		}
		return ch[idx], nil
	}})

	nodeListCls.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		ch := getChildren(a[0].(*object.Instance))
		items := make([]object.Object, len(ch))
		for k, c := range ch {
			items[k] = c
		}
		return &object.List{V: items}, nil
	}})

	// ── NamedNodeMap ──────────────────────────────────────────────────────────
	namedNodeMapCls = &object.Class{Name: "NamedNodeMap", Dict: object.NewDict()}

	getAttrs := func(inst *object.Instance) *domNodeState {
		st := getNNMState(inst)
		if st == nil {
			return nil
		}
		return getDomNode(st.owner)
	}

	namedNodeMapCls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.IntFromInt64(0), nil
		}
		ownerSt := getAttrs(a[0].(*object.Instance))
		if ownerSt == nil {
			return object.IntFromInt64(0), nil
		}
		return object.IntFromInt64(int64(len(ownerSt.attrs))), nil
	}})

	namedNodeMapCls.Dict.SetStr("length", &object.Property{Fget: &object.BuiltinFunc{Name: "length", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.IntFromInt64(0), nil
		}
		ownerSt := getAttrs(a[0].(*object.Instance))
		if ownerSt == nil {
			return object.IntFromInt64(0), nil
		}
		return object.IntFromInt64(int64(len(ownerSt.attrs))), nil
	}}})

	namedNodeMapCls.Dict.SetStr("item", &object.BuiltinFunc{Name: "item", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		ownerSt := getAttrs(a[0].(*object.Instance))
		if ownerSt == nil {
			return object.None, nil
		}
		n, ok := toInt64(a[1])
		if !ok {
			return object.None, nil
		}
		idx := int(n)
		if idx < 0 || idx >= len(ownerSt.attrs) {
			return object.None, nil
		}
		da := ownerSt.attrs[idx]
		return makeDomAttr(da.name, da.value), nil
	}})

	namedNodeMapCls.Dict.SetStr("getNamedItem", &object.BuiltinFunc{Name: "getNamedItem", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		ownerSt := getAttrs(a[0].(*object.Instance))
		if ownerSt == nil {
			return object.None, nil
		}
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		for _, da := range ownerSt.attrs {
			if da.name == name {
				return makeDomAttr(da.name, da.value), nil
			}
		}
		return object.None, nil
	}})

	namedNodeMapCls.Dict.SetStr("setNamedItem", &object.BuiltinFunc{Name: "setNamedItem", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		ownerSt := getAttrs(a[0].(*object.Instance))
		if ownerSt == nil {
			return object.None, nil
		}
		attrInst, ok := a[1].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		name, value := domAttrFromInst(attrInst)
		if name == "" {
			return object.None, nil
		}
		for idx, da := range ownerSt.attrs {
			if da.name == name {
				old := makeDomAttr(da.name, da.value)
				ownerSt.attrs[idx].value = value
				return old, nil
			}
		}
		ownerSt.attrs = append(ownerSt.attrs, domAttr{name, value})
		return object.None, nil
	}})

	namedNodeMapCls.Dict.SetStr("removeNamedItem", &object.BuiltinFunc{Name: "removeNamedItem", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		ownerSt := getAttrs(a[0].(*object.Instance))
		if ownerSt == nil {
			return object.None, nil
		}
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		for idx, da := range ownerSt.attrs {
			if da.name == name {
				old := makeDomAttr(da.name, da.value)
				ownerSt.attrs = append(ownerSt.attrs[:idx], ownerSt.attrs[idx+1:]...)
				return old, nil
			}
		}
		return nil, object.Errorf(i.keyErr, "NOT_FOUND_ERR")
	}})

	namedNodeMapCls.Dict.SetStr("__contains__", &object.BuiltinFunc{Name: "__contains__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		ownerSt := getAttrs(a[0].(*object.Instance))
		if ownerSt == nil {
			return object.False, nil
		}
		name := ""
		if s, ok := a[1].(*object.Str); ok {
			name = s.V
		}
		for _, da := range ownerSt.attrs {
			if da.name == name {
				return object.True, nil
			}
		}
		return object.False, nil
	}})

	namedNodeMapCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		ownerSt := getAttrs(a[0].(*object.Instance))
		if ownerSt == nil {
			return nil, object.Errorf(i.keyErr, "KeyError")
		}
		switch key := a[1].(type) {
		case *object.Str:
			for _, da := range ownerSt.attrs {
				if da.name == key.V {
					return makeDomAttr(da.name, da.value), nil
				}
			}
			return nil, object.Errorf(i.keyErr, "%s", key.V)
		default:
			n, ok := toInt64(a[1])
			if !ok {
				return nil, object.Errorf(i.keyErr, "KeyError")
			}
			idx := int(n)
			if idx < 0 || idx >= len(ownerSt.attrs) {
				return nil, object.Errorf(i.keyErr, "KeyError")
			}
			da := ownerSt.attrs[idx]
			return makeDomAttr(da.name, da.value), nil
		}
	}})

	namedNodeMapCls.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		ownerSt := getAttrs(a[0].(*object.Instance))
		if ownerSt == nil {
			return &object.List{V: nil}, nil
		}
		items := make([]object.Object, len(ownerSt.attrs))
		for k, da := range ownerSt.attrs {
			items[k] = &object.Str{V: da.name}
		}
		return &object.List{V: items}, nil
	}})

	// getNamedItemNS(nsURI, localName) -> Attr or None — ignores nsURI
	namedNodeMapCls.Dict.SetStr("getNamedItemNS", &object.BuiltinFunc{Name: "getNamedItemNS", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		ownerSt := getAttrs(a[0].(*object.Instance))
		if ownerSt == nil {
			return object.None, nil
		}
		name := ""
		if s, ok := a[2].(*object.Str); ok {
			name = s.V
		}
		for _, da := range ownerSt.attrs {
			if da.name == name {
				return makeDomAttr(da.name, da.value), nil
			}
		}
		return object.None, nil
	}})

	// setNamedItemNS(attr) -> old Attr or None
	namedNodeMapCls.Dict.SetStr("setNamedItemNS", &object.BuiltinFunc{Name: "setNamedItemNS", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		ownerSt := getAttrs(a[0].(*object.Instance))
		if ownerSt == nil {
			return object.None, nil
		}
		attrInst, ok := a[1].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		name, value := domAttrFromInst(attrInst)
		if name == "" {
			return object.None, nil
		}
		for idx, da := range ownerSt.attrs {
			if da.name == name {
				old := makeDomAttr(da.name, da.value)
				ownerSt.attrs[idx].value = value
				return old, nil
			}
		}
		ownerSt.attrs = append(ownerSt.attrs, domAttr{name, value})
		return object.None, nil
	}})

	// removeNamedItemNS(nsURI, localName) -> removed Attr — ignores nsURI
	namedNodeMapCls.Dict.SetStr("removeNamedItemNS", &object.BuiltinFunc{Name: "removeNamedItemNS", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		ownerSt := getAttrs(a[0].(*object.Instance))
		if ownerSt == nil {
			return object.None, nil
		}
		name := ""
		if s, ok := a[2].(*object.Str); ok {
			name = s.V
		}
		for idx, da := range ownerSt.attrs {
			if da.name == name {
				old := makeDomAttr(da.name, da.value)
				ownerSt.attrs = append(ownerSt.attrs[:idx], ownerSt.attrs[idx+1:]...)
				return old, nil
			}
		}
		return nil, object.Errorf(i.keyErr, "NOT_FOUND_ERR")
	}})

	return
}

// domAttrFromInst extracts (name, value) from an Attr instance.
// domMakeFullAttr creates an Attr instance with all CPython minidom properties set.
func domMakeFullAttr(name, value string, ownerElement *object.Instance) object.Object {
	if domSharedAttrCls == nil {
		return &object.Str{V: value}
	}
	inst := &object.Instance{Class: domSharedAttrCls, Dict: object.NewDict()}
	inst.Dict.SetStr("name", &object.Str{V: name})
	inst.Dict.SetStr("nodeName", &object.Str{V: name})
	inst.Dict.SetStr("value", &object.Str{V: value})
	inst.Dict.SetStr("nodeValue", &object.Str{V: value})
	inst.Dict.SetStr("nodeType", object.IntFromInt64(domATTRIBUTE_NODE))
	inst.Dict.SetStr("ownerElement", ownerElement)
	inst.Dict.SetStr("specified", object.False)
	inst.Dict.SetStr("prefix", object.None)
	inst.Dict.SetStr("namespaceURI", object.None)
	// localName: strip prefix if present
	local := name
	if idx := strings.LastIndex(name, ":"); idx >= 0 {
		local = name[idx+1:]
	}
	inst.Dict.SetStr("localName", &object.Str{V: local})
	return inst
}

func domAttrFromInst(inst *object.Instance) (name, value string) {
	if v, ok := inst.Dict.GetStr("name"); ok {
		if s, ok2 := v.(*object.Str); ok2 {
			name = s.V
		}
	}
	if name == "" {
		if v, ok := inst.Dict.GetStr("nodeName"); ok {
			if s, ok2 := v.(*object.Str); ok2 {
				name = s.V
			}
		}
	}
	if v, ok := inst.Dict.GetStr("value"); ok {
		if s, ok2 := v.(*object.Str); ok2 {
			value = s.V
		}
	}
	if value == "" {
		if v, ok := inst.Dict.GetStr("nodeValue"); ok {
			if s, ok2 := v.(*object.Str); ok2 {
				value = s.V
			}
		}
	}
	return
}

// domMakeNodeList creates a NodeList pointing to owner.
func domMakeNodeList(owner *object.Instance) object.Object {
	if domSharedNodeListCls == nil {
		return &object.List{V: nil}
	}
	nlInst := &object.Instance{Class: domSharedNodeListCls, Dict: object.NewDict()}
	nodeListRegistry.m.Store(nlInst, &nodeListState{owner: owner})
	return nlInst
}

// domMakeNamedNodeMap creates a NamedNodeMap pointing to owner (element).
func domMakeNamedNodeMap(owner *object.Instance) object.Object {
	if domSharedNamedNodeMapCls == nil {
		return object.NewDict()
	}
	nnmInst := &object.Instance{Class: domSharedNamedNodeMapCls, Dict: object.NewDict()}
	namedNodeMapRegistry.m.Store(nnmInst, &namedNodeMapState{owner: owner})
	return nnmInst
}

// ── xml.dom ───────────────────────────────────────────────────────────────────

func (i *Interp) buildXmlDom() *object.Module {
	ensureDomSharedClasses(i)

	m := &object.Module{Name: "xml.dom", Dict: object.NewDict()}
	m.Dict.SetStr("__path__", &object.List{V: []object.Object{&object.Str{V: ""}}})
	m.Dict.SetStr("__package__", &object.Str{V: "xml.dom"})

	// Namespace constants
	m.Dict.SetStr("EMPTY_NAMESPACE", object.None)
	m.Dict.SetStr("XML_NAMESPACE", &object.Str{V: "http://www.w3.org/XML/1998/namespace"})
	m.Dict.SetStr("XMLNS_NAMESPACE", &object.Str{V: "http://www.w3.org/2000/xmlns/"})
	m.Dict.SetStr("XHTML_NAMESPACE", &object.Str{V: "http://www.w3.org/1999/xhtml"})

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

	// DOMException base
	domExcCls := &object.Class{Name: "DOMException", Dict: object.NewDict(), Bases: []*object.Class{i.exception}}
	domExcCls.Dict.SetStr("code", object.IntFromInt64(0))
	domExcCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		// inherit class-level code if no arg given
		codeVal := object.Object(object.IntFromInt64(0))
		if cv, ok := self.Class.Dict.GetStr("code"); ok {
			codeVal = cv
		}
		if len(a) >= 2 {
			if n, ok := toInt64(a[1]); ok {
				codeVal = object.IntFromInt64(n)
			}
		}
		self.Dict.SetStr("code", codeVal)
		return object.None, nil
	}})
	m.Dict.SetStr("DOMException", domExcCls)

	// DOMException subclasses (one per error code)
	type excDef struct {
		name string
		code int64
	}
	excDefs := []excDef{
		{"IndexSizeErr", 1},
		{"DomstringSizeErr", 2},
		{"HierarchyRequestErr", 3},
		{"WrongDocumentErr", 4},
		{"InvalidCharacterErr", 5},
		{"NoDataAllowedErr", 6},
		{"NoModificationAllowedErr", 7},
		{"NotFoundErr", 8},
		{"NotSupportedErr", 9},
		{"InuseAttributeErr", 10},
		{"InvalidStateErr", 11},
		{"SyntaxErr", 12},
		{"InvalidModificationErr", 13},
		{"NamespaceErr", 14},
		{"InvalidAccessErr", 15},
	}
	for _, ed := range excDefs {
		sub := &object.Class{Name: ed.name, Dict: object.NewDict(), Bases: []*object.Class{domExcCls}}
		sub.Dict.SetStr("code", object.IntFromInt64(ed.code))
		m.Dict.SetStr(ed.name, sub)
	}

	// Shared interface classes
	m.Dict.SetStr("Attr", domSharedAttrCls)
	m.Dict.SetStr("NodeList", domSharedNodeListCls)
	m.Dict.SetStr("NamedNodeMap", domSharedNamedNodeMapCls)

	// Abstract Node class
	nodeCls := i.buildDomNodeClass(domExcCls)
	m.Dict.SetStr("Node", nodeCls)

	// DOMImplementation
	implCls := &object.Class{Name: "DOMImplementation", Dict: object.NewDict()}
	implCls.Dict.SetStr("hasFeature", &object.BuiltinFunc{Name: "hasFeature", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.True, nil
	}})
	implCls.Dict.SetStr("createDocument", &object.BuiltinFunc{Name: "createDocument", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		docCls2, _, _, _, _, _, _, _ := i.buildMinidomClasses()
		docInst := &object.Instance{Class: docCls2, Dict: object.NewDict()}
		docSt := &domNodeState{nodeType: domDOCUMENT_NODE, nodeName: "#document"}
		domNodeMap.Store(docInst, docSt)
		domSyncNodeDict(docInst, docSt)
		return docInst, nil
	}})
	implCls.Dict.SetStr("createDocumentType", &object.BuiltinFunc{Name: "createDocumentType", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		inst := &object.Instance{Class: implCls, Dict: object.NewDict()}
		inst.Dict.SetStr("nodeType", object.IntFromInt64(domDOCUMENT_TYPE_NODE))
		return inst, nil
	}})
	m.Dict.SetStr("DOMImplementation", implCls)

	// Singleton implementation
	implInst := &object.Instance{Class: implCls, Dict: object.NewDict()}
	domSharedImplInst = implInst

	// getDOMImplementation(name=None, features=())
	m.Dict.SetStr("getDOMImplementation", &object.BuiltinFunc{Name: "getDOMImplementation", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return implInst, nil
	}})

	// registerDOMImplementation(name, factory) — no-op
	m.Dict.SetStr("registerDOMImplementation", &object.BuiltinFunc{Name: "registerDOMImplementation", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

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
	ensureDomSharedClasses(i)
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
	i.installCharacterDataMethods(textCls)
	i.installTextMethods(textCls)

	// Comment
	commentCls = &object.Class{Name: "Comment", Dict: object.NewDict(), Bases: []*object.Class{nodeCls}}
	i.installDomNodeMethods(commentCls)
	i.installCharacterDataMethods(commentCls)

	// ProcessingInstruction
	piCls = &object.Class{Name: "ProcessingInstruction", Dict: object.NewDict(), Bases: []*object.Class{nodeCls}}
	i.installDomNodeMethods(piCls)

	// Attr
	attrCls = &object.Class{Name: "Attr", Dict: object.NewDict(), Bases: []*object.Class{nodeCls}}
	i.installDomNodeMethods(attrCls)

	// CDATASection
	cdataCls = &object.Class{Name: "CDATASection", Dict: object.NewDict(), Bases: []*object.Class{textCls}}
	i.installDomNodeMethods(cdataCls)
	i.installCharacterDataMethods(cdataCls)

	// DocumentType
	docTypeCls = &object.Class{Name: "DocumentType", Dict: object.NewDict(), Bases: []*object.Class{nodeCls}}
	i.installDomNodeMethods(docTypeCls)

	// DocumentFragment
	docFragCls := &object.Class{Name: "DocumentFragment", Dict: object.NewDict(), Bases: []*object.Class{nodeCls}}
	i.installDomNodeMethods(docFragCls)

	// Document
	docCls = &object.Class{Name: "Document", Dict: object.NewDict(), Bases: []*object.Class{nodeCls}}
	i.installDomNodeMethods(docCls)
	i.installDomDocumentMethods(docCls, elemCls, textCls, commentCls, piCls, attrCls, cdataCls, docTypeCls, docFragCls)

	return
}

func (i *Interp) installDomNodeMethods(cls *object.Class) {
	// nodeType, nodeName, nodeValue as instance attributes (set on create)

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
				return newChild, nil
			}
		}
		st.children = append(st.children, newChild)
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

	// toxml(encoding=None, standalone=None) -> str
	cls.Dict.SetStr("toxml", &object.BuiltinFunc{Name: "toxml", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		self := a[0].(*object.Instance)
		var buf strings.Builder
		domWriteXML(&buf, self, "", "", "")
		return &object.Str{V: buf.String()}, nil
	}})

	// toprettyxml(indent='\t', newl='\n', encoding=None, standalone=None)
	cls.Dict.SetStr("toprettyxml", &object.BuiltinFunc{Name: "toprettyxml", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		self := a[0].(*object.Instance)
		indent, newl := "\t", "\n"
		if kw != nil {
			if v, ok := kw.GetStr("indent"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					indent = s.V
				}
			}
			if v, ok := kw.GetStr("newl"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					newl = s.V
				}
			}
		}
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				indent = s.V
			}
		}
		if len(a) >= 3 {
			if s, ok := a[2].(*object.Str); ok {
				newl = s.V
			}
		}
		var buf strings.Builder
		domWriteXML(&buf, self, "", indent, newl)
		return &object.Str{V: buf.String()}, nil
	}})

	// writexml(writer, indent='', addindent='', newl='')
	cls.Dict.SetStr("writexml", &object.BuiltinFunc{Name: "writexml", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		writer := a[1]
		indent, addindent, newl := "", "", ""
		if len(a) >= 3 {
			if s, ok := a[2].(*object.Str); ok {
				indent = s.V
			}
		}
		if len(a) >= 4 {
			if s, ok := a[3].(*object.Str); ok {
				addindent = s.V
			}
		}
		if len(a) >= 5 {
			if s, ok := a[4].(*object.Str); ok {
				newl = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("indent"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					indent = s.V
				}
			}
			if v, ok := kw.GetStr("addindent"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					addindent = s.V
				}
			}
			if v, ok := kw.GetStr("newl"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					newl = s.V
				}
			}
		}
		var buf strings.Builder
		domWriteXML(&buf, self, indent, addindent, newl)
		ii := interp.(*Interp)
		writeFn, err := ii.getAttr(writer, "write")
		if err != nil {
			return object.None, nil
		}
		_, werr := ii.callObject(writeFn, []object.Object{&object.Str{V: buf.String()}}, nil)
		return object.None, werr
	}})

	// unlink() - for compatibility
	cls.Dict.SetStr("unlink", &object.BuiltinFunc{Name: "unlink", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// isSameNode(other) -> bool
	cls.Dict.SetStr("isSameNode", &object.BuiltinFunc{Name: "isSameNode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		return object.BoolOf(a[0] == a[1]), nil
	}})

	// isEqualNode(other) -> bool
	cls.Dict.SetStr("isEqualNode", &object.BuiltinFunc{Name: "isEqualNode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.False, nil
		}
		a1, ok1 := a[0].(*object.Instance)
		a2, ok2 := a[1].(*object.Instance)
		if !ok1 || !ok2 {
			return object.BoolOf(a[0] == a[1]), nil
		}
		return object.BoolOf(domNodesEqual(a1, a2)), nil
	}})

	// compareDocumentPosition stub
	cls.Dict.SetStr("compareDocumentPosition", &object.BuiltinFunc{Name: "compareDocumentPosition", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return object.IntFromInt64(0), nil
	}})

	// cloneNode(deep) -> Node
	cls.Dict.SetStr("cloneNode", &object.BuiltinFunc{Name: "cloneNode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		deep := false
		if len(a) >= 2 {
			deep = isTruthy(a[1])
		}
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		cloned := domCloneNode(self, deep, st.ownerDoc)
		if cloned == nil {
			return object.None, nil
		}
		return cloned, nil
	}})

	// normalize() - merge adjacent text nodes
	cls.Dict.SetStr("normalize", &object.BuiltinFunc{Name: "normalize", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		domNormalize(a[0].(*object.Instance))
		return object.None, nil
	}})

	// Dynamic child/sibling properties
	cls.Dict.SetStr("firstChild", &object.Property{Fget: &object.BuiltinFunc{Name: "firstChild", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		st := getDomNode(a[0].(*object.Instance))
		if st == nil || len(st.children) == 0 {
			return object.None, nil
		}
		return st.children[0], nil
	}}})

	cls.Dict.SetStr("lastChild", &object.Property{Fget: &object.BuiltinFunc{Name: "lastChild", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		st := getDomNode(a[0].(*object.Instance))
		if st == nil || len(st.children) == 0 {
			return object.None, nil
		}
		return st.children[len(st.children)-1], nil
	}}})

	cls.Dict.SetStr("previousSibling", &object.Property{Fget: &object.BuiltinFunc{Name: "previousSibling", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getDomNode(self)
		if st == nil || st.parent == nil {
			return object.None, nil
		}
		parentSt := getDomNode(st.parent)
		if parentSt == nil {
			return object.None, nil
		}
		for idx, ch := range parentSt.children {
			if ch == self {
				if idx == 0 {
					return object.None, nil
				}
				return parentSt.children[idx-1], nil
			}
		}
		return object.None, nil
	}}})

	cls.Dict.SetStr("nextSibling", &object.Property{Fget: &object.BuiltinFunc{Name: "nextSibling", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getDomNode(self)
		if st == nil || st.parent == nil {
			return object.None, nil
		}
		parentSt := getDomNode(st.parent)
		if parentSt == nil {
			return object.None, nil
		}
		for idx, ch := range parentSt.children {
			if ch == self {
				if idx == len(parentSt.children)-1 {
					return object.None, nil
				}
				return parentSt.children[idx+1], nil
			}
		}
		return object.None, nil
	}}})
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

	// getElementsByTagNameNS(namespaceURI, localName) — ignores namespace
	cls.Dict.SetStr("getElementsByTagNameNS", &object.BuiltinFunc{Name: "getElementsByTagNameNS", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return &object.List{V: nil}, nil
		}
		// a[1]=nsURI (ignored), a[2]=localName
		self := a[0].(*object.Instance)
		name := ""
		if s, ok := a[2].(*object.Str); ok {
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

	// getAttributeNode(name) -> Attr or None
	cls.Dict.SetStr("getAttributeNode", &object.BuiltinFunc{Name: "getAttributeNode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
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
		for _, da := range st.attrs {
			if da.name == name {
				return domMakeFullAttr(da.name, da.value, self), nil
			}
		}
		return object.None, nil
	}})

	// setAttributeNode(attr) — reads attr.name/value
	cls.Dict.SetStr("setAttributeNode", &object.BuiltinFunc{Name: "setAttributeNode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		attrInst, ok := a[1].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		name, value := domAttrFromInst(attrInst)
		if name == "" {
			return object.None, nil
		}
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		for idx2 := range st.attrs {
			if st.attrs[idx2].name == name {
				st.attrs[idx2].value = value
				return object.None, nil
			}
		}
		st.attrs = append(st.attrs, domAttr{name, value})
		return object.None, nil
	}})

	// removeAttributeNode(attr) — removes by attr.name
	cls.Dict.SetStr("removeAttributeNode", &object.BuiltinFunc{Name: "removeAttributeNode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		attrInst, ok := a[1].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		name, _ := domAttrFromInst(attrInst)
		st := getDomNode(self)
		if st != nil {
			for idx2, da := range st.attrs {
				if da.name == name {
					st.attrs = append(st.attrs[:idx2], st.attrs[idx2+1:]...)
					break
				}
			}
		}
		return attrInst, nil
	}})

	// getAttributeNS(nsURI, localName) — ignores nsURI
	cls.Dict.SetStr("getAttributeNS", &object.BuiltinFunc{Name: "getAttributeNS", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return &object.Str{V: ""}, nil
		}
		// a[1]=nsURI (ignored), a[2]=localName
		self := a[0].(*object.Instance)
		name := ""
		if s, ok := a[2].(*object.Str); ok {
			name = s.V
		}
		st := getDomNode(self)
		if st == nil {
			return &object.Str{V: ""}, nil
		}
		for _, da := range st.attrs {
			if da.name == name {
				return &object.Str{V: da.value}, nil
			}
		}
		return &object.Str{V: ""}, nil
	}})

	// setAttributeNS(nsURI, qname, value) — ignores nsURI
	cls.Dict.SetStr("setAttributeNS", &object.BuiltinFunc{Name: "setAttributeNS", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 4 {
			return object.None, nil
		}
		// a[1]=nsURI, a[2]=qname, a[3]=value
		self := a[0].(*object.Instance)
		name := ""
		val := ""
		if s, ok := a[2].(*object.Str); ok {
			name = s.V
		}
		if s, ok := a[3].(*object.Str); ok {
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

	// removeAttributeNS(nsURI, localName) — ignores nsURI
	cls.Dict.SetStr("removeAttributeNS", &object.BuiltinFunc{Name: "removeAttributeNS", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		name := ""
		if s, ok := a[2].(*object.Str); ok {
			name = s.V
		}
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		for idx2, da := range st.attrs {
			if da.name == name {
				st.attrs = append(st.attrs[:idx2], st.attrs[idx2+1:]...)
				return object.None, nil
			}
		}
		return object.None, nil
	}})

	// hasAttributeNS(nsURI, localName) -> bool — ignores nsURI
	cls.Dict.SetStr("hasAttributeNS", &object.BuiltinFunc{Name: "hasAttributeNS", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.False, nil
		}
		self := a[0].(*object.Instance)
		name := ""
		if s, ok := a[2].(*object.Str); ok {
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

	// getAttributeNodeNS / setAttributeNodeNS — simple aliases
	cls.Dict.SetStr("getAttributeNodeNS", &object.BuiltinFunc{Name: "getAttributeNodeNS", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		// map (nsURI, localName) to getAttributeNode(localName)
		fn, _ := cls.Dict.GetStr("getAttributeNode")
		if fn == nil {
			return object.None, nil
		}
		ii := interp.(*Interp)
		return ii.callObject(fn, []object.Object{a[0], a[2]}, nil)
	}})
	cls.Dict.SetStr("setAttributeNodeNS", &object.BuiltinFunc{Name: "setAttributeNodeNS", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		fn, _ := cls.Dict.GetStr("setAttributeNode")
		if fn == nil {
			return object.None, nil
		}
		ii := interp.(*Interp)
		return ii.callObject(fn, a, nil)
	}})
}

func (i *Interp) installDomDocumentMethods(docCls, elemCls, textCls, commentCls, piCls, attrCls, cdataCls, docTypeCls, docFragCls *object.Class) {
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

	// toxml(encoding=None, standalone=None) -> str or bytes
	docCls.Dict.SetStr("toxml", &object.BuiltinFunc{Name: "toxml", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		self := a[0].(*object.Instance)
		enc, sa := domParseEncodingStandalone(a[1:], kw)
		var buf strings.Builder
		buf.WriteString(xmlMakeDecl(enc, sa))
		domWriteXML(&buf, self, "", "", "")
		if enc != "" {
			return &object.Bytes{V: []byte(buf.String())}, nil
		}
		return &object.Str{V: buf.String()}, nil
	}})

	// toprettyxml(indent='\t', newl='\n', encoding=None, standalone=None)
	docCls.Dict.SetStr("toprettyxml", &object.BuiltinFunc{Name: "toprettyxml", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: ""}, nil
		}
		self := a[0].(*object.Instance)
		indent, newl := "\t", "\n"
		enc, sa := "", object.Object(object.None)
		if kw != nil {
			if v, ok := kw.GetStr("indent"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					indent = s.V
				}
			}
			if v, ok := kw.GetStr("newl"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					newl = s.V
				}
			}
		}
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				indent = s.V
			}
		}
		if len(a) >= 3 {
			if s, ok := a[2].(*object.Str); ok {
				newl = s.V
			}
		}
		var extra2 []object.Object
		if len(a) > 3 {
			extra2 = a[3:]
		}
		enc2, sa2 := domParseEncodingStandalone(extra2, kw)
		if enc2 != "" {
			enc = enc2
		}
		if sa2 != object.None {
			sa = sa2
		}
		var buf strings.Builder
		buf.WriteString(xmlMakeDecl(enc, sa))
		buf.WriteString(newl)
		domWriteXML(&buf, self, "", indent, newl)
		if enc != "" {
			return &object.Bytes{V: []byte(buf.String())}, nil
		}
		return &object.Str{V: buf.String()}, nil
	}})

	// writexml(writer, indent='', addindent='', newl='', encoding=None, standalone=None)
	docCls.Dict.SetStr("writexml", &object.BuiltinFunc{Name: "writexml", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		writer := a[1]
		indent, addindent, newl := "", "", ""
		if len(a) >= 3 {
			if s, ok := a[2].(*object.Str); ok {
				indent = s.V
			}
		}
		if len(a) >= 4 {
			if s, ok := a[3].(*object.Str); ok {
				addindent = s.V
			}
		}
		if len(a) >= 5 {
			if s, ok := a[4].(*object.Str); ok {
				newl = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("indent"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					indent = s.V
				}
			}
			if v, ok := kw.GetStr("addindent"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					addindent = s.V
				}
			}
			if v, ok := kw.GetStr("newl"); ok {
				if s, ok2 := v.(*object.Str); ok2 {
					newl = s.V
				}
			}
		}
		var extra []object.Object
		if len(a) > 5 {
			extra = a[5:]
		}
		enc, sa := domParseEncodingStandalone(extra, kw)
		var buf strings.Builder
		buf.WriteString(xmlMakeDecl(enc, sa))
		buf.WriteString(newl)
		domWriteXML(&buf, self, indent, addindent, newl)
		ii := interp.(*Interp)
		writeFn, err := ii.getAttr(writer, "write")
		if err != nil {
			return object.None, nil
		}
		_, werr := ii.callObject(writeFn, []object.Object{&object.Str{V: buf.String()}}, nil)
		return object.None, werr
	}})

	// __enter__ / __exit__ for context manager
	docCls.Dict.SetStr("__enter__", &object.BuiltinFunc{Name: "__enter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		return a[0], nil
	}})
	docCls.Dict.SetStr("__exit__", &object.BuiltinFunc{Name: "__exit__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		// calls unlink() equivalent: nothing to do in our implementation
		return object.False, nil
	}})

	// doctype property — returns stored doctype or None
	docCls.Dict.SetStr("doctype", &object.Property{Fget: &object.BuiltinFunc{Name: "doctype", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		for _, ch := range st.children {
			chSt := getDomNode(ch)
			if chSt != nil && chSt.nodeType == domDOCUMENT_TYPE_NODE {
				return ch, nil
			}
		}
		return object.None, nil
	}}})

	// implementation — always the DOMImplementation singleton
	docCls.Dict.SetStr("implementation", &object.Property{Fget: &object.BuiltinFunc{Name: "implementation", Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
		return domSharedImplInst, nil
	}}})

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
		childSt := getDomNode(child)
		if childSt != nil && childSt.nodeType == domELEMENT_NODE {
			self.Dict.SetStr("documentElement", child)
		}
		return child, nil
	}})

	// createElementNS(namespaceURI, qualifiedName) -> Element
	docCls.Dict.SetStr("createElementNS", &object.BuiltinFunc{Name: "createElementNS", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "createElementNS() requires 2 arguments")
		}
		doc := a[0].(*object.Instance)
		qname := ""
		if s, ok := a[2].(*object.Str); ok {
			qname = s.V
		}
		inst := &object.Instance{Class: elemCls, Dict: object.NewDict()}
		st := &domNodeState{
			nodeType: domELEMENT_NODE,
			nodeName: qname,
			ownerDoc: doc,
		}
		domNodeMap.Store(inst, st)
		domSyncNodeDict(inst, st)
		return inst, nil
	}})

	// createAttributeNS(namespaceURI, qualifiedName) -> Attr
	docCls.Dict.SetStr("createAttributeNS", &object.BuiltinFunc{Name: "createAttributeNS", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return nil, object.Errorf(i.typeErr, "createAttributeNS() requires 2 arguments")
		}
		qname := ""
		if s, ok := a[2].(*object.Str); ok {
			qname = s.V
		}
		if domSharedAttrCls != nil {
			inst := &object.Instance{Class: domSharedAttrCls, Dict: object.NewDict()}
			inst.Dict.SetStr("name", &object.Str{V: qname})
			inst.Dict.SetStr("nodeName", &object.Str{V: qname})
			inst.Dict.SetStr("value", &object.Str{V: ""})
			inst.Dict.SetStr("nodeValue", &object.Str{V: ""})
			inst.Dict.SetStr("nodeType", object.IntFromInt64(domATTRIBUTE_NODE))
			return inst, nil
		}
		// fallback: use existing attrCls
		doc := a[0].(*object.Instance)
		inst := &object.Instance{Class: attrCls, Dict: object.NewDict()}
		st := &domNodeState{nodeType: domATTRIBUTE_NODE, nodeName: qname, ownerDoc: doc}
		domNodeMap.Store(inst, st)
		domSyncNodeDict(inst, st)
		return inst, nil
	}})

	// getElementsByTagNameNS(nsURI, localName) — ignores nsURI
	docCls.Dict.SetStr("getElementsByTagNameNS", &object.BuiltinFunc{Name: "getElementsByTagNameNS", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		name := ""
		if s, ok := a[2].(*object.Str); ok {
			name = s.V
		}
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

	// createCDATASection(data) -> CDATASection
	docCls.Dict.SetStr("createCDATASection", &object.BuiltinFunc{Name: "createCDATASection", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "createCDATASection() requires 1 argument")
		}
		doc := a[0].(*object.Instance)
		data := ""
		if s, ok := a[1].(*object.Str); ok {
			data = s.V
		}
		inst := &object.Instance{Class: cdataCls, Dict: object.NewDict()}
		st := &domNodeState{
			nodeType:  domCDATA_SECTION_NODE,
			nodeName:  "#cdata-section",
			nodeValue: data,
			data:      data,
			ownerDoc:  doc,
		}
		domNodeMap.Store(inst, st)
		domSyncNodeDict(inst, st)
		return inst, nil
	}})

	// createDocumentFragment() -> DocumentFragment
	docCls.Dict.SetStr("createDocumentFragment", &object.BuiltinFunc{Name: "createDocumentFragment", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		doc := a[0].(*object.Instance)
		inst := &object.Instance{Class: docFragCls, Dict: object.NewDict()}
		st := &domNodeState{
			nodeType: domDOCUMENT_FRAGMENT_NODE,
			nodeName: "#document-fragment",
			ownerDoc: doc,
		}
		domNodeMap.Store(inst, st)
		domSyncNodeDict(inst, st)
		return inst, nil
	}})

	// importNode(node, deep) -> Node
	docCls.Dict.SetStr("importNode", &object.BuiltinFunc{Name: "importNode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "importNode() requires 2 arguments")
		}
		doc := a[0].(*object.Instance)
		src, ok := a[1].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		deep := false
		if len(a) >= 3 {
			deep = isTruthy(a[2])
		}
		cloned := domCloneNode(src, deep, doc)
		if cloned == nil {
			return object.None, nil
		}
		return cloned, nil
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
	inst.Dict.SetStr("childNodes", domMakeNodeList(inst))
	if st.ownerDoc != nil {
		inst.Dict.SetStr("ownerDocument", st.ownerDoc)
	} else {
		inst.Dict.SetStr("ownerDocument", object.None)
	}
	inst.Dict.SetStr("namespaceURI", object.None)
	inst.Dict.SetStr("prefix", object.None)
	// localName: local part of the nodeName (no prefix) for element/attr
	switch st.nodeType {
	case domELEMENT_NODE, domATTRIBUTE_NODE:
		local := st.nodeName
		if idx := strings.LastIndex(local, ":"); idx >= 0 {
			local = local[idx+1:]
		}
		inst.Dict.SetStr("localName", &object.Str{V: local})
	default:
		inst.Dict.SetStr("localName", object.None)
	}
	// attributes as NamedNodeMap for Element
	if st.nodeType == domELEMENT_NODE {
		inst.Dict.SetStr("attributes", domMakeNamedNodeMap(inst))
	}
	// DocumentType fields
	if st.nodeType == domDOCUMENT_TYPE_NODE {
		inst.Dict.SetStr("publicId", &object.Str{V: st.publicId})
		inst.Dict.SetStr("systemId", &object.Str{V: st.systemId})
		inst.Dict.SetStr("internalSubset", object.None)
		inst.Dict.SetStr("entities", object.NewDict())
		inst.Dict.SetStr("notations", object.NewDict())
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
			st.parent = parent.inst
			inst.Dict.SetStr("parentNode", parent.inst)
			parent.st.children = append(parent.st.children, inst)
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
				st.parent = parent.inst
				inst.Dict.SetStr("parentNode", parent.inst)
				parent.st.children = append(parent.st.children, inst)
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
				st.parent = parent.inst
				parent.st.children = append(parent.st.children, inst)
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
				st.parent = parent.inst
				parent.st.children = append(parent.st.children, inst)
			}
		case xml.Directive:
			directive := strings.TrimSpace(string(t))
			if strings.HasPrefix(directive, "DOCTYPE ") {
				rest := strings.TrimSpace(directive[8:])
				// parse: name [PUBLIC "pubId" "sysId" | SYSTEM "sysId"]
				name := ""
				publicId := ""
				systemId := ""
				fields := strings.Fields(rest)
				if len(fields) >= 1 {
					name = fields[0]
				}
				if len(fields) >= 2 {
					keyword := fields[1]
					// rebuild remainder after name to extract quoted ids
					rem := strings.TrimSpace(rest[len(fields[0]):])
					if keyword == "PUBLIC" || keyword == "SYSTEM" {
						quoted := extractDocTypeIDs(rem)
						if keyword == "PUBLIC" && len(quoted) >= 2 {
							publicId = quoted[0]
							systemId = quoted[1]
						} else if keyword == "SYSTEM" && len(quoted) >= 1 {
							systemId = quoted[0]
						}
					}
				}
				inst := &object.Instance{Class: docTypeCls, Dict: object.NewDict()}
				st := &domNodeState{
					nodeType: domDOCUMENT_TYPE_NODE,
					nodeName: name,
					publicId: publicId,
					systemId: systemId,
					ownerDoc: docInst,
				}
				domNodeMap.Store(inst, st)
				domSyncNodeDict(inst, st)
				st.parent = docInst
				docSt.children = append(docSt.children, inst)
			}
		}
	}
	_ = docRoot
	return docInst, nil
}

// extractDocTypeIDs returns up to 2 double-quoted strings from a DOCTYPE directive remainder.
func extractDocTypeIDs(s string) []string {
	var ids []string
	for len(ids) < 2 {
		s = strings.TrimSpace(s)
		if len(s) == 0 {
			break
		}
		// skip keyword PUBLIC/SYSTEM
		if strings.HasPrefix(s, "PUBLIC") || strings.HasPrefix(s, "SYSTEM") {
			idx := strings.IndexByte(s, '"')
			if idx < 0 {
				break
			}
			s = s[idx:]
			continue
		}
		if s[0] != '"' {
			break
		}
		end := strings.IndexByte(s[1:], '"')
		if end < 0 {
			break
		}
		ids = append(ids, s[1:end+1])
		s = s[end+2:]
	}
	return ids
}

// xmlMakeDecl builds an XML declaration string matching CPython minidom behavior.
// Plain case (no encoding, no standalone) uses a trailing space: <?xml version="1.0" ?>
// Any other case omits the trailing space: <?xml version="1.0" encoding="utf-8"?>
func xmlMakeDecl(encoding string, standalone object.Object) string {
	hasStandalone := standalone != nil && standalone != object.None && (standalone == object.True || standalone == object.False)
	if encoding == "" && !hasStandalone {
		return `<?xml version="1.0" ?>`
	}
	decl := `<?xml version="1.0"`
	if encoding != "" {
		decl += ` encoding="` + encoding + `"`
	}
	if hasStandalone {
		if standalone == object.True {
			decl += ` standalone="yes"`
		} else {
			decl += ` standalone="no"`
		}
	}
	decl += "?>"
	return decl
}

// domParseEncodingStandalone extracts encoding and standalone from positional/kw args.
func domParseEncodingStandalone(extra []object.Object, kw *object.Dict) (string, object.Object) {
	enc := ""
	sa := object.Object(object.None)
	if len(extra) >= 1 {
		if s, ok := extra[0].(*object.Str); ok {
			enc = s.V
		}
	}
	if len(extra) >= 2 {
		sa = extra[1]
	}
	if kw != nil {
		if v, ok := kw.GetStr("encoding"); ok {
			if s, ok2 := v.(*object.Str); ok2 {
				enc = s.V
			}
		}
		if v, ok := kw.GetStr("standalone"); ok {
			sa = v
		}
	}
	return enc, sa
}

// domWriteXML serializes a DOM node to a strings.Builder using CPython minidom rules.
func domWriteXML(buf *strings.Builder, inst *object.Instance, indent, addindent, newl string) {
	st := getDomNode(inst)
	if st == nil {
		// Bare Document-like node without state (legacy path)
		if docElem, ok := inst.Dict.GetStr("documentElement"); ok && docElem != object.None {
			if docInst, ok2 := docElem.(*object.Instance); ok2 {
				domWriteXML(buf, docInst, indent, addindent, newl)
			}
		}
		return
	}

	switch st.nodeType {
	case domDOCUMENT_NODE:
		for _, child := range st.children {
			domWriteXML(buf, child, indent, addindent, newl)
		}

	case domELEMENT_NODE:
		buf.WriteString(indent)
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
			buf.WriteString(newl)
		} else {
			// Single TEXT_NODE child → inline
			singleText := false
			if len(st.children) == 1 {
				if chSt := getDomNode(st.children[0]); chSt != nil && chSt.nodeType == domTEXT_NODE {
					singleText = true
				}
			}
			buf.WriteByte('>')
			if singleText {
				chSt := getDomNode(st.children[0])
				buf.WriteString(xmlEscapeText(chSt.data))
				buf.WriteString("</")
				buf.WriteString(st.nodeName)
				buf.WriteByte('>')
				buf.WriteString(newl)
			} else {
				buf.WriteString(newl)
				childIndent := indent + addindent
				for _, child := range st.children {
					domWriteXML(buf, child, childIndent, addindent, newl)
				}
				buf.WriteString(indent)
				buf.WriteString("</")
				buf.WriteString(st.nodeName)
				buf.WriteByte('>')
				buf.WriteString(newl)
			}
		}

	case domTEXT_NODE:
		buf.WriteString(indent)
		buf.WriteString(xmlEscapeText(st.data))
		buf.WriteString(newl)

	case domCDATA_SECTION_NODE:
		buf.WriteString(indent)
		buf.WriteString("<![CDATA[")
		buf.WriteString(st.data)
		buf.WriteString("]]>")
		buf.WriteString(newl)

	case domCOMMENT_NODE:
		buf.WriteString(indent)
		buf.WriteString("<!--")
		buf.WriteString(st.data)
		buf.WriteString("-->")
		buf.WriteString(newl)

	case domPROCESSING_INSTRUCTION_NODE:
		buf.WriteString(indent)
		buf.WriteString("<?")
		buf.WriteString(st.target)
		if st.data != "" {
			buf.WriteByte(' ')
			buf.WriteString(st.data)
		}
		buf.WriteString("?>")
		buf.WriteString(newl)

	case domDOCUMENT_TYPE_NODE:
		buf.WriteString(indent)
		buf.WriteString("<!DOCTYPE ")
		buf.WriteString(st.nodeName)
		if st.publicId != "" {
			buf.WriteString(` PUBLIC "`)
			buf.WriteString(st.publicId)
			buf.WriteString(`" "`)
			buf.WriteString(st.systemId)
			buf.WriteByte('"')
		} else if st.systemId != "" {
			buf.WriteString(` SYSTEM "`)
			buf.WriteString(st.systemId)
			buf.WriteByte('"')
		}
		buf.WriteByte('>')
		buf.WriteString(newl)

	case domDOCUMENT_FRAGMENT_NODE:
		for _, child := range st.children {
			domWriteXML(buf, child, indent, addindent, newl)
		}
	}
}

// domCloneNode creates a shallow or deep clone of a DOM node.
func domCloneNode(inst *object.Instance, deep bool, ownerDoc *object.Instance) *object.Instance {
	if inst == nil {
		return nil
	}
	st := getDomNode(inst)
	if st == nil {
		return nil
	}
	newAttrs := make([]domAttr, len(st.attrs))
	copy(newAttrs, st.attrs)
	newInst := &object.Instance{Class: inst.Class, Dict: object.NewDict()}
	newSt := &domNodeState{
		nodeType:  st.nodeType,
		nodeName:  st.nodeName,
		nodeValue: st.nodeValue,
		data:      st.data,
		target:    st.target,
		attrs:     newAttrs,
		ownerDoc:  ownerDoc,
	}
	domNodeMap.Store(newInst, newSt)
	domSyncNodeDict(newInst, newSt)
	if deep {
		for _, ch := range st.children {
			clonedCh := domCloneNode(ch, true, ownerDoc)
			if clonedCh != nil {
				newSt.children = append(newSt.children, clonedCh)
				clonedCh.Dict.SetStr("parentNode", newInst)
				if chSt := getDomNode(clonedCh); chSt != nil {
					chSt.parent = newInst
				}
			}
		}
	}
	return newInst
}

// domNodesEqual returns true if two nodes are structurally equal.
func domNodesEqual(a, b *object.Instance) bool {
	if a == b {
		return true
	}
	stA := getDomNode(a)
	stB := getDomNode(b)
	if stA == nil || stB == nil {
		return stA == stB
	}
	if stA.nodeType != stB.nodeType || stA.nodeName != stB.nodeName ||
		stA.nodeValue != stB.nodeValue || stA.data != stB.data ||
		stA.target != stB.target {
		return false
	}
	if len(stA.attrs) != len(stB.attrs) {
		return false
	}
	for _, da := range stA.attrs {
		found := false
		for _, db := range stB.attrs {
			if da.name == db.name && da.value == db.value {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(stA.children) != len(stB.children) {
		return false
	}
	for idx := range stA.children {
		if !domNodesEqual(stA.children[idx], stB.children[idx]) {
			return false
		}
	}
	return true
}

// domNormalize merges adjacent Text nodes in children recursively.
func domNormalize(inst *object.Instance) {
	st := getDomNode(inst)
	if st == nil {
		return
	}
	idx := 0
	for idx < len(st.children) {
		ch := st.children[idx]
		chSt := getDomNode(ch)
		if chSt != nil && chSt.nodeType == domTEXT_NODE {
			merged := chSt.data
			j := idx + 1
			for j < len(st.children) {
				nSt := getDomNode(st.children[j])
				if nSt == nil || nSt.nodeType != domTEXT_NODE {
					break
				}
				merged += nSt.data
				j++
			}
			if j > idx+1 {
				chSt.data = merged
				chSt.nodeValue = merged
				ch.Dict.SetStr("data", &object.Str{V: merged})
				ch.Dict.SetStr("nodeValue", &object.Str{V: merged})
				st.children = append(st.children[:idx+1], st.children[j:]...)
			} else {
				idx++
			}
		} else {
			if chSt != nil {
				domNormalize(ch)
			}
			idx++
		}
	}
}

// installCharacterDataMethods adds the CharacterData interface to a node class.
func (i *Interp) installCharacterDataMethods(cls *object.Class) {
	// __setattr__ syncs data/nodeValue writes to domNodeState
	cls.Dict.SetStr("__setattr__", &object.BuiltinFunc{Name: "__setattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		attrName := ""
		if s, ok2 := a[1].(*object.Str); ok2 {
			attrName = s.V
		}
		val := a[2]
		self.Dict.SetStr(attrName, val)
		if attrName == "data" || attrName == "nodeValue" {
			if st := getDomNode(self); st != nil {
				if s, ok2 := val.(*object.Str); ok2 {
					st.data = s.V
					st.nodeValue = s.V
				}
			}
		}
		return object.None, nil
	}})

	// length -> int (rune count)
	cls.Dict.SetStr("length", &object.Property{Fget: &object.BuiltinFunc{Name: "length", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.IntFromInt64(0), nil
		}
		st := getDomNode(a[0].(*object.Instance))
		if st == nil {
			return object.IntFromInt64(0), nil
		}
		return object.IntFromInt64(int64(len([]rune(st.data)))), nil
	}}})

	// substringData(offset, count) -> str
	cls.Dict.SetStr("substringData", &object.BuiltinFunc{Name: "substringData", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return &object.Str{V: ""}, nil
		}
		st := getDomNode(a[0].(*object.Instance))
		if st == nil {
			return &object.Str{V: ""}, nil
		}
		offset, ok1 := toInt64(a[1])
		count, ok2 := toInt64(a[2])
		if !ok1 || !ok2 {
			return &object.Str{V: ""}, nil
		}
		runes := []rune(st.data)
		off := int(offset)
		cnt := int(count)
		if off < 0 || off > len(runes) {
			return nil, object.Errorf(i.indexErr, "INDEX_SIZE_ERR")
		}
		end := off + cnt
		if end > len(runes) {
			end = len(runes)
		}
		return &object.Str{V: string(runes[off:end])}, nil
	}})

	// appendData(arg)
	cls.Dict.SetStr("appendData", &object.BuiltinFunc{Name: "appendData", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		if s, ok := a[1].(*object.Str); ok {
			st.data += s.V
			st.nodeValue = st.data
			self.Dict.SetStr("data", &object.Str{V: st.data})
			self.Dict.SetStr("nodeValue", &object.Str{V: st.nodeValue})
		}
		return object.None, nil
	}})

	// insertData(offset, arg)
	cls.Dict.SetStr("insertData", &object.BuiltinFunc{Name: "insertData", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		offset, ok := toInt64(a[1])
		if !ok {
			return object.None, nil
		}
		arg := ""
		if s, ok2 := a[2].(*object.Str); ok2 {
			arg = s.V
		}
		runes := []rune(st.data)
		off := int(offset)
		if off < 0 || off > len(runes) {
			return nil, object.Errorf(i.indexErr, "INDEX_SIZE_ERR")
		}
		newR := make([]rune, 0, len(runes)+len([]rune(arg)))
		newR = append(newR, runes[:off]...)
		newR = append(newR, []rune(arg)...)
		newR = append(newR, runes[off:]...)
		st.data = string(newR)
		st.nodeValue = st.data
		self.Dict.SetStr("data", &object.Str{V: st.data})
		self.Dict.SetStr("nodeValue", &object.Str{V: st.nodeValue})
		return object.None, nil
	}})

	// deleteData(offset, count)
	cls.Dict.SetStr("deleteData", &object.BuiltinFunc{Name: "deleteData", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		offset, ok1 := toInt64(a[1])
		count, ok2 := toInt64(a[2])
		if !ok1 || !ok2 {
			return object.None, nil
		}
		runes := []rune(st.data)
		off := int(offset)
		cnt := int(count)
		if off < 0 || off > len(runes) {
			return nil, object.Errorf(i.indexErr, "INDEX_SIZE_ERR")
		}
		end := off + cnt
		if end > len(runes) {
			end = len(runes)
		}
		newR := make([]rune, 0, len(runes))
		newR = append(newR, runes[:off]...)
		newR = append(newR, runes[end:]...)
		st.data = string(newR)
		st.nodeValue = st.data
		self.Dict.SetStr("data", &object.Str{V: st.data})
		self.Dict.SetStr("nodeValue", &object.Str{V: st.nodeValue})
		return object.None, nil
	}})

	// replaceData(offset, count, arg)
	cls.Dict.SetStr("replaceData", &object.BuiltinFunc{Name: "replaceData", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 4 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getDomNode(self)
		if st == nil {
			return object.None, nil
		}
		offset, ok1 := toInt64(a[1])
		count, ok2 := toInt64(a[2])
		if !ok1 || !ok2 {
			return object.None, nil
		}
		arg := ""
		if s, ok3 := a[3].(*object.Str); ok3 {
			arg = s.V
		}
		runes := []rune(st.data)
		off := int(offset)
		cnt := int(count)
		if off < 0 || off > len(runes) {
			return nil, object.Errorf(i.indexErr, "INDEX_SIZE_ERR")
		}
		end := off + cnt
		if end > len(runes) {
			end = len(runes)
		}
		newR := make([]rune, 0, len(runes))
		newR = append(newR, runes[:off]...)
		newR = append(newR, []rune(arg)...)
		newR = append(newR, runes[end:]...)
		st.data = string(newR)
		st.nodeValue = st.data
		self.Dict.SetStr("data", &object.Str{V: st.data})
		self.Dict.SetStr("nodeValue", &object.Str{V: st.nodeValue})
		return object.None, nil
	}})
}

// installTextMethods adds Text-specific methods.
func (i *Interp) installTextMethods(cls *object.Class) {
	// splitText(offset) -> Text (new node with tail)
	cls.Dict.SetStr("splitText", &object.BuiltinFunc{Name: "splitText", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.indexErr, "INDEX_SIZE_ERR")
		}
		self := a[0].(*object.Instance)
		offset, ok := toInt64(a[1])
		if !ok {
			return nil, object.Errorf(i.typeErr, "offset must be int")
		}
		st := getDomNode(self)
		if st == nil {
			return nil, object.Errorf(i.typeErr, "splitText on non-text node")
		}
		runes := []rune(st.data)
		off := int(offset)
		if off < 0 || off > len(runes) {
			return nil, object.Errorf(i.indexErr, "INDEX_SIZE_ERR")
		}
		tail := string(runes[off:])
		head := string(runes[:off])
		// Create new Text node with tail
		newInst := &object.Instance{Class: self.Class, Dict: object.NewDict()}
		newSt := &domNodeState{
			nodeType:  domTEXT_NODE,
			nodeName:  "#text",
			nodeValue: tail,
			data:      tail,
			ownerDoc:  st.ownerDoc,
		}
		domNodeMap.Store(newInst, newSt)
		domSyncNodeDict(newInst, newSt)
		// Shrink this node to the head
		st.data = head
		st.nodeValue = head
		self.Dict.SetStr("data", &object.Str{V: head})
		self.Dict.SetStr("nodeValue", &object.Str{V: head})
		// Insert new node after self in parent's children
		if st.parent != nil {
			parentSt := getDomNode(st.parent)
			if parentSt != nil {
				for idx, ch := range parentSt.children {
					if ch == self {
						newSt.parent = st.parent
						newInst.Dict.SetStr("parentNode", st.parent)
						newChildren := make([]*object.Instance, 0, len(parentSt.children)+1)
						newChildren = append(newChildren, parentSt.children[:idx+1]...)
						newChildren = append(newChildren, newInst)
						newChildren = append(newChildren, parentSt.children[idx+1:]...)
						parentSt.children = newChildren
						break
					}
				}
			}
		}
		return newInst, nil
	}})
}

// ── xml.dom.pulldom ───────────────────────────────────────────────────────────

func (i *Interp) buildXmlDomPulldom() *object.Module {
	ensureDomSharedClasses(i)
	_, elemCls, textCls, commentCls, piCls, _, _, _ := i.buildMinidomClasses()

	m := &object.Module{Name: "xml.dom.pulldom", Dict: object.NewDict()}

	// Module-level default buffer size
	m.Dict.SetStr("default_bufsize", object.IntFromInt64(8192))

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

	// helper: read _pos from instance dict
	getPos := func(self *object.Instance) int {
		v, ok := self.Dict.GetStr("_pos")
		if !ok {
			return 0
		}
		if n, ok2 := toInt64(v); ok2 {
			return int(n)
		}
		return 0
	}
	// helper: read _events slice from instance dict
	getEvents := func(self *object.Instance) []object.Object {
		v, ok := self.Dict.GetStr("_events")
		if !ok {
			return nil
		}
		if lst, ok2 := v.(*object.List); ok2 {
			return lst.V
		}
		return nil
	}

	domesCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if len(a) >= 2 {
			self.Dict.SetStr("_events", a[1])
		}
		self.Dict.SetStr("_pos", object.IntFromInt64(0))
		return object.None, nil
	}})

	// getEvent() -> (event, node) or None
	domesCls.Dict.SetStr("getEvent", &object.BuiltinFunc{Name: "getEvent", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		events := getEvents(self)
		pos := getPos(self)
		if pos >= len(events) {
			return object.None, nil
		}
		self.Dict.SetStr("_pos", object.IntFromInt64(int64(pos+1)))
		return events[pos], nil
	}})

	// reset() — restart the stream
	domesCls.Dict.SetStr("reset", &object.BuiltinFunc{Name: "reset", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		a[0].(*object.Instance).Dict.SetStr("_pos", object.IntFromInt64(0))
		return object.None, nil
	}})

	// __iter__ — returns self (proper iterator protocol)
	domesCls.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		return a[0], nil
	}})

	// __next__ — advances via getEvent, raises StopIteration when done
	domesCls.Dict.SetStr("__next__", &object.BuiltinFunc{Name: "__next__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.stopIter, "")
		}
		self := a[0].(*object.Instance)
		events := getEvents(self)
		pos := getPos(self)
		if pos >= len(events) {
			return nil, object.Errorf(i.stopIter, "")
		}
		self.Dict.SetStr("_pos", object.IntFromInt64(int64(pos+1)))
		return events[pos], nil
	}})

	// expandNode(node) — consume events and build node's full subtree
	domesCls.Dict.SetStr("expandNode", &object.BuiltinFunc{Name: "expandNode", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		node, ok := a[1].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		events := getEvents(self)
		pos := getPos(self)

		// Stack of parents: we consume until we see the END_ELEMENT that
		// matches the node passed in (depth goes back to 0).
		parents := []*object.Instance{node}
		for pos < len(events) && len(parents) > 0 {
			ev, ok2 := events[pos].(*object.Tuple)
			pos++
			if !ok2 || len(ev.V) < 2 {
				continue
			}
			tok := ""
			if s, ok3 := ev.V[0].(*object.Str); ok3 {
				tok = s.V
			}
			child, _ := ev.V[1].(*object.Instance)

			switch tok {
			case "START_ELEMENT":
				parent := parents[len(parents)-1]
				if child != nil {
					pulldomAppendChild(parent, child)
				}
				parents = append(parents, child)
			case "END_ELEMENT":
				parents = parents[:len(parents)-1]
			case "CHARACTERS", "COMMENT", "PROCESSING_INSTRUCTION", "IGNORABLE_WHITESPACE":
				if child != nil && len(parents) > 0 {
					pulldomAppendChild(parents[len(parents)-1], child)
				}
			}
		}
		self.Dict.SetStr("_pos", object.IntFromInt64(int64(pos)))
		return object.None, nil
	}})

	m.Dict.SetStr("DOMEventStream", domesCls)

	// PullDOM — internal SAX handler stub
	pulldomCls := &object.Class{Name: "PullDOM", Dict: object.NewDict()}
	m.Dict.SetStr("PullDOM", pulldomCls)

	// SAX2DOM class (stub)
	sax2domCls := &object.Class{Name: "SAX2DOM", Dict: object.NewDict()}
	m.Dict.SetStr("SAX2DOM", sax2domCls)

	// makeDOMStream creates a DOMEventStream instance from a parsed event list
	makeDOMStream := func(events []object.Object) *object.Instance {
		inst := &object.Instance{Class: domesCls, Dict: object.NewDict()}
		inst.Dict.SetStr("_events", &object.List{V: events})
		inst.Dict.SetStr("_pos", object.IntFromInt64(0))
		return inst
	}

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
		events, err := pulldomParseFull(data, elemCls, textCls, commentCls, piCls)
		if err != nil {
			return nil, err
		}
		return makeDOMStream(events), nil
	}})

	// parseString(string, parser=None) -> DOMEventStream
	m.Dict.SetStr("parseString", &object.BuiltinFunc{Name: "parseString", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "parseString() requires at least 1 argument")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "parseString() requires bytes or str")
		}
		events, perr := pulldomParseFull(data, elemCls, textCls, commentCls, piCls)
		if perr != nil {
			return nil, perr
		}
		return makeDOMStream(events), nil
	}})

	return m
}

// pulldomAppendChild appends a child to a parent in the DOM state (used by expandNode).
func pulldomAppendChild(parent, child *object.Instance) {
	if parent == nil || child == nil {
		return
	}
	parentSt := getDomNode(parent)
	childSt := getDomNode(child)
	if parentSt == nil || childSt == nil {
		return
	}
	parentSt.children = append(parentSt.children, child)
	childSt.parent = parent
	child.Dict.SetStr("parentNode", parent)
}

// pulldomParseFull parses XML bytes into a list of (event, node) tuples using real minidom nodes.
func pulldomParseFull(data []byte, elemCls, textCls, commentCls, piCls *object.Class) ([]object.Object, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false

	// Document node for START_DOCUMENT / END_DOCUMENT events
	docCls := &object.Class{Name: "Document", Dict: object.NewDict()}
	docInst := &object.Instance{Class: docCls, Dict: object.NewDict()}
	docSt := &domNodeState{nodeType: domDOCUMENT_NODE, nodeName: "#document"}
	domNodeMap.Store(docInst, docSt)
	domSyncNodeDict(docInst, docSt)

	var events []object.Object
	events = append(events, &object.Tuple{V: []object.Object{&object.Str{V: "START_DOCUMENT"}, docInst}})

	// Track element stack so END_ELEMENT returns the same instance as START_ELEMENT
	var elemStack []*object.Instance

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
			elemStack = append(elemStack, inst)
			events = append(events, &object.Tuple{V: []object.Object{&object.Str{V: "START_ELEMENT"}, inst}})

		case xml.EndElement:
			var endInst object.Object = object.None
			if len(elemStack) > 0 {
				endInst = elemStack[len(elemStack)-1]
				elemStack = elemStack[:len(elemStack)-1]
			}
			events = append(events, &object.Tuple{V: []object.Object{&object.Str{V: "END_ELEMENT"}, endInst}})

		case xml.CharData:
			text := string(t)
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
			evType := "CHARACTERS"
			if strings.TrimSpace(text) == "" {
				evType = "IGNORABLE_WHITESPACE"
			}
			events = append(events, &object.Tuple{V: []object.Object{&object.Str{V: evType}, inst}})

		case xml.Comment:
			inst := &object.Instance{Class: commentCls, Dict: object.NewDict()}
			st := &domNodeState{
				nodeType: domCOMMENT_NODE,
				nodeName: "#comment",
				data:     string(t),
				ownerDoc: docInst,
			}
			domNodeMap.Store(inst, st)
			domSyncNodeDict(inst, st)
			events = append(events, &object.Tuple{V: []object.Object{&object.Str{V: "COMMENT"}, inst}})

		case xml.ProcInst:
			if t.Target == "xml" {
				continue
			}
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
			events = append(events, &object.Tuple{V: []object.Object{&object.Str{V: "PROCESSING_INSTRUCTION"}, inst}})
		}
	}

	events = append(events, &object.Tuple{V: []object.Object{&object.Str{V: "END_DOCUMENT"}, docInst}})
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


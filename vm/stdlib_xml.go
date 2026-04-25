package vm

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/tamnd/goipy/object"
)

// ── etElem: Go-side state for xml.etree.ElementTree.Element ──────────────────

type etAttr struct{ name, value string }

type etElem struct {
	tag      string
	attrib   []etAttr
	text     string
	tail     string
	children []*object.Instance
}

type etElemRegistry struct{ m sync.Map }

func (r *etElemRegistry) Store(k *object.Instance, v *etElem) { r.m.Store(k, v) }
func (r *etElemRegistry) Load(k *object.Instance) *etElem {
	v, ok := r.m.Load(k)
	if !ok {
		return nil
	}
	return v.(*etElem)
}

var etElemMap etElemRegistry

func getEtElem(inst *object.Instance) *etElem {
	return etElemMap.Load(inst)
}

// ── namespace registry ────────────────────────────────────────────────────────

var etNamespaces struct {
	mu   sync.Mutex
	list []struct{ prefix, uri string }
}

// ── xml module ───────────────────────────────────────────────────────────────

func (i *Interp) buildXml() *object.Module {
	m := &object.Module{Name: "xml", Dict: object.NewDict()}
	m.Dict.SetStr("__path__", &object.List{V: []object.Object{&object.Str{V: ""}}})
	m.Dict.SetStr("__package__", &object.Str{V: "xml"})
	return m
}

func (i *Interp) buildXmlEtree() *object.Module {
	m := &object.Module{Name: "xml.etree", Dict: object.NewDict()}
	m.Dict.SetStr("__path__", &object.List{V: []object.Object{&object.Str{V: ""}}})
	m.Dict.SetStr("__package__", &object.Str{V: "xml.etree"})
	return m
}

// ── xml.etree.ElementTree ────────────────────────────────────────────────────

func (i *Interp) buildXmlElementTree() *object.Module {
	m := &object.Module{Name: "xml.etree.ElementTree", Dict: object.NewDict()}

	// ParseError exception
	parseErrCls := &object.Class{Name: "ParseError", Dict: object.NewDict(), Bases: []*object.Class{i.syntaxErr}}
	m.Dict.SetStr("ParseError", parseErrCls)

	// Element class
	elemCls := i.buildEtElementClass(parseErrCls)
	m.Dict.SetStr("Element", elemCls)

	// ElementTree class
	etreeCls := i.buildEtElementTreeClass(elemCls, parseErrCls)
	m.Dict.SetStr("ElementTree", etreeCls)

	// Module-level functions

	// Element() constructor
	m.Dict.SetStr("Element", &object.BuiltinFunc{Name: "Element", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return etMakeElement(elemCls, a, kw)
	}})

	// SubElement(parent, tag, attrib={}, **extra)
	m.Dict.SetStr("SubElement", &object.BuiltinFunc{Name: "SubElement", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return nil, object.Errorf(i.typeErr, "SubElement() requires at least 2 arguments")
		}
		parent, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "SubElement() first arg must be an Element")
		}
		child, err := etMakeElement(elemCls, a[1:], kw)
		if err != nil {
			return nil, err
		}
		childInst := child.(*object.Instance)
		if st := getEtElem(parent); st != nil {
			st.children = append(st.children, childInst)
		}
		return child, nil
	}})

	// Comment(text=None)
	m.Dict.SetStr("Comment", &object.BuiltinFunc{Name: "Comment", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		text := ""
		if len(a) >= 1 {
			if s, ok := a[0].(*object.Str); ok {
				text = s.V
			}
		}
		inst := &object.Instance{Class: elemCls, Dict: object.NewDict()}
		st := &etElem{tag: "<!---->"}
		st.text = text
		etElemMap.Store(inst, st)
		etSyncToDict(inst, st)
		return inst, nil
	}})

	// ProcessingInstruction(target, text=None)
	m.Dict.SetStr("ProcessingInstruction", &object.BuiltinFunc{Name: "ProcessingInstruction", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		target := ""
		text := ""
		if len(a) >= 1 {
			if s, ok := a[0].(*object.Str); ok {
				target = s.V
			}
		}
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				text = s.V
			}
		}
		inst := &object.Instance{Class: elemCls, Dict: object.NewDict()}
		st := &etElem{tag: "<??>"}
		st.text = target
		if text != "" {
			st.tail = text
		}
		etElemMap.Store(inst, st)
		etSyncToDict(inst, st)
		return inst, nil
	}})

	// parse(source, parser=None) -> ElementTree
	m.Dict.SetStr("parse", &object.BuiltinFunc{Name: "parse", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "parse() requires at least 1 argument")
		}
		var data []byte
		var err error
		switch v := a[0].(type) {
		case *object.Str:
			// treat as file path
			data, err = etReadFile(v.V)
		case *object.Bytes:
			data = v.V
		case *object.Bytearray:
			data = v.V
		default:
			// try calling .read()
			ii := interp.(*Interp)
			fn, ferr := ii.getAttr(a[0], "read")
			if ferr != nil {
				return nil, object.Errorf(i.typeErr, "parse() requires a filename or file-like object")
			}
			res, rerr := ii.callObject(fn, nil, nil)
			if rerr != nil {
				return nil, rerr
			}
			data, _ = asBytes(res)
		}
		if err != nil {
			return nil, object.Errorf(i.osErr, "parse() could not read file: %v", err)
		}
		root, perr := etParseXML(elemCls, parseErrCls, data)
		if perr != nil {
			return nil, perr
		}
		etInst := &object.Instance{Class: etreeCls, Dict: object.NewDict()}
		etInst.Dict.SetStr("_root", root)
		etInst.Dict.SetStr("_file", object.None)
		return etInst, nil
	}})

	// fromstring(text, parser=None) -> Element
	fromstringFn := &object.BuiltinFunc{Name: "fromstring", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fromstring() requires at least 1 argument")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "fromstring() requires a string or bytes object")
		}
		return etParseXML(elemCls, parseErrCls, data)
	}}
	m.Dict.SetStr("fromstring", fromstringFn)
	m.Dict.SetStr("XML", fromstringFn)

	// fromstringlist(sequence, parser=None) -> Element
	m.Dict.SetStr("fromstringlist", &object.BuiltinFunc{Name: "fromstringlist", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "fromstringlist() requires at least 1 argument")
		}
		var buf bytes.Buffer
		switch v := a[0].(type) {
		case *object.List:
			for _, item := range v.V {
				b, _ := asBytes(item)
				buf.Write(b)
			}
		case *object.Tuple:
			for _, item := range v.V {
				b, _ := asBytes(item)
				buf.Write(b)
			}
		default:
			b, _ := asBytes(a[0])
			buf.Write(b)
		}
		return etParseXML(elemCls, parseErrCls, buf.Bytes())
	}})

	// tostring(element, encoding='us-ascii', method='xml', ...)
	m.Dict.SetStr("tostring", &object.BuiltinFunc{Name: "tostring", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "tostring() requires at least 1 argument")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "tostring() requires an Element")
		}
		encoding := "us-ascii"
		method := "xml"
		xmlDecl := false
		shortEmpty := true
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				encoding = s.V
			}
		}
		if kw != nil {
			if v, ok := kw.GetStr("encoding"); ok {
				if s, ok := v.(*object.Str); ok {
					encoding = s.V
				}
			}
			if v, ok := kw.GetStr("method"); ok {
				if s, ok := v.(*object.Str); ok {
					method = s.V
				}
			}
			if v, ok := kw.GetStr("xml_declaration"); ok {
				xmlDecl = object.Truthy(v)
			}
			if v, ok := kw.GetStr("short_empty_elements"); ok {
				shortEmpty = object.Truthy(v)
			}
		}
		var buf strings.Builder
		if xmlDecl {
			buf.WriteString(`<?xml version='1.0' encoding='`)
			buf.WriteString(encoding)
			buf.WriteString(`'?>` + "\n")
		}
		etSerialize(&buf, inst, method, shortEmpty)
		result := buf.String()
		if strings.EqualFold(encoding, "unicode") {
			return &object.Str{V: result}, nil
		}
		return &object.Bytes{V: []byte(result)}, nil
	}})

	// tostringlist
	m.Dict.SetStr("tostringlist", &object.BuiltinFunc{Name: "tostringlist", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "tostringlist() requires at least 1 argument")
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return nil, object.Errorf(i.typeErr, "tostringlist() requires an Element")
		}
		var buf strings.Builder
		etSerialize(&buf, inst, "xml", true)
		return &object.List{V: []object.Object{&object.Bytes{V: []byte(buf.String())}}}, nil
	}})

	// indent(tree, space='  ', level=0)
	m.Dict.SetStr("indent", &object.BuiltinFunc{Name: "indent", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		space := "  "
		level := 0
		if kw != nil {
			if v, ok := kw.GetStr("space"); ok {
				if s, ok := v.(*object.Str); ok {
					space = s.V
				}
			}
			if v, ok := kw.GetStr("level"); ok {
				if n, ok := toInt64(v); ok {
					level = int(n)
				}
			}
		}
		var root *object.Instance
		switch v := a[0].(type) {
		case *object.Instance:
			root = v
		default:
			return object.None, nil
		}
		etIndent(root, space, level, true)
		return object.None, nil
	}})

	// dump(elem) — print to stdout
	m.Dict.SetStr("dump", &object.BuiltinFunc{Name: "dump", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		inst, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		var buf strings.Builder
		etSerialize(&buf, inst, "xml", true)
		ii := interp.(*Interp)
		fmt.Fprintln(ii.Stdout, buf.String())
		return object.None, nil
	}})

	// XMLID(text, parser=None) -> (Element, dict)
	m.Dict.SetStr("XMLID", &object.BuiltinFunc{Name: "XMLID", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "XMLID() requires at least 1 argument")
		}
		data, err := asBytes(a[0])
		if err != nil {
			return nil, object.Errorf(i.typeErr, "XMLID() requires a string or bytes object")
		}
		root, perr := etParseXML(elemCls, parseErrCls, data)
		if perr != nil {
			return nil, perr
		}
		idMap := object.NewDict()
		etCollectIDs(root.(*object.Instance), idMap)
		return &object.Tuple{V: []object.Object{root, idMap}}, nil
	}})

	// register_namespace(prefix, uri)
	m.Dict.SetStr("register_namespace", &object.BuiltinFunc{Name: "register_namespace", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		prefix := ""
		uri := ""
		if s, ok := a[0].(*object.Str); ok {
			prefix = s.V
		}
		if s, ok := a[1].(*object.Str); ok {
			uri = s.V
		}
		etNamespaces.mu.Lock()
		etNamespaces.list = append(etNamespaces.list, struct{ prefix, uri string }{prefix, uri})
		etNamespaces.mu.Unlock()
		return object.None, nil
	}})

	// QName stub
	qnameCls := &object.Class{Name: "QName", Dict: object.NewDict()}
	qnameCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		if s, ok := a[1].(*object.Str); ok {
			self.Dict.SetStr("text", s)
		}
		return object.None, nil
	}})
	m.Dict.SetStr("QName", qnameCls)

	// iterparse stub (returns list of (event, element))
	m.Dict.SetStr("iterparse", &object.BuiltinFunc{Name: "iterparse", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return nil, object.Errorf(i.typeErr, "iterparse() requires at least 1 argument")
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
			fn, _ := ii.getAttr(a[0], "read")
			res, _ := ii.callObject(fn, nil, nil)
			data, _ = asBytes(res)
		}
		root, perr := etParseXML(elemCls, parseErrCls, data)
		if perr != nil {
			return nil, perr
		}
		events := []object.Object{}
		etCollectIterparse(root.(*object.Instance), &events)
		return &object.List{V: events}, nil
	}})

	// canonicalize stub
	m.Dict.SetStr("canonicalize", &object.BuiltinFunc{Name: "canonicalize", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return object.None, nil
	}})

	// expose Element class
	m.Dict.SetStr("ElementTree", etreeCls)

	return m
}

// buildEtElementClass builds the Element class with all methods.
func (i *Interp) buildEtElementClass(parseErrCls *object.Class) *object.Class {
	elemCls := &object.Class{Name: "Element", Dict: object.NewDict()}

	// __init__(tag, attrib={}, **extra)
	elemCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		tag := ""
		if s, ok := a[1].(*object.Str); ok {
			tag = s.V
		}
		st := &etElem{tag: tag}
		// attrib from positional arg
		if len(a) >= 3 {
			if d, ok := a[2].(*object.Dict); ok {
				ks, vs := d.Items()
				for idx2, k := range ks {
					if ks2, ok2 := k.(*object.Str); ok2 {
						if vs2, ok3 := vs[idx2].(*object.Str); ok3 {
							st.attrib = append(st.attrib, etAttr{ks2.V, vs2.V})
						}
					}
				}
			}
		}
		// kw extras
		if kw != nil {
			ks, vs := kw.Items()
			for idx2, k := range ks {
				if ks2, ok2 := k.(*object.Str); ok2 {
					val := ""
					if vs2, ok3 := vs[idx2].(*object.Str); ok3 {
						val = vs2.V
					}
					st.attrib = append(st.attrib, etAttr{ks2.V, val})
				}
			}
		}
		etElemMap.Store(self, st)
		etSyncToDict(self, st)
		return object.None, nil
	}})

	// __repr__
	elemCls.Dict.SetStr("__repr__", &object.BuiltinFunc{Name: "__repr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.Str{V: "<Element>"}, nil
		}
		self := a[0].(*object.Instance)
		st := getEtElem(self)
		tag := ""
		if st != nil {
			tag = st.tag
		}
		return &object.Str{V: fmt.Sprintf("<Element '%s' at 0x%x>", tag, uintptr(fmt.Sprintf("%p", self)[2:][0]))}, nil
	}})

	// __len__
	elemCls.Dict.SetStr("__len__", &object.BuiltinFunc{Name: "__len__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.IntFromInt64(0), nil
		}
		self := a[0].(*object.Instance)
		st := getEtElem(self)
		if st == nil {
			return object.IntFromInt64(0), nil
		}
		return object.IntFromInt64(int64(len(st.children))), nil
	}})

	// __iter__ -> returns iter over children
	elemCls.Dict.SetStr("__iter__", &object.BuiltinFunc{Name: "__iter__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		st := getEtElem(self)
		if st == nil {
			return &object.List{V: nil}, nil
		}
		items := make([]object.Object, len(st.children))
		for idx2, c := range st.children {
			items[idx2] = c
		}
		return &object.List{V: items}, nil
	}})

	// __getitem__(index) -> child or slice
	elemCls.Dict.SetStr("__getitem__", &object.BuiltinFunc{Name: "__getitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getEtElem(self)
		if st == nil {
			return nil, object.Errorf(nil, "IndexError: list index out of range")
		}
		n, ok := toInt64(a[1])
		if !ok {
			return nil, object.Errorf(nil, "IndexError: invalid index")
		}
		idx2 := int(n)
		if idx2 < 0 {
			idx2 += len(st.children)
		}
		if idx2 < 0 || idx2 >= len(st.children) {
			return nil, object.Errorf(i.indexErr, "list index out of range")
		}
		return st.children[idx2], nil
	}})

	// __setitem__(index, element)
	elemCls.Dict.SetStr("__setitem__", &object.BuiltinFunc{Name: "__setitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getEtElem(self)
		if st == nil {
			return object.None, nil
		}
		n, ok := toInt64(a[1])
		if !ok {
			return object.None, nil
		}
		idx2 := int(n)
		if idx2 < 0 {
			idx2 += len(st.children)
		}
		if idx2 >= 0 && idx2 < len(st.children) {
			if child, ok := a[2].(*object.Instance); ok {
				st.children[idx2] = child
			}
		}
		return object.None, nil
	}})

	// __delitem__(index)
	elemCls.Dict.SetStr("__delitem__", &object.BuiltinFunc{Name: "__delitem__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getEtElem(self)
		if st == nil {
			return object.None, nil
		}
		n, ok := toInt64(a[1])
		if !ok {
			return object.None, nil
		}
		idx2 := int(n)
		if idx2 < 0 {
			idx2 += len(st.children)
		}
		if idx2 >= 0 && idx2 < len(st.children) {
			st.children = append(st.children[:idx2], st.children[idx2+1:]...)
		}
		return object.None, nil
	}})

	// append(subelement)
	elemCls.Dict.SetStr("append", &object.BuiltinFunc{Name: "append", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		child, ok := a[1].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		st := getEtElem(self)
		if st != nil {
			st.children = append(st.children, child)
		}
		return object.None, nil
	}})

	// insert(index, subelement)
	elemCls.Dict.SetStr("insert", &object.BuiltinFunc{Name: "insert", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getEtElem(self)
		if st == nil {
			return object.None, nil
		}
		n, ok := toInt64(a[1])
		if !ok {
			return object.None, nil
		}
		child, ok2 := a[2].(*object.Instance)
		if !ok2 {
			return object.None, nil
		}
		idx2 := int(n)
		if idx2 < 0 {
			idx2 = 0
		}
		if idx2 >= len(st.children) {
			st.children = append(st.children, child)
		} else {
			st.children = append(st.children[:idx2], append([]*object.Instance{child}, st.children[idx2:]...)...)
		}
		return object.None, nil
	}})

	// remove(subelement)
	elemCls.Dict.SetStr("remove", &object.BuiltinFunc{Name: "remove", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		child, ok := a[1].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		st := getEtElem(self)
		if st == nil {
			return object.None, nil
		}
		for idx2, c := range st.children {
			if c == child {
				st.children = append(st.children[:idx2], st.children[idx2+1:]...)
				return object.None, nil
			}
		}
		return nil, object.Errorf(i.valueErr, "list.remove(x): x not in list")
	}})

	// extend(elements)
	elemCls.Dict.SetStr("extend", &object.BuiltinFunc{Name: "extend", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getEtElem(self)
		if st == nil {
			return object.None, nil
		}
		switch v := a[1].(type) {
		case *object.List:
			for _, item := range v.V {
				if c, ok := item.(*object.Instance); ok {
					st.children = append(st.children, c)
				}
			}
		case *object.Tuple:
			for _, item := range v.V {
				if c, ok := item.(*object.Instance); ok {
					st.children = append(st.children, c)
				}
			}
		}
		return object.None, nil
	}})

	// clear()
	elemCls.Dict.SetStr("clear", &object.BuiltinFunc{Name: "clear", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getEtElem(self)
		if st != nil {
			st.attrib = nil
			st.text = ""
			st.tail = ""
			st.children = nil
			etSyncToDict(self, st)
		}
		return object.None, nil
	}})

	// get(key, default=None)
	elemCls.Dict.SetStr("get", &object.BuiltinFunc{Name: "get", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		key := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		dflt := object.Object(object.None)
		if len(a) >= 3 {
			dflt = a[2]
		} else if kw != nil {
			if v, ok := kw.GetStr("default"); ok {
				dflt = v
			}
		}
		st := getEtElem(self)
		if st == nil {
			return dflt, nil
		}
		for _, attr := range st.attrib {
			if attr.name == key {
				return &object.Str{V: attr.value}, nil
			}
		}
		return dflt, nil
	}})

	// set(key, value)
	elemCls.Dict.SetStr("set", &object.BuiltinFunc{Name: "set", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 3 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		key := ""
		val := ""
		if s, ok := a[1].(*object.Str); ok {
			key = s.V
		}
		if s, ok := a[2].(*object.Str); ok {
			val = s.V
		}
		st := getEtElem(self)
		if st == nil {
			return object.None, nil
		}
		for idx2 := range st.attrib {
			if st.attrib[idx2].name == key {
				st.attrib[idx2].value = val
				etSyncAttrDict(self, st)
				return object.None, nil
			}
		}
		st.attrib = append(st.attrib, etAttr{key, val})
		etSyncAttrDict(self, st)
		return object.None, nil
	}})

	// keys() -> list of attr names
	elemCls.Dict.SetStr("keys", &object.BuiltinFunc{Name: "keys", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		st := getEtElem(self)
		if st == nil {
			return &object.List{V: nil}, nil
		}
		result := make([]object.Object, len(st.attrib))
		for idx2, attr := range st.attrib {
			result[idx2] = &object.Str{V: attr.name}
		}
		return &object.List{V: result}, nil
	}})

	// values() -> list of attr values
	elemCls.Dict.SetStr("values", &object.BuiltinFunc{Name: "values", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		st := getEtElem(self)
		if st == nil {
			return &object.List{V: nil}, nil
		}
		result := make([]object.Object, len(st.attrib))
		for idx2, attr := range st.attrib {
			result[idx2] = &object.Str{V: attr.value}
		}
		return &object.List{V: result}, nil
	}})

	// items() -> list of (name, value) tuples
	elemCls.Dict.SetStr("items", &object.BuiltinFunc{Name: "items", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		st := getEtElem(self)
		if st == nil {
			return &object.List{V: nil}, nil
		}
		result := make([]object.Object, len(st.attrib))
		for idx2, attr := range st.attrib {
			result[idx2] = &object.Tuple{V: []object.Object{
				&object.Str{V: attr.name},
				&object.Str{V: attr.value},
			}}
		}
		return &object.List{V: result}, nil
	}})

	// find(match, namespaces=None) -> Element or None
	elemCls.Dict.SetStr("find", &object.BuiltinFunc{Name: "find", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		match := ""
		if s, ok := a[1].(*object.Str); ok {
			match = s.V
		}
		result := etFind(self, match)
		if result == nil {
			return object.None, nil
		}
		return result, nil
	}})

	// findall(match, namespaces=None) -> list
	elemCls.Dict.SetStr("findall", &object.BuiltinFunc{Name: "findall", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		match := ""
		if s, ok := a[1].(*object.Str); ok {
			match = s.V
		}
		results := etFindAll(self, match)
		items := make([]object.Object, len(results))
		for idx2, r := range results {
			items[idx2] = r
		}
		return &object.List{V: items}, nil
	}})

	// findtext(match, default=None, namespaces=None) -> str or None
	elemCls.Dict.SetStr("findtext", &object.BuiltinFunc{Name: "findtext", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		match := ""
		if s, ok := a[1].(*object.Str); ok {
			match = s.V
		}
		dflt := object.Object(object.None)
		if len(a) >= 3 {
			dflt = a[2]
		}
		result := etFind(self, match)
		if result == nil {
			return dflt, nil
		}
		st := getEtElem(result)
		if st == nil {
			return dflt, nil
		}
		return &object.Str{V: st.text}, nil
	}})

	// iter(tag=None) -> list (depth-first)
	elemCls.Dict.SetStr("iter", &object.BuiltinFunc{Name: "iter", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		tag := ""
		if len(a) >= 2 {
			if s, ok := a[1].(*object.Str); ok {
				tag = s.V
			}
		} else if kw != nil {
			if v, ok := kw.GetStr("tag"); ok {
				if s, ok := v.(*object.Str); ok {
					tag = s.V
				}
			}
		}
		var results []*object.Instance
		etIterDepth(self, tag, &results)
		items := make([]object.Object, len(results))
		for idx2, r := range results {
			items[idx2] = r
		}
		return &object.List{V: items}, nil
	}})

	// iterfind(match, namespaces=None) -> list
	elemCls.Dict.SetStr("iterfind", &object.BuiltinFunc{Name: "iterfind", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		match := ""
		if s, ok := a[1].(*object.Str); ok {
			match = s.V
		}
		results := etFindAll(self, match)
		items := make([]object.Object, len(results))
		for idx2, r := range results {
			items[idx2] = r
		}
		return &object.List{V: items}, nil
	}})

	// itertext(tag=None, with_tail=True) -> list of str
	elemCls.Dict.SetStr("itertext", &object.BuiltinFunc{Name: "itertext", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return &object.List{V: nil}, nil
		}
		self := a[0].(*object.Instance)
		withTail := true
		if kw != nil {
			if v, ok := kw.GetStr("with_tail"); ok {
				withTail = object.Truthy(v)
			}
		}
		var texts []string
		etIterText(self, withTail, &texts)
		items := make([]object.Object, len(texts))
		for idx2, t := range texts {
			items[idx2] = &object.Str{V: t}
		}
		return &object.List{V: items}, nil
	}})

	// makeelement(tag, attrib) -> new Element
	elemCls.Dict.SetStr("makeelement", &object.BuiltinFunc{Name: "makeelement", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		return etMakeElement(elemCls, a[1:], nil)
	}})

	// copy() -> shallow copy
	elemCls.Dict.SetStr("copy", &object.BuiltinFunc{Name: "copy", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		st := getEtElem(self)
		if st == nil {
			return object.None, nil
		}
		newSt := &etElem{
			tag:    st.tag,
			attrib: append([]etAttr(nil), st.attrib...),
			text:   st.text,
			tail:   st.tail,
		}
		newInst := &object.Instance{Class: elemCls, Dict: object.NewDict()}
		etElemMap.Store(newInst, newSt)
		etSyncToDict(newInst, newSt)
		return newInst, nil
	}})

	// __setattr__ hook to sync dict → Go state for tag/text/tail/attrib
	elemCls.Dict.SetStr("__setattr__", &object.BuiltinFunc{Name: "__setattr__", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
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
		st := getEtElem(self)
		if st != nil {
			switch name {
			case "tag":
				if s, ok2 := val.(*object.Str); ok2 {
					st.tag = s.V
				}
			case "text":
				if val == object.None {
					st.text = ""
				} else if s, ok2 := val.(*object.Str); ok2 {
					st.text = s.V
				}
			case "tail":
				if val == object.None {
					st.tail = ""
				} else if s, ok2 := val.(*object.Str); ok2 {
					st.tail = s.V
				}
			case "attrib":
				if d, ok2 := val.(*object.Dict); ok2 {
					st.attrib = nil
					ks, vs := d.Items()
					for idx2, k := range ks {
						if ks2, ok3 := k.(*object.Str); ok3 {
							v2 := ""
							if vs2, ok4 := vs[idx2].(*object.Str); ok4 {
								v2 = vs2.V
							}
							st.attrib = append(st.attrib, etAttr{ks2.V, v2})
						}
					}
				}
			}
		}
		return object.None, nil
	}})

	return elemCls
}

// buildEtElementTreeClass builds the ElementTree class.
func (i *Interp) buildEtElementTreeClass(elemCls *object.Class, parseErrCls *object.Class) *object.Class {
	etreeCls := &object.Class{Name: "ElementTree", Dict: object.NewDict()}

	// __init__(element=None, file=None)
	etreeCls.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		self.Dict.SetStr("_root", object.None)
		self.Dict.SetStr("_file", object.None)
		if len(a) >= 2 && a[1] != object.None {
			self.Dict.SetStr("_root", a[1])
		}
		fileArg := object.Object(object.None)
		if len(a) >= 3 {
			fileArg = a[2]
		}
		if kw != nil {
			if v, ok := kw.GetStr("file"); ok {
				fileArg = v
			}
		}
		if fileArg != object.None {
			var data []byte
			switch v := fileArg.(type) {
			case *object.Str:
				var err error
				data, err = etReadFile(v.V)
				if err != nil {
					return nil, object.Errorf(i.osErr, "%v", err)
				}
			default:
				ii := interp.(*Interp)
				fn, _ := ii.getAttr(fileArg, "read")
				res, _ := ii.callObject(fn, nil, nil)
				data, _ = asBytes(res)
			}
			root, perr := etParseXML(elemCls, parseErrCls, data)
			if perr != nil {
				return nil, perr
			}
			self.Dict.SetStr("_root", root)
		}
		return object.None, nil
	}})

	// getroot()
	etreeCls.Dict.SetStr("getroot", &object.BuiltinFunc{Name: "getroot", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		v, ok := self.Dict.GetStr("_root")
		if !ok {
			return object.None, nil
		}
		return v, nil
	}})

	// parse(source, parser=None) -> Element (root)
	etreeCls.Dict.SetStr("parse", &object.BuiltinFunc{Name: "parse", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		var data []byte
		switch v := a[1].(type) {
		case *object.Str:
			var err error
			data, err = etReadFile(v.V)
			if err != nil {
				return nil, object.Errorf(i.osErr, "%v", err)
			}
		default:
			ii := interp.(*Interp)
			fn, _ := ii.getAttr(a[1], "read")
			res, _ := ii.callObject(fn, nil, nil)
			data, _ = asBytes(res)
		}
		root, perr := etParseXML(elemCls, parseErrCls, data)
		if perr != nil {
			return nil, perr
		}
		self.Dict.SetStr("_root", root)
		return root, nil
	}})

	// write(file_or_path, ...)
	etreeCls.Dict.SetStr("write", &object.BuiltinFunc{Name: "write", Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 2 {
			return object.None, nil
		}
		self := a[0].(*object.Instance)
		rootObj, ok := self.Dict.GetStr("_root")
		if !ok || rootObj == object.None {
			return object.None, nil
		}
		root, ok := rootObj.(*object.Instance)
		if !ok {
			return object.None, nil
		}
		method := "xml"
		if kw != nil {
			if v, ok := kw.GetStr("method"); ok {
				if s, ok := v.(*object.Str); ok {
					method = s.V
				}
			}
		}
		var buf strings.Builder
		etSerialize(&buf, root, method, true)
		result := buf.String()
		// Write to file
		switch v := a[1].(type) {
		case *object.Str:
			// file path - write to file
			_ = v.V
		default:
			ii := interp.(*Interp)
			fn, err := ii.getAttr(a[1], "write")
			if err == nil {
				ii.callObject(fn, []object.Object{&object.Bytes{V: []byte(result)}}, nil)
			}
		}
		return object.None, nil
	}})

	// find/findall/findtext/iter/iterfind - delegate to root
	for _, method := range []string{"find", "findall", "findtext", "iter", "iterfind"} {
		m2 := method
		etreeCls.Dict.SetStr(m2, &object.BuiltinFunc{Name: m2, Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			rootObj, ok := self.Dict.GetStr("_root")
			if !ok || rootObj == object.None {
				return object.None, nil
			}
			root, ok := rootObj.(*object.Instance)
			if !ok {
				return object.None, nil
			}
			fn, err := i.getAttr(root, m2)
			if err != nil {
				return object.None, nil
			}
			return i.callObject(fn, append([]object.Object{root}, a[1:]...), kw)
		}})
	}

	return etreeCls
}

// ── helper functions ──────────────────────────────────────────────────────────

func etReadFile(path string) ([]byte, error) {
	return nil, fmt.Errorf("file reading not supported in goipy: %s", path)
}

func etMakeElement(cls *object.Class, a []object.Object, kw *object.Dict) (object.Object, error) {
	if len(a) < 1 {
		return nil, object.Errorf(nil, "TypeError: Element() requires at least 1 argument")
	}
	tag := ""
	if s, ok := a[0].(*object.Str); ok {
		tag = s.V
	}
	st := &etElem{tag: tag}
	if len(a) >= 2 {
		if d, ok := a[1].(*object.Dict); ok {
			ks, vs := d.Items()
			for idx2, k := range ks {
				if ks2, ok2 := k.(*object.Str); ok2 {
					val := ""
					if vs2, ok3 := vs[idx2].(*object.Str); ok3 {
						val = vs2.V
					}
					st.attrib = append(st.attrib, etAttr{ks2.V, val})
				}
			}
		}
	}
	if kw != nil {
		ks, vs := kw.Items()
		for idx2, k := range ks {
			if ks2, ok2 := k.(*object.Str); ok2 {
				val := ""
				if vs2, ok3 := vs[idx2].(*object.Str); ok3 {
					val = vs2.V
				}
				st.attrib = append(st.attrib, etAttr{ks2.V, val})
			}
		}
	}
	inst := &object.Instance{Class: cls, Dict: object.NewDict()}
	etElemMap.Store(inst, st)
	etSyncToDict(inst, st)
	return inst, nil
}

// etSyncToDict copies Go state → inst.Dict for tag/text/tail/attrib.
func etSyncToDict(inst *object.Instance, st *etElem) {
	inst.Dict.SetStr("tag", &object.Str{V: st.tag})
	if st.text == "" {
		inst.Dict.SetStr("text", object.None)
	} else {
		inst.Dict.SetStr("text", &object.Str{V: st.text})
	}
	if st.tail == "" {
		inst.Dict.SetStr("tail", object.None)
	} else {
		inst.Dict.SetStr("tail", &object.Str{V: st.tail})
	}
	etSyncAttrDict(inst, st)
}

func etSyncAttrDict(inst *object.Instance, st *etElem) {
	d := object.NewDict()
	for _, attr := range st.attrib {
		d.SetStr(attr.name, &object.Str{V: attr.value})
	}
	inst.Dict.SetStr("attrib", d)
}

// etParseXML parses XML bytes into an Element tree.
func etParseXML(elemCls *object.Class, parseErrCls *object.Class, data []byte) (object.Object, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false

	type stackEntry struct {
		inst *object.Instance
		st   *etElem
	}
	var stack []stackEntry
	var root *object.Instance

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, object.Errorf(parseErrCls, "XML parse error: %v", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			inst := &object.Instance{Class: elemCls, Dict: object.NewDict()}
			st := &etElem{tag: t.Name.Local}
			if t.Name.Space != "" {
				st.tag = "{" + t.Name.Space + "}" + t.Name.Local
			}
			for _, attr := range t.Attr {
				name := attr.Name.Local
				if attr.Name.Space != "" {
					name = "{" + attr.Name.Space + "}" + attr.Name.Local
				}
				st.attrib = append(st.attrib, etAttr{name, attr.Value})
			}
			etElemMap.Store(inst, st)
			etSyncToDict(inst, st)
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.st.children = append(parent.st.children, inst)
			} else {
				root = inst
			}
			stack = append(stack, stackEntry{inst, st})
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			text := string(t)
			if len(stack) > 0 {
				cur := stack[len(stack)-1]
				if cur.st.text == "" {
					cur.st.text = text
				} else {
					cur.st.text += text
				}
				etSyncToDict(cur.inst, cur.st)
			}
		case xml.Comment:
			// Create a comment element and add to parent
			if len(stack) > 0 {
				inst := &object.Instance{Class: elemCls, Dict: object.NewDict()}
				st := &etElem{tag: "<!---->", text: string(t)}
				etElemMap.Store(inst, st)
				etSyncToDict(inst, st)
				parent := stack[len(stack)-1]
				parent.st.children = append(parent.st.children, inst)
			}
		case xml.ProcInst:
			if len(stack) > 0 {
				inst := &object.Instance{Class: elemCls, Dict: object.NewDict()}
				st := &etElem{tag: "<??>", text: t.Target, tail: string(t.Inst)}
				etElemMap.Store(inst, st)
				etSyncToDict(inst, st)
				parent := stack[len(stack)-1]
				parent.st.children = append(parent.st.children, inst)
			}
		}
	}

	if root == nil {
		return nil, object.Errorf(parseErrCls, "no element found")
	}
	return root, nil
}

// etSerialize serializes an element tree to a string builder.
func etSerialize(buf *strings.Builder, inst *object.Instance, method string, shortEmpty bool) {
	st := getEtElem(inst)
	if st == nil {
		return
	}
	// sync text/tail from dict in case user set them directly
	if tv, ok := inst.Dict.GetStr("text"); ok {
		if s, ok := tv.(*object.Str); ok {
			st.text = s.V
		} else if tv == object.None {
			st.text = ""
		}
	}
	if tv, ok := inst.Dict.GetStr("tail"); ok {
		if s, ok := tv.(*object.Str); ok {
			st.tail = s.V
		} else if tv == object.None {
			st.tail = ""
		}
	}
	if tv, ok := inst.Dict.GetStr("tag"); ok {
		if s, ok := tv.(*object.Str); ok {
			st.tag = s.V
		}
	}

	if method == "text" {
		if st.text != "" {
			buf.WriteString(st.text)
		}
		for _, child := range st.children {
			etSerialize(buf, child, method, shortEmpty)
		}
		if st.tail != "" {
			buf.WriteString(st.tail)
		}
		return
	}

	// Open tag
	buf.WriteByte('<')
	buf.WriteString(st.tag)
	for _, attr := range st.attrib {
		buf.WriteByte(' ')
		buf.WriteString(attr.name)
		buf.WriteString(`="`)
		buf.WriteString(xmlEscapeAttr(attr.value))
		buf.WriteByte('"')
	}

	if shortEmpty && len(st.children) == 0 && st.text == "" && !isHTMLVoidElem(st.tag) {
		buf.WriteString(" />")
	} else {
		buf.WriteByte('>')
		if st.text != "" {
			buf.WriteString(xmlEscapeText(st.text))
		}
		for _, child := range st.children {
			etSerialize(buf, child, method, shortEmpty)
		}
		buf.WriteString("</")
		buf.WriteString(st.tag)
		buf.WriteByte('>')
	}
	if st.tail != "" {
		buf.WriteString(xmlEscapeText(st.tail))
	}
}

func xmlEscapeText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func xmlEscapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

var htmlVoidElems = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

func isHTMLVoidElem(tag string) bool {
	return htmlVoidElems[strings.ToLower(tag)]
}

// etIndent adds newlines and indentation in-place.
func etIndent(inst *object.Instance, space string, level int, isRoot bool) {
	st := getEtElem(inst)
	if st == nil {
		return
	}
	indent := strings.Repeat(space, level)
	childIndent := strings.Repeat(space, level+1)
	if len(st.children) > 0 {
		st.text = "\n" + childIndent
		etSyncToDict(inst, st)
		for idx2, child := range st.children {
			etIndent(child, space, level+1, false)
			childSt := getEtElem(child)
			if childSt != nil {
				if idx2 < len(st.children)-1 {
					childSt.tail = "\n" + childIndent
				} else {
					childSt.tail = "\n" + indent
				}
				etSyncToDict(child, childSt)
			}
		}
	}
}

// etIterDepth does depth-first iteration, collecting elements.
func etIterDepth(inst *object.Instance, tag string, results *[]*object.Instance) {
	st := getEtElem(inst)
	if st == nil {
		return
	}
	if tag == "" || tag == "*" || st.tag == tag {
		*results = append(*results, inst)
	}
	for _, child := range st.children {
		etIterDepth(child, tag, results)
	}
}

// etIterText collects text/tail recursively.
func etIterText(inst *object.Instance, withTail bool, texts *[]string) {
	st := getEtElem(inst)
	if st == nil {
		return
	}
	if st.text != "" {
		*texts = append(*texts, st.text)
	}
	for _, child := range st.children {
		etIterText(child, withTail, texts)
	}
	if withTail && st.tail != "" {
		*texts = append(*texts, st.tail)
	}
}

// etCollectIDs finds all elements with an 'id' attribute.
func etCollectIDs(inst *object.Instance, d *object.Dict) {
	st := getEtElem(inst)
	if st == nil {
		return
	}
	for _, attr := range st.attrib {
		if attr.name == "id" {
			d.SetStr(attr.value, inst)
			break
		}
	}
	for _, child := range st.children {
		etCollectIDs(child, d)
	}
}

// etCollectIterparse builds (event, element) tuples for iterparse.
func etCollectIterparse(inst *object.Instance, events *[]object.Object) {
	st := getEtElem(inst)
	if st == nil {
		return
	}
	startEvt := &object.Tuple{V: []object.Object{&object.Str{V: "start"}, inst}}
	*events = append(*events, startEvt)
	for _, child := range st.children {
		etCollectIterparse(child, events)
	}
	endEvt := &object.Tuple{V: []object.Object{&object.Str{V: "end"}, inst}}
	*events = append(*events, endEvt)
}

// ── XPath subset: find / findall ─────────────────────────────────────────────

func etFind(inst *object.Instance, pattern string) *object.Instance {
	results := etFindAll(inst, pattern)
	if len(results) == 0 {
		return nil
	}
	return results[0]
}

func etFindAll(inst *object.Instance, pattern string) []*object.Instance {
	st := getEtElem(inst)
	if st == nil {
		return nil
	}
	// Handle leading ./ or ./
	if pattern == "." {
		return []*object.Instance{inst}
	}
	if strings.HasPrefix(pattern, "./") {
		pattern = pattern[2:]
	}
	// Handle // (search anywhere in subtree)
	if strings.HasPrefix(pattern, "//") {
		tag := pattern[2:]
		var results []*object.Instance
		etFindAnywhere(inst, tag, &results)
		return results
	}
	if strings.Contains(pattern, "//") {
		parts := strings.SplitN(pattern, "//", 2)
		head := parts[0]
		tail := "//" + parts[1]
		intermediates := etFindAll(inst, head)
		var results []*object.Instance
		for _, m := range intermediates {
			results = append(results, etFindAll(m, tail)...)
		}
		return results
	}
	// Split on /
	parts := strings.SplitN(pattern, "/", 2)
	head := parts[0]
	if len(parts) == 1 {
		// Leaf: match direct children
		return etMatchChildren(st, head)
	}
	// Recurse
	var results []*object.Instance
	matched := etMatchChildren(st, head)
	for _, m := range matched {
		results = append(results, etFindAll(m, parts[1])...)
	}
	return results
}

func etFindAnywhere(inst *object.Instance, tag string, results *[]*object.Instance) {
	st := getEtElem(inst)
	if st == nil {
		return
	}
	for _, child := range st.children {
		childSt := getEtElem(child)
		if childSt == nil {
			continue
		}
		if tag == "*" || childSt.tag == tag {
			*results = append(*results, child)
		}
		etFindAnywhere(child, tag, results)
	}
}

// etMatchChildren matches direct children of st against a single path step.
// Step can be: tag, *, tag[@attr], tag[@attr='val'], tag[N], [@attr], etc.
func etMatchChildren(st *etElem, step string) []*object.Instance {
	// Parse predicate
	tag := step
	predicate := ""
	if idx2 := strings.Index(step, "["); idx2 >= 0 {
		tag = step[:idx2]
		predicate = step[idx2:]
	}
	var results []*object.Instance
	for idx2, child := range st.children {
		childSt := getEtElem(child)
		if childSt == nil {
			continue
		}
		if tag == "" || tag == "*" || childSt.tag == tag {
			if etMatchPredicate(childSt, child, predicate, idx2, st.children) {
				results = append(results, child)
			}
		}
	}
	return results
}

func etMatchPredicate(st *etElem, inst *object.Instance, predicate string, idx2 int, siblings []*object.Instance) bool {
	if predicate == "" {
		return true
	}
	// strip outer []
	inner := strings.TrimSuffix(strings.TrimPrefix(predicate, "["), "]")
	// [@attr='val']
	if strings.HasPrefix(inner, "@") {
		attrPart := inner[1:]
		if eqIdx := strings.Index(attrPart, "="); eqIdx >= 0 {
			attrName := attrPart[:eqIdx]
			attrVal := strings.Trim(attrPart[eqIdx+1:], "'\"")
			for _, a := range st.attrib {
				if a.name == attrName && a.value == attrVal {
					return true
				}
			}
			return false
		}
		// [@attr] - just check existence
		for _, a := range st.attrib {
			if a.name == attrPart {
				return true
			}
		}
		return false
	}
	// [N] - 1-based index
	if n, err := strconv.Atoi(inner); err == nil {
		return idx2+1 == n
	}
	// [tag] - has child with this tag
	for _, child := range st.children {
		childSt := getEtElem(child)
		if childSt != nil && childSt.tag == inner {
			return true
		}
	}
	return false
}

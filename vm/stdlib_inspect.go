package vm

import (
	"fmt"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) buildInspect() *object.Module {
	m := &object.Module{Name: "inspect", Dict: object.NewDict()}

	// Type-check helpers
	isModule := func(obj object.Object) bool {
		_, ok := obj.(*object.Module)
		return ok
	}
	isClass := func(obj object.Object) bool {
		_, ok := obj.(*object.Class)
		return ok
	}
	isFunction := func(obj object.Object) bool {
		_, ok := obj.(*object.Function)
		return ok
	}
	isBuiltin := func(obj object.Object) bool {
		_, ok := obj.(*object.BuiltinFunc)
		return ok
	}
	isMethod := func(obj object.Object) bool {
		_, ok := obj.(*object.BoundMethod)
		return ok
	}
	isRoutine := func(obj object.Object) bool {
		return isFunction(obj) || isBuiltin(obj) || isMethod(obj)
	}
	isCallable := func(obj object.Object) bool {
		switch obj.(type) {
		case *object.Function, *object.BuiltinFunc, *object.BoundMethod, *object.Class:
			return true
		case *object.Instance:
			inst := obj.(*object.Instance)
			if _, ok := inst.Dict.GetStr("__call__"); ok {
				return true
			}
			if inst.Class != nil {
				if _, ok := inst.Class.Dict.GetStr("__call__"); ok {
					return true
				}
			}
			return false
		}
		return false
	}

	boolFn := func(name string, pred func(object.Object) bool) *object.BuiltinFunc {
		return &object.BuiltinFunc{
			Name: name,
			Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
				if len(a) < 1 {
					return object.False, nil
				}
				return object.BoolOf(pred(a[0])), nil
			},
		}
	}

	m.Dict.SetStr("ismodule", boolFn("ismodule", isModule))
	m.Dict.SetStr("isclass", boolFn("isclass", isClass))
	m.Dict.SetStr("isfunction", boolFn("isfunction", isFunction))
	m.Dict.SetStr("isbuiltin", boolFn("isbuiltin", isBuiltin))
	m.Dict.SetStr("ismethod", boolFn("ismethod", isMethod))
	m.Dict.SetStr("isroutine", boolFn("isroutine", isRoutine))
	m.Dict.SetStr("callable", boolFn("callable", isCallable))
	m.Dict.SetStr("isframe", boolFn("isframe", func(_ object.Object) bool { return false }))
	m.Dict.SetStr("iscode", boolFn("iscode", func(o object.Object) bool {
		_, ok := o.(*object.Code)
		return ok
	}))
	m.Dict.SetStr("ismethodwrapper", boolFn("ismethodwrapper", func(_ object.Object) bool { return false }))
	m.Dict.SetStr("ismethoddescriptor", boolFn("ismethoddescriptor", func(_ object.Object) bool { return false }))
	m.Dict.SetStr("isdatadescriptor", boolFn("isdatadescriptor", func(_ object.Object) bool { return false }))
	m.Dict.SetStr("isgetsetdescriptor", boolFn("isgetsetdescriptor", func(_ object.Object) bool { return false }))
	m.Dict.SetStr("ismemberdescriptor", boolFn("ismemberdescriptor", func(_ object.Object) bool { return false }))
	m.Dict.SetStr("isabstract", boolFn("isabstract", func(_ object.Object) bool { return false }))
	m.Dict.SetStr("isasyncgenfunction", boolFn("isasyncgenfunction", func(o object.Object) bool {
		if fn, ok := o.(*object.Function); ok {
			return fn.Code != nil && fn.Code.Flags&0x200 != 0
		}
		return false
	}))
	m.Dict.SetStr("iscoroutinefunction", boolFn("iscoroutinefunction", func(o object.Object) bool {
		if fn, ok := o.(*object.Function); ok {
			return fn.Code != nil && fn.Code.Flags&0x80 != 0 // CO_COROUTINE = 128
		}
		return false
	}))
	m.Dict.SetStr("isgeneratorfunction", boolFn("isgeneratorfunction", func(o object.Object) bool {
		if fn, ok := o.(*object.Function); ok {
			return fn.Code != nil && fn.Code.Flags&0x20 != 0
		}
		return false
	}))
	m.Dict.SetStr("isgenerator", boolFn("isgenerator", func(o object.Object) bool {
		_, ok := o.(*object.Generator)
		return ok
	}))
	m.Dict.SetStr("iscoroutine", boolFn("iscoroutine", func(_ object.Object) bool { return false }))
	m.Dict.SetStr("isawaitable", boolFn("isawaitable", func(_ object.Object) bool { return false }))
	m.Dict.SetStr("isasyncgen", boolFn("isasyncgen", func(_ object.Object) bool { return false }))

	// getmembers helper shared between getmembers and getmembers_static
	collectMembers := func(obj object.Object, predicate object.Object) []object.Object {
		var d *object.Dict
		switch v := obj.(type) {
		case *object.Module:
			d = v.Dict
		case *object.Class:
			d = v.Dict
		case *object.Instance:
			d = v.Class.Dict
		default:
			return nil
		}
		items := []object.Object{}
		ks, vs := d.Items()
		for idx, k := range ks {
			val := vs[idx]
			if predicate != nil {
				switch p := predicate.(type) {
				case *object.BuiltinFunc:
					res, _ := p.Call(nil, []object.Object{val}, nil)
					if res == object.False || res == object.None {
						continue
					}
				}
			}
			pair := &object.Tuple{V: []object.Object{k, val}}
			items = append(items, pair)
		}
		return items
	}

	m.Dict.SetStr("getmembers", &object.BuiltinFunc{
		Name: "getmembers",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{V: []object.Object{}}, nil
			}
			var pred object.Object
			if len(a) >= 2 {
				pred = a[1]
			}
			items := collectMembers(a[0], pred)
			if items == nil {
				items = []object.Object{}
			}
			return &object.List{V: items}, nil
		},
	})

	m.Dict.SetStr("getmembers_static", &object.BuiltinFunc{
		Name: "getmembers_static",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{V: []object.Object{}}, nil
			}
			var pred object.Object
			if len(a) >= 2 {
				pred = a[1]
			}
			items := collectMembers(a[0], pred)
			if items == nil {
				items = []object.Object{}
			}
			return &object.List{V: items}, nil
		},
	})

	m.Dict.SetStr("getdoc", &object.BuiltinFunc{
		Name: "getdoc",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			if fn, ok := a[0].(*object.Function); ok {
				if fn.Code != nil && len(fn.Code.Consts) > 0 {
					if doc, ok2 := fn.Code.Consts[0].(*object.Str); ok2 {
						return &object.Str{V: strings.TrimSpace(doc.V)}, nil
					}
				}
			}
			return object.None, nil
		},
	})

	m.Dict.SetStr("getfile", &object.BuiltinFunc{
		Name: "getfile",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			switch v := a[0].(type) {
			case *object.Function:
				if v.Code != nil {
					return &object.Str{V: v.Code.Filename}, nil
				}
			case *object.Module:
				if fname, ok := v.Dict.GetStr("__file__"); ok {
					return fname, nil
				}
			}
			return object.None, nil
		},
	})

	m.Dict.SetStr("getsourcefile", &object.BuiltinFunc{
		Name: "getsourcefile",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			switch v := a[0].(type) {
			case *object.Function:
				if v.Code != nil {
					fname := v.Code.Filename
					if strings.HasSuffix(fname, ".pyc") {
						fname = fname[:len(fname)-1] // .pyc → .py
					}
					return &object.Str{V: fname}, nil
				}
			}
			return object.None, nil
		},
	})

	m.Dict.SetStr("getsource", &object.BuiltinFunc{
		Name: "getsource",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.typeErr, "source not available")
		},
	})

	m.Dict.SetStr("getsourcelines", &object.BuiltinFunc{
		Name: "getsourcelines",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return nil, object.Errorf(i.typeErr, "source not available")
		},
	})

	m.Dict.SetStr("getcomments", &object.BuiltinFunc{
		Name: "getcomments",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("getmodule", &object.BuiltinFunc{
		Name: "getmodule",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("getmodulename", &object.BuiltinFunc{
		Name: "getmodulename",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			if s, ok := a[0].(*object.Str); ok {
				path := s.V
				base := path
				if idx := strings.LastIndex(path, "/"); idx >= 0 {
					base = path[idx+1:]
				}
				if strings.HasSuffix(base, ".py") {
					return &object.Str{V: base[:len(base)-3]}, nil
				}
				if strings.HasSuffix(base, ".pyc") {
					return &object.Str{V: base[:len(base)-4]}, nil
				}
			}
			return object.None, nil
		},
	})

	m.Dict.SetStr("ispackage", &object.BuiltinFunc{
		Name: "ispackage",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.False, nil
		},
	})

	m.Dict.SetStr("walktree", &object.BuiltinFunc{
		Name: "walktree",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	m.Dict.SetStr("getannotations", &object.BuiltinFunc{
		Name: "getannotations",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewDict(), nil
		},
	})

	m.Dict.SetStr("get_annotations", &object.BuiltinFunc{
		Name: "get_annotations",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewDict(), nil
		},
	})

	m.Dict.SetStr("formatannotation", &object.BuiltinFunc{
		Name: "formatannotation",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			switch v := a[0].(type) {
			case *object.Class:
				return &object.Str{V: v.Name}, nil
			case *object.Str:
				// string annotations → repr (quoted)
				return &object.Str{V: "'" + v.V + "'"}, nil
			}
			return &object.Str{V: fmt.Sprintf("%v", a[0])}, nil
		},
	})

	m.Dict.SetStr("formatargvalues", &object.BuiltinFunc{
		Name: "formatargvalues",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "()"}, nil
		},
	})

	m.Dict.SetStr("cleandoc", &object.BuiltinFunc{
		Name: "cleandoc",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: ""}, nil
			}
			if s, ok := a[0].(*object.Str); ok {
				lines := strings.Split(s.V, "\n")
				// find minimum indentation of non-empty lines after the first
				minIndent := -1
				for _, line := range lines[1:] {
					trimmed := strings.TrimLeft(line, " \t")
					if trimmed == "" {
						continue
					}
					indent := len(line) - len(trimmed)
					if minIndent < 0 || indent < minIndent {
						minIndent = indent
					}
				}
				if minIndent < 0 {
					minIndent = 0
				}
				var result []string
				for idx, line := range lines {
					if idx == 0 {
						result = append(result, strings.TrimSpace(line))
					} else {
						if len(line) >= minIndent {
							result = append(result, line[minIndent:])
						} else {
							result = append(result, strings.TrimLeft(line, " \t"))
						}
					}
				}
				// join and trim leading/trailing blank lines
				joined := strings.Join(result, "\n")
				return &object.Str{V: strings.TrimSpace(joined)}, nil
			}
			return a[0], nil
		},
	})

	// BoundArguments class
	boundArgsCls := &object.Class{Name: "BoundArguments", Dict: object.NewDict()}
	boundArgsCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			args := &object.Tuple{V: []object.Object{}}
			kwargs := object.NewDict()
			arguments := object.NewDict()
			if len(a) >= 2 {
				if t, ok := a[1].(*object.Tuple); ok {
					args = t
				}
			}
			self.Dict.SetStr("args", args)
			self.Dict.SetStr("kwargs", kwargs)
			self.Dict.SetStr("arguments", arguments)
			return object.None, nil
		},
	})
	m.Dict.SetStr("BoundArguments", boundArgsCls)

	// Parameter class
	paramClass := &object.Class{Name: "Parameter", Dict: object.NewDict()}
	paramClass.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			if len(a) >= 2 {
				self.Dict.SetStr("name", a[1])
			}
			if len(a) >= 3 {
				self.Dict.SetStr("kind", a[2])
			}
			self.Dict.SetStr("default", object.None)
			self.Dict.SetStr("annotation", object.None)
			if kw != nil {
				ks, vs := kw.Items()
				for idx, k := range ks {
					if s, ok := k.(*object.Str); ok {
						self.Dict.SetStr(s.V, vs[idx])
					}
				}
			}
			return object.None, nil
		},
	})
	paramClass.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Str{V: "<Parameter>"}, nil
			}
			self := a[0].(*object.Instance)
			name := ""
			if n, ok := self.Dict.GetStr("name"); ok {
				if s, ok2 := n.(*object.Str); ok2 {
					name = s.V
				}
			}
			return &object.Str{V: "<Parameter \"" + name + "\">"}, nil
		},
	})
	// Kind constants
	paramClass.Dict.SetStr("POSITIONAL_ONLY", object.NewInt(0))
	paramClass.Dict.SetStr("POSITIONAL_OR_KEYWORD", object.NewInt(1))
	paramClass.Dict.SetStr("VAR_POSITIONAL", object.NewInt(2))
	paramClass.Dict.SetStr("KEYWORD_ONLY", object.NewInt(3))
	paramClass.Dict.SetStr("VAR_KEYWORD", object.NewInt(4))
	emptyObj := &object.Instance{Class: paramClass, Dict: object.NewDict()}
	emptyObj.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "<class 'inspect._empty'>"}, nil
		},
	})
	paramClass.Dict.SetStr("empty", emptyObj)
	m.Dict.SetStr("Parameter", paramClass)

	// Signature class
	sigClass := &object.Class{Name: "Signature", Dict: object.NewDict()}
	sigClass.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.None, nil
			}
			self := a[0].(*object.Instance)
			params := object.NewDict()
			if len(a) >= 2 {
				if lst, ok := a[1].(*object.List); ok {
					_ = lst
				}
			}
			self.Dict.SetStr("parameters", params)
			self.Dict.SetStr("return_annotation", emptyObj)
			return object.None, nil
		},
	})
	sigClass.Dict.SetStr("__repr__", &object.BuiltinFunc{
		Name: "__repr__",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.Str{V: "()"}, nil
		},
	})
	sigClass.Dict.SetStr("bind", &object.BuiltinFunc{
		Name: "bind",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			ba := &object.Instance{Class: boundArgsCls, Dict: object.NewDict()}
			args := []object.Object{}
			if len(a) > 1 {
				args = a[1:]
			}
			ba.Dict.SetStr("args", &object.Tuple{V: args})
			ba.Dict.SetStr("kwargs", object.NewDict())
			ba.Dict.SetStr("arguments", object.NewDict())
			return ba, nil
		},
	})
	sigClass.Dict.SetStr("bind_partial", &object.BuiltinFunc{
		Name: "bind_partial",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			ba := &object.Instance{Class: boundArgsCls, Dict: object.NewDict()}
			args := []object.Object{}
			if len(a) > 1 {
				args = a[1:]
			}
			ba.Dict.SetStr("args", &object.Tuple{V: args})
			ba.Dict.SetStr("kwargs", object.NewDict())
			ba.Dict.SetStr("arguments", object.NewDict())
			return ba, nil
		},
	})
	sigClass.Dict.SetStr("empty", emptyObj)
	m.Dict.SetStr("Signature", sigClass)

	m.Dict.SetStr("signature", &object.BuiltinFunc{
		Name: "signature",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			inst := &object.Instance{Class: sigClass, Dict: object.NewDict()}
			inst.Dict.SetStr("parameters", object.NewDict())
			inst.Dict.SetStr("return_annotation", emptyObj)
			return inst, nil
		},
	})

	m.Dict.SetStr("getfullargspec", &object.BuiltinFunc{
		Name: "getfullargspec",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			cls := &object.Class{Name: "FullArgSpec", Dict: object.NewDict()}
			inst := &object.Instance{Class: cls, Dict: object.NewDict()}
			inst.Dict.SetStr("args", &object.List{V: []object.Object{}})
			inst.Dict.SetStr("varargs", object.None)
			inst.Dict.SetStr("varkw", object.None)
			inst.Dict.SetStr("defaults", object.None)
			inst.Dict.SetStr("kwonlyargs", &object.List{V: []object.Object{}})
			inst.Dict.SetStr("kwonlydefaults", object.None)
			inst.Dict.SetStr("annotations", object.NewDict())
			return inst, nil
		},
	})

	m.Dict.SetStr("getargvalues", &object.BuiltinFunc{
		Name: "getargvalues",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			cls := &object.Class{Name: "ArgInfo", Dict: object.NewDict()}
			inst := &object.Instance{Class: cls, Dict: object.NewDict()}
			inst.Dict.SetStr("args", &object.List{V: []object.Object{}})
			inst.Dict.SetStr("varargs", object.None)
			inst.Dict.SetStr("keywords", object.None)
			inst.Dict.SetStr("locals", object.NewDict())
			return inst, nil
		},
	})

	m.Dict.SetStr("currentframe", &object.BuiltinFunc{
		Name: "currentframe",
		Call: func(ii any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})

	m.Dict.SetStr("stack", &object.BuiltinFunc{
		Name: "stack",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	m.Dict.SetStr("trace", &object.BuiltinFunc{
		Name: "trace",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	m.Dict.SetStr("getouterframes", &object.BuiltinFunc{
		Name: "getouterframes",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	m.Dict.SetStr("getinnerframes", &object.BuiltinFunc{
		Name: "getinnerframes",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return &object.List{V: []object.Object{}}, nil
		},
	})

	m.Dict.SetStr("getlineno", &object.BuiltinFunc{
		Name: "getlineno",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			return object.NewInt(0), nil
		},
	})

	m.Dict.SetStr("getframeinfo", &object.BuiltinFunc{
		Name: "getframeinfo",
		Call: func(_ any, _ []object.Object, _ *object.Dict) (object.Object, error) {
			cls := &object.Class{Name: "Traceback", Dict: object.NewDict()}
			inst := &object.Instance{Class: cls, Dict: object.NewDict()}
			inst.Dict.SetStr("filename", &object.Str{V: "<unknown>"})
			inst.Dict.SetStr("lineno", object.NewInt(0))
			inst.Dict.SetStr("function", &object.Str{V: "<unknown>"})
			inst.Dict.SetStr("code_context", object.None)
			inst.Dict.SetStr("index", object.None)
			return inst, nil
		},
	})

	// Attribute namedtuple-like class
	attributeClass := &object.Class{Name: "Attribute", Dict: object.NewDict()}
	attributeClass.Dict.SetStr("_fields", &object.Tuple{V: []object.Object{
		&object.Str{V: "name"},
		&object.Str{V: "kind"},
		&object.Str{V: "defining_class"},
		&object.Str{V: "object"},
	}})
	m.Dict.SetStr("Attribute", attributeClass)

	m.Dict.SetStr("classify_class_attrs", &object.BuiltinFunc{
		Name: "classify_class_attrs",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.List{V: []object.Object{}}, nil
			}
			var d *object.Dict
			var cls *object.Class
			switch v := a[0].(type) {
			case *object.Class:
				d = v.Dict
				cls = v
			case *object.Instance:
				if v.Class != nil {
					d = v.Class.Dict
					cls = v.Class
				}
			}
			if d == nil {
				return &object.List{V: []object.Object{}}, nil
			}
			items := []object.Object{}
			ks, vs := d.Items()
			for idx, k := range ks {
				val := vs[idx]
				kind := "data"
				if _, ok := val.(*object.Function); ok {
					kind = "method"
				}
				if _, ok := val.(*object.BuiltinFunc); ok {
					kind = "method"
				}
				attr := &object.Instance{Class: attributeClass, Dict: object.NewDict()}
				attr.Dict.SetStr("name", k)
				attr.Dict.SetStr("kind", &object.Str{V: kind})
				attr.Dict.SetStr("defining_class", cls)
				attr.Dict.SetStr("object", val)
				items = append(items, attr)
			}
			return &object.List{V: items}, nil
		},
	})

	m.Dict.SetStr("getattr_static", &object.BuiltinFunc{
		Name: "getattr_static",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return object.None, nil
			}
			name := ""
			if s, ok := a[1].(*object.Str); ok {
				name = s.V
			}
			switch v := a[0].(type) {
			case *object.Instance:
				if val, ok := v.Dict.GetStr(name); ok {
					return val, nil
				}
				if v.Class != nil {
					if val, ok := v.Class.Dict.GetStr(name); ok {
						return val, nil
					}
				}
			case *object.Class:
				if val, ok := v.Dict.GetStr(name); ok {
					return val, nil
				}
			case *object.Module:
				if val, ok := v.Dict.GetStr(name); ok {
					return val, nil
				}
			}
			if len(a) >= 3 {
				return a[2], nil
			}
			return nil, object.Errorf(i.attrErr, "'%v' has no attribute '%s'", a[0], name)
		},
	})

	m.Dict.SetStr("getmro", &object.BuiltinFunc{
		Name: "getmro",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return &object.Tuple{V: []object.Object{}}, nil
			}
			if cls, ok := a[0].(*object.Class); ok {
				var items []object.Object
				seen := map[*object.Class]bool{}
				var collect func(c *object.Class)
				collect = func(c *object.Class) {
					if seen[c] {
						return
					}
					seen[c] = true
					items = append(items, c)
					for _, base := range c.Bases {
						collect(base)
					}
				}
				collect(cls)
				return &object.Tuple{V: items}, nil
			}
			return &object.Tuple{V: []object.Object{}}, nil
		},
	})

	// FrameInfo namedtuple-like class
	frameInfoClass := &object.Class{Name: "FrameInfo", Dict: object.NewDict()}
	frameInfoClass.Dict.SetStr("_fields", &object.Tuple{V: []object.Object{
		&object.Str{V: "frame"},
		&object.Str{V: "filename"},
		&object.Str{V: "lineno"},
		&object.Str{V: "function"},
		&object.Str{V: "code_context"},
		&object.Str{V: "index"},
	}})
	m.Dict.SetStr("FrameInfo", frameInfoClass)

	m.Dict.SetStr("BlockFinder", &object.Class{Name: "BlockFinder", Dict: object.NewDict()})

	m.Dict.SetStr("indentsize", &object.BuiltinFunc{
		Name: "indentsize",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return object.NewInt(0), nil
			}
			if s, ok := a[0].(*object.Str); ok {
				expanded := expandTabs8(s.V)
				count := 0
				for _, c := range expanded {
					if c == ' ' {
						count++
					} else {
						break
					}
				}
				return object.NewInt(int64(count)), nil
			}
			return object.NewInt(0), nil
		},
	})

	// CO_* constants
	m.Dict.SetStr("CO_OPTIMIZED", object.NewInt(1))
	m.Dict.SetStr("CO_NEWLOCALS", object.NewInt(2))
	m.Dict.SetStr("CO_VARARGS", object.NewInt(4))
	m.Dict.SetStr("CO_VARKEYWORDS", object.NewInt(8))
	m.Dict.SetStr("CO_NESTED", object.NewInt(16))
	m.Dict.SetStr("CO_GENERATOR", object.NewInt(32))
	m.Dict.SetStr("CO_NOFREE", object.NewInt(64))
	m.Dict.SetStr("CO_COROUTINE", object.NewInt(128))
	m.Dict.SetStr("CO_ITERABLE_COROUTINE", object.NewInt(256))
	m.Dict.SetStr("CO_ASYNC_GENERATOR", object.NewInt(512))
	m.Dict.SetStr("CO_HAS_DOCSTRING", object.NewInt(67108864))
	m.Dict.SetStr("CO_METHOD", object.NewInt(134217728))

	m.Dict.SetStr("TPFLAGS_IS_ABSTRACT", object.NewInt(1048576))

	return m
}

// expandTabs8 replaces tabs with spaces (tab stop every 8 columns), matching
// Python's str.expandtabs(8) for indentation counting purposes.
func expandTabs8(s string) string {
	var b strings.Builder
	col := 0
	for _, c := range s {
		if c == '\t' {
			spaces := 8 - (col % 8)
			for k := 0; k < spaces; k++ {
				b.WriteByte(' ')
			}
			col += spaces
		} else {
			b.WriteRune(c)
			col++
		}
	}
	return b.String()
}

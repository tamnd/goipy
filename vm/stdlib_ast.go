package vm

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/tamnd/goipy/object"
)

// buildAst constructs the ast module with the CPython 3.14 API surface.
// ast.parse() returns a stub Module node; literal_eval() is fully implemented.
func (i *Interp) buildAst() *object.Module {
	m := &object.Module{Name: "ast", Dict: object.NewDict()}

	// ── PyCF constants ────────────────────────────────────────────────────
	m.Dict.SetStr("PyCF_ONLY_AST", object.NewInt(1024))
	m.Dict.SetStr("PyCF_ALLOW_TOP_LEVEL_AWAIT", object.NewInt(8192))
	m.Dict.SetStr("PyCF_TYPE_COMMENTS", object.NewInt(4096))
	m.Dict.SetStr("PyCF_OPTIMIZED_AST", object.NewInt(33792))

	// ── Node class factory ────────────────────────────────────────────────

	// makeNode creates an AST node class with _fields, _attributes, __init__.
	// Positional init args are assigned by _fields order; keyword args by name.
	var makeNode func(name string, fields []string, bases ...*object.Class) *object.Class
	makeNode = func(name string, fields []string, bases ...*object.Class) *object.Class {
		cls := &object.Class{Name: name, Dict: object.NewDict()}
		if len(bases) > 0 {
			cls.Bases = bases
		}
		fieldObjs := make([]object.Object, len(fields))
		for j, f := range fields {
			fieldObjs[j] = &object.Str{V: f}
		}
		cls.Dict.SetStr("_fields", &object.Tuple{V: fieldObjs})
		cls.Dict.SetStr("_attributes", &object.Tuple{V: []object.Object{
			&object.Str{V: "lineno"},
			&object.Str{V: "col_offset"},
			&object.Str{V: "end_lineno"},
			&object.Str{V: "end_col_offset"},
		}})
		cls.Dict.SetStr("__init__", &object.BuiltinFunc{
			Name: "__init__",
			Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
				if len(a) < 1 {
					return object.None, nil
				}
				self := a[0].(*object.Instance)
				for idx, arg := range a[1:] {
					if idx < len(fields) {
						self.Dict.SetStr(fields[idx], arg)
					}
				}
				if kw != nil {
					ks, vs := kw.Items()
					for j, k := range ks {
						if ks2, ok := k.(*object.Str); ok {
							self.Dict.SetStr(ks2.V, vs[j])
						}
					}
				}
				return object.None, nil
			},
		})
		return cls
	}

	// ── Class hierarchy ───────────────────────────────────────────────────

	astCls := makeNode("AST", nil)
	m.Dict.SetStr("AST", astCls)

	// abstract categories
	modCls := makeNode("mod", nil, astCls)
	m.Dict.SetStr("mod", modCls)
	stmtCls := makeNode("stmt", nil, astCls)
	m.Dict.SetStr("stmt", stmtCls)
	exprCls := makeNode("expr", nil, astCls)
	m.Dict.SetStr("expr", exprCls)
	exprCtxCls := makeNode("expr_context", nil, astCls)
	m.Dict.SetStr("expr_context", exprCtxCls)
	boolopCls := makeNode("boolop", nil, astCls)
	m.Dict.SetStr("boolop", boolopCls)
	operatorCls := makeNode("operator", nil, astCls)
	m.Dict.SetStr("operator", operatorCls)
	unaryopCls := makeNode("unaryop", nil, astCls)
	m.Dict.SetStr("unaryop", unaryopCls)
	cmopopCls := makeNode("cmpop", nil, astCls)
	m.Dict.SetStr("cmpop", cmopopCls)
	patternCls := makeNode("pattern", nil, astCls)
	m.Dict.SetStr("pattern", patternCls)
	excepthandlerCls := makeNode("excepthandler", nil, astCls)
	m.Dict.SetStr("excepthandler", excepthandlerCls)
	typeignoreCls := makeNode("type_ignore", nil, astCls)
	m.Dict.SetStr("type_ignore", typeignoreCls)

	// mod subclasses
	moduleCls := makeNode("Module", []string{"body", "type_ignores"}, modCls)
	m.Dict.SetStr("Module", moduleCls)
	interactiveCls := makeNode("Interactive", []string{"body"}, modCls)
	m.Dict.SetStr("Interactive", interactiveCls)
	expressionCls := makeNode("Expression", []string{"body"}, modCls)
	m.Dict.SetStr("Expression", expressionCls)
	funcTypeCls := makeNode("FunctionType", []string{"argtypes", "returns"}, modCls)
	m.Dict.SetStr("FunctionType", funcTypeCls)

	// stmt subclasses
	for _, spec := range []struct {
		name   string
		fields []string
	}{
		{"FunctionDef", []string{"name", "args", "body", "decorator_list", "returns", "type_comment"}},
		{"AsyncFunctionDef", []string{"name", "args", "body", "decorator_list", "returns", "type_comment"}},
		{"ClassDef", []string{"name", "bases", "keywords", "body", "decorator_list"}},
		{"Return", []string{"value"}},
		{"Delete", []string{"targets"}},
		{"Assign", []string{"targets", "value", "type_comment"}},
		{"AugAssign", []string{"target", "op", "value"}},
		{"AnnAssign", []string{"target", "annotation", "value", "simple"}},
		{"For", []string{"target", "iter", "body", "orelse", "type_comment"}},
		{"AsyncFor", []string{"target", "iter", "body", "orelse", "type_comment"}},
		{"While", []string{"test", "body", "orelse"}},
		{"If", []string{"test", "body", "orelse"}},
		{"With", []string{"items", "body", "type_comment"}},
		{"AsyncWith", []string{"items", "body", "type_comment"}},
		{"Match", []string{"subject", "cases"}},
		{"Raise", []string{"exc", "cause"}},
		{"Try", []string{"body", "handlers", "orelse", "finalbody"}},
		{"TryStar", []string{"body", "handlers", "orelse", "finalbody"}},
		{"Assert", []string{"test", "msg"}},
		{"Import", []string{"names"}},
		{"ImportFrom", []string{"module", "names", "level"}},
		{"Global", []string{"names"}},
		{"Nonlocal", []string{"names"}},
		{"Expr", []string{"value"}},
		{"Pass", nil},
		{"Break", nil},
		{"Continue", nil},
	} {
		cls := makeNode(spec.name, spec.fields, stmtCls)
		m.Dict.SetStr(spec.name, cls)
	}

	// expr subclasses
	constCls := makeNode("Constant", []string{"value", "kind"}, exprCls)
	m.Dict.SetStr("Constant", constCls)

	for _, spec := range []struct {
		name   string
		fields []string
	}{
		{"BoolOp", []string{"op", "values"}},
		{"NamedExpr", []string{"target", "value"}},
		{"BinOp", []string{"left", "op", "right"}},
		{"UnaryOp", []string{"op", "operand"}},
		{"Lambda", []string{"args", "body"}},
		{"IfExp", []string{"test", "body", "orelse"}},
		{"Dict", []string{"keys", "values"}},
		{"Set", []string{"elts"}},
		{"ListComp", []string{"elt", "generators"}},
		{"SetComp", []string{"elt", "generators"}},
		{"DictComp", []string{"key", "value", "generators"}},
		{"GeneratorExp", []string{"elt", "generators"}},
		{"Await", []string{"value"}},
		{"Yield", []string{"value"}},
		{"YieldFrom", []string{"value"}},
		{"Compare", []string{"left", "ops", "comparators"}},
		{"Call", []string{"func", "args", "keywords"}},
		{"FormattedValue", []string{"value", "conversion", "format_spec"}},
		{"JoinedStr", []string{"values"}},
		{"Attribute", []string{"value", "attr", "ctx"}},
		{"Subscript", []string{"value", "slice", "ctx"}},
		{"Starred", []string{"value", "ctx"}},
		{"Name", []string{"id", "ctx"}},
		{"List", []string{"elts", "ctx"}},
		{"Tuple", []string{"elts", "ctx"}},
		{"Slice", []string{"lower", "upper", "step"}},
	} {
		cls := makeNode(spec.name, spec.fields, exprCls)
		m.Dict.SetStr(spec.name, cls)
	}

	// expr_context singletons
	for _, name := range []string{"Load", "Store", "Del"} {
		cls := makeNode(name, nil, exprCtxCls)
		m.Dict.SetStr(name, cls)
	}

	// boolop singletons
	for _, name := range []string{"And", "Or"} {
		cls := makeNode(name, nil, boolopCls)
		m.Dict.SetStr(name, cls)
	}

	// operator singletons
	for _, name := range []string{
		"Add", "Sub", "Mult", "MatMult", "Div", "Mod", "Pow",
		"LShift", "RShift", "BitOr", "BitXor", "BitAnd", "FloorDiv",
	} {
		cls := makeNode(name, nil, operatorCls)
		m.Dict.SetStr(name, cls)
	}

	// unaryop singletons
	for _, name := range []string{"Invert", "Not", "UAdd", "USub"} {
		cls := makeNode(name, nil, unaryopCls)
		m.Dict.SetStr(name, cls)
	}

	// cmpop singletons
	for _, name := range []string{"Eq", "NotEq", "Lt", "LtE", "Gt", "GtE", "Is", "IsNot", "In", "NotIn"} {
		cls := makeNode(name, nil, cmopopCls)
		m.Dict.SetStr(name, cls)
	}

	// misc node classes
	miscNodes := []struct {
		name   string
		fields []string
		base   *object.Class
	}{
		{"comprehension", []string{"target", "iter", "ifs", "is_async"}, astCls},
		{"ExceptHandler", []string{"type", "name", "body"}, excepthandlerCls},
		{"arguments", []string{"posonlyargs", "args", "vararg", "kwonlyargs", "kw_defaults", "kwarg", "defaults"}, astCls},
		{"arg", []string{"arg", "annotation", "type_comment"}, astCls},
		{"keyword", []string{"arg", "value"}, astCls},
		{"alias", []string{"name", "asname"}, astCls},
		{"withitem", []string{"context_expr", "optional_vars"}, astCls},
		{"match_case", []string{"pattern", "guard", "body"}, astCls},
		{"TypeIgnore", []string{"lineno", "tag"}, typeignoreCls},
	}
	for _, spec := range miscNodes {
		cls := makeNode(spec.name, spec.fields, spec.base)
		m.Dict.SetStr(spec.name, cls)
	}

	// pattern subclasses
	for _, spec := range []struct {
		name   string
		fields []string
	}{
		{"MatchValue", []string{"value"}},
		{"MatchSingleton", []string{"value"}},
		{"MatchSequence", []string{"patterns"}},
		{"MatchMapping", []string{"keys", "patterns", "rest"}},
		{"MatchClass", []string{"cls", "patterns", "kwd_attrs", "kwd_patterns"}},
		{"MatchStar", []string{"name"}},
		{"MatchAs", []string{"pattern", "name"}},
		{"MatchOr", []string{"patterns"}},
	} {
		cls := makeNode(spec.name, spec.fields, patternCls)
		m.Dict.SetStr(spec.name, cls)
	}

	// ── iter_fields ───────────────────────────────────────────────────────

	iterFieldsFn := &object.BuiltinFunc{
		Name: "iter_fields",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {

			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "iter_fields() missing argument")
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return emptyIter(), nil
			}
			// Collect _fields from the class
			fieldsRaw, ok2 := inst.Class.Dict.GetStr("_fields")
			if !ok2 {
				return emptyIter(), nil
			}
			var fieldNames []string
			switch fv := fieldsRaw.(type) {
			case *object.Tuple:
				for _, f := range fv.V {
					if s, ok3 := f.(*object.Str); ok3 {
						fieldNames = append(fieldNames, s.V)
					}
				}
			}
			// Build pairs for fields that exist on the instance
			var pairs []object.Object
			for _, fname := range fieldNames {
				val, exists := inst.Dict.GetStr(fname)
				if !exists {
					continue
				}
				pairs = append(pairs, &object.Tuple{V: []object.Object{
					&object.Str{V: fname},
					val,
				}})
			}
			idx := 0
			return &object.Iter{Next: func() (object.Object, bool, error) {
				if idx >= len(pairs) {
					return nil, false, nil
				}
				v := pairs[idx]
				idx++
				return v, true, nil
			}}, nil
		},
	}
	m.Dict.SetStr("iter_fields", iterFieldsFn)

	// ── iter_child_nodes ──────────────────────────────────────────────────

	m.Dict.SetStr("iter_child_nodes", &object.BuiltinFunc{
		Name: "iter_child_nodes",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {

			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "iter_child_nodes() missing argument")
			}
			ii := interp.(*Interp)
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return emptyIter(), nil
			}
			fieldsRaw, ok2 := inst.Class.Dict.GetStr("_fields")
			if !ok2 {
				return emptyIter(), nil
			}
			var fieldNames []string
			if fv, ok3 := fieldsRaw.(*object.Tuple); ok3 {
				for _, f := range fv.V {
					if s, ok4 := f.(*object.Str); ok4 {
						fieldNames = append(fieldNames, s.V)
					}
				}
			}
			var children []object.Object
			for _, fname := range fieldNames {
				val, exists := inst.Dict.GetStr(fname)
				if !exists {
					continue
				}
				switch v := val.(type) {
				case *object.Instance:
					children = append(children, v)
				case *object.List:
					items, err := iterate(ii, v)
					if err != nil {
						continue
					}
					for _, item := range items {
						if inst2, ok5 := item.(*object.Instance); ok5 {
							children = append(children, inst2)
						}
					}
				}
			}
			idx := 0
			return &object.Iter{Next: func() (object.Object, bool, error) {
				if idx >= len(children) {
					return nil, false, nil
				}
				v := children[idx]
				idx++
				return v, true, nil
			}}, nil
		},
	})

	// ── walk ──────────────────────────────────────────────────────────────

	m.Dict.SetStr("walk", &object.BuiltinFunc{
		Name: "walk",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {

			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "walk() missing argument")
			}
			ii := interp.(*Interp)
			icnFn, ok := m.Dict.GetStr("iter_child_nodes")
			if !ok {
				return emptyIter(), nil
			}
			queue := []object.Object{a[0]}
			idx := 0
			var nextNode func() (object.Object, bool, error)
			nextNode = func() (object.Object, bool, error) {
				for idx < len(queue) {
					node := queue[idx]
					idx++
					childIter, err := ii.callObject(icnFn, []object.Object{node}, nil)
					if err == nil {
						children, _ := iterate(ii, childIter)
						queue = append(queue, children...)
					}
					return node, true, nil
				}
				return nil, false, nil
			}
			return &object.Iter{Next: nextNode}, nil
		},
	})

	// ── fix_missing_locations ─────────────────────────────────────────────

	m.Dict.SetStr("fix_missing_locations", &object.BuiltinFunc{
		Name: "fix_missing_locations",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {

			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "fix_missing_locations() missing argument")
			}
			ii := interp.(*Interp)
			walkFn, ok := m.Dict.GetStr("walk")
			if !ok {
				return a[0], nil
			}
			walkIter, err := ii.callObject(walkFn, []object.Object{a[0]}, nil)
			if err != nil {
				return a[0], nil
			}
			nodes, _ := iterate(ii, walkIter)
			for _, node := range nodes {
				inst, ok := node.(*object.Instance)
				if !ok {
					continue
				}
				for _, attr := range []string{"lineno", "col_offset", "end_lineno", "end_col_offset"} {
					if _, exists := inst.Dict.GetStr(attr); !exists {
						inst.Dict.SetStr(attr, object.NewInt(1))
					}
				}
			}
			return a[0], nil
		},
	})

	// ── copy_location ─────────────────────────────────────────────────────

	m.Dict.SetStr("copy_location", &object.BuiltinFunc{
		Name: "copy_location",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {

			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "copy_location() requires 2 args")
			}
			dst, ok1 := a[0].(*object.Instance)
			src, ok2 := a[1].(*object.Instance)
			if !ok1 || !ok2 {
				if len(a) > 0 {
					return a[0], nil
				}
				return object.None, nil
			}
			for _, attr := range []string{"lineno", "col_offset", "end_lineno", "end_col_offset"} {
				if v, exists := src.Dict.GetStr(attr); exists {
					dst.Dict.SetStr(attr, v)
				}
			}
			return dst, nil
		},
	})

	// ── increment_lineno ──────────────────────────────────────────────────

	m.Dict.SetStr("increment_lineno", &object.BuiltinFunc{
		Name: "increment_lineno",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {

			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "increment_lineno() missing argument")
			}
			ii := interp.(*Interp)
			n := int64(1)
			if len(a) >= 2 {
				if nv, ok := a[1].(*object.Int); ok {
					n = nv.Int64()
				}
			}
			walkFn, ok := m.Dict.GetStr("walk")
			if !ok {
				return a[0], nil
			}
			walkIter, err := ii.callObject(walkFn, []object.Object{a[0]}, nil)
			if err != nil {
				return a[0], nil
			}
			nodes, _ := iterate(ii, walkIter)
			for _, node := range nodes {
				inst, ok := node.(*object.Instance)
				if !ok {
					continue
				}
				for _, attr := range []string{"lineno", "end_lineno"} {
					if v, exists := inst.Dict.GetStr(attr); exists {
						if iv, ok2 := v.(*object.Int); ok2 {
							inst.Dict.SetStr(attr, object.NewInt(iv.Int64()+n))
						}
					}
				}
			}
			return a[0], nil
		},
	})

	// ── get_docstring ─────────────────────────────────────────────────────

	m.Dict.SetStr("get_docstring", &object.BuiltinFunc{
		Name: "get_docstring",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {

			if len(a) < 1 {
				return object.None, nil
			}
			inst, ok := a[0].(*object.Instance)
			if !ok {
				return object.None, nil
			}
			bodyVal, exists := inst.Dict.GetStr("body")
			if !exists {
				return object.None, nil
			}
			body, ok2 := bodyVal.(*object.List)
			if !ok2 || len(body.V) == 0 {
				return object.None, nil
			}
			// first statement must be an Expr containing a Constant string
			exprStmt, ok3 := body.V[0].(*object.Instance)
			if !ok3 || exprStmt.Class.Name != "Expr" {
				return object.None, nil
			}
			exprVal, exists2 := exprStmt.Dict.GetStr("value")
			if !exists2 {
				return object.None, nil
			}
			constNode, ok4 := exprVal.(*object.Instance)
			if !ok4 || constNode.Class.Name != "Constant" {
				return object.None, nil
			}
			strVal, exists3 := constNode.Dict.GetStr("value")
			if !exists3 {
				return object.None, nil
			}
			if s, ok5 := strVal.(*object.Str); ok5 {
				return s, nil
			}
			return object.None, nil
		},
	})

	// ── dump ──────────────────────────────────────────────────────────────

	m.Dict.SetStr("dump", &object.BuiltinFunc{
		Name: "dump",
		Call: func(interp any, a []object.Object, kw *object.Dict) (object.Object, error) {

			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "dump() missing argument")
			}
			// annotate_fields defaults to True
			annotate := true
			if kw != nil {
				if af, ok := kw.GetStr("annotate_fields"); ok {
					if b, ok2 := af.(*object.Bool); ok2 {
						annotate = bool(b.V)
					}
				}
			}
			return &object.Str{V: astDumpNode(a[0], annotate)}, nil
		},
	})

	// ── unparse ───────────────────────────────────────────────────────────

	m.Dict.SetStr("unparse", &object.BuiltinFunc{
		Name: "unparse",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {

			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "unparse() missing argument")
			}
			return &object.Str{V: astUnparse(a[0])}, nil
		},
	})

	// ── parse ─────────────────────────────────────────────────────────────

	m.Dict.SetStr("parse", &object.BuiltinFunc{
		Name: "parse",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {

			inst := &object.Instance{Class: moduleCls, Dict: object.NewDict()}
			inst.Dict.SetStr("body", &object.List{V: nil})
			inst.Dict.SetStr("type_ignores", &object.List{V: nil})
			inst.Dict.SetStr("lineno", object.NewInt(1))
			inst.Dict.SetStr("col_offset", object.NewInt(0))
			inst.Dict.SetStr("end_lineno", object.NewInt(1))
			inst.Dict.SetStr("end_col_offset", object.NewInt(0))
			return inst, nil
		},
	})

	// ── compile ───────────────────────────────────────────────────────────

	m.Dict.SetStr("compile", &object.BuiltinFunc{
		Name: "compile",
		Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
			return &object.Code{Name: "<ast.compile>"}, nil
		},
	})

	// ── literal_eval ──────────────────────────────────────────────────────

	m.Dict.SetStr("literal_eval", &object.BuiltinFunc{
		Name: "literal_eval",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {

			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "literal_eval() missing argument")
			}
			switch v := a[0].(type) {
			case *object.Str:
				return parsePyLiteral(v.V)
			case *object.Instance:
				// Called with an AST node — extract the constant value
				if v.Class.Name == "Constant" {
					if val, ok := v.Dict.GetStr("value"); ok {
						return val, nil
					}
				}
				return nil, object.Errorf(i.valueErr, "malformed node or string")
			default:
				return nil, object.Errorf(i.typeErr, "literal_eval() requires a string or AST node")
			}
		},
	})

	// ── NodeVisitor ───────────────────────────────────────────────────────

	nvCls := &object.Class{Name: "NodeVisitor", Dict: object.NewDict()}
	nvCls.Dict.SetStr("__init__", &object.BuiltinFunc{
		Name: "__init__",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			return object.None, nil
		},
	})
	nvCls.Dict.SetStr("visit", &object.BuiltinFunc{
		Name: "visit",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interp.(*Interp)
			if len(a) < 2 {
				return object.None, nil
			}
			self := a[0]
			node := a[1]
			typeName := ""
			if inst, ok := node.(*object.Instance); ok {
				typeName = inst.Class.Name
			}
			if typeName != "" {
				visitFn, err := ii.getAttr(self, "visit_"+typeName)
				if err == nil && visitFn != nil {
					return ii.callObject(visitFn, []object.Object{node}, nil)
				}
			}
			gvFn, err := ii.getAttr(self, "generic_visit")
			if err != nil {
				return object.None, nil
			}
			return ii.callObject(gvFn, []object.Object{node}, nil)
		},
	})
	nvCls.Dict.SetStr("generic_visit", &object.BuiltinFunc{
		Name: "generic_visit",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interp.(*Interp)
			if len(a) < 2 {
				return object.None, nil
			}
			node := a[1]
			icnFn, ok := m.Dict.GetStr("iter_child_nodes")
			if !ok {
				return object.None, nil
			}
			childIter, err := ii.callObject(icnFn, []object.Object{node}, nil)
			if err != nil {
				return object.None, nil
			}
			children, _ := iterate(ii, childIter)
			visitFn, err2 := ii.getAttr(a[0], "visit")
			if err2 != nil {
				return object.None, nil
			}
			for _, child := range children {
				ii.callObject(visitFn, []object.Object{child}, nil) //nolint
			}
			return object.None, nil
		},
	})
	m.Dict.SetStr("NodeVisitor", nvCls)

	// ── NodeTransformer ───────────────────────────────────────────────────

	ntCls := &object.Class{
		Name:  "NodeTransformer",
		Dict:  object.NewDict(),
		Bases: []*object.Class{nvCls},
	}
	ntCls.Dict.SetStr("generic_visit", &object.BuiltinFunc{
		Name: "generic_visit",
		Call: func(interp any, a []object.Object, _ *object.Dict) (object.Object, error) {
			ii := interp.(*Interp)
			if len(a) < 2 {
				return object.None, nil
			}
			node := a[1]
			icnFn, ok := m.Dict.GetStr("iter_child_nodes")
			if !ok {
				return node, nil
			}
			childIter, err := ii.callObject(icnFn, []object.Object{node}, nil)
			if err != nil {
				return node, nil
			}
			children, _ := iterate(ii, childIter)
			visitFn, err2 := ii.getAttr(a[0], "visit")
			if err2 != nil {
				return node, nil
			}
			for _, child := range children {
				ii.callObject(visitFn, []object.Object{child}, nil) //nolint
			}
			return node, nil
		},
	})
	m.Dict.SetStr("NodeTransformer", ntCls)

	return m
}

// emptyIter returns an iterator that yields nothing.
func emptyIter() *object.Iter {
	return &object.Iter{Next: func() (object.Object, bool, error) { return nil, false, nil }}
}

// astDumpNode converts an AST node to its string representation.
func astDumpNode(node object.Object, annotate bool) string {
	inst, ok := node.(*object.Instance)
	if !ok {
		return fmt.Sprintf("%v", node)
	}
	var sb strings.Builder
	sb.WriteString(inst.Class.Name)
	// Get _fields from class
	fieldsRaw, ok2 := inst.Class.Dict.GetStr("_fields")
	if !ok2 {
		return sb.String()
	}
	fv, ok3 := fieldsRaw.(*object.Tuple)
	if !ok3 || len(fv.V) == 0 {
		sb.WriteString("()")
		return sb.String()
	}
	sb.WriteString("(")
	first := true
	for _, fObj := range fv.V {
		fname, ok4 := fObj.(*object.Str)
		if !ok4 {
			continue
		}
		val, exists := inst.Dict.GetStr(fname.V)
		if !exists {
			continue
		}
		if !first {
			sb.WriteString(", ")
		}
		first = false
		if annotate {
			sb.WriteString(fname.V)
			sb.WriteString("=")
		}
		sb.WriteString(astDumpValue(val, annotate))
	}
	sb.WriteString(")")
	return sb.String()
}

func astDumpValue(v object.Object, annotate bool) string {
	switch val := v.(type) {
	case *object.Instance:
		return astDumpNode(v, annotate)
	case *object.List:
		var parts []string
		for _, item := range val.V {
			parts = append(parts, astDumpValue(item, annotate))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *object.Str:
		return strconv.Quote(val.V)
	case *object.Int:
		return val.V.String()
	case *object.Float:
		return strconv.FormatFloat(val.V, 'g', -1, 64)
	case *object.Bool:
		if val.V {
			return "True"
		}
		return "False"
	case *object.NoneType:
		return "None"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// astUnparse returns a simplified source representation of a node.
func astUnparse(node object.Object) string {
	inst, ok := node.(*object.Instance)
	if !ok {
		return ""
	}
	switch inst.Class.Name {
	case "Module":
		if bv, ok2 := inst.Dict.GetStr("body"); ok2 {
			if bl, ok3 := bv.(*object.List); ok3 {
				var lines []string
				for _, stmt := range bl.V {
					lines = append(lines, astUnparse(stmt))
				}
				return strings.Join(lines, "\n")
			}
		}
		return ""
	case "Expr":
		if v, ok2 := inst.Dict.GetStr("value"); ok2 {
			return astUnparse(v)
		}
		return ""
	case "Assign":
		tgt := ""
		if tv, ok2 := inst.Dict.GetStr("targets"); ok2 {
			if tl, ok3 := tv.(*object.List); ok3 && len(tl.V) > 0 {
				tgt = astUnparse(tl.V[0])
			}
		}
		val := ""
		if vv, ok2 := inst.Dict.GetStr("value"); ok2 {
			val = astUnparse(vv)
		}
		return tgt + " = " + val
	case "Name":
		if iv, ok2 := inst.Dict.GetStr("id"); ok2 {
			if s, ok3 := iv.(*object.Str); ok3 {
				return s.V
			}
		}
		return "_"
	case "Constant":
		if cv, ok2 := inst.Dict.GetStr("value"); ok2 {
			switch v := cv.(type) {
			case *object.Str:
				return strconv.Quote(v.V)
			case *object.Int:
				return v.V.String()
			case *object.Float:
				return strconv.FormatFloat(v.V, 'g', -1, 64)
			case *object.Bool:
				if v.V {
					return "True"
				}
				return "False"
			case *object.NoneType:
				return "None"
			}
		}
		return ""
	default:
		return inst.Class.Name
	}
}

// ── literal_eval parser ───────────────────────────────────────────────────────

type pyLitParser struct {
	src []rune
	pos int
}

func parsePyLiteral(s string) (object.Object, error) {
	p := &pyLitParser{src: []rune(strings.TrimSpace(s))}
	v, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	p.skipWS()
	if p.pos < len(p.src) {
		return nil, fmt.Errorf("malformed value: unexpected %q", string(p.src[p.pos:]))
	}
	return v, nil
}

func (p *pyLitParser) skipWS() {
	for p.pos < len(p.src) && unicode.IsSpace(p.src[p.pos]) {
		p.pos++
	}
}

func (p *pyLitParser) peek() (rune, bool) {
	p.skipWS()
	if p.pos >= len(p.src) {
		return 0, false
	}
	return p.src[p.pos], true
}

func (p *pyLitParser) parseValue() (object.Object, error) {
	ch, ok := p.peek()
	if !ok {
		return nil, fmt.Errorf("unexpected end of input")
	}
	switch {
	case ch == '[':
		return p.parseList()
	case ch == '(':
		return p.parseTuple()
	case ch == '{':
		return p.parseDictOrSet()
	case ch == '"' || ch == '\'':
		return p.parseStr()
	case ch == '-' || ch == '+' || unicode.IsDigit(ch):
		return p.parseNumber()
	case unicode.IsLetter(ch) || ch == '_':
		return p.parseName()
	default:
		return nil, fmt.Errorf("malformed value: unexpected %q", string(ch))
	}
}

func (p *pyLitParser) parseList() (object.Object, error) {
	p.skipWS()
	p.pos++ // consume '['
	var elems []object.Object
	for {
		p.skipWS()
		if p.pos >= len(p.src) {
			return nil, fmt.Errorf("unterminated list")
		}
		if p.src[p.pos] == ']' {
			p.pos++
			break
		}
		if len(elems) > 0 {
			if p.src[p.pos] != ',' {
				return nil, fmt.Errorf("expected ',' in list")
			}
			p.pos++
			p.skipWS()
			if p.pos < len(p.src) && p.src[p.pos] == ']' {
				p.pos++
				break
			}
		}
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		elems = append(elems, v)
	}
	return &object.List{V: elems}, nil
}

func (p *pyLitParser) parseTuple() (object.Object, error) {
	p.skipWS()
	p.pos++ // consume '('
	var elems []object.Object
	for {
		p.skipWS()
		if p.pos >= len(p.src) {
			return nil, fmt.Errorf("unterminated tuple")
		}
		if p.src[p.pos] == ')' {
			p.pos++
			break
		}
		if len(elems) > 0 {
			if p.src[p.pos] != ',' {
				return nil, fmt.Errorf("expected ',' in tuple")
			}
			p.pos++
			p.skipWS()
			if p.pos < len(p.src) && p.src[p.pos] == ')' {
				p.pos++
				break
			}
		}
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		elems = append(elems, v)
	}
	return &object.Tuple{V: elems}, nil
}

func (p *pyLitParser) parseDictOrSet() (object.Object, error) {
	p.skipWS()
	p.pos++ // consume '{'
	p.skipWS()
	if p.pos < len(p.src) && p.src[p.pos] == '}' {
		p.pos++
		return object.NewDict(), nil
	}
	// Parse first value to decide dict vs set
	first, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	p.skipWS()
	if p.pos < len(p.src) && p.src[p.pos] == ':' {
		// dict
		p.pos++ // consume ':'
		firstVal, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		d := object.NewDict()
		d.Set(first, firstVal) //nolint
		for {
			p.skipWS()
			if p.pos >= len(p.src) {
				return nil, fmt.Errorf("unterminated dict")
			}
			if p.src[p.pos] == '}' {
				p.pos++
				break
			}
			if p.src[p.pos] != ',' {
				return nil, fmt.Errorf("expected ',' in dict")
			}
			p.pos++
			p.skipWS()
			if p.pos < len(p.src) && p.src[p.pos] == '}' {
				p.pos++
				break
			}
			k, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			p.skipWS()
			if p.pos >= len(p.src) || p.src[p.pos] != ':' {
				return nil, fmt.Errorf("expected ':' in dict")
			}
			p.pos++
			v, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			d.Set(k, v) //nolint
		}
		return d, nil
	}
	// set
	s := object.NewSet()
	s.Add(first) //nolint
	for {
		p.skipWS()
		if p.pos >= len(p.src) {
			return nil, fmt.Errorf("unterminated set")
		}
		if p.src[p.pos] == '}' {
			p.pos++
			break
		}
		if p.src[p.pos] != ',' {
			return nil, fmt.Errorf("expected ',' in set")
		}
		p.pos++
		p.skipWS()
		if p.pos < len(p.src) && p.src[p.pos] == '}' {
			p.pos++
			break
		}
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		s.Add(v) //nolint
	}
	return s, nil
}

func (p *pyLitParser) parseStr() (object.Object, error) {
	p.skipWS()
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("expected string")
	}
	// Check for triple-quoted strings
	quote := p.src[p.pos]
	if quote != '"' && quote != '\'' {
		return nil, fmt.Errorf("expected string")
	}
	p.pos++
	// Check for triple quote
	triple := false
	if p.pos+1 < len(p.src) && p.src[p.pos] == quote && p.src[p.pos+1] == quote {
		triple = true
		p.pos += 2
	}
	var sb strings.Builder
	for {
		if p.pos >= len(p.src) {
			return nil, fmt.Errorf("unterminated string")
		}
		ch := p.src[p.pos]
		if ch == '\\' {
			p.pos++
			if p.pos >= len(p.src) {
				return nil, fmt.Errorf("unterminated escape")
			}
			esc := p.src[p.pos]
			p.pos++
			switch esc {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case 'r':
				sb.WriteRune('\r')
			case '\\':
				sb.WriteRune('\\')
			case '\'':
				sb.WriteRune('\'')
			case '"':
				sb.WriteRune('"')
			case '0':
				sb.WriteRune(0)
			case 'a':
				sb.WriteRune('\a')
			case 'b':
				sb.WriteRune('\b')
			case 'f':
				sb.WriteRune('\f')
			case 'v':
				sb.WriteRune('\v')
			case 'u', 'U':
				// unicode escape
				size := 4
				if esc == 'U' {
					size = 8
				}
				if p.pos+size > len(p.src) {
					return nil, fmt.Errorf("invalid unicode escape")
				}
				hex := string(p.src[p.pos : p.pos+size])
				p.pos += size
				n, err := strconv.ParseInt(hex, 16, 32)
				if err != nil {
					return nil, fmt.Errorf("invalid unicode escape: %s", hex)
				}
				sb.WriteRune(rune(n))
			case 'x':
				if p.pos+2 > len(p.src) {
					return nil, fmt.Errorf("invalid hex escape")
				}
				hex := string(p.src[p.pos : p.pos+2])
				p.pos += 2
				n, err := strconv.ParseInt(hex, 16, 32)
				if err != nil {
					return nil, fmt.Errorf("invalid hex escape: %s", hex)
				}
				sb.WriteRune(rune(n))
			default:
				sb.WriteRune('\\')
				sb.WriteRune(esc)
			}
			continue
		}
		if triple {
			if ch == quote && p.pos+2 < len(p.src) && p.src[p.pos+1] == quote && p.src[p.pos+2] == quote {
				p.pos += 3
				break
			}
		} else {
			if ch == quote {
				p.pos++
				break
			}
			if ch == '\n' {
				return nil, fmt.Errorf("EOL in string")
			}
		}
		sb.WriteRune(ch)
		p.pos++
	}
	return &object.Str{V: sb.String()}, nil
}

func (p *pyLitParser) parseNumber() (object.Object, error) {
	p.skipWS()
	start := p.pos
	if p.pos < len(p.src) && (p.src[p.pos] == '-' || p.src[p.pos] == '+') {
		p.pos++
	}
	isFloat := false
	for p.pos < len(p.src) {
		ch := p.src[p.pos]
		if ch == '.' || ch == 'e' || ch == 'E' {
			isFloat = true
			p.pos++
		} else if ch == '+' || ch == '-' {
			// exponent sign
			prev := p.src[p.pos-1]
			if prev == 'e' || prev == 'E' {
				p.pos++
			} else {
				break
			}
		} else if unicode.IsDigit(ch) || (ch == 'x' || ch == 'X' || ch == 'o' || ch == 'O' || ch == 'b' || ch == 'B') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') || ch == '_' {
			p.pos++
		} else {
			break
		}
	}
	numStr := string(p.src[start:p.pos])
	numStr = strings.ReplaceAll(numStr, "_", "")
	if isFloat {
		f, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float: %s", numStr)
		}
		return &object.Float{V: f}, nil
	}
	// Try int with auto base
	base := 10
	s := numStr
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		base = 16
		s = s[2:]
	} else if strings.HasPrefix(s, "0o") || strings.HasPrefix(s, "0O") {
		base = 8
		s = s[2:]
	} else if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
		base = 2
		s = s[2:]
	}
	n, err := strconv.ParseInt(s, base, 64)
	if err != nil {
		// try float
		f, err2 := strconv.ParseFloat(numStr, 64)
		if err2 != nil {
			return nil, fmt.Errorf("invalid number: %s", numStr)
		}
		return &object.Float{V: f}, nil
	}
	if neg {
		n = -n
	}
	return object.NewInt(n), nil
}

func (p *pyLitParser) parseName() (object.Object, error) {
	p.skipWS()
	start := p.pos
	for p.pos < len(p.src) && (unicode.IsLetter(p.src[p.pos]) || unicode.IsDigit(p.src[p.pos]) || p.src[p.pos] == '_') {
		p.pos++
	}
	name := string(p.src[start:p.pos])
	switch name {
	case "True":
		return object.True, nil
	case "False":
		return object.False, nil
	case "None":
		return object.None, nil
	default:
		return nil, fmt.Errorf("malformed value: %q is not a literal", name)
	}
}

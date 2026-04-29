package vm

import "github.com/tamnd/goipy/object"

// installExceptionMethods wires PEP 678 / PEP 654 methods onto the
// BaseException and BaseExceptionGroup classes so every exception
// subclass inherits them via classLookup.
func (i *Interp) installExceptionMethods() {
	i.baseExc.Dict.SetStr("add_note", &object.BuiltinFunc{
		Name: "add_note",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "add_note() takes exactly one argument (0 given)")
			}
			e, ok := a[0].(*object.Exception)
			if !ok {
				return nil, object.Errorf(i.typeErr, "add_note() requires an exception")
			}
			s, ok := a[1].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "note must be a str, not '%s'", object.TypeName(a[1]))
			}
			if e.Dict == nil {
				e.Dict = object.NewDict()
			}
			var notes *object.List
			if cur, ok := e.Dict.GetStr("__notes__"); ok {
				if l, ok := cur.(*object.List); ok {
					notes = l
				}
			}
			if notes == nil {
				notes = &object.List{V: nil}
				e.Dict.SetStr("__notes__", notes)
			}
			notes.V = append(notes.V, s)
			return object.None, nil
		},
	})

	i.baseExc.Dict.SetStr("with_traceback", &object.BuiltinFunc{
		Name: "with_traceback",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "with_traceback() takes exactly one argument (0 given)")
			}
			e, ok := a[0].(*object.Exception)
			if !ok {
				return nil, object.Errorf(i.typeErr, "with_traceback() requires an exception")
			}
			switch tb := a[1].(type) {
			case *object.NoneType:
				e.Traceback = nil
			case *object.Traceback:
				e.Traceback = tb
			default:
				return nil, object.Errorf(i.typeErr, "__traceback__ must be a traceback or None")
			}
			return e, nil
		},
	})

	i.baseExcGroup.Dict.SetStr("derive", &object.BuiltinFunc{
		Name: "derive",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "derive() takes exactly one argument (0 given)")
			}
			e, ok := a[0].(*object.Exception)
			if !ok {
				return nil, object.Errorf(i.typeErr, "derive() requires an exception group")
			}
			items, err := iterate(i, a[1])
			if err != nil {
				return nil, err
			}
			return i.egDerive(e, items), nil
		},
	})

	i.baseExcGroup.Dict.SetStr("split", &object.BuiltinFunc{
		Name: "split",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "split() takes exactly one argument (0 given)")
			}
			e, ok := a[0].(*object.Exception)
			if !ok {
				return nil, object.Errorf(i.typeErr, "split() requires an exception group")
			}
			matched, rest := i.egSplit(e, a[1])
			return &object.Tuple{V: []object.Object{matched, rest}}, nil
		},
	})

	i.baseExcGroup.Dict.SetStr("subgroup", &object.BuiltinFunc{
		Name: "subgroup",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "subgroup() takes exactly one argument (0 given)")
			}
			e, ok := a[0].(*object.Exception)
			if !ok {
				return nil, object.Errorf(i.typeErr, "subgroup() requires an exception group")
			}
			matched, _ := i.egSplit(e, a[1])
			return matched, nil
		},
	})
}

// allExceptionSubclass reports whether every item in seq is an exception
// instance whose class is a subclass of Exception. Used by the
// BaseExceptionGroup constructor to auto-promote to ExceptionGroup.
func allExceptionSubclass(i *Interp, seq object.Object) (bool, bool) {
	items, err := iterate(i, seq)
	if err != nil {
		return false, false
	}
	for _, it := range items {
		e, ok := it.(*object.Exception)
		if !ok {
			return false, true
		}
		if !object.IsSubclass(e.Class, i.exception) {
			return false, true
		}
	}
	return true, true
}

// egInners returns the inner exception list of a group (Args[1]). May be
// nil when the group has no inners or is shaped unexpectedly.
func egInners(g *object.Exception) []object.Object {
	if g.Args == nil || len(g.Args.V) < 2 {
		return nil
	}
	switch v := g.Args.V[1].(type) {
	case *object.List:
		return v.V
	case *object.Tuple:
		return v.V
	}
	return nil
}

// egMessage returns the group's message (Args[0]) as a string.
func egMessage(g *object.Exception) string {
	if g.Args == nil || len(g.Args.V) < 1 {
		return ""
	}
	if s, ok := g.Args.V[0].(*object.Str); ok {
		return s.V
	}
	return object.Str_(g.Args.V[0])
}

// egDerive constructs a new same-class exception group containing the
// given inner exceptions. Mirrors CPython's BaseExceptionGroup.derive.
func (i *Interp) egDerive(g *object.Exception, items []object.Object) *object.Exception {
	return &object.Exception{
		Class: g.Class,
		Args: &object.Tuple{V: []object.Object{
			&object.Str{V: egMessage(g)},
			&object.List{V: items},
		}},
	}
}

// egMatches reports whether a leaf exception satisfies the split/subgroup
// condition. Condition may be a class, a tuple of classes, or a callable.
func (i *Interp) egMatches(exc object.Object, cond object.Object) (bool, error) {
	e, ok := exc.(*object.Exception)
	if !ok {
		return false, nil
	}
	switch c := cond.(type) {
	case *object.Class:
		return object.IsSubclass(e.Class, c), nil
	case *object.Tuple:
		for _, item := range c.V {
			if cls, ok := item.(*object.Class); ok && object.IsSubclass(e.Class, cls) {
				return true, nil
			}
		}
		return false, nil
	}
	// Callable predicate.
	r, err := i.callObject(cond, []object.Object{exc}, nil)
	if err != nil {
		return false, err
	}
	return object.Truthy(r), nil
}

// egSplit recursively partitions a group by `cond`. Returns
// (matched, rest); each is either a same-class group or None when empty.
// Top-level cause/context/traceback/__notes__ are propagated to both
// halves so existing handlers continue to see them.
func (i *Interp) egSplit(g *object.Exception, cond object.Object) (object.Object, object.Object) {
	inners := egInners(g)
	var matched, rest []object.Object
	for _, inner := range inners {
		ie, ok := inner.(*object.Exception)
		if !ok {
			rest = append(rest, inner)
			continue
		}
		if object.IsSubclass(ie.Class, i.baseExcGroup) {
			m, r := i.egSplit(ie, cond)
			if m != object.None {
				matched = append(matched, m)
			}
			if r != object.None {
				rest = append(rest, r)
			}
			continue
		}
		ok2, err := i.egMatches(ie, cond)
		if err != nil {
			// Predicate failure falls through to "no match"; the user can
			// still see the original group via the rest half.
			rest = append(rest, inner)
			continue
		}
		if ok2 {
			matched = append(matched, inner)
		} else {
			rest = append(rest, inner)
		}
	}
	mkSide := func(items []object.Object) object.Object {
		if len(items) == 0 {
			return object.None
		}
		out := i.egDerive(g, items)
		out.Cause = g.Cause
		out.Ctx = g.Ctx
		out.Traceback = g.Traceback
		if g.Dict != nil {
			if notes, ok := g.Dict.GetStr("__notes__"); ok {
				if out.Dict == nil {
					out.Dict = object.NewDict()
				}
				out.Dict.SetStr("__notes__", notes)
			}
		}
		return out
	}
	return mkSide(matched), mkSide(rest)
}

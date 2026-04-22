package vm

import (
	"strings"

	"github.com/tamnd/goipy/object"
)

// Python keywords (kwlist + softkwlist for Python 3.14)
var rlKeywords = []string{
	"False", "None", "True", "and", "as", "assert", "async", "await",
	"break", "class", "continue", "def", "del", "elif", "else", "except",
	"finally", "for", "from", "global", "if", "import", "in", "is",
	"lambda", "nonlocal", "not", "or", "pass", "raise", "return",
	"try", "while", "with", "yield",
	// soft keywords
	"match", "case", "type",
}

func (i *Interp) buildRlcompleter() *object.Module {
	m := &object.Module{Name: "rlcompleter", Dict: object.NewDict()}

	isCallable := func(o object.Object) bool {
		switch o.(type) {
		case *object.BuiltinFunc, *object.Function, *object.BoundMethod, *object.Class:
			return true
		}
		return false
	}

	makeCompleteFn := func(ns *object.Dict) *object.BuiltinFunc {
		// Per-instance match cache: rebuilt on state==0.
		var cachedMatches []string

		buildMatches := func(text string) []string {
			var matches []string

			if strings.Contains(text, ".") {
				dotIdx := strings.LastIndex(text, ".")
				expr := text[:dotIdx]
				attr := text[dotIdx+1:]

				var obj object.Object
				if ns != nil {
					if v, ok := ns.GetStr(expr); ok {
						obj = v
					}
				}
				if obj == nil {
					if v, ok := i.Builtins.GetStr(expr); ok {
						obj = v
					}
				}

				if obj != nil {
					for _, name := range dirOf([]object.Object{obj}) {
						if s, ok := name.(*object.Str); ok && strings.HasPrefix(s.V, attr) {
							matches = append(matches, expr+"."+s.V)
						}
					}
				}
				return matches
			}

			// global: keywords + namespace + builtins
			seen := map[string]bool{"__builtins__": true}

			for _, kw := range rlKeywords {
				if strings.HasPrefix(kw, text) && !seen[kw] {
					seen[kw] = true
					matches = append(matches, kw+" ")
				}
			}

			for _, nspace := range []*object.Dict{ns, i.Builtins} {
				if nspace == nil {
					continue
				}
				keys, vals := nspace.Items()
				for idx, k := range keys {
					s, ok := k.(*object.Str)
					if !ok || !strings.HasPrefix(s.V, text) || seen[s.V] {
						continue
					}
					seen[s.V] = true
					word := s.V
					if isCallable(vals[idx]) {
						word += "("
					}
					matches = append(matches, word)
				}
			}
			return matches
		}

		// complete(text, state) — stored in instance dict so it's called
		// WITHOUT self prepended (instance dict lookup returns raw BuiltinFunc).
		return &object.BuiltinFunc{Name: "complete", Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 2 {
				return nil, object.Errorf(i.typeErr, "complete() requires text and state arguments")
			}
			text, ok := a[0].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "complete() text must be str")
			}
			stateObj, ok2 := a[1].(*object.Int)
			if !ok2 {
				return nil, object.Errorf(i.typeErr, "complete() state must be int")
			}
			state := int(stateObj.V.Int64())

			if state == 0 {
				cachedMatches = buildMatches(text.V)
			}
			if state < 0 || state >= len(cachedMatches) {
				return object.None, nil
			}
			return &object.Str{V: cachedMatches[state]}, nil
		}}
	}

	completerClass := &object.Class{Name: "Completer", Dict: object.NewDict()}
	completerClass.Dict.SetStr("__init__", &object.BuiltinFunc{Name: "__init__", Call: func(_ any, a []object.Object, kw *object.Dict) (object.Object, error) {
		if len(a) < 1 {
			return object.None, nil
		}
		self, ok := a[0].(*object.Instance)
		if !ok {
			return object.None, nil
		}
		var ns *object.Dict
		var nsArg object.Object
		if len(a) > 1 {
			nsArg = a[1]
		} else if kw != nil {
			if v, ok2 := kw.GetStr("namespace"); ok2 {
				nsArg = v
			}
		}
		if nsArg != nil && nsArg != object.None {
			d, ok2 := nsArg.(*object.Dict)
			if !ok2 {
				return nil, object.Errorf(i.typeErr, "namespace must be a dictionary")
			}
			ns = d
		}

		// Store complete in the instance dict as a closure over ns.
		// Instance dict lookup bypasses the class method cache, ensuring
		// self is never incorrectly injected.
		self.Dict.SetStr("complete", makeCompleteFn(ns))
		return object.None, nil
	}})

	m.Dict.SetStr("Completer", completerClass)
	return m
}

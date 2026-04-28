package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildKeyword constructs the keyword module with the CPython 3.14 kwlist,
// softkwlist, iskeyword(), and issoftkeyword() functions.
func (i *Interp) buildKeyword() *object.Module {
	m := &object.Module{Name: "keyword", Dict: object.NewDict()}

	// ── kwlist ────────────────────────────────────────────────────────────

	hardKws := []string{
		"False", "None", "True", "and", "as", "assert", "async", "await",
		"break", "class", "continue", "def", "del", "elif", "else", "except",
		"finally", "for", "from", "global", "if", "import", "in", "is",
		"lambda", "nonlocal", "not", "or", "pass", "raise", "return",
		"try", "while", "with", "yield",
	}

	kwObjs := make([]object.Object, len(hardKws))
	hardSet := make(map[string]bool, len(hardKws))
	for j, kw := range hardKws {
		kwObjs[j] = &object.Str{V: kw}
		hardSet[kw] = true
	}
	m.Dict.SetStr("kwlist", &object.List{V: kwObjs})

	// ── softkwlist ────────────────────────────────────────────────────────

	softKws := []string{"_", "case", "match", "type"}

	softObjs := make([]object.Object, len(softKws))
	softSet := make(map[string]bool, len(softKws))
	for j, kw := range softKws {
		softObjs[j] = &object.Str{V: kw}
		softSet[kw] = true
	}
	m.Dict.SetStr("softkwlist", &object.List{V: softObjs})

	// ── iskeyword ─────────────────────────────────────────────────────────

	m.Dict.SetStr("iskeyword", &object.BuiltinFunc{
		Name: "iskeyword",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "iskeyword() missing argument")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return object.False, nil
			}
			if hardSet[s.V] {
				return object.True, nil
			}
			return object.False, nil
		},
	})

	// ── issoftkeyword ─────────────────────────────────────────────────────

	m.Dict.SetStr("issoftkeyword", &object.BuiltinFunc{
		Name: "issoftkeyword",
		Call: func(_ any, a []object.Object, _ *object.Dict) (object.Object, error) {
			if len(a) < 1 {
				return nil, object.Errorf(i.typeErr, "issoftkeyword() missing argument")
			}
			s, ok := a[0].(*object.Str)
			if !ok {
				return object.False, nil
			}
			if softSet[s.V] {
				return object.True, nil
			}
			return object.False, nil
		},
	})

	return m
}

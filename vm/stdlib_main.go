package vm

import (
	"github.com/tamnd/goipy/object"
)

// buildMain returns a fallback __main__ module for contexts where Run() has
// not yet registered the live namespace (e.g. subinterpreters, bare imports).
// When Run() is called it registers the real execution namespace in i.modules
// so that subsequent `import __main__` calls return the live dict instead.
func (i *Interp) buildMain() *object.Module {
	m := &object.Module{Name: "__main__", Dict: object.NewDict()}
	m.Dict.SetStr("__name__", &object.Str{V: "__main__"})
	m.Dict.SetStr("__doc__", object.None)
	m.Dict.SetStr("__annotations__", object.NewDict())
	m.Dict.SetStr("__builtins__", i.Builtins)
	m.Dict.SetStr("__spec__", object.None)
	m.Dict.SetStr("__loader__", object.None)
	m.Dict.SetStr("__package__", object.None)
	return m
}

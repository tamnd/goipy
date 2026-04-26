package vm

import "github.com/tamnd/goipy/object"

func (i *Interp) buildXmlrpc() *object.Module {
	m := &object.Module{Name: "xmlrpc", Dict: object.NewDict()}
	m.Dict.SetStr("__path__", &object.List{V: []object.Object{&object.Str{V: ""}}})
	m.Dict.SetStr("__package__", &object.Str{V: "xmlrpc"})
	return m
}

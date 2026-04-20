package vm

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/tamnd/goipy/object"
)

func (i *Interp) setitem(container, key, val object.Object) error {
	switch c := container.(type) {
	case *object.List:
		if sl, ok := key.(*object.Slice); ok {
			return i.listSetSlice(c, sl, val)
		}
		n, ok := toInt64(key)
		if !ok {
			return object.Errorf(i.typeErr, "list indices must be integers")
		}
		L := int64(len(c.V))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return object.Errorf(i.indexErr, "list index out of range")
		}
		c.V[n] = val
		return nil
	case *object.Dict:
		return c.Set(key, val)
	}
	return object.Errorf(i.typeErr, "'%s' does not support item assignment", object.TypeName(container))
}

func (i *Interp) listSetSlice(l *object.List, sl *object.Slice, val object.Object) error {
	start, stop, step, err := i.resolveSlice(sl, len(l.V))
	if err != nil {
		return err
	}
	if step != 1 {
		return object.Errorf(i.valueErr, "extended slice assignment not supported")
	}
	items, err := iterate(i, val)
	if err != nil {
		return err
	}
	newV := make([]object.Object, 0, len(l.V)-(stop-start)+len(items))
	newV = append(newV, l.V[:start]...)
	newV = append(newV, items...)
	newV = append(newV, l.V[stop:]...)
	l.V = newV
	return nil
}

func (i *Interp) delitem(container, key object.Object) error {
	switch c := container.(type) {
	case *object.List:
		n, ok := toInt64(key)
		if !ok {
			return object.Errorf(i.typeErr, "list indices must be integers")
		}
		L := int64(len(c.V))
		if n < 0 {
			n += L
		}
		if n < 0 || n >= L {
			return object.Errorf(i.indexErr, "list index out of range")
		}
		c.V = append(c.V[:n], c.V[n+1:]...)
		return nil
	case *object.Dict:
		ok, err := c.Delete(key)
		if err != nil {
			return err
		}
		if !ok {
			return object.Errorf(i.keyErr, "%s", object.Repr(key))
		}
		return nil
	}
	return object.Errorf(i.typeErr, "'%s' does not support item deletion", object.TypeName(container))
}

func (i *Interp) unaryNeg(v object.Object) (object.Object, error) {
	switch x := v.(type) {
	case *object.Bool:
		r := int64(0)
		if x.V {
			r = -1
		}
		return object.NewInt(r), nil
	case *object.Int:
		return &object.Int{V: new(big.Int).Neg(x.V)}, nil
	case *object.Float:
		return &object.Float{V: -x.V}, nil
	}
	return nil, object.Errorf(i.typeErr, "bad operand for unary -: '%s'", object.TypeName(v))
}

func (i *Interp) length(v object.Object) (int64, error) {
	switch x := v.(type) {
	case *object.Str:
		return int64(len(x.Runes())), nil
	case *object.Bytes:
		return int64(len(x.V)), nil
	case *object.List:
		return int64(len(x.V)), nil
	case *object.Tuple:
		return int64(len(x.V)), nil
	case *object.Dict:
		return int64(x.Len()), nil
	case *object.Set:
		return int64(x.Len()), nil
	case *object.Range:
		return rangeLen(x), nil
	}
	return 0, object.Errorf(i.typeErr, "object of type '%s' has no len()", object.TypeName(v))
}

// --- attribute access ---

func (i *Interp) getAttr(o object.Object, name string) (object.Object, error) {
	// Method lookup for str.
	if s, ok := o.(*object.Str); ok {
		if m, ok := strMethod(s, name); ok {
			return m, nil
		}
	}
	if l, ok := o.(*object.List); ok {
		if m, ok := listMethod(l, name); ok {
			return m, nil
		}
	}
	if d, ok := o.(*object.Dict); ok {
		if m, ok := dictMethod(d, name); ok {
			return m, nil
		}
	}
	if s, ok := o.(*object.Set); ok {
		if m, ok := setMethod(s, name); ok {
			return m, nil
		}
	}
	if t, ok := o.(*object.Tuple); ok {
		if m, ok := tupleMethod(t, name); ok {
			return m, nil
		}
	}
	// Class attr lookup on instance
	if inst, ok := o.(*object.Instance); ok {
		if v, ok := inst.Dict.GetStr(name); ok {
			return v, nil
		}
		if v, ok := classLookup(inst.Class, name); ok {
			if fn, ok := v.(*object.Function); ok {
				return &object.BoundMethod{Self: inst, Fn: fn}, nil
			}
			return v, nil
		}
		return nil, object.Errorf(i.attrErr, "'%s' object has no attribute '%s'", inst.Class.Name, name)
	}
	if cls, ok := o.(*object.Class); ok {
		if v, ok := classLookup(cls, name); ok {
			return v, nil
		}
		return nil, object.Errorf(i.attrErr, "type object '%s' has no attribute '%s'", cls.Name, name)
	}
	if m, ok := o.(*object.Module); ok {
		if v, ok := m.Dict.GetStr(name); ok {
			return v, nil
		}
	}
	if e, ok := o.(*object.Exception); ok {
		switch name {
		case "args":
			return e.Args, nil
		}
	}
	return nil, object.Errorf(i.attrErr, "'%s' object has no attribute '%s'", object.TypeName(o), name)
}

func (i *Interp) setAttr(o object.Object, name string, val object.Object) error {
	if inst, ok := o.(*object.Instance); ok {
		inst.Dict.SetStr(name, val)
		return nil
	}
	if cls, ok := o.(*object.Class); ok {
		cls.Dict.SetStr(name, val)
		return nil
	}
	if m, ok := o.(*object.Module); ok {
		m.Dict.SetStr(name, val)
		return nil
	}
	return object.Errorf(i.attrErr, "'%s' object has no attribute '%s'", object.TypeName(o), name)
}

func (i *Interp) delAttr(o object.Object, name string) error {
	if inst, ok := o.(*object.Instance); ok {
		ok, _ := inst.Dict.Delete(&object.Str{V: name})
		if !ok {
			return object.Errorf(i.attrErr, "no attribute '%s'", name)
		}
		return nil
	}
	return object.Errorf(i.attrErr, "can't delete attribute")
}

func classLookup(c *object.Class, name string) (object.Object, bool) {
	if v, ok := c.Dict.GetStr(name); ok {
		return v, true
	}
	for _, b := range c.Bases {
		if v, ok := classLookup(b, name); ok {
			return v, true
		}
	}
	return nil, false
}

// --- iteration ---

func (i *Interp) getIter(v object.Object) (*object.Iter, error) {
	switch x := v.(type) {
	case *object.Iter:
		return x, nil
	case *object.List:
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(x.V) {
				return nil, false, nil
			}
			r := x.V[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.Tuple:
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(x.V) {
				return nil, false, nil
			}
			r := x.V[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.Str:
		rs := x.Runes()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(rs) {
				return nil, false, nil
			}
			r := &object.Str{V: string(rs[idx])}
			idx++
			return r, true, nil
		}}, nil
	case *object.Bytes:
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(x.V) {
				return nil, false, nil
			}
			r := object.NewInt(int64(x.V[idx]))
			idx++
			return r, true, nil
		}}, nil
	case *object.Dict:
		keys, _ := x.Items()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(keys) {
				return nil, false, nil
			}
			r := keys[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.Set:
		items := x.Items()
		idx := 0
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if idx >= len(items) {
				return nil, false, nil
			}
			r := items[idx]
			idx++
			return r, true, nil
		}}, nil
	case *object.Range:
		cur := x.Start
		return &object.Iter{Next: func() (object.Object, bool, error) {
			if x.Step > 0 && cur >= x.Stop {
				return nil, false, nil
			}
			if x.Step < 0 && cur <= x.Stop {
				return nil, false, nil
			}
			r := object.NewInt(cur)
			cur += x.Step
			return r, true, nil
		}}, nil
	}
	return nil, object.Errorf(i.typeErr, "'%s' object is not iterable", object.TypeName(v))
}

// iterate exhausts an iterable into a slice.
func iterate(i *Interp, v object.Object) ([]object.Object, error) {
	it, err := i.getIter(v)
	if err != nil {
		return nil, err
	}
	var out []object.Object
	for {
		x, ok, err := it.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			return out, nil
		}
		out = append(out, x)
	}
}

// --- format ---

func formatValue(v object.Object, spec string) (string, error) {
	if spec == "" {
		return object.Str_(v), nil
	}
	// Parse a minimal subset: [fill][align][sign][#][0][width][,][.precision][type]
	s := spec
	fill := byte(' ')
	align := byte(0)
	sign := byte(0)
	width := 0
	precision := -1
	typ := byte(0)

	if len(s) >= 2 && (s[1] == '<' || s[1] == '>' || s[1] == '^' || s[1] == '=') {
		fill = s[0]
		align = s[1]
		s = s[2:]
	} else if len(s) >= 1 && (s[0] == '<' || s[0] == '>' || s[0] == '^' || s[0] == '=') {
		align = s[0]
		s = s[1:]
	}
	if len(s) > 0 && (s[0] == '+' || s[0] == '-' || s[0] == ' ') {
		sign = s[0]
		s = s[1:]
	}
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	if len(s) > 0 && s[0] == '0' {
		if align == 0 {
			align = '='
			fill = '0'
		}
		s = s[1:]
	}
	// width
	for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		width = width*10 + int(s[0]-'0')
		s = s[1:]
	}
	if len(s) > 0 && s[0] == ',' {
		s = s[1:]
	}
	if len(s) > 0 && s[0] == '.' {
		s = s[1:]
		precision = 0
		for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
			precision = precision*10 + int(s[0]-'0')
			s = s[1:]
		}
	}
	if len(s) > 0 {
		typ = s[0]
	}

	var body string
	switch x := v.(type) {
	case *object.Int:
		switch typ {
		case 'b':
			body = x.V.Text(2)
		case 'o':
			body = x.V.Text(8)
		case 'x':
			body = x.V.Text(16)
		case 'X':
			body = strings.ToUpper(x.V.Text(16))
		case 'd', 0:
			body = x.V.String()
		case 'f', 'F':
			fv, _ := new(big.Float).SetInt(x.V).Float64()
			p := precision
			if p < 0 {
				p = 6
			}
			body = strconv.FormatFloat(fv, 'f', p, 64)
		case 'e', 'E':
			fv, _ := new(big.Float).SetInt(x.V).Float64()
			p := precision
			if p < 0 {
				p = 6
			}
			body = strconv.FormatFloat(fv, byte(typ), p, 64)
		default:
			body = x.V.String()
		}
		body = applySign(body, sign, x.V.Sign())
	case *object.Float:
		p := precision
		ft := typ
		if ft == 0 {
			ft = 'g'
		}
		if p < 0 {
			p = 6
		}
		if ft == 'g' || ft == 'G' {
			if precision < 0 {
				p = 6
			}
		}
		body = strconv.FormatFloat(x.V, ft, p, 64)
		signV := 0
		if x.V > 0 {
			signV = 1
		} else if x.V < 0 {
			signV = -1
		}
		body = applySign(body, sign, signV)
	case *object.Str:
		body = x.V
		if precision >= 0 && precision < len(body) {
			body = body[:precision]
		}
	default:
		body = object.Str_(v)
	}

	if width > len(body) {
		pad := width - len(body)
		padStr := strings.Repeat(string(fill), pad)
		switch align {
		case '<':
			body = body + padStr
		case '>', 0:
			body = padStr + body
		case '^':
			l := pad / 2
			r := pad - l
			body = strings.Repeat(string(fill), l) + body + strings.Repeat(string(fill), r)
		case '=':
			if len(body) > 0 && (body[0] == '+' || body[0] == '-') {
				body = string(body[0]) + padStr + body[1:]
			} else {
				body = padStr + body
			}
		}
	}
	return body, nil
}

func applySign(body string, sign byte, signV int) string {
	if signV >= 0 {
		switch sign {
		case '+':
			return "+" + body
		case ' ':
			return " " + body
		}
	}
	return body
}

// --- helpers for toException ---

func (i *Interp) toException(v object.Object) error {
	if e, ok := v.(*object.Exception); ok {
		return e
	}
	if cls, ok := v.(*object.Class); ok {
		return object.NewException(cls, "")
	}
	return object.Errorf(i.typeErr, "exceptions must derive from BaseException (got %s)", object.TypeName(v))
}

// Debug (unused).
var _ = fmt.Sprintf

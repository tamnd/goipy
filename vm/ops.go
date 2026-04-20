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
		if sl, ok := key.(*object.Slice); ok {
			start, stop, step, err := i.resolveSlice(sl, len(c.V))
			if err != nil {
				return err
			}
			if step != 1 {
				return object.Errorf(i.valueErr, "extended slice deletion not supported")
			}
			c.V = append(c.V[:start], c.V[stop:]...)
			return nil
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
			return i.bindDescriptor(v, inst, inst.Class)
		}
		return nil, object.Errorf(i.attrErr, "'%s' object has no attribute '%s'", inst.Class.Name, name)
	}
	if cls, ok := o.(*object.Class); ok {
		switch name {
		case "__name__":
			return &object.Str{V: cls.Name}, nil
		case "__bases__":
			bs := make([]object.Object, len(cls.Bases))
			for k, b := range cls.Bases {
				bs[k] = b
			}
			return &object.Tuple{V: bs}, nil
		}
		if v, ok := classLookup(cls, name); ok {
			return i.bindDescriptor(v, nil, cls)
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

// matchClass implements PEP 634 class-pattern extraction. Returns a tuple of
// attributes on match or None on miss. count is positional sub-pattern count;
// kwnamesObj is a tuple of keyword attribute names that follow them.
func (i *Interp) matchClass(subject, cls, kwnamesObj object.Object, count int) (object.Object, error) {
	kwnames, _ := kwnamesObj.(*object.Tuple)
	nkw := 0
	if kwnames != nil {
		nkw = len(kwnames.V)
	}
	if !matchTypeCheck(subject, cls) {
		return object.None, nil
	}
	if bf, ok := cls.(*object.BuiltinFunc); ok && isSpecialMatchClass(bf.Name) {
		if count == 0 && nkw == 0 {
			return &object.Tuple{V: nil}, nil
		}
		if count == 1 && nkw == 0 {
			return &object.Tuple{V: []object.Object{subject}}, nil
		}
		if count > 1 {
			return nil, object.Errorf(i.typeErr, "match() accepts at most 1 positional sub-pattern for builtin class %s", bf.Name)
		}
	}
	var matchArgs []string
	if count > 0 {
		uc, ok := cls.(*object.Class)
		if !ok {
			return nil, object.Errorf(i.typeErr, "class pattern requires __match_args__")
		}
		v, ok := classLookup(uc, "__match_args__")
		if !ok {
			return nil, object.Errorf(i.typeErr, "%s has no __match_args__", uc.Name)
		}
		t, ok := v.(*object.Tuple)
		if !ok {
			return nil, object.Errorf(i.typeErr, "__match_args__ must be a tuple")
		}
		if len(t.V) < count {
			return nil, object.Errorf(i.typeErr, "%s has %d positional patterns but __match_args__ has %d", uc.Name, count, len(t.V))
		}
		for k := 0; k < count; k++ {
			s, ok := t.V[k].(*object.Str)
			if !ok {
				return nil, object.Errorf(i.typeErr, "__match_args__ elements must be strings")
			}
			matchArgs = append(matchArgs, s.V)
		}
	}
	attrs := make([]object.Object, 0, count+nkw)
	for _, name := range matchArgs {
		v, gerr := i.getAttr(subject, name)
		if gerr != nil {
			return object.None, nil
		}
		attrs = append(attrs, v)
	}
	if kwnames != nil {
		for _, n := range kwnames.V {
			s, ok := n.(*object.Str)
			if !ok {
				continue
			}
			v, gerr := i.getAttr(subject, s.V)
			if gerr != nil {
				return object.None, nil
			}
			attrs = append(attrs, v)
		}
	}
	return &object.Tuple{V: attrs}, nil
}

func matchTypeCheck(o, t object.Object) bool {
	if cls, ok := t.(*object.Class); ok {
		if inst, ok := o.(*object.Instance); ok {
			return object.IsSubclass(inst.Class, cls)
		}
		if e, ok := o.(*object.Exception); ok {
			return object.IsSubclass(e.Class, cls)
		}
		return false
	}
	if bf, ok := t.(*object.BuiltinFunc); ok {
		return matchBuiltinType(o, bf.Name)
	}
	return false
}

func isSpecialMatchClass(name string) bool {
	switch name {
	case "bool", "bytearray", "bytes", "dict", "float", "frozenset",
		"int", "list", "set", "str", "tuple":
		return true
	}
	return false
}

func matchBuiltinType(o object.Object, name string) bool {
	switch name {
	case "int":
		if _, ok := o.(*object.Int); ok {
			return true
		}
		_, ok := o.(*object.Bool)
		return ok
	case "bool":
		_, ok := o.(*object.Bool)
		return ok
	case "float":
		_, ok := o.(*object.Float)
		return ok
	case "str":
		_, ok := o.(*object.Str)
		return ok
	case "list":
		_, ok := o.(*object.List)
		return ok
	case "tuple":
		_, ok := o.(*object.Tuple)
		return ok
	case "dict":
		_, ok := o.(*object.Dict)
		return ok
	case "set", "frozenset":
		_, ok := o.(*object.Set)
		return ok
	case "bytes", "bytearray":
		_, ok := o.(*object.Bytes)
		return ok
	}
	return false
}

// bindDescriptor applies the descriptor protocol to v found in a class MRO.
// inst is nil when the lookup came from a class rather than an instance.
func (i *Interp) bindDescriptor(v object.Object, inst *object.Instance, cls *object.Class) (object.Object, error) {
	switch d := v.(type) {
	case *object.Property:
		if inst == nil {
			return d, nil // accessed on the class itself
		}
		return i.callObject(d.Fget, []object.Object{inst}, nil)
	case *object.ClassMethod:
		return &object.BoundMethod{Self: cls, Fn: d.Fn}, nil
	case *object.StaticMethod:
		return d.Fn, nil
	case *object.Function:
		if inst != nil {
			return &object.BoundMethod{Self: inst, Fn: d}, nil
		}
		return d, nil
	}
	return v, nil
}

// lookupAfter walks MRO(instCls) and returns the first attribute named `name`
// found in a class that appears strictly after `startCls` in the order.
func lookupAfter(instCls, startCls *object.Class, name string) (object.Object, bool) {
	mro := computeMRO(instCls)
	past := false
	for _, c := range mro {
		if past {
			if v, ok := c.Dict.GetStr(name); ok {
				return v, true
			}
		}
		if c == startCls {
			past = true
		}
	}
	return nil, false
}

// computeMRO returns a simple depth-first linearization of the class
// hierarchy. Not full C3, but correct for single inheritance and good
// enough for straightforward diamonds.
func computeMRO(c *object.Class) []*object.Class {
	var out []*object.Class
	seen := map[*object.Class]bool{}
	var walk func(*object.Class)
	walk = func(x *object.Class) {
		if seen[x] {
			return
		}
		seen[x] = true
		out = append(out, x)
		for _, b := range x.Bases {
			walk(b)
		}
	}
	walk(c)
	return out
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
	altForm := false
	if len(s) > 0 && s[0] == '#' {
		altForm = true
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
	group := byte(0)
	if len(s) > 0 && (s[0] == ',' || s[0] == '_') {
		group = s[0]
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
	zeroPadHandled := false
	switch x := v.(type) {
	case *object.Int:
		it := typ
		if it == 'n' {
			it = 'd'
		}
		absV := new(big.Int).Abs(x.V)
		var prefix, digits string
		switch it {
		case 'b':
			digits = absV.Text(2)
			if altForm {
				prefix = "0b"
			}
		case 'o':
			digits = absV.Text(8)
			if altForm {
				prefix = "0o"
			}
		case 'x':
			digits = absV.Text(16)
			if altForm {
				prefix = "0x"
			}
		case 'X':
			digits = strings.ToUpper(absV.Text(16))
			if altForm {
				prefix = "0X"
			}
		case 'c':
			cp := absV.Int64()
			body = string(rune(cp))
		case 'd', 0:
			digits = absV.String()
		case 'f', 'F':
			fv, _ := new(big.Float).SetInt(x.V).Float64()
			p := precision
			if p < 0 {
				p = 6
			}
			digits = strconv.FormatFloat(fv, 'f', p, 64)
		case 'e', 'E':
			fv, _ := new(big.Float).SetInt(x.V).Float64()
			p := precision
			if p < 0 {
				p = 6
			}
			digits = strconv.FormatFloat(fv, byte(it), p, 64)
		default:
			digits = absV.String()
		}
		if it == 'c' {
			// no sign/prefix/group
		} else {
			var signHead string
			switch {
			case x.V.Sign() < 0:
				signHead = "-"
			case sign == '+':
				signHead = "+"
			case sign == ' ':
				signHead = " "
			}
			head := signHead + prefix
			grouped := group != 0 && (it == 0 || it == 'd' || it == 'b' || it == 'o' || it == 'x' || it == 'X')
			stride := 3
			if it == 'b' || it == 'o' || it == 'x' || it == 'X' {
				stride = 4
			}
			if align == '=' && fill == '0' && width > len(head)+len(digits) {
				target := width - len(head)
				if grouped {
					d := len(digits)
					for d+(d-1)/stride < target {
						d++
					}
					if d > len(digits) {
						digits = strings.Repeat("0", d-len(digits)) + digits
					}
					digits = addGroups(digits, group, stride)
					if len(digits) < target {
						digits = strings.Repeat("0", target-len(digits)) + digits
					}
				} else {
					digits = strings.Repeat("0", target-len(digits)) + digits
				}
				body = head + digits
				zeroPadHandled = true
			} else {
				if grouped {
					digits = addGroups(digits, group, stride)
				}
				body = head + digits
			}
		}
	case *object.Float:
		p := precision
		ft := typ
		if ft == 0 {
			ft = 'g'
		}
		if ft == 'n' {
			ft = 'g'
		}
		pct := false
		if ft == '%' {
			ft = 'f'
			pct = true
		}
		if p < 0 {
			p = 6
		}
		fv := x.V
		if pct {
			fv *= 100
		}
		body = strconv.FormatFloat(fv, ft, p, 64)
		if altForm && (ft == 'g' || ft == 'G') {
			body = padGTrailingZeros(body, p)
		}
		if pct {
			body += "%"
		}
		signV := 0
		if x.V > 0 {
			signV = 1
		} else if x.V < 0 {
			signV = -1
		}
		if group != 0 {
			body = applyFloatGroup(body, group)
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

	if zeroPadHandled {
		return body, nil
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

// addGroups inserts sep every `stride` digits from the right of the digit
// portion of s. s may have an optional leading '-' sign; any non-digit prefix
// (e.g. 0x) should not be passed in.
func addGroups(s string, sep byte, stride int) string {
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	n := len(s)
	if n <= stride {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	first := n % stride
	if first == 0 {
		first = stride
	}
	b.WriteString(s[:first])
	for j := first; j < n; j += stride {
		b.WriteByte(sep)
		b.WriteString(s[j : j+stride])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// padGTrailingZeros expands a 'g'/'G'-formatted float to carry its full
// significant-digit precision, matching CPython's alt-form `#g` behavior.
// The mantissa is padded with trailing zeros so it has exactly `prec`
// significant digits, and a trailing "." is added if no decimal exists.
func padGTrailingZeros(s string, prec int) string {
	if prec <= 0 {
		prec = 1
	}
	sign := ""
	if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
		sign = string(s[0])
		s = s[1:]
	}
	mantissa, exp := s, ""
	if k := strings.IndexAny(s, "eE"); k >= 0 {
		mantissa, exp = s[:k], s[k:]
	}
	sig := 0
	seenNonZero := false
	for k := 0; k < len(mantissa); k++ {
		c := mantissa[k]
		if c == '.' {
			continue
		}
		if c >= '1' && c <= '9' {
			seenNonZero = true
			sig++
		} else if c == '0' && seenNonZero {
			sig++
		}
	}
	need := prec - sig
	if need < 0 {
		need = 0
	}
	if !strings.Contains(mantissa, ".") && need > 0 {
		mantissa += "."
	} else if !strings.Contains(mantissa, ".") && exp == "" {
		// alt form requires a trailing decimal point even if full precision.
		mantissa += "."
	}
	if need > 0 {
		mantissa += strings.Repeat("0", need)
	}
	return sign + mantissa + exp
}

// applyFloatGroup inserts group separators into the integer portion of a
// formatted float body, preserving any fractional/exponent tail and the
// optional leading sign.
func applyFloatGroup(body string, sep byte) string {
	neg := false
	s := body
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	end := len(s)
	for k := 0; k < len(s); k++ {
		if s[k] == '.' || s[k] == 'e' || s[k] == 'E' {
			end = k
			break
		}
	}
	intPart := s[:end]
	tail := s[end:]
	out := addGroups(intPart, sep, 3) + tail
	if neg {
		return "-" + out
	}
	return out
}

func applySign(body string, sign byte, signV int) string {
	if signV < 0 {
		if len(body) > 0 && body[0] == '-' {
			return body
		}
		return "-" + body
	}
	switch sign {
	case '+':
		return "+" + body
	case ' ':
		return " " + body
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
